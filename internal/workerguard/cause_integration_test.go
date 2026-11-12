//go:build integration

// The causation resolver and the record_changed wait tracker (P3-42, D-3-29)
// against a real PostgreSQL, as the real nava_worker role under RLS. The
// Security condition on TICK-88: an interpreter write takes its causation
// only from the workflow_cause row of the delivery that resumed the run, and
// only if that row is the run's own.

package workerguard

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.temporal.io/sdk/temporal"

	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
)

// seedCause writes a workflow_cause row as the enqueuing transaction does.
func seedCause(t *testing.T, conn *pgx.Conn, tenant, ref uuid.UUID, origin uuid.UUID, depth int16) uuid.UUID {
	t.Helper()
	id := uuid.New()
	mustExec(t, conn, `INSERT INTO workflow_cause (tenant_id, outbox_id, ref_outbox_id, causation_origin_id, causation_depth)
		VALUES ($1, $2, $3, $4, $5)`, tenant, id, ref, origin, depth)
	return id
}

// resolveAs runs resolveCause for workflowID in a tenant transaction of the
// context's verified tenant, as nava_worker.
func resolveAs(ctx context.Context, pool *pgxpool.Pool, workflowID string, cause uuid.UUID) (outbox.Causation, error) {
	var c outbox.Causation
	err := InTenantTx(ctx, pool, func(tx pgx.Tx) error {
		var err error
		c, err = resolveCause(ctx, tx, workflowID, cause)
		return err
	})
	return c, err
}

// ResolveCause takes the context, the transaction and the cause id, and
// nothing else: no RunInput, Delivery or StepRef value can reach it.
func TestResolveCauseTakesNoWorkflowInput(t *testing.T) {
	ft := reflect.TypeOf(ResolveCause)
	want := []reflect.Type{reflect.TypeFor[context.Context](), reflect.TypeFor[pgx.Tx](), reflect.TypeFor[uuid.UUID]()}
	if ft.NumIn() != len(want) {
		t.Fatalf("ResolveCause takes %d arguments, want %d", ft.NumIn(), len(want))
	}
	for i, w := range want {
		if ft.In(i) != w {
			t.Errorf("argument %d is %s, want %s", i, ft.In(i), w)
		}
	}
}

// The resolver returns the origin and depth + 1 of the run's own cause row,
// whatever causation the run's input or its deliveries carry; it refuses a
// row of another run, of another tenant, an unknown id, and any workflow
// that is not this tenant's interpreter run.
func TestResolveCauseOnlyFromTheRunsOwnRow(t *testing.T) {
	conn, pool := migrated(t)
	xID, x := seedRef(t, conn, tenantA, outbox.InterpreterKind)
	_, y := seedRef(t, conn, tenantA, outbox.InterpreterKind)
	_, bRef := seedRef(t, conn, tenantB, outbox.InterpreterKind)
	origin := uuid.New()
	own := seedCause(t, conn, tenantA, x, origin, 3)
	foreign := seedCause(t, conn, tenantA, y, uuid.New(), 0)
	otherTenant := seedCause(t, conn, tenantB, bRef, uuid.New(), 0)
	ctx := ContextWithTenant(context.Background(), tenantA)

	// The run's input and its deliveries may carry any causation (a forged
	// RunInput, Delivery or StepRef): none of it is an argument
	// (TestResolveCauseTakesNoWorkflowInput), and the result is the row's.
	got, err := resolveAs(ctx, pool, xID, own)
	if err != nil {
		t.Fatalf("own cause: %v", err)
	}
	if got != (outbox.Causation{OriginID: origin, Depth: 4}) {
		t.Errorf("own cause resolved to %+v, want (%s, 4)", got, origin)
	}

	for name, c := range map[string]struct {
		workflowID string
		cause      uuid.UUID
		want       error
	}{
		"another run's row":       {xID, foreign, ErrForeignCause},
		"another tenant's row":    {xID, otherTenant, ErrCauseNotFound},
		"unknown id":              {xID, uuid.New(), ErrCauseNotFound},
		"no cause":                {xID, uuid.Nil, ErrCauseNotFound},
		"not an interpreter run":  {"t:" + tenantA.String() + ":approval:" + x.String(), own, ErrForeignCause},
		"workflow of tenant B":    {"t:" + tenantB.String() + ":" + outbox.InterpreterKind + ":" + x.String(), own, ErrForeignCause},
		"not a relay workflow id": {"interp-" + x.String(), own, ErrForeignCause},
	} {
		t.Run(name, func(t *testing.T) {
			if got, err := resolveAs(ctx, pool, c.workflowID, c.cause); !errors.Is(err, c.want) {
				t.Errorf("resolved %+v, %v; want %v", got, err, c.want)
			}
		})
	}

	// With no verified tenant nothing is read.
	if _, err := resolveCause(context.Background(), nil, xID, own); err == nil {
		t.Error("resolved with no verified tenant")
	}
}

func waitOf(t *testing.T, conn *pgx.Conn, tenant, ref uuid.UUID) *uuid.UUID {
	t.Helper()
	var s *uuid.UUID
	if err := conn.QueryRow(context.Background(), `SELECT record_changed_signal_id FROM workflow_ref
		WHERE tenant_id = $1 AND outbox_id = $2`, tenant, ref).Scan(&s); err != nil {
		t.Fatal(err)
	}
	return s
}

// The wait tracker sets and clears the calling run's
// record_changed_signal_id as nava_worker, on its own row only; a closed run
// or a run of another kind fails non-retryably and changes nothing.
func TestWaitTrackerSetsAndClearsOnlyTheRunsRow(t *testing.T) {
	conn, pool := migrated(t)
	xID, x := seedRef(t, conn, tenantA, outbox.InterpreterKind)
	_, y := seedRef(t, conn, tenantA, outbox.InterpreterKind)
	ctx := ContextWithTenant(context.Background(), tenantA)
	w := WaitTracker{Pool: pool}
	sig := uuid.New()

	if err := w.setWait(ctx, xID, tenantA, &sig); err != nil {
		t.Fatalf("set: %v", err)
	}
	if got := waitOf(t, conn, tenantA, x); got == nil || *got != sig {
		t.Fatalf("wait %v, want %s", got, sig)
	}
	if got := waitOf(t, conn, tenantA, y); got != nil {
		t.Errorf("another run's wait was set: %v", got)
	}
	if err := w.setWait(ctx, xID, tenantA, nil); err != nil {
		t.Fatalf("clear: %v", err)
	}
	if got := waitOf(t, conn, tenantA, x); got != nil {
		t.Errorf("wait %v after clearing, want NULL", got)
	}

	wantNonRetryable := func(name string, err error) {
		t.Helper()
		var app *temporal.ApplicationError
		if !errors.As(err, &app) || !app.NonRetryable() {
			t.Errorf("%s: %v, want a non-retryable ApplicationError", name, err)
		}
	}
	wantNonRetryable("header tenant differs", w.setWait(ctx, xID, tenantB, &sig))
	approvalID, approval := seedRef(t, conn, tenantA, "approval")
	wantNonRetryable("not an interpreter run", w.setWait(ctx, approvalID, tenantA, &sig))
	if got := waitOf(t, conn, tenantA, approval); got != nil {
		t.Errorf("a non-interpreter run's wait was set: %v", got)
	}
	mustExec(t, conn, `UPDATE workflow_ref SET closed_at = now() WHERE tenant_id = $1 AND outbox_id = $2`, tenantA, x)
	wantNonRetryable("closed run", w.setWait(ctx, xID, tenantA, &sig))
	if got := waitOf(t, conn, tenantA, x); got != nil {
		t.Errorf("a closed run's wait was set: %v", got)
	}
}

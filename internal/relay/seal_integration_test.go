//go:build integration

// The ADR-001 A6.3 relay test (P5-31): a start row not tied to a record,
// written by outbox.EnqueueSeal as nava_app, is claimed by nava_relay and
// started through the real SDK client against the in-process frontend.

package relay

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// enqueueSeal is the C5 caller as nava_app: lock the tenant's head, set its
// version to version (A1 bumps it; the test stands in for nava_worker), and
// enqueue the seal row at that version.
func enqueueSeal(t *testing.T, e env, tenant uuid.UUID, version int64) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	mustExec(t, e.super, "UPDATE audit_chain_head SET version = $2 WHERE tenant_id = $1", tenant, version)
	var id uuid.UUID
	err := tenancy.InTenantTx(ctx, e.app, tenant, func(tx pgx.Tx) error {
		var v int64
		if err := tx.QueryRow(ctx, "SELECT version FROM audit_chain_head FOR UPDATE").Scan(&v); err != nil {
			return err
		}
		var err error
		id, err = outbox.EnqueueSeal(ctx, tx, v)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return id
}

// AC (A6.3): an audit.seal start row with no workflow_ref is claimed and
// started as t:<tenant>:audit.seal:<id>, with A.2's start options; a later
// row for the same tenant starts a new run.
func TestSealStartRowStartsItsOwnRun(t *testing.T) {
	e := migrated(t)
	ctx := context.Background()
	fe, addr := startFrontend(t)
	tenant := uuid.New()
	mustExec(t, e.super, "INSERT INTO tenants (id, state) VALUES ($1, 'active')", tenant)
	mustExec(t, e.super, "INSERT INTO audit_chain_head (tenant_id, id) VALUES ($1, $2)", tenant, uuid.Must(uuid.NewV7()))
	d, _, _ := dispatcherOn(t, e.relay, addr, Options{})

	var ids []uuid.UUID
	for i, version := range []int64{1, 2} {
		id := enqueueSeal(t, e, tenant, version)
		ids = append(ids, id)
		if n := count(t, e.super, "SELECT count(*) FROM workflow_ref WHERE tenant_id = $1", tenant); n != 0 {
			t.Fatalf("%d workflow_ref rows, want 0", n)
		}
		if n, err := d.Poll(ctx); err != nil || n != 1 {
			t.Fatalf("poll %d claimed %d, %v", i, n, err)
		}
		workflowID := "t:" + tenant.String() + ":audit.seal:" + id.String()
		ex, ok := fe.execution(workflowID)
		if !ok {
			t.Fatalf("no execution %s", workflowID)
		}
		if ex.kind != outbox.SealKind {
			t.Errorf("workflow type %q, want %s", ex.kind, outbox.SealKind)
		}
		if req := fe.lastStart(); !reusePolicyOK(req) {
			t.Errorf("start options %v, want REJECT_DUPLICATE on %s", req.GetWorkflowIdReusePolicy(), TaskQueue)
		}
		if n := count(t, e.super, "SELECT count(*) FROM outbox WHERE id = $1 AND dispatched_at IS NOT NULL", id); n != 1 {
			t.Errorf("seal row %d is not dispatched", i)
		}
	}
	if ids[0] == ids[1] {
		t.Fatal("the later seal row reused the first row's id")
	}
	if executions, starts, signals := fe.counts(); executions != 2 || starts != 2 || signals != 0 {
		t.Errorf("%d executions, %d starts, %d signals; want 2, 2, 0", executions, starts, signals)
	}
}

// AC (C5 as amended on TICK-440): a seal row enqueued WithDelay(d) is not
// claimable before its available_at (now() + d on the database clock) and is
// claimed after it, by the real claim query as nava_relay.
func TestDelayedSealRowClaimedOnlyAfterAvailableAt(t *testing.T) {
	e := migrated(t)
	ctx := context.Background()
	tenant := uuid.New()
	mustExec(t, e.super, "INSERT INTO tenants (id, state) VALUES ($1, 'active')", tenant)
	mustExec(t, e.super, "INSERT INTO audit_chain_head (tenant_id, id) VALUES ($1, $2)", tenant, uuid.Must(uuid.NewV7()))
	r := newProbe(t, e.relay, Options{})

	const delay = 2 * time.Second
	var id uuid.UUID
	err := tenancy.InTenantTx(ctx, e.app, tenant, func(tx pgx.Tx) error {
		var err error
		id, err = outbox.EnqueueSeal(ctx, tx, 1, outbox.WithDelay(delay))
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	var availableAt time.Time
	var gapUS int64
	if err := e.super.QueryRow(ctx, `SELECT available_at, (extract(epoch FROM available_at - created_at) * 1000000)::bigint
		FROM outbox WHERE tenant_id = $1 AND id = $2`, tenant, id).Scan(&availableAt, &gapUS); err != nil {
		t.Fatal(err)
	}
	if got := time.Duration(gapUS) * time.Microsecond; got != delay {
		t.Fatalf("available_at - created_at = %s, want %s", got, delay)
	}

	claimedOurs := func() bool {
		cs, err := r.Claim(ctx)
		if err != nil {
			t.Fatal(err)
		}
		_, ok := claimedIDs(cs)[id]
		return ok
	}
	var dbNow time.Time
	if err := e.super.QueryRow(ctx, "SELECT clock_timestamp()").Scan(&dbNow); err != nil {
		t.Fatal(err)
	}
	if !dbNow.Before(availableAt) {
		t.Fatalf("the database clock passed available_at before the first claim; the delay is too short for this host")
	}
	if claimedOurs() {
		t.Fatal("the delayed seal row was claimed before its available_at")
	}
	waitFor(t, claimedOurs)
	if err := e.super.QueryRow(ctx, "SELECT clock_timestamp()").Scan(&dbNow); err != nil {
		t.Fatal(err)
	}
	if dbNow.Before(availableAt) {
		t.Errorf("claimed at %s, before available_at %s", dbNow, availableAt)
	}
}

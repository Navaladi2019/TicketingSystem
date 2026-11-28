//go:build integration

package isolation

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel/metric/noop"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"

	"github.com/Navaladi2019/TicketingSystem/internal/fieldindex"
	"github.com/Navaladi2019/TicketingSystem/internal/fieldindex/backfill"
	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
	"github.com/Navaladi2019/TicketingSystem/internal/workerguard"
)

// S21Backfill3b runs row 3b's check on the suite's cluster. It lives in a
// test file: only internal/fieldindex may name the field-index workflow types
// outside tests (TestOnlyFieldindexNamesItsWorkflowTypes), and the check
// drives them on the Temporal SDK test environment. fieldindex_attack_paths.go
// registers the row and calls it through backfillAttacker.
func (c cluster) S21Backfill3b(ctx context.Context, e *etIso, fx *s21Fixture) error {
	return s21Backfill3b(ctx, c, e, fx)
}

// s21Backfill3b is S21 attack row 3b (P7-08b, TICK-975; ADR-006 D3, D6):
// the real coordinator, FieldIndexBackfill and their activities, on
// nava_worker under RLS, behind the worker guard and the nava-tenant
// Propagator, as cmd/nava-worker runs them, started as the relay starts them
// for an outbox row of tenant A.
//
// Before it runs, tenant A's note and tenant B's note are both backfilling,
// so tenant B's field is a live target a fooled backfill would fill and flip.
// Tenant A's note side rows are deleted, so the control backfill has work.
//
//  1. Case 1: tenant A's row names tenant B's field id. The coordinator gets
//     that item and, as the control, one for tenant A's own note. It starts a
//     child for tenant A's note only, which fills it (2 side rows) and flips
//     it ready; tenant B's registry and side rows do not move, and no side
//     row pairs tenant A with tenant B's field.
//  2. Case 1, one layer down: FieldIndexBackfill started directly under
//     tenant A with tenant B's field id in its workflow id, as a coordinator
//     fooled into it would start it. Its batch and flip run (the control),
//     and write nothing.
//  3. Case 2: tenant A's row whose payload tenant_id is tenant B, naming
//     tenant B's field and then tenant A's own. The workflow id and the relay
//     header say tenant A; the coordinator fails closed, non-retryable
//     nava.field_index.NoChild, before any activity runs, and nothing moves:
//     not even tenant A's own backfilling note.
func s21Backfill3b(ctx context.Context, cl Cluster, e *etIso, _ *s21Fixture) error {
	a, b := uuid.MustParse(TenantA), uuid.MustParse(TenantB)
	ids := map[string]uuid.UUID{}
	for _, f := range []struct{ tenant, name string }{{TenantA, "A"}, {TenantB, "B"}} {
		id, err := s21FieldID(ctx, e, f.tenant, "note")
		if err != nil {
			return err
		}
		ids[f.name] = uuid.MustParse(id)
	}
	noteA, noteB := ids["A"], ids["B"]
	if err := s21SetState(ctx, e, TenantB, "note", "backfilling", 13); err != nil {
		return err
	}
	resetA := func() error {
		if err := s21SetState(ctx, e, TenantA, "note", "backfilling", 0); err != nil {
			return err
		}
		_, err := e.super.Exec(ctx, `DELETE FROM rfv_text WHERE tenant_id = $1 AND field_id = $2`, TenantA, noteA)
		return err
	}
	if err := resetA(); err != nil {
		return err
	}
	regB, err := s21Registry(ctx, e, TenantB)
	if err != nil {
		return err
	}
	st, err := s21Read3b(ctx, e, noteA, noteB)
	if err != nil {
		return err
	}
	if want := (s3bState{aState: "backfilling", aProgress: 0, aSide: 0, bSide: 2}); st != want {
		return fmt.Errorf("control: before the runs %+v, want %+v", st, want)
	}
	untouched := st

	// The pool is nava_worker, logged in directly: the barrier and a batch
	// may hold a connection each.
	pool, err := cl.PoolSize("nava_worker", 4)
	if err != nil {
		return err
	}
	defer pool.Close()
	acts := &backfill.Activities{Pool: pool}

	// The outbox rows, through the only enqueue path, in tenant A.
	rowB, err := s21Enqueue3b(ctx, e, a, noteB)
	if err != nil {
		return fmt.Errorf("control: enqueue tenant B's field in tenant A: %w", err)
	}
	rowA, err := s21Enqueue3b(ctx, e, a, noteA)
	if err != nil {
		return fmt.Errorf("control: enqueue tenant A's note: %w", err)
	}

	// 1. Case 1 through the coordinator.
	r := s3bRun(a, fieldindex.CoordinatorWorkflowID(TenantA), acts)
	r.signal(rowB.item(noteB), 0)
	r.signal(rowA.item(noteA), time.Second)
	if err := r.execute(fieldindex.CoordinatorWorkflowType, workerguard.RunInput{Delivery: rowB.delivery}); err != nil {
		return fmt.Errorf("case 1: the coordinator in tenant A = %w, want nil", err)
	}
	wantChild := backfill.ChildID(a, fieldindex.BackfillWorkflowType, noteA, 1)
	if got := r.childLog(); len(got) != 2 || got[0] != "start "+wantChild || got[1] != "done "+wantChild {
		return fmt.Errorf("case 1: the coordinator's children = %v, want only %s, tenant A's note (control)", got, wantChild)
	}
	if got := r.count(backfill.ClassifyActivity); got != 2 {
		return fmt.Errorf("control: case 1 classified %d items, want 2 (tenant B's field id, tenant A's note)", got)
	}
	if err := s21Check3b(ctx, e, noteA, noteB, regB, s3bState{aState: "ready", aProgress: 100, aSide: 2, bSide: 2}); err != nil {
		return fmt.Errorf("case 1 (tenant A's row names tenant B's field): %w", err)
	}

	// 2. Case 1, one layer down: the child itself, under tenant A, on
	// tenant B's field.
	if err := resetA(); err != nil {
		return err
	}
	child := backfill.ChildID(a, fieldindex.BackfillWorkflowType, noteB, 1)
	r = s3bRun(a, child, acts)
	if err := r.execute(fieldindex.BackfillWorkflowType, backfill.Input{FieldID: noteB}); err != nil {
		return fmt.Errorf("case 1: FieldIndexBackfill %s = %w, want nil (it finds no field and writes nothing)", child, err)
	}
	if r.count(backfill.BatchActivity) != 1 || r.count(backfill.ReadyActivity) != 1 {
		return fmt.Errorf("control: FieldIndexBackfill %s ran %v, want one batch and one flip", child, r.counts())
	}
	if err := s21Check3b(ctx, e, noteA, noteB, regB, untouched); err != nil {
		return fmt.Errorf("case 1 (FieldIndexBackfill under tenant A on tenant B's field): %w", err)
	}

	// 3. Case 2: the payload names tenant B, the row and the header tenant A.
	for _, field := range []uuid.UUID{noteB, noteA} {
		forged, err := rowA.forge(b, field)
		if err != nil {
			return err
		}
		r = s3bRun(a, fieldindex.CoordinatorWorkflowID(TenantA), acts)
		r.signal(forged.item(field), 0)
		err = r.execute(fieldindex.CoordinatorWorkflowType, workerguard.RunInput{Delivery: forged.delivery})
		// The guard withholds the failure from the history (nava.Opaque) and
		// logs its type: the coordinator's own refusal, not some other error.
		var app *temporal.ApplicationError
		if !errors.As(err, &app) || !app.NonRetryable() || !strings.Contains(r.log(), backfill.ErrTypeNoChild) {
			return fmt.Errorf("case 2 (payload tenant_id is tenant B, field %s): the coordinator in tenant A = %w, worker log %s; want a non-retryable %s",
				s3bName(field, noteA), err, r.log(), backfill.ErrTypeNoChild)
		}
		if n := r.counts(); len(n) != 0 {
			return fmt.Errorf("case 2 (payload tenant_id is tenant B, field %s): activities ran %v, want none", s3bName(field, noteA), n)
		}
		if err := s21Check3b(ctx, e, noteA, noteB, regB, untouched); err != nil {
			return fmt.Errorf("case 2 (payload tenant_id is tenant B, field %s): %w", s3bName(field, noteA), err)
		}
	}
	return nil
}

func s3bName(field, noteA uuid.UUID) string {
	if field == noteA {
		return "tenant A's note"
	}
	return "tenant B's note"
}

// s3bState is what row 3b watches: tenant A's note registry state, progress
// and side rows, tenant B's note side rows, and the side rows anywhere that
// pair tenant A with tenant B's field (cross).
type s3bState struct {
	aState    string
	aProgress int
	aSide     int
	bSide     int
	cross     int
}

func s21Read3b(ctx context.Context, e *etIso, noteA, noteB uuid.UUID) (s3bState, error) {
	var s s3bState
	err := e.super.QueryRow(ctx, `SELECT f.state, f.progress,
		(SELECT count(*) FROM rfv_text WHERE tenant_id = $1 AND field_id = $3),
		(SELECT count(*) FROM rfv_text WHERE tenant_id = $2 AND field_id = $4),
		(SELECT count(*) FROM (SELECT 1 FROM rfv_text WHERE tenant_id = $1 AND field_id = $4
			UNION ALL SELECT 1 FROM rfv_num WHERE tenant_id = $1 AND field_id = $4
			UNION ALL SELECT 1 FROM rfv_ts WHERE tenant_id = $1 AND field_id = $4
			UNION ALL SELECT 1 FROM rfv_bool WHERE tenant_id = $1 AND field_id = $4) x)
		FROM record_index_fields f WHERE f.tenant_id = $1 AND f.id = $3`, TenantA, TenantB, noteA, noteB).
		Scan(&s.aState, &s.aProgress, &s.aSide, &s.bSide, &s.cross)
	return s, err
}

// s21Check3b fails when tenant B's registry moved or the state is not want.
func s21Check3b(ctx context.Context, e *etIso, noteA, noteB uuid.UUID, regB string, want s3bState) error {
	if got, err := s21Registry(ctx, e, TenantB); err != nil {
		return fmt.Errorf("read tenant B's registry: %w", err)
	} else if got != regB {
		return fmt.Errorf("tenant B's registry moved:\n%s\n%s", regB, got)
	}
	got, err := s21Read3b(ctx, e, noteA, noteB)
	if err != nil {
		return err
	}
	if got != want {
		return fmt.Errorf("tenant A's note, tenant B's note side rows and the cross rows = %+v, want %+v", got, want)
	}
	return nil
}

// s3bRow is one outbox row as the relay hands it on: the Delivery that
// starts the coordinator and the payload its signal is built from.
type s3bRow struct {
	delivery outbox.Delivery
}

// item is the relay's signal body for the row (relay.enqueueItem): the
// payload's field id, the row's id, version 1.
func (r s3bRow) item(field uuid.UUID) fieldindex.EnqueueItem {
	return fieldindex.EnqueueItem{Op: fieldindex.OpBackfill, FieldID: field.String(), RequestedVersion: 1,
		OutboxID: r.delivery.OutboxID.String()}
}

// forge is r with its payload's tenant_id set to tenant and field_id to
// field: a row of tenant A whose payload names another tenant, which the
// relay refuses (relay.enqueueItem) and the coordinator must refuse too.
func (r s3bRow) forge(tenant, field uuid.UUID) (s3bRow, error) {
	var p map[string]any
	if err := json.Unmarshal(r.delivery.Payload, &p); err != nil {
		return s3bRow{}, err
	}
	if _, ok := p["tenant_id"]; !ok {
		return s3bRow{}, fmt.Errorf("control: the backfill payload %s has no tenant_id", r.delivery.Payload)
	}
	p["tenant_id"], p["field_id"] = tenant.String(), field.String()
	raw, err := json.Marshal(p)
	if err != nil {
		return s3bRow{}, err
	}
	out := r
	out.delivery.OutboxID = uuid.Must(uuid.NewV7())
	out.delivery.Payload = raw
	return out, nil
}

// s21Enqueue3b writes a backfill outbox row for field in tenant through
// outbox.EnqueueFieldIndexBackfill as nava_app, and reads it back past RLS
// (nava_app cannot read outbox payloads). s21Cleanup deletes it.
func s21Enqueue3b(ctx context.Context, e *etIso, tenant, field uuid.UUID) (s3bRow, error) {
	var id uuid.UUID
	if err := tenancy.InTenantTx(ctx, e.pool, tenant, func(tx pgx.Tx) error {
		var err error
		id, err = outbox.EnqueueFieldIndexBackfill(ctx, tx, field, 1)
		return err
	}); err != nil {
		return s3bRow{}, err
	}
	var rowTenant string
	var payload []byte
	if err := e.super.QueryRow(ctx, `SELECT tenant_id::text, payload::text FROM outbox WHERE id = $1`, id).Scan(&rowTenant, &payload); err != nil {
		return s3bRow{}, err
	}
	if rowTenant != tenant.String() {
		return s3bRow{}, fmt.Errorf("the row is tenant %s's, want %s", rowTenant, tenant)
	}
	return s3bRow{delivery: outbox.Delivery{OutboxID: id, KindVersion: 1, Payload: payload}}, nil
}

// s3bRunner is one workflow execution on the SDK test environment, behind
// the worker guard, with the relay's nava-tenant header.
type s3bRunner struct {
	env *testsuite.TestWorkflowEnvironment

	mu       sync.Mutex
	logs     bytes.Buffer // the guard's log, which has a withheld failure's type
	acts     map[string]int
	children []string
}

func s3bRun(tenant uuid.UUID, workflowID string, acts *backfill.Activities) *s3bRunner {
	var s testsuite.WorkflowTestSuite
	r := &s3bRunner{env: s.NewTestWorkflowEnvironment(), acts: map[string]int{}}
	g, err := workerguard.New(slog.New(slog.NewJSONHandler(s3bWriter{r}, nil)), noop.NewMeterProvider())
	if err != nil {
		panic(err) // a fixed logger and a no-op provider
	}
	p, err := converter.GetDefaultDataConverter().ToPayload(tenant.String())
	if err != nil {
		panic(err)
	}
	r.env.SetWorkerOptions(workerguard.WorkerOptions(worker.Options{
		Interceptors:             []interceptor.WorkerInterceptor{&s3bRelayHeader{tenant: p}},
		DeadlockDetectionTimeout: 30 * time.Second,
	}, g))
	r.env.SetContextPropagators([]workflow.ContextPropagator{workerguard.Propagator{}})
	r.env.SetStartWorkflowOptions(client.StartWorkflowOptions{ID: workflowID})
	r.env.SetHeader(&commonpb.Header{Fields: map[string]*commonpb.Payload{workerguard.TenantHeader: p}})
	r.env.SetTestTimeout(2 * time.Minute)
	acts.Register(r.env)
	(&backfill.Coordinator{IdleTimeout: time.Minute}).Register(r.env)
	(&backfill.Backfill{}).Register(r.env)
	r.env.SetOnActivityStartedListener(func(info *activity.Info, _ context.Context, _ converter.EncodedValues) {
		r.mu.Lock()
		r.acts[info.ActivityType.Name]++
		r.mu.Unlock()
	})
	r.env.SetOnChildWorkflowStartedListener(func(info *workflow.Info, _ workflow.Context, _ converter.EncodedValues) {
		r.mu.Lock()
		r.children = append(r.children, "start "+info.WorkflowExecution.ID)
		r.mu.Unlock()
	})
	r.env.SetOnChildWorkflowCompletedListener(func(info *workflow.Info, _ converter.EncodedValue, _ error) {
		r.mu.Lock()
		r.children = append(r.children, "done "+info.WorkflowExecution.ID)
		r.mu.Unlock()
	})
	return r
}

// s3bWriter appends to r's log under its lock.
type s3bWriter struct{ r *s3bRunner }

func (w s3bWriter) Write(p []byte) (int, error) {
	w.r.mu.Lock()
	defer w.r.mu.Unlock()
	return w.r.logs.Write(p)
}

func (r *s3bRunner) log() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.logs.String()
}

func (r *s3bRunner) signal(item fieldindex.EnqueueItem, delay time.Duration) {
	r.env.RegisterDelayedCallback(func() { r.env.SignalWorkflow(fieldindex.SignalEnqueue, item) }, delay)
}

func (r *s3bRunner) execute(wt string, in any) error {
	r.env.ExecuteWorkflow(wt, in)
	if !r.env.IsWorkflowCompleted() {
		return errors.New("control: the workflow did not complete")
	}
	return r.env.GetWorkflowError()
}

func (r *s3bRunner) count(activityType string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.acts[activityType]
}

func (r *s3bRunner) counts() map[string]int {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := map[string]int{}
	for k, v := range r.acts {
		out[k] = v
	}
	return out
}

func (r *s3bRunner) childLog() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.children...)
}

// s3bRelayHeader stands in for the transport: the relay's client Propagator
// writes the nava-tenant header on its SignalWithStart, and the SDK test
// environment's SignalWorkflow sends none, so this interceptor, ahead of the
// guard, sets the row's tenant on each signal.
type s3bRelayHeader struct {
	interceptor.WorkerInterceptorBase
	tenant *commonpb.Payload
}

func (h *s3bRelayHeader) InterceptWorkflow(_ workflow.Context, next interceptor.WorkflowInboundInterceptor) interceptor.WorkflowInboundInterceptor {
	return &s3bRelayHeaderWorkflow{WorkflowInboundInterceptorBase: interceptor.WorkflowInboundInterceptorBase{Next: next}, tenant: h.tenant}
}

type s3bRelayHeaderWorkflow struct {
	interceptor.WorkflowInboundInterceptorBase
	tenant *commonpb.Payload
}

func (w *s3bRelayHeaderWorkflow) HandleSignal(ctx workflow.Context, in *interceptor.HandleSignalInput) error {
	if h := interceptor.WorkflowHeader(ctx); h != nil {
		h[workerguard.TenantHeader] = w.tenant
	}
	return w.Next.HandleSignal(ctx, in)
}

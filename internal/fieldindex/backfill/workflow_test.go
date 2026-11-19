package backfill

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"

	"github.com/Navaladi2019/TicketingSystem/internal/fieldindex"
	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
	"github.com/Navaladi2019/TicketingSystem/internal/workerguard"
)

// Unit tests of the two workflows on the SDK test environment with stubbed
// activities: the paths that need no database. The integration tests run
// the real activities against PostgreSQL behind the real guard.

const unitTenant = "0192aaaa-0000-7000-8000-0000000000aa"

// stubs records what the stubbed activities and children saw.
type stubs struct {
	mu       sync.Mutex
	classify map[uuid.UUID]Classified // the answer per field; missing: {}
	failOn   map[uuid.UUID]bool       // the child of this field fails
	children []string                 // child workflow ids, in start order
	batches  int
	batchErr error
	cursors  []uuid.UUID  // the cursors Batch returns, in order, then {}
	coord    *Coordinator // registered instead of the defaults when set
	// realBackfill registers the real FieldIndexBackfill (backfillRun, or
	// backfill when set) instead of the stub child.
	realBackfill bool
	backfill     *Backfill
	// failing names the activity (Barrier, Ready, Classify) that fails.
	failing string
}

func (s *stubs) fail(name string) error {
	if s.failing == name {
		return temporal.NewNonRetryableApplicationError(name+" failed", "test.Fail", nil)
	}
	return nil
}

func newEnv(t *testing.T, s *stubs, workflowID string) *testsuite.TestWorkflowEnvironment {
	t.Helper()
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.SetStartWorkflowOptions(client.StartWorkflowOptions{ID: workflowID})
	env.SetTestTimeout(time.Minute)
	env.RegisterActivityWithOptions(func(_ context.Context, in FieldRef) (Classified, error) {
		s.mu.Lock()
		defer s.mu.Unlock()
		return s.classify[in.FieldID], s.fail(ClassifyActivity)
	}, activity.RegisterOptions{Name: ClassifyActivity})
	env.RegisterActivityWithOptions(func(context.Context) error { return s.fail(BarrierActivity) }, activity.RegisterOptions{Name: BarrierActivity})
	env.RegisterActivityWithOptions(func(context.Context) error { return s.fail(ReadyActivity) }, activity.RegisterOptions{Name: ReadyActivity})
	env.RegisterActivityWithOptions(func(_ context.Context, _ Cursor) (Cursor, error) {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.batches++
		if s.batchErr != nil {
			return Cursor{}, temporal.NewNonRetryableApplicationError(s.batchErr.Error(), "test.Batch", nil)
		}
		if len(s.cursors) == 0 {
			return Cursor{}, nil
		}
		c := s.cursors[0]
		s.cursors = s.cursors[1:]
		return Cursor{Cursor: &c}, nil
	}, activity.RegisterOptions{Name: BatchActivity})
	child := func(ctx workflow.Context) error {
		id := workflow.GetInfo(ctx).WorkflowExecution.ID
		s.mu.Lock()
		s.children = append(s.children, id)
		fail := false
		for f := range s.failOn {
			if c, err := parseAnyChildID(id); err == nil && c.field == f {
				fail = true
			}
		}
		s.mu.Unlock()
		if fail {
			return temporal.NewNonRetryableApplicationError("child failed", "test.Child", nil)
		}
		return nil
	}
	switch {
	case s.backfill != nil:
		env.RegisterWorkflowWithOptions(s.backfill.Run, workflow.RegisterOptions{Name: fieldindex.BackfillWorkflowType})
	case s.realBackfill:
		env.RegisterWorkflowWithOptions(backfillRun, workflow.RegisterOptions{Name: fieldindex.BackfillWorkflowType})
	default:
		env.RegisterWorkflowWithOptions(func(ctx workflow.Context, _ Input) error { return child(ctx) },
			workflow.RegisterOptions{Name: fieldindex.BackfillWorkflowType})
	}
	env.RegisterWorkflowWithOptions(func(ctx workflow.Context, _ FieldRef) error { return child(ctx) },
		workflow.RegisterOptions{Name: fieldindex.RemoveWorkflowType})
	if s.coord != nil {
		env.RegisterWorkflowWithOptions(s.coord.Run, workflow.RegisterOptions{Name: fieldindex.CoordinatorWorkflowType})
	} else {
		env.RegisterWorkflowWithOptions(coordinatorRun, workflow.RegisterOptions{Name: fieldindex.CoordinatorWorkflowType})
	}
	return env
}

func coordinatorID() string { return fieldindex.CoordinatorWorkflowID(unitTenant) }

func start() workerguard.RunInput {
	return workerguard.RunInput{Delivery: outbox.Delivery{OutboxID: uuid.Must(uuid.NewV7()), KindVersion: 1,
		Payload: json.RawMessage(`{"tenant_id":"` + unitTenant + `","field_id":"` + uuid.NewString() + `"}`)}}
}

func enqueue(env *testsuite.TestWorkflowEnvironment, field uuid.UUID, version int, outboxID string, delay time.Duration) {
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(fieldindex.SignalEnqueue, fieldindex.EnqueueItem{Op: fieldindex.OpBackfill,
			FieldID: field.String(), RequestedVersion: version, OutboxID: outboxID})
	}, delay)
}

func TestCoordinatorDedupsAndOrders(t *testing.T) {
	a, b := uuid.New(), uuid.New()
	s := &stubs{classify: map[uuid.UUID]Classified{a: {Backfill: &a}, b: {Backfill: &b}}}
	env := newEnv(t, s, coordinatorID())
	oa := uuid.Must(uuid.NewV7()).String()
	enqueue(env, a, 1, oa, 0)
	enqueue(env, a, 1, oa, time.Second)                                 // same outbox row again
	enqueue(env, a, 1, uuid.Must(uuid.NewV7()).String(), 2*time.Second) // same (field, version), new row
	enqueue(env, b, 1, uuid.Must(uuid.NewV7()).String(), 3*time.Second)
	env.RegisterDelayedCallback(func() { // malformed items are dropped
		env.SignalWorkflow(fieldindex.SignalEnqueue, fieldindex.EnqueueItem{Op: "drop", FieldID: a.String(), RequestedVersion: 1,
			OutboxID: uuid.Must(uuid.NewV7()).String()})
	}, 4*time.Second)
	env.ExecuteWorkflow(fieldindex.CoordinatorWorkflowType, start())
	if err := env.GetWorkflowError(); err != nil {
		t.Fatal(err)
	}
	tenant := uuid.MustParse(unitTenant)
	want := []string{ChildID(tenant, fieldindex.BackfillWorkflowType, a, 1), ChildID(tenant, fieldindex.BackfillWorkflowType, b, 1)}
	if !slices.Equal(s.children, want) {
		t.Fatalf("children %v, want %v", s.children, want)
	}
}

func TestCoordinatorRoutesRemovalAndSkipsReady(t *testing.T) {
	gone, ready := uuid.New(), uuid.New()
	s := &stubs{classify: map[uuid.UUID]Classified{gone: {Remove: &gone}}}
	env := newEnv(t, s, coordinatorID())
	enqueue(env, ready, 1, uuid.Must(uuid.NewV7()).String(), 0)
	enqueue(env, gone, 1, uuid.Must(uuid.NewV7()).String(), 0)
	env.ExecuteWorkflow(fieldindex.CoordinatorWorkflowType, start())
	if err := env.GetWorkflowError(); err != nil {
		t.Fatal(err)
	}
	want := []string{ChildID(uuid.MustParse(unitTenant), fieldindex.RemoveWorkflowType, gone, 1)}
	if !slices.Equal(s.children, want) {
		t.Fatalf("children %v, want only the removal %v", s.children, want)
	}
}

// A failed child is logged and the queue moves on.
func TestCoordinatorContinuesPastAFailedChild(t *testing.T) {
	a, b := uuid.New(), uuid.New()
	s := &stubs{classify: map[uuid.UUID]Classified{a: {Backfill: &a}, b: {Backfill: &b}}, failOn: map[uuid.UUID]bool{a: true}}
	env := newEnv(t, s, coordinatorID())
	enqueue(env, a, 1, uuid.Must(uuid.NewV7()).String(), 0)
	enqueue(env, b, 1, uuid.Must(uuid.NewV7()).String(), 0)
	env.ExecuteWorkflow(fieldindex.CoordinatorWorkflowType, start())
	if err := env.GetWorkflowError(); err != nil {
		t.Fatal(err)
	}
	if len(s.children) != 2 {
		t.Fatalf("children %v, want both", s.children)
	}
}

// After MaxItems the run continues as new with the rest, in order, and the
// Seen set; a continued run's {pending} restores them unpinned.
func TestCoordinatorContinuesAsNewWithItsQueue(t *testing.T) {
	a, b, c := uuid.New(), uuid.New(), uuid.New()
	s := &stubs{classify: map[uuid.UUID]Classified{a: {Backfill: &a}, b: {Backfill: &b}, c: {Backfill: &c}},
		coord: &Coordinator{MaxItems: 1, IdleTimeout: time.Minute}}
	env := newEnv(t, s, coordinatorID())
	for _, f := range []uuid.UUID{a, b, c} {
		enqueue(env, f, 2, uuid.Must(uuid.NewV7()).String(), 0)
	}
	env.ExecuteWorkflow(fieldindex.CoordinatorWorkflowType, start())
	var can *workflow.ContinueAsNewError
	if err := env.GetWorkflowError(); !errors.As(err, &can) {
		t.Fatalf("error %v, want continue-as-new", err)
	}
	var next workerguard.RunInput
	if err := json.Unmarshal(can.Input.GetPayloads()[0].GetData(), &next); err != nil {
		t.Fatal(err)
	}
	var refs pendingRefs
	if err := json.Unmarshal(next.Refs, &refs); err != nil || !slices.Equal(refs.Pending, []uuid.UUID{b, c}) {
		t.Fatalf("refs %s (%v), want [b c]", next.Refs, err)
	}
	if next.Seen.Len() != 3 || string(next.Payload) != "{}" {
		t.Errorf("continued input %s", can.Input.GetPayloads()[0].GetData())
	}

	s2 := &stubs{classify: s.classify}
	env2 := newEnv(t, s2, coordinatorID())
	env2.ExecuteWorkflow(fieldindex.CoordinatorWorkflowType, next)
	if err := env2.GetWorkflowError(); err != nil {
		t.Fatal(err)
	}
	tenant := uuid.MustParse(unitTenant)
	want := []string{ChildID(tenant, fieldindex.BackfillWorkflowType, b, 0), ChildID(tenant, fieldindex.BackfillWorkflowType, c, 0)}
	if !slices.Equal(s2.children, want) {
		t.Fatalf("continued children %v, want %v (unpinned)", s2.children, want)
	}
}

// TICK-748 condition, defence in depth: a start payload naming another
// tenant than the workflow id is refused before anything runs.
func TestCoordinatorRefusesAnotherTenantsPayload(t *testing.T) {
	a := uuid.New()
	for name, payload := range map[string]string{
		"other tenant":  `{"tenant_id":"0192aaaa-0000-7000-8000-0000000000bb","field_id":"` + a.String() + `"}`,
		"not an object": `[]`,
	} {
		t.Run(name, func(t *testing.T) {
			s := &stubs{classify: map[uuid.UUID]Classified{a: {Backfill: &a}}}
			env := newEnv(t, s, coordinatorID())
			enqueue(env, a, 1, uuid.Must(uuid.NewV7()).String(), 0)
			in := start()
			in.Payload = json.RawMessage(payload)
			env.ExecuteWorkflow(fieldindex.CoordinatorWorkflowType, in)
			if err := env.GetWorkflowError(); err == nil {
				t.Fatal("the coordinator accepted another tenant's payload")
			}
			if len(s.children) != 0 {
				t.Errorf("children %v, want none", s.children)
			}
		})
	}
}

func TestCoordinatorRefusesBadIDAndRefs(t *testing.T) {
	s := &stubs{}
	env := newEnv(t, s, "not-a-tenant-id")
	env.ExecuteWorkflow(fieldindex.CoordinatorWorkflowType, start())
	if err := env.GetWorkflowError(); err == nil {
		t.Error("a coordinator id with no tenant ran")
	}
	env = newEnv(t, s, coordinatorID())
	in := start()
	in.Refs = json.RawMessage(`{"pending":"x"}`)
	env.ExecuteWorkflow(fieldindex.CoordinatorWorkflowType, in)
	if err := env.GetWorkflowError(); err == nil {
		t.Error("malformed refs were accepted")
	}
}

func backfillEnv(t *testing.T, s *stubs, b *Backfill, field uuid.UUID) *testsuite.TestWorkflowEnvironment {
	t.Helper()
	s.realBackfill, s.backfill = true, b
	return newEnv(t, s, ChildID(uuid.MustParse(unitTenant), fieldindex.BackfillWorkflowType, field, 1))
}

func TestBackfillRunsBatchesThenReady(t *testing.T) {
	f := uuid.New()
	s := &stubs{cursors: []uuid.UUID{uuid.New(), uuid.New()}}
	env := backfillEnv(t, s, nil, f)
	env.ExecuteWorkflow(fieldindex.BackfillWorkflowType, Input{FieldID: f})
	if err := env.GetWorkflowError(); err != nil {
		t.Fatal(err)
	}
	if s.batches != 3 {
		t.Errorf("%d batches, want 3 (two cursors, then done)", s.batches)
	}
}

func TestBackfillContinuesAsNewWithItsCursor(t *testing.T) {
	f, c1, c2 := uuid.New(), uuid.New(), uuid.New()
	s := &stubs{cursors: []uuid.UUID{c1, c2, uuid.New()}}
	env := backfillEnv(t, s, &Backfill{MaxBatchesPerRun: 2}, f)
	env.ExecuteWorkflow(fieldindex.BackfillWorkflowType, Input{FieldID: f})
	var can *workflow.ContinueAsNewError
	if err := env.GetWorkflowError(); !errors.As(err, &can) {
		t.Fatalf("error %v, want continue-as-new", err)
	}
	var next Input
	if err := json.Unmarshal(can.Input.GetPayloads()[0].GetData(), &next); err != nil || next.FieldID != f ||
		next.Cursor == nil || *next.Cursor != c2 {
		t.Fatalf("continued input %+v (%v), want the field and cursor %s", next, err, c2)
	}
}

func TestBackfillRefusesAMismatchedInputAndReportsBatchErrors(t *testing.T) {
	f := uuid.New()
	env := backfillEnv(t, &stubs{}, nil, f)
	env.ExecuteWorkflow(fieldindex.BackfillWorkflowType, Input{FieldID: uuid.New()})
	if err := env.GetWorkflowError(); err == nil {
		t.Error("an input naming another field than the id was accepted")
	}
	s := &stubs{batchErr: errors.New("boom")}
	env = backfillEnv(t, s, nil, f)
	env.ExecuteWorkflow(fieldindex.BackfillWorkflowType, Input{FieldID: f})
	if err := env.GetWorkflowError(); err == nil {
		t.Error("a failed batch was not reported")
	}
}

func TestRegisterWorkflowRegistersBothTypes(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	RegisterWorkflow(env)
	(&Activities{}).Register(env)
	env.SetStartWorkflowOptions(client.StartWorkflowOptions{ID: "no-tenant"})
	env.ExecuteWorkflow(fieldindex.BackfillWorkflowType, Input{FieldID: uuid.New()})
	if err := env.GetWorkflowError(); err == nil {
		t.Error("a backfill with no child id ran")
	}
}

func TestPayloadTenantMatches(t *testing.T) {
	tenant := uuid.MustParse(unitTenant)
	for payload, want := range map[string]bool{
		"": true, `{}`: true, `{"tenant_id":"` + unitTenant + `"}`: true,
		`{"tenant_id":"0192aaaa-0000-7000-8000-0000000000bb"}`: false, `[`: false,
	} {
		if got := payloadTenantMatches(json.RawMessage(payload), tenant); got != want {
			t.Errorf("payloadTenantMatches(%s) = %v, want %v", payload, got, want)
		}
	}
}

func TestValueColumnAndRemovalIDs(t *testing.T) {
	for _, v := range fieldindex.ValueTypes() {
		if _, ok := valueColumn(v); !ok {
			t.Errorf("no value column for %s", v)
		}
	}
	if _, ok := valueColumn("numeric"); ok {
		t.Error("a value column for an unknown type")
	}
	f := uuid.New()
	id := ChildID(uuid.MustParse(unitTenant), fieldindex.RemoveWorkflowType, f, 3)
	if c, err := parseAnyChildID(id); err != nil || c.field != f || c.version != 3 {
		t.Errorf("parseAnyChildID(%s) = %+v, %v", id, c, err)
	}
}

// The activities refuse to run outside a field-index child or a tenant,
// before touching the database (the pool is nil here).
func TestActivitiesRefuseWithoutChildOrTenant(t *testing.T) {
	a := &Activities{}
	if _, err := a.Classify(context.Background(), FieldRef{FieldID: uuid.New()}); err == nil {
		t.Error("Classify ran with no tenant")
	}
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestActivityEnvironment()
	env.RegisterActivityWithOptions(a.Batch, activity.RegisterOptions{Name: BatchActivity})
	env.RegisterActivityWithOptions(a.Ready, activity.RegisterOptions{Name: ReadyActivity})
	if _, err := env.ExecuteActivity(BatchActivity, Cursor{}); err == nil {
		t.Error("Batch ran outside a backfill child")
	}
	if _, err := env.ExecuteActivity(ReadyActivity); err == nil {
		t.Error("Ready ran outside a backfill child")
	}
}

func TestDefaultsMetricsAndLogAttrs(t *testing.T) {
	a := &Activities{}
	if a.batchSize() != DefaultBatchSize || a.slowAfter() != DefaultSlowAfter || a.logger() == nil {
		t.Error("defaults")
	}
	var nilMetrics *Metrics
	nilMetrics.barrierSlow(context.Background()) // a nil *Metrics records nothing
	reader := sdkmetric.NewManualReader()
	m, err := NewMetrics(sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader)))
	if err != nil {
		t.Fatal(err)
	}
	m.barrierSlow(context.Background())
	var rm metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &rm); err != nil || len(rm.ScopeMetrics) != 1 ||
		rm.ScopeMetrics[0].Metrics[0].Name != "field_index_barrier_slow" {
		t.Fatalf("metrics %+v, %v", rm, err)
	}
	a.Log = slog.New(slog.DiscardHandler)
	attrs := a.attrs(workerguard.ContextWithTenant(context.Background(), uuid.MustParse(unitTenant)), slog.Int("n", 1))
	if len(attrs) != 2 {
		t.Errorf("attrs %v, want the tenant and the extra one (no activity context)", attrs)
	}
}

// Every activity reports a database failure as an error (retried by its
// policy), never as a result: here the pool points at a closed port. Each
// runs once, from a workflow whose id is a backfill child's.
func TestActivitiesReportDatabaseErrors(t *testing.T) {
	pool, err := pgxpool.New(context.Background(), "postgres://nobody:x@127.0.0.1:1/none?connect_timeout=1")
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	a := &Activities{Pool: pool, Log: slog.New(slog.DiscardHandler)}
	f := uuid.New()
	for _, name := range []string{BarrierActivity, ClassifyActivity, BatchActivity, ReadyActivity} {
		t.Run(name, func(t *testing.T) {
			var suite testsuite.WorkflowTestSuite
			env := suite.NewTestWorkflowEnvironment()
			env.SetStartWorkflowOptions(client.StartWorkflowOptions{ID: ChildID(uuid.MustParse(unitTenant), fieldindex.BackfillWorkflowType, f, 1)})
			env.SetWorkerOptions(worker.Options{BackgroundActivityContext: workerguard.ContextWithTenant(context.Background(), uuid.MustParse(unitTenant))})
			a.Register(env)
			env.RegisterWorkflowWithOptions(func(ctx workflow.Context) error {
				ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{StartToCloseTimeout: 30 * time.Second,
					RetryPolicy: &temporal.RetryPolicy{MaximumAttempts: 1}})
				var args []any
				switch name {
				case ClassifyActivity:
					args = []any{FieldRef{FieldID: f}}
				case BatchActivity:
					args = []any{Cursor{}}
				}
				return workflow.ExecuteActivity(ctx, name, args...).Get(ctx, nil)
			}, workflow.RegisterOptions{Name: "test.run"})
			env.ExecuteWorkflow("test.run")
			if err := env.GetWorkflowError(); err == nil {
				t.Fatalf("%s succeeded against a closed port", name)
			}
		})
	}
	if a.logger() != a.Log {
		t.Error("logger() ignored Log")
	}
}

// A failing barrier or flip fails the backfill (the coordinator logs it and
// moves on); a failing Classify skips the item.
func TestFailuresPropagate(t *testing.T) {
	f := uuid.New()
	for _, name := range []string{BarrierActivity, ReadyActivity} {
		env := backfillEnv(t, &stubs{failing: name}, nil, f)
		env.ExecuteWorkflow(fieldindex.BackfillWorkflowType, Input{FieldID: f})
		if err := env.GetWorkflowError(); err == nil {
			t.Errorf("a failing %s did not fail the backfill", name)
		}
	}
	s := &stubs{failing: ClassifyActivity, classify: map[uuid.UUID]Classified{f: {Backfill: &f}}}
	env := newEnv(t, s, coordinatorID())
	enqueue(env, f, 1, uuid.Must(uuid.NewV7()).String(), 0)
	env.ExecuteWorkflow(fieldindex.CoordinatorWorkflowType, start())
	if err := env.GetWorkflowError(); err != nil {
		t.Fatal(err)
	}
	if len(s.children) != 0 {
		t.Errorf("children %v after a failed Classify, want none", s.children)
	}
}

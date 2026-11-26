package backfill

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"

	"github.com/Navaladi2019/TicketingSystem/internal/fieldindex"
	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
)

// removeStubs scripts FieldIndexRemove's activities: each RemoveBatch and
// Retire call takes the next answer (true: work left), then false.
type removeStubs struct {
	mu      sync.Mutex
	batches []bool
	retires []bool
	calls   []string // activity names, in call order
	failing string
}

func (s *removeStubs) next(name string, q *[]bool, field uuid.UUID) (Pending, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls = append(s.calls, name)
	if s.failing == name {
		return Pending{}, temporal.NewNonRetryableApplicationError(name+" failed", "test.Fail", nil)
	}
	if len(*q) == 0 {
		return Pending{}, nil
	}
	more := (*q)[0]
	*q = (*q)[1:]
	if more {
		return Pending{Field: &field}, nil
	}
	return Pending{}, nil
}

func removeEnv(t *testing.T, s *removeStubs, r *Remove, field uuid.UUID) *testsuite.TestWorkflowEnvironment {
	t.Helper()
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.SetStartWorkflowOptions(client.StartWorkflowOptions{ID: ChildID(uuid.MustParse(unitTenant), fieldindex.RemoveWorkflowType, field, 2)})
	env.SetTestTimeout(time.Minute)
	env.RegisterActivityWithOptions(func(context.Context) error {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.calls = append(s.calls, BarrierActivity)
		return nil
	}, activity.RegisterOptions{Name: BarrierActivity})
	env.RegisterActivityWithOptions(func(context.Context) (Pending, error) { return s.next(RemoveBatchActivity, &s.batches, field) },
		activity.RegisterOptions{Name: RemoveBatchActivity})
	env.RegisterActivityWithOptions(func(context.Context) (Pending, error) { return s.next(RetireActivity, &s.retires, field) },
		activity.RegisterOptions{Name: RetireActivity})
	r.Register(env)
	return env
}

// Barrier first, then batches until one finds nothing more, then Retire; a
// Retire that finds side rows left goes back to the batches (D11 step 4).
func TestRemoveRunsBarrierBatchesThenRetire(t *testing.T) {
	f := uuid.New()
	s := &removeStubs{batches: []bool{true, true, false, false}, retires: []bool{true, false}}
	env := removeEnv(t, s, &Remove{}, f)
	env.ExecuteWorkflow(fieldindex.RemoveWorkflowType, FieldRef{FieldID: f})
	if err := env.GetWorkflowError(); err != nil {
		t.Fatal(err)
	}
	want := []string{BarrierActivity, RemoveBatchActivity, RemoveBatchActivity, RemoveBatchActivity, RetireActivity,
		RemoveBatchActivity, RetireActivity}
	if !slices.Equal(s.calls, want) {
		t.Fatalf("calls %v, want %v", s.calls, want)
	}
}

// After MaxBatchesPerRun steps the run continues as new with the field
// only, so a large field's history stays bounded.
func TestRemoveContinuesAsNew(t *testing.T) {
	f := uuid.New()
	s := &removeStubs{batches: []bool{true, true, true}}
	env := removeEnv(t, s, &Remove{MaxBatchesPerRun: 2}, f)
	env.ExecuteWorkflow(fieldindex.RemoveWorkflowType, FieldRef{FieldID: f})
	var can *workflow.ContinueAsNewError
	if err := env.GetWorkflowError(); !errors.As(err, &can) {
		t.Fatalf("error %v, want continue-as-new", err)
	}
	if can.WorkflowType.Name != fieldindex.RemoveWorkflowType {
		t.Errorf("continued as %s", can.WorkflowType.Name)
	}
	var in FieldRef
	if err := json.Unmarshal(can.Input.GetPayloads()[0].GetData(), &in); err != nil || in.FieldID != f {
		t.Errorf("continue-as-new input %+v (%v), want the field", in, err)
	}
}

// A workflow id that is not a removal child's, or names another field, runs
// nothing; a failed step fails the run (the coordinator logs it and moves on).
func TestRemoveRefusesAMismatchedInputAndReportsFailures(t *testing.T) {
	f := uuid.New()
	s := &removeStubs{}
	env := removeEnv(t, s, &Remove{}, f)
	env.ExecuteWorkflow(fieldindex.RemoveWorkflowType, FieldRef{FieldID: uuid.New()})
	if err := env.GetWorkflowError(); err == nil || len(s.calls) != 0 {
		t.Errorf("error %v and calls %v, want a refusal before any activity", err, s.calls)
	}
	for _, name := range []string{RemoveBatchActivity, RetireActivity} {
		s := &removeStubs{failing: name}
		env := removeEnv(t, s, &Remove{}, f)
		env.ExecuteWorkflow(fieldindex.RemoveWorkflowType, FieldRef{FieldID: f})
		if err := env.GetWorkflowError(); err == nil {
			t.Errorf("a failed %s was not reported", name)
		}
	}
}

// The coordinator starts the removal as its child under the removal's id,
// with the field as its only input, and RegisterWorkflow registers it.
func TestCoordinatorRunsTheRealRemoval(t *testing.T) {
	f := uuid.New()
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.SetStartWorkflowOptions(client.StartWorkflowOptions{ID: coordinatorID()})
	env.SetTestTimeout(time.Minute)
	RegisterWorkflow(env)
	var calls []string
	var mu sync.Mutex
	rec := func(name string) {
		mu.Lock()
		defer mu.Unlock()
		calls = append(calls, name)
	}
	env.RegisterActivityWithOptions(func(_ context.Context, in FieldRef) (Classified, error) {
		return Classified{Remove: &in.FieldID}, nil
	}, activity.RegisterOptions{Name: ClassifyActivity})
	env.RegisterActivityWithOptions(func(context.Context) error { rec(BarrierActivity); return nil },
		activity.RegisterOptions{Name: BarrierActivity})
	env.RegisterActivityWithOptions(func(ctx context.Context) (Pending, error) {
		rec(activity.GetInfo(ctx).WorkflowExecution.ID)
		return Pending{}, nil
	}, activity.RegisterOptions{Name: RemoveBatchActivity})
	env.RegisterActivityWithOptions(func(context.Context) (Pending, error) { rec(RetireActivity); return Pending{}, nil },
		activity.RegisterOptions{Name: RetireActivity})
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(fieldindex.SignalEnqueue, fieldindex.EnqueueItem{Op: fieldindex.OpRemove,
			FieldID: f.String(), RequestedVersion: 5, OutboxID: uuid.Must(uuid.NewV7()).String()})
	}, 0)
	env.ExecuteWorkflow(fieldindex.CoordinatorWorkflowType, start())
	if err := env.GetWorkflowError(); err != nil {
		t.Fatal(err)
	}
	want := []string{BarrierActivity, ChildID(uuid.MustParse(unitTenant), fieldindex.RemoveWorkflowType, f, 5), RetireActivity}
	if !slices.Equal(calls, want) {
		t.Fatalf("calls %v, want %v", calls, want)
	}
}

// The removal's activities refuse to run outside a removal child, before
// touching the database (the pool is nil).
func TestRemoveActivitiesRefuseOutsideARemovalChild(t *testing.T) {
	a := &Activities{}
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestActivityEnvironment()
	env.RegisterActivityWithOptions(a.RemoveBatch, activity.RegisterOptions{Name: RemoveBatchActivity})
	env.RegisterActivityWithOptions(a.Retire, activity.RegisterOptions{Name: RetireActivity})
	for _, name := range []string{RemoveBatchActivity, RetireActivity} {
		if _, err := env.ExecuteActivity(name); err == nil {
			t.Errorf("%s ran outside a removal child", name)
		}
	}
}

// The removal's recorded shapes pass the guard's references-only rule (CI-7).
func TestRemoveShapesAreReferencesOnly(t *testing.T) {
	f := uuid.New()
	for name, v := range map[string]any{
		"pending":   Pending{Field: &f},
		"done":      Pending{},
		"child in":  FieldRef{FieldID: f},
		"continued": FieldRef{FieldID: f},
	} {
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		if err := outbox.ValidatePayload(b); err != nil {
			t.Errorf("%s %s: %v", name, b, err)
		}
	}
}

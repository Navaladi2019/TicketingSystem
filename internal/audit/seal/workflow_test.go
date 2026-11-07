package seal_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"

	"github.com/Navaladi2019/TicketingSystem/internal/audit/seal"
	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
)

// fakeActs stands in for A1 and A2 under their registered names.
type fakeActs struct {
	a1    func(context.Context) (seal.Rescheduled, error)
	a2Err error

	mu      sync.Mutex
	a1Calls int
	a2Calls int
	infos   map[string]activity.Info
}

func (f *fakeActs) register(env *testsuite.TestWorkflowEnvironment) {
	env.RegisterActivityWithOptions(func(ctx context.Context) (seal.Rescheduled, error) {
		f.mu.Lock()
		f.a1Calls++
		f.mu.Unlock()
		return f.a1(ctx)
	}, activity.RegisterOptions{Name: seal.RescheduleActivity})
	env.RegisterActivityWithOptions(func(context.Context) error {
		f.mu.Lock()
		f.a2Calls++
		f.mu.Unlock()
		return f.a2Err
	}, activity.RegisterOptions{Name: seal.SealActivity})
	f.infos = map[string]activity.Info{}
	env.SetOnActivityStartedListener(func(info *activity.Info, _ context.Context, _ converter.EncodedValues) {
		f.mu.Lock()
		f.infos[info.ActivityType.Name] = *info
		f.mu.Unlock()
	})
}

// commandRecorder is a worker interceptor that records every child start
// and continue-as-new the workflow asks for, and what each activity is
// scheduled after.
type commandRecorder struct {
	interceptor.WorkerInterceptorBase
	mu        sync.Mutex
	children  []string
	continues int
	order     []string
	options   map[string]workflow.ActivityOptions // as scheduled, by activity type
}

func (r *commandRecorder) InterceptWorkflow(_ workflow.Context, next interceptor.WorkflowInboundInterceptor) interceptor.WorkflowInboundInterceptor {
	return &recInbound{WorkflowInboundInterceptorBase: interceptor.WorkflowInboundInterceptorBase{Next: next}, r: r}
}

type recInbound struct {
	interceptor.WorkflowInboundInterceptorBase
	r *commandRecorder
}

func (i *recInbound) Init(out interceptor.WorkflowOutboundInterceptor) error {
	return i.Next.Init(&recOutbound{WorkflowOutboundInterceptorBase: interceptor.WorkflowOutboundInterceptorBase{Next: out}, r: i.r})
}

type recOutbound struct {
	interceptor.WorkflowOutboundInterceptorBase
	r *commandRecorder
}

func (o *recOutbound) ExecuteActivity(ctx workflow.Context, activityType string, args ...any) workflow.Future {
	o.r.mu.Lock()
	o.r.order = append(o.r.order, activityType)
	if o.r.options == nil {
		o.r.options = map[string]workflow.ActivityOptions{}
	}
	o.r.options[activityType] = workflow.GetActivityOptions(ctx)
	o.r.mu.Unlock()
	return o.Next.ExecuteActivity(ctx, activityType, args...)
}

func (o *recOutbound) ExecuteChildWorkflow(ctx workflow.Context, childWorkflowType string, args ...any) workflow.ChildWorkflowFuture {
	o.r.mu.Lock()
	o.r.children = append(o.r.children, childWorkflowType)
	o.r.mu.Unlock()
	return o.Next.ExecuteChildWorkflow(ctx, childWorkflowType, args...)
}

func (o *recOutbound) NewContinueAsNewError(ctx workflow.Context, wfn any, args ...any) error {
	o.r.mu.Lock()
	o.r.continues++
	o.r.mu.Unlock()
	return o.Next.NewContinueAsNewError(ctx, wfn, args...)
}

// runWorkflow runs audit.seal once on the test environment with f and rec.
func runWorkflow(t *testing.T, f *fakeActs, rec *commandRecorder) error {
	t.Helper()
	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	env.SetWorkerOptions(worker.Options{Interceptors: []interceptor.WorkerInterceptor{rec}})
	tenant, o := uuid.New(), uuid.Must(uuid.NewV7())
	env.SetStartWorkflowOptions(client.StartWorkflowOptions{ID: "t:" + tenant.String() + ":audit.seal:" + o.String()})
	seal.RegisterWorkflow(env)
	f.register(env)
	env.ExecuteWorkflow(seal.WorkflowType, outbox.Delivery{OutboxID: o, KindVersion: 1, Payload: []byte(`{}`)})
	if !env.IsWorkflowCompleted() {
		t.Fatal("the workflow did not complete")
	}
	return env.GetWorkflowError()
}

func nextRun() (seal.Rescheduled, error) {
	o := uuid.Must(uuid.NewV7())
	return seal.Rescheduled{NextOutboxID: &o}, nil
}

// The scenarios every structural check runs over: a live lineage, a stale
// one (or a retried A1 after its commit), a poison A2 and a failed A1.
func scenarios() map[string]*fakeActs {
	poison := temporal.NewNonRetryableApplicationError("seal: entry cannot be canonicalised", seal.ErrTypeUnsealable, nil)
	noRun := temporal.NewNonRetryableApplicationError("seal: no run", seal.ErrTypeNoRun, nil)
	return map[string]*fakeActs{
		"next run":         {a1: func(context.Context) (seal.Rescheduled, error) { return nextRun() }},
		"stale":            {a1: func(context.Context) (seal.Rescheduled, error) { return seal.Rescheduled{}, nil }},
		"poison A2":        {a1: func(context.Context) (seal.Rescheduled, error) { return nextRun() }, a2Err: poison},
		"A1 no run":        {a1: func(context.Context) (seal.Rescheduled, error) { return seal.Rescheduled{}, noRun }},
		"stale, poison A2": {a1: func(context.Context) (seal.Rescheduled, error) { return seal.Rescheduled{}, nil }, a2Err: poison},
	}
}

// TestWorkflowStartsNoChildAndNeverContinuesAsNew (TICK-438 / TICK-572
// review, binding): the sealer asks for no StartChildWorkflowExecution and no
// ContinueAsNewWorkflowExecution on any path; it reschedules only through
// A1's outbox row. The Authorizer sees only the workflow id, so this test is
// where the rule is enforced.
func TestWorkflowStartsNoChildAndNeverContinuesAsNew(t *testing.T) {
	for name, f := range scenarios() {
		t.Run(name, func(t *testing.T) {
			rec := &commandRecorder{}
			err := runWorkflow(t, f, rec)
			var can *workflow.ContinueAsNewError
			if errors.As(err, &can) {
				t.Fatalf("the workflow continued as new: %v", err)
			}
			if len(rec.children) != 0 || rec.continues != 0 {
				t.Errorf("children %v, continue-as-new %d; want none", rec.children, rec.continues)
			}
		})
	}
}

// TestWorkflowRunsA2WhateverA1Returned (TICK-440 review, binding): A2 runs
// after A1 whether A1 returned o′ or nothing (a stale lineage, or a retry of
// an A1 that had committed); only an A1 failure skips it. A1 is always
// scheduled first.
func TestWorkflowRunsA2WhateverA1Returned(t *testing.T) {
	for name, f := range scenarios() {
		t.Run(name, func(t *testing.T) {
			rec := &commandRecorder{}
			err := runWorkflow(t, f, rec)
			a1Failed := name == "A1 no run"
			wantA2 := 1
			if a1Failed {
				wantA2 = 0
			}
			if f.a1Calls != 1 || f.a2Calls != wantA2 {
				t.Errorf("A1 ran %d times, A2 %d; want 1 and %d", f.a1Calls, f.a2Calls, wantA2)
			}
			if len(rec.order) == 0 || rec.order[0] != seal.RescheduleActivity {
				t.Errorf("activities scheduled %v, want A1 first", rec.order)
			}
			wantErr := a1Failed || f.a2Err != nil
			if (err != nil) != wantErr {
				t.Errorf("workflow error %v, want error %v", err, wantErr)
			}
		})
	}
}

// A retryable A1 failure after its commit is retried; the retry returns no
// o′ (the head already moved on), and A2 still runs.
func TestWorkflowRunsA2AfterRetriedA1(t *testing.T) {
	f := &fakeActs{a1: func(ctx context.Context) (seal.Rescheduled, error) {
		if activity.GetInfo(ctx).Attempt == 1 {
			return seal.Rescheduled{}, errors.New("worker lost after commit")
		}
		return seal.Rescheduled{}, nil
	}}
	if err := runWorkflow(t, f, &commandRecorder{}); err != nil {
		t.Fatal(err)
	}
	if f.a1Calls != 2 || f.a2Calls != 1 {
		t.Errorf("A1 ran %d times, A2 %d; want 2 and 1", f.a1Calls, f.a2Calls)
	}
}

// The activity bounds: A2 has a heartbeat timeout well under its
// start-to-close, so a stuck batch is noticed after one batch's worth of
// time, not the whole call (TICK-440 review, "bounded A2"). Both have an
// explicit retry policy as scheduled: A1's has no attempt cap, so an outage
// before A1 commits never ends the lineage (TICK-587), and A2's is capped at
// A2MaxAttempts. Both cap the backoff at a minute. The options are asserted
// as scheduled, not by running past ten A1 failures: the SDK test environment
// turns an uncapped policy into ten attempts (defaultMaximumAttemptsForUnitTest).
func TestWorkflowActivityOptions(t *testing.T) {
	f := scenarios()["next run"]
	rec := &commandRecorder{}
	if err := runWorkflow(t, f, rec); err != nil {
		t.Fatal(err)
	}
	p1, p2 := rec.options[seal.RescheduleActivity].RetryPolicy, rec.options[seal.SealActivity].RetryPolicy
	if p1 == nil || p1.MaximumAttempts != 0 || p1.MaximumInterval != time.Minute {
		t.Errorf("A1 scheduled with retry policy %+v, want no attempt cap and a 1 min backoff cap", p1)
	}
	if p2 == nil || p2.MaximumAttempts != seal.A2MaxAttempts || seal.A2MaxAttempts <= 0 || p2.MaximumInterval != time.Minute {
		t.Errorf("A2 scheduled with retry policy %+v, want %d attempts and a 1 min backoff cap", p2, seal.A2MaxAttempts)
	}
	a1, a2 := f.infos[seal.RescheduleActivity], f.infos[seal.SealActivity]
	if a1.StartToCloseTimeout != seal.A1StartToClose {
		t.Errorf("A1 start-to-close %v, want %v", a1.StartToCloseTimeout, seal.A1StartToClose)
	}
	if a2.StartToCloseTimeout != seal.A2StartToClose || a2.HeartbeatTimeout != seal.A2HeartbeatTimeout {
		t.Errorf("A2 start-to-close %v heartbeat %v, want %v and %v",
			a2.StartToCloseTimeout, a2.HeartbeatTimeout, seal.A2StartToClose, seal.A2HeartbeatTimeout)
	}
	if seal.A2HeartbeatTimeout <= 0 || seal.A2HeartbeatTimeout >= seal.A2StartToClose {
		t.Error("A2's heartbeat timeout does not bound a batch")
	}
}

package schemabackfill

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"

	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
)

// The workflow's branches, on the SDK test environment with stubbed
// activities: every step that reports a stop ends the run cleanly, a
// non-retryable ErrTypeNoJob ends it with that error (it never loops), and a
// plain activity error is retried.

// stubs are the six activities' stand-ins and how often each ran.
type stubs struct {
	admit   []AdmitResult // one per call; the last one repeats
	barrier error
	fill    FillResult
	fillErr error
	pages   []PageResult
	pageErr []error // per call, before pages; nil: no error
	end     EndResult
	endErr  error
	r6Err   error
	calls   map[string]int
	elapsed time.Duration // test time the run took
}

func noJobErr() error {
	return temporal.NewNonRetryableApplicationError("schema backfill: stub", ErrTypeNoJob, nil)
}

func (s *stubs) register(env *testsuite.TestWorkflowEnvironment) {
	s.calls = map[string]int{}
	reg := func(name string, fn any) {
		env.RegisterActivityWithOptions(fn, activity.RegisterOptions{Name: name})
	}
	reg(AdmitActivity, func(context.Context) (AdmitResult, error) {
		n := s.calls[AdmitActivity]
		s.calls[AdmitActivity]++
		if n >= len(s.admit) {
			n = len(s.admit) - 1
		}
		if n < 0 {
			return AdmitResult{}, noJobErr()
		}
		return s.admit[n], nil
	})
	reg(BarrierActivity, func(context.Context) error { s.calls[BarrierActivity]++; return s.barrier })
	reg(FillActivity, func(context.Context) (FillResult, error) { s.calls[FillActivity]++; return s.fill, s.fillErr })
	reg(PageActivity, func(_ context.Context, _ Cursor) (PageResult, error) {
		n := s.calls[PageActivity]
		s.calls[PageActivity]++
		if n < len(s.pageErr) && s.pageErr[n] != nil {
			return PageResult{}, s.pageErr[n]
		}
		k := n - len(s.pageErr)
		if k < 0 || k >= len(s.pages) {
			return PageResult{}, nil
		}
		return s.pages[k], nil
	})
	reg(EndActivity, func(context.Context) (EndResult, error) { s.calls[EndActivity]++; return s.end, s.endErr })
	reg(R6Activity, func(context.Context) error { s.calls[R6Activity]++; return s.r6Err })
}

// runStubbed runs the workflow for a job with s and the given input
// payload, records the test time it took in s.elapsed, and returns the run's
// error.
func runStubbed(t *testing.T, s *stubs, w *Workflow, id string, payload json.RawMessage) error {
	t.Helper()
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.SetStartWorkflowOptions(client.StartWorkflowOptions{ID: id})
	s.register(env)
	if w == nil {
		w = &Workflow{}
	}
	w.Register(env)
	start := env.Now()
	env.ExecuteWorkflow(WorkflowType, outbox.Delivery{OutboxID: uuid.New(), KindVersion: 1, Payload: payload})
	s.elapsed = env.Now().Sub(start)
	if !env.IsWorkflowCompleted() {
		t.Fatal("workflow did not complete")
	}
	return env.GetWorkflowError()
}

func stubJob() (Job, string, json.RawMessage) {
	j := Job{Tenant: uuid.New(), Definition: uuid.New(), Version: 2}
	return j, WorkflowID(j), json.RawMessage(`{"entity_definition_id":"` + j.Definition.String() + `"}`)
}

func isNoJob(err error) bool {
	var app *temporal.ApplicationError
	return errors.As(err, &app) && app.Type() == ErrTypeNoJob
}

func TestWorkflowRefusesAnIDOrPayloadThatIsNotAJob(t *testing.T) {
	j, id, _ := stubJob()
	s := &stubs{admit: []AdmitResult{{Admitted: &j.Definition}}}
	if err := runStubbed(t, s, nil, "t:"+j.Tenant.String()+":schema.backfill:x:2", nil); !isNoJob(err) || s.calls[AdmitActivity] != 0 {
		t.Errorf("bad id: %v after %d admissions; want ErrTypeNoJob before any activity", err, s.calls[AdmitActivity])
	}
	s = &stubs{admit: []AdmitResult{{Admitted: &j.Definition}}}
	other := json.RawMessage(`{"entity_definition_id":"` + uuid.NewString() + `"}`)
	if err := runStubbed(t, s, nil, id, other); !isNoJob(err) || s.calls[AdmitActivity] != 0 {
		t.Errorf("payload of another definition: %v after %d admissions; want ErrTypeNoJob before any activity", err, s.calls[AdmitActivity])
	}
}

// Admission: a missing row stops the run with ErrTypeNoJob; a busy tenant
// waits 10 s, 20 s, 40 s, 60 s, ... between asks; an ended job completes.
func TestWorkflowAdmission(t *testing.T) {
	j, id, payload := stubJob()
	s := &stubs{}
	if err := runStubbed(t, s, nil, id, payload); !isNoJob(err) || s.calls[AdmitActivity] != 1 || s.calls[BarrierActivity] != 0 {
		t.Errorf("no row: %v, %v; want ErrTypeNoJob after one admission", err, s.calls)
	}

	busy := AdmitResult{}
	s = &stubs{admit: []AdmitResult{busy, busy, busy, busy, busy, {Admitted: &j.Definition}}, fill: FillResult{Ended: &j.Definition}}
	err := runStubbed(t, s, nil, id, payload)
	if err != nil || s.calls[AdmitActivity] != 6 || s.calls[FillActivity] != 1 || s.calls[PageActivity] != 0 {
		t.Fatalf("busy tenant: %v, %v; want six admissions, a fill that ends, no page", err, s.calls)
	}
	// Five waits, backing off to the cap: 10 + 20 + 40 + 60 + 60 s.
	if want := 190 * time.Second; s.elapsed < want || s.elapsed > want+time.Minute {
		t.Errorf("busy tenant waited %v, want about %v", s.elapsed, want)
	}

	s = &stubs{admit: []AdmitResult{{Ended: &j.Definition}}}
	if err := runStubbed(t, s, nil, id, payload); err != nil || s.calls[BarrierActivity] != 0 {
		t.Errorf("ended at admission: %v, %v; want a clean end with no barrier", err, s.calls)
	}
}

// Every later step's ErrTypeNoJob ends the run with it, and nothing after
// that step runs.
func TestWorkflowStopsOnANonRetryableStep(t *testing.T) {
	j, id, payload := stubJob()
	admitted := []AdmitResult{{Admitted: &j.Definition}}
	for name, s := range map[string]*stubs{
		BarrierActivity: {admit: admitted, barrier: noJobErr()},
		FillActivity:    {admit: admitted, fillErr: noJobErr()},
		PageActivity:    {admit: admitted, pageErr: []error{noJobErr()}},
		EndActivity:     {admit: admitted, endErr: noJobErr()},
		R6Activity:      {admit: admitted, end: EndResult{FailedRecords: &j.Definition}, r6Err: noJobErr()},
	} {
		t.Run(name, func(t *testing.T) {
			err := runStubbed(t, s, &Workflow{R6After: time.Hour}, id, payload)
			if !isNoJob(err) || s.calls[name] != 1 {
				t.Errorf("%s: %v, %v; want ErrTypeNoJob after one call", name, err, s.calls)
			}
		})
	}
}

// A page's plain error is retried; a page that finds the job superseded
// ends the run before End; an empty queue goes to End, and a done job ends
// with no R6 check.
func TestWorkflowPages(t *testing.T) {
	j, id, payload := stubJob()
	admitted := []AdmitResult{{Admitted: &j.Definition}}
	a, b := uuid.New(), uuid.New()

	s := &stubs{admit: admitted, pageErr: []error{errors.New("transient")}, pages: []PageResult{{Next: &a}, {Next: &b}, {}}}
	if err := runStubbed(t, s, nil, id, payload); err != nil || s.calls[PageActivity] != 4 || s.calls[EndActivity] != 1 || s.calls[R6Activity] != 0 {
		t.Errorf("retried page: %v, %v; want 1 failed + 3 pages, End, no R6", err, s.calls)
	}

	s = &stubs{admit: admitted, pages: []PageResult{{Next: &a}, {Ended: &j.Definition}}}
	if err := runStubbed(t, s, nil, id, payload); err != nil || s.calls[EndActivity] != 0 {
		t.Errorf("superseded mid-run: %v, %v; want a clean end with no End", err, s.calls)
	}
}

// failed_records: the run waits R6After, then runs the R6 check once.
func TestWorkflowR6AfterFailedRecords(t *testing.T) {
	j, id, payload := stubJob()
	s := &stubs{admit: []AdmitResult{{Admitted: &j.Definition}}, end: EndResult{FailedRecords: &j.Definition}}
	err := runStubbed(t, s, &Workflow{R6After: 3 * time.Hour}, id, payload)
	if err != nil || s.calls[R6Activity] != 1 {
		t.Fatalf("failed_records: %v, %v; want one R6 check", err, s.calls)
	}
	if s.elapsed < 3*time.Hour {
		t.Errorf("the R6 check ran after %v, want after the 3 h window", s.elapsed)
	}

	// The default window is a week.
	if (&Workflow{}).r6After() != DefaultR6After || (&Workflow{}).admitWait() != DefaultAdmitWait ||
		(&Workflow{}).admitWaitMax() != DefaultAdmitWaitMax {
		t.Error("zero Workflow does not take the defaults")
	}
}

// RegisterWorkflow, the form the handbook build parses, registers the
// workflow with the defaults under WorkflowType.
func TestRegisterWorkflowWithDefaults(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	j, id, payload := stubJob()
	s := &stubs{admit: []AdmitResult{{Ended: &j.Definition}}}
	s.register(env)
	RegisterWorkflow(env)
	env.SetStartWorkflowOptions(client.StartWorkflowOptions{ID: id})
	env.ExecuteWorkflow(WorkflowType, outbox.Delivery{OutboxID: uuid.New(), KindVersion: 1, Payload: payload})
	if !env.IsWorkflowCompleted() || env.GetWorkflowError() != nil || s.calls[AdmitActivity] != 1 {
		t.Errorf("default registration: completed %v, %v, %v", env.IsWorkflowCompleted(), env.GetWorkflowError(), s.calls)
	}
}

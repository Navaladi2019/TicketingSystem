package interp_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/google/uuid"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/worker"

	"github.com/Navaladi2019/TicketingSystem/internal/workerguard"
	"github.com/Navaladi2019/TicketingSystem/internal/workflow/interp"
)

// movingRecords returns a version one higher on every read, as a record that
// a write changes between each read and the wait.
type movingRecords struct {
	mu      sync.Mutex
	version int64
}

func (r *movingRecords) Read(context.Context, uuid.UUID, uuid.UUID) (map[string]any, int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.version++
	return map[string]any{"priority": "high"}, r.version, nil
}

// movedWaits refuses the first moves calls with ErrRecordMoved, as the
// tracker does when the record is no longer at the version the walk read.
type movedWaits struct {
	waits
	moves int
}

func (w *movedWaits) SetRecordChangedWait(ctx context.Context, tenant, record uuid.UUID, signal *uuid.UUID, version int64) error {
	_ = w.waits.SetRecordChangedWait(ctx, tenant, record, signal, version)
	w.mu.Lock()
	defer w.mu.Unlock()
	if len(w.calls) <= w.moves {
		return workerguard.ErrRecordMoved
	}
	return nil
}

func runActivity(t *testing.T, a *interp.Activities, name string, ref interp.StepRef) error {
	t.Helper()
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestActivityEnvironment()
	env.SetWorkerOptions(worker.Options{BackgroundActivityContext: tenantCtx()})
	env.RegisterActivityWithOptions(a.NextStep, activity.RegisterOptions{Name: interp.NextStepActivity})
	env.RegisterActivityWithOptions(a.ApplyStep, activity.RegisterOptions{Name: interp.ApplyStepActivity})
	_, err := env.ExecuteActivity(name, ref)
	return err
}

// NextStep re-walks inside one attempt when the record moved between the
// walk's read and the wait (TICK-784): up to MaxRecordMovedRewalks moves the
// attempt succeeds, with the wait set on the version of the last walk; past
// that it fails the attempt retryably.
func TestNextStepRewalksWhenTheRecordMoved(t *testing.T) {
	rc, f, _ := recordChangedVersion(t)
	for _, moves := range []int{0, 1, interp.MaxRecordMovedRewalks} {
		recs := &movingRecords{}
		w := &movedWaits{moves: moves}
		a := &interp.Activities{Definitions: versionStore{rc}, Records: recs, Writer: noWrites{t}, Waits: w}
		ref := interp.StepRef{DefinitionVersionID: uuid.New(), RecordID: uuid.New(), Step: f.id(t, "await_approval"), Cause: uuid.New()}
		if err := runActivity(t, a, interp.NextStepActivity, ref); err != nil {
			t.Fatalf("%d moves: %v, want the attempt to succeed", moves, err)
		}
		if len(w.calls) != moves+1 {
			t.Errorf("%d moves: %d wait calls, want %d", moves, len(w.calls), moves+1)
		}
		if last := w.versions[len(w.versions)-1]; last != recs.version {
			t.Errorf("%d moves: wait set on version %d, want the last read %d", moves, last, recs.version)
		}
	}

	w := &movedWaits{moves: interp.MaxRecordMovedRewalks + 1}
	a := &interp.Activities{Definitions: versionStore{rc}, Records: &movingRecords{}, Writer: noWrites{t}, Waits: w}
	ref := interp.StepRef{DefinitionVersionID: uuid.New(), RecordID: uuid.New(), Step: f.id(t, "await_approval"), Cause: uuid.New()}
	err := runActivity(t, a, interp.NextStepActivity, ref)
	if err == nil {
		t.Fatal("NextStep succeeded with the record still moving")
	}
	var app *temporal.ApplicationError
	if errors.As(err, &app) && app.NonRetryable() {
		t.Errorf("a record still moving failed the run: %v, want a retryable failure", err)
	}
	if len(w.calls) != interp.MaxRecordMovedRewalks+1 {
		t.Errorf("%d wait calls, want %d (bounded)", len(w.calls), interp.MaxRecordMovedRewalks+1)
	}
}

// Another wait failure is not re-walked.
func TestNextStepDoesNotRewalkOtherWaitFailures(t *testing.T) {
	rc, f, _ := recordChangedVersion(t)
	w := &waits{err: errors.New("database down")}
	a := &interp.Activities{Definitions: versionStore{rc}, Records: &movingRecords{}, Writer: noWrites{t}, Waits: w}
	ref := interp.StepRef{DefinitionVersionID: uuid.New(), RecordID: uuid.New(), Step: f.id(t, "await_approval")}
	if err := runActivity(t, a, interp.NextStepActivity, ref); err == nil {
		t.Fatal("NextStep succeeded")
	}
	if len(w.calls) != 1 {
		t.Errorf("%d wait calls, want 1", len(w.calls))
	}
}

// staleWriter refuses every cause but accept as interp.ErrStaleCause.
type staleWriter struct {
	mu     sync.Mutex
	accept uuid.UUID
	causes []uuid.UUID
}

func (w *staleWriter) Write(_ context.Context, _, _, _, cause uuid.UUID, _ string, _ map[string]any) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.causes = append(w.causes, cause)
	if cause != w.accept {
		return interp.ErrStaleCause
	}
	return nil
}

// causes returns ids in turn, then the last one; or err.
type causes struct {
	mu  sync.Mutex
	ids []uuid.UUID
	err error
	n   int
}

func (c *causes) NewestCause(context.Context, uuid.UUID) (uuid.UUID, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.n++
	if c.err != nil {
		return uuid.Nil, c.err
	}
	id := c.ids[0]
	if len(c.ids) > 1 {
		c.ids = c.ids[1:]
	}
	return id, nil
}

// ApplyStep retries a write refused as a stale cause with the run's newest
// cause, as a reference the writer resolves again (TICK-784).
func TestApplyStepRetriesAStaleCauseWithTheNewest(t *testing.T) {
	v, f := fixtureVersion(t)
	stale, newest := uuid.New(), uuid.New()
	ref := interp.StepRef{DefinitionVersionID: uuid.New(), RecordID: uuid.New(), Step: f.id(t, "mark_approved"), Cause: stale}

	t.Run("newest accepted", func(t *testing.T) {
		w := &staleWriter{accept: newest}
		a := &interp.Activities{Definitions: versionStore{v}, Records: staticRecords{}, Writer: w, Causes: &causes{ids: []uuid.UUID{newest}}}
		if err := runActivity(t, a, interp.ApplyStepActivity, ref); err != nil {
			t.Fatal(err)
		}
		if len(w.causes) != 2 || w.causes[0] != stale || w.causes[1] != newest {
			t.Errorf("writer got causes %v, want [stale newest]", w.causes)
		}
	})

	forbidden := func(t *testing.T, err error) {
		t.Helper()
		var app *temporal.ApplicationError
		if !errors.As(err, &app) || !app.NonRetryable() || app.Type() != workerguard.ErrTypeWorkflowRejected {
			t.Errorf("%v, want the forbidden Rejection", err)
		}
	}
	t.Run("no cause source", func(t *testing.T) {
		w := &staleWriter{accept: newest}
		a := &interp.Activities{Definitions: versionStore{v}, Records: staticRecords{}, Writer: w}
		forbidden(t, runActivity(t, a, interp.ApplyStepActivity, ref))
		if len(w.causes) != 1 {
			t.Errorf("%d writes, want 1", len(w.causes))
		}
	})
	t.Run("no newer cause", func(t *testing.T) {
		w := &staleWriter{accept: newest}
		a := &interp.Activities{Definitions: versionStore{v}, Records: staticRecords{}, Writer: w, Causes: &causes{ids: []uuid.UUID{stale}}}
		forbidden(t, runActivity(t, a, interp.ApplyStepActivity, ref))
	})
	for name, err := range map[string]error{"not this run's": workerguard.ErrForeignCause, "no row": workerguard.ErrCauseNotFound} {
		t.Run(name, func(t *testing.T) {
			w := &staleWriter{accept: newest}
			a := &interp.Activities{Definitions: versionStore{v}, Records: staticRecords{}, Writer: w, Causes: &causes{err: err}}
			forbidden(t, runActivity(t, a, interp.ApplyStepActivity, ref))
		})
	}
	t.Run("lookup fails", func(t *testing.T) {
		w := &staleWriter{accept: newest}
		a := &interp.Activities{Definitions: versionStore{v}, Records: staticRecords{}, Writer: w, Causes: &causes{err: errors.New("database down")}}
		err := runActivity(t, a, interp.ApplyStepActivity, ref)
		var app *temporal.ApplicationError
		if err == nil || (errors.As(err, &app) && app.NonRetryable()) {
			t.Errorf("%v, want a retryable failure", err)
		}
	})
	t.Run("newest keeps moving", func(t *testing.T) {
		ids := make([]uuid.UUID, interp.MaxStaleCauseRetries+2)
		for i := range ids {
			ids[i] = uuid.New()
		}
		w := &staleWriter{accept: uuid.New()}
		c := &causes{ids: ids}
		a := &interp.Activities{Definitions: versionStore{v}, Records: staticRecords{}, Writer: w, Causes: c}
		err := runActivity(t, a, interp.ApplyStepActivity, ref)
		var app *temporal.ApplicationError
		if err == nil || (errors.As(err, &app) && app.NonRetryable()) {
			t.Errorf("%v, want a retryable failure", err)
		}
		if len(w.causes) != interp.MaxStaleCauseRetries+1 {
			t.Errorf("%d writes, want %d (bounded)", len(w.causes), interp.MaxStaleCauseRetries+1)
		}
	})
}

// interp.ErrStaleCause is an ErrForbidden, so any caller that maps
// ErrForbidden still refuses it.
func TestErrStaleCauseIsForbidden(t *testing.T) {
	if !errors.Is(interp.ErrStaleCause, interp.ErrForbidden) {
		t.Fatal("ErrStaleCause does not wrap ErrForbidden")
	}
}

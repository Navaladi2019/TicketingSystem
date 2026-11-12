package interp_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/worker"

	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
	"github.com/Navaladi2019/TicketingSystem/internal/workflow/def"
	"github.com/Navaladi2019/TicketingSystem/internal/workflow/interp"
)

// The interpreter listens on the one signal name the outbox enqueues an
// interpreter signal under (D-3-12, D-3-29).
func TestSignalNameIsTheOutboxInterpreterSignalKind(t *testing.T) {
	if interp.SignalName != outbox.InterpreterSignalKind || interp.WorkflowType != outbox.InterpreterKind {
		t.Fatalf("interp (%s, %s), outbox (%s, %s): they must agree", interp.WorkflowType, interp.SignalName,
			outbox.InterpreterKind, outbox.InterpreterSignalKind)
	}
}

// waits records every SetRecordChangedWait call.
type waits struct {
	mu    sync.Mutex
	calls []*uuid.UUID
	err   error
}

func (w *waits) SetRecordChangedWait(_ context.Context, _ uuid.UUID, signal *uuid.UUID) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.calls = append(w.calls, signal)
	return w.err
}

// recordChangedVersion is fixture-v1 with await_approval waiting on
// record_changed instead of approve.
func recordChangedVersion(t *testing.T) (*interp.Version, fixture, uuid.UUID) {
	t.Helper()
	v, f := fixtureVersion(t)
	mutStep(t, v, "await_approval").Name = interp.RecordChangedSignal
	sig := v.StepIDs[def.SignalKey("approve")]
	delete(v.StepIDs, def.SignalKey("approve"))
	v.StepIDs[def.SignalKey(interp.RecordChangedSignal)] = sig
	return v, f, sig
}

// NextStep sets record_changed_signal_id when it returns a wait_signal on
// record_changed, and clears it for every other Directive, a wait on
// another signal included (D-3-29).
func TestNextStepRecordsTheRecordChangedWait(t *testing.T) {
	rc, f, rcSignal := recordChangedVersion(t)
	other, _ := fixtureVersion(t)
	cases := []struct {
		name string
		v    *interp.Version
		step string
		want *uuid.UUID
	}{
		{"wait on record_changed", rc, "await_approval", &rcSignal},
		{"wait on another signal", other, "await_approval", nil},
		{"timer", rc, "cool_off", nil},
		{"update_record", rc, "mark_approved", nil},
		{"end", rc, "done", nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w := &waits{}
			a := &interp.Activities{Definitions: versionStore{c.v}, Records: staticRecords{"priority": "high"}, Writer: noWrites{t}, Waits: w}
			var suite testsuite.WorkflowTestSuite
			env := suite.NewTestActivityEnvironment()
			env.SetWorkerOptions(worker.Options{BackgroundActivityContext: tenantCtx()})
			env.RegisterActivityWithOptions(a.NextStep, activity.RegisterOptions{Name: interp.NextStepActivity})
			ref := interp.StepRef{DefinitionVersionID: uuid.New(), RecordID: uuid.New(), Step: f.id(t, c.step), Cause: uuid.New()}
			if _, err := env.ExecuteActivity(interp.NextStepActivity, ref); err != nil {
				t.Fatal(err)
			}
			if len(w.calls) != 1 {
				t.Fatalf("%d SetRecordChangedWait calls, want 1", len(w.calls))
			}
			got := w.calls[0]
			if (got == nil) != (c.want == nil) || (got != nil && *got != *c.want) {
				t.Errorf("wait %v, want %v", got, c.want)
			}
		})
	}

	// A failure to record the wait fails the attempt: the run must not wait
	// unseen by the sink.
	w := &waits{err: errors.New("database down")}
	a := &interp.Activities{Definitions: versionStore{rc}, Records: staticRecords{"priority": "high"}, Writer: noWrites{t}, Waits: w}
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestActivityEnvironment()
	env.SetWorkerOptions(worker.Options{BackgroundActivityContext: tenantCtx()})
	env.RegisterActivityWithOptions(a.NextStep, activity.RegisterOptions{Name: interp.NextStepActivity})
	ref := interp.StepRef{DefinitionVersionID: uuid.New(), RecordID: uuid.New(), Step: f.id(t, "await_approval")}
	if _, err := env.ExecuteActivity(interp.NextStepActivity, ref); err == nil {
		t.Fatal("NextStep succeeded when the wait could not be recorded")
	}
}

// StepRef.Cause is the start delivery's outbox id until a signal resumes the
// run, then that signal's outbox id; a dropped or duplicate delivery does not
// move it (D-3-29).
func TestStepRefCauseFollowsTheResumingDelivery(t *testing.T) {
	h, f, version, record := v1(t, "high")
	sig := f.id(t, def.SignalKey("approve"))
	start := startInput(t, version, record)
	resume := uuid.New()
	h.signalPayloadAt(time.Hour, uuid.New(), `{"signal":"`+uuid.NewString()+`"}`) // not the awaited one
	h.signalAt(2*time.Hour, resume, sig)
	h.env.ExecuteWorkflow(interp.WorkflowType, start)
	h.wantCompleted()
	refs := h.nextRefs()
	if len(refs) < 3 {
		t.Fatalf("%d NextStep calls, want at least 3", len(refs))
	}
	for i, r := range refs {
		want := start.OutboxID
		if r.Signal != nil || i >= 3 {
			want = resume
		}
		if r.Cause != want {
			t.Errorf("NextStep %d cause %s, want %s", i, r.Cause, want)
		}
	}
}

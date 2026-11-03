package interp_test

import (
	"encoding/json"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.temporal.io/sdk/temporal"

	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
	"github.com/Navaladi2019/TicketingSystem/internal/workerguard"
	"github.com/Navaladi2019/TicketingSystem/internal/workflow/def"
	"github.com/Navaladi2019/TicketingSystem/internal/workflow/interp"
)

// TestRunRefsRefused: carried refs that do not decode strictly, or whose
// pending arrays differ in length, fail the run with the fixed rejection
// before any activity runs.
func TestRunRefsRefused(t *testing.T) {
	for name, refs := range map[string]string{
		"unknown field":     `{"step":"` + uuid.Nil.String() + `","definition_id":"` + uuid.NewString() + `"}`,
		"pending mismatch":  `{"step":"` + uuid.Nil.String() + `","pending_outbox":["` + uuid.NewString() + `"]}`,
		"pending mismatch2": `{"step":"` + uuid.Nil.String() + `","pending_signal":["` + uuid.NewString() + `"]}`,
	} {
		t.Run(name, func(t *testing.T) {
			h := newHarness(t, tenantA)
			in := workerguard.RunInput{Delivery: startInput(t, uuid.New(), uuid.New())}
			in.Refs = json.RawMessage(refs)
			h.env.ExecuteWorkflow(interp.WorkflowType, in)
			wantRejected(t, h.env.GetWorkflowError(), workerguard.RejectInput)
			if r := h.nextRefs(); len(r) != 0 {
				t.Errorf("NextStep ran %+v, want never", r)
			}
		})
	}
}

// signalPayloadAt delivers a raw signal payload with outboxID after d.
func (h *harness) signalPayloadAt(d time.Duration, outboxID uuid.UUID, payload string) {
	del := outbox.Delivery{OutboxID: outboxID, KindVersion: 1, Payload: json.RawMessage(payload)}
	h.env.RegisterDelayedCallback(func() { h.env.SignalWorkflow(interp.SignalName, del) }, d)
}

// TestSignalWithoutSignalIDDropped: a delivery whose payload names no signal
// id (nil) is dropped, even delivered twice; it neither ends the wait nor
// uses up its outbox id, and the wait ends at the real signal.
func TestSignalWithoutSignalIDDropped(t *testing.T) {
	h, f, version, record := v1(t, "high")
	bad := uuid.New()
	h.signalPayloadAt(time.Hour, bad, `{"signal":"`+uuid.Nil.String()+`"}`)
	h.signalPayloadAt(time.Hour+time.Minute, bad, `{"signal":"`+uuid.Nil.String()+`"}`)
	h.signalAt(2*time.Hour, uuid.New(), f.id(t, def.SignalKey("approve")))
	h.start(version, record)
	h.wantCompleted()
	if h.elapsed() != 2*time.Hour {
		t.Errorf("elapsed %v, want 2h: the nil-signal deliveries must not end the wait", h.elapsed())
	}
	if c := h.calls(); len(c) != 1 || c[0].Step != f.id(t, "mark_approved") {
		t.Errorf("writes %+v, want mark_approved once", c)
	}
}

// TestCancelDuringWait: cancelling the run during a wait_signal wait ends it
// with the cancellation, without a write, a timeout path or a close.
func TestCancelDuringWait(t *testing.T) {
	h, f, version, record := v1(t, "high")
	h.env.RegisterDelayedCallback(h.env.CancelWorkflow, time.Hour)
	h.start(version, record)
	err := h.env.GetWorkflowError()
	var canceled *temporal.CanceledError
	if !errors.As(err, &canceled) {
		t.Fatalf("workflow error %v, want a CanceledError", err)
	}
	if c := h.calls(); len(c) != 0 {
		t.Errorf("writes %+v, want none", c)
	}
	if n := h.closeCount(); n != 0 {
		t.Errorf("CloseWorkflowRef ran %d times, want 0", n)
	}
	refs := h.nextRefs()
	for _, r := range refs {
		if r.Step == f.id(t, "done") {
			t.Errorf("NextStep %+v ran for the timeout edge after the cancel", r)
		}
	}
	h.wantNoDenies()
}

// TestSameInputSameCommands: the interpreter run twice on the same input and
// the same signal deliveries (one redelivered) issues the same commands: the
// same NextStep refs, timers and writes, in the same order. Determinism in
// the recorded-history sense is TestReplayRecordedHistories; this pins it for
// the redelivery path too.
func TestSameInputSameCommands(t *testing.T) {
	f := fixtureV1(t)
	version, record := uuid.New(), uuid.New()
	sig, outboxID := f.id(t, def.SignalKey("approve")), uuid.New()
	type trace struct {
		refs   []interp.StepRef
		timers []time.Duration
		steps  []uuid.UUID
	}
	runOnce := func() trace {
		h := newHarness(t, tenantA)
		h.install(tenantA, version, f)
		h.putRecord(tenantA, record, map[string]any{"priority": "high"})
		h.signalAt(time.Hour, outboxID, sig)
		h.signalAt(time.Hour+time.Second, outboxID, sig)
		h.start(version, record)
		h.wantCompleted()
		var tr trace
		tr.refs, tr.timers = h.nextRefs(), h.timerDurations()
		for _, c := range h.calls() {
			tr.steps = append(tr.steps, c.Step)
		}
		return tr
	}
	a, b := runOnce(), runOnce()
	if len(a.refs) != 4 || len(a.steps) != 1 {
		t.Fatalf("first run refs %+v writes %v, want 4 NextSteps and one write", a.refs, a.steps)
	}
	if !slices.EqualFunc(a.refs, b.refs, func(x, y interp.StepRef) bool {
		return x.DefinitionVersionID == y.DefinitionVersionID && x.RecordID == y.RecordID && x.Step == y.Step &&
			(x.Signal == nil) == (y.Signal == nil) && (x.Signal == nil || *x.Signal == *y.Signal)
	}) {
		t.Errorf("NextStep refs differ:\n%+v\n%+v", a.refs, b.refs)
	}
	if !slices.Equal(a.timers, b.timers) {
		t.Errorf("timers differ: %v vs %v", a.timers, b.timers)
	}
	if !slices.Equal(a.steps, b.steps) {
		t.Errorf("writes differ: %v vs %v", a.steps, b.steps)
	}
}

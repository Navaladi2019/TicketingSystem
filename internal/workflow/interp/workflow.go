package interp

import (
	"bytes"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"

	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
	"github.com/Navaladi2019/TicketingSystem/internal/workerguard"
)

// activityStartToClose bounds one NextStep or ApplyStep attempt. Retries use
// the SDK's default policy; a Rejection is non-retryable and ends them.
const activityStartToClose = time.Minute

// RegisterWorkflow adds Workflow to r as WorkflowType.
func RegisterWorkflow(r worker.WorkflowRegistry) {
	r.RegisterWorkflowWithOptions(Workflow, workflow.RegisterOptions{Name: WorkflowType})
}

// startPayload is the relay's start payload (D-3-12): both UUIDs, nothing
// else. The definition version is pinned here; the workflow never reads
// "latest".
type startPayload struct {
	DefinitionVersionID uuid.UUID `json:"definition_version_id"`
	RecordID            uuid.UUID `json:"record_id"`
}

// signalPayload is the payload of a delivery on SignalName.
type signalPayload struct {
	Signal uuid.UUID `json:"signal"`
}

// runRefs is RunInput.Refs, carried to the next run on continue-as-new: the
// step to resume at, and the deliveries this run received but did not act on
// yet, as two arrays of the same length (outbox id, signal id). UUIDs only,
// so outbox.ValidatePayload, and so the guard, accepts it.
type runRefs struct {
	Step          uuid.UUID   `json:"step"`
	PendingOutbox []uuid.UUID `json:"pending_outbox,omitempty"`
	PendingSignal []uuid.UUID `json:"pending_signal,omitempty"`
}

// delivered is one signal delivery: its outbox id (the dedupe key, I7) and
// the signal id from its payload.
type delivered struct{ outbox, signal uuid.UUID }

// decodeStrict decodes raw into v, refusing unknown fields and trailing data.
func decodeStrict(raw json.RawMessage, v any) bool {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if dec.Decode(v) != nil {
		return false
	}
	return !dec.More()
}

// Workflow is nava.interpreter. Its input is the relay's outbox.Delivery
// (embedded in workerguard.RunInput) with payload {definition_version_id,
// record_id}; after a continue-as-new, Refs holds the step to resume at.
//
// It acts only on the Directive NextStep returns:
//   - timer: a timer of WaitSeconds, then NextStep for Next;
//   - wait_signal: the SignalName channel, deduped on the outbox id, with a
//     WaitSeconds timeout; on the signal NextStep for Next with Signal set, on
//     the timeout NextStep for OnTimeout;
//   - update_record: ApplyStep, then NextStep for Next;
//   - end: CloseWorkflowRef, then return.
//
// It continues as new, when the server suggests it, only between steps: after
// NextStep returns and before the Directive's wait starts, never during one,
// because Refs carries only the step and a new run would restart the wait.
func Workflow(ctx workflow.Context, in workerguard.RunInput) error {
	var start startPayload
	if !decodeStrict(in.Payload, &start) || start.DefinitionVersionID == uuid.Nil || start.RecordID == uuid.Nil {
		return workerguard.Rejected(workerguard.RejectInput)
	}
	var refs runRefs
	if len(in.Refs) > 0 && (!decodeStrict(in.Refs, &refs) || len(refs.PendingOutbox) != len(refs.PendingSignal)) {
		return workerguard.Rejected(workerguard.RejectInput)
	}
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{StartToCloseTimeout: activityStartToClose})
	r := &run{
		in:      in,
		ch:      workflow.GetSignalChannel(ctx, SignalName),
		pending: make([]delivered, 0, len(refs.PendingOutbox)),
	}
	for i := range refs.PendingOutbox {
		r.pending = append(r.pending, delivered{outbox: refs.PendingOutbox[i], signal: refs.PendingSignal[i]})
	}
	ref := StepRef{DefinitionVersionID: start.DefinitionVersionID, RecordID: start.RecordID, Step: refs.Step}
	for {
		var d workerguard.Directive
		if err := workflow.ExecuteActivity(ctx, NextStepActivity, ref).Get(ctx, &d); err != nil {
			return err
		}
		if workflow.GetInfo(ctx).GetContinueAsNewSuggested() {
			return r.continueAsNew(ctx, d.Step)
		}
		ref.Signal = nil
		var (
			next uuid.UUID
			err  error
		)
		switch d.Kind {
		case workerguard.KindTimer:
			if err = workflow.NewTimer(ctx, seconds(d.WaitSeconds)).Get(ctx, nil); err == nil {
				next, err = follow(d.Next)
			}
		case workerguard.KindWaitSignal:
			next, err = r.waitSignal(ctx, d, &ref)
		case workerguard.KindUpdateRecord:
			apply := StepRef{DefinitionVersionID: ref.DefinitionVersionID, RecordID: ref.RecordID, Step: d.Step}
			if err = workflow.ExecuteActivity(ctx, ApplyStepActivity, apply).Get(ctx, nil); err == nil {
				next, err = follow(d.Next)
			}
		case workerguard.KindEnd:
			return workerguard.CloseWorkflowRef(ctx)
		default:
			// The guard refuses any other Kind in a recorded result, so this
			// is a backstop: a fixed failure, never a panic (AR-042).
			return workerguard.Rejected(workerguard.RejectDirective)
		}
		if err != nil {
			return err
		}
		ref.Step = next
	}
}

// run is one run's signal state.
type run struct {
	in      workerguard.RunInput
	ch      workflow.ReceiveChannel
	pending []delivered
}

func seconds(n int32) time.Duration { return time.Duration(n) * time.Second }

// follow dereferences a Directive edge; the guard makes a nil one
// unreachable, and it fails the run rather than panic.
func follow(id *uuid.UUID) (uuid.UUID, error) {
	if id == nil || *id == uuid.Nil {
		return uuid.Nil, workerguard.Rejected(workerguard.RejectDirective)
	}
	return *id, nil
}

// waitSignal waits for the Directive's signal, or its timeout. One timer runs
// for the whole wait. A delivery is applied only if its signal is the one
// awaited and its outbox id is new (AppliedSignals, I7); others are dropped.
// On the signal it sets ref.Signal and returns Next, on the timeout OnTimeout.
func (r *run) waitSignal(ctx workflow.Context, d workerguard.Directive, ref *StepRef) (uuid.UUID, error) {
	if d.Signal == nil {
		return uuid.Nil, workerguard.Rejected(workerguard.RejectDirective)
	}
	want := *d.Signal
	// Deliveries carried from the previous run come first, in arrival order.
	for len(r.pending) > 0 {
		p := r.pending[0]
		r.pending = r.pending[1:]
		if p.signal == want && r.in.Seen.TryApply(ctx, p.outbox) {
			ref.Signal = &want
			return follow(d.Next)
		}
	}
	timerCtx, cancel := workflow.WithCancel(ctx)
	defer cancel()
	timer := workflow.NewTimer(timerCtx, seconds(d.WaitSeconds))
	for {
		var (
			got      outbox.Delivery
			received bool
			timerErr error
			fired    bool
		)
		sel := workflow.NewSelector(ctx)
		sel.AddReceive(r.ch, func(c workflow.ReceiveChannel, _ bool) {
			c.Receive(ctx, &got)
			received = true
		})
		sel.AddFuture(timer, func(f workflow.Future) {
			fired = true
			timerErr = f.Get(ctx, nil)
		})
		sel.Select(ctx)
		if fired {
			if timerErr != nil {
				return uuid.Nil, timerErr
			}
			return follow(d.OnTimeout)
		}
		if !received {
			continue
		}
		if sig, ok := signalOf(got); ok && sig == want && r.in.Seen.TryApply(ctx, got.OutboxID) {
			ref.Signal = &want
			return follow(d.Next)
		}
	}
}

// signalOf reads the signal id from a delivery's payload.
func signalOf(d outbox.Delivery) (uuid.UUID, bool) {
	var p signalPayload
	if !decodeStrict(d.Payload, &p) || p.Signal == uuid.Nil {
		return uuid.Nil, false
	}
	return p.Signal, true
}

// continueAsNew carries step, the applied-signal set and every delivery not
// yet acted on (carried, and still buffered on the channel) to a new run.
func (r *run) continueAsNew(ctx workflow.Context, step uuid.UUID) error {
	for {
		var got outbox.Delivery
		if !r.ch.ReceiveAsync(&got) {
			break
		}
		if sig, ok := signalOf(got); ok && got.OutboxID != uuid.Nil {
			r.pending = append(r.pending, delivered{outbox: got.OutboxID, signal: sig})
		}
	}
	refs := runRefs{Step: step}
	for _, p := range r.pending {
		refs.PendingOutbox = append(refs.PendingOutbox, p.outbox)
		refs.PendingSignal = append(refs.PendingSignal, p.signal)
	}
	b, err := json.Marshal(refs)
	if err != nil {
		return workerguard.Rejected(workerguard.RejectInput)
	}
	in := r.in
	in.Refs = b
	return workflow.NewContinueAsNewError(ctx, WorkflowType, in)
}

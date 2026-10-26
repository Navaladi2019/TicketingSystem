package workerguard

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/workflow"

	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
)

// dedupeWorkflow applies "outbox" signals (the arg is the outbox id) once
// each, continues as new on "rollover" carrying the set and what it applied
// (a RunInput, the one shape the argument rule accepts), and returns what it
// applied on "done".
func dedupeWorkflow(ctx workflow.Context, in RunInput) ([]string, error) {
	var refs struct {
		Applied []string `json:"applied"`
	}
	if in.Refs != nil {
		if err := json.Unmarshal(in.Refs, &refs); err != nil {
			return nil, err
		}
	}
	signals := workflow.GetSignalChannel(ctx, "outbox")
	rollover := workflow.GetSignalChannel(ctx, "rollover")
	done := workflow.GetSignalChannel(ctx, "done")
	for {
		var next string
		sel := workflow.NewSelector(ctx)
		sel.AddReceive(signals, func(c workflow.ReceiveChannel, _ bool) {
			var id uuid.UUID
			c.Receive(ctx, &id)
			if in.Seen.TryApply(ctx, id) {
				refs.Applied = append(refs.Applied, id.String())
			}
		})
		sel.AddReceive(rollover, func(c workflow.ReceiveChannel, _ bool) { c.Receive(ctx, nil); next = "rollover" })
		sel.AddReceive(done, func(c workflow.ReceiveChannel, _ bool) { c.Receive(ctx, nil); next = "done" })
		sel.Select(ctx)
		switch next {
		case "rollover":
			b, err := json.Marshal(refs)
			if err != nil {
				return nil, err
			}
			in.Refs = b
			return nil, workflow.NewContinueAsNewError(ctx, dedupeWorkflow, in)
		case "done":
			return refs.Applied, nil
		}
	}
}

// A signal delivered once before and once after continue-as-new is applied
// once: the set travels in the continue-as-new input.
func TestSignalAppliedOnceAcrossContinueAsNew(t *testing.T) {
	id := idOf(tenantA)
	s1, s2, s3 := uuid.New(), uuid.New(), uuid.New()

	// First run: s1 twice (a relay redelivery), s2, then continue-as-new.
	h := newHarness(t, id)
	h.tr.setSignal(stringPayload(t, tenantA.String()))
	for i, sig := range []struct {
		name string
		arg  any
	}{{"outbox", s1}, {"outbox", s1}, {"outbox", s2}, {"rollover", nil}} {
		h.env.RegisterDelayedCallback(func() { h.env.SignalWorkflow(sig.name, sig.arg) }, time.Duration(i+1)*time.Second)
	}
	start := RunInput{Delivery: outboxDelivery(uuid.New())}
	h.env.ExecuteWorkflow(dedupeWorkflow, start)
	var can *workflow.ContinueAsNewError
	if err := h.env.GetWorkflowError(); !errors.As(err, &can) {
		t.Fatalf("first run: %v, want continue-as-new", err)
	}
	var carried RunInput
	if err := converter.GetDefaultDataConverter().FromPayloads(can.Input, &carried); err != nil {
		t.Fatalf("decode continue-as-new input: %v", err)
	}
	if want := `{"applied":["` + s1.String() + `","` + s2.String() + `"]}`; carried.Seen.Len() != 2 || string(carried.Refs) != want {
		t.Fatalf("first run carried refs %s (set of %d), want s1, s2 once", carried.Refs, carried.Seen.Len())
	}
	if carried.OutboxID != start.OutboxID {
		t.Errorf("carried outbox id %s, want the start's %s", carried.OutboxID, start.OutboxID)
	}

	// Second run, from the continue-as-new input: s1 and s2 are redelivered
	// after the continue-as-new, s3 is new.
	h2 := newHarness(t, id)
	h2.tr.setSignal(stringPayload(t, tenantA.String()))
	for i, sig := range []struct {
		name string
		arg  any
	}{{"outbox", s1}, {"outbox", s3}, {"outbox", s2}, {"done", nil}} {
		h2.env.RegisterDelayedCallback(func() { h2.env.SignalWorkflow(sig.name, sig.arg) }, time.Duration(i+1)*time.Second)
	}
	h2.env.ExecuteWorkflow(dedupeWorkflow, carried)
	var final []string
	if err := h2.env.GetWorkflowResult(&final); err != nil {
		t.Fatal(err)
	}
	if want := strings.Join([]string{s1.String(), s2.String(), s3.String()}, ","); strings.Join(final, ",") != want {
		t.Errorf("applied %v across continue-as-new, want each of s1, s2, s3 once", final)
	}
}

// The set forgets an id SignalDedupeWindow (48 h, D-P8) after it was
// applied, which bounds it; inside the window a redelivery is not applied.
func TestAppliedSignalsWindow(t *testing.T) {
	h := newHarness(t, idOf(tenantA))
	h.env.ExecuteWorkflow(func(ctx workflow.Context) error {
		var s AppliedSignals
		id, other := uuid.New(), uuid.New()
		if !s.TryApply(ctx, id) {
			return errors.New("a new id was not applied")
		}
		if s.TryApply(ctx, id) {
			return errors.New("a redelivery was applied")
		}
		if s.TryApply(ctx, uuid.Nil) {
			return errors.New("a nil id was applied")
		}
		if err := workflow.Sleep(ctx, 47*time.Hour); err != nil {
			return err
		}
		if s.TryApply(ctx, id) {
			return errors.New("a redelivery 47 h later was applied")
		}
		if !s.TryApply(ctx, other) {
			return errors.New("a second id was not applied")
		}
		if err := workflow.Sleep(ctx, 2*time.Hour); err != nil {
			return err
		}
		// 49 h after id, 2 h after other: id is pruned, other is kept.
		if !s.TryApply(ctx, uuid.New()) || s.Len() != 2 {
			return errors.New("the set was not pruned to the 48 h window")
		}
		if _, kept := s.Applied[id]; kept {
			return errors.New("an id older than 48 h was kept")
		}
		return nil
	})
	if err := h.env.GetWorkflowError(); err != nil {
		t.Fatal(err)
	}
	if SignalDedupeWindow != 48*time.Hour {
		t.Errorf("SignalDedupeWindow = %v, want 48h (D-P8)", SignalDedupeWindow)
	}
}

// outboxDelivery is the start input the relay sends for outbox row id.
func outboxDelivery(id uuid.UUID) outbox.Delivery {
	return outbox.Delivery{OutboxID: id, KindVersion: 1, Payload: json.RawMessage(`{}`)}
}

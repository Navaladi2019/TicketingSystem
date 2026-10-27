package workerguard

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"

	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
)

// recordData is a ticket title a tenant typed: it must never be in a
// history, an error or a log line (ADR-001 R1, CI-7).
const recordData = "CI7-RECORD-DATA: printer on fire in room 4"

func refsJSON(t *testing.T, v any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func validDelivery(t *testing.T) outbox.Delivery {
	t.Helper()
	cause := uuid.New()
	return outbox.Delivery{
		OutboxID: uuid.New(), KindVersion: 1, CausationID: &cause, CausationDepth: 2,
		Payload: refsJSON(t, map[string]string{"record_id": uuid.NewString()}),
	}
}

func validRunInput(t *testing.T) RunInput {
	t.Helper()
	in := RunInput{Delivery: validDelivery(t), Refs: refsJSON(t, map[string][]string{"order": {uuid.NewString()}})}
	in.Seen.Applied = map[uuid.UUID]time.Time{uuid.New(): time.Date(2026, 10, 26, 12, 0, 0, 123, time.UTC)}
	return in
}

// lookalike has RunInput's JSON shape, but it is not RunInput.
type lookalike struct {
	outbox.Delivery
	Seen AppliedSignals  `json:"seen"`
	Refs json.RawMessage `json:"refs,omitempty"`
}

func TestArgOK(t *testing.T) {
	ref := uuid.NewString()
	marker := refsJSON(t, map[string]string{"title": recordData})

	badPayload := validDelivery(t)
	badPayload.Payload = marker
	nilPayload := validDelivery(t)
	nilPayload.Payload = nil
	badRefs := validRunInput(t)
	badRefs.Refs = marker
	badInPayload := validRunInput(t)
	badInPayload.Payload = marker
	noRefs := validRunInput(t)
	noRefs.Refs = nil
	firstRun := RunInput{Delivery: validDelivery(t)}
	var nilDelivery *outbox.Delivery
	d, in := validDelivery(t), validRunInput(t)

	for name, tc := range map[string]struct {
		arg  any
		want bool
	}{
		"reference map":               {map[string]string{"record_id": ref}, true},
		"reference list map":          {map[string][]string{"ids": {ref, uuid.NewString()}}, true},
		"reference raw JSON":          {json.RawMessage(`{"record_id":"` + ref + `"}`), true},
		"empty object":                {map[string]string{}, true},
		"Delivery":                    {validDelivery(t), true},
		"*Delivery":                   {&d, true},
		"RunInput":                    {validRunInput(t), true},
		"*RunInput":                   {&in, true},
		"RunInput without Refs":       {noRefs, true},
		"RunInput of a first run":     {firstRun, true},
		"title in a map":              {map[string]string{"title": recordData}, false},
		"title next to a reference":   {map[string]string{"record_id": ref, "title": recordData}, false},
		"title as raw JSON":           {marker, false},
		"string":                      {recordData, false},
		"reference as a bare string":  {ref, false},
		"number":                      {5551234567, false},
		"bool":                        {true, false},
		"nil":                         {nil, false},
		"nil *Delivery":               {nilDelivery, false},
		"bytes":                       {[]byte(recordData), false},
		"nested object":               {map[string]any{"r": map[string]string{"id": ref}}, false},
		"Delivery with title payload": {badPayload, false},
		"Delivery with nil payload":   {nilPayload, false},
		"RunInput with title Refs":    {badRefs, false},
		"RunInput with title payload": {badInPayload, false},
		"RunInput lookalike":          {lookalike{Delivery: validDelivery(t)}, false},
		"unencodable":                 {func() {}, false},
	} {
		if got := argOK(tc.arg); got != tc.want {
			t.Errorf("%s: argOK = %v, want %v", name, got, tc.want)
		}
	}
	if got := argsOutcome(nil); got != outcomeAllow {
		t.Errorf("no arguments: %q, want allow", got)
	}
	if got := argsOutcome([]any{map[string]string{"record_id": ref}, recordData}); got != outcomeRecordData {
		t.Errorf("one bad argument of two: %q, want %q", got, outcomeRecordData)
	}
}

// Seen cannot carry data either: every key a UUID, every value an RFC3339
// time, and no other field. AppliedSignals cannot encode anything else, so
// the encoded forms are checked directly.
func TestSeenOK(t *testing.T) {
	id := uuid.NewString()
	for raw, want := range map[string]bool{
		`{}`:             true,
		`{"applied":{}}`: true,
		`{"applied":{"` + id + `":"2026-10-26T12:00:00.000000123Z"}}`: true,
		`{"applied":{"` + id + `":"2026-10-26T12:00:00+05:30"}}`:      true,
		`{"applied":{"` + recordData + `":"2026-10-26T12:00:00Z"}}`:   false,
		`{"applied":{"` + id + `":"` + recordData + `"}}`:             false,
		`{"applied":{"` + id + `":5551234567}}`:                       false,
		`{"applied":[]}`:                                              false,
		`{"applied":{},"title":"` + recordData + `"}`:                 false,
		`null`: false,
		`"x"`:  false,
		``:     false,
	} {
		if got := seenOK(json.RawMessage(raw)); got != want {
			t.Errorf("seenOK(%s) = %v, want %v", raw, got, want)
		}
	}
}

// wantNoRecordData fails if the record data is in err or in the guard log.
func (h *harness) wantNoRecordData(err error) {
	h.t.Helper()
	if err != nil && strings.Contains(err.Error(), "printer on fire") {
		h.t.Errorf("the error holds the record data: %v", err)
	}
	if strings.Contains(h.logs.String(), "printer on fire") {
		h.t.Errorf("the guard log holds the record data:\n%s", h.logs.String())
	}
}

func echoArg(_ context.Context, in map[string]string) (map[string]string, error) { return in, nil }

func TestActivityArgsGuard(t *testing.T) {
	id := idOf(tenantA)
	for _, tc := range []struct {
		name, guard string
		run         func(ctx workflow.Context, act, arg any) workflow.Future
	}{
		{"activity", guardActivityArgs, func(ctx workflow.Context, act, arg any) workflow.Future {
			ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
				StartToCloseTimeout: time.Minute, RetryPolicy: &temporal.RetryPolicy{MaximumAttempts: 1},
			})
			return workflow.ExecuteActivity(ctx, act, arg)
		}},
		{"local activity", guardLocalActivityArgs, func(ctx workflow.Context, act, arg any) workflow.Future {
			ctx = workflow.WithLocalActivityOptions(ctx, workflow.LocalActivityOptions{
				StartToCloseTimeout: time.Minute, RetryPolicy: &temporal.RetryPolicy{MaximumAttempts: 1},
			})
			return workflow.ExecuteLocalActivity(ctx, act, arg)
		}},
	} {
		t.Run(tc.name+" with record data fails before it is scheduled", func(t *testing.T) {
			h := newHarness(t, id)
			ran := false
			act := func(_ context.Context, in map[string]string) (map[string]string, error) { ran = true; return in, nil }
			h.env.RegisterActivityWithOptions(act, activity.RegisterOptions{Name: "probe"})
			h.env.ExecuteWorkflow(func(ctx workflow.Context) error {
				var target any = "probe"
				if tc.guard == guardLocalActivityArgs {
					target = act
				}
				return tc.run(ctx, target, map[string]string{"title": recordData}).Get(ctx, nil)
			})
			err := h.env.GetWorkflowError()
			wantGuardError(t, err, tc.guard, outcomeRecordData)
			if ran {
				t.Errorf("the %s ran", tc.name)
			}
			h.wantDenyAlert(tc.guard, outcomeRecordData, id)
			h.wantNoRecordData(err)
		})
		t.Run(tc.name+" with references runs", func(t *testing.T) {
			h := newHarness(t, id)
			h.env.RegisterActivity(echoArg)
			ref := map[string]string{"record_id": uuid.NewString()}
			h.env.ExecuteWorkflow(func(ctx workflow.Context) (map[string]string, error) {
				var out map[string]string
				err := tc.run(ctx, echoArg, ref).Get(ctx, &out)
				return out, err
			})
			var out map[string]string
			if err := h.env.GetWorkflowResult(&out); err != nil || out["record_id"] != ref["record_id"] {
				t.Fatalf("result %v, %v; want the reference back", out, err)
			}
			h.wantNoAlerts()
			if d := h.decisions()[tc.guard+"/"+outcomeAllow]; d != 1 {
				t.Errorf("%s allow count %d, want 1", tc.guard, d)
			}
		})
	}
}

func TestChildArgsGuard(t *testing.T) {
	id := idOf(tenantA)
	run := func(t *testing.T, arg any) (*harness, bool) {
		t.Helper()
		h := newHarness(t, id)
		ran := false
		h.env.RegisterWorkflowWithOptions(func(workflow.Context, any) error { ran = true; return nil },
			workflow.RegisterOptions{Name: "child"})
		h.env.ExecuteWorkflow(func(ctx workflow.Context) error {
			cid, err := ChildWorkflowID(ctx, "child")
			if err != nil {
				return err
			}
			cctx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{WorkflowID: cid})
			return workflow.ExecuteChildWorkflow(cctx, "child", arg).Get(ctx, nil)
		})
		return h, ran
	}
	t.Run("record data fails before the child starts", func(t *testing.T) {
		h, ran := run(t, map[string]string{"title": recordData})
		err := h.env.GetWorkflowError()
		wantGuardError(t, err, guardChild, outcomeRecordData)
		if ran {
			t.Error("the child ran")
		}
		h.wantDenyAlert(guardChild, outcomeRecordData, id)
		h.wantNoRecordData(err)
	})
	t.Run("a Delivery starts it", func(t *testing.T) {
		h, ran := run(t, validDelivery(t))
		if err := h.env.GetWorkflowError(); err != nil || !ran {
			t.Fatalf("error %v, ran %v; want the child run", err, ran)
		}
		h.wantNoAlerts()
	})
}

func continueWith(arg any) func(workflow.Context) error {
	return func(ctx workflow.Context) error { return workflow.NewContinueAsNewError(ctx, "next", arg) }
}

func TestContinueAsNewArgsGuard(t *testing.T) {
	id := idOf(tenantA)
	t.Run("RunInput continues", func(t *testing.T) {
		h := newHarness(t, id)
		h.env.ExecuteWorkflow(continueWith(validRunInput(t)))
		var can *workflow.ContinueAsNewError
		if err := h.env.GetWorkflowError(); !errors.As(err, &can) {
			t.Fatalf("error %v, want continue-as-new", err)
		}
		h.wantNoAlerts()
	})
	for name, arg := range map[string]any{
		"title in a map": map[string]string{"title": recordData},
		"RunInput with title Refs": func() RunInput {
			in := validRunInput(t)
			in.Refs = refsJSON(t, map[string]string{"title": recordData})
			return in
		}(),
		"RunInput lookalike": lookalike{Delivery: validDelivery(t)},
	} {
		t.Run(name+" fails the workflow", func(t *testing.T) {
			h := newHarness(t, id)
			h.env.ExecuteWorkflow(continueWith(arg))
			err := h.env.GetWorkflowError()
			wantGuardError(t, err, guardContinueAsNew, outcomeRecordData)
			h.wantDenyAlert(guardContinueAsNew, outcomeRecordData, id)
			h.wantNoRecordData(err)
		})
	}
}

// Adversarial: ContinueAsNewError is plain data. Workflow code can take the
// error NewContinueAsNewError checked and swap its Input for record data, or
// build the error itself. The input must still be one the guard checked.
func TestAdversarialContinueAsNewInputRewrittenAfterGuard(t *testing.T) {
	title, err := converter.GetDefaultDataConverter().ToPayloads(map[string]string{"title": recordData})
	if err != nil {
		t.Fatal(err)
	}
	for name, wf := range map[string]func(workflow.Context) error{
		"input swapped": func(ctx workflow.Context) error {
			err := workflow.NewContinueAsNewError(ctx, "next", validRunInput(t))
			var can *workflow.ContinueAsNewError
			if !errors.As(err, &can) {
				return err
			}
			can.Input = title
			return can
		},
		"payload data rewritten": func(ctx workflow.Context) error {
			err := workflow.NewContinueAsNewError(ctx, "next", map[string]string{"record_id": uuid.NewString()})
			var can *workflow.ContinueAsNewError
			if !errors.As(err, &can) {
				return err
			}
			can.Input.Payloads[0].Data = title.Payloads[0].Data
			return can
		},
		"error built by hand": func(ctx workflow.Context) error {
			// Header and type copied from a checked error, the input not.
			err := workflow.NewContinueAsNewError(ctx, "next")
			var can *workflow.ContinueAsNewError
			if !errors.As(err, &can) {
				return err
			}
			return &workflow.ContinueAsNewError{WorkflowType: can.WorkflowType, Input: title, Header: can.Header, TaskQueueName: can.TaskQueueName}
		},
	} {
		t.Run(name, func(t *testing.T) {
			id := idOf(tenantA)
			h := newHarness(t, id)
			h.env.ExecuteWorkflow(wf)
			err := h.env.GetWorkflowError()
			wantGuardError(t, err, guardContinueAsNew, outcomeRecordData)
			h.wantDenyAlert(guardContinueAsNew, outcomeRecordData, id)
			h.wantNoRecordData(err)
		})
	}
}

func TestCheckedInputs(t *testing.T) {
	var c checkedInputs
	if !c.has(nil) || !c.has(&commonpb.Payloads{}) {
		t.Error("an empty input (no arguments) must pass")
	}
	p, err := converter.GetDefaultDataConverter().ToPayloads(map[string]string{"record_id": uuid.NewString()})
	if err != nil {
		t.Fatal(err)
	}
	if c.has(p) {
		t.Error("an input that was never checked passed")
	}
	c.add(p)
	if !c.has(p) {
		t.Error("a checked input did not pass")
	}
	p.Payloads[0].Data = []byte(`{"title":"x"}`)
	if c.has(p) {
		t.Error("a checked input changed after the check passed: add must keep a copy")
	}
}

// withRefsUpdate counts "refs" updates in n.
func withRefsUpdate(ctx workflow.Context, n *int) error {
	if err := workflow.SetUpdateHandler(ctx, "refs", func(_ workflow.Context, _ map[string]string) error { *n++; return nil }); err != nil {
		return err
	}
	if err := workflow.Await(ctx, func() bool { return *n >= 1 }); err != nil {
		return err
	}
	return workflow.Sleep(ctx, time.Hour)
}

func TestUpdateArgsGuard(t *testing.T) {
	id := idOf(tenantA)
	h := newHarness(t, id)
	h.tr.setUpdate(stringPayload(t, tenantA.String()))
	results := map[string]*updateResult{}
	update := func(name string, arg any) {
		r := &updateResult{}
		results[name] = r
		h.env.UpdateWorkflow("refs", name, &testsuite.TestUpdateCallback{
			OnReject:   func(err error) { r.rejected = err },
			OnAccept:   func() { r.accepted = true },
			OnComplete: func(any, error) {},
		}, arg)
	}
	h.env.RegisterDelayedCallback(func() { update("title", map[string]string{"title": recordData}) }, time.Second)
	h.env.RegisterDelayedCallback(func() { update("refs", map[string]string{"record_id": uuid.NewString()}) }, 2*time.Second)
	var n int
	h.env.ExecuteWorkflow(func(ctx workflow.Context) error { return withRefsUpdate(ctx, &n) })

	if err := h.env.GetWorkflowError(); err != nil || n != 1 {
		t.Fatalf("result %d, %v; want 1 applied update", n, err)
	}
	if r := results["title"]; r.accepted {
		t.Error("the update with record data was accepted")
	}
	wantGuardError(t, results["title"].rejected, guardUpdate, outcomeRecordData)
	h.wantNoRecordData(results["title"].rejected)
	if r := results["refs"]; r.rejected != nil || !r.accepted {
		t.Errorf("references-only update: rejected %v, accepted %v", r.rejected, r.accepted)
	}
	h.wantDenyAlert(guardUpdate, outcomeRecordData, id)
}

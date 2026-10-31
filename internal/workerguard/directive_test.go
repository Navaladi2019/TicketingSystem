package workerguard

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/workflow"
)

// P3-26: the guard accepts workerguard.Directive, and nothing else that is not
// references only, as an activity result. Each refusal below is a way record
// data could ride a Directive-shaped value into a history (ADR-001 R1, CI-7).

func ref() *uuid.UUID { u := uuid.New(); return &u }

func validDirectives() map[Kind]Directive {
	return map[Kind]Directive{
		KindTimer:        {Step: uuid.New(), Kind: KindTimer, WaitSeconds: 3600, Next: ref()},
		KindWaitSignal:   {Step: uuid.New(), Kind: KindWaitSignal, WaitSeconds: MaxWaitSeconds, Signal: ref(), Next: ref(), OnTimeout: ref()},
		KindUpdateRecord: {Step: uuid.New(), Kind: KindUpdateRecord, Next: ref()},
		KindEnd:          {Step: uuid.New(), Kind: KindEnd},
	}
}

func TestDirectiveEachKindPasses(t *testing.T) {
	for kind, d := range validDirectives() {
		if !resultOK(d, true) {
			t.Errorf("%s: a valid Directive was refused", kind)
		}
		if !resultOK(&d, true) {
			t.Errorf("%s: a pointer to a valid Directive was refused", kind)
		}
	}
	// The table has a row for every Kind constant and no other.
	if len(directiveRules) != 4 {
		t.Errorf("directiveRules has %d rows, want the 4 Kind constants", len(directiveRules))
	}
	if !resultOK(&Directive{Step: uuid.New(), Kind: KindTimer, WaitSeconds: 1, Next: ref()}, true) {
		t.Error("timer with the minimum wait was refused")
	}
}

// TestDirectiveOnlyAsActivityResult: a Directive is not an argument, a workflow
// or update result, a side effect or a local activity's result.
func TestDirectiveOnlyAsActivityResult(t *testing.T) {
	d := validDirectives()[KindTimer]
	if argOK(d) {
		t.Error("a Directive passed as an activity argument")
	}
	if resultOK(d, false) {
		t.Error("a Directive passed as a workflow, update or local activity result")
	}
	if sideEffectValueOK(d) {
		t.Error("a Directive passed as a side effect value")
	}
}

// directiveLookalike has Directive's JSON shape, but it is not Directive.
type directiveLookalike struct {
	Step        uuid.UUID  `json:"step"`
	Kind        Kind       `json:"kind"`
	WaitSeconds int32      `json:"wait_seconds"`
	Signal      *uuid.UUID `json:"signal,omitempty"`
	Next        *uuid.UUID `json:"next,omitempty"`
	OnTimeout   *uuid.UUID `json:"on_timeout,omitempty"`
}

// smuggler encodes as a valid Directive plus a field of record data.
type smuggler struct{ Directive }

func (s smuggler) MarshalJSON() ([]byte, error) {
	b, err := json.Marshal(s.Directive)
	if err != nil {
		return nil, err
	}
	return append(b[:len(b)-1], []byte(`,"title":"`+recordData+`"}`)...), nil
}

func TestDirectiveOtherTypesRefused(t *testing.T) {
	d := validDirectives()[KindTimer]
	var asMap map[string]any
	if err := json.Unmarshal(refsJSON(t, d), &asMap); err != nil {
		t.Fatal(err)
	}
	for name, v := range map[string]any{
		"same shape, another type":          directiveLookalike(d),
		"map with the same keys":            asMap,
		"custom MarshalJSON adding a field": smuggler{d},
		"embedding Directive":               struct{ Directive }{d},
	} {
		if resultOK(v, true) {
			t.Errorf("%s: accepted as an activity result", name)
		}
	}
}

// TestDirectiveGoValueNegatives: Directive values that break the per-kind
// table are refused.
func TestDirectiveGoValueNegatives(t *testing.T) {
	step, next, sig, onTO := uuid.New(), ref(), ref(), ref()
	nilID := uuid.Nil
	for name, d := range map[string]Directive{
		"unknown kind":                {Step: step, Kind: "close_ticket", Next: next},
		"kind holding record data":    {Step: step, Kind: Kind(recordData), Next: next},
		"timer wait 0":                {Step: step, Kind: KindTimer, WaitSeconds: 0, Next: next},
		"timer wait 2592001":          {Step: step, Kind: KindTimer, WaitSeconds: MaxWaitSeconds + 1, Next: next},
		"timer negative wait":         {Step: step, Kind: KindTimer, WaitSeconds: -5, Next: next},
		"wait_signal wait 0":          {Step: step, Kind: KindWaitSignal, Signal: sig, Next: next, OnTimeout: onTO},
		"nil step":                    {Step: uuid.Nil, Kind: KindEnd},
		"nil next":                    {Step: step, Kind: KindUpdateRecord, Next: &nilID},
		"update_record with a wait":   {Step: step, Kind: KindUpdateRecord, WaitSeconds: 60, Next: next},
		"update_record without next":  {Step: step, Kind: KindUpdateRecord},
		"end with a wait":             {Step: step, Kind: KindEnd, WaitSeconds: 60},
		"end with a next":             {Step: step, Kind: KindEnd, Next: next},
		"wait_signal without signal":  {Step: step, Kind: KindWaitSignal, WaitSeconds: 60, Next: next, OnTimeout: onTO},
		"wait_signal without timeout": {Step: step, Kind: KindWaitSignal, WaitSeconds: 60, Signal: sig, Next: next},
		"timer with a signal":         {Step: step, Kind: KindTimer, WaitSeconds: 60, Signal: sig, Next: next},
		"timer with on_timeout":       {Step: step, Kind: KindTimer, WaitSeconds: 60, Next: next, OnTimeout: onTO},
		"timer without next":          {Step: step, Kind: KindTimer, WaitSeconds: 60},
	} {
		if resultOK(d, true) {
			t.Errorf("%s: accepted", name)
		}
	}
}

// TestDirectiveEncodingNegatives: the rule checks the recorded bytes, so a
// custom encoder cannot get these past it either.
func TestDirectiveEncodingNegatives(t *testing.T) {
	step, next, sig, onTO := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	ok := `{"step":"` + step + `","kind":"timer","wait_seconds":60,"next":"` + next + `"}`
	if !directiveOK([]byte(ok)) {
		t.Fatalf("the baseline encoding was refused: %s", ok)
	}
	for name, enc := range map[string]string{
		"extra field":          `{"step":"` + step + `","kind":"timer","wait_seconds":60,"next":"` + next + `","title":"` + recordData + `"}`,
		"repeated step":        `{"step":"` + recordData + `","step":"` + step + `","kind":"timer","wait_seconds":60,"next":"` + next + `"}`,
		"repeated kind":        `{"step":"` + step + `","kind":"` + recordData + `","kind":"timer","wait_seconds":60,"next":"` + next + `"}`,
		"float wait":           `{"step":"` + step + `","kind":"timer","wait_seconds":60.5,"next":"` + next + `"}`,
		"float wait, integral": `{"step":"` + step + `","kind":"timer","wait_seconds":60.0,"next":"` + next + `"}`,
		"exponent wait":        `{"step":"` + step + `","kind":"timer","wait_seconds":6e1,"next":"` + next + `"}`,
		"string wait":          `{"step":"` + step + `","kind":"timer","wait_seconds":"60","next":"` + next + `"}`,
		"negative wait":        `{"step":"` + step + `","kind":"timer","wait_seconds":-60,"next":"` + next + `"}`,
		"wait over int32":      `{"step":"` + step + `","kind":"timer","wait_seconds":4294967356,"next":"` + next + `"}`,
		"missing wait":         `{"step":"` + step + `","kind":"timer","next":"` + next + `"}`,
		"missing step":         `{"kind":"end","wait_seconds":0}`,
		"missing kind":         `{"step":"` + step + `","wait_seconds":0}`,
		"null signal":          `{"step":"` + step + `","kind":"timer","wait_seconds":60,"next":"` + next + `","signal":null}`,
		"null next on end":     `{"step":"` + step + `","kind":"end","wait_seconds":0,"next":null}`,
		"null kind":            `{"step":"` + step + `","kind":null,"wait_seconds":0}`,
		"non-UUID step":        `{"step":"` + recordData + `","kind":"end","wait_seconds":0}`,
		"uppercase UUID":       `{"step":"` + strings.ToUpper(step) + `","kind":"end","wait_seconds":0}`,
		"braced UUID":          `{"step":"{` + step + `}","kind":"end","wait_seconds":0}`,
		"urn UUID":             `{"step":"urn:uuid:` + step + `","kind":"end","wait_seconds":0}`,
		"nil UUID step":        `{"step":"` + uuid.Nil.String() + `","kind":"end","wait_seconds":0}`,
		"step as object":       `{"step":{"id":"` + step + `"},"kind":"end","wait_seconds":0}`,
		"kind case":            `{"step":"` + step + `","kind":"END","wait_seconds":0}`,
		"trailing data":        `{"step":"` + step + `","kind":"end","wait_seconds":0}{"title":"` + recordData + `"}`,
		"array":                `[{"step":"` + step + `","kind":"end","wait_seconds":0}]`,
		"null":                 `null`,
		"wait_signal all set but signal non-UUID": `{"step":"` + step + `","kind":"wait_signal","wait_seconds":60,"next":"` + next + `","signal":"` + recordData + `","on_timeout":"` + onTO + `"}`,
		"end with a signal":                       `{"step":"` + step + `","kind":"end","wait_seconds":0,"signal":"` + sig + `"}`,
	} {
		if directiveOK([]byte(enc)) {
			t.Errorf("%s: accepted: %s", name, enc)
		}
	}
}

// TestDirectiveInWorkflow: through the SDK test environment and the whole
// guard, an activity's Directive is recorded; references-free data is still
// withheld; a local activity's Directive is refused (documented on
// activityOutcome).
func TestDirectiveInWorkflow(t *testing.T) {
	id := idOf(tenantA)
	want := validDirectives()[KindWaitSignal]
	directive := func(context.Context) (Directive, error) { return want, nil }

	t.Run("activity Directive completes", func(t *testing.T) {
		h := newHarness(t, id)
		h.env.RegisterActivityWithOptions(directive, activity.RegisterOptions{Name: "next_step"})
		h.env.ExecuteWorkflow(func(ctx workflow.Context) error {
			var got Directive
			ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{StartToCloseTimeout: 60e9})
			err := workflow.ExecuteActivity(ctx, "next_step").Get(ctx, &got)
			if err == nil && (got.Step != want.Step || *got.Signal != *want.Signal) {
				t.Errorf("the workflow read %+v, want %+v", got, want)
			}
			return err
		})
		if err := h.env.GetWorkflowError(); err != nil {
			t.Fatal(err)
		}
		h.wantNoAlerts()
	})

	t.Run("activity state map is withheld", func(t *testing.T) {
		h := newHarness(t, id)
		act := func(context.Context) (map[string]string, error) { return map[string]string{"state": "open"}, nil }
		h.env.RegisterActivityWithOptions(act, activity.RegisterOptions{Name: "load"})
		h.env.ExecuteWorkflow(func(ctx workflow.Context) error { return runActivity(ctx, false, "load") })
		err := h.env.GetWorkflowError()
		wantGuardError(t, err, guardActivityResult, outcomeRecordData)
		h.wantDenyAlert(guardActivityResult, outcomeRecordData, id)
	})

	t.Run("local activity Directive is refused", func(t *testing.T) {
		h := newHarness(t, id)
		h.env.ExecuteWorkflow(func(ctx workflow.Context) error { return runActivity(ctx, true, directive) })
		err := h.env.GetWorkflowError()
		wantGuardError(t, err, guardActivityResult, outcomeRecordData)
		h.wantDenyAlert(guardActivityResult, outcomeRecordData, id)
	})
}

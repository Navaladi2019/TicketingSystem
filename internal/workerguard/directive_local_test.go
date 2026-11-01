package workerguard

import (
	"log/slog"
	"testing"

	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/workflow"
)

// TestDirectiveActivityOutcomeFlag pins the plumbing in activityOutcome. The
// SDK test environment hands a local activity's result to the guard already
// encoded (*commonpb.Payloads), so TestDirectiveInWorkflow refuses it via the
// generic rule whatever the flag is. A real worker hands the Go value. Here a
// Directive arrives as a Go value: it is recorded only when directiveAllowed
// said so (P3-26, TICK-367).
func TestDirectiveActivityOutcomeFlag(t *testing.T) {
	rec, err := newRecorder(slog.New(slog.DiscardHandler), metricnoop.NewMeterProvider())
	if err != nil {
		t.Fatal(err)
	}
	d := validDirectives()[KindTimer]

	if got, err := rec.activityOutcome(d, nil, true); err != nil || got == nil {
		t.Errorf("allowed Directive: got (%v, %v), want it recorded", got, err)
	}
	got, err := rec.activityOutcome(d, nil, false)
	if got != nil {
		t.Errorf("Directive from an activity that may not return one: result %v reached the history", got)
	}
	wantGuardError(t, err, guardActivityResult, outcomeRecordData)
}

// TestDirectiveAllowedOnlyForNextStep: only a regular activity of type
// DirectiveActivity, called from a workflow of type DirectiveWorkflow, may
// record a Directive. A local NextStep, another interpreter activity, a name
// that differs by case or suffix, the empty name, and a NextStep called from
// any other workflow type (or with none) may not (D-P10 regression guard,
// TICK-367; workflow binding, TICK-386).
func TestDirectiveAllowedOnlyForNextStep(t *testing.T) {
	wf := func(name string) *workflow.Type { return &workflow.Type{Name: name} }
	info := func(name string, local bool) activity.Info {
		return activity.Info{ActivityType: activity.Type{Name: name}, IsLocalActivity: local, WorkflowType: wf(DirectiveWorkflow)}
	}
	from := func(w *workflow.Type) activity.Info {
		i := info(DirectiveActivity, false)
		i.WorkflowType = w
		return i
	}
	if !directiveAllowed(info(DirectiveActivity, false)) {
		t.Fatalf("regular %s from %s may not record a Directive", DirectiveActivity, DirectiveWorkflow)
	}
	for name, i := range map[string]activity.Info{
		"NextStep from another workflow":        from(wf("nava.smoke.relay")),
		"NextStep from a workflow case variant": from(wf("nava.Interpreter")),
		"NextStep from a workflow suffix":       from(wf(DirectiveWorkflow + "2")),
		"NextStep from a workflow prefix":       from(wf("x" + DirectiveWorkflow)),
		"NextStep from the empty workflow type": from(wf("")),
		"NextStep with no workflow type":        from(nil),
	} {
		if directiveAllowed(i) {
			t.Errorf("%s: may record a Directive", name)
		}
	}
	for name, i := range map[string]activity.Info{
		"local NextStep": info(DirectiveActivity, true),
		"ApplyStep":      info("nava.interpreter.ApplyStep", false),
		"workflow_ref":   info("nava.workflowref.Close", false),
		"case variant":   info("nava.interpreter.nextstep", false),
		"suffix":         info(DirectiveActivity+"2", false),
		"prefix":         info("x"+DirectiveActivity, false),
		"empty":          info("", false),
		"other, regular": info("next_step", false),
		"other, local":   info("next_step", true),
	} {
		if directiveAllowed(i) {
			t.Errorf("%s: may record a Directive", name)
		}
	}
}

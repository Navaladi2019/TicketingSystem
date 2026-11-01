package workerguard

import (
	"log/slog"
	"testing"

	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.temporal.io/sdk/activity"
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
// DirectiveActivity may record a Directive. A local NextStep, another
// interpreter activity, a name that differs by case or suffix, and the empty
// name may not (D-P10 regression guard, TICK-367).
func TestDirectiveAllowedOnlyForNextStep(t *testing.T) {
	info := func(name string, local bool) activity.Info {
		return activity.Info{ActivityType: activity.Type{Name: name}, IsLocalActivity: local}
	}
	if !directiveAllowed(info(DirectiveActivity, false)) {
		t.Fatalf("regular %s may not record a Directive", DirectiveActivity)
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

package workerguard

import (
	"log/slog"
	"testing"

	metricnoop "go.opentelemetry.io/otel/metric/noop"
)

// TestDirectiveActivityOutcomeLocalFlag pins the local-activity plumbing in
// activityOutcome. The SDK test environment hands a local activity's result
// to the guard already encoded (*commonpb.Payloads), so TestDirectiveInWorkflow
// refuses it via the generic rule whatever the flag is. A real worker hands
// the Go value. Here a Directive arrives as a Go value: it is recorded for a
// regular activity and withheld for a local one (P3-26).
func TestDirectiveActivityOutcomeLocalFlag(t *testing.T) {
	rec, err := newRecorder(slog.New(slog.DiscardHandler), metricnoop.NewMeterProvider())
	if err != nil {
		t.Fatal(err)
	}
	d := validDirectives()[KindTimer]

	if got, err := rec.activityOutcome(d, nil, false); err != nil || got == nil {
		t.Errorf("regular activity Directive: got (%v, %v), want it recorded", got, err)
	}
	got, err := rec.activityOutcome(d, nil, true)
	if got != nil {
		t.Errorf("local activity Directive: result %v reached the history", got)
	}
	wantGuardError(t, err, guardActivityResult, outcomeRecordData)
}

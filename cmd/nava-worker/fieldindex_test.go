package main

import (
	"slices"
	"testing"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"

	"github.com/Navaladi2019/TicketingSystem/internal/fieldindex"
	"github.com/Navaladi2019/TicketingSystem/internal/fieldindex/backfill"
	"github.com/Navaladi2019/TicketingSystem/internal/workflow/interp"
	"github.com/Navaladi2019/TicketingSystem/internal/workflow/recordports"
)

// The worker registers the field-index coordinator and FieldIndexBackfill
// (TICK-745) with workflows on, and their four activities on every build.
func TestRegisterFieldIndex(t *testing.T) {
	acts := []string{backfill.BarrierActivity, backfill.ClassifyActivity, backfill.BatchActivity, backfill.ReadyActivity}
	r := &recordingRegistry{}
	registerFieldIndex(r, &backfill.Activities{}, true)
	if want := []string{fieldindex.CoordinatorWorkflowType, fieldindex.BackfillWorkflowType}; !slices.Equal(r.workflows, want) {
		t.Errorf("workflows %v, want %v", r.workflows, want)
	}
	if !slices.Equal(r.activities, acts) {
		t.Errorf("activities %v, want %v", r.activities, acts)
	}

	r = &recordingRegistry{}
	registerFieldIndex(r, &backfill.Activities{}, false)
	if len(r.workflows) != 0 || !slices.Equal(r.activities, acts) {
		t.Errorf("%s=off: workflows %v activities %v, want none and %v", WorkflowsEnv, r.workflows, r.activities, acts)
	}
}

// The backfill counts on the interpreter writer's own uncoerced counter, so
// the worker registers nava_field_index_uncoerced once (TICK-974); without
// a writer it gets one of its own, and without a meter provider none.
func TestUncoercedCounterReusesTheWritersInstance(t *testing.T) {
	mp := sdkmetric.NewMeterProvider()
	w, err := recordports.NewWriter(nil, nil, mp)
	if err != nil || w.FieldIndex == nil {
		t.Fatalf("NewWriter = (%+v, %v), want a writer with a counter", w, err)
	}
	got, err := uncoercedCounter(&interp.Activities{Writer: w}, mp)
	if err != nil {
		t.Fatal(err)
	}
	if got != backfill.UncoercedCounter(w.FieldIndex) {
		t.Errorf("uncoercedCounter = %v, want the writer's instance %p", got, w.FieldIndex)
	}

	if got, err = uncoercedCounter(nil, mp); err != nil || got == nil {
		t.Errorf("no writer: uncoercedCounter = (%v, %v), want a counter of its own", got, err)
	}
	if got, err = uncoercedCounter(nil, nil); err != nil || got != nil {
		t.Errorf("no writer, no meter provider: uncoercedCounter = (%v, %v), want (nil, nil)", got, err)
	}
}

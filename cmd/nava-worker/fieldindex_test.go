package main

import (
	"slices"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/fieldindex"
	"github.com/Navaladi2019/TicketingSystem/internal/fieldindex/backfill"
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

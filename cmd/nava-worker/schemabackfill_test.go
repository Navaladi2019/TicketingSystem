package main

import (
	"slices"
	"testing"
	"time"

	"github.com/Navaladi2019/TicketingSystem/internal/schemabackfill"
)

// The worker registers schema.backfill (TICK-920) with workflows on, and its
// six activities on every build, on the worker that polls schema-backfill.
func TestRegisterSchemaBackfill(t *testing.T) {
	acts := []string{schemabackfill.AdmitActivity, schemabackfill.BarrierActivity, schemabackfill.FillActivity,
		schemabackfill.PageActivity, schemabackfill.EndActivity, schemabackfill.R6Activity}
	r := &recordingRegistry{}
	registerSchemaBackfill(r, &schemabackfill.Activities{}, &schemabackfill.Workflow{}, true)
	if want := []string{schemabackfill.WorkflowType}; !slices.Equal(r.workflows, want) {
		t.Errorf("workflows %v, want %v", r.workflows, want)
	}
	if !slices.Equal(r.activities, acts) {
		t.Errorf("activities %v, want %v", r.activities, acts)
	}
	r = &recordingRegistry{}
	registerSchemaBackfill(r, &schemabackfill.Activities{}, &schemabackfill.Workflow{}, false)
	if len(r.workflows) != 0 || !slices.Equal(r.activities, acts) {
		t.Errorf("%s=off: workflows %v activities %v, want none and %v", WorkflowsEnv, r.workflows, r.activities, acts)
	}
}

func TestSchemaBackfillConfig(t *testing.T) {
	env := func(m map[string]string) func(string) (string, bool) {
		return func(k string) (string, bool) { v, ok := m[k]; return v, ok }
	}
	s, err := schemaBackfillConfig(env(nil))
	if err != nil || s.concurrency != 4 || s.r6After != 7*24*time.Hour {
		t.Fatalf("defaults = %+v, %v; want 4 and 168h", s, err)
	}
	s, err = schemaBackfillConfig(env(map[string]string{SchemaBackfillConcurrencyEnv: "1", SchemaBackfillR6AfterEnv: "5s"}))
	if err != nil || s.concurrency != 1 || s.r6After != 5*time.Second {
		t.Fatalf("set = %+v, %v; want 1 and 5s", s, err)
	}
	for _, bad := range []map[string]string{
		{SchemaBackfillConcurrencyEnv: "0"}, {SchemaBackfillConcurrencyEnv: "x"}, {SchemaBackfillConcurrencyEnv: "65"},
		{SchemaBackfillR6AfterEnv: "7d"}, {SchemaBackfillR6AfterEnv: "1ms"},
	} {
		if _, err := schemaBackfillConfig(env(bad)); err == nil {
			t.Errorf("%v accepted", bad)
		}
	}
}

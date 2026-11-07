package main

import (
	"log/slog"
	"slices"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/audit/seal"
	"github.com/Navaladi2019/TicketingSystem/internal/workerguard"
)

// TestRegisterSealer: the worker registers audit.seal and exactly A1 and A2
// under their fixed names, on every build, and with a workflow registered it
// polls workflow tasks.
func TestRegisterSealer(t *testing.T) {
	r := &recordingRegistry{}
	registerSealer(r, &seal.Activities{})
	if !slices.Equal(r.workflows, []string{seal.WorkflowType}) || seal.WorkflowType != "audit.seal" {
		t.Errorf("workflows %v, want [audit.seal]", r.workflows)
	}
	if want := []string{seal.RescheduleActivity, seal.SealActivity}; !slices.Equal(r.activities, want) {
		t.Errorf("activities %v, want %v", r.activities, want)
	}

	tel, _ := newTemporal(t)
	g, err := workerguard.New(slog.New(slog.DiscardHandler), tel.MeterProvider)
	if err != nil {
		t.Fatal(err)
	}
	if o := workerOptions(g, true, make(chan error, 1)); o.DisableWorkflowWorker {
		t.Error("the workflow poller is off with audit.seal registered")
	}
}

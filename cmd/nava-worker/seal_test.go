package main

import (
	"log/slog"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/audit/seal"
	"github.com/Navaladi2019/TicketingSystem/internal/workerguard"
)

// TestRegisterSealer: the worker registers audit.seal and exactly A1 and A2
// under their fixed names, on every build, and with a workflow registered it
// polls workflow tasks.
func TestRegisterSealer(t *testing.T) {
	r := &recordingRegistry{}
	registerSealer(r, &seal.Activities{}, true)
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

// Activity-only (WorkflowsEnv=off): A1 and A2 are registered, audit.seal is
// not, and the worker polls no workflow task.
func TestRegisterSealerActivitiesOnly(t *testing.T) {
	r := &recordingRegistry{}
	registerSealer(r, &seal.Activities{}, false)
	if len(r.workflows) != 0 {
		t.Errorf("workflows %v registered with %s=off", r.workflows, WorkflowsEnv)
	}
	if want := []string{seal.RescheduleActivity, seal.SealActivity}; !slices.Equal(r.activities, want) {
		t.Errorf("activities %v, want %v", r.activities, want)
	}
}

func TestWorkflowsMode(t *testing.T) {
	for v, want := range map[string]bool{"": true, "on": true, "off": false} {
		got, err := workflowsMode(func(string) (string, bool) { return v, v != "" })
		if err != nil || got != want {
			t.Errorf("%s=%q: (%v, %v), want (%v, nil)", WorkflowsEnv, v, got, err, want)
		}
	}
	for _, v := range []string{"false", "OFF", "0", "no"} {
		if _, err := workflowsMode(func(string) (string, bool) { return v, true }); err == nil {
			t.Errorf("%s=%q accepted", WorkflowsEnv, v)
		}
	}
}

// compose.yaml's nava-worker never sets WorkflowsEnv: the always-on worker
// runs audit.seal. Only the test overlay's nava-worker-activities turns it
// off, reusing the nava-worker image with the nava-worker identity.
func TestComposeWorkerWorkflowsOn(t *testing.T) {
	read := func(p string) string {
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		return strings.ReplaceAll(string(b), "\r\n", "\n")
	}
	if svc := serviceBlock(t, read("../../compose.yaml"), "nava-worker"); strings.Contains(svc, WorkflowsEnv) {
		t.Errorf("compose.yaml's nava-worker sets %s", WorkflowsEnv)
	}
	act := serviceBlock(t, read("../../deploy/compose/compose.test.yaml"), "nava-worker-activities")
	for _, want := range []string{
		WorkflowsEnv + `: "off"`, "image: nava/nava-worker:${NAVA_IMAGE_TAG:-dev}", "pull_policy: never",
		"profiles: [activityonly]", "nava-worker.pem", "postgres://nava_worker:", "/readyz",
	} {
		if !strings.Contains(act, want) {
			t.Errorf("nava-worker-activities has no %q:\n%s", want, act)
		}
	}
	if strings.Contains(act, "build:") {
		t.Error("nava-worker-activities builds an image: it must reuse nava-worker's")
	}
}

//go:build !interptest

package main

import (
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/internal/workerguard"
	"github.com/Navaladi2019/TicketingSystem/internal/workflow/recordports"
	"github.com/Navaladi2019/TicketingSystem/internal/workflowdefs"
)

// TestProductionBuildWiresTheDatabasePorts: the production worker registers
// the interpreter with the database DefinitionStore and the P3-43 record
// ports over the worker's pool, and the pipeline on a pool of its own.
func TestProductionBuildWiresTheDatabasePorts(t *testing.T) {
	pool, pipePool := &pgxpool.Pool{}, &pgxpool.Pool{}
	acts, err := interpreterActivities(func(string) (string, bool) { return "/fixtures", true }, pool, pipePool, true)
	if err != nil || acts == nil {
		t.Fatalf("interpreterActivities = (%v, %v), want the interpreter's ports", acts, err)
	}
	r, ok := acts.Definitions.(workflowdefs.Reader)
	if !ok || r.Pool != pool {
		t.Errorf("Definitions = %T, want workflowdefs.Reader over the worker's pool", acts.Definitions)
	}
	if rr, ok := acts.Records.(recordports.Reader); !ok || rr.Pool != pool {
		t.Errorf("Records = %T, want recordports.Reader over the worker's pool", acts.Records)
	}
	if w, ok := acts.Writer.(recordports.Writer); !ok || w.Pool != pool || w.Pipeline == nil {
		t.Fatalf("Writer = %T, want recordports.Writer over the worker's pool with a pipeline", acts.Writer)
	}
	if c, ok := acts.Causes.(workerguard.CauseTracker); !ok || c.Pool != pool {
		t.Errorf("Causes = %T, want workerguard.CauseTracker over the worker's pool", acts.Causes)
	}
	reg := &recordingRegistry{}
	registerInterpreter(reg, acts)
	if len(reg.workflows) == 0 || len(reg.activities) == 0 {
		t.Errorf("registered workflows %v activities %v, want the interpreter's", reg.workflows, reg.activities)
	}
}

func TestProductionBuildInterpreterNeedsAPool(t *testing.T) {
	if _, err := interpreterActivities(func(string) (string, bool) { return "", false }, nil, nil, true); err == nil {
		t.Error("interpreterActivities with no pool returned no error")
	}
}

// The activity-only worker (WorkflowsEnv=off) polls no workflow tasks, so it
// registers no interpreter and must still start: no ports, no error, with or
// without a pool.
func TestProductionBuildWithWorkflowsOffRegistersNoInterpreter(t *testing.T) {
	for _, pool := range []*pgxpool.Pool{nil, {}} {
		acts, err := interpreterActivities(func(string) (string, bool) { return "", false }, pool, pool, false)
		if err != nil || acts != nil {
			t.Fatalf("interpreterActivities(workflows off) = (%v, %v), want (nil, nil)", acts, err)
		}
	}
}

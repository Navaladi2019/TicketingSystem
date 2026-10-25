package workerguard

import (
	"testing"
	"time"

	"go.temporal.io/sdk/workflow"
)

// TICK-184: a workflow task descheduled for over the SDK's 1 s deadlock
// threshold on a loaded host failed with TMPRL1101 although it never blocked.
// The sleep stands in for that stall; it fails with the SDK default.
func TestHarnessToleratesStalledWorkflowTask(t *testing.T) {
	h := newHarness(t, idOf(tenantA))
	h.env.ExecuteWorkflow(func(workflow.Context) error {
		time.Sleep(1500 * time.Millisecond)
		return nil
	})
	if err := h.env.GetWorkflowError(); err != nil {
		t.Fatalf("workflow error %v, want nil", err)
	}
}

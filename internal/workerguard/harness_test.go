package workerguard

import (
	"testing"
	"testing/synctest"
	"time"

	"go.temporal.io/sdk/workflow"
)

// TICK-184: a workflow task descheduled for over the SDK's 1 s deadlock
// threshold on a loaded host failed with TMPRL1101 although it never blocked.
// The stall stands in for that; it fails with the SDK default. In a synctest
// bubble the stall and the SDK's deadlock timer share the fake clock, so the
// test proves the threshold without spending 1.5 s of wall time.
func TestHarnessToleratesStalledWorkflowTask(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := newHarness(t, idOf(tenantA))
		h.env.ExecuteWorkflow(func(workflow.Context) error {
			<-time.After(1500 * time.Millisecond) // a stalled task, not a workflow timer
			return nil
		})
		if err := h.env.GetWorkflowError(); err != nil {
			t.Fatalf("workflow error %v, want nil", err)
		}
	})
}

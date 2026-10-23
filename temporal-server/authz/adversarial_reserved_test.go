package authz

// Adversarial test for the GO-2026-5766 bug class (TICK-115). Temporal's own
// system workflows (batch, scheduler, delete-namespace...) run on reserved
// `temporal-sys-*` task queues and are processed by the per-namespace worker,
// which calls the frontend as `temporal-system` (all APIs, any namespace). In
// 1.29.0-1.29.4 the batch activity trusted a namespace name from its input, so
// anyone who could start that workflow in `nava` could act in another namespace.
// 1.29.5 fixes that one activity. Inside `nava` the path stays open on every
// version: a relay start of `temporal-sys-batch-workflow` on the per-ns queue
// with BatchParams{Namespace: "nava", Executions: [t:B:...]} terminated tenant
// B's workflow on stock 1.28.4 and 1.29.7 (TICK-115 spike). This test pins the
// wall one layer down: no platform identity may start or signal-with-start a
// workflow on a reserved task queue or with a reserved workflow type, whatever
// the server version.

import (
	"testing"

	commonpb "go.temporal.io/api/common/v1"
	taskqueuepb "go.temporal.io/api/taskqueue/v1"
	"go.temporal.io/api/workflowservice/v1"
)

func TestAdversarialReservedSystemWorkflowStartDenied(t *testing.T) {
	cases := map[string]struct {
		wfType, taskQueue string
	}{
		"batch workflow on per-ns queue":  {"temporal-sys-batch-workflow", "temporal-sys-per-ns-tq"},
		"own type on per-ns queue":        {"TicketWorkflow", "temporal-sys-per-ns-tq"},
		"batch workflow on own queue":     {"temporal-sys-batch-workflow", "nava-tq"},
		"scheduler workflow on per-ns":    {"temporal-sys-scheduler-workflow", "temporal-sys-per-ns-tq"},
		"deployment workflow on per-ns":   {"temporal-sys-worker-deployment-workflow", "temporal-sys-per-ns-tq"},
		"delete-namespace activity queue": {"TicketWorkflow", "temporal-sys-delete-namespace-activity-tq"},
		"default system worker queue":     {"TicketWorkflow", "default-worker-tq"},
	}
	for name, c := range cases {
		t.Run(name+"/start", func(t *testing.T) {
			called, err := callUnary(t, peerCtx(string(Relay), true), wf+"StartWorkflowExecution",
				&workflowservice.StartWorkflowExecutionRequest{
					Namespace:    Namespace,
					WorkflowId:   "t:tenantA:x",
					WorkflowType: &commonpb.WorkflowType{Name: c.wfType},
					TaskQueue:    &taskqueuepb.TaskQueue{Name: c.taskQueue},
				})
			requireDenied(t, called, err)
		})
		t.Run(name+"/signal-with-start", func(t *testing.T) {
			called, err := callUnary(t, peerCtx(string(Relay), true), wf+"SignalWithStartWorkflowExecution",
				&workflowservice.SignalWithStartWorkflowExecutionRequest{
					Namespace:    Namespace,
					WorkflowId:   "t:tenantA:x",
					SignalName:   "s",
					WorkflowType: &commonpb.WorkflowType{Name: c.wfType},
					TaskQueue:    &taskqueuepb.TaskQueue{Name: c.taskQueue},
				})
			requireDenied(t, called, err)
		})
	}

	// Control: an ordinary platform start still passes.
	called, err := callUnary(t, peerCtx(string(Relay), true), wf+"StartWorkflowExecution",
		&workflowservice.StartWorkflowExecutionRequest{
			Namespace:    Namespace,
			WorkflowId:   "t:tenantA:x",
			WorkflowType: &commonpb.WorkflowType{Name: platformType()},
			TaskQueue:    &taskqueuepb.TaskQueue{Name: platformQueue()},
		})
	if err != nil || !called {
		t.Fatalf("ordinary relay start denied: err=%v called=%v", err, called)
	}
}

// A compromised worker certificate must not poll the reserved queues either:
// it would receive the system worker's batch activity tasks for `nava`.
func TestAdversarialWorkerReservedTaskQueuePollDenied(t *testing.T) {
	for _, tq := range []string{"temporal-sys-per-ns-tq", "default-worker-tq"} {
		called, err := callUnary(t, peerCtx(string(Worker), true), wf+"PollActivityTaskQueue",
			&workflowservice.PollActivityTaskQueueRequest{Namespace: Namespace, TaskQueue: &taskqueuepb.TaskQueue{Name: tq}})
		if err == nil && called {
			t.Errorf("worker poll on reserved queue %q allowed", tq)
		}
		called, err = callUnary(t, peerCtx(string(Worker), true), wf+"PollWorkflowTaskQueue",
			&workflowservice.PollWorkflowTaskQueueRequest{Namespace: Namespace, TaskQueue: &taskqueuepb.TaskQueue{Name: tq}})
		if err == nil && called {
			t.Errorf("worker workflow-task poll on reserved queue %q allowed", tq)
		}
	}
}

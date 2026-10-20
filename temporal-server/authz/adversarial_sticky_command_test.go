package authz

// Adversarial tests for RFC-002 rev 1 D9b/D9c (TICK-115 re-check). A stolen
// nava-worker certificate must not reach Temporal's system worker through the
// workflow-task command channel or through a sticky task queue.
//
// Sticky queues: at server v1.29.7 a sticky partition's identity is its sticky
// name only. tqid.StickyPartition.Key() is {namespace, stickyName, 0, type},
// the same key as the normal root partition of a queue with that name, and
// PhysicalTaskQueueKey.PersistenceName() is the bare name for both
// (common/tqid/task_queue_id.go: "task queue kind (sticky vs normal) and normal
// name for sticky task queues are not part of the task queue partition
// identity"). So a STICKY poll named `temporal-sys-per-ns-tq` loads the same
// partition manager as the system worker's NORMAL queue and receives its tasks,
// whatever NormalName says. Checking NormalName alone does not close D9b.

import (
	"testing"

	commandpb "go.temporal.io/api/command/v1"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	taskqueuepb "go.temporal.io/api/taskqueue/v1"
	"go.temporal.io/api/workflowservice/v1"
)

var reservedQueues = []string{"temporal-sys-per-ns-tq", "default-worker-tq", "/_sys/temporal-sys-per-ns-tq/1"}

func TestAdversarialWorkerStickyPollOnReservedNameDenied(t *testing.T) {
	ctx := peerCtx(string(Worker), true)
	for _, name := range reservedQueues {
		t.Run(name, func(t *testing.T) {
			called, err := callUnary(t, ctx, wf+"PollWorkflowTaskQueue", &workflowservice.PollWorkflowTaskQueueRequest{
				Namespace: Namespace,
				TaskQueue: &taskqueuepb.TaskQueue{Name: name, Kind: enumspb.TASK_QUEUE_KIND_STICKY, NormalName: "nava-tq"},
			})
			requireDenied(t, called, err)
		})
	}
	// Activity polls have no sticky form; a STICKY activity poll is denied outright.
	called, err := callUnary(t, ctx, wf+"PollActivityTaskQueue", &workflowservice.PollActivityTaskQueueRequest{
		Namespace: Namespace,
		TaskQueue: &taskqueuepb.TaskQueue{Name: "temporal-sys-per-ns-tq", Kind: enumspb.TASK_QUEUE_KIND_STICKY, NormalName: "nava-tq"},
	})
	requireDenied(t, called, err)

	// Control: the SDK's own sticky name (<host>:<uuid>, sdk v1.49.0
	// internal_utils.go getWorkerTaskQueue) passes.
	called, err = callUnary(t, ctx, wf+"PollWorkflowTaskQueue", &workflowservice.PollWorkflowTaskQueueRequest{
		Namespace: Namespace,
		TaskQueue: &taskqueuepb.TaskQueue{Name: "worker-0:6f1c2a7e-3b4d-4c5e-8f90-a1b2c3d4e5f6",
			Kind: enumspb.TASK_QUEUE_KIND_STICKY, NormalName: "nava-tq"},
	})
	if err != nil || !called {
		t.Fatalf("SDK-shaped sticky poll denied: err=%v called=%v", err, called)
	}
}

// StickyAttributes naming a reserved queue would push the tenant workflow's next
// task onto the system worker's partition.
func TestAdversarialWorkerStickyAttributesOnReservedNameDenied(t *testing.T) {
	tok := taskToken(t, "t:tenantA:wf")
	for _, name := range reservedQueues {
		t.Run(name, func(t *testing.T) {
			req := respond(tok)
			req.StickyAttributes = &taskqueuepb.StickyExecutionAttributes{
				WorkerTaskQueue: &taskqueuepb.TaskQueue{Name: name, Kind: enumspb.TASK_QUEUE_KIND_STICKY, NormalName: "nava-tq"},
			}
			called, err := callUnary(t, peerCtx(string(Worker), true), respondWorkflowTaskCompleted, req)
			requireDenied(t, called, err)
		})
	}
}

// D9c: a tenant's own workflow task may not hand work to the system worker by
// activity queue, child type/queue or continue-as-new type/queue. Every command
// carries a valid tenantA header, so only the D9c check can deny it.
func TestAdversarialWorkerCommandToReservedWorkloadDenied(t *testing.T) {
	hdr := tenantHeader(jsonString("tenantA"))
	tq := func(n string) *taskqueuepb.TaskQueue { return &taskqueuepb.TaskQueue{Name: n} }
	wt := func(n string) *commonpb.WorkflowType { return &commonpb.WorkflowType{Name: n} }
	cmds := map[string]*commandpb.Command{
		"activity on per-ns queue": {
			CommandType: enumspb.COMMAND_TYPE_SCHEDULE_ACTIVITY_TASK,
			Attributes: &commandpb.Command_ScheduleActivityTaskCommandAttributes{ScheduleActivityTaskCommandAttributes: &commandpb.ScheduleActivityTaskCommandAttributes{
				ActivityId: "1", ActivityType: &commonpb.ActivityType{Name: "temporal-sys-batch-activity"},
				TaskQueue: tq("temporal-sys-per-ns-tq"), Header: hdr}},
		},
		"child batch workflow": {
			CommandType: enumspb.COMMAND_TYPE_START_CHILD_WORKFLOW_EXECUTION,
			Attributes: &commandpb.Command_StartChildWorkflowExecutionCommandAttributes{StartChildWorkflowExecutionCommandAttributes: &commandpb.StartChildWorkflowExecutionCommandAttributes{
				Namespace: Namespace, WorkflowId: "t:tenantA:child", WorkflowType: wt("temporal-sys-batch-workflow"),
				TaskQueue: tq("temporal-sys-per-ns-tq"), Header: hdr}},
		},
		"child on default-worker-tq": {
			CommandType: enumspb.COMMAND_TYPE_START_CHILD_WORKFLOW_EXECUTION,
			Attributes: &commandpb.Command_StartChildWorkflowExecutionCommandAttributes{StartChildWorkflowExecutionCommandAttributes: &commandpb.StartChildWorkflowExecutionCommandAttributes{
				Namespace: Namespace, WorkflowId: "t:tenantA:child", WorkflowType: wt("nava.smoke.Echo"),
				TaskQueue: tq("default-worker-tq"), Header: hdr}},
		},
		"continue-as-new as batch workflow": {
			CommandType: enumspb.COMMAND_TYPE_CONTINUE_AS_NEW_WORKFLOW_EXECUTION,
			Attributes: &commandpb.Command_ContinueAsNewWorkflowExecutionCommandAttributes{ContinueAsNewWorkflowExecutionCommandAttributes: &commandpb.ContinueAsNewWorkflowExecutionCommandAttributes{
				WorkflowType: wt("temporal-sys-batch-workflow"), TaskQueue: tq("temporal-sys-per-ns-tq"), Header: hdr}},
		},
	}
	for name, cmd := range cmds {
		t.Run(name, func(t *testing.T) {
			called, err := callUnary(t, peerCtx(string(Worker), true), respondWorkflowTaskCompleted,
				respond(taskToken(t, "t:tenantA:wf"), cmd))
			requireDenied(t, called, err)
		})
	}
}

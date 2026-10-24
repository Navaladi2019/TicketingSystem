package authz

// D9 additions of ADR-001-A2 (D9b+, D9d with its query clause, D9e, D9f, D9g):
// the allow paths the platform's own SDK takes, and the deny reasons and alert
// shape. The F2 denials themselves are in adversarial_f2*_test.go.

import (
	"testing"

	commonpb "go.temporal.io/api/common/v1"
	deploymentpb "go.temporal.io/api/deployment/v1"
	enumspb "go.temporal.io/api/enums/v1"
	taskqueuepb "go.temporal.io/api/taskqueue/v1"
	workerpb "go.temporal.io/api/worker/v1"
	"go.temporal.io/api/workflowservice/v1"
)

// D9d allow path: each task-token API passes with a token for a tenant's
// workflow, and denies (with an alert) an undecodable or unprefixed token.
func TestTaskTokenAPIs(t *testing.T) {
	for api, req := range activityTokenRequests(taskToken(t, wfA)) {
		t.Run(api+" allowed", func(t *testing.T) {
			requireWorkloadAllow(t, Worker, wf+api, req)
		})
	}
	for api, req := range activityTokenRequests(taskToken(t, deploymentID)) {
		t.Run(api+" unprefixed", func(t *testing.T) {
			requireWorkloadDeny(t, Worker, wf+api, req, ReasonNoTenantPrefix)
		})
	}
	for api, req := range activityTokenRequests([]byte("not a token")) {
		t.Run(api+" undecodable", func(t *testing.T) {
			requireWorkloadDeny(t, Worker, wf+api, req, ReasonTaskToken)
		})
	}
}

// D9d query clause: a query token that doesn't decode is denied.
func TestQueryTaskToken(t *testing.T) {
	api := wf + "RespondQueryTaskCompleted"
	for _, tq := range []string{platformQueue(), sdkSticky} {
		requireWorkloadAllow(t, Worker, api,
			&workflowservice.RespondQueryTaskCompletedRequest{Namespace: Namespace, TaskToken: queryToken(t, tq)})
	}
	requireWorkloadDeny(t, Worker, api,
		&workflowservice.RespondQueryTaskCompletedRequest{Namespace: Namespace, TaskToken: []byte("not a token")}, ReasonTaskToken)
	for _, tq := range []string{"", perNSQueue, "nava-tq:not-a-uuid"} {
		requireWorkloadDeny(t, Worker, api,
			&workflowservice.RespondQueryTaskCompletedRequest{Namespace: Namespace, TaskToken: queryToken(t, tq)}, ReasonTaskQueue)
	}
}

// sdkPolls is what sdk-go v1.49.0 sends with default worker.Options (no
// versioning, no deployment version): internal_task_pollers.go
// getNextPollRequest and activityTaskPoller.poll. The build id defaults to the
// binary checksum. If deployment is set without versioning, the SDK sends
// UNVERSIONED options.
func sdkPolls(deployment *deploymentpb.WorkerDeploymentOptions) map[string]any {
	const checksum = "0d2a5f3c9e1b7a4d6c8e0f2a4b6c8d0e"
	//nolint:staticcheck // SA1019: the SDK still sends the deprecated fields
	caps := &commonpb.WorkerVersionCapabilities{BuildId: checksum}
	return map[string]any{
		"workflow normal": &workflowservice.PollWorkflowTaskQueueRequest{Namespace: Namespace, Identity: "1@worker-0@",
			TaskQueue:      &taskqueuepb.TaskQueue{Name: platformQueue(), Kind: enumspb.TASK_QUEUE_KIND_NORMAL},
			BinaryChecksum: checksum, WorkerVersionCapabilities: caps, DeploymentOptions: deployment}, //nolint:staticcheck // SA1019
		"workflow sticky": &workflowservice.PollWorkflowTaskQueueRequest{Namespace: Namespace, Identity: "1@worker-0@",
			TaskQueue:      tqOf(sdkSticky, enumspb.TASK_QUEUE_KIND_STICKY, platformQueue()),
			BinaryChecksum: checksum, WorkerVersionCapabilities: caps, DeploymentOptions: deployment}, //nolint:staticcheck // SA1019
		"activity": &workflowservice.PollActivityTaskQueueRequest{Namespace: Namespace, Identity: "1@worker-0@",
			TaskQueue:                 &taskqueuepb.TaskQueue{Name: platformQueue(), Kind: enumspb.TASK_QUEUE_KIND_NORMAL},
			TaskQueueMetadata:         &taskqueuepb.TaskQueueMetadata{},
			WorkerVersionCapabilities: caps, DeploymentOptions: deployment}, //nolint:staticcheck // SA1019
	}
}

func pollAPI(req any) string {
	if _, ok := req.(*workflowservice.PollActivityTaskQueueRequest); ok {
		return wf + "PollActivityTaskQueue"
	}
	return wf + "PollWorkflowTaskQueue"
}

// D9b+ allow path: the exact poll sdk-go v1.49.0 sends passes; any versioned
// poll is denied with ReasonWorkerVersioning.
func TestPollVersioning(t *testing.T) {
	unversioned := &deploymentpb.WorkerDeploymentOptions{DeploymentName: "nava-worker", BuildId: "build-1",
		WorkerVersioningMode: enumspb.WORKER_VERSIONING_MODE_UNVERSIONED}
	for _, opts := range []*deploymentpb.WorkerDeploymentOptions{nil, unversioned} {
		for name, req := range sdkPolls(opts) {
			t.Run("sdk "+name, func(t *testing.T) {
				requireWorkloadAllow(t, Worker, pollAPI(req), req)
			})
		}
	}

	for _, mode := range []enumspb.WorkerVersioningMode{enumspb.WORKER_VERSIONING_MODE_VERSIONED, enumspb.WorkerVersioningMode(99)} {
		for name, req := range sdkPolls(&deploymentpb.WorkerDeploymentOptions{DeploymentName: "nava-worker",
			BuildId: "build-1", WorkerVersioningMode: mode}) {
			t.Run(mode.String()+" "+name, func(t *testing.T) {
				requireWorkloadDeny(t, Worker, pollAPI(req), req, ReasonWorkerVersioning)
			})
		}
	}
	for name, req := range sdkPolls(nil) {
		switch r := req.(type) {
		case *workflowservice.PollWorkflowTaskQueueRequest:
			r.WorkerVersionCapabilities.UseVersioning = true //nolint:staticcheck // SA1019
		case *workflowservice.PollActivityTaskQueueRequest:
			r.WorkerVersionCapabilities.UseVersioning = true //nolint:staticcheck // SA1019
		}
		t.Run("use versioning "+name, func(t *testing.T) {
			requireWorkloadDeny(t, Worker, pollAPI(req), req, ReasonWorkerVersioning)
		})
	}
}

// D9g: an absent heartbeat is allowed, a present one with an empty TaskQueue
// is denied; a non-SDK sticky name is a sticky deny.
func TestWorkerShutdownAndHeartbeat(t *testing.T) {
	shutdown, record := wf+"ShutdownWorker", wf+"RecordWorkerHeartbeat"
	requireWorkloadAllow(t, Worker, shutdown, &workflowservice.ShutdownWorkerRequest{Namespace: Namespace, StickyTaskQueue: sdkSticky})
	requireWorkloadAllow(t, Worker, record, &workflowservice.RecordWorkerHeartbeatRequest{Namespace: Namespace})
	requireWorkloadAllow(t, Worker, record, &workflowservice.RecordWorkerHeartbeatRequest{Namespace: Namespace,
		WorkerHeartbeat: []*workerpb.WorkerHeartbeat{{TaskQueue: platformQueue()}, {TaskQueue: platformQueue()}}})

	requireWorkloadDeny(t, Worker, shutdown, &workflowservice.ShutdownWorkerRequest{Namespace: Namespace,
		WorkerHeartbeat: &workerpb.WorkerHeartbeat{}}, ReasonTaskQueue)
	requireWorkloadDeny(t, Worker, record, &workflowservice.RecordWorkerHeartbeatRequest{Namespace: Namespace,
		WorkerHeartbeat: []*workerpb.WorkerHeartbeat{{TaskQueue: platformQueue()}, {}}}, ReasonTaskQueue)
	requireWorkloadDeny(t, Worker, shutdown, &workflowservice.ShutdownWorkerRequest{Namespace: Namespace,
		StickyTaskQueue: platformQueue()}, ReasonStickyQueue)
}

// D9e and D9f deny with the tenant-prefix reason and an alert; nava-operator
// stays outside D9 and still reads a system workflow's history.
func TestWorkflowIDTenantPrefixReason(t *testing.T) {
	for api, req := range coreWorkflowIDRequests(schedulerID) {
		t.Run("core "+api, func(t *testing.T) {
			requireWorkloadDeny(t, Core, wf+api, req, ReasonNoTenantPrefix)
		})
	}
	history := &workflowservice.GetWorkflowExecutionHistoryRequest{Namespace: Namespace,
		Execution: &commonpb.WorkflowExecution{WorkflowId: schedulerID}}
	requireWorkloadDeny(t, Worker, wf+"GetWorkflowExecutionHistory", history, ReasonNoTenantPrefix)
	requireWorkloadAllow(t, Operator, wf+"GetWorkflowExecutionHistory", history)
}

// A D9 check given a request of the wrong type denies it as malformed.
func TestA2ChecksDenyWrongRequestType(t *testing.T) {
	for id, apis := range workloadChecks {
		for api := range apis {
			requireWorkloadDeny(t, id, api, &workflowservice.GetSystemInfoRequest{}, ReasonMalformedRequest)
		}
	}
}

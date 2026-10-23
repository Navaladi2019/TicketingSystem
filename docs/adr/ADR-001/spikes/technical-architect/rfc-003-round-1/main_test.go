package spike

import (
	"context"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/temporal-server/authz"
	commonpb "go.temporal.io/api/common/v1"
	deploymentpb "go.temporal.io/api/deployment/v1"
	enumspb "go.temporal.io/api/enums/v1"
	taskqueuepb "go.temporal.io/api/taskqueue/v1"
	workerpb "go.temporal.io/api/worker/v1"
	"go.temporal.io/api/workflowservice/v1"
	taskqueuespb "go.temporal.io/server/api/taskqueue/v1"
	tokenspb "go.temporal.io/server/api/token/v1"
	"go.temporal.io/server/common/authorization"
	"go.temporal.io/server/common/log"
	"go.temporal.io/server/common/metrics"
	"go.temporal.io/server/common/tqid"
	"go.temporal.io/server/common/worker_versioning"
)

const wf = "/temporal.api.workflowservice.v1.WorkflowService/"

func TestWorkerReach(t *testing.T) {
	a := authz.NewAuthorizer(log.NewNoopLogger(), metrics.NoopMetricsHandler)
	worker := &authorization.Claims{Subject: "nava-worker"}
	tok := func(wid string) []byte { b, _ := (&tokenspb.Task{WorkflowId: wid, NamespaceId: "x"}).Marshal(); return b }
	qtok, _ := (&tokenspb.QueryTask{NamespaceId: "x", TaskQueue: "temporal-sys-per-ns-tq", TaskId: "id"}).Marshal()
	cases := []struct {
		name, api string
		req       any
	}{
		{"ShutdownWorker sticky=nava-tq", "ShutdownWorker", &workflowservice.ShutdownWorkerRequest{Namespace: "nava", StickyTaskQueue: "nava-tq"}},
		{"ShutdownWorker sticky=temporal-sys-per-ns-tq", "ShutdownWorker", &workflowservice.ShutdownWorkerRequest{Namespace: "nava", StickyTaskQueue: "temporal-sys-per-ns-tq"}},
		{"ShutdownWorker heartbeat tq=temporal-sys-x", "ShutdownWorker", &workflowservice.ShutdownWorkerRequest{Namespace: "nava", WorkerHeartbeat: &workerpb.WorkerHeartbeat{TaskQueue: "temporal-sys-x"}}},
		{"RecordWorkerHeartbeat tq=temporal-sys-x", "RecordWorkerHeartbeat", &workflowservice.RecordWorkerHeartbeatRequest{Namespace: "nava", WorkerHeartbeat: []*workerpb.WorkerHeartbeat{{TaskQueue: "temporal-sys-x"}}}},
		{"F2-5 GetWorkflowExecutionHistory temporal-sys-scheduler:x", "GetWorkflowExecutionHistory", &workflowservice.GetWorkflowExecutionHistoryRequest{Namespace: "nava", Execution: &commonpb.WorkflowExecution{WorkflowId: "temporal-sys-scheduler:x"}}},
		{"RespondQueryTaskCompleted token tq=temporal-sys-per-ns-tq", "RespondQueryTaskCompleted", &workflowservice.RespondQueryTaskCompletedRequest{Namespace: "nava", TaskToken: qtok}},
		{"F2-3 RespondActivityTaskCompleted token wid=temporal-sys-x", "RespondActivityTaskCompleted", &workflowservice.RespondActivityTaskCompletedRequest{Namespace: "nava", TaskToken: tok("temporal-sys-x")}},
		{"F2-1 PollWorkflowTaskQueue VERSIONED", "PollWorkflowTaskQueue", &workflowservice.PollWorkflowTaskQueueRequest{Namespace: "nava", TaskQueue: &taskqueuepb.TaskQueue{Name: "nava-tq"}, DeploymentOptions: &deploymentpb.WorkerDeploymentOptions{DeploymentName: "d", BuildId: "b", WorkerVersioningMode: enumspb.WORKER_VERSIONING_MODE_VERSIONED}}},
		{"control: ShutdownWorker other namespace", "ShutdownWorker", &workflowservice.ShutdownWorkerRequest{Namespace: "temporal-system", StickyTaskQueue: "nava-tq"}},
	}
	for _, c := range cases {
		r, err := a.Authorize(context.Background(), worker, &authorization.CallTarget{APIName: wf + c.api, Namespace: namespaceOf(c.req), Request: c.req})
		t.Logf("%-62s -> %v err=%v", c.name, decision(r), err)
	}
	p := tqid.PartitionFromPartitionProto(&taskqueuespb.TaskQueuePartition{TaskQueue: "nava-tq", TaskQueueType: enumspb.TASK_QUEUE_TYPE_WORKFLOW}, "ns")
	root := tqid.UnsafeTaskQueueFamily("ns", "nava-tq").TaskQueue(enumspb.TASK_QUEUE_TYPE_WORKFLOW).RootPartition()
	t.Logf("unloaded partition key=%v  nava-tq root WORKFLOW key=%v  equal=%v", p.Key(), root.Key(), p.Key() == root.Key())
	d0, err := worker_versioning.DeploymentFromCapabilities(&commonpb.WorkerVersionCapabilities{BuildId: "abc", UseVersioning: false}, nil)
	t.Logf("SDK default caps -> deployment=%v err=%v", d0, err)
	d1, err := worker_versioning.DeploymentFromCapabilities(&commonpb.WorkerVersionCapabilities{BuildId: "b", UseVersioning: true, DeploymentSeriesName: "s"}, nil)
	t.Logf("UseVersioning+series+build -> deployment=%v err=%v", d1, err)
	d2, err := worker_versioning.DeploymentFromCapabilities(nil, &deploymentpb.WorkerDeploymentOptions{DeploymentName: "d", BuildId: "b", WorkerVersioningMode: enumspb.WORKER_VERSIONING_MODE_UNVERSIONED})
	t.Logf("DeploymentOptions UNVERSIONED -> deployment=%v err=%v", d2, err)
}

func namespaceOf(req any) string {
	if n, ok := req.(interface{ GetNamespace() string }); ok {
		return n.GetNamespace()
	}
	return ""
}

func decision(r authorization.Result) string {
	if r.Decision == authorization.DecisionAllow {
		return "ALLOW"
	}
	return "DENY"
}

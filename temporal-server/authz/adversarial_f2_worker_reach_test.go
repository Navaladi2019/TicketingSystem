package authz

// Adversarial tests for the nava-worker reach items raised in RFC-003 round 1
// (TICK-129): F2-5 (history reads) and F2-6 (ShutdownWorker task queue). Same
// convention as adversarial_f2_test.go: a skip names an open finding, and the
// assertion below it is the fix's acceptance test.

import (
	"testing"

	commonpb "go.temporal.io/api/common/v1"
	workerpb "go.temporal.io/api/worker/v1"
	"go.temporal.io/api/workflowservice/v1"
	tokenspb "go.temporal.io/server/api/token/v1"
)

// sdkStickyQueue is the shape sdk-go v1.49.0 gives its own sticky queue.
const sdkStickyQueue = "worker-host:2f1c7a3e-9b4d-4e61-8a2f-0c5d7e9b1a34"

// F2-5: nava-worker GetWorkflowExecutionHistory has no workload check, so a
// worker certificate reads the history of Temporal's own workflows in `nava`.
func TestAdversarialF2WorkerHistoryTenantPrefix(t *testing.T) {
	worker := peerCtx(string(Worker), true)
	history := func(id string) any {
		return &workflowservice.GetWorkflowExecutionHistoryRequest{Namespace: Namespace,
			Execution: &commonpb.WorkflowExecution{WorkflowId: id}}
	}
	called, err := callUnary(t, worker, wf+"GetWorkflowExecutionHistory", history(wfA))
	requireAllowed(t, called, err)

	for _, id := range append(systemWorkflowIDs(), "", "t:", "t::x", "tenantA:x") {
		called, err := callUnary(t, worker, wf+"GetWorkflowExecutionHistory", history(id))
		if err == nil || called {
			t.Errorf("nava-worker history read of %q allowed", id)
		}
	}
}

// F2-6: ShutdownWorker makes the frontend force-unload the WORKFLOW partition
// named by StickyTaskQueue, with no partition id, which is the root partition
// of a normal queue of that name (v1.29.7 workflow_handler.go ShutdownWorker,
// common/tqid). A worker certificate can unload `nava-tq`'s root partition for
// every tenant, or a Temporal per-ns queue. The heartbeat it carries names a
// task queue too.
func TestAdversarialF2WorkerShutdownTaskQueue(t *testing.T) {
	worker := peerCtx(string(Worker), true)
	// Control: what sdk-go v1.49.0 sends for a workflow worker and for an
	// activity worker.
	for _, sticky := range []string{sdkStickyQueue, ""} {
		called, err := callUnary(t, worker, wf+"ShutdownWorker",
			&workflowservice.ShutdownWorkerRequest{Namespace: Namespace, StickyTaskQueue: sticky,
				WorkerHeartbeat: &workerpb.WorkerHeartbeat{TaskQueue: platformQueue()}})
		requireAllowed(t, called, err)
	}

	for _, sticky := range []string{platformQueue(), perNSQueue, "temporal-sys-x", "nava-tq:not-a-uuid"} {
		called, err := callUnary(t, worker, wf+"ShutdownWorker",
			&workflowservice.ShutdownWorkerRequest{Namespace: Namespace, StickyTaskQueue: sticky})
		if err == nil || called {
			t.Errorf("ShutdownWorker with StickyTaskQueue %q allowed", sticky)
		}
	}
	hb := &workerpb.WorkerHeartbeat{TaskQueue: perNSQueue}
	called, err := callUnary(t, worker, wf+"ShutdownWorker",
		&workflowservice.ShutdownWorkerRequest{Namespace: Namespace, WorkerHeartbeat: hb})
	if err == nil || called {
		t.Errorf("ShutdownWorker with heartbeat queue %q allowed", perNSQueue)
	}
	called, err = callUnary(t, worker, wf+"RecordWorkerHeartbeat",
		&workflowservice.RecordWorkerHeartbeatRequest{Namespace: Namespace, WorkerHeartbeat: []*workerpb.WorkerHeartbeat{hb}})
	if err == nil || called {
		t.Errorf("RecordWorkerHeartbeat with queue %q allowed", perNSQueue)
	}
}

// queryToken is a token.v1.QueryTask for a query task on tq.
func queryToken(t testing.TB, tq string) []byte {
	t.Helper()
	b, err := (&tokenspb.QueryTask{NamespaceId: "ns-id", TaskQueue: tq, TaskId: "id"}).Marshal()
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// RespondQueryTaskCompleted's token names a task queue (TA round-1 item 8).
func TestAdversarialF2WorkerQueryTokenTaskQueue(t *testing.T) {
	worker := peerCtx(string(Worker), true)
	for _, tq := range []string{platformQueue(), sdkStickyQueue} {
		called, err := callUnary(t, worker, wf+"RespondQueryTaskCompleted",
			&workflowservice.RespondQueryTaskCompletedRequest{Namespace: Namespace, TaskToken: queryToken(t, tq)})
		requireAllowed(t, called, err)
	}

	called, err := callUnary(t, worker, wf+"RespondQueryTaskCompleted",
		&workflowservice.RespondQueryTaskCompletedRequest{Namespace: Namespace, TaskToken: queryToken(t, perNSQueue)})
	if err == nil || called {
		t.Errorf("RespondQueryTaskCompleted with a %q token allowed", perNSQueue)
	}
}

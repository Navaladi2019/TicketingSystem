package authz

// Adversarial tests for ADR-001-A1 follow-up F2 (TICK-122), one per item, run
// against the v1.29.7 Authorizer through Temporal's own interceptor. A passing
// test closes its item. A skipped one is an open finding: the skip names the
// finding id, and the assertion below it is the fix's acceptance test, so
// removing the skip is the last step of the fix.
//
// Temporal's own workflows in `nava` have ids with no tenant prefix:
// `temporal-sys-scheduler:<schedule>`, `temporal-sys-worker-deployment:<name>`
// and `temporal-sys-worker-deployment-version:<version>` (server v1.29.7
// service/worker/scheduler/workflow.go, common/worker_versioning).

import (
	"fmt"
	"testing"

	commandpb "go.temporal.io/api/command/v1"
	commonpb "go.temporal.io/api/common/v1"
	deploymentpb "go.temporal.io/api/deployment/v1"
	enumspb "go.temporal.io/api/enums/v1"
	querypb "go.temporal.io/api/query/v1"
	taskqueuepb "go.temporal.io/api/taskqueue/v1"
	updatepb "go.temporal.io/api/update/v1"
	"go.temporal.io/api/workflowservice/v1"
)

const (
	schedulerType        = "temporal-sys-scheduler-workflow"
	deploymentType       = "temporal-sys-worker-deployment-workflow"
	deploymentVersionTyp = "temporal-sys-worker-deployment-version-workflow"
	perNSQueue           = "temporal-sys-per-ns-tq"

	schedulerID         = "temporal-sys-scheduler:nightly"
	deploymentID        = "temporal-sys-worker-deployment:nava-worker"
	deploymentVersionID = "temporal-sys-worker-deployment-version:nava-worker.build-1"
)

func systemWorkflowIDs() []string { return []string{schedulerID, deploymentID, deploymentVersionID} }

func requireAllowed(t *testing.T, called bool, err error) {
	t.Helper()
	if err != nil || !called {
		t.Fatalf("call denied (err=%v, called=%v); want allowed", err, called)
	}
}

// F2 item 1: scheduler and worker-deployment system workflows on the per-ns
// queue. Every client start path is closed by D9: a relay start or
// signal-with-start, a worker child or continue-as-new, a relay signal to an
// existing one, and a worker poll of the per-ns queue. The server-side start
// path is open (F2-1): a worker poll of `nava-tq` with versioned deployment
// options makes matching start the worker-deployment workflows in `nava`
// (v1.29.7 physical_task_queue_manager.go ensureRegisteredInDeploymentVersion,
// gated only by system.enableDeploymentVersions, default true).
func TestAdversarialF2SystemWorkflowsOnPerNSQueue(t *testing.T) {
	relay, worker := peerCtx(string(Relay), true), peerCtx(string(Worker), true)
	for _, wt := range []string{schedulerType, deploymentType, deploymentVersionTyp} {
		for _, tq := range []string{perNSQueue, platformQueue()} {
			t.Run(fmt.Sprintf("relay start %s on %s", wt, tq), func(t *testing.T) {
				called, err := callUnary(t, relay, wf+"StartWorkflowExecution",
					&workflowservice.StartWorkflowExecutionRequest{Namespace: Namespace, WorkflowId: "t:tenantA:x",
						WorkflowType: &commonpb.WorkflowType{Name: wt}, TaskQueue: &taskqueuepb.TaskQueue{Name: tq}})
				requireDenied(t, called, err)
				called, err = callUnary(t, relay, wf+"SignalWithStartWorkflowExecution",
					&workflowservice.SignalWithStartWorkflowExecutionRequest{Namespace: Namespace, WorkflowId: "t:tenantA:x",
						SignalName: "s", WorkflowType: &commonpb.WorkflowType{Name: wt}, TaskQueue: &taskqueuepb.TaskQueue{Name: tq}})
				requireDenied(t, called, err)
			})
			t.Run(fmt.Sprintf("worker child and continue-as-new %s on %s", wt, tq), func(t *testing.T) {
				child := startChild(Namespace, "t:tenantA:child")
				child.GetStartChildWorkflowExecutionCommandAttributes().WorkflowType = &commonpb.WorkflowType{Name: wt}
				child.GetStartChildWorkflowExecutionCommandAttributes().TaskQueue = &taskqueuepb.TaskQueue{Name: tq}
				can := &commandpb.Command{
					CommandType: enumspb.COMMAND_TYPE_CONTINUE_AS_NEW_WORKFLOW_EXECUTION,
					Attributes: &commandpb.Command_ContinueAsNewWorkflowExecutionCommandAttributes{
						ContinueAsNewWorkflowExecutionCommandAttributes: &commandpb.ContinueAsNewWorkflowExecutionCommandAttributes{
							WorkflowType: &commonpb.WorkflowType{Name: wt}, TaskQueue: &taskqueuepb.TaskQueue{Name: tq},
							Header: tenantHeader(jsonString("tenantA"))}},
				}
				for _, cmd := range []*commandpb.Command{child, can} {
					called, err := callUnary(t, worker, respondWorkflowTaskCompleted, respond(taskToken(t, wfA), cmd))
					requireDenied(t, called, err)
				}
			})
		}
	}
	for _, id := range systemWorkflowIDs() {
		t.Run("relay signal "+id, func(t *testing.T) {
			called, err := callUnary(t, relay, wf+"SignalWorkflowExecution",
				&workflowservice.SignalWorkflowExecutionRequest{Namespace: Namespace, SignalName: "s",
					WorkflowExecution: &commonpb.WorkflowExecution{WorkflowId: id}})
			requireDenied(t, called, err)
		})
	}
	t.Run("worker poll of the per-ns queue", func(t *testing.T) {
		called, err := callUnary(t, worker, wf+"PollWorkflowTaskQueue",
			&workflowservice.PollWorkflowTaskQueueRequest{Namespace: Namespace, TaskQueue: &taskqueuepb.TaskQueue{Name: perNSQueue}})
		requireDenied(t, called, err)
		called, err = callUnary(t, worker, wf+"PollActivityTaskQueue",
			&workflowservice.PollActivityTaskQueueRequest{Namespace: Namespace, TaskQueue: &taskqueuepb.TaskQueue{Name: perNSQueue}})
		requireDenied(t, called, err)
	})

	t.Run("worker versioned poll starts deployment workflows", func(t *testing.T) {
		// Control: the poll the platform's own SDK sends (no versioning) passes.
		called, err := callUnary(t, worker, wf+"PollWorkflowTaskQueue",
			&workflowservice.PollWorkflowTaskQueueRequest{Namespace: Namespace,
				TaskQueue: &taskqueuepb.TaskQueue{Name: platformQueue()}})
		requireAllowed(t, called, err)

		t.Skip("F2-1 (TICK-122): open. A nava-worker poll with versioned deployment options makes the server " +
			"start temporal-sys-worker-deployment* workflows in nava; D9b does not inspect deployment options.")
		versioned := &deploymentpb.WorkerDeploymentOptions{DeploymentName: "nava-worker", BuildId: "build-1",
			WorkerVersioningMode: enumspb.WORKER_VERSIONING_MODE_VERSIONED}
		// The deprecated v1 field still reaches matching (DeploymentFromCapabilities), so
		// an attacker can send it.
		capabilities := &commonpb.WorkerVersionCapabilities{BuildId: "build-1", UseVersioning: true,
			DeploymentSeriesName: "nava-worker"}
		tq := &taskqueuepb.TaskQueue{Name: platformQueue()}
		for _, c := range []struct {
			name, api string
			req       any
		}{
			{"deployment options", "PollWorkflowTaskQueue", &workflowservice.PollWorkflowTaskQueueRequest{
				Namespace: Namespace, TaskQueue: tq, DeploymentOptions: versioned}},
			{"deployment options", "PollActivityTaskQueue", &workflowservice.PollActivityTaskQueueRequest{
				Namespace: Namespace, TaskQueue: tq, DeploymentOptions: versioned}},
			{"version capabilities", "PollWorkflowTaskQueue", &workflowservice.PollWorkflowTaskQueueRequest{
				Namespace: Namespace, TaskQueue: tq, WorkerVersionCapabilities: capabilities}}, //nolint:staticcheck // SA1019: the attack sends the deprecated field
			{"version capabilities", "PollActivityTaskQueue", &workflowservice.PollActivityTaskQueueRequest{
				Namespace: Namespace, TaskQueue: tq, WorkerVersionCapabilities: capabilities}}, //nolint:staticcheck // SA1019: the attack sends the deprecated field
		} {
			called, err := callUnary(t, worker, wf+c.api, c.req)
			if err == nil || called {
				t.Errorf("%s with %s allowed", c.api, c.name)
			}
		}
	})
}

// coreWorkflowIDRequests is every nava-core API, each naming the workflow id.
func coreWorkflowIDRequests(id string) map[string]any {
	ex := &commonpb.WorkflowExecution{WorkflowId: id}
	return map[string]any{
		"RequestCancelWorkflowExecution": &workflowservice.RequestCancelWorkflowExecutionRequest{Namespace: Namespace, WorkflowExecution: ex},
		"UpdateWorkflowExecution":        updateRequest(id, "set-current-version"),
		"QueryWorkflow": &workflowservice.QueryWorkflowRequest{Namespace: Namespace, Execution: ex,
			Query: &querypb.WorkflowQuery{QueryType: "describe-deployment"}},
		"DescribeWorkflowExecution":   &workflowservice.DescribeWorkflowExecutionRequest{Namespace: Namespace, Execution: ex},
		"GetWorkflowExecutionHistory": &workflowservice.GetWorkflowExecutionHistoryRequest{Namespace: Namespace, Execution: ex},
	}
}

func updateRequest(id, name string) *workflowservice.UpdateWorkflowExecutionRequest {
	return &workflowservice.UpdateWorkflowExecutionRequest{Namespace: Namespace,
		WorkflowExecution: &commonpb.WorkflowExecution{WorkflowId: id},
		Request:           &updatepb.Request{Meta: &updatepb.Meta{UpdateId: "u"}, Input: &updatepb.Input{Name: name}}}
}

// F2 item 2: nava-core's workflow-id calls carry no D9 check, so a nava-core
// certificate reaches Temporal's own workflows in `nava` (cancel, update, query,
// describe, read history). The `t:` prefix check cannot tell tenant A from B (the
// certificate has no tenant); it keeps nava-core off non-tenant workflows. The
// tenant boundary for these calls is nava-core's own JWT-derived workflow id.
func TestAdversarialF2CoreWorkflowIDTenantPrefix(t *testing.T) {
	core := peerCtx(string(Core), true)
	for api, req := range coreWorkflowIDRequests(wfA) {
		t.Run("control "+api, func(t *testing.T) {
			called, err := callUnary(t, core, wf+api, req)
			requireAllowed(t, called, err)
		})
	}

	t.Skip("F2-2 (TICK-122): open. nava-core RequestCancel/Update/Query/Describe/GetHistory accept a workflow id " +
		"with no t:<tenant>: prefix.")
	for _, id := range append(systemWorkflowIDs(), "", "t:", "t::x", "T:tenantA:x", "tenantA:x") {
		for api, req := range coreWorkflowIDRequests(id) {
			called, err := callUnary(t, core, wf+api, req)
			if err == nil || called {
				t.Errorf("nava-core %s on workflow id %q allowed", api, id)
			}
		}
	}
}

// activityTokenRequests is every nava-worker API that acts on a workflow through
// a task token alone, other than RespondWorkflowTaskCompleted (E1, checked).
func activityTokenRequests(token []byte) map[string]any {
	return map[string]any{
		"RespondActivityTaskCompleted": &workflowservice.RespondActivityTaskCompletedRequest{Namespace: Namespace, TaskToken: token},
		"RespondActivityTaskFailed":    &workflowservice.RespondActivityTaskFailedRequest{Namespace: Namespace, TaskToken: token},
		"RespondActivityTaskCanceled":  &workflowservice.RespondActivityTaskCanceledRequest{Namespace: Namespace, TaskToken: token},
		"RecordActivityTaskHeartbeat":  &workflowservice.RecordActivityTaskHeartbeatRequest{Namespace: Namespace, TaskToken: token},
		"RespondWorkflowTaskFailed":    &workflowservice.RespondWorkflowTaskFailedRequest{Namespace: Namespace, TaskToken: token},
	}
}

// F2 item 3: task tokens are unsigned protos. At v1.29.7 history accepts an
// activity token that names only the namespace id, the workflow id and the
// activity id (no run id, scheduled event id, attempt or version), so a
// nava-worker certificate can forge one for any started activity whose workflow
// id it knows: a system workflow's in `nava` included. The Authorizer checks
// the token's `t:` prefix only on RespondWorkflowTaskCompleted (E1).
func TestAdversarialF2ActivityTaskTokenTenantPrefix(t *testing.T) {
	worker := peerCtx(string(Worker), true)
	for api, req := range activityTokenRequests(taskToken(t, wfA)) {
		t.Run("control "+api, func(t *testing.T) {
			called, err := callUnary(t, worker, wf+api, req)
			requireAllowed(t, called, err)
		})
	}

	t.Skip("F2-3 (TICK-122): open. RespondActivityTask*, RecordActivityTaskHeartbeat and RespondWorkflowTaskFailed " +
		"do not check the task token's workflow id for the t:<tenant>: prefix.")
	for _, id := range append(systemWorkflowIDs(), "", "t:", "tenantA:x") {
		for api, req := range activityTokenRequests(taskToken(t, id)) {
			called, err := callUnary(t, worker, wf+api, req)
			if err == nil || called {
				t.Errorf("nava-worker %s with a token for %q allowed", api, id)
			}
		}
	}
	for api, req := range activityTokenRequests([]byte("not a token")) {
		called, err := callUnary(t, worker, wf+api, req)
		if err == nil || called {
			t.Errorf("nava-worker %s with an undecodable token allowed", api)
		}
	}
}

// F2 item 4: nava-core UpdateWorkflowExecution reaches the worker-deployment
// workflows, whose updates change which build gets `nava-tq`'s tasks. With F2-1
// a worker certificate creates them; with this, a core certificate steers them.
func TestAdversarialF2CoreUpdateWorkerDeploymentDenied(t *testing.T) {
	core := peerCtx(string(Core), true)
	called, err := callUnary(t, core, wf+"UpdateWorkflowExecution", updateRequest(wfA, "approve"))
	requireAllowed(t, called, err)

	t.Skip("F2-4 (TICK-122): open. nava-core UpdateWorkflowExecution reaches temporal-sys-worker-deployment* " +
		"workflows; closed by the F2-2 fix.")
	for _, id := range []string{deploymentID, deploymentVersionID} {
		for _, update := range []string{"set-current-version", "set-ramping-version", "register-worker-in-deployment",
			"delete-version", "set-manager-identity"} {
			called, err := callUnary(t, core, wf+"UpdateWorkflowExecution", updateRequest(id, update))
			if err == nil || called {
				t.Errorf("nava-core update %q on %s allowed", update, id)
			}
		}
	}
}

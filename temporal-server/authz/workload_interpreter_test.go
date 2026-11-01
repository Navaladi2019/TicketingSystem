package authz

// TICK-348 (P3-21, PLAN-003 D-3-11): the generic workflow type nava.interpreter
// runs on the existing nava-tq. nava-relay may start and signal it, nava-worker
// may poll and complete it, and nothing else changes: no other identity gains a
// verb, no look-alike type is admitted, and the E1 tenant checks on children
// and continue-as-new still hold for the new type.

import (
	"testing"

	commandpb "go.temporal.io/api/command/v1"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	taskqueuepb "go.temporal.io/api/taskqueue/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/server/common/authorization"
)

const interpreterType = "nava.interpreter"

func interpreterQueue() *taskqueuepb.TaskQueue {
	return &taskqueuepb.TaskQueue{Name: "nava-tq", Kind: enumspb.TASK_QUEUE_KIND_NORMAL}
}

func interpreterStart(wfID, wt string) *workflowservice.StartWorkflowExecutionRequest {
	return &workflowservice.StartWorkflowExecutionRequest{Namespace: Namespace, WorkflowId: wfID,
		WorkflowType: &commonpb.WorkflowType{Name: wt}, TaskQueue: interpreterQueue()}
}

// requireAPIDeny asserts a gate-level deny (the identity may not call api at all).
func requireAPIDeny(t *testing.T, id Identity, api string, req any) {
	t.Helper()
	decision, recs, _ := authorizeAs(t, id, api, req)
	if decision != authorization.DecisionDeny {
		t.Fatalf("%s calling %s: decision = %v, want deny", id, api, decision)
	}
	if len(recs) != 1 || recs[0].Tags["reason"] != ReasonAPI {
		t.Fatalf("%s calling %s: want one %s recording, got %v", id, api, ReasonAPI, recs)
	}
}

func TestInterpreterTypeIsAllowListed(t *testing.T) {
	// TICK-374: nava.interpreter is the only platform workflow type.
	if got := sortedKeys(platformWorkflowTypes); len(got) != 1 || got[0] != interpreterType {
		t.Fatalf("platformWorkflowTypes = %v, want exactly [%s]", got, interpreterType)
	}
	// D-3-11: no separate interpreter queue.
	if got := sortedKeys(platformTaskQueues); len(got) != 1 || got[0] != "nava-tq" {
		t.Fatalf("platformTaskQueues = %v, want exactly [nava-tq]", got)
	}
}

// Positive: the relay starts, signal-with-starts and signals an interpreter run.
func TestRelayStartsAndSignalsInterpreter(t *testing.T) {
	requireWorkloadAllow(t, Relay, wf+"StartWorkflowExecution", interpreterStart("t:tenantA:wf:1", interpreterType))
	requireWorkloadAllow(t, Relay, wf+"SignalWithStartWorkflowExecution", &workflowservice.SignalWithStartWorkflowExecutionRequest{
		Namespace: Namespace, WorkflowId: "t:tenantA:wf:1", SignalName: "event",
		WorkflowType: &commonpb.WorkflowType{Name: interpreterType}, TaskQueue: interpreterQueue()})
	// A signal names no workflow type, so this case is not evidence for any
	// allow-list entry: it passes D9 on the workflow id alone (TICK-374).
	requireWorkloadAllow(t, Relay, wf+"SignalWorkflowExecution", &workflowservice.SignalWorkflowExecutionRequest{
		Namespace: Namespace, SignalName: "event", WorkflowExecution: &commonpb.WorkflowExecution{WorkflowId: "t:tenantA:wf:1"}})
}

// Positive: the worker polls nava-tq and completes an interpreter task whose
// commands start a prefixed child and continue as new with the tenant header.
func TestWorkerPollsAndCompletesInterpreter(t *testing.T) {
	requireWorkloadAllow(t, Worker, wf+"PollWorkflowTaskQueue",
		&workflowservice.PollWorkflowTaskQueueRequest{Namespace: Namespace, TaskQueue: interpreterQueue()})
	requireWorkloadAllow(t, Worker, wf+"PollActivityTaskQueue",
		&workflowservice.PollActivityTaskQueueRequest{Namespace: Namespace, TaskQueue: interpreterQueue()})

	tok := taskToken(t, wfA)
	requireCommandAllow(t, respond(tok, interpreterChild("t:tenantA:wf:1:child", jsonString("tenantA"))))
	requireCommandAllow(t, respond(tok, interpreterContinueAsNew(jsonString("tenantA"))))
}

// Negative: a look-alike type is not admitted, for start, signal-with-start,
// child or continue-as-new. The allow-list is exact match, not prefix.
func TestInterpreterLookAlikeTypesDenied(t *testing.T) {
	for _, wt := range []string{"nava.interpreter2", "nava.interpreter.x", "nava.Interpreter", " nava.interpreter", "nava.interpreter "} {
		t.Run(wt, func(t *testing.T) { requireTypeDenied(t, wt) })
	}
}

// requireTypeDenied asserts that workflow type wt, otherwise a valid tenantA
// workload on nava-tq, is denied as workflow_type_not_allowed for the relay's
// start and signal-with-start and for a worker's child and continue-as-new.
func requireTypeDenied(t *testing.T, wt string) {
	t.Helper()
	requireWorkloadDeny(t, Relay, wf+"StartWorkflowExecution", interpreterStart("t:tenantA:wf:1", wt), ReasonWorkflowType)
	requireWorkloadDeny(t, Relay, wf+"SignalWithStartWorkflowExecution", &workflowservice.SignalWithStartWorkflowExecutionRequest{
		Namespace: Namespace, WorkflowId: "t:tenantA:wf:1", SignalName: "event",
		WorkflowType: &commonpb.WorkflowType{Name: wt}, TaskQueue: interpreterQueue()}, ReasonWorkflowType)

	tok := taskToken(t, wfA)
	child := interpreterChild("t:tenantA:wf:1:child", jsonString("tenantA"))
	child.GetStartChildWorkflowExecutionCommandAttributes().WorkflowType.Name = wt
	requireCommandDeny(t, denyCase{name: wt, req: respond(tok, child), reason: ReasonWorkflowType,
		command: "StartChildWorkflowExecution", workflowID: wfA, target: "t:tenantA:wf:1:child", commandIdx: "0"})
	can := interpreterContinueAsNew(jsonString("tenantA"))
	can.GetContinueAsNewWorkflowExecutionCommandAttributes().WorkflowType.Name = wt
	requireCommandDeny(t, denyCase{name: wt, req: respond(tok, can), reason: ReasonWorkflowType,
		command: "ContinueAsNewWorkflowExecution", workflowID: wfA, commandIdx: "0"})
}

// Negative: the new type grants no identity a new verb. The worker cannot start
// or signal it (only the relay signals in S11a), the relay cannot poll or
// complete it, and nava-core cannot start it.
func TestInterpreterVerbsStayWithTheirIdentities(t *testing.T) {
	start := interpreterStart("t:tenantA:wf:1", interpreterType)
	signal := &workflowservice.SignalWorkflowExecutionRequest{Namespace: Namespace, SignalName: "event",
		WorkflowExecution: &commonpb.WorkflowExecution{WorkflowId: "t:tenantA:wf:1"}}
	sws := &workflowservice.SignalWithStartWorkflowExecutionRequest{Namespace: Namespace, WorkflowId: "t:tenantA:wf:1",
		SignalName: "event", WorkflowType: &commonpb.WorkflowType{Name: interpreterType}, TaskQueue: interpreterQueue()}
	poll := &workflowservice.PollWorkflowTaskQueueRequest{Namespace: Namespace, TaskQueue: interpreterQueue()}
	pollAct := &workflowservice.PollActivityTaskQueueRequest{Namespace: Namespace, TaskQueue: interpreterQueue()}
	complete := respond(taskToken(t, wfA))

	for _, c := range []struct {
		id  Identity
		api string
		req any
	}{
		{Worker, wf + "StartWorkflowExecution", start},
		{Worker, wf + "SignalWithStartWorkflowExecution", sws},
		{Worker, wf + "SignalWorkflowExecution", signal},
		{Relay, wf + "PollWorkflowTaskQueue", poll},
		{Relay, wf + "PollActivityTaskQueue", pollAct},
		{Relay, respondWorkflowTaskCompleted, complete},
		{Core, wf + "StartWorkflowExecution", start},
		{Core, wf + "SignalWorkflowExecution", signal},
		{Core, wf + "PollWorkflowTaskQueue", poll},
	} {
		t.Run(string(c.id)+" "+c.api, func(t *testing.T) { requireAPIDeny(t, c.id, c.api, c.req) })
	}
}

// Negative: the E1 tenant checks are unchanged for the interpreter type. A
// child must carry the caller's `t:<tenant>:` prefix and its tenant header;
// continue-as-new must keep the `nava-tenant` header.
func TestInterpreterKeepsTenantChecks(t *testing.T) {
	tok := taskToken(t, wfA)
	for _, c := range []denyCase{
		{name: "child in tenant B", req: respond(tok, interpreterChild("t:tenantB:wf:1:child", jsonString("tenantA"))),
			reason: ReasonCrossTenant, command: "StartChildWorkflowExecution", target: "t:tenantB:wf:1:child"},
		{name: "child without prefix", req: respond(tok, interpreterChild("wf:1:child", jsonString("tenantA"))),
			reason: ReasonCrossTenant, command: "StartChildWorkflowExecution", target: "wf:1:child"},
		{name: "continue-as-new without header", req: respond(tok, interpreterContinueAsNew(nil)),
			reason: ReasonTenantHeader, command: "ContinueAsNewWorkflowExecution"},
		{name: "continue-as-new as tenant B", req: respond(tok, interpreterContinueAsNew(jsonString("tenantB"))),
			reason: ReasonTenantHeader, command: "ContinueAsNewWorkflowExecution"},
	} {
		c.workflowID, c.commandIdx = wfA, "0"
		t.Run(c.name, func(t *testing.T) { requireCommandDeny(t, c) })
	}

	// The relay cannot start an interpreter run outside a tenant.
	requireWorkloadDeny(t, Relay, wf+"StartWorkflowExecution", interpreterStart("wf:1", interpreterType), ReasonNoTenantPrefix)
}

func interpreterChild(workflowID string, header *commonpb.Payload) *commandpb.Command {
	cmd := startChildWith(Namespace, workflowID, header, nil)
	attrs := cmd.GetStartChildWorkflowExecutionCommandAttributes()
	attrs.WorkflowType = &commonpb.WorkflowType{Name: interpreterType}
	attrs.TaskQueue = interpreterQueue()
	return cmd
}

func interpreterContinueAsNew(header *commonpb.Payload) *commandpb.Command {
	cmd := continueAsNew(header, "")
	attrs := cmd.GetContinueAsNewWorkflowExecutionCommandAttributes()
	attrs.WorkflowType = &commonpb.WorkflowType{Name: interpreterType}
	attrs.TaskQueue = interpreterQueue()
	return cmd
}

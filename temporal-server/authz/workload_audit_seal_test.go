package authz

// TICK-438 (P5-30, ADR-004 R18, ADR-001-A1): the R16 audit sealer runs as
// workflow type audit.seal on nava-tq. It has the workload rules of
// nava.interpreter for starts: nava-relay starts it on nava-tq, under a
// `t:<tenant>:` workflow id, and nothing else does. Unlike nava.interpreter it
// is start-only (TICK-449 grooming): a worker command may not name it, so a
// StartChildWorkflowExecution or ContinueAsNewWorkflowExecution of audit.seal
// is denied. The sealer reschedules itself through the outbox (R16 A1), so
// every sealer run is a relay start that D22 tracks.

import (
	"testing"

	commandpb "go.temporal.io/api/command/v1"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/server/common/authorization"
)

const (
	auditSealType = "audit.seal"
	// sealRunA is run o=42 of tenantA's sealer lineage (R16).
	sealRunA = "t:tenantA:audit.seal:42"
)

func auditSealSignalWithStart(wfID, wt string) *workflowservice.SignalWithStartWorkflowExecutionRequest {
	return &workflowservice.SignalWithStartWorkflowExecutionRequest{Namespace: Namespace, WorkflowId: wfID,
		SignalName: "event", WorkflowType: &commonpb.WorkflowType{Name: wt}, TaskQueue: interpreterQueue()}
}

// The sets hold exactly what R18 adds: audit.seal joins nava.interpreter, no
// new queue, and audit.seal is the one start-only type.
func TestAuditSealIsAllowListed(t *testing.T) {
	if got := sortedKeys(platformWorkflowTypes()); len(got) != 2 || got[0] != auditSealType || got[1] != interpreterType {
		t.Fatalf("platformWorkflowTypes = %v, want exactly [%s %s]", got, auditSealType, interpreterType)
	}
	if got := sortedKeys(startOnlyWorkflowTypes()); len(got) != 1 || got[0] != auditSealType {
		t.Fatalf("startOnlyWorkflowTypes = %v, want exactly [%s]", got, auditSealType)
	}
	// A start-only type must still be a platform type, or the relay could not
	// start it either and the entry would silently mean nothing.
	for _, wt := range sortedKeys(startOnlyWorkflowTypes()) {
		if !inSet(platformWorkflowTypes(), wt) {
			t.Errorf("start-only type %q is not in platformWorkflowTypes", wt)
		}
	}
	if got := sortedKeys(platformTaskQueues()); len(got) != 1 || got[0] != "nava-tq" {
		t.Fatalf("platformTaskQueues = %v, want exactly [nava-tq]", got)
	}
}

// Positive: the relay starts (and signal-with-starts) a sealer run on nava-tq.
func TestRelayStartsAuditSeal(t *testing.T) {
	requireWorkloadAllow(t, Relay, wf+"StartWorkflowExecution", interpreterStart(sealRunA, auditSealType))
	requireWorkloadAllow(t, Relay, wf+"SignalWithStartWorkflowExecution", auditSealSignalWithStart(sealRunA, auditSealType))
	// The worker polls the same queue the sealer runs on; no new queue.
	requireWorkloadAllow(t, Worker, wf+"PollWorkflowTaskQueue",
		&workflowservice.PollWorkflowTaskQueueRequest{Namespace: Namespace, TaskQueue: interpreterQueue()})
}

// Negative: no identity but the relay may start audit.seal. nava-worker and
// nava-core lack the API; nava-operator is read-only plus Terminate.
func TestAuditSealStartedByRelayOnly(t *testing.T) {
	start := interpreterStart(sealRunA, auditSealType)
	sws := auditSealSignalWithStart(sealRunA, auditSealType)
	for _, id := range []Identity{Worker, Core, Operator} {
		t.Run(string(id), func(t *testing.T) {
			requireAPIDeny(t, id, wf+"StartWorkflowExecution", start)
			requireAPIDeny(t, id, wf+"SignalWithStartWorkflowExecution", sws)
			requireAPIDeny(t, id, wf+"ExecuteMultiOperation", &workflowservice.ExecuteMultiOperationRequest{Namespace: Namespace})
		})
	}
	// A certificate with no platform identity is denied before any of this.
	decision, _, _ := authorizeAs(t, "", wf+"StartWorkflowExecution", start)
	if decision != authorization.DecisionDeny {
		t.Fatalf("no identity: decision = %v, want deny", decision)
	}
}

// Negative: the relay may start audit.seal only on nava-tq, as a NORMAL queue.
func TestAuditSealOtherQueuesDenied(t *testing.T) {
	for _, c := range []struct {
		name   string
		tq     string
		kind   enumspb.TaskQueueKind
		normal string
		reason string
	}{
		{"empty queue", "", enumspb.TASK_QUEUE_KIND_NORMAL, "", ReasonTaskQueue},
		{"audit queue", "audit-tq", enumspb.TASK_QUEUE_KIND_NORMAL, "", ReasonTaskQueue},
		{"look-alike queue", "nava-tq ", enumspb.TASK_QUEUE_KIND_NORMAL, "", ReasonTaskQueue},
		{"system queue", "temporal-sys-per-ns-tq", enumspb.TASK_QUEUE_KIND_NORMAL, "", ReasonTaskQueue},
		{"sticky kind", "nava-tq", enumspb.TASK_QUEUE_KIND_STICKY, "nava-tq", ReasonTaskQueueKind},
	} {
		t.Run(c.name, func(t *testing.T) {
			start := interpreterStart(sealRunA, auditSealType)
			start.TaskQueue = tqOf(c.tq, c.kind, c.normal)
			requireWorkloadDeny(t, Relay, wf+"StartWorkflowExecution", start, c.reason)
			sws := auditSealSignalWithStart(sealRunA, auditSealType)
			sws.TaskQueue = tqOf(c.tq, c.kind, c.normal)
			requireWorkloadDeny(t, Relay, wf+"SignalWithStartWorkflowExecution", sws, c.reason)
		})
	}
}

// Negative: a sealer run outside a tenant is denied, so the relay cannot start
// a sealer whose id names no tenant (or an empty one).
func TestAuditSealNeedsTenantPrefix(t *testing.T) {
	for _, id := range []string{"audit.seal:42", "t::audit.seal:42", "t:tenantA", ""} {
		t.Run(id, func(t *testing.T) {
			requireWorkloadDeny(t, Relay, wf+"StartWorkflowExecution", interpreterStart(id, auditSealType), ReasonNoTenantPrefix)
		})
	}
}

// Negative: the allow-list is exact match; no variant of audit.seal is
// admitted for start, signal-with-start, child or continue-as-new.
func TestAuditSealLookAlikeTypesDenied(t *testing.T) {
	for _, wt := range []string{
		"audit.seal2", "audit.seal.x", "audit.Seal", "Audit.seal", "AUDIT.SEAL",
		" audit.seal", "audit.seal ", "audit.seal\x00", "audit-seal", "audit_seal", "audit", "seal",
		"audit.seal:42", "nava.audit.seal", "audit.seal/nava.interpreter",
	} {
		t.Run(wt, func(t *testing.T) { requireTypeDenied(t, wt) })
	}
}

// Negative (TICK-449, binding): a worker command may not name audit.seal. A
// child or a continue-as-new of audit.seal, otherwise a valid tenantA command on
// nava-tq with the tenant header, is denied as workflow_type_not_allowed. A
// compromised worker cannot spawn a sealer that bypasses the outbox (and so the
// head's next_seal_outbox_id fencing), nor turn an interpreter run into one.
func TestWorkerCommandsCannotNameAuditSeal(t *testing.T) {
	tok := taskToken(t, wfA)
	sealTok := taskToken(t, sealRunA)

	child := func(workflowID string) *commandpb.Command {
		cmd := interpreterChild(workflowID, jsonString("tenantA"))
		cmd.GetStartChildWorkflowExecutionCommandAttributes().WorkflowType.Name = auditSealType
		return cmd
	}
	can := func() *commandpb.Command {
		cmd := interpreterContinueAsNew(jsonString("tenantA"))
		cmd.GetContinueAsNewWorkflowExecutionCommandAttributes().WorkflowType.Name = auditSealType
		return cmd
	}
	canInheritQueue := can()
	canInheritQueue.GetContinueAsNewWorkflowExecutionCommandAttributes().TaskQueue = nil

	for _, c := range []denyCase{
		{name: "interpreter starts a sealer child", req: respond(tok, child("t:tenantA:audit.seal:43")),
			reason: ReasonWorkflowType, command: "StartChildWorkflowExecution", workflowID: wfA,
			target: "t:tenantA:audit.seal:43", commandIdx: "0"},
		{name: "sealer starts a sealer child", req: respond(sealTok, child("t:tenantA:audit.seal:43")),
			reason: ReasonWorkflowType, command: "StartChildWorkflowExecution", workflowID: sealRunA,
			target: "t:tenantA:audit.seal:43", commandIdx: "0"},
		{name: "interpreter continues as a sealer", req: respond(tok, can()),
			reason: ReasonWorkflowType, command: "ContinueAsNewWorkflowExecution", workflowID: wfA, commandIdx: "0"},
		{name: "sealer continues as a sealer", req: respond(sealTok, can()),
			reason: ReasonWorkflowType, command: "ContinueAsNewWorkflowExecution", workflowID: sealRunA, commandIdx: "0"},
		{name: "sealer continues as a sealer, inherited queue", req: respond(sealTok, canInheritQueue),
			reason: ReasonWorkflowType, command: "ContinueAsNewWorkflowExecution", workflowID: sealRunA, commandIdx: "0"},
		// The whole response is denied when any command names audit.seal, even
		// after an allowed one.
		{name: "sealer child after an allowed child", req: respond(tok,
			interpreterChild("t:tenantA:wf:1:child", jsonString("tenantA")), child("t:tenantA:audit.seal:43")),
			reason: ReasonWorkflowType, command: "StartChildWorkflowExecution", workflowID: wfA,
			target: "t:tenantA:audit.seal:43", commandIdx: "1"},
	} {
		t.Run(c.name, func(t *testing.T) { requireCommandDeny(t, c) })
	}

	// The deny says why, so the alert does not read as an unknown type.
	_, _, logs := authorizeWorker(t, respond(tok, can()))
	if len(logs) != 1 || logs[0]["detail"] != `workflow type "audit.seal" is start-only` {
		t.Fatalf("log detail = %v, want the start-only detail", logs)
	}
}

// Positive: start-only bounds what a command names, not what a sealer run may
// do. A sealer task's ordinary commands (activities on nava-tq with the tenant
// header, completion) stay allowed, and the start-only rule does not touch
// nava.interpreter children or continue-as-new.
func TestAuditSealRunKeepsOrdinaryCommands(t *testing.T) {
	sealTok := taskToken(t, sealRunA)
	requireCommandAllow(t, respond(sealTok, scheduleActivity(jsonString("tenantA"))))
	requireCommandAllow(t, respond(sealTok, &commandpb.Command{CommandType: enumspb.COMMAND_TYPE_COMPLETE_WORKFLOW_EXECUTION}))

	tok := taskToken(t, wfA)
	requireCommandAllow(t, respond(tok, interpreterChild("t:tenantA:wf:1:child", jsonString("tenantA"))))
	requireCommandAllow(t, respond(tok, interpreterContinueAsNew(jsonString("tenantA"))))
}

// Negative: the E1 tenant checks hold for a sealer task too. A sealer run of
// tenantA cannot schedule an activity or signal a workflow as tenantB.
func TestAuditSealRunKeepsTenantChecks(t *testing.T) {
	sealTok := taskToken(t, sealRunA)
	for _, c := range []denyCase{
		{name: "activity as tenant B", req: respond(sealTok, scheduleActivity(jsonString("tenantB"))),
			reason: ReasonActivityHeader, command: "ScheduleActivityTask"},
		{name: "signal tenant B sealer", req: respond(sealTok, signalExternal(Namespace, "t:tenantB:audit.seal:42")),
			reason: ReasonCrossTenant, command: "SignalExternalWorkflowExecution", target: "t:tenantB:audit.seal:42"},
		{name: "cancel tenant B sealer", req: respond(sealTok, cancelExternal(Namespace, "t:tenantB:audit.seal:42")),
			reason: ReasonCrossTenant, command: "RequestCancelExternalWorkflowExecution", target: "t:tenantB:audit.seal:42"},
	} {
		c.workflowID, c.commandIdx = sealRunA, "0"
		t.Run(c.name, func(t *testing.T) { requireCommandDeny(t, c) })
	}
	// The relay cannot signal a sealer outside a tenant either.
	requireWorkloadDeny(t, Relay, wf+"SignalWorkflowExecution", &workflowservice.SignalWorkflowExecutionRequest{
		Namespace: Namespace, SignalName: "event", WorkflowExecution: &commonpb.WorkflowExecution{WorkflowId: "audit.seal:42"}},
		ReasonNoTenantPrefix)
}

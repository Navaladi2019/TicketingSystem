package authz

// TICK-745 (P7-05, ADR-006 D10, PLAN-007 D-7-12): the field-index backfill
// runs as two workflow types on nava-tq. field_index.coordinator is the one
// per-tenant queue `t:<tenant>:field_index.coordinator`: nava-relay
// signal-with-starts it from a field_index.backfill outbox row, and it
// continues as new. field_index.backfill.run is the backfill itself and is
// child-only: the coordinator starts it as a child, one at a time per tenant,
// and no relay start or signal-with-start may name it, so nothing runs a
// backfill around the coordinator (D10.5).

import (
	"testing"

	commandpb "go.temporal.io/api/command/v1"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/workflowservice/v1"
)

const (
	fieldIndexCoordinatorType = "field_index.coordinator"
	fieldIndexBackfillType    = "field_index.backfill.run"
	// coordinatorA is tenantA's coordinator (one id per tenant).
	coordinatorA = "t:tenantA:field_index.coordinator"
	// backfillRunA is a backfill child of coordinatorA for one field and
	// requested version.
	backfillRunA = "t:tenantA:field_index.backfill.run:3f0c4a52-7d6e-4b8a-9a51-2c1f0e9d8b7a:2"
)

func fieldIndexSignalWithStart(wfID, wt string) *workflowservice.SignalWithStartWorkflowExecutionRequest {
	return &workflowservice.SignalWithStartWorkflowExecutionRequest{Namespace: Namespace, WorkflowId: wfID,
		SignalName: "field_index.enqueue", WorkflowType: &commonpb.WorkflowType{Name: wt}, TaskQueue: interpreterQueue()}
}

// The child-only set is exactly the backfill, a subset of the platform types,
// and disjoint from the start-only set: a type in both could never run.
func TestChildOnlyWorkflowTypes(t *testing.T) {
	if got := sortedKeys(childOnlyWorkflowTypes()); len(got) != 1 || got[0] != fieldIndexBackfillType {
		t.Fatalf("childOnlyWorkflowTypes = %v, want exactly [%s]", got, fieldIndexBackfillType)
	}
	for _, wt := range sortedKeys(childOnlyWorkflowTypes()) {
		if !inSet(platformWorkflowTypes(), wt) {
			t.Errorf("child-only type %q is not in platformWorkflowTypes", wt)
		}
		if inSet(startOnlyWorkflowTypes(), wt) {
			t.Errorf("type %q is both child-only and start-only", wt)
		}
	}
	if inSet(childOnlyWorkflowTypes(), fieldIndexCoordinatorType) || inSet(startOnlyWorkflowTypes(), fieldIndexCoordinatorType) {
		t.Errorf("%s must be neither child-only nor start-only: the relay starts it and it continues as new", fieldIndexCoordinatorType)
	}
}

// Positive: the relay signal-with-starts the tenant's coordinator on nava-tq.
func TestRelaySignalWithStartsCoordinator(t *testing.T) {
	requireWorkloadAllow(t, Relay, wf+"SignalWithStartWorkflowExecution", fieldIndexSignalWithStart(coordinatorA, fieldIndexCoordinatorType))
	requireWorkloadAllow(t, Relay, wf+"SignalWorkflowExecution", &workflowservice.SignalWorkflowExecutionRequest{
		Namespace: Namespace, SignalName: "field_index.enqueue", WorkflowExecution: &commonpb.WorkflowExecution{WorkflowId: coordinatorA}})
}

// Adversarial (TICK-745 binding): the relay starts field_index.backfill.run
// directly, by start or by signal-with-start, under a valid tenantA id on
// nava-tq, and is denied as workflow_type_not_allowed. A forged or mis-kinded
// outbox row cannot run a second backfill beside the coordinator's.
func TestRelayCannotStartBackfillDirectly(t *testing.T) {
	requireWorkloadDeny(t, Relay, wf+"StartWorkflowExecution", interpreterStart(backfillRunA, fieldIndexBackfillType), ReasonWorkflowType)
	requireWorkloadDeny(t, Relay, wf+"SignalWithStartWorkflowExecution", fieldIndexSignalWithStart(backfillRunA, fieldIndexBackfillType), ReasonWorkflowType)
	// The same under the coordinator's own id: the id does not make the type
	// startable.
	requireWorkloadDeny(t, Relay, wf+"SignalWithStartWorkflowExecution", fieldIndexSignalWithStart(coordinatorA, fieldIndexBackfillType), ReasonWorkflowType)

	// The deny says why, so the alert does not read as an unknown type.
	_, _, logs := authorizeAs(t, Relay, wf+"StartWorkflowExecution", interpreterStart(backfillRunA, fieldIndexBackfillType))
	if len(logs) != 1 || logs[0]["detail"] != `workflow type "field_index.backfill.run" is child-only` {
		t.Fatalf("log detail = %v, want the child-only detail", logs)
	}
}

// Negative: no identity but the relay may start the coordinator.
func TestCoordinatorStartedByRelayOnly(t *testing.T) {
	sws := fieldIndexSignalWithStart(coordinatorA, fieldIndexCoordinatorType)
	start := interpreterStart(coordinatorA, fieldIndexCoordinatorType)
	for _, id := range []Identity{Worker, Core, Operator} {
		t.Run(string(id), func(t *testing.T) {
			requireAPIDeny(t, id, wf+"SignalWithStartWorkflowExecution", sws)
			requireAPIDeny(t, id, wf+"StartWorkflowExecution", start)
		})
	}
}

// Negative: a coordinator outside a tenant is denied.
func TestCoordinatorNeedsTenantPrefix(t *testing.T) {
	for _, id := range []string{"field_index.coordinator", "field-index/tenantA", "t::field_index.coordinator", "t:tenantA", ""} {
		t.Run(id, func(t *testing.T) {
			requireWorkloadDeny(t, Relay, wf+"SignalWithStartWorkflowExecution",
				fieldIndexSignalWithStart(id, fieldIndexCoordinatorType), ReasonNoTenantPrefix)
		})
	}
}

// Negative: the allow-list is exact match for both new types.
func TestFieldIndexLookAlikeTypesDenied(t *testing.T) {
	for _, wt := range []string{
		"field_index.coordinator ", " field_index.coordinator", "field_index.Coordinator", "field-index.coordinator",
		"field_index.backfill", "field_index.backfill.run2", "field_index.backfill.Run", "field_index",
		"field_index.remove.run", "field_index.coordinator\x00",
	} {
		t.Run(wt, func(t *testing.T) { requireTypeDenied(t, wt) })
	}
}

// Positive: the coordinator starts a backfill child and continues as new,
// both in its own tenant on nava-tq with the tenant header.
func TestCoordinatorStartsBackfillChildAndContinuesAsNew(t *testing.T) {
	tok := taskToken(t, coordinatorA)
	child := interpreterChild(backfillRunA, jsonString("tenantA"))
	child.GetStartChildWorkflowExecutionCommandAttributes().WorkflowType.Name = fieldIndexBackfillType
	can := interpreterContinueAsNew(jsonString("tenantA"))
	can.GetContinueAsNewWorkflowExecutionCommandAttributes().WorkflowType.Name = fieldIndexCoordinatorType
	requireCommandAllow(t, respond(tok, child))
	requireCommandAllow(t, respond(tok, can))
	requireCommandAllow(t, respond(tok, scheduleActivity(jsonString("tenantA"))))
	requireCommandAllow(t, respond(taskToken(t, backfillRunA), scheduleActivity(jsonString("tenantA"))))
}

// Negative: the E1 tenant checks hold for the coordinator. Tenant A's
// coordinator cannot start a backfill child in tenant B, with tenant B's
// header, or signal tenant B's coordinator.
func TestCoordinatorKeepsTenantChecks(t *testing.T) {
	tok := taskToken(t, coordinatorA)
	child := func(id, header string) *commandpb.Command {
		cmd := interpreterChild(id, jsonString(header))
		cmd.GetStartChildWorkflowExecutionCommandAttributes().WorkflowType.Name = fieldIndexBackfillType
		return cmd
	}
	bChild := "t:tenantB:field_index.backfill.run:3f0c4a52-7d6e-4b8a-9a51-2c1f0e9d8b7a:2"
	for _, c := range []denyCase{
		{name: "child in tenant B", req: respond(tok, child(bChild, "tenantB")),
			reason: ReasonCrossTenant, command: "StartChildWorkflowExecution", target: bChild},
		{name: "child with tenant B header", req: respond(tok, child(backfillRunA, "tenantB")),
			reason: ReasonChildHeader, command: "StartChildWorkflowExecution", target: backfillRunA},
		{name: "signal tenant B coordinator", req: respond(tok, signalExternal(Namespace, "t:tenantB:field_index.coordinator")),
			reason: ReasonCrossTenant, command: "SignalExternalWorkflowExecution", target: "t:tenantB:field_index.coordinator"},
	} {
		c.workflowID, c.commandIdx = coordinatorA, "0"
		t.Run(c.name, func(t *testing.T) { requireCommandDeny(t, c) })
	}
	// A backfill child on another queue is denied too.
	other := child(backfillRunA, "tenantA")
	other.GetStartChildWorkflowExecutionCommandAttributes().TaskQueue = tqOf("audit-tq", enumspb.TASK_QUEUE_KIND_NORMAL, "")
	requireCommandDeny(t, denyCase{name: "other queue", req: respond(tok, other), reason: ReasonTaskQueue,
		command: "StartChildWorkflowExecution", workflowID: coordinatorA, target: backfillRunA, commandIdx: "0"})
}

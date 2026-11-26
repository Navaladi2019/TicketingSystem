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

// The child-only set is exactly the backfill and the removal (TICK-750), a
// subset of the platform types, and disjoint from the start-only set: a type
// in both could never run.
func TestChildOnlyWorkflowTypes(t *testing.T) {
	if got := sortedKeys(childOnlyWorkflowTypes()); len(got) != 2 || got[0] != fieldIndexBackfillType || got[1] != fieldIndexRemoveType {
		t.Fatalf("childOnlyWorkflowTypes = %v, want exactly [%s %s]", got, fieldIndexBackfillType, fieldIndexRemoveType)
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

// TICK-750: a relay start of field_index.remove.run is denied as child-only,
// and the deny says so (adversarial_s22_s29_test.go covers every id and
// identity).
func TestRelayRemovalDenyNamesChildOnly(t *testing.T) {
	_, _, logs := authorizeAs(t, Relay, wf+"StartWorkflowExecution", interpreterStart(removeRunA, fieldIndexRemoveType))
	if len(logs) != 1 || logs[0]["detail"] != `workflow type "field_index.remove.run" is child-only` {
		t.Fatalf("log detail = %v, want the child-only detail", logs)
	}
}

// Positive: a removal run schedules its activities and continues as new in
// its own tenant (the coordinator's start of it is in
// TestCoordinatorRemovalKeepsTenantChecks).
func TestRemovalRunSchedulesAndContinuesAsNew(t *testing.T) {
	can := interpreterContinueAsNew(jsonString("tenantA"))
	can.GetContinueAsNewWorkflowExecutionCommandAttributes().WorkflowType.Name = fieldIndexRemoveType
	requireCommandAllow(t, respond(taskToken(t, removeRunA), can))
	requireCommandAllow(t, respond(taskToken(t, removeRunA), scheduleActivity(jsonString("tenantA"))))
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

// Negative: the allow-list is exact match for the field-index types.
func TestFieldIndexLookAlikeTypesDenied(t *testing.T) {
	for _, wt := range []string{
		"field_index.coordinator ", " field_index.coordinator", "field_index.Coordinator", "field-index.coordinator",
		"field_index.backfill", "field_index.backfill.run2", "field_index.backfill.Run", "field_index",
		"field_index.remove", "field_index.remove.run2", "field_index.Remove.run", "field_index.coordinator\x00",
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

// Adversarial (TICK-750): the parent rule. A child-only type is started only
// by its tenant's coordinator and continued as new only by a run of itself:
// an interpreter run, another tenant's coordinator id shape, a backfill run
// starting a removal, and a removal continuing as a backfill are denied.
func TestChildOnlyParentRule(t *testing.T) {
	childOf := func(id, wt string) *commandpb.Command {
		cmd := interpreterChild(id, jsonString("tenantA"))
		cmd.GetStartChildWorkflowExecutionCommandAttributes().WorkflowType.Name = wt
		return cmd
	}
	canAs := func(wt string) *commandpb.Command {
		cmd := interpreterContinueAsNew(jsonString("tenantA"))
		cmd.GetContinueAsNewWorkflowExecutionCommandAttributes().WorkflowType.Name = wt
		return cmd
	}
	for _, c := range []struct {
		name, parent string
		cmd          *commandpb.Command
		command      string
		target       string
	}{
		{"interpreter starts a backfill", wfA, childOf(backfillRunA, fieldIndexBackfillType), "StartChildWorkflowExecution", backfillRunA},
		{"backfill run starts a removal", backfillRunA, childOf(removeRunA, fieldIndexRemoveType), "StartChildWorkflowExecution", removeRunA},
		{"coordinator look-alike starts a removal", "t:tenantA:field_index.coordinator:x", childOf(removeRunA, fieldIndexRemoveType), "StartChildWorkflowExecution", removeRunA},
		{"interpreter continues as a removal", wfA, canAs(fieldIndexRemoveType), "ContinueAsNewWorkflowExecution", ""},
		{"removal continues as a backfill", removeRunA, canAs(fieldIndexBackfillType), "ContinueAsNewWorkflowExecution", ""},
		{"coordinator continues as a backfill", coordinatorA, canAs(fieldIndexBackfillType), "ContinueAsNewWorkflowExecution", ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			requireCommandDeny(t, denyCase{name: c.name, req: respond(taskToken(t, c.parent), c.cmd), reason: ReasonWorkflowType,
				command: c.command, workflowID: c.parent, target: c.target, commandIdx: "0"})
		})
	}
	requireCommandAllow(t, respond(taskToken(t, backfillRunA), canAs(fieldIndexBackfillType)))
	requireCommandAllow(t, respond(taskToken(t, coordinatorA), childOf(removeRunA, fieldIndexRemoveType)))
}

// The id-to-type binding (checkReservedID) on the other ways a run is
// created: a relay signal-with-start of another type under the coordinator's
// id, and a child of another type under the backfill prefix, are denied; the
// real types under their own ids are allowed.
func TestReservedIDsAreBoundToTheirTypes(t *testing.T) {
	requireWorkloadDeny(t, Relay, wf+"SignalWithStartWorkflowExecution", fieldIndexSignalWithStart(coordinatorA, interpreterType), ReasonWorkflowType)
	requireWorkloadAllow(t, Relay, wf+"SignalWithStartWorkflowExecution", fieldIndexSignalWithStart(coordinatorA, fieldIndexCoordinatorType))
	spoof := "t:tenantA:field_index.backfill.run:spoof"
	requireCommandDeny(t, denyCase{name: "interpreter child under backfill prefix",
		req:    respond(taskToken(t, wfA), interpreterChild(spoof, jsonString("tenantA"))),
		reason: ReasonWorkflowType, command: "StartChildWorkflowExecution", workflowID: wfA, target: spoof, commandIdx: "0"})
	// A removal child under a backfill id is denied too.
	cmd := interpreterChild(backfillRunA, jsonString("tenantA"))
	cmd.GetStartChildWorkflowExecutionCommandAttributes().WorkflowType.Name = fieldIndexRemoveType
	requireCommandDeny(t, denyCase{name: "removal under backfill id", req: respond(taskToken(t, coordinatorA), cmd),
		reason: ReasonWorkflowType, command: "StartChildWorkflowExecution", workflowID: coordinatorA, target: backfillRunA, commandIdx: "0"})
}

// Continue-as-new keeps the id, so the binding holds there too (TICK-750
// security review, Low residual): a coordinator run cannot continue as
// nava.interpreter and leave an interpreter holding the coordinator's id, nor
// a removal run as another type; each may continue as itself.
func TestReservedIDsStayBoundAcrossContinueAsNew(t *testing.T) {
	canAs := func(wt string) *commandpb.Command {
		cmd := interpreterContinueAsNew(jsonString("tenantA"))
		cmd.GetContinueAsNewWorkflowExecutionCommandAttributes().WorkflowType.Name = wt
		return cmd
	}
	for _, c := range []struct{ name, id, wt string }{
		{"coordinator continues as nava.interpreter", coordinatorA, interpreterType},
		{"removal continues as nava.interpreter", removeRunA, interpreterType},
	} {
		t.Run(c.name, func(t *testing.T) {
			requireCommandDeny(t, denyCase{name: c.name, req: respond(taskToken(t, c.id), canAs(c.wt)), reason: ReasonWorkflowType,
				command: "ContinueAsNewWorkflowExecution", workflowID: c.id, commandIdx: "0"})
		})
	}
	requireCommandAllow(t, respond(taskToken(t, coordinatorA), canAs(fieldIndexCoordinatorType)))
	requireCommandAllow(t, respond(taskToken(t, removeRunA), canAs(fieldIndexRemoveType)))
	requireCommandAllow(t, respond(taskToken(t, wfA), canAs(interpreterType)))
}

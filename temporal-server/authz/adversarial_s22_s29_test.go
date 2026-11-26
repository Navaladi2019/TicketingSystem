package authz

// TICK-753 (S22 P7-13 and S29 P10-11, Security & Multi-Tenancy Reviewer):
// adversarial Temporal authorization tests for the two workflow types S22 and
// S29 add. Threat model: a compromised nava-worker (it holds tenant A's task
// tokens) or a mis-kinded or forged outbox row driving nava-relay.
//
//   - schema.backfill (ADR-009 D2, contract C3): start-only, from nava-relay
//     only, under `t:<tenant>:`. No worker command may start it as a child or
//     continue as one, so no workflow can run a backfill around the outbox
//     row the publish wrote (CI-3 (1), (2)).
//   - field_index.remove.run (ADR-006 D11, PLAN-007 D-7-12): child-only, the
//     tenant's coordinator starts it, so a removal never runs beside a
//     backfill and the relay cannot start one from a forged row.
//
// The deny cases below are denied today (the types are not allow-listed yet)
// and must stay denied once they are, for the same reason: they run now. The
// cases that need the type allow-listed (positive controls, the tenant checks
// on a run of the type) skip until the policy lists the type, so they turn
// on in the PR that adds it (TICK-920 for schema.backfill, TICK-750 for
// field_index.remove.run) with no further change here.

import (
	"testing"

	commandpb "go.temporal.io/api/command/v1"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	taskqueuepb "go.temporal.io/api/taskqueue/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/server/common/authorization"
)

const (
	schemaBackfillType  = "schema.backfill"
	schemaBackfillQueue = "schema-backfill"
	// schemaBackfillA is tenantA's job for one definition and target version
	// (C3: t:<tenant_id>:schema.backfill:<entity_definition_id>:<target_version>).
	schemaBackfillA = "t:tenantA:schema.backfill:9c1d2e3f-4a5b-4c6d-8e7f-0a1b2c3d4e5f:3"
	schemaBackfillB = "t:tenantB:schema.backfill:9c1d2e3f-4a5b-4c6d-8e7f-0a1b2c3d4e5f:3"

	fieldIndexRemoveType = "field_index.remove.run"
	removeRunA           = "t:tenantA:field_index.remove.run:3f0c4a52-7d6e-4b8a-9a51-2c1f0e9d8b7a:2"
	removeRunB           = "t:tenantB:field_index.remove.run:3f0c4a52-7d6e-4b8a-9a51-2c1f0e9d8b7a:2"
)

func schemaBackfillTQ() *taskqueuepb.TaskQueue {
	return &taskqueuepb.TaskQueue{Name: schemaBackfillQueue, Kind: enumspb.TASK_QUEUE_KIND_NORMAL}
}

func schemaBackfillStart(wfID string) *workflowservice.StartWorkflowExecutionRequest {
	return &workflowservice.StartWorkflowExecutionRequest{Namespace: Namespace, WorkflowId: wfID,
		WorkflowType: &commonpb.WorkflowType{Name: schemaBackfillType}, TaskQueue: schemaBackfillTQ()}
}

// requireSchemaBackfillListed skips until schema.backfill is a platform type
// (TICK-920 adds it to platformWorkflowTypes and startOnlyWorkflowTypes).
func requireSchemaBackfillListed(t *testing.T) {
	t.Helper()
	if !inSet(platformWorkflowTypes(), schemaBackfillType) {
		t.Skip("schema.backfill is not allow-listed yet (TICK-920)")
	}
}

// requireRemoveListed skips until field_index.remove.run is a child-only
// platform type (TICK-750).
func requireRemoveListed(t *testing.T) {
	t.Helper()
	if !inSet(childOnlyWorkflowTypes(), fieldIndexRemoveType) {
		t.Skip("field_index.remove.run is not allow-listed yet (TICK-750)")
	}
}

// S29 CI-3 (2): no worker command names schema.backfill. An interpreter run,
// the tenant's field-index coordinator and a schema.backfill run itself
// cannot start a backfill child, in their own tenant, with the tenant
// header, on either queue, nor continue as one. Denied today (unknown type)
// and after TICK-920 (start-only): ReasonWorkflowType both times.
func TestWorkerCommandsCannotNameSchemaBackfill(t *testing.T) {
	child := func(id string, tq *taskqueuepb.TaskQueue) *commandpb.Command {
		cmd := interpreterChild(id, jsonString("tenantA"))
		attrs := cmd.GetStartChildWorkflowExecutionCommandAttributes()
		attrs.WorkflowType.Name, attrs.TaskQueue = schemaBackfillType, tq
		return cmd
	}
	can := func(tq *taskqueuepb.TaskQueue) *commandpb.Command {
		cmd := interpreterContinueAsNew(jsonString("tenantA"))
		attrs := cmd.GetContinueAsNewWorkflowExecutionCommandAttributes()
		attrs.WorkflowType.Name, attrs.TaskQueue = schemaBackfillType, tq
		return cmd
	}
	inherit := can(nil)
	for _, runner := range []string{wfA, coordinatorA, schemaBackfillA} {
		tok := taskToken(t, runner)
		for _, c := range []denyCase{
			{name: runner + " starts a backfill child on schema-backfill", req: respond(tok, child(schemaBackfillA, schemaBackfillTQ())),
				command: "StartChildWorkflowExecution", target: schemaBackfillA},
			{name: runner + " starts a backfill child on nava-tq", req: respond(tok, child(schemaBackfillA, interpreterQueue())),
				command: "StartChildWorkflowExecution", target: schemaBackfillA},
			{name: runner + " continues as a backfill on schema-backfill", req: respond(tok, can(schemaBackfillTQ())),
				command: "ContinueAsNewWorkflowExecution"},
			{name: runner + " continues as a backfill, inherited queue", req: respond(tok, inherit),
				command: "ContinueAsNewWorkflowExecution"},
			// One denied command denies the whole response, after an allowed one.
			{name: runner + " backfill child after an allowed activity", req: respond(tok,
				scheduleActivity(jsonString("tenantA")), child(schemaBackfillA, schemaBackfillTQ())),
				command: "StartChildWorkflowExecution", target: schemaBackfillA, commandIdx: "1"},
		} {
			c.reason, c.workflowID = ReasonWorkflowType, runner
			if c.commandIdx == "" {
				c.commandIdx = "0"
			}
			t.Run(c.name, func(t *testing.T) { requireCommandDeny(t, c) })
		}
	}
}

// S29 CI-3 (1): no identity but nava-relay starts schema.backfill, and the
// relay only under t:<tenant>:. The identity half is denied today and after.
func TestSchemaBackfillStartedByRelayOnly(t *testing.T) {
	start := schemaBackfillStart(schemaBackfillA)
	sws := &workflowservice.SignalWithStartWorkflowExecutionRequest{Namespace: Namespace, WorkflowId: schemaBackfillA,
		SignalName: "s", WorkflowType: &commonpb.WorkflowType{Name: schemaBackfillType}, TaskQueue: schemaBackfillTQ()}
	for _, id := range []Identity{Worker, Core, Operator} {
		t.Run(string(id), func(t *testing.T) {
			requireAPIDeny(t, id, wf+"StartWorkflowExecution", start)
			requireAPIDeny(t, id, wf+"SignalWithStartWorkflowExecution", sws)
		})
	}
	t.Run("relay", func(t *testing.T) {
		requireSchemaBackfillListed(t)
		// Positive control: the relay's start from a tenant A outbox row.
		requireWorkloadAllow(t, Relay, wf+"StartWorkflowExecution", start)
		for _, id := range []string{
			"schema.backfill:9c1d2e3f-4a5b-4c6d-8e7f-0a1b2c3d4e5f:3", "t::schema.backfill:9c1d2e3f-4a5b-4c6d-8e7f-0a1b2c3d4e5f:3",
			"t:tenantA", "", "T:tenantA:schema.backfill:x:3", " t:tenantA:schema.backfill:x:3",
		} {
			t.Run("id "+id, func(t *testing.T) {
				requireWorkloadDeny(t, Relay, wf+"StartWorkflowExecution", schemaBackfillStart(id), ReasonNoTenantPrefix)
			})
		}
	})
}

// S29: the queue is not a side door. Only schema.backfill may use the
// schema-backfill queue: an interpreter or a field-index coordinator started
// there, and a schema.backfill started on a queue that is neither, are
// denied. Denied today (no type or queue is listed) and after.
func TestSchemaBackfillQueueIsNotASideDoor(t *testing.T) {
	for _, c := range []struct {
		name string
		req  *workflowservice.StartWorkflowExecutionRequest
	}{
		{"interpreter on schema-backfill", &workflowservice.StartWorkflowExecutionRequest{Namespace: Namespace, WorkflowId: "t:tenantA:wf:1",
			WorkflowType: &commonpb.WorkflowType{Name: interpreterType}, TaskQueue: schemaBackfillTQ()}},
		{"coordinator on schema-backfill", &workflowservice.StartWorkflowExecutionRequest{Namespace: Namespace, WorkflowId: coordinatorA,
			WorkflowType: &commonpb.WorkflowType{Name: fieldIndexCoordinatorType}, TaskQueue: schemaBackfillTQ()}},
		{"backfill on audit-tq", &workflowservice.StartWorkflowExecutionRequest{Namespace: Namespace, WorkflowId: schemaBackfillA,
			WorkflowType: &commonpb.WorkflowType{Name: schemaBackfillType}, TaskQueue: tqOf("audit-tq", enumspb.TASK_QUEUE_KIND_NORMAL, "")}},
		{"backfill on a look-alike queue", &workflowservice.StartWorkflowExecutionRequest{Namespace: Namespace, WorkflowId: schemaBackfillA,
			WorkflowType: &commonpb.WorkflowType{Name: schemaBackfillType}, TaskQueue: tqOf("schema-backfill ", enumspb.TASK_QUEUE_KIND_NORMAL, "")}},
	} {
		t.Run(c.name, func(t *testing.T) {
			decision, _, _ := authorizeAs(t, Relay, wf+"StartWorkflowExecution", c.req)
			if decision == authorization.DecisionAllow {
				t.Fatalf("relay start %s allowed, want deny", c.name)
			}
		})
	}
}

// S29 CI-3 (3), the Temporal half: a schema.backfill run of tenant A keeps
// every E1 tenant check. It cannot schedule an activity with tenant B's
// header, signal or cancel tenant B's job, or start a child in tenant B.
func TestSchemaBackfillRunKeepsTenantChecks(t *testing.T) {
	requireSchemaBackfillListed(t)
	tok := taskToken(t, schemaBackfillA)
	// Positive control: its own activities, as nava-worker runs them.
	requireCommandAllow(t, respond(tok, scheduleActivity(jsonString("tenantA"))))
	for _, c := range []denyCase{
		{name: "activity as tenant B", req: respond(tok, scheduleActivity(jsonString("tenantB"))),
			reason: ReasonActivityHeader, command: "ScheduleActivityTask"},
		{name: "signal tenant B's job", req: respond(tok, signalExternal(Namespace, schemaBackfillB)),
			reason: ReasonCrossTenant, command: "SignalExternalWorkflowExecution", target: schemaBackfillB},
		{name: "cancel tenant B's job", req: respond(tok, cancelExternal(Namespace, schemaBackfillB)),
			reason: ReasonCrossTenant, command: "RequestCancelExternalWorkflowExecution", target: schemaBackfillB},
		{name: "signal tenant B's coordinator", req: respond(tok, signalExternal(Namespace, "t:tenantB:field_index.coordinator")),
			reason: ReasonCrossTenant, command: "SignalExternalWorkflowExecution", target: "t:tenantB:field_index.coordinator"},
	} {
		c.workflowID, c.commandIdx = schemaBackfillA, "0"
		t.Run(c.name, func(t *testing.T) { requireCommandDeny(t, c) })
	}
}

// S22 P7-13 item 1, the Temporal half: the relay cannot start a removal
// itself, by start or signal-with-start, under any id; only the tenant's
// coordinator starts one (D-7-12). Denied today (unknown type) and after
// TICK-750 (child-only): ReasonWorkflowType both times.
func TestRelayCannotStartRemovalDirectly(t *testing.T) {
	for _, id := range []string{removeRunA, coordinatorA, removeRunB} {
		t.Run(id, func(t *testing.T) {
			requireWorkloadDeny(t, Relay, wf+"StartWorkflowExecution", interpreterStart(id, fieldIndexRemoveType), ReasonWorkflowType)
			requireWorkloadDeny(t, Relay, wf+"SignalWithStartWorkflowExecution", fieldIndexSignalWithStart(id, fieldIndexRemoveType), ReasonWorkflowType)
		})
	}
	// No other identity has the API at all.
	for _, id := range []Identity{Worker, Core, Operator} {
		t.Run(string(id), func(t *testing.T) {
			requireAPIDeny(t, id, wf+"StartWorkflowExecution", interpreterStart(removeRunA, fieldIndexRemoveType))
		})
	}
}

// S22 P7-13 item 1: an interpreter run (tenant A's own business workflow)
// cannot start a removal child in its own tenant; only the coordinator may.
// Denied today and after; the coordinator's positive control is below.
func TestInterpreterCannotStartRemoval(t *testing.T) {
	child := interpreterChild(removeRunA, jsonString("tenantA"))
	child.GetStartChildWorkflowExecutionCommandAttributes().WorkflowType.Name = fieldIndexRemoveType
	decision, _, logs := authorizeWorker(t, respond(taskToken(t, wfA), child))
	if decision == authorization.DecisionAllow {
		t.Fatalf("an interpreter run started %s as a child (log %v); only the tenant's coordinator may", fieldIndexRemoveType, logs)
	}
}

// S22 P7-13 item 1: tenant A's coordinator starts its own removal child, and
// cannot start one in tenant B, with tenant B's header, or on another queue;
// a removal run cannot signal tenant B's coordinator.
func TestCoordinatorRemovalKeepsTenantChecks(t *testing.T) {
	requireRemoveListed(t)
	tok := taskToken(t, coordinatorA)
	child := func(id, header string) *commandpb.Command {
		cmd := interpreterChild(id, jsonString(header))
		cmd.GetStartChildWorkflowExecutionCommandAttributes().WorkflowType.Name = fieldIndexRemoveType
		return cmd
	}
	// Positive control.
	requireCommandAllow(t, respond(tok, child(removeRunA, "tenantA")))
	other := child(removeRunA, "tenantA")
	other.GetStartChildWorkflowExecutionCommandAttributes().TaskQueue = schemaBackfillTQ()
	for _, c := range []denyCase{
		{name: "removal child in tenant B", req: respond(tok, child(removeRunB, "tenantB")),
			reason: ReasonCrossTenant, command: "StartChildWorkflowExecution", workflowID: coordinatorA, target: removeRunB},
		{name: "removal child with tenant B header", req: respond(tok, child(removeRunA, "tenantB")),
			reason: ReasonChildHeader, command: "StartChildWorkflowExecution", workflowID: coordinatorA, target: removeRunA},
		{name: "removal child on schema-backfill", req: respond(tok, other),
			reason: ReasonTaskQueue, command: "StartChildWorkflowExecution", workflowID: coordinatorA, target: removeRunA},
		{name: "removal run signals tenant B's coordinator", req: respond(taskToken(t, removeRunA), signalExternal(Namespace, "t:tenantB:field_index.coordinator")),
			reason: ReasonCrossTenant, command: "SignalExternalWorkflowExecution", workflowID: removeRunA, target: "t:tenantB:field_index.coordinator"},
		{name: "removal run activity as tenant B", req: respond(taskToken(t, removeRunA), scheduleActivity(jsonString("tenantB"))),
			reason: ReasonActivityHeader, command: "ScheduleActivityTask", workflowID: removeRunA},
	} {
		c.commandIdx = "0"
		t.Run(c.name, func(t *testing.T) { requireCommandDeny(t, c) })
	}
}

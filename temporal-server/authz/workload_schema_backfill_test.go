package authz

// TICK-920 (PLAN-010 P10-03, ADR-009 D2; RFC-021 CI-3 (1) and (2)): the
// schema backfill runs as workflow type schema.backfill on its own queue,
// schema-backfill. nava-relay starts it under a
// `t:<tenant>:schema.backfill:<entity_definition_id>:<target_version>` id,
// and nothing else does. It is start-only: no worker command may start it as
// a child or continue as it.

import (
	"testing"

	commandpb "go.temporal.io/api/command/v1"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	taskqueuepb "go.temporal.io/api/taskqueue/v1"
	"go.temporal.io/api/workflowservice/v1"
)

const (
	schemaBackfillType  = "schema.backfill"
	schemaBackfillQueue = "schema-backfill"
	// schemaBackfillA is tenantA's job for one definition at version 3.
	schemaBackfillA = "t:tenantA:schema.backfill:0192aaaa-0000-7000-8000-0000000000d1:3"
)

func schemaBackfillTQ() *taskqueuepb.TaskQueue {
	return &taskqueuepb.TaskQueue{Name: schemaBackfillQueue, Kind: enumspb.TASK_QUEUE_KIND_NORMAL}
}

func schemaBackfillStart(wfID string) *workflowservice.StartWorkflowExecutionRequest {
	return &workflowservice.StartWorkflowExecutionRequest{Namespace: Namespace, WorkflowId: wfID,
		WorkflowType: &commonpb.WorkflowType{Name: schemaBackfillType}, TaskQueue: schemaBackfillTQ()}
}

// CI-3 (1), positive: the relay starts a job on schema-backfill, and the
// worker polls that queue for workflow and activity tasks.
func TestRelayStartsSchemaBackfill(t *testing.T) {
	requireWorkloadAllow(t, Relay, wf+"StartWorkflowExecution", schemaBackfillStart(schemaBackfillA))
	requireWorkloadAllow(t, Worker, wf+"PollWorkflowTaskQueue",
		&workflowservice.PollWorkflowTaskQueueRequest{Namespace: Namespace, TaskQueue: schemaBackfillTQ()})
	requireWorkloadAllow(t, Worker, wf+"PollActivityTaskQueue",
		&workflowservice.PollActivityTaskQueueRequest{Namespace: Namespace, TaskQueue: schemaBackfillTQ()})
}

// CI-3 (1), negative: a start with no `t:<tenant>:` prefix is denied.
func TestSchemaBackfillNeedsTenantPrefix(t *testing.T) {
	for _, id := range []string{
		"schema.backfill:0192aaaa-0000-7000-8000-0000000000d1:3", "t::schema.backfill:0192aaaa-0000-7000-8000-0000000000d1:3",
		"t:tenantA", "",
	} {
		t.Run(id, func(t *testing.T) {
			requireWorkloadDeny(t, Relay, wf+"StartWorkflowExecution", schemaBackfillStart(id), ReasonNoTenantPrefix)
		})
	}
}

// CI-3 (1), negative: no identity but the relay may start schema.backfill.
func TestSchemaBackfillStartedByRelayOnly(t *testing.T) {
	for _, id := range []Identity{Worker, Core, Operator} {
		t.Run(string(id), func(t *testing.T) {
			requireAPIDeny(t, id, wf+"StartWorkflowExecution", schemaBackfillStart(schemaBackfillA))
		})
	}
}

// Negative: the new queue is exact match, NORMAL only.
func TestSchemaBackfillOtherQueuesDenied(t *testing.T) {
	for _, c := range []struct {
		name, tq string
		kind     enumspb.TaskQueueKind
		normal   string
		reason   string
	}{
		{"look-alike queue", "schema-backfill ", enumspb.TASK_QUEUE_KIND_NORMAL, "", ReasonTaskQueue},
		{"underscore queue", "schema_backfill", enumspb.TASK_QUEUE_KIND_NORMAL, "", ReasonTaskQueue},
		{"sticky kind", schemaBackfillQueue, enumspb.TASK_QUEUE_KIND_STICKY, schemaBackfillQueue, ReasonTaskQueueKind},
	} {
		t.Run(c.name, func(t *testing.T) {
			start := schemaBackfillStart(schemaBackfillA)
			start.TaskQueue = tqOf(c.tq, c.kind, c.normal)
			requireWorkloadDeny(t, Relay, wf+"StartWorkflowExecution", start, c.reason)
		})
	}
}

// CI-3 (2): a worker command may not name schema.backfill. A child or a
// continue-as-new of it, otherwise a valid tenantA command with the tenant
// header, is denied as workflow_type_not_allowed: no workflow can run a job
// around the relay start of a row written under RLS.
func TestWorkerCommandsCannotNameSchemaBackfill(t *testing.T) {
	tok := taskToken(t, wfA)
	jobTok := taskToken(t, schemaBackfillA)

	child := func(workflowID string) *commandpb.Command {
		cmd := interpreterChild(workflowID, jsonString("tenantA"))
		cmd.GetStartChildWorkflowExecutionCommandAttributes().WorkflowType.Name = schemaBackfillType
		return cmd
	}
	can := func() *commandpb.Command {
		cmd := interpreterContinueAsNew(jsonString("tenantA"))
		cmd.GetContinueAsNewWorkflowExecutionCommandAttributes().WorkflowType.Name = schemaBackfillType
		return cmd
	}
	other := "t:tenantA:schema.backfill:0192aaaa-0000-7000-8000-0000000000d1:4"
	for _, c := range []denyCase{
		{name: "interpreter starts a job child", req: respond(tok, child(other)),
			reason: ReasonWorkflowType, command: "StartChildWorkflowExecution", workflowID: wfA, target: other, commandIdx: "0"},
		{name: "job starts a job child", req: respond(jobTok, child(other)),
			reason: ReasonWorkflowType, command: "StartChildWorkflowExecution", workflowID: schemaBackfillA, target: other, commandIdx: "0"},
		{name: "interpreter continues as a job", req: respond(tok, can()),
			reason: ReasonWorkflowType, command: "ContinueAsNewWorkflowExecution", workflowID: wfA, commandIdx: "0"},
		{name: "job continues as a job", req: respond(jobTok, can()),
			reason: ReasonWorkflowType, command: "ContinueAsNewWorkflowExecution", workflowID: schemaBackfillA, commandIdx: "0"},
	} {
		t.Run(c.name, func(t *testing.T) { requireCommandDeny(t, c) })
	}
	_, _, logs := authorizeWorker(t, respond(tok, can()))
	if len(logs) != 1 || logs[0]["detail"] != `workflow type "schema.backfill" is start-only` {
		t.Fatalf("log detail = %v, want the start-only detail", logs)
	}
}

// Positive: a job's ordinary commands (activities with the tenant header,
// timers, completion) stay allowed.
func TestSchemaBackfillRunKeepsOrdinaryCommands(t *testing.T) {
	jobTok := taskToken(t, schemaBackfillA)
	requireCommandAllow(t, respond(jobTok, scheduleActivity(jsonString("tenantA"))))
	requireCommandAllow(t, respond(jobTok, &commandpb.Command{CommandType: enumspb.COMMAND_TYPE_START_TIMER}))
	requireCommandAllow(t, respond(jobTok, &commandpb.Command{CommandType: enumspb.COMMAND_TYPE_COMPLETE_WORKFLOW_EXECUTION}))
}

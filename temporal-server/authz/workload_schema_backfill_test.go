package authz

// TICK-920 (PLAN-010 P10-03, ADR-009 D2; RFC-021 CI-3 (1) and (2)): the
// schema backfill runs as workflow type schema.backfill on its own queue,
// schema-backfill. The S29 adversarial suite (adversarial_s22_s29_test.go,
// TICK-753) holds the shared names and the relay-only, start-only and
// side-door cases; these are the workload cases it does not cover.

import (
	"testing"

	commandpb "go.temporal.io/api/command/v1"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/workflowservice/v1"
)

// CI-3 (1), positive: the relay starts a job on schema-backfill, and the
// worker polls that queue for workflow and activity tasks.
func TestRelayStartsSchemaBackfill(t *testing.T) {
	requireWorkloadAllow(t, Relay, wf+"StartWorkflowExecution", schemaBackfillStart(schemaBackfillA))
	requireWorkloadAllow(t, Worker, wf+"PollWorkflowTaskQueue",
		&workflowservice.PollWorkflowTaskQueueRequest{Namespace: Namespace, TaskQueue: schemaBackfillTQ()})
	requireWorkloadAllow(t, Worker, wf+"PollActivityTaskQueue",
		&workflowservice.PollActivityTaskQueueRequest{Namespace: Namespace, TaskQueue: schemaBackfillTQ()})
}

// Negative: schema.backfill on nava-tq is denied: each type runs on its own
// queue (workflowTypeQueue), so a job never runs outside the worker that caps
// it. The new queue is exact match, NORMAL only.
func TestSchemaBackfillOtherQueuesDenied(t *testing.T) {
	for _, c := range []struct {
		name, tq string
		kind     enumspb.TaskQueueKind
		normal   string
		reason   string
	}{
		{"nava-tq", "nava-tq", enumspb.TASK_QUEUE_KIND_NORMAL, "", ReasonTaskQueue},
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

// Negative: no worker command puts another type on schema-backfill. An
// interpreter child or continue-as-new that names the queue is denied as
// task_queue_not_allowed; the same commands on nava-tq stay allowed.
func TestWorkerCommandsCannotUseTheSchemaBackfillQueue(t *testing.T) {
	tok := taskToken(t, wfA)
	child := interpreterChild("t:tenantA:wf:1:child", jsonString("tenantA"))
	child.GetStartChildWorkflowExecutionCommandAttributes().TaskQueue = schemaBackfillTQ()
	can := interpreterContinueAsNew(jsonString("tenantA"))
	can.GetContinueAsNewWorkflowExecutionCommandAttributes().TaskQueue = schemaBackfillTQ()
	for _, c := range []denyCase{
		{name: "interpreter child on schema-backfill", req: respond(tok, child), reason: ReasonTaskQueue,
			command: "StartChildWorkflowExecution", workflowID: wfA, target: "t:tenantA:wf:1:child", commandIdx: "0"},
		{name: "interpreter continues on schema-backfill", req: respond(tok, can), reason: ReasonTaskQueue,
			command: "ContinueAsNewWorkflowExecution", workflowID: wfA, commandIdx: "0"},
	} {
		t.Run(c.name, func(t *testing.T) { requireCommandDeny(t, c) })
	}
	requireCommandAllow(t, respond(tok, interpreterChild("t:tenantA:wf:1:child", jsonString("tenantA"))))
	requireCommandAllow(t, respond(tok, interpreterContinueAsNew(jsonString("tenantA"))))
}

// Positive: a job's ordinary commands (activities with the tenant header,
// timers, completion) stay allowed.
func TestSchemaBackfillRunKeepsOrdinaryCommands(t *testing.T) {
	jobTok := taskToken(t, schemaBackfillA)
	requireCommandAllow(t, respond(jobTok, scheduleActivity(jsonString("tenantA"))))
	requireCommandAllow(t, respond(jobTok, &commandpb.Command{CommandType: enumspb.COMMAND_TYPE_START_TIMER}))
	requireCommandAllow(t, respond(jobTok, &commandpb.Command{CommandType: enumspb.COMMAND_TYPE_COMPLETE_WORKFLOW_EXECUTION}))
}

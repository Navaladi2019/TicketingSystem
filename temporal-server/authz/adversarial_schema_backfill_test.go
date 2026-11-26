package authz

// Security review of TICK-920 (PR #924): schema.backfill joins the platform
// types and schema-backfill the platform queues. Threat model: a workflow task
// of tenantA (its task token names a t:tenantA: id) tries to reach tenantB's
// schema backfill job, or to put tenantB work on the new queue. Each case must
// be denied as cross-tenant or as a header mismatch, whatever queue it names.

import (
	"testing"

	commandpb "go.temporal.io/api/command/v1"
	enumspb "go.temporal.io/api/enums/v1"
	taskqueuepb "go.temporal.io/api/taskqueue/v1"
)

// schemaBackfillB, tenantB's job, is declared with the S29 suite
// (adversarial_s22_s29_test.go).

// activityOn schedules an activity on queue tq with the tenant header set to
// tenant.
func activityOn(tenant, tq string) *commandpb.Command {
	cmd := scheduleActivity(jsonString(tenant))
	cmd.GetScheduleActivityTaskCommandAttributes().TaskQueue =
		&taskqueuepb.TaskQueue{Name: tq, Kind: enumspb.TASK_QUEUE_KIND_NORMAL}
	return cmd
}

func TestAdversarialSchemaBackfillCrossTenant(t *testing.T) {
	tok := taskToken(t, wfA)                // a tenantA interpreter task
	jobTok := taskToken(t, schemaBackfillA) // tenantA's own backfill job
	child := interpreterChild(schemaBackfillB, jsonString("tenantA"))
	child.GetStartChildWorkflowExecutionCommandAttributes().WorkflowType.Name = schemaBackfillType

	for _, c := range []denyCase{
		{name: "interpreter signals tenantB job", req: respond(tok, signalExternal("", schemaBackfillB)),
			reason: ReasonCrossTenant, command: "SignalExternalWorkflowExecution", workflowID: wfA, target: schemaBackfillB, commandIdx: "0"},
		{name: "interpreter cancels tenantB job", req: respond(tok, cancelExternal("", schemaBackfillB)),
			reason: ReasonCrossTenant, command: "RequestCancelExternalWorkflowExecution", workflowID: wfA, target: schemaBackfillB, commandIdx: "0"},
		{name: "job signals tenantB job", req: respond(jobTok, signalExternal("", schemaBackfillB)),
			reason: ReasonCrossTenant, command: "SignalExternalWorkflowExecution", workflowID: schemaBackfillA, target: schemaBackfillB, commandIdx: "0"},
		{name: "job cancels tenantB job", req: respond(jobTok, cancelExternal(Namespace, schemaBackfillB)),
			reason: ReasonCrossTenant, command: "RequestCancelExternalWorkflowExecution", workflowID: schemaBackfillA, target: schemaBackfillB, commandIdx: "0"},
		{name: "interpreter starts tenantB job as child", req: respond(tok, child),
			reason: ReasonCrossTenant, command: "StartChildWorkflowExecution", workflowID: wfA, target: schemaBackfillB, commandIdx: "0"},
		{name: "job schedules a tenantB activity on schema-backfill", req: respond(jobTok, activityOn("tenantB", schemaBackfillQueue)),
			reason: ReasonActivityHeader, command: "ScheduleActivityTask", workflowID: schemaBackfillA, commandIdx: "0"},
		{name: "interpreter schedules a tenantB activity on schema-backfill", req: respond(tok, activityOn("tenantB", schemaBackfillQueue)),
			reason: ReasonActivityHeader, command: "ScheduleActivityTask", workflowID: wfA, commandIdx: "0"},
		{name: "job schedules a tenantB activity on nava-tq", req: respond(jobTok, activityOn("tenantB", "nava-tq")),
			reason: ReasonActivityHeader, command: "ScheduleActivityTask", workflowID: schemaBackfillA, commandIdx: "0"},
	} {
		t.Run(c.name, func(t *testing.T) { requireCommandDeny(t, c) })
	}
}

// The relay may not start or signal a job without a tenant prefix, and the
// tenant segment is parsed exactly: a tenant id that is a prefix of another
// ("tenantA" vs "tenantAB") does not reach the other's job.
func TestAdversarialSchemaBackfillPrefixConfusion(t *testing.T) {
	tok := taskToken(t, schemaBackfillA)
	other := "t:tenantAB:schema.backfill:0192aaaa-0000-7000-8000-0000000000d1:3"
	requireCommandDeny(t, denyCase{name: "prefix tenant", req: respond(tok, signalExternal("", other)),
		reason: ReasonCrossTenant, command: "SignalExternalWorkflowExecution", workflowID: schemaBackfillA, target: other, commandIdx: "0"})
}

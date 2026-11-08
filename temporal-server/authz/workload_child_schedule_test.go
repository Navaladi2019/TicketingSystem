package authz

// TICK-576 (ADR-004 R16/R18, ADR-001-A1 D9c with erratum E1): a worker's
// StartChildWorkflowExecution may not ask the server to create runs on its
// own. A CronSchedule makes the server start a new child run on every tick and
// a workflow RetryPolicy makes it start one after each failure; no workflow
// task of the parent ever sees those commands again. The child stays in the
// task's tenant (E1 checks the target, header and NavaTenantId first), so this
// is defence in depth, not an isolation rule: until now only the in-process
// workerguard (internal/workerguard ExecuteChildWorkflow) denied a child cron,
// and nothing denied a child retry policy. The relay's starts have the same
// rule (TICK-573, workload_start_schedule_test.go); continue-as-new's cron has
// been denied since E1 (continue_as_new_cron).

import (
	"strconv"
	"strings"
	"testing"

	commandpb "go.temporal.io/api/command/v1"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/server/common/authorization"
	"google.golang.org/protobuf/types/known/durationpb"
)

const childCmd = "StartChildWorkflowExecution"

// childWithSchedule is tenantA's well-formed child of wfA's tenant, on the
// platform type and queue, with cron and retry set.
func childWithSchedule(cron string, retry *commonpb.RetryPolicy) *commandpb.Command {
	cmd := startChild(Namespace, "t:tenantA:child:1")
	a := cmd.GetStartChildWorkflowExecutionCommandAttributes()
	a.TaskQueue = interpreterQueue()
	a.CronSchedule, a.RetryPolicy = cron, retry
	return cmd
}

// requireChildDenyDetail asserts the full command deny and the log's detail.
func requireChildDenyDetail(t *testing.T, req any, idx, reason, target, detail string) {
	t.Helper()
	requireCommandDeny(t, denyCase{req: req, reason: reason, command: childCmd,
		workflowID: wfA, target: target, commandIdx: idx})
	_, _, logs := authorizeWorker(t, req)
	if got := logs[0]["detail"]; got != detail {
		t.Errorf("log detail = %q, want %q", got, detail)
	}
}

// Negative: any non-empty cron schedule, blank included, so the rule does not
// depend on the server's cron parser.
func TestChildStartCronScheduleDenied(t *testing.T) {
	tok := taskToken(t, wfA)
	for _, cron := range []string{"* * * * *", "@every 1m", " "} {
		t.Run(cron, func(t *testing.T) {
			requireChildDenyDetail(t, respond(tok, childWithSchedule(cron, nil)),
				"0", ReasonChildCron, "t:tenantA:child:1", "cron schedule "+strconv.Quote(cron))
		})
	}
}

// The logged cron string is quoted and bounded, like every client-supplied
// string in a deny log.
func TestChildStartCronScheduleDetailTruncated(t *testing.T) {
	long := "0 0 * * * " + strings.Repeat("x", 4*maxLoggedName)
	req := respond(taskToken(t, wfA), childWithSchedule(long, nil))
	requireChildDenyDetail(t, req, "0", ReasonChildCron, "t:tenantA:child:1",
		"cron schedule "+strconv.Quote(truncate(long, maxLoggedName)))
}

// Negative: any workflow retry policy, an empty one included (the server fills
// in its retrying defaults) and one with MaximumAttempts 1.
func TestChildStartRetryPolicyDenied(t *testing.T) {
	tok := taskToken(t, wfA)
	for name, rp := range map[string]*commonpb.RetryPolicy{
		"empty":       {},
		"unlimited":   {InitialInterval: durationpb.New(1e9)},
		"one attempt": {MaximumAttempts: 1},
	} {
		t.Run(name, func(t *testing.T) {
			requireChildDenyDetail(t, respond(tok, childWithSchedule("", rp)),
				"0", ReasonChildRetryPolicy, "t:tenantA:child:1", "workflow retry policy set")
		})
	}
}

// Both set: the cron reason is reported, as on the relay's starts.
func TestChildStartCronReportedBeforeRetryPolicy(t *testing.T) {
	req := respond(taskToken(t, wfA), childWithSchedule("* * * * *", &commonpb.RetryPolicy{}))
	requireChildDenyDetail(t, req, "0", ReasonChildCron, "t:tenantA:child:1", `cron schedule "* * * * *"`)
}

// One bad child denies the whole response, wherever it sits.
func TestChildStartScheduleDeniesWholeResponse(t *testing.T) {
	req := respond(taskToken(t, wfA),
		childWithSchedule("", nil), scheduleActivity(jsonString("tenantA")), childWithSchedule("", &commonpb.RetryPolicy{}))
	requireChildDenyDetail(t, req, "2", ReasonChildRetryPolicy, "t:tenantA:child:1", "workflow retry policy set")
}

// Positive: the children the worker sends (no cron, no retry policy) still
// pass, for the platform type and with an inherited or explicit queue.
func TestChildStartWithoutScheduleAllowed(t *testing.T) {
	tok := taskToken(t, wfA)
	requireCommandAllow(t, respond(tok, childWithSchedule("", nil)))
	inherited := childWithSchedule("", nil)
	inherited.GetStartChildWorkflowExecutionCommandAttributes().TaskQueue = nil
	requireCommandAllow(t, respond(tok, inherited))
}

// The schedule rules sit after the E1 tenant rules and the D9c workload rules,
// so a child that breaks one of those still reports it, not the cron or retry
// reason. A cross-tenant child with a cron is a cross-tenant deny first.
func TestChildStartScheduleAfterTenantAndWorkloadRules(t *testing.T) {
	tok := taskToken(t, wfA)
	retry := &commonpb.RetryPolicy{}
	cases := []struct {
		name   string
		mutate func(a *commandpb.StartChildWorkflowExecutionCommandAttributes)
		reason string
		target string
	}{
		{"cross tenant", func(a *commandpb.StartChildWorkflowExecutionCommandAttributes) { a.WorkflowId = wfB },
			ReasonCrossTenant, wfB},
		{"other namespace", func(a *commandpb.StartChildWorkflowExecutionCommandAttributes) { a.Namespace = "victim" },
			ReasonCommandNamespace, "t:tenantA:child:1"},
		{"header other tenant", func(a *commandpb.StartChildWorkflowExecutionCommandAttributes) {
			a.Header = tenantHeader(jsonString("tenantB"))
		}, ReasonChildHeader, "t:tenantA:child:1"},
		{"NavaTenantId other tenant", func(a *commandpb.StartChildWorkflowExecutionCommandAttributes) {
			a.SearchAttributes = tenantSearchAttrs(jsonString("tenantB"))
		}, ReasonTenantSearchAttr, "t:tenantA:child:1"},
		{"type off the list", func(a *commandpb.StartChildWorkflowExecutionCommandAttributes) {
			a.WorkflowType = &commonpb.WorkflowType{Name: "temporal-sys-batch-workflow"}
		}, ReasonWorkflowType, "t:tenantA:child:1"},
		{"start-only type", func(a *commandpb.StartChildWorkflowExecutionCommandAttributes) {
			a.WorkflowType = &commonpb.WorkflowType{Name: auditSealType}
		}, ReasonWorkflowType, "t:tenantA:child:1"},
		{"queue off the list", func(a *commandpb.StartChildWorkflowExecutionCommandAttributes) {
			a.TaskQueue = tqOf("temporal-sys-per-ns-tq", enumspb.TASK_QUEUE_KIND_NORMAL, "")
		}, ReasonTaskQueue, "t:tenantA:child:1"},
		{"sticky queue", func(a *commandpb.StartChildWorkflowExecutionCommandAttributes) {
			a.TaskQueue = tqOf(platformQueue(), enumspb.TASK_QUEUE_KIND_STICKY, platformQueue())
		}, ReasonTaskQueueKind, "t:tenantA:child:1"},
	}
	for _, c := range cases {
		for _, sched := range []struct {
			name  string
			cron  string
			retry *commonpb.RetryPolicy
		}{{"cron", "* * * * *", nil}, {"retry", "", retry}} {
			t.Run(c.name+"/"+sched.name, func(t *testing.T) {
				cmd := childWithSchedule(sched.cron, sched.retry)
				c.mutate(cmd.GetStartChildWorkflowExecutionCommandAttributes())
				requireCommandDeny(t, denyCase{req: respond(tok, cmd), reason: c.reason, command: childCmd,
					workflowID: wfA, target: c.target, commandIdx: "0"})
			})
		}
	}
}

// The deny never reaches the caller and is not a gate deny: a child cron from
// a task of tenantA is a command deny, recorded once.
func TestChildStartCronNotAGateDeny(t *testing.T) {
	decision, recs, _ := authorizeWorker(t, respond(taskToken(t, wfA), childWithSchedule("* * * * *", nil)))
	if decision != authorization.DecisionDeny {
		t.Fatalf("decision = %v, want deny", decision)
	}
	if len(recs) != 1 || recs[0].Tags["reason"] == ReasonAPI {
		t.Fatalf("want one command deny recording, got %v", recs)
	}
}

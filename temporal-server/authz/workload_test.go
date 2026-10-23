package authz

// D9 workload allow-list (ADR-001-A1, RFC-002 §8 CI-I (3)): the request checks
// of D9a and D9b, the D9c command checks not covered by the adversarial files,
// and the shape of a workload deny's metric and log line.

import (
	"strings"
	"testing"

	commandpb "go.temporal.io/api/command/v1"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	taskqueuepb "go.temporal.io/api/taskqueue/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/server/common/authorization"
	"go.temporal.io/server/common/metrics/metricstest"
)

const sdkSticky = "worker-0:6f1c2a7e-3b4d-4c5e-8f90-a1b2c3d4e5f6"

// authorizeAs runs one request through a fresh Authorizer as id and returns the
// decision, the deny recordings and the log entries.
func authorizeAs(t *testing.T, id Identity, api string, req any) (authorization.Decision, []*metricstest.CapturedRecording, []map[string]string) {
	t.Helper()
	capture := metricstest.NewCaptureHandler()
	rec := capture.StartCapture()
	defer capture.StopCapture(rec)
	logger := &captureLogger{}
	res, err := NewAuthorizer(logger, capture).Authorize(workerCtx(), claimsFor(id), &authorization.CallTarget{
		APIName: api, Namespace: Namespace, Request: req,
	})
	if err != nil {
		t.Fatalf("Authorize returned error %v", err)
	}
	if res.Reason != "" {
		t.Errorf("deny reason must not reach the caller, got %q", res.Reason)
	}
	return res.Decision, rec.Snapshot()[DeniedMetric], logger.entries
}

// requireWorkloadDeny asserts a deny with reason, one bounded metric recording
// and one alert=true error log line.
func requireWorkloadDeny(t *testing.T, id Identity, api string, req any, reason string) {
	t.Helper()
	decision, recs, logs := authorizeAs(t, id, api, req)
	if decision != authorization.DecisionDeny {
		t.Fatalf("decision = %v, want deny", decision)
	}
	if len(recs) != 1 || len(logs) != 1 {
		t.Fatalf("want 1 metric and 1 log line, got %d and %d", len(recs), len(logs))
	}
	want := map[string]string{"identity": string(id), "api": api, "namespace": Namespace, "reason": reason, "command": commandNone}
	if len(recs[0].Tags) != len(want) {
		t.Errorf("metric tags = %v, want exactly %v", recs[0].Tags, want)
	}
	for k, v := range want {
		if recs[0].Tags[k] != v {
			t.Errorf("metric tag %s = %q, want %q", k, recs[0].Tags[k], v)
		}
		if logs[0][k] != v {
			t.Errorf("log field %s = %q, want %q", k, logs[0][k], v)
		}
	}
	for k, v := range recs[0].Tags {
		if strings.HasPrefix(v, "t:") || strings.Contains(v, "temporal-sys") || strings.Contains(v, "victim") || strings.Contains(v, "worker-0") {
			t.Errorf("metric tag %s = %q leaks a workflow id, type or queue", k, v)
		}
	}
	if logs[0]["level"] != "error" || logs[0]["alert"] != "true" {
		t.Errorf("log level=%q alert=%q, want error and true", logs[0]["level"], logs[0]["alert"])
	}
}

func requireWorkloadAllow(t *testing.T, id Identity, api string, req any) {
	t.Helper()
	decision, recs, logs := authorizeAs(t, id, api, req)
	if decision != authorization.DecisionAllow {
		t.Fatalf("decision = %v, want allow (log %v)", decision, logs)
	}
	if len(recs) != 0 || len(logs) != 0 {
		t.Errorf("allowed request recorded a deny: metrics %d, logs %v", len(recs), logs)
	}
}

func tqOf(name string, kind enumspb.TaskQueueKind, normal string) *taskqueuepb.TaskQueue {
	return &taskqueuepb.TaskQueue{Name: name, Kind: kind, NormalName: normal}
}

func TestRelayStartWorkload(t *testing.T) {
	start := func(wfID, wt string, tq *taskqueuepb.TaskQueue) *workflowservice.StartWorkflowExecutionRequest {
		return &workflowservice.StartWorkflowExecutionRequest{Namespace: Namespace, WorkflowId: wfID,
			WorkflowType: &commonpb.WorkflowType{Name: wt}, TaskQueue: tq}
	}
	normal := tqOf(platformQueue(), enumspb.TASK_QUEUE_KIND_NORMAL, "")
	api := wf + "StartWorkflowExecution"

	requireWorkloadAllow(t, Relay, api, start("t:A:x", platformType(), normal))
	requireWorkloadAllow(t, Relay, api, start("t:A:x", platformType(), tqOf(platformQueue(), enumspb.TASK_QUEUE_KIND_UNSPECIFIED, "")))

	for name, c := range map[string]struct {
		req    *workflowservice.StartWorkflowExecutionRequest
		reason string
	}{
		"empty workflow type":      {start("t:A:x", "", normal), ReasonWorkflowType},
		"nil task queue":           {start("t:A:x", platformType(), nil), ReasonTaskQueue},
		"platform queue as sticky": {start("t:A:x", platformType(), tqOf(platformQueue(), enumspb.TASK_QUEUE_KIND_STICKY, platformQueue())), ReasonTaskQueueKind},
		"partition of the queue":   {start("t:A:x", platformType(), tqOf("/_sys/nava-tq/1", enumspb.TASK_QUEUE_KIND_NORMAL, "")), ReasonTaskQueue},
		"no tenant prefix":         {start("victim", platformType(), normal), ReasonNoTenantPrefix},
		"empty tenant":             {start("t::x", platformType(), normal), ReasonNoTenantPrefix},
	} {
		t.Run(name, func(t *testing.T) { requireWorkloadDeny(t, Relay, api, c.req, c.reason) })
	}

	sws := &workflowservice.SignalWithStartWorkflowExecutionRequest{Namespace: Namespace, WorkflowId: "t:A:x", SignalName: "s",
		WorkflowType: &commonpb.WorkflowType{Name: platformType()}, TaskQueue: normal}
	requireWorkloadAllow(t, Relay, wf+"SignalWithStartWorkflowExecution", sws)
	sws.WorkflowId = "temporal-sys-scheduler:x"
	requireWorkloadDeny(t, Relay, wf+"SignalWithStartWorkflowExecution", sws, ReasonNoTenantPrefix)
}

// CI-I (3): a relay signal to a system workflow id is denied.
func TestRelaySignalNeedsTenantPrefix(t *testing.T) {
	signal := func(wfID string) *workflowservice.SignalWorkflowExecutionRequest {
		return &workflowservice.SignalWorkflowExecutionRequest{Namespace: Namespace, SignalName: "s",
			WorkflowExecution: &commonpb.WorkflowExecution{WorkflowId: wfID}}
	}
	api := wf + "SignalWorkflowExecution"
	requireWorkloadAllow(t, Relay, api, signal("t:A:ticket:1"))
	requireWorkloadDeny(t, Relay, api, signal("temporal-sys-scheduler:x"), ReasonNoTenantPrefix)
	requireWorkloadDeny(t, Relay, api, signal(""), ReasonNoTenantPrefix)
	requireWorkloadDeny(t, Relay, api, &workflowservice.SignalWorkflowExecutionRequest{Namespace: Namespace}, ReasonNoTenantPrefix)
}

func TestWorkerPollWorkload(t *testing.T) {
	wft := func(tq *taskqueuepb.TaskQueue) any {
		return &workflowservice.PollWorkflowTaskQueueRequest{Namespace: Namespace, TaskQueue: tq}
	}
	act := func(tq *taskqueuepb.TaskQueue) any {
		return &workflowservice.PollActivityTaskQueueRequest{Namespace: Namespace, TaskQueue: tq}
	}
	pollWF, pollAct := wf+"PollWorkflowTaskQueue", wf+"PollActivityTaskQueue"

	requireWorkloadAllow(t, Worker, pollWF, wft(tqOf(platformQueue(), enumspb.TASK_QUEUE_KIND_NORMAL, "")))
	requireWorkloadAllow(t, Worker, pollWF, wft(tqOf(sdkSticky, enumspb.TASK_QUEUE_KIND_STICKY, platformQueue())))
	requireWorkloadAllow(t, Worker, pollAct, act(tqOf(platformQueue(), enumspb.TASK_QUEUE_KIND_UNSPECIFIED, "")))

	for name, c := range map[string]struct {
		api    string
		req    any
		reason string
	}{
		"normal off-list queue":      {pollWF, wft(tqOf("other-tq", enumspb.TASK_QUEUE_KIND_NORMAL, "")), ReasonTaskQueue},
		"nil task queue":             {pollWF, wft(nil), ReasonTaskQueue},
		"sticky empty NormalName":    {pollWF, wft(tqOf(sdkSticky, enumspb.TASK_QUEUE_KIND_STICKY, "")), ReasonStickyQueue},
		"sticky off-list NormalName": {pollWF, wft(tqOf(sdkSticky, enumspb.TASK_QUEUE_KIND_STICKY, "temporal-sys-per-ns-tq")), ReasonStickyQueue},
		"sticky named as platform":   {pollWF, wft(tqOf(platformQueue(), enumspb.TASK_QUEUE_KIND_STICKY, platformQueue())), ReasonStickyQueue},
		"sticky uppercase uuid":      {pollWF, wft(tqOf("w:6F1C2A7E-3B4D-4C5E-8F90-A1B2C3D4E5F6", enumspb.TASK_QUEUE_KIND_STICKY, platformQueue())), ReasonStickyQueue},
		"sticky with slash":          {pollWF, wft(tqOf("/_sys/x:"+sdkSticky[9:], enumspb.TASK_QUEUE_KIND_STICKY, platformQueue())), ReasonStickyQueue},
		"activity sticky (CI-I 3)":   {pollAct, act(tqOf(sdkSticky, enumspb.TASK_QUEUE_KIND_STICKY, platformQueue())), ReasonTaskQueueKind},
		"activity off-list queue":    {pollAct, act(tqOf("temporal-sys-per-ns-tq", enumspb.TASK_QUEUE_KIND_NORMAL, "")), ReasonTaskQueue},
		"unknown kind":               {pollWF, wft(tqOf(platformQueue(), enumspb.TaskQueueKind(99), "")), ReasonTaskQueueKind},
		"request of the wrong type":  {pollWF, act(tqOf(platformQueue(), enumspb.TASK_QUEUE_KIND_NORMAL, "")), ReasonMalformedRequest},
		"nil request":                {pollAct, nil, ReasonMalformedRequest},
		"typed nil request":          {pollWF, (*workflowservice.PollWorkflowTaskQueueRequest)(nil), ReasonTaskQueue},
	} {
		t.Run(name, func(t *testing.T) { requireWorkloadDeny(t, Worker, c.api, c.req, c.reason) })
	}
	// A missing or mistyped request body fails closed for the relay checks too.
	for _, api := range []string{wf + "StartWorkflowExecution", wf + "SignalWithStartWorkflowExecution", wf + "SignalWorkflowExecution"} {
		requireWorkloadDeny(t, Relay, api, nil, ReasonMalformedRequest)
	}
}

// System is exempt from D9, as from the API allow-list.
func TestSystemExemptFromWorkloadAllowList(t *testing.T) {
	requireWorkloadAllow(t, System, wf+"PollActivityTaskQueue", &workflowservice.PollActivityTaskQueueRequest{
		Namespace: Namespace, TaskQueue: tqOf("temporal-sys-per-ns-tq", enumspb.TASK_QUEUE_KIND_NORMAL, "")})
	requireWorkloadAllow(t, System, wf+"StartWorkflowExecution", &workflowservice.StartWorkflowExecutionRequest{
		Namespace: Namespace, WorkflowId: "temporal-sys-batch:x",
		WorkflowType: &commonpb.WorkflowType{Name: "temporal-sys-batch-workflow"}, TaskQueue: tqOf("temporal-sys-per-ns-tq", 0, "")})
}

// D9c: children and continue-as-new must name a platform type (CI-I (3): empty
// is denied, not inherited); command task queues are empty or platform queues.
func TestCommandWorkload(t *testing.T) {
	tok := taskToken(t, wfA)
	hdr := tenantHeader(jsonString("tenantA"))
	child := func(wt string, tq *taskqueuepb.TaskQueue) *commandpb.Command {
		return &commandpb.Command{
			CommandType: enumspb.COMMAND_TYPE_START_CHILD_WORKFLOW_EXECUTION,
			Attributes: &commandpb.Command_StartChildWorkflowExecutionCommandAttributes{StartChildWorkflowExecutionCommandAttributes: &commandpb.StartChildWorkflowExecutionCommandAttributes{
				Namespace: Namespace, WorkflowId: "t:tenantA:child", WorkflowType: &commonpb.WorkflowType{Name: wt}, TaskQueue: tq, Header: hdr}},
		}
	}
	can := func(wt string, tq *taskqueuepb.TaskQueue) *commandpb.Command {
		return &commandpb.Command{
			CommandType: enumspb.COMMAND_TYPE_CONTINUE_AS_NEW_WORKFLOW_EXECUTION,
			Attributes: &commandpb.Command_ContinueAsNewWorkflowExecutionCommandAttributes{ContinueAsNewWorkflowExecutionCommandAttributes: &commandpb.ContinueAsNewWorkflowExecutionCommandAttributes{
				WorkflowType: &commonpb.WorkflowType{Name: wt}, TaskQueue: tq, Header: hdr}},
		}
	}
	activity := func(tq *taskqueuepb.TaskQueue) *commandpb.Command {
		return &commandpb.Command{
			CommandType: enumspb.COMMAND_TYPE_SCHEDULE_ACTIVITY_TASK,
			Attributes: &commandpb.Command_ScheduleActivityTaskCommandAttributes{ScheduleActivityTaskCommandAttributes: &commandpb.ScheduleActivityTaskCommandAttributes{
				ActivityId: "1", TaskQueue: tq, Header: hdr}},
		}
	}
	platform := &taskqueuepb.TaskQueue{Name: platformQueue()}
	other := &taskqueuepb.TaskQueue{Name: "temporal-sys-per-ns-tq"}

	for name, cmd := range map[string]*commandpb.Command{
		"child, inherited queue":         child(platformType(), nil),
		"child, platform queue":          child(platformType(), platform),
		"continue-as-new, platform":      can(platformType(), platform),
		"activity, inherited queue":      activity(nil),
		"activity, empty-named queue":    activity(&taskqueuepb.TaskQueue{}),
		"activity, platform queue":       activity(platform),
		"continue-as-new, inherit queue": can(platformType(), nil),
	} {
		t.Run("allow "+name, func(t *testing.T) { requireCommandAllow(t, respond(tok, cmd)) })
	}

	for _, c := range []denyCase{
		{name: "child, empty type", req: respond(tok, child("", nil)), reason: ReasonWorkflowType,
			command: "StartChildWorkflowExecution", target: "t:tenantA:child"},
		{name: "child, off-list queue", req: respond(tok, child(platformType(), other)), reason: ReasonTaskQueue,
			command: "StartChildWorkflowExecution", target: "t:tenantA:child"},
		{name: "continue-as-new, empty type", req: respond(tok, can("", nil)), reason: ReasonWorkflowType,
			command: "ContinueAsNewWorkflowExecution"},
		{name: "continue-as-new, off-list queue", req: respond(tok, can(platformType(), other)), reason: ReasonTaskQueue,
			command: "ContinueAsNewWorkflowExecution"},
		{name: "activity, off-list queue", req: respond(tok, activity(other)), reason: ReasonTaskQueue,
			command: "ScheduleActivityTask"},
	} {
		c.workflowID, c.commandIdx = wfA, "0"
		t.Run("deny "+c.name, func(t *testing.T) { requireCommandDeny(t, c) })
	}
}

// D9c: StickyAttributes follow the D9b STICKY rule and are tagged
// command=StickyAttributes, so the command-deny alert covers them.
func TestStickyAttributes(t *testing.T) {
	tok := taskToken(t, wfA)
	withSticky := func(tq *taskqueuepb.TaskQueue) *workflowservice.RespondWorkflowTaskCompletedRequest {
		req := respond(tok)
		req.StickyAttributes = &taskqueuepb.StickyExecutionAttributes{WorkerTaskQueue: tq}
		return req
	}
	requireCommandAllow(t, withSticky(tqOf(sdkSticky, enumspb.TASK_QUEUE_KIND_STICKY, platformQueue())))

	for name, tq := range map[string]*taskqueuepb.TaskQueue{
		"empty NormalName":      tqOf(sdkSticky, enumspb.TASK_QUEUE_KIND_STICKY, ""),
		"platform queue name":   tqOf(platformQueue(), enumspb.TASK_QUEUE_KIND_STICKY, platformQueue()),
		"NORMAL kind, sys name": tqOf("temporal-sys-per-ns-tq", enumspb.TASK_QUEUE_KIND_NORMAL, platformQueue()),
		"no worker task queue":  nil,
	} {
		t.Run(name, func(t *testing.T) {
			requireCommandDeny(t, denyCase{req: withSticky(tq), reason: ReasonStickyQueue,
				command: commandStickyAttributes, workflowID: wfA, commandIdx: "-1"})
		})
	}
}

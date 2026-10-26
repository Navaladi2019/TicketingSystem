package authz

// The Authorizer changes the CI-C drift golden forced (T27k, Technical Lead
// ruling of 2026-10-25 on ADR-001-A2's Annex A.2 statement): D9h-links,
// D9h-failures (update protocol messages included), D9g on poll heartbeats
// and D9c's Kind pin.

import (
	"testing"

	commandpb "go.temporal.io/api/command/v1"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	failurepb "go.temporal.io/api/failure/v1"
	protocolpb "go.temporal.io/api/protocol/v1"
	querypb "go.temporal.io/api/query/v1"
	taskqueuepb "go.temporal.io/api/taskqueue/v1"
	updatepb "go.temporal.io/api/update/v1"
	workerpb "go.temporal.io/api/worker/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/server/common/authorization"
	"google.golang.org/protobuf/proto"
)

func updateResponse(f *failurepb.Failure) *updatepb.Response {
	return &updatepb.Response{Meta: &updatepb.Meta{UpdateId: "update-1"},
		Outcome: &updatepb.Outcome{Value: &updatepb.Outcome_Failure{Failure: f}}}
}

func updateRejection(f *failurepb.Failure) *updatepb.Rejection {
	return &updatepb.Rejection{RejectedRequestMessageId: "req", Failure: f}
}

// D9h-failures on update protocol messages: a Response's or Rejection's child
// failure names a tenant's workflow id; any body that is not one of the three
// update messages, or does not decode, is denied. Each deny is tagged
// command=Message with the message's index.
func TestUpdateMessageChildFailures(t *testing.T) {
	tok := taskToken(t, wfA)
	acceptance := updateMessage(t, &updatepb.Acceptance{AcceptedRequestMessageId: "req"})
	withMessages := func(msgs ...*protocolpb.Message) *workflowservice.RespondWorkflowTaskCompletedRequest {
		req := respond(tok)
		req.Messages = msgs
		return req
	}
	requireCommandAllow(t, withMessages(acceptance))
	requireCommandAllow(t, withMessages(acceptance, updateMessage(t, updateResponse(childFailure("t:tenantA:child", 2)))))
	requireCommandAllow(t, withMessages(updateMessage(t, updateRejection(childFailure("t:tenantA:child", 0)))))
	requireCommandAllow(t, withMessages(updateMessage(t, &updatepb.Response{Meta: &updatepb.Meta{UpdateId: "update-1"},
		Outcome: &updatepb.Outcome{Value: &updatepb.Outcome_Success{Success: &commonpb.Payloads{}}}})))

	for name, build := range map[string]func(*failurepb.Failure) proto.Message{
		"Response":  func(f *failurepb.Failure) proto.Message { return updateResponse(f) },
		"Rejection": func(f *failurepb.Failure) proto.Message { return updateRejection(f) },
	} {
		for forgedName, forged := range forgedIDs {
			t.Run(name+" "+forgedName, func(t *testing.T) {
				requireCommandDeny(t, denyCase{req: withMessages(acceptance, updateMessage(t, build(childFailure(forged, 2)))),
					reason: ReasonNoTenantPrefix, command: commandMessage, workflowID: wfA, target: forged, commandIdx: "1"})
			})
		}
	}

	unknown := updateMessage(t, &updatepb.Request{Meta: &updatepb.Meta{UpdateId: "update-1"}})
	undecodable := updateMessage(t, &updatepb.Response{})
	undecodable.Body.Value = []byte{0xff, 0xff, 0xff}
	for name, msg := range map[string]*protocolpb.Message{
		"unknown body type": unknown,
		"undecodable body":  undecodable,
		"no body":           {Id: "m", ProtocolInstanceId: "update-1"},
	} {
		t.Run(name, func(t *testing.T) {
			requireCommandDeny(t, denyCase{req: withMessages(msg), reason: ReasonProtocolMessage,
				command: commandMessage, workflowID: wfA, commandIdx: "0"})
		})
	}
}

// RespondWorkflowTaskFailed carries update messages too (a worker rejects an
// update by failing its workflow task); the same check applies.
func TestWorkflowTaskFailedUpdateMessages(t *testing.T) {
	api := wf + "RespondWorkflowTaskFailed"
	failed := func(msgs ...*protocolpb.Message) *workflowservice.RespondWorkflowTaskFailedRequest {
		return &workflowservice.RespondWorkflowTaskFailedRequest{Namespace: Namespace, TaskToken: taskToken(t, wfA), Messages: msgs}
	}
	requireWorkloadAllow(t, Worker, api, failed(updateMessage(t, updateRejection(childFailure("t:tenantA:child", 1)))))

	for name, c := range map[string]struct {
		msg            *protocolpb.Message
		reason, target string
	}{
		"forged rejection": {updateMessage(t, updateRejection(childFailure(schedulerID, 1))), ReasonNoTenantPrefix, schedulerID},
		"unknown body":     {updateMessage(t, &updatepb.Request{}), ReasonProtocolMessage, ""},
	} {
		t.Run(name, func(t *testing.T) {
			decision, recs, logs := authorizeAs(t, Worker, api, failed(c.msg))
			if decision != authorization.DecisionDeny || len(recs) != 1 || len(logs) != 1 {
				t.Fatalf("decision=%v metrics=%d logs=%v, want one deny", decision, len(recs), logs)
			}
			want := map[string]string{"reason": c.reason, "command": commandMessage}
			for k, v := range want {
				if recs[0].Tags[k] != v {
					t.Errorf("metric tag %s = %q, want %q", k, recs[0].Tags[k], v)
				}
			}
			for k, v := range map[string]string{"alert": "true", "command_index": "0", "workflow_id": wfA, "target_workflow_id": c.target} {
				if logs[0][k] != v {
					t.Errorf("log field %s = %q, want %q", k, logs[0][k], v)
				}
			}
		})
	}
}

func workflowLink(workflowID string) *commonpb.Link {
	return &commonpb.Link{Variant: &commonpb.Link_WorkflowEvent_{WorkflowEvent: &commonpb.Link_WorkflowEvent{
		Namespace: Namespace, WorkflowId: workflowID, RunId: "run"}}}
}

// childFailure is a failure whose cause chain is depth deep, with the child
// failure info naming childID at the bottom and a system workflow type.
func childFailure(childID string, depth int) *failurepb.Failure {
	f := &failurepb.Failure{Message: "child failed", FailureInfo: &failurepb.Failure_ChildWorkflowExecutionFailureInfo{
		ChildWorkflowExecutionFailureInfo: &failurepb.ChildWorkflowExecutionFailureInfo{
			Namespace:         Namespace,
			WorkflowExecution: &commonpb.WorkflowExecution{WorkflowId: childID, RunId: "run"},
			// D9h classes the type as failure data, so a system type alone passes.
			WorkflowType: &commonpb.WorkflowType{Name: "temporal-sys-batch-workflow"},
		}}}
	for range depth {
		f = &failurepb.Failure{Message: "wrapped", Cause: f}
	}
	return f
}

// forgedIDs are the ids a forged link or child failure might name that fail the
// prefix rule. Another tenant's id passes it on relay and core paths, as D9a
// and D9f do; on worker paths TestChildFailureOtherTenant denies it.
var forgedIDs = map[string]string{"system": schedulerID, "unprefixed": "ticket:42", "empty": "", "empty tenant": "t::x"}

// requireTarget asserts the deny log names the offending id.
func requireTarget(t *testing.T, id Identity, api string, req any, target string) {
	t.Helper()
	_, _, logs := authorizeAs(t, id, api, req)
	if len(logs) != 1 || logs[0]["target_workflow_id"] != target {
		t.Errorf("deny log target_workflow_id = %v, want %q", logs, target)
	}
}

// D9h-links: every link on a relay start or signal, or a core cancel, names a
// tenant's workflow id; a batch job link is denied.
func TestLinkWorkflowIDs(t *testing.T) {
	start := func(links ...*commonpb.Link) *workflowservice.StartWorkflowExecutionRequest {
		req := gateRequest(t, wf+"StartWorkflowExecution").(*workflowservice.StartWorkflowExecutionRequest)
		req.Links = links
		return req
	}
	callback := func(links ...*commonpb.Link) *workflowservice.StartWorkflowExecutionRequest {
		req := start()
		req.CompletionCallbacks = []*commonpb.Callback{{Links: links}}
		return req
	}
	signalWithStart := func(links ...*commonpb.Link) any {
		req := gateRequest(t, wf+"SignalWithStartWorkflowExecution").(*workflowservice.SignalWithStartWorkflowExecutionRequest)
		req.Links = links
		return req
	}
	signal := func(links ...*commonpb.Link) any {
		req := gateRequest(t, wf+"SignalWorkflowExecution").(*workflowservice.SignalWorkflowExecutionRequest)
		req.Links = links
		return req
	}
	cancel := func(links ...*commonpb.Link) any {
		req := gateRequest(t, wf+"RequestCancelWorkflowExecution").(*workflowservice.RequestCancelWorkflowExecutionRequest)
		req.Links = links
		return req
	}
	cases := []struct {
		id    Identity
		api   string
		build func(...*commonpb.Link) any
	}{
		{Relay, "StartWorkflowExecution", func(l ...*commonpb.Link) any { return start(l...) }},
		{Relay, "StartWorkflowExecution", func(l ...*commonpb.Link) any { return callback(l...) }},
		{Relay, "SignalWithStartWorkflowExecution", signalWithStart},
		{Relay, "SignalWorkflowExecution", signal},
		{Core, "RequestCancelWorkflowExecution", cancel},
	}
	batch := &commonpb.Link{Variant: &commonpb.Link_BatchJob_{BatchJob: &commonpb.Link_BatchJob{JobId: "temporal-sys-batch:x"}}}
	for _, c := range cases {
		api := wf + c.api
		requireWorkloadAllow(t, c.id, api, c.build())
		requireWorkloadAllow(t, c.id, api, c.build(workflowLink(wfA), workflowLink(wfB)))
		for name, forged := range forgedIDs {
			t.Run(c.api+" "+name, func(t *testing.T) {
				req := c.build(workflowLink(wfA), workflowLink(forged))
				requireWorkloadDeny(t, c.id, api, req, ReasonNoTenantPrefix)
				requireTarget(t, c.id, api, req, forged)
			})
		}
		t.Run(c.api+" batch job link", func(t *testing.T) {
			requireWorkloadDeny(t, c.id, api, c.build(batch), ReasonNoTenantPrefix)
		})
		t.Run(c.api+" empty link", func(t *testing.T) {
			requireWorkloadDeny(t, c.id, api, c.build(&commonpb.Link{}), ReasonNoTenantPrefix)
		})
	}
}

// D9h-failures on request-level failures: a child failure anywhere along the
// cause chain names a tenant's workflow id. A real one passes.
func TestChildFailureWorkflowIDs(t *testing.T) {
	tok := taskToken(t, wfA)
	requests := map[string]func(*failurepb.Failure) (Identity, any){
		"RespondActivityTaskFailed": func(f *failurepb.Failure) (Identity, any) {
			return Worker, &workflowservice.RespondActivityTaskFailedRequest{Namespace: Namespace, TaskToken: tok, Failure: f}
		},
		"RespondWorkflowTaskFailed": func(f *failurepb.Failure) (Identity, any) {
			return Worker, &workflowservice.RespondWorkflowTaskFailedRequest{Namespace: Namespace, TaskToken: tok, Failure: f}
		},
		"RespondQueryTaskCompleted": func(f *failurepb.Failure) (Identity, any) {
			return Worker, &workflowservice.RespondQueryTaskCompletedRequest{Namespace: Namespace,
				TaskToken: queryToken(t, platformQueue()), Failure: f}
		},
		"StartWorkflowExecution": func(f *failurepb.Failure) (Identity, any) {
			req := gateRequest(t, wf+"StartWorkflowExecution").(*workflowservice.StartWorkflowExecutionRequest)
			req.ContinuedFailure = f
			return Relay, req
		},
	}
	for method, build := range requests {
		api := wf + method
		for _, depth := range []int{0, 3} {
			id, req := build(childFailure("t:tenantA:child", depth))
			requireWorkloadAllow(t, id, api, req)
		}
		id, req := build(&failurepb.Failure{Message: "plain"})
		requireWorkloadAllow(t, id, api, req)
		for name, forged := range forgedIDs {
			t.Run(method+" "+name, func(t *testing.T) {
				id, req := build(childFailure(forged, 3))
				requireWorkloadDeny(t, id, api, req, ReasonNoTenantPrefix)
				requireTarget(t, id, api, req, forged)
			})
		}
	}
}

// D9h-failures in a workflow task response: the failure of a FailWorkflow,
// RecordMarker or continue-as-new command, and of a query result.
func TestCommandChildFailureWorkflowIDs(t *testing.T) {
	tok := taskToken(t, wfA)
	hdr := tenantHeader(jsonString("tenantA"))
	commands := map[string]func(*failurepb.Failure) *commandpb.Command{
		"FailWorkflowExecution": func(f *failurepb.Failure) *commandpb.Command {
			return &commandpb.Command{CommandType: enumspb.COMMAND_TYPE_FAIL_WORKFLOW_EXECUTION,
				Attributes: &commandpb.Command_FailWorkflowExecutionCommandAttributes{
					FailWorkflowExecutionCommandAttributes: &commandpb.FailWorkflowExecutionCommandAttributes{Failure: f}}}
		},
		"RecordMarker": func(f *failurepb.Failure) *commandpb.Command {
			return &commandpb.Command{CommandType: enumspb.COMMAND_TYPE_RECORD_MARKER,
				Attributes: &commandpb.Command_RecordMarkerCommandAttributes{
					RecordMarkerCommandAttributes: &commandpb.RecordMarkerCommandAttributes{MarkerName: "m", Failure: f}}}
		},
		"ContinueAsNewWorkflowExecution": func(f *failurepb.Failure) *commandpb.Command {
			return &commandpb.Command{CommandType: enumspb.COMMAND_TYPE_CONTINUE_AS_NEW_WORKFLOW_EXECUTION,
				Attributes: &commandpb.Command_ContinueAsNewWorkflowExecutionCommandAttributes{
					ContinueAsNewWorkflowExecutionCommandAttributes: &commandpb.ContinueAsNewWorkflowExecutionCommandAttributes{
						WorkflowType: &commonpb.WorkflowType{Name: platformType()}, Header: hdr, Failure: f}}}
		},
	}
	for name, cmd := range commands {
		requireCommandAllow(t, respond(tok, cmd(childFailure("t:tenantA:child", 2))))
		for forgedName, forged := range forgedIDs {
			t.Run(name+" "+forgedName, func(t *testing.T) {
				requireCommandDeny(t, denyCase{req: respond(tok, cmd(childFailure(forged, 2))), reason: ReasonNoTenantPrefix,
					command: name, workflowID: wfA, target: forged, commandIdx: "0"})
			})
		}
	}

	query := func(f *failurepb.Failure) *workflowservice.RespondWorkflowTaskCompletedRequest {
		req := respond(tok)
		req.QueryResults = map[string]*querypb.WorkflowQueryResult{"q1": {ResultType: enumspb.QUERY_RESULT_TYPE_FAILED, Failure: f}}
		return req
	}
	requireCommandAllow(t, query(childFailure("t:tenantA:child", 1)))
	requireCommandDeny(t, denyCase{req: query(childFailure(schedulerID, 1)), reason: ReasonNoTenantPrefix,
		command: commandNone, workflowID: wfA, target: schedulerID, commandIdx: "-1"})
}

// D9g on polls: a poll's heartbeat must name a platform queue, as on
// RecordWorkerHeartbeat, and its deny alerts.
func TestPollHeartbeatTaskQueue(t *testing.T) {
	for name, req := range sdkPolls(nil) {
		set := func(hb *workerpb.WorkerHeartbeat) {
			switch r := req.(type) {
			case *workflowservice.PollWorkflowTaskQueueRequest:
				r.WorkerHeartbeat = hb
			case *workflowservice.PollActivityTaskQueueRequest:
				r.WorkerHeartbeat = hb
			}
		}
		set(&workerpb.WorkerHeartbeat{TaskQueue: platformQueue()})
		t.Run(name+" platform heartbeat", func(t *testing.T) { requireWorkloadAllow(t, Worker, pollAPI(req), req) })
		for hbName, tq := range map[string]string{"system queue": perNSQueue, "empty queue": "", "sticky queue": sdkSticky} {
			set(&workerpb.WorkerHeartbeat{TaskQueue: tq})
			t.Run(name+" "+hbName, func(t *testing.T) { requireWorkloadDeny(t, Worker, pollAPI(req), req, ReasonTaskQueue) })
		}
	}
}

// D9c pins Kind: a command task queue that is STICKY (or unknown) is denied,
// even with a platform name, so the server never routes by its NormalName.
func TestCommandTaskQueueKind(t *testing.T) {
	tok := taskToken(t, wfA)
	hdr := tenantHeader(jsonString("tenantA"))
	activity := func(tq *taskqueuepb.TaskQueue) *commandpb.Command {
		return &commandpb.Command{CommandType: enumspb.COMMAND_TYPE_SCHEDULE_ACTIVITY_TASK,
			Attributes: &commandpb.Command_ScheduleActivityTaskCommandAttributes{ScheduleActivityTaskCommandAttributes: &commandpb.ScheduleActivityTaskCommandAttributes{
				ActivityId: "1", TaskQueue: tq, Header: hdr}}}
	}
	child := func(tq *taskqueuepb.TaskQueue) *commandpb.Command {
		return &commandpb.Command{CommandType: enumspb.COMMAND_TYPE_START_CHILD_WORKFLOW_EXECUTION,
			Attributes: &commandpb.Command_StartChildWorkflowExecutionCommandAttributes{StartChildWorkflowExecutionCommandAttributes: &commandpb.StartChildWorkflowExecutionCommandAttributes{
				Namespace: Namespace, WorkflowId: "t:tenantA:child", WorkflowType: &commonpb.WorkflowType{Name: platformType()}, TaskQueue: tq, Header: hdr}}}
	}
	can := func(tq *taskqueuepb.TaskQueue) *commandpb.Command {
		return &commandpb.Command{CommandType: enumspb.COMMAND_TYPE_CONTINUE_AS_NEW_WORKFLOW_EXECUTION,
			Attributes: &commandpb.Command_ContinueAsNewWorkflowExecutionCommandAttributes{ContinueAsNewWorkflowExecutionCommandAttributes: &commandpb.ContinueAsNewWorkflowExecutionCommandAttributes{
				WorkflowType: &commonpb.WorkflowType{Name: platformType()}, TaskQueue: tq, Header: hdr}}}
	}
	for name, c := range map[string]struct {
		build  func(*taskqueuepb.TaskQueue) *commandpb.Command
		target string
	}{"ScheduleActivityTask": {activity, ""}, "StartChildWorkflowExecution": {child, "t:tenantA:child"}, "ContinueAsNewWorkflowExecution": {can, ""}} {
		requireCommandAllow(t, respond(tok, c.build(tqOf(platformQueue(), enumspb.TASK_QUEUE_KIND_NORMAL, ""))))
		for kindName, kind := range map[string]enumspb.TaskQueueKind{"STICKY": enumspb.TASK_QUEUE_KIND_STICKY, "unknown": enumspb.TaskQueueKind(99)} {
			t.Run(name+" "+kindName, func(t *testing.T) {
				requireCommandDeny(t, denyCase{req: respond(tok, c.build(tqOf(platformQueue(), kind, perNSQueue))),
					reason: ReasonTaskQueueKind, command: name, workflowID: wfA, target: c.target, commandIdx: "0"})
			})
		}
	}
}

// D9h-failures on worker task-token paths binds the task's own tenant, as E1's
// checkTarget does: a child failure naming `t:tenantB:` in a tenant A task is
// denied as cross-tenant. The relay's start, whose certificate has no tenant,
// stays prefix-only, as D9a does.
func TestChildFailureOtherTenant(t *testing.T) {
	tok := taskToken(t, wfA)
	other := childFailure("t:tenantB:child", 2)

	for name, req := range map[string]any{
		"RespondActivityTaskFailed": &workflowservice.RespondActivityTaskFailedRequest{Namespace: Namespace, TaskToken: tok, Failure: other},
		"RespondWorkflowTaskFailed": &workflowservice.RespondWorkflowTaskFailedRequest{Namespace: Namespace, TaskToken: tok, Failure: other},
	} {
		t.Run(name, func(t *testing.T) {
			requireWorkloadDeny(t, Worker, wf+name, req, ReasonCrossTenant)
			requireTarget(t, Worker, wf+name, req, "t:tenantB:child")
		})
	}

	failWorkflow := &commandpb.Command{CommandType: enumspb.COMMAND_TYPE_FAIL_WORKFLOW_EXECUTION,
		Attributes: &commandpb.Command_FailWorkflowExecutionCommandAttributes{
			FailWorkflowExecutionCommandAttributes: &commandpb.FailWorkflowExecutionCommandAttributes{Failure: other}}}
	requireCommandDeny(t, denyCase{req: respond(tok, failWorkflow), reason: ReasonCrossTenant,
		command: "FailWorkflowExecution", workflowID: wfA, target: "t:tenantB:child", commandIdx: "0"})

	withMessage := respond(tok)
	withMessage.Messages = []*protocolpb.Message{updateMessage(t, updateResponse(other))}
	requireCommandDeny(t, denyCase{req: withMessage, reason: ReasonCrossTenant,
		command: commandMessage, workflowID: wfA, target: "t:tenantB:child", commandIdx: "0"})

	withQuery := respond(tok)
	withQuery.QueryResults = map[string]*querypb.WorkflowQueryResult{"q1": {ResultType: enumspb.QUERY_RESULT_TYPE_FAILED, Failure: other}}
	requireCommandDeny(t, denyCase{req: withQuery, reason: ReasonCrossTenant,
		command: commandNone, workflowID: wfA, target: "t:tenantB:child", commandIdx: "-1"})

	start := gateRequest(t, wf+"StartWorkflowExecution").(*workflowservice.StartWorkflowExecutionRequest)
	start.ContinuedFailure = other
	requireWorkloadAllow(t, Relay, wf+"StartWorkflowExecution", start)
}

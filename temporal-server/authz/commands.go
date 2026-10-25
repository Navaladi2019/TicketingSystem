package authz

// Command inspection for RespondWorkflowTaskCompleted from nava-worker: ADR-001
// Annex A.2 TA-B3 with its continue-as-new clause replaced by erratum E1 (plan
// task T27b). The API gate (policy.go) has already passed when this runs, so the
// request targets namespace `nava`; this checks that no command in it reaches
// another tenant, and (ADR-001-A1 D9c) that no command or StickyAttributes hands
// work to a workflow type or task queue off the workload allow-list.

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	commandpb "go.temporal.io/api/command/v1"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	failurepb "go.temporal.io/api/failure/v1"
	protocolpb "go.temporal.io/api/protocol/v1"
	taskqueuepb "go.temporal.io/api/taskqueue/v1"
	updatepb "go.temporal.io/api/update/v1"
	"go.temporal.io/api/workflowservice/v1"
	tokenspb "go.temporal.io/server/api/token/v1"
)

const respondWorkflowTaskCompleted = wf + "RespondWorkflowTaskCompleted"

// TenantHeader is the workflow header the worker's ContextPropagator sets to the
// tenant id, as a default-converter (json/plain) string payload.
const TenantHeader = "nava-tenant"

// TenantSearchAttribute is the visibility-only tenant search attribute.
const TenantSearchAttribute = "NavaTenantId"

// Command deny reasons (erratum E1), used as the `reason` log field and metric tag.
const (
	ReasonTaskToken         = "task_token_invalid"
	ReasonNoTenantPrefix    = "workflow_id_no_tenant"
	ReasonCommandNotAllowed = "command_not_allowed"
	ReasonCrossTenant       = "command_cross_tenant"
	ReasonCommandNamespace  = "command_namespace"
	ReasonTenantHeader      = "continue_as_new_header"
	ReasonActivityHeader    = "activity_header"
	ReasonChildHeader       = "child_header"
	ReasonCronSchedule      = "continue_as_new_cron"
	ReasonTenantSearchAttr  = "search_attribute_tenant"
	ReasonMalformedRequest  = "request_malformed"
	ReasonProtocolMessage   = "protocol_message_not_allowed"
)

// Values of the `command` metric tag besides the command names below.
// commandStickyAttributes tags a denied response-level StickyAttributes (D9c);
// commandMessage tags a denied update protocol message (D9h-failures), whose
// command_index is its position in Messages.
const (
	commandNone             = "none"
	commandUnknown          = "unknown"
	commandStickyAttributes = "StickyAttributes"
	commandMessage          = "Message"
)

// commandNames is every command type of go.temporal.io/api v1.53.0 (unchanged since v1.50.1), by the name
// used for the `command` metric tag and log field. A type added by a later API
// version is tagged `unknown` (and denied, since it is not in allowedCommands).
var commandNames = map[enumspb.CommandType]string{
	enumspb.COMMAND_TYPE_UNSPECIFIED:                                "Unspecified",
	enumspb.COMMAND_TYPE_SCHEDULE_ACTIVITY_TASK:                     "ScheduleActivityTask",
	enumspb.COMMAND_TYPE_REQUEST_CANCEL_ACTIVITY_TASK:               "RequestCancelActivityTask",
	enumspb.COMMAND_TYPE_START_TIMER:                                "StartTimer",
	enumspb.COMMAND_TYPE_COMPLETE_WORKFLOW_EXECUTION:                "CompleteWorkflowExecution",
	enumspb.COMMAND_TYPE_FAIL_WORKFLOW_EXECUTION:                    "FailWorkflowExecution",
	enumspb.COMMAND_TYPE_CANCEL_TIMER:                               "CancelTimer",
	enumspb.COMMAND_TYPE_CANCEL_WORKFLOW_EXECUTION:                  "CancelWorkflowExecution",
	enumspb.COMMAND_TYPE_REQUEST_CANCEL_EXTERNAL_WORKFLOW_EXECUTION: "RequestCancelExternalWorkflowExecution",
	enumspb.COMMAND_TYPE_RECORD_MARKER:                              "RecordMarker",
	enumspb.COMMAND_TYPE_CONTINUE_AS_NEW_WORKFLOW_EXECUTION:         "ContinueAsNewWorkflowExecution",
	enumspb.COMMAND_TYPE_START_CHILD_WORKFLOW_EXECUTION:             "StartChildWorkflowExecution",
	enumspb.COMMAND_TYPE_SIGNAL_EXTERNAL_WORKFLOW_EXECUTION:         "SignalExternalWorkflowExecution",
	enumspb.COMMAND_TYPE_UPSERT_WORKFLOW_SEARCH_ATTRIBUTES:          "UpsertWorkflowSearchAttributes",
	enumspb.COMMAND_TYPE_PROTOCOL_MESSAGE:                           "ProtocolMessage",
	enumspb.COMMAND_TYPE_MODIFY_WORKFLOW_PROPERTIES:                 "ModifyWorkflowProperties",
	enumspb.COMMAND_TYPE_SCHEDULE_NEXUS_OPERATION:                   "ScheduleNexusOperation",
	enumspb.COMMAND_TYPE_REQUEST_CANCEL_NEXUS_OPERATION:             "RequestCancelNexusOperation",
}

// allowedCommands is the written allow-list of E1. Every other type is denied:
// Unspecified, the Nexus commands, and any type a later API version adds.
var allowedCommands = map[enumspb.CommandType]bool{
	enumspb.COMMAND_TYPE_SCHEDULE_ACTIVITY_TASK:                     true,
	enumspb.COMMAND_TYPE_REQUEST_CANCEL_ACTIVITY_TASK:               true,
	enumspb.COMMAND_TYPE_START_TIMER:                                true,
	enumspb.COMMAND_TYPE_CANCEL_TIMER:                               true,
	enumspb.COMMAND_TYPE_RECORD_MARKER:                              true,
	enumspb.COMMAND_TYPE_COMPLETE_WORKFLOW_EXECUTION:                true,
	enumspb.COMMAND_TYPE_FAIL_WORKFLOW_EXECUTION:                    true,
	enumspb.COMMAND_TYPE_CANCEL_WORKFLOW_EXECUTION:                  true,
	enumspb.COMMAND_TYPE_CONTINUE_AS_NEW_WORKFLOW_EXECUTION:         true,
	enumspb.COMMAND_TYPE_START_CHILD_WORKFLOW_EXECUTION:             true,
	enumspb.COMMAND_TYPE_SIGNAL_EXTERNAL_WORKFLOW_EXECUTION:         true,
	enumspb.COMMAND_TYPE_REQUEST_CANCEL_EXTERNAL_WORKFLOW_EXECUTION: true,
	enumspb.COMMAND_TYPE_UPSERT_WORKFLOW_SEARCH_ATTRIBUTES:          true,
	enumspb.COMMAND_TYPE_MODIFY_WORKFLOW_PROPERTIES:                 true,
	enumspb.COMMAND_TYPE_PROTOCOL_MESSAGE:                           true,
}

// commandDenial is why a workflow task response was denied. Only reason and
// command are metric tags; the ids go to the log line (and so the alert) only.
type commandDenial struct {
	reason  string
	command string // a commandNames value, commandUnknown, commandNone, commandStickyAttributes or commandMessage
	index   int    // position in Commands (in Messages for commandMessage), or -1 for the whole response
	// workflowID is the task token's; target is the command's target id.
	workflowID, target string
	detail             string
}

// inspectCommands returns nil if every command in the response stays inside the
// task's tenant, or the first denial. The whole response is denied either way.
func inspectCommands(request any) *commandDenial {
	req, ok := request.(*workflowservice.RespondWorkflowTaskCompletedRequest)
	if !ok || req == nil {
		return &commandDenial{reason: ReasonMalformedRequest, command: commandNone, index: -1,
			detail: fmt.Sprintf("request type %T", request)}
	}
	workflowID, tenant, d := taskTokenTenant(req.GetTaskToken())
	if d != nil {
		return d
	}
	// D9c: StickyAttributes name the queue the workflow's next task goes to.
	if sa := req.GetStickyAttributes(); sa != nil {
		if reason, detail := checkStickyTaskQueue(sa.GetWorkerTaskQueue()); reason != "" {
			return &commandDenial{reason: reason, command: commandStickyAttributes, index: -1,
				workflowID: workflowID, detail: detail}
		}
	}
	// D9h-failures: a query result's failure names only the tenant's children.
	results := req.GetQueryResults()
	for _, k := range slices.Sorted(maps.Keys(results)) {
		if id, ok := failureChildID(results[k].GetFailure()); !ok {
			return &commandDenial{reason: ReasonNoTenantPrefix, command: commandNone, index: -1,
				workflowID: workflowID, target: id, detail: "query result " + childFailureDetail}
		}
	}
	if d := checkMessages(req.GetMessages()); d != nil {
		d.workflowID = workflowID
		return d
	}
	for i, cmd := range req.GetCommands() {
		if reason, target, detail := checkCommand(cmd, tenant); reason != "" {
			return &commandDenial{reason: reason, command: commandName(cmd.GetCommandType()), index: i,
				workflowID: workflowID, target: target, detail: detail}
		}
	}
	return nil
}

// taskTokenTenant decodes a token.v1.Task task token and returns its workflow id
// and that id's tenant, or the denial: the token does not decode, or the id has
// no `t:<tenant>:` prefix. It is E1's first bullet, which D9d (ADR-001-A2)
// applies to every nava-worker task-token API.
func taskTokenTenant(taskToken []byte) (workflowID, tenant string, d *commandDenial) {
	token := &tokenspb.Task{}
	if err := token.Unmarshal(taskToken); err != nil {
		return "", "", &commandDenial{reason: ReasonTaskToken, command: commandNone, index: -1, detail: err.Error()}
	}
	workflowID = token.GetWorkflowId()
	tenant, ok := tenantOf(workflowID)
	if !ok {
		return workflowID, "", &commandDenial{reason: ReasonNoTenantPrefix, command: commandNone, index: -1, workflowID: workflowID}
	}
	return workflowID, tenant, nil
}

// childFailureDetail is the log detail of a D9h-failures deny.
const childFailureDetail = "child failure workflow id"

// checkMessages is D9h-failures for update protocol messages, on
// RespondWorkflowTaskCompleted and RespondWorkflowTaskFailed. A message body
// is a google.protobuf.Any. At server v1.29.7 it must be an update Acceptance,
// Rejection or Response (service/history/workflow/update/update.go
// OnProtocolMessage), and a Rejection's or Response's failure is written to
// history, so its child failures must name the tenant's workflows. Any other
// body, or one that does not decode, is denied: the drift golden expands only
// these three (anyBodies in policy_test.go).
func checkMessages(msgs []*protocolpb.Message) *commandDenial {
	for i, m := range msgs {
		if reason, target, detail := checkMessage(m); reason != "" {
			return &commandDenial{reason: reason, command: commandMessage, index: i, target: target, detail: detail}
		}
	}
	return nil
}

func checkMessage(m *protocolpb.Message) (reason, target, detail string) {
	body := m.GetBody()
	var failure *failurepb.Failure
	var err error
	switch name := body.MessageName(); name {
	case "temporal.api.update.v1.Acceptance":
		err = body.UnmarshalTo(&updatepb.Acceptance{})
	case "temporal.api.update.v1.Rejection":
		r := &updatepb.Rejection{}
		err = body.UnmarshalTo(r)
		failure = r.GetFailure()
	case "temporal.api.update.v1.Response":
		r := &updatepb.Response{}
		err = body.UnmarshalTo(r)
		failure = r.GetOutcome().GetFailure()
	default:
		return ReasonProtocolMessage, "", "message body type " + strconv.Quote(truncate(string(name), maxLoggedName))
	}
	if err != nil {
		return ReasonProtocolMessage, "", "message body: " + err.Error()
	}
	if id, ok := failureChildID(failure); !ok {
		return ReasonNoTenantPrefix, id, "update " + childFailureDetail
	}
	return "", "", ""
}

// checkCommand returns "" if the command is allowed for a task of tenant, or
// the deny reason with the command's target workflow id, if it has one.
// Activities, children and continue-as-new must carry the task's tenant in the
// nava-tenant header; a NavaTenantId they set must be the task's tenant too.
func checkCommand(cmd *commandpb.Command, tenant string) (reason, target, detail string) {
	if !allowedCommands[cmd.GetCommandType()] {
		return ReasonCommandNotAllowed, "", ""
	}
	switch cmd.GetCommandType() {
	case enumspb.COMMAND_TYPE_SCHEDULE_ACTIVITY_TASK:
		a := cmd.GetScheduleActivityTaskCommandAttributes()
		if detail := checkTenantHeader(a.GetHeader(), tenant); detail != "" {
			return ReasonActivityHeader, "", detail
		}
		if reason, detail := checkCommandTaskQueue(a.GetTaskQueue()); reason != "" {
			return reason, "", detail
		}
	case enumspb.COMMAND_TYPE_START_CHILD_WORKFLOW_EXECUTION:
		a := cmd.GetStartChildWorkflowExecutionCommandAttributes()
		if reason, target, detail := checkTarget(a.GetNamespace(), a.GetWorkflowId(), tenant); reason != "" {
			return reason, target, detail
		}
		if detail := checkTenantHeader(a.GetHeader(), tenant); detail != "" {
			return ReasonChildHeader, a.GetWorkflowId(), detail
		}
		if !tenantSearchAttrOK(a.GetSearchAttributes(), tenant) {
			return ReasonTenantSearchAttr, a.GetWorkflowId(), ""
		}
		if reason, detail := checkCommandWorkload(a.GetWorkflowType(), a.GetTaskQueue()); reason != "" {
			return reason, a.GetWorkflowId(), detail
		}
	case enumspb.COMMAND_TYPE_SIGNAL_EXTERNAL_WORKFLOW_EXECUTION:
		a := cmd.GetSignalExternalWorkflowExecutionCommandAttributes()
		return checkTarget(a.GetNamespace(), a.GetExecution().GetWorkflowId(), tenant)
	case enumspb.COMMAND_TYPE_REQUEST_CANCEL_EXTERNAL_WORKFLOW_EXECUTION:
		a := cmd.GetRequestCancelExternalWorkflowExecutionCommandAttributes()
		return checkTarget(a.GetNamespace(), a.GetWorkflowId(), tenant)
	case enumspb.COMMAND_TYPE_CONTINUE_AS_NEW_WORKFLOW_EXECUTION:
		a := cmd.GetContinueAsNewWorkflowExecutionCommandAttributes()
		if a.GetCronSchedule() != "" {
			return ReasonCronSchedule, "", ""
		}
		if id, ok := failureChildID(a.GetFailure()); !ok {
			return ReasonNoTenantPrefix, id, childFailureDetail
		}
		if detail := checkTenantHeader(a.GetHeader(), tenant); detail != "" {
			return ReasonTenantHeader, "", detail
		}
		if !tenantSearchAttrOK(a.GetSearchAttributes(), tenant) {
			return ReasonTenantSearchAttr, "", ""
		}
		if reason, detail := checkCommandWorkload(a.GetWorkflowType(), a.GetTaskQueue()); reason != "" {
			return reason, "", detail
		}
	case enumspb.COMMAND_TYPE_FAIL_WORKFLOW_EXECUTION:
		if id, ok := failureChildID(cmd.GetFailWorkflowExecutionCommandAttributes().GetFailure()); !ok {
			return ReasonNoTenantPrefix, id, childFailureDetail
		}
	case enumspb.COMMAND_TYPE_RECORD_MARKER:
		if id, ok := failureChildID(cmd.GetRecordMarkerCommandAttributes().GetFailure()); !ok {
			return ReasonNoTenantPrefix, id, childFailureDetail
		}
	case enumspb.COMMAND_TYPE_UPSERT_WORKFLOW_SEARCH_ATTRIBUTES:
		a := cmd.GetUpsertWorkflowSearchAttributesCommandAttributes()
		if !tenantSearchAttrOK(a.GetSearchAttributes(), tenant) {
			return ReasonTenantSearchAttr, "", ""
		}
	}
	return "", "", ""
}

// checkCommandWorkload is D9c for a child or continue-as-new: the workflow type
// is a platform type (an empty one is denied, not inherited) and the task queue
// is empty (inherited) or a platform queue.
func checkCommandWorkload(wt *commonpb.WorkflowType, tq *taskqueuepb.TaskQueue) (reason, detail string) {
	if reason, detail := checkWorkflowType(wt.GetName()); reason != "" {
		return reason, detail
	}
	return checkCommandTaskQueue(tq)
}

// checkCommandTaskQueue is D9c for a command's task queue: Kind NORMAL or
// UNSPECIFIED, so the server never reads NormalName (tqid.PartitionFromProto
// rejects one on a non-sticky queue), and a name that is empty (the command
// inherits the workflow's queue) or a platform queue.
func checkCommandTaskQueue(tq *taskqueuepb.TaskQueue) (reason, detail string) {
	switch tq.GetKind() {
	case enumspb.TASK_QUEUE_KIND_NORMAL, enumspb.TASK_QUEUE_KIND_UNSPECIFIED:
	default:
		return ReasonTaskQueueKind, "task queue kind " + tq.GetKind().String()
	}
	if name := tq.GetName(); name != "" && !inSet(platformTaskQueues, name) {
		return ReasonTaskQueue, "task queue " + strconv.Quote(truncate(name, maxLoggedName))
	}
	return "", ""
}

// checkTarget allows a target in namespace `nava` (or "", which Temporal
// resolves to the task's own namespace, already gated to `nava`) whose
// workflow id carries the task's tenant prefix.
func checkTarget(namespace, workflowID, tenant string) (reason, target, detail string) {
	if namespace != "" && namespace != Namespace {
		return ReasonCommandNamespace, workflowID, "target namespace " + truncate(namespace, maxLoggedNamespace)
	}
	if got, ok := tenantOf(workflowID); !ok || got != tenant {
		return ReasonCrossTenant, workflowID, ""
	}
	return "", "", ""
}

// checkTenantHeader returns "" if the header's nava-tenant is the tenant, or
// why not, for the log line's detail.
func checkTenantHeader(h *commonpb.Header, tenant string) string {
	got, ok := payloadString(h.GetFields()[TenantHeader])
	if !ok {
		return "header missing or not a json/plain string"
	}
	if got != tenant {
		return "header tenant differs from the task's"
	}
	return ""
}

// tenantSearchAttrOK reports whether NavaTenantId is unset or is the tenant.
func tenantSearchAttrOK(sa *commonpb.SearchAttributes, tenant string) bool {
	p, set := sa.GetIndexedFields()[TenantSearchAttribute]
	if !set {
		return true
	}
	got, ok := payloadString(p)
	return ok && got == tenant
}

// tenantOf returns the tenant of a `t:<tenant>:<rest>` workflow id.
func tenantOf(workflowID string) (string, bool) {
	rest, ok := strings.CutPrefix(workflowID, "t:")
	if !ok {
		return "", false
	}
	tenant, _, ok := strings.Cut(rest, ":")
	if !ok || tenant == "" {
		return "", false
	}
	return tenant, true
}

// payloadString decodes a default-converter string payload (json/plain).
// Anything else, including a JSON list, does not decode.
func payloadString(p *commonpb.Payload) (string, bool) {
	if p == nil || string(p.GetMetadata()["encoding"]) != "json/plain" {
		return "", false
	}
	var s string
	if err := json.Unmarshal(p.GetData(), &s); err != nil {
		return "", false
	}
	return s, true
}

func commandName(t enumspb.CommandType) string {
	if name, ok := commandNames[t]; ok {
		return name
	}
	return commandUnknown
}

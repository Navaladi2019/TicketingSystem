package authz

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"

	commonpb "go.temporal.io/api/common/v1"
	deploymentpb "go.temporal.io/api/deployment/v1"
	enumspb "go.temporal.io/api/enums/v1"
	failurepb "go.temporal.io/api/failure/v1"
	protocolpb "go.temporal.io/api/protocol/v1"
	taskqueuepb "go.temporal.io/api/taskqueue/v1"
	workerpb "go.temporal.io/api/worker/v1"
	"go.temporal.io/api/workflowservice/v1"
	tokenspb "go.temporal.io/server/api/token/v1"
	"go.temporal.io/server/common/authorization"
	"go.temporal.io/server/common/log"
	"go.temporal.io/server/common/log/tag"
	"go.temporal.io/server/common/metrics"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	// Register the service descriptors used to find namespace-scoped requests
	// (workflowservice is registered by the import above).
	_ "go.temporal.io/api/operatorservice/v1"
	_ "go.temporal.io/server/api/adminservice/v1"
)

// DeniedMetric counts denied calls, tagged identity, api, namespace, reason and
// command (the denied workflow command's type, or "none" for an API deny).
const DeniedMetric = "nava_authz_denied"

// Deny reasons, used as the `reason` log field and metric tag.
const (
	ReasonNoIdentity = "no_identity"
	ReasonAPI        = "api_not_allowed"
	ReasonNamespace  = "namespace_not_allowed"
	ReasonInternal   = "internal_error"
)

// D9 deny reasons (workload allow-list), used as the `reason` log field and
// metric tag. A start or signal without a tenant prefix is ReasonNoTenantPrefix.
const (
	ReasonWorkflowType  = "workflow_type_not_allowed"
	ReasonTaskQueue     = "task_queue_not_allowed"
	ReasonTaskQueueKind = "task_queue_kind_not_allowed"
	ReasonStickyQueue   = "sticky_queue_not_allowed"
	// ReasonWorkerVersioning is a versioned poll (D9b+, ADR-001-A2).
	ReasonWorkerVersioning = "worker_versioning_not_allowed"
)

// maxLoggedNamespace bounds the client-supplied namespace written to the log.
const maxLoggedNamespace = 128

// maxLoggedWorkflowID bounds a client-supplied workflow id written to the log.
const maxLoggedWorkflowID = 256

// maxLoggedName bounds a client-supplied workflow type or task queue name
// written to the log.
const maxLoggedName = 128

var (
	allow = authorization.Result{Decision: authorization.DecisionAllow}
	// No Reason: the caller gets the server's generic "Request unauthorized".
	// The reason goes to the log and the metric only.
	deny = authorization.Result{Decision: authorization.DecisionDeny}
)

// Authorizer is the deny-by-default frontend Authorizer.
type Authorizer struct {
	logger  log.Logger
	metrics metrics.Handler
	allowed map[Identity]map[string]struct{}
	// known is every classified API; anything else is tagged "other" in metrics.
	known map[string]struct{}
	// namespaceScoped is the set of allow-listed APIs whose request has a
	// `namespace` field. Those must target Namespace; the rest must target none.
	namespaceScoped map[string]bool
}

var _ authorization.Authorizer = (*Authorizer)(nil)

// NewAuthorizer builds the Authorizer from the static policy.
func NewAuthorizer(logger log.Logger, metricsHandler metrics.Handler) *Authorizer {
	a := &Authorizer{
		logger:          logger,
		metrics:         metricsHandler,
		allowed:         make(map[Identity]map[string]struct{}, len(allowList)),
		known:           make(map[string]struct{}),
		namespaceScoped: make(map[string]bool),
	}
	for id, apis := range allowList {
		set := make(map[string]struct{}, len(apis))
		for _, api := range apis {
			set[api] = struct{}{}
			a.known[api] = struct{}{}
			a.namespaceScoped[api] = requestHasNamespace(api)
		}
		a.allowed[id] = set
	}
	for _, api := range deniedByDefault {
		a.known[api] = struct{}{}
	}
	return a
}

// Authorize never panics and never returns an error: any failure is a deny.
func (a *Authorizer) Authorize(
	ctx context.Context,
	claims *authorization.Claims,
	target *authorization.CallTarget,
) (result authorization.Result, err error) {
	var api, namespace string
	if target != nil {
		api, namespace = target.APIName, target.Namespace
	}
	id := identityOf(claims)
	defer func() {
		if r := recover(); r != nil {
			result, err = deny, nil
			a.recordDeny(ctx, id, api, namespace, ReasonInternal, fmt.Sprint(r))
		}
	}()

	if reason := a.decide(id, api, namespace); reason != "" {
		a.recordDeny(ctx, id, api, namespace, reason, "")
		return deny, nil
	}
	if id == Worker && api == respondWorkflowTaskCompleted {
		if d := inspectCommands(target.Request); d != nil {
			a.recordCommandDeny(ctx, id, api, namespace, "Temporal workflow command denied", d)
			return deny, nil
		}
	}
	if c, ok := workloadChecks[id][api]; ok {
		if d := c.check(target.Request); d != nil {
			a.recordCommandDeny(ctx, id, api, namespace, "Temporal workload denied", d)
			return deny, nil
		}
	}
	return allow, nil
}

// workloadCheck is one D9 request inspection and the ADR rule it implements.
type workloadCheck struct {
	rule  string // the Annex A.2 sub-table row, for the golden table
	check func(request any) *commandDenial
}

// workloadChecks is D9a, D9b(+), D9d, D9e, D9f, D9g and D9h: the request
// inspection for each allow-listed API that names a workflow type, a task queue
// or a workflow id (ADR-001-A1, ADR-001-A2; D9h and D9g on polls are T27k's).
// E1, D9c and D9h-failures for workflow task responses are in inspectCommands.
// System is absent: it is exempt. So is Operator: it stays outside D9.
var workloadChecks = map[Identity]map[string]workloadCheck{
	Relay: {
		wf + "StartWorkflowExecution":           {"D9a, D9h-links, D9h-failures", checkStartRequest},
		wf + "SignalWithStartWorkflowExecution": {"D9a, D9h-links", checkStartRequest},
		wf + "SignalWorkflowExecution":          {"D9a, D9h-links", checkSignalRequest},
	},
	Core: {
		wf + "RequestCancelWorkflowExecution": {"D9f, D9h-links", checkCoreWorkflowID},
		wf + "UpdateWorkflowExecution":        {"D9f", checkCoreWorkflowID},
		wf + "QueryWorkflow":                  {"D9f", checkCoreWorkflowID},
		wf + "DescribeWorkflowExecution":      {"D9f", checkCoreWorkflowID},
		wf + "GetWorkflowExecutionHistory":    {"D9f", checkCoreWorkflowID},
	},
	Worker: {
		wf + "PollWorkflowTaskQueue":        {"D9b, D9b+, D9g", checkWorkflowTaskPoll},
		wf + "PollActivityTaskQueue":        {"D9b, D9b+, D9g", checkActivityTaskPoll},
		wf + "RespondWorkflowTaskFailed":    {"D9d, D9h-failures", checkTaskToken},
		wf + "RespondActivityTaskCompleted": {"D9d", checkTaskToken},
		wf + "RespondActivityTaskFailed":    {"D9d, D9h-failures", checkTaskToken},
		wf + "RespondActivityTaskCanceled":  {"D9d", checkTaskToken},
		wf + "RecordActivityTaskHeartbeat":  {"D9d", checkTaskToken},
		wf + "RespondQueryTaskCompleted":    {"D9d query clause, D9h-failures", checkQueryTaskToken},
		wf + "GetWorkflowExecutionHistory":  {"D9e", checkWorkerHistory},
		wf + "ShutdownWorker":               {"D9g", checkShutdownWorker},
		wf + "RecordWorkerHeartbeat":        {"D9g", checkRecordWorkerHeartbeat},
	},
}

// checkStartRequest is D9a for starts: the workflow type and the task queue are
// allow-listed, the queue is NORMAL, and the workflow id has a tenant prefix.
func checkStartRequest(request any) *commandDenial {
	var workflowID string
	var wt *commonpb.WorkflowType
	var tq *taskqueuepb.TaskQueue
	var links []*commonpb.Link
	var continued *failurepb.Failure
	switch req := request.(type) {
	case *workflowservice.StartWorkflowExecutionRequest:
		workflowID, wt, tq = req.GetWorkflowId(), req.GetWorkflowType(), req.GetTaskQueue()
		links, continued = req.GetLinks(), req.GetContinuedFailure()
		for _, cb := range req.GetCompletionCallbacks() {
			links = slices.Concat(links, cb.GetLinks())
		}
	case *workflowservice.SignalWithStartWorkflowExecutionRequest:
		workflowID, wt, tq = req.GetWorkflowId(), req.GetWorkflowType(), req.GetTaskQueue()
		links = req.GetLinks()
	default:
		return malformed(request)
	}
	if d := firstDenial(linkDenial(links), failureDenial(continued)); d != nil {
		d.workflowID = workflowID
		return d
	}
	reason, detail := checkWorkflowType(wt.GetName())
	if reason == "" {
		reason, detail = checkNormalTaskQueue(tq)
	}
	if reason == "" {
		if _, ok := tenantOf(workflowID); !ok {
			reason = ReasonNoTenantPrefix
		}
	}
	if reason == "" {
		return nil
	}
	return &commandDenial{reason: reason, command: commandNone, index: -1, workflowID: workflowID, detail: detail}
}

// checkSignalRequest is D9a for signals: the target workflow id has a tenant
// prefix, so no relay signal reaches a `temporal-sys-*` workflow.
func checkSignalRequest(request any) *commandDenial {
	req, ok := request.(*workflowservice.SignalWorkflowExecutionRequest)
	if !ok {
		return malformed(request)
	}
	return requestIDDenial(req.GetWorkflowExecution().GetWorkflowId(), req.GetLinks())
}

// checkCoreWorkflowID is D9f: the workflow id a nava-core call names carries a
// tenant prefix. It proves the id is a tenant's, not which tenant's: the
// nava-core certificate has no tenant (R-CORE binds that in nava-core itself).
func checkCoreWorkflowID(request any) *commandDenial {
	var workflowID string
	var links []*commonpb.Link
	switch req := request.(type) {
	case *workflowservice.RequestCancelWorkflowExecutionRequest:
		workflowID, links = req.GetWorkflowExecution().GetWorkflowId(), req.GetLinks()
	case *workflowservice.UpdateWorkflowExecutionRequest:
		workflowID = req.GetWorkflowExecution().GetWorkflowId()
	case *workflowservice.QueryWorkflowRequest:
		workflowID = req.GetExecution().GetWorkflowId()
	case *workflowservice.DescribeWorkflowExecutionRequest:
		workflowID = req.GetExecution().GetWorkflowId()
	case *workflowservice.GetWorkflowExecutionHistoryRequest:
		workflowID = req.GetExecution().GetWorkflowId()
	default:
		return malformed(request)
	}
	return requestIDDenial(workflowID, links)
}

// checkWorkerHistory is D9e: a nava-worker history read names a tenant's
// workflow id.
func checkWorkerHistory(request any) *commandDenial {
	req, ok := request.(*workflowservice.GetWorkflowExecutionHistoryRequest)
	if !ok {
		return malformed(request)
	}
	return tenantPrefixDenial(req.GetExecution().GetWorkflowId())
}

// checkTaskToken is D9d: the task token decodes as token.v1.Task and names a
// tenant's workflow id, E1's check on RespondWorkflowTaskCompleted. Tokens are
// unsigned, so this keeps a forged token off Temporal's own workflows (F2-3).
// D9h-failures: a failure, and a workflow task failure's update messages, name
// only prefixed children.
func checkTaskToken(request any) *commandDenial {
	var taskToken []byte
	var failure *failurepb.Failure
	var messages []*protocolpb.Message
	switch req := request.(type) {
	case *workflowservice.RespondWorkflowTaskFailedRequest:
		taskToken, failure, messages = req.GetTaskToken(), req.GetFailure(), req.GetMessages()
	case *workflowservice.RespondActivityTaskCompletedRequest:
		taskToken = req.GetTaskToken()
	case *workflowservice.RespondActivityTaskFailedRequest:
		taskToken, failure = req.GetTaskToken(), req.GetFailure()
	case *workflowservice.RespondActivityTaskCanceledRequest:
		taskToken = req.GetTaskToken()
	case *workflowservice.RecordActivityTaskHeartbeatRequest:
		taskToken = req.GetTaskToken()
	default:
		return malformed(request)
	}
	workflowID, _, d := taskTokenTenant(taskToken)
	if d != nil {
		return d
	}
	if d := firstDenial(failureDenial(failure), checkMessages(messages)); d != nil {
		d.workflowID = workflowID
		return d
	}
	return nil
}

// checkQueryTaskToken is the D9d query clause: the token decodes as
// token.v1.QueryTask and its task queue is a platform queue or sticky-shaped.
// D9h-failures: the query's failure names only prefixed children.
func checkQueryTaskToken(request any) *commandDenial {
	req, ok := request.(*workflowservice.RespondQueryTaskCompletedRequest)
	if !ok {
		return malformed(request)
	}
	token := &tokenspb.QueryTask{}
	if err := token.Unmarshal(req.GetTaskToken()); err != nil {
		return &commandDenial{reason: ReasonTaskToken, command: commandNone, index: -1, detail: err.Error()}
	}
	tq := token.GetTaskQueue()
	if !inSet(platformTaskQueues, tq) && !stickyQueueName.MatchString(tq) {
		return pollDenial(ReasonTaskQueue, "query token task queue "+strconv.Quote(truncate(tq, maxLoggedName)))
	}
	return failureDenial(req.GetFailure())
}

// checkShutdownWorker is D9g for ShutdownWorker. The frontend force-unloads the
// root WORKFLOW partition that StickyTaskQueue names (F2-6), so it must be
// empty or an SDK-shaped sticky name that is not a platform queue. A heartbeat,
// if present, names a platform queue.
func checkShutdownWorker(request any) *commandDenial {
	req, ok := request.(*workflowservice.ShutdownWorkerRequest)
	if !ok {
		return malformed(request)
	}
	if name := req.GetStickyTaskQueue(); name != "" && (!stickyQueueName.MatchString(name) || inSet(platformTaskQueues, name)) {
		return pollDenial(ReasonStickyQueue, "sticky task queue "+strconv.Quote(truncate(name, maxLoggedName)))
	}
	if hb := req.GetWorkerHeartbeat(); hb != nil {
		return pollDenial(checkHeartbeatTaskQueue(hb))
	}
	return nil
}

// checkRecordWorkerHeartbeat is D9g for RecordWorkerHeartbeat: every heartbeat
// names a platform queue.
func checkRecordWorkerHeartbeat(request any) *commandDenial {
	req, ok := request.(*workflowservice.RecordWorkerHeartbeatRequest)
	if !ok {
		return malformed(request)
	}
	for _, hb := range req.GetWorkerHeartbeat() {
		if d := pollDenial(checkHeartbeatTaskQueue(hb)); d != nil {
			return d
		}
	}
	return nil
}

// checkHeartbeatTaskQueue allows a heartbeat on a platform queue; an empty
// TaskQueue is denied.
func checkHeartbeatTaskQueue(hb *workerpb.WorkerHeartbeat) (reason, detail string) {
	if inSet(platformTaskQueues, hb.GetTaskQueue()) {
		return "", ""
	}
	return ReasonTaskQueue, "heartbeat task queue " + strconv.Quote(truncate(hb.GetTaskQueue(), maxLoggedName))
}

func tenantPrefixDenial(workflowID string) *commandDenial {
	if _, ok := tenantOf(workflowID); ok {
		return nil
	}
	return &commandDenial{reason: ReasonNoTenantPrefix, command: commandNone, index: -1, workflowID: workflowID}
}

// requestIDDenial is D9a (signals) or D9f, then D9h-links: the request's
// workflow id and every link's workflow id carry a tenant prefix.
func requestIDDenial(workflowID string, links []*commonpb.Link) *commandDenial {
	if d := tenantPrefixDenial(workflowID); d != nil {
		return d
	}
	if d := linkDenial(links); d != nil {
		d.workflowID = workflowID
		return d
	}
	return nil
}

// linkDenial is D9h-links (ADR-001-A2 Annex A.2: every workflow id an
// allow-listed API names has a tenant prefix). Each link must be a workflow
// event link whose workflow id has a tenant prefix. Any other variant (a batch
// job link names a Temporal batch workflow) is denied too: the platform's SDK
// sends links only from a Nexus handler, which nava does not run. The link's id
// goes to the log's target_workflow_id.
func linkDenial(links []*commonpb.Link) *commandDenial {
	for _, l := range links {
		id := l.GetWorkflowEvent().GetWorkflowId()
		if _, ok := tenantOf(id); !ok || l.GetWorkflowEvent() == nil {
			return &commandDenial{reason: ReasonNoTenantPrefix, command: commandNone, index: -1, target: id, detail: "link workflow id"}
		}
	}
	return nil
}

// failureChildID is D9h-failures: it returns the first child workflow id in
// f's whole cause chain that has no tenant prefix. A tenant's real child
// failure passes, because E1 only lets it start children under its own prefix.
// The chain cannot cycle (it is decoded data) and its depth is bounded by the
// protobuf decoder's recursion limit, the runtime form of the golden walk's
// stop at `failure.cause → (Failure)`.
func failureChildID(f *failurepb.Failure) (id string, ok bool) {
	for ; f != nil; f = f.GetCause() {
		if info := f.GetChildWorkflowExecutionFailureInfo(); info != nil {
			id := info.GetWorkflowExecution().GetWorkflowId()
			if _, ok := tenantOf(id); !ok {
				return id, false
			}
		}
	}
	return "", true
}

// failureDenial is D9h-failures for a request-level failure.
func failureDenial(f *failurepb.Failure) *commandDenial {
	if id, ok := failureChildID(f); !ok {
		return &commandDenial{reason: ReasonNoTenantPrefix, command: commandNone, index: -1, target: id, detail: childFailureDetail}
	}
	return nil
}

// firstDenial returns the first non-nil denial.
func firstDenial(ds ...*commandDenial) *commandDenial {
	for _, d := range ds {
		if d != nil {
			return d
		}
	}
	return nil
}

// checkWorkflowTaskPoll is D9b for workflow tasks: a NORMAL poll names a
// platform queue and a STICKY poll follows the sticky rule. D9b+: it is
// unversioned. D9g: its heartbeat names a platform queue.
func checkWorkflowTaskPoll(request any) *commandDenial {
	req, ok := request.(*workflowservice.PollWorkflowTaskQueueRequest)
	if !ok {
		return malformed(request)
	}
	//nolint:staticcheck // SA1019: matching still reads the deprecated capabilities, so D9b+ must too
	if d := pollDenial(checkUnversioned(req.GetDeploymentOptions(), req.GetWorkerVersionCapabilities())); d != nil {
		return d
	}
	if d := pollHeartbeatDenial(req.GetWorkerHeartbeat()); d != nil {
		return d
	}
	if req.GetTaskQueue().GetKind() == enumspb.TASK_QUEUE_KIND_STICKY {
		return pollDenial(checkStickyTaskQueue(req.GetTaskQueue()))
	}
	return pollDenial(checkNormalTaskQueue(req.GetTaskQueue()))
}

// checkActivityTaskPoll is D9b for activity tasks: a NORMAL platform queue.
// Activities have no sticky queue, so a STICKY activity poll is denied. D9b+:
// it is unversioned. D9g: its heartbeat names a platform queue.
func checkActivityTaskPoll(request any) *commandDenial {
	req, ok := request.(*workflowservice.PollActivityTaskQueueRequest)
	if !ok {
		return malformed(request)
	}
	//nolint:staticcheck // SA1019: matching still reads the deprecated capabilities, so D9b+ must too
	if d := pollDenial(checkUnversioned(req.GetDeploymentOptions(), req.GetWorkerVersionCapabilities())); d != nil {
		return d
	}
	if d := pollHeartbeatDenial(req.GetWorkerHeartbeat()); d != nil {
		return d
	}
	return pollDenial(checkNormalTaskQueue(req.GetTaskQueue()))
}

// pollHeartbeatDenial is D9g on a poll: the frontend forwards a poll's
// WorkerHeartbeat to matching's RecordWorkerHeartbeat (server v1.29.7
// frontend/workflow_handler.go PollWorkflowTaskQueue), so a heartbeat, if
// present, names a platform queue exactly as on RecordWorkerHeartbeat. sdk-go
// v1.49.0 sends none on polls.
func pollHeartbeatDenial(hb *workerpb.WorkerHeartbeat) *commandDenial {
	if hb == nil {
		return nil
	}
	return pollDenial(checkHeartbeatTaskQueue(hb))
}

// checkUnversioned is D9b+: a versioned poll makes matching register the worker
// in a deployment, which starts the temporal-sys-worker-deployment* workflows
// in `nava` (F2-1). Absent options count as unversioned.
func checkUnversioned(opts *deploymentpb.WorkerDeploymentOptions, caps *commonpb.WorkerVersionCapabilities) (reason, detail string) {
	switch mode := opts.GetWorkerVersioningMode(); mode {
	case enumspb.WORKER_VERSIONING_MODE_UNSPECIFIED, enumspb.WORKER_VERSIONING_MODE_UNVERSIONED:
	default:
		return ReasonWorkerVersioning, "deployment options versioning mode " + mode.String()
	}
	if caps.GetUseVersioning() {
		return ReasonWorkerVersioning, "worker version capabilities use versioning"
	}
	return "", ""
}

func pollDenial(reason, detail string) *commandDenial {
	if reason == "" {
		return nil
	}
	return &commandDenial{reason: reason, command: commandNone, index: -1, detail: detail}
}

func malformed(request any) *commandDenial {
	return &commandDenial{reason: ReasonMalformedRequest, command: commandNone, index: -1,
		detail: fmt.Sprintf("request type %T", request)}
}

// checkWorkflowType allows only a platform workflow type; "" is denied.
func checkWorkflowType(name string) (reason, detail string) {
	if inSet(platformWorkflowTypes, name) {
		return "", ""
	}
	return ReasonWorkflowType, "workflow type " + strconv.Quote(truncate(name, maxLoggedName))
}

// checkNormalTaskQueue allows a NORMAL (or UNSPECIFIED) platform task queue.
func checkNormalTaskQueue(tq *taskqueuepb.TaskQueue) (reason, detail string) {
	switch tq.GetKind() {
	case enumspb.TASK_QUEUE_KIND_NORMAL, enumspb.TASK_QUEUE_KIND_UNSPECIFIED:
	default:
		return ReasonTaskQueueKind, "task queue kind " + tq.GetKind().String()
	}
	if !inSet(platformTaskQueues, tq.GetName()) {
		return ReasonTaskQueue, "task queue " + strconv.Quote(truncate(tq.GetName(), maxLoggedName))
	}
	return "", ""
}

// checkStickyTaskQueue is the D9b STICKY rule, whatever the queue's Kind field
// says: an SDK-shaped `<host>:<uuid>` name that is not itself a platform queue,
// for a platform NormalName (an empty NormalName is denied).
func checkStickyTaskQueue(tq *taskqueuepb.TaskQueue) (reason, detail string) {
	name := tq.GetName()
	if !stickyQueueName.MatchString(name) || inSet(platformTaskQueues, name) {
		return ReasonStickyQueue, "sticky task queue " + strconv.Quote(truncate(name, maxLoggedName))
	}
	if !inSet(platformTaskQueues, tq.GetNormalName()) {
		return ReasonStickyQueue, "sticky normal name " + strconv.Quote(truncate(tq.GetNormalName(), maxLoggedName))
	}
	return "", ""
}

// decide returns "" to allow, or the deny reason.
func (a *Authorizer) decide(id Identity, api, namespace string) string {
	switch {
	case api == healthCheck:
		return ""
	case id == "":
		return ReasonNoIdentity
	case id == System:
		return ""
	}
	if _, ok := a.allowed[id][api]; !ok {
		return ReasonAPI
	}
	if namespace == Namespace {
		return ""
	}
	if namespace == "" && !a.namespaceScoped[api] {
		return ""
	}
	return ReasonNamespace
}

func identityOf(claims *authorization.Claims) Identity {
	if claims == nil {
		return ""
	}
	id, _ := identityFromCN(claims.Subject)
	return id
}

func (a *Authorizer) recordDeny(ctx context.Context, id Identity, api, namespace, reason, detail string) {
	// Logging and metrics must not turn a deny into a panic either.
	defer func() { _ = recover() }()

	tags := a.denyTags(ctx, id, api, namespace, reason, commandNone)
	if detail != "" {
		tags = append(tags, tag.NewStringTag("detail", detail))
	}
	a.logger.Warn("Temporal API call denied", tags...)
}

// recordCommandDeny logs a denied workflow task response, or a D9 workload
// deny, at error level with alert=true: a denied task retries until someone
// acts, and a denied workload request means a misused certificate, so each one
// must page. The workflow ids, types and queue names go to the log line only;
// they hold the tenant and are unbounded.
//
// TODO(TODO-004): no alert rule is installed yet; install the
// nava_authz_denied alert rules (commands and D9 workload) from docs/temporal-server.md.
func (a *Authorizer) recordCommandDeny(ctx context.Context, id Identity, api, namespace, msg string, d *commandDenial) {
	defer func() { _ = recover() }()

	tags := a.denyTags(ctx, id, api, namespace, d.reason, d.command)
	tags = append(tags,
		tag.NewStringTag("alert", "true"),
		tag.NewStringTag("workflow_id", truncate(d.workflowID, maxLoggedWorkflowID)),
		tag.NewStringTag("command_index", strconv.Itoa(d.index)),
	)
	if d.target != "" {
		tags = append(tags, tag.NewStringTag("target_workflow_id", truncate(d.target, maxLoggedWorkflowID)))
	}
	if d.detail != "" {
		tags = append(tags, tag.NewStringTag("detail", d.detail))
	}
	a.logger.Error(msg, tags...)
}

// denyTags increments DeniedMetric and returns the log tags common to every deny.
func (a *Authorizer) denyTags(ctx context.Context, id Identity, api, namespace, reason, command string) []tag.Tag {
	identityTag := string(id)
	if identityTag == "" {
		identityTag = "none"
	}
	apiTag := api
	if _, ok := a.known[api]; !ok {
		apiTag = "other"
	}
	a.metrics.WithTags(
		metrics.StringTag("identity", identityTag),
		metrics.StringTag("api", apiTag),
		metrics.StringTag("namespace", namespaceTag(namespace)),
		metrics.StringTag("reason", reason),
		metrics.StringTag("command", command),
	).Counter(DeniedMetric).Record(1)

	tags := []tag.Tag{
		tag.NewStringTag("identity", identityTag),
		tag.NewStringTag("api", api),
		tag.NewStringTag("namespace", truncate(namespace, maxLoggedNamespace)),
		tag.NewStringTag("reason", reason),
		tag.NewStringTag("command", command),
	}
	if cert := authorization.PeerCert(authorization.TLSInfoFromContext(ctx)); cert != nil {
		tags = append(tags,
			tag.NewStringTag("cert_subject", cert.Subject.String()),
			tag.NewStringTag("cert_serial", cert.SerialNumber.String()),
			tag.NewStringTag("cert_issuer", cert.Issuer.String()),
		)
	} else {
		tags = append(tags, tag.NewStringTag("cert_subject", "none"))
	}
	return tags
}

// namespaceTag keeps the metric's cardinality bounded: the namespace is
// client-supplied and not yet validated when the Authorizer runs.
func namespaceTag(namespace string) string {
	switch namespace {
	case "":
		return "none"
	case Namespace, string(System):
		return namespace
	default:
		return "other"
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// requestHasNamespace reports whether the API's request message has a
// `namespace` field, from the registered proto descriptors. An API it cannot
// resolve counts as namespace-scoped, so it must target Namespace (fail closed).
func requestHasNamespace(api string) bool {
	parts := strings.Split(strings.TrimPrefix(api, "/"), "/")
	if len(parts) != 2 {
		return true
	}
	d, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(parts[0]))
	if err != nil {
		return true
	}
	svc, ok := d.(protoreflect.ServiceDescriptor)
	if !ok {
		return true
	}
	m := svc.Methods().ByName(protoreflect.Name(parts[1]))
	if m == nil {
		return true
	}
	return m.Input().Fields().ByName("namespace") != nil
}

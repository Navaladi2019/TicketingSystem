package authz

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"testing"

	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/api/operatorservice/v1"
	taskqueuepb "go.temporal.io/api/taskqueue/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/server/api/adminservice/v1"
	"go.temporal.io/server/common/api"
	"go.temporal.io/server/common/authorization"
	"go.temporal.io/server/common/log"
	"go.temporal.io/server/common/metrics"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

var update = flag.Bool("update", false, "rewrite the testdata goldens from the service and request descriptors")

// The services the frontend exposes, at the pinned versions. Every method and
// stream in them must be classified by policy.go.
var serviceDescs = []grpc.ServiceDesc{
	workflowservice.WorkflowService_ServiceDesc,
	operatorservice.OperatorService_ServiceDesc,
	adminservice.AdminService_ServiceDesc,
}

// descriptorMethods returns the full API name of every method and stream in
// serviceDescs, in descriptor order.
func descriptorMethods(t *testing.T) []string {
	t.Helper()
	var out []string
	for _, d := range serviceDescs {
		for _, m := range d.Methods {
			out = append(out, "/"+d.ServiceName+"/"+m.MethodName)
		}
		for _, s := range d.Streams {
			out = append(out, "/"+d.ServiceName+"/"+s.StreamName)
		}
	}
	if len(out) < 140 {
		t.Fatalf("only %d methods reflected from the service descriptors; wrong pins?", len(out))
	}
	return out
}

// adrAllows is ADR-001 Annex A.2's table plus the groomed nava-operator row,
// written as rules (with the ADR's wildcards) rather than as method lists, and
// evaluated against the reflected descriptors. policy.go must agree with it.
func adrAllows(id Identity, fullAPI string) bool {
	svc, method := api.ServiceName(fullAPI), api.MethodName(fullAPI)
	isWF := svc == api.WorkflowServicePrefix
	oneOf := func(names ...string) bool {
		for _, n := range names {
			if method == n {
				return true
			}
		}
		return false
	}
	switch id {
	case System:
		return true
	case Relay:
		return isWF && oneOf("GetSystemInfo",
			"StartWorkflowExecution", "SignalWorkflowExecution", "SignalWithStartWorkflowExecution")
	case Core:
		return isWF && oneOf("GetSystemInfo",
			"RequestCancelWorkflowExecution", "UpdateWorkflowExecution", "QueryWorkflow",
			"DescribeWorkflowExecution", "GetWorkflowExecutionHistory")
	case Worker:
		return isWF && (oneOf("GetSystemInfo",
			"DescribeNamespace", "ShutdownWorker", "PollWorkflowTaskQueue", "PollActivityTaskQueue",
			"RecordActivityTaskHeartbeat", "RespondQueryTaskCompleted", "GetWorkflowExecutionHistory",
			"RecordWorkerHeartbeat") ||
			strings.HasPrefix(method, "RespondWorkflowTask") ||
			// T27c: the *ById forms need no task token, so the worker may not call them.
			(strings.HasPrefix(method, "RespondActivityTask") && !strings.HasSuffix(method, "ById")))
	case Operator:
		if fullAPI == api.WorkflowServicePrefix+"TerminateWorkflowExecution" ||
			fullAPI == api.OperatorServicePrefix+"AddSearchAttributes" {
			return true
		}
		if strings.Contains(method, "NexusEndpoint") {
			return false // "the Nexus endpoint calls" are denied
		}
		if svc == api.AdminServicePrefix && strings.Contains(method, "ReplicationMessages") {
			return false // they move replication ack levels: writes behind a Get name
		}
		if svc == api.AdminServicePrefix && strings.HasPrefix(method, "GetWorkflowExecutionRawHistory") {
			return false // skip the namespace gate; GetWorkflowExecutionHistory covers the need
		}
		if fullAPI == api.WorkflowServicePrefix+"DescribeWorker" {
			return false // new at api v1.53.0; ADR-001 A1 puts all five new methods in deniedByDefault
		}
		for _, p := range []string{"Describe", "List", "Count", "Get"} {
			if strings.HasPrefix(method, p) {
				return true
			}
		}
	}
	return false
}

func TestEveryDescriptorMethodIsClassifiedExactlyOnce(t *testing.T) {
	methods := descriptorMethods(t)
	inDesc := make(map[string]bool, len(methods))
	for _, m := range methods {
		inDesc[m] = true
	}

	allowedBy := map[string][]Identity{}
	for id, apis := range allowList {
		seen := map[string]bool{}
		for _, a := range apis {
			if seen[a] {
				t.Errorf("%s lists %s twice", id, a)
			}
			seen[a] = true
			allowedBy[a] = append(allowedBy[a], id)
		}
	}
	denied := map[string]bool{}
	for _, a := range deniedByDefault {
		if denied[a] {
			t.Errorf("deniedByDefault lists %s twice", a)
		}
		denied[a] = true
	}

	for _, m := range methods {
		switch {
		case len(allowedBy[m]) == 0 && !denied[m]:
			t.Errorf("unclassified: %s is neither allow-listed nor in deniedByDefault", m)
		case len(allowedBy[m]) > 0 && denied[m]:
			t.Errorf("contradiction: %s is allowed for %v and in deniedByDefault", m, allowedBy[m])
		}
	}
	for a := range allowedBy {
		if !inDesc[a] {
			t.Errorf("stale allow-list entry: %s is not in the pinned descriptors", a)
		}
	}
	for a := range denied {
		if !inDesc[a] {
			t.Errorf("stale deniedByDefault entry: %s is not in the pinned descriptors", a)
		}
	}
}

func TestAllowListMatchesADRRules(t *testing.T) {
	for _, m := range descriptorMethods(t) {
		for _, id := range []Identity{Relay, Core, Worker, Operator} {
			listed := false
			for _, a := range allowList[id] {
				listed = listed || a == m
			}
			if want := adrAllows(id, m); listed != want {
				t.Errorf("%s %s: policy.go allows=%v, ADR rule allows=%v", id, m, listed, want)
			}
		}
	}
	if _, ok := allowList[System]; ok {
		t.Error("temporal-system must not have a list: it is allowed everything")
	}
}

// testRequestHasNamespace is the test's own descriptor lookup.
func testRequestHasNamespace(t *testing.T, fullAPI string) bool {
	t.Helper()
	parts := strings.Split(strings.TrimPrefix(fullAPI, "/"), "/")
	d, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(parts[0]))
	if err != nil {
		t.Fatalf("descriptor for %s: %v", fullAPI, err)
	}
	m := d.(protoreflect.ServiceDescriptor).Methods().ByName(protoreflect.Name(parts[1]))
	if m == nil {
		t.Fatalf("no method descriptor for %s", fullAPI)
	}
	return m.Input().Fields().ByName("namespace") != nil
}

// The namespace rule keys off the request's `namespace` field. Cross-check
// that against the server's own scope metadata where the server has it.
func TestNamespaceScopeAgreesWithServerMetadata(t *testing.T) {
	for _, m := range descriptorMethods(t) {
		if api.ServiceName(m) == api.AdminServicePrefix {
			continue // the server marks every admin method cluster-scoped
		}
		scope := api.GetMethodMetadata(m).Scope
		has := testRequestHasNamespace(t, m)
		if (scope == api.ScopeNamespace) != has {
			t.Errorf("%s: server scope=%v, request has namespace field=%v", m, scope, has)
		}
		if requestHasNamespace(m) != has {
			t.Errorf("%s: authorizer and test disagree on the namespace field", m)
		}
	}
}

var (
	matrixIdentities = []Identity{Relay, Core, Worker, Operator, System, "", "nava-unknown"}
	matrixNamespaces = []string{Namespace, "", "default", "temporal-system", "NAVA"}
)

// TestAuthorizeMatrix: every (identity, API, namespace) triple over every
// descriptor method gets the decision the ADR rules give.
func TestAuthorizeMatrix(t *testing.T) {
	a := NewAuthorizer(log.NewNoopLogger(), metrics.NoopMetricsHandler)
	ctx := context.Background()
	allowed, denied := 0, 0
	for _, m := range descriptorMethods(t) {
		hasNS := testRequestHasNamespace(t, m)
		for _, id := range matrixIdentities {
			for _, ns := range matrixNamespaces {
				want := false
				switch {
				case id == System:
					want = true
				case id == "" || id == "nava-unknown":
					want = false
				default:
					nsOK := ns == Namespace || (ns == "" && !hasNS)
					want = adrAllows(id, m) && nsOK
				}
				res, err := a.Authorize(ctx, claimsFor(id), &authorization.CallTarget{APIName: m, Namespace: ns, Request: gateRequest(t, m)})
				if err != nil {
					t.Fatalf("Authorize returned error: %v", err)
				}
				got := res.Decision == authorization.DecisionAllow
				if got != want {
					t.Errorf("identity=%q api=%s namespace=%q: allowed=%v, want %v", id, m, ns, got, want)
				}
				if got {
					allowed++
				} else {
					denied++
				}
			}
		}
	}
	t.Logf("matrix: %d allowed, %d denied", allowed, denied)
}

// gateRequest is a request body that command inspection and the D9 workload
// checks allow, so that the matrix and golden table test the API gate alone.
// Inspection is commands_test.go and workload_test.go.
func gateRequest(t testing.TB, api string) any {
	const wfID = "t:tenantA:ticket:1"
	wt := &commonpb.WorkflowType{Name: platformType()}
	tq := &taskqueuepb.TaskQueue{Name: platformQueue()}
	switch api {
	case respondWorkflowTaskCompleted:
		return respond(taskToken(t, wfID))
	case wf + "StartWorkflowExecution":
		return &workflowservice.StartWorkflowExecutionRequest{Namespace: Namespace, WorkflowId: wfID, WorkflowType: wt, TaskQueue: tq}
	case wf + "SignalWithStartWorkflowExecution":
		return &workflowservice.SignalWithStartWorkflowExecutionRequest{Namespace: Namespace, WorkflowId: wfID, WorkflowType: wt, TaskQueue: tq}
	case wf + "SignalWorkflowExecution":
		return &workflowservice.SignalWorkflowExecutionRequest{Namespace: Namespace, WorkflowExecution: &commonpb.WorkflowExecution{WorkflowId: wfID}}
	case wf + "PollWorkflowTaskQueue":
		return &workflowservice.PollWorkflowTaskQueueRequest{Namespace: Namespace, TaskQueue: tq}
	case wf + "PollActivityTaskQueue":
		return &workflowservice.PollActivityTaskQueueRequest{Namespace: Namespace, TaskQueue: tq}
	case wf + "RespondWorkflowTaskFailed":
		return &workflowservice.RespondWorkflowTaskFailedRequest{Namespace: Namespace, TaskToken: taskToken(t, wfID)}
	case wf + "RespondActivityTaskCompleted":
		return &workflowservice.RespondActivityTaskCompletedRequest{Namespace: Namespace, TaskToken: taskToken(t, wfID)}
	case wf + "RespondActivityTaskFailed":
		return &workflowservice.RespondActivityTaskFailedRequest{Namespace: Namespace, TaskToken: taskToken(t, wfID)}
	case wf + "RespondActivityTaskCanceled":
		return &workflowservice.RespondActivityTaskCanceledRequest{Namespace: Namespace, TaskToken: taskToken(t, wfID)}
	case wf + "RecordActivityTaskHeartbeat":
		return &workflowservice.RecordActivityTaskHeartbeatRequest{Namespace: Namespace, TaskToken: taskToken(t, wfID)}
	case wf + "RespondQueryTaskCompleted":
		return &workflowservice.RespondQueryTaskCompletedRequest{Namespace: Namespace, TaskToken: queryToken(t, tq.GetName())}
	case wf + "ShutdownWorker":
		return &workflowservice.ShutdownWorkerRequest{Namespace: Namespace}
	case wf + "RecordWorkerHeartbeat":
		return &workflowservice.RecordWorkerHeartbeatRequest{Namespace: Namespace}
	}
	if req := coreWorkflowIDRequests(wfID)[strings.TrimPrefix(api, wf)]; req != nil {
		return req
	}
	return nil
}

func claimsFor(id Identity) *authorization.Claims {
	if id == "" {
		return nil
	}
	return &authorization.Claims{Subject: string(id)}
}

// TestAllowListGolden regenerates the human-readable allow/deny table from the
// descriptors and the Authorizer, and fails if the checked-in copy differs.
// Run `go test ./authz -run TestAllowListGolden -update` after a reviewed change.
func TestAllowListGolden(t *testing.T) {
	a := NewAuthorizer(log.NewNoopLogger(), metrics.NoopMetricsHandler)
	ids := []Identity{Relay, Core, Worker, Operator, System}
	var b strings.Builder
	b.WriteString("<!-- Generated by `go test ./authz -run TestAllowListGolden -update`. Do not edit. -->\n")
	b.WriteString("# Temporal frontend allow-list\n\n")
	b.WriteString("go.temporal.io/api v1.53.0, go.temporal.io/server v1.29.7. ")
	b.WriteString("`Y` = allowed in namespace `nava` (or with no namespace, for APIs whose request has none). ")
	b.WriteString("Anything else, any other namespace, and any other certificate are denied.\n")
	for _, d := range serviceDescs {
		fmt.Fprintf(&b, "\n## %s\n\n| Method | ns | %s |\n|---|---|%s\n",
			d.ServiceName, joinIDs(ids), strings.Repeat("---|", len(ids)))
		var names []string
		for _, m := range d.Methods {
			names = append(names, m.MethodName)
		}
		for _, s := range d.Streams {
			names = append(names, s.StreamName+" (stream)")
		}
		sort.Strings(names)
		for _, n := range names {
			full := "/" + d.ServiceName + "/" + strings.TrimSuffix(n, " (stream)")
			scope := "-"
			if testRequestHasNamespace(t, full) {
				scope = "nava"
			}
			fmt.Fprintf(&b, "| %s | %s |", n, scope)
			for _, id := range ids {
				ns := ""
				if scope == "nava" {
					ns = Namespace
				}
				res, _ := a.Authorize(context.Background(), claimsFor(id), &authorization.CallTarget{APIName: full, Namespace: ns, Request: gateRequest(t, full)})
				cell := " |"
				if res.Decision == authorization.DecisionAllow {
					cell = " Y |"
				}
				b.WriteString(cell)
			}
			b.WriteString("\n")
		}
	}

	writeWorkloadTable(&b)

	path := filepath.Join("testdata", "allowlist.golden.md")
	if *update {
		if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden (run with -update to create): %v", err)
	}
	if strings.ReplaceAll(string(want), "\r\n", "\n") != b.String() {
		t.Errorf("%s is stale; review the policy change and rerun with -update", path)
	}
}

// writeWorkloadTable prints the D9 sub-table: the APIs whose requests are
// checked against the workload allow-list, and both compiled-in sets.
func writeWorkloadTable(b *strings.Builder) {
	b.WriteString("\n## D9 workload allow-list\n\n")
	b.WriteString("A request to one of these APIs is allowed only if every workflow type, task queue and workflow id it names is allow-listed. ")
	b.WriteString("`nava-worker` `RespondWorkflowTaskCompleted` commands, query results and `StickyAttributes` are checked the same way (E1, D9c, D9h-failures). ")
	b.WriteString("`temporal-system` is exempt, and `nava-operator` is outside D9.\n\n")
	b.WriteString("| Identity | Method | Rule |\n|---|---|---|\n")
	var rows []string
	for id, apis := range workloadChecks {
		for api, c := range apis {
			rows = append(rows, fmt.Sprintf("| `%s` | %s | %s |\n", id, strings.TrimPrefix(api, wf), c.rule))
		}
	}
	sort.Strings(rows)
	for _, r := range rows {
		b.WriteString(r)
	}
	b.WriteString("\n| Set | Entries |\n|---|---|\n")
	for _, set := range []struct {
		name    string
		entries map[string]struct{}
	}{{"platformWorkflowTypes", platformWorkflowTypes}, {"platformTaskQueues", platformTaskQueues}} {
		fmt.Fprintf(b, "| `%s` | `%s` |\n", set.name, strings.Join(sortedKeys(set.entries), "`, `"))
	}
	fmt.Fprintf(b, "\nA STICKY queue name must match `%s`, must not be in `platformTaskQueues`, and its `NormalName` must be in `platformTaskQueues`.\n", stickyQueueName)
	b.WriteString("\n| Rule | Must hold |\n|---|---|\n")
	b.WriteString("| D9b+ | `DeploymentOptions.WorkerVersioningMode` is unspecified or `UNVERSIONED`, and `WorkerVersionCapabilities.UseVersioning` is false. |\n")
	b.WriteString("| D9d | `TaskToken` decodes as `token.v1.Task` and its `WorkflowId` has a `t:<tenant>:` prefix. |\n")
	b.WriteString("| D9d query clause | `TaskToken` decodes as `token.v1.QueryTask` and its `TaskQueue` is in `platformTaskQueues` or sticky-shaped. |\n")
	b.WriteString("| D9e, D9f | The request's workflow id has a `t:<tenant>:` prefix. |\n")
	b.WriteString("| D9c | A command's task queue has Kind NORMAL or UNSPECIFIED and an empty name or one in `platformTaskQueues`; a child or continue-as-new names a type in `platformWorkflowTypes`. |\n")
	b.WriteString("| D9g | `StickyTaskQueue` is empty or sticky-shaped and not in `platformTaskQueues`; every `WorkerHeartbeat.TaskQueue`, on polls too, is in `platformTaskQueues`. |\n")
	b.WriteString("| D9h-links | Every link is a workflow event link whose `WorkflowId` has a `t:<tenant>:` prefix. |\n")
	b.WriteString("| D9h-failures | Every `ChildWorkflowExecutionFailureInfo` workflow id along a failure's `cause` chain has a `t:<tenant>:` prefix. |\n")
}

func joinIDs(ids []Identity) string {
	s := make([]string, len(ids))
	for i, id := range ids {
		s[i] = "`" + string(id) + "`"
	}
	return strings.Join(s, " | ")
}

// The CI-C drift golden (ADR-001-A2 "Drift golden (CI-C)", plan task T27k). It
// lists every field path of the request message of every API allow-listed for
// nava-relay, nava-core and nava-worker, generated from allowList, with the rule
// that checks each field. A go.temporal.io/api bump that adds, removes, renames
// or retypes a field changes the golden and fails CI until a policy PR reviews
// it. A field shaped like a workflow id, workflow type or task queue (idShaped)
// fails even when the golden matches, unless requestFieldRules names its rule.
//
// Known at api v1.53.0: sdk-go v1.49.0 (api v1.63.5) also sends
// ShutdownWorkerRequest.TaskQueue, TaskQueueTypes and WorkerInstanceKey. The
// pinned api has only fields 1-5, so neither the Authorizer nor this golden
// sees them. The api bump that adds them (v1.63.x) fails this test, and that
// policy PR must classify ShutdownWorker `task_queue` as "platform queue": a
// D9g check in checkShutdownWorker plus a requestFieldRules entry.

// notIDOrQueue is the class of every field that names no workflow id, workflow
// type or task queue and that no rule reads.
const notIDOrQueue = "not an id or queue"

// d9Identities are the identities whose allow-listed requests the golden walks.
var d9Identities = []Identity{Relay, Core, Worker}

// Field classes, as `<rule>: <what must hold>`. The rule part must be one the
// Authorizer runs on that API (workloadChecks, or inspectCommands for
// RespondWorkflowTaskCompleted); TestRequestFieldRulesMatchAuthorizer checks it.
const (
	classGate            = "API gate: namespace `nava`"
	classD9aID           = "D9a: `t:<tenant>:` prefix"
	classD9aType         = "D9a: platform workflow type"
	classD9aQueue        = "D9a: platform queue"
	classD9aKind         = "D9a: NORMAL or UNSPECIFIED"
	classD9aNormalName   = "D9a: unused, as Kind is not STICKY (tqid.PartitionFromProto rejects it)"
	classD9bQueue        = "D9b: platform queue, or sticky-shaped when STICKY"
	classD9bKind         = "D9b: selects the NORMAL or the STICKY rule"
	classD9bNormalName   = "D9b: platform queue when STICKY, else unused (tqid rejects it)"
	classD9bActQueue     = "D9b: platform queue"
	classD9bActKind      = "D9b: NORMAL or UNSPECIFIED"
	classD9bActNormal    = "D9b: unused, as Kind is not STICKY (tqid.PartitionFromProto rejects it)"
	classD9bPlusMode     = "D9b+: unspecified or UNVERSIONED"
	classD9bPlusCaps     = "D9b+: false"
	classD9dToken        = "D9d: `token.v1.Task` with a `t:<tenant>:` workflow id"
	classD9dQueryToken   = "D9d query clause: `token.v1.QueryTask` with a platform or sticky-shaped queue"
	classD9eID           = "D9e: `t:<tenant>:` prefix"
	classD9fID           = "D9f: `t:<tenant>:` prefix"
	classD9gSticky       = "D9g: empty, or sticky-shaped and not a platform queue"
	classD9gHeartbeat    = "D9g: platform queue"
	classE1Token         = "E1: `token.v1.Task` with a `t:<tenant>:` workflow id"
	classE1CommandType   = "E1: allow-listed command type"
	classE1Target        = "E1: the task's tenant prefix"
	classE1Namespace     = "E1: `nava` or empty"
	classE1Header        = "E1: `nava-tenant` header is the task's tenant"
	classE1SearchAttr    = "E1: a `NavaTenantId` it sets is the task's tenant"
	classE1Cron          = "E1: empty"
	classD9cStickyQueue  = "D9c: sticky-shaped, not a platform queue"
	classD9cStickyNormal = "D9c: platform queue"
	classD9cStickyKind   = "D9c: the sticky rule applies whatever Kind says"
	classD9cQueue        = "D9c: empty (inherited) or a platform queue"
	classD9cKind         = "D9c: NORMAL or UNSPECIFIED"
	classD9cNormalName   = "D9c: unused, as Kind is not STICKY (tqid.PartitionFromProto rejects it)"
	classD9cType         = "D9c: platform workflow type"
	classD9gPollHB       = "D9g: platform queue (the frontend forwards it to RecordWorkerHeartbeat)"
	classD9hLink         = "D9h-links: workflow event link with a `t:<tenant>:` prefix"
	classD9hFailure      = "D9h-failures: `t:<tenant>:` prefix, along the whole cause chain"
)

// failureTypeName is the one id-shaped field classed notIDOrQueue on purpose:
// a child failure's workflow type is failure data, and the Annex A.2 statement
// covers workflow ids and task queues, not type names (Technical Lead,
// 2026-10-25, docs/delivery/decision-log.md).
const failureTypeName = "child_workflow_execution_failure_info.workflow_type.name"

// withFailures adds the D9h-failures rules under each Failure-typed field path
// prefix (ending in "."). The walk stops at `cause`, so these paths stand for
// the whole cause chain, which failureChildID walks.
func withFailures(rules map[string]string, prefixes ...string) map[string]string {
	for _, p := range prefixes {
		rules[p+"child_workflow_execution_failure_info.workflow_execution.workflow_id"] = classD9hFailure
		rules[p+failureTypeName] = notIDOrQueue
	}
	return rules
}

// withQueues adds the D9c rules for the command task queue under each prefix.
func withQueues(rules map[string]string, prefixes ...string) map[string]string {
	for _, p := range prefixes {
		rules[p+"task_queue.name"] = classD9cQueue
		rules[p+"task_queue.kind"] = classD9cKind
		rules[p+"task_queue.normal_name"] = classD9cNormalName
	}
	return rules
}

const (
	cmdActivity = "commands[].schedule_activity_task_command_attributes."
	cmdChild    = "commands[].start_child_workflow_execution_command_attributes."
	cmdSignal   = "commands[].signal_external_workflow_execution_command_attributes."
	cmdCancel   = "commands[].request_cancel_external_workflow_execution_command_attributes."
	cmdCAN      = "commands[].continue_as_new_workflow_execution_command_attributes."
	cmdUpsert   = "commands[].upsert_workflow_search_attributes_command_attributes."
	cmdFail     = "commands[].fail_workflow_execution_command_attributes."
	cmdMarker   = "commands[].record_marker_command_attributes."
)

// requestFieldRules is the golden's classification column: per identity and
// method, the request field paths a rule reads. A key ending in "." covers
// that whole subtree. Every API's top-level `namespace` is classGate, and any
// other field prints as notIDOrQueue. Adding an entry is a policy PR with
// Security review.
var requestFieldRules = map[Identity]map[string]map[string]string{
	Relay: {
		"StartWorkflowExecution": withFailures(map[string]string{
			"workflow_id": classD9aID, "workflow_type.name": classD9aType,
			"task_queue.name": classD9aQueue, "task_queue.kind": classD9aKind, "task_queue.normal_name": classD9aNormalName,
			"links[].workflow_event.workflow_id":                        classD9hLink,
			"completion_callbacks[].links[].workflow_event.workflow_id": classD9hLink,
		}, "continued_failure."),
		"SignalWithStartWorkflowExecution": {
			"workflow_id": classD9aID, "workflow_type.name": classD9aType,
			"task_queue.name": classD9aQueue, "task_queue.kind": classD9aKind, "task_queue.normal_name": classD9aNormalName,
			"links[].workflow_event.workflow_id": classD9hLink,
		},
		"SignalWorkflowExecution": {"workflow_execution.workflow_id": classD9aID, "links[].workflow_event.workflow_id": classD9hLink},
	},
	Core: {
		"RequestCancelWorkflowExecution": {"workflow_execution.workflow_id": classD9fID, "links[].workflow_event.workflow_id": classD9hLink},
		"UpdateWorkflowExecution":        {"workflow_execution.workflow_id": classD9fID},
		"QueryWorkflow":                  {"execution.workflow_id": classD9fID},
		"DescribeWorkflowExecution":      {"execution.workflow_id": classD9fID},
		"GetWorkflowExecutionHistory":    {"execution.workflow_id": classD9fID},
	},
	Worker: {
		"PollWorkflowTaskQueue": {
			"task_queue.name": classD9bQueue, "task_queue.kind": classD9bKind, "task_queue.normal_name": classD9bNormalName,
			"deployment_options.worker_versioning_mode":  classD9bPlusMode,
			"worker_version_capabilities.use_versioning": classD9bPlusCaps,
			"worker_heartbeat.task_queue":                classD9gPollHB,
		},
		"PollActivityTaskQueue": {
			"task_queue.name": classD9bActQueue, "task_queue.kind": classD9bActKind, "task_queue.normal_name": classD9bActNormal,
			"deployment_options.worker_versioning_mode":  classD9bPlusMode,
			"worker_version_capabilities.use_versioning": classD9bPlusCaps,
			"worker_heartbeat.task_queue":                classD9gHeartbeat,
		},
		"RespondWorkflowTaskCompleted": withQueues(withFailures(map[string]string{
			"task_token": classE1Token,
			"sticky_attributes.worker_task_queue.name":        classD9cStickyQueue,
			"sticky_attributes.worker_task_queue.normal_name": classD9cStickyNormal,
			"sticky_attributes.worker_task_queue.kind":        classD9cStickyKind,
			"commands[].command_type":                         classE1CommandType,
			cmdActivity + "header.":                           classE1Header,
			cmdChild + "namespace":                            classE1Namespace,
			cmdChild + "workflow_id":                          classE1Target,
			cmdChild + "header.":                              classE1Header,
			cmdChild + "search_attributes.":                   classE1SearchAttr,
			cmdChild + "workflow_type.name":                   classD9cType,
			cmdSignal + "namespace":                           classE1Namespace,
			cmdSignal + "execution.workflow_id":               classE1Target,
			cmdCancel + "namespace":                           classE1Namespace,
			cmdCancel + "workflow_id":                         classE1Target,
			cmdCAN + "cron_schedule":                          classE1Cron,
			cmdCAN + "header.":                                classE1Header,
			cmdCAN + "search_attributes.":                     classE1SearchAttr,
			cmdCAN + "workflow_type.name":                     classD9cType,
			cmdUpsert + "search_attributes.":                  classE1SearchAttr,
		}, "query_results{}.failure.", cmdCAN+"failure.", cmdFail+"failure.", cmdMarker+"failure."),
			cmdActivity, cmdChild, cmdCAN),
		"RespondWorkflowTaskFailed":    withFailures(map[string]string{"task_token": classD9dToken}, "failure."),
		"RespondActivityTaskCompleted": {"task_token": classD9dToken},
		"RespondActivityTaskFailed":    withFailures(map[string]string{"task_token": classD9dToken}, "failure."),
		"RespondActivityTaskCanceled":  {"task_token": classD9dToken},
		"RecordActivityTaskHeartbeat":  {"task_token": classD9dToken},
		"RespondQueryTaskCompleted":    withFailures(map[string]string{"task_token": classD9dQueryToken}, "failure."),
		"GetWorkflowExecutionHistory":  {"execution.workflow_id": classD9eID},
		"ShutdownWorker":               {"sticky_task_queue": classD9gSticky, "worker_heartbeat.task_queue": classD9gHeartbeat},
		"RecordWorkerHeartbeat":        {"worker_heartbeat[].task_queue": classD9gHeartbeat},
	},
}

// idShapedName matches the names of fields that name a workflow id, workflow
// type or task queue, or a task token, which carries one.
var idShapedName = regexp.MustCompile(`(^|_)(workflow_id|workflow_type|task_queue|task_queue_name|task_token)$|^normal_name$`)

// idShaped reports whether a string or bytes field names a workflow id,
// workflow type or task queue: by its name, or as the `name` of a WorkflowType
// or TaskQueue message.
func idShaped(f requestField) bool {
	if f.kind != protoreflect.StringKind && f.kind != protoreflect.BytesKind {
		return false
	}
	if idShapedName.MatchString(f.name) {
		return true
	}
	return f.name == "name" && (f.parent == "temporal.api.common.v1.WorkflowType" || f.parent == "temporal.api.taskqueue.v1.TaskQueue")
}

// requestField is one row of the walk: a field path in one request message.
type requestField struct {
	path   string // e.g. `commands[].start_child_workflow_execution_command_attributes.workflow_id`
	typ    string // the scalar kind, or the enum or message full name
	name   string // the field's own name
	parent protoreflect.FullName
	kind   protoreflect.Kind
	cycle  bool // a message type already on the path; the walk stops here
}

// classMessage is the class of a message-typed field that no subtree rule
// covers: its own fields carry the classes.
const classMessage = "message: see its fields"

// walkRequestFields returns every field path of md, recursing into nested
// messages, sorted by path. `[]` marks a repeated field and `{}` a map value.
// A message type already on the current path prints as `failure.cause →
// (Failure)` and is not entered again: temporal.api.failure.v1.Failure contains
// itself (ADR-001-A2 implementation note).
func walkRequestFields(md protoreflect.MessageDescriptor) []requestField {
	var out []requestField
	onPath := map[protoreflect.FullName]bool{}
	var walk func(md protoreflect.MessageDescriptor, prefix string)
	walk = func(md protoreflect.MessageDescriptor, prefix string) {
		onPath[md.FullName()] = true
		defer delete(onPath, md.FullName())
		fields := md.Fields()
		for i := 0; i < fields.Len(); i++ {
			f := fields.Get(i)
			v, path := f, prefix+string(f.Name())
			switch {
			case f.IsMap():
				v, path = f.MapValue(), path+"{}"
			case f.IsList():
				path += "[]"
			}
			row := requestField{path: path, typ: v.Kind().String(), name: string(f.Name()), parent: md.FullName(), kind: v.Kind()}
			switch {
			case v.Enum() != nil:
				row.typ = string(v.Enum().FullName())
			case v.Message() != nil:
				row.typ = string(v.Message().FullName())
				if onPath[v.Message().FullName()] {
					row.path += " → (" + string(v.Message().Name()) + ")"
					row.cycle = true
				}
			}
			out = append(out, row)
			if v.Message() != nil && !row.cycle {
				walk(v.Message(), path+".")
			}
		}
	}
	walk(md, "")
	sort.Slice(out, func(i, j int) bool { return out[i].path < out[j].path })
	return out
}

// lookupRule returns the class rules give path: an exact key, or else the
// longest key ending in "." that prefixes it.
func lookupRule(rules map[string]string, path string) (string, bool) {
	if c, ok := rules[path]; ok {
		return c, true
	}
	best, class := "", ""
	for k, c := range rules {
		if strings.HasSuffix(k, ".") && strings.HasPrefix(path, k) && len(k) > len(best) {
			best, class = k, c
		}
	}
	return class, best != ""
}

// allowedRequest is one API that at least one D9 identity may call.
type allowedRequest struct {
	method string // the short method name, the key of requestFieldRules
	ids    []Identity
	input  protoreflect.MessageDescriptor
}

// d9AllowedRequests reads the APIs allow-listed for d9Identities from allowList
// and the registered descriptors, sorted by method.
func d9AllowedRequests(t testing.TB) []allowedRequest {
	t.Helper()
	byAPI := map[string]*allowedRequest{}
	for _, id := range d9Identities {
		for _, full := range allowList[id] {
			r := byAPI[full]
			if r == nil {
				parts := strings.Split(strings.TrimPrefix(full, "/"), "/")
				d, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(parts[0]))
				if err != nil {
					t.Fatalf("descriptor for %s: %v", full, err)
				}
				m := d.(protoreflect.ServiceDescriptor).Methods().ByName(protoreflect.Name(parts[1]))
				if m == nil {
					t.Fatalf("no method descriptor for %s", full)
				}
				r = &allowedRequest{method: parts[1], input: m.Input()}
				byAPI[full] = r
			}
			r.ids = append(r.ids, id)
		}
	}
	out := make([]allowedRequest, 0, len(byAPI))
	for _, r := range byAPI {
		out = append(out, *r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].method < out[j].method })
	return out
}

// classify returns a field's class column, and the identities for which an
// id- or queue-shaped field has no rule. When the identities that may call
// the API disagree, each class carries its identity.
func classify(r allowedRequest, f requestField) (class string, missing []Identity) {
	if f.path == "namespace" {
		return classGate, nil
	}
	classes := make([]string, len(r.ids))
	for i, id := range r.ids {
		c, ok := lookupRule(requestFieldRules[id][r.method], f.path)
		switch {
		case ok:
			classes[i] = c
		case idShaped(f):
			missing = append(missing, id)
		default:
			classes[i] = notIDOrQueue
		}
	}
	if len(missing) > 0 {
		return "**unclassified**", missing
	}
	if !slices.ContainsFunc(classes, func(c string) bool { return c != classes[0] }) {
		return classes[0], nil
	}
	parts := make([]string, len(classes))
	for i, c := range classes {
		parts[i] = fmt.Sprintf("%s (`%s`)", c, r.ids[i])
	}
	return strings.Join(parts, "; "), nil
}

// renderRequestFieldGolden prints the golden for reqs and returns its
// unclassified id- or queue-shaped fields, one line each.
func renderRequestFieldGolden(reqs []allowedRequest) (golden string, unclassified []string) {
	var b strings.Builder
	b.WriteString("<!-- Generated by `go test ./authz -run TestRequestFieldGolden -update`. Do not edit. -->\n")
	b.WriteString("# Request fields of the allow-listed D9 APIs\n\n")
	b.WriteString("go.temporal.io/api v1.53.0. Every field path of the request message of each API that ")
	b.WriteString("`nava-relay`, `nava-core` or `nava-worker` may call (ADR-001-A2, drift golden CI-C). ")
	b.WriteString("`[]` marks a repeated field, `{}` a map value, and `→ (Type)` a message type already on the path, where the walk stops. ")
	b.WriteString("A field that names a workflow id, workflow type or task queue must name the rule that checks it.\n")
	for _, r := range reqs {
		ids := make([]string, len(r.ids))
		for i, id := range r.ids {
			ids[i] = "`" + string(id) + "`"
		}
		fmt.Fprintf(&b, "\n## %s\n\n`%s`, allowed for %s.\n\n", r.method, r.input.FullName(), strings.Join(ids, ", "))
		fields := walkRequestFields(r.input)
		if len(fields) == 0 {
			b.WriteString("No fields.\n")
			continue
		}
		b.WriteString("| Field | Type | Class |\n|---|---|---|\n")
		for _, f := range fields {
			class, missing := classify(r, f)
			switch {
			case f.cycle:
				class = "cycle: its fields are listed above"
			case f.kind == protoreflect.MessageKind && class == notIDOrQueue:
				class = classMessage
			}
			fmt.Fprintf(&b, "| `%s` | `%s` | %s |\n", f.path, f.typ, class)
			if len(missing) > 0 {
				unclassified = append(unclassified, fmt.Sprintf("%s %s for %v", r.method, f.path, missing))
			}
		}
	}
	return b.String(), unclassified
}

// goldenMismatch returns nil if got equals want (ignoring CRLF), or an error
// naming the first line that differs.
func goldenMismatch(want, got string) error {
	want = strings.ReplaceAll(want, "\r\n", "\n")
	if want == got {
		return nil
	}
	w, g := strings.Split(want, "\n"), strings.Split(got, "\n")
	for i := 0; i < len(w) || i < len(g); i++ {
		var wl, gl string
		if i < len(w) {
			wl = w[i]
		}
		if i < len(g) {
			gl = g[i]
		}
		if wl != gl {
			return fmt.Errorf("line %d: golden has %q, descriptors give %q", i+1, wl, gl)
		}
	}
	return errors.New("goldens differ")
}

// requestFieldProblems is TestRequestFieldGolden's verdict: a stale golden, and
// every unclassified id- or queue-shaped field even when the golden matches.
func requestFieldProblems(want, got string, unclassified []string) []string {
	var out []string
	if err := goldenMismatch(want, got); err != nil {
		out = append(out, "testdata/request-fields.golden.md is stale; classify the changed fields and rerun with -update: "+err.Error())
	}
	for _, u := range unclassified {
		out = append(out, "id- or queue-shaped field with no rule; classify it in requestFieldRules (policy PR): "+u)
	}
	return out
}

// TestRequestFieldGolden is CI-C. Run
// `go test ./authz -run TestRequestFieldGolden -update` after a reviewed change.
func TestRequestFieldGolden(t *testing.T) {
	got, unclassified := renderRequestFieldGolden(d9AllowedRequests(t))
	if again, _ := renderRequestFieldGolden(d9AllowedRequests(t)); again != got {
		t.Fatal("the request-field golden is not byte-stable across renders")
	}
	path := filepath.Join("testdata", "request-fields.golden.md")
	if *update {
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden (run with -update to create): %v", err)
	}
	for _, p := range requestFieldProblems(string(want), got, unclassified) {
		t.Error(p)
	}
}

// TestRequestFieldRulesMatchAuthorizer keeps the classification column honest:
// every entry names a field that exists, in an API the identity may call, and
// a rule the Authorizer runs on that API.
func TestRequestFieldRulesMatchAuthorizer(t *testing.T) {
	reqs := map[string]allowedRequest{}
	for _, r := range d9AllowedRequests(t) {
		reqs[r.method] = r
	}
	for id, methods := range requestFieldRules {
		for method, rules := range methods {
			r, ok := reqs[method]
			if !ok || !slices.Contains(r.ids, id) {
				t.Errorf("%s %s: classified, but not allow-listed for %s", id, method, id)
				continue
			}
			var paths []string
			for _, f := range walkRequestFields(r.input) {
				paths = append(paths, f.path)
			}
			ran := map[string]bool{}
			if c, ok := workloadChecks[id][wf+method]; ok {
				for _, rule := range strings.Split(c.rule, ", ") {
					ran[rule] = true
				}
			}
			if id == Worker && wf+method == respondWorkflowTaskCompleted {
				ran["E1"], ran["D9c"], ran["D9h-failures"] = true, true, true // inspectCommands
			}
			for key, class := range rules {
				matched := slices.ContainsFunc(paths, func(p string) bool {
					return p == key || (strings.HasSuffix(key, ".") && strings.HasPrefix(p, key))
				})
				if !matched {
					t.Errorf("%s %s: stale rule entry %q matches no request field", id, method, key)
				}
				if class == notIDOrQueue {
					if !strings.HasSuffix(key, failureTypeName) {
						t.Errorf("%s %s %s: only a child failure's workflow type may be classed %q", id, method, key, notIDOrQueue)
					}
					continue
				}
				if rule, _, _ := strings.Cut(class, ":"); !ran[rule] {
					t.Errorf("%s %s %s: class names %s, but the Authorizer runs %v there", id, method, key, rule, ran)
				}
			}
		}
	}
}

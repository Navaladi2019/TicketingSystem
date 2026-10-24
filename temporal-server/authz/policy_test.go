package authz

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
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

var update = flag.Bool("update", false, "rewrite testdata/allowlist.golden.md from the service descriptors")

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
	b.WriteString("`nava-worker` `RespondWorkflowTaskCompleted` commands and `StickyAttributes` are checked the same way (E1, D9c). ")
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
	b.WriteString("| D9g | `StickyTaskQueue` is empty or sticky-shaped and not in `platformTaskQueues`; every `WorkerHeartbeat.TaskQueue` is in `platformTaskQueues`. |\n")
}

func joinIDs(ids []Identity) string {
	s := make([]string, len(ids))
	for i, id := range ids {
		s[i] = "`" + string(id) + "`"
	}
	return strings.Join(s, " | ")
}

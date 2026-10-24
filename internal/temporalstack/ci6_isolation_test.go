//go:build temporalstack

package temporalstack

// CI-6, first block (ADR-001 Annex B, erratum E1; PLAN-001 task T32a): the
// certificate, network and Authorizer cases, against the compose Temporal over
// mTLS. Nothing is stubbed: every call goes through the gateway to the custom
// server and its Authorizer. One named test per contract case, each named
// TestCI6_*; `make stack-test-sdk` writes the report that lists them
// (tools/ci/ci6report).
//
// The threat model is a platform process holding one valid certificate (or a
// certificate it should not have) that tries to reach another tenant's
// workflows, or an API outside its row of the allow-list.

import (
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	commandpb "go.temporal.io/api/command/v1"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/operatorservice/v1"
	"go.temporal.io/api/serviceerror"
	taskqueuepb "go.temporal.io/api/taskqueue/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/dynamicpb"
)

// wfAPI prefixes a WorkflowService method's full name.
const wfAPI = "/temporal.api.workflowservice.v1.WorkflowService/"

func ci6Context(t *testing.T) context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	t.Cleanup(cancel)
	return ctx
}

// --- certificates ------------------------------------------------------------

// A nava-relay certificate signed by another CA fails the handshake, through the
// gateway, with a TLS alert from the server; so does no certificate. The real
// nava-relay certificate, on the same code path, gets no alert.
func TestCI6_RogueCACertificateRejectedAtHandshake(t *testing.T) {
	if err := handshakeAlert(t, pairOf(t, "nava-relay")); err != nil {
		t.Fatalf("control: the installation nava-relay certificate was refused: %v", err)
	}
	for name, certs := range map[string][]tls.Certificate{
		"rogue-CA nava-relay": {rogueCertificate(t)},
		"no certificate":      nil,
	} {
		err := handshakeAlert(t, certs...)
		if err == nil || !strings.Contains(err.Error(), "remote error: tls:") {
			t.Errorf("%s: got %v, want a TLS alert from the server", name, err)
		}
	}
}

// A certificate from the installation CA whose CN is no identity completes the
// handshake, and the Authorizer denies every call it makes, health excepted.
func TestCI6_UnlistedCertificateDeniedByAuthorizer(t *testing.T) {
	ctx := ci6Context(t)
	if err := handshakeAlert(t, pairOf(t, unlistedCN)); err != nil {
		t.Fatalf("the %s certificate should pass the handshake (it is from the installation CA): %v", unlistedCN, err)
	}
	conn := rawConnAs(t, unlistedCN)
	wf := workflowservice.NewWorkflowServiceClient(conn)
	labels := map[string]string{"identity": "none", "reason": "no_identity"}
	before := deniedCount(t, labels)
	mark := logMark()
	// The anchor: a deny only this run can cause, its unique namespace logged.
	runNamespace := fmt.Sprintf("ci6-run-%d", time.Now().UnixNano())
	_, err := wf.DescribeNamespace(ctx, &workflowservice.DescribeNamespaceRequest{Namespace: runNamespace})
	wantDenied(t, unlistedCN+" DescribeNamespace (anchor)", err)

	_, err = wf.GetSystemInfo(ctx, &workflowservice.GetSystemInfoRequest{})
	wantDenied(t, unlistedCN+" GetSystemInfo", err)
	_, err = wf.DescribeNamespace(ctx, &workflowservice.DescribeNamespaceRequest{Namespace: namespace})
	wantDenied(t, unlistedCN+" DescribeNamespace", err)
	startID := uniqueID("A", "unlisted")
	_, err = wf.StartWorkflowExecution(ctx, &workflowservice.StartWorkflowExecutionRequest{
		Namespace: namespace, WorkflowId: startID, RequestId: fmt.Sprint(time.Now().UnixNano()),
		WorkflowType: &commonpb.WorkflowType{Name: echoType},
		TaskQueue:    &taskqueuepb.TaskQueue{Name: taskQueue, Kind: enumspb.TASK_QUEUE_KIND_NORMAL},
	})
	wantDenied(t, unlistedCN+" StartWorkflowExecution", err)
	_, err = operatorservice.NewOperatorServiceClient(conn).ListSearchAttributes(ctx,
		&operatorservice.ListSearchAttributesRequest{Namespace: namespace})
	wantDenied(t, unlistedCN+" ListSearchAttributes", err)

	line := func(api string) map[string]string {
		return map[string]string{"msg": "Temporal API call denied", "identity": "none", "reason": "no_identity",
			"cert_subject": "~CN=" + unlistedCN, "api": api}
	}
	anchor := line(wfAPI + "DescribeNamespace")
	anchor["namespace"] = runNamespace
	awaitDenyLogsAfter(t, mark, anchor, []map[string]string{
		line(wfAPI + "GetSystemInfo"), line(wfAPI + "DescribeNamespace"), line(wfAPI + "StartWorkflowExecution"),
		line("/temporal.api.operatorservice.v1.OperatorService/ListSearchAttributes"),
	})
	awaitDeniedCount(t, labels, before, 5)
	var notFound *serviceerror.NotFound
	if _, err := dial(t, "temporal-system").DescribeWorkflowExecution(ctx, startID, ""); !errors.As(err, &notFound) {
		t.Errorf("the denied start %s: got %v, want NotFound", startID, err)
	}
}

// --- network -------------------------------------------------------------------

// A container on the platform network (`internal`, where nava-core and every
// platform process live) reaches the frontend through the gateway and nothing
// else: not the internode ports, not membership, not metrics, and the server
// itself does not resolve.
func TestCI6_PlatformNetworkCannotReachInternalPorts(t *testing.T) {
	var targets []string
	for _, host := range []string{"temporal", "temporal-server"} {
		for _, port := range []string{"7234", "7235", "7239", "6933", "6934", "6935", "6936", "6937", "6938", "6939", "9090"} {
			targets = append(targets, host+":"+port)
		}
	}
	targets = append(targets, "temporal-server:7233")
	// One container probes every target (and the 7233 control) from the
	// platform network and prints OPEN or CLOSED for each.
	script := `for a in temporal:7233 ` + strings.Join(targets, " ") + `; do
		if /usr/local/bin/temporal-server health -tcp-only -addr "$a" -timeout 2s >/dev/null 2>&1; then echo "OPEN $a"; else echo "CLOSED $a"; fi
	done`
	out, err := dockerRun("--network", networkOf(t, "internal"), "--entrypoint", "/bin/sh", temporalImage(), "-c", script)
	if err != nil {
		t.Fatalf("probe container: %v\n%s", err, out)
	}
	lines := map[string]bool{}
	for _, l := range strings.Split(out, "\n") {
		lines[strings.TrimSpace(l)] = true
	}
	if !lines["OPEN temporal:7233"] {
		t.Fatalf("control: the gateway's 7233 is not reachable from the platform network:\n%s", out)
	}
	for _, a := range targets {
		switch {
		case lines["OPEN "+a]:
			t.Errorf("%s is reachable from the platform network", a)
		case !lines["CLOSED "+a]:
			t.Errorf("%s was not probed:\n%s", a, out)
		}
	}
}

// Even with a route to the internode ports, a platform certificate is refused
// at the internode handshake: internode has its own CA and no Authorizer. The
// server binds them to its loopback (no other container reaches them at all), so
// the probe runs in the server container's own network namespace, the worst
// case: a process that is already on the server host.
func TestCI6_PlatformCertificateRejectedAtInternodeHandshake(t *testing.T) {
	dir, err := filepath.Abs(certDir())
	if err != nil {
		t.Fatal(err)
	}
	server := "container:" + containerOf(t, "temporal-server")
	// probe runs the health check against 127.0.0.1:port as the pair cert/key,
	// trusting the internode CA.
	probe := func(port, cert, key string) (string, error) {
		return dockerRun("--network", server,
			"-v", filepath.Join(dir, "internode-ca.pem")+":/c/ca.pem:ro",
			"-v", filepath.Join(dir, cert)+":/c/client.pem:ro",
			"-v", filepath.Join(dir, key)+":/c/client.key:ro",
			"--entrypoint", "/usr/local/bin/temporal-server", temporalImage(),
			"health", "-addr", "127.0.0.1:"+port, "-server-name", "localhost",
			"-ca", "/c/ca.pem", "-cert", "/c/client.pem", "-key", "/c/client.key", "-timeout", "5s")
	}
	ports := []string{"7234", "7235", "7239"}
	cns := []string{"internode", "nava-worker", "nava-relay", "nava-core", "nava-operator", "temporal-system"}
	// Every probe is its own throwaway container and reads nothing another
	// changes, so all of them run at once (TICK-153 O9): one at a time they
	// took most of the CI-6 suite's time.
	type result struct {
		out string
		err error
	}
	results := make(map[string]result, len(ports)*len(cns))
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, port := range ports {
		for _, cn := range cns {
			wg.Go(func() {
				out, err := probe(port, cn+".pem", cn+".key")
				mu.Lock()
				results[port+"/"+cn] = result{out, err}
				mu.Unlock()
			})
		}
	}
	wg.Wait()

	serving := 0
	for _, port := range ports {
		// Control: the internode certificate is served on a listening port.
		// 7239 (worker service) has no gRPC listener at server v1.29.7.
		r := results[port+"/internode"]
		out, err := r.out, r.err
		listening := err == nil
		if !listening && !strings.Contains(out, "connection refused") {
			t.Fatalf("control: the internode certificate on %s: %v\n%s", port, err, out)
		}
		if listening {
			serving++
		}
		for _, cn := range cns[1:] {
			r := results[port+"/"+cn]
			out, err := r.out, r.err
			switch {
			case err == nil:
				t.Errorf("%s completed the internode handshake on %s: %s", cn, port, out)
			case listening && (strings.Contains(out, "connection refused") || strings.Contains(out, "x509:")):
				// Not the server refusing the client certificate.
				t.Errorf("%s on %s failed before the server judged its certificate: %s", cn, port, out)
			}
		}
	}
	if serving < 2 {
		t.Errorf("only %d internode ports serve the internode certificate; want history (7234) and matching (7235)", serving)
	}
}

// --- API allow-list --------------------------------------------------------

// goldenRow is one unary method of the allow-list golden table.
type goldenRow struct {
	service, method string
	allowed         map[string]bool // identity -> Y
}

// goldenTable reads temporal-server/authz/testdata/allowlist.golden.md, which
// TestAllowListGolden keeps in step with policy.go. Streaming methods are left
// out: none is on any platform row.
func goldenTable(t *testing.T) []goldenRow {
	t.Helper()
	f, err := os.Open(filepath.Join("..", "..", "temporal-server", "authz", "testdata", "allowlist.golden.md"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var rows []goldenRow
	var service string
	var cols []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if svc, ok := strings.CutPrefix(line, "## "); ok {
			if strings.HasPrefix(svc, "D9") {
				break
			}
			service, cols = svc, nil
			continue
		}
		if service == "" || !strings.HasPrefix(line, "|") || strings.HasPrefix(line, "|---") {
			continue
		}
		cells := strings.Split(strings.Trim(line, "|"), "|")
		for i := range cells {
			cells[i] = strings.Trim(strings.TrimSpace(cells[i]), "`")
		}
		if cells[0] == "Method" {
			cols = cells
			continue
		}
		if strings.Contains(cells[0], "(stream)") || cols == nil {
			continue
		}
		row := goldenRow{service: service, method: cells[0], allowed: map[string]bool{}}
		for i := 2; i < len(cells) && i < len(cols); i++ {
			row.allowed[cols[i]] = cells[i] == "Y"
		}
		rows = append(rows, row)
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	if len(rows) < 100 {
		t.Fatalf("golden table: only %d rows parsed", len(rows))
	}
	return rows
}

// sweepBody is the request for a swept method: namespace `nava` where the
// request has a namespace field, else empty. With no namespace, the server's
// namespace interceptor (which runs before the Authorizer) reads it from the task
// token and rejects an empty one with InvalidArgument; the sweep must reach the
// Authorizer. Admin requests are not in this module's registry and go empty:
// none of them carries a task token.
func sweepBody(t *testing.T, row goldenRow) []byte {
	t.Helper()
	d, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(row.service))
	if err != nil {
		return nil
	}
	svc, ok := d.(protoreflect.ServiceDescriptor)
	if !ok {
		return nil
	}
	m := svc.Methods().ByName(protoreflect.Name(row.method))
	if m == nil {
		return nil
	}
	msg := dynamicpb.NewMessage(m.Input())
	if f := m.Input().Fields().ByName("namespace"); f != nil {
		msg.Set(f, protoreflect.ValueOfString(namespace))
	}
	b, err := proto.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// sweepOutsideAllowList calls, as cn, every unary method of the golden table
// that is not on cn's row, with a minimal request (sweepBody), and requires PermissionDenied
// for each. It returns the number of calls made.
func sweepOutsideAllowList(ctx context.Context, t *testing.T, cn string) int {
	t.Helper()
	conn := rawConnAs(t, cn)
	n := 0
	for _, row := range goldenTable(t) {
		if row.allowed[cn] {
			continue
		}
		method := "/" + row.service + "/" + row.method
		wantDenied(t, cn+" "+method, invokeRaw(ctx, conn, method, sweepBody(t, row)))
		n++
	}
	return n
}

// Every relay, core and worker certificate call outside its row of the
// allow-list is denied, for every classified method of the three services.
func TestCI6_RelayCoreWorkerOutsideAllowListDenied(t *testing.T) {
	ctx := ci6Context(t)
	for _, cn := range []string{"nava-relay", "nava-core", "nava-worker"} {
		t.Run(cn, func(t *testing.T) {
			if n := sweepOutsideAllowList(ctx, t, cn); n < 100 {
				t.Errorf("%s: only %d methods swept", cn, n)
			}
		})
	}
	// A real request, not an empty body: tenant B's workflow, reached with
	// each certificate's most useful denied call, is untouched.
	relay, core := dial(t, "nava-relay"), dial(t, "nava-core")
	victimID := uniqueID("B", "victim")
	victim := startOnPlatformQueue(ctx, t, relay, victimID, time.Hour, "victim")
	victimExec := &commonpb.WorkflowExecution{WorkflowId: victimID, RunId: victim.GetRunID()}
	relayWF := workflowservice.NewWorkflowServiceClient(rawConnAs(t, "nava-relay"))
	coreWF := workflowservice.NewWorkflowServiceClient(rawConnAs(t, "nava-core"))
	workerWF := workflowservice.NewWorkflowServiceClient(rawConnAs(t, "nava-worker"))
	_, err := relayWF.TerminateWorkflowExecution(ctx, &workflowservice.TerminateWorkflowExecutionRequest{Namespace: namespace, WorkflowExecution: victimExec})
	wantDenied(t, "nava-relay TerminateWorkflowExecution of tenant B", err)
	_, err = relayWF.DescribeWorkflowExecution(ctx, &workflowservice.DescribeWorkflowExecutionRequest{Namespace: namespace, Execution: victimExec})
	wantDenied(t, "nava-relay DescribeWorkflowExecution of tenant B", err)
	_, err = coreWF.StartWorkflowExecution(ctx, &workflowservice.StartWorkflowExecutionRequest{
		Namespace: namespace, WorkflowId: uniqueID("B", "core-start"), RequestId: fmt.Sprint(time.Now().UnixNano()),
		WorkflowType: &commonpb.WorkflowType{Name: echoType}, TaskQueue: &taskqueuepb.TaskQueue{Name: taskQueue},
	})
	wantDenied(t, "nava-core StartWorkflowExecution", err)
	_, err = coreWF.SignalWorkflowExecution(ctx, &workflowservice.SignalWorkflowExecutionRequest{Namespace: namespace, WorkflowExecution: victimExec, SignalName: "ci6"})
	wantDenied(t, "nava-core SignalWorkflowExecution of tenant B", err)
	_, err = coreWF.ListWorkflowExecutions(ctx, &workflowservice.ListWorkflowExecutionsRequest{Namespace: namespace})
	wantDenied(t, "nava-core ListWorkflowExecutions", err)
	_, err = workerWF.TerminateWorkflowExecution(ctx, &workflowservice.TerminateWorkflowExecutionRequest{Namespace: namespace, WorkflowExecution: victimExec})
	wantDenied(t, "nava-worker TerminateWorkflowExecution of tenant B", err)
	_, err = workerWF.SignalWorkflowExecution(ctx, &workflowservice.SignalWorkflowExecutionRequest{Namespace: namespace, WorkflowExecution: victimExec, SignalName: "ci6"})
	wantDenied(t, "nava-worker SignalWorkflowExecution of tenant B", err)
	_, err = workerWF.ResetStickyTaskQueue(ctx, &workflowservice.ResetStickyTaskQueueRequest{Namespace: namespace, Execution: victimExec})
	wantDenied(t, "nava-worker ResetStickyTaskQueue of tenant B", err)

	desc, err := core.DescribeWorkflowExecution(ctx, victimID, victim.GetRunID())
	if err != nil {
		t.Fatalf("describe victim: %v", err)
	}
	if st := desc.GetWorkflowExecutionInfo().GetStatus(); st != enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING {
		t.Errorf("victim %s status %v, want RUNNING", victimID, st)
	}
	if historyHas(ctx, t, core, victimID, enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_SIGNALED) {
		t.Errorf("victim %s was signaled", victimID)
	}
}

// adminBody encodes an admin request by field number (the admin service's types
// are in go.temporal.io/server, which this module does not import): fields are
// (number, string) or (number, nested bytes) pairs.
func adminBody(fields ...any) []byte {
	var b []byte
	for i := 0; i+1 < len(fields); i += 2 {
		num := protowire.Number(fields[i].(int))
		b = protowire.AppendTag(b, num, protowire.BytesType)
		switch v := fields[i+1].(type) {
		case string:
			b = protowire.AppendString(b, v)
		case []byte:
			b = protowire.AppendBytes(b, v)
		}
	}
	return b
}

// nava-operator is held to its pinned row: ReapplyEvents, ImportWorkflowExecution,
// DeleteWorkflowExecution (both services) and DeleteNamespace are denied, aimed
// at a real tenant-B workflow and the real namespace, and so is every other
// method off its row. Its pinned reads still work (control).
func TestCI6_OperatorLimitedToPinnedList(t *testing.T) {
	ctx := ci6Context(t)
	relay, core, op := dial(t, "nava-relay"), dial(t, "nava-core"), dial(t, "nava-operator")
	victimID := uniqueID("B", "victim")
	victim := startOnPlatformQueue(ctx, t, relay, victimID, time.Hour, "victim")

	ns, err := op.WorkflowService().DescribeNamespace(ctx, &workflowservice.DescribeNamespaceRequest{Namespace: namespace})
	if err != nil {
		t.Fatalf("control: nava-operator DescribeNamespace: %v", err)
	}
	if _, err := op.DescribeWorkflowExecution(ctx, victimID, victim.GetRunID()); err != nil {
		t.Fatalf("control: nava-operator DescribeWorkflowExecution: %v", err)
	}

	conn := rawConnAs(t, "nava-operator")
	execution := adminBody(1, victimID, 2, victim.GetRunID())
	const adm = "/temporal.server.api.adminservice.v1.AdminService/"
	labels := map[string]string{"identity": "nava-operator", "reason": "api_not_allowed"}
	before := deniedCount(t, labels)
	mark := logMark()
	// The anchor: a deny only this run can cause (an allowed read aimed at a
	// namespace that is not nava), its unique namespace logged.
	runNamespace := fmt.Sprintf("ci6-run-%d", time.Now().UnixNano())
	_, err = workflowservice.NewWorkflowServiceClient(conn).DescribeNamespace(ctx, &workflowservice.DescribeNamespaceRequest{Namespace: runNamespace})
	wantDenied(t, "nava-operator DescribeNamespace (anchor)", err)
	pinned := map[string][]byte{
		adm + "ReapplyEvents":           adminBody(4, ns.GetNamespaceInfo().GetId(), 2, execution),
		adm + "ImportWorkflowExecution": adminBody(1, namespace, 2, execution),
		adm + "DeleteWorkflowExecution": adminBody(1, namespace, 2, execution),
	}
	for method, body := range pinned {
		wantDenied(t, "nava-operator "+method, invokeRaw(ctx, conn, method, body))
	}
	wf := workflowservice.NewWorkflowServiceClient(conn)
	_, err = wf.DeleteWorkflowExecution(ctx, &workflowservice.DeleteWorkflowExecutionRequest{
		Namespace: namespace, WorkflowExecution: &commonpb.WorkflowExecution{WorkflowId: victimID, RunId: victim.GetRunID()},
	})
	wantDenied(t, "nava-operator WorkflowService/DeleteWorkflowExecution", err)
	_, err = operatorservice.NewOperatorServiceClient(conn).DeleteNamespace(ctx, &operatorservice.DeleteNamespaceRequest{Namespace: namespace})
	wantDenied(t, "nava-operator OperatorService/DeleteNamespace", err)
	var wants []map[string]string
	for _, api := range []string{adm + "ReapplyEvents", adm + "ImportWorkflowExecution", adm + "DeleteWorkflowExecution",
		wfAPI + "DeleteWorkflowExecution", "/temporal.api.operatorservice.v1.OperatorService/DeleteNamespace"} {
		wants = append(wants, map[string]string{"identity": "nava-operator", "api": api, "reason": "api_not_allowed",
			"cert_subject": "~CN=nava-operator"})
	}
	awaitDenyLogsAfter(t, mark, map[string]string{"identity": "nava-operator", "reason": "namespace_not_allowed",
		"api": wfAPI + "DescribeNamespace", "namespace": runNamespace}, wants)
	awaitDeniedCount(t, labels, before, len(wants))

	if n := sweepOutsideAllowList(ctx, t, "nava-operator"); n < 50 {
		t.Errorf("nava-operator: only %d methods swept", n)
	}

	desc, err := core.DescribeWorkflowExecution(ctx, victimID, victim.GetRunID())
	if err != nil {
		t.Fatalf("victim %s is gone: %v", victimID, err)
	}
	if st := desc.GetWorkflowExecutionInfo().GetStatus(); st != enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING {
		t.Errorf("victim %s status %v, want RUNNING", victimID, st)
	}
	if _, err := op.WorkflowService().DescribeNamespace(ctx, &workflowservice.DescribeNamespaceRequest{Namespace: namespace}); err != nil {
		t.Errorf("namespace %s is gone: %v", namespace, err)
	}
}

// --- worker commands (erratum E1) --------------------------------------------

func echoTypeOf() *commonpb.WorkflowType { return &commonpb.WorkflowType{Name: echoType} }
func platformQueue() *taskqueuepb.TaskQueue {
	return &taskqueuepb.TaskQueue{Name: taskQueue, Kind: enumspb.TASK_QUEUE_KIND_NORMAL}
}

// A tenant-A workflow task that starts a child in tenant B is denied, alerts,
// and no child exists.
func TestCI6_WorkerCrossTenantChildDeniedWithAlert(t *testing.T) {
	ctx := ci6Context(t)
	childID := uniqueID("B", "child")
	attackID := expectCommandDeny(ctx, t, "command_cross_tenant", "StartChildWorkflowExecution", func(string) []*commandpb.Command {
		return []*commandpb.Command{{
			CommandType: enumspb.COMMAND_TYPE_START_CHILD_WORKFLOW_EXECUTION,
			Attributes: &commandpb.Command_StartChildWorkflowExecutionCommandAttributes{StartChildWorkflowExecutionCommandAttributes: &commandpb.StartChildWorkflowExecutionCommandAttributes{
				WorkflowId: childID, WorkflowType: echoTypeOf(), TaskQueue: platformQueue(),
				Header: tenantHeader(t, "A"),
			}},
		}}
	})
	system := dial(t, "temporal-system")
	var notFound *serviceerror.NotFound
	if _, err := system.DescribeWorkflowExecution(ctx, childID, ""); !errors.As(err, &notFound) {
		t.Errorf("tenant-B child %s: got %v, want NotFound", childID, err)
	}
	if historyHas(ctx, t, system, attackID, enumspb.EVENT_TYPE_START_CHILD_WORKFLOW_EXECUTION_INITIATED) {
		t.Errorf("attacker %s recorded a child start", attackID)
	}
}

// A tenant-A workflow task that signals a tenant-B workflow is denied, alerts,
// and the victim receives nothing.
func TestCI6_WorkerCrossTenantSignalExternalDeniedWithAlert(t *testing.T) {
	ctx := ci6Context(t)
	relay, core := dial(t, "nava-relay"), dial(t, "nava-core")
	victimID := uniqueID("B", "victim")
	victim := startOnPlatformQueue(ctx, t, relay, victimID, time.Hour, "victim")
	attackID := expectCommandDeny(ctx, t, "command_cross_tenant", "SignalExternalWorkflowExecution", func(string) []*commandpb.Command {
		return []*commandpb.Command{{
			CommandType: enumspb.COMMAND_TYPE_SIGNAL_EXTERNAL_WORKFLOW_EXECUTION,
			Attributes: &commandpb.Command_SignalExternalWorkflowExecutionCommandAttributes{SignalExternalWorkflowExecutionCommandAttributes: &commandpb.SignalExternalWorkflowExecutionCommandAttributes{
				SignalName: "ci6-steal",
				Execution:  &commonpb.WorkflowExecution{WorkflowId: victimID, RunId: victim.GetRunID()},
			}},
		}}
	})
	if historyHas(ctx, t, core, victimID, enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_SIGNALED) {
		t.Errorf("victim %s was signaled", victimID)
	}
	if historyHas(ctx, t, core, attackID, enumspb.EVENT_TYPE_SIGNAL_EXTERNAL_WORKFLOW_EXECUTION_INITIATED) {
		t.Errorf("attacker %s recorded a signal-external", attackID)
	}
}

// A tenant-A workflow task that cancels a tenant-B workflow is denied, alerts,
// and the victim keeps running with no cancel request.
func TestCI6_WorkerCrossTenantCancelExternalDeniedWithAlert(t *testing.T) {
	ctx := ci6Context(t)
	relay, core := dial(t, "nava-relay"), dial(t, "nava-core")
	victimID := uniqueID("B", "victim")
	victim := startOnPlatformQueue(ctx, t, relay, victimID, time.Hour, "victim")
	attackID := expectCommandDeny(ctx, t, "command_cross_tenant", "RequestCancelExternalWorkflowExecution", func(string) []*commandpb.Command {
		return []*commandpb.Command{{
			CommandType: enumspb.COMMAND_TYPE_REQUEST_CANCEL_EXTERNAL_WORKFLOW_EXECUTION,
			Attributes: &commandpb.Command_RequestCancelExternalWorkflowExecutionCommandAttributes{RequestCancelExternalWorkflowExecutionCommandAttributes: &commandpb.RequestCancelExternalWorkflowExecutionCommandAttributes{
				WorkflowId: victimID, RunId: victim.GetRunID(),
			}},
		}}
	})
	desc, err := core.DescribeWorkflowExecution(ctx, victimID, victim.GetRunID())
	if err != nil {
		t.Fatalf("describe victim: %v", err)
	}
	if st := desc.GetWorkflowExecutionInfo().GetStatus(); st != enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING {
		t.Errorf("victim %s status %v, want RUNNING", victimID, st)
	}
	if historyHas(ctx, t, core, victimID, enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_CANCEL_REQUESTED) {
		t.Errorf("victim %s got a cancel request", victimID)
	}
	if historyHas(ctx, t, core, attackID, enumspb.EVENT_TYPE_REQUEST_CANCEL_EXTERNAL_WORKFLOW_EXECUTION_INITIATED) {
		t.Errorf("attacker %s recorded a cancel-external", attackID)
	}
}

// continueAsNew is a continue-as-new command onto the platform workload.
func continueAsNew(header *commonpb.Header, cron string) []*commandpb.Command {
	input, _ := converter.GetDefaultDataConverter().ToPayloads("ci6")
	return []*commandpb.Command{{
		CommandType: enumspb.COMMAND_TYPE_CONTINUE_AS_NEW_WORKFLOW_EXECUTION,
		Attributes: &commandpb.Command_ContinueAsNewWorkflowExecutionCommandAttributes{ContinueAsNewWorkflowExecutionCommandAttributes: &commandpb.ContinueAsNewWorkflowExecutionCommandAttributes{
			WorkflowType: echoTypeOf(), TaskQueue: platformQueue(), Input: input, Header: header, CronSchedule: cron,
		}},
	}}
}

// assertNotContinued fails if the attacking workflow continued as new.
func assertNotContinued(ctx context.Context, t *testing.T, id string) {
	t.Helper()
	if historyHas(ctx, t, dial(t, "temporal-system"), id, enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_CONTINUED_AS_NEW) {
		t.Errorf("%s continued as new", id)
	}
}

// Erratum E1: continue-as-new without the nava-tenant header is denied.
func TestCI6_WorkerContinueAsNewMissingTenantHeaderDenied(t *testing.T) {
	ctx := ci6Context(t)
	id := expectCommandDeny(ctx, t, "continue_as_new_header", "ContinueAsNewWorkflowExecution", func(string) []*commandpb.Command {
		return continueAsNew(nil, "")
	})
	assertNotContinued(ctx, t, id)
}

// Erratum E1: continue-as-new whose nava-tenant header names another tenant is denied.
func TestCI6_WorkerContinueAsNewMismatchedTenantHeaderDenied(t *testing.T) {
	ctx := ci6Context(t)
	id := expectCommandDeny(ctx, t, "continue_as_new_header", "ContinueAsNewWorkflowExecution", func(string) []*commandpb.Command {
		return continueAsNew(tenantHeader(t, "B"), "")
	})
	assertNotContinued(ctx, t, id)
}

// Erratum E1: continue-as-new with a cron_schedule is denied, even with the
// right header.
func TestCI6_WorkerContinueAsNewCronScheduleDenied(t *testing.T) {
	ctx := ci6Context(t)
	id := expectCommandDeny(ctx, t, "continue_as_new_cron", "ContinueAsNewWorkflowExecution", func(string) []*commandpb.Command {
		return continueAsNew(tenantHeader(t, "A"), "* * * * *")
	})
	assertNotContinued(ctx, t, id)
}

// Erratum E1: a command type off the written allow-list is denied: both Nexus
// commands, and a type no API version has yet.
func TestCI6_WorkerUnlistedCommandTypeDenied(t *testing.T) {
	cases := []struct {
		name, command string
		cmd           *commandpb.Command
	}{
		{"ScheduleNexusOperation", "ScheduleNexusOperation", &commandpb.Command{
			CommandType: enumspb.COMMAND_TYPE_SCHEDULE_NEXUS_OPERATION,
			Attributes: &commandpb.Command_ScheduleNexusOperationCommandAttributes{ScheduleNexusOperationCommandAttributes: &commandpb.ScheduleNexusOperationCommandAttributes{
				Endpoint: "ci6", Service: "ci6", Operation: "steal",
			}},
		}},
		{"RequestCancelNexusOperation", "RequestCancelNexusOperation", &commandpb.Command{
			CommandType: enumspb.COMMAND_TYPE_REQUEST_CANCEL_NEXUS_OPERATION,
			Attributes: &commandpb.Command_RequestCancelNexusOperationCommandAttributes{RequestCancelNexusOperationCommandAttributes: &commandpb.RequestCancelNexusOperationCommandAttributes{
				ScheduledEventId: 1,
			}},
		}},
		{"future command type", "unknown", &commandpb.Command{CommandType: enumspb.CommandType(9999)}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctx := ci6Context(t)
			expectCommandDeny(ctx, t, "command_not_allowed", c.command, func(string) []*commandpb.Command {
				return []*commandpb.Command{c.cmd}
			})
		})
	}
}

// Erratum E1: a task token that does not decode is denied and alerts, before
// any command is looked at. The namespace is set so the server's namespace
// interceptor does not reject the call first: the deny must be the Authorizer's.
func TestCI6_WorkerUndecodableTaskTokenDenied(t *testing.T) {
	ctx := ci6Context(t)
	worker := workflowservice.NewWorkflowServiceClient(rawConnAs(t, "nava-worker"))
	labels := map[string]string{"identity": "nava-worker", "reason": "task_token_invalid", "command": "none"}
	before := deniedCount(t, labels)
	mark := logMark()
	_, err := worker.RespondWorkflowTaskCompleted(ctx, &workflowservice.RespondWorkflowTaskCompletedRequest{
		Namespace: namespace, TaskToken: []byte("\xff\xffci6-not-a-task-token"), Identity: "ci6-attacker",
		Commands: []*commandpb.Command{{CommandType: enumspb.COMMAND_TYPE_COMPLETE_WORKFLOW_EXECUTION}},
	})
	wantDenied(t, "nava-worker RespondWorkflowTaskCompleted with an undecodable token", err)
	awaitDenyLog(t, mark, map[string]string{
		"level": "error", "alert": "true", "identity": "nava-worker", "reason": "task_token_invalid",
		"command": "none", "cert_subject": "~CN=nava-worker",
	})
	awaitDeniedCount(t, labels, before, 1)
}

// Erratum E1: a task whose workflow id has no t:<tenant>: prefix is denied as a
// whole (the server's own identity can start one; no platform identity can).
func TestCI6_WorkerTaskWithoutTenantPrefixDenied(t *testing.T) {
	ctx := ci6Context(t)
	id := fmt.Sprintf("ci6-noprefix-%d", time.Now().UnixNano())
	startOnPlatformQueue(ctx, t, dial(t, "temporal-system"), id, 0, "ci6")
	respondExpectDeny(ctx, t, id, "workflow_id_no_tenant", "none",
		[]*commandpb.Command{{CommandType: enumspb.COMMAND_TYPE_COMPLETE_WORKFLOW_EXECUTION}})
}

// --- allowed ------------------------------------------------------------------

// tenantPropagator is the worker's nava-tenant header: the tenant of the
// workflow's own id, on every child it starts.
type tenantPropagator struct{}

func (tenantPropagator) Inject(context.Context, workflow.HeaderWriter) error { return nil }
func (tenantPropagator) Extract(ctx context.Context, _ workflow.HeaderReader) (context.Context, error) {
	return ctx, nil
}
func (tenantPropagator) ExtractToWorkflow(ctx workflow.Context, _ workflow.HeaderReader) (workflow.Context, error) {
	return ctx, nil
}
func (tenantPropagator) InjectFromWorkflow(ctx workflow.Context, w workflow.HeaderWriter) error {
	id := workflow.GetInfo(ctx).WorkflowExecution.ID
	rest, ok := strings.CutPrefix(id, "t:")
	if !ok {
		return fmt.Errorf("workflow id %q has no tenant prefix", id)
	}
	tenant, _, _ := strings.Cut(rest, ":")
	p, err := converter.GetDefaultDataConverter().ToPayload(tenant)
	if err != nil {
		return err
	}
	w.Set("nava-tenant", p)
	return nil
}

// parentOrEcho is the smoke workload with one branch: input "parent" starts a
// same-tenant child (its own id plus "-child") and returns its result.
func parentOrEcho(ctx workflow.Context, in string) (string, error) {
	if in != "parent" {
		return "echo:" + in, nil
	}
	cctx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
		WorkflowID: workflow.GetInfo(ctx).WorkflowExecution.ID + "-child", TaskQueue: taskQueue,
	})
	var out string
	if err := workflow.ExecuteChildWorkflow(cctx, echoType, "child").Get(cctx, &out); err != nil {
		return "", err
	}
	return "parent:" + out, nil
}

// A same-tenant child completes: a real SDK worker with the nava-worker
// certificate runs a parent that starts a child in its own tenant, through the
// same command inspection that denies the cross-tenant cases above.
func TestCI6_Allowed_SameTenantChildCompletes(t *testing.T) {
	ctx := ci6Context(t)
	wc := dial(t, "nava-worker", func(o *client.Options) {
		o.ContextPropagators = []workflow.ContextPropagator{tenantPropagator{}}
	})
	w := worker.New(wc, taskQueue, worker.Options{})
	w.RegisterWorkflowWithOptions(parentOrEcho, workflow.RegisterOptions{Name: echoType})
	if err := w.Start(); err != nil {
		t.Fatalf("worker start: %v", err)
	}
	t.Cleanup(w.Stop)

	relay, core := dial(t, "nava-relay"), dial(t, "nava-core")
	parentID := uniqueID("A", "parent")
	run := startOnPlatformQueue(ctx, t, relay, parentID, 0, "parent")
	var out string
	if err := core.GetWorkflow(ctx, parentID, run.GetRunID()).Get(ctx, &out); err != nil || out != "parent:echo:child" {
		t.Fatalf("parent %s: result %q, %v; want parent:echo:child", parentID, out, err)
	}
	desc, err := core.DescribeWorkflowExecution(ctx, parentID+"-child", "")
	if err != nil {
		t.Fatalf("describe child: %v", err)
	}
	if st := desc.GetWorkflowExecutionInfo().GetStatus(); st != enumspb.WORKFLOW_EXECUTION_STATUS_COMPLETED {
		t.Errorf("child status %v, want COMPLETED", st)
	}
}

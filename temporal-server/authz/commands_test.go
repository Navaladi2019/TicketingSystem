package authz

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"strings"
	"testing"

	commandpb "go.temporal.io/api/command/v1"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/workflowservice/v1"
	tokenspb "go.temporal.io/server/api/token/v1"
	"go.temporal.io/server/common/authorization"
	"go.temporal.io/server/common/log"
	"go.temporal.io/server/common/metrics"
	"go.temporal.io/server/common/metrics/metricstest"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
)

const (
	wfA = "t:tenantA:ticket:42"
	wfB = "t:tenantB:ticket:7"
)

func taskToken(t testing.TB, workflowID string) []byte {
	t.Helper()
	b, err := (&tokenspb.Task{NamespaceId: "ns-id", WorkflowId: workflowID, RunId: "run"}).Marshal()
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func respond(token []byte, cmds ...*commandpb.Command) *workflowservice.RespondWorkflowTaskCompletedRequest {
	return &workflowservice.RespondWorkflowTaskCompletedRequest{Namespace: Namespace, TaskToken: token, Commands: cmds}
}

func jsonString(s string) *commonpb.Payload {
	return &commonpb.Payload{Metadata: map[string][]byte{"encoding": []byte("json/plain")}, Data: []byte(`"` + s + `"`)}
}

func startChild(namespace, workflowID string) *commandpb.Command {
	return &commandpb.Command{
		CommandType: enumspb.COMMAND_TYPE_START_CHILD_WORKFLOW_EXECUTION,
		Attributes: &commandpb.Command_StartChildWorkflowExecutionCommandAttributes{
			StartChildWorkflowExecutionCommandAttributes: &commandpb.StartChildWorkflowExecutionCommandAttributes{
				Namespace: namespace, WorkflowId: workflowID}},
	}
}

func signalExternal(namespace, workflowID string) *commandpb.Command {
	return &commandpb.Command{
		CommandType: enumspb.COMMAND_TYPE_SIGNAL_EXTERNAL_WORKFLOW_EXECUTION,
		Attributes: &commandpb.Command_SignalExternalWorkflowExecutionCommandAttributes{
			SignalExternalWorkflowExecutionCommandAttributes: &commandpb.SignalExternalWorkflowExecutionCommandAttributes{
				Namespace: namespace, Execution: &commonpb.WorkflowExecution{WorkflowId: workflowID}, SignalName: "s"}},
	}
}

func cancelExternal(namespace, workflowID string) *commandpb.Command {
	return &commandpb.Command{
		CommandType: enumspb.COMMAND_TYPE_REQUEST_CANCEL_EXTERNAL_WORKFLOW_EXECUTION,
		Attributes: &commandpb.Command_RequestCancelExternalWorkflowExecutionCommandAttributes{
			RequestCancelExternalWorkflowExecutionCommandAttributes: &commandpb.RequestCancelExternalWorkflowExecutionCommandAttributes{
				Namespace: namespace, WorkflowId: workflowID}},
	}
}

// continueAsNew sets the tenant header to header, or leaves it out if header is nil.
func continueAsNew(header *commonpb.Payload, cron string) *commandpb.Command {
	attrs := &commandpb.ContinueAsNewWorkflowExecutionCommandAttributes{CronSchedule: cron}
	if header != nil {
		attrs.Header = &commonpb.Header{Fields: map[string]*commonpb.Payload{TenantHeader: header}}
	}
	return &commandpb.Command{
		CommandType: enumspb.COMMAND_TYPE_CONTINUE_AS_NEW_WORKFLOW_EXECUTION,
		Attributes:  &commandpb.Command_ContinueAsNewWorkflowExecutionCommandAttributes{ContinueAsNewWorkflowExecutionCommandAttributes: attrs},
	}
}

func upsertTenant(p *commonpb.Payload) *commandpb.Command {
	fields := map[string]*commonpb.Payload{"TicketStatus": jsonString("open")}
	if p != nil {
		fields[TenantSearchAttribute] = p
	}
	return &commandpb.Command{
		CommandType: enumspb.COMMAND_TYPE_UPSERT_WORKFLOW_SEARCH_ATTRIBUTES,
		Attributes: &commandpb.Command_UpsertWorkflowSearchAttributesCommandAttributes{
			UpsertWorkflowSearchAttributesCommandAttributes: &commandpb.UpsertWorkflowSearchAttributesCommandAttributes{
				SearchAttributes: &commonpb.SearchAttributes{IndexedFields: fields}}},
	}
}

func ofType(t enumspb.CommandType) *commandpb.Command { return &commandpb.Command{CommandType: t} }

// workerCtx is a nava-worker TLS peer with a known certificate.
func workerCtx() context.Context {
	cert := &x509.Certificate{
		Subject:      pkix.Name{CommonName: "nava-worker"},
		Issuer:       pkix.Name{CommonName: "nava-internal-ca"},
		SerialNumber: big.NewInt(77),
	}
	return peer.NewContext(context.Background(), &peer.Peer{
		Addr:     &net.TCPAddr{IP: net.IPv4(10, 0, 0, 2), Port: 4000},
		AuthInfo: credentials.TLSInfo{State: tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{{cert}}}},
	})
}

type denyCase struct {
	name       string
	req        any
	reason     string
	command    string
	workflowID string // expected log workflow_id
	target     string // expected log target_workflow_id, "" if none
	commandIdx string
}

// authorizeWorker runs one RespondWorkflowTaskCompleted through a fresh
// Authorizer and returns the decision, metric recordings and log entries.
func authorizeWorker(t *testing.T, req any) (authorization.Decision, []*metricstest.CapturedRecording, []map[string]string) {
	t.Helper()
	capture := metricstest.NewCaptureHandler()
	rec := capture.StartCapture()
	defer capture.StopCapture(rec)
	logger := &captureLogger{}
	a := NewAuthorizer(logger, capture)
	res, err := a.Authorize(workerCtx(), claimsFor(Worker), &authorization.CallTarget{
		APIName: respondWorkflowTaskCompleted, Namespace: Namespace, Request: req,
	})
	if err != nil {
		t.Fatalf("Authorize returned error %v", err)
	}
	if res.Reason != "" {
		t.Errorf("deny reason must not reach the caller, got %q", res.Reason)
	}
	return res.Decision, rec.Snapshot()[DeniedMetric], logger.entries
}

// requireCommandDeny asserts the deny, its metric (bounded tags only) and its
// alerting log line.
func requireCommandDeny(t *testing.T, c denyCase) {
	t.Helper()
	decision, recs, logs := authorizeWorker(t, c.req)
	if decision != authorization.DecisionDeny {
		t.Fatalf("decision = %v, want deny", decision)
	}
	if len(recs) != 1 {
		t.Fatalf("want 1 %s recording, got %d", DeniedMetric, len(recs))
	}
	wantTags := map[string]string{
		"identity": "nava-worker", "api": respondWorkflowTaskCompleted, "namespace": Namespace,
		"reason": c.reason, "command": c.command,
	}
	if len(recs[0].Tags) != len(wantTags) {
		t.Errorf("metric tags = %v, want exactly %v", recs[0].Tags, wantTags)
	}
	for k, v := range wantTags {
		if recs[0].Tags[k] != v {
			t.Errorf("metric tag %s = %q, want %q", k, recs[0].Tags[k], v)
		}
	}
	for k, v := range recs[0].Tags {
		if strings.Contains(v, "tenantA") || strings.Contains(v, "tenantB") || strings.HasPrefix(v, "t:") {
			t.Errorf("metric tag %s = %q leaks a workflow id or tenant", k, v)
		}
	}

	if len(logs) != 1 {
		t.Fatalf("want 1 log entry, got %d", len(logs))
	}
	wantLog := map[string]string{
		"level": "error", "alert": "true", "identity": "nava-worker", "api": respondWorkflowTaskCompleted,
		"reason": c.reason, "command": c.command, "workflow_id": c.workflowID,
		"command_index": c.commandIdx, "target_workflow_id": c.target,
		"cert_subject": "CN=nava-worker", "cert_serial": "77", "cert_issuer": "CN=nava-internal-ca",
	}
	for k, v := range wantLog {
		if logs[0][k] != v {
			t.Errorf("log field %s = %q, want %q", k, logs[0][k], v)
		}
	}
}

func requireCommandAllow(t *testing.T, req any) {
	t.Helper()
	decision, recs, logs := authorizeWorker(t, req)
	if decision != authorization.DecisionAllow {
		t.Fatalf("decision = %v, want allow (log %v)", decision, logs)
	}
	if len(recs) != 0 || len(logs) != 0 {
		t.Errorf("allowed response recorded a deny: metrics %d, logs %v", len(recs), logs)
	}
}

func TestCommandTargetsOtherTenantDenied(t *testing.T) {
	tok := taskToken(t, wfA)
	for _, c := range []denyCase{
		{name: "child other tenant", req: respond(tok, startChild(Namespace, wfB)),
			reason: ReasonCrossTenant, command: "StartChildWorkflowExecution", workflowID: wfA, target: wfB, commandIdx: "0"},
		{name: "child no prefix", req: respond(tok, startChild(Namespace, "ticket:42")),
			reason: ReasonCrossTenant, command: "StartChildWorkflowExecution", workflowID: wfA, target: "ticket:42", commandIdx: "0"},
		{name: "child empty tenant", req: respond(tok, startChild("", "t::x")),
			reason: ReasonCrossTenant, command: "StartChildWorkflowExecution", workflowID: wfA, target: "t::x", commandIdx: "0"},
		{name: "child tenant prefix lookalike", req: respond(tok, startChild(Namespace, "t:tenantAB:x")),
			reason: ReasonCrossTenant, command: "StartChildWorkflowExecution", workflowID: wfA, target: "t:tenantAB:x", commandIdx: "0"},
		{name: "child missing attributes", req: respond(tok, ofType(enumspb.COMMAND_TYPE_START_CHILD_WORKFLOW_EXECUTION)),
			reason: ReasonCrossTenant, command: "StartChildWorkflowExecution", workflowID: wfA, commandIdx: "0"},
		{name: "signal external other tenant", req: respond(tok, signalExternal(Namespace, wfB)),
			reason: ReasonCrossTenant, command: "SignalExternalWorkflowExecution", workflowID: wfA, target: wfB, commandIdx: "0"},
		{name: "cancel external other tenant", req: respond(tok, cancelExternal("", wfB)),
			reason: ReasonCrossTenant, command: "RequestCancelExternalWorkflowExecution", workflowID: wfA, target: wfB, commandIdx: "0"},
		// One bad command among good ones denies the whole response.
		{name: "second command bad", req: respond(tok, ofType(enumspb.COMMAND_TYPE_START_TIMER), signalExternal(Namespace, wfB)),
			reason: ReasonCrossTenant, command: "SignalExternalWorkflowExecution", workflowID: wfA, target: wfB, commandIdx: "1"},
	} {
		t.Run(c.name, func(t *testing.T) { requireCommandDeny(t, c) })
	}
}

func TestCommandWrongNamespaceDenied(t *testing.T) {
	tok := taskToken(t, wfA)
	sameTenant := "t:tenantA:other"
	for _, c := range []denyCase{
		{name: "child", req: respond(tok, startChild("temporal-system", sameTenant)),
			reason: ReasonCommandNamespace, command: "StartChildWorkflowExecution", workflowID: wfA, target: sameTenant, commandIdx: "0"},
		{name: "signal", req: respond(tok, signalExternal("default", sameTenant)),
			reason: ReasonCommandNamespace, command: "SignalExternalWorkflowExecution", workflowID: wfA, target: sameTenant, commandIdx: "0"},
		{name: "cancel", req: respond(tok, cancelExternal("NAVA", sameTenant)),
			reason: ReasonCommandNamespace, command: "RequestCancelExternalWorkflowExecution", workflowID: wfA, target: sameTenant, commandIdx: "0"},
	} {
		t.Run(c.name, func(t *testing.T) { requireCommandDeny(t, c) })
	}
}

func TestCommandSameTenantTargetsAllowed(t *testing.T) {
	tok := taskToken(t, wfA)
	requireCommandAllow(t, respond(tok, startChild(Namespace, "t:tenantA:child:1")))
	// An empty namespace is the task's own namespace, which the API gate pins to nava.
	requireCommandAllow(t, respond(tok, startChild("", "t:tenantA:child:2")))
	requireCommandAllow(t, respond(tok, signalExternal(Namespace, "t:tenantA:ticket:9")))
	requireCommandAllow(t, respond(tok, cancelExternal(Namespace, "t:tenantA:ticket:9")))
	requireCommandAllow(t, respond(tok))
}

func TestContinueAsNew(t *testing.T) {
	tok := taskToken(t, wfA)
	can := "ContinueAsNewWorkflowExecution"
	for _, c := range []denyCase{
		{name: "header missing", req: respond(tok, continueAsNew(nil, "")),
			reason: ReasonTenantHeader, command: can, workflowID: wfA, commandIdx: "0"},
		{name: "header other tenant", req: respond(tok, continueAsNew(jsonString("tenantB"), "")),
			reason: ReasonTenantHeader, command: can, workflowID: wfA, commandIdx: "0"},
		{name: "header empty", req: respond(tok, continueAsNew(jsonString(""), "")),
			reason: ReasonTenantHeader, command: can, workflowID: wfA, commandIdx: "0"},
		{name: "header not json/plain", req: respond(tok, continueAsNew(&commonpb.Payload{
			Metadata: map[string][]byte{"encoding": []byte("binary/plain")}, Data: []byte("tenantA")}, "")),
			reason: ReasonTenantHeader, command: can, workflowID: wfA, commandIdx: "0"},
		{name: "header not a string", req: respond(tok, continueAsNew(&commonpb.Payload{
			Metadata: map[string][]byte{"encoding": []byte("json/plain")}, Data: []byte(`["tenantA"]`)}, "")),
			reason: ReasonTenantHeader, command: can, workflowID: wfA, commandIdx: "0"},
		{name: "cron schedule", req: respond(tok, continueAsNew(jsonString("tenantA"), "* * * * *")),
			reason: ReasonCronSchedule, command: can, workflowID: wfA, commandIdx: "0"},
	} {
		t.Run(c.name, func(t *testing.T) { requireCommandDeny(t, c) })
	}
	t.Run("same tenant allowed", func(t *testing.T) {
		requireCommandAllow(t, respond(tok, continueAsNew(jsonString("tenantA"), "")))
	})
}

func TestUndecodableTaskTokenOrUnprefixedIDDenied(t *testing.T) {
	for _, c := range []denyCase{
		{name: "garbage token", req: respond([]byte{0xff, 0xff, 0xff}),
			reason: ReasonTaskToken, command: commandNone, commandIdx: "-1"},
		{name: "empty token", req: respond(nil),
			reason: ReasonNoTenantPrefix, command: commandNone, commandIdx: "-1"},
		{name: "no tenant prefix", req: respond(taskToken(t, "ticket:42"), startChild(Namespace, "ticket:43")),
			reason: ReasonNoTenantPrefix, command: commandNone, workflowID: "ticket:42", commandIdx: "-1"},
		{name: "prefix without closing colon", req: respond(taskToken(t, "t:tenantA")),
			reason: ReasonNoTenantPrefix, command: commandNone, workflowID: "t:tenantA", commandIdx: "-1"},
		{name: "not a RespondWorkflowTaskCompleted request", req: &workflowservice.RespondWorkflowTaskFailedRequest{},
			reason: ReasonMalformedRequest, command: commandNone, commandIdx: "-1"},
		{name: "nil request", req: nil,
			reason: ReasonMalformedRequest, command: commandNone, commandIdx: "-1"},
	} {
		t.Run(c.name, func(t *testing.T) { requireCommandDeny(t, c) })
	}
}

func TestUpsertTenantSearchAttribute(t *testing.T) {
	tok := taskToken(t, wfA)
	up := "UpsertWorkflowSearchAttributes"
	for _, c := range []denyCase{
		{name: "other tenant", req: respond(tok, upsertTenant(jsonString("tenantB"))),
			reason: ReasonTenantSearchAttr, command: up, workflowID: wfA, commandIdx: "0"},
		{name: "keyword list", req: respond(tok, upsertTenant(&commonpb.Payload{
			Metadata: map[string][]byte{"encoding": []byte("json/plain")}, Data: []byte(`["tenantA","tenantB"]`)})),
			reason: ReasonTenantSearchAttr, command: up, workflowID: wfA, commandIdx: "0"},
	} {
		t.Run(c.name, func(t *testing.T) { requireCommandDeny(t, c) })
	}
	t.Run("own tenant allowed", func(t *testing.T) { requireCommandAllow(t, respond(tok, upsertTenant(jsonString("tenantA")))) })
	t.Run("not set allowed", func(t *testing.T) { requireCommandAllow(t, respond(tok, upsertTenant(nil))) })
}

// Every listed type is allowed on its same-tenant path; everything else,
// including the Nexus commands and an enum value from a later API version, is
// denied with reason command_not_allowed.
func TestCommandTypeTable(t *testing.T) {
	tok := taskToken(t, wfA)
	sameTenantPath := map[enumspb.CommandType]*commandpb.Command{
		enumspb.COMMAND_TYPE_START_CHILD_WORKFLOW_EXECUTION:             startChild(Namespace, "t:tenantA:c"),
		enumspb.COMMAND_TYPE_SIGNAL_EXTERNAL_WORKFLOW_EXECUTION:         signalExternal(Namespace, "t:tenantA:c"),
		enumspb.COMMAND_TYPE_REQUEST_CANCEL_EXTERNAL_WORKFLOW_EXECUTION: cancelExternal(Namespace, "t:tenantA:c"),
		enumspb.COMMAND_TYPE_CONTINUE_AS_NEW_WORKFLOW_EXECUTION:         continueAsNew(jsonString("tenantA"), ""),
		enumspb.COMMAND_TYPE_UPSERT_WORKFLOW_SEARCH_ATTRIBUTES:          upsertTenant(jsonString("tenantA")),
	}
	wantAllowed := []enumspb.CommandType{
		enumspb.COMMAND_TYPE_SCHEDULE_ACTIVITY_TASK,
		enumspb.COMMAND_TYPE_REQUEST_CANCEL_ACTIVITY_TASK,
		enumspb.COMMAND_TYPE_START_TIMER,
		enumspb.COMMAND_TYPE_CANCEL_TIMER,
		enumspb.COMMAND_TYPE_RECORD_MARKER,
		enumspb.COMMAND_TYPE_COMPLETE_WORKFLOW_EXECUTION,
		enumspb.COMMAND_TYPE_FAIL_WORKFLOW_EXECUTION,
		enumspb.COMMAND_TYPE_CANCEL_WORKFLOW_EXECUTION,
		enumspb.COMMAND_TYPE_CONTINUE_AS_NEW_WORKFLOW_EXECUTION,
		enumspb.COMMAND_TYPE_START_CHILD_WORKFLOW_EXECUTION,
		enumspb.COMMAND_TYPE_SIGNAL_EXTERNAL_WORKFLOW_EXECUTION,
		enumspb.COMMAND_TYPE_REQUEST_CANCEL_EXTERNAL_WORKFLOW_EXECUTION,
		enumspb.COMMAND_TYPE_UPSERT_WORKFLOW_SEARCH_ATTRIBUTES,
		enumspb.COMMAND_TYPE_MODIFY_WORKFLOW_PROPERTIES,
		enumspb.COMMAND_TYPE_PROTOCOL_MESSAGE,
	}
	if len(allowedCommands) != len(wantAllowed) {
		t.Errorf("allowedCommands has %d types, the E1 list has %d", len(allowedCommands), len(wantAllowed))
	}
	for _, ct := range wantAllowed {
		cmd, ok := sameTenantPath[ct]
		if !ok {
			cmd = ofType(ct)
		}
		t.Run("allow "+commandName(ct), func(t *testing.T) { requireCommandAllow(t, respond(tok, cmd)) })
	}

	for ct, name := range map[enumspb.CommandType]string{
		enumspb.COMMAND_TYPE_SCHEDULE_NEXUS_OPERATION:       "ScheduleNexusOperation",
		enumspb.COMMAND_TYPE_REQUEST_CANCEL_NEXUS_OPERATION: "RequestCancelNexusOperation",
		enumspb.COMMAND_TYPE_UNSPECIFIED:                    "Unspecified",
		enumspb.CommandType(15):                             commandUnknown, // gap in the enum
		enumspb.CommandType(1000):                           commandUnknown, // a later API version
	} {
		t.Run("deny "+name, func(t *testing.T) {
			requireCommandDeny(t, denyCase{req: respond(tok, ofType(ct)),
				reason: ReasonCommandNotAllowed, command: name, workflowID: wfA, commandIdx: "0"})
		})
	}
}

// A Temporal upgrade that adds a command type fails here until someone names
// it in commandNames and decides whether E1 allows it.
func TestCommandNamesCoverPinnedEnum(t *testing.T) {
	for v, enumName := range enumspb.CommandType_name {
		if _, ok := commandNames[enumspb.CommandType(v)]; !ok {
			t.Errorf("command type %s (%d) is not classified in commandNames", enumName, v)
		}
	}
	if len(commandNames) != len(enumspb.CommandType_name) {
		t.Errorf("commandNames has %d entries, the enum has %d", len(commandNames), len(enumspb.CommandType_name))
	}
}

// Command inspection applies to nava-worker only: other platform identities
// cannot call the API at all, and temporal-system runs its own system workflows.
func TestCommandInspectionScope(t *testing.T) {
	a := NewAuthorizer(log.NewNoopLogger(), metrics.NoopMetricsHandler)
	bad := respond(taskToken(t, wfA), startChild(Namespace, wfB))
	res, _ := a.Authorize(context.Background(), claimsFor(System), &authorization.CallTarget{
		APIName: respondWorkflowTaskCompleted, Namespace: "temporal-system", Request: respond(taskToken(t, "temporal-sys-scheduler"))})
	if res.Decision != authorization.DecisionAllow {
		t.Error("temporal-system workflow task denied")
	}
	for _, id := range []Identity{Relay, Core, Operator} {
		res, _ := a.Authorize(context.Background(), claimsFor(id), &authorization.CallTarget{
			APIName: respondWorkflowTaskCompleted, Namespace: Namespace, Request: bad})
		if res.Decision != authorization.DecisionDeny {
			t.Errorf("%s RespondWorkflowTaskCompleted allowed", id)
		}
	}
	// Other worker APIs carry no commands and are not inspected.
	res, _ = a.Authorize(context.Background(), claimsFor(Worker), &authorization.CallTarget{
		APIName: wf + "RespondWorkflowTaskFailed", Namespace: Namespace, Request: &workflowservice.RespondWorkflowTaskFailedRequest{}})
	if res.Decision != authorization.DecisionAllow {
		t.Error("worker RespondWorkflowTaskFailed denied")
	}
}

// A panic while inspecting (here, from the metrics sink) is still a deny.
func TestCommandInspectionNeverPanics(t *testing.T) {
	a := NewAuthorizer(panickingLogger{}, panickingMetrics{})
	for _, req := range []any{nil, respond(nil), respond(taskToken(t, wfA), startChild(Namespace, wfB)),
		respond(taskToken(t, wfA), nil, &commandpb.Command{})} {
		res, err := a.Authorize(context.Background(), claimsFor(Worker), &authorization.CallTarget{
			APIName: respondWorkflowTaskCompleted, Namespace: Namespace, Request: req})
		if err != nil || res.Decision != authorization.DecisionDeny {
			t.Errorf("request %v: got %v, %v; want deny", req, res.Decision, err)
		}
	}
}

func TestTenantOf(t *testing.T) {
	for in, want := range map[string]string{
		"t:a:b": "a", "t:a:": "a", "t:a:b:c": "a",
		"t::b": "", "t:a": "", "x:a:b": "", "": "", "T:a:b": "", " t:a:b": "",
	} {
		got, ok := tenantOf(in)
		if got != want || ok != (want != "") {
			t.Errorf("tenantOf(%q) = %q, %v; want %q", in, got, ok, want)
		}
	}
}

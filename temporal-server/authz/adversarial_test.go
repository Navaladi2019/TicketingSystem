package authz

// Adversarial isolation tests (CI-6). They drive the Authorizer and ClaimMapper
// through Temporal's own authorization interceptor, so they also pin how the
// server feeds them: which certificate it trusts, which namespace it extracts,
// and which cross-namespace command targets it re-authorizes.

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"testing"

	commandpb "go.temporal.io/api/command/v1"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/server/common/authorization"
	"go.temporal.io/server/common/log"
	"go.temporal.io/server/common/metrics"
	"go.temporal.io/server/common/namespace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

type anyNamespace struct{}

func (anyNamespace) Exists(namespace.Name) error { return nil }

func newTestInterceptor() *authorization.Interceptor {
	return authorization.NewInterceptor(
		ClaimMapper{},
		NewAuthorizer(log.NewNoopLogger(), metrics.NoopMetricsHandler),
		metrics.NoopMetricsHandler,
		log.NewNoopLogger(),
		anyNamespace{},
		nil, "", "",
		func() bool { return true },  // system.enableCrossNamespaceCommands (server default)
		func() bool { return false }, // system.disableStreamingAuthorizer (server default)
	)
}

// peerCtx returns a context whose TLS peer presents a certificate with the CN.
// verified=false models a certificate the handshake did not chain to the CA.
func peerCtx(cn string, verified bool) context.Context {
	cert := &x509.Certificate{Subject: pkix.Name{CommonName: cn}, SerialNumber: big.NewInt(1)}
	state := tls.ConnectionState{PeerCertificates: []*x509.Certificate{cert}}
	if verified {
		state.VerifiedChains = [][]*x509.Certificate{{cert}}
	}
	return peer.NewContext(context.Background(), &peer.Peer{
		Addr:     &net.TCPAddr{IP: net.IPv4(10, 0, 0, 9), Port: 40000},
		AuthInfo: credentials.TLSInfo{State: state},
	})
}

func callUnary(t *testing.T, ctx context.Context, method string, req any) (called bool, err error) {
	t.Helper()
	_, err = newTestInterceptor().Intercept(ctx, req, &grpc.UnaryServerInfo{FullMethod: method},
		func(context.Context, any) (any, error) { called = true; return nil, nil })
	return called, err
}

func requireDenied(t *testing.T, called bool, err error) {
	t.Helper()
	if err == nil || called {
		t.Fatalf("call reached the handler (err=%v, called=%v); want denied", err, called)
	}
}

// A bearer token alone, even one naming a valid identity, never yields an identity.
func TestAdversarialTokenOnlyDenied(t *testing.T) {
	ctx := metadata.NewIncomingContext(context.Background(),
		metadata.Pairs("authorization", "Bearer nava-relay", "authorization-extras", "nava-operator"))
	called, err := callUnary(t, ctx, wf+"StartWorkflowExecution",
		&workflowservice.StartWorkflowExecutionRequest{Namespace: Namespace, WorkflowId: "t:tenantB:x"})
	requireDenied(t, called, err)
}

// The identity comes only from a verified chain: a presented-but-unverified
// certificate with the operator CN gets nothing.
func TestAdversarialUnverifiedCertificateDenied(t *testing.T) {
	called, err := callUnary(t, peerCtx(string(Operator), false), wf+"ListWorkflowExecutions",
		&workflowservice.ListWorkflowExecutionsRequest{Namespace: Namespace})
	requireDenied(t, called, err)
}

// Look-alike namespaces never pass the gate.
func TestAdversarialNamespaceLookalikesDenied(t *testing.T) {
	for _, ns := range []string{"", "NAVA", "nava ", " nava", "nava\x00", "nava/", "temporal-system", "default"} {
		called, err := callUnary(t, peerCtx(string(Relay), true), wf+"StartWorkflowExecution",
			&workflowservice.StartWorkflowExecutionRequest{Namespace: ns, WorkflowId: "t:tenantA:x"})
		if err == nil || called {
			t.Errorf("relay start in namespace %q allowed", ns)
		}
	}
}

// A compromised worker certificate cannot use workflow-task commands to start,
// signal or cancel in another namespace (e.g. the system namespace), while the
// same commands inside `nava` pass only for the task's own tenant.
func TestAdversarialWorkerCrossNamespaceCommandsDenied(t *testing.T) {
	commands := map[string]*commandpb.Command{
		"start child": {
			CommandType: enumspb.COMMAND_TYPE_START_CHILD_WORKFLOW_EXECUTION,
			Attributes: &commandpb.Command_StartChildWorkflowExecutionCommandAttributes{
				StartChildWorkflowExecutionCommandAttributes: &commandpb.StartChildWorkflowExecutionCommandAttributes{Namespace: "temporal-system", WorkflowId: "x"}},
		},
		"signal external": {
			CommandType: enumspb.COMMAND_TYPE_SIGNAL_EXTERNAL_WORKFLOW_EXECUTION,
			Attributes: &commandpb.Command_SignalExternalWorkflowExecutionCommandAttributes{
				SignalExternalWorkflowExecutionCommandAttributes: &commandpb.SignalExternalWorkflowExecutionCommandAttributes{Namespace: "temporal-system"}},
		},
		"cancel external": {
			CommandType: enumspb.COMMAND_TYPE_REQUEST_CANCEL_EXTERNAL_WORKFLOW_EXECUTION,
			Attributes: &commandpb.Command_RequestCancelExternalWorkflowExecutionCommandAttributes{
				RequestCancelExternalWorkflowExecutionCommandAttributes: &commandpb.RequestCancelExternalWorkflowExecutionCommandAttributes{Namespace: "temporal-system"}},
		},
	}
	for name, cmd := range commands {
		t.Run(name, func(t *testing.T) {
			called, err := callUnary(t, peerCtx(string(Worker), true), wf+"RespondWorkflowTaskCompleted",
				&workflowservice.RespondWorkflowTaskCompletedRequest{Namespace: Namespace, Commands: []*commandpb.Command{cmd}})
			requireDenied(t, called, err)
		})
	}

	// Inside `nava`, a same-tenant child passes; another tenant's is denied (T27b).
	tok := taskToken(t, "t:tenantA:parent")
	called, err := callUnary(t, peerCtx(string(Worker), true), wf+"RespondWorkflowTaskCompleted",
		&workflowservice.RespondWorkflowTaskCompletedRequest{Namespace: Namespace, TaskToken: tok,
			Commands: []*commandpb.Command{startChild(Namespace, "t:tenantA:child")}})
	if err != nil || !called {
		t.Fatalf("same-tenant child denied: err=%v called=%v", err, called)
	}
	called, err = callUnary(t, peerCtx(string(Worker), true), wf+"RespondWorkflowTaskCompleted",
		&workflowservice.RespondWorkflowTaskCompletedRequest{Namespace: Namespace, TaskToken: tok,
			Commands: []*commandpb.Command{startChild(Namespace, "t:tenantB:child")}})
	requireDenied(t, called, err)
}

type fakeStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s fakeStream) Context() context.Context { return s.ctx }

// The replication stream carries every namespace's history; no platform
// certificate, the operator included, may open it.
func TestAdversarialReplicationStreamDenied(t *testing.T) {
	for _, id := range []Identity{Relay, Core, Worker, Operator} {
		called := false
		err := newTestInterceptor().InterceptStream(nil, fakeStream{ctx: peerCtx(string(id), true)},
			&grpc.StreamServerInfo{FullMethod: adm + "StreamWorkflowReplicationMessages"},
			func(any, grpc.ServerStream) error { called = true; return nil })
		if err == nil || called {
			t.Errorf("%s opened the replication stream", id)
		}
	}
}

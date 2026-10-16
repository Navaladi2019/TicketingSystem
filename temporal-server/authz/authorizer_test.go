package authz

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"sync"
	"testing"

	"go.temporal.io/server/common/authorization"
	"go.temporal.io/server/common/log"
	"go.temporal.io/server/common/log/tag"
	"go.temporal.io/server/common/metrics"
	"go.temporal.io/server/common/metrics/metricstest"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
)

func TestClaimMapper(t *testing.T) {
	subject := func(cn string) *authorization.AuthInfo {
		return &authorization.AuthInfo{TLSSubject: &pkix.Name{CommonName: cn}}
	}
	cases := []struct {
		name string
		in   *authorization.AuthInfo
		want string
	}{
		{"relay", subject("nava-relay"), "nava-relay"},
		{"core", subject("nava-core"), "nava-core"},
		{"worker", subject("nava-worker"), "nava-worker"},
		{"operator", subject("nava-operator"), "nava-operator"},
		{"system", subject("temporal-system"), "temporal-system"},
		{"unknown CN", subject("nava-unknown"), ""},
		{"case differs", subject("NAVA-RELAY"), ""},
		{"padded", subject(" nava-relay"), ""},
		{"empty CN", subject(""), ""},
		{"no auth info", nil, ""},
		{"no certificate", &authorization.AuthInfo{}, ""},
		{"bearer token only", &authorization.AuthInfo{AuthToken: "Bearer nava-relay"}, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := ClaimMapper{}.GetClaims(c.in)
			if err != nil {
				t.Fatalf("GetClaims error: %v", err)
			}
			if got == nil || got.Subject != c.want {
				t.Fatalf("GetClaims = %+v, want subject %q", got, c.want)
			}
			if got.System != authorization.RoleUndefined || len(got.Namespaces) != 0 {
				t.Fatalf("GetClaims must not grant roles: %+v", got)
			}
		})
	}
}

// Acceptance: nava-operator cannot reapply, import or delete history, or
// delete the namespace, even though it may read everything.
func TestOperatorDeniedDestructiveCalls(t *testing.T) {
	a := NewAuthorizer(log.NewNoopLogger(), metrics.NoopMetricsHandler)
	for _, target := range []authorization.CallTarget{
		{APIName: adm + "ReapplyEvents", Namespace: Namespace},
		{APIName: adm + "ImportWorkflowExecution", Namespace: Namespace},
		{APIName: adm + "DeleteWorkflowExecution", Namespace: Namespace},
		{APIName: wf + "DeleteWorkflowExecution", Namespace: Namespace},
		{APIName: op + "DeleteNamespace", Namespace: Namespace},
		{APIName: wf + "ResetWorkflowExecution", Namespace: Namespace},
		{APIName: adm + "MergeDLQTasks"},
		{APIName: adm + "PurgeDLQMessages"},
		{APIName: op + "CreateNexusEndpoint"},
		{APIName: wf + "StartWorkflowExecution", Namespace: Namespace},
		{APIName: wf + "SignalWorkflowExecution", Namespace: Namespace},
		{APIName: wf + "UpdateWorkflowExecution", Namespace: Namespace},
	} {
		res, err := a.Authorize(context.Background(), claimsFor(Operator), &target)
		if err != nil || res.Decision != authorization.DecisionDeny {
			t.Errorf("operator %s: got %v, %v; want deny", target.APIName, res.Decision, err)
		}
	}
	for _, target := range []authorization.CallTarget{
		{APIName: wf + "TerminateWorkflowExecution", Namespace: Namespace},
		{APIName: op + "AddSearchAttributes", Namespace: Namespace},
		{APIName: wf + "GetWorkflowExecutionHistoryReverse", Namespace: Namespace},
		{APIName: adm + "DescribeCluster"},
	} {
		res, _ := a.Authorize(context.Background(), claimsFor(Operator), &target)
		if res.Decision != authorization.DecisionAllow {
			t.Errorf("operator %s: denied, want allow", target.APIName)
		}
	}
}

func TestHealthCheckAllowedForAnyCaller(t *testing.T) {
	a := NewAuthorizer(log.NewNoopLogger(), metrics.NoopMetricsHandler)
	for _, id := range matrixIdentities {
		res, _ := a.Authorize(context.Background(), claimsFor(id), &authorization.CallTarget{APIName: healthCheck})
		if res.Decision != authorization.DecisionAllow {
			t.Errorf("health check denied for %q", id)
		}
	}
	res, _ := a.Authorize(context.Background(), claimsFor(Relay), &authorization.CallTarget{APIName: "/grpc.health.v1.Health/Watch"})
	if res.Decision != authorization.DecisionDeny {
		t.Error("only Health/Check is exempt")
	}
}

// Non-Temporal services on the frontend (gRPC reflection, the Nexus HTTP
// pseudo-service) are denied to everyone but the system worker.
func TestUnclassifiedAPIsDenied(t *testing.T) {
	a := NewAuthorizer(log.NewNoopLogger(), metrics.NoopMetricsHandler)
	for _, apiName := range []string{
		"/grpc.reflection.v1.ServerReflection/ServerReflectionInfo",
		"/temporal.api.nexusservice.v1.NexusService/DispatchNexusTask",
		"",
		"garbage",
	} {
		for _, id := range []Identity{Relay, Core, Worker, Operator} {
			res, _ := a.Authorize(context.Background(), claimsFor(id), &authorization.CallTarget{APIName: apiName, Namespace: Namespace})
			if res.Decision != authorization.DecisionDeny {
				t.Errorf("%s %q allowed", id, apiName)
			}
		}
	}
}

func TestDenyEmitsLogAndMetric(t *testing.T) {
	capture := metricstest.NewCaptureHandler()
	rec := capture.StartCapture()
	defer capture.StopCapture(rec)
	logger := &captureLogger{}
	a := NewAuthorizer(logger, capture)

	cert := &x509.Certificate{
		Subject:      pkix.Name{CommonName: "nava-worker", Organization: []string{"nava"}},
		Issuer:       pkix.Name{CommonName: "nava-internal-ca"},
		SerialNumber: big.NewInt(4242),
	}
	ctx := peer.NewContext(context.Background(), &peer.Peer{
		Addr: &net.TCPAddr{IP: net.IPv4(10, 0, 0, 1), Port: 5555},
		AuthInfo: credentials.TLSInfo{State: tls.ConnectionState{
			VerifiedChains: [][]*x509.Certificate{{cert}},
		}},
	})

	res, err := a.Authorize(ctx, claimsFor(Worker), &authorization.CallTarget{
		APIName: wf + "StartWorkflowExecution", Namespace: Namespace,
	})
	if err != nil || res.Decision != authorization.DecisionDeny {
		t.Fatalf("got %v, %v; want deny", res.Decision, err)
	}
	if res.Reason != "" {
		t.Errorf("deny reason must not reach the caller, got %q", res.Reason)
	}

	recs := rec.Snapshot()[DeniedMetric]
	if len(recs) != 1 {
		t.Fatalf("want 1 %s recording, got %d", DeniedMetric, len(recs))
	}
	wantTags := map[string]string{
		"identity": "nava-worker", "api": wf + "StartWorkflowExecution",
		"namespace": "nava", "reason": ReasonAPI,
	}
	for k, v := range wantTags {
		if recs[0].Tags[k] != v {
			t.Errorf("metric tag %s = %q, want %q", k, recs[0].Tags[k], v)
		}
	}

	if len(logger.entries) != 1 {
		t.Fatalf("want 1 log entry, got %d", len(logger.entries))
	}
	got := logger.entries[0]
	for k, v := range map[string]string{
		"identity": "nava-worker", "api": wf + "StartWorkflowExecution", "namespace": "nava",
		"reason": ReasonAPI, "cert_subject": "CN=nava-worker,O=nava", "cert_serial": "4242",
		"cert_issuer": "CN=nava-internal-ca",
	} {
		if got[k] != v {
			t.Errorf("log field %s = %q, want %q", k, got[k], v)
		}
	}

	// Allowed calls emit nothing.
	_, _ = a.Authorize(ctx, claimsFor(Worker), &authorization.CallTarget{APIName: wf + "PollActivityTaskQueue", Namespace: Namespace})
	if n := len(rec.Snapshot()[DeniedMetric]); n != 1 {
		t.Errorf("allowed call recorded a deny metric")
	}
}

func TestDenyMetricCardinalityIsBounded(t *testing.T) {
	capture := metricstest.NewCaptureHandler()
	rec := capture.StartCapture()
	defer capture.StopCapture(rec)
	a := NewAuthorizer(&captureLogger{}, capture)

	_, _ = a.Authorize(context.Background(), &authorization.Claims{Subject: "attacker-chosen-cn"},
		&authorization.CallTarget{APIName: "/x.Y/Z", Namespace: "attacker-chosen-namespace"})
	r := rec.Snapshot()[DeniedMetric][0]
	for k, v := range map[string]string{"identity": "none", "api": "other", "namespace": "other", "reason": ReasonNoIdentity} {
		if r.Tags[k] != v {
			t.Errorf("tag %s = %q, want %q", k, r.Tags[k], v)
		}
	}
}

func TestDenyReasons(t *testing.T) {
	a := NewAuthorizer(log.NewNoopLogger(), metrics.NoopMetricsHandler)
	cases := []struct {
		id        Identity
		api, ns   string
		wantEmpty bool
		want      string
	}{
		{"", wf + "GetSystemInfo", "", false, ReasonNoIdentity},
		{Relay, wf + "QueryWorkflow", Namespace, false, ReasonAPI},
		{Relay, wf + "StartWorkflowExecution", "default", false, ReasonNamespace},
		{Relay, wf + "StartWorkflowExecution", "", false, ReasonNamespace},
		{Relay, wf + "StartWorkflowExecution", Namespace, true, ""},
		{Relay, wf + "GetSystemInfo", "", true, ""},
		{Operator, adm + "DescribeCluster", "", true, ""},
		{Operator, adm + "GetNamespace", "temporal-system", false, ReasonNamespace},
		{System, adm + "ReapplyEvents", "temporal-system", true, ""},
	}
	for _, c := range cases {
		if got := a.decide(c.id, c.api, c.ns); got != c.want {
			t.Errorf("decide(%q, %s, %q) = %q, want %q", c.id, c.api, c.ns, got, c.want)
		}
	}
}

func TestAuthorizeNeverPanics(t *testing.T) {
	a := NewAuthorizer(panickingLogger{}, panickingMetrics{})
	res, err := a.Authorize(context.Background(), nil, nil)
	if err != nil || res.Decision != authorization.DecisionDeny {
		t.Fatalf("nil target: got %v, %v; want deny", res.Decision, err)
	}
	res, err = a.Authorize(context.Background(), claimsFor(Relay), &authorization.CallTarget{APIName: wf + "ResetWorkflowExecution", Namespace: Namespace})
	if err != nil || res.Decision != authorization.DecisionDeny {
		t.Fatalf("panicking sinks: got %v, %v; want deny", res.Decision, err)
	}

	// A panic inside the decision itself is a deny, not a crash.
	broken := &Authorizer{logger: log.NewNoopLogger(), metrics: metrics.NoopMetricsHandler}
	res, err = broken.Authorize(context.Background(), claimsFor(Relay), &authorization.CallTarget{APIName: wf + "StartWorkflowExecution", Namespace: Namespace})
	if err != nil || res.Decision != authorization.DecisionDeny {
		t.Fatalf("zero-value authorizer: got %v, %v; want deny", res.Decision, err)
	}
}

func TestAuthorizeConcurrent(t *testing.T) {
	a := NewAuthorizer(log.NewNoopLogger(), metrics.NoopMetricsHandler)
	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			id := []Identity{Relay, Core, Worker, Operator}[i%4]
			for j := 0; j < 500; j++ {
				res, _ := a.Authorize(context.Background(), claimsFor(id), &authorization.CallTarget{APIName: wf + "GetSystemInfo"})
				if res.Decision != authorization.DecisionAllow {
					t.Errorf("%s GetSystemInfo denied", id)
					return
				}
			}
		}(i)
	}
	wg.Wait()
}

type captureLogger struct {
	mu      sync.Mutex
	entries []map[string]string
}

func (l *captureLogger) record(tags []tag.Tag) {
	l.mu.Lock()
	defer l.mu.Unlock()
	m := map[string]string{}
	for _, t := range tags {
		if s, ok := t.Value().(string); ok {
			m[t.Key()] = s
		}
	}
	l.entries = append(l.entries, m)
}

func (l *captureLogger) Debug(string, ...tag.Tag)        {}
func (l *captureLogger) Info(string, ...tag.Tag)         {}
func (l *captureLogger) Warn(_ string, tags ...tag.Tag)  { l.record(tags) }
func (l *captureLogger) Error(_ string, tags ...tag.Tag) { l.record(tags) }
func (l *captureLogger) DPanic(string, ...tag.Tag)       {}
func (l *captureLogger) Panic(string, ...tag.Tag)        {}
func (l *captureLogger) Fatal(string, ...tag.Tag)        {}

type panickingLogger struct{}

func (panickingLogger) Debug(string, ...tag.Tag)  { panic("log") }
func (panickingLogger) Info(string, ...tag.Tag)   { panic("log") }
func (panickingLogger) Warn(string, ...tag.Tag)   { panic("log") }
func (panickingLogger) Error(string, ...tag.Tag)  { panic("log") }
func (panickingLogger) DPanic(string, ...tag.Tag) { panic("log") }
func (panickingLogger) Panic(string, ...tag.Tag)  { panic("log") }
func (panickingLogger) Fatal(string, ...tag.Tag)  { panic("log") }

type panickingMetrics struct{ metrics.Handler }

func (panickingMetrics) WithTags(...metrics.Tag) metrics.Handler { panic("metrics") }

package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/api/operatorservice/v1"
	taskqueuepb "go.temporal.io/api/taskqueue/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/server/api/adminservice/v1"
	"go.temporal.io/server/common/config"
	"go.temporal.io/server/common/dynamicconfig"
	"go.temporal.io/server/common/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/Navaladi2019/TicketingSystem/temporal-server/authz"
	"github.com/Navaladi2019/TicketingSystem/temporal-server/internal/devcerts"
)

// TestServerEndToEnd starts the server from config/development.yaml (or, with
// NAVA_TEMPORAL_E2E_ADDR and NAVA_TEMPORAL_E2E_CERTS set, targets a running
// nava/temporal-server image) and checks the allow-list over real mTLS gRPC.
func TestServerEndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("starts a Temporal server")
	}
	addr, certDir, metricsURL := os.Getenv("NAVA_TEMPORAL_E2E_ADDR"), os.Getenv("NAVA_TEMPORAL_E2E_CERTS"), ""
	if addr == "" {
		addr, certDir, metricsURL = startInProcess(t)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	wf := func(cn string) workflowservice.WorkflowServiceClient {
		return workflowservice.NewWorkflowServiceClient(dial(t, addr, certDir, cn))
	}
	relay, core, worker, operator, system, unknown :=
		wf("nava-relay"), wf("nava-core"), wf("nava-worker"), wf("nava-operator"), wf("temporal-system"), wf("nava-unknown")
	operatorOps := operatorservice.NewOperatorServiceClient(dial(t, addr, certDir, "nava-operator"))
	operatorAdmin := adminservice.NewAdminServiceClient(dial(t, addr, certDir, "nava-operator"))

	eventually(t, ctx, "frontend serving nava-relay", func(ctx context.Context) error {
		_, err := relay.GetSystemInfo(ctx, &workflowservice.GetSystemInfoRequest{})
		return err
	})

	// Only temporal-system may register the namespace (bootstrap path for T28).
	_, err := operator.RegisterNamespace(ctx, &workflowservice.RegisterNamespaceRequest{Namespace: authz.Namespace})
	wantCode(t, "operator RegisterNamespace", err, codes.PermissionDenied)
	_, err = system.RegisterNamespace(ctx, &workflowservice.RegisterNamespaceRequest{
		Namespace: authz.Namespace, WorkflowExecutionRetentionPeriod: durationpb.New(24 * time.Hour),
	})
	if err != nil && status.Code(err) != codes.AlreadyExists {
		t.Fatalf("temporal-system RegisterNamespace: %v", err)
	}
	eventually(t, ctx, "nava namespace visible to nava-worker", func(ctx context.Context) error {
		_, err := worker.DescribeNamespace(ctx, &workflowservice.DescribeNamespaceRequest{Namespace: authz.Namespace})
		return err
	})

	nsReq := &workflowservice.DescribeNamespaceRequest{Namespace: authz.Namespace}
	start := func(c workflowservice.WorkflowServiceClient) error {
		_, err := c.StartWorkflowExecution(ctx, &workflowservice.StartWorkflowExecutionRequest{
			Namespace:  authz.Namespace,
			WorkflowId: fmt.Sprintf("t:e2e:%d", time.Now().UnixNano()),
			// The D9 workload allow-list's entries (authz/policy.go).
			WorkflowType: &commonpb.WorkflowType{Name: "nava.smoke.Echo"},
			TaskQueue:    &taskqueuepb.TaskQueue{Name: "nava-tq"},
			RequestId:    fmt.Sprint(time.Now().UnixNano()),
		})
		return err
	}
	call := func(f func() error) error { return f() }

	cases := []struct {
		name string
		err  error
		want codes.Code
	}{
		{"unknown CN GetSystemInfo", call(func() error {
			_, err := unknown.GetSystemInfo(ctx, &workflowservice.GetSystemInfoRequest{})
			return err
		}), codes.PermissionDenied},
		{"relay StartWorkflowExecution", start(relay), codes.OK},
		{"core StartWorkflowExecution", start(core), codes.PermissionDenied},
		{"worker StartWorkflowExecution", start(worker), codes.PermissionDenied},
		{"relay DescribeNamespace", call(func() error { _, err := relay.DescribeNamespace(ctx, nsReq); return err }), codes.PermissionDenied},
		{"worker DescribeNamespace temporal-system", call(func() error {
			_, err := worker.DescribeNamespace(ctx, &workflowservice.DescribeNamespaceRequest{Namespace: "temporal-system"})
			return err
		}), codes.PermissionDenied},
		{"operator ListNamespaces", call(func() error {
			_, err := operator.ListNamespaces(ctx, &workflowservice.ListNamespacesRequest{})
			return err
		}), codes.OK},
		{"operator DescribeCluster", call(func() error {
			_, err := operatorAdmin.DescribeCluster(ctx, &adminservice.DescribeClusterRequest{})
			return err
		}), codes.OK},
		{"operator ReapplyEvents", call(func() error {
			_, err := operatorAdmin.ReapplyEvents(ctx, &adminservice.ReapplyEventsRequest{})
			return err
		}), codes.PermissionDenied},
		{"operator GetWorkflowExecutionRawHistory", call(func() error {
			_, err := operatorAdmin.GetWorkflowExecutionRawHistory(ctx, &adminservice.GetWorkflowExecutionRawHistoryRequest{NamespaceId: "nava"})
			return err
		}), codes.PermissionDenied},
		{"operator GetWorkflowExecutionRawHistoryV2", call(func() error {
			_, err := operatorAdmin.GetWorkflowExecutionRawHistoryV2(ctx, &adminservice.GetWorkflowExecutionRawHistoryV2Request{NamespaceId: "nava"})
			return err
		}), codes.PermissionDenied},
		{"operator DeleteNamespace", call(func() error {
			_, err := operatorOps.DeleteNamespace(ctx, &operatorservice.DeleteNamespaceRequest{Namespace: authz.Namespace})
			return err
		}), codes.PermissionDenied},
	}
	for _, c := range cases {
		wantCode(t, c.name, c.err, c.want)
	}

	// A certificate from another CA, or none, fails at the handshake.
	for _, cn := range []string{devcerts.RogueCN, ""} {
		c := workflowservice.NewWorkflowServiceClient(dial(t, addr, certDir, cn))
		cctx, ccancel := context.WithTimeout(ctx, 5*time.Second)
		_, err := c.GetSystemInfo(cctx, &workflowservice.GetSystemInfoRequest{})
		ccancel()
		if err == nil || status.Code(err) == codes.PermissionDenied {
			t.Errorf("client cert %q: got %v, want a handshake failure", cn, err)
		}
	}

	if metricsURL != "" {
		// Tally flushes to the Prometheus registry on its reporting interval, and
		// its reporter rewrites '-' and '/' in tag values to '_'.
		eventually(t, ctx, "deny metric exported", func(context.Context) error {
			for _, line := range strings.Split(scrape(t, metricsURL), "\n") {
				if strings.HasPrefix(line, authz.DeniedMetric+"{") &&
					strings.Contains(line, `api="_temporal_api_workflowservice_v1_WorkflowService_StartWorkflowExecution"`) &&
					strings.Contains(line, `identity="nava_core"`) &&
					strings.Contains(line, `reason="api_not_allowed"`) {
					return nil
				}
			}
			return fmt.Errorf("no %s sample for nava-core", authz.DeniedMetric)
		})
	}
}

func startInProcess(t *testing.T) (addr, certDir, metricsURL string) {
	t.Helper()
	certDir = t.TempDir()
	if err := devcerts.Write(certDir, time.Hour); err != nil {
		t.Fatal(err)
	}
	metricsAddr := freeAddr(t)
	t.Setenv("NAVA_TEMPORAL_CERT_DIR", filepath.ToSlash(certDir))
	t.Setenv("NAVA_TEMPORAL_FRONTEND_BIND_IP", "127.0.0.1")
	t.Setenv("NAVA_TEMPORAL_METRICS_ADDR", metricsAddr)

	configDir, err := filepath.Abs(filepath.Join("..", "..", "config"))
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := config.LoadConfig("development", configDir, "")
	if err != nil {
		t.Fatalf("load dev config: %v", err)
	}
	s, err := newServer(cfg, log.NewNoopLogger(), dynamicconfig.NewNoopClient())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	t.Cleanup(func() { _ = s.Stop() })
	return "127.0.0.1:7233", certDir, "http://" + metricsAddr + "/metrics"
}

func TestCheckTLSRefusesConfigWithoutMTLS(t *testing.T) {
	err := checkTLS(&config.Config{})
	if err == nil {
		t.Fatal("empty TLS config accepted")
	}
	for _, want := range []string{"requireClientAuth", "installation CA", "internode", "temporal-system"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %q", err, want)
		}
	}
	ok := validConfig(t)
	if err := checkTLS(ok); err != nil {
		t.Fatalf("valid config refused: %v", err)
	}
	ok.Global.Authorization.Authorizer = "default"
	if err := checkTLS(ok); err == nil {
		t.Fatal("config-selected authorizer accepted")
	}
}

func dial(t *testing.T, addr, certDir, cn string) *grpc.ClientConn {
	t.Helper()
	caPEM, err := os.ReadFile(filepath.Join(certDir, devcerts.CAFile))
	if err != nil {
		t.Fatal(err)
	}
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(caPEM)
	tc := &tls.Config{RootCAs: pool, ServerName: "localhost", MinVersion: tls.VersionTLS12}
	if cn != "" {
		pair, err := tls.LoadX509KeyPair(filepath.Join(certDir, cn+".pem"), filepath.Join(certDir, cn+".key"))
		if err != nil {
			t.Fatal(err)
		}
		tc.Certificates = []tls.Certificate{pair}
	}
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(credentials.NewTLS(tc)))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

func wantCode(t *testing.T, name string, err error, want codes.Code) {
	t.Helper()
	if got := status.Code(err); got != want {
		t.Errorf("%s: got %v (%v), want %v", name, got, err, want)
	}
}

func eventually(t *testing.T, ctx context.Context, what string, f func(context.Context) error) {
	t.Helper()
	var last error
	for ctx.Err() == nil {
		cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		last = f(cctx)
		cancel()
		if last == nil {
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s: %v", what, errors.Join(last, ctx.Err()))
}

func freeAddr(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	return l.Addr().String()
}

func scrape(t *testing.T, url string) string {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("scrape %s: %v", url, err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return string(b)
}

// TestInternodeRejectsFrontendClientCertificates: the internode ports have no
// Authorizer, so a certificate from the frontend (installation) CA, such as the
// worker's, must fail at the internode handshake, while the internode
// certificate completes it. In-process only: it needs the history port.
func TestInternodeRejectsFrontendClientCertificates(t *testing.T) {
	if testing.Short() {
		t.Skip("starts a Temporal server")
	}
	if os.Getenv("NAVA_TEMPORAL_E2E_ADDR") != "" {
		t.Skip("in-process only")
	}
	addr, certDir, _ := startInProcess(t)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	eventually(t, ctx, "every service listening", func(context.Context) error {
		for _, p := range []string{"7233", "7234", "7235"} {
			c, err := net.DialTimeout("tcp", "127.0.0.1:"+p, time.Second)
			if err != nil {
				return err
			}
			c.Close()
		}
		return nil
	})
	_ = addr
	// 7239 (worker service) is not listening in this in-process setup.
	for _, port := range []string{"7234", "7235"} {
		addr := "127.0.0.1:" + port
		for _, cn := range []string{"nava-worker", "nava-core", "temporal-system"} {
			if err := handshake(t, addr, certDir, cn+".pem", cn+".key", devcerts.InternodeCAFile); err == nil {
				t.Errorf("%s: a %s certificate completed the internode handshake", addr, cn)
			}
		}
		if err := handshake(t, addr, certDir, devcerts.InternodeCertFile, devcerts.InternodeKeyFile, devcerts.InternodeCAFile); err != nil {
			t.Errorf("%s: the internode certificate failed: %v", addr, err)
		}
	}
}

// handshake dials addr with the given client certificate and completes (or
// fails) the TLS handshake plus one read, so a server-side rejection of the
// client certificate (reported after the handshake in TLS 1.3) is seen.
func handshake(t *testing.T, addr, dir, cert, key, caFile string) error {
	t.Helper()
	caPEM, err := os.ReadFile(filepath.Join(dir, caFile))
	if err != nil {
		t.Fatal(err)
	}
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(caPEM)
	pair, err := tls.LoadX509KeyPair(filepath.Join(dir, cert), filepath.Join(dir, key))
	if err != nil {
		t.Fatal(err)
	}
	d := &net.Dialer{Timeout: 5 * time.Second}
	conn, err := tls.DialWithDialer(d, "tcp", addr, &tls.Config{
		RootCAs: pool, ServerName: "localhost", Certificates: []tls.Certificate{pair}, MinVersion: tls.VersionTLS12,
		NextProtos: []string{"h2"}, // gRPC servers refuse a connection without it
	})
	if err != nil {
		return err
	}
	defer conn.Close()
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	// An accepted client gets the server's HTTP/2 SETTINGS frame; a rejected one
	// gets an alert or a close.
	_, err = conn.Read(make([]byte, 1))
	return err
}

func TestBootstrapRegistersNamespaceAndSearchAttribute(t *testing.T) {
	if testing.Short() {
		t.Skip("starts a Temporal server")
	}
	if os.Getenv("NAVA_TEMPORAL_E2E_ADDR") != "" {
		t.Skip("in-process only")
	}
	addr, certDir, _ := startInProcess(t)
	f := tlsFiles{
		CA: filepath.Join(certDir, devcerts.CAFile), ServerName: "localhost",
		Cert: filepath.Join(certDir, "temporal-system.pem"), Key: filepath.Join(certDir, "temporal-system.key"),
	}
	conn, err := f.dial(addr)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	for i := 0; i < 2; i++ { // the second run proves it is idempotent
		if err := ensureNamespace(ctx, conn); err != nil {
			t.Fatalf("run %d: %v", i+1, err)
		}
	}
	var out strings.Builder
	args := []string{"-addr", addr, "-ca", f.CA, "-cert", f.Cert, "-key", f.Key, "-bootstrapped"}
	if err := runHealth(args, &out); err != nil || !strings.Contains(out.String(), "SERVING") {
		t.Fatalf("health -bootstrapped: %q, %v", out.String(), err)
	}
	// The same probe with a certificate from another CA fails the handshake.
	if err := runHealth([]string{"-addr", addr, "-ca", f.CA,
		"-cert", filepath.Join(certDir, devcerts.RogueCN+".pem"), "-key", filepath.Join(certDir, devcerts.RogueCN+".key"),
		"-timeout", "5s"}, io.Discard); err == nil {
		t.Fatal("health with a rogue-CA certificate succeeded")
	}
}

func TestProxyForwardsBytes(t *testing.T) {
	up, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer up.Close()
	go func() {
		c, err := up.Accept()
		if err == nil {
			_, _ = io.Copy(c, c)
			c.Close()
		}
	}()
	front, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer front.Close()
	go func() { _ = serveProxy(front, up.Addr().String()) }()

	c, err := net.Dial("tcp", front.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	_ = c.SetDeadline(time.Now().Add(5 * time.Second))
	if _, err := c.Write([]byte("ping")); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 4)
	if _, err := io.ReadFull(c, buf); err != nil || string(buf) != "ping" {
		t.Fatalf("echo through the proxy: %q, %v", buf, err)
	}
}

//go:build workerstack

// Package workerstack tests the nava-worker compose service (T31b, TICK-159)
// in the running stack: it is ready, its admin listener serves /metrics and
// the probes untraced and unpublished, and it runs a workflow's activity
// through the T31 guard and the nava_worker database role.
//
//	make stack-up
//	make stack-test-worker
//
// Its test workflow polls nava-tq, the only allow-listed queue, so it runs
// after stack-test-sdk, never beside it (STACK_TESTS_SERIAL in the Makefile).
package workerstack

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
	"github.com/Navaladi2019/TicketingSystem/internal/workerguard"
)

// The D9 allow-list's only workload: workflow type nava.smoke.Echo on nava-tq.
const (
	namespace = "nava"
	echoType  = "nava.smoke.Echo"
	taskQueue = "nava-tq"
	service   = "nava-worker"
	// The guard's allow decisions on activities, as Prometheus names them.
	decisions = "nava_worker_guard_decisions_total"
)

// root is the repository root, from this package.
const root = "../.."

// compose runs `docker compose` from the repository root, so COMPOSE_FILE and
// COMPOSE_PROJECT_NAME from make apply as they do to the shell stack tests.
func compose(t *testing.T, stdin string, args ...string) (string, error) {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), "docker", append([]string{"compose"}, args...)...)
	cmd.Dir = root
	cmd.Stdin = strings.NewReader(stdin)
	out, err := cmd.Output()
	return string(out), err
}

// admin returns the raw HTTP response (status line, headers, body) for path
// on the worker's admin listener, from inside its container: the port is
// never published.
func admin(t *testing.T, path string) string {
	t.Helper()
	req := "GET " + path + " HTTP/1.0\r\nHost: localhost\r\n\r\n"
	out, err := compose(t, req, "exec", "-T", service, "nc", "127.0.0.1", "9090")
	if err != nil {
		t.Fatalf("GET %s on the %s admin listener: %v", path, service, err)
	}
	return strings.ReplaceAll(out, "\r", "")
}

func status(resp string) string {
	line, _, _ := strings.Cut(resp, "\n")
	f := strings.Fields(line)
	if len(f) < 2 {
		return ""
	}
	return f[1]
}

// awaitReady waits for /readyz 200: the service started its worker, and
// PostgreSQL and Temporal answer.
func awaitReady(t *testing.T) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Minute)
	for {
		resp := admin(t, "/readyz")
		if status(resp) == "200" {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("%s /readyz not 200 within 2m:\n%s\n/healthz:\n%s", service, resp, admin(t, "/healthz"))
		}
		time.Sleep(time.Second)
	}
}

func TestWorkerServiceReadyAndAdminListener(t *testing.T) {
	awaitReady(t)

	health := admin(t, "/healthz")
	for _, want := range []string{`"postgres":"ok"`, `"temporal":"ok"`} {
		if !strings.Contains(health, want) {
			t.Errorf("/healthz has no %s:\n%s", want, health)
		}
	}

	metrics := admin(t, "/metrics")
	if status(metrics) != "200" {
		t.Fatalf("/metrics:\n%s", metrics)
	}
	head, _, _ := strings.Cut(metrics, "\n\n")
	// httpobs.Middleware sets X-Request-Id on every response; the admin
	// listener is not wrapped in it (TICK-94 item 1).
	for _, h := range []string{"x-request-id:", "traceparent:"} {
		if strings.Contains(strings.ToLower(head), h) {
			t.Errorf("/metrics answered with %s: the admin listener is wrapped in httpobs:\n%s", h, head)
		}
	}
	if !strings.Contains(metrics, "temporal_") {
		t.Error("/metrics has no Temporal SDK metric: the client's MetricsHandler is not telemetry.NewTemporal's")
	}

	// Nothing of the service is published to the host.
	if out, err := compose(t, "", "port", service, "9090"); err == nil && strings.TrimSpace(out) != "" {
		t.Errorf("%s publishes the admin port: %s", service, out)
	}
}

// activityAllows sums the guard's allow decisions on activities.
func activityAllows(t *testing.T) float64 {
	t.Helper()
	var sum float64
	for line := range strings.SplitSeq(admin(t, "/metrics"), "\n") {
		if !strings.HasPrefix(line, decisions+"{") ||
			!strings.Contains(line, `guard="activity"`) || !strings.Contains(line, `outcome="allow"`) {
			continue
		}
		f := strings.Fields(line)
		v, err := strconv.ParseFloat(f[len(f)-1], 64)
		if err != nil {
			t.Fatalf("%s line %q: %v", decisions, line, err)
		}
		sum += v
	}
	return sum
}

// closeRef is registered as the allow-listed nava.smoke.Echo. It schedules
// only the workflow_ref activity, which the nava-worker service alone polls.
func closeRef(ctx workflow.Context) error {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		TaskQueue:           taskQueue,
		StartToCloseTimeout: 30 * time.Second,
		RetryPolicy:         &temporal.RetryPolicy{MaximumAttempts: 1},
	})
	return workflow.ExecuteActivity(ctx, workerguard.CloseWorkflowRefActivity).Get(ctx, nil)
}

// A relay-started workflow's workflow_ref activity runs in the service: the
// guard allows it (the header is the id's tenant) and counts it on /metrics,
// and the activity reaches PostgreSQL as nava_worker, where RLS and the
// tenant find no row for this new workflow.
func TestWorkerServiceRunsTheGuardedActivity(t *testing.T) {
	awaitReady(t)
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Minute)
	defer cancel()
	before := activityAllows(t)

	tel, err := telemetry.New(ctx, config.Telemetry{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tel.Shutdown(context.Background()) })
	tt, err := tel.NewTemporal(slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	setup := func(o *client.Options) { workerguard.ConfigureClient(o, tt) }

	// The test's worker runs the workflow only: no activity poller.
	w := worker.New(dial(t, "nava-worker", setup), taskQueue, worker.Options{LocalActivityWorkerOnly: true})
	w.RegisterWorkflowWithOptions(closeRef, workflow.RegisterOptions{Name: echoType})
	if err := w.Start(); err != nil {
		t.Fatalf("test worker start: %v", err)
	}
	t.Cleanup(w.Stop)

	tenant := uuid.New()
	id := "t:" + tenant.String() + ":t31b:" + uuid.NewString()
	relay := dial(t, "nava-relay", setup)
	run, err := relay.ExecuteWorkflow(workerguard.ContextWithTenant(ctx, tenant),
		client.StartWorkflowOptions{ID: id, TaskQueue: taskQueue, WorkflowExecutionTimeout: 2 * time.Minute}, echoType)
	if err != nil {
		t.Fatalf("start %s: %v", id, err)
	}
	system := dial(t, "temporal-system")
	t.Cleanup(func() { _ = system.TerminateWorkflow(context.Background(), id, "", "T31b cleanup") })

	// The relay may not read history; nava-core reads the result.
	err = dial(t, "nava-core", setup).GetWorkflow(ctx, id, run.GetRunID()).Get(ctx, nil)
	var app *temporal.ApplicationError
	if !errors.As(err, &app) || app.Type() != workerguard.ErrTypeWorkflowRef {
		t.Fatalf("workflow result %v; want the service's %s from the activity", err, workerguard.ErrTypeWorkflowRef)
	}
	if after := activityAllows(t); after < before+1 {
		t.Errorf("%s{guard=activity,outcome=allow} went from %v to %v; want one more", decisions, before, after)
	}
}

func certDir() string {
	if d := os.Getenv("NAVA_TEST_CERT_DIR"); d != "" {
		return d
	}
	return filepath.Join(root, "deploy", "compose", "certs")
}

func address() string {
	if a := os.Getenv("NAVA_TEST_TEMPORAL_ADDR"); a != "" {
		return a
	}
	if p := os.Getenv("TEMPORAL_TEST_PORT"); p != "" {
		return "127.0.0.1:" + p
	}
	return "127.0.0.1:17233"
}

// dial connects as the identity cn through the gateway compose.test.yaml
// publishes on loopback; the frontend certificate names `localhost`. It is
// the temporalstack helper (internal/temporalstack/stack_test.go).
func dial(t *testing.T, cn string, opts ...func(*client.Options)) client.Client {
	t.Helper()
	dir := certDir()
	pair, err := tls.LoadX509KeyPair(filepath.Join(dir, cn+".pem"), filepath.Join(dir, cn+".key"))
	if err != nil {
		t.Fatalf("%s: %v (run make dev-ca)", cn, err)
	}
	caPEM, err := os.ReadFile(filepath.Join(dir, "ca.pem"))
	if err != nil {
		t.Fatal(err)
	}
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(caPEM)
	o := client.Options{
		HostPort:  address(),
		Namespace: namespace,
		ConnectionOptions: client.ConnectionOptions{TLS: &tls.Config{
			Certificates: []tls.Certificate{pair}, RootCAs: pool, ServerName: "localhost", MinVersion: tls.VersionTLS12,
		}},
	}
	for _, opt := range opts {
		opt(&o)
	}
	c, err := client.Dial(o)
	if err != nil {
		t.Fatalf("%s: dial: %v", cn, err)
	}
	t.Cleanup(c.Close)
	return c
}

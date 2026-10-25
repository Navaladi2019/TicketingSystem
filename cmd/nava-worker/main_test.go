package main

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"go.temporal.io/sdk/worker"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
	"github.com/Navaladi2019/TicketingSystem/internal/workerguard"
)

func lookup(env map[string]string) config.LookupFunc {
	return func(k string) (string, bool) { v, ok := env[k]; return v, ok }
}

// Without Temporal or a database the worker has nothing to do: run fails at
// once, naming the variables and never echoing a value.
func TestRunNeedsTemporalAndDatabase(t *testing.T) {
	const dsn = "postgres://nava_worker:s3cret-value@pgbouncer:5432/nava"
	for name, env := range map[string]map[string]string{
		"neither":     {},
		"no database": {"NAVA_TEMPORAL_ADDRESS": "temporal:7233"},
		"no temporal": {"NAVA_DATABASE_URL": dsn},
	} {
		t.Run(name, func(t *testing.T) {
			var out bytes.Buffer
			err := run(t.Context(), lookup(env), &out)
			if err == nil || !strings.Contains(err.Error(), "NAVA_TEMPORAL_ADDRESS and NAVA_DATABASE_URL") {
				t.Fatalf("run = %v, want the missing-variables error", err)
			}
			if strings.Contains(err.Error(), "s3cret") || strings.Contains(out.String(), "s3cret") {
				t.Errorf("the database password reached the error or the log")
			}
		})
	}
}

func TestRunTLSErrorsNameTheVariable(t *testing.T) {
	dir := t.TempDir()
	env := map[string]string{
		"NAVA_TEMPORAL_ADDRESS":       "127.0.0.1:1",
		"NAVA_DATABASE_URL":           "postgres://nava_worker@127.0.0.1:1/nava",
		"NAVA_TEMPORAL_TLS_CERT_FILE": filepath.Join(dir, "secret-name.pem"),
		"NAVA_TEMPORAL_TLS_KEY_FILE":  filepath.Join(dir, "secret-name.key"),
		"NAVA_TEMPORAL_TLS_CA_FILE":   filepath.Join(dir, "ca.pem"),
	}
	err := run(t.Context(), lookup(env), &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "NAVA_TEMPORAL_TLS_CERT_FILE") || strings.Contains(err.Error(), "secret-name") {
		t.Fatalf("run = %v, want an error naming the variable, not the path", err)
	}
}

func TestTemporalTLSUnconfiguredIsPlaintext(t *testing.T) {
	got, err := temporalTLS(config.TemporalTLS{})
	if got != nil || err != nil {
		t.Fatalf("got %v, %v; want nil, nil", got, err)
	}
}

func newTemporal(t *testing.T) (*telemetry.Telemetry, *telemetry.Temporal) {
	t.Helper()
	tel, err := telemetry.New(t.Context(), config.Telemetry{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tel.Shutdown(context.Background()) })
	tt, err := tel.NewTemporal(slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	return tel, tt
}

// The client is namespace nava with exactly what ConfigureClient sets: the
// tenant Propagator alone, the SDK tracing interceptor first, the SDK metrics
// handler and no worker heartbeats.
func TestClientOptionsAreConfiguredByWorkerguard(t *testing.T) {
	_, tt := newTemporal(t)
	o := clientOptions(slog.New(slog.DiscardHandler), "temporal:7233", nil, tt)
	if o.Namespace != "nava" || o.HostPort != "temporal:7233" {
		t.Errorf("namespace %q, host %q; want nava, temporal:7233", o.Namespace, o.HostPort)
	}
	if len(o.ContextPropagators) != 1 {
		t.Fatalf("%d propagators, want only workerguard.Propagator", len(o.ContextPropagators))
	}
	if _, ok := o.ContextPropagators[0].(workerguard.Propagator); !ok {
		t.Errorf("propagator %T, want workerguard.Propagator", o.ContextPropagators[0])
	}
	if len(o.Interceptors) != 1 || o.Interceptors[0] != tt.Tracing {
		t.Errorf("interceptors %v, want the SDK tracing interceptor only", o.Interceptors)
	}
	if o.MetricsHandler == nil || reflect.TypeOf(o.MetricsHandler) != reflect.TypeOf(tt.Metrics) {
		t.Error("metrics handler is not telemetry.NewTemporal's")
	}
	if o.WorkerHeartbeatInterval >= 0 {
		t.Errorf("worker heartbeat interval %v, want negative (off, TICK-94 item 5)", o.WorkerHeartbeatInterval)
	}
}

// The guard is the last worker interceptor, and with no workflow registered
// the worker does not poll workflow tasks.
func TestWorkerOptionsInstallTheGuard(t *testing.T) {
	tel, _ := newTemporal(t)
	g, err := workerguard.New(slog.New(slog.DiscardHandler), tel.MeterProvider)
	if err != nil {
		t.Fatal(err)
	}
	o := workerOptions(g, make(chan error, 1))
	if n := len(o.Interceptors); n == 0 || o.Interceptors[n-1] != g {
		t.Errorf("interceptors %v, want the guard last", o.Interceptors)
	}
	if !o.DisableWorkflowWorker {
		t.Error("the workflow poller is on with no workflow registered")
	}
}

// stubWorker records Stop; serve calls nothing else.
type stubWorker struct {
	worker.Worker
	stopped bool
}

func (s *stubWorker) Stop() { s.stopped = true }

// The SDK stops the worker on a fatal poll error and reports it only through
// OnFatalError (TICK-168 item 1). serve must return it, so run exits non-zero
// and compose restarts the process instead of /readyz staying 200 with no
// poller. A second fatal error never blocks the SDK.
func TestFatalWorkerErrorEndsServe(t *testing.T) {
	tel, _ := newTemporal(t)
	g, err := workerguard.New(slog.New(slog.DiscardHandler), tel.MeterProvider)
	if err != nil {
		t.Fatal(err)
	}
	fatal := make(chan error, 1)
	o := workerOptions(g, fatal)
	if o.OnFatalError == nil {
		t.Fatal("OnFatalError is not set: a fatal worker error would go unnoticed")
	}
	cause := errors.New("namespace nava not found")
	o.OnFatalError(cause)
	o.OnFatalError(errors.New("second")) // must not block

	var logs bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&logs, nil))
	admin := &http.Server{Addr: "127.0.0.1:0", ReadHeaderTimeout: time.Second}
	w := &stubWorker{}
	done := make(chan error, 1)
	go func() {
		done <- serve(t.Context(), log, admin, w, fatal, config.Config{HTTP: config.HTTP{ShutdownTimeout: time.Second}})
	}()

	select {
	case err := <-done:
		if !errors.Is(err, cause) {
			t.Fatalf("serve = %v, want the fatal worker error", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("serve kept running after a fatal worker error")
	}
	if !w.stopped {
		t.Error("serve did not stop the worker")
	}
	if !strings.Contains(logs.String(), `"msg":"nava-worker fatal worker error"`) {
		t.Errorf("no error line for the fatal worker error:\n%s", logs.String())
	}
	if err := admin.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		t.Errorf("ListenAndServe after serve = %v, want ErrServerClosed: the listener was not shut down", err)
	}
}

// A signal still ends serve cleanly, with no error.
func TestServeStopsOnContextWithoutError(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	w := &stubWorker{}
	admin := &http.Server{Addr: "127.0.0.1:0", ReadHeaderTimeout: time.Second}
	err := serve(ctx, slog.New(slog.DiscardHandler), admin, w, make(chan error, 1), config.Config{HTTP: config.HTTP{ShutdownTimeout: time.Second}})
	if err != nil || !w.stopped {
		t.Fatalf("serve = %v, stopped %v; want nil and the worker stopped", err, w.stopped)
	}
}

// The admin port carries no auth, so compose never publishes it; the
// worker's only mention is its own healthcheck URL (as for nava-core).
func TestComposeWorkerPublishesNothing(t *testing.T) {
	b, err := os.ReadFile("../../compose.yaml")
	if err != nil {
		t.Fatal(err)
	}
	svc := serviceBlock(t, strings.ReplaceAll(string(b), "\r\n", "\n"), "nava-worker")
	for _, line := range strings.Split(svc, "\n") {
		code, _, _ := strings.Cut(line, "#")
		if strings.Contains(code, "ports:") {
			t.Errorf("nava-worker publishes a port: %s", strings.TrimSpace(line))
		}
		if strings.Contains(code, "9090") && !strings.Contains(code, "http://127.0.0.1:9090/") {
			t.Errorf("nava-worker mentions the admin port: %s", strings.TrimSpace(line))
		}
	}
	for _, want := range []string{"NAVA_TEMPORAL_TLS_CERT_FILE", "nava-worker.pem", "nava_worker:", "/readyz"} {
		if !strings.Contains(svc, want) {
			t.Errorf("nava-worker service has no %q", want)
		}
	}
	if strings.Contains(svc, "nava-core.") {
		t.Error("nava-worker mounts nava-core's certificate")
	}
	if !strings.Contains(svc, "    networks: [internal]\n") {
		t.Error("nava-worker is not on the internal network alone")
	}
}

// serviceBlock returns the lines of one top-level compose service.
func serviceBlock(t *testing.T, compose, name string) string {
	t.Helper()
	_, rest, ok := strings.Cut(compose, "\n  "+name+":\n")
	if !ok {
		t.Fatalf("no %s service in compose.yaml", name)
	}
	var b strings.Builder
	for _, line := range strings.Split(rest, "\n") {
		if line != "" && !strings.HasPrefix(line, "    ") {
			break
		}
		b.WriteString(line + "\n")
	}
	return b.String()
}

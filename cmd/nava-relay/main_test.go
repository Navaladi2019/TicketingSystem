package main

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
	"github.com/Navaladi2019/TicketingSystem/internal/workerguard"
)

func lookup(env map[string]string) config.LookupFunc {
	return func(k string) (string, bool) { v, ok := env[k]; return v, ok }
}

// Without Temporal or a database the relay has nothing to do: run fails at
// once, naming the variables and never echoing a value.
func TestRunNeedsTemporalAndDatabase(t *testing.T) {
	const dsn = "postgres://nava_relay:s3cret-value@pgbouncer:5432/nava"
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
		"NAVA_DATABASE_URL":           "postgres://nava_relay@127.0.0.1:1/nava",
		"NAVA_TEMPORAL_TLS_CERT_FILE": filepath.Join(dir, "secret-name.pem"),
		"NAVA_TEMPORAL_TLS_KEY_FILE":  filepath.Join(dir, "secret-name.key"),
		"NAVA_TEMPORAL_TLS_CA_FILE":   filepath.Join(dir, "ca.pem"),
	}
	err := run(t.Context(), lookup(env), &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "NAVA_TEMPORAL_TLS_CERT_FILE") || strings.Contains(err.Error(), "secret-name") {
		t.Fatalf("run = %v, want an error naming the variable, not the path", err)
	}
}

// The client is T31's: namespace nava with exactly what ConfigureClient sets
// on telemetry.NewTemporal (no second constructor).
func TestClientOptionsAreConfiguredByWorkerguard(t *testing.T) {
	tel, err := telemetry.New(t.Context(), config.Telemetry{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tel.Shutdown(context.Background()) })
	tt, err := tel.NewTemporal(slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
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
		t.Errorf("worker heartbeat interval %v, want negative (off)", o.WorkerHeartbeatInterval)
	}
}

// A signal ends serve cleanly, with no error and the listener shut down.
func TestServeStopsOnContextWithoutError(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	admin := &http.Server{Addr: "127.0.0.1:0", ReadHeaderTimeout: time.Second}
	err := serve(ctx, slog.New(slog.DiscardHandler), admin, nil, config.Config{HTTP: config.HTTP{ShutdownTimeout: time.Second}})
	if err != nil {
		t.Fatalf("serve = %v, want nil", err)
	}
	if err := admin.ListenAndServe(); err != http.ErrServerClosed {
		t.Errorf("ListenAndServe after serve = %v, want ErrServerClosed", err)
	}
}

// The admin port carries no auth, so compose never publishes it. The relay
// holds only its own certificate and role.
func TestComposeRelayService(t *testing.T) {
	b, err := os.ReadFile("../../compose.yaml")
	if err != nil {
		t.Fatal(err)
	}
	svc := serviceBlock(t, strings.ReplaceAll(string(b), "\r\n", "\n"), "nava-relay")
	for _, line := range strings.Split(svc, "\n") {
		code, _, _ := strings.Cut(line, "#")
		if strings.Contains(code, "ports:") {
			t.Errorf("nava-relay publishes a port: %s", strings.TrimSpace(line))
		}
		if strings.Contains(code, "9090") && !strings.Contains(code, "http://127.0.0.1:9090/") {
			t.Errorf("nava-relay mentions the admin port: %s", strings.TrimSpace(line))
		}
	}
	for _, want := range []string{"NAVA_TEMPORAL_TLS_CERT_FILE", "nava-relay.pem", "nava_relay:", "/readyz"} {
		if !strings.Contains(svc, want) {
			t.Errorf("nava-relay service has no %q", want)
		}
	}
	for _, other := range []string{"nava-core.", "nava-worker.", "nava_app", "nava_worker"} {
		if strings.Contains(svc, other) {
			t.Errorf("nava-relay service mentions %q", other)
		}
	}
	if !strings.Contains(svc, "    networks: [internal]\n") {
		t.Error("nava-relay is not on the internal network alone")
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

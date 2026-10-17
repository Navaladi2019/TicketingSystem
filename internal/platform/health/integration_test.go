//go:build integration

package health_test

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/health"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
)

// docker runs the docker CLI, failing the test with its output.
func docker(t *testing.T, args ...string) string {
	t.Helper()
	out, err := exec.CommandContext(t.Context(), "docker", args...).CombinedOutput()
	if err != nil {
		t.Fatalf("docker %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}

// startPostgres runs postgres:16.10 on a fixed free host port, so the address
// survives docker stop and docker start. It skips the test with no Docker.
func startPostgres(t *testing.T) (dsn, container string) {
	t.Helper()
	if err := exec.CommandContext(t.Context(), "docker", "info").Run(); err != nil {
		t.Skip("docker is not available")
	}
	l, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	_ = l.Close()
	container = docker(t, "run", "-d", "-e", "POSTGRES_PASSWORD=dev-only", "-p", fmt.Sprintf("127.0.0.1:%d:5432", port), "postgres:16.10")
	t.Cleanup(func() {
		_ = exec.CommandContext(context.WithoutCancel(t.Context()), "docker", "rm", "-f", container).Run()
	})
	return fmt.Sprintf("postgres://postgres:dev-only@127.0.0.1:%d/postgres?sslmode=disable", port), container
}

func waitFor(t *testing.T, limit time.Duration, what string, cond func() bool) time.Duration {
	t.Helper()
	start := time.Now()
	for !cond() {
		if time.Since(start) > limit {
			t.Fatalf("%s: not within %v", what, limit)
		}
		time.Sleep(100 * time.Millisecond)
	}
	return time.Since(start)
}

// TestReadyzTracksPostgresStopAndStart is the T04 timing criterion: /readyz is
// non-200 within 5 s of PostgreSQL stopping and 200 within 10 s of its return.
// Keycloak is a local stand-in on its own port, stopped mid-test.
func TestReadyzTracksPostgresStopAndStart(t *testing.T) {
	dsn, container := startPostgres(t)
	tel := newTel(t)
	log := slog.New(slog.DiscardHandler)

	pg, closePG, err := health.Postgres(t.Context(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(closePG)

	kc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))
	kcOut, err := telemetry.NewOutboundClient(tel, telemetry.OutboundOptions{Timeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	mon, err := health.NewMonitor(log, tel.MeterProvider,
		[]health.Check{pg, health.Keycloak(kc.URL, kcOut)}, health.Options{}) // production defaults
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() { mon.Run(ctx); close(done) }()
	t.Cleanup(func() { cancel(); <-done })
	srv := httptest.NewServer(telemetry.NewAdminHandler(log, tel, mon.Routes))
	t.Cleanup(srv.Close)

	// Postgres needs a while to initialise on first start.
	waitFor(t, 90*time.Second, "ready after start", func() bool { return readyz(t, srv) == http.StatusOK })

	// Keycloak stopped: degraded, still ready.
	kc.Close()
	waitFor(t, 10*time.Second, "keycloak degraded", func() bool { _, r := report(t, srv); return r.Status == health.StatusDegraded })
	if got := readyz(t, srv); got != http.StatusOK {
		t.Fatalf("/readyz with Keycloak stopped = %d, want 200", got)
	}

	docker(t, "stop", "-t", "2", container)
	took := waitFor(t, 5*time.Second, "/readyz non-200 after PostgreSQL stopped", func() bool { return readyz(t, srv) != http.StatusOK })
	t.Logf("not ready %v after docker stop returned", took)

	docker(t, "start", container)
	took = waitFor(t, 10*time.Second, "/readyz 200 after PostgreSQL returned", func() bool { return readyz(t, srv) == http.StatusOK })
	t.Logf("ready %v after docker start returned", took)
}

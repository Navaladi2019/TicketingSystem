package health_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/health"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
)

// toggle is a probe that fails while down is set.
type toggle struct{ down atomic.Bool }

func (t *toggle) probe(context.Context) error {
	if t.down.Load() {
		return errors.New("down")
	}
	return nil
}

func newTel(t *testing.T) *telemetry.Telemetry {
	t.Helper()
	cfg, err := config.Load(func(string) (string, bool) { return "", false })
	if err != nil {
		t.Fatal(err)
	}
	tel, err := telemetry.New(t.Context(), cfg.Telemetry)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tel.Shutdown(t.Context()) })
	return tel
}

// start returns the admin server serving the monitor, probing every 20ms.
func start(t *testing.T, checks ...health.Check) (*httptest.Server, *telemetry.Telemetry) {
	t.Helper()
	tel := newTel(t)
	log := slog.New(slog.DiscardHandler)
	mon, err := health.NewMonitor(log, tel.MeterProvider, checks, health.Options{Interval: 20 * time.Millisecond, Timeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() { mon.Run(ctx); close(done) }()
	t.Cleanup(func() { cancel(); <-done })
	srv := httptest.NewServer(telemetry.NewAdminHandler(log, tel, mon.Routes))
	t.Cleanup(srv.Close)
	return srv, tel
}

func get(t *testing.T, url string) (int, string) {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b)
}

func eventually(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func readyz(t *testing.T, srv *httptest.Server) int {
	code, _ := get(t, srv.URL+"/readyz")
	return code
}

func report(t *testing.T, srv *httptest.Server) (int, health.Report) {
	t.Helper()
	code, body := get(t, srv.URL+"/healthz")
	var r health.Report
	if err := json.Unmarshal([]byte(body), &r); err != nil {
		t.Fatalf("/healthz is not JSON: %q", body)
	}
	return code, r
}

func TestReadyzFollowsPostgresOnly(t *testing.T) {
	t.Parallel()
	pg, kc, tc := &toggle{}, &toggle{}, &toggle{}
	srv, _ := start(t,
		health.Check{Name: "postgres", Critical: true, Probe: pg.probe},
		health.Check{Name: "keycloak", Probe: kc.probe},
		health.Check{Name: "temporal", Probe: tc.probe})

	// Probes finish independently, so /readyz (postgres only) can turn 200
	// while keycloak or temporal is still unknown. Wait for the full report.
	eventually(t, "ready", func() bool { return readyz(t, srv) == http.StatusOK })
	eventually(t, "ok", func() bool { _, r := report(t, srv); return r.Status == health.StatusOK })
	code, r := report(t, srv)
	if code != http.StatusOK || r.Status != health.StatusOK ||
		r.Checks["postgres"] != "ok" || r.Checks["keycloak"] != "ok" || r.Checks["temporal"] != "ok" {
		t.Fatalf("healthy: %d %+v", code, r)
	}

	// Keycloak and Temporal down: degraded, still ready.
	kc.down.Store(true)
	tc.down.Store(true)
	eventually(t, "degraded", func() bool { _, r := report(t, srv); return r.Status == health.StatusDegraded })
	if got := readyz(t, srv); got != http.StatusOK {
		t.Errorf("/readyz with keycloak and temporal down = %d, want 200", got)
	}
	code, r = report(t, srv)
	if code != http.StatusOK || r.Checks["keycloak"] != "fail" || r.Checks["temporal"] != "fail" || r.Checks["postgres"] != "ok" {
		t.Errorf("degraded report: %d %+v", code, r)
	}

	// PostgreSQL down: not ready, down.
	pg.down.Store(true)
	eventually(t, "not ready", func() bool { return readyz(t, srv) == http.StatusServiceUnavailable })
	if code, r = report(t, srv); code != http.StatusServiceUnavailable || r.Status != health.StatusDown {
		t.Errorf("down report: %d %+v", code, r)
	}

	// Back up.
	pg.down.Store(false)
	kc.down.Store(false)
	tc.down.Store(false)
	eventually(t, "ready again", func() bool { return readyz(t, srv) == http.StatusOK })
	eventually(t, "ok again", func() bool { _, r := report(t, srv); return r.Status == health.StatusOK })
}

func TestNotReadyBeforeFirstProbe(t *testing.T) {
	t.Parallel()
	tel := newTel(t)
	started := make(chan struct{}, 1)
	block := make(chan struct{})
	t.Cleanup(func() { close(block) })
	mon, err := health.NewMonitor(slog.New(slog.DiscardHandler), tel.MeterProvider, []health.Check{{
		Name: "postgres", Critical: true,
		Probe: func(ctx context.Context) error {
			select {
			case started <- struct{}{}:
			default:
			}
			select {
			case <-block:
			case <-ctx.Done():
			}
			return nil
		},
	}}, health.Options{Timeout: time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() { mon.Run(ctx); close(done) }()
	t.Cleanup(func() { cancel(); <-done })
	<-started
	srv := httptest.NewServer(telemetry.NewAdminHandler(slog.New(slog.DiscardHandler), tel, mon.Routes))
	t.Cleanup(srv.Close)
	if got := readyz(t, srv); got != http.StatusServiceUnavailable {
		t.Errorf("/readyz before the first probe finished = %d, want 503", got)
	}
	if _, r := report(t, srv); r.Checks["postgres"] != health.StateUnknown {
		t.Errorf("state before the first probe = %q, want unknown", r.Checks["postgres"])
	}
}

func TestUnconfiguredDependencies(t *testing.T) {
	t.Parallel()
	unconf := func(context.Context) error { return health.ErrUnconfigured }
	srv, _ := start(t,
		health.Check{Name: "postgres", Critical: true, Probe: unconf},
		health.Check{Name: "keycloak", Probe: unconf})
	eventually(t, "unconfigured", func() bool { _, r := report(t, srv); return r.Checks["postgres"] == health.StateUnconfigured })
	if got := readyz(t, srv); got != http.StatusServiceUnavailable {
		t.Errorf("/readyz with no database configured = %d, want 503 (fail closed)", got)
	}
}

func TestSlowProbeTimesOut(t *testing.T) {
	t.Parallel()
	tel := newTel(t)
	mon, err := health.NewMonitor(slog.New(slog.DiscardHandler), tel.MeterProvider, []health.Check{{
		Name: "postgres", Critical: true,
		Probe: func(ctx context.Context) error { <-ctx.Done(); return ctx.Err() },
	}}, health.Options{Interval: 10 * time.Millisecond, Timeout: 50 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() { mon.Run(ctx); close(done) }()
	t.Cleanup(func() { cancel(); <-done })
	eventually(t, "timeout recorded as fail", func() bool { return mon.Report().Checks["postgres"] == health.StateFail })
}

func TestDegradedMetricHasOnlyDependencyAttribute(t *testing.T) {
	t.Parallel()
	kc := &toggle{}
	kc.down.Store(true)
	srv, tel := start(t,
		health.Check{Name: "postgres", Critical: true, Probe: (&toggle{}).probe},
		health.Check{Name: "keycloak", Probe: kc.probe})
	eventually(t, "keycloak degraded", func() bool { _, r := report(t, srv); return r.Checks["keycloak"] == health.StateFail })

	families, err := tel.Registry.Gather()
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]float64{}
	for _, f := range families {
		if f.GetName() != "nava_dependency_degraded" {
			continue
		}
		for _, m := range f.GetMetric() {
			if len(m.GetLabel()) != 1 || m.GetLabel()[0].GetName() != "dependency" {
				// otel_scope_* labels are the exporter's own; anything else is ours.
				for _, l := range m.GetLabel() {
					if l.GetName() != "dependency" && !strings.HasPrefix(l.GetName(), "otel_scope_") {
						t.Errorf("unexpected label %s=%s", l.GetName(), l.GetValue())
					}
				}
			}
			for _, l := range m.GetLabel() {
				if l.GetName() == "dependency" {
					got[l.GetValue()] = m.GetGauge().GetValue()
				}
			}
		}
	}
	if got["keycloak"] != 1 || got["postgres"] != 0 {
		t.Errorf("nava_dependency_degraded = %v, want keycloak 1 and postgres 0", got)
	}
}

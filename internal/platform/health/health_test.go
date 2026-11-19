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
	"testing/synctest"
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

// interval is how often start's monitor probes.
const interval = 20 * time.Millisecond

// start returns the admin server serving the monitor, probing every interval.
// It must run inside a synctest bubble: the monitor's ticker runs on the
// bubble's fake clock and the server listens on httptest's in-memory network,
// so a test moves from one probe round to the next with advance, not a poll.
func start(t *testing.T, checks ...health.Check) (*httptest.Server, *telemetry.Telemetry) {
	t.Helper()
	tel := newTel(t)
	log := slog.New(slog.DiscardHandler)
	mon, err := health.NewMonitor(log, tel.MeterProvider, checks, health.Options{Interval: interval, Timeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	run(t, mon)
	return httptest.NewTestServer(t, telemetry.NewAdminHandler(log, tel, mon.Routes)), tel
}

// run runs mon until the test ends.
func run(t *testing.T, mon *health.Monitor) {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() { mon.Run(ctx); close(done) }()
	t.Cleanup(func() { cancel(); <-done })
}

// advance moves the bubble's fake clock on by d, then waits until every
// goroutine in the bubble is blocked again, so all probe rounds due by then
// have finished. It takes no wall-clock time.
func advance(d time.Duration) {
	<-time.After(d)
	synctest.Wait()
}

func get(t *testing.T, srv *httptest.Server, path string) (int, string) {
	t.Helper()
	c := srv.Client() // sets srv.URL on first use
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.URL+path, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b)
}

func readyz(t *testing.T, srv *httptest.Server) int {
	code, _ := get(t, srv, "/readyz")
	return code
}

func report(t *testing.T, srv *httptest.Server) (int, health.Report) {
	t.Helper()
	code, body := get(t, srv, "/healthz")
	var r health.Report
	if err := json.Unmarshal([]byte(body), &r); err != nil {
		t.Fatalf("/healthz is not JSON: %q", body)
	}
	return code, r
}

func TestReadyzFollowsPostgresOnly(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		pg, kc, tc := &toggle{}, &toggle{}, &toggle{}
		srv, _ := start(t,
			health.Check{Name: "postgres", Critical: true, Probe: pg.probe},
			health.Check{Name: "keycloak", Probe: kc.probe},
			health.Check{Name: "temporal", Probe: tc.probe})

		// The first round starts at once; when the bubble is idle it is done.
		synctest.Wait()
		code, r := report(t, srv)
		if code != http.StatusOK || r.Status != health.StatusOK ||
			r.Checks["postgres"] != "ok" || r.Checks["keycloak"] != "ok" || r.Checks["temporal"] != "ok" {
			t.Fatalf("healthy: %d %+v", code, r)
		}
		if got := readyz(t, srv); got != http.StatusOK {
			t.Fatalf("/readyz when healthy = %d, want 200", got)
		}

		// Keycloak and Temporal down. The report is the last round's: it
		// changes on the next tick, not on the request.
		kc.down.Store(true)
		tc.down.Store(true)
		if _, r = report(t, srv); r.Status != health.StatusOK {
			t.Errorf("report changed before the next probe round: %+v", r)
		}
		advance(interval)
		if got := readyz(t, srv); got != http.StatusOK {
			t.Errorf("/readyz with keycloak and temporal down = %d, want 200", got)
		}
		code, r = report(t, srv)
		if code != http.StatusOK || r.Status != health.StatusDegraded ||
			r.Checks["keycloak"] != "fail" || r.Checks["temporal"] != "fail" || r.Checks["postgres"] != "ok" {
			t.Errorf("degraded report: %d %+v", code, r)
		}

		// PostgreSQL down: not ready, down, from the next round on.
		pg.down.Store(true)
		advance(interval)
		if got := readyz(t, srv); got != http.StatusServiceUnavailable {
			t.Errorf("/readyz with postgres down = %d, want 503", got)
		}
		if code, r = report(t, srv); code != http.StatusServiceUnavailable || r.Status != health.StatusDown {
			t.Errorf("down report: %d %+v", code, r)
		}

		// Back up.
		pg.down.Store(false)
		kc.down.Store(false)
		tc.down.Store(false)
		advance(interval)
		if got := readyz(t, srv); got != http.StatusOK {
			t.Errorf("/readyz after recovery = %d, want 200", got)
		}
		if code, r = report(t, srv); code != http.StatusOK || r.Status != health.StatusOK {
			t.Errorf("report after recovery: %d %+v", code, r)
		}
	})
}

func TestNotReadyBeforeFirstProbe(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		tel := newTel(t)
		block := make(chan struct{})
		t.Cleanup(func() { close(block) })
		mon, err := health.NewMonitor(slog.New(slog.DiscardHandler), tel.MeterProvider, []health.Check{{
			Name: "postgres", Critical: true,
			Probe: func(ctx context.Context) error {
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
		run(t, mon)
		srv := httptest.NewTestServer(t, telemetry.NewAdminHandler(slog.New(slog.DiscardHandler), tel, mon.Routes))
		// The probe is blocked; 59 s on, its timeout has not ended it yet.
		advance(time.Minute - time.Second)
		if got := readyz(t, srv); got != http.StatusServiceUnavailable {
			t.Errorf("/readyz before the first probe finished = %d, want 503", got)
		}
		if _, r := report(t, srv); r.Checks["postgres"] != health.StateUnknown {
			t.Errorf("state before the first probe = %q, want unknown", r.Checks["postgres"])
		}
	})
}

func TestUnconfiguredDependencies(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		unconf := func(context.Context) error { return health.ErrUnconfigured }
		srv, _ := start(t,
			health.Check{Name: "postgres", Critical: true, Probe: unconf},
			health.Check{Name: "keycloak", Probe: unconf})
		synctest.Wait()
		if _, r := report(t, srv); r.Checks["postgres"] != health.StateUnconfigured || r.Checks["keycloak"] != health.StateUnconfigured {
			t.Errorf("checks = %+v, want both unconfigured", r.Checks)
		}
		if got := readyz(t, srv); got != http.StatusServiceUnavailable {
			t.Errorf("/readyz with no database configured = %d, want 503 (fail closed)", got)
		}
	})
}

func TestSlowProbeTimesOut(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		tel := newTel(t)
		const timeout = 50 * time.Millisecond
		mon, err := health.NewMonitor(slog.New(slog.DiscardHandler), tel.MeterProvider, []health.Check{{
			Name: "postgres", Critical: true,
			Probe: func(ctx context.Context) error { <-ctx.Done(); return ctx.Err() },
		}}, health.Options{Interval: 10 * time.Millisecond, Timeout: timeout})
		if err != nil {
			t.Fatal(err)
		}
		run(t, mon)
		// A tick (10 ms) does not cut the running round short: only the
		// probe timeout does, and exactly at 50 ms.
		advance(timeout - time.Millisecond)
		if got := mon.Report().Checks["postgres"]; got != health.StateUnknown {
			t.Fatalf("state at %v = %q, want unknown until the %v timeout", timeout-time.Millisecond, got, timeout)
		}
		advance(time.Millisecond)
		if got := mon.Report().Checks["postgres"]; got != health.StateFail {
			t.Errorf("state at the timeout = %q, want fail", got)
		}
	})
}

func TestDegradedMetricHasOnlyDependencyAttribute(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		kc := &toggle{}
		kc.down.Store(true)
		srv, tel := start(t,
			health.Check{Name: "postgres", Critical: true, Probe: (&toggle{}).probe},
			health.Check{Name: "keycloak", Probe: kc.probe})
		// Postgres reports 1 while still unknown, so wait for its first probe too.
		synctest.Wait()
		if _, r := report(t, srv); r.Checks["postgres"] != health.StateOK || r.Checks["keycloak"] != health.StateFail {
			t.Fatalf("checks = %+v, want postgres ok and keycloak fail", r.Checks)
		}

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
	})
}

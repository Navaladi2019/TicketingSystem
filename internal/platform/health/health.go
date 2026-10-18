// Package health reports whether nava-core can serve. A Monitor probes each
// dependency in the background and caches the answer, so /healthz and
// /readyz are cheap and a probe storm never reaches the dependencies.
//
// Only PostgreSQL gates readiness: without it nothing works. Keycloak and
// Temporal failing is "degraded": /healthz and the nava.dependency.degraded
// metric say so, /readyz stays 200, because restarting or draining nava-core
// would not bring them back.
package health

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// Check results, as reported per dependency in /healthz.
const (
	StateOK           = "ok"
	StateFail         = "fail"
	StateUnconfigured = "unconfigured"
	StateUnknown      = "unknown" // not probed yet
)

// Overall status values of /healthz.
const (
	StatusOK       = "ok"
	StatusDegraded = "degraded"
	StatusDown     = "down"
)

// ErrUnconfigured is what a Probe returns when its dependency has no address.
// It counts as a failure, reported as "unconfigured".
var ErrUnconfigured = errors.New("health: dependency not configured")

// Check is one dependency. Probe must honour ctx.
type Check struct {
	// Name is the dependency attribute of the metric: postgres, keycloak or
	// temporal. Nothing else is ever an attribute value.
	Name string
	// Critical checks gate /readyz. The others only degrade /healthz.
	Critical bool
	Probe    func(ctx context.Context) error
}

// Options tune a Monitor. Zero fields take the defaults.
type Options struct {
	// Interval between probe rounds. Default 1s.
	Interval time.Duration
	// Timeout for one probe. Default 2s. Interval+Timeout bounds how long
	// an outage or a recovery takes to show, so it must stay well under the
	// 5 s (down) and 10 s (up) readiness targets.
	Timeout time.Duration
}

// Monitor runs the checks and serves their cached results.
type Monitor struct {
	checks   []Check
	interval time.Duration
	timeout  time.Duration
	log      *slog.Logger

	mu    sync.RWMutex
	state map[string]string
}

// NewMonitor registers the metric on mp and returns a Monitor. Call Run to
// start probing; until a check has answered once its state is "unknown", which
// counts as failed.
func NewMonitor(log *slog.Logger, mp metric.MeterProvider, checks []Check, opts Options) (*Monitor, error) {
	if opts.Interval <= 0 {
		opts.Interval = time.Second
	}
	if opts.Timeout <= 0 {
		opts.Timeout = 2 * time.Second
	}
	m := &Monitor{
		checks:   checks,
		interval: opts.Interval,
		timeout:  opts.Timeout,
		log:      log,
		state:    make(map[string]string, len(checks)),
	}
	for _, c := range checks {
		m.state[c.Name] = StateUnknown
	}
	gauge, err := mp.Meter("nava-core/health").Int64ObservableGauge("nava.dependency.degraded",
		metric.WithDescription("1 while a dependency's check is failing, else 0."))
	if err != nil {
		return nil, err
	}
	_, err = mp.Meter("nava-core/health").RegisterCallback(func(_ context.Context, o metric.Observer) error {
		m.mu.RLock()
		defer m.mu.RUnlock()
		for _, c := range m.checks {
			v := int64(1)
			if m.state[c.Name] == StateOK {
				v = 0
			}
			o.ObserveInt64(gauge, v, metric.WithAttributes(attribute.String("dependency", c.Name)))
		}
		return nil
	}, gauge)
	if err != nil {
		return nil, err
	}
	return m, nil
}

// Run probes every Interval until ctx is cancelled. The first round starts
// at once.
func (m *Monitor) Run(ctx context.Context) {
	t := time.NewTicker(m.interval)
	defer t.Stop()
	for {
		m.round(ctx)
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

// round probes all checks concurrently and stores the results.
func (m *Monitor) round(ctx context.Context) {
	var wg sync.WaitGroup
	for _, c := range m.checks {
		wg.Go(func() {
			pctx, cancel := context.WithTimeout(ctx, m.timeout)
			defer cancel()
			next := StateOK
			err := c.Probe(pctx)
			if ctx.Err() != nil {
				return // shutting down: the result says nothing about the dependency
			}
			switch {
			case errors.Is(err, ErrUnconfigured):
				next = StateUnconfigured
			case err != nil:
				next = StateFail
			}
			m.mu.Lock()
			prev := m.state[c.Name]
			m.state[c.Name] = next
			m.mu.Unlock()
			if prev != next {
				m.logChange(ctx, c.Name, prev, next, err)
			}
		})
	}
	wg.Wait()
}

// logChange logs a state transition once, not every round. The probe error is
// logged: pgx, net and gRPC errors name an address, never a credential.
func (m *Monitor) logChange(ctx context.Context, name, prev, next string, err error) {
	level := slog.LevelInfo
	if next != StateOK {
		level = slog.LevelWarn
	}
	attrs := []slog.Attr{slog.String("dependency", name), slog.String("from", prev), slog.String("to", next)}
	if err != nil {
		attrs = append(attrs, slog.Any("error", err))
	}
	m.log.LogAttrs(ctx, level, "dependency state changed", attrs...)
}

// Report is the /healthz body.
type Report struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks"`
}

// Report returns the current status. Down means a critical check is not OK,
// degraded that only non-critical ones are not.
func (m *Monitor) Report() Report {
	m.mu.RLock()
	defer m.mu.RUnlock()
	r := Report{Status: StatusOK, Checks: make(map[string]string, len(m.checks))}
	for _, c := range m.checks {
		s := m.state[c.Name]
		r.Checks[c.Name] = s
		switch {
		case s == StateOK:
		case c.Critical:
			r.Status = StatusDown
		case r.Status == StatusOK:
			r.Status = StatusDegraded
		}
	}
	return r
}

// Routes registers GET /healthz and GET /readyz for telemetry.NewAdminHandler.
// /healthz is 200 unless a critical dependency is down (503); its body names
// every dependency. /readyz is 200 or 503 on the critical checks alone, with no
// body detail.
func (m *Monitor) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		r := m.Report()
		code := http.StatusOK
		if r.Status == StatusDown {
			code = http.StatusServiceUnavailable
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(code)
		_ = json.NewEncoder(w).Encode(r)
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		if m.Report().Status == StatusDown {
			http.Error(w, "not ready", http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte("ok\n"))
	})
}

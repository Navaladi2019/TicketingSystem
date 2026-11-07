package lineage

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"go.opentelemetry.io/otel/metric"
)

// DeadLineagesMetric is the gauge; the Prometheus exporter names it
// audit_dead_lineages. It carries no tenant label (TICK-93): the tenants are
// on the "audit seal lineage dead" log line.
const DeadLineagesMetric = "audit.dead_lineages"

// DefaultInterval is how often a Monitor reads the heads. A dead lineage
// stays dead until chain:reseed, so a minute adds little to R17's 10 min.
const DefaultInterval = time.Minute

// Monitor counts dead lineages (R17) on a ticker and exports the count as
// DeadLineagesMetric. A failed read is logged and withdraws the value, so the
// gauge never reports a stale 0.
type Monitor struct {
	db       Querier
	log      *slog.Logger
	interval time.Duration

	mu    sync.Mutex
	count int64
	ok    bool
}

// NewMonitor registers the gauge on mp. interval <= 0 means DefaultInterval.
func NewMonitor(db Querier, mp metric.MeterProvider, log *slog.Logger, interval time.Duration) (*Monitor, error) {
	if interval <= 0 {
		interval = DefaultInterval
	}
	m := &Monitor{db: db, log: log, interval: interval}
	meter := mp.Meter("github.com/Navaladi2019/TicketingSystem/internal/audit/lineage")
	_, err := meter.Int64ObservableGauge(DeadLineagesMetric,
		metric.WithDescription("Audit seal lineages whose next run is more than 10 min overdue, the chain:reseed predicate (ADR-004 R17)."),
		metric.WithInt64Callback(func(_ context.Context, o metric.Int64Observer) error {
			if n, ok := m.Value(); ok {
				o.Observe(n)
			}
			return nil
		}))
	if err != nil {
		return nil, fmt.Errorf("lineage: %s: %w", DeadLineagesMetric, err)
	}
	return m, nil
}

// Value is the last count; ok is false before the first successful read and
// after a failed one.
func (m *Monitor) Value() (n int64, ok bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.count, m.ok
}

// Once reads the dead lineages, logs each tenant, and updates the gauge.
func (m *Monitor) Once(ctx context.Context) error {
	dead, err := DeadLineages(ctx, m.db)
	m.mu.Lock()
	m.count, m.ok = int64(len(dead)), err == nil
	m.mu.Unlock()
	if err != nil {
		m.log.ErrorContext(ctx, "audit dead lineage check failed", slog.Any("error", err))
		return err
	}
	for _, d := range dead {
		m.log.WarnContext(ctx, "audit seal lineage dead",
			slog.String("tenant_id", d.TenantID.String()),
			slog.Time("next_seal_at", d.NextSealAt))
	}
	return nil
}

// Run checks once at once, then every interval, until ctx is cancelled.
func (m *Monitor) Run(ctx context.Context) error {
	t := time.NewTicker(m.interval)
	defer t.Stop()
	for {
		_ = m.Once(ctx)
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
		}
	}
}

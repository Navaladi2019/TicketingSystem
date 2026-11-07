package seal

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
	"github.com/Navaladi2019/TicketingSystem/internal/workerguard"
)

// The R-A2 metrics. The Prometheus exporter adds the unit and _total
// suffixes: audit_unsealed_oldest_seconds and audit_seal_failures_total.
// Neither carries a tenant (TICK-93): the tenant goes on the log line.
const (
	UnsealedOldestMetric = "audit.unsealed.oldest"
	SealFailuresMetric   = "audit.seal.failures"
)

// The reason attribute of SealFailuresMetric.
const (
	ReasonUnsealable = "unsealable" // a poison entry (ErrUnsealable)
	ReasonError      = "error"      // anything else; Temporal retries it
)

// LagStaleAfter drops a tenant from the gauge when no A2 has measured it for
// this long. A live lineage measures every MaxIntervalSeconds or sooner,
// poison or not, so only a tenant whose runs moved to another worker, or
// stopped, ages out. A dead lineage is R17's (next_seal_at), not this gauge's.
const LagStaleAfter = 3 * MaxIntervalSeconds * time.Second

// Metrics are the sealer's metrics on one worker. A nil *Metrics records
// nothing.
type Metrics struct {
	failures metric.Int64Counter
	now      func() time.Time

	mu     sync.Mutex
	oldest map[uuid.UUID]lagSample
}

// lagSample is one tenant's oldest unsealed occurred_at, as A2 last measured
// it, and when.
type lagSample struct {
	oldest, measured time.Time
}

// NewMetrics registers the sealer metrics on mp.
func NewMetrics(mp metric.MeterProvider) (*Metrics, error) {
	return newMetrics(mp, time.Now)
}

func newMetrics(mp metric.MeterProvider, now func() time.Time) (*Metrics, error) {
	m := &Metrics{now: now, oldest: map[uuid.UUID]lagSample{}}
	meter := mp.Meter("github.com/Navaladi2019/TicketingSystem/internal/audit/seal")
	var err error
	m.failures, err = meter.Int64Counter(SealFailuresMetric,
		metric.WithDescription("Audit sealer A2 activity attempts that failed, by reason (ADR-004 R-A2)."))
	if err != nil {
		return nil, fmt.Errorf("seal: %s: %w", SealFailuresMetric, err)
	}
	_, err = meter.Float64ObservableGauge(UnsealedOldestMetric,
		metric.WithUnit("s"),
		metric.WithDescription("Age of the oldest unsealed audit entry over the tenants this worker sealed (ADR-004 R-A2)."),
		metric.WithFloat64Callback(func(_ context.Context, o metric.Float64Observer) error {
			o.Observe(m.oldestSeconds())
			return nil
		}))
	if err != nil {
		return nil, fmt.Errorf("seal: %s: %w", UnsealedOldestMetric, err)
	}
	return m, nil
}

// failed counts one failed A2 attempt.
func (m *Metrics) failed(ctx context.Context, err error) {
	if m == nil {
		return
	}
	reason := ReasonError
	if errors.Is(err, ErrUnsealable) {
		reason = ReasonUnsealable
	}
	m.failures.Add(ctx, 1, metric.WithAttributes(attribute.String("reason", reason)))
}

// observe records tenant's oldest unsealed entry; ok false means none.
func (m *Metrics) observe(tenant uuid.UUID, oldest time.Time, ok bool) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if !ok {
		delete(m.oldest, tenant)
		return
	}
	m.oldest[tenant] = lagSample{oldest: oldest, measured: m.now()}
}

// oldestSeconds is the gauge: the largest age, now, of any tenant's oldest
// unsealed entry, 0 with none. It keeps growing between measurements, so a
// tenant whose poison entry blocks sealing crosses the alert threshold even
// though each run fails. Samples older than LagStaleAfter are dropped.
func (m *Metrics) oldestSeconds() float64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := m.now()
	var worst time.Duration
	for tenant, s := range m.oldest {
		if now.Sub(s.measured) > LagStaleAfter {
			delete(m.oldest, tenant)
			continue
		}
		worst = max(worst, now.Sub(s.oldest))
	}
	return worst.Seconds()
}

// oldestUnsealedSQL goes through the audit_log_unsealed partial index.
const oldestUnsealedSQL = `SELECT min(occurred_at) FROM audit_log WHERE tenant_id = $1 AND sealed_at IS NULL`

// OldestUnsealed returns the occurred_at of the tenant's oldest unsealed
// entry; ok is false when every entry is sealed.
func OldestUnsealed(ctx context.Context, pool *pgxpool.Pool) (oldest time.Time, ok bool, err error) {
	tenant, found := workerguard.TenantFromContext(ctx)
	if !found {
		return time.Time{}, false, tenancy.ErrNoTenant
	}
	var t *time.Time
	err = workerguard.InTenantTx(ctx, pool, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, oldestUnsealedSQL, tenant).Scan(&t)
	})
	if err != nil {
		return time.Time{}, false, fmt.Errorf("seal: read oldest unsealed entry: %w", err)
	}
	if t == nil {
		return time.Time{}, false, nil
	}
	return *t, true, nil
}

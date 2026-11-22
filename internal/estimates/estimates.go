// Package estimates keeps entity_record_estimates current (TICK-1071, S31;
// ADR-010-A2 A2-9, A2-10, PLAN-011-A2 D-11A2-2): one row per tenant and entity
// type holding count(*) of that tenant's records of the type. A ticker loop in
// nava-worker, connected as nava_worker, runs one cycle per interval.
//
// A cycle lists the tenants, then refreshes each in its own transaction under
// that tenant's app.tenant_id, so RLS scopes every read and write. The listing
// is the only statement that runs before a tenant is set, by design. A missing
// or stale estimate costs time, never rows (A2-6), so the job fails soft: a
// failed tenant is counted and retried next cycle. The job is never called
// from the record write path.
//
// records has no soft-delete column on this branch, so "live" is count(*), of
// the rows that name a type (the bare T09 seed rows do not).
// A type left with no records has its row set to 0 by a second statement in the
// same transaction.
package estimates

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel/metric"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
)

// The metrics (A2-10). Neither carries a tenant, user or entity attribute.
// With the Prometheus exporter's suffixes they are
// entity_record_estimates_refresh_failures_total and
// authz_under_estimate_age_seconds.
const (
	FailuresMetric = "entity_record_estimates.refresh_failures"
	AgeMetric      = "authz_under_estimate_age"
)

// IntervalEnv sets the cycle interval, such as 1h. "0" disables the job.
const IntervalEnv = "NAVA_ESTIMATES_REFRESH_INTERVAL"

// DefaultInterval is the cycle interval when IntervalEnv is unset.
const DefaultInterval = time.Hour

// The statements. listSQL runs before any tenant is set; the others run in a
// tenant's transaction after setTenantSQL, with the same $1.
const (
	listSQL      = `SELECT DISTINCT tenant_id FROM tenant_identity_binding`
	setTenantSQL = `SELECT set_config('app.tenant_id', $1, true)`
	refreshSQL   = `INSERT INTO entity_record_estimates (tenant_id, entity_definition_id, live_records, refreshed_at)
		SELECT $1::uuid, entity_definition_id, count(*), now() FROM records WHERE tenant_id = $1::uuid AND entity_definition_id IS NOT NULL GROUP BY entity_definition_id
		ON CONFLICT (tenant_id, entity_definition_id)
		DO UPDATE SET live_records = EXCLUDED.live_records, refreshed_at = EXCLUDED.refreshed_at`
	zeroSQL = `UPDATE entity_record_estimates e SET live_records = 0, refreshed_at = now()
		WHERE e.tenant_id = $1::uuid AND NOT EXISTS (
		    SELECT 1 FROM records r WHERE r.tenant_id = $1::uuid AND r.entity_definition_id = e.entity_definition_id)`
)

// Beginner starts a transaction. *pgxpool.Pool satisfies it.
type Beginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

// Options tune a Refresher.
type Options struct {
	Interval time.Duration // zero or negative: the job is off, Run returns at once
	Now      func() time.Time
}

// Refresher runs the refresh cycles. Build it with New.
type Refresher struct {
	db       Beginner
	log      *slog.Logger
	interval time.Duration
	now      func() time.Time
	failures metric.Int64Counter

	mu      sync.Mutex
	started time.Time
	written map[uuid.UUID]time.Time // tenant -> when its rows were last refreshed
}

// New registers the two metrics on mp.
func New(db Beginner, mp metric.MeterProvider, log *slog.Logger, opts Options) (*Refresher, error) {
	if opts.Now == nil {
		opts.Now = time.Now
	}
	r := &Refresher{db: db, log: log, interval: opts.Interval, now: opts.Now, written: map[uuid.UUID]time.Time{}}
	r.started = r.now()
	meter := mp.Meter("github.com/Navaladi2019/TicketingSystem/internal/estimates")
	var err error
	if r.failures, err = meter.Int64Counter(FailuresMetric,
		metric.WithDescription("Tenants whose estimate refresh transaction failed (ADR-010-A2 A2-10). No tenant label.")); err != nil {
		return nil, fmt.Errorf("estimates: %s: %w", FailuresMetric, err)
	}
	if _, err = meter.Float64ObservableGauge(AgeMetric, metric.WithUnit("s"),
		metric.WithDescription("Now minus the oldest refresh time over all tenants (ADR-010-A2 A2-10). No label."),
		metric.WithFloat64Callback(func(_ context.Context, o metric.Float64Observer) error {
			o.Observe(r.Age())
			return nil
		})); err != nil {
		return nil, fmt.Errorf("estimates: %s: %w", AgeMetric, err)
	}
	return r, nil
}

// Age is now minus the oldest time any tenant's estimates were refreshed, in
// seconds. A tenant that fails keeps its old time, so a persistent failure ages
// the gauge; a tenant never refreshed counts from the process start.
func (r *Refresher) Age() float64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := r.now()
	oldest := r.started
	if len(r.written) > 0 {
		oldest = now
		for _, t := range r.written {
			if t.Before(oldest) {
				oldest = t
			}
		}
	}
	return now.Sub(oldest).Seconds()
}

// Run cycles once at start, then every interval, until ctx is done. A
// non-positive interval disables it.
func (r *Refresher) Run(ctx context.Context) {
	if r.interval <= 0 {
		return
	}
	t := time.NewTicker(r.interval)
	defer t.Stop()
	for {
		r.Cycle(ctx)
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

// Cycle refreshes every tenant and returns how many tenants failed. A failure
// is counted and logged by tenant id; the others are still refreshed.
func (r *Refresher) Cycle(ctx context.Context) int {
	tenants, err := r.tenants(ctx)
	if err != nil {
		r.log.WarnContext(ctx, "estimates: cannot list tenants", slog.Any("error", err))
		return 0
	}
	r.mu.Lock()
	known := make(map[uuid.UUID]time.Time, len(tenants))
	for _, t := range tenants {
		if at, ok := r.written[t]; ok {
			known[t] = at
		} else {
			known[t] = r.started
		}
	}
	r.written = known
	r.mu.Unlock()

	failed := 0
	for _, t := range tenants {
		if ctx.Err() != nil {
			break
		}
		if err := r.refresh(ctx, t); err != nil {
			failed++
			r.failures.Add(ctx, 1)
			r.log.WarnContext(ctx, "estimates: tenant refresh failed, retried next cycle",
				slog.String("tenant_id", t.String()), slog.Any("error", err))
			continue
		}
		r.mu.Lock()
		r.written[t] = r.now()
		r.mu.Unlock()
	}
	return failed
}

// tenants lists the tenants: the one statement that runs before a tenant is
// set. DISTINCT, since a tenant can have several bindings.
func (r *Refresher) tenants(ctx context.Context) (_ []uuid.UUID, err error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rollback(ctx, tx)) }()
	rows, err := tx.Query(ctx, listSQL)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
}

// refresh runs one tenant's transaction: set the tenant, then upsert.
func (r *Refresher) refresh(ctx context.Context, tenant uuid.UUID) (err error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, rollback(ctx, tx)) }()
	id := tenant.String()
	for _, sql := range []string{setTenantSQL, refreshSQL, zeroSQL} {
		if _, err := tx.Exec(ctx, sql, id); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// rollback ends tx unless it committed.
func rollback(ctx context.Context, tx pgx.Tx) error {
	if err := tx.Rollback(context.WithoutCancel(ctx)); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
		return err
	}
	return nil
}

// IntervalFromEnv reads IntervalEnv through lookup: unset is DefaultInterval,
// "0" is zero (disabled), anything else must be a positive duration.
func IntervalFromEnv(lookup config.LookupFunc) (time.Duration, error) {
	v, ok := lookup(IntervalEnv)
	if !ok || v == "" {
		return DefaultInterval, nil
	}
	if v == "0" {
		return 0, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil || d <= 0 {
		return 0, errors.New("config: " + IntervalEnv + " must be a positive duration such as 1h, or 0 to disable")
	}
	return d, nil
}

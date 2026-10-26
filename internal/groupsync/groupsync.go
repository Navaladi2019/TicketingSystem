// Package groupsync mirrors each tenant's users, federated links, groups,
// memberships and org units from Keycloak into the identity tables (T16),
// connected as nava_groupsync (ADR-001 A.3, PLAN-001 T25, D-P9, D-P13).
//
// One cycle visits every tenant with an identity binding. Per tenant it opens
// one transaction (tenancy.InTenantTx), takes the tenant's transaction-scoped
// advisory lock so two replicas never sync one tenant at once, reads the
// directory snapshot and applies the difference. A failing tenant rolls back
// alone and does not stop the others.
//
// The database cannot tell which Keycloak path a row came from, so this
// package enforces what it cannot: only paths under /t/<tenant_id>/ are
// written to groups, only paths under /t/<tenant_id>/ou/ to org_units, and an
// org-unit parent change that would form a cycle is refused.
package groupsync

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/metric"

	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// LagWarnAfter is the lag past which a tenant is named in a log line (D-P9).
const LagWarnAfter = 10 * time.Minute

// DefaultInterval is the time between cycles (D-P9).
const DefaultInterval = 5 * time.Minute

// Options tune a Syncer. Zero fields take the defaults.
type Options struct {
	// Interval between cycles in Run. Default DefaultInterval.
	Interval time.Duration
	// TenantTimeout bounds one tenant's transaction, snapshot included.
	// Default 2 minutes.
	TenantTimeout time.Duration
	// MaxOrgUnitDepth bounds every walk of the org-unit tree. Default
	// DefaultMaxOrgUnitDepth.
	MaxOrgUnitDepth int
	// Now defaults to time.Now.
	Now func() time.Time
}

// Result is what one cycle did to one tenant.
type Result struct {
	TenantID uuid.UUID
	// Skipped is true when another process held the tenant's lock.
	Skipped bool
	// Unchanged is true when the snapshot matched the last applied one.
	Unchanged bool
	Err       error
}

// Syncer runs the sync. Build it with New.
type Syncer struct {
	pool *pgxpool.Pool
	dir  identity.DirectoryAdmin
	log  *slog.Logger
	opts Options
	lag  metric.Float64Histogram

	mu      sync.Mutex
	cursors map[uuid.UUID]string
	lastOK  map[uuid.UUID]time.Time
	started time.Time
}

// New registers groupsync_lag_seconds on mp. pool must log in as
// nava_groupsync.
func New(pool *pgxpool.Pool, dir identity.DirectoryAdmin, mp metric.MeterProvider, log *slog.Logger, opts Options) (*Syncer, error) {
	if pool == nil || dir == nil {
		return nil, errors.New("groupsync: a pool and a directory are required")
	}
	if opts.Interval <= 0 {
		opts.Interval = DefaultInterval
	}
	if opts.TenantTimeout <= 0 {
		opts.TenantTimeout = 2 * time.Minute
	}
	if opts.MaxOrgUnitDepth <= 0 {
		opts.MaxOrgUnitDepth = DefaultMaxOrgUnitDepth
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	// One observation per tenant per cycle and no tenant attribute: a tenant
	// id is never a metric attribute (T01). A tenant over the lag limit is
	// named in a log line instead.
	h, err := mp.Meter("github.com/Navaladi2019/TicketingSystem/internal/groupsync").Float64Histogram(
		"groupsync_lag_seconds",
		metric.WithDescription("Seconds since the last successful sync of a tenant, observed once per tenant per cycle. Never labelled by tenant."),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(1, 5, 15, 30, 60, 120, 300, 600, 1800, 3600))
	if err != nil {
		return nil, fmt.Errorf("groupsync: %w", err)
	}
	return &Syncer{
		pool: pool, dir: dir, log: log, opts: opts, lag: h,
		cursors: map[uuid.UUID]string{}, lastOK: map[uuid.UUID]time.Time{}, started: opts.Now(),
	}, nil
}

// tenantsSQL runs with no tenant set: binding_lookup lets nava_groupsync read
// every binding.
const tenantsSQL = `SELECT DISTINCT tenant_id FROM tenant_identity_binding ORDER BY tenant_id`

// Tenants lists the tenants that have an identity binding.
func (s *Syncer) Tenants(ctx context.Context) ([]uuid.UUID, error) {
	rows, err := s.pool.Query(ctx, tenantsSQL)
	if err != nil {
		return nil, fmt.Errorf("groupsync: list tenants: %w", err)
	}
	defer rows.Close()
	var out []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("groupsync: list tenants: %w", err)
		}
		out = append(out, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("groupsync: list tenants: %w", err)
	}
	return out, nil
}

// Once runs one cycle over every tenant and returns one Result each. A
// tenant's failure is in its Result and the log; it does not stop the rest.
func (s *Syncer) Once(ctx context.Context) ([]Result, error) {
	tenants, err := s.Tenants(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Result, 0, len(tenants))
	for _, id := range tenants {
		if ctx.Err() != nil {
			return out, ctx.Err()
		}
		r := s.syncTenant(ctx, id)
		s.observe(ctx, r)
		out = append(out, r)
	}
	return out, nil
}

// Run syncs once at once, then every Interval, until ctx is cancelled. A
// failed cycle is logged and the next one still runs.
func (s *Syncer) Run(ctx context.Context) error {
	t := time.NewTicker(s.opts.Interval)
	defer t.Stop()
	for {
		if _, err := s.Once(ctx); err != nil && ctx.Err() == nil {
			s.log.ErrorContext(ctx, "group sync cycle failed", slog.Any("error", err))
		}
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
		}
	}
}

// lockSQL is transaction-scoped, so it is released with the transaction and
// is safe through PgBouncer's transaction mode.
const lockSQL = `SELECT pg_try_advisory_xact_lock(hashtextextended($1, 0))`

// LockName is the text whose hash is the advisory lock key of a tenant.
func LockName(tenantID uuid.UUID) string { return "nava-groupsync:" + tenantID.String() }

func (s *Syncer) syncTenant(ctx context.Context, tenantID uuid.UUID) Result {
	res := Result{TenantID: tenantID}
	ctx, cancel := context.WithTimeout(ctx, s.opts.TenantTimeout)
	defer cancel()
	s.mu.Lock()
	cursor := s.cursors[tenantID]
	s.mu.Unlock()

	var digest string
	var snapAt time.Time
	err := tenancy.InTenantTx(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var locked bool
		if err := tx.QueryRow(ctx, lockSQL, LockName(tenantID)).Scan(&locked); err != nil {
			return fmt.Errorf("advisory lock: %w", err)
		}
		if !locked {
			res.Skipped = true
			return nil
		}
		// The snapshot is read under the lock, so a process that waited out
		// another's sync never applies an older snapshot over a newer one.
		snapAt = s.opts.Now()
		changes, d, err := s.dir.ChangesSince(ctx, tenantID, cursor)
		if err != nil {
			return fmt.Errorf("directory snapshot: %w", err)
		}
		digest = d
		if changes.Unchanged {
			res.Unchanged = true
			return nil
		}
		st, err := apply(ctx, tx, tenantID, changes, s.opts.Now(), s.opts.MaxOrgUnitDepth)
		if err != nil {
			return err
		}
		s.log.InfoContext(ctx, "group sync applied", slog.String("tenant_id", tenantID.String()),
			slog.Int("principals_written", st.PrincipalsWritten), slog.Int("principals_deleted", st.PrincipalsDeleted),
			slog.Int("groups_written", st.GroupsWritten), slog.Int("groups_deleted", st.GroupsDeleted),
			slog.Int("org_units_written", st.OrgUnitsWritten), slog.Int("org_units_deleted", st.OrgUnitsDeleted),
			slog.Int("ignored_outside_subtree", st.Ignored))
		return nil
	})
	if err != nil {
		res.Err = err
		return res
	}
	if res.Skipped {
		return res
	}
	s.mu.Lock()
	s.cursors[tenantID] = digest
	s.lastOK[tenantID] = snapAt
	s.mu.Unlock()
	return res
}

// observe records the tenant's lag and logs a failure, or a lag past
// LagWarnAfter, with the tenant id.
func (s *Syncer) observe(ctx context.Context, r Result) {
	tid := slog.String("tenant_id", r.TenantID.String())
	if r.Err != nil {
		s.log.ErrorContext(ctx, "group sync failed; tenant rolled back", tid, slog.Any("error", r.Err))
	}
	if r.Skipped {
		s.log.InfoContext(ctx, "group sync skipped: another process holds the tenant", tid)
		return
	}
	s.mu.Lock()
	last, ok := s.lastOK[r.TenantID]
	s.mu.Unlock()
	if !ok {
		last = s.started
	}
	lag := max(s.opts.Now().Sub(last), 0)
	s.lag.Record(ctx, lag.Seconds())
	if lag > LagWarnAfter {
		s.log.WarnContext(ctx, "group sync lag over 10 minutes", tid, slog.Float64("lag_seconds", lag.Seconds()))
	}
}

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
	// subtree is groupsync_org_subtree_max, one observation per applied tenant.
	subtree metric.Int64Histogram
	// unrooted is groupsync_org_closure_unrooted_units (TICK-1058).
	unrooted metric.Int64Counter
	// drift is groupsync_org_closure_drift_repairs (TICK-1124).
	drift metric.Int64Counter
	// unitsLive is groupsync_org_units_live_max: set once per cycle to the
	// largest live org-unit count over the tenants the cycle visited. It never
	// gates a sync (ADR-010-A2 A2-7); the alert is the only reaction.
	unitsLive metric.Int64Gauge

	mu      sync.Mutex
	cursors map[uuid.UUID]string
	lastOK  map[uuid.UUID]time.Time
	// liveUnits is each tenant's last counted live org units; a tenant not
	// re-counted in a cycle (lock held elsewhere, failed apply) keeps it.
	liveUnits map[uuid.UUID]int
	// liveMaxTenant and liveMaxCount are the pair last logged by recordLiveMax,
	// so an unchanged pair is not logged every cycle.
	liveMaxTenant uuid.UUID
	liveMaxCount  int
	// countLive counts a tenant's live org units; a test swaps it to fail.
	countLive func(context.Context, pgx.Tx, uuid.UUID) (int, error)
	started   time.Time
}

// New registers groupsync_lag_seconds, groupsync_org_subtree_max and
// groupsync_org_closure_unrooted_units on mp. pool must log in as
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
	sub, err := mp.Meter("github.com/Navaladi2019/TicketingSystem/internal/groupsync").Int64Histogram(
		"groupsync_org_subtree_max",
		metric.WithDescription("Largest number of live org units under any one unit a user belongs to, observed once per applied tenant sync. Never labelled by tenant."),
		metric.WithUnit("{unit}"),
		metric.WithExplicitBucketBoundaries(10, 100, 500, 1000, 2500, 5000, 7500, 10000))
	if err != nil {
		return nil, fmt.Errorf("groupsync: %w", err)
	}
	// TICK-1058: reported, not failed. Failing the sync would not repair the
	// closure (the rollback keeps the stored one, still without the self row)
	// and would stop the tenant's principal tombstones and group removals.
	unr, unrErr := mp.Meter("github.com/Navaladi2019/TicketingSystem/internal/groupsync").Int64Counter(
		"groupsync_org_closure_unrooted_units",
		metric.WithDescription("Live org units found with no org_unit_closure self row after a tenant's closure rebuild, summed over applied tenant syncs. Such a unit is outside every region. Any increase is an alert. Never labelled by tenant; the tenant is named in a log line."),
		metric.WithUnit("{unit}"))
	dr, drErr := mp.Meter("github.com/Navaladi2019/TicketingSystem/internal/groupsync").Int64Counter(
		"groupsync_org_closure_drift_repairs",
		metric.WithDescription("Times a tenant's org_unit_closure differed from the live org tree and was repaired, summed over applied tenant syncs. A first import and a delta are not drift. Any increase is an alert (AR-TICK-1118-01). Never labelled by tenant; the tenant is named in a log line."),
		metric.WithUnit("{repair}"))
	live, liveErr := mp.Meter("github.com/Navaladi2019/TicketingSystem/internal/groupsync").Int64Gauge(
		"groupsync_org_units_live_max",
		metric.WithDescription("Largest live org-unit count of any tenant visited in the cycle, set once per cycle. Never labelled by tenant. Alerted above the ADR-010-A2 certified tree (rules/groupsync.yml); it never gates a sync."),
		metric.WithUnit("{unit}"))
	if err := errors.Join(unrErr, drErr, liveErr); err != nil {
		return nil, fmt.Errorf("groupsync: %w", err)
	}
	return &Syncer{
		pool: pool, dir: dir, log: log, opts: opts, lag: h, subtree: sub, unrooted: unr, drift: dr, unitsLive: live,
		cursors: map[uuid.UUID]string{}, lastOK: map[uuid.UUID]time.Time{}, liveUnits: map[uuid.UUID]int{}, countLive: countLiveOrgUnits, started: opts.Now(),
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
	s.recordLiveMax(ctx, tenants)
	return out, nil
}

// recordLiveMax sets groupsync_org_units_live_max to the largest last-known
// live org-unit count over the tenants this cycle visited, counted or not.
// The metric carries no tenant (T01), so when the largest pair (tenant, count)
// changes it is logged at INFO: that line names the tenant behind the
// GroupsyncOrgUnitsAboveCertified alert. No threshold is applied here; the
// log never fails a sync.
func (s *Syncer) recordLiveMax(ctx context.Context, visited []uuid.UUID) {
	s.mu.Lock()
	m, seen := 0, false
	var who uuid.UUID
	for _, id := range visited {
		if n, ok := s.liveUnits[id]; ok && (!seen || n > m) {
			m, who, seen = n, id, true
		}
	}
	changed := seen && (who != s.liveMaxTenant || m != s.liveMaxCount)
	if changed {
		s.liveMaxTenant, s.liveMaxCount = who, m
	}
	s.mu.Unlock()
	if seen {
		s.unitsLive.Record(ctx, int64(m))
	}
	if changed {
		s.log.InfoContext(ctx, "largest live org-unit tree",
			slog.String("tenant_id", who.String()), slog.Int("live_org_units", m))
	}
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

// countLiveOrgUnits counts the tenant's live org units inside its apply
// transaction, so no extra cross-tenant read is made.
func countLiveOrgUnits(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID) (int, error) {
	var n int
	err := tx.QueryRow(ctx, `SELECT count(*) FROM org_units WHERE tenant_id = $1::uuid AND deleted_at IS NULL`, tenantID.String()).Scan(&n)
	return n, err // the caller logs it; the count never fails a sync
}

// liveUnitsBestEffort counts in a savepoint. The count is a metric: if it
// fails, the savepoint is rolled back (so the tenant's transaction is not
// aborted and still commits its tombstones and removals), a warning names the
// tenant, and -1 is returned so the tenant keeps its last known count
// (ADR-010-A2 A2-7: nothing refuses a sync on this value).
func (s *Syncer) liveUnitsBestEffort(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID) int {
	sp, err := tx.Begin(ctx)
	if err == nil {
		var n int
		if n, err = s.countLive(ctx, sp, tenantID); err == nil {
			if err = sp.Commit(ctx); err == nil {
				return n
			}
		} else {
			_ = sp.Rollback(ctx)
		}
	}
	s.log.WarnContext(ctx, "group sync: counting live org units failed; the tenant keeps its last known count",
		slog.String("tenant_id", tenantID.String()), slog.Any("error", err))
	return -1
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
	// The cursor is this process's own: it skips the apply only when the
	// directory is unchanged since this process last applied it. With two
	// replicas, one may apply a snapshot, the other a later one, and the
	// directory then return to the first; this process sees "unchanged" and
	// skips until the other replica's next turn corrects the tables. Bounded
	// by one interval, and only while this process keeps losing the lock.
	s.mu.Lock()
	cursor := s.cursors[tenantID]
	s.mu.Unlock()

	var digest string
	var snapAt time.Time
	subtreeMax := -1 // -1: nothing applied, nothing to record
	unrooted := 0
	driftRepaired := false
	liveUnits := -1 // -1: not counted this cycle
	err := tenancy.InTenantTx(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var locked bool
		if err := tx.QueryRow(ctx, lockSQL, LockName(tenantID)).Scan(&locked); err != nil {
			return fmt.Errorf("advisory lock: %w", err)
		}
		if !locked {
			res.Skipped = true
			return nil
		}
		// Invited users who have signed in are placed before the snapshot,
		// so this cycle's snapshot already holds them.
		if p, ok := s.dir.(InviteePlacer); ok {
			if err := s.placeInvitees(ctx, tx, tenantID, p); err != nil {
				return err
			}
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
			// Nothing to apply, but the tenant's tree size is still reported.
			n, err := storedOrgSubtreeMax(ctx, tx, tenantID, s.opts.MaxOrgUnitDepth)
			if err != nil {
				return err
			}
			subtreeMax = n
			// The closure is not rebuilt, but a hand edit since the last
			// apply can still have left a live unit without its self row
			// (TICK-1058). An apply heals an orphan that is not in the
			// snapshot by tombstoning it, so this path is where one lasts.
			if unrooted, err = unrootedUnits(ctx, tx, tenantID); err != nil {
				return err
			}
			liveUnits = s.liveUnitsBestEffort(ctx, tx, tenantID)
			return nil
		}
		st, err := apply(ctx, tx, tenantID, changes, s.opts.Now(), s.opts.MaxOrgUnitDepth)
		if err != nil {
			return err
		}
		subtreeMax = st.OrgSubtreeMax
		unrooted = st.ClosureUnrooted
		driftRepaired = st.ClosureDriftRepaired
		liveUnits = s.liveUnitsBestEffort(ctx, tx, tenantID)
		if st.AdminGroupDeleted {
			s.log.WarnContext(ctx, "group sync: the tenant's admins group was deleted in the directory; the tenant has no admin until one is recorded",
				slog.String("tenant_id", tenantID.String()))
		}
		s.log.InfoContext(ctx, "group sync applied", slog.String("tenant_id", tenantID.String()),
			slog.Int("principals_written", st.PrincipalsWritten), slog.Int("principals_tombstoned", st.PrincipalsTombstoned),
			slog.Int("groups_written", st.GroupsWritten), slog.Int("groups_deleted", st.GroupsDeleted),
			slog.Int("org_units_written", st.OrgUnitsWritten), slog.Int("org_units_deleted", st.OrgUnitsDeleted),
			slog.Int("org_members_written", st.OrgMembersWritten), slog.Int("org_members_removed", st.OrgMembersRemoved),
			slog.Int("org_subtree_max", st.OrgSubtreeMax),
			slog.Int("org_closure_rows", st.ClosureRows), slog.Bool("org_closure_rebuilt", st.ClosureRebuilt),
			slog.Int("org_closure_unrooted", st.ClosureUnrooted),
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
	// Added only after the commit: a rolled-back run reports nothing.
	if driftRepaired {
		s.drift.Add(ctx, 1)
		s.log.WarnContext(ctx, "group sync: the stored org closure differed from the live org tree and was repaired",
			slog.String("tenant_id", tenantID.String()))
	}
	if unrooted > 0 {
		s.unrooted.Add(ctx, int64(unrooted))
		s.log.WarnContext(ctx, "group sync: live org units have no closure self row and are outside every region",
			slog.String("tenant_id", tenantID.String()), slog.Int("org_closure_unrooted", unrooted))
	}
	if subtreeMax >= 0 {
		s.subtree.Record(ctx, int64(subtreeMax))
	}
	s.mu.Lock()
	if liveUnits >= 0 {
		s.liveUnits[tenantID] = liveUnits
	}
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

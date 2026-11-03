//go:build integration

package groupsync_test

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/metric/noop"

	"github.com/Navaladi2019/TicketingSystem/internal/groupsync"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

var (
	errInjected  = errors.New("injected database fault")
	errSwallowed = errors.New("fault swallowed")
)

// faults numbers every place a database call can fail, in call order, and
// fails exactly the failAt-th one. A sweep over failAt = 0, 1, 2, ... until a
// run injects nothing proves each error path of the code under test returns
// the error and leaves the tenant's rows as they were.
type faults struct {
	failAt, n int
	fired     bool
}

func (f *faults) hit() bool {
	f.n++
	if f.n-1 == f.failAt {
		f.fired = true
		return true
	}
	return false
}

type faultTx struct {
	pgx.Tx
	f *faults
}

func (x *faultTx) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	if x.f.hit() {
		return nil, errInjected
	}
	rows, err := x.Tx.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	if x.f.hit() {
		return &scanFailRows{rows}, nil
	}
	if x.f.hit() {
		return &endFailRows{rows}, nil
	}
	return rows, nil
}

func (x *faultTx) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	if x.f.hit() {
		return errRow{}
	}
	return x.Tx.QueryRow(ctx, sql, args...)
}

func (x *faultTx) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults {
	br := x.Tx.SendBatch(ctx, b)
	if x.f.hit() {
		return &failExecBatch{br}
	}
	if x.f.hit() {
		return &failCloseBatch{br}
	}
	return br
}

func (x *faultTx) Begin(ctx context.Context) (pgx.Tx, error) {
	if x.f.hit() {
		return nil, errInjected
	}
	sp, err := x.Tx.Begin(ctx)
	if err != nil {
		return nil, err
	}
	return &faultTx{sp, x.f}, nil
}

func (x *faultTx) Commit(ctx context.Context) error {
	if x.f.hit() {
		_ = x.Rollback(ctx)
		return errInjected
	}
	return x.Tx.Commit(ctx)
}

type errRow struct{}

func (errRow) Scan(...any) error { return errInjected }

// scanFailRows yields one row whose Scan fails.
type scanFailRows struct{ pgx.Rows }

func (r *scanFailRows) Next() bool        { return true }
func (r *scanFailRows) Scan(...any) error { return errInjected }

// endFailRows yields no row and fails at the end, like a dropped connection.
type endFailRows struct{ pgx.Rows }

func (r *endFailRows) Next() bool { r.Close(); return false }
func (r *endFailRows) Err() error { return errInjected }

type failExecBatch struct{ pgx.BatchResults }

func (b *failExecBatch) Exec() (pgconn.CommandTag, error) {
	_ = b.Close()
	return pgconn.CommandTag{}, errInjected
}

type failCloseBatch struct{ pgx.BatchResults }

func (b *failCloseBatch) Close() error { _ = b.BatchResults.Close(); return errInjected }

// sweep runs run in a fresh tenant transaction with each fault in turn. Every
// injected fault must come back as an error wrapping errInjected and leave
// rows() as it was. The first run that injects nothing must succeed; it
// commits. It returns the number of fault points it passed.
func sweep(t *testing.T, pool *pgxpool.Pool, tid uuid.UUID, rows func() string, run func(context.Context, pgx.Tx) error) int {
	t.Helper()
	ctx := context.Background()
	before := rows()
	for failAt := 0; failAt < 2000; failAt++ {
		f := &faults{failAt: failAt}
		err := tenancy.InTenantTx(ctx, pool, tid, func(tx pgx.Tx) error {
			err := run(ctx, &faultTx{tx, f})
			if err == nil && f.fired {
				// A fault the code swallowed (an ignored Close, say): roll it
				// back so the next run starts from the same rows.
				return errSwallowed
			}
			return err
		})
		if !f.fired {
			if err != nil {
				t.Fatalf("run with no fault failed: %v", err)
			}
			return failAt
		}
		switch {
		case errors.Is(err, errSwallowed):
		case err == nil:
			t.Fatalf("fault %d: no error and no rollback", failAt)
		case !errors.Is(err, errInjected):
			t.Fatalf("fault %d: error %v does not wrap the injected fault", failAt, err)
		}
		if after := rows(); after != before {
			t.Fatalf("fault %d: rows changed despite the error\nbefore: %s\nafter:  %s", failAt, before, after)
		}
	}
	t.Fatal("sweep did not finish")
	return 0
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func unitG(id, parent, p, name string) identity.DirectoryGroup {
	g := unit(id, parent, p)
	g.Name = name
	return g
}

func cycle(t *testing.T, s *groupsync.Syncer) {
	t.Helper()
	rs, err := s.Once(context.Background())
	must(t, err)
	for _, r := range rs {
		must(t, r.Err)
	}
}

// Every database error path of apply returns the error and rolls the tenant
// back, and none of it touches another tenant.
func TestApplyDatabaseFaultsRollBackAndStayInTheTenant(t *testing.T) {
	super, pool := cluster(t)
	a, b := twoTenants()
	seedTenant(t, super, a, 1)
	seedTenant(t, super, b, 2)
	bindAliases(t, super, a, "idp-a")
	dir := &fakeDir{snaps: map[uuid.UUID]identity.DirectoryChanges{}}
	s := newSyncer(t, pool, dir)

	pa := func(rest string) string { return path(a, rest) }
	users := func(ids ...string) []identity.DirectoryUser {
		var out []identity.DirectoryUser
		for _, id := range ids {
			out = append(out, identity.DirectoryUser{KeycloakUserID: id, Email: id + "@a.test", Enabled: true,
				FederatedLinks: []identity.FederatedLink{{IdPAlias: "idp-a", ExternalUserID: "x-" + id}, {IdPAlias: "idp-other", ExternalUserID: "y"}}})
		}
		return out
	}
	dir.set(b, victimSnapshot(b))
	// Baseline: u1..u4, two groups, three org units (ou2 under ou1).
	dir.set(a, identity.DirectoryChanges{
		Users: users("u1", "u2", "u3", "u4"),
		Groups: []identity.DirectoryGroup{
			{KeycloakGroupID: "g1", Path: pa("/devs"), Name: "devs", MemberKeycloakUserIDs: []string{"u1", "u2"}},
			{KeycloakGroupID: "g2", Path: pa("/ops"), Name: "ops", MemberKeycloakUserIDs: []string{"u3"}},
		},
		OrgUnits: []identity.DirectoryGroup{
			unitG("ou1", "", pa("/ou/emea"), "emea"), unitG("ou2", "ou1", pa("/ou/emea/sales"), "sales"), unitG("ou3", "", pa("/ou/apac"), "apac"),
		},
	})
	cycle(t, s)
	// u3 leaves (tombstoned), so the sweep can revive them.
	base := dir.snaps[a]
	base.Users = users("u1", "u2", "u4")
	base.Groups[1].MemberKeycloakUserIDs = nil
	dir.set(a, base)
	cycle(t, s)

	changed := users("u1", "u3", "u4", "u5")
	changed[2].Email = "renamed@a.test"
	changed[0].FederatedLinks = nil
	snap := identity.DirectoryChanges{
		Users: changed,
		Groups: []identity.DirectoryGroup{
			{KeycloakGroupID: "g1", Path: pa("/devs2"), Name: "devs2", MemberKeycloakUserIDs: []string{"u1", "u5", "unknown-user"}},
			{KeycloakGroupID: "g3", Path: pa("/new"), Name: "new", MemberKeycloakUserIDs: []string{"u3"}},
			// Hostile entries: another tenant's path is ignored, never written.
			{KeycloakGroupID: "g-evil", Path: path(b, "/agents"), Name: "agents", MemberKeycloakUserIDs: []string{"u1"}},
		},
		OrgUnits: []identity.DirectoryGroup{
			unitG("ou1", "", pa("/ou/emea"), "emea"),
			unitG("ou2", "", pa("/ou/emea/sales"), "sales-renamed"),
			unitG("ou4", "ou1", pa("/ou/emea/new"), "new"),
			unitG("ou-evil", "", path(b, "/ou/emea"), "evil"),
		},
	}
	rowsA := func() string { return tenantRows(t, super, a) }
	rowsB := tenantRows(t, super, b)
	tried := sweep(t, pool, a, rowsA, func(ctx context.Context, tx pgx.Tx) error {
		return groupsync.ApplyForTest(ctx, tx, a, snap, time.Now(), 8)
	})
	if tried < 30 {
		t.Fatalf("only %d fault points swept; the sweep is not reaching the writes", tried)
	}
	if got := tenantRows(t, super, b); got != rowsB {
		t.Fatalf("tenant B changed:\nbefore %s\nafter  %s", rowsB, got)
	}
	// The clean run committed the changed directory for A alone.
	if got := rowsA(); !strings.Contains(got, pa("/devs2")) || strings.Contains(got, "g-evil") || strings.Contains(got, "ou-evil") {
		t.Fatalf("tenant A rows after the clean run: %s", got)
	}
}

// Every database error path of the invite placement returns the error and
// rolls the tenant back; another tenant's invites are never touched.
func TestPlaceInviteesDatabaseFaults(t *testing.T) {
	super, pool := cluster(t)
	a, b := twoTenants()
	seedTenant(t, super, a, 1)
	seedTenant(t, super, b, 2)
	seedInvite(t, super, a, "one@a.test", "7 days")
	seedInvite(t, super, a, "two@a.test", "7 days")
	seedInvite(t, super, b, "other@b.test", "7 days")
	// The placer consumes the first invite and places nobody for the second.
	dir := newPlacer(func(ctx context.Context, email string, consume func(context.Context) (bool, error)) (bool, error) {
		if email != "one@a.test" {
			return false, nil
		}
		return consume(ctx)
	})
	s, err := groupsync.New(pool, dir, noop.NewMeterProvider(), slog.New(slog.DiscardHandler), groupsync.Options{})
	must(t, err)
	invites := func() string {
		var out string
		must(t, super.QueryRow(context.Background(),
			`SELECT coalesce(string_agg(tenant_id || ':' || email || ':' || (consumed_at IS NOT NULL), ',' ORDER BY tenant_id, email), '') FROM pending_invite`).Scan(&out))
		return out
	}
	before := invites()
	tried := sweep(t, pool, a, invites, func(ctx context.Context, tx pgx.Tx) error {
		return s.PlaceInviteesForTest(ctx, tx, a, dir)
	})
	if tried < 8 {
		t.Fatalf("only %d fault points swept", tried)
	}
	after := invites()
	if !strings.Contains(after, a.String()+":one@a.test:true") || !strings.Contains(after, a.String()+":two@a.test:false") ||
		!strings.Contains(after, b.String()+":other@b.test:false") || after == before {
		t.Fatalf("after the clean run: %s", after)
	}
}

// Syncer-level error paths: bad construction, an unreachable database, a
// directory that fails, and the lag warning.
func TestSyncerErrorPaths(t *testing.T) {
	super, pool := cluster(t)
	a := uuid.New()
	seedTenant(t, super, a, 1)
	log := slog.New(slog.DiscardHandler)
	dir := &fakeDir{snaps: map[uuid.UUID]identity.DirectoryChanges{}}

	if _, err := groupsync.New(nil, dir, noop.NewMeterProvider(), log, groupsync.Options{}); err == nil {
		t.Error("New without a pool must fail")
	}
	if _, err := groupsync.New(pool, nil, noop.NewMeterProvider(), log, groupsync.Options{}); err == nil {
		t.Error("New without a directory must fail")
	}

	// A directory that fails: the tenant's Result carries the error and nothing is applied.
	failing := newPlacer(nil)
	failing.failSnapshot.Store(true)
	s := newSyncer(t, pool, failing)
	rs, err := s.Once(context.Background())
	if err != nil || len(rs) != 1 || rs[0].Err == nil || !strings.Contains(rs[0].Err.Error(), "directory snapshot") {
		t.Fatalf("failing directory: %+v %v", rs, err)
	}

	// A cancelled context stops Once between tenants and fails the lock.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := s.Once(ctx); err == nil {
		t.Error("Once with a cancelled context must fail")
	}

	// Lag past ten minutes is observed (and logged), not an error.
	clock := time.Now()
	lagged, err := groupsync.New(pool, dir, noop.NewMeterProvider(), log, groupsync.Options{Now: func() time.Time {
		clock = clock.Add(11 * time.Minute)
		return clock
	}})
	must(t, err)
	if rs, err := lagged.Once(context.Background()); err != nil || len(rs) != 1 || rs[0].Err != nil {
		t.Fatalf("lagged: %+v %v", rs, err)
	}

	// Run returns nil once its context is cancelled, after at least one cycle.
	runCtx, stop := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer stop()
	runner, err := groupsync.New(pool, dir, noop.NewMeterProvider(), log, groupsync.Options{Interval: 50 * time.Millisecond})
	must(t, err)
	if err := runner.Run(runCtx); err != nil {
		t.Errorf("Run: %v", err)
	}

	// With the database gone, Tenants and Run report it instead of an empty list.
	pool.Close()
	if ts, err := s.Tenants(context.Background()); err == nil || ts != nil {
		t.Errorf("Tenants on a closed pool = %v, %v", ts, err)
	}
	downCtx, stop2 := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer stop2()
	if err := runner.Run(downCtx); err != nil {
		t.Errorf("Run with the database down must keep running until cancelled: %v", err)
	}
}

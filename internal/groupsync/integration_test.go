//go:build integration

// T25 (TICK-78) against a real PostgreSQL. make test-integration
// (scripts/ci/test-integration.sh) starts a disposable cluster and sets
// NAVA_TEST_PG_URL to its superuser. The tests drop and recreate the manifest
// roles and databases, so the script runs the packages one at a time.

package groupsync_test

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/metric/noop"

	"github.com/Navaladi2019/TicketingSystem/db"
	"github.com/Navaladi2019/TicketingSystem/internal/db/catalog"
	"github.com/Navaladi2019/TicketingSystem/internal/db/grants"
	"github.com/Navaladi2019/TicketingSystem/internal/db/migrate"
	"github.com/Navaladi2019/TicketingSystem/internal/db/roles"
	"github.com/Navaladi2019/TicketingSystem/internal/groupsync"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

func password(role string) string { return "test-pw-" + role }

func superConfig(t *testing.T) *pgx.ConnConfig {
	t.Helper()
	url := os.Getenv("NAVA_TEST_PG_URL")
	if url == "" {
		t.Fatal("NAVA_TEST_PG_URL is not set; run make test-integration")
	}
	cfg, err := pgx.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

func ident(s string) string { return pgx.Identifier{s}.Sanitize() }

// cluster rebuilds the manifest roles and databases and runs the real
// migrations, and returns a superuser connection to the platform database and
// a pool that logs in as nava_groupsync.
func cluster(t *testing.T) (*pgx.Conn, *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	m, err := grants.Parse(db.Grants())
	if err != nil {
		t.Fatal(err)
	}
	admin, err := pgx.ConnectConfig(ctx, superConfig(t))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = admin.Close(ctx) })
	// Template databases from the dbtest package may own objects.
	rows, err := admin.Query(ctx, "SELECT datname FROM pg_database WHERE datname LIKE 'nava_tpl_%'")
	if err != nil {
		t.Fatal(err)
	}
	var tpls []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			t.Fatal(err)
		}
		tpls = append(tpls, n)
	}
	rows.Close()
	for _, n := range tpls {
		for _, q := range []string{"ALTER DATABASE " + ident(n) + " WITH IS_TEMPLATE false", "DROP DATABASE " + ident(n)} {
			if _, err := admin.Exec(ctx, q); err != nil {
				t.Fatal(err)
			}
		}
	}
	for _, d := range m.DatabaseNames() {
		if _, err := admin.Exec(ctx, "DROP DATABASE IF EXISTS "+ident(d)+" WITH (FORCE)"); err != nil {
			t.Fatal(err)
		}
	}
	for _, r := range m.RoleNames() {
		if _, err := admin.Exec(ctx, "DROP ROLE IF EXISTS "+ident(r)); err != nil {
			t.Fatal(err)
		}
	}
	log := slog.New(slog.DiscardHandler)
	pw := func(role string) (string, bool) { return password(role), true }
	if _, err := roles.Run(ctx, superConfig(t), m, pw, log); err != nil {
		t.Fatalf("db-roles: %v", err)
	}
	owner := superConfig(t).Copy()
	owner.User, owner.Password, owner.Database = m.Owner(), password(m.Owner()), m.PlatformDatabase
	gt, err := catalog.ParseGlobalTables(db.GlobalTables())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := migrate.Run(ctx, owner, db.Migrations(), m, gt, log, migrate.Options{LockRetrySeconds: 1}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	sup := superConfig(t).Copy()
	sup.Database = m.PlatformDatabase
	super, err := pgx.ConnectConfig(ctx, sup)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = super.Close(ctx) })

	pc, err := pgxpool.ParseConfig(os.Getenv("NAVA_TEST_PG_URL"))
	if err != nil {
		t.Fatal(err)
	}
	pc.ConnConfig.User, pc.ConnConfig.Password, pc.ConnConfig.Database = "nava_groupsync", password("nava_groupsync"), m.PlatformDatabase
	pc.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeExec
	pc.MaxConns = 4
	pool, err := pgxpool.NewWithConfig(ctx, pc)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return super, pool
}

func seedTenant(t *testing.T, super *pgx.Conn, id uuid.UUID, n int) {
	t.Helper()
	ctx := context.Background()
	if _, err := super.Exec(ctx, "INSERT INTO tenants (id, state) VALUES ($1, 'active')", id); err != nil {
		t.Fatal(err)
	}
	if _, err := super.Exec(ctx, "INSERT INTO tenant_identity_binding (tenant_id, issuer, org_id) VALUES ($1, 'https://kc/realms/nava', $2)",
		id, "org-"+strconv.Itoa(n)); err != nil {
		t.Fatal(err)
	}
}

// fakeDir is a DirectoryAdmin whose snapshot a test sets per tenant.
type fakeDir struct {
	identity.DirectoryAdmin
	mu    sync.Mutex
	snaps map[uuid.UUID]identity.DirectoryChanges
	// hook, if set, runs inside ChangesSince.
	hook func(uuid.UUID)
}

func (f *fakeDir) set(id uuid.UUID, c identity.DirectoryChanges) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.snaps[id] = c
}

func (f *fakeDir) ChangesSince(_ context.Context, id uuid.UUID, cursor string) (identity.DirectoryChanges, string, error) {
	if f.hook != nil {
		f.hook(id)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	snap := f.snaps[id]
	b, _ := json.Marshal(snap)
	digest := fmt.Sprintf("%x", sha256.Sum256(b))
	if cursor == digest {
		return identity.DirectoryChanges{Unchanged: true}, digest, nil
	}
	return snap, digest, nil
}

func newSyncer(t *testing.T, pool *pgxpool.Pool, dir identity.DirectoryAdmin) *groupsync.Syncer {
	t.Helper()
	s, err := groupsync.New(pool, dir, noop.NewMeterProvider(), slog.New(slog.DiscardHandler), groupsync.Options{MaxOrgUnitDepth: 8})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func once(t *testing.T, s *groupsync.Syncer) groupsync.Result {
	t.Helper()
	rs, err := s.Once(context.Background())
	if err != nil || len(rs) != 1 {
		t.Fatalf("Once = %v, %v", rs, err)
	}
	return rs[0]
}

func count(t *testing.T, super *pgx.Conn, sql string, args ...any) int {
	t.Helper()
	var n int
	if err := super.QueryRow(context.Background(), sql, args...).Scan(&n); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
	return n
}

func path(id uuid.UUID, rest string) string { return "/t/" + id.String() + rest }

func TestGroupRemovalReflectedAfterOneCycle(t *testing.T) {
	super, pool := cluster(t)
	tid := uuid.New()
	seedTenant(t, super, tid, 1)
	dir := &fakeDir{snaps: map[uuid.UUID]identity.DirectoryChanges{}}
	s := newSyncer(t, pool, dir)

	dir.set(tid, identity.DirectoryChanges{
		Users: []identity.DirectoryUser{
			{KeycloakUserID: "u1", Email: "a@example.test", Enabled: true},
			{KeycloakUserID: "u2", Email: "b@example.test", Enabled: false,
				FederatedLinks: []identity.FederatedLink{{IdPAlias: "idp-x", ExternalUserID: "e2"}}},
		},
		Groups: []identity.DirectoryGroup{
			{KeycloakGroupID: "g1", Path: path(tid, "/agents"), Name: "agents", MemberKeycloakUserIDs: []string{"u1", "u2"}},
			{KeycloakGroupID: "g2", Path: path(tid, "/admins"), Name: "admins", MemberKeycloakUserIDs: []string{"u1"}},
		},
	})
	if r := once(t, s); r.Err != nil {
		t.Fatal(r.Err)
	}
	if n := count(t, super, "SELECT count(*) FROM groups WHERE tenant_id = $1", tid); n != 2 {
		t.Fatalf("groups = %d, want 2", n)
	}
	if n := count(t, super, "SELECT count(*) FROM group_members WHERE tenant_id = $1", tid); n != 3 {
		t.Fatalf("memberships = %d, want 3", n)
	}
	if n := count(t, super, "SELECT count(*) FROM principals WHERE tenant_id = $1 AND federated_links = '[]'::jsonb AND keycloak_user_id = 'u1'", tid); n != 1 {
		t.Errorf("a user with no federated identity must get '[]', not NULL")
	}
	if n := count(t, super, "SELECT count(*) FROM principals WHERE tenant_id = $1 AND keycloak_user_id = 'u2' AND NOT enabled", tid); n != 1 {
		t.Errorf("a disabled user must be written with enabled = false")
	}

	// g2 is removed in Keycloak and u2 leaves g1.
	dir.set(tid, identity.DirectoryChanges{
		Users: []identity.DirectoryUser{
			{KeycloakUserID: "u1", Email: "a@example.test", Enabled: true},
			{KeycloakUserID: "u2", Email: "b@example.test", Enabled: true},
		},
		Groups: []identity.DirectoryGroup{
			{KeycloakGroupID: "g1", Path: path(tid, "/agents"), Name: "agents", MemberKeycloakUserIDs: []string{"u1"}},
		},
	})
	if r := once(t, s); r.Err != nil {
		t.Fatal(r.Err)
	}
	if n := count(t, super, "SELECT count(*) FROM groups WHERE tenant_id = $1 AND keycloak_group_id = 'g2'", tid); n != 0 {
		t.Errorf("removed group still mirrored")
	}
	if n := count(t, super, "SELECT count(*) FROM group_members WHERE tenant_id = $1", tid); n != 1 {
		t.Errorf("memberships = %d, want 1", n)
	}
	if n := count(t, super, "SELECT count(*) FROM principals WHERE tenant_id = $1 AND enabled", tid); n != 2 {
		t.Errorf("re-enabled user not updated")
	}
}

func TestOutsideSubtreeNeverWritten(t *testing.T) {
	super, pool := cluster(t)
	tid, other := uuid.New(), uuid.New()
	seedTenant(t, super, tid, 1)
	seedTenant(t, super, other, 2)
	dir := &fakeDir{snaps: map[uuid.UUID]identity.DirectoryChanges{}}
	s := newSyncer(t, pool, dir)

	rogue := func(id, p string) identity.DirectoryGroup {
		return identity.DirectoryGroup{KeycloakGroupID: id, Path: p, Name: id}
	}
	dir.set(tid, identity.DirectoryChanges{
		Groups: []identity.DirectoryGroup{
			rogue("g-ok", path(tid, "/agents")),
			rogue("g-other", path(other, "/agents")),
			rogue("g-top", "/agents"),
			rogue("g-ou-as-group", path(tid, "/ou/x")),
		},
		OrgUnits: []identity.DirectoryGroup{
			rogue("ou-ok", path(tid, "/ou/emea")),
			rogue("ou-outside", "/somewhere/else"),
			rogue("ou-not-ou", path(tid, "/agents")),
			rogue("ou-container", path(tid, "/ou")),
			rogue("ou-other-tenant", path(other, "/ou/emea")),
		},
	})
	dir.set(other, identity.DirectoryChanges{})
	rs, err := s.Once(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range rs {
		if r.Err != nil {
			t.Fatalf("tenant %s: %v", r.TenantID, r.Err)
		}
	}
	if n := count(t, super, "SELECT count(*) FROM groups WHERE tenant_id = $1", tid); n != 1 {
		t.Errorf("groups = %d, want only g-ok", n)
	}
	if n := count(t, super, "SELECT count(*) FROM org_units WHERE tenant_id = $1", tid); n != 1 {
		t.Errorf("org_units = %d, want only ou-ok", n)
	}
	if n := count(t, super, "SELECT count(*) FROM groups WHERE tenant_id = $1", other); n != 0 {
		t.Errorf("another tenant received %d groups", n)
	}
	if n := count(t, super, "SELECT count(*) FROM org_units WHERE tenant_id = $1", other); n != 0 {
		t.Errorf("another tenant received %d org units", n)
	}
}

func unit(id, parent, p string) identity.DirectoryGroup {
	return identity.DirectoryGroup{KeycloakGroupID: id, ParentKeycloakGroupID: parent, Path: p, Name: id}
}

func TestOrgUnitCycleRefused(t *testing.T) {
	super, pool := cluster(t)
	tid := uuid.New()
	seedTenant(t, super, tid, 1)
	dir := &fakeDir{snaps: map[uuid.UUID]identity.DirectoryChanges{}}
	s := newSyncer(t, pool, dir)

	dir.set(tid, identity.DirectoryChanges{OrgUnits: []identity.DirectoryGroup{
		unit("A", "", path(tid, "/ou/A")), unit("B", "A", path(tid, "/ou/A/B")),
	}})
	if r := once(t, s); r.Err != nil {
		t.Fatal(r.Err)
	}
	before := orgUnits(t, super, tid)

	// A moves under B, which is under A.
	dir.set(tid, identity.DirectoryChanges{OrgUnits: []identity.DirectoryGroup{
		unit("A", "B", path(tid, "/ou/B/A")), unit("B", "A", path(tid, "/ou/A/B")),
	}})
	r := once(t, s)
	if !errors.Is(r.Err, groupsync.ErrOrgUnitCycle) {
		t.Fatalf("err = %v, want ErrOrgUnitCycle", r.Err)
	}
	if after := orgUnits(t, super, tid); after != before {
		t.Errorf("org_units changed by a refused cycle:\n before %s\n after  %s", before, after)
	}
}

func orgUnits(t *testing.T, super *pgx.Conn, tid uuid.UUID) string {
	t.Helper()
	var s string
	err := super.QueryRow(context.Background(), `SELECT coalesce(string_agg(keycloak_group_id || '>' || coalesce(parent_id::text, '-'), ',' ORDER BY keycloak_group_id), '')
		FROM org_units WHERE tenant_id = $1`, tid).Scan(&s)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// A cycle seeded directly in the table, and a chain deeper than the bound,
// both end the walk with an error instead of running on.
func TestWalkStopsAtDepthBound(t *testing.T) {
	super, pool := cluster(t)
	ctx := context.Background()
	tid := uuid.New()
	seedTenant(t, super, tid, 1)

	ids := make([]uuid.UUID, 12)
	for i := range ids {
		ids[i] = uuid.New()
		if _, err := super.Exec(ctx, "INSERT INTO org_units (tenant_id, id, keycloak_group_id, name) VALUES ($1, $2, $3, 'n')",
			tid, ids[i], "k"+strconv.Itoa(i)); err != nil {
			t.Fatal(err)
		}
	}
	walk := func(start uuid.UUID, bound int) error {
		return tenancy.InTenantTx(ctx, pool, tid, func(tx pgx.Tx) error {
			return groupsync.CheckOrgUnitChains(ctx, tx, tid, []string{start.String()}, bound)
		})
	}
	// Chain ids[0] <- ids[1] <- ... <- ids[11]: 12 deep.
	for i := 1; i < len(ids); i++ {
		if _, err := super.Exec(ctx, "UPDATE org_units SET parent_id = $3 WHERE tenant_id = $1 AND id = $2", tid, ids[i], ids[i-1]); err != nil {
			t.Fatal(err)
		}
	}
	if err := walk(ids[11], 20); err != nil {
		t.Errorf("12 deep within a bound of 20: %v", err)
	}
	if err := walk(ids[11], 5); !errors.Is(err, groupsync.ErrOrgUnitDepth) {
		t.Errorf("12 deep with a bound of 5: err = %v, want ErrOrgUnitDepth", err)
	}
	// Close the chain into a cycle.
	if _, err := super.Exec(ctx, "UPDATE org_units SET parent_id = $3 WHERE tenant_id = $1 AND id = $2", tid, ids[0], ids[11]); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- walk(ids[5], 5) }()
	select {
	case err := <-done:
		if !errors.Is(err, groupsync.ErrOrgUnitCycle) {
			t.Errorf("cycle: err = %v, want ErrOrgUnitCycle", err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("the walk did not stop on a cycle")
	}
}

// Two syncers over one tenant never run its sync at the same time: the second
// finds the advisory lock taken and skips the tenant.
func TestTwoSyncersNeverSyncOneTenantConcurrently(t *testing.T) {
	super, pool := cluster(t)
	tid := uuid.New()
	seedTenant(t, super, tid, 1)

	var mu sync.Mutex
	inside, maxInside, calls := 0, 0, 0
	entered, release := make(chan struct{}), make(chan struct{})
	dir := &fakeDir{snaps: map[uuid.UUID]identity.DirectoryChanges{}}
	dir.hook = func(uuid.UUID) {
		mu.Lock()
		inside++
		calls++
		maxInside = max(maxInside, inside)
		first := calls == 1
		mu.Unlock()
		if first {
			close(entered)
			<-release
		}
		mu.Lock()
		inside--
		mu.Unlock()
	}
	a, b := newSyncer(t, pool, dir), newSyncer(t, pool, dir)

	type outcome struct {
		res groupsync.Result
		err error
	}
	const wait = 30 * time.Second
	done := make(chan outcome, 1)
	go func() {
		rs, err := a.Once(context.Background())
		o := outcome{err: err}
		if err == nil && len(rs) == 1 {
			o.res = rs[0]
		} else if err == nil {
			o.err = fmt.Errorf("syncer a visited %d tenants, want 1", len(rs))
		}
		done <- o
	}()
	select {
	case <-entered:
	case o := <-done:
		t.Fatalf("syncer a ended before reading the snapshot: %+v", o)
	case <-time.After(wait):
		t.Fatal("syncer a never reached ChangesSince")
	}
	rb := once(t, b)
	close(release)
	var ra groupsync.Result
	select {
	case o := <-done:
		if o.err != nil {
			t.Fatal(o.err)
		}
		ra = o.res
	case <-time.After(wait):
		t.Fatal("syncer a did not finish")
	}

	if !rb.Skipped {
		t.Errorf("second syncer ran the tenant while the first held it: %+v", rb)
	}
	if ra.Err != nil || ra.Skipped {
		t.Errorf("first syncer = %+v", ra)
	}
	mu.Lock()
	defer mu.Unlock()
	if maxInside != 1 || calls != 1 {
		t.Errorf("concurrent syncs = %d, snapshots = %d; want 1 and 1", maxInside, calls)
	}
}

// A second cycle over an unchanged directory skips the apply, and a change
// after it is applied.
func TestUnchangedSnapshotSkipsApply(t *testing.T) {
	super, pool := cluster(t)
	tid := uuid.New()
	seedTenant(t, super, tid, 1)
	dir := &fakeDir{snaps: map[uuid.UUID]identity.DirectoryChanges{}}
	s := newSyncer(t, pool, dir)
	snap := identity.DirectoryChanges{Users: []identity.DirectoryUser{{KeycloakUserID: "u1", Enabled: true}}}
	dir.set(tid, snap)
	if r := once(t, s); r.Err != nil || r.Unchanged {
		t.Fatalf("first cycle = %+v", r)
	}
	if r := once(t, s); r.Err != nil || !r.Unchanged {
		t.Fatalf("second cycle = %+v, want Unchanged", r)
	}
	snap.Users[0].Enabled = false
	dir.set(tid, snap)
	if r := once(t, s); r.Err != nil || r.Unchanged {
		t.Fatalf("third cycle = %+v", r)
	}
	if n := count(t, super, "SELECT count(*) FROM principals WHERE tenant_id = $1 AND NOT enabled", tid); n != 1 {
		t.Errorf("change after an unchanged cycle not applied")
	}
}

//go:build integration

package groupsync_test

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/Navaladi2019/TicketingSystem/internal/groupsync"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
)

// TICK-1058: a live org unit with no (u, u) closure row is outside every
// region since TICK-1034. Group sync counts such units after each closure
// rebuild and reports them as groupsync_org_closure_unrooted_units (it never
// fails the sync); org_unit_closure_rebuild_all() raises a WARNING.

// unrootedTotal reads groupsync_org_closure_unrooted_units; -1 when the
// counter has never been added to.
func unrootedTotal(t *testing.T, r *sdkmetric.ManualReader) int64 {
	t.Helper()
	var rm metricdata.ResourceMetrics
	if err := r.Collect(context.Background(), &rm); err != nil {
		t.Fatal(err)
	}
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != "groupsync_org_closure_unrooted_units" {
				continue
			}
			sum, ok := m.Data.(metricdata.Sum[int64])
			if !ok {
				t.Fatalf("%s is %T, want a Sum[int64]", m.Name, m.Data)
			}
			var n int64
			for _, dp := range sum.DataPoints {
				if dp.Attributes.Len() != 0 {
					t.Errorf("%s carries attributes %v; a tenant id is never a metric attribute (T01)", m.Name, dp.Attributes.ToSlice())
				}
				n += dp.Value
			}
			return n
		}
	}
	return -1
}

// warnLog collects WARN records as "<tenant_id>: <message>".
type warnLog struct {
	mu   sync.Mutex
	msgs []string
}

func (w *warnLog) Enabled(context.Context, slog.Level) bool { return true }

func (w *warnLog) Handle(_ context.Context, r slog.Record) error {
	if r.Level < slog.LevelWarn {
		return nil
	}
	var tid string
	r.Attrs(func(a slog.Attr) bool {
		if a.Key == "tenant_id" {
			tid = a.Value.String()
		}
		return true
	})
	w.mu.Lock()
	w.msgs = append(w.msgs, tid+": "+r.Message)
	w.mu.Unlock()
	return nil
}

func (w *warnLog) WithAttrs([]slog.Attr) slog.Handler { return w }
func (w *warnLog) WithGroup(string) slog.Handler      { return w }

func (w *warnLog) unrooted() []string {
	w.mu.Lock()
	defer w.mu.Unlock()
	var out []string
	for _, m := range w.msgs {
		if strings.Contains(m, "no closure self row") {
			out = append(out, m)
		}
	}
	return out
}

func meteredSyncer(t *testing.T, pool *pgxpool.Pool, dir identity.DirectoryAdmin) (*groupsync.Syncer, *sdkmetric.ManualReader, *warnLog) {
	t.Helper()
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	t.Cleanup(func() { _ = mp.Shutdown(context.Background()) })
	wl := &warnLog{}
	s, err := groupsync.New(pool, dir, mp, slog.New(wl), groupsync.Options{MaxOrgUnitDepth: 8})
	if err != nil {
		t.Fatal(err)
	}
	return s, reader, wl
}

// seedOrphan inserts a live unit whose parent does not exist, with FK
// triggers off (session_replication_role = replica): the state a bypassed FK
// or a hand edit leaves.
func seedOrphan(t *testing.T, super *pgx.Conn, tid uuid.UUID, kc string) {
	t.Helper()
	exec(t, super, "SET session_replication_role = replica")
	exec(t, super, `INSERT INTO org_units (tenant_id, id, parent_id, keycloak_group_id, name) VALUES ($1, $2, $3, $4, $4)`, tid, uuid.New(), uuid.New(), kc)
	exec(t, super, "RESET session_replication_role")
}

func syncUser(id string) identity.DirectoryUser {
	return identity.DirectoryUser{KeycloakUserID: id, Email: id + "@a.test", Enabled: true}
}

// A healthy tree reports 0 and logs nothing. An in-tenant orphan (FKs off)
// reports 1 and names its tenant in one WARN line while the snapshot is
// unchanged, and the sync does not fail. The next apply tombstones the orphan
// (it is not in the directory) and the count goes back to 0.
func TestClosureUnrootedUnitIsReported(t *testing.T) {
	super, pool := cluster(t)
	tid := uuid.New()
	seedTenant(t, super, tid, 1)
	dir := &fakeDir{snaps: map[uuid.UUID]identity.DirectoryChanges{}}
	s, reader, wl := meteredSyncer(t, pool, dir)
	p := func(rest string) string { return path(tid, "/ou"+rest) }
	units := []identity.DirectoryGroup{unit("r", "", p("/r")), unit("a", "r", p("/r/a")), unit("b", "a", p("/r/a/b"))}

	dir.set(tid, identity.DirectoryChanges{OrgUnits: units, Users: []identity.DirectoryUser{syncUser("u1")}})
	if r := once(t, s); r.Err != nil || r.Unchanged {
		t.Fatalf("first run: %+v", r)
	}
	if r := once(t, s); r.Err != nil || !r.Unchanged {
		t.Fatalf("unchanged run: %+v", r)
	}
	if n := unrootedTotal(t, reader); n > 0 {
		t.Errorf("healthy tree: groupsync_org_closure_unrooted_units = %d, want 0", n)
	}
	if got := wl.unrooted(); len(got) != 0 {
		t.Errorf("healthy tree logged %v", got)
	}

	seedOrphan(t, super, tid, "orphan")
	if r := once(t, s); r.Err != nil || !r.Unchanged {
		t.Fatalf("the unchanged run must report the orphan, not fail: %+v", r)
	}
	if n := unrootedTotal(t, reader); n != 1 {
		t.Errorf("orphan: groupsync_org_closure_unrooted_units = %d, want 1", n)
	}
	if got := wl.unrooted(); len(got) != 1 || !strings.HasPrefix(got[0], tid.String()+": ") {
		t.Errorf("orphan: warn lines = %v, want one naming tenant %s", got, tid)
	}

	// The directory changes: the apply tombstones the orphan and still
	// commits; nothing more is reported.
	dir.set(tid, identity.DirectoryChanges{OrgUnits: units, Users: []identity.DirectoryUser{syncUser("u1"), syncUser("u2")}})
	if r := once(t, s); r.Err != nil || r.Unchanged {
		t.Fatalf("apply run: %+v", r)
	}
	if n := unrootedTotal(t, reader); n != 1 {
		t.Errorf("after the apply: groupsync_org_closure_unrooted_units = %d, want still 1", n)
	}
	if n := count(t, super, "SELECT count(*) FROM org_units WHERE tenant_id = $1 AND keycloak_group_id = 'orphan' AND deleted_at IS NULL", tid); n != 0 {
		t.Errorf("the apply left the orphan live")
	}
	if n := count(t, super, "SELECT count(*) FROM principals WHERE tenant_id = $1 AND keycloak_user_id = 'u2'", tid); n != 1 {
		t.Errorf("the apply did not commit: %d u2 principals", n)
	}
}

// The apply path checks too. A parent cycle written by hand, which the
// snapshot repeats, is not a moved unit, so the chain check does not see it;
// the apply commits, and its two units are reported.
func TestClosureUnrootedCycleReportedOnApply(t *testing.T) {
	super, pool := cluster(t)
	tid := uuid.New()
	seedTenant(t, super, tid, 1)
	dir := &fakeDir{snaps: map[uuid.UUID]identity.DirectoryChanges{}}
	s, reader, wl := meteredSyncer(t, pool, dir)
	p := func(rest string) string { return path(tid, "/ou"+rest) }
	exec(t, super, `INSERT INTO org_units (tenant_id, id, keycloak_group_id, name) VALUES ($1, $2, 'cy1', 'cy1'), ($1, $3, 'cy2', 'cy2')`, tid, uuid.New(), uuid.New())
	exec(t, super, `UPDATE org_units u SET parent_id = o.id FROM org_units o WHERE u.tenant_id = $1 AND o.tenant_id = $1 AND
		((u.keycloak_group_id = 'cy1' AND o.keycloak_group_id = 'cy2') OR (u.keycloak_group_id = 'cy2' AND o.keycloak_group_id = 'cy1'))`, tid)
	dir.set(tid, identity.DirectoryChanges{OrgUnits: []identity.DirectoryGroup{
		unit("r", "", p("/r")), unit("cy1", "cy2", p("/cy2/cy1")), unit("cy2", "cy1", p("/cy1/cy2")),
	}})
	if r := once(t, s); r.Err != nil || r.Unchanged {
		t.Fatalf("the apply must report the cycle, not fail: %+v", r)
	}
	if n := unrootedTotal(t, reader); n != 2 {
		t.Errorf("cycle: groupsync_org_closure_unrooted_units = %d, want 2", n)
	}
	if got := wl.unrooted(); len(got) != 1 {
		t.Errorf("cycle: warn lines = %v, want one", got)
	}
	sameRows(t, "cycle", closureOf(t, super, tid), []string{"r>r"})
}

// Another tenant's orphan is never counted against the synced tenant.
func TestClosureUnrootedCountIsTenantScoped(t *testing.T) {
	super, pool := cluster(t)
	a, b := uuid.New(), uuid.New()
	seedTenant(t, super, a, 1)
	exec(t, super, "INSERT INTO tenants (id, state) VALUES ($1, 'active')", b)
	seedOrphan(t, super, b, "orphan-b")
	dir := &fakeDir{snaps: map[uuid.UUID]identity.DirectoryChanges{}}
	s, reader, wl := meteredSyncer(t, pool, dir)
	dir.set(a, identity.DirectoryChanges{OrgUnits: []identity.DirectoryGroup{unit("r", "", path(a, "/ou/r"))}})
	if r := once(t, s); r.Err != nil {
		t.Fatal(r.Err)
	}
	if n := unrootedTotal(t, reader); n > 0 {
		t.Errorf("tenant A's sync counted %d unrooted units; only tenant B has one", n)
	}
	if got := wl.unrooted(); len(got) != 0 {
		t.Errorf("tenant A's sync logged %v", got)
	}
}

// org_unit_closure_rebuild_all() raises one WARNING for a tenant with an
// unrooted unit, with its count, none for a healthy tenant, and succeeds.
func TestClosureRebuildAllWarnsOnUnrootedUnits(t *testing.T) {
	super, pool := cluster(t)
	ctx := context.Background()
	healthy, broken := uuid.New(), uuid.New()
	seedTenant(t, super, healthy, 1)
	seedTenant(t, super, broken, 2)
	dir := &fakeDir{snaps: map[uuid.UUID]identity.DirectoryChanges{}}
	s := newSyncer(t, pool, dir)
	for _, tid := range []uuid.UUID{healthy, broken} {
		dir.set(tid, identity.DirectoryChanges{OrgUnits: []identity.DirectoryGroup{unit("r", "", path(tid, "/ou/r")), unit("c", "r", path(tid, "/ou/r/c"))}})
	}
	rs, err := s.Once(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range rs {
		if r.Err != nil {
			t.Fatal(r.Err)
		}
	}
	seedOrphan(t, super, broken, "orphan")

	cfg := superConfig(t).Copy()
	cfg.Database = super.Config().Database
	var mu sync.Mutex
	var warnings []string
	cfg.OnNotice = func(_ *pgconn.PgConn, n *pgconn.Notice) {
		if n.Severity == "WARNING" {
			mu.Lock()
			warnings = append(warnings, n.Message)
			mu.Unlock()
		}
	}
	conn, err := pgx.ConnectConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(ctx) })
	if _, err := conn.Exec(ctx, "SET ROLE nava_migrator"); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec(ctx, "SELECT org_unit_closure_rebuild_all()"); err != nil {
		t.Fatalf("org_unit_closure_rebuild_all must warn, not fail: %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	want := "tenant " + broken.String() + " has 1 live org units with no closure self row"
	if len(warnings) != 1 || !strings.Contains(warnings[0], want) {
		t.Errorf("warnings = %q, want one containing %q", warnings, want)
	}
	for _, w := range warnings {
		if strings.Contains(w, healthy.String()) {
			t.Errorf("the healthy tenant was warned about: %q", w)
		}
	}
}

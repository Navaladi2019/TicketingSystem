//go:build integration

package groupsync_test

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"

	"github.com/Navaladi2019/TicketingSystem/internal/groupsync"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
)

// TICK-1166 (AR-TICK-1118-01): every tenant's closure is compared with the live
// tree at least once per ClosureCheckInterval, whatever the cycle did.

type hourly struct {
	s      *groupsync.Syncer
	reader *sdkmetric.ManualReader
	wl     *warnLog
	mu     sync.Mutex
	now    time.Time
	calls  []uuid.UUID // tenants the due comparison ran for
	fail   error       // when set, the comparison fails with it
}

func newHourly(t *testing.T, pool *pgxpool.Pool, dir identity.DirectoryAdmin, start time.Time) *hourly {
	t.Helper()
	h := &hourly{now: start}
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	t.Cleanup(func() { _ = mp.Shutdown(context.Background()) })
	h.reader, h.wl = reader, &warnLog{}
	s, err := groupsync.New(pool, dir, mp, slog.New(h.wl), groupsync.Options{
		MaxOrgUnitDepth: 8, ClosureCheckInterval: time.Hour,
		Now: func() time.Time { h.mu.Lock(); defer h.mu.Unlock(); return h.now },
	})
	if err != nil {
		t.Fatal(err)
	}
	s.SetCompareClosureForTest(func(ctx context.Context, tx pgx.Tx, id uuid.UUID) (bool, error) {
		h.mu.Lock()
		h.calls = append(h.calls, id)
		fail := h.fail
		h.mu.Unlock()
		if fail != nil {
			return false, fail
		}
		return groupsync.CompareAndRepairClosureForTest(ctx, tx, id)
	})
	h.s = s
	return h
}

func (h *hourly) advance(d time.Duration) { h.mu.Lock(); h.now = h.now.Add(d); h.mu.Unlock() }

func (h *hourly) setFail(err error) { h.mu.Lock(); h.fail = err; h.mu.Unlock() }

// compared returns and clears the tenants compared since the last call.
func (h *hourly) compared() []uuid.UUID {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := h.calls
	h.calls = nil
	return out
}

func plantExtraRow(t *testing.T, super *pgx.Conn, tid uuid.UUID) {
	t.Helper()
	exec(t, super, `INSERT INTO org_unit_closure (tenant_id, ancestor_id, descendant_id)
		SELECT $1, b.id, r.id FROM org_units b, org_units r
		WHERE b.tenant_id = $1 AND r.tenant_id = $1 AND b.keycloak_group_id = 'b' AND r.keycloak_group_id = 'r'`, tid)
}

func plantedRows(t *testing.T, super *pgx.Conn, tid uuid.UUID) int {
	t.Helper()
	return count(t, super, `SELECT count(*) FROM org_unit_closure c
		JOIN org_units a ON a.id = c.ancestor_id AND a.tenant_id = c.tenant_id
		JOIN org_units d ON d.id = c.descendant_id AND d.tenant_id = c.tenant_id
		WHERE c.tenant_id = $1 AND a.keycloak_group_id = 'b' AND d.keycloak_group_id = 'r'`, tid)
}

func hourlyTree(tid uuid.UUID, extra ...string) []identity.DirectoryGroup {
	p := func(rest string) string { return path(tid, "/ou"+rest) }
	us := []identity.DirectoryGroup{unit("r", "", p("/r")), unit("a", "r", p("/r/a")), unit("b", "a", p("/r/a/b"))}
	for _, e := range extra {
		us = append(us, unit(e, "a", p("/r/a/"+e)))
	}
	return us
}

func dirUsers(ids ...string) []identity.DirectoryUser {
	var out []identity.DirectoryUser
	for _, id := range ids {
		out = append(out, syncUser(id))
	}
	return out
}

func hourlyStart() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) }

func TestClosureCheckTenantsHaveSeparateTimers(t *testing.T) {
	super, pool := cluster(t)
	a, b := uuid.New(), uuid.New()
	seedTenant(t, super, a, 1)
	dir := &fakeDir{snaps: map[uuid.UUID]identity.DirectoryChanges{}}
	h := newHourly(t, pool, dir, hourlyStart())
	dir.set(a, identity.DirectoryChanges{OrgUnits: hourlyTree(a), Users: dirUsers("u1")})
	onceBoth(t, h.s, 1) // A: first cycle, due at once
	if got := h.compared(); len(got) != 1 || got[0] != a {
		t.Fatalf("first cycle compared %v, want [A]", got)
	}
	// B joins 40 minutes later: A is not due, B is.
	h.advance(40 * time.Minute)
	seedTenant(t, super, b, 2)
	dir.set(b, identity.DirectoryChanges{OrgUnits: hourlyTree(b), Users: dirUsers("u1")})
	onceBoth(t, h.s, 2)
	if got := h.compared(); len(got) != 1 || got[0] != b {
		t.Fatalf("at +40m compared %v, want only B", got)
	}
	// +70m: A is 70 minutes since its check, B 30.
	h.advance(30 * time.Minute)
	onceBoth(t, h.s, 2)
	if got := h.compared(); len(got) != 1 || got[0] != a {
		t.Fatalf("at +70m compared %v, want only A", got)
	}
	// +100m: B is 60 minutes since its check; A 30.
	h.advance(30 * time.Minute)
	onceBoth(t, h.s, 2)
	if got := h.compared(); len(got) != 1 || got[0] != b {
		t.Fatalf("at +100m compared %v, want only B", got)
	}
}

func TestClosureCheckRepairsOnUnchangedCycleWhenDue(t *testing.T) {
	super, pool := cluster(t)
	tid := uuid.New()
	seedTenant(t, super, tid, 1)
	dir := &fakeDir{snaps: map[uuid.UUID]identity.DirectoryChanges{}}
	h := newHourly(t, pool, dir, hourlyStart())
	dir.set(tid, identity.DirectoryChanges{OrgUnits: hourlyTree(tid), Users: dirUsers("u1")})
	onceBoth(t, h.s, 1)
	good := closureOf(t, super, tid)
	plantExtraRow(t, super, tid)

	h.advance(30 * time.Minute)
	if r := once(t, h.s); r.Err != nil || !r.Unchanged {
		t.Fatalf("want an unchanged cycle, got %+v", r)
	}
	if n := driftTotal(t, h.reader); n > 0 || len(h.wl.drift()) != 0 {
		t.Fatalf("not due: counter %d, warns %v; want nothing", n, h.wl.drift())
	}
	h.advance(31 * time.Minute)
	if r := once(t, h.s); r.Err != nil || !r.Unchanged {
		t.Fatalf("want an unchanged cycle, got %+v", r)
	}
	if n := driftTotal(t, h.reader); n != 1 {
		t.Errorf("due: counter = %d, want 1", n)
	}
	if got := h.wl.drift(); len(got) != 1 || !strings.HasPrefix(got[0], tid.String()+": ") {
		t.Errorf("due: warn lines = %v, want one naming tenant %s", got, tid)
	}
	sameRows(t, "after repair", closureOf(t, super, tid), good)
}

func TestClosureCheckDriftOutsideTouchedSubtree(t *testing.T) {
	super, pool := cluster(t)
	tid := uuid.New()
	seedTenant(t, super, tid, 1)
	dir := &fakeDir{snaps: map[uuid.UUID]identity.DirectoryChanges{}}
	h := newHourly(t, pool, dir, hourlyStart())
	dir.set(tid, identity.DirectoryChanges{OrgUnits: hourlyTree(tid), Users: dirUsers("u1")})
	onceBoth(t, h.s, 1)
	plantExtraRow(t, super, tid) // descendant r: outside the subtree of a new unit under a

	// Not due: a run that touches a unit leaves it (today's behaviour).
	h.advance(10 * time.Minute)
	dir.set(tid, identity.DirectoryChanges{OrgUnits: hourlyTree(tid, "c"), Users: dirUsers("u1")})
	if r := once(t, h.s); r.Err != nil || r.Unchanged {
		t.Fatalf("%+v", r)
	}
	if n := driftTotal(t, h.reader); n > 0 {
		t.Errorf("not due: counter = %d, want 0", n)
	}
	if n := plantedRows(t, super, tid); n != 1 {
		t.Errorf("not due: planted rows = %d, want 1 (left in place)", n)
	}
	// Due: the same kind of run repairs and counts.
	h.advance(51 * time.Minute)
	dir.set(tid, identity.DirectoryChanges{OrgUnits: hourlyTree(tid, "c", "d"), Users: dirUsers("u1")})
	if r := once(t, h.s); r.Err != nil || r.Unchanged {
		t.Fatalf("%+v", r)
	}
	if n := driftTotal(t, h.reader); n != 1 {
		t.Errorf("due: counter = %d, want 1", n)
	}
	if got := h.wl.drift(); len(got) != 1 || !strings.HasPrefix(got[0], tid.String()+": ") {
		t.Errorf("due: warn lines = %v", got)
	}
	if n := plantedRows(t, super, tid); n != 0 {
		t.Errorf("due: planted rows = %d, want 0", n)
	}
}

func TestClosureCheckFailureKeepsRevocationAndStaysDue(t *testing.T) {
	super, pool := cluster(t)
	tid := uuid.New()
	seedTenant(t, super, tid, 1)
	dir := &fakeDir{snaps: map[uuid.UUID]identity.DirectoryChanges{}}
	h := newHourly(t, pool, dir, hourlyStart())
	dir.set(tid, identity.DirectoryChanges{OrgUnits: hourlyTree(tid), Users: dirUsers("u1", "u2")})
	onceBoth(t, h.s, 1)
	h.compared()
	plantExtraRow(t, super, tid)

	h.advance(2 * time.Hour)
	h.setFail(errors.New("injected comparison failure"))
	// u2 leaves the directory (a tombstone) and a unit is added (a delta).
	dir.set(tid, identity.DirectoryChanges{OrgUnits: hourlyTree(tid, "c"), Users: dirUsers("u1")})
	if r := once(t, h.s); r.Err != nil {
		t.Fatalf("a failed comparison must not fail the sync: %v", r.Err)
	}
	if n := count(t, super, "SELECT count(*) FROM principals WHERE tenant_id = $1 AND keycloak_user_id = 'u2' AND NOT enabled", tid); n != 1 {
		t.Errorf("the tombstone must commit, got %d", n)
	}
	if n := count(t, super, "SELECT count(*) FROM org_units WHERE tenant_id = $1 AND keycloak_group_id = 'c'", tid); n != 1 {
		t.Errorf("the delta must commit, got %d units", n)
	}
	if n := driftTotal(t, h.reader); n > 0 {
		t.Errorf("counter = %d, want 0", n)
	}
	named := 0
	h.wl.mu.Lock()
	for _, m := range h.wl.msgs {
		if strings.HasPrefix(m, tid.String()+": ") && strings.Contains(m, "comparing the org closure") {
			named++
		}
	}
	h.wl.mu.Unlock()
	if named != 1 {
		t.Errorf("want one WARN naming the tenant, got %d", named)
	}
	// Still due on the next cycle, with the same clock.
	h.compared()
	h.setFail(nil)
	if r := once(t, h.s); r.Err != nil || !r.Unchanged {
		t.Fatalf("%+v", r)
	}
	if got := h.compared(); len(got) != 1 {
		t.Errorf("next cycle compared %v, want one (still due)", got)
	}
	if n := driftTotal(t, h.reader); n != 1 {
		t.Errorf("repair on the next cycle: counter = %d, want 1", n)
	}
}

func TestClosureCheckRolledBackCycleDoesNotResetTimer(t *testing.T) {
	super, pool := cluster(t)
	tid := uuid.New()
	seedTenant(t, super, tid, 1)
	dir := &fakeDir{snaps: map[uuid.UUID]identity.DirectoryChanges{}}
	h := newHourly(t, pool, dir, hourlyStart())
	dir.set(tid, identity.DirectoryChanges{OrgUnits: hourlyTree(tid), Users: dirUsers("u1")})
	onceBoth(t, h.s, 1)
	h.compared()
	plantExtraRow(t, super, tid)
	exec(t, super, `CREATE FUNCTION fail_closure_commit() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'injected commit failure'; END $$`)
	exec(t, super, `CREATE CONSTRAINT TRIGGER fail_closure_commit AFTER INSERT ON org_unit_closure DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION fail_closure_commit()`)

	h.advance(2 * time.Hour)
	if r := once(t, h.s); r.Err == nil || !strings.Contains(r.Err.Error(), "injected commit failure") {
		t.Fatalf("the commit was meant to fail, got %v", r.Err)
	}
	if n := driftTotal(t, h.reader); n > 0 {
		t.Errorf("failed commit: counter = %d, want 0", n)
	}
	exec(t, super, "DROP TRIGGER fail_closure_commit ON org_unit_closure")
	h.compared()
	if r := once(t, h.s); r.Err != nil {
		t.Fatal(r.Err)
	}
	if got := h.compared(); len(got) != 1 {
		t.Errorf("after the rollback the tenant must still be due; compared %v", got)
	}
	if n := driftTotal(t, h.reader); n != 1 {
		t.Errorf("retry: counter = %d, want 1", n)
	}
}

func TestClosureCheckSkippedCycleDoesNotResetTimer(t *testing.T) {
	super, pool := cluster(t)
	tid := uuid.New()
	seedTenant(t, super, tid, 1)
	dir := &fakeDir{snaps: map[uuid.UUID]identity.DirectoryChanges{}}
	h := newHourly(t, pool, dir, hourlyStart())
	dir.set(tid, identity.DirectoryChanges{OrgUnits: hourlyTree(tid), Users: dirUsers("u1")})
	onceBoth(t, h.s, 1)
	h.compared()
	h.advance(2 * time.Hour)
	// Another session holds the tenant's advisory lock: this cycle is skipped.
	ctx := context.Background()
	holder, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := holder.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1, 0))", groupsync.LockName(tid)); err != nil {
		t.Fatal(err)
	}
	if r := once(t, h.s); !r.Skipped {
		t.Fatalf("want a skipped cycle, got %+v", r)
	}
	_ = holder.Rollback(ctx)
	if got := h.compared(); len(got) != 0 {
		t.Errorf("a skipped cycle compared %v", got)
	}
	if r := once(t, h.s); r.Err != nil || r.Skipped {
		t.Fatalf("%+v", r)
	}
	if got := h.compared(); len(got) != 1 {
		t.Errorf("the tenant must still be due after a skipped cycle; compared %v", got)
	}
}

func TestClosureCheckNewSyncerComparesOnFirstCycleEvenWhenUnitsTouched(t *testing.T) {
	super, pool := cluster(t)
	tid := uuid.New()
	seedTenant(t, super, tid, 1)
	dir := &fakeDir{snaps: map[uuid.UUID]identity.DirectoryChanges{}}
	h := newHourly(t, pool, dir, hourlyStart())
	dir.set(tid, identity.DirectoryChanges{OrgUnits: hourlyTree(tid), Users: dirUsers("u1")})
	onceBoth(t, h.s, 1)
	good := closureOf(t, super, tid)
	plantExtraRow(t, super, tid)

	// Restart: a new Syncer one minute later; its first run touches a unit.
	h2 := newHourly(t, pool, dir, hourlyStart().Add(time.Minute))
	dir.set(tid, identity.DirectoryChanges{OrgUnits: hourlyTree(tid, "c"), Users: dirUsers("u1")})
	if r := once(t, h2.s); r.Err != nil || r.Unchanged {
		t.Fatalf("%+v", r)
	}
	if got := h2.compared(); len(got) != 1 {
		t.Errorf("the new syncer compared %v on its first cycle, want one", got)
	}
	if n := driftTotal(t, h2.reader); n != 1 {
		t.Errorf("counter = %d, want 1", n)
	}
	if n := plantedRows(t, super, tid); n != 0 {
		t.Errorf("planted rows = %d, want 0", n)
	}
	if got := len(closureOf(t, super, tid)); got != len(good)+3 { // c: (r,c) (a,c) (c,c)
		t.Errorf("closure has %d rows, want %d", got, len(good)+3)
	}
}

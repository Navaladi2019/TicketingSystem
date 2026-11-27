//go:build integration

package groupsync_test

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/Navaladi2019/TicketingSystem/internal/groupsync"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
)

// TICK-1124: a drift repair (the stored closure differs from the live one) adds
// one to groupsync_org_closure_drift_repairs and logs one WARN naming the
// tenant, after the commit. A first import, a delta and a clean check add
// nothing.

// driftTotal reads the counter; -1 when it was never added to.
func driftTotal(t *testing.T, r *sdkmetric.ManualReader) int64 {
	t.Helper()
	var rm metricdata.ResourceMetrics
	if err := r.Collect(context.Background(), &rm); err != nil {
		t.Fatal(err)
	}
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != "groupsync_org_closure_drift_repairs" {
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

func (w *warnLog) drift() []string {
	w.mu.Lock()
	defer w.mu.Unlock()
	var out []string
	for _, m := range w.msgs {
		if strings.Contains(m, "differed from the live org tree") {
			out = append(out, m)
		}
	}
	return out
}

func TestClosureDriftRepairIsCounted(t *testing.T) {
	for _, tc := range []struct{ name, plant string }{
		{"extra row", `INSERT INTO org_unit_closure (tenant_id, ancestor_id, descendant_id)
			SELECT $1, b.id, r.id FROM org_units b, org_units r
			WHERE b.tenant_id = $1 AND r.tenant_id = $1 AND b.keycloak_group_id = 'b' AND r.keycloak_group_id = 'r'`},
		{"missing non-self row", `DELETE FROM org_unit_closure c USING org_units a, org_units d
			WHERE c.tenant_id = $1 AND a.tenant_id = $1 AND d.tenant_id = $1
			AND a.id = c.ancestor_id AND d.id = c.descendant_id AND a.keycloak_group_id = 'r' AND d.keycloak_group_id = 'b'`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			super, pool := cluster(t)
			tid := uuid.New()
			seedTenant(t, super, tid, 1)
			dir := &fakeDir{snaps: map[uuid.UUID]identity.DirectoryChanges{}}
			s, reader, wl := meteredSyncer(t, pool, dir)
			p := func(rest string) string { return path(tid, "/ou"+rest) }
			units := []identity.DirectoryGroup{unit("r", "", p("/r")), unit("a", "r", p("/r/a")), unit("b", "a", p("/r/a/b"))}
			users := func(ids ...string) []identity.DirectoryUser {
				var out []identity.DirectoryUser
				for _, id := range ids {
					out = append(out, syncUser(id))
				}
				return out
			}

			// First import: a full rebuild, not drift.
			dir.set(tid, identity.DirectoryChanges{OrgUnits: units, Users: users("u1")})
			if r := once(t, s); r.Err != nil || r.Unchanged {
				t.Fatalf("first run: %+v", r)
			}
			// A delta: a unit is added.
			units2 := append(append([]identity.DirectoryGroup{}, units...), unit("c", "a", p("/r/a/c")))
			dir.set(tid, identity.DirectoryChanges{OrgUnits: units2, Users: users("u1")})
			if r := once(t, s); r.Err != nil || r.Unchanged {
				t.Fatalf("delta run: %+v", r)
			}
			// A clean check: an apply that touches no unit.
			dir.set(tid, identity.DirectoryChanges{OrgUnits: units2, Users: users("u1", "u2")})
			if r := once(t, s); r.Err != nil || r.Unchanged {
				t.Fatalf("clean run: %+v", r)
			}
			if n := driftTotal(t, reader); n > 0 {
				t.Errorf("first import, delta and clean check: counter = %d, want 0", n)
			}
			if got := wl.drift(); len(got) != 0 {
				t.Errorf("no drift yet, but logged %v", got)
			}
			good := closureOf(t, super, tid)

			exec(t, super, tc.plant, tid)
			dir.set(tid, identity.DirectoryChanges{OrgUnits: units2, Users: users("u1", "u2", "u3")})
			if r := once(t, s); r.Err != nil || r.Unchanged {
				t.Fatalf("repair run: %+v", r)
			}
			if n := driftTotal(t, reader); n != 1 {
				t.Errorf("repair: counter = %d, want 1", n)
			}
			if got := wl.drift(); len(got) != 1 || !strings.HasPrefix(got[0], tid.String()+": ") {
				t.Errorf("repair: warn lines = %v, want one naming tenant %s", got, tid)
			}
			sameRows(t, "after repair", closureOf(t, super, tid), good)

			// Repaired: the next clean check adds nothing.
			dir.set(tid, identity.DirectoryChanges{OrgUnits: units2, Users: users("u1", "u2", "u3", "u4")})
			if r := once(t, s); r.Err != nil {
				t.Fatal(r.Err)
			}
			if n := driftTotal(t, reader); n != 1 {
				t.Errorf("after repair: counter = %d, want still 1", n)
			}
		})
	}
}

// An emptied closure on a tenant with live units, found by a run that touches
// no unit, is drift. A tenant with no units reports nothing.
func TestClosureDriftEmptyTableIsCounted(t *testing.T) {
	super, pool := cluster(t)
	tid, empty := uuid.New(), uuid.New()
	seedTenant(t, super, tid, 1)
	seedTenant(t, super, empty, 2)
	dir := &fakeDir{snaps: map[uuid.UUID]identity.DirectoryChanges{}}
	s, reader, wl := meteredSyncer(t, pool, dir)
	units := []identity.DirectoryGroup{unit("r", "", path(tid, "/ou/r")), unit("a", "r", path(tid, "/ou/r/a"))}
	dir.set(tid, identity.DirectoryChanges{OrgUnits: units, Users: []identity.DirectoryUser{syncUser("u1")}})
	dir.set(empty, identity.DirectoryChanges{Users: []identity.DirectoryUser{syncUser("e1")}})
	onceBoth(t, s, 2)
	dir.set(empty, identity.DirectoryChanges{Users: []identity.DirectoryUser{syncUser("e1"), syncUser("e2")}})
	onceBoth(t, s, 2)
	if n := driftTotal(t, reader); n > 0 {
		t.Errorf("first import and a tenant with no units: counter = %d, want 0", n)
	}
	good := closureOf(t, super, tid)
	exec(t, super, "DELETE FROM org_unit_closure WHERE tenant_id = $1", tid)
	dir.set(tid, identity.DirectoryChanges{OrgUnits: units, Users: []identity.DirectoryUser{syncUser("u1"), syncUser("u2")}})
	onceBoth(t, s, 2)
	if n := driftTotal(t, reader); n != 1 {
		t.Errorf("emptied closure: counter = %d, want 1", n)
	}
	if got := wl.drift(); len(got) != 1 || !strings.HasPrefix(got[0], tid.String()+": ") {
		t.Errorf("emptied closure: warn lines = %v, want one naming tenant %s", got, tid)
	}
	sameRows(t, "after repair", closureOf(t, super, tid), good)
}

// The count and the WARN come after the commit: a repair whose transaction
// fails at commit adds 0 and logs nothing; the retry repairs and counts once.
func TestClosureDriftNotCountedWhenCommitFails(t *testing.T) {
	super, pool := cluster(t)
	tid := uuid.New()
	seedTenant(t, super, tid, 1)
	dir := &fakeDir{snaps: map[uuid.UUID]identity.DirectoryChanges{}}
	s, reader, wl := meteredSyncer(t, pool, dir)
	units := []identity.DirectoryGroup{unit("r", "", path(tid, "/ou/r")), unit("a", "r", path(tid, "/ou/r/a"))}
	run := func(users ...string) groupsync.Result {
		var us []identity.DirectoryUser
		for _, u := range users {
			us = append(us, syncUser(u))
		}
		dir.set(tid, identity.DirectoryChanges{OrgUnits: units, Users: us})
		return once(t, s)
	}
	if r := run("u1"); r.Err != nil {
		t.Fatal(r.Err)
	}
	exec(t, super, "DELETE FROM org_unit_closure WHERE tenant_id = $1", tid)
	// The repair inserts closure rows; this trigger fails the commit after them.
	exec(t, super, `CREATE FUNCTION fail_closure_commit() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'injected commit failure'; END $$`)
	exec(t, super, `CREATE CONSTRAINT TRIGGER fail_closure_commit AFTER INSERT ON org_unit_closure DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION fail_closure_commit()`)
	if r := run("u1", "u2"); r.Err == nil || !strings.Contains(r.Err.Error(), "injected commit failure") {
		t.Fatalf("the commit was meant to fail with the injected error, got %v", r.Err)
	}
	if n := driftTotal(t, reader); n > 0 {
		t.Errorf("failed commit: counter = %d, want 0", n)
	}
	if got := wl.drift(); len(got) != 0 {
		t.Errorf("failed commit logged %v", got)
	}
	exec(t, super, "DROP TRIGGER fail_closure_commit ON org_unit_closure")
	if r := run("u1", "u2"); r.Err != nil {
		t.Fatal(r.Err)
	}
	if n := driftTotal(t, reader); n != 1 {
		t.Errorf("retry: counter = %d, want 1", n)
	}
	if got := wl.drift(); len(got) != 1 {
		t.Errorf("retry: warn lines = %v, want one", got)
	}
}

// onceBoth runs one cycle over n tenants and fails unless every one synced.
func onceBoth(t *testing.T, s *groupsync.Syncer, n int) {
	t.Helper()
	rs, err := s.Once(context.Background())
	if err != nil || len(rs) != n {
		t.Fatalf("Once = %v, %v; want %d results", rs, err, n)
	}
	for _, r := range rs {
		if r.Err != nil {
			t.Fatal(r.Err)
		}
	}
}

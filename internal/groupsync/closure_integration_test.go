//go:build integration

package groupsync_test

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/groupsync"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
)

// TICK-1033 (S31 fix; ADR-010-A2): group sync rebuilds the tenant's
// org_unit_closure in the transaction that writes the tree.

// wantClosure computes, from a snapshot alone, the rows the closure must hold:
// each unit paired with itself and every unit above it, as "anc>desc" over
// Keycloak ids.
func wantClosure(units []identity.DirectoryGroup) []string {
	parent := map[string]string{}
	for _, u := range units {
		parent[u.KeycloakGroupID] = u.ParentKeycloakGroupID
	}
	var out []string
	for _, u := range units {
		for anc := u.KeycloakGroupID; anc != ""; anc = parent[anc] {
			out = append(out, anc+">"+u.KeycloakGroupID)
		}
	}
	sort.Strings(out)
	return out
}

// closureOf reads the tenant's closure as "anc>desc" over Keycloak ids.
func closureOf(t *testing.T, super *pgx.Conn, tid uuid.UUID) []string {
	t.Helper()
	rows, err := super.Query(context.Background(), `SELECT a.keycloak_group_id || '>' || d.keycloak_group_id
		FROM org_unit_closure c
		JOIN org_units a ON a.tenant_id = c.tenant_id AND a.id = c.ancestor_id
		JOIN org_units d ON d.tenant_id = c.tenant_id AND d.id = c.descendant_id
		WHERE c.tenant_id = $1`, tid)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			t.Fatal(err)
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	sort.Strings(out)
	return out
}

func sameRows(t *testing.T, what string, got, want []string) {
	t.Helper()
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("%s: closure = %v, want %v", what, got, want)
	}
}

// closureTIDs identifies the stored closure rows physically: a rebuild
// deletes and re-inserts them, so the ctids change.
func closureTIDs(t *testing.T, super *pgx.Conn, tid uuid.UUID) string {
	t.Helper()
	var s string
	if err := super.QueryRow(context.Background(), `SELECT coalesce(string_agg(ctid::text, ',' ORDER BY ctid), '') FROM org_unit_closure WHERE tenant_id = $1`, tid).Scan(&s); err != nil {
		t.Fatal(err)
	}
	return s
}

// Add, move, tombstone a leaf, tombstone an intermediate unit (its subtree
// leaves with it), revive: after each run the closure equals the expected set.
func TestClosureFollowsTheTree(t *testing.T) {
	super, pool := cluster(t)
	tid := uuid.New()
	seedTenant(t, super, tid, 1)
	dir := &fakeDir{snaps: map[uuid.UUID]identity.DirectoryChanges{}}
	s := newSyncer(t, pool, dir)
	p := func(rest string) string { return path(tid, "/ou"+rest) }
	step := func(what string, units ...identity.DirectoryGroup) {
		t.Helper()
		dir.set(tid, identity.DirectoryChanges{OrgUnits: units})
		if r := once(t, s); r.Err != nil {
			t.Fatalf("%s: %v", what, r.Err)
		}
		sameRows(t, what, closureOf(t, super, tid), wantClosure(units))
	}
	root := unit("emea", "", p("/emea"))
	de := unit("de", "emea", p("/emea/de"))
	fr := unit("fr", "emea", p("/emea/fr"))
	munich := unit("munich", "de", p("/emea/de/munich"))
	apac := unit("apac", "", p("/apac"))

	step("initial tree", root, de, munich, apac)
	step("add fr", root, de, munich, apac, fr)
	step("move munich under fr", root, de, unit("munich", "fr", p("/emea/fr/munich")), apac, fr)
	step("tombstone the leaf munich", root, de, apac, fr)
	step("tombstone fr", root, de, apac)
	step("munich returns under de", root, de, munich, apac)
	step("tombstone the intermediate de and its subtree", root, apac)
	if n := count(t, super, "SELECT count(*) FROM org_unit_closure c JOIN org_units u ON u.tenant_id = c.tenant_id AND u.id = c.descendant_id WHERE c.tenant_id = $1 AND u.deleted_at IS NOT NULL", tid); n != 0 {
		t.Errorf("%d closure rows name a tombstoned unit", n)
	}
	step("revive de", root, de, apac)
}

// A run that changes no unit leaves the closure's rows untouched; one that
// finds the stored closure stale (TICK-1032 shipped before this code) repairs
// it with the tree unchanged.
func TestClosureSkippedWhenTreeUnchangedAndRepairedWhenStale(t *testing.T) {
	super, pool := cluster(t)
	tid := uuid.New()
	seedTenant(t, super, tid, 1)
	dir := &fakeDir{snaps: map[uuid.UUID]identity.DirectoryChanges{}}
	s := newSyncer(t, pool, dir)
	units := []identity.DirectoryGroup{unit("a", "", path(tid, "/ou/a")), unit("b", "a", path(tid, "/ou/a/b"))}
	user := func(id string) identity.DirectoryUser {
		return identity.DirectoryUser{KeycloakUserID: id, Email: id + "@a.test", Enabled: true}
	}
	dir.set(tid, identity.DirectoryChanges{OrgUnits: units, Users: []identity.DirectoryUser{user("u1")}})
	if r := once(t, s); r.Err != nil {
		t.Fatal(r.Err)
	}
	want := wantClosure(units)
	sameRows(t, "first run", closureOf(t, super, tid), want)
	before := closureTIDs(t, super, tid)

	// Only a user changes: no unit written, closure correct, rows not rewritten.
	dir.set(tid, identity.DirectoryChanges{OrgUnits: units, Users: []identity.DirectoryUser{user("u1"), user("u2")}})
	if r := once(t, s); r.Err != nil || r.Unchanged {
		t.Fatalf("user-only run: %+v", r)
	}
	if after := closureTIDs(t, super, tid); after != before {
		t.Errorf("a run that changed no unit rewrote a correct closure")
	}

	// Drift: rows missing, then the table empty. A user-only run repairs each.
	users := []identity.DirectoryUser{user("u1"), user("u2")}
	for _, damage := range []struct{ what, sql string }{
		{"a missing row", "DELETE FROM org_unit_closure WHERE tenant_id = $1 AND ancestor_id <> descendant_id"},
		{"an empty table", "DELETE FROM org_unit_closure WHERE tenant_id = $1"},
	} {
		if _, err := super.Exec(context.Background(), damage.sql, tid); err != nil {
			t.Fatal(err)
		}
		users = append(users, user("u-"+strings.ReplaceAll(damage.what, " ", "-")))
		dir.set(tid, identity.DirectoryChanges{OrgUnits: units, Users: users})
		if r := once(t, s); r.Err != nil {
			t.Fatalf("%s: %v", damage.what, r.Err)
		}
		sameRows(t, "repair of "+damage.what, closureOf(t, super, tid), want)
	}
}

// A sync that fails after the tree write changes neither the tree nor the
// closure.
func TestClosureRolledBackWithTheTree(t *testing.T) {
	super, pool := cluster(t)
	tid := uuid.New()
	seedTenant(t, super, tid, 1)
	dir := &fakeDir{snaps: map[uuid.UUID]identity.DirectoryChanges{}}
	s := newSyncer(t, pool, dir)
	dir.set(tid, identity.DirectoryChanges{OrgUnits: []identity.DirectoryGroup{
		unit("A", "", path(tid, "/ou/A")), unit("B", "A", path(tid, "/ou/A/B")), unit("C", "", path(tid, "/ou/C")),
	}})
	if r := once(t, s); r.Err != nil {
		t.Fatal(r.Err)
	}
	units, closure := orgUnits(t, super, tid), closureOf(t, super, tid)

	// New unit D is written, then the A/B cycle is found: the run fails.
	dir.set(tid, identity.DirectoryChanges{OrgUnits: []identity.DirectoryGroup{
		unit("A", "B", path(tid, "/ou/B/A")), unit("B", "A", path(tid, "/ou/A/B")), unit("C", "", path(tid, "/ou/C")), unit("D", "C", path(tid, "/ou/C/D")),
	}})
	if r := once(t, s); !errors.Is(r.Err, groupsync.ErrOrgUnitCycle) {
		t.Fatalf("err = %v, want ErrOrgUnitCycle", r.Err)
	}
	if got := orgUnits(t, super, tid); got != units {
		t.Errorf("tree changed:\n before %s\n after  %s", units, got)
	}
	sameRows(t, "after the failed run", closureOf(t, super, tid), closure)
}

// The Go query and the migration's backfill (org_unit_closure_rebuild_all,
// run as nava_migrator) give the same rows, on a tree with tombstones and a
// parent cycle.
func TestClosureQueryMatchesMigrationBackfill(t *testing.T) {
	super, pool := cluster(t)
	ctx := context.Background()
	tid := uuid.New()
	seedTenant(t, super, tid, 1)
	dir := &fakeDir{snaps: map[uuid.UUID]identity.DirectoryChanges{}}
	s := newSyncer(t, pool, dir)
	p := func(rest string) string { return path(tid, "/ou"+rest) }
	dir.set(tid, identity.DirectoryChanges{OrgUnits: []identity.DirectoryGroup{
		unit("r", "", p("/r")), unit("a", "r", p("/r/a")), unit("b", "a", p("/r/a/b")), unit("c", "r", p("/r/c")), unit("x", "", p("/x")),
	}})
	if r := once(t, s); r.Err != nil {
		t.Fatal(r.Err)
	}
	// a and b are tombstoned; r is renamed so the run rebuilds.
	dir.set(tid, identity.DirectoryChanges{OrgUnits: []identity.DirectoryGroup{
		unitG("r", "", p("/r"), "root"), unit("c", "r", p("/r/c")), unit("x", "", p("/x")),
	}})
	if r := once(t, s); r.Err != nil {
		t.Fatal(r.Err)
	}
	// A parent cycle seeded directly: neither unit has a root above it.
	exec(t, super, `INSERT INTO org_units (tenant_id, id, keycloak_group_id, name) VALUES ($1, $2, 'cy1', 'cy1'), ($1, $3, 'cy2', 'cy2')`, tid, uuid.New(), uuid.New())
	exec(t, super, `UPDATE org_units u SET parent_id = o.id FROM org_units o WHERE u.tenant_id = $1 AND o.tenant_id = $1 AND
		((u.keycloak_group_id = 'cy1' AND o.keycloak_group_id = 'cy2') OR (u.keycloak_group_id = 'cy2' AND o.keycloak_group_id = 'cy1'))`, tid)
	// Run the Go rebuild over that state: the closure is stale, so it rebuilds.
	exec(t, super, "DELETE FROM org_unit_closure WHERE tenant_id = $1", tid)
	dir.set(tid, identity.DirectoryChanges{OrgUnits: []identity.DirectoryGroup{
		unitG("r", "", p("/r"), "root"), unit("c", "r", p("/r/c")), unit("x", "", p("/x")),
	}, Users: []identity.DirectoryUser{{KeycloakUserID: "u", Email: "u@a.test", Enabled: true}}})
	if r := once(t, s); r.Err != nil {
		t.Fatal(r.Err)
	}
	goRows := closureOf(t, super, tid)
	sameRows(t, "go rebuild", goRows, []string{"c>c", "r>c", "r>r", "x>x"})

	// The migration's function, as nava_migrator, rebuilds every tenant.
	if _, err := super.Exec(ctx, "SET ROLE nava_migrator"); err != nil {
		t.Fatal(err)
	}
	_, err := super.Exec(ctx, "SELECT org_unit_closure_rebuild_all()")
	if _, rerr := super.Exec(ctx, "RESET ROLE"); rerr != nil {
		t.Fatal(rerr)
	}
	if err != nil {
		t.Fatalf("org_unit_closure_rebuild_all: %v", err)
	}
	sameRows(t, "migration backfill", closureOf(t, super, tid), goRows)
}

// SP-1b: a full rebuild of 10,111 units is targeted at <= 0.35 s. The test logs
// the time and never fails on it. A sync that rebuilds a stale closure is
// timed against one that finds it right; the difference is the rebuild.
func TestClosureRebuildTimeAtBenchScale(t *testing.T) {
	super, pool := cluster(t)
	tid := uuid.New()
	seedTenant(t, super, tid, 1)
	dir := &fakeDir{snaps: map[uuid.UUID]identity.DirectoryChanges{}}
	s := newSyncer(t, pool, dir)

	// One root, 10 regions, 100 countries, the rest as leaves: 4 deep.
	const total = 10111
	units := []identity.DirectoryGroup{unit("u0", "", path(tid, "/ou/u0"))}
	for i := 1; i < total; i++ {
		var parent int
		switch {
		case i <= 10:
			parent = 0
		case i <= 110:
			parent = (i-11)%10 + 1
		default:
			parent = (i-111)%100 + 11
		}
		units = append(units, unit(fmt.Sprintf("u%d", i), fmt.Sprintf("u%d", parent), path(tid, fmt.Sprintf("/ou/u%d", i))))
	}
	user := func(email string) []identity.DirectoryUser {
		return []identity.DirectoryUser{{KeycloakUserID: "u", Email: email, Enabled: true}}
	}
	// Seed the tree directly (a sync of 10,111 new units is the import, not
	// the rebuild) and analyse it, as autovacuum does on a live database.
	exec(t, super, `INSERT INTO org_units (tenant_id, id, keycloak_group_id, name)
		SELECT $1, gen_random_uuid(), 'u' || i, 'u' || i FROM generate_series(0, $2::int - 1) i`, tid, total)
	exec(t, super, `UPDATE org_units c SET parent_id = p.id FROM org_units p
		WHERE c.tenant_id = $1 AND p.tenant_id = $1 AND p.keycloak_group_id = 'u' || (CASE
			WHEN substr(c.keycloak_group_id, 2)::int <= 10 THEN 0
			WHEN substr(c.keycloak_group_id, 2)::int <= 110 THEN (substr(c.keycloak_group_id, 2)::int - 11) % 10 + 1
			ELSE (substr(c.keycloak_group_id, 2)::int - 111) % 100 + 11 END)
		AND c.keycloak_group_id <> 'u0'`, tid)
	exec(t, super, "ANALYZE org_units")
	// Tree unchanged, closure empty (stale): the sync rebuilds it.
	dir.set(tid, identity.DirectoryChanges{OrgUnits: units, Users: user("u@a.test")})
	start := time.Now()
	if r := once(t, s); r.Err != nil {
		t.Fatal(r.Err)
	}
	rebuild := time.Since(start)
	// Tree unchanged, closure right: the sync only compares counts.
	dir.set(tid, identity.DirectoryChanges{OrgUnits: units, Users: user("u2@a.test")})
	start = time.Now()
	if r := once(t, s); r.Err != nil {
		t.Fatal(r.Err)
	}
	rows := count(t, super, "SELECT count(*) FROM org_unit_closure WHERE tenant_id = $1", tid)
	t.Logf("%d units, %d closure rows: sync that rebuilds %v; sync that finds the closure right %v (rebuild target <= 350ms)",
		total, rows, rebuild, time.Since(start))
	sameRows(t, "bench closure", closureOf(t, super, tid), wantClosure(units))
}

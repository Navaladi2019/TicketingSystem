//go:build integration

package dbtest

import (
	"io/fs"
	"maps"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/db/audited"
	"github.com/Navaladi2019/TicketingSystem/internal/db/grants"
)

// TICK-1032 (S31 fix; ADR-010-A2): org_unit_closure lists, for each live
// org unit, every live unit at or below it. Seeded identity ids equal the
// tenant id (seedIdentity): tenant X's root unit X has its self-row.

const orgUnitClosureMigration = "20261121050000_org_unit_closure.sql"

// beforeOrgUnitClosure leaves org_unit_closure out of both manifests when
// fsys lacks its migration, so a run that stops at an earlier version
// applies only grants on tables it has created.
func beforeOrgUnitClosure(fsys fs.FS, m *grants.Manifest, a *audited.Manifest) (*grants.Manifest, *audited.Manifest) {
	if _, err := fs.Stat(fsys, orgUnitClosureMigration); err == nil {
		return m, a
	}
	gm := *m
	gm.Roles = make(map[string]grants.Role, len(m.Roles))
	for name, r := range m.Roles {
		r.Tables = maps.Clone(r.Tables)
		delete(r.Tables, "org_unit_closure")
		gm.Roles[name] = r
	}
	am := *a
	am.Exempt = maps.Clone(a.Exempt)
	delete(am.Exempt, "org_unit_closure")
	return &gm, &am
}

// The table has the contract's keys, both FKs carry tenant_id with no
// action, the primary key is the only index, and the policies are
// org_units' app.tenant_id ones.
func TestTICK1032Schema(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	for name, want := range map[string]string{
		"org_unit_closure_pkey":            "PRIMARY KEY (tenant_id, ancestor_id, descendant_id)",
		"org_unit_closure_ancestor_fkey":   "FOREIGN KEY (tenant_id, ancestor_id) REFERENCES org_units(tenant_id, id)",
		"org_unit_closure_descendant_fkey": "FOREIGN KEY (tenant_id, descendant_id) REFERENCES org_units(tenant_id, id)",
	} {
		if got := queryString(t, super, "SELECT pg_get_constraintdef(oid) FROM pg_constraint WHERE conname = $1", name); got != want {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
		if !queryBool(t, super, "SELECT convalidated FROM pg_constraint WHERE conname = $1", name) {
			t.Errorf("%s is not validated", name)
		}
	}
	if got := queryString(t, super, "SELECT string_agg(indexrelid::regclass::text, ',') FROM pg_index WHERE indrelid = 'org_unit_closure'::regclass"); got != "org_unit_closure_pkey" {
		t.Errorf("org_unit_closure indexes = %s, want org_unit_closure_pkey only", got)
	}
	if !queryBool(t, super, "SELECT relrowsecurity AND relforcerowsecurity FROM pg_class WHERE oid = 'org_unit_closure'::regclass") {
		t.Error("org_unit_closure is not under ENABLE + FORCE RLS")
	}
	const pred = "(tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)"
	got := queryString(t, super, `SELECT string_agg(format('%s %s %s %s %s', polname, polcmd,
			(SELECT string_agg(rolname, ',' ORDER BY rolname) FROM pg_roles WHERE oid = ANY (polroles)),
			coalesce(pg_get_expr(polqual, polrelid), '-'), coalesce(pg_get_expr(polwithcheck, polrelid), '-')), E'\n' ORDER BY polname)
		FROM pg_policy WHERE polrelid = 'org_unit_closure'::regclass`)
	want := strings.Join([]string{
		"org_unit_closure_delete d nava_groupsync " + pred + " -",
		"org_unit_closure_insert a nava_groupsync - " + pred,
		"org_unit_closure_select r nava_app,nava_groupsync,nava_worker " + pred + " -",
	}, "\n")
	if got != want {
		t.Errorf("org_unit_closure policies:\n%s\nwant:\n%s", got, want)
	}
	if queryBool(t, super, "SELECT EXISTS (SELECT 1 FROM pg_policy WHERE 'nava_migrator'::regrole = ANY (polroles))") {
		t.Error("a nava_migrator policy outlived its migration (R26.3)")
	}
}

// Acceptance: nava_app and nava_worker only read the closure; group sync
// reads, inserts and deletes it, and never updates a row in place.
func TestTICK1032WriteGrants(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA}, nil)
	seedIdentity(t, super, tenantA)
	child := uuid.NewString()
	exec(t, super, insertUnit, tenantA, child, tenantA)

	insert := "INSERT INTO org_unit_closure (tenant_id, ancestor_id, descendant_id) VALUES ($1, $1, $2)"
	for _, role := range []string{"nava_app", "nava_worker"} {
		conn := asRole(t, role)
		_, err := execIn(conn, tenantA, insert, tenantA, child)
		wantSQLState(t, role+" inserts a closure row", err, "42501")
		_, err = execIn(conn, tenantA, "DELETE FROM org_unit_closure WHERE tenant_id = $1", tenantA)
		wantSQLState(t, role+" deletes closure rows", err, "42501")
		_, err = execIn(conn, tenantA, "UPDATE org_unit_closure SET descendant_id = descendant_id WHERE tenant_id = $1", tenantA)
		wantSQLState(t, role+" updates closure rows", err, "42501")
		if n, err := execIn(conn, tenantA, "SELECT descendant_id FROM org_unit_closure WHERE tenant_id = $1 AND ancestor_id = $1", tenantA); err != nil || n != 1 {
			t.Errorf("%s reads A's root's subtree: %d rows, %v; want 1, nil", role, n, err)
		}
	}

	sync := asRole(t, "nava_groupsync")
	_, err := execIn(sync, tenantA, "UPDATE org_unit_closure SET descendant_id = descendant_id WHERE tenant_id = $1", tenantA)
	wantSQLState(t, "nava_groupsync updates closure rows", err, "42501")
	if n, err := execIn(sync, tenantA, insert, tenantA, child); err != nil || n != 1 {
		t.Errorf("nava_groupsync inserts a closure row: %d rows, %v; want 1, nil", n, err)
	}
	if n, err := execIn(sync, tenantA, "DELETE FROM org_unit_closure WHERE tenant_id = $1", tenantA); err != nil || n != 2 {
		t.Errorf("nava_groupsync deletes A's closure: %d rows, %v; want 2, nil", n, err)
	}
}

// Acceptance: a closure row that joins one tenant's ancestor to another
// tenant's descendant, either way round, is refused by the foreign keys, for
// group sync under its own tenant and for the superuser (no RLS).
func TestTICK1032ForeignKeysCarryTenant(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, nil)
	seedIdentity(t, super, tenantA, tenantB)
	sync := asRole(t, "nava_groupsync")
	insert := "INSERT INTO org_unit_closure (tenant_id, ancestor_id, descendant_id) VALUES ($1, $2, $3)"
	for name, args := range map[string][]any{
		"A's ancestor, B's descendant":     {tenantA, tenantA, tenantB},
		"B's ancestor, A's descendant":     {tenantA, tenantB, tenantA},
		"B's self-row under A's tenant_id": {tenantA, tenantB, tenantB},
		"an ancestor that is no unit":      {tenantA, uuid.NewString(), tenantA},
		"a descendant that is no unit":     {tenantA, tenantA, uuid.NewString()},
	} {
		_, err := execIn(sync, tenantA, insert, args...)
		wantSQLState(t, "nava_groupsync: "+name, err, "23503")
		_, err = execIn(super, "", insert, args...)
		wantSQLState(t, "superuser: "+name, err, "23503")
	}
}

// closureRows reads every closure row of tenant as "ancestor>descendant",
// sorted, with ids mapped back to their names.
func closureRows(t *testing.T, tenant string, names map[string]string) []string {
	t.Helper()
	got := queryString(t, superNava(t), `SELECT coalesce(string_agg(ancestor_id::text || '>' || descendant_id::text, ','), '')
		FROM org_unit_closure WHERE tenant_id = $1`, tenant)
	var rows []string
	for _, r := range strings.Split(got, ",") {
		if r == "" {
			continue
		}
		a, d, _ := strings.Cut(r, ">")
		rows = append(rows, names[a]+">"+names[d])
	}
	sort.Strings(rows)
	return rows
}

// Acceptance (backfill): the migration fills the closure of every existing
// tenant from org_units' live rows. In a tree with a tombstoned intermediate
// unit, neither that unit nor anything below it has a row, live or not; a
// unit on a parent cycle has none either; and another tenant's tree is
// filled on its own. A re-run of the backfill gives the same rows.
//
// Tenant A:              Tenant B:
//
//	root                    rootB
//	├── left                └── onlyB
//	│   └── leftLeaf
//	├── mid (tombstoned)   cycle1 <-> cycle2 (no root, tenant A)
//	│   └── below
//	│       └── belowLeaf
//	└── deadLeaf (tombstoned)
func TestTICK1032Backfill(t *testing.T) {
	m := freshCluster(t)
	if err := migrateOnlyFS(t, m, migrationsBefore(t, orgUnitClosureMigration)); err != nil {
		t.Fatalf("migrate to before %s: %v", orgUnitClosureMigration, err)
	}
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, nil)
	if queryBool(t, super, "SELECT to_regclass('org_unit_closure') IS NOT NULL") {
		t.Fatal("org_unit_closure exists before its migration")
	}

	names := map[string]string{}
	id := func(name string) string {
		u := uuid.NewString()
		names[u] = name
		return u
	}
	root, left, leftLeaf, mid, below, belowLeaf, deadLeaf := id("root"), id("left"), id("leftLeaf"), id("mid"), id("below"), id("belowLeaf"), id("deadLeaf")
	cycle1, cycle2 := id("cycle1"), id("cycle2")
	rootB, onlyB := id("rootB"), id("onlyB")
	for _, u := range [][3]any{
		{tenantA, root, nil}, {tenantA, left, root}, {tenantA, leftLeaf, left},
		{tenantA, mid, root}, {tenantA, below, mid}, {tenantA, belowLeaf, below}, {tenantA, deadLeaf, root},
		{tenantA, cycle1, nil}, {tenantA, cycle2, cycle1},
		{tenantB, rootB, nil}, {tenantB, onlyB, rootB},
	} {
		exec(t, super, insertUnit, u[0], u[1], u[2])
	}
	exec(t, super, "UPDATE org_units SET parent_id = $2 WHERE tenant_id = $1 AND id = $3", tenantA, cycle2, cycle1)
	exec(t, super, tombstoneUnit, tenantA, mid)
	exec(t, super, tombstoneUnit, tenantA, deadLeaf)

	migrateAll(t, m)

	wantA := []string{
		"left>left", "left>leftLeaf", "leftLeaf>leftLeaf",
		"root>left", "root>leftLeaf", "root>root",
	}
	wantB := []string{"onlyB>onlyB", "rootB>onlyB", "rootB>rootB"}
	if got := closureRows(t, tenantA, names); !slices.Equal(got, wantA) {
		t.Errorf("tenant A's closure = %v, want %v", got, wantA)
	}
	if got := closureRows(t, tenantB, names); !slices.Equal(got, wantB) {
		t.Errorf("tenant B's closure = %v, want %v", got, wantB)
	}
	for _, gone := range []string{mid, below, belowLeaf, deadLeaf, cycle1, cycle2} {
		if queryBool(t, super, "SELECT EXISTS (SELECT 1 FROM org_unit_closure WHERE tenant_id = $1 AND $2 IN (ancestor_id, descendant_id))", tenantA, gone) {
			t.Errorf("%s has closure rows, want none", names[gone])
		}
	}

	// Re-run as nava_migrator: same rows, no policy left over. Anyone else
	// is refused.
	owner := asRole(t, m.Owner())
	if got := queryString(t, owner, "SELECT org_unit_closure_rebuild_all()::text"); got != "9" {
		t.Errorf("re-run filled %s rows, want 9", got)
	}
	if got := closureRows(t, tenantA, names); !slices.Equal(got, wantA) {
		t.Errorf("tenant A's closure after a re-run = %v, want %v", got, wantA)
	}
	if queryBool(t, super, "SELECT EXISTS (SELECT 1 FROM pg_policy WHERE 'nava_migrator'::regrole = ANY (polroles))") {
		t.Error("a nava_migrator policy outlived the re-run (R26.3)")
	}
	_, err := execIn(super, "", "SELECT org_unit_closure_rebuild_all()")
	wantSQLState(t, "the superuser runs the backfill", err, "42501")
}

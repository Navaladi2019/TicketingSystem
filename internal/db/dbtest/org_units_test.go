//go:build integration

package dbtest

import (
	"context"
	"io/fs"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/db/audited"
	"github.com/Navaladi2019/TicketingSystem/internal/db/grants"
)

// TICK-947 (S31 P11-14; ADR-010 D7, D8; PLAN-011 C5): org_unit_members,
// org_units.deleted_at, records.org_unit_id and the org-unit half of the
// acl_versions tenant-id trigger. Seeded identity ids equal the tenant id
// (seedIdentity): tenant X's unit X has principal X as its primary member.

const (
	orgUnitMembersMigration = "20261121000000_org_unit_members.sql"
	recordsOrgUnitMigration = "20261121000200_records_org_unit_id.sql"
)

// beforeOrgUnits returns the manifests as they were before TICK-947 when
// fsys lacks its migrations, so a run that stops at an earlier version
// applies only grants on tables and columns it has created.
func beforeOrgUnits(fsys fs.FS, m *grants.Manifest, a *audited.Manifest) (*grants.Manifest, *audited.Manifest) {
	_, errMembers := fs.Stat(fsys, orgUnitMembersMigration)
	_, errColumn := fs.Stat(fsys, recordsOrgUnitMigration)
	if errMembers == nil && errColumn == nil {
		return m, a
	}
	without := func(cols []string, col string) []string {
		return slices.DeleteFunc(slices.Clone(cols), func(c string) bool { return c == col })
	}
	gm := *m
	gm.Roles = make(map[string]grants.Role, len(m.Roles))
	for name, r := range m.Roles {
		tables := maps.Clone(r.Tables)
		if errColumn != nil {
			if privs, ok := tables["records"]; ok {
				p := make(map[string][]string, len(privs))
				for priv, cols := range privs {
					p[priv] = without(cols, "org_unit_id")
				}
				tables["records"] = p
			}
		}
		if errMembers != nil {
			delete(tables, "org_unit_members")
			switch name {
			case "nava_worker":
				delete(tables, "org_units")
			case "nava_groupsync":
				tables["org_units"] = map[string][]string{"SELECT": {}, "INSERT": {}, "UPDATE": {"parent_id", "name"}, "DELETE": {}}
			}
		}
		r.Tables = tables
		gm.Roles[name] = r
	}
	am := *a
	am.Audited = maps.Clone(a.Audited)
	am.Exempt = maps.Clone(a.Exempt)
	if errColumn != nil {
		if e, ok := am.Audited["records"]; ok {
			e.AuditedColumns = without(e.AuditedColumns, "org_unit_id")
			am.Audited["records"] = e
		}
	}
	if errMembers != nil {
		delete(am.Exempt, "org_unit_members")
	}
	return &gm, &am
}

const (
	tombstoneUnit = "UPDATE org_units SET deleted_at = now() WHERE tenant_id = $1 AND id = $2"
	insertUnit    = "INSERT INTO org_units (tenant_id, id, parent_id, keycloak_group_id, name) VALUES ($1::uuid, $2::uuid, $3::uuid, 'kc-' || $2::uuid::text, 'u')"
	insertUnitMem = "INSERT INTO org_unit_members (tenant_id, org_unit_id, principal_id, is_primary) VALUES ($1, $2, $3, $4)"
)

// The migrations leave the keys, indexes and constraints SP-2 designed, all
// valid.
func TestTICK947Schema(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	for _, c := range []string{"records_org_unit_fkey", "org_unit_members_unit_live_fkey", "org_unit_members_principal_fkey",
		"org_unit_members_live_check", "org_units_live_key"} {
		if !queryBool(t, super, "SELECT convalidated FROM pg_constraint WHERE conname = $1", c) {
			t.Errorf("%s is not validated", c)
		}
	}
	for name, want := range map[string]string{
		// TICK-1032 (SP-1d, critique B1): entity_definition_id joined the key.
		"records_org_unit":             "CREATE INDEX records_org_unit ON public.records USING btree (tenant_id, org_unit_id, entity_definition_id) WHERE (org_unit_id IS NOT NULL)",
		"org_unit_members_pkey":        "CREATE UNIQUE INDEX org_unit_members_pkey ON public.org_unit_members USING btree (tenant_id, org_unit_id, principal_id)",
		"org_unit_members_principal":   "CREATE INDEX org_unit_members_principal ON public.org_unit_members USING btree (tenant_id, principal_id, org_unit_id)",
		"org_unit_members_one_primary": "CREATE UNIQUE INDEX org_unit_members_one_primary ON public.org_unit_members USING btree (tenant_id, principal_id) WHERE is_primary",
	} {
		if got := queryString(t, super, "SELECT pg_get_indexdef(to_regclass($1))", name); got != want {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
		if !queryBool(t, super, "SELECT indisvalid FROM pg_index WHERE indexrelid = to_regclass($1)", name) {
			t.Errorf("%s is not valid", name)
		}
	}
	if got := queryString(t, super, "SELECT count(*)::text FROM pg_index WHERE indrelid = 'org_unit_members'::regclass"); got != "3" {
		t.Errorf("org_unit_members has %s indexes, want 3", got)
	}
	if got := queryString(t, super, "SELECT pg_get_constraintdef(oid) FROM pg_constraint WHERE conname = 'records_org_unit_fkey'"); got != "FOREIGN KEY (tenant_id, org_unit_id) REFERENCES org_units(tenant_id, id)" {
		t.Errorf("records_org_unit_fkey = %q, want NO ACTION on (tenant_id, org_unit_id)", got)
	}
	if queryBool(t, super, "SELECT EXISTS (SELECT 1 FROM pg_policy WHERE polname LIKE '%migrator%')") {
		t.Error("a migrator policy outlived its migration")
	}
	if queryBool(t, super, "SELECT EXISTS (SELECT 1 FROM pg_policy WHERE polname = 'org_units_delete')") {
		t.Error("org_units_delete still exists")
	}
}

// Acceptance: the groupsync role cannot delete an org unit, as itself or
// through a policy; it tombstones one, and revives it on the same id.
func TestTICK947GroupsyncCannotDeleteOrgUnits(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA}, nil)
	seedIdentity(t, super, tenantA)
	sync := asRole(t, "nava_groupsync")
	leaf := uuid.NewString()
	exec(t, super, insertUnit, tenantA, leaf, tenantA)

	_, err := execIn(sync, tenantA, "DELETE FROM org_units WHERE tenant_id = $1 AND id = $2", tenantA, leaf)
	wantSQLState(t, "nava_groupsync deletes a unit", err, "42501")
	if n, err := execIn(sync, tenantA, tombstoneUnit, tenantA, leaf); err != nil || n != 1 {
		t.Fatalf("nava_groupsync tombstones a unit: %d rows, %v; want 1, nil", n, err)
	}
	if got := queryString(t, super, "SELECT live::text FROM org_units WHERE tenant_id = $1 AND id = $2", tenantA, leaf); got != "false" {
		t.Errorf("org_units.live after the tombstone = %s, want false", got)
	}
	if n, err := execIn(sync, tenantA, "UPDATE org_units SET deleted_at = NULL WHERE tenant_id = $1 AND id = $2", tenantA, leaf); err != nil || n != 1 {
		t.Errorf("nava_groupsync revives a unit: %d rows, %v; want 1, nil", n, err)
	}
	for _, w := range []string{
		"UPDATE org_units SET id = gen_random_uuid() WHERE tenant_id = $1",
		"UPDATE org_units SET keycloak_group_id = 'x' WHERE tenant_id = $1",
		"UPDATE org_unit_members SET principal_id = principal_id WHERE tenant_id = $1",
		"UPDATE org_unit_members SET org_unit_id = org_unit_id WHERE tenant_id = $1",
	} {
		_, err := execIn(sync, tenantA, w, tenantA)
		wantSQLState(t, "nava_groupsync: "+w, err, "42501")
	}
	for _, role := range []string{"nava_app", "nava_worker"} {
		conn := asRole(t, role)
		for _, w := range []string{
			"UPDATE org_units SET deleted_at = now() WHERE tenant_id = $1",
			"INSERT INTO org_unit_members (tenant_id, org_unit_id, principal_id) VALUES ($1, $1, $1)",
			"DELETE FROM org_unit_members WHERE tenant_id = $1",
			"UPDATE org_unit_members SET is_primary = false WHERE tenant_id = $1",
		} {
			_, err := execIn(conn, tenantA, w, tenantA)
			wantSQLState(t, role+": "+w, err, "42501")
		}
	}
	// nava_worker reads the tree and the memberships, not the names.
	worker := asRole(t, "nava_worker")
	if _, err := execIn(worker, tenantA, "SELECT tenant_id, id, parent_id, deleted_at FROM org_units"); err != nil {
		t.Errorf("nava_worker reads the unit tree: %v", err)
	}
	if _, err := execIn(worker, tenantA, "SELECT tenant_id, org_unit_id, principal_id, is_primary FROM org_unit_members"); err != nil {
		t.Errorf("nava_worker reads memberships: %v", err)
	}
	_, err = execIn(worker, tenantA, "SELECT name FROM org_units")
	wantSQLState(t, "nava_worker reads unit names", err, "42501")
}

// D11 for org units, and D-11-6: no member in a tombstoned unit, no
// tombstone of a unit with a member, at most one primary unit per principal;
// the keys hold the tenant. For every role that can write, the superuser
// included.
func TestTICK947MembershipConstraints(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, nil)
	seedIdentity(t, super, tenantA, tenantB)
	gone, second := uuid.NewString(), uuid.NewString()
	exec(t, super, insertUnit, tenantA, gone, tenantA)
	exec(t, super, tombstoneUnit, tenantA, gone)
	exec(t, super, insertUnit, tenantA, second, tenantA)

	sync := asRole(t, "nava_groupsync")
	for who, run := range map[string]func(sql string, args ...any) error{
		"nava_groupsync": func(sql string, args ...any) error {
			return rolledBack(t, sync, func(tx pgx.Tx) error {
				if _, err := tx.Exec(context.Background(), "SELECT set_config('app.tenant_id', $1, true)", tenantA); err != nil {
					return err
				}
				_, err := tx.Exec(context.Background(), sql, args...)
				return err
			})
		},
		"superuser": func(sql string, args ...any) error {
			return rolledBack(t, super, func(tx pgx.Tx) error {
				_, err := tx.Exec(context.Background(), sql, args...)
				return err
			})
		},
	} {
		t.Run(who, func(t *testing.T) {
			wantSQLState(t, "a member of a tombstoned unit", run(insertUnitMem, tenantA, gone, tenantA, false), "23503")
			wantSQLState(t, "tombstone a unit with a member", run(tombstoneUnit, tenantA, tenantA), "23503")
			wantSQLState(t, "a member with live = false",
				run("INSERT INTO org_unit_members (tenant_id, org_unit_id, principal_id, live) VALUES ($1, $2, $1, false)", tenantA, second), "23514")
			wantSQLState(t, "a second primary unit", run(insertUnitMem, tenantA, second, tenantA, true), "23505")
			wantSQLState(t, "A's membership in B's unit", run(insertUnitMem, tenantA, tenantB, tenantA, false), "23503")
			wantSQLState(t, "A's membership for B's principal", run(insertUnitMem, tenantA, second, tenantB, false), "23503")
			if err := run(insertUnitMem, tenantA, second, tenantA, false); err != nil {
				t.Errorf("a second, non-primary unit: %v", err)
			}
		})
	}
}

// Acceptance: a record can reference only its own tenant's unit, by any
// writer; a tombstoned unit of its own tenant stays a valid reference (units
// are never deleted, D7). nava_app and nava_worker set it on create and
// cannot change it afterwards (TICK-1120).
func TestTICK947RecordOrgUnitIsTenantScoped(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, nil)
	seedIdentity(t, super, tenantA, tenantB)
	gone := uuid.NewString()
	exec(t, super, insertUnit, tenantA, gone, nil)
	exec(t, super, tombstoneUnit, tenantA, gone)

	insert := withRecordEntry("INSERT INTO records (tenant_id, id, org_unit_id) VALUES ($1, $2, $3)")
	update := withRecordEntry("UPDATE records SET org_unit_id = $2 WHERE tenant_id = $1 AND id = $1")
	for _, as := range []struct {
		name   string
		conn   *pgx.Conn
		tenant string
	}{{"superuser", super, ""}, {"nava_app", asRole(t, "nava_app"), tenantA}, {"nava_worker", asRole(t, "nava_worker"), tenantA}} {
		_, err := execIn(as.conn, as.tenant, insert, tenantA, uuid.NewString(), tenantB)
		wantSQLState(t, as.name+" creates A's record in B's unit", err, "23503")
		_, err = execIn(as.conn, as.tenant, insert, tenantA, uuid.NewString(), uuid.NewString())
		wantSQLState(t, as.name+" creates A's record in a unit of no tenant", err, "23503")
	}
	_, err := execIn(super, "", update, tenantA, tenantB)
	wantSQLState(t, "superuser moves A's record to B's unit", err, "23503")
	for _, unit := range []any{tenantA, nil, gone} {
		if n, err := execIn(super, "", update, tenantA, unit); err != nil || n != 1 {
			t.Errorf("superuser moves A's record to %v: %d rows, %v; want 1, nil", unit, n, err)
		}
	}

	// The record is now in gone, so each unit below would change it.
	const orgUnit = "SELECT coalesce(org_unit_id::text, 'NULL') FROM records WHERE tenant_id = $1 AND id = $1"
	for _, role := range []string{"nava_app", "nava_worker"} {
		for priv, want := range map[string]bool{"SELECT": true, "INSERT": true, "UPDATE": false} {
			if got := queryBool(t, super, "SELECT has_column_privilege($1, 'records', 'org_unit_id', $2)", role, priv); got != want {
				t.Errorf("%s has %s on records.org_unit_id = %v, want %v", role, priv, got, want)
			}
		}
		conn := asRole(t, role)
		for what, unit := range map[string]any{"its own unit": tenantA, "B's unit": tenantB, "no unit": nil} {
			var seen string
			if err := inTenantTx(conn, tenantA, func(tx pgx.Tx) error {
				return tx.QueryRow(context.Background(), orgUnit, tenantA).Scan(&seen)
			}); err != nil || seen != gone {
				t.Fatalf("%s reads A's record: unit %q, %v; want %q, nil", role, seen, err, gone)
			}
			_, err := execIn(conn, tenantA, update, tenantA, unit)
			wantSQLState(t, role+" moves A's record to "+what, err, "42501")
			if got := queryString(t, super, orgUnit, tenantA); got != gone {
				t.Errorf("after %s moves A's record to %s: org_unit_id = %s, want %s", role, what, got, gone)
				exec(t, super, update, tenantA, gone)
			}
		}
		// Control: the same row, tenant and audit entry, another column.
		if n, err := execIn(conn, tenantA, withRecordEntry("UPDATE records SET updated_at = now() WHERE tenant_id = $1 AND id = $1"), tenantA); err != nil || n != 1 {
			t.Errorf("control: %s updates updated_at of A's record: %d rows, %v; want 1, nil", role, n, err)
		}
	}

	// org_unit_id is audited: moving it without the record's audit entry
	// cannot commit (CI-A3, the commit check's UPDATE OF list). Only the
	// superuser can still update it.
	_, err = execIn(super, "", "UPDATE records SET org_unit_id = $1 WHERE tenant_id = $1 AND id = $1", tenantA)
	wantSQLState(t, "superuser moves org_unit_id with no audit entry", err, "23514")
}

// TICK-1120: no role of db/grants.yaml but the owner may update
// records.org_unit_id, and a deployed database that still holds the grant
// loses it on the next migrate run, with no migration of its own.
func TestTICK1120NoRoleUpdatesRecordOrgUnit(t *testing.T) {
	m := migratedCluster(t)
	super := superNava(t)
	const updates = "SELECT has_column_privilege($1, 'records', 'org_unit_id', 'UPDATE')"
	check := func(when string) {
		t.Helper()
		for role := range m.Roles {
			if role != m.Owner() && queryBool(t, super, updates, role) {
				t.Errorf("%s: %s may update records.org_unit_id", when, role)
			}
		}
	}
	check("fresh database")

	mig := connect(t, roleConfig(t, m.Owner(), "nava"))
	exec(t, mig, "GRANT UPDATE (org_unit_id) ON records TO nava_app, nava_worker")
	for _, role := range []string{"nava_app", "nava_worker"} {
		if !queryBool(t, super, updates, role) {
			t.Fatalf("%s did not get the grant to lose", role)
		}
	}
	res := migrateAll(t, m)
	check("after the migrate run")
	want := []string{
		"UPDATE column (org_unit_id) ON public.records TO nava_app",
		"UPDATE column (org_unit_id) ON public.records TO nava_worker",
	}
	if got := slices.Sorted(slices.Values(res.Grants.Revoked)); !slices.Equal(got, want) || len(res.Grants.Granted) != 0 {
		t.Errorf("migrate run revoked %q and granted %q, want %q and nothing", got, res.Grants.Granted, want)
	}
}

// Acceptance (D-11-11, AR-S30-01): an under id that is not a live org unit of
// the row's tenant is refused with foreign_id and its pointer, by the
// superuser (no RLS) and by nava_app. Another tenant's unit, a tombstoned own
// unit and an id of no unit get the same answer. (The owner is refused by
// FORCE RLS before the trigger runs, TICK-940 security review.)
func TestTICK947UnderForeignIDs(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, nil)
	seedACL(t, super, tenantA, tenantB) // seedIdentity included
	gone := uuid.NewString()
	exec(t, super, insertUnit, tenantA, gone, nil)
	exec(t, super, tombstoneUnit, tenantA, gone)
	app := asRole(t, "nava_app")

	under := func(of any) string {
		return doc2(nil, []any{
			policy("own", map[string]any{"under": map[string]any{"of": []string{tenantA}}}),
			policy("other", map[string]any{"all": []any{
				map[string]any{"member_of": map[string]any{"users": []string{tenantA}}},
				map[string]any{"under": map[string]any{"of": of}}}})})
	}
	version := 10
	for name, id := range map[string]string{
		"tenant B's unit":          tenantB,
		"a tombstoned own unit":    gone,
		"an id of no unit":         "0c000000-0000-4000-8000-0000000005c9",
		"tenant B's unit, by caps": "0B000000-0000-4000-8000-00000000000B",
	} {
		for _, as := range []struct {
			name   string
			conn   *pgx.Conn
			tenant string
		}{{"superuser", super, ""}, {"nava_app", app, tenantA}} {
			version++
			what := as.name + ": " + name
			_, err := execIn(as.conn, as.tenant, insertACLShape, tenantA, version, under([]string{tenantA, id}), `[]`)
			if pe := refusal(t, what, err); pe != nil {
				want := aclDetail{"rules", "foreign_id", "/policies/1/when/all/1/under/of/1"}
				if d := detailOf(t, what, pe); d != want {
					t.Errorf("%s: DETAIL %+v, want %+v", what, d, want)
				}
			}
		}
	}
	// The own tenant's live unit, and "caller", pass.
	for _, of := range []any{[]string{tenantA}, "caller"} {
		version++
		if _, err := execIn(app, tenantA, insertACLShape, tenantA, version, under(of), `[]`); err != nil {
			t.Errorf("under of %v: %v", of, err)
		}
	}
	// acl2_id_pointers names under ids as org units.
	if got := queryString(t, super, "SELECT string_agg(kind || ' ' || pointer, ',' ORDER BY pointer) FROM acl2_id_pointers($1::text::jsonb)",
		under([]string{tenantB})); got != "org_unit /policies/0/when/under/of/0,user /policies/1/when/all/0/member_of/users/0,org_unit /policies/1/when/all/1/under/of/0" {
		t.Errorf("acl2_id_pointers = %q", got)
	}
}

// AR-S30-01's one-off check: 20261121000100 refuses to apply while a stored
// row names an under id that is not a live org unit of its own tenant, and
// leaves no migrator policy behind; once the row is gone it applies.
func TestTICK947MigrationRefusesStoredForeignUnder(t *testing.T) {
	const trigger = "20261121000100_acl_tenant_ids_org_units.sql"
	m := freshCluster(t)
	if err := migrateOnlyFS(t, m, migrationsBefore(t, trigger)); err != nil {
		t.Fatalf("migrate to before %s: %v", trigger, err)
	}
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, nil)
	seedACL(t, super, tenantA, tenantB) // seedIdentity included: unit X has id X
	// Stored before the trigger checked org units: tenant A's row names B's unit.
	exec(t, super, insertACLShape, tenantA, 2, doc2(nil, []any{
		policy("p", map[string]any{"under": map[string]any{"of": []string{tenantA, tenantB}}})}), `[]`)
	err := migrateOnlyFS(t, m, migrationsThrough(t, trigger))
	if err == nil || !strings.Contains(err.Error(), "AR-S30-01") {
		t.Fatalf("migrate %s over a foreign under id = %v, want the AR-S30-01 refusal", trigger, err)
	}
	// The HINT never sends the operator to delete access (TICK-1133).
	if pe := refusal(t, "migrate "+trigger, err); pe != nil {
		const want = "Do not delete acl_versions rows: the newest row is the access in force. " +
			"List the ids with the AR-S30-01 query in docs/data/entity-versions-org-unit-id-reserved.md " +
			"and ask the platform operator's support before migrating again."
		if pe.Hint != want {
			t.Errorf("HINT = %q, want %q", pe.Hint, want)
		}
	}
	// What the refusal leaves (docs/data/entity-versions-org-unit-id-reserved.md):
	// the guard and 20261121000000 applied, nothing after, the row as it was.
	if got := queryString(t, super, "SELECT string_agg(version_id::text, ',' ORDER BY version_id) FROM goose_db_version WHERE is_applied AND version_id >= 20261120235900"); got != "20261120235900,20261121000000" {
		t.Errorf("migrations applied from the guard on = %s, want the guard and 20261121000000", got)
	}
	if got := queryString(t, super, "SELECT string_agg(version::text, ',' ORDER BY version) FROM acl_versions WHERE tenant_id = $1", tenantA); got != "1,2" {
		t.Errorf("tenant A's access versions after the refusal = %s, want 1,2", got)
	}
	if queryBool(t, super, "SELECT EXISTS (SELECT 1 FROM pg_policy WHERE polname LIKE '%migrator%')") {
		t.Error("a migrator policy outlived the failed migration")
	}
	if queryBool(t, super, "SELECT pg_get_functiondef('acl2_tenant_ids'::regproc) LIKE '%org_units%'") {
		t.Error("the failed migration replaced the trigger function")
	}
	// Test setup only, to reach the passing case; never a repair.
	exec(t, super, "ALTER TABLE acl_versions DISABLE TRIGGER USER")
	exec(t, super, "DELETE FROM acl_versions WHERE tenant_id = $1 AND version = 2", tenantA)
	exec(t, super, "ALTER TABLE acl_versions ENABLE TRIGGER USER")
	// The own tenant's live unit is no finding.
	exec(t, super, insertACLShape, tenantA, 3, doc2(nil, []any{
		policy("p", map[string]any{"under": map[string]any{"of": []string{tenantA}}})}), `[]`)
	migrateAll(t, m)
}

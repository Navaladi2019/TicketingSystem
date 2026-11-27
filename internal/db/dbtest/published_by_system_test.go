//go:build integration

package dbtest

import (
	"context"
	"io/fs"
	"maps"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/db/grants"
)

const publishedBySystemMigration = "20261127110000_entity_versions_published_by_system.sql"

// insertVersionSystem is insertVersion with published_by_system $9: tenant
// $1, version id $2, type $3, version $4, schema $5, schema_sha256 $6, compat
// $7, published_by $8.
const insertVersionSystem = `WITH v AS (INSERT INTO entity_definition_versions
		(tenant_id, id, definition_id, version, schema, schema_sha256, fixed_columns, states, compat, published_by, published_by_system)
		VALUES ($1, $2, $3, $4, $5::text::json, $6, '{}', '{}', $7, $8, $9)) ` + configEntry + `
		VALUES ($1, gen_random_uuid(), 'config', CASE WHEN $4 = 1 THEN 'create' ELSE 'publish' END,
			'entity_definition', $3, $4, 'system', 'system', $6)`

// P7-12a (TICK-1172): a version after v1 names exactly one publisher, a
// principal (published_by) or the system (published_by_system =
// 'onboarding'). nava_onboard publishes a provisioning tenant's v2 with the
// system publisher; neither publisher, both, or an unknown system word is
// refused (23514), as nava_onboard and as the superuser. nava_app cannot
// write published_by_system at all (42501, its column-level INSERT grant),
// so a version the API publishes always names its principal.
func TestP712aPublishedBySystem(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA}, []string{tenantX})
	seedIdentity(t, super, tenantA)
	for _, c := range []string{"entity_definition_versions_published_by_check", "entity_definition_versions_published_by_system_check"} {
		if !queryBool(t, super, `SELECT convalidated FROM pg_constraint
			WHERE conrelid = 'entity_definition_versions'::regclass AND conname = $1`, c) {
			t.Errorf("%s is not validated after the migrations", c)
		}
	}
	// nava_onboard's table-level INSERT covers the column; nava_app may
	// insert every column but published_by_system, so its list keeps up with
	// the table. RLS stays forced.
	if !queryBool(t, super, `SELECT has_column_privilege('nava_onboard', 'entity_definition_versions', 'published_by_system', 'INSERT')`) {
		t.Error("nava_onboard cannot insert published_by_system")
	}
	if got := queryString(t, super, `SELECT coalesce(string_agg(attname, ', ' ORDER BY attnum), '') FROM pg_attribute
		WHERE attrelid = 'entity_definition_versions'::regclass AND attnum > 0 AND NOT attisdropped
		  AND NOT has_column_privilege('nava_app', attrelid, attnum, 'INSERT')`); got != "published_by_system" {
		t.Errorf("columns nava_app cannot insert = %q, want only published_by_system", got)
	}
	if !queryBool(t, super, `SELECT relrowsecurity AND relforcerowsecurity FROM pg_class WHERE oid = 'entity_definition_versions'::regclass`) {
		t.Error("entity_definition_versions no longer forces RLS")
	}

	ctx := context.Background()
	onboard := asRole(t, "nava_onboard")
	err := inTenantTx(onboard, tenantX, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, insertType, tenantX, "ticket", "TCK"); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, insertVersion, tenantX, tenantX, tenantX, 1, testSchema, schemaSum(testSchema), "initial", nil); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, insertVersionSystem, tenantX, tenantB, tenantX, 2, testSchema, schemaSum(testSchema), "compatible", nil, "onboarding")
		return err
	})
	if err != nil {
		t.Fatalf("nava_onboard v2 with published_by_system 'onboarding': %v, want accepted", err)
	}
	if got := queryString(t, super, `SELECT coalesce(published_by::text, 'null') || ' ' || published_by_system
		FROM entity_definition_versions WHERE tenant_id = $1 AND version = 2`, tenantX); got != "null onboarding" {
		t.Errorf("stored v2 publisher = %q, want %q", got, "null onboarding")
	}
	for _, c := range []struct {
		what   string
		by     any
		system any
	}{
		{"v3 with neither publisher", nil, nil},
		{"v3 with both publishers", tenantA, "onboarding"},
		{"v3 with an unknown system publisher", nil, "import"},
	} {
		err := inTenantTx(onboard, tenantX, func(tx pgx.Tx) error {
			_, err := tx.Exec(ctx, insertVersionSystem, tenantX, tenantA, tenantX, 3, testSchema, schemaSum(testSchema), "compatible", c.by, c.system)
			return err
		})
		wantSQLState(t, "nava_onboard "+c.what, err, "23514")
		_, err = super.Exec(ctx, insertVersionSystem, tenantX, tenantA, tenantX, 3, testSchema, schemaSum(testSchema), "compatible", c.by, c.system)
		wantSQLState(t, c.what+" as the superuser", err, "23514")
	}

	app := asRole(t, "nava_app")
	exec(t, super, insertType, tenantA, "ticket", "TCK")
	if _, err := execIn(app, tenantA, insertVersion, tenantA, tenantA, tenantA, 1, testSchema, schemaSum(testSchema), "initial", nil); err != nil {
		t.Fatalf("nava_app v1 with no publisher: %v, want accepted", err)
	}
	_, err = execIn(app, tenantA, insertVersionSystem, tenantA, tenantB, tenantA, 2, testSchema, schemaSum(testSchema), "compatible", nil, "onboarding")
	wantSQLState(t, "nava_app v2 with published_by_system 'onboarding'", err, "42501")
	_, err = execIn(app, tenantA, insertVersionSystem, tenantA, tenantB, tenantA, 2, testSchema, schemaSum(testSchema), "compatible", tenantA, nil)
	wantSQLState(t, "nava_app naming published_by_system at all", err, "42501")
	if _, err := execIn(app, tenantA, insertVersion, tenantA, tenantB, tenantA, 2, testSchema, schemaSum(testSchema), "compatible", tenantA); err != nil {
		t.Fatalf("nava_app v2 published by A's principal: %v, want accepted", err)
	}
}

// P7-12a (TICK-1172): the migration validates both CHECKs on a database that
// already holds onboarded tenants: v1 rows seeded with no publisher and a
// later version published by a principal. Every row survives unchanged, with
// published_by_system NULL.
func TestP712aPublishedBySystemValidatesOverSeededTenants(t *testing.T) {
	m := freshCluster(t)
	if err := migrateOnlyFS(t, m, migrationsBefore(t, publishedBySystemMigration)); err != nil {
		t.Fatalf("migrate to before %s: %v", publishedBySystemMigration, err)
	}
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, []string{tenantX})
	seedIdentity(t, super, tenantA)
	for _, id := range []string{tenantA, tenantB, tenantX} {
		exec(t, super, insertType, id, "ticket", "TCK")
		exec(t, super, insertVersion, id, id, id, 1, testSchema, schemaSum(testSchema), "initial", nil)
	}
	exec(t, super, insertVersion, tenantA, tenantB, tenantA, 2, testSchema, schemaSum(testSchema), "compatible", tenantA)

	migrateAll(t, m)
	for _, c := range []string{"entity_definition_versions_published_by_check", "entity_definition_versions_published_by_system_check"} {
		if !queryBool(t, super, `SELECT convalidated FROM pg_constraint
			WHERE conrelid = 'entity_definition_versions'::regclass AND conname = $1`, c) {
			t.Fatalf("%s is not validated over the seeded tenants", c)
		}
	}
	if got := queryString(t, super, `SELECT count(*)::text || ' ' || count(published_by)::text || ' ' || count(published_by_system)::text
		FROM entity_definition_versions`); got != "4 1 0" {
		t.Errorf("rows, published_by, published_by_system after the migration = %q, want the 4 seeded, 1 with a principal, none with a system publisher", got)
	}
}

// beforePublishedBySystem gives nava_app back its table-level INSERT on
// entity_definition_versions when fsys stops before 20261127110000, as
// db/grants.yaml did before P7-12a. The column list names columns an older
// schema does not have yet (number_format_id and the other number_* columns
// arrive in 20261113210200), and the grants applier refuses a listed column
// that does not exist, so a run that stops earlier could not apply the
// manifest. With every migration in fsys the manifest is unchanged.
func beforePublishedBySystem(fsys fs.FS, m *grants.Manifest) *grants.Manifest {
	if _, err := fs.Stat(fsys, publishedBySystemMigration); err == nil {
		return m
	}
	const role, table = "nava_app", "entity_definition_versions"
	r, ok := m.Roles[role]
	if !ok {
		return m
	}
	privs, ok := r.Tables[table]
	if !ok {
		return m
	}
	gm := *m
	gm.Roles = maps.Clone(m.Roles)
	p := maps.Clone(privs)
	p["INSERT"] = []string{}
	r.Tables = maps.Clone(r.Tables)
	r.Tables[table] = p
	gm.Roles[role] = r
	return &gm
}

// P7-12a (TICK-1172): the grants manifest applies on the schema just before
// 20261127110000, where nava_app's INSERT is table-level and covers every
// column, and an upgrade from there converges it to the column list: the
// table-level grant is revoked, and published_by_system is the one column
// nava_app cannot insert. Integration CI on PR #938 found the column list
// refused on older schemas; this applies the manifest there on purpose.
func TestP712aGrantsApplyBeforeAndAfterPublishedBySystem(t *testing.T) {
	m := freshCluster(t)
	if _, err := migrateFS(t, m, migrationsBefore(t, publishedBySystemMigration)); err != nil {
		t.Fatalf("migrate with grants to before %s: %v", publishedBySystemMigration, err)
	}
	super := superNava(t)
	if !queryBool(t, super, `SELECT has_table_privilege('nava_app', 'entity_definition_versions', 'INSERT')`) {
		t.Fatal("before the migration nava_app has no table-level INSERT on entity_definition_versions")
	}

	migrateAll(t, m)
	if queryBool(t, super, `SELECT has_table_privilege('nava_app', 'entity_definition_versions', 'INSERT')`) {
		t.Error("after the migration nava_app still has table-level INSERT on entity_definition_versions")
	}
	if got := queryString(t, super, `SELECT coalesce(string_agg(attname, ', ' ORDER BY attnum), '') FROM pg_attribute
		WHERE attrelid = 'entity_definition_versions'::regclass AND attnum > 0 AND NOT attisdropped
		  AND NOT has_column_privilege('nava_app', attrelid, attnum, 'INSERT')`); got != "published_by_system" {
		t.Errorf("after the upgrade, columns nava_app cannot insert = %q, want only published_by_system", got)
	}
}

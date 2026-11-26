//go:build integration

package dbtest

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
)

const publishedBySystemMigration = "20261126170000_entity_versions_published_by_system.sql"

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
// refused (23514), for the superuser too.
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
	// No grant change: nava_onboard's table-level INSERT covers the column,
	// and RLS stays forced.
	if !queryBool(t, super, `SELECT has_column_privilege('nava_onboard', 'entity_definition_versions', 'published_by_system', 'INSERT')`) {
		t.Error("nava_onboard cannot insert published_by_system")
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

	app := asRole(t, "nava_app")
	exec(t, super, insertType, tenantA, "ticket", "TCK")
	if _, err := execIn(app, tenantA, insertVersion, tenantA, tenantA, tenantA, 1, testSchema, schemaSum(testSchema), "initial", nil); err != nil {
		t.Fatalf("v1 with no publisher: %v, want accepted", err)
	}
	for _, c := range []struct {
		what   string
		by     any
		system any
	}{
		{"v2 with neither publisher", nil, nil},
		{"v2 with both publishers", tenantA, "onboarding"},
		{"v2 with an unknown system publisher", nil, "import"},
	} {
		_, err := execIn(app, tenantA, insertVersionSystem, tenantA, tenantB, tenantA, 2, testSchema, schemaSum(testSchema), "compatible", c.by, c.system)
		wantSQLState(t, c.what, err, "23514")
		_, err = super.Exec(ctx, insertVersionSystem, tenantA, tenantB, tenantA, 2, testSchema, schemaSum(testSchema), "compatible", c.by, c.system)
		wantSQLState(t, c.what+" as the superuser", err, "23514")
	}
	if _, err := execIn(app, tenantA, insertVersionSystem, tenantA, tenantB, tenantA, 2, testSchema, schemaSum(testSchema), "compatible", tenantA, nil); err != nil {
		t.Fatalf("v2 published by A's principal: %v, want accepted", err)
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
		exec(t, super, insertTypeRaw, id, "ticket", "TCK")
		exec(t, super, insertVersionRaw, id, id, id, 1, testSchema, schemaSum(testSchema), "initial", nil)
	}
	exec(t, super, insertVersionRaw, tenantA, tenantB, tenantA, 2, testSchema, schemaSum(testSchema), "compatible", tenantA)

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

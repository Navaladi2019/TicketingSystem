//go:build integration

package dbtest

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
)

const publishedByCheckMigration = "20261101130000_entity_versions_published_by_check.sql"

// P3-06c (TICK-383): a version after v1 with published_by NULL is refused by
// entity_definition_versions_published_by_check (23514); a v1 with NULL, as
// onboarding seeds it, and a v2 that names its publisher are accepted.
func TestP306cPublishedByRequiredAfterV1(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA}, []string{tenantX})
	seedIdentity(t, super, tenantA)
	if !queryBool(t, super, `SELECT convalidated FROM pg_constraint
		WHERE conrelid = 'entity_definition_versions'::regclass AND conname = 'entity_definition_versions_published_by_check'`) {
		t.Error("entity_definition_versions_published_by_check is not validated after the migrations")
	}

	// v1 with NULL, as nava_onboard seeds a provisioning tenant.
	onboard := asRole(t, "nava_onboard")
	err := inTenantTx(onboard, tenantX, func(tx pgx.Tx) error {
		ctx := context.Background()
		if _, err := tx.Exec(ctx, insertType, tenantX, "ticket", "TCK"); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, insertVersion, tenantX, tenantX, tenantX, 1, testSchema, schemaSum(testSchema), "initial", nil)
		return err
	})
	if err != nil {
		t.Fatalf("v1 with published_by NULL: %v, want accepted", err)
	}

	app := asRole(t, "nava_app")
	exec(t, super, insertType, tenantA, "ticket", "TCK")
	if _, err := execIn(app, tenantA, insertVersion, tenantA, tenantA, tenantA, 1, testSchema, schemaSum(testSchema), "initial", nil); err != nil {
		t.Fatalf("nava_app v1 with published_by NULL: %v, want accepted", err)
	}
	_, err = execIn(app, tenantA, insertVersion, tenantA, tenantB, tenantA, 2, testSchema, schemaSum(testSchema), "compatible", nil)
	wantSQLState(t, "v2 with published_by NULL", err, "23514")
	_, err = super.Exec(context.Background(), insertVersion, tenantA, tenantB, tenantA, 2, testSchema, schemaSum(testSchema), "compatible", nil)
	wantSQLState(t, "v2 with published_by NULL as the superuser", err, "23514")
	if _, err := execIn(app, tenantA, insertVersion, tenantA, tenantB, tenantA, 2, testSchema, schemaSum(testSchema), "compatible", tenantA); err != nil {
		t.Fatalf("v2 published by A's principal: %v, want accepted", err)
	}
}

// P3-06c (TICK-383): the migration validates cleanly on a database that
// already holds onboarded tenants: v1 rows seeded with published_by NULL, and
// a later version published by a principal.
func TestP306cPublishedByCheckValidatesOverSeededTenants(t *testing.T) {
	m := freshCluster(t)
	if err := migrateOnlyFS(t, m, migrationsBefore(t, publishedByCheckMigration)); err != nil {
		t.Fatalf("migrate to before %s: %v", publishedByCheckMigration, err)
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
	if !queryBool(t, super, `SELECT convalidated FROM pg_constraint
		WHERE conrelid = 'entity_definition_versions'::regclass AND conname = 'entity_definition_versions_published_by_check'`) {
		t.Fatal("entity_definition_versions_published_by_check is not validated over the seeded tenants")
	}
	if got := queryString(t, super, "SELECT count(*)::text FROM entity_definition_versions"); got != "4" {
		t.Errorf("entity_definition_versions holds %s rows after the migration, want the 4 seeded", got)
	}
}

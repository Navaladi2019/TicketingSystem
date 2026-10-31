//go:build integration

package dbtest

import (
	"bytes"
	"context"
	"crypto/sha256"
	"testing"

	"github.com/jackc/pgx/v5"
)

// P3-02 (TICK-340): entity_definitions and entity_definition_versions under
// FORCE RLS. nava_app creates types and publishes versions, nava_worker reads
// them, nava_onboard seeds them while provisioning, and no role updates or
// deletes a version (D-3-4).

const (
	// insertType has the type's id and tenant as $1, its key as $2 and its
	// number_prefix as $3.
	insertType = `INSERT INTO entity_definitions (tenant_id, id, key, name, number_prefix, current_version)
		VALUES ($1, $1, $2, 'Ticket', $3, 1)`
	insertTypeNewID = `INSERT INTO entity_definitions (tenant_id, id, key, name, number_prefix, current_version)
		VALUES ($1, gen_random_uuid(), $2, 'Ticket', $3, 1)`
	// insertVersion: tenant $1, version id $2, type $3, version $4, schema
	// bytes $5 (as text, so nothing re-encodes them), schema_sha256 $6,
	// compat $7, published_by $8.
	insertVersion = `INSERT INTO entity_definition_versions
		(tenant_id, id, definition_id, version, schema, schema_sha256, fixed_columns, states, compat, published_by)
		VALUES ($1, $2, $3, $4, $5::text::json, $6, '{}', '{}', $7, $8)`
	testSchema = `{"type": "object"}`
)

func schemaSum(s string) []byte {
	sum := sha256.Sum256([]byte(s))
	return sum[:]
}

// seedEntityTypes writes, as the superuser, one principal (through
// seedIdentity), one type and its version 1 for each tenant, each with id =
// the tenant id, and the same key and number_prefix in every tenant.
// seedTenants must have created the tenants.
func seedEntityTypes(t *testing.T, super *pgx.Conn, tenants ...string) {
	t.Helper()
	seedIdentity(t, super, tenants...)
	for _, id := range tenants {
		exec(t, super, insertType, id, "ticket", "TCK")
		exec(t, super, insertVersion, id, id, id, 1, testSchema, schemaSum(testSchema), "initial", id)
	}
}

// Acceptance: with tenant A set, neither nava_app nor nava_worker sees a row
// of tenant B in either table; with no tenant, or an empty one, they see 0
// rows.
func TestP302TenantRowsInvisible(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, nil)
	seedEntityTypes(t, super, tenantA, tenantB)

	for _, role := range []string{"nava_app", "nava_worker"} {
		conn := asRole(t, role)
		for _, table := range []string{"entity_definitions", "entity_definition_versions"} {
			for _, mode := range []string{"A", "empty", "unset"} {
				var own, other int
				err := inTenantTx(conn, "", func(tx pgx.Tx) error {
					if mode != "unset" {
						v := map[string]string{"A": tenantA, "empty": ""}[mode]
						if _, err := tx.Exec(context.Background(), "SELECT set_config('app.tenant_id', $1, true)", v); err != nil {
							return err
						}
					}
					return tx.QueryRow(context.Background(),
						"SELECT count(*) FILTER (WHERE tenant_id = $1), count(*) FILTER (WHERE tenant_id <> $1) FROM "+table,
						tenantA).Scan(&own, &other)
				})
				if err != nil {
					t.Fatalf("%s reads %s (tenant %s): %v", role, table, mode, err)
				}
				wantOwn := 0
				if mode == "A" {
					wantOwn = 1
				}
				if own != wantOwn || other != 0 {
					t.Errorf("%s reads %s with tenant %s: own %d, other tenants %d; want %d, 0", role, table, mode, own, other, wantOwn)
				}
			}
		}
	}

	// From tenant A, nava_app neither writes a row for B nor moves B's type.
	app := asRole(t, "nava_app")
	_, err := execIn(app, tenantA, insertTypeNewID, tenantB, "incident", "INC")
	wantSQLState(t, "nava_app in tenant A creates a type for B", err, "42501")
	_, err = execIn(app, tenantA, insertVersion, tenantB, tenantA, tenantB, 2, testSchema, schemaSum(testSchema), "compatible", tenantB)
	wantSQLState(t, "nava_app in tenant A publishes a version for B", err, "42501")
	if n, err := execIn(app, tenantA, "UPDATE entity_definitions SET current_version = 2 WHERE tenant_id = $1", tenantB); err != nil || n != 0 {
		t.Errorf("nava_app in tenant A moves B's current_version: %d rows, %v; want 0, nil", n, err)
	}
}

// Acceptance (D-3-4): no runtime role updates or deletes a published
// version: permission denied, before any policy is consulted.
func TestP302VersionsImmutable(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA}, nil)
	seedEntityTypes(t, super, tenantA)

	for _, role := range []string{"nava_app", "nava_worker", "nava_onboard", "nava_groupsync", "nava_relay", "nava_purge"} {
		conn := asRole(t, role)
		for _, q := range []string{
			"UPDATE entity_definition_versions SET compat = 'needs_backfill' WHERE tenant_id = $1",
			"UPDATE entity_definition_versions SET schema = '{}'::json, schema_sha256 = sha256('{}') WHERE tenant_id = $1",
			"DELETE FROM entity_definition_versions WHERE tenant_id = $1",
			"DELETE FROM entity_definitions WHERE tenant_id = $1",
		} {
			_, err := execIn(conn, tenantA, q, tenantA)
			wantSQLState(t, role+": "+q, err, "42501")
		}
	}
	if got := queryString(t, super, "SELECT compat || ':' || count(*) OVER () FROM entity_definition_versions WHERE tenant_id = $1", tenantA); got != "initial:1" {
		t.Errorf("tenant A's version changed: %s, want initial:1", got)
	}
}

// Acceptance (TICK-363): number_prefix is unique per tenant, not globally,
// and set once; key likewise. nava_app may move current_version (control).
func TestP302NumberPrefixPerTenantAndFixed(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, nil)
	app := asRole(t, "nava_app")

	if _, err := execIn(app, tenantA, insertType, tenantA, "ticket", "TCK"); err != nil {
		t.Fatalf("nava_app creates tenant A's first type: %v", err)
	}
	_, err := execIn(app, tenantA, insertTypeNewID, tenantA, "incident", "TCK")
	wantSQLState(t, "a second type with the same number_prefix in tenant A", err, "23505")
	_, err = execIn(app, tenantA, insertTypeNewID, tenantA, "ticket", "INC")
	wantSQLState(t, "a second type with the same key in tenant A", err, "23505")
	if _, err := execIn(app, tenantB, insertType, tenantB, "ticket", "TCK"); err != nil {
		t.Errorf("the same key and number_prefix in tenant B: %v; want accepted", err)
	}

	for _, q := range []string{
		"UPDATE entity_definitions SET number_prefix = 'NEW' WHERE tenant_id = $1",
		"UPDATE entity_definitions SET key = 'renamed' WHERE tenant_id = $1",
		"UPDATE entity_definitions SET name = 'Renamed' WHERE tenant_id = $1",
	} {
		_, err := execIn(app, tenantA, q, tenantA)
		wantSQLState(t, "nava_app: "+q, err, "42501")
	}
	if n, err := execIn(app, tenantA, "UPDATE entity_definitions SET current_version = 2, updated_at = now() WHERE tenant_id = $1", tenantA); err != nil || n != 1 {
		t.Errorf("control: nava_app moves its own current_version: %d rows, %v; want 1, nil", n, err)
	}
	if got := queryString(t, super, "SELECT number_prefix || ':' || key FROM entity_definitions WHERE tenant_id = $1", tenantA); got != "TCK:ticket" {
		t.Errorf("tenant A's type is %s, want TCK:ticket", got)
	}

	for _, bad := range []struct{ key, prefix string }{
		{"Ticket", "TKA"}, {"t", "TKB"}, {"9ticket", "TKC"}, {"ticket_ok", "T"}, {"ticket_ok2", "tck"}, {"ticket_ok3", "TOOLONG"},
	} {
		_, err := execIn(app, tenantB, insertTypeNewID, tenantB, bad.key, bad.prefix)
		wantSQLState(t, "type key "+bad.key+", prefix "+bad.prefix, err, "23514")
	}
}

// Acceptance: the schema reads back byte-identical, key order, duplicate
// keys and whitespace included (json, not jsonb), and a row whose
// schema_sha256 does not hash those bytes is refused by the CHECK.
func TestP302SchemaBytesExact(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA}, nil)
	seedIdentity(t, super, tenantA)
	exec(t, super, insertType, tenantA, "ticket", "TCK")
	app := asRole(t, "nava_app")

	// Out-of-order keys, a duplicate key, tabs, a newline and non-ASCII:
	// jsonb would rewrite every one of them.
	raw := "{\"z\":1,  \"a\" :\t{\"b\": [1,2 ,3]},\n \"z\": 2, \"title\": \"Störung\"}"
	if _, err := execIn(app, tenantA, insertVersion, tenantA, tenantA, tenantA, 1, raw, schemaSum(raw), "initial", tenantA); err != nil {
		t.Fatalf("nava_app publishes version 1: %v", err)
	}
	var got []byte
	var sum []byte
	err := inTenantTx(app, tenantA, func(tx pgx.Tx) error {
		return tx.QueryRow(context.Background(),
			"SELECT schema::text, schema_sha256 FROM entity_definition_versions WHERE tenant_id = $1 AND id = $1", tenantA).Scan(&got, &sum)
	})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, []byte(raw)) {
		t.Errorf("schema read back as %q, want %q", got, raw)
	}
	if !bytes.Equal(sum, schemaSum(raw)) {
		t.Errorf("schema_sha256 = %x, want %x", sum, schemaSum(raw))
	}

	other := `{"type": "object"}`
	_, err = execIn(app, tenantA, insertVersion, tenantA, tenantB, tenantA, 2, raw, schemaSum(other), "compatible", tenantA)
	wantSQLState(t, "a version whose schema_sha256 hashes other bytes", err, "23514")
	_, err = execIn(app, tenantA, insertVersion, tenantA, tenantB, tenantA, 2, raw, schemaSum(raw+" "), "compatible", tenantA)
	wantSQLState(t, "a version whose schema_sha256 misses one trailing space", err, "23514")
	// Version 1 alone is 'initial'.
	_, err = execIn(app, tenantA, insertVersion, tenantA, tenantB, tenantA, 2, raw, schemaSum(raw), "initial", tenantA)
	wantSQLState(t, "version 2 marked initial", err, "23514")
}

// Acceptance (D-3-2, the Denied SQL path): a version in tenant A naming
// tenant B's principal as published_by, or tenant B's type, is refused by
// the composite foreign key, though the policy passes it.
func TestP302ForeignKeysRejectCrossTenant(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, nil)
	seedEntityTypes(t, super, tenantA, tenantB)
	app := asRole(t, "nava_app")

	// Seeded ids equal the tenant id: B's principal and type are tenantB.
	_, err := execIn(app, tenantA, insertVersion, tenantA, tenantX, tenantA, 2, testSchema, schemaSum(testSchema), "compatible", tenantB)
	wantSQLState(t, "A's version published by B's principal", err, "23503")
	_, err = execIn(app, tenantA, insertVersion, tenantA, tenantX, tenantB, 1, testSchema, schemaSum(testSchema), "initial", tenantA)
	wantSQLState(t, "A's version of B's type", err, "23503")

	// Control: the same version with A's own principal is accepted.
	if _, err := execIn(app, tenantA, insertVersion, tenantA, tenantX, tenantA, 2, testSchema, schemaSum(testSchema), "compatible", tenantA); err != nil {
		t.Fatalf("control: A's version 2 published by A's principal: %v", err)
	}
	// The version id is unique per tenant, not globally.
	_, err = execIn(app, tenantA, insertVersion, tenantA, tenantX, tenantA, 3, testSchema, schemaSum(testSchema), "compatible", tenantA)
	wantSQLState(t, "a second version with the same id in tenant A", err, "23505")
	if _, err := execIn(app, tenantB, insertVersion, tenantB, tenantX, tenantB, 2, testSchema, schemaSum(testSchema), "compatible", tenantB); err != nil {
		t.Errorf("the same version id in tenant B: %v; want accepted", err)
	}
}

// nava_onboard seeds types and versions (P3-06) into its own provisioning
// tenant, with no principal yet (published_by NULL), and into no active
// tenant: its own once activated, or another.
func TestP302OnboardSeedsOnlyProvisioning(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA}, []string{tenantX})
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
		t.Fatalf("control: nava_onboard seeds its provisioning tenant: %v", err)
	}

	_, err = execIn(onboard, tenantA, insertType, tenantA, "ticket", "TCK")
	wantSQLState(t, "nava_onboard seeds a type into its own active tenant", err, "42501")
	_, err = execIn(onboard, tenantX, insertTypeNewID, tenantA, "ticket", "TCK")
	wantSQLState(t, "nava_onboard in tenant X seeds a type into tenant A", err, "42501")
	exec(t, super, insertType, tenantA, "ticket", "TCK")
	_, err = execIn(onboard, tenantA, insertVersion, tenantA, tenantA, tenantA, 1, testSchema, schemaSum(testSchema), "initial", nil)
	wantSQLState(t, "nava_onboard seeds a version into its own active tenant", err, "42501")

	exec(t, super, "UPDATE tenants SET state = 'active' WHERE id = $1", tenantX)
	_, err = execIn(onboard, tenantX, insertTypeNewID, tenantX, "incident", "INC")
	wantSQLState(t, "nava_onboard seeds a type after activation", err, "42501")
	if got := queryString(t, super, "SELECT count(*)::text FROM entity_definitions WHERE tenant_id = $1", tenantX); got != "1" {
		t.Errorf("tenant X holds %s types, want the 1 seeded while provisioning", got)
	}
}

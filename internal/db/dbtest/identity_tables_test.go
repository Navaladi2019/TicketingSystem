//go:build integration

package dbtest

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"
)

// T16 (TICK-69): the identity tables under FORCE RLS. nava_groupsync writes
// them, nava_app reads them, and every key and foreign key holds tenant_id.

// identityTables are the tables T16 creates.
func identityTables() []string {
	return []string{"group_members", "groups", "org_units", "principals"}
}

// seedIdentity writes, as the superuser, one principal, group, membership
// and org unit for each tenant, each with id = the tenant id, and Keycloak
// ids shared across tenants. seedTenants must have created the tenants.
func seedIdentity(t *testing.T, super *pgx.Conn, tenants ...string) {
	t.Helper()
	for _, id := range tenants {
		exec(t, super, `INSERT INTO principals (tenant_id, id, keycloak_user_id, email, enabled, federated_links, synced_at)
			VALUES ($1, $1, 'kc-user', 'u@example.test', true, '[]', now())`, id)
		exec(t, super, "INSERT INTO groups (tenant_id, id, keycloak_group_id, path) VALUES ($1::uuid, $1, 'kc-group', '/t/' || $1::text || '/agents')", id)
		exec(t, super, "INSERT INTO group_members (tenant_id, group_id, principal_id) VALUES ($1, $1, $1)", id)
		exec(t, super, "INSERT INTO org_units (tenant_id, id, keycloak_group_id, name) VALUES ($1, $1, 'kc-ou', 'Root')", id)
	}
}

// Acceptance: with tenant A set, neither nava_app nor nava_groupsync sees a
// row of tenant B in any identity table; with no tenant, or an empty one,
// they see none at all. B's rows cannot be written from A either.
func TestT16TenantRowsInvisible(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, nil)
	seedIdentity(t, super, tenantA, tenantB)

	for _, role := range []string{"nava_app", "nava_groupsync"} {
		conn := asRole(t, role)
		for _, table := range identityTables() {
			for _, mode := range []string{"A", "empty", "unset"} {
				var own, other int
				err := inTenantTx(conn, "", func(tx pgx.Tx) error {
					if mode != "unset" {
						v := map[string]string{"A": tenantA, "empty": ""}[mode]
						if _, err := tx.Exec(context.Background(), "SELECT set_config('app.tenant_id', $1, true)", v); err != nil {
							return err
						}
					}
					return tx.QueryRow(context.Background(), fmt.Sprintf(
						"SELECT count(*) FILTER (WHERE tenant_id = $1), count(*) FILTER (WHERE tenant_id <> $1) FROM %s", table),
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

	sync := asRole(t, "nava_groupsync")
	for _, w := range []struct{ name, sql string }{
		{"principals", "INSERT INTO principals (tenant_id, id, keycloak_user_id, enabled, synced_at) VALUES ($1, gen_random_uuid(), 'kc-new', true, now())"},
		{"groups", "INSERT INTO groups (tenant_id, id, keycloak_group_id, path) VALUES ($1::uuid, gen_random_uuid(), 'kc-new', '/t/' || $1::text || '/new')"},
		{"group_members", "INSERT INTO group_members (tenant_id, group_id, principal_id) VALUES ($1, $1, $1)"},
		{"org_units", "INSERT INTO org_units (tenant_id, id, keycloak_group_id, name) VALUES ($1, gen_random_uuid(), 'kc-new', 'New')"},
	} {
		_, err := execIn(sync, tenantA, w.sql, tenantB)
		wantSQLState(t, "A writes a B row into "+w.name, err, "42501")
	}
	for _, w := range []string{
		"UPDATE principals SET enabled = false, federated_links = NULL WHERE tenant_id = $1",
		"UPDATE groups SET path = path WHERE tenant_id = $1",
		"UPDATE org_units SET name = 'x' WHERE tenant_id = $1",
		"DELETE FROM principals WHERE tenant_id = $1",
		"DELETE FROM groups WHERE tenant_id = $1",
		"DELETE FROM group_members WHERE tenant_id = $1",
		"DELETE FROM org_units WHERE tenant_id = $1",
	} {
		if n, err := execIn(sync, tenantA, w, tenantB); err != nil || n != 0 {
			t.Errorf("A: %s on B: %d rows, %v; want 0, nil", w, n, err)
		}
	}
	if left := queryString(t, super, `SELECT (SELECT count(*) FROM principals WHERE tenant_id = $1 AND enabled)
		+ (SELECT count(*) FROM groups WHERE tenant_id = $1) + (SELECT count(*) FROM group_members WHERE tenant_id = $1)
		+ (SELECT count(*) FROM org_units WHERE tenant_id = $1 AND name = 'Root')`, tenantB); left != "4" {
		t.Errorf("tenant B's identity rows after A's writes: %s of 4 intact", left)
	}
}

// Acceptance: foreign keys include tenant_id. A membership of tenant A cannot
// point at B's group or B's principal, and an org unit of A cannot hang under
// B's unit, even when tenant A writes it under its own tenant.
func TestT16ForeignKeysRejectCrossTenant(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, nil)
	seedIdentity(t, super, tenantA, tenantB)
	sync := asRole(t, "nava_groupsync")
	if _, err := execIn(sync, tenantA, "DELETE FROM group_members WHERE tenant_id = $1", tenantA); err != nil {
		t.Fatal(err)
	}

	// Seeded ids equal the tenant id, so B's group, principal and unit are tenantB.
	_, err := execIn(sync, tenantA, "INSERT INTO group_members (tenant_id, group_id, principal_id) VALUES ($1, $2, $1)", tenantA, tenantB)
	wantSQLState(t, "A's membership in B's group", err, "23503")
	_, err = execIn(sync, tenantA, "INSERT INTO group_members (tenant_id, group_id, principal_id) VALUES ($1, $1, $2)", tenantA, tenantB)
	wantSQLState(t, "A's membership for B's principal", err, "23503")
	_, err = execIn(sync, tenantA, "INSERT INTO org_units (tenant_id, id, parent_id, keycloak_group_id, name) VALUES ($1, gen_random_uuid(), $2, 'kc-child', 'Child')", tenantA, tenantB)
	wantSQLState(t, "A's org unit under B's unit", err, "23503")
	_, err = execIn(sync, tenantA, "UPDATE org_units SET parent_id = $2 WHERE tenant_id = $1", tenantA, tenantB)
	wantSQLState(t, "A's root moved under B's unit", err, "23503")

	// The same rows inside tenant A are accepted.
	if _, err := execIn(sync, tenantA, "INSERT INTO group_members (tenant_id, group_id, principal_id) VALUES ($1, $1, $1)", tenantA); err != nil {
		t.Fatalf("A's membership in its own group: %v", err)
	}
	if _, err := execIn(sync, tenantA, "INSERT INTO org_units (tenant_id, id, parent_id, keycloak_group_id, name) VALUES ($1, gen_random_uuid(), $1, 'kc-child', 'Child')", tenantA); err != nil {
		t.Fatalf("A's org unit under its own root: %v", err)
	}
	_, err = execIn(sync, tenantA, "UPDATE org_units SET parent_id = id WHERE tenant_id = $1 AND id = $1", tenantA)
	wantSQLState(t, "an org unit as its own parent", err, "23514")
}

// Removing a principal or a group removes its memberships, in its own
// tenant only; a parent unit with children cannot be removed alone, but its
// subtree can in one statement.
func TestT16DeletesCascadeWithinTenant(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, nil)
	seedIdentity(t, super, tenantA, tenantB)
	sync := asRole(t, "nava_groupsync")
	members := func(tenant string) string {
		return queryString(t, super, "SELECT count(*)::text FROM group_members WHERE tenant_id = $1", tenant)
	}

	if _, err := execIn(sync, tenantA, "DELETE FROM principals WHERE tenant_id = $1", tenantA); err != nil {
		t.Fatalf("delete A's principal: %v", err)
	}
	if members(tenantA) != "0" || members(tenantB) != "1" {
		t.Errorf("memberships after deleting A's principal: A %s, B %s; want 0, 1", members(tenantA), members(tenantB))
	}
	if _, err := execIn(sync, tenantB, "DELETE FROM groups WHERE tenant_id = $1", tenantB); err != nil {
		t.Fatalf("delete B's group: %v", err)
	}
	if members(tenantB) != "0" {
		t.Errorf("B's memberships after deleting its group: %s, want 0", members(tenantB))
	}

	if _, err := execIn(sync, tenantA, "INSERT INTO org_units (tenant_id, id, parent_id, keycloak_group_id, name) VALUES ($1, gen_random_uuid(), $1, 'kc-child', 'Child')", tenantA); err != nil {
		t.Fatal(err)
	}
	_, err := execIn(sync, tenantA, "DELETE FROM org_units WHERE tenant_id = $1 AND id = $1", tenantA)
	wantSQLState(t, "delete a parent unit with a child", err, "23503")
	if n, err := execIn(sync, tenantA, "DELETE FROM org_units WHERE tenant_id = $1", tenantA); err != nil || n != 2 {
		t.Errorf("delete A's whole tree: %d rows, %v; want 2, nil", n, err)
	}
}

// Acceptance: federated_links is nullable with no default. NULL (unknown)
// and '[]' (synced, no links) are distinct; a writer that omits the column
// gets NULL, never '[]'. The CHECK accepts only an array of objects with a
// string identityProvider.
func TestT16FederatedLinksFailClosed(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA}, nil)

	var notNull, hasDefault bool
	if err := super.QueryRow(context.Background(), `
		SELECT attnotnull, atthasdef FROM pg_attribute
		WHERE attrelid = 'principals'::regclass AND attname = 'federated_links'`).Scan(&notNull, &hasDefault); err != nil {
		t.Fatal(err)
	}
	if notNull || hasDefault {
		t.Fatalf("principals.federated_links: NOT NULL %v, default %v; want nullable with no default", notNull, hasDefault)
	}

	sync := asRole(t, "nava_groupsync")
	insert := func(kcID string, links any) error {
		_, err := execIn(sync, tenantA, `INSERT INTO principals (tenant_id, id, keycloak_user_id, enabled, federated_links, synced_at)
			VALUES ($1, gen_random_uuid(), $2, true, $3, now())`, tenantA, kcID, links)
		return err
	}
	if _, err := execIn(sync, tenantA, "INSERT INTO principals (tenant_id, id, keycloak_user_id, enabled, synced_at) VALUES ($1, gen_random_uuid(), 'omitted', true, now())", tenantA); err != nil {
		t.Fatal(err)
	}
	if !queryBool(t, super, "SELECT federated_links IS NULL FROM principals WHERE keycloak_user_id = 'omitted'") {
		t.Error("a principal written without federated_links does not read back NULL")
	}
	for i, ok := range []any{nil, "[]", `[{"identityProvider": "idp-a", "userId": "u1", "userName": "u1"}]`} {
		if err := insert(fmt.Sprintf("ok-%d", i), ok); err != nil {
			t.Errorf("federated_links %v: %v", ok, err)
		}
	}
	for i, bad := range []string{
		`{}`, `"idp-a"`, `null`, `[1]`, `["idp-a"]`,
		`[{"userId": "u1"}]`,
		`[{"identityProvider": 7}]`,
		`[{"identityProvider": null}]`,
		`[{"identityProvider": "idp-a"}, {"userId": "u2"}]`,
	} {
		wantSQLState(t, "federated_links "+bad, insert(fmt.Sprintf("bad-%d", i), bad), "23514")
	}
}

// Keycloak ids are unique per tenant, not globally; group paths stay under
// the tenant's own subtree.
func TestT16KeysArePerTenant(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, nil)
	seedIdentity(t, super, tenantB)
	sync := asRole(t, "nava_groupsync")

	add := "INSERT INTO principals (tenant_id, id, keycloak_user_id, enabled, federated_links, synced_at) VALUES ($1, gen_random_uuid(), 'kc-user', true, '[]', now())"
	if _, err := execIn(sync, tenantA, add, tenantA); err != nil {
		t.Fatalf("A adds the Keycloak user id B already uses: %v", err)
	}
	_, err := execIn(sync, tenantA, add, tenantA)
	wantSQLState(t, "A adds kc-user twice", err, "23505")
	_, err = execIn(sync, tenantA, "INSERT INTO groups (tenant_id, id, keycloak_group_id, path) VALUES ($1::uuid, gen_random_uuid(), 'kc-group', '/t/' || $1::text || '/a')", tenantA)
	if err != nil {
		t.Fatalf("A adds the Keycloak group id B already uses: %v", err)
	}
	_, err = execIn(sync, tenantA, "INSERT INTO groups (tenant_id, id, keycloak_group_id, path) VALUES ($1::uuid, gen_random_uuid(), 'kc-group', '/t/' || $1::text || '/b')", tenantA)
	wantSQLState(t, "A adds kc-group twice", err, "23505")

	for _, path := range []string{"/t/" + tenantB + "/agents", "/t/" + tenantA, "/agents", "/t/" + tenantA + "x/agents"} {
		_, err := execIn(sync, tenantA, "INSERT INTO groups (tenant_id, id, keycloak_group_id, path) VALUES ($1, gen_random_uuid(), 'kc-g-path', $2)", tenantA, path)
		wantSQLState(t, "A's group at "+path, err, "23514")
	}
}

// nava_app only reads; nava_groupsync cannot re-key a row in place.
func TestT16WriteGrants(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA}, nil)
	seedIdentity(t, super, tenantA)
	app, sync := asRole(t, "nava_app"), asRole(t, "nava_groupsync")
	for _, w := range []string{
		"INSERT INTO principals (tenant_id, id, keycloak_user_id, enabled, synced_at) VALUES ($1, gen_random_uuid(), 'x', true, now())",
		"UPDATE principals SET federated_links = '[]' WHERE tenant_id = $1",
		"DELETE FROM group_members WHERE tenant_id = $1",
		"UPDATE groups SET path = path WHERE tenant_id = $1",
		"DELETE FROM org_units WHERE tenant_id = $1",
	} {
		_, err := execIn(app, tenantA, w, tenantA)
		wantSQLState(t, "nava_app: "+w, err, "42501")
	}
	for _, w := range []string{
		"UPDATE principals SET keycloak_user_id = 'other' WHERE tenant_id = $1",
		"UPDATE principals SET id = gen_random_uuid() WHERE tenant_id = $1",
		"UPDATE principals SET tenant_id = tenant_id WHERE tenant_id = $1",
		"UPDATE groups SET keycloak_group_id = 'other' WHERE tenant_id = $1",
		"UPDATE group_members SET principal_id = principal_id WHERE tenant_id = $1",
		"UPDATE org_units SET keycloak_group_id = 'other' WHERE tenant_id = $1",
	} {
		_, err := execIn(sync, tenantA, w, tenantA)
		wantSQLState(t, "nava_groupsync: "+w, err, "42501")
	}
	if n, err := execIn(sync, tenantA, "UPDATE principals SET email = NULL, enabled = false, federated_links = NULL, synced_at = now() WHERE tenant_id = $1", tenantA); err != nil || n != 1 {
		t.Errorf("nava_groupsync updates A's principal: %d rows, %v; want 1, nil", n, err)
	}
}

//go:build integration

package dbtest

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
)

// TICK-1025 (20261120160000_worker_principal_reads.sql): nava_worker reads
// principals (tenant_id, id, deleted_at) and group_members (tenant_id,
// group_id, principal_id) under principals_select and group_members_select,
// so authz.PrincipalOf runs in a workflow run's tenant transaction.

// workerPrincipalOf is authz.PrincipalOf's targetSQL (internal/authz,
// explain.go), copied because dbtest does not import authz.
const workerPrincipalOf = `SELECT p.id::text,
       ARRAY(SELECT gm.group_id::text FROM group_members gm
             JOIN groups g ON g.tenant_id = gm.tenant_id AND g.id = gm.group_id
             WHERE gm.tenant_id = p.tenant_id AND gm.principal_id = p.id AND g.deleted_at IS NULL
             ORDER BY gm.group_id)
FROM principals p
WHERE p.tenant_id = $1 AND p.id = $2 AND p.deleted_at IS NULL`

// The worker resolves its own tenant's principal and groups; a tombstoned
// principal and another tenant's principal are not found. (A tombstoned
// group keeps no members: group_members_group_live_fkey.)
func TestTICK1025WorkerResolvesPrincipal(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, nil)
	seedIdentity(t, super, tenantA, tenantB)
	worker := asRole(t, "nava_worker")
	resolve := func(ctxTenant, tenant, principal string) (id string, groups []string, found bool) {
		t.Helper()
		err := inTenantTx(worker, ctxTenant, func(tx pgx.Tx) error {
			return tx.QueryRow(context.Background(), workerPrincipalOf, tenant, principal).Scan(&id, &groups)
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil, false
		}
		if err != nil {
			t.Fatalf("nava_worker PrincipalOf(%s) from tenant '%s': %v", principal, ctxTenant, err)
		}
		return id, groups, true
	}
	id, groups, found := resolve(tenantA, tenantA, tenantA)
	if !found || id != tenantA || len(groups) != 1 || groups[0] != tenantA {
		t.Fatalf("own principal: id %q groups %v found %v; want %s [%s] true", id, groups, found, tenantA, tenantA)
	}
	if _, _, found := resolve(tenantA, tenantB, tenantB); found {
		t.Error("from A: tenant B's principal resolved by its full key")
	}
	if _, _, found := resolve("", tenantA, tenantA); found {
		t.Error("no tenant set: principal resolved")
	}
	exec(t, super, "UPDATE principals SET enabled = false, deleted_at = now() WHERE tenant_id = $1 AND id = $1", tenantA)
	if _, _, found := resolve(tenantA, tenantA, tenantA); found {
		t.Error("tombstoned principal resolved")
	}
}

// The grants widen no row set and no column set: the worker sees only its
// transaction's tenant's principals and memberships, none with no tenant or
// an empty one, reads no other principals column (keycloak_user_id
// included) and writes neither table.
func TestTICK1025WorkerPrincipalReadsStayInTenant(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, nil)
	seedIdentity(t, super, tenantA, tenantB)
	worker := asRole(t, "nava_worker")
	count := func(ctxTenant, table, where string, args ...any) int {
		t.Helper()
		var n int
		if err := inTenantTx(worker, ctxTenant, func(tx pgx.Tx) error {
			return tx.QueryRow(context.Background(), "SELECT count(*) FROM "+table+" WHERE "+where, args...).Scan(&n)
		}); err != nil {
			t.Fatalf("count %s %q from tenant '%s': %v", table, where, ctxTenant, err)
		}
		return n
	}
	for _, table := range []string{"principals", "group_members"} {
		if n := count(tenantA, table, "true"); n != 1 {
			t.Errorf("%s from A: %d rows visible, want A's 1", table, n)
		}
		if n := count(tenantA, table, "tenant_id = $1", tenantB); n != 0 {
			t.Errorf("%s from A: %d of tenant B's rows visible", table, n)
		}
		if n := count("", table, "true"); n != 0 {
			t.Errorf("%s with no tenant set: %d rows visible, want 0", table, n)
		}
		var n int
		if err := inTenantTx(worker, "", func(tx pgx.Tx) error {
			if _, err := tx.Exec(context.Background(), "SELECT set_config('app.tenant_id', '', true)"); err != nil {
				return err
			}
			return tx.QueryRow(context.Background(), "SELECT count(*) FROM "+table).Scan(&n)
		}); err != nil {
			t.Fatalf("%s with an empty tenant: %v", table, err)
		}
		if n != 0 {
			t.Errorf("%s with an empty tenant: %d rows visible, want 0", table, n)
		}
	}
	for _, col := range []string{"keycloak_user_id", "email", "enabled", "federated_links", "synced_at", "*"} {
		_, err := execIn(worker, tenantA, "SELECT "+col+" FROM principals")
		wantSQLState(t, "nava_worker SELECT principals."+col, err, "42501")
	}
	// A filter on an ungranted column is refused too, so it is no oracle.
	_, err := execIn(worker, tenantA, "SELECT id FROM principals WHERE keycloak_user_id = 'kc-user'")
	wantSQLState(t, "nava_worker filter on keycloak_user_id", err, "42501")
	for name, stmt := range map[string]string{
		"INSERT principals":    "INSERT INTO principals (tenant_id, id, keycloak_user_id, enabled, synced_at) VALUES ($1, gen_random_uuid(), 'kc-w', true, now())",
		"UPDATE principals":    "UPDATE principals SET deleted_at = now() WHERE tenant_id = $1",
		"DELETE principals":    "DELETE FROM principals WHERE tenant_id = $1",
		"INSERT group_members": "INSERT INTO group_members (tenant_id, group_id, principal_id) VALUES ($1, $1, $1)",
		"DELETE group_members": "DELETE FROM group_members WHERE tenant_id = $1",
	} {
		_, err := execIn(worker, tenantA, stmt, tenantA)
		wantSQLState(t, "nava_worker "+name, err, "42501")
	}
}

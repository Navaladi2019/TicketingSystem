//go:build integration

package dbtest

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
)

// TICK-782 (20261113090000_groups_select_worker.sql): nava_worker reads
// groups (tenant_id, id, deleted_at) under groups_select, so the record
// store's update-path read (recordSelect's LEFT JOIN groups for the
// assignment group's tombstone) runs as the worker.

// workerRecordSelect is the groups half of records/store's recordSelect,
// which dbtest may not import (internal/records/arch_test.go).
const workerRecordSelect = `SELECT r.tenant_id, coalesce(g.deleted_at IS NOT NULL, false)
	FROM records r
	LEFT JOIN groups g ON g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id
	WHERE r.tenant_id = $1 AND r.id = $2`

// The worker reads its own tenant's assignment group and its tombstone.
func TestTICK782WorkerReadsAssignmentGroupTombstone(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, nil)
	seedPipeline(t, super, tenantA, tenantB)
	worker := asRole(t, "nava_worker")
	removed := func(tenant string) (string, bool) {
		t.Helper()
		var got string
		var gone bool
		if err := inTenantTx(worker, tenant, func(tx pgx.Tx) error {
			return tx.QueryRow(context.Background(), workerRecordSelect, tenant, pipelineRecord(tenant)).Scan(&got, &gone)
		}); err != nil {
			t.Fatalf("nava_worker recordSelect in %s: %v", tenant, err)
		}
		return got, gone
	}
	if got, gone := removed(tenantA); got != tenantA || gone {
		t.Fatalf("live group: tenant %s, removed %v; want %s, false", got, gone, tenantA)
	}
	exec(t, super, "DELETE FROM group_members WHERE tenant_id = $1 AND group_id = $1", tenantA)
	exec(t, super, tombstoneGroup, tenantA, tenantA)
	if _, gone := removed(tenantA); !gone {
		t.Error("tombstoned group: removed = false, want true")
	}
	if _, gone := removed(tenantB); gone {
		t.Error("tenant B's live group reads as removed after tenant A's tombstone")
	}
}

// The grant widens no row set and no column set: the worker sees only its
// transaction's tenant's groups, none with no tenant set, reads no other
// column and writes nothing.
func TestTICK782WorkerGroupsReadStaysInTenant(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, nil)
	seedIdentity(t, super, tenantA, tenantB)
	exec(t, super, "DELETE FROM group_members WHERE tenant_id = $1 AND group_id = $1", tenantB)
	exec(t, super, tombstoneGroup, tenantB, tenantB)
	worker := asRole(t, "nava_worker")
	count := func(ctxTenant, where string, args ...any) int {
		t.Helper()
		var n int
		if err := inTenantTx(worker, ctxTenant, func(tx pgx.Tx) error {
			return tx.QueryRow(context.Background(), "SELECT count(*) FROM groups WHERE "+where, args...).Scan(&n)
		}); err != nil {
			t.Fatalf("count %q from tenant '%s': %v", where, ctxTenant, err)
		}
		return n
	}
	if n := count(tenantA, "true"); n != 1 {
		t.Errorf("from A: %d groups visible, want A's 1", n)
	}
	if n := count(tenantA, "tenant_id = $1 AND id = $1", tenantB); n != 0 {
		t.Errorf("from A: tenant B's group by its full key visible (%d rows)", n)
	}
	// Tenant B's tombstone is not an oracle from A.
	if n := count(tenantA, "deleted_at IS NOT NULL"); n != 0 {
		t.Errorf("from A: %d tombstoned groups visible, want 0", n)
	}
	if n := count("", "true"); n != 0 {
		t.Errorf("no tenant set: %d groups visible, want 0", n)
	}
	for _, col := range []string{"path", "keycloak_group_id", "live", "*"} {
		_, err := execIn(worker, tenantA, "SELECT "+col+" FROM groups")
		wantSQLState(t, "nava_worker SELECT "+col, err, "42501")
	}
	for name, stmt := range map[string]string{
		"INSERT": "INSERT INTO groups (tenant_id, id, keycloak_group_id, path) VALUES ($1::uuid, gen_random_uuid(), 'kc-w', '/t/' || $1::text || '/w')",
		"UPDATE": "UPDATE groups SET deleted_at = now() WHERE tenant_id = $1",
		"DELETE": "DELETE FROM groups WHERE tenant_id = $1",
	} {
		_, err := execIn(worker, tenantA, stmt, tenantA)
		wantSQLState(t, "nava_worker "+name, err, "42501")
	}
}

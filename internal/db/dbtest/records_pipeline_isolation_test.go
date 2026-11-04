//go:build integration

package dbtest

import (
	"context"
	"testing"
)

// Adversarial (Security review of P3-09, TICK-353; rewritten for ADR-004-E5,
// TICK-538). Tenants A and B each hold a group with the same UUID, and each
// tenant's pipeline record names it. Group sync running for tenant A must
// reach only tenant A's group: naming tenant B's group reaches nothing, and
// neither record changes. Sync tombstones, never deletes (it holds no DELETE
// on groups), and records_assignment_group_fkey is ON DELETE RESTRICT, so no
// FK action can rewrite either tenant's record; even the owner's delete of
// the shared id in tenant A stops at the composite key's RESTRICT.
func TestP309GroupTombstoneStaysInTenant(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, nil)
	seedPipeline(t, super, tenantA, tenantB)
	shared := tenantX // the same group id in both tenants
	for _, tid := range []string{tenantA, tenantB} {
		exec(t, super, `INSERT INTO groups (tenant_id, id, keycloak_group_id, path)
			VALUES ($1::uuid, $2, 'kc-shared', '/t/' || $1::text || '/shared')`, tid, shared)
		exec(t, super, withRecordEntry("UPDATE records SET assignment_group_id = $2 WHERE tenant_id = $1 AND id = $3"),
			tid, shared, pipelineRecord(tid))
	}
	sync := asRole(t, "nava_groupsync")

	// Attack first: group sync running for tenant A names tenant B's group.
	// RLS hides B's row, so the tombstone reaches nothing.
	if n, err := execIn(sync, tenantA, "UPDATE groups SET deleted_at = now() WHERE tenant_id = $1 AND id = $2", tenantB, shared); err != nil || n != 0 {
		t.Fatalf("tenant A's sync tombstones tenant B's group: %d rows, %v; want 0, nil", n, err)
	}
	if n, err := execIn(sync, tenantA, "UPDATE groups SET deleted_at = now() WHERE tenant_id = $1 AND id = $2", tenantA, shared); err != nil || n != 1 {
		t.Fatalf("tenant A's sync tombstones its own group: %d rows, %v; want 1, nil", n, err)
	}
	// The owner bypasses RLS; the composite key still limits RESTRICT to
	// tenant A's record, which names the group, so the delete fails.
	_, err := super.Exec(context.Background(), "DELETE FROM groups WHERE tenant_id = $1 AND id = $2", tenantA, shared)
	wantSQLState(t, "owner deletes tenant A's group that tenant A's record names", err, "23503")

	group := func(tid string) string {
		return queryString(t, super, "SELECT coalesce(assignment_group_id::text, 'null') FROM records WHERE tenant_id = $1 AND id = $2",
			tid, pipelineRecord(tid))
	}
	for _, tid := range []string{tenantA, tenantB} {
		if got := group(tid); got != shared {
			t.Errorf("record group of %s after tenant A's group was tombstoned = %s, want %s (unchanged)", tid, got, shared)
		}
	}
	if got := queryString(t, super, "SELECT string_agg(tenant_id || ':' || (deleted_at IS NOT NULL)::text, ',' ORDER BY tenant_id = $1 DESC) FROM groups WHERE id = $2",
		tenantA, shared); got != tenantA+":true,"+tenantB+":false" {
		t.Errorf("shared groups (tenant:tombstoned) = %s, want %s:true,%s:false", got, tenantA, tenantB)
	}
}

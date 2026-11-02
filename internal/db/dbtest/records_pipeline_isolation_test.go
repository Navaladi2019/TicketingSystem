//go:build integration

package dbtest

import "testing"

// Adversarial (Security review of P3-09, TICK-353): the ON DELETE SET NULL
// action on records_assignment_group_fkey runs as the table owner, so RLS
// does not limit it; only the composite key does. Tenants A and B each hold
// a group with the same UUID, and each tenant's pipeline record names it.
// Group sync deleting tenant A's group must clear only tenant A's record.
// Tenant B's record must keep its group. A single-column FK on
// assignment_group_id would clear B's record as well.
func TestP309GroupDeleteSetNullStaysInTenant(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, nil)
	seedPipeline(t, super, tenantA, tenantB)
	shared := tenantX // the same group id in both tenants
	for _, tid := range []string{tenantA, tenantB} {
		exec(t, super, `INSERT INTO groups (tenant_id, id, keycloak_group_id, path)
			VALUES ($1::uuid, $2, 'kc-shared', '/t/' || $1::text || '/shared')`, tid, shared)
		exec(t, super, "UPDATE records SET assignment_group_id = $2 WHERE tenant_id = $1 AND id = $3",
			tid, shared, pipelineRecord(tid))
	}
	sync := asRole(t, "nava_groupsync")

	// Attack first: group sync running for tenant A names tenant B's group.
	// RLS hides B's row, so the delete reaches nothing.
	if n, err := execIn(sync, tenantA, "DELETE FROM groups WHERE tenant_id = $1 AND id = $2", tenantB, shared); err != nil || n != 0 {
		t.Fatalf("tenant A's sync deletes tenant B's group: %d rows, %v; want 0, nil", n, err)
	}
	if n, err := execIn(sync, tenantA, "DELETE FROM groups WHERE tenant_id = $1 AND id = $2", tenantA, shared); err != nil || n != 1 {
		t.Fatalf("tenant A's sync deletes its own group: %d rows, %v; want 1, nil", n, err)
	}

	group := func(tid string) string {
		return queryString(t, super, "SELECT coalesce(assignment_group_id::text, 'null') FROM records WHERE tenant_id = $1 AND id = $2",
			tid, pipelineRecord(tid))
	}
	if got := group(tenantA); got != "null" {
		t.Errorf("tenant A's record group after its group was deleted = %s, want null", got)
	}
	if got := group(tenantB); got != shared {
		t.Errorf("tenant B's record group after tenant A's group was deleted = %s, want %s (unchanged)", got, shared)
	}
	if got := queryString(t, super, "SELECT count(*)::text FROM groups WHERE tenant_id = $1 AND id = $2", tenantB, shared); got != "1" {
		t.Errorf("tenant B's group rows = %s, want 1", got)
	}
}

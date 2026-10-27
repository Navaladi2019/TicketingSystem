//go:build integration

package dbtest

import "testing"

// TICK-220 adversarial: deleted_at is a new column nava_groupsync can write.
// Holding tenant A's context, sync can neither tombstone tenant B's principal
// (locking B's user out with 401) nor revive B's tombstone (letting a departed
// B user back in). With no tenant set it reaches no principal at all.
func TestT220TombstoneCrossTenant(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, nil)
	seedIdentity(t, super, tenantA, tenantB)
	sync := asRole(t, "nava_groupsync")
	rowB := func() string {
		return queryString(t, super, "SELECT concat_ws(',', enabled, deleted_at IS NULL, synced_at) FROM principals WHERE tenant_id = $1", tenantB)
	}

	before := rowB()
	for _, tenant := range []string{tenantA, ""} {
		n, err := execIn(sync, tenant, "UPDATE principals SET enabled = false, deleted_at = now() WHERE tenant_id = $1 AND keycloak_user_id = 'kc-user'", tenantB)
		if err != nil || n != 0 {
			t.Errorf("tenant %q tombstones B's principal: %d rows, %v; want 0, nil", tenant, n, err)
		}
		n, err = execIn(sync, tenant, "UPDATE principals SET enabled = false, deleted_at = now() WHERE keycloak_user_id = 'kc-user'")
		if err != nil || n > 1 {
			t.Errorf("tenant %q tombstones by Keycloak id alone: %d rows, %v; want at most its own", tenant, n, err)
		}
	}
	if got := rowB(); got != before {
		t.Errorf("B's principal changed from %s to %s", before, got)
	}

	exec(t, super, "UPDATE principals SET enabled = false, deleted_at = now() WHERE tenant_id = $1", tenantB)
	before = rowB()
	for _, tenant := range []string{tenantA, ""} {
		n, err := execIn(sync, tenant, "UPDATE principals SET enabled = true, deleted_at = NULL WHERE tenant_id = $1", tenantB)
		if err != nil || n != 0 {
			t.Errorf("tenant %q revives B's tombstone: %d rows, %v; want 0, nil", tenant, n, err)
		}
		_, err = execIn(sync, tenant, "UPDATE principals SET tenant_id = $1, enabled = true, deleted_at = NULL WHERE tenant_id = $2", tenantB, tenantA)
		if tenant == tenantA {
			wantSQLState(t, "A moves its principal into B", err, "42501")
		}
	}
	if got := rowB(); got != before {
		t.Errorf("B's tombstone changed from %s to %s", before, got)
	}
}

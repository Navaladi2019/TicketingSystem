//go:build integration

package dbtest

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

// Adversarial (ADR-004-E5 D2; TICK-561, follow-up to the Security review of
// TICK-538). records_assignment_group_fkey is ON DELETE RESTRICT on the
// composite key (tenant_id, assignment_group_id) -> groups (tenant_id, id).
// Tenants A and B each hold a group with the same UUID, and only tenant B's
// record names it. If RESTRICT matched on the group id alone, tenant B's
// data would stop the owner from deleting tenant A's group: a cross-tenant
// effect. The owner's delete of tenant A's group must succeed and leave
// tenant B's group and record unchanged; the delete of tenant B's group,
// which B's record names, must fail with 23503 (control). The owner is the
// superuser, so RLS is bypassed and only the FK decides.
func TestE5RestrictIsCompositeAcrossTenants(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, nil)
	seedPipeline(t, super, tenantA, tenantB)
	shared := tenantX // the same group id in both tenants
	for _, tid := range []string{tenantA, tenantB} {
		exec(t, super, `INSERT INTO groups (tenant_id, id, keycloak_group_id, path)
			VALUES ($1::uuid, $2, 'kc-shared', '/t/' || $1::text || '/shared')`, tid, shared)
	}
	// Only tenant B's record names the shared id; tenant A's record keeps
	// tenant A's seeded group. records carries the commit-check trigger
	// (TICK-530), so the update writes its audit entry in the same statement.
	exec(t, super, withRecordEntry("UPDATE records SET assignment_group_id = $2 WHERE tenant_id = $1 AND id = $3"),
		tenantB, shared, pipelineRecord(tenantB))
	if got := queryString(t, super, "SELECT coalesce(assignment_group_id::text, 'null') FROM records WHERE tenant_id = $1 AND id = $2",
		tenantB, pipelineRecord(tenantB)); got != shared {
		t.Fatalf("setup: tenant B's record group = %s, want %s", got, shared)
	}
	if got := queryString(t, super, "SELECT count(*)::text FROM records WHERE tenant_id = $1 AND assignment_group_id = $2",
		tenantA, shared); got != "0" {
		t.Fatalf("tenant A records naming the shared group = %s, want 0", got)
	}

	ctx := context.Background()
	n, err := super.Exec(ctx, "DELETE FROM groups WHERE tenant_id = $1 AND id = $2", tenantA, shared)
	if err != nil {
		t.Fatalf("owner deletes tenant A's group that only tenant B's record shares an id with: %v; want success (RESTRICT must match on (tenant_id, id))", err)
	}
	if n.RowsAffected() != 1 {
		t.Fatalf("owner delete of tenant A's group removed %d rows, want 1", n.RowsAffected())
	}

	if got := queryString(t, super, "SELECT string_agg(tenant_id::text, ',' ORDER BY tenant_id) FROM groups WHERE id = $1",
		shared); got != tenantB {
		t.Errorf("tenants holding the shared group after the delete = %s, want %s", got, tenantB)
	}
	if got := queryString(t, super, "SELECT (deleted_at IS NOT NULL)::text FROM groups WHERE tenant_id = $1 AND id = $2",
		tenantB, shared); got != "false" {
		t.Errorf("tenant B's group tombstoned = %s, want false", got)
	}
	if got := queryString(t, super, "SELECT coalesce(assignment_group_id::text, 'null') FROM records WHERE tenant_id = $1 AND id = $2",
		tenantB, pipelineRecord(tenantB)); got != shared {
		t.Errorf("tenant B's record group after tenant A's delete = %s, want %s (unchanged)", got, shared)
	}

	// Control: tenant B's own record names tenant B's group, so RESTRICT holds.
	_, err = super.Exec(ctx, "DELETE FROM groups WHERE tenant_id = $1 AND id = $2", tenantB, shared)
	wantSQLState(t, "owner deletes tenant B's group that tenant B's record names", err, "23503")
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.ConstraintName != "records_assignment_group_fkey" {
		t.Errorf("owner delete refused by %s, want records_assignment_group_fkey", pgErr.ConstraintName)
	}
	if got := queryString(t, super, "SELECT coalesce(assignment_group_id::text, 'null') FROM records WHERE tenant_id = $1 AND id = $2",
		tenantB, pipelineRecord(tenantB)); got != shared {
		t.Errorf("tenant B's record group after the refused delete = %s, want %s", got, shared)
	}
}

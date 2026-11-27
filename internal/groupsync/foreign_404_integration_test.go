//go:build integration

// TICK-1141 against a real PostgreSQL: a 404 that is not Keycloak's, or a 200
// for another group, fails the cycle of that tenant and changes nothing; only
// Keycloak's own "group path does not exist" removes a tenant's directory.

package groupsync_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/identity/keycloak/keycloaktest"
)

// stored dumps every row of the tenant in the four tables a cycle writes,
// with xmin, so a rewrite shows.
func stored(t *testing.T, super *pgx.Conn, tid uuid.UUID) string {
	t.Helper()
	var s string
	err := super.QueryRow(context.Background(), `SELECT
	  (SELECT coalesce(string_agg(id::text||'/'||xmin::text||'/'||(deleted_at IS NOT NULL), ',' ORDER BY id), '') FROM principals WHERE tenant_id = $1)
	  || '|' || (SELECT coalesce(string_agg(id::text||'/'||xmin::text||'/'||(deleted_at IS NOT NULL), ',' ORDER BY id), '') FROM groups WHERE tenant_id = $1)
	  || '|' || (SELECT coalesce(string_agg(group_id::text||'/'||principal_id::text||'/'||xmin::text, ',' ORDER BY group_id, principal_id), '') FROM group_members WHERE tenant_id = $1)
	  || '|' || (SELECT coalesce(string_agg(group_id::text||'/'||xmin::text, ',' ORDER BY group_id), '') FROM tenant_admin_group WHERE tenant_id = $1)`, tid).Scan(&s)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// withAdminGroup makes the tenant's one group its admins group.
func withAdminGroup(t *testing.T, super *pgx.Conn, tid uuid.UUID) {
	t.Helper()
	var gid uuid.UUID
	if err := super.QueryRow(context.Background(), "SELECT id FROM groups WHERE tenant_id = $1 AND deleted_at IS NULL", tid).Scan(&gid); err != nil {
		t.Fatal(err)
	}
	if _, err := super.Exec(context.Background(), "INSERT INTO tenant_admin_group (tenant_id, group_id) VALUES ($1, $2)", tid, gid); err != nil {
		t.Fatal(err)
	}
}

func TestForeign404FailsOnlyItsTenantAndChangesNothing(t *testing.T) {
	for name, fault := range map[string]keycloaktest.Fault{
		"both user reads 404 from a proxy": keycloaktest.ProxyUsers404,
		"root group 404 from a proxy":      keycloaktest.ProxyRoot404,
		"root group 200 with another path": keycloaktest.ProxyRootEmpty,
	} {
		t.Run(name, func(t *testing.T) {
			k, s, super, a, b := vanishStart(t)
			withAdminGroup(t, super, a)
			beforeA, beforeB := stored(t, super, a), stored(t, super, b)
			if n := count(t, super, "SELECT count(*) FROM tenant_admin_group WHERE tenant_id = $1", a); n != 1 {
				t.Fatalf("setup: %d admin group rows", n)
			}

			k.SetFault(a, fault)
			res := onceAll(t, s)
			if res[a].Err == nil {
				t.Error("A: want an error")
			}
			if res[b].Err != nil {
				t.Errorf("B: %v", res[b].Err)
			}
			if after := stored(t, super, a); after != beforeA {
				t.Errorf("A changed:\n before %s\n after  %s", beforeA, after)
			}
			if after := stored(t, super, b); after != beforeB {
				t.Errorf("B changed:\n before %s\n after  %s", beforeB, after)
			}

			k.SetFault(a, keycloaktest.NoFault)
			res = onceAll(t, s)
			if res[a].Err != nil || res[b].Err != nil {
				t.Errorf("good cycle: A %v, B %v", res[a].Err, res[b].Err)
			}
			if after := stored(t, super, a); after != beforeA {
				t.Errorf("A changed after the fault went away:\n before %s\n after  %s", beforeA, after)
			}
		})
	}
}

// Keycloak's own answer that the root group does not exist still removes the
// tenant's directory: group sync follows Keycloak (ADR-001), the decision of
// TICK-1141. Green before and after the fix.
func TestKeycloakRootGroupGoneStillTombstonesTheTenant(t *testing.T) {
	k, s, super, a, b := vanishStart(t)
	withAdminGroup(t, super, a)
	beforeB := stored(t, super, b)

	k.SetFault(a, keycloaktest.KeycloakRootGone)
	res := onceAll(t, s)
	if res[a].Err != nil || res[b].Err != nil {
		t.Fatalf("results: A %v, B %v", res[a].Err, res[b].Err)
	}
	if n := count(t, super, "SELECT count(*) FROM principals WHERE tenant_id = $1 AND deleted_at IS NULL", a); n != 0 {
		t.Errorf("%d live principals of A", n)
	}
	if n := count(t, super, "SELECT count(*) FROM groups WHERE tenant_id = $1 AND deleted_at IS NULL", a); n != 0 {
		t.Errorf("%d live groups of A", n)
	}
	if n := count(t, super, "SELECT count(*) FROM tenant_admin_group WHERE tenant_id = $1", a); n != 0 {
		t.Errorf("%d admin group rows of A", n)
	}
	if after := stored(t, super, b); after != beforeB {
		t.Errorf("B changed:\n before %s\n after  %s", beforeB, after)
	}
}

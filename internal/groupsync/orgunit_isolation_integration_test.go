//go:build integration

package groupsync_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// TICK-952 (S31 P11-19) adversarial isolation tests, Security & Multi-Tenancy
// Reviewer. Attacker: whoever controls tenant A's directory snapshot. Target:
// tenant B's tombstoned org units (attack 5) and the primary unit the write
// pipeline would stamp on a ticket (attack 7).

// unitTombstones is "keycloak id:live" for every org unit of tid, sorted.
// tenantRows leaves deleted_at out, so an un-tombstone would not show there.
func unitTombstones(t *testing.T, super *pgx.Conn, tid uuid.UUID) string {
	t.Helper()
	var s *string
	if err := super.QueryRow(context.Background(), `SELECT string_agg(keycloak_group_id || ':' || id || ':' || (deleted_at IS NULL)::text || ':' || coalesce(deleted_at::text, '-'), ',' ORDER BY keycloak_group_id)
		FROM org_units WHERE tenant_id = $1`, tid).Scan(&s); err != nil {
		t.Fatal(err)
	}
	if s == nil {
		return ""
	}
	return *s
}

// Attack 5: tenant B tombstones ou-b-gone. Tenant A's snapshot then names
// ou-b-gone's Keycloak id, at B's path and at an A path, with B's member
// in it, and names B's live unit's id and path too. B's unit stays
// tombstoned with the same deleted_at, B's live unit is untouched, and
// whatever A gets is a new unit of A's own, never B's id.
func TestDirectoryPayloadNeverUntombstonesAnotherTenantsUnit(t *testing.T) {
	super, pool := cluster(t)
	a, b := twoTenants()
	seedTenant(t, super, a, 1)
	seedTenant(t, super, b, 2)
	dir := &fakeDir{snaps: map[uuid.UUID]identity.DirectoryChanges{}}
	s := newSyncer(t, pool, dir)
	pB := path(b, "")

	dir.set(a, identity.DirectoryChanges{})
	dir.set(b, identity.DirectoryChanges{
		Users: []identity.DirectoryUser{{KeycloakUserID: "u-b", Enabled: true}},
		OrgUnits: []identity.DirectoryGroup{
			memberUnit("ou-b-live", "", pB+"/ou/live", "u-b"),
			memberUnit("ou-b-gone", "", pB+"/ou/gone", "u-b"),
		},
	})
	syncAll(t, s)
	dir.set(b, identity.DirectoryChanges{
		Users:    []identity.DirectoryUser{{KeycloakUserID: "u-b", Enabled: true}},
		OrgUnits: []identity.DirectoryGroup{memberUnit("ou-b-live", "", pB+"/ou/live", "u-b")},
	})
	if r := syncAll(t, s)[b]; r.Err != nil {
		t.Fatalf("tenant B tombstones ou-b-gone: %v", r.Err)
	}
	if n := count(t, super, `SELECT count(*) FROM org_units WHERE tenant_id = $1 AND keycloak_group_id = 'ou-b-gone' AND deleted_at IS NOT NULL`, b); n != 1 {
		t.Fatalf("setup: ou-b-gone is not tombstoned in tenant B (%d rows)", n)
	}
	victimUnits, victimMembers, victimRows := unitTombstones(t, super, b), membership(t, super, b), tenantRows(t, super, b)

	for _, snap := range []identity.DirectoryChanges{
		{ // B's ids and B's paths: dropped as outside A's subtree.
			Users: []identity.DirectoryUser{{KeycloakUserID: "u-b", Enabled: true}},
			OrgUnits: []identity.DirectoryGroup{
				memberUnit("ou-b-gone", "", pB+"/ou/gone", "u-b"),
				memberUnit("ou-b-live", "", pB+"/ou/live", "u-b"),
			},
		},
		{ // B's ids at A's paths: A gets units of its own.
			Users: []identity.DirectoryUser{{KeycloakUserID: "u-b", Enabled: true}},
			OrgUnits: []identity.DirectoryGroup{
				memberUnit("ou-b-gone", "", path(a, "/ou/gone"), "u-b"),
				memberUnit("ou-b-live", "ou-b-gone", path(a, "/ou/gone/live"), "u-b"),
			},
		},
		{}, // and A tombstones them again
	} {
		dir.set(a, snap)
		if r := syncAll(t, s)[a]; r.Err != nil {
			t.Fatalf("tenant A: %v", r.Err)
		}
		if got := unitTombstones(t, super, b); got != victimUnits {
			t.Fatalf("tenant A's snapshot changed tenant B's units:\n before %s\n after  %s", victimUnits, got)
		}
		if got := membership(t, super, b); got != victimMembers {
			t.Fatalf("tenant A's snapshot changed tenant B's memberships:\n before %s\n after  %s", victimMembers, got)
		}
		if got := tenantRows(t, super, b); got != victimRows {
			t.Fatalf("tenant A's snapshot changed tenant B's rows")
		}
		if n := count(t, super, `SELECT count(*) FROM org_units a JOIN org_units b ON b.tenant_id = $2 AND b.id = a.id WHERE a.tenant_id = $1`, a, b); n != 0 {
			t.Errorf("tenant A holds %d org units on tenant B's ids", n)
		}
	}
}

// pipelinePrimaryUnit is the write pipeline's lookup (store.primaryUnitSQL,
// TICK-949), run as nava_groupsync in tenant's transaction under RLS: the
// unit a new ticket of the user gets as org_unit_id, nil for NULL. Repeated
// here because only the pipeline may import internal/records/store (D-3-6).
func pipelinePrimaryUnit(t *testing.T, super *pgx.Conn, pool *pgxpool.Pool, tenant uuid.UUID, kcUser string) *uuid.UUID {
	t.Helper()
	var principal uuid.UUID
	if err := super.QueryRow(context.Background(), `SELECT id FROM principals WHERE tenant_id = $1 AND keycloak_user_id = $2`, tenant, kcUser).Scan(&principal); err != nil {
		t.Fatalf("principal %s: %v", kcUser, err)
	}
	var unit *uuid.UUID
	if err := tenancy.InTenantTx(context.Background(), pool, tenant, func(tx pgx.Tx) error {
		return tx.QueryRow(context.Background(), `SELECT (SELECT m.org_unit_id FROM org_unit_members m WHERE m.tenant_id = $1 AND m.principal_id = $2 AND m.is_primary)`,
			tenant, principal).Scan(&unit)
	}); err != nil {
		t.Fatal(err)
	}
	return unit
}

// Attack 7 (Solution Architect CHANGES on TICK-931, 2026-11-15): a user's
// nava_primary_org_unit names a group path that exists only in tenant B. With
// the user in no A unit, or in two, no primary is set, so the pipeline's
// lookup returns NULL and a new ticket's org_unit_id stays NULL. B's own user
// with the same Keycloak id keeps B's primary.
func TestPrimaryOrgUnitPathOnlyInAnotherTenantSetsNoPrimary(t *testing.T) {
	super, pool := cluster(t)
	a, b := twoTenants()
	seedTenant(t, super, a, 1)
	seedTenant(t, super, b, 2)
	dir := &fakeDir{snaps: map[uuid.UUID]identity.DirectoryChanges{}}
	s := newSyncer(t, pool, dir)
	pB := path(b, "")
	dir.set(b, identity.DirectoryChanges{
		Users:    []identity.DirectoryUser{orgUser("u-x", pB+"/ou/only-b")},
		OrgUnits: []identity.DirectoryGroup{memberUnit("ou-only-b", "", pB+"/ou/only-b", "u-x")},
	})
	dir.set(a, identity.DirectoryChanges{})
	syncAll(t, s)
	bPrimary := pipelinePrimaryUnit(t, super, pool, b, "u-x")
	if bPrimary == nil {
		t.Fatal("setup: tenant B's u-x has no primary unit")
	}

	for _, tc := range []struct {
		name  string
		units []identity.DirectoryGroup
	}{
		{"in no unit of A", nil},
		{"in two units of A", []identity.DirectoryGroup{
			memberUnit("ou-a-1", "", path(a, "/ou/one"), "u-x"),
			memberUnit("ou-a-2", "", path(a, "/ou/two"), "u-x"),
		}},
		{"in two units of A, one carrying B's unit's Keycloak id", []identity.DirectoryGroup{
			memberUnit("ou-only-b", "", path(a, "/ou/imposter"), "u-x"),
			memberUnit("ou-a-1", "", path(a, "/ou/one"), "u-x"),
		}},
	} {
		dir.set(a, identity.DirectoryChanges{Users: []identity.DirectoryUser{orgUser("u-x", pB+"/ou/only-b")}, OrgUnits: tc.units})
		if r := syncAll(t, s)[a]; r.Err != nil {
			t.Fatalf("%s: tenant A: %v", tc.name, r.Err)
		}
		if got := pipelinePrimaryUnit(t, super, pool, a, "u-x"); got != nil {
			t.Errorf("%s: tenant A's u-x got primary unit %s, want none (a new ticket's org_unit_id must stay NULL)", tc.name, got)
		}
		if got := pipelinePrimaryUnit(t, super, pool, b, "u-x"); got == nil || *got != *bPrimary {
			t.Errorf("%s: tenant B's u-x primary = %v, want %s", tc.name, got, bPrimary)
		}
	}
}

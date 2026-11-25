//go:build integration

package groupsync_test

import (
	"testing"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/identity"
)

// TICK-948 adversarial isolation tests (Security & Multi-Tenancy Reviewer).
// Attacker: whoever controls tenant A's directory snapshot, including a user's
// nava_primary_org_unit attribute. Target: tenant B's org units and
// org_unit_members.

// victimOrgSnapshot is tenant B's directory: u-shared and u-b are members of
// B's emea unit, and u-shared's attribute names it.
func victimOrgSnapshot(tB string) identity.DirectoryChanges {
	return identity.DirectoryChanges{
		Users: []identity.DirectoryUser{
			{KeycloakUserID: "u-shared", Enabled: true, PrimaryOrgUnitPath: tB + "/ou/emea"},
			{KeycloakUserID: "u-b", Enabled: true},
		},
		OrgUnits: []identity.DirectoryGroup{
			memberUnit("ou-b-emea", "", tB+"/ou/emea", "u-shared", "u-b"),
			memberUnit("ou-b-apac", "", tB+"/ou/apac", "u-b"),
		},
	}
}

// Tenant A's user names tenant B's unit as primary, tenant A's snapshot
// injects that B unit (B's path, and B's Keycloak id under A's path), and the
// user is a member everywhere. The primary never resolves to a unit outside A,
// no A membership points at a B unit, and B's rows never change.
func TestPrimaryOrgUnitNeverResolvesAcrossTenants(t *testing.T) {
	super, pool := cluster(t)
	a, b := twoTenants()
	seedTenant(t, super, a, 1)
	seedTenant(t, super, b, 2)
	dir := &fakeDir{snaps: map[uuid.UUID]identity.DirectoryChanges{}}
	s := newSyncer(t, pool, dir)
	pB := path(b, "")

	dir.set(b, victimOrgSnapshot(pB))
	dir.set(a, identity.DirectoryChanges{})
	for id, r := range syncAll(t, s) {
		if r.Err != nil {
			t.Fatalf("tenant %s: %v", id, r.Err)
		}
	}
	victimRows, victimMembers := tenantRows(t, super, b), membership(t, super, b)
	if want := "ou-b-apac:u-b:false,ou-b-emea:u-b:false,ou-b-emea:u-shared:true"; victimMembers != want {
		t.Fatalf("tenant B seed membership = %s, want %s", victimMembers, want)
	}

	hostileUsers := func(attr string) []identity.DirectoryUser {
		return []identity.DirectoryUser{
			{KeycloakUserID: "u-shared", Enabled: true, PrimaryOrgUnitPath: attr},
			{KeycloakUserID: "u-b", Enabled: true, PrimaryOrgUnitPath: attr},
		}
	}
	steps := []struct {
		name  string
		users []identity.DirectoryUser
		units []identity.DirectoryGroup
		want  string // tenant A's membership afterwards
	}{
		{
			// Attribute = B's path; A's snapshot also carries a unit at B's
			// path (dropped as outside A's subtree) and B's Keycloak id at an
			// A path. Two A units each: no primary.
			name:  "attribute names B's path",
			users: hostileUsers(pB + "/ou/emea"),
			units: []identity.DirectoryGroup{
				memberUnit("ou-b-emea", "", path(a, "/ou/emea"), "u-shared", "u-b"),
				memberUnit("ou-a-x", "", path(a, "/ou/x"), "u-shared", "u-b"),
				memberUnit("ou-b-apac", "", pB+"/ou/apac", "u-shared", "u-b"),
			},
			want: "ou-a-x:u-b:false,ou-a-x:u-shared:false,ou-b-emea:u-b:false,ou-b-emea:u-shared:false",
		},
		{
			// Same, with the attribute pointing at B's path while the user is
			// in one A unit: the single-unit fallback picks A's unit, never B's.
			name:  "attribute names B's path, one A unit",
			users: hostileUsers(pB + "/ou/apac"),
			units: []identity.DirectoryGroup{
				memberUnit("ou-a-x", "", path(a, "/ou/x"), "u-shared", "u-b"),
				memberUnit("ou-b-apac", "", pB+"/ou/apac", "u-shared", "u-b"),
			},
			want: "ou-a-x:u-b:true,ou-a-x:u-shared:true",
		},
		{
			// Attribute = an A unit that has been tombstoned (ou-b-emea left
			// A's snapshot in the step above): never primary.
			name:  "attribute names a tombstoned A unit",
			users: hostileUsers(path(a, "/ou/emea")),
			units: []identity.DirectoryGroup{
				memberUnit("ou-a-x", "", path(a, "/ou/x"), "u-shared", "u-b"),
				memberUnit("ou-a-y", "", path(a, "/ou/y"), "u-shared", "u-b"),
			},
			want: "ou-a-x:u-b:false,ou-a-x:u-shared:false,ou-a-y:u-b:false,ou-a-y:u-shared:false",
		},
		{
			// A's directory empties: A's memberships are deleted, B's stay.
			name: "A empties",
			want: "",
		},
	}
	for _, st := range steps {
		dir.set(a, identity.DirectoryChanges{Users: st.users, OrgUnits: st.units})
		if r := syncAll(t, s)[a]; r.Err != nil {
			t.Fatalf("%s: tenant A: %v", st.name, r.Err)
		}
		if got := membership(t, super, a); got != st.want {
			t.Errorf("%s: tenant A membership =\n %s\nwant\n %s", st.name, got, st.want)
		}
		if got := membership(t, super, b); got != victimMembers {
			t.Fatalf("%s: tenant A's sync changed tenant B's memberships:\n before %s\n after  %s", st.name, victimMembers, got)
		}
		if got := tenantRows(t, super, b); got != victimRows {
			t.Fatalf("%s: tenant A's sync changed tenant B's rows:\n before %s\n after  %s", st.name, victimRows, got)
		}
		if n := count(t, super, `SELECT count(*) FROM org_unit_members m WHERE m.tenant_id = $1
			AND (m.org_unit_id IN (SELECT id FROM org_units WHERE tenant_id = $2)
			  OR m.principal_id IN (SELECT id FROM principals WHERE tenant_id = $2))`, a, b); n != 0 {
			t.Errorf("%s: tenant A has %d org_unit_members rows pointing at tenant B's units or principals", st.name, n)
		}
		if n := count(t, super, `SELECT count(*) FROM org_unit_members m JOIN org_units o
			ON o.tenant_id = m.tenant_id AND o.id = m.org_unit_id
			WHERE m.tenant_id = $1 AND (o.deleted_at IS NOT NULL OR m.is_primary AND o.keycloak_group_id NOT LIKE 'ou-a-%')`, a); n != 0 {
			t.Errorf("%s: tenant A has %d memberships of a tombstoned unit or a primary outside A's own units", st.name, n)
		}
	}

	// B still syncs, and still owns its primary.
	if r := syncAll(t, s)[b]; r.Err != nil {
		t.Fatalf("tenant B after A's attacks: %v", r.Err)
	}
	if got := membership(t, super, b); got != victimMembers {
		t.Errorf("tenant B membership after its own re-sync = %s, want %s", got, victimMembers)
	}
}

//go:build integration

package groupsync_test

import (
	"testing"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/identity"
)

// TICK-963: principals, groups and org units minted by a sync get UUIDv7 ids.
func TestSyncMintsV7Ids(t *testing.T) {
	super, pool := cluster(t)
	tid := uuid.New()
	seedTenant(t, super, tid, 1)
	dir := &fakeDir{snaps: map[uuid.UUID]identity.DirectoryChanges{}}
	s := newSyncer(t, pool, dir)
	dir.set(tid, identity.DirectoryChanges{
		Users:    []identity.DirectoryUser{{KeycloakUserID: "u1", Email: "a@example.test", Enabled: true}},
		Groups:   []identity.DirectoryGroup{{KeycloakGroupID: "g1", Path: path(tid, "/agents"), Name: "agents", MemberKeycloakUserIDs: []string{"u1"}}},
		OrgUnits: []identity.DirectoryGroup{unit("A", "", path(tid, "/ou/A")), unit("B", "A", path(tid, "/ou/A/B"))},
	})
	if r := once(t, s); r.Err != nil {
		t.Fatal(r.Err)
	}
	for _, table := range []string{"principals", "groups", "org_units"} {
		rows, err := super.Query(t.Context(), "SELECT id FROM "+table+" WHERE tenant_id = $1", tid)
		if err != nil {
			t.Fatal(err)
		}
		n := 0
		for rows.Next() {
			var id uuid.UUID
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				t.Fatal(err)
			}
			n++
			if id.Version() != 7 {
				t.Errorf("%s id %s is version %d, want 7", table, id, id.Version())
			}
		}
		rows.Close()
		if n == 0 {
			t.Errorf("%s: no rows written", table)
		}
	}
}

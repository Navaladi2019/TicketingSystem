//go:build integration

// TICK-1113 against a real PostgreSQL: a user deleted while the real
// keycloak.Directory reads a tenant's snapshot is left out of that cycle, and
// no other tenant is touched.

package groupsync_test

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel/metric/noop"

	"github.com/Navaladi2019/TicketingSystem/internal/groupsync"
	"github.com/Navaladi2019/TicketingSystem/internal/identity/keycloak/keycloaktest"
)

// rows dumps what a cycle writes for a tenant, with xmin so a rewrite shows.
func rows(t *testing.T, super *pgx.Conn, tid uuid.UUID) string {
	t.Helper()
	var s string
	err := super.QueryRow(context.Background(), `SELECT
	  (SELECT coalesce(string_agg(keycloak_user_id||'/'||enabled||'/'||(deleted_at IS NOT NULL)||'/'||xmin::text, ',' ORDER BY keycloak_user_id), '') FROM principals WHERE tenant_id = $1)
	  || '|' || (SELECT coalesce(string_agg(keycloak_group_id||'/'||(deleted_at IS NOT NULL)||'/'||xmin::text, ',' ORDER BY keycloak_group_id), '') FROM groups WHERE tenant_id = $1)
	  || '|' || (SELECT coalesce(string_agg(p.keycloak_user_id, ',' ORDER BY p.keycloak_user_id), '') FROM group_members m JOIN principals p ON p.tenant_id = m.tenant_id AND p.id = m.principal_id WHERE m.tenant_id = $1)`, tid).Scan(&s)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func principalXmin(t *testing.T, super *pgx.Conn, tid uuid.UUID, kcUser string) string {
	t.Helper()
	var x string
	if err := super.QueryRow(context.Background(), "SELECT xmin::text FROM principals WHERE tenant_id = $1 AND keycloak_user_id = $2", tid, kcUser).Scan(&x); err != nil {
		t.Fatal(err)
	}
	return x
}

func onceAll(t *testing.T, s *groupsync.Syncer) map[uuid.UUID]groupsync.Result {
	t.Helper()
	rs, err := s.Once(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	m := map[uuid.UUID]groupsync.Result{}
	for _, r := range rs {
		m[r.TenantID] = r
	}
	return m
}

func vanishStart(t *testing.T) (*keycloaktest.Vanish, *groupsync.Syncer, *pgx.Conn, uuid.UUID, uuid.UUID) {
	t.Helper()
	super, pool := cluster(t)
	a, b := uuid.New(), uuid.New()
	seedTenant(t, super, a, 1)
	seedTenant(t, super, b, 2)
	k := keycloaktest.NewVanish(t, a, b)
	s, err := groupsync.New(pool, k.Directory(), noop.NewMeterProvider(), slog.New(slog.DiscardHandler), groupsync.Options{MaxOrgUnitDepth: 8})
	if err != nil {
		t.Fatal(err)
	}
	res := onceAll(t, s)
	for _, tn := range []uuid.UUID{a, b} {
		if res[tn].Err != nil {
			t.Fatalf("first cycle %s: %v", tn, res[tn].Err)
		}
		if n := count(t, super, "SELECT count(*) FROM principals WHERE tenant_id = $1 AND enabled", tn); n != 2 {
			t.Fatalf("tenant %s has %d principals, want 2", tn, n)
		}
	}
	return k, s, super, a, b
}

func TestUserDeletedDuringSnapshotIsTombstonedOnlyInItsTenant(t *testing.T) {
	k, s, super, a, b := vanishStart(t)
	gone, kept := k.User(a, 2), k.User(a, 1)
	beforeB := rows(t, super, b)
	keptXmin := principalXmin(t, super, a, kept)
	k.Delete(gone, http.StatusNotFound) // still in the member list: deleted after it was read
	res := onceAll(t, s)
	if res[a].Err != nil || res[b].Err != nil {
		t.Fatalf("results: A %v, B %v", res[a].Err, res[b].Err)
	}
	if n := count(t, super, "SELECT count(*) FROM principals WHERE tenant_id = $1 AND keycloak_user_id = $2 AND NOT enabled AND deleted_at IS NOT NULL", a, gone); n != 1 {
		t.Errorf("the gone user is not tombstoned")
	}
	if n := count(t, super, "SELECT count(*) FROM group_members m JOIN principals p ON p.tenant_id = m.tenant_id AND p.id = m.principal_id WHERE m.tenant_id = $1 AND p.keycloak_user_id = $2", a, gone); n != 0 {
		t.Errorf("the gone user still has a membership")
	}
	if n := count(t, super, "SELECT count(*) FROM principals p JOIN group_members m ON m.tenant_id = p.tenant_id AND m.principal_id = p.id WHERE p.tenant_id = $1 AND p.keycloak_user_id = $2 AND p.enabled AND p.deleted_at IS NULL", a, kept); n != 1 {
		t.Errorf("A's other user changed")
	}
	if x := principalXmin(t, super, a, kept); x != keptXmin {
		t.Errorf("A's other user was rewritten: xmin %s, was %s", x, keptXmin)
	}
	if after := rows(t, super, b); after != beforeB {
		t.Errorf("tenant B changed:\n before %s\n after  %s", beforeB, after)
	}
	// The member list now agrees with the deletion: nothing to write.
	k.Unlist(gone)
	res = onceAll(t, s)
	if res[a].Err != nil || !res[a].Unchanged || res[b].Err != nil || !res[b].Unchanged {
		t.Errorf("further cycle: A %+v, B %+v", res[a], res[b])
	}
	k.CheckUserReads(t)
}

func TestUserStillThereOrUnreadableFailsOnlyItsTenant(t *testing.T) {
	for _, code := range []int{http.StatusOK, http.StatusInternalServerError} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			k, s, super, a, b := vanishStart(t)
			gone := k.User(a, 2)
			beforeA := rows(t, super, a)
			k.Delete(gone, code)
			k.Unlist(k.User(b, 2)) // B changes too, so a cycle that reaches B shows
			res := onceAll(t, s)
			if res[a].Err == nil {
				t.Errorf("A: want an error")
			}
			if res[b].Err != nil {
				t.Errorf("B: %v", res[b].Err)
			}
			if after := rows(t, super, a); after != beforeA {
				t.Errorf("A changed:\n before %s\n after  %s", beforeA, after)
			}
			if n := count(t, super, "SELECT count(*) FROM group_members WHERE tenant_id = $1", b); n != 1 {
				t.Errorf("B did not sync: %d memberships, want 1", n)
			}
			k.CheckUserReads(t)
		})
	}
}

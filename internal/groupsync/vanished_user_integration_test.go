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
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel/metric/noop"

	"github.com/Navaladi2019/TicketingSystem/internal/groupsync"
	"github.com/Navaladi2019/TicketingSystem/internal/identity/keycloak"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
)

// vanishKC is an httptest Keycloak with one group and two users per tenant.
// A user can answer 404 on its federated-identity read, and its own record
// (GET /users/{id}) answers a status the test sets.
type vanishKC struct {
	mu       sync.Mutex
	tenants  []uuid.UUID
	fedGone  map[string]bool // user id -> federated-identity answers 404
	userCode map[string]int  // user id -> status of GET /users/{id}
	inMember map[string]bool // user id -> listed in the group's members
	userReqs []string
}

func (k *vanishKC) uid(tenant uuid.UUID, n int) string { return fmt.Sprintf("kc-%s-%d", tenant, n) }

func (k *vanishKC) members(tenant uuid.UUID) string {
	var us []string
	for n := 1; n <= 2; n++ {
		id := k.uid(tenant, n)
		if k.inMember[id] {
			us = append(us, fmt.Sprintf(`{"id":%q,"email":"u%d@%s.test","enabled":true}`, id, n, tenant))
		}
	}
	return "[" + strings.Join(us, ",") + "]"
}

func (k *vanishKC) handler(w http.ResponseWriter, r *http.Request) {
	k.mu.Lock()
	defer k.mu.Unlock()
	p := r.URL.Path
	if strings.HasSuffix(p, "/protocol/openid-connect/token") {
		fmt.Fprint(w, `{"access_token":"tok","expires_in":300}`)
		return
	}
	p = strings.TrimPrefix(p, "/admin/realms/nava")
	for _, t := range k.tenants {
		root, grp := "root-"+t.String(), "grp-"+t.String()
		switch p {
		case "/group-by-path/t/" + t.String():
			fmt.Fprintf(w, `{"id":%q,"name":"r","path":"/t/%s","subGroupCount":1}`, root, t)
			return
		case "/groups/" + root + "/children":
			fmt.Fprintf(w, `[{"id":%q,"name":"agents","path":"/t/%s/agents","subGroupCount":0}]`, grp, t)
			return
		case "/groups/" + root + "/members":
			fmt.Fprint(w, `[]`)
			return
		case "/groups/" + grp + "/members":
			fmt.Fprint(w, k.members(t))
			return
		}
	}
	if rest, ok := strings.CutPrefix(p, "/users/"); ok {
		k.userReqs = append(k.userReqs, rest)
		if id, fed := strings.CutSuffix(rest, "/federated-identity"); fed {
			if k.fedGone[id] {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			fmt.Fprint(w, `[]`)
			return
		}
		code := k.userCode[rest]
		if code == 0 {
			code = http.StatusOK
		}
		w.WriteHeader(code)
		fmt.Fprintf(w, `{"id":%q}`, rest)
		return
	}
	w.WriteHeader(http.StatusNotFound)
}

type noReg struct{}

func (noReg) UserKeycloakID(context.Context, uuid.UUID, uuid.UUID) (string, error) { return "", nil }
func (noReg) GroupKeycloakID(context.Context, uuid.UUID, uuid.UUID) (string, error) {
	return "", nil
}
func (noReg) BoundIdPAliases(context.Context, uuid.UUID) ([]string, error) { return nil, nil }
func (noReg) TenantOwnsEmailDomain(context.Context, uuid.UUID, string) (bool, error) {
	return false, nil
}

func realDirectory(t *testing.T, url string) *keycloak.Directory {
	t.Helper()
	cfg, err := config.Load(func(string) (string, bool) { return "", false })
	if err != nil {
		t.Fatal(err)
	}
	tel, err := telemetry.New(t.Context(), cfg.Telemetry)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tel.Shutdown(context.Background()) })
	out, err := telemetry.NewOutboundClient(tel, telemetry.OutboundOptions{})
	if err != nil {
		t.Fatal(err)
	}
	d, err := keycloak.NewDirectory(keycloak.DirectoryConfig{
		BaseURL: url, Runtime: keycloak.ClientCredentials{ClientID: "nava-groupsync", ClientSecret: "s"},
		Registry: noReg{}, Logger: slog.New(slog.DiscardHandler),
	}, out)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

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

func vanishStart(t *testing.T) (*vanishKC, *groupsync.Syncer, *pgx.Conn, uuid.UUID, uuid.UUID) {
	t.Helper()
	super, pool := cluster(t)
	a, b := uuid.New(), uuid.New()
	seedTenant(t, super, a, 1)
	seedTenant(t, super, b, 2)
	k := &vanishKC{tenants: []uuid.UUID{a, b}, fedGone: map[string]bool{}, userCode: map[string]int{}, inMember: map[string]bool{}}
	for _, tn := range k.tenants {
		k.inMember[k.uid(tn, 1)], k.inMember[k.uid(tn, 2)] = true, true
	}
	srv := httptest.NewServer(http.HandlerFunc(k.handler))
	t.Cleanup(srv.Close)
	s, err := groupsync.New(pool, realDirectory(t, srv.URL), noop.NewMeterProvider(), slog.New(slog.DiscardHandler), groupsync.Options{MaxOrgUnitDepth: 8})
	if err != nil {
		t.Fatal(err)
	}
	res := onceAll(t, s)
	for _, tn := range k.tenants {
		if res[tn].Err != nil {
			t.Fatalf("first cycle %s: %v", tn, res[tn].Err)
		}
		if n := count(t, super, "SELECT count(*) FROM principals WHERE tenant_id = $1 AND enabled", tn); n != 2 {
			t.Fatalf("tenant %s has %d principals, want 2", tn, n)
		}
	}
	return k, s, super, a, b
}

// checkUserReads: every /users/ request is for a user that a member list named.
func (k *vanishKC) checkUserReads(t *testing.T) {
	t.Helper()
	k.mu.Lock()
	defer k.mu.Unlock()
	for _, r := range k.userReqs {
		id := strings.TrimSuffix(r, "/federated-identity")
		known := false
		for _, tn := range k.tenants {
			known = known || id == k.uid(tn, 1) || id == k.uid(tn, 2)
		}
		if !known {
			t.Errorf("a /users/ request for %q, which no member list named", r)
		}
	}
}

func TestUserDeletedDuringSnapshotIsTombstonedOnlyInItsTenant(t *testing.T) {
	k, s, super, a, b := vanishStart(t)
	gone, kept := k.uid(a, 2), k.uid(a, 1)
	beforeB := rows(t, super, b)
	k.mu.Lock()
	k.fedGone[gone], k.userCode[gone] = true, http.StatusNotFound // still in the member list: deleted after it was read
	k.mu.Unlock()
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
	if after := rows(t, super, b); after != beforeB {
		t.Errorf("tenant B changed:\n before %s\n after  %s", beforeB, after)
	}
	// The member list now agrees with the deletion: nothing to write.
	k.mu.Lock()
	k.inMember[gone] = false
	k.mu.Unlock()
	res = onceAll(t, s)
	if res[a].Err != nil || !res[a].Unchanged || res[b].Err != nil || !res[b].Unchanged {
		t.Errorf("further cycle: A %+v, B %+v", res[a], res[b])
	}
	k.checkUserReads(t)
}

func TestUserStillThereOrUnreadableFailsOnlyItsTenant(t *testing.T) {
	for _, code := range []int{http.StatusOK, http.StatusInternalServerError} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			k, s, super, a, b := vanishStart(t)
			gone := k.uid(a, 2)
			beforeA := rows(t, super, a)
			k.mu.Lock()
			k.fedGone[gone], k.userCode[gone] = true, code
			k.inMember[k.uid(b, 2)] = false // B changes too, so a cycle that reaches B shows
			k.mu.Unlock()
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
			k.checkUserReads(t)
		})
	}
}

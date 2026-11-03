package keycloak

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/identity"
)

// routeKC is a table-driven fake admin API: "METHOD /path" (no query) maps to
// a status and JSON body. Unlisted routes answer 404. Every request is kept.
type routeKC struct {
	mu     sync.Mutex
	routes map[string]kcAnswer
	seen   []string
}

type kcAnswer struct {
	status int
	body   string
	loc    string
}

func ok(body string) kcAnswer { return kcAnswer{status: http.StatusOK, body: body} }
func st(code int) kcAnswer    { return kcAnswer{status: code} }
func (r *routeKC) saw(route string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, s := range r.seen {
		if s == route {
			return true
		}
	}
	return false
}

func newRouteKC(t *testing.T, routes map[string]kcAnswer) (*routeKC, string) {
	t.Helper()
	k := &routeKC{routes: routes}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/protocol/openid-connect/token") {
			fmt.Fprint(w, `{"access_token":"tok","expires_in":300}`)
			return
		}
		key := r.Method + " " + strings.TrimPrefix(r.URL.Path, "/admin/realms/nava")
		k.mu.Lock()
		k.seen = append(k.seen, key)
		a, found := k.routes[key]
		k.mu.Unlock()
		if !found {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, `{}`)
			return
		}
		if a.loc != "" {
			w.Header().Set("Location", a.loc)
		}
		w.WriteHeader(a.status)
		fmt.Fprint(w, a.body)
	}))
	t.Cleanup(srv.Close)
	return k, srv.URL
}

// cfgRegistry is a Registry whose answers a test sets.
type cfgRegistry struct {
	users, groups map[uuid.UUID]string
	bound         []string
	boundErr      error
	owns          bool
	ownsErr       error
}

func (r cfgRegistry) UserKeycloakID(_ context.Context, _ uuid.UUID, p uuid.UUID) (string, error) {
	if id, ok := r.users[p]; ok {
		return id, nil
	}
	return "", ErrRegistryNotFound
}

func (r cfgRegistry) GroupKeycloakID(_ context.Context, _ uuid.UUID, g uuid.UUID) (string, error) {
	if id, ok := r.groups[g]; ok {
		return id, nil
	}
	return "", ErrRegistryNotFound
}

func (r cfgRegistry) BoundIdPAliases(context.Context, uuid.UUID) ([]string, error) {
	return r.bound, r.boundErr
}

func (r cfgRegistry) TenantOwnsEmailDomain(context.Context, uuid.UUID, string) (bool, error) {
	return r.owns, r.ownsErr
}

//nolint:gochecknoglobals // read-only test fixtures
var (
	tA        = dirTenantA
	tB        = uuid.MustParse("00000000-0000-4000-8000-0000000000b2")
	pUser     = uuid.MustParse("00000000-0000-4000-8000-000000000011")
	pGroup    = uuid.MustParse("00000000-0000-4000-8000-000000000022")
	rootA     = "/t/" + tA.String()
	covGroupA = fmt.Sprintf(`{"id":"g1","name":"x","path":%q}`, rootA+"/x")
	covGroupB = fmt.Sprintf(`{"id":"gb","name":"x","path":%q}`, "/t/"+tB.String()+"/x")
	rootAJS   = fmt.Sprintf(`{"id":"g-root","name":%q,"path":%q}`, tA.String(), rootA)
)

func TestIdPConfigKeyAllowed(t *testing.T) {
	t.Parallel()
	if !IdPConfigKeyAllowed("oidc", "clientId") || IdPConfigKeyAllowed("oidc", "entityId") {
		t.Fatal("oidc allow-list wrong")
	}
	if !IdPConfigKeyAllowed("saml", "entityId") || IdPConfigKeyAllowed("saml", "clientId") {
		t.Fatal("saml allow-list wrong")
	}
	if IdPConfigKeyAllowed("oidc", "kc.org.broker.redirect.mode.email-matches") {
		t.Fatal("policy key must be refused")
	}
}

func TestSetGroupMembers(t *testing.T) {
	t.Parallel()
	reg := cfgRegistry{
		users:  map[uuid.UUID]string{pUser: "u-new"},
		groups: map[uuid.UUID]string{pGroup: "g1"},
		bound:  []string{"idp-a"},
	}
	base := func() map[string]kcAnswer {
		return map[string]kcAnswer{
			"GET /groups/g1":                      ok(covGroupA),
			"GET /groups/g1/members":              ok(`[{"id":"u-old"}]`),
			"GET /users/u-new/federated-identity": ok(`[]`),
			"GET /users/u-new/groups":             ok(fmt.Sprintf(`[{"id":"g0","path":%q}]`, rootA)),
			"PUT /users/u-new/groups/g1":          st(204),
			"DELETE /users/u-old/groups/g1":       st(204),
		}
	}
	run := func(t *testing.T, routes map[string]kcAnswer, mut func(*cfgRegistry), tenant uuid.UUID, group uuid.UUID, members []uuid.UUID) (*routeKC, error) {
		t.Helper()
		r := reg
		if mut != nil {
			mut(&r)
		}
		kc, url := newRouteKC(t, routes)
		d, _ := directoryFor(t, url, func(c *DirectoryConfig) { c.Registry = r })
		return kc, d.SetGroupMembers(t.Context(), tenant, group, members)
	}

	t.Run("adds and removes", func(t *testing.T) {
		t.Parallel()
		kc, err := run(t, base(), nil, tA, pGroup, []uuid.UUID{pUser})
		if err != nil {
			t.Fatal(err)
		}
		if !kc.saw("PUT /users/u-new/groups/g1") || !kc.saw("DELETE /users/u-old/groups/g1") {
			t.Fatalf("writes missing: %v", kc.seen)
		}
	})
	t.Run("no tenant", func(t *testing.T) {
		t.Parallel()
		_, err := run(t, base(), nil, uuid.Nil, pGroup, nil)
		if !errors.Is(err, identity.ErrInvalidInput) {
			t.Fatalf("got %v", err)
		}
	})
	t.Run("unknown group", func(t *testing.T) {
		t.Parallel()
		_, err := run(t, base(), nil, tA, uuid.New(), nil)
		if !errors.Is(err, identity.ErrNotFound) {
			t.Fatalf("got %v", err)
		}
	})
	t.Run("group of another tenant is not found and nothing is written", func(t *testing.T) {
		t.Parallel()
		r := base()
		r["GET /groups/g1"] = ok(covGroupB)
		kc, err := run(t, r, nil, tA, pGroup, []uuid.UUID{pUser})
		if !errors.Is(err, identity.ErrNotFound) {
			t.Fatalf("got %v", err)
		}
		if kc.saw("PUT /users/u-new/groups/g1") || kc.saw("DELETE /users/u-old/groups/g1") {
			t.Fatal("wrote across tenants")
		}
	})
	t.Run("group read fails", func(t *testing.T) {
		t.Parallel()
		r := base()
		r["GET /groups/g1"] = st(500)
		if _, err := run(t, r, nil, tA, pGroup, nil); err == nil || errors.Is(err, identity.ErrNotFound) {
			t.Fatalf("got %v", err)
		}
	})
	t.Run("unknown member", func(t *testing.T) {
		t.Parallel()
		_, err := run(t, base(), nil, tA, pGroup, []uuid.UUID{uuid.New()})
		if !errors.Is(err, identity.ErrNotFound) {
			t.Fatalf("got %v", err)
		}
	})
	t.Run("members read fails", func(t *testing.T) {
		t.Parallel()
		r := base()
		r["GET /groups/g1/members"] = st(500)
		if _, err := run(t, r, nil, tA, pGroup, nil); err == nil {
			t.Fatal("want error")
		}
	})
	t.Run("added user federated to unbound idp", func(t *testing.T) {
		t.Parallel()
		r := base()
		r["GET /users/u-new/federated-identity"] = ok(`[{"identityProvider":"idp-evil","userId":"x"}]`)
		kc, err := run(t, r, nil, tA, pGroup, []uuid.UUID{pUser})
		if !errors.Is(err, identity.ErrIdPNotBound) {
			t.Fatalf("got %v", err)
		}
		if kc.saw("PUT /users/u-new/groups/g1") || kc.saw("DELETE /users/u-old/groups/g1") {
			t.Fatal("changed despite refusal")
		}
	})
	t.Run("bound idps lookup fails", func(t *testing.T) {
		t.Parallel()
		r := base()
		r["GET /users/u-new/federated-identity"] = ok(`[{"identityProvider":"idp-a","userId":"x"}]`)
		boom := errors.New("registry down")
		_, err := run(t, r, func(c *cfgRegistry) { c.boundErr = boom }, tA, pGroup, []uuid.UUID{pUser})
		if !errors.Is(err, boom) {
			t.Fatalf("got %v", err)
		}
		_, err = run(t, r, func(c *cfgRegistry) { c.boundErr = ErrRegistryNotFound }, tA, pGroup, []uuid.UUID{pUser})
		if !errors.Is(err, identity.ErrNotFound) {
			t.Fatalf("got %v", err)
		}
	})
	t.Run("federated links read fails", func(t *testing.T) {
		t.Parallel()
		r := base()
		r["GET /users/u-new/federated-identity"] = st(500)
		if _, err := run(t, r, nil, tA, pGroup, []uuid.UUID{pUser}); err == nil {
			t.Fatal("want error")
		}
	})
	t.Run("added user in no tenant group is not found", func(t *testing.T) {
		t.Parallel()
		r := base()
		r["GET /users/u-new/groups"] = ok(fmt.Sprintf(`[{"id":"gb","path":%q}]`, "/t/"+tB.String()))
		kc, err := run(t, r, nil, tA, pGroup, []uuid.UUID{pUser})
		if !errors.Is(err, identity.ErrNotFound) {
			t.Fatalf("got %v", err)
		}
		if kc.saw("PUT /users/u-new/groups/g1") {
			t.Fatal("added a user of another tenant")
		}
	})
	t.Run("user groups read fails", func(t *testing.T) {
		t.Parallel()
		r := base()
		r["GET /users/u-new/groups"] = st(500)
		if _, err := run(t, r, nil, tA, pGroup, []uuid.UUID{pUser}); err == nil {
			t.Fatal("want error")
		}
	})
	t.Run("add write fails", func(t *testing.T) {
		t.Parallel()
		r := base()
		r["PUT /users/u-new/groups/g1"] = st(500)
		if _, err := run(t, r, nil, tA, pGroup, []uuid.UUID{pUser}); err == nil {
			t.Fatal("want error")
		}
	})
	t.Run("remove write fails", func(t *testing.T) {
		t.Parallel()
		r := base()
		r["DELETE /users/u-old/groups/g1"] = st(500)
		if _, err := run(t, r, nil, tA, pGroup, []uuid.UUID{pUser}); err == nil {
			t.Fatal("want error")
		}
	})
}

func TestDisableUser(t *testing.T) {
	t.Parallel()
	reg := cfgRegistry{users: map[uuid.UUID]string{pUser: "u1"}}
	base := func() map[string]kcAnswer {
		return map[string]kcAnswer{
			"GET /users/u1/groups":  ok(fmt.Sprintf(`[{"id":"g","path":%q}]`, rootA)),
			"GET /users/u1":         ok(`{"id":"u1","enabled":true,"attributes":{"k":["v"]}}`),
			"PUT /users/u1":         st(204),
			"POST /users/u1/logout": st(204),
		}
	}
	run := func(t *testing.T, routes map[string]kcAnswer, tenant, p uuid.UUID) (*routeKC, error) {
		t.Helper()
		kc, url := newRouteKC(t, routes)
		d, _ := directoryFor(t, url, func(c *DirectoryConfig) { c.Registry = reg })
		return kc, d.DisableUser(t.Context(), tenant, p)
	}
	if kc, err := run(t, base(), tA, pUser); err != nil || !kc.saw("POST /users/u1/logout") {
		t.Fatalf("happy: %v %v", err, kc.seen)
	}
	if _, err := run(t, base(), uuid.Nil, pUser); !errors.Is(err, identity.ErrInvalidInput) {
		t.Fatalf("no tenant: %v", err)
	}
	if _, err := run(t, base(), tA, uuid.New()); !errors.Is(err, identity.ErrNotFound) {
		t.Fatalf("unknown principal: %v", err)
	}
	// A user of another tenant is indistinguishable from an unknown one.
	other := base()
	other["GET /users/u1/groups"] = ok(fmt.Sprintf(`[{"id":"g","path":%q}]`, "/t/"+tB.String()))
	if kc, err := run(t, other, tA, pUser); !errors.Is(err, identity.ErrNotFound) || kc.saw("PUT /users/u1") {
		t.Fatalf("cross-tenant: %v %v", err, kc.seen)
	}
	// A user shared with another tenant must not be disabled realm-wide.
	shared := base()
	shared["GET /users/u1/groups"] = ok(fmt.Sprintf(`[{"id":"g","path":%q},{"id":"h","path":%q}]`, rootA, "/t/"+tB.String()))
	if kc, err := run(t, shared, tA, pUser); !errors.Is(err, identity.ErrInviteRefused) || kc.saw("PUT /users/u1") {
		t.Fatalf("shared: %v %v", err, kc.seen)
	}
	for _, route := range []string{"GET /users/u1/groups", "GET /users/u1", "PUT /users/u1", "POST /users/u1/logout"} {
		r := base()
		r[route] = st(500)
		if _, err := run(t, r, tA, pUser); err == nil {
			t.Fatalf("%s failing must fail DisableUser", route)
		}
	}
}

func TestChangesSince(t *testing.T) {
	t.Parallel()
	ou := rootA + "/ou"
	full := func() map[string]kcAnswer {
		return map[string]kcAnswer{
			"GET /group-by-path/t/" + tA.String(): ok(fmt.Sprintf(`{"id":"g-root","name":"r","path":%q,"subGroupCount":2}`, rootA)),
			"GET /groups/g-root/children": ok(fmt.Sprintf(`[
				{"id":"g-ou","name":"ou","path":%q,"subGroupCount":1},
				{"id":"g-admins","name":"admins","path":%q,"subGroupCount":0}]`, ou, rootA+"/admins")),
			"GET /groups/g-ou/children":        ok(fmt.Sprintf(`[{"id":"g-unit","name":"unit","path":%q,"subGroupCount":0}]`, ou+"/unit")),
			"GET /groups/g-root/members":       ok(`[{"id":"u1","email":"a@x.test","enabled":true}]`),
			"GET /groups/g-admins/members":     ok(`[{"id":"u1","email":"a@x.test","enabled":true},{"id":"u2","email":"b@x.test","enabled":false}]`),
			"GET /groups/g-unit/members":       ok(`[]`),
			"GET /users/u1/federated-identity": ok(`[{"identityProvider":"z","userId":"1"},{"identityProvider":"a","userId":"2"}]`),
			"GET /users/u2/federated-identity": ok(`[]`),
		}
	}
	dir := func(t *testing.T, routes map[string]kcAnswer) *Directory {
		t.Helper()
		_, url := newRouteKC(t, routes)
		d, _ := directoryFor(t, url)
		return d
	}

	t.Run("snapshot is sorted, split and stable", func(t *testing.T) {
		t.Parallel()
		d := dir(t, full())
		snap, digest, err := d.ChangesSince(t.Context(), tA, "")
		if err != nil {
			t.Fatal(err)
		}
		if len(snap.OrgUnits) != 1 || snap.OrgUnits[0].Name != "unit" || snap.OrgUnits[0].ParentKeycloakGroupID != "" {
			t.Fatalf("org units: %+v", snap.OrgUnits)
		}
		if len(snap.Groups) != 1 || snap.Groups[0].Name != "admins" || len(snap.Groups[0].MemberKeycloakUserIDs) != 2 {
			t.Fatalf("groups: %+v", snap.Groups)
		}
		if len(snap.Users) != 2 || snap.Users[0].KeycloakUserID != "u1" || snap.Users[1].Enabled {
			t.Fatalf("users: %+v", snap.Users)
		}
		if l := snap.Users[0].FederatedLinks; len(l) != 2 || l[0].IdPAlias != "a" {
			t.Fatalf("links not sorted: %+v", l)
		}
		again, digest2, err := d.ChangesSince(t.Context(), tA, digest)
		if err != nil || !again.Unchanged || digest2 != digest {
			t.Fatalf("cursor: %+v %v", again, err)
		}
	})
	t.Run("no tenant", func(t *testing.T) {
		t.Parallel()
		if _, _, err := dir(t, full()).ChangesSince(t.Context(), uuid.Nil, ""); !errors.Is(err, identity.ErrInvalidInput) {
			t.Fatalf("got %v", err)
		}
	})
	t.Run("tenant without root group is empty", func(t *testing.T) {
		t.Parallel()
		snap, _, err := dir(t, map[string]kcAnswer{}).ChangesSince(t.Context(), tA, "")
		if err != nil || len(snap.Users)+len(snap.Groups)+len(snap.OrgUnits) != 0 {
			t.Fatalf("got %+v %v", snap, err)
		}
	})
	t.Run("root path mismatch is treated as absent", func(t *testing.T) {
		t.Parallel()
		r := full()
		r["GET /group-by-path/t/"+tA.String()] = ok(fmt.Sprintf(`{"id":"g-x","path":%q}`, "/t/"+tB.String()))
		snap, _, err := dir(t, r).ChangesSince(t.Context(), tA, "")
		if err != nil || len(snap.Groups) != 0 {
			t.Fatalf("got %+v %v", snap, err)
		}
	})
	t.Run("a group outside the tenant subtree stops the walk", func(t *testing.T) {
		t.Parallel()
		r := full()
		r["GET /groups/g-root/children"] = ok(fmt.Sprintf(`[{"id":"gx","name":"x","path":%q}]`, "/t/"+tB.String()+"/x"))
		if _, _, err := dir(t, r).ChangesSince(t.Context(), tA, ""); err == nil {
			t.Fatal("cross-tenant group must fail the snapshot")
		}
	})
	for _, route := range []string{
		"GET /group-by-path/t/" + tA.String(), "GET /groups/g-root/children", "GET /groups/g-ou/children",
		"GET /groups/g-root/members", "GET /groups/g-admins/members", "GET /users/u1/federated-identity",
	} {
		t.Run("read failure "+route, func(t *testing.T) {
			t.Parallel()
			r := full()
			r[route] = st(500)
			if _, _, err := dir(t, r).ChangesSince(t.Context(), tA, ""); err == nil {
				t.Fatal("want error")
			}
		})
	}
}

func TestDirectoryTooLarge(t *testing.T) {
	t.Parallel()
	var kids []string
	for i := 0; i <= maxDirectoryItems; i++ {
		kids = append(kids, fmt.Sprintf(`{"id":"k%d","path":%q}`, i, rootA+"/g"))
	}
	body := "[" + strings.Join(kids, ",") + "]"
	_, url := newRouteKC(t, map[string]kcAnswer{
		"GET /group-by-path/t/" + tA.String(): ok(fmt.Sprintf(`{"id":"g-root","path":%q}`, rootA)),
		"GET /groups/g-root/children":         ok(body),
	})
	d, _ := directoryFor(t, url)
	if _, _, err := d.ChangesSince(t.Context(), tA, ""); err == nil {
		t.Fatal("want too-large error")
	}
}

func TestEnsureAdminGroupRoutes(t *testing.T) {
	t.Parallel()
	adminPath := rootA + "/admins"
	adminJS := fmt.Sprintf(`{"id":"g-admins","path":%q}`, adminPath)
	rootRoute := "GET /group-by-path/t/" + tA.String()
	adminRoute := "GET /group-by-path/t/" + tA.String() + "/admins"
	run := func(t *testing.T, routes map[string]kcAnswer, tenant uuid.UUID) (string, *routeKC, error) {
		t.Helper()
		kc, url := newRouteKC(t, routes)
		d, _ := directoryFor(t, url)
		id, err := d.EnsureAdminGroup(t.Context(), tenant)
		return id, kc, err
	}
	if _, _, err := run(t, nil, uuid.Nil); !errors.Is(err, identity.ErrInvalidInput) {
		t.Fatalf("no tenant: %v", err)
	}
	if id, _, err := run(t, map[string]kcAnswer{rootRoute: ok(rootAJS), adminRoute: ok(adminJS)}, tA); err != nil || id != "g-admins" {
		t.Fatalf("existing: %q %v", id, err)
	}
	// A wrong-path admins group is not trusted.
	if _, _, err := run(t, map[string]kcAnswer{rootRoute: ok(rootAJS), adminRoute: ok(`{"id":"evil","path":"/t/other/admins"}`),
		"POST /groups/g-root/children": {status: 201}}, tA); err == nil {
		t.Fatal("admins group at the wrong path must not be accepted")
	}
	// Admin group lookup keeps failing after a create: error, not an empty id.
	if id, _, err := run(t, map[string]kcAnswer{rootRoute: ok(rootAJS), "POST /groups/g-root/children": {status: 201}}, tA); err == nil || id != "" {
		t.Fatalf("still missing: %q %v", id, err)
	}
	// 409 on create is tolerated (a concurrent creator), then the re-read decides.
	if _, _, err := run(t, map[string]kcAnswer{rootRoute: ok(rootAJS), "POST /groups/g-root/children": st(409)}, tA); !errors.Is(err, identity.ErrNotFound) {
		t.Fatalf("409 then missing: %v", err)
	}
	if _, _, err := run(t, map[string]kcAnswer{rootRoute: ok(rootAJS), "POST /groups/g-root/children": st(500)}, tA); err == nil || errors.Is(err, identity.ErrNotFound) {
		t.Fatalf("create fails: %v", err)
	}
	if _, _, err := run(t, map[string]kcAnswer{rootRoute: ok(rootAJS), adminRoute: st(500)}, tA); err == nil || errors.Is(err, identity.ErrNotFound) {
		t.Fatalf("lookup fails: %v", err)
	}
}

func TestEnsureRootGroup(t *testing.T) {
	t.Parallel()
	rootRoute := "GET /group-by-path/t/" + tA.String()
	run := func(t *testing.T, routes map[string]kcAnswer) error {
		t.Helper()
		_, url := newRouteKC(t, routes)
		d, _ := directoryFor(t, url)
		_, err := d.ensureRootGroup(t.Context(), tA)
		return err
	}
	if err := run(t, map[string]kcAnswer{rootRoute: st(500)}); err == nil || errors.Is(err, identity.ErrNotFound) {
		t.Fatalf("root read error must propagate: %v", err)
	}
	if err := run(t, map[string]kcAnswer{"GET /group-by-path/t": st(500)}); err == nil || errors.Is(err, identity.ErrNotFound) {
		t.Fatalf("/t read error: %v", err)
	}
	if err := run(t, map[string]kcAnswer{"GET /group-by-path/t": ok(`{"id":"top","path":"/t"}`),
		"POST /groups/top/children": st(500)}); err == nil {
		t.Fatal("create error")
	}
	// 409 then still missing: the final lookup is not found.
	if err := run(t, map[string]kcAnswer{"GET /group-by-path/t": ok(`{"id":"top","path":"/t"}`),
		"POST /groups/top/children": st(409)}); !errors.Is(err, identity.ErrNotFound) {
		t.Fatalf("got %v", err)
	}
}

func TestInviteUserBranches(t *testing.T) {
	t.Parallel()
	email := "dave@tenant-a.test"
	rootRoute := "GET /group-by-path/t/" + tA.String()
	base := func() map[string]kcAnswer {
		return map[string]kcAnswer{
			rootRoute:                          ok(rootAJS),
			"GET /users":                       ok(`[{"id":"u9","email":"dave@tenant-a.test"}]`),
			"GET /users/u9/groups":             ok(`[]`),
			"GET /users/u9/federated-identity": ok(`[{"identityProvider":"idp-a","userId":"1"}]`),
			"PUT /users/u9/groups/g-root":      st(204),
			"GET /group-by-path/t/" + tA.String() + "/admins": ok(fmt.Sprintf(`{"id":"g-admins","path":%q}`, rootA+"/admins")),
			"PUT /users/u9/groups/g-admins":                   st(204),
		}
	}
	reg := cfgRegistry{bound: []string{"idp-a"}, owns: true}
	run := func(t *testing.T, routes map[string]kcAnswer, mut func(*cfgRegistry), inv *recordedInvites, e string) (*routeKC, error) {
		t.Helper()
		r := reg
		if mut != nil {
			mut(&r)
		}
		kc, url := newRouteKC(t, routes)
		d, _ := directoryFor(t, url, func(c *DirectoryConfig) {
			c.Registry = r
			if inv != nil {
				c.Invites = inv
			}
		})
		return kc, d.InviteUser(t.Context(), tA, e)
	}

	if kc, err := run(t, base(), nil, nil, email); err != nil || !kc.saw("PUT /users/u9/groups/g-admins") {
		t.Fatalf("bound existing user: %v %v", err, kc.seen)
	}
	if _, err := run(t, base(), nil, nil, "not an email"); !errors.Is(err, identity.ErrInvalidInput) {
		t.Fatalf("bad email: %v", err)
	}
	// Existing user already in the tenant: idempotent, no federated check.
	in := base()
	in["GET /users/u9/groups"] = ok(fmt.Sprintf(`[{"id":"g","path":%q}]`, rootA+"/x"))
	in["GET /users/u9/federated-identity"] = st(500)
	if _, err := run(t, in, nil, nil, email); err != nil {
		t.Fatalf("idempotent: %v", err)
	}
	// In another tenant: refused, nothing written.
	other := base()
	other["GET /users/u9/groups"] = ok(fmt.Sprintf(`[{"id":"g","path":%q}]`, "/t/"+tB.String()))
	if kc, err := run(t, other, nil, nil, email); !errors.Is(err, identity.ErrInviteRefused) || kc.saw("PUT /users/u9/groups/g-root") {
		t.Fatalf("other tenant: %v %v", err, kc.seen)
	}
	// Local user (no federated links): refused.
	local := base()
	local["GET /users/u9/federated-identity"] = ok(`[]`)
	if _, err := run(t, local, nil, nil, email); !errors.Is(err, identity.ErrInviteRefused) {
		t.Fatalf("local: %v", err)
	}
	// Federated to an IdP the tenant does not bind: refused.
	unbound := base()
	unbound["GET /users/u9/federated-identity"] = ok(`[{"identityProvider":"idp-evil","userId":"1"}]`)
	if kc, err := run(t, unbound, nil, nil, email); !errors.Is(err, identity.ErrInviteRefused) || kc.saw("PUT /users/u9/groups/g-root") {
		t.Fatalf("unbound: %v", err)
	}
	// Bound-idps lookup errors.
	boom := errors.New("registry down")
	if _, err := run(t, base(), func(r *cfgRegistry) { r.boundErr = boom }, nil, email); !errors.Is(err, boom) {
		t.Fatalf("bound err: %v", err)
	}
	if _, err := run(t, base(), func(r *cfgRegistry) { r.boundErr = ErrRegistryNotFound }, nil, email); !errors.Is(err, identity.ErrNotFound) {
		t.Fatalf("bound not found: %v", err)
	}
	// Read/write failures along the way.
	for _, route := range []string{rootRoute, "GET /users", "GET /users/u9/groups", "GET /users/u9/federated-identity",
		"PUT /users/u9/groups/g-root", "PUT /users/u9/groups/g-admins"} {
		r := base()
		r[route] = st(500)
		if _, err := run(t, r, nil, nil, email); err == nil {
			t.Fatalf("%s failing must fail the invite", route)
		}
	}
	// New email: the domain registry decides.
	newUser := base()
	newUser["GET /users"] = ok(`[]`)
	if _, err := run(t, newUser, func(r *cfgRegistry) { r.owns = false }, nil, email); !errors.Is(err, identity.ErrInviteRefused) {
		t.Fatalf("unowned domain: %v", err)
	}
	if _, err := run(t, newUser, func(r *cfgRegistry) { r.ownsErr = boom }, nil, email); !errors.Is(err, boom) {
		t.Fatalf("owns err: %v", err)
	}
	if _, err := run(t, newUser, func(r *cfgRegistry) { r.ownsErr = ErrRegistryNotFound }, nil, email); !errors.Is(err, identity.ErrNotFound) {
		t.Fatalf("owns not found: %v", err)
	}
	if _, err := run(t, newUser, nil, nil, email); err == nil || !strings.Contains(err.Error(), "not configured") {
		t.Fatalf("no invite writer: %v", err)
	}
	inv := &recordedInvites{}
	if kc, err := run(t, newUser, nil, inv, email); err != nil || len(inv.rows) != 1 || inv.rows[0].tenant != tA || kc.saw("POST /users") {
		t.Fatalf("pending invite: %v %+v", err, inv.rows)
	}
}

func TestDigestOfRejectsUnmarshalable(t *testing.T) {
	t.Parallel()
	// A digest is stable for equal snapshots and differs for different ones.
	a, err := digestOf(identity.DirectoryChanges{})
	b, err2 := digestOf(identity.DirectoryChanges{Users: []identity.DirectoryUser{{KeycloakUserID: "u"}}})
	if err != nil || err2 != nil || a == b || len(a) != 64 {
		t.Fatalf("digests: %q %q %v %v", a, b, err, err2)
	}
	if _, err := json.Marshal(identity.DirectoryChanges{}); err != nil {
		t.Fatal(err)
	}
}

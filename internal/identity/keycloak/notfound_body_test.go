package keycloak

import (
	"context"
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

// TICK-1141: a 404 is identity.ErrNotFound only when its JSON "error" is
// Keycloak's own string for the route. The strings are literals here so that
// a wrong constant in admin.go fails these tests.
func TestStatusErrorNotFoundNeedsKeycloakBody(t *testing.T) {
	t.Parallel()
	type row struct{ route, below, errStr string }
	table := []row{
		{"/users/{id}", "/users/{id}/federated-identity", "User not found"},
		{"/group-by-path/t/{tenant}", "/group-by-path/t/{tenant}/admins", "Group path does not exist"},
		{"/groups/{id}", "/groups/{id}/children", "Could not find group by id"},
		{"/identity-provider/instances/{alias}", "/identity-provider/instances/{alias}/mappers", "HTTP 404 Not Found"},
	}
	var mu sync.Mutex
	var status int
	var body string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/token") {
			fmt.Fprint(w, `{"access_token":"tok","expires_in":300}`)
			return
		}
		mu.Lock()
		st, b := status, body
		mu.Unlock()
		w.WriteHeader(st)
		fmt.Fprint(w, b)
	}))
	t.Cleanup(srv.Close)
	a, acct := adminFor(t, srv.URL)
	call := func(route string, st int, b string) error {
		mu.Lock()
		status, body = st, b
		mu.Unlock()
		_, err := a.do(t.Context(), acct, http.MethodGet, route, "/x", nil, nil)
		return err
	}
	js := func(s string) string { return fmt.Sprintf(`{"error":%q}`, s) }

	for _, r := range table {
		for _, route := range []string{r.route, r.below} {
			if err := call(route, 404, js(r.errStr)); !errors.Is(err, identity.ErrNotFound) {
				t.Errorf("%s: own string: %v, want ErrNotFound", route, err)
			}
			for name, b := range map[string]string{
				"empty": "", "{}": `{}`, "html": `<html>not found</html>`, "array": `["` + r.errStr + `"]`,
				"number": `{"error":1}`, "errorMessage": `{"errorMessage":"` + r.errStr + `"}`,
				"realm":          `{"error":"Realm not found."}`,
				"no method":      `{"error":"Unable to find matching target resource method"}`,
				"trailing space": js(r.errStr + " "), "513 bytes": js(strings.Repeat("a", 513)),
			} {
				if err := call(route, 404, b); err == nil || errors.Is(err, identity.ErrNotFound) {
					t.Errorf("%s with %s: %v, want a plain error", route, name, err)
				}
			}
			for _, o := range table {
				if o.errStr == r.errStr {
					continue
				}
				if err := call(route, 404, js(o.errStr)); errors.Is(err, identity.ErrNotFound) {
					t.Errorf("%s with the string of %s matched", route, o.route)
				}
			}
			if err := call(route, 400, js(r.errStr)); errors.Is(err, identity.ErrNotFound) {
				t.Errorf("%s: status 400 matched", route)
			}
		}
	}
	// Routes outside the table never match, with any of the four strings.
	for _, route := range []string{"/users", "/organizations/{id}", "/groups", "/identity-provider/instances", "/usersx/{id}", "/realm"} {
		for _, r := range table {
			if err := call(route, 404, js(r.errStr)); errors.Is(err, identity.ErrNotFound) {
				t.Errorf("%s with %q matched", route, r.errStr)
			}
		}
	}
	// Error() never carries the body's string.
	err := call("/users/{id}", 404, js("User not found"))
	if err == nil || strings.Contains(err.Error(), "User not found") {
		t.Fatalf("Error() = %v", err)
	}
	// The token endpoint is not in the table, whatever its body says.
	for _, r := range table {
		e := &statusError{method: "POST", route: grantRoute, status: 404, kcError: r.errStr}
		if errors.Is(e, identity.ErrNotFound) {
			t.Errorf("token endpoint matched with %q", r.errStr)
		}
	}
}

func TestForeign404ChangesNothingOnOnboarding(t *testing.T) {
	t.Parallel()
	idpPath := "/identity-provider/instances/" + TenantIdPAlias(tA)
	writes := func(kc *seqKC) []string {
		var w []string
		for _, s := range kc.seen {
			if !strings.HasPrefix(s, "GET ") {
				w = append(w, s)
			}
		}
		return w
	}
	t.Run("FindTenantOrg sends no DELETE", func(t *testing.T) {
		t.Parallel()
		kc, url := newSeqKC(t, map[string][]kcAnswer{
			"GET /organizations":          one(ok(`[{"id":"org-1","alias":"tenant-a"}]`)),
			"GET " + idpPath:              one(proxy404()),
			"DELETE /organizations/org-1": one(st(204)),
		})
		_, found, err := onboardingDir(t, url, oidcSpec, nil).FindTenantOrg(t.Context(), tA, "tenant-a")
		if err == nil || found || errors.Is(err, identity.ErrNotFound) {
			t.Fatalf("found %v err %v, want a plain error", found, err)
		}
		if w := writes(kc); len(w) != 0 {
			t.Fatalf("sent %v", w)
		}
	})
	t.Run("CreateTenantOrg sends no write", func(t *testing.T) {
		t.Parallel()
		kc, url := newSeqKC(t, map[string][]kcAnswer{
			"GET " + idpPath:                    one(proxy404()),
			"POST /organizations":               one(kcAnswer{status: 201, loc: "http://x/admin/realms/nava/organizations/org-1"}),
			"POST /identity-provider/instances": one(st(201)),
		})
		_, err := onboardingDir(t, url, oidcSpec, nil).CreateTenantOrg(t.Context(), tA, "tenant-a", nil)
		if err == nil || errors.Is(err, identity.ErrAlreadyExists) {
			t.Fatalf("err = %v, want a plain error", err)
		}
		if w := writes(kc); len(w) != 0 {
			t.Fatalf("sent %v", w)
		}
	})
}

func TestForeign404WritesNothing(t *testing.T) {
	t.Parallel()
	rootRoute := "GET /group-by-path/t/" + tA.String()
	noWrites := func(t *testing.T, kc *routeKC) {
		t.Helper()
		for _, s := range kc.seen {
			if !strings.HasPrefix(s, "GET ") {
				t.Fatalf("sent %s", s)
			}
		}
	}
	plain := func(t *testing.T, err error) {
		t.Helper()
		if err == nil || errors.Is(err, identity.ErrNotFound) {
			t.Fatalf("err = %v, want a plain error", err)
		}
	}
	routes := func(extra map[string]kcAnswer) map[string]kcAnswer {
		r := map[string]kcAnswer{"GET /group-by-path/t": ok(`{"id":"top","path":"/t"}`), "POST /groups/top/children": st(201)}
		for k, v := range extra {
			r[k] = v
		}
		return r
	}
	t.Run("ensureRootGroup, proxy 404", func(t *testing.T) {
		t.Parallel()
		kc, url := newRouteKC(t, routes(map[string]kcAnswer{rootRoute: proxy404()}))
		d, _ := directoryFor(t, url)
		_, err := d.ensureRootGroup(t.Context(), tA)
		plain(t, err)
		noWrites(t, kc)
	})
	t.Run("ensureRootGroup, 200 with {}", func(t *testing.T) {
		t.Parallel()
		kc, url := newRouteKC(t, routes(map[string]kcAnswer{rootRoute: ok(`{}`)}))
		d, _ := directoryFor(t, url)
		_, err := d.ensureRootGroup(t.Context(), tA)
		plain(t, err)
		noWrites(t, kc)
	})
	t.Run("ensureTenantGroupReport, proxy 404 of the group read", func(t *testing.T) {
		t.Parallel()
		kc, url := newRouteKC(t, routes(map[string]kcAnswer{rootRoute: ok(rootAJS), rootRoute + "/admins": proxy404()}))
		d, _ := directoryFor(t, url)
		_, _, err := d.ensureTenantGroupReport(t.Context(), tA, "admins")
		plain(t, err)
		noWrites(t, kc)
	})
	t.Run("PlaceInvitee, proxy 404 of the user read", func(t *testing.T) {
		t.Parallel()
		kc, url := newRouteKC(t, map[string]kcAnswer{
			"GET /users":                  ok(`[{"id":"u1","email":"erin@tenant-a.test","enabled":true}]`),
			"GET /users/u1":               proxy404(),
			"PUT /users/u1/groups/g-root": st(204),
		})
		d, _ := directoryFor(t, url)
		_, err := d.PlaceInvitee(t.Context(), tA, "erin@tenant-a.test", []string{"idp-a"}, []string{"org-1"},
			func(context.Context) (bool, error) { return true, nil })
		plain(t, err)
		noWrites(t, kc)
	})
	t.Run("SetGroupMembers, proxy 404 of the group read", func(t *testing.T) {
		t.Parallel()
		kc, url := newRouteKC(t, map[string]kcAnswer{"GET /groups/g1": proxy404(), "PUT /users/u-new/groups/g1": st(204)})
		reg := cfgRegistry{users: map[uuid.UUID]string{pUser: "u-new"}, groups: map[uuid.UUID]string{pGroup: "g1"}}
		d, _ := directoryFor(t, url, func(c *DirectoryConfig) { c.Registry = reg })
		plain(t, d.SetGroupMembers(t.Context(), tA, pGroup, []uuid.UUID{pUser}))
		noWrites(t, kc)
	})
	t.Run("DisableUser, proxy 404 of the user read", func(t *testing.T) {
		t.Parallel()
		kc, url := newRouteKC(t, map[string]kcAnswer{
			"GET /users/u1/groups": ok(fmt.Sprintf(`[{"id":"g","path":%q}]`, rootA)),
			"GET /users/u1":        proxy404(),
			"PUT /users/u1":        st(204),
		})
		reg := cfgRegistry{users: map[uuid.UUID]string{pUser: "u1"}}
		d, _ := directoryFor(t, url, func(c *DirectoryConfig) { c.Registry = reg })
		plain(t, d.DisableUser(t.Context(), tA, pUser))
		noWrites(t, kc)
	})
}

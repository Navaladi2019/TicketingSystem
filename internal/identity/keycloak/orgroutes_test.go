package keycloak

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

// TestRuntimeAccountOrganizationRoutes pins what admin.do lets each account
// send under /organizations (TICK-252, AR-034). The check reads the path that
// is sent, before the token fetch, so every refused call reaches the fake
// server zero times, token endpoint included.
func TestRuntimeAccountOrganizationRoutes(t *testing.T) {
	t.Parallel()
	var hits atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		fmt.Fprint(w, `{"access_token":"tok","expires_in":300}`)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	d, _ := directoryFor(t, srv.URL, withOnboarding)

	refused := []struct{ method, path string }{
		{http.MethodGet, "/organizations"},
		{http.MethodGet, "/organizations/o1/members"},
		{http.MethodPut, "/organizations/o1"},
		{http.MethodGet, "/organizations/members/u1/organizations/x"},
		{http.MethodPost, "/organizations/members/u1/organizations"},
		{http.MethodGet, "/organizations/members//organizations"},
		{http.MethodGet, "/organizations/members/a/b/organizations"},
		{http.MethodGet, "/users/../organizations"},
		{http.MethodGet, "/users/%2e%2e/organizations"},
		{http.MethodGet, "/users/%2F..%2Forganizations"},
	}
	for _, c := range refused {
		_, err := d.api.do(t.Context(), d.runtime, c.method, c.path, c.path, nil, nil)
		if !errors.Is(err, ErrOrgRouteDenied) {
			t.Errorf("runtime %s %s: err = %v, want ErrOrgRouteDenied", c.method, c.path, err)
		}
	}
	if n := hits.Load(); n != 0 {
		t.Fatalf("refused calls sent %d requests, want 0", n)
	}

	// The one allowed route works, and so does a route outside /organizations.
	for _, p := range []string{"/organizations/members/u1/organizations", "/users"} {
		if _, err := d.api.do(t.Context(), d.runtime, http.MethodGet, p, p, nil, nil); err != nil {
			t.Errorf("runtime GET %s: %v", p, err)
		}
	}
	// The onboarding account's organization calls still work.
	for _, c := range []struct{ method, path string }{
		{http.MethodGet, "/organizations"},
		{http.MethodPut, "/organizations/o1"},
		{http.MethodGet, "/organizations/o1/members"},
	} {
		if _, err := d.api.do(t.Context(), d.onboarding, c.method, c.path, c.path, nil, nil); err != nil {
			t.Errorf("onboarding %s %s: %v", c.method, c.path, err)
		}
	}
	if hits.Load() == 0 {
		t.Error("allowed calls never reached the server")
	}
}

// An account built without an organization policy gets none.
func TestAccountWithoutPolicyHasNoOrganizationRoute(t *testing.T) {
	t.Parallel()
	if (&account{}).allows(http.MethodGet, "/organizations/members/u1/organizations") {
		t.Error("zero-value account may call /organizations")
	}
}

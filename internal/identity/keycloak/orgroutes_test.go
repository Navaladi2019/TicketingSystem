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
// send under /organizations (TICK-252, AR-038). The check reads the path that
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
		// Non-canonical spellings of an organization route (TICK-255).
		{http.MethodGet, "/organizations;x"},
		{http.MethodGet, "/organizations;x/o1/members"},
		{http.MethodGet, "/%6Frganizations"},
		{http.MethodGet, "/%6frganizations/o1/members"},
		{http.MethodGet, "//organizations"},
		{http.MethodGet, "/Organizations"},
		{http.MethodGet, "/ORGANIZATIONS/o1/members"},
		{http.MethodGet, "/organizations/members;x/u1/organizations"},
		{http.MethodGet, "/organizations/members/u1;x/organizations"},
		{http.MethodGet, "/Organizations/members/u1/organizations"},
		{http.MethodGet, "/organizations/members/u1/organizations/"},
		{http.MethodGet, "/organizations//members/u1/organizations"},
		{http.MethodGet, "/organizations/%6Dembers/u1/organizations"},
		{http.MethodGet, "/organizations/members/%2e%2e/organizations"},
		{http.MethodGet, "/users/%6Frganizations"},
		{http.MethodGet, "/users;x"},
		{http.MethodGet, "/organizations?first=0"},
		{http.MethodGet, "/organizations?x"},
		{http.MethodGet, "/organizations?/members/u/organizations"},
		{http.MethodGet, "/organizations/members/%252e%252e%252f%252e%252e/organizations"},
		{http.MethodGet, "/organizations#x"},
		{http.MethodGet, "/organizations?/o1/members"},
		{http.MethodGet, "/users?x"},
		{http.MethodGet, "/users\\x"},
		{http.MethodGet, "/organizations\\o1"},
		{http.MethodGet, "/users/a\nb"},
		{http.MethodGet, "/organizations/members/u1/organizations\x00"},
		{http.MethodGet, "/organizations/members/%2e%2e%252f%2e%2e/organizations"},
		{http.MethodGet, "/organizations/members/%5C..%5C/organizations"},
		{http.MethodGet, "/organizations/members/a%3Bb/organizations"},
		{http.MethodGet, "/organizations/members/a%3Fb/organizations"},
		{http.MethodGet, "/organizations/members/a%23b/organizations"},
		{http.MethodGet, "/organizations/members/a%0Ab/organizations"},
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
	for _, p := range []string{"/organizations/members/u1/organizations", "/organizations/members/a%40b%20c/organizations", "/users"} {
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

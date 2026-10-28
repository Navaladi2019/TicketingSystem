//go:build identitystack

package identitystack

// T21 (TICK-81): Directory.FindTenantOrg and the onboarding service's resume
// against the real realm `nava`. FindTenantOrg deletes a lone organization, so
// every branch is exercised live.

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/netip"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/identity/keycloak"
	"github.com/Navaladi2019/TicketingSystem/internal/onboard"
)

func orgsByAlias(t *testing.T, alias string) []string {
	t.Helper()
	_, b := adminRealm(t, "nava", http.MethodGet, "/organizations?exact=true&search="+alias, nil)
	var os []struct{ ID, Alias string }
	_ = json.Unmarshal(b, &os)
	var ids []string
	for _, o := range os {
		if o.Alias == alias {
			ids = append(ids, o.ID)
		}
	}
	return ids
}

func createOrg(t *testing.T, alias string) string {
	t.Helper()
	code, b := adminRealm(t, "nava", http.MethodPost, "/organizations", map[string]any{"name": alias, "alias": alias, "enabled": true})
	if code != http.StatusCreated {
		t.Fatalf("create org %s: %d %s", alias, code, b)
	}
	ids := orgsByAlias(t, alias)
	if len(ids) != 1 {
		t.Fatalf("org %s not found after create", alias)
	}
	return ids[0]
}

func createIdP(t *testing.T, alias string) {
	t.Helper()
	spec, _ := oidcSpec(t.Context(), uuid.Nil)
	code, b := adminRealm(t, "nava", http.MethodPost, "/identity-provider/instances", map[string]any{
		"alias": alias, "providerId": "oidc", "enabled": true, "config": spec.Config,
	})
	if code != http.StatusCreated {
		t.Fatalf("create idp %s: %d %s", alias, code, b)
	}
}

func orgIdPCount(t *testing.T, orgID string) int {
	t.Helper()
	_, b := adminRealm(t, "nava", http.MethodGet, "/organizations/"+orgID+"/identity-providers", nil)
	var l []json.RawMessage
	_ = json.Unmarshal(b, &l)
	return len(l)
}

func TestFindTenantOrgLive(t *testing.T) {
	d := newDirectory(t, newRegistry(), true, oidcSpec)

	t.Run("neither exists", func(t *testing.T) {
		tenant := uuid.New()
		alias := "t-" + tenant.String()
		_, found, err := d.FindTenantOrg(t.Context(), tenant, alias)
		if err != nil || found {
			t.Fatalf("found=%v err=%v", found, err)
		}
	})

	t.Run("organization and its IdP: reused", func(t *testing.T) {
		tenant := uuid.New()
		alias := "t-" + tenant.String()
		cleanupTenantOrg(t, tenant, alias)
		want, err := d.CreateTenantOrg(t.Context(), tenant, alias, []string{tenant.String()[:8] + ".test"})
		if err != nil {
			t.Fatal(err)
		}
		got, found, err := d.FindTenantOrg(t.Context(), tenant, alias)
		if err != nil || !found || got != want {
			t.Fatalf("got=%q found=%v err=%v, want %q", got, found, err, want)
		}
		if n := len(orgsByAlias(t, alias)); n != 1 {
			t.Errorf("organizations = %d, want 1", n)
		}
	})

	t.Run("organization alone: deleted", func(t *testing.T) {
		tenant := uuid.New()
		alias := "t-" + tenant.String()
		cleanupTenantOrg(t, tenant, alias)
		createOrg(t, alias)
		_, found, err := d.FindTenantOrg(t.Context(), tenant, alias)
		if err != nil || found {
			t.Fatalf("found=%v err=%v", found, err)
		}
		if n := len(orgsByAlias(t, alias)); n != 0 {
			t.Errorf("the lone organization was not deleted (%d left)", n)
		}
		// The resume then creates one org and one IdP.
		if _, err := d.CreateTenantOrg(t.Context(), tenant, alias, nil); err != nil {
			t.Fatalf("CreateTenantOrg after the cleanup: %v", err)
		}
	})

	t.Run("IdP alone: refused, IdP kept", func(t *testing.T) {
		tenant := uuid.New()
		alias := "t-" + tenant.String()
		cleanupTenantOrg(t, tenant, alias)
		idp := keycloak.TenantIdPAlias(tenant)
		createIdP(t, idp)
		_, found, err := d.FindTenantOrg(t.Context(), tenant, alias)
		if !errors.Is(err, identity.ErrAlreadyExists) || found {
			t.Fatalf("found=%v err=%v", found, err)
		}
		if code, _ := adminRealm(t, "nava", http.MethodGet, "/identity-provider/instances/"+idp, nil); code != http.StatusOK {
			t.Error("the IdP was deleted")
		}
	})

	t.Run("IdP linked to another org: refused, the other org keeps exactly one IdP", func(t *testing.T) {
		tenant := uuid.New()
		alias := "t-" + tenant.String()
		cleanupTenantOrg(t, tenant, alias)
		mine := createOrg(t, alias)
		idp := keycloak.TenantIdPAlias(tenant)
		createIdP(t, idp)
		other := createOrg(t, "other-"+tenant.String()[:8])
		t.Cleanup(func() {
			adminRealm(t, "nava", http.MethodDelete, "/organizations/"+other+"/identity-providers/"+idp, nil)
			adminRealm(t, "nava", http.MethodDelete, "/organizations/"+other, nil)
		})
		if code, b := adminRealm(t, "nava", http.MethodPost, "/organizations/"+other+"/identity-providers", idp); code != http.StatusNoContent {
			t.Fatalf("link idp to the other org: %d %s", code, b)
		}
		_, found, err := d.FindTenantOrg(t.Context(), tenant, alias)
		if !errors.Is(err, identity.ErrAlreadyExists) || found {
			t.Fatalf("found=%v err=%v", found, err)
		}
		if n := orgIdPCount(t, other); n != 1 {
			t.Errorf("the other organization has %d IdPs, want exactly 1", n)
		}
		if n := orgIdPCount(t, mine); n != 0 {
			t.Errorf("this tenant's organization was linked to an IdP (%d)", n)
		}
		if len(orgsByAlias(t, alias)) != 1 {
			t.Error("this tenant's organization was deleted by a refused find")
		}
	})
}

// ---- kill and retry against the real Keycloak ------------------------------

type resumeStore struct {
	mu      sync.Mutex
	active  bool
	hash    []byte
	binds   int
	orgID   string
	failing bool
	domains map[string]uuid.UUID
}

func (s *resumeStore) Lock(context.Context, uuid.UUID) (func(), error) { return func() {}, nil }

func (s *resumeStore) Begin(_ context.Context, id uuid.UUID, h []byte, domains []string) (onboard.State, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.hash == nil {
		s.hash = h
		for _, d := range domains {
			s.domains[d] = id
		}
	}
	return onboard.State{Active: s.active}, nil
}

func (s *resumeStore) Bind(_ context.Context, _ uuid.UUID, _, orgID, _ string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.failing {
		return errors.New("killed before the binding insert")
	}
	if s.orgID == "" {
		s.binds++
	}
	s.orgID = orgID
	return nil
}

func (s *resumeStore) Activate(context.Context, uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.active = true
	return nil
}

func (s *resumeStore) TenantOwnsEmailDomain(_ context.Context, id uuid.UUID, d string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.domains[d] == id, nil
}

type publicResolver struct{}

func (publicResolver) LookupNetIP(context.Context, string, string) ([]netip.Addr, error) {
	return []netip.Addr{netip.MustParseAddr("93.184.216.34")}, nil
}

// Org created, the process dies before the binding insert; the retry ends with
// exactly one organization, one IdP linked to it, and one binding.
func TestOnboardKillBeforeBindingThenRetryLive(t *testing.T) {
	op := identity.Operator{PrincipalID: "live-op-" + uuid.NewString()}
	key := uuid.NewString()
	tenant := onboard.TenantID(op.PrincipalID, key)
	alias := onboard.Alias(tenant)
	cleanupTenantOrg(t, tenant, alias)

	specs := onboard.NewSpecs()
	dir := newDirectory(t, newRegistry(), true, specs.Source)
	st := &resumeStore{failing: true, domains: map[string]uuid.UUID{}}
	svc, err := onboard.New(onboard.Options{
		Store: st, Directory: dir, Finder: dir, Domains: st, Specs: specs, Resolver: publicResolver{},
		Issuer: baseURL() + "/realms/nava", Log: slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	if err != nil {
		t.Fatal(err)
	}
	domain := "live-" + tenant.String()[:8] + ".test"
	req := onboard.Request{
		AdminEmail:   "admin@" + domain,
		EmailDomains: []string{domain},
		IdP: onboard.IdPRequest{ProviderID: "oidc", DisplayName: "Live", Config: map[string]string{
			"clientId": "nava-broker", "clientSecret": "dev-only-broker-secret", "clientAuthMethod": "client_secret_post",
			"authorizationUrl": "https://idp.example.org/auth", "tokenUrl": "https://idp.example.org/token",
			"jwksUrl": "https://idp.example.org/jwks", "useJwksUrl": "true",
		}},
	}
	if _, err := svc.Onboard(t.Context(), op, key, []byte("h"), req); err == nil {
		t.Fatal("the first call should fail at the binding")
	}
	if n := len(orgsByAlias(t, alias)); n != 1 || st.active {
		t.Fatalf("after the kill: %d organizations, active=%v", n, st.active)
	}
	st.failing = false
	res, err := svc.Onboard(t.Context(), op, key, []byte("h"), req)
	if err != nil || res.TenantID != tenant {
		t.Fatalf("retry: %+v, %v", res, err)
	}
	ids := orgsByAlias(t, alias)
	if len(ids) != 1 || st.binds != 1 || !st.active || st.orgID != ids[0] {
		t.Fatalf("orgs=%v binds=%d active=%v bound org=%q", ids, st.binds, st.active, st.orgID)
	}
	if n := orgIdPCount(t, ids[0]); n != 1 {
		t.Errorf("the organization has %d IdPs, want 1", n)
	}
	if !strings.HasPrefix(keycloak.TenantIdPAlias(tenant), "idp-") {
		t.Error("idp alias is not server-derived")
	}
}

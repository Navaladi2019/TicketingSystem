package keycloak

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/identity"
)

const findAlias = "t-tenant-a"

// findRealm is a fake Keycloak holding at most one organization and one IdP
// for the tenant, and recording every write.
type findRealm struct {
	orgs     string // JSON array answered to the organization search
	idp      string // JSON of the IdP, or "" for none
	mappers  string
	posts    atomic.Int32
	deletes  atomic.Int32
	orgDel   atomic.Bool
	idpDel   atomic.Bool
	otherDel atomic.Bool
}

func (f *findRealm) server(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /realms/nava/protocol/openid-connect/token", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{"access_token":"tok","expires_in":300}`)
	})
	mux.HandleFunc("GET /admin/realms/nava/organizations", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("search") != findAlias || r.URL.Query().Get("exact") != "true" {
			t.Errorf("organization search = %q", r.URL.RawQuery)
		}
		fmt.Fprint(w, f.orgs)
	})
	idpPath := "/admin/realms/nava/identity-provider/instances/" + TenantIdPAlias(dirTenantA)
	mux.HandleFunc("GET "+idpPath, func(w http.ResponseWriter, _ *http.Request) {
		if f.idp == "" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		fmt.Fprint(w, f.idp)
	})
	mux.HandleFunc("GET "+idpPath+"/mappers", func(w http.ResponseWriter, _ *http.Request) {
		if f.mappers == "" {
			fmt.Fprint(w, `[]`)
			return
		}
		fmt.Fprint(w, f.mappers)
	})
	mux.HandleFunc("POST "+idpPath+"/mappers", func(w http.ResponseWriter, r *http.Request) {
		f.posts.Add(1)
		raw, _ := io.ReadAll(r.Body)
		f.mappers = "[" + string(raw) + "]"
		var m struct {
			IdentityProviderMapper string            `json:"identityProviderMapper"`
			Config                 map[string]string `json:"config"`
		}
		if json.Unmarshal(raw, &m) != nil || m.IdentityProviderMapper != "oidc-user-attribute-idp-mapper" ||
			m.Config["claim"] != "email_verified" || m.Config["user.attribute"] != EmailVerifiedAttribute || m.Config["syncMode"] != "FORCE" {
			t.Errorf("mapper posted = %+v", m)
		}
		w.WriteHeader(http.StatusCreated)
	})
	mux.HandleFunc("DELETE /admin/realms/nava/organizations/org-1", func(w http.ResponseWriter, _ *http.Request) {
		f.deletes.Add(1)
		f.orgDel.Store(true)
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("DELETE "+idpPath, func(w http.ResponseWriter, _ *http.Request) {
		f.deletes.Add(1)
		f.idpDel.Store(true)
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("DELETE /admin/realms/nava/organizations/", func(w http.ResponseWriter, _ *http.Request) {
		f.deletes.Add(1)
		f.otherDel.Store(true)
		w.WriteHeader(http.StatusNoContent)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

const goodMapper = `[{"name":"nava-idp-email-verified","identityProviderMapper":"oidc-user-attribute-idp-mapper",` +
	`"config":{"syncMode":"FORCE","claim":"email_verified","user.attribute":"nava_idp_email_verified"}}]`

const (
	oneOrg  = `[{"id":"org-1","alias":"` + findAlias + `"}]`
	goodIdP = `{"alias":"x","trustEmail":false,"firstBrokerLoginFlowAlias":"nava-first-broker-login",` +
		`"organizationLinks":[{"organizationId":"org-1"}]}`
)

func TestFindTenantOrg(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name      string
		orgs, idp string
		mappers   string
		wantOrg   string
		wantFound bool
		wantErr   error
		wantOrgDe bool
		wantPosts int32
	}{
		{name: "neither exists", orgs: `[]`},
		// An IdP a crash left without its mapper gets it, once.
		{name: "org and policy-conforming idp: reused, mapper repaired", orgs: oneOrg, idp: goodIdP, wantOrg: "org-1", wantFound: true, wantPosts: 1},
		{name: "org and idp with the email_verified mapper: reused", orgs: oneOrg, idp: goodIdP, mappers: goodMapper, wantOrg: "org-1", wantFound: true},
		{name: "idp with the mapper but a wrong claim", orgs: oneOrg, idp: goodIdP, wantErr: identity.ErrAlreadyExists,
			mappers: `[{"name":"nava-idp-email-verified","identityProviderMapper":"oidc-user-attribute-idp-mapper","config":{"syncMode":"FORCE","claim":"email","user.attribute":"nava_idp_email_verified"}}]`},
		{name: "org alone: deleted, not found", orgs: oneOrg, wantOrgDe: true},
		{name: "idp alone", orgs: `[]`, idp: goodIdP, wantErr: identity.ErrAlreadyExists},
		{
			name: "idp linked to another org", orgs: oneOrg,
			idp:     `{"trustEmail":false,"firstBrokerLoginFlowAlias":"nava-first-broker-login","organizationLinks":[{"organizationId":"someone-else"}]}`,
			wantErr: identity.ErrAlreadyExists,
		},
		{
			name: "idp linked to this and another org", orgs: oneOrg,
			idp:     `{"trustEmail":false,"firstBrokerLoginFlowAlias":"nava-first-broker-login","organizationLinks":[{"organizationId":"org-1"},{"organizationId":"o2"}]}`,
			wantErr: identity.ErrAlreadyExists,
		},
		{
			name: "idp not linked", orgs: oneOrg,
			idp:     `{"trustEmail":false,"firstBrokerLoginFlowAlias":"nava-first-broker-login","organizationLinks":[]}`,
			wantErr: identity.ErrAlreadyExists,
		},
		{
			name: "idp trusts email", orgs: oneOrg,
			idp:     `{"trustEmail":true,"firstBrokerLoginFlowAlias":"nava-first-broker-login","organizationLinks":[{"organizationId":"org-1"}]}`,
			wantErr: identity.ErrAlreadyExists,
		},
		{
			name: "idp with another login flow", orgs: oneOrg,
			idp:     `{"trustEmail":false,"firstBrokerLoginFlowAlias":"first broker login","organizationLinks":[{"organizationId":"org-1"}]}`,
			wantErr: identity.ErrAlreadyExists,
		},
		{name: "idp with a mapper", orgs: oneOrg, idp: goodIdP, mappers: `[{"id":"m"}]`, wantErr: identity.ErrAlreadyExists},
		{name: "another org with a similar alias is ignored", orgs: `[{"id":"org-9","alias":"` + findAlias + `-x"}]`},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			f := &findRealm{orgs: c.orgs, idp: c.idp, mappers: c.mappers}
			srv := f.server(t)
			d, _ := directoryFor(t, srv.URL, withOnboarding)
			org, found, err := d.FindTenantOrg(t.Context(), dirTenantA, findAlias)
			if !errors.Is(err, c.wantErr) || (c.wantErr == nil && err != nil) {
				t.Fatalf("err = %v, want %v", err, c.wantErr)
			}
			if org != c.wantOrg || found != c.wantFound {
				t.Errorf("org=%q found=%v, want %q %v", org, found, c.wantOrg, c.wantFound)
			}
			if f.posts.Load() != c.wantPosts {
				t.Errorf("mapper posts = %d, want %d", f.posts.Load(), c.wantPosts)
			}
			if f.orgDel.Load() != c.wantOrgDe {
				t.Errorf("org deleted = %v, want %v", f.orgDel.Load(), c.wantOrgDe)
			}
			// Only the lone-organization branch writes anything, and never to an IdP.
			if f.idpDel.Load() || f.otherDel.Load() || (!c.wantOrgDe && f.deletes.Load() != 0) {
				t.Errorf("unexpected deletes: %d (idp %v, other org %v)", f.deletes.Load(), f.idpDel.Load(), f.otherDel.Load())
			}
		})
	}
}

func TestFindTenantOrgRefusals(t *testing.T) {
	t.Parallel()
	d, _ := directoryFor(t, "http://127.0.0.1:1")
	if _, _, err := d.FindTenantOrg(t.Context(), dirTenantA, findAlias); !errors.Is(err, ErrOnboardingDisabled) {
		t.Errorf("without the onboarding account: %v", err)
	}
	srv := (&findRealm{orgs: `[]`}).server(t)
	d, _ = directoryFor(t, srv.URL, withOnboarding)
	if _, _, err := d.FindTenantOrg(t.Context(), dirTenantA, "Bad Alias"); !errors.Is(err, identity.ErrInvalidInput) {
		t.Errorf("bad alias: %v", err)
	}
}

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

// seqKC answers each "METHOD /path" from a list, in order; the last answer
// repeats. Unlisted routes are 404. It records every request it served.
type seqKC struct {
	mu     sync.Mutex
	routes map[string][]kcAnswer
	n      map[string]int
	seen   []string
}

func (s *seqKC) saw(route string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, r := range s.seen {
		if r == route {
			return true
		}
	}
	return false
}

func newSeqKC(t *testing.T, routes map[string][]kcAnswer) (*seqKC, string) {
	t.Helper()
	k := &seqKC{routes: routes, n: map[string]int{}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/protocol/openid-connect/token") {
			fmt.Fprint(w, `{"access_token":"tok","expires_in":300}`)
			return
		}
		key := r.Method + " " + strings.TrimPrefix(r.URL.Path, "/admin/realms/nava")
		k.mu.Lock()
		k.seen = append(k.seen, key)
		list, found := k.routes[key]
		var a kcAnswer
		if found {
			i := k.n[key]
			if i >= len(list) {
				i = len(list) - 1
			}
			k.n[key]++
			a = list[i]
		}
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

func one(a kcAnswer) []kcAnswer { return []kcAnswer{a} }

func onboardingDir(t *testing.T, url string, spec IdPSpec, specErr error) *Directory {
	t.Helper()
	d, _ := directoryFor(t, url, func(c *DirectoryConfig) {
		c.Onboarding = &ClientCredentials{ClientID: "nava-onboarding", ClientSecret: "s"}
		c.IdPs = func(context.Context, uuid.UUID) (IdPSpec, error) { return spec, specErr }
	})
	return d
}

var oidcSpec = IdPSpec{ProviderID: "oidc", Config: map[string]string{"clientId": "x"}} //nolint:gochecknoglobals // read-only fixture

func idpJSON(orgID string, trust bool, flow string) string {
	links := "[]"
	if orgID != "" {
		links = fmt.Sprintf(`[{"organizationId":%q}]`, orgID)
	}
	return fmt.Sprintf(`{"alias":"a","providerId":"oidc","trustEmail":%t,"firstBrokerLoginFlowAlias":%q,"organizationLinks":%s}`, trust, flow, links)
}

func mapperJSON() string {
	m := emailVerifiedMapper("a", "oidc")
	return fmt.Sprintf(`[{"id":"m1","name":%q,"identityProviderAlias":"a","identityProviderMapper":%q,"config":{"syncMode":"FORCE","claim":"email_verified","user.attribute":%q}}]`,
		m.Name, m.IdentityProviderMapper, EmailVerifiedAttribute)
}

func TestCreateTenantOrgPaths(t *testing.T) {
	t.Parallel()
	idpPath := "/identity-provider/instances/" + TenantIdPAlias(tA)
	orgLoc := "http://x/admin/realms/nava/organizations/org-1"
	happy := func() map[string][]kcAnswer {
		return map[string][]kcAnswer{
			"GET " + idpPath: {st(404), {status: 200, body: idpJSON("", false, FirstBrokerLoginFlow)},
				{status: 200, body: idpJSON("org-1", false, FirstBrokerLoginFlow)}},
			"POST /organizations":                          one(kcAnswer{status: 201, loc: orgLoc}),
			"POST /identity-provider/instances":            one(st(201)),
			"POST /organizations/org-1/identity-providers": one(st(204)),
			"GET " + idpPath + "/mappers":                  {ok(`[]`), ok(mapperJSON())},
			"POST " + idpPath + "/mappers":                 one(st(201)),
			"DELETE /organizations/org-1":                  one(st(204)),
			"DELETE " + idpPath:                            one(st(204)),
		}
	}
	run := func(t *testing.T, routes map[string][]kcAnswer, spec IdPSpec, specErr error, alias string, doms []string) (*seqKC, string, error) {
		t.Helper()
		kc, url := newSeqKC(t, routes)
		id, err := onboardingDir(t, url, spec, specErr).CreateTenantOrg(t.Context(), tA, alias, doms)
		return kc, id, err
	}

	if kc, id, err := run(t, happy(), oidcSpec, nil, "tenant-a", []string{"a.test"}); err != nil || id != "org-1" || kc.saw("DELETE /organizations/org-1") {
		t.Fatalf("happy: %q %v %v", id, err, kc.seen)
	}
	if _, _, err := run(t, happy(), oidcSpec, nil, "Bad Alias", nil); !errors.Is(err, identity.ErrInvalidInput) {
		t.Fatalf("alias: %v", err)
	}
	if _, _, err := run(t, happy(), oidcSpec, nil, "tenant-a", []string{"not a domain"}); !errors.Is(err, identity.ErrInvalidInput) {
		t.Fatalf("domain: %v", err)
	}
	if _, _, err := run(t, happy(), oidcSpec, errors.New("spec down"), "tenant-a", nil); err == nil || !strings.Contains(err.Error(), "idp spec") {
		t.Fatalf("spec err: %v", err)
	}
	if _, _, err := run(t, happy(), IdPSpec{ProviderID: "ldap"}, nil, "tenant-a", nil); !errors.Is(err, identity.ErrInvalidInput) {
		t.Fatalf("provider: %v", err)
	}
	// The platform owns the organization-binding keys.
	bad := IdPSpec{ProviderID: "oidc", Config: map[string]string{"kc.org.broker.redirect.mode.email-matches": "true"}}
	if kc, _, err := run(t, happy(), bad, nil, "tenant-a", nil); !errors.Is(err, identity.ErrInvalidInput) || kc.saw("POST /organizations") {
		t.Fatalf("policy key: %v", err)
	}
	// An IdP already under the derived alias is not ours: refuse, write nothing.
	taken := happy()
	taken["GET "+idpPath] = one(ok(idpJSON("org-other", false, FirstBrokerLoginFlow)))
	if kc, _, err := run(t, taken, oidcSpec, nil, "tenant-a", nil); !errors.Is(err, identity.ErrAlreadyExists) || kc.saw("POST /organizations") || kc.saw("DELETE "+idpPath) {
		t.Fatalf("taken: %v %v", err, kc.seen)
	}
	pre := happy()
	pre["GET "+idpPath] = one(st(500))
	if _, _, err := run(t, pre, oidcSpec, nil, "tenant-a", nil); err == nil || errors.Is(err, identity.ErrNotFound) {
		t.Fatalf("precheck error: %v", err)
	}
	dom := happy()
	dom["POST /organizations"] = one(kcAnswer{status: 400, body: `{"errorMessage":"Domain a.test is already linked to organization other"}`})
	if _, _, err := run(t, dom, oidcSpec, nil, "tenant-a", []string{"a.test"}); !errors.Is(err, ErrDomainLinked) {
		t.Fatalf("domain linked: %v", err)
	}
	other400 := happy()
	other400["POST /organizations"] = one(kcAnswer{status: 400, body: `{"errorMessage":"nope"}`})
	if _, _, err := run(t, other400, oidcSpec, nil, "tenant-a", nil); err == nil || errors.Is(err, ErrDomainLinked) {
		t.Fatalf("other 400: %v", err)
	}
	noLoc := happy()
	noLoc["POST /organizations"] = one(st(201))
	if _, _, err := run(t, noLoc, oidcSpec, nil, "tenant-a", nil); err == nil || !strings.Contains(err.Error(), "location") {
		t.Fatalf("no location: %v", err)
	}

	// Every failure after the organization exists rolls the organization back.
	type fail struct {
		name   string
		mutate func(map[string][]kcAnswer)
	}
	for _, f := range []fail{
		{"idp create fails", func(r map[string][]kcAnswer) { r["POST /identity-provider/instances"] = one(st(500)) }},
		{"idp re-read fails", func(r map[string][]kcAnswer) {
			r["GET "+idpPath] = []kcAnswer{st(404), st(500)}
		}},
		{"idp already linked", func(r map[string][]kcAnswer) {
			r["GET "+idpPath] = []kcAnswer{st(404), ok(idpJSON("org-other", false, FirstBrokerLoginFlow))}
		}},
		{"link call fails", func(r map[string][]kcAnswer) { r["POST /organizations/org-1/identity-providers"] = one(st(500)) }},
		{"verify read fails", func(r map[string][]kcAnswer) {
			r["GET "+idpPath] = []kcAnswer{st(404), ok(idpJSON("", false, FirstBrokerLoginFlow)), st(500)}
		}},
		{"trustEmail set", func(r map[string][]kcAnswer) {
			r["GET "+idpPath] = []kcAnswer{st(404), ok(idpJSON("", false, FirstBrokerLoginFlow)), ok(idpJSON("org-1", true, FirstBrokerLoginFlow))}
		}},
		{"wrong first-login flow", func(r map[string][]kcAnswer) {
			r["GET "+idpPath] = []kcAnswer{st(404), ok(idpJSON("", false, FirstBrokerLoginFlow)), ok(idpJSON("org-1", false, "browser"))}
		}},
		{"linked to a different org", func(r map[string][]kcAnswer) {
			r["GET "+idpPath] = []kcAnswer{st(404), ok(idpJSON("", false, FirstBrokerLoginFlow)), ok(idpJSON("org-x", false, FirstBrokerLoginFlow))}
		}},
		{"mapper list fails", func(r map[string][]kcAnswer) { r["GET "+idpPath+"/mappers"] = one(st(500)) }},
		{"mapper create fails", func(r map[string][]kcAnswer) { r["POST "+idpPath+"/mappers"] = one(st(500)) }},
		{"mapper re-list fails", func(r map[string][]kcAnswer) { r["GET "+idpPath+"/mappers"] = []kcAnswer{ok(`[]`), st(500)} }},
		{"unexpected extra mapper", func(r map[string][]kcAnswer) {
			r["GET "+idpPath+"/mappers"] = one(ok(`[{"name":"other","identityProviderMapper":"x","config":{}}]`))
		}},
	} {
		t.Run(f.name, func(t *testing.T) {
			t.Parallel()
			r := happy()
			f.mutate(r)
			kc, id, err := run(t, r, oidcSpec, nil, "tenant-a", nil)
			if err == nil || id != "" {
				t.Fatalf("want failure, got %q %v", id, err)
			}
			if !kc.saw("DELETE /organizations/org-1") {
				t.Fatalf("organization not rolled back: %v", kc.seen)
			}
		})
	}
}

func TestEmailVerifiedMapperShapes(t *testing.T) {
	t.Parallel()
	m := emailVerifiedMapper("a", "saml")
	if m.IdentityProviderMapper != "saml-user-attribute-idp-mapper" || m.Config["attribute.name"] != "email_verified" {
		t.Fatalf("saml mapper: %+v", m)
	}
	want := emailVerifiedMapper("a", "oidc")
	if !want.matches(want) {
		t.Fatal("a mapper must match itself")
	}
	other := want
	other.Config = map[string]string{"syncMode": "FORCE", "claim": "email_verified", "user.attribute": "other"}
	if other.matches(want) {
		t.Fatal("different attribute must not match")
	}
}

func TestFindTenantOrgPaths(t *testing.T) {
	t.Parallel()
	idpPath := "/identity-provider/instances/" + TenantIdPAlias(tA)
	good := func() map[string][]kcAnswer {
		return map[string][]kcAnswer{
			"GET /organizations":          one(ok(`[{"id":"org-1","alias":"tenant-a"},{"id":"zz","alias":"tenant-ab"}]`)),
			"GET " + idpPath:              one(ok(idpJSON("org-1", false, FirstBrokerLoginFlow))),
			"GET " + idpPath + "/mappers": one(ok(mapperJSON())),
			"DELETE /organizations/org-1": one(st(204)),
		}
	}
	run := func(t *testing.T, routes map[string][]kcAnswer, tenant uuid.UUID, alias string) (*seqKC, string, bool, error) {
		t.Helper()
		kc, url := newSeqKC(t, routes)
		id, found, err := onboardingDir(t, url, oidcSpec, nil).FindTenantOrg(t.Context(), tenant, alias)
		return kc, id, found, err
	}
	if _, id, found, err := run(t, good(), tA, "tenant-a"); err != nil || !found || id != "org-1" {
		t.Fatalf("found: %q %v %v", id, found, err)
	}
	if _, _, _, err := run(t, good(), uuid.Nil, "tenant-a"); !errors.Is(err, identity.ErrInvalidInput) {
		t.Fatalf("no tenant: %v", err)
	}
	if _, _, _, err := run(t, good(), tA, "BAD"); !errors.Is(err, identity.ErrInvalidInput) {
		t.Fatalf("alias: %v", err)
	}
	d, _ := directoryFor(t, "http://127.0.0.1:1")
	if _, _, err := d.FindTenantOrg(t.Context(), tA, "tenant-a"); !errors.Is(err, ErrOnboardingDisabled) {
		t.Fatalf("disabled: %v", err)
	}
	mut := func(f func(map[string][]kcAnswer)) map[string][]kcAnswer {
		r := good()
		f(r)
		return r
	}

	if _, _, _, err := run(t, mut(func(r map[string][]kcAnswer) { r["GET /organizations"] = one(st(500)) }), tA, "tenant-a"); err == nil {
		t.Fatal("org list error")
	}
	if _, _, _, err := run(t, mut(func(r map[string][]kcAnswer) { r["GET "+idpPath] = one(st(500)) }), tA, "tenant-a"); err == nil || errors.Is(err, identity.ErrNotFound) {
		t.Fatalf("idp read error: %v", err)
	}
	// Neither exists.
	if _, id, found, err := run(t, mut(func(r map[string][]kcAnswer) {
		r["GET /organizations"] = one(ok(`[]`))
		r["GET "+idpPath] = one(st(404))
	}), tA, "tenant-a"); err != nil || found || id != "" {
		t.Fatalf("neither: %v %v", found, err)
	}
	// An IdP without its organization is not ours to adopt.
	if _, _, found, err := run(t, mut(func(r map[string][]kcAnswer) { r["GET /organizations"] = one(ok(`[]`)) }), tA, "tenant-a"); !errors.Is(err, identity.ErrAlreadyExists) || found {
		t.Fatalf("orphan idp: %v", err)
	}
	// Only the organization exists: it is deleted.
	onlyOrg := func(r map[string][]kcAnswer) { r["GET "+idpPath] = one(st(404)) }
	if kc, _, found, err := run(t, mut(onlyOrg), tA, "tenant-a"); err != nil || found || !kc.saw("DELETE /organizations/org-1") {
		t.Fatalf("only org: %v %v", err, found)
	}
	if _, _, _, err := run(t, mut(func(r map[string][]kcAnswer) {
		onlyOrg(r)
		r["DELETE /organizations/org-1"] = one(st(500))
	}), tA, "tenant-a"); err == nil {
		t.Fatal("delete error")
	}
	// Policy breaches: nothing is changed.
	for name, idp := range map[string]string{
		"linked elsewhere": idpJSON("org-other", false, FirstBrokerLoginFlow),
		"not linked":       idpJSON("", false, FirstBrokerLoginFlow),
		"trusts email":     idpJSON("org-1", true, FirstBrokerLoginFlow),
		"wrong flow":       idpJSON("org-1", false, "browser"),
	} {
		kc, _, found, err := run(t, mut(func(r map[string][]kcAnswer) { r["GET "+idpPath] = one(ok(idp)) }), tA, "tenant-a")
		if !errors.Is(err, identity.ErrAlreadyExists) || found || kc.saw("DELETE /organizations/org-1") {
			t.Fatalf("%s: %v %v", name, err, found)
		}
	}
	if _, _, _, err := run(t, mut(func(r map[string][]kcAnswer) { r["GET "+idpPath+"/mappers"] = one(st(500)) }), tA, "tenant-a"); err == nil {
		t.Fatal("mapper error")
	}
}

type placeResult struct {
	placed   bool
	consumed int
	err      error
	kc       *routeKC
}

func TestPlaceInviteePaths(t *testing.T) {
	t.Parallel()
	email := "erin@tenant-a.test"
	adminsRoute := "GET /group-by-path/t/" + tA.String() + "/admins"
	good := func() map[string]kcAnswer {
		return map[string]kcAnswer{
			"GET /users": ok(`[{"id":"u1","email":"erin@tenant-a.test","enabled":true}]`),
			"GET /users/u1": ok(fmt.Sprintf(`{"id":"u1","attributes":{%q:["true"]}}`,
				EmailVerifiedAttribute)),
			"GET /users/u1/federated-identity":            ok(`[{"identityProvider":"idp-a","userId":"1"}]`),
			"GET /users/u1/groups":                        ok(`[]`),
			"GET /organizations/members/u1/organizations": ok(`[{"id":"org-1"}]`),
			"GET /group-by-path/t/" + tA.String():         ok(rootAJS),
			adminsRoute:                                   ok(fmt.Sprintf(`{"id":"g-admins","path":%q}`, rootA+"/admins")),
			"PUT /users/u1/groups/g-root":                 st(204),
			"PUT /users/u1/groups/g-admins":               st(204),
		}
	}
	run := func(t *testing.T, routes map[string]kcAnswer, bound, orgs []string, consume func() (bool, error), e string) placeResult {
		t.Helper()
		kc, url := newRouteKC(t, routes)
		d, _ := directoryFor(t, url)
		res := placeResult{kc: kc}
		res.placed, res.err = d.PlaceInvitee(t.Context(), tA, e, bound, orgs, func(context.Context) (bool, error) {
			res.consumed++
			return consume()
		})
		return res
	}
	yes := func() (bool, error) { return true, nil }
	b, o := []string{"idp-a"}, []string{"org-1"}

	if r := run(t, good(), b, o, yes, email); r.err != nil || !r.placed || r.consumed != 1 || !r.kc.saw("PUT /users/u1/groups/g-root") {
		t.Fatalf("happy: %+v", r)
	}
	// Refusals: (false, nil), the invite is never consumed, nothing is written.
	refusals := map[string]func(map[string]kcAnswer){
		"user not found": func(m map[string]kcAnswer) { m["GET /users"] = ok(`[]`) },
		"user disabled": func(m map[string]kcAnswer) {
			m["GET /users"] = ok(`[{"id":"u1","email":"erin@tenant-a.test","enabled":false}]`)
		},
		"email not verified": func(m map[string]kcAnswer) { m["GET /users/u1"] = ok(`{"id":"u1","attributes":{}}`) },
		"email_verified not exactly true": func(m map[string]kcAnswer) {
			m["GET /users/u1"] = ok(fmt.Sprintf(`{"id":"u1","attributes":{%q:["TRUE"]}}`, EmailVerifiedAttribute))
		},
		"local user": func(m map[string]kcAnswer) { m["GET /users/u1/federated-identity"] = ok(`[]`) },
		"unbound idp": func(m map[string]kcAnswer) {
			m["GET /users/u1/federated-identity"] = ok(`[{"identityProvider":"idp-evil","userId":"1"}]`)
		},
		"member of another tenant": func(m map[string]kcAnswer) {
			m["GET /users/u1/groups"] = ok(fmt.Sprintf(`[{"id":"x","path":%q}]`, "/t/"+tB.String()))
		},
		"no organization": func(m map[string]kcAnswer) { m["GET /organizations/members/u1/organizations"] = ok(`[]`) },
		"foreign organization": func(m map[string]kcAnswer) {
			m["GET /organizations/members/u1/organizations"] = ok(`[{"id":"org-1"},{"id":"org-other"}]`)
		},
	}
	for name, mut := range refusals {
		m := good()
		mut(m)
		if r := run(t, m, b, o, yes, email); r.err != nil || r.placed || r.consumed != 0 || r.kc.saw("PUT /users/u1/groups/g-root") || r.kc.saw("PUT /users/u1/groups/g-admins") {
			t.Fatalf("%s: %+v", name, r)
		}
	}
	if r := run(t, good(), nil, o, yes, email); r.placed || r.err != nil {
		t.Fatalf("no bound idps: %+v", r)
	}
	if r := run(t, good(), b, nil, yes, email); r.placed || r.err != nil {
		t.Fatalf("no org ids: %+v", r)
	}
	if r := run(t, good(), b, o, yes, "bad"); !errors.Is(r.err, identity.ErrInvalidInput) {
		t.Fatalf("bad email: %+v", r)
	}
	{
		_, url := newRouteKC(t, good())
		d, _ := directoryFor(t, url)
		if _, err := d.PlaceInvitee(t.Context(), uuid.Nil, email, b, o, nil); !errors.Is(err, identity.ErrInvalidInput) {
			t.Fatalf("no tenant: %v", err)
		}
	}
	// An invite that is gone or expired places nobody.
	if r := run(t, good(), b, o, func() (bool, error) { return false, nil }, email); r.placed || r.err != nil || r.kc.saw("PUT /users/u1/groups/g-root") {
		t.Fatalf("no invite: %+v", r)
	}
	if r := run(t, good(), b, o, func() (bool, error) { return false, errors.New("db") }, email); r.err == nil || r.placed || r.kc.saw("PUT /users/u1/groups/g-root") {
		t.Fatalf("consume error: %+v", r)
	}
	// Already in the tenant: root write skipped, admin group still finished.
	already := good()
	already["GET /users/u1/groups"] = ok(fmt.Sprintf(`[{"id":"g","path":%q}]`, rootA+"/x"))
	if r := run(t, already, b, o, yes, email); !r.placed || r.kc.saw("PUT /users/u1/groups/g-root") || !r.kc.saw("PUT /users/u1/groups/g-admins") {
		t.Fatalf("already in tenant: %+v", r)
	}
	// Backend failures are errors, never a silent placement.
	for _, route := range []string{"GET /users", "GET /users/u1", "GET /users/u1/federated-identity", "GET /users/u1/groups",
		"GET /organizations/members/u1/organizations", "GET /group-by-path/t/" + tA.String(),
		"PUT /users/u1/groups/g-root", "PUT /users/u1/groups/g-admins"} {
		m := good()
		m[route] = st(500)
		if r := run(t, m, b, o, yes, email); r.err == nil || r.placed {
			t.Fatalf("%s failing: %+v", route, r)
		}
	}
}

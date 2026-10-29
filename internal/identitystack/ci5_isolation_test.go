//go:build identitystack && ci5

package identitystack

// CI-5, the adversarial identity suite (ADR-001 Annex B CI-5, PLAN-001 T19).
// Every case runs against the compose Keycloak and the production adapter,
// with tenant bindings and principals in a real platform database filled by
// one real group sync (ci5_env_test.go). The cases are registered in the T15
// contract harness with Require "CI-5", so the run fails if a catalogue entry
// has no case, and tools/ci/ci5report lists each one with its result.
//
// The attacker holds a valid login for one tenant, administers one tenant, or
// controls one tenant's external IdP. Every case but CI-5.14 must leave them
// without a principal of another tenant; CI-5.14 is the positive control.
//
// The ci5 build tag keeps TestCI5, which needs the platform database
// scripts/ci/identity-attacks.sh starts, out of make stack-test-identity, so
// that target runs the rest of the package under suiteran with no -skip
// (TICK-283). make identity-attacks builds with identitystack,ci5.

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/identity/contracttest"
	"github.com/Navaladi2019/TicketingSystem/internal/identity/pgstore"
)

func TestCI5(t *testing.T) {
	e := newCI5Env(t)
	rootA, rootB := "/t/"+tenantA.String(), "/t/"+tenantB.String()

	// CI-5.01: a user of tenant A's group subtree, so a principal exists,
	// but in no organization.
	_, noOrgUser, noOrgPass := ci5User(t, "ci5-noorg", nil, []string{rootA})
	noOrgTok := passwordToken(t, "nava-web", noOrgUser, noOrgPass)
	if ids := orgIDs(t, noOrgTok); len(ids) != 0 {
		t.Fatalf("setup: the no-organization user's token names %v", ids)
	}

	// CI-5.02: a member of both organizations, asking for every one.
	_, twoUser, twoPass := ci5User(t, "ci5-two", []string{e.orgA, e.orgB}, []string{rootA})
	twoTok := mustToken(t, url.Values{
		"grant_type": {"password"}, "client_id": {"nava-web"}, "username": {twoUser}, "password": {twoPass},
		"scope": {"openid organization:*"},
	})
	if ids := orgIDs(t, twoTok); len(ids) != 2 {
		t.Fatalf("setup: the two-organization token names %v, want both organizations", ids)
	}

	// CI-5.05: alice's ID token, and her access token as its control.
	code, aliceOut := tokenForm(t, "nava", url.Values{
		"grant_type": {"password"}, "client_id": {"nava-web"}, "username": {"alice@tenant-a.test"},
		"password": {"dev-only-alice"}, "scope": {"openid"},
	})
	aliceIDTok, _ := aliceOut["id_token"].(string)
	aliceTok, _ := aliceOut["access_token"].(string)
	if code != http.StatusOK || aliceIDTok == "" || aliceTok == "" {
		t.Fatalf("setup: alice's tokens: %d %v", code, aliceOut)
	}

	// CI-5.09: dave, brokered through tenant B's IdP, which tenant B's
	// binding does not list.
	daveTok := brokeredToken(t, "idp-tenant-b", "dave@tenant-b.test", "dev-only-dave")
	if c := claims(t, daveTok); c["idp"] != "idp-tenant-b" || !slices.Equal(orgIDs(t, daveTok), []string{e.orgB}) {
		t.Fatalf("setup: dave's token idp=%v organization=%v", c["idp"], c["organization"])
	}
	// Invited, so after the sync dave is a principal of tenant B and only
	// the idp check can refuse him.
	daveKC, _ := claims(t, daveTok)["sub"].(string)
	if err := e.dir.InviteUser(t.Context(), tenantB, "dave@tenant-b.test"); err != nil {
		t.Fatalf("setup: invite dave into tenant B: %v", err)
	}
	t.Cleanup(func() { adminRealm(t, "nava", http.MethodDelete, "/users/"+daveKC+"/groups/"+kcGroupID(t, rootB), nil) })

	// CI-5.14: carol, brokered through tenant A's IdP, then invited.
	carolTok := brokeredToken(t, "idp-tenant-a", "carol@tenant-a.test", "dev-only-carol")
	carolKC, _ := claims(t, carolTok)["sub"].(string)
	if err := e.dir.InviteUser(t.Context(), tenantA, "carol@tenant-a.test"); err != nil {
		t.Fatalf("setup: invite carol into tenant A: %v", err)
	}
	t.Cleanup(func() {
		adminRealm(t, "nava", http.MethodDelete, "/users/"+carolKC+"/groups/"+kcGroupID(t, rootA), nil)
	})

	// CI-5.13: tenant B's admin: tenant B's organization and root group, and
	// the realm role tenant A's admin holds.
	adminB, adminBUser, adminBPass := ci5User(t, "ci5-admin-b", []string{e.orgB}, []string{rootB})
	_, rb := adminRealm(t, "nava", http.MethodGet, "/roles/tenant-admin", nil)
	var tenantAdmin map[string]any
	_ = json.Unmarshal(rb, &tenantAdmin)
	if code, b := adminRealm(t, "nava", http.MethodPost, "/users/"+adminB+"/role-mappings/realm", []map[string]any{tenantAdmin}); code != http.StatusNoContent {
		t.Fatalf("setup: grant tenant-admin to tenant B's admin: %d %s", code, b)
	}

	// The one group sync every resolution below reads.
	e.sync(t)
	e.principalID(t, tenantB, daveKC)

	contracttest.Suite{
		Require:  []string{"CI-5"},
		Resolver: e.resolver,
		// No operator allow-list: only the garbage cases run, all refused.
		Operator: e.resolver,
		ResolverCases: []contracttest.ResolverCase{
			{Case: contracttest.CI5ZeroOrganizations, Name: "user in tenant A's groups but no organization", Token: noOrgTok, WantErr: identity.ErrTenantUnresolved},
			{Case: contracttest.CI5TwoOrganizations, Name: "organization:* for a member of tenant-a and tenant-b", Token: twoTok, WantErr: identity.ErrTenantUnresolved},
			{Case: contracttest.CI5UnknownIssuer, Name: "a Keycloak-signed token of realm master", Token: masterToken(t), WantErr: identity.ErrInvalidToken},
			{Case: contracttest.CI5IDTokenAsAccessToken, Name: "alice's ID token", Token: aliceIDTok, WantErr: identity.ErrInvalidToken},
			{Case: contracttest.CI5IdPNotBound, Name: "dave brokered through idp-tenant-b, which tenant B's binding does not list", Token: daveTok, WantErr: identity.ErrTenantUnresolved},
		},
		Directory: e.dir,
		DirectoryCases: []contracttest.DirectoryCase{
			{Case: contracttest.CI5ForeignIdPAtInvite, Name: "tenant A invites a user federated to tenant B's IdP", Run: ci5ForeignIdPAtInvite},
		},
		Scenarios: []contracttest.ScenarioCase{
			{Case: contracttest.CI5ForeignRealmKey, Name: "a test-held realm key signs a token with iss of realm nava", Run: func(t *testing.T) { ci5ForeignRealmKey(t, e, aliceTok) }},
			{Case: contracttest.CI5OrgPickerTamper, Name: "kc.org names tenant-b on the organization picker", Run: func(t *testing.T) { ci5OrgPickerTamper(t, e) }},
			{Case: contracttest.CI5BrokerExistingEmail, Name: "tenant B's IdP asserts alice's email and username", Run: func(t *testing.T) { ci5BrokerExistingEmail(t, e) }},
			{Case: contracttest.CI5BrokerOutOfDomain, Name: "tenant A's IdP asserts an address outside tenant A's domains", Run: func(t *testing.T) { ci5BrokerOutOfDomain(t, e) }},
			{Case: contracttest.CI5ForeignIdPAtLogin, Name: "a tenant A user linked to tenant B's IdP, after one group sync", Run: func(t *testing.T) { ci5ForeignIdPAtLogin(t, e) }},
			{Case: contracttest.CI5DuplicateBinding, Name: "tenant B binds tenant A's organization", Run: func(t *testing.T) { ci5DuplicateBinding(t, e, aliceTok) }},
			{Case: contracttest.CI5OtherTenantAdmin, Name: "tenant B's admin reaches for tenant A", Run: func(t *testing.T) { ci5OtherTenantAdmin(t, e, adminB, adminBUser, adminBPass) }},
			{Case: contracttest.CI5InDomainBrokeredAllow, Name: "carol brokered through idp-tenant-a", Run: func(t *testing.T) { ci5InDomainBrokered(t, e, carolTok, carolKC) }},
		},
	}.Run(t)
}

// wantNotTenant fails if tok resolves to tenant.
func wantNotTenant(t *testing.T, e *ci5Env, tok string, tenant uuid.UUID) {
	t.Helper()
	if p, err := e.resolver.Resolve(t.Context(), tok); err == nil && p.TenantID == tenant {
		t.Errorf("the token resolved to tenant %s: %+v", tenant, p)
	}
}

// CI-5.04. Realm ci5-x's key is one the test holds; both realms are trusted.
// The control (iss=ci5-x) shows the key really verifies, so the attack fails
// for the key and the issuer, not for a broken signature.
func ci5ForeignRealmKey(t *testing.T, e *ci5Env, aliceTok string) {
	iss, key, kid := foreignRealm(t, baseURL(), newClient(t), masterToken(t))
	both := ci5Adapter(t, []string{e.iss, iss}, pgstore.New(e.app))

	control := freshClaims(t, aliceTok)
	control["iss"] = iss
	_, err := both.Resolve(t.Context(), signRS256(t, key, kid, control))
	if !errors.Is(err, identity.ErrTenantUnresolved) {
		t.Fatalf("control: realm ci5-x's own token = %v, want ErrTenantUnresolved (signature good, no binding)", err)
	}
	if _, err := e.resolver.Resolve(t.Context(), signRS256(t, key, kid, control)); !errors.Is(err, identity.ErrInvalidToken) {
		t.Errorf("the production adapter (realm nava only) accepted the issuer ci5-x: %v", err)
	}

	attack := signRS256(t, key, kid, freshClaims(t, aliceTok)) // alice's claims, iss of realm nava
	for name, a := range map[string]identity.IdentityResolver{"both realms trusted": both, "production adapter": e.resolver} {
		p, err := a.Resolve(t.Context(), attack)
		if !errors.Is(err, identity.ErrInvalidToken) {
			t.Errorf("%s: realm ci5-x's key with iss=nava = %+v, %v; want ErrInvalidToken", name, p, err)
		}
	}
}

// CI-5.06 (TA-B1). A member of tenant-a and a decoy organization is shown the
// picker and posts kc.org=tenant-b, an organization they are not a member of.
// Keycloak 26.0.0-26.1.2 put tenant-b in the token
// (TestKC2612FixtureOrgPickerTamperSucceeds proves the case can see that).
func ci5OrgPickerTamper(t *testing.T, e *ci5Env) {
	decoy := ci5Org(t)
	_, user, pass := ci5User(t, "ci5-picker", []string{e.orgA, decoy}, []string{"/t/" + tenantA.String()})
	res := orgPicker(t, newClient(t), baseURL(), "nava", "nava-web", user, pass, "tenant-b")
	if !res.Shown {
		t.Fatalf("Keycloak showed no organization picker, so the tamper was never tried: %.600s", res.Page)
	}
	if slices.Contains(res.Choices, "tenant-b") {
		t.Fatalf("the picker offered tenant-b to a non-member: %v", res.Choices)
	}
	if res.Token == "" {
		t.Logf("picker offered %v; kc.org=tenant-b was refused with no token", res.Choices)
		return
	}
	ids := orgIDs(t, res.Token)
	if org := fmt.Sprint(claims(t, res.Token)["organization"]); slices.Contains(ids, e.orgB) || strings.Contains(org, "tenant-b") {
		t.Errorf("the tampered login got a token naming tenant-b: %s", org)
	}
	t.Logf("picker offered %v; kc.org=tenant-b gave a token for %s only", res.Choices, describeOrgs(ids))
	if code, b := adminRealm(t, "nava", http.MethodGet, "/organizations/"+e.orgB+"/members?search="+url.QueryEscape(user), nil); code == http.StatusOK && strings.Contains(string(b), user) {
		t.Errorf("the tamper made %s a member of tenant-b", user)
	}
	wantNotTenant(t, e, res.Token, tenantB)
}

// CI-5.07. Tenant B's IdP asserts alice's email and username. A.1 forbids
// linking by email: no token, and alice gains no federated identity.
func ci5BrokerExistingEmail(t *testing.T, e *ci5Env) {
	seedIdPUser(t, "idp-tenant-b", "alice@tenant-a.test", "alice@tenant-a.test", "dev-only-mallory")
	tok, page := brokerAttempt(t, "idp-tenant-b", "alice@tenant-a.test", "dev-only-mallory")
	if tok != "" {
		c := claims(t, tok)
		wantNotTenant(t, e, tok, tenantA)
		t.Errorf("tenant B's IdP colliding with alice got a token: sub=%v organization=%v", c["sub"], c["organization"])
	} else if !strings.Contains(string(page), "already exists") {
		// The refusal must come from first-broker-login (create-if-unique),
		// not from a login that never reached Keycloak's broker.
		t.Errorf("no token, but the login did not reach first-broker-login's duplicate check: %.600s", page)
	}
	alice := userID(t, serviceToken(t), "alice@tenant-a.test")
	if code, b := admin(t, serviceToken(t), http.MethodGet, "/users/"+alice+"/federated-identity", nil); code != http.StatusOK || strings.TrimSpace(string(b)) != "[]" {
		t.Errorf("alice's federated identities after the attempt: %d %s, want []", code, b)
	}
}

// CI-5.08. Tenant A's own IdP asserts an address in no domain of tenant A.
// The user may exist in the realm, but gets no organization and resolves to
// no tenant, even after a group sync.
func ci5BrokerOutOfDomain(t *testing.T, e *ci5Env) {
	user := "oscar-" + uuid.NewString()[:8]
	seedIdPUser(t, "idp-tenant-a", user, user+"@elsewhere.invalid", "dev-only-oscar")
	tok, page := brokerAttempt(t, "idp-tenant-a", user, "dev-only-oscar")
	// The address is unique, so first-broker-login creates the user in realm
	// nava: proof the login reached it, whatever it then decided.
	if kcUserByEmail(t, user+"@elsewhere.invalid") == "" {
		t.Fatalf("the brokered login never reached first-broker-login (no user created): %.600s", page)
	}
	if tok == "" {
		t.Log("first-broker-login created the user but issued no token")
		return
	}
	t.Log("the brokered login got a token; it must name no organization and resolve to no tenant")
	if ids := orgIDs(t, tok); len(ids) != 0 {
		t.Errorf("an out-of-domain brokered user got organization(s) %s", describeOrgs(ids))
	}
	e.sync(t)
	if p, err := e.resolver.Resolve(t.Context(), tok); !errors.Is(err, identity.ErrTenantUnresolved) {
		t.Errorf("out-of-domain brokered user resolved: %+v, %v; want ErrTenantUnresolved", p, err)
	}
}

// CI-5.10. A tenant A user, resolved before (the control), is then linked to
// tenant B's IdP. After one group sync records the link, the resolver refuses
// the old token and a new one.
func ci5ForeignIdPAtLogin(t *testing.T, e *ci5Env) {
	id, user, pass := ci5User(t, "ci5-gina", []string{e.orgA}, []string{"/t/" + tenantA.String()})
	before := passwordToken(t, "nava-web", user, pass)
	e.sync(t)
	if p, err := e.resolver.Resolve(t.Context(), before); err != nil || p.TenantID != tenantA {
		t.Fatalf("control: the unlinked user = %+v, %v; want tenant A", p, err)
	}
	extID := "ext-" + uuid.NewString()
	if code, b := adminRealm(t, "nava", http.MethodPost, "/users/"+id+"/federated-identity/idp-tenant-b", map[string]any{
		"identityProvider": "idp-tenant-b", "userId": extID, "userName": user,
	}); code != http.StatusNoContent && code != http.StatusCreated {
		t.Fatalf("link the user to idp-tenant-b: %d %s", code, b)
	}
	e.sync(t)
	var links string
	if err := e.super.QueryRow(t.Context(), "SELECT federated_links::text FROM principals WHERE tenant_id = $1 AND keycloak_user_id = $2",
		tenantA, id).Scan(&links); err != nil || !strings.Contains(links, "idp-tenant-b") {
		t.Fatalf("group sync did not mark the foreign link: %q %v", links, err)
	}
	if strings.Contains(links, extID) {
		t.Errorf("tenant A's row carries tenant B's external user id: %q", links)
	}
	for name, tok := range map[string]string{"token from before the link": before, "new token": passwordToken(t, "nava-web", user, pass)} {
		if p, err := e.resolver.Resolve(t.Context(), tok); !errors.Is(err, identity.ErrTenantUnresolved) {
			t.Errorf("%s: %+v, %v; want ErrTenantUnresolved", name, p, err)
		}
	}
}

// CI-5.11. A user created by a brokered login through tenant B's IdP is
// invited into tenant A: refused, and not placed in tenant A's subtree.
func ci5ForeignIdPAtInvite(t *testing.T, d identity.DirectoryAdmin) {
	email := "frank-" + uuid.NewString()[:8] + "@tenant-b.test"
	username := "frank-" + uuid.NewString()[:8]
	seedIdPUser(t, "idp-tenant-b", username, email, "dev-only-frank")
	deleteUserAfter(t, email)
	_, page := brokerAttempt(t, "idp-tenant-b", username, "dev-only-frank")
	uid := kcUserByEmail(t, email)
	if uid == "" {
		t.Fatalf("setup: the brokered user was not created: %.600s", page)
	}
	if err := d.InviteUser(t.Context(), tenantA, email); !errors.Is(err, identity.ErrInviteRefused) {
		t.Errorf("InviteUser(tenant A) of a user federated to idp-tenant-b = %v, want ErrInviteRefused", err)
	}
	if userInGroup(t, uid, kcGroupID(t, "/t/"+tenantA.String())) {
		t.Error("the user was placed in tenant A's group")
	}
}

// CI-5.12. (issuer, organization) is unique across tenants, for every role,
// the superuser included; tenant A still resolves to tenant A afterwards.
func ci5DuplicateBinding(t *testing.T, e *ci5Env, aliceTok string) {
	// Only the tenant B insert proves cross-tenant uniqueness: the primary key
	// holds tenant_id, so the tenant A insert hits it, never issuer_org_key.
	for _, c := range []struct {
		name       string
		tenant     uuid.UUID
		constraint string
	}{
		{"for tenant B", tenantB, "tenant_identity_binding_issuer_org_key"},
		{"for tenant A again", tenantA, "tenant_identity_binding_pkey"},
	} {
		_, err := e.super.Exec(t.Context(),
			"INSERT INTO tenant_identity_binding (tenant_id, issuer, org_id) VALUES ($1, $2, $3)", c.tenant, e.iss, e.orgA)
		var pg *pgconn.PgError
		if !errors.As(err, &pg) || pg.Code != "23505" || pg.ConstraintName != c.constraint {
			t.Errorf("%s: a second binding of tenant-a's organization = %v, want unique violation 23505 on %s", c.name, err, c.constraint)
		}
	}
	if p, err := e.resolver.Resolve(t.Context(), aliceTok); err != nil || p.TenantID != tenantA {
		t.Errorf("alice after the attempts = %+v, %v; want tenant A", p, err)
	}
}

// CI-5.13. Tenant B's admin holds every right tenant A's admin holds, in B.
// They ask for tenant-a's organization scope, exchange their token, and use
// the admin API to join tenant A. None of it yields a token that resolves to
// tenant A, and they end up in no part of tenant A.
func ci5OtherTenantAdmin(t *testing.T, e *ci5Env, id, user, pass string) {
	own := passwordToken(t, "nava-web", user, pass)
	if p, err := e.resolver.Resolve(t.Context(), own); err != nil || p.TenantID != tenantB {
		t.Fatalf("control: tenant B's admin = %+v, %v; want tenant B", p, err)
	}

	// Ask for tenant A's organization.
	code, out := tokenForm(t, "nava", url.Values{
		"grant_type": {"password"}, "client_id": {"nava-web"}, "username": {user}, "password": {pass},
		"scope": {"openid organization:tenant-a"},
	})
	if tok, _ := out["access_token"].(string); code == http.StatusOK && tok != "" {
		if slices.Contains(orgIDs(t, tok), e.orgA) {
			t.Errorf("scope organization:tenant-a put tenant-a in tenant B's admin's token")
		}
		wantNotTenant(t, e, tok, tenantA)
	}

	// Exchange the token, asking for tenant A's organization.
	code, out = tokenForm(t, "nava", url.Values{
		"grant_type": {"urn:ietf:params:oauth:grant-type:token-exchange"}, "client_id": {"nava-web"},
		"subject_token": {own}, "subject_token_type": {"urn:ietf:params:oauth:token-type:access_token"},
		"scope": {"openid organization:tenant-a"},
	})
	if tok, _ := out["access_token"].(string); code == http.StatusOK && tok != "" {
		wantNotTenant(t, e, tok, tenantA)
	}

	// Join tenant A through the admin API with their own token.
	rootA := kcGroupID(t, "/t/"+tenantA.String())
	for _, call := range []struct{ method, path string }{
		{http.MethodPost, "/organizations/" + e.orgA + "/members"},
		{http.MethodPut, "/users/" + id + "/groups/" + rootA},
		{http.MethodGet, "/organizations/" + e.orgA + "/members"},
		{http.MethodGet, "/groups/" + rootA + "/members"},
	} {
		var body any
		if call.method == http.MethodPost {
			body = id
		}
		if code, b := admin(t, own, call.method, call.path, body); code < 400 {
			t.Errorf("tenant B's admin: %s %s = %d %.200s, want 401 or 403", call.method, call.path, code, b)
		}
	}
	if userInGroup(t, id, rootA) {
		t.Error("tenant B's admin is in tenant A's group")
	}
	if _, b := adminRealm(t, "nava", http.MethodGet, "/organizations/"+e.orgA+"/members?search="+url.QueryEscape(user), nil); strings.Contains(string(b), id) {
		t.Error("tenant B's admin is a member of tenant-a")
	}

	// After all of it, and a sync, a fresh token still resolves to B only.
	e.sync(t)
	if p, err := e.resolver.Resolve(t.Context(), passwordToken(t, "nava-web", user, pass)); err != nil || p.TenantID != tenantB {
		t.Errorf("tenant B's admin after the attempts = %+v, %v; want tenant B", p, err)
	}
}

// CI-5.14, the positive control: an in-domain brokered user gets tenant-a's
// organization and idp in the token, and, once invited and synced, resolves
// to tenant A with that idp.
func ci5InDomainBrokered(t *testing.T, e *ci5Env, tok, kcUser string) {
	c := claims(t, tok)
	if ids := orgIDs(t, tok); !slices.Equal(ids, []string{e.orgA}) {
		t.Errorf("organization = %s, want tenant-a only", describeOrgs(ids))
	}
	if c["idp"] != "idp-tenant-a" {
		t.Errorf("idp = %v, want idp-tenant-a", c["idp"])
	}
	p, err := e.resolver.Resolve(t.Context(), tok)
	if err != nil {
		t.Fatalf("Resolve = %v, want tenant A", err)
	}
	if want := e.principalID(t, tenantA, kcUser); p.TenantID != tenantA || p.PrincipalID != want || p.IdPAlias != "idp-tenant-a" {
		t.Errorf("principal = %+v, want tenant %s, principal %s, idp idp-tenant-a", p, tenantA, want)
	}
}

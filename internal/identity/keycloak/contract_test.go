package keycloak

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/identity/contracttest"
)

// TestContract runs the identity port's contract suite against the adapter,
// with T17's tenant-resolution cases. T18 adds the DirectoryAdmin. T19 tags
// the CI-5 token shapes a fake IdP can mint; the cases only the real IdP or
// database can show (brokered logins, the organization picker, bindings) run
// against the compose Keycloak in internal/identitystack (TestCI5), which
// requires the whole catalogue.
func TestContract(t *testing.T) {
	t.Parallel()
	// Realm x is trusted too, so the foreign-key case is refused for the key,
	// not the issuer; realm evil is served but not trusted.
	h := newHarness(t, []string{realmNava, "x"}, "evil")
	bindDefault(h)
	// Subjects for the TICK-69 carry-forward cases.
	h.store.addPrincipal(tenantA, "kc-links-null", identity.PrincipalRecord{ID: uuid.New(), Enabled: true, LinksKnown: false})
	h.store.addPrincipal(tenantA, "kc-foreign-link", identity.PrincipalRecord{ID: uuid.New(), Enabled: true, LinksKnown: true, FederatedIdPs: []string{"idp-b"}})
	s := h.idp.signer(realmNava)
	now := h.clock.now()
	tenantWith := func(key signer, edit func(claims)) string {
		c := h.idp.tenantClaims(realmNava, now)
		edit(c)
		return sign(t, key, c)
	}
	tenant := func(edit func(claims)) string { return tenantWith(s, edit) }
	operator := func(edit func(claims)) string {
		c := h.idp.operatorClaims(realmNava, now)
		edit(c)
		return sign(t, s, c)
	}

	contracttest.Suite{
		Resolver: h.a,
		ResolverCases: []contracttest.ResolverCase{
			{Name: "bound tenant", Token: tenant(func(claims) {}),
				Want: &identity.Principal{TenantID: tenantA, PrincipalID: principalA, Groups: []uuid.UUID{groupA}}},
			{Name: "no organization", Case: contracttest.CI5ZeroOrganizations,
				Token: tenant(func(c claims) { delete(c, "organization") }), WantErr: identity.ErrTenantUnresolved},
			{Name: "two organizations", Case: contracttest.CI5TwoOrganizations, Token: tenant(func(c claims) {
				c["organization"] = map[string]any{"a": map[string]any{"id": defaultOrg}, "b": map[string]any{"id": "org-b"}}
			}), WantErr: identity.ErrTenantUnresolved},
			{Name: "unknown issuer", Case: contracttest.CI5UnknownIssuer,
				Token: tenantWith(h.idp.signer("evil"), func(c claims) { c["iss"] = h.idp.issuer("evil") }), WantErr: identity.ErrInvalidToken},
			{Name: "realm x key with iss of realm nava", Case: contracttest.CI5ForeignRealmKey,
				Token: tenantWith(h.idp.signer("x"), func(claims) {}), WantErr: identity.ErrInvalidToken},
			{Name: "ID token", Case: contracttest.CI5IDTokenAsAccessToken, Token: tenant(func(c claims) {
				c["typ"], c["aud"] = "ID", []string{"nava-web", DefaultAudience}
			}), WantErr: identity.ErrInvalidToken},
			{Name: "unbound organization", Token: tenant(func(c claims) {
				c["organization"] = map[string]any{"x": map[string]any{"id": "org-x"}}
			}), WantErr: identity.ErrTenantUnresolved},
			{Name: "unbound idp", Case: contracttest.CI5IdPNotBound,
				Token: tenant(func(c claims) { c["idp"] = "idp-evil" }), WantErr: identity.ErrTenantUnresolved},
			{Name: "federated link to a foreign IdP", Case: contracttest.CI5ForeignIdPAtLogin,
				Token: tenant(func(c claims) { c["sub"] = "kc-foreign-link" }), WantErr: identity.ErrTenantUnresolved},
			{Name: "federated links unknown (NULL)", Token: tenant(func(c claims) { c["sub"] = "kc-links-null" }), WantErr: identity.ErrTenantUnresolved},
			{Name: "no principal for the subject", Token: tenant(func(c claims) { c["sub"] = "kc-nobody" }), WantErr: identity.ErrTenantUnresolved},
			{Name: "expired", Token: tenant(func(c claims) { c["exp"] = now.Add(-time.Minute).Unix() }), WantErr: identity.ErrInvalidToken},
			{Name: "operator token", Token: operator(func(claims) {}), WantErr: identity.ErrInvalidToken},
		},
		Operator: h.a,
		OperatorCases: []contracttest.OperatorCase{
			{Name: "operator", Token: operator(func(claims) {}), Want: &identity.Operator{PrincipalID: defaultSub}},
			{Name: "no role", Token: operator(func(c claims) { delete(c, "realm_access") }), WantErr: identity.ErrInvalidToken},
			{Name: "tenant token", Token: tenant(func(claims) {}), WantErr: identity.ErrInvalidToken},
		},
	}.Run(t)
}

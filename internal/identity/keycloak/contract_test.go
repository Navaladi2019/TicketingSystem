package keycloak

import (
	"testing"
	"time"

	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/identity/contracttest"
)

// TestContract runs the identity port's contract suite against the adapter.
// T17 replaces the "one organization" case with resolving ones, T18 adds the
// DirectoryAdmin, T19 its adversarial cases.
func TestContract(t *testing.T) {
	t.Parallel()
	h := newHarness(t, []string{realmNava})
	s := h.idp.signer(realmNava)
	now := h.clock.now()
	tenant := func(edit func(claims)) string {
		c := h.idp.tenantClaims(realmNava, now)
		edit(c)
		return sign(t, s, c)
	}
	operator := func(edit func(claims)) string {
		c := h.idp.operatorClaims(realmNava, now)
		edit(c)
		return sign(t, s, c)
	}

	contracttest.Suite{
		Resolver: h.a,
		ResolverCases: []contracttest.ResolverCase{
			{Name: "one organization, no binding yet", Token: tenant(func(claims) {}), WantErr: identity.ErrTenantUnresolved},
			{Name: "no organization", Token: tenant(func(c claims) { delete(c, "organization") }), WantErr: identity.ErrTenantUnresolved},
			{Name: "expired", Token: tenant(func(c claims) { c["exp"] = now.Add(-time.Minute).Unix() }), WantErr: identity.ErrInvalidToken},
			{Name: "operator token", Token: operator(func(claims) {}), WantErr: identity.ErrInvalidToken},
		},
		Operator: h.a,
		OperatorCases: []contracttest.OperatorCase{
			{Name: "operator", Token: operator(func(claims) {}), Want: &identity.Operator{PrincipalID: "6f1c2a7e-0000-4000-8000-000000000001"}},
			{Name: "no role", Token: operator(func(c claims) { delete(c, "realm_access") }), WantErr: identity.ErrInvalidToken},
			{Name: "tenant token", Token: tenant(func(claims) {}), WantErr: identity.ErrInvalidToken},
		},
	}.Run(t)
}

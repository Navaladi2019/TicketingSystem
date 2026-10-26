package isolation

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Navaladi2019/TicketingSystem/internal/identity"
)

// CI-5 (T19, TICK-75). The whole catalogue runs against the real Keycloak in
// make identity-attacks (internal/identitystack TestCI5, ci5-report.md). Here
// every case is registered by its catalogue id, and the ones this suite can
// show run: the token shapes through the real API handler, which must answer
// 401 for the stated reason, and the duplicate binding against the suite's
// database. The cases only Keycloak can show (its broker, its organization
// picker, its key per realm, its ID tokens) are Stack paths on TestCI5, so
// TestStackPathsExist fails if that test goes away.

// HTTPOrgA is the organization of the HTTP runner's test realm, bound to
// tenant A for the run with the one IdP alias HTTPIdPA.
const (
	HTTPOrgA = "org-a-http"
	HTTPIdPA = "idp-a"
)

// TokenShape is a token an HTTP path sends instead of the runner's tenant A
// token. The runner mints it, and the path passes only when the API answers
// 401 and the resolver refuses the same token with WantErr for WantReason.
type TokenShape struct {
	// Subject is the sub claim (principals.keycloak_user_id).
	Subject string
	// Orgs maps organization alias to id; nil leaves the claim out.
	Orgs map[string]string
	// IdP is the idp claim; empty leaves it out.
	IdP string
	// Untrusted mints the token in a second realm the API does not trust.
	Untrusted bool
	// WantErr is the identity sentinel the refusal must wrap.
	WantErr error
	// WantReason is the adapter's fixed refusal reason ("idp not bound"),
	// which the sentinel alone cannot tell apart: a path refused for another
	// reason, such as a missing seed principal, proves nothing.
	WantReason string
}

// identityAttacksTarget is the make target that runs TestCI5.
const identityAttacksTarget = "make identity-attacks"

func ci5Paths() []Path {
	refused := func(id, name string, tok TokenShape) Path {
		return Path{Name: id + " " + name, Case: "CI-5", HTTP: &HTTPCall{Method: http.MethodGet, Path: "/api/v1/me", Token: &tok}}
	}
	onStack := func(id, name string) Path {
		return Path{Name: id + " " + name, Case: "CI-5", Stack: "TestCI5", StackTarget: identityAttacksTarget}
	}
	orgA := map[string]string{"tenant-a": HTTPOrgA}
	unresolved := func(tok TokenShape, reason string) TokenShape {
		tok.WantErr, tok.WantReason = identity.ErrTenantUnresolved, reason
		return tok
	}
	return []Path{
		refused("CI-5.01", "zero organizations", unresolved(TokenShape{Subject: "kc-user-shared"}, "no organization")),
		refused("CI-5.02", "two organizations", unresolved(TokenShape{Subject: "kc-user-shared",
			Orgs: map[string]string{"tenant-a": HTTPOrgA, "tenant-b": "org-b-http"}}, "organization count")),
		refused("CI-5.03", "unknown issuer", TokenShape{Subject: "kc-user-shared", Orgs: orgA, Untrusted: true,
			WantErr: identity.ErrInvalidToken, WantReason: "issuer"}),
		onStack("CI-5.04", "realm X's key with iss=Y"),
		onStack("CI-5.05", "an ID token as an access token"),
		onStack("CI-5.06", "the kc.org organization-picker tamper"),
		onStack("CI-5.07", "a brokered login with an existing email"),
		onStack("CI-5.08", "an out-of-domain brokered user"),
		refused("CI-5.09", "tenant B's IdP alias in tenant A's token",
			unresolved(TokenShape{Subject: "kc-user-shared", Orgs: orgA, IdP: "idp-b"}, "idp not bound")),
		refused("CI-5.10", "a tenant A principal federated to tenant B's IdP",
			unresolved(TokenShape{Subject: "kc-user-a-foreign", Orgs: orgA}, "federated link not bound")),
		onStack("CI-5.11", "an invitation of a user federated to a foreign IdP"),
		{Name: "CI-5.12 a duplicate binding", Case: "CI-5", DB: duplicateBindingRefused},
		refused("CI-5.13", "a principal only of tenant B names tenant A's organization",
			unresolved(TokenShape{Subject: "kc-user-b2", Orgs: orgA}, "principal not synced")),
		onStack("CI-5.14", "allowed: an in-domain brokered user"),
	}
}

// bindingIssuerOrgKey is the constraint that makes (issuer, org_id) name one
// tenant. The primary key also holds tenant_id, so only this one refuses a
// second tenant.
const bindingIssuerOrgKey = "tenant_identity_binding_issuer_org_key"

// duplicateBindingRefused: (issuer, organization) names one tenant, for every
// role, the superuser included. A fresh pair binds (the control). Tenant A's
// pair cannot be bound for tenant B (bindingIssuerOrgKey) nor again for
// tenant A (the primary key).
func duplicateBindingRefused(ctx context.Context, c Cluster) error {
	super, err := c.Super()
	if err != nil {
		return err
	}
	const issuer = "https://idp.example/realms/nava"
	insert := "INSERT INTO tenant_identity_binding (tenant_id, issuer, org_id) VALUES ($1, $2, $3)"
	if _, err := super.Exec(ctx, insert, TenantB, issuer, "org-ci5-control"); err != nil {
		return fmt.Errorf("control: a fresh binding: %w", err)
	}
	if _, err := super.Exec(ctx, "DELETE FROM tenant_identity_binding WHERE org_id = 'org-ci5-control'"); err != nil {
		return err
	}
	for _, want := range []struct{ tenant, constraint string }{
		{TenantB, bindingIssuerOrgKey},
		{TenantA, "tenant_identity_binding_pkey"},
	} {
		_, err := super.Exec(ctx, insert, want.tenant, issuer, "org-a")
		switch {
		case err == nil:
			if want.tenant != TenantA { // tenant A's own seeded row must stay
				_, _ = super.Exec(ctx, "DELETE FROM tenant_identity_binding WHERE tenant_id = $1 AND org_id = 'org-a'", want.tenant)
			}
			return fmt.Errorf("a second binding of org-a for %s was stored, want SQLSTATE 23505 on %s", want.tenant, want.constraint)
		case sqlState(err) != "23505" || constraintName(err) != want.constraint:
			return fmt.Errorf("a second binding of org-a for %s: %w, want SQLSTATE 23505 on %s", want.tenant, err, want.constraint)
		}
	}
	return nil
}

// constraintName is the constraint a server error names, or "".
func constraintName(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.ConstraintName
	}
	return ""
}

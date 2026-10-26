package isolation

import (
	"context"
	"errors"
	"fmt"
	"net/http"
)

// CI-5 (T19, TICK-75). The whole catalogue runs against the real Keycloak in
// make identity-attacks (internal/identitystack TestCI5, the ci5-report
// artifact). Here every case is registered by its catalogue id, and the ones
// this suite can show run: the token shapes through the real API handler,
// which must answer 401, and the duplicate binding against the suite's
// database. The cases only Keycloak can show (its broker, its organization
// picker, its key per realm, its ID tokens) are listed as pending on that
// suite, so the report names all fourteen.

// HTTPOrgA is the organization of the HTTP runner's test realm, bound to
// tenant A for the run with no IdP alias.
const HTTPOrgA = "org-a-http"

// TokenShape is a token an HTTP path sends instead of the runner's tenant A
// token. The runner mints it; a path with one passes only on 401.
type TokenShape struct {
	// Subject is the sub claim (principals.keycloak_user_id).
	Subject string
	// Orgs maps organization alias to id; nil leaves the claim out.
	Orgs map[string]string
	// IdP is the idp claim; empty leaves it out.
	IdP string
	// Untrusted mints the token in a second realm the API does not trust.
	Untrusted bool
}

// identityAttacks is where a Keycloak-only case runs.
const identityAttacks = "make identity-attacks (TestCI5, real Keycloak)"

func ci5Paths() []Path {
	refused := func(id, name string, tok TokenShape) Path {
		return Path{Name: id + " " + name, Case: "CI-5", HTTP: &HTTPCall{Method: http.MethodGet, Path: "/api/v1/me", Token: &tok}}
	}
	keycloakOnly := func(id, name string) Path {
		return Path{Name: id + " " + name, Case: "CI-5", DB: notHere, Pending: identityAttacks}
	}
	orgA := map[string]string{"tenant-a": HTTPOrgA}
	return []Path{
		refused("CI-5.01", "zero organizations", TokenShape{Subject: "kc-user-shared"}),
		refused("CI-5.02", "two organizations", TokenShape{Subject: "kc-user-shared",
			Orgs: map[string]string{"tenant-a": HTTPOrgA, "tenant-b": "org-b-http"}}),
		refused("CI-5.03", "unknown issuer", TokenShape{Subject: "kc-user-shared", Orgs: orgA, Untrusted: true}),
		keycloakOnly("CI-5.04", "realm X's key with iss=Y"),
		keycloakOnly("CI-5.05", "an ID token as an access token"),
		keycloakOnly("CI-5.06", "the kc.org organization-picker tamper"),
		keycloakOnly("CI-5.07", "a brokered login with an existing email"),
		keycloakOnly("CI-5.08", "an out-of-domain brokered user"),
		refused("CI-5.09", "tenant B's IdP alias in tenant A's token", TokenShape{Subject: "kc-user-shared", Orgs: orgA, IdP: "idp-b"}),
		refused("CI-5.10", "a tenant A principal federated to tenant B's IdP", TokenShape{Subject: "kc-user-a-foreign", Orgs: orgA}),
		keycloakOnly("CI-5.11", "an invitation of a user federated to a foreign IdP"),
		{Name: "CI-5.12 a duplicate binding", Case: "CI-5", DB: duplicateBindingRefused},
		refused("CI-5.13", "a principal only of tenant B names tenant A's organization", TokenShape{Subject: "kc-user-b2", Orgs: orgA}),
		keycloakOnly("CI-5.14", "allowed: an in-domain brokered user"),
	}
}

func notHere(context.Context, Cluster) error {
	return errors.New("this case needs the real Keycloak: " + identityAttacks)
}

// duplicateBindingRefused: (issuer, organization) names one tenant, for every
// role, the superuser included. A fresh pair binds (the control), and tenant
// A's pair cannot be bound again, for tenant B or for tenant A.
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
	for _, tenant := range []string{TenantB, TenantA} {
		_, err := super.Exec(ctx, insert, tenant, issuer, "org-a")
		switch {
		case err == nil:
			_, _ = super.Exec(ctx, "DELETE FROM tenant_identity_binding WHERE tenant_id = $1 AND org_id = 'org-a'", TenantB)
			return fmt.Errorf("a second binding of org-a for %s was stored, want SQLSTATE 23505", tenant)
		case sqlState(err) != "23505":
			return fmt.Errorf("a second binding of org-a for %s: %w, want SQLSTATE 23505", tenant, err)
		}
	}
	return nil
}

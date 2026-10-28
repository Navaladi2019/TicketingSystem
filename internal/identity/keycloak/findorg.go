package keycloak

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/identity"
)

// FindTenantOrg finds the organization a CreateTenantOrg that was cut short
// (a crash, a lost reply) left for tenantID, so onboarding can resume with one
// organization instead of failing on the duplicate. It uses the onboarding
// account and only ever looks at the alias and the IdP alias derived from the
// tenant id.
//
//   - Organization and IdP both exist, the IdP is linked to that organization
//     alone, and it matches the onboarding policy: found, with the org id.
//   - Only the organization exists: it is ours (the alias is derived from the
//     tenant id), so it is deleted and found=false; CreateTenantOrg then starts
//     clean.
//   - Neither exists: found=false.
//   - Anything else (the IdP is linked elsewhere or breaks the policy):
//     identity.ErrAlreadyExists. Nothing is changed.
func (d *Directory) FindTenantOrg(ctx context.Context, tenantID uuid.UUID, alias string) (string, bool, error) {
	if err := requireTenant(tenantID); err != nil {
		return "", false, err
	}
	if d.onboarding == nil {
		return "", false, ErrOnboardingDisabled
	}
	if !aliasRE.MatchString(alias) {
		return "", false, fmt.Errorf("%w: alias", identity.ErrInvalidInput)
	}
	var orgs []struct {
		ID    string `json:"id"`
		Alias string `json:"alias"`
	}
	if err := d.api.getJSON(ctx, d.onboarding, "/organizations", "/organizations",
		url.Values{"search": {alias}, "exact": {"true"}, "max": {"5"}}, &orgs); err != nil {
		return "", false, err
	}
	var orgID string
	for _, o := range orgs {
		if o.Alias == alias {
			orgID = o.ID
		}
	}
	idpAlias := TenantIdPAlias(tenantID)
	idp, idpErr := d.readIdP(ctx, idpAlias)
	switch {
	case idpErr != nil && !errors.Is(idpErr, identity.ErrNotFound):
		return "", false, idpErr
	case orgID == "" && idpErr != nil:
		return "", false, nil
	case orgID == "":
		return "", false, fmt.Errorf("%w: idp without its organization", identity.ErrAlreadyExists)
	case idpErr != nil:
		if _, err := d.api.do(ctx, d.onboarding, http.MethodDelete, "/organizations/{id}",
			"/organizations/"+url.PathEscape(orgID), nil, nil, http.StatusNoContent); err != nil {
			return "", false, err
		}
		return "", false, nil
	}
	if len(idp.OrganizationLinks) != 1 || idp.OrganizationLinks[0].OrganizationID != orgID ||
		idp.TrustEmail || idp.FirstBrokerLoginFlowAlias != FirstBrokerLoginFlow || idp.PostBrokerLoginFlowAlias != "" {
		return "", false, fmt.Errorf("%w: existing idp does not match the onboarding policy", identity.ErrAlreadyExists)
	}
	if err := d.ensureEmailVerifiedMapper(ctx, idpAlias, idp.ProviderID); err != nil {
		return "", false, err
	}
	return orgID, true, nil
}

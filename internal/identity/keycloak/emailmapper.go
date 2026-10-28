package keycloak

// The upstream email_verified claim (TICK-252, Security ruling 2026-10-27).
// trustEmail is false, so Keycloak never marks a brokered email verified; the
// claim the tenant's IdP sends is copied to a user attribute that only an
// admin may edit (declared in realm nava's user profile), and PlaceInvitee
// places a user only when it is exactly "true". An IdP that does not emit the
// claim, or emits anything else, means the invited admin is never placed: that
// fails closed.

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/Navaladi2019/TicketingSystem/internal/identity"
)

const (
	// EmailVerifiedAttribute is the user attribute the mapper writes.
	EmailVerifiedAttribute = "nava_idp_email_verified"
	// emailVerifiedMapperName is the one mapper every tenant IdP carries.
	emailVerifiedMapperName = "nava-idp-email-verified"
)

type kcMapper struct {
	ID                     string            `json:"id,omitempty"`
	Name                   string            `json:"name"`
	IdentityProviderAlias  string            `json:"identityProviderAlias"`
	IdentityProviderMapper string            `json:"identityProviderMapper"`
	Config                 map[string]string `json:"config"`
}

// emailVerifiedMapper is the mapper an IdP of providerID must carry.
func emailVerifiedMapper(idpAlias, providerID string) kcMapper {
	mapper, claimKey := "oidc-user-attribute-idp-mapper", "claim"
	if providerID == "saml" {
		mapper, claimKey = "saml-user-attribute-idp-mapper", "attribute.name"
	}
	return kcMapper{
		Name: emailVerifiedMapperName, IdentityProviderAlias: idpAlias, IdentityProviderMapper: mapper,
		Config: map[string]string{"syncMode": "FORCE", claimKey: "email_verified", "user.attribute": EmailVerifiedAttribute},
	}
}

func (m kcMapper) matches(want kcMapper) bool {
	if m.Name != want.Name || m.IdentityProviderMapper != want.IdentityProviderMapper || len(m.Config) != len(want.Config) {
		return false
	}
	for k, v := range want.Config {
		if m.Config[k] != v {
			return false
		}
	}
	return true
}

// ensureEmailVerifiedMapper makes the IdP carry exactly the email_verified
// mapper and no other. An IdP with no mapper gets it; an IdP with any other
// mapper is identity.ErrAlreadyExists and is left alone.
func (d *Directory) ensureEmailVerifiedMapper(ctx context.Context, idpAlias, providerID string) error {
	list := func() ([]kcMapper, error) {
		var ms []kcMapper
		err := d.api.getJSON(ctx, d.onboarding, "/identity-provider/instances/{alias}/mappers",
			"/identity-provider/instances/"+url.PathEscape(idpAlias)+"/mappers", nil, &ms)
		return ms, err
	}
	want := emailVerifiedMapper(idpAlias, providerID)
	ms, err := list()
	if err != nil {
		return err
	}
	if len(ms) == 0 {
		if _, err := d.api.do(ctx, d.onboarding, http.MethodPost, "/identity-provider/instances/{alias}/mappers",
			"/identity-provider/instances/"+url.PathEscape(idpAlias)+"/mappers", nil, want, http.StatusCreated); err != nil {
			return err
		}
		if ms, err = list(); err != nil {
			return err
		}
	}
	if len(ms) != 1 || !ms[0].matches(want) {
		return fmt.Errorf("%w: idp mappers are not exactly the email_verified mapper", identity.ErrAlreadyExists)
	}
	return nil
}

package keycloak

// PlaceInvitee is the second half of the invite lifecycle (TICK-252, option 2):
// group sync calls it, per pending invite, to put the invitee in the tenant's
// group after their first login through the tenant's IdP.
//
// The user is placed only if every condition of the Security ruling (2c) holds:
//   - the user exists, is enabled, and their email is the invite's email (the
//     email the tenant's IdP asserted at first login; trustEmail is false, so
//     it is a claim, not a proof, and the IdP binding below is what vouches
//     for it);
//   - the user has at least one federated link and every link is an IdP bound
//     to the tenant;
//   - the user is a member of the tenant's organization (matched by the org
//     id in tenant_identity_binding) and of no other;
//   - the user is in no other tenant's /t/ group;
//   - the invite is unexpired and unconsumed, which consume decides (2d).
//
// A user already in this tenant's group, with every other condition true, has
// the invite consumed and no second group write. Any failed condition places
// nobody and leaves the invite untouched. Nothing
// here links an identity to a user: a user the tenant's IdP did not create is
// not placed.

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"slices"

	"github.com/google/uuid"
)

type kcOrgRef struct {
	ID string `json:"id"`
}

// PlaceInvitee places the user whose email is email in /t/<tenant_id>, if the
// conditions in the file comment hold. bound are the IdP aliases and orgIDs the
// Keycloak organization ids bound to the tenant (tenant_identity_binding), read
// by the caller under row-level security. The organization is matched by id,
// never by an alias rebuilt from the tenant id. consume marks the
// tenant's invite for email consumed and reports whether it did (an unexpired,
// unconsumed invite existed); it runs after every Keycloak check and before
// the group write, so an invite is never consumed for a user who is not
// placed, and one invite places one user. It reports whether the user was
// placed. A refusal is (false, nil).
func (d *Directory) PlaceInvitee(ctx context.Context, tenantID uuid.UUID, email string, bound, orgIDs []string,
	consume func(context.Context) (bool, error)) (bool, error) {
	if err := requireTenant(tenantID); err != nil {
		return false, err
	}
	email, err := normalizeEmail(email)
	if err != nil {
		return false, err
	}
	if len(bound) == 0 || len(orgIDs) == 0 {
		return false, nil
	}
	user, err := d.userByEmail(ctx, email)
	if err != nil || user.ID == "" || !user.Enabled {
		return false, err
	}
	links, err := d.federatedLinks(ctx, user.ID)
	if err != nil {
		return false, err
	}
	if len(links) == 0 {
		return false, nil
	}
	for _, l := range links {
		if !slices.Contains(bound, l.IdentityProvider) {
			return false, nil
		}
	}
	groups, err := pages[kcGroup](ctx, d.api, d.runtime, "/users/{id}/groups", "/users/"+url.PathEscape(user.ID)+"/groups", nil)
	if err != nil {
		return false, err
	}
	if slices.ContainsFunc(groups, func(g kcGroup) bool { return inOtherTenant(tenantID, g.Path) }) {
		return false, nil
	}
	// A user already in this tenant's group (an earlier cycle placed them but
	// its transaction did not commit) is not written again, but the invite is
	// still consumed so it does not stay open.
	inTenantAlready := slices.ContainsFunc(groups, func(g kcGroup) bool { return inTenant(tenantID, g.Path) })
	var orgs []kcOrgRef
	if err := d.api.getJSON(ctx, d.runtime, "/organizations/members/{id}/organizations",
		"/organizations/members/"+url.PathEscape(user.ID)+"/organizations", nil, &orgs); err != nil {
		return false, err
	}
	if len(orgs) == 0 || slices.ContainsFunc(orgs, func(o kcOrgRef) bool { return !slices.Contains(orgIDs, o.ID) }) {
		return false, nil
	}
	root, err := d.rootGroup(ctx, tenantID)
	if err != nil {
		return false, err
	}
	ok, err := consume(ctx)
	if err != nil {
		return false, fmt.Errorf("keycloak: consume invite: %w", err)
	}
	if !ok {
		return false, nil
	}
	if inTenantAlready {
		return true, nil
	}
	if _, err := d.api.do(ctx, d.runtime, http.MethodPut, "/users/{id}/groups/{groupId}",
		"/users/"+url.PathEscape(user.ID)+"/groups/"+url.PathEscape(root.ID), nil, nil, http.StatusNoContent); err != nil {
		return false, err
	}
	return true, nil
}

// userByEmail returns the user with exactly this email, or the zero value.
func (d *Directory) userByEmail(ctx context.Context, email string) (kcUser, error) {
	var us []kcUser
	if err := d.api.getJSON(ctx, d.runtime, "/users", "/users", url.Values{"email": {email}, "exact": {"true"}}, &us); err != nil {
		return kcUser{}, err
	}
	for _, u := range us {
		if u.Email == email {
			return u, nil
		}
	}
	return kcUser{}, nil
}

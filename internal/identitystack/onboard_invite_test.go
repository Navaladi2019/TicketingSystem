//go:build identitystack

package identitystack

// Negative tests for the pending-invite lifecycle (TICK-252, option 2): the
// invite no longer creates a Keycloak user; the first brokered login does, and
// group sync places the user in /t/<tenant_id>/ only when every Security
// condition (2c) holds. Each test onboards real tenants through nava-onboard
// and carries a positive control in the same fixture: a second invitee of the
// same tenant and IdP who is placed and whose invite is consumed, so a test
// cannot pass because nothing is ever placed.
//
//	make stack-up
//	make stack-test-onboard

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/identity/keycloak"
	"github.com/Navaladi2019/TicketingSystem/internal/onboard"
)

// tenantFx is one onboarded tenant with its own IdP realm.
type tenantFx struct {
	tenant                     uuid.UUID
	alias, idpAlias, realm     string
	domain, admin, pass, orgID string
}

// onboardFx onboards a tenant whose first admin is invited and whose IdP is
// repointed at a realm holding that admin.
func onboardFx(t *testing.T) tenantFx {
	t.Helper()
	key, tenant, domain, email := newOnboarding(t)
	f := tenantFx{
		tenant: tenant, alias: onboard.Alias(tenant), idpAlias: keycloak.TenantIdPAlias(tenant),
		realm: "onb-" + tenant.String()[:8], domain: domain, admin: email, pass: "dev-only-" + uuid.NewString()[:8],
	}
	createIdPRealm(t, f.realm, email, f.pass)
	mustOnboard(t, operatorToken(t), key, onboardBody(email, domain), http.StatusCreated, tenant)
	pointIdPAt(t, f.idpAlias, f.realm)
	ids := orgsByAlias(t, f.alias)
	if len(ids) != 1 {
		t.Fatalf("organizations %s: %v", f.alias, ids)
	}
	f.orgID = ids[0]
	return f
}

func inviteState(t *testing.T, tenant uuid.UUID, email string) string {
	t.Helper()
	got := psqlSuper(t, fmt.Sprintf("SELECT CASE WHEN consumed_at IS NULL THEN 'pending' ELSE 'consumed' END "+
		"FROM pending_invite WHERE tenant_id = '%s' AND email = '%s'", tenant, email))
	if got == "" {
		return "missing"
	}
	return got
}

func (f tenantFx) placed(t *testing.T, email string) bool {
	t.Helper()
	id := kcUserByEmail(t, email)
	return id != "" && userInGroup(t, id, kcGroupID(t, "/t/"+f.tenant.String()))
}

func fedLinks(t *testing.T, email string) []string {
	t.Helper()
	id := kcUserByEmail(t, email)
	if id == "" {
		return nil
	}
	code, b := adminRealm(t, "nava", http.MethodGet, "/users/"+id+"/federated-identity", nil)
	var ls []struct{ IdentityProvider string }
	if code != http.StatusOK || json.Unmarshal(b, &ls) != nil {
		t.Fatalf("federated identities of %s: %d %s", email, code, b)
	}
	var out []string
	for _, l := range ls {
		out = append(out, l.IdentityProvider)
	}
	return out
}

// invite adds a pending invite the way onboarding would have, for a second
// invitee of the same tenant, and the IdP user who will sign in as them.
func (f tenantFx) invite(t *testing.T, email string) {
	t.Helper()
	seedIdPUser(t, f.realm, email, email, f.pass)
	deleteUserAfter(t, email)
	psqlSuper(t, fmt.Sprintf("INSERT INTO pending_invite (tenant_id, email, expires_at, created_at) "+
		"VALUES ('%s', '%s', now() + interval '7 days', now() - interval '1 day')", f.tenant, email))
}

// control signs a second invitee in through the tenant's own IdP; after the
// next group sync cycle requirePlaced must hold.
func (f tenantFx) control(t *testing.T) string {
	t.Helper()
	email := "control-" + uuid.NewString()[:8] + "@" + f.domain
	f.invite(t, email)
	brokeredSignIn(t, f.idpAlias, email, f.pass)
	return email
}

func (f tenantFx) requirePlaced(t *testing.T, email string) {
	t.Helper()
	if !f.placed(t, email) || inviteState(t, f.tenant, email) != "consumed" {
		t.Errorf("positive control %s: placed=%v invite=%s, want placed and consumed",
			email, f.placed(t, email), inviteState(t, f.tenant, email))
	}
}

// squatAdminOfA: B's IdP asserts the address of A's invited admin, which has a
// pending A invite. The user it creates is not placed in /t/<A>/, holds no
// link to A's IdP, and A's invite stays unconsumed. (Security tests 1 and 10,
// writer part, and the CTO's squatting case; the squatted admin is refused
// later, AR-033.)
func squatAdminOfA(t *testing.T) {
	stopGroupsync(t)
	a, b := onboardFx(t), onboardFx(t)
	seedIdPUser(t, b.realm, a.admin, a.admin, b.pass)
	deleteUserAfter(t, a.admin)
	ctl := a.control(t)

	// B's IdP login may succeed (it creates a user) or be refused; either way
	// nothing is linked to A and nothing is placed in A.
	brokeredAttempt(t, b.idpAlias, a.admin, b.pass)
	groupsyncOnce(t)

	if a.placed(t, a.admin) {
		t.Errorf("the user B's IdP created is in /t/%s", a.tenant)
	}
	if links := fedLinks(t, a.admin); slices.Contains(links, a.idpAlias) {
		t.Errorf("the user holds a link to A's IdP: %v", links)
	}
	if id := kcUserByEmail(t, a.admin); id != "" && slices.Contains(orgMembers(t, a.orgID), id) {
		t.Errorf("the user is a member of A's organization")
	}
	if got := inviteState(t, a.tenant, a.admin); got != "pending" {
		t.Errorf("A's invite = %q, want pending", got)
	}
	a.requirePlaced(t, ctl)
}

func TestInvitedToTenantANotLinkedThroughTenantBIdP(t *testing.T) { squatAdminOfA(t) }

func TestSquattedInviteeIsNotPlacedInTheOtherTenant(t *testing.T) { squatAdminOfA(t) }

// Test 2: A's IdP asserts the seeded local user alice@tenant-a.test, who has a
// password and was never invited. The login is refused, alice gains no link,
// and her password login still works.
func TestUninvitedLocalUserNotLinked(t *testing.T) {
	stopGroupsync(t)
	f := onboardFx(t)
	seedIdPUser(t, f.realm, "alice@tenant-a.test", "alice@tenant-a.test", f.pass)
	ctl := f.control(t)

	if tok, _ := brokeredAttempt(t, f.idpAlias, "alice@tenant-a.test", f.pass); tok != "" {
		c := claims(t, tok)
		t.Fatalf("A's IdP asserting alice got a token: sub=%v", c["sub"])
	}
	groupsyncOnce(t)

	if links := fedLinks(t, "alice@tenant-a.test"); len(links) != 0 {
		t.Errorf("alice's federated identities = %v, want none", links)
	}
	if tok := passwordToken(t, "nava-web", "alice@tenant-a.test", "dev-only-alice"); tok == "" {
		t.Error("alice's password login no longer works")
	}
	f.requirePlaced(t, ctl)
}

// Test 4: after the admin is placed and the invite consumed, a second subject
// of A's IdP with the same email is refused and the first link is unchanged.
func TestInviteIsSingleUse(t *testing.T) {
	stopGroupsync(t)
	f := onboardFx(t)
	brokeredSignIn(t, f.idpAlias, f.admin, f.pass)
	groupsyncOnce(t)
	f.requirePlaced(t, f.admin)
	before := fedLinks(t, f.admin)

	seedIdPUser(t, f.realm, "second-subject", f.admin, f.pass)
	if tok, _ := brokeredAttempt(t, f.idpAlias, "second-subject", f.pass); tok != "" {
		t.Fatalf("a second subject with the admin's email got a token: %v", claims(t, tok)["sub"])
	}
	groupsyncOnce(t)
	if got := fedLinks(t, f.admin); !slices.Equal(got, before) || len(got) != 1 {
		t.Errorf("the admin's links = %v, were %v", got, before)
	}
	if got := inviteState(t, f.tenant, f.admin); got != "consumed" {
		t.Errorf("invite = %q, want consumed", got)
	}
}

// Test 5: an invite that expired before the first login places nobody.
func TestExpiredInviteNotLinked(t *testing.T) {
	stopGroupsync(t)
	f := onboardFx(t)
	psqlSuper(t, fmt.Sprintf("UPDATE pending_invite SET created_at = now() - interval '8 days', expires_at = now() - interval '1 day' "+
		"WHERE tenant_id = '%s' AND email = '%s'", f.tenant, f.admin))
	ctl := f.control(t)

	brokeredAttempt(t, f.idpAlias, f.admin, f.pass)
	groupsyncOnce(t)

	if f.placed(t, f.admin) {
		t.Error("an expired invite placed the user")
	}
	if got := inviteState(t, f.tenant, f.admin); got != "pending" {
		t.Errorf("expired invite = %q, want untouched (pending)", got)
	}
	f.requirePlaced(t, ctl)
}

// Test 6: an invitee who also belongs to another tenant's /t/ group is not
// placed in this one.
func TestInvitedUserInTwoTenantsNotLinked(t *testing.T) {
	stopGroupsync(t)
	a, b := onboardFx(t), onboardFx(t)
	ctl := a.control(t)

	brokeredSignIn(t, a.idpAlias, a.admin, a.pass)
	id := kcUserByEmail(t, a.admin)
	if code, body := adminRealm(t, "nava", http.MethodPut, "/users/"+id+"/groups/"+kcGroupID(t, "/t/"+b.tenant.String()), nil); code != http.StatusNoContent {
		t.Fatalf("add the invitee to /t/%s: %d %s", b.tenant, code, body)
	}
	groupsyncOnce(t)

	if a.placed(t, a.admin) {
		t.Error("an invitee in another tenant's group was placed")
	}
	if got := inviteState(t, a.tenant, a.admin); got != "pending" {
		t.Errorf("invite = %q, want pending", got)
	}
	a.requirePlaced(t, ctl)
}

// Test 7: before the IdP login the invitee has no Keycloak user, so neither
// the password grant nor a browser login works; after it they have no password.
func TestInvitedUserHasNoLocalLogin(t *testing.T) {
	stopGroupsync(t)
	f := onboardFx(t)
	if kcUserByEmail(t, f.admin) != "" {
		t.Fatalf("a Keycloak user exists for %s before any login", f.admin)
	}
	passwordGrant := func() int {
		status, _ := tokenForm(t, "nava", url.Values{
			"grant_type": {"password"}, "client_id": {"nava-web"}, "username": {f.admin}, "password": {f.pass}, "scope": {"openid"},
		})
		return status
	}
	if passwordGrant() == http.StatusOK {
		t.Fatal("the password grant works for an invitee who has not signed in")
	}
	ctl := f.control(t)

	brokeredSignIn(t, f.idpAlias, f.admin, f.pass)
	id := kcUserByEmail(t, f.admin)
	code, b := adminRealm(t, "nava", http.MethodGet, "/users/"+id+"/credentials", nil)
	if code != http.StatusOK || string(b) != "[]" {
		t.Errorf("credentials of the brokered invitee: %d %s, want []", code, b)
	}
	if passwordGrant() == http.StatusOK {
		t.Error("the password grant works for the brokered invitee")
	}
	groupsyncOnce(t)
	f.requirePlaced(t, ctl)
}

// Test 8: the invitee is disabled between the first login and the next cycle.
func TestDisabledInviteeNotLinked(t *testing.T) {
	stopGroupsync(t)
	f := onboardFx(t)
	ctl := f.control(t)

	brokeredSignIn(t, f.idpAlias, f.admin, f.pass)
	id := kcUserByEmail(t, f.admin)
	if code, b := adminRealm(t, "nava", http.MethodPut, "/users/"+id, map[string]any{"enabled": false}); code != http.StatusNoContent {
		t.Fatalf("disable the invitee: %d %s", code, b)
	}
	groupsyncOnce(t)

	if f.placed(t, f.admin) {
		t.Error("a disabled invitee was placed")
	}
	if got := inviteState(t, f.tenant, f.admin); got != "pending" {
		t.Errorf("invite = %q, want pending", got)
	}
	f.requirePlaced(t, ctl)
}

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
	"html"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strings"
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

	// The upstream realm refuses a second user with the same email, so the
	// admin's upstream user goes first: the second subject is a new subject
	// of the same IdP asserting the same address. Realm nava is not touched.
	_, ub := adminRealm(t, f.realm, http.MethodGet, "/users?exact=true&email="+url.QueryEscape(f.admin), nil)
	var ups []struct{ ID string }
	if json.Unmarshal(ub, &ups) != nil || len(ups) != 1 {
		t.Fatalf("upstream users with %s: %s", f.admin, ub)
	}
	if code, b := adminRealm(t, f.realm, http.MethodDelete, "/users/"+ups[0].ID, nil); code != http.StatusNoContent {
		t.Fatalf("delete the admin's upstream user: %d %s", code, b)
	}
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

// setUpstreamEmailVerified sets emailVerified on the user with this email in
// the tenant's upstream IdP realm; the realm's email_verified claim follows.
func setUpstreamEmailVerified(t *testing.T, realm, email string, verified bool) {
	t.Helper()
	_, b := adminRealm(t, realm, http.MethodGet, "/users?exact=true&email="+url.QueryEscape(email), nil)
	var us []map[string]any
	if json.Unmarshal(b, &us) != nil || len(us) != 1 {
		t.Fatalf("upstream users with %s in %s: %s", email, realm, b)
	}
	us[0]["emailVerified"] = verified
	if code, body := adminRealm(t, realm, http.MethodPut, "/users/"+us[0]["id"].(string), us[0]); code != http.StatusNoContent {
		t.Fatalf("set emailVerified=%v: %d %s", verified, code, body)
	}
}

// storedEmailVerified is the nava_idp_email_verified attribute the IdP mapper
// wrote, as the admin API shows it ("" when absent).
func storedEmailVerified(t *testing.T, email string) string {
	t.Helper()
	id := kcUserByEmail(t, email)
	if id == "" {
		return ""
	}
	_, b := adminRealm(t, "nava", http.MethodGet, "/users/"+id, nil)
	var u struct{ Attributes map[string][]string }
	_ = json.Unmarshal(b, &u)
	return strings.Join(u.Attributes[keycloak.EmailVerifiedAttribute], ",")
}

// TestUnverifiedUpstreamEmailNotPlaced: the tenant's own IdP asserts the
// invited address with email_verified=false. The user signs in, but is not
// placed and the invite stays pending. Positive control: the same tenant and
// IdP with email_verified=true is placed and its invite consumed.
func TestUnverifiedUpstreamEmailNotPlaced(t *testing.T) {
	stopGroupsync(t)
	f := onboardFx(t)
	ctl := f.control(t)
	setUpstreamEmailVerified(t, f.realm, f.admin, false)

	brokeredSignIn(t, f.idpAlias, f.admin, f.pass)
	if got := storedEmailVerified(t, f.admin); got != "false" {
		t.Errorf("stored %s = %q, want the IdP's value \"false\"", keycloak.EmailVerifiedAttribute, got)
	}
	groupsyncOnce(t)

	if f.placed(t, f.admin) {
		t.Error("an invitee whose IdP asserted email_verified=false was placed")
	}
	if got := inviteState(t, f.tenant, f.admin); got != "pending" {
		t.Errorf("invite = %q, want pending", got)
	}
	if got := storedEmailVerified(t, ctl); got != "true" {
		t.Errorf("control %s = %q, want \"true\"", keycloak.EmailVerifiedAttribute, got)
	}
	f.requirePlaced(t, ctl)
}

// accountGet reads the signed-in user's own account with a token from the
// account-console client.
func accountGet(t *testing.T, tok string) (int, []byte) {
	t.Helper()
	get, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://"+inNetHost+"/realms/nava/account/", nil)
	get.Header.Set("Authorization", "Bearer "+tok)
	get.Header.Set("Accept", "application/json")
	return do(t, newInNetClient(t), get)
}

// accountClaims returns the token's audience and its account client roles
// (nil when the claim is absent).
func accountClaims(t *testing.T, tok string) (aud, roles any) {
	t.Helper()
	c := claims(t, tok)
	if ra, ok := c["resource_access"].(map[string]any); ok {
		if acct, ok := ra["account"].(map[string]any); ok {
			roles = acct["roles"]
		}
	}
	return c["aud"], roles
}

// TestAccountAPIClosedToPlainBrokeredUser is a canary, not a control. A plain
// brokered user's account-console token has aud=account but no
// resource_access.account claim, and the account API answers 403. That is an
// accident of token mapping (every user holds manage-account and view-profile
// through default-roles-nava, but the claim is not minted), so it protects
// nothing. If this starts failing, the account API has become reachable for
// tenant users, and the profile tests below still cover the fields: they use
// the application-initiated action forms, which need no account role. Realm
// nava's user profile is the control.
func TestAccountAPIClosedToPlainBrokeredUser(t *testing.T) {
	stopGroupsync(t)
	f := onboardFx(t)
	tok := accountConsoleSignIn(t, f.idpAlias, f.admin, f.pass)
	aud, roles := accountClaims(t, tok)
	t.Logf("account token: aud=%v resource_access.account.roles=%v", aud, roles)
	if roles != nil {
		t.Errorf("the token carries resource_access.account.roles = %v: the canary's premise no longer holds", roles)
	}
	if code, b := accountGet(t, tok); code != http.StatusForbidden {
		t.Errorf("GET the account as a plain brokered user = %d %.200s, want 403", code, b)
	}
}

// ---- application-initiated actions (AIA) through nava-web -------------------
//
// A tenant user reaches their own profile through the account API, which is
// closed to them (see the canary), and through required-action forms that need
// no account role: kc_action=UPDATE_PROFILE and UPDATE_EMAIL on a nava-web
// sign-in. Those are the self-edit paths the user profile must hold.

var (
	formTag  = regexp.MustCompile(`(?s)<form[^>]*>`)
	inputTag = regexp.MustCompile(`(?s)<input[^>]*>`)
	attrName = regexp.MustCompile(`\bname="([^"]*)"`)
	attrVal  = regexp.MustCompile(`\bvalue="([^"]*)"`)
	attrType = regexp.MustCompile(`\btype="([^"]*)"`)
	attrAct  = regexp.MustCompile(`\baction="([^"]*)"`)
	attrLock = regexp.MustCompile(`\b(readonly|disabled)\b`)
)

// formField is one <input> of a form.
type formField struct {
	value  string
	locked bool // readonly or disabled: the browser does not offer it for editing
}

// aiaForm is the required-action form Keycloak showed.
type aiaForm struct {
	action string
	fields map[string]formField
}

const aiaRedirect = "http://localhost:9/cb"

// aiaSignIn starts a nava-web sign-in through the tenant IdP with
// kc_action=action, logs in at the IdP, and returns the client (cookies kept)
// and the page Keycloak shows next.
func aiaSignIn(t *testing.T, f tenantFx, user, action string) (*http.Client, string) {
	t.Helper()
	c := newInNetClient(t)
	c.CheckRedirect = func(req *http.Request, _ []*http.Request) error {
		if strings.HasPrefix(req.URL.String(), aiaRedirect) {
			return http.ErrUseLastResponse
		}
		return nil
	}
	q := url.Values{
		"client_id": {"nava-web"}, "response_type": {"code"}, "scope": {"openid"},
		"redirect_uri": {aiaRedirect}, "kc_idp_hint": {f.idpAlias}, "kc_action": {action},
	}
	req, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, inNetIssuer+"/protocol/openid-connect/auth?"+q.Encode(), nil)
	_, login := do(t, c, req)
	m := formAction.FindSubmatch(login)
	if m == nil {
		m = formActionAlt.FindSubmatch(login)
	}
	if m == nil {
		t.Fatalf("no login form from the tenant IdP: %.400s", login)
	}
	post, _ := http.NewRequestWithContext(t.Context(), http.MethodPost, html.UnescapeString(string(m[1])),
		strings.NewReader(url.Values{"username": {user}, "password": {f.pass}}.Encode()))
	post.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	_, page := do(t, c, post)
	return c, string(page)
}

func pageSummary(page string) string {
	return strings.Join(strings.Fields(html.UnescapeString(pageText.ReplaceAllString(page, " "))), " ")
}

// parseForm finds the form with this id in the page and its inputs.
func parseForm(page, id string) (aiaForm, bool) {
	for _, loc := range formTag.FindAllStringIndex(page, -1) {
		tag := page[loc[0]:loc[1]]
		if !strings.Contains(tag, `id="`+id+`"`) {
			continue
		}
		act := attrAct.FindStringSubmatch(tag)
		if act == nil {
			continue
		}
		end := strings.Index(page[loc[1]:], "</form>")
		if end < 0 {
			end = len(page) - loc[1]
		}
		form := aiaForm{action: html.UnescapeString(act[1]), fields: map[string]formField{}}
		for _, in := range inputTag.FindAllString(page[loc[1]:loc[1]+end], -1) {
			n := attrName.FindStringSubmatch(in)
			if n == nil {
				continue
			}
			if ty := attrType.FindStringSubmatch(in); ty != nil && ty[1] == "submit" {
				continue
			}
			ff := formField{locked: attrLock.MatchString(in)}
			if v := attrVal.FindStringSubmatch(in); v != nil {
				ff.value = html.UnescapeString(v[1])
			}
			form.fields[n[1]] = ff
		}
		return form, true
	}
	return aiaForm{}, false
}

// requireForm signs in with kc_action=action and fails the test if Keycloak
// does not offer the form with this id.
func requireForm(t *testing.T, f tenantFx, user, action, formID string) (*http.Client, aiaForm) {
	t.Helper()
	c, page := aiaSignIn(t, f, user, action)
	form, ok := parseForm(page, formID)
	if !ok {
		t.Fatalf("kc_action=%s offered no form %q; last page: %.600s", action, formID, pageSummary(page))
	}
	return c, form
}

// submit posts the form as a browser would (every unlocked field with its
// shown value), with forged overriding or adding fields, and returns the
// status and the text of the page it gets back.
func (fm aiaForm) submit(t *testing.T, c *http.Client, forged url.Values) (int, string) {
	t.Helper()
	v := url.Values{}
	for n, ff := range fm.fields {
		if !ff.locked {
			v.Set(n, ff.value)
		}
	}
	for n, vals := range forged {
		v[n] = vals
	}
	post, _ := http.NewRequestWithContext(t.Context(), http.MethodPost, fm.action, strings.NewReader(v.Encode()))
	post.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	code, b := do(t, c, post)
	return code, pageSummary(string(b))
}

// requireNotEditable fails if the form offers email or the verified attribute
// for editing.
func (fm aiaForm) requireNotEditable(t *testing.T) {
	t.Helper()
	if ff, ok := fm.fields["email"]; ok && !ff.locked {
		t.Error("the update-profile form offers email for editing")
	}
	if ff, ok := fm.fields[keycloak.EmailVerifiedAttribute]; ok && !ff.locked {
		t.Errorf("the update-profile form offers %s for editing", keycloak.EmailVerifiedAttribute)
	}
}

// TestIdPEmailVerifiedAttributeNotUserEditable: through the UPDATE_PROFILE
// required action at nava-web, a signed-in user forges
// nava_idp_email_verified=true. The form does not offer the attribute, the
// forged value is rejected or ignored, the admin API still shows the IdP's
// value, and group sync does not place them. The same tenant and IdP place the
// control invitee. If the form is not offered, the test fails.
func TestIdPEmailVerifiedAttributeNotUserEditable(t *testing.T) {
	stopGroupsync(t)
	f := onboardFx(t)
	ctl := f.control(t)
	setUpstreamEmailVerified(t, f.realm, f.admin, false)

	c, form := requireForm(t, f, f.admin, "UPDATE_PROFILE", "kc-update-profile-form")
	form.requireNotEditable(t)
	code, page := form.submit(t, c, url.Values{keycloak.EmailVerifiedAttribute: {"true"}})
	t.Logf("UPDATE_PROFILE with a forged %s: %d %.200s", keycloak.EmailVerifiedAttribute, code, page)

	if got := storedEmailVerified(t, f.admin); got != "false" {
		t.Errorf("after the forged update %s = %q, want the IdP's \"false\"", keycloak.EmailVerifiedAttribute, got)
	}
	groupsyncOnce(t)
	if f.placed(t, f.admin) {
		t.Error("a user who forged the attribute was placed")
	}
	if got := inviteState(t, f.tenant, f.admin); got != "pending" {
		t.Errorf("invite = %q, want pending", got)
	}
	f.requirePlaced(t, ctl)
}

// TestUserCannotRenameEmailToPendingInvite: Mallory, a verified user of the
// tenant's own IdP, tries to rename her email to the pending admin address
// before the admin signs in, through UPDATE_PROFILE and then UPDATE_EMAIL at
// nava-web. The profile makes email admin-only, so her email is unchanged and
// nobody holds the admin address. After a sync cycle the invite is still
// pending and nobody is placed under it; then the real admin signs in and is
// placed. If the update-profile form is not offered, the test fails.
func TestUserCannotRenameEmailToPendingInvite(t *testing.T) {
	stopGroupsync(t)
	f := onboardFx(t)
	mallory := "mallory-" + uuid.NewString()[:8] + "@" + f.domain
	seedIdPUser(t, f.realm, mallory, mallory, f.pass)
	deleteUserAfter(t, mallory)

	assertUntouched := func(step string) {
		t.Helper()
		if kcUserByEmail(t, mallory) == "" {
			t.Errorf("%s: Mallory's email changed", step)
		}
		if kcUserByEmail(t, f.admin) != "" {
			t.Fatalf("%s: a user holds the pending admin address %s", step, f.admin)
		}
	}

	c, form := requireForm(t, f, mallory, "UPDATE_PROFILE", "kc-update-profile-form")
	form.requireNotEditable(t)
	code, page := form.submit(t, c, url.Values{"email": {f.admin}})
	t.Logf("UPDATE_PROFILE with email=%s: %d %.200s", f.admin, code, page)
	assertUntouched("UPDATE_PROFILE")

	// UPDATE_EMAIL is off unless the update-email feature is on: either the
	// action is not offered, or it changes nothing. Both are fine.
	uc, after := aiaSignIn(t, f, mallory, "UPDATE_EMAIL")
	if ef, ok := parseForm(after, "kc-update-email-form"); ok {
		code, page := ef.submit(t, uc, url.Values{"email": {f.admin}})
		t.Logf("UPDATE_EMAIL with email=%s: %d %.200s", f.admin, code, page)
	} else {
		t.Logf("UPDATE_EMAIL offered no form: %.200s", pageSummary(after))
	}
	assertUntouched("UPDATE_EMAIL")

	groupsyncOnce(t)
	if f.placed(t, mallory) || f.placed(t, f.admin) {
		t.Error("someone was placed through the renamed email")
	}
	if got := inviteState(t, f.tenant, f.admin); got != "pending" {
		t.Errorf("invite = %q, want pending", got)
	}

	// Positive control: the real admin signs in and is placed.
	brokeredSignIn(t, f.idpAlias, f.admin, f.pass)
	groupsyncOnce(t)
	f.requirePlaced(t, f.admin)
}

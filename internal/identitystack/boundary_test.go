//go:build identitystack

package identitystack

// Identity-boundary cases of T26 (TICK-82, PLAN-001 D-P9, Epic 03) that need
// the real Keycloak. Each one is registered in the isolation suite
// (internal/tenancy/isolation, StackPaths), which checks that it still exists.
// The attacker holds a valid login, or a confidential client's secret, and
// tries to change what decides their tenant, role or groups, or to turn a
// token into one for another tenant.

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/identity"
)

// anyForm is the first form of a Keycloak login page. With organizations on,
// the browser flow asks for the username first and the password on a second
// page.
var anyForm = regexp.MustCompile(`<form[^>]*action="([^"]+)"`)

// accountConsoleToken signs user in to Keycloak's own account-console client
// through the browser flow (authorization code with PKCE), as the user would
// to reach the account console. The token is the one the account REST API
// accepts; a nava-web token is refused there (401), so it would prove nothing.
func accountConsoleToken(t *testing.T, user, pass string) string {
	t.Helper()
	c := newClient(t)
	redirect := baseURL() + "/realms/nava/account/"
	var final *url.URL
	c.CheckRedirect = func(req *http.Request, _ []*http.Request) error {
		if strings.HasPrefix(req.URL.String(), redirect) && req.URL.Query().Get("code") != "" {
			final = req.URL
			return http.ErrUseLastResponse
		}
		return nil
	}
	verifier := rand.Text() + rand.Text()
	sum := sha256.Sum256([]byte(verifier))
	q := url.Values{
		"client_id": {"account-console"}, "response_type": {"code"}, "scope": {"openid"},
		"redirect_uri": {redirect}, "code_challenge_method": {"S256"},
		"code_challenge": {base64.RawURLEncoding.EncodeToString(sum[:])},
	}
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet,
		baseURL()+"/realms/nava/protocol/openid-connect/auth?"+q.Encode(), nil)
	code, page := do(t, c, req)
	for step := 0; final == nil && step < 3; step++ {
		m := anyForm.FindSubmatch(page)
		if code != http.StatusOK || m == nil {
			t.Fatalf("account console login, step %d: %d %.400s", step, code, page)
		}
		form := url.Values{"username": {user}}
		if strings.Contains(string(page), `name="password"`) {
			form.Set("password", pass)
		}
		post, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, html.UnescapeString(string(m[1])),
			strings.NewReader(form.Encode()))
		post.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		code, page = do(t, c, post)
	}
	if final == nil {
		t.Fatalf("account console login did not return a code: %d %.400s", code, page)
	}
	status, out := tokenForm(t, "nava", url.Values{
		"grant_type": {"authorization_code"}, "client_id": {"account-console"}, "code": {final.Query().Get("code")},
		"redirect_uri": {redirect}, "code_verifier": {verifier},
	})
	if status != http.StatusOK {
		t.Fatalf("account console code exchange: %d %v", status, out)
	}
	return out["access_token"].(string)
}

// account calls the account REST API of realm nava with tok.
func account(t *testing.T, tok, method, path string, body any) (int, []byte) {
	t.Helper()
	var rdr *strings.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rdr = strings.NewReader(string(b))
	} else {
		rdr = strings.NewReader("")
	}
	req, _ := http.NewRequestWithContext(context.Background(), method, baseURL()+"/realms/nava/account"+path, rdr)
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return do(t, newClient(t), req)
}

// identityState is everything Keycloak holds that decides a user's tenant,
// roles and groups: organization memberships, realm and client role mappings,
// group memberships, federated identities, and the custom attributes (none
// feed a claim today; a new one must not appear through self-service).
type identityState struct {
	Orgs, Roles, Groups, Links, Attributes string
}

func stateOf(t *testing.T, userID string) identityState {
	t.Helper()
	get := func(path string) string {
		code, b := adminRealm(t, "nava", http.MethodGet, path, nil)
		if code != http.StatusOK {
			t.Fatalf("GET %s: %d %s", path, code, b)
		}
		return string(b)
	}
	var u struct {
		Attributes map[string]any `json:"attributes"`
	}
	_ = json.Unmarshal([]byte(get("/users/"+userID)), &u)
	return identityState{
		Orgs:       get("/organizations/members/" + userID + "/organizations?briefRepresentation=true"),
		Roles:      get("/users/" + userID + "/role-mappings"),
		Groups:     get("/users/" + userID + "/groups"),
		Links:      get("/users/" + userID + "/federated-identity"),
		Attributes: canon(u.Attributes),
	}
}

// claimView is the part of an access token that decides tenant, role and
// group.
func claimView(t *testing.T, tok string) string {
	t.Helper()
	c := claims(t, tok)
	return canon(map[string]any{
		"sub": c["sub"], "organization": c["organization"], "realm_access": c["realm_access"],
		"resource_access": c["resource_access"], "groups": c["groups"], "idp": c["idp"],
	})
}

// tenantClaimNames are the claims that decide tenant, role or group, and any
// name a forged attribute might try to pass for one.
func tenantClaimNames() []string {
	return []string{
		"organization", "realm_access", "resource_access", "groups", "idp", "roles",
		"tenant", "tenant_id", "org", "org_id", "kc.org", "nava-operator",
	}
}

// selfServiceUser creates a tenant A member the way the platform does:
// through the admin API (InviteUser), so the realm's default roles apply and
// the user holds account/manage-account, as every invited or brokered user
// does (the seeded users of the import do not). It is a member of
// organization tenant-a, so its nava-web token resolves to tenant A. Removed
// after the test.
func selfServiceUser(t *testing.T) (id, user, pass string) {
	t.Helper()
	user, pass = "selfservice-"+strings.ToLower(rand.Text()[:8])+"@tenant-a.test", "dev-only-"+rand.Text()
	code, b := adminRealm(t, "nava", http.MethodPost, "/users", map[string]any{
		"username": user, "email": user, "emailVerified": true, "enabled": true,
		"firstName": "Self", "lastName": "Service",
		"credentials": []map[string]any{{"type": "password", "value": pass, "temporary": false}},
	})
	if code != http.StatusCreated {
		t.Fatalf("create %s: %d %s", user, code, b)
	}
	id = userID(t, serviceToken(t), user)
	t.Cleanup(func() { adminRealm(t, "nava", http.MethodDelete, "/users/"+id, nil) })
	org := orgID(t, masterToken(t), "tenant-a")
	if code, b := adminRealm(t, "nava", http.MethodPost, "/organizations/"+org+"/members", id); code != http.StatusCreated {
		t.Fatalf("add %s to tenant-a: %d %s", user, code, b)
	}
	return id, user, pass
}

// TestSelfServiceCannotChangeTenantRoleOrGroupAttributes (T26 case 3): a
// tenant A member created as the platform creates users signs in to the
// account console (Keycloak's account-console client, browser flow) and uses
// the account REST API to write every attribute and field that could pass for
// tenant, role or group, to move its email into tenant B's domain, and to
// write the group, organization, role and linked-account views; it tries the
// same with its nava-web token. The realm keeps the account API closed to
// both tokens today; the probes also run against it opened (checked by hand
// on Keycloak 26.8.1 with account-console's default scopes restored: the
// forged attributes are dropped, the username is read-only, the role and
// group fields are rejected). Whatever Keycloak answers, its organizations, role mappings, groups, federated
// links and attributes stay as they were, and a fresh nava-web token carries
// the same tenant, role and group claims. The realm's user profile and
// mappers are pinned too, so a later attribute a user may edit (at the
// account console or the update-profile login step) cannot feed a claim.
func TestSelfServiceCannotChangeTenantRoleOrGroupAttributes(t *testing.T) {
	id, user, pass := selfServiceUser(t)
	before := stateOf(t, id)
	tokBefore := claimView(t, passwordToken(t, "nava-web", user, pass))
	if !strings.Contains(tokBefore, "tenant-a") {
		t.Fatalf("the new member's token is not tenant A's: %s", tokBefore)
	}

	tokens := map[string]string{
		"nava-web":        passwordToken(t, "nava-web", user, pass),
		"account-console": accountConsoleToken(t, user, pass),
	}
	// The profile the probes edit: the member's own, as the admin reads it.
	var rep map[string]any
	_, ub := adminRealm(t, "nava", http.MethodGet, "/users/"+id, nil)
	if err := json.Unmarshal(ub, &rep); err != nil {
		t.Fatal(err)
	}
	profile := map[string]any{}
	for _, k := range []string{"username", "email", "firstName", "lastName", "attributes"} {
		if v, ok := rep[k]; ok {
			profile[k] = v
		}
	}
	// Today the account API refuses both tokens: account-console has no
	// client scopes, so its token carries no account audience or role. That
	// is a realm setting, not a guarantee, so the probes run either way; an
	// answer other than 200, 401 or 403 means the request was malformed and
	// would prove nothing.
	open := false
	for client, tok := range tokens {
		code, b := account(t, tok, http.MethodGet, "/", nil)
		switch code {
		case http.StatusOK:
			open = true
		case http.StatusUnauthorized, http.StatusForbidden:
		default:
			t.Fatalf("GET /account with a %s token: %d %.300s, want 200, 401 or 403", client, code, b)
		}
		t.Logf("GET /account with a %s token: %d", client, code)
	}
	t.Logf("account API open to the member: %v", open)

	probes := map[string]map[string]any{
		"email in tenant B":      {"email": "alice@tenant-b.test"},
		"username of tenant B":   {"username": "bob@tenant-b.test"},
		"top-level claim fields": {"realmRoles": []string{"nava-operator", "tenant-admin"}, "groups": []string{"/t/tenant-b"}, "organizations": []string{"tenant-b"}},
	}
	// One attribute per request, so a name Keycloak refuses cannot hide one
	// it would store.
	for _, n := range tenantClaimNames() {
		probes["attribute "+n] = map[string]any{"attributes": map[string]any{n: []string{"tenant-b"}}}
	}
	probes["attribute organization (claim JSON)"] = map[string]any{"attributes": map[string]any{
		"organization": []string{`{"tenant-b":{"id":"00000000-0000-4000-8000-0000000000b1"}}`}}}
	for client, tok := range tokens {
		for name, change := range probes {
			body := map[string]any{}
			for k, v := range profile {
				body[k] = v
			}
			for k, v := range change {
				body[k] = v
			}
			code, b := account(t, tok, http.MethodPost, "/", body)
			t.Logf("%s token: POST /account (%s): %d %.200s", client, name, code, b)
		}
		// The views of groups, organizations, roles and linked accounts take
		// no writes.
		for _, path := range []string{"/groups", "/organizations", "/roles", "/linked-accounts/idp-tenant-b"} {
			for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
				payload := []map[string]any{{"name": "nava-operator", "path": "/t/tenant-b", "alias": "tenant-b"}}
				if code, b := account(t, tok, method, path, payload); code < 400 {
					t.Errorf("%s token: %s /account%s = %d %.200s, want a refusal", client, method, path, code, b)
				}
			}
		}
	}

	if after := stateOf(t, id); after != before {
		t.Errorf("self-service changed the member's tenant, role or group state:\nbefore %+v\nafter  %+v", before, after)
	}
	// The email probe may succeed: email is a profile field, and nothing
	// maps it to a tenant. The claims must not move.
	if got := claimView(t, passwordToken(t, "nava-web", user, pass)); got != tokBefore {
		t.Errorf("tenant, role or group claims changed after self-service:\nbefore %s\nafter  %s", tokBefore, got)
	}

	assertNoUserEditableClaimSource(t)
}

// assertNoUserEditableClaimSource pins the realm: unmanaged attributes are
// off, a user may edit only the display fields, and no attribute or property
// mapper emits a claim that decides tenant, role or group.
func assertNoUserEditableClaimSource(t *testing.T) {
	t.Helper()
	code, b := adminRealm(t, "nava", http.MethodGet, "/users/profile", nil)
	var profile struct {
		UnmanagedAttributePolicy string `json:"unmanagedAttributePolicy"`
		Attributes               []struct {
			Name        string `json:"name"`
			Permissions struct {
				Edit []string `json:"edit"`
			} `json:"permissions"`
		} `json:"attributes"`
	}
	if code != http.StatusOK || json.Unmarshal(b, &profile) != nil || len(profile.Attributes) == 0 {
		t.Fatalf("user profile: %d %s", code, b)
	}
	if p := profile.UnmanagedAttributePolicy; p == "ENABLED" {
		t.Errorf("unmanagedAttributePolicy = %s: users may write attributes the profile does not declare", p)
	}
	display := []string{"username", "email", "firstName", "lastName"}
	for _, a := range profile.Attributes {
		if slices.Contains(a.Permissions.Edit, "user") && !slices.Contains(display, a.Name) {
			t.Errorf("user profile attribute %q is user-editable; only %v may be", a.Name, display)
		}
	}

	// Every mapper of every client scope and client: a user-model attribute
	// or property mapper must not emit a tenant, role or group claim.
	var mappers []map[string]any
	_, sb := adminRealm(t, "nava", http.MethodGet, "/client-scopes", nil)
	var scopes []struct {
		Name            string           `json:"name"`
		ProtocolMappers []map[string]any `json:"protocolMappers"`
	}
	_ = json.Unmarshal(sb, &scopes)
	for _, s := range scopes {
		mappers = append(mappers, s.ProtocolMappers...)
	}
	_, cb := adminRealm(t, "nava", http.MethodGet, "/clients", nil)
	var clients []struct {
		ClientID        string           `json:"clientId"`
		ProtocolMappers []map[string]any `json:"protocolMappers"`
	}
	_ = json.Unmarshal(cb, &clients)
	for _, c := range clients {
		mappers = append(mappers, c.ProtocolMappers...)
	}
	if len(mappers) == 0 {
		t.Fatal("no protocol mappers read: the check would pass vacuously")
	}
	for _, m := range mappers {
		kind, _ := m["protocolMapper"].(string)
		if kind != "oidc-usermodel-attribute-mapper" && kind != "oidc-usermodel-property-mapper" {
			continue
		}
		cfg, _ := m["config"].(map[string]any)
		claim, _ := cfg["claim.name"].(string)
		root := strings.SplitN(claim, ".", 2)[0]
		if slices.Contains(tenantClaimNames(), root) || slices.Contains(tenantClaimNames(), claim) {
			t.Errorf("mapper %v (%s) emits claim %q from a user attribute", m["name"], kind, claim)
		}
	}
}

// TestServiceAccountTokenCannotActInAnyTenant (T26 case 4, as groomed): every
// client with a service account (nava-platform, nava-onboarding, and any
// added later) gets a client-credentials token. Asking for the organization
// scope, an organization, or the nava-api audience never yields an
// organization claim or the nava-api audience, and the T15 adapter refuses
// the token even with a store that would bind any organization to tenant B
// and accept any principal: the token alone is what stops it.
func TestServiceAccountTokenCannotActInAnyTenant(t *testing.T) {
	secrets := map[string]string{
		"nava-platform":   "dev-only-platform-secret",
		"nava-onboarding": "dev-only-onboarding-secret",
	}
	_, cb := adminRealm(t, "nava", http.MethodGet, "/clients", nil)
	var clients []struct {
		ID                     string `json:"id"`
		ClientID               string `json:"clientId"`
		ServiceAccountsEnabled bool   `json:"serviceAccountsEnabled"`
	}
	if err := json.Unmarshal(cb, &clients); err != nil {
		t.Fatal(err)
	}
	var withSA []string
	for _, c := range clients {
		if !c.ServiceAccountsEnabled {
			continue
		}
		withSA = append(withSA, c.ClientID)
		if _, ok := secrets[c.ClientID]; !ok {
			// A client added later: read its secret as the admin.
			_, sb := adminRealm(t, "nava", http.MethodGet, "/clients/"+c.ID+"/client-secret", nil)
			var s struct{ Value string }
			_ = json.Unmarshal(sb, &s)
			secrets[c.ClientID] = s.Value
		}
	}
	if !slices.Contains(withSA, "nava-platform") || !slices.Contains(withSA, "nava-onboarding") {
		t.Fatalf("service-account clients = %v, want nava-platform and nava-onboarding at least", withSA)
	}

	store := &anyTenantStore{tenant: uuid.MustParse("0b000000-0000-4000-8000-00000000000b")}
	for _, client := range withSA {
		t.Run(client, func(t *testing.T) {
			attempts := map[string]url.Values{
				"plain":              {},
				"organization scope": {"scope": {"organization"}},
				"named organization": {"scope": {"organization:tenant-b"}},
				"all organizations":  {"scope": {"organization:*"}},
				"nava-api audience":  {"audience": {"nava-api"}},
				"nava-api scope":     {"scope": {"nava-api-audience"}},
				"nava-api resource":  {"resource": {"nava-api"}},
			}
			issued := 0
			for name, extra := range attempts {
				form := url.Values{"grant_type": {"client_credentials"}, "client_id": {client}, "client_secret": {secrets[client]}}
				for k, v := range extra {
					form[k] = v
				}
				code, out := tokenForm(t, "nava", form)
				if code != http.StatusOK {
					continue // refused: no token
				}
				issued++
				tok, _ := out["access_token"].(string)
				c := claims(t, tok)
				if _, ok := c["organization"]; ok {
					t.Errorf("%s: client-credentials token carries organization %v", name, c["organization"])
				}
				if audHas(t, tok, "nava-api") {
					t.Errorf("%s: client-credentials token has audience nava-api: %v", name, c["aud"])
				}
				a := stackAdapterWithStore(t, tok, store)
				if p, err := a.Resolve(t.Context(), tok); err == nil {
					t.Errorf("%s: adapter resolved a service-account token to tenant %s", name, p.TenantID)
				} else if !errors.Is(err, identity.ErrInvalidToken) && !errors.Is(err, identity.ErrTenantUnresolved) {
					t.Errorf("%s: Resolve = %v, want a refusal", name, err)
				}
			}
			if issued == 0 {
				t.Errorf("no client-credentials token issued for %s: the adapter checks ran on nothing", client)
			}
		})
	}
	if store.bindings != 0 {
		t.Errorf("the adapter looked up %d bindings for service-account tokens; the token alone must stop it", store.bindings)
	}
}

// TestTokenExchangeStaysDisabled (T26 case 5): no client of the realm has
// standard token exchange on, and the server has no legacy token-exchange
// feature. Every client that can authenticate (the confidential ones with
// their secrets, the public ones without) is refused an exchange of alice's
// token: for itself, for nava-api, for nava-operator-api, and for another
// user (impersonation, legacy V1 only).
func TestTokenExchangeStaysDisabled(t *testing.T) {
	_, cb := adminRealm(t, "nava", http.MethodGet, "/clients", nil)
	var clients []struct {
		ID           string            `json:"id"`
		ClientID     string            `json:"clientId"`
		PublicClient bool              `json:"publicClient"`
		BearerOnly   bool              `json:"bearerOnly"`
		Attributes   map[string]string `json:"attributes"`
	}
	if err := json.Unmarshal(cb, &clients); err != nil || len(clients) == 0 {
		t.Fatalf("clients: %v %s", err, cb)
	}
	for _, c := range clients {
		if c.Attributes["standard.token.exchange.enabled"] == "true" {
			t.Errorf("client %s has standard token exchange enabled", c.ClientID)
		}
	}

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, baseURL()+"/admin/serverinfo", nil)
	req.Header.Set("Authorization", "Bearer "+masterToken(t))
	code, sb := do(t, newClient(t), req)
	var info struct {
		Features []struct {
			Name    string `json:"name"`
			Enabled bool   `json:"enabled"`
		} `json:"features"`
	}
	if code != http.StatusOK || json.Unmarshal(sb, &info) != nil || len(info.Features) == 0 {
		t.Fatalf("server info: %d %.300s", code, sb)
	}
	for _, f := range info.Features {
		// TOKEN_EXCHANGE is the legacy V1 (impersonation, cross-client
		// minting); the standard V2 feature is on by default and gated per
		// client above.
		if f.Name == "TOKEN_EXCHANGE" && f.Enabled {
			t.Error("the legacy TOKEN_EXCHANGE feature is enabled")
		}
	}

	subject := passwordToken(t, "nava-web", "alice@tenant-a.test", "dev-only-alice")
	bob := userID(t, serviceToken(t), "bob@tenant-b.test")
	secrets := map[string]string{
		"nava-platform":   "dev-only-platform-secret",
		"nava-onboarding": "dev-only-onboarding-secret",
	}
	tried := 0
	for _, c := range clients {
		if c.BearerOnly {
			continue
		}
		secret, known := secrets[c.ClientID]
		if !c.PublicClient && !known {
			_, b := adminRealm(t, "nava", http.MethodGet, "/clients/"+c.ID+"/client-secret", nil)
			var s struct{ Value string }
			_ = json.Unmarshal(b, &s)
			secret = s.Value
		}
		for name, extra := range map[string]url.Values{
			"self":              {},
			"nava-api":          {"audience": {"nava-api"}},
			"nava-operator-api": {"audience": {"nava-operator-api"}},
			"impersonate bob":   {"requested_subject": {bob}},
		} {
			form := url.Values{
				"grant_type":           {"urn:ietf:params:oauth:grant-type:token-exchange"},
				"client_id":            {c.ClientID},
				"subject_token":        {subject},
				"subject_token_type":   {"urn:ietf:params:oauth:token-type:access_token"},
				"requested_token_type": {"urn:ietf:params:oauth:token-type:access_token"},
			}
			if !c.PublicClient {
				form.Set("client_secret", secret)
			}
			for k, v := range extra {
				form[k] = v
			}
			tried++
			if code, out := tokenForm(t, "nava", form); code == http.StatusOK {
				tok, _ := out["access_token"].(string)
				t.Errorf("token exchange via %s (%s) succeeded: sub=%v aud=%v", c.ClientID, name, claims(t, tok)["sub"], auds(t, tok))
			}
		}
	}
	if tried < 8 {
		t.Errorf("tried %d exchanges, want every client at least: the check would pass vacuously", tried)
	}
}

// TestUserCannotSelfAssignOperatorRole (T26 case 7): a tenant user and a
// tenant admin, each with their own tokens (nava-web and account console),
// try every self-service route to nava-operator: the admin API role mapping
// and user update on themselves, joining a group, and the account API. Their
// role mappings are unchanged, an operator-client login gives them no
// nava-operator role, and the operator resolver refuses that token.
func TestUserCannotSelfAssignOperatorRole(t *testing.T) {
	code, rb := adminRealm(t, "nava", http.MethodGet, "/roles/nava-operator", nil)
	if code != http.StatusOK {
		t.Fatalf("read nava-operator: %d %s", code, rb)
	}
	var role map[string]any
	_ = json.Unmarshal(rb, &role)

	for _, u := range []struct{ user, pass string }{
		{"alice@tenant-a.test", "dev-only-alice"},
		{"admin@tenant-a.test", "dev-only-admin-a"},
	} {
		t.Run(u.user, func(t *testing.T) {
			id := userID(t, serviceToken(t), u.user)
			before := stateOf(t, id)
			for client, tok := range map[string]string{
				"nava-web":        passwordToken(t, "nava-web", u.user, u.pass),
				"account-console": accountConsoleToken(t, u.user, u.pass),
			} {
				for _, call := range []struct {
					method, path string
					body         any
				}{
					{http.MethodPost, "/users/" + id + "/role-mappings/realm", []map[string]any{role}},
					{http.MethodPut, "/users/" + id, map[string]any{"realmRoles": []string{"nava-operator"}}},
				} {
					if code, b := admin(t, tok, call.method, call.path, call.body); code != http.StatusUnauthorized && code != http.StatusForbidden {
						t.Errorf("%s token: %s %s = %d %.200s, want 401/403", client, call.method, call.path, code, b)
					}
				}
				if code, b := account(t, tok, http.MethodPost, "/roles", []map[string]any{{"name": "nava-operator"}}); code < 400 {
					t.Errorf("%s token: POST /account/roles = %d %.200s, want a refusal", client, code, b)
				}
			}
			if after := stateOf(t, id); after != before {
				t.Errorf("role or group state changed:\nbefore %+v\nafter  %+v", before, after)
			}
			// An operator-client login is where the role would show.
			code, out := tokenForm(t, "nava", url.Values{
				"grant_type": {"password"}, "client_id": {"nava-operator-cli"}, "username": {u.user}, "password": {u.pass},
			})
			if code != http.StatusOK {
				// Only the role-mapping and group state above ran.
				t.Logf("nava-operator-cli password grant for %s: %d %v; claim and ResolveOperator checks not run", u.user, code, out)
			} else {
				tok, _ := out["access_token"].(string)
				if strings.Contains(fmt.Sprint(claims(t, tok)["realm_access"]), "nava-operator") {
					t.Errorf("%s holds nava-operator after the attempts", u.user)
				}
				if _, err := stackAdapter(t, tok).ResolveOperator(t.Context(), tok); !errors.Is(err, identity.ErrInvalidToken) {
					t.Errorf("ResolveOperator(%s) = %v, want ErrInvalidToken", u.user, err)
				}
			}
		})
	}
}

// anyTenantStore binds every organization to one tenant and accepts every
// principal, enabled and with no federated links: the most permissive store.
// A token it cannot resolve is refused on its own claims.
type anyTenantStore struct {
	tenant   uuid.UUID
	bindings int
}

func (s *anyTenantStore) Binding(context.Context, string, string) (identity.Binding, error) {
	s.bindings++
	return identity.Binding{TenantID: s.tenant}, nil
}

func (s *anyTenantStore) Principal(context.Context, uuid.UUID, string) (identity.PrincipalRecord, error) {
	return identity.PrincipalRecord{ID: uuid.New(), Enabled: true, LinksKnown: true}, nil
}

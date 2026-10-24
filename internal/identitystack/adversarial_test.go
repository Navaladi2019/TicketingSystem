//go:build identitystack

package identitystack

// Adversarial tests for realm `nava` (T14 security review, ADR-001 A.1, D-P12).
// The attacker holds a valid login for one tenant, or controls one tenant's
// external IdP, and tries to reach another tenant or the operator surface.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// adminRealm calls the admin API of any realm with the master bootstrap admin.
func adminRealm(t *testing.T, realm, method, path string, body any) (int, []byte) {
	t.Helper()
	var rdr *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rdr = bytes.NewReader(b)
	} else {
		rdr = bytes.NewReader(nil)
	}
	req, _ := http.NewRequestWithContext(context.Background(), method, baseURL()+"/admin/realms/"+realm+path, rdr)
	req.Header.Set("Authorization", "Bearer "+masterToken(t))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return do(t, newClient(t), req)
}

// brokerAttempt runs the brokered code flow like brokeredToken, but reports a
// refused login instead of failing the test.
func brokerAttempt(t *testing.T, idp, user, pass string) (token string, page []byte) {
	t.Helper()
	c := newClient(t)
	redirect := "http://localhost:9/cb"
	var final *url.URL
	c.CheckRedirect = func(req *http.Request, _ []*http.Request) error {
		if strings.HasPrefix(req.URL.String(), redirect) {
			final = req.URL
			return http.ErrUseLastResponse
		}
		return nil
	}
	q := url.Values{
		"client_id": {"nava-web"}, "response_type": {"code"}, "scope": {"openid"},
		"redirect_uri": {redirect}, "kc_idp_hint": {idp},
	}
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet,
		baseURL()+"/realms/nava/protocol/openid-connect/auth?"+q.Encode(), nil)
	_, login := do(t, c, req)
	m := formAction.FindSubmatch(login)
	if m == nil {
		m = formActionAlt.FindSubmatch(login)
	}
	if m == nil {
		t.Fatalf("no login form from the IdP realm: %.400s", login)
	}
	post, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, html.UnescapeString(string(m[1])),
		strings.NewReader(url.Values{"username": {user}, "password": {pass}}.Encode()))
	post.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	_, page = do(t, c, post)
	if final == nil || final.Query().Get("code") == "" {
		return "", page
	}
	status, out := tokenForm(t, "nava", url.Values{
		"grant_type": {"authorization_code"}, "client_id": {"nava-web"}, "code": {final.Query().Get("code")}, "redirect_uri": {redirect},
	})
	if status != http.StatusOK {
		return "", page
	}
	return out["access_token"].(string), page
}

// seedIdPUser creates a user in a test IdP realm, as the tenant who controls
// that IdP could, and removes it (and any user it brokered into nava) after.
func seedIdPUser(t *testing.T, idpRealm, username, email, pass string) {
	t.Helper()
	code, b := adminRealm(t, idpRealm, http.MethodPost, "/users", map[string]any{
		"username": username, "email": email, "emailVerified": true, "enabled": true,
		"firstName": "Mal", "lastName": "Lory",
		"credentials": []map[string]any{{"type": "password", "value": pass, "temporary": false}},
	})
	if code != http.StatusCreated {
		t.Fatalf("seed %s in %s: %d %s", username, idpRealm, code, b)
	}
	t.Cleanup(func() {
		for _, realm := range []string{idpRealm, "nava"} {
			for _, q := range []string{"username=" + url.QueryEscape(username), "email=" + url.QueryEscape(email)} {
				_, b := adminRealm(t, realm, http.MethodGet, "/users?exact=true&"+q, nil)
				var us []struct{ ID, Username string }
				_ = json.Unmarshal(b, &us)
				for _, u := range us {
					if realm == "nava" && u.Username == "alice@tenant-a.test" {
						continue // never delete the seeded victim
					}
					adminRealm(t, realm, http.MethodDelete, "/users/"+u.ID, nil)
				}
			}
		}
	})
}

// A tenant-b IdP asserts the email (and username) of tenant-a's alice. A.1
// forbids linking: the login must be refused, alice must gain no federated
// identity, and no token for tenant-a may come out.
func TestBrokerCannotTakeOverUserOfAnotherTenant(t *testing.T) {
	for name, username := range map[string]string{
		"same email":          "mallory@tenant-b.test",
		"same email+username": "alice@tenant-a.test",
	} {
		t.Run(name, func(t *testing.T) {
			seedIdPUser(t, "idp-tenant-b", username, "alice@tenant-a.test", "dev-only-mallory")
			if tok, _ := brokerAttempt(t, "idp-tenant-b", username, "dev-only-mallory"); tok != "" {
				c := claims(t, tok)
				t.Fatalf("tenant-b IdP colliding with alice got a token: sub=%v organization=%v", c["sub"], c["organization"])
			}
			svc := serviceToken(t)
			id := userID(t, svc, "alice@tenant-a.test")
			code, b := admin(t, svc, http.MethodGet, "/users/"+id+"/federated-identity", nil)
			if code != http.StatusOK || strings.TrimSpace(string(b)) != "[]" {
				t.Fatalf("alice federated identities after the attempt: %d %s, want []", code, b)
			}
		})
	}
}

// A tenant-b IdP asserts a fresh address in tenant-a's verified domain. The
// user may exist, but only as a member of tenant-b: an IdP never places a
// user in another tenant's organization.
func TestBrokerForeignDomainEmailStaysInIdPTenant(t *testing.T) {
	seedIdPUser(t, "idp-tenant-b", "eve-b", "eve@tenant-a.test", "dev-only-eve")
	tok, _ := brokerAttempt(t, "idp-tenant-b", "eve-b", "dev-only-eve")
	if tok == "" {
		return // refused outright: also safe
	}
	c := claims(t, tok)
	org, _ := c["organization"].(map[string]any)
	if _, ok := org["tenant-a"]; ok {
		t.Fatalf("tenant-b IdP placed eve in tenant-a: organization=%v", c["organization"])
	}
	if c["idp"] != "idp-tenant-b" {
		t.Fatalf("idp = %v, want idp-tenant-b", c["idp"])
	}
}

// No tenant user may hold a token for audience nava-operator-api. D-P12's
// OperatorResolver is specified on that audience, so minting it for every
// realm user turns a role check that someone forgets into an operator login.
func TestOperatorAudienceOnlyForOperators(t *testing.T) {
	for _, u := range []struct{ user, pass string }{
		{"alice@tenant-a.test", "dev-only-alice"},
		{"admin@tenant-a.test", "dev-only-admin-a"},
		{"bob@tenant-b.test", "dev-only-bob"},
	} {
		code, out := tokenForm(t, "nava", url.Values{
			"grant_type": {"password"}, "client_id": {"nava-operator-cli"},
			"username": {u.user}, "password": {u.pass}, "scope": {"openid"},
		})
		if code != http.StatusOK {
			continue // refused: the wanted outcome
		}
		c := claims(t, out["access_token"].(string))
		if strings.Contains(fmt.Sprint(c["aud"]), "nava-operator-api") {
			t.Errorf("%s got aud=%v (realm_access=%v) from nava-operator-cli; only nava-operator holders may", u.user, c["aud"], c["realm_access"])
		}
	}
	// A tenant client cannot ask for the operator audience scope either.
	code, _ := tokenForm(t, "nava", url.Values{
		"grant_type": {"password"}, "client_id": {"nava-web"}, "username": {"alice@tenant-a.test"},
		"password": {"dev-only-alice"}, "scope": {"openid nava-operator-audience"},
	})
	if code == http.StatusOK {
		t.Error("nava-web issued a token for scope nava-operator-audience")
	}
}

// A user may not ask for another tenant's organization through the
// organization:<alias> scope parameter.
func TestOrganizationScopeCannotNameAnotherTenant(t *testing.T) {
	code, out := tokenForm(t, "nava", url.Values{
		"grant_type": {"password"}, "client_id": {"nava-web"}, "username": {"alice@tenant-a.test"},
		"password": {"dev-only-alice"}, "scope": {"openid organization:tenant-b"},
	})
	if code != http.StatusOK {
		return
	}
	c := claims(t, out["access_token"].(string))
	if org, ok := c["organization"].(map[string]any); ok {
		if _, leak := org["tenant-b"]; leak {
			t.Fatalf("alice got organization tenant-b: %v", org)
		}
	}
}

// Every indirect route to nava-operator is closed: no composite parent, not a
// default role, not mapped to a group, no IdP mapper that grants roles, no
// fine-grained admin permissions, and only the seeded operator holds it.
func TestNoIndirectPathToOperatorRole(t *testing.T) {
	_, b := adminRealm(t, "nava", http.MethodGet, "", nil)
	var realm struct {
		AdminPermissionsEnabled bool `json:"adminPermissionsEnabled"`
	}
	_ = json.Unmarshal(b, &realm)
	if realm.AdminPermissionsEnabled {
		t.Error("adminPermissionsEnabled = true: fine-grained admin permissions open a delegated path to role mapping")
	}

	_, b = adminRealm(t, "nava", http.MethodGet, "/roles?briefRepresentation=false", nil)
	var roles []struct {
		Name      string
		Composite bool
	}
	_ = json.Unmarshal(b, &roles)
	for _, r := range roles {
		if !r.Composite || r.Name == "nava-operator" {
			continue
		}
		_, cb := adminRealm(t, "nava", http.MethodGet, "/roles/"+url.PathEscape(r.Name)+"/composites", nil)
		if strings.Contains(string(cb), `"name":"nava-operator"`) {
			t.Errorf("realm role %s is a composite containing nava-operator", r.Name)
		}
	}

	_, b = adminRealm(t, "nava", http.MethodGet, "/roles/nava-operator/groups", nil)
	if strings.TrimSpace(string(b)) != "[]" {
		t.Errorf("nava-operator is mapped to groups: %s", b)
	}

	_, b = adminRealm(t, "nava", http.MethodGet, "/roles/nava-operator/users", nil)
	var holders []struct{ Username string }
	_ = json.Unmarshal(b, &holders)
	if len(holders) != 1 || holders[0].Username != "operator@nava.test" {
		t.Errorf("nava-operator holders = %+v, want only operator@nava.test", holders)
	}

	for _, idp := range []string{"idp-tenant-a", "idp-tenant-b"} {
		_, mb := adminRealm(t, "nava", http.MethodGet, "/identity-provider/instances/"+idp+"/mappers", nil)
		var ms []struct{ Name, IdentityProviderMapper string }
		_ = json.Unmarshal(mb, &ms)
		for _, m := range ms {
			if strings.Contains(m.IdentityProviderMapper, "role") {
				t.Errorf("IdP %s mapper %s (%s) grants roles from a tenant-controlled IdP", idp, m.Name, m.IdentityProviderMapper)
			}
		}
	}
}

// The platform service account maps nava-operator, but cannot raise itself to
// realm-admin or hand out realm-management roles it does not hold.
func TestPlatformServiceAccountCannotEscalate(t *testing.T) {
	svc := serviceToken(t)
	sa := userID(t, svc, "service-account-nava-platform")
	_, b := adminRealm(t, "nava", http.MethodGet, "/clients?clientId=realm-management", nil)
	var cs []struct{ ID string }
	if json.Unmarshal(b, &cs) != nil || len(cs) != 1 {
		t.Fatalf("realm-management client: %s", b)
	}
	for _, role := range []string{"realm-admin", "manage-clients", "manage-authorization"} {
		_, rb := adminRealm(t, "nava", http.MethodGet, "/clients/"+cs[0].ID+"/roles/"+role, nil)
		var rep map[string]any
		_ = json.Unmarshal(rb, &rep)
		for _, target := range []string{sa, userID(t, svc, "admin@tenant-a.test")} {
			code, body := admin(t, svc, http.MethodPost, "/users/"+target+"/role-mappings/clients/"+cs[0].ID, []map[string]any{rep})
			if code < 400 {
				adminRealm(t, "nava", http.MethodDelete, "/users/"+target+"/role-mappings/clients/"+cs[0].ID, []map[string]any{rep})
				t.Errorf("service account granted %s to %s: %d %s", role, target, code, body)
			}
		}
	}
}

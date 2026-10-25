//go:build identitystack

package identitystack

import (
	"encoding/json"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"testing"
)

// Adversarial audience tests (security review of PR #89, TICK-151). The
// attacker holds tenant credentials for nava-web and tries every request
// shape the token endpoint offers to get an access token whose aud contains
// nava-operator-api, or to get any token at all from the bearer-only
// audience clients.

// auds returns the aud claim as a list, whether Keycloak sent a string or an
// array, so membership is checked by exact element and not by substring.
func auds(t *testing.T, tok string) []string {
	t.Helper()
	switch v := claims(t, tok)["aud"].(type) {
	case string:
		return []string{v}
	case []any:
		out := make([]string, 0, len(v))
		for _, e := range v {
			s, _ := e.(string)
			out = append(out, s)
		}
		return out
	default:
		return nil
	}
}

// TestTenantCannotRequestOperatorAudience: nava-web asks for the operator
// audience scope, the audience parameter and a resource indicator. Keycloak
// may refuse the request or ignore the extra parameters; it must never issue
// a token whose aud contains nava-operator-api.
func TestTenantCannotRequestOperatorAudience(t *testing.T) {
	base := url.Values{
		"grant_type": {"password"}, "client_id": {"nava-web"},
		"username": {"alice@tenant-a.test"}, "password": {"dev-only-alice"},
	}
	attempts := map[string]url.Values{
		"scope":               {"scope": {"openid nava-operator-audience"}},
		"scope-and-roles":     {"scope": {"openid roles nava-operator-audience"}},
		"audience-param":      {"scope": {"openid"}, "audience": {"nava-operator-api"}},
		"resource-param":      {"scope": {"openid"}, "resource": {"nava-operator-api"}},
		"operator-scope-only": {"scope": {"nava-operator-audience"}},
	}
	for name, extra := range attempts {
		t.Run(name, func(t *testing.T) {
			form := url.Values{}
			for k, v := range base {
				form[k] = v
			}
			for k, v := range extra {
				form[k] = v
			}
			code, out := tokenForm(t, "nava", form)
			if code != http.StatusOK {
				return // refused: no token, nothing to leak
			}
			tok, _ := out["access_token"].(string)
			if a := auds(t, tok); slices.Contains(a, "nava-operator-api") {
				t.Errorf("nava-web token aud = %v, contains nava-operator-api", a)
			}
		})
	}
}

// TestTenantCannotExchangeForOperatorAudience: alice exchanges her nava-web
// token for one aimed at nava-operator-api (standard token exchange, enabled
// by default since Keycloak 26.2 but only for confidential clients that opt
// in). nava-web is public and has not opted in, so the exchange must fail or,
// at the very least, not yield the operator audience.
func TestTenantCannotExchangeForOperatorAudience(t *testing.T) {
	subject := passwordToken(t, "nava-web", "alice@tenant-a.test", "dev-only-alice")
	for _, client := range []string{"nava-web", "nava-operator-cli"} {
		t.Run(client, func(t *testing.T) {
			code, out := tokenForm(t, "nava", url.Values{
				"grant_type":           {"urn:ietf:params:oauth:grant-type:token-exchange"},
				"client_id":            {client},
				"subject_token":        {subject},
				"subject_token_type":   {"urn:ietf:params:oauth:token-type:access_token"},
				"requested_token_type": {"urn:ietf:params:oauth:token-type:access_token"},
				"audience":             {"nava-operator-api"},
			})
			if code == http.StatusOK {
				tok, _ := out["access_token"].(string)
				t.Errorf("token exchange via %s succeeded, aud = %v", client, auds(t, tok))
			}
		})
	}
}

// TestBearerOnlyAudienceClientsIssueNothing: nava-api and nava-operator-api
// are audiences, not clients anyone signs in with. No grant may issue a token
// for them, with or without a guessed secret.
func TestBearerOnlyAudienceClientsIssueNothing(t *testing.T) {
	for _, client := range []string{"nava-api", "nava-operator-api"} {
		for name, form := range map[string]url.Values{
			"password": {"grant_type": {"password"}, "client_id": {client},
				"username": {"alice@tenant-a.test"}, "password": {"dev-only-alice"}, "scope": {"openid"}},
			"password-operator": {"grant_type": {"password"}, "client_id": {client},
				"username": {"operator@nava.test"}, "password": {"dev-only-operator"}, "scope": {"openid"}},
			"client-credentials": {"grant_type": {"client_credentials"}, "client_id": {client}},
			"client-credentials-guess": {"grant_type": {"client_credentials"}, "client_id": {client},
				"client_secret": {"secret"}},
		} {
			t.Run(client+"/"+name, func(t *testing.T) {
				code, out := tokenForm(t, "nava", form)
				if code == http.StatusOK || out["access_token"] != nil {
					t.Errorf("%s issued a token via %s: %d", client, name, code)
				}
			})
		}
	}
}

// TestOperatorTokenAudienceExact: the operator token names nava-operator-api
// and not nava-api, compared by element (the PR #89 check compared substrings
// and could not fail when both were present).
func TestOperatorTokenAudienceExact(t *testing.T) {
	a := auds(t, passwordToken(t, "nava-operator-cli", "operator@nava.test", "dev-only-operator"))
	if !slices.Contains(a, "nava-operator-api") || slices.Contains(a, "nava-api") {
		t.Errorf("operator aud = %v, want nava-operator-api and not nava-api", a)
	}
	for _, u := range [][2]string{
		{"alice@tenant-a.test", "dev-only-alice"},
		{"admin@tenant-a.test", "dev-only-admin-a"},
		{"bob@tenant-b.test", "dev-only-bob"},
	} {
		a := auds(t, passwordToken(t, "nava-web", u[0], u[1]))
		if !slices.Contains(a, "nava-api") || slices.Contains(a, "nava-operator-api") {
			t.Errorf("%s aud = %v, want nava-api and not nava-operator-api", u[0], a)
		}
	}
}

// TestAudienceClientsHoldNoGrantSurface pins, on the running realm, what
// keeps the two audience clients and nava-web from minting the operator
// audience by configuration: the audience clients are bearer-only with no
// service account and no client roles (the roles scope's audience-resolve
// mapper adds aud for any client whose roles land in a token), and no
// client involved has opted in to standard token exchange.
func TestAudienceClientsHoldNoGrantSurface(t *testing.T) {
	type client struct {
		ID                     string            `json:"id"`
		ClientID               string            `json:"clientId"`
		BearerOnly             bool              `json:"bearerOnly"`
		PublicClient           bool              `json:"publicClient"`
		ServiceAccountsEnabled bool              `json:"serviceAccountsEnabled"`
		FullScopeAllowed       bool              `json:"fullScopeAllowed"`
		Attributes             map[string]string `json:"attributes"`
	}
	get := func(clientID string) client {
		t.Helper()
		_, b := adminRealm(t, "nava", http.MethodGet, "/clients?clientId="+url.QueryEscape(clientID), nil)
		var cs []client
		if err := json.Unmarshal(b, &cs); err != nil || len(cs) != 1 {
			t.Fatalf("client %s: %v %s", clientID, err, b)
		}
		return cs[0]
	}
	for _, id := range []string{"nava-api", "nava-operator-api"} {
		c := get(id)
		if !c.BearerOnly || c.PublicClient || c.ServiceAccountsEnabled || c.FullScopeAllowed {
			t.Errorf("%s: bearerOnly=%v public=%v serviceAccounts=%v fullScope=%v", id, c.BearerOnly, c.PublicClient, c.ServiceAccountsEnabled, c.FullScopeAllowed)
		}
		_, rb := adminRealm(t, "nava", http.MethodGet, "/clients/"+c.ID+"/roles", nil)
		if strings.TrimSpace(string(rb)) != "[]" {
			t.Errorf("%s defines client roles %s: a scope mapping to one would add its aud via audience-resolve", id, rb)
		}
	}
	for _, id := range []string{"nava-web", "nava-api", "nava-operator-api", "nava-operator-cli"} {
		c := get(id)
		if c.Attributes["standard.token.exchange.enabled"] == "true" {
			t.Errorf("%s has standard token exchange enabled", id)
		}
	}
	web := get("nava-web")
	if web.FullScopeAllowed {
		t.Error("nava-web fullScopeAllowed = true: every role of the user, and its client's aud, would enter the token")
	}
	_, ob := adminRealm(t, "nava", http.MethodGet, "/clients/"+web.ID+"/optional-client-scopes", nil)
	if strings.Contains(string(ob), "nava-operator-audience") {
		t.Errorf("nava-web optional scopes include nava-operator-audience: %s", ob)
	}
	_, db := adminRealm(t, "nava", http.MethodGet, "/clients/"+web.ID+"/default-client-scopes", nil)
	if strings.Contains(string(db), "nava-operator-audience") || !strings.Contains(string(db), `"nava-api-audience"`) {
		t.Errorf("nava-web default scopes = %s, want nava-api-audience and not nava-operator-audience", db)
	}
}

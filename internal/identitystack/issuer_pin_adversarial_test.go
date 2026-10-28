//go:build identitystack

package identitystack

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// Adversarial issuer tests (TICK-160, deferred from the security review of
// PR #72 on TICK-156). The attacker reaches Keycloak's published port and
// sends the token request with a Host, X-Forwarded-Host or Forwarded header
// of their choosing, so that Keycloak names the token's issuer (and the
// redirect base) after the attacker's host. Whatever Keycloak issues, the
// running nava-core must accept only tokens whose iss is the one pinned
// issuer, and refuse the rest with 401 invalid_token.

// forgedHeaders are the request shapes that move Keycloak's issuer when the
// hostname is not pinned or proxy headers are trusted from anyone.
type forgery struct {
	host   string // "" keeps the real address
	header http.Header
}

func forgedHeaders() map[string]forgery {
	return map[string]forgery{
		"host":                  {host: "evil.example"},
		"host-with-port":        {host: "evil.example:8080"},
		"host-suffix-of-pinned": {host: "keycloak:8080.evil.example"},
		"x-forwarded-host": {header: http.Header{
			"X-Forwarded-Host": {"evil.example"}, "X-Forwarded-Proto": {"https"}, "X-Forwarded-Port": {"443"}}},
		"forwarded": {header: http.Header{"Forwarded": {"host=evil.example;proto=https"}}},
		"host-and-x-forwarded-host": {host: "evil.example", header: http.Header{
			"X-Forwarded-Host": {"evil.example"}}},
	}
}

// pinnedIssuer is nava-core's NAVA_OIDC_ISSUERS in compose.yaml. The Done
// list asks for exactly one value; a list would let a second, request-derived
// issuer through.
func pinnedIssuer(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "compose.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var c struct {
		Services map[string]struct {
			Environment map[string]string `yaml:"environment"`
		} `yaml:"services"`
	}
	if err := yaml.Unmarshal(raw, &c); err != nil {
		t.Fatal(err)
	}
	v := strings.TrimSpace(composeExpand(c.Services["nava-core"].Environment["NAVA_OIDC_ISSUERS"]))
	if v == "" || strings.Contains(v, ",") {
		t.Fatalf("nava-core NAVA_OIDC_ISSUERS = %q, want exactly one issuer", v)
	}
	return v
}

// composeDefault is ${VAR:-default}, the only interpolation compose.yaml uses.
var composeDefault = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*):-([^}]*)\}`)

// composeExpand resolves ${VAR:-default} as docker compose does: the
// variable when set and non-empty, else the default.
func composeExpand(s string) string {
	return composeDefault.ReplaceAllStringFunc(s, func(m string) string {
		g := composeDefault.FindStringSubmatch(m)
		if v := os.Getenv(g[1]); v != "" {
			return v
		}
		return g[2]
	})
}

func coreURL() string {
	if u := os.Getenv("NAVA_TEST_CORE_URL"); u != "" {
		return strings.TrimRight(u, "/")
	}
	if p := os.Getenv("NAVA_CORE_PORT"); p != "" {
		return "http://localhost:" + p
	}
	return "http://localhost:18080"
}

// forgedRequest sends req to Keycloak with the forged Host and headers.
func forgedRequest(t *testing.T, req *http.Request, host string, header http.Header) (int, []byte) {
	t.Helper()
	if host != "" {
		req.Host = host
	}
	for k, v := range header {
		req.Header[k] = v
	}
	// A plain client: no redirect rewriting, so nothing undoes the forgery.
	return do(t, &http.Client{Timeout: newClient(t).Timeout}, req)
}

// forgedToken is alice's nava-web password-grant token, requested with the
// forged Host and headers. It is "" when Keycloak refuses the request with a
// 4xx (a malformed Host such as keycloak:8080.evil.example gets 400): no token
// was issued, so there is no iss to check. Any other non-200 is fatal.
func forgedToken(t *testing.T, host string, header http.Header) string {
	t.Helper()
	return forgedTokenFor(t, host, header, url.Values{
		"grant_type": {"password"}, "client_id": {"nava-web"},
		"username": {"alice@tenant-a.test"}, "password": {"dev-only-alice"}, "scope": {"openid"},
	})
}

// forgedOperatorToken is forgedToken for operator@nava.test through
// nava-operator-cli, the token nava-onboard accepts.
func forgedOperatorToken(t *testing.T, host string, header http.Header) string {
	t.Helper()
	return forgedTokenFor(t, host, header, url.Values{
		"grant_type": {"password"}, "client_id": {"nava-operator-cli"},
		"username": {"operator@nava.test"}, "password": {operatorPass}, "scope": {"openid"},
	})
}

func forgedTokenFor(t *testing.T, host string, header http.Header, form url.Values) string {
	t.Helper()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost,
		baseURL()+"/realms/nava/protocol/openid-connect/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	code, body := forgedRequest(t, req, host, header)
	if code >= 400 && code < 500 {
		t.Logf("token request refused: %d %s", code, body)
		return ""
	}
	if code != http.StatusOK {
		t.Fatalf("token request: %d %s", code, body)
	}
	var out struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(body, &out); err != nil || out.AccessToken == "" {
		t.Fatalf("token response: %v %s", err, body)
	}
	return out.AccessToken
}

// me calls GET /api/v1/me on the running nava-core with tok.
func me(t *testing.T, tok string) (int, string) {
	t.Helper()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, coreURL()+"/api/v1/me", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	code, body := do(t, &http.Client{Timeout: newClient(t).Timeout}, req)
	return code, string(body)
}

// TestForgedIssuerTokenRejectedByAPI: every token whose iss is not the pinned
// issuer is refused by nava-core with 401 invalid_token. A token whose iss is
// the pinned one (Keycloak ignored the forgery) is legitimate and must get
// past signature and issuer checks, which also proves the refusals below come
// from the issuer and not from an unreachable or misconfigured API.
func TestForgedIssuerTokenRejectedByAPI(t *testing.T) {
	pinned := pinnedIssuer(t)
	pinnedHost := strings.TrimPrefix(strings.TrimPrefix(pinned, "http://"), "https://")
	pinnedHost, _, _ = strings.Cut(pinnedHost, "/")

	// Control: a token issued under the pinned issuer is not invalid_token
	// (200 or tenant_unresolved, depending on the stack's bindings).
	ctl := forgedToken(t, pinnedHost, nil)
	if ctl == "" {
		t.Fatal("control token request refused")
	}
	if iss := claims(t, ctl)["iss"]; iss != pinned {
		t.Fatalf("control token iss = %v, want %s", iss, pinned)
	}
	if code, body := me(t, ctl); code == http.StatusUnauthorized && strings.Contains(body, `"code":"invalid_token"`) {
		t.Fatalf("control token refused as invalid_token: the API does not trust the pinned issuer, so the refusals below prove nothing: %s", body)
	}

	issued := 0
	for name, f := range forgedHeaders() {
		t.Run(name, func(t *testing.T) {
			tok := forgedToken(t, f.host, f.header)
			if tok == "" {
				return // refused at issue: no token reaches the API
			}
			issued++
			iss, _ := claims(t, tok)["iss"].(string)
			if strings.Contains(iss, "evil.example") {
				t.Logf("Keycloak issued iss %s from a forged header (TICK-160 pins it)", iss)
			}
			code, body := me(t, tok)
			if iss == pinned {
				if code == http.StatusUnauthorized && strings.Contains(body, `"code":"invalid_token"`) {
					t.Errorf("pinned-issuer token refused as invalid_token: %s", body)
				}
				return
			}
			if code != http.StatusUnauthorized || !strings.Contains(body, `"code":"invalid_token"`) {
				t.Errorf("token with iss %s: %d %s, want 401 invalid_token", iss, code, body)
			}
		})
	}
	// Keycloak answers most forged shapes (X-Forwarded-Host is ignored with
	// proxy headers off), so a run where every one is refused checked nothing.
	if issued == 0 {
		t.Error("every forged token request was refused: the API checks above never ran")
	}
}

// TestForgedIssuerTokenRejectedByOnboard is the same check on the operator
// onboarding endpoint, with the operator's token: one whose iss is not the
// pinned issuer gets 401 from nava-onboard. A pinned-issuer operator token
// (Keycloak ignored the forgery) is legitimate and onboards a fresh tenant, so
// it must be neither 401 nor 403; that control proves the 401s below come from
// the issuer and not from the operator checks.
func TestForgedIssuerTokenRejectedByOnboard(t *testing.T) {
	pinned := pinnedIssuer(t)
	pinnedHost := strings.TrimPrefix(strings.TrimPrefix(pinned, "http://"), "https://")
	pinnedHost, _, _ = strings.Cut(pinnedHost, "/")

	post := func(tok string) int {
		key, _, domain, email := newOnboarding(t)
		code, _ := onboardPost(t, tok, key, onboardBody(email, domain))
		return code
	}
	ctl := forgedOperatorToken(t, pinnedHost, nil)
	if ctl == "" {
		t.Fatal("control token request refused")
	}
	if iss := claims(t, ctl)["iss"]; iss != pinned {
		t.Fatalf("control token iss = %v, want %s", iss, pinned)
	}
	if code := post(ctl); code == http.StatusUnauthorized || code == http.StatusForbidden {
		t.Fatalf("control operator token refused with %d: the refusals below prove nothing", code)
	}

	issued, foreign := 0, 0
	for name, f := range forgedHeaders() {
		t.Run(name, func(t *testing.T) {
			tok := forgedOperatorToken(t, f.host, f.header)
			if tok == "" {
				return // refused at issue: no token reaches the API
			}
			issued++
			if iss, _ := claims(t, tok)["iss"].(string); iss == pinned {
				return // Keycloak ignored the forgery: legitimate issuer
			}
			foreign++
			if code := post(tok); code != http.StatusUnauthorized {
				t.Errorf("token with a foreign iss: %d, want 401", code)
			}
		})
	}
	t.Logf("%d forged token requests issued a token, %d with a foreign iss", issued, foreign)
	if issued == 0 {
		t.Error("every forged token request was refused: the onboarding checks above never ran")
	}
}

// TestIssuerPinnedRegardlessOfHost is the TICK-160 Done list on the running
// Keycloak: with KC_HOSTNAME set and proxy headers trusted only from known
// addresses, no request header moves the issuer. The browser address, the
// back-channel address and every forged header get the same iss, in the token
// and in the discovery document (whose endpoints are the redirect base).
func TestIssuerPinnedRegardlessOfHost(t *testing.T) {
	pinned := pinnedIssuer(t)
	// The legitimate addresses must issue a token; only forged ones may be
	// refused.
	legit := map[string]forgery{
		"browser":      {host: realmHost},
		"back-channel": {host: "keycloak:8080"},
		"loopback-ip":  {host: "127.0.0.1:18081"},
	}
	hosts := map[string]forgery{}
	for k, v := range legit {
		hosts[k] = v
	}
	for k, v := range forgedHeaders() {
		hosts[k] = v
	}
	for name, h := range hosts {
		t.Run(name, func(t *testing.T) {
			_, isLegit := legit[name]
			switch tok := forgedToken(t, h.host, h.header); {
			case tok == "" && isLegit:
				t.Errorf("token request from %s refused", h.host)
			case tok != "":
				if iss := claims(t, tok)["iss"]; iss != pinned {
					t.Errorf("token iss = %v, want %s", iss, pinned)
				}
			}
			req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet,
				baseURL()+"/realms/nava/.well-known/openid-configuration", nil)
			code, body := forgedRequest(t, req, h.host, h.header)
			if code >= 400 && code < 500 && !isLegit {
				return // refused: no document, no endpoint to follow
			}
			if code != http.StatusOK {
				t.Fatalf("discovery: %d %s", code, body)
			}
			var d map[string]any
			if err := json.Unmarshal(body, &d); err != nil {
				t.Fatal(err)
			}
			if d["issuer"] != pinned {
				t.Errorf("discovery issuer = %v, want %s", d["issuer"], pinned)
			}
			// The front-channel endpoints are where a browser is sent; none
			// may follow the forged host.
			for _, k := range []string{"authorization_endpoint", "end_session_endpoint"} {
				if s, _ := d[k].(string); strings.Contains(s, "evil.example") {
					t.Errorf("discovery %s = %s follows the forged host", k, s)
				}
			}
		})
	}
}

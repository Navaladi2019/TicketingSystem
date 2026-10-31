//go:build identitystack

package identitystack

// Onboarding end to end on the compose stack (T21b, TICK-234): the operator
// onboards a tenant through the nava-onboard service, the invited admin signs
// in through the tenant's IdP (nava-first-broker-login), a group sync cycle
// runs, and nava-core's GET /api/v1/me resolves the new tenant. Also the
// Security conditions from the TICK-81 re-review: nava-onboard and nava-core
// are separate containers, nava-core holds no onboarding secret, and the
// operator route is not served through the tenant ingress; and the kill/retry
// between the Keycloak organization and the binding, against the real
// nava_onboard PGStore.
//
//	make stack-up
//	make stack-test-onboard
//
// The tokens are fetched as the pinned issuer (KC_HOSTNAME, TICK-160), the one
// nava-core and nava-onboard accept (NAVA_OIDC_ISSUERS): Keycloak names it
// whatever Host the request used, so inNet sends requests for that address to
// the published port. nava-onboard is never published; the test reaches it
// with `docker compose exec`, as an operator on the host would.

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/identity/keycloak"
	"github.com/Navaladi2019/TicketingSystem/internal/onboard"
)

const (
	// operatorSub is operator@nava.test's fixed id in the realm export, the
	// subject on NAVA_OPERATORS in compose.yaml.
	operatorSub  = "00000000-0000-4000-8000-00000000f001"
	operatorPass = "dev-only-operator"
	// The nava_onboard database URL's user: it may not appear in nava-core's
	// environment. The secret values are read from nava-onboard at run time.
	onboardDBUser = "nava_onboard:"
)

// repoRoot is the repository root, from this package; docker compose runs
// there so COMPOSE_FILE and COMPOSE_PROJECT_NAME from make apply.
const repoRoot = "../.."

func dockerOut(t *testing.T, stdin string, args ...string) (string, error) {
	t.Helper()
	// Not t.Context() alone: it is already canceled when a Cleanup runs.
	ctx, cancel := context.WithTimeout(context.WithoutCancel(t.Context()), 3*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Dir = repoRoot
	cmd.Stdin = strings.NewReader(stdin)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return string(out), fmt.Errorf("docker %s: %w: %s", strings.Join(args[:min(len(args), 3)], " "), err, strings.TrimSpace(stderr.String()))
	}
	return string(out), nil
}

func containerOf(t *testing.T, service string) string {
	t.Helper()
	out, err := dockerOut(t, "", "compose", "ps", "-q", service)
	id := strings.TrimSpace(out)
	if err != nil || id == "" || strings.Contains(id, "\n") {
		t.Fatalf("container of %s: %q %v", service, out, err)
	}
	return id
}

// psqlSuper runs sql as the superuser over the postgres container's Unix
// socket (deploy/compose/test/lib.sh psql_super), unaligned, tuples only.
func psqlSuper(t *testing.T, sql string) string {
	t.Helper()
	out, err := dockerOut(t, "", "compose", "exec", "-T", "postgres", "sh", "-c",
		`PGPASSWORD=$(cat /run/pg_superuser_password) exec psql -X -q -t -A -v ON_ERROR_STOP=1 -U pgadmin -d nava -c "$0"`, sql)
	if err != nil {
		t.Fatalf("psql %q: %v", sql, err)
	}
	return strings.TrimSpace(out)
}

// pinnedBase is the scheme://host[:port] of the pinned issuer, the address
// browsers (and the tests) use for Keycloak; pinnedHost is its host[:port].
func pinnedBase(t *testing.T) string {
	t.Helper()
	return strings.TrimSuffix(pinnedIssuer(t), "/realms/nava")
}

func pinnedHost(t *testing.T) string {
	t.Helper()
	u, err := url.Parse(pinnedBase(t))
	if err != nil || u.Host == "" {
		t.Fatalf("pinned issuer %q has no host", pinnedIssuer(t))
	}
	return u.Host
}

// inNet is a client whose requests for the pinned address (and for realmHost)
// go to the Keycloak under test, so issuers, redirects and cookies are the
// ones a browser sees.
type inNet struct{ host, pinned string }

func (s inNet) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Host == s.pinned || req.URL.Host == realmHost {
		host := req.URL.Host
		req = req.Clone(req.Context())
		req.URL.Host = s.host
		req.Host = host
		if host == realmHost {
			req.Host = s.host
		}
	}
	return http.DefaultTransport.RoundTrip(req)
}

// secureJar keeps Keycloak's Secure cookies over plain HTTP: the cookiejar
// treats only localhost as a secure origin, not the pinned host.
type secureJar struct{ http.CookieJar }

func asHTTPS(u *url.URL) *url.URL {
	c := *u
	c.Scheme = "https"
	return &c
}

func (j secureJar) SetCookies(u *url.URL, cs []*http.Cookie) { j.CookieJar.SetCookies(asHTTPS(u), cs) }
func (j secureJar) Cookies(u *url.URL) []*http.Cookie        { return j.CookieJar.Cookies(asHTTPS(u)) }

func newInNetClient(t *testing.T) *http.Client {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(baseURL())
	if err != nil {
		t.Fatal(err)
	}
	return &http.Client{Jar: secureJar{jar}, Timeout: 30 * time.Second, Transport: inNet{host: u.Host, pinned: pinnedHost(t)}}
}

func inNetToken(t *testing.T, c *http.Client, form url.Values) string {
	t.Helper()
	req, _ := http.NewRequestWithContext(t.Context(), http.MethodPost,
		pinnedIssuer(t)+"/protocol/openid-connect/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	code, body := do(t, c, req)
	var out struct {
		AccessToken string `json:"access_token"`
	}
	if code != http.StatusOK || json.Unmarshal(body, &out) != nil || out.AccessToken == "" {
		t.Fatalf("token as %s: %d %.300s", pinnedHost(t), code, body)
	}
	if iss, _ := claims(t, out.AccessToken)["iss"].(string); iss != pinnedIssuer(t) {
		t.Fatalf("token iss = %q, want %q", iss, pinnedIssuer(t))
	}
	return out.AccessToken
}

func operatorToken(t *testing.T) string {
	return inNetToken(t, newInNetClient(t), url.Values{
		"grant_type": {"password"}, "client_id": {"nava-operator-cli"},
		"username": {"operator@nava.test"}, "password": {operatorPass}, "scope": {"openid"},
	})
}

// onboardBody is an onboarding request whose IdP URLs are public address
// literals: nava-onboard refuses private hosts and resolves nothing on the
// stack's internal network. pointIdPAt later replaces them with a test realm.
func onboardBody(email, domain string) []byte {
	b, _ := json.Marshal(map[string]any{
		"admin_email":   email,
		"email_domains": []string{domain},
		"identity_provider": map[string]any{
			"provider_id": "oidc", "display_name": "Onboarded " + domain,
			"config": map[string]string{
				"clientId": "nava-broker", "clientSecret": "dev-only-broker-secret", "clientAuthMethod": "client_secret_post",
				"authorizationUrl": "https://93.184.216.34/auth", "tokenUrl": "https://93.184.216.34/token",
				"jwksUrl": "https://93.184.216.34/jwks", "useJwksUrl": "true", "defaultScope": "openid email profile",
			},
		},
	})
	return b
}

// onboardPost sends POST /api/v1/operator/tenants to nava-onboard's own
// listener, from inside its container with busybox wget. Not nc: it
// half-closes the connection when its input ends, and the server cancels the
// request on that EOF. wget keeps no body for an error status, so a 4xx or
// 5xx comes back with an empty body.
func onboardPost(t *testing.T, token, key string, body []byte) (int, []byte) {
	t.Helper()
	code, b, stderr := onboardPostRaw(t, token, key, body)
	if code == 0 {
		t.Fatalf("no HTTP response from nava-onboard: %.400s", stderr)
	}
	return code, b
}

// onboardPostRaw is onboardPost that reports no response (code 0, and wget's
// output) instead of failing the test, so it may run on another goroutine.
func onboardPostRaw(t *testing.T, token, key string, body []byte) (code int, resp []byte, stderrText string) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(t.Context()), 3*time.Minute)
	defer cancel()
	// The token goes through the environment, never argv, as
	// docs/operator/onboarding.md tells the operator to call it.
	cmd := exec.CommandContext(ctx, "docker", "compose", "exec", "-T", "-e", "NAVA_OPERATOR_TOKEN", "nava-onboard", "sh", "-c",
		`exec wget -S -T 120 -O - --header "Authorization: Bearer $NAVA_OPERATOR_TOKEN" --header "Idempotency-Key: $1" `+
			`--header "Content-Type: application/json" --post-data "$2" http://127.0.0.1:8081/api/v1/operator/tenants`,
		"sh", key, string(body))
	cmd.Dir = repoRoot
	cmd.Env = append(os.Environ(), "NAVA_OPERATOR_TOKEN="+token)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	_ = cmd.Run() // non-zero for any error status; the status line decides
	m := statusLine.FindAllStringSubmatch(stderr.String(), -1)
	if m == nil {
		return 0, stdout.Bytes(), stderr.String()
	}
	code, _ = strconv.Atoi(m[len(m)-1][1])
	return code, stdout.Bytes(), stderr.String()
}

var statusLine = regexp.MustCompile(`HTTP/1\.[01] (\d{3})`)

type onboarded struct {
	TenantID uuid.UUID `json:"tenant_id"`
	State    string    `json:"state"`
}

func mustOnboard(t *testing.T, token, key string, body []byte, want int, tenant uuid.UUID) {
	t.Helper()
	code, resp := onboardPost(t, token, key, body)
	var got onboarded
	if code != want || json.Unmarshal(resp, &got) != nil || got.TenantID != tenant || got.State != "active" {
		t.Fatalf("onboard: %d %.400s, want %d with tenant %s active", code, resp, want, tenant)
	}
}

// createIdPRealm stands in for the tenant's own IdP: a realm with the broker
// client and one user, removed after the test.
func createIdPRealm(t *testing.T, realm, email, pass string) {
	t.Helper()
	body := map[string]any{
		"realm": realm, "enabled": true, "sslRequired": "none",
		"clients": []map[string]any{{
			"clientId": "nava-broker", "enabled": true, "publicClient": false, "clientAuthenticatorType": "client-secret",
			"secret": "dev-only-broker-secret", "standardFlowEnabled": true,
			"redirectUris": []string{pinnedBase(t) + "/realms/nava/broker/*"},
		}},
		"users": []map[string]any{{
			"username": email, "email": email, "emailVerified": true, "enabled": true, "firstName": "Onboarded", "lastName": "Admin",
			"credentials": []map[string]any{{"type": "password", "value": pass, "temporary": false}},
		}},
	}
	raw, _ := json.Marshal(body)
	req, _ := http.NewRequestWithContext(t.Context(), http.MethodPost, baseURL()+"/admin/realms", bytes.NewReader(raw))
	req.Header.Set("Authorization", "Bearer "+masterToken(t))
	req.Header.Set("Content-Type", "application/json")
	if code, b := do(t, newClient(t), req); code != http.StatusCreated {
		t.Fatalf("create IdP realm %s: %d %s", realm, code, b)
	}
	t.Cleanup(func() {
		del, _ := http.NewRequestWithContext(context.Background(), http.MethodDelete, baseURL()+"/admin/realms/"+realm, nil)
		del.Header.Set("Authorization", "Bearer "+masterToken(t))
		do(t, newClient(t), del)
	})
}

// pointIdPAt repoints the onboarded IdP's endpoints from the placeholder
// public addresses to the test realm, the way the idp-tenant-* fixtures are
// configured (browser endpoint by name, back channel to Keycloak itself).
// Only the URLs change (and the secret is re-entered, unchanged): alias, organization link, first-login flow and the
// pinned validateSignature/syncMode stay as onboarding created them.
func pointIdPAt(t *testing.T, alias, realm string) {
	t.Helper()
	code, b := adminRealm(t, "nava", http.MethodGet, "/identity-provider/instances/"+url.PathEscape(alias), nil)
	var idp map[string]any
	if code != http.StatusOK || json.Unmarshal(b, &idp) != nil {
		t.Fatalf("IdP %s: %d %s", alias, code, b)
	}
	cfg, _ := idp["config"].(map[string]any)
	if cfg == nil || cfg["validateSignature"] != "true" || cfg["syncMode"] != "IMPORT" {
		t.Fatalf("IdP %s config as onboarded = %v, want validateSignature=true and syncMode=IMPORT", alias, cfg)
	}
	cfg["authorizationUrl"] = pinnedBase(t) + "/realms/" + realm + "/protocol/openid-connect/auth"
	cfg["tokenUrl"] = "http://localhost:8080/realms/" + realm + "/protocol/openid-connect/token"
	cfg["jwksUrl"] = "http://localhost:8080/realms/" + realm + "/protocol/openid-connect/certs"
	// The GET masks the secret, and Keycloak wants it re-entered when the
	// token URL changes: the same value the onboarding request sent.
	cfg["clientSecret"] = "dev-only-broker-secret"
	if code, b := adminRealm(t, "nava", http.MethodPut, "/identity-provider/instances/"+url.PathEscape(alias), idp); code != http.StatusNoContent {
		t.Fatalf("repoint IdP %s: %d %s", alias, code, b)
	}
}

var pageText = regexp.MustCompile(`(?s)<[^>]*>`)

// brokeredSignIn runs the code flow of nava-web at the pinned address with the
// tenant IdP hinted, signs in at the IdP, and exchanges the code. It fails
// with the text of the last page when Keycloak does not return a code.
func brokeredSignIn(t *testing.T, idp, user, pass string) string {
	t.Helper()
	tok, page := brokeredAttempt(t, idp, user, pass)
	if tok == "" {
		t.Fatalf("the brokered sign-in returned no code; last page: %.600s", page)
	}
	return tok
}

// brokeredAttempt is brokeredSignIn that reports a refused login (an empty
// token and the text of the last page) instead of failing the test.
func brokeredAttempt(t *testing.T, idp, user, pass string) (token, page string) {
	t.Helper()
	return brokeredAttemptAs(t, "nava-web", "http://localhost:9/cb", false, idp, user, pass)
}

// accountConsoleSignIn signs in through the tenant IdP as the built-in
// account-console client, whose tokens the account API accepts (it carries the
// account audience and roles). The client is public and requires PKCE.
func accountConsoleSignIn(t *testing.T, idp, user, pass string) string {
	t.Helper()
	tok, page := brokeredAttemptAs(t, "account-console", pinnedBase(t)+"/realms/nava/account/", true, idp, user, pass)
	if tok == "" {
		t.Fatalf("the account-console sign-in returned no code; last page: %.600s", page)
	}
	return tok
}

// brokeredAttemptAs is brokeredAttempt for any public client and redirect URI;
// pkce adds an S256 code challenge and sends the verifier at the token call.
func brokeredAttemptAs(t *testing.T, clientID, redirect string, pkce bool, idp, user, pass string) (token, page string) {
	t.Helper()
	c := newInNetClient(t)
	var final *url.URL
	c.CheckRedirect = func(req *http.Request, _ []*http.Request) error {
		if strings.HasPrefix(req.URL.String(), redirect) {
			final = req.URL
			return http.ErrUseLastResponse
		}
		return nil
	}
	q := url.Values{
		"client_id": {clientID}, "response_type": {"code"}, "scope": {"openid"},
		"redirect_uri": {redirect}, "kc_idp_hint": {idp},
	}
	verifier := strings.ReplaceAll(uuid.NewString()+uuid.NewString(), "-", "")
	if pkce {
		sum := sha256.Sum256([]byte(verifier))
		q.Set("code_challenge", base64.RawURLEncoding.EncodeToString(sum[:]))
		q.Set("code_challenge_method", "S256")
	}
	req, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, pinnedIssuer(t)+"/protocol/openid-connect/auth?"+q.Encode(), nil)
	_, login := do(t, c, req)
	m := formAction.FindSubmatch(login)
	if m == nil {
		m = formActionAlt.FindSubmatch(login)
	}
	if m == nil {
		t.Fatalf("no login form from the tenant IdP: %.400s", login)
	}
	post, _ := http.NewRequestWithContext(t.Context(), http.MethodPost, html.UnescapeString(string(m[1])),
		strings.NewReader(url.Values{"username": {user}, "password": {pass}}.Encode()))
	post.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	_, body := do(t, c, post)
	if final == nil || final.Query().Get("code") == "" {
		return "", strings.Join(strings.Fields(html.UnescapeString(pageText.ReplaceAllString(string(body), " "))), " ")
	}
	form := url.Values{
		"grant_type": {"authorization_code"}, "client_id": {clientID},
		"code": {final.Query().Get("code")}, "redirect_uri": {redirect},
	}
	if pkce {
		form.Set("code_verifier", verifier)
	}
	return inNetToken(t, c, form), ""
}

// stopGroupsync stops the long-running nava-groupsync for the test and brings
// it back, healthy, in Cleanup: while it runs, a cycle of its own can hold a
// tenant's lock, so the --once cycle skips that tenant, exits 0, and /me
// fails for a reason that is not the one under test.
func stopGroupsync(t *testing.T) {
	t.Helper()
	if out, err := dockerOut(t, "", "compose", "stop", "nava-groupsync"); err != nil {
		t.Fatalf("stop nava-groupsync: %v\n%s", err, out)
	}
	t.Cleanup(func() {
		if out, err := dockerOut(t, "", "compose", "up", "-d", "--no-deps", "--wait", "nava-groupsync"); err != nil {
			t.Errorf("restart nava-groupsync: %v\n%s", err, out)
		}
	})
}

// groupsyncOnce runs one nava-groupsync cycle, which syncs every bound
// tenant's principals; it exits non-zero when any tenant failed. Call
// stopGroupsync first.
func groupsyncOnce(t *testing.T) {
	t.Helper()
	if out, err := dockerOut(t, "", "compose", "run", "--rm", "--no-deps", "-T", "nava-groupsync", "--once"); err != nil {
		t.Fatalf("nava-groupsync --once: %v\n%s", err, out)
	}
}

type meResp struct {
	UserID   uuid.UUID   `json:"user_id"`
	TenantID uuid.UUID   `json:"tenant_id"`
	Groups   []uuid.UUID `json:"groups"`
	IdPAlias string      `json:"idp_alias"`
}

func getMe(t *testing.T, token, query string, header http.Header) (int, meResp, []byte) {
	t.Helper()
	req, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, coreURL()+"/api/v1/me"+query, nil)
	for k, v := range header {
		req.Header[k] = v
	}
	req.Header.Set("Authorization", "Bearer "+token)
	code, b := do(t, &http.Client{Timeout: 30 * time.Second}, req)
	var m meResp
	_ = json.Unmarshal(b, &m)
	return code, m, b
}

func orgMembers(t *testing.T, orgID string) []string {
	t.Helper()
	code, b := adminRealm(t, "nava", http.MethodGet, "/organizations/"+orgID+"/members?max=100", nil)
	var us []struct{ ID string }
	if code != http.StatusOK || json.Unmarshal(b, &us) != nil {
		t.Fatalf("members of organization %s: %d %s", orgID, code, b)
	}
	ids := make([]string, len(us))
	for i, u := range us {
		ids[i] = u.ID
	}
	return ids
}

// newOnboarding is a fresh key, the tenant id nava-onboard derives from it,
// and a domain no other run uses; the tenant's Keycloak objects and the
// admin's nava user are removed after the test.
func newOnboarding(t *testing.T) (key string, tenant uuid.UUID, domain, email string) {
	t.Helper()
	key = uuid.NewString()
	tenant = onboard.TenantID(operatorSub, key)
	domain = "onb-" + tenant.String()[:8] + ".test"
	email = "admin@" + domain
	cleanupTenantOrg(t, tenant, onboard.Alias(tenant))
	deleteUserAfter(t, email)
	return key, tenant, domain, email
}

// TestOnboardedAdminSignsInAndMeShowsTenant: onboard through nava-onboard,
// sign in as the invited admin through the tenant's IdP, sync, and GET
// /api/v1/me on nava-core resolves the new tenant and no other.
func TestOnboardedAdminSignsInAndMeShowsTenant(t *testing.T) {
	stopGroupsync(t)
	key, tenant, domain, email := newOnboarding(t)
	alias, idpAlias := onboard.Alias(tenant), keycloak.TenantIdPAlias(tenant)
	idpRealm, pass := "onb-"+tenant.String()[:8], "dev-only-"+uuid.NewString()[:8]
	createIdPRealm(t, idpRealm, email, pass)

	op := operatorToken(t)
	body := onboardBody(email, domain)
	mustOnboard(t, op, key, body, http.StatusCreated, tenant)
	// A replay of the completed request: 200, the same tenant.
	mustOnboard(t, op, key, body, http.StatusOK, tenant)

	orgs := orgsByAlias(t, alias)
	if len(orgs) != 1 {
		t.Fatalf("organizations %s: %v, want exactly one", alias, orgs)
	}
	if got := psqlSuper(t, fmt.Sprintf(
		"SELECT t.state || '|' || b.issuer || '|' || b.org_id || '|' || array_to_string(b.idp_aliases, ',') "+
			"FROM tenants t JOIN tenant_identity_binding b ON b.tenant_id = t.id WHERE t.id = '%s'", tenant)); got != "active|"+pinnedIssuer(t)+"|"+orgs[0]+"|"+idpAlias {
		t.Fatalf("tenant row and binding = %q, want active, bound to %s %s with IdP %s", got, pinnedIssuer(t), orgs[0], idpAlias)
	}

	// Option 2 (TICK-252): onboarding wrote a pending invite and created no
	// Keycloak user, so the admin cannot sign in locally and nothing can
	// collide with the first brokered login.
	if kcUserByEmail(t, email) != "" {
		t.Fatalf("onboarding created a Keycloak user for %s: the invite must not", email)
	}
	if got := inviteState(t, tenant, email); got != "pending" {
		t.Fatalf("invite after onboarding = %q, want pending", got)
	}

	pointIdPAt(t, idpAlias, idpRealm)
	tok := brokeredSignIn(t, idpAlias, email, pass)
	c := claims(t, tok)
	orgClaim(t, c, alias) // exactly one organization, the new tenant's

	// The first login created the user and made them an org member; group
	// sync has not placed them yet.
	sub, _ := c["sub"].(string)
	created := kcUserByEmail(t, email)
	if created == "" || sub != created {
		t.Fatalf("token sub = %q, the user with %s = %q", sub, email, created)
	}
	if !slices.Contains(orgMembers(t, orgs[0]), created) {
		t.Errorf("the admin is not a member of organization %s", alias)
	}
	if userInGroup(t, created, kcGroupID(t, "/t/"+tenant.String())) {
		t.Errorf("the admin is in /t/%s before any group sync cycle", tenant)
	}

	groupsyncOnce(t)

	// The cycle placed the admin and consumed the invite.
	if !userInGroup(t, created, kcGroupID(t, "/t/"+tenant.String())) {
		t.Errorf("the admin is not a member of /t/%s after the cycle", tenant)
	}
	if got := inviteState(t, tenant, email); got != "consumed" {
		t.Errorf("invite after the cycle = %q, want consumed", got)
	}

	code, got, raw := getMe(t, tok, "", nil)
	if code != http.StatusOK {
		t.Fatalf("GET /api/v1/me: %d %s", code, raw)
	}
	if got.TenantID != tenant || got.IdPAlias != idpAlias || got.UserID == uuid.Nil {
		t.Errorf("/me = %s, want tenant %s via %s", raw, tenant, idpAlias)
	}

	// P3-04: onboarding recorded the admins group it created, and /me shows it.
	// Bounded poll on group sync, never a sleep: each round runs one cycle.
	kcAdmins := kcGroupID(t, "/t/"+tenant.String()+"/admins")
	wantAdmin := psqlSuper(t, fmt.Sprintf(
		"SELECT g.id FROM groups g WHERE g.tenant_id = '%s' AND g.keycloak_group_id = '%s'", tenant, kcAdmins))
	if wantAdmin == "" {
		t.Fatalf("no groups row for the admins group %s of tenant %s after sync", kcAdmins, tenant)
	}
	if rec := psqlSuper(t, fmt.Sprintf("SELECT group_id FROM tenant_admin_group WHERE tenant_id = '%s'", tenant)); rec != wantAdmin {
		t.Fatalf("tenant_admin_group holds %q, want the group onboarding created, %q", rec, wantAdmin)
	}
	adminID := uuid.MustParse(wantAdmin)
	var meGroups []uuid.UUID
	for round := 0; round < 5 && !slices.Contains(meGroups, adminID); round++ {
		if round > 0 {
			groupsyncOnce(t)
		}
		code, m, raw := getMe(t, tok, "", nil)
		if code != http.StatusOK {
			t.Fatalf("GET /api/v1/me: %d %s", code, raw)
		}
		meGroups = m.Groups
	}
	if !slices.Contains(meGroups, adminID) {
		t.Errorf("/me groups = %v, want the admins group %s", meGroups, adminID)
	}

	// Security condition: the token resolves to the new tenant only, whatever
	// tenant the request names, including a real seeded tenant.
	other := tenantA.String()
	for name, try := range map[string]struct {
		query  string
		header http.Header
	}{
		"X-Tenant-Id header": {header: http.Header{"X-Tenant-Id": {other}}},
		"tenant_id query":    {query: "?tenant_id=" + other},
	} {
		code, m, raw := getMe(t, tok, try.query, try.header)
		if code != http.StatusOK || m.TenantID != tenant {
			t.Errorf("%s naming %s: %d %s, want the new tenant %s", name, other, code, raw, tenant)
		}
	}
}

// TestOnboardDeploymentSeparation: nava-onboard and nava-core are separate
// containers; nava-core's environment holds no onboarding variable or secret;
// nava-onboard publishes no port and is on the internal network only; the
// operator route through the tenant ingress (nava-core's published port) is
// not served and onboards nothing.
func TestOnboardDeploymentSeparation(t *testing.T) {
	core, onb := containerOf(t, "nava-core"), containerOf(t, "nava-onboard")
	if core == onb {
		t.Fatal("nava-core and nava-onboard run in one container")
	}
	env := func(id string) []string {
		out, err := dockerOut(t, "", "inspect", "-f", "{{json .Config.Env}}", id)
		var vs []string
		if err != nil || json.Unmarshal([]byte(out), &vs) != nil {
			t.Fatalf("env of %s: %v %s", id, err, out)
		}
		return vs
	}
	// The values the running nava-onboard holds, not the compose defaults:
	// an overridden secret or password must not reach nava-core either.
	onbEnv := env(onb)
	lookup := func(vs []string, key string) string {
		for _, v := range vs {
			if name, val, _ := strings.Cut(v, "="); name == key {
				return val
			}
		}
		return ""
	}
	secret := lookup(onbEnv, "NAVA_ONBOARD_CLIENT_SECRET")
	dbURL, err := url.Parse(lookup(onbEnv, "NAVA_ONBOARD_DATABASE_URL"))
	if err != nil {
		t.Fatalf("nava-onboard's NAVA_ONBOARD_DATABASE_URL: %v", err)
	}
	dbPass, _ := dbURL.User.Password()
	if secret == "" || dbPass == "" || dbURL.User.Username() != "nava_onboard" {
		t.Fatalf("nava-onboard holds no onboarding client secret or nava_onboard password (secret set %t, password set %t, user %q)",
			secret != "", dbPass != "", dbURL.User.Username())
	}
	// holdsOnboarding returns the variables named NAVA_ONBOARD_* or naming the
	// nava_onboard user, and, separately, those whose value holds the secret
	// or the password.
	holdsOnboarding := func(vs []string) (named, byValue []string) {
		for _, v := range vs {
			name, val, _ := strings.Cut(v, "=")
			if strings.HasPrefix(name, "NAVA_ONBOARD_") || strings.Contains(val, onboardDBUser) {
				named = append(named, name)
			}
			if strings.Contains(val, secret) || strings.Contains(val, dbPass) {
				byValue = append(byValue, name)
			}
		}
		return named, byValue
	}
	if named, byValue := holdsOnboarding(env(core)); len(named) != 0 || len(byValue) != 0 {
		t.Errorf("nava-core's environment holds onboarding variables %v or the onboarding secret values %v", named, byValue)
	}
	// The check's control: the same scan finds both values in nava-onboard.
	if _, byValue := holdsOnboarding(onbEnv); !slices.Contains(byValue, "NAVA_ONBOARD_CLIENT_SECRET") || !slices.Contains(byValue, "NAVA_ONBOARD_DATABASE_URL") {
		t.Errorf("the scan does not find the onboarding secret values in nava-onboard (%v): it proves nothing about nava-core", byValue)
	}

	out, err := dockerOut(t, "", "inspect", "-f", "{{json .NetworkSettings.Ports}}|{{range $k, $v := .NetworkSettings.Networks}}{{$k}} {{end}}", onb)
	if err != nil {
		t.Fatal(err)
	}
	ports, nets, _ := strings.Cut(strings.TrimSpace(out), "|")
	var bindings map[string][]struct{ HostPort string }
	if err := json.Unmarshal([]byte(ports), &bindings); err != nil {
		t.Fatalf("nava-onboard ports %q: %v", ports, err)
	}
	for p, bs := range bindings {
		if len(bs) != 0 {
			t.Errorf("nava-onboard publishes %s on the host: %+v", p, bs)
		}
	}
	if ns := strings.Fields(nets); len(ns) != 1 || !strings.HasSuffix(ns[0], "_internal") {
		t.Errorf("nava-onboard networks = %v, want the internal network only", ns)
	}

	// The tenant ingress: a valid operator token and a well-formed request
	// to nava-core get 501 (or 404), and no tenant row appears.
	key, tenant, domain, email := newOnboarding(t)
	req, _ := http.NewRequestWithContext(t.Context(), http.MethodPost, coreURL()+"/api/v1/operator/tenants", bytes.NewReader(onboardBody(email, domain)))
	req.Header.Set("Authorization", "Bearer "+operatorToken(t))
	req.Header.Set("Idempotency-Key", key)
	req.Header.Set("Content-Type", "application/json")
	code, b := do(t, &http.Client{Timeout: 30 * time.Second}, req)
	if code != http.StatusNotImplemented && code != http.StatusNotFound {
		t.Errorf("POST /api/v1/operator/tenants through nava-core: %d %s, want 501 or 404", code, b)
	}
	if n := psqlSuper(t, fmt.Sprintf("SELECT count(*) FROM tenants WHERE id = '%s'", tenant)); n != "0" {
		t.Errorf("the request through nava-core left %s tenant rows for %s", n, tenant)
	}
}

// TestOnboardKillBeforeBindThenRetryLiveStack (TICK-253): nava-onboard is
// killed with `docker kill` after it created the Keycloak organization and
// before its binding commits, then restarted, and the same key is retried.
//
// The test holds a SHARE lock on tenant_identity_binding, so the binding
// INSERT waits on it: the organization exists, nothing is bound, and the
// process is mid-request with its advisory-lock transaction open through
// PgBouncer. It is killed there. The advisory lock must then be free through
// PgBouncer within lockReleaseBound while the test still holds the table lock
// (the release cannot depend on the stuck INSERT), and the dead INSERT must
// end without committing. The retry gives exactly one organization, one
// binding naming it, one domain and the tenant active.
func TestOnboardKillBeforeBindThenRetryLiveStack(t *testing.T) {
	const lockReleaseBound = 30 * time.Second
	key, tenant, domain, email := newOnboarding(t)
	alias := onboard.Alias(tenant)
	onbURL := onboardDatabaseURL(t)
	// However the test ends, nava-onboard is running and healthy for the next.
	t.Cleanup(func() { startOnboard(t) })

	state := func() string {
		return psqlSuper(t, fmt.Sprintf("SELECT t.state || '|' || (SELECT count(*) FROM tenant_identity_binding WHERE tenant_id = t.id) || '|' || "+
			"(SELECT count(*) FROM tenant_email_domain WHERE tenant_id = t.id) FROM tenants t WHERE t.id = '%s'", tenant))
	}
	// A nava_onboard backend still in its binding INSERT's transaction.
	const bindingInsert = "FROM pg_stat_activity WHERE usename = 'nava_onboard' AND query LIKE 'INSERT INTO tenant_identity_binding%' AND state <> 'idle'"

	releaseTable := holdBindingInserts(t)
	op := operatorToken(t)
	body := onboardBody(email, domain)
	type reply struct {
		code int
		body []byte
	}
	first := make(chan reply, 1)
	go func() {
		code, b, _ := onboardPostRaw(t, op, key, body)
		first <- reply{code, b}
	}()

	// nava-onboard is in Bind: its INSERT waits on the test's table lock.
	within(t, 2*time.Minute, "the binding INSERT waiting on the table lock", func() bool {
		return psqlSuper(t, "SELECT count(*) "+bindingInsert+" AND wait_event_type = 'Lock'") == "1"
	})
	if n := len(orgsByAlias(t, alias)); n != 1 {
		t.Fatalf("before the kill: %d organizations, want 1 (created before the binding)", n)
	}
	// The control: the onboarding holds the tenant's advisory lock, and the
	// check through PgBouncer sees it held.
	if tryOnboardLock(t, onbURL, tenant) {
		t.Fatal("the tenant's advisory lock is free while nava-onboard onboards it: the check proves nothing")
	}

	if out, err := dockerOut(t, "", "kill", containerOf(t, "nava-onboard")); err != nil {
		t.Fatalf("docker kill nava-onboard: %v\n%s", err, out)
	}
	killed := time.Now()
	select {
	case r := <-first:
		if r.code != 0 {
			t.Fatalf("the killed request answered %d %.400s, want no response", r.code, r.body)
		}
	case <-time.After(time.Minute):
		t.Fatal("the request to the killed nava-onboard did not end within 1m")
	}

	// The dead connection's advisory lock is free through PgBouncer, the
	// table lock still held.
	within(t, lockReleaseBound, "the advisory lock acquired through PgBouncer after the kill", func() bool {
		return tryOnboardLock(t, onbURL, tenant)
	})
	t.Logf("advisory lock acquired through PgBouncer %s after the kill", time.Since(killed).Round(10*time.Millisecond))

	// Unblocked, the dead INSERT must end without committing: a backend left
	// idle in its transaction would hold the binding key and block the retry.
	releaseTable()
	within(t, lockReleaseBound, "the dead binding INSERT gone", func() bool {
		return psqlSuper(t, "SELECT count(*) "+bindingInsert) == "0"
	})
	if n := len(orgsByAlias(t, alias)); n != 1 {
		t.Fatalf("after the kill: %d organizations, want 1", n)
	}
	if got := state(); got != "provisioning|0|1" {
		t.Fatalf("after the kill: state|bindings|domains = %q, want provisioning|0|1", got)
	}

	startOnboard(t)
	mustOnboard(t, operatorToken(t), key, body, http.StatusCreated, tenant)
	ids := orgsByAlias(t, alias)
	if len(ids) != 1 {
		t.Fatalf("after the retry: organizations %v, want exactly one", ids)
	}
	if got := state(); got != "active|1|1" {
		t.Errorf("after the retry: state|bindings|domains = %q, want active|1|1", got)
	}
	if got := psqlSuper(t, fmt.Sprintf("SELECT org_id FROM tenant_identity_binding WHERE tenant_id = '%s'", tenant)); got != ids[0] {
		t.Errorf("the binding names organization %q, want %q", got, ids[0])
	}
	if n := orgIdPCount(t, ids[0]); n != 1 {
		t.Errorf("the organization has %d IdPs, want 1", n)
	}
}

// within polls cond every 250ms until it holds, and fails the test after d.
func within(t *testing.T, d time.Duration, what string, cond func() bool) {
	t.Helper()
	for deadline := time.Now().Add(d); !cond(); time.Sleep(250 * time.Millisecond) {
		if time.Now().After(deadline) {
			t.Fatalf("%s: not within %s", what, d)
		}
	}
}

// startOnboard starts nava-onboard (a no-op while it runs) and waits until it
// is healthy.
func startOnboard(t *testing.T) {
	t.Helper()
	if out, err := dockerOut(t, "", "compose", "up", "-d", "--no-deps", "--wait", "nava-onboard"); err != nil {
		t.Errorf("start nava-onboard: %v\n%s", err, out)
	}
}

// onboardDatabaseURL is the nava_onboard URL the running nava-onboard holds; it
// goes through PgBouncer.
func onboardDatabaseURL(t *testing.T) string {
	t.Helper()
	out, err := dockerOut(t, "", "inspect", "-f", "{{json .Config.Env}}", containerOf(t, "nava-onboard"))
	var env []string
	if err != nil || json.Unmarshal([]byte(out), &env) != nil {
		t.Fatalf("env of nava-onboard: %v", err)
	}
	for _, v := range env {
		if name, val, _ := strings.Cut(v, "="); name == "NAVA_ONBOARD_DATABASE_URL" {
			if u, err := url.Parse(val); err != nil || u.Host != "pgbouncer:5432" {
				t.Fatal("nava-onboard's NAVA_ONBOARD_DATABASE_URL does not go through pgbouncer:5432")
			}
			return val
		}
	}
	t.Fatal("nava-onboard has no NAVA_ONBOARD_DATABASE_URL")
	return ""
}

// tryOnboardLock tries the tenant's onboarding advisory lock (PGStore.Lock's
// key) as nava_onboard through PgBouncer, from the postgres container, and
// reports whether it was free; the transaction-scoped lock ends with the
// statement. The URL holds a password, so it goes through the environment.
func tryOnboardLock(t *testing.T, dbURL string, tenant uuid.UUID) bool {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.WithoutCancel(t.Context()), time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "docker", "compose", "exec", "-T", "-e", "NAVA_ONBOARD_DATABASE_URL", "postgres", "sh", "-c",
		`exec psql -X -q -t -A -v ON_ERROR_STOP=1 "$NAVA_ONBOARD_DATABASE_URL" -c "$0"`,
		fmt.Sprintf("SELECT pg_try_advisory_xact_lock(hashtextextended('nava-onboard:%s', 0))", tenant))
	cmd.Dir = repoRoot
	cmd.Env = append(os.Environ(), "NAVA_ONBOARD_DATABASE_URL="+dbURL)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("try the advisory lock through PgBouncer: %v: %s", err, strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(string(out)) == "t"
}

// holdBindingInserts holds a SHARE lock on tenant_identity_binding from a
// superuser session until release (or Cleanup): every INSERT into it waits,
// reads do not. Should the psql client be lost, release terminates the
// backend, and idle_in_transaction_session_timeout ends it in any case.
func holdBindingInserts(t *testing.T) (release func()) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.WithoutCancel(t.Context()))
	cmd := exec.CommandContext(ctx, "docker", "compose", "exec", "-T", "postgres", "sh", "-c",
		`PGPASSWORD=$(cat /run/pg_superuser_password) exec psql -X -q -t -A -v ON_ERROR_STOP=1 -U pgadmin -d nava`)
	cmd.Dir = repoRoot
	in, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("psql: %v", err)
	}
	_, _ = io.WriteString(in, "SET idle_in_transaction_session_timeout = '5min';\nBEGIN;\n"+
		"LOCK TABLE tenant_identity_binding IN SHARE MODE;\nSELECT 'locked|' || pg_backend_pid();\n")
	line := make(chan string, 1)
	go func() {
		defer close(line)
		for s := bufio.NewScanner(stdout); s.Scan(); {
			if l := strings.TrimSpace(s.Text()); l != "" {
				line <- l
				return
			}
		}
	}()
	var pid string
	select {
	case l := <-line:
		var ok bool
		if pid, ok = strings.CutPrefix(l, "locked|"); !ok {
			cancel()
			_ = cmd.Wait()
			t.Fatalf("lock tenant_identity_binding: %q %s", l, strings.TrimSpace(stderr.String()))
		}
	case <-time.After(time.Minute):
		cancel()
		_ = cmd.Wait()
		t.Fatalf("lock tenant_identity_binding: not within 1m: %s", strings.TrimSpace(stderr.String()))
	}
	var once sync.Once
	release = func() {
		once.Do(func() {
			_, _ = io.WriteString(in, "ROLLBACK;\n")
			_ = in.Close()
			done := make(chan struct{})
			go func() { _ = cmd.Wait(); close(done) }()
			select {
			case <-done:
			case <-time.After(30 * time.Second):
				cancel()
				<-done
			}
			cancel()
			psqlSuper(t, "SELECT count(pg_terminate_backend(pid)) FROM pg_stat_activity WHERE pid = "+pid)
		})
	}
	t.Cleanup(release)
	return release
}

// TestOnboardDomainClaimedIsConflict (TICK-271): onboarding a tenant with a
// domain an active tenant owns is 409 domain_claimed, refused before Keycloak:
// tenant A keeps the domain, no organization exists for the refused tenant and
// its tenants row was never left in provisioning.
func TestOnboardDomainClaimedIsConflict(t *testing.T) {
	keyA, tenantA, domain, email := newOnboarding(t)
	op := operatorToken(t)
	mustOnboard(t, op, keyA, onboardBody(email, domain), http.StatusCreated, tenantA)

	keyB := uuid.NewString()
	tenantB := onboard.TenantID(operatorSub, keyB)
	cleanupTenantOrg(t, tenantB, onboard.Alias(tenantB))
	code, resp := onboardPost(t, op, keyB, onboardBody(email, domain))
	if code != http.StatusConflict {
		t.Fatalf("claiming %s: %d %.400s, want 409 domain_claimed", domain, code, resp)
	}
	if owner := psqlSuper(t, fmt.Sprintf("SELECT tenant_id FROM tenant_email_domain WHERE domain = '%s'", domain)); owner != tenantA.String() {
		t.Errorf("%s is owned by %s, want %s", domain, owner, tenantA)
	}
	if orgs := orgsByAlias(t, onboard.Alias(tenantB)); len(orgs) != 0 {
		t.Errorf("organizations for the refused tenant: %v, want none", orgs)
	}
	if n := psqlSuper(t, fmt.Sprintf("SELECT count(*) FROM tenants WHERE id = '%s'", tenantB)); n != "0" {
		t.Errorf("the refused tenant left %s tenants rows, want 0", n)
	}
}

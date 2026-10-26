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
// The tokens are fetched as http://keycloak:8080, the issuer nava-core and
// nava-onboard accept (NAVA_OIDC_ISSUERS): Keycloak names the issuer after the
// request host (KC_HOSTNAME_STRICT=false), so inNet sends that Host header to
// the published port. nava-onboard is never published; the test reaches it
// with `docker compose exec`, as an operator on the host would.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/identity/keycloak"
	"github.com/Navaladi2019/TicketingSystem/internal/onboard"
)

const (
	// inNetHost is Keycloak as the platform containers reach it.
	inNetHost   = "keycloak:8080"
	inNetIssuer = "http://" + inNetHost + "/realms/nava"
	// operatorSub is operator@nava.test's fixed id in the realm export, the
	// subject on NAVA_OPERATORS in compose.yaml.
	operatorSub  = "00000000-0000-4000-8000-00000000f001"
	operatorPass = "dev-only-operator"
	// The onboarding account's secret and the nava_onboard database URL's
	// user: neither may appear in nava-core's environment.
	onboardSecret = "dev-only-onboarding-secret"
	onboardDBUser = "nava_onboard:"
)

// repoRoot is the repository root, from this package; docker compose runs
// there so COMPOSE_FILE and COMPOSE_PROJECT_NAME from make apply.
const repoRoot = "../.."

func coreURL() string {
	if u := os.Getenv("NAVA_TEST_CORE_URL"); u != "" {
		return strings.TrimRight(u, "/")
	}
	if p := os.Getenv("NAVA_CORE_PORT"); p != "" {
		return "http://localhost:" + p
	}
	return "http://localhost:18080"
}

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

// inNet is a client whose requests for keycloak:8080 (and for realmHost)
// go to the Keycloak under test with that Host header kept, so issuers,
// redirects and cookies are the ones the platform containers see.
type inNet struct{ host string }

func (s inNet) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Host == inNetHost || req.URL.Host == realmHost {
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
// treats only localhost as a secure origin, not keycloak:8080.
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
	return &http.Client{Jar: secureJar{jar}, Timeout: 30 * time.Second, Transport: inNet{host: u.Host}}
}

func inNetToken(t *testing.T, c *http.Client, form url.Values) string {
	t.Helper()
	req, _ := http.NewRequestWithContext(t.Context(), http.MethodPost,
		inNetIssuer+"/protocol/openid-connect/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	code, body := do(t, c, req)
	var out struct {
		AccessToken string `json:"access_token"`
	}
	if code != http.StatusOK || json.Unmarshal(body, &out) != nil || out.AccessToken == "" {
		t.Fatalf("token as %s: %d %.300s", inNetHost, code, body)
	}
	if iss, _ := claims(t, out.AccessToken)["iss"].(string); iss != inNetIssuer {
		t.Fatalf("token iss = %q, want %q", iss, inNetIssuer)
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
	ctx, cancel := context.WithTimeout(context.WithoutCancel(t.Context()), 3*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "docker", "compose", "exec", "-T", "nava-onboard", "wget", "-S", "-T", "120", "-O", "-",
		"--header", "Authorization: Bearer "+token, "--header", "Idempotency-Key: "+key,
		"--header", "Content-Type: application/json", "--post-data", string(body),
		"http://127.0.0.1:8081/api/v1/operator/tenants")
	cmd.Dir = repoRoot
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	_ = cmd.Run() // non-zero for any error status; the status line decides
	m := statusLine.FindAllStringSubmatch(stderr.String(), -1)
	if m == nil {
		t.Fatalf("no HTTP response from nava-onboard: %.400s", stderr.String())
	}
	code, _ := strconv.Atoi(m[len(m)-1][1])
	return code, stdout.Bytes()
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
			"redirectUris": []string{"http://" + inNetHost + "/realms/nava/broker/*"},
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
// Only the URLs change: alias, organization link, first-login flow and the
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
	cfg["authorizationUrl"] = "http://" + inNetHost + "/realms/" + realm + "/protocol/openid-connect/auth"
	cfg["tokenUrl"] = "http://localhost:8080/realms/" + realm + "/protocol/openid-connect/token"
	cfg["jwksUrl"] = "http://localhost:8080/realms/" + realm + "/protocol/openid-connect/certs"
	if code, b := adminRealm(t, "nava", http.MethodPut, "/identity-provider/instances/"+url.PathEscape(alias), idp); code != http.StatusNoContent {
		t.Fatalf("repoint IdP %s: %d %s", alias, code, b)
	}
}

var pageText = regexp.MustCompile(`(?s)<[^>]*>`)

// brokeredSignIn runs the code flow of nava-web as keycloak:8080 with the
// tenant IdP hinted, signs in at the IdP, and exchanges the code. It fails
// with the text of the last page when Keycloak does not return a code.
func brokeredSignIn(t *testing.T, idp, user, pass string) string {
	t.Helper()
	c := newInNetClient(t)
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
		strings.NewReader(url.Values{"username": {user}, "password": {pass}}.Encode()))
	post.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	_, page := do(t, c, post)
	if final == nil || final.Query().Get("code") == "" {
		text := strings.Join(strings.Fields(html.UnescapeString(pageText.ReplaceAllString(string(page), " "))), " ")
		t.Fatalf("the brokered sign-in returned no code; last page: %.600s", text)
	}
	return inNetToken(t, c, url.Values{
		"grant_type": {"authorization_code"}, "client_id": {"nava-web"},
		"code": {final.Query().Get("code")}, "redirect_uri": {redirect},
	})
}

// groupsyncOnce runs one nava-groupsync cycle, which syncs every bound
// tenant's principals; it exits non-zero when any tenant failed.
func groupsyncOnce(t *testing.T) {
	t.Helper()
	if out, err := dockerOut(t, "", "compose", "run", "--rm", "--no-deps", "-T", "nava-groupsync", "--once"); err != nil {
		t.Fatalf("nava-groupsync --once: %v\n%s", err, out)
	}
}

type me struct {
	UserID   uuid.UUID   `json:"user_id"`
	TenantID uuid.UUID   `json:"tenant_id"`
	Groups   []uuid.UUID `json:"groups"`
	IdPAlias string      `json:"idp_alias"`
}

func getMe(t *testing.T, token, query string, header http.Header) (int, me, []byte) {
	t.Helper()
	req, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, coreURL()+"/api/v1/me"+query, nil)
	for k, v := range header {
		req.Header[k] = v
	}
	req.Header.Set("Authorization", "Bearer "+token)
	code, b := do(t, &http.Client{Timeout: 30 * time.Second}, req)
	var m me
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
			"FROM tenants t JOIN tenant_identity_binding b ON b.tenant_id = t.id WHERE t.id = '%s'", tenant)); got != "active|"+inNetIssuer+"|"+orgs[0]+"|"+idpAlias {
		t.Fatalf("tenant row and binding = %q, want active, bound to %s %s with IdP %s", got, inNetIssuer, orgs[0], idpAlias)
	}

	pointIdPAt(t, idpAlias, idpRealm)
	tok := brokeredSignIn(t, idpAlias, email, pass)
	c := claims(t, tok)
	orgClaim(t, c, alias) // exactly one organization, the new tenant's

	// Ruling 2026-10-25 on TICK-81: the admin is both an org member (the
	// first-login flow) and in the tenant group (the invite), as one user.
	sub, _ := c["sub"].(string)
	invited := kcUserByEmail(t, email)
	if invited == "" || sub != invited {
		t.Fatalf("token sub = %q, the user with %s = %q: the sign-in did not land on the invited user", sub, email, invited)
	}
	if !userInGroup(t, invited, kcGroupID(t, "/t/"+tenant.String())) {
		t.Errorf("the admin is not a member of /t/%s", tenant)
	}
	if !slices.Contains(orgMembers(t, orgs[0]), invited) {
		t.Errorf("the admin is not a member of organization %s", alias)
	}

	groupsyncOnce(t)

	code, got, raw := getMe(t, tok, "", nil)
	if code != http.StatusOK {
		t.Fatalf("GET /api/v1/me: %d %s", code, raw)
	}
	if got.TenantID != tenant || got.IdPAlias != idpAlias || got.UserID == uuid.Nil {
		t.Errorf("/me = %s, want tenant %s via %s", raw, tenant, idpAlias)
	}

	// Security condition: the token resolves to the new tenant only, whatever
	// tenant the request names.
	other := uuid.NewString()
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
	holdsOnboarding := func(vs []string) []string {
		var bad []string
		for _, v := range vs {
			name, val, _ := strings.Cut(v, "=")
			if strings.HasPrefix(name, "NAVA_ONBOARD_") || strings.Contains(val, onboardSecret) || strings.Contains(val, onboardDBUser) {
				bad = append(bad, name)
			}
		}
		return bad
	}
	if bad := holdsOnboarding(env(core)); len(bad) != 0 {
		t.Errorf("nava-core's environment holds onboarding variables or secrets: %v", bad)
	}
	// The check's control: the same scan finds them in nava-onboard.
	if bad := holdsOnboarding(env(onb)); !slices.Contains(bad, "NAVA_ONBOARD_CLIENT_SECRET") || !slices.Contains(bad, "NAVA_ONBOARD_DATABASE_URL") {
		t.Errorf("the scan does not find the onboarding secrets in nava-onboard (%v): it proves nothing about nava-core", bad)
	}

	out, err := dockerOut(t, "", "inspect", "-f", "{{json .NetworkSettings.Ports}}|{{range $k, $v := .NetworkSettings.Networks}}{{$k}} {{end}}", onb)
	if err != nil {
		t.Fatal(err)
	}
	ports, nets, _ := strings.Cut(strings.TrimSpace(out), "|")
	var bindings map[string][]struct{ HostPort string }
	_ = json.Unmarshal([]byte(ports), &bindings)
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

// TestOnboardKillBeforeBindThenRetryLiveStack: the Keycloak organization is
// created, then the binding insert fails (a trigger stands in for the process
// dying there; the transaction never commits). The retry with the same key,
// against the real nava_onboard PGStore and Keycloak, ends with exactly one
// organization, one binding naming it, and the tenant active.
func TestOnboardKillBeforeBindThenRetryLiveStack(t *testing.T) {
	key, tenant, domain, email := newOnboarding(t)
	alias := onboard.Alias(tenant)
	fn := "tick234_kill_" + strings.ReplaceAll(tenant.String()[:8], "-", "")
	psqlSuper(t, "CREATE FUNCTION public."+fn+"() RETURNS trigger LANGUAGE plpgsql AS $f$ BEGIN RAISE EXCEPTION 'killed before the binding'; END $f$")
	drop := func() {
		psqlSuper(t, "DROP TRIGGER IF EXISTS "+fn+" ON tenant_identity_binding")
		psqlSuper(t, "DROP FUNCTION IF EXISTS public."+fn+"()")
	}
	t.Cleanup(drop)
	psqlSuper(t, fmt.Sprintf("CREATE TRIGGER %s BEFORE INSERT ON tenant_identity_binding FOR EACH ROW WHEN (NEW.tenant_id = '%s') EXECUTE FUNCTION public.%s()", fn, tenant, fn))

	op := operatorToken(t)
	body := onboardBody(email, domain)
	code, resp := onboardPost(t, op, key, body)
	if code < 500 {
		t.Fatalf("first call: %d %s, want 5xx at the binding", code, resp)
	}
	if n := len(orgsByAlias(t, alias)); n != 1 {
		t.Fatalf("after the kill: %d organizations, want 1 (created before the binding)", n)
	}
	state := func() string {
		return psqlSuper(t, fmt.Sprintf("SELECT t.state || '|' || (SELECT count(*) FROM tenant_identity_binding WHERE tenant_id = t.id) || '|' || "+
			"(SELECT count(*) FROM tenant_email_domain WHERE tenant_id = t.id) FROM tenants t WHERE t.id = '%s'", tenant))
	}
	if got := state(); got != "provisioning|0|0" {
		t.Fatalf("after the kill: state|bindings|domains = %q, want provisioning|0|0", got)
	}

	drop()
	mustOnboard(t, op, key, body, http.StatusCreated, tenant)
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

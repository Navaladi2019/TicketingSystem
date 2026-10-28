//go:build identitystack

// Package identitystack tests the realm `nava` that Keycloak imports from
// deploy/compose/keycloak/import (T14, ADR-001 A.1) against the real Keycloak
// of the compose stack, over plain HTTP on the published loopback port.
//
//	docker compose up -d keycloak
//	go test -tags identitystack -count=1 ./internal/identitystack
//
// NAVA_TEST_KEYCLOAK_URL, else KEYCLOAK_PORT, overrides http://localhost:18081.
package identitystack

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"regexp"
	"slices"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/identity/keycloak"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
)

// realmHost is the browser-facing Keycloak address the realm export names in
// the brokered IdPs' authorizationUrl.
const realmHost = "localhost:18081"

func baseURL() string {
	if u := os.Getenv("NAVA_TEST_KEYCLOAK_URL"); u != "" {
		return strings.TrimRight(u, "/")
	}
	if p := os.Getenv("KEYCLOAK_PORT"); p != "" {
		return "http://localhost:" + p
	}
	return "http://" + realmHost
}

// toStack sends requests for realmHost to the Keycloak under test, so a stack
// on another port (make ci-isolated) never follows a broker redirect into the
// default stack's Keycloak.
type toStack struct{ host string }

func (s toStack) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Host == realmHost && s.host != realmHost {
		req = req.Clone(req.Context())
		req.URL.Host = s.host
		req.Host = s.host
	}
	return http.DefaultTransport.RoundTrip(req)
}

func newClient(t *testing.T) *http.Client {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(baseURL())
	if err != nil {
		t.Fatal(err)
	}
	return &http.Client{Jar: jar, Timeout: 30 * time.Second, Transport: toStack{host: u.Host}}
}

func do(t *testing.T, c *http.Client, req *http.Request) (int, []byte) {
	t.Helper()
	resp, err := c.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", req.Method, req.URL, err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, b
}

func tokenForm(t *testing.T, realm string, form url.Values) (int, map[string]any) {
	t.Helper()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost,
		baseURL()+"/realms/"+realm+"/protocol/openid-connect/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	code, body := do(t, newClient(t), req)
	var out map[string]any
	_ = json.Unmarshal(body, &out)
	return code, out
}

func mustToken(t *testing.T, form url.Values) string {
	t.Helper()
	code, out := tokenForm(t, "nava", form)
	if code != http.StatusOK {
		t.Fatalf("token request: %d %v", code, out)
	}
	return out["access_token"].(string)
}

func passwordToken(t *testing.T, client, user, pass string) string {
	return mustToken(t, url.Values{
		"grant_type": {"password"}, "client_id": {client}, "username": {user}, "password": {pass}, "scope": {"openid"},
	})
}

func serviceToken(t *testing.T) string {
	return mustToken(t, url.Values{
		"grant_type": {"client_credentials"}, "client_id": {"nava-platform"}, "client_secret": {"dev-only-platform-secret"},
	})
}

// masterToken is the bootstrap admin of the master realm (KEYCLOAK_ADMIN and
// KEYCLOAK_ADMIN_PASSWORD of the compose file); only the realm export uses it,
// because exporting clients needs more than the platform service account holds.
func masterToken(t *testing.T) string {
	t.Helper()
	user, pass := os.Getenv("KEYCLOAK_ADMIN"), os.Getenv("KEYCLOAK_ADMIN_PASSWORD")
	if user == "" {
		user = "admin"
	}
	if pass == "" {
		pass = "dev-only-admin"
	}
	code, out := tokenForm(t, "master", url.Values{
		"grant_type": {"password"}, "client_id": {"admin-cli"}, "username": {user}, "password": {pass},
	})
	if code != http.StatusOK {
		t.Fatalf("master admin token: %d %v", code, out)
	}
	return out["access_token"].(string)
}

func claims(t *testing.T, jwt string) map[string]any {
	t.Helper()
	parts := strings.Split(jwt, ".")
	if len(parts) != 3 {
		t.Fatalf("not a JWT: %q", jwt)
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	return m
}

func admin(t *testing.T, tok, method, path string, body any) (int, []byte) {
	t.Helper()
	var rdr io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rdr = bytes.NewReader(b)
	}
	req, _ := http.NewRequestWithContext(context.Background(), method, baseURL()+"/admin/realms/nava"+path, rdr)
	req.Header.Set("Authorization", "Bearer "+tok)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return do(t, newClient(t), req)
}

func userID(t *testing.T, tok, username string) string {
	t.Helper()
	code, b := admin(t, tok, http.MethodGet, "/users?exact=true&username="+url.QueryEscape(username), nil)
	var us []struct{ ID string }
	if code != http.StatusOK || json.Unmarshal(b, &us) != nil || len(us) != 1 {
		t.Fatalf("lookup %s: %d %s", username, code, b)
	}
	return us[0].ID
}

func orgClaim(t *testing.T, c map[string]any, alias string) map[string]any {
	t.Helper()
	org, ok := c["organization"].(map[string]any)
	if !ok || len(org) != 1 {
		t.Fatalf("organization claim = %v, want exactly one key", c["organization"])
	}
	entry, ok := org[alias].(map[string]any)
	if !ok {
		t.Fatalf("organization claim = %v, want key %q", org, alias)
	}
	return entry
}

// orgID reads the id of an organization through the admin API.
func orgID(t *testing.T, tok, alias string) string {
	t.Helper()
	code, b := admin(t, tok, http.MethodGet, "/organizations?exact=true&search="+alias, nil)
	var os []struct{ ID, Alias string }
	if code != http.StatusOK || json.Unmarshal(b, &os) != nil {
		t.Fatalf("organizations: %d %s", code, b)
	}
	for _, o := range os {
		if o.Alias == alias {
			return o.ID
		}
	}
	t.Fatalf("organization %s not found", alias)
	return ""
}

func TestAliceTokenCarriesOrganization(t *testing.T) {
	c := claims(t, passwordToken(t, "nava-web", "alice@tenant-a.test", "dev-only-alice"))
	entry := orgClaim(t, c, "tenant-a")
	want := orgID(t, masterToken(t), "tenant-a") // the runtime account has no organization read
	if entry["id"] != want {
		t.Fatalf("organization.tenant-a.id = %v, want %s", entry["id"], want)
	}
	if len(entry) != 1 {
		t.Fatalf("organization.tenant-a = %v, want only id (addOrganizationAttributes=false)", entry)
	}
	if _, ok := c["idp"]; ok {
		t.Fatalf("a password login must carry no idp claim, got %v", c["idp"])
	}
}

func TestBobIsScopedToTenantB(t *testing.T) {
	c := claims(t, passwordToken(t, "nava-web", "bob@tenant-b.test", "dev-only-bob"))
	orgClaim(t, c, "tenant-b")
}

var formAction = regexp.MustCompile(`<form[^>]*id="kc-form-login"[^>]*action="([^"]+)"`)
var formActionAlt = regexp.MustCompile(`<form[^>]*action="([^"]+)"[^>]*id="kc-form-login"`)

// brokeredToken signs in through the tenant's IdP (a second realm) with the
// authorization-code flow and returns the access token of realm nava.
func brokeredToken(t *testing.T, idp, user, pass string) string {
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
	code, page := do(t, c, req)
	if code != http.StatusOK {
		t.Fatalf("broker redirect ended at %d: %s", code, page)
	}
	m := formAction.FindSubmatch(page)
	if m == nil {
		m = formActionAlt.FindSubmatch(page)
	}
	if m == nil {
		t.Fatalf("no login form from the IdP realm: %.400s", page)
	}
	post, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, html.UnescapeString(string(m[1])),
		strings.NewReader(url.Values{"username": {user}, "password": {pass}}.Encode()))
	post.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	code, body := do(t, c, post)
	if final == nil {
		t.Fatalf("broker login did not return to the redirect uri: %d %.400s", code, body)
	}
	authCode := final.Query().Get("code")
	if authCode == "" {
		t.Fatalf("no code in %s", final)
	}
	status, out := tokenForm(t, "nava", url.Values{
		"grant_type": {"authorization_code"}, "client_id": {"nava-web"}, "code": {authCode}, "redirect_uri": {redirect},
	})
	if status != http.StatusOK {
		t.Fatalf("code exchange: %d %v", status, out)
	}
	return out["access_token"].(string)
}

func TestBrokeredTokenCarriesOrganizationAndIdp(t *testing.T) {
	for _, tc := range []struct{ idp, tenant, user, pass string }{
		{"idp-tenant-a", "tenant-a", "carol@tenant-a.test", "dev-only-carol"},
		{"idp-tenant-b", "tenant-b", "dave@tenant-b.test", "dev-only-dave"},
	} {
		t.Run(tc.tenant, func(t *testing.T) {
			c := claims(t, brokeredToken(t, tc.idp, tc.user, tc.pass))
			orgClaim(t, c, tc.tenant)
			if c["idp"] != tc.idp {
				t.Fatalf("idp = %v, want %s", c["idp"], tc.idp)
			}
		})
	}
}

func TestAccessTokenLifespanAtMostFiveMinutes(t *testing.T) {
	for name, tok := range map[string]string{
		"alice":    passwordToken(t, "nava-web", "alice@tenant-a.test", "dev-only-alice"),
		"operator": passwordToken(t, "nava-operator-cli", "operator@nava.test", "dev-only-operator"),
		"platform": serviceToken(t),
	} {
		c := claims(t, tok)
		exp, iat := c["exp"].(float64), c["iat"].(float64)
		if d := exp - iat; d > 300 || d <= 0 {
			t.Errorf("%s: exp-iat = %v, want 0 < d <= 300", name, d)
		}
	}
}

func TestOperatorToken(t *testing.T) {
	tok := passwordToken(t, "nava-operator-cli", "operator@nava.test", "dev-only-operator")
	c := claims(t, tok)
	if !audHas(t, tok, "nava-operator-api") || audHas(t, tok, "nava-api") {
		t.Errorf("aud = %v, want nava-operator-api and not nava-api", c["aud"])
	}
	roles := fmt.Sprint(c["realm_access"])
	if !strings.Contains(roles, "nava-operator") {
		t.Errorf("realm_access = %v, want nava-operator", c["realm_access"])
	}
	// A tenant user holds neither the role nor the audience.
	at := passwordToken(t, "nava-web", "alice@tenant-a.test", "dev-only-alice")
	a := claims(t, at)
	if audHas(t, at, "nava-operator-api") || strings.Contains(fmt.Sprint(a["realm_access"]), "nava-operator") {
		t.Errorf("alice token has operator aud or role: aud=%v realm_access=%v", a["aud"], a["realm_access"])
	}
}

func TestOrgAdminCannotAssignOperator(t *testing.T) {
	svc := serviceToken(t)
	// The runtime account has no realm role, so the role is read as the
	// master admin.
	code, b := admin(t, masterToken(t), http.MethodGet, "/roles/nava-operator", nil)
	if code != http.StatusOK {
		t.Fatalf("read role: %d %s", code, b)
	}
	var role map[string]any
	_ = json.Unmarshal(b, &role)

	adminTok := passwordToken(t, "nava-web", "admin@tenant-a.test", "dev-only-admin-a")
	if !strings.Contains(fmt.Sprint(claims(t, adminTok)["realm_access"]), "tenant-admin") {
		t.Fatalf("admin@tenant-a.test lacks tenant-admin")
	}
	// Every user of the realm is a target, the admin and a tenant-b user included.
	for _, target := range []string{"alice@tenant-a.test", "admin@tenant-a.test", "bob@tenant-b.test"} {
		id := userID(t, svc, target)
		// The org admin's token reaches no realm-management route.
		for _, call := range []struct{ method, path string }{
			{http.MethodPost, "/users/" + id + "/role-mappings/realm"},
			{http.MethodGet, "/roles/nava-operator"},
			{http.MethodPost, "/groups"},
		} {
			var body any = []map[string]any{role}
			if call.path == "/groups" {
				body = map[string]any{"name": "evil"}
			}
			code, b := admin(t, adminTok, call.method, call.path, body)
			if code != http.StatusForbidden && code != http.StatusUnauthorized {
				t.Errorf("org admin %s %s: %d %s, want 401/403", call.method, call.path, code, b)
			}
		}
		// And the role is still unassigned (alice, bob); the admin never had it.
		code, b := admin(t, svc, http.MethodGet, "/users/"+id+"/role-mappings/realm", nil)
		if code != http.StatusOK || strings.Contains(string(b), "nava-operator") {
			t.Errorf("%s: role mappings %d %s must not hold nava-operator", target, code, b)
		}
	}
	// Account console: no role-mapping route in the account API.
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost,
		baseURL()+"/realms/nava/account/roles", bytes.NewReader([]byte(`[{"name":"nava-operator"}]`)))
	req.Header.Set("Authorization", "Bearer "+adminTok)
	req.Header.Set("Content-Type", "application/json")
	if code, b := do(t, newClient(t), req); code < 400 {
		t.Errorf("account API accepted a role assignment: %d %s", code, b)
	}
}

func TestPlatformServiceAccountCanAssignOperator(t *testing.T) {
	svc := serviceToken(t)
	// The role is read as the master admin: the runtime account has no
	// realm role, but its manage-users can still map the role it is handed.
	code, b := admin(t, masterToken(t), http.MethodGet, "/roles/nava-operator", nil)
	if code != http.StatusOK {
		t.Fatalf("%d %s", code, b)
	}
	var role map[string]any
	_ = json.Unmarshal(b, &role)
	id := userID(t, svc, "bob@tenant-b.test")
	path := "/users/" + id + "/role-mappings/realm"
	if code, b = admin(t, svc, http.MethodPost, path, []map[string]any{role}); code != http.StatusNoContent {
		t.Fatalf("assign: %d %s", code, b)
	}
	t.Cleanup(func() {
		if code, b := admin(t, svc, http.MethodDelete, path, []map[string]any{role}); code != http.StatusNoContent {
			t.Errorf("cleanup: could not remove nava-operator from bob: %d %s", code, b)
		}
	})

	// AR-030: the mapping gets bob a real operator token, and ResolveOperator
	// still refuses it, because bob is not on the operator allow-list.
	tok := passwordToken(t, "nava-operator-cli", "bob@tenant-b.test", "dev-only-bob")
	c := claims(t, tok)
	if !strings.Contains(fmt.Sprint(c["realm_access"]), "nava-operator") || !audHas(t, tok, "nava-operator-api") {
		t.Fatalf("bob's token is not an operator token: realm_access=%v aud=%v", c["realm_access"], c["aud"])
	}
	a := stackAdapter(t, tok)
	if _, err := a.ResolveOperator(t.Context(), tok); !errors.Is(err, identity.ErrInvalidToken) {
		t.Errorf("ResolveOperator(role-mapped, not allow-listed) = %v, want ErrInvalidToken", err)
	}
	op := passwordToken(t, "nava-operator-cli", "operator@nava.test", "dev-only-operator")
	if _, err := a.ResolveOperator(t.Context(), op); err != nil {
		t.Errorf("ResolveOperator(allow-listed operator) = %v", err)
	}
}

// stackOperatorID is the Keycloak id of operator@nava.test, pinned in the
// realm export so the stack's operator allow-list is platform config, not a
// lookup in the IdP it guards (AR-030).
const stackOperatorID = "00000000-0000-4000-8000-00000000f001"

// TestOperatorRoleMembersAllowListed is AR-030's live check: every user who
// holds the realm role nava-operator, directly or through a group, is on the
// stack's operator allow-list. A role mapping made outside the platform fails
// it.
func TestOperatorRoleMembersAllowListed(t *testing.T) {
	allow := []string{stackOperatorID}
	adm := masterToken(t)
	code, b := admin(t, adm, http.MethodGet, "/roles/nava-operator/users?max=1000", nil)
	var users []struct{ ID, Username string }
	if code != http.StatusOK || json.Unmarshal(b, &users) != nil {
		t.Fatalf("nava-operator users: %d %s", code, b)
	}
	if len(users) == 0 {
		t.Fatal("nava-operator has no members: the check would pass vacuously")
	}
	for _, u := range users {
		if !slices.Contains(allow, u.ID) {
			t.Errorf("nava-operator member %s (%s) is not on the operator allow-list", u.Username, u.ID)
		}
	}
	code, b = admin(t, adm, http.MethodGet, "/roles/nava-operator/groups", nil)
	if code != http.StatusOK {
		t.Fatalf("nava-operator groups: %d %s", code, b)
	}
	if strings.TrimSpace(string(b)) != "[]" {
		t.Errorf("nava-operator is mapped to groups, whose members bypass this check: %s", b)
	}
}

// TestOperatorRoleIsNoComposite closes the gap TestOperatorRoleMembersAllowListed
// leaves (TICK-219): a role that lists nava-operator as a composite hands it to
// every holder, and /roles/nava-operator/users does not list them. With it in
// default-roles-nava, every user of the realm would carry the role. A chain
// ends in a role that holds nava-operator directly, so checking each role's
// direct composites covers the transitive case. The probes prove the check
// sees realm and client roles.
func TestOperatorRoleIsNoComposite(t *testing.T) {
	adm := masterToken(t)
	if got := operatorHolders(t, adm); len(got) != 0 {
		t.Errorf("roles with nava-operator as a composite, whose holders bypass the allow-list check: %v", got)
	}

	code, b := admin(t, adm, http.MethodGet, "/roles/nava-operator", nil)
	var op map[string]any
	if code != http.StatusOK || json.Unmarshal(b, &op) != nil {
		t.Fatalf("nava-operator: %d %s", code, b)
	}
	code, b = admin(t, adm, http.MethodGet, "/clients?clientId=nava-platform", nil)
	var cs []struct{ ID string }
	if code != http.StatusOK || json.Unmarshal(b, &cs) != nil || len(cs) != 1 {
		t.Fatalf("client nava-platform: %d %s", code, b)
	}
	// A probe role grants nothing: no user or group holds it.
	name := fmt.Sprintf("tick219-probe-%d", time.Now().UnixNano())
	for _, base := range []string{"/roles", "/clients/" + cs[0].ID + "/roles"} {
		if code, b := admin(t, adm, http.MethodPost, base, map[string]any{"name": name}); code != http.StatusCreated {
			t.Fatalf("create probe %s/%s: %d %s", base, name, code, b)
		}
		code, b := admin(t, adm, http.MethodGet, base+"/"+name, nil)
		var probe struct{ ID string }
		if code != http.StatusOK || json.Unmarshal(b, &probe) != nil {
			t.Fatalf("probe %s: %d %s", base, code, b)
		}
		t.Cleanup(func() {
			if code, b := admin(t, adm, http.MethodDelete, "/roles-by-id/"+probe.ID, nil); code != http.StatusNoContent {
				t.Errorf("cleanup: could not delete probe role %s: %d %s", probe.ID, code, b)
			}
		})
		if code, b := admin(t, adm, http.MethodPost, "/roles-by-id/"+probe.ID+"/composites", []map[string]any{op}); code != http.StatusNoContent {
			t.Fatalf("make nava-operator a composite of probe %s: %d %s", base, code, b)
		}
		if got := operatorHolders(t, adm); !slices.Contains(got, probe.ID) {
			t.Errorf("probe %s/%s holds nava-operator as a composite and the check missed it: %v", base, name, got)
		}
	}
}

type kcRole struct{ ID, Name string }

// operatorHolders returns the ids of the realm and client roles of `nava`
// that list the realm role nava-operator among their direct composites.
func operatorHolders(t *testing.T, adm string) []string {
	t.Helper()
	list := func(path string) []kcRole {
		code, b := admin(t, adm, http.MethodGet, path, nil)
		var out []kcRole
		if code != http.StatusOK || json.Unmarshal(b, &out) != nil {
			t.Fatalf("%s: %d %s", path, code, b)
		}
		return out
	}
	code, b := admin(t, adm, http.MethodGet, "/roles/nava-operator", nil)
	var op struct{ ID string }
	if code != http.StatusOK || json.Unmarshal(b, &op) != nil || op.ID == "" {
		t.Fatalf("nava-operator: %d %s", code, b)
	}
	roles := list("/roles?max=10000")
	if !slices.ContainsFunc(roles, func(r kcRole) bool { return r.Name == "default-roles-nava" }) {
		t.Fatalf("default-roles-nava not listed: the realm role listing is incomplete")
	}
	for _, c := range list("/clients?max=10000") {
		roles = append(roles, list("/clients/"+c.ID+"/roles?max=10000")...)
	}
	var out []string
	for _, r := range roles {
		if slices.ContainsFunc(list("/roles-by-id/"+r.ID+"/composites"), func(c kcRole) bool { return c.ID == op.ID }) {
			out = append(out, r.ID)
		}
	}
	return out
}

// stackAdapter is the Keycloak adapter trusting tok's issuer, with the
// stack's operator allow-list.
func stackAdapter(t *testing.T, tok string) *keycloak.Adapter {
	t.Helper()
	return stackAdapterWithStore(t, tok, nil)
}

// stackAdapterWithStore is stackAdapter resolving tenants through store.
func stackAdapterWithStore(t *testing.T, tok string, store identity.Store) *keycloak.Adapter {
	t.Helper()
	iss, _ := claims(t, tok)["iss"].(string)
	if iss == "" {
		t.Fatal("token has no iss")
	}
	cfg, err := config.Load(func(string) (string, bool) { return "", false })
	if err != nil {
		t.Fatal(err)
	}
	tel, err := telemetry.New(t.Context(), cfg.Telemetry)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tel.Shutdown(context.Background()) })
	out, err := telemetry.NewOutboundClient(tel, telemetry.OutboundOptions{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	a, err := keycloak.New(t.Context(), keycloak.Config{
		Issuers:       []string{iss},
		FetchBaseURL:  baseURL(),
		MeterProvider: tel.MeterProvider,
		Operators:     []keycloak.OperatorSubject{{Issuer: iss, Subject: stackOperatorID}},
		Store:         store,
	}, out)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = a.Close(context.Background()) })
	return a
}

func TestGroupSubtree(t *testing.T) {
	svc := serviceToken(t)
	for _, tenant := range []string{"00000000-0000-4000-8000-0000000000a1", "00000000-0000-4000-8000-0000000000b1"} {
		code, b := admin(t, svc, http.MethodGet, "/group-by-path/t/"+tenant+"/ou/eng/platform", nil)
		if code != http.StatusOK {
			t.Errorf("group /t/%s/ou/eng/platform: %d %s", tenant, code, b)
		}
	}
}

// TestRealmRoundTrip exports the running realm and compares the keys the
// contract asserts with the import file.
func TestRealmRoundTrip(t *testing.T) {
	raw, err := os.ReadFile("../../deploy/compose/keycloak/import/realm-nava.json")
	if err != nil {
		t.Fatal(err)
	}
	var want map[string]any
	if err := json.Unmarshal(raw, &want); err != nil {
		t.Fatal(err)
	}
	code, b := admin(t, masterToken(t), http.MethodPost, "/partial-export?exportClients=true&exportGroupsAndRoles=true", nil)
	if code != http.StatusOK {
		t.Fatalf("partial-export: %d %s", code, b)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"realm", "organizationsEnabled", "accessTokenLifespan"} {
		if fmt.Sprint(got[k]) != fmt.Sprint(want[k]) {
			t.Errorf("%s: exported %v, imported %v", k, got[k], want[k])
		}
	}
	// Mapper config of the client scopes.
	for _, ws := range want["clientScopes"].([]any) {
		wsm := ws.(map[string]any)
		gs := find(got["clientScopes"], "name", wsm["name"])
		if gs == nil {
			t.Errorf("client scope %v not exported", wsm["name"])
			continue
		}
		for _, wm := range wsm["protocolMappers"].([]any) {
			wmm := wm.(map[string]any)
			gm := find(gs["protocolMappers"], "name", wmm["name"])
			if gm == nil {
				t.Errorf("mapper %v not exported", wmm["name"])
				continue
			}
			if gm["protocolMapper"] != wmm["protocolMapper"] {
				t.Errorf("mapper %v type %v != %v", wmm["name"], gm["protocolMapper"], wmm["protocolMapper"])
			}
			for _, k := range []string{"addOrganizationId", "addOrganizationAttributes", "addOrganizationDomain", "multivalued", "jsonType.label", "user.session.note", "claim.name", "access.token.claim", "id.token.claim", "included.client.audience"} {
				if wv, ok := wmm["config"].(map[string]any)[k]; ok && wv != gm["config"].(map[string]any)[k] {
					t.Errorf("mapper %v config %s: exported %v, imported %v", wmm["name"], k, gm["config"].(map[string]any)[k], wv)
				}
			}
		}
	}
	// Identity providers and the broker flow.
	for _, wi := range want["identityProviders"].([]any) {
		wim := wi.(map[string]any)
		gi := find(got["identityProviders"], "alias", wim["alias"])
		if gi == nil {
			t.Errorf("idp %v not exported", wim["alias"])
			continue
		}
		for _, k := range []string{"trustEmail", "firstBrokerLoginFlowAlias", "linkOnly"} {
			if fmt.Sprint(gi[k]) != fmt.Sprint(wim[k]) {
				t.Errorf("idp %v %s: exported %v, imported %v", wim["alias"], k, gi[k], wim[k])
			}
		}
	}
	gf := find(got["authenticationFlows"], "alias", "nava-first-broker-login")
	if gf == nil {
		t.Fatal("flow nava-first-broker-login not exported")
	}
	var seq []string
	for _, e := range gf["authenticationExecutions"].([]any) {
		em := e.(map[string]any)
		seq = append(seq, fmt.Sprintf("%v=%v", em["authenticator"], em["requirement"]))
	}
	if g := strings.Join(seq, ","); g != "idp-create-user-if-unique=REQUIRED,idp-add-organization-member=REQUIRED" {
		t.Errorf("broker flow = %s", g)
	}
	// Clients and realm roles.
	for _, id := range []string{"nava-web", "nava-api", "nava-operator-api", "nava-operator-cli", "nava-platform"} {
		if find(got["clients"], "clientId", id) == nil {
			t.Errorf("client %s not exported", id)
		}
	}
	roundTripClients(t, want, got)
	roundTripOrganizations(t, want)
	roles := got["roles"].(map[string]any)["realm"]
	if find(roles, "name", "nava-operator") == nil {
		t.Error("realm role nava-operator not exported")
	}
}

func find(list any, key string, val any) map[string]any {
	l, _ := list.([]any)
	for _, e := range l {
		if m, ok := e.(map[string]any); ok && m[key] == val {
			return m
		}
	}
	return nil
}

// roundTripClients compares the client settings the operator audience and the
// token contents depend on.
func roundTripClients(t *testing.T, want, got map[string]any) {
	t.Helper()
	if g, w := fmt.Sprint(got["defaultDefaultClientScopes"]), fmt.Sprint(want["defaultDefaultClientScopes"]); g != w {
		t.Errorf("defaultDefaultClientScopes: exported %s, imported %s", g, w)
	}
	for _, wc := range want["clients"].([]any) {
		wcm := wc.(map[string]any)
		id := wcm["clientId"]
		gc := find(got["clients"], "clientId", id)
		if gc == nil {
			t.Errorf("client %v not exported", id)
			continue
		}
		for _, k := range []string{"fullScopeAllowed", "bearerOnly", "serviceAccountsEnabled", "publicClient", "directAccessGrantsEnabled"} {
			// An absent key in the file means the Keycloak default, which is false here.
			w := wcm[k]
			if w == nil {
				w = false
			}
			if fmt.Sprint(gc[k]) != fmt.Sprint(w) {
				t.Errorf("client %v %s: exported %v, imported %v", id, k, gc[k], w)
			}
		}
		for _, k := range []string{"defaultClientScopes", "authenticationFlowBindingOverrides"} {
			if wv, ok := wcm[k]; ok && canon(wv) != canon(gc[k]) {
				t.Errorf("client %v %s: exported %v, imported %v", id, k, gc[k], wv)
			}
		}
	}
	// Realm role scope mappings of the clients (the operator role only for the CLI).
	wsm, _ := want["scopeMappings"].([]any)
	gsm, _ := got["scopeMappings"].([]any)
	nClient := 0
	for _, g := range gsm {
		if _, ok := g.(map[string]any)["client"]; ok { // Keycloak adds its own client-scope entry (offline_access)
			nClient++
		}
	}
	if nClient != len(wsm) {
		t.Errorf("scopeMappings: exported %d client entries, imported %d", nClient, len(wsm))
	}
	for _, w := range wsm {
		wm := w.(map[string]any)
		gm := find(got["scopeMappings"], "client", wm["client"])
		if gm == nil {
			t.Errorf("scopeMappings for client %v not exported", wm["client"])
			continue
		}
		if g, w := sortedStrings(gm["roles"]), sortedStrings(wm["roles"]); g != w {
			t.Errorf("scopeMappings %v roles: exported %s, imported %s", wm["client"], g, w)
		}
	}
}

// canon renders a value so that list order does not matter.
func canon(v any) string {
	if _, ok := v.([]any); ok {
		return sortedStrings(v)
	}
	return fmt.Sprint(v)
}

func sortedStrings(v any) string {
	var out []string
	l, _ := v.([]any)
	for _, e := range l {
		out = append(out, fmt.Sprint(e))
	}
	sort.Strings(out)
	return strings.Join(out, ",")
}

// roundTripOrganizations compares the organizations of the running realm with
// the file. partial-export leaves organizations out, so this reads them through
// the admin API: id, verified domains and the identity provider link.
func roundTripOrganizations(t *testing.T, want map[string]any) {
	t.Helper()
	adm := masterToken(t) // the platform service account may not read identity providers
	for _, wo := range want["organizations"].([]any) {
		wom := wo.(map[string]any)
		alias := fmt.Sprint(wom["alias"])
		id := orgID(t, adm, alias)
		if wid, _ := wom["id"].(string); wid != "" && id != wid {
			t.Errorf("organization %s: running id %s, imported id %s", alias, id, wid)
		}
		// The admin API serves the domains on the organization itself.
		code, b := admin(t, adm, http.MethodGet, "/organizations/"+id, nil)
		var org struct {
			Domains []struct {
				Name     string
				Verified bool
			}
		}
		if code != http.StatusOK || json.Unmarshal(b, &org) != nil {
			t.Errorf("organization %s: %d %s", alias, code, b)
			continue
		}
		domains := org.Domains
		var gd, wd []string
		for _, d := range domains {
			if !d.Verified {
				t.Errorf("organization %s: domain %s is not verified", alias, d.Name)
			}
			gd = append(gd, d.Name)
		}
		for _, d := range wom["domains"].([]any) {
			wd = append(wd, fmt.Sprint(d.(map[string]any)["name"]))
		}
		sort.Strings(gd)
		sort.Strings(wd)
		if strings.Join(gd, ",") != strings.Join(wd, ",") {
			t.Errorf("organization %s domains: running %v, imported %v", alias, gd, wd)
		}
		code, b = admin(t, adm, http.MethodGet, "/organizations/"+id+"/identity-providers", nil)
		var idps []struct {
			Alias             string
			OrganizationLinks []struct{ OrganizationID string }
		}
		if code != http.StatusOK || json.Unmarshal(b, &idps) != nil {
			t.Errorf("organization %s identity providers: %d %s", alias, code, b)
			continue
		}
		var gi, wi []string
		for _, p := range idps {
			gi = append(gi, p.Alias)
			if len(p.OrganizationLinks) != 1 || p.OrganizationLinks[0].OrganizationID != id {
				t.Errorf("idp %s is linked to %v, want only organization %s", p.Alias, p.OrganizationLinks, alias)
			}
		}
		for _, p := range wom["identityProviders"].([]any) {
			wi = append(wi, fmt.Sprint(p.(map[string]any)["alias"]))
		}
		sort.Strings(gi)
		sort.Strings(wi)
		if strings.Join(gi, ",") != strings.Join(wi, ",") {
			t.Errorf("organization %s identity providers: running %v, imported %v", alias, gi, wi)
		}
	}
}

// TestAdapterAcceptsTenantToken runs alice's nava-web token through the T15
// Keycloak adapter against the real realm: audience, issuer, signature and
// the organization claim all pass, so Resolve fails only at tenant resolution
// (the adapter has no store here, so it refuses). The same token is not an operator token.
func TestAdapterAcceptsTenantToken(t *testing.T) {
	tok := passwordToken(t, "nava-web", "alice@tenant-a.test", "dev-only-alice")
	a := stackAdapter(t, tok)

	_, err := a.Resolve(t.Context(), tok)
	if !errors.Is(err, identity.ErrTenantUnresolved) {
		t.Errorf("Resolve = %v, want ErrTenantUnresolved (token valid, no binding yet)", err)
	}
	if _, err := a.ResolveOperator(t.Context(), tok); !errors.Is(err, identity.ErrInvalidToken) {
		t.Errorf("ResolveOperator(tenant token) = %v, want ErrInvalidToken", err)
	}
	// The operator token still resolves as an operator.
	op := passwordToken(t, "nava-operator-cli", "operator@nava.test", "dev-only-operator")
	if _, err := a.ResolveOperator(t.Context(), op); err != nil {
		t.Errorf("ResolveOperator(operator token) = %v", err)
	}
}

// audHas reports whether the access token's aud holds want as one whole
// element (auds), not as a substring: nava-api is inside nava-operator-api.
func audHas(t *testing.T, tok, want string) bool {
	t.Helper()
	return slices.Contains(auds(t, tok), want)
}

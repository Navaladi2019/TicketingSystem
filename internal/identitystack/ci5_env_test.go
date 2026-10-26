//go:build identitystack

package identitystack

// The environment of the CI-5 suite (T19): the compose Keycloak, and a
// throwaway platform database next to it (NAVA_TEST_PG_URL, a superuser, set
// by scripts/ci/identity-attacks.sh). The database is built as a first
// `docker compose up` builds it (db-roles, then every migration), tenants A
// and B are bound to the realm's organizations, and one real group sync
// (groupsync.Syncer.Once, what `nava-groupsync --once` runs) fills the
// principals the resolver reads. The resolver is the production adapter
// reading through pgstore as nava_app.

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"html"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/metric/noop"

	"github.com/Navaladi2019/TicketingSystem/db"
	"github.com/Navaladi2019/TicketingSystem/internal/db/catalog"
	"github.com/Navaladi2019/TicketingSystem/internal/db/grants"
	"github.com/Navaladi2019/TicketingSystem/internal/db/migrate"
	"github.com/Navaladi2019/TicketingSystem/internal/db/roles"
	"github.com/Navaladi2019/TicketingSystem/internal/groupsync"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/identity/keycloak"
	"github.com/Navaladi2019/TicketingSystem/internal/identity/pgstore"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
)

// ci5Env is what every CI-5 case runs against.
type ci5Env struct {
	iss        string // realm nava's issuer, as tokens carry it
	orgA, orgB string // the organization ids of tenant-a and tenant-b
	super      *pgx.Conn
	app        *pgxpool.Pool // nava_app, what nava-core resolves with
	resolver   *keycloak.Adapter
	syncer     *groupsync.Syncer
	reg        *memRegistry
	dir        *keycloak.Directory
}

func ci5Password(role string) string { return "test-pw-" + role }

// newCI5Env builds the database and binds tenant A to organization tenant-a
// with its IdP, and tenant B to organization tenant-b with no IdP: tenant B's
// brokered users are the "idp not bound" case.
func newCI5Env(t *testing.T) *ci5Env {
	t.Helper()
	ctx := t.Context()
	pgURL := os.Getenv("NAVA_TEST_PG_URL")
	if pgURL == "" {
		t.Fatal("NAVA_TEST_PG_URL is not set: run make identity-attacks, which starts a throwaway PostgreSQL beside the stack")
	}
	superCfg, err := pgx.ParseConfig(pgURL)
	if err != nil {
		t.Fatal(err)
	}
	m, err := grants.Parse(db.Grants())
	if err != nil {
		t.Fatal(err)
	}
	gt, err := catalog.ParseGlobalTables(db.GlobalTables())
	if err != nil {
		t.Fatal(err)
	}
	conn, err := pgx.ConnectConfig(ctx, superCfg)
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range m.DatabaseNames() {
		mustExec(t, conn, "DROP DATABASE IF EXISTS "+pgx.Identifier{d}.Sanitize()+" WITH (FORCE)")
	}
	for _, r := range m.RoleNames() {
		mustExec(t, conn, "DROP ROLE IF EXISTS "+pgx.Identifier{r}.Sanitize())
	}
	_ = conn.Close(ctx)
	log := slog.New(slog.DiscardHandler)
	pw := func(role string) (string, bool) { return ci5Password(role), true }
	if _, err := roles.Run(ctx, superCfg, m, pw, log); err != nil {
		t.Fatalf("db-roles: %v", err)
	}
	owner := superCfg.Copy()
	owner.User, owner.Password, owner.Database = m.Owner(), ci5Password(m.Owner()), m.PlatformDatabase
	if _, err := migrate.Run(ctx, owner, db.Migrations(), m, gt, log, migrate.Options{LockRetrySeconds: 1}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	sup := superCfg.Copy()
	sup.Database = m.PlatformDatabase
	super, err := pgx.ConnectConfig(ctx, sup)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = super.Close(context.Background()) })
	pool := func(role string) *pgxpool.Pool {
		pc, err := pgxpool.ParseConfig(pgURL)
		if err != nil {
			t.Fatal(err)
		}
		pc.ConnConfig.User, pc.ConnConfig.Password, pc.ConnConfig.Database = role, ci5Password(role), m.PlatformDatabase
		pc.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeExec
		pc.MaxConns = 4
		p, err := pgxpool.NewWithConfig(ctx, pc)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(p.Close)
		return p
	}

	e := &ci5Env{
		iss:   claims(t, passwordToken(t, "nava-web", "alice@tenant-a.test", "dev-only-alice"))["iss"].(string),
		orgA:  orgID(t, masterToken(t), "tenant-a"),
		orgB:  orgID(t, masterToken(t), "tenant-b"),
		super: super,
		app:   pool("nava_app"),
		reg:   newRegistry(),
	}
	for _, b := range []struct {
		tenant uuid.UUID
		org    string
		idps   []string
	}{{tenantA, e.orgA, []string{"idp-tenant-a"}}, {tenantB, e.orgB, []string{}}} {
		mustExec(t, super, "INSERT INTO tenants (id, state) VALUES ($1, 'active')", b.tenant)
		mustExec(t, super, "INSERT INTO tenant_identity_binding (tenant_id, issuer, org_id, idp_aliases) VALUES ($1, $2, $3, $4)",
			b.tenant, e.iss, b.org, b.idps)
	}
	e.dir = newDirectory(t, e.reg, false, nil)
	e.syncer, err = groupsync.New(pool("nava_groupsync"), e.dir, noop.NewMeterProvider(), log, groupsync.Options{})
	if err != nil {
		t.Fatal(err)
	}
	e.resolver = ci5Adapter(t, []string{e.iss}, pgstore.New(e.app))
	return e
}

func mustExec(t *testing.T, c *pgx.Conn, sql string, args ...any) {
	t.Helper()
	if _, err := c.Exec(context.Background(), sql, args...); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
}

// sync runs one group-sync cycle and fails the test if a tenant failed.
func (e *ci5Env) sync(t *testing.T) {
	t.Helper()
	rs, err := e.syncer.Once(t.Context())
	if err != nil {
		t.Fatalf("group sync: %v", err)
	}
	for _, r := range rs {
		if r.Err != nil || r.Skipped {
			t.Fatalf("group sync of %s: skipped=%v err=%v", r.TenantID, r.Skipped, r.Err)
		}
	}
}

// principalID reads the principal group sync recorded for a Keycloak user.
func (e *ci5Env) principalID(t *testing.T, tenant uuid.UUID, kcUser string) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	if err := e.super.QueryRow(t.Context(), "SELECT id FROM principals WHERE tenant_id = $1 AND keycloak_user_id = $2",
		tenant, kcUser).Scan(&id); err != nil {
		t.Fatalf("principal of %s in %s: %v", kcUser, tenant, err)
	}
	return id
}

// ci5Adapter is the production adapter trusting issuers, reading store.
func ci5Adapter(t *testing.T, issuers []string, store identity.Store) *keycloak.Adapter {
	t.Helper()
	cfg, err := config.Load(func(string) (string, bool) { return "", false })
	if err != nil {
		t.Fatal(err)
	}
	tel, err := telemetry.New(t.Context(), cfg.Telemetry)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tel.Shutdown(context.Background()) })
	out, err := telemetry.NewOutboundClient(tel, telemetry.OutboundOptions{Timeout: 10 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	a, err := keycloak.New(t.Context(), keycloak.Config{Issuers: issuers, MeterProvider: tel.MeterProvider, Store: store}, out)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = a.Close(context.Background()) })
	return a
}

// ---- users the cases create -----------------------------------------------

// ci5User creates a password user in realm nava, a member of the given
// organizations (by id) and groups (by path), and deletes it after the test.
func ci5User(t *testing.T, prefix string, orgs, groups []string) (id, username, pass string) {
	t.Helper()
	username = prefix + "-" + uuid.NewString()[:8]
	pass = "dev-only-" + prefix
	code, b := adminRealm(t, "nava", http.MethodPost, "/users", map[string]any{
		"username": username, "email": username + "@ci5.invalid", "emailVerified": true, "enabled": true,
		"firstName": "Ci", "lastName": "Five",
		"credentials": []map[string]any{{"type": "password", "value": pass, "temporary": false}},
	})
	if code != http.StatusCreated {
		t.Fatalf("create %s: %d %s", username, code, b)
	}
	id = userID(t, serviceToken(t), username)
	t.Cleanup(func() { adminRealm(t, "nava", http.MethodDelete, "/users/"+id, nil) })
	for _, o := range orgs {
		addOrgMember(t, o, id)
	}
	for _, g := range groups {
		if code, b := adminRealm(t, "nava", http.MethodPut, "/users/"+id+"/groups/"+kcGroupID(t, g), nil); code != http.StatusNoContent {
			t.Fatalf("add %s to %s: %d %s", username, g, code, b)
		}
	}
	return id, username, pass
}

// addOrgMember adds a user to an organization as the master admin: it is the
// misconfiguration or insider step an attack starts from, not something a
// tenant can do (CI5OtherTenantAdmin shows that).
func addOrgMember(t *testing.T, org, user string) {
	t.Helper()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost,
		baseURL()+"/admin/realms/nava/organizations/"+org+"/members", strings.NewReader(`"`+user+`"`))
	req.Header.Set("Authorization", "Bearer "+masterToken(t))
	req.Header.Set("Content-Type", "application/json")
	if code, b := do(t, newClient(t), req); code != http.StatusCreated {
		t.Fatalf("add %s to organization %s: %d %s", user, org, code, b)
	}
}

// ci5Org creates an organization nobody is bound to, deleted after the test.
func ci5Org(t *testing.T) string {
	t.Helper()
	alias := "ci5-" + uuid.NewString()[:8]
	code, b := adminRealm(t, "nava", http.MethodPost, "/organizations", map[string]any{
		"name": alias, "alias": alias, "enabled": true, "domains": []map[string]any{{"name": alias + ".invalid", "verified": false}},
	})
	if code != http.StatusCreated {
		t.Fatalf("create organization %s: %d %s", alias, code, b)
	}
	id := orgID(t, masterToken(t), alias)
	t.Cleanup(func() { adminRealm(t, "nava", http.MethodDelete, "/organizations/"+id, nil) })
	return id
}

// orgIDs are the organization ids a token names.
func orgIDs(t *testing.T, tok string) []string {
	t.Helper()
	org, _ := claims(t, tok)["organization"].(map[string]any)
	var ids []string
	for _, v := range org {
		if m, ok := v.(map[string]any); ok {
			if id, ok := m["id"].(string); ok {
				ids = append(ids, id)
			}
		}
	}
	return ids
}

// ---- a realm whose private key the test holds ------------------------------

// foreignRealm creates realm ci5-x-<n> whose active RS256 key the test
// generated, so the test can sign what Keycloak would never sign: a token
// with that key and another realm's iss. It returns the realm's issuer, the
// key and its kid.
func foreignRealm(t *testing.T, base string, c *http.Client, masterTok string) (iss string, key *rsa.PrivateKey, kid string) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	realm := "ci5-x-" + uuid.NewString()[:8]
	call := func(method, path string, body any) (int, []byte) {
		var rdr io.Reader
		if body != nil {
			b, _ := json.Marshal(body)
			rdr = bytes.NewReader(b)
		}
		req, _ := http.NewRequestWithContext(context.Background(), method, base+"/admin/realms"+path, rdr)
		req.Header.Set("Authorization", "Bearer "+masterTok)
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		return do(t, c, req)
	}
	if code, b := call(http.MethodPost, "", map[string]any{"realm": realm, "enabled": true}); code != http.StatusCreated {
		t.Fatalf("create realm %s: %d %s", realm, code, b)
	}
	t.Cleanup(func() { call(http.MethodDelete, "/"+realm, nil) })
	_, b := call(http.MethodGet, "/"+realm, nil)
	var rep struct{ ID string }
	if json.Unmarshal(b, &rep) != nil || rep.ID == "" {
		t.Fatalf("realm %s: %s", realm, b)
	}
	priv := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	if code, b := call(http.MethodPost, "/"+realm+"/components", map[string]any{
		"name": "ci5-test-held-key", "providerId": "rsa", "providerType": "org.keycloak.keys.KeyProvider", "parentId": rep.ID,
		"config": map[string][]string{
			"privateKey": {string(priv)}, "priority": {"1000"}, "active": {"true"}, "enabled": {"true"}, "algorithm": {"RS256"},
		},
	}); code != http.StatusCreated {
		t.Fatalf("import key into %s: %d %s", realm, code, b)
	}
	iss = base + "/realms/" + realm
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, iss+"/protocol/openid-connect/certs", nil)
	_, jb := do(t, c, req)
	var jwks struct {
		Keys []struct{ Kid, N, Use string }
	}
	if err := json.Unmarshal(jb, &jwks); err != nil {
		t.Fatalf("jwks of %s: %v %s", realm, err, jb)
	}
	n := base64.RawURLEncoding.EncodeToString(key.N.Bytes())
	for _, k := range jwks.Keys {
		if k.N == n && k.Use != "enc" {
			return iss, key, k.Kid
		}
	}
	t.Fatalf("the imported key is not in %s's JWKS: %s", realm, jb)
	return "", nil, ""
}

// signRS256 signs claims with key as kid.
func signRS256(t *testing.T, key *rsa.PrivateKey, kid string, c map[string]any) string {
	t.Helper()
	h, _ := json.Marshal(map[string]string{"alg": "RS256", "typ": "JWT", "kid": kid})
	p, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	in := base64.RawURLEncoding.EncodeToString(h) + "." + base64.RawURLEncoding.EncodeToString(p)
	sum := sha256.Sum256([]byte(in))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, sum[:])
	if err != nil {
		t.Fatal(err)
	}
	return in + "." + base64.RawURLEncoding.EncodeToString(sig)
}

// freshClaims copies a token's claims with a new jti and a fresh exp, so a
// re-signed copy differs only where the case changes it.
func freshClaims(t *testing.T, tok string) map[string]any {
	t.Helper()
	c := claims(t, tok)
	c["jti"] = uuid.NewString()
	now := time.Now().Unix()
	c["iat"], c["exp"] = now, now+120
	return c
}

// ---- the organization picker -----------------------------------------------

var (
	pickerForm = regexp.MustCompile(`(?s)<form[^>]*action="([^"]+)"`)
	// The picker's choices: a script assignment up to 26.1, a button since.
	pickerOrg = regexp.MustCompile(`\['kc\.org'\]\.value\s*=\s*'([^']+)'|name="kc\.org"\s+value="([^"]+)"`)
)

// newJarClient is a cookie client for a Keycloak at any address.
func newJarClient(t *testing.T) *http.Client {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	return &http.Client{Jar: jar, Timeout: 30 * time.Second}
}

// pickerResult is what the browser flow ended with.
type pickerResult struct {
	Shown   bool     // Keycloak showed the organization picker
	Choices []string // the organizations it offered (ids)
	Token   string   // the access token, if the flow ended with one
	Page    string   // the last page, for the failure message
}

// orgPicker signs user in to realm through the browser flow of client with
// scope "openid organization". When Keycloak shows the organization picker,
// it posts pick as kc.org, whatever the picker offered: the TA-B1 tamper.
func orgPicker(t *testing.T, c *http.Client, base, realm, client, user, pass, pick string) pickerResult {
	t.Helper()
	const redirect = "http://localhost:9/cb"
	var final *url.URL
	c.CheckRedirect = func(req *http.Request, _ []*http.Request) error {
		if strings.HasPrefix(req.URL.String(), redirect) {
			final = req.URL
			return http.ErrUseLastResponse
		}
		return nil
	}
	q := url.Values{"client_id": {client}, "response_type": {"code"}, "scope": {"openid organization"}, "redirect_uri": {redirect}}
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, base+"/realms/"+realm+"/protocol/openid-connect/auth?"+q.Encode(), nil)
	_, page := do(t, c, req)
	var res pickerResult
	for range 6 {
		if final != nil {
			break
		}
		res.Page = string(page)
		m := pickerForm.FindSubmatch(page)
		if m == nil {
			return res
		}
		form := url.Values{}
		switch body := string(page); {
		case strings.Contains(body, `name="password"`) && strings.Contains(body, `name="username"`):
			form.Set("username", user)
			form.Set("password", pass)
		case strings.Contains(body, `name="username"`):
			form.Set("username", user)
		case strings.Contains(body, `name="password"`):
			form.Set("password", pass)
		case strings.Contains(body, "kc.org"):
			res.Shown = true
			for _, o := range pickerOrg.FindAllSubmatch(page, -1) {
				res.Choices = append(res.Choices, string(o[1])+string(o[2]))
			}
			form.Set("kc.org", pick)
		default:
			return res
		}
		post, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, html.UnescapeString(string(m[1])), strings.NewReader(form.Encode()))
		post.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		_, page = do(t, c, post)
	}
	if final == nil || final.Query().Get("code") == "" {
		return res
	}
	tr, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, base+"/realms/"+realm+"/protocol/openid-connect/token",
		strings.NewReader(url.Values{
			"grant_type": {"authorization_code"}, "client_id": {client}, "code": {final.Query().Get("code")}, "redirect_uri": {redirect},
		}.Encode()))
	tr.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	code, body := do(t, c, tr)
	var out struct {
		AccessToken string `json:"access_token"`
	}
	if code == http.StatusOK && json.Unmarshal(body, &out) == nil {
		res.Token = out.AccessToken
	}
	return res
}

// describeOrgs names the organizations a token held, for a failure message.
func describeOrgs(ids []string) string {
	if len(ids) == 0 {
		return "none"
	}
	return strings.Join(ids, ", ")
}

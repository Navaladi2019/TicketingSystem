//go:build integration

// TICK-511: GET /api/v1/me (getMe) and POST /api/v1/operator/tenants
// (onboardTenant) against a real PostgreSQL, through the real handler, the real
// Keycloak adapter (a test realm signs the tokens), the real pgstore and the
// real onboard.PGStore as nava_onboard. Only Keycloak's admin side (the
// Directory) is faked, because it is a third-party service behind a port.
//
// Each test names the documented outcome it covers with an
// "covers: <operationId> <status>..." marker, which the coverage gate
// (api/integration_coverage_test.go) reads.

package server_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/api"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/identity/keycloak/keycloaktest"
	"github.com/Navaladi2019/TicketingSystem/internal/identity/pgstore"
	"github.com/Navaladi2019/TicketingSystem/internal/onboard"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
	"github.com/Navaladi2019/TicketingSystem/internal/server"
)

const operatorSub = "op-sub-1"

// derivedTenantID restates the documented derivation, UUIDv5(namespace,
// operator + NUL + Idempotency-Key), so the tests do not call the code under test.
func derivedTenantID(key string) uuid.UUID {
	return uuid.NewSHA1(onboard.Namespace, []byte(operatorSub+"\x00"+key))
}

// fakeDir is the Keycloak side of onboarding. createErr, when set, fails the
// organization create.
type fakeDir struct {
	mu        sync.Mutex
	orgs      map[uuid.UUID]string
	creates   int
	invites   []string
	createErr error
}

func (d *fakeDir) CreateTenantOrg(_ context.Context, id uuid.UUID, _ string, _ []string) (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.createErr != nil {
		return "", d.createErr
	}
	if _, ok := d.orgs[id]; ok {
		return "", identity.ErrAlreadyExists
	}
	d.creates++
	d.orgs[id] = "org-" + id.String()
	return d.orgs[id], nil
}

func (d *fakeDir) InviteUser(_ context.Context, _ uuid.UUID, email string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.invites = append(d.invites, email)
	return nil
}

func (d *fakeDir) EnsureAdminGroup(_ context.Context, id uuid.UUID) (string, error) {
	return "kc-admins-" + id.String(), nil
}

func (d *fakeDir) EnsureAuditorGroup(_ context.Context, id uuid.UUID) (string, error) {
	return "kc-auditors-" + id.String(), nil
}

func (d *fakeDir) FindTenantOrg(_ context.Context, id uuid.UUID, _ string) (string, bool, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	o, ok := d.orgs[id]
	return o, ok, nil
}

func (d *fakeDir) inviteList() []string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]string(nil), d.invites...)
}

func (d *fakeDir) createCount() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.creates
}

// pubResolver answers every host with one public address.
type pubResolver struct{}

func (pubResolver) LookupNetIP(context.Context, string, string) ([]netip.Addr, error) {
	return []netip.Addr{netip.MustParseAddr("93.184.216.34")}, nil
}

// idEnv is the real handler over a migrated database with tenants A and B,
// each bound to a Keycloak organization of the test realm.
type idEnv struct {
	h     http.Handler
	e     *etEnv
	realm *keycloaktest.Realm
	dir   *fakeDir
	lim   *spyLimiter
}

func roleLoginPool(t *testing.T, role string, maxConns int32) *pgxpool.Pool {
	t.Helper()
	pc, err := pgxpool.ParseConfig(os.Getenv("NAVA_TEST_PG_URL"))
	if err != nil {
		t.Fatal(err)
	}
	pc.ConnConfig.User, pc.ConnConfig.Password, pc.ConnConfig.Database = role, etPassword(role), "nava"
	pc.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeExec
	pc.MaxConns = maxConns
	pool, err := pgxpool.NewWithConfig(t.Context(), pc)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func newIDEnv(t *testing.T, appPoolOverride *pgxpool.Pool) *idEnv {
	t.Helper()
	super, pool := etCluster(t)
	a, b := etSeedTenant(t, super, "a"), etSeedTenant(t, super, "b")
	e := &etEnv{pool: pool, a: a, b: b, super: super}
	realm := keycloaktest.New(t)
	for _, bind := range []struct {
		tn         etTenant
		org, alias string
	}{{a, "org-a", "idp-a"}, {b, "org-b", "idp-b"}} {
		if _, err := super.Exec(t.Context(),
			"INSERT INTO tenant_identity_binding (tenant_id, issuer, org_id, idp_aliases) VALUES ($1, $2, $3, $4)",
			bind.tn.id, realm.Issuer(), bind.org, []string{bind.alias}); err != nil {
			t.Fatal(err)
		}
	}
	appPool := pool
	if appPoolOverride != nil {
		appPool = appPoolOverride
	}
	adapter := realm.Resolver(t, pgstore.New(appPool), operatorSub)

	dir := &fakeDir{orgs: map[uuid.UUID]string{}}
	onboardPool := roleLoginPool(t, "nava_onboard", 4)
	store, err := onboard.NewPGStore(onboardPool, onboard.NewRegistry(pool))
	if err != nil {
		t.Fatal(err)
	}
	svc, err := onboard.New(onboard.Options{
		Store: store, Directory: dir, Finder: dir, Domains: onboard.NewRegistry(pool), ACL: onboard.ACLStore{Pool: pool},
		Specs: onboard.NewSpecs(), Resolver: pubResolver{}, Issuer: realm.Issuer(), Log: slog.New(slog.DiscardHandler),
	})
	if err != nil {
		t.Fatal(err)
	}
	tel, err := telemetry.New(t.Context(), config.Telemetry{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tel.Shutdown(context.Background()) })
	lim := &spyLimiter{}
	lim.reset(true)
	h, err := server.NewHandler(slog.New(slog.NewTextHandler(testLogWriter{t, nil}, &slog.HandlerOptions{Level: slog.LevelError})),
		tel, adapter, pgstore.New(appPool), server.WithLimiter(lim), server.WithPool(pool), server.WithOperator(adapter, svc))
	if err != nil {
		t.Fatal(err)
	}
	return &idEnv{h: h, e: e, realm: realm, dir: dir, lim: lim}
}

func (v *idEnv) call(t *testing.T, method, path, token, key, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), method, path, strings.NewReader(body))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	v.h.ServeHTTP(rec, req)
	return rec
}

func (v *idEnv) tenantToken(t *testing.T, sub, alias, org string) string {
	t.Helper()
	return v.realm.Mint(t, keycloaktest.Token{Subject: sub, Orgs: map[string]string{alias: org}})
}

func (v *idEnv) operatorToken(t *testing.T) string {
	t.Helper()
	return v.realm.Mint(t, keycloaktest.Token{
		Subject: operatorSub, Audiences: []string{"nava-operator-api"}, RealmRoles: []string{"nava-operator"},
	})
}

func (v *idEnv) count(t *testing.T, sql string, args ...any) int {
	t.Helper()
	var n int
	if err := v.e.super.QueryRow(t.Context(), sql, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// footprint counts, over all tenants, every table onboarding writes.
func (v *idEnv) footprint(t *testing.T) map[string]int {
	t.Helper()
	out := map[string]int{}
	for _, table := range []string{
		"tenants", "tenant_identity_binding", "tenant_email_domain", "groups",
		"tenant_admin_group", "entity_definitions", "entity_definition_versions", "acl_versions", "principals", "group_members",
	} {
		out[table] = v.count(t, "SELECT count(*) FROM "+table)
	}
	return out
}

func sameFootprint(t *testing.T, before, after map[string]int) {
	t.Helper()
	for k, n := range before {
		if after[k] != n {
			t.Errorf("table %s has %d rows, want %d (unchanged)", k, after[k], n)
		}
	}
}

func wantStatus(t *testing.T, rec *httptest.ResponseRecorder, status int, code problem.Code) {
	t.Helper()
	if rec.Code != status || problemCode(rec) != code {
		t.Fatalf("= %d %s, want %d %s", rec.Code, rec.Body, status, code)
	}
}

// ---- getMe ----

// covers: getMe 200
func TestGetMeMatchesDatabaseRows(t *testing.T) {
	v := newIDEnv(t, nil)
	rec := v.call(t, http.MethodGet, "/api/v1/me", v.tenantToken(t, "a-admin", "tenant-a", "org-a"), "", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /me = %d: %s", rec.Code, rec.Body)
	}
	var me api.Me
	if err := json.Unmarshal(rec.Body.Bytes(), &me); err != nil {
		t.Fatal(err)
	}
	// Expected values come from the rows, not from the seed variables.
	var principalID, tenantID uuid.UUID
	if err := v.e.super.QueryRow(t.Context(),
		"SELECT id, tenant_id FROM principals WHERE keycloak_user_id = 'a-admin'").Scan(&principalID, &tenantID); err != nil {
		t.Fatal(err)
	}
	rows, err := v.e.super.Query(t.Context(),
		`SELECT g.id, g.path FROM group_members gm JOIN groups g ON g.tenant_id = gm.tenant_id AND g.id = gm.group_id
		 WHERE gm.principal_id = $1 ORDER BY g.id`, principalID)
	if err != nil {
		t.Fatal(err)
	}
	var wantGroups []uuid.UUID
	var wantPaths []api.GroupPath
	for rows.Next() {
		var id uuid.UUID
		var path string
		if err := rows.Scan(&id, &path); err != nil {
			t.Fatal(err)
		}
		wantGroups = append(wantGroups, id)
		wantPaths = append(wantPaths, api.GroupPath{Id: id, Path: strings.TrimPrefix(path, "/t/"+tenantID.String())})
	}
	rows.Close()
	if len(wantGroups) != 1 {
		t.Fatalf("seed: admin is in %d groups, want 1", len(wantGroups))
	}
	if me.UserId != principalID || me.TenantId != tenantID || fmt.Sprint(me.Groups) != fmt.Sprint(wantGroups) ||
		fmt.Sprint(me.GroupPaths) != fmt.Sprint(wantPaths) || me.GroupPaths[0].Path != "/admins" {
		t.Errorf("me = %+v, want principal %s tenant %s groups %v paths %v", me, principalID, tenantID, wantGroups, wantPaths)
	}
	// A user in no group gets empty lists, as the rows say.
	rec = v.call(t, http.MethodGet, "/api/v1/me", v.tenantToken(t, "a-user", "tenant-a", "org-a"), "", "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"group_paths":[]`) || !strings.Contains(rec.Body.String(), `"groups":[]`) {
		t.Errorf("group-less user = %d %s, want 200 with empty groups and group_paths", rec.Code, rec.Body)
	}
}

// A tenant named in the header, the query or the body never wins over the
// token's organization.
func TestGetMeIgnoresSuppliedTenant(t *testing.T) {
	v := newIDEnv(t, nil)
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/me?tenant_id="+v.e.b.id.String(), nil)
	req.Header.Set("Authorization", "Bearer "+v.tenantToken(t, "a-admin", "tenant-a", "org-a"))
	req.Header.Set("X-Tenant-Id", v.e.b.id.String())
	rec := httptest.NewRecorder()
	v.h.ServeHTTP(rec, req)
	var me api.Me
	if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &me) != nil {
		t.Fatalf("GET /me = %d %s, want 200", rec.Code, rec.Body)
	}
	if me.TenantId != v.e.a.id {
		t.Errorf("tenant = %s, want A %s", me.TenantId, v.e.a.id)
	}
	if strings.Contains(rec.Body.String(), v.e.b.id.String()) {
		t.Errorf("tenant B leaked into %s", rec.Body)
	}
}

// covers: getMe 401
func TestGetMeCrossTenantAndUnknownPrincipals(t *testing.T) {
	v := newIDEnv(t, nil)
	before := v.footprint(t)
	bUser := v.tenantToken(t, "b-admin", "tenant-a", "org-a") // B's subject under A's organization
	aUser := v.tenantToken(t, "a-admin", "tenant-b", "org-b") // A's subject under B's organization
	for name, tok := range map[string]string{
		"tenant B principal under tenant A": bUser,
		"tenant A principal under tenant B": aUser,
		"unknown subject":                   v.tenantToken(t, "nobody", "tenant-a", "org-a"),
		"no token":                          "",
		"forged token":                      "a.b.c",
		"unbound organization":              v.tenantToken(t, "a-admin", "tenant-x", "org-x"),
		"operator token":                    v.operatorToken(t),
	} {
		rec := v.call(t, http.MethodGet, "/api/v1/me", tok, "", "")
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s: = %d %s, want 401", name, rec.Code, rec.Body)
			continue
		}
		if c := problemCode(rec); c != problem.TenantUnresolved && c != problem.InvalidToken {
			t.Errorf("%s: code %q", name, c)
		}
	}
	sameFootprint(t, before, v.footprint(t))
}

// The same Keycloak subject exists in both tenants: the organization picks the
// tenant, and the answer is that tenant's row.
func TestGetMeSameSubjectInTwoTenants(t *testing.T) {
	v := newIDEnv(t, nil)
	ids := map[uuid.UUID]uuid.UUID{}
	for _, tn := range []etTenant{v.e.a, v.e.b} {
		id := uuid.New()
		ids[tn.id] = id
		if _, err := v.e.super.Exec(t.Context(),
			"INSERT INTO principals (tenant_id, id, keycloak_user_id, email, enabled, federated_links, synced_at) VALUES ($1, $2, 'shared', 'shared@x.test', true, '[]', now())",
			tn.id, id); err != nil {
			t.Fatal(err)
		}
	}
	for _, c := range []struct {
		tn         etTenant
		alias, org string
	}{{v.e.a, "tenant-a", "org-a"}, {v.e.b, "tenant-b", "org-b"}} {
		rec := v.call(t, http.MethodGet, "/api/v1/me", v.tenantToken(t, "shared", c.alias, c.org), "", "")
		var me api.Me
		if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &me) != nil {
			t.Fatalf("GET /me = %d %s", rec.Code, rec.Body)
		}
		if me.TenantId != c.tn.id || me.UserId != ids[c.tn.id] {
			t.Errorf("me = %+v, want tenant %s principal %s", me, c.tn.id, ids[c.tn.id])
		}
	}
}

// covers: getMe 401
func TestGetMeTombstonedAndDisabledPrincipals(t *testing.T) {
	v := newIDEnv(t, nil)
	tok := v.tenantToken(t, "a-user", "tenant-a", "org-a")
	if rec := v.call(t, http.MethodGet, "/api/v1/me", tok, "", ""); rec.Code != http.StatusOK {
		t.Fatalf("before = %d: %s", rec.Code, rec.Body)
	}
	if _, err := v.e.super.Exec(t.Context(), "UPDATE principals SET enabled = false, deleted_at = now() WHERE tenant_id = $1 AND keycloak_user_id = 'a-user'", v.e.a.id); err != nil {
		t.Fatal(err)
	}
	if n := v.count(t, "SELECT count(*) FROM principals WHERE tenant_id = $1 AND keycloak_user_id = 'a-user' AND deleted_at IS NOT NULL", v.e.a.id); n != 1 {
		t.Fatalf("tombstone rows = %d, want 1", n)
	}
	wantStatus(t, v.call(t, http.MethodGet, "/api/v1/me", tok, "", ""), http.StatusUnauthorized, problem.TenantUnresolved)

	// A disabled (not tombstoned) principal is refused the same way.
	if _, err := v.e.super.Exec(t.Context(), "UPDATE principals SET enabled = false WHERE tenant_id = $1 AND keycloak_user_id = 'a-admin'", v.e.a.id); err != nil {
		t.Fatal(err)
	}
	wantStatus(t, v.call(t, http.MethodGet, "/api/v1/me", v.tenantToken(t, "a-admin", "tenant-a", "org-a"), "", ""), http.StatusUnauthorized, problem.TenantUnresolved)

	// A tenant that is not active resolves to nobody.
	if _, err := v.e.super.Exec(t.Context(), "UPDATE tenants SET state = 'provisioning' WHERE id = $1", v.e.b.id); err != nil {
		t.Fatal(err)
	}
	wantStatus(t, v.call(t, http.MethodGet, "/api/v1/me", v.tenantToken(t, "b-admin", "tenant-b", "org-b"), "", ""), http.StatusUnauthorized, problem.TenantUnresolved)
}

// covers: getMe 429
func TestGetMeRateLimited(t *testing.T) {
	v := newIDEnv(t, nil)
	before := v.footprint(t)
	v.lim.reset(false)
	rec := v.call(t, http.MethodGet, "/api/v1/me", v.tenantToken(t, "a-admin", "tenant-a", "org-a"), "", "")
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("= %d %s, want 429", rec.Code, rec.Body)
	}
	if rec.Header().Get("Retry-After") == "" {
		t.Error("429 without Retry-After")
	}
	// The limiter was keyed by the tenant the database resolved.
	if got := v.lim.calls(); len(got) != 1 || got[0] != v.e.a.id {
		t.Errorf("limiter saw %v, want tenant A %s", got, v.e.a.id)
	}
	sameFootprint(t, before, v.footprint(t))
}

// covers: getMe 500
func TestGetMeDatabaseDownIs500(t *testing.T) {
	dead := roleLoginPool(t, "nava_app", 2)
	v := newIDEnv(t, dead)
	dead.Close() // the identity store loses its database
	rec := v.call(t, http.MethodGet, "/api/v1/me", v.tenantToken(t, "a-admin", "tenant-a", "org-a"), "", "")
	wantStatus(t, rec, http.StatusInternalServerError, problem.InternalError)
	if strings.Contains(rec.Body.String(), "pgstore") || strings.Contains(rec.Body.String(), "5432") {
		t.Errorf("500 body leaks the cause: %s", rec.Body)
	}
}

// ---- onboardTenant ----

func onboardJSON(domain string) string {
	return fmt.Sprintf(`{"admin_email":"admin@%s","email_domains":[%q],"identity_provider":{"provider_id":"oidc","display_name":"Acme","config":{"clientId":"c","clientSecret":"fixture-not-a-secret","authorizationUrl":"https://idp.example.org/auth","tokenUrl":"https://idp.example.org/token","jwksUrl":"https://idp.example.org/jwks"}}}`, domain, domain)
}

func (v *idEnv) tenantRows(t *testing.T, id uuid.UUID) map[string]int {
	t.Helper()
	out := map[string]int{}
	for _, table := range []string{
		"tenant_identity_binding", "tenant_email_domain", "groups", "tenant_admin_group",
		"entity_definitions", "entity_definition_versions", "acl_versions",
	} {
		out[table] = v.count(t, "SELECT count(*) FROM "+table+" WHERE tenant_id = $1", id)
	}
	out["tenants"] = v.count(t, "SELECT count(*) FROM tenants WHERE id = $1", id)
	return out
}

// covers: onboardTenant 201
// covers: onboardTenant 200
func TestOnboardCreatesRowsAndReplayWritesOnce(t *testing.T) {
	v := newIDEnv(t, nil)
	op := v.operatorToken(t)
	key := "onboard-" + uuid.NewString()
	want := derivedTenantID(key)
	body := onboardJSON("acme-one.test")

	rec := v.call(t, http.MethodPost, "/api/v1/operator/tenants", op, key, body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("onboard = %d: %s", rec.Code, rec.Body)
	}
	var out struct {
		TenantID uuid.UUID `json:"tenant_id"`
		State    string    `json:"state"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil || out.TenantID != want || out.State != "active" {
		t.Fatalf("body = %s (%v), want tenant %s active", rec.Body, err, want)
	}

	var state string
	if err := v.e.super.QueryRow(t.Context(), "SELECT state FROM tenants WHERE id = $1", want).Scan(&state); err != nil || state != "active" {
		t.Fatalf("tenants.state = %q (%v), want active", state, err)
	}
	var issuer, org string
	if err := v.e.super.QueryRow(t.Context(), "SELECT issuer, org_id FROM tenant_identity_binding WHERE tenant_id = $1", want).Scan(&issuer, &org); err != nil ||
		issuer != v.realm.Issuer() || org != "org-"+want.String() {
		t.Errorf("binding = %q %q (%v), want %q org-%s", issuer, org, err, v.realm.Issuer(), want)
	}
	if n := v.count(t, "SELECT count(*) FROM tenant_email_domain WHERE tenant_id = $1 AND domain = 'acme-one.test'", want); n != 1 {
		t.Errorf("email domain rows = %d, want 1", n)
	}
	var groupPath string
	if err := v.e.super.QueryRow(t.Context(),
		"SELECT g.path FROM tenant_admin_group a JOIN groups g ON g.tenant_id = a.tenant_id AND g.id = a.group_id WHERE a.tenant_id = $1", want).Scan(&groupPath); err != nil ||
		groupPath != identity.TenantGroupPrefix(want)+identity.AdminGroupName {
		t.Errorf("admin group path = %q (%v)", groupPath, err)
	}
	types := v.count(t, "SELECT count(*) FROM entity_definitions WHERE tenant_id = $1", want)
	if types == 0 || v.count(t, "SELECT count(*) FROM entity_definition_versions WHERE tenant_id = $1 AND version = 1", want) != types {
		t.Errorf("seed: %d entity types, want >0 each at version 1", types)
	}
	if len(v.dir.inviteList()) != 1 || v.dir.inviteList()[0] != "admin@acme-one.test" {
		t.Errorf("invites = %v, want the admin once", v.dir.inviteList())
	}
	// P3-35: the incident type has the default ACL v1, naming the tenant's own
	// admins group, with its audit entry; the other types have none and fail
	// closed until an admin publishes one.
	var adminGroup uuid.UUID
	if err := v.e.super.QueryRow(t.Context(), "SELECT group_id FROM tenant_admin_group WHERE tenant_id = $1", want).Scan(&adminGroup); err != nil {
		t.Fatal(err)
	}
	var rules string
	if err := v.e.super.QueryRow(t.Context(), `SELECT a.rules::text FROM acl_versions a JOIN entity_definitions d ON d.tenant_id = a.tenant_id AND d.id = a.entity_definition_id
		WHERE a.tenant_id = $1 AND d.key = 'incident' AND a.version = 1 AND a.published_by IS NULL`, want).Scan(&rules); err != nil {
		t.Fatalf("incident ACL v1: %v", err)
	}
	for _, w := range []string{`"default-admins"`, `"default-requester"`, adminGroup.String()} {
		if !strings.Contains(rules, w) {
			t.Errorf("default incident ACL %s lacks %s", rules, w)
		}
	}
	if n := v.count(t, "SELECT count(*) FROM acl_versions WHERE tenant_id = $1", want); n != 1 {
		t.Errorf("%d ACL versions, want only the incident's", n)
	}
	if n := v.count(t, `SELECT count(*) FROM audit_log a JOIN entity_definitions d ON d.tenant_id = a.tenant_id AND d.id = a.subject_id
		WHERE a.tenant_id = $1 AND d.key = 'incident' AND a.domain = 'config' AND a.action = 'publish' AND a.version_after = 1
		  AND a.actor_system = 'onboarding' AND a.source = 'onboarding'`, want); n != 1 {
		t.Errorf("%d audit entries for the default ACL, want 1", n)
	}

	rows := v.tenantRows(t, want)
	all := v.footprint(t)
	again := v.call(t, http.MethodPost, "/api/v1/operator/tenants", op, key, body)
	if again.Code != http.StatusOK || !strings.Contains(again.Body.String(), want.String()) {
		t.Fatalf("replay = %d: %s, want 200 with the same tenant", again.Code, again.Body)
	}
	if fmt.Sprint(v.tenantRows(t, want)) != fmt.Sprint(rows) {
		t.Errorf("replay changed rows: %v -> %v", rows, v.tenantRows(t, want))
	}
	sameFootprint(t, all, v.footprint(t))
	if v.dir.createCount() != 1 || len(v.dir.inviteList()) != 1 {
		t.Errorf("replay called the directory again: %d creates, %d invites", v.dir.createCount(), len(v.dir.inviteList()))
	}
}

// A tenant id in the body, query or header is ignored: the row is the derived one.
func TestOnboardIgnoresSuppliedTenantID(t *testing.T) {
	v := newIDEnv(t, nil)
	key := "k-" + uuid.NewString()
	body := strings.Replace(onboardJSON("acme-two.test"), `{`, `{"tenant_id":"`+v.e.a.id.String()+`",`, 1)
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/operator/tenants?tenant_id="+v.e.a.id.String(), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+v.operatorToken(t))
	req.Header.Set("Idempotency-Key", key)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-Id", v.e.a.id.String())
	rec := httptest.NewRecorder()
	v.h.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("= %d: %s, want 201", rec.Code, rec.Body)
	}
	want := derivedTenantID(key)
	if !strings.Contains(rec.Body.String(), want.String()) || v.count(t, "SELECT count(*) FROM tenants WHERE id = $1", want) != 1 {
		t.Errorf("derived tenant %s has no row; body %s", want, rec.Body)
	}
	if v.count(t, "SELECT count(*) FROM tenants WHERE id = $1", v.e.a.id) != 1 || v.count(t, "SELECT count(*) FROM tenant_email_domain WHERE tenant_id = $1", v.e.a.id) != 0 {
		t.Error("tenant A was touched")
	}
	if n := v.count(t, "SELECT count(*) FROM tenant_email_domain WHERE tenant_id = $1", v.e.a.id); n != 0 {
		t.Errorf("tenant A gained %d email domains", n)
	}
}

// covers: onboardTenant 400
// covers: onboardTenant 422
func TestOnboardInvalidInputWritesNothing(t *testing.T) {
	v := newIDEnv(t, nil)
	op := v.operatorToken(t)
	before := v.footprint(t)
	post := func(key, body string) *httptest.ResponseRecorder {
		return v.call(t, http.MethodPost, "/api/v1/operator/tenants", op, key, body)
	}
	bad := map[string]string{
		"not JSON":              `{`,
		"not an object":         `[]`,
		"missing admin_email":   `{"email_domains":["acme.test"],"identity_provider":{"provider_id":"oidc","config":{}}}`,
		"empty email_domains":   strings.Replace(onboardJSON("acme.test"), `["acme.test"]`, `[]`, 1),
		"public email domain":   onboardJSON("gmail.com"),
		"admin outside domains": strings.Replace(onboardJSON("acme.test"), `admin@acme.test`, `admin@other.test`, 1),
		"malformed domain":      onboardJSON("not_a_domain"),
		"unknown provider":      strings.Replace(onboardJSON("acme.test"), `"oidc"`, `"ldap"`, 1),
	}
	for name, body := range bad {
		rec := post("bad-"+name, body)
		wantStatus(t, rec, http.StatusBadRequest, problem.ValidationFailed)
	}
	// An IdP URL that is not https fails after validation passes the shape.
	insecure := strings.Replace(onboardJSON("acme.test"), "https://idp.example.org/auth", "http://idp.example.org/auth", 1)
	wantStatus(t, post("insecure", insecure), http.StatusUnprocessableEntity, problem.InvalidIdPConfig)
	// No Idempotency-Key is 400.
	wantStatus(t, post("", onboardJSON("acme.test")), http.StatusBadRequest, problem.ValidationFailed)
	// No body is 400.
	wantStatus(t, post("nobody", ""), http.StatusBadRequest, problem.ValidationFailed)
	if v.dir.createCount() != 0 || len(v.dir.inviteList()) != 0 {
		t.Errorf("directory was called: %d creates, %d invites", v.dir.createCount(), len(v.dir.inviteList()))
	}
	sameFootprint(t, before, v.footprint(t))
}

// covers: onboardTenant 401
// covers: onboardTenant 403
func TestOnboardRefusesNonOperatorsAndWritesNothing(t *testing.T) {
	v := newIDEnv(t, nil)
	before := v.footprint(t)
	body := onboardJSON("acme-three.test")
	noRole := v.realm.Mint(t, keycloaktest.Token{Subject: operatorSub, Audiences: []string{"nava-operator-api"}})
	unlisted := v.realm.Mint(t, keycloaktest.Token{Subject: "someone-else", Audiences: []string{"nava-operator-api"}, RealmRoles: []string{"nava-operator"}})
	for _, c := range []struct {
		name, token, key string
		status           int
		code             problem.Code
	}{
		{"no token", "", "k1", 401, problem.InvalidToken},
		{"forged token", "a.b.c", "k2", 401, problem.InvalidToken},
		{"operator audience without the role", noRole, "k3", 401, problem.InvalidToken},
		{"role but not on the allow-list", unlisted, "k4", 401, problem.InvalidToken},
		{"tenant admin token", v.tenantToken(t, "a-admin", "tenant-a", "org-a"), "k5", 403, problem.Forbidden},
		{"tenant user token", v.tenantToken(t, "a-user", "tenant-a", "org-a"), "k6", 403, problem.Forbidden},
		{"tenant token and no key", v.tenantToken(t, "a-admin", "tenant-a", "org-a"), "", 403, problem.Forbidden},
	} {
		wantStatus(t, v.call(t, http.MethodPost, "/api/v1/operator/tenants", c.token, c.key, body), c.status, c.code)
	}
	if v.dir.createCount() != 0 {
		t.Errorf("directory was called %d times", v.dir.createCount())
	}
	sameFootprint(t, before, v.footprint(t))
}

// covers: onboardTenant 409
func TestOnboardConflicts(t *testing.T) {
	v := newIDEnv(t, nil)
	op := v.operatorToken(t)
	post := func(key, body string) *httptest.ResponseRecorder {
		return v.call(t, http.MethodPost, "/api/v1/operator/tenants", op, key, body)
	}
	if rec := post("first", onboardJSON("acme-four.test")); rec.Code != http.StatusCreated {
		t.Fatalf("first = %d: %s", rec.Code, rec.Body)
	}
	first := derivedTenantID("first")
	rows := v.tenantRows(t, first)

	// The same key with a different body is key reuse; the tenant is unchanged.
	before := v.footprint(t)
	wantStatus(t, post("first", onboardJSON("acme-five.test")), http.StatusConflict, problem.IdempotencyKeyReuse)
	sameFootprint(t, before, v.footprint(t))
	if fmt.Sprint(v.tenantRows(t, first)) != fmt.Sprint(rows) {
		t.Errorf("key reuse changed the tenant: %v", v.tenantRows(t, first))
	}

	// Another tenant asking for a claimed domain is refused and leaves nothing.
	wantStatus(t, post("second", onboardJSON("acme-four.test")), http.StatusConflict, problem.DomainClaimed)
	second := derivedTenantID("second")
	for table, n := range v.tenantRows(t, second) {
		if n != 0 {
			t.Errorf("claimed-domain refusal left %d rows in %s", n, table)
		}
	}
	sameFootprint(t, before, v.footprint(t))
	if n := v.count(t, "SELECT count(*) FROM tenant_email_domain WHERE domain = 'acme-four.test'"); n != 1 {
		t.Errorf("domain rows = %d, want 1 (still the first tenant's)", n)
	}

	// An organization that exists and is not this tenant's: onboarding_conflict.
	v.dir.createErr = identity.ErrAlreadyExists
	wantStatus(t, post("third", onboardJSON("acme-six.test")), http.StatusConflict, problem.OnboardingConflict)
	third := v.tenantRows(t, derivedTenantID("third"))
	if third["tenants"] != 1 || third["tenant_identity_binding"] != 0 || third["tenant_admin_group"] != 0 {
		t.Errorf("conflicted tenant rows = %v, want a provisioning tenant with no binding or admin group", third)
	}
	var state string
	if err := v.e.super.QueryRow(t.Context(), "SELECT state FROM tenants WHERE id = $1", derivedTenantID("third")).Scan(&state); err != nil || state != "provisioning" {
		t.Errorf("conflicted tenant state = %q (%v), want provisioning", state, err)
	}
}

// covers: onboardTenant 500
// A directory failure mid-onboarding is 500 with no cause in the body. The
// tenant stays provisioning and a retry with the same key and body finishes it.
func TestOnboardDirectoryFailureIs500ThenRetryResumes(t *testing.T) {
	v := newIDEnv(t, nil)
	op := v.operatorToken(t)
	v.dir.createErr = errors.New("keycloak: dial tcp 10.0.0.9:8080: secret-host-detail")
	body := onboardJSON("acme-seven.test")
	rec := v.call(t, http.MethodPost, "/api/v1/operator/tenants", op, "retry", body)
	wantStatus(t, rec, http.StatusInternalServerError, problem.InternalError)
	if strings.Contains(rec.Body.String(), "secret-host-detail") {
		t.Errorf("500 body leaks the cause: %s", rec.Body)
	}
	id := derivedTenantID("retry")
	var state string
	if err := v.e.super.QueryRow(t.Context(), "SELECT state FROM tenants WHERE id = $1", id).Scan(&state); err != nil || state != "provisioning" {
		t.Fatalf("state after failure = %q (%v), want provisioning", state, err)
	}
	v.dir.createErr = nil
	if rec := v.call(t, http.MethodPost, "/api/v1/operator/tenants", op, "retry", body); rec.Code != http.StatusCreated {
		t.Fatalf("retry = %d: %s", rec.Code, rec.Body)
	}
	if err := v.e.super.QueryRow(t.Context(), "SELECT state FROM tenants WHERE id = $1", id).Scan(&state); err != nil || state != "active" {
		t.Errorf("state after retry = %q (%v), want active", state, err)
	}
	if v.count(t, "SELECT count(*) FROM tenants WHERE id = $1", id) != 1 || v.count(t, "SELECT count(*) FROM tenant_identity_binding WHERE tenant_id = $1", id) != 1 {
		t.Error("retry did not leave exactly one tenant and one binding")
	}
}

// covers: onboardTenant 501
// Without the operator wiring the route answers 501 and writes nothing.
func TestOnboardNotConfiguredIs501(t *testing.T) {
	v := newIDEnv(t, nil)
	tel, err := telemetry.New(t.Context(), config.Telemetry{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tel.Shutdown(context.Background()) })
	adapter := v.realm.Resolver(t, pgstore.New(v.e.pool), operatorSub)
	h, err := server.NewHandler(slog.New(slog.DiscardHandler), tel, adapter, pgstore.New(v.e.pool), server.WithPool(v.e.pool))
	if err != nil {
		t.Fatal(err)
	}
	before := v.footprint(t)
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/operator/tenants", strings.NewReader(onboardJSON("acme-eight.test")))
	req.Header.Set("Authorization", "Bearer "+v.operatorToken(t))
	req.Header.Set("Idempotency-Key", "k")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	wantStatus(t, rec, http.StatusNotImplemented, problem.NotImplemented)
	sameFootprint(t, before, v.footprint(t))
}

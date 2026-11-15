//go:build spike

// Package s2 is ADR-007 spike S2 (TICK-807, RFC-017 rev 1 §7): a confidential
// client's service-account user, made a member of one Keycloak organization,
// gets a client_credentials token that the unchanged directory sync and the
// unchanged Resolve accept for that organization's tenant, and is refused
// once it is a member of two.
//
// Everything it calls is production code on main, unchanged:
// keycloak.Directory (the nava-groupsync account), groupsync.Syncer.Once,
// pgstore and keycloak.Adapter.Resolve. The database is built as `docker
// compose up` builds it (db-roles, then every migration). Run it with
// run.sh, which starts the throwaway Keycloak and PostgreSQL.
package s2

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/metric/noop"

	"github.com/Navaladi2019/TicketingSystem/db"
	"github.com/Navaladi2019/TicketingSystem/internal/db/audited"
	"github.com/Navaladi2019/TicketingSystem/internal/db/catalog"
	"github.com/Navaladi2019/TicketingSystem/internal/db/migrate"
	"github.com/Navaladi2019/TicketingSystem/internal/db/roles"
	"github.com/Navaladi2019/TicketingSystem/internal/groupsync"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/identity/keycloak"
	"github.com/Navaladi2019/TicketingSystem/internal/identity/pgstore"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
)

// The two seeded tenants of the compose realm: organization alias, tenant id
// (also the /t/<tenant_id> group).
func tenantA() uuid.UUID { return uuid.MustParse("00000000-0000-4000-8000-0000000000a1") }
func tenantB() uuid.UUID { return uuid.MustParse("00000000-0000-4000-8000-0000000000b1") }

const clientID = "s2-host-backend"

// ---- evidence -------------------------------------------------------------

type evidence struct {
	b       strings.Builder
	verdict map[string]bool
	order   []string
}

func (e *evidence) h(format string, a ...any) { fmt.Fprintf(&e.b, "\n## "+format+"\n\n", a...) }
func (e *evidence) p(format string, a ...any) { fmt.Fprintf(&e.b, format+"\n\n", a...) }
func (e *evidence) code(lang, s string) {
	fmt.Fprintf(&e.b, "```%s\n%s\n```\n\n", lang, strings.TrimRight(s, "\n"))
}

// cond records a pass condition of the ADR.
func (e *evidence) cond(id string, ok bool, format string, a ...any) {
	if e.verdict == nil {
		e.verdict = map[string]bool{}
	}
	if _, seen := e.verdict[id]; !seen {
		e.order = append(e.order, id)
	}
	e.verdict[id] = ok
	mark := "PASS"
	if !ok {
		mark = "FAIL"
	}
	fmt.Fprintf(&e.b, "- **%s %s**: %s\n\n", id, mark, fmt.Sprintf(format, a...))
}

// ---- Keycloak admin --------------------------------------------------------

func kcURL() string { return strings.TrimRight(os.Getenv("NAVA_S2_KEYCLOAK_URL"), "/") }

func httpDo(t *testing.T, req *http.Request) (int, []byte) {
	t.Helper()
	c := &http.Client{Timeout: 30 * time.Second}
	r, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Body.Close()
	b, _ := io.ReadAll(r.Body)
	return r.StatusCode, b
}

func tokenForm(t *testing.T, realm string, form url.Values) (int, map[string]any) {
	t.Helper()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost,
		kcURL()+"/realms/"+realm+"/protocol/openid-connect/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	code, b := httpDo(t, req)
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	return code, m
}

func masterToken(t *testing.T) string {
	t.Helper()
	code, out := tokenForm(t, "master", url.Values{
		"grant_type": {"password"}, "client_id": {"admin-cli"}, "username": {"admin"}, "password": {"dev-only-admin"},
	})
	if code != http.StatusOK {
		t.Fatalf("master admin token: %d %v", code, out)
	}
	return out["access_token"].(string)
}

// adm calls the admin API of realm nava as the master admin: what the
// operator API (ADR-007 D13) will do with the platform service account.
func adm(t *testing.T, method, path string, body any) (int, []byte) {
	t.Helper()
	var rdr io.Reader
	switch v := body.(type) {
	case nil:
	case string:
		rdr = strings.NewReader(v)
	default:
		b, _ := json.Marshal(v)
		rdr = bytes.NewReader(b)
	}
	req, _ := http.NewRequestWithContext(context.Background(), method, kcURL()+"/admin/realms/nava"+path, rdr)
	req.Header.Set("Authorization", "Bearer "+masterToken(t))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return httpDo(t, req)
}

func mustAdm(t *testing.T, want int, method, path string, body any) []byte {
	t.Helper()
	code, b := adm(t, method, path, body)
	if code != want {
		t.Fatalf("%s %s: %d (want %d) %s", method, path, code, want, b)
	}
	return b
}

func orgID(t *testing.T, alias string) string {
	t.Helper()
	var os []struct{ ID, Alias string }
	if err := json.Unmarshal(mustAdm(t, 200, http.MethodGet, "/organizations?exact=true&search="+alias, nil), &os); err != nil {
		t.Fatal(err)
	}
	for _, o := range os {
		if o.Alias == alias {
			return o.ID
		}
	}
	t.Fatalf("organization %s not found", alias)
	return ""
}

func scopeID(t *testing.T, name string) string {
	t.Helper()
	var ss []struct{ ID, Name string }
	if err := json.Unmarshal(mustAdm(t, 200, http.MethodGet, "/client-scopes", nil), &ss); err != nil {
		t.Fatal(err)
	}
	for _, s := range ss {
		if s.Name == name {
			return s.ID
		}
	}
	t.Fatalf("client scope %s not found", name)
	return ""
}

func groupID(t *testing.T, path string) string {
	t.Helper()
	var g struct{ ID string }
	if err := json.Unmarshal(mustAdm(t, 200, http.MethodGet, "/group-by-path"+path, nil), &g); err != nil {
		t.Fatal(err)
	}
	return g.ID
}

// claimsOf decodes a JWT payload without verifying it: only for the
// evidence; Resolve does the verification.
func claimsOf(t *testing.T, jwt string) map[string]any {
	t.Helper()
	parts := strings.Split(jwt, ".")
	if len(parts) != 3 {
		t.Fatalf("not a JWT")
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

// shown is the part of a token's claims the evidence prints. No secret is in
// a token; the subset keeps the record readable.
func shown(c map[string]any) string {
	keep := map[string]any{}
	for _, k := range []string{"iss", "sub", "typ", "azp", "aud", "scope", "organization", "idp", "preferred_username", "client_id"} {
		if v, ok := c[k]; ok {
			keep[k] = v
		}
	}
	if exp, ok := c["exp"].(float64); ok {
		if iat, ok := c["iat"].(float64); ok {
			keep["exp_minus_iat_s"] = exp - iat
		}
	}
	b, _ := json.MarshalIndent(keep, "", "  ")
	return string(b)
}

func orgEntries(c map[string]any) int {
	m, ok := c["organization"].(map[string]any)
	if !ok {
		return 0
	}
	return len(m)
}

// ---- platform ---------------------------------------------------------------

func pw(role string) string { return "test-pw-" + role }

type platform struct {
	super  *pgx.Conn
	app    *pgxpool.Pool
	syncer *groupsync.Syncer
	res    *keycloak.Adapter
}

// stubRegistry: ChangesSince, the only Directory method sync calls, reads no
// registry.
type stubRegistry struct{}

func (stubRegistry) UserKeycloakID(context.Context, uuid.UUID, uuid.UUID) (string, error) {
	return "", identity.ErrNotFound
}

func (stubRegistry) GroupKeycloakID(context.Context, uuid.UUID, uuid.UUID) (string, error) {
	return "", identity.ErrNotFound
}
func (stubRegistry) BoundIdPAliases(context.Context, uuid.UUID) ([]string, error) { return nil, nil }
func (stubRegistry) TenantOwnsEmailDomain(context.Context, uuid.UUID, string) (bool, error) {
	return false, nil
}

func outbound(t *testing.T) (*telemetry.OutboundClient, *telemetry.Telemetry) {
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
	out, err := telemetry.NewOutboundClient(tel, telemetry.OutboundOptions{Timeout: 20 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	return out, tel
}

func newPlatform(t *testing.T, iss, orgA, orgB string) *platform {
	t.Helper()
	ctx := t.Context()
	pgURL := os.Getenv("NAVA_S2_PG_URL")
	superCfg, err := pgx.ParseConfig(pgURL)
	if err != nil {
		t.Fatal(err)
	}
	m, am, err := audited.Load(db.Grants(), db.AuditedTables())
	if err != nil {
		t.Fatal(err)
	}
	gt, err := catalog.ParseGlobalTables(db.GlobalTables())
	if err != nil {
		t.Fatal(err)
	}
	log := slog.New(slog.DiscardHandler)
	if _, err := roles.Run(ctx, superCfg, m, func(r string) (string, bool) { return pw(r), true }, log); err != nil {
		t.Fatalf("db-roles: %v", err)
	}
	owner := superCfg.Copy()
	owner.User, owner.Password, owner.Database = m.Owner(), pw(m.Owner()), m.PlatformDatabase
	if _, err := migrate.Run(ctx, owner, db.Migrations(), m, am, gt, log, migrate.Options{LockRetrySeconds: 1}); err != nil {
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
		pc.ConnConfig.User, pc.ConnConfig.Password, pc.ConnConfig.Database = role, pw(role), m.PlatformDatabase
		pc.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeExec
		p, err := pgxpool.NewWithConfig(ctx, pc)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(p.Close)
		return p
	}
	// What onboarding leaves (T21): active tenants bound to (iss, org id).
	for _, b := range []struct {
		tenant uuid.UUID
		org    string
	}{{tenantA(), orgA}, {tenantB(), orgB}} {
		if _, err := super.Exec(ctx, "INSERT INTO tenants (id, state) VALUES ($1, 'active')", b.tenant); err != nil {
			t.Fatal(err)
		}
		if _, err := super.Exec(ctx, "INSERT INTO tenant_identity_binding (tenant_id, issuer, org_id, idp_aliases) VALUES ($1, $2, $3, $4)",
			b.tenant, iss, b.org, []string{"idp-" + map[uuid.UUID]string{tenantA(): "tenant-a", tenantB(): "tenant-b"}[b.tenant]}); err != nil {
			t.Fatal(err)
		}
	}
	out, tel := outbound(t)
	dir, err := keycloak.NewDirectory(keycloak.DirectoryConfig{
		BaseURL:  kcURL(),
		Runtime:  keycloak.ClientCredentials{ClientID: "nava-groupsync", ClientSecret: "dev-only-groupsync-secret"},
		Registry: stubRegistry{},
	}, out)
	if err != nil {
		t.Fatal(err)
	}
	syncer, err := groupsync.New(pool("nava_groupsync"), dir, noop.NewMeterProvider(), log, groupsync.Options{})
	if err != nil {
		t.Fatal(err)
	}
	res, err := keycloak.New(ctx, keycloak.Config{Issuers: []string{iss}, MeterProvider: tel.MeterProvider, Store: pgstore.New(pool("nava_app"))}, out)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = res.Close(context.Background()) })
	return &platform{super: super, app: pool("nava_app"), syncer: syncer, res: res}
}

func (p *platform) sync(t *testing.T) string {
	t.Helper()
	rs, err := p.syncer.Once(t.Context())
	if err != nil {
		t.Fatalf("group sync: %v", err)
	}
	var lines []string
	for _, r := range rs {
		if r.Err != nil || r.Skipped {
			t.Fatalf("group sync of %s: skipped=%v err=%v", r.TenantID, r.Skipped, r.Err)
		}
		lines = append(lines, fmt.Sprintf("tenant %s: ok (unchanged=%v)", r.TenantID, r.Unchanged))
	}
	return strings.Join(lines, "\n")
}

type principalRow struct {
	Tenant  uuid.UUID
	ID      uuid.UUID
	Email   *string
	Enabled bool
	Links   *string
	Deleted bool
}

// rows reads every principals row of a Keycloak user, in every tenant, as the
// superuser (no RLS), so a row in the wrong tenant would show.
func (p *platform) rows(t *testing.T, kcUser string) []principalRow {
	t.Helper()
	rs, err := p.super.Query(t.Context(), `SELECT tenant_id, id, email, enabled, federated_links::text, deleted_at IS NOT NULL
		FROM principals WHERE keycloak_user_id = $1 ORDER BY tenant_id`, kcUser)
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Close()
	var out []principalRow
	for rs.Next() {
		var r principalRow
		if err := rs.Scan(&r.Tenant, &r.ID, &r.Email, &r.Enabled, &r.Links, &r.Deleted); err != nil {
			t.Fatal(err)
		}
		out = append(out, r)
	}
	return out
}

func showRows(rs []principalRow) string {
	if len(rs) == 0 {
		return "(no principals row for this keycloak_user_id in any tenant)"
	}
	var b strings.Builder
	b.WriteString("tenant_id | email | enabled | federated_links | tombstoned\n")
	for _, r := range rs {
		email, links := "NULL", "NULL"
		if r.Email != nil {
			email = *r.Email
		}
		if r.Links != nil {
			links = *r.Links
		}
		fmt.Fprintf(&b, "%s | %s | %v | %s | %v\n", r.Tenant, email, r.Enabled, links, r.Deleted)
	}
	return b.String()
}

func describe(p identity.Principal, err error) string {
	switch {
	case err == nil:
		g := make([]string, 0, len(p.Groups))
		for _, x := range p.Groups {
			g = append(g, x.String())
		}
		sort.Strings(g)
		return fmt.Sprintf("OK tenant=%s principal=%s groups=%v idp=%q", p.TenantID, p.PrincipalID, g, p.IdPAlias)
	case errors.Is(err, identity.ErrTenantUnresolved):
		return "401 tenant_unresolved (" + err.Error() + ")"
	case errors.Is(err, identity.ErrInvalidToken):
		return "401 invalid_token (" + err.Error() + ")"
	}
	return "500 (" + err.Error() + ")"
}

// ---- the spike ---------------------------------------------------------------

func TestSpikeS2(t *testing.T) {
	if kcURL() == "" || os.Getenv("NAVA_S2_PG_URL") == "" {
		t.Fatal("run scratch/adr-007-s2/run.sh: it starts Keycloak and PostgreSQL and sets NAVA_S2_*")
	}
	e := &evidence{}
	defer func() {
		if path := os.Getenv("NAVA_S2_EVIDENCE"); path != "" {
			if err := os.WriteFile(path, []byte(e.b.String()), 0o600); err != nil {
				t.Error(err)
			}
		}
	}()
	e.p("# ADR-007 spike S2: raw evidence\n\nGenerated by `sh scratch/adr-007-s2/run.sh` on %s. Keycloak `quay.io/keycloak/keycloak:%s` (start-dev, realm `nava` imported from `deploy/compose/keycloak/import/realm-nava.json`), PostgreSQL from `docs/pins.md`, database built by `db-roles` + every migration. Production code on `main`, unchanged: `keycloak.Directory` as `nava-groupsync`, `groupsync.Syncer.Once`, `pgstore`, `keycloak.Adapter.Resolve`.",
		time.Now().UTC().Format(time.RFC3339), os.Getenv("NAVA_S2_KEYCLOAK_VERSION"))

	var info struct {
		SystemInfo struct{ Version string } `json:"systemInfo"`
	}
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, kcURL()+"/admin/serverinfo", nil)
	req.Header.Set("Authorization", "Bearer "+masterToken(t))
	if _, b := httpDo(t, req); json.Unmarshal(b, &info) == nil {
		e.p("Keycloak reports server version `%s` (`/admin/serverinfo`).", info.SystemInfo.Version)
	}

	orgA, orgB := orgID(t, "tenant-a"), orgID(t, "tenant-b")

	// Step 1: the confidential client, as the operator API would create it.
	e.h("Step 1: confidential client with a service account")
	rep := map[string]any{
		"clientId": clientID, "protocol": "openid-connect", "enabled": true,
		"publicClient": false, "serviceAccountsEnabled": true, "clientAuthenticatorType": "client-secret",
		"standardFlowEnabled": false, "implicitFlowEnabled": false, "directAccessGrantsEnabled": false,
		"fullScopeAllowed": false,
	}
	mustAdm(t, 201, http.MethodPost, "/clients", rep)
	var cs []struct{ ID string }
	_ = json.Unmarshal(mustAdm(t, 200, http.MethodGet, "/clients?clientId="+clientID, nil), &cs)
	if len(cs) != 1 {
		t.Fatalf("client %s not found", clientID)
	}
	cid := cs[0].ID
	// The tenant-token scopes nava-web has, minus the user-only ones: the
	// organization claim and the nava-api audience (what Resolve requires).
	var defaults []struct{ ID, Name string }
	_ = json.Unmarshal(mustAdm(t, 200, http.MethodGet, "/clients/"+cid+"/default-client-scopes", nil), &defaults)
	for _, d := range defaults {
		mustAdm(t, 204, http.MethodDelete, "/clients/"+cid+"/default-client-scopes/"+d.ID, nil)
	}
	for _, s := range []string{"basic", "roles", "organization", "nava-api-audience"} {
		mustAdm(t, 204, http.MethodPut, "/clients/"+cid+"/default-client-scopes/"+scopeID(t, s), nil)
	}
	var opt []struct{ ID string }
	_ = json.Unmarshal(mustAdm(t, 200, http.MethodGet, "/clients/"+cid+"/optional-client-scopes", nil), &opt)
	for _, o := range opt {
		mustAdm(t, 204, http.MethodDelete, "/clients/"+cid+"/optional-client-scopes/"+o.ID, nil)
	}
	var secret struct{ Value string }
	_ = json.Unmarshal(mustAdm(t, 200, http.MethodGet, "/clients/"+cid+"/client-secret", nil), &secret)
	var sa struct{ ID, Username, Email string }
	_ = json.Unmarshal(mustAdm(t, 200, http.MethodGet, "/clients/"+cid+"/service-account-user", nil), &sa)
	e.p("Client `%s`: confidential, `serviceAccountsEnabled`, every other grant off, `fullScopeAllowed=false`. Default client scopes `basic`, `roles`, `organization`, `nava-api-audience`; no optional scopes. Service-account user `%s` (email %q). The client secret is not recorded.", clientID, sa.Username, sa.Email)

	token := func(scope string) (int, string, map[string]any) {
		form := url.Values{"grant_type": {"client_credentials"}, "client_id": {clientID}, "client_secret": {secret.Value}}
		if scope != "" {
			form.Set("scope", scope)
		}
		code, out := tokenForm(t, "nava", form)
		tok, _ := out["access_token"].(string)
		if tok == "" {
			return code, "", out
		}
		return code, tok, claimsOf(t, tok)
	}

	// Baseline: before any organization membership.
	code, tok0, c0 := token("")
	e.p("Baseline token (no membership yet): HTTP %d, %d organization entries.", code, orgEntries(c0))
	if tok0 != "" {
		e.code("json", shown(c0))
	}
	iss, _ := c0["iss"].(string)
	if iss == "" {
		t.Fatal("no iss in the baseline token")
	}
	p := newPlatform(t, iss, orgA, orgB)

	// Step 2: one organization through the admin API.
	e.h("Step 2: service-account user added to organization tenant-a")
	code, b := adm(t, http.MethodPost, "/organizations/"+orgA+"/members", `"`+sa.ID+`"`)
	e.p("`POST /admin/realms/nava/organizations/{tenant-a}/members` with the service-account user id: HTTP %d %s", code, strings.TrimSpace(string(b)))
	if code != http.StatusCreated {
		e.cond("C1", false, "Keycloak refused the service-account user as an organization member")
		t.Fatalf("add member: %d %s", code, b)
	}
	code, tok1, c1 := token("")
	e.p("`client_credentials` token, no `scope` parameter: HTTP %d.", code)
	e.code("json", shown(c1))
	var org1 string
	if m, ok := c1["organization"].(map[string]any); ok {
		if a, ok := m["tenant-a"].(map[string]any); ok {
			org1, _ = a["id"].(string)
		}
	}
	e.cond("C1", orgEntries(c1) == 1 && org1 == orgA, "the token has exactly one `organization` entry, `tenant-a` with id %s (want %s)", org1, orgA)

	// Step 3: directory sync, as-is.
	e.h("Step 3: directory sync as-is (organization membership only)")
	e.code("text", p.sync(t))
	rows := p.rows(t, sa.ID)
	e.code("text", showRows(rows))
	asIs := len(rows) == 1 && rows[0].Tenant == tenantA() && rows[0].Links != nil && *rows[0].Links == "[]" && !rows[0].Deleted
	e.cond("C2-as-is", asIs, "unchanged sync creates a `principals` row in tenant A with `federated_links = []` (LinksKnown, no links)")
	pr, err := p.res.Resolve(t.Context(), tok1)
	e.p("`Resolve` (unchanged) on the step-2 token after the as-is sync: %s", describe(pr, err))
	e.cond("C3-as-is", err == nil && pr.TenantID == tenantA(), "`Resolve` returns tenant A after the as-is sync")

	// Step 3b: the smallest change that makes sync see it. Sync reads
	// membership of groups under /t/<tenant_id>/ (TICK-73 ruling 2026-10-25),
	// not organization membership; the service-account user is placed in the
	// tenant's root group /t/<tenant_a>, a Keycloak data change the operator
	// API can make. No platform code changes.
	e.h("Step 3b: candidate smallest change — service-account user in group /t/<tenant A>")
	root := groupID(t, "/t/"+tenantA().String())
	code, b = adm(t, http.MethodPut, "/users/"+sa.ID+"/groups/"+root, nil)
	e.p("`PUT /admin/realms/nava/users/{sa}/groups/{/t/%s}`: HTTP %d %s", tenantA(), code, strings.TrimSpace(string(b)))
	e.code("text", p.sync(t))
	rows = p.rows(t, sa.ID)
	e.code("text", showRows(rows))
	withGroup := len(rows) == 1 && rows[0].Tenant == tenantA() && rows[0].Links != nil && *rows[0].Links == "[]" && rows[0].Enabled && !rows[0].Deleted
	e.cond("C2", asIs || withGroup, "sync (code unchanged) creates exactly one `principals` row, in tenant A, enabled, `federated_links = []`, after the root-group placement")
	code, tok2, c2 := token("")
	e.p("Fresh token after the group placement: HTTP %d, %d organization entries (group membership adds no claim: scope set has no group mapper).", code, orgEntries(c2))
	pr, err = p.res.Resolve(t.Context(), tok2)
	e.p("`Resolve` (unchanged): %s", describe(pr, err))
	e.cond("C3", err == nil && pr.TenantID == tenantA(), "`Resolve` returns tenant A")
	pr, err = p.res.Resolve(t.Context(), tok1)
	e.p("`Resolve` on the step-2 token (minted before the group placement): %s", describe(pr, err))

	// Step 4: a second organization.
	e.h("Step 4: service-account user also added to organization tenant-b")
	code, b = adm(t, http.MethodPost, "/organizations/"+orgB+"/members", `"`+sa.ID+`"`)
	e.p("`POST .../organizations/{tenant-b}/members`: HTTP %d %s", code, strings.TrimSpace(string(b)))
	code, tok3, c3 := token("")
	e.p("`client_credentials` token, no `scope` parameter: HTTP %d.", code)
	e.code("json", shown(c3))
	pr, err = p.res.Resolve(t.Context(), tok3)
	e.p("`Resolve` (unchanged): %s", describe(pr, err))
	e.cond("C4", err != nil && errors.Is(err, identity.ErrTenantUnresolved), "same account in two organizations → 401 `tenant_unresolved` (the token carries %d `organization` entries)", orgEntries(c3))

	// Observations, not pass conditions: what a two-org account can still get
	// by asking for one organization, and that the earlier token is unchanged.
	e.h("Observations (not pass conditions)")
	for _, sc := range []string{"organization:tenant-a", "organization:tenant-b", "organization:*"} {
		code, tk, cc := token(sc)
		if tk == "" {
			e.p("Two-org account, `scope=%s`: HTTP %d %v", sc, code, cc)
			continue
		}
		pr, err := p.res.Resolve(t.Context(), tk)
		e.p("Two-org account, `scope=%s`: HTTP %d, organization = `%v`; `Resolve`: %s", sc, code, cc["organization"], describe(pr, err))
	}
	code, b = adm(t, http.MethodGet, "/organizations/members/"+sa.ID+"/organizations", nil)
	e.p("`GET .../organizations/members/{sa}/organizations`: HTTP %d %s", code, strings.TrimSpace(string(b)))

	// Verdict per RFC-017 §7 S2.
	e.h("Pass conditions")
	pass := e.verdict["C1"] && e.verdict["C2"] && e.verdict["C3"] && e.verdict["C4"]
	for _, id := range e.order {
		e.p("- %s: %v", id, e.verdict[id])
	}
	if pass {
		e.p("**SPIKE: PASS** (C2 as-is: %v; when false, the passing row needs the step-3b change).", e.verdict["C2-as-is"])
	} else {
		e.p("**SPIKE: FAIL**")
		t.Error("spike S2 failed; see the evidence")
	}
	t.Log(e.b.String())
}

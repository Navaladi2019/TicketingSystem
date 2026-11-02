//go:build integration

// P3-05a (TICK-343): the entity-type endpoints and the definition store
// against a real PostgreSQL, through the real handler (authn, ratelimit,
// idempotency, validator, strict server). make test-integration
// (scripts/ci/test-integration.sh) starts a disposable cluster and sets
// NAVA_TEST_PG_URL to its superuser. The tests drop and recreate the manifest
// roles and databases, so the script runs the packages one at a time.

package server_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/db"
	"github.com/Navaladi2019/TicketingSystem/internal/db/catalog"
	"github.com/Navaladi2019/TicketingSystem/internal/db/grants"
	"github.com/Navaladi2019/TicketingSystem/internal/db/migrate"
	"github.com/Navaladi2019/TicketingSystem/internal/db/roles"
	"github.com/Navaladi2019/TicketingSystem/internal/entitytypes"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/metadata"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/cursor"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
	"github.com/Navaladi2019/TicketingSystem/internal/server"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

func etPassword(role string) string { return "test-pw-" + role }

func etSuperConfig(t *testing.T) *pgx.ConnConfig {
	t.Helper()
	url := os.Getenv("NAVA_TEST_PG_URL")
	if url == "" {
		t.Fatal("NAVA_TEST_PG_URL is not set; run make test-integration")
	}
	cfg, err := pgx.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

func etIdent(s string) string { return pgx.Identifier{s}.Sanitize() }

// etCluster rebuilds the manifest roles and databases, runs the real
// migrations, and returns a superuser connection to the platform database and
// a pool that logs in as nava_app.
func etCluster(t *testing.T) (*pgx.Conn, *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	m, err := grants.Parse(db.Grants())
	if err != nil {
		t.Fatal(err)
	}
	admin, err := pgx.ConnectConfig(ctx, etSuperConfig(t))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = admin.Close(ctx) })
	// Template databases from the dbtest package may own objects.
	rows, err := admin.Query(ctx, "SELECT datname FROM pg_database WHERE datname LIKE 'nava_tpl_%'")
	if err != nil {
		t.Fatal(err)
	}
	var tpls []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			t.Fatal(err)
		}
		tpls = append(tpls, n)
	}
	rows.Close()
	for _, n := range tpls {
		for _, q := range []string{"ALTER DATABASE " + etIdent(n) + " WITH IS_TEMPLATE false", "DROP DATABASE " + etIdent(n)} {
			if _, err := admin.Exec(ctx, q); err != nil {
				t.Fatal(err)
			}
		}
	}
	for _, d := range m.DatabaseNames() {
		if _, err := admin.Exec(ctx, "DROP DATABASE IF EXISTS "+etIdent(d)+" WITH (FORCE)"); err != nil {
			t.Fatal(err)
		}
	}
	for _, r := range m.RoleNames() {
		if _, err := admin.Exec(ctx, "DROP ROLE IF EXISTS "+etIdent(r)); err != nil {
			t.Fatal(err)
		}
	}
	log := slog.New(slog.DiscardHandler)
	pw := func(role string) (string, bool) { return etPassword(role), true }
	if _, err := roles.Run(ctx, etSuperConfig(t), m, pw, log); err != nil {
		t.Fatalf("db-roles: %v", err)
	}
	owner := etSuperConfig(t).Copy()
	owner.User, owner.Password, owner.Database = m.Owner(), etPassword(m.Owner()), m.PlatformDatabase
	gt, err := catalog.ParseGlobalTables(db.GlobalTables())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := migrate.Run(ctx, owner, db.Migrations(), m, gt, log, migrate.Options{LockRetrySeconds: 1}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	sup := etSuperConfig(t).Copy()
	sup.Database = m.PlatformDatabase
	super, err := pgx.ConnectConfig(ctx, sup)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = super.Close(ctx) })

	pc, err := pgxpool.ParseConfig(os.Getenv("NAVA_TEST_PG_URL"))
	if err != nil {
		t.Fatal(err)
	}
	pc.ConnConfig.User, pc.ConnConfig.Password, pc.ConnConfig.Database = "nava_app", etPassword("nava_app"), m.PlatformDatabase
	pc.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeExec
	pc.MaxConns = 8
	pool, err := pgxpool.NewWithConfig(ctx, pc)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return super, pool
}

// etTenant is a seeded tenant: an admin (member of the group
// tenant_admin_group names), a plain user, and a user in a group that is
// called admins but is not the tenant's admin group.
type etTenant struct {
	id                         uuid.UUID
	adminGroup, lookalikeGroup uuid.UUID
	admin, user, lookalike     identity.Principal
}

func etSeedTenant(t *testing.T, super *pgx.Conn, tag string) etTenant {
	t.Helper()
	ctx := context.Background()
	tn := etTenant{id: uuid.New(), adminGroup: uuid.New(), lookalikeGroup: uuid.New()}
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := super.Exec(ctx, sql, args...); err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
	}
	exec("INSERT INTO tenants (id, state) VALUES ($1, 'active')", tn.id)
	prefix := "/t/" + tn.id.String()
	exec("INSERT INTO groups (tenant_id, id, keycloak_group_id, path) VALUES ($1, $2, $3, $4)", tn.id, tn.adminGroup, tag+"-admins", prefix+"/admins")
	exec("INSERT INTO groups (tenant_id, id, keycloak_group_id, path) VALUES ($1, $2, $3, $4)", tn.id, tn.lookalikeGroup, tag+"-renamed", prefix+"/admins")
	exec("INSERT INTO tenant_admin_group (tenant_id, group_id) VALUES ($1, $2)", tn.id, tn.adminGroup)
	mk := func(name string, groups ...uuid.UUID) identity.Principal {
		p := identity.Principal{TenantID: tn.id, PrincipalID: uuid.New(), Groups: groups}
		exec("INSERT INTO principals (tenant_id, id, keycloak_user_id, email, enabled, federated_links, synced_at) VALUES ($1, $2, $3, $4, true, '[]', now())",
			tn.id, p.PrincipalID, tag+"-"+name, name+"@"+tag+".test")
		for _, g := range groups {
			exec("INSERT INTO group_members (tenant_id, group_id, principal_id) VALUES ($1, $2, $3)", tn.id, g, p.PrincipalID)
		}
		return p
	}
	tn.admin = mk("admin", tn.adminGroup)
	tn.user = mk("user")
	tn.lookalike = mk("lookalike", tn.lookalikeGroup)
	return tn
}

type etTokens map[string]identity.Principal

func (e etTokens) Resolve(_ context.Context, raw string) (identity.Principal, error) {
	if p, ok := e[raw]; ok {
		return p, nil
	}
	return identity.Principal{}, fmt.Errorf("%w: unknown token", identity.ErrInvalidToken)
}

// etEnv is the handler over a migrated database with tenants A and B.
type etEnv struct {
	h      http.Handler
	tokens etTokens
	pool   *pgxpool.Pool
	a, b   etTenant
	super  *pgx.Conn
}

func newETEnv(t *testing.T) *etEnv {
	t.Helper()
	super, pool := etCluster(t)
	a, b := etSeedTenant(t, super, "a"), etSeedTenant(t, super, "b")
	tokens := etTokens{
		"admin-a": a.admin, "user-a": a.user, "lookalike-a": a.lookalike,
		"admin-b": b.admin, "user-b": b.user,
	}
	tel, err := telemetry.New(t.Context(), config.Telemetry{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tel.Shutdown(context.Background()) })
	kr, err := cursor.NewKeyring("k1", cursor.Key{ID: "k1", Secret: bytes.Repeat([]byte("s"), 32)})
	if err != nil {
		t.Fatal(err)
	}
	lim := &spyLimiter{}
	lim.reset(true)
	h, err := server.NewHandler(slog.New(slog.NewTextHandler(testLogWriter{t}, &slog.HandlerOptions{Level: slog.LevelWarn})), tel, tokens, stubPaths{},
		server.WithLimiter(lim), server.WithPool(pool), server.WithCursors(kr))
	if err != nil {
		t.Fatal(err)
	}
	return &etEnv{h: h, tokens: tokens, pool: pool, a: a, b: b, super: super}
}

func (e *etEnv) do(t *testing.T, method, path, token, idemKey, body string) *httptest.ResponseRecorder {
	t.Helper()
	var rd *strings.Reader
	if body != "" {
		rd = strings.NewReader(body)
	} else {
		rd = strings.NewReader("")
	}
	req := httptest.NewRequestWithContext(t.Context(), method, path, rd)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if idemKey != "" {
		req.Header.Set("Idempotency-Key", idemKey)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	e.h.ServeHTTP(rec, req)
	return rec
}

// etSchema is a valid schema, indented and with a duplicate-free but
// unusual layout, so a re-encoding would change the bytes.
const etSchema = `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "additionalProperties": false,
  "required": ["short_description", "state"],
  "properties": {
    "short_description": {"type": "string", "minLength": 1, "maxLength": 160},
    "state": {"type": "string", "enum": ["new", "closed"], "default": "new"},
    "note": {"type": "string", "description": "a <b>&amp; tag"}
  }
}`

const etDefTail = `"fixed_columns": {"state": "state"},
    "states": {"initial": "new", "states": ["new", "closed"], "transitions": [{"from": "new", "to": "closed"}]}`

func etBody(key, prefix string) string {
	return fmt.Sprintf(`{"key": %q, "name": "Type %s", "number_prefix": %q, "definition": {"schema": %s, %s}}`,
		key, key, prefix, etSchema, etDefTail)
}

type etProblem struct {
	Code   problem.Code         `json:"code"`
	Errors []problem.FieldError `json:"errors"`
}

func wantProblem(t *testing.T, rec *httptest.ResponseRecorder, status int, code problem.Code) etProblem {
	t.Helper()
	var p etProblem
	if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
		t.Fatalf("body is not JSON: %v: %s", err, rec.Body)
	}
	if rec.Code != status || p.Code != code {
		t.Fatalf("status %d code %q, want %d %q: %s", rec.Code, p.Code, status, code, rec.Body)
	}
	return p
}

func pointers(p etProblem) []string {
	out := make([]string, len(p.Errors))
	for i, e := range p.Errors {
		out[i] = e.Pointer
	}
	return out
}

func TestCreateThenReadVersionIsByteIdentical(t *testing.T) {
	e := newETEnv(t)
	rec := e.do(t, http.MethodPost, "/api/v1/entity-types", "admin-a", "k-create-1", etBody("incident", "INC"))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create = %d: %s", rec.Code, rec.Body)
	}
	for _, path := range []string{"/api/v1/entity-types/incident/versions/1", "/api/v1/entity-types/incident"} {
		for _, token := range []string{"admin-a", "user-a"} { // reads are open to any tenant user
			got := e.do(t, http.MethodGet, path, token, "", "")
			if got.Code != http.StatusOK {
				t.Fatalf("GET %s as %s = %d: %s", path, token, got.Code, got.Body)
			}
			if !strings.Contains(got.Body.String(), `"schema":`+etSchema+`,"schema_sha256"`) {
				t.Errorf("GET %s: schema bytes changed; body:\n%s", path, got.Body)
			}
			var v struct {
				Key     string `json:"key"`
				Version int    `json:"version"`
				Current bool   `json:"current"`
				Compat  string `json:"compat"`
				SHA     string `json:"schema_sha256"`
			}
			if err := json.Unmarshal(got.Body.Bytes(), &v); err != nil {
				t.Fatal(err)
			}
			sum := sha256.Sum256([]byte(etSchema))
			if v.Key != "incident" || v.Version != 1 || !v.Current || v.Compat != "initial" || v.SHA != hex.EncodeToString(sum[:]) {
				t.Errorf("GET %s = %+v, want incident v1 current initial sha %x", path, v, sum)
			}
		}
	}
	// The create response carries the same bytes.
	if !strings.Contains(rec.Body.String(), `"schema":`+etSchema+`,"schema_sha256"`) {
		t.Errorf("create response changed the schema bytes:\n%s", rec.Body)
	}
	// And so does the database: the table CHECK ties the hash to them.
	var stored string
	if err := e.super.QueryRow(t.Context(), "SELECT schema::text FROM entity_definition_versions WHERE tenant_id = $1", e.a.id).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if stored != etSchema {
		t.Errorf("stored schema differs from the request bytes:\n%q\n%q", stored, etSchema)
	}
}

func TestCreateRequiresTheTenantAdminGroup(t *testing.T) {
	e := newETEnv(t)
	for _, token := range []string{"user-a", "lookalike-a"} {
		rec := e.do(t, http.MethodPost, "/api/v1/entity-types", token, "k-"+token, etBody("incident", "INC"))
		wantProblem(t, rec, http.StatusForbidden, problem.Forbidden)
		// The check precedes the body: a bad body is still 403, not 422.
		rec = e.do(t, http.MethodPost, "/api/v1/entity-types", token, "k-bad-"+token, `{"key": 5}`)
		wantProblem(t, rec, http.StatusForbidden, problem.Forbidden)
	}
	// Tenant B's admin is no admin of tenant A: it creates in tenant B only.
	if rec := e.do(t, http.MethodPost, "/api/v1/entity-types", "admin-b", "k-b", etBody("incident", "INC")); rec.Code != http.StatusCreated {
		t.Fatalf("tenant B create = %d: %s", rec.Code, rec.Body)
	}
	var n int
	if err := e.super.QueryRow(t.Context(), "SELECT count(*) FROM entity_definitions WHERE tenant_id = $1", e.a.id).Scan(&n); err != nil || n != 0 {
		t.Errorf("tenant A has %d entity types (err %v), want 0", n, err)
	}
}

func TestCreateConflictsAndInvalidDefinitions(t *testing.T) {
	e := newETEnv(t)
	post := func(key, body string) *httptest.ResponseRecorder {
		return e.do(t, http.MethodPost, "/api/v1/entity-types", "admin-a", key, body)
	}
	if rec := post("k1", etBody("incident", "INC")); rec.Code != http.StatusCreated {
		t.Fatalf("create = %d: %s", rec.Code, rec.Body)
	}
	wantProblem(t, post("k2", etBody("incident", "ZZZ")), http.StatusConflict, problem.EntityTypeExists)
	wantProblem(t, post("k3", etBody("other", "INC")), http.StatusConflict, problem.NumberPrefixTaken)
	// Both clash: the key wins.
	wantProblem(t, post("k4", etBody("incident", "INC")), http.StatusConflict, problem.EntityTypeExists)
	// The same names in tenant B are free.
	if rec := e.do(t, http.MethodPost, "/api/v1/entity-types", "admin-b", "kb", etBody("incident", "INC")); rec.Code != http.StatusCreated {
		t.Fatalf("tenant B create = %d: %s", rec.Code, rec.Body)
	}

	p := wantProblem(t, post("k5", etBody("Bad Key", "x")), http.StatusUnprocessableEntity, problem.InvalidDefinition)
	if got := pointers(p); len(got) != 2 || got[0] != "/key" || got[1] != "/number_prefix" {
		t.Errorf("pointers = %v, want [/key /number_prefix]", got)
	}
	p = wantProblem(t, post("k6", `{"key": "x1", "name": "", "number_prefix": "AB", "definition": {"schema": {"type": "array"}, "fixed_columns": {}, "states": {}}}`),
		http.StatusUnprocessableEntity, problem.InvalidDefinition)
	got := pointers(p)
	if len(got) < 2 || got[0] != "/name" {
		t.Fatalf("pointers = %v, want /name first, then /definition/...", got)
	}
	for _, g := range got[1:] {
		if !strings.HasPrefix(g, "/definition/") {
			t.Errorf("pointer %q is not under /definition/", g)
		}
	}
	wantProblem(t, post("k7", `{"key": "x2", "name": "n", "number_prefix": "AB"}`), http.StatusUnprocessableEntity, problem.InvalidDefinition)
	wantProblem(t, post("k8", `[]`), http.StatusBadRequest, problem.ValidationFailed) // not an object: the contract
	// No Idempotency-Key.
	wantProblem(t, e.do(t, http.MethodPost, "/api/v1/entity-types", "admin-a", "", etBody("n1", "NN")), http.StatusBadRequest, problem.ValidationFailed)
	// No token.
	wantProblem(t, e.do(t, http.MethodPost, "/api/v1/entity-types", "", "k9", etBody("n1", "NN")), http.StatusUnauthorized, problem.InvalidToken)
	var n int
	if err := e.super.QueryRow(t.Context(), "SELECT count(*) FROM entity_definitions WHERE tenant_id = $1", e.a.id).Scan(&n); err != nil || n != 1 {
		t.Errorf("tenant A has %d entity types (err %v), want 1: a refused create must leave nothing", n, err)
	}
}

func TestCreateIdempotentReplay(t *testing.T) {
	e := newETEnv(t)
	body := etBody("incident", "INC")
	first := e.do(t, http.MethodPost, "/api/v1/entity-types", "admin-a", "same-key", body)
	if first.Code != http.StatusCreated {
		t.Fatalf("create = %d: %s", first.Code, first.Body)
	}
	again := e.do(t, http.MethodPost, "/api/v1/entity-types", "admin-a", "same-key", body)
	if again.Code != http.StatusCreated || again.Body.String() != first.Body.String() || again.Header().Get("Idempotent-Replayed") != "true" {
		t.Errorf("replay = %d replayed=%q, want the same 201 body; got:\n%s\nwant:\n%s", again.Code, again.Header().Get("Idempotent-Replayed"), again.Body, first.Body)
	}
	other := e.do(t, http.MethodPost, "/api/v1/entity-types", "admin-a", "same-key", etBody("changed", "CHG"))
	wantProblem(t, other, http.StatusConflict, problem.IdempotencyKeyReuse)
	var n int
	if err := e.super.QueryRow(t.Context(), "SELECT count(*) FROM entity_definitions WHERE tenant_id = $1", e.a.id).Scan(&n); err != nil || n != 1 {
		t.Errorf("tenant A has %d entity types (err %v), want 1", n, err)
	}
}

// etPrefix is a distinct two-to-four letter prefix for the i-th type.
func etPrefix(i int) string {
	s := ""
	for n := i + 26; n > 0; n /= 26 {
		s = string(rune('A'+n%26)) + s
	}
	return s
}

func TestListPagesWithoutDuplicates(t *testing.T) {
	e := newETEnv(t)
	const total = 120
	for i := range total {
		in := entitytypes.NewInput{Name: "T", PublishedBy: e.a.admin.PrincipalID, Definition: etDefinition(t, fmt.Sprintf("type_%03d", i), etPrefix(i))}
		if err := tenancy.InTenantTx(t.Context(), e.pool, e.a.id, func(tx pgx.Tx) error {
			_, err := entitytypes.Store{}.Create(t.Context(), tx, e.a.id, in)
			return err
		}); err != nil {
			t.Fatalf("seed %d: %v", i, err)
		}
	}
	// Tenant B has one type; it never shows up in A's list.
	if rec := e.do(t, http.MethodPost, "/api/v1/entity-types", "admin-b", "kb", etBody("only_b", "BB")); rec.Code != http.StatusCreated {
		t.Fatal(rec.Body)
	}

	type page struct {
		Items []struct {
			Key            string `json:"key"`
			CurrentVersion int    `json:"current_version"`
		} `json:"items"`
		Next string `json:"next_cursor"`
	}
	get := func(q string) (*httptest.ResponseRecorder, page) {
		rec := e.do(t, http.MethodGet, "/api/v1/entity-types"+q, "user-a", "", "")
		var p page
		if rec.Code == http.StatusOK {
			if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
				t.Fatal(err)
			}
		}
		return rec, p
	}

	wantProblem(t, func() *httptest.ResponseRecorder { r, _ := get("?limit=201"); return r }(), http.StatusBadRequest, problem.ValidationFailed)
	wantProblem(t, func() *httptest.ResponseRecorder { r, _ := get("?limit=0"); return r }(), http.StatusBadRequest, problem.ValidationFailed)
	wantProblem(t, func() *httptest.ResponseRecorder { r, _ := get("?cursor=garbage"); return r }(), http.StatusBadRequest, problem.InvalidCursor)

	if rec, p := get(""); rec.Code != http.StatusOK || len(p.Items) != 50 || p.Next == "" {
		t.Fatalf("default page: %d, %d items, next %q; want 200, 50 items, a cursor", rec.Code, len(p.Items), p.Next)
	}

	seen := map[string]bool{}
	var order []string
	q, pages := "?limit=50", 0
	for {
		rec, p := get(q)
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s = %d: %s", q, rec.Code, rec.Body)
		}
		pages++
		for _, it := range p.Items {
			if seen[it.Key] {
				t.Errorf("duplicate %s", it.Key)
			}
			seen[it.Key] = true
			order = append(order, it.Key)
		}
		if p.Next == "" {
			break
		}
		q = "?limit=50&cursor=" + p.Next
		if pages > 10 {
			t.Fatal("paging does not end")
		}
	}
	if len(seen) != total || pages != 3 {
		t.Errorf("walked %d types in %d pages, want %d in 3", len(seen), pages, total)
	}
	for i := 1; i < len(order); i++ {
		if order[i-1] >= order[i] {
			t.Fatalf("not ordered by key: %s before %s", order[i-1], order[i])
		}
	}
	// A cursor is bound to the caller: tenant B's user cannot use A's.
	_, p := get("?limit=10")
	rec := e.do(t, http.MethodGet, "/api/v1/entity-types?limit=10&cursor="+p.Next, "user-b", "", "")
	wantProblem(t, rec, http.StatusBadRequest, problem.InvalidCursor)
}

func TestUnknownAndForeignKeysGiveTheSameNotFound(t *testing.T) {
	e := newETEnv(t)
	if rec := e.do(t, http.MethodPost, "/api/v1/entity-types", "admin-a", "ka", etBody("secret_type", "SEC")); rec.Code != http.StatusCreated {
		t.Fatal(rec.Body)
	}
	for _, suffix := range []string{"", "/versions/1"} {
		unknown := e.do(t, http.MethodGet, "/api/v1/entity-types/no_such_type"+suffix, "user-b", "", "")
		foreign := e.do(t, http.MethodGet, "/api/v1/entity-types/secret_type"+suffix, "user-b", "", "")
		if unknown.Code != http.StatusNotFound || foreign.Code != http.StatusNotFound {
			t.Fatalf("statuses %d, %d; want 404, 404", unknown.Code, foreign.Code)
		}
		// Same bytes apart from nothing: the key is not echoed, and there is no instance.
		if unknown.Body.String() != foreign.Body.String() {
			t.Errorf("404 bodies differ:\n%s\n%s", unknown.Body, foreign.Body)
		}
		if strings.Contains(foreign.Body.String(), "instance") || strings.Contains(foreign.Body.String(), "secret_type") {
			t.Errorf("404 body carries instance or the key: %s", foreign.Body)
		}
		if foreign.Header().Get("Content-Type") != problem.ContentType {
			t.Errorf("Content-Type = %q", foreign.Header().Get("Content-Type"))
		}
	}
	// A missing version, version 0 and a negative one are the same 404 too.
	base := e.do(t, http.MethodGet, "/api/v1/entity-types/no_such_type", "user-a", "", "").Body.String()
	for _, path := range []string{"/api/v1/entity-types/secret_type/versions/2", "/api/v1/entity-types/secret_type/versions/0", "/api/v1/entity-types/secret_type/versions/-1", "/api/v1/entity-types/" + strings.Repeat("k", 200)} {
		rec := e.do(t, http.MethodGet, path, "user-a", "", "")
		if rec.Code != http.StatusNotFound || rec.Body.String() != base {
			t.Errorf("GET %s = %d %s, want the same 404", path, rec.Code, rec.Body)
		}
	}
	wantProblem(t, e.do(t, http.MethodGet, "/api/v1/entity-types/secret_type/versions/abc", "user-a", "", ""), http.StatusBadRequest, problem.ValidationFailed)
	wantProblem(t, e.do(t, http.MethodGet, "/api/v1/entity-types/secret_type", "", "", ""), http.StatusUnauthorized, problem.InvalidToken)
}

// etDefinition is a valid metadata.Definition with etSchema.
func etDefinition(t *testing.T, key, prefix string) metadata.Definition {
	t.Helper()
	return metadata.Definition{
		Key: key, Schema: json.RawMessage(etSchema), NumberPrefix: prefix,
		FixedColumns: map[string]string{"state": "state"},
		States: metadata.StateModel{Initial: "new", States: []string{"new", "closed"},
			Transitions: []metadata.Transition{{From: "new", To: "closed"}}},
	}
}

func TestDefinitionReaderIsTenantScoped(t *testing.T) {
	e := newETEnv(t)
	if rec := e.do(t, http.MethodPost, "/api/v1/entity-types", "admin-a", "ka", etBody("incident", "INC")); rec.Code != http.StatusCreated {
		t.Fatal(rec.Body)
	}
	var rd entitytypes.DefinitionReader = entitytypes.Store{}
	ctx := t.Context()

	if err := tenancy.InTenantTx(ctx, e.pool, e.a.id, func(tx pgx.Tx) error {
		c, err := rd.Version(ctx, tx, "incident", 1)
		if err != nil || c == nil {
			return fmt.Errorf("Version = %v, %v", c, err)
		}
		cur, n, err := rd.Current(ctx, tx, "incident")
		if err != nil || cur == nil || n != 1 {
			return fmt.Errorf("Current = %v, %d, %v", cur, n, err)
		}
		if _, err := rd.Version(ctx, tx, "incident", 2); !errors.Is(err, entitytypes.ErrNotFound) {
			return fmt.Errorf("Version 2 = %v, want ErrNotFound", err)
		}
		if _, err := rd.Version(ctx, tx, "incident", 0); !errors.Is(err, entitytypes.ErrNotFound) {
			return fmt.Errorf("Version 0 = %v, want ErrNotFound", err)
		}
		if _, err := rd.Version(ctx, tx, "nope", 1); !errors.Is(err, entitytypes.ErrNotFound) {
			return fmt.Errorf("unknown key = %v, want ErrNotFound", err)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	// Tenant B's transaction cannot see tenant A's key.
	if err := tenancy.InTenantTx(ctx, e.pool, e.b.id, func(tx pgx.Tx) error {
		if _, err := rd.Version(ctx, tx, "incident", 1); !errors.Is(err, entitytypes.ErrNotFound) {
			return fmt.Errorf("Version under B = %v, want ErrNotFound", err)
		}
		if _, _, err := rd.Current(ctx, tx, "incident"); !errors.Is(err, entitytypes.ErrNotFound) {
			return fmt.Errorf("Current under B = %v, want ErrNotFound", err)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

// Two parallel creates of one key (each with its own Idempotency-Key, so the
// idempotency layer does not serialise them) race to the unique index. One is
// 201; the other must be the registered 409, never a 500 (mapUnique).
func TestParallelCreateOfOneKeyGivesOne201AndOne409(t *testing.T) {
	e := newETEnv(t)
	const n = 2
	recs := make([]*httptest.ResponseRecorder, n)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			recs[i] = e.do(t, http.MethodPost, "/api/v1/entity-types", "admin-a", fmt.Sprintf("par-%d", i), etBody("race_key", "RAC"))
		}()
	}
	close(start)
	wg.Wait()
	created, exists := 0, 0
	for _, rec := range recs {
		switch rec.Code {
		case http.StatusCreated:
			created++
		case http.StatusConflict:
			wantProblem(t, rec, http.StatusConflict, problem.EntityTypeExists)
			exists++
		default:
			t.Errorf("parallel create = %d, want 201 or 409 entity_type_exists: %s", rec.Code, rec.Body)
		}
	}
	if created != 1 || exists != 1 {
		t.Errorf("got %d x 201 and %d x 409, want one of each", created, exists)
	}
}

// testLogWriter sends the server's warnings and errors to the test log, so a
// 500 in CI shows its wrapped cause.
type testLogWriter struct{ t *testing.T }

func (w testLogWriter) Write(p []byte) (int, error) {
	w.t.Helper()
	w.t.Log(strings.TrimRight(string(p), "\n"))
	return len(p), nil
}

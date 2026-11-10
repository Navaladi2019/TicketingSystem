//go:build integration && perf

// P3-36 (TICK-621): list p95 <= 150 ms and single read p95 <= 50 ms at 800k
// records, measured through the real HTTP handlers (GET
// /api/v1/records/{entity} and GET /api/v1/records/{entity}/{id}). The SQL is
// whatever the handlers emit: authz.CompileRead (P3-31) inside the P3-34 list
// builder (records/store.List), captured per request by a pgx QueryTracer on
// the handler's own pool. Nothing here hand-writes a list or ACL query.
//
// Data: P3-01's shape (bench/records-list/02_seed.sql, TICK-339), on the real
// migrated schema. One tenant has 400,000 records, 399 tenants have 1,002 or
// 1,003 each. Every tenant's entity type and ACL go through the real API
// (POST /api/v1/entity-types, acltest.Publish); records, principals, groups
// and shares are bulk-inserted as the superuser.
//
// Run with make bench-acl-perf (bench/acl-perf/run.sh), which starts a
// PostgreSQL with P3-01's memory settings. Not part of make test-integration:
// the perf tag keeps it out. Report: docs/quality/acl-perf-800k.md.
//
// Env: NAVA_PERF_SECONDS (20) per scenario, NAVA_PERF_CLIENTS (8),
// NAVA_PERF_OUT (the raw report, default bin/acl-perf-800k-raw.md),
// NAVA_PERF_ONLY (a regular expression over scenario names; empty runs all).

package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/authz/acltest"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/cursor"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/ratelimit"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
	"github.com/Navaladi2019/TicketingSystem/internal/server"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// The targets (PLAN-003 P3-36, D25 escalation).
const (
	perfListTargetMS = 150.0
	perfGetTargetMS  = 50.0
	// The D-3-9 trigger: create p95 > 50 ms revisits record numbering.
	perfCreateTargetMS = 50.0
	perfSmallTenants   = 399
	perfLargeRecords   = 400000
	perfLargeUsers     = 5000
	perfSmallUsers     = 20
	perfGroups         = 50
	perfLargeActors    = 200
	perfPage20Actors   = 20
)

// perfSchema is P3-01's record shape as an entity type: the fixed columns of
// the PLAN-003 cut plus about 1 KB of data.
const perfSchema = `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "additionalProperties": false,
  "required": ["short_description", "state"],
  "properties": {
    "short_description": {"type": "string", "minLength": 1, "maxLength": 160},
    "description": {"type": "string", "maxLength": 4000},
    "priority": {"type": "string", "enum": ["1", "2", "3", "4", "5"]},
    "category": {"type": "string", "maxLength": 40},
    "impact": {"type": "integer"},
    "tags": {"type": "array", "items": {"type": "string"}},
    "site": {"type": "string", "maxLength": 10},
    "state": {"type": "string", "enum": ["new", "in_progress", "on_hold", "closed"], "default": "new"},
    "group": {"type": "string", "format": "uuid"},
    "assignee": {"type": "string", "format": "uuid"},
    "requester": {"type": "string", "format": "uuid"}
  }
}`

func perfTypeBody() string {
	return fmt.Sprintf(`{"key": "ticket", "name": "Ticket", "number_prefix": "TKT", "definition": {"schema": %s,
  "fixed_columns": {"state": "state", "group": "assignment_group_id", "assignee": "assignee_id", "requester": "requester_id"},
  "states": {"initial": "new", "states": ["new", "in_progress", "on_hold", "closed"],
    "transitions": [{"from": "new", "to": "in_progress"}, {"from": "in_progress", "to": "on_hold"},
      {"from": "on_hold", "to": "in_progress"}, {"from": "new", "to": "closed"}, {"from": "in_progress", "to": "closed"},
      {"from": "on_hold", "to": "closed"}]}}}`, perfSchema)
}

// perfACL is the ACL every tenant publishes, the OR-combined shape of P3-01's
// query (d) with the real rule language: the agents group reads the records
// of its own assignment groups, anyone reads what they requested, are
// assigned or were shared, and the admins group reads everything.
func perfACL(agents, admins uuid.UUID) []authz.Rule {
	return []authz.Rule{
		{ID: "desk", Ops: []string{authz.OpRead, authz.OpWrite}, Who: authz.Subject{Groups: []uuid.UUID{agents}},
			Where: []authz.Cond{{Column: "assignment_group_id", Op: authz.CondIn, Caller: authz.CallerMyGroups}}},
		{ID: "mine", Ops: []string{authz.OpRead}, Who: authz.Subject{Requester: true, Assignee: true, Shared: true}},
		{ID: "admins", Ops: []string{authz.OpRead, authz.OpWrite}, Who: authz.Subject{Groups: []uuid.UUID{admins}}},
	}
}

// ---- statement capture -----------------------------------------------------

type perfTraceKey struct{}
type perfStartKey struct{}

// perfStmt is one statement a request ran: its SQL, arguments and duration
// (to the end of the result set, as pgx traces it).
type perfStmt struct {
	SQL  string
	Args []any
	// Start is the offset from the start of the request; Dur runs to the end
	// of the result set.
	Start, Dur time.Duration
}

// perfTrace collects the statements of one request; start is when the
// request was sent.
type perfTrace struct {
	mu    sync.Mutex
	start time.Time
	stmts []perfStmt
}

func newPerfTrace() *perfTrace { return &perfTrace{start: time.Now()} }

type perfStart struct {
	at   time.Time
	sql  string
	args []any
}

// perfTracer records every statement run under a context that carries a
// *perfTrace; the handler passes the request context to every query.
type perfTracer struct{}

func (perfTracer) TraceQueryStart(ctx context.Context, _ *pgx.Conn, d pgx.TraceQueryStartData) context.Context {
	if ctx.Value(perfTraceKey{}) == nil {
		return ctx
	}
	return context.WithValue(ctx, perfStartKey{}, perfStart{at: time.Now(), sql: d.SQL, args: d.Args})
}

func (perfTracer) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, _ pgx.TraceQueryEndData) {
	tr, _ := ctx.Value(perfTraceKey{}).(*perfTrace)
	st, ok := ctx.Value(perfStartKey{}).(perfStart)
	if tr == nil || !ok {
		return
	}
	tr.mu.Lock()
	tr.stmts = append(tr.stmts, perfStmt{SQL: st.sql, Args: st.args, Start: st.at.Sub(tr.start), Dur: time.Since(st.at)})
	tr.mu.Unlock()
}

// TraceBatchStart implements pgx.BatchTracer (TICK-648): a batch is one
// round trip, recorded as one entry whose SQL is "BATCH: " and its
// statements joined by " ; ". Its arguments are not kept, so its statements
// are not EXPLAINed.
func (perfTracer) TraceBatchStart(ctx context.Context, _ *pgx.Conn, d pgx.TraceBatchStartData) context.Context {
	if ctx.Value(perfTraceKey{}) == nil {
		return ctx
	}
	sqls := make([]string, 0, d.Batch.Len())
	for _, q := range d.Batch.QueuedQueries {
		sqls = append(sqls, normSQL(q.SQL))
	}
	return context.WithValue(ctx, perfStartKey{}, perfStart{at: time.Now(), sql: perfBatchPrefix + strings.Join(sqls, " ; ")})
}

// TraceBatchQuery implements pgx.BatchTracer.
func (perfTracer) TraceBatchQuery(context.Context, *pgx.Conn, pgx.TraceBatchQueryData) {}

// TraceBatchEnd implements pgx.BatchTracer.
func (perfTracer) TraceBatchEnd(ctx context.Context, conn *pgx.Conn, _ pgx.TraceBatchEndData) {
	perfTracer{}.TraceQueryEnd(ctx, conn, pgx.TraceQueryEndData{})
}

const perfBatchPrefix = "BATCH: "

type perfAllow struct{}

func (perfAllow) Allow(context.Context, uuid.UUID) (ratelimit.Decision, error) {
	return ratelimit.Decision{Allowed: true}, nil
}

// ---- environment -----------------------------------------------------------

type perfEnv struct {
	h      http.Handler
	pool   *pgxpool.Pool
	super  *pgx.Conn
	tokens etTokens
	large  etTenant
	// tokens per role
	agents, requesters, smallAgents, smallAdmins []string
	agentGroups                                  map[string][]uuid.UUID
}

func perfEnvInt(name string, def int) int {
	if v, err := strconv.Atoi(os.Getenv(name)); err == nil && v > 0 {
		return v
	}
	return def
}

func newPerfEnv(t *testing.T, clients int) *perfEnv {
	t.Helper()
	super, base := etCluster(t)
	pc := base.Config().Copy()
	pc.ConnConfig.Tracer = perfTracer{}
	pc.MaxConns = int32(clients + 2) //nolint:gosec // a small env setting
	pool, err := pgxpool.NewWithConfig(t.Context(), pc)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	// nava-core's allocator pool (cmd/nava-core/allocator.go, TICK-657): a
	// create takes its number on it, traced like the API pool.
	ac := pc.Copy()
	ac.MaxConns = 4
	if err := tenancy.AllocPoolConfig(ac); err != nil {
		t.Fatal(err)
	}
	alloc, err := pgxpool.NewWithConfig(t.Context(), ac)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(alloc.Close)
	large := etSeedTenant(t, super, "large")
	tokens := etTokens{"admin-large": large.admin}
	tel, err := telemetry.New(t.Context(), config.Telemetry{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tel.Shutdown(context.Background()) })
	kr, err := cursor.NewKeyring("k1", cursor.Key{ID: "k1", Secret: bytes.Repeat([]byte("s"), 32)})
	if err != nil {
		t.Fatal(err)
	}
	reader := etAuditReader(t, pool)
	h, err := server.NewHandler(slog.New(slog.NewTextHandler(testLogWriter{t, nil}, &slog.HandlerOptions{Level: slog.LevelWarn})), tel, tokens, stubPaths{},
		server.WithLimiter(perfAllow{}), server.WithPool(pool), server.WithAllocatorPool(alloc), server.WithAuditReader(reader), server.WithCursors(kr))
	if err != nil {
		t.Fatal(err)
	}
	return &perfEnv{h: h, pool: pool, super: super, tokens: tokens, large: large, agentGroups: map[string][]uuid.UUID{}}
}

func (e *perfEnv) exec(t *testing.T, sql string, args ...any) {
	t.Helper()
	if _, err := e.super.Exec(t.Context(), sql, args...); err != nil {
		t.Fatalf("%s: %v", firstLine(sql), err)
	}
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

func (e *perfEnv) serve(ctx context.Context, method, path, token, idem, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequestWithContext(ctx, method, path, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	if idem != "" {
		req.Header.Set("Idempotency-Key", idem)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	e.h.ServeHTTP(rec, req)
	return rec
}

// defineType creates the ticket type through the API and publishes perfACL.
func (e *perfEnv) defineType(t *testing.T, tn uuid.UUID, adminToken string, agents, admins uuid.UUID) {
	t.Helper()
	if rec := e.serve(t.Context(), http.MethodPost, "/api/v1/entity-types", adminToken, "perf-type", perfTypeBody()); rec.Code != http.StatusCreated {
		t.Fatalf("create type for %s = %d: %s", adminToken, rec.Code, rec.Body)
	}
	if err := tenancy.InTenantTx(t.Context(), e.pool, tn, func(tx pgx.Tx) error {
		_, err := acltest.Publish(t.Context(), tx, "ticket", perfACL(agents, admins)...)
		return err
	}); err != nil {
		t.Fatal(err)
	}
}

// seed builds P3-01's shape. Tenant 0 is the large tenant (etSeedTenant's
// "large"); tenants 1..399 are seeded the same way, each with its own type.
func (e *perfEnv) seed(t *testing.T) {
	t.Helper()
	ctx := t.Context()
	e.exec(t, `CREATE TABLE perf_tenant (tn int PRIMARY KEY, tenant_id uuid NOT NULL, nrec int NOT NULL, nusers int NOT NULL,
		agents uuid NOT NULL, admins uuid NOT NULL)`)
	tenants := []etTenant{e.large}
	for i := 1; i <= perfSmallTenants; i++ {
		tn := etSeedTenant(t, e.super, "s"+strconv.Itoa(i))
		tenants = append(tenants, tn)
		e.tokens["admin-s"+strconv.Itoa(i)] = tn.admin
	}
	for i, tn := range tenants {
		nrec, nusers := 1002, perfSmallUsers
		if i <= 202 {
			nrec = 1003
		}
		if i == 0 {
			nrec, nusers = perfLargeRecords, perfLargeUsers
		}
		agents := uuid.NewMD5(uuid.NameSpaceOID, []byte(fmt.Sprintf("agents/%d", i)))
		e.exec(t, "INSERT INTO groups (tenant_id, id, keycloak_group_id, path) VALUES ($1, $2, $3, $4)",
			tn.id, agents, fmt.Sprintf("perf-agents-%d", i), "/t/"+tn.id.String()+"/agents")
		e.exec(t, "INSERT INTO perf_tenant VALUES ($1, $2, $3, $4, $5, $6)", i, tn.id, nrec, nusers, agents, tn.adminGroup)
		admin := "admin-large"
		if i > 0 {
			admin = "admin-s" + strconv.Itoa(i)
		}
		e.defineType(t, tn.id, admin, agents, tn.adminGroup)
	}
	t.Log("perf: 400 tenants, types and ACLs created through the API")
	start := time.Now()
	for _, q := range perfSeedSQL() {
		if _, err := e.super.Exec(ctx, q); err != nil {
			t.Fatalf("seed %s: %v", firstLine(q), err)
		}
	}
	t.Logf("perf: seeded in %s", time.Since(start).Round(time.Second))
	e.actors(t)
}

// perfSeedSQL is bench/records-list/02_seed.sql on the real schema: the same
// RNG seed, distributions and per-tenant sizes. Each tenant's records all
// belong to its one entity type (P3-01's query had no type filter, so every
// list here still runs over the tenant's whole record set). The superuser
// bypasses RLS; session_replication_role = replica skips the audit commit
// check and the foreign keys for the bulk load (every id is valid anyway).
func perfSeedSQL() []string {
	return []string{
		`SET max_parallel_workers_per_gather = 0`,
		`SELECT setseed(0.339)`,
		`SET session_replication_role = replica`,
		`INSERT INTO principals (tenant_id, id, keycloak_user_id, email, enabled, federated_links, synced_at)
		 SELECT t.tenant_id, md5('u' || t.tn || '/' || u)::uuid, 'perf-u' || t.tn || '-' || u, 'u' || u || '@t' || t.tn || '.test', true, '[]', now()
		 FROM perf_tenant t, generate_series(0, t.nusers - 1) u`,
		`INSERT INTO groups (tenant_id, id, keycloak_group_id, path)
		 SELECT t.tenant_id, md5('g' || t.tn || '/' || g)::uuid, 'perf-g' || t.tn || '-' || g, '/t/' || t.tenant_id || '/g' || g
		 FROM perf_tenant t, generate_series(0, 49) g`,
		`INSERT INTO group_members (tenant_id, group_id, principal_id)
		 SELECT t.tenant_id, md5('g' || t.tn || '/' || ((u + k) % 50))::uuid, md5('u' || t.tn || '/' || u)::uuid
		 FROM perf_tenant t, generate_series(0, t.nusers - 1) u, unnest(ARRAY[0, 17, 34]) k`,
		`INSERT INTO records (tenant_id, id, version, entity_definition_id, schema_version, number, state,
		    requester_id, assignee_id, assignment_group_id, created_by, created_at, updated_at, data, causation_depth)
		 SELECT x.tenant_id,
		        md5('r' || x.tn || '/' || x.n)::uuid,
		        1, x.def, 1, 'TKT' || lpad(x.n::text, 7, '0'),
		        CASE WHEN x.r1 < 0.15 THEN (ARRAY['new', 'in_progress', 'on_hold'])[1 + floor(x.r2 * 3)::int] ELSE 'closed' END,
		        md5('u' || x.tn || '/' || floor(x.nusers * x.r3 * x.r3)::int)::uuid,
		        CASE WHEN x.r4 < 0.9 THEN md5('u' || x.tn || '/' || floor(x.nusers * x.r5)::int)::uuid END,
		        md5('g' || x.tn || '/' || floor(50 * x.r6)::int)::uuid,
		        md5('u' || x.tn || '/' || floor(x.nusers * x.r3 * x.r3)::int)::uuid,
		        x.upd - x.r7 * interval '30 days',
		        x.upd,
		        jsonb_build_object(
		            'priority', CASE WHEN x.r8 < 0.05 THEN '1' WHEN x.r8 < 0.20 THEN '2' WHEN x.r8 < 0.70 THEN '3'
		                             WHEN x.r8 < 0.90 THEN '4' ELSE '5' END,
		            'category', 'category-' || floor(x.r9 * 20)::int,
		            'impact', 1 + floor(x.r2 * 3)::int,
		            'short_description', 'Record ' || x.n || ' ' || md5(x.n::text),
		            'description', repeat(md5(x.tn || ':' || x.n), 27),
		            'tags', jsonb_build_array('t' || floor(x.r5 * 50)::int, 't' || floor(x.r6 * 50)::int),
		            -- The rare value (TICK-621, the CTO's D25 re-run condition):
		            -- 200 sites, each 0.5% of rows. It reuses r9 (category's
		            -- draw, never a sort or ACL column), so the RNG stream and
		            -- every other column stay P3-01's.
		            'site', 's' || floor(x.r9 * 200)::int),
		        0
		 FROM (
		     SELECT t.tn, t.tenant_id, t.nusers, d.id AS def, n,
		            random() AS r1, random() AS r2, random() AS r3, random() AS r4, random() AS r5,
		            random() AS r6, random() AS r7, random() AS r8, random() AS r9,
		            timestamptz '2026-10-01 00:00:00+00' - power(random(), 3) * interval '365 days' AS upd
		     FROM perf_tenant t JOIN entity_definitions d ON d.tenant_id = t.tenant_id AND d.key = 'ticket',
		          generate_series(1, t.nrec) n
		     ORDER BY t.tn, n
		 ) x`,
		`INSERT INTO record_counters (tenant_id, entity_definition_id, next)
		 SELECT t.tenant_id, d.id, t.nrec + 1 FROM perf_tenant t JOIN entity_definitions d ON d.tenant_id = t.tenant_id AND d.key = 'ticket'`,
		// 2% of records shared with one random user of the tenant.
		`INSERT INTO record_shares (tenant_id, record_id, principal_kind, user_id)
		 SELECT r.tenant_id, r.id, 'user', md5('u' || t.tn || '/' || floor(t.nusers * random())::int)::uuid
		 FROM records r JOIN perf_tenant t USING (tenant_id)
		 WHERE random() < 0.02
		 ON CONFLICT DO NOTHING`,
		`SET session_replication_role = origin`,
		// The report reads pg_settings on this session: give it the server's values back.
		`RESET max_parallel_workers_per_gather`,
		`VACUUM (ANALYZE) records`,
		`VACUUM (ANALYZE) record_shares`,
		`VACUUM (ANALYZE) principals`,
		`VACUUM (ANALYZE) groups`,
		`VACUUM (ANALYZE) group_members`,
		`CREATE EXTENSION IF NOT EXISTS pg_prewarm`,
		`SELECT pg_prewarm(c.oid) FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
		 WHERE n.nspname = 'public' AND c.relkind IN ('r', 'i')`,
	}
}

// actors registers tokens: P3-01's 200 large-tenant actors as agents (the
// agents group plus their 3 assignment groups), the large tenant's heavy
// requesters (more than 1,000 records each) as plain users, and one agent and
// the admin per small tenant.
func (e *perfEnv) actors(t *testing.T) {
	t.Helper()
	ctx := t.Context()
	type row struct {
		tn     int
		tenant uuid.UUID
		agents uuid.UUID
		nusers int
	}
	var ts []row
	rows, err := e.super.Query(ctx, "SELECT tn, tenant_id, agents, nusers FROM perf_tenant ORDER BY tn")
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.tn, &r.tenant, &r.agents, &r.nusers); err != nil {
			t.Fatal(err)
		}
		ts = append(ts, r)
	}
	rows.Close()
	md5id := func(s string) uuid.UUID {
		var id uuid.UUID
		if err := e.super.QueryRow(ctx, "SELECT md5($1)::uuid", s).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	agent := func(r row, u int) identity.Principal {
		g := []uuid.UUID{r.agents}
		for _, k := range []int{0, 17, 34} {
			g = append(g, md5id(fmt.Sprintf("g%d/%d", r.tn, (u+k)%perfGroups)))
		}
		return identity.Principal{TenantID: r.tenant, PrincipalID: md5id(fmt.Sprintf("u%d/%d", r.tn, u)), Groups: g}
	}
	for n := 1; n <= perfLargeActors; n++ {
		tok := "agent-" + strconv.Itoa(n)
		p := agent(ts[0], (n*24+7)%perfLargeUsers)
		e.tokens[tok], e.agentGroups[tok] = p, p.Groups[1:]
		e.agents = append(e.agents, tok)
	}
	hrows, err := e.super.Query(ctx, `SELECT requester_id FROM records WHERE tenant_id = $1
		GROUP BY requester_id HAVING count(*) > 1000 ORDER BY count(*) DESC, requester_id`, ts[0].tenant)
	if err != nil {
		t.Fatal(err)
	}
	for hrows.Next() {
		var id uuid.UUID
		if err := hrows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		tok := "req-" + strconv.Itoa(len(e.requesters)+1)
		e.tokens[tok] = identity.Principal{TenantID: ts[0].tenant, PrincipalID: id}
		e.requesters = append(e.requesters, tok)
	}
	hrows.Close()
	if len(e.requesters) == 0 {
		t.Fatal("no heavy requester in the large tenant")
	}
	for _, r := range ts[1:] {
		tok := "agent-s" + strconv.Itoa(r.tn)
		p := agent(r, r.tn%perfSmallUsers)
		e.tokens[tok], e.agentGroups[tok] = p, p.Groups[1:]
		e.smallAgents = append(e.smallAgents, tok)
		e.smallAdmins = append(e.smallAdmins, "admin-s"+strconv.Itoa(r.tn))
	}
}

// ---- scenarios -------------------------------------------------------------

// perfReq is one request of a scenario: a token, a path and, for a create,
// the JSON body.
type perfReq struct{ token, path, body string }

type perfScenario struct {
	name   string
	kind   string // list, get or create
	tenant string // large or small
	reqs   []perfReq
	// clients is the scenario's own client count; 0 uses NAVA_PERF_CLIENTS.
	clients int
}

// send runs r the way sc's kind does: a GET for list and get, a POST with a
// fresh Idempotency-Key for create. It returns the response and the status
// that means success.
func (e *perfEnv) send(ctx context.Context, sc perfScenario, r perfReq) (*httptest.ResponseRecorder, int) {
	if sc.kind == "create" {
		return e.serve(ctx, http.MethodPost, r.path, r.token, uuid.NewString(), r.body), http.StatusCreated
	}
	return e.serve(ctx, http.MethodGet, r.path, r.token, "", ""), http.StatusOK
}

func (e *perfEnv) list(ctx context.Context, t *testing.T, token, query string) listPage {
	t.Helper()
	path := "/api/v1/records/ticket"
	if query != "" {
		path += "?" + query
	}
	rec := e.serve(ctx, http.MethodGet, path, token, "", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET %s as %s = %d: %s", path, token, rec.Code, rec.Body)
	}
	var p listPage
	if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
		t.Fatal(err)
	}
	return p
}

func listReqs(tokens []string, query string) []perfReq {
	out := make([]perfReq, 0, len(tokens))
	for _, tok := range tokens {
		path := "/api/v1/records/ticket"
		if query != "" {
			path += "?" + query
		}
		out = append(out, perfReq{tok, path, ""})
	}
	return out
}

// page20 walks each actor's list to keyset page 20 (the cursor after the
// 950th row, as in P3-01) and returns those requests.
func (e *perfEnv) page20(t *testing.T, tokens []string, query string) []perfReq {
	t.Helper()
	var out []perfReq
	for _, tok := range tokens {
		q := query
		for page := 1; page < 20; page++ {
			p := e.list(t.Context(), t, tok, q)
			if p.Next == nil {
				// A shorter list would silently measure an earlier page as page 20.
				t.Fatalf("%s: the list of %s ends at page %d, before page 20", query, tok, page)
			}
			q = joinQuery(query, "cursor="+urlEscape(*p.Next))
		}
		out = append(out, perfReq{tok, "/api/v1/records/ticket?" + q, ""})
	}
	return out
}

func joinQuery(a, b string) string {
	if a == "" {
		return b
	}
	return a + "&" + b
}

func urlEscape(s string) string {
	return strings.NewReplacer("+", "%2B", "/", "%2F", "=", "%3D").Replace(s)
}

// getReqs picks, per actor, records the actor can read: for an agent, one in
// each of its assignment groups; for an admin, any record of the tenant.
func (e *perfEnv) getReqs(t *testing.T, tokens []string, perActor int) []perfReq {
	t.Helper()
	var out []perfReq
	for _, tok := range tokens {
		p := e.tokens[tok]
		groups := e.agentGroups[tok]
		var (
			rows pgx.Rows
			err  error
		)
		if groups != nil {
			rows, err = e.super.Query(t.Context(), `SELECT id FROM records WHERE tenant_id = $1 AND assignment_group_id = ANY($2)
				ORDER BY md5(id::text) LIMIT $3`, p.TenantID, groups, perActor)
		} else {
			rows, err = e.super.Query(t.Context(), `SELECT id FROM records WHERE tenant_id = $1 ORDER BY md5(id::text) LIMIT $2`, p.TenantID, perActor)
		}
		if err != nil {
			t.Fatal(err)
		}
		for rows.Next() {
			var id uuid.UUID
			if err := rows.Scan(&id); err != nil {
				t.Fatal(err)
			}
			out = append(out, perfReq{tok, "/api/v1/records/ticket/" + id.String(), ""})
		}
		rows.Close()
	}
	return out
}

// createReqs builds one create per actor: P3-01's record shape (about 1 KB
// of data), the actor as requester, and for an agent one of its own
// assignment groups (the desk rule's write scope); an admin's record goes to
// group 0 of its tenant.
func (e *perfEnv) createReqs(t *testing.T, tokens []string) []perfReq {
	t.Helper()
	out := make([]perfReq, 0, len(tokens))
	for i, tok := range tokens {
		p := e.tokens[tok]
		group := uuid.Nil
		if g := e.agentGroups[tok]; len(g) > 0 {
			group = g[i%len(g)]
		} else if err := e.super.QueryRow(t.Context(), `SELECT md5('g' || tn || '/0')::uuid FROM perf_tenant WHERE tenant_id = $1`,
			p.TenantID).Scan(&group); err != nil {
			t.Fatal(err)
		}
		data := map[string]any{
			"short_description": fmt.Sprintf("Perf create %d", i),
			"description":       strings.Repeat("0123456789abcdef0123456789abcdef", 27),
			"priority":          "3",
			"category":          "category-1",
			"impact":            2,
			"tags":              []string{"t1", "t2"},
			"site":              "s1",
			"group":             group.String(),
			"requester":         p.PrincipalID.String(),
		}
		b, err := json.Marshal(map[string]any{"data": data})
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, perfReq{tok, "/api/v1/records/ticket", string(b)})
	}
	return out
}

// perfDef is a scenario before its requests are built: build may walk lists
// (page 20) or query the seed, so only the selected scenarios pay for it.
type perfDef struct {
	name, kind, tenant string
	build              func() []perfReq
	// clients overrides NAVA_PERF_CLIENTS for this scenario (0: the default).
	clients int
}

// scenarios builds the scenarios whose name matches NAVA_PERF_ONLY (a regular
// expression; empty selects every one).
func (e *perfEnv) scenarios(t *testing.T) []perfScenario {
	t.Helper()
	only, err := regexp.Compile(os.Getenv("NAVA_PERF_ONLY"))
	if err != nil {
		t.Fatalf("NAVA_PERF_ONLY: %v", err)
	}
	var out []perfScenario
	for _, d := range e.defs(t) {
		if only.MatchString(d.name) {
			out = append(out, perfScenario{d.name, d.kind, d.tenant, d.build(), d.clients})
		}
	}
	if len(out) == 0 {
		t.Fatalf("NAVA_PERF_ONLY=%q selects no scenario", only)
	}
	return out
}

func (e *perfEnv) defs(t *testing.T) []perfDef {
	t.Helper()
	admin := []string{"admin-large"}
	return []perfDef{
		{"agent list by group, default sort, page 1", "list", "large", func() []perfReq { return listReqs(e.agents, "") }, 0},
		{"agent list by group, default sort, page 20", "list", "large", func() []perfReq { return e.page20(t, e.agents[:perfPage20Actors], "") }, 0},
		{"requester list (heavy requesters), page 1", "list", "large", func() []perfReq { return listReqs(e.requesters, "") }, 0},
		{"requester list (heavy requesters), page 20", "list", "large", func() []perfReq { return e.page20(t, e.requesters, "") }, 0},
		{"admin list, default sort, page 1", "list", "large", func() []perfReq { return listReqs(admin, "") }, 0},
		{"admin list, default sort, page 20", "list", "large", func() []perfReq { return e.page20(t, admin, "") }, 0},
		{"agent list, filter[state]=new", "list", "large", func() []perfReq { return listReqs(e.agents, "filter%5Bstate%5D=new") }, 0},
		{"agent list, filter[priority]=1 (data)", "list", "large", func() []perfReq { return listReqs(e.agents, "filter%5Bpriority%5D=1") }, 0},
		{"agent list, filter[site]=s7 (rare data, < 1% of visible rows)", "list", "large", func() []perfReq { return listReqs(e.agents, "filter%5Bsite%5D=s7") }, 0},
		{"agent list, sort=-updated_at", "list", "large", func() []perfReq { return listReqs(e.agents, "sort=-updated_at") }, 0},
		{"agent list, sort=assignee", "list", "large", func() []perfReq { return listReqs(e.agents, "sort=assignee") }, 0},
		{"admin list, filter[state][in]=new,in_progress, sort=-updated_at", "list", "large", func() []perfReq {
			return listReqs(admin, "filter%5Bstate%5D%5Bin%5D=new,in_progress&sort=-updated_at")
		}, 0},
		{"small tenant: agent list, page 1", "list", "small", func() []perfReq { return listReqs(e.smallAgents, "") }, 0},
		{"small tenant: admin list, page 1", "list", "small", func() []perfReq { return listReqs(e.smallAdmins, "") }, 0},
		{"single GET, agent", "get", "large", func() []perfReq { return e.getReqs(t, e.agents, 5) }, 0},
		{"single GET, admin", "get", "large", func() []perfReq { return e.getReqs(t, admin, 1000) }, 0},
		{"single GET, small tenant agent", "get", "small", func() []perfReq { return e.getReqs(t, e.smallAgents, 2) }, 0},
		// Creates run last, so the rows they add never reach a read scenario.
		// One client: no other create queues on the tenant's record_counters
		// row, so this is the create path alone (D-3-9 states no concurrency).
		{"create, agent (1 client)", "create", "large", func() []perfReq { return e.createReqs(t, e.agents) }, 1},
		{"create, agent", "create", "large", func() []perfReq { return e.createReqs(t, e.agents) }, 0},
		{"create, admin", "create", "large", func() []perfReq { return e.createReqs(t, admin) }, 0},
		{"create, small tenant agent", "create", "small", func() []perfReq { return e.createReqs(t, e.smallAgents) }, 0},
	}
}

// ---- measurement -----------------------------------------------------------

type perfResult struct {
	sc       perfScenario
	lat      []time.Duration
	stmtDur  map[string][]time.Duration // by normalized SQL
	stmtSQL  []perfStmt                 // one representative request, in order
	explains []string
	errs     int64
	holds    []perfHold // creates only, one per request
}

// perfHold is one create's record_counters window (D-3-9). wait is the
// upsert itself: the queue for the row lock plus its execution. hold runs
// from the end of the upsert, when the row is locked, to the end of the
// COMMIT that releases it; stmts is the time n statements spent in that
// window, and the rest of hold is the handler's own time between them.
type perfHold struct {
	wait, hold, stmts time.Duration
	n                 int
}

// counterHold finds the record_counters upsert and the COMMIT that ends its
// transaction in one create's statements. Since TICK-657 the upsert runs
// alone in a transaction of its own on the allocator pool, and the COMMIT
// right after it ends the window (D-3-9 as amended 2026-11-10). A COMMIT
// batched after other statements (" ; commit") still ends it, as in the
// TICK-648 layout. n counts round trips.
func counterHold(ss []perfStmt) (perfHold, bool) {
	up := slices.IndexFunc(ss, func(s perfStmt) bool { return strings.Contains(s.SQL, "INSERT INTO record_counters") })
	if up < 0 {
		return perfHold{}, false
	}
	locked := ss[up].Start + ss[up].Dur
	h := perfHold{wait: ss[up].Dur}
	for _, s := range ss[up+1:] {
		h.stmts += s.Dur
		h.n++
		if sql := strings.ToLower(strings.TrimSpace(s.SQL)); sql == "commit" || strings.HasSuffix(sql, " ; commit") {
			h.hold = s.Start + s.Dur - locked
			return h, true
		}
	}
	return perfHold{}, false
}

var perfWS = regexp.MustCompile(`\s+`)

func normSQL(s string) string { return perfWS.ReplaceAllString(strings.TrimSpace(s), " ") }

// run drives sc with clients workers for d after a warm-up of every request
// once, and returns the per-request latencies and per-statement durations.
func (e *perfEnv) run(t *testing.T, sc perfScenario, clients int, d time.Duration) *perfResult {
	t.Helper()
	res := &perfResult{sc: sc, stmtDur: map[string][]time.Duration{}}
	for i, r := range sc.reqs {
		if i >= 50 {
			break
		}
		if rec, want := e.send(t.Context(), sc, r); rec.Code != want {
			t.Fatalf("%s: warm-up %s as %s = %d: %s", sc.name, r.path, r.token, rec.Code, rec.Body)
		}
	}
	var (
		mu       sync.Mutex
		next     atomic.Int64
		wg       sync.WaitGroup
		deadline = time.Now().Add(d)
	)
	for w := 0; w < clients; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var (
				lat   []time.Duration
				holds []perfHold
			)
			stmts := map[string][]time.Duration{}
			for time.Now().Before(deadline) {
				r := sc.reqs[int(next.Add(1)-1)%len(sc.reqs)]
				tr := newPerfTrace()
				ctx := context.WithValue(t.Context(), perfTraceKey{}, tr)
				t0 := time.Now()
				rec, want := e.send(ctx, sc, r)
				el := time.Since(t0)
				if rec.Code != want {
					atomic.AddInt64(&res.errs, 1)
					continue
				}
				lat = append(lat, el)
				if h, ok := counterHold(tr.stmts); ok && sc.kind == "create" {
					holds = append(holds, h)
				}
				for _, s := range tr.stmts {
					k := normSQL(s.SQL)
					stmts[k] = append(stmts[k], s.Dur)
				}
			}
			mu.Lock()
			res.lat = append(res.lat, lat...)
			res.holds = append(res.holds, holds...)
			for k, v := range stmts {
				res.stmtDur[k] = append(res.stmtDur[k], v...)
			}
			mu.Unlock()
		}()
	}
	wg.Wait()
	if res.errs > 0 {
		t.Errorf("%s: %d requests failed", sc.name, res.errs)
	}
	e.capture(t, res)
	return res
}

// capture runs the scenario's first request once more with a trace, keeps
// its statements, and EXPLAINs each SELECT exactly as the handler sent it, as
// nava_app in the actor's tenant transaction. Writes are never EXPLAINed:
// EXPLAIN ANALYZE would run them again.
func (e *perfEnv) capture(t *testing.T, res *perfResult) {
	t.Helper()
	r := res.sc.reqs[0]
	tr := newPerfTrace()
	if rec, want := e.send(context.WithValue(t.Context(), perfTraceKey{}, tr), res.sc, r); rec.Code != want {
		t.Fatalf("%s: capture = %d: %s", res.sc.name, rec.Code, rec.Body)
	}
	res.stmtSQL = tr.stmts
	tenant := e.tokens[r.token].TenantID
	for _, s := range tr.stmts {
		if !isSelect(s.SQL) || strings.Contains(s.SQL, "set_config(") {
			res.explains = append(res.explains, "")
			continue
		}
		var lines []string
		err := tenancy.InTenantTx(t.Context(), e.pool, tenant, func(tx pgx.Tx) error {
			rows, err := tx.Query(t.Context(), "EXPLAIN (ANALYZE, BUFFERS, SETTINGS) "+s.SQL, s.Args...)
			if err != nil {
				return err
			}
			defer rows.Close()
			for rows.Next() {
				var l string
				if err := rows.Scan(&l); err != nil {
					return err
				}
				lines = append(lines, l)
			}
			return rows.Err()
		})
		if err != nil {
			t.Fatalf("%s: explain %s: %v", res.sc.name, firstLine(s.SQL), err)
		}
		res.explains = append(res.explains, strings.Join(lines, "\n"))
	}
}

func isSelect(sql string) bool {
	s := strings.ToUpper(strings.TrimSpace(sql))
	return strings.HasPrefix(s, "SELECT") || strings.HasPrefix(s, "WITH")
}

// pct is the nearest-rank percentile in ms, as P3-01's pgbench logs used.
func pct(d []time.Duration, p float64) float64 {
	if len(d) == 0 {
		return 0
	}
	s := slices.Clone(d)
	slices.Sort(s)
	i := int(float64(len(s))*p+0.999999) - 1
	i = max(0, min(i, len(s)-1))
	return float64(s[i].Microseconds()) / 1000
}

func (r *perfResult) target() float64 {
	switch r.sc.kind {
	case "get":
		return perfGetTargetMS
	case "create":
		return perfCreateTargetMS
	}
	return perfListTargetMS
}

// ---- the test --------------------------------------------------------------

func TestRecordsACLPerf800k(t *testing.T) {
	clients := perfEnvInt("NAVA_PERF_CLIENTS", 8)
	secs := perfEnvInt("NAVA_PERF_SECONDS", 20)
	out := os.Getenv("NAVA_PERF_OUT")
	if out == "" {
		out = filepath.Join("..", "..", "bin", "acl-perf-800k-raw.md")
	}
	e := newPerfEnv(t, clients)
	e.seed(t)
	floor := e.transportFloor(t, clients)
	t.Logf("FLOOR SELECT 1 round trip, %d clients: p50=%.2f p95=%.2f p99=%.2f ms", clients, pct(floor, .50), pct(floor, .95), pct(floor, .99))
	var results []*perfResult
	for _, sc := range e.scenarios(t) {
		c := clients
		if sc.clients > 0 {
			c = sc.clients
		}
		r := e.run(t, sc, c, time.Duration(secs)*time.Second)
		t.Logf("RESULT %-62s n=%6d p50=%8.2f p95=%8.2f p99=%8.2f ms", sc.name, len(r.lat), pct(r.lat, .50), pct(r.lat, .95), pct(r.lat, .99))
		results = append(results, r)
	}
	report := e.report(t, results, floor, clients, secs)
	if err := os.MkdirAll(filepath.Dir(out), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(out, []byte(report), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Logf("wrote %s", out)
	for _, r := range results {
		if p95 := pct(r.lat, .95); p95 > r.target() {
			t.Errorf("MISS %s: p95 %.2f ms > %.0f ms", r.sc.name, p95, r.target())
		}
	}
}

// transportFloor is the cost of one round trip on the handler's pool: clients
// workers each run SELECT 1 200 times. Every statement of a request pays it,
// so a request's floor is about its statement count times this p50; the
// per-statement tables are read against it.
func (e *perfEnv) transportFloor(t *testing.T, clients int) []time.Duration {
	t.Helper()
	var (
		mu  sync.Mutex
		wg  sync.WaitGroup
		out []time.Duration
	)
	for w := 0; w < clients; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var lat []time.Duration
			for i := 0; i < 200; i++ {
				t0 := time.Now()
				var one int
				if err := e.pool.QueryRow(t.Context(), "SELECT 1").Scan(&one); err != nil {
					t.Error(err)
					return
				}
				lat = append(lat, time.Since(t0))
			}
			mu.Lock()
			out = append(out, lat...)
			mu.Unlock()
		}()
	}
	wg.Wait()
	return out
}

func (e *perfEnv) report(t *testing.T, results []*perfResult, floor []time.Duration, clients, secs int) string {
	t.Helper()
	var b strings.Builder
	w := func(f string, a ...any) { fmt.Fprintf(&b, f, a...) }
	w("# P3-36 raw report (TICK-621)\n\n")
	w("Generated %s by TestRecordsACLPerf800k (internal/server/records_perf_integration_test.go): %d clients, %d s per scenario, through the real handlers.\n\n",
		time.Now().UTC().Format(time.RFC3339), clients, secs)
	w("## Server\n\n```\n%s\n```\n\n", e.superText(t, `SELECT version() UNION ALL
		SELECT name || ' = ' || setting || coalesce(' ' || unit, '') FROM pg_settings
		WHERE name IN ('shared_buffers','effective_cache_size','work_mem','random_page_cost','max_parallel_workers_per_gather','jit','fsync','synchronous_commit','track_io_timing')`))
	w("nava_app role settings: `%s`\n\n", e.superText(t, `SELECT coalesce(string_agg(c, ', '), '') FROM pg_db_role_setting s
		JOIN pg_roles r ON r.oid = s.setrole, unnest(s.setconfig) c WHERE r.rolname = 'nava_app'`))
	w("## Data\n\n```\n%s\n```\n\n", e.superText(t, `SELECT 'records ' || count(*) || ', open ' || count(*) FILTER (WHERE state <> 'closed')
		|| ', priority 1 ' || count(*) FILTER (WHERE data ->> 'priority' = '1') || ', avg data bytes ' || round(avg(pg_column_size(data))) FROM records
		UNION ALL SELECT 'tenants ' || count(*) || ', largest ' || max(nrec) || ', smallest ' || min(nrec) FROM perf_tenant
		UNION ALL SELECT 'site s7, large tenant ' || count(*) FILTER (WHERE r.data ->> 'site' = 's7') || ' of ' || count(*)
		    FROM records r JOIN perf_tenant t ON t.tenant_id = r.tenant_id AND t.tn = 0
		UNION ALL SELECT 'record_shares ' || count(*) FROM record_shares
		UNION ALL SELECT 'heap ' || pg_size_pretty(pg_relation_size('records')) || ', indexes ' || pg_size_pretty(pg_indexes_size('records'))`))
	w("## Transport floor\n\n`SELECT 1` on the handler's pool, %d clients x 200: p50 %.2f / p95 %.2f / p99 %.2f ms. Each statement in the tables below pays about this much before the database does any work.\n\n",
		clients, pct(floor, .50), pct(floor, .95), pct(floor, .99))
	w("## Results (ms, per request through the handler)\n\n| Scenario | Tenant | n | p50 | p95 | p99 | Target | Verdict |\n| --- | --- | --- | --- | --- | --- | --- | --- |\n")
	for _, r := range results {
		p95 := pct(r.lat, .95)
		v := "meets"
		if p95 > r.target() {
			v = "**MISS**"
		}
		w("| %s | %s | %d | %.2f | %.2f | %.2f | %.0f | %s |\n", r.sc.name, r.sc.tenant, len(r.lat), pct(r.lat, .5), p95, pct(r.lat, .99), r.target(), v)
	}
	e.reportHolds(w, results, floor)
	w("\n## Per statement (ms, as pgx traced it, all requests of the scenario)\n\n")
	for _, r := range results {
		w("### %s\n\n| # | Statement | n | p50 | p95 | p99 |\n| --- | --- | --- | --- | --- | --- |\n", r.sc.name)
		for i, s := range r.stmtSQL {
			k := normSQL(s.SQL)
			lbl := k
			if len(lbl) > 90 {
				lbl = lbl[:90] + "…"
			}
			d := r.stmtDur[k]
			w("| %d | `%s` | %d | %.2f | %.2f | %.2f |\n", i+1, strings.ReplaceAll(lbl, "|", "\\|"), len(d), pct(d, .5), pct(d, .95), pct(d, .99))
		}
		w("\n")
	}
	w("## Captured SQL and EXPLAIN (ANALYZE, BUFFERS)\n\nOne request per scenario (its first actor), each statement exactly as the handler sent it, with its arguments, then its plan run once more as nava_app in the same tenant. A create's writes are shown without a plan: EXPLAIN ANALYZE would run them again.\n\n")
	for _, r := range results {
		method := http.MethodGet
		if r.sc.kind == "create" {
			method = http.MethodPost
		}
		w("### %s\n\nRequest: `%s %s` as `%s`\n\n", r.sc.name, method, r.sc.reqs[0].path, r.sc.reqs[0].token)
		for i, s := range r.stmtSQL {
			args, _ := json.Marshal(s.Args)
			switch {
			case r.explains[i] != "":
				w("Statement %d:\n\n```sql\n%s\n```\n\nArguments: `%s`\n\n```\n%s\n```\n\n", i+1, strings.TrimSpace(s.SQL), args, r.explains[i])
			case r.sc.kind == "create" && !isSelect(s.SQL):
				w("Statement %d (write, not EXPLAINed):\n\n```sql\n%s\n```\n\n", i+1, strings.TrimSpace(s.SQL))
			}
		}
	}
	return b.String()
}

// reportHolds prints, per create scenario, how long each create holds its
// tenant's record_counters row and what that time is made of. Transport is
// estimated as the statements in the window times the transport floor's p50;
// database is the rest of their statement time; handler is the time between
// statements.
func (e *perfEnv) reportHolds(w func(string, ...any), results []*perfResult, floor []time.Duration) {
	floorP50 := time.Duration(pct(floor, .50) * float64(time.Millisecond))
	w("\n## Counter hold (creates, D-3-9)\n\nFrom the end of the `record_counters` upsert (the row is locked) to the end of the COMMIT that releases it, per create. Transport is estimated as statements in the window x the transport floor p50 (%.2f ms); database is the rest of their statement time; handler is the time between statements. Wait is the upsert itself: the queue for the row plus its execution.\n\n", pct(floor, .50))
	w("| Scenario | n | Wait p50 / p95 | Hold p50 / p95 | Statements in hold | Transport (est.) p50 / p95 | Database p50 / p95 | Handler p50 / p95 |\n| --- | --- | --- | --- | --- | --- | --- | --- |\n")
	for _, r := range results {
		if r.sc.kind != "create" || len(r.holds) == 0 {
			continue
		}
		var wait, hold, tr, db, app []time.Duration
		var stmts []int
		for _, h := range r.holds {
			t := time.Duration(h.n) * floorP50
			wait, hold, tr = append(wait, h.wait), append(hold, h.hold), append(tr, t)
			db = append(db, max(0, h.stmts-t))
			app = append(app, max(0, h.hold-h.stmts))
			stmts = append(stmts, h.n)
		}
		slices.Sort(stmts)
		w("| %s | %d | %.2f / %.2f | %.2f / %.2f | %d | %.2f / %.2f | %.2f / %.2f | %.2f / %.2f |\n", r.sc.name, len(r.holds),
			pct(wait, .5), pct(wait, .95), pct(hold, .5), pct(hold, .95), stmts[len(stmts)/2],
			pct(tr, .5), pct(tr, .95), pct(db, .5), pct(db, .95), pct(app, .5), pct(app, .95))
	}
}

func (e *perfEnv) superText(t *testing.T, sql string) string {
	t.Helper()
	rows, err := e.super.Query(t.Context(), sql)
	if err != nil {
		t.Fatalf("%s: %v", firstLine(sql), err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			t.Fatal(err)
		}
		out = append(out, s)
	}
	return strings.Join(out, "\n")
}

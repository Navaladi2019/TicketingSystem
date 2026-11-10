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
	"os/exec"
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
	dto "github.com/prometheus/client_model/go"

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
	// perfCommitGateMS is the storage-window gate (CTO, 2026-11-10 13:33
	// UTC): a create row is scored only when its own commit p50 is at or
	// under this, so the host disk's slow band (17 to 26 ms on the WSL2
	// reference host) is not scored as the code's cost.
	perfCommitGateMS = 10.0
	// perfCommitAttempts is how often a gated create row is run before the
	// host is reported invalid for creates.
	perfCommitAttempts = 3
	// perfHostGateMS is the host-speed gate (CTO, 2026-11-10 15:58 UTC): a
	// 1-client `BEGIN; COMMIT` probe run just before each create attempt
	// must have p50 at or under this. It catches a host whose every round
	// trip is slow, which the commit gate alone misses on the 1-client row.
	perfHostGateMS = 2.0
	// perfHostProbeN is the probe's round trips.
	perfHostProbeN = 200
	// perfFloorGateMS is the transport floor gate: a busy host gives no
	// numbers at all.
	perfFloorGateMS  = 3.0
	perfListTargetMS = 150.0
	perfGetTargetMS  = 50.0
	// perfShortTimeout is the bound check's data-filter statement timeout:
	// far below what the s7 count and page take even run alone (tens of ms
	// at 800k), so every request must hit it. 100 ms did not fire on a warm,
	// single request (2026-11-10).
	perfShortTimeout = 2 * time.Millisecond
	// perfDataFilterTimeoutLabel names the server default (store.DataFilterTimeout).
	perfDataFilterTimeoutLabel = "2s (store.DataFilterTimeout)"
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
	h http.Handler
	// hShort is the same handler with a perfShortTimeout data-filter
	// statement timeout (TICK-647), for the bound check.
	hShort http.Handler
	pool   *pgxpool.Pool
	super  *pgx.Conn
	tokens etTokens
	large  etTenant
	// tel is the handler's telemetry: the report reads the allocator's
	// nava_numbering_alloc_* series from its registry (TICK-657).
	tel *telemetry.Telemetry
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
	opts := []server.Option{server.WithLimiter(perfAllow{}), server.WithPool(pool), server.WithAllocatorPool(alloc), server.WithAuditReader(reader), server.WithCursors(kr)}
	log := slog.New(slog.NewTextHandler(testLogWriter{t, nil}, &slog.HandlerOptions{Level: slog.LevelWarn}))
	h, err := server.NewHandler(log, tel, tokens, stubPaths{}, opts...)
	if err != nil {
		t.Fatal(err)
	}
	hShort, err := server.NewHandler(log, tel, tokens, stubPaths{}, append(opts, server.WithDataFilterTimeout(perfShortTimeout))...)
	if err != nil {
		t.Fatal(err)
	}
	return &perfEnv{h: h, hShort: hShort, pool: pool, super: super, tokens: tokens, large: large, tel: tel, agentGroups: map[string][]uuid.UUID{}}
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
	return serveOn(ctx, e.h, method, path, token, idem, body)
}

func serveOn(ctx context.Context, h http.Handler, method, path, token, idem, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequestWithContext(ctx, method, path, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	if idem != "" {
		req.Header.Set("Idempotency-Key", idem)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
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
	// before runs once before the scenario (e.g. switching the type to
	// gapless numbering); nil does nothing.
	before func()
	// unscored reports the scenario without holding it to a target, with
	// this reason (the gapless create: that cost is the type's choice).
	unscored string
}

// send runs r the way sc's kind does: a GET for list and get, a POST with a
// fresh Idempotency-Key for create. It returns the response and the status
// that means success.
func (e *perfEnv) send(ctx context.Context, sc perfScenario, r perfReq) (*httptest.ResponseRecorder, bool) {
	switch sc.kind {
	case "create":
		rec := e.serve(ctx, http.MethodPost, r.path, r.token, uuid.NewString(), r.body)
		return rec, rec.Code == http.StatusCreated
	case "timeout":
		// The bound check: the short timeout must answer filter_too_expensive.
		rec := serveOn(ctx, e.hShort, http.MethodGet, r.path, r.token, "", "")
		return rec, isFilterTooExpensive(rec)
	case "data":
		// A data-filter list either answers or hits its statement timeout.
		rec := e.serve(ctx, http.MethodGet, r.path, r.token, "", "")
		return rec, rec.Code == http.StatusOK || isFilterTooExpensive(rec)
	}
	rec := e.serve(ctx, http.MethodGet, r.path, r.token, "", "")
	return rec, rec.Code == http.StatusOK
}

// isFilterTooExpensive reports whether rec is TICK-647's data-filter timeout:
// 422 with the filter_too_expensive problem code.
func isFilterTooExpensive(rec *httptest.ResponseRecorder) bool {
	return rec.Code == http.StatusUnprocessableEntity && strings.Contains(rec.Body.String(), `"filter_too_expensive"`)
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

// perfExtra is a scenario's optional setup and scoring, by scenario name.
type perfExtra struct {
	before   func()
	unscored string
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
			out = append(out, perfScenario{d.name, d.kind, d.tenant, d.build(), d.clients, e.extras(t)[d.name].before, e.extras(t)[d.name].unscored})
		}
	}
	if len(out) == 0 {
		t.Fatalf("NAVA_PERF_ONLY=%q selects no scenario", only)
	}
	return out
}

// extras are the scenarios with a setup step or no target. The gapless
// create switches the large tenant's ticket type to gapless numbering (D-3-9,
// per type) and runs last; that cost is the type's choice, so it is reported
// and not held to 50 ms.
func (e *perfEnv) extras(t *testing.T) map[string]perfExtra {
	t.Helper()
	return map[string]perfExtra{
		"create, agent, gapless type": {
			before: func() {
				// Through the API (a mode-only publish), so the change is audited.
				rec := e.serve(t.Context(), http.MethodPost, "/api/v1/entity-types/ticket/versions", "admin-large", uuid.NewString(), `{"numbering_mode": "gapless"}`)
				if rec.Code != http.StatusCreated {
					t.Fatalf("set ticket numbering_mode gapless = %d: %s", rec.Code, rec.Body)
				}
			},
			unscored: "gapless type: reported, not scored (D-3-9)",
		},
	}
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
		{"agent list, filter[priority]=1 (data)", "data", "large", func() []perfReq { return listReqs(e.agents, "filter%5Bpriority%5D=1") }, 0},
		{"agent list, filter[site]=s7 (rare data, < 1% of visible rows)", "data", "large", func() []perfReq { return listReqs(e.agents, "filter%5Bsite%5D=s7") }, 0},
		// TICK-647's bound: the same list under a short statement timeout must
		// answer 422 filter_too_expensive about that fast, never hang.
		{"agent list, filter[site]=s7, short data-filter timeout (bound check)", "timeout", "large", func() []perfReq { return listReqs(e.agents, "filter%5Bsite%5D=s7") }, 0},
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
		// Last: it switches the large tenant's ticket type to gapless
		// numbering (D-3-9, per type), reported but not held to 50 ms.
		{"create, agent, gapless type", "create", "large", func() []perfReq { return e.createReqs(t, e.agents) }, 0},
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
	errCodes map[string]int64       // failed requests by "status problem-code"
	alloc    map[string]*dto.Metric // nava_numbering_alloc_* during the scenario
	timeouts int64                  // 422 filter_too_expensive answers (data and timeout kinds)
	holds    []perfHold             // creates only, one per request
	// commits are each create's commit round trip: the last statement of
	// the request that ends its transaction ("commit" or a batch ending in
	// " ; commit"). Their p50 is the storage-window gate.
	commits []time.Duration
	attempt int // 1-based attempt of a gated create row
	// probe is the host-speed probe's p50 in ms, taken just before this
	// attempt (creates only; 0 when not probed).
	probe float64
	// setup is how the attempt was prepared and what ran during its probe
	// (creates only).
	setup perfSetup
}

// perfSetup records an attempt's method steps (Technical Lead, 2026-11-10
// 18:2x UTC): how long it waited for the bench database to quiesce and for
// the host to be free of other bench and integration databases, and what
// pg_stat_activity and docker stats showed during the host-speed probe.
type perfSetup struct {
	quiesceWait, isolationWait time.Duration
	activity                   []string // pg_stat_activity, mid-probe
	dockerStats                []string // docker stats, during the probe
}

// lastCommit is the duration of the request's last round trip that commits
// a transaction: the create's own commit (the allocator's AllocTx commit,
// asynchronous, comes earlier).
func lastCommit(ss []perfStmt) (time.Duration, bool) {
	for i := len(ss) - 1; i >= 0; i-- {
		if sql := strings.ToLower(strings.TrimSpace(ss[i].SQL)); sql == "commit" || strings.HasSuffix(sql, " ; commit") {
			return ss[i].Dur, true
		}
	}
	return 0, false
}

// label names a result in the report tables: a scored create row carries its
// attempt.
func (r *perfResult) label() string {
	if r.attempt > 0 && r.sc.kind == "create" && r.sc.unscored == "" {
		return fmt.Sprintf("%s, attempt %d", r.sc.name, r.attempt)
	}
	return r.sc.name
}

// commitP50 is the row's storage window, in ms.
func (r *perfResult) commitP50() float64 { return pct(r.commits, .5) }

// gated reports a create row attempt that failed a gate: the host-speed
// probe or the storage window. The transport floor gate stops the whole run.
func (r *perfResult) gated() bool {
	return r.sc.kind == "create" && r.sc.unscored == "" && r.gateReason() != ""
}

// gateReason names the gates the attempt failed, or "" when it is valid.
func (r *perfResult) gateReason() string {
	var why []string
	if r.probe > perfHostGateMS {
		why = append(why, fmt.Sprintf("host probe %.2f > %.1f ms", r.probe, perfHostGateMS))
	}
	if r.commitP50() > perfCommitGateMS {
		why = append(why, fmt.Sprintf("commit p50 %.2f > %.0f ms", r.commitP50(), perfCommitGateMS))
	}
	return strings.Join(why, ", ")
}

// retryLater counts the attempt's 503 retry_later answers: in a valid
// attempt each one is a miss.
func (r *perfResult) retryLater() int64 { return r.errCodes["503 retry_later"] }

// hostProbe is the host-speed gate's measurement: one client sends
// `BEGIN; COMMIT` (one round trip, simple protocol) perfHostProbeN times on
// the handler's pool, untraced. It returns the p50 in ms.
//
// Alongside it, docker stats runs (one sample, taken while the probe runs)
// and pg_stat_activity is read once at the probe's midpoint on the
// superuser connection, so the report shows what else was working.
func (e *perfEnv) hostProbe(t *testing.T, st *perfSetup) float64 {
	t.Helper()
	conn, err := e.pool.Acquire(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Release()
	stats := make(chan []string, 1)
	go func() {
		stats <- dockerLines(t.Context(), "stats", "--no-stream", "--format", "{{.Name}} cpu={{.CPUPerc}} mem={{.MemPerc}}")
	}()
	d := make([]time.Duration, 0, perfHostProbeN)
	for i := range perfHostProbeN {
		if i == perfHostProbeN/2 {
			st.activity = e.activity(t)
		}
		t0 := time.Now()
		if _, err := conn.Exec(t.Context(), "BEGIN; COMMIT"); err != nil {
			t.Fatal(err)
		}
		d = append(d, time.Since(t0))
	}
	st.dockerStats = <-stats
	return pct(d, .5)
}

// activity is every other backend of the cluster as pg_stat_activity shows
// it: backend type, database, state, how long its query has run, and the
// query's start.
func (e *perfEnv) activity(t *testing.T) []string {
	t.Helper()
	out := e.superText(t, `SELECT coalesce(backend_type, '') || ' db=' || coalesce(datname, '-') || ' state=' || coalesce(state, '-')
		|| ' running=' || coalesce(round(extract(epoch FROM clock_timestamp() - query_start) * 1000)::text || 'ms', '-')
		|| ' query=' || left(regexp_replace(coalesce(query, ''), '\s+', ' ', 'g'), 60)
		FROM pg_stat_activity WHERE pid <> pg_backend_pid() AND (state IS DISTINCT FROM 'idle' OR backend_type <> 'client backend')
		ORDER BY backend_type, pid`)
	if out == "" {
		return nil
	}
	return strings.Split(out, "\n")
}

// dockerLines runs docker with args and returns its output lines; an error
// is returned as the only line, so the report shows it.
func dockerLines(ctx context.Context, args ...string) []string {
	b, err := exec.CommandContext(ctx, "docker", args...).CombinedOutput()
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	if err != nil {
		return append(lines, "docker: "+err.Error())
	}
	return lines
}

// perfWaitLimit bounds each wait for the database to quiesce or the host to
// be free; past it the run stops rather than measure a busy host.
const perfWaitLimit = 30 * time.Minute

// quiesce waits until no autovacuum worker and no VACUUM or ANALYZE runs in
// the bench database, and returns how long that took.
func (e *perfEnv) quiesce(t *testing.T) time.Duration {
	t.Helper()
	start := time.Now()
	for {
		var n int
		if err := e.super.QueryRow(t.Context(), `SELECT count(*) FROM pg_stat_activity
			WHERE datname = current_database() AND pid <> pg_backend_pid()
			  AND (backend_type = 'autovacuum worker' OR query ~* '^\s*(vacuum|analyze|autovacuum)')
			  AND state IS DISTINCT FROM 'idle'`).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n == 0 {
			return time.Since(start)
		}
		if time.Since(start) > perfWaitLimit {
			t.Fatalf("bench database not quiet after %v: %d autovacuum or analyze backends", perfWaitLimit, n)
		}
		time.Sleep(time.Second)
	}
}

// isolate waits until no other bench or integration database container
// (nava-it-pg-*, nava-aclperf-*) runs on the host; the run's own container
// is NAVA_PERF_PG_CONTAINER (set by bench/acl-perf/run.sh). It returns how
// long that took.
func isolate(t *testing.T) time.Duration {
	t.Helper()
	own := os.Getenv("NAVA_PERF_PG_CONTAINER")
	start := time.Now()
	for {
		var others []string
		for _, n := range dockerLines(t.Context(), "ps", "--format", "{{.Names}}") {
			if n != own && (strings.HasPrefix(n, "nava-it-pg-") || strings.HasPrefix(n, "nava-aclperf-") || strings.HasPrefix(n, "docker: ")) {
				others = append(others, n)
			}
		}
		if len(others) == 0 {
			return time.Since(start)
		}
		if time.Since(start) > perfWaitLimit {
			t.Fatalf("host not isolated after %v: %v still running", perfWaitLimit, others)
		}
		t.Logf("waiting for %v to stop before the next attempt", others)
		time.Sleep(15 * time.Second)
	}
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
// transaction in one create's statements. Since TICK-657 (ADR-005 L2s) the
// upsert and its COMMIT are one AllocTx trip: the hold is inside it, and
// wait is the whole trip. Otherwise (TICK-648, gapless L1) a COMMIT batched
// after other statements (" ; commit") ends the window. n counts round
// trips.
func counterHold(ss []perfStmt) (perfHold, bool) {
	up := slices.IndexFunc(ss, func(s perfStmt) bool { return strings.Contains(s.SQL, "INSERT INTO record_counters") })
	if up < 0 {
		return perfHold{}, false
	}
	h := perfHold{wait: ss[up].Dur}
	// ADR-005 L2s (tenancy.AllocTx): the upsert and its COMMIT are one
	// pipelined trip, so the lock is held inside it, already counted in
	// wait; nothing after the trip holds it.
	if strings.HasSuffix(strings.ToLower(strings.TrimSpace(ss[up].SQL)), " ; commit") {
		return h, true
	}
	locked := ss[up].Start + ss[up].Dur
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
	res := &perfResult{sc: sc, stmtDur: map[string][]time.Duration{}, errCodes: map[string]int64{}}
	allocBefore := e.allocSnapshot(t)
	for i, r := range sc.reqs {
		if i >= 50 {
			break
		}
		if rec, ok := e.send(t.Context(), sc, r); !ok {
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
				lat     []time.Duration
				holds   []perfHold
				commits []time.Duration
			)
			stmts := map[string][]time.Duration{}
			for time.Now().Before(deadline) {
				r := sc.reqs[int(next.Add(1)-1)%len(sc.reqs)]
				tr := newPerfTrace()
				ctx := context.WithValue(t.Context(), perfTraceKey{}, tr)
				t0 := time.Now()
				rec, ok := e.send(ctx, sc, r)
				el := time.Since(t0)
				if !ok {
					atomic.AddInt64(&res.errs, 1)
					var pb struct {
						Code string `json:"code"`
					}
					_ = json.Unmarshal(rec.Body.Bytes(), &pb)
					mu.Lock()
					res.errCodes[fmt.Sprintf("%d %s", rec.Code, pb.Code)]++
					mu.Unlock()
					continue
				}
				if rec.Code == http.StatusUnprocessableEntity {
					atomic.AddInt64(&res.timeouts, 1)
				}
				lat = append(lat, el)
				if h, ok := counterHold(tr.stmts); ok && sc.kind == "create" {
					holds = append(holds, h)
				}
				if c, ok := lastCommit(tr.stmts); ok && sc.kind == "create" {
					commits = append(commits, c)
				}
				for _, s := range tr.stmts {
					k := normSQL(s.SQL)
					stmts[k] = append(stmts[k], s.Dur)
				}
			}
			mu.Lock()
			res.lat = append(res.lat, lat...)
			res.holds = append(res.holds, holds...)
			res.commits = append(res.commits, commits...)
			for k, v := range stmts {
				res.stmtDur[k] = append(res.stmtDur[k], v...)
			}
			mu.Unlock()
		}()
	}
	wg.Wait()
	res.alloc = allocDelta(allocBefore, e.allocSnapshot(t))
	if res.errs > 0 {
		t.Errorf("%s: %d requests failed: %v", sc.name, res.errs, res.errCodes)
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
	if rec, ok := e.send(context.WithValue(t.Context(), perfTraceKey{}, tr), res.sc, r); !ok {
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
	if r.sc.unscored != "" {
		return 0
	}
	switch r.sc.kind {
	case "get":
		return perfGetTargetMS
	case "create":
		return perfCreateTargetMS
	case "timeout":
		return 0 // the bound check has no latency target
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
	// Quiesce the seed (Technical Lead, 2026-11-10): VACUUM (ANALYZE) does
	// the work autovacuum would otherwise do during the attempts, and
	// CHECKPOINT flushes the dirty pages, so neither lands on the creates.
	t0 := time.Now()
	e.exec(t, "VACUUM (ANALYZE)")
	e.exec(t, "CHECKPOINT")
	t.Logf("QUIESCE VACUUM (ANALYZE) and CHECKPOINT after the seed: %v", time.Since(t0).Round(time.Second))
	floor := e.transportFloor(t, clients)
	t.Logf("FLOOR SELECT 1 round trip, %d clients: p50=%.2f p95=%.2f p99=%.2f ms", clients, pct(floor, .50), pct(floor, .95), pct(floor, .99))
	if p50 := pct(floor, .50); p50 > perfFloorGateMS {
		t.Fatalf("host busy: transport floor p50 %.2f ms > %.0f ms gate; no numbers taken", p50, perfFloorGateMS)
	}
	var results []*perfResult
	for _, sc := range e.scenarios(t) {
		c := clients
		if sc.clients > 0 {
			c = sc.clients
		}
		if sc.before != nil {
			sc.before()
		}
		// A scored create row runs until its storage window passes the
		// gate, at most perfCommitAttempts times; every attempt is reported.
		attempts := 1
		if sc.kind == "create" && sc.unscored == "" {
			attempts = perfCommitAttempts
		}
		for a := 1; a <= attempts; a++ {
			probe := 0.0
			var st perfSetup
			if sc.kind == "create" {
				st.isolationWait = isolate(t)
				st.quiesceWait = e.quiesce(t)
				probe = e.hostProbe(t, &st)
			}
			r := e.run(t, sc, c, time.Duration(secs)*time.Second)
			r.attempt, r.probe, r.setup = a, probe, st
			t.Logf("RESULT %-62s attempt %d n=%6d p50=%8.2f p95=%8.2f p99=%8.2f ms commit p50=%6.2f host probe=%5.2f retry_later=%d gated=%q",
				sc.name, a, len(r.lat), pct(r.lat, .50), pct(r.lat, .95), pct(r.lat, .99), r.commitP50(), r.probe, r.retryLater(), r.gateReason())
			results = append(results, r)
			if !r.gated() {
				break
			}
		}
	}
	report := e.report(t, results, floor, clients, secs)
	if err := os.MkdirAll(filepath.Dir(out), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(out, []byte(report), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Logf("wrote %s", out)
	invalid := map[string]bool{}
	for _, r := range results {
		switch p95 := pct(r.lat, .95); {
		case r.gated():
			invalid[r.sc.name] = r.attempt == perfCommitAttempts
		case r.target() > 0 && p95 > r.target():
			invalid[r.sc.name] = false
			t.Errorf("MISS %s: p95 %.2f ms > %.0f ms", r.sc.name, p95, r.target())
		case r.target() > 0 && r.retryLater() > 0:
			invalid[r.sc.name] = false
			t.Errorf("MISS %s: %d retry_later in a valid attempt", r.sc.name, r.retryLater())
		default:
			invalid[r.sc.name] = false
		}
	}
	for name, bad := range invalid {
		if bad {
			t.Errorf("HOST INVALID for creates: %s had no valid attempt (host probe <= %.1f ms and commit p50 <= %.0f ms) in %d attempts", name, perfHostGateMS, perfCommitGateMS, perfCommitAttempts)
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
	w("## Results (ms, per request through the handler)\n\nA create row attempt is scored only when it passes every gate (CTO, 2026-11-10): the transport floor (p50 <= %.0f ms, checked once, the run stops otherwise), the host-speed probe (a 1-client `BEGIN; COMMIT` p50 <= %.1f ms, run just before the attempt) and the storage window (the create's own commit round trip p50 <= %.0f ms, from its trace). A gated attempt is neither a pass nor a fail and is re-run, at most %d attempts. A `503 retry_later` in a valid attempt is a miss. Round trips are one request's, the first actor's.\n\n", perfFloorGateMS, perfHostGateMS, perfCommitGateMS, perfCommitAttempts)
	w("| Scenario | Attempt | Tenant | n | p50 | p95 | p99 | Host probe p50 | Commit p50 | Round trips | retry_later | Target | Verdict |\n| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |\n")
	for _, r := range results {
		p95 := pct(r.lat, .95)
		v := "meets"
		switch {
		case r.sc.unscored != "":
			v = r.sc.unscored
		case r.gated():
			v = "gated (" + r.gateReason() + ")"
		case r.target() == 0:
			v = "n/a (bound check)"
		case p95 > r.target():
			v = "**MISS**"
		case r.sc.kind == "create" && r.retryLater() > 0:
			v = "**MISS** (retry_later)"
		}
		attempt, probe, commit, trips, rl := "-", "-", "-", "-", "-"
		if r.sc.kind == "create" {
			n, _, _ := roundTrips(r.stmtSQL)
			attempt, commit, trips = strconv.Itoa(max(r.attempt, 1)), fmt.Sprintf("%.2f", r.commitP50()), strconv.Itoa(n)
			probe, rl = fmt.Sprintf("%.2f", r.probe), strconv.FormatInt(r.retryLater(), 10)
		}
		w("| %s | %s | %s | %d | %.2f | %.2f | %.2f | %s | %s | %s | %s | %.0f | %s |\n", r.sc.name, attempt, r.sc.tenant, len(r.lat), pct(r.lat, .5), p95, pct(r.lat, .99), probe, commit, trips, rl, r.target(), v)
	}
	e.reportHolds(w, results, floor)
	reportAlloc(w, results)
	reportSetup(w, results)
	reportTimeouts(w, results)
	w("\n## Per statement (ms, as pgx traced it, all requests of the scenario)\n\n")
	for _, r := range results {
		trips, stmts, sizes := roundTrips(r.stmtSQL)
		w("### %s\n\nRound trips: %d, statements: %d, statements per batch: %v (one request, the first actor).\n\n| # | Statement | n | p50 | p95 | p99 |\n| --- | --- | --- | --- | --- | --- |\n", r.label(), trips, stmts, sizes)
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
		w("### %s\n\nRequest: `%s %s` as `%s`\n\n", r.label(), method, r.sc.reqs[0].path, r.sc.reqs[0].token)
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
		w("| %s | %d | %.2f / %.2f | %.2f / %.2f | %d | %.2f / %.2f | %.2f / %.2f | %.2f / %.2f |\n", r.label(), len(r.holds),
			pct(wait, .5), pct(wait, .95), pct(hold, .5), pct(hold, .95), stmts[len(stmts)/2],
			pct(tr, .5), pct(tr, .95), pct(db, .5), pct(db, .95), pct(app, .5), pct(app, .95))
	}
}

// roundTrips counts one request's round trips and statements: a batch
// (TICK-648) is one round trip carrying several statements.
func roundTrips(ss []perfStmt) (trips, stmts int, batchSizes []int) {
	for _, s := range ss {
		trips++
		if b, ok := strings.CutPrefix(s.SQL, perfBatchPrefix); ok {
			n := len(strings.Split(b, " ; "))
			stmts += n
			batchSizes = append(batchSizes, n)
			continue
		}
		stmts++
	}
	return trips, stmts, batchSizes
}

// reportTimeouts prints how data-filter lists ended (TICK-647): answered, or
// 422 filter_too_expensive, with the latency including the 422s.
func reportTimeouts(w func(string, ...any), results []*perfResult) {
	w("\n## Data-filter statement timeout (TICK-647)\n\nDefault timeout %s; the bound check runs with %s. A 422 is `filter_too_expensive`. Latency includes the 422s.\n\n| Scenario | Requests | 422 | p50 | p95 | max |\n| --- | --- | --- | --- | --- | --- |\n",
		perfDataFilterTimeoutLabel, perfShortTimeout)
	for _, r := range results {
		if r.sc.kind != "data" && r.sc.kind != "timeout" {
			continue
		}
		w("| %s | %d | %d | %.2f | %.2f | %.2f |\n", r.sc.name, len(r.lat), r.timeouts, pct(r.lat, .5), pct(r.lat, .95), pct(r.lat, 1))
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

// ---- allocator metrics (TICK-657) -------------------------------------------

// allocSnapshot gathers the nava_numbering_alloc_* families from the
// handler's telemetry registry.
func (e *perfEnv) allocSnapshot(t *testing.T) map[string]*dto.MetricFamily {
	t.Helper()
	mfs, err := e.tel.Registry.Gather()
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]*dto.MetricFamily{}
	for _, mf := range mfs {
		if strings.HasPrefix(mf.GetName(), "nava_numbering_alloc") {
			out[mf.GetName()] = mf
		}
	}
	return out
}

// allocDelta is after minus before, each family's series summed into one:
// the allocator's series carry no tenant or type attribute (T01).
func allocDelta(before, after map[string]*dto.MetricFamily) map[string]*dto.Metric {
	out := map[string]*dto.Metric{}
	for name, a := range after {
		d := sumFamily(a)
		if b, ok := before[name]; ok {
			bs := sumFamily(b)
			switch {
			case d.Histogram != nil && bs.Histogram != nil:
				d.Histogram.SampleCount = new(d.Histogram.GetSampleCount() - bs.Histogram.GetSampleCount())
				d.Histogram.SampleSum = new(d.Histogram.GetSampleSum() - bs.Histogram.GetSampleSum())
				for i := range d.Histogram.Bucket {
					if i < len(bs.Histogram.Bucket) {
						d.Histogram.Bucket[i].CumulativeCount = new(d.Histogram.Bucket[i].GetCumulativeCount() - bs.Histogram.Bucket[i].GetCumulativeCount())
					}
				}
			case d.Counter != nil && bs.Counter != nil:
				d.Counter.Value = new(d.Counter.GetValue() - bs.Counter.GetValue())
			}
		}
		out[name] = d
	}
	return out
}

// sumFamily folds a family's series into one new metric.
func sumFamily(mf *dto.MetricFamily) *dto.Metric {
	var out dto.Metric
	for _, m := range mf.GetMetric() {
		if h := m.GetHistogram(); h != nil {
			if out.Histogram == nil {
				out.Histogram = &dto.Histogram{SampleCount: new(uint64(0)), SampleSum: new(0.0)}
				for _, b := range h.GetBucket() {
					out.Histogram.Bucket = append(out.Histogram.Bucket, &dto.Bucket{UpperBound: new(b.GetUpperBound()), CumulativeCount: new(uint64(0))})
				}
			}
			out.Histogram.SampleCount = new(out.Histogram.GetSampleCount() + h.GetSampleCount())
			out.Histogram.SampleSum = new(out.Histogram.GetSampleSum() + h.GetSampleSum())
			for i, b := range h.GetBucket() {
				if i < len(out.Histogram.Bucket) {
					out.Histogram.Bucket[i].CumulativeCount = new(out.Histogram.Bucket[i].GetCumulativeCount() + b.GetCumulativeCount())
				}
			}
		}
		if c := m.GetCounter(); c != nil {
			if out.Counter == nil {
				out.Counter = &dto.Counter{Value: new(0.0)}
			}
			out.Counter.Value = new(out.Counter.GetValue() + c.GetValue())
		}
	}
	return &out
}

// histQuantile estimates quantile q in ms from a histogram in seconds,
// linearly inside the bucket, as Prometheus histogram_quantile does. Past
// the last bound it returns that bound and over.
func histQuantile(h *dto.Histogram, q float64) (ms float64, over bool) {
	n := float64(h.GetSampleCount())
	if n == 0 {
		return 0, false
	}
	rank := q * n
	lo, prev := 0.0, 0.0
	for _, b := range h.GetBucket() {
		c := float64(b.GetCumulativeCount())
		if c >= rank {
			if c == prev {
				return b.GetUpperBound() * 1000, false
			}
			return (lo + (b.GetUpperBound()-lo)*(rank-prev)/(c-prev)) * 1000, false
		}
		lo, prev = b.GetUpperBound(), c
	}
	return lo * 1000, true
}

// reportAlloc prints the allocator's metrics per create scenario: the pool
// acquire wait, the whole allocation (acquire included) and the timeouts
// that become retry_later, plus every failed request by status and code.
func reportAlloc(w func(string, ...any), results []*perfResult) {
	w("\n## Number allocation (creates, `nava_numbering_alloc_*`, ADR-005 L2s)\n\nPer scenario, from the handler's telemetry registry (bucket bounds 0.5, 1, 2.5, 5, 10, 25, 50, 100, 200, 500 ms; quantiles interpolated inside a bucket). Gapless creates take no allocation, so their row is empty. Failed requests are by HTTP status and problem code; `retry_later` is an allocation past its 200 ms budget.\n\n")
	w("| Scenario | Allocations | Acquire mean / p50 / p95 | Allocation mean / p50 / p95 | Timeouts | Failed requests |\n| --- | --- | --- | --- | --- | --- |\n")
	q := func(h *dto.Histogram) string {
		if h == nil || h.GetSampleCount() == 0 {
			return "-"
		}
		f := func(x float64) string {
			v, over := histQuantile(h, x)
			if over {
				return fmt.Sprintf("> %.1f", v)
			}
			return fmt.Sprintf("%.2f", v)
		}
		return fmt.Sprintf("%.2f / %s / %s", h.GetSampleSum()/float64(h.GetSampleCount())*1000, f(.5), f(.95))
	}
	for _, r := range results {
		if r.sc.kind != "create" {
			continue
		}
		var acq, lat *dto.Histogram
		timeouts := 0.0
		for name, m := range r.alloc {
			switch {
			case strings.HasPrefix(name, "nava_numbering_alloc_acquire"):
				acq = m.GetHistogram()
			case strings.HasPrefix(name, "nava_numbering_alloc_timeouts"):
				timeouts = m.GetCounter().GetValue()
			case strings.HasPrefix(name, "nava_numbering_alloc_seconds"):
				lat = m.GetHistogram()
			}
		}
		n := uint64(0)
		if lat != nil {
			n = lat.GetSampleCount()
		}
		failed := "none"
		if len(r.errCodes) > 0 {
			keys := make([]string, 0, len(r.errCodes))
			for k, v := range r.errCodes {
				keys = append(keys, fmt.Sprintf("%s x%d", k, v))
			}
			slices.Sort(keys)
			failed = strings.Join(keys, ", ")
		}
		w("| %s | %d | %s | %s | %.0f | %s |\n", r.label(), n, q(acq), q(lat), timeouts, failed)
	}
}

// reportSetup prints, per create attempt, the method steps before it and
// what pg_stat_activity and docker stats showed during its host probe.
func reportSetup(w func(string, ...any), results []*perfResult) {
	w("\n## Attempt setup and host-probe samples (creates)\n\nBefore each create attempt: wait until no other `nava-it-pg-*` or `nava-aclperf-*` container runs, then until no autovacuum or analyze backend works in the bench database, then the host-speed probe. `pg_stat_activity` is read at the probe's midpoint (every non-idle backend and every non-client backend, other than the reader); `docker stats` is one sample taken while the probe runs. The seed was quiesced once with `VACUUM (ANALYZE)` and `CHECKPOINT`.\n\n")
	for _, r := range results {
		if r.sc.kind != "create" {
			continue
		}
		w("### %s\n\nIsolation wait %v, quiesce wait %v, host probe p50 %.2f ms.\n\n```\n", r.label(), r.setup.isolationWait.Round(time.Second), r.setup.quiesceWait.Round(time.Second), r.probe)
		w("pg_stat_activity:\n")
		for _, l := range r.setup.activity {
			w("  %s\n", l)
		}
		w("docker stats:\n")
		for _, l := range r.setup.dockerStats {
			w("  %s\n", l)
		}
		w("```\n\n")
	}
}

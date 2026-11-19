//go:build perf

// SP-1 (TICK-932, PLAN-011 P11-01): the agent list "sort=-updated_at", first
// page, on the P3-36 800k seed, with ADR-010 D7's region leaf as a fifth
// ReadBranches branch (pf_regional_view, last, so it carries the negation of
// the four existing branches, as authz.ReadBranches renders them). The count
// and page statements are the ones the handler emitted in P3-36
// (docs/quality/acl-perf-800k/final-2026-11-09-full.md, "agent list,
// sort=-updated_at"), plus that branch. Each request is the list's database
// work: BEGIN, set_config(app.tenant_id), count, page, COMMIT, as nava_app with
// QueryExecModeExec and plan_cache_mode = force_custom_plan (custom plans).
//
// Bench only: no runtime code is involved or changed. region.sql must have
// run first. Env: SP1_SUPER_URL, SP1_APP_URL, SP1_SECONDS (20), SP1_CLIENTS
// (8), SP1_OUT (raw report path).
package main

import (
	"context"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const cols = `r.tenant_id, r.id, r.entity_definition_id, r.version, r.schema_version, r.number, r.state,
  r.requester_id, r.assignee_id, r.assignment_group_id, r.created_by, r.created_at, r.updated_at,
  r.data, r.causation_origin_id, r.causation_depth, r.updated_at AS sort_key`

// The disjuncts in ReadBranches order; the region leaf is pf_regional_view
// with is_regional_mgr folded to true for the caller.
var disjuncts = []string{
	`(true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))`,
	`((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))`,
	`((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))`,
	`(EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[])))))`,
	`(true AND (r.org_unit_id IS NOT NULL AND r.org_unit_id = ANY($9::uuid[])))`,
}

func branches(n int) []string {
	var out []string
	for i := 0; i < n; i++ {
		b := disjuncts[i]
		for j := 0; j < i; j++ {
			b += " AND NOT " + disjuncts[j]
		}
		out = append(out, "("+b+")")
	}
	return out
}

func where(b string) string {
	return "WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (" + b + ")"
}

func countSQL(n int) string {
	var parts []string
	for _, b := range branches(n) {
		parts = append(parts, "(SELECT 1 FROM records r "+where(b)+" LIMIT 1001)")
	}
	// The count has no LIMIT $8, so the region set is its $8 (as the
	// handler numbers arguments in order of use).
	return strings.ReplaceAll("SELECT count(*) FROM (SELECT 1 FROM ("+strings.Join(parts, " UNION ALL ")+") u LIMIT 1001) c", "$9", "$8")
}

func pageSQL(n int) string {
	var parts []string
	for _, b := range branches(n) {
		parts = append(parts, "(SELECT "+cols+" FROM records r\n"+where(b)+" ORDER BY r.updated_at DESC NULLS FIRST, r.id DESC LIMIT $8)")
	}
	return `SELECT x.tenant_id, x.id, x.entity_definition_id, d.key, x.version, x.schema_version, x.number, x.state,
       x.requester_id, x.assignee_id, x.assignment_group_id, x.created_by, x.created_at, x.updated_at,
       x.data, x.causation_origin_id, coalesce(x.causation_depth, 0), coalesce(g.deleted_at IS NOT NULL, false),
       x.sort_key::text FROM (
` + strings.Join(parts, "\nUNION ALL\n") + `
ORDER BY sort_key DESC NULLS FIRST, id DESC LIMIT $8
) x
JOIN entity_definitions d ON d.tenant_id = x.tenant_id AND d.id = x.entity_definition_id
LEFT JOIN groups g ON g.tenant_id = x.tenant_id AND g.id = x.assignment_group_id
ORDER BY x.sort_key DESC NULLS FIRST, x.id DESC`
}

// expandSQL is D7's subtree expansion: one recursive query over org_units
// under the request's tenant (RLS). The bench has no deleted_at yet.
const expandSQL = `WITH RECURSIVE s(id) AS (
  SELECT id FROM org_units WHERE tenant_id = $1 AND id = ANY($2::uuid[])
  UNION
  SELECT o.id FROM org_units o JOIN s ON o.tenant_id = $1 AND o.parent_id = s.id
) SELECT count(*) FROM s`

type agent struct {
	id     string
	groups []string
}

type env struct {
	super  *pgx.Conn
	app    *pgxpool.Pool
	tenant string
	def    string
	agents []agent
	out    *strings.Builder
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func envInt(k string, d int) int {
	if v, err := strconv.Atoi(os.Getenv(k)); err == nil && v > 0 {
		return v
	}
	return d
}

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

func md5ids(ctx context.Context, c *pgx.Conn, keys []string) []string {
	var out []string
	for _, k := range keys {
		var id string
		must(c.QueryRow(ctx, "SELECT md5($1)::uuid::text", k).Scan(&id))
		out = append(out, id)
	}
	return out
}

// countArgs drops the page's LIMIT argument.
func countArgs(args []any) []any {
	return append(slices.Clone(args[:7]), args[8:]...)
}

func (e *env) args(a agent, s []string) []any {
	args := []any{e.tenant, e.def, a.groups, a.id, a.id, a.id, a.groups, 51}
	if s != nil {
		args = append(args, s)
	}
	return args
}

// request is one list's database work on one connection.
func (e *env) request(ctx context.Context, cs, ps string, args []any) (time.Duration, error) {
	t0 := time.Now()
	tx, err := e.app.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, "SELECT set_config('app.tenant_id', $1, true)", e.tenant); err != nil {
		return 0, err
	}
	var n int
	if err := tx.QueryRow(ctx, cs, countArgs(args)...).Scan(&n); err != nil {
		return 0, err
	}
	rows, err := tx.Query(ctx, ps, args...)
	if err != nil {
		return 0, err
	}
	for rows.Next() {
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return time.Since(t0), nil
}

func (e *env) load(ctx context.Context, name string, clients, secs int, one func(context.Context, agent) (time.Duration, error)) (p50, p95, p99 float64, n int) {
	for _, a := range e.agents { // warm-up: each request once
		_, err := one(ctx, a)
		must(err)
	}
	var mu sync.Mutex
	var all []time.Duration
	stop := time.Now().Add(time.Duration(secs) * time.Second)
	var wg sync.WaitGroup
	for c := 0; c < clients; c++ {
		wg.Add(1)
		go func(c int) {
			defer wg.Done()
			var mine []time.Duration
			for i := c; time.Now().Before(stop); i += clients {
				d, err := one(ctx, e.agents[i%len(e.agents)])
				must(err)
				mine = append(mine, d)
			}
			mu.Lock()
			all = append(all, mine...)
			mu.Unlock()
		}(c)
	}
	wg.Wait()
	p50, p95, p99 = pct(all, .50), pct(all, .95), pct(all, .99)
	fmt.Printf("%-40s n=%6d p50=%7.2f p95=%7.2f p99=%7.2f ms\n", name, len(all), p50, p95, p99)
	return p50, p95, p99, len(all)
}

func (e *env) explain(ctx context.Context, title, sql string, args []any) {
	tx, err := e.app.Begin(ctx)
	must(err)
	defer func() { _ = tx.Rollback(ctx) }()
	_, err = tx.Exec(ctx, "SELECT set_config('app.tenant_id', $1, true)", e.tenant)
	must(err)
	rows, err := tx.Query(ctx, "EXPLAIN (ANALYZE, BUFFERS, SETTINGS) "+sql, args...)
	must(err)
	fmt.Fprintf(e.out, "\n#### %s\n\n```\n", title)
	for rows.Next() {
		var l string
		must(rows.Scan(&l))
		fmt.Fprintln(e.out, l)
	}
	must(rows.Err())
	fmt.Fprint(e.out, "```\n")
}

func (e *env) exec(ctx context.Context, sql string) {
	t0 := time.Now()
	_, err := e.super.Exec(ctx, sql)
	must(err)
	fmt.Printf("super: %s (%v)\n", sql, time.Since(t0).Round(time.Millisecond))
}

func main() {
	ctx := context.Background()
	clients, secs := envInt("SP1_CLIENTS", 8), envInt("SP1_SECONDS", 20)
	super, err := pgx.Connect(ctx, os.Getenv("SP1_SUPER_URL"))
	must(err)
	defer super.Close(ctx)
	pc, err := pgxpool.ParseConfig(os.Getenv("SP1_APP_URL"))
	must(err)
	pc.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeExec
	pc.MaxConns = int32(clients + 2) //nolint:gosec // small env setting
	app, err := pgxpool.NewWithConfig(ctx, pc)
	must(err)
	defer app.Close()
	e := &env{super: super, app: app, out: &strings.Builder{}}

	// The P3-36 harness's 200 large-tenant agents (records_perf_integration_test.go actors).
	var agentsGroup string
	must(super.QueryRow(ctx, `SELECT t.tenant_id::text, d.id::text, t.agents::text FROM perf_tenant t
		JOIN entity_definitions d ON d.tenant_id = t.tenant_id AND d.key = 'ticket' WHERE t.tn = 0`).Scan(&e.tenant, &e.def, &agentsGroup))
	for n := 1; n <= 200; n++ {
		u := (n*24 + 7) % 5000
		g := md5ids(ctx, super, []string{fmt.Sprintf("g0/%d", u%50), fmt.Sprintf("g0/%d", (u+17)%50), fmt.Sprintf("g0/%d", (u+34)%50)})
		id := md5ids(ctx, super, []string{fmt.Sprintf("u0/%d", u)})[0]
		e.agents = append(e.agents, agent{id: id, groups: append([]string{agentsGroup}, g...)})
	}

	// S_k: the first k units, in pre-order, of the subtree rooted at an area,
	// a district, a region and the root (region.sql).
	type set struct {
		k     int
		start int // pre of the subtree root
		node  string
		size  int
		ids   []string
		match int
	}
	sets := []*set{{k: 10, start: 3, node: "area 0/0/0", size: 10}, {k: 100, start: 2, node: "district 0/0", size: 101},
		{k: 1000, start: 1, node: "region 0", size: 1011}, {k: 10000, start: 0, node: "root", size: 10111}}
	for _, s := range sets {
		rows, err := super.Query(ctx, "SELECT id::text FROM bench_ou WHERE pre >= $1 AND pre < $1 + $2 ORDER BY pre", s.start, s.k)
		must(err)
		for rows.Next() {
			var id string
			must(rows.Scan(&id))
			s.ids = append(s.ids, id)
		}
		must(rows.Err())
		must(super.QueryRow(ctx, "SELECT count(*) FROM records WHERE tenant_id = $1 AND org_unit_id = ANY($2::uuid[])", e.tenant, s.ids).Scan(&s.match))
	}
	var total, nulls int
	must(super.QueryRow(ctx, "SELECT count(*), count(*) FILTER (WHERE org_unit_id IS NULL) FROM records WHERE tenant_id = $1", e.tenant).Scan(&total, &nulls))

	fmt.Fprintf(e.out, "# SP-1 raw report (TICK-932)\n\n%s, %d clients, %d s per row, nearest-rank.\n\n", time.Now().UTC().Format(time.RFC3339), clients, secs)
	fmt.Fprintf(e.out, "Large tenant: %d records, %d with NULL org_unit_id.\n\n| k | subtree root | subtree size | records with org_unit_id in S |\n| --- | --- | --- | --- |\n", total, nulls)
	for _, s := range sets {
		fmt.Fprintf(e.out, "| %d | %s | %d | %d |\n", s.k, s.node, s.size, s.match)
	}

	prewarm := "SELECT pg_prewarm(c.oid) FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace WHERE n.nspname = 'public' AND c.relkind IN ('r', 'i')"
	fmt.Fprint(e.out, "\n## Results (ms per request: BEGIN, set_config, count, page, COMMIT)\n\n| Row | index | n | p50 | p95 | p99 |\n| --- | --- | --- | --- | --- | --- |\n")
	type row struct {
		name, index string
		s           *set
	}
	run := func(r row) {
		n := 4
		var ids []string
		if r.s != nil {
			n, ids = 5, r.s.ids
		}
		cs, ps := countSQL(n), pageSQL(n)
		p50, p95, p99, cnt := e.load(ctx, r.name+" / "+r.index, clients, secs, func(ctx context.Context, a agent) (time.Duration, error) {
			return e.request(ctx, cs, ps, e.args(a, ids))
		})
		fmt.Fprintf(e.out, "| %s | %s | %d | %.2f | **%.2f** | %.2f |\n", r.name, r.index, cnt, p50, p95, p99)
	}
	var plansOut strings.Builder
	explainRow := func(r row) {
		n := 4
		var ids []string
		if r.s != nil {
			n, ids = 5, r.s.ids
		}
		saved := e.out
		e.out = &plansOut
		e.explain(ctx, r.name+" / "+r.index+": page", pageSQL(n), e.args(e.agents[0], ids))
		e.explain(ctx, r.name+" / "+r.index+": count", countSQL(n), countArgs(e.args(e.agents[0], ids)))
		e.out = saved
	}

	phase := func(index string, ddl []string, withBaseline bool) {
		for _, q := range ddl {
			e.exec(ctx, q)
		}
		e.exec(ctx, "ANALYZE records")
		e.exec(ctx, "CHECKPOINT")
		e.exec(ctx, prewarm)
		rs := []row{}
		if withBaseline {
			rs = append(rs, row{name: "baseline (4 branches, no region leaf)", index: index})
		}
		for _, s := range sets {
			rs = append(rs, row{name: fmt.Sprintf("|S| = %d", s.k), index: index, s: s})
		}
		for _, r := range rs {
			run(r)
			explainRow(r)
		}
	}
	phase("plain", []string{
		"DROP INDEX IF EXISTS records_bench_org_unit_updated",
		"CREATE INDEX IF NOT EXISTS records_bench_org_unit ON records (tenant_id, org_unit_id)",
	}, true)
	phase("plain + R1", []string{
		"CREATE INDEX IF NOT EXISTS records_bench_org_unit_updated ON records (tenant_id, org_unit_id, updated_at DESC, id DESC)",
	}, false)

	// D7's expansion, per subtree root, alone.
	fmt.Fprint(e.out, "\n## Subtree expansion (ms per request: BEGIN, set_config, recursive query, COMMIT)\n\n| Subtree | units | p50 | p95 | p99 |\n| --- | --- | --- | --- | --- |\n")
	for _, s := range sets {
		root := []string{s.ids[0]}
		var got int
		p50, p95, p99, _ := e.load(ctx, "expand "+s.node, clients, secs, func(ctx context.Context, _ agent) (time.Duration, error) {
			t0 := time.Now()
			tx, err := e.app.Begin(ctx)
			if err != nil {
				return 0, err
			}
			defer func() { _ = tx.Rollback(ctx) }()
			if _, err := tx.Exec(ctx, "SELECT set_config('app.tenant_id', $1, true)", e.tenant); err != nil {
				return 0, err
			}
			if err := tx.QueryRow(ctx, expandSQL, e.tenant, root).Scan(&got); err != nil {
				return 0, err
			}
			return time.Since(t0), tx.Commit(ctx)
		})
		fmt.Fprintf(e.out, "| %s | %d | %.2f | **%.2f** | %.2f |\n", s.node, got, p50, p95, p99)
	}
	// The floor: a request with no statement but set_config.
	fmt.Fprint(e.out, "\n## Transport floor\n\n")
	p50, p95, _, _ := e.load(ctx, "floor", clients, secs, func(ctx context.Context, _ agent) (time.Duration, error) {
		t0 := time.Now()
		tx, err := e.app.Begin(ctx)
		if err != nil {
			return 0, err
		}
		defer func() { _ = tx.Rollback(ctx) }()
		if _, err := tx.Exec(ctx, "SELECT set_config('app.tenant_id', $1, true)", e.tenant); err != nil {
			return 0, err
		}
		return time.Since(t0), tx.Commit(ctx)
	})
	fmt.Fprintf(e.out, "BEGIN + set_config + COMMIT: p50 %.2f, p95 %.2f ms\n", p50, p95)
	fmt.Fprint(e.out, "\n## Plans (EXPLAIN (ANALYZE, BUFFERS, SETTINGS), agent-1, as nava_app in the tenant transaction)\n")
	e.out.WriteString(plansOut.String())
	must(os.WriteFile(os.Getenv("SP1_OUT"), []byte(e.out.String()), 0o600))
	fmt.Println("wrote", os.Getenv("SP1_OUT"))
}

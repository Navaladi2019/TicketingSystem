//go:build perf

// SP-1b (TICK-969, ADR-010 D7/R1): the SP-1 agent list (sort=-updated_at,
// first page, five ReadBranches branches, region last) with the region leaf
// rendered against a closure table read as a caller-only subquery:
//
//	C1  org_unit_id IN (SELECT descendant_id FROM org_unit_closure WHERE tenant_id = $1 AND ancestor_id = ANY($U))
//	C2  org_unit_id = ANY(ARRAY(SELECT descendant_id FROM org_unit_closure WHERE ...))
//	B1  org_unit_id = ANY((SELECT $S)::uuid[])      control, S bound from Go
//	D7  org_unit_id = ANY($S::uuid[])               SP-1's rendering, |S| = 1,000 only
//
// Same request, method and session settings as SP-1 (../main.go). Bench only.
// Env: SP1B_SUPER_URL, SP1B_APP_URL, SP1B_CLIENTS (8), SP1B_SECONDS (20),
// SP1B_PHASE (plain | r1 | build | single), SP1B_ONLY (regexp over row names),
// SP1B_EXPLAIN (1), SP1B_OUT (raw report path, appended).
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
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

var base = []string{ //nolint:gochecknoglobals // read-only test/bench fixture, set once
	`(true AND (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($3::uuid[])))`,
	`((r.requester_id IS NOT NULL AND r.requester_id = $4::uuid))`,
	`((r.assignee_id IS NOT NULL AND r.assignee_id = $5::uuid))`,
	`(EXISTS (SELECT 1 FROM record_shares acl_share WHERE acl_share.tenant_id = r.tenant_id AND acl_share.record_id = r.id AND ((acl_share.principal_kind = 'user' AND acl_share.principal_id = $6::uuid) OR (acl_share.principal_kind = 'group' AND acl_share.principal_id = ANY($7::uuid[])))))`,
}

const closureSub = `SELECT acl_ou.descendant_id FROM org_unit_closure acl_ou WHERE acl_ou.tenant_id = $1 AND acl_ou.ancestor_id = ANY($9::uuid[])`

var leaves = map[string]string{ //nolint:gochecknoglobals // read-only test/bench fixture, set once
	"C1": `(true AND (r.org_unit_id IS NOT NULL AND r.org_unit_id IN (` + closureSub + `)))`,
	"C2": `(true AND (r.org_unit_id IS NOT NULL AND r.org_unit_id = ANY(ARRAY(` + closureSub + `))))`,
	"B1": `(true AND (r.org_unit_id IS NOT NULL AND r.org_unit_id = ANY((SELECT $9::uuid[])::uuid[])))`,
	"D7": `(true AND (r.org_unit_id IS NOT NULL AND r.org_unit_id = ANY($9::uuid[])))`,
}

func branches(leaf string) []string {
	d := slices.Clone(base)
	if leaf != "" {
		d = append(d, leaves[leaf])
	}
	var out []string
	for i := range d {
		b := d[i]
		for j := 0; j < i; j++ {
			b += " AND NOT " + d[j]
		}
		out = append(out, "("+b+")")
	}
	return out
}

func where(b string) string {
	return "WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND true AND (" + b + ")"
}

func countSQL(leaf string) string {
	var parts []string
	for _, b := range branches(leaf) {
		parts = append(parts, "(SELECT 1 FROM records r "+where(b)+" LIMIT 1001)")
	}
	return strings.ReplaceAll("SELECT count(*) FROM (SELECT 1 FROM ("+strings.Join(parts, " UNION ALL ")+") u LIMIT 1001) c", "$9", "$8")
}

func pageSQL(leaf string) string {
	var parts []string
	for _, b := range branches(leaf) {
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

const existsSQL = `SELECT EXISTS (SELECT 1 FROM org_unit_closure WHERE tenant_id = $1 AND ancestor_id = ANY($2::uuid[]) AND descendant_id = $3)`

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
	out    *os.File
	plans  strings.Builder
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func envInt(k string, d int) int {
	if v, err := strconv.Atoi(os.Getenv(k)); err == nil && v >= 0 {
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

func (e *env) w(f string, a ...any) { fmt.Fprintf(e.out, f, a...) }

func md5id(ctx context.Context, c *pgx.Conn, k string) string {
	var id string
	must(c.QueryRow(ctx, "SELECT md5($1)::uuid::text", k).Scan(&id))
	return id
}

func countArgs(args []any) []any { return append(slices.Clone(args[:7]), args[8:]...) }

// args: $9 is U (the caller's units) for C1/C2 and S (the expanded set) for B1/D7.
func (e *env) args(a agent, p []string) []any {
	args := []any{e.tenant, e.def, a.groups, a.id, a.id, a.id, a.groups, 51}
	if p != nil {
		args = append(args, p)
	}
	return args
}

func (e *env) tx(ctx context.Context, f func(pgx.Tx) error) (time.Duration, error) {
	t0 := time.Now()
	tx, err := e.app.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, "SELECT set_config('app.tenant_id', $1, true)", e.tenant); err != nil {
		return 0, err
	}
	if err := f(tx); err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return time.Since(t0), nil
}

func (e *env) request(ctx context.Context, cs, ps string, args []any) (time.Duration, error) {
	return e.tx(ctx, func(tx pgx.Tx) error {
		var n int
		if err := tx.QueryRow(ctx, cs, countArgs(args)...).Scan(&n); err != nil {
			return err
		}
		rows, err := tx.Query(ctx, ps, args...)
		if err != nil {
			return err
		}
		for rows.Next() {
		}
		rows.Close()
		return rows.Err()
	})
}

// load: a warm-up pass over the 200 agents (spread over the clients, at most
// 15 s), then
// clients closed-loop workers for secs seconds.
func (e *env) load(ctx context.Context, name string, clients, secs int, one func(context.Context, agent) (time.Duration, error)) (p50, p95, p99 float64, n int) {
	var wg sync.WaitGroup
	warmStop := time.Now().Add(15 * time.Second) // a 1.7 s request must not stall the run
	for c := 0; c < clients; c++ {
		wg.Add(1)
		go func(c int) {
			defer wg.Done()
			for i := c; i < len(e.agents) && time.Now().Before(warmStop); i += clients {
				_, err := one(ctx, e.agents[i])
				must(err)
			}
		}(c)
	}
	wg.Wait()
	var mu sync.Mutex
	var all []time.Duration
	stop := time.Now().Add(time.Duration(secs) * time.Second)
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
	fmt.Printf("%s %-44s n=%6d p50=%7.2f p95=%7.2f p99=%7.2f ms\n", time.Now().UTC().Format("15:04:05"), name, len(all), p50, p95, p99)
	return p50, p95, p99, len(all)
}

// floor is P3-36's transport floor: clients x 200 `SELECT 1` on the pool.
func (e *env) floor(ctx context.Context, clients int) (float64, float64) {
	var mu sync.Mutex
	var wg sync.WaitGroup
	var out []time.Duration
	for c := 0; c < clients; c++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var lat []time.Duration
			for i := 0; i < 200; i++ {
				t0 := time.Now()
				var one int
				must(e.app.QueryRow(ctx, "SELECT 1").Scan(&one))
				lat = append(lat, time.Since(t0))
			}
			mu.Lock()
			out = append(out, lat...)
			mu.Unlock()
		}()
	}
	wg.Wait()
	return pct(out, .5), pct(out, .95)
}

func foreign() string {
	b, _ := exec.CommandContext(context.Background(), "docker", "ps", "--format", "{{.Names}}").Output()
	allowed := map[string]bool{"buildx_buildkit_nava-ci0": true, "multica-backend-1": true, "multica-frontend-1": true, "multica-postgres-1": true, os.Getenv("SP1B_PG"): true}
	var f []string
	for _, n := range strings.Fields(string(b)) {
		if !allowed[n] {
			f = append(f, n)
		}
	}
	return strings.Join(f, " ")
}

// planStats walks an EXPLAIN (ANALYZE, FORMAT JSON) plan and reports the
// region branch: the 5th child of the (Merge) Append over the branches.
type node = map[string]any

func num(n node, k string) float64 { v, _ := n[k].(float64); return v }

func children(n node) []node {
	var out []node
	ps, _ := n["Plans"].([]any)
	for _, p := range ps {
		out = append(out, p.(node))
	}
	return out
}

func findBranches(n node) []node {
	t, _ := n["Node Type"].(string)
	if t == "Append" || t == "Merge Append" {
		var direct []node
		for _, c := range children(n) {
			if r, _ := c["Parent Relationship"].(string); r == "Member" {
				direct = append(direct, c)
			}
		}
		if len(direct) >= 4 {
			return direct
		}
	}
	for _, c := range children(n) {
		if b := findBranches(c); b != nil {
			return b
		}
	}
	return nil
}

type stats struct {
	shape                       []string
	read, removed, closure, hbk float64
	executed                    bool
}

func walk(n node, s *stats) {
	t, _ := n["Node Type"].(string)
	loops := num(n, "Actual Loops")
	label := t
	if ix, ok := n["Index Name"].(string); ok {
		label += " " + ix
	} else if rel, ok := n["Relation Name"].(string); ok {
		label += " " + rel
	}
	if r, _ := n["Parent Relationship"].(string); r == "InitPlan" || r == "SubPlan" {
		label = r + "(" + label
		if _, h := n["Subplan Name"].(string); h {
			label += " " + n["Subplan Name"].(string)
		}
		label += ")"
	}
	if loops == 0 {
		label += " [never executed]"
	}
	s.shape = append(s.shape, label)
	rel, _ := n["Relation Name"].(string)
	if rel == "records" {
		s.read += num(n, "Actual Rows")*loops + (num(n, "Rows Removed by Filter")+num(n, "Rows Removed by Index Recheck"))*loops
		s.removed += (num(n, "Rows Removed by Filter") + num(n, "Rows Removed by Index Recheck")) * loops
		s.hbk += num(n, "Exact Heap Blocks") + num(n, "Lossy Heap Blocks")
		if loops > 0 {
			s.executed = true
		}
	}
	if t == "Bitmap Index Scan" {
		if ix, _ := n["Index Name"].(string); strings.HasPrefix(ix, "records") {
			s.read += 0 // the heap scan above counts the rows
		}
	}
	if rel == "org_unit_closure" {
		s.closure += num(n, "Actual Rows") * loops
	}
	for _, c := range children(n) {
		walk(c, s)
	}
}

type planInfo struct {
	planMS, execMS float64
	region         *stats
}

func (e *env) explain(ctx context.Context, title, sql string, args []any) planInfo {
	var pi planInfo
	_, err := e.tx(ctx, func(tx pgx.Tx) error {
		var raw []byte
		if err := tx.QueryRow(ctx, "EXPLAIN (ANALYZE, BUFFERS, SETTINGS, FORMAT JSON) "+sql, args...).Scan(&raw); err != nil {
			return err
		}
		var doc []map[string]any
		if err := json.Unmarshal(raw, &doc); err != nil {
			return err
		}
		pi.planMS, _ = doc[0]["Planning Time"].(float64)
		pi.execMS, _ = doc[0]["Execution Time"].(float64)
		if b := findBranches(doc[0]["Plan"].(node)); len(b) >= 5 {
			s := &stats{}
			walk(b[4], s)
			pi.region = s
		}
		rows, err := tx.Query(ctx, "EXPLAIN (ANALYZE, BUFFERS, SETTINGS) "+sql, args...)
		if err != nil {
			return err
		}
		fmt.Fprintf(&e.plans, "\n#### %s\n\n```\n", title)
		for rows.Next() {
			var l string
			if err := rows.Scan(&l); err != nil {
				return err
			}
			if len(l) > 300 {
				l = l[:300] + fmt.Sprintf("… (%d chars)", len(l))
			}
			fmt.Fprintf(&e.plans, "%s\n", l)
		}
		fmt.Fprint(&e.plans, "```\n")
		return rows.Err()
	})
	must(err)
	return pi
}

func (e *env) exec(ctx context.Context, sql string) time.Duration {
	t0 := time.Now()
	_, err := e.super.Exec(ctx, sql)
	must(err)
	d := time.Since(t0)
	fmt.Printf("super: %.80s (%v)\n", sql, d.Round(time.Millisecond))
	return d
}

type set struct {
	label string
	root  string   // U = {root}
	ids   []string // S = the whole subtree, as the closure returns it
	match int
}

func main() {
	ctx := context.Background()
	clients, secs := envInt("SP1B_CLIENTS", 8), envInt("SP1B_SECONDS", 20)
	phase := os.Getenv("SP1B_PHASE")
	only := regexp.MustCompile(os.Getenv("SP1B_ONLY"))
	super, err := pgx.Connect(ctx, os.Getenv("SP1B_SUPER_URL"))
	must(err)
	defer super.Close(ctx)
	pc, err := pgxpool.ParseConfig(os.Getenv("SP1B_APP_URL"))
	must(err)
	pc.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeExec
	pc.MaxConns = int32(clients + 2) //nolint:gosec // small env setting
	app, err := pgxpool.NewWithConfig(ctx, pc)
	must(err)
	defer app.Close()
	out, err := os.OpenFile(os.Getenv("SP1B_OUT"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600) //nolint:gosec // bench output path is operator-supplied
	must(err)
	defer out.Close()
	e := &env{super: super, app: app, out: out}

	var agentsGroup string
	must(super.QueryRow(ctx, `SELECT t.tenant_id::text, d.id::text, t.agents::text FROM perf_tenant t
		JOIN entity_definitions d ON d.tenant_id = t.tenant_id AND d.key = 'ticket' WHERE t.tn = 0`).Scan(&e.tenant, &e.def, &agentsGroup))
	var pcm string
	must(super.QueryRow(ctx, "SELECT coalesce((SELECT array_to_string(setconfig, ',') FROM pg_db_role_setting s JOIN pg_roles r ON r.oid = s.setrole WHERE r.rolname = 'nava_app'), '')").Scan(&pcm))

	e.w("\n## %s phase=%s clients=%d secs=%d\n\nnava_app role settings: `%s`. Foreign containers at start: `%s`.\n", time.Now().UTC().Format(time.RFC3339), phase, clients, secs, pcm, foreign())

	if phase == "build" {
		// Closure build for the whole tree, timed, three times (truncate between).
		build := `INSERT INTO org_unit_closure (tenant_id, ancestor_id, descendant_id, depth)
WITH RECURSIVE c(tenant_id, ancestor_id, descendant_id, depth) AS (
  SELECT tenant_id, id, id, 0 FROM org_units WHERE tenant_id = '` + e.tenant + `'
  UNION ALL
  SELECT c.tenant_id, c.ancestor_id, o.id, c.depth + 1
  FROM c JOIN org_units o ON o.tenant_id = c.tenant_id AND o.parent_id = c.descendant_id
) SELECT * FROM c`
		e.w("\n### Closure build (full tree, one statement, superuser)\n\n| Run | INSERT ms | ANALYZE ms | rows |\n| --- | --- | --- | --- |\n")
		for i := 1; i <= 3; i++ {
			e.exec(ctx, "TRUNCATE org_unit_closure")
			d := e.exec(ctx, build)
			a := e.exec(ctx, "ANALYZE org_unit_closure")
			var n int
			must(super.QueryRow(ctx, "SELECT count(*) FROM org_unit_closure").Scan(&n))
			e.w("| %d | %.1f | %.1f | %d |\n", i, float64(d.Microseconds())/1000, float64(a.Microseconds())/1000, n)
		}
		var units, sz int
		must(super.QueryRow(ctx, "SELECT count(*) FROM org_units WHERE tenant_id = $1", e.tenant).Scan(&units))
		must(super.QueryRow(ctx, "SELECT pg_total_relation_size('org_unit_closure')").Scan(&sz))
		e.w("\nUnits: %d. Closure size with PK: %d kB.\n", units, sz/1024)
		e.exec(ctx, "VACUUM (ANALYZE) org_unit_closure")
		return
	}

	for n := 1; n <= 200; n++ {
		u := (n*24 + 7) % 5000
		g := []string{agentsGroup, md5id(ctx, super, fmt.Sprintf("g0/%d", u%50)), md5id(ctx, super, fmt.Sprintf("g0/%d", (u+17)%50)), md5id(ctx, super, fmt.Sprintf("g0/%d", (u+34)%50))}
		e.agents = append(e.agents, agent{id: md5id(ctx, super, fmt.Sprintf("u0/%d", u)), groups: g})
	}

	// Ancestors (region.sql pre-order): area 0/0/0 (pre 3), district 0/0 (2),
	// region 0 (1), root (0). S = the subtree the closure returns.
	var sets []*set
	for _, a := range []struct {
		pre  int
		name string
	}{{3, "area"}, {2, "district"}, {1, "region"}, {0, "root"}} {
		s := &set{}
		must(super.QueryRow(ctx, "SELECT id::text FROM bench_ou WHERE pre = $1", a.pre).Scan(&s.root))
		rows, err := super.Query(ctx, "SELECT descendant_id::text FROM org_unit_closure WHERE tenant_id = $1 AND ancestor_id = $2 ORDER BY descendant_id", e.tenant, s.root)
		must(err)
		for rows.Next() {
			var id string
			must(rows.Scan(&id))
			s.ids = append(s.ids, id)
		}
		must(rows.Err())
		must(super.QueryRow(ctx, "SELECT count(*) FROM records WHERE tenant_id = $1 AND org_unit_id = ANY($2::uuid[])", e.tenant, s.ids).Scan(&s.match))
		s.label = fmt.Sprintf("%s, %d units", a.name, len(s.ids))
		sets = append(sets, s)
	}
	// SP-1's |S| = 1,000 set: the first 1,000 units of region 0 in pre-order.
	var sp1k []string
	rows, err := super.Query(ctx, "SELECT id::text FROM bench_ou WHERE pre >= 1 AND pre < 1001 ORDER BY pre")
	must(err)
	for rows.Next() {
		var id string
		must(rows.Scan(&id))
		sp1k = append(sp1k, id)
	}
	must(rows.Err())

	if phase == "single" {
		// Go Allows side: EXISTS over the closure for one record's unit.
		var outside string
		must(super.QueryRow(ctx, "SELECT id::text FROM bench_ou WHERE pre = 1011 + 1 + 5").Scan(&outside)) // a branch in region 1
		e.w("\n### Single-record check `%s`\n\nms per request: BEGIN, set_config, EXISTS, COMMIT.\n\n| U ancestor | rec unit | n | p50 | p95 | p99 |\n| --- | --- | --- | --- | --- | --- |\n", existsSQL)
		for _, s := range sets {
			for _, c := range []struct{ name, unit string }{{"inside (last descendant)", s.ids[len(s.ids)-1]}, {"outside", outside}} {
				u := []string{s.root}
				if s.root == sets[3].root && c.name == "outside" {
					continue // nothing is outside the root
				}
				p50, p95, p99, n := e.load(ctx, "exists "+s.label+" "+c.name, clients, secs, func(ctx context.Context, _ agent) (time.Duration, error) {
					return e.tx(ctx, func(tx pgx.Tx) error {
						var ok bool
						return tx.QueryRow(ctx, existsSQL, e.tenant, u, c.unit).Scan(&ok)
					})
				})
				e.w("| %s | %s | %d | %.2f | **%.2f** | %.2f |\n", s.label, c.name, n, p50, p95, p99)
			}
		}
		p50, p95, p99, n := e.load(ctx, "floor tx (BEGIN, set_config, COMMIT)", clients, secs, func(ctx context.Context, _ agent) (time.Duration, error) {
			return e.tx(ctx, func(pgx.Tx) error { return nil })
		})
		e.w("| floor: BEGIN, set_config, COMMIT | n/a | %d | %.2f | **%.2f** | %.2f |\n", n, p50, p95, p99)
		return
	}

	// List phases.
	switch phase {
	case "plain":
		e.exec(ctx, "DROP INDEX IF EXISTS records_bench_org_unit_updated")
		e.exec(ctx, "CREATE INDEX IF NOT EXISTS records_bench_org_unit ON records (tenant_id, org_unit_id)")
	case "r1":
		e.exec(ctx, "CREATE INDEX IF NOT EXISTS records_bench_org_unit ON records (tenant_id, org_unit_id)")
		e.exec(ctx, "CREATE INDEX IF NOT EXISTS records_bench_org_unit_updated ON records (tenant_id, org_unit_id, updated_at DESC, id DESC)")
	default:
		must(fmt.Errorf("SP1B_PHASE must be build, single, plain or r1"))
	}
	index := map[string]string{"plain": "plain", "r1": "plain + R1"}[phase]
	if os.Getenv("SP1B_PREP") != "0" {
		e.exec(ctx, "ANALYZE records")
		e.exec(ctx, "CHECKPOINT")
		e.exec(ctx, "SELECT pg_prewarm(c.oid) FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace WHERE n.nspname = 'public' AND c.relkind IN ('r', 'i')")
	}
	f50, f95 := e.floor(ctx, clients)
	gate := "PASS"
	if f50 > 3 {
		gate = "FAIL (not scored)"
	}
	fmt.Printf("floor SELECT 1 x%d: p50 %.2f p95 %.2f -> %s\n", clients, f50, f95, gate)
	e.w("\nTransport floor (`SELECT 1`, %d clients x 200): p50 %.2f, p95 %.2f ms. P3-36 gate (p50 <= 3 ms): **%s**.\n", clients, f50, f95, gate)
	e.w("\n| Rendering | |S| | index | n | p50 | p95 | p99 | Δp95 vs baseline | page plan / exec ms | region branch (page): plan | records read / removed | closure rows | count: region branch | foreign containers |\n| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |\n")

	type row struct {
		leaf, label string
		param       []string
	}
	rs := []row{{"", "baseline (4 branches)", nil}, {"D7", fmt.Sprintf("SP-1 set, %d units (first 1,000 of region 0)", len(sp1k)), sp1k}}
	for _, s := range sets {
		rs = append(rs, row{"C1", s.label, []string{s.root}}, row{"C2", s.label, []string{s.root}}, row{"B1", s.label, s.ids})
	}
	if os.Getenv("SP1B_PAIRED") == "1" {
		// Paired, 1 client: baseline and variant requests alternate in one
		// loop (same agent each pair), so host drift hits both equally.
		e.w("\n### Paired 1-client pass (%s): baseline and variant alternate, %d s per variant, warm-up 15 s\n\n| Rendering | set | index | pairs | base p50 | base p95 | variant p50 | variant p95 | Δp50 | **Δp95** | median of per-pair Δ |\n| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |\n", index, secs)
		bcs, bps := countSQL(""), pageSQL("")
		for _, r := range rs[1:] {
			name := strings.TrimSpace(r.leaf + " " + r.label)
			if !only.MatchString(name) {
				continue
			}
			cs, ps := countSQL(r.leaf), pageSQL(r.leaf)
			pair := func(i int) (time.Duration, time.Duration) {
				a := e.agents[i%len(e.agents)]
				var b, v time.Duration
				var err error
				if i%2 == 0 { // alternate which goes first
					b, err = e.request(ctx, bcs, bps, e.args(a, nil))
					must(err)
					v, err = e.request(ctx, cs, ps, e.args(a, r.param))
				} else {
					v, err = e.request(ctx, cs, ps, e.args(a, r.param))
					must(err)
					b, err = e.request(ctx, bcs, bps, e.args(a, nil))
				}
				must(err)
				return b, v
			}
			for i, warm := 0, time.Now().Add(15*time.Second); i < len(e.agents) && time.Now().Before(warm); i++ {
				pair(i)
			}
			var bs, vs, ds []time.Duration
			for i, stop := 0, time.Now().Add(time.Duration(secs)*time.Second); time.Now().Before(stop); i++ {
				b, v := pair(i)
				bs, vs, ds = append(bs, b), append(vs, v), append(ds, v-b)
			}
			b50, b95, v50, v95 := pct(bs, .5), pct(bs, .95), pct(vs, .5), pct(vs, .95)
			fmt.Printf("%s paired %-40s pairs=%5d base p50=%6.2f p95=%6.2f var p50=%7.2f p95=%7.2f dp95=%+7.2f\n", time.Now().UTC().Format("15:04:05"), name, len(bs), b50, b95, v50, v95, v95-b95)
			e.w("| %s | %s | %s | %d | %.2f | %.2f | %.2f | %.2f | %+.2f | **%+.2f** | %+.2f |\n", r.leaf, r.label, index, len(bs), b50, b95, v50, v95, v50-b50, v95-b95, pct(ds, .5))
		}
		f50, f95 = e.floor(ctx, clients)
		e.w("\nTransport floor at end: p50 %.2f, p95 %.2f ms.\n", f50, f95)
		fmt.Printf("floor at end: p50 %.2f p95 %.2f\n", f50, f95)
		return
	}
	var baseP95 float64
	for _, r := range rs {
		name := strings.TrimSpace(r.leaf + " " + r.label)
		if r.leaf != "" && !only.MatchString(name) {
			continue
		}
		cs, ps := countSQL(r.leaf), pageSQL(r.leaf)
		p50, p95, p99, n := e.load(ctx, name+" / "+index, clients, secs, func(ctx context.Context, a agent) (time.Duration, error) {
			return e.request(ctx, cs, ps, e.args(a, r.param))
		})
		fc := "none"
		if f := foreign(); f != "" {
			fc = strconv.Itoa(len(strings.Fields(f)))
		}
		delta := "n/a"
		if r.leaf == "" {
			baseP95 = p95
		} else {
			delta = fmt.Sprintf("%+.2f", p95-baseP95)
		}
		plan, rr, cl, cnt := "n/a", "n/a", "n/a", "n/a"
		pp := "n/a"
		if os.Getenv("SP1B_EXPLAIN") != "0" {
			pi := e.explain(ctx, name+" / "+index+": page", ps, e.args(e.agents[0], r.param))
			ci := e.explain(ctx, name+" / "+index+": count", cs, countArgs(e.args(e.agents[0], r.param)))
			pp = fmt.Sprintf("%.1f / %.1f", pi.planMS, pi.execMS)
			if pi.region != nil {
				plan = strings.Join(pi.region.shape, " > ")
				rr = fmt.Sprintf("%.0f / %.0f", pi.region.read, pi.region.removed)
				cl = fmt.Sprintf("%.0f", pi.region.closure)
			}
			if ci.region != nil {
				cnt = "executed"
				if !ci.region.executed {
					cnt = "never executed"
				}
				cnt += fmt.Sprintf(" (plan %.1f ms)", ci.planMS)
			}
		}
		lbl := r.leaf
		if lbl == "" {
			lbl = "baseline"
		}
		e.w("| %s | %s | %s | %d | %.2f | **%.2f** | %.2f | %s | %s | %s | %s | %s | %s | %s |\n", lbl, r.label, index, n, p50, p95, p99, delta, pp, plan, rr, cl, cnt, fc)
	}
	f50, f95 = e.floor(ctx, clients)
	e.w("\nTransport floor at end: p50 %.2f, p95 %.2f ms.\n", f50, f95)
	fmt.Printf("floor at end: p50 %.2f p95 %.2f\n", f50, f95)
	if e.plans.Len() > 0 {
		e.w("\n### Plans (%s)\n%s", index, e.plans.String())
	}
}

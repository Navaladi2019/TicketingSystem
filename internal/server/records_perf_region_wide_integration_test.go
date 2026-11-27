//go:build integration && perf

// TICK-1105 (ADR-010-A3 A3-R2, ADR-010-A4 A4-10): the wide-tenant region
// cell. One extra tenant, "wide", holds 250,000 records of about 1 KB in a
// scattered heap: 33,366 of type bench_wide (N̂_e = 33,366 = ⌊α·10,111⌋, the
// top of R's range) and 216,634 of the type ticket, inserted in md5 order so
// bench_wide's rows are spread over the whole heap. Its org tree is the SP-1
// tree (10,111 units) plus one unit outside it ("elsewhere").
//
// The personas sit at the tree's root, so |S| = 10,111. Two cells, both
// personas, scored against 150 ms like every region row:
//   - m = 0: every bench_wide record is in "elsewhere", so none is under S
//     and R's count reads the type to its end;
//   - m = 500: 500 random bench_wide records are then moved to random
//     branches of the tree.
//
// Each cell's captured EXPLAIN (perfEnv.capture) is the count plan A4-10 asks
// for: it must be an Index Only Scan of records_list_updated. The region
// report lists each count statement's records scan beside the row.
//
// The rows run after every TICK-1037 row and touch only the wide tenant, so
// they change none of the earlier rows' data. The method is A2-11's: one
// client, each scored request paired with the same agent without the
// regional group.

package server_test

import (
	"crypto/md5"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

const (
	perfWideRecords = 250000
	perfWideType    = 33366 // bench_wide's records: N̂_e
	perfWideUsers   = 500
	perfWideActors  = 40
	perfWideMoved   = 500 // the m = 500 cell
)

// perfWide is the wide cell's state once its setup ran.
type perfWide struct {
	tenant   etTenant
	regional uuid.UUID
	agents   []string // base tokens; "r" + token is the scored agent
	only     identity.Principal
	defs     map[string]uuid.UUID
	root     uuid.UUID
	moved    bool   // the m = 500 move is done
	current  string // the cell whose set line is written
}

type perfWideCell struct {
	name string
	m    int
}

func perfWideCells() []perfWideCell {
	return []perfWideCell{
		{"wide 250k, bench_wide at root (|S| 10,111), m=0", 0},
		{"wide 250k, bench_wide at root (|S| 10,111), m=500", perfWideMoved},
	}
}

func (e *perfEnv) wideDefs(t *testing.T) []perfDef {
	t.Helper()
	var out []perfDef
	for _, c := range perfWideCells() {
		for _, persona := range []string{"agents", "count-only"} {
			out = append(out, perfDef{"region: " + c.name + ", " + persona, "list", "wide", func() []perfReq { return e.wideReqs(t, persona) }, 1})
		}
	}
	return out
}

func (e *perfEnv) wideExtras(t *testing.T) map[string]perfExtra {
	t.Helper()
	m := map[string]perfExtra{}
	for _, c := range perfWideCells() {
		for _, persona := range []string{"agents", "count-only"} {
			m["region: "+c.name+", "+persona] = perfExtra{before: func() { e.wideBefore(t, c) }}
		}
	}
	return m
}

// wideReqs builds the data on first use (the requests need the tokens), then
// interleaves each scored request with its base, scored first.
func (e *perfEnv) wideReqs(t *testing.T, persona string) []perfReq {
	t.Helper()
	if e.wide == nil {
		e.wideSetup(t)
	}
	path := "/api/v1/records/bench_wide?sort=-updated_at"
	out := make([]perfReq, 0, 2*len(e.wide.agents))
	for _, tok := range e.wide.agents {
		scored := "r" + tok
		if persona == "count-only" {
			scored = "wregion-only"
		}
		out = append(out, perfReq{scored, path, ""}, perfReq{tok, path, ""})
	}
	return out
}

// wideBefore moves the m = 500 records in once, before the first m = 500 row,
// and writes the cell's set line.
func (e *perfEnv) wideBefore(t *testing.T, c perfWideCell) {
	t.Helper()
	if e.wide == nil {
		e.wideSetup(t)
	}
	if c.m > 0 && !e.wide.moved {
		q := strings.NewReplacer("$T", "'"+e.wide.tenant.id.String()+"'::uuid", "$W", "'"+e.wide.defs["bench_wide"].String()+"'::uuid").Replace(perfWideMoveSQL)
		if _, err := e.super.Exec(t.Context(), q); err != nil {
			t.Fatalf("wide m=500 move: %v", err)
		}
		for _, s := range []string{"VACUUM (ANALYZE) records", "CHECKPOINT"} {
			e.exec(t, s)
		}
		e.wide.moved = true
	}
	if e.wide.current != c.name {
		e.wide.current = c.name
		e.wideSetLine(t, c)
	}
}

func (e *perfEnv) wideSetLine(t *testing.T, c perfWideCell) {
	t.Helper()
	def := e.wide.defs["bench_wide"]
	var units, m, ne int64
	if err := e.super.QueryRow(t.Context(), `SELECT
		(SELECT count(*) FROM org_unit_closure WHERE tenant_id = $1 AND ancestor_id = $2),
		(SELECT count(*) FROM records WHERE tenant_id = $1 AND entity_definition_id = $3 AND org_unit_id IN
		    (SELECT descendant_id FROM org_unit_closure WHERE tenant_id = $1 AND ancestor_id = $2)),
		(SELECT live_records FROM entity_record_estimates WHERE tenant_id = $1 AND entity_definition_id = $3)`,
		e.wide.tenant.id, e.wide.root, def).Scan(&units, &m, &ne); err != nil {
		t.Fatalf("wide set %s: %v", c.name, err)
	}
	if m != int64(c.m) {
		t.Fatalf("wide set %s: m = %d, want %d", c.name, m, c.m)
	}
	k := authz.KStar(perfRegionPage, ne, true)
	line := fmt.Sprintf("| %s | bench_wide | %d | %d | %d | %d | see the shape column of the rows |", mdCell(c.name), units, m, ne, k)
	t.Logf("REGION SET %s", line)
	if e.region == nil {
		e.region = &perfRegion{defs: map[string]uuid.UUID{}, units: map[int]uuid.UUID{}}
	}
	fmt.Fprintln(&e.region.report, line)
}

// perfWideSetupSQL builds the wide tenant's tree, users and records. $T is
// the tenant, $W bench_wide's definition, $F the ticket definition.
const perfWideSetupSQL = `
SET max_parallel_workers_per_gather = 0;
SELECT setseed(0.1105);
SET session_replication_role = replica;
DROP TABLE IF EXISTS bench_ou_w;
CREATE TABLE bench_ou_w (pre int PRIMARY KEY, id uuid NOT NULL, parent uuid, depth int NOT NULL);
INSERT INTO bench_ou_w (pre, id, parent, depth)
SELECT 0, md5('w/ou/root')::uuid, NULL::uuid, 0
UNION ALL
SELECT 1 + r * 1011, md5('w/ou/' || r)::uuid, md5('w/ou/root')::uuid, 1 FROM generate_series(0, 9) r
UNION ALL
SELECT 1 + r * 1011 + 1 + d * 101, md5('w/ou/' || r || '/' || d)::uuid, md5('w/ou/' || r)::uuid, 2
FROM generate_series(0, 9) r, generate_series(0, 9) d
UNION ALL
SELECT 1 + r * 1011 + 1 + d * 101 + 1 + a * 10, md5('w/ou/' || r || '/' || d || '/' || a)::uuid, md5('w/ou/' || r || '/' || d)::uuid, 3
FROM generate_series(0, 9) r, generate_series(0, 9) d, generate_series(0, 9) a
UNION ALL
SELECT 1 + r * 1011 + 1 + d * 101 + 1 + a * 10 + 1 + b, md5('w/ou/' || r || '/' || d || '/' || a || '/' || b)::uuid,
       md5('w/ou/' || r || '/' || d || '/' || a)::uuid, 4
FROM generate_series(0, 9) r, generate_series(0, 9) d, generate_series(0, 9) a, generate_series(0, 8) b;
INSERT INTO org_units (tenant_id, id, parent_id, keycloak_group_id, name)
SELECT $T, o.id, o.parent, 'wide-ou-' || o.pre, 'ou-' || o.pre FROM bench_ou_w o ORDER BY o.pre;
INSERT INTO org_unit_closure (tenant_id, ancestor_id, descendant_id)
SELECT $T, a.id, d.id FROM bench_ou_w a JOIN bench_ou_w d
  ON d.pre >= a.pre AND d.pre < a.pre + (ARRAY[10111, 1011, 101, 10, 1])[a.depth + 1];
-- "elsewhere": a unit outside the tree, where the m = 0 cell's records are.
INSERT INTO org_units (tenant_id, id, parent_id, keycloak_group_id, name)
VALUES ($T, md5('w/ou/elsewhere')::uuid, NULL, 'wide-ou-elsewhere', 'elsewhere');
INSERT INTO org_unit_closure (tenant_id, ancestor_id, descendant_id)
VALUES ($T, md5('w/ou/elsewhere')::uuid, md5('w/ou/elsewhere')::uuid);
DROP TABLE bench_ou_w;
INSERT INTO principals (tenant_id, id, keycloak_user_id, email, enabled, federated_links, synced_at)
SELECT $T, md5('wu/' || u)::uuid, 'perf-wu-' || u, 'u' || u || '@wide.test', true, '[]', now() FROM generate_series(0, ` + "499" + `) u;
INSERT INTO groups (tenant_id, id, keycloak_group_id, path)
SELECT $T, md5('wg/' || g)::uuid, 'perf-wg-' || g, '/t/' || $T || '/g' || g FROM generate_series(0, 49) g;
INSERT INTO group_members (tenant_id, group_id, principal_id)
SELECT $T, md5('wg/' || ((u + k) % 50))::uuid, md5('wu/' || u)::uuid
FROM generate_series(0, 499) u, unnest(ARRAY[0, 17, 34]) k;
-- 250,000 records in md5 order: bench_wide's 33,366 are spread over the heap.
-- A ticket is in a random branch of the tree (branch L is md5 of its path);
-- every bench_wide record starts in "elsewhere".
INSERT INTO records (tenant_id, id, version, entity_definition_id, schema_version, number, state,
    requester_id, assignee_id, assignment_group_id, created_by, created_at, updated_at, data, causation_depth, org_unit_id)
SELECT $T, md5('wr/' || x.n)::uuid, 1, CASE WHEN x.wide THEN $W ELSE $F END, 1,
       CASE WHEN x.wide THEN 'WIDE' ELSE 'TKT' END || lpad(x.n::text, 7, '0'),
       CASE WHEN x.r1 < 0.15 THEN (ARRAY['new', 'in_progress', 'on_hold'])[1 + floor(x.r2 * 3)::int] ELSE 'closed' END,
       md5('wu/' || floor(500 * x.r3 * x.r3)::int)::uuid,
       CASE WHEN x.r4 < 0.9 THEN md5('wu/' || floor(500 * x.r5)::int)::uuid END,
       md5('wg/' || floor(50 * x.r6)::int)::uuid,
       md5('wu/' || floor(500 * x.r3 * x.r3)::int)::uuid,
       x.upd - x.r7 * interval '30 days', x.upd,
       jsonb_build_object(
           'priority', CASE WHEN x.r8 < 0.05 THEN '1' WHEN x.r8 < 0.20 THEN '2' WHEN x.r8 < 0.70 THEN '3'
                            WHEN x.r8 < 0.90 THEN '4' ELSE '5' END,
           'category', 'category-' || floor(x.r9 * 20)::int,
           'impact', 1 + floor(x.r2 * 3)::int,
           'short_description', 'Record ' || x.n || ' ' || md5(x.n::text),
           'description', repeat(md5('w:' || x.n), 27),
           'tags', jsonb_build_array('t' || floor(x.r5 * 50)::int, 't' || floor(x.r6 * 50)::int),
           'site', 's' || floor(x.r9 * 200)::int),
       0,
       CASE WHEN x.wide THEN md5('w/ou/elsewhere')::uuid
            ELSE md5('w/ou/' || (x.l / 900) || '/' || (x.l / 90 % 10) || '/' || (x.l / 9 % 10) || '/' || (x.l % 9))::uuid END
FROM (
    SELECT n, n <= ` + "33366" + ` AS wide, floor(9000 * random())::int AS l,
           random() AS r1, random() AS r2, random() AS r3, random() AS r4, random() AS r5,
           random() AS r6, random() AS r7, random() AS r8, random() AS r9,
           timestamptz '2026-10-01 00:00:00+00' - power(random(), 3) * interval '365 days' AS upd
    FROM generate_series(1, ` + "250000" + `) n
    ORDER BY md5('wr/' || n)
) x;
INSERT INTO record_counters (tenant_id, entity_definition_id, next)
VALUES ($T, $W, ` + "33367" + `), ($T, $F, ` + "250001" + `) ON CONFLICT DO NOTHING;
SET session_replication_role = origin;
RESET max_parallel_workers_per_gather;
`

// perfWideMoveSQL moves 500 random bench_wide records into random branches
// of the tree: the m = 500 cell.
const perfWideMoveSQL = `
SET session_replication_role = replica;
UPDATE records r SET org_unit_id = md5('w/ou/' || (s.l / 900) || '/' || (s.l / 90 % 10) || '/' || (s.l / 9 % 10) || '/' || (s.l % 9))::uuid
FROM (SELECT id, abs(hashtext(id::text || 'm500')) % 9000 AS l FROM records
      WHERE tenant_id = $T AND entity_definition_id = $W ORDER BY md5(id::text || 'm500') LIMIT ` + "500" + `) s
WHERE r.tenant_id = $T AND r.id = s.id;
SET session_replication_role = origin;
`

// wideSetup creates the wide tenant, its two types through the API, its data
// and personas, runs one estimates cycle and publishes the region ACL.
func (e *perfEnv) wideSetup(t *testing.T) {
	t.Helper()
	ctx := t.Context()
	start := time.Now()
	tn := etSeedTenant(t, e.super, "wide")
	e.tokens["admin-wide"] = tn.admin
	w := &perfWide{tenant: tn, defs: map[string]uuid.UUID{}, root: md5UUID("w/ou/root")}
	agents := uuid.NewMD5(uuid.NameSpaceOID, []byte("wide/agents"))
	w.regional = uuid.NewMD5(uuid.NameSpaceOID, []byte("wide/regional"))
	for _, g := range []struct {
		id  uuid.UUID
		kc  string
		sub string
	}{{agents, "perf-wide-agents", "/agents"}, {w.regional, "perf-wide-regional", "/regional"}} {
		e.exec(t, "INSERT INTO groups (tenant_id, id, keycloak_group_id, path) VALUES ($1, $2, $3, $4)", tn.id, g.id, g.kc, "/t/"+tn.id.String()+g.sub)
	}
	for _, k := range []string{"ticket", "bench_wide"} {
		body := perfTypeBody()
		if k != "ticket" {
			body = strings.Replace(body, `"key": "ticket", "name": "Ticket", "number_prefix": "TKT"`, `"key": "bench_wide", "name": "bench_wide", "number_prefix": "WIDE"`, 1)
		}
		if rec := e.serve(ctx, http.MethodPost, "/api/v1/entity-types", "admin-wide", "perf-wide-type-"+k, body); rec.Code != http.StatusCreated {
			t.Fatalf("create wide type %s = %d: %s", k, rec.Code, rec.Body)
		}
		var id uuid.UUID
		if err := e.super.QueryRow(ctx, "SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2", tn.id, k).Scan(&id); err != nil {
			t.Fatalf("wide definition %s: %v", k, err)
		}
		w.defs[k] = id
	}
	q := strings.NewReplacer("$T", "'"+tn.id.String()+"'::uuid", "$W", "'"+w.defs["bench_wide"].String()+"'::uuid",
		"$F", "'"+w.defs["ticket"].String()+"'::uuid").Replace(perfWideSetupSQL)
	if _, err := e.super.Exec(ctx, q); err != nil {
		t.Fatalf("wide setup: %v", err)
	}
	// Personas: agents in the agents group and 3 assignment groups (as
	// P3-01's), the scored copy also in the regional group; a count-only
	// principal in the regional group alone. All at the tree's root.
	for n := 1; n <= perfWideActors; n++ {
		u := (n*12 + 5) % perfWideUsers
		g := []uuid.UUID{agents}
		for _, k := range []int{0, 17, 34} {
			g = append(g, md5UUID("wg/"+strconv.Itoa((u+k)%50)))
		}
		p := identity.Principal{TenantID: tn.id, PrincipalID: md5UUID("wu/" + strconv.Itoa(u)), Groups: g}
		tok := "wagent-" + strconv.Itoa(n)
		e.tokens[tok], e.agentGroups[tok] = p, g[1:]
		e.tokens["r"+tok] = identity.Principal{TenantID: tn.id, PrincipalID: p.PrincipalID, Groups: append(append([]uuid.UUID{}, g...), w.regional)}
		e.agentGroups["r"+tok] = g[1:]
		e.pairBase[tok] = true
		w.agents = append(w.agents, tok)
		e.exec(t, "INSERT INTO group_members (tenant_id, group_id, principal_id) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING", tn.id, agents, p.PrincipalID)
	}
	w.only = identity.Principal{TenantID: tn.id, PrincipalID: uuid.NewMD5(uuid.NameSpaceOID, []byte("wide/region-only")), Groups: []uuid.UUID{w.regional}}
	e.exec(t, `INSERT INTO principals (tenant_id, id, keycloak_user_id, email, enabled, federated_links, synced_at)
		VALUES ($1, $2, 'perf-wide-region-only', 'region-only@wide.test', true, '[]', now())`, tn.id, w.only.PrincipalID)
	e.exec(t, "INSERT INTO group_members (tenant_id, group_id, principal_id) VALUES ($1, $2, $3)", tn.id, w.regional, w.only.PrincipalID)
	e.tokens["wregion-only"] = w.only
	principals := []uuid.UUID{w.only.PrincipalID}
	for _, tok := range w.agents {
		principals = append(principals, e.tokens[tok].PrincipalID)
	}
	e.exec(t, `INSERT INTO org_unit_members (tenant_id, org_unit_id, principal_id)
		SELECT $1, $2, p FROM unnest($3::uuid[]) p ON CONFLICT DO NOTHING`, tn.id, w.root, principals)
	for _, s := range []string{"VACUUM (ANALYZE) records", "ANALYZE org_units", "ANALYZE org_unit_closure", "ANALYZE org_unit_members",
		"ANALYZE principals", "ANALYZE groups", "ANALYZE group_members",
		`SELECT pg_prewarm(c.oid) FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace WHERE n.nspname = 'public' AND c.relkind IN ('r', 'i')`,
		"CHECKPOINT"} {
		e.exec(t, s)
	}
	e.exec(t, `INSERT INTO tenant_identity_binding (tenant_id, issuer, org_id) VALUES ($1, 'https://kc.perf.test/realms/nava', 'perf-org-wide') ON CONFLICT DO NOTHING`, tn.id)
	e.estimatesCycle(t)
	var ne int64
	if err := e.super.QueryRow(ctx, "SELECT live_records FROM entity_record_estimates WHERE tenant_id = $1 AND entity_definition_id = $2",
		tn.id, w.defs["bench_wide"]).Scan(&ne); err != nil {
		t.Fatalf("wide estimate: %v", err)
	}
	if ne != perfWideType {
		t.Fatalf("wide estimate: N̂_e = %d, want %d", ne, perfWideType)
	}
	d := perfRegionDoc(agents, tn.adminGroup, w.regional)
	for k := range w.defs {
		if err := tenancy.InTenantTx(ctx, e.pool, tn.id, func(tx pgx.Tx) error {
			_, err := perfPublishKeyDoc(ctx, tx, k, d)
			return err
		}); err != nil {
			t.Fatalf("publish wide region ACL on %s: %v", k, err)
		}
	}
	e.wide = w
	t.Logf("REGION wide setup (250,000 records, bench_wide N̂_e %d, tree, personas, refresh, ACLs): %s", ne, time.Since(start).Round(time.Second))
}

// md5UUID is PostgreSQL's md5(s)::uuid.
func md5UUID(s string) uuid.UUID { return uuid.UUID(md5.Sum([]byte(s))) }

// perfPlanNode is a plan line's node: its "->" column, its text and its
// actual rows and loops.
var (
	perfPlanActual  = regexp.MustCompile(`actual time=[\d.]+\.\.[\d.]+ rows=(\d+) loops=(\d+)`)
	perfPlanRemoved = regexp.MustCompile(`Rows Removed by Filter: (\d+)`)
	perfPlanExec    = regexp.MustCompile(`Execution Time: ([\d.]+) ms`)
	perfRecordsScan = regexp.MustCompile(`((?:Parallel )?(?:Index Only Scan|Index Scan|Seq Scan|Bitmap Heap Scan)(?: using \w+)? on records)\b`)
)

// perfArm is one scan of records in a count plan (A4-10 ruling, items 1-2).
type perfArm struct {
	class      string // e.g. "Index Scan using records_number_key on records"
	tenantCond bool   // the Index Cond (or Recheck Cond) starts with tenant_id
	rowsRead   int64  // (actual rows + rows removed by filter) x loops
	regional   bool   // the arm carries the region term
}

// planArms finds the records scans of an EXPLAIN (ANALYZE) text. A node's
// detail block is every following line indented deeper than its "->", up
// to its first SubPlan or child node.
func planArms(plan string) []perfArm {
	lines := strings.Split(plan, "\n")
	var out []perfArm
	for i, l := range lines {
		m := perfRecordsScan.FindStringSubmatch(l)
		if m == nil {
			continue
		}
		col := strings.Index(l, "->")
		a := perfArm{class: m[1]}
		var rows, loops, removed int64
		if am := perfPlanActual.FindStringSubmatch(l); am != nil {
			rows, _ = strconv.ParseInt(am[1], 10, 64)
			loops, _ = strconv.ParseInt(am[2], 10, 64)
		}
		for _, d := range lines[i+1:] {
			t := strings.TrimSpace(d)
			ind := len(d) - len(strings.TrimLeft(d, " "))
			if ind <= col || strings.HasPrefix(t, "->") || strings.HasPrefix(t, "SubPlan") || strings.HasPrefix(t, "InitPlan") {
				break
			}
			switch {
			case strings.HasPrefix(t, "Index Cond:"), strings.HasPrefix(t, "Recheck Cond:"):
				c := strings.TrimLeft(strings.TrimSpace(t[strings.Index(t, ":")+1:]), "(")
				a.tenantCond = strings.HasPrefix(c, "tenant_id ")
				if strings.Contains(t, "org_unit_id") {
					a.regional = true
				}
			case strings.HasPrefix(t, "Filter:"):
				if strings.Contains(t, "hashed SubPlan") || strings.Contains(t, "org_unit_id") {
					a.regional = true
				}
			}
			if rm := perfPlanRemoved.FindStringSubmatch(t); rm != nil && removed == 0 {
				removed, _ = strconv.ParseInt(rm[1], 10, 64)
			}
		}
		a.rowsRead = (rows + removed) * loops
		out = append(out, a)
	}
	return out
}

// isListCount reports whether sql is the list's count statement, which ends
// "LIMIT 1001) c"; the probe's count ends "acl_p" (A2) or is the folded probe.
func isListCount(sql string) bool {
	s := strings.TrimSpace(sql)
	return strings.HasPrefix(s, "SELECT count(*)") && strings.HasSuffix(s, ") c")
}

// countPlan summarises the list count of a captured request for the A4-10
// ruling: the count's execution time, whether any arm is an Index Only Scan
// of records_list_updated, and for the regional arm its plan class, whether
// its index condition starts with tenant_id and the rows it read against
// N̂_e.
func countPlan(r *perfResult, estimate int64) string {
	for i, s := range r.stmtSQL {
		if i >= len(r.explains) || r.explains[i] == "" || !isListCount(s.SQL) {
			continue
		}
		return summariseCountPlan(r.explains[i], estimate, i+1)
	}
	return "no list count statement (A2-4: the count is skipped)"
}

func summariseCountPlan(plan string, estimate int64, stmt int) string {
	exec := "?"
	if m := perfPlanExec.FindStringSubmatch(plan); m != nil {
		exec = m[1]
	}
	ios := "no"
	if strings.Contains(plan, "Index Only Scan using records_list_updated") {
		ios = "yes"
	}
	var arms []string
	for _, a := range planArms(plan) {
		if !a.regional {
			continue
		}
		tc := "no"
		if a.tenantCond {
			tc = "yes"
		}
		arms = append(arms, fmt.Sprintf("%s; Index Cond starts with tenant_id: %s; rows read %d (N̂_e %d)", a.class, tc, a.rowsRead, estimate))
	}
	if len(arms) == 0 {
		arms = []string{"no records scan carries the region term"}
	}
	return fmt.Sprintf("stmt %d, count %s ms; IOS of records_list_updated: %s; regional arm: %s", stmt, exec, ios, strings.Join(arms, " / "))
}

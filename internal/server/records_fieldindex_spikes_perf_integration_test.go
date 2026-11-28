//go:build integration && perf

// ADR-006-A1 spikes on the TICK-992 harness (TestFieldIndexPerf800k: 800k
// seed, 400k type, 8 clients, NAVA_PERF_LIST_SECONDS per cell, EXPLAIN
// (ANALYZE, BUFFERS) of each cell's captured statements as nava_app).
//
// Phase "edges", TICK-1009 (A1-7):
//   - S1, band edge: 5%, 10%, 20% for num lt (score lt 500 / 1000 / 2000)
//     and text eq (tier v5 / v10 / v20), with the narrowing ACL (the P3-36
//     agents, 6% visible) and without (the admin): 12 cells.
//   - S2, ACL breadth: 2% num lt and text eq for the agents seeing 6% (the
//     control), 25% and 50% of the type: 6 cells.
//
// Phase "arms", TICK-1039 (A1-6): the num field pk on the large type, one
// side row per record, in three arms through the real handler (ACL, RLS):
//   - "D7": today's single EXISTS (the control);
//   - "(a)": the adaptive shape. A capped pre-count of the value's side rows
//     (LIMIT spK+1) picks it: at most spK matches drive from the side index
//     (D7's EXISTS); more walk the records index in list order and probe
//     rfv_num_pkey per record. The probe is forced by the shape, OFFSET 0 in
//     the EXISTS (no semi-join pull-up), not by enable_* or a hint. The count
//     uses the same term, so with an ACL it probes per group-visible record up
//     to the cap;
//   - "(b)": a promoted column, scratch DDL only: records.spike_pk with an
//     index on (tenant_id, entity_definition_id, spike_pk, updated_at);
//     "(b)+group" swaps it for one that carries assignment_group_id before
//     updated_at. Dropped at the end of the phase.
//
// spK is fixed before any run: sqrt(51 x 400,000) ~ 4,500, where a 51-row
// page walk over independent matches (51 / s probes) costs what driving from
// the side index does (s x 400k side rows).
//
// Agents. The P3-36 agents (agent-N) are unchanged: the agents group plus
// assignment groups u+{0,17,34} of 50 (6%). a25-N and a50-N are new tokens on
// agent-N's principal with 13 and 25 of the 50 groups, nested (6% within 26%
// within 50%), so a broader agent sees a superset. 25% is not a multiple of a
// group's 2%: a25 sees 26%. TICK-1039's a6 and a50 are agent-1 and a50-1.

package server_test

import (
	"cmp"
	"context"
	"crypto/md5"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/fieldindex"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/server"
)

// The group offsets of each breadth, nested: fiK6 within fiK25 within fiK50.
func fiK6() []int  { return []int{0, 17, 34} }
func fiK25() []int { return []int{0, 4, 8, 12, 16, 17, 20, 24, 28, 32, 34, 36, 40} }
func fiK50() []int {
	return []int{0, 2, 4, 6, 8, 10, 12, 14, 16, 17, 18, 20, 22, 24, 26, 28, 30, 32, 34, 36, 38, 40, 42, 44, 46}
}

// spK is the adaptive arm's pre-count cap (see the header).
const spK = 4500

// broadAgents registers a25-N and a50-N for every agent-N.
func (e *fiEnv) broadAgents(t *testing.T) {
	t.Helper()
	if _, done := e.tokens["a50-1"]; done {
		return
	}
	var agents uuid.UUID
	if err := e.super.QueryRow(t.Context(), `SELECT agents FROM perf_tenant WHERE tn = 0`).Scan(&agents); err != nil {
		t.Fatal(err)
	}
	gid := func(g int) uuid.UUID { return uuid.UUID(md5.Sum([]byte(fmt.Sprintf("g0/%d", g)))) }
	for n := 1; n <= perfLargeActors; n++ {
		base := e.tokens["agent-"+strconv.Itoa(n)]
		u := (n*24 + 7) % perfLargeUsers
		// The derived groups must be the agent's own: the offsets nest.
		for i, k := range fiK6() {
			if base.Groups[i+1] != gid((u+k)%perfGroups) {
				t.Fatalf("agent-%d group %d is not g0/%d", n, i, (u+k)%perfGroups)
			}
		}
		for _, b := range []struct {
			pre string
			ks  []int
		}{{"a25-", fiK25()}, {"a50-", fiK50()}} {
			g := []uuid.UUID{agents}
			for _, k := range b.ks {
				g = append(g, gid((u+k)%perfGroups))
			}
			tok := b.pre + strconv.Itoa(n)
			e.tokens[tok] = identity.Principal{TenantID: base.TenantID, PrincipalID: base.PrincipalID, Groups: g}
			e.agentGroups[tok] = g[1:]
		}
	}
	e.w("")
	e.w("- breadth agents (agent-1's principal), records of the large type each sees through any ACL branch:")
	for _, tok := range []string{"agent-1", "a25-1", "a50-1"} {
		n := e.visibleCount(t, tok)
		e.w("  - %s: %d groups, %d records (%.1f%%)", tok, len(e.tokens[tok].Groups)-1, n, 100*float64(n)/perfLargeRecords)
	}
}

// spVisible is the ACL's visible set of principal $3 with groups $4, written
// out on records r (the four branches the compiled read has: group,
// requester, assignee, share). Superuser only: a fixture check, not the ACL.
const spVisible = `(r.assignment_group_id = ANY($4::uuid[]) OR r.requester_id = $3 OR r.assignee_id = $3
	OR EXISTS (SELECT 1 FROM record_shares s WHERE s.tenant_id = r.tenant_id AND s.record_id = r.id
		AND ((s.principal_kind = 'user' AND s.principal_id = $3) OR (s.principal_kind = 'group' AND s.principal_id = ANY($4::uuid[])))))`

func (e *fiEnv) visibleCount(t *testing.T, tok string) int64 {
	t.Helper()
	p := e.tokens[tok]
	var n int64
	if err := e.super.QueryRow(t.Context(), `SELECT count(*) FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND `+spVisible,
		e.large.id, e.largeDef, p.PrincipalID, p.Groups).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// ---- one cell ---------------------------------------------------------------

type spCell struct {
	label            string
	p95, p50, floor  float64
	n                int
	errs, timeouts   int64
	pre, page, count float64 // single-run server time (planning + execution), ms
	pageShape        string
	countShape       string
}

var (
	spScan = regexp.MustCompile(`(Index Only Scan|Index Scan Backward|Index Scan|Bitmap Heap Scan|Seq Scan)(?: using (\S+))? on (\S+)[^\n]*?actual time=[0-9.]+\.\.[0-9.]+ rows=(\d+) loops=(\d+)`)
	spBuf  = regexp.MustCompile(`Buffers: shared hit=(\d+)(?: read=(\d+))?`)
)

// spShape summarises a plan: its join kind and the three relations it read
// the most rows from (rows x loops), and the top node's shared buffers.
func spShape(plan string) string {
	if plan == "" {
		return "-"
	}
	kind := "join"
	switch {
	case strings.Contains(plan, "SubPlan"):
		kind = "probe"
	case strings.Contains(plan, "Hash Semi Join") || strings.Contains(plan, "Hash Join"):
		kind = "hash"
	case strings.Contains(plan, "Nested Loop"):
		kind = "nested loop"
	}
	if strings.Contains(plan, "temp written") {
		kind += "+spill"
	}
	rows := map[string]int64{}
	var probes int64
	for _, m := range spScan.FindAllStringSubmatch(plan, -1) {
		name := m[2]
		if name == "" {
			name = m[3]
		}
		r, _ := strconv.ParseInt(m[4], 10, 64)
		l, _ := strconv.ParseInt(m[5], 10, 64)
		rows[name] += r * l
		if l > 1 {
			probes += l
		}
	}
	names := make([]string, 0, len(rows))
	for k := range rows {
		names = append(names, k)
	}
	slices.SortFunc(names, func(a, b string) int { return cmp.Or(cmp.Compare(rows[b], rows[a]), cmp.Compare(a, b)) })
	var parts []string
	for i, k := range names {
		if i == 3 {
			break
		}
		parts = append(parts, fmt.Sprintf("%s %d", k, rows[k]))
	}
	buf := ""
	if m := spBuf.FindStringSubmatch(plan); m != nil {
		buf = ", buffers " + m[1]
		if m[2] != "" {
			buf += "+" + m[2] + " read"
		}
	}
	pr := ""
	if probes > 0 {
		pr = fmt.Sprintf(", %d inner loops", probes)
	}
	return kind + ": " + strings.Join(parts, ", ") + pr + buf
}

// cell runs one list scenario for d after a transport-floor reading, and
// splits its captured statements into pre-count, page and count.
func (e *fiEnv) cell(t *testing.T, clients int, d time.Duration, label string, tokens []string, q string, plans *[]string) spCell {
	t.Helper()
	fl := e.transportFloor(t, clients)
	sc := perfScenario{name: label, kind: "data", tenant: "large", reqs: listReqs(tokens, q)}
	r := e.run(t, sc, clients, d)
	c := spCell{label: label, p95: pct(r.lat, .95), p50: pct(r.lat, .5), floor: pct(fl, .5), n: len(r.lat), errs: r.errs, timeouts: r.timeouts}
	c.pageShape, c.countShape = "-", "-"
	for i, s := range r.stmtSQL {
		if i >= len(r.explains) || r.explains[i] == "" {
			continue
		}
		sql, plan := s.SQL, r.explains[i]
		ms := 0.0
		if m := fiPlanning.FindStringSubmatch(plan); m != nil {
			p, _ := strconv.ParseFloat(m[1], 64)
			x, _ := strconv.ParseFloat(fiExecution.FindStringSubmatch(plan)[1], 64)
			ms = p + x
		}
		switch {
		case strings.Contains(sql, "LIMIT "+strconv.Itoa(spK+1)):
			c.pre += ms
		case !strings.Contains(sql, "rfv_") && !strings.Contains(sql, "spike_pk"):
			continue
		case strings.Contains(sql, "count(*)"):
			c.count += ms
			c.countShape = spShape(plan)
		default:
			c.page += ms
			c.pageShape = spShape(plan)
		}
		*plans = append(*plans, "### "+label+"\n\n```sql\n"+strings.TrimSpace(sql)+"\n```\n\n```\n"+plan+"\n```")
	}
	return c
}

// verdict scores p95 against target: within 10% of the line either way is
// provisional (A1-7, TICK-1039 scoring).
func spVerdict(p95, target float64) string {
	v := "pass"
	if p95 > target {
		v = "**miss**"
	}
	if p95 >= 0.9*target && p95 <= 1.1*target {
		v += " (provisional)"
	}
	return v
}

func (c spCell) row(target float64) string {
	return fmt.Sprintf("| %s | **%.1f** | %.1f | %d | %s | %.1f | %.1f / %.1f / %.1f | %s | %s | %d / %d |",
		c.label, c.p95, c.p50, c.n, spVerdict(c.p95, target), c.floor, c.pre, c.page, c.count, c.pageShape, c.countShape, c.errs, c.timeouts)
}

const spHead = "| Cell | p95 ms | p50 ms | n | Score | Floor p50 ms | Server ms pre / page / count | Page plan | Count plan | errors / timeouts |\n| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |"

func (e *fiEnv) writePlans(t *testing.T, suffix, title string, plans []string) {
	t.Helper()
	out := os.Getenv("NAVA_PERF_OUT")
	if out == "" {
		out = filepath.Join("..", "..", "bin", "field-index-perf-raw.md")
	}
	p := strings.TrimSuffix(out, ".md") + "-" + suffix + "-explain.md"
	if err := os.WriteFile(p, []byte("# "+title+" (EXPLAIN (ANALYZE, BUFFERS, SETTINGS) as nava_app, one run per cell)\n\n"+strings.Join(plans, "\n\n")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func (e *fiEnv) floorGate(t *testing.T, clients int, when string) {
	t.Helper()
	fl := e.transportFloor(t, clients)
	gate := "passes"
	if pct(fl, .5) > 3 {
		gate = "**fails**"
	}
	e.w("- transport floor %s (SELECT 1, %d clients): p50 %.2f ms, p95 %.2f ms: host gate (p50 <= 3 ms) %s; other containers: %v",
		when, clients, pct(fl, .5), pct(fl, .95), gate, foreignContainers(t.Context()))
}

// ---- TICK-1009: S1 and S2 ----------------------------------------------------

func (e *fiEnv) edges(t *testing.T, clients int) {
	t.Helper()
	d := time.Duration(perfEnvInt("NAVA_PERF_LIST_SECONDS", 20)) * time.Second
	admin := []string{"admin-large"}
	var a25, a50 []string
	for n := 1; n <= perfLargeActors; n++ {
		a25, a50 = append(a25, "a25-"+strconv.Itoa(n)), append(a50, "a50-"+strconv.Itoa(n))
	}
	e.w("")
	e.w("## TICK-1009: S1 (band edge) and S2 (ACL breadth), %d clients, %s per cell", clients, d)
	e.w("")
	e.w("- tier shares, all 800k records (TICK-992's v40 / v2 / v025 / v001 unchanged): %s", e.scalar(t, `SELECT string_agg(k || ' ' || n || ' (' || round(100.0 * n / s, 3) || '%)', ', ' ORDER BY n DESC)
		FROM (SELECT CASE WHEN data->>'tier' LIKE 'o%' THEN 'others' ELSE data->>'tier' END AS k, count(*) AS n, sum(count(*)) OVER () AS s FROM records GROUP BY 1) x`))
	e.w("- large type matches: %s", e.scalar(t, `SELECT string_agg(k || ' ' || n, ', ' ORDER BY k) FROM (
		SELECT 'score<' || c AS k, (SELECT count(*) FROM records WHERE tenant_id = $1 AND entity_definition_id = $2 AND (data->>'score')::float8 < c) AS n FROM unnest(ARRAY[200, 500, 1000, 2000]) c
		UNION ALL SELECT 'tier=' || v, (SELECT count(*) FROM records WHERE tenant_id = $1 AND entity_definition_id = $2 AND data->>'tier' = v) FROM unnest(ARRAY['v2', 'v5', 'v10', 'v20']) v) x`, e.large.id, e.largeDef))
	e.floorGate(t, clients, "before")
	var plans []string
	ctlA := e.cell(t, clients, d, "control: agent list (6%), no filter", e.agents, "", &plans)
	ctlN := e.cell(t, clients, d, "control: admin list, no filter", admin, "", &plans)
	e.w("")
	e.w("Control before: ACL p95 %.1f (p50 %.1f, n %d) / no ACL p95 %.1f (p50 %.1f, n %d)", ctlA.p95, ctlA.p50, ctlA.n, ctlN.p95, ctlN.p50, ctlN.n)

	type q struct{ op, sel, q string }
	s1 := []q{
		{"num lt", "5%", "filter%5Bscore%5D%5Blt%5D=500"}, {"num lt", "10%", "filter%5Bscore%5D%5Blt%5D=1000"}, {"num lt", "20%", "filter%5Bscore%5D%5Blt%5D=2000"},
		{"text eq", "5%", "filter%5Btier%5D=v5"}, {"text eq", "10%", "filter%5Btier%5D=v10"}, {"text eq", "20%", "filter%5Btier%5D=v20"},
	}
	e.w("")
	e.w("### S1, band edge (ACL: the agents, 6%% visible; no ACL: the admin)")
	e.w("")
	e.w("Score line: 5%% cells against 350 ms (a pass with a fan-out plan extends band (i) to 5%%); 10%% and 20%% against 150 ms (over 150 ms is a new band: reported, D13 unchanged).")
	e.w("")
	e.w("%s", spHead)
	var summary []string
	for _, c := range s1 {
		target := fiListTargetMS
		if c.sel == "5%" {
			target = fiListACL2TargetMS
		}
		for _, acl := range []struct {
			name string
			toks []string
		}{{"ACL", e.agents}, {"no ACL", admin}} {
			r := e.cell(t, clients, d, fmt.Sprintf("S1 %s %s, %s", c.op, c.sel, acl.name), acl.toks, c.q, &plans)
			e.w("%s", r.row(target))
			summary = append(summary, fmt.Sprintf("%s %.1f vs %.0f: %s", r.label, r.p95, target, spVerdict(r.p95, target)))
		}
	}
	e.w("")
	e.w("### S2, ACL breadth, 2%% (line 350 ms)")
	e.w("")
	e.w("%s", spHead)
	var s2pass []string
	for _, c := range []q{{"num lt", "2%", "filter%5Bscore%5D%5Blt%5D=200"}, {"text eq", "2%", "filter%5Btier%5D=v2"}} {
		highest, broken := "none", false
		for _, b := range []struct {
			name string
			toks []string
		}{{"6% (control)", e.agents}, {"26%", a25}, {"50%", a50}} {
			r := e.cell(t, clients, d, fmt.Sprintf("S2 %s 2%%, agents seeing %s", c.op, b.name), b.toks, c.q, &plans)
			e.w("%s", r.row(fiListACL2TargetMS))
			summary = append(summary, fmt.Sprintf("%s %.1f vs 350: %s", r.label, r.p95, spVerdict(r.p95, fiListACL2TargetMS)))
			// The highest breadth that passes with every narrower one passing.
			if r.p95 > fiListACL2TargetMS {
				broken = true
			} else if !broken {
				highest = b.name
			}
		}
		s2pass = append(s2pass, c.op+": highest passing breadth "+highest)
	}
	ctlA2 := e.cell(t, clients, d, "control after: agent list, no filter", e.agents, "", &plans)
	ctlN2 := e.cell(t, clients, d, "control after: admin list, no filter", admin, "", &plans)
	e.w("")
	e.w("Control after: ACL p95 %.1f / no ACL p95 %.1f", ctlA2.p95, ctlN2.p95)
	e.floorGate(t, clients, "after")
	e.w("")
	e.w("**Scores:** %s.", strings.Join(summary, "; "))
	e.w("")
	e.w("**S2:** %s.", strings.Join(s2pass, "; "))
	e.writePlans(t, "edges", "TICK-1009 S1/S2 plans", plans)
}

// ---- TICK-1039: arms --------------------------------------------------------

// spArms is the handler's IndexTermFunc for the arms phase: arm "" and "D7"
// render the D7 term, "(a)" the adaptive shape, "(b)" the promoted column.
// Only pk is rewritten.
type spArms struct {
	mu     sync.Mutex
	arm    string
	field  uuid.UUID
	picks  map[string]int
	errors []string
}

func (s *spArms) set(arm string) {
	s.mu.Lock()
	s.arm, s.picks = arm, map[string]int{}
	s.mu.Unlock()
}

func (s *spArms) term(ctx context.Context, tx pgx.Tx, in server.IndexTerm) string {
	s.mu.Lock()
	arm, field := s.arm, s.field
	s.mu.Unlock()
	if in.FieldID != field {
		return in.D7
	}
	switch arm {
	case "(a)":
		n, err := spPreCount(ctx, tx, in)
		pick := "side index"
		term := in.D7
		if err != nil {
			s.mu.Lock()
			s.errors = append(s.errors, err.Error())
			s.mu.Unlock()
		} else if n > spK {
			pick = "walk + pkey probe"
			term = strings.TrimSuffix(in.D7, ")") + " OFFSET 0)"
		}
		s.mu.Lock()
		s.picks[pick]++
		s.mu.Unlock()
		return term
	case "(b)", "(b)+group":
		// The field placeholder stays bound (an unreferenced one has no
		// type); under force_custom_plan it folds to true.
		return "(" + strings.ReplaceAll(in.Cmp, "v.value", "r.spike_pk") + " AND " + in.FieldArg + " IS NOT NULL)"
	}
	return in.D7
}

// spPreCount counts the filter's side rows up to spK+1 in the list's own
// transaction (RLS applies).
func spPreCount(ctx context.Context, tx pgx.Tx, in server.IndexTerm) (int, error) {
	cast := map[fieldindex.ValueType]string{fieldindex.TypeText: "text", fieldindex.TypeNum: "double precision",
		fieldindex.TypeTS: "timestamptz", fieldindex.TypeBool: "boolean"}[in.ValueType]
	args := []any{in.FieldID}
	var cmps []string
	for i, op := range in.Ops {
		if op == "in" {
			args = append(args, in.Values[i])
			cmps = append(cmps, fmt.Sprintf("v.value = ANY($%d::%s[])", len(args), cast))
			continue
		}
		sym := map[string]string{"lt": "<", "lte": "<=", "gt": ">", "gte": ">="}[op]
		if sym == "" {
			sym = "="
		}
		args = append(args, in.Values[i][0])
		cmps = append(cmps, fmt.Sprintf("v.value %s $%d::%s", sym, len(args), cast))
	}
	var n int
	err := tx.QueryRow(ctx, `SELECT count(*) FROM (SELECT 1 FROM `+in.Table+` v
		WHERE v.tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid AND v.field_id = $1 AND `+strings.Join(cmps, " AND ")+`
		LIMIT `+strconv.Itoa(spK+1)+`) c`, args...).Scan(&n)
	return n, err
}

func (e *fiEnv) arms(t *testing.T, clients int) {
	t.Helper()
	d := time.Duration(perfEnvInt("NAVA_PERF_LIST_SECONDS", 20)) * time.Second
	a6, a50 := e.tokens["agent-1"], e.tokens["a50-1"]
	e.w("")
	e.w("## TICK-1039: arms D7 / (a) adaptive / (b) promoted column, %d clients, %s per cell, work_mem %s", clients, d, e.scalar(t, "current_setting('work_mem')"))

	// The fixture: visibility of every record of the large type to a6 and
	// a50, then pk's values (see below), its registry row and side rows, and
	// the scratch column for (b), filled before any arm so every arm reads
	// the same heap.
	t0 := time.Now()
	e.exec(t, `CREATE TEMP TABLE sp_rec AS SELECT r.id, r.updated_at,
			`+spVisible+` AS v6,
			`+strings.NewReplacer("$3", "$5", "$4", "$6").Replace(spVisible)+` AS v50,
			(('x' || substr(md5(r.id::text || 'sp'), 1, 7))::bit(28)::int) AS k,
			NULL::float8 AS v
		FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2`,
		e.large.id, e.largeDef, a6.PrincipalID, a6.Groups, a50.PrincipalID, a50.Groups)
	// 40: A1-6 (3), the 8,000 oldest records (2%): anti-correlated with
	// updated_at; a6's count of it stays under the cap.
	e.exec(t, `UPDATE sp_rec SET v = 40 WHERE id IN (SELECT id FROM sp_rec ORDER BY updated_at, id LIMIT 8000)`)
	// 10 / 20 / 30 on records neither agent sees (A2-11's counts): 26,593
	// none visible; 20,993 of 20's 21,487 (the other 494 are a6's, below);
	// 19,999 (the R13 step's lower side).
	e.exec(t, `UPDATE sp_rec s SET v = CASE WHEN x.rn <= 26593 THEN 10 WHEN x.rn <= 26593 + 20993 THEN 20 ELSE 30 END
		FROM (SELECT id, row_number() OVER (ORDER BY k, id) AS rn FROM sp_rec WHERE v IS NULL AND NOT v6 AND NOT v50) x
		WHERE s.id = x.id AND x.rn <= 26593 + 20993 + 19999`)
	e.exec(t, `UPDATE sp_rec s SET v = 20 FROM (SELECT id, row_number() OVER (ORDER BY k, id) AS rn FROM sp_rec WHERE v IS NULL AND v6) x
		WHERE s.id = x.id AND x.rn <= 494`)
	// 3: broad, 200,000 records (50% of the type); 4: the rest.
	e.exec(t, `UPDATE sp_rec s SET v = CASE WHEN x.rn <= 200000 THEN 3 ELSE 4 END
		FROM (SELECT id, row_number() OVER (ORDER BY k, id) AS rn FROM sp_rec WHERE v IS NULL) x WHERE s.id = x.id`)
	e.w("")
	e.w("- pk fixture (value: side rows, visible to a6 / a50): %s", e.scalar(t, `SELECT string_agg(v || ': ' || n || ', ' || n6 || ' / ' || n50, '; ' ORDER BY v)
		FROM (SELECT v, count(*) n, count(*) FILTER (WHERE v6) n6, count(*) FILTER (WHERE v50) n50 FROM sp_rec GROUP BY v) x`))
	e.w("- 40's updated_at: %s; the type's: %s", e.scalar(t, `SELECT min(updated_at) || ' .. ' || max(updated_at) FROM sp_rec WHERE v = 40`),
		e.scalar(t, `SELECT min(updated_at) || ' .. ' || max(updated_at) FROM sp_rec`))
	arm := &spArms{field: uuid.New()}
	e.exec(t, `INSERT INTO record_index_fields (tenant_id, id, entity_definition_id, field_key, value_type, state, progress, requested_version)
		VALUES ($1, $2, $3, 'pk', 'num', 'ready', 100, 1)`, e.large.id, arm.field, e.largeDef)
	e.exec(t, `INSERT INTO rfv_num (tenant_id, field_id, record_id, ord, value) SELECT $1, $2, id, 0, v FROM sp_rec ORDER BY id`, e.large.id, arm.field)
	e.exec(t, "SET session_replication_role = replica")
	e.exec(t, "ALTER TABLE records ADD COLUMN spike_pk double precision")
	e.exec(t, `UPDATE records r SET spike_pk = s.v FROM sp_rec s WHERE r.tenant_id = $1 AND r.id = s.id`, e.large.id)
	e.exec(t, "SET session_replication_role = origin")
	t.Cleanup(func() {
		_, _ = e.super.Exec(context.Background(), "ALTER TABLE records DROP COLUMN IF EXISTS spike_pk")
	})
	// The UPDATE wrote a new version of every large-tenant row: rewrite the
	// heap and its indexes so the arms read P3-36's layout, not a bloated one.
	e.exec(t, "VACUUM FULL records")
	e.exec(t, "VACUUM (ANALYZE) records, rfv_num, record_index_fields")
	e.exec(t, "CHECKPOINT")
	e.exec(t, `SELECT pg_prewarm(c.oid) FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = 'public' AND c.relkind IN ('r', 'i') AND (c.relname LIKE 'records%' OR c.relname LIKE 'rfv\_num%' OR c.relname LIKE 'record\_shares%')`)
	e.w("- fixture, side rows, scratch column, VACUUM FULL records and prewarm in %s; records: %s", time.Since(t0).Round(time.Second),
		e.scalar(t, `SELECT pg_size_pretty(pg_table_size('records')) || ' heap, ' || pg_size_pretty(pg_indexes_size('records')) || ' indexes'`))

	e.h = e.handlerWith(t, server.WithListIndexTerm(arm.term))
	type fx struct {
		name, val string
		toks      []string
		acl       bool
	}
	pk := func(v string) string { return "filter%5Bpk%5D=" + v }
	fixtures := []fx{
		{"A1-6 (3) anti-correlated 2% (8,000 oldest), a6", "40", []string{"agent-1"}, true},
		{"A1-6 (3) anti-correlated 2% (8,000 oldest), a50", "40", []string{"a50-1"}, true},
		{"A1-6 (3) anti-correlated 2% (8,000 oldest), no ACL", "40", []string{"admin-large"}, false},
		{"none visible (26,593), a6", "10", []string{"agent-1"}, true},
		{"none visible (26,593), a50", "10", []string{"a50-1"}, true},
		{"partly correlated (494 of 21,487 visible to a6), a6", "20", []string{"agent-1"}, true},
		{"partly correlated (494 of 21,487 visible to a6), a50", "20", []string{"a50-1"}, true},
		{"19,999 invisible side rows, a6", "30", []string{"agent-1"}, true},
		{"19,999 invisible side rows, a50", "30", []string{"a50-1"}, true},
		{"broad 50% (200,000), a6", "3", []string{"agent-1"}, true},
		{"broad 50% (200,000), a50", "3", []string{"a50-1"}, true},
	}
	e.floorGate(t, clients, "before")
	var plans []string
	for _, c := range []struct{ name, tok string }{{"a6", "agent-1"}, {"a50", "a50-1"}, {"no ACL", "admin-large"}} {
		r := e.cell(t, clients, d, "control: "+c.name+", no filter", []string{c.tok}, "", &plans)
		e.w("- control %s, no filter: p95 %.1f (p50 %.1f, n %d)", c.name, r.p95, r.p50, r.n)
	}
	type armRes struct {
		arm   string
		cells []spCell
		picks map[string]int
	}
	var res []armRes
	runArm := func(name string) {
		arm.set(name)
		ar := armRes{arm: name}
		e.w("")
		e.w("### Arm %s (line 150 ms)", name)
		e.w("")
		e.w("%s", spHead)
		for _, f := range fixtures {
			c := e.cell(t, clients, d, name+" "+f.name, f.toks, pk(f.val), &plans)
			e.w("%s", c.row(fiListTargetMS))
			ar.cells = append(ar.cells, c)
		}
		arm.mu.Lock()
		ar.picks = arm.picks
		arm.mu.Unlock()
		if name == "(a)" {
			e.w("")
			e.w("(a)'s picks over the arm's requests: %v (pre-count cap %d)", ar.picks, spK)
		}
		res = append(res, ar)
	}
	runArm("D7")
	runArm("(a)")
	e.exec(t, "CREATE INDEX spike_records_pk ON records (tenant_id, entity_definition_id, spike_pk, updated_at)")
	e.exec(t, "ANALYZE records")
	e.w("")
	e.w("- (b) index: %s", e.scalar(t, `SELECT pg_size_pretty(pg_relation_size('spike_records_pk'))`))
	runArm("(b)")
	e.exec(t, "DROP INDEX spike_records_pk")
	e.exec(t, "CREATE INDEX spike_records_pk_group ON records (tenant_id, entity_definition_id, spike_pk, assignment_group_id, updated_at)")
	e.exec(t, "ANALYZE records")
	e.w("")
	e.w("- (b)+group index: %s", e.scalar(t, `SELECT pg_size_pretty(pg_relation_size('spike_records_pk_group'))`))
	runArm("(b)+group")
	e.exec(t, "DROP INDEX spike_records_pk_group")
	arm.set("")
	e.floorGate(t, clients, "after")

	e.w("")
	e.w("### Summary: p95 ms by arm (line 150 ms, provisional within 10%%)")
	e.w("")
	head := "| Fixture |"
	sep := "| --- |"
	for _, r := range res {
		head += " " + r.arm + " |"
		sep += " --- |"
	}
	e.w("%s", head)
	e.w("%s", sep)
	for i, f := range fixtures {
		row := "| " + f.name + " |"
		for _, r := range res {
			row += fmt.Sprintf(" %.1f %s |", r.cells[i].p95, spVerdict(r.cells[i].p95, fiListTargetMS))
		}
		e.w("%s", row)
	}
	e.w("")
	for _, r := range res {
		var misses []string
		for i, c := range r.cells {
			if c.p95 > fiListTargetMS {
				misses = append(misses, fixtures[i].name)
			}
		}
		if len(misses) == 0 {
			e.w("- **%s meets the A1-6 exit criterion in every fixture.**", r.arm)
		} else {
			e.w("- %s misses %d of %d: %s", r.arm, len(misses), len(r.cells), strings.Join(misses, "; "))
		}
	}
	if len(arm.errors) > 0 {
		t.Errorf("(a) pre-count errors: %v", arm.errors[:min(len(arm.errors), 5)])
	}
	e.writePlans(t, "arms", "TICK-1039 arm plans", plans)
}

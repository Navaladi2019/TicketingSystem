//go:build integration && perf

// P7-07 (TICK-747): the S21 field index measured on the merged code against
// ADR-006 D10 and D13, at the spike's scale (docs/adr/ADR-006/spikes/postgres/
// s1-s3): 800k records, one 400k tenant, 8 clients, nava_app under forced RLS.
//
// What is real, and what is set up by hand:
//
//   - Lists go through GET /api/v1/records/{entity}?filter[...] on the real
//     handler: the D7 EXISTS the list builder emits, the compiled ACL, the
//     real registry read. Creates and updates go through POST and PATCH, so
//     the records trigger (20261120140200, with the E1-5 READ COMMITTED
//     guard) runs as nava_app inside the audited write.
//   - The backfill is the real FieldIndexBackfill workflow and its
//     activities, as nava_worker, on a Temporal dev server (real timers, so
//     the adaptive pause and its batch timings are the production ones). The
//     field is flagged late through the real publish (POST
//     /api/v1/entity-types/ticket/versions), which writes the registry row.
//   - Set up by hand, as the superuser: the 800k seed (P3-36's, plus the
//     band fields below), and the list phase's registry rows and side rows.
//     The quota (DefaultPerTypeLimit, 5) caps what a publish may flag, and
//     the bands need eight fields; the side rows are copied with
//     record_field_coerce, the function the trigger and the backfill call.
//
// Band fields, from a hash of (tenant, n), so selectivity is independent of
// updated_at and of the ACL: tier (text eq), zone (text in, two values),
// score (num lt), opened (ts range), b40, b2, b025, b001 (bool eq). Each has
// a value or range at 40%, 2%, 0.25% and 0.01% of every tenant's records.
//
// Run with scratch/field-index-measured/run.sh. Report:
// docs/data/field-index-measured.md.
//
// Env (besides P3-36's NAVA_PERF_CLIENTS and NAVA_PERF_OUT):
// NAVA_PERF_LIST_SECONDS (20) per list scenario, NAVA_PERF_WRITE_SECONDS
// (60) per insert round, NAVA_PERF_WRITE_ROUNDS (3), NAVA_PERF_LIVE_SECONDS
// (180) per live-mix control, NAVA_PERF_LIVE_MAX (1200) cap on the live mix
// under the backfill, NAVA_PERF_PHASES (lists,guard,writes,backfill),
// NAVA_PERF_ASIS_BATCHES (40) and NAVA_PERF_FIX_BATCHES (60) batches of each
// as-merged and fix-candidate backfill sample (0 runs to the end),
// NAVA_PERF_TEMPORAL_CLI (an existing temporal CLI binary; empty downloads
// the SDK's default).

package server_test

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
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
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/worker"

	"github.com/Navaladi2019/TicketingSystem/internal/fieldindex"
	"github.com/Navaladi2019/TicketingSystem/internal/fieldindex/backfill"
)

// The ADR-006 targets this run is scored against (PLAN-007 P7-07).
const (
	fiListTargetMS       = 150.0 // D13: every band
	fiListACL2TargetMS   = 350.0 // D13: the 2% band with a narrowing ACL
	fiInsertFloor        = 0.40  // insert tps with 5 flagged fields / baseline
	fiLiveP95DeltaBudget = 0.25  // D10: live-write p95 under the backfill
)

// fiSchemaExtra are the band fields, plain properties: the list phase flags
// them in the registry directly (see the header). The late-flagged backfill
// field is perfSchema's own priority (5 values).
const fiSchemaExtra = `"tier": {"type": "string", "maxLength": 10},
    "zone": {"type": "string", "maxLength": 10},
    "score": {"type": "number"},
    "opened": {"type": "string", "format": "date-time"},
    "b40": {"type": "boolean"}, "b2": {"type": "boolean"}, "b025": {"type": "boolean"}, "b001": {"type": "boolean"},
    `

// fiU is a uniform draw in [0, 1,000,000) from md5(<k>tn/n): no random()
// call, so P3-01's RNG stream is untouched.
func fiU(k string) string {
	return fmt.Sprintf(`((('x' || substr(md5('%s' || x.tn || '/' || x.n), 1, 7))::bit(28)::int) %% 1000000)`, k)
}

// fiDataExtra builds the band values. tier: v40 40%, v2 2%, v025 0.25%, v001
// 0.01%, else one of 97 others (about 0.6% each). zone: the same bands split
// into an a and b value, filtered with in. score: u/100, so lt 4000, 200, 25,
// 1. opened: 2024-01-01 plus u minutes, ranges from u = 300,000. bools: true
// under 400,000, 20,000, 2,500 and 100.
func fiDataExtra() string {
	u1, u2, u3, u4, u5 := fiU("t"), fiU("z"), fiU("s"), fiU("o"), fiU("b")
	return fmt.Sprintf(`,
		'tier', CASE WHEN %[1]s < 400000 THEN 'v40' WHEN %[1]s < 420000 THEN 'v2' WHEN %[1]s < 422500 THEN 'v025'
		             WHEN %[1]s < 422600 THEN 'v001' ELSE 'o' || (%[1]s %% 97) END,
		'zone', CASE WHEN %[2]s < 200000 THEN 'p40a' WHEN %[2]s < 400000 THEN 'p40b' WHEN %[2]s < 410000 THEN 'p2a'
		             WHEN %[2]s < 420000 THEN 'p2b' WHEN %[2]s < 421250 THEN 'p025a' WHEN %[2]s < 422500 THEN 'p025b'
		             WHEN %[2]s < 422550 THEN 'p001a' WHEN %[2]s < 422600 THEN 'p001b' ELSE 'q' || (%[2]s %% 97) END,
		'score', %[3]s / 100.0,
		'opened', to_char((timestamptz '2024-01-01 00:00:00+00' + %[4]s * interval '1 minute') AT TIME ZONE 'UTC',
		                  'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
		'b40', %[5]s < 400000, 'b2', %[5]s < 20000, 'b025', %[5]s < 2500, 'b001', %[5]s < 100`, u1, u2, u3, u4, u5)
}

// fiField is a band field and its value type.
type fiField struct{ key, vt string }

func fiListFields() []fiField {
	return []fiField{{"tier", "text"}, {"zone", "text"}, {"score", "num"}, {"opened", "ts"},
		{"b40", "bool"}, {"b2", "bool"}, {"b025", "bool"}, {"b001", "bool"}}
}

// fiWriteFields are the 5 flagged fields of the insert run (the D12 quota),
// one per operator.
func fiWriteFields() []string { return []string{"tier", "zone", "score", "opened", "b2"} }

// fiBands are the four selectivities, in report order.
func fiBands() []string { return []string{"40%", "2%", "0.25%", "0.01%"} }

// fiOpDef is one operator's filters, by band, as query strings (fiOps).
type fiOpDef struct {
	op, field string
	q         map[string]string
}

func fiOps() []fiOpDef {
	ts := func(minutes int) string {
		return fiEsc(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).Add(time.Duration(minutes) * time.Minute).Format(time.RFC3339))
	}
	rng := func(w int) string {
		return "filter%5Bopened%5D%5Bgte%5D=" + ts(300000) + "&filter%5Bopened%5D%5Blt%5D=" + ts(300000+w)
	}
	return []fiOpDef{
		{"text eq", "tier", map[string]string{"40%": "filter%5Btier%5D=v40", "2%": "filter%5Btier%5D=v2", "0.25%": "filter%5Btier%5D=v025", "0.01%": "filter%5Btier%5D=v001"}},
		{"text in (2 values)", "zone", map[string]string{"40%": "filter%5Bzone%5D%5Bin%5D=p40a,p40b", "2%": "filter%5Bzone%5D%5Bin%5D=p2a,p2b",
			"0.25%": "filter%5Bzone%5D%5Bin%5D=p025a,p025b", "0.01%": "filter%5Bzone%5D%5Bin%5D=p001a,p001b"}},
		{"num lt", "score", map[string]string{"40%": "filter%5Bscore%5D%5Blt%5D=4000", "2%": "filter%5Bscore%5D%5Blt%5D=200",
			"0.25%": "filter%5Bscore%5D%5Blt%5D=25", "0.01%": "filter%5Bscore%5D%5Blt%5D=1"}},
		{"ts range", "opened", map[string]string{"40%": rng(400000), "2%": rng(20000), "0.25%": rng(2500), "0.01%": rng(100)}},
		{"bool eq", "b*", map[string]string{"40%": "filter%5Bb40%5D=true", "2%": "filter%5Bb2%5D=true",
			"0.25%": "filter%5Bb025%5D=true", "0.01%": "filter%5Bb001%5D=true"}},
	}
}

func fiEsc(s string) string { return strings.NewReplacer(":", "%3A", "+", "%2B").Replace(s) }

// fiEnv is perfEnv plus what the field-index run needs.
type fiEnv struct {
	*perfEnv
	worker   *pgxpool.Pool // nava_worker: the backfill's activities
	largeDef uuid.UUID     // the large tenant's ticket type
	w        func(format string, args ...any)
	// explainOut are the captured list plans, written beside the report.
	explainOut []string
	// seqs are each backfill's batch durations and pauses, in order.
	seqs []string
	// errSample is the first response body of each live-mix error.
	errSample map[string]string
}

func (e *fiEnv) phases() map[string]bool {
	v := os.Getenv("NAVA_PERF_PHASES")
	if v == "" {
		v = "lists,guard,writes,backfill"
	}
	out := map[string]bool{}
	for _, p := range strings.Split(v, ",") {
		out[strings.TrimSpace(p)] = true
	}
	return out
}

func (e *fiEnv) scalar(t *testing.T, sql string, args ...any) string {
	t.Helper()
	var s string
	if err := e.super.QueryRow(t.Context(), "SELECT ("+sql+")::text", args...).Scan(&s); err != nil {
		t.Fatalf("%s: %v", firstLine(sql), err)
	}
	return s
}

func (e *fiEnv) timed(t *testing.T, sql string, args ...any) time.Duration {
	t.Helper()
	t0 := time.Now()
	e.exec(t, sql, args...)
	return time.Since(t0)
}

func TestFieldIndexPerf800k(t *testing.T) {
	clients := perfEnvInt("NAVA_PERF_CLIENTS", 8)
	out := os.Getenv("NAVA_PERF_OUT")
	if out == "" {
		out = filepath.Join("..", "..", "bin", "field-index-perf-raw.md")
	}
	var buf strings.Builder
	w := func(format string, args ...any) {
		line := fmt.Sprintf(format, args...)
		buf.WriteString(line + "\n")
		t.Log(line)
		// The raw report is rewritten after every line: a run cut short
		// keeps what it measured.
		_ = os.MkdirAll(filepath.Dir(out), 0o750)
		_ = os.WriteFile(out, []byte(buf.String()), 0o600)
	}
	pe := newPerfEnv(t, clients)
	pe.schemaExtra, pe.dataExtra = fiSchemaExtra, fiDataExtra()
	e := &fiEnv{perfEnv: pe, w: w, errSample: map[string]string{}}
	w("# TICK-747 field-index measurements, raw (%s)", time.Now().UTC().Format(time.RFC3339))
	w("")
	t0 := time.Now()
	e.seed(t)
	e.exec(t, "VACUUM (ANALYZE)")
	e.exec(t, "CHECKPOINT")
	w("- seed: 800k records + VACUUM (ANALYZE) + CHECKPOINT in %s", time.Since(t0).Round(time.Second))
	w("- %s", e.scalar(t, "SELECT version()"))
	w("- settings: %s", e.scalar(t, `SELECT string_agg(name || '=' || setting || coalesce(unit, ''), ', ' ORDER BY name) FROM pg_settings
		WHERE name IN ('shared_buffers','effective_cache_size','work_mem','random_page_cost','max_parallel_workers_per_gather','jit','fsync','synchronous_commit','default_statistics_target')`))
	w("- nava_app plan_cache_mode: %s", e.scalar(t, `SELECT coalesce((SELECT array_to_string(setconfig, ',') FROM pg_db_role_setting s JOIN pg_roles r ON r.oid = s.setrole WHERE r.rolname = 'nava_app'), 'default')`))
	w("- other containers at start: %v", foreignContainers(t.Context()))
	if err := e.super.QueryRow(t.Context(), `SELECT d.id FROM entity_definitions d JOIN perf_tenant p ON p.tenant_id = d.tenant_id
		WHERE p.tn = 0 AND d.key = 'ticket'`).Scan(&e.largeDef); err != nil {
		t.Fatal(err)
	}
	wc := e.pool.Config().Copy()
	wc.ConnConfig.User, wc.ConnConfig.Password = "nava_worker", etPassword("nava_worker")
	wc.ConnConfig.Tracer = nil
	// nava-worker's pool is pgxpool.ParseConfig's default, the statement
	// cache (cmd/nava-worker/main.go), not nava-core's QueryExecModeExec. The
	// batch's []uuid.UUID argument needs it.
	wc.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeCacheStatement
	wc.MaxConns = 4
	wp, err := pgxpool.NewWithConfig(t.Context(), wc)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(wp.Close)
	e.worker = wp
	floor := e.transportFloor(t, clients)
	w("- transport floor (SELECT 1, %d clients): p50 %.2f ms, p95 %.2f ms", clients, pct(floor, .5), pct(floor, .95))

	ph := e.phases()
	switch {
	case ph["lists"] || ph["writes"]:
		e.flagForLists(t, fiListFields(), false)
	case ph["backfill"] || ph["probe"]:
		// A backfill-only run flags the live mix's four fields on the large
		// type, as the full run leaves them for its backfill phase.
		e.flagForLists(t, fiListFields()[:4], true)
	}
	if ph["lists"] {
		e.lists(t, clients)
	}
	if ph["guard"] {
		e.guard(t)
	}
	if ph["writes"] {
		e.writes(t, clients)
	}
	if ph["probe"] {
		e.probe(t, clients)
	}
	if ph["backfill"] {
		e.backfillPhase(t, clients)
	}
	for k, v := range e.errSample {
		w("- live-mix error %s, first body: %s", k, v)
	}
	t.Logf("wrote %s", out)
}

// flagForLists writes a ready registry row for every band field of every
// tenant's ticket type and copies the side rows with record_field_coerce,
// then analyses the side tables (the bool MCV target is the migration's
// 10,000).
func (e *fiEnv) flagForLists(t *testing.T, fields []fiField, onlyLarge bool) {
	t.Helper()
	var vals []string
	for _, f := range fields {
		vals = append(vals, fmt.Sprintf("('%s', '%s')", f.key, f.vt))
	}
	which := uuid.Nil // every tenant's ticket type
	if onlyLarge {
		which = e.largeDef
	}
	e.exec(t, `INSERT INTO record_index_fields (tenant_id, id, entity_definition_id, field_key, value_type, state, progress, requested_version)
		SELECT d.tenant_id, gen_random_uuid(), d.id, f.k, f.vt, 'ready', 100, 1
		FROM entity_definitions d CROSS JOIN (VALUES `+strings.Join(vals, ", ")+`) f(k, vt)
		WHERE d.key = 'ticket' AND ($1 = '00000000-0000-0000-0000-000000000000'::uuid OR d.id = $1)`, which)
	t0 := time.Now()
	e.exec(t, "SET session_replication_role = replica")
	for vt, col := range sideColumn {
		e.exec(t, fmt.Sprintf(`INSERT INTO rfv_%s (tenant_id, field_id, record_id, ord, value)
			SELECT r.tenant_id, f.id, r.id, c.ord, c.%s
			FROM record_index_fields f JOIN records r ON r.tenant_id = f.tenant_id AND r.entity_definition_id = f.entity_definition_id,
			LATERAL record_field_coerce(r.data -> f.field_key, f.value_type) c
			WHERE f.value_type = $1 ORDER BY r.tenant_id, r.updated_at`, vt, col), vt)
	}
	e.exec(t, "SET session_replication_role = origin")
	copyT := time.Since(t0)
	e.exec(t, "VACUUM rfv_text, rfv_num, rfv_ts, rfv_bool")
	var an []string
	for _, tb := range []string{"rfv_text", "rfv_num", "rfv_ts", "rfv_bool"} {
		an = append(an, fmt.Sprintf("%s %s", tb, e.timed(t, "ANALYZE "+tb).Round(time.Millisecond)))
	}
	e.exec(t, "CHECKPOINT")
	e.exec(t, `SELECT pg_prewarm(c.oid) FROM pg_class c WHERE c.relname LIKE 'rfv\_%' AND c.relkind IN ('r', 'i')`)
	scope := "all 400 ticket types"
	if onlyLarge {
		scope = "the large tenant's ticket type only"
	}
	e.w("- %d fields flagged on %s; side rows copied in %s; ANALYZE: %s", len(fields), scope, copyT.Round(time.Second), strings.Join(an, ", "))
	e.w("- side tables: %s", e.scalar(t, `SELECT string_agg(c.relname || ' ' || c.reltuples::bigint || ' rows, ' ||
		pg_size_pretty(pg_table_size(c.oid)) || ' heap, ' || pg_size_pretty(pg_indexes_size(c.oid)) || ' indexes', '; ' ORDER BY c.relname)
		FROM pg_class c WHERE c.relname IN ('rfv_text', 'rfv_num', 'rfv_ts', 'rfv_bool')`))
	e.w("- records: %s", e.scalar(t, `SELECT pg_size_pretty(pg_table_size('records')) || ' heap, ' || pg_size_pretty(pg_indexes_size('records')) || ' indexes'`))
}

// fiPlanning is the "Planning Time" of an EXPLAIN (ANALYZE) output.
var fiPlanning = regexp.MustCompile(`Planning Time: ([0-9.]+) ms`)

var fiExecution = regexp.MustCompile(`Execution Time: ([0-9.]+) ms`)

func (e *fiEnv) lists(t *testing.T, clients int) {
	t.Helper()
	d := time.Duration(perfEnvInt("NAVA_PERF_LIST_SECONDS", 20)) * time.Second
	admin := []string{"admin-large"}
	type cell struct {
		p95, p50 float64
		n        int
		timeouts int64
		plan     string
	}
	runOne := func(name string, tokens []string, q string) (cell, *perfResult) {
		sc := perfScenario{name: name, kind: "data", tenant: "large", reqs: listReqs(tokens, q)}
		r := e.run(t, sc, clients, d)
		c := cell{p95: pct(r.lat, .95), p50: pct(r.lat, .5), n: len(r.lat), timeouts: r.timeouts}
		for _, x := range r.explains {
			if m := fiPlanning.FindStringSubmatch(x); m != nil && strings.Contains(x, "rfv_") {
				ex := fiExecution.FindStringSubmatch(x)
				c.plan = "planning " + m[1] + " ms, execution " + ex[1] + " ms"
			}
		}
		return c, r
	}
	e.w("")
	e.w("## Lists: p95 ms, %d clients, %s per cell, ACL (agent, narrowing) / no ACL (admin)", clients, d)
	ctlACL, _ := runOne("control: agent list, no filter", e.agents, "")
	ctlAdm, _ := runOne("control: admin list, no filter", admin, "")
	e.w("")
	e.w("Control before, no data filter: ACL p95 %.1f (p50 %.1f, n %d) / no ACL p95 %.1f (p50 %.1f, n %d)", ctlACL.p95, ctlACL.p50, ctlACL.n, ctlAdm.p95, ctlAdm.p50, ctlAdm.n)
	e.w("")
	e.w("| Operator | 40%% | 2%% | 0.25%% | 0.01%% |")
	e.w("| --- | --- | --- | --- | --- |")
	var misses, plans, explains, floors []string
	for _, op := range fiOps() {
		row := "| " + op.op + " |"
		for _, band := range fiBands() {
			q := op.q[band]
			fl := e.transportFloor(t, clients)
			a, ra := runOne(op.op+" "+band+", ACL", e.agents, q)
			n, rn := runOne(op.op+" "+band+", no ACL", admin, q)
			row += fmt.Sprintf(" %.1f / %.1f |", a.p95, n.p95)
			floors = append(floors, fmt.Sprintf("%s %s %.1f", op.op, band, pct(fl, .5)))
			target := fiListTargetMS
			if band == "2%" {
				target = fiListACL2TargetMS
			}
			if a.p95 > target {
				misses = append(misses, fmt.Sprintf("%s %s ACL p95 %.1f > %.0f", op.op, band, a.p95, target))
			}
			if n.p95 > fiListTargetMS {
				misses = append(misses, fmt.Sprintf("%s %s no ACL p95 %.1f > %.0f", op.op, band, n.p95, fiListTargetMS))
			}
			if a.timeouts+n.timeouts > 0 {
				misses = append(misses, fmt.Sprintf("%s %s: %d / %d filter_too_expensive", op.op, band, a.timeouts, n.timeouts))
			}
			plans = append(plans, fmt.Sprintf("| %s | %s | %s | %s |", op.op, band, a.plan, n.plan))
			for _, r := range []*perfResult{ra, rn} {
				for i, s := range r.stmtSQL {
					if strings.Contains(s.SQL, "rfv_") && i < len(r.explains) && r.explains[i] != "" {
						explains = append(explains, "### "+r.sc.name+"\n\n```sql\n"+strings.TrimSpace(s.SQL)+"\n```\n\n```\n"+r.explains[i]+"\n```")
					}
				}
			}
		}
		e.w("%s", row)
	}
	ctlACL2, _ := runOne("control after: agent list, no filter", e.agents, "")
	ctlAdm2, _ := runOne("control after: admin list, no filter", admin, "")
	e.w("")
	e.w("Control after: ACL p95 %.1f / no ACL p95 %.1f", ctlACL2.p95, ctlAdm2.p95)
	e.w("")
	e.w("Transport floor p50 (SELECT 1, %d clients) before each cell, ms: %s", clients, strings.Join(floors, "; "))
	e.w("")
	e.w("Targets (D13): p95 <= %.0f ms in every cell, except 2%% with ACL <= %.0f ms.", fiListTargetMS, fiListACL2TargetMS)
	if len(misses) == 0 {
		e.w("**Lists: every cell meets its target.**")
	} else {
		e.w("**Lists: %d miss(es):** %s", len(misses), strings.Join(misses, "; "))
	}
	e.w("")
	e.w("### Single-run EXPLAIN (ANALYZE) of the captured list statement")
	e.w("")
	e.w("| Operator | Band | ACL | No ACL |")
	e.w("| --- | --- | --- | --- |")
	for _, p := range plans {
		e.w("%s", p)
	}
	e.explainOut = explains
	e.writeExplains(t)
}

func (e *fiEnv) writeExplains(t *testing.T) {
	t.Helper()
	out := os.Getenv("NAVA_PERF_OUT")
	if out == "" {
		out = filepath.Join("..", "..", "bin", "field-index-perf-raw.md")
	}
	p := strings.TrimSuffix(out, ".md") + "-explain.md"
	if err := os.WriteFile(p, []byte("# TICK-747 list plans (one run per cell, as nava_app)\n\n"+strings.Join(e.explainOut, "\n\n")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
}

// guard measures the E1-5 READ COMMITTED check on its own: the statement the
// trigger runs first, a million times in plpgsql, against the same loop
// with a constant condition. The difference per iteration is its cost per
// fired row.
func (e *fiEnv) guard(t *testing.T) {
	t.Helper()
	with := `DO $$ BEGIN FOR i IN 1..1000000 LOOP
		IF current_setting('transaction_isolation') <> 'read committed' THEN RAISE EXCEPTION 'x'; END IF; END LOOP; END $$`
	without := `DO $$ BEGIN FOR i IN 1..1000000 LOOP IF i < 0 THEN RAISE EXCEPTION 'x'; END IF; END LOOP; END $$`
	var a, b []time.Duration
	for range 5 {
		a = append(a, e.timed(t, with))
		b = append(b, e.timed(t, without))
	}
	slices.Sort(a)
	slices.Sort(b)
	// ns over 1,000,000 iterations, in µs per iteration.
	per := float64((a[2] - b[2]).Nanoseconds()) / 1e6 / 1e3
	e.w("")
	e.w("## E1-5 guard (one current_setting() per fired row)")
	e.w("")
	e.w("1,000,000 iterations, median of 5: with the check %s, constant condition %s: **%.3f µs per fired row**.", a[2].Round(time.Millisecond), b[2].Round(time.Millisecond), per)
}

// fiCreateReqs are createReqs with every band field set, so a create writes
// one side row per flagged field.
func (e *fiEnv) fiCreateReqs(t *testing.T) []perfReq {
	t.Helper()
	reqs := e.createReqs(t, e.agents)
	for i := range reqs {
		var body map[string]map[string]any
		if err := json.Unmarshal([]byte(reqs[i].body), &body); err != nil {
			t.Fatal(err)
		}
		d := body["data"]
		d["tier"], d["zone"], d["score"] = "o"+strconv.Itoa(i%97), "q"+strconv.Itoa(i%97), float64(i%10000)+0.5
		d["opened"] = time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC).Add(time.Duration(i) * time.Minute).Format(time.RFC3339)
		d["b40"], d["b2"], d["b025"], d["b001"] = i%2 == 0, i%50 == 0, false, false
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		reqs[i].body = string(b)
	}
	return reqs
}

// largeFieldsTo leaves the large ticket type with exactly keep flagged (side
// rows of the others deleted first, then their registry rows).
func (e *fiEnv) largeFieldsTo(t *testing.T, keep []string) {
	t.Helper()
	for _, tb := range []string{"rfv_text", "rfv_num", "rfv_ts", "rfv_bool"} {
		e.exec(t, `DELETE FROM `+tb+` v USING record_index_fields f WHERE v.tenant_id = f.tenant_id AND v.field_id = f.id
			AND f.entity_definition_id = $1 AND NOT (f.field_key = ANY($2))`, e.largeDef, keep)
	}
	e.exec(t, `DELETE FROM record_index_fields WHERE entity_definition_id = $1 AND NOT (field_key = ANY($2))`, e.largeDef, keep)
	e.exec(t, "VACUUM (ANALYZE) rfv_text, rfv_num, rfv_ts, rfv_bool, record_index_fields")
}

// writes: insert tps through POST, 8 clients, 5 flagged fields against the
// same type with those 5 rows set to removing (the trigger still fires, runs
// the E1-5 guard and reads the registry, and copies nothing: the cost of a
// type with no flagged field). Alternating rounds, so table growth and host
// noise hit both arms.
func (e *fiEnv) writes(t *testing.T, clients int) {
	t.Helper()
	e.largeFieldsTo(t, fiWriteFields())
	d := time.Duration(perfEnvInt("NAVA_PERF_WRITE_SECONDS", 60)) * time.Second
	rounds := perfEnvInt("NAVA_PERF_WRITE_ROUNDS", 3)
	reqs := e.fiCreateReqs(t)
	e.w("")
	e.w("## Inserts: POST /api/v1/records/ticket, %d clients, %s per round, 5 flagged fields (%s) vs 0", clients, d, strings.Join(fiWriteFields(), ", "))
	e.w("")
	e.w("| Round | Arm | tps | p50 ms | p95 ms | storage probe p50 ms | errors |")
	e.w("| --- | --- | --- | --- | --- | --- | --- |")
	var base, five []float64
	for r := 1; r <= rounds; r++ {
		for _, arm := range []string{"0 flagged (baseline)", "5 flagged"} {
			state := "removing"
			if arm == "5 flagged" {
				state = "ready"
			}
			e.exec(t, `UPDATE record_index_fields SET state = $2 WHERE entity_definition_id = $1`, e.largeDef, state)
			e.exec(t, "CHECKPOINT")
			storage := e.storageProbe(t)
			sc := perfScenario{name: "create, " + arm, kind: "create", tenant: "large", reqs: reqs}
			res := e.run(t, sc, clients, d)
			tps := float64(len(res.lat)) / d.Seconds()
			if arm == "5 flagged" {
				five = append(five, tps)
			} else {
				base = append(base, tps)
			}
			e.w("| %d | %s | %.1f | %.1f | %.1f | %.2f | %d |", r, arm, tps, pct(res.lat, .5), pct(res.lat, .95), storage, res.errs)
		}
	}
	e.exec(t, `UPDATE record_index_fields SET state = 'ready' WHERE entity_definition_id = $1`, e.largeDef)
	mb, mf := fiMean(base), fiMean(five)
	verdict := "meets"
	if mf < fiInsertFloor*mb {
		verdict = "**misses**"
	}
	e.w("")
	e.w("Mean tps: baseline %.1f, 5 flagged %.1f: **%.0f%% of baseline** (floor %.0f%%): %s.", mb, mf, 100*mf/mb, 100*fiInsertFloor, verdict)
	e.w("- side rows written by the creates (large type): %s", e.scalar(t, `SELECT count(*) FROM rfv_text v JOIN record_index_fields f ON f.tenant_id = v.tenant_id AND f.id = v.field_id
		JOIN records r ON r.tenant_id = v.tenant_id AND r.id = v.record_id WHERE f.entity_definition_id = $1 AND r.number > 'TKT0400000'`, e.largeDef))
}

func fiMean(v []float64) float64 {
	if len(v) == 0 {
		return 0
	}
	s := 0.0
	for _, x := range v {
		s += x
	}
	return s / float64(len(v))
}

// ---- the live mix ----------------------------------------------------------

type fiOp struct {
	at   time.Time
	d    time.Duration
	kind string // create or patch
}

// liveMix runs clients workers until ctx ends: each alternates a create
// (fiCreateReqs) and a PATCH of score, a flagged field, on a record of its
// own (If-Match with the version it last saw). Every op is timed and
// timestamped, so a window can be cut out afterwards.
func (e *fiEnv) liveMix(ctx context.Context, t *testing.T, clients int, creates []perfReq, mine [][]uuid.UUID) ([]fiOp, map[string]int64) {
	t.Helper()
	var (
		mu   sync.Mutex
		ops  []fiOp
		errs = map[string]int64{}
		wg   sync.WaitGroup
		next atomic.Int64
	)
	for c := 0; c < clients; c++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ids := mine[c]
			ver := map[uuid.UUID]int64{}
			for _, id := range ids {
				ver[id] = 1
			}
			var local []fiOp
			local = slices.Grow(local, 4096)
			for i := 0; ctx.Err() == nil; i++ {
				t0 := time.Now()
				var rec *httptest.ResponseRecorder
				var pid uuid.UUID
				kind := "create"
				ok := false
				if i%2 == 0 {
					r := creates[int(next.Add(1)-1)%len(creates)]
					rec = e.serve(ctx, http.MethodPost, r.path, r.token, uuid.NewString(), r.body)
					ok = rec.Code == http.StatusCreated
				} else {
					kind = "patch"
					pid = ids[(i/2)%len(ids)]
					body := fmt.Sprintf(`{"data": {"score": %d.25}}`, (i*7919+c)%10000)
					req := httptest.NewRequestWithContext(ctx, http.MethodPatch, "/api/v1/records/ticket/"+pid.String(), strings.NewReader(body))
					req.Header.Set("Authorization", "Bearer admin-large")
					req.Header.Set("Content-Type", "application/merge-patch+json")
					req.Header.Set("Idempotency-Key", uuid.NewString())
					req.Header.Set("If-Match", `W/"`+strconv.FormatInt(ver[pid], 10)+`"`)
					rec = httptest.NewRecorder()
					e.h.ServeHTTP(rec, req)
					ok = rec.Code == http.StatusOK
					if ok {
						et := strings.Trim(strings.TrimPrefix(rec.Header().Get("ETag"), "W/"), `"`)
						if n, err := strconv.ParseInt(et, 10, 64); err == nil {
							ver[pid] = n
						}
					}
				}
				el := time.Since(t0)
				if ctx.Err() != nil {
					break
				}
				if !ok {
					var pb struct {
						Code string `json:"code"`
					}
					_ = json.Unmarshal(rec.Body.Bytes(), &pb)
					k := fmt.Sprintf("%s %d %s", kind, rec.Code, pb.Code)
					mu.Lock()
					if errs[k] == 0 {
						e.errSample[k] = rec.Body.String()
					}
					errs[k]++
					mu.Unlock()
					if kind == "patch" && rec.Code == http.StatusPreconditionFailed {
						if et := strings.Trim(strings.TrimPrefix(rec.Header().Get("ETag"), "W/"), `"`); et != "" {
							if n, err := strconv.ParseInt(et, 10, 64); err == nil {
								ver[pid] = n
							}
						}
					}
					continue
				}
				local = append(local, fiOp{at: t0, d: el, kind: kind})
			}
			mu.Lock()
			ops = append(ops, local...)
			mu.Unlock()
		}()
	}
	wg.Wait()
	return ops, errs
}

type fiMix struct {
	n, creates, patches int
	secs                float64
	p50, p95, p99       float64
	createP95, patchP95 float64
}

func fiSummarize(ops []fiOp, from, to time.Time) fiMix {
	var all, cr, pa []time.Duration
	for _, o := range ops {
		if o.at.Before(from) || !o.at.Before(to) {
			continue
		}
		all = append(all, o.d)
		if o.kind == "create" {
			cr = append(cr, o.d)
		} else {
			pa = append(pa, o.d)
		}
	}
	return fiMix{n: len(all), creates: len(cr), patches: len(pa), secs: to.Sub(from).Seconds(),
		p50: pct(all, .5), p95: pct(all, .95), p99: pct(all, .99), createP95: pct(cr, .95), patchP95: pct(pa, .95)}
}

func (m fiMix) row(label string) string {
	return fmt.Sprintf("| %s | %.0f s | %.1f | %.1f | **%.1f** | %.1f | %.1f | %.1f |", label, m.secs, float64(m.n)/m.secs, m.p50, m.p95, m.p99, m.createP95, m.patchP95)
}

// liveFor runs the live mix alone for d after a 10 s warm-up.
func (e *fiEnv) liveFor(t *testing.T, clients int, d time.Duration, creates []perfReq, mine [][]uuid.UUID) (fiMix, map[string]int64) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), d+10*time.Second)
	defer cancel()
	start := time.Now().Add(10 * time.Second)
	ops, errs := e.liveMix(ctx, t, clients, creates, mine)
	return fiSummarize(ops, start, start.Add(d)), errs
}

// ---- the backfill ----------------------------------------------------------

// fiBatch is one Batch activity as the wrapper saw it.
type fiBatch struct {
	start time.Time
	d     time.Duration
}

// fiBackfill is the worker's view of one backfill at a time: the batches the
// wrapper timed, and an optional cap after which the harness cancels the
// workflow (a measured sample of a run too long to finish).
type fiBackfill struct {
	c      client.Client
	acts   *backfill.Activities
	mu     sync.Mutex
	batchs []fiBatch
	cap    int
	capped chan struct{}
}

func (b *fiBackfill) batch(ctx context.Context, in backfill.Cursor) (backfill.Cursor, error) {
	t0 := time.Now()
	out, err := b.acts.Batch(ctx, in)
	b.mu.Lock()
	b.batchs = append(b.batchs, fiBatch{t0, time.Since(t0)})
	if b.cap > 0 && len(b.batchs) == b.cap {
		close(b.capped)
	}
	b.mu.Unlock()
	return out, err
}

// startTemporal starts a Temporal dev server and a worker with the real
// FieldIndexBackfill and its activities on the nava_worker pool. Batch is
// registered through a wrapper that only times it.
func (e *fiEnv) startTemporal(t *testing.T) *fiBackfill {
	t.Helper()
	opts := testsuite.DevServerOptions{LogLevel: "error", ExistingPath: os.Getenv("NAVA_PERF_TEMPORAL_CLI")}
	srv, err := testsuite.StartDevServer(t.Context(), opts)
	if err != nil {
		t.Fatalf("temporal dev server: %v", err)
	}
	t.Cleanup(func() { _ = srv.Stop() })
	c := srv.Client()
	b := &fiBackfill{c: c, acts: &backfill.Activities{Pool: e.worker}}
	wk := worker.New(c, "fi-perf", worker.Options{})
	backfill.RegisterWorkflow(wk)
	wk.RegisterActivityWithOptions(b.acts.Barrier, activity.RegisterOptions{Name: backfill.BarrierActivity})
	wk.RegisterActivityWithOptions(b.acts.Classify, activity.RegisterOptions{Name: backfill.ClassifyActivity})
	wk.RegisterActivityWithOptions(b.batch, activity.RegisterOptions{Name: backfill.BatchActivity})
	wk.RegisterActivityWithOptions(b.acts.Ready, activity.RegisterOptions{Name: backfill.ReadyActivity})
	if err := wk.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(wk.Stop)
	return b
}

type fiRun struct {
	dur, barrier1    time.Duration
	batches          []fiBatch
	pauses           []time.Duration
	walBytes         int64
	start, end       time.Time
	state            string
	capped           bool
	missing, surplus int64
	sideRows         int64
	total            int64 // the type's records at the end
}

// runBackfill runs FieldIndexBackfill for field to completion, or, with
// maxBatches > 0, cancels it once that many batches have run. It reads the
// pauses from the workflow history and, for a run that finished, diffs the
// side table against records.data.
func (e *fiEnv) runBackfill(t *testing.T, b *fiBackfill, tenant, field uuid.UUID, version, maxBatches int) fiRun {
	t.Helper()
	wid := backfill.ChildID(tenant, fieldindex.BackfillWorkflowType, field, version)
	b.mu.Lock()
	b.batchs, b.cap, b.capped = nil, maxBatches, make(chan struct{})
	b.mu.Unlock()
	var lsn0 string
	if err := e.super.QueryRow(t.Context(), "SELECT pg_current_wal_lsn()::text").Scan(&lsn0); err != nil {
		t.Fatal(err)
	}
	r := fiRun{start: time.Now()}
	run, err := b.c.ExecuteWorkflow(t.Context(), client.StartWorkflowOptions{ID: wid, TaskQueue: "fi-perf",
		WorkflowIDReusePolicy: enumspb.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE},
		fieldindex.BackfillWorkflowType, backfill.Input{FieldID: field})
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- run.Get(t.Context(), nil) }()
	select {
	case err = <-done:
	case <-b.capped:
		r.capped = true
		if cerr := b.c.CancelWorkflow(t.Context(), wid, run.GetRunID()); cerr != nil {
			t.Fatal(cerr)
		}
		err = <-done
		if temporal.IsCanceledError(err) {
			err = nil
		}
	}
	if err != nil {
		t.Fatalf("backfill: %v", err)
	}
	r.end = time.Now()
	r.dur = r.end.Sub(r.start)
	if err := e.super.QueryRow(t.Context(), "SELECT pg_wal_lsn_diff(pg_current_wal_lsn(), $1::pg_lsn)::bigint", lsn0).Scan(&r.walBytes); err != nil {
		t.Fatal(err)
	}
	b.mu.Lock()
	r.batches = slices.Clone(b.batchs)
	b.cap = 0
	b.mu.Unlock()
	if len(r.batches) > 0 {
		r.barrier1 = r.batches[0].start.Sub(r.start)
	}
	it := b.c.GetWorkflowHistory(t.Context(), wid, run.GetRunID(), false, enumspb.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT)
	for it.HasNext() {
		ev, err := it.Next()
		if err != nil {
			t.Fatal(err)
		}
		if a := ev.GetTimerStartedEventAttributes(); a != nil {
			r.pauses = append(r.pauses, a.GetStartToFireTimeout().AsDuration())
		}
	}
	r.state = e.scalar(t, "SELECT state || ' ' || progress FROM record_index_fields WHERE id = $1", field)
	if err := e.super.QueryRow(t.Context(), `SELECT count(*) FROM records WHERE tenant_id = $1 AND entity_definition_id = $2`,
		tenant, e.largeDef).Scan(&r.total); err != nil {
		t.Fatal(err)
	}
	if r.capped {
		return r
	}
	// The diff: records whose priority has no equal side row, and side rows
	// with no record value equal to them.
	if err := e.super.QueryRow(t.Context(), `SELECT
		(SELECT count(*) FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND r.data ? 'priority'
			AND NOT EXISTS (SELECT 1 FROM rfv_text v WHERE v.tenant_id = r.tenant_id AND v.record_id = r.id AND v.field_id = $3
				AND v.value = r.data ->> 'priority')),
		(SELECT count(*) FROM rfv_text v WHERE v.tenant_id = $1 AND v.field_id = $3 AND NOT EXISTS (SELECT 1 FROM records r
			WHERE r.tenant_id = v.tenant_id AND r.id = v.record_id AND r.data ->> 'priority' = v.value)),
		(SELECT count(*) FROM rfv_text v WHERE v.tenant_id = $1 AND v.field_id = $3)`,
		tenant, e.largeDef, field).Scan(&r.missing, &r.surplus, &r.sideRows); err != nil {
		t.Fatal(err)
	}
	return r
}

func (e *fiEnv) reportRun(label string, r fiRun) {
	var ds []time.Duration
	var sum time.Duration
	for _, b := range r.batches {
		ds = append(ds, b.d)
		sum += b.d
	}
	avg, mx := time.Duration(0), time.Duration(0)
	if len(ds) > 0 {
		avg, mx = sum/time.Duration(len(ds)), slices.Max(ds)
	}
	var psum time.Duration
	hist := map[time.Duration]int{}
	for _, p := range r.pauses {
		psum += p
		hist[p]++
	}
	keys := make([]time.Duration, 0, len(hist))
	for k := range hist {
		keys = append(keys, k)
	}
	slices.SortFunc(keys, cmp.Compare)
	var hs []string
	for _, k := range keys {
		hs = append(hs, fmt.Sprintf("%s ×%d", k, hist[k]))
	}
	perBatch := 0.0
	if len(ds) > 0 {
		perBatch = float64(r.walBytes) / float64(len(ds)) / 1e6
	}
	dur := r.dur.Round(time.Second).String()
	diff := fmt.Sprintf("missing %d, surplus %d (%d side rows)", r.missing, r.surplus, r.sideRows)
	if r.capped {
		// Projected: the sampled time per batch (its pause included) times
		// the batches the type needs.
		need := (r.total + 999) / 1000
		per := (r.dur - r.barrier1) / time.Duration(max(len(ds), 1))
		dur = fmt.Sprintf("stopped after %d batches at %s; projected %s for %d batches", len(ds), r.dur.Round(time.Second),
			(r.barrier1 + per*time.Duration(need)).Round(time.Minute), need)
		diff = "not checked (stopped)"
	}
	mn := time.Duration(0)
	if len(ds) > 0 {
		mn = slices.Min(ds)
	}
	var seq []string
	for _, b := range r.batches {
		seq = append(seq, strconv.FormatInt(b.d.Milliseconds(), 10))
	}
	var pseq []string
	for _, p := range r.pauses {
		pseq = append(pseq, strconv.FormatFloat(p.Seconds(), 'f', -1, 64))
	}
	e.w("| %s | %s | %d | %.0f / %.0f / %.0f / %.0f / %.0f ms | %s (%s) | %s | %.0f MB (%.2f MB per batch) | %s | %s |",
		label, dur, len(ds), fiMS(mn), pct(ds, .5), fiMS(avg), pct(ds, .95), fiMS(mx), psum.Round(time.Second), strings.Join(hs, ", "),
		r.barrier1.Round(time.Millisecond), float64(r.walBytes)/1e6, perBatch, r.state, diff)
	e.seqs = append(e.seqs, fmt.Sprintf("- %s: batch ms in order: %s; pauses s in order: %s", label, strings.Join(seq, " "), strings.Join(pseq, " ")))
}

func fiMS(d time.Duration) float64 { return float64(d.Microseconds()) / 1000 }

// resetField empties field's side rows and sets it backfilling again at the
// same version, so the same child id pins it.
func (e *fiEnv) resetField(t *testing.T, tenant, field uuid.UUID) {
	t.Helper()
	e.exec(t, `DELETE FROM rfv_text WHERE tenant_id = $1 AND field_id = $2`, tenant, field)
	e.exec(t, `UPDATE record_index_fields SET state = 'backfilling', progress = 0 WHERE id = $1`, field)
	e.exec(t, "VACUUM (ANALYZE) rfv_text, record_index_fields")
	e.exec(t, "CHECKPOINT")
}

// fiLoadRun is one backfill under the live mix.
type fiLoadRun struct {
	r      fiRun
	mix    fiMix
	errs   map[string]int64
	capped bool
}

// underLoad runs the live mix, starts the backfill 10 s in, and stops the
// mix when the backfill ends or at liveMax. The mix is cut to the
// backfill's window.
func (e *fiEnv) underLoad(t *testing.T, b *fiBackfill, clients int, creates []perfReq, mine [][]uuid.UUID,
	tenant, field uuid.UUID, version, maxBatches int, liveMax time.Duration) fiLoadRun {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	var (
		ops  []fiOp
		out  fiLoadRun
		done = make(chan struct{})
	)
	go func() {
		defer close(done)
		ops, out.errs = e.liveMix(ctx, t, clients, creates, mine)
	}()
	time.Sleep(10 * time.Second)
	bf := make(chan fiRun, 1)
	go func() { bf <- e.runBackfill(t, b, tenant, field, version, maxBatches) }()
	select {
	case out.r = <-bf:
		cancel()
		<-done
	case <-time.After(liveMax):
		out.capped = true
		cancel()
		<-done
		out.r = <-bf
	}
	end := out.r.end
	if out.capped {
		end = out.r.start.Add(liveMax)
	}
	out.mix = fiSummarize(ops, out.r.start, end)
	return out
}

// fiBatchCap reads a backfill batch cap. Unlike perfEnvInt it keeps 0, which
// runs the backfill to the end and diffs its side rows.
func fiBatchCap(name string, def int) int {
	if v, err := strconv.Atoi(os.Getenv(name)); err == nil && v >= 0 {
		return v
	}
	return def
}

// backfillPhase: priority (text, 5 values) is flagged late through the real
// publish. FieldIndexBackfill then runs four times, each from no side rows:
// as merged, alone and under the live mix; then with the fix candidate,
// nava_worker at plan_cache_mode = force_custom_plan, alone and under the
// live mix. Each is stopped after NAVA_PERF_ASIS_BATCHES or
// NAVA_PERF_FIX_BATCHES batches and its whole duration projected: the
// adaptive pause reaches its 10 s ceiling within about 15 batches on the
// reference host, so a whole 400k run takes over an hour (see the report).
// With a cap of 0 a run goes to the end and its side rows are diffed. The live mix runs alone
// before and after: the controls the D10 delta is taken against.
func (e *fiEnv) backfillPhase(t *testing.T, clients int) {
	t.Helper()
	// The quota is 5 per type: leave 4 so the publish may add one.
	e.largeFieldsTo(t, []string{"tier", "zone", "score", "opened"})
	e.exec(t, `UPDATE record_index_fields SET state = 'ready' WHERE entity_definition_id = $1`, e.largeDef)
	schema := strings.Replace(perfSchema, `"properties": {`, `"properties": {`+fiSchemaExtra, 1)
	schema = strings.Replace(schema, `"priority": {"type": "string", "enum": ["1", "2", "3", "4", "5"]}`,
		`"priority": {"type": "string", "enum": ["1", "2", "3", "4", "5"], "x-nava-filterable": true}`, 1)
	body := perfTypeBodyWith(fiSchemaExtra)
	tail := body[strings.Index(body, `"fixed_columns"`):]
	tail = strings.TrimSuffix(strings.TrimSpace(tail), "}")
	pub := `{"definition": {"schema": ` + schema + `, ` + tail + `}`
	rec := e.serve(t.Context(), http.MethodPost, "/api/v1/entity-types/ticket/versions", "admin-large", uuid.NewString(), pub)
	if rec.Code != http.StatusCreated {
		t.Fatalf("publish priority flagged = %d: %s", rec.Code, rec.Body)
	}
	var field uuid.UUID
	var version int
	if err := e.super.QueryRow(t.Context(), `SELECT id, requested_version FROM record_index_fields
		WHERE entity_definition_id = $1 AND field_key = 'priority' AND state = 'backfilling'`, e.largeDef).Scan(&field, &version); err != nil {
		t.Fatalf("publish wrote no backfilling registry row for priority: %v", err)
	}
	tenant := e.large.id
	e.exec(t, "VACUUM (ANALYZE) records, rfv_text, rfv_num, rfv_ts, record_index_fields")
	e.exec(t, "CHECKPOINT")

	creates := e.fiCreateReqs(t)
	var mine [][]uuid.UUID
	rows, err := e.super.Query(t.Context(), `SELECT id FROM records WHERE tenant_id = $1 AND number <= 'TKT0400000'
		ORDER BY md5(id::text) LIMIT $2`, tenant, clients*500)
	if err != nil {
		t.Fatal(err)
	}
	all, err := pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
	if err != nil {
		t.Fatal(err)
	}
	for c := 0; c < clients; c++ {
		mine = append(mine, all[c*500:(c+1)*500])
	}
	live := time.Duration(perfEnvInt("NAVA_PERF_LIVE_SECONDS", 180)) * time.Second
	liveMax := time.Duration(perfEnvInt("NAVA_PERF_LIVE_MAX", 1200)) * time.Second
	asis := fiBatchCap("NAVA_PERF_ASIS_BATCHES", 40)
	fix := fiBatchCap("NAVA_PERF_FIX_BATCHES", 60)

	b := e.startTemporal(t)
	e.w("")
	e.w("## Backfill: priority (text, 5 values) flagged late by publish v2 on the 400k type; FieldIndexBackfill on a Temporal dev server, nava_worker")
	e.w("")
	e.w("Live mix: %d clients through the handlers, alternating POST (every band field set; tier, zone, score and opened flagged) and PATCH of score, as nava_app.", clients)
	e.w("")
	alone1, errs1 := e.liveFor(t, clients, live, creates, mine)

	e.exec(t, "CHECKPOINT")
	r1 := e.runBackfill(t, b, tenant, field, version, asis)
	e.resetField(t, tenant, field)
	l2 := e.underLoad(t, b, clients, creates, mine, tenant, field, version, asis, liveMax)

	// The fix candidate: nava_worker plans each statement with its own
	// values, as nava_app does (db/grants.yaml). The pool is reset so every
	// connection starts with the role setting.
	e.exec(t, "ALTER ROLE nava_worker SET plan_cache_mode = force_custom_plan")
	// The RESET runs once: inline before the closing control, and from
	// t.Cleanup if the test stops first, so a kept container (BENCH_KEEP=1)
	// never keeps the setting. t.Context() is done by cleanup time.
	var resetOnce sync.Once
	resetRole := func() {
		resetOnce.Do(func() {
			if _, err := e.super.Exec(context.Background(), "ALTER ROLE nava_worker RESET plan_cache_mode"); err != nil {
				t.Errorf("reset nava_worker plan_cache_mode: %v", err)
			}
			e.worker.Reset()
		})
	}
	t.Cleanup(resetRole)
	e.worker.Reset()
	e.resetField(t, tenant, field)
	r3 := e.runBackfill(t, b, tenant, field, version, fix)
	e.resetField(t, tenant, field)
	l4 := e.underLoad(t, b, clients, creates, mine, tenant, field, version, fix, liveMax)
	resetRole()
	alone2, errs5 := e.liveFor(t, clients, live, creates, mine)

	e.w("| Live mix | Window | ops/s | p50 ms | p95 ms | p99 ms | create p95 | patch p95 |")
	e.w("| --- | --- | --- | --- | --- | --- | --- | --- |")
	e.w("%s", alone1.row("alone, before"))
	e.w("%s", l2.mix.row("with the backfill as merged"))
	e.w("%s", l4.mix.row("with the backfill, nava_worker force_custom_plan"))
	e.w("%s", alone2.row("alone, after"))
	e.w("")
	for _, c := range []struct {
		label string
		l     fiLoadRun
	}{{"as merged", l2}, {"fix candidate", l4}} {
		if c.l.capped {
			e.w("The live mix under the %s backfill stopped at the %s cap; the backfill finished alone after it.", c.label, liveMax)
		}
	}
	e.w("Errors: before %v; as merged %v; fix candidate %v; after %v", errs1, l2.errs, l4.errs, errs5)
	e.w("")
	for _, c := range []struct {
		label string
		m     fiMix
	}{{"as merged", l2.mix}, {"fix candidate", l4.mix}} {
		delta := c.m.p95/alone1.p95 - 1
		deltaHi := c.m.p95/max(alone1.p95, alone2.p95) - 1
		verdict := "meets"
		if delta > fiLiveP95DeltaBudget {
			verdict = "**misses**"
		}
		e.w("**Live-write p95 delta, %s: %+.1f%% against the control before** (%+.1f%% against the larger control), budget +%.0f%%: %s.",
			c.label, 100*delta, 100*deltaHi, 100*fiLiveP95DeltaBudget, verdict)
	}
	e.w("")
	e.w("| Backfill | Duration | Batches | Batch min / p50 / avg / p95 / max | Pauses total (each) | First barrier | WAL | Registry | Diff |")
	e.w("| --- | --- | --- | --- | --- | --- | --- | --- | --- |")
	e.reportRun("as merged, alone", r1)
	e.reportRun("as merged, under the live mix", l2.r)
	e.reportRun("force_custom_plan, alone", r3)
	e.reportRun("force_custom_plan, under the live mix", l4.r)
	e.w("")
	for _, q := range e.seqs {
		e.w("%s", q)
	}
}

// probe runs the live mix for a few seconds and reports its errors: a
// harness check, no measurement.
func (e *fiEnv) probe(t *testing.T, clients int) {
	t.Helper()
	creates := e.fiCreateReqs(t)
	rows, err := e.super.Query(t.Context(), `SELECT id FROM records WHERE tenant_id = $1 AND number <= 'TKT0400000'
		ORDER BY md5(id::text) LIMIT $2`, e.large.id, clients*10)
	if err != nil {
		t.Fatal(err)
	}
	all, err := pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
	if err != nil {
		t.Fatal(err)
	}
	var mine [][]uuid.UUID
	for c := 0; c < clients; c++ {
		mine = append(mine, all[c*10:(c+1)*10])
	}
	m, errs := e.liveFor(t, clients, 5*time.Second, creates, mine)
	e.w("- probe: %d creates, %d patches, errors %v", m.creates, m.patches, errs)
}

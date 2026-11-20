//go:build integration && perf

// TICK-1022: the >= 40% ACL list after a backfill, under the side-table
// statistics and visibility map a backfill can leave behind.
//
// The S21 kit (TICK-749) saw `filter[priority]=3` (48% of the type) as a
// narrowing agent read 217 ms single-client on one fresh stack and 34 ms on
// the next, where the sibling value 4 (50%) read 61 / 39 ms. The D13 cell was
// measured after VACUUM (ANALYZE) of the side tables (flagForLists); a real
// backfill leaves whatever autovacuum last did. This test copies one num
// field's side rows in the backfill's order (record id) and lists under
// each state a backfill can end in, with autovacuum off on rfv_num so the
// state is the one set up:
//
//	none     no statistics on rfv_num at all (no autoanalyze yet)
//	mid      ANALYZE at 50% of the rows (an autoanalyze mid-backfill)
//	late     ANALYZE at 91% (the last autoanalyze before the flip; S21 stack 2)
//	analyze  ANALYZE after the last row (the fix candidate), heap not vacuumed
//	vacuum   VACUUM (ANALYZE) after the last row (as D13 was measured)
//	vmNN     vacuum, then pg_class.relallvisible set to NN% of the pages
//	         (the planner's input alone; the map stays complete)
//	churnNN  vacuum, then NN% of the rows deleted and inserted again (record
//	         edits between two autovacuums), then ANALYZE
//	rpc      analyze, planned with random_page_cost 1.1 for nava_app
//
// Field pk (num), from an md5 of (tenant, n), the kit's distribution:
// 1 at 1%, 2 at 1%, 3 at 48%, 4 at 50%.
//
//	go test -tags 'integration perf' -count=1 -timeout 120m -v \
//	    -run '^TestFieldIndexPostBackfillStats$' ./internal/server
//
// with NAVA_TEST_PG_URL as scratch/field-index-measured/run.sh sets it (or
// scratch/tick-1022-post-backfill-stats/run.sh). NAVA_PERF_LIST_SECONDS (10)
// per cell; NAVA_PERF_STATS_STATES picks the states (comma list, default
// none..vacuum); NAVA_PERF_STATS_QUERIES keeps the first n filters;
// NAVA_PERF_STATS_P95=0 skips the 8-client cells.

package server_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

const statsSchemaExtra = `"pk": {"type": "number"},
    `

func statsDataExtra() string {
	u := fiU("k")
	return fmt.Sprintf(`,
		'pk', CASE WHEN %[1]s < 10000 THEN 1 WHEN %[1]s < 20000 THEN 2 WHEN %[1]s < 500000 THEN 3 ELSE 4 END`, u)
}

func TestFieldIndexPostBackfillStats(t *testing.T) {
	clients := perfEnvInt("NAVA_PERF_CLIENTS", 8)
	out := os.Getenv("NAVA_PERF_OUT")
	if out == "" {
		out = filepath.Join("..", "..", "bin", "tick-1022-stats-raw.md")
	}
	var buf strings.Builder
	w := func(format string, args ...any) {
		line := fmt.Sprintf(format, args...)
		buf.WriteString(line + "\n")
		t.Log(line)
		_ = os.MkdirAll(filepath.Dir(out), 0o750)
		_ = os.WriteFile(out, []byte(buf.String()), 0o600)
	}
	pe := newPerfEnv(t, clients)
	pe.schemaExtra, pe.dataExtra = statsSchemaExtra, statsDataExtra()
	e := &fiEnv{perfEnv: pe, w: w, errSample: map[string]string{}}
	w("# TICK-1022 post-backfill statistics, raw (%s)", time.Now().UTC().Format(time.RFC3339))
	w("")
	t0 := time.Now()
	e.seed(t)
	e.exec(t, "VACUUM (ANALYZE)")
	e.exec(t, "CHECKPOINT")
	w("- seed: 800k records + VACUUM (ANALYZE) + CHECKPOINT in %s", time.Since(t0).Round(time.Second))
	w("- %s", e.scalar(t, "SELECT version()"))
	w("- settings: %s", e.scalar(t, `SELECT string_agg(name || '=' || setting || coalesce(unit, ''), ', ' ORDER BY name) FROM pg_settings
		WHERE name IN ('shared_buffers','effective_cache_size','work_mem','random_page_cost','max_parallel_workers_per_gather','jit','default_statistics_target')`))
	w("- other containers at start: %v", foreignContainers(t.Context()))
	if err := e.super.QueryRow(t.Context(), `SELECT d.id FROM entity_definitions d JOIN perf_tenant p ON p.tenant_id = d.tenant_id
		WHERE p.tn = 0 AND d.key = 'ticket'`).Scan(&e.largeDef); err != nil {
		t.Fatal(err)
	}
	w("- large type pk: %s", e.scalar(t, `SELECT string_agg(v || ' ' || n, ', ' ORDER BY v) FROM (
		SELECT data->>'pk' v, count(*) n FROM records WHERE tenant_id = $1 AND entity_definition_id = $2 GROUP BY 1) x`, e.large.id, e.largeDef))

	field := uuid.New()
	e.exec(t, `INSERT INTO record_index_fields (tenant_id, id, entity_definition_id, field_key, value_type, state, progress, requested_version)
		VALUES ($1, $2, $3, 'pk', 'num', 'ready', 100, 1)`, e.large.id, field, e.largeDef)
	e.exec(t, "ALTER TABLE rfv_num SET (autovacuum_enabled = false)")

	states := []string{"none", "mid", "late", "analyze", "vacuum"}
	if v := os.Getenv("NAVA_PERF_STATS_STATES"); v != "" {
		states = strings.Split(v, ",")
	}
	d := time.Duration(perfEnvInt("NAVA_PERF_LIST_SECONDS", 10)) * time.Second
	queries := []struct{ label, q string }{
		{"pk=3 (48%)", "filter%5Bpk%5D=3"},
		{"pk=4 (50%)", "filter%5Bpk%5D=4"},
		{"pk=1 (1%)", "filter%5Bpk%5D=1"},
	}
	if n := perfEnvInt("NAVA_PERF_STATS_QUERIES", len(queries)); n < len(queries) {
		queries = queries[:n]
	}
	p95 := os.Getenv("NAVA_PERF_STATS_P95") != "0"
	var explains []string
	w("")
	w("## Lists as a narrowing agent: single-client p50 / 8-client p95, ms")
	w("")
	w("| State | rfv_num stats | %s |", strings.Join(func() []string {
		var s []string
		for _, q := range queries {
			s = append(s, q.label)
		}
		return s
	}(), " | "))
	w("| --- | --- |%s", strings.Repeat(" --- |", len(queries)))
	for _, st := range states {
		e.backfillTo(t, field, st)
		stats := e.scalar(t, `SELECT coalesce(c.reltuples::bigint::text, '?') || ' reltuples, ' || c.relpages || ' pages, ' ||
			coalesce((SELECT round(100.0 * relallvisible / nullif(relpages, 0)) FROM pg_class WHERE oid = c.oid)::text, '0') || '% all-visible, ' ||
			coalesce((SELECT array_to_string(most_common_vals::text::numeric[], '/') || ' @ ' || array_to_string(most_common_freqs, '/')
			            FROM pg_stats WHERE tablename = 'rfv_num' AND attname = 'value'), 'no value stats')
			FROM pg_class c WHERE c.relname = 'rfv_num'`)
		row := "| " + st + " | " + stats + " |"
		for _, q := range queries {
			one := e.run(t, perfScenario{name: st + " " + q.label + " 1c", kind: "data", tenant: "large", reqs: listReqs(e.agents, q.q)}, 1, d)
			many := "-"
			if p95 {
				r := e.run(t, perfScenario{name: st + " " + q.label + " 8c", kind: "data", tenant: "large", reqs: listReqs(e.agents, q.q)}, clients, d)
				many = fmt.Sprintf("%.1f", pct(r.lat, .95))
			}
			row += fmt.Sprintf(" %.1f / %s |", pct(one.lat, .5), many)
			for i, s := range one.stmtSQL {
				if strings.Contains(s.SQL, "rfv_") && i < len(one.explains) && one.explains[i] != "" {
					explains = append(explains, "### "+st+" "+q.label+"\n\n```sql\n"+strings.TrimSpace(s.SQL)+"\n```\n\n```\n"+one.explains[i]+"\n```")
				}
			}
		}
		w("%s", row)
		if st == "rpc" {
			e.exec(t, "ALTER ROLE nava_app RESET random_page_cost")
			e.pool.Reset()
		}
	}
	p := strings.TrimSuffix(out, ".md") + "-explain.md"
	if err := os.WriteFile(p, []byte("# TICK-1022 list plans (single run per cell, as nava_app)\n\n"+strings.Join(explains, "\n\n")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Logf("wrote %s and %s", out, p)
}

// backfillTo empties rfv_num and its statistics, then copies pk's side rows
// in the backfill's order (record id) and leaves the table in state st.
func (e *fiEnv) backfillTo(t *testing.T, field uuid.UUID, st string) {
	t.Helper()
	e.exec(t, "TRUNCATE rfv_num")
	e.exec(t, "DELETE FROM pg_statistic WHERE starelid = 'rfv_num'::regclass")
	e.exec(t, `DELETE FROM pg_statistic_ext_data WHERE stxoid IN (SELECT oid FROM pg_statistic_ext WHERE stxrelid = 'rfv_num'::regclass)`)
	copyUpTo := func(fromPct, toPct int) {
		e.exec(t, "SET session_replication_role = replica")
		e.exec(t, `INSERT INTO rfv_num (tenant_id, field_id, record_id, ord, value)
			SELECT r.tenant_id, $2, r.id, c.ord, c.value_num FROM (
				SELECT r.*, ntile(100) OVER (ORDER BY r.id) AS pc FROM records r
				WHERE r.tenant_id = $1 AND r.entity_definition_id = $3) r,
			LATERAL record_field_coerce(r.data -> 'pk', 'num') c
			WHERE r.pc > $4 AND r.pc <= $5 ORDER BY r.id`, e.large.id, field, e.largeDef, fromPct, toPct)
		e.exec(t, "SET session_replication_role = origin")
	}
	switch st {
	case "none":
		copyUpTo(0, 100)
	case "mid", "late":
		at := 50
		if st == "late" {
			at = 91
		}
		copyUpTo(0, at)
		e.exec(t, "ANALYZE rfv_num")
		copyUpTo(at, 100)
	case "analyze":
		copyUpTo(0, 100)
		e.exec(t, "ANALYZE rfv_num")
	case "vacuum":
		copyUpTo(0, 100)
		e.exec(t, "VACUUM (ANALYZE) rfv_num")
	case "rpc":
		// The heap as "analyze" leaves it, planned with random_page_cost 1.1.
		copyUpTo(0, 100)
		e.exec(t, "ANALYZE rfv_num")
		e.exec(t, "ALTER ROLE nava_app SET random_page_cost = 1.1")
		e.pool.Reset()
	default:
		var n int
		switch {
		case strings.HasPrefix(st, "vm"):
			// VACUUM (ANALYZE), then the planner told only n% of the pages are
			// all-visible (pg_class.relallvisible, which the planner reads; the
			// visibility map itself stays complete, so the time is the plan's).
			if _, err := fmt.Sscanf(st, "vm%d", &n); err != nil {
				t.Fatalf("state %q: %v", st, err)
			}
			copyUpTo(0, 100)
			e.exec(t, "VACUUM (ANALYZE) rfv_num")
			e.exec(t, "UPDATE pg_class SET relallvisible = relpages * $1 / 100 WHERE oid = 'rfv_num'::regclass", n)
		case strings.HasPrefix(st, "churn"):
			// VACUUM (ANALYZE), then n% of the rows deleted and inserted again,
			// as record edits do through the trigger, then ANALYZE: the
			// visibility map a steady state between two autovacuums has.
			if _, err := fmt.Sscanf(st, "churn%d", &n); err != nil {
				t.Fatalf("state %q: %v", st, err)
			}
			copyUpTo(0, 100)
			e.exec(t, "VACUUM (ANALYZE) rfv_num")
			e.exec(t, `WITH d AS (DELETE FROM rfv_num WHERE ('x' || substr(md5(record_id::text), 1, 7))::bit(28)::int % 100 < $1
				RETURNING tenant_id, field_id, record_id, ord, value)
				INSERT INTO rfv_num SELECT * FROM d`, n)
			e.exec(t, "ANALYZE rfv_num")
		default:
			t.Fatalf("unknown state %q", st)
		}
	}
	e.exec(t, "CHECKPOINT")
	e.exec(t, `SELECT pg_prewarm(c.oid) FROM pg_class c WHERE c.relname LIKE 'rfv\_num%' AND c.relkind IN ('r', 'i')`)
}

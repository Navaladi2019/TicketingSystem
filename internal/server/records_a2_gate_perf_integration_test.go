//go:build integration && perf

// ADR-006-A2 A2-8 (2), (2a) and (3a) (TICK-1048, PLAN-007-A2 T-A2-4), on the
// T-A2-2 code (TICK-1046): the list handler with CountBudget in force.
//
// Data: P3-36's seed (800k records; one tenant with 400k tickets), so that
// P3-36's host gates read against the same data P3-36 scored: the transport
// floor (SELECT 1, 8 clients, p50 <= 3 ms) and the no-filter agent list
// control (8 clients, p95 <= 1.1 x 73 ms = 80.3 ms). On top, the S21 kit's
// shape (docs/acceptance/S21/tools/seed.sql) on the large tenant: a num field
// kp, flagged and ready, at the kit's priority distribution 1/1/48/50%. The
// agent ACL is P3-36's (agents group plus 3 of 50 assignment groups: about
// 6% of the type visible); the kit's Alice sees about 5% through one group.
//
//  1. Gate attempt (up to NAVA_PERF_GATE_ATTEMPTS, 2): floor and control
//     before; band (ii), filter[kp]=3 as the agents, 8 clients for
//     NAVA_PERF_LIST_SECONDS (15) with the count skipped by WithSkipCount;
//     the same cell with the count (reported only); floor and control after.
//     An attempt counts only if all four gate readings pass.
//  2. Spike S-A2-cancel: PR #812's fixture plus pw (multi-valued, anti-correlated) (a6 sees 6%, a50 50%; num pk,
//     text pt, bool pb; autovacuum off on the side tables) in visibility-map
//     state "none" (ANALYZE, no VACUUM). Each candidate count is first run
//     unbudgeted (EXPLAIN (ANALYZE, BUFFERS), 3 times); the slowest is the
//     spike's cell. Then 100 single-client requests and one 8-client run
//     (NAVA_PERF_SPIKE_PER_CLIENT, 25, each) through the real handler. The
//     count phase is from the start of the SAVEPOINT Exec to the end of the
//     ROLLBACK TO Exec, as the pgx tracer sees it. pg_stat_database
//     temp_files / temp_bytes before, after and 60 s after.
//
//	sh scratch/tick-1048-a2-gate/run.sh
//
// Report: NAVA_PERF_OUT (raw.md) and, beside it, band-samples.txt,
// spike-samples.csv and plans.md.

package server_test

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
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

	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/records/store"
	"github.com/Navaladi2019/TicketingSystem/internal/server"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// gateSchemaExtra: kp is the kit's priority; pk, pt, pb are PR #812's.
const gateSchemaExtra = `"kp": {"type": "integer"}, "pk": {"type": "number"}, "pt": {"type": "string", "maxLength": 10}, "pb": {"type": "boolean"}, "pw": {"type": "number"},
    `

// gateControlMS is P3-36's idle no-filter list p95, and the host rule's limit.
const (
	gateControlMS    = 73.0
	gateControlLimit = 1.1 * gateControlMS
	gateBudgetMS     = 94.0
	gateOvershootMS  = 15.0
)

func gateDataExtra() string {
	h := fiU("kp") + " % 100"
	return `, 'kp', CASE WHEN ` + h + ` = 0 THEN 1 WHEN ` + h + ` = 1 THEN 2 WHEN ` + h + ` < 50 THEN 3 ELSE 4 END`
}

// gateHost is one reading of P3-36's two gates.
type gateHost struct {
	floorP50, controlP95 float64
}

func (g gateHost) ok() bool { return g.floorP50 <= perfFloorGateMS && g.controlP95 <= gateControlLimit }

func (g gateHost) String() string {
	v := func(ok bool) string {
		if ok {
			return "pass"
		}
		return "FAIL"
	}
	return fmt.Sprintf("floor p50 %.2f ms (%s, gate %.0f), control p95 %.1f ms (%s, gate %.1f)",
		g.floorP50, v(g.floorP50 <= perfFloorGateMS), perfFloorGateMS, g.controlP95, v(g.controlP95 <= gateControlLimit), gateControlLimit)
}

// gateSpike is one S-A2-cancel request.
type gateSpike struct {
	run              string
	code, items      int
	state            string
	totalNull        bool
	phase, countStmt float64 // ms; phase leaves out set
	set              float64 // ms: TICK-1083's injected SET LOCAL round trip, -1 if none
	latency          float64 // ms, client side
	after            []string
}

func TestA2Gate(t *testing.T) {
	clients := perfEnvInt("NAVA_PERF_CLIENTS", 8)
	out := os.Getenv("NAVA_PERF_OUT")
	if out == "" {
		out = filepath.Join("..", "..", "bin", "tick-1048-a2-gate", "raw.md")
	}
	dir := filepath.Dir(out)
	_ = os.MkdirAll(dir, 0o750)
	var buf strings.Builder
	w := func(format string, args ...any) {
		line := fmt.Sprintf(format, args...)
		buf.WriteString(line + "\n")
		t.Log(line)
		_ = os.WriteFile(out, []byte(buf.String()), 0o600)
	}
	var plans []string
	writePlans := func() {
		_ = os.WriteFile(filepath.Join(dir, "plans.md"), []byte("# TICK-1048 / TICK-1083 / TICK-1098 plans (EXPLAIN (ANALYZE, BUFFERS, SETTINGS) as nava_app)\n\n"+strings.Join(plans, "\n\n")+"\n"), 0o600)
	}
	keepPlans := func(r *perfResult) {
		for i, s := range r.stmtSQL {
			if i < len(r.explains) && r.explains[i] != "" && strings.Contains(s.SQL, "rfv_") {
				plans = append(plans, "### "+r.sc.name+"\n\n```sql\n"+strings.TrimSpace(s.SQL)+"\n```\n\n```\n"+r.explains[i]+"\n```")
			}
		}
		writePlans()
	}

	pe := newPerfEnv(t, clients)
	pe.schemaExtra, pe.dataExtra = gateSchemaExtra, gateDataExtra()
	e := &fiEnv{perfEnv: pe, w: w, errSample: map[string]string{}}
	normal := e.h
	skip := e.handlerWith(t, server.WithSkipCount())

	w("# TICK-1048 / TICK-1083 A2-8 (2), (2a), (3a), raw (%s)", time.Now().UTC().Format(time.RFC3339))
	w("")
	t0 := time.Now()
	e.seed(t)
	e.exec(t, "VACUUM (ANALYZE)")
	e.exec(t, "CHECKPOINT")
	w("- seed: 800k records + VACUUM (ANALYZE) + CHECKPOINT in %s", time.Since(t0).Round(time.Second))
	w("- %s", e.scalar(t, "SELECT version()"))
	w("- settings: %s", e.scalar(t, `SELECT string_agg(name || '=' || setting || coalesce(unit, ''), ', ' ORDER BY name) FROM pg_settings
		WHERE name IN ('shared_buffers','effective_cache_size','work_mem','random_page_cost','max_parallel_workers_per_gather','jit','fsync','synchronous_commit','default_statistics_target')`))
	w("- nava_app: %s", e.scalar(t, `SELECT coalesce((SELECT array_to_string(setconfig, ',') FROM pg_db_role_setting s JOIN pg_roles r ON r.oid = s.setrole WHERE r.rolname = 'nava_app'), 'default')`))
	w("- CountBudget %s; other containers at start: %v", store.CountBudget, foreignContainers(t.Context()))
	w("- parallel settings (server): %s", e.scalar(t, `SELECT string_agg(name || ' = ' || setting, ', ' ORDER BY name) FROM pg_settings
		WHERE name IN ('max_parallel_workers_per_gather', 'max_parallel_workers', 'max_worker_processes')`))
	want := map[string]bool{}
	for _, c := range strings.Split(cmp.Or(os.Getenv("NAVA_PERF_SPIKE_CELLS"), "base,spill,parallel"), ",") {
		want[strings.TrimSpace(c)] = true
	}
	if err := e.super.QueryRow(t.Context(), `SELECT d.id FROM entity_definitions d JOIN perf_tenant p ON p.tenant_id = d.tenant_id
		WHERE p.tn = 0 AND d.key = 'ticket'`).Scan(&e.largeDef); err != nil {
		t.Fatal(err)
	}
	e.flagForLists(t, []fiField{{"kp", "num"}}, true)
	large := e.large.id
	w("- kp on the large type: %s", e.scalar(t, `SELECT string_agg(v || ': ' || n, ', ' ORDER BY v) FROM (SELECT data->>'kp' v, count(*) n FROM records
		WHERE tenant_id = $1 AND entity_definition_id = $2 GROUP BY 1) x`, large, e.largeDef))
	w("- kp = 3 visible to agent-1 (desk rule groups): %s", e.scalar(t, `SELECT count(*) FROM records WHERE tenant_id = $1 AND entity_definition_id = $2
		AND data->>'kp' = '3' AND assignment_group_id = ANY($3)`, large, e.largeDef, e.agentGroups["agent-1"]))

	// ---- (2), (2a) ----
	d := time.Duration(perfEnvInt("NAVA_PERF_LIST_SECONDS", 15)) * time.Second
	attempts := 2
	if v, err := strconv.Atoi(os.Getenv("NAVA_PERF_GATE_ATTEMPTS")); err == nil && v >= 0 {
		attempts = v
	}
	host := func(label string) gateHost {
		fl := e.transportFloor(t, clients)
		e.h = normal
		c := e.run(t, perfScenario{name: label + ": control, agent list, no filter", kind: "list", tenant: "large", reqs: listReqs(e.agents, "")}, clients, d)
		g := gateHost{floorP50: pct(fl, .5), controlP95: pct(c.lat, .95)}
		w("- %s: %s; control n = %d, p50 %.1f ms; other containers: %v", label, g, len(c.lat), pct(c.lat, .5), foreignContainers(t.Context()))
		return g
	}
	band := "filter%5Bkp%5D=3"
	verdict := "not run (NAVA_PERF_GATE_ATTEMPTS=0)"
	samples := &strings.Builder{}
	for a := 1; a <= attempts; a++ {
		w("")
		w("## Gate attempt %d", a)
		w("")
		before := host("before")
		e.h = skip
		r := e.run(t, perfScenario{name: fmt.Sprintf("attempt %d: band (ii) kp=3, agents, count skipped", a), kind: "data", tenant: "large", reqs: listReqs(e.agents, band)}, clients, d)
		r1 := e.run(t, perfScenario{name: fmt.Sprintf("attempt %d: band (ii) kp=3, agent-1 only, count skipped", a), kind: "data", tenant: "large", reqs: listReqs([]string{"agent-1"}, band)}, clients, d)
		e.h = normal
		rc := e.run(t, perfScenario{name: fmt.Sprintf("attempt %d: band (ii) kp=3, agents, with the budgeted count", a), kind: "data", tenant: "large", reqs: listReqs(e.agents, band)}, clients, d)
		after := host("after")
		valid := before.ok() && after.ok()
		p95 := pct(r.lat, .95)
		w("- **non-count p95 (count skipped), agents: %.1f ms** (n = %d, p50 %.1f, p99 %.1f); agent-1 only: %.1f ms (n = %d)",
			p95, len(r.lat), pct(r.lat, .5), pct(r.lat, .99), pct(r1.lat, .95), len(r1.lat))
		w("- with the budgeted count, agents: p95 %.1f ms (n = %d, p50 %.1f); reported only", pct(rc.lat, .95), len(rc.lat), pct(rc.lat, .5))
		fmt.Fprintf(samples, "# attempt %d, %s, valid host %v: client-side ms, in arrival order per client\n", a, r.sc.name, valid)
		for _, l := range r.lat {
			fmt.Fprintf(samples, "%.3f\n", fiMS(l))
		}
		_ = os.WriteFile(filepath.Join(dir, "band-samples.txt"), []byte(samples.String()), 0o600)
		for _, x := range []*perfResult{r, rc} {
			keepPlans(x)
			var ss []string
			for _, s := range x.stmtSQL {
				ss = append(ss, fmt.Sprintf("%.2f ms %s", fiMS(s.Dur), firstLine(normSQL(s.SQL))[:min(70, len(firstLine(normSQL(s.SQL))))]))
			}
			w("- statements of one request (%s): %s", x.sc.name, strings.Join(ss, " | "))
		}
		if !valid {
			verdict = fmt.Sprintf("inconclusive (attempt %d: host gate failed)", a)
			w("- **attempt %d inconclusive: a host gate failed; not fed into the formula**", a)
			continue
		}
		if p95 <= gateBudgetMS {
			verdict = fmt.Sprintf("PASS: %.1f ms <= %.0f ms (attempt %d)", p95, gateBudgetMS, a)
		} else {
			b := 350 - p95 - 6 - 35
			b = float64(int(b/10) * 10)
			verdict = fmt.Sprintf("FAIL: %.1f ms > %.0f ms; A2-2 rerun: 350 - %.1f - 6 - 35 rounded down to 10 ms = %.0f ms (floor 100)", p95, gateBudgetMS, p95, max(b, 100))
		}
		break
	}
	w("")
	w("**(2) verdict: %s**", verdict)

	// ---- (3e) serial-count check, band (ii) cell (TICK-1098) ----
	// It runs before the S-A2-cancel fixture, which reassigns every large-
	// tenant record to the fixture's groups and so empties the agents' view.
	var serial *e3Serial
	if want["serial"] {
		serial = &e3Serial{e: e, tenant: large, clients: clients, d: d, host: host, w: w, plans: &plans, onPlans: writePlans,
			arms: map[string]e3Arm{
				"A": {key: "A", desc: fmt.Sprintf("shipped: CountBudget %s, workers off in the savepoint Exec", store.CountBudget), h: normal},
				"B": {key: "B", desc: "WithCountArm(140 ms, true): server settings", h: e.handlerWith(t, server.WithCountArm(140*time.Millisecond, true))},
			}}
		w("")
		w("## (3e) serial-count check (ADR-006-E2 E2-4)")
		w("")
		w("- arm A: %s; arm B: %s. Both on e.pool (perfTracer only): no gateCountTracer, no forced GUC.", serial.arms["A"].desc, serial.arms["B"].desc)
		w("- %d clients, blocks of %s, order A, B, B, A per cell; each cell is its own ABBA (band (ii) before the S-A2-cancel fixture, a50 after it). The (2a) host gate runs before and after each block; adjacent blocks share the reading between them. A block with a failed gate is rerun, up to 2 reruns.", clients, d)
		w("- band (ii) is TestA2Gate's: `filter[kp]=3` (48%% of the type) round robin over P3-36's 200 agents (about 6%% of the type each). The kit's Alice sees about 5%% through one group; this ACL is the harness's documented stand-in (docs/data/adr-006-a2-gate.md).")
		serial.cell(t, "band (ii) kp = 3, agents", listReqs(e.agents, band), true)
	}

	// ---- (3a) S-A2-cancel ----
	w("")
	w("## S-A2-cancel")
	w("")
	t1 := time.Now()
	var agentsGroup uuid.UUID
	if err := e.super.QueryRow(t.Context(), `SELECT agents FROM perf_tenant WHERE tn = 0`).Scan(&agentsGroup); err != nil {
		t.Fatal(err)
	}
	g6, g50, gRest := uuid.New(), uuid.New(), uuid.New()
	for _, g := range []struct {
		id   uuid.UUID
		name string
	}{{g6, "a2-g6"}, {g50, "a2-g50"}, {gRest, "a2-rest"}} {
		e.exec(t, "INSERT INTO groups (tenant_id, id, keycloak_group_id, path) VALUES ($1, $2, $3, $4)", large, g.id, g.name, "/t/"+large.String()+"/"+g.name)
	}
	e.tokens["a6"] = identity.Principal{TenantID: large, PrincipalID: uuid.New(), Groups: []uuid.UUID{agentsGroup, g6}}
	e.tokens["a50"] = identity.Principal{TenantID: large, PrincipalID: uuid.New(), Groups: []uuid.UUID{agentsGroup, g50}}
	for _, tb := range []string{"rfv_num", "rfv_text", "rfv_bool", "rfv_ts"} {
		e.exec(t, "ALTER TABLE "+tb+" SET (autovacuum_enabled = false)")
	}
	e.exec(t, "SET session_replication_role = replica")
	e.exec(t, `CREATE TEMP TABLE a2_rec AS SELECT id,
			(('x' || substr(md5(id::text || 'b'), 1, 7))::bit(28)::int) % 1000 AS b,
			(('x' || substr(md5(id::text || 'h'), 1, 7))::bit(28)::int) % 100 AS h,
			(('x' || substr(md5(id::text || 'k'), 1, 7))::bit(28)::int) AS k
		FROM records WHERE tenant_id = $1 AND entity_definition_id = $2`, large, e.largeDef)
	e.exec(t, `UPDATE records r SET assignment_group_id = CASE WHEN a.b < 60 THEN $2::uuid WHEN a.b < 560 THEN $3::uuid ELSE $4::uuid END
		FROM a2_rec a WHERE r.tenant_id = $1 AND r.id = a.id`, large, g6, g50, gRest)
	e.exec(t, `CREATE TEMP TABLE a2_pk AS
		WITH s AS (
			SELECT a.*, row_number() OVER (PARTITION BY (a.b >= 560 AND a.h BETWEEN 27 AND 98) ORDER BY a.id) AS step_rn,
			       row_number() OVER (PARTITION BY (a.b >= 560 AND a.h = 99) ORDER BY a.id) AS skew_rn
			FROM a2_rec a)
		SELECT id, CASE
			WHEN b >= 560 AND h < 15 THEN 10
			WHEN b >= 560 AND h BETWEEN 15 AND 26 THEN 20
			WHEN b < 60 AND k % 48 = 0 THEN 20
			WHEN b >= 560 AND h BETWEEN 27 AND 98 AND step_rn <= 19999 THEN 30
			WHEN b >= 560 AND h BETWEEN 27 AND 98 AND step_rn <= 39999 THEN 31
			WHEN b >= 560 AND h = 99 AND skew_rn <= 50 THEN 77
			WHEN k % 2 = 0 THEN 3 ELSE 4 END::float8 AS v,
			(k / 2) % 2 = 0 AS tx, (k / 4) % 2 = 0 AS bx
		FROM s`)
	fields := map[string]uuid.UUID{}
	for _, f := range []struct{ key, vt string }{{"pk", "num"}, {"pt", "text"}, {"pb", "bool"}} {
		id := uuid.New()
		fields[f.key] = id
		e.exec(t, `INSERT INTO record_index_fields (tenant_id, id, entity_definition_id, field_key, value_type, state, progress, requested_version)
			VALUES ($1, $2, $3, $4, $5, 'ready', 100, 1)`, large, id, e.largeDef, f.key, f.vt)
	}
	e.exec(t, `INSERT INTO rfv_num (tenant_id, field_id, record_id, ord, value) SELECT $1, $2, id, 0, v FROM a2_pk ORDER BY id`, large, fields["pk"])
	e.exec(t, `INSERT INTO rfv_text (tenant_id, field_id, record_id, ord, value) SELECT $1, $2, id, 0, CASE WHEN tx THEN 'x' ELSE 'y' END FROM a2_pk ORDER BY id`, large, fields["pt"])
	e.exec(t, `INSERT INTO rfv_bool (tenant_id, field_id, record_id, ord, value) SELECT $1, $2, id, 0, bx FROM a2_pk ORDER BY id`, large, fields["pb"])
	// pw (TICK-1048 runs 3-4): 1 on every record no fixture agent sees (~176k;
	// four values each, ord 0-3, from run 4),
	// plus 100 that a50 sees and 100 that a6 sees, so the count can't stop at
	// the first page or the 1001 cap.
	pw := uuid.New()
	e.exec(t, `INSERT INTO record_index_fields (tenant_id, id, entity_definition_id, field_key, value_type, state, progress, requested_version)
		VALUES ($1, $2, $3, 'pw', 'num', 'ready', 100, 1)`, large, pw, e.largeDef)
	e.exec(t, `INSERT INTO rfv_num (tenant_id, field_id, record_id, ord, value)
		SELECT $1, $2, id, o, 1 FROM (
			SELECT id, generate_series(0, 3) AS o FROM a2_rec WHERE b >= 560
			UNION ALL (SELECT id, 0 FROM a2_rec WHERE b >= 60 AND b < 560 ORDER BY id LIMIT 100)
			UNION ALL (SELECT id, 0 FROM a2_rec WHERE b < 60 ORDER BY id LIMIT 100)) x ORDER BY id, o`, large, pw)
	e.exec(t, "SET session_replication_role = origin")
	e.exec(t, "VACUUM (ANALYZE) records, groups")
	// Visibility-map state "none": ANALYZE, no VACUUM.
	e.exec(t, "ANALYZE rfv_num")
	e.exec(t, "ANALYZE rfv_text")
	e.exec(t, "ANALYZE rfv_bool")
	e.exec(t, "CHECKPOINT")
	e.exec(t, `SELECT pg_prewarm(c.oid) FROM pg_class c WHERE c.relname LIKE 'rfv\_%' AND c.relkind IN ('r', 'i')`)
	w("- fixtures in %s: pk values %s; visible a6 %s, a50 %s", time.Since(t1).Round(time.Second),
		e.scalar(t, `SELECT string_agg(v || ':' || n, ' ' ORDER BY v) FROM (SELECT v, count(*) n FROM a2_pk GROUP BY v) x`),
		e.scalar(t, `SELECT count(*) FROM records WHERE tenant_id = $1 AND assignment_group_id = $2`, large, g6),
		e.scalar(t, `SELECT count(*) FROM records WHERE tenant_id = $1 AND assignment_group_id = $2`, large, g50))
	w("- visibility map: %s (the kp rows of rfv_num were vacuumed before the fixture; the pk rows were not)",
		e.scalar(t, `SELECT string_agg(relname || ' ' || round(100.0 * relallvisible / nullif(relpages, 0)) || '% all-visible', ', ' ORDER BY relname)
			FROM pg_class WHERE relname IN ('rfv_num', 'rfv_text', 'rfv_bool')`))

	if serial != nil {
		c := serial.cell(t, "a50, pk = 3", listReqs([]string{"a50"}, "filter%5Bpk%5D=3"), false)
		serial.planCheck(t, c, "a50", "filter%5Bpk%5D=3")
		_ = os.WriteFile(filepath.Join(dir, "serial-samples.csv"), []byte("cell,block,try,arm,pooled,status,total_state,count_phase_ms,count_stmt_ms,latency_ms\n"+serial.csv.String()), 0o600)
		overall := serial.report()
		ta := gateTemp(t, e.super)
		w("")
		w("(3a) leak check after (3e) (ADR-006-E2 E2-3: the verdict reads pg_ls_tmpdir() only):")
		w("")
		time.Sleep(60 * time.Second)
		t60 := gateTemp(t, e.super)
		leakOK := gateLeak(w, "(3e)", ta, t60)
		if files, bytes := serial.tempRise(); files > 0 {
			w("- temp_files rose by %d (%d bytes) over the (3e) blocks alone (gates and fixture left out): above 0, so the leak reading covered blocks that spilled.", files, bytes)
		} else {
			w("- temp_files was 0 over the (3e) blocks: the (3a) leak reading did not see a spill; D-7E2-7 relies on runs 5 and 6 for the spill reading.")
		}
		lv := "FAIL"
		if leakOK {
			lv = "PASS (no pgsql_tmp file after the run or 60 s later)"
		}
		w("")
		w("**(3a) leak verdict after (3e): %s**", lv)
		w("")
		w("(3e) %s", overall)
	}

	// TICK-1083 (T-A2-4b): the spike runs on up to three cells, each picked
	// from its own candidate list. "base" is TICK-1048's: the slowest plain
	// count under the server's settings. "spill" is a hash-join count with
	// work_mem 64kB that writes temp files before the budget cancels it.
	// "parallel" is a count whose plan has a Gather that launches workers. A
	// cell's settings are a SET LOCAL inside the handler's SAVEPOINT, just
	// before the count (gateCountTracer), so the page statement runs on the
	// server's settings. That SET LOCAL is one more round trip inside the
	// count phase; the phase is reported without it, and the trip apart.
	// NAVA_PERF_SPIKE_CELLS (base,spill,parallel) picks the cells.
	type cand struct {
		label, token, q string
		params          map[string]string
	}
	type cell struct {
		name, why string
		cands     []cand
		// qualifies says whether a candidate's EXPLAIN plan has the shape the
		// cell needs (a spill, a launched Gather); "" means it does.
		qualifies func(plan string) string
	}
	spillParams := func(extra map[string]string) map[string]string {
		p := map[string]string{"work_mem": "64kB", "max_parallel_workers_per_gather": "0"}
		maps.Copy(p, extra)
		return p
	}
	cells := []cell{
		{name: "base", why: "TICK-1048's cell: the slowest plain count under the server's settings", cands: []cand{
			{"a6, pk = 3 (broad 50% value, 6% ACL)", "a6", "filter%5Bpk%5D=3", nil},
			{"a6, three broad fields", "a6", "filter%5Bpk%5D=3&filter%5Bpt%5D=x&filter%5Bpb%5D=true", nil},
			{"a6, pk = 20 (partly correlated, 500 visible)", "a6", "filter%5Bpk%5D=20", nil},
			{"a50, pk = 3 (broad 50% value, 50% ACL)", "a50", "filter%5Bpk%5D=3", nil},
			{"a6, pk = 10 (anti-correlated, 0 visible)", "a6", "filter%5Bpk%5D=10", nil},
			{"a6, pk < 5 (every 3 and 4: 78% of the side rows)", "a6", "filter%5Bpk%5D%5Blt%5D=5", nil},
			{"a50, pw = 1 (704k invisible side rows, 100 visible)", "a50", "filter%5Bpw%5D=1", nil},
			{"a6, pw = 1 (704k invisible side rows, 100 visible)", "a6", "filter%5Bpw%5D=1", nil},
		}},
		{name: "spill", why: "a hash-join count with work_mem 64kB that spills to temp files before the budget cancels it", cands: []cand{
			{"a50, pk = 3", "a50", "filter%5Bpk%5D=3", spillParams(nil)},
			{"a50, pw = 1", "a50", "filter%5Bpw%5D=1", spillParams(nil)},
			{"a50, pk = 3", "a50", "filter%5Bpk%5D=3", spillParams(map[string]string{"enable_nestloop": "off"})},
			{"a50, pw = 1", "a50", "filter%5Bpw%5D=1", spillParams(map[string]string{"enable_nestloop": "off"})},
			{"a50, pw = 1", "a50", "filter%5Bpw%5D=1", spillParams(map[string]string{"enable_nestloop": "off", "enable_mergejoin": "off"})},
			{"a50, pk < 5", "a50", "filter%5Bpk%5D%5Blt%5D=5", spillParams(map[string]string{"enable_nestloop": "off", "enable_mergejoin": "off"})},
			{"a50, pw = 1 and pk < 5", "a50", "filter%5Bpw%5D=1&filter%5Bpk%5D%5Blt%5D=5", spillParams(map[string]string{"enable_nestloop": "off", "enable_mergejoin": "off"})},
			{"a50, three broad fields", "a50", "filter%5Bpk%5D=3&filter%5Bpt%5D=x&filter%5Bpb%5D=true", spillParams(map[string]string{"enable_nestloop": "off", "enable_mergejoin": "off"})},
		}, qualifies: func(plan string) string {
			// The first UNION ALL branch runs when the budget expires.
			b := gateBranch1(plan)
			if !strings.Contains(b, "Hash Semi Join") && !strings.Contains(b, "Hash Join") {
				return "no hash join in branch 1"
			}
			if !strings.Contains(b, "temp written=") {
				return "no temp writes in branch 1"
			}
			return ""
		}},
		{name: "parallel", why: "a count whose plan has a Gather that launches workers", cands: []cand{
			{"a50, pk = 3", "a50", "filter%5Bpk%5D=3", nil},
			{"a6, three broad fields", "a6", "filter%5Bpk%5D=3&filter%5Bpt%5D=x&filter%5Bpb%5D=true", nil},
			{"a50, pw = 1", "a50", "filter%5Bpw%5D=1", map[string]string{"parallel_setup_cost": "0", "parallel_tuple_cost": "0"}},
			{"a50, pw = 1", "a50", "filter%5Bpw%5D=1", map[string]string{"parallel_setup_cost": "0", "parallel_tuple_cost": "0", "max_parallel_workers_per_gather": "1"}},
			{"a50, pw = 1", "a50", "filter%5Bpw%5D=1", map[string]string{"enable_nestloop": "off"}},
			{"a50, pk < 5", "a50", "filter%5Bpk%5D%5Blt%5D=5", map[string]string{"enable_nestloop": "off"}},
			{"a50, pw = 1 and pk < 5", "a50", "filter%5Bpw%5D=1&filter%5Bpk%5D%5Blt%5D=5", map[string]string{"enable_nestloop": "off"}},
		}, qualifies: func(plan string) string {
			// A launched Gather in the first UNION ALL branch that is still
			// running at the budget (EXPLAIN total >= 1.25 x budget): the
			// cancel lands while its workers run.
			b := gateBranch1(plan)
			m := gateGather.FindStringSubmatch(b)
			if m == nil {
				return "no Gather in branch 1"
			}
			if !gateWorkersLaunched.MatchString(b) {
				return "no worker launched in branch 1"
			}
			var total float64
			_, _ = fmt.Sscanf(m[1], "%g", &total)
			if total < 1.25*float64(store.CountBudget.Milliseconds()) {
				return fmt.Sprintf("branch-1 Gather ends at %.0f ms (EXPLAIN)", total)
			}
			return ""
		}},
	}
	// setOf is the SET LOCAL statements a cell's settings take.
	setOf := func(params map[string]string) string {
		var ss []string
		for _, k := range slices.Sorted(maps.Keys(params)) {
			ss = append(ss, "SET LOCAL "+k+" = '"+params[k]+"'")
		}
		return strings.Join(ss, "; ")
	}
	paramsOf := func(params map[string]string) string {
		if len(params) == 0 {
			return "server settings"
		}
		var kv []string
		for _, k := range slices.Sorted(maps.Keys(params)) {
			kv = append(kv, k+" = "+params[k])
		}
		return strings.Join(kv, ", ")
	}
	var setErrors atomic.Int64
	pools := map[string]*pgxpool.Pool{}
	handlers := map[string]http.Handler{}
	// A cell's pool traces like e.pool and, through gateCountTracer, sends
	// the cell's SET LOCAL right after the handler's SAVEPOINT list_count:
	// the settings hold for the count statement alone and end at ROLLBACK TO.
	poolFor := func(params map[string]string) (*pgxpool.Pool, http.Handler) {
		if len(params) == 0 {
			return e.pool, normal
		}
		key := paramsOf(params)
		if p, ok := pools[key]; ok {
			return p, handlers[key]
		}
		pc := e.pool.Config().Copy()
		pc.ConnConfig.Tracer = gateCountTracer{set: setOf(params), errs: &setErrors}
		p, err := pgxpool.NewWithConfig(t.Context(), pc)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(p.Close)
		pools[key], handlers[key] = p, e.handlerWith(t, server.WithPool(p))
		return p, handlers[key]
	}

	// The count statement of a request, as the handler sent it.
	countOf := func(h http.Handler, token, q, set string) (perfStmt, bool) {
		tr := newPerfTrace()
		rec := serveOn(context.WithValue(t.Context(), perfTraceKey{}, tr), h, http.MethodGet, "/api/v1/records/ticket?"+q, token, "", "")
		if rec.Code != http.StatusOK {
			t.Fatalf("candidate %s as %s = %d: %s", q, token, rec.Code, rec.Body)
		}
		for i, s := range tr.stmts {
			if strings.HasPrefix(strings.TrimSpace(s.SQL), "SAVEPOINT list_count") {
				for _, x := range tr.stmts[i+1:] {
					if set == "" || x.SQL != set {
						return x, true
					}
				}
			}
		}
		return perfStmt{}, false
	}
	explain := func(set string, s perfStmt) (string, float64) {
		var lines []string
		if err := tenancy.InTenantTx(t.Context(), e.pool, large, func(tx pgx.Tx) error {
			if set != "" {
				if _, err := tx.Exec(t.Context(), set); err != nil {
					return err
				}
			}
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
		}); err != nil {
			t.Fatal(err)
		}
		p := strings.Join(lines, "\n")
		return p, a2gms(p)
	}
	// plain is the count's own time without EXPLAIN's per-row timing, as the
	// handler pays it: EXPLAIN (ANALYZE) inflates a hash build severalfold.
	plain := func(set string, s perfStmt) float64 {
		var ms float64
		if err := tenancy.InTenantTx(t.Context(), e.pool, large, func(tx pgx.Tx) error {
			if set != "" {
				if _, err := tx.Exec(t.Context(), set); err != nil {
					return err
				}
			}
			var n int64
			t0 := time.Now()
			err := tx.QueryRow(t.Context(), s.SQL, s.Args...).Scan(&n)
			ms = fiMS(time.Since(t0))
			return err
		}); err != nil {
			t.Fatal(err)
		}
		return ms
	}
	// temp is gateTemp: pg_ls_tmpdir() is the leak reading (E2-3);
	// temp_files / temp_bytes only show that a spill happened.
	temp := func() gateTempReading { return gateTemp(t, e.super) }

	// P3-36's host gates around each cell (TICK-1083): a cell scored beside
	// a busy host is not evidence.
	allPass := true
	var verdicts []string
	for _, c := range cells {
		if !want[c.name] {
			continue
		}
		w("")
		w("### Cell %s: %s", c.name, c.why)
		w("")
		cellBefore := host("cell " + c.name + " host gate, before")
		w("")
		w("Unbudgeted count time of the handler's count statement, as nava_app: 3 plain runs (the cell picks on these) and 3 EXPLAIN (ANALYZE, BUFFERS) runs:")
		w("")
		w("| Candidate | settings | count statement? | plan shape | plain runs, ms | plain median ms | EXPLAIN runs, ms | EXPLAIN median ms |")
		w("| --- | --- | --- | --- | --- | --- | --- | --- |")
		var pick *cand
		var pickMS float64
		for i, cd := range c.cands {
			_, h := poolFor(cd.params)
			set := setOf(cd.params)
			s, ok := countOf(h, cd.token, cd.q, set)
			if !ok {
				w("| %s | %s | no: the first page holds every match (A2-4 shortcut) | | | | | |", cd.label, paramsOf(cd.params))
				continue
			}
			var ms []float64
			var plan string
			for range 3 {
				p, x := explain(set, s)
				ms, plan = append(ms, x), p
			}
			slices.Sort(ms)
			pl := []float64{plain(set, s), plain(set, s), plain(set, s)}
			slices.Sort(pl)
			shape, bad := "any", ""
			if c.qualifies != nil {
				if bad = c.qualifies(plan); bad == "" {
					shape = "qualifies"
				} else {
					shape = bad
				}
			}
			w("| %s | %s | yes | %s | %.1f, %.1f, %.1f | %.1f | %.1f, %.1f, %.1f | %.1f |", cd.label, paramsOf(cd.params), shape, pl[0], pl[1], pl[2], pl[1], ms[0], ms[1], ms[2], ms[1])
			plans = append(plans, "### Cell "+c.name+", unbudgeted count: "+cd.label+" ("+paramsOf(cd.params)+")\n\n```sql\n"+strings.TrimSpace(s.SQL)+"\n```\n\n```\n"+plan+"\n```")
			writePlans()
			if bad == "" && pl[1] > pickMS {
				pick, pickMS = &c.cands[i], pl[1]
			}
		}
		if pick == nil || pickMS <= float64(store.CountBudget.Milliseconds()) {
			w("")
			w("**Cell %s: no qualifying candidate count exceeds CountBudget unbudgeted: not scored.**", c.name)
			verdicts = append(verdicts, c.name+": NOT SCORED")
			allPass = false
			continue
		}
		w("")
		w("Spike cell %s: **%s** (%s), unbudgeted plain median %.1f ms against a %s budget.", c.name, pick.label, paramsOf(pick.params), pickMS, store.CountBudget)

		_, h := poolFor(pick.params)
		set := setOf(pick.params)
		path := "/api/v1/records/ticket?" + pick.q
		one := func(run string) gateSpike {
			tr := newPerfTrace()
			t0 := time.Now()
			rec := serveOn(context.WithValue(t.Context(), perfTraceKey{}, tr), h, http.MethodGet, path, pick.token, "", "")
			g := gateSpike{run: run, code: rec.Code, latency: fiMS(time.Since(t0)), phase: -1, countStmt: -1, set: -1}
			var raw map[string]json.RawMessage
			var p listPage
			if json.Unmarshal(rec.Body.Bytes(), &raw) == nil && json.Unmarshal(rec.Body.Bytes(), &p) == nil {
				g.state, g.items, g.totalNull = p.State, len(p.Items), string(raw["total"]) == "null"
			}
			sp := -1
			for i, s := range tr.stmts {
				sql := strings.TrimSpace(s.SQL)
				switch {
				case strings.HasPrefix(sql, "SAVEPOINT list_count"):
					sp = i
				case strings.HasPrefix(sql, "ROLLBACK TO SAVEPOINT list_count") && sp >= 0:
					g.phase = fiMS(s.Start + s.Dur - tr.stmts[sp].Start)
					for _, x := range tr.stmts[sp+1 : i] {
						if set != "" && x.SQL == set {
							g.set = fiMS(x.Dur)
							g.phase -= g.set
						} else {
							g.countStmt = fiMS(x.Dur)
						}
					}
					for _, a := range tr.stmts[i+1:] {
						f := firstLine(normSQL(a.SQL))
						g.after = append(g.after, f[:min(40, len(f))])
					}
				}
			}
			return g
		}
		tb := temp()
		var spikes []gateSpike
		for i := range 5 {
			_ = one(fmt.Sprintf("warm-up %d", i))
		}
		floorSpike := e.transportFloor(t, clients)
		for range 100 {
			spikes = append(spikes, one("1-client"))
		}
		per := perfEnvInt("NAVA_PERF_SPIKE_PER_CLIENT", 25)
		var mu sync.Mutex
		var wg sync.WaitGroup
		for range clients {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for range per {
					g := one(fmt.Sprintf("%d-client", clients))
					mu.Lock()
					spikes = append(spikes, g)
					mu.Unlock()
				}
			}()
		}
		wg.Wait()
		ta := temp()
		time.Sleep(60 * time.Second)
		t60 := temp()
		w("- transport floor before the spike: p50 %.2f ms; other containers: %v", pct(floorSpike, .5), foreignContainers(t.Context()))
		w("- before: %s", tb)
		w("- temp files written during the spike (spill context only: proves a spill happened, never a leak): %d files, %d bytes", ta.files-tb.files, ta.bytes-tb.bytes)
		leakOK := gateLeak(w, "cell "+c.name, ta, t60)

		name := "spike-samples.csv"
		if c.name != "base" {
			name = "spike-samples-" + c.name + ".csv"
		}
		csv := &strings.Builder{}
		csv.WriteString("run,status,total_state,total_null,items,count_phase_ms,count_stmt_ms,harness_set_local_ms,latency_ms,statements_after_rollback_to\n")
		for _, s := range spikes {
			fmt.Fprintf(csv, "%s,%d,%s,%v,%d,%.3f,%.3f,%.3f,%.3f,%s\n", s.run, s.code, s.state, s.totalNull, s.items, s.phase, s.countStmt, s.set, s.latency, strings.Join(s.after, " | "))
		}
		_ = os.WriteFile(filepath.Join(dir, name), []byte(csv.String()), 0o600)
		w("")
		w("| Run | n | status 200 + unknown + total null + full page | further statement after ROLLBACK TO | count phase p50 / p95 / p99 / max ms | p99 overshoot over %s | client latency p50 / p95 ms |", store.CountBudget)
		w("| --- | --- | --- | --- | --- | --- | --- |")
		pass := true
		for _, run := range []string{"1-client", fmt.Sprintf("%d-client", clients)} {
			var ph, lat []time.Duration
			ok, further, n := 0, 0, 0
			for _, s := range spikes {
				if s.run != run {
					continue
				}
				n++
				if s.code == http.StatusOK && s.state == "unknown" && s.totalNull && s.items == 50 {
					ok++
				}
				if len(s.after) > 0 {
					further++
				}
				if s.phase >= 0 {
					ph = append(ph, time.Duration(s.phase*float64(time.Millisecond)))
				}
				lat = append(lat, time.Duration(s.latency*float64(time.Millisecond)))
			}
			p99 := pct(ph, .99)
			over := p99 - float64(store.CountBudget.Milliseconds())
			pass = pass && ok == n && further == n && len(ph) == n && over <= gateOvershootMS
			w("| %s | %d | %d | %d | %.1f / %.1f / %.1f / %.1f | **%+.1f ms** (limit +%.0f) | %.1f / %.1f |", run, n, ok, further,
				pct(ph, .5), pct(ph, .95), p99, pct(ph, 1), over, gateOvershootMS, pct(lat, .5), pct(lat, .95))
		}
		w("")
		w("Count statement alone (QueryRow to its error, client side) on the unknown answers, and the two round trips around it (SAVEPOINT + SET LOCAL Exec, ROLLBACK TO Exec):")
		w("")
		w("| Run | unknown n | count statement p50 / p99 / max ms | statement overshoot p99 | two round trips p50 / p99 ms |")
		w("| --- | --- | --- | --- | --- |")
		for _, run := range []string{"1-client", fmt.Sprintf("%d-client", clients)} {
			var cs, rt []time.Duration
			for _, s := range spikes {
				if s.run == run && s.state == "unknown" && s.countStmt >= 0 {
					cs = append(cs, time.Duration(s.countStmt*float64(time.Millisecond)))
					rt = append(rt, time.Duration((s.phase-s.countStmt)*float64(time.Millisecond)))
				}
			}
			w("| %s | %d | %.1f / %.1f / %.1f | %+.1f ms | %.1f / %.1f |", run, len(cs), pct(cs, .5), pct(cs, .99), pct(cs, 1),
				pct(cs, .99)-float64(store.CountBudget.Milliseconds()), pct(rt, .5), pct(rt, .99))
		}
		if set != "" {
			var sd []time.Duration
			for _, s := range spikes {
				if s.set >= 0 {
					sd = append(sd, time.Duration(s.set*float64(time.Millisecond)))
				}
			}
			w("")
			w("The harness's SET LOCAL (`%s`), sent after SAVEPOINT and left out of the count phase above: %d of %d requests, p50 %.2f / p99 %.2f ms; failed %d times.",
				set, len(sd), len(spikes), pct(sd, .5), pct(sd, .99), setErrors.Load())
		}
		if len(spikes) > 0 {
			w("")
			w("Statements after ROLLBACK TO in one request: %s", strings.Join(spikes[0].after, " | "))
		}
		var notes []string
		w("")
		if cellAfter := host("cell " + c.name + " host gate, after"); !cellBefore.ok() || !cellAfter.ok() {
			pass = false
			notes = append(notes, "INCONCLUSIVE: a host gate failed")
		}
		if setErrors.Load() > 0 {
			pass = false
			notes = append(notes, "the harness's SET LOCAL failed")
		}
		// ADR-006-E2 E2-3: the leak verdict is pg_ls_tmpdir() alone.
		// temp_files, the pg_ls_dir listing and the worker count are
		// reported above and gate nothing.
		if !leakOK {
			pass = false
			notes = append(notes, "pg_ls_tmpdir() lists pgsql_tmp files after the run")
		}
		if c.name == "spill" && ta.files <= tb.files {
			notes = append(notes, "reported: temp_files did not rise, so the cell may not have spilled")
		}
		v := "PASS"
		if !pass {
			v = "FAIL"
		}
		if len(notes) > 0 {
			v += " (" + strings.Join(notes, "; ") + ")"
		}
		w("")
		w("**(3a) cell %s verdict: %s**", c.name, v)
		verdicts = append(verdicts, c.name+": "+v)
		allPass = allPass && pass
	}
	w("")
	switch {
	case len(verdicts) == 0:
		w("(3a) S-A2-cancel cells: none selected (NAVA_PERF_SPIKE_CELLS)")
	case allPass:
		w("**(3a) verdict: PASS** (%s)", strings.Join(verdicts, "; "))
	default:
		w("**(3a) verdict: FAIL or not scored** (%s)", strings.Join(verdicts, "; "))
	}
	writePlans()
}

// gateCountTracer is perfTracer plus set, sent on the same connection right
// after the handler's SAVEPOINT list_count (TICK-1083). Inside the savepoint
// it holds for the count statement alone, and ROLLBACK TO ends it. The
// nested Exec is traced like any statement, so the harness sees its time.
type gateCountTracer struct {
	perfTracer
	set string
	// errs counts failed injected SET LOCALs, shared by the pool's conns.
	errs *atomic.Int64
}

type gateSQLKey struct{}

func (g gateCountTracer) TraceQueryStart(ctx context.Context, c *pgx.Conn, d pgx.TraceQueryStartData) context.Context {
	return context.WithValue(g.perfTracer.TraceQueryStart(ctx, c, d), gateSQLKey{}, d.SQL)
}

func (g gateCountTracer) TraceQueryEnd(ctx context.Context, c *pgx.Conn, d pgx.TraceQueryEndData) {
	g.perfTracer.TraceQueryEnd(ctx, c, d)
	sql, _ := ctx.Value(gateSQLKey{}).(string)
	if g.set == "" || d.Err != nil || !strings.HasPrefix(strings.TrimSpace(sql), "SAVEPOINT list_count") {
		return
	}
	if _, err := c.Exec(ctx, g.set); err != nil {
		g.errs.Add(1)
	}
}

// gateWorkersLaunched matches a Gather that launched at least one worker;
// gateGather captures a Gather's actual total time.
var (
	gateWorkersLaunched = regexp.MustCompile(`Workers Launched: [1-9]`)
	gateGather          = regexp.MustCompile(`Gather +\(cost=[^)]*\) \(actual time=[0-9.]+\.\.([0-9.]+) `)
)

// gateBranch1 is the part of a count plan under its first UNION ALL branch,
// the one the executor runs first (the whole plan if there is no Append).
func gateBranch1(plan string) string {
	i := strings.Index(plan, `Subquery Scan on "*SELECT* 1"`)
	if i < 0 {
		return plan
	}
	b := plan[i:]
	if j := strings.Index(b, `Subquery Scan on "*SELECT* 2"`); j >= 0 {
		b = b[:j]
	}
	return b
}

// a2gms is planning plus execution time of an EXPLAIN (ANALYZE) output, in ms.
func a2gms(plan string) float64 {
	var p, x float64
	if m := fiPlanning.FindStringSubmatch(plan); m != nil {
		_, _ = fmt.Sscanf(m[1], "%g", &p)
	}
	if m := fiExecution.FindStringSubmatch(plan); m != nil {
		_, _ = fmt.Sscanf(m[1], "%g", &x)
	}
	return p + x
}

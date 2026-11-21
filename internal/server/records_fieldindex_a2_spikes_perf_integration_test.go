//go:build integration && perf

// ADR-006-A2-11 (TICK-1022): spikes S-A2-anti, S-A2-skew and S-A2-multi, with
// the pass and fail rules A2-11 fixed before they ran. P7-07's 800k seed
// (400k records of one type in the large tenant), plus:
//
//   - two agents whose visible sets are exactly their group's records: a6
//     sees 6% of the type (24k), a50 sees 50% (200k);
//   - num field pk, one side row per record:
//     10 "anti": 26k records, none visible to a6 or a50;
//     20 "part": 500 of a6's records plus 21k invisible ones;
//     30 / 31: exactly 19,999 / 20,000 invisible records (the R13 step);
//     77: 50 records here, and 200k side rows in tenant B (S-A2-skew);
//     3 / 4: every other record, half each;
//   - text field pt ('x' / 'y') and bool field pb, half each (S-A2-multi).
//
// Each cell runs in two arms: "before", today's SQL (nothing is broad), and
// "A2", the production caps. Visibility-map states "none" (ANALYZE, no
// VACUUM: 0% all-visible) and "vacuum". Autovacuum is off on the side
// tables, so each state stays as set up.
//
//	go test -tags 'integration perf' -count=1 -timeout 180m -v \
//	    -run '^TestA2Spikes$' ./internal/server
//
// with NAVA_TEST_PG_URL as scratch/field-index-measured/run.sh sets it
// (scratch/tick-1022-a2-spikes/run.sh). NAVA_PERF_LIST_SECONDS (8) per
// cell; NAVA_PERF_OUT the report.

package server_test

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

const a2SchemaExtra = `"pk": {"type": "number"}, "pt": {"type": "string", "maxLength": 10}, "pb": {"type": "boolean"},
    `

// a2Caps switches the handler between the arms.
type a2Caps struct {
	mu             sync.Mutex
	broad, visible int
}

func (c *a2Caps) get() (int, int) { c.mu.Lock(); defer c.mu.Unlock(); return c.broad, c.visible }
func (c *a2Caps) set(b, v int)    { c.mu.Lock(); c.broad, c.visible = b, v; c.mu.Unlock() }

// a2Times is one request's server time per statement kind (planning +
// execution of its EXPLAIN (ANALYZE)), in ms, and its count's join.
type a2Times struct {
	pre, guard, count, page float64
	pres                    int
	countPlan               string
}

var (
	a2Planning  = regexp.MustCompile(`Planning Time: ([0-9.]+) ms`)
	a2Execution = regexp.MustCompile(`Execution Time: ([0-9.]+) ms`)
)

func a2ms(plan string) float64 {
	p, _ := strconv.ParseFloat(a2Planning.FindStringSubmatch(plan)[1], 64)
	x, _ := strconv.ParseFloat(a2Execution.FindStringSubmatch(plan)[1], 64)
	return p + x
}

func a2Classify(r *perfResult) a2Times {
	var out a2Times
	for i, s := range r.stmtSQL {
		if i >= len(r.explains) || r.explains[i] == "" {
			continue
		}
		plan, sql := r.explains[i], s.SQL
		switch {
		case strings.Contains(sql, "rfv_") && strings.Contains(sql, "ORDER BY v."):
			out.pre += a2ms(plan)
			out.pres++
		case strings.Contains(sql, "LIMIT 30001") && !strings.Contains(sql, "rfv_"):
			out.guard += a2ms(plan)
		case strings.Contains(sql, "rfv_") && strings.Contains(sql, "count(*)"):
			out.count += a2ms(plan)
			switch {
			case strings.Contains(plan, "Hash Semi Join") || strings.Contains(plan, "Hash Join"):
				out.countPlan = "hash"
			case strings.Contains(plan, "SubPlan"):
				out.countPlan = "probe"
			default:
				out.countPlan = "join"
			}
			if strings.Contains(plan, "temp written") {
				out.countPlan += "+spill"
			}
		case strings.Contains(sql, "rfv_"):
			out.page += a2ms(plan)
		}
	}
	return out
}

func TestA2Spikes(t *testing.T) {
	clients := perfEnvInt("NAVA_PERF_CLIENTS", 8)
	out := os.Getenv("NAVA_PERF_OUT")
	if out == "" {
		out = filepath.Join("..", "..", "bin", "tick-1022-a2-spikes.md")
	}
	var buf strings.Builder
	w := func(format string, args ...any) {
		line := fmt.Sprintf(format, args...)
		buf.WriteString(line + "\n")
		t.Log(line)
		_ = os.MkdirAll(filepath.Dir(out), 0o750)
		_ = os.WriteFile(out, []byte(buf.String()), 0o600)
	}
	caps := &a2Caps{}
	caps.set(server.DefaultCountProbeCaps())
	pe := newPerfEnvWith(t, clients, server.WithCountProbe(caps.get, nil, nil))
	pe.schemaExtra = a2SchemaExtra
	e := &fiEnv{perfEnv: pe, w: w, errSample: map[string]string{}}
	w("# TICK-1022 A2-11 spikes, raw (%s)", time.Now().UTC().Format(time.RFC3339))
	w("")
	t0 := time.Now()
	e.seed(t)
	w("- seed: 800k records in %s", time.Since(t0).Round(time.Second))
	w("- %s", e.scalar(t, "SELECT version()"))
	w("- settings: %s", e.scalar(t, `SELECT string_agg(name || '=' || setting || coalesce(unit, ''), ', ' ORDER BY name) FROM pg_settings
		WHERE name IN ('shared_buffers','effective_cache_size','work_mem','random_page_cost','max_parallel_workers_per_gather','jit','default_statistics_target')`))
	w("- nava_app: %s", e.scalar(t, `SELECT coalesce((SELECT array_to_string(setconfig, ',') FROM pg_db_role_setting s JOIN pg_roles r ON r.oid = s.setrole WHERE r.rolname = 'nava_app'), 'default')`))
	w("- other containers at start: %v", foreignContainers(t.Context()))
	if err := e.super.QueryRow(t.Context(), `SELECT d.id FROM entity_definitions d JOIN perf_tenant p ON p.tenant_id = d.tenant_id
		WHERE p.tn = 0 AND d.key = 'ticket'`).Scan(&e.largeDef); err != nil {
		t.Fatal(err)
	}
	var agentsGroup, bTenant, bDef uuid.UUID
	if err := e.super.QueryRow(t.Context(), `SELECT agents FROM perf_tenant WHERE tn = 0`).Scan(&agentsGroup); err != nil {
		t.Fatal(err)
	}
	if err := e.super.QueryRow(t.Context(), `SELECT p.tenant_id, d.id FROM perf_tenant p JOIN entity_definitions d ON d.tenant_id = p.tenant_id AND d.key = 'ticket' WHERE p.tn = 1`).Scan(&bTenant, &bDef); err != nil {
		t.Fatal(err)
	}
	large := e.large.id
	g6, g50, gRest := uuid.New(), uuid.New(), uuid.New()
	for _, g := range []struct {
		id   uuid.UUID
		name string
	}{{g6, "a2-g6"}, {g50, "a2-g50"}, {gRest, "a2-rest"}} {
		e.exec(t, "INSERT INTO groups (tenant_id, id, keycloak_group_id, path) VALUES ($1, $2, $3, $4)", large, g.id, g.name, "/t/"+large.String()+"/"+g.name)
	}
	e.tokens["a6"] = identity.Principal{TenantID: large, PrincipalID: uuid.New(), Groups: []uuid.UUID{agentsGroup, g6}}
	e.tokens["a50"] = identity.Principal{TenantID: large, PrincipalID: uuid.New(), Groups: []uuid.UUID{agentsGroup, g50}}

	t1 := time.Now()
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
	// S-A2-skew: tenant B holds 200k side rows of 77 on its own pk field.
	bField := uuid.New()
	e.exec(t, `INSERT INTO record_index_fields (tenant_id, id, entity_definition_id, field_key, value_type, state, progress, requested_version)
		VALUES ($1, $2, $3, 'pk', 'num', 'ready', 100, 1)`, bTenant, bField, bDef)
	e.exec(t, `INSERT INTO rfv_num (tenant_id, field_id, record_id, ord, value) SELECT $1, $2, gen_random_uuid(), 0, 77 FROM generate_series(1, 200000)`, bTenant, bField)
	e.exec(t, "SET session_replication_role = origin")
	for _, tb := range []string{"rfv_num", "rfv_text", "rfv_bool", "rfv_ts"} {
		e.exec(t, "ALTER TABLE "+tb+" SET (autovacuum_enabled = false)")
	}
	e.exec(t, "VACUUM (ANALYZE) records, groups")
	w("- fixtures in %s: pk values %s; visible a6 %s, a50 %s", time.Since(t1).Round(time.Second),
		e.scalar(t, `SELECT string_agg(v || ':' || n, ' ' ORDER BY v) FROM (SELECT v, count(*) n FROM a2_pk GROUP BY v) x`),
		e.scalar(t, `SELECT count(*) FROM records WHERE tenant_id = $1 AND assignment_group_id = $2`, large, g6),
		e.scalar(t, `SELECT count(*) FROM records WHERE tenant_id = $1 AND assignment_group_id = $2`, large, g50))
	w("- pk=20 visible to a6: %s", e.scalar(t, `SELECT count(*) FROM a2_pk p JOIN a2_rec a USING (id) WHERE p.v = 20 AND a.b < 60`))

	d := time.Duration(perfEnvInt("NAVA_PERF_LIST_SECONDS", 8)) * time.Second
	type cell struct{ spike, label, token, q string }
	cells := []cell{
		{"anti", "6%, value on 26k invisible records", "a6", "filter%5Bpk%5D=10"},
		{"anti", "50%, value on 26k invisible records", "a50", "filter%5Bpk%5D=10"},
		{"anti", "6%, partly correlated (500 visible)", "a6", "filter%5Bpk%5D=20"},
		{"step", "6%, 19,999 invisible side rows", "a6", "filter%5Bpk%5D=30"},
		{"step", "6%, 20,000 invisible side rows", "a6", "filter%5Bpk%5D=31"},
		{"ref", "6%, broad 50% value", "a6", "filter%5Bpk%5D=3"},
		{"ref", "50%, broad 50% value", "a50", "filter%5Bpk%5D=3"},
		{"multi", "6%, three broad fields", "a6", "filter%5Bpk%5D=3&filter%5Bpt%5D=x&filter%5Bpb%5D=true"},
	}
	var explains []string
	for _, st := range []string{"none", "vacuum"} {
		if st == "none" {
			e.exec(t, "ANALYZE rfv_num")
			e.exec(t, "ANALYZE rfv_text")
			e.exec(t, "ANALYZE rfv_bool")
		} else {
			e.exec(t, "VACUUM (ANALYZE) rfv_num, rfv_text, rfv_bool")
		}
		e.exec(t, "CHECKPOINT")
		e.exec(t, `SELECT pg_prewarm(c.oid) FROM pg_class c WHERE c.relname LIKE 'rfv\_%' AND c.relkind IN ('r', 'i')`)
		w("")
		w("## State %s: %s", st, e.scalar(t, `SELECT string_agg(relname || ' ' || round(100.0 * relallvisible / nullif(relpages, 0)) || '% all-visible', ', ' ORDER BY relname)
			FROM pg_class WHERE relname IN ('rfv_num', 'rfv_text', 'rfv_bool')`))
		w("")
		w("| Spike | Cell | before: count ms (plan) | A2: pre-counts + guard + count ms (plan) | A2 - before | A2 page ms | 1-client p50 before / A2 | 8-client p95 A2 |")
		w("| --- | --- | --- | --- | --- | --- | --- | --- |")
		for _, c := range cells {
			req := []perfReq{{c.token, "/api/v1/records/ticket?" + c.q, ""}}
			caps.set(1<<30, 0)
			b := e.run(t, perfScenario{name: st + " before " + c.label, kind: "data", tenant: "large", reqs: req}, 1, d)
			bt := a2Classify(b)
			caps.set(server.DefaultCountProbeCaps())
			a := e.run(t, perfScenario{name: st + " A2 " + c.label, kind: "data", tenant: "large", reqs: req}, 1, d)
			at := a2Classify(a)
			m := e.run(t, perfScenario{name: st + " A2 8c " + c.label, kind: "data", tenant: "large", reqs: req}, clients, d)
			total := at.pre + at.guard + at.count
			w("| %s | %s | %.1f (%s) | %.1f + %.1f + %.1f = %.1f (%s, %d pre) | %+.1f | %.1f | %.1f / %.1f | %.1f |", c.spike, c.label,
				bt.count, bt.countPlan, at.pre, at.guard, at.count, total, at.countPlan, at.pres, total-bt.count, at.page,
				pct(b.lat, .5), pct(a.lat, .5), pct(m.lat, .95))
			for _, r := range []*perfResult{b, a} {
				for i, s := range r.stmtSQL {
					if strings.Contains(s.SQL, "rfv_") || strings.Contains(s.SQL, "LIMIT 30001") {
						if i < len(r.explains) && r.explains[i] != "" {
							explains = append(explains, "### "+r.sc.name+"\n\n```sql\n"+strings.TrimSpace(s.SQL)+"\n```\n\n```\n"+r.explains[i]+"\n```")
						}
					}
				}
			}
		}
		// S-A2-skew: the pre-count of 77 as tenant A, which holds 50 side rows
		// of it while tenant B holds 200k.
		sql, args := server.PreCountForTest(fieldindex.TypeNum, large, fields["pk"], "eq", 77.0)
		var lines []string
		if err := tenancy.InTenantTx(t.Context(), e.pool, large, func(tx pgx.Tx) error {
			rows, err := tx.Query(t.Context(), "EXPLAIN (ANALYZE, BUFFERS, SETTINGS) "+sql, args...)
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
		plan := strings.Join(lines, "\n")
		scan := "other"
		switch {
		case strings.Contains(plan, "Index Only Scan using rfv_num_value"):
			scan = "Index Only Scan on rfv_num_value"
		case strings.Contains(plan, "Index Scan using rfv_num_value"):
			scan = "Index Scan on rfv_num_value"
		case strings.Contains(plan, "Seq Scan"):
			scan = "Seq Scan"
		case strings.Contains(plan, "Bitmap"):
			scan = "Bitmap scan"
		}
		w("")
		w("S-A2-skew (%s): pre-count of 77 as tenant A (50 side rows; tenant B 200k): %s, %.2f ms", st, scan, a2ms(plan))
		explains = append(explains, "### "+st+" S-A2-skew pre-count\n\n```sql\n"+sql+"\n```\n\n```\n"+plan+"\n```")
	}
	p := strings.TrimSuffix(out, ".md") + "-explain.md"
	if err := os.WriteFile(p, []byte("# TICK-1022 A2-11 spike plans (as nava_app)\n\n"+strings.Join(explains, "\n\n")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Logf("wrote %s and %s", out, p)
}

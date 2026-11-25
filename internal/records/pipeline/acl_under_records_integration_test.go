//go:build integration

// TICK-1104 (ADR-010-A3) against a real PostgreSQL as nava_app, under RLS:
// shape R, the records-driven region leaf, returns exactly the rows Allows
// admits, the real selector picks it from the folded probe, and its plans on
// the pinned PostgreSQL are the ones A3-1 and A3-5 rest on.
package pipeline_test

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/entitytypes"
	"github.com/Navaladi2019/TicketingSystem/internal/records/pipeline"
)

// isRecords reports whether any region term of read is R.
func isRecords(read authz.ListRead) bool {
	return strings.Contains(read.Where+strings.Join(read.Branches, " "), ") IS TRUE)")
}

// Acceptance criterion 3 (A3-4): NULL org_unit_id under under and not under,
// a nested U, an empty S, an empty U and a tombstoned intermediate unit, every
// case with the probe and with C3, W0 and R forced, against Allows.
func TestUnderListRecordsShapeAgrees(t *testing.T) {
	e := migrated(t)
	w := seedShapeWorld(t, e, 40)
	ctx := context.Background()
	open := `{"cmp":{"column":"state","op":"eq","values":["open"]}}`
	self := `{"cmp":{"column":"requester_id","op":"eq","caller":"self"}}`
	caller := `{"under":{"of":"caller"}}`
	// split holds one leaf that drives a term and is negated in another: R's
	// text under a NOT, where a NULL org_unit_id must be admitted.
	split := func(leaf string) string {
		return `{"any":[{"all":[` + open + `,` + leaf + `]},{"all":[{"not":` + leaf + `},` + open + `]}]}`
	}
	check := func(label string, whens ...string) {
		t.Helper()
		for _, when := range whens {
			pol := underDoc(t, when)
			inTx(t, e, w.tenant, func(tx pgx.Tx) error {
				for _, h := range hows() {
					for _, sort := range []string{"updated_at", "created_at"} {
						page, _ := w.agrees(ctx, t, tx, pol, w.u1, h, sort)
						driven := strings.Contains(when, `"all"`) || !strings.HasPrefix(when, `{"not"`)
						if h.shape == authz.ShapeRecords && driven && !isRecords(page.read) {
							t.Errorf("%s, %s: R forced renders no R leaf: %s", label, when, page.read.Where)
						}
					}
				}
				return nil
			})
		}
	}
	cases := []string{
		w.of("R"), `{"not":` + w.of("R") + `}`, split(w.of("R")),
		w.of("N", "N1"), caller, split(caller), w.of("H"),
		`{"any":[` + w.of("H") + `,` + self + `]}`, `{"any":[` + self + `,` + w.of("R") + `]}`,
	}
	check("the whole tree", cases...)

	// Mutant r_without_is_true on the rows: without IS TRUE a NULL unit makes
	// the negated leaf NULL and its row disappears.
	inTx(t, e, w.tenant, func(tx pgx.Tx) error {
		pol := underDoc(t, split(w.of("R")))
		bound, err := authz.ResolveUnder(ctx, tx, pol, w.u1)
		if err != nil {
			return err
		}
		read := authz.ReadListShaped(bound, w.u1, "r", 2, authz.FixedListShapes(authz.ShapeRecords))
		q := listQuery(w.tenant, w.def, read, "updated_at", 100000)
		good, err := pipeline.List(ctx, tx, q)
		if err != nil {
			return err
		}
		mq := q
		mq.Branches = slices.Clone(q.Branches)
		for i, b := range mq.Branches {
			mq.Branches[i] = strings.ReplaceAll(b, ") IS TRUE)", "))")
		}
		bad, err := pipeline.List(ctx, tx, mq)
		if err != nil {
			return err
		}
		if len(bad.Records) >= len(good.Records) {
			t.Errorf("mutant r_without_is_true keeps %d of %d rows: not caught", len(bad.Records), len(good.Records))
		}

		// An empty U, bound past the planner (which folds it to false): the
		// leaf is false and its negation true, as in the other shapes.
		for _, tc := range []struct {
			when string
			want int
		}{{w.of("Th"), 0}, {split(w.of("Th")), 27}} {
			for _, shape := range []authz.ListShape{authz.ShapeC3, authz.ShapeWalk, authz.ShapeRecords} {
				bound, err := authz.ResolveUnder(ctx, tx, underDoc(t, tc.when), w.u1)
				if err != nil {
					return err
				}
				eq := listQuery(w.tenant, w.def, authz.ReadListShaped(bound, w.u1, "r", 2, authz.FixedListShapes(shape)), "updated_at", 100000)
				eq.Args = emptyU(eq.Args, w.unit["Th"])
				got, err := pipeline.List(ctx, tx, eq)
				if err != nil {
					return err
				}
				if len(got.Records) != tc.want || got.Total != int64(tc.want) {
					t.Errorf("empty U, shape %d, %s: %d rows, total %d, want %d", shape, tc.when, len(got.Records), got.Total, tc.want)
				}
			}
		}
		return nil
	})

	// N is tombstoned and N1, below it, stays live: both leave every region.
	mustExec(t, e.super, `DELETE FROM org_unit_members WHERE tenant_id = $1 AND org_unit_id = $2`, w.tenant, w.unit["N"])
	mustExec(t, e.super, `UPDATE org_units SET deleted_at = now() WHERE tenant_id = $1 AND id = $2`, w.tenant, w.unit["N"])
	rebuildClosure(t, e)
	check("N tombstoned", append(cases, w.of("N"), w.of("N1"), `{"not":`+w.of("N")+`}`, split(w.of("N1")))...)
}

// emptyU returns args with every bound U equal to {x} replaced by the empty
// array.
func emptyU(args []any, x uuid.UUID) []any {
	out := slices.Clone(args)
	for i, a := range out {
		if s, ok := a.([]string); ok && len(s) == 1 && s[0] == x.String() {
			out[i] = []string{}
		}
	}
	return out
}

// addUnits inserts n units of the world's tenant below parent, named and
// keyed prefix-1 to prefix-n, and rebuilds the closure.
func (w *shapeWorld) addUnits(t *testing.T, e env, prefix, parent string, n int) {
	t.Helper()
	mustExec(t, e.super, `INSERT INTO org_units (tenant_id, id, parent_id, keycloak_group_id, name)
		SELECT $1, gen_random_uuid(), $2, $3::text || '-' || g, $3::text || '-' || g FROM generate_series(1, $4::int) g`,
		w.tenant, w.unit[parent], prefix, n)
	rebuildClosure(t, e)
}

// A3-2 on real data: the estimate and the subtree's size pick R on both sides
// of the cap, a missing estimate never does, and every choice agrees with
// Allows whatever the estimate says.
func TestUnderListChoosesRecords(t *testing.T) {
	base := migrated(t)
	e, tr := tracedEnv(t, base)
	w := seedShapeWorld(t, e, 300)
	w.addUnits(t, e, "x", "R", 400) // |S| of R is 405
	ctx := context.Background()
	for _, tc := range []struct {
		name     string
		when     string
		estimate int64
		records  bool
		probes   int
	}{
		{"cap 405 = |S|", w.of("R"), 1336, true, 1},
		{"cap 406 = |S| + 1", w.of("R"), 1337, false, 1},
		{"a small subtree", w.of("H"), 1336, false, 1},
		{"rare", w.of("R"), 1000, false, 0},
		{"missing", w.of("R"), -1, false, 1},
		{"above alpha x U_cert", w.of("R"), 33_367, false, 1},
		{"region or requester", `{"any":[` + w.of("R") + `,{"cmp":{"column":"requester_id","op":"eq","caller":"self"}}]}`, 1336, true, 1},
	} {
		w.setEstimate(t, e, tc.estimate)
		pol := underDoc(t, tc.when)
		inTx(t, e, w.tenant, func(tx pgx.Tx) error {
			before := tr.count()
			l, err := w.list(ctx, tx, pol, w.u1, hows()[0], "updated_at", 51)
			if err != nil {
				return fmt.Errorf("%s: %w", tc.name, err)
			}
			if n := tr.count() - before; n != tc.probes {
				t.Errorf("%s: %d probes, want %d", tc.name, n, tc.probes)
			}
			if isRecords(l.read) != tc.records {
				t.Errorf("%s: R = %v, want %v", tc.name, isRecords(l.read), tc.records)
			}
			for _, sort := range []string{"updated_at", "created_at"} {
				w.agrees(ctx, t, tx, pol, w.u1, hows()[0], sort)
			}
			return nil
		})
	}
}

// recordsScan is how a plan node reads records: "Index Scan <index>", or
// "Bitmap Heap Scan <index>" for a bitmap of one index; "" for another node.
func recordsScan(n planNode) string {
	switch {
	case n.Relation != "records":
		return ""
	case n.Type == "Bitmap Heap Scan" && len(n.Plans) > 0 && n.Plans[0].Type == "Bitmap Index Scan":
		return n.Type + " " + n.Plans[0].Index
	}
	return n.Type + " " + n.Index
}

// recordsFaults is what an R plan gets wrong (A3-1, A3-5): the records are
// read once, in one of the ways in want, under (hashed SubPlan 1) IS TRUE, and
// the closure is read nowhere else than in that SubPlan, so the units drive
// nothing.
func recordsFaults(plan planNode, want []string) []string {
	var out []string
	var read []string
	plan.walk(nil, func(n planNode, path []planNode) {
		if r := recordsScan(n); r != "" {
			read = append(read, r)
			if slices.Contains(want, r) && strings.Contains(n.Filter, "(hashed SubPlan 1) IS TRUE") {
				read[len(read)-1] = "ok"
			}
		}
		if n.Relation == "org_unit_closure" && !slices.ContainsFunc(append(path, n), func(a planNode) bool { return a.Parent == "SubPlan" }) {
			out = append(out, "org_unit_closure is read outside the SubPlan by "+n.Type)
		}
		if strings.Contains(n.Join, "Semi") {
			out = append(out, "a "+n.Join+" join")
		}
	})
	if !slices.Equal(read, []string{"ok"}) {
		out = append(out, fmt.Sprintf("records are read by %q, want one of %q under (hashed SubPlan 1) IS TRUE", read, want))
	}
	return out
}

// recordsScans are the reads R may make: the page walks the sort index; the
// count reads an index led by (tenant_id, entity_definition_id), plain or as a
// bitmap. Several cost the planner the same there, so the count is pinned to
// the class: a plan outside it, records_org_unit or a Seq Scan, is A3-R2's
// trigger (Postgres & Data Modeling Specialist).
func recordsScans(t *testing.T, e env) (page, counted []string) {
	t.Helper()
	rows, err := e.super.Query(context.Background(), `SELECT indexname FROM pg_indexes
		WHERE tablename = 'records' AND indexdef LIKE '%(tenant_id, entity_definition_id,%' AND indexdef NOT LIKE '% WHERE %'`)
	if err != nil {
		t.Fatal(err)
	}
	names, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil || slices.Contains(names, "records_org_unit") || !slices.Contains(names, "records_list_updated") {
		t.Fatalf("(tenant_id, entity_definition_id, …) indexes %q, %v", names, err)
	}
	for _, n := range names {
		counted = append(counted, "Index Scan "+n, "Bitmap Heap Scan "+n)
	}
	return []string{"Index Scan records_list_updated"}, counted
}

// recordsPlanWorld is a records table R is chosen for: a subtree of each size
// in recordsPlanSizes, 20,000 records of the world's type in other units
// (m = 0, the bench_conc shape), 200,000 of another type over every unit, so
// the tenant's unit-tagged slice is not the type's, and 200,000 in each of two
// other tenants: the table is shared. Its size matters to the count: on a
// table of one tenant and up to about 420,000 rows PostgreSQL 16.10 costs a Seq
// Scan of it just below the type's index (TICK-1104's PR). It returns the root of each subtree, and of the 2,000 units that
// hold the type's records.
func recordsPlanWorld(t *testing.T, e env) (w *shapeWorld, roots map[int]uuid.UUID, other uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	w = seedShapeWorld(t, e, 1)
	// tree adds n units, ten children each, and their closure rows. Not by
	// org_unit_closure_rebuild_all, which is quadratic on a table with no
	// statistics (TICK-1054), and past the foreign keys (replica role), whose
	// checks keep the plan of the small tables they first ran on.
	mustExec(t, e.super, `SET session_replication_role = replica`)
	tree := func(prefix string, n int) uuid.UUID {
		t.Helper()
		mustExec(t, e.super, `INSERT INTO org_units (tenant_id, id, parent_id, keycloak_group_id, name)
			SELECT $1, md5($2::text || g)::uuid, CASE WHEN g > 0 THEN md5($2::text || (g - 1) / 10)::uuid END, 'kc-' || $2::text || g, $2::text || g
			FROM generate_series(0, $3::int - 1) g`, w.tenant, prefix, n)
		mustExec(t, e.super, `INSERT INTO org_unit_closure (tenant_id, ancestor_id, descendant_id)
			WITH RECURSIVE c (d, a) AS (SELECT g, g FROM generate_series(0, $3::int - 1) g UNION ALL SELECT d, (a - 1) / 10 FROM c WHERE a > 0)
			SELECT $1, md5($2::text || a)::uuid, md5($2::text || d)::uuid FROM c`, w.tenant, prefix, n)
		var root uuid.UUID
		if err := e.super.QueryRow(ctx, `SELECT md5($1::text || 0)::uuid`, prefix).Scan(&root); err != nil {
			t.Fatal(err)
		}
		return root
	}
	roots = map[int]uuid.UUID{}
	for _, n := range recordsPlanSizes() {
		roots[n] = tree(fmt.Sprintf("s%d-", n), n)
	}
	other = tree("o-", 2000)
	mustExec(t, e.super, `RESET session_replication_role`)
	mustExec(t, e.super, `ANALYZE org_units`)
	for n, root := range roots {
		if got := count(t, e.super, `SELECT count(*) FROM org_unit_closure WHERE tenant_id = $1 AND ancestor_id = $2`, w.tenant, root); got != n {
			t.Fatalf("the subtree of %d units has %d closure rows", n, got)
		}
	}
	var busy uuid.UUID
	inTx(t, e, w.tenant, func(tx pgx.Tx) error {
		if _, err := (entitytypes.Store{}).Create(ctx, tx, w.tenant, entitytypes.NewInput{
			Name: entityKey, PublishedBy: w.u1.PrincipalID, Definition: definition(schemaV1)}); err != nil {
			return err
		}
		return tx.QueryRow(ctx, `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2`, w.tenant, entityKey).Scan(&busy)
	})
	tx, err := e.super.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, "SET LOCAL session_replication_role = replica"); err != nil {
		t.Fatal(err)
	}
	for _, r := range []struct {
		tenant, def uuid.UUID
		like        string
		n           int
		number      string
	}{
		{w.tenant, w.def, "o-%", 20_000, "PC-"}, {w.tenant, busy, "%", 200_000, "PB-"},
		{uuid.New(), uuid.New(), "%", 200_000, "PX-"}, {uuid.New(), uuid.New(), "%", 200_000, "PY-"},
	} {
		if _, err := tx.Exec(ctx, `WITH u AS (SELECT id, row_number() OVER (ORDER BY id) - 1 AS i
				FROM org_units WHERE tenant_id = $6 AND name LIKE $3)
			INSERT INTO records (tenant_id, id, entity_definition_id, version, schema_version, number, state, org_unit_id, created_at, updated_at, data)
			SELECT $1, gen_random_uuid(), $2, 1, 1, $5::text || g, CASE WHEN g % 2 = 0 THEN 'open' ELSE 'closed' END, u.id,
				now() - ((g::bigint * 7919) % 100003) * interval '1 second', now() - g * interval '1 second', '{}'
			FROM generate_series(1, $4::int) g JOIN u ON u.i = g % (SELECT count(*) FROM u)`, r.tenant, r.def, r.like, r.n, r.number, w.tenant); err != nil {
			t.Fatal(err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	w.setEstimate(t, e, 20_000)
	// The visibility map too, as autovacuum leaves it.
	mustExec(t, e.super, `VACUUM ANALYZE records`)
	mustExec(t, e.super, `VACUUM ANALYZE org_unit_closure`)
	return w, roots, other
}

// recordsPlanSizes are the certified unit count and U_hash (A3-5): R runs
// below the second only.
func recordsPlanSizes() []int { return []int{10_111, 20_222} }

// recordsPlanSettings gives a transaction the planner settings of P3-01
// (bench/acl-perf/run.sh); the rest of them are PostgreSQL's defaults, which
// the hashed SubPlan depends on.
func recordsPlanSettings(ctx context.Context, t *testing.T, tx pgx.Tx) {
	t.Helper()
	if _, err := tx.Exec(ctx, `SET LOCAL effective_cache_size = '6GB'`); err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]string{"work_mem": "4MB", "hash_mem_multiplier": "2", "random_page_cost": "4"} {
		var got string
		if err := tx.QueryRow(ctx, `SELECT current_setting($1::text)`, name).Scan(&got); err != nil || got != want {
			t.Fatalf("%s = %q (%v), want P3-01's %q", name, got, err, want)
		}
	}
}

// Acceptance criterion 4 (A3-1, A3-5, A2-R4 extended), on the pinned
// PostgreSQL with P3-01's settings: R's page and count at 10,111 and 20,222
// units, the folded probe's P3 never executed on an R path, and the mutants
// r_without_is_true and r_count_driven.
func TestUnderListRecordsPlans(t *testing.T) {
	e := migrated(t)
	w, roots, other := recordsPlanWorld(t, e)
	ctx := context.Background()
	var version string
	if err := e.super.QueryRow(ctx, `SHOW server_version`).Scan(&version); err != nil {
		t.Fatal(err)
	}
	t.Logf("PostgreSQL %s", version)
	pageReads, countReads := recordsScans(t, e)
	inTx(t, e, w.tenant, func(tx pgx.Tx) error {
		recordsPlanSettings(ctx, t, tx)
		for _, n := range recordsPlanSizes() {
			pol := underDoc(t, `{"under":{"of":["`+roots[n].String()+`"]}}`)
			bound, err := authz.ResolveUnder(ctx, tx, pol, w.u1)
			if err != nil {
				return err
			}
			// The real selector: R at the certified size, A2 at U_hash.
			chosen, err := authz.ChooseListShapes(ctx, tx, bound, w.u1, w.def, 51)
			if err != nil {
				return err
			}
			if got := isRecords(authz.ReadListShaped(bound, w.u1, "r", 2, chosen)); got != (n == recordsPlanSizes()[0]) {
				t.Errorf("%d units: the selector picked R = %v", n, got)
			}
			q := listQuery(w.tenant, w.def, authz.ReadListShaped(bound, w.u1, "r", 2, authz.FixedListShapes(authz.ShapeRecords)), "updated_at", 51)
			page, args, err := pipeline.PageStatement(q)
			if err != nil {
				return err
			}
			cc, cargs := pipeline.CountStatement(q), append([]any{q.TenantID, q.DefinitionID}, q.Args...)
			for _, s := range []struct {
				name, sql string
				reads     []string
				args      []any
			}{{"page", page, pageReads, args}, {"count", cc, countReads, cargs}} {
				plan, raw := explain(ctx, t, tx, s.sql, s.args)
				if faults := recordsFaults(plan, s.reads); len(faults) > 0 {
					t.Errorf("R %s at %d units: %v\n%s\nEXPLAIN (FORMAT JSON):\n%s", s.name, n, faults, s.sql, raw)
				} else {
					plan.walk(nil, func(x planNode, _ []planNode) {
						if r := recordsScan(x); r != "" {
							t.Logf("R %s at %d units: %s", s.name, n, r)
						}
					})
				}
				mut := strings.ReplaceAll(s.sql, ") IS TRUE)", "))")
				if plan, raw := explain(ctx, t, tx, mut, s.args); mut == s.sql || len(recordsFaults(plan, s.reads)) == 0 {
					t.Errorf("mutant r_without_is_true is not caught, %s at %d units\n%s", s.name, n, raw)
				}
			}
			// r_count_driven: the count read through A2's region driver.
			dq := q
			for _, d := range authz.ReadList(bound, w.u1, "r", 2).Drivers {
				dq.CountDrivers = []pipeline.CountDriver{{From: d.From, Key: d.Key}}
			}
			if plan, raw := explain(ctx, t, tx, pipeline.CountStatement(dq), cargs); len(recordsFaults(plan, countReads)) == 0 {
				t.Errorf("mutant r_count_driven is not caught at %d units\n%s", n, raw)
			}
		}

		// The folded probe: with the cap reached P3 never executes; below it,
		// in the 2,000 units that hold the type, it does.
		const limit, below = int64(6061), int64(20_222)
		for _, tc := range []struct {
			name string
			root uuid.UUID
			n    int64
			runs bool
		}{{"cap reached", roots[10_111], 10_111, false}, {"U_hash reached", roots[20_222], 20_222, true}, {"below the cap", other, 2000, true}} {
			sql := authz.FoldedProbeStatement(authz.KStarMin)
			pargs := []any{w.tenant, []string{tc.root.String()}, w.def, limit, below}
			var n int64
			var m *int64
			if err := tx.QueryRow(ctx, sql, pargs...).Scan(&n, &m); err != nil {
				return err
			}
			if n != tc.n || (m != nil) != tc.runs {
				t.Errorf("folded probe, %s: n %d, P3 ran %v; want %d, %v", tc.name, n, m != nil, tc.n, tc.runs)
			}
			plan, raw := explain(ctx, t, tx, "ANALYZE "+sql, pargs)
			scans, executed := 0, 0
			plan.walk(nil, func(n planNode, _ []planNode) {
				if n.Relation == "records" || strings.HasPrefix(n.Index, "records_") {
					scans++
					if n.Loops == nil || *n.Loops > 0 {
						executed++
					}
				}
			})
			if scans == 0 || (executed > 0) != tc.runs {
				t.Errorf("folded probe, %s: %d of %d reads of records executed\n%s", tc.name, executed, scans, raw)
			}
		}
		return nil
	})
	reportGenericPlans(t, e, w, roots)
}

// reportGenericPlans logs R's plans under plan_cache_mode = force_generic_plan
// (A2-R4, extended). The server never plans generically (QueryExecModeExec),
// so a miss is reported, not failed.
func reportGenericPlans(t *testing.T, e env, w *shapeWorld, roots map[int]uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	pageReads, countReads := recordsScans(t, e)
	inTx(t, e, w.tenant, func(tx pgx.Tx) error {
		recordsPlanSettings(ctx, t, tx)
		if _, err := tx.Exec(ctx, `SET LOCAL plan_cache_mode = force_generic_plan`); err != nil {
			return err
		}
		for _, n := range recordsPlanSizes() {
			bound, err := authz.ResolveUnder(ctx, tx, underDoc(t, `{"under":{"of":["`+roots[n].String()+`"]}}`), w.u1)
			if err != nil {
				return err
			}
			q := listQuery(w.tenant, w.def, authz.ReadListShaped(bound, w.u1, "r", 2, authz.FixedListShapes(authz.ShapeRecords)), "updated_at", 51)
			page, args, err := pipeline.PageStatement(q)
			if err != nil {
				return err
			}
			for _, s := range []struct {
				name, sql string
				reads     []string
				args      []any
			}{
				{"page", page, pageReads, args},
				{"count", pipeline.CountStatement(q), countReads, append([]any{q.TenantID, q.DefinitionID}, q.Args...)},
			} {
				name := fmt.Sprintf("r_%s_%d", s.name, n)
				if _, err := tx.Exec(ctx, "PREPARE "+name+" AS "+s.sql); err != nil {
					return fmt.Errorf("prepare %s: %w", s.name, err)
				}
				plan, raw := explain(ctx, t, tx, "EXECUTE "+name+literals(s.args), nil)
				if faults := recordsFaults(plan, s.reads); len(faults) > 0 {
					t.Logf("GENERIC PLAN MISS: R %s at %d units: %v\n%s", s.name, n, faults, raw)
				} else {
					t.Logf("GENERIC PLAN OK: R %s at %d units", s.name, n)
				}
			}
		}
		return nil
	})
}

// literals renders args as an EXECUTE argument list. The arguments are the
// statement's own: uuids, arrays of uuid texts and integers.
func literals(args []any) string {
	out := make([]string, len(args))
	for i, a := range args {
		switch v := a.(type) {
		case []string:
			out[i] = "'{" + strings.Join(v, ",") + "}'"
		case int:
			out[i] = fmt.Sprint(v)
		default:
			out[i] = fmt.Sprintf("'%v'", v)
		}
	}
	return "(" + strings.Join(out, ", ") + ")"
}

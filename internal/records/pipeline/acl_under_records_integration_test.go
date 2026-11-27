//go:build integration

// TICK-1104 (ADR-010-A3) against a real PostgreSQL as nava_app, under RLS:
// shape R, the records-driven region leaf, returns exactly the rows Allows
// admits, the real selector picks it from the folded probe, and its plans on
// the pinned PostgreSQL are the ones A3-1 and A3-5 rest on.
package pipeline_test

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

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

// pageRead is the R page's read of records (A3-1, A4-3): a walk of the sort
// index with heap reads. coveringRead is the R count's (ADR-010-A4 A4-5): an
// Index Only Scan of the same index, which holds org_unit_id as INCLUDE.
const (
	pageRead     = "Index Scan records_list_updated"
	coveringRead = "Index Only Scan records_list_updated"
)

// typeSliceReads are the reads A4-7's generic-plan report accepts for the
// count: the covering Index Only Scan, or a walk or a bitmap of an index led by
// (tenant_id, entity_definition_id), the type's slice.
func typeSliceReads(t *testing.T, e env) []string {
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
	out := []string{coveringRead}
	for _, n := range names {
		out = append(out, "Index Scan "+n, "Bitmap Heap Scan "+n)
	}
	return out
}

// hashedIsTrue is R's leaf in a plan's filter: the hashed SubPlan, under IS
// TRUE.
var hashedIsTrue = regexp.MustCompile(`\(hashed SubPlan \d+\) IS TRUE`)

// rowsRead is what a node of an EXPLAIN ANALYZE plan read: (Actual Rows +
// Rows Removed by Filter) × Loops; -1 when the plan was not executed.
func rowsRead(n planNode) int64 {
	if n.Loops == nil {
		return -1
	}
	return int64((n.ActualRows + n.Removed) * *n.Loops)
}

// coveringFaults is what A4-5's pin finds wrong with arm, an R count's read of
// records in an EXPLAIN ANALYZE plan: it must be an Index Only Scan of
// records_list_updated under (hashed SubPlan n) IS TRUE, whose Index Cond
// starts with tenant_id, reading at most typeRows, the calling tenant's
// records of the type (A4-8: no row of another tenant).
func coveringFaults(arm planNode, typeRows int64) []string {
	var out []string
	if r := recordsScan(arm); r != coveringRead || !hashedIsTrue.MatchString(arm.Filter) {
		out = append(out, fmt.Sprintf("records are read by %q with filter %q, want %q under (hashed SubPlan n) IS TRUE", r, arm.Filter, coveringRead))
	}
	if !strings.HasPrefix(strings.TrimLeft(arm.IndexCond, "("), "tenant_id = ") {
		out = append(out, fmt.Sprintf("Index Cond %q does not start with tenant_id", arm.IndexCond))
	}
	if n := rowsRead(arm); n < 0 || n > typeRows {
		out = append(out, fmt.Sprintf("%d rows read, want at most the type's %d", n, typeRows))
	}
	return out
}

// sortsOrGathers names every Sort or Gather node of plan.
func sortsOrGathers(plan planNode) []string {
	var out []string
	plan.walk(nil, func(n planNode, _ []planNode) {
		if strings.HasSuffix(n.Type, "Sort") || strings.HasPrefix(n.Type, "Gather") {
			out = append(out, "a "+n.Type)
		}
	})
	return out
}

// countPinFaults is A4-5's pin on a one-arm R count's EXPLAIN ANALYZE plan:
// A3's shape (one read of records, the closure only in the SubPlan, no
// semi-join), that read covered, and no Sort or Gather.
func countPinFaults(plan planNode, typeRows int64) []string {
	out := append(recordsFaults(plan, []string{coveringRead}), sortsOrGathers(plan)...)
	plan.walk(nil, func(n planNode, _ []planNode) {
		if n.Relation == "records" {
			out = append(out, coveringFaults(n, typeRows)...)
		}
	})
	return out
}

// recordsReads are the nodes of plan that read records.
func recordsReads(plan planNode) []planNode {
	var out []planNode
	plan.walk(nil, func(n planNode, _ []planNode) {
		if recordsScan(n) != "" {
			out = append(out, n)
		}
	})
	return out
}

// planWorld is the records table of a plan test: the world's type has
// typeRows records in the 2,000 units outside every subtree (m = 0, the
// bench_conc shape), another type of the tenant busyRows over every unit, and
// each entry of others is another tenant's rows. scatter spreads the type's
// updated_at over the table's order, as A4-6's world does; otherwise it
// follows the insert order. filled sets every row's requester, assignee,
// group and creator, which the index does not hold, so the heap is as wide as
// A4-6's measured world's. estimate is N̂_e.
type planWorld struct {
	typeRows, busyRows int
	others             []int
	scatter, filled    bool
	estimate           int64
}

// sharedWorld is CI-3's (TICK-1104): 20,000 records of the type, 200,000 of
// another type over every unit, so the tenant's unit-tagged slice is not the
// type's, and 200,000 in each of two other tenants: the table is shared. That
// matters to the count: without A4's covering index PostgreSQL 16.10 picks a
// Seq Scan of records when the table's rows are few next to N̂_e and the
// tenant holds a large share of them (A3-R2; CI-4's world).
func sharedWorld() planWorld {
	return planWorld{typeRows: 20_000, busyRows: 200_000, others: []int{200_000, 200_000}, estimate: 20_000}
}

// ownedWorld is CI-4's (ADR-010-A4 A4-6): one tenant holds 150,000 of the
// table's 200,000 empty records (75 %), 33,366 of them of the type (N̂_e),
// beside 50,000 of other tenants. Without the covering index the R count is a
// Seq Scan there; with it, an Index Only Scan.
func ownedWorld() planWorld {
	return planWorld{typeRows: 33_366, busyRows: 116_634, others: []int{25_000, 25_000}, scatter: true, filled: true, estimate: 33_366}
}

// recordsPlanWorld builds pw: a subtree of each size in recordsPlanSizes and
// 2,000 units outside them, which hold the type's records. It returns the
// root of each subtree, and of the 2,000 units.
func recordsPlanWorld(t *testing.T, e env, pw planWorld) (w *shapeWorld, roots map[int]uuid.UUID, other uuid.UUID) {
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
	type batch struct {
		tenant, def uuid.UUID
		like        string
		n           int
		number      string
		scatter     bool
	}
	batches := []batch{{w.tenant, w.def, "o-%", pw.typeRows, "PC-", pw.scatter}, {w.tenant, busy, "%", pw.busyRows, "PB-", false}}
	for i, n := range pw.others {
		batches = append(batches, batch{uuid.New(), uuid.New(), "%", n, fmt.Sprintf("P%d-", i), false})
	}
	for _, r := range batches {
		if _, err := tx.Exec(ctx, `WITH u AS (SELECT id, row_number() OVER (ORDER BY id) - 1 AS i
				FROM org_units WHERE tenant_id = $6 AND name LIKE $3)
			INSERT INTO records (tenant_id, id, entity_definition_id, version, schema_version, number, state, org_unit_id, created_at, updated_at, data,
				requester_id, assignee_id, assignment_group_id, created_by)
			SELECT $1, gen_random_uuid(), $2, 1, 1, $5::text || g, CASE WHEN g % 2 = 0 THEN 'open' ELSE 'closed' END, u.id,
				now() - ((g::bigint * 7919) % 100003) * interval '1 second',
				now() - CASE WHEN $7 THEN (g::bigint * 104729) % 1000003 ELSE g END * interval '1 second', '{}',
				CASE WHEN $8 THEN md5('u' || g % 997)::uuid END, CASE WHEN $8 THEN md5('a' || g % 499)::uuid END,
				CASE WHEN $8 THEN md5('gr' || g % 50)::uuid END, CASE WHEN $8 THEN md5('u' || g % 997)::uuid END
			FROM generate_series(1, $4::int) g JOIN u ON u.i = g % (SELECT count(*) FROM u)`, r.tenant, r.def, r.like, r.n, r.number, w.tenant, r.scatter, pw.filled); err != nil {
			t.Fatal(err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	w.setEstimate(t, e, pw.estimate)
	// The visibility map too, as autovacuum leaves it. The test cluster runs
	// with synchronous_commit = off, so the inserts' commit records may not be
	// flushed yet; VACUUM cannot set their hint bits then and leaves every
	// page out of the map, so an Index Only Scan costs heap fetches and the
	// plans pinned here change. CHECKPOINT flushes the WAL first.
	mustExec(t, e.super, `CHECKPOINT`)
	mustExec(t, e.super, `VACUUM ANALYZE org_unit_closure`)
	waitAllVisible(ctx, t, e, "records")
	return w, roots, other
}

// waitAllVisible vacuums table until at least 99 % of its pages are in the
// visibility map, for at most allVisibleLimit (TICK-1173). One VACUUM cannot
// promise that: it leaves a page out while any backend in the database (an
// autovacuum or autoanalyze worker, an open pool connection) holds a snapshot
// older than the page's inserts, and it skips pages whose cleanup lock it
// cannot take at once. FREEZE makes the pass aggressive, so it waits for those
// locks instead. On failure the message names every backend that holds an
// xmin, which is the one that pinned the horizon.
func waitAllVisible(ctx context.Context, t *testing.T, e env, table string) {
	t.Helper()
	deadline := time.Now().Add(allVisibleLimit)
	var visible float64
	for attempt := 1; ; attempt++ {
		mustExec(t, e.super, `CHECKPOINT`)
		mustExec(t, e.super, `VACUUM (FREEZE, ANALYZE) `+pgx.Identifier{table}.Sanitize())
		if err := e.super.QueryRow(ctx, `SELECT relallvisible::float8 / greatest(relpages, 1) FROM pg_class WHERE oid = $1::regclass`, table).Scan(&visible); err != nil {
			t.Fatal(err)
		}
		if visible >= 0.99 {
			if attempt > 1 {
				t.Logf("%s all-visible after %d VACUUM passes", table, attempt)
			}
			return
		}
		if time.Now().After(deadline) {
			var holders string
			if err := e.super.QueryRow(ctx, `SELECT coalesce(string_agg(format('pid=%s type=%s state=%s xmin=%s xact_start=%s query=%L',
					pid, backend_type, coalesce(state, '-'), coalesce(backend_xmin::text, '-'), coalesce(xact_start::text, '-'), left(query, 80)), '; '), 'none')
				FROM pg_stat_activity
				WHERE (datname = current_database() OR datname IS NULL) AND pid <> pg_backend_pid()
				  AND (backend_xmin IS NOT NULL OR backend_xid IS NOT NULL OR backend_type LIKE 'autovacuum%')`).Scan(&holders); err != nil {
				t.Fatal(err)
			}
			t.Fatalf("%s is %.0f %% all-visible after %d VACUUM passes in %v, want all of it; backends that may hold the xmin horizon: %s",
				table, 100*visible, attempt, allVisibleLimit, holders)
		}
		time.Sleep(200 * time.Millisecond)
	}
}

// allVisibleLimit bounds waitAllVisible.
const allVisibleLimit = 5 * time.Second

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

// Acceptance criterion 4 (A3-1, A3-5, A2-R4 extended) and ADR-010-A4's CI-3
// and CI-5, on the pinned PostgreSQL with P3-01's settings, in the shared
// world: R's page and count at 10,111 and 20,222 units, the count an Index
// Only Scan of the covering records_list_updated (A4-5), the folded probe's P3
// never executed on an R path, and the mutants r_without_is_true and
// r_count_driven.
func TestUnderListRecordsPlans(t *testing.T) {
	e := migrated(t)
	pw := sharedWorld()
	w, roots, other := recordsPlanWorld(t, e, pw)
	ctx := context.Background()
	// The tenant's records of the type: pw's and seedShapeWorld's.
	typeRows := int64(count(t, e.super, `SELECT count(*) FROM records WHERE tenant_id = $1 AND entity_definition_id = $2`, w.tenant, w.def))
	var version string
	if err := e.super.QueryRow(ctx, `SHOW server_version`).Scan(&version); err != nil {
		t.Fatal(err)
	}
	t.Logf("PostgreSQL %s", version)
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

			// The page: A3-1's pin, unchanged (A4-3).
			if plan, raw := explain(ctx, t, tx, page, args); len(recordsFaults(plan, []string{pageRead})) > 0 {
				t.Errorf("R page at %d units: %v\n%s\nEXPLAIN (FORMAT JSON):\n%s", n, recordsFaults(plan, []string{pageRead}), page, raw)
			}
			// The count: A4-5's pin (CI-3), executed for the rows it reads.
			if plan, raw := explain(ctx, t, tx, "ANALYZE "+cc, cargs); len(countPinFaults(plan, typeRows)) > 0 {
				t.Errorf("R count at %d units: %v\n%s\nEXPLAIN (ANALYZE, FORMAT JSON):\n%s", n, countPinFaults(plan, typeRows), cc, raw)
			} else {
				ios := recordsReads(plan)[0]
				t.Logf("R count at %d units: %s, %d rows read", n, recordsScan(ios), rowsRead(ios))
			}
			for _, s := range []struct {
				name, sql, read string
				args            []any
			}{{"page", page, pageRead, args}, {"count", cc, coveringRead, cargs}} {
				mut := strings.ReplaceAll(s.sql, ") IS TRUE)", "))")
				if plan, raw := explain(ctx, t, tx, mut, s.args); mut == s.sql || len(recordsFaults(plan, []string{s.read})) == 0 {
					t.Errorf("mutant r_without_is_true is not caught, %s at %d units\n%s", s.name, n, raw)
				}
			}
			// r_count_driven: the count read through A2's region driver.
			dq := q
			for _, d := range authz.ReadList(bound, w.u1, "r", 2).Drivers {
				dq.CountDrivers = []pipeline.CountDriver{{From: d.From, Key: d.Key}}
			}
			if plan, raw := explain(ctx, t, tx, pipeline.CountStatement(dq), cargs); len(recordsFaults(plan, []string{coveringRead})) == 0 {
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
	unionCountPins(t, e, w, roots[recordsPlanSizes()[0]], typeRows)
}

// ADR-010-A4 CI-4 (A4-5, A4-6), in a table one tenant mostly owns: R's count
// is an Index Only Scan of the covering records_list_updated; with Index Only
// Scans off the runner-up's records node costs at least 1.3× as much, so the
// world is not at a tie; and mutant r_count_no_covering, the same world with
// today's index, is a Seq Scan reading more than the tenant's rows, which is
// what makes this test red before A4. CI-5 runs here too.
func TestUnderListRecordsCountCovered(t *testing.T) {
	e := migrated(t)
	pw := ownedWorld()
	w, roots, _ := recordsPlanWorld(t, e, pw)
	ctx := context.Background()
	// The tenant's records of the type, and all of its records.
	typeRows := int64(count(t, e.super, `SELECT count(*) FROM records WHERE tenant_id = $1 AND entity_definition_id = $2`, w.tenant, w.def))
	tenantRows := int64(count(t, e.super, `SELECT count(*) FROM records WHERE tenant_id = $1`, w.tenant))
	// count explains R's count at n units, executed when analyze is set.
	count := func(tx pgx.Tx, n int, analyze bool) (planNode, string) {
		t.Helper()
		bound, err := authz.ResolveUnder(ctx, tx, underDoc(t, `{"under":{"of":["`+roots[n].String()+`"]}}`), w.u1)
		if err != nil {
			t.Fatal(err)
		}
		q := listQuery(w.tenant, w.def, authz.ReadListShaped(bound, w.u1, "r", 2, authz.FixedListShapes(authz.ShapeRecords)), "updated_at", 51)
		sql := pipeline.CountStatement(q)
		if analyze {
			sql = "ANALYZE " + sql
		}
		return explain(ctx, t, tx, sql, append([]any{q.TenantID, q.DefinitionID}, q.Args...))
	}
	setIOS := func(tx pgx.Tx, on string) {
		t.Helper()
		if _, err := tx.Exec(ctx, "SET LOCAL enable_indexonlyscan = "+on); err != nil {
			t.Fatal(err)
		}
	}
	inTx(t, e, w.tenant, func(tx pgx.Tx) error {
		recordsPlanSettings(ctx, t, tx)
		for _, n := range recordsPlanSizes() {
			plan, raw := count(tx, n, true)
			if faults := countPinFaults(plan, typeRows); len(faults) > 0 {
				// What the planner weighed: the Seq Scan's alternative, and
				// the table's pages and visibility map it costs them from.
				if _, err := tx.Exec(ctx, "SET LOCAL enable_seqscan = off"); err != nil {
					return err
				}
				_, noSeq := count(tx, n, false)
				if _, err := tx.Exec(ctx, "SET LOCAL enable_seqscan = on"); err != nil {
					return err
				}
				var stats string
				if err := tx.QueryRow(ctx, `SELECT string_agg(relname || ' pages=' || relpages || ' visible=' || relallvisible || ' tuples=' || reltuples, '; ')
					FROM pg_class WHERE relname IN ('records', 'records_list_updated')`).Scan(&stats); err != nil {
					return err
				}
				t.Errorf("CI-4, R count at %d units: %v\n%s\nwith Seq Scans off:\n%s\n%s", n, faults, raw, noSeq, stats)
				continue
			}
			ios := recordsReads(plan)[0]
			// The guard: the records node's own run cost only. The whole
			// plan's would add the SubPlan's lost Index Only Scan of the
			// closure to every runner-up (A4, rejected options).
			setIOS(tx, "off")
			alt, altRaw := count(tx, n, false)
			setIOS(tx, "on")
			reads := recordsReads(alt)
			if len(reads) != 1 {
				t.Errorf("CI-4 guard at %d units: %d reads of records\n%s", n, len(reads), altRaw)
				continue
			}
			iosCost, altCost := ios.TotalCost-ios.StartupCost, reads[0].TotalCost-reads[0].StartupCost
			t.Logf("CI-4 at %d units: %d rows read; records node run cost %s %.0f, runner-up %s %.0f (%.2fx)",
				n, rowsRead(ios), recordsScan(ios), iosCost, recordsScan(reads[0]), altCost, altCost/iosCost)
			if altCost < 1.3*iosCost {
				t.Errorf("world at a tie: at %d units the runner-up %s costs %.0f against the Index Only Scan's %.0f, under 1.3x\n%s",
					n, recordsScan(reads[0]), altCost, iosCost, altRaw)
			}
		}
		return nil
	})

	// Mutant r_count_no_covering: today's index, without the INCLUDE.
	withListUpdated(t, e, plainListUpdated, func() {
		inTx(t, e, w.tenant, func(tx pgx.Tx) error {
			recordsPlanSettings(ctx, t, tx)
			for _, n := range recordsPlanSizes() {
				plan, raw := count(tx, n, true)
				reads := recordsReads(plan)
				if len(countPinFaults(plan, typeRows)) == 0 {
					t.Errorf("mutant r_count_no_covering is not caught at %d units\n%s", n, raw)
				}
				if len(reads) != 1 || reads[0].Type != "Seq Scan" || rowsRead(reads[0]) <= tenantRows {
					t.Errorf("mutant r_count_no_covering at %d units: want one Seq Scan reading more than the tenant's %d rows\n%s", n, tenantRows, raw)
				} else {
					t.Logf("mutant r_count_no_covering at %d units: Seq Scan, %d rows read", n, rowsRead(reads[0]))
				}
			}
			return nil
		})
	})
	unionCountPins(t, e, w, roots[recordsPlanSizes()[0]], typeRows)
}

// The two definitions of records_list_updated: ADR-010-A4 A4-1's, and the one
// before it, which the mutant and CI-5's baseline build.
const (
	coveringListUpdated = `CREATE INDEX records_list_updated ON records (tenant_id, entity_definition_id, updated_at DESC, id DESC) INCLUDE (org_unit_id)`
	plainListUpdated    = `CREATE INDEX records_list_updated ON records (tenant_id, entity_definition_id, updated_at DESC, id DESC)`
)

// withListUpdated runs fn with records_list_updated built as def, then
// rebuilds the migrated, covering one.
func withListUpdated(t *testing.T, e env, def string, fn func()) {
	t.Helper()
	swap := func(def string) {
		mustExec(t, e.super, `DROP INDEX records_list_updated`)
		mustExec(t, e.super, def)
	}
	swap(def)
	defer swap(coveringListUpdated)
	fn()
}

// unionCountPins is CI-5 (A4-5) at root's subtree: R's count when the policy
// has a second branch, requester, so the count is a UNION ALL of one arm per
// branch and reads records twice. With under first, R's arm carries no
// negation and meets the one-arm pin. With requester first, R's arm carries
// requester's negation, a column the index does not hold (A4-R5): it is pinned
// only to its plan class with today's index, or an Index Only Scan.
func unionCountPins(t *testing.T, e env, w *shapeWorld, root uuid.UUID, typeRows int64) {
	t.Helper()
	ctx := context.Background()
	under := `{"under":{"of":["` + root.String() + `"]}}`
	self := `{"cmp":{"column":"requester_id","op":"eq","caller":"self"}}`
	underFirst, requesterFirst := `{"any":[`+under+`,`+self+`]}`, `{"any":[`+self+`,`+under+`]}`
	// count runs the count of when under EXPLAIN ANALYZE and returns its plan,
	// R's arm (the read of records under (hashed SubPlan n) IS TRUE) and the
	// raw plan.
	count := func(tx pgx.Tx, when string) (plan, arm planNode, raw string) {
		t.Helper()
		bound, err := authz.ResolveUnder(ctx, tx, underDoc(t, when), w.u1)
		if err != nil {
			t.Fatal(err)
		}
		q := listQuery(w.tenant, w.def, authz.ReadListShaped(bound, w.u1, "r", 2, authz.FixedListShapes(authz.ShapeRecords)), "updated_at", 51)
		plan, raw = explain(ctx, t, tx, "ANALYZE "+pipeline.CountStatement(q), append([]any{q.TenantID, q.DefinitionID}, q.Args...))
		reads := recordsReads(plan)
		arms := slices.DeleteFunc(slices.Clone(reads), func(n planNode) bool { return !hashedIsTrue.MatchString(n.Filter) })
		if len(reads) != 2 || len(arms) != 1 {
			t.Fatalf("CI-5, %s: %d reads of records, %d of them R's arm; want 2 and 1\n%s", when, len(reads), len(arms), raw)
		}
		return plan, arms[0], raw
	}
	var negated string
	inTx(t, e, w.tenant, func(tx pgx.Tx) error {
		recordsPlanSettings(ctx, t, tx)
		plan, arm, raw := count(tx, underFirst)
		if faults := append(coveringFaults(arm, typeRows), sortsOrGathers(plan)...); len(faults) > 0 {
			t.Errorf("CI-5, under first: R's arm: %v\n%s", faults, raw)
		} else {
			t.Logf("CI-5, under first: R's arm is a %s, %d rows read", recordsScan(arm), rowsRead(arm))
		}
		_, arm, _ = count(tx, requesterFirst)
		negated = arm.Type
		return nil
	})
	withListUpdated(t, e, plainListUpdated, func() {
		inTx(t, e, w.tenant, func(tx pgx.Tx) error {
			recordsPlanSettings(ctx, t, tx)
			_, arm, raw := count(tx, requesterFirst)
			t.Logf("CI-5, requester first: R's arm is a %s; with today's index a %s", negated, arm.Type)
			if negated != "Index Only Scan" && negated != arm.Type {
				t.Errorf("CI-5, requester first: R's arm is a %s, not today's %s\n%s", negated, arm.Type, raw)
			}
			return nil
		})
	})
}

// reportGenericPlans logs R's plans under plan_cache_mode = force_generic_plan
// (A2-R4, extended; ADR-010-A4 A4-7, CI-8). The server never plans
// generically (QueryExecModeExec), so a miss is reported, not failed. The
// count's plan is logged whole: it must show an Index Only Scan, an index walk
// or a bitmap of the type's slice.
func reportGenericPlans(t *testing.T, e env, w *shapeWorld, roots map[int]uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	countReads := typeSliceReads(t, e)
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
				{"page", page, []string{pageRead}, args},
				{"count", pipeline.CountStatement(q), countReads, append([]any{q.TenantID, q.DefinitionID}, q.Args...)},
			} {
				name := fmt.Sprintf("r_%s_%d", s.name, n)
				if _, err := tx.Exec(ctx, "PREPARE "+name+" AS "+s.sql); err != nil {
					return fmt.Errorf("prepare %s: %w", s.name, err)
				}
				plan, raw := explain(ctx, t, tx, "EXECUTE "+name+literals(s.args), nil)
				var reads []string
				for _, r := range recordsReads(plan) {
					reads = append(reads, recordsScan(r))
				}
				if faults := recordsFaults(plan, s.reads); len(faults) > 0 {
					t.Logf("GENERIC PLAN MISS: R %s at %d units: %v\n%s", s.name, n, faults, raw)
				} else {
					t.Logf("GENERIC PLAN OK: R %s at %d units: %q", s.name, n, reads)
				}
				if s.name == "count" {
					t.Logf("GENERIC PLAN, R count at %d units:\n%s", n, raw)
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

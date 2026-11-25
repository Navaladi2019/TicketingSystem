package authz

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"os"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

// Tests for TICK-1036 (ADR-010-A2 A2-5, A2-5a, A2-9, A2-10) that need no
// database: K*, the rare-type rule, the SQL each shape renders, and the
// metric descriptors. The probe, the switch on real data, the plans and the
// agreement with Allows are integration tests against PostgreSQL
// (internal/records/pipeline/acl_under_shape_integration_test.go).

// Acceptance test 1: both clamp ends, the bench point, a missing row.
func TestKStar(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		page     int
		estimate int64
		known    bool
		want     int
		slack    int
	}{
		{"floor: N_e = 100", 51, 100, true, KStarMin, 0},
		{"ceiling: N_e = 10^9", 51, 1_000_000_000, true, KStarMax, 0},
		{"bench: N_e = 400k, page 51", 51, 400_000, true, 3613, 1},
		{"missing row", 51, 0, false, KStarMin, 0},
		{"missing row ignores a stray value", 51, 400_000, false, KStarMin, 0},
		{"an empty type", 51, 0, true, KStarMin, 0},
		{"page 101, N_e = 1M: 0.8 x sqrt(1.01e8)", 101, 1_000_000, true, 8039, 1},
	} {
		got := KStar(tc.page, tc.estimate, tc.known)
		if got < tc.want-tc.slack || got > tc.want+tc.slack {
			t.Errorf("%s: K* = %d, want %d ± %d", tc.name, got, tc.want, tc.slack)
		}
	}
}

// rareCases are A2-5a's decisions: N̂_e ≤ K* is rare; a missing row is not,
// so P3 runs (R2-B1 rule 1).
func rareCases() []struct {
	estimate int64
	known    bool
	k        int
	rare     bool
} {
	return []struct {
		estimate int64
		known    bool
		k        int
		rare     bool
	}{
		{500, true, 1000, true},
		{1000, true, 1000, true},
		{1001, true, 1000, false},
		{0, true, 1000, true},
		{0, false, 1000, false},
		{400_000, true, 3613, false},
	}
}

// checkRare returns the first case rare gets wrong, or "".
func checkRare(rare func(int64, bool, int) bool) string {
	for _, c := range rareCases() {
		if rare(c.estimate, c.known, c.k) != c.rare {
			return fmt.Sprintf("%+v", c)
		}
	}
	return ""
}

// Acceptance tests 1 and 3 (the decision) and mutant 6
// rare_shortcut_on_missing_row: the shortcut taken on a missing row.
func TestRareTypeShortcut(t *testing.T) {
	t.Parallel()
	if bad := checkRare(rareType); bad != "" {
		t.Errorf("rareType wrong on %s", bad)
	}
	mutant := func(estimate int64, _ bool, k int) bool { return estimate <= int64(k) }
	if checkRare(mutant) == "" {
		t.Error("mutant rare_shortcut_on_missing_row is not caught")
	}
}

// The policies a list can hold a region term in: alone, ANDed with a cmp,
// as a branch after or before another, and only under a not (no driver).
func shapeCases() []struct{ name, when string } {
	self := `{"cmp":{"column":"requester_id","op":"eq","caller":"self"}}`
	open := `{"cmp":{"column":"state","op":"eq","values":["open"]}}`
	return []struct{ name, when string }{
		{"leaf alone", `{"under":{"of":"caller"}}`},
		{"leaf and a cmp", `{"all":[{"under":{"of":"caller"}},` + open + `]}`},
		{"requester or region", `{"any":[` + self + `,{"all":[` + open + `,{"under":{"of":"caller"}}]}]}`},
		{"region or requester", `{"any":[` + utOf(utUnit(2)) + `,` + self + `]}`},
		{"not under", `{"not":{"under":{"of":"caller"}}}`},
	}
}

// pageText and countText are a ListRead's page and count terms as one text.
func pageText(lr ListRead) string  { return lr.Where + "|" + strings.Join(lr.Branches, "|") }
func countText(lr ListRead) string { return lr.CountWhere + "|" + strings.Join(lr.CountBranches, "|") }

// shapeFaults returns what lr gets wrong for shapes, compared with the
// unprobed read (W0 page, CC count): the count is CC whatever the probe
// chose, the page's region terms are in the chosen shape, and a C3 page holds
// no OFFSET 0 walk probe (acceptance tests 10 and 6 count_follows_probe).
func shapeFaults(lr, unprobed ListRead, c3 bool) []string {
	var out []string
	if countText(lr) != countText(unprobed) || !reflect.DeepEqual(lr.Drivers, unprobed.Drivers) {
		out = append(out, "the count follows the probe")
	}
	if !reflect.DeepEqual(lr.Args, unprobed.Args) {
		out = append(out, "the arguments follow the probe")
	}
	if strings.Contains(countText(lr), "OFFSET 0") || strings.Contains(countText(lr), "ARRAY(") {
		out = append(out, "the count is not in the predicate form")
	}
	terms := append([]string{lr.Where}, lr.Branches...)
	if lr.Branches != nil {
		terms = terms[1:]
	}
	for i, term := range terms {
		driven := lr.Drivers[i].From != ""
		switch {
		case driven && c3 && (!lr.Materialize[i] || !strings.Contains(term, " = ANY(ARRAY(SELECT acl_ou.descendant_id")):
			out = append(out, fmt.Sprintf("term %d is not C3", i))
		case driven && !c3 && (lr.Materialize[i] || !strings.Contains(term, "AND acl_ou.descendant_id = r.org_unit_id OFFSET 0))")):
			out = append(out, fmt.Sprintf("term %d is not W0", i))
		case !driven && lr.Materialize[i]:
			out = append(out, fmt.Sprintf("term %d is materialized without a region driver", i))
		}
	}
	if c3 && slices.Contains(lr.Materialize, true) && strings.Contains(pageText(lr), "OFFSET 0") {
		out = append(out, "a C3 page holds an OFFSET 0 walk probe")
	}
	return out
}

// Acceptance tests 2 (the switch's SQL), 10 and mutant count_follows_probe.
func TestListShapesRenderTheChosenPageAndKeepTheCount(t *testing.T) {
	t.Parallel()
	for _, tc := range shapeCases() {
		pol := utBound(t, tc.when)
		unprobed := ReadList(pol, utMe(), "r", 2)
		// The count is TICK-1034's: the same text as before A2.
		var cc string
		if br, _ := ReadBranches(pol, utMe(), "r", 2); len(br) >= 2 {
			cc = "|" + strings.Join(br, "|")
		} else {
			pred, _ := CompileRead(pol, utMe(), "r", 2)
			cc = pred + "|"
		}
		if countText(unprobed) != cc {
			t.Errorf("%s: count terms %q, want CompileRead/ReadBranches' %q", tc.name, countText(unprobed), cc)
		}
		for _, s := range []struct {
			name string
			sh   ListShapes
			c3   bool
		}{{"W0", FixedListShapes(ShapeWalk), false}, {"C3", FixedListShapes(ShapeC3), true}, {"unprobed", ListShapes{}, false}} {
			lr := ReadListShaped(pol, utMe(), "r", 2, s.sh)
			if faults := shapeFaults(lr, unprobed, s.c3); len(faults) > 0 {
				t.Errorf("%s, %s: %v\npage %s\ncount %s", tc.name, s.name, faults, pageText(lr), countText(lr))
			}
			// count_follows_probe: the count rendered as the page is caught.
			mut := lr
			mut.CountWhere, mut.CountBranches = lr.Where, lr.Branches
			if slices.Contains(lr.Materialize, true) || s.name == "W0" && slices.ContainsFunc(lr.Drivers, func(d RegionDriver) bool { return d.From != "" }) {
				if len(shapeFaults(mut, unprobed, s.c3)) == 0 {
					t.Errorf("%s, %s: mutant count_follows_probe is not caught", tc.name, s.name)
				}
			}
		}
	}
}

// A C3 leaf negated by a later branch is in the predicate form there: only
// its own term reads it through ARRAY().
func TestC3LeafIsNegatedInThePredicateForm(t *testing.T) {
	t.Parallel()
	self := `{"cmp":{"column":"requester_id","op":"eq","caller":"self"}}`
	lr := ReadListShaped(utBound(t, `{"any":[`+utOf(utUnit(2))+`,`+self+`]}`), utMe(), "r", 2, FixedListShapes(ShapeC3))
	if len(lr.Branches) != 2 || !lr.Materialize[0] || lr.Materialize[1] {
		t.Fatalf("branches %q, materialize %v", lr.Branches, lr.Materialize)
	}
	if !strings.Contains(lr.Branches[0], "ARRAY(") || strings.Contains(lr.Branches[1], "ARRAY(") ||
		!strings.Contains(lr.Branches[1], "AND NOT ((r.org_unit_id IS NOT NULL AND r.org_unit_id IN (SELECT acl_ou.descendant_id") {
		t.Errorf("branches %q", lr.Branches)
	}
}

// A2-5a: the page and the count both W0, no count driver, nothing
// materialized.
func TestRareShapeRendersW0ForPageAndCount(t *testing.T) {
	t.Parallel()
	for _, tc := range shapeCases() {
		lr := ReadListShaped(utBound(t, tc.when), utMe(), "r", 2, ListShapes{rare: true})
		if slices.Contains(lr.Materialize, true) || slices.ContainsFunc(lr.Drivers, func(d RegionDriver) bool { return d.From != "" }) {
			t.Errorf("%s: materialize %v, drivers %+v; want none", tc.name, lr.Materialize, lr.Drivers)
		}
		for _, text := range []string{pageText(lr), countText(lr)} {
			if strings.Contains(text, "ARRAY(") || strings.Contains(text, " IN (SELECT acl_ou") || !strings.Contains(text, "OFFSET 0") {
				t.Errorf("%s: not W0: %s", tc.name, text)
			}
		}
	}
}

// P3 is A2-5's text over the canonical, distinct U: per-unit LATERAL with
// LIMIT K+1, capped at K+1, and no request filter.
func TestProbeStatement(t *testing.T) {
	t.Parallel()
	sql := ProbeStatement(3613)
	for _, want := range []string{
		"SELECT count(*) FROM (SELECT 1 FROM (SELECT acl_u.id FROM (SELECT DISTINCT acl_ud.id FROM unnest($2::uuid[]) acl_ud(id)) acl_u WHERE NOT EXISTS",
		"CROSS JOIN LATERAL (SELECT acl_ou.descendant_id FROM org_unit_closure acl_ou WHERE acl_ou.tenant_id = $1::uuid AND acl_ou.ancestor_id = acl_u.id) acl_d",
		"CROSS JOIN LATERAL (SELECT 1 FROM records r WHERE r.tenant_id = $1::uuid AND r.entity_definition_id = $3::uuid AND r.org_unit_id IS NOT NULL AND r.org_unit_id = acl_d.descendant_id LIMIT 3614) acl_x LIMIT 3614) acl_p",
	} {
		if !strings.Contains(sql, want) {
			t.Errorf("P3 lacks %q:\n%s", want, sql)
		}
	}
	if strings.Count(sql, "$") != strings.Count(sql, "$1")+strings.Count(sql, "$2")+strings.Count(sql, "$3") {
		t.Errorf("P3 binds more than $1..$3: %s", sql)
	}
}

// Acceptance test 9 (metrics) and A2-10: the two descriptors, with exactly
// the shape attribute and no attribute at all.
func TestShapeMetricsCarryOnlyTheShape(t *testing.T) {
	reader := ptReader()
	ctx := context.Background()
	countShape(ctx, ShapeC3)
	countShape(ctx, ShapeWalk)
	countShape(ctx, ShapeRecords)
	observeProbe(ctx, 3*time.Millisecond)
	var rm metricdata.ResourceMetrics
	if err := reader.Collect(ctx, &rm); err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			switch m.Name {
			case UnderListShapeMetric:
				seen[m.Name] = true
				sum, _ := m.Data.(metricdata.Sum[int64])
				var shapes []string
				for _, dp := range sum.DataPoints {
					if keys := attrKeys(dp.Attributes); !slices.Equal(keys, []string{"shape"}) {
						t.Errorf("%s attributes %v, want [shape]", m.Name, keys)
					}
					v, _ := dp.Attributes.Value("shape")
					shapes = append(shapes, v.AsString())
				}
				slices.Sort(shapes)
				if !slices.Equal(shapes, []string{"c3", "records", "walk"}) {
					t.Errorf("%s shapes %v, want [c3 records walk]", m.Name, shapes)
				}
			case UnderProbeMetric:
				seen[m.Name] = true
				if m.Unit != "s" {
					t.Errorf("%s unit %q, want s", m.Name, m.Unit)
				}
				h, _ := m.Data.(metricdata.Histogram[float64])
				for _, dp := range h.DataPoints {
					if keys := attrKeys(dp.Attributes); len(keys) != 0 {
						t.Errorf("%s attributes %v, want none", m.Name, keys)
					}
				}
			}
		}
	}
	if !seen[UnderListShapeMetric] || !seen[UnderProbeMetric] {
		t.Errorf("metrics seen %v", seen)
	}
}

func attrKeys(s attribute.Set) []string {
	var out []string
	for _, kv := range s.ToSlice() {
		out = append(out, string(kv.Key))
	}
	return out
}

// A2-6 requirement 2 and the TICK-1070 condition, at the source: the shape
// code writes no log line and no span, and no error it returns formats a
// value (the probe count, K* or N̂_e are numbers; an error wraps only another
// error or a fixed text). The route-level leak test reads the responses and
// the handler's logs (TestListResponseNamesNoShapeEstimateOrProbe).
func TestShapeCodeLogsNothingAndFormatsNoValue(t *testing.T) {
	t.Parallel()
	src, err := os.ReadFile("shape.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"slog.", "log.", "trace.", "Span", "%d", "%v", "%s", "Sprint", "Itoa(m", "Itoa(k)", "Itoa(n", "FormatInt"} {
		if strings.Contains(string(src), bad) {
			t.Errorf("shape.go uses %q", bad)
		}
	}
	if strings.Contains(ErrProbe.Error(), "count") || strings.ContainsAny(ErrProbe.Error(), "0123456789") {
		t.Errorf("ErrProbe = %q", ErrProbe)
	}
}

// shapeTx is a pgx.Tx whose QueryRow answers the statements ChooseListShapes
// runs: the tenant, N̂_e, and P3 or the folded probe. It records each P3, and
// each folded probe with its arguments.
type shapeTx struct {
	pgx.Tx
	tenant   string
	tenantE  error
	estimate *int64 // nil: no row
	estE     error
	m        int64
	probeE   error
	probes   []string
	// The folded probe (ADR-010-A3): units is |S|, the closure count before
	// the statement's LIMIT, and p3 the CASE's value (nil: NULL).
	units      int64
	p3         *int64
	folded     []string
	foldedArgs [][]any
}

type shapeRow func(dest ...any) error

func (r shapeRow) Scan(dest ...any) error { return r(dest...) }

func (tx *shapeTx) QueryRow(_ context.Context, sql string, args ...any) pgx.Row {
	switch {
	case strings.Contains(sql, "entity_record_estimates"):
		return shapeRow(func(dest ...any) error {
			switch {
			case tx.estE != nil:
				return tx.estE
			case tx.estimate == nil:
				return pgx.ErrNoRows
			}
			*dest[0].(*int64) = *tx.estimate
			return nil
		})
	case strings.HasSuffix(sql, ") acl_z) acl_s"):
		tx.folded = append(tx.folded, sql)
		tx.foldedArgs = append(tx.foldedArgs, args)
		return shapeRow(func(dest ...any) error {
			if tx.probeE != nil {
				return tx.probeE
			}
			*dest[0].(*int64) = min(tx.units, args[4].(int64)) // LIMIT $5
			*dest[1].(**int64) = tx.p3
			return nil
		})
	case strings.HasSuffix(sql, ") acl_p"):
		tx.probes = append(tx.probes, sql)
		return shapeRow(func(dest ...any) error {
			if tx.probeE != nil {
				return tx.probeE
			}
			*dest[0].(*int64) = tx.m
			return nil
		})
	}
	return shapeRow(func(dest ...any) error {
		if tx.tenantE != nil {
			return tx.tenantE
		}
		*dest[0].(*string) = tx.tenant
		return nil
	})
}

// ChooseListShapes' decisions and failures without a database: the switch
// at m = K* and K* + 1, a missing row (K* = 1,000 and P3 runs), A2-5a (no
// P3), one P3 per distinct leaf, and every error fails closed with a text
// that carries no number.
func TestChooseListShapes(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	def := utUnit(99)
	region := utBound(t, `{"any":[`+utOf(utUnit(2))+`,{"all":[{"cmp":{"column":"state","op":"eq","values":["open"]}},`+utOf(utUnit(2))+`]},{"under":{"of":"caller"}}]}`)
	est := func(n int64) *int64 { return &n }
	c3 := func(t *testing.T, sh ListShapes) []bool {
		t.Helper()
		return ReadListShaped(region, utMe(), "r", 2, sh).Materialize
	}
	for _, tc := range []struct {
		name     string
		estimate *int64
		m        int64
		probes   int
		limit    string
		want     []bool
	}{
		{"missing row, m = K*", nil, KStarMin, 2, "LIMIT 1001", []bool{true, true, true}},
		{"missing row, m = K* + 1", nil, KStarMin + 1, 2, "LIMIT 1001", []bool{false, false, false}},
		{"bench estimate, m = 3,613", est(400_000), 3613, 2, "LIMIT 3614", []bool{true, true, true}},
		{"bench estimate, m = 3,614", est(400_000), 3614, 2, "LIMIT 3614", []bool{false, false, false}},
		{"rare: no probe", est(500), 0, 0, "", []bool{false, false, false}},
	} {
		tx := &shapeTx{tenant: utTenant().String(), estimate: tc.estimate, m: tc.m}
		sh, err := ChooseListShapes(ctx, tx, region, utMe(), def, 51)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if len(tx.probes) != tc.probes {
			t.Errorf("%s: %d P3 statements, want %d (one per distinct leaf)", tc.name, len(tx.probes), tc.probes)
		}
		for _, p := range tx.probes {
			if !strings.Contains(p, tc.limit+") acl_x "+tc.limit+") acl_p") {
				t.Errorf("%s: P3 is not capped at %q: %s", tc.name, tc.limit, p)
			}
		}
		if got := c3(t, sh); !slices.Equal(got, tc.want) {
			t.Errorf("%s: C3 per term %v, want %v", tc.name, got, tc.want)
		}
		if lr := ReadListShaped(region, utMe(), "r", 2, sh); tc.probes == 0 && slices.ContainsFunc(lr.Drivers, func(d RegionDriver) bool { return d.From != "" }) {
			t.Errorf("%s: A2-5a keeps a count driver", tc.name)
		}
	}

	// No region term: nothing is read.
	none := &shapeTx{tenantE: errors.New("must not be read")}
	if _, err := ChooseListShapes(ctx, none, utBound(t, `{"not":{"under":{"of":"caller"}}}`), utMe(), def, 51); err != nil {
		t.Errorf("no region term: %v", err)
	}

	boom := errors.New("connection reset")
	for _, tc := range []struct {
		name  string
		tx    *shapeTx
		probe bool
	}{
		{"tenant unreadable", &shapeTx{tenantE: boom}, false},
		{"leaf bound for another tenant", &shapeTx{tenant: utUnit(77).String()}, true},
		{"estimate unreadable", &shapeTx{tenant: utTenant().String(), estE: boom}, false},
		{"probe fails", &shapeTx{tenant: utTenant().String(), probeE: boom}, true},
	} {
		sh, err := ChooseListShapes(ctx, tc.tx, region, utMe(), def, 51)
		if err == nil || !reflect.DeepEqual(sh, ListShapes{}) {
			t.Errorf("%s: shapes %+v, err %v; want an error and no shapes", tc.name, sh, err)
			continue
		}
		if tc.probe && !errors.Is(err, ErrProbe) {
			t.Errorf("%s: %v does not wrap ErrProbe", tc.name, err)
		}
		if strings.ContainsAny(strings.ReplaceAll(err.Error(), utUnit(77).String(), ""), "0123456789") {
			t.Errorf("%s: the error carries a number: %q", tc.name, err)
		}
	}
}

// The I5 format-2 agreement's CI seed (TestI5Format2Agreement, seed
// 20261109) found ReadPredicate returning the list page's W0 text where
// ReadBranches renders the predicate form. On random documents with under
// leaves, ReadPredicate's branches are ReadBranches' and its single
// predicate is CompileRead's, and ReadListShaped's count terms are the same
// text whatever the shape. At least one document's page differs from its
// predicate, so the check would see the regression.
func TestReadPredicateMatchesReadBranchesWithUnderSeed20261109(t *testing.T) {
	t.Parallel()
	r := rand.New(rand.NewPCG(20261109, 0))
	me := utMe()
	differs := 0
	for i := range 300 {
		d := ptRandomDoc(r)
		for j := range d.Policies {
			leaf := Node{Under: &Under{Caller: true}}
			if r.IntN(2) == 0 {
				leaf = Node{Under: &Under{Units: []uuid.UUID{utUnit(1 + r.IntN(2))}}}
			}
			switch r.IntN(4) {
			case 0:
				d.Policies[j].When = Node{Any: []Node{leaf, d.Policies[j].When}}
			case 1:
				d.Policies[j].When = Node{All: []Node{leaf, d.Policies[j].When}}
			case 2:
				d.Policies[j].When = leaf
			}
		}
		pol := BindUnder(&Policy{EntityType: "pt_seed", Document: d}, me, []uuid.UUID{utUnit(1)}, utClosure())
		branches, bargs := ReadBranches(pol, me, "r", 2)
		pb, where, pargs := ReadPredicate(pol, me, "r", 2)
		if len(branches) >= 2 {
			if !slices.Equal(pb, branches) || !reflect.DeepEqual(pargs, bargs) {
				t.Fatalf("doc %d: ReadPredicate's branches differ from ReadBranches'\n%q\n%q", i, pb, branches)
			}
		} else {
			pred, cargs := CompileRead(pol, me, "r", 2)
			if pb != nil || where != pred || !reflect.DeepEqual(pargs, cargs) {
				t.Fatalf("doc %d: ReadPredicate %q, CompileRead %q", i, where, pred)
			}
		}
		for _, sh := range []ListShapes{{}, FixedListShapes(ShapeC3), FixedListShapes(ShapeWalk)} {
			lr := ReadListShaped(pol, me, "r", 2, sh)
			if !slices.Equal(lr.CountBranches, pb) || (pb == nil && lr.CountWhere != where) {
				t.Fatalf("doc %d: the count terms follow the shape", i)
			}
			if pageText(lr) != countText(lr) {
				differs++
			}
		}
	}
	if differs == 0 {
		t.Error("no document's page differs from its predicate: the test cannot see a page text in ReadPredicate")
	}
}

// Every placeholder P3, the folded probe and each leaf text binds carries its
// cast, ::uuid, ::uuid[] or ::bigint: the server's pools run in QueryExecModeExec, where an
// uncast placeholder has no type (42P18 when nothing else types it).
func TestShapeTextsCastEveryPlaceholder(t *testing.T) {
	t.Parallel()
	ph := regexp.MustCompile(`\$[0-9]+(::uuid(\[\])?|::bigint)?`)
	check := func(name, sql string) {
		t.Helper()
		found := ph.FindAllStringSubmatch(sql, -1)
		if len(found) == 0 {
			t.Errorf("%s binds nothing: %s", name, sql)
		}
		for _, m := range found {
			if m[1] == "" {
				t.Errorf("%s: %s has no cast: %s", name, m[0], sql)
			}
		}
	}
	check("P3", ProbeStatement(KStarMin))
	check("folded probe", FoldedProbeStatement(KStarMin))
	for _, tc := range shapeCases() {
		if strings.Contains(tc.when, "cmp") {
			continue // a cmp's state values are text[], cast as such
		}
		for _, sh := range []ListShapes{FixedListShapes(ShapeC3), FixedListShapes(ShapeWalk), FixedListShapes(ShapeRecords), {rare: true}} {
			lr := ReadListShaped(utBound(t, tc.when), utMe(), "r", 2, sh)
			check(tc.name+" page", pageText(lr))
			check(tc.name+" count", countText(lr))
			for _, d := range lr.Drivers {
				if d.From != "" {
					check(tc.name+" driver", d.From)
				}
			}
		}
	}
	if !strings.Contains(pageText(ReadListShaped(utBound(t, `{"under":{"of":"caller"}}`), utMe(), "r", 2, FixedListShapes(ShapeC3))), "= ANY(ARRAY(") {
		t.Error("the C3 case renders no C3 leaf")
	}
}

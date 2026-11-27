package authz

import (
	"context"
	"errors"
	"math"
	"os"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

// Tests for TICK-1104 (ADR-010-A3, shape R) that need no database: the
// constants, the selector's order and bounds, the SQL R and the folded probe
// render, and the metric label. The rows, the plans and the tenant wall are
// integration tests (internal/records/pipeline/acl_under_records_integration_test.go,
// TestUnderListShapesStayInTheTenant).

// recordsLeafSQL is A3-1's leaf, whole, so that a conjunct, a CTE or a driver
// cannot come back unnoticed.
const recordsLeafSQL = `((r.org_unit_id IN (SELECT descendant_id FROM org_unit_closure WHERE tenant_id = $3::uuid AND ancestor_id = ANY($4::uuid[]))) IS TRUE)`

// regionDriven reports whether any term of lr has a count driver.
func regionDriven(lr ListRead) bool {
	return slices.ContainsFunc(lr.Drivers, func(d RegionDriver) bool { return d.From != "" })
}

// Mutant uhash_not_two_ucert (A3-3).
func TestUHashIsTwiceUCert(t *testing.T) {
	t.Parallel()
	if got, want := int64(uHash), 2*int64(uCert); got != want || got != 20222 {
		t.Errorf("uHash = %d, want 2 x uCert = %d = 20222", got, want)
	}
}

// Mutant ucert_differs_from_alert (A3-3): the skip rule and the alert on the
// certified unit count are one number.
func TestUCertIsTheAlertThreshold(t *testing.T) {
	t.Parallel()
	b, err := os.ReadFile("../../deploy/compose/prometheus/rules/groupsync.yml")
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`(?m)^ +- alert: GroupsyncOrgUnitsAboveCertified\n(?: +#.*\n)* +expr: groupsync_org_units_live_max > ([0-9]+)$`).
		FindStringSubmatch(strings.ReplaceAll(string(b), "\r\n", "\n"))
	if m == nil {
		t.Fatal("no GroupsyncOrgUnitsAboveCertified threshold in groupsync.yml")
	}
	if got := strconv.FormatInt(int64(uCert), 10); m[1] != got || got != "10111" {
		t.Errorf("the alert threshold is %s, uCert is %s, want both 10111", m[1], got)
	}
}

// The cap is ⌈N̂_e / 3.3⌉ in integers: for n an integer, n >= cap exactly when
// N̂_e <= 3.3n (A3-2 step 4).
func TestRecordsCap(t *testing.T) {
	t.Parallel()
	for estimate, want := range map[int64]int64{33: 10, 34: 11, 1001: 304, 20_000: 6061, 33_366: 10_111} {
		if got := recordsCap(estimate); got != want {
			t.Errorf("cap of %d = %d, want %d", estimate, got, want)
		}
	}
	for estimate := int64(1); estimate <= 33_366; estimate++ {
		c := recordsCap(estimate)
		if 33*c < 10*estimate || 33*(c-1) >= 10*estimate {
			t.Fatalf("cap of %d = %d is not the least n with 3.3n >= N", estimate, c)
		}
	}
}

// choose runs ChooseListShapes for a policy of one region leaf over a fake
// transaction and renders its read.
func choose(t *testing.T, tx *shapeTx) ListRead {
	t.Helper()
	pol := utBound(t, `{"under":{"of":"caller"}}`)
	sh, err := ChooseListShapes(context.Background(), tx, pol, utMe(), utUnit(99), 51)
	if err != nil {
		t.Fatal(err)
	}
	return ReadListShaped(pol, utMe(), "r", 2, sh)
}

// pageShape names the page shape of a one-leaf read.
func pageShape(lr ListRead) string {
	switch {
	case strings.Contains(lr.Where, ") IS TRUE)"):
		return "R"
	case lr.Materialize[0]:
		return "C3"
	case strings.Contains(lr.Where, "OFFSET 0"):
		return "W0"
	}
	return lr.Where
}

// Acceptance criterion 1 (A3-2): the selector's order, its bounds at both
// ends, and the mutants r_cap_floor (n = cap - 1) and r_no_upper_bound
// (n = U_hash). Below the cap and at U_hash, P3's value decides as in A2.
func TestRecordsSelector(t *testing.T) {
	t.Parallel()
	est := func(n int64) *int64 { return &n }
	for _, tc := range []struct {
		name       string
		estimate   *int64
		units, m   int64
		p3, folded int
		want       string
		driven     bool
	}{
		{"A2-5a wins over R: N = K*", est(KStarMin), 20_000, 0, 0, 0, "W0", false},
		{"one above K* runs the folded probe", est(KStarMin + 1), 20_000, 0, 0, 1, "R", false},
		{"a missing estimate is never R", nil, 20_000, 0, 1, 0, "C3", true},
		{"a missing estimate is never R, m > K*", nil, 20_000, KStarMin + 1, 1, 0, "W0", true},
		{"N = 33,366 runs the folded probe", est(33_366), 10_111, 0, 0, 1, "R", false},
		{"N = 33,367 is above alpha x U_cert: P3, no closure count", est(33_367), 10_112, 0, 1, 0, "C3", true},
		{"ticket at bench scale: P3, no closure count", est(400_000), 10_111, 5000, 1, 0, "W0", true},
		{"the largest estimate does not overflow", est(math.MaxInt64), 10_111, 0, 1, 0, "C3", true},
		{"n = cap - 1, m <= K*", est(20_000), 6060, KStarMin, 0, 1, "C3", true},
		{"n = cap - 1, m > K*", est(20_000), 6060, KStarMin + 1, 0, 1, "W0", true},
		{"n = cap", est(20_000), 6061, 0, 0, 1, "R", false},
		{"n = U_hash - 1", est(20_000), 20_221, 0, 0, 1, "R", false},
		{"n = U_hash, m <= K*", est(20_000), 20_222, KStarMin, 0, 1, "C3", true},
		{"n = U_hash, m > K*", est(20_000), 20_222, KStarMin + 1, 0, 1, "W0", true},
		{"a subtree past U_hash", est(20_000), 1_000_000, KStarMin + 1, 0, 1, "W0", true},
	} {
		// The fake always has a P3 value, so a selector that picks R by a
		// NULL P3 is not mistaken for one that reads n.
		m := tc.m
		tx := &shapeTx{tenant: utTenant().String(), estimate: tc.estimate, units: tc.units, m: tc.m, p3: &m}
		lr := choose(t, tx)
		if len(tx.probes) != tc.p3 || len(tx.folded) != tc.folded {
			t.Errorf("%s: %d P3 and %d folded probes, want %d and %d", tc.name, len(tx.probes), len(tx.folded), tc.p3, tc.folded)
		}
		if got := pageShape(lr); got != tc.want {
			t.Errorf("%s: page shape %s, want %s", tc.name, got, tc.want)
		}
		if got := regionDriven(lr); got != tc.driven {
			t.Errorf("%s: count driven = %v, want %v", tc.name, got, tc.driven)
		}
	}
	// On an R path PostgreSQL answers NULL for P3: it never ran.
	if got := pageShape(choose(t, &shapeTx{tenant: utTenant().String(), estimate: est(20_000), units: 6061})); got != "R" {
		t.Errorf("n = cap with a NULL P3: %s, want R", got)
	}
}

// The folded probe binds the transaction's tenant, the leaf's U, the entity
// definition, ⌈N̂_e/α⌉ and U_hash, over P3 at the request's K*.
func TestFoldedProbeArguments(t *testing.T) {
	t.Parallel()
	n := int64(20_000)
	tx := &shapeTx{tenant: utTenant().String(), estimate: &n, units: 10_000, p3: new(int64)}
	choose(t, tx)
	if len(tx.folded) != 1 || tx.folded[0] != FoldedProbeStatement(KStar(51, n, true)) {
		t.Fatalf("folded probes %q", tx.folded)
	}
	want := []any{utTenant(), []string{utUnit(1).String()}, utUnit(99), int64(6061), int64(20_222)}
	if !reflect.DeepEqual(tx.foldedArgs[0], want) {
		t.Errorf("folded probe arguments %v, want %v", tx.foldedArgs[0], want)
	}
}

// One folded probe per distinct region leaf (A3-2, per leaf), and one list
// can mix R with another shape.
func TestRecordsSelectorPerLeaf(t *testing.T) {
	t.Parallel()
	pol := utBound(t, `{"any":[`+utOf(utUnit(2))+`,{"under":{"of":"caller"}}]}`)
	n := int64(20_000)
	tx := &shapeTx{tenant: utTenant().String(), estimate: &n, units: 7000, p3: new(int64)}
	sh, err := ChooseListShapes(context.Background(), tx, pol, utMe(), utUnit(99), 51)
	if err != nil {
		t.Fatal(err)
	}
	if len(tx.folded) != 2 {
		t.Errorf("%d folded probes, want one per distinct leaf", len(tx.folded))
	}
	for i, b := range ReadListShaped(pol, utMe(), "r", 2, sh).Branches {
		if !strings.Contains(b, ") IS TRUE)") {
			t.Errorf("branch %d is not R: %s", i, b)
		}
	}
	mixed := ListShapes{chosen: map[string]ListShape{
		underKey(utTenant(), []uuid.UUID{utUnit(2)}): ShapeRecords,
		underKey(utTenant(), []uuid.UUID{utUnit(1)}): ShapeC3,
	}}
	lr := ReadListShaped(pol, utMe(), "r", 2, mixed)
	if !slices.Equal(lr.Materialize, []bool{false, true}) || lr.Drivers[0].From != "" || lr.Drivers[1].From == "" ||
		!strings.Contains(lr.Branches[0], ") IS TRUE)") || !strings.Contains(lr.Branches[1], "= ANY(ARRAY(") ||
		strings.Contains(lr.Branches[1], "IS TRUE") || strings.Contains(lr.CountBranches[1], "IS TRUE") {
		t.Errorf("mixed shapes: materialize %v, drivers %+v, branches %q", lr.Materialize, lr.Drivers, lr.Branches)
	}
}

// A NULL P3 value when R is not chosen, and a failed probe, are ErrProbe with
// no shape and no number in the text (A3-2).
func TestFoldedProbeFailsClosed(t *testing.T) {
	t.Parallel()
	n := int64(20_000)
	boom := errors.New("connection reset")
	one := utBound(t, `{"under":{"of":"caller"}}`)
	two := utBound(t, `{"any":[`+utOf(utUnit(2))+`,{"under":{"of":"caller"}}]}`)
	for _, tc := range []struct {
		name string
		pol  *Policy
		tx   *shapeTx
	}{
		{"NULL P3 below the cap", one, &shapeTx{estimate: &n, units: 6060}},
		{"NULL P3 at U_hash", one, &shapeTx{estimate: &n, units: 20_222}},
		{"NULL P3 at U_hash, two leaves", two, &shapeTx{estimate: &n, units: 20_222}},
		{"the folded probe fails", one, &shapeTx{estimate: &n, units: 10_000, probeE: boom}},
		{"the folded probe fails, two leaves", two, &shapeTx{estimate: &n, units: 10_000, probeE: boom}},
	} {
		tc.tx.tenant = utTenant().String()
		sh, err := ChooseListShapes(context.Background(), tc.tx, tc.pol, utMe(), utUnit(99), 51)
		if !errors.Is(err, ErrProbe) || !reflect.DeepEqual(sh, ListShapes{}) {
			t.Errorf("%s: shapes %+v, err %v; want ErrProbe and no shapes", tc.name, sh, err)
			continue
		}
		if strings.ContainsAny(err.Error(), "0123456789") || strings.Contains(err.Error(), "records") {
			t.Errorf("%s: the error names a value: %q", tc.name, err)
		}
		if len(tc.tx.folded) != 1 || len(tc.tx.probes) != 0 {
			t.Errorf("%s: %d folded probes and %d P3 after the failure, want 1 and 0", tc.name, len(tc.tx.folded), len(tc.tx.probes))
		}
	}
}

// A3-1: R's leaf text, for the page and the count, with no driver, no
// MATERIALIZED CTE and no IS NOT NULL conjunct.
func TestRecordsLeafText(t *testing.T) {
	t.Parallel()
	lr := ReadListShaped(utBound(t, `{"under":{"of":"caller"}}`), utMe(), "r", 2, FixedListShapes(ShapeRecords))
	for side, text := range map[string]string{"page": lr.Where, "count": lr.CountWhere} {
		if want := "(" + recordsLeafSQL + ")"; text != want {
			t.Errorf("%s\n%s\nwant\n%s", side, text, want)
		}
	}
	if regionDriven(lr) || slices.Contains(lr.Materialize, true) {
		t.Errorf("drivers %+v, materialize %v; want none", lr.Drivers, lr.Materialize)
	}
	if want := []any{utTenant(), []string{utUnit(1).String()}}; !reflect.DeepEqual(lr.Args, want) {
		t.Errorf("args %v, want %v", lr.Args, want)
	}
}

// Every region term renders R on both sides when R is forced; a term no leaf
// drives (not under) keeps A2's text.
func TestRecordsShapeRendersPageAndCount(t *testing.T) {
	t.Parallel()
	for _, tc := range shapeCases() {
		pol := utBound(t, tc.when)
		a2 := ReadList(pol, utMe(), "r", 2)
		lr := ReadListShaped(pol, utMe(), "r", 2, FixedListShapes(ShapeRecords))
		if !reflect.DeepEqual(lr.Args, a2.Args) {
			t.Errorf("%s: the arguments follow the shape: %v, A2 %v", tc.name, lr.Args, a2.Args)
		}
		if !regionDriven(a2) {
			if pageText(lr) != pageText(a2) || countText(lr) != countText(a2) {
				t.Errorf("%s: no region term, but the text follows the shape\npage %s\ncount %s", tc.name, pageText(lr), countText(lr))
			}
			continue
		}
		if regionDriven(lr) || slices.Contains(lr.Materialize, true) {
			t.Errorf("%s: drivers %+v, materialize %v; want none", tc.name, lr.Drivers, lr.Materialize)
		}
		terms, counts := lr.Branches, lr.CountBranches
		if terms == nil {
			terms, counts = []string{lr.Where}, []string{lr.CountWhere}
		}
		for i := range terms {
			own := a2.Drivers[i].From != ""
			for side, text := range map[string]string{"page": terms[i], "count": counts[i]} {
				if strings.Contains(text, ") IS TRUE)") != own {
					t.Errorf("%s, %s term %d: want R = %v: %s", tc.name, side, i, own, text)
				}
				for _, bad := range []string{"MATERIALIZED", "OFFSET 0", "ARRAY("} {
					if strings.Contains(text, bad) {
						t.Errorf("%s, %s term %d holds %q: %s", tc.name, side, i, bad, text)
					}
				}
			}
		}
	}
}

// A3-1: a later branch negates an R leaf with A2's predicate form, on the
// page and in the count.
func TestRecordsLeafIsNegatedInThePredicateForm(t *testing.T) {
	t.Parallel()
	self := `{"cmp":{"column":"requester_id","op":"eq","caller":"self"}}`
	lr := ReadListShaped(utBound(t, `{"any":[`+utOf(utUnit(2))+`,`+self+`]}`), utMe(), "r", 2, FixedListShapes(ShapeRecords))
	if len(lr.Branches) != 2 || len(lr.CountBranches) != 2 {
		t.Fatalf("branches %q, count branches %q", lr.Branches, lr.CountBranches)
	}
	neg := "AND NOT ((r.org_unit_id IS NOT NULL AND r.org_unit_id IN (SELECT acl_ou.descendant_id"
	for side, b := range map[string][]string{"page": lr.Branches, "count": lr.CountBranches} {
		if !strings.Contains(b[0], ") IS TRUE)") || strings.Contains(b[1], "IS TRUE") || !strings.Contains(b[1], neg) {
			t.Errorf("%s branches %q", side, b)
		}
	}
}

// A3-2's folded statement: the closure count of the canonical U, stopped at
// $uhash, and P3 in the ELSE alone.
func TestFoldedProbeStatement(t *testing.T) {
	t.Parallel()
	a := underArgs{tenant: "$1", units: "$2"}
	want := `SELECT acl_s.n, CASE WHEN acl_s.n >= $4::bigint AND acl_s.n < $5::bigint THEN NULL ELSE (` + ProbeStatement(KStarMin) +
		`) END FROM (SELECT count(*) AS n FROM (SELECT 1 FROM ` + regionFrom(a) + ` LIMIT $5::bigint) acl_z) acl_s`
	if got := FoldedProbeStatement(KStarMin); got != want {
		t.Errorf("folded probe\n%s\nwant\n%s", got, want)
	}
}

// Acceptance criterion 5 item 5 (A3-7 item 2): the R leaf and the folded
// probe's closure count each filter by the bound tenant, $t, in every closure
// read they make.
func TestRecordsTextsCarryTheTenant(t *testing.T) {
	t.Parallel()
	leaf := ReadListShaped(utBound(t, `{"under":{"of":"caller"}}`), utMe(), "r", 2, FixedListShapes(ShapeRecords)).Where
	if strings.Count(leaf, "org_unit_closure") != 1 || !strings.Contains(leaf, "FROM org_unit_closure WHERE tenant_id = $3::uuid AND ") {
		t.Errorf("the R leaf's closure read carries no tenant_id = $t: %s", leaf)
	}
	_, count, ok := strings.Cut(FoldedProbeStatement(KStarMin), " END FROM ")
	if !ok {
		t.Fatalf("no closure count in %q", FoldedProbeStatement(KStarMin))
	}
	reads := regexp.MustCompile(`org_unit_closure (acl_[a-z]+) WHERE (acl_[a-z]+)\.tenant_id = \$1::uuid AND `).FindAllStringSubmatch(count, -1)
	if len(reads) == 0 || len(reads) != strings.Count(count, "FROM org_unit_closure") {
		t.Errorf("%d of %d closure reads of the count carry tenant_id = $t: %s", len(reads), strings.Count(count, "FROM org_unit_closure"), count)
	}
	for _, r := range reads {
		if r[1] != r[2] {
			t.Errorf("a closure read filters another alias's tenant: %s", r[0])
		}
	}
}

// A3-8: the R choice counts as shape="records", the folded probe is observed,
// and neither metric gains a label. Not parallel: the metrics are the
// process's.
func TestRecordsShapeMetrics(t *testing.T) {
	reader := ptReader()
	read := func() (records int64, probes uint64) {
		var rm metricdata.ResourceMetrics
		if err := reader.Collect(context.Background(), &rm); err != nil {
			t.Fatal(err)
		}
		for _, sm := range rm.ScopeMetrics {
			for _, m := range sm.Metrics {
				switch m.Name {
				case UnderListShapeMetric:
					sum, _ := m.Data.(metricdata.Sum[int64])
					for _, dp := range sum.DataPoints {
						if keys := attrKeys(dp.Attributes); !slices.Equal(keys, []string{"shape"}) {
							t.Errorf("%s attributes %v, want [shape]", m.Name, keys)
						}
						if v, _ := dp.Attributes.Value("shape"); v.AsString() == "records" {
							records = dp.Value
						}
					}
				case UnderProbeMetric:
					h, _ := m.Data.(metricdata.Histogram[float64])
					for _, dp := range h.DataPoints {
						if keys := attrKeys(dp.Attributes); len(keys) != 0 {
							t.Errorf("%s attributes %v, want none", m.Name, keys)
						}
						probes = dp.Count
					}
				}
			}
		}
		return records, probes
	}
	r0, p0 := read()
	n := int64(20_000)
	choose(t, &shapeTx{tenant: utTenant().String(), estimate: &n, units: 10_000, p3: new(int64)})
	if r1, p1 := read(); r1 != r0+1 || p1 != p0+1 {
		t.Errorf("shape=records %d -> %d, probes %d -> %d; want one more of each", r0, r1, p0, p1)
	}
}

// The seam the bounds are tested through: foldedProbe binds the cap and
// U_hash it is given and returns what the statement answered.
func TestFoldedProbeSeam(t *testing.T) {
	t.Parallel()
	seven := int64(7)
	tx := &shapeTx{units: 12, p3: &seven}
	n, m, err := foldedProbe(context.Background(), tx, regionLeaf{tenant: utTenant(), units: []uuid.UUID{utUnit(1)}}, utUnit(99), KStarMin, 13, 14)
	if err != nil || n != 12 || m == nil || *m != 7 {
		t.Fatalf("n %d, m %v, err %v; want 12, 7", n, m, err)
	}
	if got := tx.foldedArgs[0][3:]; !reflect.DeepEqual(got, []any{int64(13), int64(14)}) {
		t.Errorf("cap and uhash bound as %v", got)
	}
}

package authz

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/Navaladi2019/TicketingSystem/internal/identity"
)

// The under leaf in the planner and both renderings (TICK-950 P11-17,
// ADR-010 D7, D5a). Set S itself comes from a real tree in
// internal/records/pipeline (TestUnderSetFromARealTreeAgreesInSQLAndGo);
// here it is bound with BindUnder.

func utTenant() uuid.UUID    { return uuid.MustParse("0b000000-0000-4000-8000-0000000000f0") }
func utUnit(n int) uuid.UUID { return uuid.MustParse(fmt.Sprintf("0c000000-0000-4000-8000-%012d", n)) }

func utMe() identity.Principal {
	p := ptMe()
	p.TenantID = utTenant()
	return p
}

// utRows are rows by org unit: in S (unit 1 and its descendant 2), outside
// it (3, a tombstoned unit ExpandUnder left out), and NULL.
func utRows() map[string]Row {
	return map[string]Row{
		"direct unit":     {State: "open", OrgUnitID: ptRef(utUnit(1))},
		"descendant":      {State: "open", OrgUnitID: ptRef(utUnit(2))},
		"outside S":       {State: "open", OrgUnitID: ptRef(utUnit(3))},
		"NULL org unit":   {State: "open"},
		"NULL, requested": {State: "open", RequesterID: ptRef(ptCaller())},
	}
}

// utBound is a one-policy document bound with the caller's S = {1, 2} and
// the listed root 1's S = {1, 2}, root 3's S empty (tombstoned).
func utBound(t *testing.T, when string) *Policy {
	t.Helper()
	pol := ptWhen(t, when)
	return BindUnder(pol, utMe(), []uuid.UUID{utUnit(2), utUnit(1)}, map[uuid.UUID][]uuid.UUID{
		utUnit(1): {utUnit(1), utUnit(2)},
		utUnit(3): {},
	})
}

func utOf(units ...uuid.UUID) string {
	ids := make([]string, len(units))
	for i, u := range units {
		ids[i] = `"` + u.String() + `"`
	}
	return `{"under":{"of":[` + strings.Join(ids, ",") + `]}}`
}

func TestUnderLeafInGo(t *testing.T) {
	t.Parallel()
	caller := `{"under":{"of":"caller"}}`
	for _, tc := range []struct {
		name, when string
		admits     []string
	}{
		{"under caller", caller, []string{"direct unit", "descendant"}},
		{"under a listed unit", utOf(utUnit(1)), []string{"direct unit", "descendant"}},
		{"under a tombstoned unit: empty S", utOf(utUnit(3)), nil},
		{"not under: NULL and outside are outside", `{"not":` + caller + `}`, []string{"outside S", "NULL org unit", "NULL, requested"}},
		{"not under a tombstoned unit: every row", `{"not":` + utOf(utUnit(3)) + `}`, []string{"direct unit", "descendant", "outside S", "NULL org unit", "NULL, requested"}},
		{"under or requester", `{"any":[` + caller + `,{"cmp":{"column":"requester_id","op":"eq","caller":"self"}}]}`, []string{"direct unit", "descendant", "NULL, requested"}},
	} {
		pol := utBound(t, tc.when)
		for name, r := range utRows() {
			_, _, ok := Allows(pol, utMe(), OpRead, r)
			want := false
			for _, a := range tc.admits {
				want = want || a == name
			}
			if ok != want {
				t.Errorf("%s: row %q admitted = %v, want %v", tc.name, name, ok, want)
			}
		}
	}
}

func TestUnderLeafSQL(t *testing.T) {
	t.Parallel()
	pol := utBound(t, `{"not":{"under":{"of":"caller"}}}`)
	sql, args := CompileRead(pol, utMe(), "r", 2)
	want := `((NOT (r.org_unit_id IS NOT NULL AND r.org_unit_id = ANY($3::uuid[]))))`
	if sql != want {
		t.Errorf("sql = %s, want %s", sql, want)
	}
	// S is one array argument, sorted, no join in the row predicate.
	if len(args) != 1 || strings.Join(args[0].([]string), ",") != utUnit(1).String()+","+utUnit(2).String() {
		t.Errorf("args = %v", args)
	}
	if strings.Contains(strings.ToUpper(sql), "SELECT") || strings.Contains(sql, "org_units") {
		t.Errorf("the under leaf joins inside the row predicate: %s", sql)
	}
	// An empty S folds to false: no argument, and not under is true.
	if sql, args := CompileRead(utBound(t, utOf(utUnit(3))), utMe(), "r", 2); sql != "false" || len(args) != 0 {
		t.Errorf("empty S: %s %v, want false", sql, args)
	}
	if sql, _ := CompileRead(utBound(t, `{"not":`+utOf(utUnit(3))+`}`), utMe(), "r", 2); sql != "(true)" {
		t.Errorf("not under an empty S: %s, want (true)", sql)
	}
}

// ReadBranches treats under as a row leaf, like cmp: an any over it splits.
func TestUnderLeafReadBranches(t *testing.T) {
	t.Parallel()
	pol := utBound(t, `{"any":[{"under":{"of":"caller"}},{"cmp":{"column":"requester_id","op":"eq","caller":"self"}}]}`)
	branches, args := ReadBranches(pol, utMe(), "r", 2)
	if len(branches) != 2 || !strings.Contains(branches[0], "r.org_unit_id IS NOT NULL AND r.org_unit_id = ANY($3::uuid[])") ||
		!strings.Contains(branches[1], "AND NOT ((r.org_unit_id IS NOT NULL") {
		t.Fatalf("branches = %q", branches)
	}
	if len(args) != 2 {
		t.Errorf("args = %v", args)
	}
	if b := ReadBranchBound(pol.Document); b != 2 {
		t.Errorf("ReadBranchBound = %d, want 2", b)
	}
}

// Without set S bound for the principal planned, a policy reaching under is
// voided (fails closed, never false under a not) and counted.
func TestUnderLeafUnboundVoidsThePolicy(t *testing.T) {
	t.Parallel()
	when := `{"not":{"under":{"of":"caller"}}}`
	other := utMe()
	other.PrincipalID = ptOther()
	otherTenant := utMe()
	otherTenant.TenantID = uuid.New()
	otherGroups := utMe()
	otherGroups.Groups = []uuid.UUID{ptGA()}
	for name, tc := range map[string]struct {
		pol *Policy
		p   identity.Principal
	}{
		"not bound":                  {ptWhen(t, when), utMe()},
		"bound for another caller":   {utBound(t, when), other},
		"bound for another tenant":   {utBound(t, when), otherTenant},
		"bound for other groups":     {utBound(t, when), otherGroups},
		"a listed root not bound":    {utBound(t, `{"not":`+utOf(utUnit(4))+`}`), utMe()},
		"one of two roots not bound": {utBound(t, `{"not":`+utOf(utUnit(1), utUnit(4))+`}`), utMe()},
	} {
		tc.pol.EntityType = "ut_unbound_" + strings.ReplaceAll(name, " ", "_")
		planned := plan(tc.pol, tc.p, OpRead)
		var pe *PlanError
		if len(planned) != 1 || !errors.As(planned[0].err, &pe) || pe.Rule != RuleUnderUnresolved {
			t.Errorf("%s: plan = %+v, want voided with %s", name, planned, RuleUnderUnresolved)
			continue
		}
		before := ptInvalidCount(t, tc.pol.EntityType)
		for rn, r := range utRows() {
			if _, _, ok := Allows(tc.pol, tc.p, OpRead, r); ok {
				t.Errorf("%s: row %q admitted", name, rn)
			}
		}
		if sql, _ := CompileRead(tc.pol, tc.p, "r", 2); sql != "false" {
			t.Errorf("%s: CompileRead = %s, want false", name, sql)
		}
		if got := ptInvalidCount(t, tc.pol.EntityType) - before; got != int64(len(utRows())+1) {
			t.Errorf("%s: authz_policy_invalid_total rose by %d, want %d", name, got, len(utRows())+1)
		}
	}
}

// A caller with no id has no units: under caller is false without a binding,
// and does not void the policy.
func TestUnderCallerWithNoIDIsFalse(t *testing.T) {
	t.Parallel()
	pol := ptWhen(t, `{"not":{"under":{"of":"caller"}}}`)
	planned := plan(pol, ptNoID(), OpRead)
	if len(planned) != 1 || planned[0].err != nil || planned[0].tree.op != pTrue {
		t.Fatalf("plan = %+v, want not under folded to true", planned)
	}
	if UnderNeeded(pol, ptNoID()) {
		t.Error("UnderNeeded for a caller with no id")
	}
}

// ExpandUnder queries only when an under leaf survives folding, for read or
// write; it collects the caller and every listed root once.
func TestUnderNeededOnlyWhenTheLeafSurvivesFolding(t *testing.T) {
	t.Parallel()
	notMember := `{"member_of":{"groups":["` + ptGC().String() + `"]}}`
	for _, tc := range []struct {
		name, doc string
		need      bool
	}{
		{"no under", `{"format":2,"conditions":[],"policies":[{"name":"p","ops":["read"],"when":{"shared":true}}]}`, false},
		{"under folded away by a false member_of", `{"format":2,"conditions":[],"policies":[{"name":"p","ops":["read"],"when":{"all":[` + notMember + `,{"under":{"of":"caller"}}]}}]}`, false},
		{"under in a voided policy", `{"format":2,"conditions":[],"policies":[{"name":"p","ops":["read"],"when":{"all":[{"ref":"missing"},{"under":{"of":"caller"}}]}}]}`, false},
		{"under in a condition no policy reaches", `{"format":2,"conditions":[{"name":"region","expr":{"under":{"of":"caller"}}}],"policies":[{"name":"p","ops":["read"],"when":{"shared":true}}]}`, false},
		{"under for write", `{"format":2,"conditions":[],"policies":[{"name":"p","ops":["write"],"when":{"under":{"of":"caller"}}}]}`, true},
		{"under through a ref", `{"format":2,"conditions":[{"name":"region","expr":` + utOf(utUnit(1)) + `}],"policies":[{"name":"p","ops":["read"],"when":{"not":{"ref":"region"}}}]}`, true},
	} {
		d, err := Decode([]byte(tc.doc))
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if got := UnderNeeded(&Policy{Document: d}, utMe()); got != tc.need {
			t.Errorf("%s: UnderNeeded = %v, want %v", tc.name, got, tc.need)
		}
	}
	d, _ := Decode([]byte(`{"format":2,"conditions":[],"policies":[{"name":"a","ops":["read"],"when":{"any":[{"under":{"of":"caller"}},` +
		utOf(utUnit(1), utUnit(2)) + `]}},{"name":"b","ops":["write"],"when":` + utOf(utUnit(2), utUnit(3)) + `}]}`))
	n := needs(&Policy{Document: d}, utMe())
	if !n.caller || len(n.roots) != 3 {
		t.Errorf("needs = caller %v roots %v, want caller and three roots", n.caller, n.roots)
	}
}

// ExpandUnder with no surviving under leaf returns the policy as it is and
// never touches the transaction (a nil tx would panic).
func TestExpandUnderRunsNoQueryWithoutUnder(t *testing.T) {
	t.Parallel()
	pol := ptWhen(t, `{"shared":true}`)
	got, err := ExpandUnder(context.Background(), nil, pol, utMe())
	// A copy with every set read as empty (nothing survives folding), the
	// same document; a nil tx would have panicked on a query.
	if err != nil || got == nil || got.Version != pol.Version || len(got.Document.Policies) != len(pol.Document.Policies) {
		t.Errorf("ExpandUnder = %+v, %v; want pol's copy", got, err)
	}
	if got, err := ExpandUnder(context.Background(), nil, nil, utMe()); err != nil || got != nil {
		t.Errorf("ExpandUnder(nil) = %v, %v", got, err)
	}
}

// The cap's error and metric: an error naming no id, counted once.
func TestUnderSetTooLargeIsCounted(t *testing.T) {
	t.Parallel()
	pol := &Policy{EntityType: "ut_too_large", Version: 3}
	before := utTooLargeCount(t, pol.EntityType)
	countUnderTooLarge(pol)
	if got := utTooLargeCount(t, pol.EntityType) - before; got != 1 {
		t.Errorf("authz_under_set_too_large_total rose by %d, want 1", got)
	}
	err := error(&UnderSetTooLargeError{EntityType: pol.EntityType})
	if !strings.Contains(err.Error(), "10000") {
		t.Errorf("error = %v", err)
	}
}

func utTooLargeCount(t *testing.T, entityType string) int64 {
	t.Helper()
	var rm metricdata.ResourceMetrics
	if err := ptReader().Collect(context.Background(), &rm); err != nil {
		t.Fatal(err)
	}
	var n int64
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != UnderSetTooLargeMetric {
				continue
			}
			sum, _ := m.Data.(metricdata.Sum[int64])
			for _, dp := range sum.DataPoints {
				if v, ok := dp.Attributes.Value("entity_type"); ok && v.AsString() == entityType {
					n += dp.Value
				}
			}
		}
	}
	return n
}

// Prepare reports every listed unit as a tenant ref at the pointer the
// database's acl2_id_pointers uses, so the route's NotInTenant refuses the
// same ids as acl2_tenant_ids (foreign_id), in policies and conditions.
func TestPrepareCollectsUnderUnitsAsTenantRefs(t *testing.T) {
	t.Parallel()
	body := `{"rules":{"format":2,"conditions":[{"name":"c","expr":` + utOf(utUnit(5)) + `}],` +
		`"policies":[{"name":"p","ops":["read"],"when":{"any":[{"ref":"c"},` + utOf(utUnit(1), utUnit(2)) + `,{"under":{"of":"caller"}}]}}]}}`
	p, ps := Prepare([]byte(body), DefaultMaxDocumentBytes)
	if len(ps) > 0 {
		t.Fatalf("problems = %v", ps)
	}
	want := map[string]uuid.UUID{
		"/rules/conditions/0/expr/under/of/0":     utUnit(5),
		"/rules/policies/0/when/any/1/under/of/0": utUnit(1),
		"/rules/policies/0/when/any/1/under/of/1": utUnit(2),
	}
	got := 0
	for _, r := range p.Refs {
		if r.Kind != KindOrgUnit {
			continue
		}
		got++
		if want[r.Pointer] != r.ID {
			t.Errorf("ref %s %s not expected", r.Pointer, r.ID)
		}
	}
	if got != len(want) {
		t.Errorf("%d org-unit refs, want %d: %+v", got, len(want), p.Refs)
	}
}

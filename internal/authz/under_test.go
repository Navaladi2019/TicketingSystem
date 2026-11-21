package authz

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/identity"
)

// The under leaf in the planner and both renderings (TICK-950 P11-17,
// TICK-1034, ADR-010 D7 as amended, D5a). The closure itself comes from a
// real tree in internal/records/pipeline
// (TestUnderSetFromARealTreeAgreesInSQLAndGo); here it is bound with
// BindUnder from a model of the tree.

func utTenant() uuid.UUID    { return uuid.MustParse("0b000000-0000-4000-8000-0000000000f0") }
func utUnit(n int) uuid.UUID { return uuid.MustParse(fmt.Sprintf("0c000000-0000-4000-8000-%012d", n)) }

func utMe() identity.Principal {
	p := ptMe()
	p.TenantID = utTenant()
	return p
}

// utRows are rows by org unit: at or below the caller's unit 1 (1 and its
// descendant 2), outside it (3, a tombstoned unit with no closure row), and
// NULL.
func utRows() map[string]Row {
	return map[string]Row{
		"direct unit":     {State: "open", OrgUnitID: ptRef(utUnit(1))},
		"descendant":      {State: "open", OrgUnitID: ptRef(utUnit(2))},
		"outside S":       {State: "open", OrgUnitID: ptRef(utUnit(3))},
		"NULL org unit":   {State: "open"},
		"NULL, requested": {State: "open", RequesterID: ptRef(ptCaller())},
	}
}

// utClosure is the closure of the rows' units: 1 is under 1, 2 under 1 and
// 2, and 3 (tombstoned) under nothing.
func utClosure() map[uuid.UUID][]uuid.UUID {
	return map[uuid.UUID][]uuid.UUID{
		utUnit(1): {utUnit(1)},
		utUnit(2): {utUnit(1), utUnit(2)},
		utUnit(3): {},
	}
}

// utBound is a one-policy document bound with the caller's units U = {1}
// and utClosure.
func utBound(t *testing.T, when string) *Policy {
	t.Helper()
	return BindUnder(ptWhen(t, when), utMe(), []uuid.UUID{utUnit(1)}, utClosure())
}

func utOf(units ...uuid.UUID) string {
	ids := make([]string, len(units))
	for i, u := range units {
		ids[i] = `"` + u.String() + `"`
	}
	return `{"under":{"of":[` + strings.Join(ids, ",") + `]}}`
}

// TICK-950's truth table, now read from the closure: direct unit,
// descendant, tombstoned unit, NULL org_unit_id, not under.
func TestUnderLeafInGo(t *testing.T) {
	t.Parallel()
	caller := `{"under":{"of":"caller"}}`
	for _, tc := range []struct {
		name, when string
		admits     []string
	}{
		{"under caller", caller, []string{"direct unit", "descendant"}},
		{"under a listed unit", utOf(utUnit(1)), []string{"direct unit", "descendant"}},
		{"under the descendant only", utOf(utUnit(2)), []string{"descendant"}},
		{"under a tombstoned unit: no closure row", utOf(utUnit(3)), nil},
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

// The list leaf is C1 on the closure: the row's unit is a closure descendant
// of U, with the tenant and U each one argument, and no tree walk.
func TestUnderLeafSQL(t *testing.T) {
	t.Parallel()
	pol := utBound(t, `{"not":{"under":{"of":"caller"}}}`)
	sql, args := CompileRead(pol, utMe(), "r", 2)
	want := `((NOT (r.org_unit_id IS NOT NULL AND r.org_unit_id IN (SELECT acl_ou.descendant_id FROM org_unit_closure acl_ou` +
		` WHERE acl_ou.tenant_id = $3::uuid AND acl_ou.ancestor_id = ANY($4::uuid[])))))`
	if sql != want {
		t.Errorf("sql = %s, want %s", sql, want)
	}
	if len(args) != 2 || args[0] != utTenant() || strings.Join(args[1].([]string), ",") != utUnit(1).String() {
		t.Errorf("args = %v", args)
	}
	if strings.Contains(sql, "org_units") || strings.Contains(strings.ToUpper(sql), "RECURSIVE") {
		t.Errorf("the under leaf walks the tree: %s", sql)
	}
	// U listed twice, in any order, binds one sorted array.
	_, args = CompileRead(utBound(t, utOf(utUnit(2), utUnit(1), utUnit(2))), utMe(), "r", 0)
	if len(args) != 2 || strings.Join(args[1].([]string), ",") != utUnit(1).String()+","+utUnit(2).String() {
		t.Errorf("listed U args = %v", args)
	}
	// Two leaves with the same U bind it once.
	sql, args = CompileRead(utBound(t, `{"any":[`+utOf(utUnit(1))+`,{"all":[`+utOf(utUnit(1))+
		`,{"cmp":{"column":"state","op":"eq","values":["open"]}}]}]}`), utMe(), "r", 0)
	if len(args) != 3 || strings.Count(sql, "$2::uuid[]") != 2 {
		t.Errorf("one U twice: %s %v", sql, args)
	}
	// A caller with no units has an empty U: false, no argument, and not
	// under is true.
	none := BindUnder(ptWhen(t, `{"under":{"of":"caller"}}`), utMe(), nil, utClosure())
	if sql, args := CompileRead(none, utMe(), "r", 2); sql != "false" || len(args) != 0 {
		t.Errorf("empty U: %s %v, want false", sql, args)
	}
	notNone := BindUnder(ptWhen(t, `{"not":{"under":{"of":"caller"}}}`), utMe(), nil, utClosure())
	if sql, _ := CompileRead(notNone, utMe(), "r", 2); sql != "(true)" {
		t.Errorf("not under an empty U: %s, want (true)", sql)
	}
}

// ReadBranches treats under as a row leaf, like cmp: an any over it splits.
func TestUnderLeafReadBranches(t *testing.T) {
	t.Parallel()
	pol := utBound(t, `{"any":[{"under":{"of":"caller"}},{"cmp":{"column":"requester_id","op":"eq","caller":"self"}}]}`)
	branches, args := ReadBranches(pol, utMe(), "r", 2)
	if len(branches) != 2 || !strings.Contains(branches[0], "r.org_unit_id IS NOT NULL AND r.org_unit_id IN (SELECT acl_ou.descendant_id") ||
		!strings.Contains(branches[1], "AND NOT ((r.org_unit_id IS NOT NULL") {
		t.Fatalf("branches = %q", branches)
	}
	if len(args) != 3 {
		t.Errorf("args = %v", args)
	}
	if b := ReadBranchBound(pol.Document); b != 2 {
		t.Errorf("ReadBranchBound = %d, want 2", b)
	}
}

// ReadList gives a count driver to each region term (the leaf alone, or an
// all holding it) and to no other, reusing the leaf's placeholders.
func TestReadListDrivesRegionTerms(t *testing.T) {
	t.Parallel()
	self := `{"cmp":{"column":"requester_id","op":"eq","caller":"self"}}`
	open := `{"cmp":{"column":"state","op":"eq","values":["open"]}}`
	for _, tc := range []struct {
		name, when string
		branches   int
		driven     []bool
	}{
		{"leaf alone: Where", `{"under":{"of":"caller"}}`, 0, []bool{true}},
		{"leaf and a cmp: Where", `{"all":[{"under":{"of":"caller"}},` + open + `]}`, 0, []bool{true}},
		{"not under: Where, no driver", `{"not":{"under":{"of":"caller"}}}`, 0, []bool{false}},
		{"requester or region", `{"any":[` + self + `,{"all":[` + open + `,{"under":{"of":"caller"}}]}]}`, 2, []bool{false, true}},
		{"region or requester", `{"any":[` + utOf(utUnit(2)) + `,` + self + `]}`, 2, []bool{true, false}},
		{"under inside an any of a branch", `{"any":[` + self + `,{"all":[` + open + `,{"not":{"under":{"of":"caller"}}}]}]}`, 2, []bool{false, false}},
	} {
		lr := ReadList(utBound(t, tc.when), utMe(), "r", 2)
		if len(lr.Branches) != tc.branches || len(lr.Drivers) != len(tc.driven) {
			t.Errorf("%s: %d branches, %d drivers, want %d and %d", tc.name, len(lr.Branches), len(lr.Drivers), tc.branches, len(tc.driven))
			continue
		}
		text := lr.Where + strings.Join(lr.Branches, " ")
		for i, d := range lr.Drivers {
			if (d.From != "") != tc.driven[i] {
				t.Errorf("%s: term %d driven = %v, want %v", tc.name, i, d.From != "", tc.driven[i])
				continue
			}
			if d.From == "" {
				continue
			}
			// The driver's placeholders are the leaf's own: each appears in the
			// term's text, so the count binds no extra argument.
			phs := regexp.MustCompile(`\$[0-9]+`).FindAllString(d.From, -1)
			if len(phs) == 0 {
				t.Errorf("%s: driver binds nothing: %s", tc.name, d.From)
			}
			for _, ph := range phs {
				if !strings.Contains(text, ph+"::") {
					t.Errorf("%s: placeholder %s not in the term: %s", tc.name, ph, d.From)
				}
			}
			if d.Key != "acl_d.descendant_id" || !strings.Contains(d.From, "acl_u(id)") ||
				!strings.Contains(d.From, "acl_up.ancestor_id <> acl_u.id") {
				t.Errorf("%s: driver = %+v", tc.name, d)
			}
		}
	}
}

// Without a binding for the principal planned, a policy reaching under is
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
		"not bound":                {ptWhen(t, when), utMe()},
		"bound for another caller": {utBound(t, when), other},
		"bound for another tenant": {utBound(t, when), otherTenant},
		"bound for other groups":   {utBound(t, when), otherGroups},
		"a listed leaf, not bound": {ptWhen(t, `{"not":`+utOf(utUnit(1))+`}`), utMe()},
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

// Allows on a record whose unit the binding did not read voids every policy
// reaching an under leaf, counted, so `not under` never admits it; a policy
// without one still decides, and a NULL unit needs no read. Explain judges
// the same way.
func TestUnderRowNotReadVoidsThePolicy(t *testing.T) {
	t.Parallel()
	doc := `{"format":2,"conditions":[],"policies":[` +
		`{"name":"outside","ops":["read"],"when":{"not":{"under":{"of":"caller"}}}},` +
		`{"name":"open","ops":["read"],"when":{"cmp":{"column":"state","op":"eq","values":["open"]}}}]}`
	d, err := Decode([]byte(doc))
	if err != nil {
		t.Fatal(err)
	}
	pol := BindUnder(&Policy{Document: d, EntityType: "ut_row_not_read"}, utMe(), []uuid.UUID{utUnit(1)}, utClosure())
	unread := Row{State: "closed", OrgUnitID: ptRef(utUnit(9))}
	before := ptInvalidCount(t, pol.EntityType)
	if _, _, ok := Allows(pol, utMe(), OpRead, unread); ok {
		t.Error("a record whose unit was not read is admitted by not under")
	}
	if got := ptInvalidCount(t, pol.EntityType) - before; got != 1 {
		t.Errorf("authz_policy_invalid_total rose by %d, want 1", got)
	}
	unread.State = "open"
	if name, _, ok := Allows(pol, utMe(), OpRead, unread); !ok || name != "open" {
		t.Errorf("the policy without under = %q %v, want open", name, ok)
	}
	if name, _, ok := Allows(pol, utMe(), OpRead, Row{State: "closed"}); !ok || name != "outside" {
		t.Errorf("a NULL unit = %q %v, want outside", name, ok)
	}
	ex := Explain(pol, utMe(), utMe(), Row{State: "closed", OrgUnitID: ptRef(utUnit(9))})
	if ex.Allow || len(ex.Invalid) != 1 || ex.Invalid[0].Code != RuleUnderUnresolved {
		t.Errorf("Explain = %+v, want deny with outside voided", ex)
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

// ResolveUnder queries only when an under leaf survives folding, for read or
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

// ResolveUnder with no surviving under leaf returns a bound copy and never
// touches the transaction (a nil tx would panic). With one, the tenant comes
// from the transaction, never from the principal.
func TestResolveUnderRunsNoQueryWithoutUnder(t *testing.T) {
	t.Parallel()
	for _, when := range []string{`{"shared":true}`} {
		pol := ptWhen(t, when)
		got, err := ResolveUnder(context.Background(), nil, pol, utMe(), Row{State: "open"})
		if err != nil || got == nil || got.Version != pol.Version || len(got.Document.Policies) != len(pol.Document.Policies) {
			t.Errorf("%s: ResolveUnder = %+v, %v; want pol's copy", when, got, err)
		}
	}
	if got, err := ResolveUnder(context.Background(), nil, nil, utMe()); err != nil || got != nil {
		t.Errorf("ResolveUnder(nil) = %v, %v", got, err)
	}
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

// SP-1d change 5: a policy names at most MaxUnderSets distinct under U,
// refs expanded. Publishing more is a problem at the policy's pointer; the
// caller is one U, and the same ids in another order or repeated are one U.
// The planner voids a stored policy above it, so it fails closed too.
func TestUnderSetsPerPolicyAreCapped(t *testing.T) {
	t.Parallel()
	five := utOf(utUnit(1)) + `,` + utOf(utUnit(2)) + `,` + utOf(utUnit(3)) + `,` + utOf(utUnit(1), utUnit(2)) + `,{"under":{"of":"caller"}}`
	for _, tc := range []struct {
		name, conds, when string
		refused           bool
	}{
		{"four distinct", `[]`, `{"any":[` + utOf(utUnit(1)) + `,` + utOf(utUnit(2)) + `,` + utOf(utUnit(3)) + `,{"under":{"of":"caller"}}]}`, false},
		{"five distinct", `[]`, `{"any":[` + five + `]}`, true},
		{"repeats and reorders count once", `[]`, `{"any":[` + utOf(utUnit(1), utUnit(2)) + `,` + utOf(utUnit(2), utUnit(1)) + `,` + utOf(utUnit(2), utUnit(1), utUnit(2)) +
			`,{"under":{"of":"caller"}},{"not":{"under":{"of":"caller"}}},` + utOf(utUnit(3)) + `,` + utOf(utUnit(4)) + `]}`, false},
		{"five through refs", `[{"name":"a","expr":{"any":[` + utOf(utUnit(1)) + `,` + utOf(utUnit(2)) + `]}},{"name":"b","expr":{"any":[{"ref":"a"},` + utOf(utUnit(3)) + `]}}]`,
			`{"any":[{"ref":"b"},{"ref":"a"},` + utOf(utUnit(4)) + `,` + utOf(utUnit(5)) + `]}`, true},
	} {
		doc := `{"format":2,"conditions":` + tc.conds + `,"policies":[{"name":"ok","ops":["read"],"when":{"shared":true}},{"name":"p","ops":["read"],"when":` + tc.when + `}]}`
		_, ps := Prepare([]byte(`{"rules":`+doc+`}`), DefaultMaxDocumentBytes)
		want := []Problem{{"/rules/policies/1", RuleUnderSetsExceeded}}
		switch {
		case tc.refused && (len(ps) != 1 || ps[0] != want[0]):
			t.Errorf("%s: problems = %v, want %v", tc.name, ps, want)
		case !tc.refused && len(ps) > 0:
			t.Errorf("%s: problems = %v, want none", tc.name, ps)
		}
		d, err := Decode([]byte(doc))
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		pol := BindUnder(&Policy{Document: d, EntityType: "ut_under_sets"}, utMe(), []uuid.UUID{utUnit(1)}, utClosure())
		planned := plan(pol, utMe(), OpRead)
		var pe *PlanError
		voided := len(planned) == 2 && errors.As(planned[1].err, &pe) && pe.Rule == RuleUnderSetsExceeded
		if voided != tc.refused {
			t.Errorf("%s: planner voided = %v, want %v (%+v)", tc.name, voided, tc.refused, planned)
		}
	}
}

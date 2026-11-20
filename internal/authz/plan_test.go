package authz

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/Navaladi2019/TicketingSystem/internal/identity"
)

// Planner tests (TICK-937 P11-05, PLAN-011 C3). The SQL side of the same
// trees runs against PostgreSQL in internal/records/pipeline
// (TestFormat2TreesAgreeInSQLAndGo).

func ptCaller() uuid.UUID { return uuid.MustParse("0b000000-0000-4000-8000-0000000000c1") }
func ptOther() uuid.UUID  { return uuid.MustParse("0b000000-0000-4000-8000-0000000000c2") }
func ptGA() uuid.UUID     { return uuid.MustParse("0b000000-0000-4000-8000-0000000000a1") }
func ptGB() uuid.UUID     { return uuid.MustParse("0b000000-0000-4000-8000-0000000000a2") }
func ptGC() uuid.UUID     { return uuid.MustParse("0b000000-0000-4000-8000-0000000000a3") }

// The principals: the caller in ga and gb, the caller in no group, and a
// principal with no id (a system writer) that still carries ga.
func ptMe() identity.Principal {
	return identity.Principal{PrincipalID: ptCaller(), Groups: []uuid.UUID{ptGA(), ptGB()}}
}
func ptLonely() identity.Principal { return identity.Principal{PrincipalID: ptCaller()} }
func ptNoID() identity.Principal   { return identity.Principal{Groups: []uuid.UUID{ptGA()}} }

func ptRef(id uuid.UUID) *uuid.UUID { return &id }

// ptDoc decodes a format-2 document, failing the test if it does not decode.
func ptDoc(t *testing.T, raw string) *Policy {
	t.Helper()
	d, err := Decode([]byte(raw))
	if err != nil {
		t.Fatalf("decode %s: %v", raw, err)
	}
	return &Policy{Version: 1, EntityType: "pt_" + strings.ToLower(t.Name()), Document: d}
}

// ptWhen is a one-policy read document with the given when and conditions.
func ptWhen(t *testing.T, when string, conds ...string) *Policy {
	t.Helper()
	return ptDoc(t, fmt.Sprintf(`{"format":2,"conditions":[%s],"policies":[{"name":"p","ops":["read"],"when":%s}]}`,
		strings.Join(conds, ","), when))
}

func ptCond(name, expr string) string { return fmt.Sprintf(`{"name":%q,"expr":%s}`, name, expr) }

// The leaves used below, as JSON.
func isRequester() string { return `{"cmp":{"column":"requester_id","op":"eq","caller":"self"}}` }
func inMyGroup() string {
	return `{"cmp":{"column":"assignment_group_id","op":"in","caller":"my_groups"}}`
}
func stateOpen() string { return `{"cmp":{"column":"state","op":"in","values":["open"]}}` }
func memberGA() string  { return fmt.Sprintf(`{"member_of":{"groups":[%q]}}`, ptGA()) }
func memberCaller() string {
	return fmt.Sprintf(`{"member_of":{"users":[%q]}}`, ptCaller())
}

// ptRows are rows covering every NULL column and the caller's own values.
func ptRows() map[string]Row {
	return map[string]Row{
		"open, all NULL":     {State: "open"},
		"closed, all NULL":   {State: "closed"},
		"requested by me":    {State: "open", RequesterID: ptRef(ptCaller())},
		"requested by other": {State: "closed", RequesterID: ptRef(ptOther())},
		"assigned to me":     {State: "open", AssigneeID: ptRef(ptCaller())},
		"in ga":              {State: "closed", AssignmentGroupID: ptRef(ptGA())},
		"in gc":              {State: "open", AssignmentGroupID: ptRef(ptGC())},
		"shared with me":     {State: "closed", SharedWith: []uuid.UUID{ptCaller()}},
		"shared with ga":     {State: "closed", SharedWith: []uuid.UUID{ptGA()}},
	}
}

// Every node kind, folded and unfolded, with NULL columns, a nil caller and
// empty groups: the decision Allows gives on each row.
func TestPlanEveryKind(t *testing.T) {
	t.Parallel()
	all := func(names ...string) []string { return names }
	every := all("open, all NULL", "closed, all NULL", "requested by me", "requested by other", "assigned to me", "in ga", "in gc", "shared with me", "shared with ga")
	cases := []struct {
		name  string
		when  string
		conds []string
		p     identity.Principal
		want  []string // rows admitted
	}{
		{"cmp self", isRequester(), nil, ptMe(), all("requested by me")},
		{"cmp self, no caller id: false", isRequester(), nil, ptNoID(), nil},
		{"cmp my_groups", inMyGroup(), nil, ptMe(), all("in ga")},
		{"cmp my_groups, no groups: false", inMyGroup(), nil, ptLonely(), nil},
		{"cmp values", stateOpen(), nil, ptMe(), all("open, all NULL", "requested by me", "assigned to me", "in gc")},
		{"cmp in no values: false", `{"cmp":{"column":"state","op":"in","values":[]}}`, nil, ptMe(), nil},
		{"cmp literal id", fmt.Sprintf(`{"cmp":{"column":"assignment_group_id","op":"eq","values":[%q]}}`, ptGC()), nil, ptMe(), all("in gc")},
		{"member_of group folds true", memberGA(), nil, ptMe(), every},
		{"member_of group, caller with no id", memberGA(), nil, ptNoID(), every},
		{"member_of group folds false", memberGA(), nil, ptLonely(), nil},
		{"member_of user", memberCaller(), nil, ptLonely(), every},
		{"member_of user, no caller id: false", memberCaller(), nil, ptNoID(), nil},
		{"shared", `{"shared":true}`, nil, ptMe(), all("shared with me", "shared with ga")},
		{"shared, no groups", `{"shared":true}`, nil, ptLonely(), all("shared with me")},
		{"shared, no caller id, with groups: false (D5a)", `{"shared":true}`, nil, ptNoID(), nil},
		{"not over a NULL column is true (D5a)", `{"not":` + isRequester() + `}`, nil, ptMe(), all("open, all NULL", "closed, all NULL", "requested by other", "assigned to me", "in ga", "in gc", "shared with me", "shared with ga")},
		{"not over a folded false is everything", `{"not":` + isRequester() + `}`, nil, ptNoID(), every},
		{"not over a folded true is nothing", `{"not":` + memberGA() + `}`, nil, ptMe(), nil},
		{"all", `{"all":[` + stateOpen() + `,` + `{"not":` + isRequester() + `}]}`, nil, ptMe(), all("open, all NULL", "assigned to me", "in gc")},
		{"all with a folded true child", `{"all":[` + memberGA() + `,` + stateOpen() + `]}`, nil, ptMe(), all("open, all NULL", "requested by me", "assigned to me", "in gc")},
		{"all with a folded false child", `{"all":[` + memberGA() + `,` + stateOpen() + `]}`, nil, ptLonely(), nil},
		{"any", `{"any":[` + isRequester() + `,` + inMyGroup() + `]}`, nil, ptMe(), all("requested by me", "in ga")},
		{"any with a folded true child", `{"any":[` + memberGA() + `,` + isRequester() + `]}`, nil, ptMe(), every},
		{"any with a folded false child", `{"any":[` + memberGA() + `,` + isRequester() + `]}`, nil, ptLonely(), all("requested by me")},
		{"ref", `{"ref":"mine"}`, all(ptCond("mine", isRequester())), ptMe(), all("requested by me")},
		{"ref to a ref", `{"not":{"ref":"outer"}}`, all(ptCond("inner", inMyGroup()), ptCond("outer", `{"any":[{"ref":"inner"},`+isRequester()+`]}`)), ptMe(),
			all("open, all NULL", "closed, all NULL", "requested by other", "assigned to me", "in gc", "shared with me", "shared with ga")},
	}
	rows := ptRows()
	for _, c := range cases {
		pol := ptWhen(t, c.when, c.conds...)
		var got []string
		for name, r := range rows {
			if _, _, ok := Allows(pol, c.p, OpRead, r); ok {
				got = append(got, name)
			}
		}
		slices.Sort(got)
		want := slices.Clone(c.want)
		slices.Sort(want)
		if !slices.Equal(got, want) {
			t.Errorf("%s: admits %v, want %v", c.name, got, want)
		}
		for _, pp := range plan(pol, c.p, OpRead) {
			if pp.err != nil {
				t.Errorf("%s: voided: %v", c.name, pp.err)
			}
		}
	}
}

// An empty any (a stored format-1 rule with who: {}) is false, not invalid:
// it grants nothing and is not counted.
func TestPlanEmptyAnyIsFalse(t *testing.T) {
	t.Parallel()
	pol := &Policy{EntityType: "pt_empty_any", Document: ConvertFormat1([]Rule{{ID: "nobody", Ops: []string{OpRead}}})}
	if pol.Document.Policies[0].When.All[0].Any == nil || len(pol.Document.Policies[0].When.All[0].Any) != 0 {
		t.Fatalf("converted when = %+v, want all[any[]]", pol.Document.Policies[0].When)
	}
	before := ptInvalidCount(t, pol.EntityType)
	if got := plan(pol, ptMe(), OpRead); len(got) != 0 {
		t.Fatalf("plan = %+v, want the policy dropped as false", got)
	}
	if _, _, ok := Allows(pol, ptMe(), OpRead, Row{State: "open"}); ok {
		t.Fatal("an empty any grants")
	}
	if sql, args := CompileRead(pol, ptMe(), "r", 0); sql != "false" || args != nil {
		t.Fatalf("CompileRead = %q %v, want false", sql, args)
	}
	if after := ptInvalidCount(t, pol.EntityType); after != before {
		t.Fatalf("empty any counted as invalid: %d -> %d", before, after)
	}
	// Under not, false becomes true: an empty any is a value, not invalid.
	neg := &Policy{Document: Document{Policies: []NamedPolicy{{Name: "p", Ops: []string{OpRead}, When: Node{Not: &Node{Any: []Node{}}}}}}}
	if _, _, ok := Allows(neg, ptMe(), OpRead, Row{State: "open"}); !ok {
		t.Fatal("not of an empty any does not grant")
	}
}

// A policy with an invalid node anywhere is voided with the right problem,
// grants nothing in either rendering, and never passes through not (D5b).
func TestPlanInvalidVoidsThePolicy(t *testing.T) {
	t.Parallel()
	ok := `{"name":"ok","ops":["read"],"when":` + isRequester() + `}`
	deep := isRequester()
	for range MaxDepth {
		deep = `{"not":` + deep + `}`
	}
	wide := func(n int) string {
		kids := make([]string, n)
		for i := range kids {
			kids[i] = stateOpen()
		}
		return `{"any":[` + strings.Join(kids, ",") + `]}`
	}
	// 9 conditions of 16 leaves each: 144 leaves, over the 128 cap.
	var many []string
	var refs []string
	for i := range 9 {
		many = append(many, ptCond(fmt.Sprintf("c%d", i), wide(16)))
		refs = append(refs, fmt.Sprintf(`{"ref":"c%d"}`, i))
	}
	cases := []struct {
		name, conds, when, rule string
	}{
		{"unresolved ref under not", ``, `{"not":{"ref":"missing"}}`, RuleUnresolvedRef},
		{"self reference", ptCond("a", `{"not":{"ref":"a"}}`), `{"ref":"a"}`, RuleRefCycle},
		{"a -> b -> a", ptCond("a", `{"ref":"b"}`) + "," + ptCond("b", `{"any":[{"ref":"a"},`+isRequester()+`]}`), `{"not":{"ref":"a"}}`, RuleRefCycle},
		{"under with no set S bound", ``, `{"not":{"under":{"of":"caller"}}}`, RuleUnderUnresolved},
		{"depth 9", ``, deep, RuleDepthExceeded},
		{"129+ leaves after expansion", strings.Join(many, ","), `{"any":[` + strings.Join(refs, ",") + `]}`, RulePolicyLeavesExceeded},
		{"invalid beside a deciding constant", ``, `{"any":[` + memberGA() + `,{"ref":"missing"}]}`, RuleUnresolvedRef},
	}
	for _, c := range cases {
		pol := ptDoc(t, `{"format":2,"conditions":[`+c.conds+`],"policies":[{"name":"bad","ops":["read"],"when":`+c.when+`},`+ok+`]}`)
		pol.EntityType = "pt_invalid_" + strings.ReplaceAll(c.name, " ", "_")
		planned := plan(pol, ptMe(), OpRead)
		var pe *PlanError
		if len(planned) != 2 || !errors.As(planned[0].err, &pe) || pe.Rule != c.rule || pe.Policy != "bad" {
			t.Errorf("%s: plan = %+v, want policy bad voided with %s", c.name, planned, c.rule)
			continue
		}
		before := ptInvalidCount(t, pol.EntityType)
		for name, r := range ptRows() {
			_, _, ok := Allows(pol, ptMe(), OpRead, r)
			if want := name == "requested by me"; ok != want {
				t.Errorf("%s: row %q admitted = %v, want %v (only policy ok grants)", c.name, name, ok, want)
			}
		}
		if after := ptInvalidCount(t, pol.EntityType); after != before+int64(len(ptRows())) {
			t.Errorf("%s: authz_policy_invalid_total went %d -> %d over %d evaluations, want one per evaluation", c.name, before, after, len(ptRows()))
		}
		sql, _ := CompileRead(pol, ptMe(), "r", 0)
		if strings.Count(sql, "requester_id") != 2 || strings.Contains(sql, "(NOT ") {
			t.Errorf("%s: CompileRead = %s, want only policy ok", c.name, sql)
		}
		if after := ptInvalidCount(t, pol.EntityType); after != before+int64(len(ptRows()))+1 {
			t.Errorf("%s: CompileRead did not count the voided policy once", c.name)
		}
	}
}

// Hand-built documents the decoder would refuse are voided too: a node with
// no kind, an empty all, an all over MaxChildren, a cmp breaking today's Cond
// rules, and a ref to a condition declared twice.
func TestPlanVoidsUndecodableNodes(t *testing.T) {
	t.Parallel()
	leaf := Node{Shared: true}
	seventeen := make([]Node, MaxChildren+1)
	for i := range seventeen {
		seventeen[i] = leaf
	}
	cases := map[string]struct {
		when  Node
		conds []Condition
		rule  string
	}{
		"no kind":             {Node{}, nil, RuleType},
		"empty all":           {Node{All: []Node{}}, nil, RuleChildCount},
		"17 children":         {Node{Any: seventeen}, nil, RuleChildCount},
		"self on state":       {Node{Not: &Node{Cmp: &Cond{Column: "state", Op: CondEq, Caller: CallerSelf}}}, nil, RuleCondition},
		"eq with 2 values":    {Node{Not: &Node{Cmp: &Cond{Column: "state", Op: CondEq, Values: []string{"a", "b"}}}}, nil, RuleCondition},
		"unknown column":      {Node{Not: &Node{Cmp: &Cond{Column: "data", Op: CondIn, Values: []string{"x"}}}}, nil, RuleCondition},
		"bad uuid":            {Node{Not: &Node{Cmp: &Cond{Column: "assignee_id", Op: CondIn, Values: []string{"x"}}}}, nil, RuleCondition},
		"caller with values":  {Node{Not: &Node{Cmp: &Cond{Column: "assignee_id", Op: CondEq, Caller: CallerSelf, Values: []string{ptCaller().String()}}}}, nil, RuleCondition},
		"duplicate condition": {Node{Ref: "c"}, []Condition{{Name: "c", Expr: leaf}, {Name: "c", Expr: Node{Not: &leaf}}}, RuleUnresolvedRef},
	}
	for name, c := range cases {
		pol := &Policy{Document: Document{Conditions: c.conds, Policies: []NamedPolicy{{Name: "p", Ops: []string{OpRead}, When: c.when}}}}
		planned := plan(pol, ptMe(), OpRead)
		var pe *PlanError
		if len(planned) != 1 || !errors.As(planned[0].err, &pe) || pe.Rule != c.rule {
			t.Errorf("%s: plan = %+v, want voided with %s", name, planned, c.rule)
		}
		if _, _, ok := Allows(pol, ptMe(), OpRead, Row{State: "open", SharedWith: []uuid.UUID{ptCaller()}}); ok {
			t.Errorf("%s: a voided policy grants", name)
		}
		if sql, _ := CompileRead(pol, ptMe(), "r", 0); sql != "false" {
			t.Errorf("%s: CompileRead = %q, want false", name, sql)
		}
	}
}

// A document that shares a condition many times (each condition refers to
// the previous one 16 times) is planned in linear time and voided by the
// leaf cap, not expanded exponentially.
func TestPlanSharedConditionsDoNotExplode(t *testing.T) {
	t.Parallel()
	conds := []string{ptCond("c0", stateOpen())}
	for i := 1; i < 40; i++ {
		kids := make([]string, MaxChildren)
		for k := range kids {
			kids[k] = fmt.Sprintf(`{"ref":"c%d"}`, i-1)
		}
		conds = append(conds, ptCond(fmt.Sprintf("c%d", i), `{"any":[`+strings.Join(kids, ",")+`]}`))
	}
	pol := ptWhen(t, `{"ref":"c39"}`, conds...)
	start := time.Now()
	planned := plan(pol, ptMe(), OpRead)
	if d := time.Since(start); d > time.Second {
		t.Fatalf("planning took %v", d)
	}
	var pe *PlanError
	if len(planned) != 1 || !errors.As(planned[0].err, &pe) || (pe.Rule != RulePolicyLeavesExceeded && pe.Rule != RuleDepthExceeded) {
		t.Fatalf("plan = %+v, want voided by a cap", planned)
	}
}

// Order independence (D1, CI-3): shuffling the policies, every all and any's
// children and the condition declarations changes no decision.
func TestPlanOrderIndependence(t *testing.T) {
	t.Parallel()
	pol := ptDoc(t, `{"format":2,
		"conditions":[`+ptCond("mine", `{"any":[`+isRequester()+`,{"cmp":{"column":"assignee_id","op":"eq","caller":"self"}}]}`)+`,`+
		ptCond("desk", `{"all":[`+inMyGroup()+`,{"not":`+stateOpen()+`}]}`)+`,`+
		ptCond("visible", `{"any":[{"ref":"mine"},{"ref":"desk"},{"shared":true}]}`)+`],
		"policies":[
			{"name":"a","ops":["read"],"when":{"all":[{"ref":"visible"},{"not":{"ref":"desk"}}]}},
			{"name":"b","ops":["read","write"],"when":{"any":[`+memberCaller()+`,{"ref":"desk"}]}},
			{"name":"c","ops":["read"],"when":{"all":[`+stateOpen()+`,{"not":{"ref":"mine"}}]}}]}`)
	principals := []identity.Principal{ptMe(), ptLonely(), ptNoID(), {PrincipalID: ptOther(), Groups: []uuid.UUID{ptGA()}}}
	decide := func(pol *Policy) []bool {
		var out []bool
		for _, p := range principals {
			for _, op := range []string{OpRead, OpWrite} {
				for _, name := range ptSortedRows() {
					_, _, ok := Allows(pol, p, op, ptRows()[name])
					out = append(out, ok)
				}
			}
		}
		return out
	}
	want := decide(pol)
	if !slices.Contains(want, true) || !slices.Contains(want, false) {
		t.Fatalf("the fixture does not discriminate: %v", want)
	}
	r := rand.New(rand.NewPCG(937, 5))
	for i := range 50 {
		sh := shuffleDoc(r, pol.Document)
		if got := decide(&Policy{Document: sh}); !slices.Equal(got, want) {
			t.Fatalf("shuffle %d changed a decision", i)
		}
	}
}

func ptSortedRows() []string {
	out := make([]string, 0, len(ptRows()))
	for k := range ptRows() {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}

func shuffleDoc(r *rand.Rand, d Document) Document {
	out := Document{Format: d.Format}
	for _, c := range d.Conditions {
		out.Conditions = append(out.Conditions, Condition{Name: c.Name, Expr: shuffleNode(r, c.Expr)})
	}
	for _, p := range d.Policies {
		p.When = shuffleNode(r, p.When)
		out.Policies = append(out.Policies, p)
	}
	r.Shuffle(len(out.Conditions), func(i, j int) { out.Conditions[i], out.Conditions[j] = out.Conditions[j], out.Conditions[i] })
	r.Shuffle(len(out.Policies), func(i, j int) { out.Policies[i], out.Policies[j] = out.Policies[j], out.Policies[i] })
	return out
}

func shuffleNode(r *rand.Rand, n Node) Node {
	shuf := func(kids []Node) []Node {
		out := make([]Node, len(kids))
		for i, k := range kids {
			out[i] = shuffleNode(r, k)
		}
		r.Shuffle(len(out), func(i, j int) { out[i], out[j] = out[j], out[i] })
		return out
	}
	switch {
	case n.All != nil:
		n.All = shuf(n.All)
	case n.Any != nil:
		n.Any = shuf(n.Any)
	case n.Not != nil:
		inner := shuffleNode(r, *n.Not)
		n.Not = &inner
	}
	return n
}

// Propagation (CI-5): changing one condition's expr changes every policy
// that refs it.
func TestPlanConditionChangePropagates(t *testing.T) {
	t.Parallel()
	doc := func(expr string) *Policy {
		return ptDoc(t, `{"format":2,"conditions":[`+ptCond("c", expr)+`],"policies":[
			{"name":"p1","ops":["read"],"when":{"ref":"c"}},
			{"name":"p2","ops":["write"],"when":{"all":[{"ref":"c"},`+memberCaller()+`]}}]}`)
	}
	v1, v2 := doc(stateOpen()), doc(isRequester())
	row := ptRows()["open, all NULL"]
	for _, op := range []string{OpRead, OpWrite} {
		if _, _, ok := Allows(v1, ptMe(), op, row); !ok {
			t.Errorf("v1 %s: not granted", op)
		}
		if _, _, ok := Allows(v2, ptMe(), op, row); ok {
			t.Errorf("v2 %s: still granted after the condition changed", op)
		}
	}
}

// Allows reports the first granting policy in declared order and the value
// of each named condition it reaches; the explainer keeps a converted rule's
// stored id as RuleID.
func TestAllowsPolicyAndTrace(t *testing.T) {
	t.Parallel()
	pol := ptDoc(t, `{"format":2,"conditions":[`+ptCond("mine", isRequester())+`,`+ptCond("open", stateOpen())+`,`+
		ptCond("open_mine", `{"all":[{"ref":"mine"},{"ref":"open"}]}`)+`],"policies":[
		{"name":"first","ops":["read"],"when":{"all":[{"ref":"open_mine"},{"not":{"ref":"open"}}]}},
		{"name":"second","ops":["read"],"when":{"any":[{"ref":"open_mine"},{"shared":true}]}},
		{"name":"third","ops":["read"],"when":{"ref":"mine"}}]}`)
	policy, trace, ok := Allows(pol, ptMe(), OpRead, ptRows()["requested by me"])
	want := []TraceStep{{"second", "open_mine", true}, {"second", "mine", true}, {"second", "open", true}}
	if !ok || policy != "second" || !slices.Equal(trace, want) {
		t.Fatalf("Allows = %q %v %v, want second %v", policy, trace, ok, want)
	}
	if policy, trace, ok := Allows(pol, ptMe(), OpRead, ptRows()["closed, all NULL"]); ok || policy != "" || trace != nil {
		t.Fatalf("deny = %q %v %v", policy, trace, ok)
	}

	legacy := &Policy{Version: 3, Document: ConvertFormat1([]Rule{{ID: "Agents-1", Ops: []string{OpRead}, Who: Subject{Requester: true}}})}
	ex := Explain(legacy, ptMe(), ptMe(), ptRows()["requested by me"])
	if !ex.Allow || ex.RuleID != "Agents-1" || ex.Version != 3 {
		t.Fatalf("Explain = %+v, want RuleID Agents-1", ex)
	}
}

// ReadBranches splits on the OR across policies, a top-level any and a
// top-level all with exactly one any child (D6), after folding.
func TestReadBranchesOnTrees(t *testing.T) {
	t.Parallel()
	conv := func(rules ...Rule) *Policy { return &Policy{Document: ConvertFormat1(rules)} }
	rd := []string{OpRead}
	open := Cond{Column: "state", Op: CondIn, Values: []string{"open"}}
	cases := []struct {
		name string
		pol  *Policy
		p    identity.Principal
		want int
	}{
		{"mine: requester, assignee, shared", conv(Rule{ID: "m", Ops: rd, Who: Subject{Requester: true, Assignee: true, Shared: true}, Where: []Cond{open}}), ptMe(), 3},
		{"member_of folds true: one branch with the conditions", conv(Rule{ID: "m", Ops: rd, Who: Subject{Groups: []uuid.UUID{ptGA()}, Requester: true}, Where: []Cond{open}}), ptMe(), 1},
		{"member_of folds false: the other subjects", conv(Rule{ID: "m", Ops: rd, Who: Subject{Groups: []uuid.UUID{ptGC()}, Requester: true, Shared: true}}), ptMe(), 2},
		{"two rules", conv(Rule{ID: "a", Ops: rd, Who: Subject{Requester: true, Assignee: true}}, Rule{ID: "b", Ops: rd, Who: Subject{Shared: true}, Where: []Cond{open}}), ptMe(), 3},
		{"no caller id", conv(Rule{ID: "m", Ops: rd, Who: Subject{Requester: true, Shared: true}}), ptNoID(), 0},
		{"two anys: not distributed", ptWhen(t, `{"all":[{"any":[`+isRequester()+`,{"shared":true}]},{"any":[`+stateOpen()+`,`+inMyGroup()+`]}]}`), ptMe(), 1},
		{"top-level any", ptWhen(t, `{"any":[`+isRequester()+`,{"not":{"shared":true}}]}`), ptMe(), 2},
	}
	for _, c := range cases {
		branches, _ := ReadBranches(c.pol, c.p, "r", 2)
		if len(branches) != c.want {
			t.Errorf("%s: %d branches %v, want %d", c.name, len(branches), branches, c.want)
		}
		for i, b := range branches {
			if strings.Count(b, " AND NOT ") < i {
				t.Errorf("%s: branch %d does not exclude the %d before it: %s", c.name, i, i, b)
			}
		}
	}
	if b, args := ReadBranches(conv(Rule{ID: "e", Ops: rd, Who: Subject{Users: []uuid.UUID{ptCaller()}}}), ptMe(), "r", 2); !slices.Equal(b, []string{"true"}) || args != nil {
		t.Errorf("everyone: %v %v, want the single branch true", b, args)
	}
}

// Every rendered leaf is two-valued: a uuid comparison carries IS NOT NULL,
// and NOT wraps a parenthesized term.
func TestCompileReadTermsAreTwoValued(t *testing.T) {
	t.Parallel()
	pol := ptWhen(t, `{"all":[{"not":`+isRequester()+`},{"not":`+inMyGroup()+`},{"not":{"shared":true}},{"not":`+stateOpen()+`}]}`)
	sql, args := CompileRead(pol, ptMe(), "r", 2)
	for _, want := range []string{
		"(NOT (r.requester_id IS NOT NULL AND r.requester_id = $3::uuid))",
		"(NOT (r.assignment_group_id IS NOT NULL AND r.assignment_group_id = ANY($4::uuid[])))",
		"(NOT EXISTS (SELECT 1 FROM record_shares acl_share",
		"(NOT (r.state = ANY($7::text[])))",
	} {
		if !strings.Contains(sql, want) {
			t.Errorf("CompileRead = %s\nmissing %s", sql, want)
		}
	}
	if len(args) != 5 {
		t.Errorf("args = %v, want 5", args)
	}
}

// ptInvalidCount is authz_policy_invalid_total for an entity type, read from
// a test meter provider registered once for the package.
func ptInvalidCount(t *testing.T, entityType string) int64 {
	t.Helper()
	reader := ptReader()
	var rm metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &rm); err != nil {
		t.Fatal(err)
	}
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != PolicyInvalidMetric {
				continue
			}
			sum, _ := m.Data.(metricdata.Sum[int64])
			for _, dp := range sum.DataPoints {
				if v, ok := dp.Attributes.Value("entity_type"); ok && v.AsString() == entityType {
					return dp.Value
				}
			}
		}
	}
	return 0
}

// ptReader registers the package's metrics on a manual reader, once.
//
//nolint:gochecknoglobals // one meter provider per test binary, as RegisterMetrics is per process
var ptReader = sync.OnceValue(func() *sdkmetric.ManualReader {
	reader := sdkmetric.NewManualReader()
	if err := RegisterMetrics(sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))); err != nil {
		panic(err)
	}
	return reader
})

// The list read path (ReadPredicate) counts a voided policy once per request,
// whether it splits into branches or falls back to CompileRead's predicate
// (D5b). ReadBranches followed by CompileRead counted it twice.
func TestReadPredicateCountsVoidedOnce(t *testing.T) {
	t.Parallel()
	bad := `{"name":"bad","ops":["read"],"when":{"ref":"missing"}}`
	cases := []struct {
		name, policies string
		branches       int
	}{
		{"voided beside a one-branch policy", bad + `,{"name":"ok","ops":["read"],"when":` + isRequester() + `}`, 0},
		{"all voided", bad, 0},
		{"voided beside a split policy", bad + `,{"name":"ok","ops":["read"],"when":{"any":[` + isRequester() + `,` + stateOpen() + `]}}`, 2},
	}
	for _, c := range cases {
		pol := ptDoc(t, `{"format":2,"conditions":[],"policies":[`+c.policies+`]}`)
		pol.EntityType = "pt_readpredicate_" + strings.ReplaceAll(c.name, " ", "_")
		before := ptInvalidCount(t, pol.EntityType)
		branches, where, args := ReadPredicate(pol, ptMe(), "r", 2)
		if after := ptInvalidCount(t, pol.EntityType); after != before+1 {
			t.Errorf("%s: authz_policy_invalid_total went %d -> %d, want +1", c.name, before, after)
		}
		if len(branches) != c.branches {
			t.Errorf("%s: %d branches, want %d: %q", c.name, len(branches), c.branches, branches)
		}
		// The fallback is exactly CompileRead's predicate and arguments.
		if c.branches == 0 {
			sql, sqlArgs := CompileRead(pol, ptMe(), "r", 2)
			if where != sql || fmt.Sprint(args) != fmt.Sprint(sqlArgs) {
				t.Errorf("%s: where %q %v, want CompileRead's %q %v", c.name, where, args, sql, sqlArgs)
			}
		} else {
			br, brArgs := ReadBranches(pol, ptMe(), "r", 2)
			if where != "" || !slices.Equal(branches, br) || fmt.Sprint(args) != fmt.Sprint(brArgs) {
				t.Errorf("%s: branches %q %v where %q, want ReadBranches' %q %v", c.name, branches, args, where, br, brArgs)
			}
		}
	}
}

// The explainer reads the policy name and the trace from the planner: an allow
// traces the granting policy, a deny traces every valid read policy, and a
// voided read policy is listed as invalid and grants nothing (TICK-941).
func TestExplainPolicyTraceAndInvalid(t *testing.T) {
	t.Parallel()
	pol := ptDoc(t, `{"format":2,"conditions":[`+ptCond("mine", isRequester())+`,`+ptCond("open", stateOpen())+`],"policies":[
		{"name":"broken","ops":["read"],"when":{"ref":"nope"}},
		{"name":"write_only","ops":["write"],"when":{"ref":"nope"}},
		{"name":"first","ops":["read"],"when":{"all":[{"ref":"mine"},{"ref":"open"}]}}]}`)
	ex := Explain(pol, ptMe(), ptMe(), ptRows()["requested by me"])
	wantInvalid := []InvalidPolicy{{Policy: "broken", Code: RuleUnresolvedRef, Pointer: "/policies/0"}}
	if !ex.Allow || ex.Policy != "first" || ex.RuleID != "first" || !slices.Equal(ex.Invalid, wantInvalid) {
		t.Fatalf("allow: %+v", ex)
	}
	if want := []TraceStep{{"first", "mine", true}, {"first", "open", true}}; !slices.Equal(ex.Trace, want) {
		t.Fatalf("allow trace = %v", ex.Trace)
	}
	ex = Explain(pol, ptMe(), ptMe(), ptRows()["closed, all NULL"])
	if ex.Allow || ex.Policy != "" || len(ex.Trace) != 2 || ex.Trace[0].Value || !slices.Equal(ex.Invalid, wantInvalid) {
		t.Fatalf("deny: %+v", ex)
	}
	if ex = Explain(nil, ptMe(), ptMe(), Row{}); ex.Invalid == nil || len(ex.Trace) != 0 {
		t.Fatalf("nil policy: %+v", ex)
	}
}

// A converted format-1 policy shows its legacy id beside the policy name.
func TestExplainShowsLegacyIDBesideName(t *testing.T) {
	t.Parallel()
	legacy := &Policy{Document: ConvertFormat1([]Rule{{ID: "Agents-1", Ops: []string{OpRead}, Who: Subject{Requester: true}}})}
	ex := Explain(legacy, ptMe(), ptMe(), ptRows()["requested by me"])
	if ex.Policy == "" || ex.RuleID != "Agents-1" {
		t.Fatalf("Explain = %+v, want a policy name and RuleID Agents-1", ex)
	}
}

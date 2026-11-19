package isolation

// The I5 agreement generator for format-2 access documents (PLAN-011 P11-10,
// TICK-942; ADR-010 I5, D5, D6, D12; RFC-022 §5). It extends the S10 generator
// (i5_gen_test.go) with random trees over every S30 node kind and adds two
// oracles that do not share code with internal/authz:
//
//   - i5Reference evaluates a format-2 document directly on a row, from the
//     ADR's semantics: two-valued leaves (a NULL column, a nil caller id or an
//     empty set is false), plain NOT, and an invalid node, an unresolved or
//     cyclic ref or a cap exceeded voiding the whole policy (D5).
//   - i5S10Allows is the S10 meaning of a format-1 rules array, as main
//     evaluated it before S30 (2277d6b0, authz.Allows over Rule): the old path
//     the converted documents must agree with (D12).
//
// authz.Allows, authz.CompileRead and authz.ReadBranches all render the one
// plan of plan.go, so a planner bug moves both renderings together and the
// SQL-vs-Go agreement alone cannot see it. The reference is what catches it:
// TestI5Format2AllowsMatchesTheReference here (task test) and the format-2
// agreement row of the database half (i5_format2_isolation_test.go).
//
// This file has no build tag, so task test runs the generator, both oracles
// and the planner-level agreement without a database.

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
)

// The format-2 run size: documents × principals, over the S10 world's 160
// records. i5F2TreeDepth is the deepest tree the generator writes (a ref
// counts as its target, so a policy can reach deeper after expansion).
const (
	i5F2Docs      = 300
	i5F2TreeDepth = 4
	// i5F2UnitPrincipals is how many of each document's principals the
	// database-free test checks (task test runs under -race); the database
	// half checks all of them.
	i5F2UnitPrincipals = 12
)

// i5F2Case is one format-2 document and the principals it is checked for.
// Prev is the previous version's conditions: the same names, other exprs,
// for the ref_previous_version mutant.
type i5F2Case struct {
	Policy     *authz.Policy
	Prev       []authz.Condition
	Principals []identity.Principal
}

// i5False is a leaf that is valid and false for every caller and row: an in
// over no value. i5True is its negation.
func i5False() authz.Node {
	return authz.Node{Cmp: &authz.Cond{Column: "state", Op: authz.CondIn, Values: []string{}}}
}

func i5True() authz.Node { f := i5False(); return authz.Node{Not: &f} }

// i5F2Generate is the whole format-2 run for a seed. It draws the S10 world
// from the same seed (so the database half seeds one world for both runs),
// then the documents from a second stream.
func i5F2Generate(seed uint64) (*i5World, []i5F2Case) {
	w, _ := i5Generate(seed)
	r := rand.New(rand.NewPCG(seed^0x5f2, seed^0x2f5e3779b97f4a7c))
	cases := make([]i5F2Case, i5F2Docs)
	for i := range cases {
		cases[i].Policy, cases[i].Prev = w.f2Policy(r)
		for range i5Principals {
			cases[i].Principals = append(cases[i].Principals, w.principal(r))
		}
	}
	return w, cases
}

// f2Policy draws one format-2 policy: 0 to 4 named conditions (refs between
// them, forward refs that can form a cycle, now and then a duplicate name or
// a ref to a name that does not exist), then 1 to 4 policies whose trees ref
// those conditions, so policies share refs.
func (w *i5World) f2Policy(r *rand.Rand) (*authz.Policy, []authz.Condition) {
	if i5Chance(r, 2) {
		return nil, nil
	}
	nConds := r.IntN(5)
	names := make([]string, nConds)
	for i := range names {
		names[i] = "c" + strconv.Itoa(i)
	}
	doc := authz.Document{Format: authz.FormatV2, Conditions: []authz.Condition{}}
	prev := []authz.Condition{}
	for _, name := range names {
		doc.Conditions = append(doc.Conditions, authz.Condition{Name: name, Expr: w.f2Node(r, 1+r.IntN(3), names)})
		prev = append(prev, authz.Condition{Name: name, Expr: w.f2Node(r, 1+r.IntN(3), names)})
	}
	if nConds >= 2 && i5Chance(r, 10) { // a duplicate name: every ref to it is unresolved
		doc.Conditions[nConds-1].Name = doc.Conditions[0].Name
	}
	opSets := [][]string{{authz.OpRead}, {authz.OpRead}, {authz.OpRead}, {authz.OpWrite}, {authz.OpRead, authz.OpWrite}, {}}
	for i := range 1 + r.IntN(4) {
		doc.Policies = append(doc.Policies, authz.NamedPolicy{
			Name: "p" + strconv.Itoa(i),
			Ops:  i5Pick(r, opSets),
			When: w.f2Node(r, 1+r.IntN(i5F2TreeDepth), names),
		})
	}
	return &authz.Policy{EntityDefinitionID: w.Definition, Version: 2 + r.IntN(3), Document: doc}, prev
}

// f2Node draws a tree of at most depth levels (a ref is one level here).
func (w *i5World) f2Node(r *rand.Rand, depth int, names []string) authz.Node {
	if depth > 1 {
		switch n := r.IntN(100); {
		case n < 22:
			return authz.Node{All: w.f2Kids(r, depth, names, 2)}
		case n < 44:
			return authz.Node{Any: w.f2Kids(r, depth, names, 2)}
		case n < 64:
			k := w.f2Node(r, depth-1, names)
			return authz.Node{Not: &k}
		}
	}
	return w.f2Leaf(r, names)
}

// f2Kids is 1 to 3 children, or none at emptyPct percent (an empty all is
// invalid, an empty any is false).
func (w *i5World) f2Kids(r *rand.Rand, depth int, names []string, emptyPct int) []authz.Node {
	if i5Chance(r, emptyPct) {
		return []authz.Node{}
	}
	kids := make([]authz.Node, 1+r.IntN(3))
	for i := range kids {
		kids[i] = w.f2Node(r, depth-1, names)
	}
	return kids
}

func (w *i5World) f2Leaf(r *rand.Rand, names []string) authz.Node {
	switch n := r.IntN(100); {
	case n < 34: // a cmp: every Cond form of S10, malformed ones included
		c := w.cond(r)
		return authz.Node{Cmp: &c}
	case n < 50:
		return authz.Node{MemberOf: w.f2MemberOf(r)}
	case n < 64:
		return authz.Node{Shared: true}
	case n < 92:
		if len(names) == 0 {
			c := w.cond(r)
			return authz.Node{Cmp: &c}
		}
		if i5Chance(r, 4) {
			return authz.Node{Ref: "missing"}
		}
		return authz.Node{Ref: i5Pick(r, names)}
	case n < 95:
		return authz.Node{} // no kind: does not decode, voids the policy
	case n < 97:
		return authz.Node{Under: &authz.Under{Caller: true}} // not available until S31
	}
	return authz.Node{All: []authz.Node{}} // an empty all: invalid
}

// f2MemberOf names some of the tenant's groups and principals, a group the
// tenant does not have, and now and then the nil id or nobody.
func (w *i5World) f2MemberOf(r *rand.Rand) *authz.MemberOf {
	m := &authz.MemberOf{}
	groups := append(w.allGroups(), i5UUID(r))
	users := append(w.allPrincipals(), uuid.Nil)
	if i5Chance(r, 70) {
		for range 1 + r.IntN(2) {
			m.Groups = append(m.Groups, i5Pick(r, groups))
		}
	}
	if i5Chance(r, 50) {
		for range 1 + r.IntN(2) {
			m.Users = append(m.Users, i5Pick(r, users))
		}
	}
	return m
}

// ---- Fixed cases at the caps (SP-2's caps, ADR-010-A1 A1-1) -----------------

// i5CapCase is one fixed document and which of its policies must plan valid.
type i5CapCase struct {
	Name   string
	Policy *authz.Policy
	Valid  map[string]bool // policy name → valid
}

// i5CapCases are the documents at the SP-2 caps and one past each policy cap:
// depth 8 after expansion through a ref chain (valid) and 9 (void); 128
// leaves in one policy through a shared condition reached 8 times (valid) and
// 129 (void); 64 policies holding 128 leaves in all (valid). The document
// caps (64 policies, 128 leaves) are enforced at publish and by the database
// CHECK, not by the planner (TICK-939, TICK-940); one past them is their
// tests' case.
func (w *i5World) i5CapCases() []i5CapCase {
	pol := func(d authz.Document) *authz.Policy {
		d.Format = authz.FormatV2
		if d.Conditions == nil {
			d.Conditions = []authz.Condition{}
		}
		return &authz.Policy{EntityDefinitionID: w.Definition, Version: 9, Document: d}
	}
	read := []string{authz.OpRead}
	cmpIn := func(col string, ids ...uuid.UUID) authz.Node {
		vals := make([]string, len(ids))
		for i, id := range ids {
			vals[i] = id.String()
		}
		return authz.Node{Cmp: &authz.Cond{Column: col, Op: authz.CondIn, Values: vals}}
	}
	ref := func(name string) authz.Node { return authz.Node{Ref: name} }
	not := func(n authz.Node) authz.Node { return authz.Node{Not: &n} }

	// d1 = assignee_id in (three users): NULL on some rows. d_k = not(ref
	// d_{k-1}), so d_k has depth k and is d1 or its negation.
	var chain []authz.Condition
	chain = append(chain, authz.Condition{Name: "d1", Expr: cmpIn("assignee_id", w.Users[0], w.Users[1], w.Users[2])})
	for k := 2; k <= 9; k++ {
		chain = append(chain, authz.Condition{Name: "d" + strconv.Itoa(k), Expr: not(ref("d" + strconv.Itoa(k-1)))})
	}
	depth := pol(authz.Document{Conditions: chain, Policies: []authz.NamedPolicy{
		{Name: "depth8", Ops: read, When: ref("d8")},
		{Name: "depth9", Ops: read, When: ref("d9")},
	}})

	// sixteen: 16 leaves of every kind; reached 8 times it is 128 leaves.
	sixteen := []authz.Node{}
	for i := range 16 {
		switch i % 4 {
		case 0:
			sixteen = append(sixteen, cmpIn("requester_id", w.Users[i%len(w.Users)]))
		case 1:
			sixteen = append(sixteen, authz.Node{MemberOf: &authz.MemberOf{Groups: []uuid.UUID{w.Groups[i%len(w.Groups)]}}})
		case 2:
			sixteen = append(sixteen, authz.Node{Shared: true})
		default:
			sixteen = append(sixteen, cmpIn("assignment_group_id", w.Groups[i%len(w.Groups)]))
		}
	}
	eight := make([]authz.Node, 8)
	for i := range eight {
		eight[i] = ref("sixteen")
	}
	leaves := pol(authz.Document{
		Conditions: []authz.Condition{{Name: "sixteen", Expr: authz.Node{All: []authz.Node{{Any: sixteen}, cmpIn("assignee_id", w.Users[3], w.Users[4])}}}},
		Policies: []authz.NamedPolicy{
			// 8 × (16 + 1) = 136 > 128: void. any over 7 refs = 119 leaves.
			{Name: "leaves136", Ops: read, When: authz.Node{Any: eight}},
			{Name: "leaves119", Ops: read, When: authz.Node{Any: eight[:7]}},
		},
	})
	// Exactly 128 and 129 leaves, with a plain any of 16 reached 8 times.
	leaves128 := pol(authz.Document{
		Conditions: []authz.Condition{{Name: "sixteen", Expr: authz.Node{Any: sixteen}}},
		Policies: []authz.NamedPolicy{
			{Name: "leaves128", Ops: read, When: authz.Node{Any: eight}},
			{Name: "leaves129", Ops: read, When: authz.Node{All: []authz.Node{{Any: eight}, not(cmpIn("requester_id", w.Users[5]))}}},
		},
	})

	// 64 policies of 2 leaves each: 128 leaves in the document.
	var many []authz.NamedPolicy
	principals := w.allPrincipals()
	for i := range authz.MaxPolicies {
		many = append(many, authz.NamedPolicy{Name: "p" + strconv.Itoa(i), Ops: read, When: authz.Node{All: []authz.Node{
			cmpIn("requester_id", principals[i%len(principals)]),
			not(authz.Node{Cmp: &authz.Cond{Column: "state", Op: authz.CondEq, Values: []string{i5States()[i%3]}}}),
		}}})
	}
	manyValid := map[string]bool{}
	for _, np := range many {
		manyValid[np.Name] = true
	}

	return []i5CapCase{
		{Name: "depth 8 after expansion (valid) and 9 (void)", Policy: depth, Valid: map[string]bool{"depth8": true, "depth9": false}},
		{Name: "a shared condition reached 8 times: 136 leaves (void) and 119 (valid)", Policy: leaves, Valid: map[string]bool{"leaves136": false, "leaves119": true}},
		{Name: "128 leaves in a policy (valid) and 129 (void)", Policy: leaves128, Valid: map[string]bool{"leaves128": true, "leaves129": false}},
		{Name: "64 policies, 128 leaves in the document (valid)", Policy: pol(authz.Document{Policies: many}), Valid: manyValid},
	}
}

// ---- Format-1 fixtures (D12) ------------------------------------------------

// i5Format1Fixture is one stored format-1 rules array from authz's testdata,
// its ids remapped onto the world so its subjects and conditions can match.
type i5Format1Fixture struct {
	Name  string
	Rules []authz.Rule
}

// i5Format1Fixtures reads internal/authz/testdata/format1 (TICK-938's
// fixtures, the ones its CI-6 goldens hold) and maps every user id onto the
// world's principals and every group id onto its groups, first seen first.
func (w *i5World) i5Format1Fixtures() ([]i5Format1Fixture, error) {
	paths, err := filepath.Glob(filepath.Join("..", "..", "authz", "testdata", "format1", "*.json"))
	if err != nil {
		return nil, err
	}
	var out []i5Format1Fixture
	for _, path := range paths {
		if strings.HasSuffix(path, ".golden.json") {
			continue
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		var rules []authz.Rule
		if err := json.Unmarshal(raw, &rules); err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		out = append(out, i5Format1Fixture{Name: filepath.Base(path), Rules: w.remap(rules)})
	}
	if len(out) < 6 {
		return nil, fmt.Errorf("found %d format-1 fixtures, want at least 6", len(out))
	}
	return out, nil
}

func (w *i5World) remap(rules []authz.Rule) []authz.Rule {
	users, groups := map[uuid.UUID]uuid.UUID{}, map[uuid.UUID]uuid.UUID{}
	to := func(m map[uuid.UUID]uuid.UUID, pool []uuid.UUID, id uuid.UUID) uuid.UUID {
		if id == uuid.Nil {
			return id
		}
		if _, ok := m[id]; !ok {
			m[id] = pool[len(m)%len(pool)]
		}
		return m[id]
	}
	out := make([]authz.Rule, len(rules))
	for i, r := range rules {
		r.Who.Users = slices.Clone(r.Who.Users)
		r.Who.Groups = slices.Clone(r.Who.Groups)
		for j, id := range r.Who.Users {
			r.Who.Users[j] = to(users, w.Users, id)
		}
		for j, id := range r.Who.Groups {
			r.Who.Groups[j] = to(groups, w.Groups, id)
		}
		r.Where = slices.Clone(r.Where)
		for j, c := range r.Where {
			c.Values = slices.Clone(c.Values)
			for k, v := range c.Values {
				id, err := uuid.Parse(v)
				if err != nil {
					continue
				}
				switch c.Column {
				case "requester_id", "assignee_id":
					c.Values[k] = to(users, w.Users, id).String()
				case "assignment_group_id":
					c.Values[k] = to(groups, w.Groups, id).String()
				}
			}
			r.Where[j] = c
		}
		out[i] = r
	}
	return out
}

// ---- Unit tests (task test) -------------------------------------------------

// i5Quiet discards the log for the test: every voided policy the planner
// meets is logged (authz countVoided), thousands of times in a run.
func i5Quiet(t *testing.T) {
	t.Helper()
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.DiscardHandler))
	t.Cleanup(func() { slog.SetDefault(prev) })
}

// TestI5Format2GeneratorReplaysItsSeed: one seed is one run.
func TestI5Format2GeneratorReplaysItsSeed(t *testing.T) {
	_, c1 := i5F2Generate(42)
	_, c2 := i5F2Generate(42)
	if !reflect.DeepEqual(c1, c2) {
		t.Fatal("seed 42 generated two different format-2 runs")
	}
	_, c3 := i5F2Generate(43)
	if reflect.DeepEqual(c1, c3) {
		t.Fatal("seeds 42 and 43 generated the same format-2 run")
	}
	if len(c1) < i5F2Docs || len(c1[0].Principals) < i5Principals {
		t.Fatalf("run size %d × %d", len(c1), len(c1[0].Principals))
	}
}

// i5F2Shapes is what the default run holds, by name.
func i5F2Shapes(w *i5World, cases []i5F2Case) map[string]bool {
	has := map[string]bool{}
	for _, c := range cases {
		for _, p := range c.Principals {
			has["caller with no id"] = has["caller with no id"] || p.PrincipalID == uuid.Nil
			has["caller with no id, in a group"] = has["caller with no id, in a group"] || (p.PrincipalID == uuid.Nil && len(p.Groups) > 0)
			has["caller in no group"] = has["caller in no group"] || len(p.Groups) == 0
		}
		if c.Policy == nil {
			has["nil policy"] = true
			continue
		}
		d := c.Policy.Document
		reach := map[string]int{} // condition → policies that ref it directly
		for _, np := range d.Policies {
			refs := map[string]bool{}
			has["policy tree of depth 4"] = has["policy tree of depth 4"] || i5TreeDepth(np.When) >= 4
			i5Walk(np.When, func(n authz.Node, parent string) {
				has["kind "+n.Kind()] = true
				if n.Kind() == "" {
					has["node with no kind"] = true
				}
				if n.Ref != "" {
					refs[n.Ref] = true
				}
				if n.All != nil && len(n.All) == 0 {
					has["empty all"] = true
				}
				if n.Any != nil && len(n.Any) == 0 {
					has["empty any"] = true
				}
				if parent == authz.NodeNot && n.Cmp != nil && n.Cmp.Caller == "" && n.Cmp.Column != "state" {
					has["not over a uuid cmp (NULL column)"] = true
				}
				if parent == authz.NodeNot && n.MemberOf != nil {
					has["not over member_of"] = true
				}
				if n.Cmp != nil {
					if _, bad := i5RefCmp(*n.Cmp, identity.Principal{}, authz.Row{}); bad {
						has["malformed cmp"] = true
					}
				}
				if n.MemberOf != nil && len(n.MemberOf.Groups) == 0 && len(n.MemberOf.Users) == 0 {
					has["member_of naming nobody"] = true
				}
				if n.Ref == "missing" {
					has["unresolved ref"] = true
				}
			})
			for name := range refs {
				reach[name]++
			}
			for _, row := range w.Records[:20] {
				if _, valid := i5RefPolicy(d, np, identity.Principal{PrincipalID: w.Users[0]}, row.Row); !valid {
					has["voided policy"] = true
				}
			}
		}
		for _, n := range reach {
			has["ref shared between policies"] = has["ref shared between policies"] || n > 1
		}
		for i, cd := range d.Conditions {
			for j := range i {
				has["duplicate condition name"] = has["duplicate condition name"] || d.Conditions[j].Name == cd.Name
			}
			i5Walk(cd.Expr, func(n authz.Node, _ string) {
				if n.Ref == cd.Name {
					has["self-referencing condition (cycle)"] = true
				}
			})
		}
		for _, np := range d.Policies {
			p := identity.Principal{PrincipalID: w.Users[0]}
			for _, rec := range w.Records {
				v, valid := i5RefPolicy(d, np, p, rec.Row)
				pv, pvalid := i5RefPolicy(authz.Document{Conditions: c.Prev, Policies: d.Policies}, np, p, rec.Row)
				if v != pv || valid != pvalid {
					has["previous conditions differ"] = true
					break
				}
			}
		}
	}
	for _, rec := range w.Records {
		has["nil requester"] = has["nil requester"] || rec.Row.RequesterID == nil
		has["nil assignee"] = has["nil assignee"] || rec.Row.AssigneeID == nil
		has["nil group"] = has["nil group"] || rec.Row.AssignmentGroupID == nil
	}
	return has
}

func i5Walk(n authz.Node, f func(n authz.Node, parent string)) {
	var walk func(n authz.Node, parent string)
	walk = func(n authz.Node, parent string) {
		f(n, parent)
		for _, k := range n.All {
			walk(k, authz.NodeAll)
		}
		for _, k := range n.Any {
			walk(k, authz.NodeAny)
		}
		if n.Not != nil {
			walk(*n.Not, authz.NodeNot)
		}
	}
	walk(n, "")
}

// i5TreeDepth is a tree's depth as written, a ref one level.
func i5TreeDepth(n authz.Node) int {
	d := 0
	for _, k := range append(slices.Clone(n.All), n.Any...) {
		d = max(d, i5TreeDepth(k))
	}
	if n.Not != nil {
		d = i5TreeDepth(*n.Not)
	}
	return d + 1
}

// TestI5Format2GeneratorCoversTheRiskyShapes: the default run holds every
// shape the agreement must hold for, and every node kind the registry marks
// SQLSafe and Available (CI-11: authz's generatedKinds lists them on the
// strength of this test), so neither the database row nor a mutant row can
// pass for want of one.
func TestI5Format2GeneratorCoversTheRiskyShapes(t *testing.T) {
	i5Quiet(t)
	w, cases := i5F2Generate(i5DefaultSeed)
	has := i5F2Shapes(w, cases)
	for _, k := range authz.Kinds() {
		if k.SQLSafe && k.Available && !has["kind "+k.Name] {
			t.Errorf("the default seed's format-2 run builds no %q node (registry: SQLSafe, Available)", k.Name)
		}
	}
	for _, want := range []string{"caller with no id", "caller with no id, in a group", "caller in no group", "nil policy",
		"policy tree of depth 4", "node with no kind", "kind under", "empty all", "empty any",
		"not over a uuid cmp (NULL column)", "not over member_of", "malformed cmp", "member_of naming nobody",
		"unresolved ref", "voided policy", "ref shared between policies", "duplicate condition name",
		"self-referencing condition (cycle)", "previous conditions differ", "nil requester", "nil assignee", "nil group"} {
		if !has[want] {
			t.Errorf("the default seed's format-2 run has no %s", want)
		}
	}
}

// i5F2Agree compares authz.Allows with the reference over the first n
// principals of every case. It returns the first disagreement and how many
// cases admit a proper subset of the records.
func i5F2Agree(w *i5World, cases []i5F2Case, n int) (proper, total int, err error) {
	for ci, c := range cases {
		for ki, p := range c.Principals[:n] {
			admitted := 0
			for _, rec := range w.Records {
				row := rec.row(p)
				_, _, got := authz.Allows(c.Policy, p, authz.OpRead, row)
				if want := i5Reference(c.Policy, p, authz.OpRead, row); got != want {
					j, _ := json.Marshal(c.Policy)
					return proper, total, fmt.Errorf("doc #%d principal #%d record %s: Allows %v, reference %v; principal %+v; row %+v; policy %s",
						ci, ki, rec.Row.ID, got, want, p, row, j)
				}
				if got {
					admitted++
				}
			}
			total++
			if admitted > 0 && admitted < len(w.Records) {
				proper++
			}
		}
	}
	return proper, total, nil
}

// TestI5Format2AllowsMatchesTheReference: authz.Allows, the Go rendering of
// the one plan, equals the reference on every row of the default run, and the
// run discriminates. This is the test that sees a planner mutant (a guard in
// plan.go reverted moves Allows and CompileRead together); the database row
// then holds CompileRead and ReadBranches to Allows.
func TestI5Format2AllowsMatchesTheReference(t *testing.T) {
	i5Quiet(t)
	seed, err := i5Seed()
	if err != nil {
		t.Fatal(err)
	}
	w, cases := i5F2Generate(seed)
	proper, total, err := i5F2Agree(w, cases, i5F2UnitPrincipals)
	if err != nil {
		t.Fatalf("seed %d: %v", seed, err)
	}
	if proper < total/5 {
		t.Fatalf("seed %d: only %d of %d cases admit a proper subset of the records", seed, proper, total)
	}
	t.Logf("seed %d: %d of %d cases admit a proper subset", seed, proper, total)
}

// TestI5Format2CapCasesPlanAsTheCapsSay: each fixed cap case's policies are
// valid or void as the SP-2 caps say, by the reference and by Allows (a void
// policy grants no row; each valid one grants some row to some caller and
// not all of them to every caller).
func TestI5Format2CapCasesPlanAsTheCapsSay(t *testing.T) {
	i5Quiet(t)
	w, cases := i5F2Generate(i5DefaultSeed)
	principals := cases[0].Principals[:20]
	for _, cc := range w.i5CapCases() {
		for _, np := range cc.Policy.Document.Policies {
			want, listed := cc.Valid[np.Name]
			if !listed {
				t.Fatalf("%s: policy %s has no expectation", cc.Name, np.Name)
			}
			only := i5WithDocument(cc.Policy, authz.Document{Format: authz.FormatV2, Conditions: cc.Policy.Document.Conditions, Policies: []authz.NamedPolicy{np}})
			granted := 0
			for _, p := range principals {
				for _, rec := range w.Records {
					row := rec.row(p)
					if _, valid := i5RefPolicy(only.Document, np, p, row); valid != want {
						t.Fatalf("%s: policy %s valid = %v by the reference, want %v", cc.Name, np.Name, valid, want)
					}
					_, _, ok := authz.Allows(only, p, authz.OpRead, row)
					if ok != i5Reference(only, p, authz.OpRead, row) {
						t.Fatalf("%s: policy %s, principal %+v, record %s: Allows %v disagrees with the reference", cc.Name, np.Name, p, rec.Row.ID, ok)
					}
					if ok {
						granted++
					}
				}
			}
			if !want && granted > 0 {
				t.Errorf("%s: void policy %s granted %d rows", cc.Name, np.Name, granted)
			}
			if want && (granted == 0 || granted == len(principals)*len(w.Records)) && len(cc.Valid) < 10 {
				t.Errorf("%s: valid policy %s granted %d of %d rows: the case does not discriminate", cc.Name, np.Name, granted, len(principals)*len(w.Records))
			}
		}
	}
}

// TestI5Format1ConvertedMatchesS10 (D12): a format-1 rules array converted
// on read grants exactly what the S10 rules granted, for every format-1
// fixture (all principals) and for the S10 generator's random rule sets (the
// first i5F2UnitPrincipals principals of each).
func TestI5Format1ConvertedMatchesS10(t *testing.T) {
	i5Quiet(t)
	w, cases := i5Generate(i5DefaultSeed)
	fixtures, err := w.i5Format1Fixtures()
	if err != nil {
		t.Fatal(err)
	}
	check := func(what string, rules []authz.Rule, pol *authz.Policy, principals []identity.Principal) {
		t.Helper()
		for ki, p := range principals {
			for _, rec := range w.Records {
				row := rec.row(p)
				_, _, got := authz.Allows(pol, p, authz.OpRead, row)
				if want := i5S10Allows(rules, p, authz.OpRead, row); got != want {
					t.Fatalf("%s principal #%d record %s: converted %v, S10 %v; rules %+v", what, ki, rec.Row.ID, got, want, rules)
				}
			}
		}
	}
	for _, f := range fixtures {
		check(f.Name, f.Rules, &authz.Policy{Version: 1, Document: authz.ConvertFormat1(f.Rules)}, cases[0].Principals)
	}
	for ci, c := range cases {
		check(fmt.Sprintf("random rules #%d", ci), c.Rules, c.Policy, c.Principals[:6])
	}
}

// TestI5Format2MutantsChangeTheSQL: each mutant compiles a different
// predicate from the real compiler for some case, so a mutant row can never
// pass because the mutant was a no-op.
func TestI5Format2MutantsChangeTheSQL(t *testing.T) {
	i5Quiet(t)
	_, cases := i5F2Generate(i5DefaultSeed)
	for _, name := range i5F2MutantNames() {
		mutant, changed := i5F2Mutants()[name], false
		for i := range cases {
			for _, p := range cases[i].Principals {
				want, wantArgs := i5F2Real(&cases[i], p, "r", 2)
				got, gotArgs := mutant(&cases[i], p, "r", 2)
				if got != want || !reflect.DeepEqual(gotArgs, wantArgs) {
					changed = true
					break
				}
			}
			if changed {
				break
			}
		}
		if !changed {
			t.Errorf("mutant %s compiles every case as the real compiler does", name)
		}
	}
}

// TestI5ReferenceKnowsD5: the reference's own rules on hand-built cases, so a
// wrong oracle cannot hide a wrong planner.
func TestI5ReferenceKnowsD5(t *testing.T) {
	i5Quiet(t)
	u, g := uuid.MustParse("0e000000-0000-4000-8000-0000000000e1"), uuid.MustParse("0e000000-0000-4000-8000-0000000000e2")
	nilRow := authz.Row{State: "open"}
	mine := authz.Row{State: "open", AssigneeID: &u, SharedWith: []uuid.UUID{g}}
	assignee := authz.Node{Cmp: &authz.Cond{Column: "assignee_id", Op: authz.CondIn, Values: []string{u.String()}}}
	bad := authz.Node{Cmp: &authz.Cond{Column: "data", Op: authz.CondEq, Values: []string{"x"}}}
	notOf := func(n authz.Node) authz.Node { return authz.Node{Not: &n} }
	doc := func(when authz.Node, conds ...authz.Condition) *authz.Policy {
		return &authz.Policy{Document: authz.Document{Format: 2, Conditions: conds, Policies: []authz.NamedPolicy{{Name: "p", Ops: []string{authz.OpRead}, When: when}}}}
	}
	caller := identity.Principal{PrincipalID: u, Groups: []uuid.UUID{g}}
	noID := identity.Principal{Groups: []uuid.UUID{g}}
	for _, tc := range []struct {
		name string
		pol  *authz.Policy
		p    identity.Principal
		row  authz.Row
		want bool
	}{
		{"not over a NULL column is true", doc(notOf(assignee)), caller, nilRow, true},
		{"not over a matching column is false", doc(notOf(assignee)), caller, mine, false},
		{"an invalid leaf voids, even under not", doc(notOf(bad)), caller, mine, false},
		{"an invalid leaf voids beside a true one", doc(authz.Node{Any: []authz.Node{i5True(), bad}}), caller, mine, false},
		{"an unresolved ref voids", doc(notOf(authz.Node{Ref: "x"})), caller, mine, false},
		{"a cycle voids", doc(notOf(authz.Node{Ref: "a"}), authz.Condition{Name: "a", Expr: notOf(authz.Node{Ref: "a"})}), caller, mine, false},
		{"a ref resolves to its own document", doc(authz.Node{Ref: "a"}, authz.Condition{Name: "a", Expr: assignee}), caller, mine, true},
		{"shared for a caller with no id is false", doc(authz.Node{Shared: true}), noID, mine, false},
		{"shared for the caller's group", doc(authz.Node{Shared: true}), caller, mine, true},
		{"member_of folds for the caller", doc(authz.Node{MemberOf: &authz.MemberOf{Groups: []uuid.UUID{g}}}), noID, nilRow, true},
		{"not over member_of folds the other way", doc(notOf(authz.Node{MemberOf: &authz.MemberOf{Groups: []uuid.UUID{g}}})), noID, nilRow, false},
		{"an empty any is false", doc(notOf(authz.Node{Any: []authz.Node{}})), caller, nilRow, true},
		{"an empty all voids", doc(notOf(authz.Node{All: []authz.Node{}})), caller, nilRow, false},
		{"under is not available", doc(notOf(authz.Node{Under: &authz.Under{Caller: true}})), caller, nilRow, false},
	} {
		if got := i5Reference(tc.pol, tc.p, authz.OpRead, tc.row); got != tc.want {
			t.Errorf("%s: reference %v, want %v", tc.name, got, tc.want)
		}
		if _, _, got := authz.Allows(tc.pol, tc.p, authz.OpRead, tc.row); got != tc.want {
			t.Errorf("%s: Allows %v, want %v", tc.name, got, tc.want)
		}
	}
}

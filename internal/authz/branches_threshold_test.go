package authz

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/Navaladi2019/TicketingSystem/internal/identity"
)

// Tests for TICK-938 P11-06 (ADR-010 D6, PLAN-011 C6): the SP-2 total-branch
// threshold, its metric, the branch split on random trees, and the bound the
// D6 check uses.

// ptCounter is the named counter's value for an entity type.
func ptCounter(t *testing.T, name, entityType string) int64 {
	t.Helper()
	var rm metricdata.ResourceMetrics
	if err := ptReader().Collect(context.Background(), &rm); err != nil {
		t.Fatal(err)
	}
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != name {
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

// stateAny is an any over n state values: n branches for every caller.
func stateAny(n int) string {
	kids := make([]string, n)
	for i := range kids {
		kids[i] = fmt.Sprintf(`{"cmp":{"column":"state","op":"in","values":["s%d"]}}`, i)
	}
	return `{"any":[` + strings.Join(kids, ",") + `]}`
}

// At the threshold the read splits; one branch above it, ReadBranches returns
// nothing and binds nothing, ReadPredicate returns CompileRead's predicate,
// and each call counts one fallback. A voided policy beside it is still
// counted once per call (D5b).
func TestReadBranchesThreshold(t *testing.T) {
	t.Parallel()
	if maxReadBranches != 16 {
		t.Fatalf("maxReadBranches = %d, SP-2 (docs/data/acl-format2-caps.md item 3) measured 16", maxReadBranches)
	}
	bad := `{"name":"bad","ops":["read"],"when":{"ref":"missing"}}`
	doc := func(name string, policies ...string) *Policy {
		pol := ptDoc(t, `{"format":2,"conditions":[],"policies":[`+strings.Join(policies, ",")+`]}`)
		pol.EntityType = "pt_threshold_" + name
		return pol
	}
	at := doc("at", `{"name":"a","ops":["read"],"when":`+stateAny(16)+`}`)
	over := doc("over", `{"name":"a","ops":["read"],"when":`+stateAny(16)+`}`, `{"name":"b","ops":["read"],"when":`+isRequester()+`}`, bad)

	before := ptCounter(t, ReadBranchesFallbackMetric, at.EntityType)
	if br, args := ReadBranches(at, ptMe(), "r", 2); len(br) != 16 || len(args) != 16 {
		t.Fatalf("at the threshold: %d branches, %d args, want 16 and 16", len(br), len(args))
	}
	if br, where, _ := ReadPredicate(at, ptMe(), "r", 2); len(br) != 16 || where != "" {
		t.Fatalf("ReadPredicate at the threshold: %d branches, where %q", len(br), where)
	}
	if got := ptCounter(t, ReadBranchesFallbackMetric, at.EntityType); got != before {
		t.Errorf("at the threshold: fallback counter %d -> %d, want unchanged", before, got)
	}

	fb := ptCounter(t, ReadBranchesFallbackMetric, over.EntityType)
	inv := ptCounter(t, PolicyInvalidMetric, over.EntityType)
	br, args := ReadBranches(over, ptMe(), "r", 2)
	if br != nil || args != nil {
		t.Fatalf("one above the threshold: %q %v, want no branch and no argument", br, args)
	}
	if got := ptCounter(t, ReadBranchesFallbackMetric, over.EntityType); got != fb+1 {
		t.Errorf("ReadBranches: fallback counter %d -> %d, want +1", fb, got)
	}
	br, where, args := ReadPredicate(over, ptMe(), "r", 2)
	sql, sqlArgs := CompileRead(over, ptMe(), "r", 2)
	if br != nil || where != sql || fmt.Sprint(args) != fmt.Sprint(sqlArgs) {
		t.Errorf("ReadPredicate above the threshold: %q %q %v, want CompileRead's %q %v", br, where, args, sql, sqlArgs)
	}
	if got := ptCounter(t, ReadBranchesFallbackMetric, over.EntityType); got != fb+2 {
		t.Errorf("ReadPredicate: fallback counter %d -> %d, want +1 more", fb+1, got)
	}
	// ReadBranches, ReadPredicate and CompileRead: three evaluations, three counts.
	if got := ptCounter(t, PolicyInvalidMetric, over.EntityType); got != inv+3 {
		t.Errorf("voided policy counted %d -> %d, want once per call (+3)", inv, got)
	}
	if b := ReadBranchBound(over.Document); b != 17 {
		t.Errorf("ReadBranchBound = %d, want 17", b)
	}
}

// Random trees (a local generator; TICK-942 P11-10's suite runs the SQL): the
// disjuncts' OR is Allows' decision on every row, the branches are pairwise
// disjoint, each rendered branch is its disjunct ANDed with the negation of
// the ones before (as text), the count matches the rendering, and no caller
// gets more branches than ReadBranchBound.
func TestReadBranchesOnRandomTrees(t *testing.T) {
	t.Parallel()
	r := rand.New(rand.NewPCG(938, 6))
	principals := []identity.Principal{ptMe(), ptLonely(), ptNoID(), {PrincipalID: ptOther(), Groups: []uuid.UUID{ptGC()}}, {}}
	rows := ptRandomRows(r, 20)
	placeholder := regexp.MustCompile(`\$\d+`)
	norm := func(s string) string { return placeholder.ReplaceAllString(s, "$?") }
	split, fellBack := 0, 0
	for i := range 150 {
		d := ptRandomDoc(r)
		pol := &Policy{EntityType: "pt_random", Document: d}
		bound := ReadBranchBound(d)
		for _, p := range principals {
			planned := plan(pol, p, OpRead)
			var trees []pnode
			everything := false
			for _, pp := range planned {
				if pp.err == nil {
					everything = everything || pp.tree.op == pTrue
					trees = append(trees, pp.tree.disjunctTrees()...)
				}
			}
			if !everything && len(trees) > bound {
				raw, _ := json.Marshal(d)
				t.Fatalf("doc %d: %d branches for %+v, above ReadBranchBound %d: %s", i, len(trees), p, bound, raw)
			}
			for _, row := range rows {
				_, _, want := Allows(pol, p, OpRead, row)
				hits := 0
				for j, d := range trees {
					if d.eval(row) && !slices.ContainsFunc(trees[:j], func(e pnode) bool { return e.eval(row) }) {
						hits++
					}
				}
				if everything {
					if !want {
						t.Fatalf("doc %d: a policy folds to true but Allows denies", i)
					}
					continue
				}
				if (hits > 0) != want || hits > 1 {
					raw, _ := json.Marshal(d)
					t.Fatalf("doc %d, %+v, row %+v: %d branches admit, Allows %v: %s", i, p, row, hits, want, raw)
				}
			}
			branches, _ := readBranches(planned, &compiler{alias: "r", base: 2})
			switch {
			case everything:
				if !slices.Equal(branches, []string{"true"}) {
					t.Fatalf("doc %d: %q, want [true]", i, branches)
				}
			case len(trees) > maxReadBranches:
				fellBack++
				if branches != nil {
					t.Fatalf("doc %d: %d branches above the threshold", i, len(branches))
				}
			default:
				split++
				if len(branches) != len(trees) {
					t.Fatalf("doc %d: %d rendered, %d trees", i, len(branches), len(trees))
				}
				fresh := &compiler{alias: "r", base: 2}
				texts := make([]string, len(trees))
				for j, d := range trees {
					texts[j] = norm(d.branchSQL(fresh))
				}
				for j, b := range branches {
					want := texts[j]
					for _, prev := range texts[:j] {
						want += " AND NOT " + prev
					}
					if norm(b) != "("+want+")" {
						t.Fatalf("doc %d branch %d:\n got %s\nwant (%s)", i, j, norm(b), want)
					}
				}
			}
		}
	}
	t.Logf("%d split, %d above the threshold", split, fellBack)
	if split < 100 || fellBack < 5 {
		t.Fatalf("generator too narrow: %d split, %d above the threshold", split, fellBack)
	}
}

// For format 1, ReadBranchBound is the number d6.sql (SP-2 item 4) computes:
// per read rule, the larger of 1 for a groups/users subject and the count of
// requester, assignee and shared, summed. Plus the case folding widens.
func TestReadBranchBound(t *testing.T) {
	t.Parallel()
	inputs, err := filepath.Glob("testdata/format1/*.json")
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, in := range inputs {
		if strings.HasSuffix(in, ".golden.json") {
			continue
		}
		raw, err := os.ReadFile(in)
		if err != nil {
			t.Fatal(err)
		}
		var rules []Rule
		if err := json.Unmarshal(raw, &rules); err != nil {
			t.Fatal(err)
		}
		d6 := 0
		for _, r := range rules {
			if !slices.Contains(r.Ops, OpRead) {
				continue
			}
			member := 0
			if len(r.Who.Groups)+len(r.Who.Users) > 0 {
				member = 1
			}
			subjects := 0
			for _, b := range []bool{r.Who.Requester, r.Who.Assignee, r.Who.Shared} {
				if b {
					subjects++
				}
			}
			d6 += max(member, subjects)
		}
		if got := ReadBranchBound(ConvertFormat1(rules)); got != d6 {
			t.Errorf("%s: ReadBranchBound %d, d6.sql %d", in, got, d6)
		}
		n++
	}
	if n < 6 {
		t.Fatalf("%d fixtures", n)
	}
	// all[any[a,b], any[member_of, c]] is one branch as stored, two for a
	// member (the second any folds away and the first is distributed).
	widens := ptWhen(t, `{"all":[{"any":[`+isRequester()+`,{"shared":true}]},{"any":[`+memberGA()+`,`+stateOpen()+`]}]}`)
	if br, _ := ReadBranches(widens, ptMe(), "r", 2); len(br) != 2 {
		t.Fatalf("member: %d branches, want 2", len(br))
	}
	if b := ReadBranchBound(widens.Document); b != 2 {
		t.Errorf("ReadBranchBound = %d, want 2", b)
	}
	// A voided or write-only policy gives no branch.
	if b := ReadBranchBound(ptDoc(t, `{"format":2,"conditions":[],"policies":[{"name":"bad","ops":["read"],"when":{"ref":"x"}},{"name":"w","ops":["write"],"when":`+stateAny(3)+`}]}`).Document); b != 0 {
		t.Errorf("voided and write-only: %d, want 0", b)
	}
}

// ptRandomDoc is a document of 1 to 5 policies over every available leaf
// kind, refs to earlier conditions included, within the caps.
func ptRandomDoc(r *rand.Rand) Document {
	d := Document{Format: FormatV2}
	for i := range r.IntN(3) {
		d.Conditions = append(d.Conditions, Condition{Name: fmt.Sprintf("c%d", i), Expr: ptRandomNode(r, 2, i)})
	}
	for i := range 1 + r.IntN(5) {
		ops := []string{OpRead}
		if r.IntN(6) == 0 {
			ops = []string{OpWrite}
		}
		when := ptRandomNode(r, 4, len(d.Conditions))
		if r.IntN(3) == 0 {
			// A wide top-level any of row-dependent children (no member_of,
			// which could fold the policy to true), so some documents pass
			// the threshold.
			kids := make([]Node, 8+r.IntN(MaxChildren-7))
			for j := range kids {
				kids[j] = Node{Cmp: &Cond{Column: "state", Op: CondIn, Values: []string{fmt.Sprint("s", r.IntN(4))}}}
				switch r.IntN(4) {
				case 0:
					kids[j] = Node{Shared: true}
				case 1:
					kids[j] = Node{All: []Node{{Cmp: &Cond{Column: "requester_id", Op: CondEq, Caller: CallerSelf}}, kids[j]}}
				}
			}
			when = Node{Any: kids}
		}
		d.Policies = append(d.Policies, NamedPolicy{Name: fmt.Sprintf("p%d", i), Ops: ops, When: when})
	}
	return d
}

// ptRandomNode is a node of at most depth levels that may ref conditions
// c0..c(refs-1).
func ptRandomNode(r *rand.Rand, depth, refs int) Node {
	if depth > 1 {
		switch r.IntN(5) {
		case 0, 1:
			return Node{All: ptRandomKids(r, depth, refs)}
		case 2:
			return Node{Any: ptRandomKids(r, depth, refs)}
		case 3:
			n := ptRandomNode(r, depth-1, refs)
			return Node{Not: &n}
		}
	}
	id := func(u uuid.UUID) []string { return []string{u.String()} }
	leaves := []Node{
		{Cmp: &Cond{Column: "requester_id", Op: CondEq, Caller: CallerSelf}},
		{Cmp: &Cond{Column: "assignee_id", Op: CondEq, Caller: CallerSelf}},
		{Cmp: &Cond{Column: "assignment_group_id", Op: CondIn, Caller: CallerMyGroups}},
		{Cmp: &Cond{Column: "state", Op: CondIn, Values: []string{"open"}}},
		{Cmp: &Cond{Column: "state", Op: CondIn, Values: []string{"closed", "open"}}},
		{Cmp: &Cond{Column: "state", Op: CondIn, Values: []string{}}},
		{Cmp: &Cond{Column: "assignment_group_id", Op: CondEq, Values: id(ptGC())}},
		{Cmp: &Cond{Column: "requester_id", Op: CondIn, Values: id(ptOther())}},
		{MemberOf: &MemberOf{Groups: []uuid.UUID{ptGA()}}},
		{MemberOf: &MemberOf{Users: []uuid.UUID{ptCaller()}}},
		{MemberOf: &MemberOf{Groups: []uuid.UUID{ptGC()}}},
		{Shared: true},
	}
	if refs > 0 && r.IntN(5) == 0 {
		return Node{Ref: fmt.Sprintf("c%d", r.IntN(refs))}
	}
	return leaves[r.IntN(len(leaves))]
}

func ptRandomKids(r *rand.Rand, depth, refs int) []Node {
	kids := make([]Node, 1+r.IntN(6))
	for i := range kids {
		kids[i] = ptRandomNode(r, depth-1, refs)
	}
	return kids
}

// ptRandomRows are the fixed rows plus n random ones over the same ids.
func ptRandomRows(r *rand.Rand, n int) []Row {
	out := make([]Row, 0, len(ptRows())+n)
	for _, name := range ptSortedRows() {
		out = append(out, ptRows()[name])
	}
	ids := []uuid.UUID{ptCaller(), ptOther(), ptGA(), ptGB(), ptGC()}
	pick := func() *uuid.UUID {
		if r.IntN(3) == 0 {
			return nil
		}
		return ptRef(ids[r.IntN(len(ids))])
	}
	for range n {
		row := Row{State: []string{"open", "closed", "new"}[r.IntN(3)], RequesterID: pick(), AssigneeID: pick(), AssignmentGroupID: pick()}
		if s := pick(); s != nil {
			row.SharedWith = []uuid.UUID{*s}
		}
		out = append(out, row)
	}
	return out
}

package authz

import (
	"fmt"
	"slices"
	"strings"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/metadata"
)

// The format-2 planner (ADR-010 D1, D5, D6; PLAN-011 C3; TICK-937 P11-05).
//
// plan turns a policy document into one planned policy per policy that grants
// the op, in declared order. For each it expands every ref, resolves every
// leaf that depends only on the caller, and folds the constants away. What is
// left (a tree of all, any and not over cmp and shared leaves) depends only on
// the row, and has exactly two renderings: pnode.eval (Go, Allows) and
// pnode.sql (SQL, CompileRead and ReadBranches). Both walk the same tree.
//
// Two rules keep the renderings in agreement (D5):
//
//   - Two-valued leaves. Every leaf is true or false, never NULL, on both
//     sides: a cmp carries "col IS NOT NULL AND ...", records.state is NOT
//     NULL, shared is an EXISTS. A NULL column, a missing caller id or an
//     empty resolved set makes the leaf false, so NOT is plain negation on
//     both sides and NOT over a NULL column is true.
//   - Invalid voids the policy. An unresolvable ref, a cycle, a cap exceeded,
//     an unavailable kind or any node that does not decode sets the planned
//     policy's err. Such a policy is skipped by every rendering and counted
//     (authz_policy_invalid_total); it is never turned into a boolean, so it
//     can never pass through a not.

// The caps the planner enforces on a policy after expansion (ADR-010 D4 as
// lowered by ADR-010-A1 A1-1). The publish route and the database CHECK refuse
// a document above them; the planner voids a policy above them, so a document
// that reached storage some other way still fails closed.
//
// Depth counts the nodes on the longest path from the policy's when to a leaf,
// a ref not counted (it is replaced by its condition's expr): a single leaf is
// depth 1, all[any[cmp]] is depth 3. Leaves are counted after expansion,
// before folding, so a condition reached twice counts twice.
const (
	MaxDepth        = 8
	MaxPolicyLeaves = 128
)

// PlanError is why a policy was voided: Rule is one of the C1 problem codes
// (unresolved_ref, ref_cycle, depth_exceeded, policy_leaves_exceeded,
// child_count, not_available) or a shape code (type, condition, empty) for a
// node that does not decode. It carries no value from the document.
type PlanError struct {
	Policy string
	Rule   string
}

func (e *PlanError) Error() string {
	return fmt.Sprintf("authz: policy %q is invalid: %s", e.Policy, e.Rule)
}

// TraceStep is one named condition reached by the granting policy, and its
// value on the row (ADR-010 D10; the explainer's Trace, TICK-941 P11-09).
type TraceStep struct {
	Policy    string
	Condition string
	Value     bool
}

// pop is the operator of a planned node.
type pop uint8

const (
	pFalse pop = iota
	pTrue
	pAll
	pAny
	pNot
	pCmp
	pShared
)

// pnode is a planned node. A folded tree holds pTrue or pFalse only at its
// root: every constant below the root has been folded into its parent.
type pnode struct {
	op   pop
	kids []pnode // pAll, pAny (2 or more), pNot (exactly 1)
	cond plannedCond
	// caller and groups are the caller's id and groups, for pShared. The
	// caller id is never Nil: shared with no caller id folds to false.
	caller uuid.UUID
	groups []uuid.UUID
}

func constant(b bool) pnode {
	if b {
		return pnode{op: pTrue}
	}
	return pnode{op: pFalse}
}

func (n pnode) isConst() bool { return n.op == pTrue || n.op == pFalse }

// plannedCond is a cmp resolved to a non-empty set: the column must be
// non-NULL and one of the values. A uuid column has ids, state has strs.
// scalar marks a single-id comparison with the caller, which renders as an
// equality: caller: self (ADR-010-E1 (b)), and caller: my_groups for a caller
// with exactly one group (ADR-010-E3 E3-1), so the planner can walk
// records_list_group in index order. A literal list keeps = ANY even with one
// value (E3-2 (i)).
type plannedCond struct {
	column string
	ids    []uuid.UUID
	strs   []string
	scalar bool
}

// plannedPolicy is one policy that grants the op, planned for one principal.
// When err is set the policy grants nothing and tree is unset.
type plannedPolicy struct {
	name string
	tree pnode
	err  error
	// named are the conditions the policy reaches, first reach first, each
	// once, with its folded tree: the explainer's trace.
	named []namedTree
}

type namedTree struct {
	name string
	tree pnode
}

// plan returns the planned policies of pol that grant op to p, in declared
// order, voided ones included (with err set). A policy whose tree folds to
// false is dropped: it grants no row. A nil policy or an unknown op plans
// nothing.
func plan(pol *Policy, p identity.Principal, op string) []plannedPolicy {
	if pol == nil || (op != OpRead && op != OpWrite) {
		return nil
	}
	pl := newPlanner(pol.Document, p)
	var out []plannedPolicy
	for _, np := range pol.Document.Policies {
		if !slices.Contains(np.Ops, op) {
			continue
		}
		pp := pl.policy(np)
		if pp.err == nil && pp.tree.op == pFalse {
			continue
		}
		out = append(out, pp)
	}
	return out
}

// planner expands one document for one principal. Conditions are expanded
// once and shared by every policy that reaches them (they depend only on the
// principal), so a document of refs to refs never expands exponentially.
type planner struct {
	p     identity.Principal
	conds map[string]*Node
	dup   map[string]bool // condition names declared twice: a ref to one is unresolved
	memo  map[string]*expansion
	stack []string // conditions being expanded, for cycle detection
}

// expansion is a subtree planned: its folded tree, its depth and leaf count
// before folding (both saturated just above their cap), the conditions it
// reaches, or the problem that voids every policy reaching it.
type expansion struct {
	tree   pnode
	depth  int
	leaves int
	named  []namedTree
	rule   string // a problem code; "" when valid
}

func newPlanner(d Document, p identity.Principal) *planner {
	pl := &planner{p: p, conds: map[string]*Node{}, dup: map[string]bool{}, memo: map[string]*expansion{}}
	for i := range d.Conditions {
		c := &d.Conditions[i]
		if _, seen := pl.conds[c.Name]; seen {
			pl.dup[c.Name] = true
		}
		pl.conds[c.Name] = &c.Expr
	}
	return pl
}

func (pl *planner) policy(np NamedPolicy) plannedPolicy {
	e := pl.node(np.When)
	switch {
	case e.rule != "":
	case e.depth > MaxDepth:
		e.rule = RuleDepthExceeded
	case e.leaves > MaxPolicyLeaves:
		e.rule = RulePolicyLeavesExceeded
	}
	if e.rule != "" {
		return plannedPolicy{name: np.Name, err: &PlanError{Policy: np.Name, Rule: e.rule}}
	}
	return plannedPolicy{name: np.Name, tree: e.tree, named: e.named}
}

func invalid(rule string) *expansion { return &expansion{rule: rule} }

// node plans one node. Every child is planned before anything folds, so an
// invalid node voids the policy even beside a constant that would decide it.
func (pl *planner) node(n Node) *expansion {
	if k, ok := LookupKind(n.Kind()); !ok || !k.SQLSafe {
		return invalid(RuleType)
	} else if !k.Available {
		return invalid(RuleNotAvailable)
	}
	switch n.Kind() {
	case NodeAll, NodeAny:
		kids := n.All
		if n.Kind() == NodeAny {
			kids = n.Any
		}
		return pl.group(n.Kind(), kids)
	case NodeNot:
		e := pl.node(*n.Not)
		if e.rule != "" {
			return e
		}
		out := &expansion{depth: sat(e.depth+1, MaxDepth), leaves: e.leaves, named: e.named}
		if e.tree.isConst() {
			out.tree = constant(e.tree.op == pFalse)
		} else {
			out.tree = pnode{op: pNot, kids: []pnode{e.tree}}
		}
		return out
	case NodeRef:
		return pl.ref(n.Ref)
	case NodeMemberOf:
		m := n.MemberOf
		in := intersects(m.Groups, pl.p.Groups) ||
			(pl.p.PrincipalID != uuid.Nil && slices.Contains(m.Users, pl.p.PrincipalID))
		return &expansion{tree: constant(in), depth: 1, leaves: 1}
	case NodeCmp:
		pc, st := planCond(*n.Cmp, pl.p)
		switch st {
		case condInvalid:
			return invalid(RuleCondition)
		case condEmpty:
			return &expansion{tree: constant(false), depth: 1, leaves: 1}
		}
		return &expansion{tree: pnode{op: pCmp, cond: pc}, depth: 1, leaves: 1}
	case NodeShared:
		if pl.p.PrincipalID == uuid.Nil {
			return &expansion{tree: constant(false), depth: 1, leaves: 1}
		}
		return &expansion{tree: pnode{op: pShared, caller: pl.p.PrincipalID, groups: pl.p.Groups}, depth: 1, leaves: 1}
	}
	return invalid(RuleType) // a SQLSafe, Available kind the planner does not know
}

// group plans an all or an any. An all with no children is invalid (it would
// grant every row); an any with no children is false: a stored format-1 rule
// with who: {} converts to one, and grants nothing, as it did in S10.
func (pl *planner) group(kind string, kids []Node) *expansion {
	if len(kids) > MaxChildren || (kind == NodeAll && len(kids) == 0) {
		return invalid(RuleChildCount)
	}
	out := &expansion{}
	var maxDepth int
	planned := make([]pnode, 0, len(kids))
	for _, k := range kids {
		e := pl.node(k)
		if e.rule != "" {
			return e
		}
		maxDepth = max(maxDepth, e.depth)
		out.leaves = sat(out.leaves+e.leaves, MaxPolicyLeaves)
		out.named = mergeNamed(out.named, e.named)
		planned = append(planned, e.tree)
	}
	out.depth = sat(maxDepth+1, MaxDepth)
	if out.depth > MaxDepth {
		return invalid(RuleDepthExceeded)
	}
	if out.leaves > MaxPolicyLeaves {
		return invalid(RulePolicyLeavesExceeded)
	}
	// all: false wins, true is dropped; any: true wins, false is dropped.
	op, absorb := pAll, pFalse
	if kind == NodeAny {
		op, absorb = pAny, pTrue
	}
	var keep []pnode
	for _, t := range planned {
		switch {
		case t.op == absorb:
			out.tree = t
			return out
		case t.isConst():
		default:
			keep = append(keep, t)
		}
	}
	switch len(keep) {
	case 0:
		out.tree = constant(op == pAll)
	case 1:
		out.tree = keep[0]
	default:
		out.tree = pnode{op: op, kids: keep}
	}
	return out
}

// ref plans a reference to a named condition, once per document and
// principal. A ref is transparent to depth: it stands for its expr.
func (pl *planner) ref(name string) *expansion {
	if e, ok := pl.memo[name]; ok {
		return e
	}
	expr, ok := pl.conds[name]
	if !ok || pl.dup[name] {
		return invalid(RuleUnresolvedRef)
	}
	if slices.Contains(pl.stack, name) {
		return invalid(RuleRefCycle)
	}
	pl.stack = append(pl.stack, name)
	e := pl.node(*expr)
	pl.stack = pl.stack[:len(pl.stack)-1]
	if e.rule == "" {
		named := []namedTree{{name, e.tree}}
		e = &expansion{tree: e.tree, depth: e.depth, leaves: e.leaves, named: mergeNamed(named, e.named)}
	}
	pl.memo[name] = e
	return e
}

// mergeNamed appends the conditions of b that a does not hold, keeping order.
func mergeNamed(a, b []namedTree) []namedTree {
	for _, n := range b {
		if !slices.ContainsFunc(a, func(x namedTree) bool { return x.name == n.name }) {
			a = append(a, n)
		}
	}
	return a
}

// sat caps v at limit+1: enough to tell "over the cap" without overflowing on
// a document that shares a condition many times.
func sat(v, limit int) int { return min(v, limit+1) }

// condState is how a cmp resolved.
type condState uint8

const (
	condOK      condState = iota
	condEmpty             // valid, but the set is empty or the caller has no id: false
	condInvalid           // breaks a Cond rule: voids the policy
)

// planCond resolves a Cond for p. Today's Cond rules (acl.go Cond) decide
// validity; an empty set, a self comparison for a caller with no id and
// my_groups for a caller with no groups are valid and false.
func planCond(c Cond, p identity.Principal) (plannedCond, condState) {
	pc := plannedCond{column: c.Column}
	switch c.Op {
	case CondEq, CondIn:
	default:
		return pc, condInvalid
	}
	isState := c.Column == metadata.ColumnState
	switch c.Column {
	case metadata.ColumnState, metadata.ColumnRequesterID, metadata.ColumnAssigneeID, metadata.ColumnAssignmentGroupID:
	default:
		return pc, condInvalid
	}
	switch c.Caller {
	case "":
		if c.Op == CondEq && len(c.Values) != 1 {
			return pc, condInvalid
		}
		for _, v := range c.Values {
			if isState {
				pc.strs = append(pc.strs, v)
				continue
			}
			id, err := uuid.Parse(v)
			if err != nil {
				return pc, condInvalid
			}
			pc.ids = append(pc.ids, id)
		}
	case CallerSelf:
		if len(c.Values) > 0 || (c.Column != metadata.ColumnRequesterID && c.Column != metadata.ColumnAssigneeID) {
			return pc, condInvalid
		}
		if p.PrincipalID == uuid.Nil {
			return pc, condEmpty
		}
		pc.ids, pc.scalar = []uuid.UUID{p.PrincipalID}, true
	case CallerMyGroups:
		if len(c.Values) > 0 || c.Column != metadata.ColumnAssignmentGroupID {
			return pc, condInvalid
		}
		// Zero groups stays condEmpty below (E3-2 (ii)); two or more keep = ANY.
		pc.ids, pc.scalar = slices.Clone(p.Groups), len(p.Groups) == 1
	default:
		return pc, condInvalid
	}
	if len(pc.ids) == 0 && len(pc.strs) == 0 {
		return pc, condEmpty
	}
	return pc, condOK
}

// eval is the Go rendering of a folded tree.
func (n pnode) eval(r Row) bool {
	switch n.op {
	case pTrue:
		return true
	case pAll:
		for _, k := range n.kids {
			if !k.eval(r) {
				return false
			}
		}
		return true
	case pAny:
		for _, k := range n.kids {
			if k.eval(r) {
				return true
			}
		}
		return false
	case pNot:
		return !n.kids[0].eval(r)
	case pCmp:
		return n.cond.allows(r)
	case pShared:
		return sharedWithCaller(r.SharedWith, n.caller, n.groups)
	}
	return false
}

func (c plannedCond) allows(r Row) bool {
	var col *uuid.UUID
	switch c.column {
	case metadata.ColumnState:
		return slices.Contains(c.strs, r.State)
	case metadata.ColumnRequesterID:
		col = r.RequesterID
	case metadata.ColumnAssigneeID:
		col = r.AssigneeID
	case metadata.ColumnAssignmentGroupID:
		col = r.AssignmentGroupID
	}
	return col != nil && slices.Contains(c.ids, *col)
}

// sql is the SQL rendering of eval, node for node. Every term it returns is
// two-valued and parenthesized (an EXISTS is atomic), so NOT and AND NOT
// over it are plain negation.
func (n pnode) sql(c *compiler) string {
	switch n.op {
	case pTrue:
		return "true"
	case pAll, pAny:
		sep := " AND "
		if n.op == pAny {
			sep = " OR "
		}
		parts := make([]string, len(n.kids))
		for i, k := range n.kids {
			parts[i] = k.sql(c)
		}
		return "(" + strings.Join(parts, sep) + ")"
	case pNot:
		return "(NOT " + n.kids[0].sql(c) + ")"
	case pCmp:
		return n.cond.sql(c)
	case pShared:
		return c.shared(n.caller, n.groups)
	}
	return "false"
}

func (cd plannedCond) sql(c *compiler) string {
	switch {
	case cd.column == metadata.ColumnState:
		return "(" + c.col(cd.column) + " = ANY(" + c.arg(cd.strs) + "::text[]))"
	case cd.scalar:
		return c.colEq(cd.column, cd.ids[0])
	}
	return "(" + c.col(cd.column) + " IS NOT NULL AND " + c.col(cd.column) + " = ANY(" + c.arg(UUIDTexts(cd.ids)) + "::uuid[]))"
}

// trace is the value on r of every named condition pp reaches.
func (pp plannedPolicy) trace(r Row) []TraceStep {
	if len(pp.named) == 0 {
		return nil
	}
	out := make([]TraceStep, len(pp.named))
	for i, n := range pp.named {
		out[i] = TraceStep{Policy: pp.name, Condition: n.name, Value: n.tree.eval(r)}
	}
	return out
}

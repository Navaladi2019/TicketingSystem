package isolation

// The oracles and mutants of the format-2 I5 agreement (PLAN-011 P11-10,
// TICK-942). Nothing here calls into internal/authz's planner: the reference
// evaluator and the S10 rules are written from ADR-010 D5 and from main
// before S30, so a planner bug cannot hide behind its own renderings. The
// generator and the unit tests are i5_format2_gen_test.go; the database half
// is i5_format2_isolation_test.go.

import (
	"fmt"
	"regexp"
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
)

// ---- The reference evaluator (D5) -------------------------------------------

// i5Val is a node evaluated on one row for one caller: its value, its depth
// and leaf count after expansion, or bad when it voids the policy.
type i5Val struct {
	v             bool
	depth, leaves int
	bad           bool
}

// i5RefEval evaluates a document's nodes for one caller and row.
type i5RefEval struct {
	doc authz.Document
	p   identity.Principal
	row authz.Row
}

// i5Reference is the reference answer: does some policy of pol granting op
// hold on row for p, and is it valid. A nil policy grants nothing.
func i5Reference(pol *authz.Policy, p identity.Principal, op string, row authz.Row) bool {
	if pol == nil {
		return false
	}
	for _, np := range pol.Document.Policies {
		if v, valid := i5RefPolicy(pol.Document, np, p, row); slices.Contains(np.Ops, op) && valid && v {
			return true
		}
	}
	return false
}

// i5RefPolicy is one policy's value and validity: the caps of plan.go
// (MaxDepth, MaxPolicyLeaves) apply after expansion, before any folding.
func i5RefPolicy(d authz.Document, np authz.NamedPolicy, p identity.Principal, row authz.Row) (v, valid bool) {
	e := i5RefEval{doc: d, p: p, row: row}
	x := e.eval(np.When, nil)
	if x.bad || x.depth > authz.MaxDepth || x.leaves > authz.MaxPolicyLeaves {
		return false, false
	}
	return x.v, true
}

func (e i5RefEval) eval(n authz.Node, stack []string) i5Val {
	leaf := func(v bool) i5Val { return i5Val{v: v, depth: 1, leaves: 1} }
	switch n.Kind() {
	case authz.NodeAll, authz.NodeAny:
		kids, all := n.Any, n.All != nil
		if all {
			kids = n.All
		}
		if len(kids) > authz.MaxChildren || (all && len(kids) == 0) {
			return i5Val{bad: true}
		}
		out := i5Val{v: all}
		for _, k := range kids {
			x := e.eval(k, stack)
			if x.bad {
				return x
			}
			out.depth = max(out.depth, x.depth)
			out.leaves += x.leaves
			if all {
				out.v = out.v && x.v
			} else {
				out.v = out.v || x.v
			}
		}
		out.depth++
		return out
	case authz.NodeNot:
		x := e.eval(*n.Not, stack)
		x.v, x.depth = !x.v, x.depth+1
		return x
	case authz.NodeRef:
		var expr *authz.Node
		for i, c := range e.doc.Conditions {
			if c.Name == n.Ref {
				if expr != nil {
					return i5Val{bad: true} // declared twice: unresolved
				}
				expr = &e.doc.Conditions[i].Expr
			}
		}
		if expr == nil || slices.Contains(stack, n.Ref) {
			return i5Val{bad: true}
		}
		return e.eval(*expr, append(slices.Clone(stack), n.Ref))
	case authz.NodeMemberOf:
		m := n.MemberOf
		return leaf(slices.ContainsFunc(m.Groups, func(g uuid.UUID) bool { return slices.Contains(e.p.Groups, g) }) ||
			(e.p.PrincipalID != uuid.Nil && slices.Contains(m.Users, e.p.PrincipalID)))
	case authz.NodeCmp:
		v, bad := i5RefCmp(*n.Cmp, e.p, e.row)
		if bad {
			return i5Val{bad: true}
		}
		return leaf(v)
	case authz.NodeShared:
		return leaf(i5RefShared(e.p, e.row))
	case authz.NodeUnder:
		// D7, D5a: the row's unit is in set S (the live subtrees of the
		// caller's units or of the listed ones); a NULL unit never is.
		var set []uuid.UUID
		if n.Under.Caller {
			set = i5CallerSet(e.p)
		}
		for _, root := range n.Under.Units {
			set = append(set, i5Subtree(e.p.TenantID, root)...)
		}
		return leaf(e.row.OrgUnitID != nil && slices.Contains(set, *e.row.OrgUnitID))
	}
	return i5Val{bad: true} // no kind, anything unknown
}

// i5RefCmp is a Cond on a row under acl.go's Cond rules: bad when it breaks
// one (whatever the caller), false on a NULL column, a nil caller for self or
// an empty set.
func i5RefCmp(c authz.Cond, p identity.Principal, row authz.Row) (v, bad bool) {
	if c.Op != authz.CondEq && c.Op != authz.CondIn {
		return false, true
	}
	var col *uuid.UUID
	switch c.Column {
	case "state":
	case "requester_id":
		col = row.RequesterID
	case "assignee_id":
		col = row.AssigneeID
	case "assignment_group_id":
		col = row.AssignmentGroupID
	default:
		return false, true
	}
	var set []uuid.UUID
	switch c.Caller {
	case "":
		if c.Op == authz.CondEq && len(c.Values) != 1 {
			return false, true
		}
		if c.Column == "state" {
			return slices.Contains(c.Values, row.State), false
		}
		for _, s := range c.Values {
			id, err := uuid.Parse(s)
			if err != nil {
				return false, true
			}
			set = append(set, id)
		}
	case authz.CallerSelf:
		if len(c.Values) > 0 || (c.Column != "requester_id" && c.Column != "assignee_id") {
			return false, true
		}
		if p.PrincipalID != uuid.Nil {
			set = []uuid.UUID{p.PrincipalID}
		}
	case authz.CallerMyGroups:
		if len(c.Values) > 0 || c.Column != "assignment_group_id" {
			return false, true
		}
		set = p.Groups
	default:
		return false, true
	}
	return col != nil && slices.Contains(set, *col), false
}

// i5RefShared: a share names the caller (a pair-matched Row.SharedWith holds
// the caller's id or one of its groups); a caller with no id has none.
func i5RefShared(p identity.Principal, row authz.Row) bool {
	if p.PrincipalID == uuid.Nil {
		return false
	}
	return slices.ContainsFunc(row.SharedWith, func(id uuid.UUID) bool {
		return id == p.PrincipalID || slices.Contains(p.Groups, id)
	})
}

// i5S10Allows is the S10 meaning of format-1 rules (main before S30): a rule
// grants op on row when it lists op, names the caller (by group or user id,
// or, for a caller with an id, as requester, assignee or a share of the row)
// and every condition holds. A condition that breaks a Cond rule makes its
// rule match nothing.
func i5S10Allows(rules []authz.Rule, p identity.Principal, op string, row authz.Row) bool {
	for _, rule := range rules {
		if !slices.Contains(rule.Ops, op) {
			continue
		}
		who := rule.Who
		named := slices.ContainsFunc(who.Groups, func(g uuid.UUID) bool { return slices.Contains(p.Groups, g) }) ||
			(p.PrincipalID != uuid.Nil && slices.Contains(who.Users, p.PrincipalID))
		if !named && p.PrincipalID != uuid.Nil {
			is := func(col *uuid.UUID) bool { return col != nil && *col == p.PrincipalID }
			named = (who.Requester && is(row.RequesterID)) || (who.Assignee && is(row.AssigneeID)) ||
				(who.Shared && i5RefShared(p, row))
		}
		if !named {
			continue
		}
		holds := true
		for _, c := range rule.Where {
			if v, bad := i5RefCmp(c, p, row); bad || !v {
				holds = false
				break
			}
		}
		if holds {
			return true
		}
	}
	return false
}

// ---- The five format-2 mutants (P11-10) -------------------------------------

// i5F2Compiler is CompileRead for a format-2 case: the real compiler, or a
// mutant that may read the case's previous conditions.
type i5F2Compiler func(c *i5F2Case, p identity.Principal, alias string, argBase int) (string, []any)

func i5F2Real(c *i5F2Case, p identity.Principal, alias string, argBase int) (string, []any) {
	return authz.CompileRead(c.bound(p), p, alias, argBase)
}

// i5Guarded is the IS NOT NULL guard of a uuid cmp in CompileRead's text
// (plan.go plannedCond.sql and compiler.colEq).
var i5Guarded = regexp.MustCompile(`\b[a-z_]+\.(requester_id|assignee_id|assignment_group_id) IS NOT NULL AND `)

// i5SentinelCaller is a principal id no world holds: shared_nil_caller
// compiles with it so the user-share half matches nothing.
func i5SentinelCaller() uuid.UUID { return uuid.MustParse("0f000000-0000-4000-8000-00000000f2f2") }

// i5F2Mutants are deliberately broken compilers, each the bug a guard in the
// planner or compiler exists to prevent. Each breaks the SQL side only, so
// authz.Allows (and the reference) is the oracle the database run compares
// against. under_direct_only and under_ignores_tombstone are S31's (TICK-951
// P11-18, reworked for the closure by TICK-1034): the leaf's closure
// subquery rewritten to a broken walk of the org tree.
func i5F2Mutants() map[string]i5F2Compiler {
	return map[string]i5F2Compiler{
		// not_unguarded: a uuid cmp loses "col IS NOT NULL AND", so it is
		// three-valued, and NOT over a NULL column is NULL (no row) where D5
		// says true.
		"not_unguarded": func(c *i5F2Case, p identity.Principal, alias string, argBase int) (string, []any) {
			sql, args := authz.CompileRead(c.bound(p), p, alias, argBase)
			return i5Guarded.ReplaceAllString(sql, ""), args
		},
		// invalid_leaf_false: an invalid node (a cmp that breaks a Cond rule,
		// a node with no kind, an empty all, an unresolved ref) is
		// read as false instead of voiding its policy; under a not it grants.
		"invalid_leaf_false": func(c *i5F2Case, p identity.Principal, alias string, argBase int) (string, []any) {
			d := c.Policy.Document
			return authz.CompileRead(i5WithDocument(c.bound(p), i5MapDocument(d, func(n authz.Node) authz.Node {
				if i5InvalidLeaf(d, n) {
					return i5False()
				}
				return n
			})), p, alias, argBase)
		},
		// ref_previous_version: a ref resolves against the previous version's
		// conditions, not the document's own.
		"ref_previous_version": func(c *i5F2Case, p identity.Principal, alias string, argBase int) (string, []any) {
			if c.Policy == nil {
				return authz.CompileRead(nil, p, alias, argBase)
			}
			d := c.Policy.Document
			d.Conditions = c.Prev
			return authz.CompileRead(i5WithDocument(c.bound(p), d), p, alias, argBase)
		},
		// fold_wrong: the caller-only fold of member_of is inverted.
		"fold_wrong": func(c *i5F2Case, p identity.Principal, alias string, argBase int) (string, []any) {
			if c.Policy == nil {
				return authz.CompileRead(nil, p, alias, argBase)
			}
			return authz.CompileRead(i5WithDocument(c.bound(p), i5MapDocument(c.Policy.Document, func(n authz.Node) authz.Node {
				if n.MemberOf != nil {
					return authz.Node{Not: &n}
				}
				return n
			})), p, alias, argBase)
		},
		// under_direct_only: the region is the caller's units (or the listed
		// roots) without their descendants, so a record in a unit below them
		// drops out of it. The leaf reads the closure in SQL, so the mutant
		// rewrites the leaf's subquery to the closure's self rows: a live unit
		// of U, never one below it (i5SubtreeDirectOnly is its Go model).
		"under_direct_only": func(c *i5F2Case, p identity.Principal, alias string, argBase int) (string, []any) {
			sql, args := authz.CompileRead(c.bound(p), p, alias, argBase)
			return i5UnderSubquery.ReplaceAllString(sql, `IN (SELECT acl_ou.descendant_id FROM org_unit_closure acl_ou`+
				` WHERE acl_ou.tenant_id = $1::uuid AND acl_ou.ancestor_id = ANY($2::uuid[]) AND acl_ou.descendant_id = acl_ou.ancestor_id)`), args
		},
		// under_ignores_tombstone: tombstoned units stay in the region, and so
		// does a live unit below one, so access through a unit removed from
		// the directory survives the sync. Rewritten to a walk down org_units from U that never reads
		// deleted_at (i5SubtreeIgnoresTombstone is its Go model).
		"under_ignores_tombstone": func(c *i5F2Case, p identity.Principal, alias string, argBase int) (string, []any) {
			sql, args := authz.CompileRead(c.bound(p), p, alias, argBase)
			return i5UnderSubquery.ReplaceAllString(sql, `IN (WITH RECURSIVE acl_w(id) AS (SELECT o.id FROM org_units o`+
				` WHERE o.tenant_id = $1::uuid AND o.id = ANY($2::uuid[]) UNION SELECT k.id FROM acl_w`+
				` JOIN org_units k ON k.tenant_id = $1::uuid AND k.parent_id = acl_w.id) SELECT id FROM acl_w)`), args
		},
		// shared_nil_caller: shared holds for a caller with no id through its
		// groups' shares, where D5 folds it to false. Compiled as a caller
		// with a sentinel id (whose user shares match nothing) and the same
		// groups, the caller's self cmps already folded to false.
		"shared_nil_caller": func(c *i5F2Case, p identity.Principal, alias string, argBase int) (string, []any) {
			if c.Policy == nil || p.PrincipalID != uuid.Nil || len(p.Groups) == 0 {
				return authz.CompileRead(c.bound(p), p, alias, argBase)
			}
			q := p
			q.PrincipalID = i5SentinelCaller()
			return authz.CompileRead(i5WithDocument(c.bound(q), i5MapDocument(c.Policy.Document, func(n authz.Node) authz.Node {
				if n.Cmp != nil && n.Cmp.Caller == authz.CallerSelf {
					if _, bad := i5RefCmp(*n.Cmp, p, authz.Row{}); !bad {
						return i5False()
					}
				}
				return n
			})), q, alias, argBase)
		},
	}
}

// i5UnderSubquery matches the under leaf's closure subquery as
// authz.CompileRead renders it; $1 is its tenant placeholder, $2 its U.
var i5UnderSubquery = regexp.MustCompile(`IN \(SELECT acl_ou\.descendant_id FROM org_unit_closure acl_ou` +
	` WHERE acl_ou\.tenant_id = (\$[0-9]+)::uuid AND acl_ou\.ancestor_id = ANY\((\$[0-9]+)::uuid\[\]\)\)`)

// i5F2BoundWith is c.bound(p) with the closure from subtree instead of the
// reference's walk; a case with no world is returned as it is.
func i5F2BoundWith(c *i5F2Case, p identity.Principal, subtree func(tenant, root uuid.UUID) []uuid.UUID) *authz.Policy {
	if c.Policy == nil || c.World == nil {
		return c.Policy
	}
	return i5BindWith(c.Policy, p, c.Prev, subtree)
}

// i5UnderModel is an under mutant's model: Allows over the policy bound with
// S from subtree. The binding depends only on the case and the principal, so
// it is made once for each and reused for every record (the models run per
// record, and the walk is the cost).
func i5UnderModel(allows func(*authz.Policy, identity.Principal, authz.Row) bool, subtree func(tenant, root uuid.UUID) []uuid.UUID) func(c *i5F2Case, p identity.Principal, rec i5Record) bool {
	type key struct {
		c *i5F2Case
		p string
	}
	bound := map[key]*authz.Policy{}
	return func(c *i5F2Case, p identity.Principal, rec i5Record) bool {
		k := key{c, fmt.Sprint(p)}
		pol, ok := bound[k]
		if !ok {
			pol = i5F2BoundWith(c, p, subtree)
			bound[k] = pol
		}
		return allows(pol, p, rec.row(p))
	}
}

// i5F2MutantNames is i5F2Mutants' keys in order.
func i5F2MutantNames() []string {
	var names []string
	for n := range i5F2Mutants() {
		names = append(names, n)
	}
	slices.Sort(names)
	return names
}

// i5InvalidLeaf reports a node that voids its policy by itself, whatever its
// children: no kind, an empty all, a cmp that breaks a Cond rule, or
// a ref to a name that is not declared exactly once.
func i5InvalidLeaf(d authz.Document, n authz.Node) bool {
	switch n.Kind() {
	case "":
		return true
	case authz.NodeAll:
		return len(n.All) == 0
	case authz.NodeCmp:
		_, bad := i5RefCmp(*n.Cmp, identity.Principal{}, authz.Row{})
		return bad
	case authz.NodeRef:
		k := 0
		for _, c := range d.Conditions {
			if c.Name == n.Ref {
				k++
			}
		}
		return k != 1
	}
	return false
}

func i5WithDocument(pol *authz.Policy, d authz.Document) *authz.Policy {
	if pol == nil {
		return nil
	}
	out := *pol
	out.Document = d
	return &out
}

// i5MapDocument is d with f applied bottom-up to every node of every
// condition and policy; d is not modified.
func i5MapDocument(d authz.Document, f func(authz.Node) authz.Node) authz.Document {
	out := d
	out.Conditions = make([]authz.Condition, len(d.Conditions))
	for i, c := range d.Conditions {
		out.Conditions[i] = authz.Condition{Name: c.Name, Expr: i5MapNode(c.Expr, f)}
	}
	out.Policies = make([]authz.NamedPolicy, len(d.Policies))
	for i, np := range d.Policies {
		np.When = i5MapNode(np.When, f)
		out.Policies[i] = np
	}
	return out
}

func i5MapNode(n authz.Node, f func(authz.Node) authz.Node) authz.Node {
	mapKids := func(kids []authz.Node) []authz.Node {
		out := make([]authz.Node, len(kids))
		for i, k := range kids {
			out[i] = i5MapNode(k, f)
		}
		return out
	}
	switch {
	case n.All != nil:
		n.All = mapKids(n.All)
	case n.Any != nil:
		n.Any = mapKids(n.Any)
	case n.Not != nil:
		k := i5MapNode(*n.Not, f)
		n.Not = &k
	}
	return f(n)
}

// i5Kleene is a policy's value under SQL's three-valued logic when a uuid cmp
// loses its IS NOT NULL guard (the not_unguarded mutant's semantics): such a
// cmp on a NULL column is unknown, NOT unknown is unknown, and a row is
// returned only when the predicate is true. 0 false, 1 true, 2 unknown.
func i5Kleene(d authz.Document, n authz.Node, p identity.Principal, row authz.Row, stack []string) int {
	switch n.Kind() {
	case authz.NodeAll, authz.NodeAny:
		kids, all := n.Any, n.All != nil
		if all {
			kids = n.All
		}
		out := 0
		if all {
			out = 1
		}
		for _, k := range kids {
			v := i5Kleene(d, k, p, row, stack)
			switch {
			case all && (v == 0 || out == 0):
				out = 0
			case all && v == 2:
				out = 2
			case !all && (v == 1 || out == 1):
				out = 1
			case !all && v == 2:
				out = 2
			}
		}
		return out
	case authz.NodeNot:
		if v := i5Kleene(d, *n.Not, p, row, stack); v != 2 {
			return 1 - v
		}
		return 2
	case authz.NodeRef:
		for _, c := range d.Conditions {
			if c.Name == n.Ref {
				return i5Kleene(d, c.Expr, p, row, append(slices.Clone(stack), n.Ref))
			}
		}
		return 0
	case authz.NodeCmp:
		c := *n.Cmp
		var col *uuid.UUID
		switch c.Column {
		case "requester_id":
			col = row.RequesterID
		case "assignee_id":
			col = row.AssigneeID
		case "assignment_group_id":
			col = row.AssignmentGroupID
		}
		v, _ := i5RefCmp(c, p, row)
		// A cmp that folds to a constant (an empty set, a nil caller for
		// self) never reaches the SQL; one that reaches it is unknown on a
		// NULL column once unguarded.
		if c.Column != "state" && col == nil && !i5CmpFoldsEmpty(c, p) {
			return 2
		}
		if v {
			return 1
		}
		return 0
	}
	x := i5RefEval{doc: d, p: p, row: row}.eval(n, stack)
	if x.v {
		return 1
	}
	return 0
}

// i5CmpFoldsEmpty: the planner folds the cmp to false before rendering it.
func i5CmpFoldsEmpty(c authz.Cond, p identity.Principal) bool {
	switch c.Caller {
	case authz.CallerSelf:
		return p.PrincipalID == uuid.Nil
	case authz.CallerMyGroups:
		return len(p.Groups) == 0
	}
	return len(c.Values) == 0
}

// i5F2MutantModels are the mutants' meaning in Go: the rows each mutant's SQL
// admits, given the real compiler and Allows agree (the database row's
// claim). TestI5Format2MutantsChangeTheRows uses them to show each mutant
// admits a different set of rows on the default run, so a caught mutant row
// in CI is a real catch and an uncaught one a real gap.
func i5F2MutantModels() map[string]func(c *i5F2Case, p identity.Principal, rec i5Record) bool {
	allows := func(pol *authz.Policy, p identity.Principal, row authz.Row) bool {
		_, _, ok := authz.Allows(pol, p, authz.OpRead, row)
		return ok
	}
	transformed := func(f func(d authz.Document, n authz.Node) authz.Node) func(c *i5F2Case, p identity.Principal, rec i5Record) bool {
		return func(c *i5F2Case, p identity.Principal, rec i5Record) bool {
			if c.Policy == nil {
				return false
			}
			d := c.Policy.Document
			return allows(i5WithDocument(c.bound(p), i5MapDocument(d, func(n authz.Node) authz.Node { return f(d, n) })), p, rec.row(p))
		}
	}
	return map[string]func(c *i5F2Case, p identity.Principal, rec i5Record) bool{
		"not_unguarded": func(c *i5F2Case, p identity.Principal, rec i5Record) bool {
			if c.Policy == nil {
				return false
			}
			row := rec.row(p)
			for _, np := range c.Policy.Document.Policies {
				if _, valid := i5RefPolicy(c.Policy.Document, np, p, row); valid && slices.Contains(np.Ops, authz.OpRead) &&
					i5Kleene(c.Policy.Document, np.When, p, row, nil) == 1 {
					return true
				}
			}
			return false
		},
		"invalid_leaf_false": transformed(func(d authz.Document, n authz.Node) authz.Node {
			if i5InvalidLeaf(d, n) {
				return i5False()
			}
			return n
		}),
		"ref_previous_version": func(c *i5F2Case, p identity.Principal, rec i5Record) bool {
			if c.Policy == nil {
				return false
			}
			d := c.Policy.Document
			d.Conditions = c.Prev
			return allows(i5WithDocument(c.bound(p), d), p, rec.row(p))
		},
		"fold_wrong": transformed(func(_ authz.Document, n authz.Node) authz.Node {
			if n.MemberOf != nil {
				return authz.Node{Not: &n}
			}
			return n
		}),
		"under_direct_only":       i5UnderModel(allows, i5SubtreeDirectOnly),
		"under_ignores_tombstone": i5UnderModel(allows, i5SubtreeIgnoresTombstone),
		"shared_nil_caller": func(c *i5F2Case, p identity.Principal, rec i5Record) bool {
			if c.Policy == nil || p.PrincipalID != uuid.Nil || len(p.Groups) == 0 {
				return allows(c.bound(p), p, rec.row(p))
			}
			q := p
			q.PrincipalID = i5SentinelCaller()
			d := c.Policy.Document
			return allows(i5WithDocument(c.bound(q), i5MapDocument(d, func(n authz.Node) authz.Node {
				if n.Cmp != nil && n.Cmp.Caller == authz.CallerSelf {
					if _, bad := i5RefCmp(*n.Cmp, p, authz.Row{}); !bad {
						return i5False()
					}
				}
				return n
			})), q, rec.row(q))
		},
	}
}

// TestI5Format2MutantsChangeTheRows: every mutant admits a different set of
// rows from Allows for some case of the default run, and the models cover
// exactly the mutants the database run reports.
func TestI5Format2MutantsChangeTheRows(t *testing.T) {
	i5Quiet(t)
	w, cases := i5F2Generate(i5DefaultSeed)
	models := i5F2MutantModels()
	if len(models) != len(i5F2Mutants()) {
		t.Fatalf("%d models for %d mutants", len(models), len(i5F2Mutants()))
	}
	for _, name := range i5F2MutantNames() {
		model, ok := models[name]
		if !ok {
			t.Fatalf("mutant %s has no model", name)
		}
		found := ""
	search:
		for ci := range cases {
			c := &cases[ci]
			for ki, p := range c.Principals {
				for _, rec := range w.Records {
					_, _, want := authz.Allows(c.bound(p), p, authz.OpRead, rec.row(p))
					if model(c, p, rec) != want {
						found = fmt.Sprintf("document #%d principal #%d record %s", ci, ki, rec.Row.ID)
						break search
					}
				}
			}
		}
		if found == "" {
			t.Errorf("mutant %s admits the same rows as Allows on every case of the default run", name)
			continue
		}
		t.Logf("mutant %s first differs at %s", name, found)
	}
}

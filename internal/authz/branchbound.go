package authz

import (
	"slices"

	"github.com/Navaladi2019/TicketingSystem/internal/identity"
)

// ReadBranchBound is the most ReadBranches branches d can give any caller
// before the SP-2 threshold applies: an upper bound on the sum, over d's
// valid read policies, of each policy's branches after folding (ADR-010 D6).
// The D6 check at planning compares it with the threshold for each tenant's
// newest document, as docs/data/acl-format2-caps.md item 4 (d6.sql) does for
// format 1, where it gives the same number.
//
// Folding depends on the caller, and can make more branches than the stored
// tree shows (all[any[a,b], any[member_of, c]] has one branch, but three for a
// member), so the bound follows every shape each node can fold to. A voided
// policy gives no branch, as it does on the read path; validity does not
// depend on the caller.
func ReadBranchBound(d Document) int {
	pl := newPlanner(d, identity.Principal{})
	memo := map[string]shape{}
	total := 0
	for _, np := range d.Policies {
		if !slices.Contains(np.Ops, OpRead) || pl.policy(np).err != nil {
			continue
		}
		total += boundShape(np.When, d, memo).branches()
	}
	return total
}

// shape is what a node can fold to, over every caller: true, false, a single
// term (one branch), or something that splits into up to width branches (an
// any of that many children, or an all that D6 distributes over its only any
// child, which has at most as many). width is 0 when no split is possible.
type shape struct {
	t, f  bool
	term  bool
	width int
}

// branches is the most disjuncts a policy whose tree has this shape gives; a
// tree folded to true is the single branch "true".
func (s shape) branches() int {
	if s.term || s.t {
		return max(s.width, 1)
	}
	return s.width
}

// varies reports whether the node can fold to something other than a
// constant.
func (s shape) varies() bool { return s.term || s.width > 0 }

// boundShape is n's shape. n belongs to a policy the planner found valid, so
// every ref resolves, there is no cycle and every cmp is well formed.
func boundShape(n Node, d Document, memo map[string]shape) shape {
	switch n.Kind() {
	case NodeAll, NodeAny:
		kids := n.All
		if n.Kind() == NodeAny {
			kids = n.Any
		}
		shapes := make([]shape, len(kids))
		for i, k := range kids {
			shapes[i] = boundShape(k, d, memo)
		}
		return groupShape(n.Kind() == NodeAll, shapes)
	case NodeNot:
		k := boundShape(*n.Not, d, memo)
		return shape{t: k.f, f: k.t, term: k.varies()}
	case NodeRef:
		if s, ok := memo[n.Ref]; ok {
			return s
		}
		for _, c := range d.Conditions {
			if c.Name == n.Ref {
				s := boundShape(c.Expr, d, memo)
				memo[n.Ref] = s
				return s
			}
		}
	case NodeMemberOf:
		return shape{t: true, f: true}
	case NodeShared:
		return shape{f: true, term: true} // false for a caller with no id
	case NodeCmp:
		if n.Cmp.Caller != "" {
			return shape{f: true, term: true} // false with no id or no groups
		}
		if len(n.Cmp.Values) == 0 {
			return shape{f: true}
		}
		return shape{term: true}
	}
	return shape{f: true} // unreachable for a valid policy
}

// groupShape is the shape of an all (all is true) or an any over children of
// the given shapes, following planner.group's folding: the absorbing constant
// (false for an all, true for an any) wins, the other is dropped, one kept
// child stands for the group, and two or more make an all or an any of that
// many. An all of two or more splits into at most its widest child's width
// (D6 distributes only an any child), else it is one term. Children are
// treated as independent, which can only overstate.
func groupShape(all bool, kids []shape) shape {
	out := shape{t: all, f: !all}
	varying := 0
	for _, k := range kids {
		if all {
			out.t, out.f = out.t && k.t, out.f || k.f
		} else {
			out.t, out.f = out.t || k.t, out.f && k.f
		}
		if k.varies() {
			varying++
			out.term = out.term || k.term
			out.width = max(out.width, k.width)
		}
	}
	if varying >= 2 {
		if all {
			out.term = true
		} else {
			out.width = max(out.width, varying)
		}
	}
	return out
}

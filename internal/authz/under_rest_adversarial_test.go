package authz

import (
	"context"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/identity"
)

// Adversarial guard for underScope.rest (TICK-950 security review of
// 2a1a1a32). rest reads every under leaf the probe plan folded away as an
// empty S, so `not under X` reads true there. That is safe only if a
// folded-away leaf really cannot change any decision. If the probe ever
// dropped a leaf that matters, a `not under X` would grant a row in X's
// subtree without any query: the policy would admit records it must refuse.
//
// So: for random documents with under leaves (caller and listed, under
// not, inside refs), for every principal for which UnderNeeded is false
// (ResolveUnder runs no query and binds only rest), the decision on every row,
// read and write, and the read SQL must be the same as with S bound to
// nothing, to every unit, and to the two mixed bindings (caller full and
// listed roots empty, and the reverse). If they differ, the leaf mattered and
// rest decided it.
func TestUnderRestNeverDecidesALeafThatMatters(t *testing.T) {
	t.Parallel()
	r := rand.New(rand.NewPCG(20261121, 950))
	units := []uuid.UUID{utUnit(1), utUnit(2), utUnit(3)}
	tenant := utTenant()
	withTenant := func(p identity.Principal) identity.Principal { p.TenantID = tenant; return p }
	principals := map[string]identity.Principal{
		"member": utMe(), "lonely": withTenant(ptLonely()), "no id": withTenant(ptNoID()),
	}
	rows := []Row{{State: "open"}}
	for _, u := range units {
		rows = append(rows, Row{State: "open", OrgUnitID: ptRef(u)}, Row{State: "closed", OrgUnitID: ptRef(u), RequesterID: ptRef(ptCaller())})
	}
	for _, rr := range ptRandomRows(r, 4) {
		rr.OrgUnitID = ptRef(units[r.IntN(len(units))])
		rows = append(rows, rr)
	}
	none := map[uuid.UUID][]uuid.UUID{}
	all := map[uuid.UUID][]uuid.UUID{}
	for _, u := range units {
		none[u] = nil
		all[u] = slices.Clone(units)
	}
	checked, folded := 0, 0
	for range 150 {
		d := ptRandomDoc(r)
		for i := range d.Conditions {
			d.Conditions[i].Expr = utSprinkleUnder(r, d.Conditions[i].Expr, units)
		}
		for i := range d.Policies {
			d.Policies[i].When = utSprinkleUnder(r, d.Policies[i].When, units)
		}
		if !utHasUnder(d) {
			continue
		}
		for name, p := range principals {
			pol := &Policy{EntityType: "ut_rest_adversarial", Document: d}
			if UnderNeeded(pol, p) {
				continue // ResolveUnder queries; the integration and I5 suites cover that path
			}
			folded++
			got, err := ResolveUnder(context.Background(), nil, pol, p)
			if err != nil {
				t.Fatalf("%s: ResolveUnder with no surviving leaf: %v", name, err)
			}
			// The joint extremes, and the two mixed bindings: a pair such as
			// all[under caller, not under X] that matters only when the caller's
			// S and a listed root's S differ is caught by one of the mixed ones.
			oracles := []struct {
				name string
				pol  *Policy
			}{
				{"S empty", BindUnder(pol, p, nil, none)},
				{"S full", BindUnder(pol, p, slices.Clone(units), all)},
				{"caller full, units empty", BindUnder(pol, p, slices.Clone(units), none)},
				{"caller empty, units full", BindUnder(pol, p, nil, all)},
			}
			for _, op := range []string{OpRead, OpWrite} {
				for ri, row := range rows {
					_, _, g := Allows(got, p, op, row)
					for _, o := range oracles {
						checked++
						if _, _, w := Allows(o.pol, p, op, row); g != w {
							t.Fatalf("%s %s row %d: rest decided a leaf that matters: ResolveUnder=%v, %s=%v\ndocument: %+v",
								name, op, ri, g, o.name, w, d)
						}
					}
				}
			}
			gs, _ := CompileRead(got, p, "r", 2)
			for _, o := range oracles {
				if ws, _ := CompileRead(o.pol, p, "r", 2); gs != ws {
					t.Fatalf("%s: read SQL depends on S although no leaf survives: rest %q, %s %q", name, gs, o.name, ws)
				}
			}
		}
	}
	if folded < 50 {
		t.Fatalf("only %d (document, principal) pairs had every under leaf folded away; the generator does not reach the rest path", folded)
	}
	t.Logf("%d folded-away cases, %d decisions compared", folded, checked)
}

// utSprinkleUnder replaces some leaves of n with under leaves (caller, or
// listed units), sometimes under a not.
func utSprinkleUnder(r *rand.Rand, n Node, units []uuid.UUID) Node {
	switch {
	case n.All != nil:
		for i := range n.All {
			n.All[i] = utSprinkleUnder(r, n.All[i], units)
		}
		return n
	case n.Any != nil:
		for i := range n.Any {
			n.Any[i] = utSprinkleUnder(r, n.Any[i], units)
		}
		return n
	case n.Not != nil:
		k := utSprinkleUnder(r, *n.Not, units)
		n.Not = &k
		return n
	case n.Ref != "":
		return n
	}
	if r.IntN(3) != 0 {
		return n
	}
	var u Under
	if r.IntN(2) == 0 {
		u.Caller = true
	} else {
		u.Units = []uuid.UUID{units[r.IntN(len(units))]}
	}
	leaf := Node{Under: &u}
	if r.IntN(2) == 0 {
		return Node{Not: &leaf}
	}
	return leaf
}

func utHasUnder(d Document) bool {
	var has func(Node) bool
	has = func(n Node) bool {
		if n.Under != nil {
			return true
		}
		if n.Not != nil && has(*n.Not) {
			return true
		}
		for _, k := range append(slices.Clone(n.All), n.Any...) {
			if has(k) {
				return true
			}
		}
		return false
	}
	for _, c := range d.Conditions {
		if has(c.Expr) {
			return true
		}
	}
	for _, p := range d.Policies {
		if has(p.When) {
			return true
		}
	}
	return false
}

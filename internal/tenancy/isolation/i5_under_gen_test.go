package isolation

// The I5 suite for the under leaf over random org trees (TICK-951 P11-18,
// ADR-010 D7, I5). The world's tree is drawn per seed (orgTree, drawTree):
// depth up to 4, random tombstones, callers in 0 to 3 units, records in no
// unit. These tests run without a database (task test); the database half is
// the format-2 agreement row and its two under mutant rows
// (i5_format2_isolation_test.go).

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
)

// i5UnderSeeds is how many seeds after the default one the cross-seed tests
// draw. Each draws a whole format-2 run; the search stops at the first
// difference, so most seeds cost one generation.
const i5UnderSeeds = 8

// TestI5OrgTreeIsRandomAndRisky: every seed's tree holds the shapes the under
// mutants need (i5TreeRisky), stays within the depth and size bounds, lists
// parents before children, and differs between seeds; callers are in 0 to 3
// live units only (org_unit_members references live units).
func TestI5OrgTreeIsRandomAndRisky(t *testing.T) {
	shapes := map[int]bool{}
	for s := range uint64(i5UnderSeeds) {
		w, _ := i5Generate(i5DefaultSeed + s)
		if !i5TreeRisky(w.Units) {
			t.Errorf("seed %d: the org tree lacks a risky shape", i5DefaultSeed+s)
		}
		if len(w.Units) > i5MaxUnits {
			t.Errorf("seed %d: %d units", i5DefaultSeed+s, len(w.Units))
		}
		shapes[len(w.Units)] = true
		for i, u := range w.Units {
			if d, _ := i5UnitDepth(w.Units, u.ID); d > i5TreeMaxDepth {
				t.Errorf("seed %d: unit %d at depth %d", i5DefaultSeed+s, i, d)
			}
			if u.Parent != uuid.Nil && slices.IndexFunc(w.Units[:i], func(x i5Unit) bool { return x.ID == u.Parent }) < 0 {
				t.Errorf("seed %d: unit %d is listed before its parent", i5DefaultSeed+s, i)
			}
			if u.Parent == uuid.Nil && u.Dead {
				t.Errorf("seed %d: root %d is tombstoned", i5DefaultSeed+s, i)
			}
		}
		for p, units := range w.Members {
			if len(units) > 3 {
				t.Errorf("seed %d: principal %s in %d units", i5DefaultSeed+s, p, len(units))
			}
			for _, id := range units {
				if !slices.ContainsFunc(w.Units, func(u i5Unit) bool { return u.ID == id && !u.Dead }) {
					t.Errorf("seed %d: principal %s is a member of %s, not a live unit", i5DefaultSeed+s, p, id)
				}
			}
		}
	}
	if len(shapes) < 2 {
		t.Errorf("%d seeds drew trees of one size; the tree is not random", i5UnderSeeds)
	}
}

// TestI5UnderMutantsCaughtAcrossSeeds: on every seed, not only the CI one,
// each under mutant admits a different set of rows from Allows for some case.
// A random tree that happened to hide a mutant would make the database row
// pass on luck; i5TreeRisky is what rules that out, and this test checks it.
func TestI5UnderMutantsCaughtAcrossSeeds(t *testing.T) {
	i5Quiet(t)
	subtrees := map[string]func(tenant, root uuid.UUID) []uuid.UUID{
		"under_direct_only":       i5SubtreeDirectOnly,
		"under_ignores_tombstone": i5SubtreeIgnoresTombstone,
	}
	for name := range subtrees {
		if _, ok := i5F2Mutants()[name]; !ok {
			t.Fatalf("%s is not one of the database run's mutants %v", name, i5F2MutantNames())
		}
	}
	for s := range uint64(i5UnderSeeds) {
		w, cases := i5F2Generate(i5DefaultSeed + s)
		for _, name := range []string{"under_direct_only", "under_ignores_tombstone"} {
			if !i5UnderMutantDiffers(w, cases, subtrees[name]) {
				t.Errorf("seed %d: mutant %s admits the same rows as Allows on every case", i5DefaultSeed+s, name)
			}
		}
	}
}

// i5UnderMutantDiffers reports whether binding S from subtree (the mutant
// model of i5F2MutantModels) admits a different row from the reference's
// binding for some case. Each policy is bound once per principal.
func i5UnderMutantDiffers(w *i5World, cases []i5F2Case, subtree func(tenant, root uuid.UUID) []uuid.UUID) bool {
	for ci := range cases {
		c := &cases[ci]
		if c.Policy == nil || len(i5UnderLeaves(c.Policy.Document)) == 0 {
			continue
		}
		for _, p := range c.Principals {
			want, mutant := c.bound(p), i5F2BoundWith(c, p, subtree)
			for _, rec := range w.Records {
				row := rec.row(p)
				_, _, ok := authz.Allows(want, p, authz.OpRead, row)
				if _, _, got := authz.Allows(mutant, p, authz.OpRead, row); got != ok {
					return true
				}
			}
		}
	}
	return false
}

// i5UnderLeaves is every under node of d, in its conditions and policies.
func i5UnderLeaves(d authz.Document) []authz.Under {
	var out []authz.Under
	visit := func(n authz.Node, _ string) {
		if n.Under != nil {
			out = append(out, *n.Under)
		}
	}
	for _, c := range d.Conditions {
		i5Walk(c.Expr, visit)
	}
	for _, np := range d.Policies {
		i5Walk(np.When, visit)
	}
	return out
}

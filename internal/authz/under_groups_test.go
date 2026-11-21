package authz

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

// A binding holds only for the groups it was planned for (TICK-1029). Which
// under leaves fold away depends on p.Groups (member_of, shared), so a
// binding made with groups G1 and read with the same tenant and id but groups
// G2 could read a leaf that survives under G2 as underScope.rest's empty S,
// and under a not that is a grant.
func TestUnderBoundForOtherGroupsVoidsThePolicy(t *testing.T) {
	t.Parallel()
	// G1 = {ga, gb}: member_of gc is false, the under leaf folds away and
	// ExpandUnder binds rest without a query (a nil tx would panic).
	when := `{"all":[{"member_of":{"groups":["` + ptGC().String() + `"]}},{"not":{"under":{"of":"caller"}}}]}`
	pol := ptWhen(t, when)
	pol.EntityType = "ut_under_other_groups"
	g1 := utMe()
	bound, err := ExpandUnder(context.Background(), nil, pol, g1)
	if err != nil {
		t.Fatalf("ExpandUnder for G1: %v", err)
	}
	// G2 = {ga, gb, gc}: the leaf survives, and S was never resolved.
	g2 := utMe()
	g2.Groups = append(g2.Groups, ptGC())
	if !UnderNeeded(pol, g2) {
		t.Fatal("the under leaf must survive under G2 for this test to mean anything")
	}
	planned := plan(bound, g2, OpRead)
	var pe *PlanError
	if len(planned) != 1 || !errors.As(planned[0].err, &pe) || pe.Rule != RuleUnderUnresolved {
		t.Fatalf("plan for G2 = %+v, want voided with %s", planned, RuleUnderUnresolved)
	}
	before := ptInvalidCount(t, pol.EntityType)
	for rn, r := range utRows() {
		if _, _, ok := Allows(bound, g2, OpRead, r); ok {
			t.Errorf("row %q admitted under G2 with a G1 binding", rn)
		}
	}
	if sql, _ := CompileRead(bound, g2, "r", 2); sql != "false" {
		t.Errorf("CompileRead for G2 = %s, want false", sql)
	}
	if got := ptInvalidCount(t, pol.EntityType) - before; got != int64(len(utRows())+1) {
		t.Errorf("authz_policy_invalid_total rose by %d, want %d", got, len(utRows())+1)
	}
}

// The groups fingerprint ignores order and repeats: the same groups listed
// differently are the same principal and keep the binding.
func TestUnderBindingIgnoresGroupOrderAndRepeats(t *testing.T) {
	t.Parallel()
	pol := utBound(t, `{"under":{"of":"caller"}}`) // bound for {ga, gb}
	for name, groups := range map[string][]uuid.UUID{
		"reversed": {ptGB(), ptGA()},
		"repeated": {ptGA(), ptGB(), ptGA()},
	} {
		p := utMe()
		p.Groups = groups
		planned := plan(pol, p, OpRead)
		if len(planned) != 1 || planned[0].err != nil {
			t.Errorf("%s: plan = %+v, want the binding to hold", name, planned)
			continue
		}
		if _, _, ok := Allows(pol, p, OpRead, utRows()["direct unit"]); !ok {
			t.Errorf("%s: a row in S was refused", name)
		}
	}
}

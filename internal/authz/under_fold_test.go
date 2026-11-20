package authz

import (
	"context"
	"testing"
)

// A policy whose under leaf folds away beside a deciding constant (a member_of
// the caller passes, inside an any) must grant through ExpandUnder exactly as
// it would with S bound: the leaf's value cannot matter, so no query runs and
// the policy is not voided (TICK-950 review, I5 seed 20261109 doc #41).
func TestUnderLeafFoldedAwayDoesNotVoidThePolicy(t *testing.T) {
	t.Parallel()
	member := `{"member_of":{"groups":["` + ptGA().String() + `"]}}`
	for _, when := range []string{
		`{"any":[` + member + `,` + utOf(utUnit(1)) + `]}`,
		`{"any":[` + member + `,{"not":{"under":{"of":"caller"}}}]}`,
	} {
		pol := ptWhen(t, when)
		bound, err := ExpandUnder(context.Background(), nil, pol, utMe())
		if err != nil {
			t.Fatalf("%s: ExpandUnder: %v", when, err)
		}
		if _, _, ok := Allows(bound, utMe(), OpRead, Row{State: "open"}); !ok {
			t.Errorf("%s: a member is refused: the folded-away under leaf voided the policy", when)
		}
		if sql, _ := CompileRead(bound, utMe(), "r", 2); sql != "(true)" {
			t.Errorf("%s: CompileRead = %s, want (true)", when, sql)
		}
	}
}

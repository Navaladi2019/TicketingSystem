package store

import (
	"testing"
	"time"
)

// ADR-006-A2 A2-2: the budget of a count is CountBudget, or for a legacy-scan
// list min(CountBudget, Timeout), a zero Timeout meaning DataFilterTimeout.
func TestCountBudgetFor(t *testing.T) {
	for _, tc := range []struct {
		name string
		q    ListQuery
		want time.Duration
	}{
		{"indexed list", ListQuery{BudgetCount: true}, CountBudget},
		{"indexed list ignores Timeout", ListQuery{BudgetCount: true, Timeout: 50 * time.Millisecond}, CountBudget},
		{"legacy, shorter timeout", ListQuery{DataFilter: true, Timeout: 120 * time.Millisecond}, 120 * time.Millisecond},
		{"legacy, longer timeout", ListQuery{DataFilter: true, Timeout: 5 * time.Second}, CountBudget},
		{"legacy, zero timeout", ListQuery{DataFilter: true}, CountBudget},
		{"legacy, equal", ListQuery{DataFilter: true, Timeout: CountBudget}, CountBudget},
	} {
		if got := CountBudgetFor(tc.q); got != tc.want {
			t.Errorf("%s: %v, want %v", tc.name, got, tc.want)
		}
	}
	if CountBudget != 140*time.Millisecond {
		t.Errorf("CountBudget = %v, want 140ms (PLAN-007-E2 D-7E2-1)", CountBudget)
	}
}

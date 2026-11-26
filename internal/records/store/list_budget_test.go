package store

import (
	"testing"
	"time"
)

// ADR-006-A2 A2-2: the budget of a list count is the CountBudget constant.
func TestCountBudget(t *testing.T) {
	if CountBudget != 140*time.Millisecond {
		t.Errorf("CountBudget = %v, want 140ms (PLAN-007-E2 D-7E2-1)", CountBudget)
	}
}

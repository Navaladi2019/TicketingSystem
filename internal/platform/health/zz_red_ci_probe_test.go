package health

import "testing"

// Throwaway: proves CI turns red on a failing unit test (TICK-43 acceptance,
// criterion "CI gates every PR"). This PR is closed unmerged.
func TestRedCIProbeTICK43(t *testing.T) {
	t.Fatal("deliberately failing test: CI must turn red")
}

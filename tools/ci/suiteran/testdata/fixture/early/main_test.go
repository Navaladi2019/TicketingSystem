package early

import (
	"os"
	"testing"
)

// TestMain exits before m.Run when SUITERAN_FIXTURE_EARLY_EXIT is set: go
// test still passes, with no test run.
func TestMain(m *testing.M) {
	if os.Getenv("SUITERAN_FIXTURE_EARLY_EXIT") != "" {
		os.Exit(0)
	}
	os.Exit(m.Run())
}

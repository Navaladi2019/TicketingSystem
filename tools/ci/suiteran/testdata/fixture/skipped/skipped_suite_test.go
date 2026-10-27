package skipped

import (
	"os"
	"testing"
)

// A skip runs nothing, so it does not count as the suite running.
func TestSkippedSuite(t *testing.T) {
	if os.Getenv("SUITERAN_FIXTURE_SKIP") != "" {
		t.Skip("fixture")
	}
}

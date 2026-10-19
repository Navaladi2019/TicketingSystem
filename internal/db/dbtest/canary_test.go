//go:build integration

package dbtest

import (
	"os"
	"testing"
)

// TestCICanary fails on purpose when NAVA_CI_CANARY=fail. It proves that a
// failing integration test turns the pipeline red (T05): the red run is
// linked in the T05 pull request. Unset, it passes and touches nothing.
func TestCICanary(t *testing.T) {
	if os.Getenv("NAVA_CI_CANARY") == "fail" {
		t.Fatal("canary: NAVA_CI_CANARY=fail, so this integration test fails on purpose")
	}
}

package failing

import (
	"os"
	"testing"
)

func TestFails(t *testing.T) {
	if os.Getenv("SUITERAN_FIXTURE_FAIL") != "" {
		t.Fatal("fixture failure")
	}
}

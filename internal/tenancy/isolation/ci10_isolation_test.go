//go:build integration

package isolation

import (
	"os"
	"testing"
)

// TestCI10NamedCases puts the CI-10 cycle and depth-audit cases that make
// test-integration ran (TestCI10_*, NAVA_IT_EVENTS) in the report as named
// rows, and fails when one of them failed, was skipped, did not run, or has
// no row (TICK-483). Run alone (make isolation), the rows are NOT RUN.
func TestCI10NamedCases(t *testing.T) {
	rows, err := loadCI10Rows(os.Getenv("NAVA_IT_EVENTS"))
	if err != nil {
		t.Fatalf("NAVA_IT_EVENTS: %v", err)
	}
	suite.report.ci10 = rows
	for _, r := range rows {
		if r.Status == "FAIL" {
			t.Errorf("CI-10 %s (%s): %s", r.Name, r.Test, r.Detail)
		}
	}
}

// loadCI10Rows reads the go test -json stream at path; "" is no stream.
func loadCI10Rows(path string) ([]ci10Row, error) {
	if path == "" {
		return ci10Rows(nil, false), nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	events, err := readCI10Events(f)
	if err != nil {
		return nil, err
	}
	return ci10Rows(events, true), nil
}

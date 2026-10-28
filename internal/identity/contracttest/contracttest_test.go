package contracttest

import (
	"errors"
	"os"
	"os/exec"
	"slices"
	"strings"
	"testing"
)

// A Suite that requires CI-5 names every catalogue entry it does not cover,
// and nothing once each entry has a case.
func TestUncovered(t *testing.T) {
	s := Suite{Require: []string{"CI-5"}, ResolverCases: []ResolverCase{{Case: CI5UnknownIssuer}}}
	got := s.Uncovered()
	if len(got) != len(CI5Cases())-1 {
		t.Fatalf("uncovered = %d entries, want %d: %v", len(got), len(CI5Cases())-1, got)
	}
	if slices.ContainsFunc(got, func(e string) bool { return e[:len(CI5UnknownIssuer)] == CI5UnknownIssuer }) {
		t.Errorf("covered entry listed: %v", got)
	}

	for _, e := range CI5Cases() {
		s.Scenarios = append(s.Scenarios, ScenarioCase{Case: e.ID})
	}
	if got := s.Uncovered(); len(got) != 0 {
		t.Errorf("fully covered suite lists %v", got)
	}
	if got := (Suite{Require: []string{"CI-99"}}).Uncovered(); len(got) != 1 {
		t.Errorf("unknown catalogue: %v", got)
	}
}

// Catalogue ids are unique and sort in Annex B order, which the report keeps.
func TestCI5CatalogueWellFormed(t *testing.T) {
	seen := map[string]bool{}
	var prev string
	for _, e := range CI5Cases() {
		if e.ID == "" || e.Title == "" || seen[e.ID] || e.ID <= prev {
			t.Errorf("entry %+v: empty, duplicate or out of order", e)
		}
		seen[e.ID], prev = true, e.ID
	}
}

func TestCaseName(t *testing.T) {
	if got := caseName(CI5UnknownIssuer, "x"); got != "[CI-5.03] x" {
		t.Errorf("caseName = %q", got)
	}
	if got := caseName("", "x"); got != "x" {
		t.Errorf("caseName untagged = %q", got)
	}
}

// TestSuiteRunFailsWhenACaseIsMissing runs Suite.Run itself, in a child test
// process, with CI-5.06 left out of an otherwise full suite: the run must
// fail and name the missing case. A failing *testing.T cannot be observed in
// process, hence the re-exec.
func TestSuiteRunFailsWhenACaseIsMissing(t *testing.T) {
	if os.Getenv("CONTRACTTEST_MISSING_CASE_CHILD") == "1" {
		var scenarios []ScenarioCase
		for _, e := range CI5Cases() {
			if e.ID != CI5OrgPickerTamper {
				scenarios = append(scenarios, ScenarioCase{Name: "covered", Case: e.ID, Run: func(*testing.T) {}})
			}
		}
		Suite{Require: []string{"CI-5"}, Scenarios: scenarios}.Run(t)
		return
	}
	cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestSuiteRunFailsWhenACaseIsMissing$", "-test.v")
	cmd.Env = append(os.Environ(), "CONTRACTTEST_MISSING_CASE_CHILD=1")
	out, err := cmd.CombinedOutput()
	var exit *exec.ExitError
	if !errors.As(err, &exit) {
		t.Fatalf("Suite.Run with CI-5.06 missing did not fail (err %v):\n%s", err, out)
	}
	if !strings.Contains(string(out), "no case covers "+CI5OrgPickerTamper) || !strings.Contains(string(out), "--- FAIL: TestSuiteRunFailsWhenACaseIsMissing/coverage") {
		t.Errorf("the failure does not name the missing case:\n%s", out)
	}
}

// TestSuiteRunPassesWhenEveryCaseIsCovered is the control of the test above:
// the same suite with every case passes, so that failure is the gap.
func TestSuiteRunPassesWhenEveryCaseIsCovered(t *testing.T) {
	var scenarios []ScenarioCase
	for _, e := range CI5Cases() {
		scenarios = append(scenarios, ScenarioCase{Name: "covered", Case: e.ID, Run: func(*testing.T) {}})
	}
	Suite{Require: []string{"CI-5"}, Scenarios: scenarios}.Run(t)
}

package contracttest

import (
	"slices"
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

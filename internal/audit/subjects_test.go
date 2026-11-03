package audit

import (
	"slices"
	"testing"
)

func TestSubjectTypes(t *testing.T) {
	t.Parallel()
	got := SubjectTypes()
	if !slices.IsSorted(got) {
		t.Errorf("SubjectTypes() = %v, not sorted", got)
	}
	if len(slices.Compact(slices.Clone(got))) != len(got) {
		t.Errorf("SubjectTypes() = %v, has a duplicate", got)
	}
	for _, s := range got {
		if s == "" || !s.Valid() {
			t.Errorf("registered subject type %q is not Valid", s)
		}
	}
	got[0] = "mutated"
	if SubjectTypes()[0] == "mutated" {
		t.Error("SubjectTypes() returns the registry itself")
	}
	for _, s := range []SubjectType{"", "Record", "record ", "tenant"} {
		if s.Valid() {
			t.Errorf("%q is Valid", s)
		}
	}
}

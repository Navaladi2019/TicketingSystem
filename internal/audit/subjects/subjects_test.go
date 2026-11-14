package subjects

import (
	"slices"
	"testing"
)

func TestAll(t *testing.T) {
	t.Parallel()
	got := All()
	if !slices.IsSorted(got) {
		t.Errorf("All() = %v, not sorted", got)
	}
	if len(slices.Compact(slices.Clone(got))) != len(got) {
		t.Errorf("All() = %v, has a duplicate", got)
	}
	for _, s := range got {
		if s == "" || !s.Valid() {
			t.Errorf("registered subject type %q is not Valid", s)
		}
	}
	got[0] = "mutated"
	if All()[0] == "mutated" {
		t.Error("All() returns the registry itself")
	}
	if !slices.Contains(got, TenantSetting) || TenantSetting != "tenant_setting" {
		t.Errorf("All() = %v, want tenant_setting registered (TICK-854)", got)
	}
	for _, s := range []Type{"", "Record", "record ", "tenant", "tenant_settings"} {
		if s.Valid() {
			t.Errorf("%q is Valid", s)
		}
	}
}

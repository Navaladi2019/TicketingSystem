package catalog

import (
	"slices"
	"strings"
	"testing"
)

// TICK-1070 (PLAN-011-A2 D-11A2-2): binding_lookup's allow-list entry names
// the N-hat_e refresh job's role, and only the three readers it has accepted.
// It is still a SELECT with USING (true) and no WITH CHECK.
func TestTICK1070BindingLookupAllowList(t *testing.T) {
	got, ok := AllowList()["public.tenant_identity_binding.binding_lookup"]
	if !ok {
		t.Fatal("binding_lookup is not on the allow-list")
	}
	want := Policy{Cmd: "SELECT", Roles: "nava_app,nava_groupsync,nava_worker", Qual: "true", Check: ""}
	if got != want {
		t.Errorf("binding_lookup = %+v, want %+v", got, want)
	}
	// pg_policies reports the roles sorted; the catalogue gate compares the
	// string as it is.
	roles := strings.Split(got.Roles, ",")
	if !slices.IsSorted(roles) {
		t.Errorf("binding_lookup roles %q are not sorted", got.Roles)
	}
}

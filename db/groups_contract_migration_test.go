package db

import (
	"io/fs"
	"regexp"
	"strings"
	"testing"
)

// ADR-004-E5 step (ii), CI-E5-8 (TICK-538): (ii-a) sets lock_timeout, swaps
// records_assignment_group_fkey to RESTRICT NOT VALID in one ALTER TABLE,
// revokes DELETE on groups and drops groups_delete, and validates nothing;
// (ii-b) is a file of its own that validates the FK under lock_timeout.
func TestE5GroupsContractMigrationsSplit(t *testing.T) {
	read := func(name string) string {
		t.Helper()
		b, err := fs.ReadFile(Migrations(), name)
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	lockTimeout := regexp.MustCompile(`(?m)^SET LOCAL lock_timeout = '\d+s';\r?$`)
	contract := read("20261104043000_groups_contract.sql")
	validate := read("20261104043100_groups_contract_validate.sql")
	for name, body := range map[string]string{"(ii-a)": contract, "(ii-b)": validate} {
		if !lockTimeout.MatchString(body) {
			t.Errorf("%s migration does not SET LOCAL lock_timeout", name)
		}
	}
	oneAlter := regexp.MustCompile(`(?s)ALTER TABLE records\s+DROP CONSTRAINT records_assignment_group_fkey,\s+ADD CONSTRAINT records_assignment_group_fkey FOREIGN KEY \(tenant_id, assignment_group_id\)\s+REFERENCES groups \(tenant_id, id\) ON DELETE RESTRICT NOT VALID;`)
	if !oneAlter.MatchString(contract) {
		t.Error("(ii-a) does not swap records_assignment_group_fkey to ON DELETE RESTRICT NOT VALID in one ALTER TABLE")
	}
	for _, want := range []string{"REVOKE DELETE ON groups FROM nava_groupsync;", "DROP POLICY groups_delete ON groups;"} {
		if !strings.Contains(contract, want) {
			t.Errorf("(ii-a) lacks %q", want)
		}
	}
	if strings.Contains(contract, "VALIDATE") {
		t.Error("(ii-a) validates a constraint; D10 puts VALIDATE in its own migration")
	}
	if !strings.Contains(validate, "ALTER TABLE records VALIDATE CONSTRAINT records_assignment_group_fkey;") {
		t.Error("(ii-b) does not validate records_assignment_group_fkey")
	}
}

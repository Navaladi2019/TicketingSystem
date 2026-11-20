package main

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
)

// The counter on the documents the S30 kit leaves: the seeded default
// incident ACL in format 1 (4 branches, as the D6 SQL gives) and the S30
// composed document in format 2.
func TestRun(t *testing.T) {
	const (
		admins = "00000000-0000-4000-8000-00000000aaa1"
		sd     = "00000000-0000-4000-8000-00000000aaa2"
	)
	format1 := `[{"id":"default-admins","ops":["read","write"],"who":{"groups":["` + admins + `"]}},` +
		`{"id":"default-service-desk","ops":["read","write"],"who":{"groups":["` + sd + `"]},"where":[{"column":"assignment_group_id","op":"in","caller":"my_groups"}]},` +
		`{"id":"default-requester","ops":["read"],"who":{"requester":true}},` +
		`{"id":"default-requester-new","ops":["read","write"],"who":{"requester":true},"where":[{"column":"state","op":"eq","values":["new"]}]}]`
	format2 := `{"format":2,"conditions":[` +
		`{"name":"in_assigned_group","expr":{"cmp":{"column":"assignment_group_id","op":"in","caller":"my_groups"}}},` +
		`{"name":"is_requester","expr":{"cmp":{"column":"requester_id","op":"eq","caller":"self"}}}],` +
		`"policies":[{"name":"admins","ops":["read","write"],"when":{"member_of":{"groups":["` + admins + `"]}}},` +
		`{"name":"agents_see_group","ops":["read"],"when":{"all":[{"member_of":{"groups":["` + sd + `"]}},{"ref":"in_assigned_group"},{"not":{"ref":"is_requester"}}]}}]}`
	in := "a|incident|1|" + format1 + "\nb|incident|7|" + format2 + "\n"
	var out bytes.Buffer
	if code := run(strings.NewReader(in), &out, authz.ReadBranchThreshold); code != 0 {
		t.Fatalf("exit %d:\n%s", code, out.String())
	}
	for _, want := range []string{"v1   format 1  read branches 4", "v7   format 2  read branches 2", fmt.Sprintf("2 documents (1 format 2) in 2 tenants, max 4 read branches, threshold %d: PASS", authz.ReadBranchThreshold)} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("output lacks %q:\n%s", want, out.String())
		}
	}
	out.Reset()
	if code := run(strings.NewReader(in), &out, 3); code != 1 || !strings.Contains(out.String(), "FAIL") {
		t.Errorf("threshold 3: exit %d, want 1 and FAIL:\n%s", code, out.String())
	}
	out.Reset()
	if code := run(strings.NewReader("a|incident|1|{\"format\":2}\n"), &out, authz.ReadBranchThreshold); code != 1 {
		t.Errorf("an undecodable document: exit %d, want 1:\n%s", code, out.String())
	}
}

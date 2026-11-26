package verify_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/audit/verify"
	vt "github.com/Navaladi2019/TicketingSystem/internal/audit/verify/verifytest"
)

// ADR-004-E10 E10-2 and E10-4, CI-A19 (d), (g): the member is present only as
// true. In JSONL an explicit false or null is malformed_line; in CSV an empty
// cell omits the member, true is the member, and false or any other value is
// malformed_line. The honest file verifies in both encodings.
func TestDiffNULEscapedEncodings(t *testing.T) {
	honest := vt.Build(vt.Options{}).Bytes()
	const member = `"diff_nul_escaped":true`
	if n := bytes.Count(honest, []byte(member)); n != 1 {
		t.Fatalf("the built file has %d lines with the member, want 1 (the E10 vector)", n)
	}
	lines := strings.Split(strings.TrimSuffix(string(honest), "\n"), "\n")
	for name, b := range map[string][]byte{"jsonl": honest, "csv": vt.CSV(lines)} {
		if r := run(t, b); r.Outcome != verify.OutcomePass {
			t.Errorf("%s honest: %s (%s), want pass", name, r.Outcome, describe(r))
		}
	}

	malformed := func(name string, b []byte) {
		t.Helper()
		r := run(t, b)
		if r.Outcome != verify.OutcomeTampered || r.Failure == nil || r.Failure.Reason != verify.ReasonMalformedLine {
			t.Errorf("%s: %s (%s), want tampered with malformed_line", name, r.Outcome, describe(r))
		}
	}
	for _, v := range []string{"false", "null", `"true"`} {
		edited := bytes.Replace(honest, []byte(member), []byte(`"diff_nul_escaped":`+v), 1)
		malformed("jsonl "+v, edited)
	}
	// CSV cells: vt.CSV copies a JSON string's text into the cell, so a
	// string member gives the cell text exactly.
	for _, cell := range []string{"false", "TRUE", "1"} {
		edited := strings.Replace(string(honest), member, `"diff_nul_escaped":"`+cell+`"`, 1)
		malformed("csv "+cell, vt.CSV(strings.Split(strings.TrimSuffix(edited, "\n"), "\n")))
	}
}

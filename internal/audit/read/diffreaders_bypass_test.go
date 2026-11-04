package read_test

import "testing"

// TestDiffReadsLookAlikeBypasses holds Security's look-alike variants of the
// TICK-551 forms (review of PR #458). Postgres 16 runs every query and returns
// the raw diff (each was checked against a temp audit_log table), so the guard
// must report it. Variants D (whole row outside the row functions) and E
// (implicitly repeated consts, %[n] verbs) are known limits in the guard's
// header instead.
func TestDiffReadsLookAlikeBypasses(t *testing.T) {
	cases := map[string]string{
		// Item 1: the table as a parenthesised join after a comma, JOIN or FROM.
		"SELECT a.diff FROM records r, (audit_log a CROSS JOIN x)":                  "diff",
		"SELECT to_jsonb(a) FROM records r JOIN (audit_log a CROSS JOIN x) ON true": "row",
		"SELECT a.diff_salt FROM (audit_log a CROSS JOIN records r)":                "diff_salt",
		// Item 1: whitespace around the schema dot.
		"SELECT a.diff FROM public . audit_log a":                 "diff",
		"SELECT to_jsonb(a) FROM records r, public . audit_log a": "row",
		// Item 5: whitespace around the star's dot.
		"SELECT a . * FROM audit_log a": "star",
		"SELECT a. * FROM audit_log a":  "star",
	}
	for s, want := range cases {
		got := diffReads(s)
		found := false
		for _, g := range got {
			found = found || g == want
		}
		if !found {
			t.Errorf("diffReads(%q) = %v, want it to include %q", s, got, want)
		}
	}
}

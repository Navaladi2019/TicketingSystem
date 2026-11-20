package authz_test

import (
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
)

// TestPrepareFormat2RefusesInvalidUTF8: Prepare reads a format-2 body with
// strictjson, so invalid UTF-8 (bad bytes or a lone surrogate escape) in a
// member name or a value is refused with invalid_utf8 at its pointer, never
// read as U+FFFD (TICK-1019, deferred from TICK-1017). A duplicate name stays
// the duplicate scan's to name, as before.
func TestPrepareFormat2RefusesInvalidUTF8(t *testing.T) {
	t.Parallel()
	body := func(policy string) string {
		return `{"rules":{"format":2,"conditions":[],"policies":[` + policy + `]}}`
	}
	const when = `"when":{"member_of":{"groups":["` + tenantAGroup + `"]}}`
	for _, tc := range []struct {
		name, body, pointer string
	}{
		{"bad bytes in a value", body(`{"name":"` + "\xc1\xbf" + `","ops":["read"],` + when + `}`), "/rules/policies/0/name"},
		{"lone surrogate in a value", body(`{"name":"\udc00","ops":["read"],` + when + `}`), "/rules/policies/0/name"},
		{"bad bytes in a key", body(`{"name":"p","ops":["read"],` + when + `,"` + "\xff" + `":1}`), "/rules/policies/0"},
		{"lone surrogate in a key", body(`{"name":"p","ops":["read"],` + when + `,"\ud800":1}`), "/rules/policies/0"},
		{"bad bytes in a top-level key", `{"rules":{"format":2,"conditions":[],"policies":[]},"` + "\xfe" + `":1}`, ""},
		{"bad bytes in a mask", `{"rules":{"format":2,"conditions":[],"policies":[]},"masks":["` + "\xff" + `"]}`, "/masks/0"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p, ps := authz.Prepare([]byte(tc.body), authz.DefaultMaxDocumentBytes)
			if p != nil {
				t.Fatalf("Prepare accepted a body with invalid UTF-8: rules %s", p.Rules)
			}
			if len(ps) != 1 || ps[0].Pointer != tc.pointer || ps[0].Rule != authz.RuleInvalidUTF8 {
				t.Errorf("problems = %+v, want %q (%s)", ps, tc.pointer, authz.RuleInvalidUTF8)
			}
		})
	}
}

// TestPrepareFormat2InvalidUTF8AfterDuplicate: a body with a duplicate name
// before the invalid bytes is still refused for the invalid UTF-8; the
// duplicate check does not hide it.
func TestPrepareFormat2InvalidUTF8AfterDuplicate(t *testing.T) {
	t.Parallel()
	b := `{"rules":{"format":2,"format":2,"conditions":[],"policies":[{"name":"` + "\xff" + `","ops":["read"],"when":{"member_of":{"groups":[]}}}]}}`
	p, ps := authz.Prepare([]byte(b), authz.DefaultMaxDocumentBytes)
	if p != nil {
		t.Fatalf("Prepare accepted the body")
	}
	if len(ps) != 1 || ps[0].Pointer != "/rules/policies/0/name" || ps[0].Rule != authz.RuleInvalidUTF8 {
		t.Errorf("problems = %+v, want /rules/policies/0/name (%s)", ps, authz.RuleInvalidUTF8)
	}
}

package authz_test

import (
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
)

// TestParsePolicyRefusesNonStrictJSON: the publish body is decoded with
// encoding/json/v2, so a duplicate member name or invalid UTF-8 anywhere is
// one problem with its pointer, never a policy with the last duplicate kept
// (ADR-010, TICK-964).
func TestParsePolicyRefusesNonStrictJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, body, pointer, rule string
	}{
		{"duplicate top-level key", `{"rules":[],"rules":[]}`, "/rules", authz.RuleDuplicate},
		{"duplicate key inside a rule", `{"rules":[{"id":"a","ops":["read"],"who":{"requester":true},"who":{"groups":[]}}]}`, "/rules/0/who", authz.RuleDuplicate},
		{"lone surrogate in an id", `{"rules":[{"id":"\udc00","ops":["read"],"who":{"requester":true}}]}`, "/rules/0/id", authz.RuleInvalidUTF8},
		{"overlong UTF-8 in an id", "{\"rules\":[{\"id\":\"\xc1\xbf\",\"ops\":[\"read\"],\"who\":{\"requester\":true}}]}", "/rules/0/id", authz.RuleInvalidUTF8},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rules, masks, refs, ps := authz.ParsePolicy([]byte(tc.body))
			if rules != nil || masks != nil || refs != nil {
				t.Errorf("a refused body returned a policy part")
			}
			if len(ps) != 1 || ps[0].Pointer != tc.pointer || ps[0].Rule != tc.rule {
				t.Errorf("problems = %+v, want %s (%s)", ps, tc.pointer, tc.rule)
			}
		})
	}
}

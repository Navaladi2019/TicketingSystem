package authz_test

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
)

// Adversarial guard (TICK-961 security review): ParsePolicy builds the
// tenant-scope refs from one reading of the body and the stored rules from a
// second, typed reading. NotInTenant only checks the refs. If the two
// readings ever disagree (a duplicate name where one keeps the first value and
// the other the last, or a case-folded name the walker skips but the typed
// decode accepts), a tenant A admin could publish an ACL naming tenant B's
// group or user without the not_in_tenant check seeing it.
//
// The guard: every body is either refused, or every group/user id in the
// returned rules and masks is also a returned ref.

const (
	tenantAGroup = "0193a000-0000-7000-8000-00000000000a"
	tenantBGroup = "0193b000-0000-7000-8000-00000000000b"
)

func TestParsePolicyCannotSmuggleUncheckedRefs(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ name, body string }{
		{"duplicate groups, B last",
			`{"rules":[{"id":"a","ops":["read"],"who":{"groups":["` + tenantAGroup + `"],"groups":["` + tenantBGroup + `"]}}]}`},
		{"duplicate groups, B first",
			`{"rules":[{"id":"a","ops":["read"],"who":{"groups":["` + tenantBGroup + `"],"groups":["` + tenantAGroup + `"]}}]}`},
		{"case-folded Groups",
			`{"rules":[{"id":"a","ops":["read"],"who":{"Groups":["` + tenantBGroup + `"]}}]}`},
		{"case-folded Users",
			`{"rules":[{"id":"a","ops":["read"],"who":{"Users":["` + tenantBGroup + `"]}}]}`},
		{"case-folded who",
			`{"rules":[{"id":"a","ops":["read"],"who":{"requester":true},"Who":{"groups":["` + tenantBGroup + `"]}}]}`},
		{"case-folded Rules beside rules",
			`{"rules":[],"Rules":[{"id":"a","ops":["read"],"who":{"groups":["` + tenantBGroup + `"]}}]}`},
		{"duplicate mask read",
			`{"rules":[],"masks":[{"field":"f","read":{"groups":["` + tenantAGroup + `"]},"read":{"groups":["` + tenantBGroup + `"]},"write":{}}]}`},
		{"case-folded mask Write",
			`{"rules":[],"masks":[{"field":"f","read":{},"write":{},"Write":{"users":["` + tenantBGroup + `"]}}]}`},
		{"duplicate where values",
			`{"rules":[{"id":"a","ops":["read"],"who":{"requester":true},"where":[{"column":"assignment_group_id","op":"eq","values":["` + tenantAGroup + `"],"values":["` + tenantBGroup + `"]}]}]}`},
		{"escaped duplicate name",
			`{"rules":[{"id":"a","ops":["read"],"who":{"groups":["` + tenantAGroup + `"],"groups":["` + tenantBGroup + `"]}}]}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rules, masks, refs, ps := authz.ParsePolicy([]byte(tc.body))
			if len(ps) > 0 {
				return // refused: nothing is published
			}
			checked := map[uuid.UUID]bool{}
			for _, r := range refs {
				checked[r.ID] = true
			}
			var named []uuid.UUID
			for _, r := range rules {
				named = append(named, r.Who.Groups...)
				named = append(named, r.Who.Users...)
				for _, c := range r.Where {
					for _, v := range c.Values {
						if id, err := uuid.Parse(v); err == nil {
							named = append(named, id)
						}
					}
				}
			}
			for _, m := range masks {
				named = append(named, m.Read.Groups...)
				named = append(named, m.Read.Users...)
				named = append(named, m.Write.Groups...)
				named = append(named, m.Write.Users...)
			}
			for _, id := range named {
				if !checked[id] {
					t.Errorf("accepted policy names %s, which is not a tenant-scope ref (refs %v)", id, refs)
				}
			}
			if slices.Contains(named, uuid.MustParse(tenantBGroup)) {
				t.Errorf("accepted policy carries the smuggled id; problems=%v", ps)
			}
		})
	}
}

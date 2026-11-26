package authz_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
)

// TestPrepareCannotSmuggleUncheckedRefs is TICK-961's guard
// (TestParsePolicyCannotSmuggleUncheckedRefs) at Prepare, for both formats
// (TICK-1017). The route checks only Prepared.Refs against the tenant
// (NotInTenant), and stores Prepared.Rules and Prepared.Masks. If any reading
// of the body (the top-level map, the struct decode, the format-2 walk, the
// mask walk) kept an id another reading dropped, a tenant A admin could store
// tenant B's group without the foreign_id check seeing it.
//
// The invariant is checked on the stored text, not on the parsed structs:
// every string in p.Rules and p.Masks that parses as a UUID is in p.Refs. A
// body is either refused with a named problem, or it holds.
func TestPrepareCannotSmuggleUncheckedRefs(t *testing.T) {
	t.Parallel()
	const (
		a = tenantAGroup
		b = tenantBGroup
	)
	q := func(s string) string { return `"` + s + `"` }
	doc := func(conditions, policies string) string {
		return `{"format":2,"conditions":[` + conditions + `],"policies":[` + policies + `]}`
	}
	pol := func(when string) string { return `{"name":"p","ops":["read"],"when":` + when + `}` }
	when := func(w string) string { return doc("", pol(w)) }
	f2 := func(rules string) string { return `{"rules":` + rules + `}` }
	f2m := func(rules, masks string) string { return `{"rules":` + rules + `,"masks":` + masks + `}` }
	member := func(ids string) string { return `{"member_of":{"groups":[` + ids + `]}}` }
	agEq := func(values string) string {
		return `{"cmp":{"column":"assignment_group_id","op":"in","values":[` + values + `]}}`
	}
	empty := doc("", "")
	mask := func(m string) string { return f2m(empty, `[`+m+`]`) }
	f1 := func(rest string) string { return `{"rules":[{"id":"a","ops":["read"],` + rest + `}]}` }

	for _, tc := range []struct {
		name string
		body string
		// accept marks a control: a body that must be accepted, so the test
		// cannot pass by every body being refused, and so dropping a ref source
		// fails it.
		accept bool
	}{
		// Controls: each ref source of each format, accepted.
		{"control: format 2 member_of", f2(when(member(q(a)))), true},
		{"control: format 2 member_of users", f2(when(`{"member_of":{"users":[` + q(a) + `]}}`)), true},
		{"control: format 2 cmp values", f2(when(agEq(q(a) + "," + q(b)))), true},
		{"control: format 2 cmp requester_id", f2(when(`{"cmp":{"column":"requester_id","op":"in","values":[` + q(a) + `]}}`)), true},
		{"control: format 2 cmp assignee_id", f2(when(`{"cmp":{"column":"assignee_id","op":"in","values":[` + q(a) + `]}}`)), true},
		{"control: format 2 condition expr", f2(doc(`{"name":"c","expr":`+member(q(a))+`}`, pol(`{"ref":"c"}`))), true},
		{"control: format 2 mask", mask(`{"field":"f","read":{"groups":[` + q(a) + `]},"write":{"groups":[` + q(a) + `]}}`), true},
		{"control: format 1 who and where", f1(`"who":{"groups":[` + q(a) + `]},"where":[{"column":"assignment_group_id","op":"eq","values":[` + q(a) + `]}]`), true},
		{"control: format 1 where requester_id", f1(`"who":{"requester":true},"where":[{"column":"requester_id","op":"eq","values":[` + q(a) + `]}]`), true},
		{"control: format 1 mask", `{"rules":[],"masks":[{"field":"f","read":{"users":[` + q(a) + `]},"write":{}}]}`, true},
		{"control: escaped id digits", f2(when(member(`"0193b000-0000-7000-8000-00000000000\u0062"`))), true},

		// groups in member_of.
		{"duplicate groups, B last", f2(when(`{"member_of":{"groups":[` + q(a) + `],"groups":[` + q(b) + `]}}`)), false},
		{"duplicate groups, B first", f2(when(`{"member_of":{"groups":[` + q(b) + `],"groups":[` + q(a) + `]}}`)), false},
		{"case-folded Groups", f2(when(`{"member_of":{"groups":[` + q(a) + `],"Groups":[` + q(b) + `]}}`)), false},
		{"case-folded Users", f2(when(`{"member_of":{"groups":[` + q(a) + `],"Users":[` + q(b) + `]}}`)), false},
		{"escaped duplicate groups", f2(when(`{"member_of":{"groups":[` + q(a) + `],"group\u0073":[` + q(b) + `]}}`)), false},

		// member_of, when, policies, expr.
		{"duplicate member_of", f2(when(`{"member_of":{"groups":[` + q(a) + `]},"member_of":{"groups":[` + q(b) + `]}}`)), false},
		{"case-folded Member_of", f2(when(`{"Member_of":{"groups":[` + q(b) + `]}}`)), false},
		{"duplicate when", f2(doc("", `{"name":"p","ops":["read"],"when":`+member(q(a))+`,"when":`+member(q(b))+`}`)), false},
		{"case-folded When", f2(doc("", `{"name":"p","ops":["read"],"when":`+member(q(a))+`,"When":`+member(q(b))+`}`)), false},
		{"duplicate policies", f2(`{"format":2,"conditions":[],"policies":[` + pol(member(q(a))) + `],"policies":[` + pol(member(q(b))) + `]}`), false},
		{"case-folded Policies", f2(`{"format":2,"conditions":[],"policies":[` + pol(member(q(a))) + `],"Policies":[` + pol(member(q(b))) + `]}`), false},
		{"duplicate expr", f2(doc(`{"name":"c","expr":`+member(q(a))+`,"expr":`+member(q(b))+`}`, pol(`{"ref":"c"}`))), false},
		{"case-folded Expr", f2(doc(`{"name":"c","expr":`+member(q(a))+`,"Expr":`+member(q(b))+`}`, pol(`{"ref":"c"}`))), false},
		{"escaped duplicate when", f2(doc("", `{"name":"p","ops":["read"],"when":`+member(q(a))+`,"whe\u006e":`+member(q(b))+`}`)), false},

		// cmp values.
		{"cmp duplicate values", f2(when(`{"cmp":{"column":"assignment_group_id","op":"in","values":[` + q(a) + `],"values":[` + q(b) + `]}}`)), false},
		{"cmp case-folded Values", f2(when(`{"cmp":{"column":"assignment_group_id","op":"in","values":[` + q(a) + `],"Values":[` + q(b) + `]}}`)), false},
		{"cmp duplicate column", f2(when(`{"cmp":{"column":"state","column":"assignment_group_id","op":"in","values":[` + q(b) + `]}}`)), false},
		// Accepted: a state value spelled like a UUID is a state (storedIDs).
		{"cmp state column with an id-shaped value", f2(when(`{"cmp":{"column":"state","op":"in","values":[` + q(b) + `]}}`)), false},
		{"cmp duplicate column, state last", f2(when(`{"cmp":{"column":"assignment_group_id","column":"state","op":"in","values":[` + q(b) + `]}}`)), false},
		{"cmp caller with values", f2(when(`{"cmp":{"column":"assignment_group_id","op":"in","caller":"my_groups","values":[` + q(b) + `]}}`)), false},

		// under units: a ref source since S31 (TICK-950), so a control.
		{"control: format 2 under units", f2(when(`{"under":{"of":[` + q(a) + `,` + q(b) + `]}}`)), true},
		{"under duplicate of", f2(when(`{"under":{"of":"caller","of":[` + q(b) + `]}}`)), false},

		// masks: read, write.
		{"duplicate mask read", mask(`{"field":"f","read":{"groups":[` + q(a) + `]},"read":{"groups":[` + q(b) + `]},"write":{}}`), false},
		{"case-folded mask Write", mask(`{"field":"f","read":{},"write":{},"Write":{"users":[` + q(b) + `]}}`), false},
		{"case-folded mask Read", mask(`{"field":"f","read":{},"Read":{"groups":[` + q(b) + `]},"write":{}}`), false},

		// Top-level rules and masks.
		{"duplicate rules, B last", `{"rules":` + when(member(q(a))) + `,"rules":` + when(member(q(b))) + `}`, false},
		{"duplicate rules, B first", `{"rules":` + when(member(q(b))) + `,"rules":` + when(member(q(a))) + `}`, false},
		{"duplicate rules, format 1 then format 2", `{"rules":[],"rules":` + when(member(q(b))) + `}`, false},
		{"case-folded Rules", `{"rules":` + when(member(q(a))) + `,"Rules":` + when(member(q(b))) + `}`, false},
		{"case-folded Rules alone", `{"Rules":` + when(member(q(b))) + `}`, false},
		{"escaped duplicate rules", `{"rules":` + when(member(q(a))) + `,"rule\u0073":` + when(member(q(b))) + `}`, false},
		{"escaped duplicate masks", `{"rules":` + empty + `,"masks":[],"mask\u0073":[{"field":"f","read":{"groups":[` + q(b) + `]},"write":{}}]}`, false},
		{"escaped duplicate mask read", mask(`{"field":"f","read":{"groups":[` + q(a) + `]},"r\u0065ad":{"groups":[` + q(b) + `]},"write":{}}`), false},
		{"escaped duplicate cmp values", f2(when(`{"cmp":{"column":"assignment_group_id","op":"in","values":[` + q(a) + `],"v\u0061lues":[` + q(b) + `]}}`)), false},
		{"duplicate masks", `{"rules":` + empty + `,"masks":[],"masks":[{"field":"f","read":{"groups":[` + q(b) + `]},"write":{}}]}`, false},
		{"case-folded Masks", `{"rules":` + empty + `,"masks":[],"Masks":[{"field":"f","read":{"groups":[` + q(b) + `]},"write":{}}]}`, false},
		{"format 1 case-folded Masks", `{"rules":[],"masks":[],"Masks":[{"field":"f","read":{"groups":[` + q(b) + `]},"write":{}}]}`, false},

		// Format 1, for the same property at Prepare.
		{"format 1 duplicate groups", f1(`"who":{"groups":[` + q(a) + `],"groups":[` + q(b) + `]}`), false},
		{"format 1 case-folded Who", f1(`"who":{"requester":true},"Who":{"groups":[` + q(b) + `]}`), false},
		{"format 1 duplicate where values", f1(`"who":{"requester":true},"where":[{"column":"assignment_group_id","op":"eq","values":[` + q(a) + `],"values":[` + q(b) + `]}]`), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p, ps := authz.Prepare([]byte(tc.body), authz.DefaultMaxDocumentBytes)
			if len(ps) > 0 {
				if tc.accept {
					t.Fatalf("control refused: %q", pointersOf(ps))
				}
				for _, pr := range ps {
					if pr.Rule == "" {
						t.Errorf("refused with an unnamed problem at %q", pr.Pointer)
					}
				}
				return // refused: nothing is stored
			}
			if p == nil {
				t.Fatal("no problems and no Prepared")
			}
			checked := map[uuid.UUID]bool{}
			for _, r := range p.Refs {
				checked[r.ID] = true
			}
			stored := storedIDs(t, p.Rules)
			stored = append(stored, storedIDs(t, p.Masks)...)
			if tc.accept && len(stored) == 0 {
				t.Fatalf("control stores no id: rules %s masks %s", p.Rules, p.Masks)
			}
			for _, id := range stored {
				if !checked[id] {
					t.Errorf("stored text names %s, which is not a tenant-scope ref (refs %v)\nrules %s\nmasks %s", id, p.Refs, p.Rules, p.Masks)
				}
			}
		})
	}
}

// storedIDs is every string, member name or value, in the JSON text that
// uuid.Parse accepts: the stored text is read as PostgreSQL will, without the
// Go types. The one exemption is the values of a comparison on the state
// column, which the stored text itself says are states, not principals: a
// state may be spelled like a UUID and is never a tenant reference.
func storedIDs(t *testing.T, text []byte) []uuid.UUID {
	t.Helper()
	var v any
	if err := json.Unmarshal(text, &v); err != nil {
		t.Fatalf("stored text %s is not JSON: %v", text, err)
	}
	var out []uuid.UUID
	var walk func(any)
	walk = func(v any) {
		switch x := v.(type) {
		case map[string]any:
			state := x["column"] == "state"
			for k, e := range x {
				walk(k)
				if state && k == "values" {
					continue
				}
				walk(e)
			}
		case []any:
			for _, e := range x {
				walk(e)
			}
		case string:
			if id, err := uuid.Parse(strings.TrimSpace(x)); err == nil {
				out = append(out, id)
			}
		}
	}
	walk(v)
	return out
}

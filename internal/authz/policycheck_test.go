package authz_test

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
)

const (
	gid1 = "0a000000-0000-4000-8000-0000000004a1"
	gid2 = "0a000000-0000-4000-8000-0000000004a2"
	uid1 = "0a000000-0000-4000-8000-0000000003a1"
)

func problemsOf(t *testing.T, body string) []authz.Problem {
	t.Helper()
	_, _, _, ps := authz.ParsePolicy([]byte(body))
	return ps
}

func rule(who, where string) string {
	if where != "" {
		where = `, "where": [` + where + `]`
	}
	return `{"rules": [{"id": "a", "ops": ["read"], "who": ` + who + where + `}]}`
}

func mask(read, write string) string {
	return `{"rules": [], "masks": [{"field": "f", "read": ` + read + `, "write": ` + write + `}]}`
}

func TestParsePolicyAcceptsTheShapesTheCheckAccepts(t *testing.T) {
	t.Parallel()
	for name, body := range map[string]string{
		"empty rules":        `{"rules": []}`,
		"empty masks":        `{"rules": [], "masks": []}`,
		"empty subject":      rule(`{}`, ""),
		"upper case uuid":    rule(`{"groups": ["0A000000-0000-4000-8000-0000000004A1"]}`, ""),
		"flags":              `{"rules": [{"id": "a", "ops": ["read", "write"], "who": {"requester": true, "assignee": true, "shared": true}}]}`,
		"state in":           rule(`{}`, `{"column": "state", "op": "in", "values": ["open", "new"]}`),
		"state in nothing":   rule(`{}`, `{"column": "state", "op": "in"}`),
		"self":               rule(`{}`, `{"column": "assignee_id", "op": "eq", "caller": "self"}`),
		"my_groups":          rule(`{}`, `{"column": "assignment_group_id", "op": "in", "caller": "my_groups", "values": []}`),
		"uuid literal":       rule(`{}`, `{"column": "requester_id", "op": "eq", "values": ["`+uid1+`"]}`),
		"mask read == write": mask(`{"groups": ["`+gid1+`"]}`, `{"groups": ["`+gid1+`"]}`),
		"mask write < read":  mask(`{"groups": ["`+gid1+`", "`+gid2+`"]}`, `{"groups": ["`+gid1+`"]}`),
		"mask write nobody":  mask(`{"users": ["`+uid1+`"]}`, `{}`),
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			rules, masks, _, ps := authz.ParsePolicy([]byte(body))
			if len(ps) > 0 {
				t.Fatalf("problems = %v, want none", ps)
			}
			if rules == nil || masks == nil {
				t.Fatalf("rules %v and masks %v must be non-nil, so they encode as [] and never null", rules, masks)
			}
		})
	}
}

func TestParsePolicyRefusesWithPointers(t *testing.T) {
	t.Parallel()
	p := func(ptr, rule string) authz.Problem { return authz.Problem{Pointer: ptr, Rule: rule} }
	for _, tc := range []struct {
		name, body string
		want       authz.Problem
	}{
		{"not an object", `[]`, p("", "type")},
		{"not json", `{`, p("", "type")},
		{"trailing value", `{"rules": []} {}`, p("", "type")},
		{"null body", `null`, p("", "type")},
		{"no rules", `{}`, p("/rules", "required")},
		{"rules null", `{"rules": null}`, p("/rules", "type")},
		{"unknown top key", `{"rules": [], "version": 3}`, p("/version", "unknown_key")},
		{"masks not an array", `{"rules": [], "masks": {}}`, p("/masks", "type")},
		{"rule not an object", `{"rules": [1]}`, p("/rules/0", "type")},
		{"unknown rule key", `{"rules": [{"id": "a", "ops": ["read"], "who": {}, "deny": []}]}`, p("/rules/0/deny", "unknown_key")},
		{"no id", `{"rules": [{"ops": ["read"], "who": {}}]}`, p("/rules/0/id", "required")},
		{"empty id", `{"rules": [{"id": "", "ops": ["read"], "who": {}}]}`, p("/rules/0/id", "empty")},
		{"id not a string", `{"rules": [{"id": 4, "ops": ["read"], "who": {}}]}`, p("/rules/0/id", "type")},
		{"duplicate id", `{"rules": [{"id": "a", "ops": ["read"], "who": {}}, {"id": "a", "ops": ["read"], "who": {}}]}`, p("/rules/1/id", "duplicate")},
		{"no ops", `{"rules": [{"id": "a", "who": {}}]}`, p("/rules/0/ops", "required")},
		{"ops null", `{"rules": [{"id": "a", "ops": null, "who": {}}]}`, p("/rules/0/ops", "type")},
		{"empty ops", `{"rules": [{"id": "a", "ops": [], "who": {}}]}`, p("/rules/0/ops", "empty")},
		{"unknown op", `{"rules": [{"id": "a", "ops": ["read", "delete"], "who": {}}]}`, p("/rules/0/ops/1", "value")},
		{"no who", `{"rules": [{"id": "a", "ops": ["read"]}]}`, p("/rules/0/who", "required")},
		{"who null", `{"rules": [{"id": "a", "ops": ["read"], "who": null}]}`, p("/rules/0/who", "type")},
		{"unknown who key", rule(`{"roles": []}`, ""), p("/rules/0/who/roles", "unknown_key")},
		{"group not a uuid", rule(`{"groups": ["admins"]}`, ""), p("/rules/0/who/groups/0", "value")},
		{"group with braces", rule(`{"groups": ["{`+gid1+`}"]}`, ""), p("/rules/0/who/groups/0", "value")},
		{"group urn form", rule(`{"groups": ["urn:uuid:`+gid1+`"]}`, ""), p("/rules/0/who/groups/0", "value")},
		{"user number", rule(`{"users": [1]}`, ""), p("/rules/0/who/users/0", "type")},
		{"flag not a bool", rule(`{"shared": 1}`, ""), p("/rules/0/who/shared", "type")},
		{"flag null", rule(`{"requester": null}`, ""), p("/rules/0/who/requester", "type")},
		{"unknown column", rule(`{}`, `{"column": "data", "op": "eq", "values": ["x"]}`), p("/rules/0/where/0/column", "value")},
		{"unknown cond op", rule(`{}`, `{"column": "state", "op": "ne", "values": ["x"]}`), p("/rules/0/where/0/op", "value")},
		{"unknown caller", rule(`{}`, `{"column": "assignee_id", "op": "eq", "caller": "boss"}`), p("/rules/0/where/0/caller", "value")},
		{"caller null", rule(`{}`, `{"column": "assignee_id", "op": "eq", "caller": null}`), p("/rules/0/where/0/caller", "type")},
		{"no column", rule(`{}`, `{"op": "eq"}`), p("/rules/0/where/0/column", "required")},
		{"eq with two values", rule(`{}`, `{"column": "state", "op": "eq", "values": ["a", "b"]}`), p("/rules/0/where/0/values", "condition")},
		{"eq with none", rule(`{}`, `{"column": "state", "op": "eq"}`), p("/rules/0/where/0/values", "condition")},
		{"value not a string", rule(`{}`, `{"column": "state", "op": "in", "values": [1]}`), p("/rules/0/where/0/values/0", "type")},
		{"uuid column, non-uuid value", rule(`{}`, `{"column": "assignee_id", "op": "in", "values": ["bob"]}`), p("/rules/0/where/0/values/0", "value")},
		{"self on state", rule(`{}`, `{"column": "state", "op": "eq", "caller": "self"}`), p("/rules/0/where/0/caller", "condition")},
		{"self on assignment group", rule(`{}`, `{"column": "assignment_group_id", "op": "eq", "caller": "self"}`), p("/rules/0/where/0/caller", "condition")},
		{"my_groups on assignee", rule(`{}`, `{"column": "assignee_id", "op": "in", "caller": "my_groups"}`), p("/rules/0/where/0/caller", "condition")},
		{"self with values", rule(`{}`, `{"column": "assignee_id", "op": "eq", "caller": "self", "values": ["`+uid1+`"]}`), p("/rules/0/where/0/caller", "condition")},
		{"mask without field", `{"rules": [], "masks": [{"read": {}, "write": {}}]}`, p("/masks/0/field", "required")},
		{"mask empty field", `{"rules": [], "masks": [{"field": "", "read": {}, "write": {}}]}`, p("/masks/0/field", "empty")},
		{"mask without read", `{"rules": [], "masks": [{"field": "f", "write": {}}]}`, p("/masks/0/read", "required")},
		{"mask without write", `{"rules": [], "masks": [{"field": "f", "read": {}}]}`, p("/masks/0/write", "required")},
		{"mask unknown key", `{"rules": [], "masks": [{"field": "f", "read": {}, "write": {}, "hide": true}]}`, p("/masks/0/hide", "unknown_key")},
		// A2: a mask is row-independent.
		{"mask read requester", mask(`{"requester": true}`, `{}`), p("/masks/0/read/requester", "mask_subject")},
		{"mask read assignee", mask(`{"assignee": true}`, `{}`), p("/masks/0/read/assignee", "mask_subject")},
		{"mask read shared", mask(`{"shared": true}`, `{}`), p("/masks/0/read/shared", "mask_subject")},
		{"mask write requester", mask(`{}`, `{"requester": true}`), p("/masks/0/write/requester", "mask_subject")},
		{"mask write assignee", mask(`{}`, `{"assignee": true}`), p("/masks/0/write/assignee", "mask_subject")},
		{"mask write shared", mask(`{}`, `{"shared": true}`), p("/masks/0/write/shared", "mask_subject")},
		// R3: write wider than read.
		{"mask write group not in read", mask(`{"groups": ["`+gid1+`"]}`, `{"groups": ["`+gid1+`", "`+gid2+`"]}`), p("/masks/0/write/groups/1", "mask_write")},
		{"mask write user not in read", mask(`{}`, `{"users": ["`+uid1+`"]}`), p("/masks/0/write/users/0", "mask_write")},
		{"mask write group is a read user", mask(`{"users": ["`+gid1+`"]}`, `{"groups": ["`+gid1+`"]}`), p("/masks/0/write/groups/0", "mask_write")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ps := problemsOf(t, tc.body)
			if !slices.Contains(ps, tc.want) {
				t.Fatalf("problems = %v, want to contain %v", ps, tc.want)
			}
		})
	}
}

func TestParsePolicyRefusalsReturnNoPolicy(t *testing.T) {
	t.Parallel()
	rules, masks, refs, ps := authz.ParsePolicy([]byte(`{"rules": [{"id": "a", "ops": ["read"], "who": {"groups": ["` + gid1 + `"]}}, {"id": "a", "ops": [], "who": {}}]}`))
	if len(ps) == 0 || rules != nil || masks != nil || refs != nil {
		t.Fatalf("rules %v masks %v refs %v problems %v: a refused body must return no policy", rules, masks, refs, ps)
	}
}

func TestParsePolicyReportsEveryProblem(t *testing.T) {
	t.Parallel()
	ps := problemsOf(t, `{"rules": [{"id": "a", "ops": ["x"], "who": {"groups": ["nope"]}}, {"id": "a", "ops": ["read"], "who": {}}]}`)
	for _, want := range []authz.Problem{{Pointer: "/rules/0/ops/0", Rule: "value"}, {Pointer: "/rules/0/who/groups/0", Rule: "value"}, {Pointer: "/rules/1/id", Rule: "duplicate"}} {
		if !slices.Contains(ps, want) {
			t.Errorf("problems %v lack %v", ps, want)
		}
	}
}

func TestParsePolicyEscapesAnUnknownKeyInItsPointer(t *testing.T) {
	t.Parallel()
	ps := problemsOf(t, `{"rules": [], "a/b~c": 1}`)
	if want := (authz.Problem{Pointer: "/a~1b~0c", Rule: "unknown_key"}); !slices.Contains(ps, want) {
		t.Fatalf("problems = %v, want %v", ps, want)
	}
}

func TestParsePolicyCollectsEveryIdAsATenantRef(t *testing.T) {
	t.Parallel()
	body := `{"rules": [{"id": "a", "ops": ["read"], "who": {"groups": ["` + gid1 + `"], "users": ["` + uid1 + `"]},
		"where": [{"column": "assignment_group_id", "op": "in", "values": ["` + gid2 + `"]},
		          {"column": "requester_id", "op": "eq", "values": ["` + uid1 + `"]},
		          {"column": "state", "op": "in", "values": ["` + gid1 + `"]}]}],
		"masks": [{"field": "f", "read": {"groups": ["` + gid1 + `"]}, "write": {"groups": ["` + gid1 + `"]}}]}`
	_, _, refs, ps := authz.ParsePolicy([]byte(body))
	if len(ps) > 0 {
		t.Fatalf("problems = %v", ps)
	}
	got := map[string]authz.TenantRef{}
	for _, r := range refs {
		got[r.Pointer] = r
	}
	for ptr, want := range map[string]authz.Kind{
		"/rules/0/who/groups/0":     authz.KindGroup,
		"/rules/0/who/users/0":      authz.KindUser,
		"/rules/0/where/0/values/0": authz.KindGroup, // assignment_group_id
		"/rules/0/where/1/values/0": authz.KindUser,  // requester_id
		"/masks/0/read/groups/0":    authz.KindGroup,
		"/masks/0/write/groups/0":   authz.KindGroup,
	} {
		r, ok := got[ptr]
		if !ok || r.Kind != want {
			t.Errorf("ref at %s = %+v (found %v), want kind %v", ptr, r, ok, want)
		}
	}
	if _, ok := got["/rules/0/where/2/values/0"]; ok {
		t.Error("a state value is a literal, not an id, and must not be a tenant ref")
	}
	if got["/rules/0/who/groups/0"].ID != uuid.MustParse(gid1) {
		t.Error("ref id is not the parsed uuid")
	}
}

func TestCheckMaskFields(t *testing.T) {
	t.Parallel()
	masks := []authz.Mask{{Field: "title"}, {Field: "salary"}}
	ps := authz.CheckMaskFields(masks, []string{"title", "body"})
	if want := []authz.Problem{{Pointer: "/masks/1/field", Rule: "field"}}; !slices.Equal(ps, want) {
		t.Fatalf("problems = %v, want %v", ps, want)
	}
	if ps := authz.CheckMaskFields(nil, nil); len(ps) != 0 {
		t.Fatalf("no masks: %v", ps)
	}
}

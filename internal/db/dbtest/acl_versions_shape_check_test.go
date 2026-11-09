//go:build integration

package dbtest

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/authz/acltest"
)

// TICK-629: acl_versions.rules and masks have the shape internal/authz
// decodes and the S10 contract (A1) defines. Versions are immutable, so the
// insert is the only place to stop a policy that would make every read of
// its type a 500.

// insertACLShape: tenant $1 (also the type and publisher), version $2,
// rules $3 and masks $4 as text cast to jsonb.
const insertACLShape = `INSERT INTO acl_versions (tenant_id, entity_definition_id, version, rules, masks, published_by)
	VALUES ($1, $1, $2, $3::text::jsonb, $4::text::jsonb, $1)`

var aclShapeFunctions = []string{
	"acl_uuids_valid(jsonb)", "acl_subject_valid(jsonb)", "acl_cond_valid(jsonb)",
	"acl_rules_valid(jsonb)", "acl_masks_valid(jsonb)",
}

const (
	shapeU1 = "0a000000-0000-4000-8000-0000000003a1"
	shapeU2 = "0A000000-0000-4000-8000-0000000004A1" // upper case: uuid.Parse accepts it
)

// aclRuleShapes are rule lists the CHECK accepts (ok) or refuses.
var aclRuleShapes = []struct {
	name  string
	rules string
	ok    bool
}{
	{"empty policy", `[]`, true},
	{"every field", `[{"id": "r1", "ops": ["read", "write"],
		"who": {"groups": ["` + shapeU2 + `"], "users": ["` + shapeU1 + `"], "requester": true, "assignee": false, "shared": true},
		"where": [{"column": "state", "op": "in", "values": ["open", "closed"]},
		          {"column": "assignee_id", "op": "eq", "values": ["` + shapeU1 + `"]},
		          {"column": "requester_id", "op": "eq", "caller": "self"},
		          {"column": "assignee_id", "op": "in", "caller": "self", "values": []},
		          {"column": "assignment_group_id", "op": "in", "caller": "my_groups"},
		          {"column": "assignment_group_id", "op": "in", "caller": "", "values": []}]}]`, true},
	{"empty subject and where", `[{"id": "r", "ops": ["read"], "who": {}, "where": []}]`, true},
	{"empty subject arrays", `[{"id": "r", "ops": ["write"], "who": {"users": [], "groups": []}}]`, true},

	{"rules an object", `{}`, false},
	{"rules a string", `"x"`, false},
	{"rules null", `null`, false},
	{"rule a number", `[1]`, false},
	{"rule null", `[null]`, false},
	{"unknown rule key", `[{"id": "x", "ops": ["read"], "who": {}, "colour": "red"}]`, false},
	{"no id", `[{"ops": ["read"], "who": {}}]`, false},
	{"empty id", `[{"id": "", "ops": ["read"], "who": {}}]`, false},
	{"numeric id", `[{"id": 1, "ops": ["read"], "who": {}}]`, false},
	{"no ops", `[{"id": "r", "who": {}}]`, false},
	{"null ops", `[{"id": "r", "ops": null, "who": {}}]`, false},
	{"empty ops", `[{"id": "r", "ops": [], "who": {}}]`, false},
	{"op delete", `[{"id": "r", "ops": ["read", "delete"], "who": {}}]`, false},
	{"ops a string", `[{"id": "r", "ops": "read", "who": {}}]`, false},
	{"ops nested", `[{"id": "r", "ops": [["read"]], "who": {}}]`, false},
	{"no who", `[{"id": "r", "ops": ["read"]}]`, false},
	{"null who", `[{"id": "r", "ops": ["read"], "who": null}]`, false},
	{"who an array", `[{"id": "r", "ops": ["read"], "who": []}]`, false},
	{"unknown subject key", `[{"id": "r", "ops": ["read"], "who": {"roles": ["admin"]}}]`, false},
	{"groups an object", `[{"id": "r", "ops": ["read"], "who": {"groups": {}}}]`, false},
	{"group not a uuid", `[{"id": "r", "ops": ["read"], "who": {"groups": ["agents"]}}]`, false},
	{"group uuid without hyphens", `[{"id": "r", "ops": ["read"], "who": {"groups": ["0a000000000040008000000000000a01"]}}]`, false},
	{"user a number", `[{"id": "r", "ops": ["read"], "who": {"users": [1]}}]`, false},
	{"null users", `[{"id": "r", "ops": ["read"], "who": {"users": null}}]`, false},
	{"requester a string", `[{"id": "r", "ops": ["read"], "who": {"requester": "yes"}}]`, false},
	{"null shared", `[{"id": "r", "ops": ["read"], "who": {"shared": null}}]`, false},
	{"where an object", `[{"id": "r", "ops": ["read"], "who": {}, "where": {}}]`, false},
	{"null where", `[{"id": "r", "ops": ["read"], "who": {}, "where": null}]`, false},
	{"cond a number", `[{"id": "r", "ops": ["read"], "who": {}, "where": [1]}]`, false},
	{"unknown cond key", `[{"id": "r", "ops": ["read"], "who": {}, "where": [{"column": "state", "op": "eq", "values": ["open"], "not": true}]}]`, false},
	{"data column", `[{"id": "r", "ops": ["read"], "who": {}, "where": [{"column": "data.priority", "op": "eq", "values": ["1"]}]}]`, false},
	{"no column", `[{"id": "r", "ops": ["read"], "who": {}, "where": [{"op": "eq", "values": ["open"]}]}]`, false},
	{"op ne", `[{"id": "r", "ops": ["read"], "who": {}, "where": [{"column": "state", "op": "ne", "values": ["open"]}]}]`, false},
	{"no op", `[{"id": "r", "ops": ["read"], "who": {}, "where": [{"column": "state", "values": ["open"]}]}]`, false},
	{"eq with no value", `[{"id": "r", "ops": ["read"], "who": {}, "where": [{"column": "state", "op": "eq"}]}]`, false},
	{"eq with two values", `[{"id": "r", "ops": ["read"], "who": {}, "where": [{"column": "state", "op": "eq", "values": ["a", "b"]}]}]`, false},
	{"uuid column non-uuid value", `[{"id": "r", "ops": ["read"], "who": {}, "where": [{"column": "requester_id", "op": "in", "values": ["bob"]}]}]`, false},
	{"state value a number", `[{"id": "r", "ops": ["read"], "who": {}, "where": [{"column": "state", "op": "in", "values": [1]}]}]`, false},
	{"null values", `[{"id": "r", "ops": ["read"], "who": {}, "where": [{"column": "state", "op": "in", "values": null}]}]`, false},
	{"values a string", `[{"id": "r", "ops": ["read"], "who": {}, "where": [{"column": "state", "op": "eq", "values": "open"}]}]`, false},
	{"caller boss", `[{"id": "r", "ops": ["read"], "who": {}, "where": [{"column": "requester_id", "op": "eq", "caller": "boss"}]}]`, false},
	{"self on state", `[{"id": "r", "ops": ["read"], "who": {}, "where": [{"column": "state", "op": "eq", "caller": "self"}]}]`, false},
	{"self on group", `[{"id": "r", "ops": ["read"], "who": {}, "where": [{"column": "assignment_group_id", "op": "eq", "caller": "self"}]}]`, false},
	{"self with values", `[{"id": "r", "ops": ["read"], "who": {}, "where": [{"column": "requester_id", "op": "eq", "caller": "self", "values": ["` + shapeU1 + `"]}]}]`, false},
	{"my_groups on assignee", `[{"id": "r", "ops": ["read"], "who": {}, "where": [{"column": "assignee_id", "op": "in", "caller": "my_groups"}]}]`, false},
	{"null caller", `[{"id": "r", "ops": ["read"], "who": {}, "where": [{"column": "state", "op": "eq", "values": ["open"], "caller": null}]}]`, false},
	{"caller a number", `[{"id": "r", "ops": ["read"], "who": {}, "where": [{"column": "state", "op": "eq", "values": ["open"], "caller": 1}]}]`, false},
	{"one bad rule among good", `[{"id": "a", "ops": ["read"], "who": {}}, {"id": "b", "ops": ["read"], "who": {}, "x": 1}]`, false},
}

// aclMaskShapes are mask lists the CHECK accepts (ok) or refuses.
var aclMaskShapes = []struct {
	name  string
	masks string
	ok    bool
}{
	{"no masks", `[]`, true},
	{"a mask", `[{"field": "salary", "read": {"groups": ["` + shapeU2 + `"]}, "write": {}}]`, true},

	{"masks an object", `{"fields": []}`, false},
	{"masks null", `null`, false},
	{"mask a string", `["salary"]`, false},
	{"empty mask", `[{}]`, false},
	{"no field", `[{"read": {}, "write": {}}]`, false},
	{"empty field", `[{"field": "", "read": {}, "write": {}}]`, false},
	{"unknown mask key", `[{"field": "f", "read": {}, "write": {}, "hide": true}]`, false},
	{"no read", `[{"field": "f", "write": {}}]`, false},
	{"null write", `[{"field": "f", "read": {}, "write": null}]`, false},
	{"bad read subject", `[{"field": "f", "read": {"users": ["bob"]}, "write": {}}]`, false},
}

func TestTICK629ACLShapeCheck(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA}, nil)
	seedACL(t, super, tenantA)
	app := asRole(t, "nava_app")

	// Both constraints exist and are validated, not left NOT VALID.
	if got := queryString(t, super, `SELECT string_agg(conname || ':' || convalidated, ',' ORDER BY conname)
		FROM pg_constraint WHERE conrelid = 'acl_versions'::regclass
		AND conname IN ('acl_versions_rules_check', 'acl_versions_masks_check')`); got !=
		"acl_versions_masks_check:true,acl_versions_rules_check:true" {
		t.Errorf("constraints = %q; want both present and validated", got)
	}

	version := 1
	try := func(what, rules, masks string, ok bool, constraint string) {
		t.Helper()
		version++
		_, err := execIn(app, tenantA, insertACLShape, tenantA, version, rules, masks)
		if ok {
			if err != nil {
				t.Errorf("%s: %v; want accepted", what, err)
			}
			return
		}
		wantSQLState(t, what, err, "23514")
		var pe *pgconn.PgError
		if errors.As(err, &pe) && pe.ConstraintName != constraint {
			t.Errorf("%s: refused by %s, want %s", what, pe.ConstraintName, constraint)
		}
	}
	for _, c := range aclRuleShapes {
		try("rules "+c.name, c.rules, `[]`, c.ok, "acl_versions_rules_check")
		// Every list the CHECK accepts is one PolicyStore.Current decodes.
		if c.ok {
			var rules []authz.Rule
			if err := decodeStrictACL(c.rules, &rules); err != nil {
				t.Errorf("rules %s: accepted but does not decode: %v", c.name, err)
			}
		}
	}
	for _, c := range aclMaskShapes {
		try("masks "+c.name, `[]`, c.masks, c.ok, "acl_versions_masks_check")
		if c.ok {
			var masks []authz.Mask
			if err := decodeStrictACL(c.masks, &masks); err != nil {
				t.Errorf("masks %s: accepted but does not decode: %v", c.name, err)
			}
		}
	}

	// What the Go side writes is accepted: acltest's rules and a Rule and a
	// Mask with every field set, through encoding/json.
	u := uuid.MustParse(shapeU1)
	full := []authz.Rule{
		acltest.AllowUsers(u), acltest.AllowGroups(u),
		{ID: "full", Ops: []string{authz.OpRead, authz.OpWrite},
			Who: authz.Subject{Groups: []uuid.UUID{u}, Users: []uuid.UUID{u}, Requester: true, Assignee: true, Shared: true},
			Where: []authz.Cond{
				{Column: "state", Op: authz.CondIn, Values: []string{"open"}},
				{Column: "requester_id", Op: authz.CondEq, Caller: authz.CallerSelf},
				{Column: "assignment_group_id", Op: authz.CondIn, Caller: authz.CallerMyGroups},
				{Column: "assignee_id", Op: authz.CondEq, Values: []string{u.String()}},
			}},
	}
	masks := []authz.Mask{{Field: "salary", Read: authz.Subject{Groups: []uuid.UUID{u}}, Write: authz.Subject{}}}
	try("Go-encoded policy", mustJSON(t, full), mustJSON(t, masks), true, "")

	// The superuser too: a seed or a hand-written repair is refused alike.
	_, err := execIn(super, "", insertACLShape, tenantA, 1000, `{}`, `[]`)
	wantSQLState(t, "superuser inserts rules '{}'", err, "23514")

	// The functions are immutable, so the CHECK is a pure function of the row.
	if got := queryString(t, super, `SELECT string_agg(proname || ':' || provolatile::text, ',' ORDER BY proname)
		FROM pg_proc WHERE proname LIKE 'acl\_%\_valid'`); got !=
		"acl_cond_valid:i,acl_masks_valid:i,acl_rules_valid:i,acl_subject_valid:i,acl_uuids_valid:i" {
		t.Errorf("functions = %s; want all five immutable", got)
	}
}

// Only the two roles that insert acl_versions may EXECUTE the functions its
// CHECKs call; PUBLIC and every other role may not.
func TestTICK629ACLShapeFunctionGrants(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	roles := queryString(t, super, `SELECT string_agg(rolname, ',' ORDER BY rolname) FROM pg_roles
		WHERE NOT rolsuper AND rolname NOT LIKE 'pg\_%' AND rolname <> 'nava_migrator'`)
	for _, fn := range aclShapeFunctions {
		if queryBool(t, super, `SELECT EXISTS (SELECT 1 FROM pg_proc p, aclexplode(coalesce(p.proacl, acldefault('f', p.proowner))) a
			WHERE p.oid = $1::regprocedure AND a.grantee = 0)`, "public."+fn) {
			t.Errorf("PUBLIC may EXECUTE %s", fn)
		}
		for _, role := range strings.Split(roles, ",") {
			want := role == "nava_app" || role == "nava_worker"
			if got := queryBool(t, super, "SELECT has_function_privilege($1, $2, 'EXECUTE')", role, "public."+fn); got != want {
				t.Errorf("%s EXECUTE on %s = %v, want %v", role, fn, got, want)
			}
		}
	}
}

// decodeStrictACL decodes as authz.PolicyStore does: unknown fields refused.
func decodeStrictACL(s string, v any) error {
	d := json.NewDecoder(bytes.NewReader([]byte(s)))
	d.DisallowUnknownFields()
	return d.Decode(v)
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

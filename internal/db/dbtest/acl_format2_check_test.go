//go:build integration

package dbtest

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Navaladi2019/TicketingSystem/internal/db/grants"
)

// TICK-940 (S30 P11-08; ADR-010 D4, amendment A1): the database backstop for
// format-2 access documents. acl_versions_rules_check refuses every
// structural problem of the shared fixture the route refuses, with the C1
// code and pointer in DETAIL; the tenant-id trigger refuses foreign ids; the
// byte ceilings measure octet_length(<col>::text) exactly; only nava_app
// publishes.

const (
	format2Migration         = "20261119100000_acl_rules_valid_format2.sql"
	format2ValidateMigration = "20261119100100_acl_versions_bytes_validate.sql"
	// bytesCeiling is ADR-010-A1 A1-2b's constant.
	bytesCeiling = 131072
)

// beforeFormat2Checks returns m without the acl2_* function grants (TICK-940)
// when fsys lacks the migration that creates them, so a run that stops at an
// earlier version applies only grants on functions it has created.
func beforeFormat2Checks(fsys fs.FS, m *grants.Manifest) *grants.Manifest {
	if _, err := fs.Stat(fsys, format2Migration); err == nil {
		return m
	}
	gm := *m
	gm.Roles = make(map[string]grants.Role, len(m.Roles))
	for name, r := range m.Roles {
		functions := make(map[string][]string, len(r.Functions))
		for fn, privs := range r.Functions {
			if !strings.HasPrefix(fn, "public.acl2_") {
				functions[fn] = privs
			}
		}
		r.Functions = functions
		gm.Roles[name] = r
	}
	return &gm
}

// aclDetail is the DETAIL of a refusal: {"column", "code", "pointer"}.
type aclDetail struct {
	Column  string `json:"column"`
	Code    string `json:"code"`
	Pointer string `json:"pointer"`
}

type format2Fixture struct {
	Name   string `json:"name"`
	GoOnly bool   `json:"go_only"`
	// RouteOnly marks a case only the publish route refuses (not_available, ids of
	// the tenant, the configured byte limit): the database accepts it or refuses it
	// elsewhere.
	RouteOnly bool            `json:"route_only"`
	Doc       json.RawMessage `json:"doc"`
	Problems  []struct {
		Pointer string `json:"pointer"`
		Code    string `json:"code"`
	} `json:"problems"`
}

func readFormat2Fixtures(t *testing.T, name string) []format2Fixture {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "authz", "testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	var out []format2Fixture
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

// seedACLIDs makes every group, user and org-unit id in rules and masks a live
// group, principal or org unit of tenant, so a valid document passes the
// tenant-id trigger.
func seedACLIDs(t *testing.T, super *pgx.Conn, tenant string, values ...string) {
	t.Helper()
	for _, v := range values {
		exec(t, super, `INSERT INTO groups (tenant_id, id, keycloak_group_id, path)
			SELECT DISTINCT $1::uuid, id::uuid, 'kc-' || id, '/t/' || $1 || '/' || id
			FROM acl2_id_pointers($2::text::jsonb) WHERE kind = 'group' ON CONFLICT DO NOTHING`, tenant, v)
		exec(t, super, `INSERT INTO principals (tenant_id, id, keycloak_user_id, enabled, synced_at)
			SELECT DISTINCT $1::uuid, id::uuid, 'kc-' || id, true, now()
			FROM acl2_id_pointers($2::text::jsonb) WHERE kind = 'user' ON CONFLICT DO NOTHING`, tenant, v)
		// TICK-947: under ids are live org units of the tenant.
		exec(t, super, `INSERT INTO org_units (tenant_id, id, keycloak_group_id, name)
			SELECT DISTINCT $1::uuid, id::uuid, 'kc-' || id, 'u'
			FROM acl2_id_pointers($2::text::jsonb) WHERE kind = 'org_unit' ON CONFLICT DO NOTHING`, tenant, v)
	}
}

// refusal is the 23514 of err, or fails the test.
func refusal(t *testing.T, what string, err error) *pgconn.PgError {
	t.Helper()
	var pe *pgconn.PgError
	if !errors.As(err, &pe) || pe.Code != "23514" {
		t.Errorf("%s: %v; want SQLSTATE 23514", what, err)
		return nil
	}
	return pe
}

func detailOf(t *testing.T, what string, pe *pgconn.PgError) aclDetail {
	t.Helper()
	var d aclDetail
	if err := json.Unmarshal([]byte(pe.Detail), &d); err != nil {
		t.Errorf("%s: DETAIL %q is not the JSON problem: %v", what, pe.Detail, err)
	}
	return d
}

func TestTICK940Format2Fixtures(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA}, nil)
	seedACL(t, super, tenantA)
	app := asRole(t, "nava_app")

	version := 1
	insert := func(conn *pgx.Conn, tenant, rules, masks string) error {
		version++
		_, err := execIn(conn, tenant, insertACLShape, tenantA, version, rules, masks)
		return err
	}

	// Every structural invalid fixture is refused, by nava_app and by the
	// superuser (no RLS, no Go), with one of the fixture's problems in DETAIL.
	// go_only cases (duplicate keys) are invisible to jsonb.
	n := 0
	for _, f := range readFormat2Fixtures(t, "format2_invalid.json") {
		if f.GoOnly || f.RouteOnly {
			continue
		}
		n++
		for _, as := range []struct {
			name   string
			conn   *pgx.Conn
			tenant string
		}{{"nava_app", app, tenantA}, {"superuser", super, ""}} {
			what := as.name + " inserts invalid fixture " + f.Name
			pe := refusal(t, what, insert(as.conn, as.tenant, string(f.Doc), `[]`))
			if pe == nil {
				continue
			}
			if pe.ConstraintName != "acl_versions_rules_check" {
				t.Errorf("%s: constraint %q, want acl_versions_rules_check", what, pe.ConstraintName)
			}
			d := detailOf(t, what, pe)
			ok := d.Column == "rules"
			found := false
			for _, p := range f.Problems {
				found = found || (p.Code == d.Code && p.Pointer == d.Pointer)
			}
			if !ok || !found {
				t.Errorf("%s: DETAIL %+v; want column rules and one of %+v", what, d, f.Problems)
			}
		}
	}
	if n < 50 {
		t.Fatalf("only %d structural invalid fixtures; the fixture file was not read", n)
	}

	// Every valid fixture and every format-1 fixture inserts.
	for _, f := range readFormat2Fixtures(t, "format2_valid.json") {
		seedACLIDs(t, super, tenantA, string(f.Doc))
		if err := insert(app, tenantA, string(f.Doc), `[]`); err != nil {
			t.Errorf("valid fixture %s: %v", f.Name, err)
		}
	}
	format1, err := filepath.Glob(filepath.Join("..", "..", "authz", "testdata", "format1", "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	f1 := 0
	for _, p := range format1 {
		if strings.HasSuffix(p, ".golden.json") {
			continue
		}
		f1++
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		seedACLIDs(t, super, tenantA, string(b))
		if err := insert(app, tenantA, string(b), `[]`); err != nil {
			t.Errorf("format-1 fixture %s: %v", filepath.Base(p), err)
		}
	}
	if f1 == 0 {
		t.Fatal("no format-1 fixtures read")
	}
}

// leafs is n {"shared": true} nodes.
func leafs(n int) []any {
	out := make([]any, n)
	for i := range out {
		out[i] = map[string]any{"shared": true}
	}
	return out
}

// nots wraps e in k not nodes: depth + k.
func nots(e any, k int) any {
	for range k {
		e = map[string]any{"not": e}
	}
	return e
}

func any16(n int) any { return map[string]any{"any": leafs(n)} }

func policy(name string, when any) map[string]any {
	return map[string]any{"name": name, "ops": []string{"read"}, "when": when}
}

func doc2(conds, pols []any) string {
	if conds == nil {
		conds = []any{}
	}
	b, _ := json.Marshal(map[string]any{"format": 2, "conditions": conds, "policies": pols})
	return string(b)
}

// atCaps64 is 64 policies with 128 leaves in all, one of them a 16-child any
// at depth 8: every document-wide cap at once. extraLeaf adds one leaf to the
// last policy, extraPolicy one more policy.
func atCaps64(extraLeaf, extraPolicy bool) string {
	pols := []any{policy("p0", nots(any16(16), 6))} // 16 leaves, depth 8
	for i := 1; i <= 49; i++ {
		pols = append(pols, policy(fmt.Sprintf("p%d", i), any16(2))) // 98 leaves
	}
	for i := 50; i <= 63; i++ {
		pols = append(pols, policy(fmt.Sprintf("p%d", i), map[string]any{"shared": true})) // 14 leaves
	}
	if extraLeaf {
		pols[63] = policy("p63", any16(2))
	}
	if extraPolicy {
		pols[63] = policy("p63", map[string]any{"ref": "c"})
		pols = append(pols, policy("p64", map[string]any{"ref": "c"}))
		return doc2([]any{map[string]any{"name": "c", "expr": map[string]any{"shared": true}}}, pols)
	}
	return doc2(nil, pols)
}

// atCaps128 is one policy of 128 leaves at depth 8 (all of 8 any-of-16).
func atCaps128(leaves int, depth int) string {
	kids := []any{}
	for range 8 {
		kids = append(kids, any16(16))
	}
	inner := 3 // all, any, leaf
	if leaves > 128 {
		kids[7] = map[string]any{"any": append(leafs(15), map[string]any{"all": leafs(leaves - 127)})}
		inner = 4
	}
	return doc2(nil, []any{policy("p", nots(map[string]any{"all": kids}, depth-inner))})
}

func TestTICK940Caps(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA}, nil)
	seedACL(t, super, tenantA)
	app := asRole(t, "nava_app")

	version := 1
	insert := func(rules string) error {
		version++
		_, err := execIn(app, tenantA, insertACLShape, tenantA, version, rules, `[]`)
		return err
	}
	for _, c := range []struct{ name, rules string }{
		{"64 policies, 128 leaves, 16 children, depth 8", atCaps64(false, false)},
		{"one policy of 128 leaves at depth 8", atCaps128(128, 8)},
		{"depth 8 through a ref", doc2(
			[]any{map[string]any{"name": "deep", "expr": nots(map[string]any{"shared": true}, 4)}}, // depth 5
			[]any{policy("p", nots(map[string]any{"ref": "deep"}, 3))})},                           // 3 + 5
		{"128 leaves through refs", doc2(
			[]any{map[string]any{"name": "c16", "expr": any16(16)}},
			[]any{policy("p", map[string]any{"all": []any{
				map[string]any{"ref": "c16"}, map[string]any{"ref": "c16"}, map[string]any{"ref": "c16"}, map[string]any{"ref": "c16"},
				map[string]any{"ref": "c16"}, map[string]any{"ref": "c16"}, map[string]any{"ref": "c16"}, map[string]any{"ref": "c16"}}})})},
		{"an unused deep condition is allowed", doc2(
			[]any{map[string]any{"name": "unused", "expr": nots(map[string]any{"shared": true}, 20)}},
			[]any{policy("p", map[string]any{"shared": true})})},
	} {
		if err := insert(c.rules); err != nil {
			t.Errorf("at the caps, %s: %v", c.name, err)
		}
	}

	// A doubling chain: 60 conditions, each any of two refs to the previous.
	chain := []any{map[string]any{"name": "c0", "expr": map[string]any{"shared": true}}}
	for i := 1; i < 60; i++ {
		prev := map[string]any{"ref": fmt.Sprintf("c%d", i-1)}
		chain = append(chain, map[string]any{"name": fmt.Sprintf("c%d", i), "expr": map[string]any{"any": []any{prev, prev}}})
	}

	for _, c := range []struct{ name, rules, code, pointer string }{
		{"65 policies", atCaps64(false, true), "policies_exceeded", "/policies"},
		{"129 leaves in the document", atCaps64(true, false), "document_leaves_exceeded", "/policies"},
		{"129 leaves in one policy", atCaps128(129, 8), "policy_leaves_exceeded", "/policies/0/when"},
		{"depth 9", atCaps128(128, 9), "depth_exceeded", "/policies/0/when"},
		{"depth 9 through a ref", doc2(
			[]any{map[string]any{"name": "deep", "expr": nots(map[string]any{"shared": true}, 5)}},
			[]any{policy("p", nots(map[string]any{"ref": "deep"}, 3))}), "depth_exceeded", "/policies/0/when"},
		{"129 leaves through refs", doc2(
			[]any{map[string]any{"name": "c16", "expr": any16(16)}},
			[]any{policy("p", map[string]any{"all": []any{
				map[string]any{"ref": "c16"}, map[string]any{"ref": "c16"}, map[string]any{"ref": "c16"}, map[string]any{"ref": "c16"},
				map[string]any{"ref": "c16"}, map[string]any{"ref": "c16"}, map[string]any{"ref": "c16"}, map[string]any{"ref": "c16"},
				map[string]any{"shared": true}}})}), "policy_leaves_exceeded", "/policies/0/when"},
		{"17 children", doc2(nil, []any{policy("p", any16(17))}), "child_count", "/policies/0/when/any"},
		{"a doubling chain of 60", doc2(chain, []any{policy("p", map[string]any{"ref": "c59"})}), "depth_exceeded", "/policies/0/when"},
		{"a missing ref", doc2(nil, []any{policy("p", map[string]any{"ref": "nope"})}), "unresolved_ref", "/policies/0/when/ref"},
		{"a missing ref in a condition", doc2(
			[]any{map[string]any{"name": "a", "expr": map[string]any{"not": map[string]any{"ref": "nope"}}}}, []any{}),
			"unresolved_ref", "/conditions/0/expr/not/ref"},
		{"a self-reference", doc2(
			[]any{map[string]any{"name": "a", "expr": map[string]any{"ref": "a"}}}, []any{}), "ref_cycle", "/conditions/0/expr/ref"},
		{"an a-b-a cycle", doc2([]any{
			map[string]any{"name": "a", "expr": map[string]any{"ref": "b"}},
			map[string]any{"name": "b", "expr": map[string]any{"all": []any{map[string]any{"shared": true}, map[string]any{"ref": "a"}}}}},
			[]any{}), "ref_cycle", "/conditions/0/expr/ref"},
	} {
		pe := refusal(t, c.name, insert(c.rules))
		if pe == nil {
			continue
		}
		if d := detailOf(t, c.name, pe); d != (aclDetail{"rules", c.code, c.pointer}) {
			t.Errorf("%s: DETAIL %+v, want {rules %s %s}", c.name, d, c.code, c.pointer)
		}
	}
}

// TICK-940 D4 / D-11-11: a direct insert naming another tenant's group or
// user is refused by the tenant-id trigger, by the superuser (who bypasses
// RLS, so the trigger's explicit tenant filter is what holds) and by
// nava_app.
func TestTICK940ForeignIDs(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, nil)
	// seedPipeline gives each tenant a principal and a group whose id is the
	// tenant id.
	seedACL(t, super, tenantA, tenantB)
	app := asRole(t, "nava_app")

	memberOf := func(key, id string) string {
		return doc2(nil, []any{
			policy("own", map[string]any{"member_of": map[string]any{key: []string{tenantA}}}),
			policy("other", map[string]any{"member_of": map[string]any{"groups": []string{tenantA}, key: []string{tenantA, id}}})})
	}
	version := 10
	for _, c := range []struct {
		name, rules, masks string
		want               aclDetail
	}{
		{"tenant B's group in member_of", memberOf("groups", tenantB), `[]`,
			aclDetail{"rules", "foreign_id", "/policies/1/when/member_of/groups/1"}},
		{"tenant B's user in member_of", memberOf("users", tenantB), `[]`,
			aclDetail{"rules", "foreign_id", "/policies/1/when/member_of/users/1"}},
		{"tenant B's user in a cmp", doc2(nil, []any{policy("p", map[string]any{"cmp": map[string]any{
			"column": "assignee_id", "op": "in", "values": []string{tenantA, tenantB}}})}), `[]`,
			aclDetail{"rules", "foreign_id", "/policies/0/when/cmp/values/1"}},
		{"tenant B's group in a format-1 who", `[{"id": "r", "ops": ["read"], "who": {"groups": ["` + tenantB + `"]}}]`, `[]`,
			aclDetail{"rules", "foreign_id", "/0/who/groups/0"}},
		{"tenant B's user in a mask", `[]`, `[{"field": "f", "read": {"users": ["` + tenantB + `"]}, "write": {}}]`,
			aclDetail{"masks", "foreign_id", "/0/read/users/0"}},
		{"an id of no tenant", memberOf("users", "0c000000-0000-4000-8000-0000000003c9"), `[]`,
			aclDetail{"rules", "foreign_id", "/policies/1/when/member_of/users/1"}},
	} {
		for _, as := range []struct {
			name   string
			conn   *pgx.Conn
			tenant string
		}{{"superuser", super, ""}, {"nava_app", app, tenantA}} {
			version++
			what := as.name + ": " + c.name
			_, err := execIn(as.conn, as.tenant, insertACLShape, tenantA, version, c.rules, c.masks)
			if pe := refusal(t, what, err); pe != nil {
				if d := detailOf(t, what, pe); d != c.want {
					t.Errorf("%s: DETAIL %+v, want %+v", what, d, c.want)
				}
			}
		}
	}
	// A tombstoned own group is refused like a foreign one.
	const gone = "0a000000-0000-4000-8000-0000000005a9"
	exec(t, super, `INSERT INTO groups (tenant_id, id, keycloak_group_id, path, deleted_at)
		VALUES ($1::uuid, $2::uuid, 'kc-gone', '/t/' || $1::text || '/gone', now())`, tenantA, gone)
	_, err := execIn(app, tenantA, insertACLShape, tenantA, 99, doc2(nil, []any{
		policy("p", map[string]any{"member_of": map[string]any{"groups": []string{gone}}})}), `[]`)
	if pe := refusal(t, "tombstoned group", err); pe != nil {
		if d := detailOf(t, "tombstoned group", pe); d.Code != "foreign_id" {
			t.Errorf("tombstoned group: DETAIL %+v, want foreign_id", d)
		}
	}
	// The own tenant's live ids pass.
	_, err = execIn(app, tenantA, insertACLShape, tenantA, 100, doc2(nil, []any{
		policy("p", map[string]any{"member_of": map[string]any{"users": []string{tenantA}}})}),
		`[{"field": "f", "read": {"users": ["`+tenantA+`"]}, "write": {}}]`)
	if err != nil {
		t.Errorf("own tenant's user: %v", err)
	}
}

// padded returns a value of exactly size bytes of jsonb text: a format-2
// document (col rules) or a masks list (col masks), padded with one string.
func padded(t *testing.T, super *pgx.Conn, col string, size int) string {
	t.Helper()
	mk := func(pad int) string {
		if col == "masks" {
			return `[{"field": "` + strings.Repeat("f", pad+1) + `", "read": {}, "write": {}}]`
		}
		return doc2([]any{map[string]any{"name": "pad", "expr": map[string]any{"cmp": map[string]any{
			"column": "state", "op": "eq", "values": []string{strings.Repeat("a", pad)}}}}},
			[]any{policy("p", map[string]any{"shared": true})})
	}
	var base int
	if err := super.QueryRow(t.Context(), "SELECT octet_length($1::text::jsonb::text)", mk(0)).Scan(&base); err != nil {
		t.Fatal(err)
	}
	v := mk(size - base)
	var got int
	if err := super.QueryRow(t.Context(), "SELECT octet_length($1::text::jsonb::text)", v).Scan(&got); err != nil || got != size {
		t.Fatalf("padded %s to %d bytes: got %d, %v", col, size, got, err)
	}
	return v
}

// ADR-010-A1 A1-2b: the ceilings measure octet_length(<col>::text) exactly.
func TestTICK940ByteCeilings(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA}, nil)
	seedACL(t, super, tenantA)
	app := asRole(t, "nava_app")

	if got := queryString(t, super, `SELECT string_agg(conname || ':' || convalidated || ':' || pg_get_constraintdef(oid), ' | ' ORDER BY conname)
		FROM pg_constraint WHERE conrelid = 'acl_versions'::regclass AND conname LIKE 'acl\_versions\_bytes\_%'`); got !=
		"acl_versions_bytes_masks_check:true:CHECK ((octet_length((masks)::text) <= 131072)) | "+
			"acl_versions_bytes_rules_check:true:CHECK (((NOT (rules ? 'format'::text)) OR (octet_length((rules)::text) <= 131072)))" {
		t.Errorf("byte constraints = %s", got)
	}

	version := 1
	for _, col := range []string{"rules", "masks"} {
		for _, size := range []int{bytesCeiling, bytesCeiling + 1} {
			rules, masks := doc2(nil, []any{}), `[]`
			if col == "rules" {
				rules = padded(t, super, col, size)
			} else {
				masks = padded(t, super, col, size)
			}
			version++
			_, err := execIn(app, tenantA, insertACLShape, tenantA, version, rules, masks)
			what := fmt.Sprintf("%s of %d bytes", col, size)
			if size == bytesCeiling {
				if err != nil {
					t.Errorf("%s: %v; want inserted", what, err)
				}
				continue
			}
			if pe := refusal(t, what, err); pe != nil && pe.ConstraintName != "acl_versions_bytes_"+col+"_check" {
				t.Errorf("%s: refused by %q, want acl_versions_bytes_%s_check", what, pe.ConstraintName, col)
			}
		}
	}
	// A format-1 rules list is not capped by bytes (the route never stores
	// format 1): only acl_versions_bytes_rules_check's NOT (rules ? 'format').
	version++
	big := `[{"id": "` + strings.Repeat("r", bytesCeiling) + `", "ops": ["read"], "who": {}}]`
	if _, err := execIn(app, tenantA, insertACLShape, tenantA, version, big, `[]`); err != nil {
		t.Errorf("format-1 rules above the ceiling: %v; want inserted", err)
	}
}

// ADR-010-A1 A1-3: nava_worker cannot publish, even a valid format-2
// document for its own tenant's type, and may EXECUTE no acl% function.
func TestTICK940WorkerCannotPublish(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA}, nil)
	seedACL(t, super, tenantA)

	valid := doc2(nil, []any{policy("p", map[string]any{"shared": true})})
	_, err := execIn(asRole(t, "nava_worker"), tenantA, insertACLShape, tenantA, 2, valid, `[]`)
	if got := sqlState(err); got != "42501" {
		t.Errorf("nava_worker publishes a valid document in its own tenant: %v; want exactly 42501", err)
	}
	if _, err := execIn(asRole(t, "nava_app"), tenantA, insertACLShape, tenantA, 2, valid, `[]`); err != nil {
		t.Errorf("nava_app publishes the same document: %v", err)
	}

	fns := strings.Split(queryString(t, super, `SELECT string_agg(p.oid::regprocedure::text, ';' ORDER BY 1)
		FROM pg_proc p JOIN pg_namespace n ON n.oid = p.pronamespace
		WHERE n.nspname = 'public' AND p.proname LIKE 'acl%'`), ";")
	if len(fns) < 18 {
		t.Fatalf("acl%% functions = %v; want TICK-629's five and the acl2_* ones", fns)
	}
	for _, fn := range fns {
		if queryBool(t, super, "SELECT has_function_privilege('nava_worker', $1::regprocedure, 'EXECUTE')", fn) {
			t.Errorf("nava_worker may EXECUTE %s", fn)
		}
	}
	// The insert policy names nava_app alone.
	if got := queryString(t, super, `SELECT array_to_string(ARRAY(SELECT r::regrole::text FROM unnest(polroles) r ORDER BY 1), ',')
		FROM pg_policy WHERE polrelid = 'acl_versions'::regclass AND polname = 'acl_versions_insert'`); got != "nava_app" {
		t.Errorf("acl_versions_insert roles = %s, want nava_app", got)
	}
}

// The two migrations run over stored format-1 rows: every row still passes
// the replaced acl_rules_valid, and the byte ceilings validate.
func TestTICK940MigrationOverFormat1Rows(t *testing.T) {
	m := freshCluster(t)
	if err := migrateOnlyFS(t, m, migrationsBefore(t, format2Migration)); err != nil {
		t.Fatalf("migrate to before %s: %v", format2Migration, err)
	}
	super := superNava(t)
	seedTenants(t, super, []string{tenantA}, nil)
	seedACL(t, super, tenantA)
	format1, err := filepath.Glob(filepath.Join("..", "..", "authz", "testdata", "format1", "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	version := 1
	for _, p := range format1 {
		if strings.HasSuffix(p, ".golden.json") {
			continue
		}
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		version++
		exec(t, super, insertACLShape, tenantA, version, string(b),
			`[{"field": "f", "read": {"groups": ["`+tenantA+`"]}, "write": {}}]`)
	}
	if _, err := migrateFS(t, m, migrationsThrough(t, format2Migration)); err != nil {
		t.Fatalf("migrate %s: %v", format2Migration, err)
	}
	if got := queryString(t, super, `SELECT string_agg(conname || ':' || convalidated, ',' ORDER BY conname)
		FROM pg_constraint WHERE conrelid = 'acl_versions'::regclass AND conname LIKE 'acl\_versions\_bytes\_%'`); got !=
		"acl_versions_bytes_masks_check:false,acl_versions_bytes_rules_check:false" {
		t.Errorf("after the first migration: %s; want both NOT VALID", got)
	}
	if !queryBool(t, super, "SELECT bool_and(acl_rules_valid(rules) AND acl_masks_valid(masks)) FROM acl_versions") {
		t.Error("a stored format-1 row fails the replaced functions")
	}
	if n := queryString(t, super, "SELECT count(*)::text FROM acl_versions"); n == "1" || n == "0" {
		t.Fatalf("only %s rows seeded", n)
	}
	migrateAll(t, m)
	if got := queryString(t, super, `SELECT string_agg(conname || ':' || convalidated, ',' ORDER BY conname)
		FROM pg_constraint WHERE conrelid = 'acl_versions'::regclass AND contype = 'c'`); got !=
		"acl_versions_bytes_masks_check:true,acl_versions_bytes_rules_check:true,acl_versions_masks_check:true,"+
			"acl_versions_rules_check:true,acl_versions_version_check:true" {
		t.Errorf("after both migrations: %s; want all validated", got)
	}
}

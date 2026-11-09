//go:build integration

// P3-35 (TICK-620): the access routes, through the real handler against real
// PostgreSQL: read and publish an entity type's ACL, and the explainer.
// Helpers are in entitytypes_integration_test.go, records_integration_test.go
// and records_patch_integration_test.go.

package server_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/authz/acltest"
	"github.com/Navaladi2019/TicketingSystem/internal/entitytypes"
	"github.com/Navaladi2019/TicketingSystem/internal/metadata"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

const accessPath = "/api/v1/entity-types/job/access"

// access sends a request to an access route. An empty ifMatch or idemKey
// sends no such header.
func (e *etEnv) access(t *testing.T, method, path, token, idemKey, ifMatch, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), method, path, strings.NewReader(body))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if idemKey != "" {
		req.Header.Set("Idempotency-Key", idemKey)
	}
	if ifMatch != "" {
		req.Header.Set("If-Match", ifMatch)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	e.h.ServeHTTP(rec, req)
	return rec
}

type accessBody struct {
	Version     int              `json:"version"`
	Rules       []map[string]any `json:"rules"`
	Masks       []map[string]any `json:"masks"`
	PublishedBy *string          `json:"published_by"`
	PublishedAt string           `json:"published_at"`
}

func decodeAccess(t *testing.T, rec *httptest.ResponseRecorder) accessBody {
	t.Helper()
	var b accessBody
	if err := json.Unmarshal(rec.Body.Bytes(), &b); err != nil {
		t.Fatalf("not an access policy: %v: %s", err, rec.Body)
	}
	return b
}

// aclCounts is the number of ACL versions and of config audit entries on
// entity definitions in tenant a.
func (e *etEnv) aclCounts(t *testing.T) (versions, entries int) {
	t.Helper()
	if err := e.super.QueryRow(t.Context(), "SELECT count(*) FROM acl_versions WHERE tenant_id = $1", e.a.id).Scan(&versions); err != nil {
		t.Fatal(err)
	}
	if err := e.super.QueryRow(t.Context(),
		"SELECT count(*) FROM audit_log WHERE tenant_id = $1 AND domain = 'config' AND subject_type = 'acl_policy'", e.a.id).Scan(&entries); err != nil {
		t.Fatal(err)
	}
	return versions, entries
}

func uuidList(ps ...uuid.UUID) string {
	out := make([]string, len(ps))
	for i, p := range ps {
		out[i] = `"` + p.String() + `"`
	}
	return strings.Join(out, ", ")
}

// pointerRules is "pointer rule" for each error of a problem response.
func pointerRules(t *testing.T, rec *httptest.ResponseRecorder) []string {
	t.Helper()
	var p struct {
		Errors []struct {
			Pointer string `json:"pointer"`
			Rule    string `json:"rule"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
		t.Fatalf("not a problem: %v: %s", err, rec.Body)
	}
	out := make([]string, len(p.Errors))
	for i, e := range p.Errors {
		out[i] = e.Pointer + " " + e.Rule
	}
	return out
}

func containsStr(vs []string, v string) bool {
	for _, x := range vs {
		if x == v {
			return true
		}
	}
	return false
}

// covers: getEntityTypeAccess 200 403 404
func TestGetAccessReadsTheNewestPolicyForAnAdminOnly(t *testing.T) {
	e := newPatchEnv(t)

	rec := e.access(t, http.MethodGet, accessPath, "admin-a", "", "", "")
	if rec.Code != http.StatusOK || rec.Header().Get("ETag") != `W/"1"` {
		t.Fatalf("admin GET = %d ETag %q: %s", rec.Code, rec.Header().Get("ETag"), rec.Body)
	}
	b := decodeAccess(t, rec)
	if b.Version != 1 || len(b.Rules) != 1 || b.Rules[0]["id"] != "test-allow-users" || b.Masks == nil || len(b.Masks) != 0 || b.PublishedBy != nil || b.PublishedAt == "" {
		t.Fatalf("policy = %+v, want version 1 with the test rule, no masks, no publisher", b)
	}

	// Tenant B's admin reads tenant B's policy: it names none of A's principals.
	recB := e.access(t, http.MethodGet, accessPath, "admin-b", "", "", "")
	if recB.Code != http.StatusOK || strings.Contains(recB.Body.String(), e.a.admin.PrincipalID.String()) || !strings.Contains(recB.Body.String(), e.b.admin.PrincipalID.String()) {
		t.Fatalf("tenant B GET = %d: %s", recB.Code, recB.Body)
	}

	// A user that is not an admin, and the lookalike of the admins group.
	for _, tok := range []string{"user-a", "lookalike-a"} {
		wantProblem(t, e.access(t, http.MethodGet, accessPath, tok, "", "", ""), http.StatusForbidden, problem.Forbidden)
	}

	// A type with no published version, an unknown type and an over-long key
	// give the one 404.
	e.noTestACL = true
	if r := e.do(t, http.MethodPost, "/api/v1/entity-types", "admin-a", "widget-a", etBody("widget", "WGT")); r.Code != http.StatusCreated {
		t.Fatalf("create widget = %d: %s", r.Code, r.Body)
	}
	none := e.access(t, http.MethodGet, "/api/v1/entity-types/widget/access", "admin-a", "", "", "")
	unknown := e.access(t, http.MethodGet, "/api/v1/entity-types/nothing/access", "admin-a", "", "", "")
	long := e.access(t, http.MethodGet, "/api/v1/entity-types/"+strings.Repeat("k", 80)+"/access", "admin-a", "", "", "")
	for name, r := range map[string]*httptest.ResponseRecorder{"none published": none, "unknown": unknown, "over-long": long} {
		wantProblem(t, r, http.StatusNotFound, problem.NotFound)
		if r.Body.String() != unknown.Body.String() {
			t.Errorf("404 for %s differs from the unknown key's: %s vs %s", name, r.Body, unknown.Body)
		}
	}
	// Another tenant's key: tenant B has no widget.
	if r := e.access(t, http.MethodGet, "/api/v1/entity-types/widget/access", "admin-b", "", "", ""); r.Body.String() != unknown.Body.String() || r.Code != http.StatusNotFound {
		t.Errorf("tenant B reading tenant A's type = %d: %s", r.Code, r.Body)
	}
}

// covers: putEntityTypeAccess 200 412 428
func TestPutAccessPublishesTheNextVersionWithItsAuditEntry(t *testing.T) {
	e := newPatchEnv(t)
	e.noTestACL = true
	if r := e.do(t, http.MethodPost, "/api/v1/entity-types", "admin-a", "widget-a", etBody("widget", "WGT")); r.Code != http.StatusCreated {
		t.Fatalf("create widget = %d: %s", r.Code, r.Body)
	}
	const path = "/api/v1/entity-types/widget/access"
	body := `{"rules": [{"id": "r1", "ops": ["read", "write"], "who": {"users": [` + uuidList(e.a.admin.PrincipalID, e.a.user.PrincipalID) + `], "groups": [` + uuidList(e.a.adminGroup) + `]},
		"where": [{"column": "state", "op": "in", "values": ["new"]}]}],
		"masks": [{"field": "note", "read": {"groups": [` + uuidList(e.a.adminGroup) + `]}, "write": {"groups": [` + uuidList(e.a.adminGroup) + `]}}]}`
	versions0, entries0 := e.aclCounts(t)

	// The first publish is based on W/"0": 200, version 1, its ETag.
	rec := e.access(t, http.MethodPut, path, "admin-a", "put-1", `W/"0"`, body)
	if rec.Code != http.StatusOK || rec.Header().Get("ETag") != `W/"1"` {
		t.Fatalf("first publish = %d ETag %q: %s", rec.Code, rec.Header().Get("ETag"), rec.Body)
	}
	b := decodeAccess(t, rec)
	if b.Version != 1 || b.PublishedBy == nil || *b.PublishedBy != e.a.admin.PrincipalID.String() || len(b.Rules) != 1 || len(b.Masks) != 1 {
		t.Fatalf("published = %+v, want version 1 by the admin with one rule and one mask", b)
	}
	// GET returns what was published.
	if got := e.access(t, http.MethodGet, path, "admin-a", "", "", ""); got.Code != http.StatusOK || decodeAccess(t, got).Version != 1 || got.Header().Get("ETag") != `W/"1"` {
		t.Fatalf("GET after publish = %d: %s", got.Code, got.Body)
	}
	versions, entries := e.aclCounts(t)
	if versions != versions0+1 || entries != entries0+1 {
		t.Fatalf("acl versions %d -> %d, audit entries %d -> %d, want one of each", versions0, versions, entries0, entries)
	}
	var (
		action, source string
		before         *int64
		after          int64
		sum            []byte
		diff           *string
		actor          uuid.UUID
		subject, def   uuid.UUID
	)
	if err := e.super.QueryRow(t.Context(), `SELECT a.action, a.source, a.version_before, a.version_after, a.content_sha256, a.diff::text, a.actor_principal_id, a.subject_id, d.id
		FROM audit_log a JOIN entity_definitions d ON d.tenant_id = a.tenant_id AND d.key = 'widget'
		WHERE a.tenant_id = $1 AND a.domain = 'config' AND a.subject_type = 'acl_policy' AND a.subject_id = d.id AND a.version_after = 1 AND a.action = 'publish'`,
		e.a.id).Scan(&action, &source, &before, &after, &sum, &diff, &actor, &subject, &def); err != nil {
		t.Fatalf("the publish has no audit entry: %v", err)
	}
	if source != "user" || before != nil || len(sum) != 32 || diff == nil || actor != e.a.admin.PrincipalID || subject != def {
		t.Errorf("audit entry: source %q before %v hash %d bytes diff %v actor %v, want a user entry for the admin with a hash and a diff", source, before, len(sum), diff, actor)
	}

	// Version 2 needs W/"1". A weak or strong form, and a list, are read the same.
	rec = e.access(t, http.MethodPut, path, "admin-a", "put-2", `"1"`, `{"rules": []}`)
	if rec.Code != http.StatusOK || decodeAccess(t, rec).Version != 2 || rec.Header().Get("ETag") != `W/"2"` {
		t.Fatalf("second publish = %d: %s", rec.Code, rec.Body)
	}
	if e.access(t, http.MethodGet, path, "admin-a", "", "", "").Header().Get("ETag") != `W/"2"` {
		t.Error("GET does not return the newest version")
	}

	// 428: no If-Match, or a star. 412: a stale one, with the current ETag.
	versions, entries = e.aclCounts(t)
	for name, im := range map[string]string{"missing": "", "star": "*"} {
		wantProblem(t, e.access(t, http.MethodPut, path, "admin-a", "nomatch-"+name, im, `{"rules": []}`), http.StatusPreconditionRequired, problem.IfMatchRequired)
	}
	for i, stale := range []string{`W/"1"`, `W/"0"`, `W/"3"`, `banana`, `W/"-1"`, `W/"01"`} {
		rec = e.access(t, http.MethodPut, path, "admin-a", fmt.Sprintf("stale-%d", i), stale, `{"rules": []}`)
		wantProblem(t, rec, http.StatusPreconditionFailed, problem.VersionConflict)
		if rec.Header().Get("ETag") != `W/"2"` || !strings.Contains(rec.Body.String(), `W/\"2\"`) {
			t.Errorf("If-Match %s: 412 ETag %q body %s, want the current W/\"2\"", stale, rec.Header().Get("ETag"), rec.Body)
		}
	}
	// A stale If-Match wins over a bad body: the version is checked first.
	wantProblem(t, e.access(t, http.MethodPut, path, "admin-a", "stale-bad", `W/"1"`, `{"nope": 1}`), http.StatusPreconditionFailed, problem.VersionConflict)
	if v, n := e.aclCounts(t); v != versions || n != entries {
		t.Errorf("refusals wrote: versions %d -> %d, entries %d -> %d", versions, v, entries, n)
	}
}

// covers: putEntityTypeAccess 200
func TestPutAccessReplayWithTheSameKeyWritesOneVersion(t *testing.T) {
	e := newPatchEnv(t)
	versions0, entries0 := e.aclCounts(t)
	body := `{"rules": [{"id": "r", "ops": ["read"], "who": {"users": [` + uuidList(e.a.admin.PrincipalID) + `]}}]}`
	first := e.access(t, http.MethodPut, accessPath, "admin-a", "replay", `W/"1"`, body)
	second := e.access(t, http.MethodPut, accessPath, "admin-a", "replay", `W/"1"`, body)
	if first.Code != http.StatusOK || second.Code != http.StatusOK || first.Body.String() != second.Body.String() || second.Header().Get("ETag") != `W/"2"` {
		t.Fatalf("first %d %s / replay %d %s (ETag %q)", first.Code, first.Body, second.Code, second.Body, second.Header().Get("ETag"))
	}
	if v, n := e.aclCounts(t); v != versions0+1 || n != entries0+1 {
		t.Errorf("after a replay: versions %d -> %d, entries %d -> %d, want exactly one of each", versions0, v, entries0, n)
	}
	// The same key with another body is refused as a reuse, and writes nothing.
	wantProblem(t, e.access(t, http.MethodPut, accessPath, "admin-a", "replay", `W/"1"`, `{"rules": []}`), http.StatusConflict, problem.IdempotencyKeyReuse)
	if v, _ := e.aclCounts(t); v != versions0+1 {
		t.Errorf("a key reuse wrote a version: %d", v)
	}
}

// covers: putEntityTypeAccess 400 403 404 422
func TestPutAccessRefusalsWriteNothing(t *testing.T) {
	e := newPatchEnv(t)
	own := uuidList(e.a.adminGroup)
	foreign := uuidList(e.b.adminGroup)
	rule := func(who, where string) string {
		if where != "" {
			where = `, "where": [` + where + `]`
		}
		return `{"rules": [{"id": "r", "ops": ["read"], "who": ` + who + where + `}]}`
	}
	mask := func(read, write string) string {
		return `{"rules": [], "masks": [{"field": "note", "read": ` + read + `, "write": ` + write + `}]}`
	}
	for i, tc := range []struct {
		name, body string
		want       string // "pointer rule"
	}{
		{"unknown column", rule(`{}`, `{"column": "priority", "op": "eq", "values": ["1"]}`), "/rules/0/where/0/column value"},
		{"unknown op", `{"rules": [{"id": "r", "ops": ["delete"], "who": {}}]}`, "/rules/0/ops/0 value"},
		{"unknown caller", rule(`{}`, `{"column": "assignee_id", "op": "eq", "caller": "boss"}`), "/rules/0/where/0/caller value"},
		{"unknown top-level key", `{"rules": [], "extra": 1}`, "/extra unknown_key"},
		{"duplicate rule id", `{"rules": [{"id": "r", "ops": ["read"], "who": {}}, {"id": "r", "ops": ["read"], "who": {}}]}`, "/rules/1/id duplicate"},
		{"group of another tenant", rule(`{"groups": [`+foreign+`]}`, ""), "/rules/0/who/groups/0 not_in_tenant"},
		{"group of no tenant", rule(`{"groups": ["00000000-0000-4000-8000-00000000dead"]}`, ""), "/rules/0/who/groups/0 not_in_tenant"},
		{"user of another tenant", rule(`{"users": [`+uuidList(e.b.user.PrincipalID)+`]}`, ""), "/rules/0/who/users/0 not_in_tenant"},
		{"a group where a user belongs", rule(`{"users": [`+own+`]}`, ""), "/rules/0/who/users/0 not_in_tenant"},
		{"requester of another tenant in a condition", rule(`{}`, `{"column": "requester_id", "op": "eq", "values": [`+uuidList(e.b.user.PrincipalID)+`]}`), "/rules/0/where/0/values/0 not_in_tenant"},
		{"assignee of another tenant in a condition", rule(`{}`, `{"column": "assignee_id", "op": "in", "values": [`+uuidList(e.a.user.PrincipalID, e.b.admin.PrincipalID)+`]}`), "/rules/0/where/0/values/1 not_in_tenant"},
		{"assignment group of another tenant in a condition", rule(`{}`, `{"column": "assignment_group_id", "op": "eq", "values": [`+foreign+`]}`), "/rules/0/where/0/values/0 not_in_tenant"},
		{"mask group of another tenant", mask(`{"groups": [`+foreign+`]}`, `{}`), "/masks/0/read/groups/0 not_in_tenant"},
		{"mask on a field the schema lacks", `{"rules": [], "masks": [{"field": "salary", "read": {}, "write": {}}]}`, "/masks/0/field field"},
		{"mask on a nested path", `{"rules": [], "masks": [{"field": "note/x", "read": {}, "write": {}}]}`, "/masks/0/field field"},
		{"mask read by the requester", mask(`{"requester": true}`, `{}`), "/masks/0/read/requester mask_subject"},
		{"mask read by the assignee", mask(`{"assignee": true}`, `{}`), "/masks/0/read/assignee mask_subject"},
		{"mask read by a share holder", mask(`{"shared": true}`, `{}`), "/masks/0/read/shared mask_subject"},
		{"mask write by the requester", mask(`{}`, `{"requester": true}`), "/masks/0/write/requester mask_subject"},
		{"mask write wider than read", mask(`{"groups": [`+own+`]}`, `{"groups": [`+own+`, `+uuidList(e.a.lookalikeGroup)+`]}`), "/masks/0/write/groups/1 mask_write"},
	} {
		versions, entries := e.aclCounts(t)
		rec := e.access(t, http.MethodPut, accessPath, "admin-a", fmt.Sprintf("refuse-%d", i), `W/"1"`, tc.body)
		wantProblem(t, rec, http.StatusUnprocessableEntity, problem.InvalidAccessPolicy)
		if got := pointerRules(t, rec); !containsStr(got, tc.want) {
			t.Errorf("%s: errors %v, want %q", tc.name, got, tc.want)
		}
		// The response names pointers and rules, never a value of the request.
		for _, leak := range []string{e.b.adminGroup.String(), e.b.user.PrincipalID.String(), e.b.admin.PrincipalID.String(), "dead"} {
			if strings.Contains(rec.Body.String(), leak) {
				t.Errorf("%s: the 422 body repeats %q: %s", tc.name, leak, rec.Body)
			}
		}
		if v, n := e.aclCounts(t); v != versions || n != entries {
			t.Errorf("%s: wrote versions %d -> %d, entries %d -> %d", tc.name, versions, v, entries, n)
		}
	}
	// A body that is not a JSON object fails the contract's validator: 400, not
	// 422 (A3). Nothing is written, and the cases below it in this test still run.
	versions, entries := e.aclCounts(t)
	for i, body := range []string{`[]`, `"rules"`, `5`} {
		rec := e.access(t, http.MethodPut, accessPath, "admin-a", fmt.Sprintf("notobject-%d", i), `W/"1"`, body)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("body %s = %d, want 400 validation_failed: %s", body, rec.Code, rec.Body)
			continue
		}
		wantProblem(t, rec, http.StatusBadRequest, problem.ValidationFailed)
	}
	if v, n := e.aclCounts(t); v != versions || n != entries {
		t.Errorf("a non-object body wrote: versions %d -> %d, entries %d -> %d", versions, v, entries, n)
	}
	// A foreign id and an id that does not exist are refused alike.
	a := e.access(t, http.MethodPut, accessPath, "admin-a", "alike-1", `W/"1"`, rule(`{"groups": [`+foreign+`]}`, ""))
	b := e.access(t, http.MethodPut, accessPath, "admin-a", "alike-2", `W/"1"`, rule(`{"groups": ["00000000-0000-4000-8000-00000000dead"]}`, ""))
	if a.Code != b.Code || a.Body.String() != b.Body.String() {
		t.Errorf("foreign id %d %s vs unknown id %d %s: the 422 must be the same", a.Code, a.Body, b.Code, b.Body)
	}

	// Own-tenant ids of every kind are accepted.
	good := `{"rules": [{"id": "r", "ops": ["read", "write"], "who": {"groups": [` + own + `], "users": [` + uuidList(e.a.user.PrincipalID) + `]},
		"where": [{"column": "assignment_group_id", "op": "in", "values": [` + own + `]}, {"column": "requester_id", "op": "eq", "values": [` + uuidList(e.a.user.PrincipalID) + `]}]}],
		"masks": [{"field": "note", "read": {"groups": [` + own + `]}, "write": {}}]}`
	if rec := e.access(t, http.MethodPut, accessPath, "admin-a", "good", `W/"1"`, good); rec.Code != http.StatusOK {
		t.Fatalf("own-tenant ids = %d: %s", rec.Code, rec.Body)
	}

	// 403: not an admin, before the body or the version is looked at. 404: an
	// unknown type, before the body.
	versions, entries = e.aclCounts(t)
	for _, tok := range []string{"user-a", "lookalike-a"} {
		wantProblem(t, e.access(t, http.MethodPut, accessPath, tok, "403-"+tok, `W/"9"`, `{"nope": 1}`), http.StatusForbidden, problem.Forbidden)
	}
	// Tenant B's admin writes tenant B's type, so tenant A's group is a foreign id there.
	if rec := e.access(t, http.MethodPut, accessPath, "admin-b", "b-foreign", `W/"1"`, `{"rules": [{"id": "r", "ops": ["read"], "who": {"groups": [`+own+`]}}]}`); rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("tenant B naming tenant A's group = %d, want 422: %s", rec.Code, rec.Body)
	}
	wantProblem(t, e.access(t, http.MethodPut, "/api/v1/entity-types/nothing/access", "admin-a", "404", `W/"0"`, `{"nope": 1}`), http.StatusNotFound, problem.NotFound)
	wantProblem(t, e.access(t, http.MethodPut, accessPath, "admin-a", "", `W/"2"`, `{"rules": []}`), http.StatusBadRequest, problem.ValidationFailed)
	if v, n := e.aclCounts(t); v != versions || n != entries {
		t.Errorf("403, 404 and 400 wrote: versions %d -> %d, entries %d -> %d", versions, v, entries, n)
	}
}

// A tenant admin of tenant B cannot publish for tenant A's type: the key is
// resolved in B's own tenant, so it is B's type that changes.
// covers: putEntityTypeAccess 200
func TestPutAccessAdminOfAnotherTenantChangesOnlyItsOwnTenant(t *testing.T) {
	e := newPatchEnv(t)
	versions, entries := e.aclCounts(t)
	rec := e.access(t, http.MethodPut, accessPath, "admin-b", "b-own", `W/"1"`, `{"rules": [{"id": "b", "ops": ["read"], "who": {"users": [`+uuidList(e.b.admin.PrincipalID)+`]}}]}`)
	if rec.Code != http.StatusOK || decodeAccess(t, rec).Version != 2 {
		t.Fatalf("tenant B publish = %d: %s", rec.Code, rec.Body)
	}
	if v, n := e.aclCounts(t); v != versions || n != entries {
		t.Errorf("tenant A changed: versions %d -> %d, entries %d -> %d", versions, v, entries, n)
	}
	if got := e.access(t, http.MethodGet, accessPath, "admin-a", "", "", ""); decodeAccess(t, got).Version != 1 {
		t.Errorf("tenant A's policy moved: %s", got.Body)
	}
}

func (e *etEnv) explain(t *testing.T, token, entity, record, user string) *httptest.ResponseRecorder {
	t.Helper()
	return e.access(t, http.MethodGet, fmt.Sprintf("/api/v1/access/explain?entity=%s&record=%s&user=%s", entity, record, user), token, "", "", "")
}

type explanation struct {
	Decision     string   `json:"decision"`
	RuleID       *string  `json:"rule_id"`
	ACLVersion   int      `json:"acl_version"`
	MaskedFields []string `json:"masked_fields"`
}

func decodeExplanation(t *testing.T, rec *httptest.ResponseRecorder) explanation {
	t.Helper()
	var x explanation
	if err := json.Unmarshal(rec.Body.Bytes(), &x); err != nil {
		t.Fatalf("not an explanation: %v: %s", err, rec.Body)
	}
	return x
}

// covers: explainAccess 200 400 403 404
func TestExplainAccessNamesTheRuleAndHidesWhatTheAdminCannotRead(t *testing.T) {
	e := newPatchEnv(t)
	rec := e.mustCreateRec(t, "user-a", "job", "explain-1", `{"short_description": "one"}`)
	recB := e.mustCreateRec(t, "user-b", "job", "explain-b", `{"short_description": "tenant B"}`)

	// Version 2: the admin and the user may read, and write; note is hidden
	// from everyone outside the admins group; code only the lookalike group (of
	// which the asking admin is not a member) may read.
	put := e.access(t, http.MethodPut, accessPath, "admin-a", "explain-put", `W/"1"`,
		`{"rules": [{"id": "r-desk", "ops": ["read", "write"], "who": {"users": [`+uuidList(e.a.admin.PrincipalID, e.a.user.PrincipalID)+`]}}],
		 "masks": [{"field": "note", "read": {"groups": [`+uuidList(e.a.adminGroup)+`]}, "write": {"groups": [`+uuidList(e.a.adminGroup)+`]}},
		           {"field": "code", "read": {"groups": [`+uuidList(e.a.lookalikeGroup)+`]}, "write": {"groups": [`+uuidList(e.a.lookalikeGroup)+`]}}]}`)
	if put.Code != http.StatusOK {
		t.Fatalf("publish = %d: %s", put.Code, put.Body)
	}

	// Allow: the granting rule's id and the ACL version.
	got := e.explain(t, "admin-a", "job", rec.ID, e.a.user.PrincipalID.String())
	if got.Code != http.StatusOK {
		t.Fatalf("explain = %d: %s", got.Code, got.Body)
	}
	if x := decodeExplanation(t, got); x.Decision != "allow" || x.RuleID == nil || *x.RuleID != "r-desk" || x.ACLVersion != 2 {
		t.Errorf("allow = %+v, want allow by r-desk at version 2", x)
	}
	// The user is hidden note (not in the admins group) and code (not in the
	// lookalike group); the admin cannot read code itself, so only note shows.
	if x := decodeExplanation(t, got); len(x.MaskedFields) != 1 || x.MaskedFields[0] != "note" {
		t.Errorf("masked fields = %v, want [note]: code is a field the admin cannot read", x.MaskedFields)
	}
	if strings.Contains(got.Body.String(), "code") {
		t.Errorf("the explanation mentions a field the admin cannot read: %s", got.Body)
	}
	// The admin as target: reads note; code is hidden from it too, and from the asker.
	if x := decodeExplanation(t, e.explain(t, "admin-a", "job", rec.ID, e.a.admin.PrincipalID.String())); x.Decision != "allow" || len(x.MaskedFields) != 0 {
		t.Errorf("admin as target = %+v, want allow with no listed field", x)
	}
	// Deny: no rule names the lookalike, and rule_id is null, not omitted.
	deny := e.explain(t, "admin-a", "job", rec.ID, e.a.lookalike.PrincipalID.String())
	if x := decodeExplanation(t, deny); deny.Code != http.StatusOK || x.Decision != "deny" || x.RuleID != nil || x.ACLVersion != 2 {
		t.Errorf("deny = %d %+v, want deny with a null rule at version 2", deny.Code, x)
	}
	if !strings.Contains(deny.Body.String(), `"rule_id":null`) {
		t.Errorf("deny body = %s, want rule_id null", deny.Body)
	}
	// The explanation is the enforcement: the lookalike really cannot read it,
	// and the user really can.
	if r := e.do(t, http.MethodGet, "/api/v1/records/job/"+rec.ID, "lookalike-a", "", ""); r.Code != http.StatusNotFound {
		t.Errorf("lookalike reads the record = %d, the explainer said deny", r.Code)
	}
	if r := e.do(t, http.MethodGet, "/api/v1/records/job/"+rec.ID, "user-a", "", ""); r.Code != http.StatusOK {
		t.Errorf("user reads the record = %d, the explainer said allow", r.Code)
	}

	// 404: another tenant's record, another tenant's user, an unknown record, an
	// unknown user, the wrong entity and malformed ids all give the same bytes.
	missing := e.explain(t, "admin-a", "job", uuid.NewString(), e.a.user.PrincipalID.String())
	wantProblem(t, missing, http.StatusNotFound, problem.NotFound)
	for name, r := range map[string]*httptest.ResponseRecorder{
		"record of tenant B":  e.explain(t, "admin-a", "job", recB.ID, e.a.user.PrincipalID.String()),
		"user of tenant B":    e.explain(t, "admin-a", "job", rec.ID, e.b.user.PrincipalID.String()),
		"unknown user":        e.explain(t, "admin-a", "job", rec.ID, uuid.NewString()),
		"wrong entity":        e.explain(t, "admin-a", "nothing", rec.ID, e.a.user.PrincipalID.String()),
		"malformed record":    e.explain(t, "admin-a", "job", "not-a-uuid", e.a.user.PrincipalID.String()),
		"malformed user":      e.explain(t, "admin-a", "job", rec.ID, "not-a-uuid"),
		"over-long entity":    e.explain(t, "admin-a", strings.Repeat("k", 80), rec.ID, e.a.user.PrincipalID.String()),
		"user is a group id":  e.explain(t, "admin-a", "job", rec.ID, e.a.adminGroup.String()),
		"record is a user id": e.explain(t, "admin-a", "job", e.a.user.PrincipalID.String(), e.a.user.PrincipalID.String()),
	} {
		if r.Code != http.StatusNotFound || r.Body.String() != missing.Body.String() {
			t.Errorf("%s = %d %s, want the same 404 as a missing record", name, r.Code, r.Body)
		}
	}
	// Tenant B's admin asking about tenant A's record gets the same 404.
	if r := e.explain(t, "admin-b", "job", rec.ID, e.b.user.PrincipalID.String()); r.Code != http.StatusNotFound || r.Body.String() != missing.Body.String() {
		t.Errorf("tenant B's admin asking about tenant A's record = %d %s", r.Code, r.Body)
	}

	// 403: not an admin, before any lookup.
	for _, tok := range []string{"user-a", "lookalike-a"} {
		wantProblem(t, e.explain(t, tok, "job", rec.ID, e.a.user.PrincipalID.String()), http.StatusForbidden, problem.Forbidden)
		wantProblem(t, e.explain(t, tok, "job", uuid.NewString(), uuid.NewString()), http.StatusForbidden, problem.Forbidden)
	}
	// 400: a missing parameter.
	wantProblem(t, e.access(t, http.MethodGet, "/api/v1/access/explain?entity=job&record="+rec.ID, "admin-a", "", "", ""), http.StatusBadRequest, problem.ValidationFailed)
}

// An empty policy allows nothing: the explainer says deny with a null rule and
// an empty (not absent) masked list. Version 0 (no policy) is the unit test.
// covers: explainAccess 200
func TestExplainAccessUnderAnEmptyPolicyDenies(t *testing.T) {
	e := newPatchEnv(t)
	rec := e.mustCreateRec(t, "user-a", "job", "explain-none", `{"short_description": "one"}`)
	// Replace the policy with an empty one: it allows nothing.
	if r := e.access(t, http.MethodPut, accessPath, "admin-a", "explain-empty", `W/"1"`, `{"rules": []}`); r.Code != http.StatusOK {
		t.Fatalf("publish = %d: %s", r.Code, r.Body)
	}
	x := decodeExplanation(t, e.explain(t, "admin-a", "job", rec.ID, e.a.user.PrincipalID.String()))
	if x.Decision != "deny" || x.RuleID != nil || x.ACLVersion != 2 || x.MaskedFields == nil {
		t.Errorf("explanation under an empty policy = %+v", x)
	}
}

// Regression (contract amendment A3): an entity type's version 1 and its ACL
// version 1 written in one transaction. Both audit entries used to share the
// key (transaction, entity_definition, definition id, version 1) and the
// second append failed with a duplicate; the ACL has its own subject, acl_policy.
// The fixtures of the records pipeline do exactly this, and the transaction
// must stay one.
// covers: putEntityTypeAccess 200
func TestEntityTypeAndItsACLInOneTransactionHaveOneAuditEntryEach(t *testing.T) {
	e := newPatchEnv(t)
	err := tenancy.InTenantTx(t.Context(), e.pool, e.a.id, func(tx pgx.Tx) error {
		def := metadata.Definition{
			Key: "oneshot", NumberPrefix: "ONE", Schema: json.RawMessage(patchSchema), FixedColumns: map[string]string{"state": "state"},
			States: metadata.StateModel{Initial: "new", States: []string{"new", "closed"}, Transitions: []metadata.Transition{{From: "new", To: "closed"}}},
		}
		if _, err := (entitytypes.Store{}).Create(t.Context(), tx, e.a.id, entitytypes.NewInput{Name: "One shot", PublishedBy: e.a.admin.PrincipalID, Definition: def}); err != nil {
			return err
		}
		_, err := acltest.Publish(t.Context(), tx, "oneshot", acltest.AllowUsers(e.a.admin.PrincipalID))
		return err
	})
	if err != nil {
		t.Fatalf("type and ACL in one transaction: %v", err)
	}
	for subject, want := range map[string]int{"entity_definition": 1, "acl_policy": 1} {
		var n int
		if err := e.super.QueryRow(t.Context(), `SELECT count(*) FROM audit_log a JOIN entity_definitions d ON d.tenant_id = a.tenant_id AND d.id = a.subject_id
			WHERE a.tenant_id = $1 AND d.key = 'oneshot' AND a.subject_type = $2 AND a.version_after = 1`, e.a.id, subject).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != want {
			t.Errorf("%d %s entries at version 1, want %d", n, subject, want)
		}
	}
}

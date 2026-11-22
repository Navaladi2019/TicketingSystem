//go:build integration

// TICK-939 (S30 P11-07): PUT /api/v1/entity-types/{key}/access with format 2.
// The route accepts format 1 or 2, runs every D4 save-time check, stores
// format 2 in the same transaction as its audit entry, and GET returns format
// 2 for any stored version. The 422 table is the shared fixture
// internal/authz/testdata/format2_invalid.json.

package server_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/audit"
	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
	"github.com/Navaladi2019/TicketingSystem/internal/server"
)

const foreignGroup = "0b0b0b0b-0000-4000-8000-000000000001"

// format2Doc is a one-policy document: members of the group read.
func format2Doc(group uuid.UUID) string {
	return `{"format": 2, "conditions": [{"name": "open", "expr": {"cmp": {"column": "state", "op": "in", "values": ["new"]}}}],
		"policies": [{"name": "readers", "ops": ["read"], "when": {"all": [{"member_of": {"groups": ["` + group.String() + `"]}}, {"ref": "open"}]}}]}`
}

func (e *etEnv) storedFormat(t *testing.T, version int) (typ string, size int) {
	t.Helper()
	if err := e.super.QueryRow(t.Context(), `SELECT jsonb_typeof(a.rules), octet_length(a.rules::text) FROM acl_versions a
		JOIN entity_definitions d ON d.tenant_id = a.tenant_id AND d.id = a.entity_definition_id
		WHERE a.tenant_id = $1 AND d.key = 'job' AND a.version = $2`, e.a.id, version).Scan(&typ, &size); err != nil {
		t.Fatal(err)
	}
	return typ, size
}

// covers: putEntityTypeAccess 200
func TestPutAccessFormat2StoresAsSubmittedWithOneAuditEntry(t *testing.T) {
	e := newPatchEnv(t)
	versions0, entries0 := e.aclCounts(t)
	doc := format2Doc(e.a.adminGroup)
	rec := e.access(t, http.MethodPut, accessPath, "admin-a", "f2-1", `W/"1"`, `{"rules": `+doc+`, "masks": []}`)
	if rec.Code != http.StatusOK || rec.Header().Get("ETag") != `W/"2"` {
		t.Fatalf("PUT format 2 = %d ETag %q: %s", rec.Code, rec.Header().Get("ETag"), rec.Body)
	}
	b := decodeAccess(t, rec)
	if b.Version != 2 || b.Rules.Format != 2 || len(b.Rules.Policies) != 1 || len(b.Rules.Conditions) != 1 || len(b.Warnings) != 0 {
		t.Fatalf("published = %+v", b)
	}
	if v, n := e.aclCounts(t); v != versions0+1 || n != entries0+1 {
		t.Errorf("versions %d -> %d, entries %d -> %d, want one of each", versions0, v, entries0, n)
	}
	if typ, _ := e.storedFormat(t, 2); typ != "object" {
		t.Errorf("stored rules are %s, want an object", typ)
	}
	// GET returns the same body as published.
	got := e.access(t, http.MethodGet, accessPath, "admin-a", "", "", "")
	if got.Code != http.StatusOK || got.Body.String() != rec.Body.String() {
		t.Errorf("GET %d %s differs from the published body %s", got.Code, got.Body, rec.Body)
	}
}

// covers: putEntityTypeAccess 200
func TestPutAccessFormat1IsStoredAsFormat2(t *testing.T) {
	e := newPatchEnv(t)
	body := `{"rules": [{"id": "Read Own", "ops": ["read"], "who": {"requester": true}}]}`
	rec := e.access(t, http.MethodPut, accessPath, "admin-a", "f1-1", `W/"1"`, body)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT format 1 = %d: %s", rec.Code, rec.Body)
	}
	b := decodeAccess(t, rec)
	if b.Rules.Format != 2 || len(b.Rules.Policies) != 1 || b.Rules.Policies[0]["name"] != "read_own" {
		t.Errorf("response rules = %+v, want format 2", b.Rules)
	}
	if typ, _ := e.storedFormat(t, 2); typ != "object" {
		t.Errorf("stored rules are %s, want an object (format 2)", typ)
	}
	// The stored document carries no legacy_id (format-2 input refuses it).
	var legacy bool
	if err := e.super.QueryRow(t.Context(), `SELECT rules::text LIKE '%legacy_id%' FROM acl_versions WHERE tenant_id = $1 AND version = 2`, e.a.id).Scan(&legacy); err != nil || legacy {
		t.Errorf("stored text has legacy_id: %v %v", legacy, err)
	}
	// A rule with no subject is refused at its who, not stored.
	versions, entries := e.aclCounts(t)
	rec = e.access(t, http.MethodPut, accessPath, "admin-a", "f1-2", `W/"2"`, `{"rules": [{"id": "r", "ops": ["read"], "who": {}}]}`)
	wantProblem(t, rec, http.StatusUnprocessableEntity, problem.InvalidAccessPolicy)
	if got := pointerRules(t, rec); !containsStr(got, "/rules/0/who child_count") {
		t.Errorf("errors = %v", got)
	}
	if v, n := e.aclCounts(t); v != versions || n != entries {
		t.Errorf("a refused body wrote: versions %d -> %d, entries %d -> %d", versions, v, entries, n)
	}
}

// Every row of the shared fixture is a 422 with its pointer (under /rules) and
// code, and writes no version and no audit entry. A go_only row is a
// duplicate key, which the strict-JSON boundary (TICK-964) refuses before the
// access decoder: its expected answer is derived from the row, code duplicate
// at "/body" + the row's body pointer (TICK-1017/TICK-1018 decision). A row
// with a body is sent as it is; its pointers are already into the body.
// covers: putEntityTypeAccess 422
func TestPutAccessRefusesEveryFixtureRow(t *testing.T) {
	e := newPatchEnv(t)
	b, err := os.ReadFile("../authz/testdata/format2_invalid.json")
	if err != nil {
		t.Fatal(err)
	}
	var rows []struct {
		Name     string          `json:"name"`
		GoOnly   bool            `json:"go_only"`
		MaxBytes int             `json:"max_bytes"`
		Doc      json.RawMessage `json:"doc"`
		Body     json.RawMessage `json:"body"`
		Problems []struct {
			Pointer string `json:"pointer"`
			Code    string `json:"code"`
		} `json:"problems"`
	}
	if err := json.Unmarshal(b, &rows); err != nil || len(rows) < 60 {
		t.Fatalf("fixture: %d rows, %v", len(rows), err)
	}
	for i, r := range rows {
		if r.MaxBytes > 0 {
			continue // the byte limit has its own test
		}
		send, under := `{"rules": `+string(r.Doc)+`}`, "/rules"
		if r.Body != nil {
			send, under = string(r.Body), ""
		}
		versions, entries := e.aclCounts(t)
		rec := e.access(t, http.MethodPut, accessPath, "admin-a", fmt.Sprintf("fx-%d", i), `W/"1"`, send)
		if rec.Code == http.StatusBadRequest && !r.GoOnly {
			continue // a document the contract's validator refuses first is no 422 case
		}
		wantProblem(t, rec, http.StatusUnprocessableEntity, problem.InvalidAccessPolicy)
		got := pointerRules(t, rec)
		for _, p := range r.Problems {
			want := under + p.Pointer + " " + p.Code
			if r.GoOnly {
				want = "/body" + under + p.Pointer + " " + authz.RuleDuplicate
			}
			if !containsStr(got, want) {
				t.Errorf("%s: errors %v, want %q", r.Name, got, want)
			}
		}
		if v, n := e.aclCounts(t); v != versions || n != entries {
			t.Errorf("%s: wrote versions %d -> %d, entries %d -> %d", r.Name, versions, v, entries, n)
		}
	}
}

// covers: putEntityTypeAccess 412 200
func TestPutAccessStaleAndReplayedVersionWriteNothing(t *testing.T) {
	e := newPatchEnv(t)
	body := `{"rules": ` + format2Doc(e.a.adminGroup) + `}`
	versions, entries := e.aclCounts(t)
	// Two PUTs on the same expected version: one write, the second is stale.
	first := e.access(t, http.MethodPut, accessPath, "admin-a", "race-1", `W/"1"`, body)
	second := e.access(t, http.MethodPut, accessPath, "admin-a", "race-2", `W/"1"`, body)
	if first.Code != http.StatusOK {
		t.Fatalf("first = %d: %s", first.Code, first.Body)
	}
	wantProblem(t, second, http.StatusPreconditionFailed, problem.VersionConflict)
	if second.Header().Get("ETag") != `W/"2"` {
		t.Errorf("the conflict names %q, want W/\"2\"", second.Header().Get("ETag"))
	}
	if v, n := e.aclCounts(t); v != versions+1 || n != entries+1 {
		t.Errorf("versions %d -> %d, entries %d -> %d, want one of each", versions, v, entries, n)
	}
}

// covers: putEntityTypeAccess 404
func TestAccessOfAnotherTenantsKeyIsTheUnknownKeyResponse(t *testing.T) {
	e := newPatchEnv(t)
	e.noTestACL = true
	if r := e.do(t, http.MethodPost, "/api/v1/entity-types", "admin-a", "only-a", etBody("widget", "WGT")); r.Code != http.StatusCreated {
		t.Fatalf("create widget = %d: %s", r.Code, r.Body)
	}
	const path = "/api/v1/entity-types/widget/access"
	unknown := e.access(t, http.MethodPut, "/api/v1/entity-types/nothing/access", "admin-b", "k-1", `W/"0"`, `{"rules": []}`)
	other := e.access(t, http.MethodPut, path, "admin-b", "k-2", `W/"0"`, `{"rules": `+format2Doc(e.b.adminGroup)+`}`)
	wantProblem(t, other, http.StatusNotFound, problem.NotFound)
	if sameProblem(t, other) != sameProblem(t, unknown) {
		t.Errorf("another tenant's key %s differs from the unknown key %s", other.Body, unknown.Body)
	}
	if v, _ := e.aclCounts(t); v != 1 {
		t.Errorf("tenant A has %d versions, want its one", v)
	}
}

// Foreign ids in format 2 are foreign_id, with their pointer, before the
// insert, whatever node carries them. The 422 for another tenant's id is the
// same as the 422 for an id that does not exist: no existence oracle.
// covers: putEntityTypeAccess 422
func TestPutAccessFormat2ForeignIDsAreForeignID(t *testing.T) {
	e := newPatchEnv(t)
	pol := func(when string) string {
		return `{"rules": {"format": 2, "conditions": [], "policies": [{"name": "p", "ops": ["read"], "when": ` + when + `}]}}`
	}
	for _, tc := range []struct {
		name    string
		foreign string // the id of tenant B
		body    func(id string) string
		want    string
	}{
		{"member_of group", e.b.adminGroup.String(), func(id string) string {
			return pol(`{"member_of": {"groups": ["` + id + `"]}}`)
		}, "/rules/policies/0/when/member_of/groups/0 foreign_id"},
		{"member_of user", e.b.user.PrincipalID.String(), func(id string) string {
			return pol(`{"member_of": {"users": ["` + id + `"]}}`)
		}, "/rules/policies/0/when/member_of/users/0 foreign_id"},
		{"cmp on requester_id", e.b.user.PrincipalID.String(), func(id string) string {
			return pol(`{"cmp": {"column": "requester_id", "op": "eq", "values": ["` + id + `"]}}`)
		}, "/rules/policies/0/when/cmp/values/0 foreign_id"},
		{"cmp on assignment_group_id", e.b.adminGroup.String(), func(id string) string {
			return pol(`{"cmp": {"column": "assignment_group_id", "op": "in", "values": ["` + id + `"]}}`)
		}, "/rules/policies/0/when/cmp/values/0 foreign_id"},
		{"condition reached through ref", e.b.adminGroup.String(), func(id string) string {
			return `{"rules": {"format": 2, "conditions": [{"name": "c", "expr": {"member_of": {"groups": ["` + id + `"]}}}],
				"policies": [{"name": "p", "ops": ["read"], "when": {"ref": "c"}}]}}`
		}, "/rules/conditions/0/expr/member_of/groups/0 foreign_id"},
		{"mask group", e.b.adminGroup.String(), func(id string) string {
			return `{"rules": {"format": 2, "conditions": [], "policies": []}, "masks": [{"field": "note", "read": {"groups": ["` + id + `"]}, "write": {}}]}`
		}, "/masks/0/read/groups/0 foreign_id"},
	} {
		versions, entries := e.aclCounts(t)
		other := e.access(t, http.MethodPut, accessPath, "admin-a", "fid-other-"+strings.ReplaceAll(tc.name, " ", "-"), `W/"1"`, tc.body(tc.foreign))
		none := e.access(t, http.MethodPut, accessPath, "admin-a", "fid-none-"+strings.ReplaceAll(tc.name, " ", "-"), `W/"1"`, tc.body(uuid.NewString()))
		wantProblem(t, other, http.StatusUnprocessableEntity, problem.InvalidAccessPolicy)
		wantProblem(t, none, http.StatusUnprocessableEntity, problem.InvalidAccessPolicy)
		if got := pointerRules(t, other); !containsStr(got, tc.want) {
			t.Errorf("%s: errors %v, want %q", tc.name, got, tc.want)
		}
		if strings.Contains(other.Body.String(), tc.foreign) {
			t.Errorf("%s: the 422 repeats the id", tc.name)
		}
		if sameProblem(t, other) != sameProblem(t, none) {
			t.Errorf("%s: another tenant's id %s differs from an unknown id %s", tc.name, other.Body, none.Body)
		}
		if v, n := e.aclCounts(t); v != versions || n != entries {
			t.Errorf("%s: wrote versions %d -> %d, entries %d -> %d", tc.name, versions, v, entries, n)
		}
	}
}

// The warning lint accepts the document and reports the `not`.
// covers: putEntityTypeAccess 200
func TestPutAccessWarnsOnNotOverCallerLeaf(t *testing.T) {
	e := newPatchEnv(t)
	doc := `{"format": 2, "conditions": [], "policies": [{"name": "p", "ops": ["read"], "when": {"not": {"cmp": {"column": "requester_id", "op": "eq", "caller": "self"}}}}]}`
	rec := e.access(t, http.MethodPut, accessPath, "admin-a", "warn-1", `W/"1"`, `{"rules": `+doc+`}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT = %d: %s", rec.Code, rec.Body)
	}
	b := decodeAccess(t, rec)
	if len(b.Warnings) != 1 || b.Warnings[0].Code != "not_over_caller_leaf" || b.Warnings[0].Pointer != "/rules/policies/0/when/not" {
		t.Errorf("warnings = %+v", b.Warnings)
	}
	if got := decodeAccess(t, e.access(t, http.MethodGet, accessPath, "admin-a", "", "", "")); len(got.Warnings) != 0 {
		t.Errorf("GET carries warnings: %+v", got.Warnings)
	}
}

// padded returns a format-2 document of exactly n bytes of stored text: unused
// conditions pad it, and one trailing condition name takes up the slack.
func padded(t *testing.T, n int) string {
	t.Helper()
	const head = `{"format": 2, "policies": [], "conditions": [`
	const tail = `]}`
	var sb strings.Builder
	sb.WriteString(head)
	for i := 0; ; i++ {
		item := fmt.Sprintf(`{"expr": {"shared": true}, "name": "c%d"}`, i)
		sep := ", "
		if i == 0 {
			sep = ""
		}
		if sb.Len()+len(sep)+len(item)+len(tail) > n {
			break
		}
		sb.WriteString(sep + item)
	}
	// Grow the last name until the document is exactly n bytes. The stored
	// form orders keys by length, then bytes: expr, name, with the text
	// `{"expr": ..., "name": ...}` as written above.
	s := sb.String() + tail
	if short := n - len(s); short > 0 {
		s = strings.Replace(s, `"name": "c0"`, `"name": "c0`+strings.Repeat("x", short)+`"`, 1)
	}
	if len(s) != n {
		t.Fatalf("padded document is %d bytes, want %d", len(s), n)
	}
	return s
}

// covers: putEntityTypeAccess 200 422
func TestPutAccessByteLimitIsExactOnRulesAndMasks(t *testing.T) {
	e := newPatchEnv(t)
	for _, n := range []int{32768, 32769} {
		doc := padded(t, n)
		rec := e.access(t, http.MethodPut, accessPath, "admin-a", fmt.Sprintf("size-%d", n), fmt.Sprintf(`W/"%d"`, e.currentVersion(t)), `{"rules": `+doc+`}`)
		if n == 32768 {
			if rec.Code != http.StatusOK {
				t.Fatalf("exactly 32,768 bytes = %d: %s", rec.Code, rec.Body)
			}
			if typ, size := e.storedFormat(t, e.currentVersion(t)); typ != "object" || size != 32768 {
				t.Errorf("stored %s of %d bytes, want an object of exactly 32,768", typ, size)
			}
			continue
		}
		wantProblem(t, rec, http.StatusUnprocessableEntity, problem.InvalidAccessPolicy)
		if got := pointerRules(t, rec); !containsStr(got, "/rules document_too_large") {
			t.Errorf("32,769 bytes: errors %v", got)
		}
		if !strings.Contains(rec.Body.String(), "32768") {
			t.Errorf("the 422 does not state the limit: %s", rec.Body)
		}
	}
	// Masks: the same limit, pointer /masks.
	versions, entries := e.aclCounts(t)
	bigMasks := func(n int) string {
		item := `{"field": "note", "read": {}, "write": {}}`
		parts := make([]string, 0, n/len(item)+1)
		size := 2
		for size+len(item)+2 <= n {
			parts = append(parts, item)
			size += len(item) + 2
		}
		return "[" + strings.Join(parts, ", ") + "]"
	}
	m := bigMasks(40000)
	rec := e.access(t, http.MethodPut, accessPath, "admin-a", "masks-big", fmt.Sprintf(`W/"%d"`, e.currentVersion(t)), `{"rules": `+format2Doc(e.a.adminGroup)+`, "masks": `+m+`}`)
	wantProblem(t, rec, http.StatusUnprocessableEntity, problem.InvalidAccessPolicy)
	if got := pointerRules(t, rec); !containsStr(got, "/masks document_too_large") {
		t.Errorf("masks over the limit: errors %v", got)
	}
	if v, n := e.aclCounts(t); v != versions || n != entries {
		t.Errorf("a refused body wrote")
	}
}

func (e *etEnv) currentVersion(t *testing.T) int {
	t.Helper()
	var v int
	if err := e.super.QueryRow(t.Context(), `SELECT coalesce(max(a.version), 0) FROM acl_versions a
		JOIN entity_definitions d ON d.tenant_id = a.tenant_id AND d.id = a.entity_definition_id
		WHERE a.tenant_id = $1 AND d.key = 'job'`, e.a.id).Scan(&v); err != nil {
		t.Fatal(err)
	}
	return v
}

// With the setting at 65,536 a 40,000-byte document is accepted.
// covers: putEntityTypeAccess 200
func TestPutAccessHonoursTheConfiguredLimit(t *testing.T) {
	e := newPatchEnvWith(t, server.WithACLMaxDocumentBytes(65536))
	rec := e.access(t, http.MethodPut, accessPath, "admin-a", "cfg-1", `W/"1"`, `{"rules": `+padded(t, 40000)+`}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("40,000 bytes at a 65,536 limit = %d: %s", rec.Code, rec.Body)
	}
	rec = e.access(t, http.MethodPut, accessPath, "admin-a", "cfg-2", `W/"2"`, `{"rules": `+padded(t, 65537)+`}`)
	wantProblem(t, rec, http.StatusUnprocessableEntity, problem.InvalidAccessPolicy)
	if !strings.Contains(rec.Body.String(), "65536") || strings.Contains(rec.Body.String(), "32768") {
		t.Errorf("the 422 must state the configured limit: %s", rec.Body)
	}
}

// The database's byte ceiling is the backstop: when the route's limit is set
// above it (a test-only setting, nava-core refuses to start there), the 23514
// on acl_versions_bytes_rules_check is the same 422, never a 500.
// covers: putEntityTypeAccess 422
func TestPutAccessMapsTheDatabaseByteCeilingTo422(t *testing.T) {
	e := newPatchEnvWith(t, server.WithACLMaxDocumentBytes(1<<20))
	versions, entries := e.aclCounts(t)
	rec := e.access(t, http.MethodPut, accessPath, "admin-a", "ceil-1", `W/"1"`, `{"rules": `+padded(t, 131073)+`}`)
	wantProblem(t, rec, http.StatusUnprocessableEntity, problem.InvalidAccessPolicy)
	if got := pointerRules(t, rec); !containsStr(got, "/rules document_too_large") {
		t.Errorf("errors %v", got)
	}
	rec = e.access(t, http.MethodPut, accessPath, "admin-a", "ceil-2", `W/"1"`, `{"rules": [], "masks": `+bigMasksOver(131073)+`}`)
	wantProblem(t, rec, http.StatusUnprocessableEntity, problem.InvalidAccessPolicy)
	if got := pointerRules(t, rec); !containsStr(got, "/masks document_too_large") {
		t.Errorf("masks: errors %v", got)
	}
	if v, n := e.aclCounts(t); v != versions || n != entries {
		t.Errorf("a refused body wrote")
	}
}

func bigMasksOver(n int) string {
	item := `{"field": "note", "read": {}, "write": {}}`
	count := n/(len(item)+2) + 1
	parts := make([]string, count)
	for i := range parts {
		parts[i] = item
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

// R2-1: no route accepts the byte-limit setting, and no response carries its
// value except the document_too_large 422's detail.
// covers: putEntityTypeAccess 200
func TestTheByteLimitSettingIsNotExposed(t *testing.T) {
	const limit = 54321
	e := newPatchEnvWith(t, server.WithACLMaxDocumentBytes(limit))
	needle := fmt.Sprint(limit)
	b, err := os.ReadFile("../../api/openapi.bundled.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "max_document_bytes") || strings.Contains(string(b), "NAVA_ACL_MAX_DOCUMENT_BYTES") {
		t.Error("the OpenAPI document names the setting")
	}
	// Every path of the OpenAPI document, with a body that carries the setting.
	type op struct{ method, path string }
	var ops []op
	for _, line := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(line, "  /api/") && strings.HasSuffix(line, ":") {
			p := strings.TrimSuffix(strings.TrimSpace(line), ":")
			p = strings.NewReplacer("{key}", "job", "{id}", uuid.NewString(), "{record_id}", uuid.NewString()).Replace(p)
			for _, m := range []string{http.MethodGet, http.MethodPut, http.MethodPost, http.MethodPatch} {
				ops = append(ops, op{m, p})
			}
		}
	}
	if len(ops) < 20 {
		t.Fatalf("read only %d operations from the OpenAPI document", len(ops))
	}
	payload := `{"rules": [], "max_document_bytes": ` + needle + `, "NAVA_ACL_MAX_DOCUMENT_BYTES": ` + needle + `}`
	for _, o := range ops {
		rec := e.access(t, o.method, o.path, "admin-a", "expose-"+o.method+o.path, `W/"1"`, payload)
		if carriesNeedle(rec.Body.Bytes(), needle) {
			t.Errorf("%s %s = %d carries the limit: %s", o.method, o.path, rec.Code, rec.Body)
		}
	}
	// The only place it appears is the document_too_large detail.
	rec := e.access(t, http.MethodPut, accessPath, "admin-a", "expose-big", `W/"1"`, `{"rules": `+padded(t, limit+1)+`}`)
	wantProblem(t, rec, http.StatusUnprocessableEntity, problem.InvalidAccessPolicy)
	if !strings.Contains(rec.Body.String(), needle) {
		t.Errorf("the document_too_large 422 does not state the limit: %s", rec.Body)
	}
	// A successful read or publish never carries it.
	for _, rec := range []*httptest.ResponseRecorder{
		e.access(t, http.MethodGet, accessPath, "admin-a", "", "", ""),
		e.access(t, http.MethodPut, accessPath, "admin-a", "expose-ok", `W/"1"`, `{"rules": []}`),
	} {
		if carriesNeedle(rec.Body.Bytes(), needle) {
			t.Errorf("response carries the limit: %s", rec.Body)
		}
	}
}

// idKeys are the response members that carry a random request, trace or
// span id (problem+json `instance` is urn:nava:request:<hex>): a five-digit
// needle turns up inside such an id on a measurable share of runs (TICK-1102).
var idKeys = map[string]bool{
	"instance": true, "request_id": true, "requestId": true,
	"trace_id": true, "traceId": true, "span_id": true, "spanId": true, "traceparent": true,
}

// carriesNeedle reports whether body contains needle once every random id is
// removed: idKeys members are dropped from a JSON body, and in any body each
// UUID and urn:nava:request id is blanked, since a route may echo the random
// {id} of its path. What is left is matched as text.
func carriesNeedle(body []byte, needle string) bool {
	text := string(body)
	var v any
	if json.Unmarshal(body, &v) == nil {
		b, err := json.Marshal(stripIDs(v))
		if err == nil {
			text = string(b)
		}
	}
	text = randomID.ReplaceAllString(text, "<id>")
	return strings.Contains(text, needle)
}

var randomID = regexp.MustCompile(`(?i)urn:nava:request:[0-9a-z-]*|[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)

func stripIDs(v any) any {
	switch x := v.(type) {
	case map[string]any:
		for k, e := range x {
			if idKeys[k] {
				delete(x, k)
				continue
			}
			x[k] = stripIDs(e)
		}
	case []any:
		for i, e := range x {
			x[i] = stripIDs(e)
		}
	}
	return v
}

func TestCarriesNeedleIgnoresRandomIDs(t *testing.T) {
	for _, c := range []struct {
		body string
		want bool
	}{
		// Run 45024439126 attempt 1: the 405's instance held 54321.
		{`{"status": 405, "instance": "urn:nava:request:a62e543214374ceb"}`, false},
		{`{"detail": "no such record a62e5432-1437-4ceb-8f00-000000000000"}`, false},
		{`{"trace_id": "0054321", "errors": [{"request_id": "54321"}]}`, false},
		{`404 page not found urn:nava:request:a62e543214374ceb`, false},
		{`{"status": 422, "detail": "the document is over 54321 bytes", "instance": "urn:nava:request:1"}`, true},
		{`{"settings": {"max_document_bytes": 54321}}`, true},
		{`limit 54321`, true},
	} {
		if got := carriesNeedle([]byte(c.body), "54321"); got != c.want {
			t.Errorf("carriesNeedle(%s) = %v, want %v", c.body, got, c.want)
		}
	}
}

// A format-2 version the route stored is read back through the records path:
// the reader decodes it, enforces it, and the explainer names its policy.
// covers: putEntityTypeAccess 200
func TestPutAccessFormat2VersionIsEnforcedOnTheRecordsPath(t *testing.T) {
	e := newPatchEnv(t)
	rec := e.mustCreateRec(t, "user-a", "job", "f2-records", `{"short_description": "one"}`)
	doc := `{"format": 2, "conditions": [{"name": "desk", "expr": {"member_of": {"users": [` + uuidList(e.a.admin.PrincipalID, e.a.user.PrincipalID) + `]}}}],
		"policies": [{"name": "desk_reads", "ops": ["read", "write"], "when": {"ref": "desk"}}]}`
	if put := e.access(t, http.MethodPut, accessPath, "admin-a", "f2-records-put", `W/"1"`, `{"rules": `+doc+`}`); put.Code != http.StatusOK {
		t.Fatalf("publish = %d: %s", put.Code, put.Body)
	}
	if typ, _ := e.storedFormat(t, 2); typ != "object" {
		t.Fatalf("stored %s, want an object", typ)
	}
	if r := e.do(t, http.MethodGet, "/api/v1/records/job/"+rec.ID, "user-a", "", ""); r.Code != http.StatusOK {
		t.Errorf("a user the policy names reads the record = %d: %s", r.Code, r.Body)
	}
	if r := e.do(t, http.MethodGet, "/api/v1/records/job/"+rec.ID, "lookalike-a", "", ""); r.Code != http.StatusNotFound {
		t.Errorf("a user it does not name reads the record = %d, want 404", r.Code)
	}
	x := decodeExplanation(t, e.explain(t, "admin-a", "job", rec.ID, e.a.user.PrincipalID.String()))
	if x.Decision != "allow" || x.RuleID == nil || *x.RuleID != "desk_reads" || x.ACLVersion != 2 {
		t.Errorf("explain = %+v, want allow by desk_reads at version 2", x)
	}
}

// PublishIfNone (seed and onboarding) refuses what the route refuses, and
// writes no version and no audit entry.
func TestPublishIfNoneRefusesWhatTheRouteRefuses(t *testing.T) {
	e := newPatchEnv(t)
	e.noTestACL = true
	if r := e.do(t, http.MethodPost, "/api/v1/entity-types", "admin-a", "pin-widget", etBody("widget", "WGT")); r.Code != http.StatusCreated {
		t.Fatalf("create widget = %d: %s", r.Code, r.Body)
	}
	publish := func(rules []authz.Rule, masks []authz.Mask) (bool, error) {
		var done bool
		err := tenancy.InTenantTx(t.Context(), e.pool, e.a.id, func(tx pgx.Tx) error {
			var err error
			done, err = authz.PolicyStore{}.PublishIfNone(t.Context(), tx, "widget", authz.Publisher{System: "onboarding", Source: audit.SourceOnboarding}, rules, masks)
			return err
		})
		return done, err
	}
	ok := authz.Rule{ID: "ok", Ops: []string{authz.OpRead}, Who: authz.Subject{Groups: []uuid.UUID{e.a.adminGroup}}}
	for _, tc := range []struct {
		name  string
		rules []authz.Rule
		masks []authz.Mask
		want  string // "pointer rule"
	}{
		{"empty who", []authz.Rule{{ID: "r", Ops: []string{authz.OpRead}}}, nil, "/rules/0/who child_count"},
		{"a mask", []authz.Rule{ok}, []authz.Mask{{Field: "note", Read: authz.Subject{Groups: []uuid.UUID{e.a.adminGroup}}}}, "/masks value"},
		{"another tenant's group", []authz.Rule{{ID: "r", Ops: []string{authz.OpRead}, Who: authz.Subject{Groups: []uuid.UUID{e.b.adminGroup}}}}, nil, "/rules/0/who/groups/0 foreign_id"},
		{"a group of no tenant", []authz.Rule{{ID: "r", Ops: []string{authz.OpRead}, Who: authz.Subject{Groups: []uuid.UUID{uuid.MustParse(foreignGroup)}}}}, nil, "/rules/0/who/groups/0 foreign_id"},
	} {
		versions, entries := e.aclCounts(t)
		done, err := publish(tc.rules, tc.masks)
		var bad *authz.InvalidPolicyError
		if done || !errors.As(err, &bad) {
			t.Errorf("%s: PublishIfNone = %v, %v, want an *InvalidPolicyError", tc.name, done, err)
			continue
		}
		var got []string
		for _, p := range bad.Problems {
			got = append(got, p.Pointer+" "+p.Rule)
		}
		if !containsStr(got, tc.want) {
			t.Errorf("%s: problems %v, want %q", tc.name, got, tc.want)
		}
		if v, n := e.aclCounts(t); v != versions || n != entries {
			t.Errorf("%s: wrote versions %d -> %d, entries %d -> %d", tc.name, versions, v, entries, n)
		}
	}
	// A valid policy publishes once, and a second call is a no-op.
	if done, err := publish([]authz.Rule{ok}, nil); err != nil || !done {
		t.Fatalf("valid publish = %v, %v", done, err)
	}
	if done, err := publish([]authz.Rule{ok}, nil); err != nil || done {
		t.Errorf("second publish = %v, %v, want false", done, err)
	}
}

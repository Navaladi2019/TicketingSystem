//go:build integration

// P3-14 (TICK-357): POST and GET /api/v1/records/{entity}[/{id}] against a
// real PostgreSQL, through the real handler (authn, ratelimit, idempotency,
// validator, strict server, record write pipeline).

package server_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
)

const recSchema = `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "additionalProperties": false,
  "required": ["short_description", "state"],
  "properties": {
    "short_description": {"type": "string", "minLength": 1, "maxLength": 160},
    "state": {"type": "string", "enum": ["new", "closed"], "default": "new"},
    "group": {"type": "string", "format": "uuid"},
    "assignee": {"type": "string", "format": "uuid"},
    "requester": {"type": "string", "format": "uuid"}
  }
}`

func recTypeBody(key, prefix string) string {
	return fmt.Sprintf(`{"key": %q, "name": "Type %s", "number_prefix": %q, "definition": {"schema": %s,
  "fixed_columns": {"state": "state", "group": "assignment_group_id", "assignee": "assignee_id", "requester": "requester_id"},
  "states": {"initial": "new", "states": ["new", "closed"], "transitions": [{"from": "new", "to": "closed"}]}}}`,
		key, key, prefix, recSchema)
}

// recEnv is the handler with entity type "ticket" in tenant A and "incident"
// in tenant B only.
func newRecEnv(t *testing.T) *etEnv {
	t.Helper()
	e := newETEnv(t)
	if rec := e.do(t, http.MethodPost, "/api/v1/entity-types", "admin-a", "t-a", recTypeBody("ticket", "TKT")); rec.Code != http.StatusCreated {
		t.Fatalf("seed tenant A type = %d: %s", rec.Code, rec.Body)
	}
	if rec := e.do(t, http.MethodPost, "/api/v1/entity-types", "admin-a", "t-a2", recTypeBody("change", "CHG")); rec.Code != http.StatusCreated {
		t.Fatalf("seed tenant A second type = %d: %s", rec.Code, rec.Body)
	}
	if rec := e.do(t, http.MethodPost, "/api/v1/entity-types", "admin-b", "t-b", recTypeBody("incident", "INC")); rec.Code != http.StatusCreated {
		t.Fatalf("seed tenant B type = %d: %s", rec.Code, rec.Body)
	}
	return e
}

type recBody struct {
	ID            string         `json:"id"`
	Number        string         `json:"number"`
	Version       int64          `json:"version"`
	SchemaVersion int            `json:"schema_version"`
	State         string         `json:"state"`
	Data          map[string]any `json:"data"`
}

func (e *etEnv) createRec(t *testing.T, token, entity, key, data string) *httptest.ResponseRecorder {
	t.Helper()
	return e.do(t, http.MethodPost, "/api/v1/records/"+entity, token, key, `{"data": `+data+`}`)
}

func (e *etEnv) mustCreateRec(t *testing.T, token, entity, key, data string) recBody {
	t.Helper()
	rec := e.createRec(t, token, entity, key, data)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create = %d: %s", rec.Code, rec.Body)
	}
	var b recBody
	if err := json.Unmarshal(rec.Body.Bytes(), &b); err != nil {
		t.Fatal(err)
	}
	return b
}

func (e *etEnv) recordCount(t *testing.T, tenant string) int {
	t.Helper()
	var n int
	id := e.a.id
	if tenant == "b" {
		id = e.b.id
	}
	if err := e.super.QueryRow(t.Context(), "SELECT count(*) FROM records WHERE tenant_id = $1", id).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// withoutInstance parses a problem body and drops the per-request instance.
func withoutInstance(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &m); err != nil {
		t.Fatalf("body is not JSON: %v: %s", err, rec.Body)
	}
	delete(m, "instance")
	out, _ := json.Marshal(m)
	return string(out)
}

func TestCreateRecordThenReadIt(t *testing.T) {
	e := newRecEnv(t)
	rec := e.createRec(t, "user-a", "ticket", "r1", `{"short_description": "Printer on fire"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create = %d: %s", rec.Code, rec.Body)
	}
	var b recBody
	if err := json.Unmarshal(rec.Body.Bytes(), &b); err != nil {
		t.Fatal(err)
	}
	if b.Version != 1 || b.State != "new" || b.SchemaVersion != 1 || b.Data["state"] != "new" || b.Data["short_description"] != "Printer on fire" {
		t.Errorf("create body = %+v, want version 1, state new (default), schema version 1", b)
	}
	if got := rec.Header().Get("ETag"); got != `W/"1"` {
		t.Errorf("ETag = %q, want W/\"1\"", got)
	}
	if got, want := rec.Header().Get("Location"), "/api/v1/records/ticket/"+b.ID; got != want {
		t.Errorf("Location = %q, want %q", got, want)
	}
	got := e.do(t, http.MethodGet, "/api/v1/records/ticket/"+b.ID, "admin-a", "", "")
	if got.Code != http.StatusOK || got.Header().Get("ETag") != `W/"1"` {
		t.Fatalf("GET = %d ETag %q: %s", got.Code, got.Header().Get("ETag"), got.Body)
	}
	var r recBody
	if err := json.Unmarshal(got.Body.Bytes(), &r); err != nil {
		t.Fatal(err)
	}
	if r.ID != b.ID || r.Number != b.Number || r.Data["short_description"] != "Printer on fire" || r.State != "new" {
		t.Errorf("GET body = %+v, want the created record %+v", r, b)
	}
}

func TestCreateRecordRetryWithTheSameKeyCreatesOne(t *testing.T) {
	e := newRecEnv(t)
	first := e.createRec(t, "user-a", "ticket", "same-key", `{"short_description": "once"}`)
	if first.Code != http.StatusCreated {
		t.Fatalf("create = %d: %s", first.Code, first.Body)
	}
	again := e.createRec(t, "user-a", "ticket", "same-key", `{"short_description": "once"}`)
	if again.Code != http.StatusCreated || again.Body.String() != first.Body.String() || again.Header().Get("Idempotent-Replayed") != "true" {
		t.Errorf("retry = %d replayed=%q, want the same 201 body; got:\n%s\nwant:\n%s", again.Code, again.Header().Get("Idempotent-Replayed"), again.Body, first.Body)
	}
	if again.Header().Get("ETag") != first.Header().Get("ETag") || again.Header().Get("Location") != first.Header().Get("Location") {
		t.Errorf("retry headers differ: ETag %q/%q Location %q/%q", again.Header().Get("ETag"), first.Header().Get("ETag"),
			again.Header().Get("Location"), first.Header().Get("Location"))
	}
	if n := e.recordCount(t, "a"); n != 1 {
		t.Errorf("tenant A has %d records, want exactly 1", n)
	}
	// The key with another body is a conflict and writes nothing.
	wantProblem(t, e.createRec(t, "user-a", "ticket", "same-key", `{"short_description": "other"}`), http.StatusConflict, problem.IdempotencyKeyReuse)
	if n := e.recordCount(t, "a"); n != 1 {
		t.Errorf("tenant A has %d records after the conflicting retry, want 1", n)
	}
}

func TestRecordFourNotFoundCasesAreByteIdentical(t *testing.T) {
	e := newRecEnv(t)
	mine := e.mustCreateRec(t, "user-a", "ticket", "mine", `{"short_description": "a"}`)
	other := e.mustCreateRec(t, "user-b", "incident", "theirs", `{"short_description": "b"}`)
	missing := "0194f2c0-0000-7000-8000-000000000000"

	cases := map[string]*httptest.ResponseRecorder{
		"missing id":                  e.do(t, http.MethodGet, "/api/v1/records/ticket/"+missing, "user-a", "", ""),
		"other tenant's record":       e.do(t, http.MethodGet, "/api/v1/records/incident/"+other.ID, "user-a", "", ""),
		"other tenant, own type key":  e.do(t, http.MethodGet, "/api/v1/records/ticket/"+other.ID, "user-a", "", ""),
		"wrong entity":                e.do(t, http.MethodGet, "/api/v1/records/change/"+mine.ID, "user-a", "", ""),
		"unknown entity type":         e.do(t, http.MethodGet, "/api/v1/records/no_such_type/"+mine.ID, "user-a", "", ""),
		"id not a uuid":               e.do(t, http.MethodGet, "/api/v1/records/ticket/not-a-uuid", "user-a", "", ""),
		"POST unknown entity type":    e.createRec(t, "user-a", "no_such_type", "p1", `{"short_description": "x"}`),
		"POST other tenant's type":    e.createRec(t, "user-a", "incident", "p2", `{"short_description": "x"}`),
		"POST unknown type, bad body": e.do(t, http.MethodPost, "/api/v1/records/no_such_type", "user-a", "p3", `{"nope": 1}`),
	}
	var want string
	for name, rec := range cases {
		if rec.Code != http.StatusNotFound {
			t.Errorf("%s = %d, want 404: %s", name, rec.Code, rec.Body)
			continue
		}
		if strings.Contains(rec.Body.String(), "instance") {
			t.Errorf("%s: the 404 body carries instance: %s", name, rec.Body)
		}
		if want == "" {
			want = rec.Body.String()
		}
		if rec.Body.String() != want {
			t.Errorf("%s: body\n%s\ndiffers from\n%s", name, rec.Body, want)
		}
	}
	wantProblem(t, cases["missing id"], http.StatusNotFound, problem.NotFound)
	if n := e.recordCount(t, "a"); n != 1 {
		t.Errorf("tenant A has %d records, want 1: a 404 create writes nothing", n)
	}
}

func TestCreateRecordValidationAndAuth(t *testing.T) {
	e := newRecEnv(t)
	p := wantProblem(t, e.createRec(t, "user-a", "ticket", "v1", `{}`), http.StatusUnprocessableEntity, problem.RecordInvalid)
	if got := pointers(p); len(got) != 1 || got[0] != "/data/short_description" {
		t.Errorf("pointers = %v, want [/data/short_description]", got)
	}
	p = wantProblem(t, e.do(t, http.MethodPost, "/api/v1/records/ticket", "user-a", "v2", `{"other": 1}`), http.StatusUnprocessableEntity, problem.RecordInvalid)
	if got := pointers(p); len(got) != 1 || got[0] != "/data" {
		t.Errorf("pointers = %v, want [/data]", got)
	}
	wantProblem(t, e.do(t, http.MethodPost, "/api/v1/records/ticket", "user-a", "v3", `{"data": 5}`), http.StatusUnprocessableEntity, problem.RecordInvalid)
	wantProblem(t, e.createRec(t, "", "ticket", "v4", `{"short_description": "x"}`), http.StatusUnauthorized, problem.InvalidToken)
	wantProblem(t, e.createRec(t, "user-a", "ticket", "", `{"short_description": "x"}`), http.StatusBadRequest, problem.ValidationFailed)
	wantProblem(t, e.do(t, http.MethodGet, "/api/v1/records/ticket/"+"0194f2c0-0000-7000-8000-000000000000", "", "", ""), http.StatusUnauthorized, problem.InvalidToken)
	if n := e.recordCount(t, "a"); n != 0 {
		t.Errorf("tenant A has %d records, want 0: a refused create writes nothing", n)
	}
}

func TestCreateRecordForeignGroupIsTheSame422AsARandomUUID(t *testing.T) {
	e := newRecEnv(t)
	random := "00000000-0000-4000-8000-00000000dead"
	foreign := e.b.adminGroup.String() // tenant B's group, named by a tenant A user
	rr := e.createRec(t, "user-a", "ticket", "g1", fmt.Sprintf(`{"short_description": "x", "group": %q}`, random))
	rf := e.createRec(t, "user-a", "ticket", "g2", fmt.Sprintf(`{"short_description": "x", "group": %q}`, foreign))
	p := wantProblem(t, rr, http.StatusUnprocessableEntity, problem.RecordInvalid)
	wantProblem(t, rf, http.StatusUnprocessableEntity, problem.RecordInvalid)
	if got := pointers(p); len(got) != 1 || got[0] != "/data/group" {
		t.Errorf("pointers = %v, want [/data/group]", got)
	}
	if a, b := withoutInstance(t, rr), withoutInstance(t, rf); a != b {
		t.Errorf("foreign group body differs from the random UUID's:\n%s\n%s", b, a)
	}
	for _, rec := range []*httptest.ResponseRecorder{rr, rf} {
		for _, leak := range []string{"fkey", "constraint", "violates", "foreign key", "records", foreign, random} {
			if strings.Contains(rec.Body.String(), leak) {
				t.Errorf("the 422 body shows %q: %s", leak, rec.Body)
			}
		}
	}
	if n := e.recordCount(t, "a"); n != 0 {
		t.Errorf("tenant A has %d records, want 0", n)
	}
}

// A fixed uuid column naming another tenant's principal or an unknown
// principal comes back from pipeline.Write as a wrapped 23503 (P3-10 review).
// It is a 422 field error on that property and never shows the constraint.
func TestCreateRecordForeignPrincipalIs422OnThatProperty(t *testing.T) {
	e := newRecEnv(t)
	random := "00000000-0000-4000-8000-00000000beef"
	foreign := e.b.user.PrincipalID.String()
	for _, prop := range []string{"assignee", "requester"} {
		var bodies []string
		for i, id := range []string{random, foreign} {
			rec := e.createRec(t, "user-a", "ticket", fmt.Sprintf("p-%s-%d", prop, i), fmt.Sprintf(`{"short_description": "x", %q: %q}`, prop, id))
			p := wantProblem(t, rec, http.StatusUnprocessableEntity, problem.RecordInvalid)
			if got := pointers(p); len(got) != 1 || got[0] != "/data/"+prop || p.Errors[0].Rule != "reference" {
				t.Errorf("%s: errors = %+v, want one error on /data/%s with rule reference", prop, p.Errors, prop)
			}
			for _, leak := range []string{"fkey", "constraint", "violates", "foreign key", "principals", id} {
				if strings.Contains(rec.Body.String(), leak) {
					t.Errorf("%s: the 422 body shows %q: %s", prop, leak, rec.Body)
				}
			}
			bodies = append(bodies, withoutInstance(t, rec))
		}
		if bodies[0] != bodies[1] {
			t.Errorf("%s: foreign principal body differs from the unknown one's:\n%s\n%s", prop, bodies[1], bodies[0])
		}
	}
	// Own tenant's principal and group are accepted, and come back on read.
	b := e.mustCreateRec(t, "user-a", "ticket", "ok", fmt.Sprintf(`{"short_description": "x", "assignee": %q, "group": %q}`,
		e.a.user.PrincipalID, e.a.adminGroup))
	if b.Data["assignee"] != e.a.user.PrincipalID.String() || b.Data["group"] != e.a.adminGroup.String() {
		t.Errorf("data = %v, want the assignee and group read back from their columns", b.Data)
	}
}

// A record written under version 1 reads with the defaults of the current
// version: version 2 adds `priority` with a default, and the old record shows
// it on GET while keeping its own schema_version and ETag.
func TestGetRecordAppliesDefaultsOfTheCurrentSchemaVersion(t *testing.T) {
	e := newRecEnv(t)
	b := e.mustCreateRec(t, "user-a", "ticket", "d1", `{"short_description": "old"}`)
	if _, has := b.Data["priority"]; has || b.SchemaVersion != 1 {
		t.Fatalf("created record = %+v, want schema version 1 and no priority", b)
	}
	v2 := strings.Replace(recSchema, `"requester": {"type": "string", "format": "uuid"}`,
		`"requester": {"type": "string", "format": "uuid"},
    "priority": {"type": "string", "enum": ["low", "high"], "default": "low"}`, 1)
	body := fmt.Sprintf(`{"definition": {"schema": %s,
  "fixed_columns": {"state": "state", "group": "assignment_group_id", "assignee": "assignee_id", "requester": "requester_id"},
  "states": {"initial": "new", "states": ["new", "closed"], "transitions": [{"from": "new", "to": "closed"}]}}}`, v2)
	if rec := e.do(t, http.MethodPost, "/api/v1/entity-types/ticket/versions", "admin-a", "pub-2", body); rec.Code != http.StatusCreated {
		t.Fatalf("publish v2 = %d: %s", rec.Code, rec.Body)
	}
	got := e.do(t, http.MethodGet, "/api/v1/records/ticket/"+b.ID, "user-a", "", "")
	if got.Code != http.StatusOK {
		t.Fatalf("GET = %d: %s", got.Code, got.Body)
	}
	var r recBody
	if err := json.Unmarshal(got.Body.Bytes(), &r); err != nil {
		t.Fatal(err)
	}
	if r.Data["priority"] != "low" || r.SchemaVersion != 1 || r.Version != 1 || got.Header().Get("ETag") != `W/"1"` {
		t.Errorf("GET = %+v ETag %q, want priority low (default), schema_version 1, version 1", r, got.Header().Get("ETag"))
	}
	// The default is a read-time view: the stored row is unchanged.
	var stored string
	if err := e.super.QueryRow(t.Context(), "SELECT data::text FROM records WHERE tenant_id = $1 AND id = $2", e.a.id, b.ID).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(stored, "priority") {
		t.Errorf("stored data %s holds the read-time default", stored)
	}
}

// Causation (origin, depth) never comes from the request (TICK-88 security
// review): a client that could set it could suppress its own tenant's
// automation or flood audit_log with cascade_rejected rows. The handler builds
// the pipeline command with an empty Causation, so a create that tries to
// carry one in the body, the query or the headers is a depth-0 write whose
// origin is not the one the client named.
func TestCreateRecordIgnoresCausationFromTheRequest(t *testing.T) {
	e := newRecEnv(t)
	forged := "0194f2c0-0000-7000-8000-00000000f0f0"
	body := fmt.Sprintf(`{"data": {"short_description": "x"}, "causation": {"origin_id": %q, "depth": 9},
  "causation_origin_id": %q, "causation_depth": 9, "origin_id": %q, "depth": 9}`, forged, forged, forged)
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost,
		"/api/v1/records/ticket?causation_origin_id="+forged+"&causation_depth=9&depth=9", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer user-a")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "caus-1")
	for _, h := range []string{"X-Causation-Origin-Id", "X-Causation-Origin", "Causation-Origin-Id", "X-Origin-Id"} {
		req.Header.Set(h, forged)
	}
	for _, h := range []string{"X-Causation-Depth", "Causation-Depth", "X-Depth"} {
		req.Header.Set(h, "9")
	}
	rec := httptest.NewRecorder()
	e.h.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create = %d: %s", rec.Code, rec.Body)
	}
	var b recBody
	if err := json.Unmarshal(rec.Body.Bytes(), &b); err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{
		"SELECT causation_depth, causation_origin_id::text FROM records WHERE tenant_id = $1 AND id = $2",
		"SELECT causation_depth, causation_origin_id::text FROM audit_log WHERE tenant_id = $1 AND record_id = $2",
	} {
		var (
			depth  *int16
			origin *string
		)
		if err := e.super.QueryRow(t.Context(), q, e.a.id, b.ID).Scan(&depth, &origin); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
		if depth == nil || *depth != 0 {
			t.Errorf("%s: causation_depth = %v, want 0", q, depth)
		}
		if origin != nil && *origin == forged {
			t.Errorf("%s: causation_origin_id is the origin the request named", q)
		}
	}
	// The request's causation is not echoed either.
	if strings.Contains(rec.Body.String(), forged) {
		t.Errorf("the response carries the request's causation origin: %s", rec.Body)
	}
}

//go:build integration

// P3-15 (TICK-358): PATCH /api/v1/records/{entity}/{id} and its history
// against a real PostgreSQL, through the real handler.

package server_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// patchSchema: short_description is required with no default, state is
// required with a default, note is optional, tags is an array, code is unique.
const patchSchema = `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "additionalProperties": false,
  "required": ["short_description", "state"],
  "properties": {
    "short_description": {"type": "string", "minLength": 1, "maxLength": 160},
    "state": {"type": "string", "enum": ["new", "closed"], "default": "new"},
    "note": {"type": "string"},
    "tags": {"type": "array", "items": {"type": "string"}},
    "code": {"type": "string", "x-nava-unique": true}
  }
}`

func patchTypeBody(key, prefix string) string {
	return fmt.Sprintf(`{"key": %q, "name": "Type %s", "number_prefix": %q, "definition": {"schema": %s,
  "fixed_columns": {"state": "state"},
  "states": {"initial": "new", "states": ["new", "closed"], "transitions": [{"from": "new", "to": "closed"}]}}}`,
		key, key, prefix, patchSchema)
}

func newPatchEnv(t *testing.T) *etEnv {
	t.Helper()
	e := newETEnv(t)
	if rec := e.do(t, http.MethodPost, "/api/v1/entity-types", "admin-a", "pt-a", patchTypeBody("job", "JOB")); rec.Code != http.StatusCreated {
		t.Fatalf("seed type = %d: %s", rec.Code, rec.Body)
	}
	if rec := e.do(t, http.MethodPost, "/api/v1/entity-types", "admin-b", "pt-b", patchTypeBody("job", "JOB")); rec.Code != http.StatusCreated {
		t.Fatalf("seed tenant B type = %d: %s", rec.Code, rec.Body)
	}
	return e
}

var patchKey int

// patch sends a merge patch. ifMatch "" sends no If-Match header.
func (e *etEnv) patch(t *testing.T, token, entity, id, ifMatch, body string) *httptest.ResponseRecorder {
	t.Helper()
	patchKey++
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPatch, "/api/v1/records/"+entity+"/"+id, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/merge-patch+json")
	req.Header.Set("Idempotency-Key", fmt.Sprintf("patch-%d", patchKey))
	if ifMatch != "" {
		req.Header.Set("If-Match", ifMatch)
	}
	rec := httptest.NewRecorder()
	e.h.ServeHTTP(rec, req)
	return rec
}

func (e *etEnv) auditCount(t *testing.T, id string) int {
	t.Helper()
	var n int
	if err := e.super.QueryRow(t.Context(), "SELECT count(*) FROM audit_log_v0 WHERE tenant_id = $1 AND record_id = $2", e.a.id, id).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func patchProblemCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var p struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
		t.Fatalf("not a problem: %v: %s", err, rec.Body)
	}
	return p.Code
}

func TestPatchRecordNeedsTheCurrentETag(t *testing.T) {
	e := newPatchEnv(t)
	r := e.mustCreateRec(t, "user-a", "job", "c1", `{"short_description": "one"}`)
	path := r.ID

	rec := e.patch(t, "user-a", "job", path, `W/"1"`, `{"data": {"note": "n1"}}`)
	if rec.Code != http.StatusOK || rec.Header().Get("ETag") != `W/"2"` {
		t.Fatalf("patch with the current ETag = %d ETag %q: %s", rec.Code, rec.Header().Get("ETag"), rec.Body)
	}
	var b recBody
	if err := json.Unmarshal(rec.Body.Bytes(), &b); err != nil || b.Version != 2 || b.Data["note"] != "n1" || b.Data["short_description"] != "one" {
		t.Fatalf("patch body = %+v (%v)", b, err)
	}
	audits := e.auditCount(t, path)

	// The old ETag: 412 with the current ETag in the header and the detail.
	rec = e.patch(t, "user-a", "job", path, `W/"1"`, `{"data": {"note": "n2"}}`)
	if rec.Code != http.StatusPreconditionFailed || patchProblemCode(t, rec) != "version_conflict" {
		t.Fatalf("stale = %d: %s", rec.Code, rec.Body)
	}
	if got := rec.Header().Get("ETag"); got != `W/"2"` {
		t.Errorf("412 ETag = %q, want the current W/\"2\"", got)
	}
	if !strings.Contains(rec.Body.String(), `W/\"2\"`) || strings.Contains(rec.Body.String(), "n1") || strings.Contains(rec.Body.String(), "short_description") {
		t.Errorf("412 body = %s, want the current ETag in detail and no record data", rec.Body)
	}
	// No If-Match: 428. A star counts as missing.
	for name, im := range map[string]string{"missing": "", "star": "*"} {
		rec = e.patch(t, "user-a", "job", path, im, `{"data": {"note": "n3"}}`)
		if rec.Code != http.StatusPreconditionRequired || patchProblemCode(t, rec) != "if_match_required" {
			t.Errorf("%s If-Match = %d: %s", name, rec.Code, rec.Body)
		}
	}
	// Nothing changed and no audit row was added.
	if got := e.auditCount(t, path); got != audits {
		t.Errorf("audit rows = %d, want %d after a 412 and two 428s", got, audits)
	}
	got := e.do(t, http.MethodGet, "/api/v1/records/job/"+path, "user-a", "", "")
	if got.Header().Get("ETag") != `W/"2"` || !strings.Contains(got.Body.String(), `"n1"`) {
		t.Errorf("record after refusals = %s ETag %q", got.Body, got.Header().Get("ETag"))
	}
}

func TestPatchMissingRecordIsA404NotA428(t *testing.T) {
	e := newPatchEnv(t)
	r := e.mustCreateRec(t, "user-a", "job", "c1", `{"short_description": "one"}`)
	const missing = "0194f2c0-0000-7000-8000-000000000000"
	for name, c := range map[string][2]string{
		"missing id":     {"job", missing},
		"unknown type":   {"no_such_type", r.ID},
		"not a uuid":     {"job", "nope"},
		"tenant B's way": {"job", r.ID}, // checked below with B's token
	} {
		token := "user-a"
		if name == "tenant B's way" {
			token = "user-b"
		}
		rec := e.patch(t, token, c[0], c[1], "", `{"data": {"note": "x"}}`)
		if rec.Code != http.StatusNotFound {
			t.Errorf("%s without If-Match = %d, want 404: %s", name, rec.Code, rec.Body)
		}
		if rec = e.patch(t, token, c[0], c[1], `W/"1"`, `{"data": {"note": "x"}}`); rec.Code != http.StatusNotFound {
			t.Errorf("%s with If-Match = %d, want 404: %s", name, rec.Code, rec.Body)
		}
	}
	if got := e.auditCount(t, r.ID); got != 1 {
		t.Errorf("audit rows = %d, want only the create", got)
	}
}

func TestPatchMergePatchSemantics(t *testing.T) {
	e := newPatchEnv(t)
	r := e.mustCreateRec(t, "user-a", "job", "c1", `{"short_description": "one", "note": "n", "tags": ["a", "b"]}`)
	v := int64(1)
	etag := func() string { return fmt.Sprintf(`W/"%d"`, v) }
	send := func(body string, want int) *httptest.ResponseRecorder {
		t.Helper()
		rec := e.patch(t, "user-a", "job", r.ID, etag(), body)
		if rec.Code != want {
			t.Fatalf("PATCH %s = %d, want %d: %s", body, rec.Code, want, rec.Body)
		}
		if want == http.StatusOK {
			v++
		}
		return rec
	}
	send(`{"data": {"short_description": null}}`, http.StatusUnprocessableEntity) // required, no default
	send(`{"data": {"state": null}}`, http.StatusOK)                              // required, default: refilled
	rec := send(`{"data": {"note": null}}`, http.StatusOK)                        // optional: removed
	var b recBody
	if err := json.Unmarshal(rec.Body.Bytes(), &b); err != nil {
		t.Fatal(err)
	}
	if _, has := b.Data["note"]; has || b.Data["state"] != "new" {
		t.Errorf("data = %v, want note removed and state refilled", b.Data)
	}
	rec = send(`{"data": {"tags": ["z"]}}`, http.StatusOK) // arrays replace whole
	if err := json.Unmarshal(rec.Body.Bytes(), &b); err != nil {
		t.Fatal(err)
	}
	if tags, _ := b.Data["tags"].([]any); len(tags) != 1 || tags[0] != "z" {
		t.Errorf("tags = %v, want [z]", b.Data["tags"])
	}
	// The body shape.
	send(`{}`, http.StatusUnprocessableEntity)
	send(`{"data": "x"}`, http.StatusUnprocessableEntity)
	send(`{"data": null}`, http.StatusUnprocessableEntity)
	send(`[]`, http.StatusBadRequest) // the request validator refuses a non-object body
	send(`{"state": 3}`, http.StatusUnprocessableEntity)
	if rec := send(`{"data": {"nope": 1}}`, http.StatusUnprocessableEntity); patchProblemCode(t, rec) != "record_invalid" {
		t.Errorf("code = %s, want record_invalid", patchProblemCode(t, rec))
	}
}

func TestPatchUniqueViolationAndIllegalTransition(t *testing.T) {
	e := newPatchEnv(t)
	e.mustCreateRec(t, "user-a", "job", "c1", `{"short_description": "one", "code": "X-1"}`)
	r2 := e.mustCreateRec(t, "user-a", "job", "c2", `{"short_description": "two", "code": "X-2"}`)

	rec := e.patch(t, "user-a", "job", r2.ID, `W/"1"`, `{"data": {"code": "X-1"}}`)
	if rec.Code != http.StatusConflict || patchProblemCode(t, rec) != "unique_violation" || !strings.Contains(rec.Body.String(), "/data/code") {
		t.Errorf("unique = %d: %s", rec.Code, rec.Body)
	}
	// A top-level state: new -> closed is allowed, closed -> new is not.
	rec = e.patch(t, "user-a", "job", r2.ID, `W/"1"`, `{"state": "closed"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("close = %d: %s", rec.Code, rec.Body)
	}
	rec = e.patch(t, "user-a", "job", r2.ID, `W/"2"`, `{"state": "new"}`)
	if rec.Code != http.StatusUnprocessableEntity || patchProblemCode(t, rec) != "illegal_transition" {
		t.Errorf("illegal transition = %d: %s", rec.Code, rec.Body)
	}
}

func TestPatchHistoryPagesNewestFirst(t *testing.T) {
	e := newPatchEnv(t)
	r := e.mustCreateRec(t, "user-a", "job", "c1", `{"short_description": "one"}`)
	for i, v := range []string{"a", "b"} {
		if rec := e.patch(t, "user-a", "job", r.ID, fmt.Sprintf(`W/"%d"`, i+1), fmt.Sprintf(`{"data": {"note": %q}}`, v)); rec.Code != http.StatusOK {
			t.Fatalf("patch %d = %d: %s", i, rec.Code, rec.Body)
		}
	}
	type page struct {
		Items []struct {
			Action  string         `json:"action"`
			Source  string         `json:"source"`
			ActorID string         `json:"actor_id"`
			Changed []string       `json:"changed"`
			Before  map[string]any `json:"before"`
			After   map[string]any `json:"after"`
		} `json:"items"`
		Next string `json:"next_cursor"`
	}
	get := func(token, q string) (*httptest.ResponseRecorder, page) {
		rec := e.do(t, http.MethodGet, "/api/v1/records/job/"+r.ID+"/history"+q, token, "", "")
		var p page
		if rec.Code == http.StatusOK {
			if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
				t.Fatal(err)
			}
		}
		return rec, p
	}
	rec, all := get("user-a", "")
	if rec.Code != http.StatusOK || len(all.Items) != 3 {
		t.Fatalf("history = %d with %d rows: %s", rec.Code, len(all.Items), rec.Body)
	}
	if all.Items[0].Action != "update" || all.Items[0].After["note"] != "b" || all.Items[0].Before["note"] != "a" ||
		strings.Join(all.Items[0].Changed, ",") != "/data/note" || all.Items[0].Source != "user" || all.Items[0].ActorID == "" {
		t.Errorf("newest = %+v", all.Items[0])
	}
	if all.Items[2].Action != "create" || all.Items[2].Before != nil {
		t.Errorf("oldest = %+v, want the create with no before", all.Items[2])
	}
	// limit=1 walks the same three, newest first, no duplicates.
	var walked []string
	q, token := "?limit=1", ""
	var firstCursor string
	for range 5 {
		rec, p := get("user-a", q)
		if rec.Code != http.StatusOK || len(p.Items) != 1 {
			t.Fatalf("page = %d %s", rec.Code, rec.Body)
		}
		walked = append(walked, fmt.Sprint(p.Items[0].Action, p.Items[0].After["note"]))
		if p.Next == "" {
			break
		}
		if firstCursor == "" {
			firstCursor = p.Next
		}
		token = p.Next
		q = "?limit=1&cursor=" + token
	}
	if want := "updateb,updatea,create<nil>"; strings.Join(walked, ",") != want {
		t.Errorf("walk = %v, want %s", walked, want)
	}
	// A cursor of this record's history is refused for tenant B on the same
	// URL (the tenant claim), for another principal of tenant A, and for
	// another record of tenant A (the query hash).
	replay := func(token, id string) *httptest.ResponseRecorder {
		return e.do(t, http.MethodGet, "/api/v1/records/job/"+id+"/history?limit=1&cursor="+url.QueryEscape(firstCursor), token, "", "")
	}
	if rec := replay("user-a", r.ID); rec.Code != http.StatusOK {
		t.Fatalf("control: the issuing caller's replay = %d: %s", rec.Code, rec.Body)
	}
	other := e.mustCreateRec(t, "user-a", "job", "c-other", `{"short_description": "other"}`)
	for name, c := range map[string][2]string{
		"tenant B on the same record":       {"user-b", r.ID},
		"another principal of tenant A":     {"admin-a", r.ID},
		"another record of the same tenant": {"user-a", other.ID},
	} {
		if rec := replay(c[0], c[1]); rec.Code != http.StatusBadRequest || patchProblemCode(t, rec) != "invalid_cursor" {
			t.Errorf("%s: replay = %d, want 400 invalid_cursor: %s", name, rec.Code, rec.Body)
		}
	}
	// Tenant B reading tenant A's history gets the 404 of a missing record.
	rec, _ = get("user-b", "")
	missing := e.do(t, http.MethodGet, "/api/v1/records/job/0194f2c0-0000-7000-8000-000000000000/history", "user-b", "", "")
	if rec.Code != http.StatusNotFound || withoutInstance(t, rec) != withoutInstance(t, missing) {
		t.Errorf("tenant B history = %d %s, want the missing record's 404", rec.Code, rec.Body)
	}
}

// Causation never comes from the request (TICK-88 security review).
func TestPatchRecordIgnoresCausationFromTheRequest(t *testing.T) {
	e := newPatchEnv(t)
	r := e.mustCreateRec(t, "user-a", "job", "c1", `{"short_description": "one"}`)
	forged := "0194f2c0-0000-7000-8000-00000000f0f0"
	body := fmt.Sprintf(`{"data": {"note": "x"}, "causation": {"origin_id": %q, "depth": 9},
  "causation_origin_id": %q, "causation_depth": 9, "origin_id": %q, "depth": 9}`, forged, forged, forged)
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPatch,
		"/api/v1/records/job/"+r.ID+"?causation_origin_id="+forged+"&causation_depth=9&depth=9", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer user-a")
	req.Header.Set("Content-Type", "application/merge-patch+json")
	req.Header.Set("Idempotency-Key", "caus-patch")
	req.Header.Set("If-Match", `W/"1"`)
	for _, h := range []string{"X-Causation-Origin-Id", "X-Causation-Origin", "Causation-Origin-Id", "X-Origin-Id"} {
		req.Header.Set(h, forged)
	}
	for _, h := range []string{"X-Causation-Depth", "Causation-Depth", "X-Depth"} {
		req.Header.Set(h, "9")
	}
	rec := httptest.NewRecorder()
	e.h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("patch = %d: %s", rec.Code, rec.Body)
	}
	for _, q := range []string{
		"SELECT causation_depth, causation_origin_id::text FROM records WHERE tenant_id = $1 AND id = $2",
		"SELECT causation_depth, causation_origin_id::text FROM audit_log_v0 WHERE tenant_id = $1 AND record_id = $2 AND action = 'update'",
	} {
		var (
			depth  *int16
			origin *string
		)
		if err := e.super.QueryRow(t.Context(), q, e.a.id, r.ID).Scan(&depth, &origin); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
		if depth == nil || *depth != 0 {
			t.Errorf("%s: causation_depth = %v, want 0", q, depth)
		}
		if origin == nil || *origin == "" {
			t.Errorf("%s: causation_origin_id is NULL, want a freshly minted origin", q)
		} else if *origin == forged {
			t.Errorf("%s: causation_origin_id is the origin the request named", q)
		}
	}
	if strings.Contains(rec.Body.String(), forged) {
		t.Errorf("the response carries the request's causation origin: %s", rec.Body)
	}
}

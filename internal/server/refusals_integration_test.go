//go:build integration

// TICK-510: the refusals every tenant operation documents (401 without a
// token, 429 over the tenant's limit) through the real handler against a real
// PostgreSQL, asserting that a refused request writes no row anywhere. The
// same requests are then sent with a token and under the limit, and the writes
// succeed, so "nothing written" means the refusal stopped them and not a bad
// body. Helpers are in entitytypes_integration_test.go and
// records_integration_test.go.

package server_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
)

// refusalOp is one tenant operation as a request (PATCH sends If-Match W/"1").
type refusalOp struct {
	id, method, path, body string
	write                  bool
	want                   int // status once the request is allowed
}

// refusalOps returns every tenant operation of the bundle against tenant A's
// "ticket" type (newRecEnv) and record id.
func refusalOps(id string) []refusalOp {
	def := fmt.Sprintf(`"definition": {"schema": %s,
  "fixed_columns": {"state": "state", "group": "assignment_group_id", "assignee": "assignee_id", "requester": "requester_id"},
  "states": {"initial": "new", "states": ["new", "closed"], "transitions": [{"from": "new", "to": "closed"}]}}`, recSchema)
	return []refusalOp{
		{id: "listEntityTypes", method: http.MethodGet, path: "/api/v1/entity-types", want: http.StatusOK},
		{id: "createEntityType", method: http.MethodPost, path: "/api/v1/entity-types", write: true, want: http.StatusCreated,
			body: `{"key": "refused", "name": "Refused", "number_prefix": "REF", ` + def + `}`},
		{id: "getEntityType", method: http.MethodGet, path: "/api/v1/entity-types/ticket", want: http.StatusOK},
		{id: "publishEntityTypeVersion", method: http.MethodPost, path: "/api/v1/entity-types/ticket/versions", write: true, want: http.StatusCreated,
			body: `{` + def + `}`},
		{id: "getEntityTypeVersion", method: http.MethodGet, path: "/api/v1/entity-types/ticket/versions/1", want: http.StatusOK},
		{id: "createRecord", method: http.MethodPost, path: "/api/v1/records/ticket", write: true, want: http.StatusCreated,
			body: `{"data": {"short_description": "refused"}}`},
		{id: "getRecord", method: http.MethodGet, path: "/api/v1/records/ticket/" + id, want: http.StatusOK},
		{id: "patchRecord", method: http.MethodPatch, path: "/api/v1/records/ticket/" + id, write: true, want: http.StatusOK,
			body: `{"data": {"short_description": "patched"}}`},
		{id: "getRecordHistory", method: http.MethodGet, path: "/api/v1/records/ticket/" + id + "/history", want: http.StatusOK},
		{id: "listAuditEntries", method: http.MethodGet, path: "/api/v1/audit/entries", want: http.StatusOK},
	}
}

func (e *etEnv) send(t *testing.T, op refusalOp, token, idemKey string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), op.method, op.path, strings.NewReader(op.body))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if op.write {
		req.Header.Set("Idempotency-Key", idemKey)
	}
	switch {
	case op.method == http.MethodPatch:
		req.Header.Set("Content-Type", "application/merge-patch+json")
		req.Header.Set("If-Match", `W/"1"`)
	case op.body != "":
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	e.h.ServeHTTP(rec, req)
	return rec
}

// rowCounts is "schema.table=count" for every table of the database, so a
// refused request that wrote anything (a record, an audit row, an outbox
// event, an idempotency key, a counter) shows up as a difference.
func (e *etEnv) rowCounts(t *testing.T) []string {
	t.Helper()
	rows, err := e.super.Query(t.Context(), `SELECT format('%I.%I', schemaname, tablename) FROM pg_tables
		WHERE schemaname NOT IN ('pg_catalog', 'information_schema') ORDER BY 1`)
	if err != nil {
		t.Fatal(err)
	}
	var tables []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			t.Fatal(err)
		}
		tables = append(tables, n)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	out := make([]string, 0, len(tables))
	for _, tb := range tables {
		var n int64
		if err := e.super.QueryRow(t.Context(), "SELECT count(*) FROM "+tb).Scan(&n); err != nil {
			t.Fatalf("count %s: %v", tb, err)
		}
		out = append(out, fmt.Sprintf("%s=%d", tb, n))
	}
	return out
}

// checkRefusals sends every operation refused, checks the status, the code
// and that no row changed, then sends the writes allowed and checks they land.
func checkRefusals(t *testing.T, e *etEnv, status int, code problem.Code, token string, refuse, allow func()) {
	t.Helper()
	r := e.mustCreateRec(t, "user-a", "ticket", "seed", `{"short_description": "seed"}`)
	ops := refusalOps(r.ID)
	refuse()
	before := e.rowCounts(t)
	for _, op := range ops {
		rec := e.send(t, op, token, "refused-"+op.id)
		if rec.Code != status {
			t.Errorf("%s %s %s refused = %d, want %d: %s", op.id, op.method, op.path, rec.Code, status, rec.Body)
			continue
		}
		wantProblem(t, rec, status, code)
	}
	if after := e.rowCounts(t); !slices.Equal(before, after) {
		t.Errorf("refused requests changed rows:\nbefore %v\nafter  %v", before, after)
	}
	allow()
	for _, op := range ops {
		if !op.write {
			continue
		}
		if rec := e.send(t, op, "admin-a", "allowed-"+op.id); rec.Code != op.want {
			t.Errorf("%s allowed = %d, want %d (the refused body must be one that succeeds): %s", op.id, rec.Code, op.want, rec.Body)
		}
	}
	if after := e.rowCounts(t); slices.Equal(before, after) {
		t.Error("the allowed writes changed no row; the row snapshot sees nothing")
	}
}

// covers: listEntityTypes 401
// covers: createEntityType 401
// covers: getEntityType 401
// covers: publishEntityTypeVersion 401
// covers: getEntityTypeVersion 401
// covers: createRecord 401
// covers: getRecord 401
// covers: patchRecord 401
// covers: getRecordHistory 401
// covers: listAuditEntries 401
func TestEveryTenantOperationWithoutATokenIs401AndWritesNothing(t *testing.T) {
	e := newRecEnv(t)
	checkRefusals(t, e, http.StatusUnauthorized, problem.InvalidToken, "", func() {}, func() {})
}

// covers: listEntityTypes 429
// covers: createEntityType 429
// covers: getEntityType 429
// covers: publishEntityTypeVersion 429
// covers: getEntityTypeVersion 429
// covers: createRecord 429
// covers: getRecord 429
// covers: patchRecord 429
// covers: getRecordHistory 429
// covers: listAuditEntries 429
func TestEveryTenantOperationOverTheLimitIs429AndWritesNothing(t *testing.T) {
	e := newRecEnv(t)
	checkRefusals(t, e, http.StatusTooManyRequests, problem.RateLimited, "admin-a",
		func() { e.lim.reset(false) }, func() { e.lim.reset(true) })
}

// withoutIdempotencyKeys drops the idempotency_keys count: a 5xx write leaves
// its key in_flight on purpose so the client may retry after the lease
// (internal/platform/idempotency, step 4); that row is not the write.
func withoutIdempotencyKeys(counts []string) []string {
	return slices.DeleteFunc(slices.Clone(counts), func(c string) bool { return strings.HasPrefix(c, "public.idempotency_keys=") })
}

// A database failure mid-request (the pool's queries run in exec mode, so a
// renamed table fails the next statement) is a 500 internal_error, never a
// 404 or a partial write; once the table is back the same request succeeds.
//
// covers: listEntityTypes 500
// covers: createEntityType 500
// covers: getEntityType 500
// covers: publishEntityTypeVersion 500
// covers: getEntityTypeVersion 500
// covers: createRecord 500
// covers: getRecord 500
// covers: patchRecord 500
// covers: getRecordHistory 500
// covers: listAuditEntries 500
func TestEveryTenantOperationOnADatabaseFailureIs500AndWritesNothing(t *testing.T) {
	// The table each operation reads or writes first in its own store;
	// renaming it is a database failure in the middle of the request.
	brokenTable := map[string]string{
		"listEntityTypes":          "entity_definitions",
		"createEntityType":         "entity_definitions",
		"getEntityType":            "entity_definitions",
		"publishEntityTypeVersion": "entity_definition_versions",
		"getEntityTypeVersion":     "entity_definition_versions",
		"createRecord":             "records",
		"getRecord":                "records",
		"patchRecord":              "records",
		"getRecordHistory":         "audit_log",
		"listAuditEntries":         "audit_log",
	}
	e := newRecEnv(t)
	r := e.mustCreateRec(t, "user-a", "ticket", "seed", `{"short_description": "seed"}`)
	rename := func(from, to string) {
		t.Helper()
		if _, err := e.super.Exec(t.Context(), "ALTER TABLE "+etIdent(from)+" RENAME TO "+etIdent(to)); err != nil {
			t.Fatalf("rename %s to %s: %v", from, to, err)
		}
	}
	for _, op := range refusalOps(r.ID) {
		table := brokenTable[op.id]
		if table == "" {
			t.Fatalf("%s: no table to break", op.id)
		}
		before := withoutIdempotencyKeys(e.rowCounts(t))
		rename(table, table+"_broken")
		rec := e.send(t, op, "admin-a", "broken-"+op.id)
		rename(table+"_broken", table)
		if rec.Code != http.StatusInternalServerError {
			t.Errorf("%s with %s broken = %d, want 500: %s", op.id, table, rec.Code, rec.Body)
			continue
		}
		wantProblem(t, rec, http.StatusInternalServerError, problem.InternalError)
		if after := withoutIdempotencyKeys(e.rowCounts(t)); !slices.Equal(before, after) {
			t.Errorf("%s: a failed request changed rows:\nbefore %v\nafter  %v", op.id, before, after)
		}
		// A fresh key: the failed one stays in_flight until its lease passes.
		if rec := e.send(t, op, "admin-a", "repaired-"+op.id); rec.Code != op.want {
			t.Errorf("%s after the table is back = %d, want %d: %s", op.id, rec.Code, op.want, rec.Body)
		}
	}
}

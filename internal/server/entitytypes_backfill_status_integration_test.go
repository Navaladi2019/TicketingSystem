//go:build integration

// P10-07 (TICK-922, ADR-009 D10): GET /api/v1/entity-types/{key} carries the
// newest schema backfill job as `backfill`, and
// GET /api/v1/entity-types/{key}/backfill/failures lists its failure rows,
// through the real handler on real PostgreSQL as nava_app (SELECT only).

package server_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
)

const (
	etGetPath      = "/api/v1/entity-types/incident"
	etFailuresPath = "/api/v1/entity-types/incident/backfill/failures"
)

// seedJob writes a schema_backfills row of the entity type as the worker
// would, with the counts given.
func (e *etEnv) seedJob(t *testing.T, tenant uuid.UUID, entity string, target int, state string, keys []string, rewritten, skipped, failed int) {
	t.Helper()
	started, finished := "NULL::timestamptz", "NULL::timestamptz"
	switch state {
	case "running", "failed_records":
		started = "now()"
	case "done":
		started, finished = "now()", "now()"
	}
	if _, err := e.super.Exec(t.Context(), fmt.Sprintf(`INSERT INTO schema_backfills
		(tenant_id, entity_definition_id, target_version, state, default_keys, rewritten, skipped, failed, started_at, finished_at)
		SELECT d.tenant_id, d.id, $3, $4, $5, $6, $7, $8, %s, %s FROM entity_definitions d WHERE d.tenant_id = $1 AND d.key = $2`, started, finished),
		tenant, entity, target, state, keys, rewritten, skipped, failed); err != nil {
		t.Fatalf("seed job %s v%d: %v", entity, target, err)
	}
}

// seedQueue adds n pending rows with fresh record ids to the job (a queued id
// needs no record: the table has no foreign key to records).
func (e *etEnv) seedQueue(t *testing.T, tenant uuid.UUID, entity string, target, n int) {
	t.Helper()
	if _, err := e.super.Exec(t.Context(), `INSERT INTO schema_backfill_pending (tenant_id, entity_definition_id, target_version, record_id)
		SELECT d.tenant_id, d.id, $3, gen_random_uuid() FROM entity_definitions d, generate_series(1, $4)
		WHERE d.tenant_id = $1 AND d.key = $2`, tenant, entity, target, n); err != nil {
		t.Fatalf("seed queue: %v", err)
	}
}

// seedFailures adds n failure rows with fresh ids and the code to the job, and
// returns the ids in ascending order.
func (e *etEnv) seedFailures(t *testing.T, tenant uuid.UUID, entity string, target, n int, code string) []uuid.UUID {
	t.Helper()
	ids := make([]uuid.UUID, n)
	for i := range ids {
		ids[i] = uuid.New()
		if _, err := e.super.Exec(t.Context(), `INSERT INTO schema_backfill_failures (tenant_id, entity_definition_id, target_version, record_id, code)
			SELECT d.tenant_id, d.id, $3, $4, $5 FROM entity_definitions d WHERE d.tenant_id = $1 AND d.key = $2`,
			tenant, entity, target, ids[i], code); err != nil {
			t.Fatalf("seed failure: %v", err)
		}
	}
	slices.SortFunc(ids, func(a, b uuid.UUID) int { return slices.Compare(a[:], b[:]) })
	return ids
}

// backfillRows is every row of the three job tables: a read-only route leaves
// it as it found it.
func (e *etEnv) backfillRows(t *testing.T) string {
	t.Helper()
	var s string
	if err := e.super.QueryRow(t.Context(), `SELECT
		(SELECT coalesce(string_agg(b::text, ',' ORDER BY b::text), '') FROM schema_backfills b) || '|' ||
		(SELECT coalesce(string_agg(p::text, ',' ORDER BY p::text), '') FROM schema_backfill_pending p) || '|' ||
		(SELECT coalesce(string_agg(f::text, ',' ORDER BY f::text), '') FROM schema_backfill_failures f)`).Scan(&s); err != nil {
		t.Fatal(err)
	}
	return s
}

type etBackfill struct {
	State         string  `json:"state"`
	TargetVersion int     `json:"target_version"`
	Rewritten     int64   `json:"rewritten"`
	Skipped       int64   `json:"skipped"`
	Failed        int64   `json:"failed"`
	Remaining     int64   `json:"remaining"`
	StartedAt     *string `json:"started_at"`
	FinishedAt    *string `json:"finished_at"`
}

// getBackfill reads GET path as token and returns its `backfill` member, nil
// when the body has none.
func (e *etEnv) getBackfill(t *testing.T, path, token string) *etBackfill {
	t.Helper()
	rec := e.do(t, http.MethodGet, path, token, "", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("get = %d: %s", rec.Code, rec.Body)
	}
	var body map[string]json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	raw, ok := body["backfill"]
	if !ok {
		return nil
	}
	var b etBackfill
	if err := json.Unmarshal(raw, &b); err != nil {
		t.Fatalf("backfill member: %v: %s", err, raw)
	}
	return &b
}

// covers: getEntityType 200
//
// Scenario 1: with a job the read carries `backfill`, from the newest job (the
// highest target_version), with every member; without one it has no
// `backfill` member at all.
func TestGetEntityTypeCarriesTheNewestBackfillJob(t *testing.T) {
	e := newETEnv(t)
	etCreateIncident(t, e)
	if got := e.getBackfill(t, etGetPath, "user-a"); got != nil {
		t.Fatalf("a type with no job has backfill %+v, want no member", got)
	}
	e.seedJob(t, e.a.id, "incident", 2, "done", []string{"severity"}, 40, 2, 0)
	e.seedJob(t, e.a.id, "incident", 3, "running", []string{"severity", "state"}, 7, 1, 2)
	e.seedQueue(t, e.a.id, "incident", 3, 5)
	e.seedFailures(t, e.a.id, "incident", 3, 2, "validation_failed")
	e.seedFailures(t, e.a.id, "incident", 2, 9, "validation_failed") // an older job's rows do not count
	// Any user of the tenant reads it, not only an admin.
	for _, token := range []string{"user-a", "admin-a"} {
		got := e.getBackfill(t, etGetPath, token)
		if got == nil {
			t.Fatalf("%s: no backfill member", token)
		}
		if got.State != "running" || got.TargetVersion != 3 || got.Rewritten != 7 || got.Skipped != 1 || got.Failed != 2 || got.Remaining != 7 ||
			got.StartedAt == nil || got.FinishedAt != nil {
			t.Errorf("%s: backfill = %+v, want running v3 7/1/2 remaining 7 started, not finished", token, got)
		}
	}
	// Tenant B has its own type of the same key and no job: A's job is not its.
	if rec := e.do(t, http.MethodPost, "/api/v1/entity-types", "admin-b", "kb", etBody("incident", "INC")); rec.Code != http.StatusCreated {
		t.Fatalf("create B = %d: %s", rec.Code, rec.Body)
	}
	if got := e.getBackfill(t, etGetPath, "user-b"); got != nil {
		t.Errorf("tenant B reads backfill %+v from tenant A's job", got)
	}
}

// covers: getEntityType 200
//
// Scenario 2: remaining is the pending and failure rows of the newest job, up
// to 1,001: 1,200 queued ids read 1,001, 34 read 34.
func TestGetEntityTypeBackfillRemainingIsCappedAt1001(t *testing.T) {
	e := newETEnv(t)
	etCreateIncident(t, e)
	e.seedJob(t, e.a.id, "incident", 2, "running", []string{"severity"}, 0, 0, 0)
	e.seedQueue(t, e.a.id, "incident", 2, 34)
	if got := e.getBackfill(t, etGetPath, "user-a"); got == nil || got.Remaining != 34 {
		t.Fatalf("34 queued: backfill = %+v, want remaining 34", got)
	}
	e.seedQueue(t, e.a.id, "incident", 2, 1166) // 1,200 in all
	if got := e.getBackfill(t, etGetPath, "user-a"); got == nil || got.Remaining != 1001 {
		t.Fatalf("1,200 queued: backfill = %+v, want remaining 1001 (more than 1,000)", got)
	}
}

// covers: listEntityTypeBackfillFailures 200
//
// Scenario 3: the entity-type admin reads ids and codes, never a value, and a
// cursor pages through 120 failures with no id twice and none missed.
func TestListBackfillFailuresPagesThroughEveryFailureOnce(t *testing.T) {
	e := newETEnv(t)
	etCreateIncident(t, e)
	e.seedJob(t, e.a.id, "incident", 1, "done", []string{"severity"}, 0, 0, 3)
	e.seedFailures(t, e.a.id, "incident", 1, 3, "older_job") // an older job's failures are not listed
	e.seedJob(t, e.a.id, "incident", 2, "failed_records", []string{"severity"}, 0, 0, 120)
	want := e.seedFailures(t, e.a.id, "incident", 2, 120, "validation_failed")
	before := e.backfillRows(t)

	var got []uuid.UUID
	path, pages := etFailuresPath+"?limit=50", 0
	for path != "" {
		rec := e.do(t, http.MethodGet, path, "admin-a", "", "")
		if rec.Code != http.StatusOK {
			t.Fatalf("page %d = %d: %s", pages, rec.Code, rec.Body)
		}
		var raw struct {
			Items      []map[string]json.RawMessage `json:"items"`
			NextCursor string                       `json:"next_cursor"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
			t.Fatal(err)
		}
		for _, it := range raw.Items {
			if len(it) != 2 || it["record_id"] == nil || it["code"] == nil {
				t.Fatalf("item %v: want exactly record_id and code, no value", it)
			}
			var id uuid.UUID
			if err := json.Unmarshal(it["record_id"], &id); err != nil {
				t.Fatal(err)
			}
			if string(it["code"]) != `"validation_failed"` {
				t.Errorf("item code %s, want the newest job's", it["code"])
			}
			got = append(got, id)
		}
		pages++
		path = ""
		if raw.NextCursor != "" {
			path = etFailuresPath + "?limit=50&cursor=" + raw.NextCursor
		}
	}
	if pages != 3 || !slices.Equal(got, want) {
		t.Errorf("%d pages, %d ids; want 3 pages and the 120 failure ids once each in order", pages, len(got))
	}
	if after := e.backfillRows(t); after != before {
		t.Error("the read changed a job row")
	}
	// The default page size is 50.
	rec := e.do(t, http.MethodGet, etFailuresPath, "admin-a", "", "")
	var d struct{ Items []any }
	if err := json.Unmarshal(rec.Body.Bytes(), &d); err != nil || len(d.Items) != 50 {
		t.Errorf("default page = %d items (%v), want 50", len(d.Items), err)
	}
}

// covers: listEntityTypeBackfillFailures 200 404
// covers: getEntityType 404
//
// Scenarios 4 and 5: a plain tenant user, a user of a group merely named
// admins, another tenant's admin and an unknown key all get the same 404, byte
// for byte; the admin of the type's own tenant reads it, and so reads a type
// with no job (nothing listed).
func TestBackfillRoutesDoNotSayWhetherATypeExists(t *testing.T) {
	e := newETEnv(t)
	etCreateIncident(t, e)
	e.seedJob(t, e.a.id, "incident", 2, "failed_records", []string{"severity"}, 0, 0, 3)
	e.seedFailures(t, e.a.id, "incident", 2, 3, "validation_failed")
	if rec := e.do(t, http.MethodPost, "/api/v1/entity-types", "admin-b", "kb", etBodyWithSchema("other", "OTH", etSchema)); rec.Code != http.StatusCreated {
		t.Fatalf("create B = %d: %s", rec.Code, rec.Body)
	}
	missing := func(token string) string {
		rec := e.do(t, http.MethodGet, "/api/v1/entity-types/no_such_type/backfill/failures", token, "", "")
		wantProblem(t, rec, http.StatusNotFound, problem.NotFound)
		return rec.Body.String()
	}
	for _, c := range []struct{ who, token string }{
		{"a plain user (the agent role)", "user-a"},
		{"a user of a group named admins", "lookalike-a"},
		{"tenant B's admin on tenant A's key", "admin-b"},
	} {
		rec := e.do(t, http.MethodGet, etFailuresPath, c.token, "", "")
		wantProblem(t, rec, http.StatusNotFound, problem.NotFound)
		if want := missing(c.token); rec.Body.String() != want {
			t.Errorf("%s: 404 body differs from a missing type's:\n%s\n%s", c.who, rec.Body, want)
		}
	}
	// The status route: tenant B's read of A's key is the same 404 as a missing key.
	rec := e.do(t, http.MethodGet, etGetPath, "admin-b", "", "")
	ref := e.do(t, http.MethodGet, "/api/v1/entity-types/no_such_type", "admin-b", "", "")
	wantProblem(t, rec, http.StatusNotFound, problem.NotFound)
	if rec.Body.String() != ref.Body.String() {
		t.Errorf("getEntityType across tenants: 404 body differs from a missing type's:\n%s\n%s", rec.Body, ref.Body)
	}
	if rec := e.do(t, http.MethodGet, etFailuresPath, "admin-a", "", ""); rec.Code != http.StatusOK {
		t.Errorf("admin = %d, want 200: %s", rec.Code, rec.Body)
	}
	if rec := e.do(t, http.MethodGet, "/api/v1/entity-types/other/backfill/failures", "admin-b", "", ""); rec.Code != http.StatusOK {
		t.Errorf("a type with no job = %d, want 200: %s", rec.Code, rec.Body)
	}
}

// covers: listEntityTypeBackfillFailures 400
//
// Scenario 6: a limit outside 1 to 100, a limit that is not a number and a
// cursor that is not ours are 400 and nothing is written.
func TestListBackfillFailuresRefusesABadLimitAndWritesNothing(t *testing.T) {
	e := newETEnv(t)
	etCreateIncident(t, e)
	e.seedJob(t, e.a.id, "incident", 2, "failed_records", []string{"severity"}, 0, 0, 3)
	e.seedFailures(t, e.a.id, "incident", 2, 3, "validation_failed")
	before := e.rowCounts(t)
	for _, q := range []string{"limit=0", "limit=101", "limit=abc", "limit=-1"} {
		wantProblem(t, e.do(t, http.MethodGet, etFailuresPath+"?"+q, "admin-a", "", ""), http.StatusBadRequest, problem.ValidationFailed)
	}
	wantProblem(t, e.do(t, http.MethodGet, etFailuresPath+"?cursor=not-a-cursor", "admin-a", "", ""), http.StatusBadRequest, problem.InvalidCursor)
	if after := e.rowCounts(t); !slices.Equal(before, after) {
		t.Errorf("refused reads changed rows:\nbefore %v\nafter  %v", before, after)
	}
}

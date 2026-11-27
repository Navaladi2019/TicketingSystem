//go:build integration

// P10-08, P10-09 (TICK-923, TICK-924, ADR-009 D11, D12, CI-7, CI-8): while a
// schema backfill window is open, a filter on a defaulted field finds the
// records that do not hold the default yet, in the OR shape and the UNION ALL
// shape, and every surface that returns a record body shows the effective
// value, through the real handler on real PostgreSQL.

package server_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/authz/acltest"
	"github.com/Navaladi2019/TicketingSystem/internal/entitytypes"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
	"github.com/Navaladi2019/TicketingSystem/internal/server"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
	"github.com/Navaladi2019/TicketingSystem/internal/workerguard"
	"github.com/Navaladi2019/TicketingSystem/internal/workflow/recordports"
)

// bfSchema returns recSchema plus a free text property (tag) and, per extra,
// further properties.
func bfSchema(extra string) string {
	return strings.Replace(recSchema, `"requester": {"type": "string", "format": "uuid"}`,
		`"requester": {"type": "string", "format": "uuid"}, "tag": {"type": "string"}`+extra, 1)
}

const (
	bfSeverity = `, "severity": {"type": "string", "default": "low"}`
	bfExtra    = `, "extra": {"type": "string", "default": "x"}`
	bfDef      = `"fixed_columns": {"state": "state", "group": "assignment_group_id", "assignee": "assignee_id", "requester": "requester_id"},
  "states": {"initial": "new", "states": ["new", "closed"], "transitions": [{"from": "new", "to": "closed"}]}`
)

func bfCreateBody() string {
	return fmt.Sprintf(`{"key": "job", "name": "Job", "number_prefix": "JOB", "definition": {"schema": %s, %s}}`, bfSchema(""), bfDef)
}

func bfPublishBody(extra string) string {
	return fmt.Sprintf(`{"definition": {"schema": %s, %s}}`, bfSchema(extra), bfDef)
}

// bfEnv is tenant A's type "job": six records written under version 1, which
// has no severity (the old ones), then version 2 adds severity with default
// "low" (a pending job row), then two records that store the default and two
// that store "high". severity is flagged filterable and ready.
type bfEnv struct {
	*etEnv
	old, low, high []string
	mu             sync.Mutex
	stmts          []string
	args           [][]any
}

func newBFEnv(t *testing.T) *bfEnv {
	t.Helper()
	b := &bfEnv{}
	b.etEnv = newETEnvWith(t, server.WithListStatement(func(sql string, args []any) {
		b.mu.Lock()
		defer b.mu.Unlock()
		b.stmts = append(b.stmts, sql)
		b.args = append(b.args, args)
	}))
	e := b.etEnv
	for _, c := range []struct{ tok, key string }{{"admin-a", "bf-a"}, {"admin-b", "bf-b"}} {
		if rec := e.do(t, http.MethodPost, "/api/v1/entity-types", c.tok, c.key, bfCreateBody()); rec.Code != http.StatusCreated {
			t.Fatalf("create type = %d: %s", rec.Code, rec.Body)
		}
	}
	for i := range 6 {
		b.old = append(b.old, e.mustCreateRec(t, "admin-a", "job", fmt.Sprintf("old-%d", i), fmt.Sprintf(`{"short_description": "old %d"}`, i)).ID)
	}
	if rec := e.do(t, http.MethodPost, "/api/v1/entity-types/job/versions", "admin-a", "bf-v2", bfPublishBody(bfSeverity)); rec.Code != http.StatusCreated {
		t.Fatalf("publish v2 = %d: %s", rec.Code, rec.Body)
	}
	for i := range 2 {
		b.low = append(b.low, e.mustCreateRec(t, "admin-a", "job", fmt.Sprintf("low-%d", i), `{"short_description": "low"}`).ID)
		b.high = append(b.high, e.mustCreateRec(t, "admin-a", "job", fmt.Sprintf("high-%d", i), `{"short_description": "high", "severity": "high"}`).ID)
	}
	e.flagField(t, e.a.id, "job", "severity", "text", "ready", 100)
	// The test plays the backfill with plain UPDATEs, which the records audit
	// commit check would refuse; the pipeline's own writes keep their audit entry.
	e.superExec(t, "ALTER TABLE records DISABLE TRIGGER records_audit_commit_check")
	// Half of the old records are closed, so an ACL on state can hide the rest.
	for i, id := range b.old {
		if i%2 == 0 {
			if _, err := e.super.Exec(t.Context(), `UPDATE records SET state = 'closed' WHERE tenant_id = $1 AND id = $2`, e.a.id, id); err != nil {
				t.Fatal(err)
			}
		}
	}
	return b
}

// expected is the ids severity = low should return once every record is read
// as effective: the old ones, which read the default, and the ones that store it.
func (b *bfEnv) expectedLow() []string { return slices.Concat(b.old, b.low) }

func (b *bfEnv) job(t *testing.T, tenant uuid.UUID, target int) string {
	t.Helper()
	var s string
	if err := b.super.QueryRow(t.Context(), `SELECT state FROM schema_backfills WHERE tenant_id = $1 AND target_version = $2`, tenant, target).Scan(&s); err != nil {
		t.Fatal(err)
	}
	return s
}

func (b *bfEnv) setJob(t *testing.T, tenant uuid.UUID, target int, state string) {
	t.Helper()
	if _, err := b.super.Exec(t.Context(), `UPDATE schema_backfills SET state = $3,
		started_at = CASE WHEN $3 IN ('pending', 'filling') THEN NULL ELSE coalesce(started_at, now()) END
		WHERE tenant_id = $1 AND target_version = $2`, tenant, target, state); err != nil {
		t.Fatalf("job state %s: %v", state, err)
	}
}

func (b *bfEnv) queue(t *testing.T, tenant uuid.UUID, target int, ids []string) {
	t.Helper()
	for _, id := range ids {
		if _, err := b.super.Exec(t.Context(), `INSERT INTO schema_backfill_pending (tenant_id, entity_definition_id, target_version, record_id)
			SELECT d.tenant_id, d.id, $2, $3 FROM entity_definitions d WHERE d.tenant_id = $1 AND d.key = 'job'`, tenant, target, id); err != nil {
			t.Fatal(err)
		}
	}
}

func (b *bfEnv) clearQueue(t *testing.T) {
	t.Helper()
	if _, err := b.super.Exec(t.Context(), `DELETE FROM schema_backfill_pending WHERE tenant_id = $1`, b.a.id); err != nil {
		t.Fatal(err)
	}
}

// rewrite does what the backfill does to the records: stores the default.
func (b *bfEnv) rewrite(t *testing.T, ids []string) {
	t.Helper()
	for _, id := range ids {
		if _, err := b.super.Exec(t.Context(), `UPDATE records SET data = data || '{"severity": "low"}', version = version + 1 WHERE tenant_id = $1 AND id = $2`, b.a.id, id); err != nil {
			t.Fatal(err)
		}
	}
}

type bfList struct {
	Items []recBody `json:"items"`
	Total *int64    `json:"total"`
}

// list GETs the records of "job" with the query and returns the sorted ids and
// the total.
func (b *bfEnv) list(t *testing.T, token, query string) ([]string, int64) {
	t.Helper()
	rec := b.do(t, http.MethodGet, "/api/v1/records/job?limit=100&"+query, token, "", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("list %s = %d: %s", query, rec.Code, rec.Body)
	}
	var l bfList
	if err := json.Unmarshal(rec.Body.Bytes(), &l); err != nil || l.Total == nil {
		t.Fatalf("list body: %v: %s", err, rec.Body)
	}
	ids := make([]string, len(l.Items))
	for i, it := range l.Items {
		ids[i] = it.ID
	}
	slices.Sort(ids)
	return ids, *l.Total
}

func (b *bfEnv) wantList(t *testing.T, name, token, query string, want []string) {
	t.Helper()
	want = slices.Sorted(slices.Values(want))
	got, total := b.list(t, token, query)
	if !slices.Equal(got, want) || total != int64(len(want)) {
		t.Errorf("%s: %d ids total %d, want %d ids\n got  %v\n want %v", name, len(got), total, len(want), got, want)
	}
}

func (b *bfEnv) lastStatement(t *testing.T) string {
	t.Helper()
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(b.stmts) == 0 {
		t.Fatal("no list statement was captured")
	}
	return b.stmts[len(b.stmts)-1]
}

// covers: listRecords 200
//
// CI-7 (a): a job queued but not filled, and again right after a supersede,
// returns the records that lack the default, with the right total.
func TestListFilterOnADefaultFindsUnrewrittenRecordsWhilePending(t *testing.T) {
	b := newBFEnv(t)
	if s := b.job(t, b.a.id, 2); s != "pending" {
		t.Fatalf("publish left the job %s, want pending", s)
	}
	b.wantList(t, "pending", "admin-a", "filter[severity]=low", b.expectedLow())
	b.wantList(t, "pending, in", "admin-a", "filter[severity][in]=low,high", slices.Concat(b.expectedLow(), b.high))
	b.wantList(t, "filter on another value", "admin-a", "filter[severity]=high", b.high)

	// A newer publish supersedes the job: its row is superseded, the new one pending.
	if rec := b.do(t, http.MethodPost, "/api/v1/entity-types/job/versions", "admin-a", "bf-v3", bfPublishBody(bfSeverity+bfExtra)); rec.Code != http.StatusCreated {
		t.Fatalf("publish v3 = %d: %s", rec.Code, rec.Body)
	}
	b.setJob(t, b.a.id, 2, "superseded")
	b.wantList(t, "after a supersede", "admin-a", "filter[severity]=low", b.expectedLow())
	b.wantList(t, "after a supersede, other value", "admin-a", "filter[severity]=high", b.high)
}

// covers: listRecords 200
//
// CI-7 (b): a running job with more than 1,000 remaining (the OR shape) and
// with 6 remaining (the UNION ALL shape) return the same set and total, and an
// ACL that hides records never lets a hidden one through in either shape.
func TestListFilterOnADefaultIsTheSameInTheORShapeAndTheUnionShape(t *testing.T) {
	b := newBFEnv(t)
	b.setJob(t, b.a.id, 2, "running")
	// user-a reads only closed records; the admin reads all.
	rules := []authz.Rule{
		acltest.AllowUsers(b.a.admin.PrincipalID, b.a.lookalike.PrincipalID),
		{ID: "closed-only", Ops: []string{authz.OpRead}, Who: authz.Subject{Users: []uuid.UUID{b.a.user.PrincipalID}},
			Where: []authz.Cond{{Column: "state", Op: authz.CondEq, Values: []string{"closed"}}}},
	}
	if err := tenancy.InTenantTx(t.Context(), b.pool, b.a.id, func(tx pgx.Tx) error {
		_, err := acltest.PublishPolicy(t.Context(), tx, "job", rules, nil)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	var closed []string
	for i, id := range b.old {
		if i%2 == 0 {
			closed = append(closed, id)
		}
	}

	// More than 1,000 remaining: the OR shape.
	b.queue(t, b.a.id, 2, b.old)
	b.seedQueue(t, b.a.id, "job", 2, 1200)
	b.wantList(t, "OR shape, admin", "admin-a", "filter[severity]=low", b.expectedLow())
	if s := b.lastStatement(t); strings.Contains(s, "UNION ALL") && strings.Contains(s, "schema_backfill") {
		t.Errorf("more than 1,000 remaining used the UNION ALL shape:\n%s", s)
	}
	b.wantList(t, "OR shape, ACL", "user-a", "filter[severity]=low", closed)

	// 1,000 or fewer remaining: the UNION ALL shape, the same rows.
	b.clearQueue(t)
	b.queue(t, b.a.id, 2, b.old)
	b.wantList(t, "UNION shape, admin", "admin-a", "filter[severity]=low", b.expectedLow())
	if s := b.lastStatement(t); !strings.Contains(s, "schema_backfill_pending") || !strings.Contains(s, "UNION ALL") {
		t.Errorf("6 remaining did not use the UNION ALL shape:\n%s", s)
	}
	b.wantList(t, "UNION shape, ACL", "user-a", "filter[severity]=low", closed)
	b.wantList(t, "other value", "admin-a", "filter[severity]=high", b.high)
	b.wantList(t, "other value, ACL", "user-a", "filter[severity]=high", nil)

	// A page boundary inside the UNION ALL shape: pages of 4 walk the set once.
	var seen []string
	query := "filter[severity]=low"
	for page := 0; page < 10; page++ {
		rec := b.do(t, http.MethodGet, "/api/v1/records/job?limit=4&"+query, "admin-a", "", "")
		var l struct {
			Items      []recBody `json:"items"`
			Total      int64     `json:"total"`
			NextCursor string    `json:"next_cursor"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &l); err != nil || rec.Code != http.StatusOK {
			t.Fatalf("page %d = %d: %s", page, rec.Code, rec.Body)
		}
		if l.Total != int64(len(b.expectedLow())) {
			t.Errorf("page %d total = %d, want %d", page, l.Total, len(b.expectedLow()))
		}
		for _, it := range l.Items {
			seen = append(seen, it.ID)
		}
		if l.NextCursor == "" {
			break
		}
		query = "filter[severity]=low&cursor=" + l.NextCursor
	}
	slices.Sort(seen)
	if !slices.Equal(seen, slices.Sorted(slices.Values(b.expectedLow()))) {
		t.Errorf("paging the UNION ALL shape saw %v, want each of %v once", seen, b.expectedLow())
	}
}

// covers: listRecords 200
//
// After the job is done and the records are rewritten, the results are those
// of the open window; failed records that stay unrewritten still match.
func TestListFilterOnADefaultAfterDoneAndWithFailedRecords(t *testing.T) {
	b := newBFEnv(t)
	b.setJob(t, b.a.id, 2, "failed_records")
	// Two failures stay unrewritten, the other four were rewritten.
	failed, rewritten := b.old[:2], b.old[2:]
	b.rewrite(t, rewritten)
	for _, id := range failed {
		if _, err := b.super.Exec(t.Context(), `INSERT INTO schema_backfill_failures (tenant_id, entity_definition_id, target_version, record_id, code)
			SELECT d.tenant_id, d.id, 2, $2, 'validation_failed' FROM entity_definitions d WHERE d.tenant_id = $1 AND d.key = 'job'`, b.a.id, id); err != nil {
			t.Fatal(err)
		}
	}
	b.wantList(t, "failed_records", "admin-a", "filter[severity]=low", b.expectedLow())
	if s := b.lastStatement(t); !strings.Contains(s, "schema_backfill_failures") {
		t.Errorf("a failed_records job did not use the failures branch:\n%s", s)
	}
	b.wantList(t, "failed_records, other value", "admin-a", "filter[severity]=high", b.high)

	b.rewrite(t, failed)
	b.setJob(t, b.a.id, 2, "done")
	b.wantList(t, "done", "admin-a", "filter[severity]=low", b.expectedLow())
	b.wantList(t, "done, other value", "admin-a", "filter[severity]=high", b.high)
	if s := b.lastStatement(t); strings.Contains(s, "schema_backfill") {
		t.Errorf("a closed window still reads the job tables:\n%s", s)
	}
}

// covers: listRecords 200 400
//
// Another tenant's open job on the same type key changes nothing in this
// tenant's results; an invalid filter value is the existing 400 and writes
// nothing.
func TestListFilterWindowIsPerTenantAndRefusesABadValue(t *testing.T) {
	b := newBFEnv(t)
	// Tenant B publishes the same version: its job is pending and open.
	if rec := b.do(t, http.MethodPost, "/api/v1/entity-types/job/versions", "admin-b", "bf-vb", bfPublishBody(bfSeverity)); rec.Code != http.StatusCreated {
		t.Fatalf("B publish = %d: %s", rec.Code, rec.Body)
	}
	if s := b.job(t, b.b.id, 2); s != "pending" {
		t.Fatalf("B job = %s", s)
	}
	// Tenant A's window is closed with one record left unrewritten: it does not match.
	b.rewrite(t, b.old[1:])
	b.setJob(t, b.a.id, 2, "done")
	want := slices.Concat(b.old[1:], b.low)
	b.wantList(t, "A done, B open", "admin-a", "filter[severity]=low", want)

	before := b.rowCounts(t)
	rec := b.do(t, http.MethodGet, "/api/v1/records/job?filter[severity]="+strings.Repeat("x", 2001), "admin-a", "", "")
	wantProblem(t, rec, http.StatusBadRequest, problem.ValidationFailed)
	if after := b.rowCounts(t); !slices.Equal(before, after) {
		t.Errorf("a refused list changed rows:\nbefore %v\nafter  %v", before, after)
	}
	b.setJob(t, b.a.id, 2, "running")
	b.queue(t, b.a.id, 2, b.old[:1])
	rec = b.do(t, http.MethodGet, "/api/v1/records/job?filter[severity]="+strings.Repeat("x", 2001), "admin-a", "", "")
	wantProblem(t, rec, http.StatusBadRequest, problem.ValidationFailed)
}

// With 1,000 or fewer remaining the plan reads the queue, never scans records.
func TestListFilterUnionShapePlanReadsTheQueueAndNoSeqScanOnRecords(t *testing.T) {
	b := newBFEnv(t)
	b.setJob(t, b.a.id, 2, "running")
	b.queue(t, b.a.id, 2, b.old)
	if rec := b.do(t, http.MethodGet, "/api/v1/records/job?filter[severity]=low", "admin-a", "", ""); rec.Code != http.StatusOK {
		t.Fatalf("list = %d: %s", rec.Code, rec.Body)
	}
	b.mu.Lock()
	stmt, args := b.stmts[len(b.stmts)-1], b.args[len(b.args)-1]
	b.mu.Unlock()
	if !strings.Contains(stmt, "UNION ALL") {
		t.Fatalf("not the UNION ALL shape: %s", stmt)
	}
	var plan []string
	if err := tenancy.InTenantTx(t.Context(), b.pool, b.a.id, func(tx pgx.Tx) error {
		// The tables are tiny, so the planner would scan them; with the scans
		// off, a seq scan on records can only appear when no index serves a term.
		if _, err := tx.Exec(t.Context(), "SET LOCAL enable_seqscan = off"); err != nil {
			return err
		}
		rows, err := tx.Query(t.Context(), "EXPLAIN (COSTS OFF) "+stmt, args...)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var l string
			if err := rows.Scan(&l); err != nil {
				return err
			}
			plan = append(plan, l)
		}
		return rows.Err()
	}); err != nil {
		t.Fatal(err)
	}
	text := strings.Join(plan, "; ")
	if strings.Contains(text, "Seq Scan on records") || !strings.Contains(text, "schema_backfill_pending") || !strings.Contains(text, "Append") {
		t.Errorf("plan has a seq scan on records, no queue read or no Append: %s", text)
	}
}

// D11, CI-8: a record stored under version 1, which has no severity, reads the
// same JSON on the single read, in a list, in the 200 of a PATCH and through
// the workflow reader, with the default; the audit diff shows the stored
// bytes; and a role that cannot see severity never sees the default.
//
// covers: getRecord 200
// covers: patchRecord 200
func TestEveryRecordSurfaceReturnsTheEffectiveValue(t *testing.T) {
	b := newBFEnv(t)
	id := b.old[1]
	wp := b.workerPool(t)
	asWorker := func() map[string]any {
		data, _, err := (recordports.Reader{Pool: wp}).Read(workerguard.ContextWithTenant(t.Context(), b.a.id), b.a.id, uuid.MustParse(id))
		if err != nil {
			t.Fatalf("workflow read: %v", err)
		}
		raw, _ := json.Marshal(data)
		var m map[string]any
		_ = json.Unmarshal(raw, &m)
		return m
	}
	read := func() map[string]any {
		rec := b.do(t, http.MethodGet, "/api/v1/records/job/"+id, "admin-a", "", "")
		return recordData(t, rec)
	}
	listed := func() map[string]any {
		rec := b.do(t, http.MethodGet, "/api/v1/records/job?limit=100", "admin-a", "", "")
		var l bfList
		if err := json.Unmarshal(rec.Body.Bytes(), &l); err != nil {
			t.Fatal(err)
		}
		for _, it := range l.Items {
			if it.ID == id {
				return it.Data
			}
		}
		t.Fatalf("record %s not listed", id)
		return nil
	}
	stored := func() string {
		var s string
		if err := b.super.QueryRow(t.Context(), `SELECT data::text FROM records WHERE tenant_id = $1 AND id = $2`, b.a.id, id).Scan(&s); err != nil {
			t.Fatal(err)
		}
		return s
	}
	if strings.Contains(stored(), "severity") {
		t.Fatalf("the old record already stores severity: %s", stored())
	}
	want := read()
	if want["severity"] != "low" {
		t.Fatalf("single read = %v, want the default severity", want)
	}
	for name, got := range map[string]map[string]any{"list": listed(), "workflow reader": asWorker()} {
		if fmt.Sprint(got) != fmt.Sprint(want) {
			t.Errorf("%s = %v, want the single read's %v", name, got, want)
		}
	}
	if strings.Contains(stored(), "severity") {
		t.Errorf("a read wrote the default into the stored bytes: %s", stored())
	}

	// PATCH: the 200 body is the effective value after the change.
	rec := b.patch(t, "admin-a", "job", id, `W/"1"`, `{"data": {"tag": "t"}}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("patch = %d: %s", rec.Code, rec.Body)
	}
	body := recordData(t, rec)
	after := read()
	if fmt.Sprint(body) != fmt.Sprint(after) || after["severity"] != "low" || after["tag"] != "t" {
		t.Errorf("patch body %v, single read %v: want the same effective value", body, after)
	}
	if fmt.Sprint(listed()) != fmt.Sprint(after) {
		t.Errorf("list item %v after the patch, want %v", listed(), after)
	}
	// The audit entry of the change diffs from the stored bytes: the default
	// the old record never stored is an addition, not part of "before".
	var diff string
	if err := b.super.QueryRow(t.Context(), `SELECT diff::text FROM audit_log WHERE tenant_id = $1 AND subject_id = $2 AND domain = 'record'
		ORDER BY occurred_at DESC, id DESC LIMIT 1`, b.a.id, id).Scan(&diff); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(diff, "severity") {
		t.Errorf("the audit diff %s does not show severity as added to the stored bytes", diff)
	}
}

// A masked field stays masked after defaults are applied.
//
// covers: getRecord 200
func TestAMaskedFieldNeverShowsItsDefault(t *testing.T) {
	b := newBFEnv(t)
	desk := authz.Subject{Groups: []uuid.UUID{b.a.lookalikeGroup}}
	rules := []authz.Rule{acltest.AllowUsers(b.a.admin.PrincipalID, b.a.user.PrincipalID, b.a.lookalike.PrincipalID)}
	if err := tenancy.InTenantTx(t.Context(), b.pool, b.a.id, func(tx pgx.Tx) error {
		_, err := acltest.PublishPolicy(t.Context(), tx, "job", rules, []authz.Mask{{Field: "severity", Read: desk, Write: desk}})
		return err
	}); err != nil {
		t.Fatal(err)
	}
	id := b.old[1]
	for _, c := range []struct {
		token string
		sees  bool
	}{{"user-a", false}, {"lookalike-a", true}} {
		single := recordData(t, b.do(t, http.MethodGet, "/api/v1/records/job/"+id, c.token, "", ""))
		_, has := single["severity"]
		if has != c.sees {
			t.Errorf("%s single read has severity = %v, want %v: %v", c.token, has, c.sees, single)
		}
		rec := b.do(t, http.MethodGet, "/api/v1/records/job?limit=100", c.token, "", "")
		var l bfList
		if err := json.Unmarshal(rec.Body.Bytes(), &l); err != nil {
			t.Fatal(err)
		}
		for _, it := range l.Items {
			if _, has := it.Data["severity"]; has && !c.sees && it.Data["severity"] == "low" && slices.Contains(b.old, it.ID) {
				t.Errorf("%s list shows the default of a masked field on %s", c.token, it.ID)
			}
		}
	}
}

// workerPool is a nava_worker pool, as the worker's reader runs on.
func (b *bfEnv) workerPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	cfg := b.pool.Config().Copy()
	cfg.ConnConfig.User, cfg.ConnConfig.Password = "nava_worker", etPassword("nava_worker")
	cfg.MaxConns = 2
	p, err := pgxpool.NewWithConfig(t.Context(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(p.Close)
	return p
}

// covers: listRecords 200
//
// While a window is open, a filter on a fixed column or on a field the job does
// not default is answered as it always is: the window touches only the
// defaulted field.
func TestListFilterOutsideTheWindowIsUnchangedWhileItIsOpen(t *testing.T) {
	b := newBFEnv(t)
	b.flagField(t, b.a.id, "job", "tag", "text", "ready", 100)
	if s := b.job(t, b.a.id, 2); s != "pending" {
		t.Fatalf("job = %s, want an open window", s)
	}
	var closed []string
	for i, id := range b.old {
		if i%2 == 0 {
			closed = append(closed, id)
		}
	}
	b.wantList(t, "fixed column", "admin-a", "filter[state]=closed", closed)
	b.wantList(t, "field with no default", "admin-a", "filter[tag]=nothing", nil)
	if s := b.lastStatement(t); strings.Contains(s, "record_field_coerce") {
		t.Errorf("a filter outside the window reads defaults:\n%s", s)
	}
}

// covers: getEntityType 404
// covers: listEntityTypeBackfillFailures 404
//
// A key longer than any type can have is the unknown key's 404 on both routes,
// and the store reports no job for a key the tenant does not have.
func TestBackfillRoutesRefuseAnImpossibleKeyAndTheStoreKnowsNoJobForAnUnknownOne(t *testing.T) {
	e := newETEnv(t)
	etCreateIncident(t, e)
	long := strings.Repeat("k", 80)
	wantProblem(t, e.do(t, http.MethodGet, "/api/v1/entity-types/"+long, "user-a", "", ""), http.StatusNotFound, problem.NotFound)
	wantProblem(t, e.do(t, http.MethodGet, "/api/v1/entity-types/"+long+"/backfill/failures", "admin-a", "", ""), http.StatusNotFound, problem.NotFound)
	if err := tenancy.InTenantTx(t.Context(), e.pool, e.a.id, func(tx pgx.Tx) error {
		got, err := (entitytypes.Store{}).NewestBackfill(t.Context(), tx, e.a.id, "no_such_type")
		if err != nil || got != nil {
			t.Errorf("NewestBackfill of an unknown key = %v, %v; want nil, nil", got, err)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

// covers: listRecords 200
//
// ADR-009 D12 (TICK-1200): "absent" is key presence. A record that stores an
// explicit null, or a value that does not coerce, has the key, so inside the
// window a filter only the default satisfies never returns it, in the OR shape
// and in the UNION ALL shape, and the total agrees. Outside the window the
// stored value alone decides.
func TestListFilterWindowDoesNotReturnAStoredNullOrANonCoercibleValue(t *testing.T) {
	b := newBFEnv(t)
	nullRec := b.old[0]
	oddRec := b.old[1]
	for id, doc := range map[string]string{nullRec: `{"severity": null}`, oddRec: `{"severity": {"x": 1}}`} {
		if _, err := b.super.Exec(t.Context(), `UPDATE records SET data = data || $3::jsonb, version = version + 1 WHERE tenant_id = $1 AND id = $2`, b.a.id, id, doc); err != nil {
			t.Fatal(err)
		}
	}
	want := slices.DeleteFunc(slices.Clone(b.expectedLow()), func(id string) bool { return id == nullRec || id == oddRec })

	// OR shape: more than 1,000 remaining.
	b.setJob(t, b.a.id, 2, "running")
	b.queue(t, b.a.id, 2, b.old)
	b.seedQueue(t, b.a.id, "job", 2, 1200)
	b.wantList(t, "OR shape", "admin-a", "filter[severity]=low", want)
	b.wantList(t, "OR shape, in", "admin-a", "filter[severity][in]=low,high", slices.Concat(want, b.high))

	// UNION ALL shape: 1,000 or fewer remaining.
	b.clearQueue(t)
	b.queue(t, b.a.id, 2, b.old)
	b.wantList(t, "UNION ALL shape", "admin-a", "filter[severity]=low", want)
	b.wantList(t, "UNION ALL shape, in", "admin-a", "filter[severity][in]=low,high", slices.Concat(want, b.high))

	// Outside the window nothing but the stored value decides.
	b.clearQueue(t)
	b.setJob(t, b.a.id, 2, "done")
	b.wantList(t, "done", "admin-a", "filter[severity]=low", b.low)
	b.wantList(t, "done, high", "admin-a", "filter[severity]=high", b.high)
}

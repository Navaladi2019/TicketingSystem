//go:build integration

// P5-40 (TICK-444): GET /api/v1/audit/entries:export against a real
// PostgreSQL, through the real handler, the real sealer and the real offline
// verifier (ADR-004 R19-R21, CI-A6, CI-A8, CI-A14).

package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/internal/audit/patchwalk"
	"github.com/Navaladi2019/TicketingSystem/internal/audit/read"
	"github.com/Navaladi2019/TicketingSystem/internal/audit/seal"
	"github.com/Navaladi2019/TicketingSystem/internal/audit/verify"
	"github.com/Navaladi2019/TicketingSystem/internal/server"
	"github.com/Navaladi2019/TicketingSystem/internal/workerguard"
)

type exportEnv struct {
	*etEnv
	worker *pgxpool.Pool
	mu     sync.Mutex
	starts []time.Time
	tos    []string
	onPage func()
}

// newExportEnv builds the patch environment with tenants A and B each holding
// a chain head, pages of two rows, and a hook that records every page
// transaction.
func newExportEnv(t *testing.T, extra ...server.Option) *exportEnv {
	t.Helper()
	x := &exportEnv{}
	opts := append([]server.Option{server.WithExportPaging(2, func(start time.Time, st string) {
		x.mu.Lock()
		x.starts, x.tos = append(x.starts, start), append(x.tos, st)
		hook := x.onPage
		x.mu.Unlock()
		if hook != nil {
			hook()
		}
	})}, extra...)
	x.etEnv = newPatchEnvWith(t, opts...)
	ctx := t.Context()
	for _, tn := range []etTenant{x.a, x.b} {
		if _, err := x.super.Exec(ctx, "INSERT INTO audit_chain_head (tenant_id, id) VALUES ($1, $2)", tn.id, uuid.New()); err != nil {
			t.Fatal(err)
		}
	}
	// ADR-004-A1 / TICK-612: the reader reads prev_hash, entry_hash and
	// erasure_ref. Until that grant is in the migrations, grant it here.
	var has bool
	if err := x.super.QueryRow(ctx, "SELECT has_column_privilege('nava_audit_reader', 'audit_log', 'entry_hash', 'SELECT')").Scan(&has); err != nil {
		t.Fatal(err)
	}
	if !has {
		t.Log("TICK-612 grant absent: granting prev_hash, entry_hash, erasure_ref to nava_audit_reader for this test")
		if _, err := x.super.Exec(ctx, "GRANT SELECT (prev_hash, entry_hash, erasure_ref) ON audit_log TO nava_audit_reader"); err != nil {
			t.Fatal(err)
		}
	}
	u, err := url.Parse(os.Getenv("NAVA_TEST_PG_URL"))
	if err != nil {
		t.Fatal(err)
	}
	u.User = url.UserPassword("nava_worker", etPassword("nava_worker"))
	u.Path = "/" + x.pool.Config().ConnConfig.Database
	cfg, err := pgxpool.ParseConfig(u.String())
	if err != nil {
		t.Fatal(err)
	}
	if x.worker, err = pgxpool.NewWithConfig(ctx, cfg); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(x.worker.Close)
	return x
}

func (x *exportEnv) sealAll(t *testing.T) {
	t.Helper()
	for _, tn := range []etTenant{x.a, x.b} {
		if _, err := seal.Seal(workerguard.ContextWithTenant(context.Background(), tn.id), x.worker); err != nil {
			t.Fatal(err)
		}
	}
}

func (x *exportEnv) export(t *testing.T, token, query string) *httptest.ResponseRecorder {
	t.Helper()
	return x.do(t, http.MethodGet, "/api/v1/audit/entries:export"+query, token, "", "")
}

func report(t *testing.T, body []byte) *verify.Report {
	t.Helper()
	rep, err := verify.Verify(bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	return rep
}

// seedActivity writes a record with two updates, then seals.
func (x *exportEnv) seedActivity(t *testing.T) (recordID string) {
	t.Helper()
	r := x.mustCreateRec(t, "user-a", "job", "ex-1", `{"short_description": "one"}`)
	for i, note := range []string{"a", "b"} {
		if rec := x.patch(t, "user-a", "job", r.ID, fmt.Sprintf(`W/"%d"`, i+1), fmt.Sprintf(`{"data": {"note": %q}}`, note)); rec.Code != http.StatusOK {
			t.Fatalf("patch = %d: %s", rec.Code, rec.Body)
		}
	}
	x.mustCreateRec(t, "user-a", "job", "ex-2", `{"short_description": "two"}`)
	x.sealAll(t)
	return r.ID
}

// CI-A6 end to end: exports filtered by subject, time and domain verify, and
// a byte edit and a deleted first entry each fail. CI-A4: tenant B's export
// holds nothing of tenant A's.
//
// covers: exportAuditEntries 200
func TestExportVerifiesUnderEveryFilter(t *testing.T) {
	x := newExportEnv(t)
	rid := x.seedActivity(t)

	for name, q := range map[string]string{
		"unfiltered": "",
		"subject":    "?subject_id=" + rid,
		"domain":     "?domain=config",
		"time":       "?from=" + url.QueryEscape(time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)) + "&to=" + url.QueryEscape(time.Now().Add(time.Hour).UTC().Format(time.RFC3339)),
		"both":       "?domain=record&subject_id=" + rid,
	} {
		rec := x.export(t, "admin-a", q)
		if rec.Code != http.StatusOK || !strings.HasPrefix(rec.Header().Get("Content-Type"), "application/x-ndjson") {
			t.Fatalf("%s: %d %s: %s", name, rec.Code, rec.Header().Get("Content-Type"), rec.Body)
		}
		if rep := report(t, rec.Body.Bytes()); rep.Outcome != verify.OutcomePass {
			t.Errorf("%s: outcome %s, failure %+v\n%s", name, rep.Outcome, rep.Failure, rec.Body)
		}
	}

	good := x.export(t, "admin-a", "?subject_id="+rid).Body.String()
	lines := strings.Split(strings.TrimSuffix(good, "\n"), "\n")
	first := -1 // the first full entry line
	for i, l := range lines {
		if strings.HasPrefix(l, `{"kind":"entry"`) {
			first = i
			break
		}
	}
	if first < 0 {
		t.Fatal("control: no entry line")
	}
	edited := append([]string(nil), lines...)
	edited[first] = strings.Replace(edited[first], `"source":"`, `"source":"x`, 1)
	if edited[first] == lines[first] {
		t.Fatal("control: the entry line was not changed")
	}
	// The first range line (line index 2, right after header and anchor).
	noFirst := append(append([]string(nil), lines[:2]...), lines[3:]...)
	for name, got := range map[string][]string{"byte edit": edited, "deleted first entry": noFirst} {
		if rep := report(t, []byte(strings.Join(got, "\n")+"\n")); rep.Outcome != verify.OutcomeTampered {
			t.Errorf("%s: outcome %s, want tampered", name, rep.Outcome)
		}
	}

	b := x.export(t, "admin-b", "").Body.String()
	if strings.Contains(b, x.a.id.String()) || strings.Contains(b, rid) {
		t.Error("tenant B's export holds tenant A's data")
	}
}

// ADR-004-E2 acceptance against the real exporter and Postgres.
func TestExportStubsFalse(t *testing.T) {
	x := newExportEnv(t)
	rid := x.seedActivity(t)

	// (1) every entry matches: pass.
	rec := x.export(t, "admin-a", "?stubs=false")
	if rep := report(t, rec.Body.Bytes()); rep.Outcome != verify.OutcomePass {
		t.Errorf("stubs=false, nothing excluded: %s %+v", rep.Outcome, rep.Failure)
	}
	// The filter excludes entries and the tail: unverifiable, last_seq is the head.
	rec = x.export(t, "admin-a", "?stubs=false&subject_id="+rid)
	rep := report(t, rec.Body.Bytes())
	if rep.Outcome != verify.OutcomeUnverifiable {
		t.Errorf("stubs=false, tail excluded: %s %+v", rep.Outcome, rep.Failure)
	}
	var head int64
	if err := x.super.QueryRow(t.Context(), "SELECT seq FROM audit_chain_head WHERE tenant_id = $1", x.a.id).Scan(&head); err != nil {
		t.Fatal(err)
	}
	if rep.LastSeq == nil || *rep.LastSeq != head {
		t.Errorf("trailer last_seq = %v, want the head read at the start (%d)", rep.LastSeq, head)
	}
	// (2) sealed entries in the window, none match: unverifiable, not tampered.
	rec = x.export(t, "admin-a", "?stubs=false&subject_id="+uuid.NewString())
	if rep := report(t, rec.Body.Bytes()); rep.Outcome != verify.OutcomeUnverifiable {
		t.Errorf("stubs=false, nothing matches: %s %+v", rep.Outcome, rep.Failure)
	}
}

// CI-A8: the export's own entry commits before the first byte, and a second
// concurrent export gets 409 export_in_progress. R21: each page is its own
// transaction with statement_timeout = 30s. The hook runs inside the first
// page, so the lease is held and the entry has committed.
//
// covers: exportAuditEntries 409
func TestExportEntryFirstLeaseAndPageTransactions(t *testing.T) {
	x := newExportEnv(t)
	x.seedActivity(t)

	var second *httptest.ResponseRecorder
	var entryAtFirstPage int
	var once sync.Once
	x.onPage = func() {
		once.Do(func() {
			if err := x.super.QueryRow(context.Background(), "SELECT count(*) FROM audit_log WHERE tenant_id = $1 AND action = 'export' AND subject_type = 'audit_export'", x.a.id).Scan(&entryAtFirstPage); err != nil {
				t.Error(err)
			}
			second = x.export(t, "admin-a", "")
		})
	}
	rec := x.export(t, "admin-a", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("export = %d: %s", rec.Code, rec.Body)
	}
	if entryAtFirstPage != 1 {
		t.Errorf("audit/export entries committed when the first page ran = %d, want 1", entryAtFirstPage)
	}
	if second == nil || second.Code != http.StatusConflict || patchProblemCode(t, second) != "export_in_progress" {
		t.Fatalf("second concurrent export = %v", second)
	}
	// The header's export_id is that entry's id.
	var h struct {
		ExportID string `json:"export_id"`
	}
	if err := json.Unmarshal([]byte(strings.SplitN(rec.Body.String(), "\n", 2)[0]), &h); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := x.super.QueryRow(t.Context(), "SELECT count(*) FROM audit_log WHERE tenant_id = $1 AND id = $2 AND action = 'export'", x.a.id, h.ExportID).Scan(&n); err != nil || n != 1 {
		t.Errorf("entry for export_id %s: %d, %v", h.ExportID, n, err)
	}

	// No long snapshot: every page began its own transaction, each with the
	// R21 statement_timeout.
	x.mu.Lock()
	starts, tos := append([]time.Time(nil), x.starts...), append([]string(nil), x.tos...)
	x.mu.Unlock()
	if len(starts) < 4 {
		t.Fatalf("only %d page transactions with pages of two", len(starts))
	}
	seen := map[time.Time]bool{}
	for i, s := range starts {
		if seen[s] {
			t.Errorf("page %d shares a transaction with an earlier page", i)
		}
		seen[s] = true
		if tos[i] != "30s" {
			t.Errorf("page %d statement_timeout = %q, want 30s", i, tos[i])
		}
	}

	// The lease is released: another export runs.
	if rec := x.export(t, "admin-a", ""); rec.Code != http.StatusOK {
		t.Errorf("export after the first finished = %d: %s", rec.Code, rec.Body)
	}
	// The lease is per tenant: B was never blocked.
	if rec := x.export(t, "admin-b", ""); rec.Code != http.StatusOK {
		t.Errorf("tenant B export = %d", rec.Code)
	}
}

// The gate and the validation.
//
// covers: exportAuditEntries 400 401 403
func TestExportGateAndValidation(t *testing.T) {
	x := newExportEnv(t)
	if rec := x.export(t, "user-a", ""); rec.Code != http.StatusForbidden || patchProblemCode(t, rec) != "forbidden" {
		t.Errorf("member without audit.export = %d: %s", rec.Code, rec.Body)
	}
	// The gate comes before the filter is looked at.
	if rec := x.export(t, "user-a", "?from=2030-01-02T00:00:00Z&to=2030-01-01T00:00:00Z"); rec.Code != http.StatusForbidden {
		t.Errorf("member with a bad window = %d, want 403", rec.Code)
	}
	if rec := x.export(t, "", ""); rec.Code != http.StatusUnauthorized {
		t.Errorf("no token = %d", rec.Code)
	}
	if rec := x.export(t, "admin-a", "?from=2030-01-02T00:00:00Z&to=2030-01-01T00:00:00Z"); rec.Code != http.StatusBadRequest {
		t.Errorf("from after to = %d: %s", rec.Code, rec.Body)
	}
	if rec := x.export(t, "admin-a", "?domain=nope"); rec.Code != http.StatusBadRequest {
		t.Errorf("unknown domain = %d", rec.Code)
	}
	if rec := x.export(t, "admin-a", "?after_seq=-1"); rec.Code != http.StatusBadRequest {
		t.Errorf("negative after_seq = %d", rec.Code)
	}
	// A refused export takes no lease and writes no entry.
	var n int
	if err := x.super.QueryRow(t.Context(), "SELECT count(*) FROM audit_log WHERE action = 'export'").Scan(&n); err != nil || n != 0 {
		t.Errorf("refused exports left %d entries (%v)", n, err)
	}
	// A tenant with an empty log exports an empty, verifiable-as-empty file.
	rec := x.export(t, "admin-a", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("empty export = %d: %s", rec.Code, rec.Body)
	}
	if rep := report(t, rec.Body.Bytes()); rep.Outcome != verify.OutcomeEmpty {
		t.Errorf("a tenant with nothing sealed: %s %+v", rep.Outcome, rep.Failure)
	}
}

// R19: an export that cannot commit its entry leaves no lease behind.
//
// covers: exportAuditEntries 500
func TestExportWithoutHeadIs500AndTakesNothing(t *testing.T) {
	x := newExportEnv(t)
	if _, err := x.super.Exec(t.Context(), "DELETE FROM audit_chain_head WHERE tenant_id = $1", x.a.id); err != nil {
		t.Fatal(err)
	}
	if rec := x.export(t, "admin-a", ""); rec.Code != http.StatusInternalServerError {
		t.Errorf("no head = %d: %s", rec.Code, rec.Body)
	}
	var n int
	if err := x.super.QueryRow(t.Context(), "SELECT count(*) FROM audit_log WHERE action = 'export'").Scan(&n); err != nil || n != 0 {
		t.Errorf("entries after a failed begin = %d (%v)", n, err)
	}
}

// R7, CI-A14: a redacted entry carries no diff_salt in the export; an
// unredacted one carries it.
func TestExportRedactedEntriesCarryNoSalt(t *testing.T) {
	x := newExportEnv(t, server.WithAuditMasker(read.StaticMasker{patchwalk.Pointer("/short_description")}))
	x.seedActivity(t)
	rec := x.export(t, "admin-a", "?domain=record")
	var redacted, plain int
	for _, ln := range strings.Split(strings.TrimSuffix(rec.Body.String(), "\n"), "\n") {
		var e struct {
			Kind     string  `json:"kind"`
			DiffSalt *string `json:"diff_salt"`
			Redacted bool    `json:"diff_redacted"`
		}
		if err := json.Unmarshal([]byte(ln), &e); err != nil {
			t.Fatal(err)
		}
		if e.Kind != "entry" {
			continue
		}
		if e.Redacted {
			redacted++
			if e.DiffSalt != nil {
				t.Error("a redacted entry carries diff_salt")
			}
		} else {
			plain++
			if e.DiffSalt == nil {
				t.Error("an unredacted entry carries no diff_salt")
			}
		}
	}
	if redacted == 0 || plain == 0 {
		t.Errorf("redacted %d, unredacted %d: want some of each", redacted, plain)
	}
	if rep := report(t, rec.Body.Bytes()); rep.Outcome != verify.OutcomePass {
		t.Errorf("outcome %s %+v", rep.Outcome, rep.Failure)
	}
}

//go:build integration

// P5-35 (TICK-446): POST /api/v1/audit/chain:reseed against a real
// PostgreSQL, through the real handler and the real sealer (ADR-004 R17,
// ADR-001-A6.3, CI-A10, CI-A16).

package server_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Navaladi2019/TicketingSystem/internal/audit/seal"
	"github.com/Navaladi2019/TicketingSystem/internal/workerguard"
)

const reseedPath = "/api/v1/audit/chain:reseed"

// lineage points tenant's head at a fresh seal start row at the head's
// version, with next_seal_at offset from now (a Postgres interval).
func (x *exportEnv) lineage(t *testing.T, tenant uuid.UUID, offset string) uuid.UUID {
	t.Helper()
	o := uuid.Must(uuid.NewV7())
	ctx := t.Context()
	if _, err := x.super.Exec(ctx, `INSERT INTO outbox (id, tenant_id, kind, kind_version, dedupe_key, target_key, target_seq,
		causation_id, causation_depth, payload) VALUES ($1::uuid, $2::uuid, 'audit.seal', 1, 'audit.seal:' || $3::text, 'audit.seal:' || $3::text,
		(SELECT version FROM audit_chain_head WHERE tenant_id = $2::uuid), $1::uuid, 0, '{}')`, o, tenant, o.String()); err != nil {
		t.Fatal(err)
	}
	if _, err := x.super.Exec(ctx, `UPDATE audit_chain_head SET next_seal_outbox_id = $2,
		next_seal_at = now() + $3::interval WHERE tenant_id = $1`, tenant, o, offset); err != nil {
		t.Fatal(err)
	}
	return o
}

// reseed posts the reseed with a fresh Idempotency-Key, which every tenant
// write requires.
func (x *exportEnv) reseed(t *testing.T, token string) *httptest.ResponseRecorder {
	t.Helper()
	return x.do(t, http.MethodPost, reseedPath, token, uuid.NewString(), "")
}

type headState struct {
	version int64
	next    uuid.NullUUID
}

func (x *exportEnv) head(t *testing.T, tenant uuid.UUID) headState {
	t.Helper()
	var h headState
	if err := x.super.QueryRow(t.Context(), "SELECT version, next_seal_outbox_id FROM audit_chain_head WHERE tenant_id = $1", tenant).Scan(&h.version, &h.next); err != nil {
		t.Fatal(err)
	}
	return h
}

func (x *exportEnv) count(t *testing.T, sql string, args ...any) int {
	t.Helper()
	var n int
	if err := x.super.QueryRow(t.Context(), sql, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// CI-A16: 409 while alive, nothing written; after a dead lineage the reseed
// collapses two lineages to one and sealing resumes; the entry exists; the
// version is not bumped and the new row is not the dead one's.
//
// covers: reseedAuditChain 200
// covers: reseedAuditChain 409
func TestReseedAliveRefusedThenDeadRestarts(t *testing.T) {
	x := newExportEnv(t)
	dead := x.lineage(t, x.a.id, "-5 minutes") // late, but not dead: R17 says 10 minutes
	before := x.head(t, x.a.id)

	rec := x.reseed(t, "admin-a")
	if rec.Code != http.StatusConflict || patchProblemCode(t, rec) != "chain_alive" {
		t.Fatalf("alive reseed = %d: %s", rec.Code, rec.Body)
	}
	if got := x.head(t, x.a.id); got != before {
		t.Errorf("a refused reseed changed the head: %+v -> %+v", before, got)
	}
	if n := x.count(t, "SELECT count(*) FROM audit_log WHERE action = 'reseed'"); n != 0 {
		t.Errorf("a refused reseed wrote %d entries", n)
	}
	if n := x.count(t, "SELECT count(*) FROM outbox WHERE tenant_id = $1", x.a.id); n != 1 {
		t.Errorf("a refused reseed left %d outbox rows, want 1", n)
	}

	if _, err := x.super.Exec(t.Context(), "UPDATE audit_chain_head SET next_seal_at = now() - interval '11 minutes' WHERE tenant_id = $1", x.a.id); err != nil {
		t.Fatal(err)
	}
	rec = x.reseed(t, "admin-a")
	if rec.Code != http.StatusOK {
		t.Fatalf("dead reseed = %d: %s", rec.Code, rec.Body)
	}
	var out struct {
		ReseedID     uuid.UUID `json:"reseed_id"`
		SealOutboxID uuid.UUID `json:"seal_outbox_id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	after := x.head(t, x.a.id)
	if after.version != before.version {
		t.Errorf("version %d -> %d: reseed must not bump it", before.version, after.version)
	}
	if !after.next.Valid || after.next.UUID != out.SealOutboxID || out.SealOutboxID == dead {
		t.Errorf("head points at %v, response %s, dead %s: want the new row", after.next.UUID, out.SealOutboxID, dead)
	}
	var targetSeq int64
	if err := x.super.QueryRow(t.Context(), "SELECT target_seq FROM outbox WHERE id = $1", out.SealOutboxID).Scan(&targetSeq); err != nil || targetSeq != before.version {
		t.Errorf("new row target_seq %d (%v), want the head version %d", targetSeq, err, before.version)
	}
	if n := x.count(t, `SELECT count(*) FROM audit_log WHERE tenant_id = $1 AND id = $2 AND domain = 'audit'
		AND action = 'reseed' AND subject_type = 'audit_chain'`, x.a.id, out.ReseedID); n != 1 {
		t.Fatalf("%d audit/reseed entries with id %s, want 1", n, out.ReseedID)
	}

	// Two lineages become one: the dead lineage's run finds itself stale and
	// writes nothing; the new one's reschedules.
	ctx := workerguard.ContextWithTenant(context.Background(), x.a.id)
	if id, err := seal.Reschedule(ctx, x.worker, dead); err != nil || id != uuid.Nil {
		t.Errorf("Reschedule(dead) = %s, %v; want a no-op", id, err)
	}
	next, err := seal.Reschedule(ctx, x.worker, out.SealOutboxID)
	if err != nil || next == uuid.Nil {
		t.Fatalf("Reschedule(new) = %s, %v; sealing did not resume", next, err)
	}
	if got := x.head(t, x.a.id); !got.next.Valid || got.next.UUID != next || got.version != before.version+1 {
		t.Errorf("head after resume %+v, want next %s version %d", got, next, before.version+1)
	}

	// The lineage is alive again: a second call is refused.
	if rec := x.reseed(t, "admin-a"); rec.Code != http.StatusConflict {
		t.Errorf("second reseed = %d, want 409", rec.Code)
	}
}

// Auth, the tenant boundary and the unknown head (TICK-590 constraints).
//
// covers: reseedAuditChain 400 401 403
// covers: reseedAuditChain 404
func TestReseedGateTenantAndMissingHead(t *testing.T) {
	x := newExportEnv(t)
	x.lineage(t, x.a.id, "-20 minutes")
	bOld := x.lineage(t, x.b.id, "-20 minutes")

	if rec := x.reseed(t, "user-a"); rec.Code != http.StatusForbidden || patchProblemCode(t, rec) != "forbidden" {
		t.Errorf("member without audit.export = %d: %s", rec.Code, rec.Body)
	}
	if rec := x.do(t, http.MethodPost, reseedPath, "admin-a", "", ""); rec.Code != http.StatusBadRequest {
		t.Errorf("no Idempotency-Key = %d, want 400", rec.Code)
	}
	if rec := x.reseed(t, ""); rec.Code != http.StatusUnauthorized {
		t.Errorf("no token = %d", rec.Code)
	}
	if n := x.count(t, "SELECT count(*) FROM audit_log WHERE action = 'reseed'"); n != 0 {
		t.Errorf("refused calls wrote %d entries", n)
	}

	// Tenant A's admin restarts A only; B's dead head is untouched.
	if rec := x.reseed(t, "admin-a"); rec.Code != http.StatusOK {
		t.Fatalf("admin-a = %d: %s", rec.Code, rec.Body)
	}
	if h := x.head(t, x.b.id); !h.next.Valid || h.next.UUID != bOld {
		t.Errorf("tenant B's head changed: %+v", h)
	}
	if n := x.count(t, "SELECT count(*) FROM audit_log WHERE action = 'reseed' AND tenant_id = $1", x.b.id); n != 0 {
		t.Errorf("tenant B has %d reseed entries", n)
	}

	// No head: 404, never 409, and nothing is written.
	if _, err := x.super.Exec(t.Context(), "DELETE FROM audit_chain_head WHERE tenant_id = $1", x.b.id); err != nil {
		t.Fatal(err)
	}
	rec := x.reseed(t, "admin-b")
	if rec.Code != http.StatusNotFound || patchProblemCode(t, rec) != "not_found" {
		t.Errorf("no head = %d: %s", rec.Code, rec.Body)
	}
	if n := x.count(t, "SELECT count(*) FROM outbox WHERE tenant_id = $1 AND id <> $2", x.b.id, bOld); n != 0 {
		t.Errorf("a 404 reseed left %d outbox rows", n)
	}
}

// CI-A10: nava_app repoints the lineage but cannot write seq, entry_hash or
// version on the head.
func TestReseedAppCannotWriteChainColumns(t *testing.T) {
	x := newExportEnv(t)
	for _, set := range []string{"seq = 5", "entry_hash = decode(repeat('00', 32), 'hex')", "version = 9"} {
		_, err := x.pool.Exec(t.Context(), "UPDATE audit_chain_head SET "+set)
		var pg *pgconn.PgError
		if !errors.As(err, &pg) || pg.Code != "42501" {
			t.Errorf("nava_app UPDATE %s: %v, want SQLSTATE 42501", set, err)
		}
	}
}

// C5 amendment: a retry with the same Idempotency-Key replays the first
// answer, and the reseed happened once: one entry, one new outbox row.
//
// covers: reseedAuditChain 200
func TestReseedRetryWithSameKeyIsIdempotent(t *testing.T) {
	x := newExportEnv(t)
	x.lineage(t, x.a.id, "-20 minutes")
	key := uuid.NewString()
	first := x.do(t, http.MethodPost, reseedPath, "admin-a", key, "")
	second := x.do(t, http.MethodPost, reseedPath, "admin-a", key, "")
	if first.Code != http.StatusOK || second.Code != http.StatusOK {
		t.Fatalf("first %d: %s; retry %d: %s", first.Code, first.Body, second.Code, second.Body)
	}
	if first.Body.String() != second.Body.String() {
		t.Errorf("retry answered %s, want the first answer %s", second.Body, first.Body)
	}
	if n := x.count(t, "SELECT count(*) FROM audit_log WHERE tenant_id = $1 AND action = 'reseed'", x.a.id); n != 1 {
		t.Errorf("%d reseed entries after a retry, want 1", n)
	}
	if n := x.count(t, "SELECT count(*) FROM outbox WHERE tenant_id = $1", x.a.id); n != 2 {
		t.Errorf("%d outbox rows after a retry, want 2 (dead lineage, one reseed)", n)
	}
}

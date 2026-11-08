//go:build integration

// P5-42 (TICK-447): erasure end to end and export isolation, against a real
// PostgreSQL, through the real handler, the real sealer, the real offline
// verifier and the test-only erase helper logged in as nava_erasure
// (ADR-004 R11, R17, R19-R24; CI-A7, CI-A4 export/stubs/reseed, CI-A5 export
// bytes; PLAN-005 D-5-10; the TICK-449 grooming additions).
//
// The attacker holds a valid token, or a database login, for one tenant and
// aims at the other tenant's entries. Every denial has a positive control in
// the same test, so a check that sees nothing cannot pass.

package server_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/internal/audit/erasetest"
	"github.com/Navaladi2019/TicketingSystem/internal/audit/verify"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// rolePool is a pool logged in as role, directly (nava_erasure is not on
// PgBouncer's allow-list, and its credential never sits on an app host).
func (x *exportEnv) rolePool(t *testing.T, role string) *pgxpool.Pool {
	t.Helper()
	u, err := url.Parse(os.Getenv("NAVA_TEST_PG_URL"))
	if err != nil {
		t.Fatal(err)
	}
	u.User = url.UserPassword(role, etPassword(role))
	u.Path = "/" + x.pool.Config().ConnConfig.Database
	cfg, err := pgxpool.ParseConfig(u.String())
	if err != nil {
		t.Fatal(err)
	}
	cfg.MaxConns = 2
	p, err := pgxpool.NewWithConfig(t.Context(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(p.Close)
	return p
}

// erasable is tenant's first sealed record entry about subject that still has
// a diff: its id, seq and entry_hash.
func (x *exportEnv) erasable(t *testing.T, tenant uuid.UUID, subject string) (uuid.UUID, int64, []byte) {
	t.Helper()
	var id uuid.UUID
	var seq int64
	var h []byte
	if err := x.super.QueryRow(t.Context(), `SELECT id, seq, entry_hash FROM audit_log
		WHERE tenant_id = $1 AND subject_id = $2 AND domain = 'record' AND seq IS NOT NULL AND diff IS NOT NULL
		ORDER BY seq LIMIT 1`, tenant, subject).Scan(&id, &seq, &h); err != nil {
		t.Fatalf("control: no erasable entry of %s about %s: %v", tenant, subject, err)
	}
	return id, seq, h
}

// diffState is the row's erasure state as the superuser sees it.
type diffState struct {
	hasDiff, hasSalt bool
	erased           bool
	ref              uuid.NullUUID
}

func (x *exportEnv) diffState(t *testing.T, id uuid.UUID) diffState {
	t.Helper()
	var s diffState
	if err := x.super.QueryRow(t.Context(), `SELECT diff IS NOT NULL, diff_salt IS NOT NULL, erased_at IS NOT NULL, erasure_ref
		FROM audit_log WHERE id = $1`, id).Scan(&s.hasDiff, &s.hasSalt, &s.erased, &s.ref); err != nil {
		t.Fatal(err)
	}
	return s
}

func sqlStateOf(err error) string {
	var pe *pgconn.PgError
	if errors.As(err, &pe) {
		return pe.Code
	}
	return ""
}

// asSuper runs fn as the superuser with triggers off for this transaction
// only: the forger who can write the table directly (CI-A7's tampered
// cases).
func (x *exportEnv) asSuper(t *testing.T, fn func(tx pgx.Tx) error) {
	t.Helper()
	ctx := t.Context()
	tx, err := x.super.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, "SET LOCAL session_replication_role = replica"); err != nil {
		t.Fatal(err)
	}
	if err := fn(tx); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
}

func exportOK(t *testing.T, x *exportEnv, token, query string) []byte {
	t.Helper()
	rec := x.export(t, token, query)
	if rec.Code != http.StatusOK {
		t.Fatalf("export %s%s = %d: %s", token, query, rec.Code, rec.Body)
	}
	return rec.Body.Bytes()
}

// CI-A7, R23, R24: nava_erasure erases a sealed entry through the helper; the
// R11 transition nulls diff and salt and sets erased_at and erasure_ref, and
// leaves the hashed envelope and the seal alone. While the audit/erase entry
// is unsealed the export verifies as pending; once it is sealed the export
// verifies, unfiltered and under a filter that excludes the erase entry
// itself (R24: the export always includes it).
//
// covers: exportAuditEntries 200
func TestCIA7EraseVerifiesPendingThenPass(t *testing.T) {
	x := newExportEnv(t)
	rid := x.seedActivity(t)
	ctx := t.Context()
	er := x.rolePool(t, erasetest.Role)
	xID, xSeq, xHash := x.erasable(t, x.a.id, rid)

	ref := uuid.New()
	eraseID, err := erasetest.Erase(ctx, er, x.a.id, ref, []uuid.UUID{xID})
	if err != nil {
		t.Fatalf("nava_erasure erasing tenant A's entry: %v", err)
	}
	if s := x.diffState(t, xID); s.hasDiff || s.hasSalt || !s.erased || !s.ref.Valid || s.ref.UUID != ref {
		t.Fatalf("after the erase: %+v, want no diff, no salt, erased under %s", s, ref)
	}
	var seq int64
	var h []byte
	if err := x.super.QueryRow(ctx, "SELECT seq, entry_hash FROM audit_log WHERE id = $1", xID).Scan(&seq, &h); err != nil {
		t.Fatal(err)
	}
	if seq != xSeq || !bytes.Equal(h, xHash) {
		t.Errorf("the erase changed the seal: seq %d→%d, entry_hash changed %v", xSeq, seq, !bytes.Equal(h, xHash))
	}
	var subj, by string
	if err := x.super.QueryRow(ctx, `SELECT subject_id::text, actor_system FROM audit_log
		WHERE tenant_id = $1 AND id = $2 AND domain = 'audit' AND action = 'erase' AND subject_type = 'erasure_request' AND seq IS NULL`,
		x.a.id, eraseID).Scan(&subj, &by); err != nil || subj != ref.String() || by != erasetest.ActorSystem {
		t.Fatalf("audit/erase entry %s: subject %s actor %s (%v)", eraseID, subj, by, err)
	}

	// The erase entry is not sealed yet: pending, never pass.
	rep := report(t, exportOK(t, x, "admin-a", ""))
	if rep.Outcome != verify.OutcomePending || rep.Lines[xSeq] != verify.LinePending {
		t.Fatalf("unsealed erase entry: outcome %s, line %s, failure %+v; want pending", rep.Outcome, rep.Lines[xSeq], rep.Failure)
	}

	x.sealAll(t)
	for _, q := range []string{"", "?subject_id=" + rid, "?domain=record"} {
		body := exportOK(t, x, "admin-a", q)
		rep := report(t, body)
		if line, flagged := rep.Lines[xSeq]; rep.Outcome != verify.OutcomePass || flagged {
			t.Errorf("sealed erase entry, filter %q: outcome %s, line %s, failure %+v; want pass, the erased entry verified", q, rep.Outcome, line, rep.Failure)
		}
		if !bytes.Contains(body, []byte(eraseID.String())) {
			t.Errorf("filter %q: the export does not carry the audit/erase entry %s (R24)", q, eraseID)
		}
	}
}

// CI-A7: a second erase is rejected, by the helper (it erases nothing and
// writes no second audit/erase entry) and, for an UPDATE that reaches the
// erased row, by the R11 trigger.
func TestCIA7SecondEraseIsRejected(t *testing.T) {
	x := newExportEnv(t)
	rid := x.seedActivity(t)
	ctx := t.Context()
	er := x.rolePool(t, erasetest.Role)
	xID, _, _ := x.erasable(t, x.a.id, rid)
	ref := uuid.New()
	if _, err := erasetest.Erase(ctx, er, x.a.id, ref, []uuid.UUID{xID}); err != nil {
		t.Fatalf("control: first erase: %v", err)
	}

	for _, r := range []uuid.UUID{ref, uuid.New()} {
		if _, err := erasetest.Erase(ctx, er, x.a.id, r, []uuid.UUID{xID}); !errors.Is(err, erasetest.ErrNotErasable) {
			t.Errorf("second erase under %s through the helper: %v, want ErrNotErasable", r, err)
		}
		err := tenancy.InTenantTx(ctx, er, x.a.id, func(tx pgx.Tx) error {
			_, err := tx.Exec(ctx, `UPDATE audit_log SET diff = NULL, diff_salt = NULL, erased_at = now(), erasure_ref = $2
				WHERE id = $1`, xID, r)
			return err
		})
		if err == nil || !strings.Contains(err.Error(), "neither the seal nor the erase transition") {
			t.Errorf("second erase transition on the erased row: %v, want the R11 trigger's refusal", err)
		}
	}
	if s := x.diffState(t, xID); s.ref.UUID != ref {
		t.Errorf("the erasure_ref moved to %s, want %s", s.ref.UUID, ref)
	}
	if n := x.count(t, "SELECT count(*) FROM audit_log WHERE tenant_id = $1 AND action = 'erase'", x.a.id); n != 1 {
		t.Errorf("%d audit/erase entries, want 1", n)
	}
}

// CI-A7, R23: nava_erasure cannot read diff or diff_salt, by any way of
// naming them; it reads only the M1 columns.
func TestCIA7ErasureCannotReadDiff(t *testing.T) {
	x := newExportEnv(t)
	rid := x.seedActivity(t)
	ctx := t.Context()
	er := x.rolePool(t, erasetest.Role)
	xID, _, _ := x.erasable(t, x.a.id, rid)

	var n int
	if err := tenancy.InTenantTx(ctx, er, x.a.id, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `SELECT count(*) FROM audit_log WHERE id = $1 AND erased_at IS NULL AND erasure_ref IS NULL
			AND subject_type IS NOT NULL AND subject_id IS NOT NULL AND occurred_at IS NOT NULL AND tenant_id IS NOT NULL`, xID).Scan(&n)
	}); err != nil || n != 1 {
		t.Fatalf("control: nava_erasure reads the M1 columns of tenant A's entry: %d, %v", n, err)
	}
	for _, q := range []string{
		"SELECT diff FROM audit_log WHERE id = $1",
		"SELECT diff_salt FROM audit_log WHERE id = $1",
		"SELECT diff::text FROM audit_log WHERE id = $1",
		"SELECT id FROM audit_log WHERE id = $1 AND diff IS NOT NULL",
		"SELECT id FROM audit_log WHERE id = $1 ORDER BY diff_salt",
		"SELECT * FROM audit_log WHERE id = $1",
		"SELECT to_jsonb(a) FROM audit_log a WHERE id = $1",
		"UPDATE audit_log SET erasure_ref = erasure_ref WHERE id = $1 RETURNING diff",
	} {
		err := tenancy.InTenantTx(ctx, er, x.a.id, func(tx pgx.Tx) error {
			rows, err := tx.Query(ctx, q, xID)
			if err != nil {
				return err
			}
			rows.Close()
			return rows.Err()
		})
		if sqlStateOf(err) != "42501" {
			t.Errorf("nava_erasure: %s: %v, want 42501", q, err)
		}
	}
}

// CI-A7: nava_app and nava_worker cannot erase. The R23 step, run as either
// in tenant A's own transaction, is refused by the database (no UPDATE grant
// on diff, diff_salt or the erasure columns), and the entry keeps its diff;
// the helper's own role check refuses them before any SQL. Control:
// nava_erasure then erases the same entry.
func TestCIA7AppAndWorkerCannotErase(t *testing.T) {
	x := newExportEnv(t)
	rid := x.seedActivity(t)
	ctx := t.Context()
	xID, _, _ := x.erasable(t, x.a.id, rid)

	for _, role := range []string{"nava_app", "nava_worker"} {
		p := x.rolePool(t, role)
		err := tenancy.InTenantTx(ctx, p, x.a.id, func(tx pgx.Tx) error {
			_, err := erasetest.Step(ctx, tx, uuid.New(), []uuid.UUID{xID})
			return err
		})
		// The grant is the first wall: 42501 "permission denied". The R11
		// trigger also raises 42501, with its own message, so the message
		// pins which layer refused; a grant regression shows here even while
		// the trigger still holds.
		if sqlStateOf(err) != "42501" || !strings.Contains(err.Error(), "permission denied for table audit_log") {
			t.Errorf("%s running the erase step: %v, want 42501 permission denied (the column grant)", role, err)
		}
		// The helper's own role check refuses them before any SQL.
		if _, err := erasetest.Erase(ctx, p, x.a.id, uuid.New(), []uuid.UUID{xID}); !errors.Is(err, erasetest.ErrWrongRole) {
			t.Errorf("%s through the helper: %v, want ErrWrongRole", role, err)
		}
	}
	if s := x.diffState(t, xID); !s.hasDiff || !s.hasSalt || s.erased {
		t.Fatalf("after nava_app and nava_worker tried: %+v, want the diff intact", s)
	}
	if n := x.count(t, "SELECT count(*) FROM audit_log WHERE action = 'erase'"); n != 0 {
		t.Fatalf("%d audit/erase entries after refused erases, want 0", n)
	}
	if _, err := erasetest.Erase(ctx, x.rolePool(t, erasetest.Role), x.a.id, uuid.New(), []uuid.UUID{xID}); err != nil {
		t.Fatalf("control: nava_erasure erasing the same entry: %v", err)
	}
}

// CI-A7, R24: a nulled diff with no erase entry is tampered. The forger is
// the superuser, past every grant and trigger: (a) the erasure columns set
// with no audit/erase entry; (b) the diff dropped with no erasure columns;
// (c) an audit/erase entry under the same ref that does not list the entry.
// Each is restored before the next, and the restored file passes, so each
// tampered verdict is the forgery's.
func TestCIA7NulledDiffWithoutEraseEntryIsTampered(t *testing.T) {
	x := newExportEnv(t)
	rid := x.seedActivity(t)
	ctx := t.Context()
	xID, xSeq, _ := x.erasable(t, x.a.id, rid)
	var diff string
	var salt []byte
	if err := x.super.QueryRow(ctx, "SELECT diff::text, diff_salt FROM audit_log WHERE id = $1", xID).Scan(&diff, &salt); err != nil {
		t.Fatal(err)
	}
	restore := func() {
		x.asSuper(t, func(tx pgx.Tx) error {
			_, err := tx.Exec(ctx, `UPDATE audit_log SET diff = $2::json, diff_salt = $3, erased_at = NULL, erasure_ref = NULL WHERE id = $1`, xID, diff, salt)
			return err
		})
		if rep := report(t, exportOK(t, x, "admin-a", "")); rep.Outcome != verify.OutcomePass {
			t.Fatalf("control: the restored export is %s %+v, want pass", rep.Outcome, rep.Failure)
		}
	}
	restore()

	ref := uuid.New()
	forgeries := []struct {
		name  string
		forge func(tx pgx.Tx) error
	}{
		{"erasure columns set, no erase entry", func(tx pgx.Tx) error {
			_, err := tx.Exec(ctx, `UPDATE audit_log SET diff = NULL, diff_salt = NULL, erased_at = now(), erasure_ref = $2 WHERE id = $1`, xID, ref)
			return err
		}},
		{"diff dropped, not erased", func(tx pgx.Tx) error {
			_, err := tx.Exec(ctx, `UPDATE audit_log SET diff = NULL, diff_salt = NULL WHERE id = $1`, xID)
			return err
		}},
	}
	for _, f := range forgeries {
		x.asSuper(t, f.forge)
		rep := report(t, exportOK(t, x, "admin-a", ""))
		if rep.Outcome != verify.OutcomeTampered || rep.Failure == nil || rep.Failure.Seq == nil || *rep.Failure.Seq != xSeq {
			t.Errorf("%s: outcome %s, failure %+v; want tampered at seq %d", f.name, rep.Outcome, rep.Failure, xSeq)
		}
		restore()
	}

	// (c) a real, sealed erase of another entry under ref; X is then forged
	// erased under the same ref, which that erase entry does not list.
	r3 := x.mustCreateRec(t, "user-a", "job", "ex-3", `{"short_description": "three"}`)
	x.sealAll(t)
	other, _, _ := x.erasable(t, x.a.id, r3.ID)
	if _, err := erasetest.Erase(ctx, x.rolePool(t, erasetest.Role), x.a.id, ref, []uuid.UUID{other}); err != nil {
		t.Fatalf("control: erase another entry: %v", err)
	}
	x.sealAll(t)
	if rep := report(t, exportOK(t, x, "admin-a", "")); rep.Outcome != verify.OutcomePass {
		t.Fatalf("control: the honest erase verifies as %s %+v", rep.Outcome, rep.Failure)
	}
	x.asSuper(t, forgeries[0].forge)
	rep := report(t, exportOK(t, x, "admin-a", ""))
	if rep.Outcome != verify.OutcomeTampered || rep.Failure == nil || rep.Failure.Seq == nil || *rep.Failure.Seq != xSeq {
		t.Errorf("erased under a ref whose erase entry does not list it: outcome %s, failure %+v; want tampered at seq %d", rep.Outcome, rep.Failure, xSeq)
	}
}

// TICK-449 grooming, CI-A4: with app.tenant_id = B, nava_erasure naming
// tenant A's entry ids updates 0 rows, and the helper refuses before it
// writes an audit/erase entry listing A's ids; mixed with one of B's own ids,
// B's is not erased either. With no tenant set it reaches nothing. Control:
// in tenant A the same erase succeeds.
func TestCIA4EraseNeverCrossesTenants(t *testing.T) {
	x := newExportEnv(t)
	rid := x.seedActivity(t)
	bRec := x.mustCreateRec(t, "user-b", "job", "eb-1", `{"short_description": "b"}`)
	x.sealAll(t)
	ctx := t.Context()
	er := x.rolePool(t, erasetest.Role)
	aID, _, _ := x.erasable(t, x.a.id, rid)
	bID, _, _ := x.erasable(t, x.b.id, bRec.ID)
	ref := uuid.New()

	if _, err := erasetest.Erase(ctx, er, x.b.id, ref, []uuid.UUID{aID}); !errors.Is(err, erasetest.ErrNotErasable) {
		t.Errorf("tenant B erasing tenant A's entry: %v, want ErrNotErasable", err)
	}
	if _, err := erasetest.Erase(ctx, er, x.b.id, ref, []uuid.UUID{aID, bID}); !errors.Is(err, erasetest.ErrNotErasable) {
		t.Errorf("tenant B erasing A's entry with one of its own: %v, want ErrNotErasable", err)
	}
	var reached int64 = -1
	if err := tenancy.InTenantTx(ctx, er, x.b.id, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `UPDATE audit_log SET diff = NULL, diff_salt = NULL, erased_at = now(), erasure_ref = $2
			WHERE id = $1`, aID, ref)
		reached = tag.RowsAffected()
		return err
	}); err != nil || reached != 0 {
		t.Errorf("tenant B's raw erase transition on A's id: %d rows, %v; want 0 rows", reached, err)
	}
	// No tenant: RLS's NULL predicate reaches nothing, read or write.
	if err := pgx.BeginFunc(ctx, er, func(tx pgx.Tx) error {
		var n int
		if err := tx.QueryRow(ctx, "SELECT count(*) FROM audit_log WHERE id = ANY($1::uuid[])", []uuid.UUID{aID, bID}).Scan(&n); err != nil || n != 0 {
			return fmt.Errorf("no tenant: read %d of A's and B's entries (%w), want 0", n, err)
		}
		if _, err := erasetest.Step(ctx, tx, ref, []uuid.UUID{aID}); !errors.Is(err, erasetest.ErrNotErasable) {
			return fmt.Errorf("no tenant: erase step = %w, want ErrNotErasable", err)
		}
		return nil
	}); err != nil {
		t.Error(err)
	}
	for id, tn := range map[uuid.UUID]string{aID: "A", bID: "B"} {
		if s := x.diffState(t, id); !s.hasDiff || s.erased {
			t.Errorf("tenant %s's entry after the cross-tenant erases: %+v, want intact", tn, s)
		}
	}
	if n := x.count(t, "SELECT count(*) FROM audit_log WHERE subject_id = $1 OR action = 'erase'", ref); n != 0 {
		t.Errorf("%d audit/erase entries after refused cross-tenant erases, want 0", n)
	}
	if _, err := erasetest.Erase(ctx, er, x.a.id, ref, []uuid.UUID{aID}); err != nil {
		t.Fatalf("control: tenant A erasing its own entry: %v", err)
	}
}

// tenantNeedles are the strings that would show tenant A's data in another
// tenant's export: its id, every entry id, every non-genesis prev_hash and
// entry_hash (lowercase hex, as audit-export-v1.md encodes bytea) and its
// erasure refs. extra are added (record ids, principals).
func (x *exportEnv) tenantNeedles(t *testing.T, tenant uuid.UUID, extra ...string) []string {
	t.Helper()
	rows, err := x.super.Query(t.Context(), `SELECT id::text FROM audit_log WHERE tenant_id = $1
		UNION SELECT encode(entry_hash, 'hex') FROM audit_log WHERE tenant_id = $1 AND entry_hash IS NOT NULL
		UNION SELECT encode(prev_hash, 'hex') FROM audit_log WHERE tenant_id = $1 AND prev_hash IS NOT NULL AND prev_hash <> '\x0000000000000000000000000000000000000000000000000000000000000000'::bytea
		UNION SELECT erasure_ref::text FROM audit_log WHERE tenant_id = $1 AND erasure_ref IS NOT NULL
		UNION SELECT encode(entry_hash, 'hex') FROM audit_chain_head WHERE tenant_id = $1 AND entry_hash IS NOT NULL`, tenant)
	if err != nil {
		t.Fatal(err)
	}
	n, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		t.Fatal(err)
	}
	return append(append(n, tenant.String()), extra...)
}

func leaked(body []byte, needles []string) []string {
	var out []string
	for _, n := range needles {
		if bytes.Contains(body, []byte(n)) {
			out = append(out, n)
		}
	}
	return out
}

// CI-A4 for the export and its stubs: tenant B's export, under every filter
// that turns entries into stubs (and one naming A's own subject), holds
// nothing of tenant A's: no id, no hash of A's chain, no erasure ref. Every
// B file verifies, so B's stubs chain B's own hashes. Control: A's own
// export carries every one of the needles' kinds, so the scan sees them in
// the encoding the export uses.
//
// covers: exportAuditEntries 200
func TestCIA4ExportAndStubsNeverCrossTenants(t *testing.T) {
	x := newExportEnv(t)
	rid := x.seedActivity(t)
	bRec := x.mustCreateRec(t, "user-b", "job", "eb-1", `{"short_description": "b"}`)
	x.mustCreateRec(t, "user-b", "job", "eb-2", `{"short_description": "b2"}`)
	x.sealAll(t)
	ctx := t.Context()
	aID, _, _ := x.erasable(t, x.a.id, rid)
	ref := uuid.New()
	if _, err := erasetest.Erase(ctx, x.rolePool(t, erasetest.Role), x.a.id, ref, []uuid.UUID{aID}); err != nil {
		t.Fatal(err)
	}
	x.sealAll(t)
	needles := x.tenantNeedles(t, x.a.id, x.a.admin.PrincipalID.String(), x.a.user.PrincipalID.String())
	if len(needles) < 12 {
		t.Fatalf("control: only %d needles for tenant A", len(needles))
	}

	aBody := exportOK(t, x, "admin-a", "")
	if rep := report(t, aBody); rep.Outcome != verify.OutcomePass {
		t.Errorf("control: tenant A's export is %s %+v", rep.Outcome, rep.Failure)
	}
	// Control: A's export holds A's tenant id, an entry id, the erasure ref
	// and the head hash; the stubs of a filtered export hold A's hashes.
	var head string
	if err := x.super.QueryRow(ctx, "SELECT encode(entry_hash, 'hex') FROM audit_chain_head WHERE tenant_id = $1", x.a.id).Scan(&head); err != nil {
		t.Fatal(err)
	}
	aStubs := exportOK(t, x, "admin-a", "?domain=config")
	for what, c := range map[string]struct {
		b []byte
		n string
	}{"tenant id": {aBody, x.a.id.String()}, "entry id": {aBody, aID.String()}, "erasure ref": {aBody, ref.String()},
		"head hash in a stub": {aStubs, head}} {
		if !bytes.Contains(c.b, []byte(c.n)) {
			t.Fatalf("control: tenant A's own export does not carry its %s; the scan proves nothing", what)
		}
	}

	withSubject := append(slices.Clone(needles), rid)
	for _, q := range []string{
		"", "?domain=config", "?domain=record", "?domain=audit", "?subject_id=" + bRec.ID,
		"?subject_type=erasure_request", "?after_seq=1", "?stubs=false", "?stubs=false&domain=config",
	} {
		body := exportOK(t, x, "admin-b", q)
		if l := leaked(body, withSubject); len(l) > 0 {
			t.Errorf("tenant B's export %q carries tenant A's %v", q, l)
		}
		if strings.Contains(q, "stubs=false") {
			continue
		}
		if rep := report(t, body); rep.Outcome != verify.OutcomePass {
			t.Errorf("tenant B's export %q: %s %+v", q, rep.Outcome, rep.Failure)
		}
	}
	// Last, B names A's record as its subject. The id comes back only where
	// B supplied it (the header's filter and B's own audit/export entry), so
	// here it is not a needle. The file has the same shape as for an id no
	// tenant has, so it is no existence oracle either.
	body := exportOK(t, x, "admin-b", "?subject_id="+rid)
	if l := leaked(body, needles); len(l) > 0 {
		t.Errorf("tenant B's export filtered on tenant A's subject carries tenant A's %v", l)
	}
	named := report(t, body)
	unknown := report(t, exportOK(t, x, "admin-b", "?subject_id="+uuid.NewString()))
	if named.Outcome != verify.OutcomePass || named.Count != unknown.Count || named.StubCount != unknown.StubCount || named.UnsealedCount != unknown.UnsealedCount {
		t.Errorf("tenant B naming A's subject: %+v; naming an unknown id: %+v; want the same pass shape", named, unknown)
	}
}

// CI-A4, R19 (TICK-449 grooming): the export lease is per tenant. While
// tenant A's export is streaming, and while A's lease is held, tenant B
// exports; A's own second export is the 409 control.
//
// covers: exportAuditEntries 409
func TestCIA4ExportLeaseIsPerTenant(t *testing.T) {
	x := newExportEnv(t)
	x.seedActivity(t)
	x.mustCreateRec(t, "user-b", "job", "eb-1", `{"short_description": "b"}`)
	x.sealAll(t)

	var fired atomic.Bool
	var bCode, aCode int
	var bCodeStr string
	x.onPage = func() {
		if !fired.CompareAndSwap(false, true) {
			return
		}
		rec := x.export(t, "admin-b", "")
		bCode, bCodeStr = rec.Code, rec.Body.String()
		aCode = x.export(t, "admin-a", "").Code
	}
	exportOK(t, x, "admin-a", "")
	if !fired.Load() {
		t.Fatal("control: the page hook did not run")
	}
	if aCode != http.StatusConflict {
		t.Fatalf("control: tenant A's second export during its first = %d, want 409", aCode)
	}
	if bCode != http.StatusOK {
		t.Errorf("tenant B's export while tenant A's streams = %d: %.300s", bCode, bCodeStr)
	}

	x.onPage = nil
	if _, err := x.super.Exec(t.Context(), "UPDATE audit_chain_head SET export_lease_until = now() + interval '1 hour' WHERE tenant_id = $1", x.a.id); err != nil {
		t.Fatal(err)
	}
	if rec := x.export(t, "admin-a", ""); rec.Code != http.StatusConflict || patchProblemCode(t, rec) != "export_in_progress" {
		t.Fatalf("control: tenant A with its lease held = %d: %s", rec.Code, rec.Body)
	}
	if rec := x.export(t, "admin-b", ""); rec.Code != http.StatusOK {
		t.Errorf("tenant B while tenant A's lease is held = %d: %s", rec.Code, rec.Body)
	}
}

// CI-A4, R17 (TICK-449 grooming): reseed's 409 chain_alive reads only the
// caller's own head. Tenant A's dead lineage does not let tenant B reseed
// B's live one, and tenant B's reseed of its dead lineage leaves A's head
// alone; A's own call is the control each way.
//
// covers: reseedAuditChain 200
// covers: reseedAuditChain 409
func TestCIA4ReseedReadsOnlyItsOwnHead(t *testing.T) {
	x := newExportEnv(t)
	aDead := x.lineage(t, x.a.id, "-20 minutes")
	x.lineage(t, x.b.id, "+5 minutes")
	if rec := x.reseed(t, "admin-b"); rec.Code != http.StatusConflict || patchProblemCode(t, rec) != "chain_alive" {
		t.Errorf("tenant B (alive) while tenant A is dead = %d: %s, want 409 chain_alive", rec.Code, rec.Body)
	}
	if h := x.head(t, x.a.id); !h.next.Valid || h.next.UUID != aDead {
		t.Errorf("tenant B's call moved tenant A's head: %+v", h)
	}
	if n := x.count(t, "SELECT count(*) FROM audit_log WHERE action = 'reseed'"); n != 0 {
		t.Errorf("a refused reseed wrote %d entries", n)
	}
	if rec := x.reseed(t, "admin-a"); rec.Code != http.StatusOK {
		t.Fatalf("control: tenant A (dead) = %d: %s", rec.Code, rec.Body)
	}

	aLive := x.lineage(t, x.a.id, "+5 minutes")
	x.lineage(t, x.b.id, "-20 minutes")
	if rec := x.reseed(t, "admin-b"); rec.Code != http.StatusOK {
		t.Errorf("tenant B (dead) while tenant A is alive = %d: %s, want 200", rec.Code, rec.Body)
	}
	if h := x.head(t, x.a.id); !h.next.Valid || h.next.UUID != aLive {
		t.Errorf("tenant B's reseed moved tenant A's head: %+v", h)
	}
	if rec := x.reseed(t, "admin-a"); rec.Code != http.StatusConflict {
		t.Errorf("control: tenant A (alive) = %d, want 409", rec.Code)
	}
	if n := x.count(t, "SELECT count(*) FROM audit_log WHERE action = 'reseed' AND tenant_id = $1", x.a.id); n != 1 {
		t.Errorf("tenant A has %d reseed entries, want its own 1", n)
	}
}

// CI-A4: with no tenant context the export's reads and the head's lease and
// reseed columns reach zero rows, and with tenant B set they reach none of
// tenant A's, as nava_audit_reader (the export's pool) and nava_app (the
// lease and the reseed). Control: with tenant A set, A's rows are reached.
func TestCIA4NoTenantContextReachesNothing(t *testing.T) {
	x := newExportEnv(t)
	x.seedActivity(t)
	ctx := t.Context()
	reader := x.rolePool(t, "nava_audit_reader")
	probes := []struct {
		pool    *pgxpool.Pool
		role, q string
	}{
		{reader, "nava_audit_reader", "SELECT tenant_id FROM audit_log WHERE tenant_id = $1 AND seq > 0"},
		{reader, "nava_audit_reader", "SELECT tenant_id FROM audit_log WHERE tenant_id = $1 AND seq IS NOT NULL ORDER BY seq DESC LIMIT 1"},
		{x.pool, "nava_app", "SELECT tenant_id FROM audit_chain_head WHERE tenant_id = $1"},
		{x.pool, "nava_app", "UPDATE audit_chain_head SET export_lease_until = now() + interval '1 hour' WHERE tenant_id = $1 RETURNING tenant_id"},
		{x.pool, "nava_app", "UPDATE audit_chain_head SET next_seal_at = now() WHERE tenant_id = $1 RETURNING tenant_id"},
	}
	reach := func(p *pgxpool.Pool, tenant uuid.UUID, q string) (int, error) {
		var n int
		run := func(tx pgx.Tx) error {
			rows, err := tx.Query(ctx, q, x.a.id)
			if err != nil {
				return err
			}
			for rows.Next() {
				n++
			}
			rows.Close()
			if err := rows.Err(); err != nil {
				return err
			}
			return errRollbackProbe
		}
		var err error
		if tenant == uuid.Nil {
			err = pgx.BeginFunc(ctx, p, run)
		} else {
			err = tenancy.InTenantTx(ctx, p, tenant, run)
		}
		if errors.Is(err, errRollbackProbe) {
			err = nil
		}
		return n, err
	}
	for _, pr := range probes {
		if n, err := reach(pr.pool, x.a.id, pr.q); err != nil || n == 0 {
			t.Fatalf("control: %s in tenant A: %s: %d rows, %v", pr.role, pr.q, n, err)
		}
		for name, tenant := range map[string]uuid.UUID{"no tenant": uuid.Nil, "tenant B": x.b.id} {
			if n, err := reach(pr.pool, tenant, pr.q); err != nil || n != 0 {
				t.Errorf("%s, %s: %s: %d of tenant A's rows, %v; want 0", pr.role, name, pr.q, n, err)
			}
		}
	}
}

var errRollbackProbe = errors.New("rollback the probe")

// a5SecSchema is a type whose token property is x-nava-secret; note is
// described by desc, and tokenExtra is spliced into token.
func a5SecSchema(desc, tokenExtra string) string {
	return fmt.Sprintf(`{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object","additionalProperties":false,`+
		`"required":["short_description","state"],"properties":{"short_description":{"type":"string"},`+
		`"state":{"type":"string","enum":["new","closed"],"default":"new"},`+
		`"note":{"type":"string","description":%q},"token":{"type":"string","x-nava-secret":true%s}}}`, desc, tokenExtra)
}

func a5Definition(desc, tokenExtra string) string {
	return fmt.Sprintf(`{"schema": %s, %s}`, a5SecSchema(desc, tokenExtra), etDefTail)
}

// a5MaxControlBytes bounds the stored size of a control value (TICK-552
// review follow-up): a value stored out of line but uncompressed can pass
// the compression check while TOAST chunking (about 2 kB) splits a canary.
const a5MaxControlBytes = 1900

// a5ControlPlain is the TICK-552 uncompressed-control check with its size
// bound: exactly one entity_definition_versions row of iso type key carries
// marker, its schema is not compressed and is under a5MaxControlBytes.
func (x *exportEnv) a5ControlPlain(ctx context.Context, key, marker string) error {
	var n int
	var compressed []string
	var size int
	if err := x.super.QueryRow(ctx, `SELECT count(*),
		coalesce(array_agg(pg_column_compression(v.schema)) FILTER (WHERE pg_column_compression(v.schema) IS NOT NULL), '{}'),
		coalesce(max(pg_column_size(v.schema)), 0)
		FROM entity_definition_versions v JOIN entity_definitions d ON d.tenant_id = v.tenant_id AND d.id = v.definition_id
		WHERE d.tenant_id = $1 AND d.key = $2 AND strpos(v.schema::text, $3) > 0`, x.a.id, key, marker).Scan(&n, &compressed, &size); err != nil {
		return fmt.Errorf("control: compression of entity_definition_versions.schema: %w", err)
	}
	switch {
	case n != 1:
		return fmt.Errorf("control: %d entity_definition_versions rows carry the marker, want 1; the scan proves nothing", n)
	case len(compressed) > 0:
		return fmt.Errorf("control: the control schema is stored compressed (%s); the scan proves nothing", strings.Join(compressed, ", "))
	case size >= a5MaxControlBytes:
		return fmt.Errorf("control: the control schema is %d bytes stored, not under %d, so TOAST may split a canary; the scan proves nothing", size, a5MaxControlBytes)
	}
	return nil
}

// CI-A5 export bytes (CTO ruling on TICK-552, TICK-447): tenant A's admin
// tries to store a record canary (create and patch at the x-nava-secret
// property, a URL with userinfo) and S8 canaries (the secret property's
// default, a URL with userinfo in a description). Each is refused, and no
// canary's bytes are in any export of tenant A: unfiltered, by domain, and
// stubs=false. Controls: a record marker and a publish marker written the
// same way are in the export bytes; the same scan finds a canary placed in
// the output on purpose; the publish control is stored uncompressed and
// under the TOAST size bound, and an incompressible padded control fails
// that bound.
//
// covers: exportAuditEntries 200
func TestCIA5ExportBytesCarryNoCanary(t *testing.T) {
	x := newExportEnv(t)
	ctx := t.Context()
	run := strings.ReplaceAll(uuid.NewString(), "-", "")
	recordMarker, publishMarker := "a5-control-r-"+run, "a5-control-s-"+run
	key := 0
	do := func(method, path, body string) (int, string) {
		key++
		rec := x.do(t, method, path, "admin-a", fmt.Sprintf("a5-%s-%d", run[:8], key), body)
		return rec.Code, rec.Body.String()
	}
	create := fmt.Sprintf(`{"key": "a5_sec", "name": "A5 sec", "number_prefix": "AFS", "definition": %s}`, a5Definition("v1", ""))
	if code, body := do(http.MethodPost, "/api/v1/entity-types", create); code != http.StatusCreated {
		t.Fatalf("control: create a5_sec = %d %s", code, body)
	}
	rec := x.mustCreateRec(t, "admin-a", "a5_sec", "a5-r1", fmt.Sprintf(`{"short_description": "a5", "note": %q}`, recordMarker))
	publish := func(def string) (int, string) {
		return do(http.MethodPost, "/api/v1/entity-types/a5_sec/versions", fmt.Sprintf(`{"definition": %s}`, def))
	}
	if code, body := publish(a5Definition(publishMarker, "")); code != http.StatusCreated {
		t.Fatalf("control: publish the marker = %d %s", code, body)
	}

	canary := func(n int) string { return fmt.Sprintf("a5-canary-%d-%s", n, run) }
	attempts := []struct {
		name, secret string
		do           func() (int, string)
	}{
		{"record create, the secret property", canary(1), func() (int, string) {
			return do(http.MethodPost, "/api/v1/records/a5_sec", fmt.Sprintf(`{"data": {"short_description": "c", "token": %q}}`, canary(1)))
		}},
		{"record patch, the secret property", canary(2), func() (int, string) {
			r := x.patch(t, "admin-a", "a5_sec", rec.ID, `W/"1"`, fmt.Sprintf(`{"data": {"token": %q}}`, canary(2)))
			return r.Code, r.Body.String()
		}},
		{"record create, a URL with userinfo", canary(3), func() (int, string) {
			return do(http.MethodPost, "/api/v1/records/a5_sec",
				fmt.Sprintf(`{"data": {"short_description": "c", "note": "https://iso:%s@example.test/"}}`, canary(3)))
		}},
		{"S8 publish, the secret property's default", canary(4), func() (int, string) {
			return publish(a5Definition("v3", fmt.Sprintf(`,"default":%q`, canary(4))))
		}},
		{"S8 publish, a URL with userinfo in a description", canary(5), func() (int, string) {
			return publish(a5Definition(fmt.Sprintf("https://iso:%s@example.test/", canary(5)), ""))
		}},
	}
	for _, a := range attempts {
		code, body := a.do()
		if code < 400 || code >= 500 {
			t.Errorf("%s = %d; want a 4xx refusal", a.name, code)
		}
		if strings.Contains(body, a.secret) {
			t.Errorf("%s: the response carries the canary", a.name)
		}
	}
	x.sealAll(t)
	if err := x.a5ControlPlain(ctx, "a5_sec", publishMarker); err != nil {
		t.Fatal(err)
	}

	// scan is the one canary scan: the attempts whose canary is in b.
	scan := func(b []byte) []string {
		var hits []string
		for _, a := range attempts {
			if bytes.Contains(b, []byte(a.secret)) {
				hits = append(hits, a.name)
			}
		}
		return hits
	}
	var all []byte
	for _, q := range []string{"", "?domain=record", "?domain=config", "?stubs=false"} {
		body := exportOK(t, x, "admin-a", q)
		all = append(append(all, body...), '\n')
		if hits := scan(body); len(hits) > 0 {
			t.Errorf("export %q: the canaries of %v are in the export bytes", q, hits)
		}
	}
	for _, m := range []string{recordMarker, publishMarker} {
		if !bytes.Contains(all, []byte(m)) {
			t.Fatalf("control: the marker %s is not in the export bytes; the scan proves nothing", m)
		}
	}
	// Negative control: the same scan finds a canary placed in the output.
	planted := append(bytes.Clone(all), `{"kind":"entry","diff":[{"op":"add","path":"/token","value":"`+canary(4)+`"}]}`+"\n"...)
	if hits := scan(planted); len(hits) != 1 || hits[0] != attempts[3].name {
		t.Fatalf("control: the scan finds %v in output with a planted canary, want only %q", hits, attempts[3].name)
	}

	// The size bound's own red control: a control padded with incompressible
	// text is stored uncompressed and out of line, and fails on the bound.
	pad := make([]byte, 1400)
	if _, err := rand.Read(pad); err != nil {
		t.Fatal(err)
	}
	padMarker := "a5-control-pad-" + run
	if code, body := publish(a5Definition(padMarker+" "+hex.EncodeToString(pad), "")); code != http.StatusCreated {
		t.Fatalf("control: publish the padded control = %d %s", code, body)
	}
	err := x.a5ControlPlain(ctx, "a5_sec", padMarker)
	if err == nil || !strings.Contains(err.Error(), "TOAST may split") {
		t.Fatalf("control: the padded control passed the plain-bytes check (%v); the size bound proves nothing", err)
	}
}

//go:build integration

// P5-13b (TICK-452): the pipeline writes ADR-004 entries through
// audit.Append, and the write path runs against a real PostgreSQL under RLS,
// as the real writer roles. P5-13c (TICK-530): the records table carries the
// commit-check trigger, so a raw write fails at COMMIT (CI-A2, CI-A17 (c)).

package pipeline_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/internal/audit"
	"github.com/Navaladi2019/TicketingSystem/internal/audit/secrets"
	"github.com/Navaladi2019/TicketingSystem/internal/metadata"
	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
	"github.com/Navaladi2019/TicketingSystem/internal/records/cascade"
	"github.com/Navaladi2019/TicketingSystem/internal/records/pipeline"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// rolePool is a pool for role on the platform database.
func rolePool(t *testing.T, e env, role string) *pgxpool.Pool {
	t.Helper()
	return appPool(t, rolePoolDSN(t, e, role))
}

// rolePoolDSN is role's connection string on the platform database.
func rolePoolDSN(t *testing.T, e env, role string) string {
	t.Helper()
	u, err := url.Parse(os.Getenv("NAVA_TEST_PG_URL"))
	if err != nil {
		t.Fatal(err)
	}
	pw, _ := testPassword(role)
	u.User = url.UserPassword(role, pw)
	u.Path = "/" + e.platform
	return u.String()
}

// isAuditMissing reports whether err is the records commit check refusing a
// COMMIT (ADR-004 decision 2).
func isAuditMissing(err error) bool {
	var pe *pgconn.PgError
	return errors.As(err, &pe) && pe.Code == "23514" && pe.ConstraintName == "records_audit_commit_check"
}

// CI-A1: a rolled-back write leaves no entry; a committed transaction leaves
// at least one entry per record it wrote and at most one per version, all
// in the transaction's txid.
func TestCIA1EntriesFollowTheTransaction(t *testing.T) {
	e := migrated(t)
	f := seedTenant(t, e)
	p := newPipeline(e, nil)
	ctx := context.Background()

	rollback := errors.New("roll back")
	var lost uuid.UUID
	err := tenancy.InTenantTx(ctx, e.app, f.id, func(tx pgx.Tx) error {
		res, err := p.Write(ctx, tx, createCmd(f, map[string]any{"title": "rolled back"}))
		if err != nil {
			return err
		}
		lost = res.RecordID
		if n := countIn(t, tx, `SELECT count(*) FROM audit_log WHERE subject_type = 'record' AND subject_id = $1`, lost); n != 1 {
			t.Errorf("inside the transaction: %d entries, want 1", n)
		}
		return rollback
	})
	if !errors.Is(err, rollback) {
		t.Fatalf("err = %v", err)
	}
	if n := count(t, e.super, `SELECT count(*) FROM audit_log WHERE subject_id = $1`, lost); n != 0 {
		t.Errorf("the rolled-back write left %d entries", n)
	}

	// One transaction: create R, update it twice, create S.
	var r, s pipeline.Result
	err = tenancy.InTenantTx(ctx, e.app, f.id, func(tx pgx.Tx) error {
		var err error
		if r, err = p.Write(ctx, tx, createCmd(f, map[string]any{"title": "r"})); err != nil {
			return err
		}
		for v := int64(1); v <= 2; v++ {
			if r, err = p.Write(ctx, tx, updateCmd(f, r.RecordID, v, map[string]any{"summary": "v"})); err != nil {
				return err
			}
		}
		s, err = p.Write(ctx, tx, createCmd(f, map[string]any{"title": "s"}))
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if n := count(t, e.super, `SELECT count(DISTINCT txid) FROM audit_log WHERE subject_id IN ($1, $2)`, r.RecordID, s.RecordID); n != 1 {
		t.Errorf("the transaction's entries carry %d txids, want 1", n)
	}
	if got := queryText(t, e, `SELECT string_agg(action || ':' || coalesce(version_before::text, '-') || '>' || version_after, ',' ORDER BY version_after)
		FROM audit_log WHERE subject_type = 'record' AND subject_id = $1`, r.RecordID); got != "create:->1,update:1>2,update:2>3" {
		t.Errorf("R's entries %s, want one per version", got)
	}
	if n := recordEntries(t, e, s.RecordID); n != 1 {
		t.Errorf("S has %d entries, want 1", n)
	}
	if n := count(t, e.super, `SELECT count(*) FROM (SELECT subject_id, version_after FROM audit_log
		WHERE tenant_id = $1 AND domain = 'record' GROUP BY 1, 2 HAVING count(*) > 1) d`, f.id); n != 0 {
		t.Errorf("%d (record, version) pairs with more than one entry", n)
	}
}

// CI-A2 (records): a raw write to records, as each writer role, with no
// entry fails at COMMIT, naming the trigger; the pipeline's write of the same
// record commits.
func TestCIA2RawRecordWriteFailsAtCommit(t *testing.T) {
	e := migrated(t)
	f := seedTenant(t, e)
	ctx := context.Background()
	rec := mustCreate(t, e, newPipeline(e, nil), f, map[string]any{"title": "printer"})
	defID := definitionID(t, e, f.id)

	for _, role := range []string{"nava_app", "nava_worker"} {
		pool := e.app
		if role != "nava_app" {
			pool = rolePool(t, e, role)
		}
		for name, stmt := range map[string]func(tx pgx.Tx) error{
			"insert": func(tx pgx.Tx) error {
				_, err := tx.Exec(ctx, `INSERT INTO records (tenant_id, id, entity_definition_id, schema_version, number, state, data)
					VALUES ($1, $2::uuid, $3, 1, 'RAW-' || $2::text, 'open', '{}')`, f.id, uuid.New(), defID)
				return err
			},
			"update": func(tx pgx.Tx) error {
				_, err := tx.Exec(ctx, `UPDATE records SET data = data || '{"summary":"raw"}', version = version + 1 WHERE id = $1`, rec.RecordID)
				return err
			},
		} {
			t.Run(role+" "+name, func(t *testing.T) {
				err := tenancy.InTenantTx(ctx, pool, f.id, stmt)
				if !isAuditMissing(err) {
					t.Fatalf("raw %s as %s: %v, want audit_missing at COMMIT", name, role, err)
				}
			})
		}
	}
	if _, _, _, v, _ := storedData(t, e, rec.RecordID); v != 1 {
		t.Errorf("version %d after the refused raw updates, want 1", v)
	}
	if n := count(t, e.super, `SELECT count(*) FROM records WHERE tenant_id = $1`, f.id); n != 1 {
		t.Errorf("%d records after the refused raw inserts, want 1", n)
	}
}

// P5-12b contract: after audit.Append fails, the caller's transaction is
// aborted; the pipeline returns the error (here audit.ErrDuplicateEntry) and
// never retries in it, and nothing commits.
func TestDuplicateEntryAbortsTheWrite(t *testing.T) {
	e := migrated(t)
	f := seedTenant(t, e)
	p := newPipeline(e, nil)
	ctx := context.Background()
	rec := mustCreate(t, e, p, f, map[string]any{"title": "printer"})
	before := count(t, e.super, `SELECT count(*) FROM audit_log WHERE tenant_id = $1`, f.id)

	var writeErr error
	err := tenancy.InTenantTx(ctx, e.app, f.id, func(tx pgx.Tx) error {
		// An entry for R at version 2 already in this transaction: the
		// pipeline's own entry for the same (subject, version_after) is the
		// R2 commit key's duplicate.
		if _, err := tx.Exec(ctx, seedEntrySQL, f.id, uuid.Must(uuid.NewV7()), rec.RecordID, 2); err != nil {
			return err
		}
		_, writeErr = p.Write(ctx, tx, updateCmd(f, rec.RecordID, 1, map[string]any{"summary": "dup"}))
		if writeErr == nil {
			return nil
		}
		// The transaction is unusable: any statement fails.
		if _, err := tx.Exec(ctx, "SELECT 1"); err == nil {
			t.Error("the transaction still runs statements after the failed Append")
		}
		return writeErr
	})
	if !errors.Is(writeErr, audit.ErrDuplicateEntry) {
		t.Fatalf("Write = %v, want audit.ErrDuplicateEntry", writeErr)
	}
	if !errors.Is(err, audit.ErrDuplicateEntry) {
		t.Fatalf("the transaction = %v, want it rolled back with the error", err)
	}
	if _, _, _, v, _ := storedData(t, e, rec.RecordID); v != 1 {
		t.Errorf("version %d, want 1: the failed write committed", v)
	}
	if n := count(t, e.super, `SELECT count(*) FROM audit_log WHERE tenant_id = $1`, f.id); n != before {
		t.Errorf("%d entries, want %d", n, before)
	}
}

// An update that moves the record to another state is a state_change entry;
// one that keeps the state is an update.
func TestStateChangeEntry(t *testing.T) {
	e := migrated(t)
	f := seedTenant(t, e)
	p := newPipeline(e, nil)
	ctx := context.Background()
	rec := mustCreate(t, e, p, f, map[string]any{"title": "printer"})
	if _, err := write(ctx, e, p, f.id, updateCmd(f, rec.RecordID, 1, map[string]any{"status": "closed"})); err != nil {
		t.Fatal(err)
	}
	got := readEntry(t, e, f.id, string(audit.ActionStateChange), rec.RecordID)
	if got.Diff != `[{"op":"replace","path":"/status","value":"closed"}]` || *got.VersionBefore != 1 || *got.VersionAfter != 2 {
		t.Errorf("state_change entry %+v", got.deref())
	}
	if _, err := write(ctx, e, p, f.id, updateCmd(f, rec.RecordID, 2, map[string]any{"summary": "same state"})); err != nil {
		t.Fatal(err)
	}
	if got := readEntry(t, e, f.id, string(audit.ActionUpdate), rec.RecordID); *got.VersionAfter != 3 {
		t.Errorf("update entry %+v", got.deref())
	}
}

// TICK-451 (binding): the diff is built from the jsonb values PostgreSQL
// stored, so a number keeps jsonb's literal, not the request's.
func TestDiffIsBuiltFromStoredJSONB(t *testing.T) {
	e := migrated(t)
	def := definition(schemaV1)
	def.Key, def.NumberPrefix = "metric", "MET"
	def.Schema = json.RawMessage(`{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object","additionalProperties":false,
		"required":["title"],"properties":{"title":{"type":"string"},"status":{"type":"string","enum":["open","closed"]},
		"requester":{"type":"string","format":"uuid"},"n":{"type":"number"},"m":{"type":"object"}}}`)
	f := seedTenantWith(t, e, def)
	p := newPipeline(e, nil)
	// An explicit requester: this test is about jsonb's literals, not the default (A4).
	cmd := createCmd(f, map[string]any{"title": "t", "requester": f.principal.PrincipalID.String(), "n": json.Number("1e2"), "m": map[string]any{"z": 1, "a": json.Number("2.50")}})
	cmd.EntityKey = "metric"
	res, err := write(context.Background(), e, p, f.id, cmd)
	if err != nil {
		t.Fatal(err)
	}
	got := readEntry(t, e, f.id, "create", res.RecordID)
	if want := `[{"op":"add","path":"","value":{"m":{"a":2.50,"z":1},"n":100,"requester":"` + f.principal.PrincipalID.String() + `","status":"open","title":"t"}}]`; got.Diff != want {
		t.Errorf("diff %s, want %s (jsonb's literals)", got.Diff, want)
	}
}

// ADR-004 R6: a value at an x-nava-secret or writeOnly property, or a URL
// with userinfo, is refused before anything is written, as *ErrValidation
// with the code audit_secret_in_diff, naming the pointer only.
func TestSecretInRecordDiffIsRefused(t *testing.T) {
	e := migrated(t)
	def := definition(schemaV1)
	def.Key, def.NumberPrefix = "account", "ACC"
	def.Schema = json.RawMessage(`{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object","additionalProperties":false,
		"required":["title"],"properties":{"title":{"type":"string"},"status":{"type":"string","enum":["open","closed"]},
		"requester":{"type":"string","format":"uuid"},"pin":{"type":"string","writeOnly":true},"note":{"type":"string"}}}`)
	f := seedTenantWith(t, e, def)
	p := newPipeline(e, nil)
	for name, tc := range map[string]struct {
		data map[string]any
		path string
	}{
		"writeOnly property": {map[string]any{"title": "t", "pin": "1234"}, "/pin"},
		"url with userinfo":  {map[string]any{"title": "t", "note": "https://user:hunter2@example.com"}, "/note"},
	} {
		t.Run(name, func(t *testing.T) {
			cmd := createCmd(f, tc.data)
			cmd.EntityKey = "account"
			_, err := write(context.Background(), e, p, f.id, cmd)
			var ve *pipeline.ErrValidation
			if !errors.As(err, &ve) || len(ve.Fields) != 1 || ve.Fields[0] != (metadata.FieldError{Path: tc.path, Code: secrets.Code}) {
				t.Fatalf("err = %v, want *ErrValidation %s %s", err, tc.path, secrets.Code)
			}
		})
	}
	if n := count(t, e.super, `SELECT count(*) FROM records WHERE tenant_id = $1`, f.id); n != 0 {
		t.Errorf("%d records after the refused writes", n)
	}
	if n := count(t, e.super, `SELECT count(*) FROM audit_log WHERE tenant_id = $1 AND domain = 'record'`, f.id); n != 0 {
		t.Errorf("%d record entries after the refused writes", n)
	}
}

// ADR-004 R4: an AI write's entry carries its source_ref.
func TestAIWriteCarriesSourceRef(t *testing.T) {
	e := migrated(t)
	f := seedTenant(t, e)
	ref := uuid.NewString()
	cmd := createCmd(f, map[string]any{"title": "suggested"})
	// An AI write is authorized as its principal (D-3-26): f.principal, whom
	// seedTenant's test ACL (acltest.AllowUsers) lets write. A principal-less
	// AI write is refused (TestWriteScopeIsEnforcedBeforeAndAfter and the
	// unit tests cover that).
	cmd.Source, cmd.SourceRef = pipeline.SourceAI, &ref
	res, err := write(context.Background(), e, newPipeline(e, nil), f.id, cmd)
	if err != nil {
		t.Fatal(err)
	}
	want := "ai " + ref + " " + f.principal.PrincipalID.String()
	if got := queryText(t, e, `SELECT source || ' ' || source_ref || ' ' || actor_principal_id::text FROM audit_log WHERE subject_id = $1`, res.RecordID); got != want {
		t.Errorf("entry %q, want %q", got, want)
	}
}

// ADR-004-E1 CI-A17 (a): one write that suppresses D1 and D2 on R writes two
// cascade_cycle entries and one update entry, and commits. (c): a
// transaction that writes R with only a cascade_cycle entry fails at COMMIT.
func TestCIA17CycleEntries(t *testing.T) {
	e := migrated(t)
	f := seedTenant(t, e)
	ctx := context.Background()
	d1, d2 := workflowDefinition(t, e, f), workflowDefinition(t, e, f)
	sink := &startSink{defs: []uuid.UUID{d1, d2}}
	p := newPipeline(e, sink)
	r, origin := createWithStart(t, e, p, sink, f)

	if _, err := write(ctx, e, p, f.id, workflowCmd(f, r.RecordID, 1, outbox.Causation{OriginID: origin, Depth: 1}, "s1")); err != nil {
		t.Fatalf("(a) %v", err)
	}
	if n := entries(t, e, f.id, cycleAction); n != 2 {
		t.Errorf("(a) %d cascade_cycle entries, want 2", n)
	}
	if got := queryText(t, e, `SELECT string_agg(subject_id::text, ',' ORDER BY subject_id) FROM audit_log WHERE tenant_id = $1 AND action = 'cascade_cycle'`,
		f.id); got != sortedJoin(audit.CascadeKey(f.id, origin, r.RecordID, d1), audit.CascadeKey(f.id, origin, r.RecordID, d2)) {
		t.Errorf("(a) cycle subjects %s", got)
	}
	if n := count(t, e.super, `SELECT count(*) FROM audit_log WHERE subject_id = $1 AND action = 'update'`, r.RecordID); n != 1 {
		t.Errorf("(a) %d update entries, want 1", n)
	}

	// (c) A raw update of R and only a cycle entry: refused at COMMIT by the
	// records commit-check trigger (P5-13c).
	err := tenancy.InTenantTx(ctx, e.app, f.id, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `UPDATE records SET version = version + 1 WHERE id = $1`, r.RecordID); err != nil {
			return err
		}
		c, claimed, err := cascade.ClaimStart(ctx, tx, cascade.Key{RecordID: r.RecordID, DefinitionID: d1})
		if err != nil || claimed {
			return errors.Join(err, errors.New("D1 was claimed again"))
		}
		return pipeline.AuditSuppressed(ctx, tx, c, pipeline.Event{Source: pipeline.SourceWorkflow})
	})
	if !isAuditMissing(err) {
		t.Fatalf("(c) %v, want audit_missing at COMMIT", err)
	}
}

func sortedJoin(a, b uuid.UUID) string {
	if a.String() > b.String() {
		a, b = b, a
	}
	return a.String() + "," + b.String()
}

func countIn(t *testing.T, tx pgx.Tx, sql string, args ...any) int {
	t.Helper()
	var n int
	if err := tx.QueryRow(context.Background(), sql, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func queryText(t *testing.T, e env, sql string, args ...any) string {
	t.Helper()
	var s *string
	if err := e.super.QueryRow(context.Background(), sql, args...).Scan(&s); err != nil {
		t.Fatal(err)
	}
	if s == nil {
		return ""
	}
	return *s
}

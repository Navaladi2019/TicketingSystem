//go:build integration

package pipeline_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
	"github.com/Navaladi2019/TicketingSystem/internal/records/pipeline"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// P10-02 (TICK-917, ADR-009 D3–D6): OpMaterialize through the real pipeline,
// as nava_app under RLS.

// materializeAt is f's OpMaterialize of record id under causation c.
func materializeAt(f tenantFixture, id uuid.UUID, c outbox.Causation) pipeline.Command {
	return pipeline.Command{Op: pipeline.OpMaterialize, EntityKey: entityKey, RecordID: id, Actor: f.system(),
		Source: pipeline.SourceSystem, ActorSystem: pipeline.ActorSchemaBackfill, Causation: c}
}

// updatedAt is the record's stored updated_at.
func updatedAt(t *testing.T, e env, id uuid.UUID) time.Time {
	t.Helper()
	var at time.Time
	if err := e.super.QueryRow(context.Background(), "SELECT updated_at FROM records WHERE id = $1", id).Scan(&at); err != nil {
		t.Fatal(err)
	}
	return at
}

// CI-2 (pipeline half), D3–D6: a record missing a defaulted key gets it, with
// version + 1, schema_version the current one, the same updated_at, exactly
// one audit entry (action update, source system, actor_system
// schema-backfill, the given causation, the stored bytes before and after)
// and no sink call and no outbox row. A second run writes nothing.
func TestMaterializeWritesMissingDefaultsOnce(t *testing.T) {
	e := migrated(t)
	f := seedTenant(t, e)
	sink := &countingSink{}
	p := newPipeline(e, sink)
	ctx := context.Background()

	res := mustCreate(t, e, p, f, map[string]any{"title": "printer"})
	publishV2(t, e, f) // adds channel, default "email"
	before := updatedAt(t, e, res.RecordID)
	sinkCalls := sink.calls()
	outboxRows := count(t, e.super, "SELECT count(*) FROM outbox WHERE tenant_id = $1", f.id)
	origin := uuid.Must(uuid.NewV7())

	got, err := write(ctx, e, p, f.id, materializeAt(f, res.RecordID, outbox.Causation{OriginID: origin, Depth: 2}))
	if err != nil {
		t.Fatalf("materialize: %v", err)
	}
	if !got.Written || got.Version != 2 || got.SchemaVersion != 2 {
		t.Errorf("result = %+v, want Written, version 2, schema version 2", got)
	}
	data, _, _, version, schemaVersion := storedData(t, e, res.RecordID)
	if data["channel"] != "email" || data["title"] != "printer" || version != 2 || schemaVersion != 2 {
		t.Errorf("stored = %v, version %d, schema version %d; want channel email, version 2, schema version 2", data, version, schemaVersion)
	}
	if at := updatedAt(t, e, res.RecordID); !at.Equal(before) {
		t.Errorf("updated_at = %v, want it kept at %v (D5)", at, before)
	}
	if n := recordEntries(t, e, res.RecordID); n != 2 {
		t.Errorf("audit entries = %d, want the create's and one for the materialize", n)
	}
	var action, source, system, diff string
	var causation uuid.UUID
	var depth int
	var vb, va int64
	if err := e.super.QueryRow(ctx, `SELECT action, source, actor_system, causation_id, causation_depth, version_before, version_after, diff::text
		FROM audit_log WHERE subject_type = 'record' AND subject_id = $1 AND version_after = 2`, res.RecordID).
		Scan(&action, &source, &system, &causation, &depth, &vb, &va, &diff); err != nil {
		t.Fatalf("materialize entry: %v", err)
	}
	if action != "update" || source != "system" || system != pipeline.ActorSchemaBackfill || causation != origin || depth != 2 || vb != 1 || va != 2 {
		t.Errorf("entry = %s %s %s causation %s/%d versions %d->%d; want update system schema-backfill %s/2 1->2",
			action, source, system, causation, depth, vb, va, origin)
	}
	// The diff is of the stored bytes (D11 last sentence): one add of the
	// default, nothing else.
	if want := `[{"op":"add","path":"/channel","value":"email"}]`; diff != want {
		t.Errorf("diff = %s, want %s", diff, want)
	}
	if n := sink.calls(); n != sinkCalls {
		t.Errorf("sink calls = %d, want %d: a materialize tells no sink (D6)", n, sinkCalls)
	}
	if n := count(t, e.super, "SELECT count(*) FROM outbox WHERE tenant_id = $1", f.id); n != outboxRows {
		t.Errorf("outbox rows = %d, want %d (D6)", n, outboxRows)
	}

	// Idempotent by the D3 compare: the record holds every default now.
	again, err := write(ctx, e, p, f.id, materializeAt(f, res.RecordID, outbox.Causation{OriginID: origin, Depth: 2}))
	if err != nil || again.Written || again.Version != 2 {
		t.Errorf("second materialize = %+v, %v; want Written false at version 2", again, err)
	}
	if n := recordEntries(t, e, res.RecordID); n != 2 {
		t.Errorf("audit entries after the second run = %d, want 2", n)
	}
	if _, _, _, version, _ := storedData(t, e, res.RecordID); version != 2 {
		t.Errorf("version after the second run = %d, want 2", version)
	}
}

// D3: a record that already holds every default of the current version is
// not written, bumped or audited, even when its schema_version is older; a
// rollout job's write (no causation) mints an origin at depth 0.
func TestMaterializeRecordHoldingTheKeyWritesNothing(t *testing.T) {
	e := migrated(t)
	f := seedTenant(t, e)
	p := newPipeline(e, nil)
	ctx := context.Background()

	held := mustCreate(t, e, p, f, map[string]any{"title": "a", "summary": "s"})
	publishV2(t, e, f)
	// The record is given channel by a v2 update, so it holds every default.
	if _, err := write(ctx, e, p, f.id, updateCmd(f, held.RecordID, 1, map[string]any{"channel": "phone"})); err != nil {
		t.Fatal(err)
	}
	entriesBefore := recordEntries(t, e, held.RecordID)
	got, err := write(ctx, e, p, f.id, materializeAt(f, held.RecordID, outbox.Causation{}))
	if err != nil || got.Written || got.Version != 2 {
		t.Errorf("materialize of a record holding the key = %+v, %v; want Written false at version 2", got, err)
	}
	if n := recordEntries(t, e, held.RecordID); n != entriesBefore {
		t.Errorf("audit entries = %d, want %d", n, entriesBefore)
	}

	missing := mustCreate(t, e, p, f, map[string]any{"title": "b"})
	// A v2 create applies channel's default itself, so strip it to stand for
	// a record written before the default existed.
	stripChannel(t, e, f, missing.RecordID)
	if got, err := write(ctx, e, p, f.id, materializeAt(f, missing.RecordID, outbox.Causation{})); err != nil || !got.Written {
		t.Fatalf("rollout materialize = %+v, %v", got, err)
	}
	if n := count(t, e.super, `SELECT count(*) FROM audit_log WHERE subject_id = $1 AND version_after = 3
		AND causation_depth = 0 AND causation_id IS NOT NULL AND actor_system = 'schema-backfill'`, missing.RecordID); n != 1 {
		t.Errorf("rollout materialize entries at depth 0 with a minted origin = %d, want 1", n)
	}
}

// stripChannel removes channel from the stored record, as the superuser, with
// the audit entry the records commit check wants (version + 1).
func stripChannel(t *testing.T, e env, f tenantFixture, id uuid.UUID) {
	t.Helper()
	corrupt(t, e, f, id, `UPDATE records SET data = data - 'channel', version = version + 1 WHERE id = $1 RETURNING version`)
}

// corrupt runs sql (an UPDATE of record id that returns its new version) as
// the superuser, in one transaction with the audit entry of that version.
func corrupt(t *testing.T, e env, f tenantFixture, id uuid.UUID, sql string) {
	t.Helper()
	ctx := context.Background()
	err := pgx.BeginFunc(ctx, e.super, func(tx pgx.Tx) error {
		var v int64
		if err := tx.QueryRow(ctx, sql, id).Scan(&v); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `INSERT INTO audit_log (tenant_id, id, domain, action, subject_type, subject_id, version_before, version_after, actor_system, source)
			VALUES ($1, $2, 'record', 'update', 'record', $3, $4, $5, 'system', 'system')`, f.id, uuid.Must(uuid.NewV7()), id, v-1, v)
		return err
	})
	if err != nil {
		t.Fatalf("corrupt the fixture: %v", err)
	}
}

// CI-3 (4) and D7: a refusal is typed with a stable code and writes nothing;
// an unknown record and another tenant's record are not found.
func TestMaterializeRefusalsWriteNothing(t *testing.T) {
	e := migrated(t)
	f := seedTenant(t, e)
	other := seedTenant(t, e)
	p := newPipeline(e, nil)
	ctx := context.Background()

	bad := mustCreate(t, e, p, f, map[string]any{"title": "x"})
	theirs := mustCreate(t, e, p, other, map[string]any{"title": "y"})
	publishV2(t, e, f)
	publishV2(t, e, other)
	// The stored record breaks the current schema (an empty title), as a row
	// written before a constraint tightened would.
	corrupt(t, e, f, bad.RecordID, `UPDATE records SET data = jsonb_set(data, '{title}', '""'), version = version + 1 WHERE id = $1 RETURNING version`)
	entriesBefore := recordEntries(t, e, bad.RecordID)

	_, err := write(ctx, e, p, f.id, materializeAt(f, bad.RecordID, outbox.Causation{}))
	var ve *pipeline.ErrValidation
	if !errors.As(err, &ve) || pipeline.ErrorCode(err) != pipeline.CodeValidationFailed {
		t.Errorf("materialize of an invalid record: err = %v (code %q), want *ErrValidation, validation_failed", err, pipeline.ErrorCode(err))
	}
	if data, _, _, version, _ := storedData(t, e, bad.RecordID); version != 2 || data["channel"] != nil {
		t.Errorf("refused record = %v at version %d, want it unchanged at 2", data, version)
	}
	if n := recordEntries(t, e, bad.RecordID); n != entriesBefore {
		t.Errorf("audit entries after the refusal = %d, want %d", n, entriesBefore)
	}

	for name, id := range map[string]uuid.UUID{"an unknown record": uuid.New(), "another tenant's record": theirs.RecordID} {
		_, err := write(ctx, e, p, f.id, materializeAt(f, id, outbox.Causation{}))
		if !errors.Is(err, pipeline.ErrNotFound) || pipeline.ErrorCode(err) != pipeline.CodeNotFound {
			t.Errorf("%s: err = %v, want ErrNotFound", name, err)
		}
	}
	if _, _, _, version, _ := storedData(t, e, theirs.RecordID); version != 1 {
		t.Errorf("the other tenant's record is at version %d, want 1", version)
	}
}

// D7: a live write that lands between the materialize's read and its write is
// a version conflict, never a refusal: the backfill leaves the id pending.
func TestMaterializeLosesToALiveWrite(t *testing.T) {
	e := migrated(t)
	f := seedTenant(t, e)
	p := newPipeline(e, nil)
	ctx := context.Background()

	res := mustCreate(t, e, p, f, map[string]any{"title": "x"})
	publishV2(t, e, f)
	// The live write holds the row until release; the materialize, in its
	// own transaction, reads the old version and then waits on the row lock
	// at its UPDATE.
	updated, release, liveDone := make(chan error, 1), make(chan struct{}), make(chan error, 1)
	go func() {
		liveDone <- tenancy.InTenantTx(ctx, e.app, f.id, func(tx pgx.Tx) error {
			_, err := p.Write(ctx, tx, updateCmd(f, res.RecordID, 1, map[string]any{"summary": "live"}))
			updated <- err
			if err != nil {
				return err
			}
			<-release
			return nil
		})
	}()
	if err := <-updated; err != nil {
		t.Fatalf("live update: %v", err)
	}
	done := make(chan error, 1)
	go func() {
		_, err := write(ctx, e, p, f.id, materializeAt(f, res.RecordID, outbox.Causation{}))
		done <- err
	}()
	waitForLockWait(t, e)
	close(release)
	if err := <-liveDone; err != nil {
		t.Fatalf("live commit: %v", err)
	}
	err := <-done
	if !errors.Is(err, pipeline.ErrVersionConflict) || pipeline.ErrorCode(err) != pipeline.CodeVersionConflict {
		t.Errorf("materialize behind a live write: err = %v, want ErrVersionConflict", err)
	}
	if data, _, _, version, _ := storedData(t, e, res.RecordID); version != 2 || data["summary"] != "live" {
		t.Errorf("record = %v at version %d, want the live write's at 2", data, version)
	}
}

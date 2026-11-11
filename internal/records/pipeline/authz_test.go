package pipeline_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/records/pipeline"
)

// A1: only rule, workflow, function and system skip the authorizer. A user,
// an AI (no privilege through the AI path) and an empty or unknown source
// are authorized as their principal.
func TestOnlyTheSystemSourcesSkipTheAuthorizer(t *testing.T) {
	for s, skips := range map[pipeline.Source]bool{
		pipeline.SourceRule:     true,
		pipeline.SourceWorkflow: true,
		pipeline.SourceFunction: true,
		pipeline.SourceSystem:   true,
		pipeline.SourceUser:     false,
		pipeline.SourceAI:       false,
		"":                      false,
		"bogus":                 false,
		"System":                false,
	} {
		if got := pipeline.SkipsAuthorizer(s); got != skips {
			t.Errorf("source %q skips the authorizer = %v, want %v", s, got, skips)
		}
	}
}

// An update of a record the caller cannot read is ErrNotFound, decided
// before the version check, so a stale version never tells the caller the
// record exists (TICK-358 carry), and nothing is written.
func TestUpdateOfAnUnreadableRecordIsNotFound(t *testing.T) {
	f := newFixture()
	var calls []authzCall
	p := unitPipeline(fakeAuthz{readErr: authz.ErrNotVisible, calls: &calls}, fakeDefs{cur: assetDef(t), curVersion: 1})
	rec := storedRecord{tenant: f.tenant, id: uuid.New(), def: f.defID, key: "asset", version: 3, schemaVersion: 1,
		state: "active", data: `{"name":"x"}`}
	for _, expected := range []int64{3, 1} { // current, then stale
		tx := f.tx(t, stmt{match: sqlGet, scan: rec.scan})
		cmd := f.update(rec.id, map[string]any{"name": "y"})
		cmd.ExpectedVersion = expected
		if _, err := p.Write(t.Context(), tx, cmd); !errors.Is(err, pipeline.ErrNotFound) || errors.Is(err, pipeline.ErrVersionConflict) {
			t.Fatalf("expected version %d: %v, want ErrNotFound", expected, err)
		}
		if tx.ranAny(sqlUpdate) || tx.ranAny(sqlAudit) {
			t.Fatalf("expected version %d: a statement wrote after the refusal", expected)
		}
	}
	for _, c := range calls {
		if c.op != authz.OpRead {
			t.Fatalf("the authorizer was asked to %s an unreadable record", c.op)
		}
	}
}

// The update's write check sees the record before and after the patch, with
// the caller's shares on both; a create's sees no before and no shares.
func TestWriteChecksTheRowBeforeAndAfter(t *testing.T) {
	f := newFixture()
	var calls []authzCall
	p := unitPipeline(fakeAuthz{calls: &calls}, fakeDefs{cur: assetDef(t), curVersion: 1})
	share := uuid.New()
	rec := storedRecord{tenant: f.tenant, id: uuid.New(), def: f.defID, key: "asset", version: 1, schemaVersion: 1,
		state: "active", data: `{"name":"x"}`}
	tx := f.tx(t, stmt{match: sqlGet, scan: rec.scan},
		stmt{match: sqlShares, scan: func(dest ...any) error { *dest[0].(*[]uuid.UUID) = []uuid.UUID{share}; return nil }},
		stmt{match: sqlPayload, err: errBoom}) // stop the write after the authorizer
	_, _ = p.Write(t.Context(), tx, f.update(rec.id, map[string]any{"status": "retired"}))
	if len(calls) != 3 || calls[0].op != authz.OpRead || calls[1].op != "fields" || calls[2].op != authz.OpWrite {
		t.Fatalf("authorizer calls = %+v, want read, the field check, then write", calls)
	}
	w := calls[2]
	if w.before == nil || w.before.State != "active" || w.after.State != "retired" || w.before.ID != rec.id || w.after.ID != rec.id {
		t.Fatalf("write check before %+v after %+v, want active -> retired of the record", w.before, w.after)
	}
	if len(w.before.SharedWith) != 1 || w.before.SharedWith[0] != share || len(w.after.SharedWith) != 1 || w.after.SharedWith[0] != share {
		t.Fatalf("shares before %v after %v, want the caller's share on both", w.before.SharedWith, w.after.SharedWith)
	}

	calls = nil
	_, _ = p.Write(t.Context(), f.tx(t), f.create(map[string]any{"name": "x"}))
	if len(calls) != 2 || calls[0].op != "fields" || calls[1].op != authz.OpWrite || calls[1].before != nil || calls[1].after == nil || calls[1].after.SharedWith != nil {
		t.Fatalf("create's authorizer calls = %+v, want the field check, then one write with no before and no shares", calls)
	}
}

// A system source skips the authorizer entirely; an AI write does not.
func TestSystemSourcesSkipTheAuthorizerAndAIDoesNot(t *testing.T) {
	f := newFixture()
	p := unitPipeline(fakeAuthz{err: authz.ErrForbidden, readErr: authz.ErrNotVisible}, fakeDefs{cur: assetDef(t), curVersion: 1})

	ai := f.create(map[string]any{"name": "x"})
	ref := uuid.New().String()
	ai.Source, ai.SourceRef = pipeline.SourceAI, &ref
	if _, err := p.Write(t.Context(), f.tx(t), ai); !errors.Is(err, authz.ErrForbidden) {
		t.Fatalf("AI write: %v, want authz.ErrForbidden", err)
	}

	sys := f.create(map[string]any{"name": "x"})
	sys.Source, sys.Actor.PrincipalID = pipeline.SourceSystem, uuid.Nil
	_, err := p.Write(t.Context(), f.tx(t), sys)
	if errors.Is(err, authz.ErrForbidden) || !errors.Is(err, errBoom) {
		t.Fatalf("system write: %v, want it past the authorizer (to the number)", err)
	}
}

// The mask write check (P3-33) sees the caller's own data: on a create the
// sent payload, without the initial state or any default the pipeline adds;
// on an update the stored payload and that payload with the patch merged in.
func TestFieldCheckSeesOnlyTheCallersWrite(t *testing.T) {
	f := newFixture()
	var calls []authzCall
	p := unitPipeline(fakeAuthz{calls: &calls}, fakeDefs{cur: assetDef(t), curVersion: 1})
	_, _ = p.Write(t.Context(), f.tx(t), f.create(map[string]any{"name": "x"}))
	fc := fieldsCall(t, calls)
	if fc.beforeData != nil || len(fc.afterData) != 1 || fc.afterData["name"] != "x" {
		t.Fatalf("create field check before %v after %v, want nil and only the sent name", fc.beforeData, fc.afterData)
	}

	calls = nil
	rec := storedRecord{tenant: f.tenant, id: uuid.New(), def: f.defID, key: "asset", version: 1, schemaVersion: 1,
		state: "active", data: `{"name":"x"}`}
	tx := f.tx(t, stmt{match: sqlGet, scan: rec.scan}, stmt{match: sqlPayload, err: errBoom})
	_, _ = p.Write(t.Context(), tx, f.update(rec.id, map[string]any{"tag": 3}))
	fc = fieldsCall(t, calls)
	if got := authz.WrittenFields(fc.beforeData, fc.afterData); len(got) != 1 || got[0] != "tag" {
		t.Fatalf("update field check before %v after %v: written %v, want [tag]", fc.beforeData, fc.afterData, got)
	}
}

// A field the caller may not write refuses the write before any statement
// that writes: no number, no record, no audit entry.
func TestFieldNotWritableWritesNothing(t *testing.T) {
	f := newFixture()
	refusal := &authz.ErrFieldNotWritable{Fields: []string{"tag"}}
	p, numbers := numberedPipeline(t, fakeAuthz{fieldsErr: refusal}, fakeDefs{cur: assetDef(t), curVersion: 1}, &numberRunner{err: errBoom})
	tx := f.tx(t)
	_, err := p.Write(t.Context(), tx, f.create(map[string]any{"name": "x", "tag": 1}))
	var nw *authz.ErrFieldNotWritable
	if !errors.As(err, &nw) || nw.Fields[0] != "tag" {
		t.Fatalf("create: %v, want *authz.ErrFieldNotWritable for tag", err)
	}
	for _, s := range []string{sqlLock, sqlInsert, sqlAudit} {
		if tx.ranAny(s) {
			t.Fatalf("a refused write ran %q", s)
		}
	}
	if numbers.taken() {
		t.Fatal("a refused write took a number")
	}

	sys := f.create(map[string]any{"name": "x", "tag": 1})
	sys.Source, sys.Actor.PrincipalID = pipeline.SourceSystem, uuid.Nil
	if _, err := p.Write(t.Context(), f.tx(t), sys); errors.As(err, &nw) || !errors.Is(err, errBoom) {
		t.Fatalf("system write: %v, want it past the field check (to the number)", err)
	}
}

func fieldsCall(t *testing.T, calls []authzCall) authzCall {
	t.Helper()
	for _, c := range calls {
		if c.op == "fields" {
			return c
		}
	}
	t.Fatalf("no field check in %+v", calls)
	return authzCall{}
}

// TICK-633 B1: the field check comes before validation and the state model,
// so a refused field never yields record_invalid or illegal_transition: the
// patch below is both invalid and an illegal transition, and the answer is
// still the field refusal.
func TestFieldCheckRunsBeforeValidationAndTheStateModel(t *testing.T) {
	f := newFixture()
	refusal := &authz.ErrFieldNotWritable{Fields: []string{"status"}}
	p := unitPipeline(fakeAuthz{fieldsErr: refusal}, fakeDefs{cur: assetDef(t), curVersion: 1})
	rec := storedRecord{tenant: f.tenant, id: uuid.New(), def: f.defID, key: "asset", version: 1, schemaVersion: 1,
		state: "retired", data: `{"name":"x"}`}
	tx := f.tx(t, stmt{match: sqlGet, scan: rec.scan})
	_, err := p.Write(t.Context(), tx, f.update(rec.id, map[string]any{"status": "active", "tag": "not a number"}))
	var nw *authz.ErrFieldNotWritable
	if !errors.As(err, &nw) {
		t.Fatalf("update: %v, want *authz.ErrFieldNotWritable before validation and the transition check", err)
	}
	for _, s := range []string{sqlLock, sqlUpdate, sqlAudit} {
		if tx.ranAny(s) {
			t.Fatalf("a refused update ran %q", s)
		}
	}
	_, err = p.Write(t.Context(), f.tx(t), f.create(map[string]any{"status": "retired", "tag": "x"}))
	if !errors.As(err, &nw) {
		t.Fatalf("create: %v, want *authz.ErrFieldNotWritable before validation and the initial-state check", err)
	}
}

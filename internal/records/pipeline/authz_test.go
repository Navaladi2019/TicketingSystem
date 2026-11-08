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
	if len(calls) != 2 || calls[0].op != authz.OpRead || calls[1].op != authz.OpWrite {
		t.Fatalf("authorizer calls = %+v, want read then write", calls)
	}
	w := calls[1]
	if w.before == nil || w.before.State != "active" || w.after.State != "retired" || w.before.ID != rec.id || w.after.ID != rec.id {
		t.Fatalf("write check before %+v after %+v, want active -> retired of the record", w.before, w.after)
	}
	if len(w.before.SharedWith) != 1 || w.before.SharedWith[0] != share || len(w.after.SharedWith) != 1 || w.after.SharedWith[0] != share {
		t.Fatalf("shares before %v after %v, want the caller's share on both", w.before.SharedWith, w.after.SharedWith)
	}

	calls = nil
	_, _ = p.Write(t.Context(), f.tx(t, stmt{match: sqlNumber, err: errBoom}), f.create(map[string]any{"name": "x"}))
	if len(calls) == 0 || calls[0].op != authz.OpWrite || calls[0].before != nil || calls[0].after == nil || calls[0].after.SharedWith != nil {
		t.Fatalf("create's authorizer calls = %+v, want one write with no before and no shares", calls)
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
	_, err := p.Write(t.Context(), f.tx(t, stmt{match: sqlNumber, err: errBoom}), sys)
	if errors.Is(err, authz.ErrForbidden) || !errors.Is(err, errBoom) {
		t.Fatalf("system write: %v, want it past the authorizer (to the number)", err)
	}
}

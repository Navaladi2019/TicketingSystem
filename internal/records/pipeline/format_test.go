package pipeline_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Navaladi2019/TicketingSystem/internal/audit"

	"github.com/Navaladi2019/TicketingSystem/internal/records/numbering"
	"github.com/Navaladi2019/TicketingSystem/internal/records/pipeline"
)

// TICK-674 (ADR-005 §3.1, §3.5, A2, A7): a create numbers with the format
// snapshotted on its definition version, refuses a full refuse format, and
// maps a records number-index violation to a defect, never a retry.

// linked is template compiled with overflow as the type's epoch 1.
func linked(t *testing.T, template string, overflow numbering.Overflow) *numbering.Compiled {
	t.Helper()
	c, err := numbering.Compile(template, overflow)
	if err != nil {
		t.Fatal(err)
	}
	c.Epoch = 1
	return &c
}

const formatPayload = `{"name": "x", "status": "active"}`

// The number carries the date part of the allocation's scope key, and the
// record is created at the allocation's instant.
func TestCreateRendersTheLinkedFormatFromTheScopeKey(t *testing.T) {
	f := newFixture()
	format := linked(t, "{prefix}-{yyyy}{MM}{seq:4}", numbering.Refuse)
	r := &numberRunner{next: 7}
	p, _ := numberedPipeline(t, fakeAuthz{}, fakeDefs{cur: assetDef(t), curVersion: 1, format: format}, r)
	tx := f.tx(t, stmt{match: sqlPreview, scan: scanString(formatPayload)},
		stmt{match: sqlInsert, scan: scanInserted("AST-2026110007", formatPayload)}, stmt{match: sqlAudit})
	res, err := p.Write(t.Context(), tx, f.create(map[string]any{"name": "x"}))
	if err != nil {
		t.Fatal(err)
	}
	if res.Number != "AST-2026110007" {
		t.Errorf("number = %q, want AST-2026110007", res.Number)
	}
	if len(r.calls) != 1 || r.calls[0].ScopeKey != "e1|d:202611" || r.calls[0].Epoch != 1 {
		t.Errorf("allocations = %+v, want one on e1|d:202611", r.calls)
	}
}

// A refuse format whose counter outgrew its width fails the create with
// ErrNumberSpaceExhausted naming the type and scope, before any insert.
func TestCreateRefusesAnExhaustedFormat(t *testing.T) {
	f := newFixture()
	format := linked(t, "{prefix}-{yyyy}{MM}{seq:2}", numbering.Refuse)
	p, _ := numberedPipeline(t, fakeAuthz{}, fakeDefs{cur: assetDef(t), curVersion: 1, format: format}, &numberRunner{next: 100})
	tx := f.tx(t, stmt{match: sqlPreview, scan: scanString(formatPayload)})
	_, err := p.Write(t.Context(), tx, f.create(map[string]any{"name": "x"}))
	var ex *pipeline.ErrNumberSpaceExhausted
	if !errors.As(err, &ex) || ex.EntityKey != "asset" || ex.ScopeKey != "e1|d:202611" || ex.Width != 2 {
		t.Fatalf("err = %v, want ErrNumberSpaceExhausted for asset in e1|d:202611", err)
	}
	if !errors.Is(err, numbering.ErrNumberSpaceExhausted) || !strings.Contains(err.Error(), `"e1|d:202611"`) {
		t.Errorf("err = %v, want it to match numbering.ErrNumberSpaceExhausted and name the scope", err)
	}
	if tx.ranAny(sqlInsert) || tx.ranAny(sqlAudit) {
		t.Error("the create inserted with no number left")
	}
	// ADR-004-E9 R2b: the refusal carries its entry, and Write appended
	// nothing (the caller does, after the create's transaction).
	if ex.Epoch != 1 || ex.Counter != 100 {
		t.Errorf("epoch %d, counter %d; want 1 and 100", ex.Epoch, ex.Counter)
	}
	wantExhaustedEntry(t, ex.Entry, f.defID, f.actor.PrincipalID, `[{"op":"add","path":"/number_space","value":{"counter":100,"epoch":1,"scope_key":"e1|d:202611","width":2}}]`)
}

// wantExhaustedEntry checks a refusal's R2b entry (ADR-004-E9): subject the
// entity type, no versions, the command's actor and a depth-0 origin, no
// content hash, and the one /number_space add.
func wantExhaustedEntry(t *testing.T, e audit.Entry, defID, principal uuid.UUID, diff string) {
	t.Helper()
	if e.Domain != audit.DomainRecord || e.Action != audit.ActionNumberSpaceExhausted ||
		e.SubjectType != audit.SubjectEntityDefinition || e.SubjectID != defID ||
		e.EntityDefinitionID == nil || *e.EntityDefinitionID != defID {
		t.Errorf("entry %+v, want number_space_exhausted on entity_definition %s", e, defID)
	}
	if e.VersionBefore != nil || e.VersionAfter != nil || e.ContentSHA256 != nil || e.SecretPaths != nil || e.ChangeSetID != nil {
		t.Errorf("entry %+v, want no versions, content hash, secret paths or change set", e)
	}
	if e.Actor.PrincipalID == nil || *e.Actor.PrincipalID != principal || e.Actor.System != nil || e.Source != audit.SourceUser || e.SourceRef != nil {
		t.Errorf("actor %+v source %s, want the user %s", e.Actor, e.Source, principal)
	}
	if e.CausationID == nil || *e.CausationID == uuid.Nil || e.CausationDepth != 0 {
		t.Errorf("causation %v depth %d, want the minted depth-0 origin", e.CausationID, e.CausationDepth)
	}
	if string(e.Diff) != diff {
		t.Errorf("diff = %s, want %s", e.Diff, diff)
	}
}

// An allocation on another epoch's counter, or a key that is not this
// format's, is refused (A2).
func TestCreateChecksTheAllocationsScopeKey(t *testing.T) {
	f := newFixture()
	format := linked(t, "{prefix}-{yyyy}{MM}{seq:4}", numbering.Refuse)
	for name, key := range map[string]string{
		"epoch 0":    "",
		"epoch 2":    "e2|d:202611",
		"short date": "e1|d:2026",
	} {
		t.Run(name, func(t *testing.T) {
			r := &numberRunner{next: 1, forge: func(a *pipeline.Allocation) { a.ScopeKey = key }}
			p, _ := numberedPipeline(t, fakeAuthz{}, fakeDefs{cur: assetDef(t), curVersion: 1, format: format}, r)
			tx := f.tx(t, stmt{match: sqlPreview, scan: scanString(formatPayload)})
			_, err := p.Write(t.Context(), tx, f.create(map[string]any{"name": "x"}))
			if err == nil || !strings.Contains(err.Error(), "is not this create's") {
				t.Fatalf("err = %v, want the allocation refused", err)
			}
			if tx.ranAny(sqlInsert) {
				t.Error("the create inserted after a foreign allocation")
			}
		})
	}
}

// A format snapshot that does not read stops the create before its number.
func TestCreateNumberFormatReadFailure(t *testing.T) {
	f := newFixture()
	r := &numberRunner{next: 1}
	p, _ := numberedPipeline(t, fakeAuthz{}, fakeDefs{cur: assetDef(t), curVersion: 1, formatErr: errBoom}, r)
	tx := f.tx(t)
	_, err := p.Write(t.Context(), tx, f.create(map[string]any{"name": "x"}))
	if !errors.Is(err, errBoom) || !strings.Contains(err.Error(), "read number format") {
		t.Fatalf("err = %v, want the read's error", err)
	}
	if r.taken() {
		t.Error("a number was taken after the format read failed")
	}
}

// A records_number_key violation is ErrNumberTaken with the type, epoch and
// scope: a defect, not a retryable error.
func TestCreateMapsANumberIndexViolation(t *testing.T) {
	f := newFixture()
	format := linked(t, "{prefix}-{yyyy}{MM}{seq:4}", numbering.Refuse)
	p, _ := numberedPipeline(t, fakeAuthz{}, fakeDefs{cur: assetDef(t), curVersion: 1, format: format}, &numberRunner{next: 1})
	dup := &pgconn.PgError{Code: "23505", ConstraintName: "records_number_key"}
	tx := f.tx(t, stmt{match: sqlPreview, scan: scanString(formatPayload)}, stmt{match: sqlInsert, err: dup})
	_, err := p.Write(t.Context(), tx, f.create(map[string]any{"name": "x"}))
	var nt *pipeline.ErrNumberTaken
	if !errors.As(err, &nt) || nt.EntityKey != "asset" || nt.Epoch != 1 || nt.ScopeKey != "e1|d:202611" || nt.Number != "AST-2026110001" {
		t.Fatalf("err = %v, want ErrNumberTaken for AST-2026110001 in epoch 1, e1|d:202611", err)
	}
	if errors.Is(err, pipeline.ErrRetryable) {
		t.Error("a number index violation is retryable")
	}
	if !errors.Is(err, dup) || !strings.Contains(err.Error(), `"AST-2026110001"`) {
		t.Errorf("err = %v, want it to wrap the PostgreSQL error and name the number", err)
	}
	// Another unique violation is not this defect.
	other := &pgconn.PgError{Code: "23505", ConstraintName: "records_pkey"}
	tx = f.tx(t, stmt{match: sqlPreview, scan: scanString(formatPayload)}, stmt{match: sqlInsert, err: other})
	if _, err := p.Write(t.Context(), tx, f.create(map[string]any{"name": "x"})); errors.As(err, &nt) {
		t.Errorf("records_pkey violation = %v, want no ErrNumberTaken", err)
	}
}

// The tests above assume numberRunner allocates in November 2026.
func TestAllocInstantIsInNovember2026(t *testing.T) {
	if got := allocInstant().Format("200601"); got != "202611" {
		t.Fatalf("allocInstant month = %s, want 202611", got)
	}
}

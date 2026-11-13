//go:build integration

// The real EventSink (internal/records/pipeline/eventsink, P3-42) against a
// real PostgreSQL under RLS, through the real pipeline as nava_app: starts of
// the published definitions a write fires, keyed by A5; record_changed
// signals only to the runs waiting on them and never to the writer (D-3-14,
// R9); the workflow_cause row of every interpreter-bound outbox row
// (D-3-29); and CI-10 case (g), the signal ping-pong between two runs on R
// that the depth limit stops at depth 9 (CTO ruling on TICK-88, 2026-11-01).

package pipeline_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/audit"
	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/records/pipeline"
	"github.com/Navaladi2019/TicketingSystem/internal/records/pipeline/eventsink"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

const versionBody = `{"format": "v1"}`

// publishTrigger publishes version 1 of a new definition of f on the
// incident type, triggered on on, and returns the definition and version ids.
func publishTrigger(t *testing.T, e env, f tenantFixture, on string) (def, version uuid.UUID) {
	t.Helper()
	def, version = workflowDefinition(t, e, f), uuid.New()
	// The audit entry is the one the commit check wants (P3-41b, #610).
	mustExec(t, e.super, `WITH v AS (INSERT INTO workflow_definition_versions (tenant_id, id, definition_id, version, body, body_sha256,
			step_ids, trigger_on, trigger_entity_definition_id, published_by)
			SELECT $1, $2, $3, 1, $4::text::json, sha256(convert_to($4::text, 'UTF8')), '{}', $5, d.id, $6
			FROM entity_definitions d WHERE d.tenant_id = $1 AND d.key = $7 RETURNING tenant_id, definition_id)
		INSERT INTO audit_log (tenant_id, id, domain, action, subject_type, subject_id, actor_system, source, content_sha256)
		SELECT tenant_id, gen_random_uuid(), 'config', 'publish', 'workflow_definition', definition_id, 'system', 'system', sha256(''::bytea) FROM v`,
		f.id, version, def, versionBody, on, f.principal.PrincipalID, entityKey)
	return def, version
}

// openRun is an open nava.interpreter run on record, as EnqueueStart leaves
// its workflow_ref; waiting sets record_changed_signal_id as NextStep does.
// It returns the run's ref and the signal id it waits on (uuid.Nil if none).
func openRun(t *testing.T, e env, f tenantFixture, record uuid.UUID, waiting bool) (ref, signal uuid.UUID) {
	t.Helper()
	ref = uuid.Must(uuid.NewV7())
	var sig *uuid.UUID
	if waiting {
		signal = uuid.New()
		sig = &signal
	}
	mustExec(t, e.super, `INSERT INTO workflow_ref (tenant_id, outbox_id, record_id, kind, record_changed_signal_id)
		VALUES ($1, $2, $3, $4, $5)`, f.id, ref, record, outbox.InterpreterKind, sig)
	return ref, signal
}

// row is one interpreter-bound outbox row.
type interpRow struct {
	id        uuid.UUID
	targetKey string
	payload   map[string]string
	origin    uuid.UUID
	depth     int16
}

func interpRows(t *testing.T, e env, tenant uuid.UUID, kind string) []interpRow {
	t.Helper()
	rows, err := e.super.Query(context.Background(), `SELECT id, target_key, payload, causation_id, causation_depth
		FROM outbox WHERE tenant_id = $1 AND kind = $2 ORDER BY id`, tenant, kind)
	if err != nil {
		t.Fatal(err)
	}
	out, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (interpRow, error) {
		var (
			x   interpRow
			raw []byte
		)
		if err := r.Scan(&x.id, &x.targetKey, &raw, &x.origin, &x.depth); err != nil {
			return x, err
		}
		return x, json.Unmarshal(raw, &x.payload)
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// signalsTo is the record_changed signal rows whose target is the run ref.
func signalsTo(t *testing.T, e env, tenant, ref uuid.UUID) []interpRow {
	t.Helper()
	var out []interpRow
	for _, r := range interpRows(t, e, tenant, outbox.InterpreterSignalKind) {
		if r.targetKey == outbox.InterpreterKind+":"+ref.String() {
			out = append(out, r)
		}
	}
	return out
}

// wantEveryRowHasItsCause: every interpreter-bound outbox row of the tenant
// has, committed with it, the workflow_cause row with its own id, its ref
// (its own id for a start, its target for a signal) and its origin and
// depth (D-3-29).
func wantEveryRowHasItsCause(t *testing.T, e env, tenant uuid.UUID) {
	t.Helper()
	if n := count(t, e.super, `SELECT count(*) FROM outbox o WHERE o.tenant_id = $1 AND o.kind IN ($2, $3)
		AND NOT EXISTS (SELECT 1 FROM workflow_cause c WHERE c.tenant_id = o.tenant_id AND c.outbox_id = o.id
			AND c.ref_outbox_id = split_part(o.target_key, ':', 2)::uuid
			AND c.causation_origin_id = o.causation_id AND c.causation_depth = o.causation_depth)`,
		tenant, outbox.InterpreterKind, outbox.InterpreterSignalKind); n != 0 {
		t.Errorf("%d interpreter-bound outbox rows without their workflow_cause row", n)
	}
	if n := count(t, e.super, `SELECT count(*) FROM workflow_cause c WHERE c.tenant_id = $1
		AND NOT EXISTS (SELECT 1 FROM outbox o WHERE o.tenant_id = c.tenant_id AND o.id = c.outbox_id)`, tenant); n != 0 {
		t.Errorf("%d workflow_cause rows without their outbox row", n)
	}
}

// A user create of R fires, once, each definition whose current version
// triggers on record.created, and no other: the start row is
// nava.interpreter:<its own id> (D-3-13 as amended), its payload exactly
// {definition_version_id, record_id}, its causation R's, and its
// workflow_ref and workflow_cause rows are written with it.
func TestEventSink_CreateStartsTriggeredDefinitions(t *testing.T) {
	e := migrated(t)
	f := seedTenant(t, e)
	onCreate, version := publishTrigger(t, e, f, "record.created")
	onUpdate, _ := publishTrigger(t, e, f, "record.updated")
	p := newPipeline(e, eventsink.Sink{})

	r := mustCreate(t, e, p, f, map[string]any{"title": "printer"})
	rows := interpRows(t, e, f.id, outbox.InterpreterKind)
	if len(rows) != 1 {
		t.Fatalf("%d start rows, want 1", len(rows))
	}
	s := rows[0]
	if s.targetKey != outbox.InterpreterKind+":"+s.id.String() {
		t.Errorf("target_key %q, want nava.interpreter:<own id>", s.targetKey)
	}
	want := map[string]string{"definition_version_id": version.String(), "record_id": r.RecordID.String()}
	if !maps.Equal(s.payload, want) {
		t.Errorf("payload %v, want %v", s.payload, want)
	}
	var origin uuid.UUID
	var depth int16
	if err := e.super.QueryRow(context.Background(), `SELECT causation_origin_id, causation_depth FROM records WHERE id = $1`,
		r.RecordID).Scan(&origin, &depth); err != nil {
		t.Fatal(err)
	}
	if s.origin != origin || s.depth != depth || depth != 0 {
		t.Errorf("start causation (%s, %d), want the record's (%s, 0)", s.origin, s.depth, origin)
	}
	if n := starts(t, e, f.id, r.RecordID, onCreate); n != 1 {
		t.Errorf("%d cascade_start rows for the create trigger, want 1", n)
	}
	if n := starts(t, e, f.id, r.RecordID, onUpdate); n != 0 {
		t.Errorf("%d cascade_start rows for the update trigger after a create, want 0", n)
	}
	if n := count(t, e.super, `SELECT count(*) FROM workflow_ref WHERE tenant_id = $1 AND outbox_id = $2 AND record_id = $3`,
		f.id, s.id, r.RecordID); n != 1 {
		t.Errorf("%d workflow_ref rows for the start, want 1", n)
	}
	wantEveryRowHasItsCause(t, e, f.id)

	// A create of a type nothing triggers on starts nothing and takes the
	// quiet path (the sink's prefetch read said so).
	g := seedTenant(t, e)
	mustCreate(t, e, newPipeline(e, eventsink.Sink{}), g, map[string]any{"title": "quiet"})
	if n := len(interpRows(t, e, g.id, outbox.InterpreterKind)); n != 0 {
		t.Errorf("%d starts in a tenant with no definition, want 0", n)
	}
}

// Two writes on R in one cascade start a definition once; the second write
// commits and leaves the cascade_cycle entry of the suppressed start (A5-4).
func TestEventSink_TwoWritesInOneCascadeStartOnce(t *testing.T) {
	e := migrated(t)
	f := seedTenant(t, e)
	d, _ := publishTrigger(t, e, f, "record.updated")
	p := newPipeline(e, eventsink.Sink{})
	ctx := context.Background()

	r := mustCreate(t, e, p, f, map[string]any{"title": "printer"})
	var origin uuid.UUID
	if err := e.super.QueryRow(ctx, `SELECT causation_origin_id FROM records WHERE id = $1`, r.RecordID).Scan(&origin); err != nil {
		t.Fatal(err)
	}
	c := outbox.Causation{OriginID: origin, Depth: 1}
	first, err := write(ctx, e, p, f.id, workflowCmd(f, r.RecordID, r.Version, c, "step-1"))
	if err != nil {
		t.Fatalf("first write: %v", err)
	}
	if _, err := write(ctx, e, p, f.id, workflowCmd(f, r.RecordID, first.Version, c, "step-2")); err != nil {
		t.Fatalf("second write in the same cascade: %v, want it committed", err)
	}
	if n := starts(t, e, f.id, r.RecordID, d); n != 1 {
		t.Errorf("%d starts of D, want 1", n)
	}
	if n := len(interpRows(t, e, f.id, outbox.InterpreterKind)); n != 1 {
		t.Errorf("%d start rows, want 1", n)
	}
	if n := entries(t, e, f.id, string(audit.ActionCascadeCycle)); n != 1 {
		t.Errorf("%d cascade_cycle entries, want 1", n)
	}
	wantEveryRowHasItsCause(t, e, f.id)
}

// When run X writes R, run Y waiting on record_changed on R gets one signal
// with payload exactly {"signal": <Y's signal id>} at the write's causation;
// X gets none for its own write; run Z, open but not waiting, gets none
// (R9). A person's write signals both waiting runs.
func TestEventSink_SignalsOnlyWaitingRunsNeverTheWriter(t *testing.T) {
	e := migrated(t)
	f := seedTenant(t, e)
	p := newPipeline(e, eventsink.Sink{})
	ctx := context.Background()

	r := mustCreate(t, e, p, f, map[string]any{"title": "printer"})
	x, _ := openRun(t, e, f, r.RecordID, true)
	y, ySignal := openRun(t, e, f, r.RecordID, true)
	z, _ := openRun(t, e, f, r.RecordID, false)

	c := outbox.Causation{OriginID: uuid.New(), Depth: 3}
	cmd := workflowCmd(f, r.RecordID, r.Version, c, "x-writes")
	cmd.WorkflowRef = x
	res, err := write(ctx, e, p, f.id, cmd)
	if err != nil {
		t.Fatalf("X's write: %v", err)
	}
	if got := signalsTo(t, e, f.id, x); len(got) != 0 {
		t.Errorf("X was signalled about its own write: %+v", got)
	}
	if got := signalsTo(t, e, f.id, z); len(got) != 0 {
		t.Errorf("Z, not waiting, was signalled: %+v", got)
	}
	got := signalsTo(t, e, f.id, y)
	if len(got) != 1 {
		t.Fatalf("%d signals to Y, want 1", len(got))
	}
	if want := map[string]string{"signal": ySignal.String()}; !maps.Equal(got[0].payload, want) {
		t.Errorf("Y's signal payload %v, want %v", got[0].payload, want)
	}
	if got[0].origin != c.OriginID || got[0].depth != c.Depth {
		t.Errorf("Y's signal causation (%s, %d), want the write's (%s, %d)", got[0].origin, got[0].depth, c.OriginID, c.Depth)
	}

	if _, err := write(ctx, e, p, f.id, updateCmd(f, r.RecordID, res.Version, map[string]any{"summary": "a person"})); err != nil {
		t.Fatalf("person's write: %v", err)
	}
	if n := len(signalsTo(t, e, f.id, x)); n != 1 {
		t.Errorf("%d signals to X after a person's write, want 1", n)
	}
	if n := len(signalsTo(t, e, f.id, y)); n != 2 {
		t.Errorf("%d signals to Y after both writes, want 2", n)
	}
	if n := len(signalsTo(t, e, f.id, z)); n != 0 {
		t.Errorf("%d signals to Z, want 0", n)
	}

	// A closed run is not signalled, even with its wait left set.
	mustExec(t, e.super, `UPDATE workflow_ref SET closed_at = now() WHERE tenant_id = $1 AND outbox_id = $2`, f.id, y)
	if _, err := write(ctx, e, p, f.id, updateCmd(f, r.RecordID, res.Version+1, map[string]any{"summary": "again"})); err != nil {
		t.Fatalf("third write: %v", err)
	}
	if n := len(signalsTo(t, e, f.id, y)); n != 2 {
		t.Errorf("%d signals to closed Y, want still 2", n)
	}
	wantEveryRowHasItsCause(t, e, f.id)
}

// Only a system writer may name the writing run (Command.WorkflowRef).
func TestEventSink_UserWriteCannotNameAWorkflowRef(t *testing.T) {
	e := migrated(t)
	f := seedTenant(t, e)
	p := newPipeline(e, eventsink.Sink{})
	r := mustCreate(t, e, p, f, map[string]any{"title": "printer"})
	x, _ := openRun(t, e, f, r.RecordID, true)
	cmd := updateCmd(f, r.RecordID, r.Version, map[string]any{"summary": "hide from x"})
	cmd.WorkflowRef = x
	if _, err := write(context.Background(), e, p, f.id, cmd); !errors.Is(err, pipeline.ErrInvalidCommand) {
		t.Fatalf("a user write naming a workflow ref: %v, want ErrInvalidCommand", err)
	}
	if n := len(signalsTo(t, e, f.id, x)); n != 0 {
		t.Errorf("%d signals after a refused write, want 0", n)
	}
}

// resumeCause is what the P3-43 RecordWriter does with StepRef.Cause before
// a write (workerguard.ResolveCause): read the workflow_cause row under RLS
// as nava_app, refuse it unless its ref is the run's own, and write one
// deeper.
func resumeCause(t *testing.T, e env, tenant, run, cause uuid.UUID) outbox.Causation {
	t.Helper()
	var (
		ref    uuid.UUID
		origin uuid.UUID
		depth  int16
	)
	err := tenancy.InTenantTx(context.Background(), e.app, tenant, func(tx pgx.Tx) error {
		return tx.QueryRow(context.Background(), `SELECT ref_outbox_id, causation_origin_id, causation_depth
			FROM workflow_cause WHERE tenant_id = $1 AND outbox_id = $2`, tenant, cause).Scan(&ref, &origin, &depth)
	})
	if err != nil {
		t.Fatalf("read workflow_cause %s: %v", cause, err)
	}
	if ref != run {
		t.Fatalf("workflow_cause %s belongs to %s, not to the resumed run %s", cause, ref, run)
	}
	return outbox.Causation{OriginID: origin, Depth: depth + 1}
}

// CI-10 (g): runs X and Y both wait on record_changed on R. A person edits R
// (depth 0), which signals both. X resumes and writes R at depth 1, which
// signals Y and not X; Y resumes and writes at depth 2, which signals X and
// not Y; and so on. Each write takes its causation only from the
// workflow_cause row of the signal that resumed it. The write at depth 9 is
// refused (outbox.ErrCascadeTooDeep) with one cascade_too_deep entry, and the
// ping-pong stops (CTO ruling on TICK-88, 2026-11-01).
func TestCI10_G_SignalPingPongBetweenTwoRunsStopsAtDepth9(t *testing.T) {
	e := migrated(t)
	f := seedTenant(t, e)
	p := newPipeline(e, eventsink.Sink{})
	ctx := context.Background()

	r := mustCreate(t, e, p, f, map[string]any{"title": "printer"})
	x, _ := openRun(t, e, f, r.RecordID, true)
	y, _ := openRun(t, e, f, r.RecordID, true)
	res, err := write(ctx, e, p, f.id, updateCmd(f, r.RecordID, r.Version, map[string]any{"summary": "a person"}))
	if err != nil {
		t.Fatalf("person's write: %v", err)
	}
	version := res.Version
	var origin uuid.UUID
	if err := e.super.QueryRow(ctx, `SELECT causation_origin_id FROM records WHERE id = $1`, r.RecordID).Scan(&origin); err != nil {
		t.Fatal(err)
	}

	limit := int(config.CascadeDepthLimit)
	writer, other := x, y
	for depth := 1; ; depth++ {
		before := len(signalsTo(t, e, f.id, writer))
		in := signalsTo(t, e, f.id, writer)
		if len(in) == 0 {
			t.Fatalf("depth %d: the writer was never signalled", depth)
		}
		c := resumeCause(t, e, f.id, writer, in[len(in)-1].id)
		if c.OriginID != origin || int(c.Depth) != depth {
			t.Fatalf("depth %d: resolved causation (%s, %d), want (%s, %d)", depth, c.OriginID, c.Depth, origin, depth)
		}
		cmd := pipeline.Command{Op: pipeline.OpUpdate, EntityKey: entityKey, RecordID: r.RecordID, ExpectedVersion: version,
			Data: map[string]any{"summary": fmt.Sprintf("depth %d", depth)}, Actor: f.system(), Source: pipeline.SourceWorkflow,
			Causation: c, IdempotencyKey: fmt.Sprintf("wf:%s:%d", writer, depth), WorkflowRef: writer}
		otherBefore := len(signalsTo(t, e, f.id, other))
		res, err := write(ctx, e, p, f.id, cmd)
		if depth == limit+1 {
			if !errors.Is(err, outbox.ErrCascadeTooDeep) {
				t.Fatalf("the write at depth %d: %v, want ErrCascadeTooDeep", depth, err)
			}
			break
		}
		if err != nil {
			t.Fatalf("the write at depth %d: %v", depth, err)
		}
		version = res.Version
		if n := len(signalsTo(t, e, f.id, writer)); n != before {
			t.Fatalf("depth %d: the writer was signalled about its own write", depth)
		}
		got := signalsTo(t, e, f.id, other)
		if len(got) != otherBefore+1 {
			t.Fatalf("depth %d: %d new signals to the other run, want 1", depth, len(got)-otherBefore)
		}
		if s := got[len(got)-1]; s.origin != origin || int(s.depth) != depth {
			t.Fatalf("depth %d: signal causation (%s, %d), want (%s, %d)", depth, s.origin, s.depth, origin, depth)
		}
		writer, other = other, writer
	}

	if n := entries(t, e, f.id, "cascade_too_deep"); n != 1 {
		t.Errorf("%d cascade_too_deep entries, want 1", n)
	}
	// Two signals from the person's write, one from each of the 8 writes.
	if n := len(interpRows(t, e, f.id, outbox.InterpreterSignalKind)); n != 2+limit {
		t.Errorf("%d signal rows, want %d", n, 2+limit)
	}
	var stored int64
	if err := e.super.QueryRow(ctx, `SELECT version FROM records WHERE id = $1`, r.RecordID).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if stored != version {
		t.Errorf("record version %d, want %d: the refused write changed it", stored, version)
	}
	depths := []int16{}
	for _, s := range interpRows(t, e, f.id, outbox.InterpreterSignalKind) {
		depths = append(depths, s.depth)
	}
	if slices.Max(depths) != config.CascadeDepthLimit {
		t.Errorf("deepest signal %d, want %d", slices.Max(depths), config.CascadeDepthLimit)
	}
	wantEveryRowHasItsCause(t, e, f.id)
}

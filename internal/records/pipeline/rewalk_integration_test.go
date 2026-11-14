//go:build integration

// TICK-784: the interpreter's activities re-walk instead of failing the step
// when a newer delivery reaches the run mid-step (a stale cause) or when the
// record moves between the walk's read and the wait (ErrRecordMoved),
// through the real record ports on a real PostgreSQL, as nava_worker under
// RLS.

package pipeline_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/records/pipeline/eventsink"
	"github.com/Navaladi2019/TicketingSystem/internal/workerguard"
	"github.com/Navaladi2019/TicketingSystem/internal/workflow/def"
	"github.com/Navaladi2019/TicketingSystem/internal/workflow/interp"
)

// interpVersion parses a definition and gives every key and signal a step id.
func interpVersion(t *testing.T, src string) *interp.Version {
	t.Helper()
	d, err := def.Parse([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	return &interp.Version{Definition: d, StepIDs: def.AssignStepIDs(d, nil)}
}

// hookedDefinitions is a DefinitionStore of one version that runs hook on
// each load: something that happens mid-step, after the activity started.
type hookedDefinitions struct {
	v    *interp.Version
	hook func()
}

func (h hookedDefinitions) Load(context.Context, uuid.UUID, uuid.UUID) (*interp.Version, error) {
	if h.hook != nil {
		h.hook()
	}
	return h.v, nil
}

const applyDefinition = `{"format": "nava.workflow/v1", "trigger": {"on": "record.created", "entity": "incident"},
  "start": "apply", "steps": [
    {"key": "apply", "type": "update_record", "set": {"summary": "applied"}, "next": "done"},
    {"key": "done", "type": "end"}]}`

const waitDefinition = `{"format": "nava.workflow/v1", "trigger": {"on": "record.created", "entity": "incident"},
  "start": "wait", "steps": [
    {"key": "wait", "type": "wait_signal", "name": "record_changed", "timeout": "P1D", "on_timeout": "done", "next": "done"},
    {"key": "done", "type": "end"}]}`

// A newer delivery reaches the run while ApplyStep runs under the older
// cause: the step completes, written once at the newest cause's origin and
// depth + 1. A newer row of another run of the tenant, or of another
// tenant's run that shares the ref, is not the run's newest.
func TestApplyStep_NewCauseMidStepWritesWithTheNewestDepth(t *testing.T) {
	e := migrated(t)
	a, b := seedTenant(t, e), seedTenant(t, e)
	w := newWorker(t, e)
	r := mustCreate(t, e, newPipeline(e, nil), a, map[string]any{"title": "printer"})
	run, _ := openRun(t, e, a, r.RecordID, false)
	other, _ := openRun(t, e, a, r.RecordID, false)
	rb := mustCreate(t, e, newPipeline(e, nil), b, map[string]any{"title": "b"})
	_, _ = openRun(t, e, b, rb.RecordID, false)

	now := time.Now()
	c1 := seedCauseRow(t, e, a.id, run, uuid.New(), 1, now.Add(-time.Hour))
	newestOrigin := uuid.New()
	var once sync.Once
	v := interpVersion(t, applyDefinition)
	acts := &interp.Activities{
		Definitions: hookedDefinitions{v: v, hook: func() {
			once.Do(func() {
				// The relay dispatches a new delivery to the run mid-step.
				seedCauseRow(t, e, a.id, run, newestOrigin, 4, now)
				// Later rows that are not the run's: another run's, and tenant
				// B's under the same ref. Neither may be taken.
				seedCauseRow(t, e, a.id, other, uuid.New(), 0, now.Add(time.Minute))
				seedCauseRow(t, e, b.id, run, uuid.New(), 0, now.Add(time.Minute))
			})
		}},
		Records: w.reader,
		Writer:  w.writer,
		Waits:   w.waits,
		Causes:  workerguard.CauseTracker{Pool: w.pool},
	}
	ref := interp.StepRef{DefinitionVersionID: uuid.New(), RecordID: r.RecordID, Step: v.StepIDs["apply"], Cause: c1}
	if err := asRun(t, a.id, run, func(ctx context.Context) error { return acts.ApplyStep(ctx, ref) }); err != nil {
		t.Fatalf("ApplyStep with a cause gone stale mid-step: %v, want the step to complete", err)
	}
	if got := versionOf(t, e, r.RecordID); got != r.Version+1 {
		t.Fatalf("record at version %d, want %d (written once)", got, r.Version+1)
	}
	hist := historyOf(t, e, a.id, r.RecordID)
	last := hist[len(hist)-1]
	if last.Source != "workflow" || last.Depth != 5 || last.Origin == nil || *last.Origin != newestOrigin {
		t.Errorf("history entry = %+v, want a workflow write at the newest cause's origin %s and depth 5 (4 + 1)", last, newestOrigin)
	}
	if got := versionOf(t, e, rb.RecordID); got != rb.Version {
		t.Errorf("tenant B's record moved to version %d", got)
	}

	// The same step retried (its completion lost) is a replay: nothing more.
	if err := asRun(t, a.id, run, func(ctx context.Context) error { return acts.ApplyStep(ctx, ref) }); err != nil {
		t.Fatalf("replay: %v", err)
	}
	if got := len(historyOf(t, e, a.id, r.RecordID)); got != len(hist) {
		t.Errorf("%d history entries after the replay, want %d", got, len(hist))
	}
}

// The newest-cause lookup takes only the calling run's rows of the verified
// tenant; a run with none of its own is refused, never written at depth 0.
func TestApplyStep_StaleCauseWithNoRowOfItsOwnIsRefused(t *testing.T) {
	e := migrated(t)
	a := seedTenant(t, e)
	w := newWorker(t, e)
	r := mustCreate(t, e, newPipeline(e, nil), a, map[string]any{"title": "printer"})
	run, _ := openRun(t, e, a, r.RecordID, false)
	other, _ := openRun(t, e, a, r.RecordID, false)
	foreign := seedCauseRow(t, e, a.id, other, uuid.New(), 0, time.Now())
	v := interpVersion(t, applyDefinition)
	acts := &interp.Activities{Definitions: hookedDefinitions{v: v}, Records: w.reader, Writer: w.writer, Causes: workerguard.CauseTracker{Pool: w.pool}}
	ref := interp.StepRef{DefinitionVersionID: uuid.New(), RecordID: r.RecordID, Step: v.StepIDs["apply"], Cause: foreign}
	err := asRun(t, a.id, run, func(ctx context.Context) error { return acts.ApplyStep(ctx, ref) })
	if err == nil {
		t.Fatal("another run's cause was accepted")
	}
	if got := versionOf(t, e, r.RecordID); got != r.Version {
		t.Fatalf("record at version %d, want %d (no write, no depth-0 fallback)", got, r.Version)
	}
}

// changingReader is the real reader, but a real write changes the record
// after each of its first n reads: between the walk's read and its wait.
type changingReader struct {
	interp.RecordReader
	change func(version int64)
	mu     sync.Mutex
	n      int
}

func (c *changingReader) Read(ctx context.Context, tenantID, recordID uuid.UUID) (map[string]any, int64, error) {
	data, version, err := c.RecordReader.Read(ctx, tenantID, recordID)
	c.mu.Lock()
	defer c.mu.Unlock()
	if err == nil && c.n > 0 {
		c.n--
		c.change(version)
	}
	return data, version, err
}

// A record that changes N times between the walk's read and the wait, within
// one NextStep attempt: the attempt still succeeds, the run waits on the
// current version, and the next write signals it. None of the N writes
// signalled it, because it was not yet waiting.
func TestNextStep_RecordChangingBetweenReadAndWaitStillWaits(t *testing.T) {
	e := migrated(t)
	f := seedTenant(t, e)
	w := newWorker(t, e)
	p := newPipeline(e, eventsink.Sink{})
	r := mustCreate(t, e, p, f, map[string]any{"title": "printer"})
	run, _ := openRun(t, e, f, r.RecordID, false)
	const changes = interp.MaxRecordMovedRewalks
	reader := &changingReader{RecordReader: w.reader, n: changes, change: func(version int64) {
		if _, err := write(context.Background(), e, p, f.id, updateCmd(f, r.RecordID, version, map[string]any{"summary": uuid.NewString()})); err != nil {
			t.Error(err)
		}
	}}
	v := interpVersion(t, waitDefinition)
	acts := &interp.Activities{Definitions: hookedDefinitions{v: v}, Records: reader, Writer: w.writer, Waits: w.waits, Causes: workerguard.CauseTracker{Pool: w.pool}}
	ref := interp.StepRef{DefinitionVersionID: uuid.New(), RecordID: r.RecordID, Cause: uuid.New()}

	var d workerguard.Directive
	if err := asRun(t, f.id, run, func(ctx context.Context) (err error) {
		d, err = acts.NextStep(ctx, ref)
		return err
	}); err != nil {
		t.Fatalf("NextStep with the record changing %d times: %v, want one successful attempt", changes, err)
	}
	if d.Kind != workerguard.KindWaitSignal {
		t.Fatalf("directive %+v, want the wait", d)
	}
	if got := versionOf(t, e, r.RecordID); got != r.Version+changes {
		t.Fatalf("record at version %d, want %d", got, r.Version+changes)
	}
	want := v.StepIDs[def.SignalKey(interp.RecordChangedSignal)]
	if n := count(t, e.super, `SELECT count(*) FROM workflow_ref WHERE tenant_id = $1 AND outbox_id = $2 AND record_changed_signal_id = $3`, f.id, run, want); n != 1 {
		t.Fatalf("run is not waiting on record_changed")
	}
	if n := len(signalsTo(t, e, f.id, run)); n != 0 {
		t.Fatalf("%d signals while the run was not waiting, want 0", n)
	}
	if _, err := write(context.Background(), e, p, f.id, updateCmd(f, r.RecordID, r.Version+changes, map[string]any{"summary": "later"})); err != nil {
		t.Fatal(err)
	}
	if n := len(signalsTo(t, e, f.id, run)); n != 1 {
		t.Fatalf("%d signals to the waiting run after the next write, want 1", n)
	}
}

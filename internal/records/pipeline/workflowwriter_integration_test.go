//go:build integration

// The worker's RecordReader, RecordWriter and WaitTracker (P3-43, TICK-712)
// through the real records pipeline against a real PostgreSQL, as the real
// nava_worker role under RLS: an update_record step is one pipeline write as
// source=workflow, with causation only from the run's workflow_cause row, on
// a tenant the activity's header names; the burst of too-deep writes does not
// exhaust the worker's pools; and the record_changed wait is never set on a
// stale version (contract amendment of 2026-11-13).

package pipeline_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"

	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/records/pipeline/eventsink"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
	"github.com/Navaladi2019/TicketingSystem/internal/workerguard"
	"github.com/Navaladi2019/TicketingSystem/internal/workflow/interp"
	"github.com/Navaladi2019/TicketingSystem/internal/workflow/recordports"
)

// workerWidth is the worker pool's width in these tests, and pipelineWidth
// the pipeline's own (pipelineMaxConns in cmd/nava-worker/main.go).
const (
	workerWidth   = 4
	pipelineWidth = 4
)

// worker is the nava-worker's side: a nava_worker pool, the pipeline on a
// nava_worker pool of its own, and the three ports over them, as
// cmd/nava-worker wires them.
type worker struct {
	pool, pipePool *pgxpool.Pool
	reader         recordports.Reader
	writer         recordports.Writer
	waits          workerguard.WaitTracker
}

func newWorker(t *testing.T, e env) worker {
	t.Helper()
	sized := func(n int32) *pgxpool.Pool {
		cfg := serverPoolConfig(t, rolePoolDSN(t, e, "nava_worker"))
		cfg.MaxConns = n
		return newPool(t, cfg)
	}
	w := worker{pool: sized(workerWidth), pipePool: sized(pipelineWidth)}
	w.reader = recordports.Reader{Pool: w.pool}
	w.writer = recordports.NewWriter(w.pool, w.pipePool)
	w.waits = workerguard.WaitTracker{Pool: w.pool}
	return w
}

// asRun runs fn as an activity of the interpreter run ref of tenant: the
// workflow id is the relay's t:<tenant>:nava.interpreter:<ref>, and the
// context carries the tenant the activity guard would verify. It returns fn's
// error as is (the Temporal wrapper is bypassed). Safe to call concurrently.
func asRun(t *testing.T, tenant, ref uuid.UUID, fn func(ctx context.Context) error) error {
	t.Helper()
	var (
		suite testsuite.WorkflowTestSuite
		out   error
	)
	env := suite.NewTestWorkflowEnvironment()
	env.SetStartWorkflowOptions(client.StartWorkflowOptions{ID: "t:" + tenant.String() + ":" + outbox.InterpreterKind + ":" + ref.String()})
	env.RegisterActivityWithOptions(func(ctx context.Context) error {
		out = fn(workerguard.ContextWithTenant(ctx, tenant))
		return nil
	}, activity.RegisterOptions{Name: "probe"})
	env.ExecuteWorkflow(func(ctx workflow.Context) error {
		ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{StartToCloseTimeout: time.Minute})
		return workflow.ExecuteActivity(ctx, "probe").Get(ctx, nil)
	})
	if err := env.GetWorkflowError(); err != nil {
		t.Error(err)
	}
	return out
}

// seedCauseRow writes a workflow_cause row as the enqueuing transaction does,
// created at at.
func seedCauseRow(t *testing.T, e env, tenant, ref, origin uuid.UUID, depth int16, at time.Time) uuid.UUID {
	t.Helper()
	id := uuid.Must(uuid.NewV7())
	mustExec(t, e.super, `INSERT INTO workflow_cause (tenant_id, outbox_id, ref_outbox_id, causation_origin_id, causation_depth, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)`, tenant, id, ref, origin, depth, at)
	return id
}

type historyEntry struct {
	Source string
	Action string
	Origin *uuid.UUID
	Depth  int16
}

// historyOf is the record's audit entries, oldest first.
func historyOf(t *testing.T, e env, tenant, record uuid.UUID) []historyEntry {
	t.Helper()
	rows, err := e.super.Query(context.Background(), `SELECT source, action, causation_id, causation_depth
		FROM audit_log WHERE tenant_id = $1 AND subject_id = $2 AND domain = 'record' ORDER BY occurred_at, id`, tenant, record)
	if err != nil {
		t.Fatal(err)
	}
	got, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (h historyEntry, err error) {
		err = r.Scan(&h.Source, &h.Action, &h.Origin, &h.Depth)
		return
	})
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func versionOf(t *testing.T, e env, record uuid.UUID) int64 {
	t.Helper()
	_, _, _, v, _ := storedData(t, e, record)
	return v
}

// An update_record step changes the record once, as source=workflow, with the
// delivery's causation (origin kept, depth + 1) read from its workflow_cause
// row and nothing else; the record's history shows it; a re-run with the
// same key changes nothing and succeeds.
func TestWorkflowWriter_UpdateRecordOnceAsWorkflow(t *testing.T) {
	e := migrated(t)
	f := seedTenant(t, e)
	w := newWorker(t, e)
	r := mustCreate(t, e, newPipeline(e, nil), f, map[string]any{"title": "printer"})
	run, _ := openRun(t, e, f, r.RecordID, false)
	origin := uuid.New()
	cause := seedCauseRow(t, e, f.id, run, origin, 2, time.Now())
	key := interp.IdempotencyKey("t:"+f.id.String()+":"+outbox.InterpreterKind+":"+run.String(), uuid.New())

	apply := func() error {
		return asRun(t, f.id, run, func(ctx context.Context) error {
			return w.writer.Write(ctx, f.id, r.RecordID, uuid.New(), cause, key, map[string]any{"status": "closed"})
		})
	}
	if err := apply(); err != nil {
		t.Fatalf("update_record: %v", err)
	}
	data, state, _, version, _ := storedData(t, e, r.RecordID)
	if state != "closed" || version != r.Version+1 {
		t.Fatalf("record after the step: state %q version %d (data %v), want closed at version %d", state, version, data, r.Version+1)
	}
	hist := historyOf(t, e, f.id, r.RecordID)
	last := hist[len(hist)-1]
	if last.Source != "workflow" || last.Origin == nil || *last.Origin != origin || last.Depth != 3 {
		t.Fatalf("history entry = %+v, want source workflow, origin %s, depth 3 (cause depth 2 + 1)", last, origin)
	}

	// The same key again: nothing changes, the earlier result comes back.
	if err := apply(); err != nil {
		t.Fatalf("re-run with the same key: %v", err)
	}
	if got := versionOf(t, e, r.RecordID); got != version {
		t.Errorf("record at version %d after the re-run, want %d", got, version)
	}
	if got := len(historyOf(t, e, f.id, r.RecordID)); got != len(hist) {
		t.Errorf("%d history entries after the re-run, want %d", got, len(hist))
	}

	// The reader returns the record and its version.
	var (
		read map[string]any
		rv   int64
	)
	if err := asRun(t, f.id, run, func(ctx context.Context) (err error) {
		read, rv, err = w.reader.Read(ctx, f.id, r.RecordID)
		return err
	}); err != nil || rv != version || read["title"] != "printer" {
		t.Fatalf("Read = (%v, %d, %v), want the record at version %d", read, rv, err, version)
	}
}

// A business rejection maps to the port's error, and nothing is written.
func TestWorkflowWriter_PipelineRejectionsMapToPortErrors(t *testing.T) {
	e := migrated(t)
	f := seedTenant(t, e)
	w := newWorker(t, e)
	r := mustCreate(t, e, newPipeline(e, nil), f, map[string]any{"title": "printer"})
	run, _ := openRun(t, e, f, r.RecordID, false)
	cause := seedCauseRow(t, e, f.id, run, uuid.New(), 0, time.Now())
	for name, c := range map[string]struct {
		set  map[string]any
		want error
	}{
		"validation":         {map[string]any{"priority": "urgent"}, interp.ErrValidation},
		"unknown property":   {map[string]any{"nope": 1}, interp.ErrValidation},
		"illegal transition": {map[string]any{"status": "open"}, interp.ErrIllegalTransition},
	} {
		t.Run(name, func(t *testing.T) {
			err := asRun(t, f.id, run, func(ctx context.Context) error {
				return w.writer.Write(ctx, f.id, r.RecordID, uuid.New(), cause, "k-"+name, c.set)
			})
			if !errors.Is(err, c.want) {
				t.Errorf("Write = %v, want %v", err, c.want)
			}
		})
	}
	if got := versionOf(t, e, r.RecordID); got != r.Version {
		t.Errorf("record at version %d after refused writes, want %d", got, r.Version)
	}
}

// Tenant isolation: tenant B's records start no tenant A workflow, and a
// tenant A run reads and writes no tenant B record (the not-found rejection).
func TestWorkflowWriter_TenantIsolation(t *testing.T) {
	e := migrated(t)
	a, b := seedTenant(t, e), seedTenant(t, e)
	w := newWorker(t, e)
	sinkPipe := newPipeline(e, eventsink.Sink{})
	publishTrigger(t, e, a, "record.created")
	publishTrigger(t, e, a, "record.updated")

	rb := mustCreate(t, e, sinkPipe, b, map[string]any{"title": "of tenant b"})
	if _, err := write(context.Background(), e, sinkPipe, b.id, updateCmd(b, rb.RecordID, rb.Version, map[string]any{"summary": "x"})); err != nil {
		t.Fatal(err)
	}
	if n := count(t, e.super, `SELECT count(*) FROM outbox WHERE kind = $1`, outbox.InterpreterKind); n != 0 {
		t.Fatalf("%d interpreter starts after tenant B's writes, want 0 (only tenant A has triggers)", n)
	}

	ra := mustCreate(t, e, newPipeline(e, nil), a, map[string]any{"title": "of tenant a"})
	run, _ := openRun(t, e, a, ra.RecordID, false)
	cause := seedCauseRow(t, e, a.id, run, uuid.New(), 0, time.Now())
	err := asRun(t, a.id, run, func(ctx context.Context) error {
		_, _, err := w.reader.Read(ctx, a.id, rb.RecordID)
		return err
	})
	if !errors.Is(err, interp.ErrNotFound) {
		t.Errorf("A reading B's record: %v, want ErrNotFound", err)
	}
	err = asRun(t, a.id, run, func(ctx context.Context) error {
		return w.writer.Write(ctx, a.id, rb.RecordID, uuid.New(), cause, "k-iso", map[string]any{"summary": "pwned"})
	})
	if !errors.Is(err, interp.ErrNotFound) {
		t.Errorf("A writing B's record: %v, want ErrNotFound", err)
	}
	// A tenant the caller names that is not the verified one is refused.
	err = asRun(t, a.id, run, func(ctx context.Context) error {
		_, _, err := w.reader.Read(ctx, b.id, rb.RecordID)
		return err
	})
	if !errors.Is(err, interp.ErrForbidden) {
		t.Errorf("Read naming tenant B under A's header: %v, want ErrForbidden", err)
	}
	if got := versionOf(t, e, rb.RecordID); got != rb.Version+1 {
		t.Errorf("tenant B's record at version %d, want %d (untouched by A)", got, rb.Version+1)
	}
}

// Security conditions (TICK-88, P3-40 review): the adapter fails closed and
// never falls back to depth 0 when the cause is missing, another tenant's or
// another run's; and refuses a cause older than the run's newest (P3-42
// review, Low).
func TestWorkflowWriter_RefusesBadAndStaleCauses(t *testing.T) {
	e := migrated(t)
	a, b := seedTenant(t, e), seedTenant(t, e)
	w := newWorker(t, e)
	r := mustCreate(t, e, newPipeline(e, nil), a, map[string]any{"title": "printer"})
	run, _ := openRun(t, e, a, r.RecordID, false)
	other, _ := openRun(t, e, a, r.RecordID, false)
	rb := mustCreate(t, e, newPipeline(e, nil), b, map[string]any{"title": "b"})
	runB, _ := openRun(t, e, b, rb.RecordID, false)

	now := time.Now()
	old := seedCauseRow(t, e, a.id, run, uuid.New(), 5, now.Add(-time.Hour))
	newest := seedCauseRow(t, e, a.id, run, uuid.New(), 1, now)
	foreignRun := seedCauseRow(t, e, a.id, other, uuid.New(), 0, now)
	tenantB := seedCauseRow(t, e, b.id, runB, uuid.New(), 0, now)

	try := func(cause uuid.UUID, key string) error {
		return asRun(t, a.id, run, func(ctx context.Context) error {
			return w.writer.Write(ctx, a.id, r.RecordID, uuid.New(), cause, key, map[string]any{"summary": key})
		})
	}
	for name, cause := range map[string]uuid.UUID{
		"tenant B's cause":          tenantB,
		"another run's cause":       foreignRun,
		"no such cause":             uuid.New(),
		"no cause":                  uuid.Nil,
		"a cause older than newest": old, // a replayed delivery id must not reset depth
	} {
		if err := try(cause, "k-"+name); !errors.Is(err, interp.ErrForbidden) {
			t.Errorf("%s: %v, want ErrForbidden", name, err)
		}
	}
	if got := versionOf(t, e, r.RecordID); got != r.Version {
		t.Fatalf("record at version %d after refused causes, want %d (no write, no depth-0 fallback)", got, r.Version)
	}
	if err := try(newest, "k-newest"); err != nil {
		t.Fatalf("the run's newest cause: %v", err)
	}
	hist := historyOf(t, e, a.id, r.RecordID)
	if last := hist[len(hist)-1]; last.Source != "workflow" || last.Depth != 2 {
		t.Errorf("history entry = %+v, want a workflow write at depth 2 (newest cause depth 1 + 1)", last)
	}
}

// A burst of workflow writes beyond the depth limit against worker pools of
// the configured size: the pools are not exhausted (the pipeline's runner has
// a pool of its own), every write gets its cascade_too_deep entry, and the
// pool recovers.
func TestWorkflowWriteBurstDoesNotExhaustThePool(t *testing.T) {
	e := migrated(t)
	f := seedTenant(t, e)
	w := newWorker(t, e)
	const writes = 24 // six times the worker pool's width
	type job struct{ record, run, cause uuid.UUID }
	jobs := make([]job, writes)
	for i := range jobs {
		r := mustCreate(t, e, newPipeline(e, nil), f, map[string]any{"title": "deep"})
		run, _ := openRun(t, e, f, r.RecordID, false)
		// A cause at the depth limit: the write is one deeper, over it.
		jobs[i] = job{r.RecordID, run, seedCauseRow(t, e, f.id, run, uuid.New(), int16(config.CascadeDepthLimit), time.Now())}
	}

	var wg sync.WaitGroup
	errs := make([]error, writes)
	for i, j := range jobs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs[i] = asRun(t, f.id, j.run, func(ctx context.Context) error {
				return w.writer.Write(ctx, f.id, j.record, uuid.New(), j.cause, "burst", map[string]any{"summary": "x"})
			})
		}()
	}
	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Minute):
		t.Fatal("the burst did not finish: a pool is exhausted")
	}
	for i, err := range errs {
		if !errors.Is(err, interp.ErrForbidden) {
			t.Errorf("write %d: %v, want the refusal (ErrForbidden)", i, err)
		}
	}
	if n := count(t, e.super, `SELECT count(*) FROM audit_log WHERE tenant_id = $1 AND action = $2`, f.id, tooDeepAction); n != writes {
		t.Errorf("%d cascade_too_deep entries, want %d (one per write)", n, writes)
	}

	// The pools recover: nothing is held, and a write at a legal depth works.
	if n := w.pool.Stat().AcquiredConns() + w.pipePool.Stat().AcquiredConns(); n != 0 {
		t.Errorf("%d connections still acquired after the burst", n)
	}
	r := mustCreate(t, e, newPipeline(e, nil), f, map[string]any{"title": "after"})
	run, _ := openRun(t, e, f, r.RecordID, false)
	cause := seedCauseRow(t, e, f.id, run, uuid.New(), 0, time.Now())
	if err := asRun(t, f.id, run, func(ctx context.Context) error {
		return w.writer.Write(ctx, f.id, r.RecordID, uuid.New(), cause, "after", map[string]any{"status": "closed"})
	}); err != nil {
		t.Errorf("write after the burst: %v", err)
	}
}

// No lost record_changed wakeup (contract amendment, 2026-11-13): a write that
// commits between the walk's read and the wait's UPDATE is never missed. The
// wait is refused as ErrRecordMoved (NextStep re-walks) when the record is no
// longer at the version the walk read, and set when it is.
func TestWaitTracker_NeverWaitsOnAStaleVersion(t *testing.T) {
	e := migrated(t)
	f := seedTenant(t, e)
	w := newWorker(t, e)
	p := newPipeline(e, eventsink.Sink{})
	r := mustCreate(t, e, p, f, map[string]any{"title": "printer"})
	run, _ := openRun(t, e, f, r.RecordID, false)
	signal := uuid.New()
	set := func(version int64) error {
		return asRun(t, f.id, run, func(ctx context.Context) error {
			return w.waits.SetRecordChangedWait(ctx, f.id, r.RecordID, &signal, version)
		})
	}
	waitOf := func() *uuid.UUID {
		var s *uuid.UUID
		if err := e.super.QueryRow(context.Background(), `SELECT record_changed_signal_id FROM workflow_ref
			WHERE tenant_id = $1 AND outbox_id = $2`, f.id, run).Scan(&s); err != nil {
			t.Fatal(err)
		}
		return s
	}

	// The walk read version r.Version; a write commits before the wait is set.
	if _, err := write(context.Background(), e, p, f.id, updateCmd(f, r.RecordID, r.Version, map[string]any{"summary": "moved"})); err != nil {
		t.Fatal(err)
	}
	if err := set(r.Version); !errors.Is(err, workerguard.ErrRecordMoved) {
		t.Fatalf("set on a stale version: %v, want ErrRecordMoved", err)
	}
	if s := waitOf(); s != nil {
		t.Fatalf("run waits on %s after a refused set, want none", s)
	}
	// The re-walk reads the new version and the wait is set.
	if err := set(r.Version + 1); err != nil {
		t.Fatalf("set on the current version: %v", err)
	}
	if s := waitOf(); s == nil || *s != signal {
		t.Fatalf("wait = %v, want %s", s, signal)
	}
	// A clear needs no check, even at a stale version.
	if err := asRun(t, f.id, run, func(ctx context.Context) error {
		return w.waits.SetRecordChangedWait(ctx, f.id, r.RecordID, nil, 1)
	}); err != nil || waitOf() != nil {
		t.Fatalf("clear: %v, wait %v", err, waitOf())
	}
}

// The race itself: a write holds the record's lock (FOR UPDATE until commit)
// while the wait is set. The set blocks behind the write, then sees the new
// version and is refused; the run is never left waiting on the version the
// walk read.
func TestWaitTracker_SetBlocksBehindAnInFlightWrite(t *testing.T) {
	e := migrated(t)
	f := seedTenant(t, e)
	w := newWorker(t, e)
	p := newPipeline(e, eventsink.Sink{})
	r := mustCreate(t, e, p, f, map[string]any{"title": "printer"})
	run, _ := openRun(t, e, f, r.RecordID, false)
	signal := uuid.New()

	ctx := context.Background()
	inFlight := make(chan struct{})
	commit := make(chan struct{})
	writeDone := make(chan error, 1)
	go func() {
		writeDone <- tenancy.InTenantTx(ctx, e.app, f.id, func(tx pgx.Tx) error {
			if _, err := p.Write(ctx, tx, updateCmd(f, r.RecordID, r.Version, map[string]any{"summary": "in flight"})); err != nil {
				return err
			}
			close(inFlight)
			<-commit
			return nil
		})
	}()
	<-inFlight

	res := make(chan error, 1)
	go func() {
		res <- asRun(t, f.id, run, func(ctx context.Context) error {
			return w.waits.SetRecordChangedWait(ctx, f.id, r.RecordID, &signal, r.Version)
		})
	}()
	deadline := time.Now().Add(15 * time.Second)
	for count(t, e.super, `SELECT count(*) FROM pg_stat_activity WHERE usename = 'nava_worker' AND wait_event_type = 'Lock'`) == 0 {
		if time.Now().After(deadline) {
			close(commit)
			t.Fatal("the wait never blocked behind the write")
		}
		time.Sleep(10 * time.Millisecond)
	}
	close(commit)
	if err := <-writeDone; err != nil {
		t.Fatal(err)
	}
	if err := <-res; !errors.Is(err, workerguard.ErrRecordMoved) {
		t.Fatalf("set after the write committed: %v, want ErrRecordMoved", err)
	}
	if n := count(t, e.super, `SELECT count(*) FROM workflow_ref WHERE tenant_id = $1 AND outbox_id = $2 AND record_changed_signal_id IS NOT NULL`, f.id, run); n != 0 {
		t.Errorf("run waits on a stale version")
	}
}

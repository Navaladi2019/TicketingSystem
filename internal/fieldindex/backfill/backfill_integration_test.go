//go:build integration

package backfill_test

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"

	"github.com/Navaladi2019/TicketingSystem/internal/fieldindex"
	"github.com/Navaladi2019/TicketingSystem/internal/fieldindex/backfill"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
	"github.com/Navaladi2019/TicketingSystem/internal/workerguard"
)

// openWriter is a nava_app transaction that has inserted one record of f
// (its trigger has run) and has not committed: the R1 writer caught across
// the flip. It runs in tenancy.InTenantTx on its own goroutine until commit.
type openWriter struct {
	id      uuid.UUID
	xid     string // its transaction id
	release chan struct{}
	done    chan error
	once    sync.Once
}

func beginWriter(t *testing.T, e itEnv, f fixture) *openWriter {
	t.Helper()
	w := &openWriter{id: uuid.Must(uuid.NewV7()), release: make(chan struct{}), done: make(chan error, 1)}
	ready := make(chan error, 1)
	go func() {
		ctx := context.Background()
		w.done <- tenancy.InTenantTx(ctx, e.app, f.tenant, func(tx pgx.Tx) error {
			_, err := tx.Exec(ctx, insertRecordSQL, f.tenant, w.id, f.def, "INC-OPEN", recordData(f, 7777))
			if err == nil {
				err = tx.QueryRow(ctx, "SELECT pg_current_xact_id()::text").Scan(&w.xid)
			}
			ready <- err
			if err != nil {
				return err
			}
			<-w.release
			return nil
		})
	}()
	if err := <-ready; err != nil {
		t.Fatal(err)
	}
	return w
}

// commit lets the transaction commit and waits for it.
func (w *openWriter) commit(t *testing.T) {
	t.Helper()
	w.once.Do(func() {
		close(w.release)
		if err := <-w.done; err != nil {
			t.Error(err)
		}
	})
}

// CI-5 R1 (deterministic half): a record inserted before the flip to
// backfilling and committed only after the backfill has passed its id is
// copied, because the barrier waits for its transaction. With the barrier
// replaced by a no-op (the seam), the same schedule leaves it missing: the
// test bites.
func TestR1WriterAcrossTheFlip(t *testing.T) {
	e := migrated(t)
	for _, barrierOn := range []bool{true, false} {
		name := map[bool]string{true: "barrier", false: "no barrier"}[barrierOn]
		t.Run(name, func(t *testing.T) {
			f := newFixture(t, e, "num", "")
			insertRecords(t, e, f, 5)
			w := beginWriter(t, e, f) // its trigger saw no registry row
			f.register(t, e, "num", fieldindex.StateBackfilling)

			var committed atomic.Bool
			commit := func() {
				if committed.CompareAndSwap(false, true) {
					w.commit(t)
				}
			}
			o := runOptions{acts: &backfill.Activities{Pool: e.worker}}
			if barrierOn {
				// The barrier waits on w: commit it later, as a slow writer would.
				go func() { time.Sleep(1500 * time.Millisecond); commit() }()
			} else {
				// The seam: the backfill passes w's id, and flips ready, before
				// w commits.
				o.barrier = func(context.Context) error { return nil }
			}
			r := newRun(t, f.tenant, o)
			r.signal(item(f, 1), 0)
			if err := r.coordinate(workerguard.RunInput{}); err != nil {
				t.Fatal(err)
			}
			commit()
			r.requireGuardClean()

			if got := registry(t, e, f); got.state != "ready" || got.progress != 100 {
				t.Fatalf("registry %+v, want ready at 100", got)
			}
			got := diffOf(t, e, f)
			switch {
			case barrierOn && got != noDiff:
				t.Fatalf("with the barrier: %s, want %s", got, noDiff)
			case !barrierOn && got != "1 missing, 0 extra":
				t.Fatalf("without the barrier: %s, want the R1 miss (1 missing): the test does not bite", got)
			}
		})
	}
}

// CI-5 R1 (load half): writers insert and update records continuously
// across the flip to backfilling and the flip to ready; after ready the side
// rows equal a recompute, with none missing.
func TestR1WritersAcrossBothFlips(t *testing.T) {
	e := migrated(t)
	f := newFixture(t, e, "num", "")
	ids := insertRecords(t, e, f, 120)

	ctx, stop := context.WithCancel(context.Background())
	var (
		wg     sync.WaitGroup
		mu     sync.Mutex
		live   = slices.Clone(ids)
		writes atomic.Int64
		errs   = make(chan error, 4)
	)
	for wi := range 3 {
		wg.Go(func() {
			for n := 0; ctx.Err() == nil; n++ {
				var err error
				if n%2 == 0 {
					id := uuid.Must(uuid.NewV7())
					err = writeRecord(ctx, e, f, id, 10_000*(wi+1)+n)
					if err == nil {
						mu.Lock()
						live = append(live, id)
						mu.Unlock()
					}
				} else {
					mu.Lock()
					id := live[(n*7+wi)%len(live)]
					mu.Unlock()
					err = tenancy.InTenantTx(ctx, e.app, f.tenant, func(tx pgx.Tx) error {
						_, err := tx.Exec(ctx, updateRecordSQL, f.tenant, id, n, f.key)
						return err
					})
				}
				if err != nil && ctx.Err() == nil {
					errs <- err
					return
				}
				writes.Add(1)
			}
		})
	}
	// Let the writers run before the flip, flip, and keep them running
	// through the backfill and the flip to ready.
	time.Sleep(300 * time.Millisecond)
	f.register(t, e, "num", fieldindex.StateBackfilling)
	before := writes.Load()

	r := newRun(t, f.tenant, runOptions{acts: &backfill.Activities{Pool: e.worker, BatchSize: 25}})
	r.signal(item(f, 1), 0)
	err := r.coordinate(workerguard.RunInput{})
	time.Sleep(300 * time.Millisecond) // writes after ready
	stop()
	wg.Wait()
	close(errs)
	for werr := range errs {
		t.Fatalf("writer: %v", werr)
	}
	if err != nil {
		t.Fatal(err)
	}
	r.requireGuardClean()
	if writes.Load()-before < 10 {
		t.Fatalf("only %d writes overlapped the backfill: the race was not exercised", writes.Load()-before)
	}
	if got := registry(t, e, f); got.state != "ready" || got.progress != 100 {
		t.Fatalf("registry %+v, want ready at 100", got)
	}
	if got := diffOf(t, e, f); got != noDiff {
		t.Fatalf("side rows differ from the recompute: %s", got)
	}
	if r.batches[backfillChild(f, 1)] < 5 {
		t.Errorf("%d batches, want several (batch size 25)", r.batches[backfillChild(f, 1)])
	}
}

// Barrier: a transaction open at the snapshot delays ready until it ends,
// a transaction opened after the snapshot does not hold it, and the slow
// alert fires once per waiting barrier under a shortened threshold.
func TestBarrierWaitsForTheSnapshotOnly(t *testing.T) {
	e := migrated(t)
	f := newFixture(t, e, "num", "")
	insertRecords(t, e, f, 3)
	f.register(t, e, "num", fieldindex.StateBackfilling)
	held := beginWriter(t, e, newFixture(t, e, "num", "")) // another tenant's long transaction

	reader := sdkmetric.NewManualReader()
	m, err := backfill.NewMetrics(sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader)))
	if err != nil {
		t.Fatal(err)
	}
	var (
		later               *openWriter
		readyWhileLaterOpen atomic.Bool
		done                = make(chan struct{})
	)
	go func() {
		// After the first barrier's snapshot: a transaction that stays open
		// until the field is ready. It must not hold that barrier.
		time.Sleep(500 * time.Millisecond)
		later = beginWriter(t, e, newFixture(t, e, "num", ""))
		time.Sleep(time.Second)
		held.commit(t)
		for {
			select {
			case <-done:
				return
			case <-time.After(100 * time.Millisecond):
			}
			if registry(t, e, f).state == "ready" {
				// The second barrier (after ready) does wait on it.
				readyWhileLaterOpen.Store(true)
				later.commit(t)
				return
			}
		}
	}()
	acts := &backfill.Activities{Pool: e.worker, Metrics: m, SlowAfter: time.Millisecond}
	r := newRun(t, f.tenant, runOptions{acts: acts})
	r.signal(item(f, 1), 0)
	start := time.Now()
	err = r.coordinate(workerguard.RunInput{})
	close(done)
	if err != nil {
		t.Fatal(err)
	}
	r.requireGuardClean()
	if time.Since(start) < 1500*time.Millisecond {
		t.Error("ready came before the held transaction ended")
	}
	if !readyWhileLaterOpen.Load() {
		t.Fatal("the field was not ready while the later transaction was open: the barrier waited on a transaction outside its snapshot")
	}
	if got := registry(t, e, f); got.state != "ready" {
		t.Fatalf("registry %+v, want ready", got)
	}

	var rm metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &rm); err != nil {
		t.Fatal(err)
	}
	var slow int64
	for _, sm := range rm.ScopeMetrics {
		for _, mt := range sm.Metrics {
			if mt.Name == "field_index_barrier_slow" {
				for _, dp := range mt.Data.(metricdata.Sum[int64]).DataPoints {
					slow += dp.Value
					if dp.Attributes.Len() != 0 {
						t.Errorf("field_index_barrier_slow has attributes %v, want none", dp.Attributes)
					}
				}
			}
		}
	}
	// The first barrier waited past the 1 ms threshold for seconds, across
	// several polls: it fired once. The second may or may not have caught the
	// later transaction still open. Never more than once per barrier.
	if slow < 1 || slow > 2 {
		t.Errorf("field_index_barrier_slow = %d, want 1 or 2 (once per waiting barrier, not per poll)", slow)
	}
}

// A registry row set to removing mid-backfill stops the workflow at the next
// batch with no further writes, and it is not flipped to ready.
func TestRemovingMidBackfillStops(t *testing.T) {
	e := migrated(t)
	f := newFixture(t, e, "num", "")
	insertRecords(t, e, f, 50)
	f.register(t, e, "num", fieldindex.StateBackfilling)
	var flipped atomic.Bool
	var after registryState
	r := newRun(t, f.tenant, runOptions{acts: &backfill.Activities{Pool: e.worker, BatchSize: 10},
		onBatch: func(string) {
			if flipped.CompareAndSwap(false, true) {
				mustExec(t, e.super, `UPDATE record_index_fields SET state = 'removing' WHERE tenant_id = $1 AND id = $2`, f.tenant, f.field)
				after = registry(t, e, f)
			}
		}})
	r.signal(item(f, 1), 0)
	if err := r.coordinate(workerguard.RunInput{}); err != nil {
		t.Fatal(err)
	}
	r.requireGuardClean()
	if got := sideRows(t, e, f); got != "10" {
		t.Errorf("%s side rows, want the first batch's 10 only", got)
	}
	got := registry(t, e, f)
	if got.state != "removing" || got.progress != after.progress || !got.updatedAt.Equal(after.updatedAt) {
		t.Errorf("registry %+v after the flip, want it untouched since %+v", got, after)
	}
	if n := r.batches[backfillChild(f, 1)]; n != 2 {
		t.Errorf("%d batches, want 2 (one copied, one found removing)", n)
	}
}

// QA item 3: requested_version bumped while a child runs. The running child
// (pinned to version 1) stops at its next batch and writes no progress for
// the old version; the version-2 item then runs the field to ready.
func TestVersionBumpMidBackfill(t *testing.T) {
	e := migrated(t)
	f := newFixture(t, e, "num", "")
	insertRecords(t, e, f, 50)
	f.register(t, e, "num", fieldindex.StateBackfilling)
	v1, v2 := backfillChild(f, 1), backfillChild(f, 2)
	var bumped atomic.Bool
	var atBump registryState
	r := newRun(t, f.tenant, runOptions{acts: &backfill.Activities{Pool: e.worker, BatchSize: 10},
		onBatch: func(id string) {
			switch {
			case id == v1 && bumped.CompareAndSwap(false, true):
				mustExec(t, e.super, `UPDATE record_index_fields SET requested_version = 2 WHERE tenant_id = $1 AND id = $2`, f.tenant, f.field)
				atBump = registry(t, e, f)
			}
		}})
	r.signal(item(f, 1), 0)
	r.signal(item(f, 2), time.Millisecond)
	if err := r.coordinate(workerguard.RunInput{}); err != nil {
		t.Fatal(err)
	}
	r.requireGuardClean()
	if n := r.batches[v1]; n != 2 {
		t.Errorf("the version-1 child ran %d batches, want 2 (one copied, one found the bump)", n)
	}
	// The version-1 child's second batch wrote nothing: its only progress
	// write was before the bump, so the row's progress at the bump is the
	// first batch's, and the v2 child then wrote its own.
	if atBump.progress != 20 {
		t.Errorf("progress at the bump %d, want the first batch's 20", atBump.progress)
	}
	if got := registry(t, e, f); got.state != "ready" || got.version != 2 {
		t.Fatalf("registry %+v, want ready at version 2", got)
	}
	if got := diffOf(t, e, f); got != noDiff {
		t.Fatalf("side rows: %s", got)
	}
	log := r.childLog()
	if i, j := slices.Index(log, "done "+v1), slices.Index(log, "start "+v2); i < 0 || j < 0 || i > j {
		t.Errorf("children %v: the version-2 child must start after the version-1 child ends", log)
	}
}

// Coordinator: field A flagged, then field B while A runs. Both reach ready,
// B's child starts only after A's finishes, and a replay of A's outbox row
// (same outbox id) runs nothing.
func TestCoordinatorRunsOneChildAtATime(t *testing.T) {
	e := migrated(t)
	a := newFixture(t, e, "num", "")
	b := fixture{tenant: a.tenant, def: a.def, field: uuid.New(), key: "f_text"}
	insertRecords(t, e, a, 40)
	a.register(t, e, "num", fieldindex.StateBackfilling)
	b.register(t, e, "text", fieldindex.StateBackfilling)
	r := newRun(t, a.tenant, runOptions{acts: &backfill.Activities{Pool: e.worker, BatchSize: 10}})
	itA := item(a, 1)
	r.signal(itA, 0)
	// A runs four batches with 500 ms pauses (test time): B and the replay
	// arrive while A's child is running.
	r.signal(item(b, 1), 700*time.Millisecond)
	r.signal(itA, 1200*time.Millisecond) // the relay delivering A's row again
	if err := r.coordinate(workerguard.RunInput{}); err != nil {
		t.Fatal(err)
	}
	r.requireGuardClean()
	want := []string{"start " + backfillChild(a, 1), "done " + backfillChild(a, 1), "start " + backfillChild(b, 1), "done " + backfillChild(b, 1)}
	if got := r.childLog(); !slices.Equal(got, want) {
		t.Fatalf("children %v, want %v", got, want)
	}
	for _, f := range []fixture{a, b} {
		if got := registry(t, e, f); got.state != "ready" {
			t.Errorf("field %s: %+v, want ready", f.key, got)
		}
		if got := diffOf(t, e, f); got != noDiff {
			t.Errorf("field %s: %s", f.key, got)
		}
	}
}

// QA item 1, and the replay race: a replay of a finished item from outside
// the coordinator's Seen window (a new coordinator run, empty Seen) runs no
// child; and a backfill child run on a ready row writes no batch and no
// progress.
func TestStaleReplayIsANoOp(t *testing.T) {
	e := migrated(t)
	f := newFixture(t, e, "num", "")
	insertRecords(t, e, f, 15)
	f.register(t, e, "num", fieldindex.StateBackfilling)
	it := item(f, 1)
	first := newRun(t, f.tenant, runOptions{acts: &backfill.Activities{Pool: e.worker}})
	first.signal(it, 0)
	if err := first.coordinate(workerguard.RunInput{}); err != nil {
		t.Fatal(err)
	}
	done := registry(t, e, f)
	if done.state != "ready" {
		t.Fatalf("registry %+v, want ready", done)
	}
	ctids := queryString(t, e.super, `SELECT string_agg(ctid::text, ',' ORDER BY record_id) FROM rfv_num WHERE tenant_id = $1 AND field_id = $2`, f.tenant, f.field)

	replay := newRun(t, f.tenant, runOptions{acts: &backfill.Activities{Pool: e.worker}})
	replay.signal(it, 0)
	if err := replay.coordinate(workerguard.RunInput{}); err != nil {
		t.Fatal(err)
	}
	replay.requireGuardClean()
	if got := replay.childLog(); len(got) != 0 {
		t.Errorf("the stale replay started children %v, want none", got)
	}

	// The child itself, on the ready row, as a defence in depth.
	direct := newRun(t, f.tenant, runOptions{acts: &backfill.Activities{Pool: e.worker}, workflowID: backfillChild(f, 1)})
	direct.env.ExecuteWorkflow(fieldindex.BackfillWorkflowType, backfill.Input{FieldID: f.field})
	if err := direct.env.GetWorkflowError(); err != nil {
		t.Fatal(err)
	}
	direct.requireGuardClean()
	if len(direct.results) != 1 || direct.results[0].Cursor != nil {
		t.Errorf("batch results %v, want one empty result", direct.results)
	}
	if got := registry(t, e, f); got != done {
		t.Errorf("registry %+v, want it untouched (%+v)", got, done)
	}
	if got := queryString(t, e.super, `SELECT string_agg(ctid::text, ',' ORDER BY record_id) FROM rfv_num WHERE tenant_id = $1 AND field_id = $2`, f.tenant, f.field); got != ctids {
		t.Error("the replay rewrote side rows")
	}
}

// QA item 2, and "continue-as-new with a queued item loses nothing": with
// one item per run, a coordinator queued with A, B and C continues as new
// after A carrying [B, C] in order. B is set to removing between the runs, so
// the continued run classifies it as a removal; C is backfilled.
func TestContinueAsNewKeepsQueueOrderAndReadsOp(t *testing.T) {
	e := migrated(t)
	a := newFixture(t, e, "num", "")
	b := fixture{tenant: a.tenant, def: a.def, field: uuid.New(), key: "f_b"}
	c := fixture{tenant: a.tenant, def: a.def, field: uuid.New(), key: "f_c"}
	insertRecords(t, e, a, 5)
	a.register(t, e, "num", fieldindex.StateBackfilling)
	b.register(t, e, "num", fieldindex.StateBackfilling)
	c.register(t, e, "num", fieldindex.StateBackfilling)

	coord := &backfill.Coordinator{MaxItems: 1}
	var removed []uuid.UUID
	remove := func(_ workflow.Context, in backfill.FieldRef) error {
		removed = append(removed, in.FieldID)
		return nil
	}
	first := newRun(t, a.tenant, runOptions{acts: &backfill.Activities{Pool: e.worker}, coordinator: coord, remove: remove})
	first.signal(item(a, 1), 0)
	first.signal(item(b, 1), 0)
	first.signal(item(c, 1), 0)
	err := first.coordinate(workerguard.RunInput{})
	var can *workflow.ContinueAsNewError
	if !errors.As(err, &can) {
		t.Fatalf("first run: %v, want continue-as-new", err)
	}
	first.requireGuardClean()
	var next workerguard.RunInput
	if err := json.Unmarshal(can.Input.GetPayloads()[0].GetData(), &next); err != nil {
		t.Fatal(err)
	}
	var refs struct {
		Pending []uuid.UUID `json:"pending"`
	}
	if err := json.Unmarshal(next.Refs, &refs); err != nil || !slices.Equal(refs.Pending, []uuid.UUID{b.field, c.field}) {
		t.Fatalf("continued refs %s (%v), want pending [B C]", next.Refs, err)
	}
	if next.Seen.Len() != 3 {
		t.Errorf("Seen carries %d outbox ids, want 3", next.Seen.Len())
	}
	mustExec(t, e.super, `UPDATE record_index_fields SET state = 'removing' WHERE tenant_id = $1 AND id = $2`, b.tenant, b.field)

	second := newRun(t, a.tenant, runOptions{acts: &backfill.Activities{Pool: e.worker}, coordinator: &backfill.Coordinator{MaxItems: 10}, remove: remove})
	if err := second.coordinate(next); err != nil {
		t.Fatal(err)
	}
	second.requireGuardClean()
	removeB := backfill.ChildID(b.tenant, fieldindex.RemoveWorkflowType, b.field, 0)
	want := []string{"start " + removeB, "done " + removeB, "start " + backfillChild(c, 0), "done " + backfillChild(c, 0)}
	if got := second.childLog(); !slices.Equal(got, want) {
		t.Fatalf("continued children %v, want %v", got, want)
	}
	if !slices.Equal(removed, []uuid.UUID{b.field}) {
		t.Errorf("removed %v, want [B]", removed)
	}
	if got := registry(t, e, c); got.state != "ready" {
		t.Errorf("C: %+v, want ready", got)
	}
	// A replay of A's row in the continued run is dropped on its outbox id.
	if got := registry(t, e, a); got.state != "ready" {
		t.Errorf("A: %+v, want ready", got)
	}
}

// CI-5c: no record id beyond the cursor, and no field value, appears in any
// payload the runs recorded: every argument, result, child input, heartbeat
// and signal (the test environment's view of the history).
func TestHistoryCarriesReferencesOnly(t *testing.T) {
	e := migrated(t)
	f := newFixture(t, e, "num", "")
	ids := insertRecords(t, e, f, 35)
	f.register(t, e, "num", fieldindex.StateBackfilling)
	r := newRun(t, f.tenant, runOptions{acts: &backfill.Activities{Pool: e.worker, BatchSize: 10}})
	r.signal(item(f, 1), 0)
	if err := r.coordinate(workerguard.RunInput{}); err != nil {
		t.Fatal(err)
	}
	r.requireGuardClean()

	cursors := map[string]bool{}
	for _, c := range r.results {
		if c.Cursor != nil {
			cursors[c.Cursor.String()] = true
		}
	}
	records := map[string]bool{}
	for _, id := range ids {
		records[id.String()] = true
	}
	uuidRE := regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.ledger) < 10 {
		t.Fatalf("only %d payloads recorded: the ledger missed the run", len(r.ledger))
	}
	for _, p := range r.ledger {
		s := string(p)
		if strings.Contains(s, valueSecret) || strings.Contains(s, "INC-") {
			t.Errorf("payload %s carries a field value", s)
		}
		for _, id := range uuidRE.FindAllString(s, -1) {
			if records[id] && !cursors[id] {
				t.Errorf("payload %s carries record id %s, which is not a cursor", s, id)
			}
		}
	}
	if len(cursors) != 3 {
		t.Errorf("%d cursors, want 3 (35 records, batches of 10)", len(cursors))
	}
}

// A retried barrier resumes the snapshot in its heartbeat details, never a
// new one: given details that name only a finished transaction it returns at
// once although another transaction is open now, and given details that name
// an open one, in the xip or in the [lo, hi) range, it keeps waiting. Details
// without lo and hi (written before ADR-006-E1 E1-1) wait on the xip alone.
func TestBarrierResumesItsSnapshot(t *testing.T) {
	e := migrated(t)
	open := beginWriter(t, e, newFixture(t, e, "num", ""))
	defer open.commit(t)
	openXID := open.xid
	var finishedXID string
	if err := e.super.QueryRow(context.Background(), "SELECT pg_current_xact_id()::text").Scan(&finishedXID); err != nil {
		t.Fatal(err)
	}
	acts := &backfill.Activities{Pool: e.worker}
	run := func(details map[string]any, timeout time.Duration) error {
		var s testsuite.WorkflowTestSuite
		env := s.NewTestActivityEnvironment()
		env.RegisterActivityWithOptions(acts.Barrier, activity.RegisterOptions{Name: backfill.BarrierActivity})
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		env.SetTestTimeout(timeout)
		env.SetWorkerOptions(worker.Options{BackgroundActivityContext: ctx})
		details["since"], details["slow"] = time.Now(), false
		env.SetHeartbeatDetails(details)
		_, err := env.ExecuteActivity(backfill.BarrierActivity)
		return err
	}
	next := func(xid string) string { return queryString(t, e.super, `SELECT ($1::bigint + 1)::text`, xid) }
	for name, details := range map[string]map[string]any{
		"xip only":       {"xip": []string{finishedXID}},
		"range finished": {"xip": []string{}, "lo": finishedXID, "hi": next(finishedXID)},
	} {
		if err := run(details, 10*time.Second); err != nil {
			t.Fatalf("%s: resumed snapshot of a finished transaction: %v, want done at once", name, err)
		}
	}
	for name, details := range map[string]map[string]any{
		"xip only":   {"xip": []string{openXID}},
		"range open": {"xip": []string{}, "lo": openXID, "hi": next(openXID)},
	} {
		if err := run(details, 2500*time.Millisecond); err == nil {
			t.Fatalf("%s: resumed snapshot of an open transaction returned; want it to keep waiting", name)
		}
	}
}

// Replay race: a batch whose activity fails after its transaction committed
// (a worker lost before reporting) is retried from the same cursor; the
// retry rewrites the same records, and the field still ends equal to the
// recompute.
func TestBatchRetriedAfterCommit(t *testing.T) {
	e := migrated(t)
	f := newFixture(t, e, "num", "")
	insertRecords(t, e, f, 30)
	f.register(t, e, "num", fieldindex.StateBackfilling)
	acts := &backfill.Activities{Pool: e.worker, BatchSize: 10}
	var calls atomic.Int64
	r := newRun(t, f.tenant, runOptions{acts: acts})
	r.env.RegisterActivityWithOptions(func(ctx context.Context, in backfill.Cursor) (backfill.Cursor, error) {
		out, err := acts.Batch(ctx, in)
		if err == nil && calls.Add(1) == 2 {
			return backfill.Cursor{}, errors.New("worker lost after the batch committed")
		}
		return out, err
	}, activity.RegisterOptions{Name: backfill.BatchActivity})
	r.signal(item(f, 1), 0)
	if err := r.coordinate(workerguard.RunInput{}); err != nil {
		t.Fatal(err)
	}
	if calls.Load() < 4 {
		t.Fatalf("%d batch calls, want the failed one retried", calls.Load())
	}
	if got := registry(t, e, f); got.state != "ready" {
		t.Fatalf("registry %+v, want ready", got)
	}
	if got := diffOf(t, e, f); got != noDiff {
		t.Fatalf("side rows after a retried batch: %s", got)
	}
}

// Security condition 2 (PR #734 review): the coordinator's tenant is its
// workflow id's. An item naming tenant B's field reaches tenant A's
// coordinator: Classify finds no row under A's RLS, so no child runs and
// nothing is written; a child of A started directly with B's field id finds
// no registry row either and writes nothing.
func TestOtherTenantsFieldWritesNothing(t *testing.T) {
	e := migrated(t)
	a := newFixture(t, e, "num", "")
	b := newFixture(t, e, "num", "")
	insertRecords(t, e, b, 12)
	b.register(t, e, "num", fieldindex.StateBackfilling)
	before := registry(t, e, b)

	r := newRun(t, a.tenant, runOptions{acts: &backfill.Activities{Pool: e.worker}})
	r.signal(item(b, 1), 0) // B's field id, to A's coordinator
	if err := r.coordinate(workerguard.RunInput{}); err != nil {
		t.Fatal(err)
	}
	r.requireGuardClean()
	if got := r.childLog(); len(got) != 0 {
		t.Fatalf("A's coordinator started children %v for B's field", got)
	}

	child := backfill.ChildID(a.tenant, fieldindex.BackfillWorkflowType, b.field, 1)
	direct := newRun(t, a.tenant, runOptions{acts: &backfill.Activities{Pool: e.worker}, workflowID: child})
	direct.env.ExecuteWorkflow(fieldindex.BackfillWorkflowType, backfill.Input{FieldID: b.field})
	if err := direct.env.GetWorkflowError(); err != nil {
		t.Fatal(err)
	}
	direct.requireGuardClean()
	if len(direct.results) != 1 || direct.results[0].Cursor != nil {
		t.Errorf("batch results %v, want one empty result", direct.results)
	}
	if got := registry(t, e, b); got != before {
		t.Errorf("B's registry row changed: %+v, was %+v", got, before)
	}
	if got := sideRows(t, e, b); got != "0" {
		t.Errorf("B's field has %s side rows, want 0", got)
	}
	for _, f := range []fixture{a, b} {
		if n := queryString(t, e.super, `SELECT count(*)::text FROM rfv_num WHERE tenant_id = $1`, f.tenant); n != "0" {
			t.Errorf("tenant %s has %s side rows, want 0", f.tenant, n)
		}
	}
}

// A removal during the barrier's wait ends it at the next poll, not when the
// long transaction ends; the backfill then writes nothing.
func TestRemovalDuringBarrierWaitEndsIt(t *testing.T) {
	e := migrated(t)
	f := newFixture(t, e, "num", "")
	insertRecords(t, e, f, 5)
	f.register(t, e, "num", fieldindex.StateBackfilling)
	held := beginWriter(t, e, newFixture(t, e, "num", ""))
	defer held.commit(t)
	go func() {
		time.Sleep(500 * time.Millisecond)
		mustExec(t, e.super, `UPDATE record_index_fields SET state = 'removing' WHERE tenant_id = $1 AND id = $2`, f.tenant, f.field)
	}()
	r := newRun(t, f.tenant, runOptions{acts: &backfill.Activities{Pool: e.worker}})
	r.signal(item(f, 1), 0)
	start := time.Now()
	if err := r.coordinate(workerguard.RunInput{}); err != nil {
		t.Fatal(err)
	}
	r.requireGuardClean()
	if waited := time.Since(start); waited > 8*time.Second {
		t.Errorf("the barrier waited %v after the removal; want the next poll (at most 2 s)", waited)
	}
	if got := sideRows(t, e, f); got != "0" {
		t.Errorf("%s side rows, want 0", got)
	}
	if got := registry(t, e, f); got.state != "removing" || got.progress != 0 {
		t.Errorf("registry %+v, want removing and untouched", got)
	}
}

// A cancel during the wait ends the barrier promptly with the cancel error.
func TestBarrierCancelEndsTheWait(t *testing.T) {
	e := migrated(t)
	held := beginWriter(t, e, newFixture(t, e, "num", ""))
	defer held.commit(t)
	var s testsuite.WorkflowTestSuite
	env := s.NewTestActivityEnvironment()
	acts := &backfill.Activities{Pool: e.worker}
	env.RegisterActivityWithOptions(acts.Barrier, activity.RegisterOptions{Name: backfill.BarrierActivity})
	ctx, cancel := context.WithCancel(context.Background())
	env.SetWorkerOptions(worker.Options{BackgroundActivityContext: ctx})
	go func() { time.Sleep(1500 * time.Millisecond); cancel() }()
	start := time.Now()
	if _, err := env.ExecuteActivity(backfill.BarrierActivity); err == nil {
		t.Fatal("the barrier returned without error while its snapshot's transaction is open")
	}
	if waited := time.Since(start); waited > 4*time.Second {
		t.Errorf("the barrier ended %v after start; want soon after the cancel at 1.5 s", waited)
	}
}

// heartbeat is one barrier heartbeat as the test reads it: the raw JSON
// details and the wait-set bounds in them.
type heartbeat struct {
	raw []byte
	XIP []string `json:"xip"`
	Lo  string   `json:"lo"`
	Hi  string   `json:"hi"`
}

// barrierEnv runs acts.Barrier in a test activity environment under ctx and
// sends every heartbeat it records to the returned channel (buffered; a
// heartbeat is dropped when it is full).
func barrierEnv(ctx context.Context, t *testing.T, acts *backfill.Activities) (*testsuite.TestActivityEnvironment, <-chan heartbeat) {
	t.Helper()
	var s testsuite.WorkflowTestSuite
	env := s.NewTestActivityEnvironment()
	env.RegisterActivityWithOptions(acts.Barrier, activity.RegisterOptions{Name: backfill.BarrierActivity})
	env.SetWorkerOptions(worker.Options{BackgroundActivityContext: ctx})
	hbs := make(chan heartbeat, 64)
	env.SetOnActivityHeartbeatListener(func(_ *activity.Info, details converter.EncodedValues) {
		var raw json.RawMessage
		if err := details.Get(&raw); err != nil {
			t.Errorf("heartbeat details: %v", err)
			return
		}
		hb := heartbeat{raw: raw}
		if err := json.Unmarshal(raw, &hb); err != nil {
			t.Errorf("heartbeat details %s: %v", raw, err)
			return
		}
		select {
		case hbs <- hb:
		default:
		}
	})
	return env, hbs
}

// ADR-006-E1 (E1-3), the red-to-green evidence: a writer that holds the
// newest assigned xid is the snapshot's xmax and is not in its xip, so a
// barrier that waits on pg_snapshot_xip alone waits on nothing. The barrier
// waits on it (the fence), and returns only after it commits. With the fence
// removed from barrier.takeSnapshot (hi equal to lo) this test fails: the barrier
// returns at once.
//
// The precondition is read from the barrier's own first heartbeat, its lo
// and xip (TICK-980), not from a separate snapshot taken before it: a
// transaction that takes an xid between the two would put the writer in the
// barrier's xip, and the test would then pass without the fence.
func TestBarrierWaitsForTheNewestXID(t *testing.T) {
	e := migrated(t)
	acts := &backfill.Activities{Pool: e.worker}
	var (
		w      *openWriter
		done   chan error
		cancel context.CancelFunc
		first  heartbeat
	)
	for attempt := 0; ; attempt++ {
		w = beginWriter(t, e, newFixture(t, e, "num", ""))
		var ctx context.Context
		ctx, cancel = context.WithCancel(context.Background())
		env, hbs := barrierEnv(ctx, t, acts)
		done = make(chan error, 1)
		go func() {
			_, err := env.ExecuteActivity(backfill.BarrierActivity)
			done <- err
		}()
		select {
		case first = <-hbs:
		case err := <-done:
			t.Fatalf("the barrier returned (%v) before its first heartbeat", err)
		case <-time.After(30 * time.Second):
			t.Fatal("no heartbeat from the barrier")
		}
		if first.Lo == w.xid && !slices.Contains(first.XIP, w.xid) {
			break // the E1 case: the running writer is the barrier's xmax, not in its xip
		}
		cancel()
		<-done
		w.commit(t)
		if attempt == 20 {
			t.Fatalf("could not make the writer the barrier's xmax (lo %s, xip %v, writer %s): another transaction keeps running",
				first.Lo, first.XIP, w.xid)
		}
	}
	defer cancel()
	defer w.commit(t)

	select {
	case err := <-done:
		t.Fatalf("the barrier returned (%v) while the newest-xid writer was still running: it waited on the xip only", err)
	case <-time.After(2500 * time.Millisecond):
	}
	committed := time.Now()
	w.commit(t)
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("barrier: %v", err)
		}
		if waited := time.Since(committed); waited > 12*time.Second {
			t.Errorf("the barrier returned %v after the commit; want the next poll (at most 10 s)", waited)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("the barrier did not return after the writer committed")
	}
}

// ADR-006-E1 E1-1 (TICK-980): the heartbeat keeps the bounds of the wait set,
// not the expanded xid list. Thousands of xids are handed out between the
// barrier's snapshot and its fence, two thousand of them (one transaction and
// its subtransactions) still running: the barrier waits on them (the range is
// checked in SQL), and every heartbeat it records stays a few hundred bytes.
// Before E1-1 the details listed each running xid of the range, about 20 KB
// here.
func TestBarrierHeartbeatKeepsWaitSetBounds(t *testing.T) {
	e := migrated(t)
	ctx := context.Background()
	const (
		burned       = 5000 // committed xids handed out between the snapshot and the fence
		heldSubxacts = 2000 // xids of one transaction still running at the fence
	)
	var (
		held    pgx.Tx
		heldXID string
		hookErr error
	)
	consume := `DO $$ BEGIN FOR i IN 1..` + strconv.Itoa(burned/2) + ` LOOP PERFORM pg_current_xact_id(); COMMIT; END LOOP; END $$`
	acts := &backfill.Activities{Pool: e.worker}
	backfill.SetAfterSnapshot(acts, func(ctx context.Context) {
		if _, hookErr = e.super.Exec(ctx, consume); hookErr != nil {
			return
		}
		if held, hookErr = e.super.Begin(ctx); hookErr != nil {
			return
		}
		if hookErr = held.QueryRow(ctx, "SELECT pg_current_xact_id()::text").Scan(&heldXID); hookErr != nil {
			return
		}
		// Each exception block is a subtransaction whose insert takes an xid
		// of its own: subxids in progress until held ends. An expanded list
		// of the running xids in the range would hold them all.
		if _, hookErr = held.Exec(ctx, `CREATE TEMP TABLE burn (i int) ON COMMIT DROP`); hookErr != nil {
			return
		}
		if _, hookErr = held.Exec(ctx, `DO $$ BEGIN FOR i IN 1..`+strconv.Itoa(heldSubxacts)+` LOOP
			BEGIN INSERT INTO burn VALUES (i); EXCEPTION WHEN division_by_zero THEN NULL; END;
			END LOOP; END $$`); hookErr != nil {
			return
		}
		_, hookErr = e.super.Exec(ctx, consume)
	})
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	env, hbs := barrierEnv(runCtx, t, acts)
	done := make(chan error, 1)
	go func() {
		_, err := env.ExecuteActivity(backfill.BarrierActivity)
		done <- err
	}()
	var first heartbeat
	select {
	case first = <-hbs:
	case err := <-done:
		t.Fatalf("the barrier returned (%v) before its first heartbeat; hook: %v", err, hookErr)
	case <-time.After(60 * time.Second):
		t.Fatal("no heartbeat from the barrier")
	}
	if hookErr != nil {
		t.Fatal(hookErr)
	}
	defer func() { _ = held.Rollback(context.Background()) }()
	width, err := strconv.Atoi(queryString(t, e.super, `SELECT ($2::bigint - $1::bigint)::text`, first.Lo, first.Hi))
	if err != nil {
		t.Fatal(err)
	}
	if width < burned {
		t.Fatalf("heartbeat range [%s, %s) is %d xids wide, want at least %d", first.Lo, first.Hi, width, burned)
	}
	inRange := queryString(t, e.super, `SELECT ($1::bigint <= $2::bigint AND $2::bigint < $3::bigint)::text`, first.Lo, heldXID, first.Hi)
	if inRange != "true" || slices.Contains(first.XIP, heldXID) {
		t.Fatalf("held xid %s: in [%s, %s) %s, in xip %v; want it in the range only", heldXID, first.Lo, first.Hi, inRange, first.XIP)
	}

	const maxDetails = 512 // bytes: the xip, two xids, a timestamp and a flag
	check := func(hb heartbeat) {
		if len(hb.raw) > maxDetails {
			t.Errorf("heartbeat details are %d bytes, want at most %d: %.200s", len(hb.raw), maxDetails, hb.raw)
		}
		if hb.Lo != first.Lo || hb.Hi != first.Hi {
			t.Errorf("heartbeat bounds [%s, %s), want the snapshot's [%s, %s)", hb.Lo, hb.Hi, first.Lo, first.Hi)
		}
	}
	check(first)
	// The barrier keeps waiting on the held xid across polls (1 s, then 2 s).
	// The SDK throttles repeat heartbeats, so a poll's heartbeat may not
	// arrive here; any that does is checked.
	deadline := time.After(3500 * time.Millisecond)
wait:
	for {
		select {
		case hb := <-hbs:
			check(hb)
		case err := <-done:
			t.Fatalf("the barrier returned (%v) while xid %s in its range was running", err, heldXID)
		case <-deadline:
			break wait
		}
	}
	if err := held.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("barrier: %v", err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("the barrier did not return after the held xid committed")
	}
}

// typeCounter is a pgx tracer that counts the statements counting a whole
// entity type's records (the backfill's progress count).
type typeCounter struct{ n atomic.Int64 }

var typeCountSQL = regexp.MustCompile(`(?s)count\(\*\).*FROM records\s+WHERE tenant_id = \$1 AND entity_definition_id = \$2\s*$`)

func (c *typeCounter) TraceQueryStart(ctx context.Context, _ *pgx.Conn, d pgx.TraceQueryStartData) context.Context {
	if typeCountSQL.MatchString(d.SQL) {
		c.n.Add(1)
	}
	return ctx
}

func (*typeCounter) TraceQueryEnd(context.Context, *pgx.Conn, pgx.TraceQueryEndData) {}

// TICK-991: a batch no longer counts the whole type. A five-batch backfill
// counts it once, on its first batch, and still writes the progress of each
// batch: 20, 40, 60, 80, then 99 until the flip to ready writes 100.
func TestBackfillCountsTheTypeOnce(t *testing.T) {
	e := migrated(t)
	f := newFixture(t, e, "num", "")
	insertRecords(t, e, f, 50)
	f.register(t, e, "num", fieldindex.StateBackfilling)
	tc := &typeCounter{}
	cfg := e.worker.Config()
	cfg.ConnConfig.Tracer = tc
	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	var mu sync.Mutex
	var seen []int16
	r := newRun(t, f.tenant, runOptions{acts: &backfill.Activities{Pool: pool, BatchSize: 10},
		onBatch: func(string) {
			p := registry(t, e, f).progress
			mu.Lock()
			seen = append(seen, p)
			mu.Unlock()
		}})
	r.signal(item(f, 1), 0)
	if err := r.coordinate(workerguard.RunInput{}); err != nil {
		t.Fatal(err)
	}
	r.requireGuardClean()
	if n := tc.n.Load(); n != 1 {
		t.Errorf("the type was counted %d times, want once", n)
	}
	// Six batches: five copy ten records each, the sixth finds none.
	if want := []int16{20, 40, 60, 80, 99, 99}; !slices.Equal(seen, want) {
		t.Errorf("progress after each batch %v, want %v", seen, want)
	}
	if got := registry(t, e, f); got.state != "ready" || got.progress != 100 {
		t.Fatalf("registry %+v, want ready at 100", got)
	}
	if got := diffOf(t, e, f); got != noDiff {
		t.Fatalf("side rows: %s", got)
	}
}

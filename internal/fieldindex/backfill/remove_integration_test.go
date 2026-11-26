//go:build integration

package backfill_test

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/fieldindex"
	"github.com/Navaladi2019/TicketingSystem/internal/fieldindex/backfill"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
	"github.com/Navaladi2019/TicketingSystem/internal/workerguard"
)

// TICK-750 (S22 P7-10, ADR-006 D11): FieldIndexRemove against a real
// PostgreSQL, behind the worker guard, run by the real coordinator.

// flipRemoving is the publish's flip (entitytypes.retireFilterable): the
// row to removing at requested_version version.
func flipRemoving(t testing.TB, e itEnv, f fixture, version int) {
	t.Helper()
	mustExec(t, e.super, `UPDATE record_index_fields SET state = 'removing', requested_version = $3, updated_at = now()
		WHERE tenant_id = $1 AND id = $2`, f.tenant, f.field, version)
}

// allSideRows counts f's field's rows in every side table.
func allSideRows(t testing.TB, e itEnv, f fixture) string {
	t.Helper()
	return queryString(t, e.super, `SELECT ((SELECT count(*) FROM rfv_text WHERE tenant_id = $1 AND field_id = $2)
		+ (SELECT count(*) FROM rfv_num WHERE tenant_id = $1 AND field_id = $2)
		+ (SELECT count(*) FROM rfv_ts WHERE tenant_id = $1 AND field_id = $2)
		+ (SELECT count(*) FROM rfv_bool WHERE tenant_id = $1 AND field_id = $2))::text`, f.tenant, f.field)
}

func registryRows(t testing.TB, e itEnv, f fixture) string {
	t.Helper()
	return queryString(t, e.super, `SELECT count(*)::text FROM record_index_fields WHERE tenant_id = $1 AND id = $2`, f.tenant, f.field)
}

// requireRemoved fails unless f's field has no side row and no registry row.
func requireRemoved(t *testing.T, e itEnv, f fixture) {
	t.Helper()
	if n := allSideRows(t, e, f); n != "0" {
		t.Errorf("field %s: %s side rows left, want 0", f.key, n)
	}
	if n := registryRows(t, e, f); n != "0" {
		t.Errorf("field %s: %s registry rows left, want 0", f.key, n)
	}
}

// A removal empties the field's side rows in batches across continue-as-new
// and deletes its registry row; the other fields of the type keep theirs, and
// the history carries references only (CI-5c): no record id, no value.
func TestRemoveEmptiesTheFieldAndDeletesItsRow(t *testing.T) {
	e := migrated(t)
	f := newFixture(t, e, "num", fieldindex.StateReady)
	other := fixture{tenant: f.tenant, def: f.def, field: uuid.New(), key: "f_text"}
	other.register(t, e, "text", fieldindex.StateReady)
	ids := insertRecords(t, e, f, 50)
	if n := sideRows(t, e, f); n != "50" {
		t.Fatalf("%s side rows before the removal, want 50", n)
	}
	flipRemoving(t, e, f, 2)

	r := newRun(t, f.tenant, runOptions{acts: &backfill.Activities{Pool: e.worker, BatchSize: 7},
		remover: &backfill.Remove{MaxBatchesPerRun: 3}})
	r.signal(removeItem(f, 2), 0)
	if err := r.coordinate(workerguard.RunInput{}); err != nil {
		t.Fatal(err)
	}
	r.requireGuardClean()
	requireRemoved(t, e, f)
	if got := diffOf(t, e, other); got != noDiff {
		t.Errorf("the other field: %s, want it untouched", got)
	}
	// 50 rows in batches of 7, three steps a run: the child continues as new
	// (each run is a start under the same id) and completes once.
	log := r.childLog()
	if len(log) < 3 || log[len(log)-1] != "done "+removeChild(f, 2) ||
		slices.ContainsFunc(log[:len(log)-1], func(s string) bool { return s != "start "+removeChild(f, 2) }) {
		t.Errorf("children %v, want runs of the one removal, continued as new, then done", log)
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
			if records[id] {
				t.Errorf("payload %s carries record id %s", s, id)
			}
		}
	}
}

// A replayed removal item, or one for a row that is not removing, deletes
// nothing: the steps read the registry row, never the item.
func TestRemoveOfANonRemovingRowIsANoOp(t *testing.T) {
	e := migrated(t)
	f := newFixture(t, e, "num", fieldindex.StateReady)
	insertRecords(t, e, f, 12)
	r := newRun(t, f.tenant, runOptions{acts: &backfill.Activities{Pool: e.worker}})
	r.signal(removeItem(f, 1), 0)
	if err := r.coordinate(workerguard.RunInput{}); err != nil {
		t.Fatal(err)
	}
	r.requireGuardClean()
	if got := registry(t, e, f); got.state != "ready" {
		t.Errorf("registry %+v, want ready", got)
	}
	if got := diffOf(t, e, f); got != noDiff {
		t.Errorf("side rows: %s, want them untouched", got)
	}
	if got := r.childLog(); len(got) != 0 {
		t.Errorf("children %v, want none: Classify found the row ready", got)
	}

	// The removal child run directly on a ready row (a stale child) changes
	// nothing either.
	direct := newRun(t, f.tenant, runOptions{acts: &backfill.Activities{Pool: e.worker}, workflowID: removeChild(f, 1)})
	direct.env.ExecuteWorkflow(fieldindex.RemoveWorkflowType, backfill.FieldRef{FieldID: f.field})
	if err := direct.env.GetWorkflowError(); err != nil {
		t.Fatal(err)
	}
	if got := diffOf(t, e, f); got != noDiff || registryRows(t, e, f) != "1" {
		t.Errorf("after a stale removal child: %s and %s registry rows, want both untouched", got, registryRows(t, e, f))
	}
}

// Another tenant's field is out of reach: a removal child of tenant A that
// names tenant B's field (by id) deletes none of B's rows.
func TestRemoveCannotReachAnotherTenantsField(t *testing.T) {
	e := migrated(t)
	a := newFixture(t, e, "num", "")
	b := newFixture(t, e, "num", fieldindex.StateReady)
	insertRecords(t, e, b, 10)
	flipRemoving(t, e, b, 2)
	forged := fixture{tenant: a.tenant, def: a.def, field: b.field, key: b.key}
	direct := newRun(t, a.tenant, runOptions{acts: &backfill.Activities{Pool: e.worker}, workflowID: removeChild(forged, 2)})
	direct.env.ExecuteWorkflow(fieldindex.RemoveWorkflowType, backfill.FieldRef{FieldID: b.field})
	if err := direct.env.GetWorkflowError(); err != nil {
		t.Fatal(err)
	}
	if n, m := sideRows(t, e, b), registryRows(t, e, b); n != "10" || m != "1" {
		t.Errorf("tenant B: %s side rows and %s registry rows after tenant A's child, want 10 and 1", n, m)
	}
}

// Unflag A while B is backfilling: A's removal starts only after B's
// backfill finishes (one child at a time per tenant, D-7-12), and both end
// correct.
func TestRemoveWaitsForTheTenantsBackfill(t *testing.T) {
	e := migrated(t)
	a := newFixture(t, e, "num", fieldindex.StateReady)
	b := fixture{tenant: a.tenant, def: a.def, field: uuid.New(), key: "f_text"}
	insertRecords(t, e, a, 40)
	b.register(t, e, "text", fieldindex.StateBackfilling)
	var flipped atomic.Bool
	r := newRun(t, a.tenant, runOptions{acts: &backfill.Activities{Pool: e.worker, BatchSize: 10},
		onBatch: func(id string) {
			if id == backfillChild(b, 1) && flipped.CompareAndSwap(false, true) {
				flipRemoving(t, e, a, 2) // the unflag publish commits mid-backfill
			}
		}})
	r.signal(item(b, 1), 0)
	r.signal(removeItem(a, 2), 700*time.Millisecond) // B's child is running then
	if err := r.coordinate(workerguard.RunInput{}); err != nil {
		t.Fatal(err)
	}
	r.requireGuardClean()
	want := []string{"start " + backfillChild(b, 1), "done " + backfillChild(b, 1), "start " + removeChild(a, 2), "done " + removeChild(a, 2)}
	if got := r.childLog(); !slices.Equal(got, want) {
		t.Fatalf("children %v, want %v", got, want)
	}
	if got := registry(t, e, b); got.state != "ready" {
		t.Errorf("B %+v, want ready", got)
	}
	if got := diffOf(t, e, b); got != noDiff {
		t.Errorf("B: %s", got)
	}
	requireRemoved(t, e, a)
}

// CI-5 R3 (deterministic half): a writer whose trigger read the row as
// ready before the flip, and commits only after the removal has deleted the
// side rows it could see, holds an uncommitted side row. The barrier waits
// for it, so the removal deletes it and no step fails. With the barrier
// replaced by a no-op (the seam), Retire's registry DELETE runs into that
// row and the side table's FK refuses it: an attempt fails, which the worker
// guard records as a deny with alert=true. The test bites.
func TestR3WriterAcrossTheFlip(t *testing.T) {
	e := migrated(t)
	for _, barrierOn := range []bool{true, false} {
		name := map[bool]string{true: "barrier", false: "no barrier"}[barrierOn]
		t.Run(name, func(t *testing.T) {
			f := newFixture(t, e, "num", fieldindex.StateReady)
			insertRecords(t, e, f, 20)
			w := beginWriter(t, e, f) // its trigger saw the row ready: one more side row, uncommitted
			flipRemoving(t, e, f, 2)

			var committed atomic.Bool
			commit := func() {
				if committed.CompareAndSwap(false, true) {
					w.commit(t)
				}
			}
			o := runOptions{acts: &backfill.Activities{Pool: e.worker, BatchSize: 5}}
			if !barrierOn {
				o.barrier = func(context.Context) error { return nil }
			}
			// The writer commits later, as a slow writer would: with the
			// barrier the removal waits for it; without, Retire's DELETE waits
			// on its row lock and then fails on the FK.
			go func() { time.Sleep(1500 * time.Millisecond); commit() }()
			r := newRun(t, f.tenant, o)
			r.signal(removeItem(f, 2), 0)
			if err := r.coordinate(workerguard.RunInput{}); err != nil {
				t.Fatal(err)
			}
			commit()
			// Either way the retry ends correct; the barrier is what keeps a
			// step from failing (and paging) on the way.
			requireRemoved(t, e, f)
			denies := r.guardDenies()
			switch {
			case barrierOn && len(denies) != 0:
				t.Fatalf("with the barrier: guard denies %v, want none; logs: %s", denies, r.logText())
			case !barrierOn && len(denies) == 0:
				t.Fatal("without the barrier: no step failed, want Retire refused by the FK: the test does not bite")
			case !barrierOn && !strings.Contains(r.logText(), backfill.RetireActivity):
				t.Errorf("without the barrier: denies %v, want them from %s; logs: %s", denies, backfill.RetireActivity, r.logText())
			}
		})
	}
}

// CI-5 R3 (load half): writers update and insert records of the type
// continuously across the flip and during the deletes; after the removal
// there are no side rows for the field and no registry row, and the guard
// recorded no failure.
func TestR3WritersAcrossTheFlipAndTheDeletes(t *testing.T) {
	e := migrated(t)
	f := newFixture(t, e, "num", fieldindex.StateReady)
	ids := insertRecords(t, e, f, 120)

	ctx, stop := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	var writes atomic.Int64
	errs := make(chan error, 8)
	for g := range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; ctx.Err() == nil; i++ {
				var err error
				if i%5 == 4 {
					err = writeRecord(ctx, e, f, uuid.Must(uuid.NewV7()), 1000*g+i)
				} else {
					id := ids[(g*31+i)%len(ids)]
					err = tenancy.InTenantTx(ctx, e.app, f.tenant, func(tx pgx.Tx) error {
						_, err := tx.Exec(ctx, updateRecordSQL, f.tenant, id, fmt.Sprint(i), f.key)
						return err
					})
				}
				if err != nil && ctx.Err() == nil {
					errs <- err
					return
				}
				writes.Add(1)
			}
		}()
	}
	time.Sleep(200 * time.Millisecond) // writers running before the flip
	flipRemoving(t, e, f, 2)
	r := newRun(t, f.tenant, runOptions{acts: &backfill.Activities{Pool: e.worker, BatchSize: 10}})
	r.signal(removeItem(f, 2), 0)
	err := r.coordinate(workerguard.RunInput{})
	time.Sleep(100 * time.Millisecond) // writers running after the removal too
	stop()
	wg.Wait()
	close(errs)
	if err != nil {
		t.Fatal(err)
	}
	for err := range errs {
		t.Errorf("a writer failed: %v", err)
	}
	if writes.Load() < 20 {
		t.Fatalf("only %d writes: the writers did not overlap the removal", writes.Load())
	}
	r.requireGuardClean()
	requireRemoved(t, e, f)
}

//go:build integration

package isolation

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"

	"github.com/Navaladi2019/TicketingSystem/internal/fieldindex"
	"github.com/Navaladi2019/TicketingSystem/internal/fieldindex/backfill"
	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
	"github.com/Navaladi2019/TicketingSystem/internal/workerguard"
)

// S22Remove1b and S22Race2b run rows S22 1b and 2b (TICK-1198) on the
// suite's cluster. They live in a test file for row 3b's reason: only
// internal/fieldindex may name the field-index workflow types outside tests,
// and the rows drive them on the Temporal SDK test environment.
// s22_s29_attack_paths.go registers the rows and calls them through
// s22Remover.
func (c cluster) S22Remove1b(ctx context.Context, e *etIso, fx *s21Fixture) error {
	return s22Remove1b(ctx, c, e, fx)
}

func (c cluster) S22Race2b(ctx context.Context, e *etIso, fx *s21Fixture) error {
	return s22Race2b(ctx, c, e, fx, true)
}

// s22RemoveVersion is the requested_version the rows' flips set and their
// removal items carry, as the unflag publish would (v1 created the type).
const s22RemoveVersion = 2

// s22Flip is the unflag publish's flip (entitytypes.retireFilterable), past
// RLS: tenant's iso_et_s21 note to removing at s22RemoveVersion.
func s22Flip(ctx context.Context, e *etIso, tenant string) error {
	tag, err := e.super.Exec(ctx, `UPDATE record_index_fields f SET state = 'removing', requested_version = $2, updated_at = now()
		FROM entity_definitions d WHERE d.tenant_id = f.tenant_id AND d.id = f.entity_definition_id
		AND f.tenant_id = $1 AND d.key = $3 AND f.field_key = 'note'`, tenant, s22RemoveVersion, s21Key)
	if err != nil || tag.RowsAffected() != 1 {
		return fmt.Errorf("control: flip tenant %s's note to removing: %d rows, %w", tenant, tag.RowsAffected(), s22Err(err))
	}
	return nil
}

// s22Note is a field's footprint: its registry rows, and its side rows in
// every side table, past RLS.
type s22Note struct {
	registry, side int
}

func s22ReadNote(ctx context.Context, e *etIso, tenant string, field uuid.UUID) (s22Note, error) {
	var n s22Note
	err := e.super.QueryRow(ctx, `SELECT
		(SELECT count(*) FROM record_index_fields WHERE tenant_id = $1 AND id = $2),
		(SELECT count(*) FROM rfv_text WHERE tenant_id = $1 AND field_id = $2)
		+ (SELECT count(*) FROM rfv_num WHERE tenant_id = $1 AND field_id = $2)
		+ (SELECT count(*) FROM rfv_ts WHERE tenant_id = $1 AND field_id = $2)
		+ (SELECT count(*) FROM rfv_bool WHERE tenant_id = $1 AND field_id = $2)`, tenant, field).Scan(&n.registry, &n.side)
	return n, err
}

// s22Untouched fails when tenant B's registry moved, tenant B's note lost a
// side row, or any side row pairs tenant A with tenant B's field.
func s22Untouched(ctx context.Context, e *etIso, noteA, noteB uuid.UUID, regB string, sideB int) error {
	if got, err := s21Registry(ctx, e, TenantB); err != nil {
		return fmt.Errorf("read tenant B's registry: %w", err)
	} else if got != regB {
		return fmt.Errorf("tenant B's registry moved:\n%s\n%s", regB, got)
	}
	b, err := s22ReadNote(ctx, e, TenantB, noteB)
	if err != nil {
		return err
	}
	if b != (s22Note{registry: 1, side: sideB}) {
		return fmt.Errorf("tenant B's note is %+v, want its registry row and %d side rows", b, sideB)
	}
	cross, err := s22Cross(ctx, e.super, noteB)
	if err != nil {
		return fmt.Errorf("count the cross rows: %w", err)
	}
	if cross != 0 {
		return fmt.Errorf("%d side rows pair tenant A with tenant B's field", cross)
	}
	return nil
}

// s22Querier is a *pgx.Conn or a pgx.Tx.
type s22Querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// s22Cross counts the side rows, in every side table, that pair tenant A
// with field (tenant B's), past RLS. It reads the side tables alone, so it
// counts whether or not tenant A's own note is still registered.
func s22Cross(ctx context.Context, q s22Querier, field uuid.UUID) (int, error) {
	var n int
	err := q.QueryRow(ctx, `SELECT
		(SELECT count(*) FROM rfv_text WHERE tenant_id = $1 AND field_id = $2)
		+ (SELECT count(*) FROM rfv_num WHERE tenant_id = $1 AND field_id = $2)
		+ (SELECT count(*) FROM rfv_ts WHERE tenant_id = $1 AND field_id = $2)
		+ (SELECT count(*) FROM rfv_bool WHERE tenant_id = $1 AND field_id = $2)`, TenantA, field).Scan(&n)
	return n, err
}

// s22CrossBites is s22Cross's control: in a superuser transaction with the
// FKs off, one rfv_text row pairing tenant A's record with tenant B's field
// is counted, and the transaction rolls back.
func s22CrossBites(ctx context.Context, e *etIso, field uuid.UUID, record string) error {
	tx, err := e.super.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `SET LOCAL session_replication_role = replica`); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO rfv_text (tenant_id, field_id, record_id, ord, value) VALUES ($1, $2, $3, 0, 's22-cross')`,
		TenantA, field, record); err != nil {
		return err
	}
	if n, err := s22Cross(ctx, tx, field); err != nil || n != 1 {
		return fmt.Errorf("control: the cross count saw %d of 1 planted row (%w): it cannot catch a cross row", n, s22Err(err))
	}
	return nil
}

// s22RemoveItem is the relay's signal body for a field_index.remove row.
func s22RemoveItem(row s3bRow, field uuid.UUID) fieldindex.EnqueueItem {
	return fieldindex.EnqueueItem{Op: fieldindex.OpRemove, FieldID: field.String(), RequestedVersion: s22RemoveVersion,
		OutboxID: row.delivery.OutboxID.String()}
}

// s22EnqueueRemove writes a field_index.remove row for field in tenant
// through outbox.EnqueueFieldIndexRemove as nava_app (the publish's path),
// and reads it back past RLS. s21Cleanup deletes it while field's registry
// row is there; the caller deletes it after a removal.
func s22EnqueueRemove(ctx context.Context, e *etIso, tenant, field uuid.UUID) (s3bRow, error) {
	var id uuid.UUID
	if err := tenancy.InTenantTx(ctx, e.pool, tenant, func(tx pgx.Tx) error {
		var err error
		id, err = outbox.EnqueueFieldIndexRemove(ctx, tx, field, s22RemoveVersion)
		return err
	}); err != nil {
		return s3bRow{}, err
	}
	var rowTenant string
	var payload []byte
	if err := e.super.QueryRow(ctx, `SELECT tenant_id::text, payload::text FROM outbox WHERE id = $1`, id).Scan(&rowTenant, &payload); err != nil {
		return s3bRow{}, err
	}
	if rowTenant != tenant.String() {
		return s3bRow{}, fmt.Errorf("the row is tenant %s's, want %s", rowTenant, tenant)
	}
	return s3bRow{delivery: outbox.Delivery{OutboxID: id, KindVersion: 1, Payload: payload}}, nil
}

// s22DropRemoveRows deletes the rows' field_index.remove outbox rows, which
// s21Cleanup no longer finds once a removal has deleted their registry row.
func s22DropRemoveRows(ctx context.Context, e *etIso, rows ...s3bRow) error {
	for _, r := range rows {
		if _, err := e.super.Exec(ctx, `DELETE FROM outbox WHERE id = $1`, r.delivery.OutboxID); err != nil {
			return err
		}
	}
	return nil
}

// s22Run is row 3b's runner with FieldIndexRemove registered too, and, when
// noBarrier is set, the barrier replaced by a no-op (the D9 test seam, as
// TICK-750's TestR3WriterAcrossTheFlip uses it): the broken variant row 2b
// must fail against.
func s22Run(tenant uuid.UUID, workflowID string, acts *backfill.Activities, noBarrier bool) *s3bRunner {
	r := s3bRun(tenant, workflowID, acts)
	(&backfill.Remove{}).Register(r.env)
	if noBarrier {
		r.env.RegisterActivityWithOptions(func(context.Context) error { return nil }, activity.RegisterOptions{Name: backfill.BarrierActivity})
	}
	return r
}

// s22Alerts reports whether the worker guard logged a deny (alert=true): an
// activity attempt that failed, here the side-table FK refusing Retire.
func s22Alerts(r *s3bRunner) bool { return strings.Contains(r.log(), `"alert":true`) }

// s22Remove1b is S22 attack row 1b (ADR-006 D11; TICK-750's FieldIndexRemove):
// the real coordinator, FieldIndexRemove and their activities, on
// nava_worker under RLS, behind the worker guard and the nava-tenant
// Propagator, started as the relay starts them for a field_index.remove
// outbox row of tenant A.
//
// Before it runs, tenant A's note and tenant B's note are both removing, so
// tenant B's field is a live target a fooled removal would empty and retire.
//
//  1. Tenant A's row names tenant B's field id. The coordinator gets that
//     item and, as the control, one for tenant A's own note. It starts a
//     removal child for tenant A's note only, which empties it and deletes
//     its registry row; tenant B's registry and its note's side rows do not
//     move.
//  2. One layer down: FieldIndexRemove started directly under tenant A with
//     tenant B's field id in its workflow id, as a fooled coordinator would
//     start it. Its barrier, batch and retire run (the control) and delete
//     nothing.
//  3. Tenant A's row whose payload tenant_id is tenant B, naming tenant B's
//     field. The workflow id and the relay header say tenant A; the
//     coordinator fails closed, non-retryable nava.field_index.NoChild,
//     before any activity runs, and nothing of tenant B moves.
func s22Remove1b(ctx context.Context, cl Cluster, e *etIso, fx *s21Fixture) (err error) {
	a, b := uuid.MustParse(TenantA), uuid.MustParse(TenantB)
	ids := map[string]uuid.UUID{}
	for _, f := range []struct{ tenant, name string }{{TenantA, "A"}, {TenantB, "B"}} {
		id, err := s21FieldID(ctx, e, f.tenant, "note")
		if err != nil {
			return err
		}
		ids[f.name] = uuid.MustParse(id)
	}
	noteA, noteB := ids["A"], ids["B"]
	for _, t := range []string{TenantA, TenantB} {
		if err := s22Flip(ctx, e, t); err != nil {
			return err
		}
	}
	regB, err := s21Registry(ctx, e, TenantB)
	if err != nil {
		return err
	}
	nA, err := s22ReadNote(ctx, e, TenantA, noteA)
	if err != nil {
		return err
	}
	nB, err := s22ReadNote(ctx, e, TenantB, noteB)
	if err != nil {
		return err
	}
	if nA != (s22Note{registry: 1, side: 2}) || nB != (s22Note{registry: 1, side: 2}) {
		return fmt.Errorf("control: before the runs tenant A's note %+v and tenant B's %+v, want 1 registry row and 2 side rows each", nA, nB)
	}
	if err := s22CrossBites(ctx, e, noteB, fx.desk); err != nil {
		return err
	}

	pool, err := cl.PoolSize("nava_worker", 4)
	if err != nil {
		return err
	}
	defer pool.Close()
	acts := &backfill.Activities{Pool: pool, BatchSize: 1}

	rowB, err := s22EnqueueRemove(ctx, e, a, noteB)
	if err != nil {
		return fmt.Errorf("control: enqueue the removal of tenant B's field in tenant A: %w", err)
	}
	rowA, err := s22EnqueueRemove(ctx, e, a, noteA)
	if err != nil {
		return fmt.Errorf("control: enqueue the removal of tenant A's note: %w", err)
	}
	defer func() {
		if derr := s22DropRemoveRows(ctx, e, rowA, rowB); err == nil {
			err = derr
		}
	}()

	// 1. Through the coordinator.
	r := s22Run(a, fieldindex.CoordinatorWorkflowID(TenantA), acts, false)
	r.signal(s22RemoveItem(rowB, noteB), 0)
	r.signal(s22RemoveItem(rowA, noteA), time.Second)
	if err := r.execute(fieldindex.CoordinatorWorkflowType, workerguard.RunInput{Delivery: rowB.delivery}); err != nil {
		return fmt.Errorf("case 1: the coordinator in tenant A = %w, want nil; worker log %s", err, r.log())
	}
	wantChild := backfill.ChildID(a, fieldindex.RemoveWorkflowType, noteA, s22RemoveVersion)
	if got := r.childLog(); len(got) != 2 || got[0] != "start "+wantChild || got[1] != "done "+wantChild {
		return fmt.Errorf("case 1: the coordinator's children = %v, want only %s, tenant A's note (control)", got, wantChild)
	}
	if got := r.count(backfill.ClassifyActivity); got != 2 {
		return fmt.Errorf("control: case 1 classified %d items, want 2 (tenant B's field id, tenant A's note)", got)
	}
	if s22Alerts(r) {
		return fmt.Errorf("control: case 1's worker guard denied: %s", r.log())
	}
	if got, err := s22ReadNote(ctx, e, TenantA, noteA); err != nil || got != (s22Note{}) {
		return fmt.Errorf("control: case 1 left tenant A's note %+v (%w), want it removed", got, s22Err(err))
	}
	if err := s22Untouched(ctx, e, noteA, noteB, regB, 2); err != nil {
		return fmt.Errorf("case 1 (tenant A's remove row names tenant B's field): %w", err)
	}

	// 2. The removal child itself, under tenant A, on tenant B's field.
	child := backfill.ChildID(a, fieldindex.RemoveWorkflowType, noteB, s22RemoveVersion)
	r = s22Run(a, child, acts, false)
	if err := r.execute(fieldindex.RemoveWorkflowType, backfill.FieldRef{FieldID: noteB}); err != nil {
		return fmt.Errorf("case 2: FieldIndexRemove %s = %w, want nil (it finds no field and deletes nothing)", child, err)
	}
	if r.count(backfill.BarrierActivity) != 1 || r.count(backfill.RemoveBatchActivity) != 1 || r.count(backfill.RetireActivity) != 1 {
		return fmt.Errorf("control: case 2: FieldIndexRemove %s ran %v, want one barrier, one batch and one retire", child, r.counts())
	}
	if err := s22Untouched(ctx, e, noteA, noteB, regB, 2); err != nil {
		return fmt.Errorf("case 2 (FieldIndexRemove under tenant A on tenant B's field): %w", err)
	}

	// 3. The payload names tenant B, the row and the header tenant A.
	forged, err := rowB.forge(b, noteB)
	if err != nil {
		return err
	}
	r = s22Run(a, fieldindex.CoordinatorWorkflowID(TenantA), acts, false)
	r.signal(s22RemoveItem(forged, noteB), 0)
	err = r.execute(fieldindex.CoordinatorWorkflowType, workerguard.RunInput{Delivery: forged.delivery})
	var app *temporal.ApplicationError
	if !errors.As(err, &app) || !app.NonRetryable() || !strings.Contains(r.log(), backfill.ErrTypeNoChild) {
		return fmt.Errorf("case 3 (payload tenant_id is tenant B): the coordinator in tenant A = %w, worker log %s; want a non-retryable %s",
			err, r.log(), backfill.ErrTypeNoChild)
	}
	if n := r.counts(); len(n) != 0 {
		return fmt.Errorf("case 3 (payload tenant_id is tenant B): activities ran %v, want none", n)
	}
	if err := s22Untouched(ctx, e, noteA, noteB, regB, 2); err != nil {
		return fmt.Errorf("case 3 (payload tenant_id is tenant B): %w", err)
	}
	return nil
}

// s22WriteSQL rewrites record $2's note to $3 with its audit entry, in one
// statement (the records commit check): the D4 trigger indexes the note if
// its registry row is ready when the statement runs.
const s22WriteSQL = `WITH r AS (UPDATE records SET data = data || jsonb_build_object('note', $3::text), version = version + 1
	WHERE tenant_id = $1 AND id = $2 RETURNING tenant_id, id, version)
	INSERT INTO audit_log (tenant_id, id, domain, action, subject_type, subject_id, version_after, actor_system, source)
	SELECT tenant_id, gen_random_uuid(), 'record', 'update', 'record', id, version, 'system', 'system' FROM r`

// s22Held is a writer transaction open across the flip: it rewrote a
// record that had no note, so its trigger, which saw the note ready, holds
// one uncommitted side row and no lock on any committed one.
type s22Held struct {
	role    string
	release chan struct{}
	done    chan error
	once    sync.Once
	err     error
}

func s22Hold(ctx context.Context, pool *pgxpool.Pool, role, record string, noteA uuid.UUID) (*s22Held, error) {
	h := &s22Held{role: role, release: make(chan struct{}), done: make(chan error, 1)}
	ready := make(chan error, 1)
	value := "s22-2b-held-" + role
	go func() {
		h.done <- tenancy.InTenantTx(ctx, pool, uuid.MustParse(TenantA), func(tx pgx.Tx) error {
			tag, err := tx.Exec(ctx, s22WriteSQL, TenantA, record, value)
			if err == nil && tag.RowsAffected() != 1 {
				err = fmt.Errorf("%d audit rows, want 1", tag.RowsAffected())
			}
			var n int
			if err == nil {
				err = tx.QueryRow(ctx, `SELECT count(*) FROM rfv_text WHERE tenant_id = $1 AND field_id = $2 AND record_id = $3 AND value = $4`,
					TenantA, noteA, record, value).Scan(&n)
			}
			if err == nil && n != 1 {
				err = fmt.Errorf("control: %s's write before the flip indexed tenant A's ready note %d times, want 1", role, n)
			}
			ready <- err
			if err != nil {
				return err
			}
			<-h.release
			return nil
		})
	}()
	if err := <-ready; err != nil {
		return nil, fmt.Errorf("%s holding a write across the flip: %w", role, err)
	}
	return h, nil
}

// commit lets the writer commit, waits for it, and returns its result;
// every call returns the first one's.
func (h *s22Held) commit() error {
	h.once.Do(func() {
		close(h.release)
		if err := <-h.done; err != nil {
			h.err = fmt.Errorf("%s's held write: %w", h.role, err)
		}
	})
	return h.err
}

// s22RetireWaits polls, on its own superuser connection, until a backend
// waits on a lock in Retire's registry DELETE (the side-table FK check
// blocked on a held writer's uncommitted row), for up to 30 s.
func s22RetireWaits(ctx context.Context, cl Cluster) error {
	conn, err := cl.Super()
	if err != nil {
		return err
	}
	for deadline := time.Now().Add(30 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		var n int
		if err := conn.QueryRow(ctx, `SELECT count(*) FROM pg_stat_activity
			WHERE wait_event_type = 'Lock' AND query LIKE 'DELETE FROM record_index_fields%'`).Scan(&n); err != nil {
			return err
		}
		if n > 0 {
			return nil
		}
	}
	return errors.New("no Retire DELETE waited on a held writer within 30 s")
}

// s22Race2b is S22 attack row 2b, CI-5 R3 (ADR-006 D9, D11): writers of every
// records-writer role (s22RecordWriters) race a real FieldIndexRemove of
// tenant A's note, run by the coordinator as row 1b runs it.
//
//   - One writer per role holds a transaction open across the flip: its
//     trigger saw the note ready and holds an uncommitted side row (the
//     control checks it). The held writers commit at a point in the
//     removal, never on a timer: 500 ms after the removal's barrier starts,
//     so the barrier must wait for them, and Retire must not start before
//     they commit.
//   - One writer per role rewrites a noted record of tenant A in a loop,
//     committing each time, from before the flip until after the removal.
//
// The removal deletes one side row per batch, so its deletes interleave with
// the writes. After it, tenant A's note has 0 side rows and no registry row,
// no writer failed, and the worker guard recorded no failed attempt: the
// barrier waited for the held writers, so Retire never met the side-table
// FK. Tenant B's note, ready under the same key, keeps its registry row and
// side rows, and no side row pairs tenant A with it.
//
// With barrier false (the D9 seam set to a no-op), the held writers commit
// only once Retire's registry DELETE is seen waiting on their rows
// (s22RetireWaits); the side-table FK then refuses it, and the row fails on
// the guard's alert. TestS22Attack2bBitesWithoutTheBarrier keeps that so.
func s22Race2b(ctx context.Context, cl Cluster, e *etIso, _ *s21Fixture, barrier bool) (err error) {
	writers, err := s22RecordWriters()
	if err != nil {
		return err
	}
	a := uuid.MustParse(TenantA)
	ids := map[string]uuid.UUID{}
	for _, f := range []struct{ tenant, name string }{{TenantA, "A"}, {TenantB, "B"}} {
		id, err := s21FieldID(ctx, e, f.tenant, "note")
		if err != nil {
			return err
		}
		ids[f.name] = uuid.MustParse(id)
	}
	noteA, noteB := ids["A"], ids["B"]
	regB, err := s21Registry(ctx, e, TenantB)
	if err != nil {
		return err
	}
	nB, err := s22ReadNote(ctx, e, TenantB, noteB)
	if err != nil || nB.side != 2 {
		return fmt.Errorf("control: tenant B's note %+v (%w), want 2 side rows", nB, s22Err(err))
	}

	// One record with no note per held writer, one with a note per looping
	// writer, all tenant A's.
	held, loop := make([]string, len(writers)), make([]string, len(writers))
	for i := range writers {
		if held[i], err = s21Create(ctx, e, s21Admin, `{"data": {"qty": 1}}`); err != nil {
			return err
		}
		if loop[i], err = s21Create(ctx, e, s21Admin, fmt.Sprintf(`{"data": {"note": "s22-2b-loop-%d", "qty": 2}}`, i)); err != nil {
			return err
		}
	}

	// The looping writers start before the flip.
	lctx, stop := context.WithCancel(ctx)
	defer stop()
	var wg sync.WaitGroup
	var writes atomic.Int64
	errs := make(chan error, len(writers))
	var heldWriters []*s22Held
	var pools []*pgxpool.Pool
	defer func() {
		stop()
		wg.Wait()
		for _, h := range heldWriters {
			_ = h.commit()
		}
		for _, p := range pools {
			p.Close()
		}
	}()
	for i, role := range writers {
		pool, err := cl.PoolSize(role, 3)
		if err != nil {
			return err
		}
		pools = append(pools, pool)
		wg.Add(1)
		go func() {
			defer wg.Done()
			for n := 0; lctx.Err() == nil; n++ {
				err := tenancy.InTenantTx(lctx, pool, a, func(tx pgx.Tx) error {
					_, err := tx.Exec(lctx, s22WriteSQL, TenantA, loop[i], fmt.Sprintf("s22-2b-%s-%d", role, n))
					return err
				})
				if err != nil {
					if lctx.Err() == nil {
						errs <- fmt.Errorf("%s's looping write: %w", role, err)
					}
					return
				}
				writes.Add(1)
			}
		}()
		h, err := s22Hold(ctx, pool, role, held[i], noteA)
		if err != nil {
			return err
		}
		heldWriters = append(heldWriters, h)
	}
	time.Sleep(200 * time.Millisecond) // the looping writers run before the flip
	if err := s22Flip(ctx, e, TenantA); err != nil {
		return err
	}

	wpool, err := cl.PoolSize("nava_worker", 4)
	if err != nil {
		return err
	}
	defer wpool.Close()
	row, err := s22EnqueueRemove(ctx, e, a, noteA)
	if err != nil {
		return fmt.Errorf("control: enqueue the removal of tenant A's note: %w", err)
	}
	defer func() {
		if derr := s22DropRemoveRows(ctx, e, row); err == nil {
			err = derr
		}
	}()
	r := s22Run(a, fieldindex.CoordinatorWorkflowID(TenantA), &backfill.Activities{Pool: wpool, BatchSize: 1}, !barrier)
	r.signal(s22RemoveItem(row, noteA), 0)

	// The commit points. commitHeld commits every held writer once; with the
	// barrier, Retire starting before it releases them is the barrier not
	// waiting.
	var trigger sync.Once
	var released, retireEarly atomic.Bool
	commitDone := make(chan error, 1)
	commitHeld := func(wait func() error) {
		werr := wait()
		released.Store(true) // before the first commit: Retire before this cannot have waited
		for _, h := range heldWriters {
			if err := h.commit(); err != nil && werr == nil {
				werr = err
			}
		}
		commitDone <- werr
	}
	r.onStart = func(activityType string) {
		switch {
		case barrier && activityType == backfill.BarrierActivity:
			trigger.Do(func() {
				go commitHeld(func() error { time.Sleep(500 * time.Millisecond); return nil })
			})
		case activityType == backfill.RetireActivity:
			if barrier && !released.Load() {
				retireEarly.Store(true)
			}
			if !barrier {
				trigger.Do(func() { go commitHeld(func() error { return s22RetireWaits(ctx, cl) }) })
			}
		}
	}
	before := writes.Load()
	runErr := r.execute(fieldindex.CoordinatorWorkflowType, workerguard.RunInput{Delivery: row.delivery})
	during := writes.Load() - before
	time.Sleep(100 * time.Millisecond) // the looping writers run after the removal too
	stop()
	wg.Wait()
	close(errs)
	var heldErr error
	for _, h := range heldWriters {
		if err := h.commit(); err != nil && heldErr == nil {
			heldErr = err
		}
	}

	if runErr != nil {
		return fmt.Errorf("the coordinator in tenant A = %w, want nil; worker log %s", runErr, r.log())
	}
	if err, ok := <-errs; ok {
		return err
	}
	if heldErr != nil {
		return heldErr
	}
	select {
	case err := <-commitDone:
		if err != nil {
			return fmt.Errorf("control: the commit point: %w", err)
		}
	default:
		return fmt.Errorf("control: the removal never reached its commit point (activities %v)", r.counts())
	}
	if retireEarly.Load() {
		return errors.New("R3: Retire started before the writers held across the flip committed: the barrier did not wait for them")
	}
	if during < int64(2*len(writers)) {
		return fmt.Errorf("control: only %d looping writes while the removal ran: the writers did not overlap it", during)
	}
	want := backfill.ChildID(a, fieldindex.RemoveWorkflowType, noteA, s22RemoveVersion)
	if got := r.childLog(); len(got) == 0 || got[len(got)-1] != "done "+want {
		return fmt.Errorf("control: the coordinator's children = %v, want the removal %s done", got, want)
	}
	if got, err := s22ReadNote(ctx, e, TenantA, noteA); err != nil || got != (s22Note{}) {
		return fmt.Errorf("R3: after the removal tenant A's note has %+v (%w), want 0 side rows and no registry row", got, s22Err(err))
	}
	if s22Alerts(r) {
		return fmt.Errorf("R3: a removal step failed during the race (the worker guard denied), want none: %s", r.log())
	}
	if err := s22Untouched(ctx, e, noteA, noteB, regB, 2); err != nil {
		return fmt.Errorf("tenant A's removal under the race: %w", err)
	}
	return nil
}

// TestS22Attack2bBitesWithoutTheBarrier is row 2b's broken variant, kept as
// a guard: with the D9 barrier seam set to a no-op, a held writer's side row
// commits after the removal's NOT EXISTS, Retire's registry DELETE meets the
// side-table FK, and row 2b fails. If this passes the row 2b check, the row
// no longer proves the barrier.
func TestS22Attack2bBitesWithoutTheBarrier(t *testing.T) {
	cl := cluster{t: t, e: suite}
	err := withS21Fixture(t.Context(), cl, func(ctx context.Context, e *etIso, fx *s21Fixture) error {
		return s22Race2b(ctx, cl, e, fx, false)
	})
	if err == nil || !strings.Contains(err.Error(), "R3: a removal step failed") {
		t.Fatalf("row 2b with the barrier a no-op = %v, want the R3 failure (the guard's alert on Retire): the row does not bite", err)
	}
	t.Logf("row 2b without the barrier fails, as it must: %.400s", err)
	if !strings.Contains(err.Error(), backfill.RetireActivity) {
		t.Errorf("row 2b without the barrier failed on %v, want %s refused", err, backfill.RetireActivity)
	}
}

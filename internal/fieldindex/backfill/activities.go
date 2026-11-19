package backfill

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/metric"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"

	"github.com/Navaladi2019/TicketingSystem/internal/fieldindex"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
	"github.com/Navaladi2019/TicketingSystem/internal/workerguard"
)

// The registered activity names.
const (
	// BarrierActivity is fieldindex.Barrier (D9). FieldIndexRemove (TICK-750)
	// reuses it: it takes no argument and reads nothing but the snapshot.
	BarrierActivity  = "field_index.Barrier"
	ClassifyActivity = "field_index.Classify"
	BatchActivity    = "field_index.Batch"
	ReadyActivity    = "field_index.Ready"
)

// ErrTypeNoChild is the ApplicationError type of a workflow id or input that
// is not a field-index child's. It is non-retryable: a retry reads the same
// id. It is not on the worker guard's allow-list, so the guard records it as
// nava.Opaque; the type matters only to this worker's own log.
const ErrTypeNoChild = "nava.field_index.NoChild"

// DefaultBatchSize is the records one batch copies (D10.2).
const DefaultBatchSize = 1000

// DefaultSlowAfter is how long the barrier waits before it raises
// field_index_barrier_slow (D9, R11).
const DefaultSlowAfter = 15 * time.Minute

// FieldRef is the input of Classify and the coordinator's child of a removal:
// a field id, a reference only.
type FieldRef struct {
	FieldID uuid.UUID `json:"field_id"`
}

// Cursor is Batch's input and result: the last record id copied, absent
// before the first batch and in the result once nothing is left to copy, or
// the registry row is no longer this child's to fill. It is the only thing a
// batch returns: progress goes to the registry row in the batch's own
// transaction (D10 (d)), and workerguard records references only (CI-7).
type Cursor struct {
	Cursor *uuid.UUID `json:"cursor,omitempty"`
}

// Classified is Classify's result: which child, if any, the coordinator runs
// for a field. At most one field is set, and none when the registry row is
// ready or gone.
type Classified struct {
	Backfill *uuid.UUID `json:"backfill,omitempty"`
	Remove   *uuid.UUID `json:"remove,omitempty"`
}

// Activities are the coordinator's and FieldIndexBackfill's activities on a
// nava_worker pool. Register them with Register. Metrics and Log may be nil.
type Activities struct {
	Pool    *pgxpool.Pool
	Metrics *Metrics
	Log     *slog.Logger
	// BatchSize is the records one batch copies; 0 is DefaultBatchSize.
	BatchSize int
	// SlowAfter is the barrier's slow threshold; 0 is DefaultSlowAfter.
	SlowAfter time.Duration

	// afterSnapshot, when set, runs between the barrier's snapshot and its
	// fence (test-only: it widens [lo, hi)).
	afterSnapshot func(context.Context)
}

// Register adds the activities to r under their fixed names.
func (a *Activities) Register(r worker.ActivityRegistry) {
	r.RegisterActivityWithOptions(a.Barrier, activity.RegisterOptions{Name: BarrierActivity})
	r.RegisterActivityWithOptions(a.Classify, activity.RegisterOptions{Name: ClassifyActivity})
	r.RegisterActivityWithOptions(a.Batch, activity.RegisterOptions{Name: BatchActivity})
	r.RegisterActivityWithOptions(a.Ready, activity.RegisterOptions{Name: ReadyActivity})
}

// barrierState is the barrier's heartbeat details: the bounds of the wait set
// taken once on the first attempt (D9), when that was, and whether the slow
// alert has fired. The wait set is XIP ∪ [Lo, Hi): the snapshot's xip, and
// every xid from its xmax (Lo) up to the fence xid (Hi), exclusive. The range
// is kept as its bounds, never expanded, so the details stay a fixed size
// however many xids were handed out between the snapshot and the fence
// (ADR-006-E1 E1-1). Lo and Hi are empty in details written before E1-1,
// whose XIP already held the expanded range; an empty range waits on nothing.
// Transaction ids are not tenant data.
type barrierState struct {
	XIP   []string  `json:"xip"`
	Lo    string    `json:"lo,omitempty"`
	Hi    string    `json:"hi,omitempty"`
	Since time.Time `json:"since"`
	Slow  bool      `json:"slow"`
}

const (
	// snapshotSQL and fenceSQL, run in that order in one transaction, give
	// the set of transactions in progress at the snapshot. A transaction
	// that started after it sees every registry change committed before it
	// (records writers run READ COMMITTED, and the trigger's registry read is
	// a statement of a volatile plpgsql function, so it takes its own
	// snapshot).
	//
	// pg_snapshot_xip alone is not that set: it lists only xids below the
	// snapshot's xmax, and when the newest assigned xid is still running,
	// xmax is that xid, so it is not listed (found in TICK-745 testing: a
	// snapshot 1869:1869: with 1869 in progress). fenceSQL assigns this
	// transaction an xid M, after the snapshot, and the barrier also waits on
	// every xid in [xmax, M) still in progress: the xids handed out between
	// the two statements, which include any such newest one.
	snapshotSQL = `SELECT (SELECT coalesce(array_agg(x::text), '{}') FROM pg_snapshot_xip(s.s) x), pg_snapshot_xmax(s.s)::text
		FROM (SELECT pg_current_snapshot() AS s) s`
	fenceSQL = `SELECT pg_current_xact_id()::text`
	// openSQL counts the xids of XIP ∪ [Lo, Hi) still in progress; the range
	// is expanded here, per poll, never in Go. An empty Lo or Hi is an empty
	// range. pg_xact_status needs no grant.
	openSQL = `SELECT (SELECT count(*) FROM unnest($1::text[]) x WHERE pg_xact_status(x::xid8) = 'in progress')
		+ (SELECT count(*) FROM generate_series(nullif($2, '')::bigint, nullif($3, '')::bigint - 1) g(n)
			WHERE pg_xact_status(n::text::xid8) = 'in progress')`
)

// Barrier is the snapshot barrier (D9, R11). It takes pg_current_snapshot()
// and a fence xid once and keeps the bounds of the wait set in the heartbeat
// details, then polls it with pg_xact_status, once per poll, 1 s apart backing off
// to 10 s, until none is still open. Nothing sleeps in SQL, and no
// connection is held between polls. Past SlowAfter it raises
// field_index_barrier_slow once (metric and log, no tenant data) and keeps
// waiting.
//
// The polls run inside one activity attempt, heartbeating after each, not as
// failed attempts on a retry timer (TICK-745 design deviation): the worker
// guard records a non-allow-listed activity failure as a record_data deny
// with alert=true, so a retry-driven poll would page on every barrier. A
// worker lost mid-wait times the heartbeat out, and the next attempt resumes
// from the snapshot in the heartbeat details, never a new one.
func (a *Activities) Barrier(ctx context.Context) error {
	var st barrierState
	if !activity.HasHeartbeatDetails(ctx) || activity.GetHeartbeatDetails(ctx, &st) != nil || st.Since.IsZero() {
		var err error
		if st, err = a.takeSnapshot(ctx); err != nil {
			return fmt.Errorf("field index barrier: snapshot: %w", err)
		}
		activity.RecordHeartbeat(ctx, st)
	}
	c, isBackfill := childOf(ctx)
	wait := barrierFirstPoll
	for {
		// A backfill child's barrier stops waiting once its registry row is
		// removed, set to removing, or moved to another requested_version:
		// the next batch, or the flip to ready, writes nothing then, and a
		// removal runs its own barrier (D11). So a removal or a version bump
		// during a long wait ends it at the next poll, not when the long
		// transaction ends.
		if isBackfill == nil {
			gone, err := superseded(ctx, a.Pool, c)
			if err != nil {
				return fmt.Errorf("field index barrier: registry: %w", err)
			}
			if gone {
				return nil
			}
		}
		var open int64
		if err := a.Pool.QueryRow(ctx, openSQL, st.XIP, st.Lo, st.Hi).Scan(&open); err != nil {
			return fmt.Errorf("field index barrier: status: %w", err)
		}
		if open == 0 {
			return nil
		}
		if waited := time.Since(st.Since); !st.Slow && waited >= a.slowAfter() {
			st.Slow = true
			a.Metrics.barrierSlow(ctx)
			a.logger().WarnContext(ctx, "field index barrier slow: a transaction open at the flip is still running",
				a.attrs(ctx, slog.Bool("alert", true), slog.String("alert_name", "field_index_barrier_slow"),
					slog.Int64("open_transactions", open), slog.Duration("waited", waited))...)
		}
		activity.RecordHeartbeat(ctx, st)
		t := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			t.Stop()
			return ctx.Err()
		case <-t.C:
		}
		wait = min(wait*2, barrierMaxPoll)
	}
}

// takeSnapshot returns the wait set of the transactions in progress now: the
// snapshot's xip, its xmax as Lo, and as Hi a fence xid this transaction
// takes after it (see snapshotSQL). The transaction is rolled back; its only
// effect is the one xid it consumed.
func (a *Activities) takeSnapshot(ctx context.Context) (barrierState, error) {
	tx, err := a.Pool.Begin(ctx)
	if err != nil {
		return barrierState{}, err
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	st := barrierState{Since: time.Now()}
	if err := tx.QueryRow(ctx, snapshotSQL).Scan(&st.XIP, &st.Lo); err != nil {
		return barrierState{}, err
	}
	if a.afterSnapshot != nil {
		a.afterSnapshot(ctx)
	}
	if err := tx.QueryRow(ctx, fenceSQL).Scan(&st.Hi); err != nil {
		return barrierState{}, err
	}
	return st, nil
}

// supersededSQL reads the child's registry row, if any.
const supersededSQL = `SELECT state, requested_version FROM record_index_fields WHERE tenant_id = $1 AND id = $2`

// superseded reports whether c's registry row is gone, removing, or at
// another requested_version than c's pin. ready is not superseded: the
// barrier after the flip to ready still waits.
func superseded(ctx context.Context, pool *pgxpool.Pool, c child) (bool, error) {
	var state string
	var version int
	err := tenancy.InTenantTx(ctx, pool, c.tenant, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, supersededSQL, c.tenant, c.field).Scan(&state, &version)
	})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return true, nil
	case err != nil:
		return false, err
	}
	return fieldindex.State(state) == fieldindex.StateRemoving || (c.version != 0 && version != c.version), nil
}

// The barrier's poll interval: 1 s, doubling to 10 s (D9).
const (
	barrierFirstPoll = time.Second
	barrierMaxPoll   = 10 * time.Second
)

const classifySQL = `SELECT state FROM record_index_fields WHERE tenant_id = $1 AND id = $2`

// Classify tells the coordinator what a queued field needs, from its
// registry row: backfilling → a backfill, removing → a removal, ready or no
// row → nothing. The queued item's op and version are never trusted for this
// (TICK-745 guard adaptation): a replay of a finished item, or one carried
// across continue-as-new, finds the row as it is now.
func (a *Activities) Classify(ctx context.Context, in FieldRef) (Classified, error) {
	tenant, ok := workerguard.TenantFromContext(ctx)
	if !ok || in.FieldID == uuid.Nil {
		return Classified{}, temporal.NewNonRetryableApplicationError("field index: no tenant or field", ErrTypeNoChild, nil)
	}
	var state string
	err := tenancy.InTenantTx(ctx, a.Pool, tenant, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, classifySQL, tenant, in.FieldID).Scan(&state)
	})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return Classified{}, nil
	case err != nil:
		return Classified{}, fmt.Errorf("field index classify: %w", err)
	}
	f := in.FieldID
	switch fieldindex.State(state) {
	case fieldindex.StateBackfilling:
		return Classified{Backfill: &f}, nil
	case fieldindex.StateRemoving:
		return Classified{Remove: &f}, nil
	}
	return Classified{}, nil
}

// childOf parses the calling workflow's id as a backfill child's.
func childOf(ctx context.Context) (child, error) {
	c, err := parseChildID(activity.GetInfo(ctx).WorkflowExecution.ID, fieldindex.BackfillWorkflowType)
	if err != nil {
		return child{}, temporal.NewNonRetryableApplicationError("field index: "+err.Error(), ErrTypeNoChild, nil)
	}
	return c, nil
}

// Batch copies the next batch of the child's field (D10.2): one
// InTenantTx, see batch.go. The field and the pinned version come from the
// workflow id; the argument is only the cursor.
func (a *Activities) Batch(ctx context.Context, in Cursor) (Cursor, error) {
	c, err := childOf(ctx)
	if err != nil {
		return Cursor{}, err
	}
	next, err := copyBatch(ctx, a.Pool, c, in.Cursor, a.batchSize())
	if err != nil {
		return Cursor{}, fmt.Errorf("field index batch: %w", err)
	}
	return Cursor{Cursor: next}, nil
}

// Ready flips the child's registry row to ready with progress 100 (D10.6),
// only while it is still backfilling at the pinned version; otherwise it
// changes nothing.
func (a *Activities) Ready(ctx context.Context) error {
	c, err := childOf(ctx)
	if err != nil {
		return err
	}
	if err := flipReady(ctx, a.Pool, c); err != nil {
		return fmt.Errorf("field index ready: %w", err)
	}
	return nil
}

func (a *Activities) batchSize() int {
	if a.BatchSize <= 0 {
		return DefaultBatchSize
	}
	return a.BatchSize
}

func (a *Activities) slowAfter() time.Duration {
	if a.SlowAfter <= 0 {
		return DefaultSlowAfter
	}
	return a.SlowAfter
}

func (a *Activities) logger() *slog.Logger {
	if a.Log == nil {
		return slog.New(slog.DiscardHandler)
	}
	return a.Log
}

// attrs are a log line's ids: the tenant, the field when the caller is a
// child, and the workflow id. Never a value, a record id or a transaction id.
func (a *Activities) attrs(ctx context.Context, extra ...any) []any {
	out := make([]any, 0, 3+len(extra))
	if tenant, ok := workerguard.TenantFromContext(ctx); ok {
		out = append(out, slog.String("tenant_id", tenant.String()))
	}
	if activity.IsActivity(ctx) {
		id := activity.GetInfo(ctx).WorkflowExecution.ID
		if c, err := parseAnyChildID(id); err == nil {
			out = append(out, slog.String("field_id", c.field.String()))
		}
		out = append(out, slog.String("workflow_id", id))
	}
	return append(out, extra...)
}

// Metrics are the backfill's instruments. A nil *Metrics records nothing.
type Metrics struct {
	slow metric.Int64Counter
}

// NewMetrics creates the instruments on mp.
func NewMetrics(mp metric.MeterProvider) (*Metrics, error) {
	m := mp.Meter("github.com/Navaladi2019/TicketingSystem/internal/fieldindex/backfill")
	slow, err := m.Int64Counter("field_index_barrier_slow",
		metric.WithDescription("Field-index barriers still waiting after the slow threshold (15 min) for a transaction open at a registry flip (ADR-006 D9, R11). Alert on any increase; the tenant and field are in the log line, not here."))
	if err != nil {
		return nil, err
	}
	return &Metrics{slow: slow}, nil
}

func (m *Metrics) barrierSlow(ctx context.Context) {
	if m == nil {
		return
	}
	m.slow.Add(ctx, 1)
}

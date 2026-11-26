package backfill

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/metric"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"

	"github.com/Navaladi2019/TicketingSystem/internal/fieldindex"
	snapshot "github.com/Navaladi2019/TicketingSystem/internal/fieldindex/barrier"
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
// nava_worker pool. Register them with Register. Metrics, Uncoerced and Log
// may be nil.
type Activities struct {
	Pool    *pgxpool.Pool
	Metrics *Metrics
	// Uncoerced counts the values a batch skips because they do not coerce,
	// on the counter the records trigger's count goes to
	// (nava_field_index_uncoerced_total{entity_definition}, ADR-006 R5).
	Uncoerced UncoercedCounter
	Log       *slog.Logger
	// BatchSize is the records one batch copies; 0 is DefaultBatchSize.
	BatchSize int
	// SlowAfter is the barrier's slow threshold; 0 is DefaultSlowAfter.
	SlowAfter time.Duration

	// progress is the worker's progress memo (see batch.go (d)).
	progress progressMemo

	// afterSnapshot, when set, runs between the barrier's snapshot and its
	// fence (test-only: it widens [lo, hi)).
	afterSnapshot func(context.Context)
}

// UncoercedCounter is where a batch reports the values it skipped because
// they do not coerce: nava_field_index_uncoerced_total{entity_definition}.
// The worker passes the records pipeline's *pipeline.FieldIndexMetrics,
// the instance the trigger's count goes to, through
// internal/workflow/recordports: this package may not import the pipeline
// (D-3-6, TestRecordWritersOnlyInPipeline). AddUncoerced is called only
// after the batch's transaction has committed, with n > 0.
type UncoercedCounter interface {
	AddUncoerced(ctx context.Context, entityDefinition uuid.UUID, n int64)
}

// Register adds the activities to r under their fixed names.
func (a *Activities) Register(r worker.ActivityRegistry) {
	r.RegisterActivityWithOptions(a.Barrier, activity.RegisterOptions{Name: BarrierActivity})
	r.RegisterActivityWithOptions(a.Classify, activity.RegisterOptions{Name: ClassifyActivity})
	r.RegisterActivityWithOptions(a.Batch, activity.RegisterOptions{Name: BatchActivity})
	r.RegisterActivityWithOptions(a.Ready, activity.RegisterOptions{Name: ReadyActivity})
}

// Barrier is the snapshot barrier (D9, R11), fieldindex/barrier.Wait. A backfill
// child's barrier stops waiting once its registry row is removed, set to
// removing, or moved to another requested_version: the next batch, or the
// flip to ready, writes nothing then, and a removal runs its own barrier
// (D11). So a removal or a version bump during a long wait ends it at the
// next poll, not when the long transaction ends. Past SlowAfter it raises
// field_index_barrier_slow once (metric and log, no tenant data) and keeps
// waiting.
func (a *Activities) Barrier(ctx context.Context) error {
	o := snapshot.Options{SlowAfter: a.slowAfter(), AfterSnapshot: a.afterSnapshot,
		OnSlow: func(ctx context.Context, open int64, waited time.Duration) {
			a.Metrics.barrierSlow(ctx)
			a.logger().WarnContext(ctx, "field index barrier slow: a transaction open at the flip is still running",
				a.attrs(ctx, slog.Bool("alert", true), slog.String("alert_name", "field_index_barrier_slow"),
					slog.Int64("open_transactions", open), slog.Duration("waited", waited))...)
		}}
	if c, err := childOf(ctx); err == nil {
		o.StopIf = func(ctx context.Context) (bool, error) {
			gone, err := superseded(ctx, a.Pool, c)
			if err != nil {
				return false, fmt.Errorf("field index barrier: registry: %w", err)
			}
			return gone, nil
		}
	}
	if err := snapshot.Wait(ctx, a.Pool, o); err != nil {
		return fmt.Errorf("field index %w", err)
	}
	return nil
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
// workflow id; the argument is only the cursor. Once the batch has
// committed, the values it skipped because they do not coerce are added to
// Uncoerced; the count never goes into the result (CI-5c). A retry of a
// batch that committed but whose result was lost recopies the same records
// and counts them again: the counter is an alerting rate, at least once.
func (a *Activities) Batch(ctx context.Context, in Cursor) (Cursor, error) {
	c, err := childOf(ctx)
	if err != nil {
		return Cursor{}, err
	}
	res, err := copyBatch(ctx, a.Pool, c, in.Cursor, a.batchSize(), a.progress.get(c))
	if err != nil {
		return Cursor{}, fmt.Errorf("field index batch: %w", err)
	}
	a.progress.put(c, res)
	if a.Uncoerced != nil && res.uncoerced > 0 {
		a.Uncoerced.AddUncoerced(ctx, res.entityDefinition, res.uncoerced)
	}
	return Cursor{Cursor: res.next}, nil
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

// progressMemo holds, per child, the progress counts after its last batch
// on this worker, so the next batch adds its records to them instead of
// counting the type (TICK-991). It is a cache, never workflow state: a miss
// (another worker, a restart, a retried batch, a full memo) costs one count
// of the type. A child's entry goes when its last batch returns no cursor.
// One backfill runs per tenant at a time, so it holds about one entry per
// tenant backfilling; past progressMemoMax entries it is cleared.
type progressMemo struct {
	mu sync.Mutex
	m  map[child]progressCounts
}

const progressMemoMax = 4096

func (p *progressMemo) get(c child) *progressCounts {
	p.mu.Lock()
	defer p.mu.Unlock()
	pc, ok := p.m[c]
	if !ok {
		return nil
	}
	return &pc
}

func (p *progressMemo) put(c child, res batchResult) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if res.next == nil || res.progress == nil {
		delete(p.m, c)
		return
	}
	if p.m == nil || len(p.m) >= progressMemoMax {
		p.m = map[child]progressCounts{}
	}
	p.m[c] = *res.progress
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

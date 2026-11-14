package pipeline

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel/metric"

	"github.com/Navaladi2019/TicketingSystem/internal/records/numbering"
	"github.com/Navaladi2019/TicketingSystem/internal/records/store"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// DefaultAllocTimeout bounds a number allocation, pool acquire included
// (ADR-005 A5). Past it the create fails with ErrRetryable.
const DefaultAllocTimeout = 200 * time.Millisecond

// Allocation is a `gaps` type's number, as the allocator took it (ADR-005
// A2). The create checks every field against its own transaction and
// definition before it inserts the record.
type Allocation struct {
	// Tenant is the tenant the allocation ran under: the create
	// transaction's (tenancy.TenantOf), never a parameter.
	Tenant uuid.UUID
	// Definition is the entity type whose counter was taken.
	Definition uuid.UUID
	// ScopeKey is the counter's scope, as PostgreSQL computed it: '' for the
	// default, e<epoch> or e<epoch>|d:<date part> for a linked format
	// (ADR-005 §4.1).
	ScopeKey string
	// Epoch is the format epoch the allocation was made for.
	Epoch int
	// Number is the counter value taken.
	Number int64
	// Instant is the allocation's now(), the record's created_at and the
	// instant the scope key's date part was read at (A7).
	Instant time.Time
}

// Allocator takes a `gaps` type's number in a transaction of its own that
// commits before the create inserts the record (ADR-005 L2s), from the
// counter of format's scope key. Its tenant is tenancy.TenantOf(createTx).
// A TxRunner that is also an Allocator (a PoolRunner) is where the pipeline
// takes its numbers.
type Allocator interface {
	Allocate(ctx context.Context, createTx pgx.Tx, definition uuid.UUID, format numbering.Compiled) (Allocation, error)
}

// Allocate implements Allocator with tenancy.AllocTx on r.Pool, which must be
// the dedicated allocator pool (tenancy.AllocPoolConfig): on the pool the
// create's own transaction holds, concurrent creates can starve it. The
// statement computes the scope key, its date part and period_end from now()
// in the numbering time zone (format.Scope, ADR-005 A7). It gives up after
// r.AllocTimeout (DefaultAllocTimeout when zero) with ErrRetryable, and
// records r.Metrics when set. Nothing is retried: since M3 (TICK-670)
// record_counters has one unique index, the upsert's arbiter, so two first
// allocations of a counter serialise on it and the second takes the UPDATE
// arm (ADR-005-E1's race needed two).
func (r PoolRunner) Allocate(ctx context.Context, createTx pgx.Tx, definition uuid.UUID, format numbering.Compiled) (Allocation, error) {
	scope, err := format.Scope()
	if err != nil {
		return Allocation{}, fmt.Errorf("pipeline: %w", err)
	}
	timeout := r.AllocTimeout
	if timeout <= 0 {
		timeout = DefaultAllocTimeout
	}
	actx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	start := time.Now()
	var got store.Allocated
	info, err := tenancy.AllocTx(actx, r.Pool, createTx, func(b *pgx.Batch, tenant uuid.UUID) {
		store.QueueAllocate(b, tenant, definition, scope, &got)
	})
	timedOut := err != nil && errors.Is(actx.Err(), context.DeadlineExceeded) && ctx.Err() == nil
	r.Metrics.observe(ctx, info.Acquire, time.Since(start), timedOut)
	if err == nil {
		r.Metrics.fill(ctx, format, got.Number)
	}
	switch {
	case timedOut:
		return Allocation{}, fmt.Errorf("%w: the number allocation took longer than %v: %w", ErrRetryable, timeout, err)
	case err != nil:
		return Allocation{}, err
	}
	return Allocation{Tenant: info.Tenant, Definition: definition, ScopeKey: got.ScopeKey, Epoch: format.Epoch,
		Number: got.Number, Instant: got.Instant}, nil
}

// AllocMetrics are the allocator's metrics (ADR-005 A8). A nil
// *AllocMetrics records nothing. No attribute names a tenant or a type (T01).
type AllocMetrics struct {
	acquire, latency, fillRatio metric.Float64Histogram
	timeouts                    metric.Int64Counter
}

// allocBuckets are in seconds: the allocation's budget is 200 ms, and an
// uncontended one is about a round trip.
func allocBuckets() []float64 {
	return []float64{0.0005, 0.001, 0.0025, 0.005, 0.01, 0.025, 0.05, 0.1, 0.2, 0.5}
}

// NewAllocMetrics registers the allocator's metrics on mp:
// nava_numbering_alloc_acquire_seconds (the allocator pool's acquire wait),
// nava_numbering_alloc_seconds (the whole allocation, acquire included) and
// nava_numbering_alloc_timeouts (allocations past the timeout).
func NewAllocMetrics(mp metric.MeterProvider) (*AllocMetrics, error) {
	m := mp.Meter("github.com/Navaladi2019/TicketingSystem/internal/records/pipeline")
	acquire, err := m.Float64Histogram("nava_numbering_alloc_acquire_seconds",
		metric.WithDescription("Seconds a record-number allocation waited for a connection of the allocator pool."),
		metric.WithUnit("s"), metric.WithExplicitBucketBoundaries(allocBuckets()...))
	if err != nil {
		return nil, fmt.Errorf("pipeline: %w", err)
	}
	latency, err := m.Float64Histogram("nava_numbering_alloc_seconds",
		metric.WithDescription("Seconds a record-number allocation took, pool acquire included."),
		metric.WithUnit("s"), metric.WithExplicitBucketBoundaries(allocBuckets()...))
	if err != nil {
		return nil, fmt.Errorf("pipeline: %w", err)
	}
	timeouts, err := m.Int64Counter("nava_numbering_alloc_timeouts",
		metric.WithDescription("Record-number allocations that passed their timeout; the create failed with a retryable error."),
		metric.WithUnit("{allocation}"))
	if err != nil {
		return nil, fmt.Errorf("pipeline: %w", err)
	}
	fillRatio, err := m.Float64Histogram("nava_numbering_fill_ratio",
		metric.WithDescription("How full a refuse number format's {seq:N} is after a number was taken: the counter value over the largest N-digit value; above 1 the create was refused with number_space_exhausted."),
		metric.WithUnit("1"), metric.WithExplicitBucketBoundaries(fillBuckets()...))
	if err != nil {
		return nil, fmt.Errorf("pipeline: %w", err)
	}
	return &AllocMetrics{acquire: acquire, latency: latency, fillRatio: fillRatio, timeouts: timeouts}, nil
}

// fillBuckets are fill ratios. NumberSpaceNearlyExhausted
// (deploy/compose/prometheus/rules/numbering.yml) alerts on any
// observation above the 0.9 bucket: 90 % of the width (ADR-005 §3.5).
func fillBuckets() []float64 { return []float64{0.5, 0.75, 0.9, 0.95, 0.99, 1} }

// fill records how full format's counter is at value n, for a refuse format
// only: a growing one never runs out. Both numbering paths record it: a
// `gaps` allocation here, a `gapless` create through FillObserver.
func (m *AllocMetrics) fill(ctx context.Context, format numbering.Compiled, n int64) {
	if m == nil || format.Overflow != numbering.Refuse || n < 1 {
		return
	}
	largest := 1.0
	for range format.Width {
		largest *= 10
	}
	m.fillRatio.Record(context.WithoutCancel(ctx), float64(n)/(largest-1))
}

// FillObserver records the fill ratio of a number a `gapless` create took
// inside its own transaction (ADR-005 §3.5). A PoolRunner is one.
type FillObserver interface {
	ObserveFill(ctx context.Context, format numbering.Compiled, n int64)
}

// ObserveFill implements FillObserver on r.Metrics.
func (r PoolRunner) ObserveFill(ctx context.Context, format numbering.Compiled, n int64) {
	r.Metrics.fill(ctx, format, n)
}

func (m *AllocMetrics) observe(ctx context.Context, acquire, total time.Duration, timedOut bool) {
	if m == nil {
		return
	}
	// The create's context may be cancelled already; a metric is still owed.
	ctx = context.WithoutCancel(ctx)
	m.acquire.Record(ctx, acquire.Seconds())
	m.latency.Record(ctx, total.Seconds())
	if timedOut {
		m.timeouts.Add(ctx, 1)
	}
}

// exhausted is the refusal of a create whose counter value outgrew the
// format's width in scope (ADR-005 §3.5).
//
// TODO(TODO-S18-02): write the refusal's audit row on the runner, outside
// the create transaction, once its audit action is named.
func (w *write) exhausted(scopeKey string) error {
	return &ErrNumberSpaceExhausted{EntityKey: w.cmd.EntityKey, ScopeKey: scopeKey, Width: w.format.Width}
}

// allocate takes the create's number (ADR-005 L2s) on the runner and checks
// the allocation against the create's own transaction, type and format
// (A2): the scope key must be one of the format's epoch. It returns the
// allocation and the scope key's date part.
func (w *write) allocate(ctx context.Context) (Allocation, string, error) {
	a, ok := w.p.runner.(Allocator)
	if !ok {
		return Allocation{}, "", errors.New("pipeline: the TxRunner cannot allocate record numbers")
	}
	got, err := a.Allocate(ctx, w.tx, w.defID, w.format)
	if err != nil {
		return Allocation{}, "", fmt.Errorf("pipeline: %w", err)
	}
	datePart, keyErr := w.format.DatePartOf(got.ScopeKey)
	if got.Tenant != w.tenant || got.Definition != w.defID || keyErr != nil || got.Epoch != w.format.Epoch ||
		got.Number < 1 || got.Instant.IsZero() {
		return Allocation{}, "", fmt.Errorf("pipeline: the allocation (tenant %s, type %s, scope %q, epoch %d, number %d) is not this create's (tenant %s, type %s, epoch %d)",
			got.Tenant, got.Definition, got.ScopeKey, got.Epoch, got.Number, w.tenant, w.defID, w.format.Epoch)
	}
	return got, datePart, nil
}

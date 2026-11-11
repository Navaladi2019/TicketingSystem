package pipeline

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/Navaladi2019/TicketingSystem/internal/metadata"
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
	// ScopeKey is the counter's scope: '' until number formats (ADR-005 §3).
	ScopeKey string
	// Epoch is the format epoch: 0, today's {prefix}{seq:7}.
	Epoch int
	// Number is the counter value taken.
	Number int64
	// Instant is the allocation's now(), the record's created_at (A7).
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
// create's own transaction holds, concurrent creates can starve it. It gives
// up after r.AllocTimeout (DefaultAllocTimeout when zero) with ErrRetryable,
// and records r.Metrics when set.
//
// While both unique indexes of record_counters exist (ADR-005 M2 to M3), a
// type's first allocation can lose the race for the counter row with 23505
// on the index the upsert does not name (store.CounterRace, ADR-005-E1). It
// is then retried once, in a fresh AllocTx: the winner has committed, so the
// second upsert takes the UPDATE arm. No other error is retried, and the
// retry runs within the same timeout.
func (r PoolRunner) Allocate(ctx context.Context, createTx pgx.Tx, definition uuid.UUID, format numbering.Compiled) (Allocation, error) {
	timeout := r.AllocTimeout
	if timeout <= 0 {
		timeout = DefaultAllocTimeout
	}
	actx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	start := time.Now()
	scope := format.ScopeKey()
	var got store.Allocated
	alloc := func() (tenancy.AllocInfo, error) {
		return tenancy.AllocTx(actx, r.Pool, createTx, func(b *pgx.Batch, tenant uuid.UUID) {
			store.QueueAllocate(b, tenant, definition, scope, &got)
		})
	}
	info, err := alloc()
	if store.CounterRace(err) {
		r.Metrics.retried(ctx, metadata.NumberingGaps)
		var again tenancy.AllocInfo
		again, err = alloc()
		again.Acquire += info.Acquire
		info = again
	}
	timedOut := err != nil && errors.Is(actx.Err(), context.DeadlineExceeded) && ctx.Err() == nil
	r.Metrics.observe(ctx, info.Acquire, time.Since(start), timedOut)
	switch {
	case timedOut:
		return Allocation{}, fmt.Errorf("%w: the number allocation took longer than %v: %w", ErrRetryable, timeout, err)
	case err != nil:
		return Allocation{}, err
	}
	return Allocation{Tenant: info.Tenant, Definition: definition, ScopeKey: scope, Epoch: format.Epoch,
		Number: got.Number, Instant: got.Instant}, nil
}

// AllocMetrics are the allocator's metrics (ADR-005 A8). A nil
// *AllocMetrics records nothing. No attribute names a tenant or a type (T01).
type AllocMetrics struct {
	acquire, latency  metric.Float64Histogram
	timeouts, retries metric.Int64Counter
}

// allocBuckets are in seconds: the allocation's budget is 200 ms, and an
// uncontended one is about a round trip.
func allocBuckets() []float64 {
	return []float64{0.0005, 0.001, 0.0025, 0.005, 0.01, 0.025, 0.05, 0.1, 0.2, 0.5}
}

// NewAllocMetrics registers the allocator's metrics on mp:
// nava_numbering_alloc_acquire_seconds (the allocator pool's acquire wait),
// nava_numbering_alloc_seconds (the whole allocation, acquire included),
// nava_numbering_alloc_timeouts (allocations past the timeout) and
// nava_numbering_first_create_retries (upserts retried after the expand
// window's first-create race, by numbering mode: ADR-005-E1).
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
	retries, err := m.Int64Counter("nava_numbering_first_create_retries",
		metric.WithDescription("Record-number upserts retried once after a unique violation on the record_counters index they do not name (the first create of a counter while both unique indexes exist)."),
		metric.WithUnit("{retry}"))
	if err != nil {
		return nil, fmt.Errorf("pipeline: %w", err)
	}
	return &AllocMetrics{acquire: acquire, latency: latency, timeouts: timeouts, retries: retries}, nil
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

// retried counts one first-create retry of a type in mode (gaps or
// gapless). The mode is the only attribute: no tenant, no type (T01).
func (m *AllocMetrics) retried(ctx context.Context, mode metadata.NumberingMode) {
	if m == nil {
		return
	}
	m.retries.Add(context.WithoutCancel(ctx), 1, metric.WithAttributes(attribute.String("mode", string(mode))))
}

// allocMetrics is the AllocMetrics of the pipeline's runner, when it is a
// PoolRunner (or embeds one); nil, which records nothing, otherwise.
func (p *Pipeline) allocMetrics() *AllocMetrics {
	if r, ok := p.runner.(interface{ allocMetrics() *AllocMetrics }); ok {
		return r.allocMetrics()
	}
	return nil
}

func (r PoolRunner) allocMetrics() *AllocMetrics { return r.Metrics }

// allocate takes the create's number (ADR-005 L2s) on the runner and checks
// the allocation against the create's own transaction and type (A2).
func (w *write) allocate(ctx context.Context) (Allocation, error) {
	a, ok := w.p.runner.(Allocator)
	if !ok {
		return Allocation{}, errors.New("pipeline: the TxRunner cannot allocate record numbers")
	}
	format := w.format()
	got, err := a.Allocate(ctx, w.tx, w.defID, format)
	if err != nil {
		return Allocation{}, fmt.Errorf("pipeline: %w", err)
	}
	if got.Tenant != w.tenant || got.Definition != w.defID || got.ScopeKey != format.ScopeKey() || got.Epoch != format.Epoch ||
		got.Number < 1 || got.Instant.IsZero() {
		return Allocation{}, fmt.Errorf("pipeline: the allocation (tenant %s, type %s, scope %q, epoch %d, number %d) is not this create's (tenant %s, type %s)",
			got.Tenant, got.Definition, got.ScopeKey, got.Epoch, got.Number, w.tenant, w.defID)
	}
	return got, nil
}

// format is the create's compiled number format (ADR-005 §3.1). Every type
// has the default until number formats ship (S18), when it is read from the
// definition version's snapshot instead.
func (w *write) format() numbering.Compiled {
	return numbering.Default()
}

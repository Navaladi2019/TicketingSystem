//go:build integration

// TICK-657 (P3-36e): PLAN-003 D-3-9 as amended 2026-11-10, gaps allowed and
// no reclaim, built as ADR-005's L2s allocator. A create takes its number
// with tenancy.AllocTx on the dedicated allocator pool, in one pipelined
// transaction of its own that commits before the record is inserted, and
// only after the replay check, validation and the write check (ADR-001 I4,
// ADR-005 A6). The allocator pool is never the pool the create's own
// transaction holds.

package pipeline_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/entitytypes"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/records/numbering"
	"github.com/Navaladi2019/TicketingSystem/internal/records/pipeline"
	"github.com/Navaladi2019/TicketingSystem/internal/records/store"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// counterNext is record_counters.next of the tenant's type key, as the
// superuser reads it: 0 when the type has no counter row yet.
func counterNext(t *testing.T, e env, tenant uuid.UUID, key string) int {
	t.Helper()
	return count(t, e.super, `SELECT coalesce((SELECT c.next FROM record_counters c
  JOIN entity_definitions d ON d.tenant_id = c.tenant_id AND d.id = c.entity_definition_id
  WHERE c.tenant_id = $1 AND d.key = $2), 0)`, tenant, key)
}

// AC: a rollback after the number was taken leaves a gap, and the next create
// gets a higher number. The rolled-back number is never reused.
func TestRolledBackCreateLeavesAGap(t *testing.T) {
	e := migrated(t)
	f := seedTenant(t, e)
	p := newPipeline(e, nil)
	ctx := context.Background()
	mustCreate(t, e, p, f, map[string]any{"title": "one"})

	rollback := errors.New("roll back")
	var lost pipeline.Result
	err := tenancy.InTenantTx(ctx, e.app, f.id, func(tx pgx.Tx) error {
		var err error
		if lost, err = p.Write(ctx, tx, createCmd(f, map[string]any{"title": "rolled back"})); err != nil {
			return err
		}
		return rollback
	})
	if !errors.Is(err, rollback) || lost.Number != "INC0000002" {
		t.Fatalf("rolled-back create = %+v, %v; want INC0000002 then the rollback", lost, err)
	}
	// The number's own transaction committed: the counter stays moved.
	if n := counterNext(t, e, f.id, entityKey); n != 3 {
		t.Fatalf("counter next = %d after the rollback, want 3", n)
	}
	if next := mustCreate(t, e, p, f, map[string]any{"title": "two"}); next.Number != "INC0000003" {
		t.Fatalf("next create got %s, want INC0000003 (2 is the gap)", next.Number)
	}
	if got, want := numbers(t, e, f.id, entityKey), []string{"INC0000001", "INC0000003"}; !slices.Equal(got, want) {
		t.Fatalf("numbers = %v, want %v", got, want)
	}
}

// AC (I4): an unauthorised create and an invalid create leave the counter
// unchanged, as a unique violation does (TestUniqueProperties).
func TestRefusedCreatesLeaveTheCounter(t *testing.T) {
	e := migrated(t)
	f := seedTenant(t, e)
	p := newPipeline(e, nil)
	ctx := context.Background()
	mustCreate(t, e, p, f, map[string]any{"title": "one"})
	before := counterNext(t, e, f.id, entityKey)

	// A principal of the tenant the published ACL does not admit.
	stranger := identity.Principal{TenantID: f.id, PrincipalID: uuid.New()}
	mustExec(t, e.super, `INSERT INTO principals (tenant_id, id, keycloak_user_id, enabled, synced_at)
		VALUES ($1, $2, $3, true, now())`, f.id, stranger.PrincipalID, "kc-"+stranger.PrincipalID.String())
	refused := createCmd(f, map[string]any{"title": "not mine"})
	refused.Actor = stranger
	if _, err := write(ctx, e, p, f.id, refused); !errors.Is(err, authz.ErrForbidden) {
		t.Fatalf("stranger's create: %v, want authz.ErrForbidden", err)
	}
	if n := counterNext(t, e, f.id, entityKey); n != before {
		t.Fatalf("counter next = %d after an unauthorised create, want %d", n, before)
	}

	var ve *pipeline.ErrValidation
	if _, err := write(ctx, e, p, f.id, createCmd(f, map[string]any{"title": ""})); !errors.As(err, &ve) {
		t.Fatalf("invalid create: %v, want *ErrValidation", err)
	}
	if n := counterNext(t, e, f.id, entityKey); n != before {
		t.Fatalf("counter next = %d after an invalid create, want %d", n, before)
	}
	if next := mustCreate(t, e, p, f, map[string]any{"title": "two"}); next.Number != "INC0000002" {
		t.Fatalf("next create got %s, want INC0000002: the refused creates took no number", next.Number)
	}
}

// allocRecorder is the pipeline's PoolRunner with every allocation it
// returned kept.
type allocRecorder struct {
	pipeline.PoolRunner
	mu     sync.Mutex
	allocs []pipeline.Allocation
}

func (r *allocRecorder) Allocate(ctx context.Context, createTx pgx.Tx, def uuid.UUID, format numbering.Compiled) (pipeline.Allocation, error) {
	a, err := r.PoolRunner.Allocate(ctx, createTx, def, format)
	if err == nil {
		r.mu.Lock()
		r.allocs = append(r.allocs, a)
		r.mu.Unlock()
	}
	return a, err
}

func (r *allocRecorder) taken() []pipeline.Allocation {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.allocs)
}

func recordedPipeline(r *allocRecorder) *pipeline.Pipeline {
	return pipeline.New(authz.ACL{Policies: authz.PolicyStore{}}, entitytypes.Store{}, pipeline.NoopSink{}, r, pipeline.SystemClock{})
}

// AC: an idempotent replay returns the original number and allocates
// nothing.
func TestCreateReplayTakesNoNumber(t *testing.T) {
	e := migrated(t)
	f := seedTenant(t, e)
	runner := &allocRecorder{PoolRunner: e.runner()}
	p := recordedPipeline(runner)
	ctx := context.Background()
	cmd := createCmd(f, map[string]any{"title": "keyed"})
	cmd.Actor, cmd.Source, cmd.IdempotencyKey, cmd.Causation = f.system(), pipeline.SourceWorkflow, "wf:gap", workflowCausation()

	first, err := write(ctx, e, p, f.id, cmd)
	if err != nil || first.Number != "INC0000001" || len(runner.taken()) != 1 {
		t.Fatalf("first = %+v, %v, %d allocations; want INC0000001 and 1", first, err, len(runner.taken()))
	}
	replay, err := write(ctx, e, p, f.id, cmd)
	if err != nil || replay != first {
		t.Fatalf("replay = %+v, %v; want %+v", replay, err, first)
	}
	if n := len(runner.taken()); n != 1 {
		t.Fatalf("%d allocations after the replay, want 1: the replay took a number", n)
	}
	if n := counterNext(t, e, f.id, entityKey); n != 2 {
		t.Fatalf("counter next = %d after the replay, want 2", n)
	}
}

// AC (ADR-005 A2, A7): each allocation runs under the create transaction's
// tenant, for the create's type, at scope ” and epoch 0, and the record is
// stored with the allocation's instant as created_at and updated_at.
func TestAllocationIsTheCreatesAndDatesTheRecord(t *testing.T) {
	e := migrated(t)
	a, b := seedTenant(t, e), seedTenant(t, e)
	runner := &allocRecorder{PoolRunner: e.runner()}
	p := recordedPipeline(runner)
	fixtures := []tenantFixture{a, b, a}
	var made []pipeline.Result
	for _, f := range fixtures {
		made = append(made, mustCreate(t, e, p, f, map[string]any{"title": "x"}))
	}
	got := runner.taken()
	if len(got) != len(fixtures) {
		t.Fatalf("%d allocations, want %d", len(got), len(fixtures))
	}
	for i, f := range fixtures {
		al := got[i]
		if al.Tenant != f.id || al.Definition != *definitionID(t, e, f.id) || al.ScopeKey != "" || al.Epoch != 0 {
			t.Fatalf("allocation %d = %+v, want tenant %s, its type, scope '' and epoch 0", i, al, f.id)
		}
		var created, updated time.Time
		if err := e.super.QueryRow(context.Background(), `SELECT created_at, updated_at FROM records WHERE id = $1`,
			made[i].RecordID).Scan(&created, &updated); err != nil {
			t.Fatal(err)
		}
		if !created.Equal(al.Instant) || !updated.Equal(al.Instant) {
			t.Fatalf("record %d: created_at %v, updated_at %v; want the allocation's instant %v", i, created, updated, al.Instant)
		}
	}
}

// AC: the allocation transaction is tenant-scoped. AllocTx takes the tenant
// from the create's own transaction, so under RLS a tenant A allocation can
// neither take nor move a number of tenant B's type.
func TestAllocationTransactionIsTenantScoped(t *testing.T) {
	e := migrated(t)
	a, b := seedTenant(t, e), seedTenant(t, e)
	p := newPipeline(e, nil)
	ctx := context.Background()
	mustCreate(t, e, p, b, map[string]any{"title": "x"}) // tenant B's counter row: next = 2
	defB := *definitionID(t, e, b.id)

	err := tenancy.InTenantTx(ctx, e.app, a.id, func(createTx pgx.Tx) error {
		var got store.Allocated
		_, err := tenancy.AllocTx(ctx, e.alloc, createTx, func(batch *pgx.Batch, tenant uuid.UUID) {
			if tenant != a.id {
				t.Errorf("AllocTx handed the queue tenant %s, want the create's %s", tenant, a.id)
			}
			// A forged tenant: B's counter, under A's setting.
			store.QueueAllocate(batch, b.id, defB, numbering.Scope{}, &got)
		})
		return err
	})
	if err == nil {
		t.Fatal("a tenant A allocation took a number of tenant B's type")
	}
	if n := counterNext(t, e, b.id, entityKey); n != 2 {
		t.Fatalf("tenant B counter next = %d, want 2", n)
	}
	if got := numbers(t, e, b.id, entityKey); !slices.Equal(got, wantNumbers(t, "INC", 1, 1)) {
		t.Fatalf("tenant B numbers = %v, want INC0000001", got)
	}
}

// AC (ADR-005 A3): the allocator pool refuses a role that bypasses RLS.
// nava_app given BYPASSRLS after the pool was configured is refused as each
// new connection opens, and the create fails without a number.
func TestAllocatorPoolRefusesABypassRLSRole(t *testing.T) {
	e := migrated(t)
	f := seedTenant(t, e)
	mustExec(t, e.super, "ALTER ROLE nava_app BYPASSRLS")
	t.Cleanup(func() { _, _ = e.super.Exec(context.Background(), "ALTER ROLE nava_app NOBYPASSRLS") })
	e.alloc = allocPool(t, appDSN(t, e.platform)) // fresh: no connection opened yet
	_, err := write(context.Background(), e, newPipeline(e, nil), f.id, createCmd(f, map[string]any{"title": "x"}))
	if err == nil || !strings.Contains(err.Error(), "nava_app") {
		t.Fatalf("create on a BYPASSRLS allocator pool: %v, want the allocator role refused", err)
	}
	if n := counterNext(t, e, f.id, entityKey); n != 0 {
		t.Fatalf("counter next = %d, want 0: no number taken", n)
	}
}

// AC (ADR-005 A5, A8): an allocation that waits past its timeout fails the
// create with ErrRetryable, writes nothing, and is counted; acquire wait and
// latency are recorded for every allocation.
func TestAllocationTimeoutIsRetryableAndMeasured(t *testing.T) {
	e := migrated(t)
	f := seedTenant(t, e)
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	m, err := pipeline.NewAllocMetrics(mp)
	if err != nil {
		t.Fatal(err)
	}
	runner := e.runner()
	runner.AllocTimeout, runner.Metrics = 100*time.Millisecond, m
	p := pipeline.New(authz.ACL{Policies: authz.PolicyStore{}}, entitytypes.Store{}, pipeline.NoopSink{}, runner, pipeline.SystemClock{})
	ctx := context.Background()
	mustCreate(t, e, p, f, map[string]any{"title": "one"})

	// Hold the counter row's lock, so the next allocation waits on it.
	holder, err := e.super.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = holder.Rollback(ctx) }()
	if _, err := holder.Exec(ctx, `SELECT 1 FROM record_counters WHERE tenant_id = $1 FOR UPDATE`, f.id); err != nil {
		t.Fatal(err)
	}
	_, err = write(ctx, e, p, f.id, createCmd(f, map[string]any{"title": "late"}))
	if !errors.Is(err, pipeline.ErrRetryable) {
		t.Fatalf("create behind a held counter: %v, want ErrRetryable", err)
	}
	if err := holder.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	if got := numbers(t, e, f.id, entityKey); !slices.Equal(got, wantNumbers(t, "INC", 1, 1)) {
		t.Fatalf("numbers = %v, want only INC0000001", got)
	}
	// The retry takes a new number: INC0000002, or INC0000003 when the
	// timed-out allocation, already sent in full, still committed after the
	// lock was released and before its cancel arrived. That is a gap, never
	// a number on two records.
	next := mustCreate(t, e, p, f, map[string]any{"title": "retry"})
	if next.Number != "INC0000002" && next.Number != "INC0000003" {
		t.Fatalf("retry got %s, want INC0000002 or INC0000003", next.Number)
	}
	if got := numbers(t, e, f.id, entityKey); !slices.Equal(got, []string{"INC0000001", next.Number}) {
		t.Fatalf("numbers = %v, want INC0000001 and the retry's %s", got, next.Number)
	}

	var rm metricdata.ResourceMetrics
	if err := reader.Collect(ctx, &rm); err != nil {
		t.Fatal(err)
	}
	counts := map[string]uint64{}
	var timeouts int64
	for _, sm := range rm.ScopeMetrics {
		for _, mt := range sm.Metrics {
			switch d := mt.Data.(type) {
			case metricdata.Histogram[float64]:
				for _, dp := range d.DataPoints {
					counts[mt.Name] += dp.Count
				}
			case metricdata.Sum[int64]:
				for _, dp := range d.DataPoints {
					timeouts += dp.Value
				}
			}
		}
	}
	if counts["nava_numbering_alloc_acquire_seconds"] != 3 || counts["nava_numbering_alloc_seconds"] != 3 || timeouts != 1 {
		t.Fatalf("metrics: histograms %v, timeouts %d; want 3 acquires, 3 latencies, 1 timeout", counts, timeouts)
	}
}

// barrierRunner holds each allocation until n of them are waiting, so n
// creates hold their own transactions' connections at once when they ask for
// a number: the worst case for a pool both share.
type barrierRunner struct {
	pipeline.PoolRunner
	n       int
	mu      sync.Mutex
	waiting int
	release chan struct{}
}

func (r *barrierRunner) Allocate(ctx context.Context, createTx pgx.Tx, def uuid.UUID, format numbering.Compiled) (pipeline.Allocation, error) {
	r.mu.Lock()
	if r.release == nil {
		r.release = make(chan struct{})
	}
	ch := r.release
	if r.waiting++; r.waiting == r.n {
		close(ch)
		r.waiting, r.release = 0, nil
	}
	r.mu.Unlock()
	select {
	case <-ch:
	case <-ctx.Done():
		return pipeline.Allocation{}, ctx.Err()
	}
	return r.PoolRunner.Allocate(ctx, createTx, def, format)
}

// starve runs 2 x n concurrent creates whose allocations each wait until n
// of them hold their create transactions, and returns every error.
func starve(e env, f tenantFixture, alloc pipeline.PoolRunner, n int) []error {
	p := pipeline.New(authz.ACL{Policies: authz.PolicyStore{}}, entitytypes.Store{}, pipeline.NoopSink{},
		&barrierRunner{PoolRunner: alloc, n: n}, pipeline.SystemClock{})
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	errs := make([]error, 2*n)
	var wg sync.WaitGroup
	for i := range errs {
		wg.Go(func() {
			_, errs[i] = write(ctx, e, p, f.id, createCmd(f, map[string]any{"title": fmt.Sprint(i)}))
		})
	}
	wg.Wait()
	return errs
}

func allStarved(t *testing.T, errs []error) {
	t.Helper()
	for i, err := range errs {
		if !errors.Is(err, pipeline.ErrRetryable) {
			t.Fatalf("create %d on a shared pool: %v, want it starved (ErrRetryable after the allocation timeout)", i, err)
		}
	}
}

// The subtests that must not starve run on noLatencyBudget
// (harness_integration_test.go). The controls keep the production timeout:
// their starvation is what they assert.

// AC (scope 3, ADR-005 A5): no pool starvation at pool size x 2 concurrent
// creates, each holding its create connection while it waits for a number.
// Direct: an API pool of 4. Through PgBouncer: an API pool as wide as the
// pooler's server pool (5), so the open creates hold every server
// connection of the nava entry. On the allocator pool (through nava_alloc)
// all finish; the controls, on the API pool or through the nava entry,
// starve and fail fast with ErrRetryable, so the test catches an allocator
// wired back onto either.
func TestAllocatorPoolDoesNotStarveThePool(t *testing.T) {
	base := migrated(t)
	f := seedTenant(t, base)
	dsn := appDSN(t, base.platform)
	small := func(size int32) env {
		cfg := serverPoolConfig(t, dsn)
		cfg.MaxConns = size
		return env{super: base.super, app: newPool(t, cfg), platform: base.platform}
	}
	// No warm-up: the first run's creates race the first insert of the
	// counter row while both unique indexes exist (M2 to M3), which R1's one
	// retry absorbs (ADR-005-E1, TICK-666).
	const direct = 4
	t.Run("direct, allocator pool", func(t *testing.T) {
		noErrors(t, starve(small(direct), f, pipeline.PoolRunner{Pool: allocPool(t, dsn), AllocTimeout: noLatencyBudget}, direct))
	})
	t.Run("direct, control: the API pool", func(t *testing.T) {
		e := small(direct)
		allStarved(t, starve(e, f, pipeline.PoolRunner{Pool: e.app}, direct))
	})

	const serverPool = 5 // scripts/ci/pgbouncer/pooled.ini default_pool_size
	bounced := throughBouncer(t, base, serverPool)
	t.Run("PgBouncer, allocator through nava_alloc", func(t *testing.T) {
		noErrors(t, starve(bounced, f, pipeline.PoolRunner{Pool: bounced.alloc, AllocTimeout: noLatencyBudget}, serverPool))
	})
	t.Run("PgBouncer, control: allocator through the nava entry", func(t *testing.T) {
		shared := allocPool(t, bounced.app.Config().ConnString())
		allStarved(t, starve(bounced, f, pipeline.PoolRunner{Pool: shared}, serverPool))
	})
	if got := numbers(t, base, f.id, entityKey); !slices.Equal(got, wantNumbers(t, "INC", 1, 2*direct+2*serverPool)) {
		t.Fatalf("numbers = %v, want INC0000001..%d from the allocator pool runs only", got, 2*direct+2*serverPool)
	}
}

//go:build integration

// TICK-872 (PLAN-005-E9 T-E9-3, D-E9-3; ADR-004-E9 Consequences "[PG] a
// compile-time or test guard for the allocator-pool rule"; docs/budgets.md
// "no synchronous-commit transaction runs on the allocator pool"): no create
// path, success or number_space_exhausted refusal, `gaps` or `gapless`, runs
// a transaction on the allocator pool other than an allocation.
//
// The guard is a test, not a type split: the allocator runner stays a
// TxRunner so rejectTooDeep keeps its one own transaction there (RS-E9-3),
// the documented exception pinned by the cascade_too_deep subtest.
//
// Two independent checks on the allocator pool, so the guard holds whether
// a future change goes through the runner or around it:
//   - the runner: every TxRunner.InTenantTx call is counted (Allocate, the
//     allowed path, is not);
//   - the pool: every round trip on it is traced (allocPoolTracer), and
//     each must be an AllocTx batch (SET LOCAL synchronous_commit = off)
//     that never names audit_log.

package pipeline_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/internal/audit"
	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/entitytypes"
	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/records/numbering"
	"github.com/Navaladi2019/TicketingSystem/internal/records/pipeline"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// allocPoolTracer classifies every round trip on the allocator pool. An
// allocation is one batch that tenancy.AllocTenant recognises (BEGIN; the
// tenant; SET LOCAL synchronous_commit = off; ...; COMMIT) and that never
// names audit_log. The one other statement allowed is a read outside any
// transaction (the pool's role check on connect, tenancy.CheckAllocatorRole).
// Anything else, a BEGIN of tenancy.InTenantTx included, is a violation.
type allocPoolTracer struct {
	prev   pgx.QueryTracer
	mu     sync.Mutex
	allocs int
	bad    []string
}

func (a *allocPoolTracer) TraceQueryStart(ctx context.Context, c *pgx.Conn, d pgx.TraceQueryStartData) context.Context {
	sql := strings.TrimSpace(d.SQL)
	read := strings.HasPrefix(strings.ToUpper(sql), "SELECT") && !strings.Contains(sql, "audit_log")
	if !read || c.PgConn().TxStatus() != 'I' {
		a.mu.Lock()
		a.bad = append(a.bad, sql)
		a.mu.Unlock()
	}
	if a.prev != nil {
		return a.prev.TraceQueryStart(ctx, c, d)
	}
	return ctx
}

func (a *allocPoolTracer) TraceQueryEnd(ctx context.Context, c *pgx.Conn, d pgx.TraceQueryEndData) {
	if a.prev != nil {
		a.prev.TraceQueryEnd(ctx, c, d)
	}
}

// TraceBatchStart implements pgx.BatchTracer: the whole batch is classified
// before it is sent.
func (a *allocPoolTracer) TraceBatchStart(ctx context.Context, _ *pgx.Conn, d pgx.TraceBatchStartData) context.Context {
	sqls := make([]string, 0, d.Batch.Len())
	for _, q := range d.Batch.QueuedQueries {
		sqls = append(sqls, strings.TrimSpace(q.SQL))
	}
	joined := strings.Join(sqls, " ; ")
	_, isAlloc := tenancy.AllocTenant(d.Batch)
	a.mu.Lock()
	if isAlloc && !strings.Contains(joined, "audit_log") {
		a.allocs++
	} else {
		a.bad = append(a.bad, "batch: "+joined)
	}
	a.mu.Unlock()
	return ctx
}

// TraceBatchQuery implements pgx.BatchTracer.
func (*allocPoolTracer) TraceBatchQuery(context.Context, *pgx.Conn, pgx.TraceBatchQueryData) {}

// TraceBatchEnd implements pgx.BatchTracer.
func (*allocPoolTracer) TraceBatchEnd(context.Context, *pgx.Conn, pgx.TraceBatchEndData) {}

// reset forgets everything traced so far.
func (a *allocPoolTracer) reset() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.allocs, a.bad = 0, nil
}

// seen is what ran on the pool since the last reset.
func (a *allocPoolTracer) seen() (allocs int, bad []string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.allocs, append([]string(nil), a.bad...)
}

// tracedAllocPool is allocPool's allocator pool with a statementTracer.
func tracedAllocPool(t *testing.T, dsn string) (*pgxpool.Pool, *allocPoolTracer) {
	t.Helper()
	cfg := serverPoolConfig(t, dsn)
	cfg.MaxConns = allocatorMaxConns
	if err := tenancy.AllocPoolConfig(cfg); err != nil {
		t.Fatal(err)
	}
	tr := &allocPoolTracer{prev: cfg.ConnConfig.Tracer}
	cfg.ConnConfig.Tracer = tr
	return newPool(t, cfg), tr
}

// guardRuns is the runner's InTenantTx count, read under its lock.
func guardRuns(r *recordingRunner) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.runs
}

// wantAllocationsOnly fails unless, since the tracer was last reset, the
// runner has run want InTenantTx calls in all, the pool
// ran exactly allocs allocations, and, when want is 0, nothing else. step
// names what ran.
func wantAllocationsOnly(t *testing.T, step string, r *recordingRunner, tr *allocPoolTracer, want, allocs int) {
	t.Helper()
	if n := guardRuns(r); n != want {
		t.Errorf("%s: %d InTenantTx calls on the allocator runner, want %d (docs/budgets.md: no write transaction but an allocation runs on the allocator pool)", step, n, want)
	}
	got, bad := tr.seen()
	if got != allocs {
		t.Errorf("%s: %d allocations on the allocator pool, want %d (is the guard watching the pool the pipeline allocates on?)", step, got, allocs)
	}
	if want == 0 {
		for _, sql := range bad {
			t.Errorf("%s: a statement other than an allocation ran on the allocator pool: %s", step, sql)
		}
	}
}

// TestAllocatorPoolRunsOnlyAllocations is the allocator-pool guard: a
// successful and a refused create on each numbering mode run no
// InTenantTx on the allocator runner and no transaction but an allocation
// on its pool, and each refusal's R2b entry is written, once, by the caller.
func TestAllocatorPoolRunsOnlyAllocations(t *testing.T) {
	for _, mode := range modes {
		t.Run(mode, func(t *testing.T) {
			e := migrated(t)
			f := seedFor(t, e, mode)
			linkFormat(t, e, f, "{prefix}-{seq:1}", numbering.Refuse, "")
			alloc, tr := tracedAllocPool(t, appDSN(t, e.platform))
			runner := &recordingRunner{PoolRunner: pipeline.PoolRunner{Pool: alloc, AllocTimeout: noLatencyBudget}}
			p := pipeline.New(authz.ACL{Policies: authz.PolicyStore{}}, entitytypes.Store{}, pipeline.NoopSink{}, runner, pipeline.SystemClock{})
			// A `gaps` create allocates once on the pool; a `gapless` one
			// takes its number in the create's transaction, never there.
			allocs := map[string]int{"gaps": 1, "gapless": 0}[mode]

			// Success: the first create, then the rest of the format's nine.
			tr.reset()
			if n := mustCreate(t, e, p, f, map[string]any{"title": "1"}).Number; n != "INC-1" {
				t.Fatalf("create 1: number %s", n)
			}
			wantAllocationsOnly(t, "successful create", runner, tr, 0, allocs)
			tr.reset()
			for i := 2; i <= 9; i++ {
				if n := mustCreate(t, e, p, f, map[string]any{"title": fmt.Sprint(i)}).Number; n != fmt.Sprintf("INC-%d", i) {
					t.Fatalf("create %d: number %s", i, n)
				}
			}
			wantAllocationsOnly(t, "eight more successful creates", runner, tr, 0, 8*allocs)

			// Refusal: number_space_exhausted. refuseAndAppend fails unless
			// Write appended nothing and the caller's own append, on the API
			// pool after the create ended, left exactly one more entry.
			tr.reset()
			before := entries(t, e, f.id, string(audit.ActionNumberSpaceExhausted))
			_, ex := refuseAndAppend(t, e, p, e.app, f)
			wantAllocationsOnly(t, "refused create", runner, tr, 0, allocs)
			if ex.Entry.Action != audit.ActionNumberSpaceExhausted {
				t.Errorf("refusal entry action %q, want %q", ex.Entry.Action, audit.ActionNumberSpaceExhausted)
			}
			if n := entries(t, e, f.id, string(audit.ActionNumberSpaceExhausted)); n != before+1 {
				t.Errorf("%d number_space_exhausted entries, want %d: the refusal's R2b entry is missing or doubled", n, before+1)
			}
			if n := len(numbers(t, e, f.id, entityKey)); n != 9 {
				t.Errorf("%d records, want 9", n)
			}
		})
	}

	// The documented exception (docs/budgets.md): rejectTooDeep still writes
	// its cascade_too_deep entry in one own transaction on the runner, which
	// is the allocator pool in nava-core (RS-E9-3; T-E9-2 left it there).
	// When it moves off the allocator pool, this becomes want 0.
	t.Run("cascade_too_deep", func(t *testing.T) {
		e := migrated(t)
		f := seedTenant(t, e)
		alloc, tr := tracedAllocPool(t, appDSN(t, e.platform))
		runner := &recordingRunner{PoolRunner: pipeline.PoolRunner{Pool: alloc, AllocTimeout: noLatencyBudget}}
		p := pipeline.New(authz.ACL{Policies: authz.PolicyStore{}}, entitytypes.Store{}, pipeline.NoopSink{}, runner, pipeline.SystemClock{})
		cmd := createCmd(f, map[string]any{"title": "deep"})
		cmd.Source, cmd.Causation = pipeline.SourceRule, outbox.Causation{OriginID: uuid.New(), Depth: config.CascadeDepthLimit + 1}
		tr.reset()
		if _, err := write(context.Background(), e, p, f.id, cmd); !errors.Is(err, outbox.ErrCascadeTooDeep) {
			t.Fatalf("err = %v, want ErrCascadeTooDeep", err)
		}
		wantAllocationsOnly(t, "cascade_too_deep create", runner, tr, 1, 0)
		if n := entries(t, e, f.id, tooDeepAction); n != 1 {
			t.Errorf("%d cascade_too_deep entries, want 1", n)
		}
	})
}

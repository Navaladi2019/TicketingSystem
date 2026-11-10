//go:build integration

// TICK-657 (P3-36e): PLAN-003 D-3-9 as amended 2026-11-10, gaps allowed and
// no reclaim. A create takes its number on the pipeline's TxRunner, in a
// short transaction of its own that commits before the record is inserted,
// and only after the replay check, validation and the write check (ADR-001
// I4). The runner is the allocator pool, never the pool the create's own
// transaction holds.

package pipeline_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/entitytypes"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
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

// AC: an idempotent replay returns the original number and allocates
// nothing; it runs no transaction on the runner at all.
func TestCreateReplayTakesNoNumber(t *testing.T) {
	e := migrated(t)
	f := seedTenant(t, e)
	runner := &recordingRunner{TxRunner: e.runner()}
	p := pipeline.New(authz.ACL{Policies: authz.PolicyStore{}}, entitytypes.Store{}, pipeline.NoopSink{}, runner, pipeline.SystemClock{})
	ctx := context.Background()
	cmd := createCmd(f, map[string]any{"title": "keyed"})
	cmd.Actor, cmd.Source, cmd.IdempotencyKey, cmd.Causation = f.system(), pipeline.SourceWorkflow, "wf:gap", workflowCausation()

	first, err := write(ctx, e, p, f.id, cmd)
	if err != nil || first.Number != "INC0000001" || runner.runs != 1 {
		t.Fatalf("first = %+v, %v, %d runner transactions; want INC0000001 and 1", first, err, runner.runs)
	}
	replay, err := write(ctx, e, p, f.id, cmd)
	if err != nil || replay != first {
		t.Fatalf("replay = %+v, %v; want %+v", replay, err, first)
	}
	if runner.runs != 1 {
		t.Fatalf("%d runner transactions after the replay, want 1: the replay took a number", runner.runs)
	}
	if n := counterNext(t, e, f.id, entityKey); n != 2 {
		t.Fatalf("counter next = %d after the replay, want 2", n)
	}
}

// scopeRunner records, for each allocation, the tenants whose counter rows
// its transaction can see after it took the number: under RLS, only the
// tenant its BEGIN set.
type scopeRunner struct {
	pipeline.TxRunner
	mu   sync.Mutex
	seen [][]uuid.UUID
}

func (r *scopeRunner) InTenantTx(ctx context.Context, tenant uuid.UUID, fn func(pgx.Tx) error) error {
	return r.TxRunner.InTenantTx(ctx, tenant, func(tx pgx.Tx) error {
		if err := fn(tx); err != nil {
			return err
		}
		rows, err := tx.Query(ctx, `SELECT DISTINCT tenant_id FROM record_counters ORDER BY tenant_id`)
		if err != nil {
			return err
		}
		ids, err := pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
		if err != nil {
			return err
		}
		r.mu.Lock()
		r.seen = append(r.seen, ids)
		r.mu.Unlock()
		return nil
	})
}

// AC: the allocation transaction is tenant-scoped. Its BEGIN sets the write's
// tenant, so under RLS it sees only that tenant's counters, and a transaction
// of tenant A cannot take, or move, a number of tenant B's type.
func TestAllocationTransactionIsTenantScoped(t *testing.T) {
	e := migrated(t)
	a, b := seedTenant(t, e), seedTenant(t, e)
	runner := &scopeRunner{TxRunner: e.runner()}
	p := pipeline.New(authz.ACL{Policies: authz.PolicyStore{}}, entitytypes.Store{}, pipeline.NoopSink{}, runner, pipeline.SystemClock{})
	ctx := context.Background()
	for _, f := range []tenantFixture{a, b, a} {
		mustCreate(t, e, p, f, map[string]any{"title": "x"})
	}
	// The third allocation (tenant A again) runs with tenant B's row in the
	// table, and still sees only tenant A's.
	for i, want := range []uuid.UUID{a.id, b.id, a.id} {
		if i >= len(runner.seen) || !slices.Equal(runner.seen[i], []uuid.UUID{want}) {
			t.Fatalf("allocation transactions saw the counters of %v, want each only its own tenant's (%s, %s, %s)",
				runner.seen, a.id, b.id, a.id)
		}
	}
	if n := count(t, e.super, `SELECT count(DISTINCT tenant_id) FROM record_counters WHERE tenant_id IN ($1, $2)`, a.id, b.id); n != 2 {
		t.Fatalf("%d tenants have counter rows, want 2", n)
	}

	// Tenant B's counter row exists (next = 2). A tenant A transaction can
	// neither see it to update it nor insert a row for tenant B.
	defB := *definitionID(t, e, b.id)
	err := e.runner().InTenantTx(ctx, a.id, func(tx pgx.Tx) error {
		_, err := store.NextNumber(ctx, tx, b.id, defB)
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

// barrierRunner holds each allocation until n of them are waiting, so n
// creates hold their own transactions' connections at once when they ask for
// a number: the worst case for a pool both share.
type barrierRunner struct {
	pipeline.TxRunner
	n       int
	mu      sync.Mutex
	waiting int
	release chan struct{}
}

func (r *barrierRunner) InTenantTx(ctx context.Context, tenant uuid.UUID, fn func(pgx.Tx) error) error {
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
		return ctx.Err()
	}
	return r.TxRunner.InTenantTx(ctx, tenant, fn)
}

// AC (scope 3): no pool starvation. With an API pool of 4 connections, 8
// concurrent creates each hold one of the 4 while they wait for a number. On
// the allocator pool they all finish; on the API pool itself (the control)
// they wait for a connection none of them gives back, so the check catches a
// runner wired back onto the API pool.
func TestAllocatorPoolDoesNotStarveThePool(t *testing.T) {
	const poolSize = 4
	base := migrated(t)
	f := seedTenant(t, base)
	dsn := appDSN(t, base.platform)
	run := func(t *testing.T, e env, timeout time.Duration) []error {
		t.Helper()
		p := pipeline.New(authz.ACL{Policies: authz.PolicyStore{}}, entitytypes.Store{}, pipeline.NoopSink{},
			&barrierRunner{TxRunner: e.runner(), n: poolSize}, pipeline.SystemClock{})
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		errs := make([]error, 2*poolSize)
		var wg sync.WaitGroup
		for i := range errs {
			wg.Go(func() {
				_, errs[i] = write(ctx, e, p, f.id, createCmd(f, map[string]any{"title": fmt.Sprint(i)}))
			})
		}
		wg.Wait()
		return errs
	}
	small := func() env {
		cfg := serverPoolConfig(t, dsn)
		cfg.MaxConns = poolSize
		return env{super: base.super, app: newPool(t, cfg), platform: base.platform}
	}

	t.Run("allocator pool", func(t *testing.T) {
		e := small()
		e.alloc = allocPool(t, dsn)
		noErrors(t, run(t, e, 30*time.Second))
	})
	t.Run("control: the API pool", func(t *testing.T) {
		e := small() // alloc nil: the runner uses app
		for i, err := range run(t, e, 3*time.Second) {
			if !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("create %d on a shared pool: %v, want it starved (context.DeadlineExceeded)", i, err)
			}
		}
	})
	if got := numbers(t, base, f.id, entityKey); !slices.Equal(got, wantNumbers(t, "INC", 1, 2*poolSize)) {
		t.Fatalf("numbers = %v, want INC0000001..%d from the allocator pool run only", got, 2*poolSize)
	}
}

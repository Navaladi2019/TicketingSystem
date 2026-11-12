//go:build integration

// TICK-665 (S17 P6-01), ADR-005 §4.2: between M2 and M3 record_counters has
// two unique indexes, record_counters_pkey (tenant_id, entity_definition_id)
// and record_counters_scoped_key (tenant_id, entity_definition_id,
// scope_key). INSERT ... ON CONFLICT absorbs a conflict on its arbiter index
// only, so two concurrent FIRST upserts of one type can fail with 23505 on
// the other index (scratch/tick-665/README.md: 15,533 R0 and 19,829 R1
// failures in 160,000 first upserts). The fix is R1's allocation retrying
// once on that 23505 (TICK-666, P6-02). It is also the guard on the
// isolation level: at REPEATABLE READ the same race raises 40001, which is
// not retried (ADR-005-E1).

package pipeline_test

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/entitytypes"
	"github.com/Navaladi2019/TicketingSystem/internal/records/pipeline"
)

// Pinned (TICK-665 option A): concurrent first creates of a type with no
// counter row yet all succeed and take 1..n, for a `gaps` type (allocation on
// the allocator pool, released together by barrierRunner) and a `gapless`
// type (allocation last inside each create's transaction). Each round is a
// fresh tenant, so every round races the first insert of its counter row.
func TestConcurrentFirstCreatesOfATypeAllSucceed(t *testing.T) {
	const (
		creates = 8
		rounds  = 15
	)
	e := migrated(t)
	dsn := appDSN(t, e.platform)
	// Every create holds an API pool connection while barrierRunner holds
	// its allocation until all of them wait, so the API pool must fit all
	// the creates: pgxpool's default width, max(4, NumCPU), is 4 on a
	// 4-vCPU runner, and the creates past 4 would wait at BEGIN for a
	// barrier that never opens.
	cfg := serverPoolConfig(t, dsn)
	cfg.MaxConns = creates
	e.app = newPool(t, cfg)
	e.alloc = allocPool(t, dsn)
	for _, mode := range []string{"gaps", "gapless"} {
		t.Run(mode, func(t *testing.T) {
			for round := range rounds {
				f := seedTenant(t, e)
				if mode == "gapless" {
					f = seedTenantWith(t, e, gaplessDefinition(definition(schemaV1)))
				}
				var runner pipeline.TxRunner = e.runner()
				if mode == "gaps" {
					runner = &barrierRunner{PoolRunner: e.runner(), n: creates}
				}
				p := pipeline.New(authz.ACL{Policies: authz.PolicyStore{}}, entitytypes.Store{}, pipeline.NoopSink{},
					runner, pipeline.SystemClock{})
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				start := make(chan struct{})
				errs := make([]error, creates)
				var wg sync.WaitGroup
				for i := range errs {
					wg.Go(func() {
						<-start
						_, errs[i] = write(ctx, e, p, f.id, createCmd(f, map[string]any{"title": fmt.Sprint(i)}))
					})
				}
				close(start)
				wg.Wait()
				cancel()
				for i, err := range errs {
					if err != nil {
						t.Errorf("round %d, create %d: %v", round, i, err)
					}
				}
				if got := numbers(t, e, f.id, entityKey); !slices.Equal(got, wantNumbers(t, "INC", 1, creates)) {
					t.Fatalf("round %d: numbers = %v, want INC0000001..%d", round, got, creates)
				}
			}
		})
	}
}

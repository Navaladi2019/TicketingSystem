//go:build integration

// TICK-666 (PLAN-006 P6-02, ADR-005 §4.2 "Deploy" row): both allocation
// paths take their number with the 3-column upsert, scope '' for the default
// format, and continue the counter R0's 2-column upsert left. Since M3
// (TICK-670) the 3-column key is record_counters' only unique index, so two
// first creates of a counter serialise on it and the second takes the UPDATE
// arm: the expand window's first-create race (ADR-005-E1) and its one retry
// are gone.

package pipeline_test

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/entitytypes"
	"github.com/Navaladi2019/TicketingSystem/internal/metadata"
	"github.com/Navaladi2019/TicketingSystem/internal/records/pipeline"
)

// numberingModes are the two allocation paths, each with its incident type.
var numberingModes = []struct {
	mode metadata.NumberingMode
	def  func() metadata.Definition
}{
	{metadata.NumberingGaps, func() metadata.Definition { return definition(schemaV1) }},
	{metadata.NumberingGapless, func() metadata.Definition { return gaplessDefinition(definition(schemaV1)) }},
}

// racePipeline is a pipeline whose allocations may wait out a held lock
// (noLatencyBudget).
func racePipeline(e env) *pipeline.Pipeline {
	return pipeline.New(authz.ACL{Policies: authz.PolicyStore{}}, entitytypes.Store{}, pipeline.NoopSink{}, e.runner(), pipeline.SystemClock{})
}

// counterRows is the tenant's record_counters rows of the incident type, as
// "scope_key=next", as the superuser reads them.
func counterRows(t *testing.T, e env, f tenantFixture) []string {
	t.Helper()
	rows, err := e.super.Query(context.Background(), `SELECT c.scope_key, c.next FROM record_counters c
  JOIN entity_definitions d ON d.tenant_id = c.tenant_id AND d.id = c.entity_definition_id
  WHERE c.tenant_id = $1 AND d.key = $2 ORDER BY c.scope_key`, f.id, entityKey)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for rows.Next() {
		var scope string
		var next int
		if err := rows.Scan(&scope, &next); err != nil {
			t.Fatal(err)
		}
		got = append(got, fmt.Sprintf("%q=%d", scope, next))
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return got
}

// Pinned test 2 (ADR-005 §4.2): a counter that R0's 2-column upsert left
// at 42 continues at 43 on R1's create path, for a `gaps` and a `gapless`
// type, in the same row (scope ”).
func TestP602CreateContinuesTheTwoColumnCounter(t *testing.T) {
	e := migrated(t)
	for _, m := range numberingModes {
		t.Run(string(m.mode), func(t *testing.T) {
			f := seedTenantWith(t, e, m.def())
			def := *definitionID(t, e, f.id)
			ctx := context.Background()
			// The row R0's 2-column upsert left after handing out 1..42: no
			// scope_key in its INSERT, so the default scope ''. R0's
			// statement itself fails since M3 (42P10, TestP605CounterKeyContract).
			if _, err := e.super.Exec(ctx, `INSERT INTO record_counters (tenant_id, entity_definition_id, next) VALUES ($1, $2, 43)`, f.id, def); err != nil {
				t.Fatal(err)
			}
			if got := mustCreate(t, e, newPipeline(e, nil), f, map[string]any{"title": "after R0"}); got.Number != "INC0000043" {
				t.Fatalf("first R1 create = %s, want INC0000043", got.Number)
			}
			if got := counterRows(t, e, f); !slices.Equal(got, []string{`""=44`}) {
				t.Fatalf("counter rows = %v, want one row, scope '', next 44", got)
			}
		})
	}
}

// holderTx is a superuser transaction on a connection of its own, rolled
// back when the test ends. Not on e.super: a transaction keeps its first
// pg_stat_activity snapshot, so waitForLockWait polling through it would
// never see a wait that starts later.
func holderTx(t *testing.T, e env) pgx.Tx {
	t.Helper()
	ctx := context.Background()
	conn, err := pgx.ConnectConfig(ctx, e.super.Config())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(context.Background()) })
	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tx.Rollback(context.Background()) })
	return tx
}

// holdCounter opens a superuser transaction that inserts the incident type's
// counter row of f with scope scope and next, and returns it uncommitted. A
// create's upsert of the same scope waits on it in record_counters_pkey.
func holdCounter(t *testing.T, e env, f tenantFixture, scope string, next int) pgx.Tx {
	t.Helper()
	ctx := context.Background()
	holder := holderTx(t, e)
	if _, err := holder.Exec(ctx, `INSERT INTO record_counters (tenant_id, entity_definition_id, scope_key, next) VALUES ($1, $2, $3, $4)`,
		f.id, *definitionID(t, e, f.id), scope, next); err != nil {
		t.Fatal(err)
	}
	return holder
}

// createAsync starts one create of f and returns its result channel.
func createAsync(e env, p *pipeline.Pipeline, f tenantFixture, title string) <-chan error {
	done := make(chan error, 1)
	go func() {
		_, err := write(context.Background(), e, p, f.id, createCmd(f, map[string]any{"title": title}))
		done <- err
	}()
	return done
}

// AC (TICK-670, M3): a first create that meets another first insert of
// its counter, still uncommitted, waits on the key and, once the winner
// commits, takes the UPDATE arm of the winner's row: no 23505, no retry.
// Before M3 the same race on the other unique index failed with 23505
// (ADR-005-E1).
func TestP605FirstCreateWaitsForTheWinner(t *testing.T) {
	e := migrated(t)
	for _, m := range numberingModes {
		t.Run(string(m.mode), func(t *testing.T) {
			f := seedTenantWith(t, e, m.def())
			p := racePipeline(e)
			ctx := context.Background()
			holder := holdCounter(t, e, f, "", 43)
			done := createAsync(e, p, f, "loser")
			waitForLockWait(t, e)
			if err := holder.Commit(ctx); err != nil {
				t.Fatal(err)
			}
			if err := <-done; err != nil {
				t.Fatalf("create after the race: %v, want the UPDATE arm of the winner's row", err)
			}
			if got := numbers(t, e, f.id, entityKey); !slices.Equal(got, []string{"INC0000043"}) {
				t.Fatalf("numbers = %v, want INC0000043 from the winner's counter", got)
			}
			if got := counterRows(t, e, f); !slices.Equal(got, []string{`""=44`}) {
				t.Fatalf("counter rows = %v, want one row, scope '', next 44", got)
			}
		})
	}
}

// AC (TICK-670, M3): another scope's row of the same type no longer blocks
// the default scope. Before M3 it held the 2-column key, and the create
// failed with 23505 on record_counters_pkey.
func TestP605OtherScopeDoesNotBlockTheDefault(t *testing.T) {
	e := migrated(t)
	for _, m := range numberingModes {
		t.Run(string(m.mode), func(t *testing.T) {
			f := seedTenantWith(t, e, m.def())
			if _, err := e.super.Exec(context.Background(), `INSERT INTO record_counters (tenant_id, entity_definition_id, scope_key, next) VALUES ($1, $2, 'held', 7)`,
				f.id, *definitionID(t, e, f.id)); err != nil {
				t.Fatal(err)
			}
			if got := mustCreate(t, e, racePipeline(e), f, map[string]any{"title": "default"}); got.Number != "INC0000001" {
				t.Fatalf("create = %s, want INC0000001 from a new default-scope counter", got.Number)
			}
			if got := counterRows(t, e, f); !slices.Equal(got, []string{`""=2`, `"held"=7`}) {
				t.Fatalf("counter rows = %v, want the new default row and the untouched held row", got)
			}
		})
	}
}

// AC: concurrent first creates of a type, released together, all succeed
// with unique numbers 1..n, on both paths, with no retry (one unique index
// since M3). LOCK TABLE ... SHARE holds every create's upsert until all of them
// wait on it, then lets them race the first insert at once. Whether a round
// actually races is up to the scheduler; the retry count is logged.
func TestP602ConcurrentFirstCreatesAllSucceed(t *testing.T) {
	e := migrated(t)
	const creators, rounds = 8, 4
	for _, m := range numberingModes {
		t.Run(string(m.mode), func(t *testing.T) {
			p := racePipeline(e)
			ctx := context.Background()
			for round := range rounds {
				f := seedTenantWith(t, e, m.def())
				holder := holderTx(t, e)
				if _, err := holder.Exec(ctx, "LOCK TABLE record_counters IN SHARE MODE"); err != nil {
					t.Fatal(err)
				}
				errs := make([]error, creators)
				var wg sync.WaitGroup
				for i := range creators {
					wg.Go(func() {
						_, errs[i] = write(ctx, e, p, f.id, createCmd(f, map[string]any{"title": fmt.Sprintf("racer %d", i)}))
					})
				}
				// The allocator pool is allocatorMaxConns wide: at most that
				// many `gaps` upserts wait at once.
				deadline := time.Now().Add(15 * time.Second)
				for count(t, e.super, `SELECT count(*) FROM pg_stat_activity WHERE usename = 'nava_app' AND wait_event_type = 'Lock'`) < allocatorMaxConns {
					if time.Now().After(deadline) {
						t.Fatal("the creates never queued on the counter table")
					}
					time.Sleep(5 * time.Millisecond)
				}
				if err := holder.Commit(ctx); err != nil {
					t.Fatal(err)
				}
				wg.Wait()
				noErrors(t, errs)
				if got := numbers(t, e, f.id, entityKey); !slices.Equal(got, wantNumbers(t, "INC", 1, creators)) {
					t.Fatalf("round %d numbers = %v, want INC0000001..%d", round, got, creators)
				}
			}
		})
	}
}

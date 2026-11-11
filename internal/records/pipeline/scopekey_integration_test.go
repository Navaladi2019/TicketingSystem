//go:build integration

// TICK-666 (PLAN-006 P6-02, ADR-005 §4.2 "Deploy" row): both allocation
// paths take their number with R1's 3-column upsert, scope '' for the
// default format, and continue the counter R0's 2-column upsert left. While
// both unique indexes of record_counters exist (M2 to M3), a first create
// that loses the race for its counter row with 23505 on the index its
// upsert does not name is retried once (ADR-005-E1, TICK-665 decision A):
// in a fresh AllocTx for `gaps`, behind a savepoint for `gapless`.

package pipeline_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

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

// retryPipeline is a pipeline whose runner records its metrics on a manual
// reader and whose allocations may wait out a held lock (noLatencyBudget).
func retryPipeline(t *testing.T, e env) (*pipeline.Pipeline, *sdkmetric.ManualReader) {
	t.Helper()
	reader := sdkmetric.NewManualReader()
	m, err := pipeline.NewAllocMetrics(sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader)))
	if err != nil {
		t.Fatal(err)
	}
	runner := e.runner()
	runner.AllocTimeout, runner.Metrics = noLatencyBudget, m
	return pipeline.New(authz.ACL{Policies: authz.PolicyStore{}}, entitytypes.Store{}, pipeline.NoopSink{}, runner, pipeline.SystemClock{}), reader
}

// retries is nava_numbering_first_create_retries by mode.
func retries(t *testing.T, reader *sdkmetric.ManualReader) map[string]int64 {
	t.Helper()
	var rm metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &rm); err != nil {
		t.Fatal(err)
	}
	got := map[string]int64{}
	for _, sm := range rm.ScopeMetrics {
		for _, mt := range sm.Metrics {
			if mt.Name != "nava_numbering_first_create_retries" {
				continue
			}
			for _, dp := range mt.Data.(metricdata.Sum[int64]).DataPoints {
				mode, _ := dp.Attributes.Value(attribute.Key("mode"))
				got[mode.AsString()] += dp.Value
			}
		}
	}
	return got
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
			// R0, the pre-S17 statement, 42 times: the counter has handed out
			// 1..42.
			inTx(t, e, f.id, func(tx pgx.Tx) error {
				for range 42 {
					if _, err := tx.Exec(ctx, `INSERT INTO record_counters (tenant_id, entity_definition_id, next) VALUES ($1, $2, 2)
ON CONFLICT (tenant_id, entity_definition_id) DO UPDATE SET next = record_counters.next + 1 RETURNING next - 1`, f.id, def); err != nil {
						return err
					}
				}
				return nil
			})
			if got := mustCreate(t, e, newPipeline(e, nil), f, map[string]any{"title": "after R1"}); got.Number != "INC0000043" {
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
// create's R1 upsert of scope ” passes its arbiter (no (t, d, ”) row),
// then waits on this row in record_counters_pkey: the expand window's race,
// with the winner held.
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

// AC (TICK-665 decision A): a first create that loses the race on
// record_counters_pkey is retried once and takes the UPDATE arm of the
// winner's row. The winner's row is made deterministic: inserted under
// another scope, so the create's upsert passes its arbiter and waits on the
// pkey, then moved to scope ” and committed, so the retry finds it.
func TestP602FirstCreateRaceIsRetriedOnce(t *testing.T) {
	e := migrated(t)
	for _, m := range numberingModes {
		t.Run(string(m.mode), func(t *testing.T) {
			f := seedTenantWith(t, e, m.def())
			p, reader := retryPipeline(t, e)
			ctx := context.Background()
			holder := holdCounter(t, e, f, "held", 43)
			done := createAsync(e, p, f, "loser")
			waitForLockWait(t, e)
			if _, err := holder.Exec(ctx, `UPDATE record_counters SET scope_key = '' WHERE tenant_id = $1`, f.id); err != nil {
				t.Fatal(err)
			}
			if err := holder.Commit(ctx); err != nil {
				t.Fatal(err)
			}
			if err := <-done; err != nil {
				t.Fatalf("create after the race: %v, want it retried onto the winner's row", err)
			}
			if got := numbers(t, e, f.id, entityKey); !slices.Equal(got, []string{"INC0000043"}) {
				t.Fatalf("numbers = %v, want INC0000043 from the winner's counter", got)
			}
			if got := counterRows(t, e, f); !slices.Equal(got, []string{`""=44`}) {
				t.Fatalf("counter rows = %v, want one row, scope '', next 44", got)
			}
			if got := retries(t, reader); got[string(m.mode)] != 1 || len(got) != 1 {
				t.Fatalf("retries = %v, want 1 for %s only", got, m.mode)
			}
			// Further creates take the UPDATE arm with no retry.
			if got := mustCreate(t, e, p, f, map[string]any{"title": "next"}); got.Number != "INC0000044" {
				t.Fatalf("next create = %s, want INC0000044", got.Number)
			}
			if got := retries(t, reader); got[string(m.mode)] != 1 {
				t.Fatalf("retries after an uncontended create = %v, want still 1", got)
			}
		})
	}
}

// AC: the retry runs no more than once. When the second attempt meets the
// same violation (here a committed row of another scope on the old pkey,
// which no retry can get past before M3), the create fails with that 23505
// and takes no number.
func TestP602FirstCreateRaceIsRetriedNoMoreThanOnce(t *testing.T) {
	e := migrated(t)
	for _, m := range numberingModes {
		t.Run(string(m.mode), func(t *testing.T) {
			f := seedTenantWith(t, e, m.def())
			p, reader := retryPipeline(t, e)
			ctx := context.Background()
			holder := holdCounter(t, e, f, "held", 7)
			done := createAsync(e, p, f, "loser")
			waitForLockWait(t, e)
			if err := holder.Commit(ctx); err != nil {
				t.Fatal(err)
			}
			err := <-done
			var pe *pgconn.PgError
			if !errors.As(err, &pe) || pe.Code != "23505" || pe.ConstraintName != "record_counters_pkey" {
				t.Fatalf("create = %v, want 23505 on record_counters_pkey after one retry", err)
			}
			if got := retries(t, reader); got[string(m.mode)] != 1 || len(got) != 1 {
				t.Fatalf("retries = %v, want exactly 1 for %s", got, m.mode)
			}
			if got := numbers(t, e, f.id, entityKey); len(got) != 0 {
				t.Fatalf("numbers = %v, want none", got)
			}
			if got := counterRows(t, e, f); !slices.Equal(got, []string{`"held"=7`}) {
				t.Fatalf("counter rows = %v, want only the held row", got)
			}
		})
	}
}

// AC: concurrent first creates of a type, released together while both
// unique indexes exist, all succeed with unique numbers 1..n, on both
// paths. LOCK TABLE ... SHARE holds every create's upsert until all of them
// wait on it, then lets them race the first insert at once. Whether a round
// actually races is up to the scheduler; the retry count is logged.
func TestP602ConcurrentFirstCreatesAllSucceed(t *testing.T) {
	e := migrated(t)
	const creators, rounds = 8, 4
	for _, m := range numberingModes {
		t.Run(string(m.mode), func(t *testing.T) {
			p, reader := retryPipeline(t, e)
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
			t.Logf("%s: %d first-create retries over %d rounds of %d", m.mode, retries(t, reader)[string(m.mode)], rounds, creators)
		})
	}
}

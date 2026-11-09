//go:build integration

// TICK-648 (PLAN-003 P3-36b): a create is one transaction of batched round
// trips, and the record_counters row is taken in its last batch. These tests
// fail a create at each stage and check that nothing of it is kept, number
// many concurrent creates on a direct pool, and trace the round trips of a
// create to show the counter is taken last, with the tenant set before every
// one of them.

package pipeline_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/internal/records/pipeline"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy/tenancytest"
)

// failSink enqueues the write's outbox row, as the S11b sink will, then
// fails: the outbox row must roll back with the write.
type failSink struct{ enqueue *enqueueSink }

var errSink = errors.New("sink failed after enqueuing")

func (s failSink) RecordChanged(ctx context.Context, tx pgx.Tx, ev pipeline.Event) error {
	if err := s.enqueue.RecordChanged(ctx, tx, ev); err != nil {
		return err
	}
	return errSink
}

// tenantState is what a create can leave behind in a tenant.
type tenantState struct {
	records, audits, outbox, keys, next int
}

func stateOf(t *testing.T, e env, tenant uuid.UUID) tenantState {
	t.Helper()
	return tenantState{
		records: count(t, e.super, `SELECT count(*) FROM records WHERE tenant_id = $1`, tenant),
		audits:  count(t, e.super, `SELECT count(*) FROM audit_log WHERE tenant_id = $1 AND subject_type = 'record'`, tenant),
		outbox:  count(t, e.super, `SELECT count(*) FROM outbox WHERE tenant_id = $1`, tenant),
		keys:    count(t, e.super, `SELECT count(*) FROM record_write_keys WHERE tenant_id = $1`, tenant),
		next:    count(t, e.super, `SELECT coalesce(max(next), 0) FROM record_counters WHERE tenant_id = $1`, tenant),
	}
}

// failOn makes every statement of kind on table fail, as the superuser,
// until the test ends.
func failOn(t *testing.T, e env, table, kind string) {
	t.Helper()
	fn := "tick648_fail_" + strings.ReplaceAll(table, ".", "_")
	mustExec(t, e.super, `CREATE OR REPLACE FUNCTION `+fn+`() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN RAISE EXCEPTION 'injected failure on `+table+`'; END $$`)
	mustExec(t, e.super, `CREATE TRIGGER `+fn+` BEFORE `+kind+` ON `+table+` FOR EACH ROW EXECUTE FUNCTION `+fn+`()`)
	t.Cleanup(func() {
		_, _ = e.super.Exec(context.Background(), `DROP TRIGGER IF EXISTS `+fn+` ON `+table)
	})
}

// TestCreateRollsBackAtEveryStage: a create that fails at the counter, the
// record insert, the write key, the audit entry, the sink's outbox row, or
// the COMMIT's own round trip leaves no record, audit row, outbox row or
// write key, and does not move the counter; the next create takes the next
// number, with no gap.
func TestCreateRollsBackAtEveryStage(t *testing.T) {
	e := migrated(t)
	f := seedTenant(t, e)
	ok := newPipeline(e, &enqueueSink{})
	ctx := context.Background()
	// A system writer with an idempotency key, so the write key is a stage.
	cmd := func(i int) pipeline.Command {
		c := createCmd(f, map[string]any{"title": fmt.Sprintf("stage %d", i)})
		c.Actor, c.Source, c.Causation, c.IdempotencyKey = f.system(), pipeline.SourceWorkflow, workflowCausation(), fmt.Sprintf("k-%d", i)
		return c
	}
	if _, err := write(ctx, e, ok, f.id, cmd(0)); err != nil {
		t.Fatal(err)
	}
	before := stateOf(t, e, f.id)
	if before != (tenantState{records: 1, audits: 1, outbox: 1, keys: 1, next: 2}) {
		t.Fatalf("after one create: %+v", before)
	}

	stages := []struct {
		name  string
		setup func(t *testing.T) *pipeline.Pipeline
		// atCommit queues a failing statement for the COMMIT's round trip.
		atCommit bool
	}{
		{"counter", func(t *testing.T) *pipeline.Pipeline { failOn(t, e, "record_counters", "UPDATE"); return ok }, false},
		{"record", func(t *testing.T) *pipeline.Pipeline { failOn(t, e, "records", "INSERT"); return ok }, false},
		{"write key", func(t *testing.T) *pipeline.Pipeline { failOn(t, e, "record_write_keys", "INSERT"); return ok }, false},
		{"audit", func(t *testing.T) *pipeline.Pipeline { failOn(t, e, "audit_log", "INSERT"); return ok }, false},
		{"outbox", func(*testing.T) *pipeline.Pipeline { return newPipeline(e, failSink{&enqueueSink{}}) }, false},
		{"commit", func(*testing.T) *pipeline.Pipeline { return ok }, true},
	}
	for i, s := range stages {
		t.Run(s.name, func(t *testing.T) {
			p := s.setup(t)
			err := tenancy.InTenantTx(ctx, e.app, f.id, func(tx pgx.Tx) error {
				if _, err := p.Write(ctx, tx, cmd(i+1)); err != nil {
					return err
				}
				if s.atCommit && !tenancy.ExecAtCommit(tx, "SELECT 1 / 0") {
					t.Error("ExecAtCommit refused the transaction")
				}
				return nil
			})
			if err == nil {
				t.Fatal("the create succeeded, want the injected failure")
			}
			if got := stateOf(t, e, f.id); got != before {
				t.Errorf("after a failure at the %s: %+v, want %+v (nothing kept, counter unmoved)", s.name, got, before)
			}
		})
	}
	res, err := write(ctx, e, ok, f.id, cmd(99))
	if err != nil {
		t.Fatal(err)
	}
	if res.Number != "INC0000002" {
		t.Errorf("the create after the failures took %s, want INC0000002 (no gap)", res.Number)
	}
}

// TestConcurrentCreatesNumberWithoutGaps: 8 workers creating 400 records on
// a direct pool (the P3-36 8-client shape) get INC0000001..INC0000400.
func TestConcurrentCreatesNumberWithoutGaps(t *testing.T) {
	e := migrated(t)
	f := seedTenant(t, e)
	p := newPipeline(e, nil)
	took, errs := createMany(e, p, f, 400, 8)
	noErrors(t, errs)
	if got, want := numbers(t, e, f.id, entityKey), wantNumbers(t, "INC", 1, 400); !slices.Equal(got, want) {
		t.Fatalf("%d numbers, want INC0000001..INC0000400 with no gap or duplicate", len(got))
	}
	t.Logf("400 creates on 8 workers, direct: %v, %.0f creates/s", took, 400/took.Seconds())
}

// TestCreateTakesTheCounterInItsLastBatch traces a create's round trips: the
// tenancy BEGIN sets the tenant before anything else, every later trip runs
// under it, and the record_counters upsert is in the last batch before
// COMMIT, with the record, write key and audit inserts and nothing after
// them. Its trips and statements are logged for the PR (TICK-648).
func TestCreateTakesTheCounterInItsLastBatch(t *testing.T) {
	e := migrated(t)
	f := seedTenant(t, e)
	cfg := serverPoolConfig(t, appDSN(t, e.platform))
	g := &tenancytest.Guard{}
	cfg.ConnConfig.Tracer = g
	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	traced := env{super: e.super, app: pool, platform: e.platform}
	p := newPipeline(traced, nil)

	for _, c := range []struct {
		name string
		cmd  pipeline.Command
	}{
		{"user", createCmd(f, map[string]any{"title": "traced"})},
		{"system with key", func() pipeline.Command {
			c := createCmd(f, map[string]any{"title": "traced system"})
			c.Actor, c.Source, c.Causation, c.IdempotencyKey = f.system(), pipeline.SourceWorkflow, workflowCausation(), "traced-key"
			return c
		}()},
	} {
		t.Run(c.name, func(t *testing.T) {
			ctx, rec := tenancytest.Record(context.Background())
			if _, err := write(ctx, traced, p, f.id, c.cmd); err != nil {
				t.Fatal(err)
			}
			trips := rec.Trips()
			for i, tr := range trips {
				if tr.Tenant != f.id {
					t.Errorf("trip %d ran under tenant %s, want %s: %q", i, tr.Tenant, f.id, tr.SQL)
				}
			}
			last := trips[len(trips)-2]
			if !last.Batch || !strings.Contains(last.SQL[0], "record_counters") {
				t.Fatalf("the trip before COMMIT is %q, want the batch that opens with the counter upsert", last.SQL)
			}
			for _, want := range []string{"INSERT INTO records", "INSERT INTO audit_log"} {
				if !slices.ContainsFunc(last.SQL, func(s string) bool { return strings.Contains(s, want) }) {
					t.Errorf("the counter batch has no %q: %q", want, last.SQL)
				}
			}
			if end := trips[len(trips)-1]; end.SQL[len(end.SQL)-1] != "commit" {
				t.Errorf("the last trip is %q, want COMMIT", end.SQL)
			}
			for _, tr := range trips[:len(trips)-2] {
				for _, s := range tr.SQL {
					if strings.Contains(s, "record_counters") {
						t.Errorf("a trip before the last batch takes the counter: %q", s)
					}
				}
			}
			t.Logf("create (%s), pipeline only: %d round trips, %d statements", c.name, len(trips), rec.Statements())
		})
	}
	if v := g.Violations(); len(v) > 0 {
		t.Errorf("statements ran with no tenant set: %q", v)
	}
}

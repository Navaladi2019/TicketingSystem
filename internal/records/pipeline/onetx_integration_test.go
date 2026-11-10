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
// write key. Its number was taken in a transaction of its own that already
// committed (D-3-9 as amended 2026-11-10, gaps allowed): a failure at the
// counter moves nothing, and each later stage's failure leaves one gap, so
// the next create takes the number after the last one spent.
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
		// spent: the stage fails after the number committed (a gap).
		spent bool
	}{
		{"counter", func(t *testing.T) *pipeline.Pipeline { failOn(t, e, "record_counters", "UPDATE"); return ok }, false, false},
		{"record", func(t *testing.T) *pipeline.Pipeline { failOn(t, e, "records", "INSERT"); return ok }, false, true},
		{"write key", func(t *testing.T) *pipeline.Pipeline { failOn(t, e, "record_write_keys", "INSERT"); return ok }, false, true},
		{"audit", func(t *testing.T) *pipeline.Pipeline { failOn(t, e, "audit_log", "INSERT"); return ok }, false, true},
		{"outbox", func(*testing.T) *pipeline.Pipeline { return newPipeline(e, failSink{&enqueueSink{}}) }, false, true},
		{"commit", func(*testing.T) *pipeline.Pipeline { return ok }, true, true},
	}
	for i, s := range stages {
		if s.spent {
			before.next++
		}
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
				t.Errorf("after a failure at the %s: %+v, want %+v (nothing kept, the counter moved only by a spent number)", s.name, got, before)
			}
		})
	}
	res, err := write(ctx, e, ok, f.id, cmd(99))
	if err != nil {
		t.Fatal(err)
	}
	// Five numbers spent (2..6), so the next is 7.
	if res.Number != "INC0000007" {
		t.Errorf("the create after the failures took %s, want INC0000007 (2..6 are gaps)", res.Number)
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

// TestCreateTakesItsNumberInATransactionOfItsOwn traces a create's round
// trips on both pools: the tenancy BEGIN sets the tenant before anything
// else, every trip runs under it, and the record_counters upsert runs alone
// in a transaction of its own (BEGIN, upsert, COMMIT) after the checks and
// before the batch that inserts the record, which no longer names the
// counter (D-3-9 as amended 2026-11-10, TICK-657). Its trips and statements
// are logged for the PR.
func TestCreateTakesItsNumberInATransactionOfItsOwn(t *testing.T) {
	e := migrated(t)
	f := seedTenant(t, e)
	g := &tenancytest.Guard{}
	traced := func(maxConns int32) *pgxpool.Pool {
		cfg := serverPoolConfig(t, appDSN(t, e.platform))
		cfg.ConnConfig.Tracer = g
		if maxConns > 0 {
			cfg.MaxConns = maxConns
		}
		return newPool(t, cfg)
	}
	tenv := env{super: e.super, app: traced(0), alloc: traced(allocatorMaxConns), platform: e.platform}
	p := newPipeline(tenv, nil)

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
			if _, err := write(ctx, tenv, p, f.id, c.cmd); err != nil {
				t.Fatal(err)
			}
			trips := rec.Trips()
			for i, tr := range trips {
				if tr.Tenant != f.id {
					t.Errorf("trip %d ran under tenant %s, want %s: %q", i, tr.Tenant, f.id, tr.SQL)
				}
			}
			has := func(tr tenancytest.Trip, want string) bool {
				return slices.ContainsFunc(tr.SQL, func(s string) bool { return strings.Contains(s, want) })
			}
			up := slices.IndexFunc(trips, func(tr tenancytest.Trip) bool { return has(tr, "record_counters") })
			if up < 1 || up+1 >= len(trips) {
				t.Fatalf("no counter upsert between two trips: %d trips", len(trips))
			}
			if n := len(trips[up].SQL); n != 1 {
				t.Errorf("the counter upsert shares its trip with %d statements: %q", n-1, trips[up].SQL)
			}
			if begin := trips[up-1]; !strings.HasPrefix(strings.ToLower(begin.SQL[0]), "begin") {
				t.Errorf("the trip before the upsert is %q, want the allocation's BEGIN", begin.SQL)
			}
			if end := trips[up+1]; len(end.SQL) != 1 || end.SQL[0] != "commit" {
				t.Errorf("the trip after the upsert is %q, want the allocation's COMMIT", end.SQL)
			}
			ins := slices.IndexFunc(trips, func(tr tenancytest.Trip) bool { return has(tr, "INSERT INTO records") })
			if ins < up+2 || !has(trips[ins], "INSERT INTO audit_log") {
				t.Errorf("the record batch (trip %d) is not after the allocation's COMMIT (trip %d) with the audit entry", ins, up+1)
			}
			for i, tr := range trips {
				if i != up && has(tr, "record_counters") {
					t.Errorf("trip %d also names the counter: %q", i, tr.SQL)
				}
			}
			if end := trips[len(trips)-1]; end.SQL[len(end.SQL)-1] != "commit" {
				t.Errorf("the last trip is %q, want COMMIT", end.SQL)
			}
			t.Logf("create (%s), pipeline only: %d round trips, %d statements; 3 trips of them the number's own transaction",
				c.name, len(trips), rec.Statements())
		})
	}
	if v := g.Violations(); len(v) > 0 {
		t.Errorf("statements ran with no tenant set: %q", v)
	}
}

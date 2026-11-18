//go:build integration

// TICK-868 (deferred from TICK-674, P6-09; ADR-005 §3.6, A7): a create run
// as nava_worker on a type linked to a dated number format reads the
// tenant's time_zone inside the counter upsert, under tenants_tenant
// (20261115100000_tenants_select_worker.sql), and numbers in that zone.

package pipeline_test

import (
	"context"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/records/numbering"
	"github.com/Navaladi2019/TicketingSystem/internal/records/pipeline"
)

// asWorker points e's write pool at nava_worker, with the frozen clock of
// freezeClock on the worker's search_path too. The allocator pool stays
// nava_app: tenancy.AllocPoolConfig refuses any other login (ADR-005 A3), so
// a `gaps` allocation never runs as the worker, while a `gapless` create
// takes its number, zone read included, in the worker's own transaction.
func asWorker(t *testing.T, e *env) {
	t.Helper()
	for _, sql := range []string{
		`GRANT USAGE ON SCHEMA test_clock TO nava_worker`,
		`GRANT SELECT ON test_clock.frozen TO nava_worker`,
		`ALTER ROLE nava_worker IN DATABASE ` + pgx.Identifier{e.platform}.Sanitize() + ` SET search_path = test_clock, pg_catalog, "$user", public`,
	} {
		mustExec(t, e.super, sql)
	}
	t.Cleanup(func() {
		_, _ = e.super.Exec(context.Background(), `ALTER ROLE nava_worker IN DATABASE `+pgx.Identifier{e.platform}.Sanitize()+` RESET search_path`)
	})
	u, err := url.Parse(os.Getenv("NAVA_TEST_PG_URL"))
	if err != nil {
		t.Fatal(err)
	}
	pw, _ := testPassword("nava_worker")
	u.User = url.UserPassword("nava_worker", pw)
	u.Path = "/" + e.platform
	e.app = appPool(t, u.String())
}

// AC: a worker create on a {yyyy}{MM} format numbers in the tenant's zone.
// 20:00 UTC on 31 January 2031 is 01:30 on 1 February in Kolkata, so a
// number dated in UTC (or with no zone read at all, which fails the create)
// fails the test. Both numbering paths.
func TestTICK868WorkerCreateNumbersInTenantZone(t *testing.T) {
	kolkata := mustZone(t, "Asia/Kolkata")
	for _, mode := range modes {
		t.Run(mode, func(t *testing.T) {
			e := migrated(t)
			f := seedFor(t, e, mode)
			setTenantZone(t, e, f, "Asia/Kolkata")
			linkFormat(t, e, f, "{prefix}-{yyyy}{MM}-{seq:4}", numbering.Grow, "")
			at := time.Date(2031, 1, 31, 20, 0, 0, 0, time.UTC)
			freezeClock(t, &e, at)
			asWorker(t, &e)

			cmd := createCmd(f, map[string]any{"title": "from a workflow"})
			cmd.Actor, cmd.Source, cmd.IdempotencyKey, cmd.Causation = f.system(), pipeline.SourceWorkflow, "wf:tick868", workflowCausation()
			res, err := write(context.Background(), e, newPipeline(e, nil), f.id, cmd)
			if err != nil {
				t.Fatalf("worker create: %v", err)
			}
			if want := "INC-203102-0001"; res.Number != want {
				t.Errorf("number %q, want %q (Kolkata's month)", res.Number, want)
			}
			got := counters(t, e, f)
			want := []counter{{"e1|d:203102", 2, ptr(time.Date(2031, 3, 1, 0, 0, 0, 0, kolkata))}}
			if !equalCounters(got, want) {
				t.Errorf("counters %s, want %s", fmtCounters(got), fmtCounters(want))
			}
		})
	}
}

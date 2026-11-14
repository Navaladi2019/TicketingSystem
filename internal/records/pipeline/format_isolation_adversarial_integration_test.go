//go:build integration

// Security & Multi-Tenancy review of TICK-674 (PR #679): the dated counter
// upsert reads the numbering time zone from tenants and computes the scope
// key in SQL. These tests attack that path with a valid tenant A transaction
// aimed at tenant B: B's zone must never date A's number, A and B never share
// a counter row, and a forged tenant on the dated upsert takes nothing.

package pipeline_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/records/numbering"
	"github.com/Navaladi2019/TicketingSystem/internal/records/store"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// At 2031-03-31 20:00 UTC it is already 1 April in Kolkata (tenant A) and
// still 31 March in Los Angeles (tenant B). Both tenants link the same daily
// template. Each number must carry its own tenant's local date and count
// from 1 on its own tenant's row: a zone read from the other tenant, or a
// counter shared across tenants, fails.
func TestAdversarialDatedScopeUsesOnlyTheCreatesTenant(t *testing.T) {
	for _, mode := range modes {
		t.Run(mode, func(t *testing.T) {
			e := migrated(t)
			a, b := seedFor(t, e, mode), seedFor(t, e, mode)
			setTenantZone(t, e, a, "Asia/Kolkata")
			setTenantZone(t, e, b, "America/Los_Angeles")
			const tmpl = "{prefix}-{yyyy}{MM}{dd}-{seq:3}"
			linkFormat(t, e, a, tmpl, numbering.Grow, "")
			linkFormat(t, e, b, tmpl, numbering.Grow, "")
			freezeClock(t, &e, time.Date(2031, 3, 31, 20, 0, 0, 0, time.UTC))
			p := newPipeline(e, nil)

			gotA := []string{
				mustCreate(t, e, p, a, map[string]any{"title": "a1"}).Number,
				mustCreate(t, e, p, a, map[string]any{"title": "a2"}).Number,
			}
			gotB := mustCreate(t, e, p, b, map[string]any{"title": "b1"}).Number

			if want := []string{"INC-20310401-001", "INC-20310401-002"}; !slices.Equal(gotA, want) {
				t.Errorf("tenant A numbers %v, want %v (A's zone, A's counter)", gotA, want)
			}
			if want := "INC-20310331-001"; gotB != want {
				t.Errorf("tenant B number %q, want %q (B's zone, a counter of its own)", gotB, want)
			}
			kolkata, la := mustZone(t, "Asia/Kolkata"), mustZone(t, "America/Los_Angeles")
			if rows, want := counters(t, e, a), []counter{{"e1|d:20310401", 3, ptr(time.Date(2031, 4, 2, 0, 0, 0, 0, kolkata))}}; !equalCounters(rows, want) {
				t.Errorf("tenant A counters %s, want %s", fmtCounters(rows), fmtCounters(want))
			}
			if rows, want := counters(t, e, b), []counter{{"e1|d:20310331", 2, ptr(time.Date(2031, 4, 1, 0, 0, 0, 0, la))}}; !equalCounters(rows, want) {
				t.Errorf("tenant B counters %s, want %s", fmtCounters(rows), fmtCounters(want))
			}
		})
	}
}

// A forged tenant on the dated upsert: inside a tenant A create, the
// allocation batch is queued for tenant B's type and tenant B's id. The
// tenants read is under A's RLS setting, so B's zone is invisible; and the
// record_counters INSERT would fail B's WITH CHECK anyway. The allocation
// must fail and tenant B must have no counter row.
func TestAdversarialForgedTenantOnTheDatedUpsertTakesNothing(t *testing.T) {
	e := migrated(t)
	a, b := seedTenant(t, e), seedTenant(t, e)
	setTenantZone(t, e, b, "America/Los_Angeles")
	defB := *definitionID(t, e, b.id)
	c, err := numbering.Compile("{prefix}-{yyyy}{MM}-{seq:4}", numbering.Grow)
	if err != nil {
		t.Fatal(err)
	}
	c.Epoch = 1
	scope, err := c.Scope()
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	err = tenancy.InTenantTx(ctx, e.app, a.id, func(createTx pgx.Tx) error {
		var got store.Allocated
		_, err := tenancy.AllocTx(ctx, e.alloc, createTx, func(batch *pgx.Batch, tenant uuid.UUID) {
			if tenant != a.id {
				t.Errorf("AllocTx handed the queue tenant %s, want the create's %s", tenant, a.id)
			}
			store.QueueAllocate(batch, b.id, defB, scope, &got)
		})
		return err
	})
	if err == nil {
		t.Fatal("a tenant A allocation took a dated number of tenant B's type")
	}
	if rows := counters(t, e, b); len(rows) != 0 {
		t.Fatalf("tenant B counters %s after a tenant A forged allocation, want none", fmtCounters(rows))
	}
}

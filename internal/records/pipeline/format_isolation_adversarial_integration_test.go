//go:build integration

// Security & Multi-Tenancy review of TICK-674 (PR #679): the dated counter
// upsert reads the numbering time zone from tenants and computes the scope
// key in SQL. These tests attack that path with a valid tenant A transaction
// aimed at tenant B: B's zone must never date A's number, A and B never share
// a counter row, and a forged tenant on the dated upsert takes nothing.

package pipeline_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

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

// A forged tenant on the dated upsert: inside a tenant A create, the counter
// upsert is queued for tenant B's id and tenant B's type, on both create
// paths: `gaps` (store.QueueAllocate, in tenancy.AllocTx) and `gapless`
// (store.QueueInsertNumbered, in the create's own transaction). Two layers
// stop it, and each subtest pins the failure to one of them, so the other
// failing first cannot hide its removal:
//
//   - the tenant's zone: the format has none, so the upsert reads B's
//     tenants.time_zone under A's RLS setting, sees no row, has no zone and
//     writes and returns nothing. The batch fails with pgx.ErrNoRows.
//     Without the tenants policy it would read B's zone and go on to the
//     INSERT (SQLSTATE 42501 below), so this subtest fails.
//   - the format's zone: the format names its zone, so the tenants row is
//     never read and the INSERT reaches record_counters' policies, which
//     refuse B's tenant_id with SQLSTATE 42501: the INSERT's WITH CHECK, and
//     the SELECT policy RETURNING applies to the new row. Without them the
//     upsert takes B's number, so this subtest fails.
//
// Each subtest also asserts the upsert read back no counter value, so on the
// `gapless` path the records INSERT's own WITH CHECK (42501 as well) cannot
// pass for the counter's. Either way tenant B has no counter row and no
// record afterwards.
func TestAdversarialForgedTenantOnTheDatedUpsertTakesNothing(t *testing.T) {
	zones := []struct {
		name, zone string
		is         func(error) bool
		want       string
	}{
		{"tenant zone", "", func(err error) bool { return errors.Is(err, pgx.ErrNoRows) }, "pgx.ErrNoRows (B's tenants row invisible)"},
		{"format zone", "America/Los_Angeles", func(err error) bool {
			var pe *pgconn.PgError
			return errors.As(err, &pe) && pe.Code == pgInsufficientPrivilege
		}, "SQLSTATE 42501 (record_counters WITH CHECK)"},
	}
	for _, mode := range modes {
		for _, z := range zones {
			t.Run(mode+"/"+z.name, func(t *testing.T) {
				e := migrated(t)
				a, b := seedTenant(t, e), seedTenant(t, e)
				setTenantZone(t, e, b, "America/Los_Angeles")
				defB := *definitionID(t, e, b.id)
				c, err := numbering.CompileFormat(numbering.Format{
					Template: "{prefix}-{yyyy}{MM}-{seq:4}", Overflow: numbering.Grow, TimeZone: z.zone,
				})
				if err != nil {
					t.Fatal(err)
				}
				c.Epoch = 1
				taken, err := forgedDatedUpsert(t, e, mode, a, b.id, defB, c)
				if taken != 0 {
					t.Fatalf("the counter upsert of a tenant A %s create took number %d of tenant B's type (err %v)", mode, taken, err)
				}
				if err == nil {
					t.Fatalf("a tenant A %s create took a dated number of tenant B's type", mode)
				}
				if !z.is(err) {
					t.Fatalf("a tenant A %s create of tenant B's type failed with %v, want %s", mode, err, z.want)
				}
				if rows := counters(t, e, b); len(rows) != 0 {
					t.Fatalf("tenant B counters %s after a tenant A forged %s create, want none", fmtCounters(rows), mode)
				}
				if n := count(t, e.super, `SELECT count(*) FROM records WHERE tenant_id = $1`, b.id); n != 0 {
					t.Fatalf("tenant B has %d records after a tenant A forged %s create, want none", n, mode)
				}
			})
		}
	}
}

// pgInsufficientPrivilege is SQLSTATE 42501, which a row a policy's WITH
// CHECK refuses raises.
const pgInsufficientPrivilege = "42501"

// forgedDatedUpsert runs, in a tenant A create's transaction, the mode's
// counter upsert queued for tenant forged and its type def under format c:
// `gaps` through tenancy.AllocTx and store.QueueAllocate, `gapless` as the
// create's own batch from store.QueueInsertNumbered. It returns the counter
// value the upsert read back, 0 when it took none, and the transaction's
// error. The value pins a failure to the upsert itself: on the `gapless`
// path a forged upsert that took a number is still refused by the records
// INSERT after it (records' WITH CHECK, 42501 too), which must not stand in
// for the counter's own check.
func forgedDatedUpsert(t *testing.T, e env, mode string, a tenantFixture, forged, def uuid.UUID, c numbering.Compiled) (int64, error) {
	t.Helper()
	ctx := context.Background()
	var taken int64
	err := tenancy.InTenantTx(ctx, e.app, a.id, func(createTx pgx.Tx) error {
		if mode == "gaps" {
			scope, err := c.Scope()
			if err != nil {
				t.Fatal(err)
			}
			var got store.Allocated
			_, err = tenancy.AllocTx(ctx, e.alloc, createTx, func(batch *pgx.Batch, tenant uuid.UUID) {
				if tenant != a.id {
					t.Errorf("AllocTx handed the queue tenant %s, want the create's %s", tenant, a.id)
				}
				store.QueueAllocate(batch, forged, def, scope, &got)
			})
			taken = got.Number
			return err
		}
		rec := &store.Record{
			TenantID: forged, ID: uuid.New(), Version: 1, EntityDefinitionID: def, SchemaVersion: 1,
			State: "new", CreatedBy: &a.principal.PrincipalID, Data: map[string]any{"title": "forged"},
		}
		batch := &pgx.Batch{}
		var got store.Numbered
		if err := store.QueueInsertNumbered(batch, rec, "INC", c, nil, &got); err != nil {
			t.Fatal(err)
		}
		err := createTx.SendBatch(ctx, batch).Close()
		taken = got.N
		return err
	})
	return taken, err
}

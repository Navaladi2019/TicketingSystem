//go:build integration

// TICK-657, ADR-005 A1–A4: tenancy.AllocTx, the record-number allocation
// transaction. It is one pipelined send, scoped to the create transaction's
// tenant, and its SET LOCAL synchronous_commit = off ends with it: the next
// use of the same pooled connection reads the session default again, with no
// tenant.

package tenancy_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy/tenancytest"
)

// sessionSQL reads, outside any transaction, what the next transaction on a
// connection starts with: synchronous_commit, its session default (the
// value RESET would restore), the tenant and the backend.
const sessionSQL = `SELECT current_setting('synchronous_commit'),
  (SELECT reset_val FROM pg_settings WHERE name = 'synchronous_commit'),
  coalesce(current_setting('app.tenant_id', true), ''), pg_backend_pid()`

type session struct {
	sync, deflt, tenant string
	pid                 uint32
}

func readSession(t *testing.T, pool *pgxpool.Pool) session {
	t.Helper()
	var s session
	if err := pool.QueryRow(t.Context(), sessionSQL).Scan(&s.sync, &s.deflt, &s.tenant, &s.pid); err != nil {
		t.Fatal(err)
	}
	return s
}

// allocPool is a one-connection pool beside app (the same role, the same
// Guard), whose session default is synchronous_commit = on: the test
// cluster runs with it off server-wide, which would hide a leak.
func allocPool(t *testing.T, app *pgxpool.Pool) *pgxpool.Pool {
	t.Helper()
	cfg := app.Config().Copy()
	cfg.MaxConns = 1
	cfg.ConnConfig.RuntimeParams["synchronous_commit"] = "on"
	p, err := pgxpool.NewWithConfig(t.Context(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(p.Close)
	return p
}

// AC: AllocTx runs in one send under the create transaction's tenant, with
// synchronous_commit off inside it; afterwards the same pooled connection
// reads synchronous_commit as its default and no tenant.
func TestAllocTxIsOneSendAndLeavesNoSetting(t *testing.T) {
	tenantA, tenantB := uuid.New(), uuid.New()
	table, app, _ := rlsTable(t, tenantA, tenantB, 1)
	alloc := allocPool(t, app)
	before := readSession(t, alloc)
	if before.sync != "on" || before.deflt != "on" || before.tenant != "" {
		t.Fatalf("before: %+v, want synchronous_commit on by default and no tenant", before)
	}

	for _, tc := range []struct {
		tenant uuid.UUID
		row    int
	}{{tenantA, 1}, {tenantB, 2}} {
		ctx, rec := tenancytest.Record(t.Context())
		var (
			inside session
			rows   []int
			info   tenancy.AllocInfo
		)
		err := tenancy.InTenantTx(ctx, app, tc.tenant, func(createTx pgx.Tx) error {
			var err error
			info, err = tenancy.AllocTx(ctx, alloc, createTx, func(b *pgx.Batch, tenant uuid.UUID) {
				b.Queue(sessionSQL).QueryRow(func(row pgx.Row) error {
					return row.Scan(&inside.sync, &inside.deflt, &inside.tenant, &inside.pid)
				})
				b.Queue("SELECT n FROM " + table).Query(func(r pgx.Rows) error {
					var err error
					rows, err = pgx.CollectRows(r, pgx.RowTo[int])
					return err
				})
			})
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
		if info.Tenant != tc.tenant || inside.sync != "off" || inside.tenant != tc.tenant.String() ||
			!slices.Equal(rows, []int{tc.row}) || inside.pid != before.pid {
			t.Fatalf("inside AllocTx for %s: %+v, rows %v, info %+v; want synchronous_commit off, its tenant, its row only, backend %d",
				tc.tenant, inside, rows, info, before.pid)
		}
		// One trip on the allocator connection: the AllocTx batch.
		var allocTrips []tenancytest.Trip
		for _, tr := range rec.Trips() {
			if tr.Batch && tr.SQL[0] == "BEGIN" {
				allocTrips = append(allocTrips, tr)
			}
		}
		// BEGIN, the tenant, SET LOCAL, the caller's two, COMMIT; the Guard
		// sees it run under the tenant from its first statement.
		if len(allocTrips) != 1 || allocTrips[0].Tenant != tc.tenant || len(allocTrips[0].SQL) != 6 {
			t.Fatalf("allocation trips %+v, want one batch of 6 statements under %s", allocTrips, tc.tenant)
		}
		after := readSession(t, alloc)
		if after != before {
			t.Fatalf("after AllocTx: %+v, want the session as before %+v (same backend, default synchronous_commit, no tenant)", after, before)
		}
	}
}

// A failed allocation commits nothing and leaks nothing: the connection is
// destroyed, not returned, and the pool's next one starts clean. A create
// transaction with no tenant allocates nothing at all.
func TestAllocTxFailureLeaksNothing(t *testing.T) {
	tenantA, tenantB := uuid.New(), uuid.New()
	table, app, _ := rlsTable(t, tenantA, tenantB, 1)
	alloc := allocPool(t, app)
	before := readSession(t, alloc)
	err := tenancy.InTenantTx(t.Context(), app, tenantA, func(createTx pgx.Tx) error {
		_, err := tenancy.AllocTx(t.Context(), alloc, createTx, func(b *pgx.Batch, _ uuid.UUID) {
			b.Queue("UPDATE " + table + " SET n = 99")
			b.Queue("SELECT 1 / 0")
		})
		return err
	})
	if err == nil {
		t.Fatal("a failing allocation reported success")
	}
	after := readSession(t, alloc)
	if after.pid == before.pid || after.sync != "on" || after.tenant != "" {
		t.Fatalf("after a failed AllocTx: %+v (before %+v); want a new backend, default synchronous_commit, no tenant", after, before)
	}
	var n int
	if err := tenancy.InTenantTx(t.Context(), app, tenantA, func(tx pgx.Tx) error {
		return tx.QueryRow(t.Context(), "SELECT n FROM "+table).Scan(&n)
	}); err != nil || n != 1 {
		t.Fatalf("tenant A's row = %d, %v; want 1: the failed allocation committed", n, err)
	}

	acquired := alloc.Stat().AcquireCount()
	err = pgx.BeginFunc(t.Context(), app, func(plain pgx.Tx) error {
		_, err := tenancy.AllocTx(t.Context(), alloc, plain, func(*pgx.Batch, uuid.UUID) {
			t.Error("queue called for a transaction with no tenant")
		})
		return err
	})
	if !errors.Is(err, tenancy.ErrNoTenant) || alloc.Stat().AcquireCount() != acquired {
		t.Fatalf("AllocTx from an unscoped transaction: %v, %d acquires; want ErrNoTenant and none", err, alloc.Stat().AcquireCount()-acquired)
	}
}

// AC (ADR-005 A3, §5): the allocator pool refuses to start under a role
// other than nava_app, and a session that bypasses RLS is refused as it
// connects. (nava_app itself, and nava_app given BYPASSRLS, are covered in
// internal/records/pipeline, where the platform roles exist.)
func TestAllocatorPoolRefusesOtherRoles(t *testing.T) {
	_, app, _ := rlsTable(t, uuid.New(), uuid.New(), 1)
	cfg := app.Config().Copy()
	if err := tenancy.AllocPoolConfig(cfg); !errors.Is(err, tenancy.ErrAllocatorRole) {
		t.Fatalf("pool of role %s: %v, want ErrAllocatorRole", cfg.ConnConfig.User, err)
	}
	if err := tenancy.CheckAllocatorRole(t.Context(), app); !errors.Is(err, tenancy.ErrAllocatorRole) {
		t.Fatalf("role check as %s: %v, want ErrAllocatorRole", cfg.ConnConfig.User, err)
	}
	super, err := pgx.Connect(t.Context(), env(t, "NAVA_TEST_PG_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = super.Close(context.Background()) }()
	if err := tenancy.CheckAllocatorRole(t.Context(), super); !errors.Is(err, tenancy.ErrAllocatorRole) {
		t.Fatalf("role check as the superuser: %v, want ErrAllocatorRole", err)
	}
}

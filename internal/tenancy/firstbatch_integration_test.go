//go:build integration

// TICK-692: BeginInFirstBatch. The transaction opens, and its tenant is set,
// in the first statements of the first batch fn sends, so that batch costs
// no BEGIN round trip of its own. Nothing of the caller's runs before the
// tenant, a failure anywhere in the batch keeps nothing, and the connection
// goes back to the pool with no transaction and no tenant.

package tenancy_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy/tenancytest"
)

// TestFirstBatchRunsUnderTheTenant: tenants A and B interleaved on one
// connection; each first batch sees its own tenant from its first statement
// and only its own row, in two round trips (the batch, COMMIT).
func TestFirstBatchRunsUnderTheTenant(t *testing.T) {
	tenantA, tenantB := uuid.New(), uuid.New()
	table, app, g := rlsTable(t, tenantA, tenantB, 1)
	for _, tc := range []struct {
		tenant uuid.UUID
		row    int
	}{{tenantA, 1}, {tenantB, 2}, {tenantA, 1}} {
		before := len(g.Violations()) // readTenant below runs with no tenant on purpose
		ctx, rec := tenancytest.Record(t.Context())
		var seen string
		var rows []int
		err := tenancy.InTenantTx(ctx, app, tc.tenant, func(tx pgx.Tx) error {
			b := &pgx.Batch{}
			b.Queue(tenantSQL).QueryRow(func(row pgx.Row) error {
				var pid uint32
				return row.Scan(&pid, &seen)
			})
			b.Queue("SELECT n FROM " + table).Query(func(r pgx.Rows) error {
				var err error
				rows, err = pgx.CollectRows(r, pgx.RowTo[int])
				return err
			})
			return tx.SendBatch(ctx, b).Close()
		}, tenancy.BeginInFirstBatch())
		if err != nil {
			t.Fatal(err)
		}
		if seen != tc.tenant.String() {
			t.Errorf("tenant %s: the batch's first statement saw app.tenant_id %q", tc.tenant, seen)
		}
		if fmt.Sprint(rows) != fmt.Sprint([]int{tc.row}) {
			t.Errorf("tenant %s: rows %v, want only its own [%d]", tc.tenant, rows, tc.row)
		}
		trips := rec.Trips()
		if len(trips) != 2 || !trips[0].Batch || trips[0].SQL[0] != "BEGIN" || trips[0].Tenant != tc.tenant ||
			trips[1].SQL[0] != "commit" {
			t.Errorf("tenant %s: trips %+v, want the batch opened by BEGIN and the tenant, then COMMIT", tc.tenant, trips)
		}
		if v := g.Violations()[before:]; len(v) > 0 {
			t.Errorf("tenant %s: statements ran with no tenant set: %q", tc.tenant, v)
		}
		if _, s := readTenant(t, app); s != "" {
			t.Errorf("after tenant %s's transaction app.tenant_id = %q, want none", tc.tenant, s)
		}
	}
}

// TestFirstBatchFailureKeepsNothing: a first batch that writes tenant A's
// row and then tries to write B's fails on B's (42501); A's row is gone, and
// so is anything queued after the failure. The same holds when the batch
// succeeds and fn then fails.
func TestFirstBatchFailureKeepsNothing(t *testing.T) {
	tenantA, tenantB := uuid.New(), uuid.New()
	table, app, g := rlsTable(t, tenantA, tenantB, 1)
	ctx := t.Context()

	ran := false
	err := tenancy.InTenantTx(ctx, app, tenantA, func(tx pgx.Tx) error {
		b := &pgx.Batch{}
		b.Queue("INSERT INTO "+table+" VALUES ($1, 7)", tenantA)
		b.Queue("INSERT INTO "+table+" VALUES ($1, 8)", tenantB)
		b.Queue("INSERT INTO "+table+" VALUES ($1, 9)", tenantA).Exec(func(pgconn.CommandTag) error {
			ran = true
			return nil
		})
		return tx.SendBatch(ctx, b).Close()
	}, tenancy.BeginInFirstBatch())
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "42501" {
		t.Fatalf("err = %v, want the 42501 of B's row", err)
	}
	if ran {
		t.Error("a statement after the failed one ran")
	}

	sentinel := errors.New("fn failed after its first batch")
	err = tenancy.InTenantTx(ctx, app, tenantA, func(tx pgx.Tx) error {
		b := &pgx.Batch{}
		b.Queue("INSERT INTO "+table+" VALUES ($1, 10)", tenantA)
		if err := tx.SendBatch(ctx, b).Close(); err != nil {
			return err
		}
		return sentinel
	}, tenancy.BeginInFirstBatch())
	if !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want fn's", err)
	}

	if v := g.Violations(); len(v) > 0 {
		t.Errorf("violations: %q", v)
	}
	if _, s := readTenant(t, app); s != "" {
		t.Errorf("after the failed transactions app.tenant_id = %q, want none", s)
	}
	if err := tenancy.InTenantTx(ctx, app, tenantA, func(tx pgx.Tx) error {
		var n int
		if err := tx.QueryRow(ctx, "SELECT count(*) FROM "+table+" WHERE n IN (7, 8, 9, 10)").Scan(&n); err != nil {
			return err
		}
		if n != 0 {
			t.Errorf("%d rows of the failed transactions were kept", n)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

// TestFirstBatchCannotReachAnotherTenant: a first batch in A's transaction
// that names B reads none of B's rows and changes none.
func TestFirstBatchCannotReachAnotherTenant(t *testing.T) {
	tenantA, tenantB := uuid.New(), uuid.New()
	table, app, _ := rlsTable(t, tenantA, tenantB, 1)
	ctx := t.Context()
	var seen, updated int64 = -1, -1
	err := tenancy.InTenantTx(ctx, app, tenantA, func(tx pgx.Tx) error {
		b := &pgx.Batch{}
		b.Queue("SELECT count(*) FROM "+table+" WHERE tenant_id = $1", tenantB).QueryRow(func(r pgx.Row) error {
			return r.Scan(&seen)
		})
		b.Queue("UPDATE "+table+" SET n = 99 WHERE tenant_id = $1", tenantB).Exec(func(ct pgconn.CommandTag) error {
			updated = ct.RowsAffected()
			return nil
		})
		return tx.SendBatch(ctx, b).Close()
	}, tenancy.BeginInFirstBatch())
	if err != nil {
		t.Fatal(err)
	}
	if seen != 0 || updated != 0 {
		t.Errorf("A's first batch saw %d and updated %d of B's rows, want 0 and 0", seen, updated)
	}
}

// TestFirstBatchOtherFirstSends: a transaction that sends nothing sends
// nothing; one that only queues at commit commits in one round trip; one
// whose first send is not a batch opens with the usual BEGIN round trip.
func TestFirstBatchOtherFirstSends(t *testing.T) {
	tenantA, tenantB := uuid.New(), uuid.New()
	table, app, g := rlsTable(t, tenantA, tenantB, 1)

	ctx, rec := tenancytest.Record(t.Context())
	if err := tenancy.InTenantTx(ctx, app, tenantA, func(pgx.Tx) error { return nil }, tenancy.BeginInFirstBatch()); err != nil {
		t.Fatal(err)
	}
	if trips := rec.Trips(); len(trips) != 0 {
		t.Errorf("an empty transaction sent %+v, want nothing", trips)
	}

	ctx, rec = tenancytest.Record(t.Context())
	if err := tenancy.InTenantTx(ctx, app, tenantA, func(tx pgx.Tx) error {
		if !tenancy.ExecAtCommit(tx, "INSERT INTO "+table+" VALUES ($1, 11)", tenantA) {
			t.Error("ExecAtCommit refused a BeginInFirstBatch transaction")
		}
		return nil
	}, tenancy.BeginInFirstBatch()); err != nil {
		t.Fatal(err)
	}
	if trips := rec.Trips(); len(trips) != 1 || fmt.Sprint(trips[0].SQL[:2]) != "[BEGIN SELECT set_config('app.tenant_id', $1, true)]" ||
		trips[0].SQL[len(trips[0].SQL)-1] != "COMMIT" || trips[0].Tenant != tenantA {
		t.Errorf("a commit-only transaction sent %+v, want one batch: BEGIN, the tenant, the insert, COMMIT", trips)
	}

	ctx, rec = tenancytest.Record(t.Context())
	if err := tenancy.InTenantTx(ctx, app, tenantA, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, "INSERT INTO "+table+" VALUES ($1, 12)", tenantA)
		return err
	}, tenancy.BeginInFirstBatch()); err != nil {
		t.Fatal(err)
	}
	if trips := rec.Trips(); len(trips) != 3 || trips[0].Tenant != tenantA || !strings.HasPrefix(trips[0].SQL[0], "BEGIN; SELECT set_config") {
		t.Errorf("an Exec-first transaction sent %+v, want the tenancy BEGIN, the insert, COMMIT", trips)
	}

	if v := g.Violations(); len(v) > 0 {
		t.Errorf("violations: %q", v)
	}
	var got []int
	if err := tenancy.InTenantTx(t.Context(), app, tenantA, func(tx pgx.Tx) error {
		rows, err := tx.Query(t.Context(), "SELECT n FROM "+table+" ORDER BY n")
		if err != nil {
			return err
		}
		got, err = pgx.CollectRows(rows, pgx.RowTo[int])
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(got) != "[1 11 12]" {
		t.Errorf("tenant A's rows %v, want [1 11 12]", got)
	}
}

// TestFirstBatchResultsReadInOrder: a caller that reads its batch's results
// one by one gets its own first, never the BEGIN's or the tenant's.
func TestFirstBatchResultsReadInOrder(t *testing.T) {
	pool, _, _ := newPool(t, env(t, "NAVA_TEST_PG_URL"), 1)
	tenantA := uuid.New()
	err := tenancy.InTenantTx(t.Context(), pool, tenantA, func(tx pgx.Tx) error {
		b := &pgx.Batch{}
		b.Queue("SELECT 41")
		b.Queue("SELECT current_setting('app.tenant_id')")
		br := tx.SendBatch(t.Context(), b)
		var n int
		var s string
		if err := br.QueryRow().Scan(&n); err != nil {
			return err
		}
		if err := br.QueryRow().Scan(&s); err != nil {
			return err
		}
		if n != 41 || s != tenantA.String() {
			t.Errorf("read %d, %q; want 41, %q", n, s, tenantA)
		}
		return br.Close()
	}, tenancy.BeginInFirstBatch())
	if err != nil {
		t.Fatal(err)
	}
}

// TestFirstBatchSavepoint: a savepoint of a BeginInFirstBatch transaction
// keeps the tenant; rolling it back undoes its write only.
func TestFirstBatchSavepoint(t *testing.T) {
	tenantA, tenantB := uuid.New(), uuid.New()
	table, app, g := rlsTable(t, tenantA, tenantB, 1)
	ctx := context.Background()
	err := tenancy.InTenantTx(ctx, app, tenantA, func(tx pgx.Tx) error {
		b := &pgx.Batch{}
		b.Queue("INSERT INTO "+table+" VALUES ($1, 20)", tenantA)
		if err := tx.SendBatch(ctx, b).Close(); err != nil {
			return err
		}
		sp, err := tx.Begin(ctx)
		if err != nil {
			return err
		}
		if got, err := tenancy.TenantOf(ctx, sp); err != nil || got != tenantA {
			t.Errorf("savepoint TenantOf = %s, %v; want %s", got, err, tenantA)
		}
		if _, err := sp.Exec(ctx, "INSERT INTO "+table+" VALUES ($1, 21)", tenantA); err != nil {
			return err
		}
		if err := sp.Rollback(ctx); err != nil {
			return err
		}
		sp2, err := tx.Begin(ctx)
		if err != nil {
			return err
		}
		if _, err := sp2.Exec(ctx, "INSERT INTO "+table+" VALUES ($1, 22)", tenantA); err != nil {
			return err
		}
		return sp2.Commit(ctx)
	}, tenancy.BeginInFirstBatch())
	if err != nil {
		t.Fatal(err)
	}
	if v := g.Violations(); len(v) > 0 {
		t.Errorf("violations: %q", v)
	}
	var got []int
	if err := tenancy.InTenantTx(ctx, app, tenantA, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, "SELECT n FROM "+table+" WHERE n >= 20 ORDER BY n")
		if err != nil {
			return err
		}
		got, err = pgx.CollectRows(rows, pgx.RowTo[int])
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(got) != "[20 22]" {
		t.Errorf("rows %v, want [20 22]: the rolled-back savepoint's write undone, the rest kept", got)
	}
}

//go:build integration

// TICK-648: the tenant GUC path with batched round trips. The tenancy BEGIN
// sets app.tenant_id in the round trip that opens the transaction, so every
// statement and every pgx.Batch fn sends runs under the tenant; statements
// queued with ExecAtCommit ride in the COMMIT's round trip.

package tenancy_test

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy/tenancytest"
)

// guardedPool is a pool on url whose tracer is a tenancytest.Guard, with at
// most maxConns connections.
func guardedPool(t *testing.T, url string, maxConns int32) (*pgxpool.Pool, *tenancytest.Guard) {
	t.Helper()
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	cfg.MaxConns = maxConns
	g := &tenancytest.Guard{}
	cfg.ConnConfig.Tracer = g
	pool, err := pgxpool.NewWithConfig(t.Context(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool, g
}

// rlsTable is a table under the platform's tenant policy, as the platform
// tables have it, with one row for each of tenantA (n = 1) and tenantB
// (n = 2), and a pool (with a Guard) of a non-superuser role subject to it.
func rlsTable(t *testing.T, tenantA, tenantB uuid.UUID, maxConns int32) (table string, app *pgxpool.Pool, g *tenancytest.Guard) {
	t.Helper()
	role := "nava_it_tb_" + strings.ReplaceAll(uuid.NewString()[:8], "-", "")
	const password = "test-only-tenancy-batch"
	super, _, _ := newPool(t, env(t, "NAVA_TEST_PG_URL"), 2)
	ctx := t.Context()
	table = "tenancy_batch_" + strings.ReplaceAll(uuid.NewString()[:8], "-", "")
	t.Cleanup(func() {
		_, _ = super.Exec(context.Background(), "DROP TABLE IF EXISTS "+table)
		_, _ = super.Exec(context.Background(), "DROP ROLE IF EXISTS "+role)
	})
	policy := "tenant_id = nullif(current_setting('app.tenant_id', true), '')::uuid"
	for _, sql := range []string{
		"CREATE ROLE " + role + " LOGIN PASSWORD '" + password + "'",
		"CREATE TABLE " + table + " (tenant_id uuid NOT NULL, n int NOT NULL)",
		"ALTER TABLE " + table + " ENABLE ROW LEVEL SECURITY",
		"ALTER TABLE " + table + " FORCE ROW LEVEL SECURITY",
		"CREATE POLICY tenant_isolation ON " + table + " USING (" + policy + ") WITH CHECK (" + policy + ")",
		"GRANT SELECT, INSERT, UPDATE, DELETE ON " + table + " TO " + role,
		fmt.Sprintf("INSERT INTO %s VALUES ('%s', 1), ('%s', 2)", table, tenantA, tenantB),
	} {
		if _, err := super.Exec(ctx, sql); err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
	}
	u, err := url.Parse(env(t, "NAVA_TEST_PG_URL"))
	if err != nil {
		t.Fatal(err)
	}
	u.User = url.UserPassword(role, password)
	app, g = guardedPool(t, u.String(), maxConns)
	return table, app, g
}

// TestBatchRunsUnderTheTenantFromItsFirstStatement: the first statement of
// the first batch a transaction sends, and every one after it, already sees
// the tenant; there is no statement before the batch that sets it. Two
// tenants in turn on one connection each see only their own.
func TestBatchRunsUnderTheTenantFromItsFirstStatement(t *testing.T) {
	tenantA, tenantB := uuid.New(), uuid.New()
	table, app, g := rlsTable(t, tenantA, tenantB, 1)
	for _, tc := range []struct {
		tenant uuid.UUID
		row    int
	}{{tenantA, 1}, {tenantB, 2}, {tenantA, 1}} {
		ctx, rec := tenancytest.Record(t.Context())
		var seen []string
		var rows []int
		err := tenancy.InTenantTx(ctx, app, tc.tenant, func(tx pgx.Tx) error {
			b := &pgx.Batch{}
			for range 3 {
				b.Queue(tenantSQL).QueryRow(func(row pgx.Row) error {
					var pid uint32
					var s string
					err := row.Scan(&pid, &s)
					seen = append(seen, s)
					return err
				})
				b.Queue("SELECT n FROM " + table).Query(func(r pgx.Rows) error {
					ns, err := pgx.CollectRows(r, pgx.RowTo[int])
					rows = append(rows, ns...)
					return err
				})
			}
			return tx.SendBatch(ctx, b).Close()
		})
		if err != nil {
			t.Fatal(err)
		}
		for i, s := range seen {
			if s != tc.tenant.String() {
				t.Errorf("tenant %s: batch statement %d saw app.tenant_id %q", tc.tenant, 2*i, s)
			}
		}
		for _, n := range rows {
			if n != tc.row {
				t.Errorf("tenant %s: batch read row %d, want only its own %d", tc.tenant, n, tc.row)
			}
		}
		if len(rows) != 3 {
			t.Errorf("tenant %s: batch read %d rows, want 3 (its row, three times)", tc.tenant, len(rows))
		}
		// BEGIN with the tenant, the batch, COMMIT: three round trips, and
		// nothing before the BEGIN.
		trips := rec.Trips()
		if len(trips) != 3 || !trips[1].Batch || trips[1].Tenant != tc.tenant {
			t.Errorf("tenant %s: trips %+v, want BEGIN with the tenant, the batch under it, COMMIT", tc.tenant, trips)
		}
	}
	if v := g.Violations(); len(v) > 0 {
		t.Errorf("statements ran with no tenant set: %q", v)
	}
}

// TestBatchCannotReachAnotherTenant: a batch in tenant A's transaction that
// names tenant B reads none of B's rows, changes none, and cannot insert one
// for B; the insert's 42501 rolls the whole transaction back. A batch sent
// outside InTenantTx, on the connection A just used, has no tenant and sees
// no row (the empty-tenant case).
func TestBatchCannotReachAnotherTenant(t *testing.T) {
	tenantA, tenantB := uuid.New(), uuid.New()
	table, app, _ := rlsTable(t, tenantA, tenantB, 1)
	ctx := t.Context()

	err := tenancy.InTenantTx(ctx, app, tenantA, func(tx pgx.Tx) error {
		b := &pgx.Batch{}
		b.Queue("SELECT count(*) FROM "+table+" WHERE tenant_id = $1", tenantB).QueryRow(func(row pgx.Row) error {
			var n int
			if err := row.Scan(&n); err != nil {
				return err
			}
			if n != 0 {
				t.Errorf("tenant A's batch counted %d of tenant B's rows", n)
			}
			return nil
		})
		b.Queue("UPDATE "+table+" SET n = 20 WHERE tenant_id = $1", tenantB).Exec(func(ct pgconn.CommandTag) error {
			if ct.RowsAffected() != 0 {
				t.Errorf("tenant A's batch updated %d of tenant B's rows", ct.RowsAffected())
			}
			return nil
		})
		b.Queue("UPDATE " + table + " SET n = 10")
		b.Queue("INSERT INTO "+table+" VALUES ($1, 3)", tenantB)
		return tx.SendBatch(ctx, b).Close()
	})
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "42501" {
		t.Fatalf("insert for tenant B in tenant A's batch: %v, want 42501 (row-level security)", err)
	}

	// Nothing of A's batch committed: A's row is still 1.
	if err := tenancy.InTenantTx(ctx, app, tenantA, func(tx pgx.Tx) error {
		var n int
		if err := tx.QueryRow(ctx, "SELECT n FROM "+table).Scan(&n); err != nil {
			return err
		}
		if n != 1 {
			t.Errorf("tenant A's row is %d after the rolled-back batch, want 1", n)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	// The same (only) connection, outside InTenantTx: no tenant, no row.
	b := &pgx.Batch{}
	b.Queue(tenantSQL).QueryRow(func(row pgx.Row) error {
		var pid uint32
		var s string
		if err := row.Scan(&pid, &s); err != nil {
			return err
		}
		if s != "" {
			t.Errorf("a batch outside InTenantTx sees app.tenant_id %q, want none", s)
		}
		return nil
	})
	b.Queue("SELECT count(*) FROM " + table).QueryRow(func(row pgx.Row) error {
		var n int
		if err := row.Scan(&n); err != nil {
			return err
		}
		if n != 0 {
			t.Errorf("a batch outside InTenantTx sees %d rows, want none", n)
		}
		return nil
	})
	if err := app.SendBatch(ctx, b).Close(); err != nil {
		t.Fatal(err)
	}
}

// TestBeginSetsTheTenantInOneRoundTrip: InTenantTx sends the tenancy BEGIN
// as its first round trip, and TenantOf on the transaction it hands fn sends
// nothing.
func TestBeginSetsTheTenantInOneRoundTrip(t *testing.T) {
	pool, g := guardedPool(t, env(t, "NAVA_TEST_PG_URL"), 1)
	tenantA := uuid.New()
	ctx, rec := tenancytest.Record(t.Context())
	err := tenancy.InTenantTx(ctx, pool, tenantA, func(tx pgx.Tx) error {
		for range 3 {
			got, err := tenancy.TenantOf(ctx, tx)
			if err != nil || got != tenantA {
				t.Errorf("TenantOf = %s, %v; want %s", got, err, tenantA)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	trips := rec.Trips()
	if len(trips) != 2 || trips[0].Tenant != tenantA || trips[1].SQL[0] != "commit" {
		t.Errorf("trips %+v, want the tenancy BEGIN and the commit only", trips)
	}
	if v := g.Violations(); len(v) > 0 {
		t.Errorf("violations: %q", v)
	}
}

// TestExecAtCommit: statements queued with ExecAtCommit run in the COMMIT's
// round trip, after fn's own statements, and commit with them; when one
// fails, nothing of the transaction is kept, and the connection goes back to
// the pool with no transaction and no tenant.
func TestExecAtCommit(t *testing.T) {
	tenantA, tenantB := uuid.New(), uuid.New()
	table, app, g := rlsTable(t, tenantA, tenantB, 1)

	ctx, rec := tenancytest.Record(t.Context())
	err := tenancy.InTenantTx(ctx, app, tenantA, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, "INSERT INTO "+table+" VALUES ($1, 5)", tenantA); err != nil {
			return err
		}
		if !tenancy.ExecAtCommit(tx, "UPDATE "+table+" SET n = n * 10 WHERE n = 5") {
			t.Error("ExecAtCommit refused an InTenantTx transaction")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	trips := rec.Trips()
	last := trips[len(trips)-1]
	if len(trips) != 3 || !last.Batch || len(last.SQL) != 2 || last.SQL[1] != "COMMIT" {
		t.Errorf("trips %+v, want BEGIN, the insert, and one batch of the queued update and COMMIT", trips)
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
	if fmt.Sprint(got) != "[1 50]" {
		t.Errorf("tenant A's rows %v, want [1 50]: the insert and the queued update committed together", got)
	}

	// A queued statement that fails: B's row cannot be written from A, so
	// the COMMIT never runs and A's insert is gone too.
	err = tenancy.InTenantTx(t.Context(), app, tenantA, func(tx pgx.Tx) error {
		if _, err := tx.Exec(t.Context(), "INSERT INTO "+table+" VALUES ($1, 7)", tenantA); err != nil {
			return err
		}
		tenancy.ExecAtCommit(tx, "INSERT INTO "+table+" VALUES ($1, 8)", tenantB)
		return nil
	})
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "42501" || !strings.HasPrefix(err.Error(), "tenancy: commit: ") {
		t.Fatalf("err = %v, want the queued insert's 42501, wrapped as the commit", err)
	}
	// Checked before readTenant, which runs outside InTenantTx on purpose.
	if v := g.Violations(); len(v) > 0 {
		t.Errorf("violations: %q", v)
	}
	if _, s := readTenant(t, app); s != "" {
		t.Errorf("after the failed commit batch app.tenant_id = %q, want none", s)
	}
	if err := tenancy.InTenantTx(t.Context(), app, tenantA, func(tx pgx.Tx) error {
		var n int
		if err := tx.QueryRow(t.Context(), "SELECT count(*) FROM "+table+" WHERE n = 7").Scan(&n); err != nil {
			return err
		}
		if n != 0 {
			t.Errorf("the insert before the failed commit batch was kept (%d rows)", n)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

// TestSavepointKeepsTheTenant: a savepoint of an InTenantTx transaction knows
// the tenant and shares the memo, but refuses ExecAtCommit (its rollback
// could not take the statement back).
func TestSavepointKeepsTheTenant(t *testing.T) {
	pool, _, _ := newPool(t, env(t, "NAVA_TEST_PG_URL"), 1)
	tenantA := uuid.New()
	type key struct{}
	err := tenancy.InTenantTx(t.Context(), pool, tenantA, func(tx pgx.Tx) error {
		tenancy.Remember(tx, key{}, "v")
		sp, err := tx.Begin(t.Context())
		if err != nil {
			return err
		}
		defer func() { _ = sp.Rollback(t.Context()) }()
		if got, err := tenancy.TenantOf(t.Context(), sp); err != nil || got != tenantA {
			t.Errorf("TenantOf(savepoint) = %s, %v; want %s", got, err, tenantA)
		}
		if v, ok := tenancy.Recall(sp, key{}); !ok || v != "v" {
			t.Errorf("Recall(savepoint) = %v, %v; want the parent's memo", v, ok)
		}
		if tenancy.ExecAtCommit(sp, "SELECT 1") {
			t.Error("ExecAtCommit accepted a savepoint")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// TestMemoIsPerTransaction: what one transaction remembers is not in the
// next one's memo, on the same connection, for the same or another tenant.
func TestMemoIsPerTransaction(t *testing.T) {
	pool, _, _ := newPool(t, env(t, "NAVA_TEST_PG_URL"), 1)
	type key struct{}
	tenantA, tenantB := uuid.New(), uuid.New()
	for i, tenant := range []uuid.UUID{tenantA, tenantA, tenantB} {
		err := tenancy.InTenantTx(t.Context(), pool, tenant, func(tx pgx.Tx) error {
			if v, ok := tenancy.Recall(tx, key{}); ok {
				t.Errorf("transaction %d recalls %v from an earlier transaction", i, v)
			}
			tenancy.Remember(tx, key{}, tenant)
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

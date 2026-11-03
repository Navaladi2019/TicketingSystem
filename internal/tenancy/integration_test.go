//go:build integration

// T07 (TICK-61) against a real PostgreSQL and the PgBouncer of T06.
// make test-integration (scripts/ci/test-integration.sh) starts both and sets:
//
//	NAVA_TEST_PG_URL         superuser on the PostgreSQL
//	NAVA_TEST_PGBOUNCER_URL  PgBouncer in transaction mode, database nava_it_pool
//
// The role and database nava_it_pool are (re)created here, as the T06 load
// test does; the script runs the packages one at a time (-p 1).

package tenancy_test

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/db"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

const (
	bouncerRole     = "nava_it_pool"
	bouncerPassword = "test-only-pool" // scripts/ci/pgbouncer/userlist.txt
	// tenantSQL reads the setting InTenantTx writes, with the backend that
	// served it. NULL (never set on this backend) and '' (set by an earlier
	// transaction, which ended) both mean no tenant.
	tenantSQL = "SELECT pg_backend_pid(), coalesce(current_setting('app.tenant_id', true), '')"
)

func env(t *testing.T, name string) string {
	t.Helper()
	v := os.Getenv(name)
	if v == "" {
		t.Fatalf("%s is not set; run make test-integration", name)
	}
	return v
}

// newPool is the platform pool (db.NewPool) for url, with at most maxConns
// client connections and spans exported to an in-memory exporter.
func newPool(t *testing.T, url string, maxConns int32) (*pgxpool.Pool, *telemetry.Telemetry, *tracetest.InMemoryExporter) {
	t.Helper()
	exp := tracetest.NewInMemoryExporter()
	tel, err := telemetry.New(t.Context(), config.Telemetry{},
		telemetry.WithSpanExporter(exp), telemetry.WithSampler(sdktrace.AlwaysSample()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tel.Shutdown(context.Background()) })
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	cfg.MaxConns = maxConns
	pool, err := db.NewPool(t.Context(), cfg, tel)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool, tel, exp
}

// bouncerURL (re)creates the role and database PgBouncer serves and returns
// the PgBouncer URL.
func bouncerURL(t *testing.T) string {
	t.Helper()
	ctx := t.Context()
	conn, err := pgx.Connect(ctx, env(t, "NAVA_TEST_PG_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(context.Background())
	for _, sql := range []string{
		"DROP DATABASE IF EXISTS " + bouncerRole + " WITH (FORCE)",
		"DROP ROLE IF EXISTS " + bouncerRole,
		"CREATE ROLE " + bouncerRole + " LOGIN PASSWORD '" + bouncerPassword + "'",
		"CREATE DATABASE " + bouncerRole + " OWNER " + bouncerRole,
	} {
		if _, err := conn.Exec(ctx, sql); err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
	}
	return env(t, "NAVA_TEST_PGBOUNCER_URL")
}

// readTenant runs tenantSQL in a plain transaction, outside InTenantTx.
func readTenant(t *testing.T, pool *pgxpool.Pool) (pid uint32, tenant string) {
	t.Helper()
	err := pgx.BeginFunc(t.Context(), pool, func(tx pgx.Tx) error {
		return tx.QueryRow(t.Context(), tenantSQL).Scan(&pid, &tenant)
	})
	if err != nil {
		t.Fatal(err)
	}
	return pid, tenant
}

// inTenant runs InTenantTx and returns the backend and the tenant it saw.
func inTenant(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID) (pid uint32, tenant string) {
	t.Helper()
	err := tenancy.InTenantTx(t.Context(), pool, tenantID, func(tx pgx.Tx) error {
		return tx.QueryRow(t.Context(), tenantSQL).Scan(&pid, &tenant)
	})
	if err != nil {
		t.Fatal(err)
	}
	return pid, tenant
}

// TestTenantIsTransactionLocal: the tenant is set inside InTenantTx and gone
// in the next transaction on the same connection (one connection, direct).
func TestTenantIsTransactionLocal(t *testing.T) {
	pool, _, _ := newPool(t, env(t, "NAVA_TEST_PG_URL"), 1)
	tenantA := uuid.New()

	pid1, got := inTenant(t, pool, tenantA)
	if got != tenantA.String() {
		t.Fatalf("inside InTenantTx app.tenant_id = %q, want %q", got, tenantA)
	}
	pid2, got := readTenant(t, pool)
	if pid2 != pid1 {
		t.Fatalf("backend %d then %d: not the same connection", pid1, pid2)
	}
	if got != "" {
		t.Errorf("next transaction sees app.tenant_id = %q, want none", got)
	}
}

// TestTenantNotVisibleThroughPgBouncer: through PgBouncer in transaction mode
// a server connection serves whichever client comes next. Tenant A set in
// transaction 1 is not visible in transaction 2 on the same server
// connection, whether the next client is the same one or another.
func TestTenantNotVisibleThroughPgBouncer(t *testing.T) {
	url := bouncerURL(t)
	clientA, _, _ := newPool(t, url, 1)
	clientB, _, _ := newPool(t, url, 1)
	tenantA := uuid.New()

	for name, next := range map[string]*pgxpool.Pool{"same client": clientA, "other client": clientB} {
		t.Run(name, func(t *testing.T) {
			// PgBouncer hands out the most recently used server connection
			// (server_round_robin = 0), so with one client at a time the next
			// transaction lands on the same backend. Retry a few times in case
			// it does not, and fail if it never does.
			for range 20 {
				pid1, got := inTenant(t, clientA, tenantA)
				if got != tenantA.String() {
					t.Fatalf("inside InTenantTx app.tenant_id = %q, want %q", got, tenantA)
				}
				pid2, got := readTenant(t, next)
				if pid2 != pid1 {
					continue
				}
				if got != "" {
					t.Fatalf("transaction 2 on backend %d sees app.tenant_id = %q, want none", pid2, got)
				}
				return
			}
			t.Fatal("transaction 2 never reached the backend transaction 1 used")
		})
	}
}

// TestPanicRollsBackAndReraises: a panic in fn rolls back what fn wrote and
// reaches the caller unchanged; the connection is usable afterwards.
func TestPanicRollsBackAndReraises(t *testing.T) {
	pool, _, _ := newPool(t, env(t, "NAVA_TEST_PG_URL"), 1)
	table := createTable(t, pool)
	type marker struct{ s string }
	want := marker{"boom"}

	func() {
		defer func() {
			if got := recover(); got != want {
				t.Errorf("recovered %v, want the panic value %v", got, want)
			}
		}()
		_ = tenancy.InTenantTx(t.Context(), pool, uuid.New(), func(tx pgx.Tx) error {
			if _, err := tx.Exec(t.Context(), "INSERT INTO "+table+" VALUES (1)"); err != nil {
				t.Fatal(err)
			}
			panic(want)
		})
		t.Error("InTenantTx returned instead of panicking")
	}()

	if n := count(t, pool, table); n != 0 {
		t.Errorf("%d rows after the panic, want 0 (rolled back)", n)
	}
	if _, got := readTenant(t, pool); got != "" {
		t.Errorf("after the panic app.tenant_id = %q, want none", got)
	}
}

// TestErrorRollsBack: an error from fn rolls back and is returned as is.
func TestErrorRollsBack(t *testing.T) {
	pool, _, _ := newPool(t, env(t, "NAVA_TEST_PG_URL"), 1)
	table := createTable(t, pool)
	errFn := errors.New("fn failed")

	err := tenancy.InTenantTx(t.Context(), pool, uuid.New(), func(tx pgx.Tx) error {
		if _, err := tx.Exec(t.Context(), "INSERT INTO "+table+" VALUES (1)"); err != nil {
			return err
		}
		return errFn
	})
	if err != errFn { //nolint:errorlint // the very error, not wrapped
		t.Errorf("err = %v, want fn's error unwrapped", err)
	}
	if n := count(t, pool, table); n != 0 {
		t.Errorf("%d rows after the error, want 0 (rolled back)", n)
	}

	// And the commit path: the row is there.
	err = tenancy.InTenantTx(t.Context(), pool, uuid.New(), func(tx pgx.Tx) error {
		_, err := tx.Exec(t.Context(), "INSERT INTO "+table+" VALUES (2)")
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if n := count(t, pool, table); n != 1 {
		t.Errorf("%d rows after a commit, want 1", n)
	}
}

// TestFailedRollbackIsJoined: when the rollback after fn's error fails too
// (here the context is cancelled), both errors reach the caller, and the pool
// still works with no tenant on the next transaction.
func TestFailedRollbackIsJoined(t *testing.T) {
	pool, _, _ := newPool(t, env(t, "NAVA_TEST_PG_URL"), 1)
	errFn := errors.New("fn failed")
	ctx, cancel := context.WithCancel(t.Context())

	err := tenancy.InTenantTx(ctx, pool, uuid.New(), func(pgx.Tx) error {
		cancel()
		return errFn
	})
	if !errors.Is(err, errFn) {
		t.Errorf("err = %v, want fn's error", err)
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want the rollback's context.Canceled joined", err)
	}
	if _, got := readTenant(t, pool); got != "" {
		t.Errorf("after the failed rollback app.tenant_id = %q, want none", got)
	}
}

// TestTracedTxExportsNoValues: a traced InTenantTx, including a query that
// fails with 22P02 quoting its input, exports database spans with no SQL
// text, no parameter values and no tenant id. TICK-94 item 3 allows the
// tenant id only as nava.tenant_id, which InTenantTx does not set: that
// comes from the validated token, in the request context.
func TestTracedTxExportsNoValues(t *testing.T) {
	pool, tel, exp := newPool(t, env(t, "NAVA_TEST_PG_URL"), 1)
	tenantID := uuid.New()

	// otelpgx traces only under a recording span, as a request's server span.
	ctx, root := tel.TracerProvider.Tracer("test").Start(t.Context(), "request")
	err := tenancy.InTenantTx(ctx, pool, tenantID, func(tx pgx.Tx) error {
		var s string
		if err := tx.QueryRow(ctx, "SELECT $1::text", "bob@tenant-b.test").Scan(&s); err != nil {
			return err
		}
		var id uuid.UUID
		return tx.QueryRow(ctx, "SELECT $1::uuid", "alice@tenant-a.test").Scan(&id)
	})
	root.End()
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "22P02" {
		t.Fatalf("err = %v, want a 22P02", err)
	}

	var dbSpans int
	for _, s := range exp.GetSpans() {
		if s.InstrumentationScope.Name == "github.com/exaring/otelpgx" {
			dbSpans++
		}
		for _, l := range leaks(s, "alice@", "bob@", "$1", "app.tenant_id", "set_config", tenantID.String()) {
			t.Errorf("span %q leaks %s", s.Name, l)
		}
	}
	// set_config, the two queries, the acquire and the commit or rollback.
	if dbSpans < 3 {
		t.Errorf("exported %d otelpgx spans, want the transaction's queries traced", dbSpans)
	}
}

// TestUUIDRoundTrip: a uuid.UUID goes into pgx as a parameter and comes back
// from a uuid column unchanged (driver.Valuer and sql.Scanner, TICK-94 item 10).
func TestUUIDRoundTrip(t *testing.T) {
	pool, _, _ := newPool(t, env(t, "NAVA_TEST_PG_URL"), 1)
	for _, want := range []uuid.UUID{uuid.New(), uuid.NewSHA1(uuid.NameSpaceURL, []byte("nava")), uuid.Nil} {
		var got uuid.UUID
		var typ string
		if err := pool.QueryRow(t.Context(), "SELECT $1::uuid, pg_typeof($1::uuid)::text", want).Scan(&got, &typ); err != nil {
			t.Fatal(err)
		}
		if got != want || typ != "uuid" {
			t.Errorf("round trip of %s = %s (%s)", want, got, typ)
		}
	}
}

func createTable(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	table := "tenancy_it_" + strings.ReplaceAll(uuid.NewString()[:8], "-", "")
	if _, err := pool.Exec(t.Context(), "CREATE TABLE "+table+" (n int)"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), "DROP TABLE "+table) })
	return table
}

func count(t *testing.T, pool *pgxpool.Pool, table string) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(t.Context(), "SELECT count(*) FROM "+table).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// leaks lists every place in an exported span where a marker survives: the
// name, attributes, events, links and the status description.
func leaks(s tracetest.SpanStub, markers ...string) []string {
	var out []string
	check := func(where, v string) {
		for _, m := range markers {
			if strings.Contains(v, m) {
				out = append(out, fmt.Sprintf("%s = %q", where, v))
			}
		}
	}
	check("name", s.Name)
	for _, kv := range s.Attributes {
		if kv.Key != telemetry.TenantIDKey {
			check("attr "+string(kv.Key), kv.Value.String())
		}
	}
	for _, ev := range s.Events {
		check("event name", ev.Name)
		for _, kv := range ev.Attributes {
			check("event "+ev.Name+" "+string(kv.Key), kv.Value.String())
		}
	}
	for _, l := range s.Links {
		for _, kv := range l.Attributes {
			check("link "+string(kv.Key), kv.Value.String())
		}
	}
	check("status", s.Status.Description)
	return out
}

// TenantOf returns the tenant InTenantTx set, and ErrNoTenant in a plain
// transaction, so the record pipeline never writes without one.
func TestTenantOf(t *testing.T) {
	pool, _, _ := newPool(t, env(t, "NAVA_TEST_PG_URL"), 1)
	ctx := context.Background()
	tenantA := uuid.New()
	if err := tenancy.InTenantTx(ctx, pool, tenantA, func(tx pgx.Tx) error {
		got, err := tenancy.TenantOf(ctx, tx)
		if err != nil || got != tenantA {
			t.Errorf("TenantOf = %s, %v; want %s", got, err, tenantA)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tenancy.TenantOf(ctx, tx); !errors.Is(err, tenancy.ErrNoTenant) {
		t.Fatalf("no tenant: %v, want ErrNoTenant", err)
	}
}

// TestCommitFailureIsReturned: when fn swallows a failed statement and
// returns nil, PostgreSQL turns the COMMIT into a rollback; InTenantTx
// reports it, wrapped, and nothing fn wrote is kept.
func TestCommitFailureIsReturned(t *testing.T) {
	pool, _, _ := newPool(t, env(t, "NAVA_TEST_PG_URL"), 1)
	table := createTable(t, pool)

	err := tenancy.InTenantTx(t.Context(), pool, uuid.New(), func(tx pgx.Tx) error {
		if _, err := tx.Exec(t.Context(), "INSERT INTO "+table+" VALUES (1)"); err != nil {
			return err
		}
		_, _ = tx.Exec(t.Context(), "SELECT 1/0") // aborts the transaction; the error is dropped
		return nil
	})
	if !errors.Is(err, pgx.ErrTxCommitRollback) {
		t.Fatalf("err = %v, want pgx.ErrTxCommitRollback", err)
	}
	if !strings.HasPrefix(err.Error(), "tenancy: commit: ") {
		t.Errorf("message %q, want it wrapped as the commit", err)
	}
	if n := count(t, pool, table); n != 0 {
		t.Errorf("%d rows after the failed commit, want 0", n)
	}
	if _, got := readTenant(t, pool); got != "" {
		t.Errorf("after the failed commit app.tenant_id = %q, want none", got)
	}
}

// TestTenantCannotReachAnotherTenantsRows: under a row-level security policy
// on app.tenant_id, as the platform tables use, a non-superuser in tenant
// A's InTenantTx reads, updates and deletes only A's rows and cannot insert
// one for tenant B, even when the statement names B explicitly; with no
// tenant it sees nothing.
func TestTenantCannotReachAnotherTenantsRows(t *testing.T) {
	const (
		role     = "nava_it_tenancy"
		password = "test-only-tenancy"
	)
	super, _, _ := newPool(t, env(t, "NAVA_TEST_PG_URL"), 2)
	ctx := t.Context()
	table := "tenancy_rls_" + strings.ReplaceAll(uuid.NewString()[:8], "-", "")
	t.Cleanup(func() {
		_, _ = super.Exec(context.Background(), "DROP TABLE IF EXISTS "+table)
		_, _ = super.Exec(context.Background(), "DROP ROLE IF EXISTS "+role)
	})
	tenantA, tenantB := uuid.New(), uuid.New()
	policy := "tenant_id = nullif(current_setting('app.tenant_id', true), '')::uuid"
	for _, sql := range []string{
		"DROP ROLE IF EXISTS " + role,
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
	app, _, _ := newPool(t, u.String(), 1)

	visible := func(tx pgx.Tx) []int {
		rows, err := tx.Query(ctx, "SELECT n FROM "+table+" ORDER BY n")
		if err != nil {
			t.Fatal(err)
		}
		ns, err := pgx.CollectRows(rows, pgx.RowTo[int])
		if err != nil {
			t.Fatal(err)
		}
		return ns
	}

	err = tenancy.InTenantTx(ctx, app, tenantA, func(tx pgx.Tx) error {
		if got := visible(tx); len(got) != 1 || got[0] != 1 {
			t.Errorf("tenant A sees %v, want only its own row [1]", got)
		}
		// Naming B's tenant id does not reach B's row.
		tag, err := tx.Exec(ctx, "UPDATE "+table+" SET n = 20 WHERE tenant_id = $1", tenantB)
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 0 {
			t.Errorf("tenant A updated %d of tenant B's rows", tag.RowsAffected())
		}
		tag, err = tx.Exec(ctx, "DELETE FROM "+table+" WHERE tenant_id = $1", tenantB)
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 0 {
			t.Errorf("tenant A deleted %d of tenant B's rows", tag.RowsAffected())
		}
		tag, err = tx.Exec(ctx, "UPDATE "+table+" SET n = 10")
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 1 {
			t.Errorf("tenant A's unfiltered update touched %d rows, want 1 (its own)", tag.RowsAffected())
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	// Writing a row for tenant B from tenant A's transaction is refused.
	err = tenancy.InTenantTx(ctx, app, tenantA, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, "INSERT INTO "+table+" VALUES ($1, 3)", tenantB)
		return err
	})
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "42501" {
		t.Fatalf("insert for tenant B from tenant A: %v, want 42501 (row-level security)", err)
	}

	// Tenant B still has exactly its row, unchanged, and sees only that.
	err = tenancy.InTenantTx(ctx, app, tenantB, func(tx pgx.Tx) error {
		if got := visible(tx); len(got) != 1 || got[0] != 2 {
			t.Errorf("tenant B sees %v, want its untouched row [2]", got)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	// Outside InTenantTx there is no tenant and no row.
	err = pgx.BeginFunc(ctx, app, func(tx pgx.Tx) error {
		if got := visible(tx); len(got) != 0 {
			t.Errorf("a transaction with no tenant sees %v, want nothing", got)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

//go:build integration

// TICK-550: a Parse-time error inside a transaction, then the same SQL in the
// next transaction on the same pgx connection, through the PgBouncer of
// TestPgBouncerPreparedStatements (transaction mode, max_prepared_statements
// > 0). Without db.ParseFailureCloser, PgBouncer 1.24.1 closes the client on
// the second attempt with 08P01 "prepared statement name is already in use".
// With it, every attempt reports the real SQLSTATE and the connection
// survives.

package pooltest

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/db"
)

// parseErrorCases fail at Parse, before any row is read.
func parseErrorCases() []struct{ name, sql, code string } {
	return []struct{ name, sql, code string }{
		{"undefined function", "SELECT no_such_fn_550($1::int)", "42883"},
		{"undefined table", "SELECT x FROM no_such_table_550 WHERE x = $1::int", "42P01"},
	}
}

func sqlState(err error) string {
	var pe *pgconn.PgError
	if errors.As(err, &pe) {
		return pe.Code
	}
	return ""
}

// retryParseError runs sql in three transactions in a row, each after a
// statement that succeeds (tenancy.InTenantTx sets the tenant first). It
// returns each attempt's error; a nil entry means Begin failed (the
// connection was already closed).
func retryParseError(t *testing.T, c *pgx.Conn, sql string) []error {
	t.Helper()
	ctx := t.Context()
	var errs []error
	for i := 1; i <= 3; i++ {
		tx, err := c.Begin(ctx)
		if err != nil {
			t.Logf("attempt %d: begin: %v", i, err)
			errs = append(errs, nil)
			continue
		}
		if _, err := tx.Exec(ctx, "SELECT $1::text", "tick-550"); err != nil {
			t.Fatalf("attempt %d: first statement: %v", i, err)
		}
		_, err = tx.Exec(ctx, sql, i)
		t.Logf("attempt %d: %v", i, err)
		errs = append(errs, err)
		_ = tx.Rollback(ctx)
	}
	return errs
}

func connect(t *testing.T, url string, tracer pgx.QueryTracer) *pgx.Conn {
	t.Helper()
	cfg, err := pgx.ParseConfig(env(t, url))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DefaultQueryExecMode != pgx.QueryExecModeCacheStatement {
		t.Fatalf("pgx default exec mode is %v, want QueryExecModeCacheStatement (the mode under test)", cfg.DefaultQueryExecMode)
	}
	cfg.Tracer = tracer
	c, err := pgx.ConnectConfig(t.Context(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c.Close(t.Context()) })
	return c
}

// TestParseErrorInTxWithoutCloser is the negative case: it proves the pinned
// PgBouncer still has the fault, so the main case is not passing vacuously.
// If it fails because every attempt kept its SQLSTATE, PgBouncer fixed the
// fault: record "verified fixed at <version>" in docs/pins.md and remove
// db.ParseFailureCloser.
func TestParseErrorInTxWithoutCloser(t *testing.T) {
	setupDatabase(t)
	c := connect(t, "NAVA_TEST_PGBOUNCER_URL", nil)
	errs := retryParseError(t, c, parseErrorCases()[0].sql)
	if got := sqlState(errs[1]); got != "08P01" {
		t.Fatalf("attempt 2: SQLSTATE %q (%v), want 08P01 from PgBouncer: the fault TICK-550 works around is gone", got, errs[1])
	}
	if !c.IsClosed() {
		t.Error("PgBouncer reported 08P01 but the connection is still open")
	}
}

// TestParseErrorInTxWithCloser is the main case: with the tracer NewPool
// installs, every attempt reports the server's SQLSTATE and the connection
// still serves a query afterwards, through PgBouncer and, where the Close
// reaches PostgreSQL in an aborted transaction, directly.
func TestParseErrorInTxWithCloser(t *testing.T) {
	setupDatabase(t)
	for _, url := range []string{"NAVA_TEST_PGBOUNCER_URL", "NAVA_TEST_PG_URL"} {
		for _, tc := range parseErrorCases() {
			t.Run(url+"/"+tc.name, func(t *testing.T) {
				c := connect(t, url, db.ParseFailureCloser{})
				for i, err := range retryParseError(t, c, tc.sql) {
					if got := sqlState(err); got != tc.code {
						t.Errorf("attempt %d: SQLSTATE %q (%v), want %s", i+1, got, err, tc.code)
					}
				}
				var n int
				if err := c.QueryRow(t.Context(), "SELECT $1::int + 1", 1).Scan(&n); err != nil || n != 2 {
					t.Errorf("query after the failures: n=%d err=%v, want 2 and no error", n, err)
				}
			})
		}
	}
}

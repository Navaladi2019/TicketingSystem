//go:build integration

// T06 (TICK-60): pgx's default exec mode caches prepared statements per
// client connection. Through PgBouncer in transaction mode that is only safe
// when PgBouncer tracks prepared statements (max_prepared_statements > 0,
// PgBouncer >= 1.21), because consecutive transactions of one client land on
// different server connections. This test runs 50 clients x 200 statements
// through a pool of 5 server connections and expects zero errors; the same
// load through a second PgBouncer with max_prepared_statements = 0 must fail.
//
// make test-integration (scripts/ci/test-integration.sh) starts both
// PgBouncers (scripts/ci/pgbouncer/*.ini, image from docs/pins.md) in front
// of its PostgreSQL and sets:
//
//	NAVA_TEST_PG_URL                superuser, to create the test role and database
//	NAVA_TEST_PGBOUNCER_URL         PgBouncer, max_prepared_statements = 100
//	NAVA_TEST_PGBOUNCER_NOPREP_URL  PgBouncer, max_prepared_statements = 0

package pooltest

import (
	"context"
	"errors"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	clients          = 50
	perClient        = 200 // clients * perClient = 10,000 statements
	poolSize         = 5   // default_pool_size in scripts/ci/pgbouncer/*.ini
	testRole         = "nava_it_pool"
	testRolePassword = "test-only-pool" // scripts/ci/pgbouncer/userlist.txt
)

// statements are distinct SQL texts, so each client prepares several named
// statements (pgx names them by a hash of the text, the same on every
// client) and the clients interleave Parse and Bind on shared servers. Each
// returns the backend PID that served it.
func statements() []string {
	return []string{
		"SELECT pg_backend_pid(), $1::int + 1",
		"SELECT pg_backend_pid(), length($1::int::text)",
		"SELECT pg_backend_pid(), (SELECT count(*) FROM generate_series(1, $1::int % 7 + 1))::int",
		"SELECT pg_backend_pid(), $1::int * 2",
		"SELECT pg_backend_pid(), abs($1::int - 100)",
	}
}

func env(t *testing.T, name string) string {
	t.Helper()
	v := os.Getenv(name)
	if v == "" {
		t.Fatalf("%s is not set; run make test-integration", name)
	}
	return v
}

// setupDatabase (re)creates the role and database both PgBouncers serve.
func setupDatabase(t *testing.T) {
	t.Helper()
	ctx := t.Context()
	conn, err := pgx.Connect(ctx, env(t, "NAVA_TEST_PG_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(context.Background())
	for _, sql := range []string{
		"DROP DATABASE IF EXISTS " + testRole + " WITH (FORCE)",
		"DROP ROLE IF EXISTS " + testRole,
		"CREATE ROLE " + testRole + " LOGIN PASSWORD '" + testRolePassword + "'",
		"CREATE DATABASE " + testRole + " OWNER " + testRole,
	} {
		if _, err := conn.Exec(ctx, sql); err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
	}
}

// admin is a connection to PgBouncer's console (database pgbouncer), which
// speaks the simple query protocol only.
type admin struct{ conn *pgx.Conn }

func connectAdmin(t *testing.T, url string) *admin {
	t.Helper()
	cfg, err := pgx.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	cfg.Database = "pgbouncer"
	cfg.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
	conn, err := pgx.ConnectConfig(t.Context(), cfg)
	if err != nil {
		t.Fatalf("connect to the PgBouncer console: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close(context.Background()) })
	return &admin{conn}
}

// show runs a SHOW command and returns its rows as column -> text value.
func (a *admin) show(t *testing.T, cmd string) []map[string]string {
	t.Helper()
	rows, err := a.conn.Query(t.Context(), cmd)
	if err != nil {
		t.Fatalf("%s: %v", cmd, err)
	}
	defer rows.Close()
	var out []map[string]string
	for rows.Next() {
		row := map[string]string{}
		for i, f := range rows.FieldDescriptions() {
			row[f.Name] = string(rows.RawValues()[i])
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("%s: %v", cmd, err)
	}
	return out
}

// config returns one SHOW CONFIG value.
func (a *admin) config(t *testing.T, key string) string {
	t.Helper()
	for _, r := range a.show(t, "SHOW CONFIG") {
		if r["key"] == key {
			return r["value"]
		}
	}
	t.Fatalf("SHOW CONFIG has no %s", key)
	return ""
}

// row returns the SHOW POOLS / SHOW STATS row of the test database, with the
// named columns as integers.
func (a *admin) row(t *testing.T, cmd string, cols ...string) map[string]int64 {
	t.Helper()
	r := a.rowOrNil(t, cmd, cols...)
	if r == nil {
		t.Fatalf("%s has no row for database %s", cmd, testRole)
	}
	return r
}

// rowOrNil is row, or nil before PgBouncer has a pool for the database.
func (a *admin) rowOrNil(t *testing.T, cmd string, cols ...string) map[string]int64 {
	t.Helper()
	for _, r := range a.show(t, cmd) {
		if r["database"] != testRole {
			continue
		}
		out := map[string]int64{}
		for _, c := range cols {
			v, ok := r[c]
			if !ok {
				t.Fatalf("%s has no column %s", cmd, c)
			}
			n, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				t.Fatalf("%s %s = %q: %v", cmd, c, v, err)
			}
			out[c] = n
		}
		return out
	}
	return nil
}

// Columns of SHOW POOLS (clients, servers) and SHOW STATS the test reads.
func poolClientCols() []string { return []string{"cl_active", "cl_waiting"} }

func poolServerCols() []string {
	return []string{"sv_active", "sv_idle", "sv_used", "sv_tested", "sv_login"}
}

func statsCols() []string { return []string{"total_xact_count", "total_server_assignment_count"} }

func sum(m map[string]int64, cols []string) int64 {
	var n int64
	for _, c := range cols {
		n += m[c]
	}
	return n
}

// result is what one load run observed.
type result struct {
	statements   int
	preparedErrs int            // SQLSTATE 26000 / 42P05 or "prepared statement" errors
	otherErrs    int            // anything else
	firstErr     error          // first error of either kind, for the log
	clientsByPID map[uint32]int // backend PID -> how many clients it served
}

// checkNoPreparedErrors is the outcome assertion. The main case expects nil;
// the negative case expects it to fail, which proves the check detects the
// fault rather than passing vacuously.
func checkNoPreparedErrors(r result) error {
	if r.preparedErrs > 0 || r.otherErrs > 0 {
		return fmt.Errorf("%d prepared-statement errors and %d other errors in %d statements; first: %w",
			r.preparedErrs, r.otherErrs, r.statements, r.firstErr)
	}
	return nil
}

func isPreparedErr(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && (pgErr.Code == "26000" || pgErr.Code == "42P05") {
		return true
	}
	return strings.Contains(err.Error(), "prepared statement")
}

// load connects every client first, calls connected (to sample SHOW POOLS
// with all clients logged in), then runs clients x perClient statements in
// pgx's default exec mode.
func load(t *testing.T, url string, connected func()) result {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Minute)
	defer cancel()
	cfg, err := pgx.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DefaultQueryExecMode != pgx.QueryExecModeCacheStatement {
		t.Fatalf("pgx default exec mode is %v, want QueryExecModeCacheStatement (the mode under test)", cfg.DefaultQueryExecMode)
	}

	conns := make([]*pgx.Conn, clients)
	for i := range conns {
		if conns[i], err = pgx.ConnectConfig(ctx, cfg); err != nil {
			t.Fatalf("client %d: connect through PgBouncer: %v", i, err)
		}
	}
	defer func() {
		for _, c := range conns {
			_ = c.Close(context.Background())
		}
	}()
	connected()

	var (
		mu  sync.Mutex
		res = result{clientsByPID: map[uint32]int{}}
		wg  sync.WaitGroup
	)
	stmts := statements()
	start := make(chan struct{})
	for i := range conns {
		wg.Go(func() {
			<-start
			pids := map[uint32]bool{}
			var prepared, other int
			var first error
			for n := range perClient {
				// Each client walks the statements from its own offset, so
				// at any moment the clients prepare and bind different names.
				sql := stmts[(i+n)%len(stmts)]
				var pid uint32
				var v int
				err := conns[i].QueryRow(ctx, sql, n).Scan(&pid, &v)
				switch {
				case err == nil:
					pids[pid] = true
					continue
				case isPreparedErr(err):
					prepared++
				default:
					other++
				}
				if first == nil {
					first = fmt.Errorf("client %d statement %d: %w", i, n, err)
				}
				// A broken client reconnects, so the run keeps its load.
				if conns[i].IsClosed() {
					if c, err := pgx.ConnectConfig(ctx, cfg); err == nil {
						conns[i] = c
					}
				}
			}
			mu.Lock()
			defer mu.Unlock()
			res.statements += perClient
			res.preparedErrs += prepared
			res.otherErrs += other
			if res.firstErr == nil {
				res.firstErr = first
			}
			for pid := range pids {
				res.clientsByPID[pid]++
			}
		})
	}
	close(start)
	wg.Wait()
	return res
}

// TestPgBouncerPreparedStatements is the main case: zero errors, with proof
// that the 50 clients really shared the 5 server connections.
func TestPgBouncerPreparedStatements(t *testing.T) {
	setupDatabase(t)
	url := env(t, "NAVA_TEST_PGBOUNCER_URL")
	a := connectAdmin(t, url)
	if got := a.config(t, "pool_mode"); got != "transaction" {
		t.Fatalf("pool_mode = %s, want transaction", got)
	}
	if got := a.config(t, "max_prepared_statements"); got == "0" {
		t.Fatal("max_prepared_statements = 0 on the PgBouncer under test")
	}
	if got := a.config(t, "default_pool_size"); got != strconv.Itoa(poolSize) {
		t.Fatalf("default_pool_size = %s, want %d", got, poolSize)
	}
	t.Logf("SHOW VERSION: %s", a.show(t, "SHOW VERSION")[0]["version"])

	// The baseline is taken once every client is logged in; a missing row
	// (no pool yet) counts as zero.
	var before, peakClients map[string]int64
	res := load(t, url, func() {
		peakClients = a.row(t, "SHOW POOLS", poolClientCols()...)
		before = a.rowOrNil(t, "SHOW STATS", statsCols()...)
	})
	if err := checkNoPreparedErrors(res); err != nil {
		t.Fatalf("through PgBouncer with max_prepared_statements > 0: %v", err)
	}
	t.Logf("%d statements from %d clients, 0 errors", res.statements, clients)

	// Reuse, from PgBouncer's side. SHOW STATS counters are refreshed every
	// stats_period (60s by default), so poll until they include the run.
	servers := a.row(t, "SHOW POOLS", poolServerCols()...)
	t.Logf("SHOW POOLS %s: clients logged in %v, servers after the run %v", testRole, peakClients, servers)
	if got := sum(peakClients, poolClientCols()); got < clients {
		t.Errorf("SHOW POOLS: %d clients logged in during the run, want %d", got, clients)
	}
	if got := sum(servers, poolServerCols()); got < 1 || got > poolSize {
		t.Errorf("SHOW POOLS: %d server connections, want 1..%d", got, poolSize)
	}
	want := int64(res.statements)
	deadline := time.Now().Add(90 * time.Second)
	var after map[string]int64
	for {
		after = a.row(t, "SHOW STATS", statsCols()...)
		if after["total_xact_count"]-before["total_xact_count"] >= want || time.Now().After(deadline) {
			break
		}
		time.Sleep(time.Second)
	}
	xacts := after["total_xact_count"] - before["total_xact_count"]
	assigns := after["total_server_assignment_count"] - before["total_server_assignment_count"]
	t.Logf("SHOW STATS %s: +%d transactions, +%d server assignments", testRole, xacts, assigns)
	if xacts < want {
		t.Errorf("SHOW STATS: +%d transactions, want >= %d (one per statement)", xacts, want)
	}
	if assigns < want {
		t.Errorf("SHOW STATS: +%d server assignments, want >= %d: clients kept their server, so nothing was shared", assigns, want)
	}

	// Reuse, from PostgreSQL's side: each backend PID served many clients.
	pids := make([]uint32, 0, len(res.clientsByPID))
	shared := 0
	for pid, n := range res.clientsByPID {
		pids = append(pids, pid)
		if n > 1 {
			shared++
		}
	}
	slices.Sort(pids)
	for _, pid := range pids {
		t.Logf("backend pid %d served %d of %d clients", pid, res.clientsByPID[pid], clients)
	}
	if len(pids) > poolSize {
		t.Errorf("%d distinct backends, want <= default_pool_size %d", len(pids), poolSize)
	}
	if shared == 0 {
		t.Error("no backend served more than one client: the test proves nothing")
	}
}

// TestPgBouncerWithoutPreparedStatementTracking is the negative case: the
// same load and the same check through a PgBouncer with
// max_prepared_statements = 0 must report prepared-statement errors.
func TestPgBouncerWithoutPreparedStatementTracking(t *testing.T) {
	setupDatabase(t)
	url := env(t, "NAVA_TEST_PGBOUNCER_NOPREP_URL")
	a := connectAdmin(t, url)
	if got := a.config(t, "max_prepared_statements"); got != "0" {
		t.Fatalf("max_prepared_statements = %s on the negative-case PgBouncer, want 0", got)
	}
	if got := a.config(t, "default_pool_size"); got != strconv.Itoa(poolSize) {
		t.Fatalf("default_pool_size = %s, want %d", got, poolSize)
	}

	res := load(t, url, func() {})
	err := checkNoPreparedErrors(res)
	if err == nil {
		t.Fatalf("%d statements through PgBouncer with max_prepared_statements = 0 raised no error: the load test cannot detect the fault", res.statements)
	}
	if res.preparedErrs == 0 {
		t.Fatalf("the negative case failed, but not with prepared-statement errors: %v", err)
	}
	t.Logf("negative case failed as expected: %v", err)
}

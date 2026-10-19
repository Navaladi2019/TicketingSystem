//go:build integration

// These tests need a disposable PostgreSQL cluster: the tests
// create and drop cluster-wide roles and the databases in db/grants.yaml.
// make test-integration (scripts/ci/test-integration.sh) starts one from the
// docs/pins.md postgres image and points NAVA_TEST_PG_URL at its superuser.
// To use your own cluster instead:
//
//	NAVA_TEST_PG_URL=postgres://pgadmin:pw@127.0.0.1:55432/postgres go test -tags integration -p 1 ./internal/db/dbtest/
//
// Everything is in this one package so no two tests touch the cluster at
// once; the tests do not run in parallel.

package dbtest

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"io/fs"
	"log/slog"
	"os"
	"slices"
	"sync"
	"testing"
	"testing/fstest"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/stdlib"

	"github.com/Navaladi2019/TicketingSystem/db"
	"github.com/Navaladi2019/TicketingSystem/internal/db/grants"
	"github.com/Navaladi2019/TicketingSystem/internal/db/migrate"
	"github.com/Navaladi2019/TicketingSystem/internal/db/roles"
)

func superConfig(t *testing.T) *pgx.ConnConfig {
	t.Helper()
	url := os.Getenv("NAVA_TEST_PG_URL")
	if url == "" {
		t.Fatal("NAVA_TEST_PG_URL is not set; see the package comment")
	}
	cfg, err := pgx.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

// roleConfig connects as a manifest role with its test password.
func roleConfig(t *testing.T, role, database string) *pgx.ConnConfig {
	t.Helper()
	cfg := superConfig(t).Copy()
	cfg.User = role
	cfg.Password = password(role)
	cfg.Database = database
	return cfg
}

func password(role string) string { return "test-pw-" + role }

func passwords(role string) (string, bool) { return password(role), true }

func manifest(t *testing.T) *grants.Manifest {
	t.Helper()
	m, err := grants.Parse(db.Grants())
	if err != nil {
		t.Fatal(err)
	}
	return m
}

// logBuffer is a JSON slog logger whose output the test can read.
type logBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (l *logBuffer) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.buf.Write(p)
}

func (l *logBuffer) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.buf.String()
}

func newLog() (*slog.Logger, *logBuffer) {
	b := &logBuffer{}
	return slog.New(slog.NewJSONHandler(b, nil)), b
}

func connect(t *testing.T, cfg *pgx.ConnConfig) *pgx.Conn {
	t.Helper()
	conn, err := pgx.ConnectConfig(context.Background(), cfg)
	if err != nil {
		t.Fatalf("connect as %s to %s: %v", cfg.User, cfg.Database, err)
	}
	t.Cleanup(func() { _ = conn.Close(context.Background()) })
	return conn
}

func exec(t *testing.T, conn *pgx.Conn, sql string, args ...any) {
	t.Helper()
	if _, err := conn.Exec(context.Background(), sql, args...); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
}

func queryString(t *testing.T, conn *pgx.Conn, sql string, args ...any) string {
	t.Helper()
	var s string
	if err := conn.QueryRow(context.Background(), sql, args...).Scan(&s); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
	return s
}

func queryBool(t *testing.T, conn *pgx.Conn, sql string, args ...any) bool {
	t.Helper()
	var b bool
	if err := conn.QueryRow(context.Background(), sql, args...).Scan(&b); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
	return b
}

// sqlState returns err's SQLSTATE, or "" when it is not a server error.
func sqlState(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code
	}
	return ""
}

// freshCluster drops every database and role in the manifest and runs
// db-roles, leaving the cluster as a first `docker compose up` would.
func freshCluster(t *testing.T) *grants.Manifest {
	t.Helper()
	ctx := context.Background()
	m := manifest(t)
	conn := connect(t, superConfig(t))
	for _, d := range m.DatabaseNames() {
		exec(t, conn, "DROP DATABASE IF EXISTS "+pgx.Identifier{d}.Sanitize()+" WITH (FORCE)")
	}
	for _, r := range append(m.RoleNames(), "nava_test_intruder") {
		exec(t, conn, "DROP ROLE IF EXISTS "+pgx.Identifier{r}.Sanitize())
	}
	log, _ := newLog()
	if _, err := roles.Run(ctx, superConfig(t), m, passwords, log); err != nil {
		t.Fatalf("db-roles: %v", err)
	}
	return m
}

// migrateAll runs the migrator, as nava_migrator, with the real migrations.
func migrateAll(t *testing.T, m *grants.Manifest) migrate.Result {
	t.Helper()
	log, _ := newLog()
	res, err := migrate.Run(context.Background(), roleConfig(t, m.Owner(), m.PlatformDatabase),
		db.Migrations(), m, log, migrate.Options{LockRetrySeconds: 1})
	if err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return res
}

// connectErr connects and closes, returning the connection error.
func connectErr(cfg *pgx.ConnConfig) (struct{}, error) {
	conn, err := pgx.ConnectConfig(context.Background(), cfg)
	if err == nil {
		_ = conn.Close(context.Background())
	}
	return struct{}{}, err
}

func sorted(s []string) []string {
	out := slices.Clone(s)
	slices.Sort(out)
	return out
}

// withExtraMigration is the real migrations plus one test file.
func withExtraMigration(t *testing.T, name, body string) fs.FS {
	t.Helper()
	out := fstest.MapFS{name: &fstest.MapFile{Data: []byte(body)}}
	entries, err := fs.ReadDir(db.Migrations(), ".")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		b, err := fs.ReadFile(db.Migrations(), e.Name())
		if err != nil {
			t.Fatal(err)
		}
		out[e.Name()] = &fstest.MapFile{Data: b}
	}
	return out
}

func countMigrations(t *testing.T, fsys fs.FS) int {
	t.Helper()
	names, err := fs.Glob(fsys, "*.sql")
	if err != nil {
		t.Fatal(err)
	}
	return len(names)
}

func openSQL(t *testing.T, cfg *pgx.ConnConfig) *sql.DB {
	t.Helper()
	d := stdlib.OpenDB(*cfg)
	t.Cleanup(func() { _ = d.Close() })
	return d
}

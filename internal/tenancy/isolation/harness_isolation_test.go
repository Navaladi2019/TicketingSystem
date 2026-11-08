//go:build integration

// The database half of the isolation suite. make isolation
// (scripts/ci/isolation.sh) starts a throwaway PostgreSQL and a PgBouncer in
// transaction mode that logs runtime roles in with the real auth_query
// (scripts/ci/pgbouncer/isolation.ini), and sets:
//
//	NAVA_TEST_PG_URL                superuser on the PostgreSQL
//	NAVA_ISOLATION_PGBOUNCER_ADDR   host:port of the PgBouncer
//	NAVA_ISOLATION_REPORT           where the report goes (default isolation-report.md)
//
// TestMain drops and rebuilds the manifest roles and databases, so never
// point it at a cluster that matters. scripts/ci/test-integration.sh leaves
// this package out; make ci-integration runs make isolation after it.

package isolation

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"os"
	"strconv"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/db"
	"github.com/Navaladi2019/TicketingSystem/internal/audit/read"
	"github.com/Navaladi2019/TicketingSystem/internal/db/audited"
	"github.com/Navaladi2019/TicketingSystem/internal/db/catalog"
	"github.com/Navaladi2019/TicketingSystem/internal/db/grants"
	"github.com/Navaladi2019/TicketingSystem/internal/db/migrate"
	"github.com/Navaladi2019/TicketingSystem/internal/db/roles"
)

// suite is the cluster TestMain built; the tests do not run in parallel.
//
//nolint:gochecknoglobals // set once by TestMain, as m.Run gives tests no other way in
var suite *env

type env struct {
	super   *pgx.ConnConfig // bootstrap superuser, on the platform database
	bouncer string          // PgBouncer host:port
	m       *grants.Manifest
	g       *catalog.GlobalTables
	tables  []catalog.TenantTable
	report  report
}

func password(role string) string { return "test-pw-" + role }

func TestMain(m *testing.M) {
	ctx := context.Background()
	e, err := setup(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "isolation setup:", err)
		os.Exit(1)
	}
	suite = e
	code := m.Run()
	path := os.Getenv("NAVA_ISOLATION_REPORT")
	if path == "" {
		path = "isolation-report.md"
	}
	if err := os.WriteFile(path, []byte(e.report.markdown(code == 0)), 0o600); err != nil {
		fmt.Fprintln(os.Stderr, "isolation report:", err)
		code = 1
	}
	fmt.Println("isolation report:", path)
	os.Exit(code)
}

// setup rebuilds the cluster as a first `docker compose up` would (db-roles,
// then the migrator with its catalogue gate) and seeds tenants A and B.
func setup(ctx context.Context) (*env, error) {
	url, addr := os.Getenv("NAVA_TEST_PG_URL"), os.Getenv("NAVA_ISOLATION_PGBOUNCER_ADDR")
	if url == "" || addr == "" {
		return nil, errors.New("NAVA_TEST_PG_URL and NAVA_ISOLATION_PGBOUNCER_ADDR must be set; run make isolation")
	}
	super, err := pgx.ParseConfig(url)
	if err != nil {
		return nil, err
	}
	m, am, err := audited.Load(db.Grants(), db.AuditedTables())
	if err != nil {
		return nil, err
	}
	g, err := catalog.ParseGlobalTables(db.GlobalTables())
	if err != nil {
		return nil, err
	}
	conn, err := pgx.ConnectConfig(ctx, super)
	if err != nil {
		return nil, err
	}
	defer func() { _ = conn.Close(ctx) }()
	var stmts []string
	for _, d := range m.DatabaseNames() {
		stmts = append(stmts, "DROP DATABASE IF EXISTS "+pgx.Identifier{d}.Sanitize()+" WITH (FORCE)")
	}
	for _, r := range m.RoleNames() {
		stmts = append(stmts, "DROP ROLE IF EXISTS "+pgx.Identifier{r}.Sanitize())
	}
	for _, s := range stmts {
		if _, err := conn.Exec(ctx, s); err != nil {
			return nil, fmt.Errorf("%s: %w", s, err)
		}
	}
	log := slog.New(slog.DiscardHandler)
	pw := func(role string) (string, bool) { return password(role), true }
	if _, err := roles.Run(ctx, super, m, pw, log); err != nil {
		return nil, fmt.Errorf("db-roles: %w", err)
	}
	mig := super.Copy()
	mig.User, mig.Password, mig.Database = m.Owner(), password(m.Owner()), m.PlatformDatabase
	if _, err := migrate.Run(ctx, mig, db.Migrations(), m, am, g, log, migrate.Options{LockRetrySeconds: 1}); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}

	e := &env{super: super.Copy(), bouncer: addr, m: m, g: g}
	e.super.Database = m.PlatformDatabase
	seed, err := os.ReadFile("testdata/seed.sql")
	if err != nil {
		return nil, err
	}
	sc, err := pgx.ConnectConfig(ctx, e.super)
	if err != nil {
		return nil, err
	}
	defer func() { _ = sc.Close(ctx) }()
	// The simple protocol runs the whole file as one implicit transaction.
	if _, err := sc.PgConn().Exec(ctx, string(seed)).ReadAll(); err != nil {
		return nil, fmt.Errorf("seed: %w", err)
	}
	// The seed turns the commit-check triggers off for its own transaction
	// (P5-13c, P5-32). A seed that stopped partway, or a loader that ran it
	// statement by statement, must not leave the suite running unchecked.
	for _, tr := range []struct{ table, name string }{
		{"records", "records_audit_commit_check"},
		{"entity_definitions", "entity_definitions_audit_commit_check"},
		{"entity_definition_versions", "entity_definition_versions_audit_commit_check"},
	} {
		var enabled string
		if err := sc.QueryRow(ctx, `SELECT coalesce((SELECT tgenabled::text FROM pg_trigger
			WHERE tgrelid = $1::regclass AND tgname = $2), 'missing')`, tr.table, tr.name).Scan(&enabled); err != nil {
			return nil, fmt.Errorf("seed: read %s: %w", tr.name, err)
		}
		if enabled != "O" {
			return nil, fmt.Errorf("seed: %s is %q after the seed, want enabled (O)", tr.name, enabled)
		}
	}
	if e.tables, err = catalog.TenantTables(ctx, sc, g); err != nil {
		return nil, err
	}
	if len(e.tables) == 0 {
		return nil, errors.New("no tenant tables discovered")
	}
	return e, nil
}

// superConn is the bootstrap superuser on the platform database.
func (e *env) superConn(t *testing.T) *pgx.Conn {
	t.Helper()
	conn, err := pgx.ConnectConfig(t.Context(), e.super)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(context.Background()) })
	return conn
}

// roleConfig is role on the platform database, directly or through
// PgBouncer.
func (e *env) roleConfig(role string, pooled bool) *pgx.ConnConfig {
	cfg := e.super.Copy()
	cfg.User, cfg.Password = role, password(role)
	if pooled {
		host, port, _ := net.SplitHostPort(e.bouncer)
		p, _ := strconv.ParseUint(port, 10, 16)
		// The test PgBouncer has no TLS (sslmode=prefer would fall back to
		// the direct server's address otherwise).
		cfg.Host, cfg.Port = host, uint16(p)
		cfg.TLSConfig, cfg.Fallbacks = nil, nil
	}
	return cfg
}

// connect returns the connection, or the SQLSTATE that refused it.
func (e *env) connect(t *testing.T, role string, pooled bool) (*pgx.Conn, string) {
	t.Helper()
	conn, err := pgx.ConnectConfig(t.Context(), e.roleConfig(role, pooled))
	if err != nil {
		if code := sqlState(err); code != "" {
			return nil, code
		}
		t.Fatalf("connect as %s (pooled %v): %v", role, pooled, err)
	}
	t.Cleanup(func() { _ = conn.Close(context.Background()) })
	return conn, ""
}

// pool is a one-connection pgxpool for tenancy.InTenantTx.
func (e *env) pool(t *testing.T, role string, pooled bool) *pgxpool.Pool {
	t.Helper()
	return e.poolSize(t, role, pooled, 1)
}

func (e *env) poolSize(t *testing.T, role string, pooled bool, conns int32) *pgxpool.Pool {
	t.Helper()
	cfg, err := pgxpool.ParseConfig("")
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig = e.roleConfig(role, pooled)
	cfg.MaxConns = conns
	p, err := pgxpool.NewWithConfig(t.Context(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(p.Close)
	return p
}

// cluster is the Cluster a DB path gets: pools and the superuser, closed
// when the test ends.
type cluster struct {
	t *testing.T
	e *env
}

func (c cluster) Pool(role string) (*pgxpool.Pool, error) {
	r, ok := c.e.m.Roles[role]
	if !ok {
		return nil, fmt.Errorf("%s is not a manifest role", role)
	}
	return c.e.pool(c.t, role, r.PgBouncerLogin), nil
}

func (c cluster) PoolSize(role string, conns int32) (*pgxpool.Pool, error) {
	if _, ok := c.e.m.Roles[role]; !ok {
		return nil, fmt.Errorf("%s is not a manifest role", role)
	}
	return c.e.poolSize(c.t, role, false, conns), nil
}

func (c cluster) PooledSize(role string, conns int32) (*pgxpool.Pool, error) {
	r, ok := c.e.m.Roles[role]
	if !ok {
		return nil, fmt.Errorf("%s is not a manifest role", role)
	}
	if !r.PgBouncerLogin {
		return nil, fmt.Errorf("%s does not log in through PgBouncer", role)
	}
	return c.e.poolSize(c.t, role, true, conns), nil
}

func (c cluster) Super() (*pgx.Conn, error) { return c.e.superConn(c.t), nil }

func (c cluster) AuditReader() (*read.Pool, error) {
	cfg := c.e.roleConfig(audited.ReaderRole, true)
	dsn := (&url.URL{
		Scheme: "postgres", User: url.UserPassword(cfg.User, cfg.Password),
		Host: net.JoinHostPort(cfg.Host, strconv.Itoa(int(cfg.Port))), Path: "/" + cfg.Database,
		RawQuery: "sslmode=disable",
	}).String()
	p, err := read.OpenPool(c.t.Context(), dsn)
	if err != nil {
		return nil, err
	}
	c.t.Cleanup(p.Close)
	return p, nil
}

// runtimeRoles are the roles CI-2 runs as: every manifest role but the
// owner (as catalog's CI-3 and CI-4 count them), then the owner, whom only
// FORCE ROW LEVEL SECURITY holds to the policies.
func (e *env) runtimeRoles() []string {
	var out []string
	for _, r := range e.m.RoleNames() {
		if r != e.m.Owner() {
			out = append(out, r)
		}
	}
	return append(out, e.m.Owner())
}

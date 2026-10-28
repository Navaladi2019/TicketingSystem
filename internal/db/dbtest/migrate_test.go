//go:build integration

package dbtest

import (
	"context"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/pressly/goose/v3"

	"github.com/Navaladi2019/TicketingSystem/db"
	"github.com/Navaladi2019/TicketingSystem/internal/db/grants"
	"github.com/Navaladi2019/TicketingSystem/internal/db/migrate"
)

func TestMigratorOwnsDatabaseAndIsGrantedToNoOne(t *testing.T) {
	m := freshCluster(t)
	res := migrateAll(t, m)
	if len(res.Applied) == 0 {
		t.Fatal("first run applied nothing")
	}
	conn := connect(t, superConfig(t))
	if got := queryString(t, conn, "SELECT pg_get_userbyid(datdba)::text FROM pg_database WHERE datname = 'nava'"); got != "nava_migrator" {
		t.Fatalf("datdba = %s, want nava_migrator", got)
	}
	if n := queryString(t, conn, `SELECT count(*)::text FROM pg_auth_members WHERE roleid = 'nava_migrator'::regrole`); n != "0" {
		t.Fatalf("%s pg_auth_members rows grant nava_migrator", n)
	}
	// It owns what it created, goose's version table included.
	if got := queryString(t, connect(t, roleConfig(t, "nava_migrator", "nava")),
		"SELECT pg_get_userbyid(relowner)::text FROM pg_class WHERE oid = 'public.goose_db_version'::regclass"); got != "nava_migrator" {
		t.Fatalf("goose_db_version owner = %s", got)
	}
}

func TestFirstMigrationGlobalRevokes(t *testing.T) {
	m := freshCluster(t)
	migrateAll(t, m)
	conn := connect(t, roleConfig(t, "nava_migrator", "nava"))

	// No PUBLIC grant on the database or schema public.
	if got := queryString(t, conn, `
		SELECT coalesce(string_agg(x, ','), '') FROM (
		  SELECT 'database ' || a.privilege_type AS x FROM pg_database d, aclexplode(coalesce(d.datacl, acldefault('d', d.datdba))) a
		  WHERE d.datname = 'nava' AND a.grantee = 0
		  UNION ALL
		  SELECT 'schema ' || a.privilege_type FROM pg_namespace n, aclexplode(coalesce(n.nspacl, acldefault('n', n.nspowner))) a
		  WHERE n.nspname = 'public' AND a.grantee = 0) s`); got != "" {
		t.Fatalf("PUBLIC still holds: %s", got)
	}
	// A function the migrator creates later is not executable by PUBLIC.
	exec(t, conn, "CREATE FUNCTION public.t03_probe() RETURNS int LANGUAGE sql AS 'SELECT 1'")
	if queryBool(t, conn, "SELECT has_function_privilege('keycloak', 'public.t03_probe()', 'EXECUTE')") {
		t.Fatal("a new function is executable through PUBLIC")
	}
	// Roles the manifest gives nothing on nava cannot connect.
	for _, role := range []string{"keycloak", "temporal"} {
		if _, err := connectErr(roleConfig(t, role, "nava")); sqlState(err) != "42501" {
			t.Errorf("%s connecting to nava: err = %v, want insufficient_privilege", role, err)
		}
	}
}

func TestAppCannotRunDDL(t *testing.T) {
	migratedCluster(t)
	app := connect(t, roleConfig(t, "nava_app", "nava"))
	for _, stmt := range []string{
		"CREATE TABLE public.t03_app_table (id int)",
		"CREATE SCHEMA t03_app_schema",
		"CREATE TEMP TABLE t03_app_temp (id int)",
		"CREATE FUNCTION public.t03_app_fn() RETURNS int LANGUAGE sql AS 'SELECT 1'",
		"CREATE VIEW public.t03_app_view AS SELECT 1",
		"ALTER TABLE public.goose_db_version ADD COLUMN x int",
		"DROP TABLE public.goose_db_version",
		"ALTER DATABASE nava SET work_mem = '1MB'",
	} {
		_, err := app.Exec(context.Background(), stmt)
		if code := sqlState(err); code != "42501" {
			t.Errorf("%s as nava_app: err = %v (SQLSTATE %q), want 42501 insufficient_privilege", stmt, err, code)
		}
	}
}

func TestManifestAppliedTwiceIsNoOp(t *testing.T) {
	m := freshCluster(t)
	first := migrateAll(t, m)
	if len(first.Grants.Granted) == 0 {
		t.Fatal("first run granted nothing")
	}
	second := migrateAll(t, m)
	if len(second.Applied) != 0 || second.Grants.Changed() {
		t.Fatalf("second run: applied %v, grants %+v; want nothing", second.Applied, second.Grants)
	}
}

func TestManifestRevokesAndLogsExtraGrants(t *testing.T) {
	m := migratedCluster(t)
	mig := connect(t, roleConfig(t, "nava_migrator", "nava"))
	exec(t, mig, "CREATE TABLE public.t03_extra (id int, secret text)")
	exec(t, mig, "CREATE SEQUENCE public.t03_seq")
	exec(t, mig, "CREATE FUNCTION public.t03_fn(integer) RETURNS int LANGUAGE sql AS 'SELECT $1'")
	extras := []string{
		"GRANT SELECT ON public.t03_extra TO nava_app",
		"GRANT UPDATE (secret) ON public.t03_extra TO keycloak",
		"GRANT USAGE ON SEQUENCE public.t03_seq TO PUBLIC",
		"GRANT EXECUTE ON FUNCTION public.t03_fn(integer) TO nava_app",
		"GRANT USAGE ON SCHEMA public TO keycloak",
		"GRANT CREATE ON SCHEMA public TO PUBLIC",
		"GRANT TEMPORARY ON DATABASE nava TO nava_app",
		"GRANT CONNECT ON DATABASE nava TO nava_app WITH GRANT OPTION", // listed grant: only the option goes
		"ALTER DEFAULT PRIVILEGES FOR ROLE nava_migrator GRANT SELECT ON TABLES TO nava_app",
		"ALTER DEFAULT PRIVILEGES FOR ROLE nava_migrator IN SCHEMA public GRANT EXECUTE ON FUNCTIONS TO keycloak",
		"ALTER DEFAULT PRIVILEGES FOR ROLE nava_migrator GRANT EXECUTE ON FUNCTIONS TO PUBLIC",
	}
	for _, s := range extras {
		exec(t, mig, s)
	}

	log, buf := newLog()
	res, err := grants.Apply(context.Background(), mig, m, log)
	if err != nil {
		t.Fatal(err)
	}
	wantRevoked := []string{
		"SELECT table public.t03_extra TO nava_app",
		"UPDATE column (secret) ON public.t03_extra TO keycloak",
		"USAGE table public.t03_seq TO PUBLIC",
		"EXECUTE function public.t03_fn(integer) TO nava_app",
		"USAGE schema public TO keycloak",
		"CREATE schema public TO PUBLIC",
		"TEMPORARY database nava TO nava_app",
		"CONNECT database nava TO nava_app WITH GRANT OPTION",
		"SELECT default DEFAULT TABLES TO nava_app",
		"EXECUTE default DEFAULT FUNCTIONS IN SCHEMA public TO keycloak",
		"EXECUTE default DEFAULT FUNCTIONS TO PUBLIC",
	}
	if strings.Join(sorted(res.Revoked), "\n") != strings.Join(sorted(wantRevoked), "\n") {
		t.Fatalf("revoked:\n%s\nwant:\n%s", strings.Join(sorted(res.Revoked), "\n"), strings.Join(sorted(wantRevoked), "\n"))
	}
	for _, w := range wantRevoked {
		if !strings.Contains(buf.String(), `"msg":"revoked grant not in manifest","grant":"`+w+`"`) {
			t.Errorf("revocation of %q not logged", w)
		}
	}
	if len(res.Granted) != 0 {
		t.Errorf("granted %v, want nothing (CONNECT kept without its option)", res.Granted)
	}
	for _, q := range []string{
		"SELECT has_table_privilege('nava_app', 'public.t03_extra', 'SELECT')",
		"SELECT has_column_privilege('keycloak', 'public.t03_extra', 'secret', 'UPDATE')",
		"SELECT has_sequence_privilege('keycloak', 'public.t03_seq', 'USAGE')",
		"SELECT has_function_privilege('nava_app', 'public.t03_fn(integer)', 'EXECUTE')",
		"SELECT has_database_privilege('nava_app', 'nava', 'TEMPORARY')",
		"SELECT has_database_privilege('nava_app', 'nava', 'CONNECT WITH GRANT OPTION')",
	} {
		if queryBool(t, mig, q) {
			t.Errorf("%s = true after apply", q)
		}
	}
	if !queryBool(t, mig, "SELECT has_database_privilege('nava_app', 'nava', 'CONNECT')") {
		t.Error("nava_app lost CONNECT, which the manifest lists")
	}
	// Objects created after the apply get no grant from a default.
	exec(t, mig, "CREATE TABLE public.t03_after (id int)")
	exec(t, mig, "CREATE FUNCTION public.t03_after_fn() RETURNS int LANGUAGE sql AS 'SELECT 1'")
	if queryBool(t, mig, "SELECT has_table_privilege('nava_app', 'public.t03_after', 'SELECT') OR has_function_privilege('keycloak', 'public.t03_after_fn()', 'EXECUTE')") {
		t.Error("a default privilege survived")
	}

	again, err := grants.Apply(context.Background(), mig, m, log)
	if err != nil || again.Changed() {
		t.Fatalf("apply after revoking: %+v, %v; want no change", again, err)
	}
}

// The applier grants exactly the listed columns, and a table-level grant in
// place of a column-level one is replaced.
func TestManifestColumnGrants(t *testing.T) {
	m := migratedCluster(t)
	mig := connect(t, roleConfig(t, "nava_migrator", "nava"))
	exec(t, mig, "CREATE TABLE public.t03_cols (id int, tenant_id uuid, secret text)")
	app := m.Roles["nava_app"]
	app.Tables = map[string]map[string][]string{"t03_cols": {"SELECT": {"id", "tenant_id"}, "UPDATE": {"secret"}, "DELETE": {}}}
	m.Roles["nava_app"] = app

	log, _ := newLog()
	res, err := grants.Apply(context.Background(), mig, m, log)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Granted) != 4 {
		t.Fatalf("granted %v, want 4", res.Granted)
	}
	check := func() {
		t.Helper()
		for q, want := range map[string]bool{
			"SELECT has_column_privilege('nava_app', 'public.t03_cols', 'id', 'SELECT')":        true,
			"SELECT has_column_privilege('nava_app', 'public.t03_cols', 'tenant_id', 'SELECT')": true,
			"SELECT has_column_privilege('nava_app', 'public.t03_cols', 'secret', 'SELECT')":    false,
			"SELECT has_column_privilege('nava_app', 'public.t03_cols', 'secret', 'UPDATE')":    true,
			"SELECT has_column_privilege('nava_app', 'public.t03_cols', 'id', 'UPDATE')":        false,
			"SELECT has_table_privilege('nava_app', 'public.t03_cols', 'DELETE')":               true,
			"SELECT has_table_privilege('nava_app', 'public.t03_cols', 'SELECT')":               false,
		} {
			if got := queryBool(t, mig, q); got != want {
				t.Errorf("%s = %v, want %v", q, got, want)
			}
		}
	}
	check()

	exec(t, mig, "GRANT SELECT ON public.t03_cols TO nava_app")
	res, err = grants.Apply(context.Background(), mig, m, log)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Revoked) != 1 || len(res.Granted) != 2 {
		t.Fatalf("after a table-level SELECT: revoked %v, granted %v; want 1 revoke, 2 column re-grants", res.Revoked, res.Granted)
	}
	check()
}

func TestManifestRejectsMissingObjects(t *testing.T) {
	m := migratedCluster(t)
	mig := connect(t, roleConfig(t, "nava_migrator", "nava"))
	app := m.Roles["nava_app"]
	app.Tables = map[string]map[string][]string{"no_such_table": {"SELECT": {}}}
	m.Roles["nava_app"] = app
	log, _ := newLog()
	if _, err := grants.Apply(context.Background(), mig, m, log); err == nil || !strings.Contains(err.Error(), "does not exist") {
		t.Fatalf("err = %v, want a missing-table error", err)
	}
}

func TestManifestRefusesWrongRole(t *testing.T) {
	m := migratedCluster(t)
	log, _ := newLog()
	for _, cfg := range []struct{ role, db string }{{"nava_app", "nava"}} {
		conn := connect(t, roleConfig(t, cfg.role, cfg.db))
		if _, err := grants.Apply(context.Background(), conn, m, log); err == nil || !strings.Contains(err.Error(), "applied by the owner") {
			t.Errorf("apply as %s: err = %v, want a refusal", cfg.role, err)
		}
	}
}

// Audit TICK-94 item 8: a second migrator started while the first holds the
// session lock waits, then finds nothing to apply.
func TestConcurrentMigratorWaitsForSessionLock(t *testing.T) {
	m := freshCluster(t)
	fsys := withExtraMigration(t, "99999_t03_slow.sql", "-- +goose Up\nSELECT pg_sleep(3);\n")
	cfg := roleConfig(t, m.Owner(), m.PlatformDatabase)
	log, _ := newLog()

	type run struct {
		applied []string
		err     error
		done    time.Time
	}
	first := make(chan run, 1)
	go func() {
		a, err := migrate.Up(context.Background(), cfg, fsys, log, migrate.Options{LockRetrySeconds: 1})
		first <- run{a, err, time.Now()}
	}()

	// Wait until the first run holds goose's session lock.
	super := connect(t, superConfig(t))
	deadline := time.Now().Add(30 * time.Second)
	for !queryBool(t, super, `SELECT EXISTS (SELECT 1 FROM pg_locks WHERE locktype = 'advisory' AND granted
		AND ((classid::bigint << 32) | objid::bigint) = 4097083626)`) {
		if time.Now().After(deadline) {
			t.Fatal("the first run never took the session lock")
		}
		time.Sleep(50 * time.Millisecond)
	}
	secondStart := time.Now()
	b, err := migrate.Up(context.Background(), cfg, fsys, log, migrate.Options{LockRetrySeconds: 1})
	secondDone := time.Now()
	a := <-first

	if a.err != nil || err != nil {
		t.Fatalf("first: %v, second: %v", a.err, err)
	}
	if len(a.applied) != countMigrations(t, fsys) {
		t.Fatalf("first run applied %v", a.applied)
	}
	if len(b) != 0 {
		t.Fatalf("second run applied %v, want nothing", b)
	}
	if !secondStart.Before(a.done) || secondDone.Before(a.done) {
		t.Fatalf("second run did not wait: started %v, first done %v, second done %v", secondStart, a.done, secondDone)
	}
	if n := queryString(t, connect(t, cfg), "SELECT count(*)::text FROM goose_db_version WHERE version_id = 99999"); n != "1" {
		t.Fatalf("slow migration recorded %s times", n)
	}
}

// Audit TICK-94 item 8: a "-- +goose NO TRANSACTION" file leaves partial
// state with no marker when it fails, so it must be safe to run again.
// Re-run every such file over a migrated database; it must not fail.
func TestNoTransactionMigrationsAreIdempotent(t *testing.T) {
	m := migratedCluster(t)
	only := fstest.MapFS{}
	entries, err := fs.ReadDir(db.Migrations(), ".")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		b, err := fs.ReadFile(db.Migrations(), e.Name())
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(b), "+goose NO TRANSACTION") {
			only[e.Name()] = &fstest.MapFile{Data: b}
		}
	}
	if len(only) == 0 {
		t.Skip("no NO TRANSACTION migrations yet")
	}
	sqlDB := openSQL(t, roleConfig(t, m.Owner(), m.PlatformDatabase))
	p, err := goose.NewProvider(goose.DialectPostgres, sqlDB, only,
		goose.WithDisableVersioning(true), goose.WithDisableGlobalRegistry(true))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.Up(context.Background()); err != nil {
		t.Fatalf("re-running NO TRANSACTION migrations: %v", err)
	}
}

// TICK-290: a timestamp migration from a long-lived branch can be older than
// one already applied. The migrator applies it instead of refusing the run.
func TestOutOfOrderMigrationApplies(t *testing.T) {
	m := migratedCluster(t)
	late := withExtraMigration(t, "29991231000001_t290_late.sql", "-- +goose Up\nSELECT 1;\n")
	if _, err := migrateFS(t, m, late); err != nil {
		t.Fatalf("migrate with the later migration: %v", err)
	}
	both := late.(fstest.MapFS)
	both["29991231000000_t290_early.sql"] = &fstest.MapFile{Data: []byte("-- +goose Up\nSELECT 1;\n")}
	res, err := migrateFS(t, m, both)
	if err != nil {
		t.Fatalf("migrate with an older, unapplied migration: %v", err)
	}
	if len(res.Applied) != 1 || res.Applied[0] != "29991231000000_t290_early.sql" {
		t.Fatalf("applied %v, want only the older migration", res.Applied)
	}
}

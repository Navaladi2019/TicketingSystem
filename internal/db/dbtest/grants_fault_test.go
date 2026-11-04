//go:build integration

package dbtest

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/db"
	"github.com/Navaladi2019/TicketingSystem/internal/db/catalog"
	"github.com/Navaladi2019/TicketingSystem/internal/db/grants"
	"github.com/Navaladi2019/TicketingSystem/internal/db/migrate"
)

// aclSnapshot is every ACL in the platform database the manifest step
// converges: the database's, every schema's, relation's, column's and
// function's in a platform schema, and the default privileges.
func aclSnapshot(t *testing.T, conn *pgx.Conn) string {
	t.Helper()
	return queryString(t, conn, `
		SELECT concat_ws(E'\n',
		  (SELECT datacl::text FROM pg_database WHERE datname = current_database()),
		  (SELECT string_agg(nspname || '=' || coalesce(nspacl::text, ''), ',' ORDER BY nspname) FROM pg_namespace
		   WHERE nspname NOT LIKE 'pg\_%' AND nspname <> 'information_schema'),
		  (SELECT string_agg(c.oid::regclass::text || '=' || coalesce(c.relacl::text, ''), ',' ORDER BY c.oid::regclass::text)
		   FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace WHERE n.nspname = 'public'),
		  (SELECT string_agg(a.attrelid::regclass::text || '.' || a.attname || '=' || a.attacl::text, ',' ORDER BY a.attrelid::regclass::text, a.attname)
		   FROM pg_attribute a JOIN pg_class c ON c.oid = a.attrelid JOIN pg_namespace n ON n.oid = c.relnamespace
		   WHERE n.nspname = 'public' AND a.attacl IS NOT NULL),
		  (SELECT string_agg(p.oid::regprocedure::text || '=' || coalesce(p.proacl::text, ''), ',' ORDER BY p.oid::regprocedure::text)
		   FROM pg_proc p JOIN pg_namespace n ON n.oid = p.pronamespace WHERE n.nspname = 'public'),
		  (SELECT string_agg(defaclrole::regrole::text || ':' || defaclnamespace::regnamespace::text || ':' || defaclobjtype::text || '=' || defaclacl::text, ',' ORDER BY defaclrole, defaclnamespace, defaclobjtype) FROM pg_default_acl))`)
}

// TICK-522 acceptance: grant drift against real PostgreSQL. A missing grant
// and an extra grant are each reported by the catalogue gate, and the
// manifest step restores the one and revokes the other, logging both.
func TestManifestDetectsAndReportsGrantDrift(t *testing.T) {
	m := migratedCluster(t)
	mig := connect(t, roleConfig(t, m.Owner(), m.PlatformDatabase))
	if !slices.Contains(m.Roles["nava_relay"].Database, "CONNECT") {
		t.Fatal("db/grants.yaml: nava_relay has no CONNECT; pick another listed grant for this test")
	}
	exec(t, mig, "REVOKE CONNECT ON DATABASE nava FROM nava_relay") // missing
	exec(t, mig, "GRANT TEMPORARY ON DATABASE nava TO nava_relay")  // extra
	exec(t, mig, "GRANT TRUNCATE ON public.tenants TO nava_app")    // extra, on a tenant table

	problems, err := catalog.Check(context.Background(), mig, m, globalTables(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"CI-4: nava_relay lacks CONNECT on database nava, which db/grants.yaml lists",
		"CI-4: nava_relay has TEMPORARY on database nava, which db/grants.yaml does not list",
		"CI-4: nava_app has TRUNCATE on table public.tenants, which db/grants.yaml does not list",
	} {
		if !slices.Contains(problems, want) {
			t.Errorf("gate did not report %q; problems:\n%s", want, strings.Join(problems, "\n"))
		}
	}

	log, buf := newLog()
	res, err := grants.Apply(context.Background(), mig, m, log)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"CONNECT database nava TO nava_relay"}; !slices.Equal(res.Granted, want) {
		t.Errorf("granted = %v, want %v", res.Granted, want)
	}
	if want := []string{"TEMPORARY database nava TO nava_relay", "TRUNCATE table public.tenants TO nava_app"}; !slices.Equal(sorted(res.Revoked), want) {
		t.Errorf("revoked = %v, want %v", res.Revoked, want)
	}
	for _, want := range []string{
		`"msg":"granted","grant":"CONNECT database nava TO nava_relay"`,
		`"msg":"revoked grant not in manifest","grant":"TRUNCATE table public.tenants TO nava_app"`,
	} {
		if !strings.Contains(buf.String(), want) {
			t.Errorf("log has no %s:\n%s", want, buf)
		}
	}
	if problems, err := catalog.Check(context.Background(), mig, m, globalTables(t)); err != nil || len(problems) != 0 {
		t.Fatalf("after Apply: problems %v, err %v; want a clean gate", problems, err)
	}
}

// Apply runs in one transaction: a statement that fails part way leaves
// every grant as it was, the revokes already made included.
func TestManifestApplyIsAtomicOnEveryStatementFailure(t *testing.T) {
	cases := []struct {
		name  string
		match func(string) bool
		nth   int
		wrap  []string
	}{
		{name: "begin", match: sqlIs("begin"), nth: 1},
		{name: "search_path", match: sqlHas("SET LOCAL search_path"), nth: 1},
		{name: "advisory lock", match: sqlHas("pg_advisory_xact_lock"), nth: 1},
		{name: "owner check", match: sqlHas("d.datdba = r.oid, r.rolsuper"), nth: 1},
		{name: "schema lookup", match: sqlHas("FROM pg_catalog.pg_namespace n WHERE n.nspname = $1"), nth: 1},
		{name: "table lookup", match: sqlHas("SELECT c.relkind::text, c.relowner"), nth: 1},
		{name: "column lookup", match: sqlHas("SELECT EXISTS (SELECT 1 FROM pg_catalog.pg_attribute a"), nth: 1},
		{name: "read grants", match: sqlHas("WITH me AS"), nth: 1},
		{name: "re-read grants", match: sqlHas("WITH me AS"), nth: 2},
		{name: "revoke", match: sqlHas("REVOKE TEMPORARY ON DATABASE"), nth: 1, wrap: []string{"revoke TEMPORARY database nava TO nava_relay: "}},
		{name: "grant", match: sqlHas("GRANT CONNECT ON DATABASE"), nth: 1, wrap: []string{"grant CONNECT database nava TO nava_relay: "}},
	}
	// One drifted cluster for every case: each case checks it leaves every
	// grant as it was, so the next starts from the same drift.
	m := migratedCluster(t)
	mig := connect(t, roleConfig(t, m.Owner(), m.PlatformDatabase))
	exec(t, mig, "REVOKE CONNECT ON DATABASE nava FROM nava_relay")
	exec(t, mig, "GRANT TEMPORARY ON DATABASE nava TO nava_relay")
	before := aclSnapshot(t, mig)
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cfg, f := withFault(roleConfig(t, m.Owner(), m.PlatformDatabase), c.match, c.nth)
			conn := connect(t, cfg)
			log, _ := newLog()
			res, err := grants.Apply(context.Background(), conn, m, log)
			wantFault(t, err, f, c.wrap...)
			if c.name == "grant" && len(res.Revoked) != 1 {
				t.Errorf("revoked = %v; the revoke ran before the failed grant", res.Revoked)
			}
			if after := aclSnapshot(t, mig); after != before {
				t.Fatalf("a failed Apply changed grants:\nbefore %s\nafter  %s", before, after)
			}
		})
	}
}

// checkOwner refuses to apply anywhere but the platform database, as anyone
// but its owner, or as a superuser.
func TestManifestRefusesWrongDatabaseOwnerOrSuperuser(t *testing.T) {
	m := migratedCluster(t)
	super := connect(t, superConfig(t))
	log, _ := newLog()

	t.Run("another database", func(t *testing.T) {
		conn := connect(t, roleConfig(t, m.Owner(), "postgres"))
		_, err := grants.Apply(context.Background(), conn, m, log)
		if err == nil || err.Error() != `connected to database "postgres", manifest platform database is "nava"` {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("not the database owner", func(t *testing.T) {
		conn := connect(t, roleConfig(t, m.Owner(), m.PlatformDatabase))
		exec(t, super, "ALTER DATABASE nava OWNER TO CURRENT_USER")
		t.Cleanup(func() { exec(t, super, "ALTER DATABASE nava OWNER TO nava_migrator") })
		_, err := grants.Apply(context.Background(), conn, m, log)
		if err == nil || !strings.Contains(err.Error(), `"nava_migrator" is not the owner (datdba) of "nava"`) {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("superuser", func(t *testing.T) {
		conn := connect(t, roleConfig(t, m.Owner(), m.PlatformDatabase))
		exec(t, super, "ALTER ROLE nava_migrator SUPERUSER")
		t.Cleanup(func() { exec(t, super, "ALTER ROLE nava_migrator NOSUPERUSER") })
		_, err := grants.Apply(context.Background(), conn, m, log)
		if err == nil || err.Error() != `"nava_migrator" is a superuser` {
			t.Fatalf("err = %v", err)
		}
	})
}

// Every manifest object must exist and be owned by the migrator: Apply
// never grants on an object someone else owns.
func TestManifestRefusesMissingOrForeignObjects(t *testing.T) {
	m := migratedCluster(t)
	plat := connect(t, platformSuperConfig(t, m))
	exec(t, plat, "CREATE SCHEMA t522_foreign")
	exec(t, plat, "CREATE TABLE public.t522_foreign (id int)")
	exec(t, plat, "CREATE FUNCTION public.t522_foreign(integer) RETURNS int LANGUAGE sql AS 'SELECT $1'")
	mig := connect(t, roleConfig(t, m.Owner(), m.PlatformDatabase))
	log, _ := newLog()

	for name, c := range map[string]struct {
		role grants.Role
		want string
	}{
		"missing schema":   {grants.Role{Schemas: map[string][]string{"t522_missing": {"USAGE"}}}, "role nava_app: schema t522_missing does not exist"},
		"foreign schema":   {grants.Role{Schemas: map[string][]string{"t522_foreign": {"USAGE"}}}, "role nava_app: schema t522_foreign is not owned by nava_migrator"},
		"foreign table":    {grants.Role{Tables: map[string]map[string][]string{"t522_foreign": {"SELECT": nil}}}, "role nava_app: table public.t522_foreign is not owned by nava_migrator"},
		"missing function": {grants.Role{Functions: map[string][]string{"public.t522_missing(integer)": {"EXECUTE"}}}, "role nava_app: function public.t522_missing(integer) does not exist"},
		"foreign function": {grants.Role{Functions: map[string][]string{"public.t522_foreign(integer)": {"EXECUTE"}}}, "role nava_app: function public.t522_foreign(integer) is not owned by nava_migrator"},
	} {
		t.Run(name, func(t *testing.T) {
			mm := manifest(t)
			mm.Roles["nava_app"] = c.role
			before := aclSnapshot(t, mig)
			if _, err := grants.Apply(context.Background(), mig, mm, log); err == nil || err.Error() != c.want {
				t.Fatalf("err = %v, want %q", err, c.want)
			}
			if after := aclSnapshot(t, mig); after != before {
				t.Fatal("a refused Apply changed grants")
			}
		})
	}
}

// A function grant in the manifest is resolved through regprocedure and
// applied; the function lookup failing stops Apply like any other.
func TestManifestFunctionGrants(t *testing.T) {
	m := migratedCluster(t)
	mig := connect(t, roleConfig(t, m.Owner(), m.PlatformDatabase))
	exec(t, mig, "CREATE FUNCTION public.t522_fn(integer) RETURNS int LANGUAGE sql AS 'SELECT $1'")
	app := m.Roles["nava_app"]
	app.Functions = map[string][]string{"t522_fn(integer)": {"EXECUTE"}} // bare: schema public
	m.Roles["nava_app"] = app

	cfg, f := withFault(roleConfig(t, m.Owner(), m.PlatformDatabase), sqlHas("p.oid::pg_catalog.regprocedure::text,"), 1)
	log, _ := newLog()
	_, err := grants.Apply(context.Background(), connect(t, cfg), m, log)
	wantFault(t, err, f)

	res, err := grants.Apply(context.Background(), mig, m, log)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"EXECUTE function public.t522_fn(integer) TO nava_app"}; !slices.Equal(res.Granted, want) {
		t.Fatalf("granted = %v, want %v", res.Granted, want)
	}
	if !queryBool(t, mig, "SELECT has_function_privilege('nava_app', 'public.t522_fn(integer)', 'EXECUTE')") {
		t.Fatal("nava_app cannot execute the function after Apply")
	}
	if !queryBool(t, mig, "SELECT NOT has_function_privilege('nava_relay', 'public.t522_fn(integer)', 'EXECUTE')") {
		t.Fatal("a role the manifest does not list can execute the function")
	}
}

// The catalogue gate failing to run fails the migrate run with its error,
// after the manifest step: a gate that cannot run never passes.
func TestMigrateFailsWhenGateCannotRun(t *testing.T) {
	m := migratedCluster(t)
	cfg, f := withFault(roleConfig(t, m.Owner(), m.PlatformDatabase), sqlIs("SELECT current_database()::text"), 1)
	log, buf := newLog()
	_, err := migrate.Run(context.Background(), cfg, db.Migrations(), m, globalTables(t), log, migrate.Options{LockRetrySeconds: 1})
	wantFault(t, err, f)
	if strings.Contains(err.Error(), "apply grants manifest") {
		t.Fatalf("err = %v; the manifest step should have passed", err)
	}
	if strings.Contains(buf.String(), `"msg":"migrate done"`) {
		t.Fatalf("a failed run logged success:\n%s", buf)
	}
}

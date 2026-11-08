//go:build integration

package dbtest

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/db/grants"
	"github.com/Navaladi2019/TicketingSystem/internal/db/roles"
)

// TICK-522: every statement db-roles runs can fail (a cancelled job, a
// dropped connection, a server restart). Each case below makes the drift
// that sends the run down one path, fails that path's statement, and checks
// that the run stops with the error and names where it stopped. A failed run
// must leave a cluster the next run converges: each case ends with a clean
// run that succeeds and a second one that changes nothing.
func TestDBRolesStopsOnEveryStatementFailure(t *testing.T) {
	// The revokePublic and ownerOnlyDatabase reads, as the statements are sent.
	const publicRead = "WHERE d.datname = $1 AND a.grantee = 0"
	const serviceRead = "WHERE d.datname = $1 AND a.grantee <> d.datdba"
	cases := []struct {
		name  string
		setup []string // as the bootstrap superuser, in database postgres
		plat  []string // as the bootstrap superuser, in the platform database
		match func(string) bool
		nth   int
		exec  bool // QueryExecModeExec: a Query fails at rows.Err, not at Query
		wrap  []string
	}{
		{name: "identity check", match: sqlHas("SELECT current_user::text, rolsuper"), nth: 1},
		{name: "advisory lock", match: sqlHas("pg_advisory_lock(hashtext('nava.db-roles'))"), nth: 1},
		{name: "superuser check", match: sqlHas("WHERE rolsuper AND rolname <> $1"), nth: 1},
		{name: "role lookup", match: sqlHas("FROM pg_catalog.pg_authid WHERE rolname = $1"), nth: 1, wrap: []string{"role keycloak: "}},
		{
			name: "create role", setup: []string{"DROP ROLE nava_relay"},
			match: sqlHas("CREATE ROLE"), nth: 1, wrap: []string{"role nava_relay: "},
		},
		{
			name: "converge role attributes", setup: []string{"ALTER ROLE nava_app CREATEDB"},
			match: sqlHas(`ALTER ROLE "nava_app" NOSUPERUSER`), nth: 1, wrap: []string{"role nava_app: "},
		},
		{name: "settings query", match: sqlHas("FROM pg_catalog.pg_db_role_setting s"), nth: 1, wrap: []string{"role keycloak settings: "}},
		{name: "settings rows", match: sqlHas("FROM pg_catalog.pg_db_role_setting s"), nth: 2, wrap: []string{"role nava_app settings: "}},
		{
			name: "reset per-database settings", setup: []string{"ALTER ROLE nava_app IN DATABASE postgres SET work_mem = '2MB'"},
			match: sqlHas(`ALTER ROLE "nava_app" IN DATABASE`, "RESET ALL"), nth: 1, wrap: []string{"role nava_app settings: "},
		},
		{
			name: "reset role-wide settings", setup: []string{"ALTER ROLE nava_app SET work_mem = '2MB'"},
			match: sqlIs(`ALTER ROLE "nava_app" RESET ALL`), nth: 1, wrap: []string{"role nava_app settings: "},
		},
		{
			name: "set role-wide setting", setup: []string{"ALTER ROLE nava_app SET work_mem = '2MB'"},
			match: sqlHas(`ALTER ROLE "nava_app" SET `), nth: 1, wrap: []string{"role nava_app settings: "},
		},
		{name: "memberships query", match: sqlHas("FROM pg_catalog.pg_auth_members am"), nth: 1},
		{name: "memberships rows", match: sqlHas("FROM pg_catalog.pg_auth_members am"), nth: 1, exec: true},
		{
			name: "revoke membership", setup: []string{"GRANT pg_read_all_data TO keycloak"},
			match: sqlHas("GRANTED BY"), nth: 1, wrap: []string{"revoke pg_read_all_data from keycloak: "},
		},
		{name: "database lookup", match: sqlHas("FROM pg_catalog.pg_database WHERE datname = $1"), nth: 1, wrap: []string{"database keycloak: "}},
		{
			name: "create database", setup: []string{"DROP DATABASE temporal_visibility WITH (FORCE)"},
			match: sqlHas("CREATE DATABASE"), nth: 1, wrap: []string{"database temporal_visibility: "},
		},
		{
			name: "change database owner", setup: []string{"ALTER DATABASE temporal OWNER TO CURRENT_USER"},
			match: sqlHas("OWNER TO"), nth: 1, wrap: []string{"database temporal: "},
		},
		{name: "platform PUBLIC read", match: sqlHas(publicRead), nth: 1, wrap: []string{"database nava: "}},
		{
			name: "platform PUBLIC revoke", setup: []string{"GRANT CONNECT ON DATABASE nava TO PUBLIC"},
			match: sqlHas("REVOKE ALL ON DATABASE"), nth: 1, wrap: []string{"database nava: "},
		},
		{
			name: "platform PUBLIC re-read", setup: []string{"GRANT CONNECT ON DATABASE nava TO PUBLIC"},
			match: sqlHas(publicRead), nth: 2, wrap: []string{"database nava: "},
		},
		{name: "service database read", match: sqlHas(serviceRead), nth: 1, wrap: []string{"database keycloak: "}},
		{name: "service database re-read", match: sqlHas(serviceRead), nth: 2, wrap: []string{"database keycloak: "}},
		{
			name: "service database revoke", setup: []string{"GRANT CONNECT ON DATABASE keycloak TO nava_app"},
			match: sqlHas(`ON DATABASE "keycloak" FROM`), nth: 1, wrap: []string{"database keycloak: ", "revoke CONNECT from nava_app: "},
		},
		{name: "get_auth allow-list", match: sqlIs("SELECT $1::text[]::text"), nth: 1, wrap: []string{"pgbouncer_get_auth: "}},
		{name: "get_auth owner check", match: sqlHas("SELECT 'schema', r.rolname::text"), nth: 1, wrap: []string{"pgbouncer_get_auth: "}},
		{name: "get_auth owner check rows", match: sqlHas("SELECT 'schema', r.rolname::text"), nth: 1, exec: true, wrap: []string{"pgbouncer_get_auth: "}},
		{
			name: "create get_auth schema", plat: []string{"DROP SCHEMA pgbouncer CASCADE"},
			match: sqlHas("CREATE SCHEMA"), nth: 1, wrap: []string{"pgbouncer_get_auth: "},
		},
		{name: "read get_auth", match: sqlHas("SELECT p.prosrc, p.prosecdef"), nth: 1, wrap: []string{"pgbouncer_get_auth: "}},
		{
			name: "write get_auth", plat: []string{"DROP FUNCTION pgbouncer.get_auth(text)"},
			match: sqlHas("get_auth(p_usename text)"), nth: 1, wrap: []string{"pgbouncer_get_auth: "},
		},
		{name: "get_auth grants query", match: sqlHas("pg_catalog.pg_namespace n, pg_catalog.aclexplode"), nth: 1, wrap: []string{"pgbouncer_get_auth: "}},
		{name: "get_auth grants rows", match: sqlHas("pg_catalog.pg_namespace n, pg_catalog.aclexplode"), nth: 1, exec: true, wrap: []string{"pgbouncer_get_auth: "}},
		{
			name: "revoke stray get_auth grant", plat: []string{"GRANT USAGE ON SCHEMA pgbouncer TO nava_app"},
			match: sqlHas("REVOKE ALL ON SCHEMA"), nth: 1, wrap: []string{"pgbouncer_get_auth: "},
		},
		{
			name: "grant missing get_auth grant", plat: []string{"REVOKE USAGE ON SCHEMA pgbouncer FROM pgbouncer_auth"},
			match: sqlHas("GRANT USAGE ON SCHEMA"), nth: 1, wrap: []string{"pgbouncer_get_auth: "},
		},
	}
	// One fresh cluster for every case: each case ends converged (it checks
	// that), which is the state the next one starts from. A failed case stops
	// the rest, which would start from a cluster in an unknown state.
	m := freshCluster(t)
	super := connect(t, superConfig(t))
	for _, c := range cases {
		if !t.Run(c.name, func(t *testing.T) {
			for _, s := range c.setup {
				exec(t, super, s)
			}
			if len(c.plat) > 0 {
				plat := connect(t, platformSuperConfig(t, m))
				for _, s := range c.plat {
					exec(t, plat, s)
				}
			}
			cfg, f := withFault(superConfig(t), c.match, c.nth)
			if c.exec {
				cfg.DefaultQueryExecMode = pgx.QueryExecModeExec
			}
			log, buf := newLog()
			_, err := roles.Run(context.Background(), cfg, m, passwords, log)
			wantFault(t, err, f, c.wrap...)
			if strings.Contains(buf.String(), `"msg":"db-roles done"`) {
				t.Fatalf("a failed run logged success:\n%s", buf)
			}
			wantConverges(t, m)
		}) {
			t.FailNow()
		}
	}
}

// wantConverges runs db-roles twice: the first converges the cluster, the
// second changes nothing.
func wantConverges(t *testing.T, m *grants.Manifest) {
	t.Helper()
	log, buf := newLog()
	if _, err := roles.Run(context.Background(), superConfig(t), m, passwords, log); err != nil {
		t.Fatalf("clean run after the failure: %v\n%s", err, buf)
	}
	res, err := roles.Run(context.Background(), superConfig(t), m, passwords, log)
	if err != nil || res.Total() != 0 {
		t.Fatalf("second clean run: %+v, %v; want no changes\n%s", res, err, buf)
	}
}

// The second connection, to the platform database for get_auth, fails: the
// run stops with that error before it touches get_auth.
func TestDBRolesStopsWhenPlatformConnectionFails(t *testing.T) {
	m := freshCluster(t)
	plat := connect(t, platformSuperConfig(t, m))
	exec(t, plat, "REVOKE USAGE ON SCHEMA pgbouncer FROM pgbouncer_auth")
	log, _ := newLog()
	_, err := roles.Run(context.Background(), failNthConnect(superConfig(t), 2), m, passwords, log)
	if !errors.Is(err, errConnectRefused) {
		t.Fatalf("err = %v, want the refused platform connection", err)
	}
	if queryBool(t, plat, "SELECT has_schema_privilege('pgbouncer_auth', 'pgbouncer', 'USAGE')") {
		t.Fatal("get_auth grants changed although the platform connection failed")
	}
	wantConverges(t, m)
}

func TestDBRolesRefusesBeforeConnecting(t *testing.T) {
	m := freshCluster(t)
	before := snapshotRoles(t)
	log, _ := newLog()

	t.Run("not exactly the get_auth exception", func(t *testing.T) {
		for name, ex := range map[string]map[string]grants.Exception{
			"none":  {},
			"extra": {"pgbouncer_get_auth": m.InfrastructureExceptions["pgbouncer_get_auth"], "other": {}},
		} {
			mm := manifest(t)
			mm.InfrastructureExceptions = ex
			if _, err := roles.Run(context.Background(), superConfig(t), mm, passwords, log); err == nil ||
				!strings.Contains(err.Error(), "builds exactly one infrastructure exception") {
				t.Errorf("%s: err = %v, want a refusal", name, err)
			}
		}
	})
	t.Run("bad credentials", func(t *testing.T) {
		cfg := superConfig(t)
		cfg.Password = "not-the-password"
		_, err := roles.Run(context.Background(), cfg, m, passwords, log)
		if got := sqlState(err); got != "28P01" {
			t.Fatalf("err = %v (SQLSTATE %q), want invalid_password 28P01", err, got)
		}
	})
	t.Run("manifest lists the bootstrap superuser", func(t *testing.T) {
		mm := manifest(t)
		mm.Roles[superConfig(t).User] = grants.Role{}
		_, err := roles.Run(context.Background(), superConfig(t), mm, passwords, log)
		if err == nil || !strings.Contains(err.Error(), "lists the bootstrap superuser") {
			t.Fatalf("err = %v, want a refusal", err)
		}
	})
	t.Run("password not printable ASCII", func(t *testing.T) {
		pw := func(role string) (string, bool) {
			if role == "keycloak" {
				return "naïve-pässword", true
			}
			return passwords(role)
		}
		_, err := roles.Run(context.Background(), superConfig(t), m, pw, log)
		if err == nil || err.Error() != "role keycloak: password must be printable ASCII" {
			t.Fatalf("err = %v, want the password refused", err)
		}
	})
	t.Run("get_auth exception of the wrong shape", func(t *testing.T) {
		mm := manifest(t)
		ex := mm.InfrastructureExceptions["pgbouncer_get_auth"]
		ex.SecurityDefiner = false
		mm.InfrastructureExceptions["pgbouncer_get_auth"] = ex
		_, err := roles.Run(context.Background(), superConfig(t), mm, passwords, log)
		if err == nil || !strings.Contains(err.Error(), "pgbouncer_get_auth: expected a security_definer function pgbouncer.get_auth(text)") {
			t.Fatalf("err = %v, want the exception refused", err)
		}
	})
	if after := snapshotRoles(t); after != before {
		t.Fatalf("a refused run changed the roles:\nbefore %s\nafter  %s", before, after)
	}
}

// A role the manifest lists without LOGIN is NOLOGIN with no password, and
// a password or LOGIN given to it by hand is drift: detected, logged and
// removed.
func TestDBRolesNoLoginRoleHasNoPassword(t *testing.T) {
	freshCluster(t)
	m := manifest(t)
	m.Roles["nava_test_intruder"] = grants.Role{} // freshCluster drops it
	log, buf := newLog()
	res, err := roles.Run(context.Background(), superConfig(t), m, passwords, log)
	if err != nil {
		t.Fatal(err)
	}
	if res.Created != 1 {
		t.Fatalf("created = %d, want 1 (the NOLOGIN role)\n%s", res.Created, buf)
	}
	super := connect(t, superConfig(t))
	state := func() string {
		return queryString(t, super, `SELECT rolcanlogin::text || ' ' || coalesce(rolpassword, 'NULL')
			FROM pg_authid WHERE rolname = 'nava_test_intruder'`)
	}
	if got := state(); got != "false NULL" {
		t.Fatalf("NOLOGIN role: login and password = %s, want false NULL", got)
	}

	exec(t, super, "ALTER ROLE nava_test_intruder LOGIN PASSWORD 'hand-set-password'")
	log, buf = newLog()
	res, err = roles.Run(context.Background(), superConfig(t), m, passwords, log)
	if err != nil {
		t.Fatal(err)
	}
	if res.Changed != 1 {
		t.Fatalf("changed = %d, want 1\n%s", res.Changed, buf)
	}
	if !strings.Contains(buf.String(), `"msg":"converged role attributes or password","role":"nava_test_intruder"`) {
		t.Fatalf("the drift was not logged:\n%s", buf)
	}
	if got := state(); got != "false NULL" {
		t.Fatalf("after the drift: login and password = %s, want false NULL", got)
	}
}

// With no role on the pooler's allow-list, get_auth is rewritten to return
// no verifier at all: an empty list never turns into "anyone".
func TestDBRolesEmptyPgBouncerAllowList(t *testing.T) {
	freshCluster(t)
	m := manifest(t)
	for name, r := range m.Roles {
		r.PgBouncerLogin = false
		m.Roles[name] = r
	}
	log, buf := newLog()
	res, err := roles.Run(context.Background(), superConfig(t), m, passwords, log)
	if err != nil {
		t.Fatal(err)
	}
	if res.Changed != 1 || !strings.Contains(buf.String(), `"msg":"wrote PgBouncer lookup function"`) {
		t.Fatalf("result %+v, want get_auth rewritten once\n%s", res, buf)
	}
	plat := connect(t, platformSuperConfig(t, m))
	if src := queryString(t, plat, "SELECT prosrc FROM pg_proc WHERE oid = 'pgbouncer.get_auth(text)'::regprocedure"); !strings.Contains(src, "ANY ('{}'::name[])") {
		t.Fatalf("get_auth body = %s, want an empty allow-list", src)
	}
	if n := queryString(t, plat, "SELECT count(*)::text FROM pgbouncer.get_auth('nava_app')"); n != "0" {
		t.Fatalf("get_auth returned %s rows for nava_app, want 0", n)
	}
}

//go:build integration

package dbtest

import (
	"context"
	"strings"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/db/roles"
)

// T02 security review: get_auth answers only for the pgbouncer_login
// allow-list, so the migrator (and any superuser, BYPASSRLS or REPLICATION
// role) cannot log in through PgBouncer. The stack test logs in through the
// real pooler; this one checks the lookup itself.
func TestGetAuthAllowList(t *testing.T) {
	m := freshCluster(t)
	migrateAll(t, m)
	auth := connect(t, roleConfig(t, "pgbouncer_auth", "nava"))
	super := connect(t, superConfig(t))
	me := queryString(t, super, "SELECT current_user::text")
	exec(t, super, "CREATE ROLE nava_test_intruder LOGIN BYPASSRLS PASSWORD 'x'")

	for role, want := range map[string]string{
		"nava_app":           "1",
		"nava_migrator":      "0",
		"keycloak":           "0",
		"temporal":           "0",
		"pgbouncer_auth":     "0",
		me:                   "0",
		"nava_test_intruder": "0",
	} {
		if got := queryString(t, auth, "SELECT count(*)::text FROM pgbouncer.get_auth($1)", role); got != want {
			t.Errorf("get_auth(%s) returned %s rows, want %s", role, got, want)
		}
	}

	// Even on the allow-list, a role that gains BYPASSRLS or REPLICATION drops out.
	exec(t, super, "ALTER ROLE nava_app BYPASSRLS")
	if got := queryString(t, auth, "SELECT count(*)::text FROM pgbouncer.get_auth('nava_app')"); got != "0" {
		t.Error("get_auth returned a BYPASSRLS role")
	}
	exec(t, super, "ALTER ROLE nava_app NOBYPASSRLS")
}

func TestGetAuthReachableOnlyByPgBouncerAuth(t *testing.T) {
	m := freshCluster(t)
	migrateAll(t, m)
	app := connect(t, roleConfig(t, "nava_app", "nava"))
	if _, err := app.Exec(context.Background(), "SELECT * FROM pgbouncer.get_auth('nava_app')"); sqlState(err) != "42501" {
		t.Fatalf("nava_app calling get_auth: err = %v, want insufficient_privilege", err)
	}

	// A stray grant on the exception is revoked by the next db-roles.
	nava := superConfig(t).Copy()
	nava.Database = "nava"
	sconn := connect(t, nava)
	exec(t, sconn, "GRANT EXECUTE ON FUNCTION pgbouncer.get_auth(text) TO PUBLIC")
	exec(t, sconn, "GRANT USAGE ON SCHEMA pgbouncer TO nava_app")
	log, buf := newLog()
	res, err := roles.Run(context.Background(), superConfig(t), m, passwords, log)
	if err != nil {
		t.Fatal(err)
	}
	if res.Revoked != 2 || !strings.Contains(buf.String(), "revoked grant not in manifest") {
		t.Fatalf("revoked = %d, want 2\n%s", res.Revoked, buf)
	}
	if got := queryString(t, sconn, `
		SELECT coalesce(string_agg(r, ',' ORDER BY r), '') FROM (
		  SELECT 'fn:' || CASE WHEN a.grantee = 0 THEN 'PUBLIC' ELSE pg_get_userbyid(a.grantee)::text END AS r
		  FROM pg_proc p, aclexplode(coalesce(p.proacl, acldefault('f', p.proowner))) a
		  WHERE p.oid = 'pgbouncer.get_auth(text)'::regprocedure AND a.grantee <> p.proowner
		  UNION ALL
		  SELECT 'schema:' || CASE WHEN a.grantee = 0 THEN 'PUBLIC' ELSE pg_get_userbyid(a.grantee)::text END
		  FROM pg_namespace n, aclexplode(coalesce(n.nspacl, acldefault('n', n.nspowner))) a
		  WHERE n.nspname = 'pgbouncer' AND a.grantee <> n.nspowner) s`); got != "fn:pgbouncer_auth,schema:pgbouncer_auth" {
		t.Fatalf("exception grants = %s, want only pgbouncer_auth", got)
	}
	if !queryBool(t, sconn, "SELECT prosecdef FROM pg_proc WHERE oid = 'pgbouncer.get_auth(text)'::regprocedure") {
		t.Fatal("get_auth is not SECURITY DEFINER; PgBouncer could not read pg_shadow")
	}
}

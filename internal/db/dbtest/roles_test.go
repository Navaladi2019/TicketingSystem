//go:build integration

package dbtest

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/db/grants"
	"github.com/Navaladi2019/TicketingSystem/internal/db/roles"
)

// snapshotRoles is every manifest role's oid, attributes and password
// verifier, plus every membership involving one: what a no-op run must keep.
func snapshotRoles(t *testing.T) string {
	t.Helper()
	conn := connect(t, superConfig(t))
	m := manifest(t)
	return queryString(t, conn, `
		SELECT coalesce(string_agg(format('%s %s %s%s%s%s%s%s %s', a.rolname, a.oid, a.rolsuper, a.rolcreatedb,
		       a.rolcreaterole, a.rolbypassrls, a.rolcanlogin, a.rolreplication, a.rolpassword), E'\n' ORDER BY a.rolname), '')
		    || '|' || (SELECT count(*) FROM pg_auth_members am JOIN pg_roles r ON r.oid = am.roleid JOIN pg_roles u ON u.oid = am.member
		               WHERE r.rolname = ANY ($1) OR u.rolname = ANY ($1))
		    || '|' || (SELECT string_agg(datname || '=' || pg_get_userbyid(datdba), ',' ORDER BY datname) FROM pg_database WHERE datname = ANY ($2))
		FROM pg_authid a WHERE a.rolname = ANY ($1)`, m.RoleNames(), m.DatabaseNames())
}

func TestDBRolesTwiceIsNoOp(t *testing.T) {
	m := freshCluster(t)
	before := snapshotRoles(t)

	log, buf := newLog()
	res, err := roles.Run(context.Background(), superConfig(t), m, passwords, log)
	if err != nil {
		t.Fatal(err)
	}
	if res.Total() != 0 || res.Existing != len(m.Roles)+len(m.Databases) {
		t.Fatalf("second run: %+v, want no changes and %d existing\n%s", res, len(m.Roles)+len(m.Databases), buf)
	}
	if after := snapshotRoles(t); after != before {
		t.Fatalf("second run changed the cluster:\nbefore %s\nafter  %s", before, after)
	}
}

func TestDBRolesRolesHaveNoDangerousAttributes(t *testing.T) {
	m := freshCluster(t)
	conn := connect(t, superConfig(t))
	if bad := queryString(t, conn, `
		SELECT coalesce(string_agg(rolname, ','), '') FROM pg_roles
		WHERE rolname = ANY ($1) AND (rolsuper OR rolcreatedb OR rolcreaterole OR rolbypassrls)`, m.RoleNames()); bad != "" {
		t.Fatalf("roles with SUPERUSER, CREATEDB, CREATEROLE or BYPASSRLS: %s", bad)
	}
	// Every role exists and LOGIN follows the manifest.
	for _, name := range m.RoleNames() {
		want := m.Roles[name].HasAttribute("LOGIN")
		if got := queryBool(t, conn, "SELECT rolcanlogin FROM pg_roles WHERE rolname = $1", name); got != want {
			t.Errorf("role %s: rolcanlogin = %v, want %v", name, got, want)
		}
	}
}

func TestDBRolesConvergesAttributeAndPasswordDrift(t *testing.T) {
	m := freshCluster(t)
	conn := connect(t, superConfig(t))
	exec(t, conn, "ALTER ROLE nava_app SUPERUSER CREATEDB CREATEROLE BYPASSRLS REPLICATION")
	exec(t, conn, "ALTER ROLE keycloak NOLOGIN")

	newPw := func(role string) (string, bool) {
		if role == "nava_app" {
			return "rotated-pw", true
		}
		return passwords(role)
	}
	log, _ := newLog()
	res, err := roles.Run(context.Background(), superConfig(t), m, newPw, log)
	if err != nil {
		t.Fatal(err)
	}
	if res.Changed != 2 {
		t.Fatalf("changed = %d, want 2 (nava_app, keycloak)", res.Changed)
	}
	if queryBool(t, conn, `SELECT rolsuper OR rolcreatedb OR rolcreaterole OR rolbypassrls OR rolreplication OR NOT rolcanlogin
		FROM pg_roles WHERE rolname = 'nava_app'`) {
		t.Fatal("nava_app attributes not converged")
	}
	if !queryBool(t, conn, "SELECT rolcanlogin FROM pg_roles WHERE rolname = 'keycloak'") {
		t.Fatal("keycloak LOGIN not restored")
	}
	if !strings.HasPrefix(queryString(t, conn, "SELECT rolpassword FROM pg_authid WHERE rolname = 'nava_app'"), "SCRAM-SHA-256$") {
		t.Fatal("password not stored as a SCRAM verifier")
	}
	cfg := roleConfig(t, "nava_app", "postgres")
	cfg.Password = "rotated-pw"
	connect(t, cfg) // the rotated password works
}

// T02 security review: a stray membership must not survive db-roles.
func TestDBRolesRevokesStrayMemberships(t *testing.T) {
	m := freshCluster(t)
	conn := connect(t, superConfig(t))
	exec(t, conn, "GRANT pg_read_all_data TO keycloak")
	exec(t, conn, "GRANT nava_migrator TO nava_app")
	exec(t, conn, "GRANT nava_migrator TO CURRENT_USER")
	exec(t, conn, "CREATE ROLE nava_test_intruder")
	exec(t, conn, "GRANT nava_test_intruder TO temporal WITH ADMIN OPTION")

	log, buf := newLog()
	res, err := roles.Run(context.Background(), superConfig(t), m, passwords, log)
	if err != nil {
		t.Fatal(err)
	}
	if res.Revoked != 4 {
		t.Fatalf("revoked = %d, want 4\n%s", res.Revoked, buf)
	}
	if !strings.Contains(buf.String(), `"msg":"revoked role membership not in manifest","role":"pg_read_all_data","member":"keycloak"`) {
		t.Fatalf("revocation not logged:\n%s", buf)
	}
	if n := queryString(t, conn, `SELECT count(*)::text FROM pg_auth_members am
		JOIN pg_roles r ON r.oid = am.roleid JOIN pg_roles u ON u.oid = am.member
		WHERE r.rolname = ANY ($1) OR u.rolname = ANY ($1)`, m.RoleNames()); n != "0" {
		t.Fatalf("%s memberships involving manifest roles remain", n)
	}
}

func TestDBRolesConvergesDatabaseOwner(t *testing.T) {
	m := freshCluster(t)
	conn := connect(t, superConfig(t))
	exec(t, conn, "ALTER DATABASE nava OWNER TO CURRENT_USER")
	log, _ := newLog()
	if _, err := roles.Run(context.Background(), superConfig(t), m, passwords, log); err != nil {
		t.Fatal(err)
	}
	if got := queryString(t, conn, "SELECT pg_get_userbyid(datdba)::text FROM pg_database WHERE datname = 'nava'"); got != "nava_migrator" {
		t.Fatalf("nava owner = %s, want nava_migrator", got)
	}
}

func TestDBRolesRequiresPasswords(t *testing.T) {
	m := freshCluster(t)
	log, _ := newLog()
	_, err := roles.Run(context.Background(), superConfig(t), m, func(string) (string, bool) { return "", false }, log)
	if err == nil || !strings.Contains(err.Error(), "DBROLES_PASSWORD_") {
		t.Fatalf("err = %v, want a missing-password error naming the variable", err)
	}
}

func TestDBRolesRefusesNonSuperuser(t *testing.T) {
	m := freshCluster(t)
	log, _ := newLog()
	_, err := roles.Run(context.Background(), roleConfig(t, "nava_app", "postgres"), m, passwords, log)
	if err == nil || !strings.Contains(err.Error(), "bootstrap superuser") {
		t.Fatalf("err = %v, want a refusal", err)
	}
}

// T08 security review: the pg_hba.conf superuser rules match the bootstrap
// superuser by name, so db-roles refuses a cluster with a second one, and
// fails before it changes anything.
func TestDBRolesRefusesSecondSuperuser(t *testing.T) {
	m := freshCluster(t)
	conn := connect(t, superConfig(t))
	exec(t, conn, "CREATE ROLE nava_test_intruder SUPERUSER")
	exec(t, conn, "GRANT pg_read_all_data TO keycloak") // drift a passing run would revoke
	before := snapshotRoles(t)

	log, _ := newLog()
	_, err := roles.Run(context.Background(), superConfig(t), m, passwords, log)
	if err == nil || !strings.Contains(err.Error(), "nava_test_intruder") {
		t.Fatalf("db-roles with a second superuser: err = %v, want one naming nava_test_intruder", err)
	}
	if after := snapshotRoles(t); after != before {
		t.Fatalf("a failed run changed the cluster:\nbefore %s\nafter  %s", before, after)
	}

	exec(t, conn, "DROP ROLE nava_test_intruder")
	if _, err := roles.Run(context.Background(), superConfig(t), m, passwords, log); err != nil {
		t.Fatalf("db-roles after the second superuser is gone: %v", err)
	}
}

// T08 security review: PUBLIC holds nothing on the platform database between
// db-roles and migrate, while keycloak and temporal are already starting.
func TestDBRolesNewPlatformDatabaseHasNoPublicGrants(t *testing.T) {
	m := freshCluster(t) // db-roles only, no migrate
	conn := connect(t, superConfig(t))
	if got := queryString(t, conn, `
		SELECT coalesce(string_agg(a.privilege_type, ',' ORDER BY a.privilege_type), '')
		FROM pg_database d, aclexplode(coalesce(d.datacl, acldefault('d', d.datdba))) a
		WHERE d.datname = $1 AND a.grantee = 0`, m.PlatformDatabase); got != "" {
		t.Fatalf("PUBLIC privileges on %s before migrate: %s", m.PlatformDatabase, got)
	}
	for _, r := range m.RoleNames() {
		if r == m.Owner() {
			continue
		}
		if queryBool(t, conn, "SELECT has_database_privilege($1, $2, 'CONNECT') OR has_database_privilege($1, $2, 'TEMPORARY')",
			r, m.PlatformDatabase) {
			t.Errorf("%s has CONNECT or TEMPORARY on %s before migrate", r, m.PlatformDatabase)
		}
	}
}

// snapshotGetAuth is the pgbouncer schema's and get_auth's owner, body and
// ACLs: what a refused run must leave untouched.
func snapshotGetAuth(t *testing.T, m *grants.Manifest) string {
	t.Helper()
	conn := connect(t, platformSuperConfig(t, m))
	return queryString(t, conn, `
		SELECT coalesce((SELECT format('%s %s', nspowner::regrole, nspacl) FROM pg_namespace WHERE nspname = 'pgbouncer'), 'no schema')
		    || '|' || coalesce((SELECT format('%s %s %s', proowner::regrole, proacl, prosrc) FROM pg_proc
		                        WHERE oid = to_regprocedure('pgbouncer.get_auth(text)')), 'no function')`)
}

// T03 security review (TICK-107): the migrator owns the platform database,
// so it can create schema pgbouncer before db-roles does. db-roles refuses to
// put get_auth into it and changes nothing.
func TestDBRolesRefusesPgBouncerSchemaNotOwnedBySuperuser(t *testing.T) {
	m := freshCluster(t)
	sconn := connect(t, platformSuperConfig(t, m))
	exec(t, sconn, "DROP SCHEMA pgbouncer CASCADE")
	mig := connect(t, roleConfig(t, m.Owner(), m.PlatformDatabase))
	exec(t, mig, "CREATE SCHEMA pgbouncer")
	before := snapshotGetAuth(t, m)

	log, _ := newLog()
	_, err := roles.Run(context.Background(), superConfig(t), m, passwords, log)
	if err == nil || !strings.Contains(err.Error(), "schema pgbouncer") || !strings.Contains(err.Error(), `"nava_migrator"`) {
		t.Fatalf("db-roles with a migrator-owned pgbouncer schema: err = %v, want one naming the schema and nava_migrator", err)
	}
	if after := snapshotGetAuth(t, m); after != before {
		t.Fatalf("a refused run changed the schema:\nbefore %s\nafter  %s", before, after)
	}
	if !strings.Contains(before, "no function") {
		t.Fatalf("get_auth exists in the migrator's schema: %s", before)
	}
}

// T03 security review (TICK-107): get_auth owned by a non-superuser in the
// superuser-owned schema fails the run; CREATE OR REPLACE would keep that
// owner, so db-roles does not try.
func TestDBRolesRefusesGetAuthNotOwnedBySuperuser(t *testing.T) {
	m := freshCluster(t)
	sconn := connect(t, platformSuperConfig(t, m))
	exec(t, sconn, "ALTER FUNCTION pgbouncer.get_auth(text) OWNER TO nava_migrator")
	before := snapshotGetAuth(t, m)

	log, _ := newLog()
	_, err := roles.Run(context.Background(), superConfig(t), m, passwords, log)
	if err == nil || !strings.Contains(err.Error(), "function pgbouncer.get_auth(text)") || !strings.Contains(err.Error(), `"nava_migrator"`) {
		t.Fatalf("db-roles with a migrator-owned get_auth: err = %v, want one naming the function and nava_migrator", err)
	}
	if after := snapshotGetAuth(t, m); after != before {
		t.Fatalf("a refused run changed get_auth:\nbefore %s\nafter  %s", before, after)
	}

	exec(t, sconn, "ALTER FUNCTION pgbouncer.get_auth(text) OWNER TO CURRENT_USER")
	if _, err := roles.Run(context.Background(), superConfig(t), m, passwords, log); err != nil {
		t.Fatalf("db-roles after ownership is restored: %v", err)
	}
}

// platformSuperConfig is the bootstrap superuser on the platform database.
func platformSuperConfig(t *testing.T, m *grants.Manifest) *pgx.ConnConfig {
	t.Helper()
	cfg := superConfig(t).Copy()
	cfg.Database = m.PlatformDatabase
	return cfg
}

// getAuthState is get_auth's oid, owner oid and name, and whether the owner is a superuser,
// then its body, SECURITY DEFINER flag, settings and ACL.
func getAuthState(t *testing.T, conn *pgx.Conn) (identity, definition string) {
	t.Helper()
	if err := conn.QueryRow(context.Background(), `
		SELECT format('oid=%s owner=%s(%s) super=%s', p.oid, p.proowner, r.rolname, r.rolsuper),
		       format('secdef=%s config=%s acl=%s src=%s', p.prosecdef, p.proconfig, p.proacl, p.prosrc)
		FROM pg_proc p JOIN pg_roles r ON r.oid = p.proowner
		WHERE p.oid = to_regprocedure('pgbouncer.get_auth(text)')`).Scan(&identity, &definition); err != nil {
		t.Fatalf("read get_auth: %v", err)
	}
	return identity, definition
}

// TICK-144 (ADR-001 I3, CI-3): a superuser-owned get_auth whose body or
// search_path has drifted is rewritten with CREATE OR REPLACE. The canonical
// definition comes back and the function keeps its oid and its superuser
// owner, so the repair neither recreates the function nor re-owns it.
func TestDBRolesRepairsDriftedSuperuserOwnedGetAuth(t *testing.T) {
	for _, tc := range []struct{ name, drift string }{
		// The allow-list and the superuser/BYPASSRLS/REPLICATION filters gone.
		{"body", `CREATE OR REPLACE FUNCTION pgbouncer.get_auth(p_usename text)
			RETURNS TABLE (usename name, passwd text)
			LANGUAGE sql STABLE SECURITY DEFINER SET search_path = pg_catalog
			AS 'SELECT usename, passwd FROM pg_catalog.pg_shadow WHERE usename = p_usename'`},
		// A schema the caller can write to ahead of pg_catalog.
		{"search_path", "ALTER FUNCTION pgbouncer.get_auth(text) SET search_path = public, pg_catalog"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := freshCluster(t)
			sconn := connect(t, platformSuperConfig(t, m))
			wantID, wantDef := getAuthState(t, sconn)
			if !strings.HasSuffix(wantID, " super=t") {
				t.Fatalf("fresh get_auth is not superuser-owned: %s", wantID)
			}

			exec(t, sconn, tc.drift)
			if id, def := getAuthState(t, sconn); id != wantID || def == wantDef {
				t.Fatalf("drift setup: want same identity and a changed definition, got\n%s\n%s", id, def)
			}

			log, buf := newLog()
			res, err := roles.Run(context.Background(), superConfig(t), m, passwords, log)
			if err != nil {
				t.Fatalf("db-roles on a drifted get_auth: %v\n%s", err, buf)
			}
			if res.Total() != 1 || res.Changed != 1 {
				t.Fatalf("result = %+v, want exactly one change (the function)\n%s", res, buf)
			}
			if !strings.Contains(buf.String(), `"msg":"wrote PgBouncer lookup function"`) {
				t.Fatalf("rewrite not logged:\n%s", buf)
			}
			id, def := getAuthState(t, sconn)
			if id != wantID {
				t.Fatalf("get_auth identity changed:\nbefore %s\nafter  %s", wantID, id)
			}
			if def != wantDef {
				t.Fatalf("get_auth not restored:\nwant %s\ngot  %s", wantDef, def)
			}

			// Converged: the next run changes nothing.
			if res, err := roles.Run(context.Background(), superConfig(t), m, passwords, log); err != nil || res.Total() != 0 {
				t.Fatalf("run after the repair: %+v, %v; want no changes", res, err)
			}
		})
	}
}

// T08b (TICK-114): PUBLIC holds nothing on the platform database after any
// db-roles run, not only the one that creates it, so a run that died between
// CREATE DATABASE and the revoke is repaired before migrate runs.
func TestDBRolesRevokesPublicOnExistingPlatformDatabase(t *testing.T) {
	for _, tc := range []struct{ name, setup string }{
		// The first run died after CREATE DATABASE: PostgreSQL's default ACL.
		{"died after create", "CREATE DATABASE nava OWNER nava_migrator"},
		{"explicit grant", "GRANT CONNECT, TEMPORARY ON DATABASE nava TO PUBLIC"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := freshCluster(t)
			conn := connect(t, superConfig(t))
			if strings.HasPrefix(tc.setup, "CREATE DATABASE") {
				exec(t, conn, "DROP DATABASE nava WITH (FORCE)")
			}
			exec(t, conn, tc.setup)
			public := func() string {
				return queryString(t, conn, `
					SELECT coalesce(string_agg(a.privilege_type, ',' ORDER BY a.privilege_type), '')
					FROM pg_database d, aclexplode(coalesce(d.datacl, acldefault('d', d.datdba))) a
					WHERE d.datname = $1 AND a.grantee = 0`, m.PlatformDatabase)
			}
			if got := public(); got != "CONNECT,TEMPORARY" {
				t.Fatalf("setup: PUBLIC holds %q on nava, want CONNECT,TEMPORARY", got)
			}

			log, buf := newLog()
			res, err := roles.Run(context.Background(), superConfig(t), m, passwords, log)
			if err != nil {
				t.Fatalf("db-roles: %v\n%s", err, buf)
			}
			// Plus, after the drop, PUBLIC's default EXECUTE on the rebuilt get_auth.
			if want := map[bool]int{true: 2, false: 1}[strings.HasPrefix(tc.setup, "CREATE DATABASE")]; res.Revoked != want {
				t.Fatalf("revoked = %d, want %d\n%s", res.Revoked, want, buf)
			}
			if !strings.Contains(buf.String(), `"msg":"revoked PUBLIC grants on database","database":"nava","privileges":"CONNECT,TEMPORARY"`) {
				t.Fatalf("revocation not logged:\n%s", buf)
			}
			if got := public(); got != "" {
				t.Fatalf("PUBLIC holds %s on nava after db-roles, before migrate", got)
			}

			migrateAll(t, m)
			if res, err := roles.Run(context.Background(), superConfig(t), m, passwords, log); err != nil || res.Total() != 0 {
				t.Fatalf("db-roles after migrate: %+v, %v; want no changes", res, err)
			}
		})
	}
}

// T08b (TICK-114): a role made SUPERUSER while db-roles runs, after its
// first check, fails the run at its last check, which excludes no manifest
// role. The test holds db-roles on CREATE SCHEMA pgbouncer with an
// uncommitted CREATE SCHEMA of the same name, grants SUPERUSER, then lets it go.
func TestDBRolesFinalCheckRefusesSuperuserGrantedDuringRun(t *testing.T) {
	for _, tc := range []struct{ name, role, grant, undo string }{
		{"outside manifest", "nava_test_intruder", "CREATE ROLE nava_test_intruder SUPERUSER", "DROP ROLE nava_test_intruder"},
		{"manifest role", "nava_app", "ALTER ROLE nava_app SUPERUSER", "ALTER ROLE nava_app NOSUPERUSER"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			m := freshCluster(t)
			conn := connect(t, superConfig(t))
			sconn := connect(t, platformSuperConfig(t, m))
			exec(t, sconn, "DROP SCHEMA pgbouncer CASCADE")
			holder := connect(t, platformSuperConfig(t, m))
			exec(t, holder, "BEGIN")
			exec(t, holder, "CREATE SCHEMA pgbouncer")

			log, buf := newLog()
			done := make(chan error, 1)
			go func() {
				_, err := roles.Run(ctx, superConfig(t), m, passwords, log)
				done <- err
			}()
			deadline := time.Now().Add(30 * time.Second)
			for !queryBool(t, conn, `SELECT EXISTS (SELECT FROM pg_stat_activity
				WHERE wait_event_type = 'Lock' AND query LIKE 'CREATE SCHEMA%pgbouncer%' AND pid <> pg_backend_pid())`) {
				select {
				case err := <-done:
					t.Fatalf("db-roles finished before it reached CREATE SCHEMA: %v\n%s", err, buf)
				default:
				}
				if time.Now().After(deadline) {
					t.Fatalf("db-roles never blocked on CREATE SCHEMA pgbouncer\n%s", buf)
				}
				time.Sleep(20 * time.Millisecond)
			}

			exec(t, conn, tc.grant)
			t.Cleanup(func() { _, _ = conn.Exec(ctx, tc.undo) })
			exec(t, holder, "ROLLBACK")

			err := <-done
			if err == nil || !strings.Contains(err.Error(), "must be the only superuser") || !strings.Contains(err.Error(), tc.role) {
				t.Fatalf("db-roles with %s made SUPERUSER mid-run: err = %v, want one naming it\n%s", tc.role, err, buf)
			}
			if strings.Contains(buf.String(), `"msg":"db-roles done"`) {
				t.Fatalf("a failed run logged done:\n%s", buf)
			}
		})
	}
}

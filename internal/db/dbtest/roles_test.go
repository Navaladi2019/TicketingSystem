//go:build integration

package dbtest

import (
	"context"
	"strings"
	"testing"

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

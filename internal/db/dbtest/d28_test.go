//go:build integration

package dbtest

import (
	"context"
	"strings"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/db/grants"
	"github.com/Navaladi2019/TicketingSystem/internal/db/roles"
)

// d28Roles is ADR-001 A.3 (D28): one role per system actor.
func d28Roles() []string {
	return []string{
		"keycloak", "nava_app", "nava_backup", "nava_groupsync", "nava_migrator", "nava_onboard",
		"nava_purge", "nava_relay", "nava_worker", "pgbouncer_auth", "temporal",
	}
}

// serviceDatabases are the manifest databases other than the platform one.
func serviceDatabases(m *grants.Manifest) []string {
	var out []string
	for _, d := range m.DatabaseNames() {
		if d != m.PlatformDatabase {
			out = append(out, d)
		}
	}
	return out
}

// runtimeRoles are the roles that own no database.
func runtimeRoles(m *grants.Manifest) []string {
	owners := map[string]bool{}
	for _, d := range m.Databases {
		owners[d.Owner] = true
	}
	var out []string
	for _, r := range m.RoleNames() {
		if !owners[r] {
			out = append(out, r)
		}
	}
	return out
}

func TestD28EveryRoleExists(t *testing.T) {
	m := freshCluster(t)
	conn := connect(t, superConfig(t))
	want := d28Roles()
	got := queryString(t, conn, `SELECT string_agg(rolname, ',' ORDER BY rolname) FROM pg_roles WHERE rolname = ANY ($1)`, want)
	if got != strings.Join(want, ",") {
		t.Fatalf("roles present = %s, want %s", got, strings.Join(want, ","))
	}
	// REPLICATION follows the manifest: nava_backup only.
	if got := queryString(t, conn, `SELECT coalesce(string_agg(rolname, ','), '') FROM pg_roles
		WHERE rolname = ANY ($1) AND rolreplication`, m.RoleNames()); got != "nava_backup" {
		t.Fatalf("REPLICATION roles = %q, want nava_backup", got)
	}
}

// Acceptance: recursive pg_auth_members is empty for every runtime role.
// Checked both ways (member of nothing, granted to no one) and through
// pg_has_role, which follows the same chain PostgreSQL uses for privileges.
func TestD28NoRoleMembershipsRecursive(t *testing.T) {
	m := freshCluster(t)
	migrateAll(t, m)
	conn := connect(t, superConfig(t))
	got := queryString(t, conn, `
		WITH RECURSIVE up(member, roleid) AS (
		    SELECT am.member, am.roleid FROM pg_auth_members am
		    WHERE am.member IN (SELECT oid FROM pg_roles WHERE rolname = ANY ($1))
		  UNION
		    SELECT up.member, am.roleid FROM up JOIN pg_auth_members am ON am.member = up.roleid
		), down(roleid, member) AS (
		    SELECT am.roleid, am.member FROM pg_auth_members am
		    WHERE am.roleid IN (SELECT oid FROM pg_roles WHERE rolname = ANY ($1))
		  UNION
		    SELECT down.roleid, am.member FROM down JOIN pg_auth_members am ON am.roleid = down.member
		)
		SELECT coalesce(string_agg(x, ', '), '') FROM (
		    SELECT pg_get_userbyid(member) || ' member of ' || pg_get_userbyid(roleid) FROM up
		  UNION ALL
		    SELECT pg_get_userbyid(roleid) || ' granted to ' || pg_get_userbyid(member) FROM down
		) s(x)`, m.RoleNames())
	if got != "" {
		t.Fatalf("memberships: %s", got)
	}
	if got := queryString(t, conn, `
		SELECT coalesce(string_agg(r.rolname || ' has ' || g.rolname, ', '), '')
		FROM pg_roles r CROSS JOIN pg_roles g
		WHERE r.rolname = ANY ($1) AND g.oid <> r.oid AND pg_has_role(r.oid, g.oid, 'MEMBER')`, runtimeRoles(m)); got != "" {
		t.Fatalf("pg_has_role: %s", got)
	}
}

// Acceptance: no runtime role has rolbypassrls or rolsuper (nor CREATEDB or
// CREATEROLE), and a drifted one is put back.
func TestD28NoSuperuserOrBypassRLS(t *testing.T) {
	m := freshCluster(t)
	conn := connect(t, superConfig(t))
	for _, r := range runtimeRoles(m) {
		exec(t, conn, "ALTER ROLE "+r+" SUPERUSER BYPASSRLS")
	}
	log, _ := newLog()
	if _, err := roles.Run(context.Background(), superConfig(t), m, passwords, log); err != nil {
		t.Fatal(err)
	}
	if bad := queryString(t, conn, `
		SELECT coalesce(string_agg(rolname, ','), '') FROM pg_roles
		WHERE rolname = ANY ($1) AND (rolsuper OR rolbypassrls OR rolcreatedb OR rolcreaterole)`, m.RoleNames()); bad != "" {
		t.Fatalf("roles with SUPERUSER, BYPASSRLS, CREATEDB or CREATEROLE: %s", bad)
	}
}

// Acceptance: temporal has no CONNECT on nava; nava_app and every other
// runtime role have none on temporal, temporal_visibility or keycloak.
func TestD28DatabaseConnectSeparation(t *testing.T) {
	m := freshCluster(t)
	migrateAll(t, m)
	conn := connect(t, superConfig(t))

	for _, owner := range []string{"temporal", "keycloak"} {
		if queryBool(t, conn, "SELECT has_database_privilege($1, $2, 'CONNECT')", owner, m.PlatformDatabase) {
			t.Errorf("%s has CONNECT on %s", owner, m.PlatformDatabase)
		}
	}
	for _, d := range serviceDatabases(m) {
		owner := m.Databases[d].Owner
		for _, r := range m.RoleNames() {
			if r == owner {
				continue
			}
			if queryBool(t, conn, "SELECT has_database_privilege($1, $2, 'CONNECT')", r, d) {
				t.Errorf("%s has CONNECT on %s", r, d)
			}
		}
	}

	// And the server refuses the real connection (insufficient_privilege).
	for _, c := range []struct{ role, database string }{
		{"temporal", "nava"}, {"keycloak", "nava"},
		{"nava_app", "temporal"}, {"nava_app", "temporal_visibility"}, {"nava_app", "keycloak"},
		{"nava_worker", "temporal"}, {"nava_relay", "keycloak"},
		{"nava_backup", "nava"},
	} {
		if _, err := connectErr(roleConfig(t, c.role, c.database)); sqlState(err) != "42501" {
			t.Errorf("%s connecting to %s: err = %v, want SQLSTATE 42501", c.role, c.database, err)
		}
	}
	// The owners and the platform roles still get in where they belong.
	connect(t, roleConfig(t, "temporal", "temporal_visibility"))
	connect(t, roleConfig(t, "keycloak", "keycloak"))
	connect(t, roleConfig(t, "nava_worker", "nava"))
}

// A grant on a service database that the manifest does not list, PUBLIC's
// default included, does not survive db-roles, and the revoke is logged.
func TestD28DBRolesRevokesServiceDatabaseGrants(t *testing.T) {
	m := freshCluster(t)
	conn := connect(t, superConfig(t))
	exec(t, conn, "GRANT CONNECT, TEMPORARY ON DATABASE keycloak TO PUBLIC")
	exec(t, conn, "GRANT CONNECT ON DATABASE temporal TO nava_app")
	exec(t, conn, "GRANT CREATE ON DATABASE temporal_visibility TO nava_relay")

	log, buf := newLog()
	res, err := roles.Run(context.Background(), superConfig(t), m, passwords, log)
	if err != nil {
		t.Fatal(err)
	}
	if res.Revoked != 4 {
		t.Fatalf("revoked = %d, want 4\n%s", res.Revoked, buf)
	}
	if !strings.Contains(buf.String(), `"msg":"revoked database grant not in manifest","database":"temporal","grantee":"nava_app","privilege":"CONNECT"`) {
		t.Fatalf("revocation not logged:\n%s", buf)
	}
	if got := queryString(t, conn, `
		SELECT coalesce(string_agg(d.datname || ':' || a.grantee::regrole::text || ':' || a.privilege_type, ','), '')
		FROM pg_database d, aclexplode(coalesce(d.datacl, acldefault('d', d.datdba))) a
		WHERE d.datname = ANY ($1) AND a.grantee <> d.datdba`, serviceDatabases(m)); got != "" {
		// grantee 0 (PUBLIC) prints as "-".
		t.Fatalf("non-owner grants remain: %s", got)
	}
}

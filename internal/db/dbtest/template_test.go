//go:build integration

package dbtest

import (
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/db/grants"
)

// migratedCluster leaves the cluster as freshCluster followed by migrateAll
// would, but copies each manifest database from a template taken after the
// first real run in this process (TICK-153 O2), so most tests skip db-roles
// and every migration.
//
// Roles are cluster-wide and some tests change them on purpose, so a copy is
// made only when every role, membership and role setting is still exactly as
// it was when the templates were taken; otherwise it falls back to the full
// rebuild and takes new templates. The copied databases must then match the
// originals in owner, grants and settings, or the test fails. Tests of db-roles
// or of a migrator run from scratch call freshCluster instead.
func migratedCluster(t *testing.T) *grants.Manifest {
	t.Helper()
	m := manifest(t)
	conn := connect(t, superConfig(t))
	if tpl != nil {
		for _, d := range m.DatabaseNames() {
			exec(t, conn, "DROP DATABASE IF EXISTS "+ident(d)+" WITH (FORCE)")
		}
		exec(t, conn, "DROP ROLE IF EXISTS nava_test_intruder")
		if got := queryString(t, conn, rolesFingerprint); got == tpl.roles {
			for _, d := range m.DatabaseNames() {
				exec(t, conn, "CREATE DATABASE "+ident(d)+" TEMPLATE "+ident(templateName(d))+" OWNER "+ident(m.Databases[d].Owner))
				exec(t, conn, "REVOKE ALL ON DATABASE "+ident(d)+" FROM PUBLIC")
				for _, g := range tpl.grants[d] {
					stmt := "GRANT " + g.Priv + " ON DATABASE " + ident(d) + " TO " + g.Grantee
					if g.Grantable {
						stmt += " WITH GRANT OPTION"
					}
					exec(t, conn, stmt)
				}
			}
			if got := queryString(t, conn, databasesFingerprint, m.DatabaseNames()); got != tpl.databases {
				t.Fatalf("databases copied from templates differ from the migrated ones:\n got %s\nwant %s", got, tpl.databases)
			}
			return m
		}
		t.Log("migratedCluster: roles changed since the templates were taken; full rebuild")
	}

	m = freshCluster(t)
	migrateAll(t, m)
	next := &template{grants: map[string][]databaseGrant{}}
	next.roles = queryString(t, conn, rolesFingerprint)
	next.databases = queryString(t, conn, databasesFingerprint, m.DatabaseNames())
	for _, d := range m.DatabaseNames() {
		rows, err := conn.Query(t.Context(), databaseGrants, d)
		if err != nil {
			t.Fatal(err)
		}
		if next.grants[d], err = pgx.CollectRows(rows, pgx.RowToStructByPos[databaseGrant]); err != nil {
			t.Fatal(err)
		}
		// A template database must have no other session.
		exec(t, conn, "SELECT pg_catalog.pg_terminate_backend(pid) FROM pg_catalog.pg_stat_activity WHERE datname = $1 AND pid <> pg_catalog.pg_backend_pid()", d)
		name := ident(templateName(d))
		exec(t, conn, "CREATE DATABASE "+name+" TEMPLATE "+ident(d))
		// Nobody connects to it, so nothing changes it between copies.
		exec(t, conn, "REVOKE ALL ON DATABASE "+name+" FROM PUBLIC")
		exec(t, conn, "ALTER DATABASE "+name+" WITH IS_TEMPLATE true ALLOW_CONNECTIONS false")
	}
	tpl = next
	return m
}

// tpl is what migratedCluster took its templates from; nil when there are
// none. The tests in this package do not run in parallel.
var tpl *template

type template struct {
	roles, databases string
	grants           map[string][]databaseGrant
}

type databaseGrant struct {
	Grantee, Priv string // grantee quoted, or PUBLIC
	Grantable     bool
}

func templateName(database string) string { return "nava_tpl_" + database }

// dropTemplates removes the templates, whose objects the manifest roles own,
// so freshCluster can drop those roles.
func dropTemplates(t *testing.T, conn *pgx.Conn, m *grants.Manifest) {
	t.Helper()
	tpl = nil
	for _, d := range m.DatabaseNames() {
		name := templateName(d)
		if !queryBool(t, conn, "SELECT EXISTS (SELECT 1 FROM pg_catalog.pg_database WHERE datname = $1)", name) {
			continue
		}
		exec(t, conn, "ALTER DATABASE "+ident(name)+" WITH IS_TEMPLATE false")
		exec(t, conn, "DROP DATABASE "+ident(name))
	}
}

func ident(s string) string { return pgx.Identifier{s}.Sanitize() }

// rolesFingerprint is every cluster-wide fact about roles a test can change:
// each pg_authid row (attributes and password verifier), each membership and
// each role-wide setting, whole rows so a new column counts too.
const rolesFingerprint = `
	SELECT concat_ws(' | ',
	  (SELECT string_agg(x, ',' ORDER BY x) FROM (SELECT to_jsonb(r)::text AS x FROM pg_catalog.pg_authid r) s),
	  (SELECT string_agg(x, ',' ORDER BY x) FROM (SELECT to_jsonb(a)::text AS x FROM pg_catalog.pg_auth_members a) s),
	  (SELECT string_agg(x, ',' ORDER BY x) FROM (SELECT to_jsonb(rs)::text AS x FROM pg_catalog.pg_db_role_setting rs
	                                             WHERE rs.setdatabase = 0) s))`

// databaseGrants is each privilege on database $1 held by anyone but its
// owner. Every grant on a manifest database is made by its owner (db-roles
// and grants.Apply), and a copy needs them made again: CREATE DATABASE does
// not copy the ACL.
const databaseGrants = `
	SELECT CASE WHEN a.grantee = 0 THEN 'PUBLIC' ELSE pg_catalog.quote_ident(pg_catalog.pg_get_userbyid(a.grantee)) END,
	       a.privilege_type, a.is_grantable
	FROM pg_catalog.pg_database d, pg_catalog.aclexplode(coalesce(d.datacl, pg_catalog.acldefault('d', d.datdba))) a
	WHERE d.datname = $1 AND a.grantee <> d.datdba`

// databasesFingerprint is what a copy of the databases $1 could get wrong:
// owner, every ACL entry with its grantor, settings and connection options.
const databasesFingerprint = `
	SELECT coalesce(string_agg(concat_ws(':', d.datname, pg_catalog.pg_get_userbyid(d.datdba),
	       (SELECT string_agg(x, ',' ORDER BY x) FROM (
	            SELECT concat_ws('/', a.grantor, a.grantee, a.privilege_type, a.is_grantable) AS x
	            FROM pg_catalog.aclexplode(coalesce(d.datacl, pg_catalog.acldefault('d', d.datdba))) a) s),
	       d.datallowconn, d.datistemplate, d.datconnlimit, d.encoding, d.datcollate, d.datctype,
	       (SELECT count(*) FROM pg_catalog.pg_db_role_setting rs WHERE rs.setdatabase = d.oid)),
	       ',' ORDER BY d.datname), '')
	FROM pg_catalog.pg_database d WHERE d.datname = ANY ($1)`

// A copy is the migrated cluster: the same databases, grants and platform
// catalogue as a real db-roles and migrator run, and a role changed since the
// templates were taken forces the full rebuild instead of a copy.
func TestMigratedClusterMatchesFreshRun(t *testing.T) {
	m := freshCluster(t)
	migrateAll(t, m)
	super := connect(t, superConfig(t))
	wantDBs := queryString(t, super, databasesFingerprint, m.DatabaseNames())
	want := queryString(t, superNava(t), platformCatalogue)
	for _, step := range []string{"takes templates", "copies", "rebuilds after a role change"} {
		if step == "rebuilds after a role change" {
			exec(t, super, "ALTER ROLE nava_app CONNECTION LIMIT 5")
		}
		migratedCluster(t)
		if got := queryString(t, super, databasesFingerprint, m.DatabaseNames()); got != wantDBs {
			t.Fatalf("%s: databases\n got %s\nwant %s", step, got, wantDBs)
		}
		if got := queryString(t, superNava(t), platformCatalogue); got != want {
			t.Fatalf("%s: platform catalogue\n got %s\nwant %s", step, got, want)
		}
		if tpl == nil {
			t.Fatalf("%s: no templates afterwards", step)
		}
	}
	if got := queryString(t, super, "SELECT rolconnlimit::text FROM pg_roles WHERE rolname = 'nava_app'"); got != "-1" {
		t.Fatalf("nava_app connection limit = %s after the rebuild, want -1", got)
	}
}

// platformCatalogue is the platform database's objects, owners, ACLs, RLS
// flags, policies, function bodies, default privileges and applied migrations.
const platformCatalogue = `
	SELECT concat_ws(' | ',
	  (SELECT string_agg(x, ',' ORDER BY x) FROM (
	      SELECT concat_ws(':', n.nspname, c.relname, c.relkind, pg_catalog.pg_get_userbyid(c.relowner), c.relacl::text,
	             c.relrowsecurity, c.relforcerowsecurity, c.reloptions::text) AS x
	      FROM pg_catalog.pg_class c JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
	      WHERE n.nspname !~ '^pg_' AND n.nspname <> 'information_schema') s),
	  (SELECT string_agg(x, ',' ORDER BY x) FROM (
	      SELECT concat_ws(':', a.attrelid::regclass, a.attname, a.attacl::text) AS x
	      FROM pg_catalog.pg_attribute a WHERE a.attacl IS NOT NULL) s),
	  (SELECT string_agg(x, ',' ORDER BY x) FROM (
	      SELECT concat_ws(':', n.nspname, pg_catalog.pg_get_userbyid(n.nspowner), n.nspacl::text) AS x
	      FROM pg_catalog.pg_namespace n) s),
	  (SELECT string_agg(x, ',' ORDER BY x) FROM (
	      SELECT concat_ws(':', p.oid::regprocedure, pg_catalog.pg_get_userbyid(p.proowner), p.proacl::text,
	             p.prosecdef, p.proconfig::text, md5(p.prosrc)) AS x
	      FROM pg_catalog.pg_proc p JOIN pg_catalog.pg_namespace n ON n.oid = p.pronamespace
	      WHERE n.nspname !~ '^pg_' AND n.nspname <> 'information_schema') s),
	  (SELECT string_agg(x, ',' ORDER BY x) FROM (
	      SELECT concat_ws(':', pol.polrelid::regclass, pol.polname, pol.polcmd, pol.polpermissive,
	             pol.polroles::regrole[]::text, pg_catalog.pg_get_expr(pol.polqual, pol.polrelid),
	             pg_catalog.pg_get_expr(pol.polwithcheck, pol.polrelid)) AS x
	      FROM pg_catalog.pg_policy pol) s),
	  (SELECT string_agg(x, ',' ORDER BY x) FROM (
	      SELECT concat_ws(':', d.defaclrole::regrole, d.defaclnamespace::regnamespace, d.defaclobjtype, d.defaclacl::text) AS x
	      FROM pg_catalog.pg_default_acl d) s),
	  (SELECT string_agg(version_id::text, ',' ORDER BY version_id) FROM public.goose_db_version))`

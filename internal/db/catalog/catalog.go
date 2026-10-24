// Package catalog is the catalogue gate (ADR-001 Annex B CI-1, CI-3, CI-4;
// PLAN-001 T10). It reads the platform database's catalogue and reports every
// way it departs from the isolation invariants:
//
//   - CI-1: every tenant table is under ENABLE + FORCE ROW LEVEL SECURITY and
//     carries its tenant key; every policy that does not test app.tenant_id is
//     on the A.3 allow-list, FOR one command and TO named roles; no
//     SECURITY DEFINER function outside db/grants.yaml's infrastructure
//     exceptions; every view is security_invoker; no materialized view.
//   - CI-3: no manifest role is a member of any role, or has one as member,
//     recursively; no runtime role is a member of any platform object's owner;
//     datdba is the manifest owner; PUBLIC holds nothing on any platform
//     object or default privilege.
//   - CI-4: has_table_privilege, has_column_privilege, has_function_privilege
//     (plus the sequence, schema and database forms) equal db/grants.yaml
//     exactly for every role but the owner.
//
// Tenant tables are discovered fail-closed (PLAN-001 D-P16): a platform
// schema is any schema not named pg_* or information_schema, and every table
// in one is a tenant table unless db/global-tables.yaml lists it.
//
// The migrator runs Check after every run (R2) and fails if it reports
// anything; the integration suite runs it on every PR.
package catalog

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/db/grants"
)

// Querier is a connection or a transaction.
type Querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// versionTable is goose's bookkeeping table. The migrator owns it, no other
// role holds any privilege on it (CI-4 checks that), and FORCE RLS would lock
// the migrator out of it, so it is the one table that is neither a tenant
// table nor listed in db/global-tables.yaml.
const versionTable = "public.goose_db_version"

// platformSchema is the SQL test for a platform schema on column col:
// PostgreSQL refuses to create a schema named pg_*, so nothing escapes it.
func platformSchema(col string) string {
	return "(" + col + " !~ '^pg_' AND " + col + " <> 'information_schema')"
}

// funcSig is the SQL for the manifest name of function p in schema n,
// "schema.name(argtypes)". It is built from the catalogue rather than
// resolved with to_regprocedure, which needs USAGE on the schema, and the
// migrator has none on the infrastructure exceptions' schemas. A name that
// does not match fails closed: the function is reported, never skipped.
func funcSig(n, p string) string {
	return "(pg_catalog.quote_ident(" + n + ".nspname) || '.' || pg_catalog.quote_ident(" + p + ".proname) || '(' || " +
		"pg_catalog.replace(pg_catalog.oidvectortypes(" + p + ".proargtypes), ', ', ',') || ')')"
}

// relName is the SQL for "schema.table" of relation c in schema n.
func relName(n, c string) string { return "(" + n + ".nspname || '.' || " + c + ".relname)" }

// tenantKey is the column a table's rows are scoped by.
func tenantKey(schema, table string) string {
	if schema == "public" && table == "tenants" {
		return "id"
	}
	return "tenant_id"
}

// Check runs CI-1, CI-3 and CI-4 against the database q is connected to and
// returns every problem found, sorted, each prefixed with its check. An error
// means the gate could not run, not that it failed.
func Check(ctx context.Context, q Querier, m *grants.Manifest, g *GlobalTables) ([]string, error) {
	var db string
	if err := scanOne(ctx, q, "SELECT current_database()::text", nil, &db); err != nil {
		return nil, err
	}
	if db != m.PlatformDatabase {
		return nil, fmt.Errorf("catalogue gate: connected to %q, the platform database is %q", db, m.PlatformDatabase)
	}
	var problems []string
	for _, check := range []func(context.Context, Querier, *grants.Manifest, *GlobalTables) ([]string, error){
		ci1Relations, ci1Policies, ci1SecurityDefiner,
		ci3Memberships, ci3Owners, ci3Public,
		ci4Privileges,
	} {
		p, err := check(ctx, q, m, g)
		if err != nil {
			return nil, fmt.Errorf("catalogue gate: %w", err)
		}
		problems = append(problems, p...)
	}
	slices.Sort(problems)
	return slices.Compact(problems), nil
}

// ci1Relations: tenant tables under RLS + FORCE with their tenant key, views
// security_invoker, no materialized views.
func ci1Relations(ctx context.Context, q Querier, _ *grants.Manifest, g *GlobalTables) ([]string, error) {
	rows, err := q.Query(ctx, `
		SELECT n.nspname::text, c.relname::text, c.relkind::text, c.relrowsecurity, c.relforcerowsecurity,
		       EXISTS (SELECT 1 FROM pg_catalog.pg_attribute a
		               WHERE a.attrelid = c.oid AND a.attnum > 0 AND NOT a.attisdropped AND a.attnotnull
		                 AND a.atttypid = 'pg_catalog.uuid'::pg_catalog.regtype
		                 AND a.attname = CASE WHEN n.nspname = 'public' AND c.relname = 'tenants' THEN 'id' ELSE 'tenant_id' END),
		       coalesce((SELECT o.option_value::boolean FROM pg_catalog.pg_options_to_table(c.reloptions) o
		                 WHERE o.option_name = 'security_invoker'), false)
		FROM pg_catalog.pg_class c JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
		WHERE `+platformSchema("n.nspname")+` AND c.relkind IN ('r', 'p', 'f', 'v', 'm')
		ORDER BY 1, 2`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	global := map[string]bool{}
	for _, name := range g.Names() {
		global[name] = true
	}
	seen := map[string]bool{}
	var out []string
	for rows.Next() {
		var schema, table, kind string
		var rls, force, keyed, invoker bool
		if err := rows.Scan(&schema, &table, &kind, &rls, &force, &keyed, &invoker); err != nil {
			return nil, err
		}
		name := schema + "." + table
		seen[name] = true
		switch kind {
		case "m":
			out = append(out, "CI-1: materialized view "+name+" exists; a materialized view cannot carry RLS")
		case "v":
			if !invoker {
				out = append(out, "CI-1: view "+name+" is not security_invoker")
			}
		default:
			if name == versionTable || global[name] {
				continue
			}
			if !rls {
				out = append(out, "CI-1: tenant table "+name+" does not ENABLE ROW LEVEL SECURITY")
			}
			if !force {
				out = append(out, "CI-1: tenant table "+name+" does not FORCE ROW LEVEL SECURITY")
			}
			if !keyed {
				out = append(out, fmt.Sprintf("CI-1: table %s has no tenant key (%s uuid NOT NULL) and is not in db/global-tables.yaml",
					name, tenantKey(schema, table)))
			}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for name := range global {
		if !seen[name] {
			out = append(out, "CI-1: db/global-tables.yaml lists "+name+", which does not exist")
		}
	}
	return out, nil
}

// ci1Policies is the T09 allow-list guard.
func ci1Policies(ctx context.Context, q Querier, _ *grants.Manifest, _ *GlobalTables) ([]string, error) {
	problems, _, err := PolicyProblems(ctx, q)
	for i, p := range problems {
		problems[i] = "CI-1: policy " + p
	}
	return problems, err
}

// ci1SecurityDefiner: no SECURITY DEFINER function but the manifest's
// infrastructure exceptions.
func ci1SecurityDefiner(ctx context.Context, q Querier, m *grants.Manifest, _ *GlobalTables) ([]string, error) {
	var allowed []string
	for _, ex := range m.InfrastructureExceptions {
		if ex.SecurityDefiner {
			fn, err := grants.NormalizeFunction(ex.Function)
			if err != nil {
				return nil, err
			}
			allowed = append(allowed, fn)
		}
	}
	return strs(ctx, q, `
		SELECT 'CI-1: function ' || p.oid::pg_catalog.regprocedure::text || ' is SECURITY DEFINER'
		FROM pg_catalog.pg_proc p JOIN pg_catalog.pg_namespace n ON n.oid = p.pronamespace
		WHERE `+platformSchema("n.nspname")+` AND p.prosecdef AND `+funcSig("n", "p")+` <> ALL ($1::text[])`, allowed)
}

// ci3Memberships: recursive pg_auth_members is empty for every manifest role,
// both as member and as granted role, so no one is a member of nava_migrator.
func ci3Memberships(ctx context.Context, q Querier, m *grants.Manifest, _ *GlobalTables) ([]string, error) {
	return strs(ctx, q, `
		WITH RECURSIVE up(member, roleid) AS (
		    SELECT am.member, am.roleid FROM pg_catalog.pg_auth_members am
		    WHERE am.member IN (SELECT oid FROM pg_catalog.pg_roles WHERE rolname = ANY ($1))
		  UNION
		    SELECT up.member, am.roleid FROM up JOIN pg_catalog.pg_auth_members am ON am.member = up.roleid
		), down(roleid, member) AS (
		    SELECT am.roleid, am.member FROM pg_catalog.pg_auth_members am
		    WHERE am.roleid IN (SELECT oid FROM pg_catalog.pg_roles WHERE rolname = ANY ($1))
		  UNION
		    SELECT down.roleid, am.member FROM down JOIN pg_catalog.pg_auth_members am ON am.roleid = down.member
		)
		SELECT 'CI-3: ' || pg_catalog.pg_get_userbyid(member) || ' is a member of ' || pg_catalog.pg_get_userbyid(roleid) FROM up
		UNION
		SELECT 'CI-3: ' || pg_catalog.pg_get_userbyid(member) || ' is a member of ' || pg_catalog.pg_get_userbyid(roleid) FROM down`,
		m.RoleNames())
}

// ci3Owners: datdba is the manifest owner, and no runtime role is a member
// (pg_has_role, which also covers being the owner) of the owner of the
// database or of any platform schema, relation, function or type.
func ci3Owners(ctx context.Context, q Querier, m *grants.Manifest, _ *GlobalTables) ([]string, error) {
	var datdba string
	if err := scanOne(ctx, q, `SELECT pg_catalog.pg_get_userbyid(datdba)::text FROM pg_catalog.pg_database
		WHERE datname = current_database()`, nil, &datdba); err != nil {
		return nil, err
	}
	var out []string
	if datdba != m.Owner() {
		out = append(out, fmt.Sprintf("CI-3: datdba of %s is %s, want %s", m.PlatformDatabase, datdba, m.Owner()))
	}
	more, err := strs(ctx, q, `
		WITH owners(obj, owner) AS (
		    SELECT 'database ' || d.datname, d.datdba FROM pg_catalog.pg_database d WHERE d.datname = current_database()
		  UNION ALL
		    SELECT 'schema ' || n.nspname, n.nspowner FROM pg_catalog.pg_namespace n WHERE `+platformSchema("n.nspname")+`
		  UNION ALL
		    SELECT 'relation ' || c.oid::pg_catalog.regclass::text, c.relowner
		    FROM pg_catalog.pg_class c JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace WHERE `+platformSchema("n.nspname")+`
		  UNION ALL
		    SELECT 'function ' || p.oid::pg_catalog.regprocedure::text, p.proowner
		    FROM pg_catalog.pg_proc p JOIN pg_catalog.pg_namespace n ON n.oid = p.pronamespace WHERE `+platformSchema("n.nspname")+`
		  UNION ALL
		    SELECT 'type ' || t.oid::pg_catalog.regtype::text, t.typowner
		    FROM pg_catalog.pg_type t JOIN pg_catalog.pg_namespace n ON n.oid = t.typnamespace WHERE `+platformSchema("n.nspname")+`
		)
		SELECT 'CI-3: ' || r.rolname || ' is a member of ' || pg_catalog.pg_get_userbyid(o.owner) || ', owner of ' || o.obj
		FROM owners o CROSS JOIN pg_catalog.pg_roles r
		WHERE r.rolname = ANY ($1) AND pg_catalog.pg_has_role(r.oid, o.owner, 'MEMBER')`, runtimeRoles(m))
	return append(out, more...), err
}

// ci3Public: aclexplode(coalesce(acl, acldefault(type, owner))) has no PUBLIC
// grantee on the platform database or any object in a platform schema, and
// no default privilege gives PUBLIC anything. Types count as platform
// objects; a table's row type and an array type take their privileges from
// the table and the element type, so they are skipped. The built-in function
// default (EXECUTE to PUBLIC) counts unless the owner has revoked it, as
// migration 00001 does.
func ci3Public(ctx context.Context, q Querier, m *grants.Manifest, _ *GlobalTables) ([]string, error) {
	ps := platformSchema("n.nspname")
	return strs(ctx, q, `
		WITH acl(obj, priv) AS (
		    SELECT 'database ' || d.datname, a.privilege_type
		    FROM pg_catalog.pg_database d,
		         LATERAL pg_catalog.aclexplode(coalesce(d.datacl, pg_catalog.acldefault('d', d.datdba))) a
		    WHERE d.datname = current_database() AND a.grantee = 0
		  UNION ALL
		    SELECT 'schema ' || n.nspname, a.privilege_type
		    FROM pg_catalog.pg_namespace n,
		         LATERAL pg_catalog.aclexplode(coalesce(n.nspacl, pg_catalog.acldefault('n', n.nspowner))) a
		    WHERE `+ps+` AND a.grantee = 0
		  UNION ALL
		    SELECT 'relation ' || c.oid::pg_catalog.regclass::text, a.privilege_type
		    FROM pg_catalog.pg_class c JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace,
		         LATERAL pg_catalog.aclexplode(coalesce(c.relacl,
		             pg_catalog.acldefault(CASE WHEN c.relkind = 'S' THEN 's' ELSE 'r' END::"char", c.relowner))) a
		    WHERE `+ps+` AND c.relkind IN ('r', 'p', 'v', 'm', 'f', 'S') AND a.grantee = 0
		  UNION ALL
		    SELECT 'column ' || c.oid::pg_catalog.regclass::text || '.' || at.attname, a.privilege_type
		    FROM pg_catalog.pg_attribute at JOIN pg_catalog.pg_class c ON c.oid = at.attrelid
		         JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace,
		         LATERAL pg_catalog.aclexplode(at.attacl) a
		    WHERE `+ps+` AND at.attnum > 0 AND NOT at.attisdropped AND at.attacl IS NOT NULL AND a.grantee = 0
		  UNION ALL
		    SELECT 'function ' || p.oid::pg_catalog.regprocedure::text, a.privilege_type
		    FROM pg_catalog.pg_proc p JOIN pg_catalog.pg_namespace n ON n.oid = p.pronamespace,
		         LATERAL pg_catalog.aclexplode(coalesce(p.proacl, pg_catalog.acldefault('f', p.proowner))) a
		    WHERE `+ps+` AND a.grantee = 0
		  UNION ALL
		    SELECT 'type ' || t.oid::pg_catalog.regtype::text, a.privilege_type
		    FROM pg_catalog.pg_type t JOIN pg_catalog.pg_namespace n ON n.oid = t.typnamespace
		         LEFT JOIN pg_catalog.pg_class c ON c.oid = t.typrelid,
		         LATERAL pg_catalog.aclexplode(coalesce(t.typacl, pg_catalog.acldefault('T', t.typowner))) a
		    WHERE `+ps+` AND a.grantee = 0
		      AND (t.typrelid = 0 OR c.relkind = 'c')
		      AND NOT EXISTS (SELECT 1 FROM pg_catalog.pg_type e WHERE e.typarray = t.oid)
		  UNION ALL
		    SELECT 'default privilege of ' || pg_catalog.pg_get_userbyid(da.defaclrole) || ' on '
		           || CASE da.defaclobjtype WHEN 'r' THEN 'tables' WHEN 'S' THEN 'sequences' WHEN 'f' THEN 'functions'
		                                    WHEN 'T' THEN 'types' WHEN 'n' THEN 'schemas' ELSE da.defaclobjtype::text END
		           || coalesce(' in schema ' || n.nspname, ''), a.privilege_type
		    FROM pg_catalog.pg_default_acl da LEFT JOIN pg_catalog.pg_namespace n ON n.oid = da.defaclnamespace,
		         LATERAL pg_catalog.aclexplode(da.defaclacl) a
		    WHERE a.grantee = 0
		  UNION ALL
		    SELECT 'built-in default privilege of ' || r.rolname || ' on functions', a.privilege_type
		    FROM pg_catalog.pg_roles r, LATERAL pg_catalog.aclexplode(pg_catalog.acldefault('f', r.oid)) a
		    WHERE r.rolname = $1 AND a.grantee = 0
		      AND NOT EXISTS (SELECT 1 FROM pg_catalog.pg_default_acl da
		                      WHERE da.defaclrole = r.oid AND da.defaclnamespace = 0 AND da.defaclobjtype = 'f')
		)
		SELECT 'CI-3: PUBLIC holds ' || priv || ' on ' || obj FROM acl`, m.Owner())
}

// runtimeRoles are every manifest role but the platform database's owner.
func runtimeRoles(m *grants.Manifest) []string {
	var out []string
	for _, r := range m.RoleNames() {
		if r != m.Owner() {
			out = append(out, r)
		}
	}
	return out
}

func strs(ctx context.Context, q Querier, sql string, args ...any) ([]string, error) {
	rows, err := q.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}

func scanOne(ctx context.Context, q Querier, sql string, args []any, dst ...any) error {
	rows, err := q.Query(ctx, sql, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return err
		}
		return errors.New("no row: " + strings.TrimSpace(sql))
	}
	if err := rows.Scan(dst...); err != nil {
		return err
	}
	rows.Close()
	return rows.Err()
}

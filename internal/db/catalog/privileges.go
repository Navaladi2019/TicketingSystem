package catalog

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/Navaladi2019/TicketingSystem/internal/db/grants"
)

// effective is one has_*_privilege answer: whether role holds priv on obj,
// and whether it holds it WITH GRANT OPTION (never expected).
type effective struct {
	role, obj, priv string
	has, grantable  bool
}

// ci4Privileges compares the effective privileges of every role but the
// owner, on the platform database and every object in a platform schema, with
// db/grants.yaml. The has_*_privilege functions answer for what the role can
// really do, so a grant through PUBLIC or a membership counts too.
func ci4Privileges(ctx context.Context, q Querier, m *grants.Manifest, _ *GlobalTables) ([]string, error) {
	// The privileges CI-4 asks has_*_privilege about, per object kind.
	var (
		tablePrivileges    = []string{"SELECT", "INSERT", "UPDATE", "DELETE", "TRUNCATE", "REFERENCES", "TRIGGER"}
		columnPrivileges   = []string{"SELECT", "INSERT", "UPDATE", "REFERENCES"}
		sequencePrivileges = []string{"USAGE", "SELECT", "UPDATE"}
		schemaPrivileges   = []string{"USAGE", "CREATE"}
		databasePrivileges = []string{"CONNECT", "CREATE", "TEMPORARY"}
	)
	roles := runtimeRoles(m)
	var out []string

	present, err := strs(ctx, q, "SELECT rolname::text FROM pg_catalog.pg_roles WHERE rolname = ANY ($1)", roles)
	if err != nil {
		return nil, err
	}
	for _, r := range roles {
		if !slices.Contains(present, r) {
			out = append(out, "CI-4: role "+r+" of db/grants.yaml does not exist")
		}
	}

	// want[role][object][privilege], objects named as the queries below name them.
	want := map[string]map[string]map[string]bool{}
	set := func(role, obj, priv string) {
		if want[role] == nil {
			want[role] = map[string]map[string]bool{}
		}
		if want[role][obj] == nil {
			want[role][obj] = map[string]bool{}
		}
		want[role][obj][priv] = true
	}
	tables := map[string]string{} // schema.table -> a role that lists it
	var funcs []string
	for _, role := range roles {
		r := m.Roles[role]
		for _, p := range r.Database {
			set(role, "database "+m.PlatformDatabase, p)
		}
		for s, privs := range r.Schemas {
			for _, p := range privs {
				set(role, "schema "+s, p)
			}
		}
		for t, privs := range r.Tables {
			schema, table, err := grants.SplitTable(t)
			if err != nil {
				return nil, err
			}
			name := schema + "." + table
			tables[name] = role
			for p, cols := range privs {
				if len(cols) == 0 {
					set(role, "table "+name, p)
					continue
				}
				for _, c := range cols {
					set(role, "column "+name+"."+c, p)
				}
			}
		}
		for f, privs := range r.Functions {
			fn, err := grants.NormalizeFunction(f)
			if err != nil {
				return nil, err
			}
			funcs = append(funcs, fn)
			for _, p := range privs {
				set(role, "function "+fn, p)
			}
		}
	}

	// Every manifest object must exist, or its grants mean nothing.
	var names []string
	for t := range tables {
		names = append(names, t)
	}
	missing, err := strs(ctx, q, `
		SELECT 'CI-4: db/grants.yaml grants on ' || x || ', which does not exist'
		FROM unnest($1::text[]) x
		WHERE NOT EXISTS (SELECT 1 FROM pg_catalog.pg_class c JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
		                  WHERE `+relName("n", "c")+` = x)
		UNION ALL
		SELECT 'CI-4: db/grants.yaml grants on function ' || x || ', which does not exist'
		FROM unnest($2::text[]) x
		WHERE NOT EXISTS (SELECT 1 FROM pg_catalog.pg_proc p JOIN pg_catalog.pg_namespace n ON n.oid = p.pronamespace
		                  WHERE `+funcSig("n", "p")+` = x)`, names, funcs)
	if err != nil {
		return nil, err
	}
	out = append(out, missing...)

	// A function is named by its manifest signature (funcSig), not by
	// regprocedure, whose output depends on search_path.
	ps := platformSchema("n.nspname")
	rows, err := q.Query(ctx, `
		SELECT r.rolname::text, 'database ' || d.datname, p.priv,
		       pg_catalog.has_database_privilege(r.oid, d.oid, p.priv),
		       pg_catalog.has_database_privilege(r.oid, d.oid, p.priv || ' WITH GRANT OPTION')
		FROM pg_catalog.pg_roles r, pg_catalog.pg_database d, unnest($2::text[]) p(priv)
		WHERE r.rolname = ANY ($1) AND d.datname = current_database()
	  UNION ALL
		SELECT r.rolname::text, 'schema ' || n.nspname, p.priv,
		       pg_catalog.has_schema_privilege(r.oid, n.oid, p.priv),
		       pg_catalog.has_schema_privilege(r.oid, n.oid, p.priv || ' WITH GRANT OPTION')
		FROM pg_catalog.pg_roles r, pg_catalog.pg_namespace n, unnest($3::text[]) p(priv)
		WHERE r.rolname = ANY ($1) AND `+ps+`
	  UNION ALL
		SELECT r.rolname::text, 'table ' || n.nspname || '.' || c.relname, p.priv,
		       pg_catalog.has_table_privilege(r.oid, c.oid, p.priv),
		       pg_catalog.has_table_privilege(r.oid, c.oid, p.priv || ' WITH GRANT OPTION')
		FROM pg_catalog.pg_roles r, pg_catalog.pg_class c JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace,
		     unnest($4::text[]) p(priv)
		WHERE r.rolname = ANY ($1) AND `+ps+` AND c.relkind IN ('r', 'p', 'v', 'm', 'f')
	  UNION ALL
		SELECT r.rolname::text, 'table ' || n.nspname || '.' || c.relname, p.priv,
		       pg_catalog.has_sequence_privilege(r.oid, c.oid, p.priv),
		       pg_catalog.has_sequence_privilege(r.oid, c.oid, p.priv || ' WITH GRANT OPTION')
		FROM pg_catalog.pg_roles r, pg_catalog.pg_class c JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace,
		     unnest($5::text[]) p(priv)
		WHERE r.rolname = ANY ($1) AND `+ps+` AND c.relkind = 'S'
	  UNION ALL
		SELECT r.rolname::text, 'column ' || n.nspname || '.' || c.relname || '.' || a.attname, p.priv,
		       pg_catalog.has_column_privilege(r.oid, c.oid, a.attnum, p.priv),
		       pg_catalog.has_column_privilege(r.oid, c.oid, a.attnum, p.priv || ' WITH GRANT OPTION')
		FROM pg_catalog.pg_roles r, pg_catalog.pg_attribute a JOIN pg_catalog.pg_class c ON c.oid = a.attrelid
		     JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace, unnest($6::text[]) p(priv)
		WHERE r.rolname = ANY ($1) AND `+ps+` AND c.relkind IN ('r', 'p', 'v', 'm', 'f')
		  AND a.attnum > 0 AND NOT a.attisdropped
	  UNION ALL
		SELECT r.rolname::text, 'function ' || `+funcSig("n", "f")+`, 'EXECUTE',
		       pg_catalog.has_function_privilege(r.oid, f.oid, 'EXECUTE'),
		       pg_catalog.has_function_privilege(r.oid, f.oid, 'EXECUTE WITH GRANT OPTION')
		FROM pg_catalog.pg_roles r, pg_catalog.pg_proc f JOIN pg_catalog.pg_namespace n ON n.oid = f.pronamespace
		WHERE r.rolname = ANY ($1) AND `+ps,
		roles, databasePrivileges, schemaPrivileges, tablePrivileges, sequencePrivileges, columnPrivileges)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var got []effective
	for rows.Next() {
		var e effective
		if err := rows.Scan(&e.role, &e.obj, &e.priv, &e.has, &e.grantable); err != nil {
			return nil, err
		}
		got = append(got, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// A table-level grant gives every column its privilege, so a column's
	// expected answer is the table's or its own. A table-level mismatch is
	// reported once, not again for each of its columns.
	tableOff := map[string]bool{}
	for _, e := range got {
		if strings.HasPrefix(e.obj, "table ") {
			if exp := want[e.role][e.obj][e.priv]; e.has != exp {
				tableOff[e.role+"\x00"+e.obj+"\x00"+e.priv] = true
			}
		}
	}
	for _, e := range got {
		exp := want[e.role][e.obj][e.priv]
		if col, ok := strings.CutPrefix(e.obj, "column "); ok {
			table := "table " + col[:strings.LastIndexByte(col, '.')]
			if tableOff[e.role+"\x00"+table+"\x00"+e.priv] {
				continue
			}
			exp = exp || want[e.role][table][e.priv]
		}
		switch {
		case e.has && !exp:
			out = append(out, fmt.Sprintf("CI-4: %s has %s on %s, which db/grants.yaml does not list", e.role, e.priv, e.obj))
		case !e.has && exp:
			out = append(out, fmt.Sprintf("CI-4: %s lacks %s on %s, which db/grants.yaml lists", e.role, e.priv, e.obj))
		}
		if e.grantable {
			out = append(out, fmt.Sprintf("CI-4: %s holds %s on %s WITH GRANT OPTION", e.role, e.priv, e.obj))
		}
	}
	return out, nil
}

package grants

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5"
)

// Grant is one privilege held by one grantee on one object.
type Grant struct {
	Kind      string // database, schema, table, column, function, default
	Schema    string // table, column and default (empty = the global default)
	Name      string // database, schema, table, function signature, default object type
	Column    string
	Grantee   string // role name, or PUBLIC
	Privilege string
	Grantable bool // WITH GRANT OPTION; never expected
	Sequence  bool // a table-kind object that is a sequence
}

func (g Grant) key() string {
	return strings.Join([]string{g.Kind, g.Schema, g.Name, g.Column, g.Grantee, g.Privilege}, "\x00")
}

// String is the grant's log form, e.g. "SELECT (id) ON public.outbox TO nava_relay".
func (g Grant) String() string {
	obj := g.Name
	switch g.Kind {
	case "table":
		obj = g.Schema + "." + g.Name
	case "column":
		obj = fmt.Sprintf("(%s) ON %s.%s", g.Column, g.Schema, g.Name)
	case "default":
		w, _ := defaultObjType(g.Name)
		obj = "DEFAULT " + w
		if g.Schema != "" {
			obj += " IN SCHEMA " + g.Schema
		}
	}
	s := fmt.Sprintf("%s %s %s TO %s", g.Privilege, g.Kind, obj, g.Grantee)
	if g.Grantable {
		s += " WITH GRANT OPTION"
	}
	return s
}

// Result lists what one Apply changed, in the log form of Grant.String.
type Result struct {
	Granted []string
	Revoked []string
}

// Changed reports whether Apply changed anything.
func (r Result) Changed() bool { return len(r.Granted) > 0 || len(r.Revoked) > 0 }

// applyLockKey serialises concurrent appliers (pg_advisory_xact_lock).
const applyLockKey = "nava.grants.apply"

var privRe = regexp.MustCompile(`^[A-Z]+$`)

// defaultObjType maps pg_default_acl.defaclobjtype to the ALTER DEFAULT
// PRIVILEGES object word.
func defaultObjType(c string) (string, bool) {
	w, ok := map[string]string{"r": "TABLES", "S": "SEQUENCES", "f": "FUNCTIONS", "T": "TYPES", "n": "SCHEMAS"}[c]
	return w, ok
}

// Apply makes the grants on every object the connected role owns in the
// platform database equal the manifest: it revokes, and logs, every grant the
// manifest does not list (PUBLIC and default privileges included), then
// grants what is missing. It runs as the owner (the migrator), after the
// migrations, in one transaction. Objects of infrastructure exceptions are
// owned by the bootstrap superuser and left to db-roles. Running it again
// changes nothing.
func Apply(ctx context.Context, conn *pgx.Conn, m *Manifest, log *slog.Logger) (Result, error) {
	var res Result
	tx, err := conn.Begin(ctx)
	if err != nil {
		return res, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// search_path '' makes regprocedure print every function schema-qualified.
	if _, err := tx.Exec(ctx, "SET LOCAL search_path = ''"); err != nil {
		return res, err
	}
	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtext($1))", applyLockKey); err != nil {
		return res, err
	}
	if err := checkOwner(ctx, tx, m); err != nil {
		return res, err
	}
	expected, err := resolveExpected(ctx, tx, m)
	if err != nil {
		return res, err
	}
	want := map[string]bool{}
	for _, g := range expected {
		want[g.key()] = true
	}

	actual, err := readActual(ctx, tx)
	if err != nil {
		return res, err
	}
	owner := m.Owner()
	for _, g := range actual {
		listed := want[g.key()]
		if listed && !g.Grantable {
			continue
		}
		// A listed grant held WITH GRANT OPTION keeps the grant, loses the option.
		stmt, err := statement("REVOKE", g, owner, listed)
		if err != nil {
			return res, err
		}
		if _, err := tx.Exec(ctx, stmt); err != nil {
			return res, fmt.Errorf("revoke %s: %w", g, err)
		}
		log.WarnContext(ctx, "revoked grant not in manifest", slog.String("grant", g.String()))
		res.Revoked = append(res.Revoked, g.String())
	}

	// Revoking a table privilege also drops its column privileges, and a
	// revoked grant option cascades, so read again before granting.
	actual, err = readActual(ctx, tx)
	if err != nil {
		return res, err
	}
	have := map[string]bool{}
	for _, g := range actual {
		if !want[g.key()] || g.Grantable {
			return res, fmt.Errorf("grant %s survived its revoke; its grantor is not %s", g, owner)
		}
		have[g.key()] = true
	}
	for _, g := range expected {
		if have[g.key()] {
			continue
		}
		stmt, err := statement("GRANT", g, owner, false)
		if err != nil {
			return res, err
		}
		if _, err := tx.Exec(ctx, stmt); err != nil {
			return res, fmt.Errorf("grant %s: %w", g, err)
		}
		log.InfoContext(ctx, "granted", slog.String("grant", g.String()))
		res.Granted = append(res.Granted, g.String())
	}
	return res, tx.Commit(ctx)
}

// checkOwner refuses to run anywhere but the platform database, as anyone
// but its owner, or as a superuser (whose grants would carry another grantor).
func checkOwner(ctx context.Context, tx pgx.Tx, m *Manifest) error {
	var db, user string
	var owns, super bool
	err := tx.QueryRow(ctx, `
		SELECT current_database()::text, current_user::text,
		       d.datdba = r.oid, r.rolsuper
		FROM pg_catalog.pg_database d, pg_catalog.pg_roles r
		WHERE d.datname = current_database() AND r.rolname = current_user`).Scan(&db, &user, &owns, &super)
	if err != nil {
		return err
	}
	switch {
	case db != m.PlatformDatabase:
		return fmt.Errorf("connected to database %q, manifest platform database is %q", db, m.PlatformDatabase)
	case user != m.Owner():
		return fmt.Errorf("connected as %q; grants are applied by the owner %q", user, m.Owner())
	case !owns:
		return fmt.Errorf("%q is not the owner (datdba) of %q; run db-roles first", user, db)
	case super:
		return fmt.Errorf("%q is a superuser", user)
	}
	return nil
}

// resolveExpected turns the manifest into grants on catalogue objects,
// checking each object exists and is owned by the connected role.
func resolveExpected(ctx context.Context, tx pgx.Tx, m *Manifest) ([]Grant, error) {
	exSchemas, exFuncs := map[string]bool{}, map[string]bool{}
	for _, ex := range m.InfrastructureExceptions {
		exSchemas[ex.Schema] = true
		fn, _ := normalizeFunction(ex.Function)
		exFuncs[fn] = true
	}
	var out []Grant
	for _, role := range m.RoleNames() {
		r := m.Roles[role]
		for _, p := range r.Database {
			out = append(out, Grant{Kind: "database", Name: m.PlatformDatabase, Grantee: role, Privilege: p})
		}
		for _, s := range sortedKeys(r.Schemas) {
			if exSchemas[s] {
				continue
			}
			var owned bool
			err := tx.QueryRow(ctx, `
				SELECT n.nspowner = (SELECT oid FROM pg_catalog.pg_roles WHERE rolname = current_user)
				    OR n.nspowner = 'pg_database_owner'::pg_catalog.regrole
				FROM pg_catalog.pg_namespace n WHERE n.nspname = $1`, s).Scan(&owned)
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, fmt.Errorf("role %s: schema %s does not exist", role, s)
			} else if err != nil {
				return nil, err
			}
			if !owned {
				return nil, fmt.Errorf("role %s: schema %s is not owned by %s", role, s, m.Owner())
			}
			for _, p := range r.Schemas[s] {
				out = append(out, Grant{Kind: "schema", Name: s, Grantee: role, Privilege: p})
			}
		}
		for _, t := range sortedKeys(r.Tables) {
			schema, table, _ := splitTable(t)
			var relkind string
			var owned bool
			err := tx.QueryRow(ctx, `
				SELECT c.relkind::text, c.relowner = (SELECT oid FROM pg_catalog.pg_roles WHERE rolname = current_user)
				FROM pg_catalog.pg_class c JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
				WHERE n.nspname = $1 AND c.relname = $2`, schema, table).Scan(&relkind, &owned)
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, fmt.Errorf("role %s: table %s.%s does not exist", role, schema, table)
			} else if err != nil {
				return nil, err
			}
			if !owned {
				return nil, fmt.Errorf("role %s: table %s.%s is not owned by %s", role, schema, table, m.Owner())
			}
			for _, p := range sortedKeys(r.Tables[t]) {
				cols := r.Tables[t][p]
				if len(cols) == 0 {
					out = append(out, Grant{Kind: "table", Schema: schema, Name: table, Grantee: role, Privilege: p, Sequence: relkind == "S"})
					continue
				}
				for _, c := range cols {
					var ok bool
					if err := tx.QueryRow(ctx, `
						SELECT EXISTS (SELECT 1 FROM pg_catalog.pg_attribute a
						    JOIN pg_catalog.pg_class c ON c.oid = a.attrelid JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
						    WHERE n.nspname = $1 AND c.relname = $2 AND a.attname = $3 AND a.attnum > 0 AND NOT a.attisdropped)`,
						schema, table, c).Scan(&ok); err != nil {
						return nil, err
					}
					if !ok {
						return nil, fmt.Errorf("role %s: column %s.%s.%s does not exist", role, schema, table, c)
					}
					out = append(out, Grant{Kind: "column", Schema: schema, Name: table, Column: c, Grantee: role, Privilege: p})
				}
			}
		}
		for _, f := range sortedKeys(r.Functions) {
			fn, _ := normalizeFunction(f)
			if exFuncs[fn] {
				continue
			}
			var sig *string
			var owned bool
			err := tx.QueryRow(ctx, `
				SELECT p.oid::pg_catalog.regprocedure::text,
				       p.proowner = (SELECT oid FROM pg_catalog.pg_roles WHERE rolname = current_user)
				FROM pg_catalog.pg_proc p WHERE p.oid = pg_catalog.to_regprocedure($1)`, fn).Scan(&sig, &owned)
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, fmt.Errorf("role %s: function %s does not exist", role, fn)
			} else if err != nil {
				return nil, err
			}
			if !owned {
				return nil, fmt.Errorf("role %s: function %s is not owned by %s", role, fn, m.Owner())
			}
			for _, p := range r.Functions[f] {
				out = append(out, Grant{Kind: "function", Name: *sig, Grantee: role, Privilege: p})
			}
		}
	}
	return out, nil
}

// actualGrantsSQL lists every privilege held by anyone other than the owner
// on the objects the current role owns in the current database, PUBLIC and
// the implicit defaults (acldefault) included, plus its default privileges.
// The public schema is owned by pg_database_owner, i.e. by datdba.
const actualGrantsSQL = `
WITH me AS (SELECT oid FROM pg_catalog.pg_roles WHERE rolname = current_user),
raw AS (
    SELECT 'database' AS kind, '' AS schema, d.datname::text AS name, '' AS col, '' AS relkind, a.grantee, a.privilege_type, a.is_grantable
    FROM pg_catalog.pg_database d CROSS JOIN me,
         LATERAL pg_catalog.aclexplode(coalesce(d.datacl, pg_catalog.acldefault('d', d.datdba))) a
    WHERE d.datname = current_database() AND d.datdba = me.oid AND a.grantee <> d.datdba
  UNION ALL
    SELECT 'schema', '', n.nspname::text, '', '', a.grantee, a.privilege_type, a.is_grantable
    FROM pg_catalog.pg_namespace n CROSS JOIN me,
         LATERAL pg_catalog.aclexplode(coalesce(n.nspacl, pg_catalog.acldefault('n', n.nspowner))) a
    WHERE (n.nspowner = me.oid OR n.nspowner = 'pg_database_owner'::pg_catalog.regrole)
      AND a.grantee <> n.nspowner AND a.grantee <> me.oid
  UNION ALL
    SELECT 'table', n.nspname::text, c.relname::text, '', c.relkind::text, a.grantee, a.privilege_type, a.is_grantable
    FROM pg_catalog.pg_class c JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace CROSS JOIN me,
         LATERAL pg_catalog.aclexplode(coalesce(c.relacl,
             pg_catalog.acldefault(CASE WHEN c.relkind = 'S' THEN 's' ELSE 'r' END::"char", c.relowner))) a
    WHERE c.relowner = me.oid AND c.relkind IN ('r', 'p', 'v', 'm', 'f', 'S') AND a.grantee <> c.relowner
  UNION ALL
    SELECT 'column', n.nspname::text, c.relname::text, at.attname::text, c.relkind::text, a.grantee, a.privilege_type, a.is_grantable
    FROM pg_catalog.pg_attribute at JOIN pg_catalog.pg_class c ON c.oid = at.attrelid
         JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace CROSS JOIN me,
         LATERAL pg_catalog.aclexplode(at.attacl) a
    WHERE c.relowner = me.oid AND at.attnum > 0 AND NOT at.attisdropped AND at.attacl IS NOT NULL
      AND a.grantee <> c.relowner
  UNION ALL
    SELECT 'function', '', p.oid::pg_catalog.regprocedure::text, '', '', a.grantee, a.privilege_type, a.is_grantable
    FROM pg_catalog.pg_proc p CROSS JOIN me,
         LATERAL pg_catalog.aclexplode(coalesce(p.proacl, pg_catalog.acldefault('f', p.proowner))) a
    WHERE p.proowner = me.oid AND a.grantee <> p.proowner
  UNION ALL
    -- Default privileges. With no global FUNCTIONS entry the built-in default
    -- (EXECUTE to PUBLIC) applies, so it counts as a grant too.
    SELECT 'default', coalesce(n.nspname::text, ''), d.objtype::text, '', '', a.grantee, a.privilege_type, a.is_grantable
    FROM (
        SELECT da.defaclnamespace AS nsp, da.defaclobjtype AS objtype, da.defaclacl AS acl
        FROM pg_catalog.pg_default_acl da, me WHERE da.defaclrole = me.oid
      UNION ALL
        SELECT 0, 'f', pg_catalog.acldefault('f', me.oid) FROM me
        WHERE NOT EXISTS (SELECT 1 FROM pg_catalog.pg_default_acl da
                          WHERE da.defaclrole = me.oid AND da.defaclnamespace = 0 AND da.defaclobjtype = 'f')
    ) d CROSS JOIN me
    LEFT JOIN pg_catalog.pg_namespace n ON n.oid = d.nsp,
    LATERAL pg_catalog.aclexplode(d.acl) a
    WHERE a.grantee <> me.oid
)
SELECT kind, schema, name, col, relkind,
       CASE WHEN grantee = 0 THEN 'PUBLIC' ELSE (SELECT rolname::text FROM pg_catalog.pg_roles WHERE oid = grantee) END,
       privilege_type, is_grantable
FROM raw ORDER BY 1, 2, 3, 4, 6, 7`

func readActual(ctx context.Context, tx pgx.Tx) ([]Grant, error) {
	rows, err := tx.Query(ctx, actualGrantsSQL)
	if err != nil {
		return nil, err
	}
	var out []Grant
	for rows.Next() {
		var g Grant
		var relkind string
		var grantee *string
		if err := rows.Scan(&g.Kind, &g.Schema, &g.Name, &g.Column, &relkind, &grantee, &g.Privilege, &g.Grantable); err != nil {
			return nil, err
		}
		if grantee == nil {
			return nil, fmt.Errorf("grant %s held by a dropped role", g)
		}
		g.Grantee = *grantee
		g.Sequence = relkind == "S"
		out = append(out, g)
	}
	return out, rows.Err()
}

// statement builds the GRANT or REVOKE for g. Identifiers are quoted;
// privileges come from the catalogue or the validated manifest and are
// checked again; a function signature is regprocedure output, already quoted.
func statement(verb string, g Grant, owner string, optionOnly bool) (string, error) {
	if !privRe.MatchString(g.Privilege) {
		return "", fmt.Errorf("unexpected privilege %q", g.Privilege)
	}
	grantee := "PUBLIC"
	if g.Grantee != "PUBLIC" {
		grantee = ident(g.Grantee)
	}
	priv := g.Privilege
	if optionOnly {
		priv = "GRANT OPTION FOR " + priv
	}
	if g.Kind == "column" {
		priv += " (" + ident(g.Column) + ")"
	}
	var on string
	switch g.Kind {
	case "database":
		on = "DATABASE " + ident(g.Name)
	case "schema":
		on = "SCHEMA " + ident(g.Name)
	case "table", "column":
		on = "TABLE " + ident(g.Schema) + "." + ident(g.Name)
		if g.Sequence {
			on = "SEQUENCE " + ident(g.Schema) + "." + ident(g.Name)
		}
	case "function":
		on = "ROUTINE " + g.Name
	case "default":
		objs, ok := defaultObjType(g.Name)
		if !ok || verb != "REVOKE" {
			return "", fmt.Errorf("cannot %s default privilege %s", verb, g)
		}
		in := ""
		if g.Schema != "" {
			in = " IN SCHEMA " + ident(g.Schema)
		}
		return fmt.Sprintf("ALTER DEFAULT PRIVILEGES FOR ROLE %s%s REVOKE %s ON %s FROM %s CASCADE",
			ident(owner), in, priv, objs, grantee), nil
	default:
		return "", fmt.Errorf("unknown grant kind %q", g.Kind)
	}
	if verb == "REVOKE" {
		return fmt.Sprintf("REVOKE %s ON %s FROM %s CASCADE", priv, on, grantee), nil
	}
	return fmt.Sprintf("GRANT %s ON %s TO %s", priv, on, grantee), nil
}

func ident(s string) string { return pgx.Identifier{s}.Sanitize() }

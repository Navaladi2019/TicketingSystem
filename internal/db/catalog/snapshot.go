package catalog

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/jackc/pgx/v5"
)

// SnapshotFormat is the version of the schema snapshot's shape
// (docs/handbook/schema.md). New keys are additive and keep it; a change to
// an existing key, or a key removed, bumps it.
const SnapshotFormat = 1

// Schema is the schema snapshot (TICK-733, D-HB-9): every table and view of
// the platform schemas as a freshly migrated database's catalogue shows it,
// in a stable order. The integration suite writes it to
// docs/handbook/src/database/schema.json and fails when the committed file is
// stale, so the handbook reads the schema without a database. It holds no
// connection details: no host, port, role password or database name.
type Schema struct {
	Format    int     `json:"format"`
	Postgres  string  `json:"postgres"`  // server major version
	Migration string  `json:"migration"` // the latest applied goose version
	Tables    []Table `json:"tables"`
	Views     []View  `json:"views"`
}

// Table is one table: ordinary, partitioned, a partition, or foreign.
type Table struct {
	Schema         string       `json:"schema"`
	Name           string       `json:"name"`
	Kind           string       `json:"kind"` // table | partitioned | partition | foreign
	PartitionKey   string       `json:"partition_key,omitempty"`
	PartitionOf    string       `json:"partition_of,omitempty"`
	PartitionBound string       `json:"partition_bound,omitempty"`
	RLS            RLS          `json:"rls"`
	Columns        []Column     `json:"columns"`
	Constraints    []Constraint `json:"constraints"`
	Indexes        []Index      `json:"indexes"`
	Policies       []RLSPolicy  `json:"policies"`
}

// RLS is a table's ENABLE and FORCE ROW LEVEL SECURITY flags.
type RLS struct {
	Enabled bool `json:"enabled"`
	Forced  bool `json:"forced"`
}

// Column is one column, in attnum order.
type Column struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	NotNull   bool   `json:"not_null"`
	Default   string `json:"default,omitempty"`
	Identity  string `json:"identity,omitempty"`  // always | by_default
	Generated string `json:"generated,omitempty"` // the stored expression
}

// Constraint is a primary key, foreign key, unique, check or exclusion
// constraint.
type Constraint struct {
	Name       string     `json:"name"`
	Type       string     `json:"type"`
	Columns    []string   `json:"columns,omitempty"`
	References *Reference `json:"references,omitempty"`
	Definition string     `json:"definition"`
	Inherited  bool       `json:"inherited,omitempty"` // a partition's copy of its parent's
	NotValid   bool       `json:"not_valid,omitempty"`
}

// Reference is a foreign key's target.
type Reference struct {
	Table   string   `json:"table"`
	Columns []string `json:"columns"`
}

// Index is one index; Definition is pg_get_indexdef.
type Index struct {
	Name       string   `json:"name"`
	Method     string   `json:"method"`
	Unique     bool     `json:"unique"`
	Primary    bool     `json:"primary"`
	Keys       []string `json:"keys"`
	Include    []string `json:"include,omitempty"`
	Predicate  string   `json:"predicate,omitempty"`
	Definition string   `json:"definition"`
	Invalid    bool     `json:"invalid,omitempty"`
}

// RLSPolicy is one row of pg_policies.
type RLSPolicy struct {
	Name       string   `json:"name"`
	Command    string   `json:"command"`
	Roles      []string `json:"roles"`
	Permissive bool     `json:"permissive"`
	Using      string   `json:"using,omitempty"`
	WithCheck  string   `json:"with_check,omitempty"`
}

// View is one view and its deparsed query.
type View struct {
	Schema     string `json:"schema"`
	Name       string `json:"name"`
	Definition string `json:"definition"`
}

// RefName is how the snapshot and the handbook's front matter name a table:
// bare in schema public, "schema.name" otherwise (the db/grants.yaml rule).
func RefName(schema, name string) string {
	if schema == "public" {
		return name
	}
	return schema + "." + name
}

// snapRelations is platformRelations without the goose version table, which
// is the migrator's bookkeeping, not the platform's schema.
func snapRelations() string {
	return platformRelations() + ` AND (n.nspname || '.' || c.relname) <> '` + versionTable + `'`
}

// TakeSnapshot reads the schema snapshot from the database q is connected to.
// It needs a role that can read the whole catalogue; the integration suite
// uses the bootstrap superuser on the migrated platform database.
func TakeSnapshot(ctx context.Context, q Querier) (*Schema, error) {
	s := &Schema{Format: SnapshotFormat, Tables: []Table{}, Views: []View{}}
	if err := scanOne(ctx, q, `SELECT (current_setting('server_version_num')::int / 10000)::text`, nil, &s.Postgres); err != nil {
		return nil, fmt.Errorf("schema snapshot: %w", err)
	}
	if err := scanOne(ctx, q, `SELECT coalesce(max(version_id), 0)::text FROM `+versionTable+` WHERE is_applied`, nil, &s.Migration); err != nil {
		return nil, fmt.Errorf("schema snapshot: %w", err)
	}
	byName := map[string]*Table{}
	for _, step := range []func(context.Context, Querier, *Schema, map[string]*Table) error{
		snapTables, snapColumns, snapConstraints, snapIndexes, snapPolicies, snapViews,
	} {
		if err := step(ctx, q, s, byName); err != nil {
			return nil, fmt.Errorf("schema snapshot: %w", err)
		}
	}
	return s, nil
}

func snapTables(ctx context.Context, q Querier, s *Schema, byName map[string]*Table) error {
	rows, err := q.Query(ctx, `
		SELECT n.nspname::text, c.relname::text,
		       CASE WHEN c.relispartition THEN 'partition'
		            WHEN c.relkind = 'p' THEN 'partitioned'
		            WHEN c.relkind = 'f' THEN 'foreign'
		            ELSE 'table' END,
		       CASE WHEN c.relkind = 'p' THEN pg_catalog.pg_get_partkeydef(c.oid) ELSE '' END,
		       coalesce((SELECT pn.nspname::text || '.' || pc.relname::text
		                 FROM pg_catalog.pg_inherits i
		                 JOIN pg_catalog.pg_class pc ON pc.oid = i.inhparent
		                 JOIN pg_catalog.pg_namespace pn ON pn.oid = pc.relnamespace
		                 WHERE i.inhrelid = c.oid AND c.relispartition), ''),
		       CASE WHEN c.relispartition THEN pg_catalog.pg_get_expr(c.relpartbound, c.oid) ELSE '' END,
		       c.relrowsecurity, c.relforcerowsecurity
		`+snapRelations()+` AND `+isTable+`
		ORDER BY 1, 2`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		t := Table{Columns: []Column{}, Constraints: []Constraint{}, Indexes: []Index{}, Policies: []RLSPolicy{}}
		var parent string
		if err := rows.Scan(&t.Schema, &t.Name, &t.Kind, &t.PartitionKey, &parent, &t.PartitionBound,
			&t.RLS.Enabled, &t.RLS.Forced); err != nil {
			return err
		}
		if ps, pn, ok := strings.Cut(parent, "."); ok {
			t.PartitionOf = RefName(ps, pn)
		}
		s.Tables = append(s.Tables, t)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for i := range s.Tables {
		t := &s.Tables[i]
		byName[t.Schema+"."+t.Name] = t
	}
	return nil
}

// table finds the snapshot table a catalogue row belongs to; a row for a
// relation the table query did not return is a bug in the queries.
func table(byName map[string]*Table, schema, name string) (*Table, error) {
	t, ok := byName[schema+"."+name]
	if !ok {
		return nil, fmt.Errorf("catalogue row for %s.%s, which is not a snapshot table", schema, name)
	}
	return t, nil
}

func snapColumns(ctx context.Context, q Querier, _ *Schema, byName map[string]*Table) error {
	rows, err := q.Query(ctx, `
		SELECT n.nspname::text, c.relname::text, a.attname::text,
		       pg_catalog.format_type(a.atttypid, a.atttypmod), a.attnotnull,
		       CASE WHEN a.attgenerated = '' THEN coalesce(pg_catalog.pg_get_expr(d.adbin, d.adrelid), '') ELSE '' END,
		       CASE a.attidentity WHEN 'a' THEN 'always' WHEN 'd' THEN 'by_default' ELSE '' END,
		       CASE WHEN a.attgenerated <> '' THEN coalesce(pg_catalog.pg_get_expr(d.adbin, d.adrelid), '') ELSE '' END
		`+strings.Replace(snapRelations(), "WHERE", `JOIN pg_catalog.pg_attribute a ON a.attrelid = c.oid
		LEFT JOIN pg_catalog.pg_attrdef d ON d.adrelid = a.attrelid AND d.adnum = a.attnum
		WHERE`, 1)+` AND `+isTable+` AND a.attnum > 0 AND NOT a.attisdropped
		ORDER BY 1, 2, a.attnum`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var schema, rel string
		var col Column
		if err := rows.Scan(&schema, &rel, &col.Name, &col.Type, &col.NotNull, &col.Default, &col.Identity, &col.Generated); err != nil {
			return err
		}
		t, err := table(byName, schema, rel)
		if err != nil {
			return err
		}
		t.Columns = append(t.Columns, col)
	}
	return rows.Err()
}

// constraintType maps a pg_constraint.contype the snapshot keeps to its name.
// Others (constraint triggers; PG 17's NOT NULL rows, already in
// Column.NotNull) are left out by the query.
func constraintType(contype string) string {
	switch contype {
	case "p":
		return "primary_key"
	case "f":
		return "foreign_key"
	case "u":
		return "unique"
	case "c":
		return "check"
	default:
		return "exclusion"
	}
}

func snapConstraints(ctx context.Context, q Querier, _ *Schema, byName map[string]*Table) error {
	rows, err := q.Query(ctx, `
		SELECT n.nspname::text, c.relname::text, con.conname::text, con.contype::text,
		       ARRAY(SELECT a.attname::text FROM unnest(con.conkey) WITH ORDINALITY k(num, ord)
		             JOIN pg_catalog.pg_attribute a ON a.attrelid = con.conrelid AND a.attnum = k.num ORDER BY k.ord),
		       coalesce((SELECT fn.nspname::text || '.' || fc.relname::text FROM pg_catalog.pg_class fc
		                 JOIN pg_catalog.pg_namespace fn ON fn.oid = fc.relnamespace WHERE fc.oid = con.confrelid), ''),
		       ARRAY(SELECT a.attname::text FROM unnest(con.confkey) WITH ORDINALITY k(num, ord)
		             JOIN pg_catalog.pg_attribute a ON a.attrelid = con.confrelid AND a.attnum = k.num ORDER BY k.ord),
		       pg_catalog.pg_get_constraintdef(con.oid, true), con.conparentid <> 0, NOT con.convalidated
		`+strings.Replace(snapRelations(), "WHERE", `JOIN pg_catalog.pg_constraint con ON con.conrelid = c.oid
		WHERE`, 1)+` AND `+isTable+` AND con.contype IN ('p', 'f', 'u', 'c', 'x')
		ORDER BY 1, 2, 3`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var schema, rel, typ, target string
		var refCols []string
		var con Constraint
		if err := rows.Scan(&schema, &rel, &con.Name, &typ, &con.Columns, &target, &refCols,
			&con.Definition, &con.Inherited, &con.NotValid); err != nil {
			return err
		}
		con.Type = constraintType(typ)
		if ts, tn, ok := strings.Cut(target, "."); ok {
			con.References = &Reference{Table: RefName(ts, tn), Columns: refCols}
		}
		t, err := table(byName, schema, rel)
		if err != nil {
			return err
		}
		t.Constraints = append(t.Constraints, con)
	}
	return rows.Err()
}

func snapIndexes(ctx context.Context, q Querier, _ *Schema, byName map[string]*Table) error {
	rows, err := q.Query(ctx, `
		SELECT n.nspname::text, c.relname::text, ic.relname::text, am.amname::text, i.indisunique, i.indisprimary,
		       ARRAY(SELECT pg_catalog.pg_get_indexdef(i.indexrelid, k, true) FROM generate_series(1, i.indnkeyatts) k ORDER BY k),
		       ARRAY(SELECT pg_catalog.pg_get_indexdef(i.indexrelid, k, true) FROM generate_series(i.indnkeyatts + 1, i.indnatts) k ORDER BY k),
		       coalesce(pg_catalog.pg_get_expr(i.indpred, i.indrelid, true), ''),
		       pg_catalog.pg_get_indexdef(i.indexrelid), NOT i.indisvalid
		`+strings.Replace(snapRelations(), "WHERE", `JOIN pg_catalog.pg_index i ON i.indrelid = c.oid
		JOIN pg_catalog.pg_class ic ON ic.oid = i.indexrelid
		JOIN pg_catalog.pg_am am ON am.oid = ic.relam
		WHERE`, 1)+` AND `+isTable+`
		ORDER BY 1, 2, 3`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var schema, rel string
		var ix Index
		if err := rows.Scan(&schema, &rel, &ix.Name, &ix.Method, &ix.Unique, &ix.Primary, &ix.Keys, &ix.Include,
			&ix.Predicate, &ix.Definition, &ix.Invalid); err != nil {
			return err
		}
		t, err := table(byName, schema, rel)
		if err != nil {
			return err
		}
		t.Indexes = append(t.Indexes, ix)
	}
	return rows.Err()
}

func snapPolicies(ctx context.Context, q Querier, _ *Schema, byName map[string]*Table) error {
	rows, err := q.Query(ctx, `
		SELECT schemaname::text, tablename::text, policyname::text, cmd, roles::text[], permissive = 'PERMISSIVE',
		       coalesce(qual, ''), coalesce(with_check, '')
		FROM pg_catalog.pg_policies WHERE `+platformSchema("schemaname")+`
		ORDER BY 1, 2, 3`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var schema, rel string
		var p RLSPolicy
		if err := rows.Scan(&schema, &rel, &p.Name, &p.Command, &p.Roles, &p.Permissive, &p.Using, &p.WithCheck); err != nil {
			return err
		}
		slices.Sort(p.Roles)
		t, err := table(byName, schema, rel)
		if err != nil {
			return err
		}
		t.Policies = append(t.Policies, p)
	}
	return rows.Err()
}

func snapViews(ctx context.Context, q Querier, s *Schema, _ map[string]*Table) error {
	rows, err := q.Query(ctx, `
		SELECT n.nspname::text, c.relname::text, pg_catalog.pg_get_viewdef(c.oid, true)
		`+snapRelations()+` AND c.relkind IN ('v', 'm')
		ORDER BY 1, 2`)
	if err != nil {
		return err
	}
	views, err := pgx.CollectRows(rows, pgx.RowToStructByPos[View])
	if err != nil {
		return err
	}
	s.Views = append(s.Views, views...)
	return nil
}

// Marshal is the snapshot as committed: two-space indented JSON with a
// trailing newline and no HTML escaping, so a predicate such as
// "dispatched_at < now()" stays readable in a diff.
func (s *Schema) Marshal() ([]byte, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(s); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// snapshotLeakPatterns are what a schema snapshot must never contain: the
// connection details of the database it was taken from (D-HB-9; the
// repository is public). TICK-735's handbook scanner covers every generated
// file; this guard runs where the snapshot is written and read.
func snapshotLeakPatterns() []struct{ what, re string } {
	return []struct{ what, re string }{
		{"a URL or DSN", `://`},
		{"a keyword DSN", `(?i)\b(host|hostaddr|port|user|password|dbname|sslmode)\s*=`},
		{"a password", `(?i)passw(or)?d|\btest-pw-|\btest-only\b`},
		{"the bootstrap superuser", `\bpgadmin\b`},
		{"a pg_hba.conf entry", `(?i)\bscram-sha-256\b|\b(local|host|hostssl|hostnossl)\s+(all|replication)\s`},
		{"an IPv4 address", `\b\d{1,3}(\.\d{1,3}){3}\b`},
		{"an IPv6 address", `::/\d|\b[0-9a-fA-F]{1,4}(:[0-9a-fA-F]{1,4}){7}\b`},
		{"an absolute local path", `(?i)\b[a-z]:\\|/home/|/Users/`},
	}
}

// SnapshotLeaks lists every connection detail b contains: each of
// snapshotLeakPatterns, and each non-empty value in also (the actual host,
// user and password the snapshot was taken with). Empty means clean.
func SnapshotLeaks(b []byte, also ...string) []string {
	var out []string
	for _, p := range snapshotLeakPatterns() {
		if m := regexp.MustCompile(`(?m)` + p.re).Find(b); m != nil {
			out = append(out, fmt.Sprintf("%s: %q", p.what, m))
		}
	}
	for _, v := range also {
		if v != "" && bytes.Contains(b, []byte(v)) {
			out = append(out, fmt.Sprintf("a connection value: %q", v))
		}
	}
	return out
}

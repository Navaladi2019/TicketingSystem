package catalog

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// SnapshotVersion is the version of the schema snapshot's contract, the shape
// agreed between TICK-733 and the handbook reader (TICK-719) and documented
// in docs/handbook/src/database/schema.md. New keys are additive and keep it;
// a key renamed or removed bumps it.
const SnapshotVersion = 1

// Schema is the schema snapshot (TICK-733, D-HB-9): every table of schema
// public as a freshly migrated database's catalogue shows it, in a stable
// order. The integration suite writes it to
// docs/handbook/src/database/schema.json and fails when the committed file is
// stale, so the handbook reads the schema without a database. It holds no
// connection details: no host, port, role password or database name.
type Schema struct {
	Version int     `json:"version"`
	Tables  []Table `json:"tables"`
}

// Table is one table. Kind is table, partitioned or partition; the three
// partition keys are "" where they do not apply.
type Table struct {
	Name           string       `json:"name"`
	Kind           string       `json:"kind"`
	PartitionOf    string       `json:"partition_of"`
	PartitionBy    string       `json:"partition_by"`
	PartitionBound string       `json:"partition_bound"`
	RLS            RLS          `json:"rls"`
	Columns        []Column     `json:"columns"`
	PrimaryKey     []string     `json:"primary_key"`
	ForeignKeys    []ForeignKey `json:"foreign_keys"`
	Indexes        []Index      `json:"indexes"`
	Policies       []RLSPolicy  `json:"policies"`
}

// RLS is a table's ENABLE and FORCE ROW LEVEL SECURITY flags.
type RLS struct {
	Enabled bool `json:"enabled"`
	Forced  bool `json:"forced"`
}

// Column is one column, in ordinal (attnum) order. Default is the deparsed
// expression or "".
type Column struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Nullable bool   `json:"nullable"`
	Default  string `json:"default"`
}

// ForeignKey is one foreign key constraint.
type ForeignKey struct {
	Name       string   `json:"name"`
	Columns    []string `json:"columns"`
	RefTable   string   `json:"ref_table"`
	RefColumns []string `json:"ref_columns"`
	OnDelete   string   `json:"on_delete"`
}

// Index is one index. Columns are the key columns, each a column name or the
// deparsed expression of an expression index; Include is the INCLUDE list;
// Predicate is the partial index's WHERE, or "".
type Index struct {
	Name      string   `json:"name"`
	Method    string   `json:"method"`
	Unique    bool     `json:"unique"`
	Columns   []string `json:"columns"`
	Include   []string `json:"include"`
	Predicate string   `json:"predicate"`
}

// RLSPolicy is one row of pg_policies; Using and Check are the deparsed
// predicates, or "".
type RLSPolicy struct {
	Name       string   `json:"name"`
	Command    string   `json:"command"`
	Roles      []string `json:"roles"`
	Permissive bool     `json:"permissive"`
	Using      string   `json:"using"`
	Check      string   `json:"check"`
}

// snapRelations is platformRelations without the goose version table, which
// is the migrator's bookkeeping, not the platform's schema.
func snapRelations() string {
	return platformRelations() + ` AND (n.nspname || '.' || c.relname) <> '` + versionTable + `'`
}

// withJoin is snapRelations with join inserted before its WHERE.
func withJoin(join string) string {
	return strings.Replace(snapRelations(), "WHERE", join+"\n\t\tWHERE", 1)
}

// TakeSnapshot reads the schema snapshot from the database q is connected to.
// It needs a role that can read the whole catalogue; the integration suite
// uses the bootstrap superuser on the migrated platform database. A table
// outside schema public, or a foreign table, is an error: version 1 of the
// contract has no way to name either.
func TakeSnapshot(ctx context.Context, q Querier) (*Schema, error) {
	s := &Schema{Version: SnapshotVersion, Tables: []Table{}}
	byName := map[string]*Table{}
	for _, step := range []func(context.Context, Querier, *Schema, map[string]*Table) error{
		snapTables, snapColumns, snapConstraints, snapIndexes, snapPolicies,
	} {
		if err := step(ctx, q, s, byName); err != nil {
			return nil, fmt.Errorf("schema snapshot: %w", err)
		}
	}
	return s, nil
}

// tableKind is the contract's kind for a relation, or an error for one it
// cannot name.
func tableKind(schema, name, relkind string, isPartition bool) (string, error) {
	switch {
	case schema != "public":
		return "", fmt.Errorf("table %s.%s is outside schema public, which snapshot version %d cannot name", schema, name, SnapshotVersion)
	case relkind == "f":
		return "", fmt.Errorf("foreign table %s: snapshot version %d has no kind for it", name, SnapshotVersion)
	case isPartition:
		return "partition", nil
	case relkind == "p":
		return "partitioned", nil
	default:
		return "table", nil
	}
}

func snapTables(ctx context.Context, q Querier, s *Schema, byName map[string]*Table) error {
	rows, err := q.Query(ctx, `
		SELECT n.nspname::text, c.relname::text, c.relkind::text, c.relispartition,
		       CASE WHEN c.relkind = 'p' THEN pg_catalog.pg_get_partkeydef(c.oid) ELSE '' END,
		       coalesce((SELECT pc.relname::text
		                 FROM pg_catalog.pg_inherits i JOIN pg_catalog.pg_class pc ON pc.oid = i.inhparent
		                 WHERE i.inhrelid = c.oid AND c.relispartition), ''),
		       CASE WHEN c.relispartition THEN pg_catalog.pg_get_expr(c.relpartbound, c.oid) ELSE '' END,
		       c.relrowsecurity, c.relforcerowsecurity
		`+snapRelations()+` AND `+isTable+`
		ORDER BY 2, 1`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		t := Table{Columns: []Column{}, PrimaryKey: []string{}, ForeignKeys: []ForeignKey{}, Indexes: []Index{}, Policies: []RLSPolicy{}}
		var schema, relkind string
		var isPartition bool
		if err := rows.Scan(&schema, &t.Name, &relkind, &isPartition, &t.PartitionBy, &t.PartitionOf, &t.PartitionBound,
			&t.RLS.Enabled, &t.RLS.Forced); err != nil {
			return err
		}
		if t.Kind, err = tableKind(schema, t.Name, relkind, isPartition); err != nil {
			return err
		}
		s.Tables = append(s.Tables, t)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for i := range s.Tables {
		byName[s.Tables[i].Name] = &s.Tables[i]
	}
	return nil
}

// table finds the snapshot table a catalogue row belongs to; a row for a
// relation the table query did not return is a bug in the queries.
func table(byName map[string]*Table, name string) (*Table, error) {
	t, ok := byName[name]
	if !ok {
		return nil, fmt.Errorf("catalogue row for %s, which is not a snapshot table", name)
	}
	return t, nil
}

func snapColumns(ctx context.Context, q Querier, _ *Schema, byName map[string]*Table) error {
	rows, err := q.Query(ctx, `
		SELECT c.relname::text, a.attname::text, pg_catalog.format_type(a.atttypid, a.atttypmod), NOT a.attnotnull,
		       coalesce(pg_catalog.pg_get_expr(d.adbin, d.adrelid), '')
		`+withJoin(`JOIN pg_catalog.pg_attribute a ON a.attrelid = c.oid
		LEFT JOIN pg_catalog.pg_attrdef d ON d.adrelid = a.attrelid AND d.adnum = a.attnum`)+`
		  AND `+isTable+` AND a.attnum > 0 AND NOT a.attisdropped
		ORDER BY 1, a.attnum`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var rel string
		var col Column
		if err := rows.Scan(&rel, &col.Name, &col.Type, &col.Nullable, &col.Default); err != nil {
			return err
		}
		t, err := table(byName, rel)
		if err != nil {
			return err
		}
		t.Columns = append(t.Columns, col)
	}
	return rows.Err()
}

// onDelete is a foreign key's ON DELETE action from pg_constraint.confdeltype.
func onDelete(confdeltype string) string {
	switch confdeltype {
	case "r":
		return "RESTRICT"
	case "c":
		return "CASCADE"
	case "n":
		return "SET NULL"
	case "d":
		return "SET DEFAULT"
	default: // "a"
		return "NO ACTION"
	}
}

// snapConstraints fills each table's primary key and foreign keys. A
// partition lists its own copies of its parent's.
func snapConstraints(ctx context.Context, q Querier, _ *Schema, byName map[string]*Table) error {
	rows, err := q.Query(ctx, `
		SELECT c.relname::text, con.conname::text, con.contype::text,
		       ARRAY(SELECT a.attname::text FROM unnest(con.conkey) WITH ORDINALITY k(num, ord)
		             JOIN pg_catalog.pg_attribute a ON a.attrelid = con.conrelid AND a.attnum = k.num ORDER BY k.ord),
		       coalesce((SELECT fc.relname::text FROM pg_catalog.pg_class fc WHERE fc.oid = con.confrelid), ''),
		       ARRAY(SELECT a.attname::text FROM unnest(con.confkey) WITH ORDINALITY k(num, ord)
		             JOIN pg_catalog.pg_attribute a ON a.attrelid = con.confrelid AND a.attnum = k.num ORDER BY k.ord),
		       con.confdeltype::text
		`+withJoin(`JOIN pg_catalog.pg_constraint con ON con.conrelid = c.oid`)+`
		  AND `+isTable+` AND con.contype IN ('p', 'f')
		ORDER BY 1, 2`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var rel, typ, delType string
		var fk ForeignKey
		if err := rows.Scan(&rel, &fk.Name, &typ, &fk.Columns, &fk.RefTable, &fk.RefColumns, &delType); err != nil {
			return err
		}
		t, err := table(byName, rel)
		if err != nil {
			return err
		}
		if typ == "p" {
			t.PrimaryKey = fk.Columns
			continue
		}
		fk.OnDelete = onDelete(delType)
		t.ForeignKeys = append(t.ForeignKeys, fk)
	}
	return rows.Err()
}

func snapIndexes(ctx context.Context, q Querier, _ *Schema, byName map[string]*Table) error {
	rows, err := q.Query(ctx, `
		SELECT c.relname::text, ic.relname::text, am.amname::text, i.indisunique,
		       ARRAY(SELECT pg_catalog.pg_get_indexdef(i.indexrelid, k, true) FROM generate_series(1, i.indnkeyatts) k ORDER BY k),
		       ARRAY(SELECT pg_catalog.pg_get_indexdef(i.indexrelid, k, true) FROM generate_series(i.indnkeyatts + 1, i.indnatts) k ORDER BY k),
		       coalesce(pg_catalog.pg_get_expr(i.indpred, i.indrelid, true), '')
		`+withJoin(`JOIN pg_catalog.pg_index i ON i.indrelid = c.oid
		JOIN pg_catalog.pg_class ic ON ic.oid = i.indexrelid
		JOIN pg_catalog.pg_am am ON am.oid = ic.relam`)+`
		  AND `+isTable+`
		ORDER BY 1, 2`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var rel string
		var ix Index
		if err := rows.Scan(&rel, &ix.Name, &ix.Method, &ix.Unique, &ix.Columns, &ix.Include, &ix.Predicate); err != nil {
			return err
		}
		t, err := table(byName, rel)
		if err != nil {
			return err
		}
		t.Indexes = append(t.Indexes, ix)
	}
	return rows.Err()
}

func snapPolicies(ctx context.Context, q Querier, _ *Schema, byName map[string]*Table) error {
	rows, err := q.Query(ctx, `
		SELECT tablename::text, policyname::text, cmd, roles::text[], permissive = 'PERMISSIVE',
		       coalesce(qual, ''), coalesce(with_check, '')
		FROM pg_catalog.pg_policies WHERE `+platformSchema("schemaname")+`
		ORDER BY 1, 2`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var rel string
		var p RLSPolicy
		if err := rows.Scan(&rel, &p.Name, &p.Command, &p.Roles, &p.Permissive, &p.Using, &p.Check); err != nil {
			return err
		}
		slices.Sort(p.Roles)
		t, err := table(byName, rel)
		if err != nil {
			return err
		}
		t.Policies = append(t.Policies, p)
	}
	return rows.Err()
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

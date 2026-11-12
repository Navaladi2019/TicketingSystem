package catalog

import (
	"bytes"
	"encoding/json"
	"os"
	"slices"
	"strings"
	"testing"
)

// committedSnapshot is docs/handbook/src/database/schema.json (TICK-733),
// which the integration suite keeps current; these tests read it without a
// database, as the handbook build does. contractDoc is the page whose key
// table it must match.
const (
	committedSnapshot = "../../../docs/handbook/src/database/schema.json"
	contractDoc       = "../../../docs/handbook/src/database/schema.md"
)

func readFile(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestCommittedSnapshotShape(t *testing.T) {
	b := readFile(t, committedSnapshot)
	if bytes.Contains(b, []byte("\r")) || !bytes.HasSuffix(b, []byte("}\n")) {
		t.Error("schema.json must be LF with one trailing newline, as Marshal writes it")
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	var s Schema
	if err := dec.Decode(&s); err != nil {
		t.Fatal(err)
	}
	if again, err := s.Marshal(); err != nil || !bytes.Equal(again, b) {
		t.Errorf("schema.json is not in Marshal's form (hand-edited?); regenerate it with -update (err %v)", err)
	}
	if s.Version != SnapshotVersion {
		t.Errorf("version = %d, want %d", s.Version, SnapshotVersion)
	}
	if len(s.Tables) == 0 {
		t.Fatal("no tables")
	}
	if !slices.IsSortedFunc(s.Tables, func(a, b Table) int { return strings.Compare(a.Name, b.Name) }) {
		t.Error("tables are not sorted by name")
	}
	for _, tb := range s.Tables {
		if tb.Name == "goose_db_version" {
			t.Error("goose_db_version is migrator bookkeeping, not schema")
		}
		if len(tb.Columns) == 0 {
			t.Errorf("%s has no columns", tb.Name)
		}
		switch tb.Kind {
		case "table":
			if tb.PartitionOf != "" || tb.PartitionBy != "" || tb.PartitionBound != "" {
				t.Errorf("%s is a table with partition keys set", tb.Name)
			}
		case "partitioned":
			if tb.PartitionBy == "" {
				t.Errorf("%s is partitioned with no partition_by", tb.Name)
			}
		case "partition":
			if tb.PartitionOf == "" || tb.PartitionBound == "" {
				t.Errorf("%s is a partition with no parent or bound", tb.Name)
			}
		default:
			t.Errorf("%s has kind %q", tb.Name, tb.Kind)
		}
		for _, p := range tb.Policies {
			if !slices.IsSorted(p.Roles) {
				t.Errorf("%s.%s roles %q are not sorted", tb.Name, p.Name, p.Roles)
			}
		}
		for i := 1; i < len(tb.Indexes); i++ {
			if tb.Indexes[i-1].Name >= tb.Indexes[i].Name {
				t.Errorf("%s indexes are not sorted by name", tb.Name)
			}
		}
	}
	if problems := SnapshotKeyProblems(readFile(t, contractDoc), b); len(problems) > 0 {
		t.Errorf("schema.json does not match the key table in schema.md: %q", problems)
	}
	if leaks := SnapshotLeaks(b); len(leaks) > 0 {
		t.Errorf("schema.json holds connection details: %q", leaks)
	}
}

// The contract document lists every key version 1 defines.
func TestContractDocumentListsEveryKey(t *testing.T) {
	keys := SnapshotContractKeys(readFile(t, contractDoc))
	for _, want := range []string{
		"version", "tables", "tables[].kind", "tables[].partition_by", "tables[].rls.forced",
		"tables[].columns[].nullable", "tables[].primary_key", "tables[].foreign_keys[].on_delete",
		"tables[].indexes[].columns", "tables[].indexes[].include", "tables[].policies[].permissive",
		"tables[].policies[].check",
	} {
		if !slices.Contains(keys, want) {
			t.Errorf("schema.md does not list %s; listed: %q", want, keys)
		}
	}
}

// Fixtures: a key missing from one object, a key not in the document, a
// listed key with nothing to carry it, and an empty document each fail.
func TestSnapshotKeyProblems(t *testing.T) {
	doc := []byte("| Key | Value |\n|---|---|\n| `version` | 1 |\n| `tables` | x |\n| `tables[].name` | x |\n" +
		"| `tables[].indexes` | x |\n| `tables[].indexes[].include` | x |\n")
	good := `{"version": 1, "tables": [{"name": "a", "indexes": [{"include": []}]}, {"name": "b", "indexes": []}]}`
	if got := SnapshotKeyProblems(doc, []byte(good)); len(got) != 0 {
		t.Fatalf("a snapshot with every key: %q", got)
	}
	for _, c := range []struct{ name, doc, snap, want string }{
		{"missing key", string(doc), `{"version": 1, "tables": [{"name": "a", "indexes": [{"include": []}, {}]}]}`,
			"tables[].indexes[].include: missing from 1 of 2 objects"},
		{"missing top-level key", string(doc), `{"tables": [{"name": "a", "indexes": [{"include": []}]}]}`,
			"version: missing from 1 of 1 objects"},
		{"unlisted key", string(doc), `{"version": 1, "tables": [{"name": "a", "extra": "", "indexes": [{"include": []}]}]}`,
			"tables[].extra: in the snapshot but not listed"},
		{"nothing to carry it", string(doc), `{"version": 1, "tables": [{"name": "a", "indexes": []}]}`,
			"tables[].indexes[].include: the snapshot has no tables[].indexes[] object"},
		{"empty document", "no table here", good, "the contract document lists no keys"},
		{"not JSON", string(doc), `{`, "snapshot is not JSON"},
	} {
		got := SnapshotKeyProblems([]byte(c.doc), []byte(c.snap))
		if !slices.ContainsFunc(got, func(p string) bool { return strings.HasPrefix(p, c.want) }) {
			t.Errorf("%s: problems %q, want one starting %q", c.name, got, c.want)
		}
	}
}

func TestSnapshotLeaks(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{`"x": "postgres://u@db/nava"`, "a URL or DSN"},
		{`"x": "host=db dbname=nava"`, "a keyword DSN"},
		{`"x": "test-pw-nava_app"`, "a password"},
		{`"password_hash"`, "a password"},
		{`"owner": "pgadmin"`, "the bootstrap superuser"},
		{`host    all          nava_backup  0.0.0.0/0      reject`, "a pg_hba.conf entry"},
		{`"x": "10.1.2.3"`, "an IPv4 address"},
		{`"x": "::/0"`, "an IPv6 address"},
		{`"x": "C:\\Users\\me"`, "an absolute local path"},
		{`"x": "/home/me/repo"`, "an absolute local path"},
	} {
		if got := SnapshotLeaks([]byte(c.in)); len(got) == 0 || !strings.HasPrefix(got[0], c.want) {
			t.Errorf("SnapshotLeaks(%s) = %q, want %q first", c.in, got, c.want)
		}
	}
	if got := SnapshotLeaks([]byte(`"x": "db-host-7"`), "db-host-7"); len(got) != 1 {
		t.Errorf("an extra connection value is not reported: %q", got)
	}
	clean := `{"using": "(tenant_id = (NULLIF(current_setting('app.tenant_id'::text, true), ''::text))::uuid)",
	  "default": "'{}'::jsonb", "check": "(dispatched_at < (now() - '7 days'::interval))", "user": "current_user"}`
	if got := SnapshotLeaks([]byte(clean), ""); len(got) != 0 {
		t.Errorf("a clean snapshot fragment reports %q", got)
	}
}

func TestTableKind(t *testing.T) {
	for _, c := range []struct {
		schema, relkind string
		partition       bool
		want            string
	}{
		{"public", "r", false, "table"},
		{"public", "p", false, "partitioned"},
		{"public", "r", true, "partition"},
		{"public", "p", true, "partition"},
	} {
		if got, err := tableKind(c.schema, "t", c.relkind, c.partition); err != nil || got != c.want {
			t.Errorf("tableKind(%+v) = %q, %v; want %q", c, got, err, c.want)
		}
	}
	if _, err := tableKind("audit", "t", "r", false); err == nil || !strings.Contains(err.Error(), "outside schema public") {
		t.Errorf("a table outside public: err = %v", err)
	}
	if _, err := tableKind("public", "t", "f", false); err == nil || !strings.Contains(err.Error(), "foreign table") {
		t.Errorf("a foreign table: err = %v", err)
	}
}

func TestOnDelete(t *testing.T) {
	for in, want := range map[string]string{"a": "NO ACTION", "r": "RESTRICT", "c": "CASCADE", "n": "SET NULL", "d": "SET DEFAULT"} {
		if got := onDelete(in); got != want {
			t.Errorf("onDelete(%q) = %q, want %q", in, got, want)
		}
	}
}

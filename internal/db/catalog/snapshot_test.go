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
// database, as the handbook build does.
const committedSnapshot = "../../../docs/handbook/src/database/schema.json"

func TestCommittedSnapshotShape(t *testing.T) {
	b, err := os.ReadFile(committedSnapshot)
	if err != nil {
		t.Fatal(err)
	}
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
	if s.Format != SnapshotFormat || s.Postgres == "" || s.Migration == "" {
		t.Errorf("header = format %d, postgres %q, migration %q", s.Format, s.Postgres, s.Migration)
	}
	if len(s.Tables) == 0 {
		t.Fatal("no tables")
	}
	if !slices.IsSortedFunc(s.Tables, func(a, b Table) int {
		return strings.Compare(a.Schema+"\x00"+a.Name, b.Schema+"\x00"+b.Name)
	}) {
		t.Error("tables are not sorted by (schema, name)")
	}
	for _, tb := range s.Tables {
		if tb.Name == "goose_db_version" {
			t.Error("goose_db_version is migrator bookkeeping, not schema")
		}
		if len(tb.Columns) == 0 {
			t.Errorf("%s has no columns", tb.Name)
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
	if leaks := SnapshotLeaks(b); len(leaks) > 0 {
		t.Errorf("schema.json holds connection details: %q", leaks)
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

func TestRefName(t *testing.T) {
	if RefName("public", "records") != "records" || RefName("audit", "x") != "audit.x" {
		t.Error("RefName: bare in public, schema.name elsewhere")
	}
}

func TestConstraintType(t *testing.T) {
	for in, want := range map[string]string{"p": "primary_key", "f": "foreign_key", "u": "unique", "c": "check", "x": "exclusion"} {
		if got := constraintType(in); got != want {
			t.Errorf("constraintType(%q) = %q, want %q", in, got, want)
		}
	}
}

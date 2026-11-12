//go:build integration

package dbtest

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/db/catalog"
)

// TICK-733 (D-HB-9): the schema snapshot the handbook reads instead of a
// database. TestSchemaSnapshot takes it from a freshly migrated platform
// database and fails when the committed file differs; refresh it with
//
//	go test -tags integration ./internal/db/dbtest -run '^TestSchemaSnapshot$' -update
//
// (NAVA_TEST_PG_URL set as in helpers_test.go), or with task schema:snapshot,
// which starts the throwaway cluster itself.

//nolint:gochecknoglobals // the standard golden-file -update flag
var update = flag.Bool("update", false, "rewrite docs/handbook/src/database/schema.json")

// schemaSnapshotPath is the committed snapshot, from this package's directory.
const schemaSnapshotPath = "../../../docs/handbook/src/database/schema.json"

const refreshHint = "go test -tags integration ./internal/db/dbtest -run '^TestSchemaSnapshot$' -update (or task schema:snapshot)"

// updating is -update, or NAVA_SCHEMA_SNAPSHOT_UPDATE=1 for task
// schema:snapshot: scripts/ci/test-integration.sh passes its arguments to
// every package's test binary, and only this one defines -update.
func updating() bool { return *update || os.Getenv("NAVA_SCHEMA_SNAPSHOT_UPDATE") == "1" }

// checkGolden compares got with the file at path. With write it writes got
// there instead. A mismatch names the tables added or removed, or the first
// line that differs, and how to refresh the file.
func checkGolden(path string, got []byte, write bool) error {
	if write {
		return os.WriteFile(path, got, 0o644) //nolint:gosec // a committed docs file, world-readable like the rest of the tree
	}
	want, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("%w; write it with %s", err, refreshHint)
	}
	if bytes.Equal(want, got) {
		return nil
	}
	return fmt.Errorf("%s is stale (%s); refresh it with %s and commit it", filepath.ToSlash(path), snapshotDiff(want, got), refreshHint)
}

// snapshotDiff describes how got differs from want: the tables only one of
// them has, else the first line that differs.
func snapshotDiff(want, got []byte) string {
	names := func(b []byte) []string {
		var s catalog.Schema
		if json.Unmarshal(b, &s) != nil {
			return nil
		}
		var out []string
		for _, t := range s.Tables {
			out = append(out, catalog.RefName(t.Schema, t.Name))
		}
		return out
	}
	w, g := names(want), names(got)
	var parts []string
	if added := minus(g, w); len(added) > 0 {
		parts = append(parts, "tables not in the committed file: "+strings.Join(added, ", "))
	}
	if gone := minus(w, g); len(gone) > 0 {
		parts = append(parts, "tables no longer in the database: "+strings.Join(gone, ", "))
	}
	if len(parts) > 0 {
		return strings.Join(parts, "; ")
	}
	wl, gl := strings.Split(string(want), "\n"), strings.Split(string(got), "\n")
	for i := range max(len(wl), len(gl)) {
		var a, b string
		if i < len(wl) {
			a = wl[i]
		}
		if i < len(gl) {
			b = gl[i]
		}
		if a != b {
			return fmt.Sprintf("line %d: committed %q, migrated %q", i+1, strings.TrimSpace(a), strings.TrimSpace(b))
		}
	}
	return "same lines, different bytes"
}

func minus(a, b []string) []string {
	var out []string
	for _, x := range a {
		if !slices.Contains(b, x) {
			out = append(out, x)
		}
	}
	return out
}

// takeSnapshot is the platform database's snapshot, marshalled.
func takeSnapshot(t *testing.T) (*catalog.Schema, []byte) {
	t.Helper()
	s, err := catalog.TakeSnapshot(t.Context(), superNava(t))
	if err != nil {
		t.Fatal(err)
	}
	b, err := s.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	return s, b
}

func findTable(s *catalog.Schema, name string) *catalog.Table {
	for i := range s.Tables {
		if catalog.RefName(s.Tables[i].Schema, s.Tables[i].Name) == name {
			return &s.Tables[i]
		}
	}
	return nil
}

// The committed snapshot is what a fresh migrate produces. It has every
// table the gates discover, the runtime-built audit policies and the DEFAULT
// partition, none of the dropped objects, and no connection detail.
func TestSchemaSnapshot(t *testing.T) {
	migratedPlatform(t)
	s, b := takeSnapshot(t)

	// Deterministic: a second read is byte-for-byte the same.
	if _, again := takeSnapshot(t); !bytes.Equal(b, again) {
		t.Fatal("two snapshots of the same database differ")
	}

	// The same tables the catalogue gate and the isolation suite discover
	// (catalog.TenantTables) plus db/global-tables.yaml.
	tenant, err := catalog.TenantTables(t.Context(), superNava(t), globalTables(t))
	if err != nil {
		t.Fatal(err)
	}
	var want []string
	for _, tt := range tenant {
		want = append(want, catalog.RefName(tt.Schema, tt.Name))
	}
	for _, g := range globalTables(t).Names() {
		sch, n, _ := strings.Cut(g, ".")
		want = append(want, catalog.RefName(sch, n))
	}
	var got []string
	for _, tt := range s.Tables {
		got = append(got, catalog.RefName(tt.Schema, tt.Name))
	}
	if slices.Sort(want); !slices.Equal(sorted(got), want) {
		t.Errorf("snapshot tables %q, want the gate's %q", got, want)
	}

	// Dropped objects are history, not schema.
	if findTable(s, "audit_log_v0") != nil {
		t.Error("audit_log_v0 is in the snapshot; 20261104100000 drops it")
	}
	for _, tt := range s.Tables {
		for _, ix := range tt.Indexes {
			if slices.Contains([]string{"records_tenant_updated", "records_tenant_requester", "records_tenant_group_open"}, ix.Name) {
				t.Errorf("S9 list index %s is in the snapshot; 20261109140600 drops it", ix.Name)
			}
		}
	}

	// audit_log: partitioned, its DEFAULT partition, the runtime-built
	// policies (db/audited-tables.yaml), under ENABLE + FORCE.
	al := findTable(s, "audit_log")
	if al == nil || al.Kind != "partitioned" || al.PartitionKey == "" || !al.RLS.Enabled || !al.RLS.Forced {
		t.Fatalf("audit_log = %+v, want partitioned under ENABLE + FORCE RLS", al)
	}
	if d := findTable(s, "audit_log_default"); d == nil || d.Kind != "partition" || d.PartitionOf != "audit_log" || d.PartitionBound != "DEFAULT" {
		t.Errorf("audit_log_default = %+v, want the DEFAULT partition of audit_log", d)
	}
	gen := auditedManifest(t).Policies()
	if len(gen) == 0 {
		t.Fatal("db/audited-tables.yaml generates no audit_log policies")
	}
	for _, p := range gen {
		found := false
		for _, sp := range al.Policies {
			if sp.Name == p.Name {
				found = sp.Command == p.Cmd && strings.Join(sp.Roles, ",") == p.Roles && sp.Using == p.Qual && sp.WithCheck == p.Check
			}
		}
		if !found {
			t.Errorf("runtime-built policy %s (%+v) is missing or differs on audit_log: %+v", p.Name, p.Policy, al.Policies)
		}
	}

	// No connection detail: no DSN, password, host or pg_hba.conf address.
	cfg := superConfig(t)
	if leaks := catalog.SnapshotLeaks(b, cfg.Host, cfg.User, cfg.Password); len(leaks) > 0 {
		t.Fatalf("the snapshot holds connection details: %q", leaks)
	}

	if err := checkGolden(schemaSnapshotPath, b, updating()); err != nil {
		t.Fatal(err)
	}
}

// Fixture: a migration the committed file does not know about makes the
// check fail, naming the new table; -update rewrites the file, after which
// the same check passes.
func TestSchemaSnapshotFailsWhenStale(t *testing.T) {
	m := migratedCluster(t)
	// migrateOnlyFS: the fixture table is in no manifest, so the full
	// migrator run would stop on the catalogue gate before the snapshot.
	if err := migrateOnlyFS(t, m, withExtraMigration(t, "99999_t733_fixture.sql", `-- +goose Up
CREATE TABLE t733_fixture (tenant_id uuid NOT NULL);
ALTER TABLE t733_fixture ENABLE ROW LEVEL SECURITY;
ALTER TABLE t733_fixture FORCE ROW LEVEL SECURITY;
`)); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	_, b := takeSnapshot(t)

	err := checkGolden(schemaSnapshotPath, b, false)
	if err == nil || !strings.Contains(err.Error(), "tables not in the committed file: t733_fixture") ||
		!strings.Contains(err.Error(), "-update") {
		t.Fatalf("stale committed snapshot: err = %v, want it to name t733_fixture and -update", err)
	}

	committed, err := os.ReadFile(schemaSnapshotPath)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "schema.json")
	if err := os.WriteFile(path, committed, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := checkGolden(path, b, false); err == nil {
		t.Fatal("a copy of the committed file passed against the fixture database")
	}
	if err := checkGolden(path, b, true); err != nil {
		t.Fatal(err)
	}
	if err := checkGolden(path, b, false); err != nil {
		t.Fatalf("after -update: %v", err)
	}

	// A change inside a table, not a new one, names the first line.
	edited := bytes.Replace(b, []byte(`"enabled": true`), []byte(`"enabled": false`), 1)
	if err := os.WriteFile(path, edited, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := checkGolden(path, b, false); err == nil || !strings.Contains(err.Error(), "line ") {
		t.Fatalf("an edited flag: err = %v, want the first differing line", err)
	}

	// A missing file says how to write it.
	if err := checkGolden(filepath.Join(t.TempDir(), "absent.json"), b, false); err == nil || !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing file: err = %v", err)
	}
}

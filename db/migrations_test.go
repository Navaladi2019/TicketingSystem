package db

import (
	"io/fs"
	"regexp"
	"testing"
)

var migrationVersion = regexp.MustCompile(`^(\d+)_.+\.sql$`)

// TestMigrationVersionsUnique catches two migrations sharing a goose version.
// Git merges differently named files without a conflict, but goose refuses to
// run them, and the integration tests that migrate only run in CI.
func TestMigrationVersionsUnique(t *testing.T) {
	entries, err := fs.ReadDir(Migrations(), ".")
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]string{}
	for _, e := range entries {
		m := migrationVersion.FindStringSubmatch(e.Name())
		if m == nil {
			t.Errorf("migration %q does not match <version>_<name>.sql", e.Name())
			continue
		}
		if prev, dup := seen[m[1]]; dup {
			t.Errorf("duplicate migration version %s: %s and %s", m[1], prev, e.Name())
		}
		seen[m[1]] = e.Name()
	}
	if len(seen) == 0 {
		t.Error("no migrations found")
	}
}

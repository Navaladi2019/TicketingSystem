package db

import (
	"io/fs"
	"regexp"
	"strconv"
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
	seen := map[int64]string{}
	for _, e := range entries {
		m := migrationVersion.FindStringSubmatch(e.Name())
		if m == nil {
			t.Errorf("migration %q does not match <version>_<name>.sql", e.Name())
			continue
		}
		// Parse, so 6_x.sql and 00006_x.sql count as the same version.
		v, err := strconv.ParseInt(m[1], 10, 64)
		if err != nil {
			t.Errorf("migration %q: bad version: %v", e.Name(), err)
			continue
		}
		if prev, dup := seen[v]; dup {
			t.Errorf("duplicate migration version %d: %s and %s", v, prev, e.Name())
		}
		seen[v] = e.Name()
	}
	if len(seen) == 0 {
		t.Error("no migrations found")
	}
}

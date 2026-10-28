package db

import (
	"io/fs"
	"regexp"
	"strconv"
	"testing"
)

var migrationVersion = regexp.MustCompile(`^(\d+)_.+\.sql$`)

// TestMigrationVersionsAreUnique fails when two migrations share a goose
// version. Branches that each add a migration merge without a git conflict,
// since the file names differ, and goose then refuses to migrate. The version
// is compared as an integer, so 6_x.sql and 00006_x.sql count as the same.
func TestMigrationVersionsAreUnique(t *testing.T) {
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
		v, err := strconv.ParseInt(m[1], 10, 64)
		if err != nil {
			t.Errorf("migration %q: bad version: %v", e.Name(), err)
			continue
		}
		if prev, dup := seen[v]; dup {
			t.Errorf("migrations %s and %s share version %d", prev, e.Name(), v)
		}
		seen[v] = e.Name()
	}
	if len(seen) == 0 {
		t.Fatal("no migrations found")
	}
}

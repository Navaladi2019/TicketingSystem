package db

import (
	"io/fs"
	"strings"
	"testing"
)

// TestMigrationVersionsAreUnique fails when two migrations share a goose
// version. Branches that each add a migration merge without a git conflict,
// since the file names differ, and goose then refuses to migrate.
func TestMigrationVersionsAreUnique(t *testing.T) {
	entries, err := fs.ReadDir(Migrations(), ".")
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]string{}
	for _, e := range entries {
		version, _, ok := strings.Cut(e.Name(), "_")
		if !ok {
			t.Errorf("migration %s has no version prefix", e.Name())
			continue
		}
		if prev, dup := seen[version]; dup {
			t.Errorf("migrations %s and %s share version %s", prev, e.Name(), version)
		}
		seen[version] = e.Name()
	}
	if len(seen) == 0 {
		t.Fatal("no migrations found")
	}
}

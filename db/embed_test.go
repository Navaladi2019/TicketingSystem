package db

import (
	"io/fs"
	"strings"
	"testing"
)

func TestMustSubPanicsOnAnInvalidDirectory(t *testing.T) {
	t.Parallel()
	defer func() {
		r := recover()
		err, ok := r.(error)
		if !ok || !strings.Contains(err.Error(), "invalid") {
			t.Fatalf("recover() = %v, want fs.Sub's invalid-name error", r)
		}
	}()
	mustSub(migrations, "../outside")
	t.Fatal("mustSub returned for an invalid directory name")
}

func TestEmbeddedFilesArePresent(t *testing.T) {
	t.Parallel()
	names, err := fs.Glob(Migrations(), "*.sql")
	if err != nil || len(names) == 0 {
		t.Fatalf("Migrations() has no .sql files at its root: %v, %v", names, err)
	}
	for name, b := range map[string][]byte{"grants.yaml": Grants(), "global-tables.yaml": GlobalTables(), "pg_hba.conf": HBA()} {
		if len(b) == 0 {
			t.Errorf("%s is embedded empty", name)
		}
	}
}

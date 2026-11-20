// TICK-943 (S30 P11-11): legacyACLInsert, the helper that turns the
// tenant-id trigger off (acl_paths.go), is unexported in this package, whose
// non-test files carry no build tag. It is reachable from tests only as long
// as no other package of the module imports this one. This test fails the
// moment one does, a binary under cmd/ above all.

package isolation

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestLegacyACLInsertIsTestOnly(t *testing.T) {
	const self = "github.com/Navaladi2019/TicketingSystem/internal/tenancy/isolation"
	here, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(here, "..", "..", "..")
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("module root: %v", err)
	}
	// Every non-test .go file of the module, whatever its build tags: a file
	// that imports this package under any tag counts. Nested modules (their
	// own go.mod) are other modules and cannot import an internal package.
	var files int
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path == root {
				return nil
			}
			name := d.Name()
			if strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") || name == "testdata" || name == "vendor" || name == "node_modules" {
				return filepath.SkipDir
			}
			if _, err := os.Stat(filepath.Join(path, "go.mod")); err == nil {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		files++
		f, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, imp := range f.Imports {
			if p, _ := strconv.Unquote(imp.Path.Value); p == self && filepath.Dir(path) != here {
				t.Errorf("%s imports %s, so its code can reach legacyACLInsert", path, self)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if files < 300 {
		t.Fatalf("walked only %d non-test files", files)
	}
}

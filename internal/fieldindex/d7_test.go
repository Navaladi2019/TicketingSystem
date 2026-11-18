package fieldindex_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// D7 (ADR-006): a side table is read only as a correlated EXISTS inside the
// single list statement that carries the compiled row ACL. This test walks
// every non-test Go and SQL file of the module and fails on any mention of an
// rfv_* table outside the places that may have one:
//
//   - internal/fieldindex/fieldindex.go: the table names themselves;
//   - internal/server/records_list_index.go, inside existsTerm: the EXISTS;
//   - the trigger and the removal/backfill (P7-02, P7-05, P7-07): their
//     files are listed below when they land, by the task that adds them;
//   - the isolation suite's attack-path fixture (allowedFiles);
//   - db/migrations, docs and the spike scripts: DDL and notes, not queries.
//
// A new reader of a side table (a count, a facet, a progress query) fails here.
func TestD7SideTablesAreReadOnlyInTheListEXISTS(t *testing.T) {
	root := filepath.Join("..", "..")
	allowedFiles := map[string]bool{
		"internal/fieldindex/fieldindex.go": true,
		// The isolation suite's attack paths: adversarial SQL aimed at tenant
		// B's side rows, run only by the isolation tests.
		"internal/tenancy/isolation/fieldindex_paths.go": true,
		// The S10 attack paths' cleanup of the side rows of their own fixture.
		"internal/tenancy/isolation/acl_attack_paths.go": true,
	}
	skipDirs := map[string]bool{".git": true, "node_modules": true, "docs": true, "scratch": true, "sdk": true, "bench": true}
	var scanned, found int
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			if skipDirs[d.Name()] || rel == "db/migrations" || strings.HasSuffix(rel, "/testdata") {
				return filepath.SkipDir
			}
			return nil
		}
		isGo, isSQL := strings.HasSuffix(p, ".go") && !strings.HasSuffix(p, "_test.go"), strings.HasSuffix(p, ".sql")
		if !isGo && !isSQL {
			return nil
		}
		scanned++
		if isSQL {
			b, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			if strings.Contains(strings.ToLower(string(b)), "rfv_") {
				t.Errorf("%s reads or writes an rfv_* table; only the list EXISTS may (ADR-006 D7)", rel)
			}
			return nil
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, p, nil, 0)
		if err != nil {
			return err
		}
		var funcs []*ast.FuncDecl
		for _, decl := range f.Decls {
			if fd, ok := decl.(*ast.FuncDecl); ok {
				funcs = append(funcs, fd)
			}
		}
		ast.Inspect(f, func(n ast.Node) bool {
			lit, ok := n.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING || !strings.Contains(strings.ToLower(lit.Value), "rfv_") {
				return true
			}
			found++
			if allowedFiles[rel] {
				return true
			}
			if rel == "internal/server/records_list_index.go" {
				for _, fd := range funcs {
					if fd.Name.Name == "existsTerm" && lit.Pos() >= fd.Pos() && lit.End() <= fd.End() {
						if !strings.Contains(lit.Value, "EXISTS (SELECT 1 FROM") && !strings.Contains(lit.Value, "rfv_") {
							t.Errorf("%s: unexpected literal %s", rel, lit.Value)
						}
						return true
					}
				}
			}
			t.Errorf("%s:%d names an rfv_* table outside the list EXISTS (ADR-006 D7): %s", rel, fset.Position(lit.Pos()).Line, lit.Value)
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if scanned < 100 || found == 0 {
		t.Fatalf("scanned %d files and found %d rfv_ literals: the walk is not reaching the module", scanned, found)
	}
}

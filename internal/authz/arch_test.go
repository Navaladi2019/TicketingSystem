package authz_test

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

// The interim S9 authorizer let any tenant member read and write every record
// (AR-S9-01, PLAN-003 D-3-7). P3-31 deleted it; the CTO's condition (TICK-363)
// is that it can never come back. This test fails if any Go file of the
// module, test files and every build tag included, defines or references an
// identifier of that name, in any package and any form (type, value, field,
// selector).

// interimAuthorizer is the forbidden identifier, split so that this file
// does not hold it.
const interimAuthorizer = "TenantMember" + "Authorizer"

// interimUses returns the positions of interimAuthorizer in a Go source file.
func interimUses(fset *token.FileSet, path string, src []byte) ([]string, error) {
	f, err := parser.ParseFile(fset, path, src, parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}
	var at []string
	ast.Inspect(f, func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok && id.Name == interimAuthorizer {
			at = append(at, fset.Position(id.Pos()).String())
		}
		return true
	})
	return at, nil
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("no go.mod above the test")
		}
		dir = parent
	}
}

func TestInterimAuthorizerIsGone(t *testing.T) {
	root := moduleRoot(t)
	fset := token.NewFileSet()
	files := 0
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if name := d.Name(); path != root && (strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files++
		at, err := interimUses(fset, path, src)
		if err != nil {
			return err
		}
		for _, p := range at {
			t.Errorf("%s: %s is deleted (P3-31, AR-S9-01); use authz.ACL", p, interimAuthorizer)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if files < 100 {
		t.Fatalf("scanned only %d Go files under %s", files, root)
	}
}

// The check finds each form of the identifier, so it cannot rot into a test
// that passes on nothing.
func TestInterimAuthorizerCheckFindsEachForm(t *testing.T) {
	for name, src := range map[string]string{
		"type":      "package x\ntype " + interimAuthorizer + " struct{}",
		"selector":  "package x\nvar a = authz." + interimAuthorizer + "{}",
		"func":      "package x\nfunc " + interimAuthorizer + "() {}",
		"field":     "package x\ntype t struct{ " + interimAuthorizer + " int }",
		"tagged":    "//go:build never\n\npackage x\nvar _ = " + interimAuthorizer,
		"in a test": "package x_test\nfunc TestX() { _ = authz." + interimAuthorizer + "{} }",
	} {
		at, err := interimUses(token.NewFileSet(), name+".go", []byte(src))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if len(at) != 1 {
			t.Errorf("%s: found %d uses, want 1", name, len(at))
		}
	}
	at, _ := interimUses(token.NewFileSet(), "clean.go", []byte("package x\n// a comment naming it is history, not a use\nvar s = \"x\""))
	if len(at) != 0 {
		t.Errorf("a clean file: %v", at)
	}
}

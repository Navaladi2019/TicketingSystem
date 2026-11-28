package server

import (
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// listSQLHooks are the options fields that replace SQL the list route renders
// inside the tenant transaction (I5 mutants, TICK-1039 spike arms). A shipped
// binary that could set one would let a caller swap the D7 EXISTS, with its
// v.tenant_id = r.tenant_id predicate, for any term.
var listSQLHooks = map[string]bool{"listDataTerm": true, "legacyScan": true, "listIndexTerm": true} //nolint:gochecknoglobals // read-only test fixture

// TestListSQLHooksUnsettableInShippedBuild parses every non-test file of this
// package that the default build (no tags, what deploy/compose/*/Dockerfile
// compiles) includes, and fails if any of them assigns a list SQL hook on
// options. Only the integration-tagged With* setters may.
func TestListSQLHooksUnsettableInShippedBuild(t *testing.T) {
	ctx := build.Default
	ctx.BuildTags = nil
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	seen := 0
	for _, de := range entries {
		name := de.Name()
		if de.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		ok, err := ctx.MatchFile(".", name)
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			continue
		}
		seen++
		f, err := parser.ParseFile(fset, filepath.Join(".", name), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			as, isAssign := n.(*ast.AssignStmt)
			if !isAssign {
				return true
			}
			for _, lhs := range as.Lhs {
				sel, isSel := lhs.(*ast.SelectorExpr)
				if isSel && listSQLHooks[sel.Sel.Name] {
					t.Errorf("%s: assigns %s in a file the shipped build compiles; only an integration-tagged With* option may set it",
						fset.Position(as.Pos()), sel.Sel.Name)
				}
			}
			return true
		})
	}
	if seen == 0 {
		t.Fatal("no default-build source files matched; the guard checked nothing")
	}
}

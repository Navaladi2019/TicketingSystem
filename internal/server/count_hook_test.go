package server

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

// listQueryHooks are the ListQuery test seams.
func listQueryHook(name string) bool {
	switch name {
	case "CountDelay", "SkipCount", "CountBudgetOverride", "CountParallel":
		return true
	}
	return false
}

// ADR-006-A2 D-7A2-5 (3d): the count delay and skip hooks are test-only. The
// handlers fields are set from options, which only export_test.go's
// WithCountDelay / WithSkipCount fill, and store.ListQuery.CountDelay and
// SkipCount are set in non-test code only from those handlers fields.
func TestCountHooksAreTestOnly(t *testing.T) {
	fset := token.NewFileSet()
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	hooks := map[string]bool{"countDelay": true, "skipCount": true, "countBudgetOverride": true, "countParallel": true}
	sawListQuery := 0
	for _, name := range files {
		f, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		isTest := strings.HasSuffix(name, "_test.go")
		ast.Inspect(f, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.AssignStmt:
				for i, lhs := range x.Lhs {
					sel, ok := lhs.(*ast.SelectorExpr)
					if !ok {
						continue
					}
					field := sel.Sel.Name
					switch {
					case hooks[field] && !isTest:
						t.Errorf("%s: %s assigns the %s hook in non-test code", fset.Position(x.Pos()), name, field)
					case hooks[field] && name != "export_test.go":
						t.Errorf("%s: the %s hook is assigned outside export_test.go", fset.Position(x.Pos()), field)
					case listQueryHook(field) && !isTest:
						// lq.CountDelay, lq.SkipCount = h.countDelay, h.skipCount
						if len(x.Lhs) != len(x.Rhs) {
							t.Errorf("%s: %s assigned from a call", fset.Position(x.Pos()), field)
							continue
						}
						want := strings.ToLower(field[:1]) + field[1:]
						rhs, ok := x.Rhs[i].(*ast.SelectorExpr)
						if !ok || rhs.Sel.Name != want {
							t.Errorf("%s: ListQuery.%s must be set from handlers.%s only", fset.Position(x.Pos()), field, want)
							continue
						}
						if recv, rok := rhs.X.(*ast.Ident); !rok || recv.Name != "h" {
							t.Errorf("%s: ListQuery.%s must be set from the handler's own field", fset.Position(x.Pos()), field)
							continue
						}
						sawListQuery++
					}
				}
			case *ast.KeyValueExpr:
				id, ok := x.Key.(*ast.Ident)
				if !ok || isTest {
					return true
				}
				if hooks[id.Name] {
					// handlers{countDelay: opts.countDelay}: the one copy from the options.
					v, vok := x.Value.(*ast.SelectorExpr)
					base, bok := ast.Expr(nil), false
					if vok {
						base, bok = v.X, true
					}
					if id2, iok := base.(*ast.Ident); !bok || !iok || id2.Name != "opts" || v.Sel.Name != id.Name {
						t.Errorf("%s: %s set from something other than opts.%s", fset.Position(x.Pos()), id.Name, id.Name)
					}
				}
				if listQueryHook(id.Name) {
					t.Errorf("%s: ListQuery.%s set in a literal in non-test code", fset.Position(x.Pos()), id.Name)
				}
			}
			return true
		})
	}
	if sawListQuery != 4 {
		t.Errorf("found %d ListQuery hook assignments in non-test code, want 4 (the list handler)", sawListQuery)
	}
}

// The whole module's non-test code: ListQuery.CountDelay and SkipCount are
// assigned (by selector or keyed literal) only in the list handler, from the
// handlers fields. A write in internal/records/pipeline or any other package
// would reach production through the same query.
func TestListQueryCountHooksAreSetOnlyByTheListHandler(t *testing.T) {
	root := filepath.Join("..", "..")
	handler := filepath.Join(root, "internal", "server", "records_list.go")
	fset := token.NewFileSet()
	assigned := 0
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "node_modules", "vendor":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return nil // not this test's business
		}
		ast.Inspect(f, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.AssignStmt:
				for i, lhs := range x.Lhs {
					sel, ok := lhs.(*ast.SelectorExpr)
					if !ok || !listQueryHook(sel.Sel.Name) {
						continue
					}
					want := strings.ToLower(sel.Sel.Name[:1]) + sel.Sel.Name[1:]
					var rhs *ast.SelectorExpr
					if len(x.Lhs) == len(x.Rhs) {
						rhs, _ = x.Rhs[i].(*ast.SelectorExpr)
					}
					if path != handler || rhs == nil || rhs.Sel.Name != want {
						t.Errorf("%s: %s assigned in non-test code other than the list handler's copy from handlers.%s", fset.Position(x.Pos()), sel.Sel.Name, want)
						continue
					}
					assigned++
				}
			case *ast.KeyValueExpr:
				if id, ok := x.Key.(*ast.Ident); ok && listQueryHook(id.Name) {
					t.Errorf("%s: %s set in a keyed literal in non-test code", fset.Position(x.Pos()), id.Name)
				}
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if assigned != 4 {
		t.Errorf("found %d handler copies, want 4 (CountDelay, SkipCount, CountBudgetOverride, CountParallel in records_list.go)", assigned)
	}
}

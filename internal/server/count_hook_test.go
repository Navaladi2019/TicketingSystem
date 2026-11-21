package server

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
)

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
	hooks := map[string]bool{"countDelay": true, "skipCount": true}
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
					case (field == "CountDelay" || field == "SkipCount") && !isTest:
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
				if id.Name == "CountDelay" || id.Name == "SkipCount" {
					t.Errorf("%s: ListQuery.%s set in a literal in non-test code", fset.Position(x.Pos()), id.Name)
				}
			}
			return true
		})
	}
	if sawListQuery != 2 {
		t.Errorf("found %d ListQuery hook assignments in non-test code, want 2 (the list handler)", sawListQuery)
	}
}

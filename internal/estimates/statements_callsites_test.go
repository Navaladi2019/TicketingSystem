package estimates

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TICK-1071 adversarial (Security & Multi-Tenancy Reviewer), hardening
// acceptance 5. TestEveryStatementButTheListingRunsInATenantTx checks the
// order inside refresh; this checks that refresh and tenants are the only
// places the package talks to the database at all, so a new helper that runs
// SQL without set_config, or SQL built inline, fails here:
//
//   - Tenants issues exactly one statement, listSQL;
//   - refresh never issues the listing (the order inside it is the other
//     test's job);
//   - no other function, in any non-test file of the package, calls a
//     database method.
func TestOnlyTenantsAndRefreshIssueStatements(t *testing.T) {
	dbMethods := map[string]bool{
		"Exec": true, "Query": true, "QueryRow": true, "SendBatch": true,
		"CopyFrom": true, "Prepare": true, "LargeObjects": true,
	}
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	calls := map[string][]*ast.CallExpr{} // function -> its database calls
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		src, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		f, err := parser.ParseFile(fset, name, src, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			fn, ok := n.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				return true
			}
			ast.Inspect(fn.Body, func(m ast.Node) bool {
				if call, ok := m.(*ast.CallExpr); ok {
					if sel, ok := call.Fun.(*ast.SelectorExpr); ok && dbMethods[sel.Sel.Name] {
						calls[fn.Name.Name] = append(calls[fn.Name.Name], call)
					}
				}
				return true
			})
			return false
		})
	}

	for fn, cs := range calls {
		switch fn {
		case "Tenants":
			if len(cs) != 1 || !argIs(cs[0], 1, "listSQL") {
				t.Errorf("Tenants issues %d statements; want exactly one, listSQL", len(cs))
			}
		case "refresh":
			for _, c := range cs {
				if argIs(c, 1, "listSQL") {
					t.Error("refresh issues the tenant listing")
				}
			}
		default:
			t.Errorf("%s calls a database method; only Tenants and refresh may", fn)
		}
	}
	if len(calls["Tenants"]) == 0 || len(calls["refresh"]) == 0 {
		t.Errorf("database calls = %v; want them in Tenants and refresh", calls)
	}
}

// argIs reports whether call's i-th argument is the identifier name.
func argIs(call *ast.CallExpr, i int, name string) bool {
	if len(call.Args) <= i {
		return false
	}
	id, ok := call.Args[i].(*ast.Ident)
	return ok && id.Name == name
}

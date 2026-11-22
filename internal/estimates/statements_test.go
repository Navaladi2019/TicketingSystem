package estimates

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"strings"
	"testing"
)

// TICK-1071 acceptance 5: every statement the package issues, except the
// tenant listing, runs inside a tenancy.InTenantTx callback, so after
// the tenant setting in the same transaction; the package sets
// no tenant itself (navalint/setconfig). Read from the source: the SQL
// constants, and the one place a tenant's statements are issued.
func TestEveryStatementButTheListingRunsInATenantTx(t *testing.T) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "estimates.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	sql := map[string]string{}
	var inTenantCalls, execs int
	var execArgs []string
	ast.Inspect(f, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.ValueSpec:
			for i, v := range n.Values {
				if lit, ok := v.(*ast.BasicLit); ok && lit.Kind == token.STRING && strings.HasSuffix(n.Names[i].Name, "SQL") {
					s, _ := strconv.Unquote(lit.Value)
					sql[n.Names[i].Name] = s
				}
			}
		case *ast.FuncDecl:
			if n.Name.Name != "refresh" {
				return true
			}
			ast.Inspect(n, func(m ast.Node) bool {
				switch m := m.(type) {
				case *ast.CallExpr:
					if sel, ok := m.Fun.(*ast.SelectorExpr); ok {
						switch sel.Sel.Name {
						case "InTenant":
							inTenantCalls++
						case "Exec":
							execs++
						}
					}
				case *ast.CompositeLit:
					for _, e := range m.Elts {
						if id, ok := e.(*ast.Ident); ok {
							execArgs = append(execArgs, id.Name)
						}
					}
				}
				return true
			})
		}
		return true
	})
	if len(sql) != 3 {
		t.Fatalf("SQL constants = %v, want listSQL, refreshSQL, zeroSQL", sql)
	}
	if inTenantCalls != 1 || execs != 1 || strings.Join(execArgs, ",") != "refreshSQL,zeroSQL" {
		t.Errorf("refresh: %d InTenant calls, %d Exec calls, statements %v; want its statements run inside one InTenant callback",
			inTenantCalls, execs, execArgs)
	}
	for name, s := range sql {
		if name != "listSQL" && strings.Contains(s, "tenant_identity_binding") {
			t.Errorf("%s reads tenant_identity_binding", name)
		}
	}
	if sql["listSQL"] != "SELECT DISTINCT tenant_id FROM tenant_identity_binding" {
		t.Errorf("listSQL = %q", sql["listSQL"])
	}
}

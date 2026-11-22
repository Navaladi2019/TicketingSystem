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
// tenant listing, runs after set_config('app.tenant_id', ...) in the same
// transaction. Read from the source: the SQL constants, and the one place a
// tenant's statements are issued.
func TestEveryStatementButTheListingFollowsSetConfig(t *testing.T) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "estimates.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	sql := map[string]string{}
	var order []string
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
				if cl, ok := m.(*ast.CompositeLit); ok {
					for _, e := range cl.Elts {
						if id, ok := e.(*ast.Ident); ok {
							order = append(order, id.Name)
						}
					}
				}
				return true
			})
		}
		return true
	})
	if len(sql) != 4 {
		t.Fatalf("SQL constants = %v, want listSQL, setTenantSQL, refreshSQL, zeroSQL", sql)
	}
	if len(order) == 0 || order[0] != "setTenantSQL" {
		t.Errorf("refresh issues %v, want set_config first", order)
	}
	if !strings.Contains(sql["setTenantSQL"], "set_config('app.tenant_id', $1, true)") {
		t.Errorf("setTenantSQL = %q, want a transaction-local set_config", sql["setTenantSQL"])
	}
	for _, name := range order {
		if name == "listSQL" {
			t.Error("refresh issues the tenant listing")
		}
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

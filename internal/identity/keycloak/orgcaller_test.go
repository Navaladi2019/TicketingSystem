package keycloak

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
)

// TestPlaceInviteeIsTheOnlyRuntimeOrganizationsCaller guards the group sync
// account's view-organizations role (TICK-252, Security ruling): it reads every
// organization realm-wide, so the only code that may call an /organizations
// path with the runtime account is PlaceInvitee. Calls with the onboarding
// account (CreateTenantOrg, FindTenantOrg) are not the runtime account.
func TestPlaceInviteeIsTheOnlyRuntimeOrganizationsCaller(t *testing.T) {
	t.Parallel()
	files, err := filepath.Glob("*.go")
	if err != nil || len(files) == 0 {
		t.Fatalf("glob: %v %d", err, len(files))
	}
	fset := token.NewFileSet()
	var found int
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, f, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				var org, runtime bool
				for _, a := range call.Args {
					ast.Inspect(a, func(m ast.Node) bool {
						switch v := m.(type) {
						case *ast.BasicLit:
							if v.Kind == token.STRING && strings.Contains(v.Value, "/organizations") {
								org = true
							}
						case *ast.SelectorExpr:
							if v.Sel.Name == "runtime" {
								runtime = true
							}
						}
						return true
					})
				}
				if org && runtime {
					found++
					if fn.Name.Name != "PlaceInvitee" {
						t.Errorf("%s: %s calls an /organizations path with the runtime account", fset.Position(call.Pos()), fn.Name.Name)
					}
				}
				return true
			})
		}
	}
	if found == 0 {
		t.Error("guard found no runtime /organizations call: PlaceInvitee moved or the guard is blind")
	}
}

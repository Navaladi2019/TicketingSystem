// Package idpclaims defines an analyzer that reports reads of
// Keycloak-specific token claims, and calls to the Keycloak admin API,
// outside internal/identity/keycloak (D-P6, CI-11). The identity port is the
// only code that may know Keycloak's token shape or admin API; anything else
// gets a tenant or operator from identity.IdentityResolver or
// identity.OperatorResolver.
//
// It reports:
//   - a string constant equal to a Keycloak claim name (organization, idp,
//     realm_access, resource_access), as in claims["organization"] or
//     tok.Get("idp", &v);
//   - a struct field whose json tag names one of those claims;
//   - a string constant holding the admin API path /admin/realms.
//
// The _test.go files of internal/identitystack are exempt: they are the
// realm export's own acceptance tests and read raw tokens and the admin API
// on purpose (T14). Its non-test files are not.
//
// Like setconfig, it folds constant expressions but cannot see strings built
// at run time; it is defence in depth next to depguard and the contract suite
// (TICK-94 item 15).
package idpclaims

import (
	"go/ast"
	"go/constant"
	"go/token"
	"reflect"
	"strconv"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// AdapterPkg is the Keycloak adapter; it and the packages below it may use
// the claims and the admin API.
const AdapterPkg = "github.com/Navaladi2019/TicketingSystem/internal/identity/keycloak"

// StackTestPkg is the package whose _test.go files are exempt.
const StackTestPkg = "github.com/Navaladi2019/TicketingSystem/internal/identitystack"

const adminPath = "/admin/realms"

// Analyzer reports Keycloak claim reads and admin API calls outside AdapterPkg.
var Analyzer = &analysis.Analyzer{
	Name: "idpclaims",
	Doc:  "reports Keycloak-specific claims and admin API calls outside internal/identity/keycloak; use the identity port",
	Run:  run,
}

func isClaim(s string) bool {
	switch s {
	case "organization", "idp", "realm_access", "resource_access":
		return true
	}
	return false
}

func run(pass *analysis.Pass) (any, error) {
	path := strings.TrimSuffix(pass.Pkg.Path(), "_test")
	if path == AdapterPkg || strings.HasPrefix(path, AdapterPkg+"/") {
		return nil, nil
	}
	for _, f := range pass.Files {
		if path == StackTestPkg && strings.HasSuffix(pass.Fset.File(f.Pos()).Name(), "_test.go") {
			continue
		}
		ast.Inspect(f, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.Field:
				checkTag(pass, n)
			case *ast.BasicLit, *ast.BinaryExpr, *ast.ParenExpr:
				return checkConst(pass, n.(ast.Expr))
			}
			return true
		})
	}
	return nil, nil
}

// checkConst reports a string constant naming a claim or the admin API. The
// outermost constant is checked once; its parts are not checked again.
func checkConst(pass *analysis.Pass, e ast.Expr) bool {
	tv, ok := pass.TypesInfo.Types[e]
	if !ok || tv.Value == nil || tv.Value.Kind() != constant.String {
		return true
	}
	s := constant.StringVal(tv.Value)
	switch {
	case isClaim(s):
		reportClaim(pass, e.Pos(), s)
	case strings.Contains(strings.ToLower(s), adminPath):
		pass.Reportf(e.Pos(), "Keycloak admin API outside internal/identity/keycloak: use identity.DirectoryAdmin")
	}
	return false
}

// checkTag reports a struct field that decodes a claim by its json tag.
func checkTag(pass *analysis.Pass, f *ast.Field) {
	if f.Tag == nil {
		return
	}
	tag, err := strconv.Unquote(f.Tag.Value)
	if err != nil {
		return
	}
	name, _, _ := strings.Cut(reflect.StructTag(tag).Get("json"), ",")
	if isClaim(name) {
		reportClaim(pass, f.Tag.Pos(), name)
	}
}

func reportClaim(pass *analysis.Pass, pos token.Pos, claim string) {
	pass.Reportf(pos, "Keycloak claim %q read outside internal/identity/keycloak: use the identity port", claim)
}

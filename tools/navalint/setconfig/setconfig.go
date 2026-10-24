// Package setconfig defines an analyzer that reports set_config and
// app.tenant_id in Go string constants outside internal/tenancy (PG-N1, R8).
// tenancy.InTenantTx is the one place that scopes a transaction to a tenant;
// a second place could set another tenant's id, or a session-wide one that
// outlives the transaction on a pooled connection.
//
// SQL files are not Go source and are not checked: the row-level security
// policies under db/ read current_setting('app.tenant_id') by design.
//
// The analyzer folds constant expressions, so "set_" + "config" is caught,
// but a string built at run time is not. It is defence in depth; the
// database grants are the control (TICK-94 item 15).
package setconfig

import (
	"go/ast"
	"go/constant"
	"go/token"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// TenancyPkg is the one package allowed to use the needles, with its tests.
const TenancyPkg = "github.com/Navaladi2019/TicketingSystem/internal/tenancy"

// needles are matched case-insensitively: SQL keywords and function names
// are, and quoted "app.tenant_id" is the same setting.
var needles = []string{"set_config", "app.tenant_id"}

// Analyzer reports the needles in string constants outside TenancyPkg.
var Analyzer = &analysis.Analyzer{
	Name: "setconfig",
	Doc:  "reports set_config and app.tenant_id outside internal/tenancy; use tenancy.InTenantTx",
	Run:  run,
}

func run(pass *analysis.Pass) (any, error) {
	if path := strings.TrimSuffix(pass.Pkg.Path(), "_test"); path == TenancyPkg {
		return nil, nil
	}
	for _, f := range pass.Files {
		ast.Inspect(f, func(n ast.Node) bool {
			e, ok := n.(ast.Expr)
			if !ok {
				return true
			}
			switch e.(type) {
			case *ast.BasicLit, *ast.BinaryExpr, *ast.ParenExpr:
			default:
				return true
			}
			tv, ok := pass.TypesInfo.Types[e]
			if !ok || tv.Value == nil || tv.Value.Kind() != constant.String {
				return true
			}
			if needle := match(constant.StringVal(tv.Value)); needle != "" {
				report(pass, e.Pos(), needle)
			}
			// The outermost constant is reported once; its parts are not
			// checked again.
			return false
		})
	}
	return nil, nil
}

func match(s string) string {
	s = strings.ToLower(s)
	for _, n := range needles {
		if strings.Contains(s, n) {
			return n
		}
	}
	return ""
}

func report(pass *analysis.Pass, pos token.Pos, needle string) {
	pass.Reportf(pos, "%s outside internal/tenancy: scope the transaction with tenancy.InTenantTx", needle)
}

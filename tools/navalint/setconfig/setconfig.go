// Package setconfig defines an analyzer that reports set_config and
// app.tenant_id in Go string constants outside internal/tenancy (PG-N1, R8).
// tenancy.InTenantTx is the one place that scopes a transaction to a tenant;
// a second place could set another tenant's id, or a session-wide one that
// outlives the transaction on a pooled connection.
//
// The _test.go files of internal/db/dbtest are also exempt: the RLS policy
// tests set app.tenant_id directly to test the policies themselves, including
// set_config(..., false) leaks (TICK-61 decision 2026-10-24). Its non-test
// files, and any other package named dbtest, are not. The _test.go files of
// internal/tenancy/isolation are exempt for the same reason: the CI-2 suite
// (T11, TICK-67) sets no tenant, an empty one and a prior one by hand to prove each
// gives 0 rows. Its registry (registry.go) is not exempt; registered paths
// scope their transactions with tenancy.InTenantTx.
//
// internal/db/catalog may read the setting, nothing more: the catalogue gate
// compares every policy against the deparsed predicate
// current_setting('app.tenant_id'::text, true) (T10, TICK-149). In that one
// package a constant is checked with its current_setting('app.tenant_id')
// reads removed, so set_config and a SET of app.tenant_id are still reported,
// in the same constant or alone. Reads anywhere else are still reported.
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
	"regexp"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// TenancyPkg is the one package allowed to use the needles, with its tests.
const TenancyPkg = "github.com/Navaladi2019/TicketingSystem/internal/tenancy"

// PolicyTestPkg is the package whose _test.go files may use the needles.
const PolicyTestPkg = "github.com/Navaladi2019/TicketingSystem/internal/db/dbtest"

// IsolationTestPkg is the CI-2 isolation suite, whose _test.go files may use
// them.
const IsolationTestPkg = "github.com/Navaladi2019/TicketingSystem/internal/tenancy/isolation"

// CatalogPkg is the package, with its tests, that may read app.tenant_id
// with current_setting.
const CatalogPkg = "github.com/Navaladi2019/TicketingSystem/internal/db/catalog"

// tenantRead is a current_setting read of app.tenant_id as written by hand or
// deparsed by PostgreSQL, matched against the lowered constant.
var tenantRead = regexp.MustCompile(`current_setting\s*\(\s*'app\.tenant_id'(::text)?`)

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
	path := strings.TrimSuffix(pass.Pkg.Path(), "_test")
	if path == TenancyPkg {
		return nil, nil
	}
	for _, f := range pass.Files {
		if (path == PolicyTestPkg || path == IsolationTestPkg) && strings.HasSuffix(pass.Fset.File(f.Pos()).Name(), "_test.go") {
			continue
		}
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
			if needle := match(constant.StringVal(tv.Value), path == CatalogPkg); needle != "" {
				report(pass, e.Pos(), needle)
			}
			// The outermost constant is reported once; its parts are not
			// checked again.
			return false
		})
	}
	return nil, nil
}

// match returns the first needle in s; with reads, current_setting reads of
// app.tenant_id are not counted.
func match(s string, reads bool) string {
	s = strings.ToLower(s)
	if reads {
		s = tenantRead.ReplaceAllString(s, "")
	}
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

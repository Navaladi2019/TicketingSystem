// Package tenancyconn defines an analyzer that reports .Conn() and
// .LargeObjects() on a pgx.Tx outside internal/tenancy (TICK-698, findings on
// PR #570).
//
// A transaction from tenancy.InTenantTx with BeginInFirstBatch sends no BEGIN
// until fn sends something (TICK-692, internal/tenancy/firstbatch.go):
//   - Until then its Conn() is a pooled connection with no transaction open
//     and no tenant set: a statement sent on it runs outside the transaction,
//     in autocommit, with no app.tenant_id. Row-level security fails closed,
//     so today that is a failed query rather than a leak, but the connection
//     is not the transaction and must not be used as one.
//   - Its LargeObjects() panics: pgx cannot build an error-valued
//     pgx.LargeObjects without unsafe. Reporting the call turns that runtime
//     panic into a lint failure. Nothing in the platform uses large objects.
//
// InTenantTx hands fn a pgx.Tx, and the helpers fn calls take a pgx.Tx, so the
// static type is the one signal that survives a call: the analyzer reports
// either method on every value whose type is pgx.Tx, called or taken as a
// method value. A pgxpool.Conn's Conn(), or any other type's, is not reported.
//
// Exempt: internal/tenancy and the packages under it (tenancytest, the CI-2
// isolation harness), which own the transaction, and every _test.go file.
package tenancyconn

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// TenancyPkg is the package that, with the packages under it, may call
// Conn() and LargeObjects() on a pgx.Tx.
const TenancyPkg = "github.com/Navaladi2019/TicketingSystem/internal/tenancy"

const pgxPkg = "github.com/jackc/pgx/v5"

// Analyzer reports .Conn() and .LargeObjects() on a pgx.Tx outside
// internal/tenancy.
var Analyzer = &analysis.Analyzer{
	Name: "tenancyconn",
	Doc:  "reports .Conn() and .LargeObjects() on a pgx.Tx outside internal/tenancy and _test.go files: a BeginInFirstBatch transaction's connection has no transaction and no tenant until its first batch, and its LargeObjects panics (TICK-698)",
	Run:  run,
}

// msgs are the reports, by method.
var msgs = map[string]string{
	"Conn":         ".Conn() on a pgx.Tx outside internal/tenancy: before its first batch a tenancy.InTenantTx transaction's connection has no transaction and no tenant; use the transaction's own methods (TICK-698)",
	"LargeObjects": ".LargeObjects() on a pgx.Tx outside internal/tenancy: large objects are not supported on tenant transactions (TICK-698)",
}

func run(pass *analysis.Pass) (any, error) {
	path := strings.TrimSuffix(pass.Pkg.Path(), "_test")
	if path == TenancyPkg || strings.HasPrefix(path, TenancyPkg+"/") {
		return nil, nil
	}
	for _, f := range pass.Files {
		if strings.HasSuffix(pass.Fset.File(f.Pos()).Name(), "_test.go") {
			continue
		}
		ast.Inspect(f, func(n ast.Node) bool {
			sel, ok := n.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			msg, banned := msgs[sel.Sel.Name]
			if !banned {
				return true
			}
			s, ok := pass.TypesInfo.Selections[sel]
			if ok && s.Kind() == types.MethodVal && isPgxTx(s.Recv()) {
				pass.Reportf(sel.Sel.Pos(), "%s", msg)
			}
			return true
		})
	}
	return nil, nil
}

// isPgxTx reports whether t is pgx.Tx itself, not a type that implements it.
func isPgxTx(t types.Type) bool {
	named, ok := types.Unalias(t).(*types.Named)
	if !ok {
		return false
	}
	obj := named.Obj()
	return obj.Name() == "Tx" && obj.Pkg() != nil && obj.Pkg().Path() == pgxPkg
}

// Package guarderrtype defines an analyzer that reports uses of the worker
// guard's allowed failure types outside internal/workerguard (AR-044, P3-26).
//
// The guard records the message of an ApplicationError unchanged only when
// its type is ErrTypeTenantGuard ("nava.TenantGuard") or ErrTypeWorkflowRef
// ("nava.WorkflowRefNotFound"), because only the guard builds those, with
// fixed text or ids. Workflow or activity code that built one could put
// record data in its message, and the history would keep it. So outside
// internal/workerguard it reports:
//   - any reference to workerguard.ErrTypeTenantGuard or
//     workerguard.ErrTypeWorkflowRef;
//   - a string constant equal to either value, as in
//     temporal.NewApplicationError(msg, "nava."+"TenantGuard").
//
// _test.go files are exempt: they assert the type of the guard's errors and
// never build into a worker. Like idpclaims, it folds constant expressions
// but cannot see strings built at run time.
package guarderrtype

import (
	"go/ast"
	"go/constant"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// GuardPkg is the worker guard; it alone may use the types.
const GuardPkg = "github.com/Navaladi2019/TicketingSystem/internal/workerguard"

// Analyzer reports the guard's allowed failure types outside GuardPkg.
var Analyzer = &analysis.Analyzer{
	Name: "guarderrtype",
	Doc:  "reports the worker guard's allowed failure types (nava.TenantGuard, nava.WorkflowRefNotFound) outside internal/workerguard (AR-044)",
	Run:  run,
}

const msg = "outside internal/workerguard: only the worker guard builds this failure type (AR-044)"

func isGuardName(name string) bool {
	return name == "ErrTypeTenantGuard" || name == "ErrTypeWorkflowRef"
}

func isGuardValue(s string) bool {
	return s == "nava.TenantGuard" || s == "nava.WorkflowRefNotFound"
}

func run(pass *analysis.Pass) (any, error) {
	if strings.TrimSuffix(pass.Pkg.Path(), "_test") == GuardPkg {
		return nil, nil
	}
	for _, f := range pass.Files {
		if strings.HasSuffix(pass.Fset.File(f.Pos()).Name(), "_test.go") {
			continue
		}
		ast.Inspect(f, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.Ident:
				checkUse(pass, n)
			case *ast.BasicLit, *ast.BinaryExpr, *ast.ParenExpr:
				return checkConst(pass, n.(ast.Expr))
			}
			return true
		})
	}
	return nil, nil
}

// checkUse reports an identifier that refers to one of the guard's constants.
func checkUse(pass *analysis.Pass, id *ast.Ident) {
	obj := pass.TypesInfo.Uses[id]
	if obj == nil || obj.Pkg() == nil || obj.Pkg().Path() != GuardPkg || !isGuardName(obj.Name()) {
		return
	}
	pass.Reportf(id.Pos(), "workerguard.%s used %s", obj.Name(), msg)
}

// checkConst reports a string constant equal to a guard type. The outermost
// constant is checked once; its parts are not checked again.
func checkConst(pass *analysis.Pass, e ast.Expr) bool {
	tv, ok := pass.TypesInfo.Types[e]
	if !ok || tv.Value == nil || tv.Value.Kind() != constant.String {
		return true
	}
	if s := constant.StringVal(tv.Value); isGuardValue(s) {
		pass.Reportf(e.Pos(), "failure type %q %s", s, msg)
	}
	return false
}

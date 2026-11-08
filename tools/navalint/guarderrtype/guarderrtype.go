// Package guarderrtype defines an analyzer that reports uses of the worker
// guard's allowed failure types outside the packages that may build them
// (AR-044, P3-26, TICK-594).
//
// The guard records the message of an ApplicationError unchanged only when
// its type is on its allow-list, because only trusted code builds those, with
// fixed text or ids:
//   - ErrTypeTenantGuard ("nava.TenantGuard"), ErrTypeWorkflowRef
//     ("nava.WorkflowRefNotFound") and ErrTypeWorkflowRejected
//     ("nava.WorkflowRejected", built only by workerguard.Rejected, P3-22a),
//     which only internal/workerguard may use;
//   - ErrTypeAuditSealUnsealable ("nava.audit.seal.Unsealable") and
//     ErrTypeAuditSealHeadNotFound ("nava.audit.seal.HeadNotFound"), which
//     internal/workerguard and internal/audit/seal may use (TICK-585).
//
// Workflow or activity code that built one could put record data in its
// message, and the history would keep it. So the analyzer reports:
//   - outside the allowed packages, any reference to one of these constants
//     (workerguard's, or seal's re-exports of the audit.seal types) and any
//     string constant equal to one of their values, as in
//     temporal.NewApplicationError(msg, "nava."+"TenantGuard");
//   - inside internal/audit/seal, any call that passes an audit.seal type
//     after its first argument while that first argument, the message, is
//     not a constant, as in
//     temporal.NewNonRetryableApplicationError("seal: "+err.Error(), ErrTypeUnsealable, nil).
//
// _test.go files are exempt: they assert the type of the guard's errors and
// never build into a worker. Like idpclaims, it folds constant expressions
// but cannot see strings built at run time, so a type passed through a
// variable is not seen.
package guarderrtype

import (
	"go/ast"
	"go/constant"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// GuardPkg is the worker guard; it alone may use the guard's own types.
const GuardPkg = "github.com/Navaladi2019/TicketingSystem/internal/workerguard"

// SealPkg is the audit sealer; it may build the audit.seal types, with a
// constant message only (TICK-594).
const SealPkg = "github.com/Navaladi2019/TicketingSystem/internal/audit/seal"

// Analyzer reports the guard's allowed failure types outside the packages
// that may build them.
var Analyzer = &analysis.Analyzer{
	Name: "guarderrtype",
	Doc:  "reports the worker guard's allowed failure types (nava.TenantGuard, nava.WorkflowRefNotFound, nava.WorkflowRejected, nava.audit.seal.Unsealable, nava.audit.seal.HeadNotFound) outside internal/workerguard and, for the audit.seal types, internal/audit/seal, and an audit.seal type built with a non-constant message (AR-044, TICK-594)",
	Run:  run,
}

const (
	msg      = "outside internal/workerguard: only the worker guard builds this failure type (AR-044)"
	sealMsg  = "outside internal/workerguard and internal/audit/seal: only the worker guard and the audit sealer build this failure type (TICK-594)"
	constMsg = "built with a non-constant message: the worker guard records it as is, so the message must be a constant (TICK-594)"
)

func isGuardName(name string) bool {
	return name == "ErrTypeTenantGuard" || name == "ErrTypeWorkflowRef" || name == "ErrTypeWorkflowRejected" ||
		isSealName(name)
}

func isGuardValue(s string) bool {
	return s == "nava.TenantGuard" || s == "nava.WorkflowRefNotFound" || s == "nava.WorkflowRejected" ||
		isSealValue(s)
}

func isSealName(name string) bool {
	return name == "ErrTypeAuditSealUnsealable" || name == "ErrTypeAuditSealHeadNotFound"
}

func isSealValue(s string) bool {
	return s == "nava.audit.seal.Unsealable" || s == "nava.audit.seal.HeadNotFound"
}

// outsideMsg is the finding text for a type used where it may not be.
func outsideMsg(seal bool) string {
	if seal {
		return sealMsg
	}
	return msg
}

func run(pass *analysis.Pass) (any, error) {
	pkg := strings.TrimSuffix(pass.Pkg.Path(), "_test")
	if pkg == GuardPkg {
		return nil, nil
	}
	inSeal := pkg == SealPkg
	for _, f := range pass.Files {
		if strings.HasSuffix(pass.Fset.File(f.Pos()).Name(), "_test.go") {
			continue
		}
		ast.Inspect(f, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.Ident:
				checkUse(pass, n, inSeal)
			case *ast.BasicLit, *ast.BinaryExpr, *ast.ParenExpr:
				return checkConst(pass, n.(ast.Expr), inSeal)
			case *ast.CallExpr:
				if inSeal {
					checkSealCall(pass, n)
				}
			}
			return true
		})
	}
	return nil, nil
}

// checkUse reports an identifier that refers to one of the guard's
// constants, or to seal's re-export of an audit.seal type, where it may not
// be used.
func checkUse(pass *analysis.Pass, id *ast.Ident, inSeal bool) {
	obj := pass.TypesInfo.Uses[id]
	if obj == nil || obj.Pkg() == nil {
		return
	}
	switch obj.Pkg().Path() {
	case GuardPkg:
		if !isGuardName(obj.Name()) {
			return
		}
		seal := isSealName(obj.Name())
		if seal && inSeal {
			return
		}
		pass.Reportf(id.Pos(), "workerguard.%s used %s", obj.Name(), outsideMsg(seal))
	case SealPkg:
		if inSeal || !isSealConst(obj) {
			return
		}
		pass.Reportf(id.Pos(), "seal.%s used %s", obj.Name(), sealMsg)
	}
}

// isSealConst reports whether obj is a string constant equal to an
// audit.seal type, as seal.ErrTypeUnsealable is.
func isSealConst(obj types.Object) bool {
	c, ok := obj.(*types.Const)
	return ok && c.Val().Kind() == constant.String && isSealValue(constant.StringVal(c.Val()))
}

// checkConst reports a string constant equal to a guard type where it may
// not be used. The outermost constant is checked once; its parts are not
// checked again.
func checkConst(pass *analysis.Pass, e ast.Expr, inSeal bool) bool {
	s, ok := stringConst(pass, e)
	if !ok {
		return true
	}
	if isGuardValue(s) {
		seal := isSealValue(s)
		if !(seal && inSeal) {
			pass.Reportf(e.Pos(), "failure type %q %s", s, outsideMsg(seal))
		}
	}
	return false
}

// checkSealCall reports, inside internal/audit/seal, a call that passes an
// audit.seal type after its first argument while the first argument, the
// message, is not a constant: temporal.NewApplicationError,
// NewNonRetryableApplicationError and NewApplicationErrorWithOptions all take
// the message first and the type second.
func checkSealCall(pass *analysis.Pass, call *ast.CallExpr) {
	if len(call.Args) < 2 {
		return
	}
	if tv, ok := pass.TypesInfo.Types[call.Args[0]]; ok && tv.Value != nil {
		return
	}
	for _, a := range call.Args[1:] {
		if s, ok := stringConst(pass, a); ok && isSealValue(s) {
			pass.Reportf(call.Args[0].Pos(), "failure type %q %s", s, constMsg)
			return
		}
	}
}

// stringConst returns e's value if it is a string constant.
func stringConst(pass *analysis.Pass, e ast.Expr) (string, bool) {
	tv, ok := pass.TypesInfo.Types[e]
	if !ok || tv.Value == nil || tv.Value.Kind() != constant.String {
		return "", false
	}
	return constant.StringVal(tv.Value), true
}

package keycloak

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/parser"
	"go/token"
	"go/types"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

type stubImporter struct{}

func (stubImporter) Import(path string) (*types.Package, error) {
	p := types.NewPackage(path, filepath.Base(path))
	p.MarkComplete()
	return p, nil
}

// orgCallViolations type-checks the package in dir and reports every call that
// passes an /organizations path (a string constant, a named constant or a
// concatenation with one) and does not use the allowed account:
//   - an account argument must be exactly <recv>.onboarding, or <recv>.runtime
//     inside the function named allowRuntimeIn;
//   - an argument of type ClientCredentials is never allowed.
//
// It also returns the number of calls it saw that used the runtime account.
func orgCallViolations(dir, allowRuntimeIn string) (violations []string, runtimeCalls int, err error) {
	fset := token.NewFileSet()
	matches, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		return nil, 0, err
	}
	var files []*ast.File
	for _, m := range matches {
		if strings.HasSuffix(m, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, m, nil, 0)
		if err != nil {
			return nil, 0, err
		}
		files = append(files, f)
	}
	info := &types.Info{Types: map[ast.Expr]types.TypeAndValue{}}
	// Imports are stubbed with empty packages: type-checking the real
	// dependencies from source takes minutes. Everything the guard reads
	// (the account types, the Directory fields, string constants) is declared
	// in this package, so errors about the stubbed imports are ignored.
	conf := types.Config{Importer: stubImporter{}, Error: func(error) {}}
	_, _ = conf.Check("guarded", fset, files, info)

	orgPath := func(e ast.Expr) bool {
		var walk func(ast.Expr) bool
		walk = func(e ast.Expr) bool {
			if tv, ok := info.Types[e]; ok && tv.Value != nil && tv.Value.Kind() == constant.String {
				return strings.Contains(constant.StringVal(tv.Value), "/organizations")
			}
			switch v := e.(type) {
			case *ast.BinaryExpr:
				return walk(v.X) || walk(v.Y)
			case *ast.ParenExpr:
				return walk(v.X)
			}
			return false
		}
		return walk(e)
	}
	typeName := func(e ast.Expr) string {
		t := info.Types[e].Type
		if p, ok := t.(*types.Pointer); ok {
			t = p.Elem()
		}
		if n, ok := t.(*types.Named); ok {
			return n.Obj().Name()
		}
		return ""
	}
	field := func(e ast.Expr) string {
		if s, ok := e.(*ast.SelectorExpr); ok {
			return s.Sel.Name
		}
		return ""
	}

	for _, f := range files {
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				var org bool
				for _, a := range call.Args {
					org = org || orgPath(a)
				}
				if !org {
					return true
				}
				pos := fset.Position(call.Pos())
				for _, a := range call.Args {
					switch typeName(a) {
					case "ClientCredentials":
						violations = append(violations, fmt.Sprintf("%s: %s passes ClientCredentials to an /organizations call", pos, fn.Name.Name))
					case "account":
						switch field(a) {
						case "onboarding":
						case "runtime":
							runtimeCalls++
							if fn.Name.Name != allowRuntimeIn {
								violations = append(violations, fmt.Sprintf("%s: %s calls /organizations with d.runtime", pos, fn.Name.Name))
							}
						default:
							violations = append(violations, fmt.Sprintf("%s: %s calls /organizations with an account that is not d.onboarding or d.runtime", pos, fn.Name.Name))
						}
					}
				}
				return true
			})
		}
	}
	sort.Strings(violations)
	return violations, runtimeCalls, nil
}

// TestPlaceInviteeIsTheOnlyRuntimeOrganizationsCaller guards the group sync
// account's view-organizations role (TICK-252, Security ruling). That role
// reads every organization realm-wide, so the only code that may call an
// /organizations path through the d.runtime field is PlaceInvitee; everything
// else must use d.onboarding.
//
// The guard pins the d.runtime field, not an account. The account behind the
// field depends on the process: nava-groupsync in group sync and nava-platform
// in nava-onboard. That account is pinned by the compose wiring plus
// TestGroupsyncAccountFloor and TestRuntimeAccountHasNoRealmOrIdPReach (if
// nava-onboard ever reached PlaceInvitee, its account would get a 403).
func TestPlaceInviteeIsTheOnlyRuntimeOrganizationsCaller(t *testing.T) {
	t.Parallel()
	v, n, err := orgCallViolations(".", "PlaceInvitee")
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range v {
		t.Error(s)
	}
	if n == 0 {
		t.Error("guard found no runtime /organizations call: PlaceInvitee moved or the guard is blind")
	}
}

// The guard catches the rewrites that a literal-string check misses, and
// allows the call shapes that are fine.
func TestOrganizationsGuardCatchesRewrites(t *testing.T) {
	t.Parallel()
	v, _, err := orgCallViolations("testdata/orgguard", "PlaceInvitee")
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Join(v, "\n")
	for _, fn := range []string{"LiteralRuntime", "ConstPath", "ConcatConst", "CopiedAccount", "CredsArg"} {
		if !strings.Contains(got, " "+fn+" ") {
			t.Errorf("guard missed %s:\n%s", fn, got)
		}
	}
	for _, fn := range []string{"PlaceInvitee", "FindOrg", "Users"} {
		if strings.Contains(got, " "+fn+" ") {
			t.Errorf("guard flagged the allowed %s:\n%s", fn, got)
		}
	}
}

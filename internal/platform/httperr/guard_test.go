package httperr_test

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/api"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
)

const (
	modulePath = "github.com/Navaladi2019/TicketingSystem"
	problemPkg = modulePath + "/internal/platform/problem"
)

const moduleRoot = "../../.."

// guardAllowed lists the files, relative to the module root, that may break a
// rule: the 501 not_implemented placeholder and the panic recovery path.
type guardAllowed struct {
	notImplemented string
	internal       string
}

func repoAllowed() guardAllowed {
	return guardAllowed{
		notImplemented: filepath.Join("internal", "server", "api.go"),
		internal:       filepath.Join("internal", "platform", "httpobs", "recover.go"),
	}
}

// TestRegistryWriteIsClientErrorsOnly: Registry.Write and Body send a problem
// without reporting a cause, so they may only be given a constant code whose
// registry status is below 500. A 5xx goes through WriteError. The 501
// not_implemented placeholder is the one allowed exception. problem.Internal
// also reports nothing, so only the panic recovery path may call it.
//
// Calls are resolved by import path, not by name: the selector's package
// identifier must map to the problem package's import, and a Write or Body
// receiver must be declared as a *problem.Registry (parameter, variable or
// struct field). There is no type-checking: the go/build source importer was
// tried first and did not finish loading the module's dependencies in
// reasonable time.
func TestRegistryWriteIsClientErrorsOnly(t *testing.T) {
	t.Parallel()
	violations, calls := checkProblemCalls(t, loadRegistry(t), repoAllowed(), "internal", "cmd")
	for _, v := range violations {
		t.Error(v)
	}
	if calls == 0 {
		t.Error("found no registry Write or Body calls; the scan is broken")
	}
}

// TestGuardFixtures runs the guard over testdata packages, including the
// mutation check: a problem.Internal call added to a handler must fail it.
func TestGuardFixtures(t *testing.T) {
	t.Parallel()
	reg := loadRegistry(t)
	const base = "internal/platform/httperr/testdata/guard/"
	tests := []struct {
		name      string
		pkg       string
		wantCalls int
		wantErrs  []string
	}{
		{"binary.Write and unrelated Write/Body pass", "binarywrite", 0, nil},
		{"Registry.Write with a 4xx passes", "okcall", 1, nil},
		{"problem.Internal in a handler fails", "internalcall", 0, []string{"problem.Internal"}},
		{"5xx and non-constant codes fail", "badcode", 2, []string{"use WriteError", "not a problem.<Const>"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			violations, calls := checkProblemCalls(t, reg, repoAllowed(), base+tc.pkg)
			if calls != tc.wantCalls {
				t.Errorf("registry calls = %d, want %d", calls, tc.wantCalls)
			}
			if len(violations) != len(tc.wantErrs) {
				t.Fatalf("violations = %q, want %d", violations, len(tc.wantErrs))
			}
			for i, want := range tc.wantErrs {
				if !strings.Contains(violations[i], want) {
					t.Errorf("violation %d = %q, want it to contain %q", i, violations[i], want)
				}
			}
		})
	}
}

func loadRegistry(t *testing.T) *problem.Registry {
	t.Helper()
	data, err := api.Files.ReadFile("errors.yaml")
	if err != nil {
		t.Fatal(err)
	}
	reg, err := problem.LoadRegistry(data)
	if err != nil {
		t.Fatal(err)
	}
	return reg
}

// problemConsts maps each constant name of the problem package to its code.
func problemConsts(t *testing.T, fset *token.FileSet) map[string]problem.Code {
	t.Helper()
	pf, err := parser.ParseFile(fset, filepath.Join(moduleRoot, "internal", "platform", "problem", "problem.go"), nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	consts := map[string]problem.Code{}
	for _, d := range pf.Decls {
		if gd, ok := d.(*ast.GenDecl); ok && gd.Tok == token.CONST {
			for _, spec := range gd.Specs {
				vs := spec.(*ast.ValueSpec)
				for i, n := range vs.Names {
					if lit, ok := vs.Values[i].(*ast.BasicLit); ok {
						consts[n.Name] = problem.Code(strings.Trim(lit.Value, `"`))
					}
				}
			}
		}
	}
	return consts
}

// checkProblemCalls scans the non-test Go files under roots (relative to the
// module root, testdata skipped) and returns one message per violation and the
// number of *problem.Registry Write and Body calls it inspected.
func checkProblemCalls(t *testing.T, reg *problem.Registry, allowed guardAllowed, roots ...string) (violations []string, calls int) {
	t.Helper()
	fset := token.NewFileSet()
	consts := problemConsts(t, fset)
	problemDir := filepath.Join(moduleRoot, "internal", "platform", "problem")
	for _, r := range roots {
		root := filepath.Join(moduleRoot, r)
		pkgs := map[string][]*ast.File{} // dir -> files
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			switch {
			case err != nil:
				return err
			case d.IsDir() && d.Name() == "testdata" && path != root:
				return filepath.SkipDir
			case d.IsDir(), !strings.HasSuffix(path, ".go"), strings.HasSuffix(path, "_test.go"),
				filepath.Dir(path) == problemDir:
				return nil
			}
			f, err := parser.ParseFile(fset, path, nil, 0)
			if err != nil {
				return err
			}
			pkgs[filepath.Dir(path)] = append(pkgs[filepath.Dir(path)], f)
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		for _, files := range pkgs {
			fields := registryFieldNames(files)
			for _, f := range files {
				rel, _ := filepath.Rel(moduleRoot, fset.Position(f.Pos()).Filename)
				v, c := checkFile(fset, f, rel, reg, consts, fields, allowed)
				violations, calls = append(violations, v...), calls+c
			}
		}
	}
	return violations, calls
}

// problemName returns the local name f gives the problem package, or "".
func problemName(f *ast.File) string {
	for _, imp := range f.Imports {
		if strings.Trim(imp.Path.Value, `"`) != problemPkg {
			continue
		}
		if imp.Name != nil {
			return imp.Name.Name
		}
		return "problem"
	}
	return ""
}

// isRegistryType reports whether e is *<problem>.Registry.
func isRegistryType(e ast.Expr, pname string) bool {
	star, ok := e.(*ast.StarExpr)
	if !ok {
		return false
	}
	sel, ok := star.X.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	id, ok := sel.X.(*ast.Ident)
	return ok && pname != "" && id.Name == pname && sel.Sel.Name == "Registry"
}

// registryFieldNames returns the struct fields of a package declared as
// *problem.Registry.
func registryFieldNames(files []*ast.File) map[string]bool {
	out := map[string]bool{}
	for _, f := range files {
		pname := problemName(f)
		ast.Inspect(f, func(n ast.Node) bool {
			if st, ok := n.(*ast.StructType); ok {
				for _, fld := range st.Fields.List {
					if isRegistryType(fld.Type, pname) {
						for _, name := range fld.Names {
							out[name.Name] = true
						}
					}
				}
			}
			return true
		})
	}
	return out
}

// isRegistry reports whether x is a *problem.Registry: an identifier declared
// with that type, or a struct field declared with it.
func isRegistry(x ast.Expr, pname string, fields map[string]bool) bool {
	switch x := x.(type) {
	case *ast.Ident:
		if x.Obj == nil {
			return false
		}
		switch d := x.Obj.Decl.(type) {
		case *ast.Field:
			return isRegistryType(d.Type, pname)
		case *ast.ValueSpec:
			return d.Type != nil && isRegistryType(d.Type, pname)
		}
	case *ast.SelectorExpr:
		return fields[x.Sel.Name]
	}
	return false
}

func checkFile(fset *token.FileSet, f *ast.File, rel string, reg *problem.Registry, consts map[string]problem.Code,
	fields map[string]bool, allowed guardAllowed) (violations []string, calls int) {
	pname := problemName(f)
	if pname == "" {
		return nil, 0
	}
	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		pos := fset.Position(call.Pos())
		if id, ok := sel.X.(*ast.Ident); ok && id.Name == pname && id.Obj == nil && sel.Sel.Name == "Internal" {
			if rel != allowed.internal {
				violations = append(violations, fmt.Sprintf("%s: problem.Internal reports no cause; only %s may call it", pos, allowed.internal))
			}
			return true
		}
		var idx int
		switch {
		case sel.Sel.Name == "Write" && len(call.Args) >= 3: // reg.Write(w, r, code, ...)
			idx = 2
		case sel.Sel.Name == "Body" && len(call.Args) >= 2: // reg.Body(ctx, code, ...)
			idx = 1
		default:
			return true
		}
		if !isRegistry(sel.X, pname, fields) {
			return true
		}
		calls++
		if msg := checkCode(call.Args[idx], pname, consts, reg, sel.Sel.Name, rel == allowed.notImplemented); msg != "" {
			violations = append(violations, fmt.Sprintf("%s: %s", pos, msg))
		}
		return true
	})
	return violations, calls
}

// checkCode returns a message when arg is not a problem.<Const> naming a
// registered code with a status below 500 (501 not_implemented where allowed).
func checkCode(arg ast.Expr, pname string, consts map[string]problem.Code, reg *problem.Registry, method string, allowNotImplemented bool) string {
	const notConst = "%s code is not a problem.<Const>; use WriteError for anything that can be a 5xx"
	sel, ok := arg.(*ast.SelectorExpr)
	if !ok {
		return fmt.Sprintf(notConst, method)
	}
	if id, ok := sel.X.(*ast.Ident); !ok || id.Name != pname {
		return fmt.Sprintf(notConst, method)
	}
	code, known := consts[sel.Sel.Name]
	e, registered := reg.Lookup(code)
	switch {
	case !known || !registered:
		return fmt.Sprintf("%s.%s is not a registered problem code", method, sel.Sel.Name)
	case e.Status >= http.StatusInternalServerError && (!allowNotImplemented || code != problem.NotImplemented):
		return fmt.Sprintf("%s sends %s (%d); use WriteError so the cause is reported", method, code, e.Status)
	}
	return ""
}

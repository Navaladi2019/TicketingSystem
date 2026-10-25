package httperr_test

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"net/http"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/api"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
)

const (
	modulePath = "github.com/Navaladi2019/TicketingSystem"
	problemPkg = modulePath + "/internal/platform/problem"
	moduleRoot = "../../.."
)

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
// also reports nothing, so only the panic recovery path may reference it.
//
// There is no type-checking (the go/build source importer did not finish
// loading the module's dependencies in reasonable time), so the guard resolves
// by import path and declared type, and fails closed:
//   - a dot-import of problem is a violation;
//   - any reference to <problem>.Internal outside the allowed file is one,
//     called or not;
//   - a Write or Body receiver must be declared as *problem.Registry (typed
//     parameter, typed var, or struct field anywhere in the module) or as some
//     other declared type; a receiver the guard cannot resolve (:=, untyped
//     var, call result) is a violation when the call has the registry's arity,
//     or when it is uncalled (a .Write or .Body method value); an uncalled
//     .Body is exempt only in a file that does not import problem, where it
//     is a field such as resp.Body;
//   - a Write or Body method value, a method expression, and an embedded
//     problem.Registry are violations.
//
// A legitimate call the guard cannot resolve is fixed by giving its receiver
// a typed declaration.
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

// TestGuardFixtures runs the guard over testdata packages. Each bypass row has
// a fixture that must fail; binarywrite and okcall must pass.
func TestGuardFixtures(t *testing.T) {
	t.Parallel()
	reg := loadRegistry(t)
	const base = "internal/platform/httperr/testdata/guard/"
	tests := []struct {
		pkg       string
		wantCalls int
		wantErrs  []string
	}{
		{"binarywrite", 0, nil},
		{"okcall", 1, nil},
		{"internalcall", 0, []string{"problem.Internal"}},
		{"badcode", 2, []string{"use WriteError", "not a problem.<Const>"}},
		{"aliasimport", 1, []string{"problem.Internal", "use WriteError"}},
		{"dotimport", 0, []string{"dot-import"}},
		{"shortvar", 0, []string{"cannot resolve the receiver"}},
		{"untypedvar", 0, []string{"cannot resolve the receiver"}},
		{"methodvalue", 0, []string{"method value"}},
		{"methodexpr", 0, []string{"method expression"}},
		{"embedded", 1, []string{"embeds problem.Registry"}},
		{"callresult", 0, []string{"cannot resolve the receiver"}},
		{"internalvalue", 0, []string{"problem.Internal", "problem.Internal"}},
		{"bodyvalue", 0, []string{"cannot resolve the receiver"}},
		{"bodyarg", 0, []string{"cannot resolve the receiver"}},
		{"respbody", 0, nil},
		{"crosspkg", 1, []string{"use WriteError"}},
	}
	for _, tc := range tests {
		t.Run(tc.pkg, func(t *testing.T) {
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

// problemConsts maps each constant name of the problem package to its code. A
// constant without a literal of its own (an implicit repeat, or another
// constant) is left out, so a use of it fails as an unregistered code.
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
					if i >= len(vs.Values) {
						continue
					}
					if lit, ok := vs.Values[i].(*ast.BasicLit); ok {
						consts[n.Name] = problem.Code(strings.Trim(lit.Value, `"`))
					}
				}
			}
		}
	}
	return consts
}

// guardFile is one parsed non-test source file.
type guardFile struct {
	f       *ast.File
	rel     string          // path relative to the module root
	pname   string          // local name of the problem package, or ""
	imports map[string]bool // local names of every import
}

// guardCtx is what the checks share: the registry codes, and the struct field
// names declared anywhere in the scanned files, by whether they hold a
// *problem.Registry.
type guardCtx struct {
	fset      *token.FileSet
	reg       *problem.Registry
	consts    map[string]problem.Code
	allowed   guardAllowed
	regFields map[string]bool
	otherFlds map[string]bool
}

// checkProblemCalls scans the non-test Go files under roots (relative to the
// module root, testdata skipped) and returns one message per violation and the
// number of *problem.Registry Write and Body calls it inspected.
func checkProblemCalls(t *testing.T, reg *problem.Registry, allowed guardAllowed, roots ...string) (violations []string, calls int) {
	t.Helper()
	fset := token.NewFileSet()
	ctx := &guardCtx{
		fset: fset, reg: reg, consts: problemConsts(t, fset), allowed: allowed,
		regFields: map[string]bool{}, otherFlds: map[string]bool{},
	}
	problemDir := filepath.Join(moduleRoot, "internal", "platform", "problem")
	var files []*guardFile
	for _, r := range roots {
		root := filepath.Join(moduleRoot, r)
		err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
			switch {
			case err != nil:
				return err
			case d.IsDir() && d.Name() == "testdata" && p != root:
				return filepath.SkipDir
			case d.IsDir(), !strings.HasSuffix(p, ".go"), strings.HasSuffix(p, "_test.go"),
				filepath.Dir(p) == problemDir:
				return nil
			}
			f, err := parser.ParseFile(fset, p, nil, 0)
			if err != nil {
				return err
			}
			rel, _ := filepath.Rel(moduleRoot, p)
			gf := &guardFile{f: f, rel: rel, imports: map[string]bool{}}
			for _, imp := range f.Imports {
				ipath := strings.Trim(imp.Path.Value, `"`)
				name := path.Base(ipath)
				if imp.Name != nil {
					name = imp.Name.Name
				}
				gf.imports[name] = true
				if ipath == problemPkg && name != "." && name != "_" {
					gf.pname = name
				}
			}
			files = append(files, gf)
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, gf := range files {
		ctx.collectFields(gf)
	}
	for _, gf := range files {
		v, c := ctx.checkFile(gf)
		violations, calls = append(violations, v...), calls+c
	}
	return violations, calls
}

// isRegistryType reports whether e is problem.Registry or *problem.Registry,
// under the local name pname.
func isRegistryType(e ast.Expr, pname string) bool {
	if star, ok := e.(*ast.StarExpr); ok {
		e = star.X
	}
	sel, ok := e.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	id, ok := sel.X.(*ast.Ident)
	return ok && pname != "" && id.Name == pname && sel.Sel.Name == "Registry"
}

// collectFields records every struct field name of gf, by whether its type is
// a registry. An embedded registry is recorded under the name "Registry".
func (c *guardCtx) collectFields(gf *guardFile) {
	ast.Inspect(gf.f, func(n ast.Node) bool {
		st, ok := n.(*ast.StructType)
		if !ok {
			return true
		}
		for _, fld := range st.Fields.List {
			isReg := isRegistryType(fld.Type, gf.pname)
			switch {
			case len(fld.Names) == 0 && isReg:
				c.regFields["Registry"] = true
			case isReg:
				for _, name := range fld.Names {
					c.regFields[name.Name] = true
				}
			default:
				for _, name := range fld.Names {
					c.otherFlds[name.Name] = true
				}
			}
		}
		return true
	})
}

type recvKind int

const (
	recvOther recvKind = iota // a declared type that is not the registry
	recvRegistry
	recvUnresolved
	recvMethodExpr
)

// classify decides what the receiver x of a Write or Body selector is.
func (c *guardCtx) classify(gf *guardFile, x ast.Expr) recvKind {
	byType := func(t ast.Expr) recvKind {
		if isRegistryType(t, gf.pname) {
			return recvRegistry
		}
		return recvOther
	}
	switch x := x.(type) {
	case *ast.Ident:
		if x.Obj == nil {
			if gf.imports[x.Name] { // a package name, as in binary.Write
				return recvOther
			}
			return recvUnresolved
		}
		switch d := x.Obj.Decl.(type) {
		case *ast.Field: // parameter, result or receiver
			return byType(d.Type)
		case *ast.ValueSpec:
			if d.Type != nil {
				return byType(d.Type)
			}
		}
		return recvUnresolved // :=, range, untyped var
	case *ast.SelectorExpr:
		if id, ok := x.X.(*ast.Ident); ok && id.Obj == nil && gf.imports[id.Name] {
			switch {
			case id.Name != gf.pname:
				return recvOther // os.Stdout.Write
			case x.Sel.Name == "Registry":
				return recvMethodExpr // problem.Registry.Write
			}
			return recvUnresolved
		}
		switch {
		case c.regFields[x.Sel.Name]:
			return recvRegistry
		case c.otherFlds[x.Sel.Name]:
			return recvOther
		}
	case *ast.ParenExpr:
		if isRegistryType(x.X, gf.pname) {
			return recvMethodExpr // (*problem.Registry).Write
		}
	}
	return recvUnresolved
}

func (c *guardCtx) checkFile(gf *guardFile) (violations []string, calls int) {
	report := func(pos token.Pos, format string, args ...any) {
		violations = append(violations, fmt.Sprintf("%s: %s", c.fset.Position(pos), fmt.Sprintf(format, args...)))
	}
	for _, imp := range gf.f.Imports {
		if imp.Name != nil && imp.Name.Name == "." && strings.Trim(imp.Path.Value, `"`) == problemPkg {
			report(imp.Pos(), "dot-import of problem hides problem.Internal and the registry calls; import it by name")
		}
	}
	called := map[*ast.SelectorExpr]*ast.CallExpr{}
	ast.Inspect(gf.f, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
				called[sel] = call
			}
		}
		return true
	})
	ast.Inspect(gf.f, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.StructType:
			for _, fld := range n.Fields.List {
				if len(fld.Names) == 0 && isRegistryType(fld.Type, gf.pname) {
					report(fld.Pos(), "struct embeds problem.Registry, which hides its Write and Body calls; use a named field")
				}
			}
		case *ast.SelectorExpr:
			if id, ok := n.X.(*ast.Ident); ok && gf.pname != "" && id.Name == gf.pname && id.Obj == nil && n.Sel.Name == "Internal" {
				if gf.rel != c.allowed.internal {
					report(n.Pos(), "problem.Internal reports no cause; only %s may reference it", c.allowed.internal)
				}
				return true
			}
			if n.Sel.Name != "Write" && n.Sel.Name != "Body" {
				return true
			}
			call := called[n]
			idx := 2 // reg.Write(w, r, code, ...)
			if n.Sel.Name == "Body" {
				idx = 1 // reg.Body(ctx, code, ...)
			}
			hasArity := call != nil && len(call.Args) > idx
			switch c.classify(gf, n.X) {
			case recvMethodExpr:
				report(n.Pos(), "%s method expression on problem.Registry hides its code argument; call it on a typed receiver", n.Sel.Name)
			case recvRegistry:
				switch {
				case call == nil:
					report(n.Pos(), "%s method value on a problem.Registry hides its code argument; call it directly", n.Sel.Name)
				case hasArity:
					calls++
					if msg := c.checkCode(gf, call.Args[idx], n.Sel.Name); msg != "" {
						report(call.Pos(), "%s", msg)
					}
				}
			case recvUnresolved:
				// An uncalled .Body on an unresolved receiver is a method value
				// that can send a 5xx, so it fails in any file that imports
				// problem. Files that do not import it may use resp.Body.
				if (call == nil && (n.Sel.Name == "Write" || gf.pname != "")) || hasArity {
					report(n.Pos(), "cannot resolve the receiver of %s; declare it with an explicit type (*problem.Registry or another)", n.Sel.Name)
				}
			}
		}
		return true
	})
	return violations, calls
}

// checkCode returns a message when arg is not a problem.<Const> naming a
// registered code with a status below 500 (501 not_implemented where allowed).
func (c *guardCtx) checkCode(gf *guardFile, arg ast.Expr, method string) string {
	const notConst = "%s code is not a problem.<Const>; use WriteError for anything that can be a 5xx"
	sel, ok := arg.(*ast.SelectorExpr)
	if !ok {
		return fmt.Sprintf(notConst, method)
	}
	if id, ok := sel.X.(*ast.Ident); !ok || gf.pname == "" || id.Name != gf.pname {
		return fmt.Sprintf(notConst, method)
	}
	code, known := c.consts[sel.Sel.Name]
	e, registered := c.reg.Lookup(code)
	switch {
	case !known || !registered:
		return fmt.Sprintf("%s.%s is not a registered problem code", method, sel.Sel.Name)
	case e.Status >= http.StatusInternalServerError && (gf.rel != c.allowed.notImplemented || code != problem.NotImplemented):
		return fmt.Sprintf("%s sends %s (%d); use WriteError so the cause is reported", method, code, e.Status)
	}
	return ""
}

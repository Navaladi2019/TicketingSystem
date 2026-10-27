package httperr_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/constant"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/api"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
)

const (
	modulePath = "github.com/Navaladi2019/TicketingSystem"
	problemPkg = modulePath + "/internal/platform/problem"
	moduleRoot = "../../.."
	fixtures   = "internal/platform/httperr/testdata/guard"

	// guardTags selects the files that build only under a tag, so the scan
	// covers them too. relaytest and spike are left out: relaytest excludes a
	// file that the plain build has, and neither holds a handler.
	guardTags = "integration,identitystack,temporalstack,lintfixture"
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
// The guard type-checks the scanned packages with go/types, so a receiver is
// resolved by its type however it was obtained (a := call, a field of another
// package, an embedded struct). Standard-library imports come from the export data of one
// batched `go list -export` run; module packages are checked from source and
// third-party packages are stubs, so only the standard library is compiled.
// The guard reports:
//   - a dot-import of problem;
//   - any reference to problem.Internal outside the allowed file, called or
//     not;
//   - a Write or Body method value or method expression on *problem.Registry,
//     in any file, because the code argument is then out of sight;
//   - a struct that embeds problem.Registry;
//   - a Write or Body call whose code is not a problem.<Const> naming a
//     registered code with a status below 500.
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
// a fixture that must fail; binarywrite, okcall and respbody must pass.
func TestGuardFixtures(t *testing.T) {
	t.Parallel()
	reg := loadRegistry(t)
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
		{"shortvar", 1, []string{"use WriteError"}},
		{"untypedvar", 1, []string{"use WriteError"}},
		{"methodvalue", 0, []string{"method value"}},
		{"methodexpr", 0, []string{"method expression"}},
		{"embedded", 2, []string{"embeds problem.Registry", "use WriteError"}},
		{"callresult", 1, []string{"use WriteError"}},
		{"internalvalue", 0, []string{"problem.Internal", "problem.Internal"}},
		{"bodyvalue", 0, []string{"method value"}},
		{"bodyarg", 0, []string{"method value"}},
		{"respbody", 0, nil},
		{"crosspkg", 1, []string{"use WriteError"}},
		{"crosspkgvalue", 0, []string{"method value"}},
	}
	for _, tc := range tests {
		t.Run(tc.pkg, func(t *testing.T) {
			t.Parallel()
			violations, calls := checkProblemCalls(t, reg, repoAllowed(), fixtures+"/"+tc.pkg)
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

// listedPkg is the part of `go list -json` the guard reads.
type listedPkg struct {
	ImportPath string
	Dir        string
	GoFiles    []string
	Export     string
	Standard   bool
	Error      *struct{ Err string }
}

// guardWorld is the package graph of the scanned roots, listed once for all
// tests of the run.
type guardWorld struct {
	absRoot string
	pkgs    map[string]*listedPkg
	err     error
}

//nolint:gochecknoglobals // listing the module once keeps the guard tests fast
var (
	worldOnce sync.Once
	world     guardWorld
)

// goList runs `go list -e -json=<fields> args...` in the module root and
// decodes the stream of packages.
func goList(dir, fields string, args ...string) ([]*listedPkg, error) {
	cmd := exec.CommandContext(context.Background(), "go", append([]string{"list", "-e", "-json=" + fields}, args...)...)
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("go list: %w\n%s", err, stderr.String())
	}
	var pkgs []*listedPkg
	dec := json.NewDecoder(bytes.NewReader(out))
	for {
		var p listedPkg
		if err := dec.Decode(&p); err == io.EOF {
			return pkgs, nil
		} else if err != nil {
			return nil, err
		}
		pkgs = append(pkgs, &p)
	}
}

// loadWorld lists internal/..., cmd/... and the fixtures with their
// dependencies, then fetches export data for the standard-library packages
// only. Module packages are type-checked from source and third-party packages
// are stubs, so nothing outside the standard library is compiled.
func loadWorld(t *testing.T) *guardWorld {
	t.Helper()
	worldOnce.Do(func() {
		abs, err := filepath.Abs(moduleRoot)
		if err != nil {
			world.err = err
			return
		}
		world.absRoot = abs
		listed, err := goList(abs, "ImportPath,Dir,GoFiles,Standard,Error", "-deps", "-tags", guardTags,
			"./internal/...", "./cmd/...", "./"+fixtures+"/...")
		if err != nil {
			world.err = err
			return
		}
		world.pkgs = map[string]*listedPkg{}
		var std []string
		for _, p := range listed {
			world.pkgs[p.ImportPath] = p
			if p.Standard {
				std = append(std, p.ImportPath)
			}
		}
		exports, err := goList(abs, "ImportPath,Export", append([]string{"-export", "-tags", guardTags}, std...)...)
		if err != nil {
			world.err = err
			return
		}
		for _, e := range exports {
			world.pkgs[e.ImportPath].Export = e.Export
		}
	})
	if world.err != nil {
		t.Fatal(world.err)
	}
	return &world
}

// selected reports whether dir (absolute) is under one of roots (relative to
// the module root), outside any testdata directory below the root.
func (w *guardWorld) selected(dir string, roots []string) bool {
	rel, err := filepath.Rel(w.absRoot, dir)
	if err != nil {
		return false
	}
	for _, r := range roots {
		r = filepath.FromSlash(r)
		if rel == r {
			return true
		}
		if sub, ok := strings.CutPrefix(rel, r+string(filepath.Separator)); ok &&
			!slices.Contains(strings.Split(sub, string(filepath.Separator)), "testdata") {
			return true
		}
	}
	return false
}

// checked is one module package, type-checked from source.
type checked struct {
	pkg   *types.Package
	files []*ast.File
	info  *types.Info
}

// guardImporter resolves an import path: the standard library from export
// data, a module package from source (once, so type identity holds across
// packages), and anything else as an empty stub whose uses are type errors
// that the guard ignores.
type guardImporter struct {
	t    *testing.T
	w    *guardWorld
	fset *token.FileSet
	std  types.Importer
	done map[string]*checked
	stub map[string]*types.Package
}

func (g *guardImporter) Import(p string) (*types.Package, error) {
	lp := g.w.pkgs[p]
	switch {
	case lp != nil && lp.Standard:
		return g.std.Import(p)
	case strings.HasPrefix(p, modulePath+"/"):
		return g.check(p).pkg, nil
	}
	if s, ok := g.stub[p]; ok {
		return s, nil
	}
	s := types.NewPackage(p, path.Base(p))
	s.MarkComplete()
	g.stub[p] = s
	return s, nil
}

func (g *guardImporter) check(p string) *checked {
	if c, ok := g.done[p]; ok {
		return c
	}
	lp := g.w.pkgs[p]
	if lp == nil {
		g.t.Fatalf("package %s is not in the go list output", p)
	}
	if lp.Error != nil {
		g.t.Fatalf("package %s: %s", p, lp.Error.Err)
	}
	c := &checked{info: &types.Info{
		Types:      map[ast.Expr]types.TypeAndValue{},
		Uses:       map[*ast.Ident]types.Object{},
		Selections: map[*ast.SelectorExpr]*types.Selection{},
	}}
	g.done[p] = c
	for _, name := range lp.GoFiles {
		f, err := parser.ParseFile(g.fset, filepath.Join(lp.Dir, name), nil, parser.SkipObjectResolution)
		if err != nil {
			g.t.Fatal(err)
		}
		c.files = append(c.files, f)
	}
	conf := types.Config{Importer: g, Error: func(error) {}}
	c.pkg, _ = conf.Check(p, g.fset, c.files, c.info)
	return c
}

// guardCtx is what the checks share.
type guardCtx struct {
	fset    *token.FileSet
	reg     *problem.Registry
	allowed guardAllowed
	absRoot string
}

// checkProblemCalls type-checks the non-test Go packages under roots (relative
// to the module root, testdata skipped) and returns one message per violation
// and the number of *problem.Registry Write and Body calls it inspected.
func checkProblemCalls(t *testing.T, reg *problem.Registry, allowed guardAllowed, roots ...string) (violations []string, calls int) {
	t.Helper()
	w := loadWorld(t)
	fset := token.NewFileSet()
	g := &guardImporter{
		t: t, w: w, fset: fset,
		done: map[string]*checked{}, stub: map[string]*types.Package{},
	}
	g.std = importer.ForCompiler(fset, "gc", func(p string) (io.ReadCloser, error) {
		lp := w.pkgs[p]
		if lp == nil || lp.Export == "" {
			return nil, fmt.Errorf("no export data for %q", p)
		}
		return os.Open(lp.Export)
	})
	ctx := &guardCtx{fset: fset, reg: reg, allowed: allowed, absRoot: w.absRoot}

	paths := make([]string, 0, len(w.pkgs))
	for p := range w.pkgs {
		paths = append(paths, p)
	}
	slices.Sort(paths)
	for _, p := range paths {
		lp := w.pkgs[p]
		if lp.Standard || p == problemPkg || len(lp.GoFiles) == 0 || !w.selected(lp.Dir, roots) {
			continue
		}
		c := g.check(p)
		for _, f := range c.files {
			v, n := ctx.checkFile(f, c.info)
			violations, calls = append(violations, v...), calls+n
		}
	}
	return violations, calls
}

// isRegistry reports whether typ is problem.Registry or a pointer to it.
func isRegistry(typ types.Type) bool {
	if p, ok := typ.(*types.Pointer); ok {
		typ = p.Elem()
	}
	n, ok := typ.(*types.Named)
	return ok && n.Obj().Pkg() != nil && n.Obj().Pkg().Path() == problemPkg && n.Obj().Name() == "Registry"
}

// isRegistryMethod reports whether obj is Write or Body of problem.Registry,
// however the receiver was reached, and which of the two it is.
func isRegistryMethod(obj types.Object) (string, bool) {
	fn, ok := obj.(*types.Func)
	if !ok || (fn.Name() != "Write" && fn.Name() != "Body") {
		return "", false
	}
	recv := fn.Type().(*types.Signature).Recv()
	return fn.Name(), recv != nil && isRegistry(recv.Type())
}

func (c *guardCtx) checkFile(f *ast.File, info *types.Info) (violations []string, calls int) {
	filename := c.fset.Position(f.Pos()).Filename
	rel, err := filepath.Rel(c.absRoot, filename)
	if err != nil {
		rel = filename
	}
	report := func(pos token.Pos, format string, args ...any) {
		p := c.fset.Position(pos)
		violations = append(violations, fmt.Sprintf("%s:%d: %s", filepath.ToSlash(rel), p.Line, fmt.Sprintf(format, args...)))
	}
	for _, imp := range f.Imports {
		if imp.Name != nil && imp.Name.Name == "." && strings.Trim(imp.Path.Value, `"`) == problemPkg {
			report(imp.Pos(), "dot-import of problem hides problem.Internal and the registry calls; import it by name")
		}
	}
	called := map[*ast.SelectorExpr]*ast.CallExpr{}
	ast.Inspect(f, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			if sel, ok := ast.Unparen(call.Fun).(*ast.SelectorExpr); ok {
				called[sel] = call
			}
		}
		return true
	})
	ast.Inspect(f, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.StructType:
			for _, fld := range n.Fields.List {
				if len(fld.Names) == 0 && isRegistry(info.TypeOf(fld.Type)) {
					report(fld.Pos(), "struct embeds problem.Registry, which hides its Write and Body calls; use a named field")
				}
			}
		case *ast.SelectorExpr:
			if obj, ok := info.Uses[n.Sel].(*types.Func); ok && obj.Pkg() != nil && obj.Pkg().Path() == problemPkg &&
				obj.Name() == "Internal" && obj.Type().(*types.Signature).Recv() == nil {
				if rel != c.allowed.internal {
					report(n.Pos(), "problem.Internal reports no cause; only %s may reference it", c.allowed.internal)
				}
				return true
			}
			sel := info.Selections[n]
			if sel == nil {
				return true
			}
			method, ok := isRegistryMethod(sel.Obj())
			if !ok {
				return true
			}
			call := called[n]
			idx := 2 // reg.Write(w, r, code, ...)
			if method == "Body" {
				idx = 1 // reg.Body(ctx, code, ...)
			}
			switch {
			case sel.Kind() == types.MethodExpr:
				report(n.Pos(), "%s method expression on problem.Registry hides its code argument; call it on a typed receiver", method)
			case call == nil:
				report(n.Pos(), "%s method value on a problem.Registry hides its code argument; call it directly", method)
			default:
				calls++
				if msg := c.checkCode(rel, info, call, idx, method); msg != "" {
					report(call.Pos(), "%s", msg)
				}
			}
		}
		return true
	})
	return violations, calls
}

// checkCode returns a message when the code argument of call is not a
// problem.<Const> naming a registered code with a status below 500 (501
// not_implemented where allowed).
func (c *guardCtx) checkCode(rel string, info *types.Info, call *ast.CallExpr, idx int, method string) string {
	const notConst = "%s code is not a problem.<Const>; use WriteError for anything that can be a 5xx"
	if len(call.Args) <= idx {
		return fmt.Sprintf(notConst, method)
	}
	var id *ast.Ident
	switch a := ast.Unparen(call.Args[idx]).(type) {
	case *ast.SelectorExpr:
		id = a.Sel
	default:
		return fmt.Sprintf(notConst, method)
	}
	k, ok := info.Uses[id].(*types.Const)
	if !ok || k.Pkg() == nil || k.Pkg().Path() != problemPkg || k.Val().Kind() != constant.String {
		return fmt.Sprintf(notConst, method)
	}
	code := problem.Code(constant.StringVal(k.Val()))
	e, registered := c.reg.Lookup(code)
	switch {
	case !registered:
		return fmt.Sprintf("%s.%s is not a registered problem code", method, k.Name())
	case e.Status >= http.StatusInternalServerError && (rel != c.allowed.notImplemented || code != problem.NotImplemented):
		return fmt.Sprintf("%s sends %s (%d); use WriteError so the cause is reported", method, code, e.Status)
	}
	return ""
}

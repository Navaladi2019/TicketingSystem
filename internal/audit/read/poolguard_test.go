// Rule 1 of the ADR-004-A1 static guard (A1-10, PLAN-005-A1 D-A1-6,
// TICK-559): only internal/audit/read opens the nava_audit_reader pool, and
// only the config loader names its DSN. AR-TICK-549-01 accepts the
// in-process risk of the reader's credential partly on the strength of this
// rule. Rule 2, the SQL-text check, is diffreaders_test.go.
//
// The check is static and runs in `task test`. Unlike rule 2 it reads Go
// identifiers and string constants, not SQL, and it covers every Go file in
// the repository, test files included (testdata, vendor and hidden
// directories aside). It reports:
//
//   - openpool: a reference to read.OpenPool, called or not, through the
//     default name, an alias or a dot import. Allowed in internal/audit/read
//     (and its sub-packages) and exactly once, as a call, in
//     cmd/nava-core/main.go. A second reference there reports both, and any
//     other file of cmd/nava-core reports its own.
//   - env: a string constant that contains the reader DSN's env name: a
//     literal, a package or function-local const, a `+` concatenation of
//     constants or a constant fmt.Sprintf. Allowed in internal/platform/config
//     only, its tests included. The name is not written here: it is the value
//     of config.AuditReaderDatabaseURLKey as TICK-558 merged it.
//   - key: a reference to config.AuditReaderDatabaseURLKey (alias or dot
//     import too). Allowed in internal/platform/config only.
//   - dsnfield: a selector of the DSN field, AuditReader.DatabaseURL, or a
//     composite literal of config.AuditReader that sets it. Allowed in
//     internal/platform/config, and in cmd/nava-core/main.go only inside the
//     arguments of its one allowed read.OpenPool call.
//
// A few test files TICK-558 merged need the reader pool or the env name to
// test it; poolGuardTestFiles lists each by exact path with the kinds it may
// use and why. Non-test files never get an exception.
//
// Known limits (static, no type checker):
//   - The DSN field is found by name flow inside one file: a variable
//     assigned from config.LoadAuditReader, or a variable, parameter, result
//     or struct field declared as config.AuditReader. A value that reaches a
//     file through an interface, a generic or another function's untyped
//     result is not followed. A same-named variable of another type in the
//     same file is over-reported, which fails closed.
//   - A local variable or parameter named OpenPool in a file that
//     dot-imports read is over-reported (fails closed).
//   - Only constants fold, as in rule 2: an env name assembled at run time
//     ([]byte, strings.Join of vars) is not seen. The database is still the
//     gate: nava_app, nava_worker and the rest get 42501 on diff and
//     diff_salt whatever DSN they hold (TICK-558).
//
// testdata/poolguard holds one fixture tree per directory, laid out as the
// repository; TestReaderPoolGuardFixture proves every planted line is
// reported and every look-alike is not.
package read_test

import (
	"fmt"
	"go/ast"
	"go/token"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

const (
	readPkgPath   = modulePath + "/internal/audit/read"
	configPkgPath = modulePath + "/internal/platform/config"
	readerPkgDir  = "internal/audit/read"
	configPkgDir  = "internal/platform/config"
	compositeRoot = "cmd/nava-core/main.go"

	// The names TICK-558 merged; poolGuardNames fails when one is renamed.
	openPoolName  = "OpenPool"
	readerKeyName = "AuditReaderDatabaseURLKey"
	readerCfgType = "AuditReader"
	readerLoader  = "LoadAuditReader"
	readerDSNName = "DatabaseURL"
)

// poolGuardTestFiles returns the kinds the test file at rel may use, by
// exact path. These files are compiled only into test binaries, never into a
// service.
func poolGuardTestFiles(rel string) []string {
	return map[string][]string{
		// The isolation suite's cluster opens the reader pool to prove
		// nava_audit_reader is tenant-scoped and masked (ADR-004-A1 reader-scoped,
		// reader-masks).
		"internal/tenancy/isolation/harness_isolation_test.go": {"openpool"},
		// The audit endpoints' integration test serves through a real reader
		// pool (A1-5, A1-6).
		"internal/server/entitytypes_integration_test.go": {"openpool"},
		// nava-core's startup tests set the DSN and check that a missing one is
		// refused with its name (A1-6).
		"cmd/nava-core/main_test.go": {"env", "key"},
		// The compose test checks that only nava-core's service sets the DSN,
		// naming it in its failure messages.
		"cmd/nava-core/compose_test.go": {"env", "key"},
	}[rel]
}

// poolRef is one reported reference.
type poolRef struct {
	file string
	line int
	what string // openpool, env, key or dsnfield
}

func (r poolRef) String() string { return fmt.Sprintf("%s:%d: %s", r.file, r.line, r.what) }

// poolGuardHow says how to fix a finding of kind what.
func poolGuardHow(what string) string {
	return map[string]string{
		"openpool": "only internal/audit/read and the one call in cmd/nava-core/main.go may open the nava_audit_reader pool; take a *read.Pool or a read.Reader instead",
		"env":      "only internal/platform/config may name the audit reader DSN's env variable; nava-core gets it through config.LoadAuditReader",
		"key":      "only internal/platform/config may use config.AuditReaderDatabaseURLKey",
		"dsnfield": "the audit reader DSN (config.AuditReader.DatabaseURL) may only be passed straight to the one read.OpenPool call in cmd/nava-core/main.go",
	}[what]
}

// poolGuardNames checks that the names this rule looks for still exist as
// TICK-558 merged them, and returns the env name, so a rename cannot turn
// the rule into a test that passes on nothing.
func poolGuardNames(t *testing.T, root string, idx map[string]pkgConsts) string {
	t.Helper()
	env := idx[configPkgPath].consts[readerKeyName]
	if !strings.HasPrefix(env, "NAVA_") {
		t.Fatalf("config.%s does not resolve to an env name (%q); update the guard", readerKeyName, env)
	}
	found := map[string]bool{}
	for _, dir := range []string{configPkgDir, readerPkgDir} {
		paths := goDirs(t, filepath.Join(root, filepath.FromSlash(dir)))[filepath.Join(root, filepath.FromSlash(dir))]
		for _, f := range parseDir(t, token.NewFileSet(), paths) {
			for _, d := range f.Decls {
				switch d := d.(type) {
				case *ast.FuncDecl:
					if d.Recv == nil {
						found[dir+"."+d.Name.Name] = true
					}
				case *ast.GenDecl:
					for _, sp := range d.Specs {
						ts, ok := sp.(*ast.TypeSpec)
						if !ok {
							continue
						}
						st, ok := ts.Type.(*ast.StructType)
						if !ok {
							continue
						}
						for _, fl := range st.Fields.List {
							for _, n := range fl.Names {
								found[dir+"."+ts.Name.Name+"."+n.Name] = true
							}
						}
					}
				}
			}
		}
	}
	for _, want := range []string{
		readerPkgDir + "." + openPoolName,
		configPkgDir + "." + readerLoader,
		configPkgDir + "." + readerCfgType + "." + readerDSNName,
	} {
		if !found[want] {
			t.Fatalf("%s is gone; the reader pool guard would pass on nothing, update its names", want)
		}
	}
	return env
}

// poolGoFiles returns every .go file under start, test files included, by
// directory, skipping testdata, vendor and hidden directories below start.
func poolGoFiles(t *testing.T, start string) map[string][]string {
	t.Helper()
	byDir := map[string][]string{}
	err := filepath.WalkDir(start, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if p != start && (name == "testdata" || name == "vendor" || name == "node_modules" ||
				strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_")) {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(p, ".go") {
			byDir[filepath.Dir(p)] = append(byDir[filepath.Dir(p)], p)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return byDir
}

// scanPoolTree runs rule 1 over every Go file under start, with paths
// relative to start, which is the repository root or a fixture tree laid
// out as one.
func scanPoolTree(t *testing.T, start string, idx map[string]pkgConsts, env string) []poolRef {
	t.Helper()
	var out []poolRef
	fset := token.NewFileSet()
	for _, paths := range poolGoFiles(t, start) {
		files := parseDir(t, fset, paths)
		// A directory holds a package and maybe its external test package;
		// each folds its own constants.
		byPkg := map[string][]int{}
		for i, f := range files {
			byPkg[f.Name.Name] = append(byPkg[f.Name.Name], i)
		}
		for _, is := range byPkg {
			pkgFiles := make([]*ast.File, len(is))
			scopes := make([]map[string]string, len(is))
			for j, i := range is {
				pkgFiles[j] = files[i]
				scopes[j] = importScope(files[i], idx)
			}
			pkg := packageConsts(pkgFiles, scopes)
			for j, i := range is {
				rel, err := filepath.Rel(start, paths[i])
				if err != nil {
					t.Fatal(err)
				}
				out = append(out, scanPoolFile(fset, filepath.ToSlash(rel), pkgFiles[j], fileConsts(pkg, scopes[j]), env)...)
			}
		}
	}
	slices.SortFunc(out, func(a, b poolRef) int { return strings.Compare(a.String(), b.String()) })
	return slices.Compact(out)
}

// importName returns the name f gives the package at importPath: its alias,
// "." for a dot import, def for a plain import, or "" when f does not import
// it.
func importName(f *ast.File, importPath, def string) string {
	for _, is := range f.Imports {
		if p, _ := strconv.Unquote(is.Path.Value); p == importPath {
			if is.Name != nil {
				return is.Name.Name
			}
			return def
		}
	}
	return ""
}

// pkgRef reports whether e names member of the package imported as local:
// local.member, or a bare member under a dot import.
func pkgRef(e ast.Expr, local, member string) bool {
	switch e := ast.Unparen(e).(type) {
	case *ast.SelectorExpr:
		x, ok := e.X.(*ast.Ident)
		return ok && local != "" && local != "." && x.Name == local && e.Sel.Name == member
	case *ast.Ident:
		return local == "." && e.Name == member
	}
	return false
}

// scanPoolFile runs rule 1 over one file at rel. consts holds the string
// constants the file sees.
func scanPoolFile(fset *token.FileSet, rel string, f *ast.File, consts map[string]string, env string) []poolRef {
	readName := importName(f, readPkgPath, "read")
	cfgName := importName(f, configPkgPath, "config")
	line := func(n ast.Node) int { return fset.Position(n.Pos()).Line }

	// openpool and key: every reference, minus declared names, struct keys
	// and the Sel of another package's selector (pgutil.OpenPool).
	type ref struct {
		node ast.Expr
		what string
	}
	var refs []ref
	var visit func(n ast.Node) bool
	visit = func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.SelectorExpr:
			switch {
			case pkgRef(n, readName, openPoolName):
				refs = append(refs, ref{n, "openpool"})
			case pkgRef(n, cfgName, readerKeyName):
				refs = append(refs, ref{n, "key"})
			}
			ast.Inspect(n.X, visit)
			return false
		case *ast.Ident:
			switch {
			case pkgRef(n, readName, openPoolName):
				refs = append(refs, ref{n, "openpool"})
			case pkgRef(n, cfgName, readerKeyName):
				refs = append(refs, ref{n, "key"})
			}
		case *ast.FuncDecl:
			if n.Recv != nil {
				ast.Inspect(n.Recv, visit)
			}
			ast.Inspect(n.Type, visit)
			if n.Body != nil {
				ast.Inspect(n.Body, visit)
			}
			return false
		case *ast.Field:
			ast.Inspect(n.Type, visit)
			return false
		case *ast.ValueSpec:
			if n.Type != nil {
				ast.Inspect(n.Type, visit)
			}
			for _, v := range n.Values {
				ast.Inspect(v, visit)
			}
			return false
		case *ast.KeyValueExpr:
			if _, ok := n.Key.(*ast.Ident); !ok {
				ast.Inspect(n.Key, visit)
			}
			ast.Inspect(n.Value, visit)
			return false
		}
		return true
	}
	ast.Inspect(f, visit)

	calls := map[ast.Expr]*ast.CallExpr{}
	ast.Inspect(f, func(n ast.Node) bool {
		if c, ok := n.(*ast.CallExpr); ok {
			calls[ast.Unparen(c.Fun)] = c
		}
		return true
	})

	// The one allowed call: the only openpool reference of main.go, called.
	var rootCall *ast.CallExpr
	if rel == compositeRoot {
		var pools []ref
		for _, r := range refs {
			if r.what == "openpool" {
				pools = append(pools, r)
			}
		}
		if len(pools) == 1 {
			rootCall = calls[pools[0].node]
		}
	}
	inRootArgs := func(n ast.Node) bool {
		return rootCall != nil && n.Pos() >= rootCall.Lparen && n.End() <= rootCall.Rparen
	}

	var out []poolRef
	add := func(n ast.Node, what string) { out = append(out, poolRef{rel, line(n), what}) }
	for _, r := range refs {
		if r.what == "openpool" && rootCall != nil && calls[r.node] == rootCall {
			continue
		}
		add(r.node, r.what)
	}

	// env: string constants, folded as rule 2 folds them.
	var walk func(root ast.Node, consts map[string]string)
	walk = func(root ast.Node, consts map[string]string) {
		ast.Inspect(root, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.FuncDecl:
				if n.Body != nil {
					walk(n.Body, localConsts(n.Body, consts))
				}
				return false
			case *ast.FuncLit:
				walk(n.Body, localConsts(n.Body, consts))
				return false
			case *ast.BasicLit, *ast.BinaryExpr:
				if s, ok := stringValue(n.(ast.Expr), consts); ok && strings.Contains(s, env) {
					add(n, "env")
				}
			case *ast.CallExpr:
				if s, ok := sprintfValue(n, consts); ok && strings.Contains(s, env) {
					add(n, "env")
				}
			}
			return true
		})
	}
	walk(f, consts)

	// dsnfield: names that hold a config.AuditReader in this file, then
	// their .DatabaseURL selectors and the type's composite literals.
	isCfgType := func(e ast.Expr) bool {
		if s, ok := e.(*ast.StarExpr); ok {
			e = s.X
		}
		return pkgRef(e, cfgName, readerCfgType)
	}
	isLoad := func(e ast.Expr) bool {
		c, ok := ast.Unparen(e).(*ast.CallExpr)
		return ok && pkgRef(c.Fun, cfgName, readerLoader)
	}
	isLit := func(e ast.Expr) bool {
		e = ast.Unparen(e)
		if u, ok := e.(*ast.UnaryExpr); ok && u.Op == token.AND {
			e = ast.Unparen(u.X)
		}
		c, ok := e.(*ast.CompositeLit)
		return ok && c.Type != nil && isCfgType(c.Type)
	}
	held := map[string]bool{}
	if cfgName != "" {
		ast.Inspect(f, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.AssignStmt:
				for i, l := range n.Lhs {
					id, ok := l.(*ast.Ident)
					if !ok {
						continue
					}
					if i == 0 && len(n.Rhs) == 1 && isLoad(n.Rhs[0]) || len(n.Rhs) == len(n.Lhs) && isLit(n.Rhs[i]) {
						held[id.Name] = true
					}
				}
			case *ast.ValueSpec:
				for i, id := range n.Names {
					if n.Type != nil && isCfgType(n.Type) || i == 0 && len(n.Values) == 1 && isLoad(n.Values[0]) ||
						i < len(n.Values) && isLit(n.Values[i]) {
						held[id.Name] = true
					}
				}
			case *ast.Field:
				if isCfgType(n.Type) {
					for _, id := range n.Names {
						held[id.Name] = true
					}
				}
			}
			return true
		})
		ast.Inspect(f, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.SelectorExpr:
				if n.Sel.Name != readerDSNName {
					return true
				}
				x := ast.Unparen(n.X)
				if s, ok := x.(*ast.StarExpr); ok {
					x = ast.Unparen(s.X)
				}
				hit := isLit(x)
				switch x := x.(type) {
				case *ast.Ident:
					hit = hit || held[x.Name]
				case *ast.SelectorExpr:
					hit = hit || held[x.Sel.Name]
				}
				if hit && !inRootArgs(n) {
					add(n.Sel, "dsnfield")
				}
			case *ast.CompositeLit:
				if n.Type == nil || !isCfgType(n.Type) {
					return true
				}
				// A positional element or the DatabaseURL key sets the DSN.
				for _, e := range n.Elts {
					kv, keyed := e.(*ast.KeyValueExpr)
					sets := !keyed
					if keyed {
						k, ok := kv.Key.(*ast.Ident)
						sets = ok && k.Name == readerDSNName
					}
					if sets && !inRootArgs(e) {
						add(e, "dsnfield")
					}
				}
			}
			return true
		})
	}

	// The allowlist.
	inReader := strings.HasPrefix(rel, readerPkgDir+"/")
	inConfig := path.Dir(rel) == configPkgDir
	testOK := poolGuardTestFiles(rel)
	kept := out[:0]
	for _, r := range out {
		switch {
		case slices.Contains(testOK, r.what):
		case r.what == "openpool" && inReader:
		case r.what != "openpool" && inConfig:
		default:
			kept = append(kept, r)
		}
	}
	return kept
}

// TestReaderPoolOnlyInReader is the A1-10 rule 1 gate over the repository.
func TestReaderPoolOnlyInReader(t *testing.T) {
	root := repoRoot(t)
	idx := constIndex(t, root)
	env := poolGuardNames(t, root, idx)
	for _, r := range scanPoolTree(t, root, idx, env) {
		t.Errorf("%s: %s (ADR-004-A1 A1-10 rule 1, AR-TICK-549-01)", r, poolGuardHow(r.what))
	}
}

// TestReaderPoolGuardFixture is the positive control: in every tree under
// testdata/poolguard, each `// want: <what>` line is reported and nothing
// else is.
func TestReaderPoolGuardFixture(t *testing.T) {
	root := repoRoot(t)
	idx := constIndex(t, root)
	env := poolGuardNames(t, root, idx)
	base := filepath.Join(root, "internal", "audit", "read", "testdata", "poolguard")
	trees, err := os.ReadDir(base)
	if err != nil {
		t.Fatal(err)
	}
	total := 0
	for _, tree := range trees {
		if !tree.IsDir() {
			continue
		}
		t.Run(tree.Name(), func(t *testing.T) {
			dir := filepath.Join(base, tree.Name())
			got := scanPoolTree(t, dir, idx, env)
			var want []poolRef
			err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
				if err != nil || d.IsDir() || !strings.HasSuffix(p, ".go") {
					return err
				}
				b, err := os.ReadFile(p)
				if err != nil {
					return err
				}
				r, _ := filepath.Rel(dir, p)
				for i, l := range strings.Split(string(b), "\n") {
					for _, m := range wantRE.FindAllStringSubmatch(l, -1) {
						for _, w := range strings.Fields(m[1]) {
							want = append(want, poolRef{filepath.ToSlash(r), i + 1, w})
						}
					}
				}
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			total += len(want)
			for _, w := range want {
				if !slices.Contains(got, w) {
					t.Errorf("planted reference not reported: %s", w)
				}
			}
			for _, g := range got {
				if !slices.Contains(want, g) {
					t.Errorf("look-alike reported: %s", g)
				}
			}
		})
	}
	if total < 20 {
		t.Fatalf("the fixture plants %d references; it lost some", total)
	}
}

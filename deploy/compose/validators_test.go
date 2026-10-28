package compose

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// keycloakPkg is the package whose New builds a token verifier.
const keycloakPkg = "github.com/Navaladi2019/TicketingSystem/internal/identity/keycloak"

// goPkg is what the check needs of one package in the module, read from its
// non-test files whatever their build tags, so a tagged file cannot hide a
// call site.
type goPkg struct {
	Main    bool
	Imports []string
	// Verifiers are the "file:line" of each reference to keycloak.New.
	Verifiers []string
}

// loadModule parses every non-test .go file under root (the module root) into
// packages keyed by import path. It skips testdata, vendor, hidden dirs and
// nested modules, none of which build into this module's binaries.
func loadModule(root string) (map[string]*goPkg, error) {
	b, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return nil, err
	}
	var mod string
	for _, l := range strings.Split(string(b), "\n") {
		if f := strings.Fields(l); len(f) == 2 && f[0] == "module" {
			mod = f[1]
		}
	}
	if mod == "" {
		return nil, fmt.Errorf("%s/go.mod: no module line", root)
	}
	pkgs := map[string]*goPkg{}
	fset := token.NewFileSet()
	err = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := d.Name()
		if d.IsDir() {
			if p == root {
				return nil
			}
			if name == "testdata" || name == "vendor" || strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") {
				return filepath.SkipDir
			}
			if _, err := os.Stat(filepath.Join(p, "go.mod")); err == nil {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			return nil
		}
		f, err := parser.ParseFile(fset, p, nil, parser.SkipObjectResolution)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, filepath.Dir(p))
		if err != nil {
			return err
		}
		ip := path.Join(mod, filepath.ToSlash(rel))
		pkg := pkgs[ip]
		if pkg == nil {
			pkg = &goPkg{}
			pkgs[ip] = pkg
		}
		pkg.Main = pkg.Main || f.Name.Name == "main"
		file, _ := filepath.Rel(root, p)
		for _, line := range verifierRefs(fset, f, ip) {
			pkg.Verifiers = append(pkg.Verifiers, fmt.Sprintf("%s:%d", filepath.ToSlash(file), line))
		}
		for _, im := range f.Imports {
			v, _ := strconv.Unquote(im.Path.Value)
			pkg.Imports = append(pkg.Imports, v)
		}
		return nil
	})
	return pkgs, err
}

// verifierRefs returns the line of every reference to keycloak.New in f, a
// file of package ip: a call, or the function taken as a value, through any
// import name, a dot import, or unqualified inside keycloak itself.
func verifierRefs(fset *token.FileSet, f *ast.File, ip string) []int {
	names := map[string]bool{}
	dot := ip == keycloakPkg
	for _, im := range f.Imports {
		if v, _ := strconv.Unquote(im.Path.Value); v != keycloakPkg {
			continue
		}
		switch {
		case im.Name == nil:
			names[path.Base(keycloakPkg)] = true
		case im.Name.Name == ".":
			dot = true
		case im.Name.Name != "_":
			names[im.Name.Name] = true
		}
	}
	// Names that spell New but are not a use of it: selector fields and
	// methods, declarations and struct fields.
	notRef := map[*ast.Ident]bool{}
	ast.Inspect(f, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.SelectorExpr:
			notRef[n.Sel] = true
		case *ast.FuncDecl:
			notRef[n.Name] = true
		case *ast.Field:
			for _, id := range n.Names {
				notRef[id] = true
			}
		}
		return true
	})
	var lines []int
	ast.Inspect(f, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.SelectorExpr:
			if x, ok := n.X.(*ast.Ident); ok && names[x.Name] && n.Sel.Name == "New" {
				lines = append(lines, fset.Position(n.Pos()).Line)
			}
		case *ast.Ident:
			if dot && n.Name == "New" && !notRef[n] {
				lines = append(lines, fset.Position(n.Pos()).Line)
			}
		}
		return true
	})
	return lines
}

// checkVerifierBinaries returns one message per main package that reaches a
// keycloak.New call site, directly or through any in-module import, but is
// not in validators; and one per validator no main package of that name
// reaches, so a stale entry does not pass for coverage.
func checkVerifierBinaries(pkgs map[string]*goPkg, validators []string) []string {
	listed := map[string]bool{}
	for _, v := range validators {
		listed[v] = true
	}
	var mains []string
	for ip, p := range pkgs {
		if p.Main {
			mains = append(mains, ip)
		}
	}
	sort.Strings(mains)
	var errs []string
	reached := map[string]bool{}
	for _, m := range mains {
		var sites []string
		seen := map[string]bool{}
		var walk func(string)
		walk = func(ip string) {
			p := pkgs[ip]
			if p == nil || seen[ip] {
				return
			}
			seen[ip] = true
			sites = append(sites, p.Verifiers...)
			for _, im := range p.Imports {
				walk(im)
			}
		}
		walk(m)
		if len(sites) == 0 {
			continue
		}
		sort.Strings(sites)
		bin := path.Base(m)
		reached[bin] = true
		if !listed[bin] {
			errs = append(errs, fmt.Sprintf("binary %s (%s) builds a token verifier at %s but is not in tokenValidators(); add it there and set its NAVA_OIDC_ISSUERS in compose.yaml",
				bin, m, strings.Join(sites, ", ")))
		}
	}
	for _, v := range validators {
		if !reached[v] {
			errs = append(errs, fmt.Sprintf("tokenValidators() lists %s, but no main package of that name reaches keycloak.New; remove or rename it", v))
		}
	}
	return errs
}

// TestEveryVerifierBinaryIsATokenValidator (TICK-304): tokenValidators() is a
// hand-kept list, so TestEveryTokenValidatorTrustsPinnedIssuer only checks the
// services someone remembered to add. This finds every non-test keycloak.New
// reference in the module, follows the import graph up to each main package,
// and fails if a binary that can build a verifier is not on the list, since it
// could build one from Go defaults with no pinned issuer in compose.yaml.
func TestEveryVerifierBinaryIsATokenValidator(t *testing.T) {
	t.Parallel()
	pkgs, err := loadModule(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	if p := pkgs[keycloakPkg]; p == nil {
		t.Fatalf("package %s not found; update keycloakPkg if it moved", keycloakPkg)
	}
	for _, e := range checkVerifierBinaries(pkgs, tokenValidators()) {
		t.Error(e)
	}
}

// TestVerifierRefs: each way of naming keycloak.New is seen, and nothing else
// called New is.
func TestVerifierRefs(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, ip, src string
		want          int
	}{
		{"call", "x/cmd/a", `package main; import "` + keycloakPkg + `"; func f() { keycloak.New(nil, keycloak.Config{}, nil) }`, 1},
		{"renamed import", "x/cmd/a", `package main; import kc "` + keycloakPkg + `"; func f() { kc.New(nil, kc.Config{}, nil) }`, 1},
		{"function value", "x/cmd/a", `package main; import "` + keycloakPkg + `"; var g = keycloak.New`, 1},
		{"dot import", "x/cmd/a", `package main; import . "` + keycloakPkg + `"; func f() { New(nil, Config{}, nil) }`, 1},
		{"inside keycloak", keycloakPkg, `package keycloak; func New() {}; func g() { New() }`, 1},
		{"another New", "x/cmd/a", `package main; import "errors"; import "` + keycloakPkg + `"; var _ keycloak.Config; var e = errors.New("x")`, 0},
		{"shadowing selector", "x/cmd/a", `package main; import "` + keycloakPkg + `"; var _ keycloak.Config; func f(s struct{ keycloak struct{ New int } }) { _ = s.keycloak.New }`, 0},
		{"no import", "x/cmd/a", `package main; func New() {}; func f() { New() }`, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, "x.go", c.src, parser.SkipObjectResolution)
			if err != nil {
				t.Fatal(err)
			}
			if got := verifierRefs(fset, f, c.ip); len(got) != c.want {
				t.Errorf("got %d references at lines %v, want %d", len(got), got, c.want)
			}
		})
	}
}

// TestCheckVerifierBinariesCatchesUnlisted: a binary that reaches keycloak.New,
// directly or through a library, fails the check unless it is listed; a
// listed name with no verifier fails too.
func TestCheckVerifierBinariesCatchesUnlisted(t *testing.T) {
	t.Parallel()
	const m = "example.com/m/"
	good := func() map[string]*goPkg {
		return map[string]*goPkg{
			m + "cmd/nava-core":    {Main: true, Imports: []string{keycloakPkg}, Verifiers: []string{"cmd/nava-core/identity.go:52"}},
			m + "cmd/nava-onboard": {Main: true, Imports: []string{m + "internal/onboard"}, Verifiers: []string{"cmd/nava-onboard/main.go:137"}},
			m + "cmd/nava-worker":  {Main: true, Imports: []string{m + "internal/onboard"}},
			m + "internal/onboard": {Imports: []string{keycloakPkg}},
			m + "internal/lib":     {Imports: []string{keycloakPkg}, Verifiers: []string{"internal/lib/lib.go:9"}},
			keycloakPkg:            {},
		}
	}
	validators := []string{"nava-core", "nava-onboard"}
	if errs := checkVerifierBinaries(good(), validators); len(errs) != 0 {
		t.Fatalf("good module: %v", errs)
	}
	cases := []struct {
		name   string
		mutate func(map[string]*goPkg)
		want   string
	}{
		{"new binary calls keycloak.New", func(p map[string]*goPkg) {
			p[m+"cmd/nava-rogue"] = &goPkg{Main: true, Verifiers: []string{"cmd/nava-rogue/main.go:10"}}
		}, "binary nava-rogue (" + m + "cmd/nava-rogue) builds a token verifier at cmd/nava-rogue/main.go:10"},
		{"unlisted binary reaches it through a library", func(p map[string]*goPkg) {
			p[m+"cmd/nava-worker"].Imports = append(p[m+"cmd/nava-worker"].Imports, m+"internal/lib")
		}, "binary nava-worker (" + m + "cmd/nava-worker) builds a token verifier at internal/lib/lib.go:9"},
		{"library reached through a cycle-free chain", func(p map[string]*goPkg) {
			p[m+"internal/onboard"].Imports = append(p[m+"internal/onboard"].Imports, m+"internal/lib")
		}, "binary nava-worker"},
		{"listed validator no longer builds a verifier", func(p map[string]*goPkg) {
			p[m+"cmd/nava-core"].Verifiers = nil
		}, "tokenValidators() lists nava-core, but no main package"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			p := good()
			c.mutate(p)
			errs := checkVerifierBinaries(p, validators)
			if !strings.Contains(strings.Join(errs, "\n"), c.want) {
				t.Errorf("errors %q, want one containing %q", errs, c.want)
			}
		})
	}
}

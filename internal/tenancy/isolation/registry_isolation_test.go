package isolation

import (
	"go/ast"
	"go/build/constraint"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/db"
	"github.com/Navaladi2019/TicketingSystem/internal/db/catalog"
	"github.com/Navaladi2019/TicketingSystem/internal/db/grants"
)

// TestRegistryIsWellFormed: every path has a unique name, a covered ADR test
// id and exactly one call; SQL paths run as a manifest role that may connect
// to the platform database; HTTP paths name an /api/v1 route.
func TestRegistryIsWellFormed(t *testing.T) {
	m, err := grants.Parse(db.Grants())
	if err != nil {
		t.Fatal(err)
	}
	cov := Coverage()
	seen := map[string]bool{}
	for _, p := range Paths() {
		if p.Name == "" || seen[p.Name] {
			t.Errorf("path %q: empty or duplicate name", p.Name)
		}
		seen[p.Name] = true
		if _, ok := cov[p.Case]; !ok {
			t.Errorf("path %q: case %q is not in Coverage", p.Name, p.Case)
		}
		calls := 0
		for _, set := range []bool{p.SQL != nil, p.HTTP != nil, p.Check != nil, p.DB != nil, p.API != nil, p.Stack != ""} {
			if set {
				calls++
			}
		}
		if calls != 1 {
			t.Errorf("path %q: %d calls, want exactly one of SQL, HTTP, Check, DB, API, Stack", p.Name, calls)
		}
		if p.SQL != nil {
			r, ok := m.Roles[p.Role]
			if !ok || !slices.Contains(r.Database, "CONNECT") {
				t.Errorf("path %q: role %q is not a manifest role with CONNECT", p.Name, p.Role)
			}
			if p.SQL.Expect != OwnRowsOnly && p.SQL.Expect != Denied {
				t.Errorf("path %q: no expectation", p.Name)
			}
		}
		if p.HTTP != nil && (p.HTTP.Method == "" || !strings.HasPrefix(p.HTTP.Path, "/api/v1/")) {
			t.Errorf("path %q: an HTTP path needs a method and an /api/v1 path", p.Name)
		}
		if p.HTTP != nil && p.HTTP.Token != nil && (p.HTTP.Token.WantErr == nil || p.HTTP.Token.WantReason == "") {
			t.Errorf("path %q: a Token path needs WantErr and WantReason, or any 401 would pass", p.Name)
		}
		if p.StackTarget != "" {
			target, ok := strings.CutPrefix(p.StackTarget, "make ")
			switch {
			case p.Stack == "":
				t.Errorf("path %q: StackTarget without Stack", p.Name)
			case !ok || !makeTargets(t)[target]:
				t.Errorf("path %q: StackTarget %q is not a make target of the Makefile", p.Name, p.StackTarget)
			}
		}
	}
}

// makeTargets are the rule names of the repository's Makefile.
func makeTargets(t *testing.T) map[string]bool {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "Makefile"))
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]bool{}
	for _, line := range strings.Split(string(raw), "\n") {
		if name, _, ok := strings.Cut(line, ":"); ok && name != "" && !strings.ContainsAny(name, " \t=$#.%") {
			out[name] = true
		}
	}
	return out
}

// TestExemptionsAreAllowListed: an exemption names an ADR-001 A.3 policy, on
// its table, for its command and role, whose USING clause is true. Anything
// else would let the suite pass a cross-tenant read that no ADR accepted.
func TestExemptionsAreAllowListed(t *testing.T) {
	allow := catalog.AllowList()
	for _, e := range Exemptions() {
		p, ok := allow[e.Table+"."+e.Policy]
		if !ok {
			t.Errorf("%+v: %s.%s is not on the allow-list", e, e.Table, e.Policy)
			continue
		}
		if p.Cmd != e.Command || !slices.Contains(strings.Split(p.Roles, ","), e.Role) {
			t.Errorf("%+v: the allow-list has it FOR %s TO %s", e, p.Cmd, p.Roles)
		}
		if e.Command == "SELECT" && p.Qual != "true" {
			t.Errorf("%+v: USING (%s) is not true; the generic checks should see no row", e, p.Qual)
		}
	}
}

// TestCursorPathsHold runs the CI-8 cursor checks without a database, so
// ci-unit catches a cursor regression before the isolation stage.
func TestCursorPathsHold(t *testing.T) {
	for _, p := range Paths() {
		if p.Check == nil {
			continue
		}
		if err := p.Check(t.Context()); err != nil {
			t.Errorf("%s: %v", p.Name, err)
		}
	}
}

// TestStackPathsExist: every Stack path names a Test function of
// internal/identitystack in a file built under the identitystack tag, which
// make stack-test-identity sets. Renaming or deleting the live-Keycloak test,
// or moving it to a file under another tag, fails here (ci-unit) instead of
// leaving a registration that proves nothing.
func TestStackPathsExist(t *testing.T) {
	dir := filepath.Join("..", "..", "identitystack")
	files, err := filepath.Glob(filepath.Join(dir, "*_test.go"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no tests in %s: %v", dir, err)
	}
	tests := map[string]bool{}
	fset := token.NewFileSet()
	for _, f := range files {
		af, err := parser.ParseFile(fset, f, nil, parser.SkipObjectResolution|parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}
		if !builtWithIdentityStack(af) {
			continue
		}
		for _, d := range af.Decls {
			if fn, ok := d.(*ast.FuncDecl); ok && fn.Recv == nil && strings.HasPrefix(fn.Name.Name, "Test") {
				tests[fn.Name.Name] = true
			}
		}
	}
	n := 0
	for _, p := range Paths() {
		if p.Stack == "" {
			continue
		}
		n++
		if !tests[p.Stack] {
			t.Errorf("path %q: %s is not a test in an identitystack-tagged file of internal/identitystack", p.Name, p.Stack)
		}
	}
	if n == 0 {
		t.Error("no Stack path registered: T26 cases 3, 4, 5 and 7 are missing")
	}
}

// builtWithIdentityStack reports whether f has a //go:build line that holds
// with the identitystack tag and fails without it, so the file's tests run in
// make stack-test-identity and only there.
func builtWithIdentityStack(f *ast.File) bool {
	for _, cg := range f.Comments {
		if cg.Pos() > f.Package {
			break
		}
		for _, c := range cg.List {
			if !constraint.IsGoBuild(c.Text) {
				continue
			}
			expr, err := constraint.Parse(c.Text)
			if err != nil {
				return false
			}
			return expr.Eval(func(tag string) bool { return tag == "identitystack" }) &&
				!expr.Eval(func(string) bool { return false })
		}
	}
	return false
}

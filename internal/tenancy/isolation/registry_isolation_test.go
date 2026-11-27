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
	"github.com/Navaladi2019/TicketingSystem/internal/db/audited"
	"github.com/Navaladi2019/TicketingSystem/internal/db/catalog"
)

// TestRegistryIsWellFormed: every path has a unique name, a covered ADR test
// id and exactly one call; SQL paths run as a manifest role that may connect
// to the platform database; HTTP paths name an /api/v1 route.
func TestRegistryIsWellFormed(t *testing.T) {
	m, _, err := audited.Load(db.Grants(), db.AuditedTables())
	if err != nil {
		t.Fatal(err)
	}
	cov := Coverage()
	seen := map[string]bool{}
	keys := map[string]string{} // StoredUnderA Idempotency-Key -> path name
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
		// C4 (CTO D-P10 on TICK-366): every StoredUnderA call has its own
		// Idempotency-Key. The request hash covers method, path and body, not
		// headers, so a shared key lets a later case replay an earlier case's
		// stored response without the handler reading its forged field.
		if p.HTTP != nil && p.HTTP.Expect == StoredUnderA {
			key := p.HTTP.Header.Get("Idempotency-Key")
			switch {
			case key == "":
				t.Errorf("path %q: a StoredUnderA call needs an Idempotency-Key", p.Name)
			case keys[key] != "":
				t.Errorf("path %q: Idempotency-Key %q is shared with path %q", p.Name, key, keys[key])
			}
			keys[key] = p.Name
			if !strings.HasPrefix(p.HTTP.Unknown, "/api/v1/") || p.HTTP.Unknown == p.HTTP.Path {
				t.Errorf("path %q: a StoredUnderA call needs Unknown, another /api/v1 path, for its 404 control", p.Name)
			}
		}
		if p.HTTP != nil && p.HTTP.Expect != 0 && p.HTTP.Expect != StoredUnderA {
			t.Errorf("path %q: an HTTP call's Expect is zero or StoredUnderA", p.Name)
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

// makeTargets are the rule names of the repository's Makefile and its
// make/*.mk fragments.
func makeTargets(t *testing.T) map[string]bool {
	t.Helper()
	root := filepath.Join("..", "..", "..")
	files, err := filepath.Glob(filepath.Join(root, "make", "*.mk"))
	if err != nil {
		t.Fatal(err)
	}
	files = append(files, filepath.Join(root, "Makefile"))
	out := map[string]bool{}
	for _, f := range files {
		raw, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for line := range strings.SplitSeq(string(raw), "\n") {
			if name, _, ok := strings.Cut(line, ":"); ok && name != "" && !strings.ContainsAny(name, " \t=$#.%") {
				out[name] = true
			}
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

// TestTICK1070EstimatesRegistered: entity_record_estimates keeps its
// isolation-registry rows (TICK-1070, ADR-010-A2 A2-9). Each reader reads
// only its own tenant's rows; the app's write, the worker's cross-tenant
// insert, upsert, update and delete, and the worker's read of a binding
// column other than tenant_id are each refused; and the worker's
// binding_lookup read is an exemption the allow-list backs.
func TestTICK1070EstimatesRegistered(t *testing.T) {
	type want struct {
		role   string
		expect Expect
		verb   string // the statement's first word
		table  string
	}
	wants := []want{
		{"nava_app", OwnRowsOnly, "SELECT", "entity_record_estimates"},
		{"nava_app", Denied, "SELECT", "entity_record_estimates"},
		{"nava_app", Denied, "UPDATE", "entity_record_estimates"},
		{"nava_worker", OwnRowsOnly, "SELECT", "entity_record_estimates"},
		{"nava_worker", Denied, "INSERT", "entity_record_estimates"},
		{"nava_worker", Denied, "UPDATE", "entity_record_estimates"},
		{"nava_worker", Denied, "DELETE", "entity_record_estimates"},
		{"nava_worker", Denied, "SELECT", "tenant_identity_binding"},
	}
	for _, w := range wants {
		found := false
		for _, p := range Paths() {
			if p.SQL == nil || p.Role != w.role || p.SQL.Expect != w.expect {
				continue
			}
			q := strings.TrimSpace(p.SQL.Query)
			if strings.HasPrefix(q, w.verb+" ") && strings.Contains(q, " "+w.table) &&
				strings.HasPrefix(p.Name, w.table+": ") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("no registry path: %s %s on %s, expect %v", w.role, w.verb, w.table, w.expect)
		}
	}
	upsert := false
	for _, p := range Paths() {
		if p.SQL != nil && p.Role == "nava_worker" && p.SQL.Expect == Denied && strings.Contains(p.SQL.Query, "entity_record_estimates") &&
			strings.Contains(p.SQL.Query, "ON CONFLICT (tenant_id, entity_definition_id) DO UPDATE") {
			upsert = true
		}
	}
	if !upsert {
		t.Error("no registry path for the worker's cross-tenant upsert of entity_record_estimates")
	}
	if !slices.Contains(Exemptions(), Exemption{"nava_worker", "public.tenant_identity_binding", "SELECT", "binding_lookup"}) {
		t.Error("nava_worker's binding_lookup read is not an exemption")
	}
}

// TestCursorPathsHold runs the CI-8 cursor checks without a database, so
// ci-unit catches a cursor regression before the isolation stage. A pending
// check is listed, not run, as in TestRegisteredPaths.
func TestCursorPathsHold(t *testing.T) {
	for _, p := range Paths() {
		if p.Check == nil || p.Pending != "" {
			continue
		}
		if err := p.Check(t.Context()); err != nil {
			t.Errorf("%s: %v", p.Name, err)
		}
	}
}

// TestStackPathsExist: every Stack path names a Test function of
// internal/identitystack in a file built under its target's tags and not
// without them. Renaming or deleting the
// live-Keycloak test, or moving it to a file under another tag, fails here
// (ci-unit) instead of leaving a registration that proves nothing.
func TestStackPathsExist(t *testing.T) {
	dir := filepath.Join("..", "..", "identitystack")
	files, err := filepath.Glob(filepath.Join(dir, "*_test.go"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no tests in %s: %v", dir, err)
	}
	// The build tags each Stack target runs internal/identitystack with: make
	// identity-attacks adds ci5, the tag TestCI5 sits behind so make
	// stack-test-identity runs the rest under suiteran with no -skip
	// (TICK-283).
	stackTargetTags := map[string][]string{
		DefaultStackTarget:    {"identitystack"},
		identityAttacksTarget: {"identitystack", "ci5"},
	}
	fset := token.NewFileSet()
	parsed := make([]*ast.File, 0, len(files))
	for _, f := range files {
		af, err := parser.ParseFile(fset, f, nil, parser.SkipObjectResolution|parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}
		parsed = append(parsed, af)
	}
	// tests[target] are the Test functions the target's tags build.
	tests := map[string]map[string]bool{}
	for target, tags := range stackTargetTags {
		tests[target] = map[string]bool{}
		for _, af := range parsed {
			if !builtOnlyWith(af, tags) {
				continue
			}
			for _, d := range af.Decls {
				if fn, ok := d.(*ast.FuncDecl); ok && fn.Recv == nil && strings.HasPrefix(fn.Name.Name, "Test") {
					tests[target][fn.Name.Name] = true
				}
			}
		}
	}
	n := 0
	for _, p := range Paths() {
		if p.Stack == "" {
			continue
		}
		n++
		target := p.StackTarget
		if target == "" {
			target = DefaultStackTarget
		}
		built, ok := tests[target]
		if !ok {
			t.Errorf("path %q: StackTarget %q has no build tags here", p.Name, target)
			continue
		}
		if !built[p.Stack] {
			t.Errorf("path %q: %s is not a test in a file of internal/identitystack built with %s's tags %q", p.Name, p.Stack, target, stackTargetTags[target])
		}
	}
	if n == 0 {
		t.Error("no Stack path registered: T26 cases 3, 4, 5 and 7 are missing")
	}
}

// builtOnlyWith reports whether f has a //go:build line that holds with tags
// and fails with no tag, so the file's tests run in a target that sets tags
// and in no plain go test.
func builtOnlyWith(f *ast.File, tags []string) bool {
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
			return expr.Eval(func(tag string) bool { return slices.Contains(tags, tag) }) &&
				!expr.Eval(func(string) bool { return false })
		}
	}
	return false
}

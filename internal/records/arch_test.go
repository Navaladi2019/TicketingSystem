// Package records holds the architecture test for the record write path
// (PLAN-003 D-3-6, Epic 05 AC5, ADR-001 I4, TICK-356): only
// internal/records/pipeline and its store write the record tables, and only
// the callers D-3-6 names import them.
//
// The check is static and runs in `task test`. It parses every non-test Go
// file of the root module (build tags ignored, so a tagged file cannot hide a
// write) and reports:
//
//   - a string, constant, `+` concatenation or fmt.Sprintf result whose SQL
//     INSERTs into, UPDATEs, DELETEs from, MERGEs into, TRUNCATEs or COPYs
//     into a record table, matched case-insensitively with any whitespace,
//     with or without double quotes and a schema prefix;
//   - a pgx.Identifier literal naming a record table (the CopyFrom form);
//   - an import of internal/records/pipeline outside internal/server,
//     internal/workflow and internal/onboard, or of internal/records/store
//     outside the pipeline.
//
// testdata/writers plants every form; TestRecordWriterFixture proves each one
// is reported, so the check cannot rot into a test that passes on nothing.
package records

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"
)

const modulePath = "github.com/Navaladi2019/TicketingSystem"

// recordTable reports whether name (unquoted, lower case, any schema prefix
// removed) is one of the tables only the pipeline writes.
func recordTable(name string) bool {
	switch name {
	case "records", "audit_log", "record_counters", "comments", "record_links":
		return true
	}
	return false
}

// writerAllowed reports whether a file at rel (slash-separated, relative to
// the module root) may write the record tables. Only the pipeline and its
// store, not the rest of internal/records, so S10's list code is not allowed
// by accident; and migrations.
func writerAllowed(rel string) bool {
	return underAny(rel, "internal/records/pipeline", "internal/records/store", "db/migrations")
}

// probeExempt is the tenant isolation suite's path registry: its INSERT and
// UPDATE statements are adversarial probes the suite runs as tenant A against
// tenant B and expects the database to deny (or to reach none of B's rows).
// They are not a write path. Editing that package already needs CTO sign-off
// (docs/delivery/review-paths.yaml).
func probeExempt(rel string) bool {
	return underAny(rel, "internal/tenancy/isolation")
}

// knownViolation is a write outside the pipeline that exists today and has a
// fix task. Each entry must still be found (TestRecordWritersOnlyInPipeline
// fails on a stale one), so the list can only shrink. A new write, even in a
// listed file, is still reported unless it matches an entry exactly.
type knownViolation struct {
	file, op, table, why string
}

// TODO(TODO-TICK-356-01): empty this list once TICK-411 moves outbox.Enqueue
// and cmd/demo-outbox off direct records writes.
func knownViolations() []knownViolation {
	return []knownViolation{
		{
			file: "internal/outbox/outbox.go", op: "update", table: "records",
			why: "outbox.Enqueue still bumps records.version (lockRecordSQL); D-3-6 leaves the bump to the pipeline. Fix: TICK-411.",
		},
		{
			file: "cmd/demo-outbox/main.go", op: "insert", table: "records",
			why: "the S7 demo seeds its record by INSERT; goes with the outbox.Enqueue fix, TICK-411.",
		},
	}
}

// importAllowed reports whether a package at dir (slash-separated, relative
// to the module root) may import the module-relative package imp.
func importAllowed(dir, imp string) bool {
	switch imp {
	case "internal/records/pipeline":
		return underAny(dir, "internal/records/pipeline", "internal/server", "internal/workflow", "internal/onboard")
	case "internal/records/store":
		return underAny(dir, "internal/records/store", "internal/records/pipeline")
	}
	return true
}

func underAny(rel string, roots ...string) bool {
	for _, r := range roots {
		if rel == r || strings.HasPrefix(rel, r+"/") {
			return true
		}
	}
	return false
}

// finding is one reported write or import.
type finding struct {
	file  string
	line  int
	op    string // insert, update, delete, merge, truncate, copy, identifier, import
	table string // or, for an import, the imported package
}

func (f finding) String() string {
	return fmt.Sprintf("%s:%d: %s %s", f.file, f.line, f.op, f.table)
}

const ident = `[a-z_][a-z0-9_$]*`

var (
	// A table reference: optional schema, optional ONLY, the table name.
	writeRE = regexp.MustCompile(`\b(insert\s+into|update|delete\s+from|merge\s+into)\s+(?:only\s+)?((?:` + ident + `\.)?` + ident + `)`)
	// TRUNCATE takes a list.
	truncateRE = regexp.MustCompile(`\btruncate(?:\s+table)?\s+((?:only\s+)?(?:` + ident + `\.)?` + ident + `(?:\s*\*)?(?:\s*,\s*(?:only\s+)?(?:` + ident + `\.)?` + ident + `(?:\s*\*)?)*)`)
	// COPY <table> [(cols)] FROM.
	copyRE = regexp.MustCompile(`\bcopy\s+((?:` + ident + `\.)?` + ident + `)\s*(?:\(|from\b)`)
	// SQL comments, removed before matching.
	lineCommentRE  = regexp.MustCompile(`--[^\n]*`)
	blockCommentRE = regexp.MustCompile(`(?s)/\*.*?\*/`)
	// A fmt verb, for substituting Sprintf arguments.
	verbRE = regexp.MustCompile(`%(?:\[\d+\])?[-+# 0]*\d*(?:\.\d+)?[a-zA-Z%]`)
)

// sqlWrites returns the (op, table) pairs of every record-table write in s.
func sqlWrites(s string) [][2]string {
	s = strings.ToLower(s)
	s = blockCommentRE.ReplaceAllString(s, " ")
	s = lineCommentRE.ReplaceAllString(s, " ")
	s = strings.ReplaceAll(s, `"`, "")
	s = strings.ReplaceAll(s, "`", "")
	var out [][2]string
	add := func(op, ref string) {
		ref = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(ref), "*"))
		ref = strings.TrimPrefix(ref, "only ")
		ref = strings.TrimSpace(ref)
		if i := strings.LastIndexByte(ref, '.'); i >= 0 {
			ref = ref[i+1:]
		}
		if recordTable(ref) {
			out = append(out, [2]string{op, ref})
		}
	}
	for _, m := range writeRE.FindAllStringSubmatch(s, -1) {
		add(strings.Fields(m[1])[0], m[2])
	}
	for _, m := range truncateRE.FindAllStringSubmatch(s, -1) {
		for _, ref := range strings.Split(m[1], ",") {
			add("truncate", ref)
		}
	}
	for _, m := range copyRE.FindAllStringSubmatch(s, -1) {
		add("copy", m[1])
	}
	return out
}

// scanFile reports the record-table writes and forbidden imports of one Go
// source file at rel (slash-separated, relative to the module root). consts
// holds the string constants of the file's package, so a write built from a
// named constant is seen too.
func scanFile(fset *token.FileSet, rel string, f *ast.File, consts map[string]string) []finding {
	var out []finding
	dir := path.Dir(rel)
	pgxNames := map[string]bool{}
	for _, is := range f.Imports {
		p, _ := strconv.Unquote(is.Path.Value)
		if p == "github.com/jackc/pgx/v5" {
			name := "pgx"
			if is.Name != nil {
				name = is.Name.Name
			}
			pgxNames[name] = true
		}
		if imp, ok := strings.CutPrefix(p, modulePath+"/"); ok && !importAllowed(dir, imp) {
			out = append(out, finding{rel, fset.Position(is.Pos()).Line, "import", imp})
		}
	}
	if writerAllowed(rel) || probeExempt(rel) {
		return out
	}

	report := func(n ast.Node, s string) {
		for _, w := range sqlWrites(s) {
			out = append(out, finding{rel, fset.Position(n.Pos()).Line, w[0], w[1]})
		}
	}
	ast.Inspect(f, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.BasicLit:
			if s, ok := stringValue(n, consts); ok {
				report(n, s)
			}
		case *ast.BinaryExpr:
			if s, ok := stringValue(n, consts); ok {
				report(n, s)
			}
		case *ast.CallExpr:
			if s, ok := sprintfValue(n, consts); ok {
				report(n, s)
			}
		case *ast.CompositeLit:
			if isPgxIdentifier(n.Type, pgxNames) && len(n.Elts) > 0 {
				if s, ok := stringValue(n.Elts[len(n.Elts)-1], consts); ok && recordTable(strings.ToLower(s)) {
					out = append(out, finding{rel, fset.Position(n.Pos()).Line, "identifier", strings.ToLower(s)})
				}
			}
		}
		return true
	})
	return dedupe(out)
}

// stringValue folds e to a constant string: a literal, a parenthesised or
// `+`-joined string, or a package string constant.
func stringValue(e ast.Expr, consts map[string]string) (string, bool) {
	switch e := e.(type) {
	case *ast.BasicLit:
		if e.Kind != token.STRING {
			return "", false
		}
		s, err := strconv.Unquote(e.Value)
		return s, err == nil
	case *ast.ParenExpr:
		return stringValue(e.X, consts)
	case *ast.Ident:
		s, ok := consts[e.Name]
		return s, ok
	case *ast.BinaryExpr:
		if e.Op != token.ADD {
			return "", false
		}
		l, lok := stringValue(e.X, consts)
		r, rok := stringValue(e.Y, consts)
		switch {
		case lok && rok:
			return l + r, true
		case lok:
			return l + " ? ", true
		case rok:
			return " ? " + r, true
		}
	}
	return "", false
}

// sprintfValue renders a fmt.Sprintf (or Fprintf, Errorf, ...) call with a
// constant format, substituting every constant string argument and "?" for
// the rest, so `fmt.Sprintf("INSERT INTO %s", "audit_log")` is seen.
func sprintfValue(c *ast.CallExpr, consts map[string]string) (string, bool) {
	sel, ok := c.Fun.(*ast.SelectorExpr)
	if !ok || !strings.HasSuffix(sel.Sel.Name, "printf") && sel.Sel.Name != "Errorf" {
		return "", false
	}
	for i, a := range c.Args {
		format, ok := stringValue(a, consts)
		if !ok || !strings.Contains(format, "%") {
			continue
		}
		args := c.Args[i+1:]
		next := 0
		return verbRE.ReplaceAllStringFunc(format, func(v string) string {
			if v == "%%" {
				return "%"
			}
			if next >= len(args) {
				return "?"
			}
			s, ok := stringValue(args[next], consts)
			next++
			if !ok {
				return "?"
			}
			return s
		}), true
	}
	return "", false
}

func isPgxIdentifier(t ast.Expr, pgxNames map[string]bool) bool {
	sel, ok := t.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Identifier" {
		return false
	}
	x, ok := sel.X.(*ast.Ident)
	return ok && pgxNames[x.Name]
}

func dedupe(fs []finding) []finding {
	sort.Slice(fs, func(i, j int) bool { return fs[i].String() < fs[j].String() })
	return slices.Compact(fs)
}

// packageConsts returns the string constants declared at package level in
// files (one package's non-test files).
func packageConsts(files []*ast.File) map[string]string {
	consts := map[string]string{}
	for range 2 { // a second pass resolves constants built from constants
		for _, f := range files {
			for _, d := range f.Decls {
				g, ok := d.(*ast.GenDecl)
				if !ok || g.Tok != token.CONST {
					continue
				}
				for _, sp := range g.Specs {
					vs := sp.(*ast.ValueSpec)
					for i, name := range vs.Names {
						if i < len(vs.Values) {
							if s, ok := stringValue(vs.Values[i], consts); ok {
								consts[name.Name] = s
							}
						}
					}
				}
			}
		}
	}
	return consts
}

// scanDir parses every non-test .go file under root (walking from start,
// skipping testdata, vendor, hidden directories and nested modules) and
// returns the findings with paths relative to root, as if the files lived at
// prefix instead of start when prefix is not empty.
func scanDir(t *testing.T, root, start, prefix string) []finding {
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
			if p != start {
				if _, err := os.Stat(filepath.Join(p, "go.mod")); err == nil {
					return filepath.SkipDir // another module, e.g. tools/navalint
				}
			}
			return nil
		}
		if strings.HasSuffix(p, ".go") && !strings.HasSuffix(p, "_test.go") {
			byDir[filepath.Dir(p)] = append(byDir[filepath.Dir(p)], p)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	var out []finding
	fset := token.NewFileSet()
	for _, paths := range byDir {
		files := make([]*ast.File, 0, len(paths))
		for _, p := range paths {
			f, err := parser.ParseFile(fset, p, nil, parser.SkipObjectResolution)
			if err != nil {
				t.Fatalf("parse %s: %v", p, err)
			}
			files = append(files, f)
		}
		consts := packageConsts(files)
		for i, f := range files {
			rel, err := filepath.Rel(root, paths[i])
			if err != nil {
				t.Fatal(err)
			}
			rel = filepath.ToSlash(rel)
			if prefix != "" {
				r, _ := filepath.Rel(start, paths[i])
				rel = path.Join(prefix, filepath.ToSlash(r))
			}
			out = append(out, scanFile(fset, rel, f, consts)...)
		}
	}
	return dedupe(out)
}

// moduleRoot is the directory of the root go.mod, two levels up.
func moduleRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil || !strings.HasPrefix(string(b), "module "+modulePath+"\n") {
		t.Fatalf("%s is not the module root: %v", root, err)
	}
	return root
}

// TestRecordWritersOnlyInPipeline is the D-3-6 gate: nothing outside the
// pipeline writes the record tables or imports the pipeline's internals.
func TestRecordWritersOnlyInPipeline(t *testing.T) {
	root := moduleRoot(t)
	found := scanDir(t, root, root, "")

	known := knownViolations()
	seen := make([]bool, len(known))
	for _, f := range found {
		i := slices.IndexFunc(known, func(k knownViolation) bool {
			return k.file == f.file && k.op == f.op && k.table == f.table
		})
		if i >= 0 && !seen[i] {
			seen[i] = true
			t.Logf("known violation, has a fix task: %s (%s)", f, known[i].why)
			continue
		}
		if f.op == "import" {
			t.Errorf("%s: only internal/server, internal/workflow and internal/onboard import internal/records/pipeline, and only the pipeline imports internal/records/store (D-3-6)", f)
			continue
		}
		t.Errorf("%s: only internal/records/pipeline and internal/records/store write the record tables; go through pipeline.Write (D-3-6)", f)
	}
	for i, k := range known {
		if !seen[i] {
			t.Errorf("known violation %s %s in %s is gone: remove it from knownViolations", k.op, k.table, k.file)
		}
	}
}

// wantRE is a fixture's expectation, `// want: <op> <table>` on the line.
var wantRE = regexp.MustCompile(`// want: (\w+) (\S+)`)

// TestRecordWriterFixture proves every planted write form in
// testdata/writers is reported, on its line, and nothing else is: the
// fixture's unmarked lines are look-alikes that must stay quiet.
func TestRecordWriterFixture(t *testing.T) {
	root := moduleRoot(t)
	dir := filepath.Join(root, "internal", "records", "testdata", "writers")
	got := scanDir(t, root, dir, "internal/records/testdata/writers")

	var want []finding
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".go") {
			return err
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		r, _ := filepath.Rel(dir, p)
		rel := path.Join("internal/records/testdata/writers", filepath.ToSlash(r))
		for i, line := range strings.Split(string(b), "\n") {
			for _, m := range wantRE.FindAllStringSubmatch(line, -1) {
				want = append(want, finding{rel, i + 1, m[1], m[2]})
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	want = dedupe(want)
	if len(want) < 12 {
		t.Fatalf("the fixture plants %d writes; it lost some", len(want))
	}
	for _, w := range want {
		if !slices.Contains(got, w) {
			t.Errorf("planted write not reported: %s", w)
		}
	}
	for _, g := range got {
		if !slices.Contains(want, g) {
			t.Errorf("look-alike reported: %s", g)
		}
	}

	// The same files inside the pipeline are allowed: the check is by path.
	if got := scanDir(t, root, dir, "internal/records/pipeline/fixture"); len(got) != 0 {
		t.Errorf("the pipeline may write the record tables, got %v", got)
	}
	// But not elsewhere under internal/records (S10's list code).
	if got := scanDir(t, root, dir, "internal/records/list"); len(got) != len(want) {
		t.Errorf("internal/records/list is not the pipeline: want %d findings, got %d", len(want), len(got))
	}
}

// TestPlantedOutboxWriteReported is the TICK-356 criterion: a planted UPDATE
// records in internal/outbox is reported, even beside the known
// lockRecordSQL violation.
func TestPlantedOutboxWriteReported(t *testing.T) {
	const src = `package outbox

const sneakySQL = "UPDATE records SET state = $2 WHERE id = $1"
`
	got := scanSource(t, "internal/outbox/planted.go", src)
	want := []finding{{"internal/outbox/planted.go", 3, "update", "records"}}
	if !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for _, k := range knownViolations() {
		if k.file == "internal/outbox/planted.go" {
			t.Fatal("the planted file must not match a known violation")
		}
	}
}

// TestPipelineImportRule plants the forbidden imports.
func TestPipelineImportRule(t *testing.T) {
	const src = "package x\n\nimport (\n\t_ \"" + modulePath + "/internal/records/pipeline\"\n\t_ \"" + modulePath + "/internal/records/store\"\n)\n"
	cases := []struct {
		file string
		want []finding
	}{
		{"internal/outbox/x.go", []finding{
			{"internal/outbox/x.go", 4, "import", "internal/records/pipeline"},
			{"internal/outbox/x.go", 5, "import", "internal/records/store"},
		}},
		{"internal/records/list/x.go", []finding{
			{"internal/records/list/x.go", 4, "import", "internal/records/pipeline"},
			{"internal/records/list/x.go", 5, "import", "internal/records/store"},
		}},
		{"internal/server/records/x.go", []finding{
			{"internal/server/records/x.go", 5, "import", "internal/records/store"},
		}},
		{"internal/workflow/x.go", []finding{
			{"internal/workflow/x.go", 5, "import", "internal/records/store"},
		}},
		{"internal/onboard/x.go", []finding{
			{"internal/onboard/x.go", 5, "import", "internal/records/store"},
		}},
		{"internal/records/pipeline/x.go", nil},
	}
	for _, c := range cases {
		got := scanSource(t, c.file, src)
		if !slices.Equal(got, dedupe(c.want)) {
			t.Errorf("%s: got %v, want %v", c.file, got, c.want)
		}
	}
}

func scanSource(t *testing.T, rel, src string) []finding {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, rel, src, parser.SkipObjectResolution)
	if err != nil {
		t.Fatal(err)
	}
	return scanFile(fset, rel, f, packageConsts([]*ast.File{f}))
}

func TestSQLWrites(t *testing.T) {
	cases := map[string][][2]string{
		"UPDATE records SET x = 1":                                              {{"update", "records"}},
		"update\n\t  RECORDS set x = 1":                                         {{"update", "records"}},
		`INSERT INTO "audit_log" (a) VALUES (1)`:                                {{"insert", "audit_log"}},
		`DELETE FROM public."comments" WHERE true`:                              {{"delete", "comments"}},
		"UPDATE ONLY public.record_links SET a = 1":                             {{"update", "record_links"}},
		"TRUNCATE outbox, records":                                              {{"truncate", "records"}},
		"truncate table only record_counters *, audit_log":                      {{"truncate", "record_counters"}, {"truncate", "audit_log"}},
		"COPY records (id) FROM STDIN":                                          {{"copy", "records"}},
		"MERGE INTO records r USING x ON true":                                  {{"merge", "records"}},
		"SELECT version FROM records WHERE id = $1 FOR UPDATE":                  nil,
		"INSERT INTO outbox (a) VALUES (1) ON CONFLICT (a) DO UPDATE SET a = 1": nil,
		"INSERT INTO records_archive (a) VALUES (1)":                            nil,
		"UPDATE recordsx SET a = 1":                                             nil,
		"-- UPDATE records\nSELECT 1":                                           nil,
		"SELECT 1 /* DELETE FROM comments */":                                   nil,
	}
	for in, want := range cases {
		if got := sqlWrites(in); !slices.Equal(got, want) {
			t.Errorf("sqlWrites(%q) = %v, want %v", in, got, want)
		}
	}
}

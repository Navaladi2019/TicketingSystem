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
//   - a use of a package-level string constant or variable holding such SQL
//     from a file other than the one declaring it, or of an exported one from
//     another package of the module, judged by the using file's path, so SQL
//     parked in an allowed file cannot be run from a forbidden one (TICK-581);
//   - a pgx.Identifier literal naming a record table (the CopyFrom form);
//   - an import of internal/records/pipeline outside internal/server,
//     internal/workflow, internal/onboard and cmd/demo-outbox (the S7 demo,
//     TICK-411), or of internal/records/store outside the pipeline.
//
// testdata/writers plants every form; TestRecordWriterFixture proves each one
// is reported, so the check cannot rot into a test that passes on nothing.
// testdata/sqlconsts does the same for the cross-file forms
// (TestSQLConstantUsedElsewhere).
//
// Known limit: the check has no dataflow. SQL that leaves an allowed file
// through a function result (`func q() string { return sealSQL }` called from
// a sibling) or is built at runtime is not seen. The database grants, RLS and
// the audit_log_guard_update trigger remain the control for those.
package records

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"maps"
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
	// audit_log is ADR-004's table; outside the pipeline only its one writer,
	// audit.Append, and the sealer's seal transition may write it
	// (auditWriter).
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

// auditWriter reports whether the file at rel may run a write of kind on
// table because it is ADR-004's one inserting writer (decision 1):
// internal/audit/append.go may INSERT INTO audit_log and nothing else. An
// UPDATE, DELETE, MERGE, TRUNCATE, COPY or pgx.Identifier (CopyFrom) there
// is still reported: those would skip validate, the secret check and the
// ::json binding, and this gate must not lean on the database grants
// (P5-12b, TICK-430).
//
// internal/audit/seal/seal.go may UPDATE audit_log and nothing else: it is the
// sealer's R11 seal transition (ADR-004 R11, R16; P5-34a, TICK-440), which the
// audit_log_guard_update trigger limits to nava_worker setting the four seal
// columns of an unsealed row. Any other write there is still reported.
//
// The seal.go allowance covers only an UPDATE whose SET list names nothing but
// the seal columns (sealColumnsOnly); the gate checks this itself rather than
// leaning on the trigger (TICK-581).
func auditWriter(rel, kind, table, sql string) bool {
	if table != "audit_log" {
		return false
	}
	return kind == "insert" && rel == "internal/audit/append.go" ||
		kind == "update" && rel == "internal/audit/seal/seal.go" && sealColumnsOnly(sql)
}

// sealColumn reports whether col is one of the four columns of the R11 seal
// transition (ADR-004 R11).
func sealColumn(col string) bool {
	switch col {
	case "seq", "prev_hash", "entry_hash", "sealed_at":
		return true
	}
	return false
}

var (
	// The head of an UPDATE of audit_log, up to and including SET:
	// UPDATE [ONLY] [schema.]audit_log [*] [[AS] alias] SET.
	auditUpdateRE = regexp.MustCompile(`\bupdate\s+(?:only\s+)?(?:` + ident + `\.)?audit_log\b(?:\s*\*)?(?:\s+(?:as\s+)?` + ident + `)?\s+set\b`)
	// A PostgreSQL dollar quote, $$ or $tag$: not read, so it fails closed.
	dollarQuoteRE = regexp.MustCompile(`\$[a-z_]*\$`)
)

// sealColumnsOnly reports whether every UPDATE of audit_log in sql sets only
// the seal columns (sealColumn). It fails closed: a row-form SET (a, b) =
// (...), a dollar-quoted string, an UPDATE it cannot find the SET of, or
// anything else it cannot read counts as setting other columns.
//
// Each SET list is read with string literals blanked and up to the first
// FROM, WHERE, RETURNING or semicolon outside parentheses, or the `)` closing
// an enclosing CTE, so a keyword inside a subquery or a string does not end it
// early (TICK-581 review).
func sealColumnsOnly(sql string) bool {
	s := scanSQL(normalizeSQL(sql), true)
	if dollarQuoteRE.MatchString(s) {
		return false
	}
	heads := auditUpdateRE.FindAllStringIndex(s, -1)
	updates := 0
	for _, w := range sqlWrites(s) {
		if w == [2]string{"update", "audit_log"} {
			updates++
		}
	}
	if len(heads) == 0 || len(heads) != updates {
		return false // an UPDATE of audit_log whose SET list this cannot find
	}
	for _, h := range heads {
		for _, a := range strings.Split(topLevelSetList(s[h[1]:]), ",") {
			col, _, ok := strings.Cut(a, "=")
			col = strings.TrimSpace(col)
			if i := strings.LastIndexByte(col, '.'); i >= 0 {
				col = col[i+1:] // a qualified column, a.seq
			}
			if !ok || !sealColumn(col) {
				return false
			}
		}
	}
	return true
}

// topLevelSetList returns the SET list at the start of s (normalized, string
// literals blanked) with every parenthesised group replaced by "_": up to the
// first FROM, WHERE, RETURNING or semicolon at depth 0, an unmatched ")" (the
// end of a CTE) or the end.
func topLevelSetList(s string) string {
	var b strings.Builder
	depth := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '(':
			if depth == 0 {
				b.WriteByte('_')
			}
			depth++
		case c == ')':
			if depth == 0 {
				return b.String()
			}
			depth--
		case depth > 0:
		case c == ';':
			return b.String()
		case identByte(c) && (i == 0 || !identByte(s[i-1])):
			j := i
			for j < len(s) && identByte(s[j]) {
				j++
			}
			switch word := s[i:j]; word {
			case "from", "where", "returning":
				return b.String()
			default:
				b.WriteString(word)
			}
			i = j - 1
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}

func identByte(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '_' || c == '$'
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

// knownViolations is empty: TICK-411 moved outbox.Enqueue and cmd/demo-outbox
// off direct record writes. Keep the mechanism for a future write that needs
// a fix task; the list can only shrink once it has entries.
func knownViolations() []knownViolation {
	return nil
}

// importAllowed reports whether a package at dir (slash-separated, relative
// to the module root) may import the module-relative package imp.
func importAllowed(dir, imp string) bool {
	switch imp {
	case "internal/records/pipeline":
		// cmd/demo-outbox is the S7 demo: it creates and updates its record
		// through Write like any other writer (TICK-411).
		return underAny(dir, "internal/records/pipeline", "internal/server", "internal/workflow", "internal/onboard",
			"cmd/demo-outbox")
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
	// A fmt verb, for substituting Sprintf arguments.
	verbRE = regexp.MustCompile(`%(?:\[\d+\])?[-+# 0]*\d*(?:\.\d+)?[a-zA-Z%]`)
)

// normalizeSQL lower-cases s and removes SQL comments and identifier quotes.
func normalizeSQL(s string) string {
	s = strings.ToLower(s)
	s = scanSQL(s, false)
	s = strings.ReplaceAll(s, `"`, "")
	return strings.ReplaceAll(s, "`", "")
}

// scanSQL replaces each SQL comment in s with a space, skipping comment
// markers inside single-quoted strings, so `'--'` cannot hide the rest of
// the line (TICK-581 review). With blank, it also replaces each
// single-quoted string with '_', so a keyword inside one is not read as SQL.
// Block comments nest, as in PostgreSQL. An unterminated string or comment
// runs to the end.
func scanSQL(s string, blank bool) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		switch {
		case s[i] == '\'':
			j := i + 1
			for j < len(s) {
				if s[j] == '\'' {
					if j+1 < len(s) && s[j+1] == '\'' { // '' is a quote inside the string
						j += 2
						continue
					}
					j++
					break
				}
				j++
			}
			if blank {
				b.WriteString("'_'")
			} else {
				b.WriteString(s[i:j])
			}
			i = j
		case strings.HasPrefix(s[i:], "--"):
			for i < len(s) && s[i] != '\n' {
				i++
			}
			b.WriteByte(' ')
		case strings.HasPrefix(s[i:], "/*"):
			depth := 0
			for i < len(s) {
				if strings.HasPrefix(s[i:], "/*") {
					depth++
					i += 2
				} else if strings.HasPrefix(s[i:], "*/") {
					depth--
					i += 2
					if depth == 0 {
						break
					}
				} else {
					i++
				}
			}
			b.WriteByte(' ')
		default:
			b.WriteByte(s[i])
			i++
		}
	}
	return b.String()
}

// sqlWrites returns the (op, table) pairs of every record-table write in s.
func sqlWrites(s string) [][2]string {
	s = normalizeSQL(s)
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

// pkgString is a package-level string constant or variable and the file
// declaring it.
type pkgString struct {
	value string
	decl  *ast.File
}

// packageStrings are a package's package-level string constants and
// variables, by name.
type packageStrings map[string]pkgString

// modulePackage is one package of the module, as another package sees it.
type modulePackage struct {
	name     string         // the package clause name
	exported packageStrings // its exported string constants and variables
}

// env resolves the names a string expression in one file may use: the
// package's strings, and the exported strings of the module packages the file
// imports, by local import name.
type env struct {
	pkg     packageStrings
	imports map[string]packageStrings
}

// scanFile reports the record-table writes and forbidden imports of one Go
// source file at rel (slash-separated, relative to the module root). pkg holds
// the string constants and variables of the file's package and mod the
// module's packages by module-relative directory (nil in single-file tests),
// so a write built from a named string is seen, and so is one run from this
// file with SQL declared in another file or package (TICK-581).
func scanFile(fset *token.FileSet, rel string, f *ast.File, pkg packageStrings, mod map[string]modulePackage) []finding {
	var out []finding
	dir := path.Dir(rel)
	pgxNames := map[string]bool{}
	en := env{pkg: pkg, imports: map[string]packageStrings{}}
	for _, is := range f.Imports {
		p, _ := strconv.Unquote(is.Path.Value)
		if p == "github.com/jackc/pgx/v5" {
			name := "pgx"
			if is.Name != nil {
				name = is.Name.Name
			}
			pgxNames[name] = true
		}
		imp, ok := strings.CutPrefix(p, modulePath+"/")
		if !ok {
			continue
		}
		if !importAllowed(dir, imp) {
			out = append(out, finding{rel, fset.Position(is.Pos()).Line, "import", imp})
		}
		if mp, ok := mod[imp]; ok && len(mp.exported) > 0 {
			name := mp.name
			if is.Name != nil {
				name = is.Name.Name
			}
			switch name {
			case "_":
			case ".": // a dot import: its names are used unqualified
				merged := maps.Clone(en.pkg)
				maps.Copy(merged, mp.exported)
				en.pkg = merged
			default:
				en.imports[name] = mp.exported
			}
		}
	}
	if writerAllowed(rel) || probeExempt(rel) {
		return out
	}

	report := func(n ast.Node, s string) {
		for _, w := range sqlWrites(s) {
			if !auditWriter(rel, w[0], w[1], s) {
				out = append(out, finding{rel, fset.Position(n.Pos()).Line, w[0], w[1]})
			}
		}
	}
	declared := declNames(f)
	var visit func(ast.Node) bool
	visit = func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.BasicLit:
			if s, ok := stringValue(n, en); ok {
				report(n, s)
			}
		case *ast.BinaryExpr:
			if s, ok := stringValue(n, en); ok {
				report(n, s)
			}
		case *ast.CallExpr:
			if s, ok := sprintfValue(n, en); ok {
				report(n, s)
			}
		case *ast.CompositeLit:
			if isPgxIdentifier(n.Type, pgxNames) && len(n.Elts) > 0 {
				if s, ok := stringValue(n.Elts[len(n.Elts)-1], en); ok && recordTable(strings.ToLower(s)) {
					out = append(out, finding{rel, fset.Position(n.Pos()).Line, "identifier", strings.ToLower(s)})
				}
			}
		case *ast.Ident:
			// A package string declared in another file: its SQL runs here,
			// so this file's path decides (TICK-581). One declared in this
			// file was already judged at its literal.
			if ps, ok := en.pkg[n.Name]; ok && ps.decl != f && !declared[n] {
				report(n, ps.value)
			}
		case *ast.SelectorExpr:
			// pkg.Name of another module package. Otherwise walk only the
			// operand: Sel is a field or method name, never a package string.
			if s, ok := stringValue(n, en); ok {
				report(n, s)
			}
			ast.Inspect(n.X, visit)
			return false
		}
		return true
	}
	ast.Inspect(f, visit)
	return dedupe(out)
}

// declNames returns the identifiers of f that name something rather than use
// it: struct fields and parameters, declared constants, variables, types and
// functions, `:=` targets, struct-literal keys and labels. A use of a
// local that shadows a package string is still reported: the gate errs on the
// side of a finding.
func declNames(f *ast.File) map[*ast.Ident]bool {
	out := map[*ast.Ident]bool{}
	mapTypes := map[string]bool{}
	ast.Inspect(f, func(n ast.Node) bool {
		if ts, ok := n.(*ast.TypeSpec); ok {
			if _, ok := ts.Type.(*ast.MapType); ok {
				mapTypes[ts.Name.Name] = true
			}
		}
		return true
	})
	add := func(ids ...*ast.Ident) {
		for _, id := range ids {
			if id != nil {
				out[id] = true
			}
		}
	}
	ast.Inspect(f, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.Field:
			add(n.Names...)
		case *ast.ValueSpec:
			add(n.Names...)
		case *ast.TypeSpec:
			add(n.Name)
		case *ast.FuncDecl:
			add(n.Name)
		case *ast.AssignStmt:
			if n.Tok == token.DEFINE {
				for _, l := range n.Lhs {
					if id, ok := l.(*ast.Ident); ok {
						add(id)
					}
				}
			}
		case *ast.CompositeLit:
			// A struct literal's key names a field; a map literal's key is a
			// use (PR #487 review). Without types a literal counts as a map
			// when its type is a map type or a name this file declares as
			// one; a map type from another package is the known miss.
			if isMapType(n.Type, mapTypes) {
				break
			}
			for _, e := range n.Elts {
				if kv, ok := e.(*ast.KeyValueExpr); ok {
					if id, ok := kv.Key.(*ast.Ident); ok {
						add(id)
					}
				}
			}
		case *ast.LabeledStmt:
			add(n.Label)
		case *ast.BranchStmt:
			add(n.Label)
		}
		return true
	})
	return out
}

// isMapType reports whether t is a map type or a name in mapTypes.
func isMapType(t ast.Expr, mapTypes map[string]bool) bool {
	switch t := t.(type) {
	case *ast.MapType:
		return true
	case *ast.Ident:
		return mapTypes[t.Name]
	}
	return false
}

// stringValue folds e to a constant string: a literal, a parenthesised or
// `+`-joined string, a package string constant or variable, or an exported
// one of an imported module package.
func stringValue(e ast.Expr, en env) (string, bool) {
	switch e := e.(type) {
	case *ast.BasicLit:
		if e.Kind != token.STRING {
			return "", false
		}
		s, err := strconv.Unquote(e.Value)
		return s, err == nil
	case *ast.ParenExpr:
		return stringValue(e.X, en)
	case *ast.Ident:
		ps, ok := en.pkg[e.Name]
		return ps.value, ok
	case *ast.SelectorExpr:
		x, ok := e.X.(*ast.Ident)
		if !ok {
			return "", false
		}
		ps, ok := en.imports[x.Name][e.Sel.Name]
		return ps.value, ok
	case *ast.BinaryExpr:
		if e.Op != token.ADD {
			return "", false
		}
		l, lok := stringValue(e.X, en)
		r, rok := stringValue(e.Y, en)
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
func sprintfValue(c *ast.CallExpr, en env) (string, bool) {
	sel, ok := c.Fun.(*ast.SelectorExpr)
	if !ok || !strings.HasSuffix(sel.Sel.Name, "printf") && sel.Sel.Name != "Errorf" {
		return "", false
	}
	for i, a := range c.Args {
		format, ok := stringValue(a, en)
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
			s, ok := stringValue(args[next], en)
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

// newPackageStrings returns the string constants and variables declared at
// package level in files (one package's non-test files). A variable counts at
// its initial value: the check is static, and SQL parked in a variable is
// still SQL (TICK-581).
func newPackageStrings(files []*ast.File) packageStrings {
	strs := packageStrings{}
	// Repeat so strings built from strings resolve, whatever the declaration
	// order; each pass resolves at least one more level, so ten is plenty.
	for pass, changed := 0, true; changed && pass < 10; pass++ {
		changed = false
		for _, f := range files {
			for _, d := range f.Decls {
				g, ok := d.(*ast.GenDecl)
				if !ok || g.Tok != token.CONST && g.Tok != token.VAR {
					continue
				}
				for _, sp := range g.Specs {
					vs := sp.(*ast.ValueSpec)
					for i, name := range vs.Names {
						if i >= len(vs.Values) || name.Name == "_" {
							continue
						}
						s, ok := stringValue(vs.Values[i], env{pkg: strs})
						if old, had := strs[name.Name]; ok && (!had || old.value != s) {
							strs[name.Name] = pkgString{s, f}
							changed = true
						}
					}
				}
			}
		}
	}
	return strs
}

// exported returns the exported subset of strs.
func (strs packageStrings) exported() packageStrings {
	out := packageStrings{}
	for name, ps := range strs {
		if ast.IsExported(name) {
			out[name] = ps
		}
	}
	return out
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
	type pkg struct {
		rels  []string
		files []*ast.File
		strs  packageStrings
	}
	fset := token.NewFileSet()
	pkgs := map[string]*pkg{} // by module-relative directory
	for _, paths := range byDir {
		p := &pkg{}
		for _, fp := range paths {
			f, err := parser.ParseFile(fset, fp, nil, parser.SkipObjectResolution)
			if err != nil {
				t.Fatalf("parse %s: %v", fp, err)
			}
			rel, err := filepath.Rel(root, fp)
			if err != nil {
				t.Fatal(err)
			}
			rel = filepath.ToSlash(rel)
			if prefix != "" {
				r, _ := filepath.Rel(start, fp)
				rel = path.Join(prefix, filepath.ToSlash(r))
			}
			p.files = append(p.files, f)
			p.rels = append(p.rels, rel)
		}
		p.strs = newPackageStrings(p.files)
		pkgs[path.Dir(p.rels[0])] = p
	}
	mod := map[string]modulePackage{}
	for dir, p := range pkgs {
		mod[dir] = modulePackage{p.files[0].Name.Name, p.strs.exported()}
	}
	var out []finding
	for _, p := range pkgs {
		for i, f := range p.files {
			out = append(out, scanFile(fset, p.rels[i], f, p.strs, mod)...)
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
			t.Errorf("%s: only internal/server, internal/workflow, internal/onboard and cmd/demo-outbox import internal/records/pipeline, and only the pipeline imports internal/records/store (D-3-6)", f)
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
	want := plantedWrites(t, dir, "internal/records/testdata/writers")
	if len(want) < 12 {
		t.Fatalf("the fixture plants %d writes; it lost some", len(want))
	}
	compareFindings(t, got, want)

	// The same files inside the pipeline are allowed: the check is by path.
	if got := scanDir(t, root, dir, "internal/records/pipeline/fixture"); len(got) != 0 {
		t.Errorf("the pipeline may write the record tables, got %v", got)
	}
	// But not elsewhere under internal/records (S10's list code).
	if got := scanDir(t, root, dir, "internal/records/list"); len(got) != len(want) {
		t.Errorf("internal/records/list is not the pipeline: want %d findings, got %d", len(want), len(got))
	}
}

// plantedWrites returns the `// want:` lines of the Go files under dir, with
// paths as if dir were prefix.
func plantedWrites(t *testing.T, dir, prefix string) []finding {
	t.Helper()
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
		rel := path.Join(prefix, filepath.ToSlash(r))
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
	return dedupe(want)
}

// compareFindings reports a planted write that was missed and a finding
// nothing planted.
func compareFindings(t *testing.T, got, want []finding) {
	t.Helper()
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
}

// TestSQLConstantUsedElsewhere is the TICK-581 criterion: SQL parked in a
// package constant or variable of an allowed file (here the fixture's
// internal/audit/seal/seal.go) is reported where another file of the package,
// or another package through an exported name, runs it. seal.go's own use and
// its seal-column UPDATEs stay allowed; its UPDATEs of other columns do not.
func TestSQLConstantUsedElsewhere(t *testing.T) {
	root := moduleRoot(t)
	dir := filepath.Join(root, "internal", "records", "testdata", "sqlconsts")
	got := scanDir(t, root, dir, "internal")
	want := plantedWrites(t, dir, "internal")
	if len(want) < 6 {
		t.Fatalf("the fixture plants %d writes; it lost some", len(want))
	}
	compareFindings(t, got, want)

	// The two-file case exactly as the issue states it.
	files := map[string]string{
		"internal/audit/seal/seal.go":  "package seal\n\nconst rewriteSQL = `UPDATE audit_log SET seq = $2 WHERE id = $1`\n",
		"internal/audit/seal/other.go": "package seal\n\nfunc run(exec func(string)) {\n\texec(rewriteSQL)\n}\n",
	}
	want = []finding{{"internal/audit/seal/other.go", 4, "update", "audit_log"}}
	if got := scanSources(t, files); !slices.Equal(got, want) {
		t.Errorf("two-file case: got %v, want %v", got, want)
	}
}

// TestSealColumnsOnly: the seal.go allowance reads the SET list.
func TestSealColumnsOnly(t *testing.T) {
	cases := map[string]bool{
		"UPDATE audit_log SET seq = $2 WHERE id = $1":                                          true,
		"update public.audit_log as a set a.seq = u.seq, sealed_at = now() from u where true":  true,
		"UPDATE audit_log SET prev_hash = $2, entry_hash = coalesce($3, $4, $5) WHERE id = $1": true,
		"UPDATE audit_log SET entry_hash = $2;":                                                true,
		"UPDATE audit_log SET diff = NULL WHERE id = $1":                                       false,
		"UPDATE audit_log SET seq = $2, diff_salt = NULL WHERE id = $1":                        false,
		"UPDATE audit_log SET (seq, sealed_at) = ($2, now()) WHERE id = $1":                    false,
		"UPDATE audit_log SET seq = 1; UPDATE audit_log SET diff = NULL":                       false,
		"INSERT INTO audit_log (seq) VALUES (1)":                                               false,

		// PR #487 review: the SET list must not end at a keyword inside a
		// subquery or a string literal, and every UPDATE must be read.
		"UPDATE audit_log SET seq = (SELECT max(seq) FROM audit_log), diff = NULL WHERE id = $1":     false,
		"UPDATE audit_log SET sealed_at = 'x where y', diff = NULL WHERE id = $1":                    false,
		"UPDATE audit_log SET seq = (SELECT 1 WHERE true), diff = NULL":                              false,
		"UPDATE audit_log SET seq = 'where', diff = NULL WHERE id = $1":                              false,
		"UPDATE audit_log SET seq = 1; UPDATE audit_log* SET diff = NULL":                            false,
		"UPDATE audit_log SET seq = '--', diff = NULL WHERE id = $1":                                 false,
		"UPDATE audit_log SET seq = 'it''s', diff = NULL WHERE id = $1":                              false,
		"UPDATE audit_log SET seq = $$x$$, diff = NULL WHERE id = $1":                                false,
		"UPDATE audit_log SET seq = 1 RETURNING (SELECT diff FROM audit_log)":                        true,
		"UPDATE audit_log SET seq = (SELECT max(seq) FROM audit_log) WHERE id = $1":                  true,
		"UPDATE audit_log SET sealed_at = 'x where y' WHERE id = $1":                                 true,
		"WITH s AS (UPDATE audit_log SET seq = 1 WHERE id = $1) SELECT 1":                            true,
		"WITH s AS (UPDATE audit_log SET seq = 1), t AS (UPDATE audit_log SET diff = NULL) SELECT 1": false,
	}
	for in, want := range cases {
		if got := sealColumnsOnly(in); got != want {
			t.Errorf("sealColumnsOnly(%q) = %v, want %v", in, got, want)
		}
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

// TestAuditAppendIsTheAuditLogWriter: internal/audit/append.go may INSERT
// INTO audit_log (ADR-004 decision 1) and nothing else: no other write on
// audit_log, no pgx.Identifier (CopyFrom) form, no other table. Any other
// file under internal/audit may not write audit_log at all.
func TestAuditAppendIsTheAuditLogWriter(t *testing.T) {
	const src = `package audit

import "github.com/jackc/pgx/v5"

const insertSQL = "INSERT INTO audit_log (tenant_id) VALUES ($1)"

const sneakySQL = "UPDATE records SET state = $2 WHERE id = $1"

const updateSQL = "UPDATE audit_log SET diff = NULL WHERE id = $1"

const deleteSQL = "DELETE FROM audit_log WHERE id = $1"

var copyTable = pgx.Identifier{"audit_log"}
`
	if got, want := scanSource(t, "internal/audit/append.go", src), dedupe([]finding{
		{"internal/audit/append.go", 7, "update", "records"},
		{"internal/audit/append.go", 9, "update", "audit_log"},
		{"internal/audit/append.go", 11, "delete", "audit_log"},
		{"internal/audit/append.go", 13, "identifier", "audit_log"},
	}); !slices.Equal(got, want) {
		t.Errorf("append.go: got %v, want %v", got, want)
	}
	if got, want := scanSource(t, "internal/audit/other.go", src), dedupe([]finding{
		{"internal/audit/other.go", 5, "insert", "audit_log"},
		{"internal/audit/other.go", 7, "update", "records"},
		{"internal/audit/other.go", 9, "update", "audit_log"},
		{"internal/audit/other.go", 11, "delete", "audit_log"},
		{"internal/audit/other.go", 13, "identifier", "audit_log"},
	}); !slices.Equal(got, want) {
		t.Errorf("other.go: got %v, want %v", got, want)
	}
}

// TestSealerIsTheAuditLogSealer: internal/audit/seal/seal.go may UPDATE
// audit_log (the R11 seal transition) and nothing else: no INSERT or DELETE
// on audit_log, no pgx.Identifier (CopyFrom) form, no other table. Another
// file of the package may not write audit_log at all.
func TestSealerIsTheAuditLogSealer(t *testing.T) {
	const src = `package seal

import "github.com/jackc/pgx/v5"

const sealSQL = "UPDATE audit_log SET seq = $2 WHERE id = $1"

const sneakySQL = "UPDATE records SET state = $2 WHERE id = $1"

const insertSQL = "INSERT INTO audit_log (tenant_id) VALUES ($1)"

const deleteSQL = "DELETE FROM audit_log WHERE id = $1"

var copyTable = pgx.Identifier{"audit_log"}

const scrubSQL = "UPDATE audit_log SET diff = NULL WHERE id = $1"
`
	if got, want := scanSource(t, "internal/audit/seal/seal.go", src), dedupe([]finding{
		{"internal/audit/seal/seal.go", 7, "update", "records"},
		{"internal/audit/seal/seal.go", 9, "insert", "audit_log"},
		{"internal/audit/seal/seal.go", 11, "delete", "audit_log"},
		{"internal/audit/seal/seal.go", 13, "identifier", "audit_log"},
		{"internal/audit/seal/seal.go", 15, "update", "audit_log"}, // not a seal column (TICK-581)
	}); !slices.Equal(got, want) {
		t.Errorf("seal.go: got %v, want %v", got, want)
	}
	if got, want := scanSource(t, "internal/audit/seal/activities.go", src), dedupe([]finding{
		{"internal/audit/seal/activities.go", 5, "update", "audit_log"},
		{"internal/audit/seal/activities.go", 7, "update", "records"},
		{"internal/audit/seal/activities.go", 9, "insert", "audit_log"},
		{"internal/audit/seal/activities.go", 11, "delete", "audit_log"},
		{"internal/audit/seal/activities.go", 13, "identifier", "audit_log"},
		{"internal/audit/seal/activities.go", 15, "update", "audit_log"},
	}); !slices.Equal(got, want) {
		t.Errorf("activities.go: got %v, want %v", got, want)
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
	return scanFile(fset, rel, f, newPackageStrings([]*ast.File{f}), nil)
}

// scanSources scans one package made of files (rel path to source).
func scanSources(t *testing.T, files map[string]string) []finding {
	t.Helper()
	fset := token.NewFileSet()
	rels := slices.Sorted(maps.Keys(files))
	parsed := make([]*ast.File, len(rels))
	for i, rel := range rels {
		f, err := parser.ParseFile(fset, rel, files[rel], parser.SkipObjectResolution)
		if err != nil {
			t.Fatal(err)
		}
		parsed[i] = f
	}
	strs := newPackageStrings(parsed)
	var out []finding
	for i, f := range parsed {
		out = append(out, scanFile(fset, rels[i], f, strs, nil)...)
	}
	return dedupe(out)
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
		"SELECT 1 /* a /* nested */ DELETE FROM comments */":                    nil,
		"SELECT '--'; UPDATE records SET x = 1":                                 {{"update", "records"}},
		"SELECT '/*'; DELETE FROM comments; SELECT '*/'":                        {{"delete", "comments"}},
		"SELECT 'it''s -- not a comment'; UPDATE records SET x = 1":             {{"update", "records"}},
	}
	for in, want := range cases {
		if got := sqlWrites(in); !slices.Equal(got, want) {
			t.Errorf("sqlWrites(%q) = %v, want %v", in, got, want)
		}
	}
}

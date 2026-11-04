// The architecture test for the audit diff read path (ADR-004 R7, TICK-541):
// only internal/audit/read reads audit_log.diff or audit_log.diff_salt. That
// package is the only place the R7 masker runs. Since ADR-004-A1 (TICK-558)
// Postgres is the gate: only nava_audit_reader, whose pool only that package
// holds, may SELECT either column, so a query anywhere else gets 42501 at
// Execute. This check stays as layer 2 (A1-10 rule 2), an early warning.
//
// The check is static and runs in `task test`. It parses every non-test Go
// file in the repository (every module, build tags ignored, so a tagged file
// cannot hide a read) and reports a string, constant, `+` concatenation or
// fmt.Sprintf result whose SQL reads audit_log (FROM or JOIN, a write with
// RETURNING, or COPY ... TO) and names diff or diff_salt, selects a star,
// uses the whole row (to_jsonb(a), json_agg(a), array_agg(a), a::text,
// SELECT audit_log FROM audit_log, SELECT a FROM audit_log a, ...), COPYs
// the table out without a column list, or runs TABLE audit_log. Matching
// is case-insensitive, with any whitespace, quotes or schema prefix; SQL
// comments are ignored. diff_sha256, diff_truncated and
// diff_redacted are other columns and do not match.
//
// The table is seen after FROM, JOIN or a comma in a FROM list (FROM
// records r, audit_log a), also inside a parenthesised join and with spaces
// around a schema dot (public . audit_log), after ONLY with or without
// parentheses (FROM ONLY (audit_log) a) and before a descendant star (FROM
// audit_log * a, TICK-559); (a).* and a . * are stars. TABLE audit_log fails closed:
// it counts after any word but the DDL and ACL ones in tableDDL, so INSERT
// ... OVERRIDING SYSTEM VALUE TABLE and DECLARE c CURSOR FOR TABLE are seen.
//
// Constants fold across packages: `"SELECT diff FROM " + audited.LogTable`
// and `fmt.Sprintf("... FROM %s", audited.LogTable)` are seen, through a
// named import (aud.LogTable) or a dot import (LogTable). Imports are scoped
// per file, so two files that give the same local name to different packages
// do not shadow each other. A const declared inside a function folds too.
//
// Known limits (static, one string at a time; review owns these, and the
// database-layer fix is RFC TICK-549):
//   - Only constants fold. A part held in a package or local var, a function
//     result or a parameter becomes "?", so `"SELECT " + colsVar + " FROM
//     audit_log"` with diff in colsVar is not seen. Keep column lists in
//     consts.
//   - A query assembled across calls (strings.Builder.WriteString("SELECT
//     diff_salt") then WriteString(" FROM audit_log"), a slice joined at run
//     time) is never one string, so it is not seen.
//   - A constant built from another package's constant is two hops and is
//     not resolved.
//   - A whole row used outside the listed row functions, casts and bare
//     select-list items is not seen: SELECT (a), a AS z, CAST(a AS text),
//     jsonb_build_object('r', a), ARRAY[a] (Security D on PR #458).
//   - An implicitly repeated const (const ( x = "audit_log"; y )) does not
//     fold, and fmt verbs with an explicit index (%[2]s) are filled in
//     order, not by index (Security E on PR #458).
//   - D and E are layer-2 gaps: Postgres is the gate since TICK-558 revoked
//     nava_app on diff. Revisit if one shows up in a review.
//   - A function's local consts apply to its whole body, not per block, and a
//     local var that shadows a const of the same name still folds as the
//     const.
//
// testdata/diffreaders plants every form; TestAuditDiffReaderFixture proves
// each one is reported, so the check cannot rot into a test that passes on
// nothing. The string folding mirrors internal/records/arch_test.go.
package read_test

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

// diffReaderAllowed reports whether the file at rel (slash-separated,
// relative to the repository root) may read audit_log.diff or diff_salt:
// only the reader package, which masks every diff it returns, and the
// ADR-004-A1 isolation probes, each of which must be refused with 42501 or
// runs as nava_audit_reader (TICK-558).
func diffReaderAllowed(rel string) bool {
	return rel == "internal/audit/read" || strings.HasPrefix(rel, "internal/audit/read/") ||
		rel == "internal/tenancy/isolation/audit_reader_paths.go"
}

// diffRead is one reported read.
type diffRead struct {
	file string
	line int
	what string // diff, diff_salt, star, row, copy or table
}

func (d diffRead) String() string {
	return fmt.Sprintf("%s:%d: reads audit_log %s", d.file, d.line, d.what)
}

const (
	sqlIdent = `[a-z_][a-z0-9_$]*`
	// An optional schema prefix, with any whitespace around the dot
	// (public . audit_log).
	schemaPrefix = `(?:` + sqlIdent + `\s*\.\s*)?`
	// audit_log or one of its partitions (audit_log_default) as a read
	// source; audit_log_v0 and other look-alike names are not it.
	auditTable = schemaPrefix + `audit_log(?:_default|_p` + sqlIdent + `)?\b`
	// FROM or JOIN, then any opening parentheses of a parenthesised join
	// (FROM (audit_log a CROSS JOIN x)).
	fromStart = `\b(?:from|join)(?:\s*\(\s*|\s+)+`
	// A FROM-list comma, then any opening parentheses.
	listStart = `,\s*(?:\(\s*)*`
	// ONLY before the table, with or without a parenthesis: FROM ONLY
	// audit_log, FROM ONLY (audit_log) (TICK-559).
	onlyPrefix = `(?:only(?:\s+|\s*\(\s*))?`
	// A reference to audit_log whose row name is group 1 or group 2: ONLY
	// (audit_log), [ONLY] audit_log, or audit_log * (the descendant star,
	// which Postgres 16 accepts before an alias: FROM audit_log * a).
	auditRef = `(?:only\s*\(\s*` + schemaPrefix + `(audit_log(?:_default|_p` + sqlIdent + `)?)\b\s*\)|(?:only\s+)?` +
		schemaPrefix + `(audit_log(?:_default|_p` + sqlIdent + `)?)\b(?:\s*\*)?)`
)

var (
	fromAudit = regexp.MustCompile(fromStart + onlyPrefix + auditTable)
	// audit_log as a later item of a FROM list: FROM records r, audit_log a.
	fromList = regexp.MustCompile(`\bfrom\s[^;]*?` + listStart + onlyPrefix + auditTable)
	// The table's row names in a statement: audit_log itself and the alias
	// after FROM, JOIN or a FROM-list comma: audit_log [AS] alias, also after
	// ONLY (audit_log) and audit_log * (groups 1 or 2, then 3). A keyword is
	// not an alias.
	aliasRE = regexp.MustCompile(`(?:` + fromStart + `|` + listStart + `)` + auditRef + `(?:\s+(?:as\s+)?(` + sqlIdent + `))?`)
	// A write on audit_log that returns columns reads them too.
	writeAudit = regexp.MustCompile(`\b(?:insert\s+into|update|delete\s+from|merge\s+into)\s+(?:only\s+)?` + auditTable)
	returning  = regexp.MustCompile(`\breturning\b`)
	// COPY audit_log [(cols)] TO: a bare one exports every column. COPY
	// audit_log FROM is a write; COPY (SELECT ...) TO is seen by its SELECT.
	copyTo = regexp.MustCompile(`\bcopy\s+` + auditTable + `\s*(\([^)]*\))?\s*to\b`)
	// TABLE audit_log is SELECT * FROM audit_log. Group 1 is the word before
	// TABLE, if any; tableRead decides on it.
	tableCmd = regexp.MustCompile(`(?:([a-z0-9_$]+)\s+)?\btable\s+` + onlyPrefix + auditTable)
	// The protected columns. \b stops at diff_sha256 and friends, because _
	// is a word character.
	diffCol = regexp.MustCompile(`\b(diff_salt|diff)\b`)
	// SELECT *, alias.*, alias . *, (alias).*, RETURNING * (count(*) is not
	// a star select).
	starSel = regexp.MustCompile(`(?:\bselect\s+(?:distinct\s+)?|\breturning\s+|(?:` + sqlIdent + `|\))\s*\.\s*)\*`)
)

// tableDDL reports whether w may stand before TABLE audit_log without
// reading it: DDL (CREATE, ALTER, DROP, TRUNCATE, LOCK, LIKE, REFERENCES)
// and GRANT, REVOKE or COMMENT ... ON TABLE.
func tableDDL(w string) bool {
	switch w {
	case "create", "alter", "lock", "drop", "truncate", "on", "like", "references":
		return true
	}
	return false
}

// tableRead reports whether s runs TABLE audit_log as a query. It fails
// closed: every TABLE audit_log counts unless the word before it is in
// tableDDL, so a prefix nobody listed (INSERT ... OVERRIDING SYSTEM VALUE,
// DECLARE c CURSOR FOR, WITH x AS MATERIALIZED (...)) is still seen. The
// cost is over-reporting a DDL form that is not listed, such as CREATE TEMP
// TABLE audit_log; add its word here when one turns up.
func tableRead(s string) bool {
	for _, m := range tableCmd.FindAllStringSubmatch(s, -1) {
		if !tableDDL(m[1]) {
			return true
		}
	}
	return false
}

// sqlKeyword reports whether w may follow FROM audit_log without being an
// alias.
func sqlKeyword(w string) bool {
	switch w {
	case "where", "join", "inner", "left", "right", "full", "cross", "natural", "on", "using", "order",
		"group", "having", "limit", "offset", "for", "union", "intersect", "except", "window",
		"returning", "fetch", "tablesample", "set", "from", "and", "or":
		return true
	}
	return false
}

// wholeRow reports whether s uses a whole audit_log row, which carries diff
// and diff_salt, through one of its row names: as an argument of a row
// function or aggregate (to_jsonb(a), row_to_json(a), json_agg(a),
// jsonb_agg(a), array_agg(a), ...), in a cast (a::text), or as a bare
// select-list item (SELECT audit_log FROM audit_log, SELECT a.id, a FROM
// audit_log a). A column of the row (a.id, a.id::text) is not the row.
func wholeRow(s string) bool {
	names := map[string]bool{}
	for _, m := range aliasRE.FindAllStringSubmatch(s, -1) {
		names[m[1]+m[2]] = true // exactly one of the two is set
		if m[3] != "" && !sqlKeyword(m[3]) {
			names[m[3]] = true
		}
	}
	for n := range names {
		q := regexp.QuoteMeta(n)
		for _, re := range []string{
			// A row function or aggregate over the row.
			`\b(?:to_jsonb?|row_to_json|json_agg|jsonb_agg|array_agg|string_agg|jsonb?_populate_record|hstore|json_object_agg|jsonb_object_agg|row)\s*\(\s*(?:distinct\s+)?` + q + `\s*(?:\)|,|\border\b)`,
			// A cast of the row; a.col::text has a dot before the cast.
			`(?:^|[^.a-z0-9_$])` + q + `\s*::`,
			// A bare select-list item.
			`(?:\bselect\s+(?:distinct\s+)?|,\s*)` + q + `\s*(?:,|\bfrom\b)`,
		} {
			if regexp.MustCompile(re).MatchString(s) {
				return true
			}
		}
	}
	return false
}

var (
	lineCommentRE  = regexp.MustCompile(`--[^\n]*`)
	blockCommentRE = regexp.MustCompile(`(?s)/\*.*?\*/`)
	verbRE         = regexp.MustCompile(`%(?:\[\d+\])?[-+# 0]*\d*(?:\.\d+)?[a-zA-Z%]`)
)

// diffReads returns what s reads of audit_log's protected columns, or nil
// when s does not read audit_log or reads only other columns.
func diffReads(s string) []string {
	s = strings.ToLower(s)
	s = blockCommentRE.ReplaceAllString(s, " ")
	s = lineCommentRE.ReplaceAllString(s, " ")
	s = strings.ReplaceAll(s, `"`, "")
	s = strings.ReplaceAll(s, "`", "")
	copied := copyTo.FindStringSubmatch(s)
	table := tableRead(s)
	if reads := fromAudit.MatchString(s) || fromList.MatchString(s) || writeAudit.MatchString(s) && returning.MatchString(s) || copied != nil || table; !reads {
		return nil
	}
	var out []string
	if table {
		out = append(out, "table")
	}
	for _, m := range diffCol.FindAllStringSubmatch(s, -1) {
		out = append(out, m[1])
	}
	if starSel.MatchString(s) {
		out = append(out, "star")
	}
	if wholeRow(s) {
		out = append(out, "row")
	}
	if copied != nil && copied[1] == "" {
		out = append(out, "copy")
	}
	slices.Sort(out)
	return slices.Compact(out)
}

// scanFile reports the protected-column reads in one Go file at rel. consts
// holds the string constants the file sees (its package's and its own
// imports'), so a query built from a named constant is seen too; a function
// body adds its local consts on top.
func scanFile(fset *token.FileSet, rel string, f *ast.File, consts map[string]string) []diffRead {
	if diffReaderAllowed(rel) {
		return nil
	}
	var out []diffRead
	report := func(n ast.Node, s string) {
		for _, w := range diffReads(s) {
			out = append(out, diffRead{rel, fset.Position(n.Pos()).Line, w})
		}
	}
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
				if s, ok := stringValue(n.(ast.Expr), consts); ok {
					report(n, s)
				}
			case *ast.CallExpr:
				if s, ok := sprintfValue(n, consts); ok {
					report(n, s)
				}
			}
			return true
		})
	}
	walk(f, consts)
	return dedupe(out)
}

// localConsts returns outer plus the string consts declared in body, not
// counting nested function literals, which add their own.
func localConsts(body *ast.BlockStmt, outer map[string]string) map[string]string {
	env := maps.Clone(outer)
	if env == nil {
		env = map[string]string{}
	}
	for range 2 { // a second pass resolves constants built from constants
		ast.Inspect(body, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.FuncLit:
				return false
			case *ast.DeclStmt:
				if g, ok := n.Decl.(*ast.GenDecl); ok {
					foldConsts(g, env, nil)
				}
			}
			return true
		})
	}
	return env
}

// stringValue folds e to a constant string: a literal, a parenthesised or
// `+`-joined string, a package string constant, or another package's
// (audited.LogTable). An unknown operand of a
// `+` becomes "?", so `"SELECT " + cols + " FROM audit_log"` is still seen.
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
	case *ast.SelectorExpr:
		// pkg.Const, resolved through the file's imports (importScope).
		if x, ok := e.X.(*ast.Ident); ok {
			s, ok := consts[x.Name+"."+e.Sel.Name]
			return s, ok
		}
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

// sprintfValue renders a fmt.Sprintf-like call with a constant format,
// substituting every constant string argument and "?" for the rest.
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

func dedupe(ds []diffRead) []diffRead {
	sort.Slice(ds, func(i, j int) bool { return ds[i].String() < ds[j].String() })
	return slices.Compact(ds)
}

// packageConsts returns the string constants declared at package level in
// files (one directory's non-test files). Each file's declarations fold
// against scopes[i], the constants its own imports bring in (importScope);
// scopes may be nil.
func packageConsts(files []*ast.File, scopes []map[string]string) map[string]string {
	pkg := map[string]string{}
	for range 2 { // a second pass resolves constants built from constants
		for i, f := range files {
			var scope map[string]string
			if i < len(scopes) {
				scope = scopes[i]
			}
			env := fileConsts(pkg, scope)
			for _, d := range f.Decls {
				if g, ok := d.(*ast.GenDecl); ok {
					foldConsts(g, env, pkg)
				}
			}
		}
	}
	return pkg
}

// fileConsts is what one file sees: its imports' constants and its
// package's. Go rejects a dot-imported name that clashes with a package-level
// name, so the order does not matter.
func fileConsts(pkg, scope map[string]string) map[string]string {
	env := make(map[string]string, len(pkg)+len(scope))
	maps.Copy(env, scope)
	maps.Copy(env, pkg)
	return env
}

// foldConsts adds the string constants of a const declaration to env, and
// to out when it is not nil.
func foldConsts(g *ast.GenDecl, env, out map[string]string) {
	if g.Tok != token.CONST {
		return
	}
	for _, sp := range g.Specs {
		vs := sp.(*ast.ValueSpec)
		for i, name := range vs.Names {
			if i >= len(vs.Values) {
				continue
			}
			if s, ok := stringValue(vs.Values[i], env); ok {
				env[name.Name] = s
				if out != nil {
					out[name.Name] = s
				}
			}
		}
	}
}

// goDirs returns the non-test .go files under start, by directory, skipping
// testdata, vendor and hidden directories. Nested modules such as
// tools/navalint are included.
func goDirs(t *testing.T, start string) map[string][]string {
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
		if strings.HasSuffix(p, ".go") && !strings.HasSuffix(p, "_test.go") {
			byDir[filepath.Dir(p)] = append(byDir[filepath.Dir(p)], p)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return byDir
}

func parseDir(t *testing.T, fset *token.FileSet, paths []string) []*ast.File {
	t.Helper()
	files := make([]*ast.File, 0, len(paths))
	for _, p := range paths {
		f, err := parser.ParseFile(fset, p, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("parse %s: %v", p, err)
		}
		files = append(files, f)
	}
	return files
}

// pkgConsts is one package's name and string constants.
type pkgConsts struct {
	name   string
	consts map[string]string
}

// moduleAt returns the module path and directory of the go.mod that owns
// dir, looking upward no further than root; ok is false when there is none.
func moduleAt(root, dir string, cache map[string][2]string) (mod, modDir string, ok bool) {
	if m, hit := cache[dir]; hit {
		return m[0], m[1], m[0] != ""
	}
	var m [2]string
	if b, err := os.ReadFile(filepath.Join(dir, "go.mod")); err == nil {
		for _, line := range strings.Split(strings.ReplaceAll(string(b), "\r\n", "\n"), "\n") {
			if mp, found := strings.CutPrefix(strings.TrimSpace(line), "module "); found {
				m = [2]string{strings.Trim(strings.TrimSpace(mp), `"`), dir}
				break
			}
		}
	} else if parent := filepath.Dir(dir); parent != dir && dir != root {
		pm, pd, _ := moduleAt(root, parent, cache)
		m = [2]string{pm, pd}
	}
	cache[dir] = m
	return m[0], m[1], m[0] != ""
}

// constIndex maps the import path of every package in the repository (every
// module) to its string constants, so `audited.LogTable` in another package
// folds to "audit_log" (Security finding 1 on PR #443).
func constIndex(t *testing.T, root string) map[string]pkgConsts {
	t.Helper()
	idx := indexDir(t, root, root)
	if idx[modulePath+"/internal/db/audited"].consts["LogTable"] != "audit_log" {
		t.Fatal("the const index does not resolve audited.LogTable; the guard would miss it")
	}
	return idx
}

// indexDir maps the import path of every package under start to its string
// constants.
func indexDir(t *testing.T, root, start string) map[string]pkgConsts {
	t.Helper()
	cache := map[string][2]string{}
	idx := map[string]pkgConsts{}
	fset := token.NewFileSet()
	for dir, paths := range goDirs(t, start) {
		mod, modDir, ok := moduleAt(root, dir, cache)
		if !ok {
			continue
		}
		rel, err := filepath.Rel(modDir, dir)
		if err != nil {
			t.Fatal(err)
		}
		files := parseDir(t, fset, paths)
		idx[path.Join(mod, filepath.ToSlash(rel))] = pkgConsts{files[0].Name.Name, packageConsts(files, nil)}
	}
	return idx
}

// importScope returns the constants that one file's imports of repository
// packages bring in: qualified by the import's local name
// ("audited.LogTable", "aud.LogTable"), or bare for a dot import. It is per
// file, because two files of a package may give one local name to two
// different packages (PSE finding on TICK-541).
func importScope(f *ast.File, idx map[string]pkgConsts) map[string]string {
	scope := map[string]string{}
	for _, is := range f.Imports {
		p, _ := strconv.Unquote(is.Path.Value)
		pc, ok := idx[p]
		if !ok {
			continue
		}
		local := pc.name
		if is.Name != nil {
			local = is.Name.Name
		}
		if local == "_" {
			continue
		}
		for k, v := range pc.consts {
			if local == "." {
				scope[k] = v
			} else {
				scope[local+"."+k] = v
			}
		}
	}
	return scope
}

// scanDir parses every non-test .go file under start and returns the
// findings with paths relative to root, or as if the files lived at prefix
// when prefix is not empty. idx resolves other packages' constants.
func scanDir(t *testing.T, root, start, prefix string, idx map[string]pkgConsts) []diffRead {
	t.Helper()
	var out []diffRead
	fset := token.NewFileSet()
	for _, paths := range goDirs(t, start) {
		files := parseDir(t, fset, paths)
		scopes := make([]map[string]string, len(files))
		for i, f := range files {
			scopes[i] = importScope(f, idx)
		}
		pkg := packageConsts(files, scopes)
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
			out = append(out, scanFile(fset, rel, f, fileConsts(pkg, scopes[i]))...)
		}
	}
	return dedupe(out)
}

// repoRoot is the directory of the root go.mod, three levels up.
func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil || !strings.HasPrefix(strings.ReplaceAll(string(b), "\r\n", "\n"), "module "+modulePath+"\n") {
		t.Fatalf("%s is not the module root: %v", root, err)
	}
	return root
}

// TestAuditDiffReadOnlyInReader is the TICK-541 gate: no non-test Go file
// outside internal/audit/read reads audit_log.diff or diff_salt.
func TestAuditDiffReadOnlyInReader(t *testing.T) {
	root := repoRoot(t)
	for _, f := range scanDir(t, root, root, "", constIndex(t, root)) {
		t.Errorf("%s: only internal/audit/read may read audit_log.diff or diff_salt, because only it applies the R7 masker; go through read.Reader (ADR-004 R7, TICK-541). Since ADR-004-A1 such a query gets 42501 at Execute rather than leaking the diff, because only nava_audit_reader may read those columns; this check is the early warning (A1-10 rule 2)", f)
	}
}

// wantRE is a fixture's expectation, `// want: <what>` on the line; a line
// may carry several.
var wantRE = regexp.MustCompile(`// want: ([\w ]+)`)

// TestAuditDiffReaderFixture is the positive control: every planted read in
// testdata/diffreaders is reported on its line, and nothing else is. The
// fixture's unmarked lines are look-alikes that must stay quiet.
func TestAuditDiffReaderFixture(t *testing.T) {
	root := repoRoot(t)
	dir := filepath.Join(root, "internal", "audit", "read", "testdata", "diffreaders")
	const at = "internal/server/planted"
	// The fixture's own packages (diffreaders/shadow) resolve like any
	// repository package.
	idx := constIndex(t, root)
	maps.Copy(idx, indexDir(t, root, dir))
	got := scanDir(t, root, dir, at, idx)

	var want []diffRead
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".go") {
			return err
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		r, _ := filepath.Rel(dir, p)
		rel := path.Join(at, filepath.ToSlash(r))
		for i, line := range strings.Split(string(b), "\n") {
			for _, m := range wantRE.FindAllStringSubmatch(line, -1) {
				for _, w := range strings.Fields(m[1]) {
					want = append(want, diffRead{rel, i + 1, w})
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	want = dedupe(want)
	if len(want) < 61 {
		t.Fatalf("the fixture plants %d reads; it lost some", len(want))
	}
	for _, w := range want {
		if !slices.Contains(got, w) {
			t.Errorf("planted read not reported: %s", w)
		}
	}
	for _, g := range got {
		if !slices.Contains(want, g) {
			t.Errorf("look-alike reported: %s", g)
		}
	}

	// The same files inside the reader are allowed: the check is by path.
	if got := scanDir(t, root, dir, "internal/audit/read/fixture", idx); len(got) != 0 {
		t.Errorf("internal/audit/read may read the diff, got %v", got)
	}
	// But not a sibling with a shared prefix, nor the rest of internal/audit.
	for _, at := range []string{"internal/audit/reader", "internal/audit"} {
		if got := scanDir(t, root, dir, at, idx); len(got) != len(want) {
			t.Errorf("%s is not the reader: want %d findings, got %d", at, len(want), len(got))
		}
	}
}

// TestPlantedHandlerDiffReadReported is the TICK-541 criterion as written: a
// handler outside internal/audit/read that selects diff from audit_log fails
// the gate.
func TestPlantedHandlerDiffReadReported(t *testing.T) {
	const src = `package server

const rawHistorySQL = "SELECT id, diff, diff_salt FROM audit_log WHERE subject_id = $1"
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "internal/server/planted.go", src, parser.SkipObjectResolution)
	if err != nil {
		t.Fatal(err)
	}
	got := scanFile(fset, "internal/server/planted.go", f, packageConsts([]*ast.File{f}, nil))
	want := []diffRead{
		{"internal/server/planted.go", 3, "diff"},
		{"internal/server/planted.go", 3, "diff_salt"},
	}
	if !slices.Equal(got, dedupe(want)) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestDiffReads(t *testing.T) {
	cases := map[string][]string{
		"SELECT diff FROM audit_log":                                           {"diff"},
		"select\n\tDIFF_SALT\nfrom   PUBLIC.\"audit_log\" where true":          {"diff_salt"},
		"SELECT a.diff FROM records r JOIN audit_log a ON a.subject_id = r.id": {"diff"},
		"SELECT * FROM audit_log":                                              {"star"},
		"SELECT a.* FROM audit_log a":                                          {"star"},
		"SELECT to_jsonb(a) FROM audit_log a":                                  {"row"},
		"SELECT diff FROM ONLY audit_log_default":                              {"diff"},
		"UPDATE audit_log SET erased_at = now() RETURNING diff":                {"diff"},
		"DELETE FROM audit_log WHERE id = $1 RETURNING *":                      {"star"},
		"SELECT json_agg(a) FROM audit_log a":                                  {"row"},
		"SELECT jsonb_agg(l ORDER BY l.seq) FROM audit_log AS l":               {"row"},
		"SELECT array_agg(DISTINCT audit_log) FROM audit_log":                  {"row"},
		"SELECT a::text FROM audit_log a":                                      {"row"},
		"SELECT audit_log FROM audit_log":                                      {"row"},
		"SELECT a.id, a FROM audit_log a WHERE true":                           {"row"},
		"COPY audit_log TO STDOUT":                                             {"copy"},
		"COPY public.audit_log (id, diff_salt) TO STDOUT":                      {"diff_salt"},
		"TABLE audit_log":                                                      {"table"},
		"SELECT 1 UNION ALL TABLE only public.audit_log":                       {"table"},
		"SELECT a FROM audit_log a":                                            {"row"},
		// TICK-551.
		"SELECT a.diff FROM records r, audit_log a":                 {"diff"},
		"SELECT to_jsonb(a) FROM records r, audit_log a":            {"row"},
		"SELECT x FROM records r , ONLY public.audit_log_default a": nil,
		"SELECT a.id FROM records r,audit_log a, other o":           nil,
		"SELECT * FROM records r, audit_log a":                      {"star"},
		"WITH x AS (SELECT 1) TABLE audit_log":                      {"table"},
		"INSERT INTO t TABLE audit_log":                             {"table"},
		"INSERT INTO s.t AS x TABLE audit_log":                      {"table"},
		"INSERT INTO t (a, b) TABLE audit_log":                      {"table"},
		"CREATE TABLE t AS TABLE audit_log":                         {"table"},
		"SELECT (a).* FROM audit_log a":                             {"star"},
		"SELECT ( a ) . * FROM audit_log a":                         {"star"},
		"SELECT diff FROM records r, audit_log_v0 a":                nil,
		"SELECT r.diff, r.audit_log_id FROM records r, other o":     nil,
		"WITH x AS (SELECT 1) TABLE audit_log_v0":                   nil,
		"INSERT INTO audit_log TABLE staging":                       nil,
		"SELECT (a).id FROM audit_log a":                            nil,
		"SELECT count(*) FROM records r, audit_log a":               nil,
		// Review round 1: TABLE fails closed, so unlisted prefixes are seen
		// and only the DDL and ACL forms stay quiet.
		"INSERT INTO t OVERRIDING SYSTEM VALUE TABLE audit_log":        {"table"},
		"INSERT INTO t (a, b) OVERRIDING USER VALUE TABLE audit_log":   {"table"},
		"DECLARE c CURSOR FOR TABLE audit_log":                         {"table"},
		"DECLARE c CURSOR WITH HOLD FOR TABLE ONLY public.audit_log":   {"table"},
		"WITH x AS MATERIALIZED (SELECT 1) TABLE audit_log":            {"table"},
		"PREPARE p AS TABLE audit_log":                                 {"table"},
		"(TABLE audit_log)":                                            {"table"},
		"ALTER TABLE ONLY audit_log ADD COLUMN x int; TABLE audit_log": {"table"},
		"SELECT x FROM astable audit_log":                              nil,
		"SELECT x FROM audit_table audit_log":                          nil,
		"DROP TABLE audit_log":                                         nil,
		"TRUNCATE TABLE ONLY audit_log":                                nil,
		"LOCK TABLE audit_log IN SHARE MODE":                           nil,
		"REVOKE ALL ON TABLE public.audit_log FROM nava_app":           nil,
		"COMMENT ON TABLE audit_log IS 'x'":                            nil,
		"CREATE TABLE audit_log (id int)":                              nil,
		"CREATE TABLE t (id int REFERENCES audit_log)":                 nil,
		// Security A, B, C look-alikes (the positives are in
		// diffreaders_bypass_test.go).
		"SELECT a.diff FROM records r, (audit_log_v0 a CROSS JOIN x)": nil,
		"SELECT s.diff FROM (SELECT diff FROM record_diffs) s":        nil,
		"SELECT diff FROM public . audit_log_v0":                      nil,
		"SELECT a . id, a . seq * 2 FROM audit_log a":                 nil,
		"SELECT a.diff FROM ( ( audit_log a ) )":                      {"diff"},
		// TICK-559 (carried over from TICK-551): ONLY (audit_log) and
		// audit_log *, each with look-alikes.
		"SELECT a.diff FROM ONLY (audit_log) a":                           {"diff"},
		"SELECT to_jsonb(a) FROM ONLY ( public.audit_log ) AS a":          {"row"},
		"SELECT to_jsonb(a) FROM records r, only(audit_log) a":            {"row"},
		"SELECT to_jsonb(a) FROM audit_log * a":                           {"row"},
		"SELECT to_jsonb(a) FROM records r JOIN audit_log * AS a ON true": {"row"},
		"TABLE ONLY (audit_log)":                                          {"table"},
		"SELECT a.id FROM ONLY (audit_log) a":                             nil,
		"SELECT a.id, a.seq * 2 FROM audit_log * a":                       nil,
		"SELECT to_jsonb(s) FROM (SELECT id FROM audit_log) s":            nil,
		"SELECT to_jsonb(a) FROM onlyaudit_log a":                         nil,
		"SELECT a.diff FROM ONLY (audit_log_v0) a":                        nil,
		"SELECT diff FROM records WHERE kind IN ('x', 'audit_log')":       nil,
		// Not a read of the protected columns.
		"SELECT diff_sha256, diff_truncated, diff_redacted FROM audit_log": nil,
		"SELECT count(*) FROM audit_log":                                   nil,
		"SELECT diff FROM audit_log_v0":                                    nil,
		"SELECT diff FROM other_log":                                       nil,
		"INSERT INTO audit_log (diff, diff_salt) VALUES ($1, $2)":          nil,
		"-- SELECT diff FROM audit_log\nSELECT 1":                          nil,
		"SELECT 1 /* diff FROM audit_log */":                               nil,
		"SELECT array_agg(id) FROM audit_log":                              nil,
		"SELECT tenant_id::text FROM audit_log a":                          nil,
		"SELECT a.id::text, a.seq FROM audit_log a":                        nil,
		"SELECT json_agg(a.id) FROM audit_log a WHERE true":                nil,
		"COPY audit_log (id) FROM STDIN":                                   nil,
		"CREATE TABLE audit_log_copy (LIKE audit_log)":                     nil,
		"ALTER TABLE audit_log ADD COLUMN x int":                           nil,
		"GRANT SELECT ON TABLE audit_log TO nava_app":                      nil,
		"COPY audit_log (id, diff_sha256) TO STDOUT":                       nil,
	}
	for in, want := range cases {
		if got := diffReads(in); !slices.Equal(got, want) {
			t.Errorf("diffReads(%q) = %v, want %v", in, got, want)
		}
	}
}

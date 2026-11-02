package interp_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// Retired histories (ADR-003 §1 and §3.1-§3.3, RFC-007 rev 1, PLAN-004 T-1).
//
// A history whose GetVersion DefaultVersion arm was retired is deleted from
// histories/ and leaves a tombstone in retired-histories.json, beside
// histories/ so the replay glob and checkHistorySet do not see it. This file
// holds the rules that need no BASE (R-4, R-5', R-7'); the BASE rules live in
// TestHistoriesAppendOnly.

// retiredManifestFormat is the only manifest format this test reads.
const retiredManifestFormat = "nava.retired-histories/v1"

// temporalWorkflowPkg is the import path whose GetVersion the scan matches.
const temporalWorkflowPkg = "go.temporal.io/sdk/workflow"

// retiredManifestPath is the manifest, relative to this package.
func retiredManifestPath() string {
	return filepath.Join("..", "testdata", "retired-histories.json")
}

// retiredEntry is one tombstone: a deleted history and the change id whose
// DefaultVersion arm its deletion retired.
type retiredEntry struct {
	History  string `json:"history"`   // file name under histories/: non-empty, no "/", ends ".json"
	Blob     string `json:"blob"`      // 40 lowercase hex
	ChangeID string `json:"change_id"` // non-empty
	Issue    string `json:"issue"`     // ^TICK-[0-9]+$
}

type retiredManifest struct {
	Format  string         `json:"format"`  // exactly "nava.retired-histories/v1"
	Retired []retiredEntry `json:"retired"` // present, may be empty
}

// decodeRetiredManifest decodes b strictly: unknown fields, trailing data, a
// missing or null retired list, a wrong format, a malformed entry and a
// duplicate history all fail. Every entry problem is reported, not only the
// first.
func decodeRetiredManifest(b []byte) (retiredManifest, error) {
	var raw struct {
		Format  string          `json:"format"`
		Retired *[]retiredEntry `json:"retired"`
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&raw); err != nil {
		return retiredManifest{}, fmt.Errorf("retired-histories manifest: %w", err)
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return retiredManifest{}, errors.New("retired-histories manifest: trailing data after the object")
	}
	if raw.Format != retiredManifestFormat {
		return retiredManifest{}, fmt.Errorf("retired-histories manifest: format is %q, want %q", raw.Format, retiredManifestFormat)
	}
	if raw.Retired == nil {
		return retiredManifest{}, errors.New(`retired-histories manifest: "retired" is missing or null (an empty list is []`)
	}
	m := retiredManifest{Format: raw.Format, Retired: *raw.Retired}
	blobRE := regexp.MustCompile(`^[0-9a-f]{40}$`)
	issueRE := regexp.MustCompile(`^TICK-[0-9]+$`)
	var problems []string
	seen := map[string]int{}
	for i, e := range m.Retired {
		at := fmt.Sprintf("retired[%d]", i)
		switch {
		case e.History == "":
			problems = append(problems, at+": history is empty")
		case strings.Contains(e.History, "/"):
			problems = append(problems, fmt.Sprintf("%s: history %q contains \"/\" (a file name under histories/, not a path)", at, e.History))
		case !strings.HasSuffix(e.History, ".json"):
			problems = append(problems, fmt.Sprintf("%s: history %q does not end in .json", at, e.History))
		}
		if j, dup := seen[e.History]; dup && e.History != "" {
			problems = append(problems, fmt.Sprintf("%s: duplicate history %q (also retired[%d])", at, e.History, j))
		} else {
			seen[e.History] = i
		}
		if !blobRE.MatchString(e.Blob) {
			problems = append(problems, fmt.Sprintf("%s: blob %q is not 40 lowercase hex characters", at, e.Blob))
		}
		if e.ChangeID == "" {
			problems = append(problems, at+": change_id is empty")
		}
		if !issueRE.MatchString(e.Issue) {
			problems = append(problems, fmt.Sprintf("%s: issue %q does not match ^TICK-[0-9]+$", at, e.Issue))
		}
	}
	if len(problems) > 0 {
		return retiredManifest{}, errors.New("retired-histories manifest:\n" + strings.Join(problems, "\n"))
	}
	return m, nil
}

// getVersionCall is one workflow.GetVersion call. DefaultVersion is recorded
// as -1.
type getVersionCall struct {
	File         string // slash path relative to the repo root
	Line         int
	ChangeID     string
	MinSupported int
	MaxSupported int
}

// scanGetVersionCalls parses each Go file in srcs (slash path -> content) and
// returns every call whose function resolves to GetVersion in
// go.temporal.io/sdk/workflow, under any local name or alias. R-7': the
// error names EVERY call whose changeID is not a string literal, whose
// minSupported is not workflow.DefaultVersion (via the same import) or an int
// literal >= 0, or whose maxSupported is not an int literal; every file that
// dot-imports the package; every use of GetVersion that is not a direct call
// (a func value would hide the arguments); and every file that does not
// parse. The calls it could read are returned with the error, sorted by file
// and line. It never touches disk.
func scanGetVersionCalls(srcs map[string][]byte) ([]getVersionCall, error) {
	files := make([]string, 0, len(srcs))
	for f := range srcs {
		files = append(files, f)
	}
	slices.Sort(files)
	var calls []getVersionCall
	var problems []string
	for _, name := range files {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, name, srcs[name], parser.SkipObjectResolution)
		if err != nil {
			problems = append(problems, fmt.Sprintf("%s: does not parse: %v", name, err))
			continue
		}
		// pkgNames: the file's local names for the workflow package.
		pkgNames := map[string]bool{}
		for _, imp := range f.Imports {
			path, err := strconv.Unquote(imp.Path.Value)
			if err != nil || path != temporalWorkflowPkg {
				continue
			}
			local := "workflow"
			if imp.Name != nil {
				local = imp.Name.Name
			}
			switch local {
			case ".":
				problems = append(problems, fmt.Sprintf("%s:%d: dot-imports %s; import it by name so GetVersion calls can be found (R-7')",
					name, fset.Position(imp.Pos()).Line, temporalWorkflowPkg))
			case "_":
			default:
				pkgNames[local] = true
			}
		}
		if len(pkgNames) == 0 {
			continue
		}
		// isPkgSel: e is <local name of the workflow package>.<sel>. A local
		// variable that shadows the package name would also match; the
		// workflowcheck lint and review keep that out of internal/workflow.
		isPkgSel := func(e ast.Expr, sel string) bool {
			s, ok := e.(*ast.SelectorExpr)
			if !ok || s.Sel.Name != sel {
				return false
			}
			id, ok := s.X.(*ast.Ident)
			return ok && pkgNames[id.Name]
		}
		called := map[ast.Expr]bool{}
		ast.Inspect(f, func(n ast.Node) bool {
			c, ok := n.(*ast.CallExpr)
			if !ok || !isPkgSel(c.Fun, "GetVersion") {
				return true
			}
			called[c.Fun] = true
			line := fset.Position(c.Pos()).Line
			at := fmt.Sprintf("%s:%d", name, line)
			call, why := readGetVersionArgs(c, func(e ast.Expr) bool { return isPkgSel(e, "DefaultVersion") })
			if len(why) > 0 {
				problems = append(problems, at+": GetVersion "+strings.Join(why, "; ")+" (R-7')")
				return true
			}
			call.File, call.Line = name, line
			calls = append(calls, call)
			return true
		})
		ast.Inspect(f, func(n ast.Node) bool {
			if e, ok := n.(ast.Expr); ok && isPkgSel(e, "GetVersion") && !called[e] {
				problems = append(problems, fmt.Sprintf("%s:%d: GetVersion is used as a value, not called; call it directly (R-7')",
					name, fset.Position(e.Pos()).Line))
			}
			return true
		})
	}
	slices.SortFunc(calls, func(a, b getVersionCall) int {
		if c := strings.Compare(a.File, b.File); c != 0 {
			return c
		}
		return a.Line - b.Line
	})
	if len(problems) > 0 {
		return calls, errors.New("GetVersion calls must use literal arguments (RFC-007 R-7'):\n" + strings.Join(problems, "\n"))
	}
	return calls, nil
}

// readGetVersionArgs reads GetVersion(ctx, changeID, minSupported,
// maxSupported) or returns why it cannot.
func readGetVersionArgs(c *ast.CallExpr, isDefaultVersion func(ast.Expr) bool) (getVersionCall, []string) {
	if len(c.Args) != 4 || c.Ellipsis.IsValid() {
		return getVersionCall{}, []string{"does not have exactly four plain arguments"}
	}
	var call getVersionCall
	var why []string
	if lit, ok := c.Args[1].(*ast.BasicLit); ok && lit.Kind == token.STRING {
		s, err := strconv.Unquote(lit.Value)
		if err != nil {
			why = append(why, "changeID literal does not unquote")
		}
		call.ChangeID = s
	} else {
		why = append(why, "changeID is not a string literal")
	}
	switch {
	case isDefaultVersion(c.Args[2]):
		call.MinSupported = -1
	default:
		v, ok := intLit(c.Args[2])
		if !ok {
			why = append(why, "minSupported is not workflow.DefaultVersion or an int literal >= 0")
		}
		call.MinSupported = v
	}
	v, ok := intLit(c.Args[3])
	if !ok {
		why = append(why, "maxSupported is not an int literal")
	}
	call.MaxSupported = v
	return call, why
}

// intLit reads a non-negative int literal. A negative number is a unary
// expression in the AST, so it is refused here too.
func intLit(e ast.Expr) (int, bool) {
	lit, ok := e.(*ast.BasicLit)
	if !ok || lit.Kind != token.INT {
		return 0, false
	}
	v, err := strconv.ParseInt(lit.Value, 0, 32)
	if err != nil {
		return 0, false
	}
	return int(v), true
}

// workflowSources reads every non-test .go file under <repo>/internal/workflow/...
// with the go tool's "..." rules: testdata and directories starting with "."
// or "_" are skipped. Build tags are ignored, so every variant is scanned.
func workflowSources(repo string) (map[string][]byte, error) {
	root := filepath.Join(repo, "internal", "workflow")
	srcs := map[string][]byte{}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := d.Name()
		if d.IsDir() {
			if p != root && (name == "testdata" || strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_")) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			return nil
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(repo, p)
		if err != nil {
			return err
		}
		srcs[filepath.ToSlash(rel)] = b
		return nil
	})
	if err != nil {
		return nil, err
	}
	return srcs, nil
}

// checkRetiredManifest applies R-4 (no entry's history exists in
// historyNames) and R-5' (for every entry: at least one call has ChangeID ==
// entry.ChangeID, and every such call has MinSupported > -1). It returns one
// error listing every violation.
func checkRetiredManifest(m retiredManifest, historyNames []string, calls []getVersionCall) error {
	var problems []string
	for i, e := range m.Retired {
		at := fmt.Sprintf("retired[%d] (%s, change_id %q)", i, e.History, e.ChangeID)
		if slices.Contains(historyNames, e.History) {
			problems = append(problems, at+": R-4: histories/ holds a file with this retired name; a retired name is never reused, record the new history under a new name")
		}
		found := false
		for _, c := range calls {
			if c.ChangeID != e.ChangeID {
				continue
			}
			found = true
			if c.MinSupported <= -1 {
				problems = append(problems, fmt.Sprintf("%s: R-5': %s:%d still supports workflow.DefaultVersion; raise minSupported to the first non-default version",
					at, c.File, c.Line))
			}
		}
		if !found {
			problems = append(problems, at+": R-5': no workflow.GetVersion call has this change id; the call stays in the code forever, with minSupported raised")
		}
	}
	if len(problems) > 0 {
		return errors.New("retired histories (RFC-007 §3.3):\n" + strings.Join(problems, "\n"))
	}
	return nil
}

// TestRetiredHistoriesManifest runs R-4, R-5' and R-7' on the real manifest,
// the real internal/workflow/... sources and the real histories/. It needs
// no BASE, network or Temporal server.
func TestRetiredHistoriesManifest(t *testing.T) {
	b, err := os.ReadFile(retiredManifestPath())
	if err != nil {
		t.Fatal(err)
	}
	m, err := decodeRetiredManifest(b)
	if err != nil {
		t.Fatal(err)
	}
	repo, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	srcs, err := workflowSources(repo)
	if err != nil {
		t.Fatal(err)
	}
	// A walk from the wrong root would find nothing and pass every rule.
	if _, ok := srcs["internal/workflow/interp/workflow.go"]; !ok {
		t.Fatalf("workflowSources(%s) did not find internal/workflow/interp/workflow.go; got %d files", repo, len(srcs))
	}
	for f := range srcs {
		if strings.HasSuffix(f, "_test.go") || strings.Contains(f, "/testdata/") {
			t.Errorf("workflowSources returned %s, which is not a scanned source", f)
		}
	}
	calls, err := scanGetVersionCalls(srcs)
	if err != nil {
		t.Fatal(err)
	}
	ents, err := os.ReadDir(historiesDir)
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(ents))
	for _, e := range ents {
		names = append(names, e.Name())
	}
	if len(names) == 0 {
		t.Fatalf("%s is empty", historiesDir)
	}
	if err := checkRetiredManifest(m, names, calls); err != nil {
		t.Fatal(err)
	}
}

// --- Proof the checks work ------------------------------------------------

const (
	okBlob = "0123456789abcdef0123456789abcdef01234567"
	// okEntry is a valid manifest entry; the decode cases break one field.
	okEntry = `{"history": "else-default.json", "blob": "` + okBlob + `", "change_id": "step-ref-before-timer", "issue": "TICK-412"}`
)

func manifestJSON(entries ...string) string {
	return `{"format": "nava.retired-histories/v1", "retired": [` + strings.Join(entries, ", ") + `]}`
}

func TestDecodeRetiredManifest(t *testing.T) {
	entry := func(field, value string) string {
		var m map[string]any
		if err := json.Unmarshal([]byte(okEntry), &m); err != nil {
			t.Fatal(err)
		}
		m[field] = value
		b, err := json.Marshal(m)
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	t.Run("valid", func(t *testing.T) {
		for _, in := range []string{manifestJSON(), manifestJSON(okEntry)} {
			m, err := decodeRetiredManifest([]byte(in))
			if err != nil {
				t.Fatalf("%s: %v", in, err)
			}
			if m.Format != retiredManifestFormat || m.Retired == nil {
				t.Fatalf("%s decoded to %+v", in, m)
			}
		}
		m, _ := decodeRetiredManifest([]byte(manifestJSON(okEntry)))
		want := retiredEntry{History: "else-default.json", Blob: okBlob, ChangeID: "step-ref-before-timer", Issue: "TICK-412"}
		if len(m.Retired) != 1 || m.Retired[0] != want {
			t.Fatalf("got %+v, want [%+v]", m.Retired, want)
		}
	})
	bad := []struct {
		name, in, want string
	}{
		{"unknown top-level field", `{"format": "nava.retired-histories/v1", "retired": [], "note": "x"}`, `unknown field "note"`},
		{"unknown entry field", manifestJSON(strings.Replace(okEntry, `{`, `{"reason": "x", `, 1)), `unknown field "reason"`},
		{"trailing data", manifestJSON() + `{}`, "trailing data"},
		{"retired missing", `{"format": "nava.retired-histories/v1"}`, `"retired" is missing`},
		{"retired null", `{"format": "nava.retired-histories/v1", "retired": null}`, `"retired" is missing`},
		{"wrong format", `{"format": "nava.retired-histories/v2", "retired": []}`, "format is"},
		{"format missing", `{"retired": []}`, "format is"},
		{"duplicate history", manifestJSON(okEntry, entry("change_id", "other")), `duplicate history "else-default.json"`},
		{"history empty", manifestJSON(entry("history", "")), "history is empty"},
		{"history with /", manifestJSON(entry("history", "sub/else-default.json")), `contains "/"`},
		{"history not .json", manifestJSON(entry("history", "else-default.yaml")), "does not end in .json"},
		{"blob uppercase", manifestJSON(entry("blob", strings.ToUpper(okBlob))), "not 40 lowercase hex"},
		{"blob short", manifestJSON(entry("blob", okBlob[:39])), "not 40 lowercase hex"},
		{"blob not hex", manifestJSON(entry("blob", "g"+okBlob[1:])), "not 40 lowercase hex"},
		{"change_id empty", manifestJSON(entry("change_id", "")), "change_id is empty"},
		{"issue lowercase", manifestJSON(entry("issue", "tick-412")), "does not match ^TICK-"},
		{"issue no number", manifestJSON(entry("issue", "TICK-")), "does not match ^TICK-"},
		{"issue other prefix", manifestJSON(entry("issue", "MUL-412")), "does not match ^TICK-"},
		{"issue trailing text", manifestJSON(entry("issue", "TICK-412 x")), "does not match ^TICK-"},
	}
	for _, c := range bad {
		t.Run(c.name, func(t *testing.T) {
			_, err := decodeRetiredManifest([]byte(c.in))
			if err == nil {
				t.Fatalf("decoded %s", c.in)
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Fatalf("error %q does not contain %q", err, c.want)
			}
		})
	}
	t.Run("every entry problem is reported", func(t *testing.T) {
		_, err := decodeRetiredManifest([]byte(manifestJSON(entry("blob", "x"), entry("issue", "x"))))
		if err == nil || !strings.Contains(err.Error(), "retired[0]: blob") || !strings.Contains(err.Error(), "retired[1]: issue") {
			t.Fatalf("got %v, want both entries reported", err)
		}
	})
}

// goSrc is one in-memory Go file in package p that imports the workflow
// package under imp (an import spec such as `"go.temporal.io/sdk/workflow"`)
// and runs body inside a workflow function.
func goSrc(imp, body string) []byte {
	return []byte("package p\n\nimport " + imp + "\n\nfunc f(ctx workflow.Context) {\n" + body + "\n}\n")
}

const wfImport = `"go.temporal.io/sdk/workflow"`

func TestScanGetVersionCalls(t *testing.T) {
	t.Run("literal calls are read", func(t *testing.T) {
		srcs := map[string][]byte{
			"internal/workflow/b.go": goSrc(wfImport, "_ = workflow.GetVersion(ctx, \"b\", workflow.DefaultVersion, 1)"),
			"internal/workflow/a.go": goSrc(wfImport, "_ = workflow.GetVersion(ctx, `a`, 1, 0x2)\n_ = workflow.GetVersion(ctx, \"a2\", 0, 3)"),
		}
		calls, err := scanGetVersionCalls(srcs)
		if err != nil {
			t.Fatal(err)
		}
		want := []getVersionCall{
			{File: "internal/workflow/a.go", Line: 6, ChangeID: "a", MinSupported: 1, MaxSupported: 2},
			{File: "internal/workflow/a.go", Line: 7, ChangeID: "a2", MinSupported: 0, MaxSupported: 3},
			{File: "internal/workflow/b.go", Line: 6, ChangeID: "b", MinSupported: -1, MaxSupported: 1},
		}
		if !slices.Equal(calls, want) {
			t.Fatalf("got %+v\nwant %+v", calls, want)
		}
	})
	t.Run("alias is matched", func(t *testing.T) {
		src := []byte("package p\n\nimport tw \"go.temporal.io/sdk/workflow\"\n\nfunc f(ctx tw.Context) {\n" +
			"_ = tw.GetVersion(ctx, \"x\", 1, 2)\n_ = tw.GetVersion(ctx, \"y\", tw.DefaultVersion, 1)\n}\n")
		calls, err := scanGetVersionCalls(map[string][]byte{"internal/workflow/p.go": src})
		if err != nil {
			t.Fatal(err)
		}
		want := []getVersionCall{
			{File: "internal/workflow/p.go", Line: 6, ChangeID: "x", MinSupported: 1, MaxSupported: 2},
			{File: "internal/workflow/p.go", Line: 7, ChangeID: "y", MinSupported: -1, MaxSupported: 1},
		}
		if !slices.Equal(calls, want) {
			t.Fatalf("got %+v\nwant %+v", calls, want)
		}
	})
	t.Run("GetVersion from another package is not matched", func(t *testing.T) {
		srcs := map[string][]byte{
			// "workflow" names another package here; its GetVersion is not Temporal's.
			"internal/workflow/a.go": []byte("package p\n\nimport workflow \"example.com/other/workflow\"\n\nfunc f() {\n" +
				"_ = workflow.GetVersion(nil, name, v, w)\n}\n"),
			// Temporal's package under an alias; workflow.GetVersion is someone else's.
			"internal/workflow/b.go": []byte("package p\n\nimport (\n\ttw \"go.temporal.io/sdk/workflow\"\n\t\"example.com/workflow\"\n)\n\n" +
				"func f(ctx tw.Context) {\n_ = workflow.GetVersion(ctx, id, v, w)\n}\n"),
			// No import at all.
			"internal/workflow/c.go": []byte("package p\n\nfunc f() { _ = workflow.GetVersion(nil, id, v, w) }\n"),
		}
		calls, err := scanGetVersionCalls(srcs)
		if err != nil || len(calls) != 0 {
			t.Fatalf("got %+v, %v; want no calls and no error", calls, err)
		}
	})
	bad := []struct {
		name string
		src  []byte
		want string
	}{
		{"non-literal change id (a const)",
			goSrc(wfImport, "const id = \"x\"\n_ = workflow.GetVersion(ctx, id, 1, 2)"),
			"p.go:7: GetVersion changeID is not a string literal"},
		{"non-literal change id (an expression)",
			goSrc(wfImport, "_ = workflow.GetVersion(ctx, \"x\"+\"y\", 1, 2)"),
			"p.go:6: GetVersion changeID is not a string literal"},
		{"non-literal minSupported (a variable)",
			goSrc(wfImport, "v := workflow.Version(1)\n_ = workflow.GetVersion(ctx, \"x\", v, 2)"),
			"p.go:7: GetVersion minSupported is not"},
		{"negative minSupported",
			goSrc(wfImport, "_ = workflow.GetVersion(ctx, \"x\", -1, 2)"),
			"minSupported is not"},
		{"DefaultVersion of another package",
			[]byte("package p\n\nimport (\n\t\"go.temporal.io/sdk/workflow\"\n\tother \"example.com/workflow\"\n)\n\n" +
				"func f(ctx workflow.Context) {\n_ = workflow.GetVersion(ctx, \"x\", other.DefaultVersion, 2)\n}\n"),
			"minSupported is not"},
		{"non-literal maxSupported (a variable)",
			goSrc(wfImport, "v := workflow.Version(2)\n_ = workflow.GetVersion(ctx, \"x\", 1, v)"),
			"p.go:7: GetVersion maxSupported is not an int literal"},
		{"maxSupported DefaultVersion",
			goSrc(wfImport, "_ = workflow.GetVersion(ctx, \"x\", workflow.DefaultVersion, workflow.DefaultVersion)"),
			"maxSupported is not an int literal"},
		{"dot-import",
			[]byte("package p\n\nimport . \"go.temporal.io/sdk/workflow\"\n\nfunc f(ctx Context) {\n_ = GetVersion(ctx, \"x\", 1, 2)\n}\n"),
			"p.go:3: dot-imports go.temporal.io/sdk/workflow"},
		{"func value",
			goSrc(wfImport, "gv := workflow.GetVersion\n_ = gv(ctx, \"x\", 1, 2)"),
			"p.go:6: GetVersion is used as a value"},
		{"does not parse",
			[]byte("package p\n\nfunc f( {\n"),
			"p.go: does not parse"},
	}
	for _, c := range bad {
		t.Run(c.name, func(t *testing.T) {
			_, err := scanGetVersionCalls(map[string][]byte{"internal/workflow/p.go": c.src})
			if err == nil {
				t.Fatal("the scan accepted it")
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Fatalf("error %q does not contain %q", err, c.want)
			}
		})
	}
	t.Run("every bad call is named", func(t *testing.T) {
		calls, err := scanGetVersionCalls(map[string][]byte{
			"internal/workflow/a.go": goSrc(wfImport, "v := 1\n_ = workflow.GetVersion(ctx, \"x\", v, 2)\n_ = workflow.GetVersion(ctx, \"ok\", 1, 2)"),
			"internal/workflow/b.go": goSrc(wfImport, "id := \"y\"\n_ = workflow.GetVersion(ctx, id, 1, 2)"),
		})
		if err == nil || !strings.Contains(err.Error(), "a.go:7:") || !strings.Contains(err.Error(), "b.go:7:") {
			t.Fatalf("got %v, want both bad calls named", err)
		}
		if len(calls) != 1 || calls[0].ChangeID != "ok" {
			t.Fatalf("got %+v, want the one readable call", calls)
		}
	})
}

func TestCheckRetiredManifest(t *testing.T) {
	m := retiredManifest{Format: retiredManifestFormat, Retired: []retiredEntry{
		{History: "else-default.json", Blob: okBlob, ChangeID: "x", Issue: "TICK-1"},
	}}
	scan := func(t *testing.T, body string) []getVersionCall {
		t.Helper()
		calls, err := scanGetVersionCalls(map[string][]byte{"internal/workflow/p.go": goSrc(wfImport, body)})
		if err != nil {
			t.Fatal(err)
		}
		return calls
	}
	histories := []string{"signal.json", "timeout.json"}
	cases := []struct {
		name      string
		histories []string
		body      string
		want      string // "" passes
	}{
		{"retired and raised", histories, `_ = workflow.GetVersion(ctx, "x", 1, 1)`, ""},
		{"two calls, both raised", histories, "_ = workflow.GetVersion(ctx, \"x\", 1, 1)\n_ = workflow.GetVersion(ctx, \"x\", 2, 2)", ""},
		{"another id may keep DefaultVersion", histories,
			"_ = workflow.GetVersion(ctx, \"x\", 1, 2)\n_ = workflow.GetVersion(ctx, \"y\", workflow.DefaultVersion, 1)", ""},
		{"R-4: retired name reused", append(slices.Clone(histories), "else-default.json"),
			`_ = workflow.GetVersion(ctx, "x", 1, 1)`, "R-4: histories/ holds a file with this retired name"},
		{"R-5': call deleted", histories, `_ = workflow.GetVersion(ctx, "other", 1, 1)`,
			"R-5': no workflow.GetVersion call has this change id"},
		{"R-5': no calls at all", histories, `_ = ctx`, "R-5': no workflow.GetVersion call has this change id"},
		{"R-5': minSupported still DefaultVersion", histories, `_ = workflow.GetVersion(ctx, "x", workflow.DefaultVersion, 1)`,
			"R-5': internal/workflow/p.go:6 still supports workflow.DefaultVersion"},
		{"R-5': two calls, only the first raised", histories,
			"_ = workflow.GetVersion(ctx, \"x\", 1, 1)\n_ = workflow.GetVersion(ctx, \"x\", workflow.DefaultVersion, 1)",
			"R-5': internal/workflow/p.go:7 still supports workflow.DefaultVersion"},
		{"R-5': two calls, only the second raised", histories,
			"_ = workflow.GetVersion(ctx, \"x\", workflow.DefaultVersion, 1)\n_ = workflow.GetVersion(ctx, \"x\", 1, 1)",
			"R-5': internal/workflow/p.go:6 still supports workflow.DefaultVersion"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := checkRetiredManifest(m, c.histories, scan(t, c.body))
			switch {
			case c.want == "" && err != nil:
				t.Fatal(err)
			case c.want != "" && err == nil:
				t.Fatal("the check passed")
			case c.want != "" && !strings.Contains(err.Error(), c.want):
				t.Fatalf("error %q does not contain %q", err, c.want)
			}
		})
	}
	t.Run("every violation is listed", func(t *testing.T) {
		two := retiredManifest{Format: retiredManifestFormat, Retired: []retiredEntry{
			m.Retired[0],
			{History: "signal.json", Blob: okBlob, ChangeID: "gone", Issue: "TICK-2"},
		}}
		err := checkRetiredManifest(two, histories, scan(t, `_ = workflow.GetVersion(ctx, "x", workflow.DefaultVersion, 1)`))
		if err == nil {
			t.Fatal("the check passed")
		}
		for _, want := range []string{"retired[0] (else-default.json", "still supports", "retired[1] (signal.json", "R-4", "no workflow.GetVersion call"} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("error %q does not contain %q", err, want)
			}
		}
	})
	t.Run("an empty manifest passes anything", func(t *testing.T) {
		empty := retiredManifest{Format: retiredManifestFormat, Retired: []retiredEntry{}}
		if err := checkRetiredManifest(empty, histories, nil); err != nil {
			t.Fatal(err)
		}
	})
}

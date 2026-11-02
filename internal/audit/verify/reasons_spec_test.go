package verify_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// specPath is audit-export-v1.md, relative to this package.
const specPath = "../../../docs/api/audit-export-v1.md"

// TestReasonCodesListedInSpec holds audit-export-v1.md §4.1 to its word: it
// is the closed list of every reason code the verifier can emit
// (ADR-004-E2, decision 5). Every Reason* constant of this package must be a
// row of the §4.1 table, and every row must be a constant, so a new check
// cannot ship a code the spec does not name.
func TestReasonCodesListedInSpec(t *testing.T) {
	spec, err := specReasonCodes(specPath)
	if err != nil {
		t.Fatal(err)
	}
	consts, err := reasonConstants(".")
	if err != nil {
		t.Fatal(err)
	}
	if len(consts) == 0 {
		t.Fatal("no Reason* constants found in internal/audit/verify")
	}
	for name, code := range consts {
		if !spec[code] {
			t.Errorf("%s = %q is not listed in audit-export-v1.md §4.1", name, code)
		}
	}
	emitted := map[string]bool{}
	for _, code := range consts {
		emitted[code] = true
	}
	for code := range spec {
		if !emitted[code] {
			t.Errorf("audit-export-v1.md §4.1 lists %q, but no Reason* constant has that value", code)
		}
	}
}

// The parser itself must see a code that is missing from §4.1, or the test
// above proves nothing.
func TestReasonCodesListedInSpecDetectsMissing(t *testing.T) {
	b, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatal(err)
	}
	row := "| `seq_missing` |"
	if !strings.Contains(string(b), row) {
		t.Fatalf("§4.1 has no %s row", row)
	}
	var kept []string
	for _, l := range strings.Split(string(b), "\n") {
		if !strings.HasPrefix(l, row) {
			kept = append(kept, l)
		}
	}
	p := filepath.Join(t.TempDir(), "spec.md")
	if err := os.WriteFile(p, []byte(strings.Join(kept, "\n")), 0o600); err != nil {
		t.Fatal(err)
	}
	spec, err := specReasonCodes(p)
	if err != nil {
		t.Fatal(err)
	}
	if spec["seq_missing"] || !spec["anchor_mismatch"] {
		t.Fatalf("parser read %v from a §4.1 without the seq_missing row", spec)
	}
}

var reasonRow = regexp.MustCompile("^\\| `([a-z0-9_]+)` \\|")

// specReasonCodes reads the codes in the first column of the §4.1 table:
// the rows after the "### 4.1 Reason codes" heading, up to the next heading.
func specReasonCodes(path string) (map[string]bool, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	codes := map[string]bool{}
	in := false
	for _, l := range strings.Split(strings.ReplaceAll(string(b), "\r\n", "\n"), "\n") {
		switch {
		case strings.HasPrefix(l, "### 4.1 Reason codes"):
			in = true
		case in && strings.HasPrefix(l, "#"):
			in = false
		case in:
			if m := reasonRow.FindStringSubmatch(l); m != nil {
				codes[m[1]] = true
			}
		}
	}
	if len(codes) == 0 {
		return nil, os.ErrNotExist
	}
	return codes, nil
}

// reasonConstants returns every string constant named Reason* declared in the
// non-test Go files of dir, by name.
func reasonConstants(dir string) (map[string]string, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	fset := token.NewFileSet()
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		af, err := parser.ParseFile(fset, f, nil, 0)
		if err != nil {
			return nil, err
		}
		for _, d := range af.Decls {
			g, ok := d.(*ast.GenDecl)
			if !ok || g.Tok != token.CONST {
				continue
			}
			for _, sp := range g.Specs {
				vs, ok := sp.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for i, n := range vs.Names {
					if !strings.HasPrefix(n.Name, "Reason") || i >= len(vs.Values) {
						continue
					}
					lit, ok := vs.Values[i].(*ast.BasicLit)
					if !ok || lit.Kind != token.STRING {
						continue
					}
					v, err := strconv.Unquote(lit.Value)
					if err != nil {
						return nil, err
					}
					out[n.Name] = v
				}
			}
		}
	}
	return out, nil
}

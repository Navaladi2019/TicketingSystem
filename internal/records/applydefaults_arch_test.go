package records

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ADR-009 D11: every surface that returns a record body shows the effective
// value, pipeline.Effective(cur, rec, fixed). That helper is the one caller of
// metadata.Compiled.ApplyDefaults outside the write path, so a future surface
// (GraphQL, the embed SDK, an export) that calls ApplyDefaults itself, or
// returns the stored payload, is caught here and sent to the helper.
//
// The check is static and runs in `task test`: it parses every non-test Go
// file of the root module and reports each call of a method named
// ApplyDefaults outside internal/records/pipeline (the helper and the write
// path's own use) and internal/metadata (the method itself).

// applyDefaultsAllowed reports whether the file at rel (slash-separated,
// relative to the module root) may call ApplyDefaults.
func applyDefaultsAllowed(rel string) bool {
	return underAny(rel, "internal/records/pipeline", "internal/metadata")
}

// applyDefaultsCalls returns the position of every call of a method named
// ApplyDefaults in src, a file at rel.
func applyDefaultsCalls(t *testing.T, rel, src string) []string {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, rel, src, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("parse %s: %v", rel, err)
	}
	var out []string
	ast.Inspect(f, func(n ast.Node) bool {
		if c, ok := n.(*ast.CallExpr); ok {
			if sel, ok := c.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "ApplyDefaults" {
				out = append(out, fset.Position(c.Pos()).String())
			}
		}
		return true
	})
	return out
}

func TestApplyDefaultsOnlyInHelper(t *testing.T) {
	root := moduleRoot(t)
	var bad []string
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := d.Name()
		if d.IsDir() {
			if p != root && (name == "testdata" || name == "vendor" || name == "node_modules" ||
				strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_")) {
				return filepath.SkipDir
			}
			if p != root {
				if _, err := os.Stat(filepath.Join(p, "go.mod")); err == nil {
					return filepath.SkipDir // another module
				}
			}
			return nil
		}
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			return nil
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if applyDefaultsAllowed(rel) {
			return nil
		}
		src, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		bad = append(bad, applyDefaultsCalls(t, rel, string(src))...)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, b := range bad {
		t.Errorf("%s: ApplyDefaults called outside the effective-value helper; use pipeline.Effective (ADR-009 D11)", b)
	}
}

// TestApplyDefaultsFixture proves the check can fail: a stray call, as a new
// surface would write it, is reported, and the helper's own package is not.
func TestApplyDefaultsFixture(t *testing.T) {
	const stray = `package surface

func body(cur *Compiled, rec *Record) map[string]any {
	return cur.ApplyDefaults(rec.Payload(nil))
}
`
	if got := applyDefaultsCalls(t, "internal/server/graphql.go", stray); len(got) != 1 {
		t.Fatalf("a stray ApplyDefaults call was reported %d times, want 1: %v", len(got), got)
	}
	if applyDefaultsAllowed("internal/server/graphql.go") {
		t.Error("internal/server may not call ApplyDefaults")
	}
	for _, rel := range []string{"internal/records/pipeline/effective.go", "internal/metadata/validate.go"} {
		if !applyDefaultsAllowed(rel) {
			t.Errorf("%s may call ApplyDefaults", rel)
		}
	}
	if got := applyDefaultsCalls(t, "x.go", "package x\nfunc f() { _ = ApplyDefaults }\n"); len(got) != 0 {
		t.Errorf("a mention that is not a call was reported: %v", got)
	}
}

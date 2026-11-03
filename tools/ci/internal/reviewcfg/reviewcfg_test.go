package reviewcfg

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func writeFile(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "review-paths.yaml")
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoadDecodesBothEntryForms(t *testing.T) {
	p := writeFile(t, `security:
  - db/
  - path: cmd/nava-relay/
    planned: "PLAN-001 T30b"
cto_signoff:
  paths:
    - internal/db/catalog/policy.go
  isolation_tests:
    - "**/*isolation*_test.go"
`)
	cfg, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if want := (Entries{{Path: "db/"}, {Path: "cmd/nava-relay/", Planned: "PLAN-001 T30b"}}); !slices.Equal(cfg.Security, want) {
		t.Errorf("security = %+v, want %+v", cfg.Security, want)
	}
	if got := cfg.CTOSignoff.Paths.Patterns(); !slices.Equal(got, []string{"internal/db/catalog/policy.go"}) {
		t.Errorf("cto_signoff.paths = %q", got)
	}
	if !cfg.IsIsolationTest("a/b/tenant_isolation_test.go") || cfg.IsIsolationTest("a/b/tenant_test.go") {
		t.Error("isolation_tests not applied")
	}
}

func TestLoadRefusesWhatItCannotRead(t *testing.T) {
	if _, err := Load(filepath.Join(t.TempDir(), "absent.yaml")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("missing file: err = %v, want not-exist", err)
	}
	for _, tc := range []struct {
		name, doc, want string
	}{
		{"unknown top-level key", "security:\n  - db/\nsecurity_review:\n  - x/\n", "security_review"},
		{"unknown entry key", "security:\n  - path: db/\n    plan: \"PLAN-001 T09\"\n", "field plan not found in path entry"},
		{"entry without path", "security:\n  - planned: \"PLAN-001 T09\"\n", "path entry has no path"},
		{"entry neither form", "security:\n  - [db/]\n", "a path entry is a string or a {path, planned} mapping"},
		{"repeated entry key", "security:\n  - path: db/\n    path: deploy/\n", "field path already set in path entry"},
		{"path not a string", "security:\n  - path: [db/]\n", "cannot unmarshal"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := writeFile(t, tc.doc)
			_, err := Load(p)
			if err == nil {
				t.Fatalf("Load(%q) succeeded", tc.doc)
			}
			if !strings.HasPrefix(err.Error(), p+": ") || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("err = %v, want %q prefixed with the file", err, tc.want)
			}
		})
	}
}

func TestMatchAny(t *testing.T) {
	for _, tc := range []struct {
		pat, p string
		want   bool
	}{
		{"db/", "db/migrations/1.sql", true},
		{"db/", "dbx/1.sql", false},
		{"compose.yaml", "compose.yaml", true},
		{"compose.yaml", "deploy/compose.yaml", false},
		{"internal/*/x.go", "internal/a/x.go", true},
		{"internal/*/x.go", "internal/a/b/x.go", false},
		{"**/*isolation*_test.go", "isolation_test.go", true},
		{"**/*isolation*_test.go", "a/b/c/tenant_isolation_test.go", true},
		{"**/*isolation*_test.go", "isolation/x_test.go", false},
		{"[", "[", false}, // a malformed pattern matches nothing
	} {
		if got := MatchAny([]string{tc.pat}, tc.p); got != tc.want {
			t.Errorf("MatchAny(%q, %q) = %v, want %v", tc.pat, tc.p, got, tc.want)
		}
	}
	if MatchAny(nil, "db/x") {
		t.Error("an empty list matched")
	}
	if !MatchAny([]string{"**/nomatch", "db/", "x"}, "db/x") {
		t.Error("a later pattern in the list was not tried")
	}
}

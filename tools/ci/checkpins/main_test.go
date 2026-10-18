package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fixture copies testdata/<name> into a temp dir, renaming go.mod.txt to
// go.mod (a real go.mod under testdata would be a nested module to the go
// tool), and returns it.
func fixture(t *testing.T, name string) string {
	t.Helper()
	dir := t.TempDir()
	entries, err := os.ReadDir(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		data, err := os.ReadFile(filepath.Join("testdata", name, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, strings.TrimSuffix(e.Name(), ".txt")), data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func run(t *testing.T, dir string) string {
	t.Helper()
	pinsPath, err := filepath.Abs(filepath.Join("testdata", "pins.md"))
	if err != nil {
		t.Fatal(err)
	}
	problems, err := check(dir, pinsPath)
	if err != nil {
		t.Fatal(err)
	}
	return strings.Join(problems, "\n")
}

func TestPinnedFixturePasses(t *testing.T) {
	if got := run(t, fixture(t, "ok")); got != "" {
		t.Fatalf("want no problems, got:\n%s", got)
	}
}

// The T05 acceptance fixture: a go.mod that requires coreos/go-oidc/v3 fails,
// even though the module has a row (a row never makes a forbidden module ok).
func TestForbiddenRequireFails(t *testing.T) {
	got := run(t, fixture(t, "forbidden"))
	if !strings.Contains(got, "github.com/coreos/go-oidc/v3 v3.17.0, which is forbidden (github.com/coreos/go-oidc)") {
		t.Fatalf("want the forbidden require reported, got:\n%s", got)
	}
}

// An indirect dependency is caught through go.sum.
func TestForbiddenInGoSumFails(t *testing.T) {
	dir := fixture(t, "ok")
	sum := "github.com/coreos/go-oidc/v3 v3.17.0/go.mod h1:AAAA=\n"
	if err := os.WriteFile(filepath.Join(dir, "go.sum"), []byte(sum), 0o600); err != nil {
		t.Fatal(err)
	}
	got := run(t, dir)
	if !strings.Contains(got, "go.sum: github.com/coreos/go-oidc/v3 is in the module graph") {
		t.Fatalf("want the go.sum entry reported, got:\n%s", got)
	}
}

func TestUnpinnedOrDriftedRequireFails(t *testing.T) {
	got := run(t, fixture(t, "unpinned"))
	for _, want := range []string{
		"github.com/google/uuid v1.6.0 has no row",
		"github.com/jackc/pgx/v5 is v5.10.0, but docs/pins.md pins",
		"golang.org/x/crypto is v0.55.0, but docs/pins.md pins",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}

func TestImagesMustMatchARow(t *testing.T) {
	got := run(t, fixture(t, "image"))
	for _, want := range []string{
		"image postgres:16.9, but docs/pins.md pins",
		"image quay.io/keycloak/keycloak has no row",
		"image postgres has no exact tag",
		"image alpine:latest has no exact tag",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}

// The repository itself is in step with docs/pins.md.
func TestRepository(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	problems, err := check(root, filepath.Join(root, "docs", "pins.md"))
	if err != nil {
		t.Fatal(err)
	}
	if len(problems) > 0 {
		t.Fatalf("docs/pins.md is out of step:\n%s", strings.Join(problems, "\n"))
	}
}

func TestVersionListed(t *testing.T) {
	for _, tc := range []struct {
		cell, v string
		want    bool
	}{
		{"v1.47.0", "v1.47.0", true},
		{"v1.47.0", "v1.4", false},
		{"v1.84.0 (platform), v1.79.3 (`temporal-server`)", "v1.79.3", true},
		{"≥ v0.56.0", "v0.56.0", true},
		{"≥ v0.56.0", "v0.60.1", true},
		{"≥ v0.56.0", "v0.55.0", false},
		{"1.26.9-alpine (nava-core), 1.25.8-bookworm", "1.25.8-bookworm", true},
	} {
		if got := versionListed(tc.cell, tc.v); got != tc.want {
			t.Errorf("versionListed(%q, %q) = %v, want %v", tc.cell, tc.v, got, tc.want)
		}
	}
}

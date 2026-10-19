package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fixture copies testdata/<name> (subdirectories included) into a temp dir,
// dropping a ".txt" suffix: a real go.mod or .go file under testdata would be
// a nested module or source to the go tool. It returns the temp dir.
func fixture(t *testing.T, name string) string {
	t.Helper()
	dir := t.TempDir()
	src := filepath.Join("testdata", name)
	err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		dst := filepath.Join(dir, strings.TrimSuffix(rel, ".txt"))
		if d.IsDir() {
			return os.MkdirAll(dst, 0o700)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(dst, data, 0o600)
	})
	if err != nil {
		t.Fatal(err)
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

// The reviewer's probe (PR #19 round 1): the platform go.mod requiring
// temporal-server's vulnerable grpc v1.79.3 fails, while temporal-server/go.mod
// may hold it, because the row scopes each version to a module directory.
func TestScopedVersionFailsOutsideItsModule(t *testing.T) {
	got := run(t, fixture(t, "scoped"))
	want := `go.mod: google.golang.org/grpc is v1.79.3, but docs/pins.md pins "v1.84.0 @., v1.79.3 @temporal-server"`
	if got != want {
		t.Fatalf("got:\n%s\nwant exactly:\n%s", got, want)
	}
}

// Image tags come from docs/pins.md (TICK-94 item 16): a Go or shell file that
// spells a pinned image with a tag fails; a host:port does not.
func TestInlineImageLiteralFails(t *testing.T) {
	got := run(t, fixture(t, "literal"))
	for _, want := range []string{
		"start.sh:2: image postgres:16.10 spelled inline",
		"probe.go:6: image postgres:16.10 spelled inline",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, "keycloak") || strings.Count(got, "\n") != 1 {
		t.Errorf("want exactly the two literals, got:\n%s", got)
	}
}

func TestVersionListed(t *testing.T) {
	const grpc = "v1.84.0 @., v1.79.3 @temporal-server"
	const golang = "1.26.9-alpine @deploy/compose, 1.25.8-bookworm @temporal-server"
	for _, tc := range []struct {
		cell, v, dir string
		image        bool
		want         bool
	}{
		{"v1.47.0", "v1.47.0", ".", false, true},
		{"v1.47.0", "v1.4", ".", false, false},
		{grpc, "v1.84.0", ".", false, true},
		{grpc, "v1.79.3", ".", false, false},
		{grpc, "v1.79.3", "temporal-server", false, true},
		{grpc, "v1.84.0", "temporal-server", false, false},
		{grpc, "v1.79.3", "temporal-server/sub", false, false}, // go.mod: exact dir only
		{"v0.52.0 (navalint tools module v0.51.0)", "v0.51.0", ".", false, false},
		{"≥ v0.56.0", "v0.56.0", ".", false, true},
		{"≥ v0.56.0", "v0.60.1", ".", false, true},
		{"≥ v0.56.0", "v0.55.0", ".", false, false},
		{golang, "1.26.9-alpine", "deploy/compose/nava-core", true, true},
		{golang, "1.25.8-bookworm", "deploy/compose/nava-core", true, false},
		{golang, "1.25.8-bookworm", "temporal-server", true, true},
		{golang, "1.26.9-alpine", "deploy/composer", true, false},
	} {
		if got := versionListed(tc.cell, tc.v, tc.dir, tc.image); got != tc.want {
			t.Errorf("versionListed(%q, %q, %q) = %v, want %v", tc.cell, tc.v, tc.dir, got, tc.want)
		}
	}
}

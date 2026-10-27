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

// writeFile replaces the contents of dir/rel.
func writeFile(t *testing.T, dir, rel, data string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, filepath.FromSlash(rel)), []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
}

// replaceIn swaps old for new in dir/rel, failing if old is not there.
func replaceIn(t *testing.T, dir, rel, old, new string) {
	t.Helper()
	path := filepath.Join(dir, filepath.FromSlash(rel))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), old) {
		t.Fatalf("%s does not contain %q", rel, old)
	}
	writeFile(t, dir, rel, strings.Replace(string(data), old, new, 1))
}

// TICK-193: a locally built nava/* image spelled ${NAVA_IMAGE_TAG:-dev} is
// checked against its row, in compose.yaml and in the overlays.
func TestBuiltImageDefaultPasses(t *testing.T) {
	if got := run(t, fixture(t, "built")); got != "" {
		t.Fatalf("want no problems, got:\n%s", got)
	}
}

func TestBuiltImageTagDriftFails(t *testing.T) {
	dir := fixture(t, "built")
	replaceIn(t, dir, "compose.yaml", "nava/keycloak:${NAVA_IMAGE_TAG:-dev}", "nava/keycloak:${NAVA_IMAGE_TAG:-dev2}")
	replaceIn(t, dir, "deploy/compose/compose.ci.yaml", "${NAVA_IMAGE_TAG:-dev}", "26.8.1")
	got := run(t, dir)
	want := `compose.yaml:3: image nava/keycloak:dev2, but docs/pins.md pins "dev"` + "\n" +
		`deploy/compose/compose.ci.yaml:4: image nava/keycloak:26.8.1, but docs/pins.md pins "dev"`
	if got != want {
		t.Fatalf("got:\n%s\nwant exactly:\n%s", got, want)
	}
}

// A value in .env.example wins over the default, as it does once copied to
// .env; an empty value falls back to the default for :- only.
func TestBuiltImageTagFromEnvExample(t *testing.T) {
	dir := fixture(t, "built")
	writeFile(t, dir, ".env.example", "# sample\nNAVA_IMAGE_TAG=\"v9\"\n")
	got := run(t, dir)
	for _, want := range []string{
		`compose.yaml:3: image nava/keycloak:v9, but docs/pins.md pins "dev"`,
		`compose.yaml:5: image nava/nava-core:v9, but docs/pins.md pins "dev"`,
		`deploy/compose/compose.ci.yaml:4: image nava/keycloak:v9, but docs/pins.md pins "dev"`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}

	writeFile(t, dir, ".env.example", "NAVA_IMAGE_TAG=\n")
	got = run(t, dir)
	want := `compose.yaml:5: image nava/nava-core: has no exact tag`
	if got != want {
		t.Fatalf("NAVA_IMAGE_TAG empty: got:\n%s\nwant exactly:\n%s", got, want)
	}
}

func TestBuiltImageWithoutRowOrValueFails(t *testing.T) {
	dir := fixture(t, "built")
	replaceIn(t, dir, "compose.yaml", "nava/nava-core:", "nava/nava-purge:")
	replaceIn(t, dir, "deploy/compose/compose.ci.yaml", "${NAVA_IMAGE_TAG:-dev}", "${NAVA_KC_TAG}")
	got := run(t, dir)
	want := `compose.yaml:5: image nava/nava-purge has no row in docs/pins.md` + "\n" +
		`deploy/compose/compose.ci.yaml:4: image nava/keycloak:${NAVA_KC_TAG}: ${NAVA_KC_TAG} has no value in .env.example and no default`
	if got != want {
		t.Fatalf("got:\n%s\nwant exactly:\n%s", got, want)
	}
}

// The TICK-193 acceptance probe on the real files: bumping the nava/keycloak
// tag in compose.yaml without touching docs/pins.md fails the check.
func TestRepositoryKeycloakTagDriftFails(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	dir := t.TempDir()
	for _, rel := range []string{"compose.yaml", ".env.example", "docs/pins.md"} {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(dir, filepath.Dir(filepath.FromSlash(rel))), 0o700); err != nil {
			t.Fatal(err)
		}
		writeFile(t, dir, rel, string(data))
	}
	if got, err := check(dir, filepath.Join(dir, "docs", "pins.md")); err != nil || len(got) > 0 {
		t.Fatalf("unchanged copy: problems %q, err %v", got, err)
	}
	replaceIn(t, dir, "compose.yaml", "nava/keycloak:${NAVA_IMAGE_TAG:-dev}", "nava/keycloak:${NAVA_IMAGE_TAG:-26.8.1}")
	got, err := check(dir, filepath.Join(dir, "docs", "pins.md"))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || !strings.Contains(got[0], "image nava/keycloak:26.8.1, but docs/pins.md pins") ||
		!strings.HasPrefix(got[0], "compose.yaml:") {
		t.Fatalf("want one drift problem on compose.yaml, got %q", got)
	}
}

func TestResolveRef(t *testing.T) {
	env := map[string]string{"SET": "v1", "EMPTY": ""}
	for _, tc := range []struct {
		ref, want string
		ok        bool
	}{
		{"nava/x:${SET:-dev}", "nava/x:v1", true},
		{"nava/x:${UNSET:-dev}", "nava/x:dev", true},
		{"nava/x:${EMPTY:-dev}", "nava/x:dev", true},
		{"nava/x:${EMPTY-dev}", "nava/x:", true},
		{"nava/x:${UNSET-dev}", "nava/x:dev", true},
		{"nava/x:${SET}", "nava/x:v1", true},
		{"nava/x:$SET", "nava/x:v1", true},
		{"nava/x:${UNSET}", "", false},
		{"nava/x:${UNSET:?required}", "", false},
	} {
		got, err := resolveRef(tc.ref, env)
		if (err == nil) != tc.ok || got != tc.want {
			t.Errorf("resolveRef(%q) = %q, %v; want %q, ok %v", tc.ref, got, err, tc.want, tc.ok)
		}
	}
}

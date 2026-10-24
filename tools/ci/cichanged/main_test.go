package main

import (
	"os"
	"strings"
	"testing"
)

func set(s ...string) map[string]bool {
	m := map[string]bool{}
	for _, v := range s {
		m[v] = true
	}
	return m
}

func testGraph() graph {
	return graph{
		packages:    set("internal/db/roles", "internal/db", "internal/server", "tools/ci/checkpins", "db", "internal/platform/telemetry"),
		integration: set("internal/db/roles", "db", "internal/platform/telemetry"),
		stack:       set("internal/server", "db", "internal/platform/telemetry"),
	}
}

func TestClassify(t *testing.T) {
	for _, c := range []struct {
		name  string
		files []string
		want  string
	}{
		{"nothing changed", nil, "ci-lint"},
		{"docs only", []string{"docs/quality/ci-pipeline.md", "README.md", "docs/adr/ADR-001-explained.html"}, "ci-lint"},
		{"pins pick the images", []string{"docs/pins.md"}, "ci-lint ci-unit ci-integration ci-stack"},
		{"unit-only package", []string{"tools/ci/checkpins/main.go"}, "ci-lint ci-unit"},
		{"integration package", []string{"internal/db/roles/roles.go"}, "ci-lint ci-unit ci-integration"},
		{"stack package", []string{"internal/server/server.go"}, "ci-lint ci-unit ci-stack"},
		{"both", []string{"internal/platform/telemetry/filter.go"}, "ci-lint ci-unit ci-integration ci-stack"},
		{"embedded file in a package subdirectory", []string{"db/migrations/00009_x.sql"}, "ci-lint ci-unit ci-integration ci-stack"},
		{"deepest package wins", []string{"internal/db/roles/testdata/x.txt"}, "ci-lint ci-unit ci-integration"},
		{"parent package", []string{"internal/db/pool.go"}, "ci-lint ci-unit"},
		{"compose, which unit tests read too", []string{"deploy/compose/keycloak/realm.json", "compose.yaml"}, "ci-lint ci-unit ci-stack"},
		{"non-documentation file under docs", []string{"docs/delivery/review-paths.yaml"}, "ci-lint ci-unit ci-integration ci-stack"},
		{"testdata markdown", []string{"tools/ci/checkpins/testdata/pins.md"}, "ci-lint ci-unit"},
		{"markdown in a package", []string{"internal/server/NOTES.md"}, "ci-lint ci-unit ci-stack"},
		{"temporal-server module", []string{"temporal-server/authz/authz.go"}, "ci-lint ci-unit ci-stack"},
		{"navalint module", []string{"tools/navalint/setconfig/setconfig.go"}, "ci-lint ci-unit"},
		{"root dependencies", []string{"go.sum"}, "ci-lint ci-unit ci-integration ci-stack"},
		{"CI definition", []string{"Makefile"}, "ci-lint ci-unit ci-integration ci-stack"},
		{"integration script", []string{"scripts/ci/test-integration.sh", "scripts/ci/pgbouncer/pooled.ini"}, "ci-lint ci-integration"},
		{"lint script", []string{"scripts/ci/vulncheck.sh"}, "ci-lint"},
		{"other CI script", []string{"scripts/ci/ci-isolated.sh"}, "ci-lint ci-unit ci-integration ci-stack"},
		{"unknown directory", []string{"newdir/thing.yaml"}, "ci-lint ci-unit ci-integration ci-stack"},
	} {
		t.Run(c.name, func(t *testing.T) {
			got, reasons := classify(c.files, testGraph())
			if strings.Join(got, " ") != c.want {
				t.Fatalf("stages = %q, want %q\n%s", strings.Join(got, " "), c.want, strings.Join(reasons, "\n"))
			}
			if len(reasons) != len(c.files) {
				t.Fatalf("%d reasons for %d files", len(reasons), len(c.files))
			}
		})
	}
}

// The real graph: the integration stage covers the db packages,
// the stack covers nava-core's server, and a CI tool is unit-only.
func TestLoadGraph(t *testing.T) {
	g, err := loadGraph()
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		pkg                string
		integration, stack bool
	}{
		{"internal/db/roles", true, true},
		{"internal/db/dbtest", true, false},
		{"internal/server", false, true},
		{"internal/temporalstack", false, true},
		{"tools/ci/checkpins", false, false},
	} {
		if !g.packages[c.pkg] {
			t.Errorf("%s: not a package", c.pkg)
			continue
		}
		if g.integration[c.pkg] != c.integration || g.stack[c.pkg] != c.stack {
			t.Errorf("%s: integration %v stack %v, want %v %v", c.pkg, g.integration[c.pkg], g.stack[c.pkg], c.integration, c.stack)
		}
	}
}

// A file moved out of a package counts at its old path too.
func TestChangedFilesSeesBothSidesOfARename(t *testing.T) {
	t.Chdir(t.TempDir())
	run := func(args ...string) {
		t.Helper()
		if _, err := git(args...); err != nil {
			t.Fatal(err)
		}
	}
	run("init", "-q", "-b", "base")
	run("config", "user.email", "ci@example.invalid")
	run("config", "user.name", "ci")
	if err := os.MkdirAll("internal/server", 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("internal/server/notes.go", []byte("package server\n\n// A long enough body for rename detection.\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	run("add", ".")
	run("commit", "-q", "-m", "base")
	run("checkout", "-q", "-b", "change")
	if err := os.MkdirAll("docs", 0o750); err != nil {
		t.Fatal(err)
	}
	run("mv", "internal/server/notes.go", "docs/notes.md")
	run("commit", "-q", "-m", "move")
	files, err := changedFiles("base")
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(files, " "); got != "docs/notes.md internal/server/notes.go" {
		t.Fatalf("changed files = %q, want both paths", got)
	}
}

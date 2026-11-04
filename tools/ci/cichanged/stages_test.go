package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// stagesClone clones the repository's HEAD (sharing its objects) onto a new
// branch, applies edit there and commits it, and returns the clone and the
// base commit. scripts/ci/ci-stages.sh then sees exactly edit as the change.
func stagesClone(t *testing.T, edit func(dir string)) (string, string) {
	t.Helper()
	root, err := git("rev-parse", "--show-toplevel")
	if err != nil {
		t.Fatal(err)
	}
	root = strings.TrimSpace(root)
	base, err := git("rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	base = strings.TrimSpace(base)
	dir := filepath.Join(t.TempDir(), "repo")
	if _, err := git("clone", "-q", "--shared", "--no-checkout", root, dir); err != nil {
		t.Fatal(err)
	}
	run := func(args ...string) {
		t.Helper()
		if _, err := git(append([]string{"-C", dir}, args...)...); err != nil {
			t.Fatal(err)
		}
	}
	run("checkout", "-q", "-b", "change", base)
	run("config", "user.email", "ci@example.invalid")
	run("config", "user.name", "ci")
	edit(dir)
	run("add", "-A")
	run("commit", "-q", "-m", "change")
	return dir, base
}

// ciStages runs this checkout's scripts/ci/ci-stages.sh (the recipe of make
// ci-stages) in dir with env, and returns the GITHUB_OUTPUT it wrote.
func ciStages(t *testing.T, dir, base string, env ...string) string {
	t.Helper()
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("no sh on PATH")
	}
	root, err := git("rev-parse", "--show-toplevel")
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "github-output")
	script := filepath.Join(strings.TrimSpace(root), "scripts", "ci", "ci-stages.sh")
	cmd := exec.CommandContext(context.Background(), sh, script)
	cmd.Dir = dir
	// Later entries win, so the runner's own GITHUB_OUTPUT, CI_ALL or
	// CI_LINT_ONLY never leak in.
	cmd.Env = append(os.Environ(), append([]string{"BASE=" + base, "GITHUB_OUTPUT=" + out, "CI_ALL=", "CI_LINT_ONLY="}, env...)...)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	if _, err := cmd.Output(); err != nil {
		t.Fatalf("ci-stages.sh: %v\n%s", err, stderr.String())
	}
	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func writeFiles(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for name, body := range files {
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

const allFalse = "unit=false\nintegration=false\nstack=false\n"

// TICK-563: a pull request whose diff is only documentation under docs/ runs
// lint alone, even with the ci:run label: unit, integration and stack are
// false in the outputs the workflow's job conditions read.
func TestDocsOnlyDiffRunsNoUnitIntegrationOrStack(t *testing.T) {
	t.Parallel()
	dir, base := stagesClone(t, func(dir string) {
		writeFiles(t, dir, map[string]string{
			"docs/quality/tick-563-test-plan.md":    "# plan\n",
			"docs/reviews/tick-563-review.md":       "# review\n",
			"docs/adr/ADR-999-tick-563.html":        "<p>adr</p>\n",
			"docs/delivery/tick-563-decision.md":    "decision\n",
			"docs/acceptance/S99/tick-563-notes.md": "notes\n",
		})
	})
	if got := ciStages(t, dir, base); got != allFalse {
		t.Fatalf("GITHUB_OUTPUT = %q, want %q", got, allFalse)
	}
}

// TICK-563: a pull request without the ci:run label (CI_LINT_ONLY=1) runs
// lint alone even for a Go change that, labelled, needs every stage.
func TestLintOnlyGatesAGoChange(t *testing.T) {
	t.Parallel()
	dir, base := stagesClone(t, func(dir string) {
		p := filepath.Join(dir, "cmd", "nava-core", "main.go")
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, append(b, []byte("\n// TICK-563 test change.\n")...), 0o600); err != nil {
			t.Fatal(err)
		}
	})
	if got := ciStages(t, dir, base, "CI_LINT_ONLY=1"); got != allFalse {
		t.Errorf("without ci:run: GITHUB_OUTPUT = %q, want %q", got, allFalse)
	}
	if got, want := ciStages(t, dir, base), "unit=true\nintegration=true\nstack=true\n"; got != want {
		t.Errorf("with ci:run: GITHUB_OUTPUT = %q, want %q", got, want)
	}
}

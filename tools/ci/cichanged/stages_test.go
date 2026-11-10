package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// stagesClone clones the repository's HEAD (sharing its objects) and commits
// this checkout's own cichanged sources over it as the base (so the clone
// runs the code under test, not HEAD's), and returns the clone and the base
// commit. Every ci-stages.sh scenario reuses this one clone through
// stagesChange: go list's first pass over a freshly written tree costs about
// twice a later one, and each scenario runs cichanged, so a clone per
// scenario helped push this package past go test's 10-minute timeout under
// task test (TICK-597).
func stagesClone(t *testing.T) (string, string) {
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
	gitIn(t, dir, "checkout", "-q", "-b", "base", base)
	gitIn(t, dir, "config", "user.email", "ci@example.invalid")
	gitIn(t, dir, "config", "user.name", "ci")
	srcs, err := filepath.Glob("*.go")
	if err != nil || len(srcs) == 0 {
		t.Fatalf("no cichanged sources: %v", err)
	}
	for _, f := range srcs {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		writeFiles(t, dir, map[string]string{"tools/ci/cichanged/" + f: string(b)})
	}
	gitIn(t, dir, "add", "-A")
	gitIn(t, dir, "commit", "-q", "--allow-empty", "-m", "base")
	out, err := git("-C", dir, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	return dir, strings.TrimSpace(out)
}

// stagesChange resets the clone's change branch to base, applies edit and
// commits it, so scripts/ci/ci-stages.sh then sees exactly edit as the
// change, whatever an earlier scenario committed.
func stagesChange(t *testing.T, dir, base string, edit func(dir string)) {
	t.Helper()
	gitIn(t, dir, "checkout", "-q", "-B", "change", base)
	gitIn(t, dir, "reset", "-q", "--hard", base)
	gitIn(t, dir, "clean", "-q", "-fdx")
	edit(dir)
	gitIn(t, dir, "add", "-A")
	gitIn(t, dir, "commit", "-q", "-m", "change")
}

func gitIn(t *testing.T, dir string, args ...string) {
	t.Helper()
	if _, err := git(append([]string{"-C", dir}, args...)...); err != nil {
		t.Fatal(err)
	}
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
	cmd.Env = append(os.Environ(), append([]string{"BASE=" + base, "GITHUB_OUTPUT=" + out, "CI_ALL=", "CI_FULL=", "CI_LINT_ONLY=", "CI_AREAS_ALL=", "CI_IMAGES="}, env...)...)
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

const allFalse = "unit=false\nintegration=false\nstack=false\nareas=none\nimages=real\n"

// The ci-stages.sh scenarios share one clone and run one after another
// (TICK-597): each runs cichanged, whose go list passes over the whole module
// only compete with each other when run in parallel.
func TestCIStages(t *testing.T) {
	dir, base := stagesClone(t)
	t.Run("docs-only diff runs no unit, integration or stack", func(t *testing.T) {
		testDocsOnlyDiffRunsNoUnitIntegrationOrStack(t, dir, base)
	})
	t.Run("lint-only gates a Go change", func(t *testing.T) {
		testLintOnlyGatesAGoChange(t, dir, base)
	})
	t.Run("a Dockerfile change builds the real images", func(t *testing.T) {
		testDockerfileChangeBuildsRealImages(t, dir, base)
	})
}

// TICK-628: a change to deploy/compose/nava-core/Dockerfile builds the real
// Dockerfiles in the stack job, even beside Go changes that alone would use
// the thin CI images, and main (CI_ALL=1) is always real, whatever a manual
// run's images input says.
func testDockerfileChangeBuildsRealImages(t *testing.T, dir, base string) {
	stagesChange(t, dir, base, func(dir string) {
		for f, line := range map[string]string{
			"deploy/compose/nava-core/Dockerfile": "# TICK-628 test change.",
			"cmd/nava-core/main.go":               "// TICK-628 test change.",
		} {
			p := filepath.Join(dir, filepath.FromSlash(f))
			b, err := os.ReadFile(p)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(p, append(b, []byte("\n"+line+"\n")...), 0o600); err != nil {
				t.Fatal(err)
			}
		}
	})
	images := func(env ...string) string {
		// CI_AREAS_ALL=1 skips the cichanged -areas pass, which this
		// scenario does not check, to stay inside go test's 10 minutes
		// under task test (TICK-597). CI_ALL=1 runs no cichanged at all.
		out := ciStages(t, dir, base, append([]string{"CI_AREAS_ALL=1"}, env...)...)
		for _, l := range strings.Split(out, "\n") {
			if v, ok := strings.CutPrefix(l, "images="); ok {
				return v
			}
		}
		t.Fatalf("GITHUB_OUTPUT has no images=: %q", out)
		return ""
	}
	if got := images(); got != "real" {
		t.Errorf("Dockerfile change: images=%s, want real", got)
	}
	if got := images("CI_ALL=1", "CI_IMAGES=thin"); got != "real" {
		t.Errorf("CI_ALL=1, CI_IMAGES=thin: images=%s, want real", got)
	}
}

// TICK-563: a pull request whose diff is only documentation under docs/ runs
// lint alone, even with the ci:run label: unit, integration and stack are
// false in the outputs the workflow's job conditions read.
func testDocsOnlyDiffRunsNoUnitIntegrationOrStack(t *testing.T, dir, base string) {
	stagesChange(t, dir, base, func(dir string) {
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
	// TICK-638: a pull request into a story branch (CI_FULL=1) runs every
	// stage and every stack-case area whatever its diff, even this docs-only
	// one, and wins over CI_LINT_ONLY. The images still follow the rule: no
	// Dockerfile changed, so thin. Checked on this diff to save a cichanged
	// pass (TICK-597).
	const full = "unit=true\nintegration=true\nstack=true\nareas=all\nimages=thin\n"
	if got := ciStages(t, dir, base, "CI_FULL=1"); got != full {
		t.Errorf("CI_FULL=1: GITHUB_OUTPUT = %q, want %q", got, full)
	}
	if got := ciStages(t, dir, base, "CI_FULL=1", "CI_LINT_ONLY=1"); got != full {
		t.Errorf("CI_FULL=1 CI_LINT_ONLY=1: GITHUB_OUTPUT = %q, want %q", got, full)
	}
	// TICK-652: a push to main diffs against github.event.before (BASE, a
	// sha); this docs-only diff then runs lint alone, as above. A zero
	// before (a new ref) or one this clone lacks (a force push) has no diff,
	// so the push runs as CI_ALL=1: every stage and area, real images.
	const all = "unit=true\nintegration=true\nstack=true\nareas=all\nimages=real\n"
	for _, b := range []string{"0000000000000000000000000000000000000000", "0123456789abcdef0123456789abcdef01234567"} {
		if got := ciStages(t, dir, b); got != all {
			t.Errorf("BASE=%s: GITHUB_OUTPUT = %q, want %q", b, got, all)
		}
	}
}

// TICK-563: a pull request without the ci:run label (CI_LINT_ONLY=1) runs
// lint alone even for a Go change that, labelled, needs every stage.
func testLintOnlyGatesAGoChange(t *testing.T, dir, base string) {
	stagesChange(t, dir, base, func(dir string) {
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
	// cmd/nava-core is in no stack-case area, so the keycloak, pgbouncer and
	// db cases are skipped, unless ci:full (or a PR into main) sets
	// CI_AREAS_ALL.
	if got, want := ciStages(t, dir, base), "unit=true\nintegration=true\nstack=true\nareas=none\nimages=thin\n"; got != want {
		t.Errorf("with ci:run: GITHUB_OUTPUT = %q, want %q", got, want)
	}
	// TICK-628: a manual run's images input (CI_IMAGES) overrides the rule;
	// checked here, beside CI_AREAS_ALL, to save a cichanged pass (TICK-597).
	if got, want := ciStages(t, dir, base, "CI_AREAS_ALL=1", "CI_IMAGES=real"), "unit=true\nintegration=true\nstack=true\nareas=all\nimages=real\n"; got != want {
		t.Errorf("with ci:full and CI_IMAGES=real: GITHUB_OUTPUT = %q, want %q", got, want)
	}
}

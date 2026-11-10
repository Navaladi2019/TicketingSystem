package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// tsrvChanged runs this checkout's scripts/ci/temporal-server-changed.sh (the
// recipe of make ci-temporal-server-changed) in dir with env, and returns the
// GITHUB_OUTPUT it wrote.
func tsrvChanged(t *testing.T, dir, base string, env ...string) string {
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
	script := filepath.Join(strings.TrimSpace(root), "scripts", "ci", "temporal-server-changed.sh")
	cmd := exec.CommandContext(context.Background(), sh, script)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), append([]string{"BASE=" + base, "GITHUB_OUTPUT=" + out, "CI_ALL=", "CI_FULL=", "CI_LINT_ONLY=", "PUSH_BEFORE="}, env...)...)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	if _, err := cmd.Output(); err != nil {
		t.Fatalf("temporal-server-changed.sh: %v\n%s", err, stderr.String())
	}
	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(b))
}

// TICK-641, TICK-695: the temporal-server job runs only when the module, how
// it is built, tested or linted, or a pinned golangci-lint or Go version
// changed. A full run (CI_FULL) always runs it, and a pull request without
// ci:run never does. On a push to main the push's own commits decide, and a
// push base the checkout lacks runs it.
func TestTemporalServerChanged(t *testing.T) {
	dir, base := stagesClone(t)
	const yes, no = "temporal_server=true", "temporal_server=false"
	push := []string{"CI_ALL=1", "PUSH_BEFORE=" + base}
	lintOnly := []string{"CI_LINT_ONLY=1"}

	appendTo := func(f string) func(string) {
		return func(dir string) {
			appendLine(t, filepath.Join(dir, filepath.FromSlash(f)), "# TICK-695 test change.")
		}
	}
	replaceIn := func(f, old, repl string) func(string) {
		return func(dir string) {
			replaceOnce(t, filepath.Join(dir, filepath.FromSlash(f)), old, repl)
		}
	}
	const pin = "GOLANGCI_LINT_VERSION := "
	type check struct {
		env  []string
		want string
	}
	pr := func(want string) []check { return []check{{nil, want}} }
	// Each edit is committed once; every check runs against that commit.
	for _, c := range []struct {
		name   string
		edit   func(dir string)
		checks []check
	}{
		// What decides, whatever the change.
		{"root Go change", appendTo("cmd/nava-core/main.go"), []check{
			{nil, no},
			{[]string{"CI_FULL=1", "CI_LINT_ONLY=1"}, yes}, // full run
			{[]string{"CI_ALL=1"}, yes},                    // manual run on main
			{push, no},
			{[]string{"CI_ALL=1", "PUSH_BEFORE=0000000000000000000000000000000000000000"}, yes}, // first push
			{[]string{"CI_ALL=1", "PUSH_BEFORE=1111111111111111111111111111111111111111"}, yes}, // unknown push base
		}},

		// Whole-file inputs.
		{"module change", appendTo("temporal-server/cmd/server/main.go"), []check{{nil, yes}, {push, yes}, {lintOnly, no}}},
		{"make/temporal-server.mk", appendTo("make/temporal-server.mk"), pr(yes)},
		{".golangci.yml", appendTo(".golangci.yml"), []check{{nil, yes}, {push, yes}, {lintOnly, no}}},
		{"tools/ci/suiteran", appendTo("tools/ci/suiteran/main.go"), pr(yes)},
		{"the script", appendTo("scripts/ci/temporal-server-changed.sh"), pr(yes)},

		// Pin lines only (TICK-695).
		{"golangci-lint pin bump", replaceIn("Makefile", pin, pin+"9"), []check{{nil, yes}, {push, yes}, {lintOnly, no}}},
		{"golangci-lint install line", replaceIn("make/lint.mk", "golangci-lint@v$(GOLANGCI_LINT_VERSION)", "golangci-lint@v$(GOLANGCI_LINT_VERSION) # pinned"), pr(yes)},
		{"other make/lint.mk edit", appendTo("make/lint.mk"), []check{{nil, no}, {push, no}}},
		{"other Makefile edit", appendTo("Makefile"), pr(no)},
		{"pins.md golangci-lint row", replaceIn("docs/pins.md", "| `golangci-lint` | v", "| `golangci-lint` | v9"), pr(yes)},
		{"pins.md Go (temporal-server) row", replaceIn("docs/pins.md", "| Go (`temporal-server`) | `go 1.", "| Go (`temporal-server`) | `go 9."), pr(yes)},
		{"pins.md Go (platform module) row", replaceIn("docs/pins.md", "| Go (platform module) | `go 1.", "| Go (platform module) | `go 9."), pr(no)},
		{"other pins.md edit", appendTo("docs/pins.md"), pr(no)},

		// Not inputs (TICK-695).
		{"ci.yml only", appendTo(".github/workflows/ci.yml"), []check{{nil, no}, {push, no}}},
	} {
		stagesChange(t, dir, base, c.edit)
		for _, k := range c.checks {
			if got := tsrvChanged(t, dir, base, k.env...); got != k.want {
				t.Errorf("%s %v: %s, want %s", c.name, k.env, got, k.want)
			}
		}
	}
}

// TICK-695: a pull request into a TICK-* story branch (BASE=origin/TICK-*,
// with no CI_FULL and no CI_LINT_ONLY, as ci.yml sets them) diffs against the
// story branch, not main, so the story's own module change does not run the
// job for every later task PR. The story -> main PR still runs it.
func TestTemporalServerChangedStoryPR(t *testing.T) {
	dir, base := stagesClone(t)
	const yes, no = "temporal_server=true", "temporal_server=false"

	// The story branch already changed the module.
	stagesChange(t, dir, base, func(dir string) {
		appendLine(t, filepath.Join(dir, "temporal-server", "cmd", "server", "main.go"), "// TICK-695 story change.")
	})
	gitIn(t, dir, "branch", "-f", "story", "HEAD")
	story, err := git("-C", dir, "rev-parse", "story")
	if err != nil {
		t.Fatal(err)
	}
	story = strings.TrimSpace(story)

	for _, c := range []struct {
		name string
		file string
		want string
	}{
		{"task PR without a module change", "cmd/nava-core/main.go", no},
		{"task PR with only a ci.yml change", ".github/workflows/ci.yml", no},
		{"task PR with a module change", "temporal-server/cmd/server/main.go", yes},
	} {
		stagesChange(t, dir, story, func(dir string) {
			appendLine(t, filepath.Join(dir, filepath.FromSlash(c.file)), "# TICK-695 task change.")
		})
		if got := tsrvChanged(t, dir, "story"); got != c.want {
			t.Errorf("%s: %s, want %s", c.name, got, c.want)
		}
	}

	// The story -> main PR sees the story's module change.
	stagesChange(t, dir, story, func(dir string) {
		appendLine(t, filepath.Join(dir, "cmd", "nava-core", "main.go"), "// TICK-695 task change.")
	})
	if got := tsrvChanged(t, dir, base); got != yes {
		t.Errorf("story -> main PR: %s, want %s", got, yes)
	}
}

func appendLine(t *testing.T, p, line string) {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, append(b, []byte("\n"+line+"\n")...), 0o600); err != nil {
		t.Fatal(err)
	}
}

func replaceOnce(t *testing.T, p, old, repl string) {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(string(b), old); n != 1 {
		t.Fatalf("%s: %d of %q, want 1", p, n, old)
	}
	if err := os.WriteFile(p, []byte(strings.Replace(string(b), old, repl, 1)), 0o600); err != nil {
		t.Fatal(err)
	}
}

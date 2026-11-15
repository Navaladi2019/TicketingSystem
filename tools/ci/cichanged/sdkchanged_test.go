package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// sdkChanged runs this checkout's scripts/ci/sdk-changed.sh (the recipe of
// make ci-sdk-changed) in dir with env, and returns the GITHUB_OUTPUT it
// wrote.
func sdkChanged(t *testing.T, dir, base string, env ...string) string {
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
	script := filepath.Join(strings.TrimSpace(root), "scripts", "ci", "sdk-changed.sh")
	cmd := exec.CommandContext(context.Background(), sh, script)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), append([]string{"BASE=" + base, "GITHUB_OUTPUT=" + out, "CI_ALL=", "CI_FULL=", "CI_LINT_ONLY=", "PUSH_BEFORE="}, env...)...)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	if _, err := cmd.Output(); err != nil {
		t.Fatalf("sdk-changed.sh: %v\n%s", err, stderr.String())
	}
	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(b))
}

// TICK-827 (RFC-017 CI-E10): the sdk job runs when api/, sdk/, make/sdk.mk,
// its script or ci.yml changed. A full run always runs it, a pull request
// without ci:run never does, and on a push to main the push's own commits
// decide.
func TestSDKChanged(t *testing.T) {
	t.Parallel()
	dir, base := stagesClone(t)
	const yes, no = "sdk=true", "sdk=false"
	push := []string{"CI_ALL=1", "PUSH_BEFORE=" + base}
	lintOnly := []string{"CI_LINT_ONLY=1"}
	appendTo := func(f string) func(string) {
		return func(dir string) {
			appendLine(t, filepath.Join(dir, filepath.FromSlash(f)), "# TICK-827 test change.")
		}
	}
	type check struct {
		env  []string
		want string
	}
	pr := func(want string) []check { return []check{{nil, want}} }
	for _, c := range []struct {
		name   string
		edit   func(dir string)
		checks []check
	}{
		{"root Go change", appendTo("cmd/nava-core/main.go"), []check{
			{nil, no},
			{[]string{"CI_FULL=1", "CI_LINT_ONLY=1"}, yes},
			{[]string{"CI_ALL=1"}, yes},
			{push, no},
			{[]string{"CI_ALL=1", "PUSH_BEFORE=0000000000000000000000000000000000000000"}, yes},
			{[]string{"CI_ALL=1", "PUSH_BEFORE=1111111111111111111111111111111111111111"}, yes},
		}},
		{"the bundle", appendTo("api/openapi.bundled.yaml"), []check{{nil, yes}, {push, yes}, {lintOnly, no}}},
		{"an API source", appendTo("api/openapi.yaml"), pr(yes)},
		{"the SDK package", appendTo("sdk/ts/package.json"), []check{{nil, yes}, {push, yes}}},
		{"make/sdk.mk", appendTo("make/sdk.mk"), pr(yes)},
		{"the script", appendTo("scripts/ci/sdk-changed.sh"), pr(yes)},
		{"ci.yml", appendTo(".github/workflows/ci.yml"), pr(yes)},
		{"another make fragment", appendTo("make/lint.mk"), pr(no)},
		{"documentation", appendTo("docs/pins.md"), pr(no)},
	} {
		stagesChange(t, dir, base, c.edit)
		for _, k := range c.checks {
			if got := sdkChanged(t, dir, base, k.env...); got != k.want {
				t.Errorf("%s %v: %s, want %s", c.name, k.env, got, k.want)
			}
		}
	}
}

// TICK-827: a pull request into a TICK-* story branch diffs against the story
// branch, so the story's own SDK work does not run the job for a later task
// PR that leaves the API and the SDK alone. The story -> main PR still runs it.
func TestSDKChangedStoryPR(t *testing.T) {
	t.Parallel()
	dir, base := stagesClone(t)
	const yes, no = "sdk=true", "sdk=false"

	stagesChange(t, dir, base, func(dir string) {
		appendLine(t, filepath.Join(dir, "sdk", "ts", "package.json"), "")
	})
	gitIn(t, dir, "branch", "-f", "story", "HEAD")
	story, err := git("-C", dir, "rev-parse", "story")
	if err != nil {
		t.Fatal(err)
	}
	story = strings.TrimSpace(story)

	for _, c := range []struct {
		name, file, want string
	}{
		{"task PR without an API or SDK change", "cmd/nava-core/main.go", no},
		{"task PR with an SDK change", "sdk/ts/src/index.ts", yes},
		{"task PR with a bundle change", "api/openapi.bundled.yaml", yes},
	} {
		stagesChange(t, dir, story, func(dir string) {
			appendLine(t, filepath.Join(dir, filepath.FromSlash(c.file)), "// TICK-827 task change.")
		})
		if got := sdkChanged(t, dir, "story"); got != c.want {
			t.Errorf("%s: %s, want %s", c.name, got, c.want)
		}
	}

	stagesChange(t, dir, story, func(dir string) {
		appendLine(t, filepath.Join(dir, "cmd", "nava-core", "main.go"), "// TICK-827 task change.")
	})
	if got := sdkChanged(t, dir, base); got != yes {
		t.Errorf("story -> main PR: %s, want %s", got, yes)
	}
}

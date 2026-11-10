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

// TICK-641: the temporal-server job runs only when the module, or how it is
// built, tested or linted, changed; a story PR (CI_FULL) always runs it, and
// a pull request without ci:run never does. On a push to main the push's own
// commits decide, and a push base the checkout lacks runs it.
func TestTemporalServerChanged(t *testing.T) {
	dir, base := stagesClone(t)
	const yes, no = "temporal_server=true", "temporal_server=false"

	stagesChange(t, dir, base, func(dir string) {
		appendLine(t, filepath.Join(dir, "cmd", "nava-core", "main.go"), "// TICK-641 test change.")
	})
	for _, c := range []struct {
		name string
		env  []string
		want string
	}{
		{"root Go change", nil, no},
		{"story PR", []string{"CI_FULL=1", "CI_LINT_ONLY=1"}, yes},
		{"manual run on main", []string{"CI_ALL=1"}, yes},
		{"push to main", []string{"CI_ALL=1", "PUSH_BEFORE=" + base}, no},
		{"first push", []string{"CI_ALL=1", "PUSH_BEFORE=0000000000000000000000000000000000000000"}, yes},
		{"unknown push base", []string{"CI_ALL=1", "PUSH_BEFORE=1111111111111111111111111111111111111111"}, yes},
	} {
		if got := tsrvChanged(t, dir, base, c.env...); got != c.want {
			t.Errorf("%s: %s, want %s", c.name, got, c.want)
		}
	}

	for _, f := range []string{"temporal-server/cmd/server/main.go", "make/temporal-server.mk", ".golangci.yml", "docs/pins.md"} {
		stagesChange(t, dir, base, func(dir string) {
			appendLine(t, filepath.Join(dir, filepath.FromSlash(f)), "# TICK-641 test change.")
		})
		if got := tsrvChanged(t, dir, base); got != yes {
			t.Errorf("%s changed: %s, want %s", f, got, yes)
		}
		if got := tsrvChanged(t, dir, base, "CI_ALL=1", "PUSH_BEFORE="+base); got != yes {
			t.Errorf("%s pushed to main: %s, want %s", f, got, yes)
		}
		if got := tsrvChanged(t, dir, base, "CI_LINT_ONLY=1"); got != no {
			t.Errorf("%s without ci:run: %s, want %s", f, got, no)
		}
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

package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TICK-641: scripts/ci/runner.sh (make ci-runner, the ci workflow's runner
// job) picks the runner every job uses. The manual run's input wins over the
// CI_RUNNER variable, an unknown value fails, and a pull request from a fork
// never runs on the self-hosted runner, whatever the input or variable say.
func TestRunner(t *testing.T) {
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("no sh on PATH")
	}
	root, err := git("rev-parse", "--show-toplevel")
	if err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(strings.TrimSpace(root), "scripts", "ci", "runner.sh")
	const (
		github     = "runs_on=\"ubuntu-latest\"\nself_hosted=false\n"
		selfHosted = "runs_on=[\"self-hosted\",\"wsl\"]\nself_hosted=true\n"
	)
	for _, tc := range []struct {
		name                   string
		input, variable, cross string
		want                   string // the $GITHUB_OUTPUT lines; "" wants a failure
	}{
		{"no input, no variable: github", "", "", "", github},
		{"variable github", "", "github", "", github},
		{"variable self-hosted", "", "self-hosted", "", selfHosted},
		{"input self-hosted over variable github", "self-hosted", "github", "", selfHosted},
		{"input github over variable self-hosted", "github", "self-hosted", "", github},
		{"fork pull request, variable self-hosted", "", "self-hosted", "true", github},
		{"fork pull request, input self-hosted", "self-hosted", "", "true", github},
		{"same-repository pull request, variable self-hosted", "", "self-hosted", "false", selfHosted},
		{"unknown variable", "", "wsl", "", ""},
		{"unknown input", "Self-Hosted", "", "", ""},
		{"unknown variable on a fork still fails", "", "wsl", "true", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out := filepath.Join(t.TempDir(), "output")
			cmd := exec.CommandContext(context.Background(), sh, script)
			cmd.Env = append(os.Environ(), "GITHUB_OUTPUT="+out,
				"RUNNER_INPUT="+tc.input, "RUNNER_VAR="+tc.variable, "CROSS_REPO="+tc.cross)
			log, err := cmd.CombinedOutput()
			if tc.want == "" {
				if err == nil {
					t.Fatalf("want failure, got success\n%s", log)
				}
				return
			}
			if err != nil {
				t.Fatalf("want success, got %v\n%s", err, log)
			}
			got, err := os.ReadFile(out)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tc.want {
				t.Errorf("outputs:\n%s\nwant:\n%s", got, tc.want)
			}
		})
	}
}

package main

// The -copy mode (TICK-1149, ADR-001-E4 vulncheck contract rule 9): the whole
// of a workflow, comment lines aside, must equal a checked-in copy byte for
// byte, and the two vulncheck make recipes must be the pinned lines. An edit
// of one file alone (a step if:, an exit 0, a shell:, a paths filter, a
// top-level env: or defaults:, a second job reporting the gate's name, a
// `|| true` in a recipe) fails lint. A deliberate edit of the workflow and
// the copy together passes: that is ADR RS3, caught by review, not by this.

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// pinnedRecipe is a make target the govulncheck workflow runs and the exact
// recipe `make -n --no-print-directory <target>` must print.
type pinnedRecipe struct{ target, recipe string }

// pinnedRecipes are contract rule 9's two recipes, with rule 7's GOWORK=off.
func pinnedRecipes() []pinnedRecipe {
	return []pinnedRecipe{
		{"vulncheck", "GOWORK=off govulncheck ./..."},
		{"vulncheck-temporal-server", "cd temporal-server && GOWORK=off govulncheck ./..."},
	}
}

// makeRunner returns what `make -n` prints for target.
type makeRunner func(target string) ([]byte, error)

// makeIn runs `make -n --no-print-directory target` in dir ("" is the current
// directory), with stdout and stderr together so a warning counts as output.
// A parent make's MAKEFLAGS (jobserver, -k, -s) and MAKELEVEL are dropped: the
// dry run must print the same lines under `make check-ci-steps` as by hand.
func makeIn(dir string) makeRunner {
	return func(target string) ([]byte, error) {
		args := []string{"-n", "--no-print-directory"}
		if dir != "" {
			args = append(args, "-C", dir)
		}
		cmd := exec.CommandContext(context.Background(), "make", append(args, target)...) //nolint:gosec // target is one of pinnedRecipes, dir the caller's
		for _, kv := range os.Environ() {
			k, _, _ := strings.Cut(kv, "=")
			if k != "MAKEFLAGS" && k != "MFLAGS" && k != "MAKELEVEL" {
				cmd.Env = append(cmd.Env, kv)
			}
		}
		return cmd.CombinedOutput()
	}
}

// checkCopy returns the problems of workflowPath against copyPath and of the
// pinned recipes, or an error when a file or make cannot be read.
func checkCopy(workflowPath, copyPath string, mk makeRunner) ([]string, error) {
	wf, err := os.ReadFile(workflowPath) //nolint:gosec // a path the caller chose
	if err != nil {
		return nil, err
	}
	cp, err := os.ReadFile(copyPath) //nolint:gosec // a path the caller chose
	if err != nil {
		return nil, err
	}
	var problems []string
	if p := compareUncommented(wf, cp); p != "" {
		problems = append(problems, fmt.Sprintf("%s differs from %s, comment lines aside: %s", workflowPath, copyPath, p))
	}
	for _, r := range pinnedRecipes() {
		out, err := mk(r.target)
		if err != nil {
			return nil, fmt.Errorf("make -n %s: %w: %s", r.target, err, strings.TrimSpace(string(out)))
		}
		got := strings.TrimSuffix(strings.ReplaceAll(string(out), "\r\n", "\n"), "\n")
		if got != r.recipe {
			problems = append(problems, fmt.Sprintf("make -n %s prints %q, want exactly %q", r.target, got, r.recipe))
		}
	}
	return problems, nil
}

// numbered is a line of a file and its 1-based line number.
type numbered struct {
	n    int
	text []byte
}

// uncommented returns the lines of data whose first non-blank character is
// not #. Every other byte counts: blank lines, indentation, trailing comments
// and line endings.
func uncommented(data []byte) []numbered {
	var out []numbered
	for i, line := range bytes.SplitAfter(data, []byte("\n")) {
		if len(line) == 0 {
			continue
		}
		if t := bytes.TrimLeft(line, " \t"); len(t) > 0 && t[0] == '#' {
			continue
		}
		out = append(out, numbered{i + 1, line})
	}
	return out
}

// compareUncommented returns "" when a and b are equal once comment lines are
// dropped, else the first differing line of each.
func compareUncommented(a, b []byte) string {
	la, lb := uncommented(a), uncommented(b)
	for i := 0; ; i++ {
		switch {
		case i >= len(la) && i >= len(lb):
			return ""
		case i >= len(la):
			return fmt.Sprintf("workflow ends, copy line %d is %q", lb[i].n, lb[i].text)
		case i >= len(lb):
			return fmt.Sprintf("workflow line %d is %q, copy ends", la[i].n, la[i].text)
		case !bytes.Equal(la[i].text, lb[i].text):
			return fmt.Sprintf("workflow line %d is %q, copy line %d is %q", la[i].n, la[i].text, lb[i].n, lb[i].text)
		}
	}
}

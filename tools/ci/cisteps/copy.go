package main

// The -copy mode (TICK-1149, ADR-001-E4 vulncheck contract rule 9): the whole
// of a workflow, comment lines aside, must equal a checked-in copy byte for
// byte, and the two vulncheck make recipes must be the pinned lines. An edit
// of one file alone (a step if:, an exit 0, a shell:, a paths filter, a
// top-level env: or defaults:, a second job reporting the gate's name, a
// `|| true` or a - prefix in a recipe, .IGNORE, a changed SHELL) fails lint.
// A deliberate edit of the workflow and the copy together passes: that is
// ADR RS3, caught by review, not by this.

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// pinnedRecipe is a make target the govulncheck workflow runs and the exact
// recipe `make -n --no-print-directory <target>` must print. Its rule in
// make's database must be `<target>:` with that one recipe line.
type pinnedRecipe struct{ target, recipe string }

// pinnedRecipes are contract rule 9's two recipes, with rule 7's GOWORK=off.
func pinnedRecipes() []pinnedRecipe {
	return []pinnedRecipe{
		{"vulncheck", "GOWORK=off govulncheck ./..."},
		{"vulncheck-temporal-server", "cd temporal-server && GOWORK=off govulncheck ./..."},
	}
}

// makeRunner runs make with args and returns stdout and stderr together.
type makeRunner func(args ...string) ([]byte, error)

// makeIn runs make in dir ("" is the current directory), with stdout and
// stderr together so a warning counts as output. A parent make's MAKEFLAGS
// (jobserver, -k, -s) and MAKELEVEL are dropped: make must print the same
// lines under `make check-ci-steps` as by hand.
func makeIn(dir string) makeRunner {
	return func(args ...string) ([]byte, error) {
		if dir != "" {
			args = append([]string{"-C", dir}, args...)
		}
		cmd := exec.CommandContext(context.Background(), "make", args...) //nolint:gosec // targets are pinnedRecipes, dir the caller's
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
	dbProblems, bad, err := checkMakeDatabase(mk)
	if err != nil {
		return nil, err
	}
	problems = append(problems, dbProblems...)
	for _, r := range pinnedRecipes() {
		// make runs a +-prefixed recipe line even under -n: dry-run only a
		// rule whose recipe in the database is the pinned line.
		if bad[r.target] {
			continue
		}
		out, err := mk("-n", "--no-print-directory", r.target)
		if err != nil {
			return nil, fmt.Errorf("make -n %s: %w: %s", r.target, err, strings.TrimSpace(string(out)))
		}
		if got := strings.TrimSuffix(normalize(out), "\n"); got != r.recipe {
			problems = append(problems, fmt.Sprintf("make -n %s prints %q, want exactly %q", r.target, got, r.recipe))
		}
	}
	return problems, nil
}

// noopGoal is the goal checkMakeDatabase asks make for, added by --eval, so
// that printing the database runs no recipe of the makefile.
const noopGoal = "cisteps-noop"

// databaseArgs are the make arguments that print the database and run nothing.
func databaseArgs(extra ...string) []string {
	return append(append([]string{"-pn", "--no-print-directory"}, extra...), "--eval="+noopGoal+":", noopGoal)
}

func normalize(out []byte) string {
	return strings.ReplaceAll(string(out), "\r\n", "\n")
}

// checkMakeDatabase checks what `make -n` cannot see (TICK-1149 review
// finding 1). make strips the -, @ and + prefixes when it echoes a recipe,
// and prints nothing of .IGNORE, .ONESHELL, -i in MAKEFLAGS, or a changed
// SHELL or .SHELLFLAGS, yet each one can turn a scan with findings into exit
// 0. This reads make's database (`make -pn`) instead. It compares the global
// SHELL, .SHELLFLAGS and MAKEFLAGS with those of a makefile that sets
// nothing, run the same way: GNU make reports SHELL as set by the makefile
// even when nothing sets it, so the origin cannot tell.
// It also returns the targets whose rule is not the pinned one.
func checkMakeDatabase(mk makeRunner) ([]string, map[string]bool, error) {
	out, err := mk(databaseArgs()...)
	if err != nil {
		return nil, nil, fmt.Errorf("make -pn: %w: %s", err, lastLine(out))
	}
	db := normalize(out)
	base, err := baselineDatabase(mk)
	if err != nil {
		return nil, nil, err
	}

	var problems []string
	want, got := makeGlobals(base), makeGlobals(db)
	for _, name := range []string{"SHELL", ".SHELLFLAGS", "MAKEFLAGS"} {
		if got[name] != want[name] {
			problems = append(problems, fmt.Sprintf("make sets %s to %q, want make's default %q: the vulncheck recipes must run as written", name, got[name], want[name]))
		}
	}
	for line := range strings.SplitSeq(db, "\n") {
		switch {
		case strings.HasPrefix(line, ".IGNORE:"), strings.HasPrefix(line, ".ONESHELL:"):
			problems = append(problems, fmt.Sprintf("make has %q: a failing recipe could pass", line))
		case !strings.HasPrefix(line, "\t") && !globalVar.MatchString(line) && shellVar.MatchString(line):
			problems = append(problems, fmt.Sprintf("make sets a target- or pattern-specific SHELL or .SHELLFLAGS: %q", line))
		}
	}
	bad := map[string]bool{}
	for _, r := range pinnedRecipes() {
		if p := recipeInDatabase(db, r); p != "" {
			problems = append(problems, fmt.Sprintf("make -pn %s: %s", r.target, p))
			bad[r.target] = true
		}
	}
	return problems, bad, nil
}

// baselineDatabase is make's database for a makefile that sets nothing.
func baselineDatabase(mk makeRunner) (string, error) {
	dir, err := os.MkdirTemp("", "cisteps-make")
	if err != nil {
		return "", err
	}
	defer func() { _ = os.RemoveAll(dir) }()
	empty := filepath.Join(dir, "baseline.mk")
	if err := os.WriteFile(empty, nil, 0o600); err != nil {
		return "", err
	}
	out, err := mk(databaseArgs("-f", empty)...)
	if err != nil {
		return "", fmt.Errorf("make -pn (baseline): %w: %s", err, lastLine(out))
	}
	return normalize(out), nil
}

var (
	// globalVar is a global variable line of make's database: NAME = value.
	globalVar = regexp.MustCompile(`^(SHELL|\.SHELLFLAGS|MAKEFLAGS) :{0,3}= ?(.*)$`)
	// shellVar is any other assignment to SHELL or .SHELLFLAGS: a target- or
	// pattern-specific one (`vulncheck: SHELL := x`, `# SHELL := x`).
	shellVar = regexp.MustCompile(`(^|[\s:])(SHELL|\.SHELLFLAGS)\s*(:::|::|:|\+|\?|!)?=`)
)

// makeGlobals returns the global SHELL, .SHELLFLAGS and MAKEFLAGS of db.
func makeGlobals(db string) map[string]string {
	vars := map[string]string{}
	for line := range strings.SplitSeq(db, "\n") {
		if m := globalVar.FindStringSubmatch(line); m != nil {
			vars[m[1]] = m[2]
		}
	}
	return vars
}

// isRuleLine is a line of db that starts target's rule: `target:` or
// `target: prerequisites`, not a target-specific variable.
func isRuleLine(line, target string) bool {
	rest, ok := strings.CutPrefix(line, target+":")
	return ok && !strings.HasPrefix(rest, ":") && !strings.Contains(rest, "=")
}

// recipeInDatabase returns why r's rule in db is not exactly `r.target:` with
// the one recipe line \t<r.recipe>, or "".
func recipeInDatabase(db string, r pinnedRecipe) string {
	var rules [][]string
	for para := range strings.SplitSeq(db, "\n\n") {
		lines := strings.Split(para, "\n")
		for _, l := range lines {
			if isRuleLine(l, r.target) {
				rules = append(rules, lines)
				break
			}
		}
	}
	if len(rules) != 1 {
		return fmt.Sprintf("want one rule, found %d", len(rules))
	}
	var head string
	var recipe []string
	for _, l := range rules[0] {
		switch {
		case strings.HasPrefix(l, "\t"):
			recipe = append(recipe, l)
		case isRuleLine(l, r.target):
			head = l
		}
	}
	if head != r.target+":" {
		return fmt.Sprintf("rule is %q, want %q with no prerequisites", head, r.target+":")
	}
	if want := "\t" + r.recipe; len(recipe) != 1 || recipe[0] != want {
		return fmt.Sprintf("recipe is %q, want exactly %q (no -, @ or + prefix)", recipe, want)
	}
	return ""
}

// lastLine is the last line of out, for an error message.
func lastLine(out []byte) string {
	lines := strings.Split(strings.TrimSpace(normalize(out)), "\n")
	return lines[len(lines)-1]
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

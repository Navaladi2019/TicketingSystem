package main

// The -copy mode (TICK-1149, ADR-001-E4 vulncheck contract rule 9): the whole
// of a workflow, comment lines aside, must equal a checked-in copy byte for
// byte, and the two vulncheck make recipes must be the pinned lines. An edit
// of one file alone (a step if:, an exit 0, a shell:, a paths filter, a
// top-level env: or defaults:, a second job reporting the gate's name, a
// `|| true` or a - prefix in a recipe, .IGNORE, a changed SHELL) fails lint.
// A deliberate edit of the workflow and the copy together passes: that is
// ADR RS3, caught by review, not by this.
//
// TICK-1178: tools-govulncheck, which installs the binary the two scans run,
// is pinned too, and so is GOVULNCHECK_VERSION. The makefile may not change
// the environment its recipes run with (a PATH with a fake govulncheck first,
// an exported GOFLAGS, GOENV or LD_PRELOAD), globally or for a pinned target.

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// pinnedRecipe is a make target the govulncheck workflow runs and its one
// recipe line as make's database holds it. `make -n --no-print-directory
// <target>` must print the same line with $(GOVULNCHECK_VERSION) expanded.
type pinnedRecipe struct{ target, recipe string }

// versionVar is the make variable that pins govulncheck (docs/pins.md).
const versionVar = "GOVULNCHECK_VERSION"

// pinnedRecipes are the three targets the workflow's scan jobs run: contract
// rule 9's two scans, with rule 7's GOWORK=off, and the install of the
// govulncheck binary they run (TICK-1178).
func pinnedRecipes() []pinnedRecipe {
	return []pinnedRecipe{
		{"tools-govulncheck", "go install golang.org/x/vuln/cmd/govulncheck@$(" + versionVar + ")"},
		{"vulncheck", "GOWORK=off govulncheck ./..."},
		{"vulncheck-temporal-server", "cd temporal-server && GOWORK=off govulncheck ./..."},
	}
}

// dryRun is what `make -n` prints for r when GOVULNCHECK_VERSION is version.
func (r pinnedRecipe) dryRun(version string) string {
	return strings.ReplaceAll(r.recipe, "$("+versionVar+")", version)
}

// releasedVersion is the only value GOVULNCHECK_VERSION may have: a release
// tag, never latest, a branch, an expression or a second word.
var releasedVersion = regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+$`)

// allowedExports are the variables the repository's makefiles add to every
// recipe's environment today. None is read by go or govulncheck. A new one
// fails check-ci-steps until it is added here, which a reviewer sees, and
// which runs both scans (tools/ci/cisteps/ is in vulncheck-detect's gate).
func allowedExports(name string) bool {
	switch name {
	case "NAVA_IMAGE_TAG", // Makefile
		"OTEL_TAIL_SAMPLE_PERCENT", "COMPOSE_PROFILES", "COMPOSE_FILE", "COMPOSE_PATH_SEPARATOR": // make/stack.mk
		return true
	}
	return false
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
	dbProblems, bad, version, err := checkMakeDatabase(mk)
	if err != nil {
		return nil, err
	}
	problems = append(problems, dbProblems...)
	envProblems, err := checkRecipeEnv(mk)
	if err != nil {
		return nil, err
	}
	problems = append(problems, envProblems...)
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
		if want := r.dryRun(version); strings.TrimSuffix(normalize(out), "\n") != want {
			problems = append(problems, fmt.Sprintf("make -n %s prints %q, want exactly %q", r.target, strings.TrimSuffix(normalize(out), "\n"), want))
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
// It also returns the targets whose rule is not the pinned one, and
// GOVULNCHECK_VERSION when it is a released version.
func checkMakeDatabase(mk makeRunner) ([]string, map[string]bool, string, error) {
	out, err := mk(databaseArgs()...)
	if err != nil {
		return nil, nil, "", fmt.Errorf("make -pn: %w: %s", err, lastLine(out))
	}
	db := normalize(out)
	base, err := baselineDatabase(mk)
	if err != nil {
		return nil, nil, "", err
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
	problems = append(problems, scopedVariables(db)...)
	bad := map[string]bool{}
	for _, r := range pinnedRecipes() {
		if p := recipeInDatabase(db, r); p != "" {
			problems = append(problems, fmt.Sprintf("make -pn %s: %s", r.target, p))
			bad[r.target] = true
		}
	}
	version, p := pinnedVersion(db)
	if p != "" {
		problems = append(problems, p)
		for _, r := range pinnedRecipes() {
			if r.dryRun("") != r.recipe {
				bad[r.target] = true
			}
		}
	}
	return problems, bad, version, nil
}

// pinnedVersion returns GOVULNCHECK_VERSION from db, or why it is not one
// simply expanded variable set to a released version.
func pinnedVersion(db string) (string, string) {
	var values []string
	for line := range strings.SplitSeq(db, "\n") {
		if name, value, ok := strings.Cut(line, " "); ok && name == versionVar {
			values = append(values, value)
		}
	}
	if len(values) != 1 {
		return "", fmt.Sprintf("make sets %s %d times, want once as %s := vX.Y.Z", versionVar, len(values), versionVar)
	}
	v, ok := strings.CutPrefix(values[0], ":= ")
	if !ok || !releasedVersion.MatchString(v) {
		return "", fmt.Sprintf("make sets %s %s, want %s := vX.Y.Z, a released version", versionVar, values[0], versionVar)
	}
	return v, ""
}

// scopedVariables returns a problem for each target- or pattern-specific
// variable that applies to a pinned target: make passes such a PATH or an
// exported GOFLAGS to the recipe, and neither `make -n` nor the rule shows it.
// make prints a target-specific variable as `<target>: NAME := value`, and a
// pattern-specific one as a `<pattern> :` line followed by `# NAME := value`.
func scopedVariables(db string) []string {
	var problems []string
	lines := strings.Split(db, "\n")
	for i, line := range lines {
		for _, r := range pinnedRecipes() {
			if strings.HasPrefix(line, r.target+": ") && !isRuleLine(line, r.target) {
				problems = append(problems, fmt.Sprintf("make sets a target-specific variable for %s: %q", r.target, line))
			}
			if pat, ok := strings.CutSuffix(line, " :"); ok && strings.Contains(pat, "%") && !strings.HasPrefix(pat, "#") &&
				i+2 < len(lines) && strings.HasPrefix(lines[i+1], "# ") && strings.Contains(lines[i+2], "=") && patternMatches(pat, r.target) {
				problems = append(problems, fmt.Sprintf("make sets a pattern-specific variable for %s (pattern %s): %q", r.target, pat, strings.TrimPrefix(lines[i+2], "# ")))
			}
		}
	}
	return problems
}

// patternMatches reports whether make's pattern (one %) matches target.
func patternMatches(pattern, target string) bool {
	pre, suf, _ := strings.Cut(pattern, "%")
	return len(target) >= len(pre)+len(suf) && strings.HasPrefix(target, pre) && strings.HasSuffix(target, suf)
}

// envGoalPrefix starts the goal checkRecipeEnv adds with --eval. The rest is
// random, so that no makefile can define a rule of that name in advance.
const envGoalPrefix = "cisteps-env-"

// checkRecipeEnv compares the environment a recipe of the makefile gets with
// the one a recipe of an empty makefile gets, both run the same way. Every
// variable that differs and is not in allowedExports is a problem: a changed
// PATH, an exported GOFLAGS, GOENV or LD_PRELOAD, .EXPORT_ALL_VARIABLES, an
// unexport. The probe runs `env` through make's SHELL, so a changed SHELL
// also shows here, besides checkMakeDatabase's own problem.
func checkRecipeEnv(mk makeRunner) ([]string, error) {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return nil, err
	}
	goal := envGoalPrefix + hex.EncodeToString(b[:])
	probe := func(extra ...string) ([]string, error) {
		args := append(append([]string{"--no-print-directory"}, extra...), "--eval="+goal+":\n\t@env", goal)
		out, err := mk(args...)
		if err != nil {
			return nil, fmt.Errorf("make %s (env): %w: %s", goal, err, lastLine(out))
		}
		return strings.Split(normalize(out), "\n"), nil
	}
	got, err := probe()
	if err != nil {
		return nil, err
	}
	dir, err := os.MkdirTemp("", "cisteps-env")
	if err != nil {
		return nil, err
	}
	defer func() { _ = os.RemoveAll(dir) }()
	empty := filepath.Join(dir, "baseline.mk")
	if err := os.WriteFile(empty, nil, 0o600); err != nil {
		return nil, err
	}
	want, err := probe("-f", empty)
	if err != nil {
		return nil, err
	}
	changed := map[string]bool{}
	for _, d := range [][2][]string{{got, want}, {want, got}} {
		for _, line := range d[0] {
			if line != "" && !slices.Contains(d[1], line) {
				name, _, _ := strings.Cut(line, "=")
				if !allowedExports(name) {
					changed[name] = true
				}
			}
		}
	}
	if len(changed) == 0 {
		return nil, nil
	}
	names := slices.Sorted(maps.Keys(changed))
	return []string{fmt.Sprintf("make changes the environment of its recipes: %s (the vulncheck recipes must run with the job's environment; a deliberate export that go and govulncheck do not read goes in allowedExports)", strings.Join(names, ", "))}, nil
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

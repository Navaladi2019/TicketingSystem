package main

// The -copy mode (TICK-1149, ADR-001-E4 vulncheck contract rule 9): the whole
// of a workflow, comment lines aside, must equal a checked-in copy byte for
// byte, and the two vulncheck make recipes and the installer
// (tools-govulncheck, TICK-1167) must be the pinned lines, run with each
// target as the goal in the environment an empty makefile gives them, by a
// makefile that runs no code at parse time outside an allow-list
// (parsetime.go). A makefile edit that acts only when a CI-only variable
// matches, such as $(GITHUB_JOB), is not caught: AR-TICK-1167-01, TICK-1183. An edit
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
	"slices"
	"strings"
)

// pinnedRecipe is a make target the govulncheck workflow runs. Its rule in
// make's database must be `<target>:` with the one recipe line recipe, as
// written, and `make -n --no-print-directory <target>` must print exactly
// printed (recipe when printed is empty).
type pinnedRecipe struct{ target, recipe, printed string }

// dryRun is what `make -n` of r must print.
func (r pinnedRecipe) dryRun() string {
	if r.printed != "" {
		return r.printed
	}
	return r.recipe
}

// govulncheckVersion is docs/pins.md's govulncheck version.
const govulncheckVersion = "v1.9.0"

// pinnedRecipes are contract rule 9's two recipes, with rule 7's GOWORK=off,
// and the installer the scan jobs run before them (TICK-1167): an installer
// edited to write a stub govulncheck that exits 0 turns every scan green.
// The installed module path and version are pinned in the printed line; a
// version bump edits Makefile, docs/pins.md and govulncheckVersion together.
func pinnedRecipes() []pinnedRecipe {
	return []pinnedRecipe{
		{"vulncheck", "GOWORK=off govulncheck ./...", ""},
		{"vulncheck-temporal-server", "cd temporal-server && GOWORK=off govulncheck ./...", ""},
		{"tools-govulncheck", "go install golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)", "go install golang.org/x/vuln/cmd/govulncheck@" + govulncheckVersion},
	}
}

// allowedExport is whether name is one of the variables the repository's
// makefiles export, none of which go reads (TICK-1167): NAVA_IMAGE_TAG
// (Makefile) and the compose and OpenTelemetry settings of make/stack.mk. Any
// other difference between the environment of a recipe and that of an empty
// makefile fails the check.
func allowedExport(name string) bool {
	switch name {
	case "NAVA_IMAGE_TAG", "OTEL_TAIL_SAMPLE_PERCENT", "COMPOSE_PROFILES", "COMPOSE_FILE", "COMPOSE_PATH_SEPARATOR":
		return true
	}
	return false
}

// allowedGoVar is whether name is a GO* make variable a makefile may set:
// tool names and versions, not settings of the go command.
// GOVULNCHECK_VERSION is pinned through tools-govulncheck's printed line.
func allowedGoVar(name string) bool {
	switch name {
	case "GOLANGCI_LINT", "GOLANGCI_LINT_VERSION", "GOVULNCHECK_VERSION":
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

// checkCopy returns the problems of workflowPath against copyPath, of the
// makefiles in makeDir ("" is the current directory) and of the pinned
// recipes, or an error when a file or make cannot be read. mk runs make in
// makeDir.
func checkCopy(workflowPath, copyPath, makeDir string, mk makeRunner) ([]string, error) {
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
	pt, err := parseTimeProblems(makeDir)
	if err != nil {
		return nil, err
	}
	problems = append(problems, pt...)
	base, err := baseline(mk)
	if err != nil {
		return nil, err
	}
	var makeErr error
	for _, r := range pinnedRecipes() {
		p, err := targetProblems(r, base, mk)
		problems = append(problems, p...)
		if err != nil && makeErr == nil {
			makeErr = err
		}
	}
	if makeErr != nil {
		// A failing make is an error (exit 2) unless the makefile already
		// fails the check: a + recipe line runs even under -n, and a load
		// fails, yet make prints its database and the problem is known.
		if len(problems) == 0 {
			return nil, makeErr
		}
		problems = append(problems, makeErr.Error())
	}
	return dedupe(problems), nil
}

// makeBaseline is what make gives a makefile that sets nothing but the
// pinned targets, each with the recipe true: its database, and the
// environment of a recipe (nil when !envCaptured()).
type makeBaseline struct {
	db  string
	env map[string]string
}

// targetProblems checks r with r.target as the goal, as CI runs it (TICK-1167
// merge condition 2): a makefile can set a variable, a prerequisite or an
// export only when vulncheck is in MAKECMDGOALS, and a database read for
// another goal never sees it. `make -pn <target>` prints the dry run, then the
// database; `make SHELL=<dump> <target>` runs the recipe through the dump.
// make runs a +-prefixed recipe line even under -n, as `make vulncheck` in
// the lint job would: such a line is in the database, so it fails the check.
// When make fails, the problems its database shows come with the error.
func targetProblems(r pinnedRecipe, base makeBaseline, mk makeRunner) ([]string, error) {
	out, err := mk("-pn", "--no-print-directory", r.target)
	dry, db := splitDatabase(normalize(out))
	var problems []string
	if db != "" || err == nil {
		problems = databaseProblems(db, base.db)
		if p := recipeInDatabase(db, r); p != "" {
			problems = append(problems, fmt.Sprintf("make -pn %s: %s", r.target, p))
		}
	}
	if err != nil {
		return problems, fmt.Errorf("make -pn %s: %w: %s", r.target, err, lastLine(out))
	}
	if len(problems) > 0 {
		// The dump would run the recipe under a SHELL the makefile may have
		// changed: report what the database shows and stop.
		return problems, nil
	}
	if got := strings.TrimSuffix(dry, "\n"); got != r.dryRun() {
		return []string{fmt.Sprintf("make -n %s prints %q, want exactly %q", r.target, got, r.dryRun())}, nil
	}
	if base.env == nil {
		return nil, nil
	}
	return environmentProblems(r, base.env, mk)
}

// splitDatabase splits `make -pn` output into the dry run and the database,
// which starts with its "# GNU Make" header.
func splitDatabase(out string) (dry, db string) {
	if strings.HasPrefix(out, "# GNU Make ") {
		return "", out
	}
	if i := strings.Index(out, "\n# GNU Make "); i >= 0 {
		return out[:i+1], out[i+1:]
	}
	return out, ""
}

// databaseProblems checks what `make -n` cannot see (TICK-1149 review
// finding 1). make strips the -, @ and + prefixes when it echoes a recipe,
// and prints nothing of .IGNORE, .ONESHELL, -i in MAKEFLAGS, or a changed
// SHELL or .SHELLFLAGS, yet each one can turn a scan with findings into exit
// 0. It compares the global SHELL, .SHELLFLAGS and MAKEFLAGS with those of the
// baseline, run the same way: GNU make reports SHELL as set by the makefile
// even when nothing sets it, so the origin cannot tell. TICK-1167 adds the
// environment: no makefile assignment to PATH or a GO* variable (but
// allowedGoVar), no pattern-specific variable, no .EXPORT_ALL_VARIABLES.
func databaseProblems(db, base string) []string {
	var problems []string
	want, got := makeGlobals(base), makeGlobals(db)
	for _, name := range []string{"SHELL", ".SHELLFLAGS", "MAKEFLAGS"} {
		if got[name] != want[name] {
			problems = append(problems, fmt.Sprintf("make sets %s to %q, want make's default %q: the vulncheck recipes must run as written", name, got[name], want[name]))
		}
	}
	for line := range strings.SplitSeq(db, "\n") {
		switch {
		case strings.HasPrefix(line, ".IGNORE:"), strings.HasPrefix(line, ".ONESHELL:"), strings.HasPrefix(line, ".EXPORT_ALL_VARIABLES:"):
			problems = append(problems, fmt.Sprintf("make has %q: a failing recipe could pass, or a recipe get every variable", line))
		case !strings.HasPrefix(line, "\t") && !globalVar.MatchString(line) && shellVar.MatchString(line):
			problems = append(problems, fmt.Sprintf("make sets a target- or pattern-specific SHELL or .SHELLFLAGS: %q", line))
		}
	}
	for _, v := range makefileVars(db) {
		if v.name == "PATH" || strings.HasPrefix(v.name, "GO") && !allowedGoVar(v.name) {
			problems = append(problems, fmt.Sprintf("make assigns %s (%s): no makefile sets PATH or a GO* variable, the scans run in the environment CI gives them", v.name, v.origin))
		}
	}
	inBase := map[string]bool{}
	for _, p := range patternVars(base) {
		inBase[p] = true
	}
	for _, p := range patternVars(db) {
		if !inBase[p] {
			problems = append(problems, fmt.Sprintf("make has a pattern-specific variable for %q: it can change the environment of a pinned recipe", p))
		}
	}
	return problems
}

// envArgs are the make arguments that run goal's recipes through the dump.
func envArgs(shell, goal string, extra ...string) []string {
	return append(append([]string{"-s", "--no-print-directory", "SHELL=" + shell, envDumpVar + "=1"}, extra...), goal)
}

// environmentProblems runs r.target's recipe through the dump. make must call
// the shell once, with -c and the pinned line, and give it the environment
// the baseline gets (but allowedExport): `export GOFLAGS`, `PATH :=`, a
// goal-conditional export and .EXPORT_ALL_VARIABLES all show here.
func environmentProblems(r pinnedRecipe, baseEnv map[string]string, mk makeRunner) ([]string, error) {
	shell, err := envDumpShell()
	if err != nil {
		return nil, err
	}
	out, err := mk(envArgs(shell, r.target)...)
	if err != nil {
		return []string{fmt.Sprintf("make %s through the environment dump fails: %v: %s", r.target, err, lastLine(out))}, nil
	}
	recs, other, err := parseEnvDump(normalize(out))
	if err != nil {
		return nil, err
	}
	want := []string{"-c", r.dryRun()}
	if len(recs) != 1 || !slices.Equal(recs[0].Args, want) || len(other) > 0 {
		var ran [][]string
		for _, rec := range recs {
			ran = append(ran, rec.Args)
		}
		return []string{fmt.Sprintf("make %s runs %q and prints %q, want one shell call %q and no output", r.target, ran, other, want)}, nil
	}
	got := envMap(recs[0].Env)
	names := map[string]bool{}
	for k := range got {
		names[k] = true
	}
	for k := range baseEnv {
		names[k] = true
	}
	var problems []string
	for _, k := range sortedKeys(names) {
		g, gok := got[k]
		w, wok := baseEnv[k]
		if allowedExport(k) || gok == wok && g == w {
			continue
		}
		switch {
		case !gok:
			problems = append(problems, fmt.Sprintf("make %s: recipes get no %s, an empty makefile gives %s=%q: no makefile may unexport it", r.target, k, k, w))
		case !wok:
			problems = append(problems, fmt.Sprintf("make %s: recipes get %s=%q, an empty makefile gives none: no makefile may export it (export, .EXPORT_ALL_VARIABLES)", r.target, k, g))
		default:
			problems = append(problems, fmt.Sprintf("make %s: recipes get %s=%q, an empty makefile gives %q: no makefile may change it", r.target, k, g, w))
		}
	}
	return problems, nil
}

// baseline is make's database, and the environment of a recipe, for a
// makefile with the pinned targets and nothing else, run as targetProblems
// runs the real one.
func baseline(mk makeRunner) (makeBaseline, error) {
	dir, err := os.MkdirTemp("", "cisteps-make")
	if err != nil {
		return makeBaseline{}, err
	}
	defer func() { _ = os.RemoveAll(dir) }()
	var text strings.Builder
	for _, r := range pinnedRecipes() {
		text.WriteString(r.target + ":\n\ttrue\n")
	}
	file := filepath.Join(dir, "baseline.mk")
	if err := os.WriteFile(file, []byte(text.String()), 0o600); err != nil {
		return makeBaseline{}, err
	}
	goal := pinnedRecipes()[0].target
	out, err := mk("-pn", "--no-print-directory", "-f", file, goal)
	if err != nil {
		return makeBaseline{}, fmt.Errorf("make -pn (baseline): %w: %s", err, lastLine(out))
	}
	_, db := splitDatabase(normalize(out))
	base := makeBaseline{db: db}
	if !envCaptured() {
		return base, nil
	}
	shell, err := envDumpShell()
	if err != nil {
		return makeBaseline{}, err
	}
	envOut, err := mk(envArgs(shell, goal, "-f", file)...)
	if err != nil {
		return makeBaseline{}, fmt.Errorf("make %s through the environment dump (baseline): %w: %s", goal, err, lastLine(envOut))
	}
	recs, _, err := parseEnvDump(normalize(envOut))
	if err != nil {
		return makeBaseline{}, err
	}
	if len(recs) != 1 {
		return makeBaseline{}, fmt.Errorf("make %s through the environment dump (baseline): %d shell calls, want 1: %s", goal, len(recs), lastLine(envOut))
	}
	base.env = envMap(recs[0].Env)
	return base, nil
}

func normalize(out []byte) string {
	return strings.ReplaceAll(string(out), "\r\n", "\n")
}

// dedupe drops repeats of a problem the three targets share, keeping order.
func dedupe(problems []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, p := range problems {
		if !seen[p] {
			seen[p] = true
			out = append(out, p)
		}
	}
	return out
}

// makeVar is a global variable of make's database that a makefile set.
type makeVar struct{ name, origin string }

var (
	// varOrigin is the comment above a variable a makefile or an override
	// directive set: `# makefile (from 'make/lint.mk', line 3)`.
	varOrigin = regexp.MustCompile(`^# (makefile|'override' directive)( \(from .*\))?$`)
	// varName is the name in a variable line: `NAME = v`, `NAME := v`,
	// `define NAME`.
	varName = regexp.MustCompile(`^(?:define )?([^\s:#=+?!]+)(?:\s|$)`)
)

// makefileVars are the global variables of db that a makefile set.
func makefileVars(db string) []makeVar {
	var vars []makeVar
	lines := strings.Split(db, "\n")
	for i := 0; i+1 < len(lines); i++ {
		if !varOrigin.MatchString(lines[i]) {
			continue
		}
		if m := varName.FindStringSubmatch(lines[i+1]); m != nil {
			vars = append(vars, makeVar{m[1], strings.TrimPrefix(lines[i], "# ")})
		}
	}
	return vars
}

// patternVars are the patterns of db's pattern-specific variables: the
// `pattern :` lines of its "Pattern-specific Variable Values" section.
func patternVars(db string) []string {
	_, section, ok := strings.Cut(db, "# Pattern-specific Variable Values\n")
	if !ok {
		return nil
	}
	var pats []string
	for line := range strings.SplitSeq(section, "\n") {
		if strings.HasPrefix(line, "# Directories") || strings.HasPrefix(line, "# Files") || strings.HasSuffix(line, "pattern-specific variable values") || strings.HasSuffix(line, "pattern-specific variable values.") {
			break
		}
		if p, ok := strings.CutSuffix(line, " :"); ok && !strings.HasPrefix(line, "#") && !strings.HasPrefix(line, "\t") {
			pats = append(pats, p)
		}
	}
	return pats
}

func sortedKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
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
	rest, ok := targetLine(line, target)
	return ok && !strings.Contains(rest, "=")
}

// isTargetVarLine is a line of db that sets a target-specific variable of
// target: `target: NAME := value`.
func isTargetVarLine(line, target string) bool {
	rest, ok := targetLine(line, target)
	return ok && strings.Contains(rest, "=")
}

// targetLine returns the rest of a line that starts `target:` (not
// `target::`).
func targetLine(line, target string) (string, bool) {
	rest, ok := strings.CutPrefix(line, target+":")
	return rest, ok && !strings.HasPrefix(rest, ":")
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
	for l := range strings.SplitSeq(db, "\n") {
		if isTargetVarLine(l, r.target) {
			return fmt.Sprintf("target-specific variable %q: a pinned recipe runs in the environment CI gives it", l)
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

// blockHeader is a line that starts a YAML block scalar (`run: |`, `- >-`,
// `key: |2 # note`). A plain scalar that only looks like one makes the
// comparison stricter, never looser.
var blockHeader = regexp.MustCompile(`(?:^|:\s|-\s)\s*[|>][1-9+-]{0,2}\s*(?:#.*)?$`)

// uncommented returns the lines of data whose first non-blank character is
// not #. Every other byte counts: blank lines, indentation, trailing comments
// and line endings. A # line inside a block scalar (`run: |`) is content, not
// a comment, and counts too (TICK-1167 AC 3): a line indented deeper than
// the block's header line belongs to the block.
func uncommented(data []byte) []numbered {
	var out []numbered
	block := -1 // the indentation of the open block scalar's header line
	for i, line := range bytes.SplitAfter(data, []byte("\n")) {
		if len(line) == 0 {
			continue
		}
		t := bytes.TrimLeft(line, " \t")
		indent := len(line) - len(t)
		blank := len(bytes.TrimSpace(t)) == 0
		if block >= 0 && (blank || indent > block) {
			out = append(out, numbered{i + 1, line})
			continue
		}
		block = -1
		if len(t) > 0 && t[0] == '#' {
			continue
		}
		if blockHeader.Match(bytes.TrimRight(t, "\r\n")) {
			block = indent
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

// Command cichanged prints the CI stages a change needs (TICK-153 O7), from
// the files that differ from the merge base with -base (default origin/main),
// the working tree's uncommitted and untracked files included. `make
// ci-changed` runs ci-isolated with those stages.
//
// Every change runs ci-lint. Documentation (.md and .html outside a Go
// package and testdata) runs nothing else. A Go package's files add ci-unit, plus
// ci-integration when an integration test depends on the package, plus
// ci-stack when an image or a stack test does and the package is one of
// stackTriggers (TICK-652): a command, the migrations and database bootstrap,
// the configuration, the Keycloak adapter, or a stack suite. Plain Go logic
// that the unit and integration stages already cover no longer starts the
// stack; the daily full run on main covers it there (AR-TICK-652-01). A file
// no rule knows runs every stage, so a new kind of file is never skipped by
// default.
//
// With -areas it prints instead the stack-case areas the change touches
// (TICK-563): all, none, or a comma list of keycloak, pgbouncer and db, which
// `make stack-tests AREAS=...` uses to skip the cases of untouched areas. A
// file no area rule knows turns on every area.
//
// With -images it prints instead how the stack job builds its images
// (TICK-628): real (the deploy/compose Dockerfiles, as main always does) when
// the change touches a Dockerfile, .dockerignore, go.mod, go.sum or a compose
// build: block, else thin (deploy/compose/Dockerfile.ci over binaries built
// on the runner, make image-ci IMAGE_THIN=1).
//
// Usage: cichanged [-base origin/main] [-v] [-areas | -images]. -v prints why
// each file needs its stages (or areas, or image mode) on stderr.
package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// allStages are the stages in the order ci-isolated lists them.
func allStages() []string { return []string{"ci-lint", "ci-unit", "ci-integration", "ci-stack"} }

// stackRoots are the go list arguments for the Go packages behind the stack:
// the commands the images build, the S7 demo binary stack-test-demo_outbox
// builds (cmd/demo-outbox), the interptest stack worker (cmd/nava-worker
// with the S11a fakes, TICK-367), and the stack test suites with their tags
// (demooutbox: the S7 suite, TICK-292; interpcrash: the interpreter crash
// case, TICK-351).
func stackRoots() [][]string {
	return [][]string{
		{"./cmd/nava-core", "./cmd/nava-db", "./cmd/nava-worker", "./cmd/nava-relay", "./cmd/nava-onboard", "./cmd/nava-groupsync", "./cmd/nava-purge", "./cmd/demo-outbox", "./tools/ci/ci6report", "./tools/ci/ci5report"},
		{"-tags", "interptest", "./cmd/nava-worker"},
		{"-tags", "temporalstack,demooutbox", "-test", "./internal/temporalstack"},
		{"-tags", "identitystack", "-test", "./internal/identitystack"},
		{"-tags", "workerstack", "-test", "./internal/workerstack"},
		{"-tags", "workerstack,interpcrash", "-test", "./internal/workerstack"},
	}
}

// stackTriggers are the package directories (and, ending in "/", directory
// prefixes) whose change starts the stack stage, when an image or a stack test
// depends on them (TICK-652): what decides whether the stack starts and how
// it is wired, not the logic inside it. cmd/: each command is an image's
// main or a stack test's binary. db, internal/db/{migrate,roles,grants}:
// the migrations and the db-roles bootstrap every stack runs first.
// internal/platform/config: the environment the compose file sets. Keycloak:
// the adapter the realm and the identity case meet. The stack suites
// themselves and their report tools. Every other package's change runs unit
// (and integration) only, and its stack cases on the next full run.
func stackTriggers() []string {
	return []string{
		"cmd/",
		"db", "internal/db/migrate", "internal/db/roles", "internal/db/grants",
		"internal/platform/config",
		"internal/identity/keycloak",
		"internal/temporalstack", "internal/identitystack", "internal/workerstack",
		"tools/ci/ci5report", "tools/ci/ci6report",
	}
}

// stackTrigger reports whether a change to package directory pkg starts the
// stack stage (with g.stack[pkg]), and the trigger it matched.
func stackTrigger(pkg string) (string, bool) {
	for _, t := range stackTriggers() {
		if pkg == t || (strings.HasSuffix(t, "/") && strings.HasPrefix(pkg, t)) {
			return t, true
		}
	}
	return "", false
}

func main() {
	os.Exit(cli(os.Args[1:], os.Stdout, os.Stderr))
}

// cli runs cichanged with the arguments after the program name and returns
// the exit code: 2 for a bad flag or a failed git or go list.
func cli(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("cichanged", flag.ContinueOnError)
	fs.SetOutput(stderr)
	base := fs.String("base", "origin/main", "base ref of the change")
	verbose := fs.Bool("v", false, "print the reason for each file on stderr")
	areas := fs.Bool("areas", false, "print the stack-case areas the change touches instead of the stages")
	images := fs.Bool("images", false, "print the stack image build mode (real or thin) instead of the stages")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *areas && *images {
		_, _ = fmt.Fprintln(stderr, "cichanged: -areas and -images are exclusive")
		return 2
	}
	if *images {
		if err := runImages(*base, *verbose, stdout, stderr); err != nil {
			_, _ = fmt.Fprintln(stderr, "cichanged:", err)
			return 2
		}
		return 0
	}
	if err := run(*base, *verbose, *areas, stdout, stderr); err != nil {
		_, _ = fmt.Fprintln(stderr, "cichanged:", err)
		return 2
	}
	return 0
}

func run(base string, verbose, areas bool, stdout, stderr io.Writer) error {
	files, err := changedFiles(base)
	if err != nil {
		return err
	}
	g, err := loadGraph()
	if err != nil {
		return err
	}
	if areas {
		out, reasons := classifyAreas(files, g)
		if verbose {
			for _, r := range reasons {
				_, _ = fmt.Fprintln(stderr, "cichanged:", r)
			}
		}
		_, _ = fmt.Fprintln(stdout, out)
		return nil
	}
	stages, reasons := classify(files, g)
	if verbose {
		for _, r := range reasons {
			_, _ = fmt.Fprintln(stderr, "cichanged:", r)
		}
	}
	_, _ = fmt.Fprintln(stdout, strings.Join(stages, " "))
	return nil
}

// runImages prints real or thin for the change against base. It needs no
// package graph: the rule reads only paths and compose build: blocks.
func runImages(base string, verbose bool, stdout, stderr io.Writer) error {
	files, err := changedFiles(base)
	if err != nil {
		return err
	}
	mb, err := mergeBase(base)
	if err != nil {
		return err
	}
	atBase := func(f string) ([]byte, error) {
		if _, err := git("cat-file", "-e", mb+":"+f); err != nil {
			return nil, nil // not on the base side
		}
		out, err := git("show", mb+":"+f)
		return []byte(out), err
	}
	inTree := func(f string) ([]byte, error) {
		b, err := os.ReadFile(filepath.FromSlash(f))
		if os.IsNotExist(err) {
			return nil, nil
		}
		return b, err
	}
	mode, reasons := classifyImages(files, atBase, inTree)
	if verbose {
		for _, r := range reasons {
			_, _ = fmt.Fprintln(stderr, "cichanged:", r)
		}
	}
	_, _ = fmt.Fprintln(stdout, mode)
	return nil
}

// graph is the root module's package directories (slash paths relative to
// the repository root) and those the integration and stack stages depend on.
type graph struct {
	packages, integration, stack map[string]bool
}

// classify returns the stages files need, in run order, and one reason per
// file.
func classify(files []string, g graph) ([]string, []string) {
	need := map[string]bool{"ci-lint": true}
	var reasons []string
	for _, f := range files {
		s, why := fileStages(f, g)
		for _, st := range s {
			need[st] = true
		}
		reasons = append(reasons, fmt.Sprintf("%s: %s (%s)", f, strings.Join(s, " "), why))
	}
	var out []string
	for _, st := range allStages() {
		if need[st] {
			out = append(out, st)
		}
	}
	return out, reasons
}

func fileStages(f string, g graph) ([]string, string) {
	lint := []string{"ci-lint"}
	switch {
	case f == "go.mod" || f == "go.sum":
		return allStages(), "root module dependencies"
	case f == "Makefile" || strings.HasPrefix(f, "make/"):
		return allStages(), "the Makefile and its make/*.mk fragments define every stage"
	case f == "docs/pins.md":
		return allStages(), "pins pick the test and stack images"
	case strings.HasPrefix(f, "temporal-server/"):
		return []string{"ci-lint", "ci-unit", "ci-stack"}, "temporal-server module and image"
	case strings.HasPrefix(f, "tools/navalint/"):
		return []string{"ci-lint", "ci-unit"}, "navalint module"
	case strings.HasPrefix(f, "docs/acceptance/S13/"):
		// The S13 kit's samples, expected outputs and README are checked by
		// tools/acceptance/s13kit (TICK-426).
		return []string{"ci-lint", "ci-unit"}, "S13 acceptance kit, checked by tools/acceptance/s13kit"
	case (path.Ext(f) == ".md" || path.Ext(f) == ".html") && !strings.Contains("/"+f, "/testdata/"):
		// Only outside a Go package: a package may embed or read its own
		// .md files, which the package rule below covers.
		if _, ok := packageOf(f, g.packages); !ok {
			return lint, "documentation"
		}
	case strings.HasPrefix(f, ".github/") || f == ".golangci.yml" ||
		f == "scripts/ci/vulncheck.sh" || f == "scripts/ci/navalint.sh" || f == "scripts/ci/lint-fixture.sh":
		return lint, "lint configuration, lint script or workflows"
	case f == "scripts/ci/test-integration.sh" || f == "scripts/ci/isolation.sh" || strings.HasPrefix(f, "scripts/ci/pgbouncer/"):
		return []string{"ci-lint", "ci-integration"}, "integration script"
	case strings.HasPrefix(f, "deploy/") || strings.HasPrefix(path.Base(f), "compose") || f == ".dockerignore":
		// Unit tests read the compose and collector files too
		// (cmd/nava-core, internal/platform/telemetry).
		return []string{"ci-lint", "ci-unit", "ci-stack"}, "compose stack"
	}
	if pkg, ok := packageOf(f, g.packages); ok {
		s := []string{"ci-lint", "ci-unit"}
		var why []string
		if g.integration[pkg] {
			s = append(s, "ci-integration")
			why = append(why, "integration tests depend on it")
		}
		if g.stack[pkg] {
			if _, ok := stackTrigger(pkg); ok {
				s = append(s, "ci-stack")
				why = append(why, "an image or stack test depends on it")
			} else {
				why = append(why, "stack cases on the full run only (TICK-652)")
			}
		}
		return s, "package " + pkg + strings.Join(append([]string{""}, why...), "; ")
	}
	return allStages(), "no rule for this file, so every stage"
}

// packageOf returns the deepest package directory holding f.
func packageOf(f string, packages map[string]bool) (string, bool) {
	for d := path.Dir(f); d != "." && d != "/"; d = path.Dir(d) {
		if packages[d] {
			return d, true
		}
	}
	return "", false
}

// changedFiles lists the files that differ from the merge base with base,
// committed or not, and the untracked ones.
func changedFiles(base string) ([]string, error) {
	mb, err := mergeBase(base)
	if err != nil {
		return nil, err
	}
	// --no-renames: a moved file counts at its old path too, so a file moved
	// out of a package into docs/ still runs that package's stages.
	diff, err := git("diff", "--name-only", "--no-renames", mb)
	if err != nil {
		return nil, err
	}
	untracked, err := git("ls-files", "--others", "--exclude-standard")
	if err != nil {
		return nil, err
	}
	var out []string
	for _, l := range strings.Split(diff+"\n"+untracked, "\n") {
		if l = strings.TrimSpace(l); l != "" && !slices.Contains(out, l) {
			out = append(out, l)
		}
	}
	slices.Sort(out)
	return out, nil
}

// mergeBase is the merge base of base and HEAD.
func mergeBase(base string) (string, error) {
	mb, err := git("merge-base", base, "HEAD")
	if err != nil {
		return "", fmt.Errorf("%w (fetch %s first)", err, base)
	}
	return strings.TrimSpace(mb), nil
}

func loadGraph() (graph, error) {
	root, err := git("rev-parse", "--show-toplevel")
	if err != nil {
		return graph{}, err
	}
	root = strings.TrimSpace(root)
	g := graph{}
	if g.packages, err = dirs(root, "-tags", "integration,temporalstack,identitystack,workerstack,demooutbox,interptest,interpcrash", "./..."); err != nil {
		return g, err
	}
	var integ []string
	for d := range g.packages {
		if hasIntegrationTests(filepath.Join(root, filepath.FromSlash(d))) {
			integ = append(integ, "./"+d)
		}
	}
	slices.Sort(integ)
	if len(integ) > 0 {
		if g.integration, err = dirs(root, append([]string{"-deps", "-test", "-tags", "integration"}, integ...)...); err != nil {
			return g, err
		}
	}
	g.stack = map[string]bool{}
	for _, args := range stackRoots() {
		s, err := dirs(root, append([]string{"-deps"}, args...)...)
		if err != nil {
			return g, err
		}
		for d := range s {
			g.stack[d] = true
		}
	}
	return g, nil
}

// dirs runs go list with args and returns the root module's package
// directories, as slash paths relative to root.
func dirs(root string, args ...string) (map[string]bool, error) {
	//nolint:gosec // go list with arguments from this file
	cmd := exec.CommandContext(context.Background(), "go", append([]string{"list", "-f", "{{if not .Standard}}{{.Dir}}{{end}}"}, args...)...)
	cmd.Dir = root
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("go list %s: %w\n%s", strings.Join(args, " "), err, stderr.String())
	}
	m := map[string]bool{}
	for _, l := range strings.Split(string(out), "\n") {
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}
		rel, err := filepath.Rel(root, l)
		if err != nil || strings.HasPrefix(rel, "..") {
			continue // a dependency outside the repository
		}
		m[filepath.ToSlash(rel)] = true
	}
	return m, nil
}

// The same rule as scripts/ci/test-integration.sh's grep.
var integrationTag = regexp.MustCompile(`(?m)^//go:build integration`)

func hasIntegrationTests(dir string) bool {
	files, _ := filepath.Glob(filepath.Join(dir, "*_test.go"))
	for _, f := range files {
		b, err := os.ReadFile(f) //nolint:gosec // a test file found by the glob
		if err == nil && integrationTag.Match(b) {
			return true
		}
	}
	return false
}

func git(args ...string) (string, error) {
	var stderr bytes.Buffer
	cmd := exec.CommandContext(context.Background(), "git", args...) //nolint:gosec // git with arguments from this file
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return string(out), nil
}

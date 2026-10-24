// Command cichanged prints the CI stages a change needs (TICK-153 O7), from
// the files that differ from the merge base with -base (default origin/main),
// the working tree's uncommitted and untracked files included. `make
// ci-changed` runs ci-isolated with those stages.
//
// Every change runs ci-lint. A Go package's files add ci-unit, plus
// ci-integration when an integration test depends on the package, plus
// ci-stack when an image or a stack test does. A file no rule knows runs every
// stage, so a new kind of file is never skipped by default.
//
// Usage: cichanged [-base origin/main] [-v]. -v prints why each file needs
// its stages on stderr.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// The stages in the order ci-isolated runs them.
var allStages = []string{"ci-lint", "ci-unit", "ci-integration", "ci-stack"}

// The Go packages behind the stack: the commands the images build and the
// stack test suites with their tags.
var stackRoots = [][]string{
	{"./cmd/nava-core", "./cmd/nava-db", "./tools/ci/ci6report"},
	{"-tags", "temporalstack", "-test", "./internal/temporalstack"},
	{"-tags", "identitystack", "-test", "./internal/identitystack"},
}

func main() {
	base := flag.String("base", "origin/main", "base ref of the change")
	verbose := flag.Bool("v", false, "print the reason for each file on stderr")
	flag.Parse()
	if err := run(*base, *verbose); err != nil {
		fmt.Fprintln(os.Stderr, "cichanged:", err)
		os.Exit(2)
	}
}

func run(base string, verbose bool) error {
	files, err := changedFiles(base)
	if err != nil {
		return err
	}
	g, err := loadGraph()
	if err != nil {
		return err
	}
	stages, reasons := classify(files, g)
	if verbose {
		for _, r := range reasons {
			fmt.Fprintln(os.Stderr, "cichanged:", r)
		}
	}
	fmt.Println(strings.Join(stages, " "))
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
	for _, st := range allStages {
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
		return allStages, "root module dependencies"
	case f == "docs/pins.md":
		return allStages, "pins pick the test and stack images"
	case strings.HasPrefix(f, "temporal-server/"):
		return []string{"ci-lint", "ci-unit", "ci-stack"}, "temporal-server module and image"
	case strings.HasPrefix(f, "tools/navalint/"):
		return []string{"ci-lint", "ci-unit"}, "navalint module"
	case strings.HasPrefix(f, "docs/") || path.Ext(f) == ".md" || path.Ext(f) == ".html":
		return lint, "documentation"
	case strings.HasPrefix(f, ".github/") || f == ".golangci.yml" ||
		f == "scripts/ci/vulncheck.sh" || f == "scripts/ci/navalint.sh" || f == "scripts/ci/lint-fixture.sh":
		return lint, "lint configuration, lint script or workflows"
	case f == "scripts/ci/test-integration.sh" || strings.HasPrefix(f, "scripts/ci/pgbouncer/"):
		return []string{"ci-lint", "ci-integration"}, "integration script"
	case strings.HasPrefix(f, "deploy/") || strings.HasPrefix(path.Base(f), "compose") || f == ".dockerignore":
		return []string{"ci-lint", "ci-stack"}, "compose stack"
	}
	if pkg, ok := packageOf(f, g.packages); ok {
		s := []string{"ci-lint", "ci-unit"}
		var why []string
		if g.integration[pkg] {
			s = append(s, "ci-integration")
			why = append(why, "integration tests depend on it")
		}
		if g.stack[pkg] {
			s = append(s, "ci-stack")
			why = append(why, "an image or stack test depends on it")
		}
		return s, "package " + pkg + strings.Join(append([]string{""}, why...), "; ")
	}
	return allStages, "no rule for this file, so every stage"
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
	mb, err := git("merge-base", base, "HEAD")
	if err != nil {
		return nil, fmt.Errorf("%w (fetch %s first)", err, base)
	}
	diff, err := git("diff", "--name-only", strings.TrimSpace(mb))
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

func loadGraph() (graph, error) {
	root, err := git("rev-parse", "--show-toplevel")
	if err != nil {
		return graph{}, err
	}
	root = strings.TrimSpace(root)
	g := graph{}
	if g.packages, err = dirs(root, "-tags", "integration,temporalstack,identitystack", "./..."); err != nil {
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
	for _, args := range stackRoots {
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
	cmd := exec.Command("go", append([]string{"list", "-f", "{{if not .Standard}}{{.Dir}}{{end}}"}, args...)...)
	cmd.Dir = root
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("go list %s: %v\n%s", strings.Join(args, " "), err, stderr.String())
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
		b, err := os.ReadFile(f)
		if err == nil && integrationTag.Match(b) {
			return true
		}
	}
	return false
}

func git(args ...string) (string, error) {
	var stderr bytes.Buffer
	cmd := exec.Command("git", args...)
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %v: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return string(out), nil
}

package main

// TICK-1167: the installer recipe (tools-govulncheck) is pinned like the two
// scan recipes, and the environment the pinned recipes run in is the one an
// empty makefile gives them. Every case runs real make on a temporary
// Makefile, and each was red against the TICK-1149 head (bbe8631b).

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// pinnedMakefile is a Makefile with the three pinned rules as make/lint.mk
// and Makefile write them.
const pinnedMakefile = "GOVULNCHECK_VERSION := v1.9.0\n" +
	"vulncheck:\n\tGOWORK=off govulncheck ./...\n" +
	"vulncheck-temporal-server:\n\tcd temporal-server && GOWORK=off govulncheck ./...\n" +
	"tools-govulncheck:\n\tgo install golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)\n"

const installerLine = "\tgo install golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)\n"

// makefileCase is one edit of pinnedMakefile and a substring one of the
// check's problems must contain.
type makefileCase struct {
	name string
	mk   string
	want string
}

func withInstaller(line string) string {
	return strings.Replace(pinnedMakefile, installerLine, "\t"+line+"\n", 1)
}

func checkMakefile(t *testing.T, mk string) []string {
	t.Helper()
	if _, err := exec.LookPath("make"); err != nil {
		t.Skip("no make on PATH")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Makefile"), []byte(mk), 0o600); err != nil {
		t.Fatal(err)
	}
	problems, err := checkCopy(repo(repoWorkflow), repo(repoCopy), makeIn(dir))
	if err != nil {
		t.Fatal(err)
	}
	return problems
}

func runMakefileCases(t *testing.T, cases []makefileCase) {
	t.Helper()
	t.Run("pinned", func(t *testing.T) {
		t.Parallel()
		if problems := checkMakefile(t, pinnedMakefile); len(problems) > 0 {
			t.Fatalf("want none, got %q", problems)
		}
	})
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if c.mk == pinnedMakefile {
				t.Fatal("the edit changed nothing")
			}
			problems := checkMakefile(t, c.mk)
			if !slices.ContainsFunc(problems, func(p string) bool { return strings.Contains(p, c.want) }) {
				t.Fatalf("want a problem containing %q, got %q", c.want, problems)
			}
		})
	}
}

// TICK-1167 AC 2 a-e: an edit of the installer recipe alone fails.
func TestCopyCatchesInstallerEditsWithMake(t *testing.T) {
	stub := `mkdir -p "$$(go env GOPATH)/bin" && printf '#!/bin/sh\nexit 0\n' >"$$(go env GOPATH)/bin/govulncheck" && chmod +x "$$(go env GOPATH)/bin/govulncheck"`
	runMakefileCases(t, []makefileCase{
		{"a module path changed", withInstaller("go install example.com/fork/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)"), "make -pn tools-govulncheck: recipe is"},
		{"b version @latest in the recipe", withInstaller("go install golang.org/x/vuln/cmd/govulncheck@latest"), "make -pn tools-govulncheck: recipe is"},
		{"b version @latest by the variable", pinnedMakefile + "GOVULNCHECK_VERSION := latest\n", "make -n tools-govulncheck prints"},
		{"b version literal, not the variable", withInstaller("go install golang.org/x/vuln/cmd/govulncheck@v1.9.0"), "make -pn tools-govulncheck: recipe is"},
		{"c dash prefix", withInstaller("-go install golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)"), "make -pn tools-govulncheck: recipe is"},
		{"d .IGNORE: tools-govulncheck", ".IGNORE: tools-govulncheck\n" + pinnedMakefile, `make has ".IGNORE: tools-govulncheck"`},
		{"e a stub govulncheck", withInstaller(stub), "make -pn tools-govulncheck: recipe is"},
	})
}

// TICK-1167, added 2026-11-26 (PR #927 re-review): a one-line edit that
// changes the environment of a pinned recipe fails.
func TestCopyCatchesEnvironmentEditsWithMake(t *testing.T) {
	cases := []makefileCase{
		{"export GOOS", "export GOOS := plan9\n" + pinnedMakefile, `GOOS`},
		{"export GOFLAGS", "export GOFLAGS := -tags=nothing\n" + pinnedMakefile, `GOFLAGS`},
		{"export GOVULNDB", "export GOVULNDB := https://vulndb.example\n" + pinnedMakefile, `GOVULNDB`},
		{"GOFLAGS then a bare export", "GOFLAGS := -tags=nothing\nexport GOFLAGS\n" + pinnedMakefile, `GOFLAGS`},
		{"GOOS assigned, not exported", "GOOS := plan9\n" + pinnedMakefile, `make assigns GOOS`},
		{"export GOPROXY", "export GOPROXY := https://proxy.example\n" + pinnedMakefile, `GOPROXY`},
		{"PATH assigned", "PATH := /tmp/fake\n" + pinnedMakefile, `PATH`},
		{"export HOME", "export HOME := /tmp/fake\n" + pinnedMakefile, `recipes get HOME`},
		{"export of another name", "export CGO_ENABLED := 0\n" + pinnedMakefile, `recipes get CGO_ENABLED`},
		{".EXPORT_ALL_VARIABLES", ".EXPORT_ALL_VARIABLES:\n" + pinnedMakefile, `.EXPORT_ALL_VARIABLES`},
		{"GNUMAKEFLAGS -i", "GNUMAKEFLAGS := -i\n" + pinnedMakefile, "make sets MAKEFLAGS to"},
		{"pattern-specific export GOFLAGS", "vuln%: export GOFLAGS := -tags=nothing\n" + pinnedMakefile, "pattern-specific variable"},
		{"pattern-specific PATH on every target", "%: PATH := /tmp/fake\n" + pinnedMakefile, "pattern-specific variable"},
	}
	for _, r := range pinnedRecipes() {
		target := r.target
		cases = append(cases,
			makefileCase{target + "/export GOFLAGS", target + ": export GOFLAGS := -tags=nothing\n" + pinnedMakefile, "make -pn " + target + ": target-specific variable"},
			makefileCase{target + "/PATH", target + ": PATH := /tmp/fake\n" + pinnedMakefile, "make -pn " + target + ": target-specific variable"},
			makefileCase{target + "/GOVULNCHECK_VERSION", target + ": GOVULNCHECK_VERSION := latest\n" + pinnedMakefile, "make -pn " + target + ": target-specific variable"},
		)
	}
	runMakefileCases(t, cases)
}

// The pinned installer version is the one docs/pins.md and Makefile name: a
// version bump that misses this file fails here, not only in CI.
func TestPinnedGovulncheckVersionIsDocsPins(t *testing.T) {
	pins := readRepo(t, "docs/pins.md")
	if !strings.Contains(pins, "| `govulncheck` (`golang.org/x/vuln`) | "+govulncheckVersion+" |") {
		t.Errorf("docs/pins.md does not pin govulncheck %s; update govulncheckVersion with it", govulncheckVersion)
	}
	if mk := readRepo(t, "Makefile"); !strings.Contains(mk, "\nGOVULNCHECK_VERSION := "+govulncheckVersion+"\n") {
		t.Errorf("Makefile's GOVULNCHECK_VERSION is not %s; update govulncheckVersion with it", govulncheckVersion)
	}
}

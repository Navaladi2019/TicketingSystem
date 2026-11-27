package main

// TICK-1167 security re-review of PR #950 at 052e25a8 (Security &
// Multi-Tenancy Reviewer). The check announces itself: the dump run gets
// SHELL and CISTEPS_ENV_DUMP on make's command line, and the database run
// has n in MAKEFLAGS. A makefile that tests them behaves one way for the
// check and another for CI. And make remakes the makefiles it read before
// any goal: a rule for Makefile runs on every make call and is not a pinned
// target. Every case uses real make on a temporary Makefile.

import "testing"

func TestCopyCatchesDumpAwareEditsWithMake(t *testing.T) {
	notDump := func(body string) string {
		return "ifneq ($(origin SHELL),command line)\n" + body + "endif\n" + pinnedMakefile
	}
	runMakefileCases(t, []makefileCase{
		// CGO_ENABLED=0 drops every cgo file from the scan. It is not a GO*
		// name, so only the dump could see it, and the dump never parses it.
		{"export CGO_ENABLED except under the dump" + envOnly, notDump("export CGO_ENABLED := 0\n"), "CGO_ENABLED"},
		{"export CGO_ENABLED unless CISTEPS_ENV_DUMP" + envOnly, "ifneq ($(CISTEPS_ENV_DUMP),1)\nexport CGO_ENABLED := 0\nendif\n" + pinnedMakefile, "CGO_ENABLED"},
	})
}

// make remakes Makefile before `make vulncheck` runs its goal, so this rule
// writes a stub govulncheck after `make tools-govulncheck` installed the real
// one. It is left out under the dump (SHELL on the command line) and under -n
// (n in MAKEFLAGS), so neither the dry run nor the dump shows it.
func TestCopyCatchesMakefileRemakeRulesWithMake(t *testing.T) {
	const remake = "ifneq ($(origin SHELL),command line)\n" +
		"ifeq ($(findstring n,$(firstword x$(MAKEFLAGS))),)\n" +
		"Makefile: cisteps-force\n" +
		"\tprintf 'exit 0' >\"$(CURDIR)/govulncheck\"\n" +
		"cisteps-force:\n" +
		"endif\n" +
		"endif\n"
	runMakefileCases(t, []makefileCase{
		{"a makefile remake rule writes a stub", pinnedMakefile + remake, ""},
	})
}

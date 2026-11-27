package main

// TICK-1167 security review of PR #950 (Security & Multi-Tenancy Reviewer,
// commit 0044896). Each case is a one-file edit of a makefile that changes
// what a scan job runs. At the PR #950 head (fb2992c7) the check reported
// nothing for each: it built its database and its environment with the goals
// cisteps-noop and cisteps-env, not with the goals CI runs, and it did not
// look at what a makefile runs at parse time. Every case uses real make on a
// temporary Makefile.

import "testing"

func TestCopyCatchesGoalConditionalEditsWithMake(t *testing.T) {
	onVulncheck := func(body string) string {
		return "ifneq ($(filter vulncheck vulncheck-temporal-server tools-govulncheck,$(MAKECMDGOALS)),)\n" + body + "endif\n" + pinnedMakefile
	}
	runMakefileCases(t, []makefileCase{
		// `make vulncheck` runs with GOFLAGS=-tags=nothing.
		{"export GOFLAGS only for the CI goals", onVulncheck("export GOFLAGS := -tags=nothing\n"), "GOFLAGS"},
		// The target-specific variable exists only when vulncheck is a goal.
		{"target-specific PATH only for the CI goals", onVulncheck("vulncheck: PATH := /tmp/fake\n"), "vulncheck"},
		// Not a GO* or PATH name: only the environment of the real goal shows it.
		{"export HOME only for the CI goals" + envOnly, onVulncheck("export HOME := /tmp/fake\n"), "recipes get HOME"},
	})
}

// AR-TICK-1167-01 (docs/security/accepted-risks.md): an edit that acts only
// when a variable CI alone sets matches, such as the scan jobs' GITHUB_JOB,
// is not seen by a check that runs in the lint job. TICK-1183 (the workflow
// calls govulncheck directly with a pinned environment) closes it; until
// then RS3 review is the control.
func TestCopyCatchesCIVariableConditionalEditsWithMake(t *testing.T) {
	t.Skip("AR-TICK-1167-01: a makefile edit conditional on $(GITHUB_JOB) is not caught by check-ci-steps; closed by TICK-1183")
	runMakefileCases(t, []makefileCase{
		{"export GOFLAGS only in the scan jobs", "ifneq ($(filter vulncheck-root vulncheck-temporal-server,$(GITHUB_JOB)),)\nexport GOFLAGS := -tags=nothing\nendif\n" + pinnedMakefile, "GOFLAGS"},
	})
}

// A makefile line runs at parse time on every make call, including
// `make vulncheck` after `make tools-govulncheck`. $(file ...) writes a stub
// govulncheck (here into the test's directory) without leaving a variable,
// a rule or a recipe in make's database.
func TestCopyCatchesParseTimeWritesWithMake(t *testing.T) {
	runMakefileCases(t, []makefileCase{
		{"a stub govulncheck written at parse time", "$(file >$(CURDIR)/govulncheck,#!/bin/sh)\n" + pinnedMakefile, "govulncheck"},
	})
}

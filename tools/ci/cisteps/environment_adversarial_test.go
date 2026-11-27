package main

// TICK-1167 security review of PR #950 (Security & Multi-Tenancy Reviewer).
// Each case is a one-file edit of a makefile that changes what a scan job
// runs while the check, as written, reports nothing. The check builds its
// database and its environment with the goals cisteps-noop and cisteps-env,
// not with the goals CI runs, and it does not look at what a makefile runs at
// parse time. Every case uses real make on a temporary Makefile.

import "testing"

func TestCopyCatchesGoalConditionalEditsWithMake(t *testing.T) {
	onVulncheck := func(body string) string {
		return "ifneq ($(filter vulncheck vulncheck-temporal-server tools-govulncheck,$(MAKECMDGOALS)),)\n" + body + "endif\n" + pinnedMakefile
	}
	runMakefileCases(t, []makefileCase{
		// `make vulncheck` runs with GOFLAGS=-tags=nothing; `make -pn ...
		// cisteps-noop` and the cisteps-env recipe never parse the export.
		{"export GOFLAGS only for the CI goals", onVulncheck("export GOFLAGS := -tags=nothing\n"), "GOFLAGS"},
		// The target-specific variable exists only when vulncheck is a goal,
		// so recipeInDatabase never sees it.
		{"target-specific PATH only for the CI goals", onVulncheck("vulncheck: PATH := /tmp/fake\n"), "vulncheck"},
		// The same with the job name CI gives the scan jobs: the lint job's
		// GITHUB_JOB is "lint", so the check runs without the edit.
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

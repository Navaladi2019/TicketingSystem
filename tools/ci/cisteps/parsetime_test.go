package main

// TICK-1167 merge condition 3 (Technical Lead, 2026-11-27): make runs some
// makefile text while it reads the makefile, on every make call, including
// `make vulncheck` right after `make tools-govulncheck`. Such a line leaves no
// variable, rule or recipe in make's database, so only the makefile text
// shows it. Each case is one parse-time construct outside the allow-list.

import "testing"

func TestCopyCatchesParseTimeCodeWithMake(t *testing.T) {
	runMakefileCases(t, []makefileCase{
		{"$(shell", "STAMP := $(shell echo stub)\n" + pinnedMakefile, "runs at parse time"},
		{"${shell", "STAMP := ${shell echo stub}\n" + pinnedMakefile, "runs at parse time"},
		{"$(call shell", "STAMP := $(call shell,echo stub)\n" + pinnedMakefile, "runs at parse time"},
		{"$(call of a computed name", "F := shell\nSTAMP := $(call $(F),echo stub)\n" + pinnedMakefile, "runs at parse time"},
		{"$(file", "$(file >$(CURDIR)/stub,#!/bin/sh)\n" + pinnedMakefile, "runs at parse time"},
		{"$(eval", "$(eval STAMP := 1)\n" + pinnedMakefile, "runs at parse time"},
		{"$(guile", "STAMP := $(guile #t)\n" + pinnedMakefile, "runs at parse time"},
		{"!= assignment", "STAMP != echo stub\n" + pinnedMakefile, "runs at parse time"},
		{"load", "-load ./cisteps-none.so\n" + pinnedMakefile, "runs at parse time"},
		{"include", withFile("include extra.mk\n"+pinnedMakefile, "extra.mk", "X := 1\n"), "include"},
		{"-include", "-include cisteps-none.mk\n" + pinnedMakefile, "include"},
		{"sinclude", "sinclude cisteps-none.mk\n" + pinnedMakefile, "include"},
		{"$(shell on a continued line", "STAMP := a \\\n  $(shell echo stub)\n" + pinnedMakefile, "runs at parse time"},
		{"a GNUmakefile beside the Makefile", withFile(pinnedMakefile, "GNUmakefile", "include Makefile\n"), "GNUmakefile"},
	})
}

// $$( is the shell's command substitution in a recipe, not make's, and make
// expands no comment line: both pass.
func TestParseTimeScanSkipsShellSubstitution(t *testing.T) {
	if problems := checkMakefile(t, "# uses no $(shell or $(file\n"+pinnedMakefile+"other:\n\techo $$(go env GOPATH) $$(shell)\n"); len(problems) > 0 {
		t.Fatalf("want none, got %q", problems)
	}
}

// make expands a recipe line, # and all, when it runs the recipe.
func TestParseTimeScanReadsRecipeComments(t *testing.T) {
	problems, _, err := scanMakefile(".", "Makefile", []byte("other:\n\t# $(shell echo stub)\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(problems) != 1 {
		t.Fatalf("want one problem, got %q", problems)
	}
}

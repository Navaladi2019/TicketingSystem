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
		// PR #950 re-review (Principal Software Engineer, 052e25a8): a # line
		// inside define is part of the value, and make expands it with the
		// variable.
		{"# line in a define body", "define STAMP\n# $(shell echo stub)\nendef\nX := $(STAMP)\n" + pinnedMakefile, "runs at parse time"},
		{"# line in an override define body", "override define STAMP\n  # $(shell echo stub)\nendef\nX := $(STAMP)\n" + pinnedMakefile, "runs at parse time"},
		{"# line in a nested define body", "define OUTER\ndefine INNER\nendef\n# $(shell echo stub)\nendef\nX := $(OUTER)\n" + pinnedMakefile, "runs at parse time"},
		// A tab-indented line with no rule open is makefile text, not a
		// recipe: make reads its include, load or != assignment.
		{"tab-indented include after a variable", withFile("X := 1\n\tinclude extra.mk\n"+pinnedMakefile, "extra.mk", "X := 1\n"), "include"},
		{"tab-indented load after a variable", "X := 1\n\t-load ./cisteps-none.so\n" + pinnedMakefile, "runs at parse time"},
		{"tab-indented != after a variable", "X := 1\n\tSTAMP != echo stub\n" + pinnedMakefile, "runs at parse time"},
		{"tab-indented != after a target-specific variable", "other: X := 1\n\tSTAMP != echo stub\n" + pinnedMakefile, "runs at parse time"},
		// The scan reads a tab as the recipe prefix.
		{".RECIPEPREFIX changed", ".RECIPEPREFIX := >\n" + pinnedMakefile, ".RECIPEPREFIX"},
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

// A recipe's != is the shell's test, also after a blank line, a comment or a
// conditional inside the rule, and a define body without parse-time code
// passes.
func TestParseTimeScanSkipsRecipeTests(t *testing.T) {
	mk := "define BANNER\n# a comment in the value\nendef\n" + pinnedMakefile +
		"other:\n\t[ a != b ]\n\n# note\nifeq (1,1)\n\t[ c != d ]\nendif\n"
	problems, _, err := scanMakefile(".", "Makefile", []byte(mk))
	if err != nil {
		t.Fatal(err)
	}
	if len(problems) > 0 {
		t.Fatalf("want none, got %q", problems)
	}
}

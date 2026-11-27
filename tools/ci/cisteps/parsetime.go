package main

// The parse-time scan (TICK-1167 merge condition 3, Technical Lead
// 2026-11-27). make runs some makefile text while it reads the makefile, on
// every make call: $(shell ...), $(file ...), $(eval ...), $(guile ...), a !=
// assignment, a load directive, and the makefiles an include names. Such a
// line can overwrite the govulncheck that tools-govulncheck installed, right
// before `make vulncheck` runs it, and leaves no variable, rule or recipe in
// make's database. So this reads the makefile text: Makefile, and the files
// its allowed include lines name. Any such construct not on the allow-list
// fails the check. This is a static list of syntax, not a sandbox: no further
// rules are added here (TICK-1183 is the durable fix).

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// allowedParseTime is a makefile line, as written but for its surrounding
// space, that may run code at parse time.
type allowedParseTime struct{ file, line string }

// allowedParseTimeLines is the allow-list: the include of make/*.mk, and the
// compose profiles of make/stack.mk (line 74 when this was written).
func allowedParseTimeLines() []allowedParseTime {
	return []allowedParseTime{
		{"Makefile", "include make/*.mk"},
		{"make/stack.mk", "COMPOSE_PROFILES := $(shell STACK_TIER=$(STACK_TIER) AREAS=$(AREAS) sh deploy/compose/test/run-cases.sh profiles $(JOB) || echo PROFILES-ERROR)"},
	}
}

var (
	// includeLine is an include directive: include, -include, sinclude.
	includeLine = regexp.MustCompile(`^\s*(-include|sinclude|include)(\s|$)`)
	// loadLine is a load directive: load, -load.
	loadLine = regexp.MustCompile(`^\s*-?load(\s|$)`)
)

// parseTimeFuncs are the make functions that run something when expanded.
func parseTimeFuncs() []string { return []string{"shell", "file", "eval", "guile"} }

// parseTimeProblems returns the parse-time code of the makefiles in dir ("" is
// the current directory) that the allow-list does not name.
func parseTimeProblems(dir string) ([]string, error) {
	if dir == "" {
		dir = "."
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var problems []string
	for _, e := range entries {
		// make reads GNUmakefile, then makefile, before Makefile.
		if n := e.Name(); n == "GNUmakefile" || n == "makefile" {
			problems = append(problems, fmt.Sprintf("%s: make reads it instead of Makefile; the makefile is Makefile alone", n))
		}
	}
	seen := map[string]bool{}
	queue := []string{"Makefile"}
	for len(queue) > 0 {
		rel := queue[0]
		queue = queue[1:]
		if seen[rel] {
			continue
		}
		seen[rel] = true
		data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(rel))) //nolint:gosec // a makefile of the caller's directory
		if err != nil {
			return nil, err
		}
		p, includes, err := scanMakefile(dir, rel, data)
		if err != nil {
			return nil, err
		}
		problems = append(problems, p...)
		queue = append(queue, includes...)
	}
	return problems, nil
}

// scanMakefile returns the problems of one makefile, rel, and the files its
// allowed include lines name.
func scanMakefile(dir, rel string, data []byte) ([]string, []string, error) {
	var problems, includes []string
	lines := bytes.Split(bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n")), []byte("\n"))
	for i := 0; i < len(lines); i++ {
		// A logical line: backslash-newline continues it.
		n := i + 1
		text := string(lines[i])
		for strings.HasSuffix(text, `\`) && i+1 < len(lines) {
			i++
			text = text[:len(text)-1] + " " + string(lines[i])
		}
		trimmed := strings.TrimSpace(text)
		if slices.Contains(allowedParseTimeLines(), allowedParseTime{rel, trimmed}) {
			if m := includeLine.FindStringSubmatch(trimmed); m != nil {
				names, err := includedFiles(dir, strings.TrimSpace(trimmed[len(m[0]):]))
				if err != nil {
					return nil, nil, err
				}
				includes = append(includes, names...)
			}
			continue
		}
		recipe := strings.HasPrefix(text, "\t")
		if !recipe && strings.HasPrefix(trimmed, "#") {
			// make expands no comment; a recipe line's # is the shell's, and
			// make expands the line.
			continue
		}
		at := fmt.Sprintf("%s:%d", rel, n)
		for _, f := range parseTimeCalls(text) {
			problems = append(problems, fmt.Sprintf("%s: %s runs at parse time, on every make call: %q", at, f, trimmed))
		}
		if recipe {
			continue
		}
		switch {
		case includeLine.MatchString(text):
			problems = append(problems, fmt.Sprintf("%s: an include not on the allow-list: %q", at, trimmed))
		case loadLine.MatchString(text):
			problems = append(problems, fmt.Sprintf("%s: load runs at parse time, on every make call: %q", at, trimmed))
		case strings.Contains(text, "!="):
			problems = append(problems, fmt.Sprintf("%s: a != assignment runs at parse time, on every make call: %q", at, trimmed))
		}
	}
	return problems, includes, nil
}

// includedFiles are the files, relative to dir, that an include's patterns
// name, in the order make reads them.
func includedFiles(dir, patterns string) ([]string, error) {
	var names []string
	for p := range strings.FieldsSeq(patterns) {
		matches, err := filepath.Glob(filepath.Join(dir, filepath.FromSlash(p)))
		if err != nil {
			return nil, err
		}
		for _, m := range matches {
			r, err := filepath.Rel(dir, m)
			if err != nil {
				return nil, err
			}
			names = append(names, filepath.ToSlash(r))
		}
	}
	return names, nil
}

// parseTimeCalls are the references in text to a function that runs
// something: $(shell, ${shell, $(file, $(eval, $(guile, and a $(call of one of
// them or of a computed name. $$ is a literal $, so $$(cmd) in a recipe is
// the shell's command substitution, not make's.
func parseTimeCalls(text string) []string {
	var calls []string
	for i := 0; i+1 < len(text); i++ {
		if text[i] != '$' {
			continue
		}
		if text[i+1] == '$' {
			i++
			continue
		}
		if text[i+1] != '(' && text[i+1] != '{' {
			continue
		}
		name := funcName(text[i+2:])
		switch {
		case slices.Contains(parseTimeFuncs(), name):
			calls = append(calls, "$("+name)
		case name == "call":
			arg, _, _ := strings.Cut(strings.TrimLeft(text[i+2+len(name):], " \t"), ",")
			arg = strings.TrimSpace(arg)
			if slices.Contains(parseTimeFuncs(), arg) || strings.Contains(arg, "$") {
				calls = append(calls, "$(call "+arg)
			}
		}
	}
	return calls
}

// funcName is the word at the start of s, up to a space, a tab or a bracket.
func funcName(s string) string {
	end := strings.IndexAny(s, " \t(){}:,")
	if end < 0 {
		return s
	}
	return s[:end]
}

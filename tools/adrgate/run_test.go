package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

// 8. The lint contract: exit 1 and one `path: CI-n: message` line per finding
// on stdout; exit 0 and no output when clean.
func TestCLIExitCodesAndFormat(t *testing.T) {
	clean := base(t)
	dirty := base(t)
	write(t, dirty, "docs/adr/ADR-009-x.md", "---\ndate: 2026-11-20\n---\n# x\nM-X9\n")

	cases := []struct {
		name         string
		args         []string
		code         int
		stdout, errs string
	}{
		{"clean", []string{"-dir", clean}, 0, "", ""},
		{"findings", []string{"-dir", dirty}, 1,
			"docs/adr/ADR-009-x.md: CI-1: no ## Competitive Gate section (ADR-008 D6)\n" +
				"docs/adr/ADR-009-x.md: CI-4: M-X9 is not a method in ADR-008 \"The axes\"\n", ""},
		{"unknown flag", []string{"-nope"}, 2, "", "flag provided but not defined: -nope"},
		{"extra argument", []string{"-dir", clean, "x"}, 2, "", "adrgate: unexpected arguments: x"},
		{"help", []string{"-h"}, 0, "", "-dir"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if code := cli(c.args, &stdout, &stderr); code != c.code {
				t.Fatalf("exit %d, want %d; stdout %q stderr %q", code, c.code, stdout.String(), stderr.String())
			}
			if stdout.String() != c.stdout {
				t.Errorf("stdout %q, want %q", stdout.String(), c.stdout)
			}
			if c.errs == "" && stderr.Len() != 0 || !strings.Contains(stderr.String(), c.errs) {
				t.Errorf("stderr %q, want %q", stderr.String(), c.errs)
			}
		})
	}
}

// 7. The repository as it is passes: no existing record needs a change.
func TestRepositoryPasses(t *testing.T) {
	got, err := run("../..")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("findings on the repository:\n%s", strings.Join(got, "\n"))
	}
	// The real ADR-008 is parsed, not a fallback.
	_, methods, err := readGateADR(os.DirFS("../.."))
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range []string{"M-X1a", "M-X1b", "M-X2", "M-X3", "M-X4"} {
		if !methods[m] {
			t.Errorf("ADR-008 axes: %s missing from %v", m, methods)
		}
	}
}

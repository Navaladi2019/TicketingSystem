package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

// The gate: a duplicate id exits 1 and names the id and its lines.
func TestRunExitCodesAndMessages(t *testing.T) {
	clean := repo(t, "| TODO-001 | Open |\n", "| AR-001 | Low |\n")
	dup := repo(t, "| TODO-001 | Open |\n| TODO-001 | Fixed |\n", "| `AR-033` | Low |\n")
	missing := filepath.Join(t.TempDir(), "nope")
	cases := []struct {
		name string
		args []string
		code int
		want []string
	}{
		{"clean", []string{"-dir", clean}, 0, nil},
		{"duplicate and malformed", []string{"-dir", dup}, 1, []string{
			"dupids: docs/delivery/todo.md: id TODO-001 appears 2 times (lines 5, 6); give each row its own id\n",
			"dupids: docs/security/accepted-risks.md: line 5: malformed id cell \"`AR-033`\"; write the bare id, such as AR-S5-01\n",
		}},
		{"missing root", []string{"-dir", missing}, 2, []string{"dupids: ", "nope"}},
		{"unknown flag", []string{"-nope"}, 2, []string{"flag provided but not defined: -nope"}},
		{"help", []string{"-h"}, 0, []string{"-dir"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var stderr bytes.Buffer
			if code := run(c.args, &stderr); code != c.code {
				t.Fatalf("exit %d, want %d; stderr %q", code, c.code, stderr.String())
			}
			for _, w := range c.want {
				if !strings.Contains(stderr.String(), w) {
					t.Errorf("stderr %q lacks %q", stderr.String(), w)
				}
			}
			if c.want == nil && stderr.Len() != 0 {
				t.Errorf("stderr %q, want empty", stderr.String())
			}
		})
	}
}

func TestRootIsAFile(t *testing.T) {
	root := repo(t, "", "")
	got, err := check(filepath.Join(root, "docs", "delivery", "todo.md"))
	if err == nil || got != nil {
		t.Fatalf("check = %v, %v; want an error and no problems", got, err)
	}
}

func TestOverlongLineIsAnError(t *testing.T) {
	// A line longer than the scanner's 4 MiB buffer must fail, not be skipped.
	root := repo(t, "| TODO-001 | "+strings.Repeat("x", 5*1024*1024)+" |\n", "")
	got, err := check(root)
	if err == nil || !strings.Contains(err.Error(), "docs/delivery/todo.md: bufio.Scanner: token too long") {
		t.Fatalf("check = %v, %v; want a token-too-long error naming todo.md", got, err)
	}
	if got != nil {
		t.Errorf("problems %v on error, want none", got)
	}
}

func TestFirstCellAndRowID(t *testing.T) {
	for _, c := range []struct{ line, cell, id string }{
		{"| TODO-001 | Open |", "TODO-001", "TODO-001"},
		{"|  AR-001 (R8) | Low |", "AR-001 (R8)", "AR-001"},
		{"|", "", ""},
		{"not a row", "", ""},
		{"", "", ""},
	} {
		if got := firstCell(c.line); got != c.cell {
			t.Errorf("firstCell(%q) = %q, want %q", c.line, got, c.cell)
		}
		if got := rowID(c.line); got != c.id {
			t.Errorf("rowID(%q) = %q, want %q", c.line, got, c.id)
		}
	}
}

func TestJoinInts(t *testing.T) {
	if got := joinInts([]int{3, 10, 42}); got != "3, 10, 42" {
		t.Errorf("joinInts = %q", got)
	}
	if got := joinInts(nil); got != "" {
		t.Errorf("joinInts(nil) = %q", got)
	}
}

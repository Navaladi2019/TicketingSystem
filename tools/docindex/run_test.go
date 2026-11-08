package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The gate: -check on a stale index exits 1 with the regenerate hint.
func TestCLIExitCodesAndMessages(t *testing.T) {
	fresh := t.TempDir()
	write(t, fresh, logPath, logDoc)
	write(t, fresh, entriesDir+"/2026-10-02-first.md", "## first\n")
	if _, err := run(fresh, false); err != nil {
		t.Fatal(err)
	}
	stale := t.TempDir()
	write(t, stale, logPath, logDoc)
	write(t, stale, entriesDir+"/2026-10-02-first.md", "## first\n")
	bad := t.TempDir()
	write(t, bad, logPath, logDoc)
	write(t, bad, entriesDir+"/notes.md", "## a\n")

	cases := []struct {
		name string
		args []string
		code int
		want string
	}{
		{"fresh", []string{"-dir", fresh, "-check"}, 0, ""},
		{"stale", []string{"-dir", stale, "-check"}, 1, "docindex: the index in docs/delivery/decision-log.md is stale; run `go run ./tools/docindex`\n"},
		{"bad entry name", []string{"-dir", bad, "-check"}, 2, "docindex: notes.md: name must be <YYYY-MM-DD>-<slug>.md\n"},
		{"unknown flag", []string{"-nope"}, 2, "flag provided but not defined: -nope"},
		{"help", []string{"-h"}, 0, "-check"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var stderr bytes.Buffer
			if code := cli(c.args, &stderr); code != c.code {
				t.Fatalf("exit %d, want %d; stderr %q", code, c.code, stderr.String())
			}
			if c.want == "" && stderr.Len() != 0 {
				t.Errorf("stderr %q, want empty", stderr.String())
			}
			if !strings.Contains(stderr.String(), c.want) {
				t.Errorf("stderr %q lacks %q", stderr.String(), c.want)
			}
		})
	}

	// Without -check a stale index is rewritten, exits 0, and is then fresh.
	var stderr bytes.Buffer
	if code := cli([]string{"-dir", stale}, &stderr); code != 0 || stderr.Len() != 0 {
		t.Fatalf("rewrite: exit %d, stderr %q", code, stderr.String())
	}
	if code := cli([]string{"-dir", stale, "-check"}, &stderr); code != 0 {
		t.Fatalf("after rewrite: exit %d, stderr %q", code, stderr.String())
	}
}

func TestMissingInputsAreErrors(t *testing.T) {
	noLog := t.TempDir()
	write(t, noLog, entriesDir+"/2026-10-02-a.md", "## a\n")
	noEntries := t.TempDir()
	write(t, noEntries, logPath, logDoc)
	entriesIsFile := t.TempDir()
	write(t, entriesIsFile, logPath, logDoc)
	write(t, entriesIsFile, entriesDir, "not a directory\n")

	for name, root := range map[string]string{
		"no root":           filepath.Join(t.TempDir(), "nope"),
		"no log":            noLog,
		"no entries dir":    noEntries,
		"entries is a file": entriesIsFile,
	} {
		t.Run(name, func(t *testing.T) {
			stale, err := run(root, true)
			if err == nil || stale {
				t.Fatalf("stale=%v err=%v, want an error", stale, err)
			}
		})
	}
}

func TestSkipsReadmeAndSubdirsAndReadsH1(t *testing.T) {
	root := t.TempDir()
	write(t, root, logPath, logDoc)
	write(t, root, entriesDir+"/README.md", "no heading here\n")
	write(t, root, entriesDir+"/drafts/not-an-entry.txt", "x\n")
	write(t, root, entriesDir+"/2026-10-02-first.md", "intro\n#  Top title  \n## later\n")
	if _, err := run(root, false); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(logPath)))
	if err != nil {
		t.Fatal(err)
	}
	want := startMarker + "\n\n- [Top title](decisions/2026-10-02-first.md)\n\n" + endMarker
	if !strings.Contains(string(got), want) {
		t.Errorf("index:\n%s\nwant it to contain:\n%s", got, want)
	}
	if strings.Contains(string(got), "README") || strings.Contains(string(got), "drafts") {
		t.Errorf("index lists README or a subdirectory:\n%s", got)
	}
}

func TestEntryOutsideRootIsAnError(t *testing.T) {
	root := t.TempDir()
	write(t, root, logPath, logDoc)
	outside := filepath.Join(t.TempDir(), "outside.md")
	if err := os.WriteFile(outside, []byte("## outside\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, filepath.FromSlash(entriesDir))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(dir, "2026-10-02-link.md")); err != nil {
		t.Skipf("cannot create a symlink here: %v", err)
	}
	if _, err := run(root, true); err == nil || !strings.Contains(err.Error(), "2026-10-02-link.md") {
		t.Fatalf("err = %v, want an error naming the escaping entry", err)
	}
}

func TestUnwritableLogIsAnError(t *testing.T) {
	root := t.TempDir()
	write(t, root, logPath, logDoc)
	write(t, root, entriesDir+"/2026-10-02-first.md", "## first\n")
	p := filepath.Join(root, filepath.FromSlash(logPath))
	if err := os.Chmod(p, 0o444); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(p, 0o644) })
	if f, err := os.OpenFile(p, os.O_WRONLY, 0); err == nil {
		_ = f.Close()
		t.Skip("read-only mode is not enforced here (running as root?)")
	}
	stale, err := run(root, false)
	if err == nil || stale {
		t.Fatalf("stale=%v err=%v, want a write error", stale, err)
	}
	got, _ := os.ReadFile(p)
	if string(got) != logDoc {
		t.Errorf("log changed despite the error:\n%s", got)
	}
}

func TestHeading(t *testing.T) {
	for _, c := range []struct{ body, want string }{
		{"## a\n# b\n", "a"},
		{"# b\r\n## a\r\n", "b"},
		{"###  deep\n", ""},
		{"text\n#notaheading\n", ""},
		{"", ""},
	} {
		if got := heading(c.body); got != c.want {
			t.Errorf("heading(%q) = %q, want %q", c.body, got, c.want)
		}
	}
}

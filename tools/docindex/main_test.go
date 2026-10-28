package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func write(t *testing.T, root, rel, body string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

const logDoc = "# Decision log\n\n" + startMarker + "\n" + endMarker + "\n\n## 2026-10-01: old entry\n- kept\n"

func TestIndexIsNewestFirstAndKeepsOldEntries(t *testing.T) {
	root := t.TempDir()
	write(t, root, logPath, logDoc)
	write(t, root, entriesDir+"/2026-10-02-first.md", "## 2026-10-02: first\n")
	write(t, root, entriesDir+"/2026-10-05-second.md", "## 2026-10-05: second\r\n")

	if stale, err := run(root, false); err != nil || stale {
		t.Fatalf("run: stale=%v err=%v", stale, err)
	}
	got, _ := os.ReadFile(filepath.Join(root, filepath.FromSlash(logPath)))
	s := string(got)
	if strings.Index(s, "2026-10-05: second") > strings.Index(s, "2026-10-02: first") {
		t.Errorf("not newest first:\n%s", s)
	}
	if !strings.Contains(s, "(decisions/2026-10-05-second.md)") || !strings.Contains(s, "## 2026-10-01: old entry\n- kept") {
		t.Errorf("index or old entry wrong:\n%s", s)
	}
	if stale, err := run(root, true); err != nil || stale {
		t.Errorf("fresh index reported stale=%v err=%v", stale, err)
	}
}

func TestCheckFailsWhenIndexIsStale(t *testing.T) {
	root := t.TempDir()
	write(t, root, logPath, logDoc)
	write(t, root, entriesDir+"/2026-10-02-first.md", "## first\n")
	if _, err := run(root, false); err != nil {
		t.Fatal(err)
	}
	// A new entry file without regenerating the index.
	write(t, root, entriesDir+"/2026-10-03-added.md", "## added\n")
	stale, err := run(root, true)
	if err != nil || !stale {
		t.Fatalf("stale=%v err=%v, want stale", stale, err)
	}
	// -check must not rewrite the file.
	if stale2, _ := run(root, true); !stale2 {
		t.Error("check rewrote the log")
	}
}

func TestCheckFailsWhenIndexLineEdited(t *testing.T) {
	root := t.TempDir()
	write(t, root, logPath, logDoc)
	write(t, root, entriesDir+"/2026-10-02-first.md", "## first\n")
	if _, err := run(root, false); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(root, filepath.FromSlash(logPath))
	b, _ := os.ReadFile(p)
	_ = os.WriteFile(p, []byte(strings.Replace(string(b), "first", "renamed", 1)), 0o644)
	if stale, err := run(root, true); err != nil || !stale {
		t.Fatalf("stale=%v err=%v, want stale", stale, err)
	}
}

func TestBadInputs(t *testing.T) {
	cases := map[string]struct{ log, file, body string }{
		"no markers": {"# Decision log\n", "2026-10-02-a.md", "## a\n"},
		"bad name":   {logDoc, "notes.md", "## a\n"},
		"no heading": {logDoc, "2026-10-02-a.md", "text only\n"},
		"underscore": {logDoc, "2026-10-02_a.md", "## a\n"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			write(t, root, logPath, c.log)
			write(t, root, entriesDir+"/"+c.file, c.body)
			if _, err := run(root, true); err == nil {
				t.Error("want an error")
			}
		})
	}
}

func TestCRLFLogStaysCRLF(t *testing.T) {
	root := t.TempDir()
	write(t, root, logPath, strings.ReplaceAll(logDoc, "\n", "\r\n"))
	write(t, root, entriesDir+"/2026-10-02-first.md", "## first\n")
	if _, err := run(root, false); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(filepath.Join(root, filepath.FromSlash(logPath)))
	if strings.Contains(strings.ReplaceAll(string(got), "\r\n", ""), "\n") {
		t.Errorf("bare LF in a CRLF log: %q", got)
	}
}

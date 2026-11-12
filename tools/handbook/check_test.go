package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fakeGit struct {
	ancestor bool
	err      error
	asked    string
}

func (f *fakeGit) IsAncestor(sha string) (bool, error) {
	f.asked = sha
	return f.ancestor, f.err
}

func removeFile(root, rel string) error {
	return os.Remove(filepath.Join(root, filepath.FromSlash(rel)))
}

// readRepoFile reads a file of the real repository (two levels above this package).
func readRepoFile(rel string) ([]byte, error) {
	return os.ReadFile(filepath.Join("..", "..", filepath.FromSlash(rel)))
}

// pinFixture adds the docs/pins.md row the check compares mermaid.min.js with.
func pinFixture(t *testing.T, root string) {
	t.Helper()
	sum := sha256.Sum256([]byte("/* pinned */"))
	write(t, root, "docs/pins.md", "| `mermaid` (`docs/handbook/assets/mermaid.min.js`) | 12.3.0, sha256 `"+hex.EncodeToString(sum[:])+"` | x | y |\n")
}

// committed builds the fixture handbook with sha and writes it, like `task handbook`.
func committed(t *testing.T, root, sha string) {
	t.Helper()
	out, err := Build(Config{Root: root, Sha: sha, Date: "2026-11-11"})
	if err != nil {
		t.Fatal(err)
	}
	if err := Write(root, out); err != nil {
		t.Fatal(err)
	}
}

func TestCheckIsCleanAfterBuildEvenWhenMainMoves(t *testing.T) {
	root := newFixture(t)
	pinFixture(t, root)
	page(t, root, "system/a.md", "A", "system", "delivered", "", "text")
	committed(t, root, testSha)
	g := &fakeGit{ancestor: true}
	problems, err := Check(Config{Root: root}, g)
	if err != nil || len(problems) != 0 {
		t.Fatalf("clean tree: %v %v", problems, err)
	}
	if g.asked != testSha[:12] {
		t.Errorf("ancestry asked about %q, want the stamped sha", g.asked)
	}
	// main moved: the committed footer carries an older sha, and nothing else changed.
	committed(t, root, "1234567890abcdef1234567890abcdef12345678")
	if problems, err := Check(Config{Root: root}, &fakeGit{ancestor: true}); err != nil || len(problems) != 0 {
		t.Fatalf("a different stamp made the check stale: %v %v", problems, err)
	}
}

func TestCheckDetectsStaleMissingAndExtraFiles(t *testing.T) {
	root := newFixture(t)
	pinFixture(t, root)
	page(t, root, "system/a.md", "A", "system", "delivered", "", "text")
	committed(t, root, testSha)

	page(t, root, "system/a.md", "A", "system", "delivered", "", "edited without a rebuild")
	problems, err := Check(Config{Root: root}, &fakeGit{ancestor: true})
	if err != nil || !hasProblem(problems, "system/a.html is stale") || !hasProblem(problems, "search-index.js is stale") {
		t.Fatalf("stale source not reported: %v %v", problems, err)
	}

	committed(t, root, testSha)
	if err := removeFile(root, "docs/handbook/system/a.html"); err != nil {
		t.Fatal(err)
	}
	write(t, root, "docs/handbook/orphan.html", "x")
	problems, _ = Check(Config{Root: root}, &fakeGit{ancestor: true})
	if !hasProblem(problems, "system/a.html is missing") || !hasProblem(problems, "orphan.html is not generated") {
		t.Fatalf("missing or extra file not reported: %v", problems)
	}
}

func TestCheckRequiresStampToBeAncestor(t *testing.T) {
	root := newFixture(t)
	pinFixture(t, root)
	committed(t, root, testSha)
	problems, _ := Check(Config{Root: root}, &fakeGit{ancestor: false})
	if !hasProblem(problems, "is not an ancestor of HEAD") {
		t.Errorf("non-ancestor stamp accepted: %v", problems)
	}
	problems, _ = Check(Config{Root: root}, &fakeGit{err: errors.New("shallow clone")})
	if !hasProblem(problems, "shallow clone") {
		t.Errorf("git failure swallowed: %v", problems)
	}
}

func TestCheckComparesMermaidWithPin(t *testing.T) {
	root := newFixture(t)
	pinFixture(t, root)
	committed(t, root, testSha)
	write(t, root, "docs/handbook/assets/mermaid.min.js", "/* tampered */")
	problems, _ := Check(Config{Root: root}, &fakeGit{ancestor: true})
	if !hasProblem(problems, "docs/pins.md pins") {
		t.Errorf("mermaid sha mismatch not reported: %v", problems)
	}
	write(t, root, "docs/pins.md", "no row")
	problems, _ = Check(Config{Root: root}, &fakeGit{ancestor: true})
	if !hasProblem(problems, "no `mermaid` row") {
		t.Errorf("missing pin row not reported: %v", problems)
	}
}

func TestScannerHookSeesEveryArtefact(t *testing.T) {
	root := newFixture(t)
	pinFixture(t, root)
	page(t, root, "system/a.md", "A", "system", "delivered", "", "![x](../diagrams/ok.excalidraw.svg)")
	committed(t, root, testSha)
	seen := map[string]bool{}
	scan := func(rel string, data []byte) []string {
		seen[rel] = true
		if rel == "system/a.html" && bytes.Contains(data, []byte("Content generated")) {
			return []string{"rule X matched"}
		}
		return nil
	}
	problems, err := Check(Config{Root: root, Scanner: scan}, &fakeGit{ancestor: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"index.html", "system/a.html", "assets/search-index.js", "assets/mermaid.min.js", "src/database/schema.json", "src/diagrams/ok.excalidraw.svg", "diagrams/ok.excalidraw.svg"} {
		if !seen[want] {
			t.Errorf("scanner never saw %s", want)
		}
	}
	if !hasProblem(problems, "sensitivity scan: system/a.html: rule X matched") {
		t.Errorf("scanner finding not reported: %v", problems)
	}
	// The shipped default has no rules.
	if got := noRules("x", []byte("password=hunter2")); len(got) != 0 {
		t.Errorf("default scanner has rules: %v", got)
	}
}

func hasProblem(problems []string, sub string) bool {
	for _, p := range problems {
		if strings.Contains(p, sub) {
			return true
		}
	}
	return false
}

func TestWriteRemovesStaleGeneratedFilesButKeepsSourcesAndPin(t *testing.T) {
	root := newFixture(t)
	page(t, root, "system/a.md", "A", "system", "delivered", "", "x")
	committed(t, root, testSha)
	write(t, root, "docs/handbook/README.md", "readme")
	write(t, root, "docs/handbook/gone/old.html", "stale")
	committed(t, root, testSha)
	for _, keep := range []string{"README.md", "src/index.md", "assets/mermaid.min.js", "system/a.html"} {
		if _, err := os.Stat(filepath.Join(root, "docs", "handbook", filepath.FromSlash(keep))); err != nil {
			t.Errorf("%s was removed: %v", keep, err)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "docs", "handbook", "gone", "old.html")); err == nil {
		t.Errorf("stale generated file kept")
	}
}

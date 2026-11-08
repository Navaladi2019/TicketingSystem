package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The real repository against HEAD: the command prints the stages classify
// picks for the uncommitted files, and with -v one reason per file.
func testCLIRealRepo(t *testing.T, g graph) {
	files, err := changedFiles("HEAD")
	if err != nil {
		t.Fatal(err)
	}
	stages, reasons := classify(files, g)
	var stdout, stderr strings.Builder
	if code := cli([]string{"-base", "HEAD", "-v"}, &stdout, &stderr); code != 0 {
		t.Fatalf("exit %d, stderr %q", code, stderr.String())
	}
	if got, want := stdout.String(), strings.Join(stages, " ")+"\n"; got != want {
		t.Errorf("stdout %q, want %q", got, want)
	}
	if !strings.HasPrefix(stdout.String(), "ci-lint") {
		t.Errorf("stdout %q does not start with ci-lint", stdout.String())
	}
	var want strings.Builder
	for _, r := range reasons {
		want.WriteString("cichanged: " + r + "\n")
	}
	if stderr.String() != want.String() {
		t.Errorf("stderr %q, want %q", stderr.String(), want.String())
	}
}

func TestCLIBadFlag(t *testing.T) {
	var stdout, stderr strings.Builder
	if code := cli([]string{"-nope"}, &stdout, &stderr); code != 2 {
		t.Fatalf("exit %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "flag provided but not defined: -nope") {
		t.Errorf("stderr %q", stderr.String())
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout %q, want nothing", stdout.String())
	}
}

// tempRepo makes a git repository with one commit of files in a temp dir,
// and changes into it.
func tempRepo(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("GIT_CEILING_DIRECTORIES", filepath.Dir(dir))
	t.Setenv("GOFLAGS", "")
	t.Setenv("GOWORK", "off")
	t.Setenv("GOPROXY", "off")
	t.Chdir(dir)
	for name, body := range files {
		if err := os.MkdirAll(filepath.Dir(name), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(name, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	for _, args := range [][]string{
		{"init", "-q", "-b", "main"},
		{"config", "user.email", "ci@example.invalid"},
		{"config", "user.name", "ci"},
		{"add", "."},
		{"commit", "-q", "--allow-empty", "-m", "base"},
	} {
		if _, err := git(args...); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// Every failure of git or go list stops the command with exit 2 and the
// failing command on stderr; nothing is printed on stdout, so ci-changed
// never runs a partial stage list.
func TestCLIFailures(t *testing.T) {
	const integrationTest = "//go:build integration\n\npackage a\n\nimport (\n\t\"testing\"\n\t%s\n)\n\nfunc TestA(t *testing.T) {}\n"
	for _, c := range []struct {
		name  string
		files map[string]string
		base  string
		want  string
	}{
		{
			name: "unknown base",
			base: "nope",
			want: "cichanged: git merge-base nope HEAD: exit status",
		},
		{
			name:  "not a Go module",
			files: map[string]string{"README.md": "x\n"},
			base:  "HEAD",
			want:  "cichanged: go list -tags integration,",
		},
		{
			name: "integration test with a missing import",
			files: map[string]string{
				"go.mod":      "module example.invalid/m\n\ngo 1.26\n",
				"a/a.go":      "package a\n",
				"a/a_test.go": strings.Replace(integrationTest, "%s", `_ "example.invalid/nope"`, 1),
			},
			base: "HEAD",
			want: "cichanged: go list -deps -test -tags integration ./a:",
		},
		{
			name: "stack root missing",
			files: map[string]string{
				"go.mod":      "module example.invalid/m\n\ngo 1.26\n",
				"a/a.go":      "package a\n",
				"a/a_test.go": strings.Replace(integrationTest, "%s", `_ "strings"`, 1),
			},
			base: "HEAD",
			want: "cichanged: go list -deps ./cmd/nava-core",
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			tempRepo(t, c.files)
			var stdout, stderr strings.Builder
			if code := cli([]string{"-base", c.base}, &stdout, &stderr); code != 2 {
				t.Fatalf("exit %d, want 2; stdout %q", code, stdout.String())
			}
			if !strings.HasPrefix(stderr.String(), c.want) {
				t.Errorf("stderr %q, want prefix %q", stderr.String(), c.want)
			}
			if stdout.Len() != 0 {
				t.Errorf("stdout %q, want nothing", stdout.String())
			}
		})
	}
}

// The merge-base error tells the user to fetch the base.
func TestChangedFilesUnknownBase(t *testing.T) {
	tempRepo(t, nil)
	_, err := changedFiles("origin/main")
	if err == nil || !strings.HasSuffix(err.Error(), "(fetch origin/main first)") {
		t.Fatalf("err %v, want the fetch hint", err)
	}
}

// A broken index fails git diff after merge-base succeeded.
func TestChangedFilesDiffFails(t *testing.T) {
	dir := tempRepo(t, map[string]string{"a.txt": "a\n"})
	bad := filepath.Join(dir, "bad-index")
	if err := os.WriteFile(bad, []byte("not an index file"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_INDEX_FILE", bad)
	_, err := changedFiles("HEAD")
	if err == nil || !strings.HasPrefix(err.Error(), "git diff --name-only --no-renames ") {
		t.Fatalf("err %v, want the git diff failure", err)
	}
}

// Outside a repository there is no root to list packages from.
func TestLoadGraphOutsideARepository(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GIT_CEILING_DIRECTORIES", filepath.Dir(dir))
	t.Chdir(dir)
	_, err := loadGraph()
	if err == nil || !strings.HasPrefix(err.Error(), "git rev-parse --show-toplevel: ") {
		t.Fatalf("err %v, want the git rev-parse failure", err)
	}
}

// Committed, uncommitted and untracked changes are listed once each, sorted.
func TestChangedFilesListsEachFileOnce(t *testing.T) {
	tempRepo(t, map[string]string{"b.txt": "b\n", "a.txt": "a\n"})
	if _, err := git("checkout", "-q", "-b", "change"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("b.txt", []byte("b2\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := git("commit", "-q", "-am", "b2"); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{"b.txt": "b3\n", "a.txt": "a2\n", "c.txt": "c\n"} {
		if err := os.WriteFile(name, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	files, err := changedFiles("main")
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(files, " "); got != "a.txt b.txt c.txt" {
		t.Fatalf("changed files = %q", got)
	}
}

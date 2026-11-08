package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// testPaths is a small path list: db/ needs Security review, the catalog
// policy and every *isolation*_test.go need CTO sign-off.
const testPaths = `security:
  - db/
cto_signoff:
  paths:
    - internal/db/catalog/policy.go
  isolation_tests:
    - "**/*isolation*_test.go"
`

// gitRepo makes a throwaway repository with a commit tagged "base" and one
// commit on top of it that adds db/m.sql, skips x/a_isolation_test.go,
// deletes y/b_isolation_test.go, extends z/c_isolation_test.go and edits
// README.md. It isolates git from the host's config and returns the repo dir.
func gitRepo(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	global := filepath.Join(home, "gitconfig")
	if err := os.WriteFile(global, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", global)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	for _, k := range []string{"GIT_AUTHOR", "GIT_COMMITTER"} {
		t.Setenv(k+"_NAME", "test")
		t.Setenv(k+"_EMAIL", "test@example.invalid")
	}
	repo := t.TempDir()
	g := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...) //nolint:noctx // runs to completion
		cmd.Dir = repo
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %s: %v: %s", strings.Join(args, " "), err, out)
		}
	}
	write := func(name, content string) {
		t.Helper()
		p := filepath.Join(repo, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	g("init", "-q")
	write("README.md", "readme\n")
	write("x/a_isolation_test.go", "package x\n")
	write("y/b_isolation_test.go", "package y\n")
	write("z/c_isolation_test.go", "package z\n")
	g("add", "-A")
	g("commit", "-q", "-m", "base")
	g("tag", "base")
	write("README.md", "readme\nmore\n")
	write("db/m.sql", "select 1;\n")
	write("x/a_isolation_test.go", "package x\n\tt.Skip(\"flaky\")\n")
	write("z/c_isolation_test.go", "package z\n\tassertNoRows(t, rows)\n")
	g("rm", "-q", "y/b_isolation_test.go")
	g("add", "-A")
	g("commit", "-q", "-m", "change")
	return repo
}

func writePaths(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "review-paths.yaml")
	if err := os.WriteFile(p, []byte(testPaths), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

const wantReport = "security review required: yes\n" +
	"CTO sign-off required: yes\n" +
	"  security: db/m.sql\n" +
	"  CTO: x/a_isolation_test.go (isolation test skipped or excluded)\n" +
	"  CTO: y/b_isolation_test.go (isolation test removed)\n"

func TestCLIFlagsAChangeAgainstTheBase(t *testing.T) {
	repo := gitRepo(t)
	paths := writePaths(t)
	t.Chdir(repo)
	t.Setenv("GITHUB_STEP_SUMMARY", "")
	var stdout, stderr bytes.Buffer
	if code := cli([]string{"-base", "base", "-paths", paths}, &stdout, &stderr); code != 0 {
		t.Fatalf("exit %d, stderr: %s", code, stderr.String())
	}
	if stdout.String() != wantReport {
		t.Fatalf("stdout:\n%s\nwant:\n%s", stdout.String(), wantReport)
	}
	if stderr.Len() != 0 {
		t.Errorf("stderr: %s", stderr.String())
	}
}

func TestCLIAppendsTheReportToTheStepSummary(t *testing.T) {
	repo := gitRepo(t)
	paths := writePaths(t)
	summary := filepath.Join(t.TempDir(), "summary.md")
	if err := os.WriteFile(summary, []byte("earlier step\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(repo)
	t.Setenv("GITHUB_STEP_SUMMARY", summary)
	var stdout, stderr bytes.Buffer
	if code := cli([]string{"-base", "base", "-paths", paths}, &stdout, &stderr); code != 0 {
		t.Fatalf("exit %d, stderr: %s", code, stderr.String())
	}
	if stdout.String() != wantReport {
		t.Errorf("stdout:\n%s\nwant:\n%s", stdout.String(), wantReport)
	}
	got, err := os.ReadFile(summary)
	if err != nil {
		t.Fatal(err)
	}
	want := "earlier step\n### Review paths (D-P10)\n\n```\n" + wantReport + "```\n"
	if string(got) != want {
		t.Errorf("summary:\n%s\nwant:\n%s", got, want)
	}
}

func TestCLIReportsNothingForAnUnchangedTree(t *testing.T) {
	repo := gitRepo(t)
	paths := writePaths(t)
	t.Chdir(repo)
	t.Setenv("GITHUB_STEP_SUMMARY", "")
	var stdout, stderr bytes.Buffer
	if code := cli([]string{"-base", "HEAD", "-paths", paths}, &stdout, &stderr); code != 0 {
		t.Fatalf("exit %d, stderr: %s", code, stderr.String())
	}
	if want := "security review required: no\nCTO sign-off required: no\n"; stdout.String() != want {
		t.Errorf("stdout:\n%s\nwant:\n%s", stdout.String(), want)
	}
}

// The tool only flags, but it must not report "no" when it could not check:
// every failure to run is exit 2 with the reason on stderr.
func TestCLIFailsWhenItCannotCheck(t *testing.T) {
	repo := gitRepo(t)
	paths := writePaths(t)
	badYAML := filepath.Join(t.TempDir(), "bad.yaml")
	if err := os.WriteFile(badYAML, []byte("security:\n  - db/\nunknown: 1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	summaryDir := t.TempDir() // a directory cannot be opened for append
	for _, tc := range []struct {
		name    string
		args    []string
		summary string
		want    []string // substrings of stderr
	}{
		{"unknown flag", []string{"-nope"}, "", []string{"flag provided but not defined: -nope", "Usage of reviewpaths"}},
		{"missing path list", []string{"-base", "base", "-paths", filepath.Join(repo, "absent.yaml")}, "", []string{"review-paths: ", "absent.yaml"}},
		{"invalid path list", []string{"-base", "base", "-paths", badYAML}, "", []string{"review-paths: " + badYAML + ": ", "unknown"}},
		{"unknown base ref", []string{"-base", "no-such-ref", "-paths", paths}, "", []string{"review-paths: git diff --no-renames --name-status no-such-ref...HEAD: exit status"}},
		{"unwritable step summary", []string{"-base", "base", "-paths", paths}, summaryDir, []string{"review-paths: open " + summaryDir}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Chdir(repo)
			t.Setenv("GITHUB_STEP_SUMMARY", tc.summary)
			var stdout, stderr bytes.Buffer
			if code := cli(tc.args, &stdout, &stderr); code != 2 {
				t.Fatalf("exit %d, want 2; stdout: %s; stderr: %s", code, stdout.String(), stderr.String())
			}
			for _, w := range tc.want {
				if !strings.Contains(stderr.String(), w) {
					t.Errorf("stderr %q does not contain %q", stderr.String(), w)
				}
			}
		})
	}
}

func TestCLIHelpExitsZero(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := cli([]string{"-h"}, &stdout, &stderr); code != 0 {
		t.Fatalf("exit %d, want 0", code)
	}
	for _, w := range []string{"-base", "origin/main", "-paths", "docs/delivery/review-paths.yaml"} {
		if !strings.Contains(stderr.String(), w) {
			t.Errorf("usage %q does not mention %q", stderr.String(), w)
		}
	}
}

func TestAddedLinesSkipsTheFileHeader(t *testing.T) {
	diff := "diff --git a/x_test.go b/x_test.go\n--- a/x_test.go\n+++ b/x_test.go\n@@ -1,0 +2,2 @@\n-gone\n+\tt.Skip()\n++x\n"
	if got, want := addedLines(diff), []string{"\tt.Skip()", "+x"}; !slices.Equal(got, want) {
		t.Errorf("addedLines = %q, want %q", got, want)
	}
}

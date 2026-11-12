package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunBuildsWithInjectedStampAndNoGit(t *testing.T) {
	root := newFixture(t)
	var stdout, stderr bytes.Buffer
	code := run([]string{"-root", root, "-sha", testSha, "-date", "2026-11-11"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stderr.String())
	}
	data, err := os.ReadFile(filepath.Join(root, "docs", "handbook", "index.html"))
	if err != nil || !strings.Contains(string(data), "main @ abcdef012345 (2026-11-11)") {
		t.Fatalf("index.html: %v\n%s", err, data)
	}
}

func TestRunTakesStampFromEnvironment(t *testing.T) {
	root := newFixture(t)
	t.Setenv("HANDBOOK_SHA", testSha)
	t.Setenv("HANDBOOK_DATE", "2026-03-04")
	// Re-read the env: flag defaults are read inside run.
	var stdout, stderr bytes.Buffer
	if code := run([]string{"-root", root}, &stdout, &stderr); code != 0 {
		t.Fatalf("exit %d: %s", code, stderr.String())
	}
	data, _ := os.ReadFile(filepath.Join(root, "docs", "handbook", "index.html"))
	if !strings.Contains(string(data), "(2026-03-04)") {
		t.Errorf("env date not used")
	}
}

func TestRunReportsBuildFailureWithExit1(t *testing.T) {
	root := newFixture(t)
	page(t, root, "system/a.md", "A", "system", "delivered", "operations: [nope]\n", "x")
	var stdout, stderr bytes.Buffer
	if code := run([]string{"-root", root, "-sha", testSha, "-date", "2026-11-11"}, &stdout, &stderr); code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "docs/handbook/src/system/a.md") || !strings.Contains(stderr.String(), "nope") {
		t.Errorf("failure does not name page and reference: %s", stderr.String())
	}
}

func TestRunUsageErrorExit2(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"-bogus"}, &stdout, &stderr); code != 2 {
		t.Errorf("exit %d, want 2", code)
	}
}

func TestRealHandbookBuilds(t *testing.T) {
	out, err := Build(Config{Root: "../..", Sha: testSha, Date: "2026-11-11"})
	if err != nil {
		t.Fatalf("docs/handbook does not build: %v", err)
	}
	assertOffline(t, "real", out)
	for _, want := range []string{"index.html", "assets/handbook.css", "assets/handbook.js", "assets/search-index.js", "assets/mermaid.min.js"} {
		if len(out[want]) == 0 {
			t.Errorf("real build lacks %s", want)
		}
	}
	for _, s := range sections() {
		if len(out[s+"/index.html"]) == 0 {
			t.Errorf("section %s has no landing page", s)
		}
	}
}

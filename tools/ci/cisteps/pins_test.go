package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const checkoutPin = "actions/checkout@11d5960a326750d5838078e36cf38b85af677262 # v4.4.0"

func TestPinsFixturePasses(t *testing.T) {
	problems, err := unpinned("testdata/ok.yml")
	if err != nil {
		t.Fatal(err)
	}
	if len(problems) > 0 {
		t.Fatalf("%q", problems)
	}
}

// Each way of naming an action that a moved tag, a pushed branch or a
// replaced image could change fails, by file and line (TICK-1194).
func TestUnpinnedFails(t *testing.T) {
	ok, err := os.ReadFile("testdata/ok.yml")
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name, new, want string
	}{
		{"major tag", "actions/checkout@v4", "uses: actions/checkout@v4 is not pinned by full commit SHA"},
		{"exact tag", "actions/checkout@v4.4.0", "uses: actions/checkout@v4.4.0 is not pinned by full commit SHA"},
		{"branch", "actions/checkout@main", "uses: actions/checkout@main is not pinned by full commit SHA"},
		{"short SHA", "actions/checkout@11d5960 # v4.4.0", "uses: actions/checkout@11d5960 is not pinned by full commit SHA"},
		{"upper-case SHA", "actions/checkout@11D5960A326750D5838078E36CF38B85AF677262 # v4.4.0", "is not pinned by full commit SHA"},
		{"SHA without its version", "actions/checkout@11d5960a326750d5838078e36cf38b85af677262", "lacks its version"},
		{"SHA with a free-text comment", "actions/checkout@11d5960a326750d5838078e36cf38b85af677262 # latest", "lacks its version"},
		{"no ref", "actions/checkout", "is not owner/repo@ref"},
		{"docker tag", "docker://alpine:3.20", "is not pinned by digest"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			wf := writeTemp(t, strings.Replace(string(ok), checkoutPin, c.new, 1))
			problems, err := unpinned(wf)
			if err != nil {
				t.Fatal(err)
			}
			if len(problems) != 1 || !strings.Contains(problems[0], c.want) || !strings.Contains(problems[0], wf+":16:") {
				t.Fatalf("want one problem at %s:16 containing %q, got %q", wf, c.want, problems)
			}
		})
	}
}

// A job-level uses: (a reusable workflow) is checked like a step's; a local
// action and a docker image pinned by digest pass.
func TestPinsJobUsesAndExemptions(t *testing.T) {
	wf := writeTemp(t, `on: push
jobs:
  reuse:
    uses: other/repo/.github/workflows/x.yml@main
  local:
    runs-on: ubuntu-latest
    steps:
      - uses: ./.github/actions/setup
      - uses: docker://alpine@sha256:`+strings.Repeat("a", 64)+`
`)
	problems, err := unpinned(wf)
	if err != nil {
		t.Fatal(err)
	}
	if len(problems) != 1 || !strings.Contains(problems[0], "other/repo/.github/workflows/x.yml@main is not pinned") {
		t.Fatalf("got %q", problems)
	}
}

// Every workflow of the repository and every checked-in copy is pinned. The
// glob, not a list, so a new workflow is covered the day it lands.
func TestRepositoryPins(t *testing.T) {
	var paths []string
	for _, g := range []string{".github/workflows/*.yml", ".github/workflows/*.yaml", "tools/ci/cisteps/*.copy.yml"} {
		m, err := filepath.Glob(repo(g))
		if err != nil {
			t.Fatal(err)
		}
		paths = append(paths, m...)
	}
	if len(paths) < 3 {
		t.Fatalf("found only %q", paths)
	}
	for _, p := range paths {
		problems, err := unpinned(p)
		if err != nil {
			t.Fatal(err)
		}
		if len(problems) > 0 {
			t.Errorf("%q", problems)
		}
	}
}

func TestRunPins(t *testing.T) {
	var out, errb bytes.Buffer
	if code := run([]string{"-pins", "testdata/ok.yml"}, &out, &errb); code != 0 {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	if code := run([]string{"-pins"}, &out, &errb); code != 2 {
		t.Fatalf("no files: exit %d", code)
	}
	if code := run([]string{"-pins", "-workflow", "x.yml", "testdata/ok.yml"}, &out, &errb); code != 2 {
		t.Fatalf("-pins with -workflow: exit %d", code)
	}
	bad := writeTemp(t, "jobs:\n  a:\n    steps:\n      - uses: actions/checkout@v4\n")
	errb.Reset()
	if code := run([]string{"-pins", "testdata/ok.yml", bad}, &out, &errb); code != 1 || !strings.Contains(errb.String(), bad+":4:") {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
}

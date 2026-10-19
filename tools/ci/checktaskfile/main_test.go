package main

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func targets() []string { return []string{"build", "test", "lint"} }

func needMake(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("make"); err != nil {
		t.Skip("make is not on PATH")
	}
}

func TestInSyncFixturePasses(t *testing.T) {
	needMake(t)
	problems, err := check(filepath.Join("testdata", "insync"), targets(), makeCommands)
	if err != nil {
		t.Fatal(err)
	}
	if len(problems) > 0 {
		t.Fatalf("want no problems, got %q", problems)
	}
}

// The T05 acceptance fixture: the Taskfile's test command drifts from the
// Makefile's (it lost -race), and the check fails on test alone.
func TestDriftedTestCommandFails(t *testing.T) {
	needMake(t)
	problems, err := check(filepath.Join("testdata", "drift"), targets(), makeCommands)
	if err != nil {
		t.Fatal(err)
	}
	if len(problems) != 1 || !strings.HasPrefix(problems[0], `test: Makefile runs ["go test -race ./..."], Taskfile.yml runs ["go test ./..."]`) {
		t.Fatalf("want one problem for test, got %q", problems)
	}
}

func TestMissingTaskFails(t *testing.T) {
	fake := func(string, string) ([]string, error) { return []string{"true"}, nil }
	problems, err := check(filepath.Join("testdata", "insync"), []string{"generate"}, fake)
	if err != nil {
		t.Fatal(err)
	}
	if len(problems) != 1 || problems[0] != `Taskfile.yml has no task "generate"` {
		t.Fatalf("got %q", problems)
	}
}

// The repository's own Makefile and Taskfile.yml are in step.
func TestRepository(t *testing.T) {
	needMake(t)
	problems, err := check(filepath.Join("..", "..", ".."), targets(), makeCommands)
	if err != nil {
		t.Fatal(err)
	}
	if len(problems) > 0 {
		t.Fatalf("Makefile and Taskfile.yml drifted: %q", problems)
	}
}

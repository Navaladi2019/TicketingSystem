package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func writeTaskfile(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Taskfile.yml"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return dir
}

// fakeMake answers from a table, as `make -n` would.
func fakeMake(cmds map[string][]string) func(dir, target string) ([]string, error) {
	return func(_, target string) ([]string, error) { return cmds[target], nil }
}

func runCLI(args []string, makeCmds func(dir, target string) ([]string, error)) (int, string, string) {
	var stdout, stderr bytes.Buffer
	code := run(args, &stdout, &stderr, makeCmds)
	return code, stdout.String(), stderr.String()
}

// The gate: a drifted Taskfile fails with exit 1 and names the drift.
func TestRunDriftFails(t *testing.T) {
	needMake(t)
	code, stdout, stderr := runCLI([]string{"-dir", filepath.Join("testdata", "drift")}, makeCommands)
	if code != 1 {
		t.Errorf("exit %d, want 1", code)
	}
	if want := `check-taskfile: test: Makefile runs ["go test -race ./..."], Taskfile.yml runs ["go test ./..."]`; !strings.Contains(stderr, want) {
		t.Errorf("stderr %q, want %q", stderr, want)
	}
	if stdout != "" {
		t.Errorf("stdout %q, want nothing on failure", stdout)
	}
}

func TestRunInSyncPasses(t *testing.T) {
	needMake(t)
	code, stdout, stderr := runCLI([]string{"-dir", filepath.Join("testdata", "insync")}, makeCommands)
	if code != 0 || stdout != "check-taskfile: ok: build, test, lint\n" || stderr != "" {
		t.Errorf("exit %d, stdout %q, stderr %q", code, stdout, stderr)
	}
}

// Explicit targets replace the defaults, and a target missing from the
// Taskfile is drift (exit 1).
func TestRunTargets(t *testing.T) {
	dir := filepath.Join("testdata", "insync")
	code, stdout, _ := runCLI([]string{"-dir", dir, "test"}, fakeMake(map[string][]string{"test": {"go test -race ./..."}}))
	if code != 0 || stdout != "check-taskfile: ok: test\n" {
		t.Errorf("test only: exit %d, stdout %q", code, stdout)
	}
	code, _, stderr := runCLI([]string{"-dir", dir, "generate"}, fakeMake(nil))
	if code != 1 || !strings.Contains(stderr, `check-taskfile: Taskfile.yml has no task "generate"`) {
		t.Errorf("missing task: exit %d, stderr %q", code, stderr)
	}
}

// Bad input is exit 2, not drift: no Taskfile, broken YAML, a failing make,
// a bad flag.
func TestRunBadInput(t *testing.T) {
	failingMake := func(string, string) ([]string, error) { return nil, errors.New("make broke") }
	for _, c := range []struct {
		name     string
		args     []string
		makeCmds func(dir, target string) ([]string, error)
		want     string
	}{
		{"no Taskfile", []string{"-dir", t.TempDir()}, fakeMake(nil), "Taskfile.yml"},
		{"bad YAML", []string{"-dir", writeTaskfile(t, "tasks: [unclosed\n")}, fakeMake(nil), "Taskfile.yml: yaml:"},
		{"make fails", []string{"-dir", filepath.Join("testdata", "insync")}, failingMake, "check-taskfile: make broke"},
		{"unknown flag", []string{"-bogus"}, fakeMake(nil), "flag provided but not defined: -bogus"},
	} {
		t.Run(c.name, func(t *testing.T) {
			code, _, stderr := runCLI(c.args, c.makeCmds)
			if code != 2 || !strings.Contains(stderr, c.want) {
				t.Errorf("exit %d, stderr %q; want 2 and %q", code, stderr, c.want)
			}
		})
	}
}

func TestRunHelpExitsZero(t *testing.T) {
	code, _, stderr := runCLI([]string{"-h"}, fakeMake(nil))
	if code != 0 || !strings.Contains(stderr, "-dir") {
		t.Errorf("exit %d, stderr %q", code, stderr)
	}
}

// make itself failing (an unknown target) is an error carrying make's stderr.
func TestMakeCommandsUnknownTarget(t *testing.T) {
	needMake(t)
	_, err := makeCommands(filepath.Join("testdata", "insync"), "no-such-target")
	if err == nil || !strings.Contains(err.Error(), "make -n no-such-target") || !strings.Contains(err.Error(), "no-such-target") {
		t.Fatalf("err = %v, want a make -n no-such-target error", err)
	}
}

// Commands may be scalars (multi-line ones split per line, {{exeExt}}
// dropped, whitespace collapsed) or mappings with cmd; a mapping without cmd
// (a task call) contributes nothing.
func TestTaskCommandsForms(t *testing.T) {
	dir := writeTaskfile(t, `version: "3"
tasks:
  build:
    cmds:
      - go   build -o bin/x{{exeExt}} ./cmd/x
      - cmd: go vet ./...
      - task: other
      - |
        echo one

        echo   two
`)
	got, err := taskCommands(filepath.Join(dir, "Taskfile.yml"))
	if err != nil {
		t.Fatal(err)
	}
	want := map[string][]string{"build": {"go build -o bin/x ./cmd/x", "go vet ./...", "echo one", "echo two"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestTaskCommandsBadCmdMapping(t *testing.T) {
	dir := writeTaskfile(t, "tasks:\n  build:\n    cmds:\n      - cmd: {nested: map}\n")
	if _, err := taskCommands(filepath.Join(dir, "Taskfile.yml")); err == nil {
		t.Fatal("cmd as a mapping: want a decode error")
	}
}

// Package stackcases tests deploy/compose/test/run-cases.sh, the runner that derives the stack-test
// registry from the files in deploy/compose/test/cases (TICK-296). The tests run it on fixture
// directories (CASES_DIR) with cases that only print, so they need no stack.
package stackcases

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const script = "../../../deploy/compose/test/run-cases.sh"

// run executes run-cases.sh with the arguments against dir; it returns stdout+stderr and the exit code.
func run(t *testing.T, dir string, env []string, args ...string) (string, int) {
	t.Helper()
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("no sh on PATH")
	}
	cmd := exec.CommandContext(t.Context(), sh, append([]string{script}, args...)...)
	cmd.Env = append(os.Environ(), append([]string{"CASES_DIR=" + dir}, env...)...)
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	err = cmd.Run()
	code := 0
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		code = ee.ExitCode()
	} else if err != nil {
		t.Fatal(err)
	}
	return out.String(), code
}

// fixture writes one case file per name: header is its "# stack-case:" line ("" for none), body its script.
func fixture(t *testing.T, cases map[string][2]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, c := range cases {
		text := "#!/bin/sh\n"
		if c[0] != "" {
			text += "# stack-case: " + c[0] + "\n"
		}
		text += c[1] + "\n"
		if err := os.WriteFile(filepath.Join(dir, name+".sh"), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func ran(name string) string { return "echo RAN-" + name }

func TestRealCasesHaveValidHeaders(t *testing.T) {
	out, code := run(t, filepath.Join("..", "..", "..", "deploy", "compose", "test", "cases"), nil, "check")
	if code != 0 {
		t.Fatalf("check failed (%d): %s", code, out)
	}
}

// A bad header fails every command loudly, whatever sorts after it (the silent-skip bug).
func TestBadHeaderFailsLoudly(t *testing.T) {
	good := [2]string{"job=stack order=1", ran("a")}
	bad := map[string]string{
		"missing header": "",
		"bad order":      "job=stack order=1O",
		"unknown job":    "job=stakc order=1",
		"no job":         "order=1",
		"bad mode":       "job=stack mode=sequential",
		"unknown field":  "job=stack ordr=2",
	}
	for what, header := range bad {
		t.Run(what, func(t *testing.T) {
			dir := fixture(t, map[string][2]string{"a": good, "b": {header, ran("b")}, "c": good})
			for _, args := range [][]string{{"check"}, {"list", "stack"}, {"run", "stack"}, {"registry"}} {
				out, code := run(t, dir, nil, args...)
				if code == 0 {
					t.Errorf("%v exited 0: %s", args, out)
				}
				if !strings.Contains(out, "b.sh") {
					t.Errorf("%v does not name b.sh: %s", args, out)
				}
				if strings.Contains(out, "RAN-") {
					t.Errorf("%v ran a case despite the bad header: %s", args, out)
				}
			}
		})
	}
}

func TestNoCasesFails(t *testing.T) {
	if out, code := run(t, t.TempDir(), nil, "check"); code == 0 {
		t.Fatalf("an empty cases dir passed: %s", out)
	}
}

func TestJobWithNoCasesFails(t *testing.T) {
	dir := fixture(t, map[string][2]string{"a": {"job=stack order=1", ran("a")}})
	if out, code := run(t, dir, nil, "run", "stack-db"); code == 0 {
		t.Fatalf("a job with no case passed: %s", out)
	}
}

func order(out string, names ...string) bool {
	last := -1
	for _, n := range names {
		i := strings.Index(out, "RAN-"+n)
		if i < 0 || i < last {
			return false
		}
		last = i
	}
	return true
}

func TestSerialRunOrderAndJobs(t *testing.T) {
	dir := fixture(t, map[string][2]string{
		"late":   {"job=stack order=50", ran("late")},
		"first":  {"job=stack,stack-db order=0", ran("first")},
		"dbonly": {"job=stack-db order=5", ran("dbonly")},
		"mid":    {"job=stack order=10 mode=parallel", ran("mid")},
		"hand":   {"job=none order=1", ran("hand")},
	})
	out, code := run(t, dir, nil, "run", "stack")
	if code != 0 || !order(out, "first", "mid", "late") {
		t.Fatalf("exit %d, output: %s", code, out)
	}
	if strings.Contains(out, "RAN-dbonly") || strings.Contains(out, "RAN-hand") {
		t.Fatalf("ran a case of another job: %s", out)
	}
}

// PARALLEL=1: the serial cases before the first parallel one gate the parallel group, which runs
// before the later serial cases.
func TestParallelRunsAfterHealthyBeforeLateSerial(t *testing.T) {
	dir := fixture(t, map[string][2]string{
		"healthy": {"job=stack order=0", ran("healthy")},
		"par1":    {"job=stack order=10 mode=parallel", ran("par1")},
		"par2":    {"job=stack order=10 mode=parallel", ran("par2")},
		"late":    {"job=stack order=50", ran("late")},
	})
	out, code := run(t, dir, []string{"PARALLEL=1"}, "run", "stack")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, out)
	}
	h, p1, p2, l := strings.Index(out, "RAN-healthy"), strings.Index(out, "RAN-par1"), strings.Index(out, "RAN-par2"), strings.Index(out, "RAN-late")
	if h < 0 || h > p1 || h > p2 || l < p1 || l < p2 {
		t.Fatalf("want healthy, then par1 and par2, then late: %s", out)
	}
}

func TestFailingCaseIsNamedAndStopsTheRun(t *testing.T) {
	for _, parallel := range []string{"", "1"} {
		dir := fixture(t, map[string][2]string{
			"healthy": {"job=stack order=0", ran("healthy")},
			"broken":  {"job=stack order=10 mode=parallel", "echo boom; exit 3"},
			"late":    {"job=stack order=50", ran("late")},
		})
		out, code := run(t, dir, []string{"PARALLEL=" + parallel}, "run", "stack")
		if code == 0 {
			t.Fatalf("PARALLEL=%q: a failing case exited 0: %s", parallel, out)
		}
		if !strings.Contains(out, "::error title=stack case failed::broken") || !strings.Contains(out, "FAILED stack cases: broken") {
			t.Errorf("PARALLEL=%q: the failing case is not named: %s", parallel, out)
		}
		if strings.Contains(out, "RAN-late") {
			t.Errorf("PARALLEL=%q: a serial case ran after the failure: %s", parallel, out)
		}
	}
}

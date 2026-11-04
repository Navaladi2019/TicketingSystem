package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// runCLI runs the command on in, writing the report under a temp dir, and
// returns the exit code, stdout, stderr and the report ("" if none).
func runCLI(t *testing.T, in string, args ...string) (int, string, string, string) {
	t.Helper()
	out := filepath.Join(t.TempDir(), "ci6-report.md")
	var stdout, stderr strings.Builder
	code := cli(append([]string{"-out", out}, args...), strings.NewReader(in), &stdout, &stderr)
	b, err := os.ReadFile(out)
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	return code, stdout.String(), stderr.String(), string(b)
}

func TestCLIAllPass(t *testing.T) {
	code, stdout, stderr, md := runCLI(t, stream(t, nil))
	if code != 0 {
		t.Fatalf("exit %d, stderr %q", code, stderr)
	}
	if stderr != "" {
		t.Errorf("stderr %q, want nothing", stderr)
	}
	// The test output streams through, then the report path.
	first := contract()[0].Test
	if !strings.HasPrefix(stdout, "    detail of "+first+"\n") || !strings.Contains(stdout, "ci6report: wrote ") {
		t.Errorf("stdout %q", stdout)
	}
	if !strings.Contains(md, "**All 43 CI-6 and CI-7 cases pass.**") {
		t.Errorf("report:\n%s", md)
	}
}

// The gate: a failed contract test fails the run with exit 1, named on
// stderr and in the report.
func TestCLIFailedCaseExits1(t *testing.T) {
	c := contract()[0]
	code, _, stderr, md := runCLI(t, stream(t, map[string]string{c.Test: "fail"}))
	if code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	if want := "ci6report: " + c.Test + ": FAIL (" + c.Case + ")\n"; stderr != want {
		t.Errorf("stderr %q, want %q", stderr, want)
	}
	if !strings.Contains(md, "**CI-6 and CI-7 FAILED:**") {
		t.Errorf("report:\n%s", md)
	}
}

// The gate: a contract test that did not run fails the run.
func TestCLIMissingCaseExits1(t *testing.T) {
	c := contract()[len(contract())-1]
	code, _, stderr, _ := runCLI(t, stream(t, map[string]string{c.Test: ""}))
	if code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	if want := "ci6report: " + c.Test + ": MISSING (" + c.Case + ")\n"; stderr != want {
		t.Errorf("stderr %q, want %q", stderr, want)
	}
}

// A contract test that started and never finished (a timeout) does not pass.
func TestDidNotFinish(t *testing.T) {
	c := contract()[3]
	in := stream(t, map[string]string{c.Test: ""}) +
		`{"Action":"run","Test":"` + c.Test + `"}` + "\n" +
		`{"Action":"pass","Test":"` + c.Test + `/"}` + "\n"
	out, problems := run(t, in)
	if want := c.Test + ": DID NOT FINISH (" + c.Case + ")"; len(problems) != 1 || problems[0] != want {
		t.Fatalf("problems %q, want %q", problems, want)
	}
	// An empty subtest name is not counted as a passed subtest.
	if !strings.Contains(out, "| `"+c.Test+"` | DID NOT FINISH |  | 0.0s |") {
		t.Errorf("report lacks the unfinished row:\n%s", out)
	}
}

func TestCLIBadFlag(t *testing.T) {
	var stdout, stderr strings.Builder
	if code := cli([]string{"-nope"}, strings.NewReader(""), &stdout, &stderr); code != 2 {
		t.Fatalf("exit %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "flag provided but not defined: -nope") {
		t.Errorf("stderr %q", stderr.String())
	}
}

// An unreadable stream writes no report: exit 2.
func TestCLIReadError(t *testing.T) {
	code, _, stderr, md := runCLI(t, strings.Repeat("x", 1<<24+1))
	if code != 2 {
		t.Fatalf("exit %d, want 2", code)
	}
	if stderr != "ci6report: bufio.Scanner: token too long\n" {
		t.Errorf("stderr %q", stderr)
	}
	if md != "" {
		t.Errorf("a report was written:\n%s", md)
	}
}

func TestCLICannotCreateReport(t *testing.T) {
	out := filepath.Join(t.TempDir(), "missing", "ci6-report.md")
	var stdout, stderr strings.Builder
	if code := cli([]string{"-out", out}, strings.NewReader(stream(t, nil)), &stdout, &stderr); code != 2 {
		t.Fatalf("exit %d, want 2", code)
	}
	if !strings.HasPrefix(stderr.String(), "ci6report: open "+out) {
		t.Errorf("stderr %q", stderr.String())
	}
	if strings.Contains(stdout.String(), "wrote") {
		t.Errorf("stdout %q claims a report", stdout.String())
	}
}

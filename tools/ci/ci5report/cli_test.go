package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/identity/contracttest"
)

// runCLI runs the command on in, writing the report under a temp dir, and
// returns the exit code, stdout, stderr and the report ("" if none).
func runCLI(t *testing.T, in string, args ...string) (int, string, string, string) {
	t.Helper()
	out := filepath.Join(t.TempDir(), "ci5-report.md")
	var stdout, stderr strings.Builder
	code := cli(append([]string{"-out", out}, args...), strings.NewReader(in), &stdout, &stderr)
	b, err := os.ReadFile(out)
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	return code, stdout.String(), stderr.String(), string(b)
}

func TestCLIAllPass(t *testing.T) {
	in := `{"Action":"output","Test":"TestCI5","Output":"=== RUN   TestCI5\n"}` + "\n" + stream(t, "")
	code, stdout, stderr, md := runCLI(t, in)
	if code != 0 {
		t.Fatalf("exit %d, stderr %q", code, stderr)
	}
	if stderr != "" {
		t.Errorf("stderr %q, want nothing", stderr)
	}
	// The test output streams through, then the report path.
	if !strings.HasPrefix(stdout, "=== RUN   TestCI5\n") || !strings.Contains(stdout, "ci5report: wrote ") {
		t.Errorf("stdout %q", stdout)
	}
	if !strings.Contains(md, "**All 14 CI-5 cases pass.**") {
		t.Errorf("report:\n%s", md)
	}
}

// The gate: a catalogue case with no subtest fails the run with exit 1 and
// names the case on stderr and in the report.
func TestCLIMissingCaseExits1(t *testing.T) {
	code, _, stderr, md := runCLI(t, stream(t, contracttest.CI5OrgPickerTamper))
	if code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	if !strings.Contains(stderr, "ci5report: CI-5.06: MISSING (") {
		t.Errorf("stderr %q", stderr)
	}
	if !strings.Contains(md, "**CI-5 FAILED:**") || !strings.Contains(md, "- CI-5.06: MISSING") {
		t.Errorf("report:\n%s", md)
	}
}

// A failed case also fails the run, with the failed top-level test's output
// in the report.
func TestCLIFailedCaseExits1(t *testing.T) {
	in := stream(t, contracttest.CI5DuplicateBinding,
		event{Action: "output", Test: "TestCI5/Scenarios/[CI-5.12]_dup", Output: "    dup bound twice\n"},
		event{Action: "fail", Test: "TestCI5/Scenarios/[CI-5.12]_dup"}) +
		// After stream's own final pass of TestCI5, so the parent ends failed.
		`{"Action":"output","Test":"TestCI5","Output":"--- FAIL: TestCI5\n"}` + "\n" +
		`{"Action":"fail","Test":"TestCI5"}` + "\n"
	code, _, stderr, md := runCLI(t, in)
	if code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	if !strings.Contains(stderr, "ci5report: TestCI5/Scenarios/[CI-5.12]_dup: FAIL (CI-5.12)\n") ||
		!strings.Contains(stderr, "ci5report: TestCI5: FAIL\n") {
		t.Errorf("stderr %q", stderr)
	}
	if !strings.Contains(md, "<details><summary>TestCI5 output</summary>\n\n```\n--- FAIL: TestCI5\n```") {
		t.Errorf("report lacks the failed test's output:\n%s", md)
	}
	// A subtest's output is not dumped on its own.
	if strings.Contains(md, "<details><summary>TestCI5/") {
		t.Errorf("report dumps a subtest:\n%s", md)
	}
}

// A case subtest or another test that started and never finished (a
// timeout) does not pass.
func TestDidNotFinish(t *testing.T) {
	in := stream(t, contracttest.CI5DuplicateBinding,
		event{Action: "run", Test: "TestCI5/Scenarios/[CI-5.12]_dup"},
		event{Action: "run", Test: "TestCI5/Hung"})
	problems, md := run(t, in)
	want := []string{"TestCI5/Scenarios/[CI-5.12]_dup: DID NOT FINISH (CI-5.12)", "TestCI5/Hung: DID NOT FINISH"}
	if strings.Join(problems, "|") != strings.Join(want, "|") {
		t.Errorf("problems %q, want %q", problems, want)
	}
	for _, row := range []string{"| `TestCI5/Scenarios/[CI-5.12]_dup` | DID NOT FINISH |", "| `TestCI5/Hung` | DID NOT FINISH |"} {
		if !strings.Contains(md, row) {
			t.Errorf("report lacks %q:\n%s", row, md)
		}
	}
}

// A package-level fail (no test) is a package failure.
func TestPackageLevelFail(t *testing.T) {
	problems, _ := run(t, stream(t, "")+`{"Action":"fail"}`+"\n")
	if len(problems) != 1 || !strings.Contains(problems[0], "package failed") {
		t.Errorf("problems %v", problems)
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
	if stderr != "ci5report: bufio.Scanner: token too long\n" {
		t.Errorf("stderr %q", stderr)
	}
	if md != "" {
		t.Errorf("a report was written:\n%s", md)
	}
}

func TestCLICannotCreateReport(t *testing.T) {
	out := filepath.Join(t.TempDir(), "missing", "ci5-report.md")
	var stdout, stderr strings.Builder
	if code := cli([]string{"-out", out}, strings.NewReader(stream(t, "")), &stdout, &stderr); code != 2 {
		t.Fatalf("exit %d, want 2", code)
	}
	if !strings.HasPrefix(stderr.String(), "ci5report: open "+out) {
		t.Errorf("stderr %q", stderr.String())
	}
	if strings.Contains(stdout.String(), "wrote") {
		t.Errorf("stdout %q claims a report", stdout.String())
	}
}

//go:build integration

package isolation

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// reportCrashChildEnv makes TestMain skip the cluster and run only
// TestReportCrashChild, which the crash test starts as a child process.
const reportCrashChildEnv = "NAVA_ISOLATION_REPORT_CRASH_CHILD"

// TestReportSurvivesAKilledBinary is TICK-1180: in TICK-753 V9 the generic
// probe went red, a later path hung, and the go test -timeout panic killed
// the binary before TestMain wrote the report, so no row of that run was
// left. The child records a red CI-2 line and one path, starts a second path
// and hangs past its -test.timeout; the report on disk must still hold all of
// that.
func TestReportSurvivesAKilledBinary(t *testing.T) {
	if os.Getenv(reportCrashChildEnv) != "" {
		t.Skip("the child runs TestReportCrashChild only")
	}
	ps := Paths()
	if len(ps) < 3 {
		t.Fatalf("%d registered paths, want at least 3", len(ps))
	}
	path := filepath.Join(t.TempDir(), "isolation-report.md")
	cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestReportCrashChild$", "-test.timeout=3s")
	cmd.Env = append(os.Environ(), reportCrashChildEnv+"=1", "NAVA_ISOLATION_REPORT="+path)
	out, err := cmd.CombinedOutput()
	var exit *exec.ExitError
	if !errors.As(err, &exit) {
		t.Fatalf("the child should die of its -test.timeout, got err %v:\n%s", err, out)
	}
	if !strings.Contains(string(out), "test timed out") {
		t.Fatalf("the child did not stop on its -test.timeout:\n%s", out)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("no report after the binary was killed: %v", err)
	}
	md := string(b)
	for _, want := range []string{
		"Result: **" + resultIncomplete + "**",
		"**Rows had already failed.**",
		"| nava_app | public.crash_probe | FAIL | reached tenant B |",
		reportRow(ps[0], statusPass) + "finished before the hang |",
		reportRow(ps[1], statusRunning),
		reportRow(ps[2], statusNotRun),
	} {
		if !strings.Contains(md, want) {
			t.Errorf("the report lacks %q:\n%s", want, md)
		}
	}
	if m, _ := filepath.Glob(path + ".*.tmp"); len(m) > 0 {
		t.Errorf("temporary report files left: %v", m)
	}
}

// reportRow is the start of p's line in the report's path table.
func reportRow(p Path, status string) string {
	role := p.Role
	if role == "" {
		role = "-"
	}
	return "| " + cellText(p.Name) + " | " + role + " | " + status + " | "
}

// TestReportCrashChild is the child of TestReportSurvivesAKilledBinary; on
// its own it skips.
func TestReportCrashChild(t *testing.T) {
	if os.Getenv(reportCrashChildEnv) == "" {
		t.Skip("run by TestReportSurvivesAKilledBinary")
	}
	ps := Paths()
	suite.setCI2([]ci2Line{{Role: "nava_app", Table: "public.crash_probe", Status: statusFail, Detail: "reached tenant B"}})
	suite.startPath(ps[0].Name)
	suite.addPath(pathResult{Path: ps[0], Status: statusPass, Detail: "finished before the hang"})
	suite.startPath(ps[1].Name)
	time.Sleep(time.Hour) // the -test.timeout panic ends the binary here
}

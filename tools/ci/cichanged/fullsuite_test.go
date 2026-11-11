package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TICK-638: scripts/ci/full-suite.sh (make check-full-suite, the ci
// workflow's full-suite job on a pull request into a story branch) fails
// unless every suite job succeeded: a skipped job is not green. TICK-695: the
// one exception is temporal-server, skipped because the changes job found the
// module unchanged (TEMPORAL_SERVER=false).
func TestFullSuite(t *testing.T) {
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("no sh on PATH")
	}
	root, err := git("rev-parse", "--show-toplevel")
	if err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(strings.TrimSpace(root), "scripts", "ci", "full-suite.sh")
	green := "changes=success lint=success unit=success temporal-server=success integration=success commit-check-budget=success stack=success coverage=success"
	tsrv := func(result string) string {
		return strings.Replace(green, "temporal-server=success", "temporal-server="+result, 1)
	}
	for _, tc := range []struct {
		name    string
		results string
		tsrv    string // needs.changes.outputs.temporal_server
		ok      bool
	}{
		{"every job succeeded", green, "true", true},
		{"a skipped job", strings.Replace(green, "stack=success", "stack=skipped", 1), "", false},
		{"a failed job", strings.Replace(green, "unit=success", "unit=failure", 1), "", false},
		{"a cancelled job", strings.Replace(green, "coverage=success", "coverage=cancelled", 1), "", false},
		{"a missing job", strings.Replace(green, " integration=success", "", 1), "", false},
		{"no results", "", "", false},
		{"temporal-server skipped, module unchanged", tsrv("skipped"), "false", true},
		{"temporal-server skipped, module changed", tsrv("skipped"), "true", false},
		{"temporal-server skipped, no decision", tsrv("skipped"), "", false},
		{"temporal-server failed, module unchanged", tsrv("failure"), "false", false},
		{"temporal-server cancelled, module unchanged", tsrv("cancelled"), "false", false},
		{"another job skipped, module unchanged", strings.Replace(green, "stack=success", "stack=skipped", 1), "false", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.CommandContext(context.Background(), sh, script)
			cmd.Env = append(os.Environ(), "RESULTS="+tc.results, "TEMPORAL_SERVER="+tc.tsrv)
			out, err := cmd.CombinedOutput()
			if tc.ok && err != nil {
				t.Fatalf("want success, got %v\n%s", err, out)
			}
			if !tc.ok && err == nil {
				t.Fatalf("want failure, got success\n%s", out)
			}
		})
	}
}

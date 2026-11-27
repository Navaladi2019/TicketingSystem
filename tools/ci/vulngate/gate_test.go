// Package vulngate tests the govulncheck workflow's detect and gate scripts,
// scripts/ci/vulncheck-detect.sh and scripts/ci/vulncheck-gate.sh (ADR-001-E4
// contract rules 1, 2 and 4, tests V-1 and V-2, TICK-1148). It has no
// non-test code: a change here counts as a change to the gate (D-E4-3).
package vulngate

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

const repoRoot = "../../.."

// script returns the absolute path of scripts/ci/<name> and the sh to run it
// with, skipping the test when there is no sh.
func script(t *testing.T, name string) (sh, path string) {
	t.Helper()
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("no sh on PATH: the vulncheck scripts are POSIX sh")
	}
	path, err = filepath.Abs(filepath.Join(repoRoot, "scripts", "ci", name))
	if err != nil {
		t.Fatal(err)
	}
	// Read it here, not only from sh: go test caches a result keyed on the
	// files the test process opened, so without this an edit to the script
	// would replay a stale pass.
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(b), "#!/bin/sh\n") {
		t.Fatalf("%s: does not start with #!/bin/sh and an LF line end", name)
	}
	return sh, path
}

// clean drops every input either script reads from the inherited environment,
// so a CI runner's own GITHUB_OUTPUT or EVENT cannot leak into a case.
func clean(env []string) []string {
	drop := map[string]bool{
		"GITHUB_OUTPUT": true, "EVENT": true, "BASE_SHA": true, "HEAD_SHA": true,
		"DETECT_RESULT": true, "ROOT_NEEDED": true, "TS_NEEDED": true, "ROOT_RESULT": true, "TS_RESULT": true,
	}
	out := make([]string, 0, len(env))
	for _, kv := range env {
		k, _, _ := strings.Cut(kv, "=")
		if !drop[strings.ToUpper(k)] {
			out = append(out, kv)
		}
	}
	return out
}

// exitCode runs cmd and returns its exit code; a failure to start is fatal.
func exitCode(t *testing.T, cmd *exec.Cmd) int {
	t.Helper()
	err := cmd.Run()
	var ee *exec.ExitError
	switch {
	case err == nil:
		return 0
	case errors.As(err, &ee):
		return ee.ExitCode()
	default:
		t.Fatalf("%s: %v", cmd.Path, err)
		return -1
	}
}

// gateCase is one input set of the gate.
type gateCase struct {
	detect, rootNeeded, rootResult, tsNeeded, tsResult string
}

func (c gateCase) String() string {
	return fmt.Sprintf("DETECT_RESULT=%q ROOT_NEEDED=%q ROOT_RESULT=%q TS_NEEDED=%q TS_RESULT=%q",
		c.detect, c.rootNeeded, c.rootResult, c.tsNeeded, c.tsResult)
}

// want is contract rule 4, written independently of the script: green only
// when detect succeeded and each module ran because it was needed or was
// skipped because it was not.
func (c gateCase) want() int {
	module := func(needed, result string) bool {
		return (needed == "true" && result == "success") || (needed == "false" && result == "skipped")
	}
	if c.detect == "success" && module(c.rootNeeded, c.rootResult) && module(c.tsNeeded, c.tsResult) {
		return 0
	}
	return 1
}

// V-1: every detect result x each module's detect output x each module's job
// result. A skip, a cancel, an empty or unknown value, or a scan that ran
// although detect said it was not needed is never read as a pass.
func TestGate(t *testing.T) {
	t.Parallel()
	sh, gate := script(t, "vulncheck-gate.sh")
	results := []string{"success", "failure", "cancelled", "skipped", ""}
	needed := []string{"true", "false", "", "maybe"}

	var cases []gateCase
	for _, d := range results {
		for _, rn := range needed {
			for _, rr := range results {
				for _, tn := range needed {
					for _, tr := range results {
						cases = append(cases, gateCase{d, rn, rr, tn, tr})
					}
				}
			}
		}
	}
	base := clean(os.Environ())
	got := make([]int, len(cases))
	outs := make([]string, len(cases))
	// 2000 sh runs: bounded parallel workers keep it to seconds on Linux.
	work := make(chan int)
	var wg sync.WaitGroup
	for range runtime.GOMAXPROCS(0) * 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range work {
				c := cases[i]
				cmd := exec.CommandContext(context.Background(), sh, gate)
				cmd.Env = append(append([]string{}, base...),
					"DETECT_RESULT="+c.detect, "ROOT_NEEDED="+c.rootNeeded, "ROOT_RESULT="+c.rootResult,
					"TS_NEEDED="+c.tsNeeded, "TS_RESULT="+c.tsResult)
				var out strings.Builder
				cmd.Stdout, cmd.Stderr = &out, &out
				got[i] = exitCode(t, cmd)
				outs[i] = out.String()
			}
		}()
	}
	for i := range cases {
		work <- i
	}
	close(work)
	wg.Wait()

	green, wrong := 0, 0
	for i, c := range cases {
		if got[i] == 0 {
			green++
		}
		if got[i] != c.want() {
			wrong++
			if wrong <= 20 {
				t.Errorf("%v: exit %d, want %d\n%s", c, got[i], c.want(), outs[i])
			}
		}
		// A failing gate says which input failed.
		if got[i] != 0 && !strings.Contains(outs[i], "vulncheck-gate:") {
			t.Errorf("%v: exit %d with no reason printed", c, got[i])
		}
	}
	if wrong > 20 {
		t.Errorf("... and %d more wrong cases (%d of %d)", wrong-20, wrong, len(cases))
	}
	// Exactly the four needed x {true,false} combinations are green.
	if green != 4 && wrong == 0 {
		t.Errorf("%d green cases, want 4", green)
	}
}

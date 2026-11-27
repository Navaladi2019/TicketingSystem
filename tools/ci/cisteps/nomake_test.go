package main

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// ADR-001-A7 rule 9 (A7.3), contract test V-3 as amended: no step of
// govulncheck.yml runs make. Each edit is made to the workflow and its copy
// together, so the copy comparison passes and only the make rule can fail.

// writeBoth writes body as govulncheck.yml and as its copy, in a new
// directory, and returns both paths.
func writeBoth(t *testing.T, body string) (wf, cp string) {
	t.Helper()
	dir := t.TempDir()
	wf, cp = filepath.Join(dir, "govulncheck.yml"), filepath.Join(dir, "govulncheck.copy.yml")
	for _, p := range []string{wf, cp} {
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return wf, cp
}

// The issue's cases (TICK-1187 Tests 1-3): make in a scan step, as a word of
// any line of the run: string, fails and names the job and the step.
func TestCopyRefusesMakeInTheWorkflow(t *testing.T) {
	wf := readRepo(t, repoWorkflow)
	cases := []struct {
		name string
		edit edit
		want string
	}{
		{"run: make vulncheck", atScanLine("vulncheck", func(ind, _ string) []string {
			return []string{ind + "run: make vulncheck"}
		}), "govulncheck.yml job vulncheck-root step 3: runs make (ADR-001-A7 rule 9)"},
		{"make on line 2 of run: |", atScanLine("vulncheck-temporal-server", func(ind, _ string) []string {
			return []string{ind + "run: |", ind + "  echo scanning", ind + "  cd temporal-server && make vulncheck-temporal-server"}
		}), "govulncheck.yml job vulncheck-temporal-server step 3: runs make (ADR-001-A7 rule 9)"},
		{"echo ok; make vulncheck", atScanLine("vulncheck", func(ind, _ string) []string {
			return []string{ind + "run: echo ok; make vulncheck"}
		}), "govulncheck.yml job vulncheck-root step 3: runs make (ADR-001-A7 rule 9)"},
		{"make in a non-scan job", replaceLine(`^\s+sh "\$RUNNER_TEMP/vulncheck-detect\.sh"`, `          make vulncheck-detect`),
			"govulncheck.yml job vulncheck-detect step 2: runs make (ADR-001-A7 rule 9)"},
		{"make in a new step", atScanLine("vulncheck", func(ind, run string) []string {
			return []string{ind + run, "      - run: make tools-govulncheck"}
		}), "govulncheck.yml job vulncheck-root step 4: runs make (ADR-001-A7 rule 9)"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p, cp := writeBoth(t, c.edit(t, wf))
			var stdout, stderr bytes.Buffer
			if code := run([]string{"-workflow", p, "-copy", cp}, &stdout, &stderr); code != 1 {
				t.Fatalf("exit %d, want 1; stderr %q", code, stderr.String())
			}
			if want := "check-ci-steps: " + c.want + "\n"; !strings.HasPrefix(stderr.String(), want) {
				t.Errorf("stderr %q, want it to start %q", stderr.String(), want)
			}
			problems, err := checkCopy(p, cp)
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(problems, []string{c.want}) {
				t.Errorf("problems %q, want exactly %q", problems, c.want)
			}
		})
	}
}

// TICK-1187 Test 3: make only as part of another word is not make.
func TestCopyAllowsMakeInsideAWord(t *testing.T) {
	wf := readRepo(t, repoWorkflow)
	for _, line := range []string{"go run ./cmake-tool", "echo automake", "echo make-x", "echo remake.sh", "echo makefile"} {
		t.Run(line, func(t *testing.T) {
			p, cp := writeBoth(t, atScanLine("vulncheck", func(ind, run string) []string {
				return []string{ind + "run: |", ind + "  " + line, ind + "  " + strings.TrimPrefix(run, "run: ")}
			})(t, wf))
			problems, err := checkCopy(p, cp)
			if err != nil {
				t.Fatal(err)
			}
			if len(problems) > 0 {
				t.Fatalf("%q", problems)
			}
		})
	}
}

// The word rule of A7.3, `(^|[\s;&|(])make(\s|$)`, line by line.
func TestMakeWord(t *testing.T) {
	for line, want := range map[string]bool{
		"make":                   true,
		"make vulncheck":         true,
		"  make vulncheck":       true,
		"\tmake vulncheck":       true,
		"echo ok; make x":        true,
		"echo ok;make x":         true,
		"true && make x":         true,
		"true&&make x":           true,
		"false || make x":        true,
		"echo | make -f -":       true,
		"(make x)":               true,
		"$(make x)":              true,
		"x=1 make":               true,
		"go run ./cmake-tool":    false,
		"echo automake":          false,
		"echo make-x":            false,
		"echo makefile":          false,
		"make;":                  false,
		"go run x@v1.9.0 ./...":  false,
		"echo 'make x'":          false,
		"echo \"make x\"":        false,
		"/usr/bin/make x":        false,
		"sh scripts/ci/x.sh":     false,
		"git show origin/main:x": false,
	} {
		if got := makeWord.MatchString(line); got != want {
			t.Errorf("makeWord(%q) = %v, want %v", line, got, want)
		}
	}
}

// A workflow that is not YAML, or whose run: is not a string, is an error
// (exit 2), not a pass.
func TestCopyWorkflowNotYAML(t *testing.T) {
	for name, body := range map[string]string{
		"not yaml":          "jobs: [\n",
		"run is a sequence": "jobs:\n  a:\n    steps:\n      - run: [make, x]\n",
	} {
		t.Run(name, func(t *testing.T) {
			p, cp := writeBoth(t, body)
			if _, err := checkCopy(p, cp); err == nil {
				t.Fatal("want an error")
			}
			var stdout, stderr bytes.Buffer
			if code := run([]string{"-workflow", p, "-copy", cp}, &stdout, &stderr); code != 2 {
				t.Fatalf("exit %d, want 2; stderr %q", code, stderr.String())
			}
		})
	}
}

// Problems come in job id order, then step order, one per step even when
// several of its lines run make.
func TestMakeStepsOrder(t *testing.T) {
	p, _ := writeBoth(t, "jobs:\n  b:\n    steps:\n      - run: 'true'\n      - run: |\n          make x\n          make y\n  a:\n    steps:\n      - run: make\n")
	got, err := makeSteps(p)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"govulncheck.yml job a step 1: runs make (ADR-001-A7 rule 9)",
		"govulncheck.yml job b step 2: runs make (ADR-001-A7 rule 9)",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

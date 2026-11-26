package main

import (
	"os"
	"strings"
	"testing"
)

// TICK-1146: GitHub names a skipped job's check run after its unevaluated
// name: expression when the expression reads a context, and a run's job
// after the evaluated one. A job named
// ${{ github.event_name && 'isolation' || 'isolation' }} therefore reports
// the required check "isolation" only when it ran; a label-only run, a fast
// run or an unlabelled run that skips it can no longer stand in for a red
// result on the same commit (spike PR 898: a literal name went BLOCKED ->
// CLEAN on adding a label). A constant ${{ 'isolation' }} does not work:
// GitHub folds it to the literal name on a skipped run too (PR 904).

const checksRequired = `steps:
  - job: isolation
    run: make isolation
    why: isolation suite
checks:
  - job: isolation
    check: isolation
    why: a skipped run hides a red isolation
`

const checksWorkflow = `jobs:
  isolation:
    name: ${{ github.event_name && 'isolation' || 'isolation' }}
    runs-on: ubuntu-latest
    steps:
      - run: make isolation
`

func TestRequiredCheckFromAJobThatRan(t *testing.T) {
	req := writeTemp(t, checksRequired)
	cases := []struct {
		name, old, new, want string
	}{
		{"expression name", "", "", ""},
		{"spaces inside the expression", "${{ github.event_name && 'isolation' || 'isolation' }}", "${{   github.event_name  &&  'isolation' || 'isolation'   }}", ""},
		{"constant expression", "${{ github.event_name && 'isolation' || 'isolation' }}", "${{ 'isolation' }}", `job isolation: check "isolation" is reported by a skipped run: name is "${{ 'isolation' }}", want "${{ github.event_name && 'isolation' || 'isolation' }}" [a skipped run hides a red isolation]`},
		{"literal name", "${{ github.event_name && 'isolation' || 'isolation' }}", "isolation", `job isolation: check "isolation" is reported by a skipped run: name is "isolation", want "${{ github.event_name && 'isolation' || 'isolation' }}" [a skipped run hides a red isolation]`},
		{"no name", "    name: ${{ github.event_name && 'isolation' || 'isolation' }}\n", "", `job isolation: check "isolation" is reported by a skipped run: name is "", want "${{ github.event_name && 'isolation' || 'isolation' }}" [a skipped run hides a red isolation]`},
		{"other check name", "${{ github.event_name && 'isolation' || 'isolation' }}", "${{ github.event_name && 'isolation-suite' || 'isolation-suite' }}", `job isolation: check "isolation" is never reported: name is "${{ github.event_name && 'isolation-suite' || 'isolation-suite' }}", want "${{ github.event_name && 'isolation' || 'isolation' }}" [a skipped run hides a red isolation]`},
		{"job removed", "  isolation:\n", "  tenancy:\n", `job isolation: check "isolation" is missing: no such job [a skipped run hides a red isolation]`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			wf := strings.Replace(checksWorkflow, c.old, c.new, 1)
			if c.old != "" && wf == checksWorkflow {
				t.Fatalf("fixture has no %q", c.old)
			}
			problems, n, err := check(writeTemp(t, wf), req)
			if err != nil {
				t.Fatal(err)
			}
			if n != 2 {
				t.Errorf("counted %d entries, want 2", n)
			}
			var steps []string
			for _, p := range problems {
				if !strings.Contains(p, `step "make isolation"`) {
					steps = append(steps, p)
				}
			}
			switch {
			case c.want == "" && len(steps) != 0:
				t.Fatalf("want no problem, got %q", steps)
			case c.want != "" && (len(steps) != 1 || steps[0] != c.want):
				t.Fatalf("want [%q], got %q", c.want, steps)
			}
		})
	}
}

func TestBadChecksEntryIsAnError(t *testing.T) {
	wf := writeTemp(t, checksWorkflow)
	for name, body := range map[string]string{
		"no job":   "steps:\n  - job: isolation\n    run: make isolation\nchecks:\n  - check: isolation\n",
		"no check": "steps:\n  - job: isolation\n    run: make isolation\nchecks:\n  - job: isolation\n",
	} {
		t.Run(name, func(t *testing.T) {
			if _, _, err := check(wf, writeTemp(t, body)); err == nil {
				t.Fatal("want an error")
			}
		})
	}
}

// The repository's list pins the isolation check (TICK-1146).
func TestRepositoryPinsIsolationCheck(t *testing.T) {
	data, err := os.ReadFile(repo("tools/ci/cisteps/required.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "\n  - job: isolation\n    check: isolation\n") {
		t.Fatal("required.yaml has no checks entry for job isolation, check isolation")
	}
}

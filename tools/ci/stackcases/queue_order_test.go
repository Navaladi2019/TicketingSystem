package stackcases

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// The nava-tq sharing rule (TICK-374, Technical Lead 2026-11-01): the
// internal/temporalstack suite registers its own test bodies as
// nava.interpreter on nava-tq, and the internal/workerstack interpreter cases
// start nava-worker-interptest, which polls the same type on the same queue.
// The two must never poll nava-tq at the same time. These tests assert the
// stack-case registry that keeps them apart, so a header edit that breaks it
// fails `task test` instead of a stack run.

const (
	repoRoot = "../../.."
	casesDir = repoRoot + "/deploy/compose/test/cases"
)

var (
	makeExec  = regexp.MustCompile(`(?m)^\s*exec make\b.*\s(stack-test-[A-Za-z0-9_-]+)\s*$`)
	targetDef = regexp.MustCompile(`^(stack-test-[A-Za-z0-9_-]+):`)
)

// stackCase is one row of `run-cases.sh registry`.
type stackCase struct {
	name, mode string
	order      int
	jobs       []string
}

// registry parses `run-cases.sh registry` for the real cases.
func registry(t *testing.T) map[string]stackCase {
	t.Helper()
	out, code := run(t, casesDir, nil, "registry")
	if code != 0 {
		t.Fatalf("registry failed (%d): %s", code, out)
	}
	cases := map[string]stackCase{}
	var c *stackCase
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		k, v, ok := strings.Cut(strings.TrimPrefix(line, "- "), ": ")
		if !ok {
			continue
		}
		switch k {
		case "name":
			c = &stackCase{name: v}
			cases[v] = *c
		case "serial":
			c.mode = map[string]string{"true": "serial", "false": "parallel"}[v]
		case "order":
			n, err := strconv.Atoi(v)
			if err != nil {
				t.Fatalf("case %s: order %q", c.name, v)
			}
			c.order = n
		case "ci_jobs":
			c.jobs = strings.Split(strings.Trim(v, "[]"), ",")
		}
		if c != nil {
			cases[c.name] = *c
		}
	}
	return cases
}

// recipes maps each stack-test-* make target in make/*.mk to its recipe text.
func recipes(t *testing.T) map[string]string {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(repoRoot, "make", "*.mk"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no make/*.mk files: %v", err)
	}
	out := map[string]string{}
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		target := ""
		for _, line := range strings.Split(string(b), "\n") {
			if m := targetDef.FindStringSubmatch(line); m != nil {
				target = m[1]
				continue
			}
			if target != "" && strings.HasPrefix(line, "\t") {
				out[target] += line + "\n"
				continue
			}
			target = ""
		}
	}
	return out
}

// queueUsers sorts the real cases into the two nava-tq users: bodies (the
// case runs internal/temporalstack: the sdk suite, and demo_outbox, whose
// test starts a relayProbe worker) and interp (the case runs
// internal/workerstack, whose interpreter cases start nava-worker-interptest;
// its closeRef body runs in the same serial go test process).
func queueUsers(t *testing.T, cases map[string]stackCase) (bodies, interp []stackCase) {
	t.Helper()
	recs := recipes(t)
	for name, c := range cases {
		b, err := os.ReadFile(filepath.Join(casesDir, name+".sh"))
		if err != nil {
			t.Fatal(err)
		}
		m := makeExec.FindStringSubmatch(string(b))
		if m == nil {
			continue
		}
		rec, ok := recs[m[1]]
		if !ok {
			t.Fatalf("case %s runs make target %s, which no make/*.mk defines", name, m[1])
		}
		switch {
		case strings.Contains(rec, "./internal/workerstack"):
			interp = append(interp, c)
		case strings.Contains(rec, "./internal/temporalstack"):
			bodies = append(bodies, c)
		}
	}
	return bodies, interp
}

func TestInterpreterQueueOrdering(t *testing.T) {
	cases := registry(t)
	bodies, interp := queueUsers(t, cases)
	// Not vacuous: the cases this rule is about are found.
	has := func(cs []stackCase, name string) bool {
		for _, c := range cs {
			if c.name == name {
				return true
			}
		}
		return false
	}
	if !has(bodies, "sdk") || !has(bodies, "demo_outbox") {
		t.Fatalf("the sdk and demo_outbox cases (internal/temporalstack) were not classified as test-body cases: %v", bodies)
	}
	if !has(interp, "worker") || !has(interp, "interpreter") {
		t.Fatalf("the worker and interpreter cases were not classified as nava-worker-interptest cases: %v", interp)
	}
	for _, v := range queueRuleViolations(bodies, interp) {
		t.Error(v)
	}
}

// queueRuleViolations returns one message per broken rule:
//   - a case that starts nava-worker-interptest is serial, so with or without
//     PARALLEL=1 it runs alone and never shares nava-tq with another case;
//   - a parallel test-body case (sdk) sorts before it, so the parallel group
//     has finished and stopped its workers before nava-worker-interptest
//     starts, in both run modes (the parallel group runs before every serial
//     case of a later order).
//
// A serial test-body case (demo_outbox) may sort on either side: serial cases
// never overlap, the interpreter case removes its container before it ends,
// and the body's worker refuses to start while one runs (requireQueueFree).
func queueRuleViolations(bodies, interp []stackCase) []string {
	var out []string
	for _, i := range interp {
		if i.mode != "serial" {
			out = append(out, fmt.Sprintf("case %s starts nava-worker-interptest but is mode=%s; it must be serial", i.name, i.mode))
		}
		for _, b := range bodies {
			if b.mode == "parallel" && b.order >= i.order {
				out = append(out, fmt.Sprintf("case %s (order %d, temporalstack test bodies on nava-tq) must sort before %s (order %d, nava-worker-interptest)",
					b.name, b.order, i.name, i.order))
			}
		}
	}
	return out
}

// Mutation check: each way of breaking the rule is reported.
func TestQueueRuleViolationsCatchesBrokenOrders(t *testing.T) {
	sdk := stackCase{name: "sdk", mode: "parallel", order: 12}
	for what, tc := range map[string]struct {
		bodies, interp []stackCase
		want           int
	}{
		"today's order":             {[]stackCase{sdk}, []stackCase{{name: "worker", mode: "serial", order: 20}}, 0},
		"interpreter case parallel": {[]stackCase{sdk}, []stackCase{{name: "worker", mode: "parallel", order: 20}}, 1},
		"interpreter case first":    {[]stackCase{sdk}, []stackCase{{name: "worker", mode: "serial", order: 5}}, 1},
		"same order":                {[]stackCase{sdk}, []stackCase{{name: "worker", mode: "serial", order: 12}}, 1},
		"serial body after":         {[]stackCase{sdk, {name: "demo_outbox", mode: "serial", order: 22}}, []stackCase{{name: "worker", mode: "serial", order: 20}}, 0},
	} {
		if got := queueRuleViolations(tc.bodies, tc.interp); len(got) != tc.want {
			t.Errorf("%s: got %d violations %v, want %d", what, len(got), got, tc.want)
		}
	}
}

package isolation

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"testing"
)

// The CI-10 cycle and depth-audit cases run as TestCI10_* integration tests
// in internal/records/pipeline and internal/outbox, against real PostgreSQL
// in make test-integration (TICK-483). That script keeps their `go test
// -json` stream in NAVA_IT_EVENTS, and this suite turns it into one named
// PASS/FAIL row per case in the report, so the rows come from the same tests.

const modulePath = "github.com/Navaladi2019/TicketingSystem/"

// ci10Case is one named CI-10 row and the test that proves it.
type ci10Case struct {
	Name    string // the row: ADR-001 CI-10 letter, then what is proved
	Package string // repo-relative package path
	Test    string // top-level test name
}

// ci10Cases is every TestCI10_* test in the integration run. A TestCI10_*
// test missing from this list fails the suite, so a new case gets a row.
func ci10Cases() []ci10Case {
	const pipeline, outbox = "internal/records/pipeline", "internal/outbox"
	return []ci10Case{
		{"(a) a write back to the triggering record commits", pipeline, "TestCI10_WriteBackToTriggeringRecordCommits"},
		{"(b)(c) sibling and self re-triggering writes are not cycles: one start, one cascade_cycle row per write", pipeline, "TestCI10_SiblingAndSelfRetriggeringWritesAreNotCycles"},
		{"(d) a retried activity creates no second start", pipeline, "TestCI10_RetriedActivityCreatesNoSecondStart"},
		{"(e) depth 9 is rejected before any write and audited once on the TxRunner; depth 8 is accepted", pipeline, "TestCI10_DepthBeyondLimitRejectedAndAuditedOnTheRunner"},
		{"(e) a too-deep update naming another tenant's record leaks nothing", pipeline, "TestCI10_TooDeepUpdateOfAnotherTenantsRecordLeaksNothing"},
		{"(f) the start key is per tenant", pipeline, "TestCI10_StartKeyIsPerTenant"},
		{"two transactions claiming one start key: one start", pipeline, "TestCI10_ConcurrentClaimsOfOneKey"},
		{"outbox: an entry beyond the depth limit writes nothing", outbox, "TestCI10_DepthBeyondLimitWritesNothing"},
		{"outbox: concurrent producers get consecutive target_seq", outbox, "TestCI10_ConcurrentProducersConsecutiveTargetSeq"},
		{"outbox: a duplicate dedupe key writes one row", outbox, "TestCI10_DuplicateDedupeKeyWritesOneRow"},
		{"outbox: a signal with no open workflow writes nothing", outbox, "TestCI10_SignalWithNoOpenWorkflow"},
	}
}

// ci10Row is one line of the report's CI-10 case table.
type ci10Row struct {
	Name, Test, Status, Detail string
}

// testEvent is one line of `go test -json` (test2json).
type testEvent struct {
	Action  string
	Package string
	Test    string
	Elapsed float64
}

// ci10Outcome is what the stream says about one top-level TestCI10_* test:
// its last pass, fail or skip, keyed by package and test.
type ci10Outcome struct {
	action  string
	elapsed float64
}

// readCI10Events keeps the final action of every top-level TestCI10_* test
// in a `go test -json` stream; other tests, subtests and non-JSON lines are
// ignored (the integration script has already judged the run).
func readCI10Events(r io.Reader) (map[[2]string]ci10Outcome, error) {
	out := map[[2]string]ci10Outcome{}
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 1<<20), 1<<24)
	for sc.Scan() {
		var e testEvent
		if json.Unmarshal(sc.Bytes(), &e) != nil {
			continue
		}
		if !strings.HasPrefix(e.Test, "TestCI10_") || strings.Contains(e.Test, "/") {
			continue
		}
		switch e.Action {
		case "pass", "fail", "skip":
			out[[2]string{strings.TrimPrefix(e.Package, modulePath), e.Test}] = ci10Outcome{e.Action, e.Elapsed}
		}
	}
	return out, sc.Err()
}

// ci10Rows is the CI-10 case table. With no stream (make isolation on its
// own), every row is NOT RUN. Otherwise a case passes only when its test
// passed; a failed, skipped or absent test, or a TestCI10_* test with no
// case, is a FAIL row.
func ci10Rows(events map[[2]string]ci10Outcome, haveStream bool) []ci10Row {
	var rows []ci10Row
	named := map[[2]string]bool{}
	for _, c := range ci10Cases() {
		key := [2]string{c.Package, c.Test}
		named[key] = true
		row := ci10Row{Name: c.Name, Test: c.Package + "." + c.Test}
		o, ok := events[key]
		switch {
		case !haveStream:
			row.Status, row.Detail = "NOT RUN", "make isolation on its own; make test-integration runs it first"
		case !ok:
			row.Status, row.Detail = "FAIL", "did not run in make test-integration"
		case o.action == "pass":
			row.Status, row.Detail = "PASS", fmt.Sprintf("%.2fs", o.elapsed)
		case o.action == "skip":
			row.Status, row.Detail = "FAIL", "skipped"
		default:
			row.Status, row.Detail = "FAIL", "failed; see the integration step's log"
		}
		rows = append(rows, row)
	}
	var extra [][2]string
	for key := range events {
		if !named[key] {
			extra = append(extra, key)
		}
	}
	slices.SortFunc(extra, func(a, b [2]string) int { return strings.Compare(a[0]+"."+a[1], b[0]+"."+b[1]) })
	for _, key := range extra {
		rows = append(rows, ci10Row{Name: "(no case)", Test: key[0] + "." + key[1], Status: "FAIL",
			Detail: "a TestCI10_* test with no row: add it to ci10Cases"})
	}
	return rows
}

// TestCI10CasesAreWellFormed: unique names and tests, each test exists in
// its package's integration test files.
func TestCI10CasesAreWellFormed(t *testing.T) {
	names, tests := map[string]bool{}, map[string]bool{}
	for _, c := range ci10Cases() {
		if names[c.Name] || tests[c.Package+"."+c.Test] {
			t.Errorf("duplicate case %q / %s", c.Name, c.Test)
		}
		names[c.Name], tests[c.Package+"."+c.Test] = true, true
		if !strings.HasPrefix(c.Test, "TestCI10_") {
			t.Errorf("%s: not a TestCI10_* test", c.Test)
		}
	}
	// Every TestCI10_* function in the two packages has a case, and every case
	// a function.
	found := map[string]bool{}
	for _, pkg := range []string{"internal/records/pipeline", "internal/outbox"} {
		entries, err := os.ReadDir("../../../" + pkg)
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range entries {
			if !strings.HasSuffix(e.Name(), "_test.go") {
				continue
			}
			src, err := os.ReadFile("../../../" + pkg + "/" + e.Name())
			if err != nil {
				t.Fatal(err)
			}
			for _, line := range strings.Split(string(src), "\n") {
				if rest, ok := strings.CutPrefix(line, "func TestCI10_"); ok {
					found[pkg+".TestCI10_"+rest[:strings.Index(rest, "(")]] = true
				}
			}
		}
	}
	for test := range found {
		if !tests[test] {
			t.Errorf("%s has no CI-10 case in ci10Cases", test)
		}
	}
	for test := range tests {
		if !found[test] {
			t.Errorf("ci10Cases names %s, which does not exist", test)
		}
	}
}

func TestCI10RowsFromStream(t *testing.T) {
	ev := func(action, pkg, test string) string {
		return fmt.Sprintf(`{"Action":%q,"Package":%q,"Test":%q,"Elapsed":0.5}`, action, modulePath+pkg, test)
	}
	cases := ci10Cases()
	var lines []string
	for i, c := range cases {
		switch i {
		case 0:
			lines = append(lines, ev("run", c.Package, c.Test), ev("fail", c.Package, c.Test))
		case 1:
			lines = append(lines, ev("skip", c.Package, c.Test))
		case 2: // absent
		default:
			lines = append(lines, ev("run", c.Package, c.Test), ev("pass", c.Package, c.Test))
		}
	}
	lines = append(lines,
		ev("fail", cases[3].Package, cases[3].Test+"/sub"), // a subtest: ignored
		ev("pass", "internal/outbox", "TestCI10_Unlisted"),
		ev("fail", "internal/outbox", "TestOther"),
		"FAIL\tsome build error", // plain text: ignored
	)
	events, err := readCI10Events(strings.NewReader(strings.Join(lines, "\n")))
	if err != nil {
		t.Fatal(err)
	}
	rows := ci10Rows(events, true)
	if len(rows) != len(cases)+1 {
		t.Fatalf("%d rows, want %d", len(rows), len(cases)+1)
	}
	want := map[int][2]string{0: {"FAIL", "failed"}, 1: {"FAIL", "skipped"}, 2: {"FAIL", "did not run"}, 3: {"PASS", "0.50s"}}
	for i, w := range want {
		if rows[i].Status != w[0] || !strings.Contains(rows[i].Detail, w[1]) {
			t.Errorf("row %d (%s): %s %q, want %s %q", i, rows[i].Test, rows[i].Status, rows[i].Detail, w[0], w[1])
		}
	}
	last := rows[len(rows)-1]
	if last.Status != "FAIL" || last.Test != "internal/outbox.TestCI10_Unlisted" {
		t.Errorf("unlisted test row: %+v", last)
	}

	for _, r := range ci10Rows(nil, false) {
		if r.Status != "NOT RUN" {
			t.Errorf("no stream: %s is %s, want NOT RUN", r.Test, r.Status)
		}
	}
}

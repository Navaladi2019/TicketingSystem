package main

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"
)

// stream builds a go test -json stream: every contract test passes, except the
// ones in override, which get that action ("" leaves a test out).
func stream(t *testing.T, override map[string]string) string {
	t.Helper()
	var b strings.Builder
	enc := json.NewEncoder(&b)
	emit := func(e event) {
		if err := enc.Encode(e); err != nil {
			t.Fatal(err)
		}
	}
	for _, c := range contract() {
		action, ok := override[c.Test]
		if !ok {
			action = "pass"
		}
		if action == "" {
			continue
		}
		emit(event{Action: "run", Test: c.Test})
		emit(event{Action: "output", Test: c.Test, Output: "    detail of " + c.Test + "\n"})
		emit(event{Action: "run", Test: c.Test + "/sub"})
		emit(event{Action: "pass", Test: c.Test + "/sub"})
		emit(event{Action: action, Test: c.Test, Elapsed: 1.5})
	}
	emit(event{Action: "pass", Test: "TestCIHReservedSystemWorkloadDenied"})
	return b.String()
}

func run(t *testing.T, in string) (string, []string) {
	t.Helper()
	s, err := read(strings.NewReader(in), io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	problems := report(&out, s, time.Unix(0, 0))
	return out.String(), problems
}

func TestAllPass(t *testing.T) {
	out, problems := run(t, stream(t, nil))
	if len(problems) != 0 {
		t.Fatalf("problems: %v", problems)
	}
	for _, c := range contract() {
		if !strings.Contains(out, "`"+c.Test+"` | PASS | 1 | 1.5s |") {
			t.Errorf("report lacks a PASS row for %s:\n%s", c.Test, out)
		}
		// The tag comes from the rendered row, not from c.tag(): a CI-7 case
		// must read CI-7 in the artifact.
		want := "CI-6"
		if strings.HasPrefix(c.Test, "TestCI7_") {
			want = "CI-7"
		}
		if !strings.Contains(out, " | "+want+" | "+c.Case+" | `"+c.Test+"` |") {
			t.Errorf("report row for %s does not carry tag %s:\n%s", c.Test, want, out)
		}
	}
	if !strings.Contains(out, "| CI-7 | Producer schema rejects record-data fields | `TestCI7_ProducerSchemaRejectsRecordDataFields` |") {
		t.Errorf("report does not list the CI-7 producer schema case as CI-7:\n%s", out)
	}
	if !strings.Contains(out, "| CI-6 | NotFound on a signal is terminal | `TestCI6_RelaySignalNotFoundIsTerminal` |") {
		t.Errorf("report does not list the CI-6 NotFound case as CI-6:\n%s", out)
	}
	if !strings.Contains(out, "`TestCIHReservedSystemWorkloadDenied` | PASS |") {
		t.Errorf("report lacks the other test:\n%s", out)
	}
}

func TestEveryCaseHasItsOwnCI6Test(t *testing.T) {
	seen := map[string]bool{}
	for _, c := range contract() {
		if !strings.HasPrefix(c.Test, "TestCI6_") && !strings.HasPrefix(c.Test, "TestCI7_") {
			t.Errorf("%s is not tagged CI-6 or CI-7 (TestCI6_ or TestCI7_ prefix)", c.Test)
		}
		if seen[c.Test] {
			t.Errorf("%s proves two cases", c.Test)
		}
		seen[c.Test] = true
	}
}

func TestFailures(t *testing.T) {
	first, second := contract()[0].Test, contract()[1].Test
	third := contract()[2].Test
	out, problems := run(t, stream(t, map[string]string{first: "fail", second: "skip", third: ""}))
	if len(problems) != 3 {
		t.Fatalf("problems %v, want 3", problems)
	}
	for _, want := range []string{"`" + first + "` | FAIL", "`" + second + "` | SKIP", "`" + third + "` | MISSING", "CI-6 and CI-7 FAILED", "detail of " + first} {
		if !strings.Contains(out, want) {
			t.Errorf("report lacks %q:\n%s", want, out)
		}
	}
}

const packageMsg = "the package failed (build error or panic): see the test output"

// go test -json ends every package with a failed test with a package-level
// fail; that is not a package failure.
func TestFailedTestIsNotAPackageFailure(t *testing.T) {
	first := contract()[0].Test
	in := stream(t, map[string]string{first: "fail"}) + `{"Action":"fail","Package":"x","Elapsed":3}` + "\n"
	_, problems := run(t, in)
	if len(problems) != 1 || !strings.HasPrefix(problems[0], first+": FAIL") {
		t.Fatalf("problems %v, want only the failed test", problems)
	}
}

// The same for a failed subtest of a test outside the contract.
func TestFailedSubtestIsNotAPackageFailure(t *testing.T) {
	in := stream(t, nil) +
		`{"Action":"fail","Test":"TestOther/sub"}` + "\n" +
		`{"Action":"fail","Test":"TestOther"}` + "\n" +
		`{"Action":"fail","Package":"x"}` + "\n"
	_, problems := run(t, in)
	if len(problems) != 1 || problems[0] != "TestOther: FAIL" {
		t.Fatalf("problems %v, want only the failed test", problems)
	}
}

// A go test timeout panic fails a subtest but never finishes its parent; the
// package failure is then the only signal.
func TestFailedSubtestOfUnfinishedTestIsAPackageFailure(t *testing.T) {
	in := stream(t, nil) +
		`{"Action":"run","Test":"TestOther"}` + "\n" +
		`{"Action":"run","Test":"TestOther/sub"}` + "\n" +
		`{"Action":"fail","Test":"TestOther/sub"}` + "\n" +
		`{"Action":"output","Package":"x","Output":"panic: test timed out after 10m0s\n"}` + "\n" +
		`{"Action":"fail","Package":"x","Elapsed":600}` + "\n"
	_, problems := run(t, in)
	if len(problems) != 1 || problems[0] != packageMsg {
		t.Fatalf("problems %v, want the package failure", problems)
	}
}

// Go 1.24+ reports build errors as JSON (build-output, build-fail), then a
// package fail with no test events.
func TestJSONBuildFailure(t *testing.T) {
	in := `{"ImportPath":"x","Action":"build-output","Output":"./a.go:1: syntax error\n"}` + "\n" +
		`{"ImportPath":"x","Action":"build-fail"}` + "\n" +
		`{"Action":"output","Package":"x","Output":"FAIL\tx [build failed]\n"}` + "\n" +
		`{"Action":"fail","Package":"x","FailedBuild":"x"}` + "\n"
	_, problems := run(t, in)
	if len(problems) != len(contract())+1 || problems[len(problems)-1] != packageMsg {
		t.Fatalf("problems %v, want every case missing plus the package", problems)
	}
}

// A package that fails after every test passed (a panic in TestMain, a leak
// check) is a package failure.
func TestPackageFailsWithNoFailedTest(t *testing.T) {
	_, problems := run(t, stream(t, nil)+`{"Action":"fail","Package":"x"}`+"\n")
	if len(problems) != 1 || problems[0] != packageMsg {
		t.Fatalf("problems %v, want the package failure", problems)
	}
}

func TestBuildFailure(t *testing.T) {
	_, problems := run(t, "# github.com/x\n./a.go:1: syntax error\n"+`{"Action":"fail","Package":"x"}`+"\n")
	if len(problems) != len(contract())+1 || problems[len(problems)-1] != packageMsg {
		t.Fatalf("problems %v, want every case missing plus the package", problems)
	}
}

// Plain text in the stream is a package failure even when every test passed.
func TestNonJSONOutputIsAPackageFailure(t *testing.T) {
	_, problems := run(t, stream(t, nil)+"panic: something outside test2json\n")
	if len(problems) != 1 || problems[0] != packageMsg {
		t.Fatalf("problems %v, want the package failure", problems)
	}
}

func TestSkippedSubtestFails(t *testing.T) {
	name := contract()[4].Test
	in := stream(t, nil) + `{"Action":"skip","Test":"` + name + `/nava-core"}` + "\n"
	out, problems := run(t, in)
	if len(problems) != 1 || !strings.Contains(problems[0], name+"/nava-core: SKIP") {
		t.Fatalf("problems %v, want the skipped subtest", problems)
	}
	if !strings.Contains(out, "subtest(s) SKIPPED") {
		t.Errorf("report does not show the skipped subtest:\n%s", out)
	}
}

func TestSkippedOtherTestFails(t *testing.T) {
	in := strings.Replace(stream(t, nil), `"Action":"pass","Package":"","Test":"TestCIHReservedSystemWorkloadDenied"`,
		`"Action":"skip","Package":"","Test":"TestCIHReservedSystemWorkloadDenied"`, 1)
	_, problems := run(t, in)
	if len(problems) != 1 || problems[0] != "TestCIHReservedSystemWorkloadDenied: SKIP" {
		t.Fatalf("problems %v, want the skipped CI-H test", problems)
	}
	in = stream(t, nil) + `{"Action":"skip","Test":"TestCIHReservedSystemWorkloadDenied/sub"}` + "\n"
	if _, problems := run(t, in); len(problems) != 1 {
		t.Fatalf("problems %v, want the skipped CI-H subtest", problems)
	}
}

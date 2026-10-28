package main

import (
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/Navaladi2019/TicketingSystem/internal/identity/contracttest"
)

// stream builds `go test -json` lines: one passing tagged subtest per CI-5
// case, minus skip, with extra events appended.
func stream(t *testing.T, skip string, extra ...event) string {
	t.Helper()
	var b strings.Builder
	enc := json.NewEncoder(&b)
	for _, c := range contracttest.CI5Cases() {
		if c.ID == skip {
			continue
		}
		_ = enc.Encode(event{Action: "pass", Test: "TestCI5/Scenarios/[" + c.ID + "]_case"})
	}
	for _, e := range extra {
		_ = enc.Encode(e)
	}
	_ = enc.Encode(event{Action: "pass", Test: "TestCI5"})
	return b.String()
}

func run(t *testing.T, in string) ([]string, string) {
	t.Helper()
	s, err := read(strings.NewReader(in), io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	var out strings.Builder
	return report(&out, s, time.Unix(0, 0)), out.String()
}

func TestAllCasesPass(t *testing.T) {
	problems, md := run(t, stream(t, ""))
	if len(problems) != 0 {
		t.Fatalf("problems: %v", problems)
	}
	if !strings.Contains(md, "All 14 CI-5 cases pass") {
		t.Errorf("report: %s", md)
	}
}

func TestMissingCaseFails(t *testing.T) {
	problems, md := run(t, stream(t, contracttest.CI5OrgPickerTamper))
	if len(problems) != 1 || !strings.Contains(problems[0], "CI-5.06: MISSING") || !strings.Contains(md, "MISSING") {
		t.Fatalf("problems %v", problems)
	}
}

func TestFailedOrSkippedCaseFails(t *testing.T) {
	for _, action := range []string{"fail", "skip"} {
		problems, _ := run(t, stream(t, contracttest.CI5DuplicateBinding,
			event{Action: action, Test: "TestCI5/Scenarios/[CI-5.12]_dup"}))
		if len(problems) != 1 || !strings.Contains(problems[0], "CI-5.12") {
			t.Errorf("%s: problems %v", action, problems)
		}
	}
}

// A tag must be the start of the last name element: a parent named after a
// case does not prove it.
func TestTagOnlyOnLastElement(t *testing.T) {
	problems, _ := run(t, stream(t, contracttest.CI5UnknownIssuer,
		event{Action: "pass", Test: "TestCI5/[CI-5.03]_parent/child"}))
	if len(problems) != 1 || !strings.Contains(problems[0], "CI-5.03: MISSING") {
		t.Errorf("problems %v", problems)
	}
}

func TestOtherFailuresAndBrokenPackageFail(t *testing.T) {
	problems, _ := run(t, stream(t, "", event{Action: "skip", Test: "TestCI5/OperatorResolver"}))
	if len(problems) != 1 || !strings.Contains(problems[0], "OperatorResolver: SKIP") {
		t.Errorf("skipped untagged subtest: %v", problems)
	}
	problems, _ = run(t, "# build failed\n"+stream(t, ""))
	if len(problems) != 1 || !strings.Contains(problems[0], "package failed") {
		t.Errorf("build error: %v", problems)
	}
}

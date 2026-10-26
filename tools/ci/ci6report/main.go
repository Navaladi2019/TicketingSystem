// Command ci6report reads `go test -json` of the Temporal stack suite
// (internal/temporalstack) on stdin, prints the test output as it streams, and
// writes the CI-6 report (PLAN-001 tasks T32a and T32b): every contract case of
// ADR-001 Annex B CI-6 (first block with erratum E1, then the relay and worker
// cases) and CI-7, with the named test that proves it and its result. It exits 1 when a case's test failed, was skipped or did not run,
// or had a skipped subtest; when any other test in the run failed or was
// skipped; or when the package failed (a build error, a panic).
//
// Usage: go test -tags temporalstack -json ./internal/temporalstack | ci6report -out ci6-report.md
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"
)

// contractCase is one CI-6 or CI-7 contract case and the test that proves it;
// the tag is the TestCI6_ or TestCI7_ prefix.
type contractCase struct {
	Case, Test string
}

// tag is the case's Annex B row, from its test name.
func (c contractCase) tag() string {
	if strings.HasPrefix(c.Test, "TestCI7_") {
		return "CI-7"
	}
	return "CI-6"
}

// contract is the CI-6 first block, in the order of the T32a issue, then the
// CI-6 relay and worker cases and CI-7, in the order of the T32b issue. The
// first block lives in internal/temporalstack/ci6_isolation_test.go, except
// its last case, which T28 wrote in stack_test.go; the T32b cases live in
// ci6_relay_isolation_test.go and ci7_isolation_test.go.
func contract() []contractCase {
	return []contractCase{
		{"Unlisted certificate (installation CA, CN no identity) denied", "TestCI6_UnlistedCertificateDeniedByAuthorizer"},
		{"Rogue-CA certificate rejected at the handshake", "TestCI6_RogueCACertificateRejectedAtHandshake"},
		{"Platform network cannot reach the internal ports", "TestCI6_PlatformNetworkCannotReachInternalPorts"},
		{"Platform certificate refused at the internode handshake (T28 handoff)", "TestCI6_PlatformCertificateRejectedAtInternodeHandshake"},
		{"Relay, core and worker certificates outside their allow-list denied", "TestCI6_RelayCoreWorkerOutsideAllowListDenied"},
		{"nava-operator limited to its pinned list (ReapplyEvents, ImportWorkflowExecution, DeleteWorkflowExecution, DeleteNamespace denied)", "TestCI6_OperatorLimitedToPinnedList"},
		{"Worker cross-tenant child denied, with an alert", "TestCI6_WorkerCrossTenantChildDeniedWithAlert"},
		{"Worker cross-tenant signal-external denied, with an alert", "TestCI6_WorkerCrossTenantSignalExternalDeniedWithAlert"},
		{"Worker cross-tenant cancel-external denied, with an alert", "TestCI6_WorkerCrossTenantCancelExternalDeniedWithAlert"},
		{"Continue-as-new with a missing nava-tenant header denied (E1)", "TestCI6_WorkerContinueAsNewMissingTenantHeaderDenied"},
		{"Continue-as-new with a mismatched nava-tenant header denied (E1)", "TestCI6_WorkerContinueAsNewMismatchedTenantHeaderDenied"},
		{"Continue-as-new with cron_schedule denied (E1)", "TestCI6_WorkerContinueAsNewCronScheduleDenied"},
		{"Unlisted command type (Nexus, future type) denied (E1)", "TestCI6_WorkerUnlistedCommandTypeDenied"},
		{"Undecodable task token denied (E1)", "TestCI6_WorkerUndecodableTaskTokenDenied"},
		{"Task of a workflow id without a tenant prefix denied (E1)", "TestCI6_WorkerTaskWithoutTenantPrefixDenied"},
		{"Allowed: same-tenant child completes", "TestCI6_Allowed_SameTenantChildCompletes"},
		{"Allowed: every SDK client dials and its worker starts", "TestCI6_Allowed_EverySDKClientDialsAndItsWorkerStarts"},
		{"No or mismatched nava-tenant header on a start fails the workflow", "TestCI6_WorkerStartWithMissingOrMismatchedHeaderFailsWorkflow"},
		{"Wrong-tenant signal dropped with an alert; the workflow keeps running", "TestCI6_WorkerWrongTenantSignalDroppedWithAlertWorkflowKeepsRunning"},
		{"Redelivered start after completion starts no second run", "TestCI6_RelayRedeliveredStartAfterCompletionStartsNoSecondRun"},
		{"Redelivered signal applied once, including across continue-as-new", "TestCI6_RelayRedeliveredSignalAppliedOnceAcrossContinueAsNew"},
		{"Signal queues behind its start", "TestCI6_RelaySignalQueuesBehindItsStart"},
		{"Two relays never reorder one target", "TestCI6_RelayTwoRelaysNeverReorderOneTarget"},
		{"NotFound on a signal is terminal", "TestCI6_RelaySignalNotFoundIsTerminal"},
		{"Producer schema rejects record-data fields", "TestCI7_ProducerSchemaRejectsRecordDataFields"},
		{"Workflow input and signal bodies in history are references only", "TestCI7_RelayDeliveryInHistoryIsReferencesOnly"},
		{"Record-data payload written outside Enqueue refused; nothing reaches the history (TICK-214)", "TestCI7_RelayRefusesRecordDataPayloadWrittenOutsideEnqueue"},
		{"Recorded headers carry only nava-tenant and trace context, never baggage (TICK-93)", "TestCI7_RecordedHeadersCarryOnlyTenantAndTraceContext"},
	}
}

// event is one line of `go test -json` (test2json).
type event struct {
	Action  string
	Package string
	Test    string
	Elapsed float64
	Output  string
}

// result is what the stream says about one top-level test.
type result struct {
	Action      string // pass, fail, skip; "" if it never finished
	Elapsed     float64
	Subs        int      // subtests that passed
	SkippedSubs []string // subtests that were skipped; a skip proves nothing
}

// summary is the whole run.
type summary struct {
	tests      map[string]*result
	packageBad bool
	failOutput map[string][]string // test -> its output lines, kept for failures
}

func read(r io.Reader, echo io.Writer) (*summary, error) {
	s := &summary{tests: map[string]*result{}, failOutput: map[string][]string{}}
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 1<<20), 1<<24)
	for sc.Scan() {
		var e event
		if err := json.Unmarshal(sc.Bytes(), &e); err != nil {
			// go test prints build errors as plain text.
			_, _ = fmt.Fprintln(echo, sc.Text())
			s.packageBad = true
			continue
		}
		if e.Output != "" {
			_, _ = io.WriteString(echo, e.Output)
		}
		if e.Test == "" {
			if e.Action == "fail" {
				s.packageBad = true
			}
			continue
		}
		top, sub, isSub := strings.Cut(e.Test, "/")
		res := s.tests[top]
		if res == nil {
			res = &result{}
			s.tests[top] = res
		}
		if e.Action == "output" {
			s.failOutput[top] = append(s.failOutput[top], strings.TrimRight(e.Output, "\n"))
		}
		switch e.Action {
		case "pass", "fail", "skip":
			if isSub {
				switch {
				case sub == "":
				case e.Action == "pass":
					res.Subs++
				case e.Action == "skip":
					res.SkippedSubs = append(res.SkippedSubs, e.Test)
				}
				continue
			}
			res.Action, res.Elapsed = e.Action, e.Elapsed
		}
	}
	return s, sc.Err()
}

// report writes the markdown report and returns the problems that fail the run.
func report(w io.Writer, s *summary, now time.Time) []string {
	var problems []string
	p := func(format string, a ...any) { _, _ = fmt.Fprintf(w, format, a...) }
	p("# CI-6 and CI-7 report: certificate, network, Authorizer, relay and worker suite\n\n")
	p("Generated by `make stack-test-sdk` (tools/ci/ci6report) on %s, against the compose Temporal over mTLS (no stubbed Authorizer). ", now.UTC().Format(time.RFC3339))
	p("ADR-001 Annex B CI-6 (first block with erratum E1; relay and worker cases) and CI-7; PLAN-001 tasks T32a and T32b.\n\n")
	p("| # | Tag | Contract case | Test | Result | Subtests | Time |\n|---|---|---|---|---|---|---|\n")
	listed := map[string]bool{}
	for i, c := range contract() {
		listed[c.Test] = true
		res := s.tests[c.Test]
		status, subs, elapsed := "MISSING", "", ""
		if res != nil {
			status = strings.ToUpper(res.Action)
			if status == "" {
				status = "DID NOT FINISH"
			}
			if res.Subs > 0 {
				subs = fmt.Sprint(res.Subs)
			}
			elapsed = fmt.Sprintf("%.1fs", res.Elapsed)
		}
		if status != "PASS" {
			problems = append(problems, fmt.Sprintf("%s: %s (%s)", c.Test, status, c.Case))
		}
		if res != nil && len(res.SkippedSubs) > 0 {
			status += fmt.Sprintf(", %d subtest(s) SKIPPED", len(res.SkippedSubs))
			for _, sub := range res.SkippedSubs {
				problems = append(problems, sub+": SKIP ("+c.Case+")")
			}
		}
		p("| %d | %s | %s | `%s` | %s | %s | %s |\n", i+1, c.tag(), c.Case, c.Test, status, subs, elapsed)
	}
	var others []string
	for name := range s.tests {
		if !listed[name] {
			others = append(others, name)
		}
	}
	sort.Strings(others)
	if len(others) > 0 {
		p("\nOther tests in the same run:\n\n| Test | Result |\n|---|---|\n")
		for _, name := range others {
			res := s.tests[name]
			p("| `%s` | %s |\n", name, strings.ToUpper(res.Action))
			switch {
			case res.Action == "fail", res.Action == "skip":
				problems = append(problems, name+": "+strings.ToUpper(res.Action))
			case len(res.SkippedSubs) > 0:
				problems = append(problems, fmt.Sprintf("%s: %d subtest(s) SKIPPED", name, len(res.SkippedSubs)))
			}
		}
	}
	if s.packageBad {
		problems = append(problems, "the package failed (build error or panic): see the test output")
	}
	if len(problems) == 0 {
		p("\n**All %d CI-6 and CI-7 cases pass.**\n", len(contract()))
		return nil
	}
	p("\n**CI-6 and CI-7 FAILED:**\n\n")
	for _, pr := range problems {
		p("- %s\n", pr)
	}
	for _, c := range contract() {
		if res := s.tests[c.Test]; res != nil && res.Action == "fail" {
			p("\n<details><summary>%s output</summary>\n\n```\n%s\n```\n</details>\n", c.Test, strings.Join(s.failOutput[c.Test], "\n"))
		}
	}
	return problems
}

func main() {
	out := flag.String("out", "ci6-report.md", "report file")
	flag.Parse()
	s, err := read(os.Stdin, os.Stdout)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ci6report:", err)
		os.Exit(2)
	}
	f, err := os.Create(*out)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ci6report:", err)
		os.Exit(2)
	}
	problems := report(f, s, time.Now())
	if err := f.Close(); err != nil {
		fmt.Fprintln(os.Stderr, "ci6report:", err)
		os.Exit(2)
	}
	fmt.Println("ci6report: wrote", *out)
	for _, pr := range problems {
		fmt.Fprintln(os.Stderr, "ci6report:", pr)
	}
	if len(problems) > 0 {
		os.Exit(1)
	}
}

//go:build integration

package isolation

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// report is what make isolation writes as the CI artifact.
type report struct {
	ci2      []ci2Line
	paths    []pathResult
	fixtures []fixtureResult
	running  string // the registered path TestRegisteredPaths is in, if any
}

// Report results. Every write before TestMain's last one says
// resultIncomplete (TICK-1180): a panic or the go test -timeout kills the
// test binary without returning to TestMain, so the copy on disk must already
// hold every row that finished.
const (
	resultPass       = "PASS"
	resultFail       = "FAIL"
	resultIncomplete = "INCOMPLETE"
	statusRunning    = "RUNNING"
	statusNotRun     = "NOT RUN"
)

// setCI2, addPath, addFixture and startPath record a row and rewrite the
// report at once, so the artifact keeps it whatever stops the binary later.
func (e *env) setCI2(lines []ci2Line) {
	e.report.ci2 = lines
	e.flushReport()
}

func (e *env) addPath(res pathResult) {
	e.report.paths = append(e.report.paths, res)
	e.report.running = ""
	e.flushReport()
}

func (e *env) addFixture(f fixtureResult) {
	e.report.fixtures = append(e.report.fixtures, f)
	e.flushReport()
}

func (e *env) startPath(name string) {
	e.report.running = name
	e.flushReport()
}

// flushReport rewrites the report as INCOMPLETE. TestMain's last write
// decides the run; a failed write here is only logged.
func (e *env) flushReport() {
	if err := writeReport(e.reportPath, e.report.markdown(resultIncomplete, "")); err != nil {
		fmt.Fprintln(os.Stderr, "isolation report:", err)
	}
}

// writeReport replaces path through a rename, so a binary killed mid-write
// leaves the previous whole report, never half of one.
func writeReport(path, md string) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	_, err = tmp.WriteString(md)
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Chmod(tmp.Name(), 0o600)
	}
	if err == nil {
		err = os.Rename(tmp.Name(), path)
	}
	if err != nil {
		_ = os.Remove(tmp.Name())
	}
	return err
}

// failed reports whether any row recorded so far failed. A fixture fails
// when it is not PASS before and FAIL with the fixture.
func (r report) failed() bool {
	for _, l := range r.ci2 {
		if l.Status == statusFail {
			return true
		}
	}
	for _, p := range r.paths {
		if p.Status == statusFail {
			return true
		}
	}
	for _, f := range r.fixtures {
		if f.Before != statusPass || f.After != statusFail {
			return true
		}
	}
	return false
}

// markdown groups everything by ADR test id: CI-2's generic lines (one per
// role × table) and fixtures, then each id's registered paths, with its
// coverage note, so a partial id shows as partial. A registered path with no
// result shows as RUNNING (the one under way) or NOT RUN. setupErr, when
// set, is why there are no rows.
func (r report) markdown(result, setupErr string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Tenant isolation suite (T11)\n\nResult: **%s**. Tenant A `%s`, tenant B `%s`.\n\n", result, TenantA, TenantB)
	switch {
	case setupErr != "":
		fmt.Fprintf(&b, "The suite's setup failed, so no check ran: %s\n\n", cellText(setupErr))
	case result == resultIncomplete:
		b.WriteString("The test binary stopped before the end: a panic or the go test -timeout (see the job log). " +
			"Every row below with a result finished before that; RUNNING is the path it was in.")
		if r.failed() {
			b.WriteString(" **Rows had already failed.**")
		}
		b.WriteString("\n\n")
	}
	b.WriteString("Statuses: PASS (0 rows or an error), FAIL (a row was reached), " +
		"ALLOWLISTED (reached by an ADR-001 A.3 policy that does not test the tenant, by design), PENDING (registered, not runnable yet), " +
		"RUNNING (under way when the binary stopped), NOT RUN (registered, no result in this run).\n")

	cov := Coverage()
	ids := make([]string, 0, len(cov))
	for id := range cov {
		ids = append(ids, id)
	}
	slices.SortFunc(ids, func(a, b string) int { return idNum(a) - idNum(b) })
	for _, id := range ids {
		fmt.Fprintf(&b, "\n## %s\n\nCoverage: %s.\n", id, cov[id])
		if id == "CI-2" {
			r.writeCI2(&b)
		}
		var rows []pathResult
		done := map[string]bool{}
		for _, p := range r.paths {
			if p.Path.Case == id {
				rows = append(rows, p)
				done[p.Path.Name] = true
			}
		}
		for _, p := range Paths() {
			if p.Case != id || done[p.Name] {
				continue
			}
			res := pathResult{Path: p, Status: statusNotRun}
			if p.Name == r.running {
				res.Status, res.Detail = statusRunning, "under way when the report was last written"
			}
			rows = append(rows, res)
		}
		if len(rows) == 0 {
			b.WriteString("\nNo registered path.\n")
			continue
		}
		b.WriteString("\n### Registered paths\n\n| Path | Role | Status | Detail |\n|---|---|---|---|\n")
		for _, p := range rows {
			role := p.Path.Role
			if role == "" {
				role = "-"
			}
			fmt.Fprintf(&b, "| %s | %s | %s | %s |\n", cellText(p.Path.Name), role, p.Status, cellText(p.Detail))
		}
		if id == "CI-9" {
			r.writeFixtures(&b, id)
		}
	}
	return b.String()
}

func (r report) writeCI2(b *strings.Builder) {
	b.WriteString("\n### Every runtime role × every tenant table\n\n" +
		"Each granted SELECT, UPDATE and DELETE, run with no tenant, `''` and after a prior SET LOCAL of tenant A " +
		"(directly and through PgBouncer), then with tenant A set through tenancy.InTenantTx, aimed at other tenants' rows.\n\n" +
		"| Role | Table | Status | Detail |\n|---|---|---|---|\n")
	if len(r.ci2) == 0 {
		b.WriteString("| - | - | FAIL | the generic checks did not run |\n")
	}
	for _, l := range r.ci2 {
		fmt.Fprintf(b, "| %s | %s | %s | %s |\n", l.Role, l.Table, l.Status, cellText(l.Detail))
	}
	r.writeFixtures(b, "CI-2")
}

// writeFixtures lists the fixtures of ADR test id id.
func (r report) writeFixtures(b *strings.Builder, id string) {
	b.WriteString("\n### Fixtures (the checks can fail)\n\n| Fixture | Role | Table | Before | With the fixture |\n|---|---|---|---|---|\n")
	n := 0
	for _, f := range r.fixtures {
		if f.Case == id {
			n++
			fmt.Fprintf(b, "| %s | %s | %s | %s | %s |\n", cellText(f.Name), f.Role, f.Table, f.Before, f.After)
		}
	}
	if n == 0 {
		b.WriteString("| - | - | - | - | did not run |\n")
	}
}

// idNum orders ADR test ids by number; a task id such as T26 comes after them.
func idNum(id string) int {
	var n int
	if _, err := fmt.Sscanf(id, "CI-%d", &n); err != nil {
		_, _ = fmt.Sscanf(id, "T%d", &n)
		return 1000 + n
	}
	return n
}

func cellText(s string) string {
	return strings.NewReplacer("|", `\|`, "\n", " ").Replace(s)
}

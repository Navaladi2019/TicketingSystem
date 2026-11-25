//go:build integration && perf

package server_test

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// A reported-only plan never ends the test (TICK-1098): the reason goes into
// plans.md and both arms are still tried. No database: the pool's address
// refuses connections.
func TestE3ReportedPlanKeepsGoing(t *testing.T) {
	pool, err := pgxpool.New(t.Context(), "postgres://u:p@127.0.0.1:1/x?connect_timeout=2")
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	savepoint := func(stmts ...string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			tr, _ := r.Context().Value(perfTraceKey{}).(*perfTrace)
			for _, q := range stmts {
				tr.stmts = append(tr.stmts, perfStmt{SQL: q})
			}
			w.WriteHeader(http.StatusOK)
		}
	}
	for _, c := range []struct {
		name string
		h    http.HandlerFunc
		want string
	}{
		{"non-200", func(w http.ResponseWriter, _ *http.Request) { http.Error(w, "boom", http.StatusInternalServerError) }, "500"},
		{"no savepoint", savepoint("SELECT 1"), "no savepoint Exec"},
		{"savepoint is the last statement", savepoint("SAVEPOINT list_count"), "no savepoint Exec"},
		{"EXPLAIN error", savepoint("SAVEPOINT list_count; SET LOCAL statement_timeout = '170'", "SELECT count(*)"), "tenancy: begin"},
	} {
		var plans, lines []string
		saved := 0
		s := &e3Serial{e: &fiEnv{perfEnv: &perfEnv{pool: pool}}, tenant: uuid.New(), plans: &plans, onPlans: func() { saved++ },
			w:    func(f string, a ...any) { lines = append(lines, fmt.Sprintf(f, a...)) },
			arms: map[string]e3Arm{"A": {h: c.h}, "B": {h: c.h}}}
		s.planReported(t, "agent-1 (reported only)", "agent-1", "q=1")
		if len(plans) != 2 || saved != 2 || len(lines) != 2 {
			t.Errorf("%s: %d plans, %d saves, %d lines; want 2 of each", c.name, len(plans), saved, len(lines))
			continue
		}
		for i, arm := range []string{"arm A", "arm B"} {
			if !strings.Contains(plans[i], arm) || !strings.Contains(plans[i], "not recorded") || !strings.Contains(plans[i], c.want) {
				t.Errorf("%s: plans.md entry %q, want %q, \"not recorded\" and %q", c.name, plans[i], arm, c.want)
			}
		}
	}
}

// A smoke run says so and never prints a verdict word, whatever its numbers
// are, and neither does any line its writer passes on.
func TestE3SmokePrintsNoVerdict(t *testing.T) {
	block := func(unknown bool) []e3Block {
		st := "exact"
		if unknown {
			st = "unknown"
		}
		return []e3Block{{samples: []e3Sample{{code: http.StatusOK, state: st, latency: 900}, {code: http.StatusInternalServerError}}}}
	}
	word := regexp.MustCompile(`PASS|FAIL|INCONCLUSIVE`)
	for _, c := range []struct {
		name  string
		cells []*e3CellResult
	}{
		{"share miss", []*e3CellResult{{name: "x", band: true, blocks: map[string][]e3Block{"A": block(true), "B": block(false)}}}},
		{"equal", []*e3CellResult{{name: "x", blocks: map[string][]e3Block{"A": block(false), "B": block(false)}}}},
		{"no cells", nil},
	} {
		var lines []string
		s := &e3Serial{smoke: true, cells: c.cells, arms: map[string]e3Arm{}, w: func(f string, a ...any) { lines = append(lines, fmt.Sprintf(f, a...)) }}
		overall, kind := s.report()
		all := strings.Join(append(lines, overall, kind), "\n")
		if word.MatchString(all) || !strings.Contains(all, "SMOKE, no verdict") {
			t.Errorf("%s: %s", c.name, all)
		}
	}
	for _, l := range []string{gateHost{floorP50: 99, controlP95: 999}.String(), "**(3a) leak verdict after (3e): PASS**", "INCONCLUSIVE (host)"} {
		if got := e3SmokeLine(l); word.MatchString(got) {
			t.Errorf("e3SmokeLine(%q) = %q", l, got)
		}
	}
}

// The cell names hold commas: a row has to keep the header's columns.
func TestE3CSVRow(t *testing.T) {
	row := e3CSVRow("a50, pk = 3", 2, 1, "B", true, e3Sample{code: http.StatusOK, state: "unknown", phase: 1, countStmt: 2, latency: 3})
	rows, err := csv.NewReader(strings.NewReader(e3CSVHeader + row)).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[1][0] != "a50, pk = 3" || rows[1][3] != "B" || rows[1][5] != "200" || rows[1][9] != "3.000" {
		t.Errorf("%q", rows)
	}
}

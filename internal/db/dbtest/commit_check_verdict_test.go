package dbtest

// The verdict rule of the commit-check budget rows. The method and the
// thresholds are ADR-004-E3's ("Budget"); the verdict rule is ADR-004-E4's
// (docs/adr/ADR-004-E4-commit-check-ceilings-report-only.md, "Verdict
// table"): in CI the absolute ceilings are report-only review thresholds,
// and a run fails only when nothing was measured, a same-run ratio is out
// of range, or the per-call row is not flat. This file is untagged, so
// `task test` unit-tests the rule on fixed inputs; TestCommitCheckBudget
// (integration) applies it to the numbers measured in CI. The constants keep
// E3's values: nobody moves them to make CI pass (E3, D-E3-6).

import (
	"fmt"
	"math"
	"os"
	"regexp"
	"strings"
	"testing"
)

const (
	// budgetPerCallCeiling is the single-subject row's review threshold, µs
	// of self time per call (ADR-004-E3); report-only in CI (ADR-004-E4).
	budgetPerCallCeiling = 300.0
	// budgetFlatLimit: the clause arm's max/min across the three N must be
	// below it, or the run fails (fail_flat).
	budgetFlatLimit = 1.5
	// budgetImportCeiling is the import row's review threshold, µs of
	// COMMIT time per probe (ADR-004-E3); report-only in CI (ADR-004-E4).
	budgetImportCeiling = 175.0
	// budgetImportRatioMax and budgetPerCallRatioMax bound the same-run
	// ratios, clause / reference; above them the run fails (fail_ratio).
	budgetImportRatioMax  = 15.0
	budgetPerCallRatioMax = 5.0
)

// commitCheckBudget is one run's numbers. Clause is the shipped
// audit_commit_check (with SET plan_cache_mode = force_custom_plan); Ref is
// the same-run reference, a copy of it without the clause, on the same
// fixture in the same job.
type commitCheckBudget struct {
	PerCallClause, PerCallRef [3]float64 // µs per call at N = 200, 1,000, 5,000
	ImportClause, ImportRef   float64    // µs per probe, median of 3 imports
}

// budgetVerdict is the run's verdict (ADR-004-E4 verdict table). The order
// is the table's: the first failing condition wins.
type budgetVerdict int

const (
	budgetFailNoRows budgetVerdict = iota // a measured value is ≤ 0, NaN or missing
	budgetFailRatio                       // a same-run ratio is out of range
	budgetFailFlat                        // clause per call max/min ≥ 1.5
	budgetWarn                            // a row over its review threshold; the job passes
	budgetPass
)

func (v budgetVerdict) String() string {
	switch v {
	case budgetFailNoRows:
		return "fail_no_rows"
	case budgetFailRatio:
		return "fail_ratio"
	case budgetFailFlat:
		return "fail_flat"
	case budgetWarn:
		return "warn"
	case budgetPass:
		return "pass"
	}
	return fmt.Sprintf("budgetVerdict(%d)", int(v))
}

// failed reports whether the verdict fails the job.
func (v budgetVerdict) failed() bool { return v < budgetWarn }

// measured: every value is a positive number (not NaN, not ±Inf).
func (b commitCheckBudget) measured() bool {
	vs := append(append([]float64{}, b.PerCallClause[:]...), b.PerCallRef[:]...)
	for _, v := range append(vs, b.ImportClause, b.ImportRef) {
		if !(v > 0) || math.IsInf(v, 0) {
			return false
		}
	}
	return true
}

// flatness is max/min of the clause arm's per-call values.
func (b commitCheckBudget) flatness() float64 {
	lo, hi := b.PerCallClause[0], b.PerCallClause[0]
	for _, v := range b.PerCallClause[1:] {
		lo, hi = math.Min(lo, v), math.Max(hi, v)
	}
	return hi / lo
}

// perCallRatio is clause / reference per call at N = 200.
func (b commitCheckBudget) perCallRatio() float64 { return b.PerCallClause[0] / b.PerCallRef[0] }

// importRatio is clause / reference per probe on the import.
func (b commitCheckBudget) importRatio() float64 { return b.ImportClause / b.ImportRef }

// ratioOutOfRange: per call at N = 200 > 5, or import > 15; NaN counts as
// out of range.
func (b commitCheckBudget) ratioOutOfRange() bool {
	return !(b.perCallRatio() <= budgetPerCallRatioMax) || !(b.importRatio() <= budgetImportRatioMax)
}

func (b commitCheckBudget) notFlat() bool { return !(b.flatness() < budgetFlatLimit) }

// singleOver: any of N = 200, 1,000, 5,000 is over the per-call review
// threshold (E3's singleWithin, without flatness, which now fails alone).
func (b commitCheckBudget) singleOver() bool {
	for _, v := range b.PerCallClause {
		if v > budgetPerCallCeiling {
			return true
		}
	}
	return false
}

func (b commitCheckBudget) importOver() bool { return b.ImportClause > budgetImportCeiling }

// failures is every failing condition, in table order; the first is the
// verdict.
func (b commitCheckBudget) failures() []budgetVerdict {
	if !b.measured() {
		return []budgetVerdict{budgetFailNoRows}
	}
	var fs []budgetVerdict
	if b.ratioOutOfRange() {
		fs = append(fs, budgetFailRatio)
	}
	if b.notFlat() {
		fs = append(fs, budgetFailFlat)
	}
	return fs
}

// verdict is the run's verdict under the ADR-004-E4 table.
func (b commitCheckBudget) verdict() budgetVerdict {
	if fs := b.failures(); len(fs) > 0 {
		return fs[0]
	}
	if b.singleOver() || b.importOver() {
		return budgetWarn
	}
	return budgetPass
}

// cpuModel is NAVA_BUDGET_CPU (scripts/ci/commit-check-budget.sh reads it
// from the runner), else the first "model name" in /proc/cpuinfo, else
// "unknown".
func cpuModel() string {
	if m := os.Getenv("NAVA_BUDGET_CPU"); m != "" {
		return m
	}
	b, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return "unknown"
	}
	if m := regexp.MustCompile(`(?m)^model name\s*:\s*(.+)$`).FindSubmatch(b); m != nil {
		return strings.TrimSpace(string(m[1]))
	}
	return "unknown"
}

// singleLine and importLine are the greppable log lines, one per row. Each
// names the run's verdict and whether that row is over its review threshold;
// scripts/ci/commit-check-budget.sh turns an over_threshold=true line of a
// warn run into a ::warning:: annotation.
func (b commitCheckBudget) singleLine(partitions int, cpu string) string {
	return fmt.Sprintf("COMMIT-CHECK-BUDGET row=single_subject partitions=%d cpu=%q "+
		"clause_us_n200=%.1f clause_us_n1000=%.1f clause_us_n5000=%.1f max_min=%.2f "+
		"ref_us_n200=%.1f ref_us_n1000=%.1f ref_us_n5000=%.1f ratio_n200=%.2f ratio_import=%.2f "+
		"threshold_us=%.0f flat_below=%.1f over_threshold=%t verdict=%s",
		partitions, cpu, b.PerCallClause[0], b.PerCallClause[1], b.PerCallClause[2], b.flatness(),
		b.PerCallRef[0], b.PerCallRef[1], b.PerCallRef[2], b.perCallRatio(), b.importRatio(),
		budgetPerCallCeiling, budgetFlatLimit, b.singleOver(), b.verdict())
}

func (b commitCheckBudget) importLine(partitions int, cpu string, clauseRuns, refRuns []float64) string {
	return fmt.Sprintf("COMMIT-CHECK-BUDGET row=import partitions=%d cpu=%q subjects=10000 "+
		"clause_us_per_probe=%.1f clause_runs=%.1f ref_us_per_probe=%.1f ref_runs=%.1f "+
		"ratio_import=%.2f ratio_n200=%.2f threshold_us=%.0f over_threshold=%t verdict=%s",
		partitions, cpu, b.ImportClause, clauseRuns, b.ImportRef, refRuns,
		b.importRatio(), b.perCallRatio(), budgetImportCeiling, b.importOver(), b.verdict())
}

func TestCommitCheckBudgetVerdict(t *testing.T) {
	// in is a run under both thresholds with in-range ratios, flat (CI run
	// 41673342312); each case changes what it names.
	in := commitCheckBudget{
		PerCallClause: [3]float64{170.5, 163.6, 164.8}, PerCallRef: [3]float64{133.5, 82.1, 212.0},
		ImportClause: 127.7, ImportRef: 13.1,
	}
	with := func(f func(*commitCheckBudget)) commitCheckBudget { b := in; f(&b); return b }
	for _, c := range []struct {
		name                string
		b                   commitCheckBudget
		want                budgetVerdict
		failures            []budgetVerdict
		singleOver, impOver bool
	}{
		// pass: under both thresholds, ratios in range, flat.
		{"pass", in, budgetPass, nil, false, false},
		{"thresholds, ratios and flatness just inside", with(func(b *commitCheckBudget) {
			b.PerCallClause = [3]float64{300, 250, 300} // max/min 1.2
			b.PerCallRef[0] = 60                        // 5 exactly
			b.ImportClause, b.ImportRef = 165, 11       // 15 exactly
		}), budgetPass, nil, false, false},

		// warn: over a threshold with in-range ratios, flat. The job passes.
		{"warn: both rows over, ratios in range (E3 rerun)", with(func(b *commitCheckBudget) {
			b.PerCallClause = [3]float64{235.7, 327.7, 330.5} // 1.40
			b.PerCallRef[0] = 161.9
			b.ImportClause, b.ImportRef = 194.9, 22.7 // 8.58
		}), budgetWarn, nil, true, true},
		{"warn: only the import row over (run 41679998550)", with(func(b *commitCheckBudget) {
			b.PerCallClause = [3]float64{235.7, 227.7, 230.5}
			b.PerCallRef[0] = 161.9
			b.ImportClause, b.ImportRef = 194.9, 22.7
		}), budgetWarn, nil, false, true},
		{"warn: only N = 5,000 over 300", with(func(b *commitCheckBudget) {
			b.PerCallClause = [3]float64{250, 260, 301} // 1.20
		}), budgetWarn, nil, true, false},

		// fail_ratio: a same-run ratio out of range, whatever the absolutes.
		{"fail_ratio: per-call ratio over 5, absolutes in range", with(func(b *commitCheckBudget) {
			b.PerCallRef[0] = 30 // 170.5 / 30 = 5.68
		}), budgetFailRatio, []budgetVerdict{budgetFailRatio}, false, false},
		{"fail_ratio: import ratio over 15, absolutes in range", with(func(b *commitCheckBudget) {
			b.ImportRef = 8 // 127.7 / 8 = 15.96
		}), budgetFailRatio, []budgetVerdict{budgetFailRatio}, false, false},
		{"fail_ratio: over a threshold too (the S3 shape)", with(func(b *commitCheckBudget) {
			b.ImportClause, b.ImportRef = 300, 15 // 20
		}), budgetFailRatio, []budgetVerdict{budgetFailRatio}, false, true},
		{"fail_ratio wins over fail_flat; both logged", with(func(b *commitCheckBudget) {
			b.PerCallClause = [3]float64{170, 200, 300} // 1.76
			b.ImportRef = 5
		}), budgetFailRatio, []budgetVerdict{budgetFailRatio, budgetFailFlat}, false, false},

		// fail_flat: clause per call max/min ≥ 1.5, ratios in range.
		{"fail_flat: not flat, ratios in range (E3 rerun)", with(func(b *commitCheckBudget) {
			b.PerCallClause = [3]float64{150, 200, 290} // 1.93
		}), budgetFailFlat, []budgetVerdict{budgetFailFlat}, false, false},
		{"fail_flat: max/min exactly 1.5", with(func(b *commitCheckBudget) {
			b.PerCallClause = [3]float64{150, 200, 225}
		}), budgetFailFlat, []budgetVerdict{budgetFailFlat}, false, false},
		{"fail_flat under the thresholds and over one alike", with(func(b *commitCheckBudget) {
			b.PerCallClause = [3]float64{170, 200, 400} // 2.35, N = 5,000 over 300
		}), budgetFailFlat, []budgetVerdict{budgetFailFlat}, true, false},

		// Ratio and flatness both exactly at the limit: ratios in range (≤),
		// flatness not (< 1.5 required).
		{"ratio and flatness both exactly at the limit", with(func(b *commitCheckBudget) {
			b.PerCallClause = [3]float64{200, 250, 300} // 1.5
			b.PerCallRef[0] = 40                        // 5
			b.ImportClause, b.ImportRef = 150, 10       // 15
		}), budgetFailFlat, []budgetVerdict{budgetFailFlat}, false, false},

		// fail_no_rows: a measured value ≤ 0, NaN or missing.
		{"fail_no_rows: zero import reference", with(func(b *commitCheckBudget) {
			b.ImportRef = 0
		}), budgetFailNoRows, []budgetVerdict{budgetFailNoRows}, false, false},
		{"fail_no_rows: zero per-call reference", with(func(b *commitCheckBudget) {
			b.PerCallRef[0] = 0
		}), budgetFailNoRows, []budgetVerdict{budgetFailNoRows}, false, false},
		{"fail_no_rows: NaN clause value", with(func(b *commitCheckBudget) {
			b.PerCallClause[1] = math.NaN()
		}), budgetFailNoRows, []budgetVerdict{budgetFailNoRows}, false, false},
		{"fail_no_rows: negative import", with(func(b *commitCheckBudget) {
			b.ImportClause = -1
		}), budgetFailNoRows, []budgetVerdict{budgetFailNoRows}, false, false},
		{"fail_no_rows: nothing measured", commitCheckBudget{}, budgetFailNoRows, []budgetVerdict{budgetFailNoRows}, false, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			line := c.b.singleLine(38, "test") + " | " + c.b.importLine(38, "test", nil, nil)
			if got := c.b.verdict(); got != c.want {
				t.Errorf("verdict %s, want %s: %s", got, c.want, line)
			}
			if got := c.b.verdict().failed(); got != (c.want < budgetWarn) {
				t.Errorf("failed() = %t for %s", got, c.want)
			}
			if got := c.b.failures(); fmt.Sprint(got) != fmt.Sprint(c.failures) {
				t.Errorf("failures %v, want %v", got, c.failures)
			}
			if c.want != budgetFailNoRows {
				if got := c.b.singleOver(); got != c.singleOver {
					t.Errorf("single over threshold %t, want %t", got, c.singleOver)
				}
				if got := c.b.importOver(); got != c.impOver {
					t.Errorf("import over threshold %t, want %t", got, c.impOver)
				}
			}
		})
	}
}

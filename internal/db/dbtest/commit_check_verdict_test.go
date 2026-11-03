package dbtest

// The verdict rule of the commit-check budget rows (ADR-004-E3 "Budget",
// PLAN-005-E3, TICK-509). It is untagged, so `task test` unit-tests it on
// fixed inputs; TestCommitCheckBudget (integration) applies it to the
// numbers measured in CI. Nothing here may move a ceiling: an over-ceiling
// result with in-range ratios is a runner problem (re-run), never a reason
// to edit these constants (D-E3-6).

import (
	"fmt"
	"math"
	"testing"
)

const (
	// budgetPerCallCeiling is the single-subject row: µs of self time per call.
	budgetPerCallCeiling = 300.0
	// budgetFlatLimit: max/min of the three per-call values must be below it.
	budgetFlatLimit = 1.5
	// budgetImportCeiling is the import row: µs of COMMIT time per probe.
	budgetImportCeiling = 175.0
	// budgetImportRatioMax and budgetPerCallRatioMax bound the same-run
	// ratios, clause / reference; above them the ratio is out of range.
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

type budgetVerdict int

const (
	budgetPass  budgetVerdict = iota // within the ceiling
	budgetRerun                      // over the ceiling, ratios in range: runner problem
	budgetS3                         // over the ceiling and a ratio out of range: E3-R2 S3
)

func (v budgetVerdict) String() string {
	switch v {
	case budgetPass:
		return "pass"
	case budgetRerun:
		return "rerun"
	case budgetS3:
		return "s3"
	}
	return fmt.Sprintf("budgetVerdict(%d)", int(v))
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

// singleWithin: every per-call value at most the ceiling, and flat.
func (b commitCheckBudget) singleWithin() bool {
	for _, v := range b.PerCallClause {
		if !(v <= budgetPerCallCeiling) {
			return false
		}
	}
	return b.flatness() < budgetFlatLimit
}

func (b commitCheckBudget) importWithin() bool { return b.ImportClause <= budgetImportCeiling }

// ratioOutOfRange: a same-run ratio is out of range (import > 15, or per
// call at N = 200 > 5). A NaN ratio (a zero or missing measurement) counts
// as out of range, so a broken reference cannot hide an S3.
func (b commitCheckBudget) ratioOutOfRange() bool {
	imp, pc := b.importRatio(), b.perCallRatio()
	return !(imp <= budgetImportRatioMax) || !(pc <= budgetPerCallRatioMax)
}

// rowVerdict applies the rule to one row: within its ceiling passes; over
// it fires S3 only if a same-run ratio is out of range, and is otherwise a
// re-run.
func (b commitCheckBudget) rowVerdict(within bool) budgetVerdict {
	switch {
	case within:
		return budgetPass
	case b.ratioOutOfRange():
		return budgetS3
	default:
		return budgetRerun
	}
}

func (b commitCheckBudget) singleVerdict() budgetVerdict { return b.rowVerdict(b.singleWithin()) }
func (b commitCheckBudget) importVerdict() budgetVerdict { return b.rowVerdict(b.importWithin()) }

// singleLine and importLine are the greppable log lines, one per row.
func (b commitCheckBudget) singleLine(partitions int) string {
	return fmt.Sprintf("COMMIT-CHECK-BUDGET row=single_subject partitions=%d "+
		"clause_us_n200=%.1f clause_us_n1000=%.1f clause_us_n5000=%.1f max_min=%.2f "+
		"ref_us_n200=%.1f ref_us_n1000=%.1f ref_us_n5000=%.1f ratio_n200=%.2f ratio_import=%.2f "+
		"ceiling_us=%.0f flat_below=%.1f verdict=%s",
		partitions, b.PerCallClause[0], b.PerCallClause[1], b.PerCallClause[2], b.flatness(),
		b.PerCallRef[0], b.PerCallRef[1], b.PerCallRef[2], b.perCallRatio(), b.importRatio(),
		budgetPerCallCeiling, budgetFlatLimit, b.singleVerdict())
}

func (b commitCheckBudget) importLine(partitions int, clauseRuns, refRuns []float64) string {
	return fmt.Sprintf("COMMIT-CHECK-BUDGET row=import partitions=%d subjects=10000 "+
		"clause_us_per_probe=%.1f clause_runs=%.1f ref_us_per_probe=%.1f ref_runs=%.1f "+
		"ratio_import=%.2f ratio_n200=%.2f ceiling_us=%.0f verdict=%s",
		partitions, b.ImportClause, clauseRuns, b.ImportRef, refRuns,
		b.importRatio(), b.perCallRatio(), budgetImportCeiling, b.importVerdict())
}

func TestCommitCheckBudgetVerdict(t *testing.T) {
	// in is a run inside both ceilings with in-range ratios (about the
	// TA's round-1 laptop numbers); each case changes what it names.
	in := commitCheckBudget{
		PerCallClause: [3]float64{204, 190, 249}, PerCallRef: [3]float64{50, 80, 280},
		ImportClause: 130, ImportRef: 11,
	}
	with := func(f func(*commitCheckBudget)) commitCheckBudget { b := in; f(&b); return b }
	for _, c := range []struct {
		name        string
		b           commitCheckBudget
		single, imp budgetVerdict
	}{
		{"within both ceilings", in, budgetPass, budgetPass},
		{"at the ceilings exactly", with(func(b *commitCheckBudget) {
			b.PerCallClause = [3]float64{300, 250, 300}
			b.ImportClause, b.ImportRef = 175, 175.0/15
		}), budgetPass, budgetPass},
		// Over a ceiling, each ratio out of range on its own: S3.
		{"single over, per-call ratio out of range", with(func(b *commitCheckBudget) {
			b.PerCallClause = [3]float64{420, 430, 440}
			b.PerCallRef[0] = 80 // 5.25
		}), budgetS3, budgetPass},
		{"single over, import ratio out of range", with(func(b *commitCheckBudget) {
			b.PerCallClause = [3]float64{420, 430, 440}
			b.ImportRef = 8 // 16.25; the import row is still within its ceiling
		}), budgetS3, budgetPass},
		{"import over, import ratio out of range", with(func(b *commitCheckBudget) {
			b.ImportClause, b.ImportRef = 300, 15 // 20
		}), budgetPass, budgetS3},
		{"import over, per-call ratio out of range", with(func(b *commitCheckBudget) {
			b.ImportClause, b.ImportRef = 300, 30 // 10, in range
			b.PerCallRef[0] = 30                  // 6.8
		}), budgetPass, budgetS3},
		{"single not flat, per-call ratio out of range", with(func(b *commitCheckBudget) {
			b.PerCallClause = [3]float64{150, 200, 290} // 1.93
			b.PerCallRef[0] = 25                        // 6
		}), budgetS3, budgetPass},
		// Over a ceiling, both ratios in range: a runner problem.
		{"both over, both ratios in range", with(func(b *commitCheckBudget) {
			b.PerCallClause = [3]float64{450, 500, 520}
			b.PerCallRef[0] = 100                 // 4.5
			b.ImportClause, b.ImportRef = 294, 24 // 12.25
		}), budgetRerun, budgetRerun},
		{"single not flat, ratios in range", with(func(b *commitCheckBudget) {
			b.PerCallClause = [3]float64{150, 200, 225} // exactly 1.5: not below it
		}), budgetRerun, budgetPass},
		{"ratios at the limits are in range", with(func(b *commitCheckBudget) {
			b.PerCallClause = [3]float64{400, 400, 400}
			b.PerCallRef[0] = 80                  // 5
			b.ImportClause, b.ImportRef = 300, 20 // 15
		}), budgetRerun, budgetRerun},
		{"a zero reference cannot hide S3", with(func(b *commitCheckBudget) {
			b.ImportClause, b.ImportRef = 300, 0
		}), budgetPass, budgetS3},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := c.b.singleVerdict(); got != c.single {
				t.Errorf("single-subject verdict %s, want %s: %s", got, c.single, c.b.singleLine(38))
			}
			if got := c.b.importVerdict(); got != c.imp {
				t.Errorf("import verdict %s, want %s: %s", got, c.imp, c.b.importLine(38, nil, nil))
			}
		})
	}
}

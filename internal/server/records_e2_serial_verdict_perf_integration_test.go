//go:build integration && perf

package server_test

import "testing"

// The (3e) verdict rule (TICK-1098), as the Technical Architect ruled on
// 2026-11-23: FAIL first (an arm-A-only failure, or a share miss on a cell
// that is itself conclusive), then INCONCLUSIVE, then PASS.
func TestE3Verdict(t *testing.T) {
	band := e3Inputs{name: "band", band: true, pA: 0, nA: 800, pB: 0, nB: 800, p95A: 100}
	a50 := e3Inputs{name: "a50", planned: true, pA: .10, nA: 800, pB: .10, nB: 800, gatherB: true, launchedB: true, peakB: 8}
	with := func(x e3Inputs, f func(*e3Inputs)) e3Inputs { f(&x); return x }
	noWorkers := func(x *e3Inputs) { x.launchedB, x.peakB = false, 0 }
	noGatherB := func(x *e3Inputs) { x.gatherB, x.launchedB = false, false }
	shareMiss := func(x *e3Inputs) { x.pA = .60 }
	void := func(x *e3Inputs) { x.hostBad = "gates" }
	errs := func(a, b int) func(*e3Inputs) { return func(x *e3Inputs) { x.errsA, x.errsB = a, b } }
	for _, c := range []struct {
		name      string
		in        []e3Inputs
		want      string
		noWorkers bool
	}{
		{"both pass", []e3Inputs{band, a50}, "PASS", false},
		{"share miss", []e3Inputs{band, with(a50, shareMiss)}, "FAIL", false},
		{"one point of slack", []e3Inputs{with(band, func(x *e3Inputs) { x.pA = .0099 }), a50}, "PASS", false},
		{"beyond one point", []e3Inputs{with(band, func(x *e3Inputs) { x.pA = .02 }), a50}, "FAIL", false},
		{"p95 over", []e3Inputs{with(band, func(x *e3Inputs) { x.p95A = 350.1 }), a50}, "FAIL", false},
		{"p95 tie passes", []e3Inputs{with(band, func(x *e3Inputs) { x.p95A = 350 }), a50}, "PASS", false},
		{"no workers", []e3Inputs{band, with(a50, noWorkers)}, "INCONCLUSIVE", true},
		{"no Gather B, no workers", []e3Inputs{band, with(with(a50, noWorkers), noGatherB)}, "INCONCLUSIVE", true},
		{"no Gather B, workers ran", []e3Inputs{band, with(a50, noGatherB)}, "INCONCLUSIVE", false},
		{"workers only in the peak", []e3Inputs{band, with(a50, func(x *e3Inputs) { x.launchedB = false })}, "PASS", false},
		{"share miss, no workers", []e3Inputs{band, with(with(a50, noWorkers), shareMiss)}, "INCONCLUSIVE", true},
		{"band share miss, a50 no workers", []e3Inputs{with(band, shareMiss), with(a50, noWorkers)}, "FAIL", false},
		{"band share miss, a50 void", []e3Inputs{with(band, shareMiss), with(a50, void)}, "FAIL", false},
		{"a50 share miss, band void", []e3Inputs{with(band, void), with(a50, shareMiss)}, "FAIL", false},
		{"p95 over, no workers", []e3Inputs{with(band, func(x *e3Inputs) { x.p95A = 400 }), with(a50, noWorkers)}, "FAIL", false},
		{"Gather A, no workers", []e3Inputs{band, with(with(a50, noWorkers), func(x *e3Inputs) { x.gatherA = true })}, "FAIL", false},
		{"Gather A, void cell", []e3Inputs{band, with(with(a50, void), func(x *e3Inputs) { x.gatherA = true })}, "FAIL", false},
		{"void band, p95 not read", []e3Inputs{with(with(band, void), func(x *e3Inputs) { x.p95A = 900 }), a50}, "INCONCLUSIVE", false},
		{"void band, a50 no workers", []e3Inputs{with(band, void), with(a50, noWorkers)}, "INCONCLUSIVE", false},
		{"void a50", []e3Inputs{band, with(a50, void)}, "INCONCLUSIVE", false},
		{"void a50 without workers", []e3Inputs{band, with(with(a50, void), noWorkers)}, "INCONCLUSIVE", false},
		{"no samples", []e3Inputs{with(band, func(x *e3Inputs) { x.nB = 0 }), a50}, "INCONCLUSIVE", false},
		{"4 errors in A only", []e3Inputs{with(band, errs(4, 0)), a50}, "INCONCLUSIVE", false},
		{"5 errors in A only", []e3Inputs{with(band, errs(5, 0)), a50}, "FAIL", false},
		{"5 errors in A only, a50 no workers", []e3Inputs{with(band, errs(5, 0)), with(a50, noWorkers)}, "FAIL", false},
		{"5 errors in A, void cell", []e3Inputs{with(with(band, void), errs(5, 0)), a50}, "INCONCLUSIVE", false},
		{"errors in both", []e3Inputs{with(band, errs(50, 1)), a50}, "INCONCLUSIVE", false},
		{"errors in B only", []e3Inputs{band, with(a50, errs(0, 9))}, "INCONCLUSIVE", false},
		{"an error, a50 no workers", []e3Inputs{with(band, errs(1, 0)), with(a50, noWorkers)}, "INCONCLUSIVE", false},
		{"no cells", nil, "INCONCLUSIVE", false},
	} {
		if got, nw, _ := e3Verdict(c.in); got != c.want || nw != c.noWorkers {
			t.Errorf("%s: %s %v, want %s %v", c.name, got, nw, c.want, c.noWorkers)
		}
	}
}

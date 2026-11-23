//go:build integration && perf

package server_test

import "testing"

// The (3e) verdict rule (TICK-1098): an arm-A-only failure beats everything,
// then INCONCLUSIVE (a void cell, or arm B's a50 cell launching no workers),
// then the unknown-share comparison.
func TestE3Verdict(t *testing.T) {
	band := e3Inputs{name: "band", band: true, pA: 0, nA: 800, pB: 0, nB: 800, p95A: 100}
	a50 := e3Inputs{name: "a50", planned: true, pA: .10, nA: 800, pB: .10, nB: 800, launchedB: true, peakB: 8}
	with := func(x e3Inputs, f func(*e3Inputs)) e3Inputs { f(&x); return x }
	noWorkers := func(x *e3Inputs) { x.launchedB, x.peakB = false, 0 }
	shareMiss := func(x *e3Inputs) { x.pA = .60 }
	void := func(x *e3Inputs) { x.hostBad = "gates" }
	for _, c := range []struct {
		name string
		in   []e3Inputs
		want string
	}{
		{"both pass", []e3Inputs{band, a50}, "PASS"},
		{"share miss", []e3Inputs{band, with(a50, shareMiss)}, "FAIL"},
		{"one point of slack", []e3Inputs{with(band, func(x *e3Inputs) { x.pA = .0099 }), a50}, "PASS"},
		{"beyond one point", []e3Inputs{with(band, func(x *e3Inputs) { x.pA = .02 }), a50}, "FAIL"},
		{"p95 over", []e3Inputs{with(band, func(x *e3Inputs) { x.p95A = 350.1 }), a50}, "FAIL"},
		{"p95 tie passes", []e3Inputs{with(band, func(x *e3Inputs) { x.p95A = 350 }), a50}, "PASS"},
		{"no workers", []e3Inputs{band, with(a50, noWorkers)}, "INCONCLUSIVE"},
		{"workers only in the peak", []e3Inputs{band, with(a50, func(x *e3Inputs) { x.launchedB = false })}, "PASS"},
		{"share miss, no workers", []e3Inputs{band, with(with(a50, noWorkers), shareMiss)}, "INCONCLUSIVE"},
		{"p95 over, no workers", []e3Inputs{with(band, func(x *e3Inputs) { x.p95A = 400 }), with(a50, noWorkers)}, "FAIL"},
		{"Gather A, no workers", []e3Inputs{band, with(with(a50, noWorkers), func(x *e3Inputs) { x.gatherA = true })}, "FAIL"},
		{"Gather A, void cell", []e3Inputs{band, with(with(a50, void), func(x *e3Inputs) { x.gatherA = true })}, "FAIL"},
		{"void band, p95 not read", []e3Inputs{with(with(band, void), func(x *e3Inputs) { x.p95A = 900 }), a50}, "INCONCLUSIVE"},
		{"void a50", []e3Inputs{band, with(a50, void)}, "INCONCLUSIVE"},
		{"no samples", []e3Inputs{with(band, func(x *e3Inputs) { x.nB = 0 }), a50}, "INCONCLUSIVE"},
		{"errors", []e3Inputs{with(band, func(x *e3Inputs) { x.errs = 1 }), a50}, "INCONCLUSIVE"},
		{"no cells", nil, "INCONCLUSIVE"},
	} {
		if got, _ := e3Verdict(c.in); got != c.want {
			t.Errorf("%s: %s, want %s", c.name, got, c.want)
		}
	}
}

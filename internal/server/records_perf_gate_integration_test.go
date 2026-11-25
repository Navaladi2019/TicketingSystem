//go:build integration && perf

// TICK-1040: the P3-36 harness voids a list or get attempt that shared its
// window with another container, as it does for creates. Runs without a
// database: go test -tags 'integration perf' -run TestPerfGate ./internal/server/

package server_test

import (
	"strings"
	"testing"
)

func TestPerfGateVoidsContendedRows(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name      string
		sc        perfScenario
		intruders []string
		probe     float64
		gated     bool
		label     string
	}{
		{"list, quiet", perfScenario{name: "agent list", kind: "list"}, nil, 0, false, "agent list, attempt 1"},
		{"list, contended", perfScenario{name: "agent list", kind: "list"}, []string{"nava-it-pg-7"}, 0, true, "agent list, attempt 1"},
		{"get, contended", perfScenario{name: "agent get", kind: "get"}, []string{"nava-aclperf-pg-575"}, 0, true, "agent get, attempt 1"},
		{"create, contended", perfScenario{name: "create", kind: "create"}, []string{"nava-it-pg-7"}, 0, true, "create, attempt 1"},
		{"create, slow host", perfScenario{name: "create", kind: "create"}, nil, perfHostGateMS + 1, true, "create, attempt 1"},
		{"unscored list, contended", perfScenario{name: "diag", kind: "list", unscored: "diagnostic"}, []string{"nava-it-pg-7"}, 0, false, "diag"},
		{"timeout bound check, contended", perfScenario{name: "bound", kind: "timeout"}, []string{"nava-it-pg-7"}, 0, false, "bound"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			r := &perfResult{sc: c.sc, intruders: c.intruders, probe: c.probe, attempt: 1}
			if got := r.gated(); got != c.gated {
				t.Fatalf("gated() = %v, want %v (reason %q)", got, c.gated, r.gateReason())
			}
			if c.gated && len(c.intruders) > 0 && !strings.Contains(r.gateReason(), c.intruders[0]) {
				t.Errorf("gateReason() = %q, want it to name %s", r.gateReason(), c.intruders[0])
			}
			if got := r.label(); got != c.label {
				t.Errorf("label() = %q, want %q", got, c.label)
			}
		})
	}
}

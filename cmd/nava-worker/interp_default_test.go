//go:build !interptest

package main

import "testing"

// TestProductionBuildRegistersNoInterpreter: with no real ports yet, the
// production worker registers nothing for the interpreter and stays
// activity-only.
func TestProductionBuildRegistersNoInterpreter(t *testing.T) {
	acts, err := interpreterActivities(func(string) (string, bool) { return "/fixtures", true })
	if err != nil || acts != nil {
		t.Fatalf("interpreterActivities = (%v, %v), want (nil, nil) in a production build", acts, err)
	}
	r := &recordingRegistry{}
	registerInterpreter(r, nil)
	if len(r.workflows)+len(r.activities) != 0 {
		t.Errorf("registered %v %v with no ports", r.workflows, r.activities)
	}
}

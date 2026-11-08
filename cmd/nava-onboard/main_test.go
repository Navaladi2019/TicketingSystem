package main

import (
	"io"
	"testing"
)

func TestRunRefusesMissingConfig(t *testing.T) {
	lookup := func(string) (string, bool) { return "", false }
	if err := run(t.Context(), lookup, io.Discard); err == nil {
		t.Fatal("run succeeded with no configuration")
	}
}

func TestParseTenantIDs(t *testing.T) {
	good := "00000000-0000-4000-8000-0000000000a1"
	if ids, err := parseTenantIDs([]string{good, good}); err != nil || len(ids) != 2 {
		t.Fatalf("good ids = %v, %v", ids, err)
	}
	for _, args := range [][]string{nil, {"not-a-uuid"}, {"00000000-0000-0000-0000-000000000000"}, {good, "x"}} {
		if _, err := parseTenantIDs(args); err == nil {
			t.Errorf("parseTenantIDs(%v) succeeded", args)
		}
	}
}

func TestReconcileAuditorsRefusesBadInput(t *testing.T) {
	lookup := func(string) (string, bool) { return "", false }
	if err := reconcileAuditors(t.Context(), lookup, io.Discard, nil); err == nil {
		t.Fatal("no tenant ids accepted")
	}
	if err := reconcileAuditors(t.Context(), lookup, io.Discard, []string{"00000000-0000-4000-8000-0000000000a1"}); err == nil {
		t.Fatal("reconcile ran with no configuration")
	}
}

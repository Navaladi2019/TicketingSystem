package main

import (
	"bytes"
	"io"
	"log/slog"
	"strings"
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

func TestWarnDevIdPHosts(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&buf, nil))
	warnDevIdPHosts(t.Context(), log, nil)
	if buf.Len() != 0 {
		t.Fatalf("logged with an empty list: %s", buf.String())
	}
	warnDevIdPHosts(t.Context(), log, []string{"localhost:8080", "keycloak:8080"})
	out := buf.String()
	if strings.Count(out, "\n") != 1 || !strings.Contains(out, `"level":"WARN"`) ||
		!strings.Contains(out, "localhost:8080") || !strings.Contains(out, "keycloak:8080") {
		t.Fatalf("want one WARN naming both hosts, got: %s", out)
	}
}

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/strictyaml"
)

// repoConfig is the committed docs/delivery/review-paths.yaml.
func repoConfig(t *testing.T) Config {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "docs", "delivery", "review-paths.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var cfg Config
	if err := strictyaml.Decode(data, &cfg); err != nil {
		t.Fatal(err)
	}
	return cfg
}

// The T05 acceptance fixture: a PR touching db/ is flagged for security review.
func TestFixturePRTouchingDBIsFlagged(t *testing.T) {
	changes := parseNameStatus("A\tdb/migrations/00002_tickets.sql\nM\tREADME.md\n")
	got := Evaluate(repoConfig(t), changes, nil).String()
	want := "security review required: yes\nCTO sign-off required: no\n  security: db/migrations/00002_tickets.sql\n"
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestOrdinaryChangeIsNotFlagged(t *testing.T) {
	changes := parseNameStatus("M\tinternal/server/api.go\nM\tdocs/pins.md\n")
	got := Evaluate(repoConfig(t), changes, nil).String()
	if got != "security review required: no\nCTO sign-off required: no\n" {
		t.Fatalf("got:\n%s", got)
	}
}

func TestSecurityPaths(t *testing.T) {
	cfg := repoConfig(t)
	for _, p := range []string{
		"temporal-server/authz/policy.go",
		"compose.yaml",
		"deploy/compose/compose.internal.yaml",
		".github/workflows/ci.yml",
		".golangci.yml",
		"internal/platform/telemetry/filter.go",
		"docs/delivery/review-paths.yaml",
	} {
		if !matchAny(cfg.Security, p) {
			t.Errorf("%s is not on the security list", p)
		}
	}
	for _, p := range []string{"databases.md", "internal/dbx/x.go", "deploy/compose/test/stack-test.sh"} {
		if matchAny(cfg.Security, p) {
			t.Errorf("%s matched the security list", p)
		}
	}
}

func TestCTOSignoff(t *testing.T) {
	cfg := repoConfig(t)
	const (
		iso  = "internal/tenancy/isolation/records_test.go"
		ci6  = "temporal-server/authz/adversarial_test.go"
		deep = "internal/a/b/c/tenant_isolation_test.go"
	)
	for _, tc := range []struct {
		name    string
		changes string
		added   map[string][]string
		want    string
	}{
		{"allow-list edited", "M\tdb/rls-allowlist.yaml", nil, "db/rls-allowlist.yaml (changed)"},
		{"isolation test removed", "D\t" + iso, nil, iso + " (isolation test removed)"},
		{"isolation test skipped", "M\t" + iso, map[string][]string{iso: {"\tt.Skip(\"flaky\")"}}, iso + " (isolation test skipped or excluded)"},
		{"short guard", "M\t" + iso, map[string][]string{iso: {"\tif testing.Short() {"}}, iso + " (isolation test skipped or excluded)"},
		{"isolation test extended", "M\t" + iso, map[string][]string{iso: {"\tassertNoRows(t, rows)"}}, ""},
		// PR #19 round 1: the CI-6 suite is registered.
		{"CI-6 suite removed", "D\t" + ci6, nil, ci6 + " (isolation test removed)"},
		{"CI-6 suite skipped", "M\t" + ci6, map[string][]string{ci6: {"\tt.Skipf(\"%s\", why)"}}, ci6 + " (isolation test skipped or excluded)"},
		// *isolation*_test.go at any depth, and at the root.
		{"deep isolation test removed", "D\t" + deep, nil, deep + " (isolation test removed)"},
		{"root isolation test removed", "D\tisolation_test.go", nil, "isolation_test.go (isolation test removed)"},
		// A build constraint can drop the file from the run as surely as a skip.
		{"go:build ignore", "M\t" + deep, map[string][]string{deep: {"//go:build ignore"}}, deep + " (isolation test skipped or excluded)"},
		{"+build tag", "M\t" + iso, map[string][]string{iso: {"// +build never"}}, iso + " (isolation test skipped or excluded)"},
		{"constraint changed", "M\t" + ci6, map[string][]string{ci6: {"//go:build integration && !race"}}, ci6 + " (isolation test skipped or excluded)"},
		// PR #19 round 2: a new isolation test needs its build tag; that is no skip.
		{"new isolation test", "A\t" + deep, map[string][]string{deep: {"//go:build integration", "package c"}}, ""},
		{"new isolation test skipped", "A\t" + deep, map[string][]string{deep: {"//go:build integration", "\tt.Skip(\"todo\")"}}, deep + " (isolation test skipped or excluded)"},
		// Not an isolation test: no CTO line.
		{"ordinary test removed", "D\tinternal/server/api_test.go", nil, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := Evaluate(cfg, parseNameStatus(tc.changes), tc.added)
			got := strings.Join(r.CTO, ",")
			if got != tc.want {
				t.Fatalf("CTO = %q, want %q", got, tc.want)
			}
		})
	}
}

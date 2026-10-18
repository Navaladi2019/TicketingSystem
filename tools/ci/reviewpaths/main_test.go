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
	const iso = "internal/tenancy/isolation/records_test.go"
	for _, tc := range []struct {
		name    string
		changes string
		added   map[string][]string
		want    string
	}{
		{"allow-list edited", "M\tdb/rls-allowlist.yaml", nil, "db/rls-allowlist.yaml (changed)"},
		{"isolation test removed", "D\t" + iso, nil, iso + " (isolation test removed)"},
		{"isolation test skipped", "M\t" + iso, map[string][]string{iso: {"\tt.Skip(\"flaky\")"}}, iso + " (isolation test skipped)"},
		{"short guard", "M\t" + iso, map[string][]string{iso: {"\tif testing.Short() {"}}, iso + " (isolation test skipped)"},
		{"isolation test extended", "M\t" + iso, map[string][]string{iso: {"\tassertNoRows(t, rows)"}}, ""},
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

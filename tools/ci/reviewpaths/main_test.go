package main

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
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
		"internal/identitystack/stack_test.go",
		"internal/platform/auth/middleware.go",
		"cmd/nava-relay/main.go",
		// Planned (PLAN-001 T21): flagged from the PR that creates it.
		"internal/onboard/onboard.go",
	} {
		if !matchAny(cfg.Security.Patterns(), p) {
			t.Errorf("%s is not on the security list", p)
		}
	}
	// Still holds once the repo has no planned entry left.
	planned := Entries{{Path: "internal/next/", Planned: "PLAN-001 T99"}}
	if !matchAny(planned.Patterns(), "internal/next/x.go") {
		t.Error("a planned entry is not matched")
	}
	for _, p := range []string{"databases.md", "internal/dbx/x.go", "deploy/compose/test/stack-test.sh"} {
		if matchAny(cfg.Security.Patterns(), p) {
			t.Errorf("%s matched the security list", p)
		}
	}
}

func TestCTOSignoff(t *testing.T) {
	cfg := repoConfig(t)
	const (
		iso  = "internal/tenancy/isolation/records_test.go"
		ci6  = "temporal-server/authz/adversarial_test.go"
		aud  = "internal/identitystack/audience_attack_test.go"
		dir  = "internal/identitystack/directory_adversarial_test.go"
		deep = "internal/a/b/c/tenant_isolation_test.go"
		cur  = "internal/platform/cursor/cursor_adversarial_test.go"
		t27k = "temporal-server/authz/adversarial_t27k_test.go"
		sa   = "internal/workerguard/adversarial_sa_keytype_test.go"
	)
	for _, tc := range []struct {
		name    string
		changes string
		added   map[string][]string
		want    string
	}{
		{"allow-list edited", "M\tinternal/db/catalog/policy.go", nil, "internal/db/catalog/policy.go (changed)"},
		{"isolation registry edited", "M\tinternal/tenancy/isolation/registry.go", nil, "internal/tenancy/isolation/registry.go (changed)"},
		{"isolation test removed", "D\t" + iso, nil, iso + " (isolation test removed)"},
		{"isolation test skipped", "M\t" + iso, map[string][]string{iso: {"\tt.Skip(\"flaky\")"}}, iso + " (isolation test skipped or excluded)"},
		{"short guard", "M\t" + iso, map[string][]string{iso: {"\tif testing.Short() {"}}, iso + " (isolation test skipped or excluded)"},
		{"isolation test extended", "M\t" + iso, map[string][]string{iso: {"\tassertNoRows(t, rows)"}}, ""},
		// PR #19 round 1: the CI-6 suite is registered.
		{"CI-6 suite removed", "D\t" + ci6, nil, ci6 + " (isolation test removed)"},
		{"CI-6 suite skipped", "M\t" + ci6, map[string][]string{ci6: {"\tt.Skipf(\"%s\", why)"}}, ci6 + " (isolation test skipped or excluded)"},
		// TICK-209: the identitystack adversarial suites are registered; the
		// rest of the package is not.
		{"audience suite skipped", "M\t" + aud, map[string][]string{aud: {"\tt.Skip(\"keycloak\")"}}, aud + " (isolation test skipped or excluded)"},
		{"directory suite removed", "D\t" + dir, nil, dir + " (isolation test removed)"},
		{"realm test removed", "D\tinternal/identitystack/realm_test.go", nil, ""},
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
		// TICK-229: every *adversarial*_test.go suite is registered, by the
		// **/*adversarial*_test.go glob, as suffix, prefix or whole name.
		{"cursor suite removed", "D\t" + cur, nil, cur + " (isolation test removed)"},
		{"T27k suite skipped", "M\t" + t27k, map[string][]string{t27k: {"\tt.Skip(\"flaky\")"}}, t27k + " (isolation test skipped or excluded)"},
		{"adversarial_ prefix suite removed", "D\t" + sa, nil, sa + " (isolation test removed)"},
		{"adversarial suite extended", "M\t" + cur, map[string][]string{cur: {"\tassertRejected(t, err)"}}, ""},
		// Not a test file, or "adversarial" only in a directory name.
		{"adversarial helper removed", "D\tinternal/platform/cursor/adversarial_helpers.go", nil, ""},
		{"adversarial directory test removed", "D\tinternal/adversarial/x_test.go", nil, ""},
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

// TICK-183: the realm export lives under deploy/compose/keycloak/ (since
// TICK-57); a realm change must be flagged. TICK-181: so must the external
// IdP test realms under test-realms/ (since PR #90).
func TestFixturePRTouchingRealmIsFlagged(t *testing.T) {
	for _, f := range []string{
		"deploy/compose/keycloak/import/realm-nava.json",
		"deploy/compose/keycloak/test-realms/realm-idp-tenant-a.json",
	} {
		changes := parseNameStatus("M\t" + f + "\n")
		got := Evaluate(repoConfig(t), changes, nil).String()
		want := "security review required: yes\nCTO sign-off required: no\n  security: " + f + "\n"
		if got != want {
			t.Errorf("%s: got:\n%s\nwant:\n%s", f, got, want)
		}
	}
}

// TICK-209: the realm tests under internal/identitystack/ need Security
// review, and deleting a registered adversarial suite also needs CTO sign-off.
func TestFixturePRTouchingIdentitystackIsFlagged(t *testing.T) {
	for _, tc := range []struct {
		name, changes, want string
	}{
		{
			"realm test changed",
			"M\tinternal/identitystack/realm_test.go\n",
			"security review required: yes\nCTO sign-off required: no\n" +
				"  security: internal/identitystack/realm_test.go\n",
		},
		{
			"adversarial suite deleted",
			"D\tinternal/identitystack/adversarial_test.go\n",
			"security review required: yes\nCTO sign-off required: yes\n" +
				"  security: internal/identitystack/adversarial_test.go\n" +
				"  CTO: internal/identitystack/adversarial_test.go (isolation test removed)\n",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := Evaluate(repoConfig(t), parseNameStatus(tc.changes), nil).String()
			if got != tc.want {
				t.Fatalf("got:\n%s\nwant:\n%s", got, tc.want)
			}
		})
	}
}

// TICK-229 acceptance fixture: deleting the cursor suite needs Security review
// and CTO sign-off.
func TestFixturePRDeletingCursorSuiteIsFlagged(t *testing.T) {
	const f = "internal/platform/cursor/cursor_adversarial_test.go"
	got := Evaluate(repoConfig(t), parseNameStatus("D\t"+f+"\n"), nil).String()
	want := "security review required: yes\nCTO sign-off required: yes\n" +
		"  security: " + f + "\n" +
		"  CTO: " + f + " (isolation test removed)\n"
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

// TICK-229: an adversarial suite missing from isolation_tests can be deleted
// or skipped with Security review alone. Every tracked *adversarial*_test.go
// file must be a registered isolation test.
func TestAdversarialSuitesAreRegistered(t *testing.T) {
	cmd := exec.Command("git", "ls-files", "--full-name") //nolint:noctx // runs to completion
	cmd.Dir = filepath.Join("..", "..", "..")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git ls-files (this check needs the git checkout): %v", err)
	}
	cfg := repoConfig(t)
	n := 0
	for _, f := range strings.Fields(string(out)) {
		base := path.Base(f)
		if !strings.HasSuffix(base, "_test.go") || !strings.Contains(base, "adversarial") || strings.Contains(f, "/testdata/") {
			continue
		}
		n++
		if !cfg.IsIsolationTest(f) {
			t.Errorf("%s is not in cto_signoff.isolation_tests: deleting or skipping it would need no CTO sign-off", f)
		}
	}
	if n == 0 {
		t.Fatal("no tracked adversarial test file found")
	}
}

// plannedRe is the marker a pattern with no tracked file yet must carry: the
// PLAN-001 task that creates it.
var plannedRe = regexp.MustCompile(`^PLAN-001 T\d{2}[a-z]?$`)

// staleEntries checks a path list against the tracked files, with the gate's
// own matching: a pattern no tracked file matches must carry a planned marker
// naming its task, and a planned pattern that now matches must drop it.
func staleEntries(list string, es Entries, tracked []string) []string {
	var bad []string
	for _, e := range es {
		live := false
		for _, f := range tracked {
			if matchAny([]string{e.Path}, f) {
				live = true
				break
			}
		}
		switch {
		case e.Planned != "" && !plannedRe.MatchString(e.Planned):
			bad = append(bad, fmt.Sprintf("%s: %s: planned %q does not name a task (want \"PLAN-001 T<nn>\")", list, e.Path, e.Planned))
		case live && e.Planned != "":
			bad = append(bad, fmt.Sprintf("%s: %s: exists now, drop planned %q", list, e.Path, e.Planned))
		case !live && e.Planned == "":
			bad = append(bad, fmt.Sprintf("%s: %s: matches no tracked file (fix the path, or mark it planned: \"PLAN-001 T<nn>\")", list, e.Path))
		}
	}
	return bad
}

func TestStaleEntries(t *testing.T) {
	tracked := []string{"compose.yaml", "deploy/compose/keycloak/import/realm-nava.json", "internal/db/catalog/policy.go"}
	for _, tc := range []struct {
		name string
		e    Entry
		ok   bool
	}{
		{"file", Entry{Path: "compose.yaml"}, true},
		{"directory", Entry{Path: "deploy/compose/keycloak/"}, true},
		{"glob", Entry{Path: "internal/db/catalog/*.go"}, true},
		{"any-depth glob", Entry{Path: "**/realm-*.json"}, true},
		{"missing file", Entry{Path: "db/rls-allowlist.yaml"}, false},
		{"missing directory", Entry{Path: "deploy/keycloak/"}, false},
		{"directory prefix is not a file", Entry{Path: "deploy/compose/keycloak"}, false},
		{"glob matching nothing", Entry{Path: "db/*.yaml"}, false},
		{"planned", Entry{Path: "cmd/nava-relay/", Planned: "PLAN-001 T30b"}, true},
		{"planned without task", Entry{Path: "cmd/nava-relay/", Planned: "later"}, false},
		{"planned without plan", Entry{Path: "cmd/nava-relay/", Planned: "T30b"}, false},
		{"planned but exists", Entry{Path: "compose.yaml", Planned: "PLAN-001 T02"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bad := staleEntries("security", Entries{tc.e}, tracked)
			if (len(bad) == 0) != tc.ok {
				t.Fatalf("ok = %v, want %v: %v", len(bad) == 0, tc.ok, bad)
			}
		})
	}
}

// TICK-183: a pattern that matches nothing flags nothing, silently. Every
// security and cto_signoff.paths entry must match a tracked file or name the
// PLAN-001 task that creates it.
func TestPathsMatchTrackedFiles(t *testing.T) {
	cmd := exec.Command("git", "ls-files", "--full-name") //nolint:noctx // runs to completion
	cmd.Dir = filepath.Join("..", "..", "..")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git ls-files (this check needs the git checkout): %v", err)
	}
	tracked := strings.Fields(string(out))
	cfg := repoConfig(t)
	bad := staleEntries("security", cfg.Security, tracked)
	bad = append(bad, staleEntries("cto_signoff.paths", cfg.CTOSignoff.Paths, tracked)...)
	for _, b := range bad {
		t.Error(b)
	}
}

func TestEntryForms(t *testing.T) {
	var cfg Config
	err := strictyaml.Decode([]byte("security:\n  - db/\n  - path: cmd/nava-relay/\n    planned: \"PLAN-001 T30b\"\n"), &cfg)
	if err != nil {
		t.Fatal(err)
	}
	want := Entries{{Path: "db/"}, {Path: "cmd/nava-relay/", Planned: "PLAN-001 T30b"}}
	if len(cfg.Security) != 2 || cfg.Security[0] != want[0] || cfg.Security[1] != want[1] {
		t.Fatalf("got %+v", cfg.Security)
	}
	for _, doc := range []string{
		"security:\n  - path: db/\n    plan: \"PLAN-001 T09\"\n", // misspelt key
		"security:\n  - planned: \"PLAN-001 T09\"\n",             // no path
		"security:\n  - [db/]\n",                                 // neither form
		"security:\n  - path: db/\n    path: deploy/\n",          // repeated key
	} {
		if err := strictyaml.Decode([]byte(doc), &Config{}); err == nil {
			t.Errorf("decoded %q", doc)
		}
	}
}

// TICK-230 bypass guard: the CI scripts decide which tags, packages and -run
// filters the isolation suites run with, and suiteran trusts their -tags. An
// edit there that drops both -tags (or the package, or adds -run) leaves the
// suite out of the run with CI green, so it must get the Security review.
func TestCIScriptsAreSecurityPaths(t *testing.T) {
	cfg := repoConfig(t)
	for _, p := range []string{
		"scripts/ci/isolation.sh",
		"scripts/ci/test-integration.sh",
	} {
		if !matchAny(cfg.Security.Patterns(), p) {
			t.Errorf("%s is not on the security list: an edit to it can drop an isolation suite from CI without Security review", p)
		}
	}
}

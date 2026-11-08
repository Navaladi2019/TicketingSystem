package main

import (
	"strings"
	"testing"
)

func areaGraph() graph {
	g := testGraph()
	for _, p := range []string{"internal/identity/keycloak", "internal/db/pooltest", "internal/records", "cmd/nava-db"} {
		g.packages[p] = true
		g.stack[p] = true
	}
	return g
}

// TICK-563: which stack-case areas a change turns on. A file no rule knows
// turns on every area; a stack package outside the areas turns on none.
func TestClassifyAreas(t *testing.T) {
	for _, c := range []struct {
		name  string
		files []string
		want  string
	}{
		{"nothing changed", nil, "none"},
		{"docs only", []string{"docs/quality/ci-pipeline.md"}, "none"},
		{"non-identity package", []string{"internal/records/store.go", "internal/server/server.go"}, "none"},
		{"identity adapter", []string{"internal/identity/keycloak/client.go"}, "keycloak"},
		{"realm export", []string{"deploy/compose/keycloak/realm.json"}, "keycloak"},
		{"keycloak case", []string{"deploy/compose/test/cases/dependencies.sh"}, "keycloak"},
		{"pgbouncer config", []string{"deploy/compose/pgbouncer/pgbouncer.ini"}, "pgbouncer"},
		{"pool code", []string{"internal/db/pool.go"}, "pgbouncer"},
		{"pool tests", []string{"internal/db/pooltest/pool_test.go"}, "pgbouncer"},
		{"migration", []string{"db/migrations/00009_x.sql"}, "db"},
		{"db roles", []string{"internal/db/roles/roles.go"}, "db"},
		{"nava-db command", []string{"cmd/nava-db/main.go"}, "db"},
		{"two areas", []string{"db/migrations/00009_x.sql", "internal/identity/keycloak/client.go"}, "keycloak,db"},
		{"case of no area", []string{"deploy/compose/test/cases/identity_attacks.sh"}, "none"},
		{"temporal-server", []string{"temporal-server/authz/authz.go"}, "none"},
		{"lint-only file", []string{".golangci.yml"}, "none"},
		{"unknown file", []string{"newdir/thing.yaml"}, "all"},
		{"unknown file beside a package", []string{"internal/records/store.go", "newdir/thing.yaml"}, "all"},
		{"compose file", []string{"compose.yaml"}, "all"},
		{"other deploy file", []string{"deploy/compose/compose.test.yaml"}, "all"},
		{"test runner", []string{"deploy/compose/test/run-cases.sh"}, "all"},
		{"root dependencies", []string{"go.sum"}, "all"},
		{"Makefile", []string{"Makefile"}, "all"},
		{"pins", []string{"docs/pins.md"}, "all"},
	} {
		t.Run(c.name, func(t *testing.T) {
			got, reasons := classifyAreas(c.files, areaGraph())
			if got != c.want {
				t.Fatalf("areas = %q, want %q\n%s", got, c.want, strings.Join(reasons, "\n"))
			}
			if len(reasons) != len(c.files) {
				t.Fatalf("%d reasons for %d files", len(reasons), len(c.files))
			}
		})
	}
}

// Every area a rule names is one run-cases.sh accepts.
func TestAreaRulesNameKnownAreas(t *testing.T) {
	for _, r := range areaRules() {
		found := false
		for _, a := range allAreas() {
			found = found || a == r.area
		}
		if !found {
			t.Errorf("rule %s: unknown area %q", r.prefix, r.area)
		}
	}
}

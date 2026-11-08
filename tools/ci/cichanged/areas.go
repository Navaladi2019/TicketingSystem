package main

import (
	"fmt"
	"path"
	"slices"
	"strings"
)

// allAreas are the stack-case areas (the area= of a case header in
// deploy/compose/test/cases, TICK-563), in output order.
func allAreas() []string { return []string{"keycloak", "pgbouncer", "db"} }

// areaRule maps a path prefix (a directory ending in /, or one file) to the
// area it touches.
type areaRule struct{ prefix, area string }

// areaRules: the services' own configuration, the Go code that talks to
// them, and the cases that test them. First match wins, so the db packages
// under internal/db come before internal/db itself.
func areaRules() []areaRule {
	return []areaRule{
		// Keycloak: its realm and image, the identity port and adapter, the
		// identity stack suite, group sync (reads the directory).
		{"deploy/compose/keycloak/", "keycloak"},
		{"internal/identity/", "keycloak"},
		{"internal/identitystack/", "keycloak"},
		{"internal/groupsync/", "keycloak"},
		{"cmd/nava-groupsync/", "keycloak"},
		{"deploy/compose/nava-groupsync/", "keycloak"},
		{"deploy/compose/test/cases/keycloak_restart.sh", "keycloak"},
		{"deploy/compose/test/cases/dependencies.sh", "keycloak"},
		{"deploy/compose/test/cases/identity.sh", "keycloak"},
		{"make/identity.mk", "keycloak"},
		// db: schema, migrations, roles and the tools that apply them.
		{"db/", "db"},
		{"cmd/nava-db/", "db"},
		{"deploy/compose/nava-db/", "db"},
		{"internal/db/roles/", "db"},
		{"internal/db/migrate/", "db"},
		{"internal/db/grants/", "db"},
		{"internal/db/hba/", "db"},
		{"internal/db/catalog/", "db"},
		{"internal/db/scram/", "db"},
		{"deploy/compose/test/cases/idempotent.sh", "db"},
		// PgBouncer: its configuration and the pool code in front of it.
		{"deploy/compose/pgbouncer/", "pgbouncer"},
		{"internal/db/pooltest/", "pgbouncer"},
		{"internal/db/", "pgbouncer"},
		{"deploy/compose/test/cases/pgbouncer.sh", "pgbouncer"},
	}
}

// classifyAreas returns the areas files touch, as all, none or a comma
// list in allAreas order, and one reason per file.
func classifyAreas(files []string, g graph) (string, []string) {
	on := map[string]bool{}
	var reasons []string
	for _, f := range files {
		a, why := fileAreas(f, g)
		for _, x := range a {
			on[x] = true
		}
		reasons = append(reasons, fmt.Sprintf("%s: areas %s (%s)", f, areaList(a), why))
	}
	var out []string
	for _, a := range allAreas() {
		if on[a] {
			out = append(out, a)
		}
	}
	return areaList(out), reasons
}

// areaList is all when every area is on, none when none is.
func areaList(a []string) string {
	switch {
	case len(a) == 0:
		return "none"
	case len(a) == len(allAreas()):
		return "all"
	}
	return strings.Join(a, ",")
}

func fileAreas(f string, g graph) ([]string, string) {
	for _, r := range areaRules() {
		if f == r.prefix || (strings.HasSuffix(r.prefix, "/") && strings.HasPrefix(f, r.prefix)) {
			return []string{r.area}, "under " + r.prefix
		}
	}
	stages, _ := fileStages(f, g)
	switch {
	case !slices.Contains(stages, "ci-stack"):
		return nil, "the stack does not depend on it"
	case strings.HasPrefix(f, "temporal-server/"):
		return nil, "temporal-server: no area tests it"
	case strings.HasPrefix(f, "deploy/compose/test/cases/") && path.Ext(f) == ".sh":
		// A case of no area always runs; the area cases are listed above.
		return nil, "a stack case of no area"
	case !strings.HasPrefix(f, "deploy/") && f != ".dockerignore" && !strings.HasPrefix(path.Base(f), "compose"):
		if pkg, ok := packageOf(f, g.packages); ok {
			return nil, "package " + pkg + " is in no area"
		}
	}
	// go.mod, the Makefile, the pins, compose files, the test runner, a file
	// no rule knows: every area.
	return allAreas(), "no area rule for this file, so every area"
}

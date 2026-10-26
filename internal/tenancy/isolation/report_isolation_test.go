//go:build integration

package isolation

import (
	"fmt"
	"slices"
	"strings"
)

// report is what make isolation writes as the CI artifact.
type report struct {
	ci2      []ci2Line
	paths    []pathResult
	fixtures []fixtureResult
}

// markdown groups everything by ADR test id: CI-2's generic lines (one per
// role × table) and fixtures, then each id's registered paths, with its
// coverage note, so a partial id shows as partial.
func (r report) markdown(ok bool) string {
	var b strings.Builder
	result := "PASS"
	if !ok {
		result = "FAIL"
	}
	fmt.Fprintf(&b, "# Tenant isolation suite (T11)\n\nResult: **%s**. Tenant A `%s`, tenant B `%s`.\n\n", result, TenantA, TenantB)
	b.WriteString("Statuses: PASS (0 rows or an error), FAIL (a row was reached), " +
		"ALLOWLISTED (reached by an ADR-001 A.3 policy that does not test the tenant, by design), PENDING (registered, not runnable yet).\n")

	cov := Coverage()
	ids := make([]string, 0, len(cov))
	for id := range cov {
		ids = append(ids, id)
	}
	slices.SortFunc(ids, func(a, b string) int { return idNum(a) - idNum(b) })
	for _, id := range ids {
		fmt.Fprintf(&b, "\n## %s\n\nCoverage: %s.\n", id, cov[id])
		if id == "CI-2" {
			r.writeCI2(&b)
		}
		var rows []pathResult
		for _, p := range r.paths {
			if p.Path.Case == id {
				rows = append(rows, p)
			}
		}
		if len(rows) == 0 {
			b.WriteString("\nNo registered path.\n")
			continue
		}
		b.WriteString("\n### Registered paths\n\n| Path | Role | Status | Detail |\n|---|---|---|---|\n")
		for _, p := range rows {
			role := p.Path.Role
			if role == "" {
				role = "-"
			}
			fmt.Fprintf(&b, "| %s | %s | %s | %s |\n", cellText(p.Path.Name), role, p.Status, cellText(p.Detail))
		}
		if id == "CI-9" {
			r.writeFixtures(&b, id)
		}
	}
	return b.String()
}

func (r report) writeCI2(b *strings.Builder) {
	b.WriteString("\n### Every runtime role × every tenant table\n\n" +
		"Each granted SELECT, UPDATE and DELETE, run with no tenant, `''` and after a prior SET LOCAL of tenant A " +
		"(directly and through PgBouncer), then with tenant A set through tenancy.InTenantTx, aimed at other tenants' rows.\n\n" +
		"| Role | Table | Status | Detail |\n|---|---|---|---|\n")
	if len(r.ci2) == 0 {
		b.WriteString("| - | - | FAIL | the generic checks did not run |\n")
	}
	for _, l := range r.ci2 {
		fmt.Fprintf(b, "| %s | %s | %s | %s |\n", l.Role, l.Table, l.Status, cellText(l.Detail))
	}
	r.writeFixtures(b, "CI-2")
}

// writeFixtures lists the fixtures of ADR test id id.
func (r report) writeFixtures(b *strings.Builder, id string) {
	b.WriteString("\n### Fixtures (the checks can fail)\n\n| Fixture | Role | Table | Before | With the fixture |\n|---|---|---|---|---|\n")
	n := 0
	for _, f := range r.fixtures {
		if f.Case == id {
			n++
			fmt.Fprintf(b, "| %s | %s | %s | %s | %s |\n", cellText(f.Name), f.Role, f.Table, f.Before, f.After)
		}
	}
	if n == 0 {
		b.WriteString("| - | - | - | - | did not run |\n")
	}
}

// idNum orders ADR test ids by number; a task id such as T26 comes after them.
func idNum(id string) int {
	var n int
	if _, err := fmt.Sscanf(id, "CI-%d", &n); err != nil {
		_, _ = fmt.Sscanf(id, "T%d", &n)
		return 1000 + n
	}
	return n
}

func cellText(s string) string {
	return strings.NewReplacer("|", `\|`, "\n", " ").Replace(s)
}

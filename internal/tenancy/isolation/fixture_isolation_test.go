//go:build integration

package isolation

import (
	"context"
	"testing"
)

// fixtureResult is one fixture in the report.
type fixtureResult struct {
	Name, Role, Table, Before, After string
}

// TestFixturesMakeTheSuiteFail is T11's second acceptance criterion and its
// siblings: each fixture breaks one isolation control on one table, and the
// CI-2 checks must turn that role × table from PASS to FAIL. A suite that
// stayed green here would be green for the wrong reason.
//
//   - FORCE dropped: the owner is the role FORCE holds to the policies, so
//     the owner × table line must fail. Runtime roles are not owners and
//     stay under RLS.
//   - RLS disabled: every role with a grant sees every tenant.
//   - a fail-open policy: one that matches when no tenant is set.
//   - a session-wide SET through PgBouncer: the tenant outlives its
//     transaction on the pooled server connection, the bleed the "prior"
//     case exists to catch.
func TestFixturesMakeTheSuiteFail(t *testing.T) {
	owner := suite.m.Owner()
	for _, f := range []struct {
		name, role, table string
		apply, undo       string
		sessionWide       bool
	}{
		{
			name: "FORCE ROW LEVEL SECURITY removed", role: owner, table: "public.records",
			apply: "ALTER TABLE public.records NO FORCE ROW LEVEL SECURITY",
			undo:  "ALTER TABLE public.records FORCE ROW LEVEL SECURITY",
		},
		{
			name: "ROW LEVEL SECURITY disabled", role: "nava_app", table: "public.outbox",
			apply: "ALTER TABLE public.outbox DISABLE ROW LEVEL SECURITY",
			undo:  "ALTER TABLE public.outbox ENABLE ROW LEVEL SECURITY",
		},
		{
			name: "fail-open policy (matches when no tenant is set)", role: "nava_worker", table: "public.records",
			apply: "CREATE POLICY isolation_fixture_fail_open ON public.records FOR SELECT TO nava_worker " +
				"USING (NULLIF(current_setting('app.tenant_id', true), '') IS NULL)",
			undo: "DROP POLICY IF EXISTS isolation_fixture_fail_open ON public.records",
		},
		{
			name: "session-wide SET through PgBouncer", role: "nava_app", table: "public.records",
			sessionWide: true,
		},
	} {
		t.Run(f.name, func(t *testing.T) {
			opts := ci2Opts{roles: []string{f.role}, tables: []string{f.table}}
			before := one(t, runCI2(t, opts))
			if before.Status != statusPass {
				t.Fatalf("before the fixture %s × %s is %s, want PASS: %s", f.role, f.table, before.Status, before.Detail)
			}
			super := suite.superConn(t)
			if f.apply != "" {
				// Undone even if the test fails, so later tests see the
				// real schema.
				t.Cleanup(func() {
					if _, err := super.Exec(context.Background(), f.undo); err != nil {
						t.Errorf("undo fixture: %v", err)
					}
				})
				if _, err := super.Exec(t.Context(), f.apply); err != nil {
					t.Fatal(err)
				}
			}
			opts.sessionWidePrior = f.sessionWide
			after := one(t, runCI2(t, opts))
			if after.Status != statusFail {
				t.Errorf("with the fixture %s × %s is %s, want FAIL: %s", f.role, f.table, after.Status, after.Detail)
			}
			suite.report.fixtures = append(suite.report.fixtures, fixtureResult{
				Name: f.name, Role: f.role, Table: f.table, Before: before.Status, After: after.Status,
			})
		})
	}
}

func one(t *testing.T, lines []ci2Line) ci2Line {
	t.Helper()
	if len(lines) != 1 {
		t.Fatalf("%d lines, want 1", len(lines))
	}
	return lines[0]
}

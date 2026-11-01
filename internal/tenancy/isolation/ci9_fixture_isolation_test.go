//go:build integration

package isolation

import (
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

// TestCI9FixturesMakePathsFail: each fixture widens one role's reach the
// way a bad grant or migration would, and the CI-9 path that guards it must
// turn from PASS to FAIL. The undo restores the policy as
// 00002_platform_tables.sql writes it.
func TestCI9FixturesMakePathsFail(t *testing.T) {
	const tenantSetting = "NULLIF(current_setting('app.tenant_id', true), '')::uuid"
	activate := "ALTER POLICY tenants_activate ON tenants USING (id = " + tenantSetting + " AND state = 'provisioning') " +
		"WITH CHECK (id = " + tenantSetting + " AND state = 'active')"
	onboardSeed := func(table string) string {
		return "ALTER POLICY " + table + "_onboard ON " + table + " WITH CHECK (tenant_id = " + tenantSetting +
			" AND tenant_id IN (SELECT id FROM tenants WHERE state = 'provisioning'))"
	}
	paths := map[string]Path{}
	for _, p := range rolePaths() {
		paths[p.Name] = p
	}
	for _, f := range []struct {
		name, path, table string
		apply, undo       string
	}{
		{
			name: "nava_relay granted a column of records", path: "nava_relay cannot read records", table: "public.records",
			apply: "GRANT SELECT (tenant_id) ON records TO nava_relay",
			undo:  "REVOKE SELECT (tenant_id) ON records FROM nava_relay",
		},
		{
			name: "nava_purge granted a data column", path: purgePath, table: "public.idempotency_keys",
			apply: "GRANT SELECT (request_hash) ON idempotency_keys TO nava_purge",
			undo:  "REVOKE SELECT (request_hash) ON idempotency_keys FROM nava_purge",
		},
		{
			name: "purge SELECT policy stops testing expiry", path: purgePath, table: "public.outbox",
			apply: "ALTER POLICY outbox_purge_select ON outbox USING (true)",
			undo:  "ALTER POLICY outbox_purge_select ON outbox USING (dispatched_at < now() - interval '7 days' OR dead_at < now() - interval '30 days')",
		},
		{
			name: "purge DELETE policy matches nothing", path: purgePath, table: "public.idempotency_keys",
			apply: "ALTER POLICY idempotency_purge ON idempotency_keys USING (false)",
			undo:  "ALTER POLICY idempotency_purge ON idempotency_keys USING (expires_at < now())",
		},
		{
			name: "tenants_activate stops testing provisioning", path: "nava_onboard cannot write to a non-provisioning tenant", table: "public.tenants",
			apply: "ALTER POLICY tenants_activate ON tenants USING (id = " + tenantSetting + ")",
			undo:  activate,
		},
		{
			name: "binding_onboard accepts any tenant", path: "nava_onboard cannot bind an active tenant", table: "public.tenant_identity_binding",
			apply: "ALTER POLICY binding_onboard ON tenant_identity_binding WITH CHECK (true)",
			undo:  "ALTER POLICY binding_onboard ON tenant_identity_binding WITH CHECK (tenant_id IN (SELECT id FROM tenants WHERE state = 'provisioning'))",
		},
		{
			name: "tenants_activate allows any state", path: "nava_onboard cannot move an active tenant back to provisioning", table: "public.tenants",
			apply: "ALTER POLICY tenants_activate ON tenants USING (id = " + tenantSetting + ") WITH CHECK (id = " + tenantSetting + ")",
			undo:  activate,
		},
		{
			// P3-06a (TICK-381): each seed table's guard on its own, since
			// Seed stops at its first refusal.
			name: "entity_definitions_onboard stops testing provisioning", path: seedActivePath, table: "public.entity_definitions",
			apply: "ALTER POLICY entity_definitions_onboard ON entity_definitions WITH CHECK (tenant_id = " + tenantSetting + ")",
			undo:  onboardSeed("entity_definitions"),
		},
		{
			name: "entity_definition_versions_onboard stops testing provisioning", path: seedActivePath, table: "public.entity_definition_versions",
			apply: "ALTER POLICY entity_definition_versions_onboard ON entity_definition_versions WITH CHECK (tenant_id = " + tenantSetting + ")",
			undo:  onboardSeed("entity_definition_versions"),
		},
		{
			// pg_hba is embedded, so its fixtures are internal/db/hba's
			// unit tests; this one breaks the grant layer under it.
			name: "nava_backup granted CONNECT", path: backupPath, table: "-",
			apply: "GRANT CONNECT ON DATABASE nava TO nava_backup",
			undo:  "REVOKE CONNECT ON DATABASE nava FROM nava_backup",
		},
	} {
		t.Run(f.name, func(t *testing.T) {
			p, ok := paths[f.path]
			if !ok {
				t.Fatalf("no CI-9 path %q", f.path)
			}
			before := statusOf(p.DB(t.Context(), cluster{t: t, e: suite}))
			if before != statusPass {
				t.Fatalf("before the fixture %q is %s, want PASS", f.path, before)
			}
			super := suite.superConn(t)
			snap := aclSnapshot(t.Context(), t, super)
			t.Cleanup(func() {
				ctx := context.Background()
				if _, err := super.Exec(ctx, f.undo); err != nil {
					t.Errorf("undo fixture: %v", err)
					return
				}
				// The undo text is hand-written: it must restore the
				// policies and grants exactly, or every later test runs
				// on a schema the migrations never built.
				if got := aclSnapshot(ctx, t, super); got != snap {
					t.Errorf("the undo did not restore the policies and grants:\nbefore %s\nafter  %s", snap, got)
				}
			})
			if _, err := super.Exec(t.Context(), f.apply); err != nil {
				t.Fatal(err)
			}
			err := p.DB(t.Context(), cluster{t: t, e: suite})
			after := statusOf(err)
			switch {
			case after != statusFail:
				t.Errorf("with the fixture %q is %s, want FAIL", f.path, after)
			case strings.Contains(err.Error(), "control:"):
				// A broken positive control proves nothing about the denial.
				after = statusFail + " (control)"
				t.Errorf("with the fixture %q failed its positive control, not a denial: %v", f.path, err)
			default:
				t.Logf("caught: %v", err)
			}
			suite.report.fixtures = append(suite.report.fixtures, fixtureResult{
				Case: "CI-9", Name: f.name, Role: p.Role, Table: f.table, Before: before, After: after,
			})
		})
	}
}

const (
	purgePath  = "nava_purge cannot read data columns or unexpired rows, and its purge deletes expired rows"
	backupPath = "nava_backup is rejected for SQL sessions and logical walsenders (T08's pg_hba cases)"
)

func statusOf(err error) string {
	if err != nil {
		return statusFail
	}
	return statusPass
}

// aclSnapshot is every policy and every ACL of the platform database's
// public tables, their columns and the database itself, as one string.
// ACLs are what has_table_privilege, has_any_column_privilege and
// has_database_privilege read, for every role at once.
func aclSnapshot(ctx context.Context, t *testing.T, super *pgx.Conn) string {
	t.Helper()
	var s string
	if err := super.QueryRow(ctx, `SELECT concat_ws(chr(10),
		(SELECT string_agg(format('%s.%s %s %s using=%s check=%s', tablename, policyname, cmd, roles, qual, with_check), chr(10)
		        ORDER BY tablename, policyname) FROM pg_policies WHERE schemaname = 'public'),
		(SELECT string_agg(format('%s %s', c.relname, c.relacl), chr(10) ORDER BY c.relname)
		   FROM pg_class c WHERE c.relnamespace = 'public'::regnamespace AND c.relkind IN ('r', 'p', 'v', 'm')),
		(SELECT string_agg(format('%s.%s %s', c.relname, a.attname, a.attacl), chr(10) ORDER BY c.relname, a.attname)
		   FROM pg_attribute a JOIN pg_class c ON c.oid = a.attrelid
		  WHERE c.relnamespace = 'public'::regnamespace AND a.attacl IS NOT NULL),
		(SELECT format('database %s', datacl) FROM pg_database WHERE datname = current_database()))`).Scan(&s); err != nil {
		t.Fatal(err)
	}
	return s
}

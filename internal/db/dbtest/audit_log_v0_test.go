//go:build integration

package dbtest

import (
	"strings"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/db"
)

// P5-10a (TICK-427), PLAN-005 D-5-1: S9's audit_log becomes audit_log_v0 with
// its rows, grants and RLS intact, and nothing on it keeps an audit_log_*
// name, so ADR-004's audit_log (P5-10b) can take the name and its _pkey.
// The test stops at the rename: P5-10b's migration then creates the new
// audit_log, and TestP510bAuditLog covers it.

const auditLogV0Migration = "20261102130000_audit_log_v0.sql"

func TestP510aAuditLogRenamed(t *testing.T) {
	m := freshCluster(t)
	if err := migrateOnlyFS(t, m, migrationsBefore(t, auditLogV0Migration)); err != nil {
		t.Fatalf("migrate to before %s: %v", auditLogV0Migration, err)
	}
	super := superNava(t)
	seedTenants(t, super, []string{tenantA}, nil)
	seedEntityTypes(t, super, tenantA)
	rec := pipelineRecord(tenantA)
	exec(t, super, insertRecord, tenantA, rec, tenantA, "TCK0000001", tenantA, tenantA, tenantA, tenantA)
	// The old name, as S9 wrote it before this migration.
	exec(t, super, strings.Replace(insertAudit, "audit_log_v0", "audit_log", 1), tenantA, rec, "pre-rename-key")
	before := queryString(t, super, "SELECT relfilenode::text FROM pg_class WHERE oid = 'audit_log'::regclass")

	if err := migrateOnlyFS(t, m, migrationsThrough(t, auditLogV0Migration)); err != nil {
		t.Fatalf("migrate through %s: %v", auditLogV0Migration, err)
	}

	// Same table, same storage, same row; the old name is free.
	if got := queryString(t, super, "SELECT relfilenode::text FROM pg_class WHERE oid = 'audit_log_v0'::regclass"); got != before {
		t.Errorf("audit_log_v0 relfilenode %s, want %s: the rename rewrote the table", got, before)
	}
	if got := queryString(t, super, "SELECT count(*)::text FROM audit_log_v0 WHERE tenant_id = $1 AND idempotency_key = 'pre-rename-key'", tenantA); got != "1" {
		t.Errorf("rows written before the rename = %s, want 1", got)
	}
	if got := queryString(t, super, "SELECT coalesce(to_regclass('public.audit_log')::text, '')"); got != "" {
		t.Errorf("audit_log still resolves to %q after the rename", got)
	}

	// Acceptance: every constraint, index and policy on audit_log_v0 is
	// named audit_log_v0_*, and the set is the one S9 created.
	names := `SELECT kind || ' ' || name FROM (
		SELECT 'constraint' AS kind, conname::text AS name FROM pg_constraint WHERE conrelid = 'audit_log_v0'::regclass
		UNION ALL SELECT 'index', indexrelid::regclass::text FROM pg_index WHERE indrelid = 'audit_log_v0'::regclass
		UNION ALL SELECT 'policy', polname::text FROM pg_policy WHERE polrelid = 'audit_log_v0'::regclass) o`
	if bad := queryString(t, super, "SELECT coalesce(string_agg(n, ', '), '') FROM ("+names+") x(n) WHERE n !~ ' audit_log_v0_'"); bad != "" {
		t.Errorf("still named for audit_log: %s", bad)
	}
	want := strings.Join([]string{
		"constraint audit_log_v0_causation_depth_check",
		"constraint audit_log_v0_pkey",
		"constraint audit_log_v0_record_fkey",
		"constraint audit_log_v0_source_check",
		"constraint audit_log_v0_tenant_fkey",
		"index audit_log_v0_idempotency_key",
		"index audit_log_v0_pkey",
		"index audit_log_v0_record",
		"policy audit_log_v0_insert",
		"policy audit_log_v0_select",
	}, ",")
	if got := queryString(t, super, "SELECT string_agg(n, ',' ORDER BY n) FROM ("+names+") x(n)"); got != want {
		t.Errorf("objects on audit_log_v0:\n%s\nwant:\n%s", got, want)
	}
	// No other relation in the schema keeps an audit_log_* name either.
	if bad := queryString(t, super, `SELECT coalesce(string_agg(relname::text, ', '), '') FROM pg_class
		WHERE relnamespace = 'public'::regnamespace AND relname ~ '^audit_log(_|$)' AND relname !~ '^audit_log_v0(_|$)'`); bad != "" {
		t.Errorf("relations still named audit_log*: %s", bad)
	}

	// The rest of the migrations, P5-10b's new audit_log included, then the
	// grants manifest: RLS and the append-only grants survived.
	if _, err := migrateFS(t, m, db.Migrations()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if got := queryString(t, super, "SELECT relrowsecurity::text || relforcerowsecurity::text FROM pg_class WHERE oid = 'audit_log_v0'::regclass"); got != "truetrue" {
		t.Errorf("audit_log_v0 RLS enabled/forced = %s, want truetrue", got)
	}
	for _, role := range []string{"nava_app", "nava_worker"} {
		got := queryString(t, super, `SELECT string_agg(p || '=' || has_table_privilege($1, 'audit_log_v0', p)::text, ',' ORDER BY p)
			FROM unnest(ARRAY['DELETE', 'INSERT', 'SELECT', 'TRUNCATE', 'UPDATE']) p`, role)
		// Read-only since P5-13b (TICK-452): INSERT is revoked too.
		if want := "DELETE=false,INSERT=false,SELECT=true,TRUNCATE=false,UPDATE=false"; got != want {
			t.Errorf("%s on audit_log_v0: %s, want %s", role, got, want)
		}
	}
}

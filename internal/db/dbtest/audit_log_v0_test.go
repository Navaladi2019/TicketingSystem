//go:build integration

package dbtest

import (
	"context"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/Navaladi2019/TicketingSystem/db"
	"github.com/Navaladi2019/TicketingSystem/internal/db/audited"
	"github.com/Navaladi2019/TicketingSystem/internal/db/grants"
	"github.com/Navaladi2019/TicketingSystem/internal/db/migrate"
)

// P5-10a (TICK-427), PLAN-005 D-5-1: S9's audit_log becomes audit_log_v0 with
// its rows, grants and RLS intact, and nothing on it keeps an audit_log_*
// name, so ADR-004's audit_log (P5-10b) can take the name and its _pkey.
// The test stops at the rename: P5-10b's migration then creates the new
// audit_log, and TestP510bAuditLog covers it.

// insertAudit is S9's audit insert, as it ran before the rename: tenant $1,
// record $2, idempotency key $3. audit_log_v0 itself was dropped by P5-17.
const insertAudit = `INSERT INTO audit_log_v0 (tenant_id, id, record_id, action, actor_id, source, after, idempotency_key)
	VALUES ($1, gen_random_uuid(), $2, 'create', NULL, 'workflow', '{}', $3)`

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

	// The migrations up to the one that drops the table, P5-10b's new
	// audit_log included, then the grants manifest: RLS and the read-only
	// grants survived. db/grants.yaml no longer lists audit_log_v0 (P5-17
	// dropped it), so this run lists it as it did until then: SELECT for the
	// two runtime roles, and the table exempt from audit.
	if _, err := migrateTo(t, m, migrationsBefore(t, auditLogV0Drop)); err != nil {
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

	// P5-17 (TICK-435): the full set of migrations drops it.
	if _, err := migrateFS(t, m, db.Migrations()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if got := queryString(t, super, "SELECT coalesce(to_regclass('public.audit_log_v0')::text, '')"); got != "" {
		t.Errorf("audit_log_v0 still exists after %s: %s", auditLogV0Drop, got)
	}
}

const auditLogV0Drop = "20261104100000_drop_audit_log_v0.sql"

// recordsAttachMigration attaches the commit-check trigger to records.
const recordsAttachMigration = "20261104110000_records_audit_commit_check.sql"

// migrateTo runs the migrations in fsys and the grants manifest and gate as
// migrateFS does, but with the manifests as they were at that version:
// audit_log_v0 in the grants and exempt from audit before the drop, and
// records not yet attached before its attach migration.
func migrateTo(t *testing.T, m *grants.Manifest, fsys fs.FS) (migrate.Result, error) {
	t.Helper()
	withV0 := *m
	withV0.Roles = make(map[string]grants.Role, len(m.Roles))
	for name, r := range m.Roles {
		if name == "nava_app" || name == "nava_worker" {
			tables := make(map[string]map[string][]string, len(r.Tables)+1)
			for tbl, privs := range r.Tables {
				tables[tbl] = privs
			}
			tables["audit_log_v0"] = map[string][]string{"SELECT": {}}
			r.Tables = tables
		}
		withV0.Roles[name] = r
	}
	a := *auditedManifest(t)
	a.Exempt = make(map[string]audited.Exemption, len(a.Exempt)+1)
	for name, e := range auditedManifest(t).Exempt {
		a.Exempt[name] = e
	}
	a.Exempt["audit_log_v0"] = audited.Exemption{Reason: "read-only until P5-17 drops it"}
	// The manifest as it was at that version: records is attached by
	// 20261104110000 (P5-13c, TICK-530), so before it records is not
	// attached: true yet.
	if _, err := fs.Stat(fsys, recordsAttachMigration); err != nil {
		a.Audited = make(map[string]audited.Table, len(a.Audited))
		for name, e := range auditedManifest(t).Audited {
			a.Audited[name] = e
		}
		r := a.Audited["records"]
		r.Attached = false
		a.Audited["records"] = r
	}
	// CI-A3 pins audit_log_select's roles (ADR-004-A1 A1-9), which only
	// 20261104133000 sets. It alters two policies of tables that exist from
	// 20261102233600 on and depends on nothing after it, so a run that stops
	// earlier applies it out of order, as the migrate job allows.
	if _, err := fs.Stat(fsys, auditReaderPoliciesMigration); err != nil {
		fsys = withMigrationFrom(t, fsys, auditReaderPoliciesMigration)
	}
	log, _ := newLog()
	return migrate.Run(context.Background(), roleConfig(t, withV0.Owner(), withV0.PlatformDatabase),
		fsys, &withV0, &a, globalTables(t), log, migrate.Options{LockRetrySeconds: 1})
}

// auditReaderPoliciesMigration is ADR-004-A1 A1-3's policy migration (TICK-558).
const auditReaderPoliciesMigration = "20261104133000_audit_reader_policies.sql"

// withMigrationFrom is fsys plus the real migration name.
func withMigrationFrom(t *testing.T, fsys fs.FS, name string) fs.FS {
	t.Helper()
	out := fstest.MapFS{}
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		b, err := fs.ReadFile(fsys, e.Name())
		if err != nil {
			t.Fatal(err)
		}
		out[e.Name()] = &fstest.MapFile{Data: b}
	}
	b, err := fs.ReadFile(db.Migrations(), name)
	if err != nil {
		t.Fatal(err)
	}
	out[name] = &fstest.MapFile{Data: b}
	return out
}

//go:build integration

package dbtest

import (
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/db"
)

// TICK-715: under FORCE ROW LEVEL SECURITY the owner's bare VALIDATE
// CONSTRAINT of a foreign key checks nothing (docs/data/fk-validate-force-rls.md).
// 20261104013500 and 20261104043100 validated their keys that way. The two
// re-check migrations validate them again under nava_migrator SELECT policies
// that exist only inside their transaction (the P3-40 pattern).
const (
	membersRecheckMigration = "20261112100000_group_members_group_live_fkey_recheck.sql"
	recordsRecheckMigration = "20261112100100_records_assignment_group_fkey_recheck.sql"
)

// Acceptance: an orphan makes each re-check migration fail, records no
// version and leaves no policy behind; once the orphan is gone the migration
// applies and the key is validated.
func TestTICK715FKRecheckFailsOnOrphan(t *testing.T) {
	m := freshCluster(t)
	if err := migrateOnlyFS(t, m, migrationsBefore(t, membersRecheckMigration)); err != nil {
		t.Fatalf("migrate to before %s: %v", membersRecheckMigration, err)
	}
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, nil)
	seedPipeline(t, super, tenantA, tenantB) // a group, a member and a record in that group per tenant

	// Orphan 1: tenant A's group is tombstoned while it still has a member,
	// written past the RI triggers, as a vacuous VALIDATE would have let a
	// pre-existing row through.
	execReplica(t, super, "UPDATE groups SET deleted_at = now() WHERE tenant_id = $1 AND id = $1", tenantA)
	// Orphan 2: tenant B's record names a group that does not exist.
	execReplica(t, super, "UPDATE records SET assignment_group_id = gen_random_uuid() WHERE tenant_id = $1 AND id = $2",
		tenantB, pipelineRecord(tenantB))
	if got := orphanCounts(t, super); got != "1/1" {
		t.Fatalf("orphans before the re-check (members/records) = %s, want 1/1", got)
	}

	_, err := migrateFS(t, m, db.Migrations())
	if err == nil || !strings.Contains(err.Error(), "group_members_group_live_fkey") {
		t.Fatalf("re-check over a member of a tombstoned group: %v, want a violation of group_members_group_live_fkey", err)
	}
	wantVersionRows(t, super, membersRecheckMigration, "0")
	noMigratorValidatePolicies(t, super, "after the failed group_members re-check")
	if !queryBool(t, super, "SELECT convalidated FROM pg_constraint WHERE conname = 'group_members_group_live_fkey'") {
		t.Error("the failed re-check changed group_members_group_live_fkey; it must roll back whole")
	}

	exec(t, super, "UPDATE groups SET deleted_at = NULL WHERE tenant_id = $1 AND id = $1", tenantA)
	_, err = migrateFS(t, m, db.Migrations())
	if err == nil || !strings.Contains(err.Error(), "records_assignment_group_fkey") {
		t.Fatalf("re-check over a record naming a missing group: %v, want a violation of records_assignment_group_fkey", err)
	}
	wantVersionRows(t, super, membersRecheckMigration, "1")
	wantVersionRows(t, super, recordsRecheckMigration, "0")
	noMigratorValidatePolicies(t, super, "after the failed records re-check")

	execReplica(t, super, "UPDATE records SET assignment_group_id = $1 WHERE tenant_id = $1 AND id = $2",
		tenantB, pipelineRecord(tenantB))
	if _, err := migrateFS(t, m, db.Migrations()); err != nil {
		t.Fatalf("re-check on a clean database: %v", err)
	}
	wantVersionRows(t, super, recordsRecheckMigration, "1")
	noMigratorValidatePolicies(t, super, "after both re-checks")
	if got := queryString(t, super, `SELECT string_agg(conname || '=' || convalidated, ',' ORDER BY conname) FROM pg_constraint
		WHERE conname IN ('group_members_group_live_fkey', 'records_assignment_group_fkey')`); got !=
		"group_members_group_live_fkey=true,records_assignment_group_fkey=true" {
		t.Errorf("keys after the re-check = %s", got)
	}
	// The re-created keys keep their definitions, RESTRICT included.
	if got := queryString(t, super, `SELECT string_agg(conname || ': ' || pg_get_constraintdef(oid), E'\n' ORDER BY conname) FROM pg_constraint
		WHERE conname IN ('group_members_group_live_fkey', 'records_assignment_group_fkey')`); got != strings.Join([]string{
		"group_members_group_live_fkey: FOREIGN KEY (tenant_id, group_id, live) REFERENCES groups(tenant_id, id, live)",
		"records_assignment_group_fkey: FOREIGN KEY (tenant_id, assignment_group_id) REFERENCES groups(tenant_id, id) ON DELETE RESTRICT",
	}, "\n") {
		t.Errorf("key definitions after the re-check:\n%s", got)
	}
}

// orphanCounts is scratch/tick-715-fk-recheck/orphans.sql run as the
// superuser, which bypasses RLS: "<group_members orphans>/<records orphans>".
func orphanCounts(t *testing.T, super *pgx.Conn) string {
	t.Helper()
	return queryString(t, super, `SELECT
		(SELECT count(*) FROM group_members gm WHERE NOT EXISTS (SELECT 1 FROM groups g
			WHERE g.tenant_id = gm.tenant_id AND g.id = gm.group_id AND g.live = gm.live))::text || '/' ||
		(SELECT count(*) FROM records r WHERE r.assignment_group_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM groups g
			WHERE g.tenant_id = r.tenant_id AND g.id = r.assignment_group_id))::text`)
}

// execReplica runs one statement as the superuser with
// session_replication_role = replica, so no trigger fires: it writes the
// orphan a foreign key, or the audit commit check, would refuse.
func execReplica(t *testing.T, super *pgx.Conn, sql string, args ...any) {
	t.Helper()
	ctx := context.Background()
	tx, err := super.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, "SET LOCAL session_replication_role = replica"); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, sql, args...); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
}

func wantVersionRows(t *testing.T, super *pgx.Conn, migration, want string) {
	t.Helper()
	version := migration[:strings.IndexByte(migration, '_')]
	if got := queryString(t, super, "SELECT count(*)::text FROM goose_db_version WHERE version_id = $1::bigint", version); got != want {
		t.Errorf("goose_db_version rows for %s = %s, want %s", migration, got, want)
	}
}

// noMigratorValidatePolicies: the SELECT USING (true) policies a re-check
// gives nava_migrator must not outlive its transaction, whether VALIDATE
// fails (rollback) or succeeds (dropped before commit). A survivor would be
// an unscoped read path on a tenant table.
func noMigratorValidatePolicies(t *testing.T, conn *pgx.Conn, when string) {
	t.Helper()
	if got := queryString(t, conn, `SELECT coalesce(string_agg(tablename || '.' || policyname, ', '), '') FROM pg_policies
		WHERE 'nava_migrator' = ANY (roles) OR policyname LIKE '%migrator_validate'`); got != "" {
		t.Errorf("%s: nava_migrator policies left behind: %s", when, got)
	}
}

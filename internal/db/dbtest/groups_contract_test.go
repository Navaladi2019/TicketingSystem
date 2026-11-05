//go:build integration

package dbtest

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Navaladi2019/TicketingSystem/db"
)

// ADR-004-E5 step (ii), contract (TICK-538; D2, D10). Seeded identity ids
// equal the tenant id, and the seeded pipeline record of a tenant names that
// tenant's group (seedPipeline).

const (
	groupsContractMigration = "20261104043000_groups_contract.sql"
	groupsValidateMigration = "20261104043100_groups_contract_validate.sql"
)

// CI-E5-2: after step (ii), DELETE FROM groups as nava_groupsync is denied
// (42501) and no groups_delete policy is left; the owner's delete of a group
// a record names fails with 23503 (RESTRICT), and the record keeps its
// group; nava_groupsync cannot UPDATE id or keycloak_group_id. The owner
// here is the superuser: it bypasses RLS, so the delete reaches the row and
// only the FK stops it.
func TestE5GroupsContractNoGroupDelete(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA}, nil)
	seedPipeline(t, super, tenantA)
	sync := asRole(t, "nava_groupsync")

	_, err := execIn(sync, tenantA, "DELETE FROM groups WHERE tenant_id = $1 AND id = $1", tenantA)
	wantSQLState(t, "nava_groupsync deletes its own tenant's group", err, "42501")
	if got := queryString(t, super, "SELECT count(*)::text FROM pg_policies WHERE tablename = 'groups' AND (policyname = 'groups_delete' OR cmd = 'DELETE')"); got != "0" {
		t.Errorf("DELETE policies left on groups: %s, want 0", got)
	}

	_, err = super.Exec(context.Background(), "DELETE FROM groups WHERE tenant_id = $1 AND id = $1", tenantA)
	wantSQLState(t, "owner deletes a group a record names", err, "23503")
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.ConstraintName != "records_assignment_group_fkey" {
		t.Errorf("owner delete refused by %s, want records_assignment_group_fkey", pgErr.ConstraintName)
	}
	if got := queryString(t, super, "SELECT coalesce(assignment_group_id::text, 'null') FROM records WHERE tenant_id = $1 AND id = $2",
		tenantA, pipelineRecord(tenantA)); got != tenantA {
		t.Errorf("record group after the refused delete = %s, want %s", got, tenantA)
	}

	for _, w := range []string{
		"UPDATE groups SET id = gen_random_uuid() WHERE tenant_id = $1",
		"UPDATE groups SET keycloak_group_id = 'other' WHERE tenant_id = $1",
	} {
		_, err := execIn(sync, tenantA, w, tenantA)
		wantSQLState(t, "nava_groupsync: "+w, err, "42501")
	}
}

// CI-E5-8: (ii-a) runs under lock_timeout. With a write transaction open on
// records it fails with 55P03 inside the timeout instead of queueing, and
// changes nothing; once the writer is gone it applies, leaving the FK
// RESTRICT and NOT VALID. (ii-b) is a migration of its own and validates
// it. After both: confdeltype = 'r', convalidated = true.
func TestE5GroupsContractMigrationLocks(t *testing.T) {
	m := freshCluster(t)
	if err := migrateOnlyFS(t, m, migrationsBefore(t, groupsContractMigration)); err != nil {
		t.Fatalf("migrate to before %s: %v", groupsContractMigration, err)
	}
	super := superNava(t)
	fk := func() string {
		return queryString(t, super, `SELECT confdeltype::text || ':' || convalidated::text FROM pg_constraint
			WHERE conrelid = 'records'::regclass AND conname = 'records_assignment_group_fkey'`)
	}
	if got := fk(); got != "n:true" {
		t.Fatalf("before step (ii): %s, want n:true (SET NULL, validated)", got)
	}
	groupsyncDelete := func() bool {
		return queryBool(t, super, "SELECT has_table_privilege('nava_groupsync', 'public.groups', 'DELETE')")
	}
	// migrateOnlyFS applies no grants manifest; give nava_groupsync the
	// DELETE that step (i)'s db/grants.yaml held, so (ii-a)'s REVOKE shows.
	exec(t, super, "GRANT DELETE ON groups TO nava_groupsync")

	// A record writer holds ROW EXCLUSIVE on records; (ii-a)'s DROP
	// CONSTRAINT needs ACCESS EXCLUSIVE, which conflicts with it.
	writer := superNava(t)
	ctx := context.Background()
	tx, err := writer.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, "LOCK TABLE records IN ROW EXCLUSIVE MODE"); err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	err = migrateOnlyFS(t, m, migrationsThrough(t, groupsContractMigration))
	waited := time.Since(start)
	_ = tx.Rollback(ctx)
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "55P03" {
		t.Fatalf("(ii-a) behind an open records writer: %v, want SQLSTATE 55P03 (lock_timeout)", err)
	}
	// lock_timeout is 5s; the margin covers connecting and goose.
	if waited > 15*time.Second {
		t.Errorf("(ii-a) waited %s behind the writer, want about lock_timeout (5s)", waited)
	}
	if got := fk(); got != "n:true" {
		t.Errorf("after the failed (ii-a): %s, want n:true (nothing changed)", got)
	}
	if !groupsyncDelete() {
		t.Error("after the failed (ii-a): DELETE on groups already revoked; the migration is not atomic")
	}

	if err := migrateOnlyFS(t, m, migrationsThrough(t, groupsContractMigration)); err != nil {
		t.Fatalf("(ii-a) with no writer: %v", err)
	}
	if got := fk(); got != "r:false" {
		t.Errorf("after (ii-a): %s, want r:false (RESTRICT, NOT VALID until ii-b)", got)
	}
	if groupsyncDelete() {
		t.Error("after (ii-a): nava_groupsync still holds DELETE on groups")
	}

	if _, err := migrateFS(t, m, db.Migrations()); err != nil {
		t.Fatalf("(ii-b) and the rest: %v", err)
	}
	if got := fk(); got != "r:true" {
		t.Errorf("after (ii-b): %s, want r:true (RESTRICT, validated)", got)
	}
}

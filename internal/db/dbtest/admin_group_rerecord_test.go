//go:build integration

package dbtest

import (
	"reflect"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// TICK-530 (ADR-004-E5 R-E5-4, N3): the SQL of docs/operator/onboarding.md,
// "Pointer missing after a group came back (R-E5-4)", runs against the real
// schema as pgadmin and re-records the admins or auditors pointer of an active
// tenant whose group group sync revived. Read out of the runbook, like
// admin_group_runbook_test.go.

const (
	reConfirm, reRecord, reCommit, reRollback, reCheck = 0, 1, 2, 3, 4
	reBlockCount                                       = 5
)

func reRecordBlocks(t *testing.T, tenant, kcGroup string, g pointerGroup) []string {
	t.Helper()
	out := sectionBlocks(t, "## Pointer missing after a group came back (R-E5-4)", "", tenant, kcGroup, g)
	if len(out) != reBlockCount {
		t.Fatalf("R-E5-4 section has %d sql blocks, want %d (confirm, re-record, commit, rollback, check)", len(out), reBlockCount)
	}
	return out
}

// revivedGroup leaves tenant (active) with g's group live under Keycloak id
// kc, with a member, and no pointer row: what group sync leaves after it
// tombstoned the group, deleted its pointer (D5) and revived it (D1).
func revivedGroup(t *testing.T, super *pgx.Conn, tenant, kc string, g pointerGroup) uuid.UUID {
	t.Helper()
	tid := uuid.MustParse(tenant)
	id := g.id(tid)
	exec(t, super, `INSERT INTO groups (tenant_id, id, keycloak_group_id, path)
		VALUES ($1::uuid, $2, $3, '/t/' || $1::text || '/' || $4::text)`, tenant, id, kc, g.name)
	p := uuid.New()
	exec(t, super, `INSERT INTO principals (tenant_id, id, keycloak_user_id, enabled, synced_at) VALUES ($1::uuid, $2, $3, true, now())`,
		tenant, p, "kc-user-"+g.name)
	exec(t, super, `INSERT INTO group_members (tenant_id, group_id, principal_id) VALUES ($1::uuid, $2, $3)`, tenant, id, p)
	return id
}

func TestRE54RunbookReRecordsPointer(t *testing.T) {
	for _, g := range pointerGroups {
		t.Run(g.name, func(t *testing.T) {
			migratedPlatform(t)
			super := superNava(t)
			seedTenants(t, super, []string{tenantA}, nil)
			kc := "kc-" + g.name
			id := revivedGroup(t, super, tenantA, kc, g)
			blocks := reRecordBlocks(t, tenantA, kc, g)

			if got := runBlock(t, super, "confirm", blocks[reConfirm]).counts(); !reflect.DeepEqual(got, []int{1, 0}) {
				t.Fatalf("confirm rows = %v, want [1 0]", got)
			}
			if got := runBlock(t, super, "re-record", blocks[reRecord]).tags; !reflect.DeepEqual(got, []string{"BEGIN", "INSERT 0 1"}) {
				t.Fatalf("re-record tags = %v, want [BEGIN INSERT 0 1]", got)
			}
			if got := runBlock(t, super, "commit", blocks[reCommit]).tags; !reflect.DeepEqual(got, []string{"COMMIT"}) {
				t.Fatalf("commit tags = %v", got)
			}
			chk := runBlock(t, super, "check", blocks[reCheck])
			if len(chk.rows) != 1 || len(chk.rows[0]) != 1 {
				t.Fatalf("check rows = %v, want one row", chk.rows)
			}
			if row := chk.rows[0][0]; row[0] != "active" || row[2] != kc || row[3] != nil {
				t.Errorf("check row = %v, want state active, keycloak_group_id %s, deleted_at empty", row, kc)
			}
			if got := queryString(t, super, `SELECT group_id::text FROM `+g.table+` WHERE tenant_id = $1`, tenantA); got != id.String() {
				t.Errorf("%s names %s, want %s", g.table, got, id)
			}
		})
	}
}

// A group that is still tombstoned is not re-recorded: the confirm step finds
// no live row, the insert writes nothing, and the operator rolls back.
func TestRE54RunbookRefusesTombstonedGroup(t *testing.T) {
	for _, g := range pointerGroups {
		t.Run(g.name, func(t *testing.T) {
			migratedPlatform(t)
			super := superNava(t)
			seedTenants(t, super, []string{tenantA}, nil)
			kc := "kc-" + g.name
			id := revivedGroup(t, super, tenantA, kc, g)
			exec(t, super, "DELETE FROM group_members WHERE tenant_id = $1 AND group_id = $2", tenantA, id)
			exec(t, super, "UPDATE groups SET deleted_at = now() WHERE tenant_id = $1 AND id = $2", tenantA, id)
			blocks := reRecordBlocks(t, tenantA, kc, g)

			if got := runBlock(t, super, "confirm", blocks[reConfirm]).counts(); !reflect.DeepEqual(got, []int{0, 0}) {
				t.Fatalf("confirm rows = %v, want [0 0]", got)
			}
			if got := runBlock(t, super, "re-record", blocks[reRecord]).tags; !reflect.DeepEqual(got, []string{"BEGIN", "INSERT 0 0"}) {
				t.Fatalf("re-record tags = %v, want [BEGIN INSERT 0 0]", got)
			}
			runBlock(t, super, "rollback", blocks[reRollback])
			if got := queryString(t, super, `SELECT count(*)::text FROM `+g.table+` WHERE tenant_id = $1`, tenantA); got != "0" {
				t.Errorf("%s rows = %s, want 0", g.table, got)
			}
		})
	}
}

// Adversarial (N3): the re-record runs as the superuser, so only its WHERE
// clause scopes it. Tenant B holds the same revived group under the same
// Keycloak id, also with no pointer. Re-recording A's pointer must write
// exactly A's row and leave every one of B's identity rows unchanged; B stays
// without a pointer (fail closed).
func TestRE54RunbookReRecordCannotTouchOtherTenant(t *testing.T) {
	for _, g := range pointerGroups {
		t.Run(g.name, func(t *testing.T) {
			migratedPlatform(t)
			super := superNava(t)
			seedTenants(t, super, []string{tenantA, tenantB}, nil)
			kc := "kc-shared-" + g.name
			revivedGroup(t, super, tenantA, kc, g)
			revivedGroup(t, super, tenantB, kc, g)
			snapshotB := tenantIdentitySnapshot(t, super, tenantB)
			before := snapshotB()

			blocks := reRecordBlocks(t, tenantA, kc, g)
			if got := runBlock(t, super, "confirm", blocks[reConfirm]).counts(); !reflect.DeepEqual(got, []int{1, 0}) {
				t.Fatalf("confirm rows = %v, want [1 0]: the confirm step saw another tenant's rows", got)
			}
			if got := runBlock(t, super, "re-record", blocks[reRecord]).tags; !reflect.DeepEqual(got, []string{"BEGIN", "INSERT 0 1"}) {
				t.Fatalf("re-record tags = %v, want exactly INSERT 0 1: the insert reached beyond tenant A", got)
			}
			runBlock(t, super, "commit", blocks[reCommit])
			if after := snapshotB(); after != before {
				t.Fatalf("tenant B identity rows changed by tenant A's re-record:\nbefore %s\nafter  %s", before, after)
			}
			if got := queryString(t, super, `SELECT count(*)::text FROM `+g.table+` WHERE tenant_id = $1`, tenantB); got != "0" {
				t.Errorf("tenant B %s rows = %s, want 0", g.table, got)
			}
			if rows := runBlock(t, super, "check", blocks[reCheck]).counts(); !reflect.DeepEqual(rows, []int{1}) {
				t.Fatalf("check rows = %v, want [1]: the check reported another tenant's pointer", rows)
			}
		})
	}
}

// As for TICK-372: nava_migrator, the owner, is NOBYPASSRLS and the tables are
// FORCE RLS, so the re-record as nava_migrator inserts nothing.
func TestRE54RunbookSeesNothingAsMigrator(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA}, nil)
	revivedGroup(t, super, tenantA, "kc-admins", adminsGroup)
	blocks := reRecordBlocks(t, tenantA, "kc-admins", adminsGroup)

	owner := asRole(t, "nava_migrator")
	if got := runBlock(t, owner, "confirm as nava_migrator", blocks[reConfirm]).counts(); !reflect.DeepEqual(got, []int{0, 0}) {
		t.Errorf("confirm rows as nava_migrator = %v, want [0 0]", got)
	}
	if got := runBlock(t, owner, "re-record as nava_migrator", blocks[reRecord]).tags; !reflect.DeepEqual(got, []string{"BEGIN", "INSERT 0 0"}) {
		t.Errorf("re-record tags as nava_migrator = %v, want [BEGIN INSERT 0 0]", got)
	}
	runBlock(t, owner, "rollback", blocks[reRollback])
}

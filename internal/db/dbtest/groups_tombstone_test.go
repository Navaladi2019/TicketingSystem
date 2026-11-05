//go:build integration

package dbtest

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/db/catalog"
)

// ADR-004-E5 step (i) (TICK-534): groups are tombstoned, the database refuses
// members in a tombstoned group (D11), and nava_groupsync deletes an admins
// or auditors pointer only once its group is tombstoned (D5, I1). Seeded
// identity ids equal the tenant id: tenant X's group X has principal X as its
// only member.

const (
	tombstoneGroup = "UPDATE groups SET deleted_at = now() WHERE tenant_id = $1 AND id = $2"
	insertMember   = "INSERT INTO group_members (tenant_id, group_id, principal_id) VALUES ($1, $2, $3)"
	insertGroup    = "INSERT INTO groups (tenant_id, id, keycloak_group_id, path) VALUES ($1::uuid, $2::uuid, 'kc-'||$2::uuid::text, '/t/' || $1::text || '/g')"
)

// CI-E5-5 (database half): both refusals hold as nava_groupsync, as the
// table owner and as the superuser, and a group that has lost its members
// tombstones and revives.
func TestE5MembersOfTombstonedGroupRefused(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA}, nil)
	seedIdentity(t, super, tenantA)
	for _, c := range []string{"group_members_live_check", "group_members_group_live_fkey"} {
		if !queryBool(t, super, "SELECT convalidated FROM pg_constraint WHERE conname = $1", c) {
			t.Errorf("%s is not validated after the validate migration", c)
		}
	}
	gone := uuid.NewString()
	exec(t, super, insertGroup, tenantA, gone)
	exec(t, super, tombstoneGroup, tenantA, gone)

	// Each case runs its two writes in transactions of their own, rolled
	// back, so the next one starts from the same rows.
	type runner func(t *testing.T, sql string, args ...any) error
	sync := asRole(t, "nava_groupsync")
	runners := map[string]runner{
		"nava_groupsync": func(t *testing.T, sql string, args ...any) error {
			return rolledBack(t, sync, func(tx pgx.Tx) error {
				if _, err := tx.Exec(context.Background(), "SELECT set_config('app.tenant_id', $1, true)", tenantA); err != nil {
					return err
				}
				_, err := tx.Exec(context.Background(), sql, args...)
				return err
			})
		},
		// The owner, with FORCE RLS lifted in the same rolled-back
		// transaction so that RLS (42501) cannot be what refuses the write.
		"owner nava_migrator": func(t *testing.T, sql string, args ...any) error {
			return rolledBack(t, super, func(tx pgx.Tx) error {
				for _, s := range []string{
					"ALTER TABLE groups NO FORCE ROW LEVEL SECURITY",
					"ALTER TABLE group_members NO FORCE ROW LEVEL SECURITY",
					"SET LOCAL ROLE nava_migrator",
				} {
					if _, err := tx.Exec(context.Background(), s); err != nil {
						t.Fatalf("%s: %v", s, err)
					}
				}
				_, err := tx.Exec(context.Background(), sql, args...)
				return err
			})
		},
		"superuser": func(t *testing.T, sql string, args ...any) error {
			return rolledBack(t, super, func(tx pgx.Tx) error {
				_, err := tx.Exec(context.Background(), sql, args...)
				return err
			})
		},
	}
	for who, run := range runners {
		t.Run(who, func(t *testing.T) {
			wantSQLState(t, "insert a member into a tombstoned group", run(t, insertMember, tenantA, gone, tenantA), "23503")
			wantSQLState(t, "tombstone a group with a member", run(t, tombstoneGroup, tenantA, tenantA), "23503")
			wantSQLState(t, "insert a member with live = false", run(t,
				"INSERT INTO group_members (tenant_id, group_id, principal_id, live) VALUES ($1, $2, $3, false)", tenantA, tenantA, uuid.NewString()), "23514")
		})
	}

	// D1 as sync runs it: members first, then the tombstone; then the revive
	// on the same id, then the memberships again.
	err := inTenantTx(sync, tenantA, func(tx pgx.Tx) error {
		ctx := context.Background()
		if _, err := tx.Exec(ctx, "DELETE FROM group_members WHERE tenant_id = $1 AND group_id = $1", tenantA); err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, tombstoneGroup, tenantA, tenantA)
		if err == nil && tag.RowsAffected() != 1 {
			t.Errorf("tombstone after the members are gone: %d rows, want 1", tag.RowsAffected())
		}
		return err
	})
	if err != nil {
		t.Fatalf("nava_groupsync tombstones an emptied group: %v", err)
	}
	if got := queryString(t, super, "SELECT live::text FROM groups WHERE tenant_id = $1 AND id = $1", tenantA); got != "false" {
		t.Errorf("groups.live after the tombstone = %s, want false", got)
	}
	err = inTenantTx(sync, tenantA, func(tx pgx.Tx) error {
		ctx := context.Background()
		if _, err := tx.Exec(ctx, "UPDATE groups SET deleted_at = NULL WHERE tenant_id = $1 AND id = $1", tenantA); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, insertMember, tenantA, tenantA, tenantA)
		return err
	})
	if err != nil {
		t.Fatalf("nava_groupsync revives the group and restores its member: %v", err)
	}
}

// AC4 (D2): nava_groupsync's UPDATE on groups is path and deleted_at only.
func TestE5GroupsyncCannotRekeyGroups(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA}, nil)
	seedIdentity(t, super, tenantA)
	sync := asRole(t, "nava_groupsync")
	for _, w := range []string{
		"UPDATE groups SET id = gen_random_uuid() WHERE tenant_id = $1",
		"UPDATE groups SET keycloak_group_id = 'other' WHERE tenant_id = $1",
		"UPDATE groups SET tenant_id = tenant_id WHERE tenant_id = $1",
	} {
		_, err := execIn(sync, tenantA, w, tenantA)
		wantSQLState(t, "nava_groupsync: "+w, err, "42501")
	}
	if n, err := execIn(sync, tenantA, "UPDATE groups SET path = path, deleted_at = NULL WHERE tenant_id = $1", tenantA); err != nil || n != 1 {
		t.Errorf("nava_groupsync updates path and deleted_at: %d rows, %v; want 1, nil", n, err)
	}
}

// pointerTables are the two D5 pointer tables.
var pointerTables = []string{"tenant_admin_group", "tenant_auditor_group"}

// CI-E5-5b, I1: in one transaction, as nava_groupsync, sync removes the
// recorded group's members, tombstones it, reads the pointer and deletes it.
// The transaction commits and the pointer row is gone. Run for the admins
// and the auditors pointer.
func TestE5TombstoneRecordedGroupDeletesPointer(t *testing.T) {
	for _, table := range pointerTables {
		t.Run(table, func(t *testing.T) {
			migratedPlatform(t)
			super := superNava(t)
			seedTenants(t, super, []string{tenantA}, nil)
			seedIdentity(t, super, tenantA)
			exec(t, super, "INSERT INTO "+table+" (tenant_id, group_id) VALUES ($1, $1)", tenantA)
			sync := asRole(t, "nava_groupsync")
			err := inTenantTx(sync, tenantA, func(tx pgx.Tx) error {
				ctx := context.Background()
				var recorded string
				if err := tx.QueryRow(ctx, "SELECT group_id::text FROM "+table+" WHERE tenant_id = $1", tenantA).Scan(&recorded); err != nil {
					return err
				}
				if recorded != tenantA {
					t.Errorf("probe read %s, want %s", recorded, tenantA)
				}
				if _, err := tx.Exec(ctx, "DELETE FROM group_members WHERE tenant_id = $1 AND group_id = $2", tenantA, recorded); err != nil {
					return err
				}
				if _, err := tx.Exec(ctx, tombstoneGroup, tenantA, recorded); err != nil {
					return err
				}
				tag, err := tx.Exec(ctx, "DELETE FROM "+table+" WHERE tenant_id = $1 AND group_id = $2", tenantA, recorded)
				if err == nil && tag.RowsAffected() != 1 {
					t.Errorf("pointer delete: %d rows, want 1", tag.RowsAffected())
				}
				return err
			})
			if err != nil {
				t.Fatalf("tombstone of the recorded group: %v", err)
			}
			if got := queryString(t, super, "SELECT count(*)::text FROM "+table+" WHERE tenant_id = $1", tenantA); got != "0" {
				t.Errorf("%s rows after the commit: %s, want 0", table, got)
			}
		})
	}
}

// AC3: holding tenant A's context, or none, nava_groupsync cannot delete
// tenant B's pointer, even though B's group is tombstoned.
func TestE5PointerDeleteCrossTenant(t *testing.T) {
	for _, table := range pointerTables {
		t.Run(table, func(t *testing.T) {
			migratedPlatform(t)
			super := superNava(t)
			seedTenants(t, super, []string{tenantA, tenantB}, nil)
			seedIdentity(t, super, tenantA, tenantB)
			for _, id := range []string{tenantA, tenantB} {
				exec(t, super, "DELETE FROM group_members WHERE tenant_id = $1", id)
				exec(t, super, tombstoneGroup, id, id)
				exec(t, super, "INSERT INTO "+table+" (tenant_id, group_id) VALUES ($1, $1)", id)
			}
			sync := asRole(t, "nava_groupsync")
			for _, ctxTenant := range []string{tenantA, ""} {
				for _, sql := range []string{
					"DELETE FROM " + table + " WHERE tenant_id = $1",
					"DELETE FROM " + table + " WHERE group_id = $1",
				} {
					if n, err := execIn(sync, ctxTenant, sql, tenantB); err != nil || n != 0 {
						t.Errorf("tenant %q: %s for B: %d rows, %v; want 0, nil", ctxTenant, sql, n, err)
					}
				}
			}
			if got := queryString(t, super, "SELECT count(*)::text FROM "+table+" WHERE tenant_id = $1", tenantB); got != "1" {
				t.Errorf("B's %s rows = %s, want 1", table, got)
			}
			// An unscoped delete in A's context reaches A's row only.
			if n, err := execIn(sync, tenantA, "DELETE FROM "+table); err != nil || n != 1 {
				t.Errorf("unscoped delete in A: %d rows, %v; want 1 (A's own), nil", n, err)
			}
			if got := queryString(t, super, "SELECT count(*)::text FROM "+table+" WHERE tenant_id = $1", tenantB); got != "1" {
				t.Errorf("B's %s rows after A's unscoped delete = %s, want 1", table, got)
			}
		})
	}
}

// CI-E5-5b, the catalogue half: both D5 policies are pinned in
// catalog.PinnedTenantPolicies, the database has them in exactly that form,
// and the guard flags a copy that drops the tombstone test (a plain tenant
// DELETE policy would let sync remove a live group's pointer).
func TestE5PointerDeletePoliciesPinned(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	pinned := catalog.PinnedTenantPolicies()
	ctx := context.Background()
	for _, table := range pointerTables {
		key := "public." + table + "." + table + "_delete_tombstoned"
		want, ok := pinned[key]
		if !ok {
			t.Fatalf("%s is not in PinnedTenantPolicies", key)
		}
		var got catalog.Policy
		var roles []string
		err := super.QueryRow(ctx, `SELECT cmd, roles::text[], coalesce(qual, ''), coalesce(with_check, '')
			FROM pg_policies WHERE schemaname = 'public' AND tablename = $1 AND policyname = $2`,
			table, table+"_delete_tombstoned").Scan(&got.Cmd, &roles, &got.Qual, &got.Check)
		if err != nil {
			t.Fatalf("%s: %v", key, err)
		}
		got.Roles = strings.Join(roles, ",")
		if got != want {
			t.Errorf("%s = %q, want %q", key, got, want)
		}

		t.Run("guard flags "+table+" without the tombstone test", func(t *testing.T) {
			tx, err := super.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = tx.Rollback(ctx) }()
			for _, s := range []string{
				"DROP POLICY " + table + "_delete_tombstoned ON " + table,
				"CREATE POLICY " + table + "_delete_tombstoned ON " + table + " FOR DELETE TO nava_groupsync USING (" +
					"tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid AND EXISTS (SELECT 1 FROM groups g " +
					"WHERE g.tenant_id = " + table + ".tenant_id AND g.id = " + table + ".group_id))",
			} {
				if _, err := tx.Exec(ctx, s); err != nil {
					t.Fatalf("%s: %v", s, err)
				}
			}
			problems, _, err := catalog.PolicyProblems(ctx, tx)
			if err != nil {
				t.Fatal(err)
			}
			if len(problems) != 1 || !strings.Contains(problems[0], table+"_delete_tombstoned") {
				t.Errorf("problems = %q, want one naming %s_delete_tombstoned", problems, table)
			}
		})
	}
	// I1: nava_groupsync is on tenant_auditor_group_select's TO list.
	if !queryBool(t, super, `SELECT 'nava_groupsync' = ANY (roles::text[]) FROM pg_policies
		WHERE tablename = 'tenant_auditor_group' AND policyname = 'tenant_auditor_group_select'`) {
		t.Error("nava_groupsync is not in tenant_auditor_group_select's TO list")
	}
}

// rolledBack runs fn in a transaction on conn and rolls it back, returning
// fn's error.
func rolledBack(t *testing.T, conn *pgx.Conn, fn func(tx pgx.Tx) error) error {
	t.Helper()
	ctx := context.Background()
	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	return fn(tx)
}

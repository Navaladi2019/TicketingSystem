//go:build integration

package dbtest

import (
	"reflect"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// TICK-372 adversarial: the runbook repair runs as the superuser, so
// RLS does not scope it; only its WHERE clause does. Tenant B holds the same
// stale row as tenant A, under the same Keycloak id ("kc-old-<group>" is
// legal in both: groups is unique on (tenant_id, keycloak_group_id) only), plus
// a recorded pointer row and a member. Repairing A with either option must
// leave every one of B's identity rows byte-for-byte unchanged. TICK-530
// (ADR-004-E5 N3): the same, for the auditors group and tenant_auditor_group.
func TestTick372RunbookRepairCannotTouchOtherTenant(t *testing.T) {
	for _, g := range pointerGroups {
		for _, tc := range []struct {
			name  string
			block int
			tag   string
		}{{"option A update", blkA, "UPDATE 1"}, {"option B delete", blkB, "DELETE 1"}} {
			t.Run(g.name+" "+tc.name, func(t *testing.T) {
				migratedPlatform(t)
				super := superNava(t)
				ta, tb := uuid.MustParse(tenantA), uuid.MustParse(tenantB)
				staleGroup(t, super, ta, g)

				// Tenant B: active, with the group under the very same Keycloak
				// id as A's stale row, recorded and with a member.
				seedTenants(t, super, []string{tenantB}, nil)
				gb := g.id(tb)
				exec(t, super, `INSERT INTO groups (tenant_id, id, keycloak_group_id, path)
					VALUES ($1::uuid, $2, $3, '/t/' || $1::text || '/' || $4::text)`, tenantB, gb, "kc-old-"+g.name, g.name)
				exec(t, super, `INSERT INTO `+g.table+` (tenant_id, group_id) VALUES ($1::uuid, $2)`, tenantB, gb)
				pb := uuid.New()
				exec(t, super, `INSERT INTO principals (tenant_id, id, keycloak_user_id, enabled, synced_at) VALUES ($1::uuid, $2, 'kc-user-b', true, now())`, tenantB, pb)
				exec(t, super, `INSERT INTO group_members (tenant_id, group_id, principal_id) VALUES ($1::uuid, $2, $3)`, tenantB, gb, pb)

				snapshotB := tenantIdentitySnapshot(t, super, tenantB)
				before := snapshotB()

				kc := "kc-new-" + g.name
				blocks := runbookBlocks(t, tenantA, kc, g)
				if rows := runBlock(t, super, "confirm", blocks[blkConfirm]).counts(); !reflect.DeepEqual(rows, []int{1, 0, 0}) {
					t.Fatalf("confirm rows = %v, want [1 0 0]: the confirm step saw another tenant's rows", rows)
				}
				if tags := runBlock(t, super, "repair", blocks[tc.block]).tags; !reflect.DeepEqual(tags, []string{"BEGIN", tc.tag}) {
					t.Fatalf("repair tags = %v, want exactly %s: the repair reached beyond tenant A", tags, tc.tag)
				}
				runBlock(t, super, "commit", blocks[blkCommit])
				if after := snapshotB(); after != before {
					t.Fatalf("tenant B identity rows changed by tenant A's repair:\nbefore %s\nafter  %s", before, after)
				}

				// The check block for A must not report B's pointer either.
				if err := recordGroup(t, g, ta, kc); err != nil {
					t.Fatalf("retry record %s: %v", g.name, err)
				}
				if rows := runBlock(t, super, "check", blocks[blkCheck]).counts(); !reflect.DeepEqual(rows, []int{1}) {
					t.Fatalf("check rows = %v, want [1]", rows)
				}
				if after := snapshotB(); after != before {
					t.Fatalf("tenant B identity rows changed by tenant A's retry:\nbefore %s\nafter  %s", before, after)
				}
			})
		}
	}
}

// tenantIdentitySnapshot returns a function that renders tenant's groups,
// both pointer rows and memberships as one string, for a before/after check.
func tenantIdentitySnapshot(t *testing.T, super *pgx.Conn, tenant string) func() string {
	t.Helper()
	return func() string {
		return queryString(t, super, `SELECT concat_ws('|',
			(SELECT string_agg(concat_ws(',', id, keycloak_group_id, path, deleted_at), ';' ORDER BY id) FROM groups WHERE tenant_id = $1::uuid),
			(SELECT string_agg(group_id::text, ';') FROM tenant_admin_group WHERE tenant_id = $1::uuid),
			(SELECT string_agg(group_id::text, ';') FROM tenant_auditor_group WHERE tenant_id = $1::uuid),
			(SELECT string_agg(concat_ws(',', group_id, principal_id), ';' ORDER BY group_id, principal_id) FROM group_members WHERE tenant_id = $1::uuid))`, tenant)
	}
}

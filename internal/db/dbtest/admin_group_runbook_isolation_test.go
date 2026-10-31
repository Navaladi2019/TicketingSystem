//go:build integration

package dbtest

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/onboard"
)

// TICK-372 adversarial: the runbook repair runs as the owner or a superuser, so
// RLS does not scope it; only its WHERE clause does. Tenant B holds the same
// stale admins row as tenant A, under the same Keycloak id ("kc-old-admins" is
// legal in both: groups is unique on (tenant_id, keycloak_group_id) only), plus
// a recorded tenant_admin_group row and a member. Repairing A with either option
// must leave every one of B's identity rows byte-for-byte unchanged.
func TestTick372RunbookRepairCannotTouchOtherTenant(t *testing.T) {
	for _, tc := range []struct {
		name  string
		block int
		tag   string
	}{{"option A update", 1, "UPDATE 1"}, {"option B delete", 2, "DELETE 1"}} {
		t.Run(tc.name, func(t *testing.T) {
			migratedPlatform(t)
			super := superNava(t)
			ta, tb := uuid.MustParse(tenantA), uuid.MustParse(tenantB)
			staleAdminGroup(t, super, ta)

			// Tenant B: active, with its admins group under the very same
			// Keycloak id as A's stale row, recorded and with a member.
			seedTenants(t, super, []string{tenantB}, nil)
			gb := onboard.AdminGroupID(tb)
			exec(t, super, `INSERT INTO groups (tenant_id, id, keycloak_group_id, path)
				VALUES ($1::uuid, $2, 'kc-old-admins', '/t/' || $1::text || '/admins')`, tenantB, gb)
			exec(t, super, `INSERT INTO tenant_admin_group (tenant_id, group_id) VALUES ($1::uuid, $2)`, tenantB, gb)
			pb := uuid.New()
			exec(t, super, `INSERT INTO principals (tenant_id, id, keycloak_user_id, enabled, synced_at) VALUES ($1::uuid, $2, 'kc-user-b', true, now())`, tenantB, pb)
			exec(t, super, `INSERT INTO group_members (tenant_id, group_id, principal_id) VALUES ($1::uuid, $2, $3)`, tenantB, gb, pb)

			snapshotB := func() string {
				return queryString(t, super, `SELECT concat_ws('|',
					(SELECT string_agg(concat_ws(',', id, keycloak_group_id, path), ';' ORDER BY id) FROM groups WHERE tenant_id = $1::uuid),
					(SELECT string_agg(group_id::text, ';') FROM tenant_admin_group WHERE tenant_id = $1::uuid),
					(SELECT string_agg(concat_ws(',', group_id, principal_id), ';' ORDER BY group_id, principal_id) FROM group_members WHERE tenant_id = $1::uuid))`, tenantB)
			}
			before := snapshotB()

			blocks := runbookBlocks(t, tenantA, "kc-new-admins")
			rows, _ := runBlock(t, super, "confirm", blocks[0])
			if len(rows) != 3 || rows[0] != 1 || rows[1] != 0 || rows[2] != 0 {
				t.Fatalf("confirm rows = %v, want [1 0 0]: the confirm step saw another tenant's rows", rows)
			}
			_, tags := runBlock(t, super, "repair", blocks[tc.block])
			if !slicesContain(tags, tc.tag) {
				t.Fatalf("repair tags = %v, want exactly %s: the repair reached beyond tenant A", tags, tc.tag)
			}
			if after := snapshotB(); after != before {
				t.Fatalf("tenant B identity rows changed by tenant A's repair:\nbefore %s\nafter  %s", before, after)
			}

			// The check block for A must not report B's binding either.
			if err := onboardStore(t, 4).RecordAdminGroup(context.Background(), ta, "kc-new-admins"); err != nil {
				t.Fatalf("retry RecordAdminGroup: %v", err)
			}
			rows, _ = runBlock(t, super, "check", blocks[3])
			if len(rows) != 1 || rows[0] != 1 {
				t.Fatalf("check rows = %v, want [1]", rows)
			}
			if after := snapshotB(); after != before {
				t.Fatalf("tenant B identity rows changed by tenant A's retry:\nbefore %s\nafter  %s", before, after)
			}
		})
	}
}

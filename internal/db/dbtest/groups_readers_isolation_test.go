//go:build integration

package dbtest

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/onboard"
)

// Security review of TICK-536 (ADR-004-E5 D4, CI-E5-5): adversarial guards
// for the deleted_at filters in recordGroup and the nava_onboard SELECT
// (deleted_at) grant. The attacker holds onboarding for tenant A.

// The auditors path shares recordGroup: a tombstoned auditors group is
// refused and leaves no pointer row.
func TestE5SecRecordAuditorGroupRefusesTombstoned(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, nil, []string{tenantA})
	tid := uuid.MustParse(tenantA)
	exec(t, super, "INSERT INTO groups (tenant_id, id, keycloak_group_id, path) VALUES ($1::uuid, $2, 'kc-auditors', '/t/' || $1::text || '/auditors')",
		tenantA, onboard.AuditorGroupID(tid))
	exec(t, super, tombstoneGroup, tenantA, onboard.AuditorGroupID(tid))
	err := onboardStore(t, 4).RecordAuditorGroup(t.Context(), tid, "kc-auditors")
	if !errors.Is(err, onboard.ErrAdminGroupMissing) {
		t.Fatalf("RecordAuditorGroup = %v, want ErrAdminGroupMissing", err)
	}
	if got := queryString(t, super, "SELECT count(*)::text FROM tenant_auditor_group WHERE tenant_id = $1", tenantA); got != "0" {
		t.Errorf("tenant_auditor_group rows = %s, want 0", got)
	}
}

// The filter is tenant-scoped: B's tombstoned group with the same Keycloak id
// neither blocks A nor lends A its id, and A's live group does not rescue B.
func TestE5SecRecordAdminGroupTombstoneIsPerTenant(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, nil, []string{tenantA, tenantB})
	a, b := uuid.MustParse(tenantA), uuid.MustParse(tenantB)
	exec(t, super, insertAdminsGroup, tenantA, onboard.AdminGroupID(a))
	exec(t, super, insertAdminsGroup, tenantB, onboard.AdminGroupID(b))
	exec(t, super, tombstoneGroup, tenantB, onboard.AdminGroupID(b))
	st := onboardStore(t, 4)
	if err := st.RecordAdminGroup(t.Context(), a, "kc-admins"); err != nil {
		t.Fatalf("A with a live admins group: %v", err)
	}
	if got := queryString(t, super, "SELECT group_id::text FROM tenant_admin_group WHERE tenant_id = $1", tenantA); got != onboard.AdminGroupID(a).String() {
		t.Errorf("A's pointer = %s, want A's own group %s", got, onboard.AdminGroupID(a))
	}
	if err := st.RecordAdminGroup(t.Context(), b, "kc-admins"); !errors.Is(err, onboard.ErrAdminGroupMissing) {
		t.Fatalf("B with a tombstoned admins group: %v, want ErrAdminGroupMissing", err)
	}
	if got := queryString(t, super, "SELECT count(*)::text FROM tenant_admin_group WHERE tenant_id = $1", tenantB); got != "0" {
		t.Errorf("B's pointer rows = %s, want 0", got)
	}
}

// SELECT (deleted_at) widens no row set: nava_onboard still sees only its
// own provisioning tenant's groups, and no other column of groups.
func TestE5SecOnboardDeletedAtGrantStaysInTenant(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantB}, []string{tenantA})
	exec(t, super, insertAdminsGroup, tenantA, uuid.NewString())
	exec(t, super, insertAdminsGroup, tenantB, uuid.NewString())
	exec(t, super, "INSERT INTO tenants (id, state) VALUES ($1, 'provisioning')", "0190a000-0000-7000-8000-00000000000c")
	exec(t, super, insertAdminsGroup, "0190a000-0000-7000-8000-00000000000c", uuid.NewString())
	exec(t, super, "UPDATE groups SET deleted_at = now() WHERE tenant_id <> $1", tenantA)
	ob := asRole(t, "nava_onboard")
	count := func(ctxTenant string) int {
		var n int
		if err := inTenantTx(ob, ctxTenant, func(tx pgx.Tx) error {
			return tx.QueryRow(context.Background(), "SELECT count(*) FROM groups WHERE deleted_at IS NOT NULL OR deleted_at IS NULL").Scan(&n)
		}); err != nil {
			t.Fatalf("count from tenant '%s': %v", ctxTenant, err)
		}
		return n
	}
	if n := count(tenantA); n != 1 {
		t.Errorf("from A: %d groups visible, want A's 1", n)
	}
	if n := count(""); n != 0 {
		t.Errorf("no tenant set: %d groups visible, want 0", n)
	}
	if n := count(tenantB); n != 0 {
		t.Errorf("from active B: %d groups visible, want 0", n)
	}
	// Tombstones of other tenants are not an oracle: A's filtered probe
	// reads only A's rows.
	var tomb int
	if err := inTenantTx(ob, tenantA, func(tx pgx.Tx) error {
		return tx.QueryRow(context.Background(), "SELECT count(*) FROM groups WHERE deleted_at IS NOT NULL").Scan(&tomb)
	}); err != nil || tomb != 0 {
		t.Errorf("from A: %d tombstoned groups visible (%v), want 0", tomb, err)
	}
	for _, col := range []string{"path", "live", "*"} {
		_, err := execIn(ob, tenantA, "SELECT "+col+" FROM groups")
		wantSQLState(t, "nava_onboard SELECT "+col, err, "42501")
	}
}

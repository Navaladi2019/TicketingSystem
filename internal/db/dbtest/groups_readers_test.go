//go:build integration

package dbtest

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/identity/pgstore"
	"github.com/Navaladi2019/TicketingSystem/internal/onboard"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// ADR-004-E5 D4 (TICK-536): readers that must not see a removed group filter
// deleted_at IS NULL. D11 keeps a tombstoned group's group_members empty, so
// these tests plant a stale member row with the FK triggers off to prove the
// reader filter holds on its own (defence in depth).

// tombstoneWithStaleMember tombstones group tenantA/tenantA while leaving its
// member row in place, which only a superuser with triggers off can do.
func tombstoneWithStaleMember(t *testing.T) {
	t.Helper()
	super := superNava(t)
	exec(t, super, "SET session_replication_role = replica")
	exec(t, super, tombstoneGroup, tenantA, tenantA)
	exec(t, super, "SET session_replication_role = DEFAULT")
}

// AC1: the /me group path read drops a tombstoned group.
func TestE5GroupPathsOmitTombstonedGroup(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA}, nil)
	seedIdentity(t, super, tenantA)
	s := pgstore.New(appPool(t))
	tid := uuid.MustParse(tenantA)
	before, err := s.GroupPaths(t.Context(), tid, tid)
	if err != nil || len(before) != 1 {
		t.Fatalf("control: GroupPaths = %v, %v; want the live group", before, err)
	}
	tombstoneWithStaleMember(t)
	got, err := s.GroupPaths(t.Context(), tid, tid)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("GroupPaths returns tombstoned group: %v", got)
	}
}

// AC2 (CI-E5-5): a former member of the tombstoned admins group is forbidden.
// D11 empties the group's members first; principalSQL reads them unchanged.
func TestE5FormerAdminForbidden(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA}, nil)
	seedIdentity(t, super, tenantA)
	exec(t, super, insertAdminGroupRow, tenantA, tenantA)
	tid := uuid.MustParse(tenantA)
	pool := appPool(t)
	s := pgstore.New(pool)
	check := func() error {
		rec, err := s.Principal(t.Context(), tid, "kc-user")
		if err != nil {
			t.Fatal(err)
		}
		p := identity.Principal{TenantID: tid, PrincipalID: rec.ID, Groups: rec.Groups}
		return tenancy.InTenantTx(t.Context(), pool, tid, func(tx pgx.Tx) error {
			return authz.RequireTenantAdminTx(t.Context(), tx, p)
		})
	}
	if err := check(); err != nil {
		t.Fatalf("control: member of the live admins group: %v", err)
	}
	exec(t, super, "DELETE FROM group_members WHERE tenant_id = $1 AND group_id = $1", tenantA)
	exec(t, super, tombstoneGroup, tenantA, tenantA)
	if err := check(); !errors.Is(err, authz.ErrForbidden) {
		t.Fatalf("former admin: %v, want ErrForbidden", err)
	}
}

// AC3 (CI-E5-5): RecordAdminGroup against a tombstoned admins group.
func TestE5RecordAdminGroupRefusesTombstoned(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, nil, []string{tenantA})
	tid := uuid.MustParse(tenantA)
	exec(t, super, insertAdminsGroup, tenantA, onboard.AdminGroupID(tid))
	exec(t, super, tombstoneGroup, tenantA, onboard.AdminGroupID(tid))
	err := onboardStore(t, 4).RecordAdminGroup(t.Context(), tid, "kc-admins")
	if !errors.Is(err, onboard.ErrAdminGroupMissing) {
		t.Fatalf("RecordAdminGroup = %v, want ErrAdminGroupMissing", err)
	}
	if got := queryString(t, super, "SELECT count(*)::text FROM tenant_admin_group WHERE tenant_id = $1", tenantA); got != "0" {
		t.Errorf("tenant_admin_group rows = %s, want 0", got)
	}
}

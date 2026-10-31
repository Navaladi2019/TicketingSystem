//go:build integration

package dbtest

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/onboard"
)

// P3-04 (TICK-342): tenant_admin_group under FORCE RLS, and the helper that
// reads it. Seeded identity ids equal the tenant id, so tenant X's group is X.

const (
	insertAdminGroupRow = "INSERT INTO tenant_admin_group (tenant_id, group_id) VALUES ($1, $2)"
	insertAdminsGroup   = "INSERT INTO groups (tenant_id, id, keycloak_group_id, path) VALUES ($1::uuid, $2, 'kc-admins', '/t/' || $1::text || '/admins')"
)

func TestP304OnboardWritesOwnProvisioningTenantOnly(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantB}, []string{tenantA})
	onboard := asRole(t, "nava_onboard")
	gid := uuid.NewString()

	for _, ctxTenant := range []string{tenantB, ""} {
		_, err := execIn(onboard, ctxTenant, insertAdminsGroup, tenantA, gid)
		wantSQLState(t, "groups insert for A from tenant '"+ctxTenant+"'", err, "42501")
		_, err = execIn(onboard, ctxTenant, insertAdminGroupRow, tenantA, gid)
		wantSQLState(t, "tenant_admin_group insert for A from tenant '"+ctxTenant+"'", err, "42501")
	}
	// An active tenant's row cannot be planted by onboarding.
	_, err := execIn(onboard, tenantB, insertAdminsGroup, tenantB, gid)
	wantSQLState(t, "groups insert into an active tenant", err, "42501")
	_, err = execIn(onboard, tenantB, insertAdminGroupRow, tenantB, gid)
	wantSQLState(t, "tenant_admin_group insert into an active tenant", err, "42501")

	// The statements the store runs, in one transaction, twice (a retry).
	for i := 0; i < 2; i++ {
		err := inTenantTx(onboard, tenantA, func(tx pgx.Tx) error {
			if _, err := tx.Exec(context.Background(), insertAdminsGroup+" ON CONFLICT DO NOTHING", tenantA, gid); err != nil {
				return err
			}
			_, err := tx.Exec(context.Background(), insertAdminGroupRow+" ON CONFLICT DO NOTHING", tenantA, gid)
			return err
		})
		if err != nil {
			t.Fatalf("onboarding write %d: %v", i, err)
		}
	}
	if got := queryString(t, super, "SELECT count(*)::text FROM tenant_admin_group WHERE tenant_id = $1 AND group_id = $2", tenantA, gid); got != "1" {
		t.Errorf("rows after a retried write: %s, want 1", got)
	}
	if got := queryString(t, super, "SELECT count(*)::text FROM groups WHERE tenant_id = $1", tenantA); got != "1" {
		t.Errorf("groups after a retried write: %s, want 1", got)
	}
}

func TestP304ForeignKeyAndRuntimeRolesCannotWrite(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantB}, []string{tenantA})
	seedIdentity(t, super, tenantA, tenantB)

	// Tenant A's row naming tenant B's group: the composite FK refuses it.
	onboard := asRole(t, "nava_onboard")
	_, err := execIn(onboard, tenantA, insertAdminGroupRow, tenantA, tenantB)
	wantSQLState(t, "A's row naming B's group", err, "23503")
	if _, err = execIn(onboard, tenantA, insertAdminGroupRow, tenantA, tenantA); err != nil {
		t.Fatalf("A's row naming A's group: %v", err)
	}

	exec(t, super, "UPDATE tenants SET state = 'active' WHERE id = $1", tenantA)
	for _, role := range []string{"nava_app", "nava_worker", "nava_groupsync", "nava_onboard"} {
		conn := asRole(t, role)
		for what, sql := range map[string]string{
			"UPDATE": "UPDATE tenant_admin_group SET group_id = $2 WHERE tenant_id = $1",
			"DELETE": "DELETE FROM tenant_admin_group WHERE tenant_id = $1 AND $2::uuid IS NOT NULL",
		} {
			_, err := execIn(conn, tenantA, sql, tenantA, tenantA)
			wantSQLState(t, role+" "+what, err, "42501")
		}
	}
	for _, role := range []string{"nava_app", "nava_worker", "nava_groupsync"} {
		_, err := execIn(asRole(t, role), tenantA, insertAdminGroupRow, tenantB, tenantB)
		wantSQLState(t, role+" INSERT", err, "42501")
	}
	if got := queryString(t, super, "SELECT group_id::text FROM tenant_admin_group WHERE tenant_id = $1", tenantA); got != tenantA {
		t.Errorf("A's admin group = %s, want %s", got, tenantA)
	}
}

func TestP304RequireTenantAdminTx(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, nil)
	seedIdentity(t, super, tenantA, tenantB)
	// Tenant A has an admins row; tenant B has none.
	exec(t, super, insertAdminGroupRow, tenantA, tenantA)
	app := asRole(t, "nava_app")
	a, b := uuid.MustParse(tenantA), uuid.MustParse(tenantB)

	check := func(ctxTenant string, p identity.Principal) error {
		var got error
		if err := inTenantTx(app, ctxTenant, func(tx pgx.Tx) error {
			got = authz.RequireTenantAdminTx(context.Background(), tx, p)
			return nil
		}); err != nil {
			t.Fatal(err)
		}
		return got
	}

	if err := check(tenantA, identity.Principal{TenantID: a, Groups: []uuid.UUID{a}}); err != nil {
		t.Errorf("member of A's admin group: %v", err)
	}
	if err := check(tenantA, identity.Principal{TenantID: a, Groups: []uuid.UUID{uuid.New()}}); !errors.Is(err, authz.ErrForbidden) {
		t.Errorf("non-member: %v", err)
	}
	// Tenant B's admin principal calling inside tenant A's transaction: RLS
	// shows only A's rows, so the row for B is absent and it is refused.
	if err := check(tenantA, identity.Principal{TenantID: b, Groups: []uuid.UUID{a, b}}); !errors.Is(err, authz.ErrForbidden) {
		t.Errorf("tenant B principal in A's transaction: %v", err)
	}
	// Tenant B has no row: everyone is refused.
	if err := check(tenantB, identity.Principal{TenantID: b, Groups: []uuid.UUID{b}}); !errors.Is(err, authz.ErrForbidden) {
		t.Errorf("tenant without a row: %v", err)
	}
	// No tenant set in the transaction: nothing is visible, so refused.
	if err := check("", identity.Principal{TenantID: a, Groups: []uuid.UUID{a}}); !errors.Is(err, authz.ErrForbidden) {
		t.Errorf("no app.tenant_id: %v", err)
	}
}

// Moving another group to /t/<tenant>/admins, as group sync does on a
// Keycloak rename, grants nothing: the check compares ids only.
func TestP304RenamedGroupGrantsNothing(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA}, nil)
	seedIdentity(t, super, tenantA) // group A at /t/A/agents, principal A is its member
	other := uuid.NewString()
	exec(t, super, insertAdminsGroup, tenantA, other)
	exec(t, super, "INSERT INTO principals (tenant_id, id, keycloak_user_id, enabled, federated_links, synced_at) VALUES ($1, $2, 'kc-other', true, '[]', now())", tenantA, other)
	exec(t, super, "INSERT INTO group_members (tenant_id, group_id, principal_id) VALUES ($1, $2, $2)", tenantA, other)
	// The recorded admin group is group A; the group now called /admins is not.
	exec(t, super, insertAdminGroupRow, tenantA, tenantA)

	app := asRole(t, "nava_app")
	a := uuid.MustParse(tenantA)
	for who, want := range map[string]bool{tenantA: true, other: false} {
		var got error
		err := inTenantTx(app, tenantA, func(tx pgx.Tx) error {
			var groups []uuid.UUID
			rows, err := tx.Query(context.Background(), "SELECT group_id FROM group_members WHERE tenant_id = $1 AND principal_id = $2", tenantA, who)
			if err != nil {
				return err
			}
			for rows.Next() {
				var id uuid.UUID
				if err := rows.Scan(&id); err != nil {
					rows.Close()
					return err
				}
				groups = append(groups, id)
			}
			rows.Close()
			if err := rows.Err(); err != nil {
				return err
			}
			got = authz.RequireTenantAdminTx(context.Background(), tx, identity.Principal{TenantID: a, Groups: groups})
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		if (got == nil) != want {
			t.Errorf("principal %s admin = %v, want %v (err %v)", who, got == nil, want, got)
		}
	}
}

// Group sync can create the admins group before onboarding records it, under
// an id of its own. RecordAdminGroup must then point the row at that real id,
// and a re-run must finish onboarding.
func TestP304RecordAdminGroupUsesTheRealGroupID(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, nil, []string{tenantA})
	synced := uuid.NewString()
	exec(t, super, insertAdminsGroup, tenantA, synced)

	store := onboardStore(t, 4)
	tid := uuid.MustParse(tenantA)
	for i := 0; i < 2; i++ {
		if err := store.RecordAdminGroup(context.Background(), tid, "kc-admins"); err != nil {
			t.Fatalf("RecordAdminGroup run %d: %v", i, err)
		}
	}
	if got := queryString(t, super, "SELECT group_id::text FROM tenant_admin_group WHERE tenant_id = $1", tenantA); got != synced {
		t.Errorf("recorded group %s, want the synced group %s", got, synced)
	}
	if got := queryString(t, super, "SELECT count(*)::text FROM groups WHERE tenant_id = $1", tenantA); got != "1" {
		t.Errorf("groups = %s, want 1 (no second admins group)", got)
	}
	if err := store.Activate(context.Background(), tid); err != nil {
		t.Errorf("onboarding does not finish after the record: %v", err)
	}
}

// The admins group was deleted and re-created in Keycloak between two
// onboarding retries: the stale groups row holds AdminGroupID's primary key
// under the old Keycloak id, and the current id has no row. RecordAdminGroup
// must return ErrAdminGroupMissing and write no tenant_admin_group row; the
// tenant stays provisioning because onboarding stops before Activate.
func TestP304RecordAdminGroupErrorsOnStaleGroupRow(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, nil, []string{tenantA})
	tid := uuid.MustParse(tenantA)
	exec(t, super, `INSERT INTO groups (tenant_id, id, keycloak_group_id, path)
		VALUES ($1::uuid, $2, 'kc-old-admins', '/t/' || $1::text || '/admins')`, tenantA, onboard.AdminGroupID(tid))

	store := onboardStore(t, 4)
	err := store.RecordAdminGroup(context.Background(), tid, "kc-new-admins")
	if !errors.Is(err, onboard.ErrAdminGroupMissing) {
		t.Fatalf("RecordAdminGroup = %v, want ErrAdminGroupMissing", err)
	}
	if got := queryString(t, super, "SELECT count(*)::text FROM tenant_admin_group WHERE tenant_id = $1", tenantA); got != "0" {
		t.Errorf("tenant_admin_group rows = %s, want 0", got)
	}
	if got := queryString(t, super, "SELECT state FROM tenants WHERE id = $1", tenantA); got != "provisioning" {
		t.Errorf("tenant state = %s, want provisioning", got)
	}
}

// With no synced group, RecordAdminGroup writes the group and the row.
func TestP304RecordAdminGroupCreatesTheGroup(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, nil, []string{tenantA})
	store := onboardStore(t, 4)
	if err := store.RecordAdminGroup(context.Background(), uuid.MustParse(tenantA), "kc-admins"); err != nil {
		t.Fatal(err)
	}
	if got := queryBool(t, super, `SELECT EXISTS (SELECT 1 FROM tenant_admin_group a JOIN groups g
		ON (g.tenant_id, g.id) = (a.tenant_id, a.group_id) WHERE a.tenant_id = $1 AND g.keycloak_group_id = 'kc-admins')`, tenantA); !got {
		t.Error("tenant_admin_group does not name the group onboarding created")
	}
}

// The tenant's admins group going away removes the row (ON DELETE CASCADE).
func TestP304DeletingTheGroupRemovesTheRow(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA}, nil)
	seedIdentity(t, super, tenantA)
	exec(t, super, insertAdminGroupRow, tenantA, tenantA)
	sync := asRole(t, "nava_groupsync")
	if n, err := execIn(sync, tenantA, "DELETE FROM groups WHERE tenant_id = $1", tenantA); err != nil || n != 1 {
		t.Fatalf("nava_groupsync deletes the admins group: %d rows, %v", n, err)
	}
	if got := queryString(t, super, "SELECT count(*)::text FROM tenant_admin_group WHERE tenant_id = $1", tenantA); got != "0" {
		t.Errorf("rows after the group is deleted: %s, want 0", got)
	}
}

// TICK-373: Activate changes no row of a provisioning tenant without a
// tenant_admin_group row, returns ErrNoTenantAdmin, and the tenant stays
// provisioning. Another tenant's row does not count. Once the row exists,
// Activate succeeds, and a second Activate is ErrNotProvisioning.
func TestP304hActivateRequiresATenantAdminRow(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, nil, []string{tenantA, tenantB})
	seedIdentity(t, super, tenantB)
	exec(t, super, insertAdminGroupRow, tenantB, tenantB) // B has an admin, A has none

	store := onboardStore(t, 4)
	a := uuid.MustParse(tenantA)
	for i := 0; i < 2; i++ { // a retry is refused the same way
		if err := store.Activate(context.Background(), a); !errors.Is(err, onboard.ErrNoTenantAdmin) {
			t.Fatalf("Activate #%d of a tenant without an admins row = %v, want ErrNoTenantAdmin", i+1, err)
		}
	}
	if got := queryString(t, super, "SELECT state FROM tenants WHERE id = $1", tenantA); got != "provisioning" {
		t.Fatalf("tenant state = %s, want provisioning", got)
	}
	if got := queryString(t, super, "SELECT state FROM tenants WHERE id = $1", tenantB); got != "provisioning" {
		t.Errorf("tenant B state = %s, want provisioning (untouched)", got)
	}

	if err := store.RecordAdminGroup(context.Background(), a, "kc-admins"); err != nil {
		t.Fatal(err)
	}
	if err := store.Activate(context.Background(), a); err != nil {
		t.Fatalf("Activate after RecordAdminGroup: %v", err)
	}
	if got := queryString(t, super, "SELECT state FROM tenants WHERE id = $1", tenantA); got != "active" {
		t.Errorf("tenant state = %s, want active", got)
	}
	if err := store.Activate(context.Background(), a); !errors.Is(err, onboard.ErrNotProvisioning) {
		t.Errorf("second Activate = %v, want ErrNotProvisioning", err)
	}
	// A tenant that does not exist is not provisioning.
	if err := store.Activate(context.Background(), uuid.New()); !errors.Is(err, onboard.ErrNotProvisioning) {
		t.Errorf("Activate of an unknown tenant = %v, want ErrNotProvisioning", err)
	}
}

// TICK-373: tenant_admin_group_select_onboard shows nava_onboard only its own
// provisioning tenant's row, and only the tenant_id column.
func TestP304hOnboardReadsOnlyItsOwnProvisioningAdminRow(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA}, []string{tenantX, tenantY})
	seedIdentity(t, super, tenantA, tenantX, tenantY)
	for _, id := range []string{tenantA, tenantX, tenantY} {
		exec(t, super, insertAdminGroupRow, id, id)
	}
	onb := asRole(t, "nava_onboard")
	count := func(ctxTenant, sql string, args ...any) int {
		var n int
		err := inTenantTx(onb, ctxTenant, func(tx pgx.Tx) error {
			return tx.QueryRow(context.Background(), sql, args...).Scan(&n)
		})
		if err != nil {
			t.Fatalf("%s as tenant '%s': %v", sql, ctxTenant, err)
		}
		return n
	}
	const byID = "SELECT count(*)::int FROM tenant_admin_group WHERE tenant_id = $1"
	const all = "SELECT count(*)::int FROM tenant_admin_group"

	if n := count(tenantX, byID, tenantX); n != 1 {
		t.Errorf("own provisioning tenant's row: %d, want 1", n)
	}
	// Cross-tenant: Y's transaction sees nothing of X, by id or by scan.
	if n := count(tenantY, byID, tenantX); n != 0 {
		t.Errorf("tenant X's row from tenant Y: %d, want 0", n)
	}
	if n := count(tenantY, all); n != 1 {
		t.Errorf("rows visible to tenant Y: %d, want 1 (its own)", n)
	}
	if n := count("", all); n != 0 {
		t.Errorf("rows visible with no app.tenant_id: %d, want 0", n)
	}
	// An active tenant's row is out of reach, even its own.
	if n := count(tenantA, byID, tenantA); n != 0 {
		t.Errorf("active tenant's own row: %d, want 0", n)
	}
	// Column grant: tenant_id only.
	err := inTenantTx(onb, tenantX, func(tx pgx.Tx) error {
		var g string
		return tx.QueryRow(context.Background(), "SELECT group_id::text FROM tenant_admin_group WHERE tenant_id = $1", tenantX).Scan(&g)
	})
	wantSQLState(t, "nava_onboard SELECT group_id", err, "42501")
}

//go:build integration

package groupsync_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel/metric/noop"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/groupsync"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// P3-04 (TICK-342): the directory loses the tenant's admins group. The delete
// cascades to tenant_admin_group, sync still commits the rest of the cycle,
// it logs a WARN, and the admin check then denies.
func TestDeletedAdminGroupCascadesAndSyncStillCommits(t *testing.T) {
	super, pool := cluster(t)
	tid := uuid.New()
	seedTenant(t, super, tid, 1)
	dir := &fakeDir{snaps: map[uuid.UUID]identity.DirectoryChanges{}}
	var logs bytes.Buffer
	s, err := groupsync.New(pool, dir, noop.NewMeterProvider(),
		slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelInfo})), groupsync.Options{MaxOrgUnitDepth: 8})
	if err != nil {
		t.Fatal(err)
	}

	users := []identity.DirectoryUser{
		{KeycloakUserID: "u1", Email: "a@example.test", Enabled: true},
		{KeycloakUserID: "u2", Email: "b@example.test", Enabled: true},
	}
	dir.set(tid, identity.DirectoryChanges{
		Users: users,
		Groups: []identity.DirectoryGroup{
			{KeycloakGroupID: "g1", Path: path(tid, "/agents"), Name: "agents", MemberKeycloakUserIDs: []string{"u1", "u2"}},
			{KeycloakGroupID: "gadm", Path: path(tid, "/admins"), Name: "admins", MemberKeycloakUserIDs: []string{"u1"}},
		},
	})
	if r := once(t, s); r.Err != nil {
		t.Fatal(r.Err)
	}
	var adminID uuid.UUID
	if err := super.QueryRow(context.Background(), "SELECT id FROM groups WHERE tenant_id = $1 AND keycloak_group_id = 'gadm'", tid).Scan(&adminID); err != nil {
		t.Fatal(err)
	}
	if _, err := super.Exec(context.Background(), "INSERT INTO tenant_admin_group (tenant_id, group_id) VALUES ($1, $2)", tid, adminID); err != nil {
		t.Fatal(err)
	}

	// A cycle that keeps the admins group does not warn.
	if strings.Contains(logs.String(), "admins group was deleted") {
		t.Fatal("warned while the admins group exists")
	}

	// The admins group is deleted in Keycloak and u2 leaves g1.
	dir.set(tid, identity.DirectoryChanges{
		Users: users,
		Groups: []identity.DirectoryGroup{
			{KeycloakGroupID: "g1", Path: path(tid, "/agents"), Name: "agents", MemberKeycloakUserIDs: []string{"u1"}},
		},
	})
	if r := once(t, s); r.Err != nil {
		t.Fatalf("sync must commit when the admins group is deleted: %v", r.Err)
	}
	if n := count(t, super, "SELECT count(*) FROM groups WHERE tenant_id = $1 AND keycloak_group_id = 'gadm'", tid); n != 0 {
		t.Errorf("deleted admins group still mirrored")
	}
	if n := count(t, super, "SELECT count(*) FROM tenant_admin_group WHERE tenant_id = $1", tid); n != 0 {
		t.Errorf("tenant_admin_group rows after the cascade = %d, want 0", n)
	}
	if n := count(t, super, "SELECT count(*) FROM group_members WHERE tenant_id = $1", tid); n != 1 {
		t.Errorf("memberships = %d, want 1: the removal of u2 must land", n)
	}
	if !strings.Contains(logs.String(), "level=WARN") || !strings.Contains(logs.String(), "admins group was deleted") {
		t.Errorf("no WARN for the deleted admins group; log:\n%s", logs.String())
	}

	// Fail closed: a principal still carrying the old group id is refused.
	p := identity.Principal{TenantID: tid, Groups: []uuid.UUID{adminID}}
	var got error
	if err := tenancy.InTenantTx(context.Background(), pool, tid, func(tx pgx.Tx) error {
		got = authz.RequireTenantAdminTx(context.Background(), tx, p)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if !errors.Is(got, authz.ErrForbidden) {
		t.Errorf("RequireTenantAdminTx after the cascade = %v, want ErrForbidden", got)
	}
}

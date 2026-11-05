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
)

// P5-18 (TICK-434): tenant_auditor_group and authz.RequireAuditPermission.
// Seeded identity ids equal the tenant id, so tenant X's admins group is X.

func TestP518RequireAuditPermission(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA, tenantB}, nil)
	seedIdentity(t, super, tenantA, tenantB)
	a, b := uuid.MustParse(tenantA), uuid.MustParse(tenantB)
	auditors, lookalike, sub, bAuditors := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	for _, g := range []struct {
		tenant string
		id     uuid.UUID
		path   string
	}{
		{tenantA, auditors, "/t/" + tenantA + "/auditors"},
		{tenantA, lookalike, "/t/" + tenantA + "/ou/auditors"},
		{tenantA, sub, "/t/" + tenantA + "/auditors/night"},
		{tenantB, bAuditors, "/t/" + tenantB + "/auditors"},
	} {
		exec(t, super, "INSERT INTO groups (tenant_id, id, keycloak_group_id, path) VALUES ($1, $2::uuid, 'kc-'||$2::uuid::text, $3)", g.tenant, g.id, g.path)
	}
	// Tenant A has admin and auditor rows. Tenant B has only an auditors row.
	exec(t, super, insertAdminGroupRow, tenantA, tenantA)
	exec(t, super, "INSERT INTO tenant_auditor_group (tenant_id, group_id) VALUES ($1, $2)", tenantA, auditors)
	exec(t, super, "INSERT INTO tenant_auditor_group (tenant_id, group_id) VALUES ($1, $2)", tenantB, bAuditors)
	app := asRole(t, "nava_app")

	check := func(ctxTenant string, p identity.Principal, perm string) error {
		var got error
		if err := inTenantTx(app, ctxTenant, func(tx pgx.Tx) error {
			got = authz.RequireAuditPermission(context.Background(), tx, p, perm)
			return nil
		}); err != nil {
			t.Fatal(err)
		}
		return got
	}
	p := func(t uuid.UUID, g ...uuid.UUID) identity.Principal {
		return identity.Principal{TenantID: t, Groups: g}
	}

	for _, perm := range []string{authz.PermAuditRead, authz.PermAuditExport} {
		if err := check(tenantA, p(a, a), perm); err != nil {
			t.Errorf("%s admin: %v", perm, err)
		}
		if err := check(tenantA, p(a, auditors), perm); err != nil {
			t.Errorf("%s auditor: %v", perm, err)
		}
		for name, pr := range map[string]identity.Principal{
			"agent":                  p(a, uuid.New()),
			"no groups":              p(a),
			"lookalike path":         p(a, lookalike),
			"subgroup of auditors":   p(a, sub),
			"tenant B auditor in A":  p(b, bAuditors),
			"tenant B auditor, A id": p(a, bAuditors),
		} {
			if err := check(tenantA, pr, perm); !errors.Is(err, authz.ErrForbidden) {
				t.Errorf("%s %s: %v", perm, name, err)
			}
		}
	}
	// With no auditor row, the auditors group is denied.
	exec(t, super, "DELETE FROM tenant_auditor_group WHERE tenant_id = $1", tenantA)
	if err := check(tenantA, p(a, auditors), authz.PermAuditRead); !errors.Is(err, authz.ErrForbidden) {
		t.Errorf("missing row: %v", err)
	}
	// Other permissions and unknown ones never pass.
	if err := check(tenantA, p(a, a), "records.write"); !errors.Is(err, authz.ErrForbidden) {
		t.Errorf("other perm: %v", err)
	}
}

func TestP518AuditorRowIsOnboardOnlyAndSameTenant(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantB}, []string{tenantA})
	seedIdentity(t, super, tenantA, tenantB)
	const ins = "INSERT INTO tenant_auditor_group (tenant_id, group_id) VALUES ($1, $2)"
	onboard := asRole(t, "nava_onboard")
	_, err := execIn(onboard, tenantA, ins, tenantA, tenantB)
	wantSQLState(t, "A's row naming B's group", err, "23503")
	// Tenant B is active: onboarding may not write it.
	_, err = execIn(onboard, tenantB, ins, tenantB, tenantB)
	wantSQLState(t, "active tenant", err, "42501")
	if _, err = execIn(onboard, tenantA, ins, tenantA, tenantA); err != nil {
		t.Fatalf("A's row naming A's group: %v", err)
	}
	exec(t, super, "UPDATE tenants SET state = 'active' WHERE id = $1", tenantA)
	const update = "UPDATE tenant_auditor_group SET group_id = $2 WHERE tenant_id = $1"
	const del = "DELETE FROM tenant_auditor_group WHERE tenant_id = $1 AND $2::uuid IS NOT NULL"
	for _, role := range []string{"nava_app", "nava_worker", "nava_groupsync", "nava_onboard"} {
		conn := asRole(t, role)
		_, err := execIn(conn, tenantA, update, tenantA, tenantA)
		wantSQLState(t, role+" UPDATE", err, "42501")
		if role == "nava_groupsync" {
			continue // ADR-004-E5 D5: DELETE under a policy, below
		}
		_, err = execIn(conn, tenantA, del, tenantA, tenantA)
		wantSQLState(t, role+" DELETE", err, "42501")
	}
	// CI-E5-5b (ADR-004-E5 D5): nava_groupsync deletes the pointer only once
	// its group is tombstoned. While the group is live, 0 rows.
	sync := asRole(t, "nava_groupsync")
	if n, err := execIn(sync, tenantA, del, tenantA, tenantA); err != nil || n != 0 {
		t.Errorf("nava_groupsync DELETE of a live group's pointer: %d rows, %v; want 0, nil", n, err)
	}
	exec(t, super, "DELETE FROM group_members WHERE tenant_id = $1 AND group_id = $1", tenantA)
	exec(t, super, "UPDATE groups SET deleted_at = now() WHERE tenant_id = $1 AND id = $1", tenantA)
	if n, err := execIn(sync, tenantA, del, tenantA, tenantA); err != nil || n != 1 {
		t.Errorf("nava_groupsync DELETE of a tombstoned group's pointer: %d rows, %v; want 1, nil", n, err)
	}
	exec(t, super, "UPDATE groups SET deleted_at = NULL WHERE tenant_id = $1 AND id = $1", tenantA)
	exec(t, super, "INSERT INTO tenant_auditor_group (tenant_id, group_id) VALUES ($1, $1)", tenantA)
	for _, role := range []string{"nava_app", "nava_worker", "nava_groupsync"} {
		_, err := execIn(asRole(t, role), tenantA, ins, tenantB, tenantB)
		wantSQLState(t, role+" INSERT", err, "42501")
	}
}

func TestP518RecordAuditorGroupIsIdempotent(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, nil, []string{tenantA})
	st := onboardStore(t, 4)
	for range 2 {
		if err := st.RecordAuditorGroup(context.Background(), uuid.MustParse(tenantA), "kc-auditors-a"); err != nil {
			t.Fatal(err)
		}
	}
	if got := queryString(t, super, "SELECT count(*)::text FROM tenant_auditor_group WHERE tenant_id = $1", tenantA); got != "1" {
		t.Errorf("rows = %s, want 1", got)
	}
	if got := queryString(t, super, "SELECT path FROM groups WHERE tenant_id = $1 AND keycloak_group_id = 'kc-auditors-a'", tenantA); got != "/t/"+tenantA+"/auditors" {
		t.Errorf("path = %s", got)
	}
}

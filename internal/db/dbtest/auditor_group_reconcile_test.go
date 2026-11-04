//go:build integration

package dbtest

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/onboard"
)

// TICK-567: the upgrade reconcile (onboard.ReconcileAuditorGroup) on the real
// nava_onboard policies of TICK-568. Tenants A and B are active and have no
// auditors group, as a tenant activated before S14.

type recordingDir struct{ ensured []uuid.UUID }

func (d *recordingDir) EnsureAuditorGroup(_ context.Context, id uuid.UUID) (string, error) {
	d.ensured = append(d.ensured, id)
	return "kc-auditors-" + id.String(), nil
}

// A reconcile for tenant A never creates or reads tenant B's group or row.
func TestTick567ReconcileForAOnlyTouchesA(t *testing.T) {
	super := activeAB(t)
	a, b := uuid.MustParse(tenantA), uuid.MustParse(tenantB)
	dir := &recordingDir{}
	created, err := onboard.ReconcileAuditorGroup(t.Context(), onboardStore(t, 4), dir, a)
	if err != nil || !created {
		t.Fatalf("reconcile A = %v, %v", created, err)
	}
	if len(dir.ensured) != 1 || dir.ensured[0] != a {
		t.Errorf("Keycloak was asked for %v, want only A", dir.ensured)
	}
	if got := queryString(t, super, "SELECT count(*)::text FROM groups WHERE tenant_id = $1 AND path LIKE '%/auditors'", tenantB); got != "0" {
		t.Errorf("B's auditors groups = %s, want 0", got)
	}
	if got := queryString(t, super, "SELECT count(*)::text FROM tenant_auditor_group WHERE tenant_id = $1", tenantB); got != "0" {
		t.Errorf("B's pointer rows = %s, want 0", got)
	}
	if got := queryString(t, super, "SELECT group_id::text FROM tenant_auditor_group WHERE tenant_id = $1", tenantA); got != onboard.AuditorGroupID(a).String() {
		t.Errorf("A's pointer = %s", got)
	}
	// The group starts empty: no membership row of any group.
	if got := queryString(t, super, "SELECT count(*)::text FROM group_members WHERE tenant_id IN ($1, $2) AND group_id = $3", tenantA, tenantB, onboard.AuditorGroupID(a)); got != "0" {
		t.Errorf("auditors members = %s, want 0", got)
	}
	// B reconciled afterwards gets its own.
	if _, err := onboard.ReconcileAuditorGroup(t.Context(), onboardStore(t, 4), dir, b); err != nil {
		t.Fatalf("reconcile B: %v", err)
	}
	if got := queryString(t, super, "SELECT group_id::text FROM tenant_auditor_group WHERE tenant_id = $1", tenantB); got != onboard.AuditorGroupID(b).String() {
		t.Errorf("B's pointer = %s", got)
	}
}

// A second run changes nothing and makes no Keycloak call.
func TestTick567ReconcileTwiceChangesNothing(t *testing.T) {
	super := activeAB(t)
	a := uuid.MustParse(tenantA)
	dir := &recordingDir{}
	st := onboardStore(t, 4)
	if _, err := onboard.ReconcileAuditorGroup(t.Context(), st, dir, a); err != nil {
		t.Fatal(err)
	}
	before := queryString(t, super, "SELECT (SELECT count(*) FROM groups WHERE tenant_id = $1)::text || '/' || (SELECT count(*) FROM tenant_auditor_group WHERE tenant_id = $1)::text", tenantA)
	created, err := onboard.ReconcileAuditorGroup(t.Context(), st, dir, a)
	if err != nil || created {
		t.Fatalf("second run = %v, %v; want a no-op", created, err)
	}
	if after := queryString(t, super, "SELECT (SELECT count(*) FROM groups WHERE tenant_id = $1)::text || '/' || (SELECT count(*) FROM tenant_auditor_group WHERE tenant_id = $1)::text", tenantA); after != before {
		t.Errorf("rows %s -> %s", before, after)
	}
	if len(dir.ensured) != 1 {
		t.Errorf("Keycloak calls = %d, want 1", len(dir.ensured))
	}
}

// A provisioning or unknown tenant is refused before Keycloak is called.
func TestTick567ReconcileRefusesProvisioningTenant(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, nil, []string{tenantA})
	dir := &recordingDir{}
	err := func() error {
		_, err := onboard.ReconcileAuditorGroup(t.Context(), onboardStore(t, 4), dir, uuid.MustParse(tenantA))
		return err
	}()
	if !errors.Is(err, onboard.ErrReconcileNotActive) || len(dir.ensured) != 0 {
		t.Errorf("err = %v, keycloak calls = %d", err, len(dir.ensured))
	}
}

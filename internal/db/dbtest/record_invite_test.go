//go:build integration

package dbtest

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/identity/keycloak"
)

// TICK-252: onboard.PGStore.RecordInvite as the real nava_onboard role. The
// upsert names only expires_at; INSERT 0 0 (a consumed invite) is an error.
func TestT252RecordInvite(t *testing.T) {
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, nil, []string{tenantX})
	s := onboardStore(t, 4)
	id := uuid.MustParse(tenantX)
	exp := time.Now().Add(24 * time.Hour)

	if err := s.RecordInvite(t.Context(), id, inviteEmail, exp); err != nil {
		t.Fatalf("first invite: %v", err)
	}
	if err := s.RecordInvite(t.Context(), id, inviteEmail, exp.Add(24*time.Hour)); err != nil {
		t.Fatalf("re-invite of a pending invite: %v", err)
	}
	if got := queryString(t, super, "SELECT (expires_at > now() + interval '40 hours')::text FROM pending_invite WHERE tenant_id = $1", tenantX); got != "true" {
		t.Error("re-invite did not extend expires_at")
	}

	// Only group sync consumes.
	exec(t, super, "UPDATE pending_invite SET consumed_at = now() WHERE tenant_id = $1", tenantX)
	if err := s.RecordInvite(t.Context(), id, inviteEmail, exp.Add(72*time.Hour)); !errors.Is(err, keycloak.ErrInviteConsumed) {
		t.Fatalf("re-invite of a consumed invite: %v, want ErrInviteConsumed", err)
	}
	if got := queryString(t, super, "SELECT (consumed_at IS NOT NULL AND expires_at < now() + interval '40 hours')::text FROM pending_invite WHERE tenant_id = $1", tenantX); got != "true" {
		t.Error("the consumed invite was reopened or extended")
	}

	// An active tenant takes no new invite.
	exec(t, super, "UPDATE tenants SET state = 'active' WHERE id = $1", tenantX)
	if err := s.RecordInvite(t.Context(), id, "late@tenant.test", exp); err == nil || errors.Is(err, keycloak.ErrInviteConsumed) {
		t.Errorf("invite in an active tenant: %v, want a policy error", err)
	}
}

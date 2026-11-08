//go:build integration

package groupsync

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/identity"
)

// Test seams for the fault-injection sweeps (package groupsync_test cannot
// reach the unexported apply and placeInvitees).

// ApplyForTest runs apply in tx.
func ApplyForTest(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID, c identity.DirectoryChanges, now time.Time, depth int) error {
	_, err := apply(ctx, tx, tenantID, c, now, depth)
	return err
}

// PlaceInviteesForTest runs placeInvitees in tx.
func (s *Syncer) PlaceInviteesForTest(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID, p InviteePlacer) error {
	return s.placeInvitees(ctx, tx, tenantID, p)
}

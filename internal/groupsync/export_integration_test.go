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

// RebuildClosureForTest runs the rebuild group sync runs after a tree write
// (rebuildClosure with treeChanged), in tx, under tx's tenant. It lets a test
// rebuild over a tree state no snapshot produces: a tombstoned unit with a
// live child (a snapshot that drops a parent but keeps its child is refused
// with ErrBadSnapshot). It returns the closure's row count.
func RebuildClosureForTest(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID) (int, error) {
	n, _, err := rebuildClosure(ctx, tx, tenantID, true)
	return n, err
}

// SetCountLiveForTest replaces the live org-unit count (a metric read) so a
// test can make it fail.
func (s *Syncer) SetCountLiveForTest(f func(context.Context, pgx.Tx, uuid.UUID) (int, error)) {
	s.countLive = f
}

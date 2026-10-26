package groupsync

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// InviteePlacer is the optional part of a directory that places an invited
// user in the tenant's group after their first login (TICK-252). The Keycloak
// Directory implements it; a directory that does not leaves invites pending.
type InviteePlacer interface {
	// PlaceInvitee places the user with this email in the tenant's group if
	// every placement condition holds, calling consume once they do, and
	// reports whether it placed them. bound are the tenant's bound IdP aliases.
	PlaceInvitee(ctx context.Context, tenantID uuid.UUID, email string, bound []string,
		consume func(context.Context) (bool, error)) (bool, error)
}

// maxInvitesPerCycle bounds the Keycloak calls one tenant's cycle spends on
// pending invites.
const maxInvitesPerCycle = 100

const (
	boundSQL   = `SELECT idp_aliases FROM tenant_identity_binding WHERE tenant_id = $1::uuid`
	pendingSQL = `SELECT email FROM pending_invite
		WHERE tenant_id = $1::uuid AND consumed_at IS NULL AND expires_at > now()
		ORDER BY created_at LIMIT $2`
	// consumeSQL is single use: of two racing cycles only one gets a row.
	consumeSQL = `UPDATE pending_invite SET consumed_at = now()
		WHERE tenant_id = $1::uuid AND email = $2 AND consumed_at IS NULL AND expires_at > now()
		RETURNING email`
)

// placeInvitees places the tenant's invited users who have signed in. tx is
// the tenant's transaction (app.tenant_id set), so the invite reads and the
// consume run under row-level security. Each invite runs in a savepoint: a
// failure rolls back that invite's consume and is logged, and the rest of the
// tenant still syncs. Only a failure of the transaction itself is returned.
func (s *Syncer) placeInvitees(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID, p InviteePlacer) error {
	rows, err := tx.Query(ctx, pendingSQL, tenantID.String(), maxInvitesPerCycle)
	if err != nil {
		return fmt.Errorf("read pending invites: %w", err)
	}
	var emails []string
	for rows.Next() {
		var e string
		if err := rows.Scan(&e); err != nil {
			rows.Close()
			return fmt.Errorf("read pending invites: %w", err)
		}
		emails = append(emails, e)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return fmt.Errorf("read pending invites: %w", err)
	}
	if len(emails) == 0 {
		return nil
	}
	var bound []string
	brows, err := tx.Query(ctx, boundSQL, tenantID.String())
	if err != nil {
		return fmt.Errorf("read bound idps: %w", err)
	}
	for brows.Next() {
		var a []string
		if err := brows.Scan(&a); err != nil {
			brows.Close()
			return fmt.Errorf("read bound idps: %w", err)
		}
		bound = append(bound, a...)
	}
	brows.Close()
	if err := brows.Err(); err != nil {
		return fmt.Errorf("read bound idps: %w", err)
	}

	for _, email := range emails {
		sp, err := tx.Begin(ctx)
		if err != nil {
			return fmt.Errorf("invite savepoint: %w", err)
		}
		placed, err := p.PlaceInvitee(ctx, tenantID, email, bound, func(ctx context.Context) (bool, error) {
			var got string
			err := sp.QueryRow(ctx, consumeSQL, tenantID.String(), email).Scan(&got)
			if errors.Is(err, pgx.ErrNoRows) {
				return false, nil
			}
			return err == nil, err
		})
		if err != nil || !placed {
			_ = sp.Rollback(ctx)
			if err != nil {
				// The email is not logged: it names a person.
				s.log.WarnContext(ctx, "group sync: could not place an invited user",
					slog.String("tenant_id", tenantID.String()), slog.Any("error", err))
			}
			continue
		}
		if err := sp.Commit(ctx); err != nil {
			return fmt.Errorf("invite savepoint: %w", err)
		}
		s.log.InfoContext(ctx, "group sync: placed an invited user", slog.String("tenant_id", tenantID.String()))
	}
	return nil
}

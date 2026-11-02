package authz

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/identity"
)

// The audit permissions (ADR-001-A6, D-5-4). No other permission implies
// either of them (R8).
const (
	PermAuditRead   = "audit.read"
	PermAuditExport = "audit.export"
)

// RequireAuditPermission returns nil when perm is audit.read or audit.export
// and p is a tenant admin or a member of the tenant's auditors group. Both
// groups are read by platform id inside the caller's InTenantTx, so row-level
// security scopes them to the tenant of the transaction; a principal of
// another tenant finds no row and is refused. Membership is direct, as for
// tenant admins: a member of a subgroup of auditors is not an auditor. A
// missing row, an unknown perm or an empty Groups denies (ErrForbidden); any
// other error is wrapped, and the caller answers 500.
func RequireAuditPermission(ctx context.Context, tx pgx.Tx, p identity.Principal, perm string) error {
	if perm != PermAuditRead && perm != PermAuditExport {
		return ErrForbidden
	}
	if err := RequireTenantAdminTx(ctx, tx, p); !errors.Is(err, ErrForbidden) {
		return err // nil (admin) or a store failure
	}
	var id uuid.UUID
	err := tx.QueryRow(ctx, "SELECT group_id FROM tenant_auditor_group WHERE tenant_id = $1", p.TenantID).Scan(&id)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return ErrForbidden
	case err != nil:
		return fmt.Errorf("authz: read auditor group: %w", err)
	case id == uuid.Nil, !slices.Contains(p.Groups, id):
		return ErrForbidden
	}
	return nil
}

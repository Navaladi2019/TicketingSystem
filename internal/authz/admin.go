// Package authz holds the platform's authorization checks. The tenant admin
// check compares a principal's platform group ids with the id onboarding
// recorded in tenant_admin_group. It never reads a group path or name, and
// never a Keycloak claim or role (ADR-002 section 1.4, CI-11).
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

var (
	// ErrForbidden means the principal is not an admin of its tenant.
	ErrForbidden = errors.New("authz: forbidden")
	// ErrNoAdminGroup means the tenant has no tenant_admin_group row. The helpers
	// map it to ErrForbidden (fail closed); a store returns it as is.
	ErrNoAdminGroup = errors.New("authz: tenant has no admin group")
)

// AdminGroupStore reads the tenant's admin group id.
//
// TODO(TODO-S8-01): tenants that predate tenant_admin_group have no row and
// are denied; back-fill the dev tenants.
type AdminGroupStore interface {
	// AdminGroupID reads tenant_admin_group; ErrNoAdminGroup if no row.
	AdminGroupID(ctx context.Context, tenantID uuid.UUID) (uuid.UUID, error)
}

// RequireTenantAdmin returns nil when p belongs to its tenant's admin group,
// ErrForbidden when it does not or the tenant has no admin group, and any
// other store error wrapped (the caller answers 500, never 403, for it).
// Callers that already hold a transaction use RequireTenantAdminTx.
func RequireTenantAdmin(ctx context.Context, store AdminGroupStore, p identity.Principal) error {
	id, err := store.AdminGroupID(ctx, p.TenantID)
	return decide(id, err, p)
}

// RequireTenantAdminTx is RequireTenantAdmin inside the caller's InTenantTx:
// the read runs under that transaction's app.tenant_id, so row-level security
// scopes it to the tenant of the transaction. A principal of another tenant
// finds no row there and is refused.
func RequireTenantAdminTx(ctx context.Context, tx pgx.Tx, p identity.Principal) error {
	var id uuid.UUID
	err := tx.QueryRow(ctx, "SELECT group_id FROM tenant_admin_group WHERE tenant_id = $1", p.TenantID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		err = ErrNoAdminGroup
	}
	return decide(id, err, p)
}

func decide(adminGroup uuid.UUID, err error, p identity.Principal) error {
	switch {
	case errors.Is(err, ErrNoAdminGroup):
		return ErrForbidden
	case err != nil:
		return fmt.Errorf("authz: read admin group: %w", err)
	case adminGroup == uuid.Nil, !slices.Contains(p.Groups, adminGroup):
		return ErrForbidden
	}
	return nil
}

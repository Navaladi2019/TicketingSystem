package onboard

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/internal/identity/keycloak"
	"github.com/Navaladi2019/TicketingSystem/internal/identity/pgstore"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// Registry is the keycloak.Registry of the onboarding directory. Onboarding
// only invites the first admin, so it needs the tenant's bound IdP aliases and
// email domains; it holds no user or group ids. The pool must log in as
// nava_app, and every read runs in InTenantTx.
type Registry struct {
	*pgstore.Store
	pool *pgxpool.Pool
}

var _ keycloak.Registry = (*Registry)(nil)

// NewRegistry returns a Registry on the nava_app pool.
func NewRegistry(pool *pgxpool.Pool) *Registry {
	return &Registry{Store: pgstore.New(pool), pool: pool}
}

// UserKeycloakID implements keycloak.Registry; onboarding holds no users.
func (*Registry) UserKeycloakID(context.Context, uuid.UUID, uuid.UUID) (string, error) {
	return "", keycloak.ErrRegistryNotFound
}

// GroupKeycloakID implements keycloak.Registry; onboarding holds no groups.
func (*Registry) GroupKeycloakID(context.Context, uuid.UUID, uuid.UUID) (string, error) {
	return "", keycloak.ErrRegistryNotFound
}

// Binding implements BindingReader: the (issuer, org id) the tenant is bound
// to, read in InTenantTx as nava_app. binding_lookup is USING (true) for
// nava_app, so the WHERE clause on tenant_id scopes the read, not an RLS policy.
//
// It assumes one binding row per tenant. The primary key (tenant_id, issuer,
// org_id) allows several and QueryRow has no ORDER BY, so with more than one
// row it reads an arbitrary one; Bind then compares against it and fails
// closed with a false ErrConflict. Revisit trigger: if a second binding writer
// (re-binding, IdP migration) is ever designed, that design must add
// UNIQUE (tenant_id) or make Binding fail on more than one row, and the check
// becomes "a row with exactly (tenant, issuer, org) exists".
func (r *Registry) Binding(ctx context.Context, tenantID uuid.UUID) (issuer, orgID string, found bool, err error) {
	err = tenancy.InTenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		e := tx.QueryRow(ctx, "SELECT issuer, org_id FROM tenant_identity_binding WHERE tenant_id = $1", tenantID).Scan(&issuer, &orgID)
		if errors.Is(e, pgx.ErrNoRows) {
			return nil
		}
		found = e == nil
		return e
	})
	return issuer, orgID, found, err
}

// BoundIdPAliases implements keycloak.Registry.
func (r *Registry) BoundIdPAliases(ctx context.Context, tenantID uuid.UUID) ([]string, error) {
	var out []string
	err := tenancy.InTenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, "SELECT idp_aliases FROM tenant_identity_binding WHERE tenant_id = $1", tenantID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var a []string
			if err := rows.Scan(&a); err != nil {
				return err
			}
			out = append(out, a...)
		}
		return rows.Err()
	})
	return out, err
}

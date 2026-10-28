// Package pgstore is the PostgreSQL identity.Store: the rows tenant
// resolution reads (ADR-001 A.1 steps 3-4, PLAN-001 T17). It runs as
// nava_app, which may only SELECT them.
//
// The binding lookup runs before any tenant is known, through the
// allow-listed binding_lookup policy. The bound tenant's state is read right
// after it, inside tenancy.InTenantTx for that tenant (tenants_tenant lets
// nava_app see only that row). The principal lookup runs inside
// tenancy.InTenantTx for the bound tenant, so row-level security holds it to
// that tenant even though the query names the tenant too.
package pgstore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// bindingSQL has no LIMIT: UNIQUE (issuer, org_id) allows one row, and a
// second one is an error, not a choice (A.1 step 4).
const bindingSQL = `SELECT tenant_id, idp_aliases FROM tenant_identity_binding
WHERE issuer = $1 AND org_id = $2`

// tenantStateSQL reads the bound tenant's state by primary key, inside
// InTenantTx(tenant_id) so tenants_tenant lets the row through.
const tenantStateSQL = `SELECT state FROM tenants WHERE id = $1`

// principalSQL is the statement measured in docs/data/identity.md (R1).
const principalSQL = `SELECT p.id, p.enabled, p.federated_links,
       ARRAY(SELECT gm.group_id FROM group_members gm
             WHERE gm.tenant_id = p.tenant_id AND gm.principal_id = p.id
             ORDER BY gm.group_id) AS groups
FROM principals p
WHERE p.tenant_id = $1 AND p.keycloak_user_id = $2`

// emailDomainSQL is the statement measured in docs/data/identity.md (T18a):
// an index-only scan of tenant_email_domain_pkey.
const emailDomainSQL = `SELECT EXISTS (SELECT 1 FROM tenant_email_domain
WHERE tenant_id = $1 AND domain = $2)`

// groupPathsSQL reads the caller's groups by principal, never by group id
// (ADR-002 section 3). Every predicate and join key names tenant_id.
const groupPathsSQL = `SELECT g.id, g.path
FROM group_members gm
JOIN groups g ON g.tenant_id = gm.tenant_id AND g.id = gm.group_id
WHERE gm.tenant_id = $1 AND gm.principal_id = $2
ORDER BY g.id`

// Store reads the identity rows through pool, which must log in as nava_app.
type Store struct {
	pool *pgxpool.Pool
}

var (
	_ identity.Store           = (*Store)(nil)
	_ identity.GroupPathReader = (*Store)(nil)
)

// New returns a Store on pool.
func New(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

// Binding implements identity.Store.
func (s *Store) Binding(ctx context.Context, issuer, orgID string) (identity.Binding, error) {
	rows, err := s.pool.Query(ctx, bindingSQL, issuer, orgID)
	if err != nil {
		return identity.Binding{}, fmt.Errorf("pgstore: binding: %w", err)
	}
	found, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (identity.Binding, error) {
		var b identity.Binding
		err := r.Scan(&b.TenantID, &b.IdPAliases)
		return b, err
	})
	if err != nil {
		return identity.Binding{}, fmt.Errorf("pgstore: binding: %w", err)
	}
	switch len(found) {
	case 0:
		return identity.Binding{}, identity.ErrNoBinding
	case 1:
		b := found[0]
		if b.TenantID == uuid.Nil {
			return identity.Binding{}, errors.New("pgstore: binding has a nil tenant")
		}
		err := tenancy.InTenantTx(ctx, s.pool, b.TenantID, func(tx pgx.Tx) error {
			return tx.QueryRow(ctx, tenantStateSQL, b.TenantID).Scan(&b.TenantState)
		})
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			// tenant_identity_binding_tenant_fkey makes this a broken database.
			return identity.Binding{}, errors.New("pgstore: binding's tenant row not found")
		case err != nil:
			return identity.Binding{}, fmt.Errorf("pgstore: tenant state: %w", err)
		}
		return b, nil
	}
	return identity.Binding{}, fmt.Errorf("pgstore: %d bindings for one (issuer, org_id)", len(found))
}

// link is one element of principals.federated_links, as the Keycloak admin
// API returns it; principals_federated_links_check guarantees the field.
type link struct {
	IdentityProvider string `json:"identityProvider"`
}

// Principal implements identity.Store.
func (s *Store) Principal(ctx context.Context, tenantID uuid.UUID, subject string) (identity.PrincipalRecord, error) {
	var (
		rec   identity.PrincipalRecord
		links []byte
	)
	err := tenancy.InTenantTx(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, principalSQL, tenantID, subject).Scan(&rec.ID, &rec.Enabled, &links, &rec.Groups)
	})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return identity.PrincipalRecord{}, identity.ErrNoPrincipal
	case err != nil:
		return identity.PrincipalRecord{}, fmt.Errorf("pgstore: principal: %w", err)
	}
	// NULL is "unknown": LinksKnown stays false and the resolver refuses.
	if links != nil {
		var ls []link
		if err := json.Unmarshal(links, &ls); err != nil {
			return identity.PrincipalRecord{}, errors.New("pgstore: principal: federated_links is not an array of links")
		}
		rec.LinksKnown = true
		rec.FederatedIdPs = make([]string, len(ls))
		for i, l := range ls {
			rec.FederatedIdPs[i] = l.IdentityProvider
		}
	}
	if rec.Groups == nil {
		rec.Groups = []uuid.UUID{}
	}
	return rec, nil
}

// TenantOwnsEmailDomain implements keycloak.Registry's method of that name
// (T18a). It reads inside tenancy.InTenantTx, so another tenant's domain is
// no row and reads as false, the same as a domain nobody owns: the answer
// never tells the caller that someone else owns it.
func (s *Store) TenantOwnsEmailDomain(ctx context.Context, tenantID uuid.UUID, domain string) (bool, error) {
	var owns bool
	err := tenancy.InTenantTx(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, emailDomainSQL, tenantID, strings.ToLower(domain)).Scan(&owns)
	})
	if err != nil {
		return false, fmt.Errorf("pgstore: email domain: %w", err)
	}
	return owns, nil
}

// GroupPaths implements identity.GroupPathReader. It returns the raw
// groups.path, prefix included, for the groups principalID belongs to in
// tenantID.
func (s *Store) GroupPaths(ctx context.Context, tenantID, principalID uuid.UUID) ([]identity.GroupPath, error) {
	var out []identity.GroupPath
	err := tenancy.InTenantTx(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, groupPathsSQL, tenantID, principalID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var gp identity.GroupPath
			if err := rows.Scan(&gp.ID, &gp.Path); err != nil {
				return err
			}
			out = append(out, gp)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, fmt.Errorf("pgstore: group paths: %w", err)
	}
	return out, nil
}

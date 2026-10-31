package onboard

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/identity/keycloak"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// PGStore is the Store on PostgreSQL. Its pool must log in as nava_onboard.
// Every statement runs inside tenancy.InTenantTx(tenantID), with the tenant id
// the service derived on the server, so nava_onboard reads and writes only
// that tenant's rows (ADR-001-E3).
type PGStore struct {
	pool *pgxpool.Pool
	// sem bounds concurrent onboardings. It is per process, like the pool it
	// protects: two nava-onboard processes each hold their own pool and their
	// own sem, and the advisory lock is what serialises one tenant across them.
	// Each holds one connection for its lock and takes a second for every
	// statement transaction, so at most MaxConns/2 may run at once; more would
	// each hold a lock connection and wait for a second one that never frees.
	sem      chan struct{}
	bindings BindingReader
}

var _ Store = (*PGStore)(nil)

// NewPGStore returns a PGStore on pool, which needs at least two connections.
// bindings reads as nava_app (onboard.Registry).
func NewPGStore(pool *pgxpool.Pool, bindings BindingReader) (*PGStore, error) {
	if bindings == nil {
		return nil, errors.New("onboard: a binding reader is required")
	}
	n := int(pool.Config().MaxConns) / 2
	if n < 1 {
		return nil, errors.New("onboard: the nava_onboard pool needs at least 2 connections")
	}
	return &PGStore{pool: pool, sem: make(chan struct{}, n), bindings: bindings}, nil
}

// Lock holds a transaction-scoped advisory lock keyed on the tenant until
// release is called. The lock lives in a transaction that stays open, so it
// is held on one backend even behind PgBouncer's transaction mode, and a
// crashed process releases it with its connection.
func (s *PGStore) Lock(ctx context.Context, tenantID uuid.UUID) (func(), error) {
	select {
	case s.sem <- struct{}{}:
	case <-ctx.Done():
		return nil, fmt.Errorf("onboard: lock: %w", ctx.Err())
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		<-s.sem
		return nil, fmt.Errorf("onboard: lock: %w", err)
	}
	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1, 0))", "nava-onboard:"+tenantID.String()); err != nil {
		_ = tx.Rollback(context.WithoutCancel(ctx))
		<-s.sem
		return nil, fmt.Errorf("onboard: lock: %w", err)
	}
	return func() { _ = tx.Rollback(context.WithoutCancel(ctx)); <-s.sem }, nil
}

// Begin implements Store.
func (s *PGStore) Begin(ctx context.Context, tenantID uuid.UUID, hash []byte, domains []string) (State, error) {
	var st State
	err := tenancy.InTenantTx(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var state string
		var stored []byte
		err := tx.QueryRow(ctx, "SELECT state, onboarding_request_hash FROM tenants WHERE id = $1", tenantID).Scan(&state, &stored)
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			_, err = tx.Exec(ctx,
				"INSERT INTO tenants (id, state, onboarding_request_hash) VALUES ($1, 'provisioning', $2)", tenantID, hash)
			if err != nil {
				return err
			}
			// The claim is the ownership check: the primary key is on domain
			// alone, so a domain another tenant owns fails the whole
			// transaction, the tenants row included, and Keycloak is never
			// called. Two requests for one domain serialise on the key.
			for _, d := range domains {
				if _, err := tx.Exec(ctx, "INSERT INTO tenant_email_domain (tenant_id, domain) VALUES ($1, $2)", tenantID, d); err != nil {
					return err
				}
			}
			return nil
		case err != nil:
			return err
		}
		if !bytes.Equal(stored, hash) {
			return ErrKeyReuse
		}
		st.Active = state == "active"
		return nil
	})
	if err != nil {
		if errors.Is(err, ErrKeyReuse) {
			return State{}, err
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "tenant_email_domain_pkey" {
			return State{}, ErrDomainClaimed
		}
		return State{}, fmt.Errorf("onboard: begin: %w", err)
	}
	return st, nil
}

// BindingReader returns the (issuer, org id) the tenant is already bound to,
// found false when it has no binding row. It reads as nava_app, scoped by
// WHERE tenant_id (not by an RLS policy): nava_onboard has no SELECT on
// tenant_identity_binding, and its grants are the isolation control.
type BindingReader interface {
	Binding(ctx context.Context, tenantID uuid.UUID) (issuer, orgID string, found bool, err error)
}

// Bind implements Store. The caller holds the per-tenant lock. If the tenant's
// binding row exists, an earlier Bind committed it, so a retry with the same
// (issuer, org id) inserts nothing; a different pair is ErrConflict (the stored
// org was lost outside the platform and this attempt resolved another), and
// the tenant stays provisioning. Otherwise it is a first bind: a plain INSERT
// of the binding as nava_onboard. A unique violation on the (issuer, org id)
// key means the organization is bound to another tenant (ErrConflict). The
// domains were claimed in Begin; the caller reads ownership as a second check.
func (s *PGStore) Bind(ctx context.Context, tenantID uuid.UUID, issuer, orgID, idpAlias string) error {
	// Only a provisioning tenant is bound, whether or not a binding row exists:
	// this refusal comes before the retry decision. nava_onboard reads its own
	// tenant's row in either state (ADR-001-E3).
	var state string
	if err := tenancy.InTenantTx(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, "SELECT state FROM tenants WHERE id = $1", tenantID).Scan(&state)
	}); err != nil {
		return fmt.Errorf("onboard: bind: %w", err)
	}
	if state != "provisioning" {
		return ErrNotProvisioning
	}
	boundIssuer, boundOrg, bound, err := s.bindings.Binding(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("onboard: bind: %w", err)
	}
	if bound {
		if boundIssuer != issuer || boundOrg != orgID {
			return ErrConflict
		}
		return nil
	}
	err = tenancy.InTenantTx(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx,
			"INSERT INTO tenant_identity_binding (tenant_id, issuer, org_id, idp_aliases) VALUES ($1, $2, $3, $4)",
			tenantID, issuer, orgID, []string{idpAlias}); err != nil {
			return err
		}
		return nil
	})
	var pgErr *pgconn.PgError
	switch {
	case err == nil:
		return nil
	case errors.As(err, &pgErr) && pgErr.Code == "23505":
		return ErrConflict
	default:
		return fmt.Errorf("onboard: bind: %w", err)
	}
}

// ErrAdminGroupMissing is returned by RecordAdminGroup when, after the insert,
// no groups row carries the Keycloak admins group id. That happens when the
// group was deleted and re-created in Keycloak between retries: the insert
// hits the primary key of the stale row and does nothing. Onboarding must
// stop, not activate a tenant without an admin.
var ErrAdminGroupMissing = errors.New("onboard: admins group has no groups row")

// RecordAdminGroup implements Store. The groups insert is ON CONFLICT DO
// NOTHING: if group sync already created the group, under any id, nothing is
// inserted. If no groups row carries the Keycloak id afterwards, it returns
// ErrAdminGroupMissing. Otherwise the tenant_admin_group row is written from
// that row, so it always names the group's real id. A retry inserts nothing.
func (s *PGStore) RecordAdminGroup(ctx context.Context, tenantID uuid.UUID, kcGroupID string) error {
	id := AdminGroupID(tenantID)
	path := identity.TenantGroupPrefix(tenantID) + identity.AdminGroupName
	err := tenancy.InTenantTx(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx,
			"INSERT INTO groups (tenant_id, id, keycloak_group_id, path) VALUES ($1, $2, $3, $4) ON CONFLICT DO NOTHING",
			tenantID, id, kcGroupID, path); err != nil {
			return err
		}
		var exists bool
		if err := tx.QueryRow(ctx,
			"SELECT EXISTS (SELECT 1 FROM groups WHERE tenant_id = $1 AND keycloak_group_id = $2)",
			tenantID, kcGroupID).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return ErrAdminGroupMissing
		}
		// The row names the group's real id, whoever created it: group sync
		// may have inserted the group first, under an id of its own.
		_, err := tx.Exec(ctx,
			`INSERT INTO tenant_admin_group (tenant_id, group_id)
			 SELECT tenant_id, id FROM groups WHERE tenant_id = $1 AND keycloak_group_id = $2
			 ON CONFLICT DO NOTHING`, tenantID, kcGroupID)
		return err
	})
	if err != nil {
		if errors.Is(err, ErrAdminGroupMissing) {
			return err
		}
		return fmt.Errorf("onboard: record admin group: %w", err)
	}
	return nil
}

// ErrNotProvisioning is returned by Activate when it found no provisioning row of the tenant.
var ErrNotProvisioning = errors.New("onboard: tenant is not provisioning")

// Activate implements Store.
func (s *PGStore) Activate(ctx context.Context, tenantID uuid.UUID) error {
	err := tenancy.InTenantTx(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, "UPDATE tenants SET state = 'active' WHERE id = $1 AND state = 'provisioning'", tenantID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 1 {
			return ErrNotProvisioning
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, ErrNotProvisioning) {
			return err
		}
		return fmt.Errorf("onboard: activate: %w", err)
	}
	return nil
}

// ReleaseDomains deletes the email domains the tenant claimed in Begin and
// returns how many it freed. The email_domain_release policy limits the
// DELETE to the tenant of the transaction while it is provisioning, so an
// active tenant's rows, and another tenant's, are no rows and the count is 0.
// It is not on Store: whether the abandon operation (AR-036) calls it is that
// operation's design.
func (s *PGStore) ReleaseDomains(ctx context.Context, tenantID uuid.UUID) (int64, error) {
	var n int64
	err := tenancy.InTenantTx(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, "DELETE FROM tenant_email_domain WHERE tenant_id = $1", tenantID)
		n = tag.RowsAffected()
		return err
	})
	if err != nil {
		return 0, fmt.Errorf("onboard: release domains: %w", err)
	}
	return n, nil
}

// RecordInvite implements keycloak.InviteWriter: it writes the tenant's
// pending invite for email, or moves the expiry of one that is still pending.
// A consumed invite stays consumed (the conflict update matches no row), so a
// repeat onboarding cannot re-arm an admin who has already signed in. Runs as
// nava_onboard in InTenantTx, so the row-level policy scopes it to the tenant.
func (s *PGStore) RecordInvite(ctx context.Context, tenantID uuid.UUID, email string, expiresAt time.Time) error {
	err := tenancy.InTenantTx(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		// SET names expires_at only: nava_onboard has no grant on consumed_at.
		tag, err := tx.Exec(ctx, `INSERT INTO pending_invite (tenant_id, email, expires_at) VALUES ($1, $2, $3)
			ON CONFLICT (tenant_id, email) DO UPDATE SET expires_at = EXCLUDED.expires_at
			WHERE pending_invite.consumed_at IS NULL`, tenantID, email, expiresAt)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return keycloak.ErrInviteConsumed
		}
		return nil
	})
	if errors.Is(err, keycloak.ErrInviteConsumed) {
		return err
	}
	if err != nil {
		return fmt.Errorf("onboard: record invite: %w", err)
	}
	return nil
}

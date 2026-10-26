package onboard

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

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
	// Each holds one connection for its lock and takes a second for every statement transaction, so at most
	// MaxConns/2 may run at once; more would each hold a lock connection and
	// wait for a second one that never frees.
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
func (s *PGStore) Begin(ctx context.Context, tenantID uuid.UUID, hash []byte) (State, error) {
	var st State
	err := tenancy.InTenantTx(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		var state string
		var stored []byte
		err := tx.QueryRow(ctx, "SELECT state, onboarding_request_hash FROM tenants WHERE id = $1", tenantID).Scan(&state, &stored)
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			_, err = tx.Exec(ctx,
				"INSERT INTO tenants (id, state, onboarding_request_hash) VALUES ($1, 'provisioning', $2)", tenantID, hash)
			return err
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
		return State{}, fmt.Errorf("onboard: begin: %w", err)
	}
	return st, nil
}

// BindingReader reports whether the tenant already has its binding row. It
// reads as nava_app, under the tenant predicate: nava_onboard has no SELECT on
// tenant_identity_binding, and its grants are the isolation control.
type BindingReader interface {
	HasBinding(ctx context.Context, tenantID uuid.UUID) (bool, error)
}

// Bind implements Store. The caller holds the per-tenant lock. If the tenant's
// binding row exists, an earlier Bind committed the binding and every domain
// in one transaction (and the request hash fixes the domain set), so a retry
// inserts nothing. Otherwise it is a first bind: plain INSERTs of the binding
// and every domain in one transaction as nava_onboard. A unique violation
// rolls everything back: on tenant_email_domain_pkey the domain belongs to
// another tenant (ErrDomainClaimed); on the (issuer, org id) key the
// organization is bound to another tenant (ErrConflict). The caller still
// reads domain ownership under the tenant predicate as a second check.
func (s *PGStore) Bind(ctx context.Context, tenantID uuid.UUID, issuer, orgID, idpAlias string, domains []string) error {
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
	bound, err := s.bindings.HasBinding(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("onboard: bind: %w", err)
	}
	if bound {
		return nil
	}
	err = tenancy.InTenantTx(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx,
			"INSERT INTO tenant_identity_binding (tenant_id, issuer, org_id, idp_aliases) VALUES ($1, $2, $3, $4)",
			tenantID, issuer, orgID, []string{idpAlias}); err != nil {
			return err
		}
		for _, d := range domains {
			if _, err := tx.Exec(ctx, "INSERT INTO tenant_email_domain (tenant_id, domain) VALUES ($1, $2)", tenantID, d); err != nil {
				return err
			}
		}
		return nil
	})
	var pgErr *pgconn.PgError
	switch {
	case err == nil:
		return nil
	case errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "tenant_email_domain_pkey":
		return ErrDomainClaimed
	case errors.As(err, &pgErr) && pgErr.Code == "23505":
		return ErrConflict
	default:
		return fmt.Errorf("onboard: bind: %w", err)
	}
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

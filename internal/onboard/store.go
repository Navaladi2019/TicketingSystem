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
	// sem bounds concurrent onboardings. Each holds one connection for its
	// lock and takes a second for every statement transaction, so at most
	// MaxConns/2 may run at once; more would each hold a lock connection and
	// wait for a second one that never frees.
	sem chan struct{}
}

var _ Store = (*PGStore)(nil)

// NewPGStore returns a PGStore on pool, which needs at least two connections.
func NewPGStore(pool *pgxpool.Pool) (*PGStore, error) {
	n := int(pool.Config().MaxConns) / 2
	if n < 1 {
		return nil, errors.New("onboard: the nava_onboard pool needs at least 2 connections")
	}
	return &PGStore{pool: pool, sem: make(chan struct{}, n)}, nil
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

// Bind implements Store. The statements are INSERT ... ON CONFLICT ON
// CONSTRAINT <primary key> DO NOTHING, which needs INSERT privilege only: a
// retry skips the tenant's own rows. A binding of the same (issuer, org id) to
// another tenant is a different unique key and still fails with 23505. A
// domain another tenant owns is skipped too (the conflict check ignores RLS),
// so the caller reads ownership afterwards under the tenant predicate.
func (s *PGStore) Bind(ctx context.Context, tenantID uuid.UUID, issuer, orgID, idpAlias string, domains []string) error {
	err := tenancy.InTenantTx(ctx, s.pool, tenantID, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx,
			`INSERT INTO tenant_identity_binding (tenant_id, issuer, org_id, idp_aliases) VALUES ($1, $2, $3, $4)
ON CONFLICT ON CONSTRAINT tenant_identity_binding_pkey DO NOTHING`,
			tenantID, issuer, orgID, []string{idpAlias}); err != nil {
			return err
		}
		for _, d := range domains {
			if _, err := tx.Exec(ctx,
				`INSERT INTO tenant_email_domain (tenant_id, domain) VALUES ($1, $2)
ON CONFLICT ON CONSTRAINT tenant_email_domain_pkey DO NOTHING`, tenantID, d); err != nil {
				return err
			}
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

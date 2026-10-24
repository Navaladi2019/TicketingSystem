// Package tenancy runs tenant-scoped database work. InTenantTx is the only
// way platform code sets app.tenant_id, the setting the row-level security
// policies read; the navalint setconfig analyzer fails the build on
// set_config or app.tenant_id in Go code outside this package (PG-N1, R8).
//
// The setting is transaction-local (set_config(..., true)), so it ends with
// the transaction: through PgBouncer in transaction mode the next transaction
// on the same server connection, which may belong to another client, starts
// with no tenant.
package tenancy

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNoTenant is returned for uuid.Nil. No transaction is opened.
var ErrNoTenant = errors.New("tenancy: no tenant")

// setTenantSQL scopes the transaction to one tenant. The tenant id is a bind
// parameter, which the pool's tracer never records (otelpgx records
// parameters only with WithIncludeQueryParameters, which lint bans).
const setTenantSQL = "SELECT set_config('app.tenant_id', $1, true)"

// InTenantTx runs fn in a transaction whose app.tenant_id is tenantID, and
// commits if fn returns nil. It rolls back if fn returns an error, which it
// returns unwrapped, or panics, which it re-raises after the rollback. If the
// rollback itself fails, its error is joined to fn's; pgx has then closed the
// connection, so the tenant cannot leak to the next transaction.
//
// tenantID must come from the request's resolved principal, never from a
// path, query or body.
func InTenantTx(ctx context.Context, pool *pgxpool.Pool, tenantID uuid.UUID, fn func(pgx.Tx) error) (err error) {
	if tenantID == uuid.Nil {
		return ErrNoTenant
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("tenancy: begin: %w", err)
	}
	// Rollback after Commit is a no-op, so one deferred call covers the error
	// return, the panic and a failed commit.
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}
		if err != nil {
			// ErrTxClosed: the transaction already ended (a failed commit).
			if rbErr := tx.Rollback(ctx); rbErr != nil && !errors.Is(rbErr, pgx.ErrTxClosed) {
				err = errors.Join(err, fmt.Errorf("tenancy: rollback: %w", rbErr))
			}
		}
	}()
	if _, err := tx.Exec(ctx, setTenantSQL, tenantID.String()); err != nil {
		return fmt.Errorf("tenancy: set tenant: %w", err)
	}
	if err := fn(tx); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("tenancy: commit: %w", err)
	}
	return nil
}

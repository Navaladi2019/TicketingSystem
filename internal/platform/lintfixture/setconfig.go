//go:build lintfixture

package lintfixture

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// SetTenant scopes a transaction to a tenant outside internal/tenancy. The
// navalint setconfig analyzer must report both statements (PG-N1, R8).
func SetTenant(ctx context.Context, tx pgx.Tx, tenant string) error {
	if _, err := tx.Exec(ctx, "SELECT set_config('app.tenant_id', $1, true)", tenant); err != nil {
		return err
	}
	// The session-wide form, which outlives the transaction on a pooled
	// connection.
	_, err := tx.Exec(ctx, "SET app.tenant_id = '"+tenant+"'")
	return err
}

// AsyncCommit makes a transaction's commit asynchronous outside
// internal/tenancy. Only tenancy.AllocTx may, inside its own allocation
// transaction (ADR-005 A1, A4); the setconfig analyzer must report it.
func AsyncCommit(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, "SET LOCAL synchronous_commit = off")
	return err
}

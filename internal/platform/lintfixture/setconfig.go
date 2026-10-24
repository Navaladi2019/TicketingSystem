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

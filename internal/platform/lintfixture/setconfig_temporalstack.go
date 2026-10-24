//go:build lintfixture && temporalstack

package lintfixture

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// SetTenantTemporalstack sets a tenant by hand in a file built only with
// the temporalstack tag. The tagged pass of navalint.sh must report it.
func SetTenantTemporalstack(ctx context.Context, tx pgx.Tx, tenant string) error {
	_, err := tx.Exec(ctx, "SELECT set_config('app.tenant_id', $1, false)", tenant)
	return err
}

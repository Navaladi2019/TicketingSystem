//go:build lintfixture && integration

package lintfixture

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// SetTenantIntegration sets a tenant by hand in a file built only with
// the integration tag. The tagged pass of navalint.sh must report it.
func SetTenantIntegration(ctx context.Context, tx pgx.Tx, tenant string) error {
	_, err := tx.Exec(ctx, "SELECT set_config('app.tenant_id', $1, false)", tenant)
	return err
}

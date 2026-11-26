//go:build integration

package tenancy_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// TICK-920 (ADR-009 D7): WithSynchronousCommitOff turns synchronous_commit
// off for the transaction only, with the tenant set as usual; the
// connection's next transaction, without the option, is back at the default.
func TestWithSynchronousCommitOff(t *testing.T) {
	pool, _, _ := newPool(t, env(t, "NAVA_TEST_PG_URL"), 1)
	tenantA := uuid.New()
	setting := func(tx pgx.Tx) (string, string) {
		var sc, tenant string
		if err := tx.QueryRow(t.Context(), `SELECT current_setting('synchronous_commit'), current_setting('app.tenant_id')`).
			Scan(&sc, &tenant); err != nil {
			t.Fatal(err)
		}
		return sc, tenant
	}
	if err := tenancy.InTenantTx(t.Context(), pool, tenantA, func(tx pgx.Tx) error {
		if sc, tenant := setting(tx); sc != "off" || tenant != tenantA.String() {
			t.Errorf("with the option: synchronous_commit %q, tenant %q; want off and %s", sc, tenant, tenantA)
		}
		return nil
	}, tenancy.WithSynchronousCommitOff()); err != nil {
		t.Fatal(err)
	}
	if err := tenancy.InTenantTx(t.Context(), pool, tenantA, func(tx pgx.Tx) error {
		if sc, _ := setting(tx); sc != "on" {
			t.Errorf("next transaction on the connection: synchronous_commit %q, want on", sc)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

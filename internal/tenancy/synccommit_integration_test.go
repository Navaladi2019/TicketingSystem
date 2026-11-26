//go:build integration

package tenancy_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// TICK-920 (ADR-009 D7): WithSynchronousCommitOff turns synchronous_commit
// off for the transaction only, with the tenant set as usual; the
// connection's next transaction, without the option, is back at its baseline.
// The pool pins that baseline to on (a runtime parameter of its one
// connection), so both checks mean something whatever the server's default:
// CI's cluster runs with synchronous_commit off.
func TestWithSynchronousCommitOff(t *testing.T) {
	cfg, err := pgxpool.ParseConfig(env(t, "NAVA_TEST_PG_URL"))
	if err != nil {
		t.Fatal(err)
	}
	cfg.MaxConns = 1
	cfg.ConnConfig.RuntimeParams["synchronous_commit"] = "on"
	pool, err := pgxpool.NewWithConfig(t.Context(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	tenantA := uuid.New()
	baseline := func(tx pgx.Tx) string {
		var sc string
		if err := tx.QueryRow(t.Context(), `SELECT current_setting('synchronous_commit')`).Scan(&sc); err != nil {
			t.Fatal(err)
		}
		return sc
	}
	if err := tenancy.InTenantTx(t.Context(), pool, tenantA, func(tx pgx.Tx) error {
		if sc := baseline(tx); sc != "on" {
			t.Fatalf("baseline: synchronous_commit %q, want the pool's on", sc)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
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

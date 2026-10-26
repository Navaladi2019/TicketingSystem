//go:build integration

package dbtest

import (
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/purge"
)

// TICK-217: purge.AssertRole's SQL against real roles, so a broken predicate
// (OR for AND, a dropped NOT rolsuper) fails here and not only in review.
// Every ALTER ROLE is reverted in t.Cleanup, and nava_test_intruder is
// dropped there too: db-roles refuses a cluster with a second superuser.
func TestT13PurgeAssertRoleRealRoles(t *testing.T) {
	migratedCluster(t)
	super := connect(t, superConfig(t))
	me := queryString(t, super, "SELECT current_user::text")
	t.Cleanup(func() {
		ctx := context.Background()
		conn, err := pgx.ConnectConfig(ctx, superConfig(t))
		if err != nil {
			t.Errorf("cleanup: %v", err)
			return
		}
		defer conn.Close(ctx)
		for _, sql := range []string{
			"ALTER ROLE nava_purge NOSUPERUSER NOBYPASSRLS",
			"DROP ROLE IF EXISTS nava_test_intruder",
		} {
			if _, err := conn.Exec(ctx, sql); err != nil {
				t.Errorf("cleanup %q: %v", sql, err)
			}
		}
	})

	// assertAs opens a fresh connection with cfg and runs the check on it.
	assertAs := func(cfg *pgx.ConnConfig) error {
		t.Helper()
		return purge.AssertRole(context.Background(), connect(t, cfg))
	}
	wantRefused := func(name string, err error, role string) {
		t.Helper()
		if err == nil || !strings.Contains(err.Error(), `"`+role+`"`) {
			t.Errorf("%s: err = %v, want an error naming %s", name, err, role)
		}
	}

	if err := assertAs(roleConfig(t, "nava_purge", "nava")); err != nil {
		t.Errorf("plain nava_purge: %v, want nil", err)
	}

	exec(t, super, "ALTER ROLE nava_purge BYPASSRLS")
	wantRefused("nava_purge BYPASSRLS", assertAs(roleConfig(t, "nava_purge", "nava")), "nava_purge")
	exec(t, super, "ALTER ROLE nava_purge NOBYPASSRLS SUPERUSER")
	wantRefused("nava_purge SUPERUSER", assertAs(roleConfig(t, "nava_purge", "nava")), "nava_purge")
	exec(t, super, "ALTER ROLE nava_purge NOSUPERUSER")

	// The plain role passes again, so the refusals above came from the
	// attributes and not from a broken connection.
	if err := assertAs(roleConfig(t, "nava_purge", "nava")); err != nil {
		t.Errorf("nava_purge after revert: %v, want nil", err)
	}

	// pg_roles is cluster-wide, so the intruder checks from the postgres
	// database, which it may connect to without a grant.
	exec(t, super, "CREATE ROLE nava_test_intruder LOGIN NOSUPERUSER BYPASSRLS PASSWORD '"+
		password("nava_test_intruder")+"'")
	wantRefused("other BYPASSRLS role", assertAs(roleConfig(t, "nava_test_intruder", "postgres")), "nava_test_intruder")

	wantRefused("superuser", assertAs(superConfig(t)), me)
}

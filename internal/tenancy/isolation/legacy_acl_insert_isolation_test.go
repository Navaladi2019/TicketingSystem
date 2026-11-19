//go:build integration

// TICK-943 (S30 P11-11): the two checks on legacyACLInsert (acl_paths.go),
// the test helper that turns acl_versions' tenant-id trigger (TICK-940) off:
// the trigger is on again after fn, whatever fn does, and no runtime role can
// turn it off or skip it. That it is reachable from tests only is
// TestLegacyACLInsertIsTestOnly (reach_isolation_test.go).

package isolation

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const tenantIDTriggerSQL = `SELECT tgenabled FROM pg_trigger
	WHERE tgrelid = 'acl_versions'::regclass AND tgname = 'acl_versions_tenant_ids'`

// tenantIDTrigger is the trigger's pg_trigger.tgenabled: O is on, D off.
func tenantIDTrigger(t *testing.T, conn *pgx.Conn) string {
	t.Helper()
	var s string
	if err := conn.QueryRow(context.Background(), tenantIDTriggerSQL).Scan(&s); err != nil {
		t.Fatal(err)
	}
	return s
}

func TestLegacyACLInsertAlwaysReenablesTheTrigger(t *testing.T) {
	watch := suite.superConn(t)
	if s := tenantIDTrigger(t, watch); s != "O" {
		t.Fatalf("before: trigger is %q, want O", s)
	}
	errFn := errors.New("fn failed")
	for _, tc := range []struct {
		name string
		run  func(super *pgx.Conn, fn func() error) error
		fn   func(t *testing.T) error
	}{
		{"fn succeeds", nil, func(*testing.T) error { return nil }},
		{"fn fails", nil, func(*testing.T) error { return errFn }},
		{"fn panics", func(super *pgx.Conn, fn func() error) (err error) {
			defer func() {
				if r := recover(); r == nil {
					err = errors.New("fn did not panic")
				}
			}()
			return legacyACLInsert(context.Background(), super, fn)
		}, func(*testing.T) error { panic("fn panicked") }},
		// t.FailNow and t.Fatal end fn with runtime.Goexit.
		{"fn calls runtime.Goexit", func(super *pgx.Conn, fn func() error) error {
			done := make(chan struct{})
			go func() {
				defer close(done)
				_ = legacyACLInsert(context.Background(), super, fn)
			}()
			<-done
			// Read on this goroutine, after the helper's goroutine ended.
			var after string
			if err := super.QueryRow(context.Background(), tenantIDTriggerSQL).Scan(&after); err != nil {
				return err
			}
			if after != "O" {
				return fmt.Errorf("after Goexit: trigger is %q, want O", after)
			}
			return nil
		}, func(*testing.T) error { runtime.Goexit(); return nil }},
		// The caller's context ends inside fn: the ENABLE does not use it.
		{"the context is cancelled inside fn", func(super *pgx.Conn, fn func() error) error {
			ctx, cancel := context.WithCancel(context.Background())
			return legacyACLInsert(ctx, super, func() error { cancel(); return fn() })
		}, func(*testing.T) error { return nil }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			super := suite.superConn(t)
			var during string
			fn := func() error {
				// Control: the helper really turned it off, for every session.
				during = tenantIDTrigger(t, watch)
				return tc.fn(t)
			}
			run := tc.run
			if run == nil {
				run = func(super *pgx.Conn, fn func() error) error {
					return legacyACLInsert(context.Background(), super, fn)
				}
			}
			err := run(super, fn)
			if during != "D" {
				t.Errorf("inside fn: trigger is %q, want D", during)
			}
			if s := tenantIDTrigger(t, watch); s != "O" {
				t.Fatalf("after: trigger is %q, want O", s)
			}
			if tc.name == "fn fails" && !errors.Is(err, errFn) {
				t.Errorf("err = %v, want fn's error", err)
			}
		})
	}
}

// When the ENABLE itself fails (here: fn loses the superuser connection), the
// helper says so even though fn failed too, so the run fails instead of going
// on with the trigger off. Finding F2: it used to drop the ENABLE error
// whenever fn had one.
func TestLegacyACLInsertReportsAFailedReenable(t *testing.T) {
	watch := suite.superConn(t)
	t.Cleanup(func() {
		if _, err := watch.Exec(context.Background(), "ALTER TABLE acl_versions ENABLE TRIGGER acl_versions_tenant_ids"); err != nil {
			t.Errorf("restore the trigger: %v", err)
		}
	})
	super := suite.superConn(t)
	errFn := errors.New("fn failed")
	err := legacyACLInsert(context.Background(), super, func() error {
		_ = super.Close(context.Background())
		return errFn
	})
	if !errors.Is(err, errFn) {
		t.Errorf("err = %v, want fn's error kept", err)
	}
	if err == nil || !strings.Contains(err.Error(), "re-enable acl_versions_tenant_ids") {
		t.Errorf("err = %v, want the failed ENABLE reported", err)
	}
	if s := tenantIDTrigger(t, watch); s != "D" {
		t.Errorf("control: trigger is %q, want D (the ENABLE could not run)", s)
	}
}

// No runtime role can turn the tenant-id trigger off or skip it: ALTER TABLE
// needs the owner, and session_replication_role (which skips every ordinary
// trigger) needs a superuser. So legacyACLInsert's DISABLE cannot be done by
// any service connection, wherever the code runs.
func TestRuntimeRolesCannotSkipTheTenantIDTrigger(t *testing.T) {
	for _, role := range suite.runtimeRoles() {
		if role == suite.m.Owner() {
			continue // the migrator owns the table; no service connects as it
		}
		t.Run(role, func(t *testing.T) {
			conn, state := suite.connect(t, role, false)
			if conn == nil {
				t.Logf("%s cannot connect (%s): it can do nothing at all", role, state)
				return
			}
			for _, sql := range []string{
				"ALTER TABLE acl_versions DISABLE TRIGGER acl_versions_tenant_ids",
				"ALTER TABLE acl_versions DISABLE TRIGGER ALL",
				"SET session_replication_role = replica",
			} {
				_, err := conn.Exec(context.Background(), sql)
				var pe *pgconn.PgError
				// 42501: not the owner, not a superuser. pgbouncer_auth alone
				// cannot even see the table: 42P01 for its ALTERs.
				want := "42501"
				if role == "pgbouncer_auth" && strings.HasPrefix(sql, "ALTER") {
					want = "42P01"
				}
				if !errors.As(err, &pe) || pe.Code != want {
					t.Errorf("%s: %s = %v, want %s", role, sql, err, want)
				}
			}
			if s := tenantIDTrigger(t, suite.superConn(t)); s != "O" {
				t.Fatalf("trigger is %q after %s's attempts", s, role)
			}
		})
	}
}

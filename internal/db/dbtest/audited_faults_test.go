//go:build integration

package dbtest

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
)

// failEach runs fn on a fresh connection whose nth statement fails, for n =
// 1, 2, ... until a run in which no statement was failed, in both of pgx's
// failure shapes: a statement prepared on first use fails at the prepare
// (Query returns the error), one sent at once in exec mode fails when its
// rows are read (rows.Err, pgx.CollectRows). Every failed run must return the
// injected error: a catalogue read that cannot run never passes the gate.
// It returns the statement count of the clean run.
func failEach(t *testing.T, base *pgx.ConnConfig, fn func(context.Context, *pgx.Conn) error) int {
	t.Helper()
	statements := 0
	for _, mode := range []pgx.QueryExecMode{pgx.QueryExecModeCacheStatement, pgx.QueryExecModeExec} {
		for n := 1; ; n++ {
			if n > 1000 {
				t.Fatalf("mode %v: still failing a statement at n = %d", mode, n)
			}
			cfg, f := withFault(base, func(string) bool { return true }, n)
			cfg.DefaultQueryExecMode = mode
			conn, err := pgx.ConnectConfig(context.Background(), cfg)
			if err != nil {
				t.Fatal(err)
			}
			err = fn(context.Background(), conn)
			_ = conn.Close(context.Background())
			if !f.fired() {
				if err != nil {
					t.Fatalf("mode %v: the run with no fault failed: %v", mode, err)
				}
				statements = n - 1
				break
			}
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("mode %v: statement %d failed (%s), but the run returned %v, want the injected error",
					mode, n, f.hits[0], err)
			}
		}
	}
	return statements
}

// CI-A3 (audited.Manifest.Check) returns the error of any catalogue read
// that fails, at every one of its statements, and never a partial problem
// list that could read as a pass.
func TestCIA3CheckFailsOnEveryReadError(t *testing.T) {
	migratedPlatform(t)
	m, a, g := manifest(t), auditedManifest(t), globalTables(t)
	base := superConfig(t).Copy()
	base.Database = m.PlatformDatabase
	n := failEach(t, base, func(ctx context.Context, conn *pgx.Conn) error {
		p, err := a.Check(ctx, conn, m, g)
		if err == nil && len(p) != 0 {
			t.Fatalf("the real catalogue has CI-A3 problems: %v", p)
		}
		return err
	})
	if n < 10 {
		t.Errorf("Check ran %d statements, want every catalogue read (at least 10)", n)
	}
}

// ApplyPolicies, as the owner, rolls back and returns the error when any of
// its statements fails, and on the migrated database changes nothing.
func TestApplyPoliciesFailsOnEveryStatementError(t *testing.T) {
	migratedPlatform(t)
	m, a := manifest(t), auditedManifest(t)
	log, _ := newLog()
	n := failEach(t, roleConfig(t, m.Owner(), m.PlatformDatabase), func(ctx context.Context, conn *pgx.Conn) error {
		res, err := a.ApplyPolicies(ctx, conn, log)
		if err == nil && res.Changed() {
			t.Fatalf("ApplyPolicies changed the migrated policies: %+v", res)
		}
		return err
	})
	if n < 4 {
		t.Errorf("ApplyPolicies ran %d statements, want begin, lock, two reads and commit", n)
	}
}

//go:build integration

package dbtest

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// TICK-1122: the locks of org_unit_closure_rebuild_all() that
// docs/data/org-unit-closure.md ("Run the rebuild by hand") states. CREATE
// POLICY takes ACCESS EXCLUSIVE until commit, so the function stops readers.
// A change to the function that makes one of these fail must change that page.

const (
	rebuildAll = "SELECT org_unit_closure_rebuild_all()"
	readUnits  = "SELECT count(*) FROM org_units"
	readTree   = "SELECT count(*) FROM org_unit_closure"
	readTenant = "SELECT count(*) FROM tenants"
)

// rebuildLocksFixture is a migrated cluster with one tenant and its root unit.
func rebuildLocksFixture(t *testing.T) (super *pgx.Conn) {
	t.Helper()
	migratedPlatform(t)
	super = superNava(t)
	seedTenants(t, super, []string{tenantA}, nil)
	seedIdentity(t, super, tenantA)
	return super
}

// beginIn opens a transaction with app.tenant_id set to tenant A, and
// lock_timeout when it is not "". The cleanup rolls back what is still open.
func beginIn(t *testing.T, conn *pgx.Conn, lockTimeout string) pgx.Tx {
	t.Helper()
	ctx := context.Background()
	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tx.Rollback(ctx) })
	mustTx(t, tx, "SELECT set_config('app.tenant_id', $1, true)", tenantA)
	if lockTimeout != "" {
		mustTx(t, tx, "SELECT set_config('lock_timeout', $1, true)", lockTimeout)
	}
	return tx
}

func mustTx(t *testing.T, tx pgx.Tx, sql string, args ...any) {
	t.Helper()
	if _, err := tx.Exec(context.Background(), sql, args...); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
}

// readOnce runs sql in its own transaction under lock_timeout 500 ms.
func readOnce(t *testing.T, conn *pgx.Conn, sql string) error {
	t.Helper()
	ctx := context.Background()
	tx := beginIn(t, conn, "500ms")
	_, err := tx.Exec(ctx, sql)
	if rerr := tx.Rollback(ctx); rerr != nil {
		t.Fatal(rerr)
	}
	return err
}

// startRebuild runs the function on tx and returns once it waits for relation.
// The transaction commits or rolls back before the error is sent.
func startRebuild(t *testing.T, super *pgx.Conn, tx pgx.Tx, relation string) <-chan error {
	t.Helper()
	ctx := context.Background()
	pid := int64(tx.Conn().PgConn().PID())
	done := make(chan error, 1)
	go func() {
		_, err := tx.Exec(ctx, rebuildAll)
		if err == nil {
			err = tx.Commit(ctx)
		} else {
			_ = tx.Rollback(ctx)
		}
		done <- err
	}()
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); {
		if queryBool(t, super, "SELECT EXISTS (SELECT 1 FROM pg_locks WHERE pid = $1 AND NOT granted AND relation = $2::text::regclass)", pid, relation) {
			return done
		}
	}
	t.Fatalf("the function does not wait for %s", relation)
	return nil
}

func wantNoMigratorPolicy(t *testing.T, super *pgx.Conn) {
	t.Helper()
	if queryBool(t, super, "SELECT EXISTS (SELECT 1 FROM pg_policy WHERE 'nava_migrator'::regrole = ANY (polroles))") {
		t.Error("a nava_migrator policy outlived the function (R26.3)")
	}
}

// Until its transaction commits, the function holds ACCESS EXCLUSIVE on the
// three tables, and a plain read of any of them waits.
func TestTICK1122RebuildAllStopsReaders(t *testing.T) {
	super := rebuildLocksFixture(t)
	mig := asRole(t, "nava_migrator")
	app := asRole(t, "nava_app")
	tx := beginIn(t, mig, "")
	mustTx(t, tx, rebuildAll)

	got := queryString(t, super, `SELECT coalesce(string_agg(relation::regclass::text, ',' ORDER BY relation::regclass::text), '')
		FROM pg_locks WHERE pid = $1 AND granted AND mode = 'AccessExclusiveLock'
		AND relation IN ('tenants'::regclass, 'org_units'::regclass, 'org_unit_closure'::regclass)`, int64(mig.PgConn().PID()))
	if want := "org_unit_closure,org_units,tenants"; got != want {
		t.Errorf("ACCESS EXCLUSIVE on %q, want %q", got, want)
	}
	for _, sql := range []string{readTenant, readUnits, readTree} {
		wantSQLState(t, "nava_app, "+sql+" while the function is uncommitted", readOnce(t, app, sql), "55P03")
	}
	if err := tx.Commit(context.Background()); err != nil {
		t.Fatal(err)
	}
	for _, sql := range []string{readTenant, readUnits, readTree} {
		if err := readOnce(t, app, sql); err != nil {
			t.Errorf("nava_app, %s after the commit: %v", sql, err)
		}
	}
	wantNoMigratorPolicy(t, super)
}

// The function waits for a transaction that has only read the closure, and
// later readers of all three tables wait behind it. lock_timeout ends the wait.
func TestTICK1122RebuildAllWaitsForAReader(t *testing.T) {
	super := rebuildLocksFixture(t)
	reader, late := asRole(t, "nava_app"), asRole(t, "nava_app")

	open := beginIn(t, reader, "")
	mustTx(t, open, readTree)
	done := startRebuild(t, super, beginIn(t, asRole(t, "nava_migrator"), ""), "org_unit_closure")
	for _, sql := range []string{readTenant, readUnits, readTree} {
		wantSQLState(t, "a later nava_app, "+sql+" while the function waits", readOnce(t, late, sql), "55P03")
	}
	if err := open.Rollback(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Errorf("the function after the reader ended: %v", err)
	}

	mustTx(t, beginIn(t, reader, ""), readTree)
	_, err := beginIn(t, asRole(t, "nava_migrator"), "500ms").Exec(context.Background(), rebuildAll)
	wantSQLState(t, "the function with lock_timeout while a reader is open", err, "55P03")
}

// A transaction that has read org_units or the closure and then needs tenants
// (a foreign key check, a read) deadlocks with the function. The side whose
// deadlock_timeout (1 s) fires on the cycle is rolled back, so it can be group
// sync or a user's request. When the function waits once, lock_timeout below
// deadlock_timeout makes it give up first and the other side finishes. With
// more than one wait it may not: lock_timeout restarts at each
// (TestTICK1122ProcedureSparesARequest).
func TestTICK1122RebuildAllDeadlocks(t *testing.T) {
	for _, other := range []struct {
		name, role, read, waits, then string
		args                          []any
	}{
		{"group sync inserts a unit", "nava_groupsync", readUnits, "org_units",
			insertUnit, []any{tenantA, uuid.NewString(), tenantA}},
		{"a request reads the closure, then tenants", "nava_app", readTree, "org_unit_closure",
			readTenant, nil},
	} {
		// The function checks current_user and deadlock_timeout is
		// superuser-only, so the function's session is the superuser's under
		// SET ROLE.
		run := func(t *testing.T, deadlockTimeout, lockTimeout string) (rebuildErr, otherErr error) {
			t.Helper()
			super := rebuildLocksFixture(t)
			mig := superNava(t)
			exec(t, mig, "SELECT set_config('deadlock_timeout', $1, false)", deadlockTimeout)
			exec(t, mig, "SET ROLE nava_migrator")
			otherTx := beginIn(t, asRole(t, other.role), "")
			mustTx(t, otherTx, other.read)
			done := startRebuild(t, super, beginIn(t, mig, lockTimeout), other.waits)
			_, otherErr = otherTx.Exec(context.Background(), other.then, other.args...)
			if otherErr == nil {
				otherErr = otherTx.Commit(context.Background())
			} else {
				_ = otherTx.Rollback(context.Background())
			}
			rebuildErr = <-done
			wantNoMigratorPolicy(t, super)
			t.Logf("function: %v; other: %v", rebuildErr, otherErr)
			return rebuildErr, otherErr
		}
		t.Run(other.name+", one side is the victim", func(t *testing.T) {
			r, o := run(t, "1s", "")
			if (sqlState(r) == "40P01") == (sqlState(o) == "40P01") {
				t.Errorf("function: %v, other: %v; want 40P01 on exactly one", r, o)
			}
		})
		// 1min stands for a function that has waited past its one check.
		t.Run(other.name+", the other side is the victim after the function's check", func(t *testing.T) {
			r, o := run(t, "1min", "")
			wantSQLState(t, "the other side", o, "40P01")
			if r != nil {
				t.Errorf("function: %v, want success", r)
			}
		})
		t.Run(other.name+", one wait and lock_timeout below deadlock_timeout spare it", func(t *testing.T) {
			r, o := run(t, "1s", "500ms")
			wantSQLState(t, "the function", r, "55P03")
			if o != nil {
				t.Errorf("other: %v, want success", o)
			}
		})
	}
}

// runProcedure runs the SQL of the page's "Procedure" in its own transaction.
func runProcedure(conn *pgx.Conn) error {
	ctx := context.Background()
	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	for _, sql := range []string{
		"SET LOCAL lock_timeout = '500ms'",
		"LOCK TABLE tenants, org_units, org_unit_closure IN ACCESS EXCLUSIVE MODE NOWAIT",
		rebuildAll,
	} {
		if _, err := tx.Exec(ctx, sql); err != nil {
			_ = tx.Rollback(ctx)
			return err
		}
	}
	return tx.Commit(ctx)
}

// The documented procedure never makes a request the deadlock victim: while a
// request has read the closure, it fails with 55P03 and the request then reads
// tenants. The two other readers are the case that lock_timeout alone gets
// wrong: without the LOCK TABLE line the procedure waits for tenants, org_units
// and the closure, each under 500 ms, and the request, queued on tenants since
// the first wait, reaches its deadlock check first. The timers run only then;
// with the LOCK TABLE line the procedure ends at once.
func TestTICK1122ProcedureSparesARequest(t *testing.T) {
	ctx := context.Background()
	super := rebuildLocksFixture(t)
	onTenants := beginIn(t, asRole(t, "nava_app"), "")
	mustTx(t, onTenants, readTenant)
	onUnits := beginIn(t, asRole(t, "nava_app"), "")
	mustTx(t, onUnits, readUnits)
	request := beginIn(t, asRole(t, "nava_app"), "")
	mustTx(t, request, readTree)

	mig := asRole(t, "nava_migrator")
	pid := int64(mig.PgConn().PID())
	var procedureErr error
	done := make(chan struct{})
	go func() {
		defer close(done)
		procedureErr = runProcedure(mig)
	}()
	for started := false; !started; {
		select {
		case <-done:
			started = true
		default:
			started = queryBool(t, super, "SELECT EXISTS (SELECT 1 FROM pg_locks WHERE pid = $1 AND NOT granted)", pid)
		}
	}
	requestErr := make(chan error, 1)
	go func() {
		_, err := request.Exec(ctx, readTenant)
		if err == nil {
			err = request.Commit(ctx)
		} else {
			_ = request.Rollback(ctx)
		}
		requestErr <- err
	}()
	for _, reader := range []pgx.Tx{onTenants, onUnits} {
		select {
		case <-done:
		case <-time.After(400 * time.Millisecond):
		}
		_ = reader.Rollback(ctx)
	}
	err := <-requestErr
	<-done
	wantSQLState(t, "the procedure", procedureErr, "55P03")
	if err != nil {
		t.Errorf("the request: %v, want success", err)
	}

	// Nothing holds the tables now: the same procedure succeeds.
	if err := runProcedure(mig); err != nil {
		t.Errorf("the procedure on idle tables: %v", err)
	}
	wantNoMigratorPolicy(t, super)
}

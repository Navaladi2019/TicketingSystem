//go:build integration

package dbtest

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/db/migrate"
)

// TICK-579: CREATE POLICY ... ON outbox needs ACCESS EXCLUSIVE. Behind a long
// transaction on outbox it must give up after lock_timeout (so producers do
// not queue behind it) and retry, not wait without bound.
const t579Name = "29991231000000_t579_outbox_policy.sql"

const t579Body = "-- +goose Up\nCREATE POLICY t579_probe ON outbox FOR SELECT TO nava_app USING (false);\n"

// holdOutbox opens a superuser transaction that reads outbox and so holds
// ACCESS SHARE on it until the returned release is called, as a long report
// query or an idle-in-transaction producer would.
func holdOutbox(t *testing.T, database string) (release func()) {
	t.Helper()
	ctx := context.Background()
	conn := connect(t, roleConfigDB(t, database))
	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, "SELECT count(*) FROM outbox"); err != nil {
		t.Fatal(err)
	}
	return func() { _ = tx.Rollback(ctx) }
}

// roleConfigDB is the superuser on database d.
func roleConfigDB(t *testing.T, d string) *pgx.ConnConfig {
	t.Helper()
	cfg := superConfig(t).Copy()
	cfg.Database = d
	return cfg
}

func t579Options(attempts int) migrate.Options {
	return migrate.Options{
		LockRetrySeconds: 1,
		LockTimeout:      200 * time.Millisecond,
		DDLAttempts:      attempts,
		DDLBackoff:       100 * time.Millisecond,
		DDLMaxBackoff:    200 * time.Millisecond,
	}
}

func policyExists(t *testing.T, conn *pgx.Conn) bool {
	t.Helper()
	return queryBool(t, conn, "SELECT EXISTS (SELECT 1 FROM pg_policies WHERE tablename = 'outbox' AND policyname = 't579_probe')")
}

// The holder outlives the first attempts: each one times out at
// lock_timeout, the runner retries, and once the holder commits the
// migration applies.
func TestMigrationLockTimeoutRetriesUntilLockFree(t *testing.T) {
	m := migratedCluster(t)
	fsys := withExtraMigration(t, t579Name, t579Body)
	cfg := roleConfig(t, m.Owner(), m.PlatformDatabase)
	log, buf := newLog()
	release := holdOutbox(t, m.PlatformDatabase)
	defer release()

	type run struct {
		applied []string
		err     error
	}
	done := make(chan run, 1)
	go func() {
		a, err := migrate.Up(context.Background(), cfg, fsys, log, t579Options(50))
		done <- run{a, err}
	}()

	// Two lock_timeout retries prove the DDL gave up instead of queueing.
	deadline := time.Now().Add(30 * time.Second)
	for strings.Count(buf.String(), `"msg":"migration hit lock_timeout, retrying"`) < 2 {
		select {
		case r := <-done:
			t.Fatalf("Up returned while outbox was held: %v, %v", r.applied, r.err)
		default:
		}
		if time.Now().After(deadline) {
			t.Fatalf("no lock_timeout retries logged; log:\n%s", buf.String())
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !strings.Contains(buf.String(), "55P03") {
		t.Errorf("retry log does not carry SQLSTATE 55P03:\n%s", buf.String())
	}

	// A producer's write lock queues behind the waiting ACCESS EXCLUSIVE for
	// at most one lock_timeout (200ms), so under a 1s lock_timeout of its
	// own it always gets through. Without the runner's lock_timeout it would
	// wait as long as the holder lives.
	super := connect(t, roleConfigDB(t, m.PlatformDatabase))
	for i := range 5 {
		ctx := context.Background()
		tx, err := super.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := tx.Exec(ctx, "SET LOCAL lock_timeout = '1s'"); err != nil {
			t.Fatal(err)
		}
		if _, err := tx.Exec(ctx, "LOCK TABLE outbox IN ROW EXCLUSIVE MODE"); err != nil {
			t.Fatalf("producer lock %d queued behind the migration: %v", i, err)
		}
		if err := tx.Commit(ctx); err != nil {
			t.Fatal(err)
		}
	}

	release()
	r := <-done
	if r.err != nil {
		t.Fatalf("Up after the holder ended: %v", r.err)
	}
	if len(r.applied) != 1 || !strings.HasSuffix(r.applied[0], t579Name) {
		t.Fatalf("applied %v, want only %s", r.applied, t579Name)
	}
	if !policyExists(t, super) {
		t.Fatal("t579_probe policy was not created")
	}
}

// A holder that never ends: the runner stops after DDLAttempts with the
// lock_timeout error, records nothing, and leaves no policy behind.
func TestMigrationLockTimeoutGivesUpAfterBoundedAttempts(t *testing.T) {
	m := migratedCluster(t)
	fsys := withExtraMigration(t, t579Name, t579Body)
	cfg := roleConfig(t, m.Owner(), m.PlatformDatabase)
	log, buf := newLog()
	release := holdOutbox(t, m.PlatformDatabase)
	defer release()

	start := time.Now()
	applied, err := migrate.Up(context.Background(), cfg, fsys, log, t579Options(3))
	elapsed := time.Since(start)

	if code := sqlState(err); code != "55P03" {
		t.Fatalf("Up = %v, %v (SQLSTATE %q); want lock_not_available", applied, err, code)
	}
	if !strings.Contains(err.Error(), "on 3 attempts") {
		t.Errorf("error %q does not name the attempts", err)
	}
	if applied != nil {
		t.Errorf("applied = %v with an error, want nil", applied)
	}
	// 3 x 200ms lock_timeout + 100ms + 200ms backoff, plus connection setup.
	if elapsed > 10*time.Second {
		t.Errorf("Up took %s; the attempts are not bounded", elapsed)
	}
	if n := strings.Count(buf.String(), `"msg":"migration hit lock_timeout, retrying"`); n != 2 {
		t.Errorf("%d retries logged, want 2", n)
	}
	release()
	super := connect(t, roleConfigDB(t, m.PlatformDatabase))
	if policyExists(t, super) {
		t.Error("t579_probe exists after the failed run")
	}
	if queryBool(t, super, "SELECT EXISTS (SELECT 1 FROM goose_db_version WHERE version_id = 29991231000000)") {
		t.Error("the failed migration has a version row")
	}
}

// A run cancelled while it backs off stops at once, with both the
// cancellation and the lock_timeout error in the chain.
func TestMigrationLockTimeoutBackoffHonoursCancel(t *testing.T) {
	m := migratedCluster(t)
	fsys := withExtraMigration(t, t579Name, t579Body)
	cfg := roleConfig(t, m.Owner(), m.PlatformDatabase)
	log, _ := newLog()
	release := holdOutbox(t, m.PlatformDatabase)
	defer release()

	opt := t579Options(1000)
	opt.DDLBackoff, opt.DDLMaxBackoff = time.Minute, time.Minute
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	start := time.Now()
	_, err := migrate.Up(ctx, cfg, fsys, log, opt)
	if !errors.Is(err, context.DeadlineExceeded) || sqlState(err) != "55P03" {
		t.Fatalf("Up = %v; want the deadline and the lock_timeout error", err)
	}
	if elapsed := time.Since(start); elapsed > 10*time.Second {
		t.Fatalf("Up took %s; the backoff ignored the cancellation", elapsed)
	}
}

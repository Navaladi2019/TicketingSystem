//go:build integration

package barrier_test

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"

	"github.com/Navaladi2019/TicketingSystem/internal/fieldindex/barrier"
)

// The barrier's error paths against a real PostgreSQL (NAVA_TEST_PG_URL): it
// reads no tenant data, so any database of the cluster will do. Each failure
// ends Wait with an error the activity retries; none hangs.

func pool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("NAVA_TEST_PG_URL")
	if url == "" {
		t.Fatal("NAVA_TEST_PG_URL is not set; run make test-integration")
	}
	p, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(p.Close)
	return p
}

// wait runs barrier.Wait as an activity, as the backfills do, and returns
// its error.
func wait(t *testing.T, p *pgxpool.Pool, o barrier.Options) error {
	t.Helper()
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestActivityEnvironment()
	env.SetTestTimeout(30 * time.Second)
	env.RegisterActivityWithOptions(func(ctx context.Context) error { return barrier.Wait(ctx, p, o) },
		activity.RegisterOptions{Name: "barrier"})
	_, err := env.ExecuteActivity("barrier")
	return err
}

// With nothing open, the barrier returns at once.
func TestWaitReturnsWithNothingOpen(t *testing.T) {
	if err := wait(t, pool(t), barrier.Options{}); err != nil {
		t.Fatalf("wait: %v", err)
	}
}

// StopIf's error ends the wait with that error, before any status poll.
func TestWaitStopIfError(t *testing.T) {
	cause := errors.New("registry unreadable")
	err := wait(t, pool(t), barrier.Options{StopIf: func(context.Context) (bool, error) { return false, cause }})
	if err == nil || !strings.Contains(err.Error(), "registry unreadable") {
		t.Fatalf("wait = %v, want StopIf's error", err)
	}
	// StopIf reporting true ends the wait cleanly.
	if err := wait(t, pool(t), barrier.Options{StopIf: func(context.Context) (bool, error) { return true, nil }}); err != nil {
		t.Fatalf("wait with StopIf true: %v", err)
	}
}

// A database gone before the snapshot, or before the first status poll,
// ends the wait with a wrapped error naming the step.
func TestWaitDatabaseErrors(t *testing.T) {
	down := pool(t)
	down.Close()
	if err := wait(t, down, barrier.Options{}); err == nil || !strings.Contains(err.Error(), "barrier: snapshot") {
		t.Errorf("closed pool = %v, want a snapshot error", err)
	}

	// The snapshot is taken, then the pool goes before the status poll.
	p := pool(t)
	err := wait(t, p, barrier.Options{StopIf: func(context.Context) (bool, error) { p.Close(); return false, nil }})
	if err == nil || !strings.Contains(err.Error(), "barrier: status") {
		t.Errorf("pool closed after the snapshot = %v, want a status error", err)
	}

	// The fence fails when the snapshot's transaction loses its connection
	// between the two statements.
	q := pool(t)
	err = wait(t, q, barrier.Options{AfterSnapshot: func(ctx context.Context) {
		_, _ = pool(t).Exec(ctx, `SELECT pg_terminate_backend(pid) FROM pg_stat_activity
			WHERE pid <> pg_backend_pid() AND datname = current_database() AND state LIKE 'idle in transaction%'`)
	}})
	if err == nil || !strings.Contains(err.Error(), "barrier: snapshot") {
		t.Errorf("connection lost before the fence = %v, want a snapshot error", err)
	}
}

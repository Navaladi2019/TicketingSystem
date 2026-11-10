package tenancy_test

import (
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// TestQueueUnderLockTimeout: the statement is queued between a transaction-
// local lock_timeout of the bound and its reset to 0, and the returned query
// is the statement's own (TICK-688).
func TestQueueUnderLockTimeout(t *testing.T) {
	t.Parallel()
	b := &pgx.Batch{}
	b.Queue("SELECT 'before'")
	q := tenancy.QueueUnderLockTimeout(b, 2*time.Second, "INSERT INTO t VALUES ($1)", 7)
	b.Queue("SELECT 'after'")
	got := b.QueuedQueries
	if len(got) != 5 {
		t.Fatalf("%d statements queued, want 5", len(got))
	}
	set, stmt, reset := got[1], got[2], got[3]
	if !strings.Contains(set.SQL, "set_config('lock_timeout', $1, true)") || len(set.Arguments) != 1 || set.Arguments[0] != "2000ms" {
		t.Errorf("set = %q %v, want a transaction-local lock_timeout of 2000ms", set.SQL, set.Arguments)
	}
	if stmt != q || stmt.SQL != "INSERT INTO t VALUES ($1)" || len(stmt.Arguments) != 1 || stmt.Arguments[0] != 7 {
		t.Errorf("statement = %q %v, want the caller's, returned", stmt.SQL, stmt.Arguments)
	}
	if !strings.Contains(reset.SQL, "set_config('lock_timeout', '0', true)") || len(reset.Arguments) != 0 {
		t.Errorf("reset = %q, want lock_timeout back to 0, transaction-local", reset.SQL)
	}
}

// TestQueueUnderLockTimeoutFloor: a bound under 1 ms is 1 ms, never 0, which
// would mean no bound at all.
func TestQueueUnderLockTimeoutFloor(t *testing.T) {
	t.Parallel()
	b := &pgx.Batch{}
	tenancy.QueueUnderLockTimeout(b, 300*time.Microsecond, "SELECT 1")
	if got := b.QueuedQueries[0].Arguments[0]; got != "1ms" {
		t.Errorf("lock_timeout = %v, want 1ms", got)
	}
}

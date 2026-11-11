package tenancy

import (
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
)

// The statements QueueUnderLockTimeout sends around the caller's. Both are
// transaction-local (is_local true), so neither outlives the transaction:
// through PgBouncer in transaction mode the next transaction on the server
// connection starts at the server default. The reset is to '0', no timeout,
// which is the platform's value: no role, database or pool sets lock_timeout
// (TestNoRoleOrDatabaseDefaultSetsLockTimeout in internal/db/dbtest fails if
// one does; restoring the previous value instead would cost a statement).
const (
	lockTimeoutSetSQL   = "SELECT set_config('lock_timeout', $1, true)"
	lockTimeoutResetSQL = "SELECT set_config('lock_timeout', '0', true)"
)

// QueueUnderLockTimeout queues sql on b so that it, and only it, waits at most
// timeout for a lock: lock_timeout is set to timeout just before sql and back
// to 0 just after it, in the same batch, so the bound costs no round trip. A
// lock wait past the bound fails sql with SQLSTATE 55P03 (lock_not_available)
// and aborts the transaction; statements queued after it do not run. Lock
// waits elsewhere in the transaction keep their usual, unbounded behaviour.
//
// The returned QueuedQuery is sql's, for the caller's result callback.
// timeout is rounded down to whole milliseconds and must be at least 1 ms.
func QueueUnderLockTimeout(b *pgx.Batch, timeout time.Duration, sql string, args ...any) *pgx.QueuedQuery {
	b.Queue(lockTimeoutSetSQL, strconv.FormatInt(max(timeout.Milliseconds(), 1), 10)+"ms")
	q := b.Queue(sql, args...)
	b.Queue(lockTimeoutResetSQL)
	return q
}

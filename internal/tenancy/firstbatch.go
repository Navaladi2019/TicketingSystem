package tenancy

import (
	"context"
	"fmt"
	"strconv"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Begin in the first batch (TICK-692, PLAN-003 P3-36i). With the
// BeginInFirstBatch option, InTenantTx sends no BEGIN of its own: the
// transaction opens in the round trip of the first thing fn sends.
//
//   - A pgx.Batch goes out as BEGIN; set_config('app.tenant_id', $1, true);
//     then the batch's own statements, in one round trip. PostgreSQL runs a
//     pipeline in order and skips everything after a failed statement, so
//     nothing of the caller's can run before the tenant is set, and a
//     failure leaves an aborted transaction, which InTenantTx rolls back.
//   - Anything else (Exec, Query, QueryRow, CopyFrom, Prepare, a savepoint)
//     first sends InTenantTx's usual BEGIN round trip, then itself.
//   - A transaction that sends nothing sends nothing, COMMIT included; one
//     that only queued ExecAtCommit statements sends BEGIN, the tenant, those
//     statements and COMMIT in one round trip.

// The statements a first batch opens with. The tenant is a bind parameter,
// uuid.UUID.String(), as in AllocTx.
const (
	firstBeginSQL  = "BEGIN"
	firstTenantSQL = "SELECT set_config('app.tenant_id', $1, true)"
)

// TxOption changes how InTenantTx opens its transaction.
type TxOption func(*txOptions)

type txOptions struct {
	beginInFirstBatch bool
}

// BeginInFirstBatch makes InTenantTx open its transaction, and set the
// tenant, in the round trip of the first batch fn sends, saving the BEGIN
// round trip. Use it where fn's first statement is a batch, such as record
// create's claim and prefetch.
func BeginInFirstBatch() TxOption {
	return func(o *txOptions) { o.beginInFirstBatch = true }
}

// FirstBatchTenant reports whether b is the first batch of a
// BeginInFirstBatch transaction, and the tenant it sets. It is for tracers
// that check the GUC ordering from the driver's side (tenancytest.Guard), so
// the statements' text stays owned by this package. An AllocTx batch is not
// one (AllocTenant).
func FirstBatchTenant(b *pgx.Batch) (uuid.UUID, bool) {
	q := b.QueuedQueries
	if len(q) < 2 || q[0].SQL != firstBeginSQL || q[1].SQL != firstTenantSQL || len(q[0].Arguments) != 0 || len(q[1].Arguments) != 1 {
		return uuid.Nil, false
	}
	if _, ok := AllocTenant(b); ok {
		return uuid.Nil, false
	}
	s, ok := q[1].Arguments[0].(string)
	if !ok {
		return uuid.Nil, false
	}
	tenant, err := uuid.Parse(s)
	if err != nil || tenant == uuid.Nil || tenant.String() != s {
		return uuid.Nil, false
	}
	return tenant, true
}

// lazyTx is a pgx.Tx whose BEGIN has not been sent until something is. Only
// InTenantTx builds one (with BeginInFirstBatch), wrapped in a scopedTx. Its
// Commit and Rollback do what pgx's own transaction does: COMMIT reported as
// ROLLBACK is pgx.ErrTxCommitRollback, and a failed ROLLBACK closes the
// connection. Like pgx.Tx it is not safe for concurrent use.
type lazyTx struct {
	conn   *pgx.Conn
	tenant uuid.UUID
	begun  bool
	closed bool
	sp     int64 // savepoints opened
}

var _ pgx.Tx = (*lazyTx)(nil)

// begin sends InTenantTx's BEGIN round trip, once.
func (t *lazyTx) begin(ctx context.Context) error {
	if t.closed {
		return pgx.ErrTxClosed
	}
	if t.begun {
		return nil
	}
	if _, err := t.conn.Exec(ctx, beginSQL(t.tenant)); err != nil {
		// As pgx does for a failed BEGIN: the connection's state is unknown.
		_ = t.conn.Close(ctx)
		t.closed = true
		return fmt.Errorf("tenancy: begin: %w", err)
	}
	t.begun = true
	return nil
}

// SendBatch sends b; the first one carries the BEGIN and the tenant.
func (t *lazyTx) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults {
	if t.closed {
		return errBatch{pgx.ErrTxClosed}
	}
	if t.begun {
		return t.conn.SendBatch(ctx, b)
	}
	first := &pgx.Batch{QueuedQueries: make([]*pgx.QueuedQuery, 0, len(b.QueuedQueries)+2)}
	first.Queue(firstBeginSQL)
	first.Queue(firstTenantSQL, t.tenant.String())
	first.QueuedQueries = append(first.QueuedQueries, b.QueuedQueries...)
	// From here the server may have run BEGIN, so Rollback must send
	// ROLLBACK; if it never arrived, the ROLLBACK is only a warning.
	t.begun = true
	return &firstResults{BatchResults: t.conn.SendBatch(ctx, first), pending: 2}
}

func (t *lazyTx) Begin(ctx context.Context) (pgx.Tx, error) {
	if err := t.begin(ctx); err != nil {
		return nil, err
	}
	t.sp++
	if _, err := t.conn.Exec(ctx, "savepoint sp_"+strconv.FormatInt(t.sp, 10)); err != nil {
		return nil, err
	}
	return &lazySavepoint{lazyTx: t, n: t.sp}, nil
}

func (t *lazyTx) Commit(ctx context.Context) error {
	if t.closed {
		return pgx.ErrTxClosed
	}
	t.closed = true
	if !t.begun {
		return nil
	}
	ct, err := t.conn.Exec(ctx, "commit")
	if err != nil {
		if t.conn.PgConn().TxStatus() != 'I' {
			_ = t.conn.Close(ctx)
		}
		return err
	}
	if ct.String() == "ROLLBACK" {
		return pgx.ErrTxCommitRollback
	}
	return nil
}

func (t *lazyTx) Rollback(ctx context.Context) error {
	if t.closed {
		return pgx.ErrTxClosed
	}
	t.closed = true
	if !t.begun {
		return nil
	}
	if _, err := t.conn.Exec(ctx, "rollback"); err != nil {
		// The connection's state is unknown: never reuse it.
		_ = t.conn.Close(ctx)
		return fmt.Errorf("rollback failed: %w", err)
	}
	return nil
}

func (t *lazyTx) CopyFrom(ctx context.Context, table pgx.Identifier, cols []string, src pgx.CopyFromSource) (int64, error) {
	if err := t.begin(ctx); err != nil {
		return 0, err
	}
	return t.conn.CopyFrom(ctx, table, cols, src)
}

// errLargeObjects is LargeObjects' panic value.
const errLargeObjects = "tenancy: LargeObjects is not supported on tenant transactions (BeginInFirstBatch)"

// LargeObjects panics: pgx builds a pgx.LargeObjects only for its own
// transactions, and its one field is unexported, so no error-valued one can
// be returned without unsafe. Nothing in the platform uses large objects, and
// navalint tenancyconn reports .LargeObjects() on a pgx.Tx outside
// internal/tenancy, so code that passes lint cannot reach this (TICK-698). A
// feature that needs large objects needs its own design for setting the
// tenant first.
func (t *lazyTx) LargeObjects() pgx.LargeObjects {
	panic(errLargeObjects)
}

func (t *lazyTx) Prepare(ctx context.Context, name, sql string) (*pgconn.StatementDescription, error) {
	if err := t.begin(ctx); err != nil {
		return nil, err
	}
	return t.conn.Prepare(ctx, name, sql)
}

func (t *lazyTx) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	if err := t.begin(ctx); err != nil {
		return pgconn.CommandTag{}, err
	}
	return t.conn.Exec(ctx, sql, args...)
}

func (t *lazyTx) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	if err := t.begin(ctx); err != nil {
		return nil, err
	}
	return t.conn.Query(ctx, sql, args...)
}

func (t *lazyTx) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	if err := t.begin(ctx); err != nil {
		return errRow{err}
	}
	return t.conn.QueryRow(ctx, sql, args...)
}

func (t *lazyTx) Conn() *pgx.Conn { return t.conn }

// lazySavepoint is a savepoint of a lazyTx, as pgx's own: its Begin opens
// another savepoint of the transaction, Commit releases it and Rollback
// rolls back to it.
type lazySavepoint struct {
	*lazyTx
	n      int64
	closed bool
}

func (s *lazySavepoint) name() string { return "sp_" + strconv.FormatInt(s.n, 10) }

func (s *lazySavepoint) Begin(ctx context.Context) (pgx.Tx, error) {
	if s.closed {
		return nil, pgx.ErrTxClosed
	}
	return s.lazyTx.Begin(ctx)
}

func (s *lazySavepoint) Commit(ctx context.Context) error {
	if s.closed {
		return pgx.ErrTxClosed
	}
	_, err := s.Exec(ctx, "release savepoint "+s.name())
	s.closed = true
	return err
}

func (s *lazySavepoint) Rollback(ctx context.Context) error {
	if s.closed {
		return pgx.ErrTxClosed
	}
	_, err := s.Exec(ctx, "rollback to savepoint "+s.name())
	s.closed = true
	return err
}

// firstResults hides the first batch's BEGIN and tenant results from the
// caller, who reads its own statements' results only. Close reads them
// itself (they have no callback).
type firstResults struct {
	pgx.BatchResults
	pending int
}

func (r *firstResults) skip() error {
	for r.pending > 0 {
		r.pending--
		if _, err := r.BatchResults.Exec(); err != nil {
			r.pending = 0
			return err
		}
	}
	return nil
}

func (r *firstResults) Exec() (pgconn.CommandTag, error) {
	if err := r.skip(); err != nil {
		return pgconn.CommandTag{}, err
	}
	return r.BatchResults.Exec()
}

func (r *firstResults) Query() (pgx.Rows, error) {
	if err := r.skip(); err != nil {
		return nil, err
	}
	return r.BatchResults.Query()
}

func (r *firstResults) QueryRow() pgx.Row {
	if err := r.skip(); err != nil {
		return errRow{err}
	}
	return r.BatchResults.QueryRow()
}

// errBatch is the BatchResults of a batch that was not sent.
type errBatch struct{ err error }

func (e errBatch) Exec() (pgconn.CommandTag, error) { return pgconn.CommandTag{}, e.err }
func (e errBatch) Query() (pgx.Rows, error)         { return nil, e.err }
func (e errBatch) QueryRow() pgx.Row                { return errRow(e) }
func (e errBatch) Close() error                     { return e.err }

// errRow is a pgx.Row whose Scan returns err.
type errRow struct{ err error }

func (e errRow) Scan(...any) error { return e.err }

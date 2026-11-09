// Package tenancy runs tenant-scoped database work. InTenantTx is the only
// way platform code sets app.tenant_id, the setting the row-level security
// policies read; the navalint setconfig analyzer fails the build on
// set_config or app.tenant_id in Go code outside this package (PG-N1, R8).
//
// The setting is transaction-local (set_config(..., true)), so it ends with
// the transaction: through PgBouncer in transaction mode the next transaction
// on the same server connection, which may belong to another client, starts
// with no tenant.
//
// The transaction's BEGIN makes the setting itself (TICK-648): one round trip
// opens the transaction and scopes it, so no statement fn sends, alone or in
// a pgx.Batch, can run in the transaction before the tenant is set. See
// docs/architecture/tenant-tx-batching.md.
package tenancy

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNoTenant is returned for uuid.Nil. No transaction is opened.
var ErrNoTenant = errors.New("tenancy: no tenant")

// beginSQL opens the transaction and scopes it to tenant in one round trip.
// pgx runs a BeginQuery with no arguments over the simple protocol, which
// takes several statements, and fails the begin (closing the connection) if
// either fails. A simple query has no bind parameters, so the tenant is a
// literal: uuid.UUID.String(), 36 characters of hex and hyphens, never caller
// text, so nothing can break out of the quotes. The tracer does not export
// statement text (TestTracedTxExportsNoValues).
func beginSQL(tenant uuid.UUID) string {
	return beginPrefix + tenant.String() + beginSuffix
}

const (
	beginPrefix = "BEGIN; SELECT set_config('app.tenant_id', '"
	beginSuffix = "', true)"
)

// BeginTenant reports whether sql is the statement InTenantTx opens a
// transaction with, and the tenant it sets. It is for tracers that check
// the GUC ordering from the driver's side (tenancytest.Guard), so the
// statement's text stays owned by this package.
func BeginTenant(sql string) (uuid.UUID, bool) {
	rest, ok := strings.CutPrefix(sql, beginPrefix)
	if !ok {
		return uuid.Nil, false
	}
	id, ok := strings.CutSuffix(rest, beginSuffix)
	if !ok {
		return uuid.Nil, false
	}
	tenant, err := uuid.Parse(id)
	if err != nil || tenant == uuid.Nil || beginSQL(tenant) != sql {
		return uuid.Nil, false
	}
	return tenant, true
}

// InTenantTx runs fn in a transaction whose app.tenant_id is tenantID, and
// commits if fn returns nil. It rolls back if fn returns an error, which it
// returns unwrapped, or panics, which it re-raises after the rollback. If the
// rollback itself fails, its error is joined to fn's; the connection is then
// destroyed, not returned to the pool, so the tenant cannot leak to the next
// transaction.
//
// The tx fn gets knows its tenant (TenantOf answers without a round trip),
// holds a per-transaction memo (Remember, Recall) and can queue statements
// for the commit's round trip (ExecAtCommit).
//
// tenantID must come from the request's resolved principal, never from a
// path, query or body.
func InTenantTx(ctx context.Context, pool *pgxpool.Pool, tenantID uuid.UUID, fn func(pgx.Tx) error) (err error) {
	if tenantID == uuid.Nil {
		return ErrNoTenant
	}
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("tenancy: begin: %w", err)
	}
	// Release puts the connection back in the pool only when it is idle and
	// out of any transaction (TxStatus 'I'); otherwise it destroys it.
	// Deferred first, so it runs after the rollback below.
	defer conn.Release()
	tx, err := conn.BeginTx(ctx, pgx.TxOptions{BeginQuery: beginSQL(tenantID)})
	if err != nil {
		return fmt.Errorf("tenancy: begin: %w", err)
	}
	st := &scopedTx{Tx: tx, tenant: tenantID, memo: map[any]any{}}
	// Rollback after Commit is a no-op for pgx, and only a warning for
	// PostgreSQL after a batched COMMIT, so one deferred call covers the
	// error return, the panic and a failed commit.
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}
		if err != nil {
			// ErrTxClosed: the transaction already ended (a failed commit).
			if rbErr := tx.Rollback(ctx); rbErr != nil && !errors.Is(rbErr, pgx.ErrTxClosed) {
				err = errors.Join(err, fmt.Errorf("tenancy: rollback: %w", rbErr))
			}
		}
	}()
	if err := fn(st); err != nil {
		return err
	}
	if len(st.atCommit) == 0 {
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("tenancy: commit: %w", err)
		}
		return nil
	}
	return st.commitBatch(ctx, conn.Conn().PgConn())
}

// atCommit is a statement queued with ExecAtCommit.
type atCommit struct {
	sql  string
	args []any
}

// scopedTx is the transaction InTenantTx hands fn: pgx's own, plus the tenant
// its BEGIN set, the memo and the statements queued for the commit. Only
// InTenantTx builds one, so holding one proves the tenant was set at BEGIN.
// Like pgx.Tx it is not safe for concurrent use.
type scopedTx struct {
	pgx.Tx
	tenant   uuid.UUID
	memo     map[any]any
	atCommit []atCommit
	// nested: a savepoint (Begin). It shares the parent's tenant and memo but
	// cannot queue statements for the commit: rolling the savepoint back
	// would not take them back.
	nested bool
}

// Begin opens a savepoint that keeps the tenant and the memo.
func (s *scopedTx) Begin(ctx context.Context) (pgx.Tx, error) {
	sp, err := s.Tx.Begin(ctx)
	if err != nil {
		return nil, err
	}
	return &scopedTx{Tx: sp, tenant: s.tenant, memo: s.memo, nested: true}, nil
}

// commitBatch sends the queued statements and COMMIT as one batch: one round
// trip. A failing statement makes PostgreSQL skip the rest of the batch,
// COMMIT included, and InTenantTx's deferred rollback ends the transaction.
func (s *scopedTx) commitBatch(ctx context.Context, conn *pgconn.PgConn) error {
	b := &pgx.Batch{}
	for _, q := range s.atCommit {
		b.Queue(q.sql, q.args...)
	}
	b.Queue("COMMIT").Exec(func(ct pgconn.CommandTag) error {
		// PostgreSQL answers the COMMIT of an aborted transaction with ROLLBACK.
		if ct.String() == "ROLLBACK" {
			return pgx.ErrTxCommitRollback
		}
		return nil
	})
	if err := s.Tx.SendBatch(ctx, b).Close(); err != nil {
		return fmt.Errorf("tenancy: commit: %w", err)
	}
	// pgx's tx object never saw this COMMIT. The connection must be out of
	// any transaction, or Release destroys it.
	if st := conn.TxStatus(); st != 'I' {
		return fmt.Errorf("tenancy: commit: connection left in transaction status %q", st)
	}
	return nil
}

// ExecAtCommit queues sql to run in the round trip that commits tx, after
// everything fn sent and immediately before COMMIT, in the order queued. It
// returns false, queuing nothing, when tx is not an InTenantTx transaction or
// is a savepoint; the caller then runs the statement itself.
//
// The statement's result is not returned: by the time it is read the
// transaction has committed. A statement that must stop the commit has to
// fail in SQL, and one whose row count matters must be made certain by the
// caller first, for example by locking the row earlier in tx.
func ExecAtCommit(tx pgx.Tx, sql string, args ...any) bool {
	s, ok := tx.(*scopedTx)
	if !ok || s.nested {
		return false
	}
	s.atCommit = append(s.atCommit, atCommit{sql: sql, args: args})
	return true
}

// Remember stores v under key in tx's memo, for Recall later in the same
// transaction: a read prefetched in a batch is then used by the code that
// would otherwise read it again. key must be a comparable value of a type
// the calling package owns, as with context keys. The memo dies with tx,
// which is one tenant's, so nothing in it crosses a tenant or a transaction.
// A tx InTenantTx did not open has no memo, and Remember does nothing.
//
// Memo only reads whose result this transaction does not change later, or
// Forget the key in the code that changes it.
func Remember(tx pgx.Tx, key, v any) {
	if s, ok := tx.(*scopedTx); ok {
		s.memo[key] = v
	}
}

// Recall returns what Remember stored under key in tx, if anything.
func Recall(tx pgx.Tx, key any) (any, bool) {
	s, ok := tx.(*scopedTx)
	if !ok {
		return nil, false
	}
	v, found := s.memo[key]
	return v, found
}

// Forget drops key from tx's memo.
func Forget(tx pgx.Tx, key any) {
	if s, ok := tx.(*scopedTx); ok {
		delete(s.memo, key)
	}
}

// currentTenantSQL reads the transaction's tenant; empty when none is set.
const currentTenantSQL = "SELECT coalesce(current_setting('app.tenant_id', true), '')"

// TenantOf returns the tenant InTenantTx scoped tx to, or ErrNoTenant when tx
// has none. Code that must name the tenant in a statement, such as the record
// write pipeline, takes it from here rather than from its caller, so a
// command can never write as another tenant than its transaction's.
//
// For a tx InTenantTx opened it answers, with no round trip, with the tenant
// that transaction's BEGIN set. Any other tx is asked for its setting.
func TenantOf(ctx context.Context, tx pgx.Tx) (uuid.UUID, error) {
	if s, ok := tx.(*scopedTx); ok {
		return s.tenant, nil
	}
	var s string
	if err := tx.QueryRow(ctx, currentTenantSQL).Scan(&s); err != nil {
		return uuid.Nil, fmt.Errorf("tenancy: read tenant: %w", err)
	}
	if s == "" {
		return uuid.Nil, ErrNoTenant
	}
	id, err := uuid.Parse(s)
	if err != nil || id == uuid.Nil {
		return uuid.Nil, ErrNoTenant
	}
	return id, nil
}

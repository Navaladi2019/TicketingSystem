package tenancy

// syncCommitOffSQL is WithSynchronousCommitOff's first statement. Like
// alloc.go's, it is the only place this setting is named outside the
// allocator: the navalint setconfig analyzer reports it anywhere else.
const syncCommitOffSQL = "SET LOCAL synchronous_commit = off"

// WithSynchronousCommitOff makes InTenantTx run SET LOCAL synchronous_commit
// = off as the first statement of the transaction, before fn. The commit then
// returns before its WAL is flushed: a server crash can lose the transaction,
// never half of it. SET LOCAL ends with the transaction, so the connection's
// next one (through PgBouncer, maybe another client's) runs at the server
// default.
//
// Only the schema backfill uses it, for each record's own transaction
// (ADR-009 D7, TICK-920): a rewrite lost in a crash leaves its id pending, and
// the next pass repeats it. Use it for nothing that must survive a crash once
// it has committed.
func WithSynchronousCommitOff() TxOption {
	return func(o *txOptions) { o.synchronousCommitOff = true }
}

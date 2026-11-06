package db

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// ParseFailureCloser is a pgx tracer that sends a protocol Close for a named
// statement whose Parse the server rejected (TICK-550).
//
// PgBouncer in transaction mode with max_prepared_statements > 0 (1.24.1
// through 1.26.1, the latest tested) records a client's statement name when
// it forwards the Parse. If the Parse fails inside a transaction (42883,
// 42P01, ...), it keeps that name. pgx does not cache the failed statement,
// so the next run of the same SQL sends a Parse with the same name, and
// PgBouncer closes the client with 08P01 "prepared statement name is already
// in use". A Close of the name makes PgBouncer forget it. PostgreSQL answers
// a Close of a statement that does not exist with CloseComplete, so the cost
// is one round trip per failed Parse and nothing on success.
//
// Only named statements are affected: QueryExecModeCacheStatement and
// explicit Prepare. Unnamed statements (QueryExecModeExec, CacheDescribe)
// skip it. NewPool installs it on every pool.
type ParseFailureCloser struct{}

type prepareStartKey struct{}

// TraceQueryStart implements pgx.QueryTracer, which multitracer.New needs.
func (ParseFailureCloser) TraceQueryStart(ctx context.Context, _ *pgx.Conn, _ pgx.TraceQueryStartData) context.Context {
	return ctx
}

// TraceQueryEnd implements pgx.QueryTracer.
func (ParseFailureCloser) TraceQueryEnd(context.Context, *pgx.Conn, pgx.TraceQueryEndData) {}

// TracePrepareStart remembers the statement name for TracePrepareEnd.
func (ParseFailureCloser) TracePrepareStart(ctx context.Context, _ *pgx.Conn, data pgx.TracePrepareStartData) context.Context {
	return context.WithValue(ctx, prepareStartKey{}, data)
}

// TracePrepareEnd closes the statement name when the server rejected its
// Parse. A failed Close is ignored: at worst the connection behaves as it
// would without this tracer.
func (ParseFailureCloser) TracePrepareEnd(ctx context.Context, conn *pgx.Conn, data pgx.TracePrepareEndData) {
	var pe *pgconn.PrepareError
	if !errors.As(data.Err, &pe) || pe.ParseComplete {
		// Success, a non-server error, or a failed Describe, which pgx
		// deallocates itself.
		return
	}
	start, ok := ctx.Value(prepareStartKey{}).(pgx.TracePrepareStartData)
	if !ok {
		return
	}
	if name := wireName(start.Name, start.SQL); name != "" {
		_ = conn.PgConn().Deallocate(ctx, name)
	}
}

// wireName is the name pgx sends in Parse for Conn.Prepare(name, sql): the
// name itself, or a digest when the name is the SQL text (pgx v5 conn.go).
func wireName(name, sql string) string {
	if name != "" && name == sql {
		digest := sha256.Sum256([]byte(sql))
		return "stmt_" + hex.EncodeToString(digest[0:24])
	}
	return name
}

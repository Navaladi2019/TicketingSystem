// Package pooltest holds the PgBouncer load test (T06, build tag
// integration): pgx in its default exec mode, many clients interleaved over
// a small transaction-mode pool; see load_test.go. parse_error_test.go is the
// TICK-550 regression test: a Parse error inside a transaction, retried,
// must not get the client closed by PgBouncer (db.ParseFailureCloser).
package pooltest

// Package seal is the TICK-581 fixture of the D-3-6 gate
// (internal/records/arch_test.go, TestSQLConstantUsedElsewhere). The test
// scans this tree as internal/..., so this file stands in for
// internal/audit/seal/seal.go, the one file allowed to UPDATE audit_log, and
// only the seal columns. Lines marked `// want: <op> <table>` must be
// reported; nothing else may be. It lives under testdata, so it is never built.
package seal

import "context"

type execer interface {
	Exec(ctx context.Context, sql string, args ...any) error
}

// Allowed here: the seal transition, used in this file too.
const sealSQL = `UPDATE audit_log a
SET seq = u.seq, prev_hash = u.prev_hash, entry_hash = u.entry_hash, sealed_at = now()
FROM unnest($2::uuid[], $3::bigint[]) AS u(id, seq)
WHERE a.tenant_id = $1 AND a.id = u.id AND a.sealed_at IS NULL`

// Exported, so another package can reach it: allowed here, reported there.
const SealSQL = `UPDATE audit_log SET sealed_at = now() WHERE id = $1`

// A variable holds SQL as well as a constant does.
var rewriteVar = "UPDATE audit_log SET seq = $2 WHERE id = $1"

// Not the seal columns: the seal.go allowance does not cover it.
const scrubSQL = `UPDATE audit_log SET diff = NULL, seq = $2 WHERE id = $1` // want: update audit_log

// Row form: the gate cannot read the columns, so it reports it.
const rowSQL = `UPDATE audit_log SET (seq, sealed_at) = ($2, now()) WHERE id = $1` // want: update audit_log

func seal(ctx context.Context, tx execer) error {
	return tx.Exec(ctx, sealSQL)
}

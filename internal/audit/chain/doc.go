// Package chain is the hash-chain contract of the audit log (ADR-004 R13,
// PLAN-005 C1).
//
// It turns an audit entry's envelope (the R1 envelope columns, with
// diff_truncated added by ADR-004-E1) into its canonical bytes and hashes:
//
//	envelope_sha256 = SHA-256(JCS(envelope))
//	entry_hash      = SHA-256(prev_hash ‖ envelope_sha256)   Genesis = 32 zero bytes
//	diff_sha256     = SHA-256(diff_salt ‖ diff bytes)
//
// JCS is RFC 8785, restricted to the value kinds the envelope uses: strings,
// integers, booleans and null (no floats, so no ES6 number formatting). The
// value mapping (RFC 3339 UTC timestamps with microseconds, lowercase hex for
// bytea, lowercase UUIDs, NULL emitted and never omitted) is pinned in
// docs/api/audit-export-v1.md.
//
// The writer (audit.Append), the sealer and the offline verifier all compute
// hashes with this package, and all three test against the golden vectors in
// testdata/vectors.json. Rebuild them with
//
//	go run ./internal/audit/chain/cmd/genvectors
//
// Changing the canonical form or the vectors is a breaking change to the
// export format and needs Technical Lead review (PLAN-005 D-5-7).
//
// The package does no I/O and imports no database or network code, so the
// offline verifier can link it (PLAN-005 D-5-6).
package chain

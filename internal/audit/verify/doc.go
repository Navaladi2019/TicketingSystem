// Package verify checks a nava.audit.export/v1 JSONL file offline
// (ADR-004 R20, R22, R24; R7 for redacted diffs; docs/api/audit-export-v1.md
// §4). It is the engine of `nava audit verify <file>` (PLAN-005 C4, P5-03).
//
// Verify streams the file line by line and returns one Report: the file
// outcome (pass, tampered, pending, unverifiable or empty), the first failing
// line and its reason code when tampered, and the per-seq outcome of every
// sealed line that is not plainly verified (diff_unverified for a redacted
// diff, pending for an erased entry whose audit/erase entry is not sealed
// yet, tampered for the failing line).
//
// Hashes are recomputed with internal/audit/chain, never read from the file.
// The package does no I/O beyond the reader it is given and imports no
// database or network code, so the offline CLI can link it (PLAN-005 D-5-6;
// arch_test.go enforces it).
//
// What a pass proves, and what it does not, is in audit-export-v1.md §4
// step 9 and in `nava audit verify -h`.
package verify

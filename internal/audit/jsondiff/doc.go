// Package jsondiff builds the RFC 6902 JSON Patch stored in audit_log.diff
// (ADR-004 R1, PLAN-005 P5-12c).
//
// Diff(before, after) returns the ops that turn before into after. The output
// bytes are the contract: diff_sha256 = SHA-256(diff_salt ‖ diff bytes), and
// the audit writer (P5-13b) and the S8 publish path (P5-14) must hash the same
// bytes for the same change. So the output is fully determined by the two
// documents' JSON values:
//
//   - Only add, remove and replace are emitted; no move, copy or test.
//   - Object members are visited in RFC 8785 order (UTF-16 code units of the
//     key), whatever order the input used.
//   - Arrays are diffed index by index: shared indices recurse, extra after
//     elements are added in ascending index order, extra before elements are
//     removed from the highest index down. Reorders show as replaces; there
//     is no move or copy detection.
//   - A value that changes kind (object, array, string, number, boolean,
//     null) is replaced whole.
//   - The output is compact: no whitespace. Each op is written as
//     {"op":…,"path":…,"value":…} in that member order, with "value" left out
//     of remove. Values are re-encoded, never copied: object members sorted as
//     above, strings escaped as RFC 8785 §3.2.2.2 (only '"', '\\' and
//     U+0000..U+001F; '<', '>', '&', U+2028, U+2029 and non-ASCII stay raw
//     UTF-8). Numbers keep their input literal, so 1 and 1.0 are different
//     values and their change is a replace.
//   - A nil (empty) before is a create: one add at path "". Equal documents
//     give [].
//
// The output is a JSON array of whole ops, so a caller can split it into
// []json.RawMessage and rejoin a prefix with "," to cut it at an op boundary
// (ADR-004-E1); truncation itself lives in the writer (P5-12b), not here.
//
// Inputs must each be one valid UTF-8 JSON value. Duplicate object keys keep
// the last member, as encoding/json does. The package does no I/O.
package jsondiff

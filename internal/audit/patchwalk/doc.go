// Package patchwalk is the one JSON Patch walker of the audit log (ADR-004
// R5, PLAN-005 P5-12a).
//
// Walk parses an RFC 6902 patch and calls fn for every pointer the patch
// touches:
//
//   - once per op for its "path" (Field = FieldPath, v = nil);
//   - once per copy or move op for its "from" (Field = FieldFrom, v = nil);
//   - for an add, replace or test op, once for the "value" at "path" and once
//     for every nested member and element below it, parents before children,
//     in document order (Field = FieldValue, v = the raw JSON of that node).
//
// test ops are walked like any other op: a test value is stored in the diff
// and can carry a secret as easily as an add.
//
// The parser is stricter than RFC 6902 on purpose, because the audit log
// stores the diff as Postgres json (verbatim bytes) and a walker that skips
// bytes the store keeps would let a value through unchecked:
//
//   - a duplicate object key, at any depth, is an error (encoding/json keeps
//     only the last one);
//   - an op member other than op, path, from (copy and move) and value (add,
//     replace and test) is an error, so a value cannot hide under "Value" or an
//     unknown member;
//   - invalid UTF-8, trailing data and nesting deeper than MaxDepth are errors.
//
// Every parse error wraps ErrMalformed and names the op index and member
// only, never a value. The package is pure Go and does no I/O.
package patchwalk

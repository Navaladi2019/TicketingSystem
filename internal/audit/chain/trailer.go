package chain

// TrailerVerifiable returns the export trailer's verifiable member
// (audit-export-v1.md §3 trailer, ADR-004-E2 decisions 3 and 4): true only
// when the range has an anchor, is non-empty (lastSeq > anchorSeq), and every
// seq in anchorSeq+1..lastSeq has a line (count+stubCount == lastSeq-anchorSeq).
//
// The exporter calls it once, at the end of the stream, with the counts it
// kept while writing; it needs no buffering and no pre-count query.
//
//   - anchorPresent: the export wrote an anchor line (false for an empty range).
//   - count: full sealed range lines only. Unsealed lines are not counted.
//   - stubCount: stub lines; the anchor is not one.
//   - lastSeq: the head seq read at the START of the export
//     (audit_chain_head.seq), never the seq of the last line written.
//     Passing the last written seq hides a tail gap: with range 1..5 and
//     seq 5 excluded, lastSeq 4 would give true where the correct 5 gives
//     false.
//   - anchorSeq: the anchor line's seq (0 for genesis).
func TrailerVerifiable(anchorPresent bool, count, stubCount, lastSeq, anchorSeq int64) bool {
	return anchorPresent && lastSeq > anchorSeq && count+stubCount == lastSeq-anchorSeq
}

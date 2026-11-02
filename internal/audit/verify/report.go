package verify

// Outcome is the verdict for a whole file (audit-export-v1.md §4).
type Outcome string

// File outcomes. Pending is the R24 outcome lifted to the file: nothing is
// tampered, but an erased entry is vouched for only by an audit/erase entry
// that is not sealed yet, so the file must be verified again later.
const (
	OutcomePass         Outcome = "pass"
	OutcomeTampered     Outcome = "tampered"
	OutcomePending      Outcome = "pending"
	OutcomeUnverifiable Outcome = "unverifiable"
	OutcomeEmpty        Outcome = "empty"
)

// LineOutcome is the verdict for one sealed line (audit-export-v1.md §4).
type LineOutcome string

// Line outcomes.
const (
	LineVerified       LineOutcome = "verified"
	LineDiffUnverified LineOutcome = "diff_unverified"
	LinePending        LineOutcome = "pending"
	LineTampered       LineOutcome = "tampered"
)

// Reason codes for a tampered file. The first eight are the codes the golden
// vectors use (internal/audit/chain/vectors); the rest name the structural
// checks of §4 steps 1-3, 6, 8 and 10 that the vectors leave unnamed.
const (
	ReasonEnvelopeMismatch       = "envelope_sha256_mismatch"
	ReasonDiffMismatch           = "diff_sha256_mismatch"
	ReasonEntryHashMismatch      = "entry_hash_mismatch"
	ReasonHeaderTenantMismatch   = "header_tenant_mismatch"
	ReasonDiffMissing            = "diff_missing"
	ReasonDiffStateMismatch      = "diff_state_mismatch"
	ReasonVerifiableFlagMismatch = "verifiable_flag_mismatch"
	ReasonErasureUnlisted        = "erasure_unlisted"

	// ReasonMalformedLine: not one strict JSON object of its kind (step 10),
	// a blank line, a CR line end, or invalid UTF-8.
	ReasonMalformedLine = "malformed_line"
	// ReasonUnknownKind: a kind other than header, anchor, entry, stub, trailer.
	ReasonUnknownKind = "unknown_kind"
	// ReasonHeaderInvalid: the first line is not a v1 header.
	ReasonHeaderInvalid = "header_invalid"
	// ReasonLineOrder: a line out of the §3 order (a second header, a range
	// line after an unsealed line, anything after the trailer).
	ReasonLineOrder = "line_order"
	// ReasonTrailerMissing: the file ends without a trailer.
	ReasonTrailerMissing = "trailer_missing"
	// ReasonAnchorMissing: the first chain line is not the anchor (R20).
	ReasonAnchorMissing = "anchor_missing"
	// ReasonAnchorInvalid: an anchor whose own members disagree (genesis with
	// seq not 0, a non-genesis anchor without hashes).
	ReasonAnchorInvalid = "anchor_invalid"
	// ReasonAnchorMismatch: the anchor is not at first_seq - 1 as the header
	// filter fixes it (after_seq, or genesis when the window has no lower
	// bound), or the trailer's anchor_seq names another seq (R20).
	ReasonAnchorMismatch = "anchor_mismatch"
	// ReasonSeqDuplicate: a seq that already appeared (R15).
	ReasonSeqDuplicate = "seq_duplicate"
	// ReasonSeqOutOfOrder: a seq lower than the one before it.
	ReasonSeqOutOfOrder = "seq_out_of_order"
	// ReasonSeqMissing: a gap in seq in a file whose header says stubs=true,
	// so every seq of the range must appear (R15, R20).
	ReasonSeqMissing = "seq_missing"
	// ReasonLineStateMismatch: a line's sealed / in_verified_range / seq
	// members contradict each other.
	ReasonLineStateMismatch = "line_state_mismatch"
	// ReasonCountMismatch: the trailer's counts disagree with the lines.
	ReasonCountMismatch = "count_mismatch"
	// ReasonHeadMismatch: the range does not end at last_seq / head_hash (step 6).
	ReasonHeadMismatch = "head_mismatch"
)

// Finding locates the first check that failed.
type Finding struct {
	// Line is the 1-based line number in the file.
	Line int
	// Seq is the line's seq, when it has one.
	Seq *int64
	// Reason is one of the Reason* codes.
	Reason string
	// Detail says what disagreed. It names members, never values.
	Detail string
}

// Report is the result of verifying one file.
type Report struct {
	Outcome Outcome
	// Failure is set exactly when Outcome is OutcomeTampered.
	Failure *Finding
	// Lines holds, by seq, every sealed line whose outcome is not
	// LineVerified, up to the point verification stopped.
	Lines map[int64]LineOutcome

	// The range as the file states it, once it has been checked.
	AnchorSeq *int64
	LastSeq   *int64
	HeadHash  string
	// Count, StubCount and UnsealedCount are the lines seen.
	Count, StubCount, UnsealedCount int
	// Gaps is the number of seq gaps (a stubs=false file).
	Gaps int
}

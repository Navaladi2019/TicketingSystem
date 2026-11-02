package verify

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/audit/chain"
)

// Format is the only export format this package verifies.
const Format = "nava.audit.export/v1"

// MaxLineBytes bounds one line. A diff is at most 64 KB stored (ADR-004-E1);
// with JSON escaping and the envelope a line stays far below this.
const MaxLineBytes = 4 << 20

// maxErasedIDs is the R23 cap on ids per audit/erase entry.
const maxErasedIDs = 10000

// Verify reads one JSONL export from r and checks it. The error is non-nil
// only when r fails; a file that is malformed or does not verify is reported
// in the Report.
func Verify(r io.Reader) (*Report, error) {
	v := &verifier{rep: &Report{Lines: map[int64]LineOutcome{}}, contiguous: true,
		erases: map[string]map[string]bool{}, pendingErases: map[string]map[string]bool{}}
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64<<10), MaxLineBytes)
	sc.Split(splitLF)
	for !v.done() && sc.Scan() {
		v.lineNo++
		v.line(sc.Bytes())
	}
	if err := sc.Err(); err != nil {
		if errors.Is(err, bufio.ErrTooLong) {
			v.lineNo++
			v.fail(nil, ReasonMalformedLine, fmt.Sprintf("line longer than %d bytes", MaxLineBytes))
			return v.rep, nil
		}
		return nil, err
	}
	if !v.done() && v.phase != afterTrailer {
		v.finish()
	}
	return v.rep, nil
}

// splitLF splits on "\n" only, so a CR before it stays in the line and is
// rejected (bufio.ScanLines would silently drop it).
func splitLF(data []byte, atEOF bool) (int, []byte, error) {
	if i := bytes.IndexByte(data, '\n'); i >= 0 {
		return i + 1, data[:i], nil
	}
	if atEOF && len(data) > 0 {
		return len(data), data, nil
	}
	return 0, nil, nil
}

type phase int

const (
	wantHeader phase = iota // nothing read yet
	wantRange               // header read; the anchor, an unsealed line or the trailer may follow
	inRange                 // anchor read; range lines follow
	inUnsealed              // an unsealed line read; only unsealed lines or the trailer follow
	afterTrailer
)

type erased struct {
	line int
	seq  int64
	id   string
	ref  string
}

type verifier struct {
	rep    *Report
	lineNo int
	phase  phase
	header headerLine
	tenant uuid.UUID

	anchorSeq  int64
	prevSeq    int64
	prevHash   string
	contiguous bool

	erasedEntries []erased
	// erases and pendingErases map an erasure_ref to the ids its sealed
	// (respectively unsealed) audit/erase entries list.
	erases, pendingErases map[string]map[string]bool
}

func (v *verifier) done() bool { return v.rep.Outcome == OutcomeTampered }

func (v *verifier) fail(seq *int64, reason, detail string) {
	f := &Finding{Line: v.lineNo, Reason: reason, Detail: detail}
	if seq != nil {
		s := *seq
		f.Seq = &s
		v.rep.Lines[s] = LineTampered
	}
	v.rep.Outcome, v.rep.Failure = OutcomeTampered, f
}

func (v *verifier) line(b []byte) {
	if v.phase == afterTrailer {
		v.fail(nil, ReasonLineOrder, "line after the trailer")
		return
	}
	if len(b) == 0 {
		v.fail(nil, ReasonMalformedLine, "blank line")
		return
	}
	if b[len(b)-1] == '\r' {
		v.fail(nil, ReasonMalformedLine, `CR line end (lines end with "\n" only)`)
		return
	}
	if !utf8.Valid(b) {
		v.fail(nil, ReasonMalformedLine, "invalid UTF-8")
		return
	}
	kind, _, err := lineKind(b)
	if errors.Is(err, errUnknownKind) {
		v.fail(nil, ReasonUnknownKind, fmt.Sprintf("kind %q", kind))
		return
	}
	if err != nil {
		if v.phase == wantHeader {
			v.fail(nil, ReasonHeaderInvalid, err.Error())
			return
		}
		v.fail(nil, ReasonMalformedLine, fmt.Sprintf("%s line: %v", kind, err))
		return
	}
	if v.phase == wantHeader && kind != "header" {
		v.fail(nil, ReasonHeaderInvalid, "the first line is not a header")
		return
	}
	switch kind {
	case "header":
		v.onHeader(b)
	case "anchor":
		var a anchorLine
		if err := decodeAs(b, &a); err != nil {
			v.fail(nil, ReasonMalformedLine, "anchor line: "+err.Error())
			return
		}
		v.onAnchor(a)
	case "stub":
		var s stubLine
		if err := decodeAs(b, &s); err != nil {
			v.fail(nil, ReasonMalformedLine, "stub line: "+err.Error())
			return
		}
		v.onStub(s)
	case "entry":
		var e entryLine
		if err := decodeAs(b, &e); err != nil {
			v.fail(nil, ReasonMalformedLine, "entry line: "+err.Error())
			return
		}
		if e.Sealed {
			v.onSealedEntry(e)
		} else {
			v.onUnsealedEntry(e)
		}
	case "trailer":
		var t trailerLine
		if err := decodeAs(b, &t); err != nil {
			v.fail(nil, ReasonMalformedLine, "trailer line: "+err.Error())
			return
		}
		v.onTrailer(t)
	}
}

func (v *verifier) onHeader(b []byte) {
	if v.phase != wantHeader {
		v.fail(nil, ReasonLineOrder, "a second header")
		return
	}
	if err := decodeAs(b, &v.header); err != nil {
		v.fail(nil, ReasonHeaderInvalid, err.Error())
		return
	}
	if v.header.Format != Format {
		v.fail(nil, ReasonHeaderInvalid, fmt.Sprintf("format is not %s", Format))
		return
	}
	t, err := uuid.Parse(v.header.TenantID)
	if err != nil || t.String() != v.header.TenantID {
		v.fail(nil, ReasonHeaderInvalid, "tenant_id is not a lowercase UUID")
		return
	}
	v.tenant = t
	v.phase = wantRange
}

// expectedAnchor is first_seq - 1 when the header filter fixes it (R20,
// audit-export-v1.md §3): after_seq = n gives first_seq = n + 1, and a window
// with no lower bound starts at seq 1, so the anchor is genesis. A window with
// a lower time bound fixes first_seq by occurred_at, which a stub does not
// carry, so the anchor cannot be checked offline then.
func (v *verifier) expectedAnchor() (int64, bool) {
	f := v.header.Filter
	if f.AfterSeq != nil {
		return *f.AfterSeq, true
	}
	if f.From == nil {
		return 0, true
	}
	return 0, false
}

func (v *verifier) onAnchor(a anchorLine) {
	seq := a.Seq
	if v.phase != wantRange {
		v.fail(&seq, ReasonLineOrder, "an anchor that is not the first chain line")
		return
	}
	zero := chain.Hex(chain.Genesis[:])
	if a.Genesis {
		if a.Seq != 0 || a.PrevHash != nil || a.EnvelopeSHA256 != nil || a.EntryHash != zero {
			v.fail(nil, ReasonAnchorInvalid, "a genesis anchor has seq 0, null prev_hash and envelope_sha256, and the zero entry_hash")
			return
		}
	} else {
		if a.Seq < 1 || a.PrevHash == nil || a.EnvelopeSHA256 == nil {
			v.fail(&seq, ReasonAnchorInvalid, "a non-genesis anchor has seq >= 1, prev_hash and envelope_sha256")
			return
		}
		if !v.linkHolds(&seq, *a.PrevHash, *a.EnvelopeSHA256, a.EntryHash) {
			return
		}
	}
	if want, ok := v.expectedAnchor(); ok && a.Seq != want {
		v.fail(&seq, ReasonAnchorMismatch,
			fmt.Sprintf("the header filter puts the anchor at seq %d (R20: first_seq - 1)", want))
		return
	}
	v.anchorSeq, v.prevSeq, v.prevHash = a.Seq, a.Seq, a.EntryHash
	anchor := a.Seq
	v.rep.AnchorSeq = &anchor
	v.phase = inRange
}

// enterRange checks a range line's position and its link to the line before
// it (§4 steps 2 and 3).
func (v *verifier) enterRange(seq int64, prevHash string) bool {
	switch v.phase {
	case wantRange:
		v.fail(&seq, ReasonAnchorMissing, "the first chain line must be the anchor at first_seq - 1 (R20)")
		return false
	case inUnsealed:
		v.fail(&seq, ReasonLineOrder, "a sealed range line after an unsealed line")
		return false
	case wantHeader, inRange, afterTrailer:
	}
	switch {
	case seq == v.prevSeq:
		v.fail(&seq, ReasonSeqDuplicate, fmt.Sprintf("seq %d appears twice", seq))
		return false
	case seq < v.prevSeq:
		v.fail(&seq, ReasonSeqOutOfOrder, fmt.Sprintf("seq %d follows seq %d", seq, v.prevSeq))
		return false
	case seq > v.prevSeq+1:
		if v.header.Stubs {
			v.fail(&seq, ReasonSeqMissing,
				fmt.Sprintf("seq %d..%d missing from a stubs=true file", v.prevSeq+1, seq-1))
			return false
		}
		v.contiguous = false
		v.rep.Gaps++
	default:
		if prevHash != v.prevHash {
			v.fail(&seq, ReasonEntryHashMismatch, "prev_hash is not the entry_hash of the line before")
			return false
		}
	}
	return true
}

// linkHolds recomputes entry_hash = SHA-256(prev_hash ‖ envelope_sha256).
func (v *verifier) linkHolds(seq *int64, prevHash, envelopeSHA, entryHash string) bool {
	prev, err := chain.ParseHash32(prevHash)
	if err != nil {
		v.fail(seq, ReasonMalformedLine, "prev_hash: "+err.Error())
		return false
	}
	esha, err := chain.ParseHash32(envelopeSHA)
	if err != nil {
		v.fail(seq, ReasonMalformedLine, "envelope_sha256: "+err.Error())
		return false
	}
	if _, err := chain.ParseHash32(entryHash); err != nil {
		v.fail(seq, ReasonMalformedLine, "entry_hash: "+err.Error())
		return false
	}
	if eh := chain.EntryHash(prev, esha); chain.Hex(eh[:]) != entryHash {
		v.fail(seq, ReasonEntryHashMismatch, "entry_hash is not SHA-256(prev_hash ‖ envelope_sha256)")
		return false
	}
	return true
}

func (v *verifier) advance(seq int64, entryHash string) {
	v.prevSeq, v.prevHash = seq, entryHash
}

func (v *verifier) onStub(s stubLine) {
	seq := s.Seq
	if !v.enterRange(seq, s.PrevHash) || !v.linkHolds(&seq, s.PrevHash, s.EnvelopeSHA256, s.EntryHash) {
		return
	}
	v.rep.StubCount++
	v.advance(seq, s.EntryHash)
}

func (v *verifier) onSealedEntry(e entryLine) {
	if !e.InVerifiedRange || e.Seq == nil || e.PrevHash == nil || e.EntryHash == nil || e.SealedAt == nil {
		v.fail(e.Seq, ReasonLineStateMismatch,
			"a sealed entry has in_verified_range true and seq, prev_hash, entry_hash and sealed_at set")
		return
	}
	seq := *e.Seq
	if !v.enterRange(seq, *e.PrevHash) {
		return
	}
	if _, err := time.Parse(chain.TimeLayout, *e.SealedAt); err != nil {
		v.fail(&seq, ReasonMalformedLine, "sealed_at is not a canonical timestamp")
		return
	}
	// Step 4a: the entry belongs to the header's tenant.
	if e.Envelope.TenantID != v.tenant {
		v.fail(&seq, ReasonHeaderTenantMismatch, "envelope.tenant_id is not the header's tenant_id")
		return
	}
	// Step 4: recompute envelope_sha256 from the parsed envelope.
	got, err := chain.EnvelopeSHA256(e.Envelope)
	if err != nil {
		v.fail(&seq, ReasonMalformedLine, err.Error())
		return
	}
	if chain.Hex(got[:]) != e.EnvelopeSHA256 {
		v.fail(&seq, ReasonEnvelopeMismatch, "envelope_sha256 is not SHA-256(JCS(envelope))")
		return
	}
	// Step 5: the diff state.
	outcome, reason, detail := diffState(e)
	if reason != "" {
		v.fail(&seq, reason, detail)
		return
	}
	if outcome != LineVerified {
		v.rep.Lines[seq] = outcome
	}
	if !v.linkHolds(&seq, *e.PrevHash, e.EnvelopeSHA256, *e.EntryHash) {
		return
	}
	// Step 7 needs every erase entry in the file, so erased entries are
	// decided at the trailer.
	if e.ErasedAt != nil {
		v.erasedEntries = append(v.erasedEntries,
			erased{line: v.lineNo, seq: seq, id: e.Envelope.ID.String(), ref: *e.ErasureRef})
	}
	if outcome == LineVerified {
		recordErase(v.erases, e)
	}
	v.rep.Count++
	v.advance(seq, *e.EntryHash)
}

func (v *verifier) onUnsealedEntry(e entryLine) {
	if v.phase == inRange || v.phase == wantRange {
		v.phase = inUnsealed
	}
	if e.InVerifiedRange || e.Seq != nil || e.PrevHash != nil || e.EntryHash != nil || e.SealedAt != nil {
		v.fail(nil, ReasonLineStateMismatch,
			"an unsealed entry has in_verified_range false and null seq, prev_hash, entry_hash and sealed_at")
		return
	}
	if e.Envelope.TenantID != v.tenant {
		v.fail(nil, ReasonHeaderTenantMismatch, "envelope.tenant_id is not the header's tenant_id")
		return
	}
	// envelope_sha256 of an unsealed line is the exporter's, for information
	// only (§3). Its diff still has to match envelope.diff_sha256 for the line
	// to vouch for an erasure as pending.
	if o, reason, _ := diffState(e); reason == "" && o == LineVerified {
		recordErase(v.pendingErases, e)
	}
	v.rep.UnsealedCount++
}

// diffState is §4 step 5. It returns the line outcome, or a reason code.
func diffState(e entryLine) (LineOutcome, string, string) {
	h := e.Envelope.DiffSHA256
	if (e.ErasedAt == nil) != (e.ErasureRef == nil) {
		return "", ReasonDiffStateMismatch, "erased_at and erasure_ref are set together"
	}
	if e.ErasureRef != nil {
		if u, err := uuid.Parse(*e.ErasureRef); err != nil || u.String() != *e.ErasureRef {
			return "", ReasonDiffStateMismatch, "erasure_ref is not a lowercase UUID"
		}
	}
	switch {
	case h == nil:
		if e.Diff != nil || e.DiffSalt != nil || e.ErasedAt != nil || e.DiffRedacted {
			return "", ReasonDiffStateMismatch, "diff_sha256 is null, so diff, diff_salt and erased_at are null and diff_redacted is false"
		}
		return LineVerified, "", ""
	case e.ErasedAt != nil:
		if e.Diff != nil || e.DiffSalt != nil {
			return "", ReasonDiffStateMismatch, "an erased entry has null diff and diff_salt"
		}
		return LineVerified, "", "" // step 7 decides
	case e.DiffRedacted:
		if e.DiffSalt != nil {
			return "", ReasonDiffStateMismatch, "a redacted diff has a null diff_salt"
		}
		return LineDiffUnverified, "", ""
	default:
		if e.Diff == nil || e.DiffSalt == nil {
			return "", ReasonDiffMissing, "diff_sha256 is set but diff or diff_salt is missing on an entry that is neither erased nor redacted"
		}
		salt, err := chain.ParseSalt16(*e.DiffSalt)
		if err != nil {
			return "", ReasonMalformedLine, "diff_salt: " + err.Error()
		}
		if chain.DiffSHA256(salt, []byte(*e.Diff)) != *h {
			return "", ReasonDiffMismatch, "diff_sha256 is not SHA-256(diff_salt ‖ diff)"
		}
		return LineVerified, "", ""
	}
}

// recordErase adds the ids an audit/erase entry lists to into. The diff must
// be exactly [{"op":"add","path":"/erased_ids","value":[ids]}] with 1 to
// 10,000 lowercase ids (R23, §4 step 7); any other shape lists nothing.
func recordErase(into map[string]map[string]bool, e entryLine) {
	env := e.Envelope
	if env.Domain != "audit" || env.Action != "erase" || env.SubjectType != "erasure_request" || e.Diff == nil {
		return
	}
	var ops []struct {
		Op    string   `json:"op"`
		Path  string   `json:"path"`
		Value []string `json:"value"`
	}
	d := json.NewDecoder(bytes.NewReader([]byte(*e.Diff)))
	d.DisallowUnknownFields()
	if err := d.Decode(&ops); err != nil || len(ops) != 1 || ops[0].Op != "add" || ops[0].Path != "/erased_ids" ||
		len(ops[0].Value) == 0 || len(ops[0].Value) > maxErasedIDs {
		return
	}
	ref := env.SubjectID.String()
	if into[ref] == nil {
		into[ref] = map[string]bool{}
	}
	for _, id := range ops[0].Value {
		if u, err := uuid.Parse(id); err == nil && u.String() == id {
			into[ref][id] = true
		}
	}
}

func (v *verifier) onTrailer(t trailerLine) {
	v.phase = afterTrailer
	v.rep.LastSeq = t.LastSeq
	if t.HeadHash != nil {
		v.rep.HeadHash = *t.HeadHash
	}
	if t.UnsealedCount != v.rep.UnsealedCount {
		v.fail(nil, ReasonCountMismatch, "unsealed_count disagrees with the unsealed lines")
		return
	}
	if v.rep.AnchorSeq == nil {
		v.emptyRange(t)
		return
	}
	if t.AnchorSeq == nil || *t.AnchorSeq != v.anchorSeq {
		v.fail(nil, ReasonAnchorMismatch, "trailer anchor_seq is not the anchor line's seq")
		return
	}
	if t.Count != v.rep.Count || t.StubCount != v.rep.StubCount {
		v.fail(nil, ReasonCountMismatch, "count or stub_count disagrees with the lines")
		return
	}
	// Step 6: the range ends at the head.
	switch {
	case t.LastSeq == nil || t.HeadHash == nil:
		v.fail(nil, ReasonHeadMismatch, "a non-empty range has last_seq and head_hash")
		return
	case v.prevSeq == v.anchorSeq:
		v.fail(nil, ReasonHeadMismatch, "an anchor with no range lines after it")
		return
	case v.prevSeq > *t.LastSeq:
		v.fail(nil, ReasonHeadMismatch, "a range line beyond last_seq")
		return
	case v.contiguous && (v.prevSeq != *t.LastSeq || v.prevHash != *t.HeadHash):
		v.fail(nil, ReasonHeadMismatch, "the last range line is not last_seq with entry_hash head_hash")
		return
	}
	// Step 7: erased entries (R24).
	pending := false
	for _, x := range v.erasedEntries {
		switch {
		case v.erases[x.ref][x.id]:
		case v.pendingErases[x.ref][x.id]:
			v.rep.Lines[x.seq] = LinePending
			pending = true
		default:
			v.lineNo = x.line
			seq := x.seq
			v.fail(&seq, ReasonErasureUnlisted,
				"erased, but no chain-valid audit/erase entry for its erasure_ref lists its id (R24)")
			return
		}
	}
	// Step 8: verifiability is computed, then the flags must agree.
	if v.header.Stubs != v.contiguous || t.Verifiable != v.contiguous {
		v.fail(nil, ReasonVerifiableFlagMismatch,
			fmt.Sprintf("header stubs and trailer verifiable must both be %t for this range", v.contiguous))
		return
	}
	switch {
	case !v.contiguous:
		v.rep.Outcome = OutcomeUnverifiable
	case pending:
		v.rep.Outcome = OutcomePending
	default:
		v.rep.Outcome = OutcomePass
	}
}

func (v *verifier) emptyRange(t trailerLine) {
	switch {
	case t.AnchorSeq != nil:
		v.fail(nil, ReasonAnchorMissing, "trailer anchor_seq is set but the file has no anchor (R20)")
	case t.LastSeq != nil || t.HeadHash != nil || t.Count != 0 || t.StubCount != 0:
		v.fail(nil, ReasonCountMismatch, "an empty range has null last_seq and head_hash and zero counts")
	case t.Verifiable:
		v.fail(nil, ReasonVerifiableFlagMismatch, "an empty range is not verifiable")
	default:
		v.rep.Outcome = OutcomeEmpty
	}
}

// finish runs at end of input when no trailer was read.
func (v *verifier) finish() {
	v.lineNo++
	if v.phase == wantHeader {
		v.fail(nil, ReasonHeaderInvalid, "empty file")
		return
	}
	v.fail(nil, ReasonTrailerMissing, "the file ends without a trailer")
}

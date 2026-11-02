package vectors

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/audit/chain"
)

// Export is a whole nava.audit.export/v1 file built from the chain vectors,
// with the outcome a verifier following audit-export-v1.md §4 must reach.
// The verifier (P5-03) and its acceptance kit (TICK-425) use these as
// fixtures.
type Export struct {
	Name string `json:"name"`
	// Review is the review finding the case comes from, if any.
	Review string `json:"review"`
	// Expect is the file outcome: pass, tampered, empty or unverifiable.
	Expect string `json:"expect"`
	// Reason is the §4 check that fails, for a tampered file.
	Reason *string `json:"reason"`
	// LineOutcomes lists, by seq, every sealed line whose outcome is not
	// "verified".
	LineOutcomes map[string]string `json:"line_outcomes"`
	// JSONL holds the file's lines in order, without line ends. The file is
	// the lines joined by "\n", with a final "\n".
	JSONL []string `json:"jsonl"`
}

// File outcomes (§4).
const (
	FilePass         = "pass"
	FileTampered     = "tampered"
	FileEmpty        = "empty"
	FileUnverifiable = "unverifiable"
)

// Line outcomes (§4).
const (
	LineVerified       = "verified"
	LineDiffUnverified = "diff_unverified"
	LinePending        = "pending"
	LineTampered       = "tampered"
)

// Reasons a file is tampered (§4), beyond the Expect* hash mismatches.
const (
	ReasonHeaderTenantMismatch   = "header_tenant_mismatch"
	ReasonDiffMissing            = "diff_missing"
	ReasonDiffStateMismatch      = "diff_state_mismatch"
	ReasonVerifiableFlagMismatch = "verifiable_flag_mismatch"
	ReasonErasureUnlisted        = "erasure_unlisted"
	ReasonSeqMissing             = "seq_missing"
	ReasonAnchorMismatch         = "anchor_mismatch"
	ReasonHeadMismatch           = "head_mismatch"
)

// The line shapes of audit-export-v1.md §3. Field order is the order the
// spec's examples use.

type headerLine struct {
	Kind       string      `json:"kind"`
	Format     string      `json:"format"`
	TenantID   uuid.UUID   `json:"tenant_id"`
	ExportID   uuid.UUID   `json:"export_id"`
	ExportedAt string      `json:"exported_at"`
	Stubs      bool        `json:"stubs"`
	Filter     filterBlock `json:"filter"`
}

type filterBlock struct {
	From        *string `json:"from"`
	To          *string `json:"to"`
	Domain      *string `json:"domain"`
	SubjectType *string `json:"subject_type"`
	SubjectID   *string `json:"subject_id"`
	ChangeSetID *string `json:"change_set_id"`
	AfterSeq    *int64  `json:"after_seq"`
}

type anchorLine struct {
	Kind           string  `json:"kind"`
	Genesis        bool    `json:"genesis"`
	Seq            int64   `json:"seq"`
	PrevHash       *string `json:"prev_hash"`
	EntryHash      string  `json:"entry_hash"`
	EnvelopeSHA256 *string `json:"envelope_sha256"`
}

type entryLine struct {
	Kind            string         `json:"kind"`
	Sealed          bool           `json:"sealed"`
	InVerifiedRange bool           `json:"in_verified_range"`
	Seq             *int64         `json:"seq"`
	PrevHash        *string        `json:"prev_hash"`
	EntryHash       *string        `json:"entry_hash"`
	EnvelopeSHA256  string         `json:"envelope_sha256"`
	SealedAt        *string        `json:"sealed_at"`
	Envelope        chain.Envelope `json:"envelope"`
	Diff            *string        `json:"diff"`
	DiffSalt        *string        `json:"diff_salt"`
	DiffRedacted    bool           `json:"diff_redacted"`
	ErasedAt        *string        `json:"erased_at"`
	ErasureRef      *string        `json:"erasure_ref"`
}

type stubLine struct {
	Kind           string `json:"kind"`
	Seq            int64  `json:"seq"`
	PrevHash       string `json:"prev_hash"`
	EntryHash      string `json:"entry_hash"`
	EnvelopeSHA256 string `json:"envelope_sha256"`
}

type trailerLine struct {
	Kind          string  `json:"kind"`
	AnchorSeq     *int64  `json:"anchor_seq"`
	LastSeq       *int64  `json:"last_seq"`
	HeadHash      *string `json:"head_hash"`
	Count         int     `json:"count"`
	StubCount     int     `json:"stub_count"`
	UnsealedCount int     `json:"unsealed_count"`
	Verifiable    bool    `json:"verifiable"`
}

// exportFile is an export under construction, before it is serialised.
type exportFile struct {
	header  headerLine
	body    []any // anchorLine, *entryLine or stubLine, in order
	trailer trailerLine
}

const (
	exportedAt = "2026-11-02T12:30:00.000000Z"
	sealedAt   = "2026-11-02T12:00:00.000000Z"
)

func entryOf(c ChainCase) *entryLine {
	seq, prev, eh, at := c.Seq, c.PrevHash, c.EntryHash, sealedAt
	return &entryLine{
		Kind: "entry", Sealed: true, InVerifiedRange: true,
		Seq: &seq, PrevHash: &prev, EntryHash: &eh, EnvelopeSHA256: c.EnvelopeSHA256, SealedAt: &at,
		Envelope: c.Envelope, Diff: c.Diff, DiffSalt: c.DiffSalt,
		ErasedAt: c.ErasedAt, ErasureRef: c.ErasureRef,
	}
}

func stubOf(c ChainCase) stubLine {
	return stubLine{Kind: "stub", Seq: c.Seq, PrevHash: c.PrevHash, EntryHash: c.EntryHash, EnvelopeSHA256: c.EnvelopeSHA256}
}

// rangeExport exports seq 1..n from genesis, in full where match is true and
// as a stub otherwise.
func rangeExport(cs []ChainCase, filter filterBlock, match func(ChainCase) bool) exportFile {
	return windowExport(cs, 1, filter, match)
}

// windowExport exports seq first..len(cs) with stubs=true: the anchor is
// genesis when first is 1 and the line for first-1 otherwise, then each seq
// in full where match is true and as a stub otherwise.
func windowExport(cs []ChainCase, first int64, filter filterBlock, match func(ChainCase) bool) exportFile {
	f := exportFile{
		header: headerLine{
			Kind: "header", Format: "nava.audit.export/v1", TenantID: cs[0].Envelope.TenantID,
			ExportID: uuid.MustParse("0190f3a0-0000-7000-8000-0000000000ee"), ExportedAt: exportedAt,
			Stubs: true, Filter: filter,
		},
	}
	if first == 1 {
		f.body = []any{anchorLine{Kind: "anchor", Genesis: true, Seq: 0, EntryHash: chain.Hex(chain.Genesis[:])}}
	} else {
		a := cs[first-2]
		prev, esha := a.PrevHash, a.EnvelopeSHA256
		f.body = []any{anchorLine{Kind: "anchor", Seq: a.Seq, PrevHash: &prev, EntryHash: a.EntryHash, EnvelopeSHA256: &esha}}
	}
	anchor := first - 1
	last := cs[len(cs)-1]
	lastSeq, head := last.Seq, last.EntryHash
	f.trailer = trailerLine{Kind: "trailer", AnchorSeq: &anchor, LastSeq: &lastSeq, HeadHash: &head, Verifiable: true}
	for _, c := range cs[first-1:] {
		if match(c) {
			f.body = append(f.body, entryOf(c))
			f.trailer.Count++
		} else {
			f.body = append(f.body, stubOf(c))
			f.trailer.StubCount++
		}
	}
	return f
}

// entryAt returns the full line for seq.
func (f *exportFile) entryAt(seq int64) *entryLine {
	for _, l := range f.body {
		if e, ok := l.(*entryLine); ok && e.Seq != nil && *e.Seq == seq {
			return e
		}
	}
	panic(fmt.Sprintf("vectors: no full line for seq %d", seq))
}

// drop removes the range lines for seqs and lowers the trailer's counts to
// match. It leaves the header's stubs and the trailer's verifiable alone.
func (f *exportFile) drop(seqs ...int64) {
	gone := map[int64]bool{}
	for _, s := range seqs {
		gone[s] = true
	}
	kept := f.body[:0]
	for _, l := range f.body {
		switch x := l.(type) {
		case *entryLine:
			if x.Seq != nil && gone[*x.Seq] {
				f.trailer.Count--
				continue
			}
		case stubLine:
			if gone[x.Seq] {
				f.trailer.StubCount--
				continue
			}
		}
		kept = append(kept, l)
	}
	f.body = kept
}

func (f exportFile) lines() ([]string, error) {
	all := append(append([]any{f.header}, f.body...), f.trailer)
	out := make([]string, 0, len(all))
	for _, l := range all {
		var b bytes.Buffer
		enc := json.NewEncoder(&b)
		enc.SetEscapeHTML(false)
		if err := enc.Encode(l); err != nil {
			return nil, err
		}
		out = append(out, string(bytes.TrimSuffix(b.Bytes(), []byte("\n"))))
	}
	return out, nil
}

// exportCase is one Export before its file is serialised.
type exportCase struct {
	name, review, expect string
	reason               *string
	lines                map[string]string
	file                 exportFile
}

func exportCases(cs []ChainCase) ([]Export, error) {
	all := func(ChainCase) bool { return true }
	unfiltered := func() exportFile { return rangeExport(cs, filterBlock{}, all) }
	reason := func(s string) *string { return &s }

	var cases []exportCase

	cases = append(cases, exportCase{name: "unfiltered", expect: FilePass, file: unfiltered()})

	cases = append(cases, exportCase{
		name: "filtered-subject-type", expect: FilePass,
		file: rangeExport(cs, filterBlock{SubjectType: ptr("entity_definition")},
			func(c ChainCase) bool { return c.Envelope.SubjectType == "entity_definition" }),
	})

	{ // TA case B: diff and salt dropped from an entry that was not erased.
		f := unfiltered()
		e := f.entryAt(2)
		e.Diff, e.DiffSalt = nil, nil
		cases = append(cases, exportCase{name: "diff-dropped-not-erased", review: "TA-6B", expect: FileTampered,
			reason: reason(ReasonDiffMissing), lines: map[string]string{"2": LineTampered}, file: f})
	}
	{ // TA case C: diff replaced and marked redacted. Not tampered, but not attested.
		f := unfiltered()
		e := f.entryAt(2)
		d := `[{"op":"replace","path":"/status","value":{"redacted":"mask"}}]`
		e.Diff, e.DiffSalt, e.DiffRedacted = &d, nil, true
		cases = append(cases, exportCase{name: "diff-redacted-replaced", review: "TA-6C", expect: FilePass,
			lines: map[string]string{"2": LineDiffUnverified}, file: f})
	}
	{ // TA case D: a diff injected where the envelope has no diff_sha256.
		f := unfiltered()
		e := f.entryAt(3)
		d, s := `[{"op":"add","path":"/injected","value":1}]`, "000102030405060708090a0b0c0d0e0f"
		e.Diff, e.DiffSalt = &d, &s
		cases = append(cases, exportCase{name: "diff-injected-without-hash", review: "TA-6D", expect: FileTampered,
			reason: reason(ReasonDiffStateMismatch), lines: map[string]string{"3": LineTampered}, file: f})
	}
	{ // TA case E / Security S2: header relabelled to another tenant.
		f := unfiltered()
		f.header.TenantID = uuid.MustParse("0190f3a0-0000-7000-8000-000000000002")
		cases = append(cases, exportCase{name: "header-tenant-mismatch", review: "TA-6E, Sec-S2", expect: FileTampered,
			reason: reason(ReasonHeaderTenantMismatch), lines: map[string]string{"1": LineTampered}, file: f})
	}
	{ // TA case F: the trailer claims a complete file is not verifiable.
		f := unfiltered()
		f.header.Stubs = false
		f.trailer.Verifiable = false
		cases = append(cases, exportCase{name: "verifiable-flag-false-on-complete-file", review: "TA-6F", expect: FileTampered,
			reason: reason(ReasonVerifiableFlagMismatch), file: f})
	}
	{ // R24: an erased entry whose erase entry is only a stub in the file.
		f := rangeExport(cs, filterBlock{SubjectType: ptr("record")},
			func(c ChainCase) bool { return c.Envelope.SubjectType == "record" })
		cases = append(cases, exportCase{name: "erasure-not-listed", expect: FileTampered,
			reason: reason(ReasonErasureUnlisted), lines: map[string]string{"6": LineTampered}, file: f})
	}
	// stubs=false: excluded entries are left out, honestly flagged.
	cases = append(cases, exportCase{name: "stubs-false-honest", expect: FileUnverifiable, file: stubsFalseHonest(cs)})
	{ // No sealed entry in the window (§3 range rules, step 8).
		f := unfiltered()
		f.header.Filter = filterBlock{From: ptr("2026-12-01T00:00:00.000000Z"), To: ptr("2026-12-02T00:00:00.000000Z")}
		f.body = nil
		f.trailer = trailerLine{Kind: "trailer"}
		cases = append(cases, exportCase{name: "empty-range", expect: FileEmpty, file: f})
	}

	cases = append(cases, e2Cases(cs)...)

	out := make([]Export, 0, len(cases))
	for _, c := range cases {
		lines, err := c.file.lines()
		if err != nil {
			return nil, fmt.Errorf("export %s: %w", c.name, err)
		}
		lo := c.lines
		if lo == nil {
			lo = map[string]string{}
		}
		out = append(out, Export{Name: c.name, Review: c.review, Expect: c.expect, Reason: c.reason, LineOutcomes: lo, JSONL: lines})
	}
	return out, nil
}

// stubsFalseHonest is a stubs=false export of seq 1..7 filtered to
// entity_definition (seq 1 and 4): the excluded entries, the tail included,
// are left out and the trailer honestly says verifiable: false.
func stubsFalseHonest(cs []ChainCase) exportFile {
	f := rangeExport(cs, filterBlock{SubjectType: ptr("entity_definition")},
		func(c ChainCase) bool { return c.Envelope.SubjectType == "entity_definition" })
	withoutStubs(&f)
	f.header.Stubs = false
	f.trailer.Verifiable = false
	return f
}

// withoutStubs turns a stubs=true export into what stubs=false sends: the
// stub lines are left out and stub_count is 0.
func withoutStubs(f *exportFile) {
	kept := f.body[:0]
	for _, l := range f.body {
		if _, stub := l.(stubLine); !stub {
			kept = append(kept, l)
		}
	}
	f.body = kept
	f.trailer.StubCount = 0
}

// e2Cases are the ten export vectors of ADR-004-E2 ("What the vectors should
// say"): the header's stubs is the request, only the trailer's verifiable
// must equal the computed contiguity, a tail gap is a gap, and seq_missing
// and anchor_mismatch are named. Ranges that end at seq 5 leave out seq 6 and
// 7 (the erased entry and its audit/erase), so R24 does not fire first.
func e2Cases(cs []ChainCase) []exportCase {
	all := func(ChainCase) bool { return true }
	none := func(ChainCase) bool { return false }
	reason := func(s string) *string { return &s }
	upTo5 := cs[:5]
	var out []exportCase

	{ // stubs=false, but the filter excluded nothing: contiguous, so verifiable.
		f := rangeExport(cs, filterBlock{}, all)
		f.header.Stubs = false
		out = append(out, exportCase{name: "stubs-false-contiguous", review: "ADR-004-E2", expect: FilePass, file: f})
	}
	{ // stubs=false over 1..5; the window's upper bound excludes seq 5, the tail.
		to := "2026-11-02T10:00:04.000000Z" // seq 5 occurred at 10:00:04.5
		f := rangeExport(upTo5, filterBlock{To: &to}, func(c ChainCase) bool { return c.Seq != 5 })
		withoutStubs(&f)
		f.header.Stubs = false
		f.trailer.Verifiable = false
		out = append(out, exportCase{name: "stubs-false-tail-excluded", review: "ADR-004-E2", expect: FileUnverifiable, file: f})
	}
	{ // stubs-false-honest whose trailer claims the gapped range is verifiable.
		f := stubsFalseHonest(cs)
		f.trailer.Verifiable = true
		out = append(out, exportCase{name: "stubs-false-gap-flagged-verifiable", review: "ADR-004-E2", expect: FileTampered,
			reason: reason(ReasonVerifiableFlagMismatch), file: f})
	}
	{ // stubs=true over 1..5 with the seq 2 line deleted and the counts adjusted.
		f := rangeExport(upTo5, filterBlock{}, all)
		f.drop(2)
		out = append(out, exportCase{name: "stubs-true-seq-missing", review: "ADR-004-E2", expect: FileTampered,
			reason: reason(ReasonSeqMissing), lines: map[string]string{"3": LineTampered}, file: f})
	}
	{ // stubs=true over 1..5 with the seq 5 line (the tail) deleted.
		f := rangeExport(upTo5, filterBlock{}, all)
		f.drop(5)
		out = append(out, exportCase{name: "stubs-true-tail-missing", review: "ADR-004-E2", expect: FileTampered,
			reason: reason(ReasonSeqMissing), file: f})
	}
	{ // The header fixes the anchor at after_seq 2, but the file starts at genesis.
		f := rangeExport(cs, filterBlock{AfterSeq: ptr(int64(2))}, all)
		out = append(out, exportCase{name: "anchor-not-after-seq", review: "ADR-004-E2", expect: FileTampered,
			reason: reason(ReasonAnchorMismatch), lines: map[string]string{"0": LineTampered}, file: f})
	}
	{ // No from bound fixes the anchor at genesis, but the file starts after seq 2.
		f := windowExport(cs, 3, filterBlock{}, all)
		out = append(out, exportCase{name: "anchor-not-genesis", review: "ADR-004-E2", expect: FileTampered,
			reason: reason(ReasonAnchorMismatch), lines: map[string]string{"2": LineTampered}, file: f})
	}
	{ // stubs=false over 1..5 where no entry matched: the anchor and no range lines.
		f := rangeExport(upTo5, filterBlock{SubjectType: ptr("erasure_request")}, none)
		withoutStubs(&f)
		f.header.Stubs = false
		f.trailer.Verifiable = false
		out = append(out, exportCase{name: "stubs-false-none-matched", review: "ADR-004-E2", expect: FileUnverifiable, file: f})
	}
	{ // stubs=true over 1..5 with every range line deleted.
		f := rangeExport(upTo5, filterBlock{}, all)
		f.drop(1, 2, 3, 4, 5)
		out = append(out, exportCase{name: "stubs-true-all-missing", review: "ADR-004-E2", expect: FileTampered,
			reason: reason(ReasonSeqMissing), file: f})
	}
	{ // stubs=false over 1..5, seq 2 excluded, and the trailer's head_hash forged.
		// The last line is at last_seq, so its entry_hash is checked, gap or not.
		// No filter member excludes seq 2 alone; the header filter is not
		// checked offline, so the line is simply left out.
		f := rangeExport(upTo5, filterBlock{}, func(c ChainCase) bool { return c.Seq != 2 })
		withoutStubs(&f)
		f.header.Stubs = false
		f.trailer.Verifiable = false
		forged := cs[3].EntryHash // seq 4's entry_hash, not seq 5's
		f.trailer.HeadHash = &forged
		out = append(out, exportCase{name: "stubs-false-gap-head-forged", review: "ADR-004-E2", expect: FileTampered,
			reason: reason(ReasonHeadMismatch), file: f})
	}
	return out
}

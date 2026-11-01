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
	zero := chain.Hex(chain.Genesis[:])
	f := exportFile{
		header: headerLine{
			Kind: "header", Format: "nava.audit.export/v1", TenantID: cs[0].Envelope.TenantID,
			ExportID: uuid.MustParse("0190f3a0-0000-7000-8000-0000000000ee"), ExportedAt: exportedAt,
			Stubs: true, Filter: filter,
		},
		body: []any{anchorLine{Kind: "anchor", Genesis: true, Seq: 0, EntryHash: zero}},
	}
	var anchor int64
	last := cs[len(cs)-1]
	lastSeq, head := last.Seq, last.EntryHash
	f.trailer = trailerLine{Kind: "trailer", AnchorSeq: &anchor, LastSeq: &lastSeq, HeadHash: &head, Verifiable: true}
	for _, c := range cs {
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

func exportCases(cs []ChainCase) ([]Export, error) {
	all := func(ChainCase) bool { return true }
	unfiltered := func() exportFile { return rangeExport(cs, filterBlock{}, all) }
	reason := func(s string) *string { return &s }

	type tc struct {
		name, review, expect string
		reason               *string
		lines                map[string]string
		file                 exportFile
	}
	var cases []tc

	cases = append(cases, tc{name: "unfiltered", expect: FilePass, file: unfiltered()})

	cases = append(cases, tc{
		name: "filtered-subject-type", expect: FilePass,
		file: rangeExport(cs, filterBlock{SubjectType: ptr("entity_definition")},
			func(c ChainCase) bool { return c.Envelope.SubjectType == "entity_definition" }),
	})

	{ // TA case B: diff and salt dropped from an entry that was not erased.
		f := unfiltered()
		e := f.entryAt(2)
		e.Diff, e.DiffSalt = nil, nil
		cases = append(cases, tc{name: "diff-dropped-not-erased", review: "TA-6B", expect: FileTampered,
			reason: reason(ReasonDiffMissing), lines: map[string]string{"2": LineTampered}, file: f})
	}
	{ // TA case C: diff replaced and marked redacted. Not tampered, but not attested.
		f := unfiltered()
		e := f.entryAt(2)
		d := `[{"op":"replace","path":"/status","value":{"redacted":"mask"}}]`
		e.Diff, e.DiffSalt, e.DiffRedacted = &d, nil, true
		cases = append(cases, tc{name: "diff-redacted-replaced", review: "TA-6C", expect: FilePass,
			lines: map[string]string{"2": LineDiffUnverified}, file: f})
	}
	{ // TA case D: a diff injected where the envelope has no diff_sha256.
		f := unfiltered()
		e := f.entryAt(3)
		d, s := `[{"op":"add","path":"/injected","value":1}]`, "000102030405060708090a0b0c0d0e0f"
		e.Diff, e.DiffSalt = &d, &s
		cases = append(cases, tc{name: "diff-injected-without-hash", review: "TA-6D", expect: FileTampered,
			reason: reason(ReasonDiffStateMismatch), lines: map[string]string{"3": LineTampered}, file: f})
	}
	{ // TA case E / Security S2: header relabelled to another tenant.
		f := unfiltered()
		f.header.TenantID = uuid.MustParse("0190f3a0-0000-7000-8000-000000000002")
		cases = append(cases, tc{name: "header-tenant-mismatch", review: "TA-6E, Sec-S2", expect: FileTampered,
			reason: reason(ReasonHeaderTenantMismatch), lines: map[string]string{"1": LineTampered}, file: f})
	}
	{ // TA case F: a complete file whose flags claim it is not verifiable.
		f := unfiltered()
		f.header.Stubs = false
		f.trailer.Verifiable = false
		cases = append(cases, tc{name: "verifiable-flag-false-on-complete-file", review: "TA-6F", expect: FileTampered,
			reason: reason(ReasonVerifiableFlagMismatch), file: f})
	}
	{ // R24: an erased entry whose erase entry is only a stub in the file.
		f := rangeExport(cs, filterBlock{SubjectType: ptr("record")},
			func(c ChainCase) bool { return c.Envelope.SubjectType == "record" })
		cases = append(cases, tc{name: "erasure-not-listed", expect: FileTampered,
			reason: reason(ReasonErasureUnlisted), lines: map[string]string{"6": LineTampered}, file: f})
	}
	{ // stubs=false: excluded entries are left out, honestly flagged.
		f := rangeExport(cs, filterBlock{SubjectType: ptr("entity_definition")},
			func(c ChainCase) bool { return c.Envelope.SubjectType == "entity_definition" })
		kept := f.body[:0]
		for _, l := range f.body {
			if _, stub := l.(stubLine); !stub {
				kept = append(kept, l)
			}
		}
		f.body = kept
		f.header.Stubs = false
		f.trailer.StubCount = 0
		f.trailer.Verifiable = false
		cases = append(cases, tc{name: "stubs-false-honest", expect: FileUnverifiable, file: f})
	}
	{ // No sealed entry in the window (§3 range rules, step 8).
		f := unfiltered()
		f.header.Filter = filterBlock{From: ptr("2026-12-01T00:00:00.000000Z"), To: ptr("2026-12-02T00:00:00.000000Z")}
		f.body = nil
		f.trailer = trailerLine{Kind: "trailer"}
		cases = append(cases, tc{name: "empty-range", expect: FileEmpty, file: f})
	}

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

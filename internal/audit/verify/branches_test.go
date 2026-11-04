package verify_test

import (
	"bytes"
	"errors"
	"io"
	"strconv"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/Navaladi2019/TicketingSystem/internal/audit/chain"
	"github.com/Navaladi2019/TicketingSystem/internal/audit/chain/vectors"
	"github.com/Navaladi2019/TicketingSystem/internal/audit/verify"
	vt "github.com/Navaladi2019/TicketingSystem/internal/audit/verify/verifytest"
)

// failure is the exact first failure a broken file must report: the line
// (counted in the JSONL form; CSV adds its column row), the seq it is pinned
// to (noSeq when the finding names none), the reason code and the detail.
type failure struct {
	line   int
	seq    int64
	reason string
	detail string
}

const noSeq = -1

func wantFailure(t *testing.T, e encoding, r *verify.Report, w failure) {
	t.Helper()
	wantTampered(t, r, w.reason)
	f := r.Failure
	line := w.line + e.before - 1
	if f.Line != line || f.Reason != w.reason || f.Detail != w.detail {
		t.Fatalf("got %s\nwant line %d: %s: %s", describe(r), line, w.reason, w.detail)
	}
	switch {
	case w.seq == noSeq && f.Seq != nil:
		t.Fatalf("failure pinned to seq %d, want no seq", *f.Seq)
	case w.seq != noSeq && (f.Seq == nil || *f.Seq != w.seq):
		t.Fatalf("failure seq %v, want %d", f.Seq, w.seq)
	case w.seq != noSeq && r.Lines[w.seq] != verify.LineTampered:
		t.Fatalf("line outcome for seq %d is %q, want %q", w.seq, r.Lines[w.seq], verify.LineTampered)
	}
}

// bodyLine is the JSONL line number of body index i (the header is line 1).
func bodyLine(i int) int { return i + 2 }

// Broken anchors, links, positions and trailers (§4 steps 2, 3, 6 and 8):
// each edit is reported as exactly one finding, at the line and seq that
// broke, with the package's detail text.
func TestChainBreaks(t *testing.T) { eachEncoding(t, testChainBreaks) }

func testChainBreaks(t *testing.T, e encoding) {
	n := len(vt.Build(vt.Options{}).Body) // anchor + seq 1..last
	trailer := bodyLine(n)
	last := int64(n - 1)
	malformedHex := func(member string, got int) string {
		return member + ": chain: invalid envelope: want 64 lowercase hex characters, got " + strconv.Itoa(got)
	}
	cases := []struct {
		name string
		opts vt.Options
		edit func(*vt.File)
		want failure
	}{
		{"second-anchor", vt.Options{}, func(f *vt.File) {
			f.Body = append([]vt.Line{f.Body[0], f.Body[1]}, append([]vt.Line{f.Body[0]}, f.Body[2:]...)...)
		}, failure{bodyLine(2), 0, verify.ReasonLineOrder, "an anchor that is not the first chain line"}},
		{"genesis-anchor-with-seq", vt.Options{}, func(f *vt.File) { f.Body[0]["seq"] = 1 },
			failure{bodyLine(0), noSeq, verify.ReasonAnchorInvalid,
				"a genesis anchor has seq 0, null prev_hash and envelope_sha256, and the zero entry_hash"}},
		{"genesis-anchor-with-forged-hash", vt.Options{}, func(f *vt.File) { f.Body[0]["entry_hash"] = strings.Repeat("ab", 32) },
			failure{bodyLine(0), noSeq, verify.ReasonAnchorInvalid,
				"a genesis anchor has seq 0, null prev_hash and envelope_sha256, and the zero entry_hash"}},
		{"stub-anchor-without-prev-hash", vt.Options{AfterSeq: 2}, func(f *vt.File) { f.Body[0]["prev_hash"] = nil },
			failure{bodyLine(0), 2, verify.ReasonAnchorInvalid, "a non-genesis anchor has seq >= 1, prev_hash and envelope_sha256"}},
		{"stub-anchor-forged-entry-hash", vt.Options{AfterSeq: 2}, func(f *vt.File) { f.Body[0]["entry_hash"] = strings.Repeat("ab", 32) },
			failure{bodyLine(0), 2, verify.ReasonEntryHashMismatch, "entry_hash is not SHA-256(prev_hash ‖ envelope_sha256)"}},
		{"stub-anchor-short-prev-hash", vt.Options{AfterSeq: 2}, func(f *vt.File) { f.Body[0]["prev_hash"] = "abc" },
			failure{bodyLine(0), 2, verify.ReasonMalformedLine, malformedHex("prev_hash", 3)}},
		{"stub-anchor-uppercase-envelope-sha", vt.Options{AfterSeq: 2}, func(f *vt.File) {
			f.Body[0]["envelope_sha256"] = strings.ToUpper(f.Body[0]["envelope_sha256"].(string)) //nolint:forcetypeassert // builder's anchor
		}, failure{bodyLine(0), 2, verify.ReasonMalformedLine, "envelope_sha256: chain: invalid envelope: hex must be lowercase [0-9a-f]"}},
		{"stub-anchor-long-entry-hash", vt.Options{AfterSeq: 2}, func(f *vt.File) { f.Body[0]["entry_hash"] = strings.Repeat("ab", 33) },
			failure{bodyLine(0), 2, verify.ReasonMalformedLine, malformedHex("entry_hash", 66)}},
		{"anchor-not-where-the-filter-puts-it", vt.Options{AfterSeq: 2}, func(f *vt.File) {
			f.Header["filter"].(vt.Line)["after_seq"] = nil //nolint:forcetypeassert // builder's header
		}, failure{bodyLine(0), 2, verify.ReasonAnchorMismatch, "the header filter puts the anchor at seq 0 (R20: first_seq - 1)"}},
		{"replayed-earlier-line", vt.Options{}, func(f *vt.File) {
			i := f.Index(3)
			f.Body = append(f.Body[:i+1], append([]vt.Line{f.Body[f.Index(2)]}, f.Body[i+1:]...)...)
		}, failure{bodyLine(4), 2, verify.ReasonSeqOutOfOrder, "seq 2 follows seq 3"}},
		{"sealed-entry-forged-entry-hash", vt.Options{}, func(f *vt.File) { f.Body[f.Index(1)]["entry_hash"] = strings.Repeat("ab", 32) },
			failure{bodyLine(1), 1, verify.ReasonEntryHashMismatch, "entry_hash is not SHA-256(prev_hash ‖ envelope_sha256)"}},
		{"sealed-line-after-unsealed", vt.Options{Unsealed: []vectors.ChainCase{vt.Chain()[0]}}, func(f *vt.File) {
			f.Body[n-1], f.Body[n] = f.Body[n], f.Body[n-1]
		}, failure{bodyLine(n), last, verify.ReasonLineOrder, "a sealed range line after an unsealed line"}},
		{"trailer-unsealed-count", vt.Options{}, func(f *vt.File) { f.Trailer["unsealed_count"] = 1 },
			failure{trailer, noSeq, verify.ReasonCountMismatch, "unsealed_count disagrees with the unsealed lines"}},
		{"trailer-anchor-seq-moved", vt.Options{}, func(f *vt.File) { f.Trailer["anchor_seq"] = 1 },
			failure{trailer, noSeq, verify.ReasonAnchorMismatch, "trailer anchor_seq is not the anchor line's seq"}},
		{"trailer-anchor-seq-null", vt.Options{}, func(f *vt.File) { f.Trailer["anchor_seq"] = nil },
			failure{trailer, noSeq, verify.ReasonAnchorMismatch, "trailer anchor_seq is not the anchor line's seq"}},
		{"trailer-count", vt.Options{}, func(f *vt.File) { f.Trailer["count"] = n },
			failure{trailer, noSeq, verify.ReasonCountMismatch, "count or stub_count disagrees with the lines"}},
		{"trailer-without-head", vt.Options{}, func(f *vt.File) { f.Trailer["last_seq"] = nil },
			failure{trailer, noSeq, verify.ReasonHeadMismatch, "a non-empty range has last_seq and head_hash"}},
		{"trailer-head-hash-null", vt.Options{}, func(f *vt.File) { f.Trailer["head_hash"] = nil },
			failure{trailer, noSeq, verify.ReasonHeadMismatch, "a non-empty range has last_seq and head_hash"}},
		{"trailer-head-before-last-line", vt.Options{}, func(f *vt.File) { f.Trailer["last_seq"] = last - 1 },
			failure{trailer, noSeq, verify.ReasonHeadMismatch, "a range line beyond last_seq"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := vt.Build(c.opts)
			c.edit(f)
			wantFailure(t, e, run(t, e.file(f)), c.want)
		})
	}

	// A window with a lower time bound cannot fix the anchor offline, so a
	// stub anchor anywhere verifies (expectedAnchor's "cannot check").
	t.Run("time-window-anchor-not-checked", func(t *testing.T) {
		f := vt.Build(vt.Options{AfterSeq: 2})
		flt := f.Header["filter"].(vt.Line) //nolint:forcetypeassert // builder's header
		flt["after_seq"], flt["from"] = nil, "2026-11-02T00:00:00.000000Z"
		r := run(t, e.file(f))
		wantOutcome(t, r, verify.OutcomePass)
		if r.AnchorSeq == nil || *r.AnchorSeq != 2 {
			t.Fatalf("anchor seq %v, want 2", r.AnchorSeq)
		}
	})
}

// Sealed and unsealed entries whose state members contradict each other, and
// diff states that break §4 step 5.
func TestEntryStateBreaks(t *testing.T) { eachEncoding(t, testEntryStateBreaks) }

func testEntryStateBreaks(t *testing.T, e encoding) {
	erasedSeq := seqOf(t, "erased-entry")
	diffSeq := seqOf(t, "record-update-all-fields")
	n := len(vt.Build(vt.Options{}).Body)
	cases := []struct {
		name string
		opts vt.Options
		edit func(*vt.File)
		want failure
	}{
		{"sealed-entry-outside-verified-range", vt.Options{}, func(f *vt.File) { f.Body[f.Index(1)]["in_verified_range"] = false },
			failure{bodyLine(1), 1, verify.ReasonLineStateMismatch,
				"a sealed entry has in_verified_range true and seq, prev_hash, entry_hash and sealed_at set"}},
		{"sealed-entry-without-sealed-at", vt.Options{}, func(f *vt.File) { f.Body[f.Index(1)]["sealed_at"] = nil },
			failure{bodyLine(1), 1, verify.ReasonLineStateMismatch,
				"a sealed entry has in_verified_range true and seq, prev_hash, entry_hash and sealed_at set"}},
		{"sealed-entry-without-seq", vt.Options{}, func(f *vt.File) { f.Body[f.Index(1)]["seq"] = nil },
			failure{bodyLine(1), noSeq, verify.ReasonLineStateMismatch,
				"a sealed entry has in_verified_range true and seq, prev_hash, entry_hash and sealed_at set"}},
		{"sealed-at-not-canonical", vt.Options{}, func(f *vt.File) { f.Body[f.Index(1)]["sealed_at"] = "2026-11-02T12:00:00Z" },
			failure{bodyLine(1), 1, verify.ReasonMalformedLine, "sealed_at is not a canonical timestamp"}},
		{"unsealed-entry-with-seq", vt.Options{Unsealed: []vectors.ChainCase{vt.Chain()[0]}}, func(f *vt.File) { f.Body[n]["seq"] = 1 },
			failure{bodyLine(n), noSeq, verify.ReasonLineStateMismatch,
				"an unsealed entry has in_verified_range false and null seq, prev_hash, entry_hash and sealed_at"}},
		{"unsealed-entry-in-verified-range", vt.Options{Unsealed: []vectors.ChainCase{vt.Chain()[0]}}, func(f *vt.File) {
			f.Body[n]["in_verified_range"] = true
		}, failure{bodyLine(n), noSeq, verify.ReasonLineStateMismatch,
			"an unsealed entry has in_verified_range false and null seq, prev_hash, entry_hash and sealed_at"}},
		{"erased-without-erasure-ref", vt.Options{}, func(f *vt.File) { f.Body[f.Index(erasedSeq)]["erasure_ref"] = nil },
			failure{bodyLine(int(erasedSeq)), erasedSeq, verify.ReasonDiffStateMismatch, "erased_at and erasure_ref are set together"}},
		{"erasure-ref-without-erased-at", vt.Options{}, func(f *vt.File) { f.Body[f.Index(erasedSeq)]["erased_at"] = nil },
			failure{bodyLine(int(erasedSeq)), erasedSeq, verify.ReasonDiffStateMismatch, "erased_at and erasure_ref are set together"}},
		{"erasure-ref-uppercase", vt.Options{}, func(f *vt.File) {
			l := f.Body[f.Index(erasedSeq)]
			l["erasure_ref"] = strings.ToUpper(*l["erasure_ref"].(*string)) //nolint:forcetypeassert // builder's entry
		}, failure{bodyLine(int(erasedSeq)), erasedSeq, verify.ReasonDiffStateMismatch, "erasure_ref is not a lowercase UUID"}},
		{"erasure-ref-not-a-uuid", vt.Options{}, func(f *vt.File) { f.Body[f.Index(erasedSeq)]["erasure_ref"] = "erasure-1" },
			failure{bodyLine(int(erasedSeq)), erasedSeq, verify.ReasonDiffStateMismatch, "erasure_ref is not a lowercase UUID"}},
		{"diff-salt-malformed", vt.Options{}, func(f *vt.File) { f.Body[f.Index(diffSeq)]["diff_salt"] = "0f0f" },
			failure{bodyLine(int(diffSeq)), diffSeq, verify.ReasonMalformedLine,
				"diff_salt: chain: invalid envelope: want 32 lowercase hex characters, got 4"}},
		{"diff-salt-uppercase", vt.Options{}, func(f *vt.File) {
			l := f.Body[f.Index(diffSeq)]
			l["diff_salt"] = strings.ToUpper(*l["diff_salt"].(*string)) //nolint:forcetypeassert // builder's entry
		}, failure{bodyLine(int(diffSeq)), diffSeq, verify.ReasonMalformedLine, "diff_salt: chain: invalid envelope: hex must be lowercase [0-9a-f]"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := vt.Build(c.opts)
			c.edit(f)
			wantFailure(t, e, run(t, e.file(f)), c.want)
		})
	}
}

// R23/R24: an audit/erase entry vouches for an erased id only when its diff
// is exactly one add of /erased_ids listing 1 to 10,000 lowercase ids. An
// erase entry with any other diff shape, its diff_sha256 consistent with that
// diff, lists nothing, so the erased entry is erasure_unlisted.
func TestEraseEntryShape(t *testing.T) { eachEncoding(t, testEraseEntryShape) }

func testEraseEntryShape(t *testing.T, e encoding) {
	erasedSeq := seqOf(t, "erased-entry")
	erasedID := chainCase(t, "erased-entry").Envelope.ID.String()
	// pendingWith exports seq 1..erasedSeq with the audit/erase entry
	// unsealed, its diff replaced by d and diff_sha256 recomputed, so only
	// the shape of d decides the outcome.
	pendingWith := func(t *testing.T, d string) *verify.Report {
		t.Helper()
		c := chainCase(t, "audit-erase")
		salt, err := chain.ParseSalt16(*c.DiffSalt)
		if err != nil {
			t.Fatal(err)
		}
		h := chain.DiffSHA256(salt, []byte(d))
		c.Envelope.DiffSHA256 = &h
		c.Diff = &d
		return run(t, e.file(vt.Build(vt.Options{HeadSeq: erasedSeq, Unsealed: []vectors.ChainCase{c}})))
	}
	ids := func(vals ...string) string {
		return `[{"op":"add","path":"/erased_ids","value":["` + strings.Join(vals, `","`) + `"]}]`
	}
	t.Run("exact-shape-pending", func(t *testing.T) {
		r := pendingWith(t, ids(erasedID))
		wantOutcome(t, r, verify.OutcomePending)
		if r.Lines[erasedSeq] != verify.LinePending {
			t.Fatalf("line outcomes %v, want pending at seq %d", r.Lines, erasedSeq)
		}
	})
	many := make([]string, 10001)
	for i := range many {
		many[i] = erasedID
	}
	for name, d := range map[string]string{
		"not-json":            `{`,
		"unknown-member":      `[{"op":"add","path":"/erased_ids","value":["` + erasedID + `"],"from":"/x"}]`,
		"two-ops":             `[` + strings.Trim(ids(erasedID), "[]") + `,` + strings.Trim(ids(erasedID), "[]") + `]`,
		"remove-op":           `[{"op":"remove","path":"/erased_ids","value":["` + erasedID + `"]}]`,
		"other-path":          `[{"op":"add","path":"/ids","value":["` + erasedID + `"]}]`,
		"empty-list":          `[{"op":"add","path":"/erased_ids","value":[]}]`,
		"over-the-r23-cap":    ids(many...),
		"uppercase-id-listed": ids(strings.ToUpper(erasedID)),
		"trailing-data":       ids(erasedID) + ` {}`,
		"trailing-array":      ids(erasedID) + ids(erasedID),
		"duplicate-member":    `[{"op":"remove","op":"add","path":"/erased_ids","value":["` + erasedID + `"]}]`,
		"duplicate-value":     `[{"op":"add","path":"/erased_ids","value":[],"value":["` + erasedID + `"]}]`,
		"missing-member":      `[{"op":"add","value":["` + erasedID + `"]}]`,
	} {
		t.Run(name, func(t *testing.T) {
			wantFailure(t, e, pendingWith(t, d), failure{bodyLine(int(erasedSeq)), erasedSeq, verify.ReasonErasureUnlisted,
				"erased, but no chain-valid audit/erase entry for its erasure_ref lists its id (R24)"})
		})
	}
}

// A file with a header and a trailer but no chain line is an empty range
// only when the trailer says so exactly.
func TestEmptyRange(t *testing.T) { eachEncoding(t, testEmptyRange) }

func testEmptyRange(t *testing.T, e encoding) {
	empty := func() *vt.File {
		f := vt.Build(vt.Options{})
		f.Body = nil
		f.Trailer["anchor_seq"], f.Trailer["last_seq"], f.Trailer["head_hash"], f.Trailer["verifiable"] = nil, nil, nil, false
		f.Recount()
		return f
	}
	r := run(t, e.file(empty()))
	wantOutcome(t, r, verify.OutcomeEmpty)
	if r.AnchorSeq != nil || r.LastSeq != nil || r.Count != 0 {
		t.Fatalf("empty report anchor %v last %v count %d", r.AnchorSeq, r.LastSeq, r.Count)
	}
	cases := []struct {
		name string
		edit func(*vt.File)
		want failure
	}{
		{"anchor-seq-without-anchor", func(f *vt.File) { f.Trailer["anchor_seq"] = 0 },
			failure{2, noSeq, verify.ReasonAnchorMissing, "trailer anchor_seq is set but the file has no anchor (R20)"}},
		{"last-seq", func(f *vt.File) { f.Trailer["last_seq"] = 3 },
			failure{2, noSeq, verify.ReasonCountMismatch, "an empty range has null last_seq and head_hash and zero counts"}},
		{"head-hash", func(f *vt.File) { f.Trailer["head_hash"] = strings.Repeat("ab", 32) },
			failure{2, noSeq, verify.ReasonCountMismatch, "an empty range has null last_seq and head_hash and zero counts"}},
		{"count", func(f *vt.File) { f.Trailer["count"] = 1 },
			failure{2, noSeq, verify.ReasonCountMismatch, "an empty range has null last_seq and head_hash and zero counts"}},
		{"stub-count", func(f *vt.File) { f.Trailer["stub_count"] = 1 },
			failure{2, noSeq, verify.ReasonCountMismatch, "an empty range has null last_seq and head_hash and zero counts"}},
		{"verifiable", func(f *vt.File) { f.Trailer["verifiable"] = true },
			failure{2, noSeq, verify.ReasonVerifiableFlagMismatch, "an empty range is not verifiable"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := empty()
			c.edit(f)
			wantFailure(t, e, run(t, e.file(f)), c.want)
		})
	}
}

// Header lines that are not a v1 header: line 1 is header_invalid, with the
// strict parser's own detail.
func TestHeaderInvalid(t *testing.T) { eachEncoding(t, testHeaderInvalid) }

func testHeaderInvalid(t *testing.T, e encoding) {
	cases := []struct {
		name   string
		edit   func(vt.Line)
		detail string
	}{
		{"tenant-uppercase", func(h vt.Line) { h["tenant_id"] = strings.ToUpper(h["tenant_id"].(string)) }, //nolint:forcetypeassert // builder's header
			"tenant_id is not a lowercase UUID"},
		{"tenant-not-a-uuid", func(h vt.Line) { h["tenant_id"] = "tenant-1" }, "tenant_id is not a lowercase UUID"},
		{"member-missing", func(h vt.Line) { delete(h, "exported_at") }, `missing member "exported_at"`},
		{"member-null", func(h vt.Line) { h["format"] = nil }, `member "format" must not be null`},
		{"filter-not-an-object", func(h vt.Line) { h["filter"] = []int{} }, "filter: not a JSON object"},
		{"filter-member-missing", func(h vt.Line) { delete(h["filter"].(vt.Line), "after_seq") }, //nolint:forcetypeassert // builder's header
			`filter: missing member "after_seq"`},
		{"filter-member-unknown", func(h vt.Line) { h["filter"].(vt.Line)["until_seq"] = nil }, //nolint:forcetypeassert // builder's header
			`filter: unknown member "until_seq"`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := vt.Build(vt.Options{})
			c.edit(f.Header)
			wantFailure(t, e, run(t, e.file(f)), failure{1, noSeq, verify.ReasonHeaderInvalid, c.detail})
		})
	}
}

// insertLine is the clean JSONL file with raw inserted as line at (1-based).
func insertLine(at int, raw string) []byte {
	ls := strings.SplitAfter(string(vt.Build(vt.Options{}).Bytes()), "\n")
	ls = append(ls[:at-1], append([]string{raw + "\n"}, ls[at-1:]...)...)
	return []byte(strings.Join(ls, ""))
}

// §4 step 10, JSONL: a line that is not one strict object of its kind is
// malformed_line (header_invalid as line 1), with the parser's detail.
func TestStrictLineParser(t *testing.T) {
	cases := []struct {
		name, raw, detail string
	}{
		{"not-json", `x`, ` line: invalid character 'x' looking for beginning of value`},
		{"array", `[1]`, ` line: not a JSON object`},
		{"non-string-key", `{1:2}`, ` line: invalid character '1'`},
		{"member-value-invalid", `{"kind":}`, ` line: member "kind": invalid character '}' looking for beginning of value`},
		{"unterminated-object", `{"kind":"stub"`, ` line: EOF`},
		{"trailing-data", `{"kind":"stub"} {}`, ` line: trailing data after the object`},
		{"kind-missing", `{"seq":1}`, ` line: missing member "kind"`},
		{"kind-not-a-string", `{"kind":1}`, ` line: member "kind" is not a string`},
		{"member-missing", `{"kind":"stub","seq":1}`, `stub line: missing member "prev_hash"`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			wantFailure(t, encodings()[0], run(t, insertLine(2, c.raw)),
				failure{2, noSeq, verify.ReasonMalformedLine, c.detail})
			// The same line in place of the header is header_invalid, with the
			// parser's detail unprefixed.
			wantFailure(t, encodings()[0], run(t, []byte(c.raw+"\n")),
				failure{1, noSeq, verify.ReasonHeaderInvalid, strings.TrimPrefix(strings.TrimPrefix(c.detail, "stub"), " line: ")})
		})
	}
	t.Run("filter-is-not-a-line-kind", func(t *testing.T) {
		wantFailure(t, encodings()[0], run(t, insertLine(2, `{"kind":"filter"}`)),
			failure{2, noSeq, verify.ReasonUnknownKind, `kind "filter"`})
	})
}

// A line of the right shape whose member has the wrong JSON type fails the
// typed decode: malformed_line naming the kind (header_invalid for the
// header), with encoding/json's detail.
func TestTypedDecode(t *testing.T) {
	jsonl := encodings()[0]
	n := len(vt.Build(vt.Options{}).Body)
	cases := []struct {
		name   string
		edit   func(*vt.File)
		line   int
		reason string
		prefix string
		field  string
	}{
		{"header", func(f *vt.File) { f.Header["stubs"] = "yes" }, 1, verify.ReasonHeaderInvalid, "", "headerLine.stubs"},
		{"anchor", func(f *vt.File) { f.Body[0]["seq"] = "0" }, bodyLine(0), verify.ReasonMalformedLine, "anchor line: ", "anchorLine.seq"},
		{"stub", func(f *vt.File) { f.Body[1] = vt.Stub(vt.Chain()[0]); f.Body[1]["seq"] = "1" },
			bodyLine(1), verify.ReasonMalformedLine, "stub line: ", "stubLine.seq"},
		{"trailer", func(f *vt.File) { f.Trailer["count"] = "7" }, bodyLine(n), verify.ReasonMalformedLine, "trailer line: ", "trailerLine.count"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := vt.Build(vt.Options{})
			c.edit(f)
			r := run(t, jsonl.file(f))
			wantTampered(t, r, c.name)
			fl := r.Failure
			if fl.Line != c.line || fl.Reason != c.reason || fl.Seq != nil ||
				!strings.HasPrefix(fl.Detail, c.prefix+"json: cannot unmarshal string into Go struct field ") ||
				!strings.Contains(fl.Detail, c.field) {
				t.Fatalf("got %s, want line %d: %s: %sjson: cannot unmarshal ... %s", describe(r), c.line, c.reason, c.prefix, c.field)
			}
		})
	}
}

// CSV rows whose cells cannot be rebuilt into a line (§5).
func TestCSVRowRebuild(t *testing.T) {
	csv := encodings()[1]
	lines := strings.Split(strings.TrimSuffix(string(vt.Build(vt.Options{}).Bytes()), "\n"), "\n")
	cols := len(verify.CSVColumns)
	t.Run("envelope-integer-column", func(t *testing.T) {
		ls := append([]string(nil), lines...)
		edited := strings.Replace(ls[2], `"causation_depth":0`, `"causation_depth":"x"`, 1)
		if edited == ls[2] {
			t.Fatal("seq 1 entry has no causation_depth 0")
		}
		ls[2] = edited
		wantFailure(t, csv, run(t, vt.CSV(ls)),
			failure{bodyLine(1), noSeq, verify.ReasonMalformedLine, `column "causation_depth" is not an integer`})
	})
	t.Run("envelope-boolean-column", func(t *testing.T) {
		ls := append([]string(nil), lines...)
		edited := strings.Replace(ls[2], `"backfilled":false`, `"backfilled":"no"`, 1)
		if edited == ls[2] {
			t.Fatal("seq 1 entry has no backfilled false")
		}
		ls[2] = edited
		wantFailure(t, csv, run(t, vt.CSV(ls)),
			failure{bodyLine(1), noSeq, verify.ReasonMalformedLine, `column "backfilled" is not true or false`})
	})
	for _, kind := range []string{"header", "trailer"} {
		t.Run(kind+"-row-without-meta", func(t *testing.T) {
			rs := rows(t, vt.Build(vt.Options{}).CSV())
			empty := kind + strings.Repeat(",", cols-1) + "\r\n"
			line, reason := 1, verify.ReasonHeaderInvalid
			if kind == "header" {
				rs[1] = empty
			} else {
				rs[len(rs)-1] = empty
				line, reason = len(rs)-1, verify.ReasonMalformedLine
			}
			wantFailure(t, csv, run(t, []byte(strings.Join(rs, ""))),
				failure{line, noSeq, reason, "a " + kind + " row has its JSONL line in meta"})
		})
	}
}

// A reader that fails is a read error, not a verdict: Verify returns the
// reader's error and no report, wherever in the file the failure comes.
func TestReadErrors(t *testing.T) {
	boom := errors.New("disk gone")
	clean := vt.Build(vt.Options{})
	jsonlHead := clean.Bytes()[:bytes.IndexByte(clean.Bytes(), '\n')+1]
	csvRows := rows(t, clean.CSV())
	cases := []struct {
		name   string
		prefix string
	}{
		{"before-the-first-byte", ""},
		{"jsonl-after-the-header", string(jsonlHead)},
		{"csv-after-the-column-row", csvRows[0]},
		{"csv-inside-a-cell", "kind,se"},
		{"csv-inside-a-quoted-cell", `kind,"se`},
		{"csv-after-a-cr", "kind,seq\r"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r, err := verify.Verify(io.MultiReader(strings.NewReader(c.prefix), iotest.ErrReader(boom)))
			if !errors.Is(err, boom) || r != nil {
				t.Fatalf("Verify = %+v, %v; want nil, %v", r, err, boom)
			}
		})
	}
}

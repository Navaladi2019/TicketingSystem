package chain_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/audit/chain"
	"github.com/Navaladi2019/TicketingSystem/internal/audit/chain/vectors"
)

// TestGoldenVectorsExports runs a reference implementation of
// audit-export-v1.md §4 over every export vector and requires the outcome
// the vector states. It proves the vectors and the spec agree; the shipped
// verifier is P5-03's.
func TestGoldenVectorsExports(t *testing.T) {
	f := loadVectors(t)
	want := map[string]bool{
		"diff-dropped-not-erased": false, "diff-redacted-replaced": false,
		"header-tenant-mismatch": false, "verifiable-flag-false-on-complete-file": false,
	}
	for _, x := range f.Exports {
		t.Run(x.Name, func(t *testing.T) {
			got := referenceVerify(x.JSONL)
			if got.file != x.Expect {
				t.Fatalf("file outcome %q (%s), want %q", got.file, got.reason, x.Expect)
			}
			wantReason := ""
			if x.Reason != nil {
				wantReason = *x.Reason
			}
			if got.reason != wantReason {
				t.Errorf("reason %q, want %q", got.reason, wantReason)
			}
			if fmt.Sprint(got.lines) != fmt.Sprint(x.LineOutcomes) {
				t.Errorf("line outcomes %v, want %v", got.lines, x.LineOutcomes)
			}
		})
		if _, ok := want[x.Name]; ok {
			want[x.Name] = true
		}
	}
	for name, seen := range want {
		if !seen {
			t.Errorf("no export vector %s (TA PR #365 finding 6)", name)
		}
	}
}

type verdict struct {
	file, reason string
	lines        map[string]string // seq -> outcome, for outcomes other than verified
}

type xHeader struct {
	Kind       string          `json:"kind"`
	Format     string          `json:"format"`
	TenantID   string          `json:"tenant_id"`
	ExportID   string          `json:"export_id"`
	ExportedAt string          `json:"exported_at"`
	Stubs      bool            `json:"stubs"`
	Filter     json.RawMessage `json:"filter"`
}

type xLine struct {
	Kind            string          `json:"kind"`
	Genesis         *bool           `json:"genesis"`
	Sealed          *bool           `json:"sealed"`
	InVerifiedRange *bool           `json:"in_verified_range"`
	Seq             *int64          `json:"seq"`
	PrevHash        *string         `json:"prev_hash"`
	EntryHash       *string         `json:"entry_hash"`
	EnvelopeSHA256  *string         `json:"envelope_sha256"`
	SealedAt        *string         `json:"sealed_at"`
	Envelope        *chain.Envelope `json:"envelope"`
	Diff            *string         `json:"diff"`
	DiffSalt        *string         `json:"diff_salt"`
	DiffRedacted    *bool           `json:"diff_redacted"`
	ErasedAt        *string         `json:"erased_at"`
	ErasureRef      *string         `json:"erasure_ref"`
}

type xTrailer struct {
	Kind          string  `json:"kind"`
	AnchorSeq     *int64  `json:"anchor_seq"`
	LastSeq       *int64  `json:"last_seq"`
	HeadHash      *string `json:"head_hash"`
	Count         int     `json:"count"`
	StubCount     int     `json:"stub_count"`
	UnsealedCount int     `json:"unsealed_count"`
	Verifiable    bool    `json:"verifiable"`
}

// strictDecode is step 10: no duplicate or unknown member.
func strictDecode(line string, v any) error {
	dec := json.NewDecoder(bytes.NewReader([]byte(line)))
	if _, err := dec.Token(); err != nil {
		return err
	}
	seen := map[string]bool{}
	for dec.More() {
		k, err := dec.Token()
		if err != nil {
			return err
		}
		key, _ := k.(string)
		if seen[key] {
			return fmt.Errorf("duplicate member %q", key)
		}
		seen[key] = true
		var skip json.RawMessage
		if err := dec.Decode(&skip); err != nil {
			return err
		}
	}
	d := json.NewDecoder(bytes.NewReader([]byte(line)))
	d.DisallowUnknownFields()
	return d.Decode(v)
}

func referenceVerify(lines []string) verdict {
	out := verdict{lines: map[string]string{}}
	fail := func(reason string, seq *int64) verdict {
		out.file, out.reason = vectors.FileTampered, reason
		if seq != nil {
			out.lines[strconv.FormatInt(*seq, 10)] = vectors.LineTampered
		}
		return out
	}
	if len(lines) < 2 {
		return fail("structure", nil)
	}
	// Step 1: header and trailer.
	var h xHeader
	if err := strictDecode(lines[0], &h); err != nil || h.Kind != "header" || h.Format != "nava.audit.export/v1" {
		return fail("header", nil)
	}
	var tr xTrailer
	if err := strictDecode(lines[len(lines)-1], &tr); err != nil || tr.Kind != "trailer" {
		return fail("trailer", nil)
	}
	var body []xLine
	for _, l := range lines[1 : len(lines)-1] {
		var x xLine
		if err := strictDecode(l, &x); err != nil {
			return fail("parse", nil)
		}
		if !slices.Contains([]string{"anchor", "entry", "stub"}, x.Kind) {
			return fail("unknown_kind", nil)
		}
		body = append(body, x)
	}
	var rng, unsealed []xLine
	for _, x := range body {
		if x.Kind == "entry" && (x.Sealed == nil || !*x.Sealed) {
			unsealed = append(unsealed, x)
		} else {
			rng = append(rng, x)
		}
	}
	for _, x := range unsealed {
		if x.Envelope.TenantID.String() != h.TenantID { // step 4a
			return fail(vectors.ReasonHeaderTenantMismatch, nil)
		}
	}
	if tr.UnsealedCount != len(unsealed) {
		return fail("count", nil)
	}

	// Empty range (§3): nothing to verify.
	if tr.AnchorSeq == nil {
		if len(rng) != 0 || tr.LastSeq != nil || tr.HeadHash != nil || tr.Verifiable {
			return fail(vectors.ReasonVerifiableFlagMismatch, nil)
		}
		out.file = vectors.FileEmpty
		return out
	}

	// Step 2: the anchor comes first.
	a := rng[0]
	if a.Kind != "anchor" || a.Seq == nil || *a.Seq != *tr.AnchorSeq || a.EntryHash == nil {
		return fail("anchor", nil)
	}
	if *a.Genesis && (*a.Seq != 0 || *a.EntryHash != chain.Hex(chain.Genesis[:])) {
		return fail("anchor", nil)
	}
	contiguous := true
	prevSeq, prevHash := *a.Seq, *a.EntryHash
	count, stubs := 0, 0
	for _, x := range rng[1:] {
		seq := *x.Seq
		if x.Kind == "anchor" {
			return fail("anchor", &seq)
		}
		// Step 3: contiguity and links. A gap is allowed only in a file that
		// says it is not verifiable; step 8 then decides.
		linked := seq == prevSeq+1
		if !linked {
			contiguous = false
		}
		if linked && *x.PrevHash != prevHash {
			return fail(vectors.ExpectEntryHashMismatch, &seq)
		}
		esha, err := chain.ParseHash32(*x.EnvelopeSHA256)
		if err != nil {
			return fail("parse", &seq)
		}
		if x.Kind == "entry" {
			count++
			// Step 4a: the entry belongs to the header's tenant.
			if x.Envelope.TenantID.String() != h.TenantID {
				return fail(vectors.ReasonHeaderTenantMismatch, &seq)
			}
			// Step 4: recompute envelope_sha256.
			got, err := chain.EnvelopeSHA256(*x.Envelope)
			if err != nil || got != esha {
				return fail(vectors.ExpectEnvelopeMismatch, &seq)
			}
			// Step 5: diff state.
			if o, reason := diffState(x); reason != "" {
				return fail(reason, &seq)
			} else if o != vectors.LineVerified {
				out.lines[strconv.FormatInt(seq, 10)] = o
			}
		} else {
			stubs++
		}
		prev, err := chain.ParseHash32(*x.PrevHash)
		if err != nil {
			return fail("parse", &seq)
		}
		if eh := chain.EntryHash(prev, esha); chain.Hex(eh[:]) != *x.EntryHash {
			return fail(vectors.ExpectEntryHashMismatch, &seq)
		}
		prevSeq, prevHash = seq, *x.EntryHash
	}
	if count != tr.Count || stubs != tr.StubCount {
		return fail("count", nil)
	}
	// Step 6: the range ends at the head (checkable only without gaps).
	if tr.LastSeq == nil || tr.HeadHash == nil || prevSeq > *tr.LastSeq {
		return fail("head", nil)
	}
	if contiguous && (prevSeq != *tr.LastSeq || prevHash != *tr.HeadHash) {
		return fail("head", nil)
	}
	// Step 7: erased entries need a chain-valid erase entry listing them.
	for _, x := range rng {
		if x.Kind != "entry" || x.ErasedAt == nil {
			continue
		}
		if !erasureListed(rng, *x.ErasureRef, x.Envelope.ID.String()) {
			return fail(vectors.ReasonErasureUnlisted, x.Seq)
		}
	}
	// Step 8: verifiability is computed, then the flags must agree.
	if h.Stubs != contiguous || tr.Verifiable != contiguous {
		return fail(vectors.ReasonVerifiableFlagMismatch, nil)
	}
	if !contiguous {
		out.file = vectors.FileUnverifiable
		return out
	}
	out.file = vectors.FilePass
	return out
}

// diffState is §4 step 5. It returns the line outcome, or a tamper reason.
func diffState(x xLine) (outcome, reason string) {
	e := x.Envelope.DiffSHA256
	redacted := x.DiffRedacted != nil && *x.DiffRedacted
	switch {
	case e == nil:
		if x.Diff != nil || x.DiffSalt != nil || x.ErasedAt != nil || redacted {
			return "", vectors.ReasonDiffStateMismatch
		}
		return vectors.LineVerified, ""
	case x.ErasedAt != nil:
		if x.Diff != nil || x.DiffSalt != nil {
			return "", vectors.ReasonDiffStateMismatch
		}
		return vectors.LineVerified, "" // step 7 decides
	case redacted:
		if x.DiffSalt != nil {
			return "", vectors.ReasonDiffStateMismatch
		}
		return vectors.LineDiffUnverified, ""
	default:
		if x.Diff == nil || x.DiffSalt == nil {
			return "", vectors.ReasonDiffMissing
		}
		salt, err := chain.ParseSalt16(*x.DiffSalt)
		if err != nil || chain.DiffSHA256(salt, []byte(*x.Diff)) != *e {
			return "", vectors.ExpectDiffMismatch
		}
		return vectors.LineVerified, ""
	}
}

// erasureListed looks for a full audit/erase line for ref whose diff is
// exactly [{"op":"add","path":"/erased_ids","value":[ids]}] and lists id.
func erasureListed(rng []xLine, ref, id string) bool {
	for _, y := range rng {
		if y.Kind != "entry" || y.Envelope.Domain != "audit" || y.Envelope.Action != "erase" ||
			y.Envelope.SubjectID.String() != ref || y.Diff == nil {
			continue
		}
		var ops []struct {
			Op    string   `json:"op"`
			Path  string   `json:"path"`
			Value []string `json:"value"`
		}
		if err := json.Unmarshal([]byte(*y.Diff), &ops); err != nil || len(ops) != 1 ||
			ops[0].Op != "add" || ops[0].Path != "/erased_ids" || len(ops[0].Value) > 10000 {
			continue
		}
		if slices.Contains(ops[0].Value, id) {
			return true
		}
	}
	return false
}

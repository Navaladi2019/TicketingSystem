package verify_test

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Navaladi2019/TicketingSystem/internal/audit/chain"
	"github.com/Navaladi2019/TicketingSystem/internal/audit/chain/vectors"
	"github.com/Navaladi2019/TicketingSystem/internal/audit/verify"
	vt "github.com/Navaladi2019/TicketingSystem/internal/audit/verify/verifytest"
)

func run(t *testing.T, b []byte) *verify.Report {
	t.Helper()
	r, err := verify.Verify(bytes.NewReader(b))
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if (r.Outcome == verify.OutcomeTampered) != (r.Failure != nil) {
		t.Fatalf("outcome %s with failure %+v", r.Outcome, r.Failure)
	}
	return r
}

func reason(r *verify.Report) string {
	if r.Failure == nil {
		return ""
	}
	return r.Failure.Reason
}

func wantTampered(t *testing.T, r *verify.Report, why string) {
	t.Helper()
	if r.Outcome != verify.OutcomeTampered {
		t.Fatalf("outcome %s, want tampered (%s)", r.Outcome, why)
	}
}

func wantOutcome(t *testing.T, r *verify.Report, o verify.Outcome) {
	t.Helper()
	if r.Outcome != o {
		t.Fatalf("outcome %s (%s), want %s", r.Outcome, describe(r), o)
	}
}

func describe(r *verify.Report) string {
	if r.Failure == nil {
		return "no failure"
	}
	return fmt.Sprintf("line %d: %s: %s", r.Failure.Line, r.Failure.Reason, r.Failure.Detail)
}

// The reason codes shared with the golden vectors must be the same strings.
func TestReasonCodesMatchVectors(t *testing.T) {
	pairs := [][2]string{
		{verify.ReasonEnvelopeMismatch, vectors.ExpectEnvelopeMismatch},
		{verify.ReasonDiffMismatch, vectors.ExpectDiffMismatch},
		{verify.ReasonEntryHashMismatch, vectors.ExpectEntryHashMismatch},
		{verify.ReasonHeaderTenantMismatch, vectors.ReasonHeaderTenantMismatch},
		{verify.ReasonDiffMissing, vectors.ReasonDiffMissing},
		{verify.ReasonDiffStateMismatch, vectors.ReasonDiffStateMismatch},
		{verify.ReasonVerifiableFlagMismatch, vectors.ReasonVerifiableFlagMismatch},
		{verify.ReasonErasureUnlisted, vectors.ReasonErasureUnlisted},
		{string(verify.OutcomePass), vectors.FilePass},
		{string(verify.OutcomeTampered), vectors.FileTampered},
		{string(verify.OutcomeEmpty), vectors.FileEmpty},
		{string(verify.OutcomeUnverifiable), vectors.FileUnverifiable},
		{string(verify.LineVerified), vectors.LineVerified},
		{string(verify.LineDiffUnverified), vectors.LineDiffUnverified},
		{string(verify.LinePending), vectors.LinePending},
		{string(verify.LineTampered), vectors.LineTampered},
	}
	for _, p := range pairs {
		if p[0] != p[1] {
			t.Errorf("%q != vectors %q", p[0], p[1])
		}
	}
}

// Every whole-file golden vector produces the outcome, reason and per-line
// outcomes it states (audit-export-v1.md §6).
func TestGoldenExports(t *testing.T) {
	ex := vt.Vectors().Exports
	if len(ex) == 0 {
		t.Fatal("no export vectors")
	}
	for _, x := range ex {
		t.Run(x.Name, func(t *testing.T) {
			r := run(t, vt.JSONL(x.JSONL))
			if string(r.Outcome) != x.Expect {
				t.Fatalf("outcome %s (%s), want %s", r.Outcome, describe(r), x.Expect)
			}
			want := ""
			if x.Reason != nil {
				want = *x.Reason
			}
			if reason(r) != want {
				t.Errorf("reason %q, want %q", reason(r), want)
			}
			got := map[string]string{}
			for seq, o := range r.Lines {
				got[strconv.FormatInt(seq, 10)] = string(o)
			}
			if fmt.Sprint(got) != fmt.Sprint(x.LineOutcomes) {
				t.Errorf("line outcomes %v, want %v", got, x.LineOutcomes)
			}
		})
	}
}

// Each tampered golden vector, put in place of its line in an otherwise
// clean file, fails with the vector's reason. This covers the P5-02 /
// ADR-004-E1 case of a cleared diff_truncated flag.
func TestGoldenTamperedLines(t *testing.T) {
	sawTruncated := false
	for _, tv := range vt.Vectors().Tampered {
		t.Run(tv.Name, func(t *testing.T) {
			f := vt.Build(vt.Options{})
			f.Body[f.Index(tv.Seq)] = vt.Tampered(tv)
			r := run(t, f.Bytes())
			wantTampered(t, r, tv.Name)
			if reason(r) != tv.Expect {
				t.Errorf("reason %q (%s), want %q", reason(r), describe(r), tv.Expect)
			}
			if r.Failure.Seq == nil || *r.Failure.Seq != tv.Seq {
				t.Errorf("failure at seq %v, want %d", r.Failure.Seq, tv.Seq)
			}
		})
		if tv.Name == "diff-truncated-flag-cleared" {
			sawTruncated = true
		}
	}
	if !sawTruncated {
		t.Fatal("no diff-truncated-flag-cleared vector")
	}
}

// The truncated-but-honest entry verifies.
func TestTruncatedHonestVerifies(t *testing.T) {
	var seq int64
	for _, c := range vt.Chain() {
		if c.Name == "truncated-honest" {
			seq = c.Seq
			if !c.Envelope.DiffTruncated {
				t.Fatal("truncated-honest vector is not truncated")
			}
		}
	}
	r := run(t, vt.Build(vt.Options{Match: func(c vectors.ChainCase) bool { return c.Seq == seq }}).Bytes())
	wantOutcome(t, r, verify.OutcomePass)
	if o, ok := r.Lines[seq]; ok {
		t.Fatalf("truncated-honest line outcome %s, want verified", o)
	}
}

type filterCase struct {
	name string
	opts vt.Options
}

func filters() []filterCase {
	entityDef, record := "entity_definition", "record"
	return []filterCase{
		{"unfiltered", vt.Options{}},
		{"subject-type-entity-definition", vt.Options{SubjectType: &entityDef,
			Match: func(c vectors.ChainCase) bool { return c.Envelope.SubjectType == entityDef }}},
		{"subject-type-record", vt.Options{SubjectType: &record,
			Match: func(c vectors.ChainCase) bool { return c.Envelope.SubjectType == record }}},
		{"after-seq-2", vt.Options{AfterSeq: 2}},
		{"after-seq-2-record", vt.Options{AfterSeq: 2, SubjectType: &record,
			Match: func(c vectors.ChainCase) bool { return c.Envelope.SubjectType == record }}},
	}
}

// CI-A6 (file half): clean files verify, and a byte edit, a deleted line, a
// consistent single-row rewrite and a deleted first entry each fail, under
// every filter. A file without the anchor is rejected.
func TestCIA6(t *testing.T) {
	for _, fc := range filters() {
		t.Run(fc.name, func(t *testing.T) {
			wantOutcome(t, run(t, vt.Build(fc.opts).Bytes()), verify.OutcomePass)

			n := len(vt.Build(fc.opts).Body)
			for i := 1; i < n; i++ {
				f := vt.Build(fc.opts)
				l := f.Body[i]
				t.Run(fmt.Sprintf("byte-edit-envelope-sha-%v", l["seq"]), func(t *testing.T) {
					f := vt.Build(fc.opts)
					l := f.Body[i]
					s := l["envelope_sha256"].(string)
					l["envelope_sha256"] = s[:63] + flipHex(s[63])
					wantTampered(t, run(t, f.Bytes()), "envelope_sha256 edited")
				})
				if env, ok := l["envelope"].(chain.Envelope); ok {
					t.Run(fmt.Sprintf("byte-edit-envelope-%v", l["seq"]), func(t *testing.T) {
						f := vt.Build(fc.opts)
						env.OccurredAt = env.OccurredAt.Add(time.Microsecond)
						f.Body[i]["envelope"] = env
						r := run(t, f.Bytes())
						wantTampered(t, r, "envelope edited")
						if reason(r) != verify.ReasonEnvelopeMismatch {
							t.Errorf("reason %q, want %q", reason(r), verify.ReasonEnvelopeMismatch)
						}
					})
					t.Run(fmt.Sprintf("single-row-rewrite-%v", l["seq"]), func(t *testing.T) {
						f := vt.Build(fc.opts)
						env.OccurredAt = env.OccurredAt.Add(time.Second)
						f.Body[i]["envelope"] = env
						f.Rehash(i) // the row's own hashes are consistent
						r := run(t, f.Bytes())
						wantTampered(t, r, "rewritten row")
						if r := reason(r); r != verify.ReasonEntryHashMismatch && r != verify.ReasonHeadMismatch {
							t.Errorf("reason %q, want the next link or the head to break", r)
						}
					})
				} else {
					t.Run(fmt.Sprintf("single-stub-rewrite-%v", l["seq"]), func(t *testing.T) {
						f := vt.Build(fc.opts)
						s := f.Body[i]["envelope_sha256"].(string)
						f.Body[i]["envelope_sha256"] = flipHex(s[0]) + s[1:]
						f.Rehash(i)
						wantTampered(t, run(t, f.Bytes()), "rewritten stub")
					})
				}
				t.Run(fmt.Sprintf("deleted-line-%v", l["seq"]), func(t *testing.T) {
					f := vt.Build(fc.opts)
					f.Body = append(f.Body[:i], f.Body[i+1:]...)
					wantTampered(t, run(t, f.Bytes()), "line deleted")
				})
				t.Run(fmt.Sprintf("deleted-line-recounted-%v", l["seq"]), func(t *testing.T) {
					f := vt.Build(fc.opts)
					f.Body = append(f.Body[:i], f.Body[i+1:]...)
					f.Recount()
					wantTampered(t, run(t, f.Bytes()), "line deleted, trailer recounted")
				})
			}

			t.Run("deleted-first-entry", func(t *testing.T) {
				f := vt.Build(fc.opts)
				f.Body = append(f.Body[:1], f.Body[2:]...)
				f.Recount()
				r := run(t, f.Bytes())
				wantTampered(t, r, "first range entry deleted")
				if reason(r) != verify.ReasonSeqMissing {
					t.Errorf("reason %q, want %q", reason(r), verify.ReasonSeqMissing)
				}
			})
			t.Run("deleted-first-entry-reanchored", func(t *testing.T) {
				// The forger drops the first entry and turns it into the
				// anchor, so every remaining link holds. The header filter
				// still fixes where the anchor must be.
				f := vt.Build(fc.opts)
				first := f.Body[1]
				f.Body[1] = vt.Line{
					"kind": "anchor", "genesis": false, "seq": first["seq"], "prev_hash": first["prev_hash"],
					"entry_hash": first["entry_hash"], "envelope_sha256": first["envelope_sha256"],
				}
				f.Body = f.Body[1:]
				f.Trailer["anchor_seq"] = first["seq"]
				f.Recount()
				r := run(t, f.Bytes())
				wantTampered(t, r, "first entry deleted and re-anchored")
				if reason(r) != verify.ReasonAnchorMismatch {
					t.Errorf("reason %q, want %q", reason(r), verify.ReasonAnchorMismatch)
				}
			})
			t.Run("no-anchor", func(t *testing.T) {
				f := vt.Build(fc.opts)
				f.Body = f.Body[1:]
				r := run(t, f.Bytes())
				wantTampered(t, r, "anchor removed")
				if reason(r) != verify.ReasonAnchorMissing {
					t.Errorf("reason %q, want %q", reason(r), verify.ReasonAnchorMissing)
				}
			})
			t.Run("no-anchor-trailer-says-empty", func(t *testing.T) {
				f := vt.Build(fc.opts)
				f.Body = f.Body[1:]
				f.Trailer["anchor_seq"] = nil
				wantTampered(t, run(t, f.Bytes()), "anchor removed, trailer cleared")
			})
		})
	}
}

func flipHex(c byte) string {
	if c == '0' {
		return "1"
	}
	return "0"
}

func seqOf(t *testing.T, name string) int64 {
	t.Helper()
	for _, c := range vt.Chain() {
		if c.Name == name {
			return c.Seq
		}
	}
	t.Fatalf("no chain vector %s", name)
	return 0
}

func chainCase(t *testing.T, name string) vectors.ChainCase {
	t.Helper()
	for _, c := range vt.Chain() {
		if c.Name == name {
			return c
		}
	}
	t.Fatalf("no chain vector %s", name)
	return vectors.ChainCase{}
}

// CI-A7 (verifier half), R24.
func TestCIA7Erasure(t *testing.T) {
	erasedSeq, eraseSeq := seqOf(t, "erased-entry"), seqOf(t, "audit-erase")
	if eraseSeq != erasedSeq+1 {
		t.Fatalf("vectors moved: erase entry at %d, erased at %d", eraseSeq, erasedSeq)
	}
	t.Run("sealed-erase-entry-ok", func(t *testing.T) {
		f := vt.Build(vt.Options{Match: func(c vectors.ChainCase) bool { return c.Seq == erasedSeq }})
		if f.Body[f.Index(eraseSeq)]["kind"] != "entry" {
			t.Fatal("builder did not export the erase entry in full")
		}
		r := run(t, f.Bytes())
		wantOutcome(t, r, verify.OutcomePass)
		if len(r.Lines) != 0 {
			t.Errorf("line outcomes %v, want all verified", r.Lines)
		}
	})
	t.Run("no-erase-entry-tampered", func(t *testing.T) {
		r := run(t, vt.Build(vt.Options{HeadSeq: erasedSeq}).Bytes())
		wantTampered(t, r, "erasure without an erase entry")
		if reason(r) != verify.ReasonErasureUnlisted || r.Lines[erasedSeq] != verify.LineTampered {
			t.Errorf("reason %q, lines %v; want %q at seq %d", reason(r), r.Lines, verify.ReasonErasureUnlisted, erasedSeq)
		}
	})
	t.Run("unsealed-erase-entry-pending", func(t *testing.T) {
		f := vt.Build(vt.Options{HeadSeq: erasedSeq, Unsealed: []vectors.ChainCase{chainCase(t, "audit-erase")}})
		r := run(t, f.Bytes())
		wantOutcome(t, r, verify.OutcomePending)
		if r.Lines[erasedSeq] != verify.LinePending {
			t.Errorf("lines %v, want pending at seq %d", r.Lines, erasedSeq)
		}
		if r.UnsealedCount != 1 {
			t.Errorf("unsealed count %d, want 1", r.UnsealedCount)
		}
	})
	t.Run("unsealed-erase-entry-with-edited-diff-tampered", func(t *testing.T) {
		f := vt.Build(vt.Options{HeadSeq: erasedSeq, Unsealed: []vectors.ChainCase{chainCase(t, "audit-erase")}})
		u := f.Body[len(f.Body)-1]
		d := strings.Replace(*u["diff"].(*string), `"]`, `","0190f3a0-0000-7000-8000-000000000999"]`, 1)
		u["diff"] = &d
		wantTampered(t, run(t, f.Bytes()), "unsealed erase entry whose diff does not match its hash")
	})
	t.Run("erase-entry-for-another-ref-tampered", func(t *testing.T) {
		f := vt.Build(vt.Options{})
		e := f.Body[f.Index(erasedSeq)]
		other := "0190f3a0-0000-7000-8000-0000000000f9"
		e["erasure_ref"] = &other
		r := run(t, f.Bytes())
		wantTampered(t, r, "erasure_ref points at no erase entry")
		if reason(r) != verify.ReasonErasureUnlisted {
			t.Errorf("reason %q, want %q", reason(r), verify.ReasonErasureUnlisted)
		}
	})
	t.Run("erased-with-diff-left-tampered", func(t *testing.T) {
		f := vt.Build(vt.Options{})
		d := "[]"
		f.Body[f.Index(erasedSeq)]["diff"] = &d
		r := run(t, f.Bytes())
		wantTampered(t, r, "erased entry still carries a diff")
		if reason(r) != verify.ReasonDiffStateMismatch {
			t.Errorf("reason %q, want %q", reason(r), verify.ReasonDiffStateMismatch)
		}
	})
}

// CI-A14 (verifier half), R7.
func TestCIA14Redaction(t *testing.T) {
	seq := seqOf(t, "record-update-all-fields")
	t.Run("redacted-diff-not-verifiable", func(t *testing.T) {
		f := vt.Build(vt.Options{})
		l := f.Body[f.Index(seq)]
		d := `[{"op":"replace","path":"/status","value":{"redacted":"mask"}}]`
		l["diff"], l["diff_salt"], l["diff_redacted"] = &d, nil, true
		r := run(t, f.Bytes())
		wantOutcome(t, r, verify.OutcomePass)
		if r.Lines[seq] != verify.LineDiffUnverified {
			t.Errorf("line %d outcome %q, want %q", seq, r.Lines[seq], verify.LineDiffUnverified)
		}
	})
	t.Run("redacted-with-salt-tampered", func(t *testing.T) {
		f := vt.Build(vt.Options{})
		f.Body[f.Index(seq)]["diff_redacted"] = true
		r := run(t, f.Bytes())
		if reason(r) != verify.ReasonDiffStateMismatch {
			t.Errorf("reason %q, want %q", reason(r), verify.ReasonDiffStateMismatch)
		}
	})
	t.Run("unredacted-diff-verifies", func(t *testing.T) {
		r := run(t, vt.Build(vt.Options{}).Bytes())
		wantOutcome(t, r, verify.OutcomePass)
		if _, ok := r.Lines[seq]; ok {
			t.Errorf("line %d outcome %q, want verified", seq, r.Lines[seq])
		}
	})
	t.Run("unredacted-diff-edited-tampered", func(t *testing.T) {
		f := vt.Build(vt.Options{})
		d := *f.Body[f.Index(seq)]["diff"].(*string) + " "
		f.Body[f.Index(seq)]["diff"] = &d
		if r := run(t, f.Bytes()); reason(r) != verify.ReasonDiffMismatch {
			t.Errorf("reason %q, want %q", reason(r), verify.ReasonDiffMismatch)
		}
	})
}

// R15: a duplicate or missing seq is reported, naming the line.
func TestSeqDuplicateAndMissing(t *testing.T) {
	t.Run("duplicate", func(t *testing.T) {
		f := vt.Build(vt.Options{})
		i := f.Index(3)
		f.Body = append(f.Body[:i+1], f.Body[i:]...)
		r := run(t, f.Bytes())
		if reason(r) != verify.ReasonSeqDuplicate || *r.Failure.Seq != 3 || r.Failure.Line != i+3 {
			t.Errorf("got %s, want %s at seq 3, line %d", describe(r), verify.ReasonSeqDuplicate, i+3)
		}
	})
	t.Run("out-of-order", func(t *testing.T) {
		f := vt.Build(vt.Options{})
		i := f.Index(3)
		f.Body[i], f.Body[i+1] = f.Body[i+1], f.Body[i]
		wantTampered(t, run(t, f.Bytes()), "swapped lines")
	})
	t.Run("missing", func(t *testing.T) {
		f := vt.Build(vt.Options{})
		i := f.Index(4)
		f.Body = append(f.Body[:i], f.Body[i+1:]...)
		r := run(t, f.Bytes())
		if reason(r) != verify.ReasonSeqMissing || *r.Failure.Seq != 5 {
			t.Errorf("got %s, want %s at seq 5", describe(r), verify.ReasonSeqMissing)
		}
	})
}

// A stubs=false file honestly flagged is unverifiable; one that lies about
// its flags is tampered.
func TestStubsOmitted(t *testing.T) {
	strip := func(f *vt.File) {
		kept := f.Body[:0]
		for _, l := range f.Body {
			if l["kind"] != "stub" {
				kept = append(kept, l)
			}
		}
		f.Body = kept
		f.Header["stubs"] = false
		f.Recount()
	}
	record := "record"
	opts := vt.Options{SubjectType: &record, Match: func(c vectors.ChainCase) bool { return c.Envelope.SubjectType == record }}
	t.Run("honest", func(t *testing.T) {
		f := vt.Build(opts)
		strip(f)
		f.Trailer["verifiable"] = false
		r := run(t, f.Bytes())
		wantOutcome(t, r, verify.OutcomeUnverifiable)
		if r.Gaps == 0 {
			t.Error("no gaps counted")
		}
	})
	t.Run("trailer-claims-verifiable", func(t *testing.T) {
		f := vt.Build(opts)
		strip(f)
		if r := run(t, f.Bytes()); reason(r) != verify.ReasonVerifiableFlagMismatch {
			t.Errorf("got %s, want %s", describe(r), verify.ReasonVerifiableFlagMismatch)
		}
	})
	t.Run("header-claims-stubs", func(t *testing.T) {
		f := vt.Build(opts)
		strip(f)
		f.Header["stubs"] = true
		f.Trailer["verifiable"] = false
		if r := run(t, f.Bytes()); reason(r) != verify.ReasonSeqMissing {
			t.Errorf("got %s, want %s", describe(r), verify.ReasonSeqMissing)
		}
	})
}

// §4 steps 1 and 10: the file and every line are parsed strictly.
func TestStrictStructure(t *testing.T) {
	clean := vt.Build(vt.Options{}).Bytes()
	lines := strings.SplitAfter(string(clean), "\n")
	lines = lines[:len(lines)-1] // the empty string after the final "\n"
	join := func(ls ...string) []byte { return []byte(strings.Join(ls, "")) }
	n := len(lines)
	cases := []struct {
		name, reason string
		file         []byte
	}{
		{"empty-file", verify.ReasonHeaderInvalid, nil},
		{"no-header", verify.ReasonHeaderInvalid, join(lines[1:]...)},
		{"no-trailer", verify.ReasonTrailerMissing, join(lines[:n-1]...)},
		{"line-after-trailer", verify.ReasonLineOrder, join(append(slicesClone(lines), lines[1])...)},
		{"second-header", verify.ReasonLineOrder, join(append([]string{lines[0]}, lines...)...)},
		{"blank-line", verify.ReasonMalformedLine, join(lines[0], "\n", strings.Join(lines[1:], ""))},
		{"crlf", verify.ReasonMalformedLine, []byte(strings.ReplaceAll(string(clean), "\n", "\r\n"))},
		{"unknown-kind", verify.ReasonUnknownKind, join(lines[0], `{"kind":"note"}`+"\n", strings.Join(lines[1:], ""))},
		{"unknown-member", verify.ReasonMalformedLine, join(lines[0], strings.Replace(lines[1], `{`, `{"x":1,`, 1), strings.Join(lines[2:], ""))},
		{"duplicate-member", verify.ReasonMalformedLine, join(lines[0], strings.Replace(lines[1], `{`, `{"seq":0,`, 1), strings.Join(lines[2:], ""))},
		{"invalid-utf8", verify.ReasonMalformedLine, join(lines[0], "{\"kind\":\"\xff\"}\n", strings.Join(lines[1:], ""))},
		{"wrong-format", verify.ReasonHeaderInvalid, join(strings.Replace(lines[0], "export/v1", "export/v2", 1), strings.Join(lines[1:], ""))},
		{"non-canonical-envelope", verify.ReasonMalformedLine, join(lines[0], lines[1], strings.Replace(lines[2], `"domain":"config"`, `"domain":"con`+`\`+`u0066ig"`, 1), strings.Join(lines[3:], ""))},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := run(t, c.file)
			if reason(r) != c.reason {
				t.Errorf("got %s, want %s", describe(r), c.reason)
			}
		})
	}
	// A missing final newline is tolerated.
	wantOutcome(t, run(t, bytes.TrimSuffix(clean, []byte("\n"))), verify.OutcomePass)
}

func slicesClone(s []string) []string { return append([]string(nil), s...) }

// A tenant label swapped on an unsealed line is caught too (§4 step 4a).
func TestUnsealedTenantMismatch(t *testing.T) {
	c := chainCase(t, "config-create-minimal")
	c.Envelope.TenantID = chainCase(t, "audit-erase").Envelope.SubjectID // any other UUID
	f := vt.Build(vt.Options{Unsealed: []vectors.ChainCase{c}})
	if r := run(t, f.Bytes()); reason(r) != verify.ReasonHeaderTenantMismatch {
		t.Errorf("got %s, want %s", describe(r), verify.ReasonHeaderTenantMismatch)
	}
}

// An over-long line is malformed, not a read error.
func TestLineTooLong(t *testing.T) {
	big := append([]byte(`{"kind":"header","x":"`), bytes.Repeat([]byte("a"), verify.MaxLineBytes)...)
	if r := run(t, big); reason(r) != verify.ReasonMalformedLine {
		t.Errorf("got %s, want %s", describe(r), verify.ReasonMalformedLine)
	}
}

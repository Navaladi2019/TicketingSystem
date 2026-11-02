package main

import (
	"bytes"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/audit/chain"
	"github.com/Navaladi2019/TicketingSystem/internal/audit/chain/vectors"
	"github.com/Navaladi2019/TicketingSystem/internal/audit/verify"
	vt "github.com/Navaladi2019/TicketingSystem/internal/audit/verify/verifytest"
)

// stubsFalse turns f into what a stubs=false exporter streams (stub lines
// left out, header stubs false) and sets the trailer's verifiable the way the
// exporter must: with chain.TrailerVerifiable over the counts it kept and the
// head seq read at the start (ADR-004-E2, TICK-468).
func stubsFalse(t *testing.T, f *vt.File) {
	t.Helper()
	kept := f.Body[:0]
	anchor := false
	for _, l := range f.Body {
		switch l["kind"] {
		case "stub":
			continue
		case "anchor":
			anchor = true
		}
		kept = append(kept, l)
	}
	f.Body = kept
	f.Header["stubs"] = false
	f.Recount()
	n := func(k string) int64 {
		switch v := f.Trailer[k].(type) {
		case int:
			return int64(v)
		case int64:
			return v
		}
		t.Fatalf("trailer %s is %T", k, f.Trailer[k])
		return 0
	}
	f.Trailer["verifiable"] = chain.TrailerVerifiable(anchor, n("count"), n("stub_count"), n("last_seq"), n("anchor_seq"))
}

// verifyBoth runs the file through verify.Verify and through nava audit verify.
func verifyBoth(t *testing.T, f *vt.File) (verify.Outcome, int, string) {
	t.Helper()
	b := f.Bytes()
	rep, err := verify.Verify(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	code := run([]string{"audit", "verify", writeFile(t, b)}, nil, &out, &out)
	return rep.Outcome, code, out.String()
}

// Test 1: a stubs=false export in which every entry matches verifies pass.
func TestTrailerVerifiableStubsFalseAllMatch(t *testing.T) {
	f := vt.Build(vt.Options{})
	stubsFalse(t, f)
	if f.Trailer["verifiable"] != true {
		t.Errorf("TrailerVerifiable gave %v for a complete range", f.Trailer["verifiable"])
	}
	outcome, code, out := verifyBoth(t, f)
	if outcome != verify.OutcomePass {
		t.Errorf("verify outcome %q, want %q: %s", outcome, verify.OutcomePass, out)
	}
	if code != ExitOK {
		t.Errorf("nava audit verify exit %d, want %d: %s", code, ExitOK, out)
	}
}

// Test 2: a stubs=false window with sealed entries but no matching entry
// (the anchor, no range lines, count 0) verifies unverifiable, with its own
// non-zero exit code, not the tampered one.
func TestTrailerVerifiableStubsFalseNoneMatched(t *testing.T) {
	st := "erasure_request"
	f := vt.Build(vt.Options{HeadSeq: 5, SubjectType: &st, Match: func(vectors.ChainCase) bool { return false }})
	stubsFalse(t, f)
	if len(f.Body) != 1 || f.Body[0]["kind"] != "anchor" || f.Trailer["count"] != 0 {
		t.Fatalf("fixture is not the anchor alone with count 0: body %d lines, count %v", len(f.Body), f.Trailer["count"])
	}
	if f.Trailer["verifiable"] != false {
		t.Errorf("TrailerVerifiable gave %v for a range with no lines", f.Trailer["verifiable"])
	}
	outcome, code, out := verifyBoth(t, f)
	if outcome != verify.OutcomeUnverifiable {
		t.Errorf("verify outcome %q, want %q: %s", outcome, verify.OutcomeUnverifiable, out)
	}
	if code != ExitUnverifiable {
		t.Errorf("nava audit verify exit %d, want %d: %s", code, ExitUnverifiable, out)
	}
	if ExitUnverifiable == 0 || ExitUnverifiable == ExitTampered {
		t.Errorf("ExitUnverifiable %d must be non-zero and differ from ExitTampered %d", ExitUnverifiable, ExitTampered)
	}
}

package export_test

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/audit/chain"
	"github.com/Navaladi2019/TicketingSystem/internal/audit/chain/vectors"
	"github.com/Navaladi2019/TicketingSystem/internal/audit/export"
	"github.com/Navaladi2019/TicketingSystem/internal/audit/verify"
	vt "github.com/Navaladi2019/TicketingSystem/internal/audit/verify/verifytest"
)

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
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

func rowOf(c vectors.ChainCase, sealed bool) export.Row {
	r := export.Row{Sealed: sealed, Envelope: c.Envelope, Diff: c.Diff, Match: true}
	if sealed {
		r.Seq, r.PrevHash, r.EntryHash = c.Seq, must(chain.ParseHash32(c.PrevHash)), must(chain.ParseHash32(c.EntryHash))
		at := time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)
		r.SealedAt = &at
	}
	if c.DiffSalt != nil {
		s := must(chain.ParseSalt16(*c.DiffSalt))
		r.DiffSalt = &s
	}
	if c.ErasedAt != nil {
		at := must(time.Parse(time.RFC3339Nano, *c.ErasedAt))
		r.ErasedAt = &at
	}
	if c.ErasureRef != nil {
		u := uuid.MustParse(*c.ErasureRef)
		r.ErasureRef = &u
	}
	return r
}

// fakeSource is an in-memory log. BeforePage runs before each Page call: it
// is the hook that lets a test commit something between two pages.
type fakeSource struct {
	sealed     []export.Row
	unsealed   []export.Row
	BeforePage func(n int)
	pages      int
	eraseCalls map[uuid.UUID]int
}

func (f *fakeSource) FirstSeq(context.Context) (int64, bool, error) { return 1, true, nil }

func (f *fakeSource) Anchor(context.Context, int64) (p, e, s [32]byte, err error) { return }

func (f *fakeSource) Page(_ context.Context, from, to int64, limit int) ([]export.Row, error) {
	f.pages++
	if f.BeforePage != nil {
		f.BeforePage(f.pages)
	}
	var out []export.Row
	for _, r := range f.sealed {
		if r.Seq >= from && r.Seq <= to && len(out) < limit {
			out = append(out, r)
		}
	}
	return out, nil
}

func (f *fakeSource) Unsealed(context.Context) ([]export.Row, error) {
	var out []export.Row
	for _, r := range f.unsealed {
		if r.Match { // the Source returns the matching unsealed rows only
			out = append(out, r)
		}
	}
	return out, nil
}

func (f *fakeSource) EraseEntry(_ context.Context, ref uuid.UUID) (*export.Row, error) {
	if f.eraseCalls == nil {
		f.eraseCalls = map[uuid.UUID]int{}
	}
	f.eraseCalls[ref]++
	for _, rows := range [][]export.Row{f.sealed, f.unsealed} {
		for _, r := range rows {
			if r.Envelope.Domain == "audit" && r.Envelope.Action == "erase" && r.Envelope.SubjectID == ref {
				return &r, nil
			}
		}
	}
	return nil, nil
}

// scenario builds the CI-A6 extension: the head is read at the erased entry X
// (seq erasedSeq); the erase entry E commits after that, between two pages and
// before the page holding X.
func scenario(t *testing.T) (*fakeSource, export.Params, int64) {
	t.Helper()
	x, e := chainCase(t, "erased-entry"), chainCase(t, "audit-erase")
	src := &fakeSource{}
	for _, c := range vt.Chain() {
		if c.Seq <= x.Seq {
			src.sealed = append(src.sealed, rowOf(c, true))
		}
	}
	// E commits after the lease and head statement and before X's page.
	src.BeforePage = func(n int) {
		if n == int(x.Seq) { // page size 1: the page holding X is page x.Seq
			er := rowOf(e, false)
			er.Match = false // the filter does not select it; only R24 does
			src.unsealed = append(src.unsealed, er)
		}
	}
	head := src.sealed[len(src.sealed)-1]
	p := export.Params{
		TenantID: x.Envelope.TenantID, ExportID: uuid.MustParse("0190f3a0-0000-7000-8000-0000000000ee"),
		ExportedAt: time.Date(2026, 11, 2, 12, 30, 0, 0, time.UTC), Stubs: true,
		HeadSeq: x.Seq, HeadHash: head.EntryHash, PageSize: 1,
	}
	return src, p, x.Seq
}

func verifyFile(t *testing.T, b []byte) *verify.Report {
	t.Helper()
	r, err := verify.Verify(bytes.NewReader(b))
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	return r
}

func reasonOf(r *verify.Report) string {
	if r.Failure == nil {
		return ""
	}
	return r.Failure.Reason
}

// CI-A6 extension (TICK-613): an erasure commits after the lease and head
// statement and before the page holding the erased entry. The erase entry is
// outside the verified range, and the file must be pending, not tampered.
func TestEraseAfterHeadIsPending(t *testing.T) {
	src, p, erasedSeq := scenario(t)
	var out bytes.Buffer
	if err := export.Write(context.Background(), &out, src, p); err != nil {
		t.Fatal(err)
	}
	r := verifyFile(t, out.Bytes())
	if r.Outcome != verify.OutcomePending {
		t.Fatalf("outcome %s (reason %q), want pending\n%s", r.Outcome, reasonOf(r), out.String())
	}
	if r.Lines[erasedSeq] != verify.LinePending {
		t.Errorf("lines %v, want pending at seq %d", r.Lines, erasedSeq)
	}
	if r.UnsealedCount != 1 {
		t.Errorf("unsealed count %d, want 1", r.UnsealedCount)
	}
}

// Dropping the tail erase line leaves the erased entry unlisted (R24).
func TestDroppedTailEraseLineIsUnlisted(t *testing.T) {
	src, p, _ := scenario(t)
	var out bytes.Buffer
	if err := export.Write(context.Background(), &out, src, p); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSuffix(out.String(), "\n"), "\n")
	var kept []string
	dropped := 0
	for _, l := range lines {
		var m map[string]any
		if err := json.Unmarshal([]byte(l), &m); err != nil {
			t.Fatal(err)
		}
		if m["kind"] == "entry" && m["sealed"] == false {
			dropped++
			continue
		}
		if m["kind"] == "trailer" {
			m["unsealed_count"] = 0
			l = string(must(json.Marshal(m)))
		}
		kept = append(kept, l)
	}
	if dropped != 1 {
		t.Fatalf("dropped %d tail lines, want 1", dropped)
	}
	r := verifyFile(t, []byte(strings.Join(kept, "\n")+"\n"))
	if r.Outcome != verify.OutcomeTampered || reasonOf(r) != verify.ReasonErasureUnlisted {
		t.Fatalf("outcome %s reason %q, want tampered erasure_unlisted", r.Outcome, reasonOf(r))
	}
}

// The tail never repeats an erase entry, and each erasure_ref is fetched once.
func TestEraseEntryFetchedAndWrittenOnce(t *testing.T) {
	src, p, _ := scenario(t)
	x := chainCase(t, "erased-entry")
	// A second erased entry with the same erasure_ref.
	second := src.sealed[len(src.sealed)-1]
	second.Seq++
	second.Envelope.ID = uuid.MustParse("0190f3a0-0000-7000-8000-0000000009a1")
	src.sealed = append(src.sealed, second)
	p.HeadSeq = second.Seq
	var out bytes.Buffer
	if err := export.Write(context.Background(), &out, src, p); err != nil {
		t.Fatal(err)
	}
	if n := src.eraseCalls[uuid.MustParse(*x.ErasureRef)]; n != 1 {
		t.Errorf("EraseEntry called %d times for one erasure_ref, want 1", n)
	}
	if n := strings.Count(out.String(), `"action":"erase"`); n != 1 {
		t.Errorf("erase entry written %d times, want 1", n)
	}
}

// A sealed erase entry inside the range is written in full even when the
// filter would make it a stub, and is not repeated in the tail.
func TestSealedEraseEntryInRangeIsFullNotRepeated(t *testing.T) {
	x, e := chainCase(t, "erased-entry"), chainCase(t, "audit-erase")
	src := &fakeSource{}
	for _, c := range vt.Chain() {
		if c.Seq <= e.Seq {
			r := rowOf(c, true)
			r.Match = c.Seq == x.Seq // only X matches the filter
			src.sealed = append(src.sealed, r)
		}
	}
	p := export.Params{
		TenantID: x.Envelope.TenantID, ExportID: uuid.New(), ExportedAt: time.Now(), Stubs: true,
		HeadSeq: e.Seq, HeadHash: src.sealed[len(src.sealed)-1].EntryHash, PageSize: 2,
	}
	var out bytes.Buffer
	if err := export.Write(context.Background(), &out, src, p); err != nil {
		t.Fatal(err)
	}
	r := verifyFile(t, out.Bytes())
	if r.Outcome != verify.OutcomePass {
		t.Fatalf("outcome %s (reason %q), want pass", r.Outcome, reasonOf(r))
	}
	if r.UnsealedCount != 0 {
		t.Errorf("unsealed count %d, want 0 (erase entry is in range)", r.UnsealedCount)
	}
}

package export_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/audit/chain"
	"github.com/Navaladi2019/TicketingSystem/internal/audit/export"
	"github.com/Navaladi2019/TicketingSystem/internal/audit/read"
	"github.com/Navaladi2019/TicketingSystem/internal/audit/verify"
)

// fakeLog is a sealed chain of real hashes behind the Source port, so the
// exporter's output goes through the real verifier.
type fakeLog struct {
	tenant   uuid.UUID
	rows     []read.ExportRow // seq 1..n, index seq-1
	unsealed []read.ExportRow
	pages    int
	failAt   int // fail the nth ExportRange call (1-based); 0 never
	calls    int
}

func subjA() uuid.UUID { return uuid.MustParse("0190f3a0-0000-7000-8000-0000000000a1") }
func subjB() uuid.UUID { return uuid.MustParse("0190f3a0-0000-7000-8000-0000000000b1") }
func base() time.Time  { return time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC) }

func newFakeLog(t *testing.T, n int) *fakeLog {
	t.Helper()
	l := &fakeLog{tenant: uuid.MustParse("0190f3a0-0000-7000-8000-000000000001")}
	prev := chain.Genesis
	for i := 1; i <= n; i++ {
		subj := subjA()
		if i%2 == 0 {
			subj = subjB()
		}
		seq := int64(i)
		e := read.Entry{
			ID: uuid.MustParse("0190f3a0-0000-7000-8000-" + padHex(i)), OccurredAt: base().Add(time.Duration(i) * time.Minute),
			Domain: "record", Action: "update", SubjectType: "record", SubjectID: subj, Source: "user",
			ActorSystem: ptr("system"), Seq: &seq,
		}
		diff := []byte(`[{"op":"add","path":"/note","value":"<n` + padHex(i) + `>"}]`)
		salt := [16]byte{byte(i)}
		d := chain.DiffSHA256(salt, diff)
		e.DiffSHA256, e.Diff, e.DiffSalt = d[:], diff, salt[:]
		row := read.ExportRow{TenantID: l.tenant, Entry: e}
		env := envOf(row)
		sha, err := chain.EnvelopeSHA256(env)
		if err != nil {
			t.Fatal(err)
		}
		eh := chain.EntryHash(prev, sha)
		row.PrevHash, row.EntryHash = append([]byte(nil), prev[:]...), eh[:]
		sealed := e.OccurredAt
		row.Entry.SealedAt = &sealed
		l.rows = append(l.rows, row)
		prev = eh
	}
	return l
}

func padHex(i int) string { return strings.Repeat("0", 12-len(itoa(i))) + itoa(i) }
func itoa(i int) string   { b, _ := json.Marshal(i); return string(b) }
func ptr[T any](v T) *T   { return &v }

func envOf(r read.ExportRow) chain.Envelope {
	e := r.Entry
	h := func(b []byte) *[32]byte {
		if len(b) != 32 {
			return nil
		}
		var a [32]byte
		copy(a[:], b)
		return &a
	}
	return chain.Envelope{TenantID: r.TenantID, ID: e.ID, OccurredAt: e.OccurredAt, Domain: e.Domain, Action: e.Action,
		SubjectType: e.SubjectType, SubjectID: e.SubjectID, ActorSystem: e.ActorSystem, Source: e.Source, DiffSHA256: h(e.DiffSHA256)}
}

func (l *fakeLog) head() export.Session {
	last := l.rows[len(l.rows)-1]
	var h [32]byte
	copy(h[:], last.EntryHash)
	return export.Session{TenantID: l.tenant, ExportID: uuid.New(), ExportedAt: base().Add(24 * time.Hour), HeadSeq: int64(len(l.rows)), HeadHash: h}
}

func matches(f read.ExportFilter, r read.ExportRow) bool {
	e := r.Entry
	return (f.SubjectID == nil || *f.SubjectID == e.SubjectID) &&
		(f.From == nil || !e.OccurredAt.Before(*f.From)) && (f.To == nil || e.OccurredAt.Before(*f.To)) &&
		(f.Domain == nil || *f.Domain == e.Domain)
}

func (l *fakeLog) ExportFirstSeq(_ context.Context, _ uuid.UUID, f read.ExportFilter, head int64, _ read.ExportPageOptions) (int64, bool, error) {
	for _, r := range l.rows {
		e := r.Entry
		if *e.Seq <= head && (f.From == nil || !e.OccurredAt.Before(*f.From)) && (f.To == nil || e.OccurredAt.Before(*f.To)) {
			return *e.Seq, true, nil
		}
	}
	return 0, false, nil
}

func (l *fakeLog) ExportErasureRefs(context.Context, uuid.UUID, read.ExportFilter, read.ExportPageOptions) ([]uuid.UUID, error) {
	return nil, nil
}

func (l *fakeLog) ExportSeq(_ context.Context, _ uuid.UUID, seq int64, _ read.ExportPageOptions) (read.ExportRow, bool, error) {
	if seq < 1 || seq > int64(len(l.rows)) {
		return read.ExportRow{}, false, nil
	}
	r := l.rows[seq-1]
	r.Entry.Diff, r.Entry.DiffSalt = nil, nil
	return r, true, nil
}

func (l *fakeLog) ExportRange(_ context.Context, _ uuid.UUID, f read.ExportFilter, _ []uuid.UUID, after, head int64, limit int, only bool, _ read.ExportPageOptions) ([]read.ExportRow, error) {
	l.calls++
	l.pages++
	if l.failAt != 0 && l.calls == l.failAt {
		return nil, errors.New("page failed")
	}
	var out []read.ExportRow
	for _, r := range l.rows {
		if *r.Entry.Seq <= after || *r.Entry.Seq > head {
			continue
		}
		r.Match = matches(f, r)
		if only && !r.Match {
			continue
		}
		if !r.Match {
			r.Entry.Diff, r.Entry.DiffSalt = nil, nil
		}
		out = append(out, r)
		if len(out) == limit {
			break
		}
	}
	return out, nil
}

func (l *fakeLog) ExportUnsealed(_ context.Context, _ uuid.UUID, _ read.ExportFilter, _ []uuid.UUID, _ int64, _ time.Time, after *read.Position, limit int, _ read.ExportPageOptions) ([]read.ExportRow, error) {
	if after != nil {
		return nil, nil
	}
	return l.unsealed, nil
}

func run(t *testing.T, l *fakeLog, pageSize int, req export.Request) ([]byte, error) {
	t.Helper()
	var buf bytes.Buffer
	ex := &export.Exporter{Source: l, PageSize: pageSize}
	s := l.head()
	s.Request = req
	err := ex.Write(t.Context(), &buf, nil, s)
	return buf.Bytes(), err
}

func verdict(t *testing.T, b []byte) *verify.Report {
	t.Helper()
	rep, err := verify.Verify(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	return rep
}

func mustPass(t *testing.T, b []byte) *verify.Report {
	t.Helper()
	rep := verdict(t, b)
	if rep.Outcome != verify.OutcomePass {
		t.Fatalf("outcome = %s, failure %+v\n%s", rep.Outcome, rep.Failure, b)
	}
	return rep
}

func TestUnfilteredExportVerifies(t *testing.T) {
	t.Parallel()
	l := newFakeLog(t, 12)
	b, err := run(t, l, 5, export.Request{Stubs: true})
	if err != nil {
		t.Fatal(err)
	}
	rep := mustPass(t, b)
	if rep.Count != 12 || rep.StubCount != 0 || *rep.AnchorSeq != 0 {
		t.Errorf("count %d stubs %d anchor %d, want 12 0 0", rep.Count, rep.StubCount, *rep.AnchorSeq)
	}
	if l.pages != 3 {
		t.Errorf("12 rows in pages of 5 took %d page reads, want 3", l.pages)
	}
}

// CI-A6: filtered by subject, by time and by domain.
func TestFilteredExportsVerify(t *testing.T) {
	t.Parallel()
	from, to := base().Add(4*time.Minute), base().Add(10*time.Minute)
	cases := map[string]read.ExportFilter{
		"subject": {SubjectID: ptr(subjA())},
		"time":    {From: &from, To: &to},
		"domain":  {Domain: ptr("record"), SubjectID: ptr(subjB())},
		"both":    {From: &from, To: &to, SubjectID: ptr(subjA())},
	}
	for name, f := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			b, err := run(t, newFakeLog(t, 12), 4, export.Request{Stubs: true, Filter: f})
			if err != nil {
				t.Fatal(err)
			}
			rep := mustPass(t, b)
			if f.From != nil && *rep.AnchorSeq != 3 {
				t.Errorf("anchor seq = %d, want 3 (the entry before the first one in the window)", *rep.AnchorSeq)
			}
			if rep.StubCount == 0 {
				t.Error("a filter that excludes entries produced no stubs")
			}
		})
	}
}

func lines(b []byte) []string { return strings.Split(strings.TrimSuffix(string(b), "\n"), "\n") }

// CI-A6: each tamper fails under a filter.
func TestTamperedExportsFail(t *testing.T) {
	t.Parallel()
	f := read.ExportFilter{SubjectID: ptr(subjA())}
	b, err := run(t, newFakeLog(t, 10), 100, export.Request{Stubs: true, Filter: f})
	if err != nil {
		t.Fatal(err)
	}
	mustPass(t, b)
	ls := lines(b)
	join := func(ls []string) []byte { return []byte(strings.Join(ls, "\n") + "\n") }
	cp := func() []string { return append([]string(nil), ls...) }

	byteEdit := cp()
	byteEdit[2] = strings.Replace(byteEdit[2], `"record"`, `"recorc"`, 1) // first entry after the anchor: seq 1 is a match
	deleted := cp()
	deleted = append(deleted[:4], deleted[5:]...)
	rewrite := cp()
	rewrite[3] = strings.Replace(rewrite[3], `"seq":2`, `"seq":2`, 1)
	rewrite[3] = rewriteEntryHash(t, rewrite[3])
	noFirst := cp()
	noFirst = append(noFirst[:2], noFirst[3:]...) // header, anchor, then seq 1 gone

	for name, got := range map[string][]string{"byte edit": byteEdit, "deleted row": deleted, "single-row rewrite": rewrite, "deleted first entry": noFirst} {
		if rep := verdict(t, join(got)); rep.Outcome != verify.OutcomeTampered {
			t.Errorf("%s: outcome %s, want tampered", name, rep.Outcome)
		}
	}
}

// rewriteEntryHash flips the first hex digit of a line's entry_hash.
func rewriteEntryHash(t *testing.T, line string) string {
	t.Helper()
	var m map[string]json.RawMessage
	if err := json.Unmarshal([]byte(line), &m); err != nil {
		t.Fatal(err)
	}
	var h string
	_ = json.Unmarshal(m["entry_hash"], &h)
	c := byte('0')
	if h[0] == '0' {
		c = '1'
	}
	return strings.Replace(line, h, string(c)+h[1:], 1)
}

// ADR-004-E2 acceptance additions, against the real verifier.
func TestStubsFalse(t *testing.T) {
	t.Parallel()
	t.Run("every entry matches: pass", func(t *testing.T) {
		t.Parallel()
		b, err := run(t, newFakeLog(t, 6), 100, export.Request{Stubs: false})
		if err != nil {
			t.Fatal(err)
		}
		mustPass(t, b)
	})
	t.Run("filter excludes the tail: trailer last_seq is the head read at the start", func(t *testing.T) {
		t.Parallel()
		l := newFakeLog(t, 6)
		to := base().Add(4 * time.Minute) // seq 1..3 match; 4..6 do not
		b, err := run(t, l, 100, export.Request{Stubs: false, Filter: read.ExportFilter{To: &to, SubjectID: ptr(subjA())}})
		if err != nil {
			t.Fatal(err)
		}
		ls := lines(b)
		var tr struct {
			LastSeq    int64 `json:"last_seq"`
			Verifiable bool  `json:"verifiable"`
		}
		if err := json.Unmarshal([]byte(ls[len(ls)-1]), &tr); err != nil {
			t.Fatal(err)
		}
		if tr.LastSeq != 6 || tr.Verifiable {
			t.Errorf("trailer last_seq %d verifiable %v, want 6 false", tr.LastSeq, tr.Verifiable)
		}
		if rep := verdict(t, b); rep.Outcome != verify.OutcomeUnverifiable {
			t.Errorf("outcome %s, want unverifiable", rep.Outcome)
		}
	})
	t.Run("sealed entries in the window but none match: unverifiable, not tampered", func(t *testing.T) {
		t.Parallel()
		other := uuid.New()
		b, err := run(t, newFakeLog(t, 6), 100, export.Request{Stubs: false, Filter: read.ExportFilter{SubjectID: &other}})
		if err != nil {
			t.Fatal(err)
		}
		if rep := verdict(t, b); rep.Outcome != verify.OutcomeUnverifiable {
			t.Errorf("outcome %s (%+v), want unverifiable", rep.Outcome, rep.Failure)
		}
	})
}

func TestEmptyWindowHasNoAnchor(t *testing.T) {
	t.Parallel()
	from := base().Add(100 * time.Hour)
	b, err := run(t, newFakeLog(t, 4), 100, export.Request{Stubs: true, Filter: read.ExportFilter{From: &from}})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), `"kind":"anchor"`) {
		t.Error("an empty window wrote an anchor")
	}
	if rep := verdict(t, b); rep.Outcome != verify.OutcomeEmpty {
		t.Errorf("outcome %s, want empty", rep.Outcome)
	}
}

func TestAfterSeqAnchorsOnThatSeq(t *testing.T) {
	t.Parallel()
	b, err := run(t, newFakeLog(t, 8), 100, export.Request{Stubs: true, Filter: read.ExportFilter{AfterSeq: ptr(int64(5))}})
	if err != nil {
		t.Fatal(err)
	}
	rep := mustPass(t, b)
	if *rep.AnchorSeq != 5 || rep.Count+rep.StubCount != 3 {
		t.Errorf("anchor %d, lines %d, want anchor 5 and 3 lines", *rep.AnchorSeq, rep.Count+rep.StubCount)
	}
}

// R7, CI-A14: a redacted entry carries no diff_salt.
func TestRedactedEntryCarriesNoSalt(t *testing.T) {
	t.Parallel()
	l := newFakeLog(t, 3)
	l.rows[1].Entry.DiffRedacted, l.rows[1].Entry.DiffSalt = true, nil
	b, err := run(t, l, 100, export.Request{Stubs: true})
	if err != nil {
		t.Fatal(err)
	}
	var seen int
	for _, ln := range lines(b) {
		var e struct {
			Kind     string  `json:"kind"`
			Seq      int64   `json:"seq"`
			DiffSalt *string `json:"diff_salt"`
			Redacted bool    `json:"diff_redacted"`
		}
		if err := json.Unmarshal([]byte(ln), &e); err != nil {
			t.Fatal(err)
		}
		if e.Kind != "entry" {
			continue
		}
		seen++
		if (e.Seq == 2) != (e.DiffSalt == nil) || (e.Seq == 2) != e.Redacted {
			t.Errorf("seq %d: salt %v redacted %v", e.Seq, e.DiffSalt, e.Redacted)
		}
	}
	if seen != 3 {
		t.Fatalf("saw %d entries", seen)
	}
	if rep := verdict(t, b); rep.Outcome != verify.OutcomePass && rep.Lines[2] != verify.LineDiffUnverified {
		t.Errorf("redacted line outcome %v, file %s", rep.Lines[2], rep.Outcome)
	}
}

// Unsealed matches come last, flagged, and are not part of the verified range.
func TestUnsealedMatchesComeLast(t *testing.T) {
	t.Parallel()
	l := newFakeLog(t, 3)
	u := l.rows[2]
	u.Entry.ID = uuid.New()
	u.Entry.Seq, u.Entry.SealedAt, u.PrevHash, u.EntryHash, u.Match = nil, nil, nil, nil, true
	l.unsealed = []read.ExportRow{u}
	b, err := run(t, l, 100, export.Request{Stubs: true})
	if err != nil {
		t.Fatal(err)
	}
	rep := mustPass(t, b)
	if rep.UnsealedCount != 1 {
		t.Errorf("unsealed count %d, want 1", rep.UnsealedCount)
	}
	ls := lines(b)
	if !strings.Contains(ls[len(ls)-2], `"sealed":false,"in_verified_range":false,"seq":null`) {
		t.Errorf("the line before the trailer is not the unsealed one: %s", ls[len(ls)-2])
	}
}

// A page that fails after the first byte leaves a file with no trailer.
func TestPageFailureLeavesNoTrailer(t *testing.T) {
	t.Parallel()
	l := newFakeLog(t, 10)
	l.failAt = 2
	b, err := run(t, l, 4, export.Request{Stubs: true})
	if err == nil {
		t.Fatal("a failing page returned nil")
	}
	if strings.Contains(string(b), `"kind":"trailer"`) {
		t.Error("a failed export wrote a trailer")
	}
	if rep := verdict(t, b); rep.Outcome != verify.OutcomeTampered {
		t.Errorf("a truncated file verified as %s", rep.Outcome)
	}
}

// The header's stubs is the request, written first; the filter is echoed.
func TestHeaderRecordsTheRequest(t *testing.T) {
	t.Parallel()
	b, err := run(t, newFakeLog(t, 2), 100, export.Request{Stubs: false, Filter: read.ExportFilter{SubjectID: ptr(subjA())}})
	if err != nil {
		t.Fatal(err)
	}
	var h struct {
		Kind   string `json:"kind"`
		Format string `json:"format"`
		Stubs  bool   `json:"stubs"`
		Filter struct {
			SubjectID *string `json:"subject_id"`
			Domain    *string `json:"domain"`
		} `json:"filter"`
	}
	if err := json.Unmarshal([]byte(lines(b)[0]), &h); err != nil {
		t.Fatal(err)
	}
	if h.Kind != "header" || h.Format != export.Format || h.Stubs || h.Filter.SubjectID == nil || *h.Filter.SubjectID != subjA().String() || h.Filter.Domain != nil {
		t.Errorf("header = %+v", h)
	}
}

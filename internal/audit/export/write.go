package export

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/audit/chain"
	"github.com/Navaladi2019/TicketingSystem/internal/audit/read"
)

// Format is the header's format id.
const Format = "nava.audit.export/v1"

// Source is the page reads the exporter needs; *read.Reader implements it.
type Source interface {
	ExportFirstSeq(ctx context.Context, tenantID uuid.UUID, f read.ExportFilter, head int64, o read.ExportPageOptions) (int64, bool, error)
	ExportErasureRefs(ctx context.Context, tenantID uuid.UUID, f read.ExportFilter, o read.ExportPageOptions) ([]uuid.UUID, error)
	ExportSeq(ctx context.Context, tenantID uuid.UUID, seq int64, o read.ExportPageOptions) (read.ExportRow, bool, error)
	ExportRange(ctx context.Context, tenantID uuid.UUID, f read.ExportFilter, erased []uuid.UUID, after, head int64, limit int, onlyMatches bool, o read.ExportPageOptions) ([]read.ExportRow, error)
	// ExportEraseEntries finds the audit/erase entries for refs, whatever their
	// occurred_at or seq (R24, ADR-004-E8 Follow-up).
	ExportEraseEntries(ctx context.Context, tenantID uuid.UUID, refs []uuid.UUID, o read.ExportPageOptions) ([]read.ExportRow, error)
	ExportUnsealed(ctx context.Context, tenantID uuid.UUID, f read.ExportFilter, erased []uuid.UUID, head int64, start time.Time, after *read.Position, limit int, o read.ExportPageOptions) ([]read.ExportRow, error)
}

// Exporter writes the JSONL stream. Zero values take the R21 defaults.
type Exporter struct {
	Source           Source
	PageSize         int
	StatementTimeout time.Duration
	Cap              time.Duration
	// OnPage is a test hook, see read.ExportPageOptions.
	OnPage func(txStart time.Time, statementTimeout string)
}

func (e *Exporter) pageSize() int {
	if e.PageSize > 0 {
		return e.PageSize
	}
	return DefaultPageSize
}

func (e *Exporter) opts() read.ExportPageOptions {
	st := e.StatementTimeout
	if st <= 0 {
		st = DefaultStatementTimeout
	}
	return read.ExportPageOptions{StatementTimeout: st, OnPage: e.OnPage}
}

// Deadline is how long Write may run (R21: 15 minutes); callers extend the
// response's write deadline by it.
func (e *Exporter) Deadline() time.Duration {
	if e.Cap > 0 {
		return e.Cap
	}
	return DefaultCap
}

// Write streams the export of s to w (audit-export-v1.md section 3): header,
// anchor, the sealed range, unsealed matches, trailer. flush, when not nil, is
// called after each page. An error after the first byte leaves the file
// without its trailer, which the verifier rejects as truncated. Write stops at
// the cap (R21).
func (e *Exporter) Write(ctx context.Context, w io.Writer, flush func(), s Session) error {
	ctx, cancel := context.WithTimeout(ctx, e.Deadline())
	defer cancel()
	bw := bufio.NewWriterSize(w, 64<<10)
	enc := json.NewEncoder(bw)
	enc.SetEscapeHTML(false) // the canonical envelope must keep < > & raw
	emit := func(v any) error { return enc.Encode(v) }
	endPage := func() error {
		if err := bw.Flush(); err != nil {
			return err
		}
		if flush != nil {
			flush()
		}
		return nil
	}
	o := e.opts()
	req := s.Request

	if err := emit(headerLine{Kind: "header", Format: Format, TenantID: s.TenantID.String(), ExportID: s.ExportID.String(),
		ExportedAt: ts(s.ExportedAt), Stubs: req.Stubs, Filter: filterJSON(req.Filter)}); err != nil {
		return err
	}

	erased, err := e.Source.ExportErasureRefs(ctx, s.TenantID, req.Filter, o)
	if err != nil {
		return err
	}

	// R20: the first sealed seq in the time window, whatever the filter says.
	first, found := int64(0), false
	switch {
	case s.HeadSeq <= 0:
	case req.Filter.AfterSeq != nil:
		if f := *req.Filter.AfterSeq + 1; f <= s.HeadSeq {
			first, found = f, true
		}
	default:
		if first, found, err = e.Source.ExportFirstSeq(ctx, s.TenantID, req.Filter, s.HeadSeq, o); err != nil {
			return err
		}
	}

	var count, stubCount, unsealed int64
	// wanted is every erasure_ref of an erased entry written, in first-seen
	// order; each is fetched once below.
	wanted, wantedSet := []uuid.UUID{}, map[uuid.UUID]bool{}
	want := func(r read.ExportRow) {
		if r.Entry.ErasedAt != nil && r.ErasureRef != nil && !wantedSet[*r.ErasureRef] {
			wantedSet[*r.ErasureRef] = true
			wanted = append(wanted, *r.ErasureRef)
		}
	}
	written := map[uuid.UUID]bool{} // ids written in the tail
	anchorSeq := int64(0)
	if found {
		anchorSeq = first - 1
		if err := e.writeAnchor(ctx, emit, s.TenantID, anchorSeq, o); err != nil {
			return err
		}
		cursor := anchorSeq
		for {
			rows, err := e.Source.ExportRange(ctx, s.TenantID, req.Filter, erased, cursor, s.HeadSeq, e.pageSize(), !req.Stubs, o)
			if err != nil {
				return err
			}
			for _, r := range rows {
				if r.Entry.Seq == nil {
					return errors.New("audit/export: a row in the sealed range has no seq")
				}
				if r.Match {
					line, err := entryLine(r, true)
					if err != nil {
						return err
					}
					count++
					want(r)
					if err = emit(line); err != nil {
						return err
					}
				} else {
					line, err := stubLine(r)
					if err != nil {
						return err
					}
					stubCount++
					if err = emit(line); err != nil {
						return err
					}
				}
				cursor = *r.Entry.Seq
			}
			if err := endPage(); err != nil {
				return err
			}
			if len(rows) < e.pageSize() {
				break
			}
		}
	}

	var after *read.Position
	for {
		rows, err := e.Source.ExportUnsealed(ctx, s.TenantID, req.Filter, erased, s.HeadSeq, s.ExportedAt, after, e.pageSize(), o)
		if err != nil {
			return err
		}
		for _, r := range rows {
			line, err := entryLine(r, false)
			if err != nil {
				return err
			}
			unsealed++
			want(r)
			written[r.Entry.ID] = true
			if err = emit(line); err != nil {
				return err
			}
			after = &read.Position{At: r.Entry.OccurredAt, ID: r.Entry.ID}
		}
		if err := endPage(); err != nil {
			return err
		}
		if len(rows) < e.pageSize() {
			break
		}
	}

	// R24 (ADR-004-E8 Follow-up): for each erased entry written, the matching
	// audit/erase entry is exported whatever its occurred_at. One in the
	// verified range is already there; one that is unsealed or beyond head
	// goes in the tail, once. It may have committed after the export start,
	// so the unsealed query's occurred_at bound did not see it.
	for len(wanted) > 0 {
		n := min(len(wanted), e.pageSize())
		rows, err := e.Source.ExportEraseEntries(ctx, s.TenantID, wanted[:n], o)
		if err != nil {
			return err
		}
		wanted = wanted[n:]
		for _, r := range rows {
			if (r.Entry.Seq != nil && *r.Entry.Seq <= s.HeadSeq) || written[r.Entry.ID] {
				continue
			}
			line, err := entryLine(r, false)
			if err != nil {
				return err
			}
			unsealed++
			written[r.Entry.ID] = true
			if err = emit(line); err != nil {
				return err
			}
		}
		if err := endPage(); err != nil {
			return err
		}
	}

	t := trailerLine{Kind: "trailer", Count: count, StubCount: stubCount, UnsealedCount: unsealed}
	if found {
		// last_seq is the head read at the start, never the last seq written
		// (ADR-004-E2).
		last, hash := s.HeadSeq, chain.Hex(s.HeadHash[:])
		t.AnchorSeq, t.LastSeq, t.HeadHash = &anchorSeq, &last, &hash
	}
	t.Verifiable = chain.TrailerVerifiable(found, count, stubCount, s.HeadSeq, anchorSeq)
	if err := emit(t); err != nil {
		return err
	}
	return endPage()
}

func (e *Exporter) writeAnchor(ctx context.Context, emit func(any) error, tenantID uuid.UUID, seq int64, o read.ExportPageOptions) error {
	if seq == 0 {
		return emit(anchorLine{Kind: "anchor", Genesis: true, Seq: 0, EntryHash: chain.Hex(chain.Genesis[:])})
	}
	r, ok, err := e.Source.ExportSeq(ctx, tenantID, seq, o)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("audit/export: the anchor seq %d is not in the log", seq)
	}
	env, err := envelopeSHA(r)
	if err != nil {
		return err
	}
	prev, es := chain.Hex(r.PrevHash), chain.Hex(env[:])
	return emit(anchorLine{Kind: "anchor", Seq: seq, PrevHash: &prev, EntryHash: chain.Hex(r.EntryHash), EnvelopeSHA: &es})
}

// ---- lines

type filterLine struct {
	From        *string `json:"from"`
	To          *string `json:"to"`
	Domain      *string `json:"domain"`
	SubjectType *string `json:"subject_type"`
	SubjectID   *string `json:"subject_id"`
	ChangeSetID *string `json:"change_set_id"`
	AfterSeq    *int64  `json:"after_seq"`
}

type headerLine struct {
	Kind       string     `json:"kind"`
	Format     string     `json:"format"`
	TenantID   string     `json:"tenant_id"`
	ExportID   string     `json:"export_id"`
	ExportedAt string     `json:"exported_at"`
	Stubs      bool       `json:"stubs"`
	Filter     filterLine `json:"filter"`
}

type anchorLine struct {
	Kind        string  `json:"kind"`
	Genesis     bool    `json:"genesis"`
	Seq         int64   `json:"seq"`
	PrevHash    *string `json:"prev_hash"`
	EntryHash   string  `json:"entry_hash"`
	EnvelopeSHA *string `json:"envelope_sha256"`
}

type stubOut struct {
	Kind        string `json:"kind"`
	Seq         int64  `json:"seq"`
	PrevHash    string `json:"prev_hash"`
	EntryHash   string `json:"entry_hash"`
	EnvelopeSHA string `json:"envelope_sha256"`
}

type entryOut struct {
	Kind            string         `json:"kind"`
	Sealed          bool           `json:"sealed"`
	InVerifiedRange bool           `json:"in_verified_range"`
	Seq             *int64         `json:"seq"`
	PrevHash        *string        `json:"prev_hash"`
	EntryHash       *string        `json:"entry_hash"`
	EnvelopeSHA     string         `json:"envelope_sha256"`
	SealedAt        *string        `json:"sealed_at"`
	Envelope        chain.Envelope `json:"envelope"`
	Diff            *string        `json:"diff"`
	DiffSalt        *string        `json:"diff_salt"`
	DiffRedacted    bool           `json:"diff_redacted"`
	ErasedAt        *string        `json:"erased_at"`
	ErasureRef      *string        `json:"erasure_ref"`
}

type trailerLine struct {
	Kind          string  `json:"kind"`
	AnchorSeq     *int64  `json:"anchor_seq"`
	LastSeq       *int64  `json:"last_seq"`
	HeadHash      *string `json:"head_hash"`
	Count         int64   `json:"count"`
	StubCount     int64   `json:"stub_count"`
	UnsealedCount int64   `json:"unsealed_count"`
	Verifiable    bool    `json:"verifiable"`
}

func stubLine(r read.ExportRow) (stubOut, error) {
	env, err := envelopeSHA(r)
	if err != nil {
		return stubOut{}, err
	}
	return stubOut{Kind: "stub", Seq: *r.Entry.Seq, PrevHash: chain.Hex(r.PrevHash), EntryHash: chain.Hex(r.EntryHash), EnvelopeSHA: chain.Hex(env[:])}, nil
}

// entryLine builds a full line. sealed false is an unsealed line: its seq and
// hash members are null even for a row sealed after the head was read.
func entryLine(r read.ExportRow, sealed bool) (entryOut, error) {
	e := r.Entry
	envelope := buildEnvelope(r)
	sha, err := chain.EnvelopeSHA256(envelope)
	if err != nil {
		return entryOut{}, fmt.Errorf("audit/export: entry %s: %w", e.ID, err)
	}
	out := entryOut{Kind: "entry", Sealed: sealed, InVerifiedRange: sealed, EnvelopeSHA: chain.Hex(sha[:]), Envelope: envelope,
		DiffRedacted: e.DiffRedacted, ErasedAt: tsp(e.ErasedAt)}
	if sealed {
		out.Seq = e.Seq
		ph, eh := chain.Hex(r.PrevHash), chain.Hex(r.EntryHash)
		out.PrevHash, out.EntryHash = &ph, &eh
		out.SealedAt = tsp(e.SealedAt)
	}
	if e.Diff != nil {
		d := string(e.Diff)
		out.Diff = &d
	}
	if len(e.DiffSalt) > 0 { // the reader already dropped it with a redacted diff (R7)
		s := chain.Hex(e.DiffSalt)
		out.DiffSalt = &s
	}
	if r.ErasureRef != nil {
		s := r.ErasureRef.String()
		out.ErasureRef = &s
	}
	return out, nil
}

func envelopeSHA(r read.ExportRow) ([32]byte, error) {
	sha, err := chain.EnvelopeSHA256(buildEnvelope(r))
	if err != nil {
		return sha, fmt.Errorf("audit/export: entry %s: %w", r.Entry.ID, err)
	}
	return sha, nil
}

func buildEnvelope(r read.ExportRow) chain.Envelope {
	e := r.Entry
	return chain.Envelope{
		TenantID: r.TenantID, ID: e.ID, OccurredAt: e.OccurredAt.UTC(), Domain: e.Domain, Action: e.Action,
		SubjectType: e.SubjectType, SubjectID: e.SubjectID, EntityDefinitionID: e.EntityDefinitionID,
		VersionBefore: e.VersionBefore, VersionAfter: e.VersionAfter, ActorPrincipalID: e.ActorPrincipalID,
		ActorSystem: e.ActorSystem, Source: e.Source, SourceRef: e.SourceRef, ChangeSetID: e.ChangeSetID,
		CausationID: e.CausationID, CausationDepth: e.CausationDepth, RequestID: e.RequestID, Backfilled: e.Backfilled,
		ContentSHA256: hash32(e.ContentSHA256), DiffSHA256: hash32(e.DiffSHA256), DiffTruncated: e.DiffTruncated,
	}
}

func hash32(b []byte) *[32]byte {
	if len(b) != 32 {
		return nil
	}
	var h [32]byte
	copy(h[:], b)
	return &h
}

func ts(t time.Time) string { return t.UTC().Format(chain.TimeLayout) }

func tsp(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := ts(*t)
	return &s
}

func filterJSON(f read.ExportFilter) filterLine {
	out := filterLine{From: tsp(f.From), To: tsp(f.To), Domain: f.Domain, SubjectType: f.SubjectType, AfterSeq: f.AfterSeq}
	if f.SubjectID != nil {
		s := f.SubjectID.String()
		out.SubjectID = &s
	}
	if f.ChangeSetID != nil {
		s := f.ChangeSetID.String()
		out.ChangeSetID = &s
	}
	return out
}

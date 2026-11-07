// Package export writes the nava.audit.export/v1 JSONL file
// (docs/api/audit-export-v1.md §3; ADR-004 R19–R21, R24, ADR-004-E2).
//
// The writer streams: header, anchor, the sealed range in pages (each page one
// Source call, so the caller makes it one short transaction), the unsealed
// tail, then the trailer. It keeps only counters and the small sets R24 needs
// (the erase entries to include), never the range.
//
// Everything the Source returns comes from the reader pool under the export's
// tenant scope; this package has no database access of its own.
package export

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/audit/chain"
)

// Format is the header's format member.
const Format = "nava.audit.export/v1"

// DefaultPageSize is the rows per page (R21).
const DefaultPageSize = 5000

// Row is one audit_log row as the exporter needs it.
type Row struct {
	// A sealed row carries Seq, PrevHash, EntryHash and SealedAt. An unsealed
	// row has Sealed false and the rest zero.
	Sealed    bool
	Seq       int64
	PrevHash  [32]byte
	EntryHash [32]byte
	SealedAt  *time.Time

	Envelope     chain.Envelope
	Diff         *string // stored diff bytes, or the masked diff; nil when none or erased
	DiffSalt     *[16]byte
	DiffRedacted bool
	ErasedAt     *time.Time
	ErasureRef   *uuid.UUID

	// Match is true when the row passes the request's filter (a sealed row
	// that does not is exported as a stub).
	Match bool
}

// Source reads the log for one export. Every method reads on the reader pool
// under the export's tenant scope (RLS); the request's filter is the Source's.
type Source interface {
	// FirstSeq is the first sealed seq whose occurred_at is in the window,
	// whatever the other filters say (R20). ok is false when none is.
	FirstSeq(ctx context.Context) (seq int64, ok bool, err error)
	// Anchor returns the hashes of the sealed row at seq (seq >= 1).
	Anchor(ctx context.Context, seq int64) (prev, entry, envelopeSHA [32]byte, err error)
	// Page returns the sealed rows with from <= seq <= to in seq order, at
	// most limit of them. It is one short transaction.
	Page(ctx context.Context, from, to int64, limit int) ([]Row, error)
	// Unsealed returns the matching unsealed rows with occurred_at at or
	// before the export start, in a stable order.
	Unsealed(ctx context.Context) ([]Row, error)
	// EraseEntry returns the audit/erase row with subject_id = ref, sealed or
	// not and whatever its occurred_at (R24), or nil when there is none.
	EraseEntry(ctx context.Context, ref uuid.UUID) (*Row, error)
}

// Filter is the request as the header echoes it. Every member is written,
// null when unset.
type Filter struct {
	From, To    *time.Time
	Domain      *string
	SubjectType *string
	SubjectID   *uuid.UUID
	ChangeSetID *uuid.UUID
	AfterSeq    *int64
}

// Params describe one export.
type Params struct {
	TenantID   uuid.UUID
	ExportID   uuid.UUID
	ExportedAt time.Time
	// Stubs is the request: write stubs for non-matching range entries.
	Stubs  bool
	Filter Filter
	// HeadSeq and HeadHash are the chain head read once, at the start, in
	// the lease transaction.
	HeadSeq  int64
	HeadHash [32]byte
	PageSize int
}

// Write streams the export to w.
func Write(ctx context.Context, w io.Writer, src Source, p Params) error {
	if p.PageSize <= 0 {
		p.PageSize = DefaultPageSize
	}
	e := &exporter{w: w, src: src, p: p, forced: map[int64]*Row{}, written: map[uuid.UUID]bool{}, cur: p.HeadSeq, seenRefs: map[uuid.UUID]bool{}}
	return e.run(ctx)
}

type exporter struct {
	w   io.Writer
	src Source
	p   Params

	anchorPresent bool
	anchorSeq     int64
	count, stubs  int64
	unsealed      int64

	// forced holds the sealed erase entries inside the range, to write in
	// full when the stream reaches them, whatever the filter says (R24).
	forced   map[int64]*Row
	seenRefs map[uuid.UUID]bool
	tail     []*Row // erase entries outside the verified range (R24)
	cur      int64  // seq of the range row being written; head_seq in the tail
	written  map[uuid.UUID]bool
}

func (e *exporter) run(ctx context.Context) error {
	if err := e.header(); err != nil {
		return err
	}
	first, ok, err := e.src.FirstSeq(ctx)
	if err != nil {
		return fmt.Errorf("audit/export: first seq: %w", err)
	}
	if ok {
		if err := e.anchor(ctx, first); err != nil {
			return err
		}
		if err := e.rangeLines(ctx, first); err != nil {
			return err
		}
	}
	if err := e.tailLines(ctx); err != nil {
		return err
	}
	return e.trailer(ok)
}

func (e *exporter) header() error {
	f := e.p.Filter
	return e.line(map[string]any{
		"kind": "header", "format": Format, "tenant_id": e.p.TenantID, "export_id": e.p.ExportID,
		"exported_at": ts(e.p.ExportedAt), "stubs": e.p.Stubs,
		"filter": map[string]any{
			"from": tsp(f.From), "to": tsp(f.To), "domain": f.Domain, "subject_type": f.SubjectType,
			"subject_id": f.SubjectID, "change_set_id": f.ChangeSetID, "after_seq": f.AfterSeq,
		},
	})
}

func (e *exporter) anchor(ctx context.Context, first int64) error {
	e.anchorPresent = true
	if first <= 1 {
		return e.line(map[string]any{"kind": "anchor", "genesis": true, "seq": 0, "prev_hash": nil,
			"entry_hash": chain.Hex(chain.Genesis[:]), "envelope_sha256": nil})
	}
	e.anchorSeq = first - 1
	prev, entry, esha, err := e.src.Anchor(ctx, first-1)
	if err != nil {
		return fmt.Errorf("audit/export: anchor: %w", err)
	}
	return e.line(map[string]any{"kind": "anchor", "genesis": false, "seq": first - 1,
		"prev_hash": chain.Hex(prev[:]), "entry_hash": chain.Hex(entry[:]), "envelope_sha256": chain.Hex(esha[:])})
}

func (e *exporter) rangeLines(ctx context.Context, first int64) error {
	for from := first; from <= e.p.HeadSeq; {
		rows, err := e.src.Page(ctx, from, e.p.HeadSeq, e.p.PageSize)
		if err != nil {
			return fmt.Errorf("audit/export: page from seq %d: %w", from, err)
		}
		if len(rows) == 0 {
			break
		}
		for i := range rows {
			if err := e.rangeRow(ctx, &rows[i]); err != nil {
				return err
			}
		}
		from = rows[len(rows)-1].Seq + 1
	}
	return nil
}

func (e *exporter) rangeRow(ctx context.Context, r *Row) error {
	e.cur = r.Seq
	if f := e.forced[r.Seq]; f != nil {
		r = f
		r.Match = true
	}
	esha, err := chain.EnvelopeSHA256(r.Envelope)
	if err != nil {
		return fmt.Errorf("audit/export: seq %d: %w", r.Seq, err)
	}
	if !r.Match {
		if !e.p.Stubs {
			return nil
		}
		e.stubs++
		return e.line(map[string]any{"kind": "stub", "seq": r.Seq, "prev_hash": chain.Hex(r.PrevHash[:]),
			"entry_hash": chain.Hex(r.EntryHash[:]), "envelope_sha256": chain.Hex(esha[:])})
	}
	e.count++
	if err := e.line(entryLine(r, esha)); err != nil {
		return err
	}
	return e.erased(ctx, r)
}

func (e *exporter) tailLines(ctx context.Context) error {
	rows, err := e.src.Unsealed(ctx)
	if err != nil {
		return fmt.Errorf("audit/export: unsealed: %w", err)
	}
	e.cur = e.p.HeadSeq
	for i := range rows {
		if err := e.unsealedRow(ctx, &rows[i]); err != nil {
			return err
		}
	}
	// The erase entries outside the range come last. One already written as a
	// matching unsealed entry is not repeated.
	for _, r := range e.tail {
		if e.written[r.Envelope.ID] {
			continue
		}
		if err := e.tailRow(r); err != nil {
			return err
		}
	}
	return nil
}

func (e *exporter) unsealedRow(ctx context.Context, r *Row) error {
	if err := e.tailRow(r); err != nil {
		return err
	}
	return e.erased(ctx, r)
}

func (e *exporter) tailRow(r *Row) error {
	esha, err := chain.EnvelopeSHA256(r.Envelope)
	if err != nil {
		return fmt.Errorf("audit/export: unsealed %s: %w", r.Envelope.ID, err)
	}
	e.unsealed++
	e.written[r.Envelope.ID] = true
	// An entry in the tail is never in the verified range, sealed or not.
	t := *r
	t.Sealed = false
	return e.line(entryLine(&t, esha))
}

// erased is R24, as amended by ADR-004-E8 "Follow-up": for an erased entry it
// writes, fetch the audit/erase entry with subject_id = erasure_ref, on the
// Source's reader pool, whatever its occurred_at and whether or not the filter
// selects it. Each erasure_ref is fetched once.
//
//   - Sealed with seq <= head_seq (the verified range): it is written in full
//     at its place in the range, as every erase entry in range must be.
//   - Unsealed, or sealed beyond head_seq (the erasure committed after the
//     head was read): it goes in the tail as sealed:false,
//     in_verified_range:false. The verifier then reports pending.
//
// An erase entry that is not found is not invented; the verifier reports
// erasure_unlisted.
func (e *exporter) erased(ctx context.Context, r *Row) error {
	if r.ErasureRef == nil || e.seenRefs[*r.ErasureRef] {
		return nil
	}
	ref := *r.ErasureRef
	e.seenRefs[ref] = true
	er, err := e.src.EraseEntry(ctx, ref)
	if err != nil {
		return fmt.Errorf("audit/export: erase entry %s: %w", ref, err)
	}
	if er == nil {
		return nil
	}
	if er.Sealed && er.Seq <= e.p.HeadSeq {
		if er.Seq > e.cur { // still ahead of the stream; behind it, it was already written
			e.forced[er.Seq] = er
		}
		return nil
	}
	e.tail = append(e.tail, er)
	return nil
}

func (e *exporter) trailer(haveRange bool) error {
	t := map[string]any{"kind": "trailer", "anchor_seq": nil, "last_seq": nil, "head_hash": nil,
		"count": e.count, "stub_count": e.stubs, "unsealed_count": e.unsealed, "verifiable": false}
	if haveRange {
		t["anchor_seq"], t["last_seq"], t["head_hash"] = e.anchorSeq, e.p.HeadSeq, chain.Hex(e.p.HeadHash[:])
		t["verifiable"] = chain.TrailerVerifiable(e.anchorPresent, e.count, e.stubs, e.p.HeadSeq, e.anchorSeq)
	}
	return e.line(t)
}

func entryLine(r *Row, esha [32]byte) map[string]any {
	l := map[string]any{
		"kind": "entry", "sealed": r.Sealed, "in_verified_range": r.Sealed, "seq": nil, "prev_hash": nil,
		"entry_hash": nil, "envelope_sha256": chain.Hex(esha[:]), "sealed_at": nil, "envelope": r.Envelope,
		"diff": r.Diff, "diff_salt": nil, "diff_redacted": r.DiffRedacted,
		"erased_at": tsp(r.ErasedAt), "erasure_ref": r.ErasureRef,
	}
	if r.Sealed {
		l["seq"], l["prev_hash"], l["entry_hash"] = r.Seq, chain.Hex(r.PrevHash[:]), chain.Hex(r.EntryHash[:])
		l["sealed_at"] = tsp(r.SealedAt)
	}
	if r.DiffSalt != nil && !r.DiffRedacted {
		l["diff_salt"] = chain.Hex(r.DiffSalt[:])
	}
	return l
}

func (e *exporter) line(v any) error {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return fmt.Errorf("audit/export: encode: %w", err)
	}
	if _, err := e.w.Write(buf.Bytes()); err != nil { // Encode ends the line with "\n"
		return fmt.Errorf("audit/export: write: %w", err)
	}
	return nil
}

func ts(t time.Time) string { return t.UTC().Format("2006-01-02T15:04:05.000000Z") }

func tsp(t *time.Time) any {
	if t == nil {
		return nil
	}
	return ts(*t)
}

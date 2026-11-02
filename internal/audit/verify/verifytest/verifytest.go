// Package verifytest builds nava.audit.export/v1 files for the verifier's
// tests from the P5-02 golden-vector generator (internal/audit/chain/vectors).
// Every hash in a fixture is either the generator's or recomputed with
// internal/audit/chain; none is written by hand (TICK-425 acceptance).
//
// Only test code imports this package; cmd/nava does not link it.
package verifytest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"sync"

	"github.com/Navaladi2019/TicketingSystem/internal/audit/chain"
	"github.com/Navaladi2019/TicketingSystem/internal/audit/chain/vectors"
)

// Line is one export line. Member order in the output is sorted, which the
// format allows (§3: member order is free).
type Line = map[string]any

// File is an export under construction.
type File struct {
	Header  Line
	Body    []Line // anchor, range lines, then unsealed lines
	Trailer Line
}

const (
	exportedAt = "2026-11-02T12:30:00.000000Z"
	sealedAt   = "2026-11-02T12:00:00.000000Z"
	exportID   = "0190f3a0-0000-7000-8000-0000000000ee"
)

//nolint:gochecknoglobals // the generated vectors, built once per test binary
var (
	once sync.Once
	vf   vectors.File
	verr error
)

// Vectors returns the generator's vectors file, parsed.
func Vectors() vectors.File {
	once.Do(func() {
		var b []byte
		b, verr = vectors.Generate()
		if verr == nil {
			verr = json.Unmarshal(b, &vf)
		}
	})
	if verr != nil {
		panic(fmt.Sprintf("verifytest: vectors: %v", verr))
	}
	return vf
}

// Chain returns the chain vectors, seq 1..n of one tenant.
func Chain() []vectors.ChainCase { return slices.Clone(Vectors().Chain) }

// Options shape an export.
type Options struct {
	// AfterSeq starts the range at AfterSeq+1 with a stub anchor at AfterSeq
	// (the header's after_seq filter). Zero is from genesis.
	AfterSeq int64
	// HeadSeq ends the range there (the head read at export start). Zero is
	// the last chain vector.
	HeadSeq int64
	// Match picks the entries exported in full; the rest are stubs. Nil
	// matches everything. As the exporter must (R24), an audit/erase entry
	// that lists a matched erased entry is always exported in full.
	Match func(vectors.ChainCase) bool
	// SubjectType is written into the header filter, for documentation.
	SubjectType *string
	// Unsealed are chain vectors appended as unsealed lines, with seq and the
	// seal columns null.
	Unsealed []vectors.ChainCase
}

// Build returns an export of the chain vectors shaped by o.
func Build(o Options) *File {
	cs := Chain()
	head := o.HeadSeq
	if head == 0 {
		head = cs[len(cs)-1].Seq
	}
	match := o.Match
	if match == nil {
		match = func(vectors.ChainCase) bool { return true }
	}
	inRange := func(c vectors.ChainCase) bool { return c.Seq > o.AfterSeq && c.Seq <= head }
	full := map[int64]bool{}
	for _, c := range cs {
		if inRange(c) && match(c) {
			full[c.Seq] = true
		}
	}
	for _, c := range cs {
		if !full[c.Seq] || c.ErasureRef == nil {
			continue
		}
		for _, e := range cs {
			if inRange(e) && e.Envelope.Domain == "audit" && e.Envelope.Action == "erase" &&
				e.Envelope.SubjectID.String() == *c.ErasureRef {
				full[e.Seq] = true
			}
		}
	}

	var after any
	if o.AfterSeq > 0 {
		after = o.AfterSeq
	}
	var st any
	if o.SubjectType != nil {
		st = *o.SubjectType
	}
	f := &File{
		Header: Line{
			"kind": "header", "format": "nava.audit.export/v1", "tenant_id": cs[0].Envelope.TenantID.String(),
			"export_id": exportID, "exported_at": exportedAt, "stubs": true,
			"filter": Line{
				"from": nil, "to": nil, "domain": nil, "subject_type": st, "subject_id": nil,
				"change_set_id": nil, "after_seq": after,
			},
		},
	}
	if o.AfterSeq == 0 {
		f.Body = append(f.Body, Line{
			"kind": "anchor", "genesis": true, "seq": 0, "prev_hash": nil,
			"entry_hash": chain.Hex(chain.Genesis[:]), "envelope_sha256": nil,
		})
	} else {
		a := byseq(cs, o.AfterSeq)
		f.Body = append(f.Body, Line{
			"kind": "anchor", "genesis": false, "seq": a.Seq, "prev_hash": a.PrevHash,
			"entry_hash": a.EntryHash, "envelope_sha256": a.EnvelopeSHA256,
		})
	}
	for _, c := range cs {
		if !inRange(c) {
			continue
		}
		if full[c.Seq] {
			f.Body = append(f.Body, Entry(c))
		} else {
			f.Body = append(f.Body, Stub(c))
		}
	}
	for _, c := range o.Unsealed {
		f.Body = append(f.Body, Unsealed(c))
	}
	last := byseq(cs, head)
	f.Trailer = Line{
		"kind": "trailer", "anchor_seq": o.AfterSeq, "last_seq": head, "head_hash": last.EntryHash,
		"verifiable": true,
	}
	f.Recount()
	return f
}

func byseq(cs []vectors.ChainCase, seq int64) vectors.ChainCase {
	for _, c := range cs {
		if c.Seq == seq {
			return c
		}
	}
	panic(fmt.Sprintf("verifytest: no chain vector seq %d", seq))
}

// Entry is the full sealed line of a chain vector.
func Entry(c vectors.ChainCase) Line {
	return Line{
		"kind": "entry", "sealed": true, "in_verified_range": true, "seq": c.Seq,
		"prev_hash": c.PrevHash, "entry_hash": c.EntryHash, "envelope_sha256": c.EnvelopeSHA256,
		"sealed_at": sealedAt, "envelope": c.Envelope, "diff": c.Diff, "diff_salt": c.DiffSalt,
		"diff_redacted": false, "erased_at": c.ErasedAt, "erasure_ref": c.ErasureRef,
	}
}

// Stub is the stub line of a chain vector.
func Stub(c vectors.ChainCase) Line {
	return Line{"kind": "stub", "seq": c.Seq, "prev_hash": c.PrevHash, "entry_hash": c.EntryHash, "envelope_sha256": c.EnvelopeSHA256}
}

// Unsealed is a chain vector exported before it was sealed.
func Unsealed(c vectors.ChainCase) Line {
	return Line{
		"kind": "entry", "sealed": false, "in_verified_range": false, "seq": nil,
		"prev_hash": nil, "entry_hash": nil, "envelope_sha256": c.EnvelopeSHA256,
		"sealed_at": nil, "envelope": c.Envelope, "diff": c.Diff, "diff_salt": c.DiffSalt,
		"diff_redacted": false, "erased_at": c.ErasedAt, "erasure_ref": c.ErasureRef,
	}
}

// Tampered is the full sealed line of a tampered vector.
func Tampered(t vectors.Tampered) Line {
	c := byseq(Chain(), t.Seq)
	l := Entry(c)
	l["prev_hash"], l["envelope"], l["envelope_sha256"], l["entry_hash"] = t.PrevHash, t.Envelope, t.EnvelopeSHA256, t.EntryHash
	l["diff"], l["diff_salt"] = t.Diff, t.DiffSalt
	return l
}

// Index returns the body index of the range line for seq, or -1.
func (f *File) Index(seq int64) int {
	for i, l := range f.Body {
		if l["kind"] != "anchor" && l["seq"] == seq {
			return i
		}
	}
	return -1
}

// Recount sets the trailer counts from the body.
func (f *File) Recount() {
	count, stubs, unsealed := 0, 0, 0
	for _, l := range f.Body {
		switch {
		case l["kind"] == "stub":
			stubs++
		case l["kind"] == "entry" && l["sealed"] == true:
			count++
		case l["kind"] == "entry":
			unsealed++
		}
	}
	f.Trailer["count"], f.Trailer["stub_count"], f.Trailer["unsealed_count"] = count, stubs, unsealed
}

// Rehash recomputes body line i's envelope_sha256 (for a full line) and
// entry_hash from its own members, the way a forger rewriting one row would.
func (f *File) Rehash(i int) {
	l := f.Body[i]
	esha := l["envelope_sha256"].(string) //nolint:forcetypeassert // fixture built by this package
	if env, ok := l["envelope"].(chain.Envelope); ok {
		h, err := chain.EnvelopeSHA256(env)
		if err != nil {
			panic(err)
		}
		esha = chain.Hex(h[:])
		l["envelope_sha256"] = esha
	}
	prev, err := chain.ParseHash32(l["prev_hash"].(string)) //nolint:forcetypeassert // fixture built by this package
	if err != nil {
		panic(err)
	}
	e, err := chain.ParseHash32(esha)
	if err != nil {
		panic(err)
	}
	eh := chain.EntryHash(prev, e)
	l["entry_hash"] = chain.Hex(eh[:])
}

// Bytes serialises the file: one line each, "\n" line ends.
func (f *File) Bytes() []byte {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	for _, l := range append(append([]Line{f.Header}, f.Body...), f.Trailer) {
		if err := enc.Encode(l); err != nil {
			panic(err)
		}
	}
	return b.Bytes()
}

// JSONL joins an export vector's lines into its file bytes.
func JSONL(lines []string) []byte {
	var b bytes.Buffer
	for _, l := range lines {
		b.WriteString(l)
		b.WriteByte('\n')
	}
	return b.Bytes()
}

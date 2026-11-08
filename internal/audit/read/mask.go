package read

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/audit/patchwalk"
)

// Subject is what a masker is asked about: the kind of entry and, for a
// record or a configuration entry, the entity definition whose schema marks
// its secrets.
type Subject struct {
	Domain             string
	SubjectType        string
	EntityDefinitionID *uuid.UUID
}

// Masker is the port that says which pointers of an entry's diff must not
// leave the log for this reader (ADR-004 R7, PLAN-005 D-5-5). The pointers
// are in the diff's own coordinates: for a record entry, "/<property>" of the
// record payload. S10's per-field masks come through the same port
// (FieldMasks, P3-33). A masker
// runs in the caller's tenant transaction.
type Masker interface {
	MaskedPaths(ctx context.Context, tx pgx.Tx, s Subject) ([]patchwalk.Pointer, error)
}

// StaticMasker is the fixture masker: the same pointers for every subject.
type StaticMasker []patchwalk.Pointer

// MaskedPaths returns the fixed pointers.
func (m StaticMasker) MaskedPaths(context.Context, pgx.Tx, Subject) ([]patchwalk.Pointer, error) {
	return slices.Clone([]patchwalk.Pointer(m)), nil
}

// The two replacements R7 writes. A masked value becomes MaskedValue; an op
// that cannot be cut cleanly becomes RedactedOp.
const (
	MaskedValue = `{"redacted":"mask"}`
	RedactedOp  = `{"redacted":"op"}`
	// redactedDiff replaces a diff that is not a JSON Patch the walker can
	// read, when there is something to mask: fail closed.
	redactedDiff = `{"redacted":"diff"}`
)

// MaskDiff applies masked to diff (R7) and reports whether it changed
// anything. A masked pointer reached through an op's path, or inside a nested
// value, becomes {"redacted":"mask"} in place of its value (the path stays: it
// names the field, not what it held). A copy or move whose from overlaps a
// masked pointer is redacted whole, as {"redacted":"op"}, because its
// destination would hold the value. A diff that cannot be parsed is redacted
// whole. With nothing to mask, or nothing masked in the diff, the stored bytes
// come back unchanged, so the caller can return the salt with them.
func MaskDiff(diff []byte, masked []patchwalk.Pointer) (out []byte, redacted bool, err error) {
	if len(diff) == 0 || len(masked) == 0 {
		return diff, false, nil
	}
	if !touches(diff, masked) {
		return diff, false, nil
	}
	var ops []rawOp
	if err := patchwalk.Walk(diff, func(patchwalk.Op, patchwalk.Pointer, json.RawMessage) error { return nil }); err != nil {
		return []byte(redactedDiff), true, nil
	}
	if err := json.Unmarshal(diff, &ops); err != nil {
		return []byte(redactedDiff), true, nil
	}
	var buf bytes.Buffer
	changed := false
	buf.WriteByte('[')
	for i, o := range ops {
		if i > 0 {
			buf.WriteByte(',')
		}
		b, c, err := maskOp(o, masked)
		if err != nil {
			b, c = []byte(RedactedOp), true
		}
		changed = changed || c
		buf.Write(b)
	}
	buf.WriteByte(']')
	if !changed {
		return diff, false, nil
	}
	return buf.Bytes(), true, nil
}

// touches reports whether any path, from or nested member of diff is, is
// under, or is a parent of a masked pointer. A malformed diff counts as
// touched, so it is redacted whole.
func touches(diff []byte, masked []patchwalk.Pointer) bool {
	hit := false
	err := patchwalk.Walk(diff, func(op patchwalk.Op, ptr patchwalk.Pointer, _ json.RawMessage) error {
		if op.Field == patchwalk.FieldValue && op.Kind == patchwalk.KindRemove {
			return nil
		}
		if overlaps(ptr, masked) {
			hit = true
		}
		return nil
	})
	return err != nil || hit
}

// overlaps reports whether p equals, is under, or is a parent of any masked
// pointer.
func overlaps(p patchwalk.Pointer, masked []patchwalk.Pointer) bool {
	for _, m := range masked {
		if p.HasPrefix(m) || m.HasPrefix(p) {
			return true
		}
	}
	return false
}

func under(p patchwalk.Pointer, masked []patchwalk.Pointer) bool {
	for _, m := range masked {
		if p.HasPrefix(m) {
			return true
		}
	}
	return false
}

// rawOp is one op in its output order. From is a pointer so an absent member
// stays absent.
type rawOp struct {
	Op    string          `json:"op"`
	Path  string          `json:"path"`
	From  *string         `json:"from,omitempty"`
	Value json.RawMessage `json:"value,omitempty"`
}

// maskOp returns the op as it leaves the log and whether it differs from the
// stored one.
func maskOp(o rawOp, masked []patchwalk.Pointer) ([]byte, bool, error) {
	path := patchwalk.Pointer(o.Path)
	if o.From != nil && overlaps(patchwalk.Pointer(*o.From), masked) {
		return []byte(RedactedOp), true, nil
	}
	changed := false
	if o.Value != nil {
		if under(path, masked) {
			o.Value, changed = json.RawMessage(MaskedValue), true
		} else {
			v, c, err := cutNested(o.Value, path, masked)
			if err != nil {
				return nil, false, err
			}
			o.Value, changed = v, c
		}
	}
	b, err := json.Marshal(o)
	return b, changed, err
}

// cutNested replaces, inside value (the op's value at path), every member a
// masked pointer below path names. It errors when value cannot be decoded
// and re-encoded exactly, so the caller redacts the whole op.
func cutNested(value json.RawMessage, path patchwalk.Pointer, masked []patchwalk.Pointer) (json.RawMessage, bool, error) {
	var rels [][]string
	for _, m := range masked {
		if m != path && m.HasPrefix(path) {
			rels = append(rels, relPointer(m[len(path):]).segments())
		}
	}
	if len(rels) == 0 {
		return value, false, nil
	}
	dec := json.NewDecoder(bytes.NewReader(value))
	dec.UseNumber()
	var doc any
	if err := dec.Decode(&doc); err != nil {
		return nil, false, fmt.Errorf("decode value: %w", err)
	}
	changed := false
	for _, segs := range rels {
		var c bool
		doc, c = setMasked(doc, segs)
		changed = changed || c
	}
	if !changed {
		return value, false, nil
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(doc); err != nil {
		return nil, false, fmt.Errorf("encode value: %w", err)
	}
	return json.RawMessage(bytes.TrimSuffix(buf.Bytes(), []byte("\n"))), true, nil
}

type relPointer string

func (r relPointer) segments() []string { return patchwalk.Pointer(r).Segments() }

// setMasked walks doc along segs and replaces the member it reaches with the
// mask object. A segment that is not there leaves doc as it is.
func setMasked(doc any, segs []string) (any, bool) {
	if len(segs) == 0 {
		return json.RawMessage(MaskedValue), true
	}
	changed := false
	switch d := doc.(type) {
	case map[string]any:
		if child, ok := d[segs[0]]; ok {
			d[segs[0]], changed = setMasked(child, segs[1:])
		}
	case []any:
		if i, err := strconv.Atoi(segs[0]); err == nil && i >= 0 && i < len(d) && strconv.Itoa(i) == segs[0] {
			d[i], changed = setMasked(d[i], segs[1:])
		}
	}
	return doc, changed
}

// The domain and subject type of a record entry, the only entries whose diff
// is a record payload and so the only ones field masks apply to.
const (
	domainRecord  = "record"
	subjectRecord = "record"
)

// FieldMasks adds the record ACL's field masks (PLAN-003 S10, P3-33) to a
// Masker, for one reader: Fields maps an entity definition id to the
// top-level properties its current policy hides from that reader
// (authz.ACL.ReadMasks, read by the caller on nava_app, where acl_versions is
// readable). For a record entry of one of those definitions it returns
// Base's pointers plus "/<field>" for each field; every other entry gets
// Base's pointers only. A masked field is masked exactly as a secret is
// (R7): {"redacted":"mask"}, and no salt with the diff.
type FieldMasks struct {
	Base   Masker
	Fields map[uuid.UUID][]string
}

// MaskedPaths implements Masker.
func (m FieldMasks) MaskedPaths(ctx context.Context, tx pgx.Tx, s Subject) ([]patchwalk.Pointer, error) {
	paths, err := m.Base.MaskedPaths(ctx, tx, s)
	if err != nil {
		return nil, err
	}
	if s.Domain != domainRecord || s.SubjectType != subjectRecord || s.EntityDefinitionID == nil {
		return paths, nil
	}
	for _, f := range m.Fields[*s.EntityDefinitionID] {
		p := patchwalk.FromSegments([]string{f})
		if !slices.Contains(paths, p) {
			paths = append(paths, p)
		}
	}
	return paths, nil
}

// WithFieldMasks returns a copy of r whose masker also applies fields
// (FieldMasks). r itself is unchanged, so one Reader serves every caller.
func (r *Reader) WithFieldMasks(fields map[uuid.UUID][]string) *Reader {
	c := *r
	c.Masker = FieldMasks{Base: r.Masker, Fields: fields}
	return &c
}

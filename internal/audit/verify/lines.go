package verify

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"

	"github.com/Navaladi2019/TicketingSystem/internal/audit/chain"
)

// The line shapes of audit-export-v1.md §3. Every member listed in a shape
// must be present exactly once (step 10); nullable lists the members that
// may be JSON null.

type headerLine struct {
	Kind       string      `json:"kind"`
	Format     string      `json:"format"`
	TenantID   string      `json:"tenant_id"`
	ExportID   string      `json:"export_id"`
	ExportedAt string      `json:"exported_at"`
	Stubs      bool        `json:"stubs"`
	Filter     filterBlock `json:"filter"`
}

type filterBlock struct {
	From        *string `json:"from"`
	To          *string `json:"to"`
	Domain      *string `json:"domain"`
	SubjectType *string `json:"subject_type"`
	SubjectID   *string `json:"subject_id"`
	ChangeSetID *string `json:"change_set_id"`
	AfterSeq    *int64  `json:"after_seq"`
}

type anchorLine struct {
	Kind           string  `json:"kind"`
	Genesis        bool    `json:"genesis"`
	Seq            int64   `json:"seq"`
	PrevHash       *string `json:"prev_hash"`
	EntryHash      string  `json:"entry_hash"`
	EnvelopeSHA256 *string `json:"envelope_sha256"`
}

type entryLine struct {
	Kind            string         `json:"kind"`
	Sealed          bool           `json:"sealed"`
	InVerifiedRange bool           `json:"in_verified_range"`
	Seq             *int64         `json:"seq"`
	PrevHash        *string        `json:"prev_hash"`
	EntryHash       *string        `json:"entry_hash"`
	EnvelopeSHA256  string         `json:"envelope_sha256"`
	SealedAt        *string        `json:"sealed_at"`
	Envelope        chain.Envelope `json:"envelope"`
	Diff            *string        `json:"diff"`
	DiffSalt        *string        `json:"diff_salt"`
	DiffRedacted    bool           `json:"diff_redacted"`
	ErasedAt        *string        `json:"erased_at"`
	ErasureRef      *string        `json:"erasure_ref"`
}

type stubLine struct {
	Kind           string `json:"kind"`
	Seq            int64  `json:"seq"`
	PrevHash       string `json:"prev_hash"`
	EntryHash      string `json:"entry_hash"`
	EnvelopeSHA256 string `json:"envelope_sha256"`
}

type trailerLine struct {
	Kind          string  `json:"kind"`
	AnchorSeq     *int64  `json:"anchor_seq"`
	LastSeq       *int64  `json:"last_seq"`
	HeadHash      *string `json:"head_hash"`
	Count         int     `json:"count"`
	StubCount     int     `json:"stub_count"`
	UnsealedCount int     `json:"unsealed_count"`
	Verifiable    bool    `json:"verifiable"`
}

type shape struct {
	members  []string
	nullable []string
}

//nolint:gochecknoglobals // read-only line-shape table (audit-export-v1.md §3)
var shapes = map[string]shape{
	"header": {members: []string{"kind", "format", "tenant_id", "export_id", "exported_at", "stubs", "filter"}},
	"filter": {
		members:  []string{"from", "to", "domain", "subject_type", "subject_id", "change_set_id", "after_seq"},
		nullable: []string{"from", "to", "domain", "subject_type", "subject_id", "change_set_id", "after_seq"},
	},
	"anchor": {
		members:  []string{"kind", "genesis", "seq", "prev_hash", "entry_hash", "envelope_sha256"},
		nullable: []string{"prev_hash", "envelope_sha256"},
	},
	"entry": {
		members: []string{
			"kind", "sealed", "in_verified_range", "seq", "prev_hash", "entry_hash", "envelope_sha256",
			"sealed_at", "envelope", "diff", "diff_salt", "diff_redacted", "erased_at", "erasure_ref",
		},
		nullable: []string{"seq", "prev_hash", "entry_hash", "sealed_at", "diff", "diff_salt", "erased_at", "erasure_ref"},
	},
	"stub": {members: []string{"kind", "seq", "prev_hash", "entry_hash", "envelope_sha256"}},
	"trailer": {
		members:  []string{"kind", "anchor_seq", "last_seq", "head_hash", "count", "stub_count", "unsealed_count", "verifiable"},
		nullable: []string{"anchor_seq", "last_seq", "head_hash"},
	},
}

var errUnknownKind = errors.New("unknown kind")

// members splits one JSON object into its members, rejecting duplicates and
// trailing data.
func members(b []byte) (map[string]json.RawMessage, error) {
	dec := json.NewDecoder(bytes.NewReader(b))
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	if d, ok := tok.(json.Delim); !ok || d != '{' {
		return nil, errors.New("not a JSON object")
	}
	out := map[string]json.RawMessage{}
	for dec.More() {
		kt, err := dec.Token()
		if err != nil {
			return nil, err
		}
		key, _ := kt.(string)
		if _, dup := out[key]; dup {
			return nil, fmt.Errorf("duplicate member %q", key)
		}
		var v json.RawMessage
		if err := dec.Decode(&v); err != nil {
			return nil, fmt.Errorf("member %q: %w", key, err)
		}
		out[key] = v
	}
	if _, err := dec.Token(); err != nil {
		return nil, err
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return nil, errors.New("trailing data after the object")
	}
	return out, nil
}

// checkShape requires exactly the shape's members, and null only where the
// shape allows it.
func checkShape(m map[string]json.RawMessage, s shape) error {
	for _, name := range s.members {
		v, ok := m[name]
		if !ok {
			return fmt.Errorf("missing member %q", name)
		}
		if bytes.Equal(v, []byte("null")) && !slices.Contains(s.nullable, name) {
			return fmt.Errorf("member %q must not be null", name)
		}
	}
	for name := range m {
		if !slices.Contains(s.members, name) {
			return fmt.Errorf("unknown member %q", name)
		}
	}
	return nil
}

// lineKind returns the kind of one line after checking it is a strict object.
func lineKind(b []byte) (string, map[string]json.RawMessage, error) {
	m, err := members(b)
	if err != nil {
		return "", nil, err
	}
	raw, ok := m["kind"]
	if !ok {
		return "", nil, errors.New(`missing member "kind"`)
	}
	var kind string
	if err := json.Unmarshal(raw, &kind); err != nil {
		return "", nil, errors.New(`member "kind" is not a string`)
	}
	s, ok := shapes[kind]
	if !ok || kind == "filter" {
		return kind, nil, errUnknownKind
	}
	if err := checkShape(m, s); err != nil {
		return kind, nil, err
	}
	if kind == "header" {
		f, err := members(m["filter"])
		if err != nil {
			return kind, nil, fmt.Errorf("filter: %w", err)
		}
		if err := checkShape(f, shapes["filter"]); err != nil {
			return kind, nil, fmt.Errorf("filter: %w", err)
		}
	}
	return kind, m, nil
}

// decodeAs decodes a line already checked by lineKind into its struct.
func decodeAs(b []byte, v any) error {
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	return d.Decode(v)
}

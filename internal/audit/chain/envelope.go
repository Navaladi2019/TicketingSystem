package chain

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

// ErrInvalidEnvelope is returned when an envelope cannot be canonicalised or
// parsed. The wrapped message names the field, never a value.
var ErrInvalidEnvelope = errors.New("chain: invalid envelope")

// TimeLayout is the canonical timestamp form: RFC 3339, UTC, exactly six
// fractional digits (PostgreSQL timestamptz precision).
const TimeLayout = "2006-01-02T15:04:05.000000Z"

// MaxSafeInteger bounds every integer in the envelope (I-JSON, RFC 7493 §2.2),
// so any JSON implementation reads it back exactly.
const MaxSafeInteger = 1<<53 - 1

// Envelope is exactly the hashed envelope of one audit_log entry: the ADR-004
// R1 envelope columns plus diff_truncated (ADR-004-E1). The JSON names are the
// column names. diff, diff_salt, txid, and the seal and erasure columns are
// not part of it.
//
// A pointer field is a nullable column; nil is NULL, which the canonical form
// emits as null. Text values that are present must be non-empty (an empty
// string and NULL would otherwise share one CSV cell; see audit-export-v1.md).
type Envelope struct {
	TenantID           uuid.UUID
	ID                 uuid.UUID
	OccurredAt         time.Time
	Domain             string
	Action             string
	SubjectType        string
	SubjectID          uuid.UUID
	EntityDefinitionID *uuid.UUID
	VersionBefore      *int64
	VersionAfter       *int64
	ActorPrincipalID   *uuid.UUID
	ActorSystem        *string
	Source             string
	SourceRef          *string
	ChangeSetID        *uuid.UUID
	CausationID        *uuid.UUID
	CausationDepth     int16
	RequestID          *string
	Backfilled         bool
	ContentSHA256      *[32]byte
	DiffSHA256         *[32]byte
	DiffTruncated      bool
}

// CanonicalEnvelope returns the RFC 8785 (JCS) bytes of e: members sorted by
// UTF-16 code units, no insignificant whitespace, every column present.
func CanonicalEnvelope(e Envelope) ([]byte, error) {
	type member struct {
		key string
		val []byte
	}
	members := make([]member, 0, len(envelopeFields))
	for _, f := range envelopeFields {
		v, err := f.enc(&e)
		if err != nil {
			return nil, fmt.Errorf("%w: %s: %w", ErrInvalidEnvelope, f.name, err)
		}
		members = append(members, member{f.name, v})
	}
	sortByUTF16(members, func(m member) string { return m.key })

	var b bytes.Buffer
	b.WriteByte('{')
	for i, m := range members {
		if i > 0 {
			b.WriteByte(',')
		}
		writeJCSString(&b, m.key)
		b.WriteByte(':')
		b.Write(m.val)
	}
	b.WriteByte('}')
	return b.Bytes(), nil
}

// MarshalJSON emits the canonical form, which is also the envelope object of
// an export line.
func (e Envelope) MarshalJSON() ([]byte, error) { return CanonicalEnvelope(e) }

// UnmarshalJSON parses an envelope object strictly: every column must be
// present exactly once, no other member is allowed, and each value must be in
// its canonical form (lowercase UUID and hex, six-digit UTC timestamp, plain
// integer). Member order and whitespace are free; hash the result with
// EnvelopeSHA256, never the input bytes.
func (e *Envelope) UnmarshalJSON(b []byte) error {
	// encoding/json silently turns invalid UTF-8 into U+FFFD; reject it
	// before decoding so the value hashed is the value in the file (§2).
	if !utf8.Valid(b) {
		return fmt.Errorf("%w: invalid UTF-8", ErrInvalidEnvelope)
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	tok, err := dec.Token()
	if err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidEnvelope, err)
	}
	if d, ok := tok.(json.Delim); !ok || d != '{' {
		return fmt.Errorf("%w: not a JSON object", ErrInvalidEnvelope)
	}
	raw := make(map[string]json.RawMessage, len(envelopeFields))
	for dec.More() {
		kt, err := dec.Token()
		if err != nil {
			return fmt.Errorf("%w: %w", ErrInvalidEnvelope, err)
		}
		key, _ := kt.(string)
		if _, dup := raw[key]; dup {
			return fmt.Errorf("%w: duplicate member %q", ErrInvalidEnvelope, key)
		}
		var v json.RawMessage
		if err := dec.Decode(&v); err != nil {
			return fmt.Errorf("%w: %s: %w", ErrInvalidEnvelope, key, err)
		}
		raw[key] = v
	}
	if _, err := dec.Token(); err != nil { // the closing brace
		return fmt.Errorf("%w: %w", ErrInvalidEnvelope, err)
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return fmt.Errorf("%w: trailing data after the object", ErrInvalidEnvelope)
	}

	var out Envelope
	for _, f := range envelopeFields {
		v, ok := raw[f.name]
		if !ok {
			return fmt.Errorf("%w: missing member %q (NULL is emitted, never omitted)", ErrInvalidEnvelope, f.name)
		}
		delete(raw, f.name)
		if err := f.dec(&out, v); err != nil {
			return fmt.Errorf("%w: %s: %w", ErrInvalidEnvelope, f.name, err)
		}
	}
	for k := range raw {
		return fmt.Errorf("%w: unknown member %q", ErrInvalidEnvelope, k)
	}
	// Round-trip through the encoder so a parsed envelope always canonicalises.
	if _, err := CanonicalEnvelope(out); err != nil {
		return err
	}
	*e = out
	return nil
}

// envelopeField maps one column to its canonical value and back.
type envelopeField struct {
	name string
	enc  func(*Envelope) ([]byte, error)
	dec  func(*Envelope, json.RawMessage) error
}

// envelopeFields lists every envelope column. TestEveryFieldIsHashed fails if
// a struct field has no entry here.
//
//nolint:gochecknoglobals // read-only column table
var envelopeFields = []envelopeField{
	uuidField("tenant_id", func(e *Envelope) *uuid.UUID { return &e.TenantID }),
	uuidField("id", func(e *Envelope) *uuid.UUID { return &e.ID }),
	{
		name: "occurred_at",
		enc:  func(e *Envelope) ([]byte, error) { return encTime(e.OccurredAt) },
		dec: func(e *Envelope, v json.RawMessage) error {
			t, err := decTime(v)
			e.OccurredAt = t
			return err
		},
	},
	textField("domain", func(e *Envelope) *string { return &e.Domain }),
	textField("action", func(e *Envelope) *string { return &e.Action }),
	textField("subject_type", func(e *Envelope) *string { return &e.SubjectType }),
	uuidField("subject_id", func(e *Envelope) *uuid.UUID { return &e.SubjectID }),
	nullUUIDField("entity_definition_id", func(e *Envelope) **uuid.UUID { return &e.EntityDefinitionID }),
	nullIntField("version_before", func(e *Envelope) **int64 { return &e.VersionBefore }),
	nullIntField("version_after", func(e *Envelope) **int64 { return &e.VersionAfter }),
	nullUUIDField("actor_principal_id", func(e *Envelope) **uuid.UUID { return &e.ActorPrincipalID }),
	nullTextField("actor_system", func(e *Envelope) **string { return &e.ActorSystem }),
	textField("source", func(e *Envelope) *string { return &e.Source }),
	nullTextField("source_ref", func(e *Envelope) **string { return &e.SourceRef }),
	nullUUIDField("change_set_id", func(e *Envelope) **uuid.UUID { return &e.ChangeSetID }),
	nullUUIDField("causation_id", func(e *Envelope) **uuid.UUID { return &e.CausationID }),
	{
		name: "causation_depth",
		enc:  func(e *Envelope) ([]byte, error) { return encInt(int64(e.CausationDepth)) },
		dec: func(e *Envelope, v json.RawMessage) error {
			n, err := decInt(v)
			if err != nil {
				return err
			}
			if n < -1<<15 || n > 1<<15-1 {
				return errors.New("out of smallint range")
			}
			e.CausationDepth = int16(n)
			return nil
		},
	},
	nullTextField("request_id", func(e *Envelope) **string { return &e.RequestID }),
	boolField("backfilled", func(e *Envelope) *bool { return &e.Backfilled }),
	nullHashField("content_sha256", func(e *Envelope) **[32]byte { return &e.ContentSHA256 }),
	nullHashField("diff_sha256", func(e *Envelope) **[32]byte { return &e.DiffSHA256 }),
	boolField("diff_truncated", func(e *Envelope) *bool { return &e.DiffTruncated }),
}

var null = []byte("null") //nolint:gochecknoglobals // read-only literal

func isNull(v json.RawMessage) bool { return bytes.Equal(v, null) }

func uuidField(name string, p func(*Envelope) *uuid.UUID) envelopeField {
	return envelopeField{
		name: name,
		enc:  func(e *Envelope) ([]byte, error) { return encUUID(*p(e)), nil },
		dec: func(e *Envelope, v json.RawMessage) error {
			u, err := decUUID(v)
			*p(e) = u
			return err
		},
	}
}

func nullUUIDField(name string, p func(*Envelope) **uuid.UUID) envelopeField {
	return envelopeField{
		name: name,
		enc: func(e *Envelope) ([]byte, error) {
			if *p(e) == nil {
				return null, nil
			}
			return encUUID(**p(e)), nil
		},
		dec: func(e *Envelope, v json.RawMessage) error {
			if isNull(v) {
				*p(e) = nil
				return nil
			}
			u, err := decUUID(v)
			*p(e) = &u
			return err
		},
	}
}

func textField(name string, p func(*Envelope) *string) envelopeField {
	return envelopeField{
		name: name,
		enc:  func(e *Envelope) ([]byte, error) { return encText(*p(e)) },
		dec: func(e *Envelope, v json.RawMessage) error {
			s, err := decText(v)
			*p(e) = s
			return err
		},
	}
}

func nullTextField(name string, p func(*Envelope) **string) envelopeField {
	return envelopeField{
		name: name,
		enc: func(e *Envelope) ([]byte, error) {
			if *p(e) == nil {
				return null, nil
			}
			return encText(**p(e))
		},
		dec: func(e *Envelope, v json.RawMessage) error {
			if isNull(v) {
				*p(e) = nil
				return nil
			}
			s, err := decText(v)
			*p(e) = &s
			return err
		},
	}
}

func nullIntField(name string, p func(*Envelope) **int64) envelopeField {
	return envelopeField{
		name: name,
		enc: func(e *Envelope) ([]byte, error) {
			if *p(e) == nil {
				return null, nil
			}
			return encInt(**p(e))
		},
		dec: func(e *Envelope, v json.RawMessage) error {
			if isNull(v) {
				*p(e) = nil
				return nil
			}
			n, err := decInt(v)
			*p(e) = &n
			return err
		},
	}
}

func boolField(name string, p func(*Envelope) *bool) envelopeField {
	return envelopeField{
		name: name,
		enc: func(e *Envelope) ([]byte, error) {
			return []byte(strconv.FormatBool(*p(e))), nil
		},
		dec: func(e *Envelope, v json.RawMessage) error {
			switch string(v) {
			case "true":
				*p(e) = true
			case "false":
				*p(e) = false
			default:
				return errors.New("want true or false")
			}
			return nil
		},
	}
}

func nullHashField(name string, p func(*Envelope) **[32]byte) envelopeField {
	return envelopeField{
		name: name,
		enc: func(e *Envelope) ([]byte, error) {
			if *p(e) == nil {
				return null, nil
			}
			return []byte(`"` + Hex((**p(e))[:]) + `"`), nil
		},
		dec: func(e *Envelope, v json.RawMessage) error {
			if isNull(v) {
				*p(e) = nil
				return nil
			}
			s, err := decString(v)
			if err != nil {
				return err
			}
			h, err := ParseHash32(s)
			*p(e) = &h
			return err
		},
	}
}

func encUUID(u uuid.UUID) []byte { return []byte(`"` + u.String() + `"`) }

func decUUID(v json.RawMessage) (uuid.UUID, error) {
	s, err := decString(v)
	if err != nil {
		return uuid.Nil, err
	}
	u, err := uuid.Parse(s)
	if err != nil || u.String() != s {
		return uuid.Nil, errors.New("want a lowercase 36-character UUID")
	}
	return u, nil
}

func encTime(t time.Time) ([]byte, error) {
	if t.Nanosecond()%1000 != 0 {
		return nil, errors.New("sub-microsecond precision (timestamptz stores microseconds)")
	}
	u := t.UTC()
	if u.Year() < 1 || u.Year() > 9999 {
		return nil, errors.New("year outside 0001-9999")
	}
	return []byte(`"` + u.Format(TimeLayout) + `"`), nil
}

func decTime(v json.RawMessage) (time.Time, error) {
	s, err := decString(v)
	if err != nil {
		return time.Time{}, err
	}
	t, err := time.Parse(TimeLayout, s)
	if err != nil || t.Format(TimeLayout) != s {
		return time.Time{}, errors.New("want RFC 3339 UTC with six fractional digits, like 2026-11-02T10:00:00.000000Z")
	}
	return t, nil
}

func encText(s string) ([]byte, error) {
	if s == "" {
		return nil, errors.New("empty text (use NULL)")
	}
	if !utf8.ValidString(s) {
		return nil, errors.New("invalid UTF-8")
	}
	if strings.IndexByte(s, 0) >= 0 {
		return nil, errors.New("U+0000 (PostgreSQL text cannot store it)")
	}
	var b bytes.Buffer
	writeJCSString(&b, s)
	return b.Bytes(), nil
}

// decString decodes a JSON string and requires it to be in canonical (JCS)
// form. encoding/json maps a lone surrogate escape or an invalid byte to
// U+FFFD without an error, so the decoded value would differ from the bytes
// in the file; re-encoding and comparing rejects that, and any other
// non-canonical escape such as A (Technical Architect, PR #365).
func decString(v json.RawMessage) (string, error) {
	var s string
	if err := json.Unmarshal(v, &s); err != nil {
		return "", errors.New("want a string")
	}
	var c bytes.Buffer
	writeJCSString(&c, s)
	if !bytes.Equal(c.Bytes(), v) {
		return "", errors.New("string not in canonical form")
	}
	return s, nil
}

func decText(v json.RawMessage) (string, error) {
	s, err := decString(v)
	if err != nil {
		return "", err
	}
	if s == "" {
		return "", errors.New("empty text (use null)")
	}
	return s, nil
}

func encInt(n int64) ([]byte, error) {
	if n > MaxSafeInteger || n < -MaxSafeInteger {
		return nil, errors.New("integer outside ±(2^53-1)")
	}
	return []byte(strconv.FormatInt(n, 10)), nil
}

// decInt accepts only the canonical integer form: no sign on zero, no leading
// zeros, no fraction or exponent.
func decInt(v json.RawMessage) (int64, error) {
	s := string(v)
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil || strconv.FormatInt(n, 10) != s {
		return 0, errors.New("want a plain integer")
	}
	if n > MaxSafeInteger || n < -MaxSafeInteger {
		return 0, errors.New("integer outside ±(2^53-1)")
	}
	return n, nil
}

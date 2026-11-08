package jsondiff

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

// ErrInvalidDocument is returned when before or after is not exactly one
// valid UTF-8 JSON value.
var ErrInvalidDocument = errors.New("jsondiff: invalid JSON document")

// ErrNoAfter is returned when after is nil or empty. An audit diff always
// describes a resulting document; a create passes a nil before instead.
var ErrNoAfter = errors.New("jsondiff: after document is required")

// Diff returns the RFC 6902 JSON Patch that turns before into after, as
// compact JSON bytes that are identical for identical inputs (see the package
// comment for the exact form). A nil before gives one add at path "".
// Equal documents give [].
func Diff(before, after json.RawMessage) (json.RawMessage, error) {
	if len(after) == 0 {
		return nil, ErrNoAfter
	}
	b, err := decode(after)
	if err != nil {
		return nil, fmt.Errorf("after: %w", err)
	}
	w := &writer{}
	w.buf.WriteByte('[')
	if len(before) == 0 {
		w.op("add", "", b, true)
	} else {
		a, err := decode(before)
		if err != nil {
			return nil, fmt.Errorf("before: %w", err)
		}
		w.diff("", a, b)
	}
	w.buf.WriteByte(']')
	return json.RawMessage(w.buf.Bytes()), nil
}

// decode parses doc as one JSON value, keeping numbers as their literals.
func decode(doc json.RawMessage) (any, error) {
	if !utf8.Valid(doc) {
		return nil, fmt.Errorf("%w: not valid UTF-8", ErrInvalidDocument)
	}
	dec := json.NewDecoder(bytes.NewReader(doc))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidDocument, err)
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("%w: data after the first value", ErrInvalidDocument)
	}
	return v, nil
}

// writer accumulates ops into one compact JSON array.
type writer struct {
	buf bytes.Buffer
	n   int
}

func (w *writer) diff(path string, a, b any) {
	switch av := a.(type) {
	case map[string]any:
		if bv, ok := b.(map[string]any); ok {
			w.diffObject(path, av, bv)
			return
		}
	case []any:
		if bv, ok := b.([]any); ok {
			w.diffArray(path, av, bv)
			return
		}
	default:
		if scalarEqual(a, b) {
			return
		}
	}
	w.op("replace", path, b, true)
}

func (w *writer) diffObject(path string, a, b map[string]any) {
	keys := make([]string, 0, len(a)+len(b))
	for k := range a {
		keys = append(keys, k)
	}
	for k := range b {
		if _, ok := a[k]; !ok {
			keys = append(keys, k)
		}
	}
	sortKeys(keys)
	for _, k := range keys {
		p := path + "/" + escapePointer(k)
		av, inA := a[k]
		bv, inB := b[k]
		switch {
		case inA && inB:
			w.diff(p, av, bv)
		case inA:
			w.op("remove", p, nil, false)
		default:
			w.op("add", p, bv, true)
		}
	}
}

func (w *writer) diffArray(path string, a, b []any) {
	n := min(len(a), len(b))
	for i := range n {
		w.diff(path+"/"+strconv.Itoa(i), a[i], b[i])
	}
	for i := n; i < len(b); i++ {
		w.op("add", path+"/"+strconv.Itoa(i), b[i], true)
	}
	// Highest index first, so each remove's index is still valid when applied.
	for i := len(a) - 1; i >= n; i-- {
		w.op("remove", path+"/"+strconv.Itoa(i), nil, false)
	}
}

func (w *writer) op(name, path string, value any, withValue bool) {
	if w.n > 0 {
		w.buf.WriteByte(',')
	}
	w.n++
	w.buf.WriteString(`{"op":"`)
	w.buf.WriteString(name)
	w.buf.WriteString(`","path":`)
	writeString(&w.buf, path)
	if withValue {
		w.buf.WriteString(`,"value":`)
		writeValue(&w.buf, value)
	}
	w.buf.WriteByte('}')
}

// scalarEqual reports whether two non-container values are the same JSON
// value. Numbers compare by literal.
func scalarEqual(a, b any) bool {
	switch av := a.(type) {
	case nil:
		return b == nil
	case bool:
		bv, ok := b.(bool)
		return ok && av == bv
	case string:
		bv, ok := b.(string)
		return ok && av == bv
	case json.Number:
		bv, ok := b.(json.Number)
		return ok && av == bv
	}
	return false
}

// writeValue writes v compactly, object members in sortKeys order.
func writeValue(b *bytes.Buffer, v any) {
	switch t := v.(type) {
	case nil:
		b.WriteString("null")
	case bool:
		b.WriteString(strconv.FormatBool(t))
	case string:
		writeString(b, t)
	case json.Number:
		b.WriteString(t.String())
	case []any:
		b.WriteByte('[')
		for i, e := range t {
			if i > 0 {
				b.WriteByte(',')
			}
			writeValue(b, e)
		}
		b.WriteByte(']')
	case map[string]any:
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		sortKeys(keys)
		b.WriteByte('{')
		for i, k := range keys {
			if i > 0 {
				b.WriteByte(',')
			}
			writeString(b, k)
			b.WriteByte(':')
			writeValue(b, t[k])
		}
		b.WriteByte('}')
	}
}

const hexDigits = "0123456789abcdef"

// writeString writes s as an RFC 8785 §3.2.2.2 string: only '"', '\\' and
// U+0000..U+001F are escaped, with the short forms \b \t \n \f \r where they
// exist and \u00xx (lowercase hex) otherwise. s is valid UTF-8 (decode
// checked the input).
func writeString(b *bytes.Buffer, s string) {
	b.WriteByte('"')
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '"':
			b.WriteString(`\"`)
		case c == '\\':
			b.WriteString(`\\`)
		case c == '\b':
			b.WriteString(`\b`)
		case c == '\t':
			b.WriteString(`\t`)
		case c == '\n':
			b.WriteString(`\n`)
		case c == '\f':
			b.WriteString(`\f`)
		case c == '\r':
			b.WriteString(`\r`)
		case c < 0x20:
			b.WriteString(`\u00`)
			b.WriteByte(hexDigits[c>>4])
			b.WriteByte(hexDigits[c&0xf])
		default:
			b.WriteByte(c)
		}
	}
	b.WriteByte('"')
}

// sortKeys sorts object keys by UTF-16 code units (RFC 8785 §3.2.3). Go's
// string order is by UTF-8 bytes, which differs above U+FFFF.
func sortKeys(keys []string) {
	slices.SortFunc(keys, func(a, b string) int {
		return slices.Compare(utf16.Encode([]rune(a)), utf16.Encode([]rune(b)))
	})
}

// escapePointer escapes one RFC 6901 reference token: "~" → "~0", "/" → "~1".
func escapePointer(token string) string {
	if !strings.ContainsAny(token, "~/") {
		return token
	}
	return strings.ReplaceAll(strings.ReplaceAll(token, "~", "~0"), "/", "~1")
}

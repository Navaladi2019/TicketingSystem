// Package acl holds the pure helpers of the access-policy store that have no
// database or HTTP dependency.
package acl

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"
)

// The canonical writer of ADR-010-A1 A1-2d: it writes a JSON value in the text
// form PostgreSQL prints for jsonb, byte for byte, so that
//
//	CanonicalSize(d) == octet_length(d::jsonb::text)
//
// The route stores this writer's output, never the client's bytes, so client
// number forms and escapes never reach jsonb, and it measures the same number
// the database's bytes_ constraints measure.
//
// The jsonb text form (PostgreSQL 16, UTF8 database):
//   - ", " between members and elements, ": " after a key, no other space;
//   - an object's keys sorted by byte length, then bytewise, and a repeated
//     key keeps its last value;
//   - a string escapes only `"`, `\`, \b, \f, \n, \r, \t and the other C0
//     controls as \u00xx (lower-case hex); `/`, `<`, `>`, `&`, DEL, U+2028 and
//     U+2029 and every other character are written as UTF-8;
//   - a number is numeric's text: no exponent, the decimal scale kept (1.0
//     stays 1.0, 1e2 is 100), no leading zeros, no negative zero.

// Errors of CanonicalJSON. They never carry a value from the input.
var (
	// ErrNotJSON means the input is not one JSON value.
	ErrNotJSON = errors.New("acl: not a JSON value")
	// ErrInvalidText means the input has invalid UTF-8, a \u0000 escape or an
	// unpaired surrogate escape, which jsonb refuses.
	ErrInvalidText = errors.New("acl: text jsonb cannot hold")
	// ErrNumber means a number outside the range this writer prints (an
	// exponent beyond +-1000 or more than 1000 digits).
	ErrNumber = errors.New("acl: number out of range")
)

// maxNumberDigits bounds a printed number; far inside numeric's own limits.
const (
	maxNumberDigits = 1000
	maxExponent     = 1000
)

// CanonicalJSON returns raw, one JSON value, in PostgreSQL's jsonb text form.
func CanonicalJSON(raw []byte) ([]byte, error) {
	if !utf8.Valid(raw) {
		return nil, ErrInvalidText
	}
	if err := checkEscapes(raw); err != nil {
		return nil, err
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, ErrNotJSON
	}
	if _, err := dec.Token(); err == nil || !errors.Is(err, io.EOF) {
		return nil, ErrNotJSON // trailing data
	}
	var b bytes.Buffer
	if err := write(&b, v); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// CanonicalSize is the length of CanonicalJSON(raw): octet_length of the
// jsonb text of the value.
func CanonicalSize(raw []byte) (int, error) {
	b, err := CanonicalJSON(raw)
	return len(b), err
}

func write(b *bytes.Buffer, v any) error {
	switch x := v.(type) {
	case nil:
		b.WriteString("null")
	case bool:
		b.WriteString(strconv.FormatBool(x))
	case json.Number:
		s, err := number(string(x))
		if err != nil {
			return err
		}
		b.WriteString(s)
	case string:
		writeString(b, x)
	case []any:
		b.WriteByte('[')
		for i, e := range x {
			if i > 0 {
				b.WriteString(", ")
			}
			if err := write(b, e); err != nil {
				return err
			}
		}
		b.WriteByte(']')
	case map[string]any:
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		slices.SortFunc(keys, func(a, c string) int {
			if len(a) != len(c) {
				return len(a) - len(c)
			}
			return strings.Compare(a, c)
		})
		b.WriteByte('{')
		for i, k := range keys {
			if i > 0 {
				b.WriteString(", ")
			}
			writeString(b, k)
			b.WriteString(": ")
			if err := write(b, x[k]); err != nil {
				return err
			}
		}
		b.WriteByte('}')
	default:
		return fmt.Errorf("%w: %T", ErrNotJSON, v)
	}
	return nil
}

const hexDigits = "0123456789abcdef"

func writeString(b *bytes.Buffer, s string) {
	b.WriteByte('"')
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '"':
			b.WriteString(`\"`)
		case c == 0x5c:
			b.WriteString("\\\\")
		case c == '\b':
			b.WriteString(`\b`)
		case c == '\f':
			b.WriteString(`\f`)
		case c == '\n':
			b.WriteString(`\n`)
		case c == '\r':
			b.WriteString(`\r`)
		case c == '\t':
			b.WriteString(`\t`)
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

// number prints a JSON number token as numeric's text.
func number(tok string) (string, error) {
	neg := strings.HasPrefix(tok, "-")
	tok = strings.TrimPrefix(tok, "-")
	mant, expText, _ := strings.Cut(strings.ToLower(tok), "e")
	exp := 0
	if expText != "" {
		n, err := strconv.Atoi(strings.TrimPrefix(expText, "+"))
		if err != nil || n > maxExponent || n < -maxExponent {
			return "", ErrNumber
		}
		exp = n
	}
	whole, frac, _ := strings.Cut(mant, ".")
	digits := strings.TrimLeft(whole+frac, "0")
	scale := len(frac) - exp
	if len(digits) > maxNumberDigits || scale > maxNumberDigits || -scale > maxNumberDigits {
		return "", ErrNumber
	}
	var out string
	switch {
	case scale <= 0:
		if digits == "" {
			out = "0"
		} else {
			out = digits + strings.Repeat("0", -scale)
		}
	case digits == "":
		out = "0." + strings.Repeat("0", scale)
	default:
		if len(digits) <= scale {
			digits = strings.Repeat("0", scale-len(digits)+1) + digits
		}
		out = digits[:len(digits)-scale] + "." + digits[len(digits)-scale:]
	}
	if neg && strings.TrimLeft(out, "0.") != "" {
		out = "-" + out
	}
	return out, nil
}

// checkEscapes refuses what jsonb refuses and encoding/json would silently
// repair: \u0000 and surrogate escapes that are not a high then a low pair.
func checkEscapes(raw []byte) error {
	inString := false
	for i := 0; i < len(raw); i++ {
		c := raw[i]
		if !inString {
			inString = c == '"'
			continue
		}
		switch c {
		case '"':
			inString = false
		case 0x5c:
			if i+1 >= len(raw) {
				return ErrNotJSON
			}
			i++
			if raw[i] != 'u' {
				continue
			}
			r, ok := hex4(raw, i+1)
			if !ok {
				return ErrNotJSON
			}
			i += 4
			switch {
			case r == 0:
				return ErrInvalidText
			case r >= 0xD800 && r <= 0xDBFF:
				if i+6 >= len(raw) || raw[i+1] != 0x5c || raw[i+2] != 'u' {
					return ErrInvalidText
				}
				lo, ok := hex4(raw, i+3)
				if !ok || lo < 0xDC00 || lo > 0xDFFF {
					return ErrInvalidText
				}
				i += 6
			case r >= 0xDC00 && r <= 0xDFFF:
				return ErrInvalidText
			}
		}
	}
	return nil
}

func hex4(raw []byte, at int) (int, bool) {
	if at+4 > len(raw) {
		return 0, false
	}
	n, err := strconv.ParseInt(string(raw[at:at+4]), 16, 32)
	return int(n), err == nil
}

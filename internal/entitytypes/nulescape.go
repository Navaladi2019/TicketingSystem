package entitytypes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// nulText is what each U+0000 in a string or key becomes: the six characters
// backslash, u, 0, 0, 0, 0. jsonb refuses the escape \u0000 (22P05) but
// stores this text.
const nulText = `\u0000`

// nulEscape is the result of escapeNUL.
type nulEscape struct {
	// Doc is the escaped document, or the input as given when Escaped is
	// false. It is nil when Collided.
	Doc json.RawMessage
	// Escaped reports that at least one NUL was replaced. It sets
	// audit.Entry.DiffNULEscaped (ADR-004-E10 E10-3).
	Escaped bool
	// Collided reports that two keys of one object escape to the same text,
	// so the escaped document would lose a member (E10-5). Escaped is true
	// then too.
	Collided bool
}

// escapeNUL rewrites a stored schema so jsonb accepts it: each U+0000 in a
// string or an object key becomes nulText (TICK-1121, option A;
// ADR-004-E10). A schema with no NUL is returned as given, so its bytes, and
// so the audit diff, are unchanged.
//
// It works on decoded values, never on the JSON text. A text replace of
// \u0000 would also rewrite a stored literal \\u0000 (an escaped backslash
// then u0000), which holds no NUL. Key order and formatting are not kept:
// the caller casts the result to jsonb, which sets both.
func escapeNUL(raw json.RawMessage) (nulEscape, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nulEscape{}, fmt.Errorf("entitytypes: escape NUL: %w", err)
	}
	var w nulWalk
	v = w.value(v)
	switch {
	case w.collided:
		return nulEscape{Escaped: true, Collided: true}, nil
	case !w.escaped:
		return nulEscape{Doc: raw}, nil
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nulEscape{}, fmt.Errorf("entitytypes: escape NUL: %w", err)
	}
	return nulEscape{Doc: bytes.TrimSuffix(buf.Bytes(), []byte("\n")), Escaped: true}, nil
}

type nulWalk struct{ escaped, collided bool }

func (w *nulWalk) string(s string) string {
	if !strings.Contains(s, "\x00") {
		return s
	}
	w.escaped = true
	return strings.ReplaceAll(s, "\x00", nulText)
}

func (w *nulWalk) value(v any) any {
	switch x := v.(type) {
	case string:
		return w.string(x)
	case []any:
		for i, e := range x {
			x[i] = w.value(e)
		}
		return x
	case map[string]any:
		out := make(map[string]any, len(x))
		for k, e := range x {
			k2 := w.string(k)
			// The output map holds every key's escaped form, so this also
			// catches two NUL keys that escape alike (E10-5, TA #1).
			if _, taken := out[k2]; taken {
				w.collided = true
			}
			out[k2] = w.value(e)
		}
		return out
	default:
		return v
	}
}

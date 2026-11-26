package entitytypes

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// nulText is what each U+0000 in a string or key becomes: the six characters
// backslash, u, 0, 0, 0, 0. jsonb refuses the escape \u0000 (22P05) but
// stores this text.
const nulText = `\u0000`

// escapeNUL rewrites a stored schema so jsonb accepts it: each U+0000 in a
// string or an object key becomes nulText (TICK-1121, option A). It reports
// whether it replaced any. A schema with no NUL is returned as given, so its
// bytes, and so the audit diff, are unchanged.
//
// It works on decoded values, never on the JSON text. A text replace of
// \u0000 would also rewrite a stored literal \\u0000 (an escaped backslash
// then u0000), which holds no NUL. Key order and formatting are not kept:
// the caller casts the result to jsonb, which sets both.
func escapeNUL(raw json.RawMessage) (json.RawMessage, bool, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, false, fmt.Errorf("entitytypes: escape NUL: %w", err)
	}
	v, escaped, err := escapeNULValue(v)
	if err != nil {
		return nil, false, err
	}
	if !escaped {
		return raw, false, nil
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, false, fmt.Errorf("entitytypes: escape NUL: %w", err)
	}
	return json.RawMessage(bytes.TrimSuffix(buf.Bytes(), []byte("\n"))), true, nil
}

func escapeNULString(s string) (string, bool) {
	if !strings.Contains(s, "\x00") {
		return s, false
	}
	return strings.ReplaceAll(s, "\x00", nulText), true
}

// errNULKeyCollision: an object holds both a key with a NUL and the same
// key with the literal text in its place, so escaping would drop one. No
// schema is known to hold this; the publish fails rather than audit a diff
// that lost a member.
var errNULKeyCollision = errors.New("entitytypes: escape NUL: escaped key collides with an existing key")

func escapeNULValue(v any) (any, bool, error) {
	switch x := v.(type) {
	case string:
		s, ok := escapeNULString(x)
		return s, ok, nil
	case []any:
		escaped := false
		for i, e := range x {
			e2, ok, err := escapeNULValue(e)
			if err != nil {
				return nil, false, err
			}
			x[i], escaped = e2, escaped || ok
		}
		return x, escaped, nil
	case map[string]any:
		out := make(map[string]any, len(x))
		escaped := false
		for k, e := range x {
			k2, kok := escapeNULString(k)
			if _, taken := x[k2]; kok && taken {
				return nil, false, errNULKeyCollision
			}
			e2, eok, err := escapeNULValue(e)
			if err != nil {
				return nil, false, err
			}
			out[k2] = e2
			escaped = escaped || kok || eok
		}
		return out, escaped, nil
	default:
		return v, false, nil
	}
}

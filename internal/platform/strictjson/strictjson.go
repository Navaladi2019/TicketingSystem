// Package strictjson is the request-boundary JSON policy (TICK-964): a body
// with a duplicate object member name or invalid UTF-8 is refused, never
// read. encoding/json v1 keeps the last duplicate and turns bad bytes or a
// lone surrogate into U+FFFD without an error, so two readers of one body
// (the validator, a handler, PostgreSQL's json type) could see two different
// documents. encoding/json/v2 refuses both by default; this package is the
// one place the boundary reads that policy from.
package strictjson

import (
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"io"
)

// The rules of a Violation, the FieldError.Rule values of the 422 problem.
const (
	RuleDuplicate   = "duplicate"
	RuleInvalidUTF8 = "invalid_utf8"
)

// Violation names the first place a JSON text breaks the policy. Pointer is
// an RFC 6901 pointer into the text. Its tokens are names the decoder had
// already accepted, so it never carries the invalid bytes themselves.
type Violation struct {
	Pointer string
	Rule    string
}

// Check reports the first duplicate member name or invalid UTF-8 (including
// a lone surrogate escape) in the first JSON value of b. Any other syntax
// error, and anything after the first value, is not a violation: the request
// validator answers those as it always has.
func Check(b []byte) (Violation, bool) {
	ptr, offset, err := firstError(b)
	if err == nil {
		return Violation{}, false
	}
	if errors.Is(err, jsontext.ErrDuplicateName) {
		return Violation{Pointer: ptr, Rule: RuleDuplicate}, true
	}
	// jsontext exports no sentinel for invalid UTF-8. Read the text again
	// with only that check off: if the second pass gets past the offset the
	// first one stopped at, the first error was invalid UTF-8.
	_, permOffset, permErr := firstError(b, jsontext.AllowInvalidUTF8(true))
	if permErr == nil || permOffset != offset {
		return Violation{Pointer: ptr, Rule: RuleInvalidUTF8}, true
	}
	return Violation{}, false
}

// firstError reads the first JSON value of b token by token and returns the
// pointer and byte offset of the first syntactic error, if any.
func firstError(b []byte, opts ...jsontext.Options) (string, int64, error) {
	dec := jsontext.NewDecoder(bytes.NewReader(b), opts...)
	for {
		_, err := dec.ReadToken()
		if errors.Is(err, io.EOF) {
			return "", 0, nil
		}
		if err != nil {
			var se *jsontext.SyntacticError
			if errors.As(err, &se) {
				return string(se.JSONPointer), se.ByteOffset, err
			}
			return "", 0, err
		}
		if dec.StackDepth() == 0 {
			return "", 0, nil // the first value is complete
		}
	}
}

// Unmarshal decodes b into v with encoding/json/v2, which refuses duplicate
// names and invalid UTF-8. It is the decode for request bodies and for
// documents the server stores as sent.
func Unmarshal(b []byte, v any, opts ...json.Options) error {
	return json.Unmarshal(b, v, opts...)
}

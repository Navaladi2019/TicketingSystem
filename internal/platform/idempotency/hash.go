package idempotency

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"io"
	"strings"
	"unicode/utf8"
)

// hasDuplicateKeys reports whether any JSON object in body repeats a key. A
// body that does not tokenise reports false; the caller's decode handles it.
func hasDuplicateKeys(body []byte) bool {
	dec := json.NewDecoder(bytes.NewReader(body))
	// One entry per open container: a key set for an object, nil for an array.
	var stack []map[string]struct{}
	expectKey := false
	for {
		tok, err := dec.Token()
		if err != nil {
			return false
		}
		switch t := tok.(type) {
		case json.Delim:
			switch t {
			case '{':
				stack = append(stack, map[string]struct{}{})
				expectKey = true
				continue
			case '[':
				stack = append(stack, nil)
				expectKey = false
				continue
			}
			stack = stack[:len(stack)-1]
			expectKey = len(stack) > 0 && stack[len(stack)-1] != nil
		case string:
			if expectKey {
				seen := stack[len(stack)-1]
				k := strings.ToLower(t) // encoding/json matches field names case-insensitively
				if _, dup := seen[k]; dup {
					return true
				}
				seen[k] = struct{}{}
				expectKey = false
				continue
			}
			expectKey = len(stack) > 0 && stack[len(stack)-1] != nil
		default:
			expectKey = len(stack) > 0 && stack[len(stack)-1] != nil
		}
		if len(stack) == 0 {
			return false
		}
	}
}

// Body kinds in the hash input, so a raw body that happens to spell a JSON
// document's canonical text never collides with that document.
const (
	kindEmpty byte = iota
	kindJSON
	kindRaw
)

// RequestHash is the fingerprint stored in idempotency_keys.request_hash: a
// SHA-256 over the method, the path (with its query string, if any) and the
// canonical body. The same key with a request whose hash differs is a reuse
// (409 idempotency_key_reuse).
//
// A JSON body is canonical when it is re-encoded with sorted object keys, no
// insignificant whitespace, and numbers kept as written, so a client that
// reorders keys or reformats the document still replays. Any other body is
// hashed as its raw bytes. The method is upper-cased. Every field is length
// prefixed, so no two inputs share an encoding.
//
// The operator onboarding handler (T21) calls this with its own request.
func RequestHash(method, path string, body []byte) []byte {
	h := sha256.New()
	field := func(b []byte) {
		var n [8]byte
		binary.BigEndian.PutUint64(n[:], uint64(len(b)))
		h.Write(n[:])
		h.Write(b)
	}
	field([]byte(strings.ToUpper(method)))
	field([]byte(path))
	kind, canon := canonicalBody(body)
	field([]byte{kind})
	field(canon)
	return h.Sum(nil)
}

func canonicalBody(body []byte) (byte, []byte) {
	if len(bytes.TrimSpace(body)) == 0 {
		return kindEmpty, nil
	}
	// Re-encoding would change what these bodies mean (U+FFFD substitution,
	// last duplicate key wins), so two different bodies could share a hash.
	if !utf8.Valid(body) || hasDuplicateKeys(body) {
		return kindRaw, body
	}
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return kindRaw, body
	}
	if _, err := dec.Token(); err != io.EOF { // trailing data: not one JSON document
		return kindRaw, body
	}
	out, err := json.Marshal(v) // map keys are sorted
	if err != nil {
		return kindRaw, body
	}
	return kindJSON, out
}

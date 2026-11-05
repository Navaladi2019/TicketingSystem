package patchwalk

import (
	"errors"
	"strings"
)

// Pointer is an RFC 6901 JSON Pointer in its string form: "" for the whole
// document, otherwise "/"-separated reference tokens with "~" escaped as "~0"
// and "/" as "~1".
type Pointer string

// errBadPointer is wrapped by Pointer.Validate; it never carries the pointer.
var errBadPointer = errors.New("invalid JSON Pointer")

// Validate reports whether p is a well-formed RFC 6901 pointer.
func (p Pointer) Validate() error {
	s := string(p)
	if s == "" {
		return nil
	}
	if s[0] != '/' {
		return errBadPointer
	}
	for i := 0; i < len(s); i++ {
		if s[i] != '~' {
			continue
		}
		if i+1 >= len(s) || (s[i+1] != '0' && s[i+1] != '1') {
			return errBadPointer
		}
	}
	return nil
}

// Segments returns p's unescaped reference tokens; the root pointer has none.
// p must be valid.
func (p Pointer) Segments() []string {
	if p == "" {
		return nil
	}
	parts := strings.Split(string(p)[1:], "/")
	for i, s := range parts {
		parts[i] = unescape(s)
	}
	return parts
}

// Child returns the pointer to member or element seg below p.
func (p Pointer) Child(seg string) Pointer {
	return p + "/" + Pointer(escape(seg))
}

// HasPrefix reports whether prefix equals p or is an ancestor of p, compared
// token by token (so "/a" is a prefix of "/a/b" but not of "/ab").
func (p Pointer) HasPrefix(prefix Pointer) bool {
	if prefix == "" || p == prefix {
		return true
	}
	return strings.HasPrefix(string(p), string(prefix)+"/")
}

// FromSegments builds a pointer from unescaped reference tokens.
func FromSegments(segs []string) Pointer {
	var b strings.Builder
	for _, s := range segs {
		b.WriteByte('/')
		b.WriteString(escape(s))
	}
	return Pointer(b.String())
}

func escape(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "~", "~0"), "/", "~1")
}

func unescape(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "~1", "/"), "~0", "~")
}

// Package cursor seals pagination positions into opaque, tamper-proof tokens
// bound to a set of claims.
//
// A token is "<key id>.<position>.<mac>", each part base64url. The MAC is
// HMAC-SHA256 over the key id, the claims and the position, so a token only
// decodes for the caller that holds the same claims it was made with: the
// same tenant, principal, query and, where used, workflow. The claims are not
// in the token; the decoder supplies them from the request, which is what
// makes a replayed token fail.
//
// Every failure to decode is the one error, ErrInvalid. It does not say whether
// the token was malformed, tampered, signed by an unknown key or bound to other
// claims, so a caller cannot probe which binding failed. Map it to the
// invalid_cursor problem and nothing else.
//
// The position is signed, not encrypted: do not put anything in it that the
// holder of the token must not read.
package cursor

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
)

// Claim names used by the list endpoints. Claims is open: any other name works.
const (
	ClaimTenant    = "tenant"
	ClaimPrincipal = "principal"
	ClaimQuery     = "query"
	ClaimWorkflow  = "workflow" // optional; set only for workflow-scoped cursors
)

const (
	minKeyLen   = 32
	maxKeyIDLen = 32
	maxPosition = 1024
	maxTokenLen = 4096
	macDomain   = "nava-cursor-v1"
	tokenParts  = 3
)

// ErrInvalid is returned for every token that does not decode for the given
// claims, whatever the reason.
var ErrInvalid = errors.New("invalid cursor")

// Claims is the set of values a cursor is bound to. A claim that is absent is
// not the same as one that is empty; names and values must be non-empty.
type Claims map[string]string

// Key is one signing key. ID is carried in the token so that old cursors
// still decode after the active key changes.
type Key struct {
	ID     string
	Secret []byte
}

// Keyring signs with one key and verifies with any of its keys.
type Keyring struct {
	active string
	keys   map[string][]byte
}

// NewKeyring builds a Keyring. activeID must name one of keys. Keys that are
// short (under 32 bytes) or have a bad id are refused, so a weak key fails at
// startup and not on the first request.
func NewKeyring(activeID string, keys ...Key) (*Keyring, error) {
	kr := &Keyring{active: activeID, keys: make(map[string][]byte, len(keys))}
	for _, k := range keys {
		if !validKeyID(k.ID) {
			return nil, errors.New("cursor keyring: key id must be 1-32 characters of A-Z a-z 0-9 _ -")
		}
		if len(k.Secret) < minKeyLen {
			return nil, fmt.Errorf("cursor keyring: key %q is shorter than %d bytes", k.ID, minKeyLen)
		}
		if _, dup := kr.keys[k.ID]; dup {
			return nil, fmt.Errorf("cursor keyring: duplicate key id %q", k.ID)
		}
		kr.keys[k.ID] = append([]byte(nil), k.Secret...)
	}
	if _, ok := kr.keys[activeID]; !ok {
		return nil, fmt.Errorf("cursor keyring: active key %q is not in the keyring", activeID)
	}
	return kr, nil
}

// Encode seals position under the active key, bound to claims. Claims
// without ClaimTenant are refused with ErrInvalid.
func (kr *Keyring) Encode(claims Claims, position []byte) (string, error) {
	if len(position) > maxPosition {
		return "", fmt.Errorf("cursor: position is longer than %d bytes", maxPosition)
	}
	canon, err := canonical(claims)
	if err != nil {
		return "", err
	}
	mac := sign(kr.keys[kr.active], kr.active, canon, position)
	enc := base64.RawURLEncoding.Strict()
	return kr.active + "." + enc.EncodeToString(position) + "." + enc.EncodeToString(mac), nil
}

// Decode returns the position of token if it was sealed by a key in the
// keyring for exactly these claims. Every other outcome is ErrInvalid.
func (kr *Keyring) Decode(claims Claims, token string) ([]byte, error) {
	canon, err := canonical(claims)
	if err != nil {
		return nil, ErrInvalid
	}
	// The base64 decoder skips CR and LF, so a token holding either would
	// decode to the same bytes as the clean one; Strict does not stop that.
	parts := strings.Split(token, ".")
	if len(token) > maxTokenLen || len(parts) != tokenParts || strings.ContainsAny(token, "\r\n") {
		return nil, ErrInvalid
	}
	enc := base64.RawURLEncoding.Strict()
	position, perr := enc.DecodeString(parts[1])
	got, merr := enc.DecodeString(parts[2])
	secret, known := kr.keys[parts[0]]
	if !known {
		// Same work as a known key, so timing does not tell a retired key
		// from a forged one.
		secret = kr.keys[kr.active]
	}
	want := sign(secret, parts[0], canon, position)
	// hmac.Equal is constant time in the MAC bytes; the checks are folded
	// into one branch so no step answers early.
	ok := hmac.Equal(got, want) && perr == nil && merr == nil && known && len(position) <= maxPosition
	if !ok {
		return nil, ErrInvalid
	}
	return position, nil
}

// QueryHash digests the filter and sort part of a request for the ClaimQuery
// claim. Pass the query without the paging parameters (cursor, limit), or the
// hash changes from page to page.
func QueryHash(q url.Values) string {
	sum := sha256.Sum256([]byte(q.Encode())) // Encode sorts by key
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func validKeyID(id string) bool {
	if id == "" || len(id) > maxKeyIDLen {
		return false
	}
	for _, r := range id {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_', r == '-':
		default:
			return false
		}
	}
	return true
}

// canonical encodes claims as sorted, length-prefixed name/value pairs, so no
// two different claim sets share an encoding.
func canonical(claims Claims) ([]byte, error) {
	if claims[ClaimTenant] == "" {
		// A cursor not bound to a tenant could be replayed across tenants, so
		// the library refuses it rather than leave that to each endpoint.
		return nil, ErrInvalid
	}
	names := make([]string, 0, len(claims))
	for n, v := range claims {
		if n == "" || v == "" {
			return nil, errors.New("cursor: claim names and values must be non-empty")
		}
		names = append(names, n)
	}
	sort.Strings(names)
	var b []byte
	for _, n := range names {
		b = appendLP(b, n)
		b = appendLP(b, claims[n])
	}
	return b, nil
}

func appendLP(b []byte, s string) []byte {
	b = binary.BigEndian.AppendUint64(b, uint64(len(s)))
	return append(b, s...)
}

func sign(secret []byte, keyID string, canon, position []byte) []byte {
	m := hmac.New(sha256.New, secret)
	m.Write([]byte(macDomain))
	m.Write(appendLP(nil, keyID))
	m.Write(appendLP(nil, string(canon)))
	m.Write(position)
	return m.Sum(nil)
}

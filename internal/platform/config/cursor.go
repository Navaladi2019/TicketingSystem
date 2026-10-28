package config

import (
	"encoding/base64"
	"errors"
	"strings"
)

// CursorKey is one pagination cursor signing key. The secret is raw bytes.
type CursorKey struct {
	ID     string
	Secret []byte
}

// Cursor configures the pagination cursor keyring. No keys means not
// configured: nava-core serves no list endpoint then.
type Cursor struct {
	// ActiveKeyID is NAVA_CURSOR_ACTIVE_KEY_ID, the key new cursors are
	// signed with. It must be one of Keys.
	ActiveKeyID string
	// Keys is NAVA_CURSOR_KEYS: comma-separated "id=secret" pairs, the secret
	// base64url without padding, at least 32 bytes decoded. Every key can
	// verify, so rotation is: add the new key, make it active, and drop the
	// old key once its cursors are no longer wanted. The secrets are never
	// logged or echoed in an error.
	Keys []CursorKey
}

const minCursorKeyLen = 32

// cursor reads the keyring variables. Errors name the variable only.
func (r *reader) cursor() Cursor {
	c := Cursor{ActiveKeyID: r.set("NAVA_CURSOR_ACTIVE_KEY_ID")}
	raw := r.set("NAVA_CURSOR_KEYS")
	if raw == "" {
		if c.ActiveKeyID != "" {
			r.errs = append(r.errs, errors.New("NAVA_CURSOR_ACTIVE_KEY_ID: set without NAVA_CURSOR_KEYS"))
		}
		return Cursor{}
	}
	seen := map[string]bool{}
	for _, entry := range strings.Split(raw, ",") {
		id, secret, ok := strings.Cut(strings.TrimSpace(entry), "=")
		dec, err := base64.RawURLEncoding.Strict().DecodeString(secret)
		if !ok || id == "" || err != nil || len(dec) < minCursorKeyLen || seen[id] {
			r.errs = append(r.errs, errors.New("NAVA_CURSOR_KEYS: want unique id=secret pairs, secret base64url of at least 32 bytes"))
			return Cursor{}
		}
		seen[id] = true
		c.Keys = append(c.Keys, CursorKey{ID: id, Secret: dec})
	}
	if !seen[c.ActiveKeyID] {
		r.errs = append(r.errs, errors.New("NAVA_CURSOR_ACTIVE_KEY_ID: must name a key in NAVA_CURSOR_KEYS"))
		return Cursor{}
	}
	return c
}

// set returns key's value, empty when unset.
func (r *reader) set(key string) string {
	v, ok := r.lookup(key)
	if !ok {
		return ""
	}
	return v
}

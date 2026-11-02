package chain

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// Genesis is the prev_hash of a tenant's first sealed entry (seq 1): 32 zero
// bytes (ADR-004 R13). Treat it as a constant; TestGenesisIsZero fails if
// anything changes it.
var Genesis [32]byte //nolint:gochecknoglobals // C1 contract: the chain's fixed genesis value, read-only

// EnvelopeSHA256 returns SHA-256(JCS(e)), the envelope_sha256 of an entry.
func EnvelopeSHA256(e Envelope) ([32]byte, error) {
	b, err := CanonicalEnvelope(e)
	if err != nil {
		return [32]byte{}, err
	}
	return sha256.Sum256(b), nil
}

// EntryHash returns SHA-256(prev ‖ envelopeSHA), the entry_hash that links an
// entry to the one sealed before it in seq order.
func EntryHash(prev [32]byte, envelopeSHA [32]byte) [32]byte {
	var buf [64]byte
	copy(buf[:32], prev[:])
	copy(buf[32:], envelopeSHA[:])
	return sha256.Sum256(buf[:])
}

// DiffSHA256 returns SHA-256(salt ‖ diff), the diff_sha256 of an entry. diff is
// the exact bytes stored in the json (not jsonb) column, never re-encoded: two
// diffs that differ only in whitespace hash differently. A nil or empty diff
// hashes the salt alone; callers store a NULL diff_sha256 for an entry with no
// diff.
func DiffSHA256(salt [16]byte, diff []byte) [32]byte {
	h := sha256.New()
	h.Write(salt[:])
	h.Write(diff)
	var out [32]byte
	h.Sum(out[:0])
	return out
}

// Hex renders a hash as lowercase hex, the bytea form of the export format.
func Hex(b []byte) string { return hex.EncodeToString(b) }

// ParseHash32 parses exactly 64 lowercase hex characters.
func ParseHash32(s string) ([32]byte, error) {
	var out [32]byte
	if err := parseLowerHex(s, out[:]); err != nil {
		return out, err
	}
	return out, nil
}

// ParseSalt16 parses exactly 32 lowercase hex characters (a diff_salt).
func ParseSalt16(s string) ([16]byte, error) {
	var out [16]byte
	if err := parseLowerHex(s, out[:]); err != nil {
		return out, err
	}
	return out, nil
}

func parseLowerHex(s string, dst []byte) error {
	if len(s) != 2*len(dst) {
		return fmt.Errorf("%w: want %d lowercase hex characters, got %d", ErrInvalidEnvelope, 2*len(dst), len(s))
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return fmt.Errorf("%w: hex must be lowercase [0-9a-f]", ErrInvalidEnvelope)
		}
	}
	_, err := hex.Decode(dst, []byte(s))
	if err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidEnvelope, err)
	}
	return nil
}

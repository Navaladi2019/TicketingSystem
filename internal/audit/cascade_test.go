package audit

import (
	"crypto/sha1" //nolint:gosec // RFC 4122 UUIDv5 is defined over SHA-1; this is an independent check of it
	"testing"

	"github.com/google/uuid"
)

// cascadeKeyByHand is RFC 4122 §4.3 written out, independent of
// uuid.NewSHA1: SHA-1 over the namespace's 16 bytes and the 64-byte name,
// the first 16 bytes, version 5 and the RFC 4122 variant.
func cascadeKeyByHand(t, o, r, d uuid.UUID) uuid.UUID {
	h := sha1.New() //nolint:gosec // see the import
	h.Write(CascadeKeyNamespace[:])
	for _, id := range []uuid.UUID{t, o, r, d} {
		h.Write(id[:])
	}
	var u uuid.UUID
	copy(u[:], h.Sum(nil))
	u[6] = u[6]&0x0f | 0x50
	u[8] = u[8]&0x3f | 0x80
	return u
}

// ADR-004-E1 R2a: K(T, O, R, D) over the raw bytes in that order, in the
// pinned namespace. The golden values hold the namespace and the order; a
// permuted tuple gives another key.
func TestCascadeKey(t *testing.T) {
	if got := CascadeKeyNamespace.String(); got != "e3045a06-15ad-46d2-b1a9-0dd05b182547" {
		t.Fatalf("namespace %s", got)
	}
	tn := uuid.MustParse("0190a000-0000-7000-8000-000000000001")
	o := uuid.MustParse("0190a000-0000-7000-8000-000000000002")
	r := uuid.MustParse("0190a000-0000-7000-8000-000000000003")
	d := uuid.MustParse("0190a000-0000-7000-8000-000000000004")
	for _, tc := range []struct {
		name       string
		t, o, r, d uuid.UUID
		want       string
	}{
		{"cycle", tn, o, r, d, "de856e0f-9056-5a56-add3-df9f13dd5d4a"},
		{"permuted origin and record", tn, r, o, d, "71893c5e-d402-5e8d-add4-6184b59eed20"},
		{"too deep, no record", tn, o, uuid.Nil, uuid.Nil, "0780c08a-1844-5eec-8c92-8df21ead3af8"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := CascadeKey(tc.t, tc.o, tc.r, tc.d)
			if got.String() != tc.want {
				t.Errorf("CascadeKey = %s, want %s", got, tc.want)
			}
			if by := cascadeKeyByHand(tc.t, tc.o, tc.r, tc.d); by != got {
				t.Errorf("CascadeKey = %s, RFC 4122 by hand = %s", got, by)
			}
			if got.Version() != 5 || got.Variant() != uuid.RFC4122 {
				t.Errorf("version %d variant %v, want 5 and RFC 4122", got.Version(), got.Variant())
			}
		})
	}
}

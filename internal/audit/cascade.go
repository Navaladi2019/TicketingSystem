package audit

import "github.com/google/uuid"

// cascadeKeyNamespace is the pinned UUIDv5 namespace of the cascade key
// (ADR-004-E1 R2a): a fresh random UUID used for this purpose only, not an
// RFC 4122 namespace, not onboard.Namespace, not a tenant id.
const cascadeKeyNamespace = "e3045a06-15ad-46d2-b1a9-0dd05b182547"

// CascadeKeyNamespace returns the cascade key's namespace (ADR-004-E1 R2a).
func CascadeKeyNamespace() uuid.UUID { return uuid.MustParse(cascadeKeyNamespace) }

// CascadeKey is K(T, O, R, D), the subject_id of a cascade_key entry
// (ADR-004-E1 R2a): uuid.NewSHA1(CascadeKeyNamespace(), T ‖ O ‖ R ‖ D) over the
// 64 raw bytes in that order. A too-deep entry with no record passes
// uuid.Nil for R and D.
func CascadeKey(tenant, origin, record, definition uuid.UUID) uuid.UUID {
	b := make([]byte, 0, 64)
	b = append(b, tenant[:]...)
	b = append(b, origin[:]...)
	b = append(b, record[:]...)
	b = append(b, definition[:]...)
	return uuid.NewSHA1(CascadeKeyNamespace(), b)
}

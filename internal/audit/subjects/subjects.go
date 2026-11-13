// Package subjects is the subject-type registry of ADR-004 R2.
// audit_log's audit_log_subject_type_check mirrors it exactly; a new subject
// type is a platform migration (I8) that changes both in the same pull
// request, and the integration test TestP510bSubjectTypesMirrorRegistry
// fails on any drift. Each configuration epic registers its own types
// (ADR-004 Consequences).
//
// It is a leaf package with no platform imports, so both internal/audit and
// internal/db/audited (run by internal/db/migrate) can use it without a
// dependency from db/ back to audit (TICK-428).
package subjects

import "slices"

// Type is what an audit entry is about: audit_log.subject_type.
type Type string

// The registered subject types.
const (
	// Record is a record of any entity type (domain record).
	Record Type = "record"
	// CascadeKey is a suppressed or rejected cascade start, keyed by the
	// UUIDv5 cascade key K(T, O, R, D) (ADR-004-E1 R2a).
	CascadeKey Type = "cascade_key"
	// EntityDefinition is an entity type and its published versions (S8;
	// domain config).
	EntityDefinition Type = "entity_definition"
	// ACLPolicy is an entity type's access policy and its published versions
	// (S10, P3-35; domain config): subject_id is the entity definition id,
	// version_before and version_after are ACL versions. Its own subject so
	// the ACL chain never shares a (subject, version) key with the entity
	// type's schema versions (contract amendment A3).
	ACLPolicy Type = "acl_policy"
	// NumberFormat is a number format and its published versions (S18,
	// P6-06, ADR-005 §3.7 C12; domain config): subject_id is the format id,
	// version_before and version_after are format versions.
	NumberFormat Type = "number_format"
	// AuditExport is one export of the log (R19; domain audit).
	AuditExport Type = "audit_export"
	// AuditChain is a tenant's chain head, the subject of chain:reseed (R17;
	// domain audit).
	AuditChain Type = "audit_chain"
	// ErasureRequest is one erasure request; its audit/erase entries list
	// the entries it erased (R23; domain audit).
	ErasureRequest Type = "erasure_request"
)

// All returns every registered subject type, sorted.
func All() []Type {
	out := []Type{
		Record,
		CascadeKey,
		EntityDefinition,
		ACLPolicy,
		NumberFormat,
		AuditExport,
		AuditChain,
		ErasureRequest,
	}
	slices.Sort(out)
	return out
}

// Valid reports whether s is a registered subject type.
func (s Type) Valid() bool { return slices.Contains(All(), s) }

// Package audit is the platform's one audit log (ADR-004).
//
// This file is the subject-type registry of ADR-004 R2. audit_log's
// audit_log_subject_type_check mirrors it exactly; a new subject type is a
// platform migration (I8) that changes both in the same pull request, and
// the integration test TestP510bSubjectTypesMirrorRegistry fails on any
// drift. Each configuration epic registers its own types (ADR-004
// Consequences).
package audit

import "slices"

// SubjectType is what an audit entry is about: audit_log.subject_type.
type SubjectType string

// The registered subject types.
const (
	// SubjectRecord is a record of any entity type (domain record).
	SubjectRecord SubjectType = "record"
	// SubjectCascadeKey is a suppressed or rejected cascade start, keyed by
	// the UUIDv5 cascade key K(T, O, R, D) (ADR-004-E1 R2a).
	SubjectCascadeKey SubjectType = "cascade_key"
	// SubjectEntityDefinition is an entity type and its published versions
	// (S8; domain config).
	SubjectEntityDefinition SubjectType = "entity_definition"
	// SubjectAuditExport is one export of the log (R19; domain audit).
	SubjectAuditExport SubjectType = "audit_export"
	// SubjectAuditChain is a tenant's chain head, the subject of chain:reseed
	// (R17; domain audit).
	SubjectAuditChain SubjectType = "audit_chain"
	// SubjectErasureRequest is one erasure request; its audit/erase entries
	// list the entries it erased (R23; domain audit).
	SubjectErasureRequest SubjectType = "erasure_request"
)

// SubjectTypes returns every registered subject type, sorted.
func SubjectTypes() []SubjectType {
	out := []SubjectType{
		SubjectRecord,
		SubjectCascadeKey,
		SubjectEntityDefinition,
		SubjectAuditExport,
		SubjectAuditChain,
		SubjectErasureRequest,
	}
	slices.Sort(out)
	return out
}

// Valid reports whether s is a registered subject type.
func (s SubjectType) Valid() bool { return slices.Contains(SubjectTypes(), s) }

// Package audit is the platform's one audit log (ADR-004).
//
// The subject-type registry of ADR-004 R2 lives in the leaf package
// internal/audit/subjects, so internal/db/audited can validate the manifest
// without importing this package (TICK-428). The names below keep this
// package's API.
package audit

import "github.com/Navaladi2019/TicketingSystem/internal/audit/subjects"

// SubjectType is what an audit entry is about: audit_log.subject_type.
type SubjectType = subjects.Type

// The registered subject types (subjects.All).
const (
	SubjectRecord           = subjects.Record
	SubjectCascadeKey       = subjects.CascadeKey
	SubjectEntityDefinition = subjects.EntityDefinition
	SubjectACLPolicy        = subjects.ACLPolicy
	SubjectNumberFormat     = subjects.NumberFormat
	SubjectTenantSetting    = subjects.TenantSetting
	SubjectAuditExport      = subjects.AuditExport
	SubjectAuditChain       = subjects.AuditChain
	SubjectErasureRequest   = subjects.ErasureRequest
)

// SubjectTypes returns every registered subject type, sorted.
func SubjectTypes() []SubjectType { return subjects.All() }

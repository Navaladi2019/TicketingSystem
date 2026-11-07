package workerguard

// The audit.seal ApplicationError types the guard records as is (TICK-585).
// They live here, not in internal/audit/seal, because seal imports the guard;
// seal.ErrTypeUnsealable and seal.ErrTypeHeadNotFound are these constants.
// Both are built only with a constant message and no cause or details
// (internal/audit/seal/activities.go), which is what failureTypeAllowed
// requires. navalint guarderrtype enforces it: it reports these types outside
// internal/workerguard and internal/audit/seal, and a non-constant message
// inside internal/audit/seal (TICK-594).
const (
	// ErrTypeAuditSealUnsealable: a poison entry fails A2 (ErrUnsealable).
	ErrTypeAuditSealUnsealable = "nava.audit.seal.Unsealable"
	// ErrTypeAuditSealHeadNotFound: the tenant has no audit_chain_head (A1).
	ErrTypeAuditSealHeadNotFound = "nava.audit.seal.HeadNotFound"
)

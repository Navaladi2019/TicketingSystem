package workerguard

const (
	ErrTypeTenantGuard = "nava.TenantGuard"
	ErrTypeWorkflowRef = "nava.WorkflowRefNotFound"
	ErrTypeOpaque      = "nava.Opaque"

	ErrTypeWorkflowRejected = "nava.WorkflowRejected"

	ErrTypeAuditSealUnsealable   = "nava.audit.seal.Unsealable"
	ErrTypeAuditSealHeadNotFound = "nava.audit.seal.HeadNotFound"
)

func guardError(text string) (string, string) { return text, ErrTypeTenantGuard }

func refError() (string, string) { return "no row", "nava.WorkflowRefNotFound" }

// The guard itself may build a seal failure with any message.
func sealError(text string) (string, string) { return text, ErrTypeAuditSealUnsealable }

package workerguard

const (
	ErrTypeTenantGuard = "nava.TenantGuard"
	ErrTypeWorkflowRef = "nava.WorkflowRefNotFound"
	ErrTypeOpaque      = "nava.Opaque"

	ErrTypeWorkflowRejected = "nava.WorkflowRejected"
)

func guardError(text string) (string, string) { return text, ErrTypeTenantGuard }

func refError() (string, string) { return "no row", "nava.WorkflowRefNotFound" }

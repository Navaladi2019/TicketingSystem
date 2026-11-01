package feature

import (
	"github.com/Navaladi2019/TicketingSystem/internal/workerguard"
)

func newApplicationError(msg, typ string) error { return nil }

func forge(title string) []error {
	return []error{
		newApplicationError(title, workerguard.ErrTypeTenantGuard),      // want `workerguard.ErrTypeTenantGuard used outside internal/workerguard`
		newApplicationError(title, workerguard.ErrTypeWorkflowRef),      // want `workerguard.ErrTypeWorkflowRef used outside`
		newApplicationError(title, "nava.TenantGuard"),                  // want `failure type "nava.TenantGuard" outside internal/workerguard`
		newApplicationError(title, "nava."+"WorkflowRefNotFound"),       // want `failure type "nava.WorkflowRefNotFound" outside`
		newApplicationError(title, workerguard.ErrTypeWorkflowRejected), // want `workerguard.ErrTypeWorkflowRejected used outside`
		newApplicationError(title, "nava.WorkflowRejected"),             // want `failure type "nava.WorkflowRejected" outside`
		newApplicationError(title, workerguard.ErrTypeOpaque),
		newApplicationError(title, "billing.Declined"),
		newApplicationError("the nava.TenantGuard type is the guard's", "x"), // prose is fine
	}
}

const forged = "nava.TenantGuard" // want `failure type "nava.TenantGuard" outside`

func isGuard(typ string) bool {
	return typ == workerguard.ErrTypeTenantGuard // want `workerguard.ErrTypeTenantGuard used outside`
}

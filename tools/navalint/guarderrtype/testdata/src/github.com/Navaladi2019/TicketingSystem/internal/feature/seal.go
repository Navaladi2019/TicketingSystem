package feature

import (
	"github.com/Navaladi2019/TicketingSystem/internal/audit/seal"
	"github.com/Navaladi2019/TicketingSystem/internal/workerguard"
)

// The audit.seal failure types are built only by the guard and the sealer
// (TICK-594); feature code may not build one, even with a constant message.
func forgeSeal(title string) []error {
	return []error{
		newApplicationError(title, workerguard.ErrTypeAuditSealUnsealable), // want `workerguard.ErrTypeAuditSealUnsealable used outside internal/workerguard and internal/audit/seal`
		newApplicationError("x", workerguard.ErrTypeAuditSealHeadNotFound), // want `workerguard.ErrTypeAuditSealHeadNotFound used outside internal/workerguard and internal/audit/seal`
		newApplicationError(title, seal.ErrTypeUnsealable),                 // want `seal.ErrTypeUnsealable used outside internal/workerguard and internal/audit/seal`
		newApplicationError(title, seal.ErrTypeHeadNotFound),               // want `seal.ErrTypeHeadNotFound used outside internal/workerguard and internal/audit/seal`
		newApplicationError(title, "nava.audit.seal.Unsealable"),           // want `failure type "nava.audit.seal.Unsealable" outside internal/workerguard and internal/audit/seal`
		newApplicationError(title, "nava.audit.seal."+"HeadNotFound"),      // want `failure type "nava.audit.seal.HeadNotFound" outside internal/workerguard and internal/audit/seal`
		newApplicationError(title, seal.ErrTypeNoRun),                      // not a guard type
		newApplicationError("the nava.audit.seal.Unsealable type is", "x"), // prose is fine
	}
}

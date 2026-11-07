//go:build lintfixture

package lintfixture

import (
	"go.temporal.io/sdk/temporal"

	"github.com/Navaladi2019/TicketingSystem/internal/audit/seal"
	"github.com/Navaladi2019/TicketingSystem/internal/workerguard"
)

// ForgeGuardFailure builds the worker guard's allowed failure types outside
// internal/workerguard, so a title would be recorded in the history unchanged
// (AR-044). The navalint guarderrtype analyzer must report both.
func ForgeGuardFailure(title string) []error {
	return []error{
		temporal.NewApplicationError(title, workerguard.ErrTypeTenantGuard),
		temporal.NewApplicationError(title, "nava.WorkflowRefNotFound"),
	}
}

// ForgeSealFailure builds the audit.seal failure types the guard records as
// is outside internal/workerguard and internal/audit/seal (TICK-594). The
// navalint guarderrtype analyzer must report both. The non-constant message
// inside internal/audit/seal is the analyzer's own testdata case, as this
// package cannot sit in internal/audit/seal.
func ForgeSealFailure(title string) []error {
	return []error{
		temporal.NewNonRetryableApplicationError(title, seal.ErrTypeUnsealable, nil),
		temporal.NewNonRetryableApplicationError(title, workerguard.ErrTypeAuditSealHeadNotFound, nil),
	}
}

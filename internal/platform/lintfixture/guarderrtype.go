//go:build lintfixture

package lintfixture

import (
	"go.temporal.io/sdk/temporal"

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

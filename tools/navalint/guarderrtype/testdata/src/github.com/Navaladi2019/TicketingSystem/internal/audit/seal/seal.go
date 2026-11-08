package seal

import (
	"github.com/Navaladi2019/TicketingSystem/internal/workerguard"
)

func newApplicationError(msg, typ string) error { return nil }

func newNonRetryableApplicationError(msg, typ string, cause error, details ...any) error { return nil }

func plainError(msg string) error { return nil }

const (
	ErrTypeNoRun        = "nava.audit.seal.NoRun"
	ErrTypeUnsealable   = workerguard.ErrTypeAuditSealUnsealable
	ErrTypeHeadNotFound = workerguard.ErrTypeAuditSealHeadNotFound
)

const headMsg = "seal: audit chain head not found"

// The producers: a constant message is fine, however it is spelled.
func produce() []error {
	return []error{
		newNonRetryableApplicationError("seal: audit chain head not found", ErrTypeHeadNotFound, nil),
		newNonRetryableApplicationError("seal: entry cannot be canonicalised", ErrTypeUnsealable, nil),
		newNonRetryableApplicationError(headMsg, workerguard.ErrTypeAuditSealHeadNotFound, nil),
		newApplicationError("seal: "+"unsealable", "nava.audit.seal."+"Unsealable"),
	}
}

// A message from row data would reach the history unchanged.
func forge(title string, err error) []error {
	return []error{
		newNonRetryableApplicationError(title, ErrTypeUnsealable, nil),                  // want `failure type "nava.audit.seal.Unsealable" built with a non-constant message`
		newNonRetryableApplicationError("seal: "+err.Error(), ErrTypeHeadNotFound, nil), // want `failure type "nava.audit.seal.HeadNotFound" built with a non-constant message`
		newApplicationError(title, workerguard.ErrTypeAuditSealHeadNotFound),            // want `failure type "nava.audit.seal.HeadNotFound" built with a non-constant message`
		newApplicationError(title, "nava.audit.seal.Unsealable"),                        // want `failure type "nava.audit.seal.Unsealable" built with a non-constant message`
		newNonRetryableApplicationError("seal: "+err.Error(), ErrTypeNoRun, nil),        // not a guard type
		newApplicationError(title, workerguard.ErrTypeTenantGuard),                      // want `workerguard.ErrTypeTenantGuard used outside internal/workerguard`
		plainError(title),
	}
}

// Comparing the type is not building one.
func isUnsealable(typ string) bool { return typ == ErrTypeUnsealable }

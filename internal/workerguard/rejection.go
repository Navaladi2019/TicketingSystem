package workerguard

import "go.temporal.io/sdk/temporal"

// ErrTypeWorkflowRejected is the ApplicationError type of a run the workflow
// interpreter fails on purpose (P3-22a, PLAN-003 D-3-15): a business rejection
// of an update_record step, a definition version, record or step the tenant
// does not have, or a Directive the workflow cannot act on. The guard records
// its message only when it is one of the Rejection constants, so record data
// and definition literals never reach the history through it.
//
// Only Rejected builds it; AR-044's navalint check reports the type anywhere
// else outside this package.
const ErrTypeWorkflowRejected = "nava.WorkflowRejected"

// Rejection is the fixed text of one interpreter failure. The constants below
// are the only values; their text is code, never data.
type Rejection string

// The interpreter's rejections.
const (
	// RejectNotFound: the definition version, the record or a step id is not
	// the tenant's, or does not exist.
	RejectNotFound Rejection = "definition version, record or step not found"
	// RejectInput: the start input is not {definition_version_id, record_id}.
	RejectInput Rejection = "start input is not a definition version and a record"
	// RejectStep: a step id names a step that cannot be applied that way.
	RejectStep Rejection = "step cannot be applied"
	// RejectDirective: the workflow got a Directive it cannot act on (the
	// Kind switch default; the guard makes it unreachable).
	RejectDirective Rejection = "directive cannot be acted on"
	// RejectValidation: the record write failed schema validation.
	RejectValidation Rejection = "update_record rejected: validation failed"
	// RejectTransition: the record write is an illegal state transition.
	RejectTransition Rejection = "update_record rejected: illegal state transition"
	// RejectForbidden: the workflow actor may not write the record.
	RejectForbidden Rejection = "update_record rejected: forbidden"
)

// rejectionPrefix starts every Rejected message.
const rejectionPrefix = "workflow rejected: "

// known reports whether r is one of the constants.
func (r Rejection) known() bool {
	switch r {
	case RejectNotFound, RejectInput, RejectStep, RejectDirective,
		RejectValidation, RejectTransition, RejectForbidden:
		return true
	}
	return false
}

// Rejected returns the non-retryable ApplicationError for r, with no details
// and no cause. A value that is not one of the constants (a conversion from a
// string built at run time) gets ErrTypeOpaque and fixed text instead.
func Rejected(r Rejection) error {
	if !r.known() {
		return temporal.NewNonRetryableApplicationError(opaqueMessage, ErrTypeOpaque, nil)
	}
	return temporal.NewNonRetryableApplicationError(rejectionPrefix+string(r), ErrTypeWorkflowRejected, nil)
}

// rejectionMessageOK reports whether msg is the message Rejected writes for
// one of the constants. recordable keeps an ErrTypeWorkflowRejected failure's
// message only then: a failure of that type built anywhere else, with other
// text, is withheld like any other.
func rejectionMessageOK(msg string) bool {
	r, ok := cutPrefix(msg, rejectionPrefix)
	return ok && Rejection(r).known()
}

func cutPrefix(s, prefix string) (string, bool) {
	if len(s) < len(prefix) || s[:len(prefix)] != prefix {
		return "", false
	}
	return s[len(prefix):], true
}

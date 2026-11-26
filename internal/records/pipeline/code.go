package pipeline

import (
	"errors"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
)

// The stable codes ErrorCode returns. Each matches
// schema_backfill_failures_code_check (^[a-z_]{1,64}$), so the schema
// backfill stores it as a failure's code (ADR-009 D7). None carries a value.
const (
	CodeInvalidCommand         = "invalid_command"
	CodeNotFound               = "not_found"
	CodeVersionConflict        = "version_conflict"
	CodeValidationFailed       = "validation_failed"
	CodeUniqueViolation        = "unique_violation"
	CodeIllegalTransition      = "illegal_transition"
	CodeAssignmentGroupRemoved = "assignment_group_removed"
	CodeForbidden              = "forbidden"
	CodeCascadeTooDeep         = "cascade_too_deep"
	CodeRetryable              = "retryable"
	CodeDatabaseError          = "database_error"
)

// ErrorCode is err's stable lower-snake code: what a Write refusal or
// failure was, never what the record held. nil is "". A version conflict is
// CodeVersionConflict, which the schema backfill does not store: it leaves
// the id pending and repeats it (ADR-009 D7). Any error the pipeline does not
// type is CodeDatabaseError.
func ErrorCode(err error) string {
	var (
		ve *ErrValidation
		uv *ErrUniqueViolation
		it *ErrIllegalTransition
		fw *authz.ErrFieldNotWritable
	)
	switch {
	case err == nil:
		return ""
	case errors.Is(err, ErrRetryable):
		return CodeRetryable
	case errors.Is(err, ErrVersionConflict):
		return CodeVersionConflict
	case errors.Is(err, ErrInvalidCommand):
		return CodeInvalidCommand
	case errors.Is(err, ErrNotFound):
		return CodeNotFound
	case errors.As(err, &ve):
		return CodeValidationFailed
	case errors.As(err, &uv):
		return CodeUniqueViolation
	case errors.As(err, &it):
		return CodeIllegalTransition
	case errors.Is(err, ErrAssignmentGroupRemoved):
		return CodeAssignmentGroupRemoved
	case errors.Is(err, authz.ErrForbidden), errors.As(err, &fw):
		return CodeForbidden
	case errors.Is(err, outbox.ErrCascadeTooDeep):
		return CodeCascadeTooDeep
	}
	return CodeDatabaseError
}

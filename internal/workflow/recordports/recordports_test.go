package recordports

import (
	"errors"
	"fmt"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
	"github.com/Navaladi2019/TicketingSystem/internal/records/pipeline"
	"github.com/Navaladi2019/TicketingSystem/internal/workerguard"
	"github.com/Navaladi2019/TicketingSystem/internal/workflow/interp"
)

// mapError sends every refusal to its port error and leaves transient
// failures retryable.
func TestMapError(t *testing.T) {
	boom := errors.New("connection reset")
	for name, c := range map[string]struct {
		err  error
		want error // nil: returned as a plain error, not a port error
	}{
		"nil":                {nil, nil},
		"not found":          {pipeline.ErrNotFound, interp.ErrNotFound},
		"no record":          {errNoRecord, interp.ErrNotFound},
		"forbidden":          {authz.ErrForbidden, interp.ErrForbidden},
		"too deep":           {fmt.Errorf("x: %w", outbox.ErrCascadeTooDeep), interp.ErrForbidden},
		"stale cause":        {workerguard.ErrStaleCause, interp.ErrStaleCause},
		"foreign cause":      {workerguard.ErrForeignCause, interp.ErrForbidden},
		"missing cause":      {workerguard.ErrCauseNotFound, interp.ErrForbidden},
		"transition":         {&pipeline.ErrIllegalTransition{}, interp.ErrIllegalTransition},
		"validation":         {&pipeline.ErrValidation{}, interp.ErrValidation},
		"invalid command":    {pipeline.ErrInvalidCommand, interp.ErrValidation},
		"version conflict":   {pipeline.ErrVersionConflict, nil},
		"retryable":          {pipeline.ErrRetryable, nil},
		"other database err": {boom, nil},
	} {
		t.Run(name, func(t *testing.T) {
			got := mapError(c.err)
			switch {
			case c.err == nil:
				if got != nil {
					t.Errorf("mapError(nil) = %v", got)
				}
			case c.want != nil:
				if !errors.Is(got, c.want) {
					t.Errorf("mapError = %v, want %v", got, c.want)
				}
			default:
				for _, port := range []error{interp.ErrNotFound, interp.ErrValidation, interp.ErrIllegalTransition, interp.ErrForbidden} {
					if errors.Is(got, port) {
						t.Errorf("mapError = %v: a transient failure became the non-retryable %v", got, port)
					}
				}
			}
		})
	}
}

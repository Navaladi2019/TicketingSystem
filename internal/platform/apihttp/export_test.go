package apihttp

import (
	"context"
	"net/http"

	nethttpmiddleware "github.com/oapi-codegen/nethttp-middleware"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
)

// ValidationError and FieldErrors expose the unexported failure paths to the
// external tests.
func (a *API) ValidationError(err error, w http.ResponseWriter, r *http.Request, opts nethttpmiddleware.ErrorHandlerOpts) {
	a.validationError(context.Background(), err, w, r, opts)
}

func FieldErrors(err error) []problem.FieldError { return fieldErrors(err) }

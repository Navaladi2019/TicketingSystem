// Package bodyarg passes Body of a call result to a helper that supplies the
// code; the guard must fail the method value.
package bodyarg

import (
	"context"
	"net/http"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
)

func get() *problem.Registry { return nil }

func apply(mk func(context.Context, problem.Code, ...problem.FieldError) problem.Body, ctx context.Context) problem.Body {
	return mk(ctx, problem.InternalError)
}

func Handler(w http.ResponseWriter, r *http.Request) {
	b := apply(get().Body, r.Context())
	w.WriteHeader(b.Status)
}

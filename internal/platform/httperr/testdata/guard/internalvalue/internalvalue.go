// Package internalvalue uses problem.Internal as a value; the guard must fail
// both references.
package internalvalue

import (
	"net/http"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
)

var h http.HandlerFunc = problem.Internal

func Handler(w http.ResponseWriter, r *http.Request) {
	f := problem.Internal
	f(w, r)
}

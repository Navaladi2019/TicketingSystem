// Package internalcall calls problem.Internal from a handler; the guard must
// fail it.
package internalcall

import (
	"net/http"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	problem.Internal(w, r)
}

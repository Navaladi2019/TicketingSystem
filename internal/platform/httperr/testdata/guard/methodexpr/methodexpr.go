// Package methodexpr calls Registry.Write as a method expression; the guard
// must fail it.
package methodexpr

import (
	"net/http"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
)

func Handler(reg *problem.Registry, w http.ResponseWriter, r *http.Request) {
	(*problem.Registry).Write(reg, w, r, problem.InternalError)
}

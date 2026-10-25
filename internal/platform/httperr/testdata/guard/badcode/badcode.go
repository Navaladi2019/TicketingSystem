// Package badcode sends a 5xx and a non-constant code through Registry.Write;
// the guard must fail both.
package badcode

import (
	"net/http"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
)

func Handler(reg *problem.Registry, w http.ResponseWriter, r *http.Request, code problem.Code) {
	reg.Write(w, r, problem.InternalError)
	reg.Write(w, r, code)
}

// Package methodvalue takes Registry.Write as a value, so the code argument is
// out of sight; the guard must fail it.
package methodvalue

import (
	"net/http"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
)

func Handler(reg *problem.Registry, w http.ResponseWriter, r *http.Request) {
	f := reg.Write
	f(w, r, problem.InternalError)
}

// Package untypedvar declares the registry with a var that has no type; the
// guard must fail closed.
package untypedvar

import (
	"net/http"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	var reg = new(problem.Registry)
	reg.Write(w, r, problem.InternalError)
}

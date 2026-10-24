// Package callresult calls Write on a call result; the guard must fail closed.
package callresult

import (
	"net/http"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
)

func get() *problem.Registry { return nil }

func Handler(w http.ResponseWriter, r *http.Request) {
	get().Write(w, r, problem.InternalError)
}

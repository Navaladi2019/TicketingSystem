// Package shortvar takes the registry from a := call, so its type is not
// declared; the guard must fail closed.
package shortvar

import (
	"net/http"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
)

func get() *problem.Registry { return nil }

func Handler(w http.ResponseWriter, r *http.Request) {
	reg := get()
	reg.Write(w, r, problem.InternalError)
}

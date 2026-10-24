// Package okcall sends a 4xx through Registry.Write; the guard must pass it.
package okcall

import (
	"net/http"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
)

func Handler(reg *problem.Registry, w http.ResponseWriter, r *http.Request) {
	reg.Write(w, r, problem.NotFound)
}

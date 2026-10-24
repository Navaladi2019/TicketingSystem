// Package use sends a 5xx through a registry field declared in another
// package; the guard must see it.
package use

import (
	"net/http"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/httperr/testdata/guard/crosspkg/deps"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
)

func Handler(d *deps.Deps, w http.ResponseWriter, r *http.Request) {
	d.Reg.Write(w, r, problem.InternalError)
}

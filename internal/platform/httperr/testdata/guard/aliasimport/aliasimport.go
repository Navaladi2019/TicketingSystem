// Package aliasimport imports problem under another name; the guard must
// still see problem.Internal and the 5xx code.
package aliasimport

import (
	"net/http"

	pr "github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
)

func Handler(reg *pr.Registry, w http.ResponseWriter, r *http.Request) {
	pr.Internal(w, r)
	reg.Write(w, r, pr.InternalError)
}

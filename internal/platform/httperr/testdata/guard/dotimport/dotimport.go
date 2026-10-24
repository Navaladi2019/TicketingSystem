// Package dotimport dot-imports problem, which hides Internal; the guard must
// fail the import.
package dotimport

import (
	"net/http"

	. "github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	Internal(w, r)
}

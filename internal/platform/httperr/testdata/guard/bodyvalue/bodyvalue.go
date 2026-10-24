// Package bodyvalue takes Body from an unresolved registry as a value and
// builds a 500 from it; the guard must fail the method value.
package bodyvalue

import (
	"encoding/json"
	"net/http"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
)

func get() *problem.Registry { return nil }

func Handler(w http.ResponseWriter, r *http.Request) {
	reg := get()
	mk := reg.Body
	b := mk(r.Context(), problem.InternalError)
	w.WriteHeader(b.Status)
	_ = json.NewEncoder(w).Encode(b)
}

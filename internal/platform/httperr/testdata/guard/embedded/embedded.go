// Package embedded embeds *problem.Registry in a struct, so s.Write has no
// visible receiver type; the guard must fail the embedding.
package embedded

import (
	"net/http"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
)

type server struct {
	*problem.Registry
}

func Handler(s server, w http.ResponseWriter, r *http.Request) {
	s.Write(w, r, problem.InternalError)
	s.Registry.Write(w, r, problem.NotFound)
}

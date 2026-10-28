package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/api"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
)

// TICK-300: a tenant route that binds a required parameter answers 401, not
// 400, to a caller with no token. The downstream handler stands for the
// generated wrapper: it refuses a request missing X-Required with 400.
func TestAuthGateRunsBeforeParameterBinding(t *testing.T) {
	data, err := api.Files.ReadFile("errors.yaml")
	if err != nil {
		t.Fatal(err)
	}
	reg, err := problem.LoadRegistry(data)
	if err != nil {
		t.Fatal(err)
	}
	bind := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Required") == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	authn := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Authorization") != "Bearer ok" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
	h := authGate(reg, false, authn, authn)(bind)

	for _, c := range []struct {
		name, auth, required string
		want                 int
	}{
		{"no token, parameter missing", "", "", http.StatusUnauthorized},
		{"no token, parameter present", "", "v", http.StatusUnauthorized},
		{"token, parameter missing", "Bearer ok", "", http.StatusBadRequest},
		{"token, parameter present", "Bearer ok", "v", http.StatusNoContent},
	} {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/tickets", nil)
		if c.auth != "" {
			req.Header.Set("Authorization", c.auth)
		}
		if c.required != "" {
			req.Header.Set("X-Required", c.required)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != c.want {
			t.Errorf("%s: %d, want %d", c.name, rec.Code, c.want)
		}
	}
}

package server_test

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
	"github.com/Navaladi2019/TicketingSystem/internal/server"
)

// Without a pool there is no Idempotency-Key store. A tenant write must fail
// closed with 501 (not run unprotected), and only after authentication, so an
// unauthenticated caller still gets 401 and learns nothing about the route
// (TICK-343 security review).
func TestTenantWriteWithoutPoolFailsClosedAfterAuthn(t *testing.T) {
	tel, err := telemetry.New(t.Context(), config.Telemetry{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tel.Shutdown(t.Context()) })
	h, err := server.NewHandler(slog.New(slog.DiscardHandler), tel, twoTenants{}, stubPaths{})
	if err != nil {
		t.Fatal(err)
	}
	const body = `{"key":"incident","name":"Incident","number_prefix":"INC","definition":{}}`
	for _, tc := range []struct {
		token, idemKey string
		want           int
	}{
		{"", "k1", http.StatusUnauthorized},
		{"forged", "k1", http.StatusUnauthorized},
		{"token-a", "k1", http.StatusNotImplemented},
		{"token-a", "", http.StatusNotImplemented},
	} {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/entity-types", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if tc.token != "" {
			req.Header.Set("Authorization", "Bearer "+tc.token)
		}
		if tc.idemKey != "" {
			req.Header.Set("Idempotency-Key", tc.idemKey)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != tc.want {
			t.Errorf("token %q key %q: status %d, want %d: %s", tc.token, tc.idemKey, rec.Code, tc.want, rec.Body)
		}
		if rec.Code == http.StatusNotImplemented && !strings.Contains(rec.Body.String(), `"not_implemented"`) {
			t.Errorf("501 body is not the not_implemented problem: %s", rec.Body)
		}
	}
}

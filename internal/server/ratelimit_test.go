package server_test

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
	"github.com/Navaladi2019/TicketingSystem/internal/server"
)

type twoTenants struct{}

func (twoTenants) Resolve(_ context.Context, raw string) (identity.Principal, error) {
	switch raw {
	case "token-a":
		return identity.Principal{TenantID: tenantA, PrincipalID: principalA}, nil
	case "token-b":
		return identity.Principal{TenantID: tenantB, PrincipalID: uuid.New()}, nil
	}
	return identity.Principal{}, fmt.Errorf("%w: signature", identity.ErrInvalidToken)
}

func newLimitedHandler(t *testing.T, burst int) http.Handler {
	t.Helper()
	tel, err := telemetry.New(t.Context(), config.Telemetry{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tel.Shutdown(t.Context()) })
	h, err := server.NewHandler(slog.New(slog.DiscardHandler), tel, twoTenants{},
		server.WithRateLimit(config.RateLimit{Rate: 0.001, Burst: burst}))
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func meStatus(t *testing.T, h http.Handler, token string, hdr http.Header) int {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	for k, vs := range hdr {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Code
}

// A request with a forged tenant header but a token for tenant B is limited
// as tenant B.
func TestRateLimitKeyIsTheResolvedTenantNotAHeader(t *testing.T) {
	t.Parallel()
	h := newLimitedHandler(t, 2)
	forged := http.Header{"X-Tenant-Id": {tenantA.String()}, "X-Nava-Tenant": {tenantA.String()}}
	for i := range 2 {
		if got := meStatus(t, h, "token-b", forged); got != http.StatusOK {
			t.Fatalf("request %d = %d, want 200", i, got)
		}
	}
	if got := meStatus(t, h, "token-b", forged); got != http.StatusTooManyRequests {
		t.Errorf("tenant B over its limit = %d, want 429", got)
	}
	// Tenant A's bucket is untouched by B's forged headers.
	if got := meStatus(t, h, "token-a", http.Header{"X-Tenant-Id": {tenantB.String()}}); got != http.StatusOK {
		t.Errorf("tenant A = %d, want 200", got)
	}
}

func TestUnauthenticatedRequestsAreRefusedBeforeTheLimit(t *testing.T) {
	t.Parallel()
	h := newLimitedHandler(t, 1)
	for range 5 {
		if got := meStatus(t, h, "bad", nil); got != http.StatusUnauthorized {
			t.Fatalf("bad token = %d, want 401", got)
		}
	}
	if got := meStatus(t, h, "token-a", nil); got != http.StatusOK {
		t.Errorf("tenant A = %d, want 200", got)
	}
}

package health_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/health"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
)

// TICK-260: a redirect from Keycloak's management port must not report ready.
func TestKeycloakProbeRedirectIsUnhealthy(t *testing.T) {
	t.Parallel()
	var hits atomic.Int64
	ok := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { hits.Add(1) }))
	t.Cleanup(ok.Close)
	kc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, ok.URL, http.StatusFound)
	}))
	t.Cleanup(kc.Close)
	err := health.Keycloak(kc.URL, outboundClient(t)).Probe(t.Context())
	if !errors.Is(err, telemetry.ErrRedirectRefused) {
		t.Fatalf("err = %v, want ErrRedirectRefused", err)
	}
	if hits.Load() != 0 {
		t.Fatal("probe followed the redirect")
	}
}

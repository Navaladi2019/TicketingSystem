package server_test

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/Navaladi2019/TicketingSystem/api"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
	"github.com/Navaladi2019/TicketingSystem/internal/server"
)

func newAPIHandler(t *testing.T) (http.Handler, *tracetest.InMemoryExporter, *problem.Registry) {
	t.Helper()
	return newAPIHandlerWith(t, stubResolver{})
}

func newAPIHandlerWith(t *testing.T, r identity.IdentityResolver) (http.Handler, *tracetest.InMemoryExporter, *problem.Registry) {
	t.Helper()
	spans := tracetest.NewInMemoryExporter()
	tel, err := telemetry.New(t.Context(), config.Telemetry{}, telemetry.WithSpanExporter(spans))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tel.Shutdown(t.Context()) })
	h, err := server.NewHandler(slog.New(slog.DiscardHandler), tel, r)
	if err != nil {
		t.Fatal(err)
	}
	data, err := api.Files.ReadFile("errors.yaml")
	if err != nil {
		t.Fatal(err)
	}
	reg, err := problem.LoadRegistry(data)
	if err != nil {
		t.Fatal(err)
	}
	return h, spans, reg
}

func serve(t *testing.T, h http.Handler, method, target string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), method, target, nil))
	return rec
}

func routeOf(spans *tracetest.InMemoryExporter, i int) (string, bool) {
	for _, kv := range spans.GetSpans()[i].Attributes {
		if kv.Key == attribute.Key("http.route") {
			return kv.Value.AsString(), true
		}
	}
	return "", false
}

// TestMeIsDeclaredAndRecordsRoute: GET /api/v1/me without a token answers 401
// invalid_token as a registered problem and records http.route=/api/v1/me;
// an unknown /api/v1/x is a 404 problem that records no route.
func TestMeIsDeclaredAndRecordsRoute(t *testing.T) {
	t.Parallel()
	h, spans, reg := newAPIHandler(t)

	rec := serve(t, h, http.MethodGet, "/api/v1/me")
	var b problem.Body
	if err := json.Unmarshal(rec.Body.Bytes(), &b); err != nil {
		t.Fatalf("body: %v: %s", err, rec.Body)
	}
	if _, ok := reg.Lookup(b.Code); !ok || rec.Code != http.StatusUnauthorized || b.Code != problem.InvalidToken {
		t.Errorf("GET /api/v1/me = %d %+v, want 401 invalid_token", rec.Code, b)
	}
	if !strings.HasPrefix(b.Instance, "urn:nava:request:") || b.Instance == "urn:nava:request:" {
		t.Errorf("instance = %q", b.Instance)
	}

	rec = serve(t, h, http.MethodGet, "/api/v1/x")
	if rec.Code != http.StatusNotFound || rec.Header().Get("Content-Type") != problem.ContentType {
		t.Errorf("GET /api/v1/x = %d %s, want a 404 problem", rec.Code, rec.Header().Get("Content-Type"))
	}

	if got, ok := routeOf(spans, 0); !ok || got != "/api/v1/me" {
		t.Errorf("http.route = %q (set %v), want /api/v1/me", got, ok)
	}
	if got, ok := routeOf(spans, 1); ok {
		t.Errorf("unknown /api/v1/x http.route = %q, want none", got)
	}
}

func TestMethodNotAllowedIsAProblem(t *testing.T) {
	t.Parallel()
	h, _, _ := newAPIHandler(t)
	rec := serve(t, h, http.MethodPost, "/api/v1/me")
	if rec.Code != http.StatusMethodNotAllowed || rec.Header().Get("Content-Type") != problem.ContentType ||
		rec.Header().Get("Allow") == "" || !strings.Contains(rec.Body.String(), `"code":"method_not_allowed"`) {
		t.Errorf("POST /api/v1/me = %d %v %s", rec.Code, rec.Header(), rec.Body)
	}
}

func TestDocsAndSpecAreServed(t *testing.T) {
	t.Parallel()
	h, _, _ := newAPIHandler(t)

	rec := serve(t, h, http.MethodGet, "/api/v1/docs")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "/api/v1/me") ||
		!strings.Contains(rec.Header().Get("Content-Security-Policy"), "default-src 'none'") {
		t.Errorf("docs = %d %v %s", rec.Code, rec.Header(), rec.Body)
	}
	rec = serve(t, h, http.MethodGet, "/api/v1/openapi.yaml")
	if rec.Code != http.StatusOK || !strings.HasPrefix(rec.Body.String(), "openapi: 3.1") {
		t.Errorf("openapi.yaml = %d %.40s", rec.Code, rec.Body)
	}
}

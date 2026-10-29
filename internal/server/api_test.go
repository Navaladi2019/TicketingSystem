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
	return newAPIHandlerFull(t, r, stubPaths{}, slog.New(slog.DiscardHandler))
}

func newAPIHandlerFull(t *testing.T, r identity.IdentityResolver, paths identity.GroupPathReader, log *slog.Logger) (http.Handler, *tracetest.InMemoryExporter, *problem.Registry) {
	t.Helper()
	spans := tracetest.NewInMemoryExporter()
	tel, err := telemetry.New(t.Context(), config.Telemetry{}, telemetry.WithSpanExporter(spans))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tel.Shutdown(t.Context()) })
	h, err := server.NewHandler(log, tel, r, paths)
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
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("GET /api/v1/x without a token = %d, want 401 (fail closed)", rec.Code)
	}
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/x", nil)
	req.Header.Set("Authorization", "Bearer token-a")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
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
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/me", nil)
	req.Header.Set("Authorization", "Bearer token-a")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
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
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "openapi: 3.1") {
		t.Errorf("openapi.yaml = %d %.40s", rec.Code, rec.Body)
	}
}

// TICK-300: an unmatched or wrong-method path under /api/v1, whatever its
// spelling, is 401 without a token (fail closed) and 404/405 only behind one.
// Only the spec and docs routes are public.
func TestUnmatchedPathsFailClosed(t *testing.T) {
	t.Parallel()
	h, _, _ := newAPIHandler(t)
	for _, c := range []struct{ method, target string }{
		{http.MethodGet, "/api/v1/nothing"},
		{http.MethodGet, "/api/v1/openapi.yaml/"},
		{http.MethodGet, "/api/v1/OpenAPI.yaml"},
		{http.MethodGet, "/api/v1/docs/"},
		{http.MethodPost, "/api/v1/openapi.yaml"},
		{http.MethodGet, "/api/v1/%6Fpenapi.yaml/x"},
		{http.MethodGet, "/api/v1/me/"},
		{http.MethodPost, "/api/v1/me"},
		{http.MethodGet, "/api/v1"},
	} {
		if rec := serve(t, h, c.method, c.target); rec.Code != http.StatusUnauthorized {
			t.Errorf("%s %s without a token = %d, want 401", c.method, c.target, rec.Code)
		}
	}
	for _, p := range []string{"/api/v1/openapi.yaml", "/api/v1/docs"} {
		if rec := serve(t, h, http.MethodGet, p); rec.Code != http.StatusOK {
			t.Errorf("public GET %s = %d, want 200", p, rec.Code)
		}
	}
}

// A dot-dot path resolves to the cleaned route's boundary: no token is 401.
func TestDotDotPathIsAuthenticated(t *testing.T) {
	t.Parallel()
	h, _, _ := newAPIHandler(t)
	if rec := serve(t, h, http.MethodGet, "/api/v1/x/../me"); rec.Code != http.StatusUnauthorized {
		t.Errorf("GET /api/v1/x/../me = %d, want 401", rec.Code)
	}
}

// TestHeadIsServedAsGet: HEAD is advertised in Allow for every GET route, so
// it answers the GET status with no body (R3); without a token it is the
// uniform 401.
func TestHeadIsServedAsGet(t *testing.T) {
	t.Parallel()
	h, _, _ := newAPIHandler(t)
	head := func(path, auth string) *httptest.ResponseRecorder {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodHead, path, nil)
		if auth != "" {
			req.Header.Set("Authorization", auth)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}
	for _, c := range []struct {
		name, path, auth string
		want             int
	}{
		{"me with a token", "/api/v1/me", "Bearer token-a", http.StatusOK},
		{"me without a token", "/api/v1/me", "", http.StatusUnauthorized},
		{"me with a bad token", "/api/v1/me", "Bearer nope", http.StatusUnauthorized},
		{"spec", "/api/v1/openapi.yaml", "", http.StatusOK},
		{"unknown path", "/api/v1/x", "Bearer token-a", http.StatusNotFound},
	} {
		rec := head(c.path, c.auth)
		// A recorder keeps error bodies that net/http drops on HEAD; only the
		// success path, which this change owns, must be empty here.
		if rec.Code != c.want || (c.path == "/api/v1/me" && c.want == http.StatusOK && rec.Body.Len() != 0) {
			t.Errorf("%s: HEAD %s = %d with %d body bytes, want %d", c.name, c.path, rec.Code, rec.Body.Len(), c.want)
		}
	}
	get := serve(t, h, http.MethodGet, "/api/v1/me")
	if get.Code != http.StatusUnauthorized || head("/api/v1/me", "").Header().Get("Content-Type") != get.Header().Get("Content-Type") {
		t.Errorf("HEAD headers differ from GET")
	}
}

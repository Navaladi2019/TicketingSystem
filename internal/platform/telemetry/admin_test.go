package telemetry_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
)

func adminHandler(t *testing.T, log *slog.Logger, routes ...telemetry.AdminRoutes) (http.Handler, *tracetest.InMemoryExporter) {
	t.Helper()
	spans := tracetest.NewInMemoryExporter()
	tel, err := telemetry.New(t.Context(), load(t, nil), telemetry.WithSpanExporter(spans))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tel.Shutdown(t.Context()) })
	return telemetry.NewAdminHandler(log, tel, routes...), spans
}

func stubRoutes(mux *http.ServeMux) {
	ok := func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("ok")) }
	mux.HandleFunc("GET /healthz", ok)
	mux.HandleFunc("GET /readyz", ok)
	mux.HandleFunc("GET /boom", func(http.ResponseWriter, *http.Request) { panic("kaboom") })
}

// TestAdminHandlerStartsNoSpan: a scrape or probe must not start a public root
// span, an HTTP metric series or an access line.
func TestAdminHandlerStartsNoSpan(t *testing.T) {
	t.Parallel()
	var logs bytes.Buffer
	h, spans := adminHandler(t, slog.New(slog.NewJSONHandler(&logs, nil)), stubRoutes)

	for _, path := range []string{"/metrics", "/healthz", "/readyz"} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil))
		if rec.Code != http.StatusOK {
			t.Errorf("GET %s = %d, want 200", path, rec.Code)
		}
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/metrics", nil))
	if strings.Contains(rec.Body.String(), "http_server_request_duration_seconds_count") {
		t.Error("admin requests are recorded as HTTP server metrics")
	}
	if n := len(spans.GetSpans()); n != 0 {
		t.Errorf("admin requests started %d spans, want 0", n)
	}
	if logs.Len() != 0 {
		t.Errorf("admin requests wrote log lines: %s", logs.String())
	}
}

func TestAdminHandlerUnknownRouteAnd405(t *testing.T) {
	t.Parallel()
	h, _ := adminHandler(t, slog.New(slog.DiscardHandler), stubRoutes)
	for _, tc := range []struct {
		method, path string
		want         int
	}{
		{http.MethodGet, "/api/v1/tickets", http.StatusNotFound},
		{http.MethodPost, "/metrics", http.StatusMethodNotAllowed},
		{http.MethodPost, "/healthz", http.StatusMethodNotAllowed},
	} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), tc.method, tc.path, nil))
		if rec.Code != tc.want {
			t.Errorf("%s %s = %d, want %d", tc.method, tc.path, rec.Code, tc.want)
		}
	}
}

func TestAdminHandlerRecoversPanic(t *testing.T) {
	t.Parallel()
	var logs bytes.Buffer
	h, _ := adminHandler(t, slog.New(slog.NewJSONHandler(&logs, nil)), stubRoutes)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/boom", nil))
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("panicking handler = %d, want 500", rec.Code)
	}
	if !strings.Contains(logs.String(), "admin handler panic") {
		t.Errorf("panic not logged: %s", logs.String())
	}
	var line map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(logs.Bytes()), &line); err != nil {
		t.Fatalf("panic log is not one JSON line: %v\n%s", err, logs.String())
	}
	if s, _ := line["err"].(string); !strings.Contains(s, "kaboom") || line["level"] != "ERROR" {
		t.Errorf("panic line = %v, want level ERROR and err containing kaboom", line)
	}
}

func TestNewAdminServerSetsTimeouts(t *testing.T) {
	t.Parallel()
	cfg, err := config.Load(func(string) (string, bool) { return "", false })
	if err != nil {
		t.Fatal(err)
	}
	srv := telemetry.NewAdminServer(cfg.HTTP.AdminAddr, cfg.HTTP, http.NotFoundHandler(), slog.New(slog.DiscardHandler))
	if srv.Addr != ":9090" {
		t.Errorf("Addr = %q, want :9090", srv.Addr)
	}
	if srv.ReadHeaderTimeout == 0 || srv.ReadTimeout == 0 || srv.WriteTimeout == 0 || srv.IdleTimeout == 0 {
		t.Errorf("server has an unset timeout: %+v", srv)
	}
}

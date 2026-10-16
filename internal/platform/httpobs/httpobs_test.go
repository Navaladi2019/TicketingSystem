package httpobs_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/httpobs"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/trace"
)

const inbound = "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"

// harness wires the middleware around a mux with one route whose handler logs
// through the request-scoped logger and records what it saw in its context.
type harness struct {
	handler http.Handler
	logs    *bytes.Buffer
	reg     *prometheus.Registry
	seen    struct {
		requestID string
		span      trace.SpanContext
	}
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	h := &harness{logs: &bytes.Buffer{}, reg: prometheus.NewRegistry()}
	log := slog.New(slog.NewJSONHandler(h.logs, nil))
	m, err := httpobs.NewMetrics(h.reg)
	if err != nil {
		t.Fatalf("NewMetrics: %v", err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /items/{id}", func(w http.ResponseWriter, r *http.Request) {
		h.seen.requestID = httpobs.RequestID(r.Context())
		h.seen.span, _ = trace.FromContext(r.Context())
		httpobs.Logger(r.Context(), nil).InfoContext(r.Context(), "handler line")
		w.WriteHeader(http.StatusTeapot)
	})
	h.handler = httpobs.Middleware(log, m, mux)
	return h
}

func (h *harness) do(t *testing.T, path, traceparent string) *httptest.ResponseRecorder {
	t.Helper()
	return h.doMethod(t, http.MethodGet, path, traceparent)
}

func (h *harness) doMethod(t *testing.T, method, path, traceparent string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), method, path, nil)
	if traceparent != "" {
		req.Header.Set(trace.Header, traceparent)
	}
	rec := httptest.NewRecorder()
	h.handler.ServeHTTP(rec, req)
	return rec
}

func (h *harness) lines(t *testing.T) []map[string]any {
	t.Helper()
	var out []map[string]any
	for _, l := range strings.Split(strings.TrimSpace(h.logs.String()), "\n") {
		var m map[string]any
		if err := json.Unmarshal([]byte(l), &m); err != nil {
			t.Fatalf("log line %q is not JSON: %v", l, err)
		}
		out = append(out, m)
	}
	return out
}

func TestInboundTraceparentIsLoggedWithSameTraceID(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	rec := h.do(t, "/items/42", inbound)

	lines := h.lines(t)
	if len(lines) != 2 {
		t.Fatalf("got %d log lines, want handler line + access line: %s", len(lines), h.logs)
	}
	reqID := rec.Header().Get(httpobs.RequestIDHeader)
	if reqID == "" {
		t.Fatal("response has no request id header")
	}
	for _, l := range lines {
		if l["trace_id"] != "4bf92f3577b34da6a3ce929d0e0e4736" {
			t.Errorf("%q: trace_id = %v, want the inbound trace id", l["msg"], l["trace_id"])
		}
		if l["request_id"] != reqID {
			t.Errorf("%q: request_id = %v, want %s", l["msg"], l["request_id"], reqID)
		}
	}
	if h.seen.span.TraceIDString() != "4bf92f3577b34da6a3ce929d0e0e4736" || h.seen.requestID != reqID {
		t.Errorf("handler context: trace %s, request %s", h.seen.span.TraceIDString(), h.seen.requestID)
	}

	access := lines[1]
	if access["msg"] != "http request" || access["route"] != "GET /items/{id}" || access["status"] != float64(http.StatusTeapot) {
		t.Errorf("access line = %v, want route pattern and status 418", access)
	}
}

func TestMissingOrInvalidTraceparentStartsNewTrace(t *testing.T) {
	t.Parallel()
	for _, tp := range []string{"", "00-zz-bad"} {
		h := newHarness(t)
		h.do(t, "/items/1", tp)
		got := h.lines(t)[1]["trace_id"].(string)
		if len(got) != 32 || got == "4bf92f3577b34da6a3ce929d0e0e4736" {
			t.Errorf("traceparent %q: trace_id = %q, want a fresh 32-hex id", tp, got)
		}
	}
}

func TestRequestIDIsServerGenerated(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/items/1", nil)
	req.Header.Set(httpobs.RequestIDHeader, "client-chosen\nforged=1")
	rec := httptest.NewRecorder()
	h.handler.ServeHTTP(rec, req)
	if got := rec.Header().Get(httpobs.RequestIDHeader); got == "client-chosen\nforged=1" || len(got) != 32 {
		t.Errorf("request id = %q, want a server-generated 32-hex id", got)
	}
}

func TestMetricsLabelByRoutePatternAndStatusClass(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.do(t, "/items/1", "")
	h.do(t, "/items/2", "")
	h.do(t, "/nope/secret-id", "")
	// Made-up methods collapse into one OTHER series, however many a client invents.
	for _, m := range []string{"FOO1", "FOO2", "XYZ"} {
		h.doMethod(t, m, "/items/1", "")
	}

	want := `
# HELP http_requests_total HTTP requests completed, by method, route pattern and status class.
# TYPE http_requests_total counter
http_requests_total{method="GET",route="GET /items/{id}",status_class="4xx"} 2
http_requests_total{method="GET",route="unmatched",status_class="4xx"} 1
http_requests_total{method="OTHER",route="unmatched",status_class="4xx"} 3
`
	if err := testutil.GatherAndCompare(h.reg, strings.NewReader(want), "http_requests_total"); err != nil {
		t.Error(err)
	}
	if n := testutil.CollectAndCount(h.reg, "http_request_duration_seconds"); n != 3 {
		t.Errorf("duration series = %d, want 3", n)
	}

	// The access log keeps the raw method for drill-down.
	lines := h.lines(t)
	if got := lines[len(lines)-1]["method"]; got != "XYZ" {
		t.Errorf("access line method = %v, want the raw XYZ", got)
	}
}

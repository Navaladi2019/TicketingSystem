package httpobs_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"regexp"
	"slices"
	"strings"
	"testing"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/baggage"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/httpobs"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/logging"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/requestid"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
)

const (
	inboundTraceID = "4bf92f3577b34da6a3ce929d0e0e4736"
	inboundSpanID  = "00f067aa0ba902b7"
	inbound        = "00-" + inboundTraceID + "-" + inboundSpanID + "-01" // flags 01: sampled
)

var hex32 = regexp.MustCompile(`^[0-9a-f]{32}$`)

// harness wires the real middleware and telemetry around a mux with one
// route. The handler logs one line and records what its context carried.
type harness struct {
	handler http.Handler
	logs    *bytes.Buffer
	spans   *tracetest.SpanRecorder
	seen    struct {
		span     trace.SpanContext
		baggage  baggage.Baggage
		reqID    string
		requests int
	}
}

func newHarness(t *testing.T, samplerArg float64) *harness {
	t.Helper()
	h := &harness{logs: &bytes.Buffer{}, spans: tracetest.NewSpanRecorder()}
	tel, err := telemetry.New(t.Context(), config.Telemetry{
		ServiceName: "nava-core-test",
		Sampler:     config.SamplerParentBasedTraceIDRatio,
		SamplerArg:  samplerArg,
	}, telemetry.WithSpanProcessor(h.spans))
	if err != nil {
		t.Fatalf("telemetry.New: %v", err)
	}
	t.Cleanup(func() { _ = tel.Shutdown(t.Context()) })

	log := slog.New(logging.NewContextHandler(slog.NewJSONHandler(h.logs, nil)))
	mux := http.NewServeMux()
	mux.HandleFunc("GET /items/{id}", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		h.seen.span = trace.SpanContextFromContext(ctx)
		h.seen.baggage = baggage.FromContext(ctx)
		h.seen.reqID = requestid.FromContext(ctx)
		h.seen.requests++
		log.InfoContext(ctx, "handler line")
		w.WriteHeader(http.StatusTeapot)
	})
	h.handler = httpobs.Middleware(log, tel, mux)
	return h
}

func (h *harness) do(t *testing.T, path string, header map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil)
	for k, v := range header {
		req.Header.Set(k, v)
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

// accessLine returns the last "http request" line.
func (h *harness) accessLine(t *testing.T) map[string]any {
	t.Helper()
	lines := h.lines(t)
	l := lines[len(lines)-1]
	if l["msg"] != "http request" {
		t.Fatalf("last line is %v, want the access line", l)
	}
	return l
}

func (h *harness) onlySpan(t *testing.T) sdktrace.ReadOnlySpan {
	t.Helper()
	ended := h.spans.Ended()
	if len(ended) != 1 {
		t.Fatalf("recorded %d spans, want 1", len(ended))
	}
	return ended[0]
}

func TestInboundTraceparentIsLinkedNotContinued(t *testing.T) {
	t.Parallel()
	h := newHarness(t, 1)
	h.do(t, "/items/42", map[string]string{"traceparent": inbound})

	span := h.onlySpan(t)
	got := span.SpanContext().TraceID().String()
	if got == inboundTraceID || !hex32.MatchString(got) {
		t.Errorf("trace id = %s, want a new server-generated id", got)
	}
	if span.Parent().IsValid() {
		t.Errorf("span has parent %v, want a root span", span.Parent())
	}
	links := span.Links()
	if len(links) != 1 {
		t.Fatalf("span has %d links, want 1", len(links))
	}
	if l := links[0].SpanContext; l.TraceID().String() != inboundTraceID || l.SpanID().String() != inboundSpanID {
		t.Errorf("link = %s/%s, want the inbound span context", l.TraceID(), l.SpanID())
	}

	access := h.accessLine(t)
	if access["inbound_trace_id"] != inboundTraceID {
		t.Errorf("inbound_trace_id = %v, want %s", access["inbound_trace_id"], inboundTraceID)
	}
	if access["trace_id"] != got {
		t.Errorf("access trace_id = %v, want the server span's %s", access["trace_id"], got)
	}
}

func TestRequestWithoutTraceparentHasCorrelatedIDs(t *testing.T) {
	t.Parallel()
	h := newHarness(t, 1)
	rec := h.do(t, "/items/42", nil)

	lines := h.lines(t)
	if len(lines) != 2 {
		t.Fatalf("got %d log lines, want handler + access: %s", len(lines), h.logs)
	}
	handler, access := lines[0], lines[1]
	traceID, _ := handler["trace_id"].(string)
	if !hex32.MatchString(traceID) || traceID == strings.Repeat("0", 32) {
		t.Errorf("trace_id = %q, want 32 hex, non-zero", traceID)
	}
	for _, key := range []string{"trace_id", "span_id", "request_id"} {
		if handler[key] == nil || handler[key] != access[key] {
			t.Errorf("%s: handler %v, access %v, want equal and set", key, handler[key], access[key])
		}
	}
	if access["request_id"] != rec.Header().Get(requestid.Header) {
		t.Errorf("request_id %v differs from the %s header", access["request_id"], requestid.Header)
	}
	if _, ok := access["inbound_trace_id"]; ok {
		t.Error("inbound_trace_id is set with no inbound traceparent")
	}
}

func TestClientCannotForceSampling(t *testing.T) {
	t.Parallel()
	h := newHarness(t, 0)
	h.do(t, "/items/42", map[string]string{"traceparent": inbound}) // flags 01

	if h.seen.span.IsSampled() {
		t.Error("span is sampled at ratio 0: the inbound sampled flag was trusted")
	}
	if !h.seen.span.IsValid() {
		t.Error("unsampled span has no valid span context; logs would lose their trace id")
	}
	if n := len(h.spans.Ended()); n != 0 {
		t.Errorf("recorded %d spans at ratio 0, want 0", n)
	}
}

func TestInboundBaggageIsDropped(t *testing.T) {
	t.Parallel()
	h := newHarness(t, 1)
	h.do(t, "/items/42", map[string]string{"traceparent": inbound, "baggage": "tenant_id=forged,role=admin"})

	if h.seen.requests != 1 {
		t.Fatal("handler did not run")
	}
	if n := h.seen.baggage.Len(); n != 0 {
		t.Errorf("handler sees %d baggage members, want 0: %s", n, h.seen.baggage)
	}
}

func TestSpanHasRouteAndNoForbiddenAttributes(t *testing.T) {
	t.Parallel()
	h := newHarness(t, 1)
	h.do(t, "/items/42?email=a@b.test", map[string]string{
		"User-Agent":      "curl/8.0 secret-agent",
		"X-Forwarded-For": "203.0.113.7",
	})

	span := h.onlySpan(t)
	if span.Name() != "GET /items/{id}" {
		t.Errorf("span name = %q, want the route pattern", span.Name())
	}
	attrs := attrMap(span.Attributes())
	if attrs["http.route"] != "/items/{id}" {
		t.Errorf("http.route = %q, want /items/{id}", attrs["http.route"])
	}
	for _, k := range []string{
		"url.path", "url.query", "url.full", "user_agent.original", "client.address",
		"network.peer.address", "network.peer.port", "server.port", "http.request.method_original",
		"network.protocol.version",
	} {
		if v, ok := attrs[attribute.Key(k)]; ok {
			t.Errorf("span carries forbidden %s = %q", k, v)
		}
	}
	for k, v := range attrs {
		if strings.Contains(v, "42") || strings.Contains(v, "a@b.test") || strings.Contains(v, "203.0.113.7") {
			t.Errorf("span attribute %s = %q leaks a raw request value", k, v)
		}
	}
}

// TestServerSpanAttributeKeysAreGolden is the allow-list behind the filter's
// deny-list. One request carries every client-controlled input; the server
// span's key set must equal the expected set exactly. A key added by an
// otelhttp upgrade fails here until someone decides whether it may leave the
// process (deny it in telemetry.spanAttributeAllowed, or add it below).
func TestServerSpanAttributeKeysAreGolden(t *testing.T) {
	t.Parallel()
	const (
		host       = "tenant-a.evil.test:9"
		remoteAddr = "198.51.100.23:4567"
		xff        = "203.0.113.7"
		agent      = "curl/8.0 secret-agent"
		query      = "email=a@b.test&token=s3cr3t"
	)
	tests := []struct {
		name, method string
		want         []string
	}{
		{
			name:   "matched route",
			method: http.MethodGet,
			want: []string{
				"server.address", // WithServerName: "nava-core", never the Host header
				"http.request.method",
				"url.scheme",
				"http.response.status_code",
				"http.route", // set by httpobs from r.Pattern
			},
		},
		{
			name:   "made-up method",
			method: "FOO",
			want: []string{
				"server.address",
				"http.request.method", // _OTHER
				"url.scheme",
				"http.response.status_code",
				"http.response.body.size", // ServeMux writes the 405 body
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			h := newHarness(t, 1)
			req := httptest.NewRequestWithContext(t.Context(), tc.method, "/items/42?"+query, nil)
			req.Host = host
			req.RemoteAddr = remoteAddr
			req.Header.Set("User-Agent", agent)
			req.Header.Set("X-Forwarded-For", xff)
			req.Header.Set("traceparent", inbound)
			h.handler.ServeHTTP(httptest.NewRecorder(), req)

			attrs := attrMap(h.onlySpan(t).Attributes())
			got := make([]string, 0, len(attrs))
			for k := range attrs {
				got = append(got, string(k))
			}
			slices.Sort(got)
			want := slices.Sorted(slices.Values(tc.want))
			if !slices.Equal(got, want) {
				t.Errorf("span attribute keys = %v, want %v. A new key needs a decision: "+
					"deny it in telemetry.spanAttributeAllowed or add it to this test.", got, want)
			}

			if attrs["server.address"] != "nava-core" {
				t.Errorf("server.address = %q, want nava-core", attrs["server.address"])
			}
			for k, v := range attrs {
				for _, raw := range []string{"tenant-a.evil.test", "198.51.100.23", "4567", xff, agent, "a@b.test", "s3cr3t", "/items/42", "FOO"} {
					if strings.Contains(v, raw) {
						t.Errorf("span attribute %s = %q leaks client value %q", k, v, raw)
					}
				}
				if v == "9" {
					t.Errorf("span attribute %s = %q is the Host header's port", k, v)
				}
			}
		})
	}
}

func TestUnmatchedRequestHasNoRoute(t *testing.T) {
	t.Parallel()
	h := newHarness(t, 1)
	h.do(t, "/nope/secret-id", nil)

	span := h.onlySpan(t)
	if v, ok := attrMap(span.Attributes())["http.route"]; ok {
		t.Errorf("unmatched span has http.route = %q", v)
	}
	if strings.Contains(span.Name(), "secret-id") {
		t.Errorf("span name %q carries the raw path", span.Name())
	}
	if got := h.accessLine(t)["route"]; got != "unmatched" {
		t.Errorf("access route = %v, want unmatched", got)
	}
}

func TestRequestIDIsServerGenerated(t *testing.T) {
	t.Parallel()
	h := newHarness(t, 1)
	rec := h.do(t, "/items/1", map[string]string{
		requestid.Header: "client-chosen\nforged=1",
		"traceparent":    inbound,
	})
	got := rec.Header().Get(requestid.Header)
	if !hex32.MatchString(got) {
		t.Errorf("request id = %q, want a server-generated 32-hex id", got)
	}
	if got == inboundTraceID || got == h.seen.span.TraceID().String() {
		t.Errorf("request id %q is derived from a trace id", got)
	}
	if h.seen.reqID != got {
		t.Errorf("context request id %q differs from the header %q", h.seen.reqID, got)
	}
}

func attrMap(kvs []attribute.KeyValue) map[attribute.Key]string {
	m := make(map[attribute.Key]string, len(kvs))
	for _, kv := range kvs {
		m[kv.Key] = kv.Value.String()
	}
	return m
}

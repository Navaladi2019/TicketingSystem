package httpobs_test

// Adversarial tests for the 4a edge rules (TICK-50 telemetry contract).
// Every request here is written byte for byte onto a real TCP connection, so
// the client controls what a hostile client really controls: the request
// line (method, request-target, protocol), the Host header and every other
// header. A marker string is planted in each of those places; no span, link,
// metric label or log field may carry it back out.

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel/baggage"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/httpobs"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/logging"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
)

// marker is what the attacker plants. It is lower case so a case-insensitive
// search finds it however a component normalises it.
const marker = "zq7tenantb"

type edge struct {
	srv   *httptest.Server
	spans *tracetest.InMemoryExporter
	mu    sync.Mutex
	logs  bytes.Buffer
	seen  []trace.SpanContext
	bags  []baggage.Baggage
}

func (e *edge) Write(p []byte) (int, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.logs.Write(p)
}

func newEdge(t *testing.T, samplerArg float64) *edge {
	t.Helper()
	// The in-memory exporter sits behind the export filter, so the tests see
	// spans exactly as the OTLP exporter would ship them.
	e := &edge{spans: tracetest.NewInMemoryExporter()}
	tel, err := telemetry.New(t.Context(), config.Telemetry{},
		telemetry.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(samplerArg))),
		telemetry.WithSpanExporter(e.spans))
	if err != nil {
		t.Fatalf("telemetry.New: %v", err)
	}
	t.Cleanup(func() { _ = tel.Shutdown(t.Context()) })

	log := slog.New(logging.NewContextHandler(slog.NewJSONHandler(e, nil)))
	mux := http.NewServeMux()
	mux.Handle("GET /metrics", promhttp.HandlerFor(tel.Registry, promhttp.HandlerOpts{Registry: tel.Registry}))
	mux.HandleFunc("GET /probe/{id}", func(w http.ResponseWriter, r *http.Request) {
		e.mu.Lock()
		e.seen = append(e.seen, trace.SpanContextFromContext(r.Context()))
		e.bags = append(e.bags, baggage.FromContext(r.Context()))
		e.mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	})
	e.srv = httptest.NewServer(httpobs.Middleware(log, tel, mux))
	t.Cleanup(e.srv.Close)
	return e
}

// raw writes one request exactly as given and returns the status code.
func (e *edge) raw(t *testing.T, req string) int {
	t.Helper()
	var d net.Dialer
	conn, err := d.DialContext(t.Context(), "tcp", e.srv.Listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if _, err := io.WriteString(conn, req); err != nil {
		t.Fatal(err)
	}
	resp, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		t.Fatalf("reading response to %q: %v", firstLine(req), err)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
	return resp.StatusCode
}

func (e *edge) scrape(t *testing.T) string {
	t.Helper()
	resp, err := http.Get(e.srv.URL + "/metrics") //nolint:noctx // test
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return string(b)
}

func firstLine(s string) string { l, _, _ := strings.Cut(s, "\r\n"); return l }

// leaks lists every place in an exported span the marker survives: name,
// own trace state, attributes, status, events, links (their trace state and
// attributes).
func leaks(span tracetest.SpanStub) []string {
	var out []string
	has := func(s string) bool { return strings.Contains(strings.ToLower(s), marker) }
	if has(span.Name) {
		out = append(out, "name="+span.Name)
	}
	for _, kv := range span.Attributes {
		if has(kv.Value.String()) || has(string(kv.Key)) {
			out = append(out, fmt.Sprintf("attr %s=%s", kv.Key, kv.Value.String()))
		}
	}
	if has(span.SpanContext.TraceState().String()) {
		out = append(out, "tracestate="+span.SpanContext.TraceState().String())
	}
	if has(span.Status.Description) {
		out = append(out, "status="+span.Status.Description)
	}
	for _, ev := range span.Events {
		for _, kv := range ev.Attributes {
			if has(kv.Value.String()) {
				out = append(out, fmt.Sprintf("event %s %s=%s", ev.Name, kv.Key, kv.Value.String()))
			}
		}
	}
	for _, l := range span.Links {
		if has(l.SpanContext.TraceState().String()) {
			out = append(out, "link tracestate="+l.SpanContext.TraceState().String())
		}
		for _, kv := range l.Attributes {
			if has(kv.Value.String()) {
				out = append(out, fmt.Sprintf("link attr %s=%s", kv.Key, kv.Value.String()))
			}
		}
	}
	return out
}

// craftedRequests puts the marker in every client-controlled slot of the
// request: Host, method, request-target (path, query, absolute form), and
// the headers otelhttp knows about.
//
//nolint:gochecknoglobals // read-only test table shared by three tests
var craftedRequests = []struct{ name, req string }{
	{"host header", "GET /probe/1 HTTP/1.1\r\nHost: " + marker + ".example:4443\r\n\r\n"},
	{"made-up method", marker + " /probe/1 HTTP/1.1\r\nHost: a\r\n\r\n"},
	{"mixed-case standard method", "Get /probe/1 HTTP/1.1\r\nHost: a\r\n\r\n"},
	{"path id and query", "GET /probe/" + marker + "?email=" + marker + " HTTP/1.1\r\nHost: a\r\n\r\n"},
	{"unmatched path", "GET /" + marker + "/42 HTTP/1.1\r\nHost: a\r\n\r\n"},
	{"method not allowed", "POST /probe/" + marker + " HTTP/1.1\r\nHost: a\r\nContent-Length: 0\r\n\r\n"},
	{"cleaned-path redirect", "GET /" + marker + "/../probe/1 HTTP/1.1\r\nHost: a\r\n\r\n"},
	{"absolute-form target", "GET http://" + marker + ".example/probe/1?q=" + marker + " HTTP/1.1\r\nHost: a\r\n\r\n"},
	{"identity headers", "GET /probe/1 HTTP/1.1\r\nHost: a\r\nUser-Agent: " + marker +
		"\r\nX-Forwarded-For: " + marker + "\r\nForwarded: for=" + marker +
		"\r\nX-Real-Ip: " + marker + "\r\nX-" + marker + ": " + marker + "\r\n\r\n"},
	{"tracestate on a linked traceparent", "GET /probe/1 HTTP/1.1\r\nHost: a\r\ntraceparent: " + inbound +
		"\r\ntracestate: " + marker + "=" + marker + "\r\n\r\n"},
}

// TestEdgeSpansCarryNoClientControlledValue: whatever the client puts in the
// request, the recorded server span (the thing an exporter would ship) does
// not carry it, under any attribute key, not only the five the contract names.
func TestEdgeSpansCarryNoClientControlledValue(t *testing.T) {
	t.Parallel()
	for _, tc := range craftedRequests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			e := newEdge(t, 1)
			e.raw(t, tc.req)
			ended := e.spans.GetSpans()
			if len(ended) != 1 {
				t.Fatalf("recorded %d spans, want 1", len(ended))
			}
			for _, l := range leaks(ended[0]) {
				t.Errorf("%s: span carries the client's value: %s", firstLine(tc.req), l)
			}
		})
	}
}

// TestEdgeMetricsCarryNoClientControlledValue: the same crafted requests,
// repeated, never put the marker into a metric label, and only add series
// from a bounded set (method in the standard set or _OTHER, route a pattern).
func TestEdgeMetricsCarryNoClientControlledValue(t *testing.T) {
	t.Parallel()
	e := newEdge(t, 1)
	for i := range 3 {
		for _, tc := range craftedRequests {
			e.raw(t, strings.ReplaceAll(tc.req, marker, fmt.Sprintf("%s%d", marker, i)))
		}
	}
	// A protocol version the client made up never reaches a label.
	if got := e.raw(t, "GET /probe/1 HTTP/1.9\r\nHost: a\r\n\r\n"); got != http.StatusHTTPVersionNotSupported {
		t.Logf("HTTP/1.9 got %d", got)
	}
	body := e.scrape(t)
	if strings.Contains(strings.ToLower(body), marker) {
		for _, l := range strings.Split(body, "\n") {
			if strings.Contains(strings.ToLower(l), marker) {
				t.Errorf("/metrics carries the client's value: %s", l)
			}
		}
	}
	// Label names and label values only: a sample value such as a memory size
	// or a duration sum may contain "1.9" by chance.
	for _, forbidden := range []string{"server_address", "server_port", "url_path", "client_address", "network_peer", "user_agent", "network_protocol_version", `="1.9"`} {
		for _, l := range strings.Split(body, "\n") {
			if strings.Contains(l, forbidden) {
				t.Errorf("/metrics has %q: %s", forbidden, l)
			}
		}
	}
	series := 0
	for _, l := range strings.Split(body, "\n") {
		if strings.HasPrefix(l, "http_server_request_duration_seconds_count{") {
			series++
		}
	}
	// method x route x status over the crafted set: well under 20. Thirty
	// requests with ten distinct markers each would be 30+ if anything leaked.
	if series == 0 || series > 12 {
		t.Errorf("http_server_request_duration_seconds has %d series, want a small bounded set:\n%s", series, body)
	}
}

var hexTraceID = regexp.MustCompile(`^[0-9a-f]{32}$`)

// TestEdgeNeverContinuesAnyInboundTrace tries every traceparent shape and
// every non-W3C propagation header. None of them may become the parent or set
// the trace id, and none may force sampling at ratio 0.
func TestEdgeNeverContinuesAnyInboundTrace(t *testing.T) {
	t.Parallel()
	const tid = "0af7651916cd43dd8448eb211c80319c"
	headers := []string{
		"traceparent: 00-" + tid + "-b7ad6b7169203331-01",
		"traceparent: cc-" + tid + "-b7ad6b7169203331-01-future",                                             // higher version, extra field
		"traceparent: 00-" + tid + "-b7ad6b7169203331-ff",                                                    // every flag set
		"traceparent: 00-" + tid + "-b7ad6b7169203331-01\r\ntraceparent: 00-" + tid + "-b7ad6b7169203331-01", // twice
		"Traceparent: 00-" + tid + "-b7ad6b7169203331-01",                                                    // header name case
		"X-B3-TraceId: " + tid + "\r\nX-B3-SpanId: b7ad6b7169203331\r\nX-B3-Sampled: 1",
		"b3: " + tid + "-b7ad6b7169203331-1",
		"uber-trace-id: " + tid + ":b7ad6b7169203331:0:1",
		"X-Cloud-Trace-Context: " + tid + "/1;o=1",
		"X-Amzn-Trace-Id: Root=1-0af76519-16cd43dd8448eb211c80319c;Sampled=1",
	}
	for _, ratio := range []float64{1, 0} {
		for _, hdr := range headers {
			t.Run(fmt.Sprintf("ratio=%v/%s", ratio, firstLine(hdr)), func(t *testing.T) {
				t.Parallel()
				e := newEdge(t, ratio)
				if got := e.raw(t, "GET /probe/1 HTTP/1.1\r\nHost: a\r\n"+hdr+"\r\n\r\n"); got != http.StatusNoContent {
					t.Fatalf("status %d", got)
				}
				if len(e.seen) != 1 {
					t.Fatal("handler did not run")
				}
				sc := e.seen[0]
				if !sc.IsValid() || !hexTraceID.MatchString(sc.TraceID().String()) {
					t.Fatalf("handler span context %v is not a valid server span", sc)
				}
				if sc.TraceID().String() == tid {
					t.Errorf("trace id %s was taken from the client", tid)
				}
				if sc.IsRemote() {
					t.Error("handler span context is remote: the inbound context was continued")
				}
				if ratio == 0 && sc.IsSampled() {
					t.Error("client forced sampling at ratio 0")
				}
				for _, s := range e.spans.GetSpans() {
					if s.Parent.IsValid() {
						t.Errorf("server span has parent %v", s.Parent)
					}
				}
			})
		}
	}
}

// TestEdgeDropsEveryBaggageShape: no inbound baggage format reaches the
// handler's context, whatever its header name, casing or repetition.
func TestEdgeDropsEveryBaggageShape(t *testing.T) {
	t.Parallel()
	for _, hdr := range []string{
		"baggage: tenant_id=" + marker + ",role=admin",
		"Baggage: tenant_id=" + marker,
		"baggage: tenant_id=" + marker + "\r\nbaggage: role=admin",
		"traceparent: " + inbound + "\r\nbaggage: tenant_id=" + marker,
		"ot-baggage-tenant_id: " + marker,
		"uberctx-tenant_id: " + marker,
		"Correlation-Context: tenant_id=" + marker,
	} {
		t.Run(firstLine(hdr), func(t *testing.T) {
			t.Parallel()
			e := newEdge(t, 1)
			e.raw(t, "GET /probe/1 HTTP/1.1\r\nHost: a\r\n"+hdr+"\r\n\r\n")
			if len(e.bags) != 1 {
				t.Fatal("handler did not run")
			}
			if n := e.bags[0].Len(); n != 0 {
				t.Errorf("handler sees %d baggage members: %s", n, e.bags[0])
			}
		})
	}
}

// TestEdgeAccessLogCarriesNoClientControlledValue: the access line is the
// other thing that leaves the process. Its method and route fields must be
// bounded the same way the metric labels are.
func TestEdgeAccessLogCarriesNoClientControlledValue(t *testing.T) {
	t.Parallel()
	e := newEdge(t, 1)
	for _, tc := range craftedRequests {
		if tc.name == "identity headers" || tc.name == "tracestate on a linked traceparent" {
			continue // headers are never logged; covered by the span test
		}
		e.raw(t, tc.req)
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, l := range strings.Split(strings.TrimSpace(e.logs.String()), "\n") {
		if strings.Contains(strings.ToLower(l), marker) {
			t.Errorf("access line carries the client's value: %s", l)
		}
	}
}

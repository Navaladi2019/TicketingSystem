package telemetry

import (
	"context"
	"strings"
	"time"

	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/exemplar"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// spanAttributeAllowed is the span attribute deny-list. Raw paths, queries,
// user agents, client or peer addresses, the Host header's port, a raw
// non-standard method, the protocol version, SQL text and free-form error
// messages are client-chosen or free-form values that may carry ids and PII;
// none may leave the process once an exporter is on. Older semconv names are
// listed too, in case an instrumentation still emits them.
//
// It is a deny-list because application spans need free attribute keys. The
// golden key-set test in httpobs is the allow-list that backs it: a key a
// future otelhttp release adds fails that test until someone decides on it.
func spanAttributeAllowed(kv attribute.KeyValue) bool {
	switch kv.Key {
	case "url.path", "url.query", "url.full",
		"http.url", "http.target",
		"user_agent.original", "http.user_agent",
		"client.address", "client.port", "http.client_ip",
		"network.peer.address", "network.peer.port", "net.sock.peer.addr", "net.peer.ip",
		"server.port", // from the Host header when the server name has no port
		"http.request.method_original",
		"network.protocol.version", // the client picks it from HTTP/1.0 to HTTP/1.9
		"db.query.text", "db.statement",
		"exception.message":
		return false
	}
	return !strings.HasPrefix(string(kv.Key), "http.request.header.")
}

func filterAttributes(kvs []attribute.KeyValue) []attribute.KeyValue {
	out := kvs[:0:0]
	for _, kv := range kvs {
		if spanAttributeAllowed(kv) {
			out = append(out, kv)
		}
	}
	return out
}

// filteringExporter applies the deny-list at export, so it holds however the
// data reached the span: start options, SetAttributes, events (RecordError),
// links or the status. Every exporter, test ones included, sits behind it.
type filteringExporter struct {
	next sdktrace.SpanExporter
}

func (e filteringExporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	out := make([]sdktrace.ReadOnlySpan, len(spans))
	for i, s := range spans {
		out[i] = filteredSpan{ReadOnlySpan: s}
	}
	return e.next.ExportSpans(ctx, out)
}

func (e filteringExporter) Shutdown(ctx context.Context) error { return e.next.Shutdown(ctx) }

// filteredSpan is the exported view of a span with denied attributes removed.
type filteredSpan struct {
	sdktrace.ReadOnlySpan
}

func (s filteredSpan) Attributes() []attribute.KeyValue {
	return filterAttributes(s.ReadOnlySpan.Attributes())
}

func (s filteredSpan) Events() []sdktrace.Event {
	events := s.ReadOnlySpan.Events()
	out := make([]sdktrace.Event, len(events))
	for i, ev := range events {
		ev.Attributes = filterAttributes(ev.Attributes)
		out[i] = ev
	}
	return out
}

// Links keeps only the trace and span id of each link, with filtered
// attributes. A linked inbound traceparent arrives with the client's
// tracestate, which must not be exported.
func (s filteredSpan) Links() []sdktrace.Link {
	links := s.ReadOnlySpan.Links()
	out := make([]sdktrace.Link, len(links))
	for i, l := range links {
		l.SpanContext = l.SpanContext.WithTraceState(trace.TraceState{})
		l.Attributes = filterAttributes(l.Attributes)
		out[i] = l
	}
	return out
}

// Status keeps the code and drops the description: instrumentation fills it
// with err.Error(), and a *url.Error, for one, quotes the full URL with its
// query. error.type stays on the span.
func (s filteredSpan) Status() sdktrace.Status {
	return sdktrace.Status{Code: s.ReadOnlySpan.Status().Code}
}

// httpServerMetricsView keeps only bounded attributes on the otelhttp server
// metrics. otelhttp also records server.address and server.port, taken from
// the client's Host header; any client could then add a series per request.
func httpServerMetricsView() sdkmetric.View {
	return sdkmetric.NewView(
		sdkmetric.Instrument{Name: "http.server.*"},
		sdkmetric.Stream{
			AttributeFilter: attribute.NewAllowKeysFilter(
				"http.request.method", // standard methods, else _OTHER
				"http.response.status_code",
				"http.route", // the ServeMux pattern's path, never the raw path
				"url.scheme",
				"network.protocol.name", // network.protocol.version is client-chosen (HTTP/1.0 to 1.9)
				"error.type",
			),
			ExemplarReservoirProviderSelector: traceOnlyExemplars,
		},
	)
}

// traceOnlyExemplars wraps the default reservoir so an exemplar keeps only its
// trace and span id. The SDK otherwise attaches the attributes the View
// dropped (server.port from the Host header, network.protocol.version) to each
// exemplar, which puts client-chosen values back on /metrics as exemplar
// labels; OpenMetrics also rejects an exemplar whose labels pass 128
// characters, and these push it over.
func traceOnlyExemplars(agg sdkmetric.Aggregation) exemplar.ReservoirProvider {
	next := sdkmetric.DefaultExemplarReservoirProviderSelector(agg)
	return func(attr attribute.Set) exemplar.Reservoir { return attributelessReservoir{next(attr)} }
}

type attributelessReservoir struct{ exemplar.Reservoir }

func (r attributelessReservoir) Offer(ctx context.Context, t time.Time, val exemplar.Value, _ []attribute.KeyValue) {
	r.Reservoir.Offer(ctx, t, val, nil)
}

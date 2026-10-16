package telemetry

import (
	"context"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/trace"
)

// spanAttributeAllowed is the span attribute deny-list. Raw paths, queries,
// user agents, client or peer addresses, the Host header's port, a raw
// non-standard method and the protocol version are client-chosen values that may carry ids and PII;
// none may leave the process once an exporter is on. The rule is enforced
// here, on every tracer the provider hands out, not left to each call site.
//
// It is a deny-list because application spans need free attribute keys. The
// golden key-set test in httpobs is the allow-list that backs it: a key a
// future otelhttp release adds fails that test until someone decides on it.
func spanAttributeAllowed(kv attribute.KeyValue) bool {
	switch kv.Key {
	case "url.path", "url.query", "url.full",
		"user_agent.original",
		"client.address", "client.port",
		"network.peer.address", "network.peer.port",
		"server.port", // from the Host header when the server name has no port
		"http.request.method_original",
		"network.protocol.version": // the client picks it from HTTP/1.0 to HTTP/1.9
		return false
	}
	return !strings.HasPrefix(string(kv.Key), "http.request.header.")
}

func filterSpanAttributes(kvs []attribute.KeyValue) []attribute.KeyValue {
	out := kvs[:0:0]
	for _, kv := range kvs {
		if spanAttributeAllowed(kv) {
			out = append(out, kv)
		}
	}
	return out
}

// filteringTracerProvider hands out tracers whose spans drop denied attributes,
// both at start and on SetAttributes.
type filteringTracerProvider struct {
	trace.TracerProvider
}

func (p filteringTracerProvider) Tracer(name string, opts ...trace.TracerOption) trace.Tracer {
	return filteringTracer{Tracer: p.TracerProvider.Tracer(name, opts...)}
}

type filteringTracer struct {
	trace.Tracer
}

func (t filteringTracer) Start(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	cfg := trace.NewSpanStartConfig(opts...)
	clean := []trace.SpanStartOption{
		trace.WithAttributes(filterSpanAttributes(cfg.Attributes())...),
		trace.WithLinks(cleanLinks(cfg.Links())...),
		trace.WithSpanKind(cfg.SpanKind()),
	}
	if cfg.NewRoot() {
		clean = append(clean, trace.WithNewRoot())
	}
	if ts := cfg.Timestamp(); !ts.IsZero() {
		clean = append(clean, trace.WithTimestamp(ts))
	}
	ctx, span := t.Tracer.Start(ctx, name, clean...)
	fs := filteringSpan{Span: span}
	return trace.ContextWithSpan(ctx, fs), fs
}

// cleanLinks keeps only the trace and span id of each link. A linked inbound
// traceparent arrives with the client's tracestate, which must not be exported.
func cleanLinks(links []trace.Link) []trace.Link {
	out := make([]trace.Link, 0, len(links))
	for _, l := range links {
		out = append(out, trace.Link{
			SpanContext: l.SpanContext.WithTraceState(trace.TraceState{}),
			Attributes:  filterSpanAttributes(l.Attributes),
		})
	}
	return out
}

type filteringSpan struct {
	trace.Span
}

func (s filteringSpan) SetAttributes(kvs ...attribute.KeyValue) {
	s.Span.SetAttributes(filterSpanAttributes(kvs)...)
}

// TracerProvider returns the filtering provider, so child tracers obtained
// from a span stay filtered.
func (s filteringSpan) TracerProvider() trace.TracerProvider {
	return filteringTracerProvider{TracerProvider: s.Span.TracerProvider()}
}

// httpServerMetricsView keeps only bounded attributes on the otelhttp server
// metrics. otelhttp also records server.address and server.port, taken from
// the client's Host header; any client could then add a series per request.
func httpServerMetricsView() sdkmetric.View {
	return sdkmetric.NewView(
		sdkmetric.Instrument{Name: "http.server.*"},
		sdkmetric.Stream{AttributeFilter: attribute.NewAllowKeysFilter(
			"http.request.method", // standard methods, else _OTHER
			"http.response.status_code",
			"http.route", // the ServeMux pattern's path, never the raw path
			"url.scheme",
			"network.protocol.name", // network.protocol.version is client-chosen (HTTP/1.0 to 1.9)
			"error.type",
		)},
	)
}

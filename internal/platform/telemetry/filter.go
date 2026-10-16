package telemetry

import (
	"context"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/trace"
)

// spanAttributeAllowed is the span attribute deny-list. Raw paths, queries,
// user agents and client or peer addresses carry ids and PII, which must not
// leave the process once an exporter is on. The rule is enforced here, on
// every tracer the provider hands out, not left to each call site.
func spanAttributeAllowed(kv attribute.KeyValue) bool {
	switch kv.Key {
	case "url.path", "url.query", "url.full",
		"user_agent.original",
		"client.address", "client.port",
		"network.peer.address", "network.peer.port":
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
		trace.WithLinks(cfg.Links()...),
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
			"network.protocol.name",
			"network.protocol.version",
			"error.type",
		)},
	)
}

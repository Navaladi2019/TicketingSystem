package telemetry

import (
	"context"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// spanAttributeAllowed is the span attribute deny-list. Raw paths, queries,
// user agents, client or peer addresses, SQL text and free-form error messages
// carry ids and PII, which must not leave the process once an exporter is on.
// Older semconv names are listed too, in case an instrumentation still emits
// them.
func spanAttributeAllowed(kv attribute.KeyValue) bool {
	switch kv.Key {
	case "url.path", "url.query", "url.full",
		"http.url", "http.target",
		"user_agent.original", "http.user_agent",
		"client.address", "client.port", "http.client_ip",
		"network.peer.address", "network.peer.port", "net.sock.peer.addr", "net.peer.ip",
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

func (s filteredSpan) Links() []sdktrace.Link {
	links := s.ReadOnlySpan.Links()
	out := make([]sdktrace.Link, len(links))
	for i, l := range links {
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

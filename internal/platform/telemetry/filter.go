package telemetry

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/exemplar"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	"go.opentelemetry.io/otel/trace"
)

// The nava.* span attribute keys. Each one is a platform-generated id, never
// a value a client or the IdP supplies and never one that identifies a person
// (TICK-94 item 3 span id policy). Principal ids, emails and names go in logs
// only. A key is exported only if it is declared here and listed in
// spanAttributeAllowed, so a new nava.* key is a reviewed change to this file.
const (
	// TenantIDKey is the tenant the work runs for, as resolved server-side.
	TenantIDKey = attribute.Key("nava.tenant_id")
	// RequestIDKey is the server-generated request id (package requestid).
	RequestIDKey = attribute.Key("nava.request_id")
	// OutboxIDKey is the id of an outbox row the platform created.
	OutboxIDKey = attribute.Key("nava.outbox_id")
)

// The Temporal SDK's own keys for the workflow and run ids. The workflow id
// is built only by the relay (t:<tenant>:<target_key>), the run id by Temporal.
const (
	temporalWorkflowIDKey = attribute.Key("temporalWorkflowID")
	temporalRunIDKey      = attribute.Key("temporalRunID")
)

// spanAttributeAllowed is the span attribute allow-list, applied to span,
// event and link attributes. Any other key is dropped, so a key a new library
// or a library upgrade emits never leaves the process until someone adds it
// here.
//
// The semconv keys are the ones the httpobs golden key-set test pins for the
// server span, plus the error keys. exception.message is not listed:
// instrumentation fills it with err.Error(), which can quote input values.
func spanAttributeAllowed(kv attribute.KeyValue) bool {
	switch kv.Key {
	case "server.address", // fixed to nava-core (otelhttp.WithServerName)
		"http.request.method", // standard methods, else _OTHER
		"http.response.status_code",
		"http.response.body.size",
		"http.route", // the ServeMux pattern's path, never the raw path
		"url.scheme",
		"error.type",
		"exception.type",
		"exception.stacktrace",
		TenantIDKey, RequestIDKey, OutboxIDKey,
		temporalWorkflowIDKey, temporalRunIDKey:
		return true
	}
	return false
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

// filteringExporter applies the allow-list at export, so it holds however the
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

// filteredSpan is the exported view of a span: only allowed attributes, no
// link tracestate, and error.type as the status description.
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

// Status keeps the code and replaces the description with the span's
// error.type, or empty if it has none. Instrumentation fills the description
// with err.Error(), and a *url.Error, for one, quotes the full URL with its
// query; a PostgreSQL 22P02 quotes the bad input. Only an Error status carries
// a description, as in the OTel API.
func (s filteredSpan) Status() sdktrace.Status {
	st := sdktrace.Status{Code: s.ReadOnlySpan.Status().Code}
	if st.Code == codes.Error {
		for _, kv := range s.ReadOnlySpan.Attributes() {
			if kv.Key == semconv.ErrorTypeKey {
				st.Description = kv.Value.String()
			}
		}
	}
	return st
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

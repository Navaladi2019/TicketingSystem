package telemetry

import (
	"context"
	"regexp"
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
// server span, the error keys, and the otelpgx db.* keys below.
// exception.message is not listed: instrumentation fills it with err.Error(),
// which can quote input values. The error and exception keys are exported
// only as bounded tokens (boundAttribute).
//
// Each db.* key was checked against the TICK-94 item 3 span id policy; none is
// an id, a client-supplied value or a person:
//   - db.system.name: the constant "postgresql".
//   - db.namespace: the database name from the server's own config.
//   - db.operation.name: a SQL keyword, bounded by DBSpanName.
//   - db.operation.batch.size: a count.
//
// db.query.text and pgx.query.parameters stay out, as do user.name (the
// database role) and the pgx.* keys.
func spanAttributeAllowed(kv attribute.KeyValue) bool {
	switch kv.Key {
	case "server.address", // nava-core (otelhttp.WithServerName), or the database host from config
		"http.request.method", // standard methods, else _OTHER
		"http.response.status_code",
		"http.response.body.size",
		"http.route", // the ServeMux pattern's path, never the raw path
		"url.scheme",
		"error.type",
		"exception.type",
		"exception.stacktrace",
		"db.system.name",
		"db.namespace",
		"db.operation.name",
		"db.operation.batch.size",
		TenantIDKey, RequestIDKey, OutboxIDKey,
		temporalWorkflowIDKey, temporalRunIDKey:
		return true
	}
	return false
}

// otherValue replaces a value outside the expected set, as semconv spells it.
const otherValue = "_OTHER"

// boundedToken is the shape error.type and the exception keys must have to be
// exported: a type or code name such as *pgconn.PgError, 500 or
// context.DeadlineExceeded. A library that writes err.Error() or a formatted
// stack trace there produces something else, which becomes _OTHER.
var boundedToken = regexp.MustCompile(`^[A-Za-z0-9_.*/\[\]-]{1,64}$`)

// boundAttribute returns an allowed attribute with any value that must be a
// bounded token, and is not, replaced by _OTHER.
func boundAttribute(kv attribute.KeyValue) attribute.KeyValue {
	switch kv.Key {
	case semconv.ErrorTypeKey, semconv.ExceptionTypeKey, semconv.ExceptionStacktraceKey:
		if kv.Value.Type() == attribute.STRING && boundedToken.MatchString(kv.Value.AsString()) {
			return kv
		}
		return kv.Key.String(otherValue)
	case semconv.DBOperationNameKey:
		return kv.Key.String(DBSpanName(kv.Value.String()))
	}
	return kv
}

func filterAttributes(kvs []attribute.KeyValue) []attribute.KeyValue {
	out := kvs[:0:0]
	for _, kv := range kvs {
		if spanAttributeAllowed(kv) {
			out = append(out, boundAttribute(kv))
		}
	}
	return out
}

// DBSpanName returns op if it is a known SQL keyword, else _OTHER. The pool
// constructor (internal/platform/db) names otelpgx spans with it, so a span
// name never carries SQL text.
func DBSpanName(op string) string {
	switch op {
	case "SELECT", "INSERT", "UPDATE", "DELETE", "MERGE", "WITH", "VALUES",
		"BEGIN", "COMMIT", "ROLLBACK", "SAVEPOINT", "RELEASE",
		"SET", "SHOW", "RESET", "CALL", "COPY", "LOCK", "TRUNCATE",
		"UNKNOWN": // otelpgx's name for a statement with no keyword
		return op
	}
	return otherValue
}

// otelpgxScope is the instrumentation scope name of otelpgx's tracer.
const otelpgxScope = "github.com/exaring/otelpgx"

// dbSpanName is the exported name of an otelpgx span: one of the fixed names
// otelpgx uses for connect, acquire and batch spans, or a DBSpanName keyword.
// otelpgx names a span with the whole statement under WithFullSQLInSpanName,
// and with the table under copy_from; both become _OTHER.
func dbSpanName(name string) string {
	switch name {
	case "connect", "pool.acquire", "batch start":
		return name
	}
	return DBSpanName(name)
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
// link tracestate, error.type as the status description, and for otelpgx
// spans only a known name.
type filteredSpan struct {
	sdktrace.ReadOnlySpan
}

// Name keeps every span name except an otelpgx one outside dbSpanName.
func (s filteredSpan) Name() string {
	name := s.ReadOnlySpan.Name()
	if s.InstrumentationScope().Name == otelpgxScope {
		return dbSpanName(name)
	}
	return name
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
// error.type as a bounded token, or empty if it has none. Instrumentation fills the description
// with err.Error(), and a *url.Error, for one, quotes the full URL with its
// query; a PostgreSQL 22P02 quotes the bad input. Only an Error status carries
// a description, as in the OTel API.
func (s filteredSpan) Status() sdktrace.Status {
	st := sdktrace.Status{Code: s.ReadOnlySpan.Status().Code}
	if st.Code == codes.Error {
		for _, kv := range s.ReadOnlySpan.Attributes() {
			if kv.Key == semconv.ErrorTypeKey {
				st.Description = boundAttribute(kv).Value.AsString()
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

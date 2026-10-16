// Package httpobs is the HTTP observability edge for the public listener:
// a server-generated request id, OpenTelemetry tracing and metrics through
// otelhttp, and one access-log line per request.
//
// The listener is a trace trust boundary. otelhttp treats every request as a
// public endpoint (WithPublicEndpointFn returning true), so every request starts a new root span with a
// server-generated trace id, and a valid inbound traceparent is only linked.
// The propagator is TraceContext only, so inbound baggage is dropped.
package httpobs

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/requestid"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
)

// unmatchedRoute is what the access log shows when no route matched. Raw
// paths are never logged as the route: they are unbounded and may carry ids.
const unmatchedRoute = "unmatched"

// Middleware wraps mux, outermost first:
//
//  1. request id: generated, put in the context and echoed in X-Request-Id;
//  2. otelhttp: new root span (inbound traceparent linked), server metrics;
//  3. access: route attribute on the span and the access-log line.
//
// Nothing between otelhttp and mux may replace the *http.Request, because
// otelhttp reads r.Pattern from the request it handed down.
func Middleware(log *slog.Logger, tel *telemetry.Telemetry, mux *http.ServeMux) http.Handler {
	traced := otelhttp.NewHandler(access(log, tel.Propagator, mux), "nava-core",
		otelhttp.WithTracerProvider(tel.TracerProvider),
		otelhttp.WithMeterProvider(tel.MeterProvider),
		otelhttp.WithPropagators(tel.Propagator),
		// server.address is our name, never the client's Host header.
		otelhttp.WithServerName("nava-core"),
		// Every request on this listener is public. This is what the removed
		// otelhttp.WithPublicEndpoint() did.
		otelhttp.WithPublicEndpointFn(func(*http.Request) bool { return true }),
	)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := requestid.New()
		w.Header().Set(requestid.Header, id)
		traced.ServeHTTP(w, r.WithContext(requestid.NewContext(r.Context(), id)))
	})
}

func access(log *slog.Logger, prop propagation.TextMapPropagator, mux *http.ServeMux) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w}
		mux.ServeHTTP(rec, r)

		ctx := r.Context()
		route := unmatchedRoute
		if r.Pattern != "" { // ServeMux set it on this same request
			route = r.Pattern
			// otelhttp names the span after routing but sets http.route only
			// on metrics; the span gets it here. Unmatched requests get none.
			trace.SpanFromContext(ctx).SetAttributes(semconv.HTTPRoute(routePath(r.Pattern)))
		}

		attrs := []slog.Attr{
			slog.String("method", methodLabel(r.Method)),
			slog.String("route", route),
			slog.Int("status", rec.statusCode()),
			slog.Duration("duration", time.Since(start)),
		}
		if in := inboundSpanContext(prop, r.Header); in.IsValid() {
			attrs = append(attrs, slog.String("inbound_trace_id", in.TraceID().String()))
		}
		log.LogAttrs(ctx, slog.LevelInfo, "http request", attrs...)
	})
}

// methodLabel is the method as the metrics record it: a standard method, or
// _OTHER. net/http accepts any token as a method, so the raw value is a
// client-chosen string and never goes into a log line.
func methodLabel(m string) string {
	switch m = strings.ToUpper(m); m {
	case http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut, http.MethodPatch,
		http.MethodDelete, http.MethodOptions, http.MethodConnect, http.MethodTrace:
		return m
	default:
		return "_OTHER"
	}
}

// inboundSpanContext decodes the caller's traceparent, the one otelhttp
// linked. It is logged for correlation and never becomes the parent.
func inboundSpanContext(prop propagation.TextMapPropagator, h http.Header) trace.SpanContext {
	return trace.SpanContextFromContext(prop.Extract(context.Background(), propagation.HeaderCarrier(h)))
}

// routePath strips the method from a ServeMux pattern ("GET /items/{id}" →
// "/items/{id}"), as otelhttp does for the metric's http.route.
func routePath(pattern string) string {
	for i := range len(pattern) {
		if pattern[i] == '/' {
			return pattern[i:]
		}
	}
	return pattern
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	if s.status == 0 {
		s.status = code
	}
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusRecorder) Write(b []byte) (int, error) {
	if s.status == 0 {
		s.status = http.StatusOK
	}
	return s.ResponseWriter.Write(b)
}

// Unwrap lets http.ResponseController reach the underlying writer.
func (s *statusRecorder) Unwrap() http.ResponseWriter { return s.ResponseWriter }

func (s *statusRecorder) statusCode() int {
	if s.status == 0 {
		return http.StatusOK
	}
	return s.status
}

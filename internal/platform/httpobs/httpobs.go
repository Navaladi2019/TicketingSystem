// Package httpobs is the HTTP observability middleware: a request id, the W3C
// trace context, a request-scoped logger, one access-log line per request and
// RED metrics. It wraps a *http.ServeMux so it can label by route pattern.
package httpobs

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/trace"
)

// RequestIDHeader is echoed on every response so a caller can quote it.
const RequestIDHeader = "X-Request-Id"

// unmatchedRoute labels requests no route matched. Raw paths are never used as
// a label value: they are unbounded and may carry ids.
const unmatchedRoute = "unmatched"

// Metrics holds the RED metrics for HTTP requests.
type Metrics struct {
	requests *prometheus.CounterVec
	duration *prometheus.HistogramVec
}

// NewMetrics registers the HTTP metrics on reg. Labels are bounded: method,
// route pattern and status class. Never tenant or user ids.
func NewMetrics(reg prometheus.Registerer) (*Metrics, error) {
	labels := []string{"method", "route", "status_class"}
	m := &Metrics{
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "HTTP requests completed, by method, route pattern and status class.",
		}, labels),
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latency, by method, route pattern and status class.",
			Buckets: prometheus.DefBuckets,
		}, labels),
	}
	for _, c := range []prometheus.Collector{m.requests, m.duration} {
		if err := reg.Register(c); err != nil {
			return nil, err
		}
	}
	return m, nil
}

// Middleware wraps mux. Each request gets a server-generated request id and a
// span context (continuing an inbound traceparent), both in its context and in
// a derived logger; on completion it logs one line and records the metrics.
func Middleware(log *slog.Logger, m *Metrics, mux *http.ServeMux) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		reqID := newRequestID()
		sc := trace.FromRequest(r.Header.Get(trace.Header))
		reqLog := log.With(
			slog.String("request_id", reqID),
			slog.String("trace_id", sc.TraceIDString()),
			slog.String("span_id", sc.SpanIDString()),
		)

		ctx := trace.WithSpanContext(r.Context(), sc)
		ctx = context.WithValue(ctx, requestIDKey{}, reqID)
		ctx = context.WithValue(ctx, loggerKey{}, reqLog)
		r = r.WithContext(ctx)

		w.Header().Set(RequestIDHeader, reqID)
		rec := &statusRecorder{ResponseWriter: w}
		mux.ServeHTTP(rec, r)

		// ServeMux sets r.Pattern on the request it routed.
		route := r.Pattern
		if route == "" {
			route = unmatchedRoute
		}
		status := rec.statusCode()
		elapsed := time.Since(start)
		class := strconv.Itoa(status/100) + "xx"
		m.requests.WithLabelValues(r.Method, route, class).Inc()
		m.duration.WithLabelValues(r.Method, route, class).Observe(elapsed.Seconds())

		reqLog.LogAttrs(ctx, slog.LevelInfo, "http request",
			slog.String("method", r.Method),
			slog.String("route", route),
			slog.Int("status", status),
			slog.Duration("duration", elapsed),
		)
	})
}

type (
	requestIDKey struct{}
	loggerKey    struct{}
)

// RequestID returns the request id in ctx, or "" outside a request.
func RequestID(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}

// Logger returns the request-scoped logger in ctx, or fallback outside a
// request. Handlers log through it so every line carries the request ids.
func Logger(ctx context.Context, fallback *slog.Logger) *slog.Logger {
	if l, ok := ctx.Value(loggerKey{}).(*slog.Logger); ok {
		return l
	}
	return fallback
}

func newRequestID() string {
	var b [16]byte
	_, _ = rand.Read(b[:]) // crypto/rand.Read never returns an error (Go ≥ 1.24)
	return hex.EncodeToString(b[:])
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

package telemetry_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
)

func load(t *testing.T, env map[string]string) config.Telemetry {
	t.Helper()
	cfg, err := config.Load(func(k string) (string, bool) { v, ok := env[k]; return v, ok })
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	return cfg.Telemetry
}

func TestNoEndpointBuildsNoExporter(t *testing.T) {
	t.Parallel()
	tel, err := telemetry.New(t.Context(), load(t, nil))
	if err != nil {
		t.Fatal(err)
	}
	if tel.Exporting() {
		t.Error("a trace exporter was built with no OTLP endpoint")
	}
	if err := tel.Shutdown(t.Context()); err != nil {
		t.Errorf("Shutdown: %v", err)
	}
}

func TestEitherEndpointBuildsExporter(t *testing.T) {
	t.Parallel()
	for _, key := range []string{"OTEL_EXPORTER_OTLP_ENDPOINT", "OTEL_EXPORTER_OTLP_TRACES_ENDPOINT"} {
		// Building the OTLP/HTTP exporter does not dial; nothing listens here.
		tel, err := telemetry.New(t.Context(), load(t, map[string]string{key: "http://127.0.0.1:4318"}))
		if err != nil {
			t.Fatalf("%s: %v", key, err)
		}
		if !tel.Exporting() {
			t.Errorf("no trace exporter with only %s set", key)
		}
		if err := tel.Shutdown(t.Context()); err != nil {
			t.Errorf("%s: Shutdown with no spans to flush: %v", key, err)
		}
	}
}

// TestExportFilter drives every path data reaches a span by: start options,
// SetAttributes, a child tracer, RecordError, a link and the status. The
// exported spans must carry none of the denied values.
func TestExportFilter(t *testing.T) {
	t.Parallel()
	exp := tracetest.NewInMemoryExporter()
	tel, err := telemetry.New(t.Context(), load(t, nil), telemetry.WithSpanExporter(exp))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tel.Shutdown(t.Context()) })

	const secret = "token=s3cr3t"
	tracer := tel.TracerProvider.Tracer("test")
	ctx, span := tracer.Start(t.Context(), "op",
		trace.WithAttributes(
			attribute.String("url.path", "/users/42"),
			attribute.String("http.route", "/users/{id}"),
			attribute.String("client.address", "203.0.113.7"),
		),
		trace.WithLinks(trace.Link{
			SpanContext: span0(),
			Attributes:  []attribute.KeyValue{attribute.String("url.full", "https://x/y?"+secret)},
		}),
	)
	span.SetAttributes(
		attribute.String("url.query", secret),
		attribute.String("user_agent.original", "curl"),
		attribute.String("http.request.header.authorization", "Bearer x"),
		attribute.String("db.query.text", "SELECT * FROM users WHERE email = 'a@b.test'"),
		attribute.String("app.dropped", "yes"), // not on the allow-list
	)
	// What otelhttp.Transport does with a client error: the *url.Error text
	// quotes the full URL, query included.
	clientErr := &url.Error{Op: "Get", URL: "https://api.test/users/42?" + secret, Err: errors.New("timeout")}
	span.RecordError(clientErr)
	span.SetStatus(codes.Error, clientErr.Error())

	_, child := tracer.Start(ctx, "child", trace.WithAttributes(attribute.String("url.full", "https://x/y?"+secret)))
	child.End()
	span.End()

	spans := exp.GetSpans()
	if len(spans) != 2 {
		t.Fatalf("exported %d spans, want 2", len(spans))
	}
	for _, s := range spans {
		for _, kv := range s.Attributes {
			switch kv.Key {
			case "http.route":
			default:
				t.Errorf("span %s kept %s = %q", s.Name, kv.Key, kv.Value.String())
			}
		}
		for _, ev := range s.Events {
			for _, kv := range ev.Attributes {
				if strings.Contains(kv.Value.String(), secret) {
					t.Errorf("span %s event %s leaks %s = %q", s.Name, ev.Name, kv.Key, kv.Value.String())
				}
			}
		}
		for _, l := range s.Links {
			if len(l.Attributes) != 0 {
				t.Errorf("span %s link kept %v", s.Name, l.Attributes)
			}
		}
		if strings.Contains(s.Status.Description, secret) {
			t.Errorf("span %s status description leaks the URL: %q", s.Name, s.Status.Description)
		}
	}
	if parent := spans[1]; parent.Status.Code != codes.Error {
		t.Errorf("status code = %v, want Error kept", parent.Status.Code)
	}
}

// The resource tests set process environment variables, so they do not run
// in parallel.

func TestResourceReadsStandardEnv(t *testing.T) {
	t.Setenv("OTEL_SERVICE_NAME", "nava-core-eu")
	t.Setenv("OTEL_RESOURCE_ATTRIBUTES", "deployment.environment.name=staging")
	exp := tracetest.NewInMemoryExporter()
	tel, err := telemetry.New(t.Context(), load(t, nil), telemetry.WithSpanExporter(exp))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tel.Shutdown(t.Context()) })
	_, span := tel.TracerProvider.Tracer("test").Start(t.Context(), "op")
	span.End()

	res := exp.GetSpans()[0].Resource
	got := map[attribute.Key]string{}
	for _, kv := range res.Attributes() {
		got[kv.Key] = kv.Value.String()
	}
	for k, want := range map[attribute.Key]string{
		"service.name":                "nava-core-eu",
		"deployment.environment.name": "staging",
		"telemetry.sdk.language":      "go",
	} {
		if got[k] != want {
			t.Errorf("resource %s = %q, want %q", k, got[k], want)
		}
	}
}

func TestInvalidResourceEnvNeverEchoesValues(t *testing.T) {
	t.Setenv("OTEL_RESOURCE_ATTRIBUTES", "s3cr3t-without-equals")
	_, err := telemetry.New(t.Context(), load(t, nil))
	if err == nil {
		t.Fatal("New: want error for malformed OTEL_RESOURCE_ATTRIBUTES")
	}
	if !strings.Contains(err.Error(), "OTEL_RESOURCE_ATTRIBUTES") || strings.Contains(err.Error(), "s3cr3t") {
		t.Errorf("error = %q, want it to name the variable and not echo the value", err)
	}
}

// TestRouteErrors changes OTel's process-wide error handler, so it does not
// run in parallel, and restores a silent one afterwards.
func TestRouteErrors(t *testing.T) {
	var buf bytes.Buffer
	telemetry.RouteErrors(slog.New(slog.NewJSONHandler(&buf, nil)))
	t.Cleanup(func() { telemetry.RouteErrors(slog.New(slog.DiscardHandler)) })

	otel.Handle(errors.New("traces export: connection refused"))
	if got := buf.String(); !strings.Contains(got, `"msg":"opentelemetry error"`) ||
		!strings.Contains(got, "connection refused") || !strings.Contains(got, `"level":"WARN"`) {
		t.Errorf("OTel error not routed to slog as a warning: %q", got)
	}
}

// TestValidHeadersNeverReachTheLog builds the real OTLP exporter, which reads
// the header variables from the process environment, so it uses t.Setenv and
// does not run in parallel. Once config.Load has accepted the headers, the
// exporter has nothing to report, so their values never reach the log.
func TestValidHeadersNeverReachTheLog(t *testing.T) {
	const secret = "s3cr3t-token"
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://127.0.0.1:1")
	t.Setenv("OTEL_EXPORTER_OTLP_HEADERS", "Authorization=Bearer%20"+secret+",x-tenant=a")
	t.Setenv("OTEL_EXPORTER_OTLP_TRACES_HEADERS", "api-key="+secret)
	t.Setenv("OTEL_EXPORTER_OTLP_TIMEOUT", "1000")

	var buf bytes.Buffer
	telemetry.RouteErrors(slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { telemetry.RouteErrors(slog.New(slog.DiscardHandler)) })

	cfg, err := config.Load(os.LookupEnv)
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	tel, err := telemetry.New(t.Context(), cfg.Telemetry)
	if err != nil {
		t.Fatal(err)
	}
	if !tel.Exporting() {
		t.Fatal("no trace exporter was built with an OTLP endpoint set")
	}
	if err := tel.Shutdown(t.Context()); err != nil {
		t.Errorf("Shutdown: %v", err)
	}

	// The route is live, so an absent secret means nothing logged it.
	otel.Handle(errors.New("route-check"))
	got := buf.String()
	if !strings.Contains(got, "route-check") {
		t.Fatalf("OTel errors are not reaching the buffer: %q", got)
	}
	if strings.Contains(got, secret) {
		t.Errorf("a header value reached the log: %q", got)
	}
}

func span0() trace.SpanContext {
	return trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: trace.TraceID{1},
		SpanID:  trace.SpanID{1},
	})
}

type logRecorder struct {
	mu   sync.Mutex
	recs []sdklog.Record
}

func (r *logRecorder) Export(_ context.Context, recs []sdklog.Record) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, rec := range recs {
		r.recs = append(r.recs, rec.Clone())
	}
	return nil
}
func (r *logRecorder) Shutdown(context.Context) error   { return nil }
func (r *logRecorder) ForceFlush(context.Context) error { return nil }

func TestLogHandlerOnlyWithBaseEndpoint(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		env  map[string]string
		want bool
	}{
		{nil, false},
		{map[string]string{"OTEL_EXPORTER_OTLP_TRACES_ENDPOINT": "http://127.0.0.1:4318"}, false},
		{map[string]string{"OTEL_EXPORTER_OTLP_ENDPOINT": "http://127.0.0.1:4318"}, true},
	} {
		tel, err := telemetry.New(t.Context(), load(t, tc.env))
		if err != nil {
			t.Fatal(err)
		}
		if got := tel.LogHandler() != nil; got != tc.want {
			t.Errorf("env %v: LogHandler non-nil = %v, want %v", tc.env, got, tc.want)
		}
		if err := tel.Shutdown(t.Context()); err != nil {
			t.Errorf("Shutdown: %v", err)
		}
	}
}

// A record logged with the request ctx carries that request's trace and span
// id as record fields; one logged without it carries none.
func TestExportedLogRecordCarriesTraceID(t *testing.T) {
	t.Parallel()
	rec := &logRecorder{}
	tel, err := telemetry.New(t.Context(), load(t, nil), telemetry.WithLogExporter(rec))
	if err != nil {
		t.Fatal(err)
	}
	log := slog.New(tel.LogHandler())

	ctx, span := tel.TracerProvider.Tracer("test").Start(t.Context(), "req")
	log.InfoContext(ctx, "with ctx", slog.String("k", "v"))
	log.InfoContext(t.Context(), "without ctx")
	span.End()

	if len(rec.recs) != 2 {
		t.Fatalf("exported %d records, want 2", len(rec.recs))
	}
	sc := span.SpanContext()
	if got := rec.recs[0]; got.TraceID() != sc.TraceID() || got.SpanID() != sc.SpanID() {
		t.Errorf("trace/span = %s/%s, want %s/%s", got.TraceID(), got.SpanID(), sc.TraceID(), sc.SpanID())
	}
	if got := rec.recs[1]; got.TraceID().IsValid() {
		t.Errorf("record logged without ctx has trace id %s", got.TraceID())
	}
	if err := tel.Shutdown(t.Context()); err != nil {
		t.Errorf("Shutdown: %v", err)
	}
}

// A denied key is removed before export, at the top level, inside a
// slog.Group, and in attributes bound with With and WithGroup.
func TestExportedLogRecordDropsDeniedAttributes(t *testing.T) {
	t.Parallel()
	rec := &logRecorder{}
	tel, err := telemetry.New(t.Context(), load(t, nil), telemetry.WithLogExporter(rec))
	if err != nil {
		t.Fatal(err)
	}
	log := slog.New(tel.LogHandler())

	log.InfoContext(t.Context(), "m",
		slog.String("url.path", "/secret"),
		slog.String("keep", "ok"),
		slog.Group("http",
			slog.String("url.query", "token=x"),
			slog.String("http.request.header.cookie", "c"),
			slog.Group("deep", slog.String("exception.message", "boom"), slog.String("ok", "1"))),
		slog.Group("only", slog.String("db.statement", "select 1")))
	log.With(slog.String("client.address", "10.0.0.1")).WithGroup("g").
		InfoContext(t.Context(), "m2", slog.String("server.port", "1"), slog.String("fine", "2"))

	if len(rec.recs) != 2 {
		t.Fatalf("exported %d records, want 2", len(rec.recs))
	}
	for i := range rec.recs {
		r := rec.recs[i]
		body := ""
		r.WalkAttributes(func(kv attribute.KeyValue) bool { body += " " + string(kv.Key) + "=" + kv.Value.Emit(); return true })
		for _, bad := range []string{"url.path", "url.query", "header.cookie", "exception.message", "db.statement", "client.address", "server.port", "only"} {
			if strings.Contains(body, bad) {
				t.Errorf("record %d still carries %q: %s", i, bad, body)
			}
		}
		for _, good := range []string{"keep", "ok", "fine"} {
			if i == 0 && good == "fine" || i == 1 && good != "fine" {
				continue
			}
			if !strings.Contains(body, good) {
				t.Errorf("record %d lost %q: %s", i, good, body)
			}
		}
	}
}

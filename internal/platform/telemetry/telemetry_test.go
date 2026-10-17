package telemetry_test

import (
	"bytes"
	"errors"
	"log/slog"
	"net/url"
	"strings"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
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
		attribute.String("app.kept", "yes"),
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
			case "http.route", "app.kept":
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

func span0() trace.SpanContext {
	return trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: trace.TraceID{1},
		SpanID:  trace.SpanID{1},
	})
}

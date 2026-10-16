package telemetry_test

import (
	"testing"

	"go.opentelemetry.io/otel/attribute"
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
		t.Error("a trace exporter was built with no OTEL_EXPORTER_OTLP_ENDPOINT")
	}
	if err := tel.Shutdown(t.Context()); err != nil {
		t.Errorf("Shutdown: %v", err)
	}
}

func TestEndpointBuildsExporter(t *testing.T) {
	t.Parallel()
	// Building the OTLP/HTTP exporter does not dial; nothing listens here.
	tel, err := telemetry.New(t.Context(), load(t, map[string]string{"OTEL_EXPORTER_OTLP_ENDPOINT": "http://127.0.0.1:4318"}))
	if err != nil {
		t.Fatal(err)
	}
	if !tel.Exporting() {
		t.Error("no trace exporter with OTEL_EXPORTER_OTLP_ENDPOINT set")
	}
	if err := tel.Shutdown(t.Context()); err != nil {
		t.Errorf("Shutdown with no spans to flush: %v", err)
	}
}

func TestEverySamplerNameBuilds(t *testing.T) {
	t.Parallel()
	for _, name := range []string{
		config.SamplerAlwaysOn, config.SamplerAlwaysOff, config.SamplerTraceIDRatio,
		config.SamplerParentBasedAlwaysOn, config.SamplerParentBasedAlwaysOff, config.SamplerParentBasedTraceIDRatio,
	} {
		tel, err := telemetry.New(t.Context(), load(t, map[string]string{"OTEL_TRACES_SAMPLER": name}))
		if err != nil {
			t.Errorf("sampler %s: %v", name, err)
			continue
		}
		_ = tel.Shutdown(t.Context())
	}
}

func TestSpanAttributeFilter(t *testing.T) {
	t.Parallel()
	rec := tracetest.NewSpanRecorder()
	tel, err := telemetry.New(t.Context(), load(t, nil), telemetry.WithSpanProcessor(rec))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tel.Shutdown(t.Context()) })

	tracer := tel.TracerProvider.Tracer("test")
	_, span := tracer.Start(t.Context(), "op", traceAttrs(
		"url.path", "/users/42", "http.route", "/users/{id}", "client.address", "203.0.113.7"))
	span.SetAttributes(
		attribute.String("url.query", "email=a@b.test"),
		attribute.String("user_agent.original", "curl"),
		attribute.String("http.request.header.authorization", "Bearer x"),
		attribute.String("app.kept", "yes"),
	)
	// A tracer reached through the span's provider is filtered too.
	_, child := span.TracerProvider().Tracer("child").Start(t.Context(), "child", traceAttrs("url.full", "https://x/y?z"))
	child.End()
	span.End()

	ended := rec.Ended()
	if len(ended) != 2 {
		t.Fatalf("recorded %d spans, want 2", len(ended))
	}
	for _, s := range ended {
		for _, kv := range s.Attributes() {
			switch kv.Key {
			case "http.route", "app.kept":
			default:
				t.Errorf("span %s kept %s = %q", s.Name(), kv.Key, kv.Value.String())
			}
		}
	}
}

func traceAttrs(kv ...string) trace.SpanStartEventOption {
	attrs := make([]attribute.KeyValue, 0, len(kv)/2)
	for i := 0; i+1 < len(kv); i += 2 {
		attrs = append(attrs, attribute.String(kv[i], kv[i+1]))
	}
	return trace.WithAttributes(attrs...)
}

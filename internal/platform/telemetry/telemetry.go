// Package telemetry builds the OpenTelemetry providers for nava-core: an SDK
// tracer provider, a meter provider exported as Prometheus on our own
// registry, and the edge propagator. Nothing here is global. There is no
// otel.Set*; callers pass the providers to the instrumentation explicitly.
package telemetry

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
)

// Telemetry holds the providers. Build it once in main and Shutdown it during
// graceful shutdown.
type Telemetry struct {
	// Registry is the Prometheus registry /metrics serves. It holds the OTel
	// exporter's metrics and the process collector, and nothing else.
	Registry *prometheus.Registry
	// TracerProvider is the SDK provider behind the span attribute filter.
	TracerProvider trace.TracerProvider
	// MeterProvider exports through Registry.
	MeterProvider metric.MeterProvider
	// Propagator is the edge propagator: W3C TraceContext only. There is no
	// Baggage propagator, so inbound baggage never reaches a request context.
	Propagator propagation.TextMapPropagator

	tp        *sdktrace.TracerProvider
	mp        *sdkmetric.MeterProvider
	exporting bool
}

// Option adjusts New. Tests use it to record spans in memory.
type Option func(*options)

type options struct {
	spanProcessors []sdktrace.SpanProcessor
}

// WithSpanProcessor adds a span processor, such as tracetest.SpanRecorder.
func WithSpanProcessor(sp sdktrace.SpanProcessor) Option {
	return func(o *options) { o.spanProcessors = append(o.spanProcessors, sp) }
}

// New builds the providers from cfg. With no OTLP endpoint there is no trace
// exporter and nothing dials out; spans are still created, so every request
// has a trace id for its logs.
func New(ctx context.Context, cfg config.Telemetry, opts ...Option) (*Telemetry, error) {
	var o options
	for _, opt := range opts {
		opt(&o)
	}

	res := resource.NewWithAttributes(semconv.SchemaURL,
		semconv.ServiceName(cfg.ServiceName),
		semconv.ServiceVersion(version()),
	)

	sampler, err := newSampler(cfg.Sampler, cfg.SamplerArg)
	if err != nil {
		return nil, err
	}
	tpOpts := []sdktrace.TracerProviderOption{sdktrace.WithResource(res), sdktrace.WithSampler(sampler)}
	exporting := cfg.OTLPEndpoint != ""
	if exporting {
		exp, err := otlptracehttp.New(ctx,
			otlptracehttp.WithEndpointURL(strings.TrimRight(cfg.OTLPEndpoint, "/")+"/v1/traces"))
		if err != nil {
			return nil, fmt.Errorf("telemetry: otlp trace exporter: %w", err)
		}
		tpOpts = append(tpOpts, sdktrace.WithBatcher(exp))
	}
	for _, sp := range o.spanProcessors {
		tpOpts = append(tpOpts, sdktrace.WithSpanProcessor(sp))
	}
	tp := sdktrace.NewTracerProvider(tpOpts...)

	reg := prometheus.NewRegistry()
	if err := reg.Register(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{})); err != nil {
		return nil, fmt.Errorf("telemetry: process collector: %w", err)
	}
	reader, err := otelprom.New(otelprom.WithRegisterer(reg))
	if err != nil {
		return nil, fmt.Errorf("telemetry: prometheus exporter: %w", err)
	}
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(reader),
		sdkmetric.WithView(httpServerMetricsView()),
	)
	if err := runtime.Start(runtime.WithMeterProvider(mp)); err != nil {
		return nil, fmt.Errorf("telemetry: runtime metrics: %w", err)
	}

	return &Telemetry{
		Registry:       reg,
		TracerProvider: filteringTracerProvider{TracerProvider: tp},
		MeterProvider:  mp,
		Propagator:     propagation.TraceContext{},
		tp:             tp,
		mp:             mp,
		exporting:      exporting,
	}, nil
}

// Exporting reports whether a trace exporter was built.
func (t *Telemetry) Exporting() bool { return t.exporting }

// Shutdown flushes and stops both providers.
func (t *Telemetry) Shutdown(ctx context.Context) error {
	return errors.Join(t.tp.Shutdown(ctx), t.mp.Shutdown(ctx))
}

func newSampler(name string, ratio float64) (sdktrace.Sampler, error) {
	switch name {
	case config.SamplerAlwaysOn:
		return sdktrace.AlwaysSample(), nil
	case config.SamplerAlwaysOff:
		return sdktrace.NeverSample(), nil
	case config.SamplerTraceIDRatio:
		return sdktrace.TraceIDRatioBased(ratio), nil
	case config.SamplerParentBasedAlwaysOn:
		return sdktrace.ParentBased(sdktrace.AlwaysSample()), nil
	case config.SamplerParentBasedAlwaysOff:
		return sdktrace.ParentBased(sdktrace.NeverSample()), nil
	case config.SamplerParentBasedTraceIDRatio:
		return sdktrace.ParentBased(sdktrace.TraceIDRatioBased(ratio)), nil
	default:
		return nil, fmt.Errorf("telemetry: unknown sampler %q", name)
	}
}

// version is the module version from the build, or "(devel)" for a local build.
func version() string {
	if bi, ok := debug.ReadBuildInfo(); ok && bi.Main.Version != "" {
		return bi.Main.Version
	}
	return "(devel)"
}

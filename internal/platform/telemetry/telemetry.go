// Package telemetry builds the OpenTelemetry providers for nava-core: an SDK
// tracer provider, a meter provider exported as Prometheus on our own
// registry, and the edge propagator. Callers pass the providers to the
// instrumentation explicitly. The one global is where OTel reports its own
// errors (RouteErrors).
//
// The SDK and the OTLP exporter read their standard OTEL_* variables
// themselves: OTEL_SERVICE_NAME, OTEL_RESOURCE_ATTRIBUTES, OTEL_TRACES_SAMPLER
// and _ARG, OTEL_BSP_*, the span limits, and every OTEL_EXPORTER_OTLP_*
// transport setting. config decides only whether a trace exporter exists.
package telemetry

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime/debug"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
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
	// TracerProvider is the SDK provider. Every exporter behind it applies the
	// span attribute allow-list.
	TracerProvider trace.TracerProvider
	// MeterProvider exports through Registry.
	MeterProvider metric.MeterProvider
	// Propagator is the edge propagator: W3C TraceContext only. There is no
	// Baggage propagator, so inbound baggage never reaches a request context.
	Propagator propagation.TextMapPropagator

	tp        *sdktrace.TracerProvider
	mp        *sdkmetric.MeterProvider
	lp        *sdklog.LoggerProvider
	exporting bool
}

// Option adjusts New. Tests use them to export spans in memory and to fix the
// sampler without touching the process environment.
type Option func(*options)

type options struct {
	exporters    []sdktrace.SpanExporter
	logExporters []sdklog.Exporter
	sampler      sdktrace.Sampler
}

// WithLogExporter adds a synchronous log exporter, so a test sees the records
// the OTLP exporter would send. Unlike the OTLP exporter it does not need
// cfg.ExportLogs.
func WithLogExporter(exp sdklog.Exporter) Option {
	return func(o *options) { o.logExporters = append(o.logExporters, exp) }
}

// WithSpanExporter adds a synchronous exporter, such as
// tracetest.InMemoryExporter. It sees spans exactly as the OTLP exporter
// would: sampled, ended and filtered.
func WithSpanExporter(exp sdktrace.SpanExporter) Option {
	return func(o *options) { o.exporters = append(o.exporters, exp) }
}

// WithSampler overrides OTEL_TRACES_SAMPLER.
func WithSampler(s sdktrace.Sampler) Option {
	return func(o *options) { o.sampler = s }
}

// New builds the providers. With cfg.ExportTraces false there is no trace
// exporter and nothing dials out; spans are still created, so every request
// has a trace id for its logs.
func New(ctx context.Context, cfg config.Telemetry, opts ...Option) (_ *Telemetry, err error) {
	var o options
	for _, opt := range opts {
		opt(&o)
	}

	// Later options win: OTEL_SERVICE_NAME and OTEL_RESOURCE_ATTRIBUTES
	// override the defaults.
	res, err := resource.New(ctx,
		resource.WithTelemetrySDK(),
		resource.WithAttributes(semconv.ServiceName("nava-core"), semconv.ServiceVersion(version())),
		resource.WithFromEnv(),
	)
	if err != nil {
		// The detector's error quotes the variable's content; name it only.
		return nil, errors.New("telemetry: OTEL_SERVICE_NAME or OTEL_RESOURCE_ATTRIBUTES is invalid")
	}

	// Whatever is built is shut down again if a later step fails.
	var built []func(context.Context) error
	defer func() {
		if err != nil {
			for _, shutdown := range built {
				_ = shutdown(ctx)
			}
		}
	}()

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
	built = append(built, mp.Shutdown)
	if err := runtime.Start(runtime.WithMeterProvider(mp)); err != nil {
		return nil, fmt.Errorf("telemetry: runtime metrics: %w", err)
	}

	tpOpts := []sdktrace.TracerProviderOption{sdktrace.WithResource(res)}
	if o.sampler != nil {
		tpOpts = append(tpOpts, sdktrace.WithSampler(o.sampler))
	}
	if cfg.ExportTraces {
		// No endpoint option: the exporter resolves OTEL_EXPORTER_OTLP_TRACES_ENDPOINT,
		// then OTEL_EXPORTER_OTLP_ENDPOINT + /v1/traces, as the spec says.
		exp, err := otlptracehttp.New(ctx)
		if err != nil {
			return nil, fmt.Errorf("telemetry: otlp trace exporter: %w", err)
		}
		tpOpts = append(tpOpts, sdktrace.WithBatcher(filteringExporter{next: exp}))
	}
	for _, exp := range o.exporters {
		tpOpts = append(tpOpts, sdktrace.WithSyncer(filteringExporter{next: exp}))
	}
	tp := sdktrace.NewTracerProvider(tpOpts...)
	built = append(built, tp.Shutdown)

	// Log export is off unless OTEL_EXPORTER_OTLP_ENDPOINT is set. The log
	// provider shares res, so records carry the same service.name as spans.
	var lp *sdklog.LoggerProvider
	if cfg.ExportLogs || len(o.logExporters) > 0 {
		lpOpts := []sdklog.LoggerProviderOption{sdklog.WithResource(res)}
		if cfg.ExportLogs {
			// No endpoint option: the exporter resolves OTEL_EXPORTER_OTLP_LOGS_ENDPOINT,
			// then OTEL_EXPORTER_OTLP_ENDPOINT + /v1/logs.
			exp, err := otlploghttp.New(ctx)
			if err != nil {
				return nil, fmt.Errorf("telemetry: otlp log exporter: %w", err)
			}
			lpOpts = append(lpOpts, sdklog.WithProcessor(sdklog.NewBatchProcessor(exp)))
		}
		for _, exp := range o.logExporters {
			lpOpts = append(lpOpts, sdklog.WithProcessor(sdklog.NewSimpleProcessor(exp)))
		}
		lp = sdklog.NewLoggerProvider(lpOpts...)
	}

	return &Telemetry{
		Registry:       reg,
		TracerProvider: tp,
		MeterProvider:  mp,
		Propagator:     propagation.TraceContext{},
		tp:             tp,
		mp:             mp,
		lp:             lp,
		exporting:      cfg.ExportTraces,
	}, nil
}

// Exporting reports whether a trace exporter was built.
func (t *Telemetry) Exporting() bool { return t.exporting }

// LogHandler is the slog handler that exports records over OTLP, or nil when
// log export is off. It takes the trace and span id from the ctx passed to
// the *Context logging methods, so call sites must log with the request ctx.
func (t *Telemetry) LogHandler() slog.Handler {
	if t.lp == nil {
		return nil
	}
	return otelslog.NewHandler("nava-core", otelslog.WithLoggerProvider(t.lp))
}

// Shutdown flushes and stops the providers.
func (t *Telemetry) Shutdown(ctx context.Context) error {
	errs := []error{t.tp.Shutdown(ctx), t.mp.Shutdown(ctx)}
	if t.lp != nil {
		errs = append(errs, t.lp.Shutdown(ctx))
	}
	return errors.Join(errs...)
}

// RouteErrors sends OTel's own errors and internal log lines (failed exports,
// an invalid OTEL_* value) to log, instead of the default plain-text stderr
// logger. Handled errors arrive as "opentelemetry error" warnings; internal
// lines keep their own message and fields, which can quote a variable's value,
// so config.Load rejects the OTLP values the exporter would log. It sets OTel's
// global error handler and logger: process-wide, so main calls it once, before
// New. It changes no instrumentation.
func RouteErrors(log *slog.Logger) {
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		log.WarnContext(context.Background(), "opentelemetry error", slog.Any("error", err))
	}))
	otel.SetLogger(logr.FromSlogHandler(log.Handler()))
}

// version is the module version from the build, or "(devel)" for a local build.
func version() string {
	if bi, ok := debug.ReadBuildInfo(); ok && bi.Main.Version != "" {
		return bi.Main.Version
	}
	return "(devel)"
}

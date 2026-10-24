package telemetry

import (
	"context"
	"errors"
	"log/slog"

	"go.opentelemetry.io/otel/propagation"
	"go.temporal.io/sdk/client"
	temporalotel "go.temporal.io/sdk/contrib/opentelemetry"
	"go.temporal.io/sdk/interceptor"
)

// temporalScope is the instrumentation scope of the Temporal SDK's spans and
// metrics.
const temporalScope = "temporal-sdk-go"

// Temporal is the Temporal SDK's telemetry, built once per process from the
// same providers as everything else. The relay and the worker both use it
// (TICK-94 item 5).
type Temporal struct {
	// Tracing is the tracing interceptor. Put it first in
	// client.Options.Interceptors: the SDK puts client interceptors before
	// worker ones, so a worker guard's error lands on its span.
	Tracing interceptor.Interceptor
	// Metrics is the SDK metrics handler, for client.Options.MetricsHandler.
	Metrics client.MetricsHandler
}

// NewTemporal builds the Temporal SDK telemetry on t's providers. Each setting
// below is mandatory (TICK-94 item 5):
//   - the Tracer comes from t, so SDK spans go through the span allow-list;
//   - TextMapPropagator is W3C TraceContext only;
//   - DisableBaggage is true, so no baggage enters or leaves a workflow;
//   - the metrics handler's Meter comes from t, and OnError logs instead of
//     panicking (the contrib default panics).
//
// Spans carry the workflow and run id, which the allow-list keeps; everything
// else the SDK records (activity id, update id, error messages) is dropped or
// bounded at export.
func (t *Telemetry) NewTemporal(log *slog.Logger) (*Temporal, error) {
	if t == nil || t.TracerProvider == nil || t.MeterProvider == nil || log == nil {
		return nil, errors.New("telemetry: nil telemetry or logger for the Temporal SDK")
	}
	tracing, err := temporalotel.NewTracingInterceptor(temporalotel.TracerOptions{
		Tracer:            t.TracerProvider.Tracer(temporalScope),
		TextMapPropagator: propagation.TraceContext{},
		DisableBaggage:    true,
	})
	if err != nil {
		return nil, err
	}
	metrics := temporalotel.NewMetricsHandler(temporalotel.MetricsHandlerOptions{
		Meter: t.MeterProvider.Meter(temporalScope),
		OnError: func(err error) {
			log.WarnContext(context.Background(), "temporal sdk metric error", slog.Any("error", err))
		},
		// The SDK's counters only grow; export them as Prometheus counters.
		UseMonotonicCounters: true,
	})
	return &Temporal{Tracing: tracing, Metrics: metrics}, nil
}

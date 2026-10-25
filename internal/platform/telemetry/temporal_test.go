package telemetry

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	tracenoop "go.opentelemetry.io/otel/trace/noop"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
)

func TestNewTemporalRejectsNil(t *testing.T) {
	var nilTel *Telemetry
	if _, err := nilTel.NewTemporal(slog.New(slog.DiscardHandler)); err == nil {
		t.Error("NewTemporal on a nil Telemetry succeeded")
	}
	tel, err := New(t.Context(), config.Telemetry{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tel.Shutdown(context.Background()) })
	if _, err := tel.NewTemporal(nil); err == nil {
		t.Error("NewTemporal with a nil logger succeeded")
	}
}

// The SDK's metrics go to the platform registry, and its counters are
// Prometheus counters.
func TestNewTemporalMetricsOnRegistry(t *testing.T) {
	tel, err := New(t.Context(), config.Telemetry{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tel.Shutdown(context.Background()) })
	tt, err := tel.NewTemporal(slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	if tt.Tracing == nil || tt.Metrics == nil {
		t.Fatal("nil tracing interceptor or metrics handler")
	}
	tt.Metrics.Counter("nava_t31_probe").Inc(2)

	families, err := tel.Registry.Gather()
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range families {
		if f.GetName() == "nava_t31_probe_total" {
			if f.GetType().String() != "COUNTER" {
				t.Errorf("type %v, want COUNTER", f.GetType())
			}
			if v := f.GetMetric()[0].GetCounter().GetValue(); v != 2 {
				t.Errorf("value %v, want 2", v)
			}
			return
		}
	}
	t.Error("nava_t31_probe_total is not on the registry")
}

// TICK-94 item 5: the constructor refuses a nil Tracer or Meter source, so
// the contrib never falls back to the global providers.
func TestNewTemporalRejectsNilProviders(t *testing.T) {
	log := slog.New(slog.DiscardHandler)
	if _, err := (&Telemetry{MeterProvider: metricnoop.NewMeterProvider()}).NewTemporal(log); err == nil {
		t.Error("NewTemporal with a nil TracerProvider succeeded")
	}
	if _, err := (&Telemetry{TracerProvider: tracenoop.NewTracerProvider()}).NewTemporal(log); err == nil {
		t.Error("NewTemporal with a nil MeterProvider succeeded")
	}
}

// failingMeter fails every instrument it is asked for.
type failingMeter struct{ metricnoop.Meter }

func (failingMeter) Int64Counter(string, ...metric.Int64CounterOption) (metric.Int64Counter, error) {
	return nil, errors.New("instrument refused")
}

type failingMeterProvider struct{ metricnoop.MeterProvider }

func (failingMeterProvider) Meter(string, ...metric.MeterOption) metric.Meter { return failingMeter{} }

// TICK-94 item 5: a metrics error reaches OnError, which logs and does not
// panic (the contrib's default panics).
func TestNewTemporalMetricErrorLogs(t *testing.T) {
	var logs bytes.Buffer
	tel := &Telemetry{TracerProvider: tracenoop.NewTracerProvider(), MeterProvider: failingMeterProvider{}}
	tt, err := tel.NewTemporal(slog.New(slog.NewTextHandler(&logs, nil)))
	if err != nil {
		t.Fatal(err)
	}
	tt.Metrics.Counter("nava_probe").Inc(1)
	if !strings.Contains(logs.String(), "temporal sdk metric error") {
		t.Errorf("OnError did not log:\n%s", logs.String())
	}
}

package telemetry

import (
	"context"
	"log/slog"
	"testing"

	dto "github.com/prometheus/client_model/go"

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
			if f.GetType() != dto.MetricType_COUNTER {
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

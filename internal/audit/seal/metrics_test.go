package seal

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

// scrape returns each sample of the registry as name{reason} -> value.
func scrape(t *testing.T, reg *prometheus.Registry) map[string]float64 {
	t.Helper()
	mfs, err := reg.Gather()
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]float64{}
	for _, mf := range mfs {
		for _, m := range mf.GetMetric() {
			key := mf.GetName()
			for _, l := range m.GetLabel() {
				if l.GetName() == "reason" {
					key += "{" + l.GetValue() + "}"
				}
				if l.GetName() == "tenant_id" || l.GetName() == "tenant" {
					t.Errorf("%s carries a tenant label", mf.GetName())
				}
			}
			switch {
			case m.GetCounter() != nil:
				out[key] = m.GetCounter().GetValue()
			case m.GetGauge() != nil:
				out[key] = m.GetGauge().GetValue()
			}
		}
	}
	return out
}

// The R-A2 metrics reach Prometheus as audit_unsealed_oldest_seconds and
// audit_seal_failures_total, the names the alert rule uses, with no tenant
// label. The gauge is the worst tenant's age now, keeps growing between
// measurements, goes back down when the tenant is sealed, and drops a tenant
// no A2 has measured for LagStaleAfter.
func TestMetrics(t *testing.T) {
	reg := prometheus.NewRegistry()
	reader, err := otelprom.New(otelprom.WithRegisterer(reg))
	if err != nil {
		t.Fatal(err)
	}
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	t.Cleanup(func() { _ = mp.Shutdown(context.Background()) })
	now := time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC)
	m, err := newMetrics(mp, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	if got := scrape(t, reg)["audit_unsealed_oldest_seconds"]; got != 0 {
		t.Errorf("gauge %v with nothing measured, want 0", got)
	}
	stuck, healthy := uuid.New(), uuid.New()
	m.observe(stuck, now.Add(-100*time.Second), true)
	m.observe(healthy, now.Add(-5*time.Second), true)
	m.failed(ctx, fmt.Errorf("%w: entry x: version_after", ErrUnsealable))
	m.failed(ctx, errors.New("connection reset"))
	m.failed(ctx, ErrUnsealable)
	m.rescheduleFailed(ctx)

	got := scrape(t, reg)
	if got["audit_unsealed_oldest_seconds"] != 100 {
		t.Errorf("gauge %v, want 100 (the worst tenant)", got["audit_unsealed_oldest_seconds"])
	}
	if got["audit_seal_failures_total{unsealable}"] != 2 || got["audit_seal_failures_total{error}"] != 1 ||
		got["audit_seal_failures_total{reschedule}"] != 1 {
		t.Errorf("failures %v, want unsealable 2 and error 1", got)
	}

	// Between runs the stuck tenant keeps aging past the 600 s alert.
	now = now.Add(9 * time.Minute)
	if g := scrape(t, reg)["audit_unsealed_oldest_seconds"]; g != 640 {
		t.Errorf("gauge %v after 9 min, want 640", g)
	}
	// Sealed: the tenant leaves the gauge.
	m.observe(stuck, time.Time{}, false)
	if g := scrape(t, reg)["audit_unsealed_oldest_seconds"]; g != 9*60+5 {
		t.Errorf("gauge %v, want the healthy tenant's 545", g)
	}
	// Not measured for LagStaleAfter: dropped.
	now = now.Add(LagStaleAfter)
	if g := scrape(t, reg)["audit_unsealed_oldest_seconds"]; g != 0 {
		t.Errorf("gauge %v with only stale samples, want 0", g)
	}

	var none *Metrics
	none.failed(ctx, ErrUnsealable)
	none.rescheduleFailed(ctx)
	none.observe(stuck, now, true)
}

// The R-A2 alert (deploy/compose/prometheus/rules/audit.yml) fires on the
// exported gauge name at 600 s, and the local Prometheus loads it.
func TestAlertRule(t *testing.T) {
	read := func(p string) string {
		b, err := os.ReadFile(filepath.Join("..", "..", "..", p))
		if err != nil {
			t.Fatal(err)
		}
		return strings.ReplaceAll(string(b), "\r\n", "\n")
	}
	if rule := read("deploy/compose/prometheus/rules/audit.yml"); !strings.Contains(rule,
		"expr: max(audit_unsealed_oldest_seconds) > 600\n") || !strings.Contains(rule, "alert: AuditSealLag\n") {
		t.Errorf("audit.yml has no AuditSealLag on max(audit_unsealed_oldest_seconds) > 600:\n%s", rule)
	}
	if !strings.Contains(read("deploy/compose/prometheus/prometheus.yml"), "rule_files:\n  - /etc/prometheus/rules/*.yml\n") {
		t.Error("prometheus.yml does not load /etc/prometheus/rules/*.yml")
	}
	if !strings.Contains(read("compose.yaml"), "- ./deploy/compose/prometheus/rules:/etc/prometheus/rules:ro\n") {
		t.Error("compose.yaml does not mount the rules directory into prometheus")
	}
}

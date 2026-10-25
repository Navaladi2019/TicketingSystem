package relay

import (
	"testing"

	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

// scannedHistogram is the single outbox_relay_rows_scanned data point in rm.
func scannedHistogram(t *testing.T, rm metricdata.ResourceMetrics) metricdata.HistogramDataPoint[int64] {
	t.Helper()
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != "outbox_relay_rows_scanned" {
				continue
			}
			dps := m.Data.(metricdata.Histogram[int64]).DataPoints
			if len(dps) != 1 {
				t.Fatalf("%d data points, want 1 (no tenant label)", len(dps))
			}
			return dps[0]
		}
	}
	t.Fatal("no outbox_relay_rows_scanned")
	return metricdata.HistogramDataPoint[int64]{}
}

// cumulativeAt is the Prometheus _bucket{le=bound} value of h.
func cumulativeAt(h metricdata.HistogramDataPoint[int64], bound float64) uint64 {
	var n uint64
	for i, b := range h.Bounds {
		n += h.BucketCounts[i]
		if b == bound {
			return n
		}
	}
	return h.Count
}

package relay

import (
	"bytes"
	"sync"
	"testing"
	"time"

	"go.opentelemetry.io/otel/attribute"

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

func attrKey(k string) attribute.Key { return attribute.Key(k) }

func hasAttr(kvs []attribute.KeyValue, k attribute.Key, v string) bool {
	for _, kv := range kvs {
		if kv.Key == k && kv.Value.AsString() == v {
			return true
		}
	}
	return false
}

// syncBuffer is a bytes.Buffer safe for one writer and one reader.
type syncBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

// waitFor polls cond for up to 10 s.
func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal("condition not met within 10 s")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

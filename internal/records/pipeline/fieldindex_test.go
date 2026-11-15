package pipeline

import (
	"context"
	"testing"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

// observe counts only the trigger's shape, a JSON object of entity
// definition id to a positive count, and ignores anything else a session
// could have put in the setting.
func TestFieldIndexObserve(t *testing.T) {
	const a, b = "0190a000-0000-7000-8000-00000000000a", "0190a000-0000-7000-8000-00000000000b"
	reader := sdkmetric.NewManualReader()
	m, err := NewFieldIndexMetrics(sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader)))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // a committed write's context may be cancelled already
	for _, v := range []string{
		"",
		"not json",
		`[1, 2]`,
		`{"` + a + `": "3"}`,
		`{"` + a + `": 1.5}`,
		`{"not-a-uuid": 4, "` + a + `": -2, "` + b + `": 0}`,
		`{"` + a + `": 2, "` + b + `": 1}`,
		`{"` + a + `": 1}`,
	} {
		m.observe(ctx, v)
	}
	var nilMetrics *FieldIndexMetrics
	nilMetrics.watch(nil) // records nothing, needs no transaction

	var rm metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &rm); err != nil {
		t.Fatal(err)
	}
	got := map[string]int64{}
	for _, sm := range rm.ScopeMetrics {
		for _, mt := range sm.Metrics {
			for _, dp := range mt.Data.(metricdata.Sum[int64]).DataPoints {
				def, _ := dp.Attributes.Value("entity_definition")
				got[def.AsString()] += dp.Value
				if dp.Attributes.Len() != 1 {
					t.Errorf("attributes %v, want entity_definition only", dp.Attributes.ToSlice())
				}
			}
		}
	}
	if len(got) != 2 || got[a] != 3 || got[b] != 1 {
		t.Errorf("counts = %v, want {%s: 3, %s: 1}", got, a, b)
	}
}

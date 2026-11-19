//go:build integration

package backfill_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/Navaladi2019/TicketingSystem/internal/fieldindex"
	"github.com/Navaladi2019/TicketingSystem/internal/fieldindex/backfill"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
	"github.com/Navaladi2019/TicketingSystem/internal/workerguard"
	"github.com/Navaladi2019/TicketingSystem/internal/workflow/recordports"
)

// uncoercedTotals sums nava_field_index_uncoerced from reader by
// entity_definition, and fails on any other attribute (R5: no tenant id,
// no value).
func uncoercedTotals(t *testing.T, reader *sdkmetric.ManualReader) map[string]int64 {
	t.Helper()
	var rm metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &rm); err != nil {
		t.Fatal(err)
	}
	out := map[string]int64{}
	for _, sm := range rm.ScopeMetrics {
		for _, mt := range sm.Metrics {
			if mt.Name != "nava_field_index_uncoerced" {
				continue
			}
			for _, dp := range mt.Data.(metricdata.Sum[int64]).DataPoints {
				def, ok := dp.Attributes.Value("entity_definition")
				if !ok || dp.Attributes.Len() != 1 {
					t.Fatalf("nava_field_index_uncoerced has attributes %v, want entity_definition only", dp.Attributes)
				}
				out[def.AsString()] += dp.Value
			}
		}
	}
	return out
}

// TICK-974 (ADR-006 D10, R5): the backfill counts the values it skips
// because they do not coerce, on the trigger's counter, by exactly that
// number, only once committed, and puts nothing new in any payload.
func TestBackfillCountsUncoercedValues(t *testing.T) {
	e := migrated(t)
	f := newFixture(t, e, "num", "")
	// Written before the registry row exists, so the trigger counts nothing:
	// every count below is the backfill's.
	data := []struct {
		json      string
		uncoerced int64
	}{
		{`{"f_num": 1}`, 0},
		{`{"f_num": "fieldvalue-secret-a"}`, 1},
		{`{"f_num": [1, "x", null, true, 2.5]}`, 2}, // null is not a value
		{`{"f_num": null}`, 0},
		{`{"other": 1}`, 0}, // absent
		{`{"f_num": {"nested": 1}}`, 1},
		{`{"f_num": [[1], 3]}`, 1}, // a nested array does not coerce
		{`{"f_num": true}`, 1},
		{`{"f_num": 7}`, 0},
	}
	var want int64
	for i, d := range data {
		id := uuid.Must(uuid.NewV7())
		err := tenancy.InTenantTx(context.Background(), e.app, f.tenant, func(tx pgx.Tx) error {
			_, err := tx.Exec(context.Background(), insertRecordSQL, f.tenant, id, f.def, fmt.Sprintf("INC-U%d", i), d.json)
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
		want += d.uncoerced
	}
	f.register(t, e, "num", fieldindex.StateBackfilling)

	// Another tenant's type, all values coercing, backfilled on the same
	// counter: it gets no count, and f's count is not attributed to it.
	g := newFixture(t, e, "num", "")
	insertRecords(t, e, g, 4)
	g.register(t, e, "num", fieldindex.StateBackfilling)

	reader := sdkmetric.NewManualReader()
	m, err := recordports.NewUncoercedCounter(sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader)))
	if err != nil {
		t.Fatal(err)
	}
	for _, x := range []fixture{f, g} {
		// Batches of 4 over 9 records: three batches, the count summed.
		r := newRun(t, x.tenant, runOptions{acts: &backfill.Activities{Pool: e.worker, Uncoerced: m, BatchSize: 4}})
		r.signal(item(x, 1), 0)
		if err := r.coordinate(workerguard.RunInput{}); err != nil {
			t.Fatal(err)
		}
		r.requireGuardClean()
		if got := registry(t, e, x); got.state != "ready" {
			t.Fatalf("registry %+v, want ready", got)
		}
		if d := diffOf(t, e, x); d != noDiff {
			t.Fatalf("side tables vs recompute: %s", d)
		}
		// CI-5c: the count is a metric, never a payload. Every Batch result
		// is the cursor alone, and no payload names the count.
		r.mu.Lock()
		for _, p := range r.ledger {
			if strings.Contains(strings.ToLower(string(p)), "uncoerced") {
				t.Errorf("payload %s carries the uncoerced count", p)
			}
		}
		for _, c := range r.results {
			if b, _ := json.Marshal(c); string(b) != "{}" && !strings.HasPrefix(string(b), `{"cursor":`) {
				t.Errorf("Batch result %s, want {cursor} or {}", b)
			}
		}
		r.mu.Unlock()
	}

	got := uncoercedTotals(t, reader)
	if got[f.def.String()] != want {
		t.Errorf("uncoerced for the backfilled type = %d, want %d (all of %v)", got[f.def.String()], want, got)
	}
	if n, ok := got[g.def.String()]; ok {
		t.Errorf("uncoerced for the coercing type = %d, want no count", n)
	}
	if len(got) != 1 {
		t.Errorf("uncoerced counts %v, want only %s", got, f.def)
	}
}

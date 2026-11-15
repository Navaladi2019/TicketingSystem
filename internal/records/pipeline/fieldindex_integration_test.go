//go:build integration

package pipeline_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/entitytypes"
	"github.com/Navaladi2019/TicketingSystem/internal/records/pipeline"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// uncoercedCounts collects nava_field_index_uncoerced from reader: the sum
// by entity_definition, and every attribute key seen.
func uncoercedCounts(t *testing.T, reader *sdkmetric.ManualReader) (map[string]int64, map[string]bool) {
	t.Helper()
	var rm metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &rm); err != nil {
		t.Fatal(err)
	}
	counts, keys := map[string]int64{}, map[string]bool{}
	for _, sm := range rm.ScopeMetrics {
		for _, mt := range sm.Metrics {
			if mt.Name != "nava_field_index_uncoerced" {
				continue
			}
			sum, ok := mt.Data.(metricdata.Sum[int64])
			if !ok || !sum.IsMonotonic {
				t.Fatalf("nava_field_index_uncoerced is %T, want a monotonic int64 sum", mt.Data)
			}
			for _, dp := range sum.DataPoints {
				for _, kv := range dp.Attributes.ToSlice() {
					keys[string(kv.Key)] = true
				}
				def, _ := dp.Attributes.Value("entity_definition")
				counts[def.AsString()] += dp.Value
			}
		}
	}
	return counts, keys
}

// Acceptance (P7-02, ADR-006 D5, R5): a pipeline write whose flagged value
// does not coerce succeeds, is stored in records.data as sent, writes no side
// row for that value, and increments nava_field_index_uncoerced by
// entity_definition once the write commits. A write that rolls back counts
// nothing, and a write that coerces counts nothing.
func TestUncoercedValueIsCountedNotRaised(t *testing.T) {
	e := migrated(t)
	f := seedTenant(t, e)
	ctx := context.Background()
	var def uuid.UUID
	if err := e.super.QueryRow(ctx, `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2`, f.id, entityKey).Scan(&def); err != nil {
		t.Fatal(err)
	}
	// title and tags are flagged as text (P7-04 will write these rows on
	// publish). A title over 2,000 bytes does not coerce: it would not fit
	// the value index.
	for i, key := range []string{"title", "tags"} {
		mustExec(t, e.super, `INSERT INTO record_index_fields (tenant_id, id, entity_definition_id, field_key, value_type, state, requested_version)
			VALUES ($1, $2, $3, $4, 'text', 'ready', 1)`, f.id, uuid.NewSHA1(def, []byte{byte(i)}), def, key)
	}

	reader := sdkmetric.NewManualReader()
	m, err := pipeline.NewFieldIndexMetrics(sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader)))
	if err != nil {
		t.Fatal(err)
	}
	runner := e.runner()
	runner.FieldIndex = m
	p := pipeline.New(authz.ACL{Policies: authz.PolicyStore{}}, entitytypes.Store{}, pipeline.NoopSink{}, runner, pipeline.SystemClock{})
	sideRows := func(id uuid.UUID) int {
		return count(t, e.super, `SELECT count(*) FROM rfv_text WHERE record_id = $1`, id)
	}

	long := strings.Repeat("x", 2001)
	res := mustCreate(t, e, p, f, map[string]any{"title": long, "tags": []any{"a", "b"}})
	if data, _, _, _, _ := storedData(t, e, res.RecordID); data["title"] != long {
		t.Fatal("the stored title is not the one sent")
	}
	if got := sideRows(res.RecordID); got != 2 {
		t.Errorf("side rows = %d, want 2 (the tags; the title does not coerce)", got)
	}
	counts, keys := uncoercedCounts(t, reader)
	if counts[def.String()] != 1 || len(counts) != 1 {
		t.Errorf("after the create: counts = %v, want {%s: 1}", counts, def)
	}
	if len(keys) != 1 || !keys["entity_definition"] {
		t.Errorf("attributes = %v, want entity_definition only (no tenant, no value)", keys)
	}

	// Two bad values in one transaction count 2, read once at its commit.
	err = tenancy.InTenantTx(ctx, e.app, f.id, func(tx pgx.Tx) error {
		if _, err := p.Write(ctx, tx, updateCmd(f, res.RecordID, res.Version, map[string]any{"title": long + "y"})); err != nil {
			return err
		}
		_, err := p.Write(ctx, tx, createCmd(f, map[string]any{"title": long + "z"}))
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if counts, _ = uncoercedCounts(t, reader); counts[def.String()] != 3 {
		t.Errorf("after two bad writes in one transaction: counts = %v, want 3", counts)
	}

	// A rolled-back write counts nothing; a coercing one counts nothing.
	errRollback := errors.New("roll back")
	err = tenancy.InTenantTx(ctx, e.app, f.id, func(tx pgx.Tx) error {
		if _, err := p.Write(ctx, tx, createCmd(f, map[string]any{"title": long})); err != nil {
			return err
		}
		return errRollback
	})
	if !errors.Is(err, errRollback) {
		t.Fatal(err)
	}
	good := mustCreate(t, e, p, f, map[string]any{"title": "short"})
	if got := sideRows(good.RecordID); got != 1 {
		t.Errorf("a coercing title: side rows = %d, want 1", got)
	}
	if counts, _ = uncoercedCounts(t, reader); counts[def.String()] != 3 {
		t.Errorf("after a rollback and a good write: counts = %v, want still 3", counts)
	}
}

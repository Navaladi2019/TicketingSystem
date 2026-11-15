package pipeline

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// UncoercedSetting is the transaction-local setting in which the records
// trigger counts the values of flagged fields that do not coerce (ADR-006
// D5, R5; db/migrations/20261115130000_field_index_trigger.sql): a JSON
// object of entity definition id to count. The trigger writes no side row
// for such a value and never fails the write.
const UncoercedSetting = "nava.field_index_uncoerced"

// FieldIndexMetrics are the field index's write-side metrics. A nil
// *FieldIndexMetrics records nothing.
type FieldIndexMetrics struct {
	uncoerced metric.Int64Counter
}

// NewFieldIndexMetrics registers nava_field_index_uncoerced (exported to
// Prometheus as nava_field_index_uncoerced_total) on mp: values of flagged
// fields, in committed record writes, that did not coerce to their field's
// value type, by entity_definition. The entity definition id is the only
// attribute: no tenant id and no value (R5).
func NewFieldIndexMetrics(mp metric.MeterProvider) (*FieldIndexMetrics, error) {
	m := mp.Meter("github.com/Navaladi2019/TicketingSystem/internal/records/pipeline")
	uncoerced, err := m.Int64Counter("nava_field_index_uncoerced",
		metric.WithDescription("Values of filterable fields in committed record writes that did not coerce to the field's value type; they are stored in records.data but not filterable (ADR-006 R5)."),
		metric.WithUnit("{value}"))
	if err != nil {
		return nil, fmt.Errorf("pipeline: %w", err)
	}
	return &FieldIndexMetrics{uncoerced: uncoerced}, nil
}

// watch has tx read UncoercedSetting in its commit's round trip and, once
// committed, add the counts to the counter. Once per transaction, whatever
// the number of writes: the setting accumulates over them.
func (m *FieldIndexMetrics) watch(tx pgx.Tx) {
	if m == nil {
		return
	}
	tenancy.ReadSettingAtCommit(tx, UncoercedSetting, m.observe)
}

// observe adds each entity definition's count in value, the setting as the
// transaction left it, to the counter. Anything that is not the trigger's
// shape is ignored: any session may set a custom setting.
func (m *FieldIndexMetrics) observe(ctx context.Context, value string) {
	if value == "" {
		return
	}
	var counts map[string]int64
	if err := json.Unmarshal([]byte(value), &counts); err != nil {
		return
	}
	// The write has committed; its context may be cancelled already.
	ctx = context.WithoutCancel(ctx)
	for def, n := range counts {
		if _, err := uuid.Parse(def); err != nil || n <= 0 {
			continue
		}
		m.uncoerced.Add(ctx, n, metric.WithAttributes(attribute.String("entity_definition", def)))
	}
}

// fieldIndexMetrics is the FieldIndexMetrics of the pipeline's runner, when
// it is a PoolRunner (or embeds one); nil, which records nothing, otherwise.
func (p *Pipeline) fieldIndexMetrics() *FieldIndexMetrics {
	if r, ok := p.runner.(interface{ fieldIndexMetrics() *FieldIndexMetrics }); ok {
		return r.fieldIndexMetrics()
	}
	return nil
}

func (r PoolRunner) fieldIndexMetrics() *FieldIndexMetrics { return r.FieldIndex }

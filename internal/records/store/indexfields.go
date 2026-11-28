package store

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/fieldindex"
)

// IndexField is one field-index registry row of an entity type (ADR-006 D2).
type IndexField struct {
	ID        uuid.UUID
	Key       string
	ValueType fieldindex.ValueType
	State     fieldindex.State
	// Progress is the backfill's whole percentage, 0 to 100.
	Progress int
}

// IndexFields reads the registry rows of one entity type of the transaction's
// tenant, by field key. It reads the registry only, never a side table (D7).
func IndexFields(ctx context.Context, tx pgx.Tx, tenantID, definitionID uuid.UUID) (map[string]IndexField, error) {
	rows, err := tx.Query(ctx, `SELECT id, field_key, value_type, state, progress FROM record_index_fields
WHERE tenant_id = $1 AND entity_definition_id = $2`, tenantID, definitionID)
	if err != nil {
		return nil, fmt.Errorf("records/store: read field index registry: %w", err)
	}
	return collectIndexFields(rows)
}

// collectIndexFields reads every registry row of rows and closes it.
func collectIndexFields(rows pgx.Rows) (map[string]IndexField, error) {
	defer rows.Close()
	out := map[string]IndexField{}
	for rows.Next() {
		var f IndexField
		var vt, st string
		if err := rows.Scan(&f.ID, &f.Key, &vt, &st, &f.Progress); err != nil {
			return nil, fmt.Errorf("records/store: scan field index registry: %w", err)
		}
		f.ValueType, f.State = fieldindex.ValueType(vt), fieldindex.State(st)
		out[f.Key] = f
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("records/store: read field index registry: %w", err)
	}
	return out, nil
}

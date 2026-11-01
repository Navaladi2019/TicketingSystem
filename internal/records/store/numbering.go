package store

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// NextNumber takes the next record number of the tenant's entity type
// (D-3-9): one upsert on record_counters, in the caller's transaction. The
// first call for a type inserts next = 2 and returns 1; every later one adds
// one. The upsert holds the counter row's lock until the transaction ends, so
// concurrent creates of the same type queue here and get consecutive numbers,
// and a rollback gives its number back to the next create (no gap). There is
// no sequence (ADR-001 I8) and no step creates the row in advance.
func NextNumber(ctx context.Context, tx pgx.Tx, tenantID, entityDefinitionID uuid.UUID) (int64, error) {
	var n int64
	err := tx.QueryRow(ctx, `
INSERT INTO record_counters (tenant_id, entity_definition_id, next) VALUES ($1, $2, 2)
ON CONFLICT (tenant_id, entity_definition_id) DO UPDATE SET next = record_counters.next + 1
RETURNING next - 1`, tenantID, entityDefinitionID).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("records/store: take record number: %w", err)
	}
	return n, nil
}

// FormatNumber is a record number: the prefix and the counter value, zero
// padded to 7 digits and growing past 7 as needed (INC0000001, INC12345678).
// Past 7 digits the text no longer sorts in counter order; lists sort on
// created_at.
func FormatNumber(prefix string, n int64) string {
	return fmt.Sprintf("%s%07d", prefix, n)
}

package store

import (
	"time"

	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// NextNumber takes the next record number of the tenant's entity type
// (D-3-9 as amended 2026-11-10: gaps allowed, no reclaim): one upsert on
// record_counters. The first call for a type inserts next = 2 and returns 1;
// every later one adds one. The upsert holds the counter row's lock until tx
// ends, so tx should be a short transaction of its own that commits before
// the record is inserted: the record pipeline runs it on its TxRunner, never
// in the create's transaction, so concurrent creates of a type queue only on
// this upsert and its COMMIT. A create that fails after its number committed
// leaves a gap; numbers are never reused, and stay unique and increasing per
// tenant and type. There is no sequence (ADR-001 I8) and no step creates the
// row in advance.
func NextNumber(ctx context.Context, tx pgx.Tx, tenantID, entityDefinitionID uuid.UUID) (int64, error) {
	var n int64
	err := tx.QueryRow(ctx, nextNumberSQL, tenantID, entityDefinitionID).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("records/store: take record number: %w", err)
	}
	return n, nil
}

// nextNumberSQL takes the next number: the first call for a type inserts
// next = 2 and returns 1, every later one adds one.
const nextNumberSQL = `
INSERT INTO record_counters (tenant_id, entity_definition_id, next) VALUES ($1, $2, 2)
ON CONFLICT (tenant_id, entity_definition_id) DO UPDATE SET next = record_counters.next + 1
RETURNING next - 1`

// allocateSQL is nextNumberSQL, the same upsert of the same row, also
// returning now(): the allocation's instant, which the create stores as
// created_at (ADR-005 A7). now() is fixed at the allocation transaction's
// start, so it adds no statement and no round trip.
const allocateSQL = nextNumberSQL + ", now()"

// Allocated is what QueueAllocate reads back when the batch runs.
type Allocated struct {
	// Number is the counter value taken.
	Number int64
	// Instant is the allocation transaction's now().
	Instant time.Time
}

// QueueAllocate queues a `gaps` type's number allocation (ADR-005 L2s) for
// tenancy.AllocTx: the record_counters upsert of the tenant's type, in the
// allocation's own transaction, which commits before the create inserts the
// record. When the batch runs, *dst holds the number and the instant.
func QueueAllocate(b *pgx.Batch, tenantID, entityDefinitionID uuid.UUID, dst *Allocated) {
	b.Queue(allocateSQL, tenantID, entityDefinitionID).QueryRow(func(row pgx.Row) error {
		if err := row.Scan(&dst.Number, &dst.Instant); err != nil {
			return fmt.Errorf("records/store: take record number: %w", err)
		}
		return nil
	})
}

// FormatNumber is a record number: the prefix and the counter value, zero
// padded to 7 digits and growing past 7 as needed (INC0000001, INC12345678).
// Past 7 digits the text no longer sorts in counter order; lists sort on
// created_at.
func FormatNumber(prefix string, n int64) string {
	return fmt.Sprintf("%s%07d", prefix, n)
}

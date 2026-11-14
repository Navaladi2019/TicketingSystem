package store

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/records/numbering"
)

// NextNumber takes the next record number of the tenant's entity type in
// the counter of scope scopeKey (D-3-9 as amended 2026-11-10: gaps allowed,
// no reclaim; ADR-005 §4.2): one upsert on record_counters. The first call
// for a (type, scope) inserts next = 2 and returns 1; every later one adds
// one. The upsert holds the counter row's lock until tx ends, so tx should
// be a short transaction of its own that commits before the record is
// inserted: the record pipeline runs it on its allocator pool, never in the
// create's transaction, so concurrent creates of a type queue only on this
// upsert and its COMMIT. A create that fails after its number committed
// leaves a gap; numbers are never reused, and stay unique and increasing per
// tenant, type and scope. There is no sequence (ADR-001 I8) and no step
// creates the row in advance.
func NextNumber(ctx context.Context, tx pgx.Tx, tenantID, entityDefinitionID uuid.UUID, scopeKey string) (int64, error) {
	var n int64
	err := tx.QueryRow(ctx, nextNumberSQL, tenantID, entityDefinitionID, scopeKey).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("records/store: take record number: %w", err)
	}
	return n, nil
}

// nextNumberSQL takes the next number of a (tenant, type, scope) counter
// (ADR-005 §4.2, R1): the first call inserts next = 2 and returns 1, every
// later one adds one. Its arbiter is record_counters_pkey, the 3-column key
// since M3 (TICK-670) and the table's only unique index.
const nextNumberSQL = `
INSERT INTO record_counters (tenant_id, entity_definition_id, scope_key, next) VALUES ($1, $2, $3, 2)
ON CONFLICT (tenant_id, entity_definition_id, scope_key) DO UPDATE SET next = record_counters.next + 1
RETURNING next - 1`

// The counter upsert of a scope with a date part (ADR-005 A7, §4.1, §3.6):
// PostgreSQL computes the scope key and period_end from now() in the
// numbering time zone, the format's own ($4) or else the tenant's
// time_zone, read under the transaction's tenant (RLS). $3 is the key's
// prefix (e<epoch>|d:), $5 the date parts' to_char patterns in order, $6 the
// period's date_trunc unit or ” for none. The INSERT arm sets period_end;
// the UPDATE arm sets next only, as nextNumberSQL's. A tenant row the
// statement cannot read yields no zone and so no row, never a key without
// its date: the upsert then writes nothing and returns nothing.
const (
	scopeZoneSQL  = `coalesce(nullif($4::text, ''), (SELECT t.time_zone FROM tenants t WHERE t.id = $1))`
	scopeDatedSQL = `
WITH z AS (
    SELECT s.tz, now() AT TIME ZONE s.tz AS local
    FROM (SELECT ` + scopeZoneSQL + ` AS tz) s
    WHERE s.tz IS NOT NULL)
INSERT INTO record_counters (tenant_id, entity_definition_id, scope_key, next, period_end)
SELECT $1, $2,
       $3::text || (SELECT string_agg(to_char(z.local, p.f), '' ORDER BY p.i) FROM unnest($5::text[]) WITH ORDINALITY AS p(f, i)),
       2,
       CASE WHEN $6::text = '' THEN NULL
            ELSE (date_trunc($6::text, z.local) + ('1 ' || $6::text)::interval) AT TIME ZONE z.tz END
FROM z
ON CONFLICT (tenant_id, entity_definition_id, scope_key) DO UPDATE SET next = record_counters.next + 1
RETURNING next - 1, scope_key`
)

// upsertSQL is the counter upsert of scope and its arguments: nextNumberSQL
// with the key itself for a scope with no date part (the default, epoch 0,
// is exactly R1's statement), else scopeDatedSQL.
func upsertSQL(tenantID, entityDefinitionID uuid.UUID, scope numbering.Scope) (string, []any) {
	if !scope.Dated() {
		return nextNumberSQL + ", scope_key", []any{tenantID, entityDefinitionID, scope.KeyPrefix}
	}
	return scopeDatedSQL, []any{tenantID, entityDefinitionID, scope.KeyPrefix, scope.TimeZone, scope.Patterns, scope.PeriodUnit}
}

// Allocated is what QueueAllocate reads back when the batch runs.
type Allocated struct {
	// Number is the counter value taken.
	Number int64
	// ScopeKey is the key of the counter row the upsert took it from, as
	// PostgreSQL computed it.
	ScopeKey string
	// Instant is the allocation transaction's now().
	Instant time.Time
}

// QueueAllocate queues a `gaps` type's number allocation (ADR-005 L2s) for
// tenancy.AllocTx: the record_counters upsert of the tenant's type in
// scope, in the allocation's own transaction, which commits before the
// create inserts the record. It also returns now(): the allocation's
// instant, which the create stores as created_at, and the instant the scope
// key's date part was computed from (ADR-005 A7). now() is fixed at the
// allocation transaction's start, so it adds no statement and no round trip.
// When the batch runs, *dst holds the number, the key and the instant; a
// tenant whose zone the statement cannot read fails the batch.
func QueueAllocate(b *pgx.Batch, tenantID, entityDefinitionID uuid.UUID, scope numbering.Scope, dst *Allocated) {
	sql, args := upsertSQL(tenantID, entityDefinitionID, scope)
	b.Queue(sql+", now()", args...).QueryRow(func(row pgx.Row) error {
		if err := row.Scan(&dst.Number, &dst.ScopeKey, &dst.Instant); err != nil {
			return fmt.Errorf("records/store: take record number: %w", err)
		}
		return nil
	})
}

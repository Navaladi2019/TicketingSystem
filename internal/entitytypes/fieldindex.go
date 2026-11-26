package entitytypes

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/fieldindex"
	"github.com/Navaladi2019/TicketingSystem/internal/metadata"
	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
)

// QuotaScope says which limit an IndexQuotaError hit.
type QuotaScope string

// The two ADR-006 D12 limits.
const (
	QuotaPerType QuotaScope = "type"
	QuotaTenant  QuotaScope = "tenant"
)

// IndexQuotaError is returned by Create and Publish when the new version would flag more
// fields than the type or the tenant may index (ADR-006 D12, code
// index_quota_exceeded). Nothing is written once it is returned.
type IndexQuotaError struct {
	Scope QuotaScope
	Limit int
	// Paths are JSON pointers into the definition, one per newly flagged field.
	Paths []string
}

func (e *IndexQuotaError) Error() string {
	return fmt.Sprintf("entitytypes: index_quota_exceeded: more than %d filterable fields per %s", e.Limit, e.Scope)
}

// FieldIndexBusyError is returned by Publish when a newly flagged field's
// registry row is being removed (code field_index_busy). Nothing is written.
type FieldIndexBusyError struct{ Paths []string }

func (e *FieldIndexBusyError) Error() string {
	return "entitytypes: field_index_busy at " + strings.Join(e.Paths, ", ")
}

// fieldPointer is the JSON pointer of a top-level property, relative to the
// definition (the handler prefixes /definition).
func fieldPointer(key string) string {
	return "/schema/properties/" + strings.NewReplacer("~", "~0", "/", "~1").Replace(key)
}

// removedFilterable returns the keys of the fields prev flags that next does
// not, sorted: the flag is gone (ADR-006 D8). A property deleted outright is
// Breaking, so Publish refuses it before this matters.
func removedFilterable(prev, next *metadata.Compiled) []string {
	nextKeys := map[string]bool{}
	for _, f := range next.Filterable() {
		nextKeys[f.Key] = true
	}
	var keys []string
	for _, f := range prev.Filterable() { // sorted by key
		if !nextKeys[f.Key] {
			keys = append(keys, f.Key)
		}
	}
	return keys
}

// addedFilterable returns the fields next flags that prev does not.
func addedFilterable(prev, next *metadata.Compiled) []metadata.FilterableField {
	had := map[string]bool{}
	for _, f := range prev.Filterable() {
		had[f.Key] = true
	}
	var out []metadata.FilterableField
	for _, f := range next.Filterable() {
		if !had[f.Key] {
			out = append(out, f)
		}
	}
	return out
}

// tenantQuotaLockSQL takes a transaction-scoped advisory lock keyed on the
// tenant (PLAN-007 D-7-13). Two publishes of one tenant that both add flags
// queue here, so the second counts after the first has inserted. The lock is
// released at commit or rollback; it is also safe through PgBouncer's
// transaction pooling.
const tenantQuotaLockSQL = `SELECT pg_advisory_xact_lock(hashtextextended('field_index.quota:' || $1::text, 0))`

// registerFilterable writes, in the publish transaction, one registry row
// (backfilling, progress 0, requested_version = version) and one outbox row
// per newly flagged field (ADR-006 D8, D12, P7-04). It holds the tenant's
// quota lock before it counts, so two concurrent publishes cannot both pass.
// A field that already has a registry row gets none: a row in state removing
// is *FieldIndexBusyError. This is the only nava_app path that writes the
// registry. Its two callers, Store.Create (version 1) and Store.Publish, run
// it inside the audited version write, which is why the registry is exempt
// from the audit commit check.
func registerFilterable(ctx context.Context, tx pgx.Tx, tenantID, defID uuid.UUID, version int, added []metadata.FilterableField) error {
	if len(added) == 0 {
		return nil
	}
	if _, err := tx.Exec(ctx, tenantQuotaLockSQL, tenantID); err != nil {
		return fmt.Errorf("entitytypes: lock field index quota: %w", err)
	}
	rows, err := tx.Query(ctx, `
SELECT field_key, state FROM record_index_fields WHERE tenant_id = $1 AND entity_definition_id = $2`, tenantID, defID)
	if err != nil {
		return fmt.Errorf("entitytypes: read field index registry: %w", err)
	}
	existing := map[string]fieldindex.State{}
	for rows.Next() {
		var key string
		var st fieldindex.State
		if err := rows.Scan(&key, &st); err != nil {
			rows.Close()
			return fmt.Errorf("entitytypes: read field index registry: %w", err)
		}
		existing[key] = st
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("entitytypes: read field index registry: %w", err)
	}
	var busy, fresh []string
	var create []metadata.FilterableField
	for _, f := range added {
		switch existing[f.Key] {
		case "":
			create = append(create, f)
			fresh = append(fresh, fieldPointer(f.Key))
		case fieldindex.StateRemoving:
			busy = append(busy, fieldPointer(f.Key))
		}
	}
	if len(busy) > 0 {
		return &FieldIndexBusyError{Paths: busy}
	}
	if len(create) == 0 {
		return nil
	}
	typeLimit, err := fieldindex.PerTypeLimit(ctx, tx, tenantID)
	if err != nil {
		return fmt.Errorf("entitytypes: per-type index limit: %w", err)
	}
	if len(existing)+len(create) > typeLimit {
		return &IndexQuotaError{Scope: QuotaPerType, Limit: typeLimit, Paths: fresh}
	}
	var inTenant int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM record_index_fields WHERE tenant_id = $1`, tenantID).Scan(&inTenant); err != nil {
		return fmt.Errorf("entitytypes: count field index registry: %w", err)
	}
	if inTenant+len(create) > fieldindex.TenantLimit {
		return &IndexQuotaError{Scope: QuotaTenant, Limit: fieldindex.TenantLimit, Paths: fresh}
	}
	for _, f := range create {
		id, err := uuid.NewV7()
		if err != nil {
			return fmt.Errorf("entitytypes: field index registry id: %w", err)
		}
		if _, err := tx.Exec(ctx, `
INSERT INTO record_index_fields (tenant_id, id, entity_definition_id, field_key, value_type, state, progress, requested_version)
VALUES ($1, $2, $3, $4, $5, $6, 0, $7)`,
			tenantID, id, defID, f.Key, string(f.ValueType), string(fieldindex.StateBackfilling), version); err != nil {
			return fmt.Errorf("entitytypes: insert field index registry row: %w", err)
		}
		if _, err := outbox.EnqueueFieldIndexBackfill(ctx, tx, id, version); err != nil {
			return fmt.Errorf("entitytypes: enqueue field index backfill: %w", err)
		}
	}
	return nil
}

// retireFilterableSQL flips the type's registry rows for keys to removing at
// the publishing version (ADR-006 D11 step 1). A row already removing is left
// alone. From the commit on, the records trigger writes nothing for the field
// and a filter on it answers 400 field_not_filterable.
const retireFilterableSQL = `
UPDATE record_index_fields SET state = 'removing', requested_version = $4, updated_at = now()
WHERE tenant_id = $1 AND entity_definition_id = $2 AND field_key = ANY($3::text[]) AND state <> 'removing'
RETURNING id`

// retireFilterable writes, in the publish transaction, the flip to removing
// of each field the new version no longer flags, and one field_index.remove
// outbox row per flipped row (ADR-006 D8, D11, TICK-750). The relay signals
// the tenant's coordinator, which runs FieldIndexRemove after any backfill
// ahead of it. A key with no registry row writes nothing. The tenant is the
// caller's, from InTenantTx, never the request's.
func retireFilterable(ctx context.Context, tx pgx.Tx, tenantID, defID uuid.UUID, version int, keys []string) error {
	if len(keys) == 0 {
		return nil
	}
	rows, err := tx.Query(ctx, retireFilterableSQL, tenantID, defID, keys, version)
	if err != nil {
		return fmt.Errorf("entitytypes: flip field index registry rows to removing: %w", err)
	}
	ids, err := pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
	if err != nil {
		return fmt.Errorf("entitytypes: flip field index registry rows to removing: %w", err)
	}
	for _, id := range ids {
		if _, err := outbox.EnqueueFieldIndexRemove(ctx, tx, id, version); err != nil {
			return fmt.Errorf("entitytypes: enqueue field index removal: %w", err)
		}
	}
	return nil
}

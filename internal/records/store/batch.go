package store

// Batched forms of the store's statements (TICK-648). Each Queue function
// adds one statement to a pgx.Batch the caller sends in its tenancy.InTenantTx
// transaction, whose BEGIN already set the tenant, and fills its destination
// when the batch's results are read (pgx.BatchResults.Close). A not-found
// result is a zero destination, never a batch error; any other failure fails
// the batch, wrapped as the unbatched function wraps it.

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Navaladi2019/TicketingSystem/internal/metadata"
)

// queueExec queues sql as a statement whose failure is wrap(err). pgx's own
// Exec callback is not called on a failure, so the item reads its result
// itself.
func queueExec(b *pgx.Batch, wrap func(error) error, sql string, args ...any) {
	qq := b.Queue(sql, args...)
	qq.Fn = func(br pgx.BatchResults) error {
		if _, err := br.Exec(); err != nil {
			return wrap(err)
		}
		return nil
	}
}

// QueueDefinitionID queues DefinitionID: *dst is the id of the tenant's
// entity type key, or uuid.Nil when it has none.
func QueueDefinitionID(b *pgx.Batch, tenantID uuid.UUID, key string, dst *uuid.UUID) {
	b.Queue(`SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = $2`, tenantID, key).
		QueryRow(func(row pgx.Row) error {
			err := row.Scan(dst)
			if errors.Is(err, pgx.ErrNoRows) {
				*dst = uuid.Nil
				return nil
			}
			if err != nil {
				return fmt.Errorf("records/store: read entity type: %w", err)
			}
			return nil
		})
}

// QueueGet queues Get: *dst is the record, or nil when the tenant has no
// such pipeline record.
func QueueGet(b *pgx.Batch, tenantID, id uuid.UUID, dst **Record) {
	b.Queue(recordSelect, tenantID, id).QueryRow(func(row pgx.Row) error {
		r, err := scan(row)
		if errors.Is(err, ErrNotFound) {
			*dst = nil
			return nil
		}
		*dst = r
		return err
	})
}

// QueueGroupLive queues GroupLive: *dst reports whether the tenant has group
// id, not tombstoned.
func QueueGroupLive(b *pgx.Batch, tenantID, id uuid.UUID, dst *bool) {
	b.Queue(`SELECT EXISTS (SELECT 1 FROM groups WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL)`, tenantID, id).
		QueryRow(func(row pgx.Row) error {
			if err := row.Scan(dst); err != nil {
				return fmt.Errorf("records/store: read group: %w", err)
			}
			return nil
		})
}

// QueueWriteKeyByKey queues WriteKeyByKey: *dst is the write by actor that
// holds key, or nil when there is none.
func QueueWriteKeyByKey(b *pgx.Batch, tenantID uuid.UUID, actor *uuid.UUID, key string, dst **WriteKey) {
	b.Queue(writeKeySelect, tenantID, key, actor).QueryRow(func(row pgx.Row) error {
		k := WriteKey{TenantID: tenantID, IdempotencyKey: key, ActorPrincipalID: actor}
		err := row.Scan(&k.DefinitionID, &k.Action, &k.RecordID, &k.Number, &k.Version, &k.SchemaVersion, &k.AuditEntryID)
		if errors.Is(err, pgx.ErrNoRows) {
			*dst = nil
			return nil
		}
		if err != nil {
			return fmt.Errorf("records/store: read write key: %w", err)
		}
		*dst = &k
		return nil
	})
}

// QueueLockUnique queues LockUnique.
func QueueLockUnique(b *pgx.Batch, tenantID, entityDefinitionID uuid.UUID, field, canonical string) {
	queueExec(b, func(err error) error { return fmt.Errorf("records/store: lock unique value: %w", err) },
		lockUniqueSQL, tenantID, entityDefinitionID, field, canonical)
}

// QueueUniqueTaken queues UniqueTaken: *dst reports whether another record
// holds value in field. Queue it after the value's QueueLockUnique in the
// same batch: PostgreSQL runs a batch's statements in order, and at READ
// COMMITTED the check's snapshot is taken after the lock was granted.
func QueueUniqueTaken(b *pgx.Batch, tenantID, entityDefinitionID, except uuid.UUID,
	field string, value any, fixed map[string]string, dst *bool) error {
	sql, args, err := uniqueTakenSQL(tenantID, entityDefinitionID, except, field, value, fixed)
	if err != nil {
		return err
	}
	b.Queue(sql, args...).QueryRow(func(row pgx.Row) error {
		if err := row.Scan(dst); err != nil {
			return fmt.Errorf("records/store: check unique value: %w", err)
		}
		return nil
	})
	return nil
}

// QueuePayloadPreview queues the payload PayloadJSON would return for r once
// inserted: the same expression, over r's values as bind parameters instead
// of the stored row, so *dst holds PostgreSQL's own jsonb text of them.
// InsertNumbered returns the stored row's payload, which the pipeline
// compares with this one, so an audit diff built from the preview is the
// stored row's.
func QueuePayloadPreview(b *pgx.Batch, r *Record, fixed map[string]string, dst *json.RawMessage) error {
	data, err := json.Marshal(r.Data)
	if err != nil {
		return fmt.Errorf("records/store: encode data: %w", err)
	}
	expr, args, err := payloadExpr(fixed, []any{jsonParam(data), r.State, r.RequesterID, r.AssigneeID, r.AssignmentGroupID})
	if err != nil {
		return err
	}
	b.Queue("SELECT ("+expr+")::text FROM (SELECT $1::jsonb AS data, $2::text AS state, $3::uuid AS requester_id, "+
		"$4::uuid AS assignee_id, $5::uuid AS assignment_group_id) AS records", args...).
		QueryRow(func(row pgx.Row) error {
			var s string
			if err := row.Scan(&s); err != nil {
				return fmt.Errorf("records/store: preview payload as jsonb: %w", err)
			}
			*dst = json.RawMessage(s)
			return nil
		})
	return nil
}

// payloadExpr is PayloadJSON's expression: the jsonb payload of a records
// row (columns data, state, requester_id, assignee_id and
// assignment_group_id) under fixed. args are the statement's first
// arguments; the expression's own follow them, and the returned args hold
// both.
func payloadExpr(fixed map[string]string, args []any) (string, []any, error) {
	props := slices.Sorted(maps.Keys(fixed))
	args = append(args, props)
	expr := fmt.Sprintf("(data - $%d::text[])", len(args))
	for _, prop := range props {
		col := fixed[prop]
		switch col {
		case metadata.ColumnState, metadata.ColumnRequesterID, metadata.ColumnAssigneeID, metadata.ColumnAssignmentGroupID:
		default:
			return "", nil, fmt.Errorf("records/store: unknown fixed column %q", col)
		}
		args = append(args, prop)
		n := len(args)
		// col is one of the four constants above, never input.
		if col == metadata.ColumnState {
			expr += fmt.Sprintf(" || jsonb_build_object($%d::text, %s)", n, col)
			continue
		}
		expr += fmt.Sprintf(" || CASE WHEN %s IS NULL THEN '{}'::jsonb ELSE jsonb_build_object($%d::text, %s) END", col, n, col)
	}
	return expr, args, nil
}

// Numbered is what QueueInsertNumbered reads back when the batch runs.
type Numbered struct {
	// N is the counter value the create took.
	N int64
	// Number is the number the INSERT stored, built by PostgreSQL from N.
	Number string
	// Payload is the stored row's payload, as PayloadJSON returns it.
	Payload json.RawMessage
	// CreatedAt and UpdatedAt are the stored timestamps, at PostgreSQL's
	// precision.
	CreatedAt, UpdatedAt time.Time
}

// numberSQL is FormatNumber in SQL over the counter value the batch's first
// statement took: the prefix, then the value zero padded to 7 digits and
// longer as needed (lpad alone would cut an 8-digit value).
const numberSQL = `$15::text || (SELECT CASE WHEN c.next - 1 < 10000000 THEN lpad((c.next - 1)::text, 7, '0')
    ELSE (c.next - 1)::text END
  FROM record_counters c WHERE c.tenant_id = $1 AND c.entity_definition_id = $4)`

// QueueInsertNumbered queues a create's numbering and insert, the last
// statements before the audit entry (D-3-9, TICK-648): the record_counters
// upsert, which locks the tenant's counter row of the type until the
// transaction ends, then the INSERT of r, whose number PostgreSQL builds from
// the counter value the upsert left, under prefix. r.Number is ignored.
// When the batch runs, *dst holds the value, the number, the stored
// timestamps and the stored payload under fixed. A rollback after this gives the number back (no gap),
// as NextNumber's does.
func QueueInsertNumbered(b *pgx.Batch, r *Record, prefix string, fixed map[string]string, dst *Numbered) error {
	data, err := json.Marshal(r.Data)
	if err != nil {
		return fmt.Errorf("records/store: encode data: %w", err)
	}
	b.Queue(nextNumberSQL, r.TenantID, r.EntityDefinitionID).QueryRow(func(row pgx.Row) error {
		if err := row.Scan(&dst.N); err != nil {
			return fmt.Errorf("records/store: take record number: %w", err)
		}
		return nil
	})
	expr, args, err := payloadExpr(fixed, []any{
		r.TenantID, r.ID, r.Version, r.EntityDefinitionID, r.SchemaVersion, r.State,
		r.RequesterID, r.AssigneeID, r.AssignmentGroupID, r.CreatedBy, r.CreatedAt, jsonParam(data),
		r.CausationOriginID, r.CausationDepth, prefix,
	})
	if err != nil {
		return err
	}
	b.Queue(`
INSERT INTO records (tenant_id, id, version, entity_definition_id, schema_version, number, state,
    requester_id, assignee_id, assignment_group_id, created_by, created_at, updated_at, data,
    causation_origin_id, causation_depth)
VALUES ($1, $2, $3, $4, $5, `+numberSQL+`, $6, $7, $8, $9, $10, $11, $11, $12, $13, $14)
RETURNING number, created_at, updated_at, (`+expr+`)::text`, args...).QueryRow(func(row pgx.Row) error {
		var payload string
		if err := row.Scan(&dst.Number, &dst.CreatedAt, &dst.UpdatedAt, &payload); err != nil {
			return fmt.Errorf("records/store: insert record: %w", err)
		}
		dst.Payload = json.RawMessage(payload)
		return nil
	})
	return nil
}

// QueueInsertWriteKey queues InsertWriteKey for a record the same batch
// inserted before it: k.Number is ignored and the stored number is the
// record's. A key another write by the same principal holds fails the batch
// with ErrDuplicateKey.
func QueueInsertWriteKey(b *pgx.Batch, k *WriteKey) {
	queueExec(b, writeKeyError, `
INSERT INTO record_write_keys (tenant_id, idempotency_key, definition_id, action, record_id, number,
    version, schema_version, audit_entry_id, actor_principal_id)
VALUES ($1, $2, $3, $4, $5, (SELECT number FROM records WHERE tenant_id = $1 AND id = $5), $6, $7, $8, $9)`,
		k.TenantID, k.IdempotencyKey, k.DefinitionID, k.Action, k.RecordID,
		k.Version, k.SchemaVersion, k.AuditEntryID, k.ActorPrincipalID)
}

// writeKeyError maps a failed write-key insert as InsertWriteKey does.
func writeKeyError(err error) error {
	var pe *pgconn.PgError
	if errors.As(err, &pe) && pe.Code == "23505" && pe.ConstraintName == idempotencyIndex {
		return ErrDuplicateKey
	}
	return fmt.Errorf("records/store: insert write key: %w", err)
}

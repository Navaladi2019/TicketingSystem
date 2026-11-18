package backfill

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/internal/fieldindex"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// One batch (ADR-006 D10.2) is one InTenantTx as nava_worker:
//
//	(a) the registry row FOR SHARE: stop, writing nothing, unless it is
//	    backfilling (and at the child's pinned requested_version). The share
//	    lock holds a concurrent flip to removing until the batch commits.
//	(b) the next record ids after the cursor, keyset on the records primary
//	    key, ORDER BY id LIMIT n FOR SHARE: a concurrent update of one of them
//	    waits for this commit, then its trigger rewrites the field from the
//	    newer data.
//	(c) for those records, DELETE the field's side rows, then INSERT the
//	    record_field_coerce rows: the one coercion (D5), in SQL; record data
//	    never reaches Go.
//	(d) progress, a whole percentage below 100, on the registry row.
//
// The side table name and value column come from fieldindex.ValueType's
// fixed set, never from the row, so they may be spliced into the SQL.

const (
	lockFieldSQL = `SELECT entity_definition_id, field_key, value_type, state, requested_version
		FROM record_index_fields WHERE tenant_id = $1 AND id = $2 FOR SHARE`
	nextIDsSQL = `SELECT id FROM records
		WHERE tenant_id = $1 AND entity_definition_id = $2 AND id > $3
		ORDER BY id LIMIT $4 FOR SHARE`
	deleteSideSQL = `DELETE FROM %s WHERE tenant_id = $1 AND field_id = $2 AND record_id = ANY($3::uuid[])`
	insertSideSQL = `INSERT INTO %[1]s (tenant_id, field_id, record_id, ord, value)
		SELECT r.tenant_id, $2, r.id, c.ord, c.%[2]s
		FROM records r CROSS JOIN LATERAL record_field_coerce(r.data -> $4::text, $5::text) c
		WHERE r.tenant_id = $1 AND r.id = ANY($3::uuid[])`
	progressSQL = `SELECT count(*) FILTER (WHERE id <= $3), count(*) FROM records
		WHERE tenant_id = $1 AND entity_definition_id = $2`
	setProgressSQL = `UPDATE record_index_fields SET progress = $3, updated_at = now()
		WHERE tenant_id = $1 AND id = $2 AND progress <> $3`
	readySQL = `UPDATE record_index_fields SET state = 'ready', progress = 100, updated_at = now()
		WHERE tenant_id = $1 AND id = $2 AND state = 'backfilling' AND ($3 = 0 OR requested_version = $3)`
)

// valueColumn is record_field_coerce's output column for v.
func valueColumn(v fieldindex.ValueType) (string, bool) {
	switch v {
	case fieldindex.TypeText:
		return "value_text", true
	case fieldindex.TypeNum:
		return "value_num", true
	case fieldindex.TypeTS:
		return "value_ts", true
	case fieldindex.TypeBool:
		return "value_bool", true
	}
	return "", false
}

// registryRow is the part of a registry row a batch reads.
type registryRow struct {
	entityDefinition uuid.UUID
	key              string
	valueType        fieldindex.ValueType
	state            fieldindex.State
	version          int
}

// fills reports whether c may write the row's side rows: it is backfilling,
// at c's pinned version when c has one.
func (r registryRow) fills(c child) bool {
	return r.state == fieldindex.StateBackfilling && (c.version == 0 || r.version == c.version)
}

// copyBatch runs one batch of c's field after cursor (nil: from the start)
// and returns the new cursor, or nil when the field is done or no longer
// c's to fill. Nothing is written when it returns nil without copying.
func copyBatch(ctx context.Context, pool *pgxpool.Pool, c child, cursor *uuid.UUID, size int) (*uuid.UUID, error) {
	var next *uuid.UUID
	err := tenancy.InTenantTx(ctx, pool, c.tenant, func(tx pgx.Tx) error {
		next = nil
		var row registryRow
		var vt, state string
		err := tx.QueryRow(ctx, lockFieldSQL, c.tenant, c.field).
			Scan(&row.entityDefinition, &row.key, &vt, &state, &row.version)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("lock registry row: %w", err)
		}
		row.valueType, row.state = fieldindex.ValueType(vt), fieldindex.State(state)
		if !row.fills(c) {
			return nil
		}
		table, ok := row.valueType.SideTable()
		col, ok2 := valueColumn(row.valueType)
		if !ok || !ok2 {
			return fmt.Errorf("registry row has value type %q", vt)
		}

		after := uuid.Nil
		if cursor != nil {
			after = *cursor
		}
		rows, err := tx.Query(ctx, nextIDsSQL, c.tenant, row.entityDefinition, after, size)
		if err != nil {
			return fmt.Errorf("next records: %w", err)
		}
		ids, err := pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
		if err != nil {
			return fmt.Errorf("next records: %w", err)
		}
		if len(ids) == 0 {
			return nil
		}
		if _, err := tx.Exec(ctx, fmt.Sprintf(deleteSideSQL, table), c.tenant, c.field, ids); err != nil {
			return fmt.Errorf("delete side rows: %w", err)
		}
		if _, err := tx.Exec(ctx, fmt.Sprintf(insertSideSQL, table, col), c.tenant, c.field, ids, row.key, vt); err != nil {
			return fmt.Errorf("insert side rows: %w", err)
		}

		last := ids[len(ids)-1]
		var done, total int64
		if err := tx.QueryRow(ctx, progressSQL, c.tenant, row.entityDefinition, last).Scan(&done, &total); err != nil {
			return fmt.Errorf("progress: %w", err)
		}
		if _, err := tx.Exec(ctx, setProgressSQL, c.tenant, c.field, progressPercent(done, total)); err != nil {
			return fmt.Errorf("set progress: %w", err)
		}
		if len(ids) == size {
			next = &last
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return next, nil
}

// progressPercent is done/total as a whole percentage, at most 99: only the
// flip to ready writes 100 (D10.6).
func progressPercent(done, total int64) int16 {
	if total <= 0 || done <= 0 {
		return 0
	}
	p := done * 100 / total
	if p > 99 {
		p = 99
	}
	return int16(p)
}

// flipReady flips c's row to ready, progress 100, while it is backfilling at
// c's pinned version (D10.6). Otherwise it changes nothing.
func flipReady(ctx context.Context, pool *pgxpool.Pool, c child) error {
	return tenancy.InTenantTx(ctx, pool, c.tenant, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, readySQL, c.tenant, c.field, c.version)
		return err
	})
}

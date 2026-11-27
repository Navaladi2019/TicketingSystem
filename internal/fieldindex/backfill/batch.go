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
//	(d) progress, a whole percentage below 100, on the registry row. The
//	    records of the type are counted once per child per worker (progressSQL,
//	    an index-only scan of the whole type), then carried in the worker's
//	    progress memo with the done count: a batch adds its own records to it
//	    instead of counting the type again (TICK-991: 159 ms a batch at 400k,
//	    O(N²) over a backfill). The cursor cannot carry the counts: activity
//	    arguments and results are references only (CI-7).
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
	// insertSideSQL inserts the side rows and returns how many of the
	// batch's values it skipped because they did not coerce: the non-null
	// values (a top-level array's non-null elements, else 1 for a non-null
	// scalar) minus the rows written, as the records trigger counts them
	// (db/migrations/20261127110100_field_index_trigger.sql). Only that
	// integer leaves the database.
	insertSideSQL = `WITH ins AS (
			INSERT INTO %[1]s (tenant_id, field_id, record_id, ord, value)
			SELECT r.tenant_id, $2, r.id, c.ord, c.%[2]s
			FROM records r CROSS JOIN LATERAL record_field_coerce(r.data -> $4::text, $5::text) c
			WHERE r.tenant_id = $1 AND r.id = ANY($3::uuid[])
			RETURNING 1
		), vals AS (
			SELECT CASE
				WHEN r.data -> $4::text IS NULL OR jsonb_typeof(r.data -> $4::text) = 'null' THEN 0
				WHEN jsonb_typeof(r.data -> $4::text) = 'array' THEN
					(SELECT count(*) FROM jsonb_array_elements(r.data -> $4::text) x WHERE jsonb_typeof(x) <> 'null')
				ELSE 1 END AS n
			FROM records r WHERE r.tenant_id = $1 AND r.id = ANY($3::uuid[])
		)
		SELECT coalesce((SELECT sum(n) FROM vals), 0)::bigint - (SELECT count(*) FROM ins)`
	// progressSQL counts the records of the type up to the batch's last id
	// and in all: run once per child per worker, on a progress memo miss.
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

// batchResult is what one committed batch reports to its activity: the
// next cursor, the count of values it skipped because they did not coerce,
// with their entity definition, for nava_field_index_uncoerced_total, and
// the progress counts at its last record. Only the cursor goes into the
// activity result (CI-5c): the rest are a metric and the worker's memo,
// never a payload.
type batchResult struct {
	next             *uuid.UUID
	entityDefinition uuid.UUID
	uncoerced        int64
	progress         *progressCounts
}

// progressCounts are a child's progress counts after a batch: last, the
// batch's last record id; done, the type's records up to it; total, the
// type's records when they were counted.
type progressCounts struct {
	last        uuid.UUID
	done, total int64
}

// after returns the counts after a batch that copied n records ending at
// last, from known, the counts at the batch's cursor, when known is for that
// cursor. Otherwise it returns false and the batch counts the type.
func (known *progressCounts) after(cursor *uuid.UUID, last uuid.UUID, n int) (progressCounts, bool) {
	if known == nil || cursor == nil || known.last != *cursor {
		return progressCounts{}, false
	}
	return progressCounts{last: last, done: known.done + int64(n), total: known.total}, true
}

// copyBatch runs one batch of c's field after cursor (nil: from the start)
// and returns the new cursor, or nil when the field is done or no longer
// c's to fill, with the batch's uncoerced count and progress counts. known
// are the counts at cursor from an earlier batch, or nil; when they do not
// match cursor the batch counts the type. Nothing is written when it
// returns nil without copying.
func copyBatch(ctx context.Context, pool *pgxpool.Pool, c child, cursor *uuid.UUID, size int, known *progressCounts) (batchResult, error) {
	var res batchResult
	err := tenancy.InTenantTx(ctx, pool, c.tenant, func(tx pgx.Tx) error {
		res = batchResult{}
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
		var uncoerced int64
		if err := tx.QueryRow(ctx, fmt.Sprintf(insertSideSQL, table, col), c.tenant, c.field, ids, row.key, vt).Scan(&uncoerced); err != nil {
			return fmt.Errorf("insert side rows: %w", err)
		}
		res.entityDefinition, res.uncoerced = row.entityDefinition, uncoerced

		last := ids[len(ids)-1]
		pc, ok := known.after(cursor, last, len(ids))
		if !ok {
			pc.last = last
			if err := tx.QueryRow(ctx, progressSQL, c.tenant, row.entityDefinition, last).Scan(&pc.done, &pc.total); err != nil {
				return fmt.Errorf("progress: %w", err)
			}
		}
		res.progress = &pc
		if _, err := tx.Exec(ctx, setProgressSQL, c.tenant, c.field, progressPercent(pc.done, pc.total)); err != nil {
			return fmt.Errorf("set progress: %w", err)
		}
		if len(ids) == size {
			res.next = &last
		}
		return nil
	})
	if err != nil {
		return batchResult{}, err
	}
	return res, nil
}

// progressPercent is done/total as a whole percentage, at most 99: only the
// flip to ready writes 100 (D10.6). done can pass total when records are
// created after the type was counted; the cap holds then too.
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

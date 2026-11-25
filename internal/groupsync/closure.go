package groupsync

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// TICK-1033 (S31 fix; ADR-010 D7 as amended by ADR-010-A2): group sync rebuilds
// the tenant's org_unit_closure in the same transaction as the org-tree write,
// so a reader never sees a tree and a closure that disagree.
//
// closureCTE is the one rebuild query. The migration's backfill function
// org_unit_closure_rebuild_all() (20261121050000_org_unit_closure.sql) holds a
// copy, because a migration is frozen once it ships; TestClosureQueryMatchesMigrationBackfill
// proves the two give the same rows. $1 is the tenant id.
//
// live: units whose whole path from a root is live. A unit hangs under one
// parent, so a walk down from the roots is a tree and ends; a unit on a parent
// cycle has no root above it and is never reached. closure: from each live
// unit, down through live children.
const closureCTE = `WITH RECURSIVE live (id) AS (
	SELECT u.id FROM org_units u
	WHERE u.tenant_id = $1::uuid AND u.parent_id IS NULL AND u.deleted_at IS NULL
	UNION ALL
	SELECT u.id FROM live l
	JOIN org_units u ON u.tenant_id = $1::uuid AND u.parent_id = l.id
	WHERE u.deleted_at IS NULL
), closure (ancestor_id, descendant_id) AS (
	SELECT id, id FROM live
	UNION ALL
	SELECT c.ancestor_id, u.id FROM closure c
	JOIN org_units u ON u.tenant_id = $1::uuid AND u.parent_id = c.descendant_id
	WHERE u.deleted_at IS NULL
) `

const (
	closureInsertSQL = `INSERT INTO org_unit_closure (tenant_id, ancestor_id, descendant_id) ` +
		closureCTE + `SELECT $1::uuid, ancestor_id, descendant_id FROM closure`

	// closureUnrootedSQL (TICK-1058) counts the tenant's live units with no
	// (u, u) row: one anti-join, each unit probing org_unit_closure_pkey. Such
	// a unit (an orphan, or one on a parent cycle) is outside every region
	// since TICK-1034: under never grants for it and not under admits it.
	closureUnrootedSQL = `SELECT count(*) FROM org_units u
	WHERE u.tenant_id = $1::uuid AND u.deleted_at IS NULL
	AND NOT EXISTS (SELECT 1 FROM org_unit_closure c
		WHERE c.tenant_id = u.tenant_id AND c.ancestor_id = u.id AND c.descendant_id = u.id)`
)

// unrootedUnits returns the number of the tenant's live org units that have no
// closure self row. It runs after every rebuildClosure call, rebuilt or not,
// and on every unchanged-snapshot sync: an orphan or a cycle written by hand
// leaves the stored and live closure counts equal, so a check run only on a
// rebuild would miss it.
func unrootedUnits(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID) (int, error) {
	var n int
	if err := tx.QueryRow(ctx, closureUnrootedSQL, tenantID.String()).Scan(&n); err != nil {
		return 0, fmt.Errorf("count units without a closure self row: %w", err)
	}
	return n, nil
}

// closureDriftSQL (TICK-1054) reports whether the stored closure and the live
// one differ as sets: a row in one and not the other, either way. Comparing row
// counts would miss a wrong row that replaces a right one.
const closureDriftSQL = closureCTE + `SELECT EXISTS (
	(SELECT ancestor_id, descendant_id FROM closure
	 EXCEPT SELECT ancestor_id, descendant_id FROM org_unit_closure WHERE tenant_id = $1::uuid)
	UNION ALL
	(SELECT ancestor_id, descendant_id FROM org_unit_closure WHERE tenant_id = $1::uuid
	 EXCEPT SELECT ancestor_id, descendant_id FROM closure))`

const (
	// closureAffectedSQL: the units whose closure rows can change when the
	// touched units ($2) were added, moved, revived or tombstoned: the touched
	// units, everything that was under them (old closure) and everything now
	// under them (live-or-not children, walked down; UNION ends a cycle).
	closureAffectedSQL = `WITH RECURSIVE seed (id) AS (
	SELECT unnest($2::uuid[])
	UNION SELECT c.descendant_id FROM org_unit_closure c
	WHERE c.tenant_id = $1::uuid AND c.ancestor_id = ANY($2::uuid[])
), d (id) AS (
	SELECT id FROM seed
	UNION SELECT u.id FROM d JOIN org_units u ON u.tenant_id = $1::uuid AND u.parent_id = d.id
	WHERE u.deleted_at IS NULL
) SELECT id FROM d`

	// closureDeltaInsertSQL writes the rows of the affected units ($2): for each
	// live one, walk up through live parents; it has rows only if the walk
	// reaches a root. Same rows as closureCTE for these descendants.
	closureDeltaInsertSQL = `WITH RECURSIVE up (descendant_id, ancestor_id) AS (
	SELECT u.id, u.id FROM org_units u
	WHERE u.tenant_id = $1::uuid AND u.id = ANY($2::uuid[]) AND u.deleted_at IS NULL
	UNION
	SELECT up.descendant_id, p.id FROM up
	JOIN org_units c ON c.tenant_id = $1::uuid AND c.id = up.ancestor_id
	JOIN org_units p ON p.tenant_id = $1::uuid AND p.id = c.parent_id
	WHERE p.deleted_at IS NULL
), rooted AS (
	SELECT DISTINCT up.descendant_id FROM up
	JOIN org_units r ON r.tenant_id = $1::uuid AND r.id = up.ancestor_id AND r.parent_id IS NULL
)
INSERT INTO org_unit_closure (tenant_id, ancestor_id, descendant_id)
SELECT $1::uuid, up.ancestor_id, up.descendant_id FROM up
WHERE up.descendant_id IN (SELECT descendant_id FROM rooted)`
)

func closureRowCount(ctx context.Context, tx pgx.Tx, tid string) (int, error) {
	var n int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM org_unit_closure WHERE tenant_id = $1::uuid`, tid).Scan(&n); err != nil {
		return 0, fmt.Errorf("count closure: %w", err)
	}
	return n, nil
}

// rebuildClosure makes the tenant's closure equal its live tree, in tx.
//
// touched holds the ids of the units the run added, moved, revived or
// tombstoned (nil when none). With touched units and a non-empty stored
// closure it writes a delta: only the rows of the touched units' subtrees are
// replaced. With none it checks the stored closure against the live one as
// sets and, on any difference, repairs it by a full rebuild (that repairs a
// closure left stale before this code shipped, or by hand). An empty stored
// closure (first import) also takes the full rebuild. A run that touched units
// does not check the rest of the closure for drift; the next run that touches
// none does. It returns the closure's
// row count after the call, whether it wrote any row, and whether it repaired
// drift (TICK-1124): true only when the set check found the stored closure
// different from the live one. A delta, a first import and a clean check are
// not drift.
func rebuildClosure(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID, touched []string) (rows int, rebuilt, repaired bool, err error) {
	tid := tenantID.String()
	have, err := closureRowCount(ctx, tx, tid)
	if err != nil {
		return 0, false, false, err
	}
	if len(touched) == 0 && have == 0 {
		// Empty stored closure: nothing to compare, rebuild without the set check.
		// A run that touched no unit is not a first import (a first import adds
		// units), so a closure that is empty while the tenant has live units is
		// drift. A tenant with no live units writes no row and reports none.
		rows, rebuilt, err = fullRebuildClosure(ctx, tx, tid)
		return rows, rebuilt, err == nil && rows > 0, err
	}
	if len(touched) == 0 {
		var drift bool
		if err := withHashPlan(ctx, tx, func() error {
			return tx.QueryRow(ctx, closureDriftSQL, tid).Scan(&drift)
		}); err != nil {
			return 0, false, false, fmt.Errorf("compare closure: %w", err)
		}
		if !drift {
			return have, false, false, nil
		}
		rows, rebuilt, err = fullRebuildClosure(ctx, tx, tid)
		return rows, rebuilt, err == nil, err
	}
	if have == 0 {
		// Not counted: a first import and a closure wiped in the same run as a
		// unit change look the same here. Accepted limit (TICK-1124).
		rows, rebuilt, err = fullRebuildClosure(ctx, tx, tid)
		return rows, rebuilt, false, err
	}
	ar, err := tx.Query(ctx, closureAffectedSQL, tid, touched)
	if err != nil {
		return 0, false, false, fmt.Errorf("find affected units: %w", err)
	}
	var affected []string
	for ar.Next() {
		var id uuid.UUID
		if err := ar.Scan(&id); err != nil {
			ar.Close()
			return 0, false, false, fmt.Errorf("find affected units: %w", err)
		}
		affected = append(affected, id.String())
	}
	ar.Close()
	if err := ar.Err(); err != nil {
		return 0, false, false, fmt.Errorf("find affected units: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM org_unit_closure WHERE tenant_id = $1::uuid AND descendant_id = ANY($2::uuid[])`, tid, affected); err != nil {
		return 0, false, false, fmt.Errorf("clear affected closure rows: %w", err)
	}
	if _, err := tx.Exec(ctx, closureDeltaInsertSQL, tid, affected); err != nil {
		return 0, false, false, fmt.Errorf("write closure delta: %w", err)
	}
	rows, err = closureRowCount(ctx, tx, tid)
	return rows, true, false, err
}

// withHashPlan runs fn with nested loops off, in tx, then turns them back on.
// TICK-1054: on a table with no planner statistics (a bulk import before
// autovacuum has run) the planner estimates one row for org_units, puts it on
// the outer side of the recursive step and scans the work table once per unit:
// 45 to 98 s at 10,111 units. The sync role cannot ANALYZE (only the table
// owner can on Postgres 16, and it would lock the table against another
// tenant's rebuild), so the plan is made independent of the estimates: with
// nested loops off the join is a hash join over the work table, linear in the
// rows. Used for the full-rebuild insert and the drift check, which walk the same tree. SET LOCAL ends with the transaction.
func withHashPlan(ctx context.Context, tx pgx.Tx, fn func() error) error {
	if _, err := tx.Exec(ctx, "SET LOCAL enable_nestloop = off"); err != nil {
		return fmt.Errorf("set enable_nestloop: %w", err)
	}
	if err := fn(); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, "SET LOCAL enable_nestloop = on"); err != nil {
		return fmt.Errorf("reset enable_nestloop: %w", err)
	}
	return nil
}

// fullRebuildClosure deletes the tenant's closure and re-inserts it from the
// live tree: the first import and the drift repair path.
func fullRebuildClosure(ctx context.Context, tx pgx.Tx, tid string) (rows int, rebuilt bool, err error) {
	if _, err := tx.Exec(ctx, `DELETE FROM org_unit_closure WHERE tenant_id = $1::uuid`, tid); err != nil {
		return 0, false, fmt.Errorf("clear closure: %w", err)
	}
	err = withHashPlan(ctx, tx, func() error {
		tag, err := tx.Exec(ctx, closureInsertSQL, tid)
		if err != nil {
			return fmt.Errorf("rebuild closure: %w", err)
		}
		rows = int(tag.RowsAffected())
		return nil
	})
	return rows, err == nil, err
}

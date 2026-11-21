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
	closureExpectedSQL = closureCTE + `SELECT count(*) FROM closure`

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

// rebuildClosure makes the tenant's closure equal its live tree, in tx. It
// runs when the run changed the tree (treeChanged), and also when the stored
// row count disagrees with the live tree's: that repairs a closure left stale
// before this code shipped (the TICK-1032 migration to this task) or by hand.
// It returns the closure's row count after the call.
func rebuildClosure(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID, treeChanged bool) (rows int, rebuilt bool, err error) {
	tid := tenantID.String()
	var want, have int
	if !treeChanged {
		if err := tx.QueryRow(ctx, closureExpectedSQL, tid).Scan(&want); err != nil {
			return 0, false, fmt.Errorf("count live closure: %w", err)
		}
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM org_unit_closure WHERE tenant_id = $1::uuid`, tid).Scan(&have); err != nil {
			return 0, false, fmt.Errorf("count closure: %w", err)
		}
		if want == have {
			return have, false, nil
		}
	}
	if _, err := tx.Exec(ctx, `DELETE FROM org_unit_closure WHERE tenant_id = $1::uuid`, tid); err != nil {
		return 0, false, fmt.Errorf("clear closure: %w", err)
	}
	tag, err := tx.Exec(ctx, closureInsertSQL, tid)
	if err != nil {
		return 0, false, fmt.Errorf("rebuild closure: %w", err)
	}
	return int(tag.RowsAffected()), true, nil
}

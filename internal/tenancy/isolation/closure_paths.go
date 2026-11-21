package isolation

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// closurePaths are the org_unit_closure writes of group sync, its only
// writer (TICK-1035, deferred from the CTO's TICK-1042 review; ADR-010-A2).
// Each checks one exact outcome, not Denied's "0 rows, 42501 or class 23",
// so that a change in which layer refuses is seen (TICK-1074):
//
//  1. tenant A's row with tenant B's unit as the ancestor passes the
//     WITH CHECK (its tenant_id is A) and must be refused by
//     org_unit_closure_ancestor_fkey with 23503;
//  2. tenant A's row with tenant B's unit as the descendant, the same, by
//     org_unit_closure_descendant_fkey;
//  3. a valid row of tenant A is written: the positive control of the
//     WITH CHECK path, so the others cannot pass on a broken connection,
//     role or tenant;
//  4. an UPDATE of both key columns of a closure row is refused with 42501:
//     there is no UPDATE grant. An UPDATE grant added later, table-wide or on
//     either column, even with no UPDATE policy (the UPDATE would then reach
//     0 rows and pass as Denied), turns it red.
//
// Paths 1 and 2 each go red when their FK is dropped: the other FK holds
// (its unit is tenant A's), so the row is written or refused by another
// constraint.
func closurePaths() []Path {
	return []Path{
		{Name: "org_unit_closure: group sync files tenant A's row under tenant B's unit as ancestor (A, B1, A1): the ancestor FK refuses it with 23503",
			Case: "CI-2", Role: "nava_groupsync", DB: closureForeignAncestor},
		{Name: "org_unit_closure: group sync hangs tenant B's unit under tenant A's (A, A1, B2): the descendant FK refuses it with 23503",
			Case: "CI-2", Role: "nava_groupsync", DB: closureForeignDescendant},
		{Name: "org_unit_closure: group sync writes its own valid tenant A row (positive control of the WITH CHECK path)",
			Case: "CI-2", Role: "nava_groupsync", DB: closureOwnInsert},
		{Name: "org_unit_closure: group sync updates both key columns of a closure row: refused with 42501 (no UPDATE grant, table or column)",
			Case: "CI-2", Role: "nava_groupsync", DB: closureUpdateRefused},
	}
}

// closureControl deletes tenant A's (A1, A2) row and writes it again, as the
// rebuild does, in its own savepoint of tx.
func closureControl(ctx context.Context, tx pgx.Tx) error {
	n, err := execOne(ctx, tx, "DELETE FROM org_unit_closure WHERE tenant_id = $1 AND ancestor_id = $2 AND descendant_id = $3",
		TenantA, OrgUnitA1, OrgUnitA2)
	if err == nil && n == 1 {
		n, err = execOne(ctx, tx, "INSERT INTO org_unit_closure (tenant_id, ancestor_id, descendant_id) VALUES ($1, $2, $3)",
			TenantA, OrgUnitA1, OrgUnitA2)
	}
	switch {
	case err != nil:
		return fmt.Errorf("control: group sync rewrites tenant A's (A1, A2) row: %w", err)
	case n != 1:
		return fmt.Errorf("control: group sync rewrote %d of tenant A's (A1, A2) rows, want 1", n)
	}
	return nil
}

func closureForeignAncestor(ctx context.Context, c Cluster) error {
	return closureForeignKey(ctx, c, "(A, B1, A1)", OrgUnitB1, OrgUnitA1, "org_unit_closure_ancestor_fkey")
}

func closureForeignDescendant(ctx context.Context, c Cluster) error {
	return closureForeignKey(ctx, c, "(A, A1, B2)", OrgUnitA1, OrgUnitB2, "org_unit_closure_descendant_fkey")
}

// closureForeignKey writes tenant A's (ancestor, descendant) row, one of
// which is tenant B's unit, after the control, and wants 23503 from fkey.
func closureForeignKey(ctx context.Context, c Cluster, row, ancestor, descendant, fkey string) error {
	pool, err := c.Pool("nava_groupsync")
	if err != nil {
		return err
	}
	return inTenant(ctx, pool, TenantA, func(tx pgx.Tx) error {
		if err := closureControl(ctx, tx); err != nil {
			return err
		}
		_, err := execOne(ctx, tx, "INSERT INTO org_unit_closure (tenant_id, ancestor_id, descendant_id) VALUES ($1, $2, $3)",
			TenantA, ancestor, descendant)
		switch {
		case err == nil:
			return fmt.Errorf("%s was written", row)
		case sqlState(err) != "23503" || constraintName(err) != fkey:
			return fmt.Errorf("%s: %w (constraint %q), want SQLSTATE 23503 from %s", row, err, constraintName(err), fkey)
		}
		return nil
	})
}

func closureOwnInsert(ctx context.Context, c Cluster) error {
	pool, err := c.Pool("nava_groupsync")
	if err != nil {
		return err
	}
	return inTenant(ctx, pool, TenantA, func(tx pgx.Tx) error {
		if err := closureControl(ctx, tx); err != nil {
			return err
		}
		var tenant string
		if err := tx.QueryRow(ctx, "SELECT tenant_id::text FROM org_unit_closure WHERE tenant_id = $1 AND ancestor_id = $2 AND descendant_id = $3",
			TenantA, OrgUnitA1, OrgUnitA2).Scan(&tenant); err != nil {
			return fmt.Errorf("read back the written row: %w", err)
		}
		if tenant != TenantA {
			return fmt.Errorf("the written row belongs to %s", tenant)
		}
		return nil
	})
}

func closureUpdateRefused(ctx context.Context, c Cluster) error {
	pool, err := c.Pool("nava_groupsync")
	if err != nil {
		return err
	}
	return inTenant(ctx, pool, TenantA, func(tx pgx.Tx) error {
		if err := closureControl(ctx, tx); err != nil {
			return err
		}
		_, err := execOne(ctx, tx, "UPDATE org_unit_closure SET ancestor_id = ancestor_id, descendant_id = descendant_id WHERE tenant_id = $1 AND ancestor_id = $2",
			TenantA, OrgUnitA1)
		return refused("group sync updates tenant A's closure rows", err)
	})
}

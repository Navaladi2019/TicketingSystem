package isolation

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// closurePaths are the org_unit_closure writes of group sync, its only
// writer (TICK-1035, deferred from the CTO's TICK-1042 review; ADR-010-A2).
// Each checks one exact outcome, not Denied's "0 rows, 42501 or class 23",
// so that a change in which layer refuses is seen:
//
//  1. tenant A's row with tenant B's unit as the ancestor passes the
//     WITH CHECK (its tenant_id is A) and must be refused by the ancestor FK
//     with 23503;
//  2. a valid row of tenant A is written: the positive control of the
//     WITH CHECK path, so the other two cannot pass on a broken connection,
//     role or tenant;
//  3. an UPDATE of a closure row is refused with 42501: there is no UPDATE
//     grant. An UPDATE grant added later, even with no UPDATE policy (the
//     UPDATE would then reach 0 rows and pass as Denied), turns it red.
func closurePaths() []Path {
	return []Path{
		{Name: "org_unit_closure: group sync files tenant A's row under tenant B's unit as ancestor (A, B1, A1): the ancestor FK refuses it with 23503",
			Case: "CI-2", Role: "nava_groupsync", DB: closureForeignAncestor},
		{Name: "org_unit_closure: group sync writes its own valid tenant A row (positive control of the WITH CHECK path)",
			Case: "CI-2", Role: "nava_groupsync", DB: closureOwnInsert},
		{Name: "org_unit_closure: group sync updates a closure row: refused with 42501 (no UPDATE grant)",
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
	pool, err := c.Pool("nava_groupsync")
	if err != nil {
		return err
	}
	return inTenant(ctx, pool, TenantA, func(tx pgx.Tx) error {
		if err := closureControl(ctx, tx); err != nil {
			return err
		}
		_, err := execOne(ctx, tx, "INSERT INTO org_unit_closure (tenant_id, ancestor_id, descendant_id) VALUES ($1, $2, $3)",
			TenantA, OrgUnitB1, OrgUnitA1)
		switch code := sqlState(err); {
		case err == nil:
			return errors.New("(A, B1, A1) was written")
		case code != "23503":
			return fmt.Errorf("(A, B1, A1): %w, want SQLSTATE 23503 from org_unit_closure_ancestor_fkey", err)
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
		_, err := execOne(ctx, tx, "UPDATE org_unit_closure SET descendant_id = descendant_id WHERE tenant_id = $1 AND ancestor_id = $2",
			TenantA, OrgUnitA1)
		return refused("group sync updates tenant A's closure rows", err)
	})
}

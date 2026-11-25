package isolation

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// orgUnitFKPaths insert tenant A's record into tenant B's unit, one path per
// role that may INSERT records (TICK-1126). The row's tenant_id is A, so it
// passes records_insert's WITH CHECK and only records_org_unit_fkey
// (tenant_id, org_unit_id) can refuse it. Each path wants exactly 23503 from
// that constraint, not Denied, which also accepts 42501 and would stay green
// on a lost INSERT grant without reaching the FK. The UPDATE path in Paths
// went silent that way (TICK-1120).
//
// The control is the same INSERT, same id included, into tenant A's own unit:
// it writes 1 row, so no other constraint, grant or policy refuses the shape.
// It runs second, so that its row cannot refuse the first with 23505.
// records_audit_commit_check is deferred to COMMIT and never fires: the
// transaction is rolled back.
func orgUnitFKPaths() []Path {
	path := func(role string) Path {
		return Path{
			Name: "records: " + role + " inserts tenant A's record in tenant B's unit: records_org_unit_fkey refuses it with 23503",
			Case: "CI-2", Role: role,
			DB: func(ctx context.Context, c Cluster) error { return orgUnitForeignInsert(ctx, c, role) },
		}
	}
	return []Path{path("nava_app"), path("nava_worker")}
}

func orgUnitForeignInsert(ctx context.Context, c Cluster, role string) error {
	const insert = "INSERT INTO records (tenant_id, id, org_unit_id) VALUES ($1, $2, $3)"
	const fkey = "records_org_unit_fkey"
	pool, err := c.Pool(role)
	if err != nil {
		return err
	}
	return inTenant(ctx, pool, TenantA, func(tx pgx.Tx) error {
		_, err := execOne(ctx, tx, insert, TenantA, NewID, OrgUnitB2)
		switch {
		case err == nil:
			return fmt.Errorf("%s wrote tenant A's record in tenant B's unit", role)
		case sqlState(err) != "23503" || constraintName(err) != fkey:
			return fmt.Errorf("%s inserts tenant A's record in tenant B's unit: %w (constraint %q), want SQLSTATE 23503 from %s",
				role, err, constraintName(err), fkey)
		}
		n, err := execOne(ctx, tx, insert, TenantA, NewID, OrgUnitA1)
		switch {
		case err != nil:
			return fmt.Errorf("control: %s inserts tenant A's record in its own unit: %w", role, err)
		case n != 1:
			return fmt.Errorf("control: %s inserted %d of tenant A's records in its own unit, want 1", role, n)
		}
		return nil
	})
}

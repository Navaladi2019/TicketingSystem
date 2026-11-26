//go:build integration

package dbtest

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/tenantlimits"
)

// Adversarial (Security & Multi-Tenancy Reviewer, PR #921, TICK-751): only
// the operator may raise a tenant's filterable-field limit. A tenant-scoped
// nava_app transaction, which is what every tenant route on nava-api opens,
// must not be able to raise its own tenant's limit. The operator route's
// authentication is not enough: the guard belongs to the role, so a future
// tenant route, a bug or an injection that reaches tenant_limits as nava_app
// is refused by PostgreSQL.
//
// The attack runs the same audited write the operator route runs
// (tenantlimits.SetPerType, with its audit entry), as nava_app in tenant A's
// own transaction, aiming at tenant A's own row. Want 42501.
//
// Fails today (AR-TICK-916-01): nava_app holds INSERT and UPDATE on
// tenant_limits. Remove the skip in the change that moves the write to
// nava_onboard.
func TestSecTenantLimitsTenantRoleCannotRaiseItsOwnLimit(t *testing.T) {
	t.Skip("AR-TICK-916-01: nava_app can still write tenant_limits; unskip with the fix")
	migratedPlatform(t)
	super := superNava(t)
	seedTenants(t, super, []string{tenantA}, nil)
	app := asRole(t, "nava_app")

	err := inTenantTx(app, tenantA, func(tx pgx.Tx) error {
		_, err := tenantlimits.SetPerType(context.Background(), tx, uuid.MustParse(tenantA), tenantlimits.MaxPerType,
			"self-service raise", "tenant-a-admin")
		return err
	})
	wantSQLState(t, "nava_app in tenant A raises tenant A's own limit through the audited write", err, "42501")
	if got := queryString(t, super, `SELECT count(*)::text FROM tenant_limits WHERE tenant_id = $1`, tenantA); got != "0" {
		t.Errorf("tenant A's tenant_limits rows after its own raise = %s, want 0", got)
	}

	// The plain statements, without the audit entry, are refused at the
	// statement, not only by the deferred commit check.
	_, err = execIn(app, tenantA, `INSERT INTO tenant_limits (tenant_id, filterable_fields_per_type, updated_by) VALUES ($1, 10, 'x')`, tenantA)
	wantSQLState(t, "nava_app in tenant A inserts tenant A's limits row", err, "42501")
}

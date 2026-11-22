package recordports

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/internal/authz"
	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
	"github.com/Navaladi2019/TicketingSystem/internal/workerguard"
	"github.com/Navaladi2019/TicketingSystem/internal/workflow/interp"
)

// PolicyTester is interp.PolicyTester over authz.TestPolicyRef (ADR-010 D9,
// TICK-957). It never touches Pool outside a tenant transaction: the one
// transaction it opens is keyed on the stored tenant_id of the calling run's
// workflow_ref row, and the policy lookup, the record and the principal
// resolution all happen in it.
type PolicyTester struct {
	Pool     *pgxpool.Pool
	Policies authz.PolicyReader
}

var _ interp.PolicyTester = PolicyTester{}

// runTenantSQL reads the stored tenant of the calling run. The tenant
// predicate is explicit as well as RLS's.
const runTenantSQL = `SELECT tenant_id FROM workflow_ref WHERE tenant_id = $1 AND outbox_id = $2 AND kind = $3`

// Test implements interp.PolicyTester. tenantID must be the activity's
// verified tenant, and the activity must belong to an interpreter run of
// that tenant (workerguard.RunRef); otherwise the call is ErrForbidden. The
// run's workflow_ref row is read first, in the transaction, and its stored
// tenant_id must equal the tenant the transaction is keyed on.
func (t PolicyTester) Test(ctx context.Context, tenantID uuid.UUID, in interp.PolicyTest) (bool, error) {
	if err := verified(ctx, tenantID); err != nil {
		return false, err
	}
	ref, err := workerguard.RunRef(ctx)
	if err != nil {
		return false, fmt.Errorf("%w: not an interpreter run of this tenant", interp.ErrForbidden)
	}
	var ok bool
	err = workerguard.InTenantTx(ctx, t.Pool, func(tx pgx.Tx) error {
		var stored uuid.UUID
		switch err := tx.QueryRow(ctx, runTenantSQL, tenantID, ref, outbox.InterpreterKind).Scan(&stored); {
		case errors.Is(err, pgx.ErrNoRows):
			return fmt.Errorf("%w: no workflow_ref for this run", interp.ErrForbidden)
		case err != nil:
			return fmt.Errorf("recordports: read run tenant: %w", err)
		case stored != tenantID:
			return fmt.Errorf("%w: run belongs to another tenant", interp.ErrForbidden)
		}
		var err error
		ok, err = authz.TestPolicyRef(ctx, tx, t.Policies, authz.PolicyTest{
			RecordID: in.RecordID, EntityType: in.EntityType, Policy: in.Policy, Who: in.Who, Op: in.Op,
		})
		return err
	})
	switch {
	case errors.Is(err, authz.ErrPolicyTestRecord):
		return false, fmt.Errorf("%w: record", interp.ErrNotFound)
	case err != nil:
		return false, err
	}
	return ok, nil
}

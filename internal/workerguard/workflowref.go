package workerguard

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"

	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// InTenantTx runs fn in a transaction scoped to the tenant the activity guard
// verified from the activity's nava-tenant header. Activities reach the
// database only through it. With no verified tenant on ctx (the guard did not
// run) it returns tenancy.ErrNoTenant and opens no transaction.
func InTenantTx(ctx context.Context, pool *pgxpool.Pool, fn func(pgx.Tx) error) error {
	tenant, ok := TenantFromContext(ctx)
	if !ok {
		return tenancy.ErrNoTenant
	}
	return tenancy.InTenantTx(ctx, pool, tenant, fn)
}

// CloseWorkflowRefActivity is the registered name of the activity that sets
// workflow_ref.closed_at.
const CloseWorkflowRefActivity = "nava.workflow_ref.Close"

// ErrTypeWorkflowRef is the ApplicationError type when the workflow has no
// workflow_ref row to close. It is not retryable.
const ErrTypeWorkflowRef = "nava.WorkflowRefNotFound"

// closeWorkflowRefSQL keeps the first closed_at, so a retried activity is a
// no-op that still finds its row. RLS limits it to the transaction's tenant.
const closeWorkflowRefSQL = `UPDATE workflow_ref SET closed_at = COALESCE(closed_at, now())
WHERE tenant_id = $1 AND outbox_id = $2 AND kind = $3`

// Activities are the workflow_ref activities. Register them on the worker with
// Register.
type Activities struct {
	Pool *pgxpool.Pool
}

// Register adds the activities to r under their fixed names.
func (a *Activities) Register(r worker.ActivityRegistry) {
	r.RegisterActivityWithOptions(a.CloseWorkflowRef, activity.RegisterOptions{Name: CloseWorkflowRefActivity})
}

// CloseWorkflowRef sets closed_at on the workflow_ref row of the workflow that
// scheduled it (ADR-001 A.3). The row is found from the workflow id, which the
// relay built as t:<tenant>:<kind>:<start outbox id>; nothing comes from an
// argument. It runs in InTenantTx with the header tenant.
func (a *Activities) CloseWorkflowRef(ctx context.Context) error {
	tenant, ok := TenantFromContext(ctx)
	if !ok {
		return temporal.NewNonRetryableApplicationError("workflow_ref: no verified tenant", ErrTypeTenantGuard, nil)
	}
	kind, outboxID, err := startRef(activity.GetInfo(ctx).WorkflowExecution.ID)
	if err != nil {
		return temporal.NewNonRetryableApplicationError("workflow_ref: "+err.Error(), ErrTypeWorkflowRef, nil)
	}
	var closed int64
	err = InTenantTx(ctx, a.Pool, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, closeWorkflowRefSQL, tenant, outboxID, kind)
		closed = tag.RowsAffected()
		return err
	})
	if err != nil {
		return err
	}
	if closed == 0 {
		return temporal.NewNonRetryableApplicationError("workflow_ref: no row for this workflow", ErrTypeWorkflowRef, nil)
	}
	return nil
}

// startRef splits a relay-built id t:<tenant>:<kind>:<outbox id> into the
// kind and the start row's outbox id. The kind may itself contain colons; the
// outbox id is the last segment.
func startRef(workflowID string) (kind string, outboxID uuid.UUID, err error) {
	if _, err := TenantOf(workflowID); err != nil {
		return "", uuid.Nil, err
	}
	_, targetKey, _ := strings.Cut(strings.TrimPrefix(workflowID, workflowIDPrefix), ":")
	i := strings.LastIndexByte(targetKey, ':')
	if i <= 0 {
		return "", uuid.Nil, errors.New("workflow id has no <kind>:<outbox id> target key")
	}
	outboxID, err = uuid.Parse(targetKey[i+1:])
	if err != nil {
		return "", uuid.Nil, errors.New("workflow id target key does not end in an outbox id")
	}
	return targetKey[:i], outboxID, nil
}

// CloseWorkflowRef runs the CloseWorkflowRefActivity. A relay-started workflow
// calls it as its final activity, before it returns.
func CloseWorkflowRef(ctx workflow.Context) error {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Second,
	})
	return workflow.ExecuteActivity(ctx, CloseWorkflowRefActivity).Get(ctx, nil)
}

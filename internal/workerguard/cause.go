package workerguard

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"

	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// The causation rule of an interpreter write (Security condition on TICK-88,
// S11b contract D-3-29): origin and depth come only from the workflow_cause
// row the enqueuing transaction wrote for the delivery that resumed the run,
// read under RLS, and only if that row belongs to this run. The run is the
// one its workflow id names, never an argument.

var (
	// ErrCauseNotFound means the transaction's tenant has no workflow_cause
	// row with that outbox id, or none was named.
	ErrCauseNotFound = errors.New("workerguard: workflow cause not found")
	// ErrForeignCause means the workflow_cause row belongs to another run,
	// or the calling workflow is not a relay-started interpreter run.
	ErrForeignCause = errors.New("workerguard: workflow cause is not this run's")
)

// causeSQL reads one workflow_cause row by primary key. The tenant predicate
// is explicit as well as RLS's (TICK-480).
const causeSQL = `SELECT ref_outbox_id, causation_origin_id, causation_depth FROM workflow_cause
WHERE tenant_id = $1 AND outbox_id = $2`

// ResolveCause returns the causation of the next write the calling
// interpreter run makes: the origin of the workflow_cause row named by cause
// (interp.StepRef.Cause) and its depth plus one (A5-7). It reads the row in
// tx, which must be a tenant transaction of the activity's verified tenant,
// under RLS. The run's own ref comes from the activity's workflow id
// (t:<tenant>:nava.interpreter:<start outbox id>), as CloseWorkflowRef takes
// it; a row whose ref_outbox_id is not that ref is ErrForeignCause.
//
// It reads nothing else: not RunInput, not the Delivery, not any other
// StepRef field. A depth past the limit is returned as is; the pipeline
// refuses that write and audits it (cascade_too_deep).
//
// It must run inside an activity. The P3-43 RecordWriter adapter calls it.
func ResolveCause(ctx context.Context, tx pgx.Tx, cause uuid.UUID) (outbox.Causation, error) {
	return resolveCause(ctx, tx, activity.GetInfo(ctx).WorkflowExecution.ID, cause)
}

// resolveCause is ResolveCause for the workflow id workflowID.
func resolveCause(ctx context.Context, tx pgx.Tx, workflowID string, cause uuid.UUID) (outbox.Causation, error) {
	tenant, ok := TenantFromContext(ctx)
	if !ok {
		return outbox.Causation{}, tenancy.ErrNoTenant
	}
	txTenant, err := tenancy.TenantOf(ctx, tx)
	if err != nil {
		return outbox.Causation{}, fmt.Errorf("workerguard: %w", err)
	}
	if txTenant != tenant {
		return outbox.Causation{}, ErrForeignCause
	}
	kind, ref, err := startRef(workflowID)
	if err != nil || kind != outbox.InterpreterKind {
		return outbox.Causation{}, ErrForeignCause
	}
	if wfTenant, err := TenantOf(workflowID); err != nil || wfTenant != tenant {
		return outbox.Causation{}, ErrForeignCause
	}
	if cause == uuid.Nil {
		return outbox.Causation{}, ErrCauseNotFound
	}
	var (
		owner  uuid.UUID
		origin uuid.UUID
		depth  int16
	)
	switch err := tx.QueryRow(ctx, causeSQL, tenant, cause).Scan(&owner, &origin, &depth); {
	case errors.Is(err, pgx.ErrNoRows):
		return outbox.Causation{}, ErrCauseNotFound
	case err != nil:
		return outbox.Causation{}, fmt.Errorf("workerguard: read workflow cause: %w", err)
	}
	if owner != ref {
		return outbox.Causation{}, ErrForeignCause
	}
	return outbox.Causation{OriginID: origin, Depth: depth + 1}, nil
}

// setWaitSQL sets or clears the run's record_changed wait. nava_worker may
// UPDATE this column only. A closed run is left alone.
const setWaitSQL = `UPDATE workflow_ref SET record_changed_signal_id = $3
WHERE tenant_id = $1 AND outbox_id = $2 AND kind = $4 AND closed_at IS NULL`

// WaitTracker is interp.WaitTracker over workflow_ref.record_changed_signal_id
// (D-3-29). The row is the calling run's, found from the activity's workflow
// id as CloseWorkflowRef finds it; the tenant is the activity's verified one.
type WaitTracker struct {
	Pool *pgxpool.Pool
}

// SetRecordChangedWait records signal (nil clears it) on the calling run's
// workflow_ref row. tenantID must be the activity's verified tenant. The
// signal id is stored as an opaque id. A run with no open row of its own
// fails non-retryably with ErrTypeWorkflowRef.
func (w WaitTracker) SetRecordChangedWait(ctx context.Context, tenantID uuid.UUID, signal *uuid.UUID) error {
	return w.setWait(ctx, activity.GetInfo(ctx).WorkflowExecution.ID, tenantID, signal)
}

func (w WaitTracker) setWait(ctx context.Context, workflowID string, tenantID uuid.UUID, signal *uuid.UUID) error {
	tenant, ok := TenantFromContext(ctx)
	if !ok || tenant != tenantID {
		return temporal.NewNonRetryableApplicationError("workflow_ref: no verified tenant", ErrTypeTenantGuard, nil)
	}
	kind, ref, err := startRef(workflowID)
	if err != nil || kind != outbox.InterpreterKind {
		return temporal.NewNonRetryableApplicationError("workflow_ref: not an interpreter run", ErrTypeWorkflowRef, nil)
	}
	var n int64
	err = InTenantTx(ctx, w.Pool, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, setWaitSQL, tenant, ref, signal, kind)
		n = tag.RowsAffected()
		return err
	})
	if err != nil {
		return err
	}
	if n == 0 {
		return temporal.NewNonRetryableApplicationError("workflow_ref: no open row for this workflow", ErrTypeWorkflowRef, nil)
	}
	return nil
}

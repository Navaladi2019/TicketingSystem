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
	// ErrStaleCause means the run has a newer workflow_cause row than the
	// one named: the named delivery is not the run's latest. A raw Temporal
	// signal that replays an old delivery id could otherwise reset the
	// cascade depth (Security review of P3-42, Low).
	ErrStaleCause = errors.New("workerguard: workflow cause is older than the run's newest")
)

// causeSQL reads one workflow_cause row by primary key. The tenant predicate
// is explicit as well as RLS's (TICK-480).
//
// newer is true when another row of the same run is later, by created_at and
// then by id: the run's newest row is the only one that may cause a write.
const causeSQL = `SELECT c.ref_outbox_id, c.causation_origin_id, c.causation_depth,
       EXISTS (SELECT 1 FROM workflow_cause n
               WHERE n.tenant_id = c.tenant_id AND n.ref_outbox_id = c.ref_outbox_id
                 AND (n.created_at, n.outbox_id) > (c.created_at, c.outbox_id)) AS newer
FROM workflow_cause c
WHERE c.tenant_id = $1 AND c.outbox_id = $2`

// ResolveCause returns the causation of the next write the calling
// interpreter run makes: the origin of the workflow_cause row named by cause
// (interp.StepRef.Cause) and its depth plus one (A5-7). It reads the row in
// tx, which must be a tenant transaction of the activity's verified tenant,
// under RLS. The run's own ref comes from the activity's workflow id
// (t:<tenant>:nava.interpreter:<start outbox id>), as CloseWorkflowRef takes
// it; a row whose ref_outbox_id is not that ref is ErrForeignCause, and a row
// older than the run's newest is ErrStaleCause.
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
		newer  bool
	)
	switch err := tx.QueryRow(ctx, causeSQL, tenant, cause).Scan(&owner, &origin, &depth, &newer); {
	case errors.Is(err, pgx.ErrNoRows):
		return outbox.Causation{}, ErrCauseNotFound
	case err != nil:
		return outbox.Causation{}, fmt.Errorf("workerguard: read workflow cause: %w", err)
	}
	if owner != ref {
		return outbox.Causation{}, ErrForeignCause
	}
	if newer {
		return outbox.Causation{}, ErrStaleCause
	}
	return outbox.Causation{OriginID: origin, Depth: depth + 1}, nil
}

// setWaitSQL sets or clears the run's record_changed wait. nava_worker may
// UPDATE this column only. A closed run is left alone.
const setWaitSQL = `UPDATE workflow_ref SET record_changed_signal_id = $3
WHERE tenant_id = $1 AND outbox_id = $2 AND kind = $4 AND closed_at IS NULL`

// ErrRecordMoved means the record changed after the walk read it, so the
// run's record_changed wait was not set. It is retryable: NextStep fails the
// attempt and the retry re-walks against the new version.
var ErrRecordMoved = errors.New("workerguard: record changed since the walk read it")

// recordVersionSQL locks the record against a concurrent write (a write holds
// FOR UPDATE until commit) and reads its version. Exactly one of the wait's
// UPDATE and that write's EventSink lookup sees the other: this FOR SHARE
// waits for the write to commit, and the wait it then sets is read by any
// later write's waitingSQL.
const recordVersionSQL = `SELECT version FROM records WHERE tenant_id = $1 AND id = $2 FOR SHARE`

// WaitTracker is interp.WaitTracker over workflow_ref.record_changed_signal_id
// (D-3-29). The row is the calling run's, found from the activity's workflow
// id as CloseWorkflowRef finds it; the tenant is the activity's verified one.
type WaitTracker struct {
	Pool *pgxpool.Pool
}

// SetRecordChangedWait records signal (nil clears it) on the calling run's
// workflow_ref row. tenantID must be the activity's verified tenant. A
// non-nil signal is set only if the record is still at version, the one the
// walk read, checked in the same transaction under FOR SHARE; if it moved the
// result is ErrRecordMoved (retryable). A clear needs no check. The signal id
// is stored as an opaque id. A run with no open row of its own fails
// non-retryably with ErrTypeWorkflowRef.
func (w WaitTracker) SetRecordChangedWait(ctx context.Context, tenantID, recordID uuid.UUID, signal *uuid.UUID, version int64) error {
	return w.setWait(ctx, activity.GetInfo(ctx).WorkflowExecution.ID, tenantID, recordID, signal, version)
}

func (w WaitTracker) setWait(ctx context.Context, workflowID string, tenantID, recordID uuid.UUID, signal *uuid.UUID, version int64) error {
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
		if signal != nil {
			var current int64
			switch err := tx.QueryRow(ctx, recordVersionSQL, tenant, recordID).Scan(&current); {
			case errors.Is(err, pgx.ErrNoRows):
				return ErrRecordMoved
			case err != nil:
				return fmt.Errorf("workerguard: lock record: %w", err)
			}
			if current != version {
				return ErrRecordMoved
			}
		}
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

// RunRef is the outbox id of the calling interpreter run's workflow_ref,
// taken from the activity's workflow id as ResolveCause and WaitTracker take
// it, never from an argument. ErrForeignCause if the workflow is not a
// relay-started interpreter run of the activity's verified tenant. It must run
// inside an activity.
func RunRef(ctx context.Context) (uuid.UUID, error) {
	tenant, ok := TenantFromContext(ctx)
	if !ok {
		return uuid.Nil, tenancy.ErrNoTenant
	}
	id := activity.GetInfo(ctx).WorkflowExecution.ID
	kind, ref, err := startRef(id)
	if err != nil || kind != outbox.InterpreterKind {
		return uuid.Nil, ErrForeignCause
	}
	if wfTenant, err := TenantOf(id); err != nil || wfTenant != tenant {
		return uuid.Nil, ErrForeignCause
	}
	return ref, nil
}

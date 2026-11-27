package backfill

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"

	"github.com/Navaladi2019/TicketingSystem/internal/fieldindex"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// FieldIndexRemove (ADR-006 D11, TICK-750) empties an unflagged field's side
// rows and deletes its registry row. The publish that drops the flag has
// already flipped the row to removing, in its own transaction (step 1); from
// then on the records trigger writes nothing for the field and a filter on it
// answers 400 field_not_filterable. The coordinator runs the workflow as a
// child, one at a time per tenant, so a removal never runs beside a backfill
// of the same tenant (PLAN-007 D-7-12):
//
//  2. the barrier (D9): a writer that read the row before the flip may still
//     write a side row for it; the barrier waits for every such transaction;
//  3. batches: each deletes up to BatchSize side rows of the field, in one
//     InTenantTx, until one deletes fewer;
//  4. and 5. Retire, in one InTenantTx: DELETE the registry row only if NOT
//     EXISTS a side row of the field, else back to 3. The required side-table
//     FKs refuse the DELETE too if a row is still there.
//
// Every step reads the field and its value type from the registry row, and
// the field and tenant from the workflow id: nothing an activity acts on comes
// from an argument. Each step changes nothing once the row is gone or no
// longer removing, so a retry, a replayed item or a second removal child is a
// no-op.

// The removal's activity names.
const (
	RemoveBatchActivity = "field_index.RemoveBatch"
	RetireActivity      = "field_index.Retire"
)

// Pending is RemoveBatch's and Retire's result: the field's id while work
// is left (a batch that deleted a full batch, or a Retire that found side rows
// still there), absent once the step is done. It is a reference, never a
// count or a record id (CI-5c), because the worker guard admits only UUIDs.
type Pending struct {
	Field *uuid.UUID `json:"pending,omitempty"`
}

// Remove is FieldIndexRemove. Zero fields take the defaults.
type Remove struct {
	// MaxBatchesPerRun is how many steps one run takes before it continues as
	// new; 0 is DefaultMaxBatchesPerRun.
	MaxBatchesPerRun int
}

// Register adds r.Run to reg as fieldindex.RemoveWorkflowType.
func (r *Remove) Register(reg worker.WorkflowRegistry) {
	reg.RegisterWorkflowWithOptions(r.Run, workflow.RegisterOptions{Name: fieldindex.RemoveWorkflowType})
}

func removeRun(ctx workflow.Context, in FieldRef) error { return (&Remove{}).Run(ctx, in) }

// Run removes one field's index. The child's workflow id, ChildID, names the
// field. A continued run runs the barrier again: it waits only for the
// transactions open at its own snapshot, which by then cannot have read the
// row as anything but removing, so it costs one poll.
func (r *Remove) Run(ctx workflow.Context, in FieldRef) error {
	c, err := parseChildID(workflow.GetInfo(ctx).WorkflowExecution.ID, fieldindex.RemoveWorkflowType)
	if err != nil || c.field != in.FieldID {
		return temporal.NewNonRetryableApplicationError("field index: input does not match the workflow id", ErrTypeNoChild, nil)
	}
	if err := barrier(ctx); err != nil {
		return err
	}
	maxSteps := r.MaxBatchesPerRun
	if maxSteps <= 0 {
		maxSteps = DefaultMaxBatchesPerRun
	}
	sctx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: batchStartToClose, RetryPolicy: retryForever(time.Minute)})
	for n := 0; ; n++ {
		if n == maxSteps {
			return workflow.NewContinueAsNewError(ctx, fieldindex.RemoveWorkflowType, FieldRef{FieldID: in.FieldID})
		}
		var p Pending
		if err := workflow.ExecuteActivity(sctx, RemoveBatchActivity).Get(sctx, &p); err != nil {
			return err
		}
		if p.Field != nil {
			continue
		}
		if err := workflow.ExecuteActivity(sctx, RetireActivity).Get(sctx, &p); err != nil {
			return err
		}
		if p.Field == nil {
			return nil
		}
	}
}

const (
	// removalRowSQL reads the child's registry row. A step acts only while
	// it is removing.
	removalRowSQL = `SELECT value_type, state FROM record_index_fields WHERE tenant_id = $1 AND id = $2`
	// removeBatchSQL deletes up to $3 of the field's side rows and counts
	// them. The subquery walks the side table's (tenant_id, field_id, ...)
	// value index.
	removeBatchSQL = `WITH d AS (
			DELETE FROM %[1]s WHERE tenant_id = $1 AND field_id = $2 AND (record_id, ord) IN (
				SELECT record_id, ord FROM %[1]s WHERE tenant_id = $1 AND field_id = $2 LIMIT $3)
			RETURNING 1)
		SELECT count(*) FROM d`
	// retireSQL deletes the registry row if the field has no side row left
	// (D11 steps 4 and 5 in one statement). A side row that a transaction
	// open across the flip still holds uncommitted makes the DELETE wait for
	// it, and then fail on the side table's FK: the barrier is what keeps that
	// from happening.
	retireSQL = `DELETE FROM record_index_fields WHERE tenant_id = $1 AND id = $2 AND state = 'removing'
		AND NOT EXISTS (SELECT 1 FROM %s WHERE tenant_id = $1 AND field_id = $2)`
)

// removalTable reads c's registry row in tx and returns its side table, or
// false when the row is gone or not removing: the step then does nothing.
func removalTable(ctx context.Context, tx pgx.Tx, c child) (string, bool, error) {
	var vt, state string
	err := tx.QueryRow(ctx, removalRowSQL, c.tenant, c.field).Scan(&vt, &state)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return "", false, nil
	case err != nil:
		return "", false, fmt.Errorf("read registry row: %w", err)
	case fieldindex.State(state) != fieldindex.StateRemoving:
		return "", false, nil
	}
	table, ok := fieldindex.ValueType(vt).SideTable()
	if !ok {
		return "", false, fmt.Errorf("registry row has value type %q", vt)
	}
	return table, true, nil
}

// removeBatch deletes up to size of c's side rows and reports whether it
// deleted a full batch, so more may be left.
func removeBatch(ctx context.Context, pool *pgxpool.Pool, c child, size int) (bool, error) {
	more := false
	err := tenancy.InTenantTx(ctx, pool, c.tenant, func(tx pgx.Tx) error {
		more = false
		table, ok, err := removalTable(ctx, tx, c)
		if err != nil || !ok {
			return err
		}
		var n int
		if err := tx.QueryRow(ctx, fmt.Sprintf(removeBatchSQL, table), c.tenant, c.field, size).Scan(&n); err != nil {
			return fmt.Errorf("delete side rows: %w", err)
		}
		more = n >= size
		return nil
	})
	return more, err
}

// retire deletes c's registry row when no side row is left, and reports
// whether side rows are still there. A row that is gone or not removing is
// done.
func retire(ctx context.Context, pool *pgxpool.Pool, c child) (left, deleted bool, err error) {
	err = tenancy.InTenantTx(ctx, pool, c.tenant, func(tx pgx.Tx) error {
		left, deleted = false, false
		table, ok, err := removalTable(ctx, tx, c)
		if err != nil || !ok {
			return err
		}
		tag, err := tx.Exec(ctx, fmt.Sprintf(retireSQL, table), c.tenant, c.field)
		if err != nil {
			return fmt.Errorf("delete registry row: %w", err)
		}
		deleted = tag.RowsAffected() == 1
		left = !deleted
		return nil
	})
	return left, deleted, err
}

// removalChildOf parses the calling workflow's id as a removal child's.
func removalChildOf(ctx context.Context) (child, error) {
	c, err := parseChildID(activity.GetInfo(ctx).WorkflowExecution.ID, fieldindex.RemoveWorkflowType)
	if err != nil {
		return child{}, temporal.NewNonRetryableApplicationError("field index: "+err.Error(), ErrTypeNoChild, nil)
	}
	return c, nil
}

// RemoveBatch is step 3: one batch of the child's side rows deleted, in one
// InTenantTx. Its result is Pending with the field while a full batch was
// deleted.
func (a *Activities) RemoveBatch(ctx context.Context) (Pending, error) {
	c, err := removalChildOf(ctx)
	if err != nil {
		return Pending{}, err
	}
	more, err := removeBatch(ctx, a.Pool, c, a.batchSize())
	if err != nil {
		return Pending{}, fmt.Errorf("field index remove batch: %w", err)
	}
	if more {
		return Pending{Field: &c.field}, nil
	}
	return Pending{}, nil
}

// Retire is steps 4 and 5: the registry row deleted if the field has no side
// row left. Its result is Pending with the field when side rows are still
// there, so the workflow goes back to RemoveBatch.
func (a *Activities) Retire(ctx context.Context) (Pending, error) {
	c, err := removalChildOf(ctx)
	if err != nil {
		return Pending{}, err
	}
	left, deleted, err := retire(ctx, a.Pool, c)
	if err != nil {
		return Pending{}, fmt.Errorf("field index retire: %w", err)
	}
	if deleted {
		a.logger().InfoContext(ctx, "field index removed", a.attrs(ctx)...)
	}
	if left {
		return Pending{Field: &c.field}, nil
	}
	return Pending{}, nil
}

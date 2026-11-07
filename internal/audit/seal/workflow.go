package seal

import (
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"

	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
)

// WorkflowType is the sealer's registered workflow type: the outbox kind the
// relay starts it as, on nava-tq, with id t:<tenant>:audit.seal:<o> (ADR-004
// R16, R18). The Authorizer allows only its start (TICK-438).
const WorkflowType = outbox.SealKind

// The activity bounds (TICK-440 review, "bounded A2"). A1 is one short
// transaction. A2 seals at most DefaultMaxBatches batches and heartbeats after
// each, so A2HeartbeatTimeout bounds one batch and A2StartToClose the call;
// an attempt on a worker that died is retried after A2HeartbeatTimeout, not
// A2StartToClose.
const (
	A1StartToClose     = 10 * time.Second
	A2StartToClose     = 5 * time.Minute
	A2HeartbeatTimeout = 30 * time.Second
)

// MaxAttempts bounds the retries of A1 and A2. The SDK default retries
// forever; with this a persistent retryable fault fails the run after MaxAttempts, about six
// minutes of backoff, and the next run (already committed by A1) tries
// again. NoRun and Unsealable are non-retryable and end the retries at once.
const MaxAttempts = 10

// RetryPolicy is the explicit RetryPolicy of A1 and A2.
func RetryPolicy() *temporal.RetryPolicy {
	return &temporal.RetryPolicy{
		InitialInterval:    time.Second,
		BackoffCoefficient: 2,
		MaximumInterval:    time.Minute,
		MaximumAttempts:    MaxAttempts,
	}
}

// RegisterWorkflow adds Workflow to r as WorkflowType.
func RegisterWorkflow(r worker.WorkflowRegistry) {
	r.RegisterWorkflowWithOptions(Workflow, workflow.RegisterOptions{Name: WorkflowType})
}

// Workflow is audit.seal, run o of the tenant's seal lineage. Its input is the
// relay's outbox.Delivery, which it does not read: A1 takes o from the
// workflow id, and the tenant comes from the guard (I2).
//
//   - A1 (RescheduleActivity) commits the next run's start row first, so a
//     run that fails afterwards, a poison batch included, still has a
//     successor (R16).
//   - A2 (SealActivity) then runs whatever A1 returned. A1 returns no o′ for
//     a stale lineage, and for a retry of an A1 that already committed; A2
//     runs either way, which R15's head lock makes safe when runs overlap
//     (TICK-440 review).
//
// It starts no child and never continues as new (TICK-438, TICK-572): the
// only way to the next run is A1's outbox row, which the relay starts and the
// head fences. The Authorizer cannot check that, so
// TestWorkflowStartsNoChildAndNeverContinuesAsNew does.
func Workflow(ctx workflow.Context, _ outbox.Delivery) error {
	a1 := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: A1StartToClose,
		RetryPolicy:         RetryPolicy(),
	})
	var next Rescheduled
	if err := workflow.ExecuteActivity(a1, RescheduleActivity).Get(a1, &next); err != nil {
		return err
	}
	a2 := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: A2StartToClose,
		HeartbeatTimeout:    A2HeartbeatTimeout,
		RetryPolicy:         RetryPolicy(),
	})
	return workflow.ExecuteActivity(a2, SealActivity).Get(a2, nil)
}

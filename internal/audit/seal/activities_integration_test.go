//go:build integration

package seal_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"

	"github.com/Navaladi2019/TicketingSystem/internal/audit/chain"
	"github.com/Navaladi2019/TicketingSystem/internal/audit/seal"
)

// sealRun is a stand-in for the P5-34b workflow: A1, then A2, no retries.
func sealRun(ctx workflow.Context) (seal.Rescheduled, error) {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute,
		RetryPolicy:         &temporal.RetryPolicy{MaximumAttempts: 1},
	})
	var r seal.Rescheduled
	if err := workflow.ExecuteActivity(ctx, seal.RescheduleActivity).Get(ctx, &r); err != nil {
		return r, err
	}
	return r, workflow.ExecuteActivity(ctx, seal.SealActivity).Get(ctx, nil)
}

// runSeal runs sealRun as workflow id with the registered activities, the
// activity context carrying tenant as the guard would set it.
func runSeal(t *testing.T, e itEnv, tenant uuid.UUID, id string) (seal.Rescheduled, error) {
	t.Helper()
	var s testsuite.WorkflowTestSuite
	env := s.NewTestWorkflowEnvironment()
	env.SetWorkerOptions(worker.Options{BackgroundActivityContext: asTenant(tenant)})
	env.SetStartWorkflowOptions(client.StartWorkflowOptions{ID: id})
	(&seal.Activities{Pool: e.worker}).Register(env)
	env.RegisterWorkflow(sealRun)
	env.ExecuteWorkflow(sealRun)
	if !env.IsWorkflowCompleted() {
		t.Fatal("the workflow did not complete")
	}
	var r seal.Rescheduled
	err := env.GetWorkflowError()
	if err == nil {
		err = env.GetWorkflowResult(&r)
	}
	return r, err
}

func runID(tenant, o uuid.UUID) string { return "t:" + tenant.String() + ":audit.seal:" + o.String() }

// The registered activities: A1 reads run o from the workflow id and returns
// o′ as a reference; A2 seals. A stale run returns no o′.
func TestActivities(t *testing.T) {
	e := migrated(t)
	tenant := newTenant(t, e)
	o := uuid.Must(uuid.NewV7())
	mustExec(t, e.super, "UPDATE audit_chain_head SET next_seal_outbox_id = $2 WHERE tenant_id = $1", tenant, o)
	writeEntries(t, e, tenant, 3)

	r, err := runSeal(t, e, tenant, runID(tenant, o))
	if err != nil {
		t.Fatal(err)
	}
	h := readHead(t, e, tenant)
	if r.NextOutboxID == nil || h.next == nil || *r.NextOutboxID != *h.next {
		t.Errorf("A1 returned %v, head points at %v", r.NextOutboxID, h.next)
	}
	if got := assertChain(t, e, tenant); got != 3 {
		t.Errorf("%d sealed, want 3", got)
	}

	// The same run again is a stale lineage: no o′, and A2 still runs.
	writeEntries(t, e, tenant, 1)
	r, err = runSeal(t, e, tenant, runID(tenant, o))
	if err != nil || r.NextOutboxID != nil {
		t.Errorf("stale run = %v, %v; want no o′", r.NextOutboxID, err)
	}
	if got := assertChain(t, e, tenant); got != 4 {
		t.Errorf("%d sealed, want 4", got)
	}
}

// A workflow id that is not a sealer run's fails A1 with the non-retryable
// nava.audit.seal.NoRun; an unsealable entry fails A2 with the non-retryable
// nava.audit.seal.Unsealable, whose message names no value.
func TestActivityErrors(t *testing.T) {
	e := migrated(t)
	tenant := newTenant(t, e)

	_, err := runSeal(t, e, tenant, "t:"+tenant.String()+":nava.interpreter:"+uuid.NewString())
	assertAppError(t, err, seal.ErrTypeNoRun)

	writeEntries(t, e, tenant, 1)
	mustExec(t, e.super, `INSERT INTO audit_log (tenant_id, id, domain, action, subject_type, subject_id,
		version_after, actor_system, source) VALUES ($1, $2, 'record', 'update', 'record', gen_random_uuid(), $3,
		'system', 'system')`, tenant, uuid.Must(uuid.NewV7()), int64(chain.MaxSafeInteger)+1)
	_, err = runSeal(t, e, tenant, runID(tenant, uuid.Must(uuid.NewV7())))
	assertAppError(t, err, seal.ErrTypeUnsealable)
	if got := count(t, e.super, "SELECT count(*) FROM audit_log WHERE tenant_id = $1 AND seq IS NOT NULL", tenant); got != 0 {
		t.Errorf("%d entries sealed past the poison batch, want 0", got)
	}
}

func assertAppError(t *testing.T, err error, typ string) {
	t.Helper()
	var app *temporal.ApplicationError
	if !errors.As(err, &app) || app.Type() != typ || !app.NonRetryable() {
		t.Fatalf("error %v, want a non-retryable %s", err, typ)
	}
}

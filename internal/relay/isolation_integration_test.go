//go:build integration

// Adversarial tenant-isolation tests for the relay's Temporal dispatch (T30b,
// security review of PR #96). The attacker controls what tenant A's outbox
// rows say, including a target_key naming tenant B's workflow. The relay must
// still put only outbox.tenant_id in the workflow id and the nava-tenant
// header, so nothing tenant A writes reaches tenant B's workflow.

package relay

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/internal/workerguard"
)

// A tenant A signal row whose target_key is tenant B's live workflow goes to
// t:<A>:<key>, which does not exist. It is marked dead, and B's workflow gets
// no signal. The row is written as the superuser because Enqueue only copies
// target_key from A's own workflow_ref; this models a producer bug or a forged
// row.
func TestAdversarialSignalCannotReachOtherTenantWorkflow(t *testing.T) {
	e := migrated(t)
	ctx := context.Background()
	fe, addr := startFrontend(t)
	victim := newProducer(t, e)
	victimStart := victim.start()
	mustExec(t, e.super, "UPDATE outbox SET dispatched_at = now() WHERE id = $1", victimStart)
	victimWF := victim.workflowID(victimStart)
	fe.addExecution(victimWF, false)

	attacker, row := uuid.New(), uuid.Must(uuid.NewV7())
	mustExec(t, e.super, "INSERT INTO tenants (id, state) VALUES ($1, 'active')", attacker)
	mustExec(t, e.super, `INSERT INTO outbox (id, tenant_id, kind, dedupe_key, target_key, target_seq)
		VALUES ($1, $2, 'approve', 'forged', $3, 1)`, row, attacker, "ticket.approval:"+victimStart.String())

	d, _, _ := dispatcherOn(t, e.relay, addr, Options{})
	if n, err := d.Poll(ctx); err != nil || n != 1 {
		t.Fatalf("poll claimed %d, %v", n, err)
	}
	if ex, _ := fe.execution(victimWF); len(ex.signals) != 0 {
		t.Fatalf("tenant B's workflow received %d signal(s) from tenant A's row", len(ex.signals))
	}
	fe.mu.Lock()
	signals := fe.signals
	fe.mu.Unlock()
	if len(signals) != 1 {
		t.Fatalf("%d signal calls, want 1", len(signals))
	}
	wantWF := "t:" + attacker.String() + ":ticket.approval:" + victimStart.String()
	if got := signals[0].GetWorkflowExecution().GetWorkflowId(); got != wantWF {
		t.Errorf("signal went to %q, want %q", got, wantWF)
	}
	if got, _ := headerString(t, signals[0].GetHeader(), workerguard.TenantHeader); got != attacker.String() {
		t.Errorf("nava-tenant header %q, want the row's tenant %s", got, attacker)
	}
	if n := count(t, e.super, "SELECT count(*) FROM outbox WHERE id = $1 AND dead_at IS NOT NULL AND last_error = $2",
		row, CauseTargetMissing); n != 1 {
		t.Error("tenant A's forged signal row is not dead with the target-missing cause")
	}
}

// A tenant A row of a workflow kind whose target_key is tenant B's started
// workflow is not a start (IsStart needs the row's own id in target_key), so
// it can neither get AlreadyStarted from B's execution and be marked
// dispatched against it, nor signal it: it is a signal to t:<A>:<key>, which
// does not exist, and is dead.
func TestAdversarialStartKindWithOtherTenantKeyIsNotAStart(t *testing.T) {
	e := migrated(t)
	ctx := context.Background()
	fe, addr := startFrontend(t)
	victim := newProducer(t, e)
	victimStart := victim.start()
	mustExec(t, e.super, "UPDATE outbox SET dispatched_at = now() WHERE id = $1", victimStart)
	victimWF := victim.workflowID(victimStart)
	fe.addExecution(victimWF, false)

	attacker, row := uuid.New(), uuid.Must(uuid.NewV7())
	mustExec(t, e.super, "INSERT INTO tenants (id, state) VALUES ($1, 'active')", attacker)
	mustExec(t, e.super, `INSERT INTO outbox (id, tenant_id, kind, dedupe_key, target_key, target_seq)
		VALUES ($1, $2, 'ticket.approval', 'forged', $3, 0)`, row, attacker, "ticket.approval:"+victimStart.String())

	d, _, _ := dispatcherOn(t, e.relay, addr, Options{})
	if n, err := d.Poll(ctx); err != nil || n != 1 {
		t.Fatalf("poll claimed %d, %v", n, err)
	}
	if req := fe.lastStart(); req != nil {
		t.Fatalf("a start was sent for %q", req.GetWorkflowId())
	}
	if ex, _ := fe.execution(victimWF); len(ex.signals) != 0 {
		t.Fatalf("tenant B's workflow received %d signal(s) from tenant A's row", len(ex.signals))
	}
	fe.mu.Lock()
	signals := fe.signals
	fe.mu.Unlock()
	wantWF := "t:" + attacker.String() + ":ticket.approval:" + victimStart.String()
	if len(signals) != 1 || signals[0].GetWorkflowExecution().GetWorkflowId() != wantWF {
		t.Fatalf("signals %v, want one to %q", signals, wantWF)
	}
	if n := count(t, e.super, "SELECT count(*) FROM outbox WHERE id = $1 AND dispatched_at IS NULL AND dead_at IS NOT NULL",
		row); n != 1 {
		t.Error("tenant A's forged row is not dead, or was marked dispatched")
	}
}

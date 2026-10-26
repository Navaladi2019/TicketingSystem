//go:build temporalstack

package temporalstack

// CI-6, relay and worker cases (ADR-001 Annex B CI-6 "Other cases", A.2;
// PLAN-001 task T32b), against the compose Temporal, its Authorizer, the
// stack's nava-relay service and a guarded worker. One named test per
// contract case, each named TestCI6_*; tools/ci/ci6report lists them.
//
// Threat model: a platform process holding the nava-relay certificate (or a
// relay bug) that sends a start or a signal with no nava-tenant header, or
// with another tenant's, to reach or disturb tenant A's workflow; and relay
// faults (redelivery, two relays, a missing target) that would apply a
// tenant's event twice, out of order, or forever.

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"

	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
	"github.com/Navaladi2019/TicketingSystem/internal/relay"
	"github.com/Navaladi2019/TicketingSystem/internal/workerguard"
)

// A start that reaches the worker with no nava-tenant header, or with another
// tenant's, fails the workflow non-retryably with an alert; the workflow code
// never runs. The Authorizer lets both starts through (it does not read the
// start header), so the worker is the control here.
func TestCI6_WorkerStartWithMissingOrMismatchedHeaderFailsWorkflow(t *testing.T) {
	ctx := ci6Context(t)
	s := newRelayStack(t)
	bare := dial(t, "nava-relay") // no Propagator: no header at all
	for _, tc := range []struct {
		name, outcome string
		start         func(id string, tenant uuid.UUID) (client.WorkflowRun, error)
	}{
		{"missing", "missing", func(id string, _ uuid.UUID) (client.WorkflowRun, error) {
			return bare.ExecuteWorkflow(ctx, client.StartWorkflowOptions{ID: id, TaskQueue: taskQueue}, relayType, relayRun{})
		}},
		{"mismatched", "mismatch", func(id string, _ uuid.UUID) (client.WorkflowRun, error) {
			return s.relay.ExecuteWorkflow(workerguard.ContextWithTenant(ctx, uuid.New()),
				client.StartWorkflowOptions{ID: id, TaskQueue: taskQueue}, relayType, relayRun{})
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tenant := uuid.New()
			id := relayWorkflowID(tenant, uuid.New())
			run, err := tc.start(id, tenant)
			if err != nil {
				t.Fatalf("start %s: %v (the Authorizer must let it reach the worker)", id, err)
			}
			s.terminateAtEnd(t, id)
			err = s.core.GetWorkflow(ctx, id, run.GetRunID()).Get(ctx, nil)
			var app *temporal.ApplicationError
			if !errors.As(err, &app) || app.Type() != workerguard.ErrTypeTenantGuard ||
				!strings.Contains(app.Error(), "workflow_start: "+tc.outcome) {
				t.Fatalf("result %v; want the worker's workflow_start %s guard error", err, tc.outcome)
			}
			desc, err := s.core.DescribeWorkflowExecution(ctx, id, run.GetRunID())
			if err != nil {
				t.Fatal(err)
			}
			if st := desc.GetWorkflowExecutionInfo().GetStatus(); st != enumspb.WORKFLOW_EXECUTION_STATUS_FAILED {
				t.Errorf("status %v, want FAILED", st)
			}
			if n := guardAlerts(s.logs, id, "workflow_start", tc.outcome); n != 1 {
				t.Errorf("%d workflow_start alerts for %s, want 1:\n%s", n, id, s.logs.String())
			}
		})
	}
}

// A signal from another tenant (nava-relay certificate, the victim's workflow
// id, another tenant's header) is dropped with an alert: it is not applied,
// even a "done" does not end the workflow, which keeps running and then
// applies its own tenant's signals from the outbox.
func TestCI6_WorkerWrongTenantSignalDroppedWithAlertWorkflowKeepsRunning(t *testing.T) {
	ctx := ci6Context(t)
	s := newRelayStack(t)
	tg := newTarget(t)
	psqlSuper(t, tg.startSQL(tg.refs()))
	id := tg.workflowID()
	s.terminateAtEnd(t, id)
	if r := awaitSettled(t, tg.start); !r.dispatched {
		t.Fatalf("start row %+v, want dispatched", r)
	}

	attacker := workerguard.ContextWithTenant(ctx, uuid.New())
	forged := uuid.New()
	if err := s.relay.SignalWorkflow(attacker, id, "", sigNote, outbox.Delivery{OutboxID: forged}); err != nil {
		t.Fatalf("wrong-tenant note: %v", err)
	}
	if err := s.relay.SignalWorkflow(attacker, id, "", sigDone, outbox.Delivery{OutboxID: uuid.New()}); err != nil {
		t.Fatalf("wrong-tenant done: %v", err)
	}
	deadline := time.Now().Add(30 * time.Second)
	for guardAlerts(s.logs, id, "signal", "mismatch") < 2 {
		if time.Now().After(deadline) {
			t.Fatalf("no two signal alerts for %s within 30 s:\n%s", id, s.logs.String())
		}
		time.Sleep(200 * time.Millisecond)
	}
	desc, err := s.core.DescribeWorkflowExecution(ctx, id, "")
	if err != nil {
		t.Fatal(err)
	}
	if st := desc.GetWorkflowExecutionInfo().GetStatus(); st != enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING {
		t.Fatalf("status %v after the wrong-tenant signals, want RUNNING", st)
	}

	own, noteSQL := tg.signalSQL(t, sigNote)
	_, doneSQL := tg.signalSQL(t, sigDone)
	psqlSuper(t, noteSQL+doneSQL)
	if out := s.result(ctx, t, id); out != own.String() {
		t.Errorf("applied %q, want only the own-tenant note %s (forged %s must be dropped)", out, own, forged)
	}
}

// A start redelivered after its workflow completed (the relay crashed between
// Temporal accepting it and MarkDispatched) starts no second run:
// REJECT_DUPLICATE answers AlreadyStarted and the row is dispatched.
func TestCI6_RelayRedeliveredStartAfterCompletionStartsNoSecondRun(t *testing.T) {
	ctx := ci6Context(t)
	s := newRelayStack(t)
	tg := newTarget(t)
	_, doneSQL := tg.signalSQL(t, sigDone)
	psqlSuper(t, tg.startSQL(tg.refs())+doneSQL)
	id := tg.workflowID()
	s.terminateAtEnd(t, id)
	if out := s.result(ctx, t, id); out != "" {
		t.Fatalf("first run result %q, want empty", out)
	}
	first, err := s.core.DescribeWorkflowExecution(ctx, id, "")
	if err != nil {
		t.Fatal(err)
	}
	if st := first.GetWorkflowExecutionInfo().GetStatus(); st != enumspb.WORKFLOW_EXECUTION_STATUS_COMPLETED {
		t.Fatalf("first run status %v, want COMPLETED", st)
	}
	// The final activity ran in the nava-worker service as nava_worker.
	if closed := psqlSuper(t, "SELECT closed_at IS NOT NULL FROM workflow_ref WHERE outbox_id = '"+tg.start.String()+"';"); closed != "t" {
		t.Errorf("workflow_ref.closed_at set: %q, want t", closed)
	}

	redeliver(t, tg.start)
	r := awaitSettled(t, tg.start)
	if !r.dispatched || r.attempts != 2 {
		t.Fatalf("redelivered start row %+v, want dispatched at attempt 2 (AlreadyStarted)", r)
	}
	again, err := s.core.DescribeWorkflowExecution(ctx, id, "")
	if err != nil {
		t.Fatal(err)
	}
	if a, b := again.GetWorkflowExecutionInfo().GetExecution().GetRunId(), first.GetWorkflowExecutionInfo().GetExecution().GetRunId(); a != b {
		t.Fatalf("latest run %s, want the completed run %s: the redelivery started a second run", a, b)
	}
	if st := again.GetWorkflowExecutionInfo().GetStatus(); st != enumspb.WORKFLOW_EXECUTION_STATUS_COMPLETED {
		t.Errorf("status %v after the redelivery, want COMPLETED", st)
	}
}

// A signal row redelivered by the relay is applied once: before
// continue-as-new, and again after it (the applied set is carried in the new
// run's input, I7).
func TestCI6_RelayRedeliveredSignalAppliedOnceAcrossContinueAsNew(t *testing.T) {
	ctx := ci6Context(t)
	s := newRelayStack(t)
	tg := newTarget(t)
	s1, s1SQL := tg.signalSQL(t, sigNote)
	psqlSuper(t, tg.startSQL(tg.refs())+s1SQL)
	id := tg.workflowID()
	s.terminateAtEnd(t, id)
	if r := awaitSettled(t, s1); !r.dispatched {
		t.Fatalf("note row %+v, want dispatched", r)
	}
	firstRun := s.awaitStarted(ctx, t, id)

	redeliver(t, s1)
	if r := awaitSettled(t, s1); !r.dispatched || r.attempts != 2 {
		t.Fatalf("redelivered note %+v, want dispatched at attempt 2", r)
	}

	rollover, rolloverSQL := tg.signalSQL(t, sigRollover)
	psqlSuper(t, rolloverSQL)
	if r := awaitSettled(t, rollover); !r.dispatched {
		t.Fatalf("rollover row %+v, want dispatched", r)
	}
	deadline := time.Now().Add(60 * time.Second)
	for s.awaitStarted(ctx, t, id) == firstRun {
		if time.Now().After(deadline) {
			t.Fatalf("no new run of %s after continue-as-new within 60 s", id)
		}
		time.Sleep(200 * time.Millisecond)
	}

	redeliver(t, s1) // after continue-as-new
	if r := awaitSettled(t, s1); !r.dispatched || r.attempts != 3 {
		t.Fatalf("note redelivered after continue-as-new %+v, want dispatched at attempt 3", r)
	}
	s2, s2SQL := tg.signalSQL(t, sigNote)
	_, doneSQL := tg.signalSQL(t, sigDone)
	psqlSuper(t, s2SQL+doneSQL)
	if out, want := s.result(ctx, t, id), s1.String()+","+s2.String(); out != want {
		t.Errorf("applied %q, want %q: each note once across three deliveries and continue-as-new", out, want)
	}
}

// A signal written with its start, in one transaction, waits behind it: while
// the start is not yet available, the signal is not even attempted (so it
// cannot hit NotFound and die); once the start is dispatched the signal is
// delivered at its first attempt and applied.
func TestCI6_RelaySignalQueuesBehindItsStart(t *testing.T) {
	ctx := ci6Context(t)
	s := newRelayStack(t)
	tg := newTarget(t)
	note, noteSQL := tg.signalSQL(t, sigNote)
	done, doneSQL := tg.signalSQL(t, sigDone)
	hold := "UPDATE outbox SET available_at = now() + interval '15 seconds' WHERE id = '" + tg.start.String() + "';\n"
	psqlSuper(t, tg.startSQL(tg.refs())+hold+noteSQL+doneSQL)
	id := tg.workflowID()
	s.terminateAtEnd(t, id)

	time.Sleep(3 * time.Second) // six relay polls while the start is held
	for _, rid := range []uuid.UUID{tg.start, note, done} {
		if r := outboxRow(t, rid); r.attempts != 0 || r.dispatched || r.dead {
			t.Fatalf("row %s %+v while its start is held, want untouched", rid, r)
		}
	}

	if out := s.result(ctx, t, id); out != note.String() {
		t.Errorf("applied %q, want %s", out, note)
	}
	for _, rid := range []uuid.UUID{tg.start, note, done} {
		if r := outboxRow(t, rid); !r.dispatched || r.attempts != 1 {
			t.Errorf("row %s %+v, want dispatched at its first attempt", rid, r)
		}
	}
	order := psqlSuper(t, "SELECT string_agg(id::text, ',' ORDER BY dispatched_at, target_seq) FROM outbox WHERE tenant_id = '"+tg.tenant.String()+"';")
	if want := tg.start.String() + "," + note.String() + "," + done.String(); order != want {
		t.Errorf("dispatch order %s, want %s", order, want)
	}
}

// Two more relay processes beside the stack's take over one another's
// targets partway through, and each target's signals are still applied in
// exactly their target_seq order. The hand-off is forced, not hoped for:
// extra relay 0 serves the first notes alone (the others paused), extra relay
// 1 the next ones alone, then all three run for done. Each phase waits until
// its rows are settled before the next relay takes over, so the paused relay
// holds no lease, and each extra relay must have dispatched some signals.
func TestCI6_RelayTwoRelaysNeverReorderOneTarget(t *testing.T) {
	ctx := ci6Context(t)
	s := newRelayStack(t)
	extras := []string{extraRelay(t), extraRelay(t)}
	before := make([]float64, len(extras))
	for i, c := range extras {
		before[i] = relayMetric(t, c, "outbox_relay_dispatches_total", `op="signal"`, `outcome="dispatched"`)
	}
	stack := containerOf(t, "nava-relay")
	all := append([]string{stack}, extras...)
	paused := map[string]bool{}
	pause := func(ids ...string) {
		t.Helper()
		for _, id := range ids {
			if !paused[id] {
				docker(t, "pause", id)
				paused[id] = true
			}
		}
	}
	unpause := func(ids ...string) {
		t.Helper()
		for _, id := range ids {
			if paused[id] {
				docker(t, "unpause", id)
				paused[id] = false
			}
		}
	}
	// A failed run must not leave a relay paused for the next test.
	t.Cleanup(func() {
		for _, id := range all {
			if paused[id] {
				_, _ = dockerOut("unpause", id)
			}
		}
	})

	const targets, firstNotes, secondNotes = 3, 8, 7
	tgs := make([]*target, targets)
	want := make([][]string, targets)
	// phase writes n notes on every target (and the start first, if start)
	// in one transaction, and waits until the last note of each is settled:
	// rows of one target dispatch in target_seq order, so all are.
	phase := func(name string, n int, start bool) {
		t.Helper()
		var sql strings.Builder
		last := make([]uuid.UUID, targets)
		for i, tg := range tgs {
			if start {
				sql.WriteString(tg.startSQL(tg.refs()))
			}
			for range n {
				id, q := tg.signalSQL(t, sigNote)
				want[i] = append(want[i], id.String())
				last[i] = id
				sql.WriteString(q)
			}
		}
		psqlSuper(t, sql.String())
		for i, id := range last {
			if r := awaitSettled(t, id); !r.dispatched {
				t.Fatalf("%s: target %d note %s %+v, want dispatched", name, i, id, r)
			}
		}
	}
	for i := range tgs {
		tgs[i] = newTarget(t)
		s.terminateAtEnd(t, tgs[i].workflowID())
	}

	pause(stack, extras[1])
	phase("extra relay 0 alone", firstNotes, true)
	pause(extras[0])
	unpause(extras[1])
	phase("extra relay 1 alone", secondNotes, false)
	unpause(all...)

	var done strings.Builder
	for _, tg := range tgs {
		_, q := tg.signalSQL(t, sigDone)
		done.WriteString(q)
	}
	psqlSuper(t, done.String())
	for i, tg := range tgs {
		if out := s.result(ctx, t, tg.workflowID()); out != strings.Join(want[i], ",") {
			t.Errorf("target %d applied\n%s\nwant target_seq order\n%s", i, out, strings.Join(want[i], ","))
		}
	}
	for i, c := range extras {
		if n := relayMetric(t, c, "outbox_relay_dispatches_total", `op="signal"`, `outcome="dispatched"`) - before[i]; n <= 0 {
			t.Errorf("extra relay %d dispatched %v signals, want some: it served a phase alone", i, n)
		}
	}
}

// NotFound on a signal is terminal: a signal whose target never started, or
// has completed, is marked dead at its first attempt with CauseTargetMissing,
// counted on outbox_relay_signal_target_missing_total and logged with the
// workflow id, and never retried.
func TestCI6_RelaySignalNotFoundIsTerminal(t *testing.T) {
	ctx := ci6Context(t)
	s := newRelayStack(t)
	svc := containerOf(t, "nava-relay")
	for _, tc := range []struct {
		name  string
		setup func(t *testing.T, tg *target) string // returns SQL that leaves no open target
	}{
		{"never started", func(_ *testing.T, tg *target) string {
			// The workflow_ref of a start that was never dispatched.
			return "INSERT INTO workflow_ref (tenant_id, outbox_id, record_id, kind) VALUES ('" +
				tg.tenant.String() + "', '" + tg.start.String() + "', '" + tg.record.String() + "', '" + relayType + "');\n"
		}},
		{"completed", func(t *testing.T, tg *target) string {
			_, doneSQL := tg.signalSQL(t, sigDone)
			psqlSuper(t, tg.startSQL(tg.refs())+doneSQL)
			s.terminateAtEnd(t, tg.workflowID())
			if out := s.result(ctx, t, tg.workflowID()); out != "" {
				t.Fatalf("result %q, want empty", out)
			}
			return ""
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tg := newTarget(t)
			mark := logMark()
			missingBefore := relayMetric(t, svc, "outbox_relay_signal_target_missing_total")
			setup := tc.setup(t, tg)
			note, noteSQL := tg.signalSQL(t, sigNote)
			psqlSuper(t, setup+noteSQL)

			r := awaitSettled(t, note)
			if !r.dead || r.dispatched || r.attempts != 1 || r.lastError != relay.CauseTargetMissing {
				t.Fatalf("signal row %+v, want dead at attempt 1 with %q", r, relay.CauseTargetMissing)
			}
			time.Sleep(2 * time.Second) // four more polls
			if again := outboxRow(t, note); again != r {
				t.Errorf("signal row changed after it died: %+v, then %+v", r, again)
			}
			if tc.name == "never started" {
				_, err := s.core.DescribeWorkflowExecution(ctx, tg.workflowID(), "")
				var nf *serviceerror.NotFound
				if !errors.As(err, &nf) {
					t.Errorf("describe %s: %v, want NotFound", tg.workflowID(), err)
				}
			}
			deadline := time.Now().Add(20 * time.Second)
			for !relayLogHas(t, mark, "outbox signal target workflow not found", tg.workflowID()) ||
				relayMetric(t, svc, "outbox_relay_signal_target_missing_total") <= missingBefore {
				if time.Now().After(deadline) {
					t.Fatalf("no target-missing alert (log line with %s, metric rise) within 20 s", tg.workflowID())
				}
				time.Sleep(500 * time.Millisecond)
			}
		})
	}
}

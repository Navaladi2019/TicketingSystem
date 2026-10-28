//go:build temporalstack

package temporalstack

// T31 (TICK-65) against the real compose Temporal and its Authorizer: a worker
// set up with workerguard (Propagator, tracing, Guard) and clients set up the
// same way. Each test checks a guard where the worker sees it, then that the
// Authorizer accepts what the guarded worker sends.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"

	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
	"github.com/Navaladi2019/TicketingSystem/internal/workerguard"
)

// guardedInput selects what guardedWorkflow does. A child gets no arguments
// and a continued run gets the workerguard.RunInput its first run carried
// (the CI-7 argument rule accepts nothing else), so neither has a Mode.
type guardedInput struct {
	Mode string `json:"mode,omitempty"`
	workerguard.RunInput
}

// guardedRefs is what guardedSignals carries in RunInput.Refs, and what
// guardedWorkflow returns: outbox ids only (a result is recorded too, so the
// guard accepts nothing else, TICK-245). Empty, it encodes as {}.
type guardedRefs struct {
	Applied []string `json:"applied,omitempty"`
}

// guardedWorkflow is registered as the allow-listed nava.smoke.Echo.
//   - "leaf", or no mode in a first run (a child), completes;
//   - "child-helper" starts a "leaf" child named with ChildWorkflowID;
//   - "child-default" starts one with the SDK's default id;
//   - "signals", or no mode in a continued run, applies "outbox" signals once
//     each by outbox id, continues as new on "rollover", and returns what it
//     applied on "done"; update "inc" is accepted too.
func guardedWorkflow(ctx workflow.Context, in guardedInput) (guardedRefs, error) {
	mode := in.Mode
	if mode == "" {
		mode = "leaf"
		if workflow.GetInfo(ctx).ContinuedExecutionRunID != "" {
			mode = "signals"
		}
	}
	switch mode {
	case "leaf":
		return guardedRefs{}, nil
	case "child-helper", "child-default":
		opts := workflow.ChildWorkflowOptions{TaskQueue: taskQueue}
		if mode == "child-helper" {
			id, err := workerguard.ChildWorkflowID(ctx, "leaf")
			if err != nil {
				return guardedRefs{}, err
			}
			opts.WorkflowID = id
		}
		return guardedRefs{}, workflow.ExecuteChildWorkflow(workflow.WithChildOptions(ctx, opts), echoType).Get(ctx, nil)
	case "signals":
		return guardedSignals(ctx, in.RunInput)
	}
	return guardedRefs{}, errors.New("unknown mode")
}

func guardedSignals(ctx workflow.Context, in workerguard.RunInput) (guardedRefs, error) {
	var refs guardedRefs
	if in.Refs != nil {
		if err := json.Unmarshal(in.Refs, &refs); err != nil {
			return guardedRefs{}, err
		}
	}
	if err := workflow.SetUpdateHandler(ctx, "inc", func(workflow.Context) error { return nil }); err != nil {
		return guardedRefs{}, err
	}
	signals := workflow.GetSignalChannel(ctx, "outbox")
	apply := func(c workflow.ReceiveChannel) {
		var id uuid.UUID
		c.Receive(ctx, &id)
		if in.Seen.TryApply(ctx, id) {
			refs.Applied = append(refs.Applied, id.String())
		}
	}
	for {
		var next string
		sel := workflow.NewSelector(ctx)
		sel.AddReceive(signals, func(c workflow.ReceiveChannel, _ bool) { apply(c) })
		sel.AddReceive(workflow.GetSignalChannel(ctx, "rollover"), func(c workflow.ReceiveChannel, _ bool) { c.Receive(ctx, nil); next = "rollover" })
		sel.AddReceive(workflow.GetSignalChannel(ctx, "done"), func(c workflow.ReceiveChannel, _ bool) { c.Receive(ctx, nil); next = "done" })
		sel.Select(ctx)
		switch next {
		case "rollover":
			// Drain what arrived with the rollover, so no signal is lost.
			for signals.Len() > 0 {
				apply(signals)
			}
			b, err := json.Marshal(refs)
			if err != nil {
				return guardedRefs{}, err
			}
			in.Refs = b
			return guardedRefs{}, workflow.NewContinueAsNewError(ctx, echoType, in)
		case "done":
			return refs, nil
		}
	}
}

type lockedBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (l *lockedBuffer) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.Write(p)
}

func (l *lockedBuffer) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.String()
}

// guardedStack starts a guarded worker and returns relay and core clients set
// up as the platform sets them up, plus the guard's log.
func guardedStack(t *testing.T) (relay, core client.Client, logs *lockedBuffer) {
	t.Helper()
	tel, err := telemetry.New(t.Context(), config.Telemetry{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tel.Shutdown(context.Background()) })
	tt, err := tel.NewTemporal(slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	setup := func(o *client.Options) { workerguard.ConfigureClient(o, tt) }
	logs = &lockedBuffer{}
	g, err := workerguard.New(slog.New(slog.NewJSONHandler(logs, nil)), tel.MeterProvider)
	if err != nil {
		t.Fatal(err)
	}
	w := worker.New(dial(t, "nava-worker", setup), taskQueue, workerguard.WorkerOptions(worker.Options{}, g))
	w.RegisterWorkflowWithOptions(guardedWorkflow, workflow.RegisterOptions{Name: echoType})
	if err := w.Start(); err != nil {
		t.Fatalf("worker start: %v", err)
	}
	t.Cleanup(w.Stop)
	return dial(t, "nava-relay", setup), dial(t, "nava-core", setup), logs
}

// guardedStart starts guardedWorkflow in tenant as the relay does: the id
// t:<tenant>:<kind>:<outbox id>, the tenant on the context for the header.
func guardedStart(ctx context.Context, t *testing.T, relay client.Client, tenant uuid.UUID, mode string) (string, client.WorkflowRun) {
	t.Helper()
	start := uuid.New()
	id := "t:" + tenant.String() + ":t31:" + start.String()
	// The relay's Delivery for the start row: its payload is carried across
	// continue-as-new, so it must be references only.
	in := guardedInput{Mode: mode, RunInput: workerguard.RunInput{
		Delivery: outbox.Delivery{OutboxID: start, KindVersion: 1, Payload: json.RawMessage(`{}`)},
	}}
	run, err := relay.ExecuteWorkflow(workerguard.ContextWithTenant(ctx, tenant),
		client.StartWorkflowOptions{ID: id, TaskQueue: taskQueue}, echoType, in)
	if err != nil {
		t.Fatalf("start %s: %v", id, err)
	}
	system := dial(t, "temporal-system")
	t.Cleanup(func() { _ = system.TerminateWorkflow(context.Background(), id, "", "T31 cleanup") })
	return id, run
}

// A same-tenant child named with the helper starts and completes through the
// Authorizer; a child with the SDK default id is rejected in the worker, so
// the parent fails at once instead of retrying a denied workflow task.
func TestT31_ChildIDBuilderThroughAuthorizer(t *testing.T) {
	ctx := ci6Context(t)
	relay, core, _ := guardedStack(t)
	tenant := uuid.New()

	_, run := guardedStart(ctx, t, relay, tenant, "child-helper")
	// The parent completes only after its child completed.
	if err := core.GetWorkflow(ctx, run.GetID(), run.GetRunID()).Get(ctx, nil); err != nil {
		t.Fatalf("helper child: %v; want parent and child completed", err)
	}

	id, run := guardedStart(ctx, t, relay, tenant, "child-default")
	err := core.GetWorkflow(ctx, run.GetID(), run.GetRunID()).Get(ctx, nil)
	var app *temporal.ApplicationError
	if !errors.As(err, &app) || app.Type() != workerguard.ErrTypeTenantGuard || !strings.Contains(app.Error(), "child: cross_tenant") {
		t.Fatalf("default-id child: %v; want the worker's child cross_tenant guard error", err)
	}
	desc, err := core.DescribeWorkflowExecution(ctx, id, run.GetRunID())
	if err != nil {
		t.Fatal(err)
	}
	if st := desc.GetWorkflowExecutionInfo().GetStatus(); st != enumspb.WORKFLOW_EXECUTION_STATUS_FAILED {
		t.Errorf("parent status %v, want FAILED", st)
	}
	if n := len(desc.GetPendingChildren()); n != 0 {
		t.Errorf("%d pending children, want none", n)
	}
}

// A wrong-tenant signal is dropped and alerts while the workflow keeps
// running; a wrong-tenant update is rejected by the validator; a signal
// redelivered after a real continue-as-new (whose header the Authorizer
// checks) is applied once.
func TestT31_SignalsUpdatesAndContinueAsNew(t *testing.T) {
	ctx := ci6Context(t)
	relay, core, logs := guardedStack(t)
	tenant, other := uuid.New(), uuid.New()
	id, first := guardedStart(ctx, t, relay, tenant, "signals")
	own, wrong := workerguard.ContextWithTenant(ctx, tenant), workerguard.ContextWithTenant(ctx, other)
	signal := func(ctx context.Context, name string, arg any) {
		t.Helper()
		if err := relay.SignalWorkflow(ctx, id, "", name, arg); err != nil {
			t.Fatalf("signal %s: %v", name, err)
		}
	}
	s1, s2, s3 := uuid.New(), uuid.New(), uuid.New()

	signal(wrong, "outbox", uuid.New()) // dropped
	signal(wrong, "done", nil)          // dropped: must not end the workflow
	signal(own, "outbox", s1)
	signal(own, "outbox", s1) // a relay redelivery
	signal(own, "outbox", s2)

	// SDK v1.49: UpdateWorkflow returns (handle, nil) for a rejected update
	// and keeps the rejection on the handle (internal_workflow_client.go,
	// updateHandleFromResponse), so the rejection is read from Get.
	rejected, err := core.UpdateWorkflow(wrong, client.UpdateWorkflowOptions{
		WorkflowID: id, UpdateName: "inc", WaitForStage: client.WorkflowUpdateStageCompleted,
	})
	if err != nil {
		t.Fatalf("wrong-tenant update request: %v", err)
	}
	err = rejected.Get(ctx, nil)
	var app *temporal.ApplicationError
	if !errors.As(err, &app) || app.Type() != workerguard.ErrTypeTenantGuard ||
		!strings.Contains(app.Error(), "tenant guard denied update: mismatch") {
		t.Fatalf("wrong-tenant update: %v; want the validator's rejection", err)
	}
	h, err := core.UpdateWorkflow(own, client.UpdateWorkflowOptions{
		WorkflowID: id, UpdateName: "inc", WaitForStage: client.WorkflowUpdateStageCompleted,
	})
	if err != nil {
		t.Fatalf("own-tenant update: %v", err)
	}
	if err := h.Get(ctx, nil); err != nil {
		t.Fatalf("own-tenant update: %v", err)
	}

	signal(own, "rollover", nil)
	// Wait for the new run, so the redeliveries land after continue-as-new.
	deadline := time.Now().Add(time.Minute)
	for {
		desc, err := core.DescribeWorkflowExecution(ctx, id, "")
		if err == nil && desc.GetWorkflowExecutionInfo().GetExecution().GetRunId() != first.GetRunID() {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("no new run after continue-as-new: %v", err)
		}
		time.Sleep(200 * time.Millisecond)
	}
	signal(own, "outbox", s1) // redelivered after continue-as-new
	signal(own, "outbox", s3)
	signal(own, "outbox", s2) // redelivered after continue-as-new
	signal(own, "done", nil)

	var out guardedRefs
	if err := core.GetWorkflow(ctx, id, "").Get(ctx, &out); err != nil {
		t.Fatalf("result: %v", err)
	}
	if got, want := strings.Join(out.Applied, ","), s1.String()+","+s2.String()+","+s3.String(); got != want {
		t.Errorf("result %q, want %q: each signal once across continue-as-new", got, want)
	}
	var mismatches int
	for line := range strings.Lines(logs.String()) {
		if strings.Contains(line, `"alert":true`) && strings.Contains(line, `"guard":"signal"`) && strings.Contains(line, id) {
			mismatches++
		}
	}
	if mismatches != 2 {
		t.Errorf("%d signal alerts for %s, want 2:\n%s", mismatches, id, logs.String())
	}
	var updateAlerts int
	for line := range strings.Lines(logs.String()) {
		if strings.Contains(line, `"alert":true`) && strings.Contains(line, `"guard":"update"`) &&
			strings.Contains(line, `"outcome":"mismatch"`) && strings.Contains(line, id) {
			updateAlerts++
		}
	}
	if updateAlerts != 1 {
		t.Errorf("%d update alerts for %s, want 1", updateAlerts, id)
	}
}

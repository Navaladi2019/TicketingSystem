package backfill

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"

	"github.com/Navaladi2019/TicketingSystem/internal/fieldindex"
	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
	"github.com/Navaladi2019/TicketingSystem/internal/workerguard"
)

// Defaults of the two workflows.
const (
	// DefaultMaxItems is how many items one coordinator run handles before it
	// continues as new with what is still queued (D-7-12, "pin N").
	DefaultMaxItems = 100
	// DefaultIdleTimeout is how long a coordinator with nothing queued waits
	// for a signal before its run completes. The relay's SignalWithStart
	// starts a new run for the next item.
	DefaultIdleTimeout = 5 * time.Minute
	// DefaultMaxBatchesPerRun is how many batches one backfill run copies
	// before it continues as new with its cursor, so a large tenant's history
	// stays bounded (about seven events per batch).
	DefaultMaxBatchesPerRun = 1000
)

// The adaptive pause between batches (D10.4, CTO ruling 5, PLAN-007 D-7-15).
const (
	// InitialPause is the first pause, and the floor: the spike measured
	// +31% live-write p95 at 200 ms.
	InitialPause = 500 * time.Millisecond
	// MaxPause caps the pause.
	MaxPause = 10 * time.Second
)

// Activity bounds. Every activity retries without an attempt cap (backoff up
// to a minute; the barrier's up to 10 s): the coordinator runs one item at a
// time, so a failed child would leave its field backfilling with nothing to
// report it. Only ErrTypeNoChild is non-retryable.
const (
	shortStartToClose   = 30 * time.Second
	batchStartToClose   = 2 * time.Minute
	barrierStartToClose = 24 * time.Hour
	barrierHeartbeat    = 30 * time.Second
)

// RegisterWorkflow adds the coordinator and FieldIndexBackfill to r under
// their workflow types, with the defaults. It is the form the handbook build
// parses (docs/handbook/build/build.py GO_REGISTER_FUNCS).
func RegisterWorkflow(r worker.WorkflowRegistry) {
	r.RegisterWorkflowWithOptions(coordinatorRun, workflow.RegisterOptions{Name: fieldindex.CoordinatorWorkflowType})
	r.RegisterWorkflowWithOptions(backfillRun, workflow.RegisterOptions{Name: fieldindex.BackfillWorkflowType})
}

// coordinatorRun and backfillRun are the workflows with the defaults.
func coordinatorRun(ctx workflow.Context, in workerguard.RunInput) error {
	return (&Coordinator{}).Run(ctx, in)
}

func backfillRun(ctx workflow.Context, in Input) error { return (&Backfill{}).Run(ctx, in) }

func retryForever(maxInterval time.Duration) *temporal.RetryPolicy {
	return &temporal.RetryPolicy{
		InitialInterval:        time.Second,
		BackoffCoefficient:     2,
		MaximumInterval:        maxInterval,
		MaximumAttempts:        0,
		NonRetryableErrorTypes: []string{ErrTypeNoChild},
	}
}

func shortActivity(ctx workflow.Context) workflow.Context {
	return workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: shortStartToClose, RetryPolicy: retryForever(time.Minute)})
}

// barrier runs fieldindex.Barrier, which polls inside its attempt and
// heartbeats after each poll. A lost worker times the heartbeat out and the
// retry resumes the same snapshot. No overall deadline: a long transaction
// at the flip delays ready, and the slow alert reports it (R11).
func barrier(ctx workflow.Context) error {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: barrierStartToClose,
		HeartbeatTimeout:    barrierHeartbeat,
		RetryPolicy:         retryForever(10 * time.Second),
	})
	return workflow.ExecuteActivity(ctx, BarrierActivity).Get(ctx, nil)
}

// Input is FieldIndexBackfill's input: the field, references only,
// and on a continued run the cursor to resume after. The field must be the
// one in the workflow id, which is what the activities act on.
type Input struct {
	FieldID uuid.UUID  `json:"field_id"`
	Cursor  *uuid.UUID `json:"cursor,omitempty"`
}

// Backfill is FieldIndexBackfill (D9, D10). Zero fields take the defaults.
type Backfill struct {
	MaxBatchesPerRun int
}

// Register adds b.Run to r as fieldindex.BackfillWorkflowType.
func (b *Backfill) Register(r worker.WorkflowRegistry) {
	r.RegisterWorkflowWithOptions(b.Run, workflow.RegisterOptions{Name: fieldindex.BackfillWorkflowType})
}

// Run brings one field to ready:
//
//  1. the barrier (D9), skipped by a continued run, whose first run ran it;
//  2. batches until the cursor runs out or the registry row is no longer
//     this child's (removing, ready, gone, or another requested_version),
//     with the adaptive pause between them;
//  3. the flip to ready, which changes nothing in the second case;
//  4. the barrier again.
//
// The child's workflow id, ChildID, names the field and the pinned version.
func (b *Backfill) Run(ctx workflow.Context, in Input) error {
	c, err := parseChildID(workflow.GetInfo(ctx).WorkflowExecution.ID, fieldindex.BackfillWorkflowType)
	if err != nil || c.field != in.FieldID {
		return temporal.NewNonRetryableApplicationError("field index: input does not match the workflow id", ErrTypeNoChild, nil)
	}
	if in.Cursor == nil {
		if err := barrier(ctx); err != nil {
			return err
		}
	}
	bctx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: batchStartToClose, RetryPolicy: retryForever(time.Minute)})
	maxBatches := b.MaxBatchesPerRun
	if maxBatches <= 0 {
		maxBatches = DefaultMaxBatchesPerRun
	}
	cursor := in.Cursor
	var p pacer
	for n := 0; ; n++ {
		if n == maxBatches && cursor != nil {
			return workflow.NewContinueAsNewError(ctx, fieldindex.BackfillWorkflowType, Input{FieldID: in.FieldID, Cursor: cursor})
		}
		start := workflow.Now(ctx)
		var out Cursor
		if err := workflow.ExecuteActivity(bctx, BatchActivity, Cursor{Cursor: cursor}).Get(bctx, &out); err != nil {
			return err
		}
		if out.Cursor == nil {
			break
		}
		cursor = out.Cursor
		if err := workflow.Sleep(ctx, p.next(workflow.Now(ctx).Sub(start))); err != nil {
			return err
		}
	}
	sctx := shortActivity(ctx)
	if err := workflow.ExecuteActivity(sctx, ReadyActivity).Get(sctx, nil); err != nil {
		return err
	}
	return barrier(ctx)
}

// pacer is the adaptive pause (D10.4, D-7-15). Its only input is each
// batch's duration as the workflow measured it, workflow.Now() around the
// activity, so a replay computes the same pauses. The fastest batch seen is
// the uncontended baseline; a batch more than 25% slower than it means the
// batches compete with live writes, and the pause doubles; a batch within
// 10% of it halves the pause, never below InitialPause.
type pacer struct {
	pause, baseline time.Duration
}

func (p *pacer) next(d time.Duration) time.Duration {
	if p.pause == 0 {
		p.pause = InitialPause
	}
	if d <= 0 {
		return p.pause
	}
	if p.baseline == 0 || d < p.baseline {
		p.baseline = d
	}
	switch {
	case d*4 > p.baseline*5:
		p.pause = min(p.pause*2, MaxPause)
	case d*10 <= p.baseline*11:
		p.pause = max(p.pause/2, InitialPause)
	}
	return p.pause
}

// Coordinator is the per-tenant queue (D10.5, PLAN-007 D-7-12): one child at
// a time, in arrival order. Zero fields take the defaults.
type Coordinator struct {
	MaxItems    int
	IdleTimeout time.Duration
}

// Register adds c.Run to r as fieldindex.CoordinatorWorkflowType.
func (c *Coordinator) Register(r worker.WorkflowRegistry) {
	r.RegisterWorkflowWithOptions(c.Run, workflow.RegisterOptions{Name: fieldindex.CoordinatorWorkflowType})
}

// queued is one coordinator item: a field and the requested_version to pin
// its child to, 0 for an item carried across continue-as-new.
type queued struct {
	field   uuid.UUID
	version int
}

// pendingRefs is RunInput.Refs across continue-as-new: the queued field ids
// in arrival order, references only (CI-7). Their op and version are not
// carried: Classify re-reads the op from the registry, and the child runs
// unpinned (TICK-745 guard adaptation).
type pendingRefs struct {
	Pending []uuid.UUID `json:"pending"`
}

// Run is the coordinator, t:<tenant>:field_index.coordinator. The relay
// signal-with-starts it: its first input is the outbox row's Delivery, which
// it does not read; a continued run's input carries the applied-signal set
// (Seen) and the pending field ids (Refs).
//
// Each SignalEnqueue item is deduplicated twice: on its outbox id through
// Seen (I7, kept across continue-as-new), and on (field, requested_version)
// within the run. Then, one item at a time, Classify reads the field's
// registry row and the matching child runs: FieldIndexBackfill for
// backfilling, FieldIndexRemove (TICK-750) for removing, nothing for ready
// or gone. A child that fails is logged and the queue moves on.
//
// After MaxItems items with more still queued the run continues as new
// carrying them; with nothing queued for IdleTimeout it completes.
func (c *Coordinator) Run(ctx workflow.Context, in workerguard.RunInput) error {
	tenant, err := workerguard.TenantOf(workflow.GetInfo(ctx).WorkflowExecution.ID)
	if err != nil {
		return temporal.NewNonRetryableApplicationError("field index: coordinator id has no tenant", ErrTypeNoChild, nil)
	}
	// The tenant is the workflow id's, which the relay built from the outbox
	// row. A start payload naming another tenant is refused, fail closed,
	// before anything is read or written (TICK-748 review condition; the
	// relay refuses such a row first).
	if !payloadTenantMatches(in.Payload, tenant) {
		return temporal.NewNonRetryableApplicationError("field index: start payload names another tenant", ErrTypeNoChild, nil)
	}
	maxItems, idle := c.MaxItems, c.IdleTimeout
	if maxItems <= 0 {
		maxItems = DefaultMaxItems
	}
	if idle <= 0 {
		idle = DefaultIdleTimeout
	}
	log := workflow.GetLogger(ctx)
	seen := in.Seen
	var queue []queued
	if len(in.Refs) > 0 {
		var refs pendingRefs
		if err := json.Unmarshal(in.Refs, &refs); err != nil {
			return temporal.NewNonRetryableApplicationError("field index: coordinator refs are not {pending}", ErrTypeNoChild, nil)
		}
		for _, f := range refs.Pending {
			queue = append(queue, queued{field: f})
		}
	}
	keys := map[queued]bool{}
	accept := func(it fieldindex.EnqueueItem) {
		q, outboxID, ok := parseItem(it)
		if !ok {
			log.Warn("field index coordinator dropped a malformed item")
			return
		}
		if !seen.TryApply(ctx, outboxID) || keys[q] {
			return
		}
		keys[q] = true
		queue = append(queue, q)
	}
	ch := workflow.GetSignalChannel(ctx, fieldindex.SignalEnqueue)
	drain := func() {
		for {
			var it fieldindex.EnqueueItem
			if !ch.ReceiveAsync(&it) {
				return
			}
			accept(it)
		}
	}

	handled := 0
	for {
		drain()
		if len(queue) == 0 {
			if !waitForItem(ctx, ch, idle, accept) {
				drain()
				if len(queue) == 0 {
					return nil
				}
			}
			continue
		}
		if handled >= maxItems {
			return continueWith(ctx, seen, queue)
		}
		item := queue[0]
		queue = queue[1:]
		if err := runItem(ctx, tenant, item); err != nil {
			if errors.Is(ctx.Err(), workflow.ErrCanceled) {
				return err
			}
			log.Warn("field index child failed; the field stays as it is", "field_id", item.field.String())
		}
		handled++
	}
}

// waitForItem blocks until a signal arrives (accepted through accept) or
// idle passes, and reports whether a signal arrived.
func waitForItem(ctx workflow.Context, ch workflow.ReceiveChannel, idle time.Duration, accept func(fieldindex.EnqueueItem)) bool {
	tctx, cancel := workflow.WithCancel(ctx)
	defer cancel()
	got := false
	sel := workflow.NewSelector(ctx)
	sel.AddReceive(ch, func(rc workflow.ReceiveChannel, _ bool) {
		var it fieldindex.EnqueueItem
		rc.Receive(ctx, &it)
		accept(it)
		got = true
	})
	sel.AddFuture(workflow.NewTimer(tctx, idle), func(workflow.Future) {})
	sel.Select(ctx)
	return got
}

// continueWith continues the coordinator as new with queue's fields, in
// order, and the applied-signal set.
func continueWith(ctx workflow.Context, seen workerguard.AppliedSignals, queue []queued) error {
	refs := pendingRefs{Pending: make([]uuid.UUID, len(queue))}
	for i, q := range queue {
		refs.Pending[i] = q.field
	}
	raw, err := json.Marshal(refs)
	if err != nil {
		return err
	}
	next := workerguard.RunInput{Delivery: outbox.Delivery{Payload: json.RawMessage(`{}`)}, Seen: seen, Refs: raw}
	return workflow.NewContinueAsNewError(ctx, fieldindex.CoordinatorWorkflowType, next)
}

// parseItem validates a signal body: a known op, canonical ids, a version in
// range. The op is checked but not used: Classify decides from the registry.
func parseItem(it fieldindex.EnqueueItem) (queued, uuid.UUID, bool) {
	if it.Op != fieldindex.OpBackfill && it.Op != fieldindex.OpRemove {
		return queued{}, uuid.Nil, false
	}
	field, err := uuid.Parse(it.FieldID)
	if err != nil || field == uuid.Nil || field.String() != it.FieldID {
		return queued{}, uuid.Nil, false
	}
	outboxID, err := uuid.Parse(it.OutboxID)
	if err != nil || outboxID == uuid.Nil || outboxID.String() != it.OutboxID {
		return queued{}, uuid.Nil, false
	}
	if it.RequestedVersion < 1 || it.RequestedVersion > maxVersion {
		return queued{}, uuid.Nil, false
	}
	return queued{field: field, version: it.RequestedVersion}, outboxID, true
}

// runItem classifies q and runs its child, if any, to completion.
func runItem(ctx workflow.Context, tenant uuid.UUID, q queued) error {
	sctx := shortActivity(ctx)
	var cls Classified
	if err := workflow.ExecuteActivity(sctx, ClassifyActivity, FieldRef{FieldID: q.field}).Get(sctx, &cls); err != nil {
		return err
	}
	switch {
	case cls.Backfill != nil:
		cctx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
			WorkflowID: ChildID(tenant, fieldindex.BackfillWorkflowType, q.field, q.version)})
		return workflow.ExecuteChildWorkflow(cctx, fieldindex.BackfillWorkflowType, Input{FieldID: q.field}).Get(cctx, nil)
	case cls.Remove != nil:
		cctx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
			WorkflowID: ChildID(tenant, fieldindex.RemoveWorkflowType, q.field, q.version)})
		return workflow.ExecuteChildWorkflow(cctx, fieldindex.RemoveWorkflowType, FieldRef{FieldID: q.field}).Get(cctx, nil)
	}
	return nil
}

// payloadTenantMatches reports whether a start payload's tenant_id, when it
// has one, is tenant. A continued run's payload is {} and matches.
func payloadTenantMatches(payload json.RawMessage, tenant uuid.UUID) bool {
	if len(payload) == 0 {
		return true
	}
	var p struct {
		TenantID *string `json:"tenant_id"`
	}
	if json.Unmarshal(payload, &p) != nil {
		return false
	}
	return p.TenantID == nil || *p.TenantID == tenant.String()
}

package relay

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"

	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
	"github.com/Navaladi2019/TicketingSystem/internal/workerguard"
)

// TaskQueue is the queue every relay start uses: the only one on the
// Authorizer's allow-list (docs/temporal-server.md, D9).
const TaskQueue = "nava-tq"

// The last_error texts the dispatcher writes. Every cause is one of these, or
// failureCause's fixed text with an op and a gRPC code name: nothing from an
// error message, a row or a payload reaches last_error (TICK-74 security
// review).
const (
	// CauseTargetMissing is a signal whose workflow does not exist or has
	// already completed (A.2: terminal at once).
	CauseTargetMissing = "relay: signal target workflow not found or already completed"
	// CauseInvalidTarget is a row whose target_key or kind is not in the only
	// form the relay dispatches (TICK-94 item 3).
	CauseInvalidTarget = "relay: target_key is not <kind>:<uuid>"
	// CauseInvalidPayload is a row whose payload is not references only
	// (outbox.ValidatePayload, CI-7): record data written outside Enqueue
	// (T32b finding 1, second line). It never reaches Temporal.
	CauseInvalidPayload = "relay: payload is not references only"
	// CauseLeaseTooShort is a row the poll claimed but reached only after its
	// dispatch deadline.
	CauseLeaseTooShort = "relay: dispatch deadline passed before the temporal call"
)

// Temporal is the part of the Temporal client the dispatcher calls. A
// client.Client built with workerguard.ConfigureClient satisfies it.
type Temporal interface {
	ExecuteWorkflow(ctx context.Context, options client.StartWorkflowOptions, workflow any, args ...any) (client.WorkflowRun, error)
	SignalWorkflow(ctx context.Context, workflowID, runID, signalName string, arg any) error
}

// The values of the op and outcome attributes of outbox_relay_dispatches.
const (
	opStart  = "start"
	opSignal = "signal"

	outcomeDispatched     = "dispatched"
	outcomeAlreadyStarted = "already_started"
	outcomeTargetMissing  = "target_missing"
	outcomeInvalidTarget  = "invalid_target"
	outcomeInvalidPayload = "invalid_payload"
	outcomeBackoff        = "backoff"
	outcomeDead           = "dead" // Backoff found the row MaxRedeliveryAge old
	outcomeLeaseLost      = "lease_lost"
	outcomeSettleFailed   = "settle_failed"
)

// The attributes of outbox_relay_dispatches; both take only the values above.
const (
	attrOp      = attribute.Key("op")
	attrOutcome = attribute.Key("outcome")
)

// settleTimeout bounds the Mark* or Backoff call after a Temporal call. It
// runs even when the poll's context is cancelled, so a row Temporal accepted
// during shutdown is still marked.
const settleTimeout = 5 * time.Second

// DispatcherOptions tune a Dispatcher. Zero fields take the defaults.
type DispatcherOptions struct {
	// Concurrency is how many rows of one poll are dispatched at once. The
	// rows of one poll are on distinct targets (Claim takes only heads), so
	// they may go in any order. Default 8.
	Concurrency int
}

// Dispatcher sends claimed outbox rows to Temporal (ADR-001 A.2 "What the
// relay does"; PLAN-001 T30b) and settles each one: MarkDispatched,
// MarkDead or Backoff. No dispatch error leaves a row claimed and unsettled;
// only a failed settle does, and the row is then redelivered after its lease.
type Dispatcher struct {
	relay    *Relay
	temporal Temporal
	log      *slog.Logger
	tracer   trace.Tracer
	opts     DispatcherOptions

	dispatches metric.Int64Counter
	missing    metric.Int64Counter
	refused    metric.Int64Counter

	// beforeStart runs after the claim and before each StartWorkflow call. It
	// is the crash seam (T33): a no-op unless the binary is built with the
	// relaytest tag.
	beforeStart func(context.Context, Claimed)
	// now is the dispatcher's clock, for the dispatch deadline.
	now func() time.Time
}

// NewDispatcher builds a Dispatcher on r. tc must be built with
// workerguard.ConfigureClient on telemetry.NewTemporal, and tp and mp must
// come from internal/platform/telemetry in production.
func NewDispatcher(r *Relay, tc Temporal, tp trace.TracerProvider, mp metric.MeterProvider, log *slog.Logger, opts DispatcherOptions) (*Dispatcher, error) {
	if r == nil || tc == nil || tp == nil || mp == nil || log == nil {
		return nil, errors.New("relay: nil relay, temporal client, provider or logger")
	}
	if opts.Concurrency <= 0 {
		opts.Concurrency = 8
	}
	meter := mp.Meter("github.com/Navaladi2019/TicketingSystem/internal/relay")
	dispatches, err := meter.Int64Counter("outbox_relay_dispatches",
		metric.WithDescription("Outbox rows the relay dispatched or settled, by op (start, signal) and outcome. Never labelled by tenant or workflow."))
	if err != nil {
		return nil, err
	}
	missing, err := meter.Int64Counter("outbox_relay_signal_target_missing",
		metric.WithDescription("Signal rows marked dead because their workflow was not found or had completed. Alert on any increase; the workflow id is in the log line, not here."))
	if err != nil {
		return nil, err
	}
	refused, err := meter.Int64Counter("outbox_relay_payload_refused",
		metric.WithDescription("Outbox rows marked dead because their payload is not references only (CI-7). Alert on any increase; the outbox id is in the log line, not here."))
	if err != nil {
		return nil, err
	}
	return &Dispatcher{
		relay: r, temporal: tc, log: log, opts: opts,
		tracer:     tp.Tracer("github.com/Navaladi2019/TicketingSystem/internal/relay"),
		dispatches: dispatches, missing: missing, refused: refused,
		beforeStart: crashSeam, now: time.Now,
	}, nil
}

// Run polls until ctx is done, every interval when a poll claimed nothing and
// at once when it claimed something. A poll error is logged and the next poll
// waits interval.
func (d *Dispatcher) Run(ctx context.Context, interval time.Duration) {
	t := time.NewTimer(0)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		if ctx.Err() != nil {
			return
		}
		n, err := d.Poll(ctx)
		switch {
		case err != nil && ctx.Err() == nil:
			d.log.ErrorContext(ctx, "outbox relay poll failed", slog.Any("error", err))
			t.Reset(interval)
		case n == 0:
			t.Reset(interval)
		default:
			t.Reset(0)
		}
	}
}

// Poll claims once and dispatches and settles every claimed row. It returns
// how many rows it claimed. Only a Claim failure is an error; each row's
// outcome is counted and logged.
//
// Every Temporal call must end before the dispatch deadline, half a lease
// after the claim by the dispatcher's own clock. Past it the row's lease may
// be taken over and its successor dispatched, so a late call could reorder
// the target. A row the poll reaches after the deadline is backed off without
// a Temporal call.
func (d *Dispatcher) Poll(ctx context.Context) (int, error) {
	deadline := d.now().Add(d.relay.opts.Lease / 2)
	rows, err := d.relay.Claim(ctx)
	if err != nil {
		return 0, err
	}
	sem := make(chan struct{}, d.opts.Concurrency)
	var wg sync.WaitGroup
	for _, c := range rows {
		sem <- struct{}{}
		wg.Go(func() {
			defer func() { <-sem }()
			d.dispatch(ctx, c, deadline)
		})
	}
	wg.Wait()
	return len(rows), nil
}

// dispatch sends c and settles it. The span is a new root linked to the
// producer's trace (decision log 2026-10-25, T29), and carries only the
// outbox id; the SDK's child span adds the workflow and run id.
func (d *Dispatcher) dispatch(ctx context.Context, c Claimed, deadline time.Time) {
	op := opSignal
	if c.IsStart() {
		op = opStart
	}
	ctx, span := d.tracer.Start(ctx, "outbox.dispatch "+op,
		trace.WithNewRoot(), trace.WithSpanKind(trace.SpanKindProducer),
		trace.WithLinks(producerLink(c.TraceContext)...),
		trace.WithAttributes(telemetry.OutboxIDKey.String(c.ID.String())))
	defer span.End()

	log := d.log.With(slog.String("tenant_id", c.TenantID.String()), slog.String("outbox_id", c.ID.String()),
		slog.String("op", op), slog.Int("attempts", int(c.Attempts)))
	workflowID, ok := WorkflowID(c)
	if !ok {
		d.settle(ctx, log, c, op, outcomeInvalidTarget, CauseInvalidTarget)
		return
	}
	log = log.With(slog.String("workflow_id", workflowID))
	// The same rule Enqueue applies, so a row written around it (T32b finding
	// 1) is never sent. The error is not logged: it names a payload field.
	if outbox.ValidatePayload(c.Payload) != nil {
		d.settle(ctx, log, c, op, outcomeInvalidPayload, CauseInvalidPayload)
		return
	}
	if !d.now().Before(deadline) {
		d.settle(ctx, log, c, op, outcomeBackoff, CauseLeaseTooShort)
		return
	}

	tctx, cancel := context.WithDeadline(workerguard.ContextWithTenant(ctx, c.TenantID), deadline)
	defer cancel()
	delivery := outbox.Delivery{OutboxID: c.ID, KindVersion: c.KindVersion, CausationID: c.CausationID,
		CausationDepth: c.CausationDepth, Payload: c.Payload}
	var err error
	if op == opStart {
		d.beforeStart(tctx, c)
		_, err = d.temporal.ExecuteWorkflow(tctx, startOptions(workflowID), c.Kind, delivery)
	} else {
		err = d.temporal.SignalWorkflow(tctx, workflowID, "", c.Kind, delivery)
	}
	outcome, cause := classify(op, err)
	if err != nil && outcome != outcomeAlreadyStarted {
		// The span carries the gRPC code, never the message. The SDK's child
		// span records the message too; the export filter drops it there
		// (TICK-191).
		code := errorCode(err)
		span.SetAttributes(attribute.String("error.type", code))
		span.SetStatus(codes.Error, code)
	}
	if outcome == outcomeBackoff {
		// The error's message is not logged: Temporal and SDK errors can
		// quote a payload. Its gRPC code is.
		log = log.With(slog.String("error_code", errorCode(err)))
	}
	d.settle(ctx, log, c, op, outcome, cause)
}

// settle records outcome for c: MarkDispatched, MarkDead or Backoff with
// cause. It uses a context that outlives ctx's cancellation.
func (d *Dispatcher) settle(ctx context.Context, log *slog.Logger, c Claimed, op, outcome, cause string) {
	sctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), settleTimeout)
	defer cancel()
	var err error
	switch outcome {
	case outcomeDispatched, outcomeAlreadyStarted:
		err = d.relay.MarkDispatched(sctx, c)
	case outcomeTargetMissing, outcomeInvalidTarget, outcomeInvalidPayload:
		err = d.relay.MarkDead(sctx, c, cause)
	default:
		var dead bool
		dead, err = d.relay.Backoff(sctx, c, cause)
		if dead {
			outcome = outcomeDead
		}
	}
	switch {
	case errors.Is(err, ErrLeaseLost):
		log.WarnContext(ctx, "outbox row lease lost before it was settled", slog.String("outcome", outcome))
		outcome = outcomeLeaseLost
	case err != nil:
		log.ErrorContext(ctx, "outbox row not settled; it is redelivered after its lease",
			slog.String("outcome", outcome), slog.Any("error", err))
		outcome = outcomeSettleFailed
	}
	d.dispatches.Add(ctx, 1, metric.WithAttributes(attrOp.String(op), attrOutcome.String(outcome)))

	switch outcome {
	case outcomeTargetMissing:
		d.missing.Add(ctx, 1)
		log.ErrorContext(ctx, "outbox signal target workflow not found or completed; row marked dead")
	case outcomeInvalidTarget:
		log.ErrorContext(ctx, "outbox row target_key is not <kind>:<uuid>; row marked dead")
	case outcomeInvalidPayload:
		d.refused.Add(ctx, 1)
		log.ErrorContext(ctx, "outbox row payload is not references only; row marked dead without a temporal call",
			slog.Bool("alert", true))
	case outcomeDead:
		log.ErrorContext(ctx, "outbox row passed its max redelivery age and was marked dead", slog.String("cause", cause))
	case outcomeBackoff:
		log.WarnContext(ctx, "outbox dispatch failed; row backed off", slog.String("cause", cause))
	case outcomeDispatched, outcomeAlreadyStarted:
		log.DebugContext(ctx, "outbox row dispatched", slog.String("outcome", outcome))
	}
}

// startOptions are A.2's start options: REJECT_DUPLICATE, and AlreadyStarted
// returned as an error instead of the existing run.
func startOptions(workflowID string) client.StartWorkflowOptions {
	return client.StartWorkflowOptions{
		ID:                                       workflowID,
		TaskQueue:                                TaskQueue,
		WorkflowIDReusePolicy:                    enumspb.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE,
		WorkflowExecutionErrorWhenAlreadyStarted: true,
	}
}

// WorkflowID is the Temporal workflow id of c, t:<tenant_id>:<target_key>
// (A.2), and whether c may be dispatched at all. target_key must be
// <kind>:<uuid in canonical form> with a kind outbox.ValidKind accepts, and
// c.Kind (the workflow type or signal name) must be valid too. That form
// carries no free text, so the id carries no PII (TICK-94 item 3). Nothing is
// read from any other column.
func WorkflowID(c Claimed) (string, bool) {
	if c.TenantID == uuid.Nil || !outbox.ValidKind(c.Kind) {
		return "", false
	}
	kind, id, ok := strings.Cut(c.TargetKey, ":")
	if !ok || !outbox.ValidKind(kind) {
		return "", false
	}
	if u, err := uuid.Parse(id); err != nil || u == uuid.Nil || u.String() != id {
		return "", false
	}
	return "t:" + c.TenantID.String() + ":" + c.TargetKey, true
}

// classify maps a Temporal call's error to an outcome and the cause to
// record. AlreadyStarted is dispatched (REJECT_DUPLICATE: a redelivered start
// starts nothing). A signal's NotFound, which is also what Temporal returns
// for a completed workflow, is terminal. Everything else is retried.
func classify(op string, err error) (outcome, cause string) {
	if err == nil {
		return outcomeDispatched, ""
	}
	if op == opStart {
		var started *serviceerror.WorkflowExecutionAlreadyStarted
		if errors.As(err, &started) {
			return outcomeAlreadyStarted, ""
		}
	} else {
		var notFound *serviceerror.NotFound
		if errors.As(err, &notFound) {
			return outcomeTargetMissing, CauseTargetMissing
		}
	}
	return outcomeBackoff, failureCause(op, err)
}

// failureCause is the last_error of a retried dispatch: fixed text, the op
// and the error's code. It never includes err.Error().
func failureCause(op string, err error) string {
	return "relay: temporal " + op + " failed: " + errorCode(err)
}

// errorCode is a bounded name for err: a gRPC code name for a Temporal
// service error, else the context error, else Unknown.
func errorCode(err error) string {
	var svc serviceerror.ServiceError
	switch {
	case err == nil:
		return "OK"
	case errors.As(err, &svc):
		return svc.Status().Code().String()
	case errors.Is(err, context.DeadlineExceeded):
		return "DeadlineExceeded"
	case errors.Is(err, context.Canceled):
		return "Canceled"
	}
	return "Unknown"
}

// producerLink is a link to the producer's span from the row's traceparent,
// or none.
func producerLink(traceparent *string) []trace.Link {
	if traceparent == nil {
		return nil
	}
	ctx := propagation.TraceContext{}.Extract(context.Background(),
		propagation.MapCarrier{"traceparent": *traceparent})
	sc := trace.SpanContextFromContext(ctx)
	if !sc.IsValid() {
		return nil
	}
	return []trace.Link{{SpanContext: sc}}
}

package relay

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel/baggage"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
	"go.temporal.io/api/serviceerror"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
	"github.com/Navaladi2019/TicketingSystem/internal/workerguard"
)

// recordMarker stands for record data or a payload value that a Temporal or
// SDK error message quotes.
const recordMarker = "record-data-alice@example.com-7f3a"

// settleCall is one Mark* or Backoff statement the fake database received.
type settleCall struct {
	op    string // dispatched, dead, backoff
	id    uuid.UUID
	cause string
}

// fakeDB answers the settle statements as if the lease still held, and
// records them. Claim is not called through it.
type fakeDB struct {
	mu    sync.Mutex
	calls []settleCall
	// dead makes Backoff report the row past MaxRedeliveryAge.
	dead bool
}

func (f *fakeDB) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, errors.New("fakeDB: Claim is not supported")
}

func (f *fakeDB) QueryRow(_ context.Context, sql string, args ...any) pgx.Row {
	f.mu.Lock()
	defer f.mu.Unlock()
	c := settleCall{id: args[1].(uuid.UUID)}
	switch {
	case sql == backoffSQL:
		c.op, c.cause = "backoff", args[4].(string)
	case strings.Contains(sql, "SET dispatched_at"):
		c.op = "dispatched"
	case strings.Contains(sql, "SET dead_at"):
		c.op, c.cause = "dead", args[3].(string)
	}
	f.calls = append(f.calls, c)
	return fakeRow{dead: f.dead}
}

func (f *fakeDB) only(t *testing.T) settleCall {
	t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.calls) != 1 {
		t.Fatalf("%d settle calls %+v, want exactly 1", len(f.calls), f.calls)
	}
	return f.calls[0]
}

type fakeRow struct{ dead bool }

func (r fakeRow) Scan(dest ...any) error {
	if len(dest) == 2 { // Backoff: dead, pushed
		*dest[0].(*bool) = r.dead
	}
	return nil
}

// harness is a Dispatcher on a fakeDB and the in-process frontend, with its
// metrics, logs and spans readable.
type harness struct {
	d      *Dispatcher
	db     *fakeDB
	fe     *frontend
	reader *sdkmetric.ManualReader
	logs   *bytes.Buffer
	spans  *tracetest.InMemoryExporter
}

func newHarness(t *testing.T) harness {
	t.Helper()
	fe, addr := startFrontend(t)
	spans := tracetest.NewInMemoryExporter()
	tel := newTelemetry(t, telemetry.WithSpanExporter(spans), telemetry.WithSampler(sdktrace.AlwaysSample()))
	tc := dialFrontend(t, addr, tel)
	db := &fakeDB{}
	var logs bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug}))
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	r, err := New(db, mp, log, Options{Lease: 10 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	d, err := NewDispatcher(r, tc, tel.TracerProvider, mp, log, DispatcherOptions{})
	if err != nil {
		t.Fatal(err)
	}
	return harness{d: d, db: db, fe: fe, reader: reader, logs: &logs, spans: spans}
}

func (h harness) dispatch(ctx context.Context, c Claimed) {
	h.d.dispatch(ctx, c, h.d.now().Add(h.d.relay.opts.Lease/2))
}

// counter returns the sum of metric name's data points whose attributes
// match attrs (all of them when attrs is empty), and the number of points.
func (h harness) counter(t *testing.T, name string, attrs map[string]string) (sum int64, points int) {
	t.Helper()
	var rm metricdata.ResourceMetrics
	if err := h.reader.Collect(context.Background(), &rm); err != nil {
		t.Fatal(err)
	}
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != name {
				continue
			}
		points:
			for _, dp := range m.Data.(metricdata.Sum[int64]).DataPoints {
				points++
				for k, v := range attrs {
					if got, _ := dp.Attributes.Value(attrKey(k)); got.AsString() != v {
						continue points
					}
				}
				sum += dp.Value
			}
		}
	}
	return sum, points
}

// The tenant and record of the rows these tests build.
const (
	tenantAID = "0192aaaa-0000-7000-8000-00000000000a"
	recordXID = "0192cccc-0000-7000-8000-0000000000cc"
)

// startRow is a claimed start row of kind ticket.approval.
func startRow() Claimed {
	id := uuid.Must(uuid.NewV7())
	return Claimed{ID: id, TenantID: uuid.MustParse(tenantAID), Kind: "ticket.approval", KindVersion: 1,
		TargetKey: "ticket.approval:" + id.String(), TargetSeq: 1, Attempts: 1,
		Payload: json.RawMessage(`{"record_id":"` + recordXID + `"}`)}
}

// signalRow is a claimed signal row to the workflow start started.
func signalRow(start Claimed, seq int64) Claimed {
	return Claimed{ID: uuid.Must(uuid.NewV7()), TenantID: start.TenantID, Kind: "approve", KindVersion: 1,
		TargetKey: start.TargetKey, TargetSeq: seq, Attempts: 1, Payload: json.RawMessage(`{}`)}
}

func wfID(c Claimed) string { return "t:" + c.TenantID.String() + ":" + c.TargetKey }

// A.2: a start uses REJECT_DUPLICATE on nava-tq, the workflow id built from
// tenant_id and target_key, the kind as the workflow type and the Delivery as
// input; the nava-tenant header is the row's tenant.
func TestStartRequest(t *testing.T) {
	h := newHarness(t)
	c := startRow()
	h.dispatch(t.Context(), c)

	req := h.fe.lastStart()
	if req == nil {
		t.Fatal("no start reached the frontend")
	}
	if req.GetWorkflowId() != wfID(c) || req.GetWorkflowType().GetName() != c.Kind || !reusePolicyOK(req) {
		t.Errorf("start id %q type %q policy %v queue %q", req.GetWorkflowId(), req.GetWorkflowType().GetName(),
			req.GetWorkflowIdReusePolicy(), req.GetTaskQueue().GetName())
	}
	if got, _ := headerString(t, req.GetHeader(), workerguard.TenantHeader); got != tenantAID {
		t.Errorf("nava-tenant header %q, want %s", got, tenantAID)
	}
	var d outbox.Delivery
	if err := json.Unmarshal(req.GetInput().GetPayloads()[0].GetData(), &d); err != nil || d.OutboxID != c.ID ||
		string(d.Payload) != string(c.Payload) {
		t.Errorf("input %s (%v), want the Delivery of %s", req.GetInput().GetPayloads()[0].GetData(), err, c.ID)
	}
	if got := h.db.only(t); got.op != "dispatched" || got.id != c.ID {
		t.Errorf("settled %+v, want dispatched", got)
	}
}

// A.2: AlreadyStarted counts as dispatched, so a redelivered start starts
// nothing new.
func TestStartAlreadyStartedIsDispatched(t *testing.T) {
	h := newHarness(t)
	c := startRow()
	h.fe.addExecution(wfID(c), true) // closed: REJECT_DUPLICATE still refuses
	h.dispatch(t.Context(), c)
	if got := h.db.only(t); got.op != "dispatched" {
		t.Errorf("settled %+v, want dispatched", got)
	}
	if n, _, _ := h.fe.counts(); n != 1 {
		t.Errorf("%d executions, want 1", n)
	}
	if sum, _ := h.counter(t, "outbox_relay_dispatches", map[string]string{"op": "start", "outcome": "already_started"}); sum != 1 {
		t.Errorf("already_started count %d, want 1", sum)
	}
}

// A.2: a signal's NotFound, missing or already completed, marks the row dead
// at once and raises the alert metric, which has no attributes; the log line
// has the workflow id.
func TestSignalTargetMissingIsDeadWithAlert(t *testing.T) {
	for _, completed := range []bool{false, true} {
		t.Run(fmt.Sprintf("completed=%v", completed), func(t *testing.T) {
			h := newHarness(t)
			start := startRow()
			if completed {
				h.fe.addExecution(wfID(start), true)
			}
			c := signalRow(start, 2)
			h.dispatch(t.Context(), c)
			if got := h.db.only(t); got.op != "dead" || got.cause != CauseTargetMissing {
				t.Errorf("settled %+v, want dead with %q", got, CauseTargetMissing)
			}
			sum, points := h.counter(t, "outbox_relay_signal_target_missing", nil)
			if sum != 1 || points != 1 {
				t.Errorf("alert metric %d over %d points, want 1 over 1", sum, points)
			}
			var rm metricdata.ResourceMetrics
			_ = h.reader.Collect(context.Background(), &rm)
			for _, sm := range rm.ScopeMetrics {
				for _, m := range sm.Metrics {
					if m.Name == "outbox_relay_signal_target_missing" {
						if a := m.Data.(metricdata.Sum[int64]).DataPoints[0].Attributes; a.Len() != 0 {
							t.Errorf("alert metric attributes %v, want none", a)
						}
					}
				}
			}
			if !strings.Contains(h.logs.String(), `"workflow_id":"`+wfID(start)+`"`) {
				t.Errorf("log has no workflow_id:\n%s", h.logs)
			}
		})
	}
}

// A signal carries the Delivery with its outbox id (I7) and the tenant
// header.
func TestSignalRequest(t *testing.T) {
	h := newHarness(t)
	start := startRow()
	h.fe.addExecution(wfID(start), false)
	c := signalRow(start, 2)
	h.dispatch(t.Context(), c)
	if got := h.db.only(t); got.op != "dispatched" {
		t.Fatalf("settled %+v, want dispatched", got)
	}
	e, _ := h.fe.execution(wfID(start))
	if len(e.signals) != 1 || e.signals[0] != c.ID {
		t.Errorf("signals %v, want [%s]", e.signals, c.ID)
	}
	h.fe.mu.Lock()
	req := h.fe.signals[0]
	h.fe.mu.Unlock()
	if req.GetSignalName() != c.Kind {
		t.Errorf("signal name %q, want %q", req.GetSignalName(), c.Kind)
	}
	if got, _ := headerString(t, req.GetHeader(), workerguard.TenantHeader); got != tenantAID {
		t.Errorf("nava-tenant header %q, want %s", got, tenantAID)
	}
}

// causeShape is every cause the dispatcher writes for a retried failure.
var causeShape = regexp.MustCompile(`^relay: temporal (start|signal) failed: [A-Za-z]+$`)

// TICK-74 security review: a dispatch error whose message quotes record data
// is backed off, and the cause that reaches last_error holds none of it.
// Every error path calls Backoff (or MarkDead / MarkDispatched): exactly one
// settle per row.
func TestEveryDispatchErrorBacksOffWithACleanCause(t *testing.T) {
	cases := map[string]error{
		"InvalidArgument":    serviceerror.NewInvalidArgument("bad input " + recordMarker),
		"PermissionDenied":   serviceerror.NewPermissionDenied("denied for "+recordMarker, "workflow_type_not_allowed"),
		"Internal":           serviceerror.NewInternal("history said " + recordMarker),
		"FailedPrecondition": status.Error(codes.FailedPrecondition, "precondition "+recordMarker),
		// NotFound on a start is not terminal.
		"NotFoundOnStart": serviceerror.NewNotFound("namespace " + recordMarker),
	}
	for name, injected := range cases {
		for _, op := range []string{opStart, opSignal} {
			if name == "NotFoundOnStart" && op == opSignal {
				continue
			}
			t.Run(name+"/"+op, func(t *testing.T) {
				h := newHarness(t)
				start := startRow()
				c := start
				if op == opSignal {
					h.fe.addExecution(wfID(start), false)
					c = signalRow(start, 2)
				}
				h.fe.setFail(func(string, string) error { return injected })
				h.dispatch(t.Context(), c)
				got := h.db.only(t)
				if got.op != "backoff" {
					t.Fatalf("settled %+v, want backoff", got)
				}
				if strings.Contains(got.cause, recordMarker) || !causeShape.MatchString(got.cause) {
					t.Errorf("cause %q carries error text or has the wrong shape", got.cause)
				}
				if strings.Contains(h.logs.String(), recordMarker) {
					t.Errorf("the log carries the error text:\n%s", h.logs)
				}
			})
		}
	}
}

// A Temporal call that outlives the dispatch deadline, and a row reached
// after it, are both backed off; neither leaves the row unsettled.
func TestDeadlineBacksOff(t *testing.T) {
	h := newHarness(t)
	h.fe.mu.Lock()
	h.fe.delay = func(string) time.Duration { return time.Hour }
	h.fe.mu.Unlock()
	c := startRow()
	h.d.dispatch(t.Context(), c, time.Now().Add(200*time.Millisecond))
	if got := h.db.only(t); got.op != "backoff" || got.cause != "relay: temporal start failed: DeadlineExceeded" {
		t.Errorf("settled %+v, want backoff DeadlineExceeded", got)
	}

	h2 := newHarness(t)
	h2.d.dispatch(t.Context(), startRow(), time.Now().Add(-time.Second))
	if got := h2.db.only(t); got.op != "backoff" || got.cause != CauseLeaseTooShort {
		t.Errorf("settled %+v, want backoff %q", got, CauseLeaseTooShort)
	}
	if _, starts, _ := h2.fe.counts(); starts != 0 {
		t.Errorf("%d starts after the deadline, want 0", starts)
	}
}

// AR-026: a failing row past MaxRedeliveryAge (by created_at, decided in
// Backoff's SQL) is counted dead, not backed off.
func TestBackoffDeadIsCounted(t *testing.T) {
	h := newHarness(t)
	h.db.dead = true
	h.fe.setFail(func(string, string) error { return serviceerror.NewInvalidArgument("x") })
	h.dispatch(t.Context(), startRow())
	if sum, _ := h.counter(t, "outbox_relay_dispatches", map[string]string{"outcome": "dead"}); sum != 1 {
		t.Errorf("dead count %d, want 1", sum)
	}
}

// TICK-94 item 3: target_key must be <kind>:<canonical uuid>. A row in any
// other form is marked dead without a Temporal call.
func TestInvalidTargetKeyIsDead(t *testing.T) {
	id := uuid.Must(uuid.NewV7())
	for name, c := range map[string]Claimed{
		"email":        {Kind: "approve", TargetKey: "ticket.approval:alice@example.com"},
		"name":         {Kind: "approve", TargetKey: "ticket.approval:Alice Smith"},
		"no kind":      {Kind: "approve", TargetKey: id.String()},
		"upper uuid":   {Kind: "approve", TargetKey: "ticket.approval:" + strings.ToUpper(id.String())},
		"nil uuid":     {Kind: "approve", TargetKey: "ticket.approval:" + uuid.Nil.String()},
		"bad kind":     {Kind: "approve", TargetKey: "Ticket Approval:" + id.String()},
		"extra part":   {Kind: "approve", TargetKey: "ticket.approval:" + id.String() + ":x"},
		"bad row kind": {Kind: "Approve Now", TargetKey: "ticket.approval:" + id.String()},
	} {
		t.Run(name, func(t *testing.T) {
			h := newHarness(t)
			c.ID, c.TenantID, c.Attempts = uuid.Must(uuid.NewV7()), uuid.MustParse(tenantAID), 1
			h.dispatch(t.Context(), c)
			if got := h.db.only(t); got.op != "dead" || got.cause != CauseInvalidTarget {
				t.Errorf("settled %+v, want dead %q", got, CauseInvalidTarget)
			}
			if _, starts, signals := h.fe.counts(); starts+signals != 0 {
				t.Errorf("%d Temporal calls, want 0", starts+signals)
			}
		})
	}
}

// T32b finding 1, second line: a row whose payload is not references only
// (written around Enqueue) never reaches Temporal. It is dead at its first
// attempt with a fixed cause, and it alerts; nothing of the payload reaches
// the log, last_error or the span.
func TestRecordDataPayloadIsRefusedBeforeDispatch(t *testing.T) {
	for name, payload := range map[string]string{
		"string value":  `{"record_id":"` + recordXID + `","email":"` + recordMarker + `"}`,
		"object value":  `{"record":{"email":"` + recordMarker + `"}}`,
		"array value":   `{"ids":["` + recordXID + `","` + recordMarker + `"]}`,
		"number value":  `{"amount":42}`,
		"not an object": `["` + recordMarker + `"]`,
		"bare string":   `"` + recordMarker + `"`,
		"null":          `null`,
	} {
		for _, op := range []string{opStart, opSignal} {
			t.Run(name+"/"+op, func(t *testing.T) {
				h := newHarness(t)
				start := startRow()
				c := start
				if op == opSignal {
					h.fe.addExecution(wfID(start), false)
					c = signalRow(start, 2)
				}
				c.Payload = json.RawMessage(payload)
				h.dispatch(t.Context(), c)

				if _, starts, signals := h.fe.counts(); starts+signals != 0 {
					t.Errorf("%d Temporal calls, want 0", starts+signals)
				}
				if got := h.db.only(t); got.op != "dead" || got.id != c.ID || got.cause != CauseInvalidPayload {
					t.Errorf("settled %+v, want dead %q at its first attempt", got, CauseInvalidPayload)
				}
				if sum, points := h.counter(t, "outbox_relay_payload_refused", nil); sum != 1 || points != 1 {
					t.Errorf("alert metric %d over %d points, want 1 over 1 (no attributes)", sum, points)
				}
				if sum, _ := h.counter(t, "outbox_relay_dispatches", map[string]string{"op": op, "outcome": outcomeInvalidPayload}); sum != 1 {
					t.Errorf("outbox_relay_dispatches{outcome=invalid_payload} = %d, want 1", sum)
				}
				logs := h.logs.String()
				if !strings.Contains(logs, `"alert":true`) || !strings.Contains(logs, `"outbox_id":"`+c.ID.String()+`"`) {
					t.Errorf("no alert log line naming the outbox id:\n%s", logs)
				}
				spans := h.spans.GetSpans()
				if len(spans) == 0 {
					t.Fatal("no dispatch span")
				}
				for _, where := range []string{logs, fmt.Sprintf("%+v", spans)} {
					if strings.Contains(where, recordMarker) {
						t.Errorf("the payload marker leaked:\n%s", where)
					}
				}
			})
		}
	}
}

func TestWorkflowID(t *testing.T) {
	c := startRow()
	if got, ok := WorkflowID(c); !ok || got != "t:"+tenantAID+":"+c.TargetKey {
		t.Errorf("WorkflowID = %q, %v", got, ok)
	}
	c.TenantID = uuid.Nil
	if _, ok := WorkflowID(c); ok {
		t.Error("WorkflowID accepted a nil tenant")
	}
}

// TICK-94 item 5: the _tracer-data header of a started workflow holds only
// traceparent (and tracestate), with no baggage, even when the dispatching
// context has baggage. The dispatch span is a new root linked to the
// producer's trace and carries the outbox id.
func TestTracerHeaderHasNoBaggage(t *testing.T) {
	h := newHarness(t)
	m, _ := baggage.NewMember("user.email", "alice@example.com")
	b, _ := baggage.New(m)
	ctx := baggage.ContextWithBaggage(t.Context(), b)
	// An ambient span the dispatch span must not be a child of.
	ambient, ambientSpan := sdktrace.NewTracerProvider().Tracer("t").Start(ctx, "poll")
	defer ambientSpan.End()

	c := startRow()
	producer := "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"
	c.TraceContext = &producer
	h.dispatch(ambient, c)

	req := h.fe.lastStart()
	p := req.GetHeader().GetFields()["_tracer-data"]
	if p == nil {
		t.Fatal("no _tracer-data header")
	}
	var carrier map[string]string
	if err := json.Unmarshal(p.GetData(), &carrier); err != nil {
		t.Fatalf("_tracer-data %s: %v", p.GetData(), err)
	}
	if _, ok := carrier["traceparent"]; !ok {
		t.Errorf("_tracer-data %v has no traceparent", carrier)
	}
	for k := range carrier {
		if k != "traceparent" && k != "tracestate" {
			t.Errorf("_tracer-data carries %q", k)
		}
	}
	if strings.Contains(string(p.GetData()), "alice") {
		t.Errorf("_tracer-data carries the baggage: %s", p.GetData())
	}

	var dispatch *tracetest.SpanStub
	for _, s := range h.spans.GetSpans() {
		if s.Name == "outbox.dispatch start" {
			dispatch = &s
		}
	}
	if dispatch == nil {
		t.Fatal("no outbox.dispatch start span")
	}
	if dispatch.Parent.IsValid() {
		t.Errorf("dispatch span has parent %v, want a new root", dispatch.Parent)
	}
	if len(dispatch.Links) != 1 || dispatch.Links[0].SpanContext.TraceID().String() != "4bf92f3577b34da6a3ce929d0e0e4736" {
		t.Errorf("dispatch span links %+v, want the producer's trace", dispatch.Links)
	}
	if !hasAttr(dispatch.Attributes, telemetry.OutboxIDKey, c.ID.String()) {
		t.Errorf("dispatch span attributes %v, want the outbox id", dispatch.Attributes)
	}
	if got := trace.SpanContextFromContext(ambient).TraceID(); dispatch.SpanContext.TraceID() == got {
		t.Error("dispatch span joined the ambient trace")
	}
}

// The crash seam pauses a start until released, writing the marker first.
func TestStdinPause(t *testing.T) {
	r, w := io.Pipe()
	var out syncBuffer
	pause := StdinPause(r, &out)
	c := startRow()
	done := make(chan struct{})
	go func() { pause(context.Background(), c); close(done) }()
	waitFor(t, func() bool { return strings.Contains(out.String(), PauseMarker+" "+c.ID.String()) })
	select {
	case <-done:
		t.Fatal("the seam did not pause")
	case <-time.After(50 * time.Millisecond):
	}
	_, _ = w.Write([]byte("release\n"))
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("a line on stdin did not release the seam")
	}

	ctx, cancel := context.WithCancel(context.Background())
	done = make(chan struct{})
	go func() { pause(ctx, c); close(done) }()
	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("a cancelled context did not release the seam")
	}
}

func TestErrorCode(t *testing.T) {
	for err, want := range map[error]string{
		serviceerror.NewUnavailable("x " + recordMarker):    "Unavailable",
		fmt.Errorf("wrapped: %w", context.DeadlineExceeded): "DeadlineExceeded",
		context.Canceled:                     "Canceled",
		errors.New("encode " + recordMarker): "Unknown",
	} {
		if got := errorCode(err); got != want {
			t.Errorf("errorCode(%v) = %q, want %q", err, got, want)
		}
	}
}

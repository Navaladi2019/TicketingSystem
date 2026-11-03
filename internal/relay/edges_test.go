package relay

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
)

var errMeter = errors.New("meter refused the instrument")

// failingMeterProvider hands out a meter that refuses the instrument named
// fail and registers every other one.
type failingMeterProvider struct {
	metricnoop.MeterProvider
	fail string
}

func (p failingMeterProvider) Meter(string, ...metric.MeterOption) metric.Meter {
	return failingMeter{fail: p.fail}
}

type failingMeter struct {
	metricnoop.Meter
	fail string
}

func (m failingMeter) Int64Counter(name string, o ...metric.Int64CounterOption) (metric.Int64Counter, error) {
	if name == m.fail {
		return nil, errMeter
	}
	return m.Meter.Int64Counter(name, o...)
}

func (m failingMeter) Int64Histogram(name string, o ...metric.Int64HistogramOption) (metric.Int64Histogram, error) {
	if name == m.fail {
		return nil, errMeter
	}
	return m.Meter.Int64Histogram(name, o...)
}

// errDB fails every statement: Claim's query with queryErr, each settle's
// row with rowErr.
type errDB struct {
	queryErr, rowErr error
	rows             int
}

func (f *errDB) Query(context.Context, string, ...any) (pgx.Rows, error) { return nil, f.queryErr }

func (f *errDB) QueryRow(context.Context, string, ...any) pgx.Row {
	f.rows++
	return errRow{f.rowErr}
}

type errRow struct{ err error }

func (r errRow) Scan(...any) error { return r.err }

// nopTemporal is a Temporal that is never called.
type nopTemporal struct{}

func (nopTemporal) ExecuteWorkflow(context.Context, client.StartWorkflowOptions, any, ...any) (client.WorkflowRun, error) {
	return nil, errors.New("nopTemporal: not called")
}

func (nopTemporal) SignalWorkflow(context.Context, string, string, string, any) error {
	return errors.New("nopTemporal: not called")
}

func TestNewMeterError(t *testing.T) {
	r, err := New(&fakeDB{}, failingMeterProvider{fail: "outbox_relay_rows_scanned"}, slog.New(slog.DiscardHandler), Options{})
	if !errors.Is(err, errMeter) || r != nil || !strings.HasPrefix(err.Error(), "relay: ") {
		t.Fatalf("New = %v, %v; want nil, relay: %v", r, err, errMeter)
	}
}

func TestNewDispatcherRequiresDependencies(t *testing.T) {
	log := slog.New(slog.DiscardHandler)
	mp := metricnoop.NewMeterProvider()
	tp := tracenoop.NewTracerProvider()
	r, err := New(&fakeDB{}, mp, log, Options{})
	if err != nil {
		t.Fatal(err)
	}
	const want = "relay: nil relay, temporal client, provider or logger"
	for name, build := range map[string]func() (*Dispatcher, error){
		"relay": func() (*Dispatcher, error) {
			return NewDispatcher(nil, nopTemporal{}, tp, mp, log, DispatcherOptions{})
		},
		"temporal": func() (*Dispatcher, error) { return NewDispatcher(r, nil, tp, mp, log, DispatcherOptions{}) },
		"tracer":   func() (*Dispatcher, error) { return NewDispatcher(r, nopTemporal{}, nil, mp, log, DispatcherOptions{}) },
		"meter":    func() (*Dispatcher, error) { return NewDispatcher(r, nopTemporal{}, tp, nil, log, DispatcherOptions{}) },
		"logger":   func() (*Dispatcher, error) { return NewDispatcher(r, nopTemporal{}, tp, mp, nil, DispatcherOptions{}) },
	} {
		if d, err := build(); d != nil || err == nil || err.Error() != want {
			t.Errorf("nil %s: %v, %v; want %q", name, d, err, want)
		}
	}
	for _, name := range []string{"outbox_relay_dispatches", "outbox_relay_signal_target_missing", "outbox_relay_payload_refused"} {
		d, err := NewDispatcher(r, nopTemporal{}, tp, failingMeterProvider{fail: name}, log, DispatcherOptions{})
		if !errors.Is(err, errMeter) || d != nil {
			t.Errorf("%s refused: %v, %v; want nil, %v", name, d, err, errMeter)
		}
	}
	d, err := NewDispatcher(r, nopTemporal{}, tp, mp, log, DispatcherOptions{})
	if err != nil || d.opts.Concurrency != 8 {
		t.Fatalf("defaults: %v, concurrency %d; want 8", err, d.opts.Concurrency)
	}
}

// Duplicate delivery: a start delivered a second time after the first
// delivery was settled (its lease taken over, or it was already marked) is
// AlreadyStarted at Temporal, so one workflow runs, and its settle finds the
// row no longer under this lease: counted lease_lost, never settled twice.
func TestRedeliveredStartAfterSettleIsLeaseLost(t *testing.T) {
	h := newHarness(t)
	c := startRow()
	h.dispatch(t.Context(), c)
	if got := h.db.only(t); got.op != "dispatched" {
		t.Fatalf("first delivery settled %+v, want dispatched", got)
	}
	// The row is dispatched now: MarkDispatched matches no row.
	lost := &errDB{rowErr: pgx.ErrNoRows}
	h.d.relay.db = lost
	h.dispatch(t.Context(), c)
	if lost.rows != 1 {
		t.Errorf("%d settle statements for the redelivery, want 1", lost.rows)
	}
	if n, starts, _ := h.fe.counts(); n != 1 || starts != 2 {
		t.Errorf("%d executions from %d starts, want 1 from 2", n, starts)
	}
	if sum, _ := h.counter(t, "outbox_relay_dispatches", map[string]string{"op": "start", "outcome": "lease_lost"}); sum != 1 {
		t.Errorf("lease_lost count %d, want 1", sum)
	}
	if sum, _ := h.counter(t, "outbox_relay_dispatches", map[string]string{"op": "start", "outcome": "dispatched"}); sum != 1 {
		t.Errorf("dispatched count %d, want 1 (the first delivery only)", sum)
	}
	if !strings.Contains(h.logs.String(), "outbox row lease lost before it was settled") {
		t.Errorf("no lease-lost warning in %s", h.logs)
	}
}

// A settle that fails for any other reason leaves the row claimed; it is
// counted settle_failed and logged, whatever the outcome it tried to record.
func TestSettleFailureIsCounted(t *testing.T) {
	cases := map[string]func(h harness) Claimed{
		"mark dispatched": func(harness) Claimed { return startRow() },
		"mark dead": func(h harness) Claimed {
			return signalRow(startRow(), 2) // no execution: NotFound, so MarkDead
		},
		"backoff": func(h harness) Claimed {
			h.fe.setFail(func(string, string) error { return serviceerror.NewInvalidArgument("x") })
			return startRow()
		},
	}
	for name, row := range cases {
		t.Run(name, func(t *testing.T) {
			h := newHarness(t)
			c := row(h)
			db := &errDB{rowErr: errors.New("connection reset")}
			h.d.relay.db = db
			h.dispatch(t.Context(), c)
			if db.rows != 1 {
				t.Errorf("%d settle statements, want 1", db.rows)
			}
			if sum, _ := h.counter(t, "outbox_relay_dispatches", map[string]string{"outcome": "settle_failed"}); sum != 1 {
				t.Errorf("settle_failed count %d, want 1", sum)
			}
			if !strings.Contains(h.logs.String(), "outbox row not settled; it is redelivered after its lease") {
				t.Errorf("no settle failure in %s", h.logs)
			}
		})
	}
}

// Mark* and Backoff wrap a database error with their op, and map no row to
// ErrLeaseLost.
func TestSettleErrors(t *testing.T) {
	boom := errors.New("connection reset")
	r, err := New(&errDB{rowErr: boom}, metricnoop.NewMeterProvider(), slog.New(slog.DiscardHandler), Options{})
	if err != nil {
		t.Fatal(err)
	}
	c := startRow()
	if err := r.MarkDispatched(t.Context(), c); !errors.Is(err, boom) || !strings.HasPrefix(err.Error(), "relay: mark dispatched: ") {
		t.Errorf("MarkDispatched = %v", err)
	}
	if err := r.MarkDead(t.Context(), c, "x"); !errors.Is(err, boom) || !strings.HasPrefix(err.Error(), "relay: mark dead: ") {
		t.Errorf("MarkDead = %v", err)
	}
	if dead, err := r.Backoff(t.Context(), c, "x"); dead || !errors.Is(err, boom) || !strings.HasPrefix(err.Error(), "relay: backoff: ") {
		t.Errorf("Backoff = %v, %v", dead, err)
	}
	r.db = &errDB{rowErr: pgx.ErrNoRows}
	if dead, err := r.Backoff(t.Context(), c, "x"); dead || !errors.Is(err, ErrLeaseLost) {
		t.Errorf("Backoff on no row = %v, %v; want ErrLeaseLost", dead, err)
	}
}

// A failed claim is Poll's error, and Run logs it and keeps polling until
// its context ends.
func TestPollAndRunClaimError(t *testing.T) {
	boom := errors.New("connection refused")
	logs := &syncBuffer{}
	log := slog.New(slog.NewJSONHandler(logs, nil))
	mp := metricnoop.NewMeterProvider()
	r, err := New(&errDB{queryErr: boom}, mp, log, Options{})
	if err != nil {
		t.Fatal(err)
	}
	d, err := NewDispatcher(r, nopTemporal{}, tracenoop.NewTracerProvider(), mp, log, DispatcherOptions{})
	if err != nil {
		t.Fatal(err)
	}
	n, err := d.Poll(t.Context())
	if n != 0 || !errors.Is(err, boom) || !strings.HasPrefix(err.Error(), "relay: claim: ") {
		t.Fatalf("Poll = %d, %v; want 0, relay: claim: %v", n, err, boom)
	}

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() { d.Run(ctx, time.Hour); close(done) }()
	waitFor(t, func() bool { return strings.Contains(logs.String(), "outbox relay poll failed") })
	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return after its context ended")
	}
	if got := strings.Count(logs.String(), "outbox relay poll failed"); got != 1 {
		t.Errorf("%d poll failures logged, want 1 (the next poll waits the interval)", got)
	}
}

func TestErrorCodeNil(t *testing.T) {
	if got := errorCode(nil); got != "OK" {
		t.Errorf("errorCode(nil) = %q, want OK", got)
	}
}

func TestProducerLink(t *testing.T) {
	valid := "00-01020300000000000000000000000000-0405060000000000-01"
	if links := producerLink(&valid); len(links) != 1 || links[0].SpanContext.TraceID().String() != "01020300000000000000000000000000" {
		t.Errorf("valid traceparent: %+v, want one link to its trace", links)
	}
	for _, bad := range []string{"", "not-a-traceparent", "00-00000000000000000000000000000000-0405060000000000-01"} {
		if links := producerLink(&bad); links != nil {
			t.Errorf("traceparent %q: %+v, want no link", bad, links)
		}
	}
	if links := producerLink(nil); links != nil {
		t.Errorf("no traceparent: %+v, want no link", links)
	}
}

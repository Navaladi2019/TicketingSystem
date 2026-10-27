//go:build integration

// T30b against a real PostgreSQL as nava_relay (see relay_integration_test.go)
// and the real Temporal SDK client, built as in production, talking to the
// in-process frontend (frontend_test.go).

package relay

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.temporal.io/api/serviceerror"

	"github.com/Navaladi2019/TicketingSystem/internal/outbox"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// dispatcherOn is a Dispatcher as nava_relay on pool, sending to addr.
func dispatcherOn(t *testing.T, pool Querier, addr string, opts Options) (*Dispatcher, *sdkmetric.ManualReader, *syncBuffer) {
	t.Helper()
	tel := newTelemetry(t)
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	var logs syncBuffer
	log := slog.New(slog.NewJSONHandler(&logs, nil))
	r, err := New(pool, mp, log, opts)
	if err != nil {
		t.Fatal(err)
	}
	d, err := NewDispatcher(r, dialFrontend(t, addr, tel), tel.TracerProvider, mp, log, DispatcherOptions{})
	if err != nil {
		t.Fatal(err)
	}
	return d, reader, &logs
}

// producer writes rows through outbox.Enqueue as nava_app, one transaction
// each, on one record of tenant.
type producer struct {
	t      *testing.T
	e      env
	tenant uuid.UUID
	record uuid.UUID
	n      int
}

func newProducer(t *testing.T, e env) *producer {
	t.Helper()
	p := &producer{t: t, e: e, tenant: uuid.New(), record: uuid.New()}
	mustExec(t, e.super, "INSERT INTO tenants (id, state) VALUES ($1, 'active') ON CONFLICT DO NOTHING", p.tenant)
	mustExec(t, e.super, "INSERT INTO records (tenant_id, id) VALUES ($1, $2)", p.tenant, p.record)
	return p
}

func (p *producer) enqueue(target outbox.Target, kind string) uuid.UUID {
	p.t.Helper()
	p.n++
	var id uuid.UUID
	err := tenancy.InTenantTx(context.Background(), p.e.app, p.tenant, func(tx pgx.Tx) error {
		var err error
		id, err = outbox.Enqueue(context.Background(), tx, outbox.Entry{Kind: kind, KindVersion: 1,
			DedupeKey: fmt.Sprintf("%s-%d", p.record, p.n), RecordID: p.record, Target: target,
			Payload: []byte(`{"record_id":"` + p.record.String() + `"}`)})
		return err
	})
	if err != nil {
		p.t.Fatal(err)
	}
	return id
}

func (p *producer) start() uuid.UUID {
	return p.enqueue(outbox.TargetStart{Kind: "ticket.approval"}, "ticket.approval")
}

func (p *producer) signal() uuid.UUID {
	return p.enqueue(outbox.TargetSignal{Kind: "ticket.approval", SignalName: "approve"}, "approve")
}

func (p *producer) workflowID(start uuid.UUID) string {
	return "t:" + p.tenant.String() + ":ticket.approval:" + start.String()
}

// AC: forced redelivery of a start row starts one workflow. The first
// delivery reaches Temporal, then the relay dies before MarkDispatched; the
// lease ends and the row is delivered again. Temporal answers AlreadyStarted,
// which is dispatched.
func TestForcedRedeliveryStartsOneWorkflow(t *testing.T) {
	e := migrated(t)
	ctx := context.Background()
	fe, addr := startFrontend(t)
	p := newProducer(t, e)
	start := p.start()
	d, _, _ := dispatcherOn(t, e.relay, addr, Options{})

	cs, err := d.relay.Claim(ctx)
	if err != nil || len(cs) != 1 || cs[0].ID != start {
		t.Fatalf("claim %+v, %v", cs, err)
	}
	// The first delivery: the start reaches Temporal, nothing is marked.
	if _, err := d.temporal.ExecuteWorkflow(ctx, startOptions(p.workflowID(start)), "ticket.approval", nil); err != nil {
		t.Fatal(err)
	}
	mustExec(t, e.super, "UPDATE outbox SET available_at = now() - interval '1 second' WHERE id = $1", start)

	if n, err := d.Poll(ctx); err != nil || n != 1 {
		t.Fatalf("redelivery poll claimed %d, %v", n, err)
	}
	executions, starts, _ := fe.counts()
	if executions != 1 || starts != 2 {
		t.Errorf("%d executions from %d starts, want 1 from 2", executions, starts)
	}
	if n := count(t, e.super, "SELECT count(*) FROM outbox WHERE id = $1 AND dispatched_at IS NOT NULL AND attempts = 2", start); n != 1 {
		t.Error("the redelivered start is not dispatched at attempt 2")
	}
}

// AC: two relays never reorder one target's signals, while dispatches fail
// and back off at random and the frontend answers with random delays.
func TestTwoRelaysNeverReorderSignals(t *testing.T) {
	e := migrated(t)
	fe, addr := startFrontend(t)
	const targets, signals = 4, 12
	type target struct {
		p       *producer
		start   uuid.UUID
		signals []uuid.UUID
	}
	var ts []target
	for range targets {
		p := newProducer(t, e)
		tg := target{p: p, start: p.start()}
		for range signals {
			tg.signals = append(tg.signals, p.signal())
		}
		ts = append(ts, tg)
	}
	var mu sync.Mutex
	rng := rand.New(rand.NewPCG(1, 2))
	fe.mu.Lock()
	fe.delay = func(string) time.Duration {
		mu.Lock()
		defer mu.Unlock()
		return time.Duration(rng.IntN(15)) * time.Millisecond
	}
	fe.fail = func(string, string) error {
		mu.Lock()
		defer mu.Unlock()
		if rng.IntN(4) == 0 {
			return serviceerror.NewFailedPrecondition("injected")
		}
		return nil
	}
	fe.mu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	for range 2 {
		d, _, _ := dispatcherOn(t, e.relay, addr, Options{Lease: 4 * time.Second, BackoffBase: 5 * time.Millisecond, BackoffMax: 20 * time.Millisecond})
		wg.Go(func() { d.Run(ctx, 5*time.Millisecond) })
	}
	deadline := time.Now().Add(90 * time.Second)
	for count(t, e.super, "SELECT count(*) FROM outbox WHERE dispatched_at IS NULL") > 0 {
		if time.Now().After(deadline) {
			cancel()
			wg.Wait()
			t.Fatal("rows still pending after 90 s")
		}
		time.Sleep(20 * time.Millisecond)
	}
	cancel()
	wg.Wait()

	if n := count(t, e.super, "SELECT count(*) FROM outbox WHERE dead_at IS NOT NULL"); n != 0 {
		t.Errorf("%d rows dead, want 0", n)
	}
	for _, tg := range ts {
		ex, ok := fe.execution(tg.p.workflowID(tg.start))
		if !ok {
			t.Errorf("target %s never started", tg.start)
			continue
		}
		// A redelivered signal may repeat (the worker dedupes on outbox id);
		// it may never come before an earlier row's first delivery.
		var firsts []uuid.UUID
		seen := map[uuid.UUID]bool{}
		for _, id := range ex.signals {
			if !seen[id] {
				seen[id] = true
				firsts = append(firsts, id)
			}
		}
		if fmt.Sprint(firsts) != fmt.Sprint(tg.signals) {
			t.Errorf("target %s signal order %v, want %v", tg.start, firsts, tg.signals)
		}
	}
	if n, _, _ := fe.counts(); n != targets {
		t.Errorf("%d executions, want %d", n, targets)
	}
}

// AC: a signal whose workflow is NotFound marks the row dead at once, with
// the fixed cause, and emits the alert metric.
func TestSignalNotFoundMarksDead(t *testing.T) {
	e := migrated(t)
	ctx := context.Background()
	_, addr := startFrontend(t)
	p := newProducer(t, e)
	start := p.start()
	sig := p.signal()
	// The start was dispatched by an earlier relay, but its workflow is gone.
	mustExec(t, e.super, "UPDATE outbox SET dispatched_at = now() WHERE id = $1", start)
	d, reader, logs := dispatcherOn(t, e.relay, addr, Options{})

	if n, err := d.Poll(ctx); err != nil || n != 1 {
		t.Fatalf("poll claimed %d, %v", n, err)
	}
	if n := count(t, e.super, `SELECT count(*) FROM outbox WHERE id = $1 AND dead_at IS NOT NULL
		AND attempts = 1 AND last_error = $2`, sig, CauseTargetMissing); n != 1 {
		t.Error("the signal row is not dead after one attempt with the target-missing cause")
	}
	h := harness{reader: reader}
	if sum, points := h.counter(t, "outbox_relay_signal_target_missing", nil); sum != 1 || points != 1 {
		t.Errorf("alert metric %d over %d points, want 1", sum, points)
	}
	if !strings.Contains(logs.String(), p.workflowID(start)) {
		t.Errorf("log line has no workflow id:\n%s", logs.String())
	}
}

// TICK-74 security review: the last_error a failed dispatch stores holds no
// record data, even when Temporal's error message quotes it. A failing row
// past MaxRedeliveryAge, measured from created_at, is dead; one claimed long
// after it was written but younger than that is backed off.
func TestDispatchFailureLastError(t *testing.T) {
	e := migrated(t)
	ctx := context.Background()
	fe, addr := startFrontend(t)
	fe.setFail(func(string, string) error { return serviceerror.NewInvalidArgument("rejected " + recordMarker) })
	young, old := newProducer(t, e), newProducer(t, e)
	youngStart, oldStart := young.start(), old.start()
	// Both rows waited before their first claim; only the old one was created
	// more than MaxRedeliveryAge ago.
	mustExec(t, e.super, "UPDATE outbox SET created_at = now() - interval '23 hours' WHERE id = $1", youngStart)
	mustExec(t, e.super, "UPDATE outbox SET created_at = now() - interval '24 hours 1 minute' WHERE id = $1", oldStart)
	d, _, logs := dispatcherOn(t, e.relay, addr, Options{})

	if n, err := d.Poll(ctx); err != nil || n != 2 {
		t.Fatalf("poll claimed %d, %v", n, err)
	}
	want := "relay: temporal start failed: InvalidArgument"
	if n := count(t, e.super, `SELECT count(*) FROM outbox WHERE id = $1 AND dead_at IS NULL
		AND available_at > now() AND last_error = $2`, youngStart, want); n != 1 {
		t.Errorf("the young row is not backed off with last_error %q", want)
	}
	if n := count(t, e.super, `SELECT count(*) FROM outbox WHERE id = $1 AND dead_at IS NOT NULL
		AND last_error = $2`, oldStart, want); n != 1 {
		t.Errorf("the old row is not dead with last_error %q", want)
	}
	if n := count(t, e.super, "SELECT count(*) FROM outbox WHERE last_error LIKE '%' || $1 || '%'", recordMarker); n != 0 {
		t.Error("last_error carries the error message")
	}
	if strings.Contains(logs.String(), recordMarker) {
		t.Errorf("the log carries the error message:\n%s", logs.String())
	}
}

// TICK-94 item 3: a row whose target_key is not <kind>:<uuid> is dead
// without a Temporal call (written as the superuser: Enqueue cannot write it).
func TestInvalidTargetKeyRow(t *testing.T) {
	e := migrated(t)
	ctx := context.Background()
	fe, addr := startFrontend(t)
	tenant, id := uuid.New(), uuid.Must(uuid.NewV7())
	mustExec(t, e.super, "INSERT INTO tenants (id, state) VALUES ($1, 'active')", tenant)
	mustExec(t, e.super, `INSERT INTO outbox (id, tenant_id, kind, dedupe_key, target_key, target_seq)
		VALUES ($1, $2, 'approve', 'd', 'ticket.approval:alice@example.com', 1)`, id, tenant)
	d, _, _ := dispatcherOn(t, e.relay, addr, Options{})
	if n, err := d.Poll(ctx); err != nil || n != 1 {
		t.Fatalf("poll claimed %d, %v", n, err)
	}
	if n := count(t, e.super, "SELECT count(*) FROM outbox WHERE id = $1 AND dead_at IS NOT NULL AND last_error = $2",
		id, CauseInvalidTarget); n != 1 {
		t.Error("the row is not dead with the invalid-target cause")
	}
	if _, starts, signals := fe.counts(); starts+signals != 0 {
		t.Errorf("%d Temporal calls, want 0", starts+signals)
	}
}

// The environment of the relay subprocess (TestRelayProcess).
const (
	helperEnv     = "NAVA_RELAY_TEST_HELPER"
	helperDSNEnv  = "NAVA_RELAY_TEST_DSN"
	helperAddrEnv = "NAVA_RELAY_TEST_FRONTEND"
	helperPause   = "NAVA_RELAY_TEST_PAUSE"
	helperLease   = "NAVA_RELAY_TEST_LEASE"
)

// TestRelayProcess is the relay subprocess of TestCrashSeam and
// TestT33CrashBetweenClaimAndStart, not a test of its own: with
// NAVA_RELAY_TEST_HELPER=1 it runs a Dispatcher as nava_relay until it is
// killed, pausing every start at StdinPause when NAVA_RELAY_TEST_PAUSE=1,
// exactly as a relaytest build of nava-relay does with
// NAVA_RELAYTEST_PAUSE=1. Its lease is NAVA_RELAY_TEST_LEASE.
//
// Run directly by `go test` it always skips, by design (since T30b, TICK-85):
// relayProcessLease re-executes the test binary with -test.run=^TestRelayProcess$
// and the helper environment, and those parent tests run in the integration
// job (make ci-integration). A SKIP here in that job is expected.
func TestRelayProcess(t *testing.T) {
	if os.Getenv(helperEnv) != "1" {
		t.Skip("intended: helper subprocess started by TestCrashSeam and TestT33CrashBetweenClaimAndStart " +
			"with " + helperEnv + "=1; its behaviour is covered by those tests in the integration job (make ci-integration)")
	}
	pool, err := pgxpool.New(context.Background(), os.Getenv(helperDSNEnv))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	lease, err := time.ParseDuration(os.Getenv(helperLease))
	if err != nil {
		t.Fatal(err)
	}
	d, _, _ := dispatcherOn(t, pool, os.Getenv(helperAddrEnv), Options{Lease: lease})
	if os.Getenv(helperPause) == "1" {
		d.beforeStart = StdinPause(os.Stdin, os.Stderr)
	}
	fmt.Fprintln(os.Stderr, "relay subprocess running")
	d.Run(context.Background(), 20*time.Millisecond)
}

// relayDSN is the nava_relay connection string on the platform database.
func relayDSN(t *testing.T) string {
	t.Helper()
	u, err := url.Parse(os.Getenv("NAVA_TEST_PG_URL"))
	if err != nil {
		t.Fatal(err)
	}
	pw, _ := testPassword("nava_relay")
	u.User = url.UserPassword("nava_relay", pw)
	u.Path = "/nava"
	return u.String()
}

// relayProcess starts TestRelayProcess with a 2 s lease and returns it with a
// channel of its stderr lines.
func relayProcess(t *testing.T, addr string, pause bool) (*exec.Cmd, <-chan string) {
	t.Helper()
	return relayProcessLease(t, addr, pause, 2*time.Second)
}

// relayProcessLease is relayProcess with the given lease.
func relayProcessLease(t *testing.T, addr string, pause bool, lease time.Duration) (*exec.Cmd, <-chan string) {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestRelayProcess$", "-test.count=1")
	p := "0"
	if pause {
		p = "1"
	}
	cmd.Env = append(os.Environ(), helperEnv+"=1", helperDSNEnv+"="+relayDSN(t), helperAddrEnv+"="+addr, helperPause+"="+p,
		helperLease+"="+lease.String())
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = stdin.Close() })
	stderr, err := cmd.StderrPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill(); _ = cmd.Wait() })
	lines := make(chan string, 64)
	go func() {
		s := bufio.NewScanner(stderr)
		for s.Scan() {
			lines <- s.Text()
		}
		close(lines)
	}()
	return cmd, lines
}

func waitLine(t *testing.T, lines <-chan string, prefix string) string {
	t.Helper()
	timeout := time.After(30 * time.Second)
	for {
		select {
		case l, ok := <-lines:
			if !ok {
				t.Fatalf("relay subprocess exited before %q", prefix)
			}
			if strings.HasPrefix(l, prefix) {
				return l
			}
		case <-timeout:
			t.Fatalf("no %q from the relay subprocess within 30 s", prefix)
		}
	}
}

// AC: the crash seam stops the relay between claim and StartWorkflow; the
// relay is killed there (SIGKILL), and after a restart exactly one execution
// exists.
func TestCrashSeam(t *testing.T) {
	e := migrated(t)
	fe, addr := startFrontend(t)
	p := newProducer(t, e)
	start := p.start()

	first, lines := relayProcess(t, addr, true)
	if got := waitLine(t, lines, PauseMarker); got != PauseMarker+" "+start.String() {
		t.Fatalf("paused at %q, want the start %s", got, start)
	}
	if err := first.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = first.Wait()
	if _, starts, _ := fe.counts(); starts != 0 {
		t.Fatalf("%d starts before the kill, want 0", starts)
	}
	if n := count(t, e.super, `SELECT count(*) FROM outbox WHERE id = $1 AND attempts = 1
		AND dispatched_at IS NULL AND dead_at IS NULL AND available_at > now()`, start); n != 1 {
		t.Fatal("the killed relay's row is not claimed and pending")
	}

	second, lines := relayProcess(t, addr, false)
	waitLine(t, lines, "relay subprocess running")
	deadline := time.Now().Add(30 * time.Second)
	for count(t, e.super, "SELECT count(*) FROM outbox WHERE id = $1 AND dispatched_at IS NOT NULL", start) == 0 {
		if time.Now().After(deadline) {
			t.Fatal("the restarted relay did not dispatch the row within 30 s")
		}
		time.Sleep(20 * time.Millisecond)
	}
	_ = second.Process.Kill()
	if executions, starts, _ := fe.counts(); executions != 1 || starts != 1 {
		t.Errorf("%d executions from %d starts, want 1 from 1", executions, starts)
	}
	if _, ok := fe.execution(p.workflowID(start)); !ok {
		t.Errorf("no execution %s", p.workflowID(start))
	}
}

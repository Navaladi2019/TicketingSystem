//go:build integration

// T33 (PLAN-001, R9): the relay's worst cases against a real PostgreSQL as
// nava_relay and the real SDK client on the in-process frontend. Every case
// drives the dispatcher one Poll at a time and asserts in poll cycles and row
// counts, never in wall-clock time: a lease ends because the test moves
// available_at, not because a timer fired.

package relay

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.temporal.io/api/serviceerror"
)

// crashLease is the lease of the relays in the crash cases: long enough that
// it never ends by itself during the test, so only endLease ends it.
const crashLease = 5 * time.Minute

// stepper drives a Dispatcher one poll at a time and reads, per poll, what
// that poll added to outbox_relay_rows_scanned.
type stepper struct {
	t      *testing.T
	d      *Dispatcher
	reader *sdkmetric.ManualReader
	prev   metricdata.HistogramDataPoint[int64] // cumulative, after the last poll
	polls  int
}

// pollStats is one poll: the rows it claimed, and the scan observations it
// recorded (one per tenant with pending rows), how many of them fell in
// le="200", and their sum.
type pollStats struct {
	claimed      int
	observations uint64
	underCap     uint64
	scanned      int64
}

func newStepper(t *testing.T, e env, addr string, opts Options) *stepper {
	t.Helper()
	d, reader, _ := dispatcherOn(t, e.relay, addr, opts)
	return &stepper{t: t, d: d, reader: reader}
}

// poll runs one Poll and fails the test unless every scan observation it
// recorded is at most ScanCap (TICK-94 item 7: le="200" == _count, per
// cycle, never across cycles).
func (s *stepper) poll() pollStats {
	s.t.Helper()
	n, err := s.d.Poll(context.Background())
	if err != nil {
		s.t.Fatalf("poll %d: %v", s.polls+1, err)
	}
	s.polls++
	cur, ok := s.histogram()
	if !ok {
		return pollStats{claimed: n}
	}
	st := pollStats{
		claimed:      n,
		observations: cur.Count - s.prev.Count,
		underCap:     cumulativeAt(cur, ScanCap) - cumulativeAt(s.prev, ScanCap),
		scanned:      cur.Sum - s.prev.Sum,
	}
	s.prev = cur
	if st.underCap != st.observations {
		s.t.Errorf(`poll %d: le="200" grew by %d, _count by %d: a tenant scan passed the cap`,
			s.polls, st.underCap, st.observations)
	}
	return st
}

// histogram is the cumulative outbox_relay_rows_scanned point, if any poll
// has recorded one yet.
func (s *stepper) histogram() (metricdata.HistogramDataPoint[int64], bool) {
	var rm metricdata.ResourceMetrics
	if err := s.reader.Collect(context.Background(), &rm); err != nil {
		s.t.Fatal(err)
	}
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == "outbox_relay_rows_scanned" {
				return scannedHistogram(s.t, rm), true
			}
		}
	}
	return metricdata.HistogramDataPoint[int64]{}, false
}

// endLease ends the lease (or backoff) of the given rows, or of every
// pending row of tenant when ids is empty: the stepped stand-in for time
// passing.
func endLease(t *testing.T, e env, tenant uuid.UUID, ids ...uuid.UUID) {
	t.Helper()
	if len(ids) == 0 {
		mustExec(t, e.super, `UPDATE outbox SET available_at = now() - interval '1 second'
			WHERE tenant_id = $1 AND dispatched_at IS NULL AND dead_at IS NULL`, tenant)
		return
	}
	mustExec(t, e.super, `UPDATE outbox SET available_at = now() - interval '1 second'
		WHERE tenant_id = $1 AND id = ANY($2)`, tenant, ids)
}

// startTarget is one seeded target: its start row and its signal rows, in
// target_seq order.
type startTarget struct {
	start   uuid.UUID
	signals []uuid.UUID
}

func (s startTarget) workflowID(tenant uuid.UUID) string {
	return "t:" + tenant.String() + ":ticket.approval:" + s.start.String()
}

// seedTargets COPYs, as the superuser, targets dispatchable targets for
// tenant: each a start row (target_seq 1) and signals signal rows on the
// same target. A target's rows are consecutive in available_at, one
// microsecond apart from first, so a scan window holds whole targets in
// order. The rows are the shape outbox.Enqueue writes, with the default
// payload.
func seedTargets(tb testing.TB, conn *pgx.Conn, tenant uuid.UUID, targets, signals int, first time.Time) []startTarget {
	tb.Helper()
	mustExec(tb, conn, "INSERT INTO tenants (id, state) VALUES ($1, 'active') ON CONFLICT DO NOTHING", tenant)
	out := make([]startTarget, targets)
	rows := make([][]any, 0, targets*(1+signals))
	at := first
	for i := range out {
		head := uuid.Must(uuid.NewV7())
		key := "ticket.approval:" + head.String()
		out[i].start = head
		rows = append(rows, []any{head, tenant, "ticket.approval", "t33-" + head.String(), key, int64(1), at})
		for j := range signals {
			at = at.Add(time.Microsecond)
			id := uuid.Must(uuid.NewV7())
			out[i].signals = append(out[i].signals, id)
			rows = append(rows, []any{id, tenant, "approve", "t33-" + id.String(), key, int64(j + 2), at})
		}
		at = at.Add(time.Microsecond)
	}
	if _, err := conn.CopyFrom(context.Background(), pgx.Identifier{"outbox"},
		[]string{"id", "tenant_id", "kind", "dedupe_key", "target_key", "target_seq", "available_at"},
		pgx.CopyFromRows(rows)); err != nil {
		tb.Fatal(err)
	}
	return out
}

// T33 case 1: the relay is killed (SIGKILL) between its claim and the
// Temporal start, and a relay that starts after it leaves exactly one
// execution. Killed before StartWorkflow (the T30b crash seam) and killed
// after Temporal accepted the start but before MarkDispatched are both
// covered: the commit of the claim is the only thing either crash leaves.
func TestT33CrashBetweenClaimAndStart(t *testing.T) {
	t.Run("killed at the crash seam before StartWorkflow", func(t *testing.T) {
		e := migrated(t)
		fe, addr := startFrontend(t)
		p := newProducer(t, e)
		start := p.start()

		dead, lines := relayProcessLease(t, addr, true, crashLease)
		if got := waitLine(t, lines, PauseMarker); got != PauseMarker+" "+start.String() {
			t.Fatalf("paused at %q, want the start %s", got, start)
		}
		kill(t, dead)
		if _, starts, _ := fe.counts(); starts != 0 {
			t.Fatalf("%d starts before the kill, want 0", starts)
		}
		restartAfterCrash(t, e, fe, addr, p, start, 1)
	})

	t.Run("killed after Temporal accepted the start, before MarkDispatched", func(t *testing.T) {
		e := migrated(t)
		fe, addr := startFrontend(t)
		p := newProducer(t, e)
		start := p.start()
		// The frontend creates the execution and then holds the reply, so
		// the relay is killed with the start accepted and nothing marked.
		accepted, release := make(chan struct{}), make(chan struct{})
		var once sync.Once
		t.Cleanup(func() { closeOnce(release) })
		fe.setFail(func(method, workflowID string) error {
			if method != "start" {
				return nil
			}
			fe.addExecution(workflowID, false)
			once.Do(func() { close(accepted) })
			<-release
			return serviceerror.NewFailedPrecondition("relay gone")
		})

		dead, _ := relayProcessLease(t, addr, false, crashLease)
		select {
		case <-accepted:
		case <-time.After(30 * time.Second):
			t.Fatal("the relay subprocess did not reach StartWorkflow within 30 s")
		}
		kill(t, dead)
		closeOnce(release)
		fe.setFail(nil)
		restartAfterCrash(t, e, fe, addr, p, start, 2)
	})
}

// restartAfterCrash is the restarted relay: it must not redeliver start while
// the killed relay's lease holds, must redeliver it in the first poll after
// the lease ends, and must leave exactly one execution after wantStarts
// StartWorkflow calls in all.
func restartAfterCrash(t *testing.T, e env, fe *frontend, addr string, p *producer, start uuid.UUID, wantStarts int) {
	t.Helper()
	if n := count(t, e.super, `SELECT count(*) FROM outbox WHERE id = $1 AND attempts = 1
		AND dispatched_at IS NULL AND dead_at IS NULL AND available_at > now()`, start); n != 1 {
		t.Fatal("the killed relay's row is not claimed, leased and pending")
	}
	s := newStepper(t, e, addr, Options{Lease: crashLease})

	if st := s.poll(); st.claimed != 0 {
		t.Fatalf("poll 1 under the dead relay's lease claimed %d rows, want 0", st.claimed)
	}
	endLease(t, e, p.tenant, start)
	if st := s.poll(); st.claimed != 1 {
		t.Fatalf("poll 2, after the lease ended, claimed %d rows, want the start", st.claimed)
	}
	if n := count(t, e.super, `SELECT count(*) FROM outbox WHERE id = $1 AND attempts = 2
		AND dispatched_at IS NOT NULL`, start); n != 1 {
		t.Error("the start is not dispatched at attempt 2 by poll 2")
	}
	if st := s.poll(); st.claimed != 0 {
		t.Errorf("poll 3 claimed %d rows, want 0", st.claimed)
	}
	executions, starts, _ := fe.counts()
	if executions != 1 || starts != wantStarts {
		t.Errorf("%d executions from %d starts, want 1 from %d", executions, starts, wantStarts)
	}
	if _, ok := fe.execution(p.workflowID(start)); !ok {
		t.Errorf("no execution %s", p.workflowID(start))
	}
	t.Logf("T33 crash: 1 execution, %d StartWorkflow calls, redelivered in poll 2 of %d", starts, s.polls)
}

// kill SIGKILLs the relay subprocess and reaps it.
func kill(t *testing.T, cmd *exec.Cmd) {
	t.Helper()
	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = cmd.Wait()
}

func closeOnce(ch chan struct{}) {
	select {
	case <-ch:
	default:
		close(ch)
	}
}

// T33 case 2: tenant A has 50,000 pending rows; tenant B's one row is
// dispatched within 2 poll cycles, and A's scan stays at the cap in every
// cycle.
func TestT33FloodDoesNotStarveOtherTenant(t *testing.T) {
	e := migrated(t)
	fe, addr := startFrontend(t)
	const flood = 50_000
	a := uuid.New()
	seedTargets(t, e.super, a, flood, 0, time.Now().Add(-time.Hour))
	b := newProducer(t, e)
	bStart := b.start() // written after the flood, through Enqueue
	s := newStepper(t, e, addr, Options{})

	const bound = 2
	at := 0
	for at == 0 && s.polls < bound {
		began := time.Now()
		st := s.poll()
		if st.observations != 2 {
			t.Errorf("poll %d recorded %d scan observations, want 2 (A and B)", s.polls, st.observations)
		}
		t.Logf("T33 flood: poll %d claimed %d rows, scanned %d (A capped at %d), took %v",
			s.polls, st.claimed, st.scanned, ScanCap, time.Since(began).Round(time.Millisecond))
		if count(t, e.super, "SELECT count(*) FROM outbox WHERE id = $1 AND dispatched_at IS NOT NULL", bStart) == 1 {
			at = s.polls
		}
	}
	if at == 0 {
		t.Fatalf("B's row is not dispatched after %d polls", bound)
	}
	if _, ok := fe.execution(b.workflowID(bStart)); !ok {
		t.Errorf("no execution for B's start")
	}
	if n := count(t, e.super, "SELECT count(*) FROM outbox WHERE tenant_id = $1 AND attempts > 0", a); n != PerTenant*s.polls {
		t.Errorf("A had %d rows claimed in %d polls, want %d per poll", n, s.polls, PerTenant)
	}
	t.Logf("T33 flood: B dispatched in poll %d of at most %d with %d rows pending for A", at, bound, flood)
}

// AR-027, which T33's flood covers: a leased head that is not backing off
// leaves its available successors in its tenant's 200-row window, so the
// tenant's later targets wait for the lease. Other tenants do not wait and no
// scan passes the cap. Once the lease ends (the relay holding it died), the
// tenant must drain in order: the head's available_at is then later than its
// 250 successors', and it must still be claimed.
func TestT33LeasedHeadOnDeepTarget(t *testing.T) {
	e := migrated(t)
	fe, addr := startFrontend(t)
	a := uuid.New()
	const deep = 250 // successors on target X, all older than target Y
	first := time.Now().Add(-time.Hour)
	x := seedTargets(t, e.super, a, 1, deep, first)[0]
	y := seedTargets(t, e.super, a, 1, 0, first.Add(time.Second))[0]

	// Another relay claims X's head and stalls mid-dispatch.
	other := newProbe(t, e.relay, Options{Lease: crashLease})
	if cs, err := other.Claim(context.Background()); err != nil || len(cs) != 1 || cs[0].ID != x.start {
		t.Fatalf("the other relay claimed %+v, %v; want X's head", cs, err)
	}
	b := newProducer(t, e)
	bStart := b.start()
	s := newStepper(t, e, addr, Options{Lease: crashLease})

	const held = 3
	for range held {
		st := s.poll()
		if st.observations != 2 && s.polls == 1 {
			t.Errorf("poll 1 recorded %d scan observations, want 2 (A and B)", st.observations)
		}
		if st.scanned < ScanCap {
			t.Errorf("poll %d scanned %d rows, want A's window full at %d", s.polls, st.scanned, ScanCap)
		}
	}
	if count(t, e.super, "SELECT count(*) FROM outbox WHERE id = $1 AND dispatched_at IS NOT NULL", bStart) != 1 {
		t.Error("B's row waited for A's leased head")
	}
	if n := count(t, e.super, "SELECT count(*) FROM outbox WHERE tenant_id = $1 AND attempts > 0", a); n != 1 {
		t.Errorf("%d of A's rows claimed while X's head is leased, want only the head", n)
	}

	// The other relay dies; its lease ends. Y enters A's window once at
	// most ScanCap-1 older rows are pending, and X drains one row a poll.
	endLease(t, e, a, x.start)
	s.poll()
	if count(t, e.super, "SELECT count(*) FROM outbox WHERE id = $1 AND dispatched_at IS NOT NULL", x.start) != 1 {
		t.Fatalf("X's head was not dispatched in the first poll after its lease ended: its available_at is the "+
			"lease end, so its %d older successors fill A's %d-row window and A is stalled for good", deep, ScanCap)
	}
	maxPolls := deep + 1 - (ScanCap - 1) + 1
	polls := 1
	for count(t, e.super, "SELECT count(*) FROM outbox WHERE id = $1 AND dispatched_at IS NOT NULL", y.start) == 0 {
		if polls == maxPolls {
			t.Fatalf("Y not dispatched within %d polls after the lease ended", maxPolls)
		}
		s.poll()
		polls++
	}
	t.Logf("AR-027: A held for the lease (%d polls), then Y dispatched in poll %d after it (bound %d); B never waited",
		held, polls, maxPolls)
	if _, ok := fe.execution(y.workflowID(a)); !ok {
		t.Error("no execution for Y")
	}
}

// T33 case 3: every head of a tenant backs off. Each poll scans at most
// ScanCap rows of the tenant, no successor is claimed or signalled before its
// head, and the tenant goes quiet in exactly heads/PerTenant polls. When
// Temporal recovers, every target drains in order.
func TestT33EveryHeadBackingOff(t *testing.T) {
	e := migrated(t)
	fe, addr := startFrontend(t)
	a := uuid.New()
	const heads, signals = 300, 2
	ts := seedTargets(t, e.super, a, heads, signals, time.Now().Add(-time.Hour))
	b := newProducer(t, e)
	bStart := b.start()
	fe.setFail(func(method, workflowID string) error {
		if strings.HasPrefix(workflowID, "t:"+a.String()+":") {
			// Not Unavailable: the SDK retries that itself until the
			// dispatch deadline (half a lease) before the relay sees it.
			return serviceerror.NewFailedPrecondition("temporal refuses A")
		}
		return nil
	})
	s := newStepper(t, e, addr, Options{BackoffBase: time.Hour, BackoffMax: time.Hour})

	successorsTouched := func() int {
		return count(t, e.super, `SELECT count(*) FROM outbox
			WHERE tenant_id = $1 AND target_seq > 1 AND attempts > 0`, a)
	}
	const quietAt = heads/PerTenant + 1
	for {
		st := s.poll()
		if st.claimed == 0 {
			if st.observations != 1 || st.scanned != 0 {
				t.Errorf("quiet poll %d: %d observations scanning %d rows, want A alone scanning 0",
					s.polls, st.observations, st.scanned)
			}
			break
		}
		if n := successorsTouched(); n != 0 {
			t.Fatalf("poll %d: %d successors claimed while their heads back off", s.polls, n)
		}
		if s.polls == quietAt {
			t.Fatalf("A still claiming at poll %d, want quiet at poll %d", s.polls, quietAt)
		}
	}
	if s.polls != quietAt {
		t.Errorf("A went quiet at poll %d, want %d (%d heads, %d a poll)", s.polls, quietAt, heads, PerTenant)
	}
	if _, _, sigs := fe.counts(); sigs != 0 {
		t.Errorf("%d signals sent while every head backed off, want 0", sigs)
	}
	if n := count(t, e.super, `SELECT count(*) FROM outbox WHERE tenant_id = $1 AND target_seq = 1
		AND attempts = 1 AND available_at > now() + interval '59 minutes' AND dead_at IS NULL`, a); n != heads {
		t.Errorf("%d heads backed off by their delay, want %d", n, heads)
	}
	if n := count(t, e.super, `SELECT count(*) FROM outbox o JOIN outbox h
		ON h.tenant_id = o.tenant_id AND h.target_key = o.target_key AND h.target_seq = 1
		WHERE o.tenant_id = $1 AND o.target_seq > 1 AND o.available_at < h.available_at`, a); n != 0 {
		t.Errorf("%d successors available before their backing-off head", n)
	}
	if count(t, e.super, "SELECT count(*) FROM outbox WHERE id = $1 AND dispatched_at IS NOT NULL", bStart) != 1 {
		t.Error("B's row was not dispatched while A backed off")
	}
	t.Logf("T33 backoff: %d heads backed off in %d polls, then a poll scanned 0 rows; 0 successors claimed",
		heads, s.polls-1)

	// Temporal recovers and the backoff ends: every target drains, signals
	// after their start, in order.
	fe.setFail(nil)
	endLease(t, e, a)
	recovered := s.polls
	maxPolls := heads*(1+signals)/PerTenant + 1
	for count(t, e.super, "SELECT count(*) FROM outbox WHERE tenant_id = $1 AND dispatched_at IS NULL", a) > 0 {
		if s.polls-recovered == maxPolls {
			t.Fatalf("A not drained within %d polls after recovery", maxPolls)
		}
		s.poll()
	}
	for _, tg := range ts {
		ex, ok := fe.execution(tg.workflowID(a))
		if !ok {
			t.Errorf("target %s never started", tg.start)
			continue
		}
		if fmt.Sprint(ex.signals) != fmt.Sprint(tg.signals) {
			t.Errorf("target %s signals %v, want %v", tg.start, ex.signals, tg.signals)
		}
	}
	if n := count(t, e.super, "SELECT count(*) FROM outbox WHERE dead_at IS NOT NULL"); n != 0 {
		t.Errorf("%d rows dead, want 0", n)
	}
	t.Logf("T33 backoff: after recovery A drained %d rows in %d polls (bound %d)",
		heads*(1+signals), s.polls-recovered, maxPolls)
}

// BenchmarkClaim is the claim latency against ADR-001's baseline (5.4 ms,
// spike round 2 C1): 200 tenants pending, one flood tenant with 45,000 rows
// on 1,000 targets 45 deep and 199 tenants sharing 5,000 rows one per target.
// The spike's 3,000,000 dispatched rows are left out: the partial pending
// indexes never hold them. Each iteration claims in a transaction it rolls
// back, so every claim sees the same state. Nightly and report-only
// (make bench-relay), never a gate.
func BenchmarkClaim(b *testing.B) {
	e := migrated(b)
	first := time.Now().Add(-time.Hour)
	seedTargets(b, e.super, uuid.New(), 1_000, 44, first)
	for range 199 {
		seedTargets(b, e.super, uuid.New(), 5_000/199, 0, first)
	}
	mustExec(b, e.super, "ANALYZE outbox")
	ctx := context.Background()
	for b.Loop() {
		tx, err := e.relay.Begin(ctx)
		if err != nil {
			b.Fatal(err)
		}
		r, err := New(tx, sdkmetric.NewMeterProvider(), slog.New(slog.DiscardHandler), Options{})
		if err != nil {
			b.Fatal(err)
		}
		if _, err := r.Claim(ctx); err != nil {
			b.Fatal(err)
		}
		if err := tx.Rollback(ctx); err != nil {
			b.Fatal(err)
		}
	}
	b.ReportMetric(float64(b.Elapsed().Microseconds())/1000/float64(b.N), "ms/claim")
}

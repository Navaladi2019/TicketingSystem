//go:build integration && perf

// TICK-700: open-loop create rate mode for TestRecordsACLPerf800k,
// informational only (no target, no verdict). With NAVA_PERF_RATE set
// (requests per second, a comma-separated list such as "100" or "100,200"),
// the selected create scenarios run open loop instead of with closed-loop
// clients: creates are sent on a fixed schedule, one every 1/rate, whether or
// not earlier ones have answered. Each request's latency runs from its
// scheduled send time, so queueing in the harness, the pool and the counter
// row is counted (no coordinated omission).
//
// Env: NAVA_PERF_RATE (off), NAVA_PERF_WARMUP (10 s at the same rate before
// the measured window), NAVA_PERF_SECONDS (the measured window, set it to 60
// or more). A gapless (unscored) scenario runs only the first rate.

package server_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	dto "github.com/prometheus/client_model/go"
)

// perfMaxInFlight bounds the open-loop backlog: an arrival that finds this
// many requests still in flight is not sent and is counted as dropped. A
// non-zero count means the system did not keep up with the offered rate.
const perfMaxInFlight = 2000

// perfOpenLabel is the verdict column of an open-loop row.
const perfOpenLabel = "open loop, informational (TICK-700)"

// perfAcq is one connection acquire a request waited for, and on which pool.
type perfAcq struct {
	pool *pgxpool.Pool
	dur  time.Duration
}

type perfAcqKey struct{}

// TraceAcquireStart implements pgxpool.AcquireTracer: the wait for a pooled
// connection, recorded on the request's trace.
func (perfTracer) TraceAcquireStart(ctx context.Context, _ *pgxpool.Pool, _ pgxpool.TraceAcquireStartData) context.Context {
	if ctx.Value(perfTraceKey{}) == nil {
		return ctx
	}
	return context.WithValue(ctx, perfAcqKey{}, time.Now())
}

// TraceAcquireEnd implements pgxpool.AcquireTracer.
func (perfTracer) TraceAcquireEnd(ctx context.Context, pool *pgxpool.Pool, _ pgxpool.TraceAcquireEndData) {
	tr, _ := ctx.Value(perfTraceKey{}).(*perfTrace)
	at, ok := ctx.Value(perfAcqKey{}).(time.Time)
	if tr == nil || !ok {
		return
	}
	tr.mu.Lock()
	tr.acqs = append(tr.acqs, perfAcq{pool, time.Since(at)})
	tr.mu.Unlock()
}

// perfOpen is what an open-loop row measured beyond the closed-loop fields.
type perfOpen struct {
	rate            int
	warm, window    time.Duration
	sent, dropped   int64 // arrivals in the window: sent, and not sent (backlog cap)
	completedInWin  int64 // successful answers that arrived inside the window
	inFlight        []int // in flight at each measured arrival, itself included
	maxInFlight     int64
	acquire         []time.Duration // API pool acquire wait, per request (summed)
	lateMax         time.Duration   // the scheduler's worst lag behind an arrival
	drain           time.Duration   // from the end of the schedule to the last answer
	poolMax         int32
	poolEmptyAcq    int64 // acquires that found no idle connection (pool.Stat delta)
	poolAcquireMean time.Duration
}

// perfRates parses NAVA_PERF_RATE; nil means closed-loop mode.
func perfRates(t *testing.T) []int {
	t.Helper()
	v := strings.TrimSpace(os.Getenv("NAVA_PERF_RATE"))
	if v == "" {
		return nil
	}
	var out []int
	for _, f := range strings.Split(v, ",") {
		n, err := strconv.Atoi(strings.TrimSpace(f))
		if err != nil || n <= 0 {
			t.Fatalf("NAVA_PERF_RATE=%q: want positive integers separated by commas", v)
		}
		out = append(out, n)
	}
	return out
}

// openLoop runs every selected create scenario at each rate (a gapless,
// unscored scenario at the first rate only) and returns one result per row.
// Before each row it isolates, quiesces and probes the host exactly like a
// closed-loop create attempt; the probes are reported, never gated.
func (e *perfEnv) openLoop(t *testing.T, rates []int, warm, window time.Duration) []*perfResult {
	t.Helper()
	var results []*perfResult
	for _, sc := range e.scenarios(t) {
		if sc.kind != "create" {
			t.Fatalf("NAVA_PERF_RATE: %q is not a create scenario; select creates with NAVA_PERF_ONLY", sc.name)
		}
		if sc.before != nil {
			sc.before()
		}
		rs := rates
		if sc.unscored != "" {
			rs = rates[:1]
		}
		for _, rate := range rs {
			row := sc
			row.name = fmt.Sprintf("%s, open loop %d/s", sc.name, rate)
			row.unscored = perfOpenLabel
			var st perfSetup
			st.isolationWait = isolate(t)
			st.quiesceWait = e.quiesce(t)
			probe := e.hostProbe(t, &st)
			storage := e.storageProbe(t)
			stop := make(chan struct{})
			watch := watchContainers(t.Context(), stop)
			r := e.runOpen(t, row, rate, warm, window)
			close(stop)
			r.intruders = <-watch
			r.attempt, r.probe, r.setup, r.storage = 1, probe, st, storage
			o := r.open
			t.Logf("OPEN %-58s offered=%d/s achieved=%.1f/s n=%d p50=%.2f p95=%.2f p99=%.2f max=%.2f ms in-flight max=%d p95=%d dropped=%d errors=%v acquire p95=%.2f ms",
				row.name, rate, o.achieved(), len(r.lat), pct(r.lat, .5), pct(r.lat, .95), pct(r.lat, .99), pct(r.lat, 1),
				o.maxInFlight, pctInt(o.inFlight, .95), o.dropped, r.errCodes, pct(o.acquire, .95))
			results = append(results, r)
		}
	}
	return results
}

// runOpen sends sc's requests open loop at rate per second: warm at the
// start is sent and not measured, then window is measured. Latency runs from
// each request's scheduled send time. Failed requests are counted by status
// and problem code, never fatal: they are a measurement here.
func (e *perfEnv) runOpen(t *testing.T, sc perfScenario, rate int, warm, window time.Duration) *perfResult {
	t.Helper()
	res := &perfResult{sc: sc, stmtDur: map[string][]time.Duration{}, errCodes: map[string]int64{}}
	o := &perfOpen{rate: rate, warm: warm, window: window, poolMax: e.pool.Config().MaxConns}
	res.open = o
	for i, r := range sc.reqs {
		if i >= 50 {
			break
		}
		if rec, ok := e.send(t.Context(), sc, r); !ok {
			t.Fatalf("%s: warm-up %s as %s = %d: %s", sc.name, r.path, r.token, rec.Code, rec.Body)
		}
	}
	var (
		mu       sync.Mutex
		wg       sync.WaitGroup
		inFlight atomic.Int64
		interval = time.Second / time.Duration(rate)
		start    = time.Now().Add(50 * time.Millisecond)
		winStart = start.Add(warm)
		end      = winStart.Add(window)
		lastDone atomic.Int64 // UnixNano of the latest answer
	)
	var (
		statBefore  *pgxpool.Stat
		allocBefore map[string]*dto.MetricFamily
	)
	for i := 0; ; i++ {
		at := start.Add(time.Duration(i) * interval)
		if !at.Before(end) {
			break
		}
		measured := !at.Before(winStart)
		if measured && statBefore == nil {
			// The window starts: snapshot the pool and the allocator series.
			statBefore = e.pool.Stat()
			allocBefore = e.allocSnapshot(t)
		}
		if d := time.Until(at); d > 0 {
			time.Sleep(d)
		} else if lag := -d; lag > o.lateMax {
			o.lateMax = lag
		}
		n := inFlight.Add(1)
		if n > perfMaxInFlight {
			inFlight.Add(-1)
			if measured {
				o.dropped++
			}
			continue
		}
		if measured {
			o.sent++
			o.inFlight = append(o.inFlight, int(n))
			if n > o.maxInFlight {
				o.maxInFlight = n
			}
		}
		r := sc.reqs[i%len(sc.reqs)]
		wg.Add(1)
		go func(at time.Time, r perfReq, measured bool) {
			defer wg.Done()
			defer inFlight.Add(-1)
			tr := newPerfTrace()
			ctx := context.WithValue(t.Context(), perfTraceKey{}, tr)
			rec, ok := e.send(ctx, sc, r)
			done := time.Now()
			el := done.Sub(at)
			for {
				prev := lastDone.Load()
				if done.UnixNano() <= prev || lastDone.CompareAndSwap(prev, done.UnixNano()) {
					break
				}
			}
			if !measured {
				return
			}
			// The allocator pool acquires outside the request's context, so
			// only the API pool shows here; the allocator's acquire wait is
			// nava_numbering_alloc_acquire, in the number allocation table.
			var acq time.Duration
			tr.mu.Lock()
			for _, a := range tr.acqs {
				if a.pool == e.pool {
					acq += a.dur
				}
			}
			stmts := tr.stmts
			tr.mu.Unlock()
			mu.Lock()
			defer mu.Unlock()
			o.acquire = append(o.acquire, acq)
			if !ok {
				res.errs++
				var pb struct {
					Code string `json:"code"`
				}
				_ = json.Unmarshal(rec.Body.Bytes(), &pb)
				res.errCodes[fmt.Sprintf("%d %s", rec.Code, pb.Code)]++
				return
			}
			if !done.After(end) {
				o.completedInWin++
			}
			res.lat = append(res.lat, el)
			if h, ok := counterHold(stmts); ok {
				res.holds = append(res.holds, h)
			}
			if c, ok := lastCommit(stmts); ok {
				res.commits = append(res.commits, c)
			}
			for _, s := range stmts {
				k := normSQL(s.SQL)
				res.stmtDur[k] = append(res.stmtDur[k], s.Dur)
			}
		}(at, r, measured)
	}
	wg.Wait()
	if last := time.Unix(0, lastDone.Load()); last.After(end) {
		o.drain = last.Sub(end)
	}
	if statBefore != nil {
		after := e.pool.Stat()
		o.poolEmptyAcq = after.EmptyAcquireCount() - statBefore.EmptyAcquireCount()
		if n := after.AcquireCount() - statBefore.AcquireCount(); n > 0 {
			o.poolAcquireMean = (after.AcquireDuration() - statBefore.AcquireDuration()) / time.Duration(n)
		}
		res.alloc = allocDelta(allocBefore, e.allocSnapshot(t))
	}
	e.capture(t, res)
	return res
}

// achieved is the successful creates answered inside the window, per second.
func (o *perfOpen) achieved() float64 {
	return float64(o.completedInWin) / o.window.Seconds()
}

// pctInt is the nearest-rank percentile of ints.
func pctInt(v []int, p float64) int {
	d := make([]time.Duration, len(v))
	for i, x := range v {
		d[i] = time.Duration(x) * time.Millisecond
	}
	return int(pct(d, p))
}

// reportOpen writes the open-loop table and its method note.
func reportOpen(w func(string, ...any), results []*perfResult) {
	var open []*perfResult
	for _, r := range results {
		if r.open != nil {
			open = append(open, r)
		}
	}
	if len(open) == 0 {
		return
	}
	o0 := open[0].open
	w("## Open-loop create (TICK-700, informational: no target, no verdict)\n\n")
	w("Creates are sent on a fixed schedule at the offered rate, whether or not earlier ones have answered, after a %s warm-up at the same rate; the window below is %s. Latency runs from each request's **scheduled** send time, so queueing counts (no coordinated omission). Achieved is the successful creates answered inside the window, per second. In flight is sampled at each arrival, itself included; an arrival that finds %d in flight is dropped. API pool: %d connections (NAVA_PERF_CLIENTS + 2); allocator pool: 4. In-process `httptest`, no PgBouncer, no network. Host probes are reported, not gated. The allocator pool's acquire wait is `nava_numbering_alloc_acquire`, in the number allocation table.\n\n",
		o0.warm, o0.window, perfMaxInFlight, o0.poolMax)
	w("| Row | Offered /s | Achieved /s | Sent | Dropped | OK | p50 | p95 | p99 | max | In flight max / p95 | 409 | 503 retry_later | Other errors | Counter wait p50 / p95 | Commit p50 | API pool acquire p50 / p95 / max | Drain after window | Host probe p50 | Storage probe p50 | Other containers |\n")
	w("| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |\n")
	for _, r := range open {
		o := r.open
		var c409, other int64
		var others []string
		for k, n := range r.errCodes {
			switch {
			case strings.HasPrefix(k, "409"):
				c409 += n
			case k == "503 retry_later":
			default:
				other += n
				others = append(others, fmt.Sprintf("%s: %d", k, n))
			}
		}
		otherCell := strconv.FormatInt(other, 10)
		if len(others) > 0 {
			otherCell += " (" + strings.Join(others, ", ") + ")"
		}
		waits := make([]time.Duration, 0, len(r.holds))
		for _, h := range r.holds {
			waits = append(waits, h.wait)
		}
		intr := "none"
		if len(r.intruders) > 0 {
			intr = strings.Join(r.intruders, " ")
		}
		w("| %s | %d | %.1f | %d | %d | %d | %.2f | %.2f | %.2f | %.2f | %d / %d | %d | %d | %s | %.2f / %.2f | %.2f | %.2f / %.2f / %.2f | %s | %.2f | %.2f | %s |\n",
			r.sc.name, o.rate, o.achieved(), o.sent, o.dropped, len(r.lat),
			pct(r.lat, .5), pct(r.lat, .95), pct(r.lat, .99), pct(r.lat, 1),
			o.maxInFlight, pctInt(o.inFlight, .95), c409, r.retryLater(), otherCell,
			pct(waits, .5), pct(waits, .95), r.commitP50(),
			pct(o.acquire, .5), pct(o.acquire, .95), pct(o.acquire, 1),
			o.drain.Round(time.Millisecond), r.probe, r.storage, intr)
	}
	w("\nAPI pool, from `pgxpool.Stat` over the window: acquires that found no idle connection, and the mean acquire wait.\n\n| Row | Empty acquires | Mean acquire | Scheduler worst lag |\n| --- | --- | --- | --- |\n")
	for _, r := range open {
		o := r.open
		w("| %s | %d | %.2f ms | %.2f ms |\n", r.sc.name, o.poolEmptyAcq, float64(o.poolAcquireMean.Microseconds())/1000, float64(o.lateMax.Microseconds())/1000)
	}
	w("\n")
}

//go:build integration && perf

// ADR-006-E2 E2-4 (3e) (TICK-1098, PLAN-007-E2 T-E2-2): the serial-count
// check, run inside TestA2Gate (NAVA_PERF_SPIKE_CELLS=serial) so that it uses
// the same seed, the same host gate and one test binary.
//
//   - Arm A: the shipped handler (CountBudget 170 ms, the count's savepoint
//     Exec sets max_parallel_workers_per_gather = 0).
//   - Arm B: e.handlerWith(t, server.WithCountArm(140 ms, true)), the fallback
//     as it would ship: 140 ms, server settings.
//
// Both arms use e.pool, traced by perfTracer only: no gateCountTracer, so no
// GUC is forced. Each cell runs its own ABBA: blocks of NAVA_PERF_LIST_SECONDS
// (15) at NAVA_PERF_CLIENTS (8), with the (2a) host gate (floor and control)
// before and after each block. Adjacent blocks share the gate reading between
// them. A block with a failed gate is rerun up to twice.

package server_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// e3BandP95MS is (3e)'s whole-request p95 limit on band (ii), arm A.
const e3BandP95MS = 350.0

// e3Arm is one side of the comparison.
type e3Arm struct {
	key, desc string
	h         http.Handler
}

// e3Sample is one request of a block.
type e3Sample struct {
	code             int
	state            string
	phase, countStmt float64 // ms, SAVEPOINT start to ROLLBACK TO end; -1 if no count ran
	latency          float64 // ms, client side
	// at and body are kept for a non-200 answer only.
	at   time.Time
	body string
}

// e3Block is one accepted or rejected block of a cell.
type e3Block struct {
	arm, label    string
	before, after gateHost
	samples       []e3Sample
	peakWorkers   int64
	workerSamples int
	// tempFiles and tempBytes are pg_stat_database's rise over this block
	// (its warm-up included, the gates not): spill context only.
	tempFiles, tempBytes int64
}

// e3Settle outlasts the 10 s after which an idle backend flushes its
// pg_stat_database counters.
const e3Settle = 11 * time.Second

func (b e3Block) ok() bool { return b.before.ok() && b.after.ok() }

// e3Serial runs (3e). host is TestA2Gate's (2a) gate; w its report writer.
type e3Serial struct {
	e       *fiEnv
	tenant  uuid.UUID
	clients int
	d       time.Duration
	arms    map[string]e3Arm
	host    func(label string) gateHost
	w       func(format string, args ...any)
	// last is the gate reading right after the previous block, the next
	// block's "before"; nil at the start of a cell.
	last *gateHost
	// cells in run order, and their results.
	cells   []*e3CellResult
	plans   *[]string
	csv     strings.Builder
	onPlans func()
	// smoke (NAVA_E3_SMOKE=1): every step runs, the gates are recorded and
	// ignored, and there is no verdict.
	smoke bool
}

// e3SmokeLine takes the verdict words out of a smoke run's report line.
func e3SmokeLine(l string) string {
	return strings.NewReplacer("INCONCLUSIVE", "[smoke]", "PASS", "[smoke]", "FAIL", "[smoke]").Replace(l)
}

type e3CellResult struct {
	name    string
	blocks  map[string][]e3Block // accepted, by arm
	tried   []e3Block            // every attempt, in order
	hostBad string               // non-empty: a block failed its gates three times
	band    bool                 // the band (ii) cell: arm A's p95 is gated
	plan    map[string]e3Plan    // a50 only
}

type e3Plan struct {
	savepoint string
	pre       []string
	count     perfStmt
	text      string
	gather    bool
	launched  bool
}

// cell runs one cell's ABBA on reqs (round robin, one request per client
// turn).
func (s *e3Serial) cell(t *testing.T, name string, reqs []perfReq, band bool) *e3CellResult {
	t.Helper()
	r := &e3CellResult{name: name, blocks: map[string][]e3Block{}, band: band, plan: map[string]e3Plan{}}
	s.cells = append(s.cells, r)
	s.last = nil
	s.w("")
	s.w("### (3e) cell %s: ABBA", name)
	s.w("")
	for i, arm := range []string{"A", "B", "B", "A"} {
		var got *e3Block
		for try := 0; try < 3; try++ {
			label := fmt.Sprintf("%s block %d (arm %s), try %d", name, i+1, arm, try+1)
			var before gateHost
			if s.last != nil && (s.smoke || s.last.ok()) {
				before = *s.last
				s.w("- %s: before-gate = the previous block's after-gate (%s)", label, before)
			} else {
				before = s.host(label + ": before")
			}
			b := s.block(t, arm, label, reqs)
			b.before = before
			b.after = s.host(label + ": after")
			s.last = &b.after
			r.tried = append(r.tried, b)
			for _, x := range b.samples {
				s.csv.WriteString(e3CSVRow(name, i+1, try+1, arm, b.ok(), x))
			}
			s.w("- %s: n %d, unknown %d, peak parallel workers %d (%d samples), temp_files +%d (%d bytes, spill context only); gates %s", label, len(b.samples), unknownOf(b.samples), b.peakWorkers, b.workerSamples, b.tempFiles, b.tempBytes, okWord(b.ok()))
			if s.smoke || b.ok() {
				got = &b
				break
			}
		}
		if got == nil {
			r.hostBad = fmt.Sprintf("block %d (arm %s) failed its host gates on 3 tries", i+1, arm)
			s.w("- **%s: %s; the cell is inconclusive for host reasons**", name, r.hostBad)
			return r
		}
		r.blocks[arm] = append(r.blocks[arm], *got)
	}
	return r
}

// block runs one arm for s.d at s.clients and samples pg_stat_activity's
// parallel worker backends every 50 ms on its own connection.
func (s *e3Serial) block(t *testing.T, arm, label string, reqs []perfReq) e3Block {
	t.Helper()
	h := s.arms[arm].h
	time.Sleep(e3Settle)
	t0 := gateTemp(t, s.e.super)
	for _, r := range reqs[:min(len(reqs), 2*s.clients)] {
		_ = s.one(t, h, r)
	}
	b := e3Block{arm: arm, label: label}
	conn, err := pgx.ConnectConfig(t.Context(), s.e.super.Config())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close(context.Background()) }()
	stop := make(chan struct{})
	var sw sync.WaitGroup
	sw.Add(1)
	go func() {
		defer sw.Done()
		tk := time.NewTicker(50 * time.Millisecond)
		defer tk.Stop()
		for {
			select {
			case <-stop:
				return
			case <-tk.C:
				var n int64
				if err := conn.QueryRow(t.Context(), `SELECT count(*) FROM pg_stat_activity WHERE backend_type = 'parallel worker'`).Scan(&n); err == nil {
					b.workerSamples++
					b.peakWorkers = max(b.peakWorkers, n)
				}
			}
		}
	}()
	var (
		mu       sync.Mutex
		next     atomic.Int64
		wg       sync.WaitGroup
		deadline = time.Now().Add(s.d)
	)
	for range s.clients {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var mine []e3Sample
			for time.Now().Before(deadline) {
				mine = append(mine, s.one(t, h, reqs[int(next.Add(1)-1)%len(reqs)]))
			}
			mu.Lock()
			b.samples = append(b.samples, mine...)
			mu.Unlock()
		}()
	}
	wg.Wait()
	close(stop)
	sw.Wait()
	time.Sleep(e3Settle)
	t1 := gateTemp(t, s.e.super)
	b.tempFiles, b.tempBytes = t1.files-t0.files, t1.bytes-t0.bytes
	return b
}

// one sends r through h with a per-request trace and reads its count phase
// as TestA2Gate's spike does: SAVEPOINT start to ROLLBACK TO end.
func (s *e3Serial) one(t *testing.T, h http.Handler, r perfReq) e3Sample {
	tr := newPerfTrace()
	t0 := time.Now()
	rec := serveOn(context.WithValue(t.Context(), perfTraceKey{}, tr), h, http.MethodGet, r.path, r.token, "", "")
	x := e3Sample{code: rec.Code, latency: fiMS(time.Since(t0)), phase: -1, countStmt: -1}
	var p listPage
	if json.Unmarshal(rec.Body.Bytes(), &p) == nil {
		x.state = p.State
	}
	if rec.Code != http.StatusOK {
		x.at, x.body = t0, rec.Body.String()
	}
	sp := -1
	for i, st := range tr.stmts {
		sql := strings.TrimSpace(st.SQL)
		switch {
		case strings.HasPrefix(sql, "SAVEPOINT list_count"):
			sp = i
		case strings.HasPrefix(sql, "ROLLBACK TO SAVEPOINT list_count") && sp >= 0:
			x.phase = fiMS(st.Start + st.Dur - tr.stmts[sp].Start)
			if i > sp+1 {
				x.countStmt = fiMS(tr.stmts[sp+1].Dur)
			}
		}
	}
	return x
}

// planCheck is the scored plan check: any error ends the test.
func (s *e3Serial) planCheck(t *testing.T, r *e3CellResult, token, q string) {
	t.Helper()
	for _, arm := range []string{"A", "B"} {
		p, err := s.plan(t.Context(), arm, token, q)
		if err != nil {
			t.Fatalf("(3e) plan check arm %s: %v", arm, err)
		}
		r.plan[arm] = p
		s.keepPlan(r.name, arm, p)
	}
}

// planReported records the same plans outside the verdict. An error goes into
// plans.md as the reason and the run goes on.
func (s *e3Serial) planReported(t *testing.T, name, token, q string) {
	t.Helper()
	for _, arm := range []string{"A", "B"} {
		ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
		p, err := s.plan(ctx, arm, token, q)
		cancel()
		if err != nil {
			*s.plans = append(*s.plans, fmt.Sprintf("### (3e) %s, arm %s: not recorded\n\n%v", name, arm, err))
			s.onPlans()
			s.w("- plan (reported only), %s, arm %s: not recorded: %v", name, arm, err)
			continue
		}
		s.keepPlan(name, arm, p)
	}
}

// plan captures the savepoint Exec and the count statement of one request of
// q as arm's handler sent them (the pgx tracer), replays the request's SET
// LOCALs and the savepoint Exec on one connection in a tenant transaction,
// and EXPLAINs the captured count statement.
func (s *e3Serial) plan(ctx context.Context, arm, token, q string) (e3Plan, error) {
	var p e3Plan
	tr := newPerfTrace()
	rec := serveOn(context.WithValue(ctx, perfTraceKey{}, tr), s.arms[arm].h, http.MethodGet, "/api/v1/records/ticket?"+q, token, "", "")
	if rec.Code != http.StatusOK {
		return p, fmt.Errorf("status %d, body %q", rec.Code, rec.Body.String())
	}
	for i, st := range tr.stmts {
		sql := strings.TrimSpace(st.SQL)
		if strings.HasPrefix(sql, "SAVEPOINT list_count") {
			if i+1 < len(tr.stmts) {
				p.savepoint, p.count = st.SQL, tr.stmts[i+1]
			}
			break
		}
		if strings.HasPrefix(strings.ToUpper(sql), "SET LOCAL") && len(st.Args) == 0 {
			p.pre = append(p.pre, st.SQL)
		}
	}
	if p.savepoint == "" {
		return p, errors.New("no savepoint Exec followed by a count in the request (A2-4 shortcut?)")
	}
	var lines []string
	if err := tenancy.InTenantTx(ctx, s.e.pool, s.tenant, func(tx pgx.Tx) error {
		for _, sql := range append(slices.Clone(p.pre), p.savepoint) {
			if _, err := tx.Exec(ctx, sql); err != nil {
				return fmt.Errorf("%s: %w", sql, err)
			}
		}
		// The one addition: EXPLAIN ANALYZE must finish to show the
		// executed plan. statement_timeout is not a planner setting.
		if _, err := tx.Exec(ctx, "SET LOCAL statement_timeout = 0"); err != nil {
			return err
		}
		rows, err := tx.Query(ctx, "EXPLAIN (ANALYZE, BUFFERS, SETTINGS) "+p.count.SQL, p.count.Args...)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var l string
			if err := rows.Scan(&l); err != nil {
				return err
			}
			lines = append(lines, l)
		}
		return rows.Err()
	}); err != nil {
		return p, err
	}
	p.text = strings.Join(lines, "\n")
	p.gather = strings.Contains(p.text, "Gather")
	p.launched = gateWorkersLaunched.MatchString(p.text)
	return p, nil
}

// keepPlan writes p to plans.md and its summary line to raw.md.
func (s *e3Serial) keepPlan(name, arm string, p e3Plan) {
	*s.plans = append(*s.plans, fmt.Sprintf("### (3e) %s, arm %s (%s)\n\nReplayed on one connection, in `tenancy.InTenantTx` for the large tenant:\n\n```sql\n%s\n-- harness addition, so that ANALYZE can finish (not a planner setting):\nSET LOCAL statement_timeout = 0;\nEXPLAIN (ANALYZE, BUFFERS, SETTINGS)\n%s\n```\n\nGather: %v; workers launched: %v\n\n```\n%s\n```",
		name, arm, s.arms[arm].desc, strings.Join(append(slices.Clone(p.pre), p.savepoint), ";\n")+";", strings.TrimSpace(p.count.SQL), p.gather, p.launched, p.text))
	s.onPlans()
	s.w("- plan check, %s, arm %s: savepoint Exec `%s`; Gather %v, workers launched %v (plans.md)", name, arm, p.savepoint, p.gather, p.launched)
}

// e3CSVHeader and e3CSVRow are serial-samples.csv; the cell name holds commas.
const e3CSVHeader = "cell,block,try,arm,pooled,status,total_state,count_phase_ms,count_stmt_ms,latency_ms\n"

func e3CSVRow(cell string, block, try int, arm string, pooled bool, x e3Sample) string {
	return fmt.Sprintf("%q,%d,%d,%s,%v,%d,%s,%.3f,%.3f,%.3f\n", cell, block, try, arm, pooled, x.code, x.state, x.phase, x.countStmt, x.latency)
}

func unknownOf(ss []e3Sample) int {
	n := 0
	for _, x := range ss {
		if x.code == http.StatusOK && x.state == "unknown" {
			n++
		}
	}
	return n
}

func okWord(ok bool) string {
	if ok {
		return "pass"
	}
	return "FAIL"
}

// e3Stats pools one arm's accepted blocks.
type e3Stats struct {
	n, unknown, errs, counted int
	p                         float64
	phase95, phase99          float64
	stmt99                    float64
	lat95, lat50              float64
	peak                      int64
	bad                       []e3Sample // the non-200 answers, oldest first
}

func e3Pool(bs []e3Block) e3Stats {
	var st e3Stats
	var ph, cs, lat []time.Duration
	for _, b := range bs {
		st.peak = max(st.peak, b.peakWorkers)
		for _, x := range b.samples {
			if x.code != http.StatusOK {
				st.errs++
				st.bad = append(st.bad, x)
				continue
			}
			st.n++
			if x.state == "unknown" {
				st.unknown++
			}
			if x.phase >= 0 {
				st.counted++
				ph = append(ph, time.Duration(x.phase*float64(time.Millisecond)))
			}
			if x.countStmt >= 0 {
				cs = append(cs, time.Duration(x.countStmt*float64(time.Millisecond)))
			}
			lat = append(lat, time.Duration(x.latency*float64(time.Millisecond)))
		}
	}
	if st.n > 0 {
		st.p = float64(st.unknown) / float64(st.n)
	}
	st.phase95, st.phase99, st.stmt99 = pct(ph, .95), pct(ph, .99), pct(cs, .99)
	st.lat50, st.lat95 = pct(lat, .5), pct(lat, .95)
	slices.SortFunc(st.bad, func(x, y e3Sample) int { return x.at.Compare(y.at) })
	return st
}

// report writes the table, the per-cell verdict lines and the overall line,
// and returns the overall verdict word and, for an INCONCLUSIVE, which kind
// it is (only "no workers" can open E2-4's bounded-tail argument).
func (s *e3Serial) report() (overall, kind string) {
	w := s.w
	w("")
	w("## (3e) result")
	w("")
	w("| Cell | Arm | n (200) | non-200 | unknown | unknown share | ran a count | count phase p95 / p99 ms | count statement p99 ms | whole request p50 / p95 ms | peak parallel workers |")
	w("| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |")
	for _, c := range s.cells {
		for _, arm := range []string{"A", "B"} {
			st := e3Pool(c.blocks[arm])
			w("| %s | %s (%s) | %d | %d | %d | %.2f%% | %d | %.1f / %.1f | %.1f | %.1f / %.1f | %d |", c.name, arm, s.arms[arm].desc, st.n, st.errs, st.unknown, 100*st.p, st.counted, st.phase95, st.phase99, st.stmt99, st.lat50, st.lat95, st.peak)
		}
	}
	w("")
	for _, c := range s.cells {
		for _, arm := range []string{"A", "B"} {
			for i, x := range e3Pool(c.blocks[arm]).bad {
				if i == 3 {
					break
				}
				w("- non-200 in %s, arm %s, number %d: status %d, body %q", c.name, arm, i+1, x.code, x.body)
			}
		}
	}
	var in []e3Inputs
	for _, c := range s.cells {
		a, b := e3Pool(c.blocks["A"]), e3Pool(c.blocks["B"])
		x := e3Inputs{name: c.name, band: c.band, hostBad: c.hostBad, pA: a.p, pB: b.p, nA: a.n, nB: b.n, errsA: a.errs, errsB: b.errs, p95A: a.lat95, peakB: b.peak}
		if pa, ok := c.plan["A"]; ok {
			x.planned, x.gatherA = true, pa.gather
			x.gatherB, x.launchedB = c.plan["B"].gather, c.plan["B"].launched
		}
		in = append(in, x)
	}
	overall, noWorkers, lines := e3Verdict(in)
	if s.smoke {
		w("SMOKE, no verdict")
		return "SMOKE, no verdict", ""
	}
	for _, l := range lines {
		w("- %s", l)
	}
	w("")
	w("**(3e) %s**", overall)
	if overall == "INCONCLUSIVE" {
		kind = "INCONCLUSIVE: another run is required; the merge gate stays closed"
		if noWorkers {
			kind = "INCONCLUSIVE, sole reason: arm B launched no workers"
		}
		w("")
		w("%s", kind)
	}
	return overall, kind
}

// tempRise is the temp_files and temp_bytes rise over every block tried.
func (s *e3Serial) tempRise() (files, bytes int64) {
	for _, c := range s.cells {
		for _, b := range c.tried {
			files, bytes = files+b.tempFiles, bytes+b.tempBytes
		}
	}
	return files, bytes
}

// e3ErrsFail non-200 answers in arm A with none in arm B are a FAIL. At equal
// n, k errors all land in arm A by chance with probability 2^-k (3% at 5).
const e3ErrsFail = 5

type e3Inputs struct {
	name                        string
	band                        bool   // arm A's whole-request p95 is gated
	hostBad                     string // the cell is void: a block never passed its gates
	pA, pB                      float64
	nA, nB                      int
	errsA, errsB                int // non-200 answers in the pooled blocks, per arm
	p95A                        float64
	planned                     bool // the plan check ran on this cell
	gatherA, gatherB, launchedB bool
	peakB                       int64
}

// e3Verdict is (3e)'s rule. FAIL first: an arm-A-only failure on any cell, or
// an unknown-share miss on a cell that is itself conclusive. Then
// INCONCLUSIVE, then PASS. noWorkers: the verdict is INCONCLUSIVE and its sole
// reason is that arm B launched no workers (E2-4's bounded-tail argument).
func e3Verdict(in []e3Inputs) (overall string, noWorkers bool, lines []string) {
	fail, other, idle := false, len(in) == 0, false
	for _, x := range in {
		var bad, inc, notes []string
		if x.planned && x.gatherA {
			bad = append(bad, "Gather under arm A")
		}
		if x.band && x.hostBad == "" && x.nA > 0 && x.p95A > e3BandP95MS {
			bad = append(bad, "arm A band (ii) p95 over the limit")
		}
		if x.hostBad == "" && x.errsA >= e3ErrsFail && x.errsB == 0 {
			bad = append(bad, fmt.Sprintf("%d non-200 answers in arm A, none in arm B", x.errsA))
		} else if x.errsA+x.errsB > 0 {
			inc = append(inc, fmt.Sprintf("non-200 answers: arm A %d, arm B %d", x.errsA, x.errsB))
		}
		if x.hostBad != "" {
			inc = append(inc, "host: "+x.hostBad)
		} else if x.nA == 0 || x.nB == 0 {
			inc = append(inc, "an arm has no samples")
		}
		if x.planned && !x.gatherB && x.peakB > 0 {
			inc = append(inc, "no Gather in arm B's plan, but workers ran")
		}
		other = other || len(inc) > 0
		if x.planned && !x.launchedB && x.peakB == 0 {
			inc, idle = append(inc, "arm B launched no workers (plan and peak 0)"), true
		}
		se := math.Sqrt(x.pA*(1-x.pA)/float64(max(x.nA, 1)) + x.pB*(1-x.pB)/float64(max(x.nB, 1)))
		margin := math.Max(.01, 2*se)
		if x.pA > x.pB+margin {
			if len(inc) == 0 {
				bad = append(bad, "unknown share over the margin")
			} else {
				notes = append(notes, "unknown share over the margin, not scored")
			}
		}
		v := "PASS"
		switch {
		case len(bad) > 0:
			v, fail = "FAIL", true
		case len(inc) > 0:
			v = "INCONCLUSIVE"
		}
		l := fmt.Sprintf("**(3e) cell %s: %s**: pA %.4f (n %d), pB %.4f (n %d), SE %.4f, margin %.4f, limit pB + margin %.4f; arm A whole-request p95 %.1f ms",
			x.name, v, x.pA, x.nA, x.pB, x.nB, se, margin, x.pB+margin, x.p95A)
		if x.band {
			l += fmt.Sprintf(" (limit %.0f)", e3BandP95MS)
		}
		if x.planned {
			l += fmt.Sprintf("; plan: Gather A %v, Gather B %v, B launched %v; arm B peak workers %d", x.gatherA, x.gatherB, x.launchedB, x.peakB)
		}
		if all := slices.Concat(bad, inc, notes); len(all) > 0 {
			l += " (" + strings.Join(all, "; ") + ")"
		}
		lines = append(lines, l)
	}
	switch {
	case fail:
		return "FAIL", false, lines
	case other || idle:
		return "INCONCLUSIVE", idle && !other, lines
	}
	return "PASS", false, lines
}

// gateTempReading is TestA2Gate's temp reading: pg_ls_tmpdir() is the (3a)
// leak check (ADR-006-E2 E2-3); temp_files / temp_bytes are spill context
// only; base/pgsql_tmp entries and parallel worker backends are reported.
type gateTempReading struct {
	files, bytes, tmpFiles, tmpBytes, tmpEntries, workers int64
}

func gateTemp(t *testing.T, super *pgx.Conn) gateTempReading {
	t.Helper()
	var r gateTempReading
	for _, q := range []struct {
		sql  string
		dest []any
	}{
		{`SELECT temp_files, temp_bytes FROM pg_stat_database WHERE datname = current_database()`, []any{&r.files, &r.bytes}},
		{`SELECT count(*), coalesce(sum(size), 0) FROM pg_ls_tmpdir()`, []any{&r.tmpFiles, &r.tmpBytes}},
		{`SELECT count(*) FROM pg_ls_dir('base/pgsql_tmp', true, false)`, []any{&r.tmpEntries}},
		{`SELECT count(*) FROM pg_stat_activity WHERE backend_type = 'parallel worker'`, []any{&r.workers}},
	} {
		if err := super.QueryRow(t.Context(), q.sql).Scan(q.dest...); err != nil {
			t.Fatal(err)
		}
	}
	return r
}

func (r gateTempReading) String() string {
	return fmt.Sprintf("pg_ls_tmpdir() %d files (%d bytes); spill context only (proves a spill happened, never a leak): temp_files %d, temp_bytes %d; reported, not gated: base/pgsql_tmp entries %d, parallel workers running %d",
		r.tmpFiles, r.tmpBytes, r.files, r.bytes, r.tmpEntries, r.workers)
}

// gateLeak is the (3a) leak verdict (E2-3): no pg_ls_tmpdir() file after the
// run and none 60 s later. It writes one line per reading and a FAIL line for
// any file, and reports whether both readings were clean.
func gateLeak(w func(string, ...any), label string, after, after60 gateTempReading) bool {
	ok := true
	for _, x := range []struct {
		when string
		r    gateTempReading
	}{{"after the run", after}, {"60 s after the run", after60}} {
		w("- (3a) leak reading %s, %s: %s", label, x.when, x.r)
		if x.r.tmpFiles != 0 {
			ok = false
			w("- **FAIL: (3a) leak, %s, %s: pg_ls_tmpdir() lists %d pgsql_tmp files (%d bytes)**", label, x.when, x.r.tmpFiles, x.r.tmpBytes)
		}
	}
	return ok
}

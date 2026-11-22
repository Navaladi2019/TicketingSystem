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
	"fmt"
	"math"
	"net/http"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"encoding/json"

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
}

// e3Block is one accepted or rejected block of a cell.
type e3Block struct {
	arm, label    string
	before, after gateHost
	samples       []e3Sample
	peakWorkers   int64
	workerSamples int
}

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
			if s.last != nil && s.last.ok() {
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
			s.w("- %s: n %d, unknown %d, peak parallel workers %d (%d samples); gates %s", label, len(b.samples), unknownOf(b.samples), b.peakWorkers, b.workerSamples, okWord(b.ok()))
			if b.ok() {
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
		for _, x := range got.samples {
			fmt.Fprintf(&s.csv, "%s,%d,%s,%d,%s,%.3f,%.3f,%.3f\n", name, i+1, arm, x.code, x.state, x.phase, x.countStmt, x.latency)
		}
	}
	return r
}

// block runs one arm for s.d at s.clients and samples pg_stat_activity's
// parallel worker backends every 50 ms on its own connection.
func (s *e3Serial) block(t *testing.T, arm, label string, reqs []perfReq) e3Block {
	t.Helper()
	h := s.arms[arm].h
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

// planCheck captures, per arm, the savepoint Exec and the count statement of
// one request of q as the handler sent them (the pgx tracer), replays the
// request's SET LOCALs and the savepoint Exec on one connection in a tenant
// transaction, and EXPLAINs the captured count statement.
func (s *e3Serial) planCheck(t *testing.T, r *e3CellResult, token, q string) {
	t.Helper()
	for _, arm := range []string{"A", "B"} {
		tr := newPerfTrace()
		rec := serveOn(context.WithValue(t.Context(), perfTraceKey{}, tr), s.arms[arm].h, http.MethodGet, "/api/v1/records/ticket?"+q, token, "", "")
		if rec.Code != http.StatusOK {
			t.Fatalf("(3e) plan check arm %s: %d: %s", arm, rec.Code, rec.Body)
		}
		var p e3Plan
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
			t.Fatalf("(3e) plan check arm %s: no savepoint Exec in the request (A2-4 shortcut?)", arm)
		}
		var lines []string
		if err := tenancy.InTenantTx(t.Context(), s.e.pool, s.tenant, func(tx pgx.Tx) error {
			for _, sql := range append(slices.Clone(p.pre), p.savepoint) {
				if _, err := tx.Exec(t.Context(), sql); err != nil {
					return fmt.Errorf("%s: %w", sql, err)
				}
			}
			// The one addition: EXPLAIN ANALYZE must finish to show the
			// executed plan. statement_timeout is not a planner setting.
			if _, err := tx.Exec(t.Context(), "SET LOCAL statement_timeout = 0"); err != nil {
				return err
			}
			rows, err := tx.Query(t.Context(), "EXPLAIN (ANALYZE, BUFFERS, SETTINGS) "+p.count.SQL, p.count.Args...)
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
			t.Fatal(err)
		}
		p.text = strings.Join(lines, "\n")
		p.gather = strings.Contains(p.text, "Gather")
		p.launched = gateWorkersLaunched.MatchString(p.text)
		r.plan[arm] = p
		*s.plans = append(*s.plans, fmt.Sprintf("### (3e) %s, arm %s (%s)\n\nReplayed on one connection, in `tenancy.InTenantTx` for the large tenant:\n\n```sql\n%s\n-- harness addition, so that ANALYZE can finish (not a planner setting):\nSET LOCAL statement_timeout = 0;\nEXPLAIN (ANALYZE, BUFFERS, SETTINGS)\n%s\n```\n\nGather: %v; workers launched: %v\n\n```\n%s\n```",
			r.name, arm, s.arms[arm].desc, strings.Join(append(slices.Clone(p.pre), p.savepoint), ";\n")+";", strings.TrimSpace(p.count.SQL), p.gather, p.launched, p.text))
		s.onPlans()
		s.w("- plan check, arm %s: savepoint Exec `%s`; Gather %v, workers launched %v (plans.md)", arm, p.savepoint, p.gather, p.launched)
	}
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
}

func e3Pool(bs []e3Block) e3Stats {
	var st e3Stats
	var ph, cs, lat []time.Duration
	for _, b := range bs {
		st.peak = max(st.peak, b.peakWorkers)
		for _, x := range b.samples {
			if x.code != http.StatusOK {
				st.errs++
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
	return st
}

// report writes the table, the per-cell verdict lines and the overall line,
// and returns the overall verdict word.
func (s *e3Serial) report() string {
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
	fail, inconclusive := false, false
	var why []string
	for _, c := range s.cells {
		if c.hostBad != "" {
			inconclusive = true
			w("- **(3e) cell %s: INCONCLUSIVE (host): %s**", c.name, c.hostBad)
			why = append(why, c.name+": host")
			continue
		}
		a, b := e3Pool(c.blocks["A"]), e3Pool(c.blocks["B"])
		se := math.Sqrt(a.p*(1-a.p)/float64(max(a.n, 1)) + b.p*(1-b.p)/float64(max(b.n, 1)))
		margin := math.Max(0.01, 2*se)
		shareOK := a.p <= b.p+margin
		var notes []string
		cellFail := !shareOK
		if !shareOK {
			notes = append(notes, "unknown share")
		}
		// Arm-A-only checks (FAIL beats INCONCLUSIVE).
		if c.band && a.lat95 > e3BandP95MS {
			cellFail = true
			notes = append(notes, fmt.Sprintf("arm A band (ii) p95 %.1f > %.0f ms", a.lat95, e3BandP95MS))
		}
		pa, hasPlan := c.plan["A"]
		if hasPlan && pa.gather {
			cellFail = true
			notes = append(notes, "Gather under arm A")
		}
		cellInc := false
		if hasPlan {
			pb := c.plan["B"]
			if !pb.launched && b.peak == 0 {
				cellInc = true
				notes = append(notes, "arm B launched no workers (plan and peak 0)")
			}
			if !pb.gather {
				notes = append(notes, "no Gather under arm B (reported)")
			}
		}
		v := "PASS"
		switch {
		case cellFail:
			v, fail = "FAIL", true
		case cellInc:
			v, inconclusive = "INCONCLUSIVE", true
		}
		line := fmt.Sprintf("pA %.4f (%d/%d), pB %.4f (%d/%d), SE %.4f, margin max(1 point, 2 x SE) = %.4f: pA %s pB + margin (%.4f)",
			a.p, a.unknown, a.n, b.p, b.unknown, b.n, se, margin, map[bool]string{true: "<=", false: ">"}[shareOK], b.p+margin)
		if c.band {
			line += fmt.Sprintf("; arm A whole-request p95 %.1f ms (limit %.0f)", a.lat95, e3BandP95MS)
		} else {
			line += fmt.Sprintf("; arm A whole-request p95 %.1f ms (reported)", a.lat95)
		}
		if hasPlan {
			line += fmt.Sprintf("; plan: arm A Gather %v, arm B Gather %v (workers launched %v), arm B peak workers %d", pa.gather, c.plan["B"].gather, c.plan["B"].launched, b.peak)
		}
		if len(notes) > 0 {
			line += " (" + strings.Join(notes, "; ") + ")"
		}
		w("- **(3e) cell %s: %s**: %s", c.name, v, line)
		why = append(why, c.name+": "+v)
	}
	overall := "PASS"
	switch {
	case fail:
		overall = "FAIL"
	case inconclusive:
		overall = "INCONCLUSIVE"
	}
	w("")
	w("**(3e) %s** (%s)", overall, strings.Join(why, "; "))
	return overall
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

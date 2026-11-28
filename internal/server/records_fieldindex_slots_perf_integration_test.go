//go:build integration && perf

// Phase "slots", TICK-1205 (ADR-006-A3 draft, A3-10): what arm (b)+group
// costs in the slot form A3-2 gives it, on the TICK-992 harness (800k seed,
// 400k type, 8 clients, real handler, ACL and RLS). Scratch DDL only, dropped
// at the end; nothing here is a migration.
//
// The slot form, mirrored in scratch DDL:
//   - A3-2: one statement adds k = 2 nullable typed columns per value type
//     (fx_num_1..2, fx_text_1..2 COLLATE "C", fx_ts_1..2, fx_bool_1..2),
//     CREATE STATISTICS (mcv) on (tenant_id, entity_definition_id, slot) and
//     one partial index per slot, (tenant_id, entity_definition_id, slot,
//     assignment_group_id, updated_at) WHERE slot IS NOT NULL, built
//     CONCURRENTLY over all-NULL columns.
//   - A3-4/A3-5: the field-to-slot map is a scratch table (spike_slot_map),
//     standing in for record_index_fields.slot/slot_state; a BEFORE INSERT OR
//     UPDATE OF data trigger sets NEW.<slot> with record_field_coerce for each
//     mapped slot in backfilling or ready. D4's side-row trigger is unchanged,
//     so a promoted field keeps its side rows.
//   - A3-5 backfill: 1,000-record batches (D10.2) of UPDATE records SET slot
//     = record_field_coerce(data -> key), no version or updated_at change.
//     As the superuser, not nava_worker (no scratch grants), and without the
//     D9 barrier (no concurrent publish runs here).
//   - A3-6: the list term is the slot predicate (WithListIndexTerm); S-b1's
//     per-group merge page splits the ACL's group branch into one ordered
//     UNION ALL arm per visible group (WithListQuery).
//
// Slots: score -> fx_num_1, pk -> fx_num_2, tier -> fx_text_1, zone ->
// fx_text_2, opened -> fx_ts_1, b2 -> fx_bool_1. pk is TICK-1039's fixture
// (arms phase), written into data as well, so the trigger keeps it.
//
// Env: NAVA_PERF_LIST_SECONDS (20) per list cell, NAVA_PERF_SLOT_WRITE_SECONDS
// (30) per write cell, NAVA_PERF_SLOT_LIVE_SECONDS (60) per live-mix control,
// NAVA_PERF_SLOT_CURVE (0,8,16,32,64) extra slot indexes for item 4,
// NAVA_PERF_SLOT_COPIES (4) copies of the large type for S-b3 (2M).

package server_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Navaladi2019/TicketingSystem/internal/authz/acltest"
	"github.com/Navaladi2019/TicketingSystem/internal/records/pipeline"
	"github.com/Navaladi2019/TicketingSystem/internal/server"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

type slCol struct{ col, sqlType, vt string }

// slA3 is A3-2's slot set at k = 2.
func slA3() []slCol {
	return []slCol{
		{"fx_num_1", "double precision", "num"}, {"fx_num_2", "double precision", "num"},
		{"fx_text_1", `text COLLATE "C"`, "text"}, {"fx_text_2", `text COLLATE "C"`, "text"},
		{"fx_ts_1", "timestamptz", "ts"}, {"fx_ts_2", "timestamptz", "ts"},
		{"fx_bool_1", "boolean", "bool"}, {"fx_bool_2", "boolean", "bool"},
	}
}

// slFields is the large type's field-to-slot map.
func slFields() map[string]string {
	return map[string]string{"score": "fx_num_1", "pk": "fx_num_2", "tier": "fx_text_1", "zone": "fx_text_2", "opened": "fx_ts_1", "b2": "fx_bool_1"}
}

func slVTs() map[string]string {
	return map[string]string{"score": "num", "pk": "num", "tier": "text", "zone": "text", "opened": "ts", "b2": "bool"}
}

// slSets are the write arms' promoted fields, by count.
func slSets() map[int][]string {
	return map[int][]string{0: nil, 1: {"score"}, 3: {"score", "tier", "opened"}, 5: {"score", "tier", "opened", "zone", "b2"}}
}

func slGroupIdx(col string) string { return "spike_" + col + "_grp" }
func slPlainIdx(col string) string { return "spike_" + col + "_plain" }

// ---- the list seams ---------------------------------------------------------

// slState renders a mapped field's filter as its slot predicate and, with
// split, the ACL's group branch as one arm per group.
type slState struct {
	mu      sync.Mutex
	cols    map[uuid.UUID]string
	split   bool
	splits  int
	skipped []string
}

func (s *slState) set(cols map[uuid.UUID]string, split bool) {
	s.mu.Lock()
	s.cols, s.split, s.splits, s.skipped = cols, split, 0, nil
	s.mu.Unlock()
}

func (s *slState) term(_ context.Context, _ pgx.Tx, in server.IndexTerm) string {
	s.mu.Lock()
	col := s.cols[in.FieldID]
	s.mu.Unlock()
	if col == "" {
		return in.D7
	}
	return "(" + strings.ReplaceAll(in.Cmp, "v.value", "r."+col) + " AND " + in.FieldArg + " IS NOT NULL)"
}

var slGroupArm = regexp.MustCompile(`r\.assignment_group_id = ANY\(\$(\d+)::uuid\[\]\)`)

func (s *slState) query(q *pipeline.ListQuery) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.split {
		return
	}
	if len(q.Branches) < 2 || q.Keyset != "" {
		s.skipped = append(s.skipped, fmt.Sprintf("%d branches, keyset %q", len(q.Branches), q.Keyset))
		return
	}
	for i, b := range q.Branches {
		if strings.Contains(b, "NOT (") {
			continue
		}
		m := slGroupArm.FindStringSubmatch(b)
		if m == nil {
			continue
		}
		// The store binds $1 and $2 (tenant, type): Args starts at $3.
		k, _ := strconv.Atoi(m[1])
		var gs []uuid.UUID
		switch a := q.Args[k-3].(type) {
		case []uuid.UUID:
			gs = a
		case []string:
			for _, x := range a {
				gs = append(gs, uuid.MustParse(x))
			}
		default:
			s.skipped = append(s.skipped, fmt.Sprintf("group argument is %T", a))
			return
		}
		var nums []int
		for _, g := range gs {
			q.Args = append(q.Args, g)
			nums = append(nums, len(q.Args)+2)
		}
		arms := func(text string) []string {
			out := make([]string, 0, len(nums))
			for _, n := range nums {
				out = append(out, strings.Replace(text, m[0], fmt.Sprintf("r.assignment_group_id = $%d::uuid", n), 1))
			}
			return out
		}
		n := len(q.Branches)
		splice := func(xs []string, with []string) []string {
			out := append(slices.Clone(xs[:i]), with...)
			return append(out, xs[i+1:]...)
		}
		if len(q.CountBranches) == n {
			q.CountBranches = splice(q.CountBranches, arms(q.CountBranches[i]))
		}
		if len(q.Materialize) == n {
			rep := make([]bool, len(nums))
			for j := range rep {
				rep[j] = q.Materialize[i]
			}
			q.Materialize = append(append(slices.Clone(q.Materialize[:i]), rep...), q.Materialize[i+1:]...)
		}
		if len(q.CountDrivers) == n {
			rep := make([]pipeline.CountDriver, len(nums))
			for j := range rep {
				rep[j] = q.CountDrivers[i]
			}
			q.CountDrivers = append(append(slices.Clone(q.CountDrivers[:i]), rep...), q.CountDrivers[i+1:]...)
		}
		q.Branches = splice(q.Branches, arms(b))
		s.splits++
		return
	}
	s.skipped = append(s.skipped, "no group branch")
}

// ---- plan readers -------------------------------------------------------------

var (
	slNode      = regexp.MustCompile(`^\s*(?:->\s+)?([A-Z][A-Za-z ]*?)(?: using (\S+))?(?: on \S+(?: \S+)?)?\s+\(cost=[0-9.]+\.\.[0-9.]+ rows=(\d+) width=\d+\) \(actual time=[0-9.]+\.\.[0-9.]+ rows=(\d+) loops=(\d+)\)`)
	slHeapFetch = regexp.MustCompile(`Heap Fetches: (\d+)`)
)

type slPlanNode struct {
	kind, index string
	est, rows   int64 // rows x loops for rows
	depth       int
}

func slNodes(plan string) []slPlanNode {
	var out []slPlanNode
	for _, l := range strings.Split(plan, "\n") {
		m := slNode.FindStringSubmatch(l)
		if m == nil {
			continue
		}
		est, _ := strconv.ParseInt(m[3], 10, 64)
		r, _ := strconv.ParseInt(m[4], 10, 64)
		lp, _ := strconv.ParseInt(m[5], 10, 64)
		out = append(out, slPlanNode{kind: strings.TrimSpace(m[1]), index: m[2], est: est, rows: r * lp, depth: len(l) - len(strings.TrimLeft(l, " "))})
	}
	return out
}

// slPageShape reads S-b1's pass conditions off a page plan: the most index
// rows any one scan of idx read, and the largest input of a full (not
// incremental) Sort node.
func slPageShape(plan, idx string) (maxScan, maxSortIn int64, scans int) {
	ns := slNodes(plan)
	for i, n := range ns {
		if n.index == idx {
			scans++
			maxScan = max(maxScan, n.rows)
		}
		if n.kind == "Sort" {
			// The child is the next node one level deeper.
			for _, c := range ns[i+1:] {
				if c.depth > n.depth {
					maxSortIn = max(maxSortIn, c.rows)
					break
				}
			}
		}
	}
	return maxScan, maxSortIn, scans
}

func slHeapFetches(plan string) int64 {
	var n int64
	for _, m := range slHeapFetch.FindAllStringSubmatch(plan, -1) {
		x, _ := strconv.ParseInt(m[1], 10, 64)
		n += x
	}
	return n
}

// ---- connections, stats, DDL under load ----------------------------------------

func (e *fiEnv) superConn(t *testing.T) *pgx.Conn {
	t.Helper()
	c, err := pgx.ConnectConfig(t.Context(), e.super.Config().Copy())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c.Close(context.Background()) })
	return c
}

type slStats struct {
	lsn                string
	ins, upd, hot      int64
	walRecs, fpi, wByt int64
}

// slStatsWait is longer than PGSTAT_IDLE_INTERVAL (10 s): an idle backend's
// pending table and WAL counters are flushed by then.
const slStatsWait = 11 * time.Second

func (e *fiEnv) slStats(t *testing.T) slStats {
	t.Helper()
	var s slStats
	if err := e.super.QueryRow(t.Context(), `SELECT pg_current_wal_lsn()::text,
		(SELECT n_tup_ins FROM pg_stat_user_tables WHERE relid = 'records'::regclass),
		(SELECT n_tup_upd FROM pg_stat_user_tables WHERE relid = 'records'::regclass),
		(SELECT n_tup_hot_upd FROM pg_stat_user_tables WHERE relid = 'records'::regclass),
		wal_records, wal_fpi, wal_bytes::bigint FROM pg_stat_wal`).Scan(&s.lsn, &s.ins, &s.upd, &s.hot, &s.walRecs, &s.fpi, &s.wByt); err != nil {
		t.Fatal(err)
	}
	return s
}

func (e *fiEnv) walSince(t *testing.T, lsn string) int64 {
	t.Helper()
	var n int64
	if err := e.super.QueryRow(t.Context(), `SELECT pg_wal_lsn_diff(pg_current_wal_lsn(), $1::pg_lsn)::bigint`, lsn).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// slDDL is one DDL statement run on its own connection while a sampler reads
// its locks on records and the backends waiting on a lock, every 2 ms.
type slDDL struct {
	label       string
	start, end  time.Time
	modes       map[string][2]time.Time // first and last sample holding the mode
	waited      time.Duration           // the DDL itself waiting on a lock
	blockedMax  int                     // other backends waiting on a lock at once
	blockedTime time.Duration
	samples     int
}

func (e *fiEnv) ddl(t *testing.T, label, sql string) slDDL {
	t.Helper()
	ctx := t.Context()
	run, smp := e.superConn(t), e.superConn(t)
	var pid int
	if err := run.QueryRow(ctx, "SELECT pg_backend_pid()").Scan(&pid); err != nil {
		t.Fatal(err)
	}
	if _, err := run.Exec(ctx, "SET lock_timeout = '30s'"); err != nil {
		t.Fatal(err)
	}
	r := slDDL{label: label, modes: map[string][2]time.Time{}}
	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		last := time.Now()
		for {
			select {
			case <-stop:
				return
			default:
			}
			var modes string
			var waiting bool
			var blocked int
			if err := smp.QueryRow(ctx, `SELECT coalesce((SELECT string_agg(DISTINCT mode, ',') FROM pg_locks WHERE pid = $1 AND relation = 'records'::regclass AND granted), ''),
				EXISTS (SELECT 1 FROM pg_locks WHERE pid = $1 AND NOT granted),
				(SELECT count(*) FROM pg_stat_activity WHERE wait_event_type = 'Lock' AND pid <> $1)::int`, pid).Scan(&modes, &waiting, &blocked); err != nil {
				return
			}
			now := time.Now()
			r.samples++
			for _, m := range strings.Split(modes, ",") {
				if m == "" {
					continue
				}
				sp, ok := r.modes[m]
				if !ok {
					sp[0] = now
				}
				sp[1] = now
				r.modes[m] = sp
			}
			if waiting {
				r.waited += now.Sub(last)
			}
			if blocked > 0 {
				r.blockedTime += now.Sub(last)
				r.blockedMax = max(r.blockedMax, blocked)
			}
			last = now
			time.Sleep(2 * time.Millisecond)
		}
	}()
	time.Sleep(20 * time.Millisecond)
	r.start = time.Now()
	_, err := run.Exec(ctx, sql)
	r.end = time.Now()
	close(stop)
	wg.Wait()
	if err != nil {
		t.Fatalf("%s: %v", label, err)
	}
	return r
}

func (d slDDL) modesText() string {
	if len(d.modes) == 0 {
		return "none sampled (held < 2 ms)"
	}
	var ks []string
	for k := range d.modes {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	var out []string
	for _, k := range ks {
		sp := d.modes[k]
		out = append(out, fmt.Sprintf("%s (seen over %s)", k, sp[1].Sub(sp[0]).Round(time.Millisecond)))
	}
	return strings.Join(out, ", ")
}

// overlap summarises the live ops in flight at any point of [from, to].
func slOverlap(ops []fiOp, from, to time.Time) (n int, p95, maxMS float64) {
	var d []time.Duration
	for _, o := range ops {
		if o.at.Before(to) && o.at.Add(o.d).After(from) {
			d = append(d, o.d)
		}
	}
	if len(d) == 0 {
		return 0, 0, 0
	}
	return len(d), pct(d, .95), fiMS(slices.Max(d))
}

func (e *fiEnv) ddlRow(d slDDL, ops []fiOp) string {
	n, p95, mx := slOverlap(ops, d.start, d.end)
	return fmt.Sprintf("| %s | %s | %s | %s | %d / %s | %d | %.1f | %.1f |", d.label, d.end.Sub(d.start).Round(time.Millisecond), d.modesText(),
		d.waited.Round(time.Millisecond), d.blockedMax, d.blockedTime.Round(time.Millisecond), n, p95, mx)
}

const slDDLHead = "| Statement | Wall | Locks on records held (sampled every 2 ms) | DDL waiting on a lock | Writers waiting on a lock: max / time | Live ops overlapping | their p95 ms | their max ms |\n| --- | --- | --- | --- | --- | --- | --- | --- |"

// liveRun is the live mix running in the background.
type liveRun struct {
	cancel context.CancelFunc
	done   chan struct{}
	ops    []fiOp
	errs   map[string]int64
	start  time.Time
}

func (e *fiEnv) startLive(t *testing.T, clients int, creates []perfReq, mine [][]uuid.UUID) *liveRun {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	l := &liveRun{cancel: cancel, done: make(chan struct{}), start: time.Now()}
	go func() {
		defer close(l.done)
		l.ops, l.errs = e.liveMix(ctx, t, clients, creates, mine)
	}()
	return l
}

func (l *liveRun) stop() {
	l.cancel()
	<-l.done
}

// ---- write cells ------------------------------------------------------------

type slWrite struct {
	arm, op   string
	p50, p95  float64
	n         int
	errs      int64
	walPerOp  float64
	fpiPerOp  float64
	recPerOp  float64
	upd, hot  int64
	ins       int64
	tps       float64
	walTotalB int64
}

func (w slWrite) row() string {
	hot := "-"
	if w.upd > 0 {
		hot = fmt.Sprintf("%d / %d (%.0f%%)", w.hot, w.upd, 100*float64(w.hot)/float64(w.upd))
	}
	return fmt.Sprintf("| %s | %s | %d | %.1f | %.1f | **%.1f** | %.0f | %.1f | %.2f | %s | %d |", w.arm, w.op, w.n, w.tps, w.p50, w.p95, w.walPerOp, w.recPerOp, w.fpiPerOp, hot, w.errs)
}

const slWriteHead = "| Arm | Write | n | ops/s | p50 ms | p95 ms | WAL B/op | WAL records/op | FPI/op | records HOT / updates | errors |\n| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |"

// slVer is each patch target's version, carried across cells.
type slVer struct {
	mu sync.Mutex
	v  map[uuid.UUID]int64
}

func (s *slVer) get(id uuid.UUID) int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	if v, ok := s.v[id]; ok {
		return v
	}
	return 1
}

func (s *slVer) put(id uuid.UUID, v int64) {
	s.mu.Lock()
	s.v[id] = v
	s.mu.Unlock()
}

// patchRun PATCHes ids (one slice per client) round-robin for d, as the
// large tenant's admin, with body(visit) as the merge patch.
func (e *fiEnv) patchRun(t *testing.T, d time.Duration, ids [][]uuid.UUID, ver *slVer, body func(visit int) string) ([]time.Duration, int64) {
	t.Helper()
	var (
		mu   sync.Mutex
		lat  []time.Duration
		errs atomic.Int64
		wg   sync.WaitGroup
	)
	deadline := time.Now().Add(d)
	for c := range ids {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var local []time.Duration
			for i := 0; time.Now().Before(deadline); i++ {
				id := ids[c][i%len(ids[c])]
				visit := i / len(ids[c])
				req := httptest.NewRequestWithContext(t.Context(), http.MethodPatch, "/api/v1/records/ticket/"+id.String(), strings.NewReader(body(visit+c)))
				req.Header.Set("Authorization", "Bearer admin-large")
				req.Header.Set("Content-Type", "application/merge-patch+json")
				req.Header.Set("Idempotency-Key", uuid.NewString())
				req.Header.Set("If-Match", `W/"`+strconv.FormatInt(ver.get(id), 10)+`"`)
				rec := httptest.NewRecorder()
				t0 := time.Now()
				e.h.ServeHTTP(rec, req)
				el := time.Since(t0)
				if et := strings.Trim(strings.TrimPrefix(rec.Header().Get("ETag"), "W/"), `"`); et != "" {
					if n, err := strconv.ParseInt(et, 10, 64); err == nil {
						ver.put(id, n)
					}
				}
				if rec.Code != http.StatusOK {
					errs.Add(1)
					continue
				}
				local = append(local, el)
			}
			mu.Lock()
			lat = append(lat, local...)
			mu.Unlock()
		}()
	}
	wg.Wait()
	return lat, errs.Load()
}

// slMeasure runs one write cell between two stats snapshots. ops is the
// number of writes the cell sent beyond its timed ones (warm-ups).
func (e *fiEnv) slMeasure(t *testing.T, arm, op string, d time.Duration, fn func() ([]time.Duration, int64, int)) slWrite {
	t.Helper()
	before := e.slStats(t)
	lat, errs, extra := fn()
	wal := e.walSince(t, before.lsn)
	time.Sleep(slStatsWait)
	after := e.slStats(t)
	ops := float64(len(lat) + extra)
	w := slWrite{arm: arm, op: op, n: len(lat), errs: errs, p50: pct(lat, .5), p95: pct(lat, .95), tps: float64(len(lat)) / d.Seconds(), walTotalB: wal}
	if ops > 0 {
		w.walPerOp = float64(wal) / ops
		w.fpiPerOp = float64(after.fpi-before.fpi) / ops
		w.recPerOp = float64(after.walRecs-before.walRecs) / ops
	}
	w.upd, w.hot, w.ins = after.upd-before.upd, after.hot-before.hot, after.ins-before.ins
	return w
}

// ---- the phase --------------------------------------------------------------

func (e *fiEnv) slots(t *testing.T, clients int) {
	t.Helper()
	ctx := t.Context()
	listD := time.Duration(perfEnvInt("NAVA_PERF_LIST_SECONDS", 20)) * time.Second
	writeD := time.Duration(perfEnvInt("NAVA_PERF_SLOT_WRITE_SECONDS", 30)) * time.Second
	liveD := time.Duration(perfEnvInt("NAVA_PERF_SLOT_LIVE_SECONDS", 60)) * time.Second
	a6, a50 := e.tokens["agent-1"], e.tokens["a50-1"]
	// NAVA_PERF_SLOT_QUICK=1 is a harness smoke: no write arms, no plain
	// indexes, no S-c2/S-c3 cells, no backfill pause. Never scored.
	quick := os.Getenv("NAVA_PERF_SLOT_QUICK") == "1"
	e.w("")
	e.w("## TICK-1205: promoted filter slots (ADR-006-A3 draft), %d clients, lists %s per cell, writes %s per cell, work_mem %s", clients, listD, writeD, e.scalar(t, "current_setting('work_mem')"))
	e.w("")
	e.w("- autovacuum: %s", e.scalar(t, `SELECT string_agg(name || '=' || setting, ', ' ORDER BY name) FROM pg_settings WHERE name IN ('autovacuum', 'autovacuum_vacuum_scale_factor', 'checkpoint_timeout', 'max_wal_size', 'full_page_writes', 'wal_compression')`))
	t0 := time.Now()

	// The large type keeps the insert run's five D7 fields (tier, zone,
	// score, opened, b2); every other tenant keeps all eight band fields.
	e.largeFieldsTo(t, fiWriteFields())
	// pk, TICK-1039's fixture, written into data so the slot trigger keeps it.
	e.exec(t, `CREATE TABLE sl_rec AS SELECT r.id, r.updated_at,
			`+spVisible+` AS v6,
			`+strings.NewReplacer("$3", "$5", "$4", "$6").Replace(spVisible)+` AS v50,
			(('x' || substr(md5(r.id::text || 'sp'), 1, 7))::bit(28)::int) AS k,
			NULL::float8 AS v
		FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2`,
		e.large.id, e.largeDef, a6.PrincipalID, a6.Groups, a50.PrincipalID, a50.Groups)
	t.Cleanup(func() { _, _ = e.super.Exec(context.Background(), "DROP TABLE IF EXISTS sl_rec") })
	e.exec(t, `UPDATE sl_rec SET v = 40 WHERE id IN (SELECT id FROM sl_rec ORDER BY updated_at, id LIMIT 8000)`)
	e.exec(t, `UPDATE sl_rec s SET v = CASE WHEN x.rn <= 26593 THEN 10 WHEN x.rn <= 26593 + 20993 THEN 20 ELSE 30 END
		FROM (SELECT id, row_number() OVER (ORDER BY k, id) AS rn FROM sl_rec WHERE v IS NULL AND NOT v6 AND NOT v50) x
		WHERE s.id = x.id AND x.rn <= 26593 + 20993 + 19999`)
	e.exec(t, `UPDATE sl_rec s SET v = 20 FROM (SELECT id, row_number() OVER (ORDER BY k, id) AS rn FROM sl_rec WHERE v IS NULL AND v6) x
		WHERE s.id = x.id AND x.rn <= 494`)
	e.exec(t, `UPDATE sl_rec s SET v = CASE WHEN x.rn <= 200000 THEN 3 ELSE 4 END
		FROM (SELECT id, row_number() OVER (ORDER BY k, id) AS rn FROM sl_rec WHERE v IS NULL) x WHERE s.id = x.id`)
	e.exec(t, "CREATE INDEX ON sl_rec (id)")
	e.exec(t, "ANALYZE sl_rec")
	e.w("- pk fixture (value: records, visible to a6 / a50): %s", e.scalar(t, `SELECT string_agg(v || ': ' || n || ', ' || n6 || ' / ' || n50, '; ' ORDER BY v)
		FROM (SELECT v, count(*) n, count(*) FILTER (WHERE v6) n6, count(*) FILTER (WHERE v50) n50 FROM sl_rec GROUP BY v) x`))
	e.exec(t, "SET session_replication_role = replica")
	e.exec(t, `UPDATE records r SET data = r.data || jsonb_build_object('pk', s.v) FROM sl_rec s WHERE r.tenant_id = $1 AND r.id = s.id`, e.large.id)
	e.exec(t, "SET session_replication_role = origin")
	var pkField uuid.UUID
	if err := e.super.QueryRow(ctx, `INSERT INTO record_index_fields (tenant_id, id, entity_definition_id, field_key, value_type, state, progress, requested_version)
		VALUES ($1, gen_random_uuid(), $2, 'pk', 'num', 'ready', 100, 1) RETURNING id`, e.large.id, e.largeDef).Scan(&pkField); err != nil {
		t.Fatal(err)
	}
	// The data UPDATE wrote a new version of every large-type row: start the
	// phase from P3-36's layout, as TICK-1039's arms did. This is the last
	// VACUUM FULL; everything after it churns the heap the way writes do.
	e.exec(t, "VACUUM FULL records")
	e.exec(t, "VACUUM (ANALYZE) records")
	e.exec(t, "CHECKPOINT")

	// Patch targets, all with pk 4 (the value no cell filters on): one set of
	// clients x 500 per live run, and clients x 200 per write arm set.
	var pool4 []uuid.UUID
	rows, err := e.super.Query(ctx, `SELECT id FROM sl_rec WHERE v = 4 ORDER BY md5(id::text || 'slots')`)
	if err != nil {
		t.Fatal(err)
	}
	if pool4, err = pgx.CollectRows(rows, pgx.RowTo[uuid.UUID]); err != nil {
		t.Fatal(err)
	}
	take := func(per int) [][]uuid.UUID {
		var out [][]uuid.UUID
		for c := 0; c < clients; c++ {
			out = append(out, pool4[:per])
			pool4 = pool4[per:]
		}
		return out
	}
	flat := func(s [][]uuid.UUID) []uuid.UUID { return slices.Concat(s...) }
	live := [4][][]uuid.UUID{take(500), take(500), take(500), take(500)}
	tset := map[int][][]uuid.UUID{0: take(200), 1: take(200), 3: take(200), 5: take(200)}
	tCurve := take(200)
	ver := &slVer{v: map[uuid.UUID]int64{}}
	creates := e.fiCreateReqs(t)

	sl := &slState{}
	e.h = e.handlerWith(t, server.WithListIndexTerm(sl.term), server.WithListQuery(sl.query))
	e.w("- fixture (pk into data, VACUUM FULL, targets) in %s; records: %s", time.Since(t0).Round(time.Second),
		e.scalar(t, `SELECT pg_size_pretty(pg_table_size('records')) || ' heap, ' || pg_size_pretty(pg_indexes_size('records')) || ' indexes, ' || (SELECT count(*) FROM pg_indexes WHERE tablename = 'records') || ' indexes'`))
	e.floorGate(t, clients, "before")

	// ---- item 3 (A3-2 migration) under the live mix ----
	e.w("")
	e.w("### Item 3: the A3-2 migration under the live mix (%d clients, POST + PATCH score as nava_app)", clients)
	e.w("")
	var cols []string
	for _, c := range slA3() {
		cols = append(cols, "ADD COLUMN "+c.col+" "+c.sqlType)
	}
	e.exec(t, `CREATE TABLE spike_slot_map (tenant_id uuid NOT NULL, entity_definition_id uuid NOT NULL, col text NOT NULL,
		field_key text NOT NULL, value_type text NOT NULL, state text NOT NULL, PRIMARY KEY (tenant_id, entity_definition_id, col))`)
	e.exec(t, "GRANT SELECT ON spike_slot_map TO nava_app, nava_worker")
	extra := perfEnvList("NAVA_PERF_SLOT_CURVE", []int{0, 8, 16, 32, 64})
	t.Cleanup(func() {
		c := context.Background()
		_, _ = e.super.Exec(c, "DROP TRIGGER IF EXISTS spike_slot_sync ON records")
		_, _ = e.super.Exec(c, "DROP FUNCTION IF EXISTS spike_slot_sync()")
		_, _ = e.super.Exec(c, "DROP TABLE IF EXISTS spike_slot_map")
		for _, s := range slA3() {
			_, _ = e.super.Exec(c, "ALTER TABLE records DROP COLUMN IF EXISTS "+s.col)
		}
		_, _ = e.super.Exec(c, "ALTER TABLE records DROP COLUMN IF EXISTS fx_own")
		for i := 1; i <= slices.Max(extra); i++ {
			_, _ = e.super.Exec(c, fmt.Sprintf("ALTER TABLE records DROP COLUMN IF EXISTS fx_c_%d", i))
		}
	})
	ctl := e.startLive(t, clients, creates, live[0])
	time.Sleep(liveD + 10*time.Second)
	var ddls []slDDL
	ddls = append(ddls, e.ddl(t, "ALTER TABLE records ADD COLUMN x 8 (one statement)", "ALTER TABLE records "+strings.Join(cols, ", ")))
	time.Sleep(2 * time.Second)
	for _, c := range slA3() {
		ddls = append(ddls, e.ddl(t, "CREATE STATISTICS (mcv) "+c.col, fmt.Sprintf("CREATE STATISTICS spike_%s_stx (mcv) ON tenant_id, entity_definition_id, %s FROM records", c.col, c.col)))
	}
	e.exec(t, slFuncSQL(slA3()))
	ddls = append(ddls, e.ddl(t, "CREATE TRIGGER spike_slot_sync (BEFORE INSERT OR UPDATE OF data)",
		"CREATE TRIGGER spike_slot_sync BEFORE INSERT OR UPDATE OF data ON records FOR EACH ROW EXECUTE FUNCTION spike_slot_sync()"))
	time.Sleep(2 * time.Second)
	for _, c := range slA3() {
		ddls = append(ddls, e.ddl(t, "CREATE INDEX CONCURRENTLY "+slGroupIdx(c.col)+" (all NULL)",
			fmt.Sprintf("CREATE INDEX CONCURRENTLY %s ON records (tenant_id, entity_definition_id, %s, assignment_group_id, updated_at) WHERE %s IS NOT NULL", slGroupIdx(c.col), c.col, c.col)))
	}
	migEnd := time.Now()
	// The per-field form arm (b) literally measured: one more column, for
	// comparison, dropped at once.
	ddls = append(ddls, e.ddl(t, "ALTER TABLE records ADD COLUMN (one per-field column, arm (b) literally)", "ALTER TABLE records ADD COLUMN spike_pf double precision"))
	e.exec(t, "ALTER TABLE records DROP COLUMN spike_pf")
	time.Sleep(liveD)
	ctl.stop()
	before := fiSummarize(ctl.ops, ctl.start.Add(10*time.Second), ctl.start.Add(10*time.Second+liveD))
	during := fiSummarize(ctl.ops, ddls[0].start, migEnd)
	after := fiSummarize(ctl.ops, migEnd.Add(time.Second), migEnd.Add(time.Second+liveD))
	e.w("%s", slDDLHead)
	for _, d := range ddls {
		e.w("%s", e.ddlRow(d, ctl.ops))
	}
	e.w("")
	e.w("| Live mix window | Length | ops/s | p50 ms | **p95 ms** | p99 ms | create p95 | patch p95 |\n| --- | --- | --- | --- | --- | --- | --- | --- |")
	e.w("%s", before.row("before the migration"))
	e.w("%s", during.row("during the migration (first ALTER to last CIC)"))
	e.w("%s", after.row("after"))
	e.w("- live errors: %v", ctl.errs)
	e.w("- records after the migration: %s", e.scalar(t, `SELECT pg_size_pretty(pg_table_size('records')) || ' heap, ' || pg_size_pretty(pg_indexes_size('records')) || ' indexes; slot indexes: ' ||
		(SELECT string_agg(c.relname || ' ' || pg_size_pretty(pg_relation_size(c.oid)), ', ' ORDER BY c.relname) FROM pg_class c WHERE c.relname LIKE 'spike\_fx\_%\_grp')`))

	// ---- A3-5 backfill (R-e) ----
	e.w("")
	e.w("### A3-5 backfill: 1,000-record batches, UPDATE records SET slot = record_field_coerce(data -> key) (no version or updated_at change)")
	e.w("")
	e.w("| Field -> slot | Load | Duration | Batches | Batch p50 / p95 / max ms | WAL | WAL B/record | FPI | records HOT / updates | Heap before -> after | Slot index after | n_dead_tup after |\n| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |")
	var bfLive *liveRun
	var bfRows []string
	var bfWin [2]time.Time
	for i, f := range []string{"score", "tier", "opened", "zone", "b2", "pk"} {
		col := slFields()[f]
		e.exec(t, `INSERT INTO spike_slot_map VALUES ($1, $2, $3, $4, $5, 'backfilling')`, e.large.id, e.largeDef, col, f, slVTs()[f])
		pause, load := time.Duration(0), "alone"
		if i == 0 {
			// The first runs under the live mix, with D10's 500 ms floor pause.
			pause, load = 500*time.Millisecond, "live mix, 500 ms pause"
			if quick {
				pause = 0
			}
			bfLive = e.startLive(t, clients, creates, live[1])
			time.Sleep(liveD + 10*time.Second)
		}
		time.Sleep(slStatsWait)
		s0 := e.slStats(t)
		heap0 := e.scalar(t, `SELECT pg_size_pretty(pg_table_size('records'))`)
		bs := time.Now()
		batches := e.slBackfill(t, col, f, slVTs()[f], e.large.id, e.largeDef, pause)
		be := time.Now()
		wal := e.walSince(t, s0.lsn)
		if i == 0 {
			bfWin = [2]time.Time{bs, be}
			time.Sleep(liveD)
			bfLive.stop()
		}
		time.Sleep(slStatsWait)
		s1 := e.slStats(t)
		e.exec(t, `UPDATE spike_slot_map SET state = 'ready' WHERE entity_definition_id = $1 AND col = $2`, e.largeDef, col)
		bms := slices.Clone(batches)
		slices.Sort(bms)
		n := e.scalar(t, `SELECT count(*) FROM records WHERE tenant_id = $1 AND entity_definition_id = $2 AND `+col+` IS NOT NULL`, e.large.id, e.largeDef)
		nRows, _ := strconv.ParseFloat(e.scalar(t, `SELECT count(*) FROM records WHERE tenant_id = $1 AND entity_definition_id = $2`, e.large.id, e.largeDef), 64)
		bfRows = append(bfRows, fmt.Sprintf("| %s -> %s (%s filled) | %s | %s | %d | %.0f / %.0f / %.0f | %s | %.0f | %d | %d / %d | %s -> %s | %s | %s |", f, col, n, load,
			be.Sub(bs).Round(time.Second), len(bms), pct(bms, .5), pct(bms, .95), fiMS(slices.Max(bms)), slBytes(wal), float64(wal)/nRows,
			s1.fpi-s0.fpi, s1.hot-s0.hot, s1.upd-s0.upd, heap0, e.scalar(t, `SELECT pg_size_pretty(pg_table_size('records'))`),
			e.scalar(t, `SELECT pg_size_pretty(pg_relation_size($1::regclass))`, slGroupIdx(col)),
			e.scalar(t, `SELECT n_dead_tup FROM pg_stat_user_tables WHERE relid = 'records'::regclass`)))
		e.w("%s", bfRows[len(bfRows)-1])
	}
	bfBefore := fiSummarize(bfLive.ops, bfLive.start.Add(10*time.Second), bfLive.start.Add(10*time.Second+liveD))
	bfDuring := fiSummarize(bfLive.ops, bfWin[0], bfWin[1])
	bfAfter := fiSummarize(bfLive.ops, bfWin[1].Add(time.Second), bfWin[1].Add(time.Second+liveD))
	e.w("")
	e.w("Live mix around the score backfill (D10: p95 within +25%%):")
	e.w("")
	e.w("| Live mix window | Length | ops/s | p50 ms | **p95 ms** | p99 ms | create p95 | patch p95 |\n| --- | --- | --- | --- | --- | --- | --- | --- |")
	e.w("%s", bfBefore.row("alone, before"))
	e.w("%s", bfDuring.row("with the backfill"))
	e.w("%s", bfAfter.row("alone, after"))
	e.w("- D10 delta: %+.1f%% against before, %+.1f%% against after (budget +25%%); live errors %v", 100*(bfDuring.p95/bfBefore.p95-1), 100*(bfDuring.p95/bfAfter.p95-1), bfLive.errs)
	e.w("- autovacuum on records so far: %s", e.scalar(t, `SELECT 'vacuums ' || autovacuum_count || ', analyzes ' || autoanalyze_count || ', last ' || coalesce(last_autovacuum::text, 'never') FROM pg_stat_user_tables WHERE relid = 'records'::regclass`))
	// What autovacuum would do after a backfill (no VACUUM FULL).
	vac := e.timed(t, "VACUUM (ANALYZE) records")
	e.w("- VACUUM (ANALYZE) records after the backfills: %s; records %s", vac.Round(time.Second), e.scalar(t, `SELECT pg_size_pretty(pg_table_size('records')) || ' heap, ' || pg_size_pretty(pg_indexes_size('records')) || ' indexes'`))

	// ---- item 2: sizes ----
	e.w("")
	e.w("### Item 2: size per promoted field")
	e.w("")
	e.w("| Slot (field) | Filled rows | Value bytes in heap (sum of pg_column_size) | Group index (as grown by the backfill) | Fresh build, group key | Fresh build, plain key |\n| --- | --- | --- | --- | --- | --- |")
	for _, f := range []string{"score", "tier", "opened", "zone", "b2", "pk"} {
		col := slFields()[f]
		g, p := e.freshSizes(t, col, "")
		e.w("| %s (%s) | %s | %s | %s | %s | %s |", col, f, e.scalar(t, `SELECT count(`+col+`) FROM records`), e.scalar(t, `SELECT pg_size_pretty(sum(pg_column_size(`+col+`)))  FROM records`),
			e.scalar(t, `SELECT pg_size_pretty(pg_relation_size($1::regclass))`, slGroupIdx(col)), g, p)
	}

	// ---- item 1, group index only ----
	fieldIDs := map[string]uuid.UUID{}
	rows, err = e.super.Query(ctx, `SELECT field_key, id FROM record_index_fields WHERE entity_definition_id = $1`, e.largeDef)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var k string
		var id uuid.UUID
		if err := rows.Scan(&k, &id); err != nil {
			t.Fatal(err)
		}
		fieldIDs[k] = id
	}
	rows.Close()
	// Every target set starts with no slot value; arm n then fills its own
	// set's n slots, as a promoted record of that type would carry.
	var setCols []string
	for _, c := range slA3() {
		setCols = append(setCols, c.col+" = NULL")
	}
	all := slices.Concat(flat(tset[0]), flat(tset[1]), flat(tset[3]), flat(tset[5]), flat(tCurve))
	e.exec(t, `UPDATE records SET `+strings.Join(setCols, ", ")+` WHERE tenant_id = $1 AND id = ANY($2)`, e.large.id, all)
	for _, n := range []int{1, 3, 5} {
		var fill []string
		for _, f := range slSets()[n] {
			fill = append(fill, fmt.Sprintf("%s = (SELECT c.value_%s FROM record_field_coerce(data -> '%s', '%s') c ORDER BY c.ord LIMIT 1)", slFields()[f], slVTs()[f], f, slVTs()[f]))
		}
		e.exec(t, `UPDATE records SET `+strings.Join(fill, ", ")+` WHERE tenant_id = $1 AND id = ANY($2)`, e.large.id, flat(tset[n]))
	}
	e.exec(t, "VACUUM (ANALYZE) records")
	var writes []slWrite
	arm := func(label string, n int, slot bool) {
		if quick {
			return
		}
		e.exec(t, `UPDATE record_index_fields SET state = CASE WHEN field_key = ANY($2) THEN 'ready' ELSE 'removing' END
			WHERE entity_definition_id = $1 AND field_key = ANY($3)`, e.largeDef, slSets()[n], fiWriteFields())
		var mapped []string
		if slot {
			for _, f := range slSets()[n] {
				mapped = append(mapped, slFields()[f])
			}
		}
		e.exec(t, `UPDATE spike_slot_map SET state = CASE WHEN col = ANY($2) THEN 'ready' ELSE 'removing' END WHERE entity_definition_id = $1`, e.largeDef, mapped)
		targets := tset[0]
		if slot {
			targets = tset[n]
		}
		sc := perfScenario{name: "create, " + label, kind: "create", tenant: "large", reqs: creates}
		writes = append(writes, e.slMeasure(t, label, "insert", writeD, func() ([]time.Duration, int64, int) {
			r := e.run(t, sc, clients, writeD)
			return r.lat, r.errs, min(50, len(creates)) + 1
		}))
		e.w("%s", writes[len(writes)-1].row())
		writes = append(writes, e.slMeasure(t, label, "update promoted field (score)", writeD, func() ([]time.Duration, int64, int) {
			l, er := e.patchRun(t, writeD, targets, ver, func(v int) string { return fmt.Sprintf(`{"data": {"score": %d.75}}`, (v*7919)%10000) })
			return l, er, 0
		}))
		e.w("%s", writes[len(writes)-1].row())
		writes = append(writes, e.slMeasure(t, label, "update unrelated field (b001)", writeD, func() ([]time.Duration, int64, int) {
			l, er := e.patchRun(t, writeD, targets, ver, func(v int) string { return fmt.Sprintf(`{"data": {"b001": %t}}`, v%2 == 0) })
			return l, er, 0
		}))
		e.w("%s", writes[len(writes)-1].row())
	}
	e.w("")
	e.w("### Item 1: write amplification, POST and PATCH through the handler, %d clients, %s per cell", clients, writeD)
	e.w("")
	e.w("D7 n: n of (score, tier, opened, zone, b2) flagged (side rows), no slot. Slot n: the same n flagged and mapped to slots (A3-5 keeps the side rows), each slot with its index. An update with n = 0 still names score, which is then neither flagged nor mapped. The PATCH targets of a slot arm carry exactly its n slot values.")
	e.w("")
	e.w("%s", slWriteHead)
	time.Sleep(slStatsWait)
	for _, n := range []int{0, 1, 3, 5} {
		arm(fmt.Sprintf("D7 %d", n), n, false)
	}
	for _, n := range []int{1, 3, 5} {
		arm(fmt.Sprintf("slot %d, group index", n), n, true)
	}

	// ---- S-b2 (i) and the no-ACL S-c2 cells on the group index only ----
	e.exec(t, `UPDATE record_index_fields SET state = 'ready' WHERE entity_definition_id = $1`, e.largeDef)
	e.exec(t, `UPDATE spike_slot_map SET state = 'ready' WHERE entity_definition_id = $1`, e.largeDef)
	e.exec(t, "ANALYZE records")
	slotCols := map[uuid.UUID]string{}
	for f, col := range slFields() {
		if id, ok := fieldIDs[f]; ok {
			slotCols[id] = col
		}
	}
	var plans []string
	// Plans are written when the phase ends, however it ends.
	defer func() { e.writePlans(t, "slots", "TICK-1205 slot plans", plans) }()
	type q struct{ name, q string }
	pk := func(v string) string { return "filter%5Bpk%5D=" + v }
	sc2 := []q{
		{"num lt (score < 200, 2%)", "filter%5Bscore%5D%5Blt%5D=200"},
		{"num gte (score >= 9800, 2%)", "filter%5Bscore%5D%5Bgte%5D=9800"},
		{"num range (2000 <= score < 2200, 2%)", "filter%5Bscore%5D%5Bgte%5D=2000&filter%5Bscore%5D%5Blt%5D=2200"},
		{"text eq (tier v2, 2%)", "filter%5Btier%5D=v2"},
		{"text in 3 values (tier v2, v025, v001, 2.26%)", "filter%5Btier%5D%5Bin%5D=v2,v025,v001"},
		{"num in 3 values (pk 10, 20, 30, 17%)", "filter%5Bpk%5D%5Bin%5D=10,20,30"},
	}
	var verdicts []string
	score := func(c spCell, line float64) {
		verdicts = append(verdicts, fmt.Sprintf("%s %.1f: %s", c.label, c.p95, spVerdict(c.p95, line)))
	}
	sl.set(slotCols, false)
	e.w("")
	e.w("### S-b2 (i) and S-c2 without an ACL: the group index only (A3-2's set), line 150 ms")
	e.w("")
	e.w("%s", spHead)
	sb2i := e.cell(t, clients, listD, "S-b2 (i) broad 50% (pk 3), no ACL, group index only", []string{"admin-large"}, pk("3"), &plans)
	e.w("%s", sb2i.row(fiListTargetMS))
	score(sb2i, fiListTargetMS)
	for _, c := range sc2 {
		if quick {
			break
		}
		r := e.cell(t, clients, listD, "S-c2 "+c.name+", no ACL, group index only", []string{"admin-large"}, c.q, &plans)
		e.w("%s", r.row(fiListTargetMS))
		score(r, fiListTargetMS)
	}
	if !quick {
		// ---- plain indexes; item 1 with both ----
		e.w("")
		e.w("### Item 3: the plain index, CREATE INDEX CONCURRENTLY on a filled slot")
		e.w("")
		plainLive := e.startLive(t, clients, creates, live[2])
		time.Sleep(liveD + 10*time.Second)
		var pd []slDDL
		pd = append(pd, e.ddl(t, "CREATE INDEX CONCURRENTLY "+slPlainIdx("fx_num_1")+" (400k filled), live mix",
			fmt.Sprintf("CREATE INDEX CONCURRENTLY %s ON records (tenant_id, entity_definition_id, fx_num_1, updated_at) WHERE fx_num_1 IS NOT NULL", slPlainIdx("fx_num_1"))))
		// The same build of the group key, for (b) literally: a new field's
		// index on a filled column.
		pd = append(pd, e.ddl(t, "CREATE INDEX CONCURRENTLY group key on fx_num_1 (400k filled, a second copy), live mix",
			"CREATE INDEX CONCURRENTLY spike_fx_num_1_grp2 ON records (tenant_id, entity_definition_id, fx_num_1, assignment_group_id, updated_at) WHERE fx_num_1 IS NOT NULL"))
		pdEnd := time.Now()
		e.exec(t, "DROP INDEX spike_fx_num_1_grp2")
		time.Sleep(liveD)
		plainLive.stop()
		pBefore := fiSummarize(plainLive.ops, plainLive.start.Add(10*time.Second), plainLive.start.Add(10*time.Second+liveD))
		pDuring := fiSummarize(plainLive.ops, pd[0].start, pdEnd)
		for _, f := range []string{"tier", "opened", "zone", "b2", "pk"} {
			col := slFields()[f]
			pd = append(pd, e.ddl(t, "CREATE INDEX CONCURRENTLY "+slPlainIdx(col)+" (400k filled), alone",
				fmt.Sprintf("CREATE INDEX CONCURRENTLY %s ON records (tenant_id, entity_definition_id, %s, updated_at) WHERE %s IS NOT NULL", slPlainIdx(col), col, col)))
		}
		e.w("%s", slDDLHead)
		for _, d := range pd {
			e.w("%s", e.ddlRow(d, plainLive.ops))
		}
		e.w("")
		e.w("| Live mix window | Length | ops/s | p50 ms | **p95 ms** | p99 ms | create p95 | patch p95 |\n| --- | --- | --- | --- | --- | --- | --- | --- |")
		e.w("%s", pBefore.row("before the builds"))
		e.w("%s", pDuring.row("during the two builds"))
		e.w("- live errors: %v", plainLive.errs)
		e.exec(t, "ANALYZE records")
		e.w("")
		e.w("### Item 1 with both indexes (group + plain) per promoted field")
		e.w("")
		e.w("%s", slWriteHead)
		time.Sleep(slStatsWait)
		for _, n := range []int{1, 3, 5} {
			arm(fmt.Sprintf("slot %d, group + plain", n), n, true)
		}
		e.exec(t, `UPDATE record_index_fields SET state = 'ready' WHERE entity_definition_id = $1`, e.largeDef)
		e.exec(t, `UPDATE spike_slot_map SET state = 'ready' WHERE entity_definition_id = $1`, e.largeDef)
		e.exec(t, "ANALYZE records")
		e.w("")
		e.w("### Item 1 summary: p95 ms and WAL B/op against D7 0 (the same writes, nothing flagged or mapped)")
		e.w("")
		e.w("| Write | D7 0 | D7 1 | D7 3 | D7 5 | slot 1, group | slot 3, group | slot 5, group | slot 1, both | slot 3, both | slot 5, both |\n| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |")
		for _, op := range []string{"insert", "update promoted field (score)", "update unrelated field (b001)"} {
			row := "| " + op + " |"
			for _, w := range writes {
				if w.op == op {
					row += fmt.Sprintf(" %.1f / %.0f |", w.p95, w.walPerOp)
				}
			}
			e.w("%s", row)
		}

		e.w("")
		e.w("### S-b2 (ii) and S-c2 without an ACL: group + plain indexes, line 150 ms")
		e.w("")
		e.w("%s", spHead)
		sb2ii := e.cell(t, clients, listD, "S-b2 (ii) broad 50% (pk 3), no ACL, group + plain", []string{"admin-large"}, pk("3"), &plans)
		e.w("%s", sb2ii.row(fiListTargetMS))
		score(sb2ii, fiListTargetMS)
		for _, c := range sc2 {
			r := e.cell(t, clients, listD, "S-c2 "+c.name+", no ACL, group + plain", []string{"admin-large"}, c.q, &plans)
			e.w("%s", r.row(fiListTargetMS))
			score(r, fiListTargetMS)
		}
		// The ACL cells run on A3-2's set: the group index alone.
		for _, f := range []string{"score", "tier", "opened", "zone", "b2", "pk"} {
			e.exec(t, "DROP INDEX "+slPlainIdx(slFields()[f]))
		}
		e.exec(t, "ANALYZE records")
	}

	e.w("- churn before the ACL cells (no VACUUM FULL since the fixture): %s", e.churn(t))

	// ---- S-b1: compiled shape vs per-group merge ----
	e.w("")
	e.w("### S-b1: per-group merge page (one ordered arm per visible group), group index only, line 150 ms")
	e.w("")
	e.w("Pass: no arm's scan of the slot index reads more than 51 + epsilon rows (taken as 102), no full Sort over more than 1,000 rows, p95 <= 150 ms.")
	e.w("")
	e.w("%s", spHead)
	var sb1 []string
	var hf []string
	for _, fx := range []struct{ name, v string }{{"broad 50% (pk 3)", "3"}, {"anti-correlated 2% (pk 40)", "40"}} {
		for _, ag := range []struct{ name, tok string }{{"a6", "agent-1"}, {"a25", "a25-1"}, {"a50", "a50-1"}} {
			for _, split := range []bool{false, true} {
				shape := "compiled (b)+group"
				if split {
					shape = "per-group merge"
				}
				sl.set(slotCols, split)
				r := e.cell(t, clients, listD, fmt.Sprintf("S-b1 %s, %s, %s", fx.name, ag.name, shape), []string{ag.tok}, pk(fx.v), &plans)
				sl.mu.Lock()
				splits, skipped := sl.splits, sl.skipped
				sl.mu.Unlock()
				e.w("%s", r.row(fiListTargetMS))
				ms, si, scans := slPageShape(r.pagePlan, slGroupIdx("fx_num_2"))
				v := "pass"
				if r.p95 > fiListTargetMS || (split && (ms > 102 || si > 1000)) {
					v = "**fail**"
				}
				if !split {
					v = "control"
				}
				if split && splits == 0 {
					v = "**not split**: " + strings.Join(skipped[:min(len(skipped), 2)], "; ")
				}
				sb1 = append(sb1, fmt.Sprintf("| %s | %s | %s | %.1f | %d | %d | %d | %s |", fx.name, ag.name, shape, r.p95, scans, ms, si, v))
				hf = append(hf, fmt.Sprintf("| %s | %d | %d |", r.label, slHeapFetches(r.countPlan), slHeapFetches(r.pagePlan)))
			}
		}
	}
	sl.set(slotCols, false)
	e.w("")
	e.w("| Fixture | Agent | Page shape | p95 ms | slot-index scans in the page | max rows one scan read | max full-Sort input | S-b1 |\n| --- | --- | --- | --- | --- | --- | --- | --- |")
	for _, r := range sb1 {
		e.w("%s", r)
	}
	e.w("")
	e.w("Heap fetches under churn (count / page plan, one EXPLAIN ANALYZE each):")
	e.w("")
	e.w("| Cell | count Heap Fetches | page Heap Fetches |\n| --- | --- | --- |")
	for _, r := range hf {
		e.w("%s", r)
	}

	// ---- S-c2 and S-c3 with an ACL ----
	e.w("")
	e.w("### S-c2 (operators and types) and S-c3 (10%% and 20%%) with an ACL, compiled shape and per-group merge, line 150 ms")
	e.w("")
	e.w("%s", spHead)
	sc3 := []q{
		{"S-c3 num lt 10% (score < 1000)", "filter%5Bscore%5D%5Blt%5D=1000"},
		{"S-c3 num lt 20% (score < 2000)", "filter%5Bscore%5D%5Blt%5D=2000"},
		{"S-c3 text eq 10% (tier v10)", "filter%5Btier%5D=v10"},
		{"S-c3 text eq 20% (tier v20)", "filter%5Btier%5D=v20"},
	}
	var aclCells []q
	for _, c := range sc2 {
		aclCells = append(aclCells, q{"S-c2 " + c.name, c.q})
	}
	aclCells = append(aclCells, sc3...)
	for _, c := range aclCells {
		if quick {
			break
		}
		for _, ag := range []struct{ name, tok string }{{"a6", "agent-1"}, {"a50", "a50-1"}} {
			for _, split := range []bool{false, true} {
				shape := "compiled"
				if split {
					shape = "per-group merge"
				}
				sl.set(slotCols, split)
				r := e.cell(t, clients, listD, fmt.Sprintf("%s, %s, %s", c.name, ag.name, shape), []string{ag.tok}, c.q, &plans)
				e.w("%s", r.row(fiListTargetMS))
				score(r, fiListTargetMS)
			}
		}
	}
	sl.set(slotCols, false)

	// ---- S-c1: a shared slot ----
	e.sharedSlot(t, clients, listD, sl, slotCols, &plans, score)

	// ---- item 4: how many slots records carries ----
	e.carry(t, clients, listD, writeD, sl, slotCols, creates, tCurve, ver, extra, &plans)

	// ---- S-b3: 2M records of one type ----
	e.scale(t, clients, listD, sl, slotCols, &plans, score)

	e.floorGate(t, clients, "after")
	e.w("")
	e.w("**Scores (line 150 ms, provisional within 10%%):** %s.", strings.Join(verdicts, "; "))
}

// slFuncSQL is the scratch A3-5 trigger function over the slot columns cols.
func slFuncSQL(cols []slCol) string {
	var b strings.Builder
	b.WriteString(`CREATE OR REPLACE FUNCTION spike_slot_sync() RETURNS trigger
    LANGUAGE plpgsql SECURITY INVOKER
    SET search_path = pg_catalog, public
AS $f$
DECLARE
    m record;
    v jsonb;
BEGIN
    FOR m IN
        SELECT s.col, s.field_key FROM spike_slot_map s
        WHERE s.tenant_id = NEW.tenant_id AND s.entity_definition_id = NEW.entity_definition_id
          AND s.state IN ('backfilling', 'ready')
    LOOP
        v := NEW.data -> m.field_key;
        CASE m.col
`)
	for _, c := range cols {
		fmt.Fprintf(&b, "        WHEN '%s' THEN NEW.%s := (SELECT c.value_%s FROM record_field_coerce(v, '%s') c ORDER BY c.ord LIMIT 1);\n", c.col, c.col, c.vt, c.vt)
	}
	b.WriteString(`        ELSE NULL;
        END CASE;
    END LOOP;
    RETURN NEW;
END
$f$`)
	return b.String()
}

// slBackfill copies field f into col for one type in 1,000-record batches,
// each its own transaction, pausing between batches. It returns each
// batch's duration.
func (e *fiEnv) slBackfill(t *testing.T, col, f, vt string, tenant, def uuid.UUID, pause time.Duration) []time.Duration {
	t.Helper()
	ctx := t.Context()
	var out []time.Duration
	last := uuid.Nil
	for {
		t0 := time.Now()
		var ids []uuid.UUID
		rows, err := e.super.Query(ctx, `SELECT id FROM records WHERE tenant_id = $1 AND entity_definition_id = $2 AND id > $3 ORDER BY id LIMIT 1000`, tenant, def, last)
		if err != nil {
			t.Fatal(err)
		}
		if ids, err = pgx.CollectRows(rows, pgx.RowTo[uuid.UUID]); err != nil {
			t.Fatal(err)
		}
		if len(ids) == 0 {
			return out
		}
		e.exec(t, fmt.Sprintf(`UPDATE records r SET %s = (SELECT c.value_%s FROM record_field_coerce(r.data -> '%s', '%s') c ORDER BY c.ord LIMIT 1)
			WHERE r.tenant_id = $1 AND r.id = ANY($2)`, col, vt, f, vt), tenant, ids)
		out = append(out, time.Since(t0))
		last = ids[len(ids)-1]
		time.Sleep(pause)
	}
}

// freshSizes builds the group and plain keys on col (scoped by where, if
// set) and returns their sizes, then drops them.
func (e *fiEnv) freshSizes(t *testing.T, col, where string) (string, string) {
	t.Helper()
	cond := col + " IS NOT NULL"
	if where != "" {
		cond += " AND " + where
	}
	e.exec(t, fmt.Sprintf("CREATE INDEX sl_tmp_g ON records (tenant_id, entity_definition_id, %s, assignment_group_id, updated_at) WHERE %s", col, cond))
	e.exec(t, fmt.Sprintf("CREATE INDEX sl_tmp_p ON records (tenant_id, entity_definition_id, %s, updated_at) WHERE %s", col, cond))
	g := e.scalar(t, `SELECT pg_size_pretty(pg_relation_size('sl_tmp_g'))`)
	p := e.scalar(t, `SELECT pg_size_pretty(pg_relation_size('sl_tmp_p'))`)
	e.exec(t, "DROP INDEX sl_tmp_g")
	e.exec(t, "DROP INDEX sl_tmp_p")
	return g, p
}

func (e *fiEnv) churn(t *testing.T) string {
	t.Helper()
	return e.scalar(t, `SELECT 'n_dead_tup ' || s.n_dead_tup || ', n_live_tup ' || s.n_live_tup || ', updates since the fixture ' || s.n_tup_upd || ' (HOT ' || s.n_tup_hot_upd ||
		'), autovacuums ' || s.autovacuum_count || ', relallvisible ' || c.relallvisible || ' of ' || c.relpages || ' pages'
		FROM pg_stat_user_tables s JOIN pg_class c ON c.oid = s.relid WHERE s.relid = 'records'::regclass`)
}

func slBytes(n int64) string {
	switch {
	case n >= 1<<30:
		return fmt.Sprintf("%.2f GB", float64(n)/(1<<30))
	case n >= 1<<20:
		return fmt.Sprintf("%.0f MB", float64(n)/(1<<20))
	}
	return fmt.Sprintf("%.0f kB", float64(n)/(1<<10))
}

func perfEnvList(name string, def []int) []int {
	v := strings.TrimSpace(os.Getenv(name))
	if v == "" {
		return def
	}
	var out []int
	for _, p := range strings.Split(v, ",") {
		n, err := strconv.Atoi(strings.TrimSpace(p))
		if err == nil && n >= 0 {
			out = append(out, n)
		}
	}
	if len(out) == 0 {
		return def
	}
	return out
}

// ---- S-c1 -------------------------------------------------------------------

// slEstimate is the planner's row estimate against the actual count for one
// predicate on records, as the superuser with literal values (the custom
// plan nava_app gets under force_custom_plan sees the same constants).
func (e *fiEnv) slEstimate(t *testing.T, where string) (est, act int64) {
	t.Helper()
	var plan string
	if err := e.super.QueryRow(t.Context(), `EXPLAIN (FORMAT JSON) SELECT 1 FROM records WHERE `+where).Scan(&plan); err != nil {
		t.Fatalf("estimate %s: %v", where, err)
	}
	m := regexp.MustCompile(`"Plan Rows": (\d+)`).FindStringSubmatch(plan)
	est, _ = strconv.ParseInt(m[1], 10, 64)
	if err := e.super.QueryRow(t.Context(), `SELECT count(*) FROM records WHERE `+where).Scan(&act); err != nil {
		t.Fatal(err)
	}
	return est, act
}

func slRatio(est, act int64) (float64, string) {
	a, b := float64(max(est, 1)), float64(max(act, 1))
	r := max(a/b, b/a)
	if r > 10 {
		return r, "**fail**"
	}
	return r, "pass"
}

func (e *fiEnv) sharedSlot(t *testing.T, clients int, d time.Duration, sl *slState, slotCols map[uuid.UUID]string, plans *[]string, score func(spCell, float64)) {
	t.Helper()
	ctx := t.Context()
	e.w("")
	e.w("### S-c1: a shared slot (R-d)")
	e.w("")
	// A second type in the large tenant (ticket2), 100k records copied from
	// the large type, pk uniform over 0..49 (2% each). Its own-slot control
	// is fx_own, filled for ticket2 only.
	body := strings.Replace(strings.Replace(perfTypeBodyWith(e.schemaExtra), `"key": "ticket"`, `"key": "ticket2"`, 1), `"number_prefix": "TKT"`, `"number_prefix": "TKB"`, 1)
	if rec := e.serve(ctx, http.MethodPost, "/api/v1/entity-types", "admin-large", uuid.NewString(), body); rec.Code != http.StatusCreated {
		t.Fatalf("create ticket2 = %d: %s", rec.Code, rec.Body)
	}
	var agents uuid.UUID
	if err := e.super.QueryRow(ctx, `SELECT agents FROM perf_tenant WHERE tn = 0`).Scan(&agents); err != nil {
		t.Fatal(err)
	}
	if err := tenancy.InTenantTx(ctx, e.pool, e.large.id, func(tx pgx.Tx) error {
		_, err := acltest.Publish(ctx, tx, "ticket2", perfACL(agents, e.large.adminGroup)...)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	var def2 uuid.UUID
	if err := e.super.QueryRow(ctx, `SELECT id FROM entity_definitions WHERE tenant_id = $1 AND key = 'ticket2'`, e.large.id).Scan(&def2); err != nil {
		t.Fatal(err)
	}
	e.exec(t, "ALTER TABLE records ADD COLUMN fx_own double precision")
	t0 := time.Now()
	colList, sel := e.copyColumns(t, map[string]string{
		"id": "md5('t2/' || r.id::text)::uuid", "entity_definition_id": "$2::uuid", "number": "'TKB' || substr(r.number, 4)",
		"data":     "r.data || jsonb_build_object('pk', (('x' || substr(md5(r.id::text || 't2'), 1, 7))::bit(28)::int) % 50)",
		"fx_num_2": "NULL", "fx_own": "((('x' || substr(md5(r.id::text || 't2'), 1, 7))::bit(28)::int) % 50)::float8",
	})
	e.exec(t, "SET session_replication_role = replica")
	e.exec(t, `INSERT INTO records (`+colList+`) SELECT `+sel+` FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $3
		ORDER BY md5(r.id::text || 't2') LIMIT 100000`, e.large.id, def2, e.largeDef)
	e.exec(t, "SET session_replication_role = origin")
	var pk2 uuid.UUID
	if err := e.super.QueryRow(ctx, `INSERT INTO record_index_fields (tenant_id, id, entity_definition_id, field_key, value_type, state, progress, requested_version)
		VALUES ($1, gen_random_uuid(), $2, 'pk', 'num', 'ready', 100, 1) RETURNING id`, e.large.id, def2).Scan(&pk2); err != nil {
		t.Fatal(err)
	}
	e.exec(t, "CREATE INDEX CONCURRENTLY spike_fx_own_grp ON records (tenant_id, entity_definition_id, fx_own, assignment_group_id, updated_at) WHERE fx_own IS NOT NULL")
	e.exec(t, "CREATE STATISTICS spike_fx_own_stx (mcv) ON tenant_id, entity_definition_id, fx_own FROM records")
	e.exec(t, "VACUUM (ANALYZE) records")
	e.w("- ticket2: 100k records copied in %s; a6 sees %s of them", time.Since(t0).Round(time.Second),
		e.scalar(t, `SELECT count(*) FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND `+spVisible, e.large.id, def2, e.tokens["agent-1"].PrincipalID, e.tokens["agent-1"].Groups))
	pkWhere := func(tenant, def uuid.UUID, col string, v int, acl bool) string {
		w := fmt.Sprintf("tenant_id = '%s' AND entity_definition_id = '%s' AND %s = %d", tenant, def, col, v)
		if acl {
			var gs []string
			for _, g := range e.tokens["agent-1"].Groups {
				gs = append(gs, "'"+g.String()+"'")
			}
			w += " AND assignment_group_id = ANY(ARRAY[" + strings.Join(gs, ", ") + "]::uuid[])"
		}
		return w
	}
	type est struct {
		label, where string
	}
	var smallT, smallDef uuid.UUID
	if err := e.super.QueryRow(ctx, `SELECT p.tenant_id, d.id FROM perf_tenant p JOIN entity_definitions d ON d.tenant_id = p.tenant_id AND d.key = 'ticket' WHERE p.tn = 1`).Scan(&smallT, &smallDef); err != nil {
		t.Fatal(err)
	}
	estimates := func(stage string) []est {
		col2 := "fx_own"
		if stage == "shared" {
			col2 = "fx_num_2"
		}
		out := []est{
			{"large type, pk 3 (50%)", pkWhere(e.large.id, e.largeDef, "fx_num_2", 3, false)},
			{"large type, pk 40 (2%)", pkWhere(e.large.id, e.largeDef, "fx_num_2", 40, false)},
			{"large type, pk 3, a6's groups", pkWhere(e.large.id, e.largeDef, "fx_num_2", 3, true)},
			{"ticket2, pk 3 (2%, uniform)", pkWhere(e.large.id, def2, col2, 3, false)},
			{"ticket2, pk 40 (2%, uniform)", pkWhere(e.large.id, def2, col2, 40, false)},
			{"ticket2, pk 3, a6's groups", pkWhere(e.large.id, def2, col2, 3, true)},
		}
		if stage == "shared" {
			out = append(out, est{"tenant s1's ticket, pk 3 (2%, uniform)", pkWhere(smallT, smallDef, "fx_num_2", 3, false)})
		}
		return out
	}
	e.w("")
	e.w("| Stage | Predicate | Estimate | Actual | Off by | S-c1 |\n| --- | --- | --- | --- | --- | --- |")
	var estRows []string
	for _, x := range estimates("own") {
		es, ac := e.slEstimate(t, x.where)
		r, v := slRatio(es, ac)
		estRows = append(estRows, fmt.Sprintf("| own slot | %s | %d | %d | %.1fx | %s |", x.label, es, ac, r, v))
	}
	cellsOwn := map[string]spCell{}
	own := map[uuid.UUID]string{}
	for k, v := range slotCols {
		own[k] = v
	}
	own[pk2] = "fx_own"
	sl.set(own, false)
	e.w("%s", spHead)
	type c1 struct{ name, path, tok string }
	c1s := []c1{
		{"large type, pk 3, a6", "ticket", "agent-1"}, {"large type, pk 40, a6", "ticket", "agent-1"},
		{"ticket2, pk 3, a6", "ticket2", "agent-1"}, {"ticket2, pk 40, a6", "ticket2", "agent-1"},
		{"ticket2, pk 3, no ACL", "ticket2", "admin-large"},
	}
	val := func(name string) string {
		if strings.Contains(name, "pk 40") {
			return "40"
		}
		return "3"
	}
	runC1 := func(stage string, c c1) spCell {
		sc := perfScenario{name: "S-c1 " + stage + " " + c.name, kind: "data", tenant: "large",
			reqs: []perfReq{{c.tok, "/api/v1/records/" + c.path + "?filter%5Bpk%5D=" + val(c.name), ""}}}
		fl := e.transportFloor(t, clients)
		r := e.run(t, sc, clients, d)
		cell := spCell{label: sc.name, p95: pct(r.lat, .95), p50: pct(r.lat, .5), floor: pct(fl, .5), n: len(r.lat), errs: r.errs, timeouts: r.timeouts, pageShape: "-", countShape: "-"}
		for i, s := range r.stmtSQL {
			if i >= len(r.explains) || r.explains[i] == "" || !strings.Contains(s.SQL, "r.fx_") {
				continue
			}
			if strings.Contains(s.SQL, "count(*)") {
				cell.countShape, cell.countPlan = spShape(r.explains[i]), r.explains[i]
			} else {
				cell.pageShape, cell.pagePlan = spShape(r.explains[i]), r.explains[i]
			}
			*plans = append(*plans, "### "+sc.name+"\n\n```sql\n"+strings.TrimSpace(s.SQL)+"\n```\n\n```\n"+r.explains[i]+"\n```")
		}
		e.w("%s", cell.row(fiListTargetMS))
		return cell
	}
	for _, c := range c1s {
		cellsOwn[c.name] = runC1("own slot", c)
	}
	// Share fx_num_2: ticket2 and every small tenant's ticket type, uniform.
	t1 := time.Now()
	e.exec(t, `UPDATE records SET fx_num_2 = fx_own WHERE tenant_id = $1 AND entity_definition_id = $2`, e.large.id, def2)
	e.exec(t, `INSERT INTO spike_slot_map SELECT $1, $2, 'fx_num_2', 'pk', 'num', 'ready'`, e.large.id, def2)
	e.exec(t, "SET session_replication_role = replica")
	e.exec(t, `UPDATE records r SET data = r.data || jsonb_build_object('pk', (('x' || substr(md5(r.id::text || 's'), 1, 7))::bit(28)::int) % 50)
		FROM perf_tenant p WHERE p.tn > 0 AND r.tenant_id = p.tenant_id`)
	e.exec(t, "SET session_replication_role = origin")
	e.exec(t, `UPDATE records r SET fx_num_2 = (r.data ->> 'pk')::float8 FROM perf_tenant p WHERE p.tn > 0 AND r.tenant_id = p.tenant_id`)
	e.exec(t, `INSERT INTO spike_slot_map SELECT d.tenant_id, d.id, 'fx_num_2', 'pk', 'num', 'ready' FROM entity_definitions d JOIN perf_tenant p ON p.tenant_id = d.tenant_id
		WHERE p.tn > 0 AND d.key = 'ticket'`)
	e.exec(t, `INSERT INTO record_index_fields (tenant_id, id, entity_definition_id, field_key, value_type, state, progress, requested_version)
		SELECT d.tenant_id, gen_random_uuid(), d.id, 'pk', 'num', 'ready', 100, 1 FROM entity_definitions d JOIN perf_tenant p ON p.tenant_id = d.tenant_id WHERE p.tn > 0 AND d.key = 'ticket'`)
	e.exec(t, "VACUUM (ANALYZE) records")
	e.w("")
	e.w("- shared: fx_num_2 now holds pk for the large type (skewed), ticket2 and all 399 small tenants (uniform), %s filled; filled in %s",
		e.scalar(t, `SELECT count(fx_num_2) FROM records`), time.Since(t1).Round(time.Second))
	g8, p8 := e.freshSizes(t, "fx_num_2", "")
	g4, p4 := e.freshSizes(t, "fx_num_2", fmt.Sprintf("tenant_id = '%s' AND entity_definition_id = '%s'", e.large.id, e.largeDef))
	e.w("- item 2 at 800k: fx_num_2 fresh builds, all %s filled rows: group %s, plain %s; the large type's 400k only: group %s, plain %s; value bytes %s",
		e.scalar(t, `SELECT count(fx_num_2) FROM records`), g8, p8, g4, p4, e.scalar(t, `SELECT pg_size_pretty(sum(pg_column_size(fx_num_2))) FROM records`))
	shared := map[uuid.UUID]string{}
	for k, v := range slotCols {
		shared[k] = v
	}
	shared[pk2] = "fx_num_2"
	var pkS1 uuid.UUID
	if err := e.super.QueryRow(ctx, `SELECT id FROM record_index_fields WHERE entity_definition_id = $1 AND field_key = 'pk'`, smallDef).Scan(&pkS1); err != nil {
		t.Fatal(err)
	}
	shared[pkS1] = "fx_num_2"
	sl.set(shared, false)
	for _, x := range estimates("shared") {
		es, ac := e.slEstimate(t, x.where)
		r, v := slRatio(es, ac)
		estRows = append(estRows, fmt.Sprintf("| shared slot | %s | %d | %d | %.1fx | %s |", x.label, es, ac, r, v))
	}
	e.w("")
	e.w("%s", spHead)
	var cmpRows []string
	for _, c := range c1s {
		s := runC1("shared slot", c)
		o := cellsOwn[c.name]
		v := "pass"
		if s.p95 > fiListTargetMS {
			v = "**fail**"
		}
		cmpRows = append(cmpRows, fmt.Sprintf("| %s | %.1f | %.1f | %s | %s | %s |", c.name, o.p95, s.p95, o.pageShape, s.pageShape, v))
		score(s, fiListTargetMS)
	}
	s1 := runC1("shared slot", c1{"tenant s1, pk 3, admin", "ticket", "admin-s1"})
	score(s1, fiListTargetMS)
	e.w("")
	e.w("| Stage | Predicate | Estimate | Actual | Off by | S-c1 |\n| --- | --- | --- | --- | --- | --- |")
	for _, r := range estRows {
		e.w("%s", r)
	}
	e.w("")
	e.w("| Cell | own slot p95 | shared slot p95 | own page plan | shared page plan | S-c1 |\n| --- | --- | --- | --- | --- | --- |")
	for _, r := range cmpRows {
		e.w("%s", r)
	}
	sl.set(slotCols, false)
	// ticket2 leaves: the carry curve and S-b3 run on the large type.
	e.exec(t, "SET session_replication_role = replica")
	e.exec(t, `DELETE FROM records WHERE tenant_id = $1 AND entity_definition_id = $2`, e.large.id, def2)
	e.exec(t, "SET session_replication_role = origin")
	e.exec(t, "ALTER TABLE records DROP COLUMN fx_own")
}

// copyColumns is records' column list and a SELECT list over r with
// overrides; every other column is copied as it is.
func (e *fiEnv) copyColumns(t *testing.T, over map[string]string) (string, string) {
	t.Helper()
	rows, err := e.super.Query(t.Context(), `SELECT column_name FROM information_schema.columns WHERE table_schema = 'public' AND table_name = 'records'
		AND is_generated = 'NEVER' ORDER BY ordinal_position`)
	if err != nil {
		t.Fatal(err)
	}
	names, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		t.Fatal(err)
	}
	var sel []string
	for _, n := range names {
		if o, ok := over[n]; ok {
			sel = append(sel, o)
		} else {
			sel = append(sel, "r."+n)
		}
	}
	return strings.Join(names, ", "), strings.Join(sel, ", ")
}

// ---- item 4 -----------------------------------------------------------------

func (e *fiEnv) carry(t *testing.T, clients int, listD, writeD time.Duration, sl *slState, slotCols map[uuid.UUID]string, creates []perfReq, targets [][]uuid.UUID, ver *slVer, steps []int, plans *[]string) {
	t.Helper()
	top := slices.Max(steps)
	e.w("")
	e.w("### Item 4: how many filled, indexed slots records carries (extra num slots fx_c_1..%d, each filled for the large type and indexed with the group key)", top)
	e.w("")
	t0 := time.Now()
	var add, fill []string
	var extraCols []slCol
	for i := 1; i <= top; i++ {
		c := fmt.Sprintf("fx_c_%d", i)
		add = append(add, "ADD COLUMN "+c+" double precision")
		fill = append(fill, fmt.Sprintf("%s = (data ->> 'score')::float8 + %d", c, i))
		extraCols = append(extraCols, slCol{c, "double precision", "num"})
	}
	e.exec(t, "ALTER TABLE records "+strings.Join(add, ", "))
	e.exec(t, slFuncSQL(append(slA3(), extraCols...)))
	e.exec(t, `UPDATE records SET `+strings.Join(fill, ", ")+` WHERE tenant_id = $1 AND entity_definition_id = $2`, e.large.id, e.largeDef)
	// The curve's targets carry the extra slots only.
	var nulls []string
	for _, c := range slA3() {
		nulls = append(nulls, c.col+" = NULL")
	}
	e.exec(t, `UPDATE records SET `+strings.Join(nulls, ", ")+` WHERE tenant_id = $1 AND id = ANY($2)`, e.large.id, slices.Concat(targets...))
	// The curve isolates the extra slots: the A3 slots are unmapped, and the
	// five D7 fields stay flagged in every step.
	e.exec(t, `UPDATE spike_slot_map SET state = 'removing' WHERE entity_definition_id = $1`, e.largeDef)
	for i := 1; i <= top; i++ {
		e.exec(t, `INSERT INTO spike_slot_map VALUES ($1, $2, $3, 'score', 'num', 'removing')`, e.large.id, e.largeDef, fmt.Sprintf("fx_c_%d", i))
	}
	e.exec(t, "VACUUM (ANALYZE) records")
	e.w("- %d extra columns added and filled for 400k records in %s; records %s", top, time.Since(t0).Round(time.Second),
		e.scalar(t, `SELECT pg_size_pretty(pg_table_size('records')) || ' heap, ' || pg_size_pretty(pg_indexes_size('records')) || ' indexes'`))
	e.w("")
	e.w("| Extra slots (indexes on records) | insert p50 / p95 ms | insert WAL B/op | update unrelated p50 / p95 ms | update WAL B/op | list a6 pk 3 per-group merge p95 ms / page planning ms | list a6 no filter p95 ms | records indexes size |\n| --- | --- | --- | --- | --- | --- | --- | --- |")
	built := 0
	for _, n := range steps {
		for i := built + 1; i <= n; i++ {
			c := fmt.Sprintf("fx_c_%d", i)
			e.exec(t, fmt.Sprintf("CREATE INDEX spike_%s_grp ON records (tenant_id, entity_definition_id, %s, assignment_group_id, updated_at) WHERE %s IS NOT NULL", c, c, c))
		}
		built = max(built, n)
		e.exec(t, `UPDATE spike_slot_map SET state = CASE WHEN col ~ '^fx_c_[0-9]+$' THEN (CASE WHEN substring(col FROM 6)::int <= $2 THEN 'ready' ELSE 'removing' END) ELSE 'removing' END
			WHERE entity_definition_id = $1`, e.largeDef, n)
		e.exec(t, "ANALYZE records")
		time.Sleep(slStatsWait)
		sc := perfScenario{name: fmt.Sprintf("create, %d extra slots", n), kind: "create", tenant: "large", reqs: creates}
		ins := e.slMeasure(t, fmt.Sprintf("%d extra", n), "insert", writeD, func() ([]time.Duration, int64, int) {
			r := e.run(t, sc, clients, writeD)
			return r.lat, r.errs, min(50, len(creates)) + 1
		})
		upd := e.slMeasure(t, fmt.Sprintf("%d extra", n), "update unrelated", writeD, func() ([]time.Duration, int64, int) {
			l, er := e.patchRun(t, writeD, targets, ver, func(v int) string { return fmt.Sprintf(`{"data": {"b001": %t}}`, v%2 == 0) })
			return l, er, 0
		})
		sl.set(slotCols, true)
		lf := e.cell(t, clients, listD, fmt.Sprintf("item 4, %d extra slots: a6 pk 3, per-group merge", n), []string{"agent-1"}, "filter%5Bpk%5D=3", plans)
		sl.set(slotCols, false)
		ln := e.cell(t, clients, listD, fmt.Sprintf("item 4, %d extra slots: a6, no filter", n), []string{"agent-1"}, "", plans)
		e.w("| %d (%s) | %.1f / %.1f | %.0f | %.1f / %.1f | %.0f | %.1f / %.2f | %.1f | %s |", n, e.scalar(t, `SELECT count(*) FROM pg_indexes WHERE tablename = 'records'`),
			ins.p50, ins.p95, ins.walPerOp, upd.p50, upd.p95, upd.walPerOp, lf.p95, lf.pagePlanning, ln.p95, e.scalar(t, `SELECT pg_size_pretty(pg_indexes_size('records'))`))
	}
	var drop []string
	for i := 1; i <= top; i++ {
		drop = append(drop, fmt.Sprintf("DROP COLUMN fx_c_%d", i))
	}
	e.exec(t, "ALTER TABLE records "+strings.Join(drop, ", "))
	e.exec(t, `DELETE FROM spike_slot_map WHERE col ~ '^fx_c_'`)
	e.exec(t, `UPDATE spike_slot_map SET state = 'ready' WHERE entity_definition_id = $1`, e.largeDef)
	e.exec(t, slFuncSQL(slA3()))
}

// ---- S-b3 -------------------------------------------------------------------

func (e *fiEnv) scale(t *testing.T, clients int, d time.Duration, sl *slState, slotCols map[uuid.UUID]string, plans *[]string, score func(spCell, float64)) {
	t.Helper()
	copies := perfEnvInt("NAVA_PERF_SLOT_COPIES", 4)
	e.w("")
	e.w("### S-b3: the large type at %d records (%d copies of its 400k, same values, ACL and groups), group index only, line 150 ms", perfLargeRecords*(copies+1), copies)
	e.w("")
	t0 := time.Now()
	e.exec(t, "SET session_replication_role = replica")
	for k := 1; k <= copies; k++ {
		colList, sel := e.copyColumns(t, map[string]string{
			"id":     fmt.Sprintf("md5('c%d/' || r.id::text)::uuid", k),
			"number": fmt.Sprintf("'C%d' || r.number", k),
		})
		e.exec(t, `INSERT INTO records (`+colList+`) SELECT `+sel+` FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND r.number LIKE 'TKT%'`, e.large.id, e.largeDef)
	}
	e.exec(t, "SET session_replication_role = origin")
	e.exec(t, "VACUUM (ANALYZE) records")
	e.w("- %s records of the large type after the copies (%s); a6 sees %s; pk 3 visible to a6: %s; records %s", e.scalar(t, `SELECT count(*) FROM records WHERE tenant_id = $1 AND entity_definition_id = $2`, e.large.id, e.largeDef),
		time.Since(t0).Round(time.Second), e.visibleCountS(t, "agent-1"),
		e.scalar(t, `SELECT count(*) FROM records r WHERE r.tenant_id = $1 AND r.entity_definition_id = $2 AND r.fx_num_2 = 3 AND `+spVisible, e.large.id, e.largeDef, e.tokens["agent-1"].PrincipalID, e.tokens["agent-1"].Groups),
		e.scalar(t, `SELECT pg_size_pretty(pg_table_size('records')) || ' heap, ' || pg_size_pretty(pg_indexes_size('records')) || ' indexes'`))
	e.w("")
	e.w("%s", spHead)
	for _, split := range []bool{false, true} {
		shape := "compiled (b)+group"
		if split {
			shape = "per-group merge"
		}
		sl.set(slotCols, split)
		r := e.cell(t, clients, d, fmt.Sprintf("S-b3 2M, broad 50%% (pk 3), a6, %s", shape), []string{"agent-1"}, "filter%5Bpk%5D=3", plans)
		e.w("%s", r.row(fiListTargetMS))
		ms, si, scans := slPageShape(r.pagePlan, slGroupIdx("fx_num_2"))
		e.w("- %s: %d slot-index scans, max %d rows per scan, max full-Sort input %d, count Heap Fetches %d", shape, scans, ms, si, slHeapFetches(r.countPlan))
		score(r, fiListTargetMS)
	}
	sl.set(slotCols, false)
}

func (e *fiEnv) visibleCountS(t *testing.T, tok string) string {
	t.Helper()
	return strconv.FormatInt(e.visibleCount(t, tok), 10)
}

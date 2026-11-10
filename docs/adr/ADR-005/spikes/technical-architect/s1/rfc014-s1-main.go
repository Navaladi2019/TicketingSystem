// Spike S1 (TICK-653, RFC-014): counter lock hold and wait for L0 / L1 / L2.
// THROWAWAY. Not production code. Never import.
//
// Statements are the ones at 4fc57183 (internal/records/store/batch.go,
// internal/tenancy/tenancy.go, internal/audit/append.go), against the real
// migrations at 4fc57183. Round trips per create:
//   L0: claim(tx) | BEGIN+tenant | def read | check batch | write batch (upsert first) | idem UPDATE+COMMIT
//   L1: claim(tx) | BEGIN+tenant | def read | check batch | write batch + idem UPDATE + COMMIT (one send)
//   L2: claim(tx) | BEGIN+tenant | def read | check batch | ALLOC tx on 2nd pool (one send) | write batch | idem UPDATE+COMMIT
// Server clocks: t_req = clock_timestamp() of the statement queued just
// before the upsert; t_acq = clock_timestamp() in the upsert's RETURNING
// (row lock granted). wait = t_acq - t_req. Hold (saturated) = t_acq(n+1) -
// t_acq(n) ordered by number n, over consecutive numbers at 8 clients.
// Hold (server, excl. WAL flush) = pg_xact_commit_timestamp(xid) - t_acq.
package main

import (
	"context"
	"crypto/rand"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync/atomic"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	pgAddr  = flag.String("pg", "127.0.0.1:50347", "postgres host:port")
	mode    = flag.String("mode", "L0", "L0|L1|L2|L2s|L2p (L2p = L2-prime, alloc on main pool before BEGIN, sync)|verify")
	crashAt = flag.Duration("crashafter", 0, "docker kill -s KILL the container after this long (A4)")
	ctr     = flag.String("container", "ta-s1-pg", "container to kill")
	clients = flag.Int("clients", 8, "concurrent creators")
	dur     = flag.Duration("dur", 20*time.Second, "measured duration")
	warm    = flag.Duration("warm", 2*time.Second, "warmup")
	oneWay  = flag.Duration("delay", 0, "injected one-way delay per direction (proxy)")
	allocN  = flag.Int("allocpool", 4, "L2 allocator pool size")
)

var tenant = uuid.MustParse("11111111-1111-1111-1111-111111111111")
var principal = uuid.MustParse("22222222-2222-2222-2222-222222222222")
var defID = uuid.MustParse("33333333-3333-3333-3333-333333333333")

// --- latency proxy: delays every chunk by oneWay in each direction, in order.
func proxy(target string, d time.Duration) string {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			s, err := net.Dial("tcp", target)
			if err != nil {
				c.Close()
				continue
			}
			go pipe(c, s, d)
			go pipe(s, c, d)
		}
	}()
	return ln.Addr().String()
}

type chunk struct {
	b  []byte
	at time.Time
}

func pipe(src, dst net.Conn, d time.Duration) {
	ch := make(chan chunk, 4096)
	go func() {
		for c := range ch {
			if w := time.Until(c.at); w > 0 {
				time.Sleep(w)
			}
			if _, err := dst.Write(c.b); err != nil {
				return
			}
		}
		dst.Close()
	}()
	buf := make([]byte, 64*1024)
	for {
		n, err := src.Read(buf)
		if n > 0 {
			b := make([]byte, n)
			copy(b, buf[:n])
			ch <- chunk{b, time.Now().Add(d)}
		}
		if err != nil {
			if err != io.EOF {
				_ = err
			}
			close(ch)
			return
		}
	}
}

type sample struct {
	n          int64
	xid        int64
	tReq, tAcq time.Time
	total      time.Duration
	start      time.Time
}

const beginSQL = "BEGIN; SELECT set_config('app.tenant_id', '11111111-1111-1111-1111-111111111111', true)"

const upsertSQL = `INSERT INTO record_counters (tenant_id, entity_definition_id, next) VALUES ($1, $2, 2)
ON CONFLICT (tenant_id, entity_definition_id) DO UPDATE SET next = record_counters.next + 1
RETURNING next - 1, clock_timestamp(), pg_current_xact_id()::text::bigint`

const numberSQL = `$15::text || (SELECT CASE WHEN c.next - 1 < 10000000 THEN lpad((c.next - 1)::text, 7, '0')
    ELSE (c.next - 1)::text END
  FROM record_counters c WHERE c.tenant_id = $1 AND c.entity_definition_id = $4)`

const insertRecordCols = `INSERT INTO records (tenant_id, id, version, entity_definition_id, schema_version, number, state,
    requester_id, assignee_id, assignment_group_id, created_by, created_at, updated_at, data,
    causation_origin_id, causation_depth) VALUES ($1, $2, $3, $4, $5, `

const insertRecordTail = `, $6, $7, $8, $9, $10, $11, $11, $12, $13, $14)
RETURNING number, created_at, updated_at, (data || jsonb_build_object('state', state))::text`

const writeKeySQL = `INSERT INTO record_write_keys (tenant_id, idempotency_key, definition_id, action, record_id, number,
    version, schema_version, audit_entry_id, actor_principal_id)
VALUES ($1, $2, $3, $4, $5, (SELECT number FROM records WHERE tenant_id = $1 AND id = $5), $6, $7, $8, $9)`

const auditSQL = `INSERT INTO audit_log (
	tenant_id, id, domain, action, subject_type, subject_id, entity_definition_id,
	version_before, version_after, actor_principal_id, actor_system, source, source_ref,
	change_set_id, causation_id, causation_depth, request_id, content_sha256, diff_sha256,
	diff_truncated, diff, diff_salt)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21::json, $22)`

// The completion in L0/L2 carries a body rendered in Go (it holds the number
// the write batch returned). In L1 the body must be built in SQL, because the
// number is not known to Go before the COMMIT is sent.
const idemDoneGo = `UPDATE idempotency_keys SET state = 'completed', response_status = 201, response_body = $4, response_headers = '{}'
WHERE tenant_id = $1 AND principal_id = $2 AND key = $3 AND state = 'in_flight'`

const idemDoneSQL = `UPDATE idempotency_keys SET state = 'completed', response_status = 201,
  response_body = convert_to(json_build_object('id', $4::uuid, 'number', (SELECT number FROM records WHERE tenant_id = $1 AND id = $4), 'version', 1)::text, 'UTF8'),
  response_headers = '{}'
WHERE tenant_id = $1 AND principal_id = $2 AND key = $3 AND state = 'in_flight'`

var maxAlloc atomic.Int64

func allocate(ctx context.Context, p *pgxpool.Pool, async bool, s *sample) error {
	b := &pgx.Batch{}
	b.Queue("BEGIN")
	b.Queue("SELECT set_config('app.tenant_id', $1, true), clock_timestamp()", tenant.String()).QueryRow(func(r pgx.Row) error {
		var x string
		return r.Scan(&x, &s.tReq)
	})
	if async {
		b.Queue("SET LOCAL synchronous_commit = off")
	}
	b.Queue(upsertSQL, tenant, defID).QueryRow(func(r pgx.Row) error {
		if err := r.Scan(&s.n, &s.tAcq, &s.xid); err != nil {
			return err
		}
		for {
			c := maxAlloc.Load()
			if s.n <= c || maxAlloc.CompareAndSwap(c, s.n) {
				break
			}
		}
		return nil
	})
	b.Queue("COMMIT")
	if err := p.SendBatch(ctx, b).Close(); err != nil {
		return fmt.Errorf("alloc: %w", err)
	}
	return nil
}

func rnd(n int) []byte { b := make([]byte, n); rand.Read(b); return b }

func create(ctx context.Context, main, alloc *pgxpool.Pool, m string) (sample, error) {
	var s sample
	s.start = time.Now()
	key := uuid.NewString()
	recID, _ := uuid.NewV7()
	auditID, _ := uuid.NewV7()
	data := []byte(`{"title":"spike create","priority":"p3","impact":"low","description":"s1 spike payload of moderate size"}`)
	diff := `[{"op":"add","path":"/title","value":"spike create"},{"op":"add","path":"/priority","value":"p3"}]`

	// 1. idempotency claim, own transaction (commit fsync #1)
	{
		b := &pgx.Batch{}
		b.Queue("BEGIN")
		b.Queue("SELECT set_config('app.tenant_id', $1, true)", tenant.String())
		b.Queue(`INSERT INTO idempotency_keys (tenant_id, principal_id, key, request_hash, expires_at) VALUES ($1,$2,$3,$4, now() + interval '1 day')`,
			tenant, principal, key, rnd(32))
		b.Queue("COMMIT")
		if err := main.SendBatch(ctx, b).Close(); err != nil {
			return s, fmt.Errorf("claim: %w", err)
		}
	}
	// L2-prime: allocate before the create transaction, on the main pool
	// (no second pool; the create's checks have not run yet).
	if m == "L2p" {
		if err := allocate(ctx, main, false, &s); err != nil {
			return s, err
		}
	}
	conn, err := main.Acquire(ctx)
	if err != nil {
		return s, err
	}
	defer conn.Release()
	// 2. BEGIN + tenant (simple protocol, one round trip)
	if _, err := conn.Exec(ctx, beginSQL, pgx.QueryExecModeSimpleProtocol); err != nil {
		return s, fmt.Errorf("begin: %w", err)
	}
	rollback := func() { _, _ = conn.Exec(context.Background(), "ROLLBACK") }
	// 3. definition read
	var prefix string
	if err := conn.QueryRow(ctx, `SELECT number_prefix FROM entity_definitions WHERE tenant_id = $1 AND id = $2`, tenant, defID).Scan(&prefix); err != nil {
		rollback()
		return s, fmt.Errorf("def: %w", err)
	}
	// 4. check batch (write-key lookup + payload preview)
	{
		b := &pgx.Batch{}
		b.Queue(`SELECT record_id FROM record_write_keys WHERE tenant_id = $1 AND idempotency_key = $2 AND actor_principal_id = $3`, tenant, key, principal)
		b.Queue(`SELECT ($1::jsonb || jsonb_build_object('state', 'new'::text))::text`, string(data))
		if err := conn.SendBatch(ctx, b).Close(); err != nil {
			rollback()
			return s, fmt.Errorf("checks: %w", err)
		}
	}
	// 5. L2/L2s: allocate in its own transaction on the allocator pool, one send.
	var number string
	if m == "L2" || m == "L2s" {
		if err := allocate(ctx, alloc, m == "L2s", &s); err != nil {
			rollback()
			return s, err
		}
		number = fmt.Sprintf("%s%07d", prefix, s.n)
	}
	if m == "L2p" {
		number = fmt.Sprintf("%s%07d", prefix, s.n)
	}
	recArgs := []any{tenant, recID, int64(1), defID, 1, "new", principal, nil, nil, principal, time.Now(), string(data), nil, int16(0)}
	b := &pgx.Batch{}
	if m == "L0" || m == "L1" {
		b.Queue("SELECT clock_timestamp()").QueryRow(func(r pgx.Row) error { return r.Scan(&s.tReq) })
		b.Queue(upsertSQL, tenant, defID).QueryRow(func(r pgx.Row) error { return r.Scan(&s.n, &s.tAcq, &s.xid) })
		b.Queue(insertRecordCols+numberSQL+insertRecordTail, append(recArgs, prefix)...)
	} else {
		b.Queue(insertRecordCols+"$15"+insertRecordTail, append(recArgs, number)...)
	}
	b.Queue(writeKeySQL, tenant, key, defID, "create", recID, int64(1), 1, auditID, principal)
	b.Queue(auditSQL, tenant, auditID, "record", "create", "record", recID, defID, nil, int64(1), principal, nil, "user", "spike",
		nil, nil, int16(0), "req-"+key[:8], nil, rnd(32), false, diff, rnd(16))
	switch m {
	case "L0", "L2", "L2s", "L2p":
		if err := conn.SendBatch(ctx, b).Close(); err != nil {
			rollback()
			return s, fmt.Errorf("write: %w", err)
		}
		if m == "L0" {
			number = fmt.Sprintf("%s%07d", prefix, s.n)
		}
		// Go renders the response (it now knows the number), then the
		// completion rides with COMMIT (tenancy.commitBatch).
		body := []byte(fmt.Sprintf(`{"id":%q,"number":%q,"version":1}`, recID, number))
		cb := &pgx.Batch{}
		cb.Queue(idemDoneGo, tenant, principal, key, body)
		cb.Queue("COMMIT")
		if err := conn.SendBatch(ctx, cb).Close(); err != nil {
			rollback()
			return s, fmt.Errorf("commit: %w", err)
		}
	case "L1":
		b.Queue(idemDoneSQL, tenant, principal, key, recID)
		b.Queue("COMMIT")
		if err := conn.SendBatch(ctx, b).Close(); err != nil {
			rollback()
			return s, fmt.Errorf("write+commit: %w", err)
		}
	}
	if st := conn.Conn().PgConn().TxStatus(); st != 'I' {
		return s, fmt.Errorf("tx status %q", st)
	}
	s.total = time.Since(s.start)
	return s, nil
}

func pct(d []float64, p float64) float64 {
	if len(d) == 0 {
		return 0
	}
	sort.Float64s(d)
	i := int(p*float64(len(d))+0.999999) - 1
	if i < 0 {
		i = 0
	}
	if i >= len(d) {
		i = len(d) - 1
	}
	return d[i]
}

func ms(d time.Duration) float64 { return float64(d.Microseconds()) / 1000 }

func main() {
	flag.Parse()
	ctx := context.Background()
	addr := *pgAddr
	if *oneWay > 0 {
		addr = proxy(*pgAddr, *oneWay)
	}
	dsn := "postgres://nava_app:spike@" + addr + "/spike?sslmode=disable"
	cfg, _ := pgxpool.ParseConfig(dsn)
	cfg.MaxConns = int32(*clients*2 + 2) // claim + create conns; never the bottleneck
	cfg.MinConns = cfg.MaxConns
	main_, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		panic(err)
	}
	acfg, _ := pgxpool.ParseConfig(dsn)
	acfg.MaxConns = int32(*allocN)
	acfg.MinConns = acfg.MaxConns
	alloc, err := pgxpool.NewWithConfig(ctx, acfg)
	if err != nil {
		panic(err)
	}
	if *mode == "verify" {
		verify(ctx, main_)
		return
	}
	// transport floor and bare commit cost
	var ping, commit []float64
	for i := 0; i < 200; i++ {
		t := time.Now()
		main_.Exec(ctx, "SELECT 1")
		ping = append(ping, ms(time.Since(t)))
		t = time.Now()
		bb := &pgx.Batch{}
		bb.Queue("BEGIN")
		bb.Queue("SELECT set_config('app.tenant_id', $1, true)", tenant.String())
		bb.Queue(`INSERT INTO record_counters (tenant_id, entity_definition_id, next) VALUES ($1, $2, 2) ON CONFLICT (tenant_id, entity_definition_id) DO UPDATE SET next = record_counters.next`, tenant, uuid.MustParse("44444444-4444-4444-4444-444444444444"))
		bb.Queue("COMMIT")
		if err := alloc.SendBatch(ctx, bb).Close(); err != nil {
			panic(err)
		}
		commit = append(commit, ms(time.Since(t)))
	}

	var mu sync.Mutex
	var out []sample
	var maxAcked atomic.Int64
	if *crashAt > 0 {
		go func() {
			time.Sleep(*warm + *crashAt)
			o, err := exec.Command("docker", "kill", "-s", "KILL", *ctr).CombinedOutput()
			fmt.Printf("  CRASH: docker kill -s KILL %s at %s -> %s %v (max allocated so far %d)\n", *ctr, time.Now().Format(time.RFC3339Nano), strings.TrimSpace(string(o)), err, maxAlloc.Load())
		}()
	}
	errs := 0
	var errOnce string
	deadline := time.Now().Add(*warm + *dur)
	measureFrom := time.Now().Add(*warm)
	var wg sync.WaitGroup
	for c := 0; c < *clients; c++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for time.Now().Before(deadline) {
				s, err := create(ctx, main_, alloc, *mode)
				mu.Lock()
				if err != nil {
					errs++
					if errOnce == "" {
						errOnce = err.Error()
					}
				} else if s.start.After(measureFrom) || *crashAt > 0 {
					for {
						c := maxAcked.Load()
						if s.n <= c || maxAcked.CompareAndSwap(c, s.n) {
							break
						}
					}
					out = append(out, s)
				}
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if *crashAt > 0 {
		fmt.Printf("mode=%s crash run: acked creates=%d errors=%d max_acked_n=%d max_allocated_n=%d first_error=%q\n", *mode, len(out), errs, maxAcked.Load(), maxAlloc.Load(), errOnce)
		return
	}
	// commit timestamps
	var hsrv []float64
	conn, _ := main_.Acquire(ctx)
	for _, s := range out {
		var ct time.Time
		if err := conn.QueryRow(ctx, `SELECT pg_xact_commit_timestamp((($1::bigint) % 4294967296)::text::xid)`, s.xid).Scan(&ct); err == nil {
			hsrv = append(hsrv, ms(ct.Sub(s.tAcq)))
		}
	}
	conn.Release()
	sort.Slice(out, func(i, j int) bool { return out[i].n < out[j].n })
	var tot, wait, gap []float64
	for i, s := range out {
		tot = append(tot, ms(s.total))
		wait = append(wait, ms(s.tAcq.Sub(s.tReq)))
		if i > 0 && out[i-1].n == s.n-1 {
			gap = append(gap, ms(s.tAcq.Sub(out[i-1].tAcq)))
		}
	}
	fmt.Printf("mode=%s clients=%d delay_oneway=%v creates=%d errors=%d rate=%.1f/s\n", *mode, *clients, *oneWay, len(out), errs, float64(len(out))/dur.Seconds())
	if errOnce != "" {
		fmt.Println("  first error:", errOnce)
	}
	fmt.Printf("  transport SELECT 1 p50/p95 %.2f/%.2f ms; bare 1-row commit txn p50/p95 %.2f/%.2f ms\n", pct(ping, .5), pct(ping, .95), pct(commit, .5), pct(commit, .95))
	fmt.Printf("  create  p50/p95/p99  %.2f / %.2f / %.2f ms\n", pct(tot, .5), pct(tot, .95), pct(tot, .99))
	fmt.Printf("  wait    p50/p95/p99  %.2f / %.2f / %.2f ms\n", pct(wait, .5), pct(wait, .95), pct(wait, .99))
	fmt.Printf("  hold (acq->commit_ts, excl WAL flush) p50/p95 %.2f / %.2f ms  [n=%d]\n", pct(hsrv, .5), pct(hsrv, .95), len(hsrv))
	fmt.Printf("  hold (inter-acquisition, consecutive n; = hold incl flush when saturated) p50/p95 %.2f / %.2f ms [n=%d]\n", pct(gap, .5), pct(gap, .95), len(gap))
	os.Exit(0)
}

// verify (A4): after restart, every committed record number must be below the
// durable counter, no number may repeat, and fresh creates must not collide.
func verify(ctx context.Context, p *pgxpool.Pool) {
	conn, err := p.Acquire(ctx)
	if err != nil {
		panic(err)
	}
	defer conn.Release()
	conn.Exec(ctx, beginSQL, pgx.QueryExecModeSimpleProtocol)
	var next, maxN, cnt, dups int64
	if err := conn.QueryRow(ctx, `SELECT c.next,
	  (SELECT coalesce(max(substr(number,4)::bigint),0) FROM records r WHERE r.tenant_id=c.tenant_id AND r.entity_definition_id=c.entity_definition_id),
	  (SELECT count(*) FROM records r WHERE r.tenant_id=c.tenant_id AND r.entity_definition_id=c.entity_definition_id),
	  (SELECT count(*) FROM (SELECT number FROM records r WHERE r.tenant_id=c.tenant_id AND r.entity_definition_id=c.entity_definition_id GROUP BY number HAVING count(*)>1) d)
	  FROM record_counters c WHERE c.tenant_id=$1 AND c.entity_definition_id=$2`, tenant, defID).Scan(&next, &maxN, &cnt, &dups); err != nil {
		panic(err)
	}
	conn.Exec(ctx, "COMMIT")
	ok := maxN <= next-1 && dups == 0
	fmt.Printf("verify: counter next-1=%d max_record_n=%d records=%d duplicate_numbers=%d invariant(max_record_n<=next-1 && no dups)=%v\n", next-1, maxN, cnt, dups, ok)
	_ = strconv.Itoa
}

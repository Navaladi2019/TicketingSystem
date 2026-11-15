//go:build integration

// TICK-865 (ADR-004-E9, CI-A18 (a), (b), (c), (f), (g), (i), Sec P3): a
// create refused with number_space_exhausted leaves exactly one R2b audit
// entry, appended by CreateRecord on the API pool after the create's
// transaction has ended, on both numbering modes, through the real handler
// and a real PostgreSQL.

package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/internal/audit/verify"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/cursor"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
	"github.com/Navaladi2019/TicketingSystem/internal/server"
	"github.com/Navaladi2019/TicketingSystem/internal/tenancy"
)

// exhKey is the entity type whose format, {prefix}-{seq:1} with overflow
// refuse, is full after nine creates.
const exhKey = "exh"

var exhModes = []string{"gaps", "gapless"}

// poolProbe traces a pool's acquires and releases while armed. onRelease,
// when set, runs once, on the first release while armed, before the
// connection goes back to the pool.
type poolProbe struct {
	prev      pgx.QueryTracer
	mu        sync.Mutex
	armed     bool
	events    []string
	held, max int
	onRelease func()
}

func (p *poolProbe) TraceQueryStart(ctx context.Context, c *pgx.Conn, d pgx.TraceQueryStartData) context.Context {
	if p.prev != nil {
		return p.prev.TraceQueryStart(ctx, c, d)
	}
	return ctx
}

func (p *poolProbe) TraceQueryEnd(ctx context.Context, c *pgx.Conn, d pgx.TraceQueryEndData) {
	if p.prev != nil {
		p.prev.TraceQueryEnd(ctx, c, d)
	}
}

func (p *poolProbe) TraceAcquireStart(ctx context.Context, _ *pgxpool.Pool, _ pgxpool.TraceAcquireStartData) context.Context {
	return ctx
}

func (p *poolProbe) TraceAcquireEnd(_ context.Context, _ *pgxpool.Pool, d pgxpool.TraceAcquireEndData) {
	if d.Err != nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.armed {
		p.events = append(p.events, "acquire")
		p.held++
		p.max = max(p.max, p.held)
	}
}

func (p *poolProbe) TraceRelease(_ *pgxpool.Pool, _ pgxpool.TraceReleaseData) {
	p.mu.Lock()
	var fn func()
	if p.armed {
		p.events = append(p.events, "release")
		p.held--
		fn, p.onRelease = p.onRelease, nil
	}
	p.mu.Unlock()
	if fn != nil {
		fn()
	}
}

func (p *poolProbe) arm(onRelease func()) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.armed, p.events, p.held, p.max, p.onRelease = true, nil, 0, 0, onRelease
}

func (p *poolProbe) disarm() (events []string, maxHeld int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.armed = false
	return p.events, p.max
}

// probedPool is a pool on base's configuration, traced by a poolProbe; alloc
// makes it an allocator pool as etAllocPool does.
func probedPool(t *testing.T, base *pgxpool.Pool, alloc bool) (*pgxpool.Pool, *poolProbe) {
	t.Helper()
	cfg := base.Config().Copy()
	if alloc {
		cfg.MaxConns = 4
		if err := tenancy.AllocPoolConfig(cfg); err != nil {
			t.Fatal(err)
		}
	}
	probe := &poolProbe{prev: cfg.ConnConfig.Tracer}
	cfg.ConnConfig.Tracer = probe
	p, err := pgxpool.NewWithConfig(t.Context(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(p.Close)
	return p, probe
}

type exhEnv struct {
	*exportEnv
	defID uuid.UUID
}

// newExhEnv is the export environment (chain heads, the sealer's pool) with
// tenant A's type exh in mode, linked to a one-digit refuse format and
// filled to EXH-9, so its next create is refused.
func newExhEnv(t *testing.T, mode string, extra ...server.Option) *exhEnv {
	t.Helper()
	open, now := true, time.Now()
	x := &exhEnv{exportEnv: newExportEnv(t, append([]server.Option{server.WithNumberFormatGate(nfGate(&open, &now))}, extra...)...)}
	body := etBody(exhKey, "EXH")
	if mode == "gapless" {
		body = strings.Replace(body, `"number_prefix": "EXH",`, `"number_prefix": "EXH", "numbering_mode": "gapless",`, 1)
	}
	if rec := x.do(t, http.MethodPost, "/api/v1/entity-types", "admin-a", linkKey("exh-type"), body); rec.Code != http.StatusCreated {
		t.Fatalf("create type = %d: %s", rec.Code, rec.Body)
	}
	rec := x.do(t, http.MethodPost, nfPath, "admin-a", linkKey("exh-fmt"),
		`{"key":"exh_one_digit","document":{"template":"{prefix}-{seq:1}","overflow":"refuse"}}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create format = %d: %s", rec.Code, rec.Body)
	}
	if rec := x.doIf(t, http.MethodPost, "/api/v1/entity-types/"+exhKey+"/versions", "admin-a", linkKey("exh-link"), "", pubBody(ref(nfID(t, rec.Body.Bytes()), 1))); rec.Code != http.StatusCreated {
		t.Fatalf("link format = %d: %s", rec.Code, rec.Body)
	}
	var stored string
	if err := x.super.QueryRow(t.Context(), "SELECT id, numbering_mode FROM entity_definitions WHERE tenant_id = $1 AND key = $2", x.a.id, exhKey).Scan(&x.defID, &stored); err != nil {
		t.Fatal(err)
	}
	if stored != mode {
		t.Fatalf("numbering mode %q, want %q", stored, mode)
	}
	for i := 1; i <= 9; i++ {
		if r := x.mustCreateRec(t, "user-a", exhKey, linkKey("exh-fill"), `{"short_description": "fill"}`); r.Number != fmt.Sprintf("EXH-%d", i) {
			t.Fatalf("create %d: number %s", i, r.Number)
		}
	}
	return x
}

// serveOn replaces the handler with one on the API pool api and the
// allocator pool alloc, after the setup, so a probe on them sees only what
// the test sends next.
func (x *exhEnv) serveOn(t *testing.T, api, alloc *pgxpool.Pool) {
	t.Helper()
	kr, err := cursor.NewKeyring("k1", cursor.Key{ID: "k1", Secret: bytes.Repeat([]byte("s"), 32)})
	if err != nil {
		t.Fatal(err)
	}
	log := slog.New(slog.NewTextHandler(testLogWriter{t, x.logs}, &slog.HandlerOptions{Level: slog.LevelWarn}))
	if x.h, err = server.NewHandler(log, x.tel, x.tokens, stubPaths{}, server.WithLimiter(x.lim), server.WithPool(api),
		server.WithAllocatorPool(alloc), server.WithAuditReader(etAuditReader(t, x.pool)), server.WithCursors(kr)); err != nil {
		t.Fatal(err)
	}
}

// refuse sends one create of exh and wants the 409 number_space_exhausted.
func (x *exhEnv) refuse(t *testing.T, idemKey string) etProblem {
	t.Helper()
	return wantProblem(t, x.createRec(t, "user-a", exhKey, idemKey, `{"short_description": "refused"}`), http.StatusConflict, problem.NumberSpaceExhausted)
}

// exhRow is a number_space_exhausted entry as stored.
type exhRow struct {
	txid, domain, subjectType, source string
	subjectID                         uuid.UUID
	defID, actor, causation           *uuid.UUID
	versionBefore, versionAfter       *int64
	actorSystem                       *string
	depth                             int16
	noContentHash                     bool
	diff                              string
}

func (x *exhEnv) exhRows(t *testing.T, tenant uuid.UUID) []exhRow {
	t.Helper()
	rows, err := x.super.Query(t.Context(), `SELECT txid::text, domain, subject_type, source, subject_id, entity_definition_id,
  actor_principal_id, causation_id, version_before, version_after, actor_system, causation_depth, content_sha256 IS NULL, diff::text
FROM audit_log WHERE tenant_id = $1 AND action = 'number_space_exhausted' ORDER BY occurred_at, id`, tenant)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []exhRow
	for rows.Next() {
		var r exhRow
		if err := rows.Scan(&r.txid, &r.domain, &r.subjectType, &r.source, &r.subjectID, &r.defID, &r.actor, &r.causation,
			&r.versionBefore, &r.versionAfter, &r.actorSystem, &r.depth, &r.noContentHash, &r.diff); err != nil {
			t.Fatal(err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

func (x *exhEnv) exhRecords(t *testing.T) int {
	t.Helper()
	var n int
	if err := x.super.QueryRow(t.Context(), "SELECT count(*) FROM records WHERE tenant_id = $1 AND entity_definition_id = $2", x.a.id, x.defID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func (x *exhEnv) counterNext(t *testing.T) []int64 {
	t.Helper()
	rows, err := x.super.Query(t.Context(), "SELECT next FROM record_counters WHERE tenant_id = $1 AND entity_definition_id = $2 ORDER BY scope_key", x.a.id, x.defID)
	if err != nil {
		t.Fatal(err)
	}
	out, err := pgx.CollectRows(rows, pgx.RowTo[int64])
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// wantR2b checks one entry against the R2b table (CI-A18 (a)): subject the
// type, no versions, user-a as actor, a depth-0 origin, no content hash, and
// a diff of exactly one add of /number_space with the four keys and the
// counter; it returns the scope key.
func (x *exhEnv) wantR2b(t *testing.T, r exhRow, counter int64) string {
	t.Helper()
	user := x.a.user.PrincipalID
	if r.domain != "record" || r.subjectType != "entity_definition" || r.subjectID != x.defID || r.defID == nil || *r.defID != x.defID {
		t.Errorf("entry %+v, want record domain, subject entity_definition %s", r, x.defID)
	}
	if r.versionBefore != nil || r.versionAfter != nil || !r.noContentHash {
		t.Errorf("entry %+v, want versions and content_sha256 NULL", r)
	}
	if r.actor == nil || *r.actor != user || r.actorSystem != nil || r.source != "user" || r.causation == nil || r.depth != 0 {
		t.Errorf("entry %+v, want actor user-a, source user, a depth-0 origin", r)
	}
	var ops []struct {
		Op, Path string
		Value    map[string]json.RawMessage
	}
	if err := json.Unmarshal([]byte(r.diff), &ops); err != nil || len(ops) != 1 || ops[0].Op != "add" || ops[0].Path != "/number_space" {
		t.Fatalf("diff %s (%v), want one add of /number_space", r.diff, err)
	}
	keys := slices.Sorted(func(yield func(string) bool) {
		for k := range ops[0].Value {
			if !yield(k) {
				return
			}
		}
	})
	if !slices.Equal(keys, []string{"counter", "epoch", "scope_key", "width"}) {
		t.Errorf("diff keys %v, want counter, epoch, scope_key, width only (no record data)", keys)
	}
	var scope string
	_ = json.Unmarshal(ops[0].Value["scope_key"], &scope)
	if string(ops[0].Value["counter"]) != fmt.Sprint(counter) || string(ops[0].Value["epoch"]) != "1" || string(ops[0].Value["width"]) != "1" || scope != "e1" {
		t.Errorf("diff %s, want counter %d, epoch 1, width 1, scope e1 (the format has no date part)", r.diff, counter)
	}
	if strings.Contains(r.diff, "refused") || strings.Contains(r.diff, "short_description") {
		t.Errorf("diff %s carries record data", r.diff)
	}
	return scope
}

// exhaustedCount reads records_number_space_exhausted_total, failing on a
// tenant or definition label.
func exhaustedCount(t *testing.T, x *exhEnv) float64 {
	t.Helper()
	fams, err := x.tel.Registry.Gather()
	if err != nil {
		t.Fatal(err)
	}
	var n float64
	for _, f := range fams {
		if f.GetName() != "records_number_space_exhausted_total" {
			continue
		}
		for _, m := range f.GetMetric() {
			for _, l := range m.GetLabel() {
				if name := l.GetName(); strings.Contains(name, "tenant") || strings.Contains(name, "definition") || strings.Contains(name, "entity") {
					t.Errorf("records_number_space_exhausted_total has label %s", name)
				}
			}
			n += m.GetCounter().GetValue()
		}
	}
	return n
}

// CI-A18 (a), (b), (c), (g): each refused create answers 409, writes no
// record and exactly one R2b entry, in a transaction of its own. `gaps`
// counters rise by one per refusal; `gapless` ones repeat max + 1 and the
// counter row is unchanged. A create with room writes no such entry; a replay
// of a refused create's key is a new attempt with a new entry (I7). The
// metric counts each refusal; the log names the tenant and type, never the
// scope; the audit chain verifies with the new rows.
//
// covers: createRecord 409
func TestNumberSpaceExhaustedCreateAppendsOneEntryPerRefusal(t *testing.T) {
	for _, mode := range exhModes {
		t.Run(mode, func(t *testing.T) {
			x := newExhEnv(t, mode)
			if rows := x.exhRows(t, x.a.id); len(rows) != 0 {
				t.Fatalf("201 creates wrote %d number_space_exhausted entries", len(rows))
			}
			before := x.counterNext(t)
			const replayKey = "exh-replayed"
			txids := map[string]bool{}
			var scope string
			for i := range 5 {
				key := linkKey("exh-refuse")
				if i < 2 {
					key = replayKey
				}
				p := x.refuse(t, key)
				rows := x.exhRows(t, x.a.id)
				if len(rows) != i+1 {
					t.Fatalf("after refusal %d: %d entries, want %d", i+1, len(rows), i+1)
				}
				counter := map[string]int64{"gaps": int64(10 + i), "gapless": 10}[mode]
				scope = x.wantR2b(t, rows[i], counter)
				if len(p.Errors) != 1 || p.Errors[0].Rule != "scope:"+scope {
					t.Errorf("problem fields %+v, want the scope %s", p.Errors, scope)
				}
				txids[rows[i].txid] = true
			}
			if len(txids) != 5 {
				t.Errorf("five entries in %d transactions, want 5 (one entry per transaction)", len(txids))
			}
			if n := x.exhRecords(t); n != 9 {
				t.Errorf("%d records, want 9", n)
			}
			after := x.counterNext(t)
			want := map[string][]int64{"gaps": {15}, "gapless": before}[mode]
			if !slices.Equal(after, want) {
				t.Errorf("counter next %v (before %v), want %v", after, before, want)
			}
			if n := exhaustedCount(t, x); n != 5 {
				t.Errorf("records_number_space_exhausted_total = %v, want 5", n)
			}
			logs := x.logs.String()
			if got := strings.Count(logs, "record create refused: number space exhausted"); got != 5 {
				t.Errorf("%d refusal log lines, want 5:\n%s", got, logs)
			}
			if !strings.Contains(logs, x.a.id.String()) || !strings.Contains(logs, "entity_key="+exhKey) {
				t.Errorf("refusal log lines name no tenant or type:\n%s", logs)
			}
			// The refusal line carries tenant_id and entity_key only: no scope
			// attribute and no scope value. The bare value "e1" can occur in
			// a UUID, so only the attribute forms are matched.
			for _, line := range strings.Split(logs, "\n") {
				if !strings.Contains(line, "number space exhausted") {
					continue
				}
				if strings.Contains(line, "scope") || strings.Contains(line, "="+scope+" ") ||
					strings.HasSuffix(line, "="+scope) || strings.Contains(line, `"`+scope+`"`) {
					t.Errorf("a refusal log line names the scope key %s: %s", scope, line)
				}
			}
			if n := len(x.exhRows(t, x.b.id)); n != 0 {
				t.Errorf("tenant B has %d number_space_exhausted entries", n)
			}
			x.sealAll(t)
			if rep := report(t, x.export(t, "admin-a", "").Body.Bytes()); rep.Outcome != verify.OutcomePass {
				t.Errorf("tenant A's chain: outcome %s, failure %+v", rep.Outcome, rep.Failure)
			}
		})
	}
}

// CI-A18 (f): an append that fails answers the refusal (409, never a
// success), writes no record and no entry, and logs the audit error, which
// the handler joined to the refusal.
func TestNumberSpaceExhaustedFailingAppendJoinsTheAuditError(t *testing.T) {
	for _, mode := range exhModes {
		t.Run(mode, func(t *testing.T) {
			x := newExhEnv(t, mode)
			for _, q := range []string{
				`CREATE FUNCTION exh_fail() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'exh test: the audit append fails'; END $$`,
				`CREATE TRIGGER exh_fail BEFORE INSERT ON audit_log FOR EACH ROW WHEN (NEW.action = 'number_space_exhausted') EXECUTE FUNCTION exh_fail()`,
			} {
				if _, err := x.super.Exec(t.Context(), q); err != nil {
					t.Fatal(err)
				}
			}
			x.refuse(t, linkKey("exh-fail"))
			if n := len(x.exhRows(t, x.a.id)); n != 0 {
				t.Errorf("%d entries, want 0", n)
			}
			if n := x.exhRecords(t); n != 9 {
				t.Errorf("%d records, want 9", n)
			}
			logs := x.logs.String()
			if !strings.Contains(logs, "audit of a refused record create failed") || !strings.Contains(logs, "exh test: the audit append fails") {
				t.Errorf("no log of the failed append and its cause:\n%s", logs)
			}
		})
	}
}

// CI-A18 (i): the entry is written on the API pool after the create's
// connection was released (never two API connections held at once: no
// hold-and-wait), with the allocator pool closed once the `gaps` allocation
// returned, or before a `gapless` create, which never uses it.
func TestNumberSpaceExhaustedEntryIsAppendedOnTheAPIPoolAfterTheCreate(t *testing.T) {
	for _, mode := range exhModes {
		t.Run(mode, func(t *testing.T) {
			x := newExhEnv(t, mode)
			api, apiProbe := probedPool(t, x.pool, false)
			alloc, allocProbe := probedPool(t, x.pool, true)
			x.serveOn(t, api, alloc)

			closed := make(chan struct{})
			if mode == "gapless" {
				alloc.Close()
				close(closed)
				allocProbe.arm(nil)
			} else {
				allocProbe.arm(func() { go func() { alloc.Close(); close(closed) }() })
			}
			apiProbe.arm(nil)
			x.refuse(t, linkKey("exh-pool"))
			events, maxHeld := apiProbe.disarm()
			allocEvents, _ := allocProbe.disarm()

			if rows := x.exhRows(t, x.a.id); len(rows) != 1 {
				t.Fatalf("%d entries, want 1", len(rows))
			}
			if want := []string{"acquire", "release", "acquire", "release"}; !slices.Equal(events, want) || maxHeld != 1 {
				t.Errorf("API pool events %v (at most %d held), want %v with one held at a time", events, maxHeld, want)
			}
			wantAlloc := map[string][]string{"gaps": {"acquire", "release"}, "gapless": nil}[mode]
			if !slices.Equal(allocEvents, wantAlloc) {
				t.Errorf("allocator pool events %v, want %v", allocEvents, wantAlloc)
			}
			select {
			case <-closed:
			case <-time.After(10 * time.Second):
				t.Fatal("the allocator pool did not close")
			}
			if _, err := alloc.Acquire(t.Context()); err == nil {
				t.Error("the allocator pool is still open")
			}
		})
	}
}

// Sec P3 (ADR-004-E9 N1): a request cancelled right after the refusal, once
// the create's connection is released and before the append, still writes
// its entry.
func TestNumberSpaceExhaustedEntrySurvivesACancelledRequest(t *testing.T) {
	x := newExhEnv(t, "gaps")
	api, probe := probedPool(t, x.pool, false)
	x.serveOn(t, api, etAllocPool(t, x.pool))

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	probe.arm(cancel)
	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/api/v1/records/"+exhKey, strings.NewReader(`{"data": {"short_description": "refused"}}`))
	req.Header.Set("Authorization", "Bearer user-a")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", linkKey("exh-cancel"))
	rec := httptest.NewRecorder()
	x.h.ServeHTTP(rec, req)
	events, _ := probe.disarm()
	if ctx.Err() == nil {
		t.Fatalf("the request was never cancelled (API pool events %v)", events)
	}
	if rows := x.exhRows(t, x.a.id); len(rows) != 1 {
		t.Fatalf("%d entries after a cancelled request (status %d: %s), want 1", len(rows), rec.Code, rec.Body)
	}
	if n := x.exhRecords(t); n != 9 {
		t.Errorf("%d records, want 9", n)
	}
}

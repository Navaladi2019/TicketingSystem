package isolation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/metric/noop"

	"github.com/Navaladi2019/TicketingSystem/api"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/idempotency"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
)

// idempotencyPaths are the CI-8 idempotency cases of T22 (TICK-76): the
// middleware against the suite's real nava_app role and idempotency_keys
// table. They use keys no seed row has and delete what they insert.
func idempotencyPaths() []Path {
	dbP := func(pk poolKind, name string, fn func(context.Context, *idemEnv) error) Path {
		return Path{Name: "idempotency: " + name, Case: "CI-8", DB: func(ctx context.Context, c Cluster) error {
			e, err := newIdemEnv(c, pk)
			if err != nil {
				return err
			}
			defer e.cleanup(ctx)
			return fn(ctx, e)
		}}
	}
	db := func(name string, fn func(context.Context, *idemEnv) error) Path { return dbP(poolKind{}, name, fn) }
	// N+1 takeovers of N+1 abandoned keys on a pool of N (T22a, TICK-212).
	// Every handler waits until all of them have started, then needs a
	// connection of its own: if a takeover held a connection while its
	// handler ran, the handlers could not all start, or could not get one,
	// and the case would time out. Through PgBouncer the pool is N clients on
	// one server connection, as R nava-core replicas share default_pool_size.
	overflow := func(pk poolKind, how string) Path {
		return dbP(pk, fmt.Sprintf("%d concurrent takeovers on a pool of %d (%s) all finish, each handler once", pk.conns+1, pk.conns, how),
			func(ctx context.Context, e *idemEnv) error {
				n := int(pk.conns) + 1
				keys := make([]string, n)
				for i := range keys {
					keys[i] = fmt.Sprintf("k-overflow-%d", i)
					if err := e.seedInFlight(ctx, e.tenantA, e.principalA, keys[i], `{"a":1}`, "2 minutes"); err != nil {
						return err
					}
				}
				var arrived atomic.Int64
				all := make(chan struct{})
				var failed atomic.Value
				h := e.handler(func() {
					if arrived.Add(1) == int64(n) {
						close(all)
					}
					select {
					case <-all:
					case <-time.After(15 * time.Second):
						failed.Store(fmt.Sprintf("only %d of %d handlers ran at once", arrived.Load(), n))
						return
					}
					qctx, cancel := context.WithTimeout(ctx, 15*time.Second)
					defer cancel()
					if _, err := e.pool.Exec(qctx, "SELECT 1"); err != nil {
						failed.Store("a handler could not get a connection: " + err.Error())
					}
				})
				recs := make([]*httptest.ResponseRecorder, n)
				var wg sync.WaitGroup
				for i, k := range keys {
					wg.Go(func() { recs[i] = e.do(h, e.tenantA, e.principalA, k, `{"a":1}`) })
				}
				wg.Wait()
				if f := failed.Load(); f != nil {
					return fmt.Errorf("%s", f)
				}
				for i, rec := range recs {
					if rec.Code != http.StatusCreated {
						return fmt.Errorf("takeover of %s = %d %s, want 201", keys[i], rec.Code, bodyOf(rec))
					}
				}
				if err := e.executed(int64(n)); err != nil {
					return err
				}
				return e.completed(ctx, keys...)
			})
	}
	return []Path{
		overflow(poolKind{conns: 4}, "direct"),
		overflow(poolKind{conns: 4, pooled: true}, "through PgBouncer, one server connection"),
		// Both takeovers read the row as abandoned, then queue on its row
		// lock behind a transaction the case holds open, so their UPDATEs
		// race for certain. The first commits a fresh lease; the second
		// re-checks the row after that commit, matches nothing and gets 409.
		dbP(poolKind{conns: 4}, "two concurrent takeovers of one abandoned row run the handler once, the other gets 409",
			func(ctx context.Context, e *idemEnv) error {
				const key = "k-takeover-race"
				if err := e.seedInFlight(ctx, e.tenantA, e.principalA, key, `{"a":1}`, "2 minutes"); err != nil {
					return err
				}
				holder, err := e.c.Super()
				if err != nil {
					return err
				}
				tx, err := holder.Begin(ctx)
				if err != nil {
					return err
				}
				defer func() { _ = tx.Rollback(ctx) }()
				if _, err := tx.Exec(ctx, "SELECT 1 FROM idempotency_keys WHERE tenant_id = $1 AND principal_id = $2 AND key = $3 FOR UPDATE",
					e.tenantA, e.principalA, key); err != nil {
					return err
				}
				h := e.handler(nil)
				recs := make(chan *httptest.ResponseRecorder, 2)
				for range 2 {
					go func() { recs <- e.do(h, e.tenantA, e.principalA, key, `{"a":1}`) }()
				}
				if err := e.waitLockWaiters(ctx, 2); err != nil {
					_ = tx.Rollback(ctx)
					<-recs
					<-recs
					return err
				}
				if err := tx.Commit(ctx); err != nil {
					return err
				}
				codes := map[int]int{}
				for range 2 {
					rec := <-recs
					codes[rec.Code]++
					if rec.Code == http.StatusConflict && e.code(rec) != problem.IdempotencyInFlight {
						return fmt.Errorf("loser = 409 %s, want idempotency_in_flight", bodyOf(rec))
					}
				}
				if codes[http.StatusCreated] != 1 || codes[http.StatusConflict] != 1 {
					return fmt.Errorf("takeovers = %v, want one 201 and one 409", codes)
				}
				return e.executed(1)
			}),
		// T22's semaphore let one tenant's takeovers fill every slot and turn
		// another tenant's away. Takeovers now hold nothing while the handler
		// runs, so tenant A's stuck handlers leave tenant B's takeover alone.
		dbP(poolKind{conns: 4}, "a tenant's stuck takeovers do not turn another tenant's takeover away",
			func(ctx context.Context, e *idemEnv) error {
				const stuck = 3 // over T22's MaxConns/4 = 1 slot
				for i := range stuck {
					if err := e.seedInFlight(ctx, e.tenantA, e.principalA, fmt.Sprintf("k-fair-a-%d", i), `{"a":1}`, "2 minutes"); err != nil {
						return err
					}
				}
				if err := e.seedInFlight(ctx, e.tenantB, e.principalA, "k-fair-b", `{"a":1}`, "2 minutes"); err != nil {
					return err
				}
				release := make(chan struct{})
				startedA := make(chan struct{}, stuck)
				h := e.handler(nil)
				blockA := e.handler(func() { startedA <- struct{}{}; <-release })
				var wg sync.WaitGroup
				recsA := make([]*httptest.ResponseRecorder, stuck)
				for i := range stuck {
					wg.Go(func() { recsA[i] = e.do(blockA, e.tenantA, e.principalA, fmt.Sprintf("k-fair-a-%d", i), `{"a":1}`) })
				}
				for i := range stuck {
					select {
					case <-startedA:
					case <-time.After(15 * time.Second):
						close(release)
						wg.Wait()
						return fmt.Errorf("only %d of tenant A's %d takeovers ran at once", i, stuck)
					}
				}
				b := e.do(h, e.tenantB, e.principalA, "k-fair-b", `{"a":1}`)
				close(release)
				wg.Wait()
				if b.Code != http.StatusCreated {
					return fmt.Errorf("tenant B's takeover while tenant A's are running = %d %s, want 201", b.Code, bodyOf(b))
				}
				for i, rec := range recsA {
					if rec.Code != http.StatusCreated {
						return fmt.Errorf("tenant A's takeover %d = %d %s, want 201", i, rec.Code, bodyOf(rec))
					}
				}
				return e.executed(stuck + 1)
			}),
		// The principal is an application predicate (there is no
		// app.principal_id setting): another user of the same tenant sending
		// the same key gets a row of its own and never moves the first user's
		// lease.
		db("another principal's request with the same key leaves the abandoned row's lease alone", func(ctx context.Context, e *idemEnv) error {
			const key = "k-lease-principal"
			if err := e.seedInFlight(ctx, e.tenantA, e.principalA, key, `{"a":1}`, "2 minutes"); err != nil {
				return err
			}
			before, err := e.lease(ctx, e.tenantA, e.principalA, key)
			if err != nil {
				return err
			}
			h := e.handler(nil)
			if rec := e.do(h, e.tenantA, uuid.New(), key, `{"a":1}`); rec.Code != http.StatusCreated {
				return fmt.Errorf("other principal = %d %s, want 201 (its own key)", rec.Code, bodyOf(rec))
			}
			after, err := e.lease(ctx, e.tenantA, e.principalA, key)
			if err != nil {
				return err
			}
			if !after.Equal(before) {
				return fmt.Errorf("first principal's lease moved from %s to %s", before, after)
			}
			return e.executed(1)
		}),
		db("two concurrent identical requests execute once", func(ctx context.Context, e *idemEnv) error {
			release := make(chan struct{})
			started := make(chan struct{})
			var once sync.Once
			h := e.handler(func() { once.Do(func() { close(started) }); <-release })
			var wg sync.WaitGroup
			first := make(chan *httptest.ResponseRecorder, 1)
			wg.Go(func() { first <- e.do(h, e.tenantA, e.principalA, "k-concurrent", `{"a":1}`) })
			<-started
			second := e.do(h, e.tenantA, e.principalA, "k-concurrent", `{"a":1}`)
			close(release)
			wg.Wait()
			if got := (<-first).Code; got != http.StatusCreated {
				return fmt.Errorf("first request = %d, want 201", got)
			}
			if second.Code != http.StatusConflict || e.code(second) != problem.IdempotencyInFlight {
				return fmt.Errorf("concurrent duplicate = %d %s, want 409 idempotency_in_flight", second.Code, bodyOf(second))
			}
			return e.executed(1)
		}),
		db("same key, different body is 409", func(ctx context.Context, e *idemEnv) error {
			h := e.handler(nil)
			e.do(h, e.tenantA, e.principalA, "k-diff", `{"a":1}`)
			rec := e.do(h, e.tenantA, e.principalA, "k-diff", `{"a":2}`)
			if rec.Code != http.StatusConflict || e.code(rec) != problem.IdempotencyKeyReuse {
				return fmt.Errorf("different body = %d %s, want 409 idempotency_key_reuse", rec.Code, bodyOf(rec))
			}
			return e.executed(1)
		}),
		db("tenant B reusing tenant A's key is independent", func(ctx context.Context, e *idemEnv) error {
			h := e.handler(nil)
			e.do(h, e.tenantA, e.principalA, "k-tenants", `{"a":1}`)
			rec := e.do(h, e.tenantB, e.principalA, "k-tenants", `{"a":2}`)
			if rec.Code != http.StatusCreated {
				return fmt.Errorf("tenant B = %d %s, want 201 (its own key)", rec.Code, bodyOf(rec))
			}
			return e.executed(2)
		}),
		// The leak case: an identical request (same key, same body) from
		// another tenant must run on its own, never replay tenant A's stored
		// response.
		db("tenant B replaying tenant A's key and body gets no stored response", func(ctx context.Context, e *idemEnv) error {
			h := e.handler(nil)
			a := e.do(h, e.tenantA, e.principalA, "k-tenant-replay", `{"a":1}`)
			b := e.do(h, e.tenantB, e.principalA, "k-tenant-replay", `{"a":1}`)
			if b.Code != http.StatusCreated || b.Header().Get(idempotency.ReplayedHeader) != "" || bytes.Equal(bodyBytes(b), bodyBytes(a)) {
				return fmt.Errorf("tenant B = %d %q replayed=%q, tenant A = %q; want B's own execution",
					b.Code, bodyOf(b), b.Header().Get(idempotency.ReplayedHeader), bodyOf(a))
			}
			return e.executed(2)
		}),
		// RLS scopes idempotency_keys by tenant only; the principal is an
		// application predicate. Another user of the same tenant must not get
		// the first user's stored response.
		db("another principal of the same tenant gets no stored response", func(ctx context.Context, e *idemEnv) error {
			h := e.handler(nil)
			other := uuid.New()
			a := e.do(h, e.tenantA, e.principalA, "k-principal-replay", `{"a":1}`)
			b := e.do(h, e.tenantA, other, "k-principal-replay", `{"a":1}`)
			if b.Code != http.StatusCreated || b.Header().Get(idempotency.ReplayedHeader) != "" || bytes.Equal(bodyBytes(b), bodyBytes(a)) {
				return fmt.Errorf("other principal = %d %q replayed=%q, first = %q; want its own execution",
					b.Code, bodyOf(b), b.Header().Get(idempotency.ReplayedHeader), bodyOf(a))
			}
			return e.executed(2)
		}),
		db("a replay is byte-identical", func(ctx context.Context, e *idemEnv) error {
			h := e.handler(nil)
			a := e.do(h, e.tenantA, e.principalA, "k-replay", `{"b": 2, "a": 1}`)
			// Reordered, reformatted JSON is the same request.
			b := e.do(h, e.tenantA, e.principalA, "k-replay", `{"a":1,"b":2}`)
			if b.Code != a.Code || b.Header().Get("Content-Type") != a.Header().Get("Content-Type") || !bytes.Equal(bodyBytes(b), bodyBytes(a)) {
				return fmt.Errorf("replay = %d %q %q, original = %d %q %q",
					b.Code, b.Header().Get("Content-Type"), bodyOf(b), a.Code, a.Header().Get("Content-Type"), bodyOf(a))
			}
			if b.Header().Get(idempotency.ReplayedHeader) != "true" {
				return fmt.Errorf("replay lacks %s", idempotency.ReplayedHeader)
			}
			if ct := a.Header().Get("Content-Type"); ct != "application/json" {
				return fmt.Errorf("first response Content-Type = %q, want the canonical application/json", ct)
			}
			// A stored problem keeps its status, body and problem+json type.
			e1 := e.do(h, e.tenantA, e.principalA, "k-replay-4xx", `{"fail":true}`)
			e2 := e.do(h, e.tenantA, e.principalA, "k-replay-4xx", `{"fail":true}`)
			if e1.Code != http.StatusBadRequest || e2.Code != e1.Code ||
				e2.Header().Get("Content-Type") != problem.ContentType || !bytes.Equal(bodyBytes(e2), bodyBytes(e1)) {
				return fmt.Errorf("replayed error = %d %q %q, original = %d %q", e2.Code, e2.Header().Get("Content-Type"), bodyOf(e2), e1.Code, bodyOf(e1))
			}
			return e.executed(2)
		}),
		db("an in_flight row past the 60 s lease is retaken, a younger one is not", func(ctx context.Context, e *idemEnv) error {
			h := e.handler(nil)
			if err := e.seedInFlight(ctx, e.tenantA, e.principalA, "k-lease-old", `{"a":1}`, "2 minutes"); err != nil {
				return err
			}
			if err := e.seedInFlight(ctx, e.tenantA, e.principalA, "k-lease-young", `{"a":1}`, "10 seconds"); err != nil {
				return err
			}
			young := e.do(h, e.tenantA, e.principalA, "k-lease-young", `{"a":1}`)
			if young.Code != http.StatusConflict || e.code(young) != problem.IdempotencyInFlight {
				return fmt.Errorf("young in_flight = %d %s, want 409 idempotency_in_flight", young.Code, bodyOf(young))
			}
			if err := e.executed(0); err != nil {
				return err
			}
			old := e.do(h, e.tenantA, e.principalA, "k-lease-old", `{"a":1}`)
			if old.Code != http.StatusCreated {
				return fmt.Errorf("abandoned in_flight = %d %s, want 201 (retaken)", old.Code, bodyOf(old))
			}
			again := e.do(h, e.tenantA, e.principalA, "k-lease-old", `{"a":1}`)
			if again.Code != http.StatusCreated || !bytes.Equal(bodyBytes(again), bodyBytes(old)) {
				return fmt.Errorf("replay after retake = %d %s, want the retaken response", again.Code, bodyOf(again))
			}
			return e.executed(1)
		}),
	}
}

// idemEnv is a middleware on the suite's nava_app pool and two fresh tenants.
type idemEnv struct {
	c                Cluster
	pool             *pgxpool.Pool
	mw               *idempotency.Middleware
	reg              *problem.Registry
	tenantA, tenantB uuid.UUID
	principalA       uuid.UUID
	runs             atomic.Int64
}

// poolKind is the middleware's nava_app pool: the zero value is one
// connection through PgBouncer; conns > 0 is that many connections, direct,
// or through PgBouncer's one server connection when pooled.
type poolKind struct {
	conns  int32
	pooled bool
}

func newIdemEnv(c Cluster, pk poolKind) (*idemEnv, error) {
	var pool *pgxpool.Pool
	var err error
	switch {
	case pk.conns == 0:
		pool, err = c.Pool("nava_app")
	case pk.pooled:
		pool, err = c.PooledSize("nava_app", pk.conns)
	default:
		pool, err = c.PoolSize("nava_app", pk.conns)
	}
	if err != nil {
		return nil, err
	}
	data, err := api.Files.ReadFile("errors.yaml")
	if err != nil {
		return nil, err
	}
	reg, err := problem.LoadRegistry(data)
	if err != nil {
		return nil, err
	}
	mw, err := idempotency.New(pool, reg, slog.New(slog.DiscardHandler), noop.NewMeterProvider())
	if err != nil {
		return nil, err
	}
	return &idemEnv{c: c, pool: pool, mw: mw, reg: reg, tenantA: uuid.New(), tenantB: uuid.New(), principalA: uuid.New()}, nil
}

// handler is a write: 201 with a body that is unique per execution, or a 400
// problem when the body says fail. before runs first, if set.
func (e *idemEnv) handler(before func()) http.Handler {
	return e.mw.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if before != nil {
			before()
		}
		body, _ := io.ReadAll(r.Body)
		n := e.runs.Add(1)
		if bytes.Contains(body, []byte(`"fail"`)) {
			e.reg.Write(w, r, problem.ValidationFailed)
			return
		}
		// Not the canonical type: the middleware must normalise it on the
		// first run, so a replay is byte-identical.
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusCreated)
		_, _ = fmt.Fprintf(w, `{"run":%d}`, n)
	}))
}

func (e *idemEnv) do(h http.Handler, tenant, principal uuid.UUID, key, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/things", bytes.NewBufferString(body))
	req.Header.Set(idempotency.Header, key)
	req = req.WithContext(identity.NewContext(req.Context(), identity.Principal{TenantID: tenant, PrincipalID: principal}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func (e *idemEnv) code(rec *httptest.ResponseRecorder) problem.Code {
	var b problem.Body
	_ = json.Unmarshal(bodyBytes(rec), &b)
	return b.Code
}

func (e *idemEnv) executed(want int64) error {
	if got := e.runs.Load(); got != want {
		return fmt.Errorf("handler ran %d times, want %d", got, want)
	}
	return nil
}

// seedInFlight inserts an in_flight row created age ago, with the 60 s lease
// it was given then, as a request that died would have left it. The hash is
// the one the middleware computes for the same request.
func (e *idemEnv) seedInFlight(ctx context.Context, tenant, principal uuid.UUID, key, body, age string) error {
	conn, err := e.c.Super()
	if err != nil {
		return err
	}
	_, err = conn.Exec(ctx, `
INSERT INTO idempotency_keys (tenant_id, principal_id, key, request_hash, created_at, lease_until, expires_at)
VALUES ($1, $2, $3, $4, now() - $5::interval, now() - $5::interval + interval '60 seconds', now() + interval '1 day')`,
		tenant, principal, key, idempotency.RequestHash(http.MethodPost, "/api/v1/things", []byte(body)), age)
	return err
}

// lease reads a row's lease_until as the superuser.
func (e *idemEnv) lease(ctx context.Context, tenant, principal uuid.UUID, key string) (time.Time, error) {
	conn, err := e.c.Super()
	if err != nil {
		return time.Time{}, err
	}
	var t time.Time
	err = conn.QueryRow(ctx, "SELECT lease_until FROM idempotency_keys WHERE tenant_id = $1 AND principal_id = $2 AND key = $3",
		tenant, principal, key).Scan(&t)
	return t, err
}

// completed checks that tenant A's principal stored a response for every key.
func (e *idemEnv) completed(ctx context.Context, keys ...string) error {
	conn, err := e.c.Super()
	if err != nil {
		return err
	}
	var n int
	if err := conn.QueryRow(ctx, `
SELECT count(*) FROM idempotency_keys
 WHERE tenant_id = $1 AND principal_id = $2 AND key = ANY($3) AND state = 'completed'`,
		e.tenantA, e.principalA, keys).Scan(&n); err != nil {
		return err
	}
	if n != len(keys) {
		return fmt.Errorf("%d of %d keys completed", n, len(keys))
	}
	return nil
}

// waitLockWaiters waits until n nava_app backends wait on a lock.
func (e *idemEnv) waitLockWaiters(ctx context.Context, n int) error {
	conn, err := e.c.Super()
	if err != nil {
		return err
	}
	deadline := time.Now().Add(15 * time.Second)
	for {
		var got int
		if err := conn.QueryRow(ctx, `
SELECT count(*) FROM pg_stat_activity WHERE usename = 'nava_app' AND wait_event_type = 'Lock'`).Scan(&got); err != nil {
			return err
		}
		if got >= n {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("%d of %d takeovers queued on the row lock", got, n)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// cleanup deletes every row the case made, as the superuser (nava_app cannot
// delete).
func (e *idemEnv) cleanup(ctx context.Context) {
	if conn, err := e.c.Super(); err == nil {
		_, _ = conn.Exec(ctx, `DELETE FROM idempotency_keys WHERE tenant_id = ANY($1)`, []uuid.UUID{e.tenantA, e.tenantB})
	}
}

func bodyBytes(rec *httptest.ResponseRecorder) []byte { return rec.Body.Bytes() }

func bodyOf(rec *httptest.ResponseRecorder) string { return rec.Body.String() }

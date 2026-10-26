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
	dbN := func(conns int32, name string, fn func(context.Context, *idemEnv) error) Path {
		return Path{Name: "idempotency: " + name, Case: "CI-8", DB: func(ctx context.Context, c Cluster) error {
			e, err := newIdemEnv(c, conns)
			if err != nil {
				return err
			}
			defer e.cleanup(ctx)
			return fn(ctx, e)
		}}
	}
	db := func(name string, fn func(context.Context, *idemEnv) error) Path { return dbN(0, name, fn) }
	// A retake holds a connection while its handler runs, so two concurrent
	// retakers need a pool with room for both (PoolSize, direct, since the
	// suite's PgBouncer gives a role one server connection). With 8
	// connections two retakes may run (MaxConns/4), so the second is stopped
	// by the row lock; with 4 only one may, so the second is stopped by the
	// limit.
	retakers := func(conns int32, name, key1, key2 string, check func(ctx context.Context, e *idemEnv, h http.Handler, second *httptest.ResponseRecorder, release func(), first <-chan *httptest.ResponseRecorder) error) Path {
		return dbN(conns, name, func(ctx context.Context, e *idemEnv) error {
			for i, k := range []string{key1, key2} {
				if i == 1 && k == key1 {
					break
				}
				if err := e.seedInFlight(ctx, k, `{"a":1}`, "2 minutes"); err != nil {
					return err
				}
			}
			release := make(chan struct{})
			started := make(chan struct{})
			var once sync.Once
			h := e.handler(func() { once.Do(func() { close(started) }); <-release })
			first := make(chan *httptest.ResponseRecorder, 1)
			go func() { first <- e.do(h, e.tenantA, e.principalA, key1, `{"a":1}`) }()
			<-started
			second := e.do(h, e.tenantA, e.principalA, key2, `{"a":1}`)
			var once2 sync.Once
			return check(ctx, e, h, second, func() { once2.Do(func() { close(release) }) }, first)
		})
	}
	busy := func(e *idemEnv, second *httptest.ResponseRecorder) error {
		if second.Code != http.StatusConflict || e.code(second) != problem.IdempotencyInFlight {
			return fmt.Errorf("second retaker = %d %s, want 409 idempotency_in_flight", second.Code, bodyOf(second))
		}
		return nil
	}
	finish := func(release func(), first <-chan *httptest.ResponseRecorder) error {
		release()
		if got := (<-first).Code; got != http.StatusCreated {
			return fmt.Errorf("first retaker = %d, want 201", got)
		}
		return nil
	}
	// Both retakers aim at one stale row: only the row lock can stop the second.
	sameRow := func(ctx context.Context, e *idemEnv, _ http.Handler, second *httptest.ResponseRecorder, release func(), first <-chan *httptest.ResponseRecorder) error {
		if err := busy(e, second); err != nil {
			release()
			<-first
			return err
		}
		if err := finish(release, first); err != nil {
			return err
		}
		return e.executed(1)
	}
	// Two stale rows: only the limit can stop the second, and once the first
	// retaker is done the same request goes through.
	limit := func(ctx context.Context, e *idemEnv, h http.Handler, second *httptest.ResponseRecorder, release func(), first <-chan *httptest.ResponseRecorder) error {
		if err := busy(e, second); err != nil {
			release()
			<-first
			return err
		}
		if err := finish(release, first); err != nil {
			return err
		}
		if err := e.executed(1); err != nil {
			return err
		}
		retry := e.do(h, e.tenantA, e.principalA, "k-retake-limit-2", `{"a":1}`)
		if retry.Code != http.StatusCreated {
			return fmt.Errorf("second key after the first retaker finished = %d %s, want 201", retry.Code, bodyOf(retry))
		}
		return e.executed(2)
	}
	return []Path{
		retakers(8, "a second retaker of one stale row (row lock) gets 409 and the handler runs once",
			"k-two-retakers", "k-two-retakers", sameRow),
		retakers(4, "a retaker over the concurrent-retake limit gets 409, then succeeds once the first is done",
			"k-retake-limit-1", "k-retake-limit-2", limit),
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
			if err := e.seedInFlight(ctx, "k-lease-old", `{"a":1}`, "2 minutes"); err != nil {
				return err
			}
			if err := e.seedInFlight(ctx, "k-lease-young", `{"a":1}`, "10 seconds"); err != nil {
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
	mw               *idempotency.Middleware
	reg              *problem.Registry
	tenantA, tenantB uuid.UUID
	principalA       uuid.UUID
	runs             atomic.Int64
}

func newIdemEnv(c Cluster, conns int32) (*idemEnv, error) {
	var pool *pgxpool.Pool
	var err error
	if conns == 0 {
		pool, err = c.Pool("nava_app") // through PgBouncer, one connection
	} else {
		pool, err = c.PoolSize("nava_app", conns)
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
	return &idemEnv{c: c, mw: mw, reg: reg, tenantA: uuid.New(), tenantB: uuid.New(), principalA: uuid.New()}, nil
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

// seedInFlight inserts an in_flight row for tenant A created age ago, as a
// request that died would have left it. The hash is the one the middleware
// computes for the same request.
func (e *idemEnv) seedInFlight(ctx context.Context, key, body, age string) error {
	conn, err := e.c.Super()
	if err != nil {
		return err
	}
	_, err = conn.Exec(ctx, `
INSERT INTO idempotency_keys (tenant_id, principal_id, key, request_hash, created_at, expires_at)
VALUES ($1, $2, $3, $4, now() - $5::interval, now() + interval '1 day')`,
		e.tenantA, e.principalA, key, idempotency.RequestHash(http.MethodPost, "/api/v1/things", []byte(body)), age)
	return err
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

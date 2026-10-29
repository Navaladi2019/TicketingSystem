// Package ratelimit is the core API's per-tenant rate limit (ADR-001,
// PLAN-001 T24). One token bucket (golang.org/x/time/rate) per tenant, keyed
// only on the tenant the authentication middleware resolved from the token
// (identity.FromContext), never on a header, query or body value. A tenant
// over its limit gets 429 rate_limited with Retry-After; other tenants have
// their own buckets and are unaffected.
//
// Limits are per replica: the buckets live in this process's memory, so N
// replicas of nava-core allow N times the configured rate for one tenant. The
// Limiter interface lets a shared store replace the in-memory one later.
//
// A bucket idle for IdleTTL is evicted, so the map is bounded by the tenants
// active within that window, not by every tenant ever seen.
//
// No metric carries a tenant attribute (unbounded cardinality). The throttled
// tenant is a log field only.
package ratelimit

import (
	"context"
	"errors"
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"golang.org/x/time/rate"

	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
)

// Defaults for a zero Config field.
const (
	DefaultRate    = 50.0 // requests per second per tenant, per replica
	DefaultBurst   = 100
	DefaultIdleTTL = 10 * time.Minute
)

// Config sets the bucket of every tenant.
type Config struct {
	Rate    float64       // tokens per second; <= 0 means DefaultRate
	Burst   int           // bucket size; <= 0 means DefaultBurst
	IdleTTL time.Duration // idle age after which a bucket is evicted; <= 0 means DefaultIdleTTL
}

func (c Config) withDefaults() Config {
	if c.Rate <= 0 {
		c.Rate = DefaultRate
	}
	if c.Burst <= 0 {
		c.Burst = DefaultBurst
	}
	if c.IdleTTL <= 0 {
		c.IdleTTL = DefaultIdleTTL
	}
	return c
}

// Decision is the outcome of one Allow call.
type Decision struct {
	Allowed    bool
	RetryAfter time.Duration // set when not Allowed
}

// Limiter decides whether a tenant may make one more request now. A shared
// store (Redis, Postgres) can implement it later; the middleware depends only
// on this interface.
type Limiter interface {
	Allow(ctx context.Context, tenant uuid.UUID) (Decision, error)
}

type entry struct {
	lim  *rate.Limiter
	last time.Time
}

// Memory is the in-process Limiter.
type Memory struct {
	cfg Config
	now func() time.Time

	mu        sync.Mutex
	buckets   map[uuid.UUID]*entry
	lastSweep time.Time
}

// NewMemory returns an in-memory Limiter. now may be nil (time.Now).
func NewMemory(cfg Config, now func() time.Time) *Memory {
	if now == nil {
		now = time.Now
	}
	return &Memory{cfg: cfg.withDefaults(), now: now, buckets: map[uuid.UUID]*entry{}}
}

// Allow takes one token from the tenant's bucket.
func (m *Memory) Allow(_ context.Context, tenant uuid.UUID) (Decision, error) {
	now := m.now()
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sweep(now)
	e, ok := m.buckets[tenant]
	if !ok {
		e = &entry{lim: rate.NewLimiter(rate.Limit(m.cfg.Rate), m.cfg.Burst)}
		m.buckets[tenant] = e
	}
	e.last = now
	r := e.lim.ReserveN(now, 1)
	if !r.OK() {
		return Decision{RetryAfter: time.Second}, nil
	}
	if d := r.DelayFrom(now); d > 0 {
		r.CancelAt(now) // a refused request does not spend a token
		return Decision{RetryAfter: d}, nil
	}
	return Decision{Allowed: true}, nil
}

// Len is the number of live buckets.
func (m *Memory) Len() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.buckets)
}

// sweep removes buckets idle past IdleTTL. It runs at most once per half TTL,
// so the cost is amortised over requests and needs no goroutine to stop.
// m.mu is held.
func (m *Memory) sweep(now time.Time) {
	if now.Sub(m.lastSweep) < m.cfg.IdleTTL/2 {
		return
	}
	m.lastSweep = now
	for id, e := range m.buckets {
		if now.Sub(e.last) >= m.cfg.IdleTTL {
			delete(m.buckets, id)
		}
	}
}

// Middleware limits the requests it wraps by tenant. It must run after the
// authentication middleware: a request with no resolved principal is a wiring
// bug and gets 500.
type Middleware struct {
	limiter Limiter
	reg     *problem.Registry
	log     *slog.Logger
}

// New returns the middleware. All arguments are required.
func New(l Limiter, reg *problem.Registry, log *slog.Logger) (*Middleware, error) {
	if l == nil || reg == nil || log == nil {
		return nil, errors.New("ratelimit: limiter, registry and logger are required")
	}
	return &Middleware{limiter: l, reg: reg, log: log}, nil
}

var errNoPrincipal = errors.New("ratelimit: no resolved principal in the request context")

// Handler wraps next.
func (m *Middleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		p, ok := identity.FromContext(ctx)
		if !ok {
			m.reg.WriteError(w, r, m.log, problem.InternalError, errNoPrincipal, slog.String("route", r.Pattern))
			return
		}
		d, err := m.limiter.Allow(ctx, p.TenantID)
		if err != nil {
			m.reg.WriteError(w, r, m.log, problem.InternalError, err, slog.String("route", r.Pattern))
			return
		}
		if !d.Allowed {
			secs := max(1, int(math.Ceil(d.RetryAfter.Seconds())))
			// tenant_id is added from ctx by the logging ContextHandler.
			m.log.InfoContext(ctx, "request rate limited", slog.Int("retry_after_s", secs))
			w.Header().Set("Retry-After", strconv.Itoa(secs))
			m.reg.Write(w, r, problem.RateLimited)
			return
		}
		next.ServeHTTP(w, r)
	})
}

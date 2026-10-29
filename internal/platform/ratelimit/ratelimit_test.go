package ratelimit_test

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Navaladi2019/TicketingSystem/api"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/logging"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/ratelimit"
)

//nolint:gochecknoglobals // fixed test ids, never written
var (
	tenantA = uuid.MustParse("0a000000-0000-4000-8000-00000000000a")
	tenantB = uuid.MustParse("0b000000-0000-4000-8000-00000000000b")
)

type clock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *clock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *clock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
}

func newClock() *clock { return &clock{t: time.Unix(1_800_000_000, 0)} }

func registry(t *testing.T) *problem.Registry {
	t.Helper()
	data, err := api.Files.ReadFile("errors.yaml")
	if err != nil {
		t.Fatal(err)
	}
	reg, err := problem.LoadRegistry(data)
	if err != nil {
		t.Fatal(err)
	}
	return reg
}

func TestMemoryIsolatesTenantsAndReportsRetryAfter(t *testing.T) {
	t.Parallel()
	c := newClock()
	m := ratelimit.NewMemory(ratelimit.Config{Rate: 1, Burst: 2, IdleTTL: time.Minute}, c.now)
	for i := range 2 {
		if d, _ := m.Allow(t.Context(), tenantA); !d.Allowed {
			t.Fatalf("burst request %d refused", i)
		}
	}
	d, err := m.Allow(t.Context(), tenantA)
	if err != nil || d.Allowed || d.RetryAfter <= 0 || d.RetryAfter > time.Second {
		t.Fatalf("over-limit decision = %+v, %v; want refused with 0 < RetryAfter <= 1s", d, err)
	}
	if d, _ := m.Allow(t.Context(), tenantB); !d.Allowed {
		t.Error("tenant B refused because tenant A is flooded")
	}
	c.advance(time.Second)
	if d, _ := m.Allow(t.Context(), tenantA); !d.Allowed {
		t.Error("tenant A still refused after the bucket refilled")
	}
}

// A limiter idle past the eviction age is removed.
func TestMemoryEvictsIdleBuckets(t *testing.T) {
	t.Parallel()
	c := newClock()
	ttl := time.Minute
	m := ratelimit.NewMemory(ratelimit.Config{Rate: 1, Burst: 1, IdleTTL: ttl}, c.now)
	const tenants = 1000
	for range tenants {
		if _, err := m.Allow(t.Context(), uuid.New()); err != nil {
			t.Fatal(err)
		}
	}
	if got := m.Len(); got != tenants {
		t.Fatalf("Len = %d, want %d", got, tenants)
	}
	c.advance(ttl + time.Second)
	if _, err := m.Allow(t.Context(), tenantA); err != nil {
		t.Fatal(err)
	}
	if got := m.Len(); got != 1 {
		t.Errorf("Len after idle TTL = %d, want only the active tenant", got)
	}
}

// The map does not grow without bound under many tenants.
func TestMemoryMapStaysBoundedUnderManyTenants(t *testing.T) {
	t.Parallel()
	c := newClock()
	ttl := time.Minute
	m := ratelimit.NewMemory(ratelimit.Config{Rate: 1, Burst: 1, IdleTTL: ttl}, c.now)
	peak := 0
	// One new tenant per second for an hour: live buckets never exceed ~1.5 TTL.
	for range 3600 {
		_, _ = m.Allow(t.Context(), uuid.New())
		peak = max(peak, m.Len())
		c.advance(time.Second)
	}
	if limit := int(1.5*ttl.Seconds()) + 1; peak > limit {
		t.Errorf("peak buckets = %d, want <= %d", peak, limit)
	}
}

func TestActiveBucketIsNotEvicted(t *testing.T) {
	t.Parallel()
	c := newClock()
	m := ratelimit.NewMemory(ratelimit.Config{Rate: 0.001, Burst: 1, IdleTTL: time.Minute}, c.now)
	_, _ = m.Allow(t.Context(), tenantA)
	for range 5 {
		c.advance(40 * time.Second)
		if d, _ := m.Allow(t.Context(), tenantA); d.Allowed {
			t.Fatal("an active, over-limit tenant got a fresh bucket: eviction reset its limit")
		}
	}
}

// withTenant stands in for the authentication middleware.
func withTenant(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, err := uuid.Parse(r.Header.Get("X-Test-Resolved-Tenant"))
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}
		next.ServeHTTP(w, r.WithContext(identity.NewContext(r.Context(), identity.Principal{TenantID: id})))
	})
}

func newHandler(t *testing.T, l ratelimit.Limiter, log *slog.Logger) http.Handler {
	t.Helper()
	mw, err := ratelimit.New(l, registry(t), log)
	if err != nil {
		t.Fatal(err)
	}
	return withTenant(mw.Handler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})))
}

func do(t *testing.T, h http.Handler, tenant uuid.UUID) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/x", nil)
	if tenant != uuid.Nil {
		req.Header.Set("X-Test-Resolved-Tenant", tenant.String())
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestFloodedTenantGets429WithRetryAfterAndLog(t *testing.T) {
	t.Parallel()
	var logs strings.Builder
	log := slog.New(logging.NewContextHandler(slog.NewJSONHandler(&logs, nil)))
	m := ratelimit.NewMemory(ratelimit.Config{Rate: 1, Burst: 3}, newClock().now)
	h := newHandler(t, m, log)
	for i := range 3 {
		if rec := do(t, h, tenantA); rec.Code != http.StatusNoContent {
			t.Fatalf("request %d = %d, want 204", i, rec.Code)
		}
	}
	rec := do(t, h, tenantA)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", rec.Code)
	}
	if ra, err := strconv.Atoi(rec.Header().Get("Retry-After")); err != nil || ra < 1 {
		t.Errorf("Retry-After = %q, want a positive integer", rec.Header().Get("Retry-After"))
	}
	if ct := rec.Header().Get("Content-Type"); ct != problem.ContentType {
		t.Errorf("Content-Type = %q", ct)
	}
	var body problem.Body
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || body.Code != problem.RateLimited {
		t.Errorf("body = %s (%v), want code rate_limited", rec.Body, err)
	}
	if got := strings.Count(logs.String(), `"tenant_id"`); got != 1 {
		t.Errorf("throttle log has %d tenant_id fields, want 1: %s", got, logs.String())
	}
	if !strings.Contains(logs.String(), `"tenant_id":"`+tenantA.String()+`"`) {
		t.Errorf("throttle log lacks tenant_id: %s", logs.String())
	}
	if rec := do(t, h, tenantB); rec.Code != http.StatusNoContent {
		t.Errorf("tenant B = %d while A is flooded, want 204", rec.Code)
	}
}

// A flood from tenant A neither changes B's decisions nor spends B's tokens.
// The clock is fixed, so no bucket refills and the outcome is exact: B gets
// its burst and not one request more, whatever A sent before, during or after.
func TestFloodOfOneTenantDoesNotThrottleAnother(t *testing.T) {
	t.Parallel()
	const burst = 5
	m := ratelimit.NewMemory(ratelimit.Config{Rate: 1, Burst: burst}, newClock().now)
	h := newHandler(t, m, slog.New(slog.DiscardHandler))
	allowedB := func(n int) {
		t.Helper()
		for range n {
			if rec := do(t, h, tenantB); rec.Code != http.StatusNoContent {
				t.Fatalf("tenant B within its budget = %d, want 204", rec.Code)
			}
		}
	}
	allowedB(2)

	var wg sync.WaitGroup
	var sent, allowedA atomic.Int64
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 250 {
				sent.Add(1)
				if do(t, h, tenantA).Code == http.StatusNoContent {
					allowedA.Add(1)
				}
			}
		}()
	}
	wg.Wait()

	if got := allowedA.Load(); got != burst {
		t.Fatalf("tenant A allowed %d of %d requests, want exactly its burst %d", got, sent.Load(), burst)
	}
	allowedB(burst - 2) // the rest of B's budget is intact
	if rec := do(t, h, tenantB); rec.Code != http.StatusTooManyRequests {
		t.Errorf("tenant B over its own budget = %d, want 429", rec.Code)
	}
}

type failing struct{}

func (failing) Allow(context.Context, uuid.UUID) (ratelimit.Decision, error) {
	return ratelimit.Decision{}, errors.New("store down: secret-host-detail")
}

func TestLimiterErrorIs500WithoutDetail(t *testing.T) {
	t.Parallel()
	rec := do(t, newHandler(t, failing{}, slog.New(slog.DiscardHandler)), tenantA)
	if rec.Code != http.StatusInternalServerError || strings.Contains(rec.Body.String(), "secret-host-detail") {
		t.Errorf("got %d %s, want 500 without the cause", rec.Code, rec.Body)
	}
}

func TestNoPrincipalIs500NotUnlimited(t *testing.T) {
	t.Parallel()
	m := ratelimit.NewMemory(ratelimit.Config{}, nil)
	rec := do(t, newHandler(t, m, slog.New(slog.DiscardHandler)), uuid.Nil)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("no principal = %d, want 500", rec.Code)
	}
}

func TestNewRequiresArguments(t *testing.T) {
	t.Parallel()
	if _, err := ratelimit.New(nil, registry(t), slog.New(slog.DiscardHandler)); err == nil {
		t.Error("nil limiter accepted")
	}
}

func TestConfigDefaults(t *testing.T) {
	t.Parallel()
	m := ratelimit.NewMemory(ratelimit.Config{}, nil)
	for i := range ratelimit.DefaultBurst {
		if d, _ := m.Allow(t.Context(), tenantA); !d.Allowed {
			t.Fatalf("request %d refused under default burst", i)
		}
	}
	if d, _ := m.Allow(t.Context(), tenantA); d.Allowed {
		t.Error("request beyond default burst allowed")
	}
}

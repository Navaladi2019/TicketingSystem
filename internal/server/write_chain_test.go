package server_test

// The write-chain tripwire (TICK-326, regression R2). Every tenant write route
// of the contract (POST, PUT, PATCH or DELETE outside /api/v1/operator/) must
// run authn -> ratelimit -> idempotency, in that order, before anything else
// reads the request. The routes come from the embedded api/openapi.yaml, the
// same contract the generated router mounts; there is no hand-written list.
//
// The check is behavioural: it drives the real handler and looks for each
// middleware's own signature, so it cannot be satisfied by naming a layer.
//
//   - P1, no token: 401 with WWW-Authenticate. The limiter is not asked and
//     the idempotency counter does not move (authn is outermost).
//   - P2, tenant A's token, limiter refusing, no Idempotency-Key: 429
//     rate_limited, the limiter asked for tenant A only, the idempotency
//     counter unmoved (ratelimit sits inside authn and outside idempotency).
//   - P3, tenant A's token, limiter allowing, no Idempotency-Key, a malformed
//     body: 400 validation_failed on /header/Idempotency-Key alone, and
//     nava.idempotency.requests{result="rejected"} up by one (the real
//     idempotency middleware answered, before the validator or the handler).
//
// Operator routes are out of scope: they carry no tenant, authenticate through
// the OperatorResolver and do their own idempotency (T21).

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"regexp"
	"slices"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Navaladi2019/TicketingSystem/api"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/apihttp"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/auth"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/idempotency"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/ratelimit"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
	"github.com/Navaladi2019/TicketingSystem/internal/server"
)

// newTripwireHandler builds the public handler the way cmd/nava-core does,
// with lim as the limiter. When the first tenant write route lands, mount the
// idempotency middleware here with the same option nava-core uses, built on
// tel.MeterProvider; until then such a route fails P3.
func newTripwireHandler(t *testing.T, tel *telemetry.Telemetry, lim ratelimit.Limiter) http.Handler {
	t.Helper()
	h, err := server.NewHandler(slog.New(slog.DiscardHandler), tel, twoTenants{}, stubPaths{}, server.WithLimiter(lim))
	if err != nil {
		t.Fatal(err)
	}
	return h
}

// TestTenantWriteRoutesMountAuthnRatelimitIdempotency is the tripwire. Every
// task that adds a tenant write route keeps it green.
func TestTenantWriteRoutesMountAuthnRatelimitIdempotency(t *testing.T) {
	t.Parallel()
	all, tenant := contractWriteOps(t)
	// The operator onboarding route is a write in the contract: a parser that
	// found no write at all would make the tripwire pass on nothing.
	if len(all) == 0 {
		t.Fatal("no write operation found in api/openapi.yaml; the enumeration is broken")
	}
	t.Logf("contract writes: %v; tenant writes checked: %v", all, tenant)
	for _, op := range tenant {
		t.Run(op.String(), func(t *testing.T) {
			t.Parallel()
			tel := newTripwireTelemetry(t)
			lim := &spyLimiter{}
			for _, f := range probeWriteChain(t, newTripwireHandler(t, tel, lim), tel, lim, op) {
				t.Errorf("%s: %s", op, f)
			}
		})
	}
}

// TestWriteChainTripwireFixtures proves the probe. The fixtures compose the
// real auth, ratelimit and idempotency middleware around one tenant write
// route; only the correct chain may pass.
func TestWriteChainTripwireFixtures(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		chain []string // outermost first
		pass  bool
	}{
		{"correct chain (positive control)", []string{"authn", "ratelimit", "idempotency"}, true},
		{"idempotency missing", []string{"authn", "ratelimit"}, false},
		{"idempotency before ratelimit", []string{"authn", "idempotency", "ratelimit"}, false},
		{"idempotency outside authn", []string{"idempotency", "authn", "ratelimit"}, false},
		{"ratelimit outside authn", []string{"ratelimit", "authn", "idempotency"}, false},
		{"authn only", []string{"authn"}, false},
		{"no middleware", nil, false},
	}
	op := writeOp{Method: http.MethodPost, Path: "/api/v1/widgets/{widget_id}"}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tel := newTripwireTelemetry(t)
			lim := &spyLimiter{}
			var ran sync.Map
			h := chainFixture(t, tel, lim, tc.chain, &ran)
			fails := probeWriteChain(t, h, tel, lim, op)
			if _, ok := ran.Load("handler"); ok && tc.pass {
				fails = append(fails, "the route handler ran during a probe")
			}
			switch {
			case tc.pass && len(fails) > 0:
				t.Errorf("chain %v: want PASS, got %v", tc.chain, fails)
			case !tc.pass && len(fails) == 0:
				t.Errorf("chain %v: want FAIL, the tripwire stayed green", tc.chain)
			default:
				t.Logf("chain %v: %v", tc.chain, fails)
			}
		})
	}
}

// TestWriteChainTripwireIsRedOnTheRealHandler probes today's real handler as
// if POST /api/v1/me were a tenant write route: authn and ratelimit are
// there, idempotency is not, so P3 must fail. This is what the tripwire
// reports when a write route lands on main without idempotency.
func TestWriteChainTripwireIsRedOnTheRealHandler(t *testing.T) {
	t.Parallel()
	tel := newTripwireTelemetry(t)
	lim := &spyLimiter{}
	fails := probeWriteChain(t, newTripwireHandler(t, tel, lim), tel, lim, writeOp{Method: http.MethodPost, Path: "/api/v1/me"})
	if len(fails) == 0 {
		t.Fatal("POST /api/v1/me passed the probe; the real handler has no idempotency, so P3 must fail")
	}
	for _, f := range fails {
		if strings.HasPrefix(f, "P1") || strings.HasPrefix(f, "P2") {
			t.Errorf("authn and ratelimit wrap every tenant route today, but: %s", f)
		}
	}
}

type writeOp struct{ Method, Path string }

func (o writeOp) String() string { return o.Method + " " + o.Path }

var pathParam = regexp.MustCompile(`\{[^}/]+\}`)

// url fills every path parameter with a fixed UUID.
func (o writeOp) url() string {
	return pathParam.ReplaceAllString(o.Path, "0b000000-0000-4000-8000-0000000000b1")
}

// contractWriteOps returns every write operation of the embedded contract and
// the tenant ones (outside the operator prefix), sorted.
func contractWriteOps(t *testing.T) (all, tenant []writeOp) {
	t.Helper()
	data, err := api.Files.ReadFile("openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	spec, err := apihttp.LoadSpec(data)
	if err != nil {
		t.Fatal(err)
	}
	writes := []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete}
	for path, item := range spec.Paths.Map() {
		for method := range item.Operations() {
			method = strings.ToUpper(method)
			if !slices.Contains(writes, method) {
				continue
			}
			op := writeOp{Method: method, Path: path}
			all = append(all, op)
			if !strings.HasPrefix(path, server.OperatorPrefix) {
				tenant = append(tenant, op)
			}
		}
	}
	for _, s := range [][]writeOp{all, tenant} {
		sort.Slice(s, func(i, j int) bool { return s[i].String() < s[j].String() })
	}
	return all, tenant
}

// probeWriteChain runs P1 to P3 against op and returns one line per broken
// expectation; none means the chain is authn -> ratelimit -> idempotency.
func probeWriteChain(t *testing.T, h http.Handler, tel *telemetry.Telemetry, lim *spyLimiter, op writeOp) []string {
	t.Helper()
	var fails []string
	fail := func(format string, args ...any) { fails = append(fails, fmt.Sprintf(format, args...)) }

	// P1: no token.
	lim.reset(true)
	before := idempotencyRejected(t, tel)
	rec := sendWrite(t, h, op, "", "{}")
	if rec.Code != http.StatusUnauthorized || rec.Header().Get("WWW-Authenticate") != "Bearer" {
		fail("P1 no token = %d %s, want 401 invalid_token from authn (authn must be outermost)", rec.Code, problemCode(rec))
	}
	if got := lim.calls(); len(got) != 0 {
		fail("P1 no token: the limiter was asked %d times, want 0 (ratelimit must sit inside authn)", len(got))
	}
	if d := idempotencyRejected(t, tel) - before; d != 0 {
		fail("P1 no token: idempotency rejected %v requests, want 0 (idempotency must sit inside authn)", d)
	}

	// P2: tenant A over its limit, no Idempotency-Key.
	lim.reset(false)
	before = idempotencyRejected(t, tel)
	rec = sendWrite(t, h, op, "token-a", "{}")
	if rec.Code != http.StatusTooManyRequests || problemCode(rec) != problem.RateLimited {
		fail("P2 over the limit = %d %s, want 429 rate_limited (ratelimit must sit outside idempotency)", rec.Code, problemCode(rec))
	}
	if got := lim.calls(); !slices.Equal(got, []uuid.UUID{tenantA}) {
		fail("P2 over the limit: the limiter was asked for %v, want [%s] (the tenant authn resolved)", got, tenantA)
	}
	if d := idempotencyRejected(t, tel) - before; d != 0 {
		fail("P2 over the limit: idempotency rejected %v requests, want 0 (idempotency must sit inside ratelimit)", d)
	}

	// P3: tenant A within its limit, no Idempotency-Key, malformed body.
	lim.reset(true)
	before = idempotencyRejected(t, tel)
	rec = sendWrite(t, h, op, "token-a", "{")
	want := []problem.FieldError{{Pointer: "/header/" + idempotency.Header, Rule: "required"}}
	if got := problemErrors(rec); rec.Code != http.StatusBadRequest || problemCode(rec) != problem.ValidationFailed || !slices.Equal(got, want) {
		fail("P3 no Idempotency-Key = %d %s %v, want 400 validation_failed %v from idempotency (mount it inside ratelimit)", rec.Code, problemCode(rec), got, want)
	}
	if d := idempotencyRejected(t, tel) - before; d != 1 {
		fail("P3 no Idempotency-Key: nava.idempotency.requests{result=rejected} moved by %v, want 1 (the idempotency middleware did not answer)", d)
	}
	if got := lim.calls(); !slices.Equal(got, []uuid.UUID{tenantA}) {
		fail("P3: the limiter was asked for %v, want [%s]", got, tenantA)
	}
	return fails
}

func sendWrite(t *testing.T, h http.Handler, op writeOp, token, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), op.Method, op.url(), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

type problemBody struct {
	Code   problem.Code         `json:"code"`
	Errors []problem.FieldError `json:"errors"`
}

func decodeProblem(rec *httptest.ResponseRecorder) problemBody {
	var p problemBody
	_ = json.Unmarshal(rec.Body.Bytes(), &p)
	return p
}

func problemCode(rec *httptest.ResponseRecorder) problem.Code { return decodeProblem(rec).Code }

func problemErrors(rec *httptest.ResponseRecorder) []problem.FieldError {
	return decodeProblem(rec).Errors
}

// idempotencyRejected reads nava.idempotency.requests{result="rejected"} from
// the Prometheus registry the admin listener serves.
func idempotencyRejected(t *testing.T, tel *telemetry.Telemetry) float64 {
	t.Helper()
	fams, err := tel.Registry.Gather()
	if err != nil {
		t.Fatal(err)
	}
	var n float64
	for _, f := range fams {
		if !strings.HasPrefix(f.GetName(), "nava_idempotency_requests") {
			continue
		}
		for _, m := range f.GetMetric() {
			for _, l := range m.GetLabel() {
				if l.GetName() == "result" && l.GetValue() == "rejected" {
					n += m.GetCounter().GetValue()
				}
			}
		}
	}
	return n
}

func newTripwireTelemetry(t *testing.T) *telemetry.Telemetry {
	t.Helper()
	tel, err := telemetry.New(t.Context(), config.Telemetry{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tel.Shutdown(context.Background()) })
	return tel
}

// spyLimiter allows or refuses every request and records the tenants asked.
type spyLimiter struct {
	mu      sync.Mutex
	allow   bool
	tenants []uuid.UUID
}

func (s *spyLimiter) Allow(_ context.Context, tenant uuid.UUID) (ratelimit.Decision, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tenants = append(s.tenants, tenant)
	if s.allow {
		return ratelimit.Decision{Allowed: true}, nil
	}
	return ratelimit.Decision{RetryAfter: 30 * time.Second}, nil
}

func (s *spyLimiter) reset(allow bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.allow, s.tenants = allow, nil
}

func (s *spyLimiter) calls() []uuid.UUID {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.tenants)
}

// chainFixture mounts POST /api/v1/widgets/{widget_id} behind the real
// middleware named in chain, outermost first. The idempotency pool points at
// a closed port: the probes never reach storage, and pgxpool connects lazily.
func chainFixture(t *testing.T, tel *telemetry.Telemetry, lim ratelimit.Limiter, chain []string, ran *sync.Map) http.Handler {
	t.Helper()
	errData, err := api.Files.ReadFile("errors.yaml")
	if err != nil {
		t.Fatal(err)
	}
	reg, err := problem.LoadRegistry(errData)
	if err != nil {
		t.Fatal(err)
	}
	log := slog.New(slog.DiscardHandler)
	authn, err := auth.New(twoTenants{}, reg, log, tel.MeterProvider)
	if err != nil {
		t.Fatal(err)
	}
	rl, err := ratelimit.New(lim, reg, log)
	if err != nil {
		t.Fatal(err)
	}
	pool, err := pgxpool.New(t.Context(), "postgres://nobody@127.0.0.1:1/none?connect_timeout=1")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	idem, err := idempotency.New(pool, reg, log, tel.MeterProvider)
	if err != nil {
		t.Fatal(err)
	}
	layers := map[string]func(http.Handler) http.Handler{
		"authn":       authn.Middleware,
		"ratelimit":   rl.Handler,
		"idempotency": idem.Handler,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/widgets/{widget_id}", func(w http.ResponseWriter, _ *http.Request) {
		ran.Store("handler", true)
		w.WriteHeader(http.StatusCreated)
	})
	var h http.Handler = mux
	for _, name := range slices.Backward(chain) {
		h = layers[name](h)
	}
	return h
}

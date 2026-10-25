package auth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/baggage"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/Navaladi2019/TicketingSystem/api"
	"github.com/Navaladi2019/TicketingSystem/internal/identity"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/auth"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/config"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/httpobs"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/logging"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/problem"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
)

//nolint:gochecknoglobals // fixed test ids, never written
var (
	tenantA    = uuid.MustParse("0a000000-0000-4000-8000-00000000000a")
	tenantB    = uuid.MustParse("0b000000-0000-4000-8000-00000000000b")
	principalA = uuid.MustParse("0a000000-0000-4000-8000-0000000003a1")
)

const secretToken = "eyJsecret.token-body.signature"

type resolver struct{}

func (resolver) Resolve(_ context.Context, raw string) (identity.Principal, error) {
	if raw == secretToken {
		return identity.Principal{TenantID: tenantA, PrincipalID: principalA, Groups: []uuid.UUID{}}, nil
	}
	return identity.Principal{}, fmt.Errorf("%w: signature", identity.ErrInvalidToken)
}

type syncBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuffer) lines(t *testing.T) []map[string]any {
	t.Helper()
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []map[string]any
	for _, l := range strings.Split(strings.TrimSpace(s.b.String()), "\n") {
		if l == "" {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(l), &m); err != nil {
			t.Fatalf("log line %q: %v", l, err)
		}
		out = append(out, m)
	}
	return out
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

type stack struct {
	h      http.Handler
	logs   *syncBuffer
	reader *sdkmetric.ManualReader
	seen   chan context.Context
}

// newStack is the public edge as nava-core builds it: httpobs, then the auth
// middleware, then a handler that logs one line and hands its context over.
func newStack(t *testing.T) *stack {
	t.Helper()
	logs := &syncBuffer{}
	log := logging.New(logs, config.Log{Level: slog.LevelDebug, Format: config.FormatJSON})
	reader := sdkmetric.NewManualReader()
	tel, err := telemetry.New(t.Context(), config.Telemetry{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tel.Shutdown(context.Background()) })
	data, err := api.Files.ReadFile("errors.yaml")
	if err != nil {
		t.Fatal(err)
	}
	reg, err := problem.LoadRegistry(data)
	if err != nil {
		t.Fatal(err)
	}
	a, err := auth.New(resolver{}, reg, log, sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader)))
	if err != nil {
		t.Fatal(err)
	}
	seen := make(chan context.Context, 1)
	mux := http.NewServeMux()
	mux.Handle("GET /api/v1/me", a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.InfoContext(r.Context(), "handler ran")
		seen <- r.Context()
		w.WriteHeader(http.StatusNoContent)
	})))
	return &stack{h: httpobs.Middleware(log, tel, mux), logs: logs, reader: reader, seen: seen}
}

func (s *stack) get(t *testing.T, hdr http.Header) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/me", nil)
	req.Header = hdr
	rec := httptest.NewRecorder()
	s.h.ServeHTTP(rec, req)
	return rec
}

// TICK-93: a log line written in a handler after resolution carries the
// tenant and principal ids.
func TestHandlerLogCarriesTenantAndPrincipal(t *testing.T) {
	t.Parallel()
	s := newStack(t)
	if rec := s.get(t, http.Header{"Authorization": {"Bearer " + secretToken}}); rec.Code != http.StatusNoContent {
		t.Fatalf("= %d %s", rec.Code, rec.Body)
	}
	var found bool
	for _, l := range s.logs.lines(t) {
		if l["msg"] != "handler ran" {
			continue
		}
		found = true
		if l["tenant_id"] != tenantA.String() || l["principal_id"] != principalA.String() {
			t.Errorf("handler log line = %v, want tenant_id and principal_id", l)
		}
		if l["request_id"] == nil {
			t.Errorf("handler log line lost request_id: %v", l)
		}
	}
	if !found {
		t.Fatalf("no handler log line in %s", s.logs)
	}
	if strings.Contains(s.logs.String(), secretToken) {
		t.Errorf("a log line carries the raw token: %s", s.logs)
	}
}

// A refusal logs the fixed reason only: never the token, and no tenant or
// principal id, since none was resolved.
func TestRefusalLogsNoToken(t *testing.T) {
	t.Parallel()
	s := newStack(t)
	forged := "eyJforged.secret-claims.sig"
	if rec := s.get(t, http.Header{"Authorization": {"Bearer " + forged}}); rec.Code != http.StatusUnauthorized {
		t.Fatalf("= %d %s", rec.Code, rec.Body)
	}
	out := s.logs.String()
	if strings.Contains(out, "secret-claims") {
		t.Errorf("a log line carries the token: %s", out)
	}
	if !strings.Contains(out, `"msg":"authentication refused"`) || !strings.Contains(out, `"reason":"invalid_token: signature"`) {
		t.Errorf("no refusal line with the fixed reason: %s", out)
	}
	if strings.Contains(out, `"tenant_id"`) {
		t.Errorf("a refused request logged a tenant: %s", out)
	}
}

// TICK-93 CI-2: inbound baggage naming tenant B, with a token for tenant A,
// runs as tenant A, and the baggage never reaches the request context.
func TestForgedBaggageIsNotInTheContext(t *testing.T) {
	t.Parallel()
	s := newStack(t)
	rec := s.get(t, http.Header{
		"Authorization": {"Bearer " + secretToken},
		"Baggage":       {"tenant_id=" + tenantB.String() + ",nava.tenant_id=" + tenantB.String()},
	})
	if rec.Code != http.StatusNoContent {
		t.Fatalf("= %d %s", rec.Code, rec.Body)
	}
	ctx := <-s.seen
	if b := baggage.FromContext(ctx); b.Len() != 0 {
		t.Errorf("baggage reached the request context: %s", b)
	}
	if p, ok := identity.FromContext(ctx); !ok || p.TenantID != tenantA {
		t.Errorf("principal = %+v (%v), want tenant A", p, ok)
	}
}

// nava.auth.requests counts by result only: no tenant or principal attribute.
func TestMetricHasNoTenantAttribute(t *testing.T) {
	t.Parallel()
	s := newStack(t)
	s.get(t, http.Header{"Authorization": {"Bearer " + secretToken}})
	s.get(t, http.Header{"Authorization": {"Bearer nope"}})
	s.get(t, http.Header{})
	var rm metricdata.ResourceMetrics
	if err := s.reader.Collect(t.Context(), &rm); err != nil {
		t.Fatal(err)
	}
	got := map[string]int64{}
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != "nava.auth.requests" {
				continue
			}
			for _, dp := range m.Data.(metricdata.Sum[int64]).DataPoints {
				for _, kv := range dp.Attributes.ToSlice() {
					if kv.Key != "result" {
						t.Errorf("attribute %s on nava.auth.requests", kv.Key)
					}
				}
				r, _ := dp.Attributes.Value("result")
				got[r.AsString()] += dp.Value
			}
		}
	}
	if got["ok"] != 1 || got["invalid_token"] != 2 {
		t.Errorf("nava.auth.requests = %v, want ok 1, invalid_token 2", got)
	}
}

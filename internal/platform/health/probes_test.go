package health_test

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"go.temporal.io/api/workflowservice/v1"
	"google.golang.org/grpc"

	"github.com/Navaladi2019/TicketingSystem/internal/platform/health"
	"github.com/Navaladi2019/TicketingSystem/internal/platform/telemetry"
)

func outboundClient(t *testing.T) *telemetry.OutboundClient {
	t.Helper()
	c, err := telemetry.NewOutboundClient(newTel(t), telemetry.OutboundOptions{Timeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestKeycloakProbe(t *testing.T) {
	t.Parallel()
	var status atomicInt
	status.Store(http.StatusOK)
	kc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health/ready" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(int(status.Load()))
	}))
	t.Cleanup(kc.Close)
	check := health.Keycloak(kc.URL, "", outboundClient(t))
	if check.Critical || check.Name != "keycloak" {
		t.Fatalf("check = %+v, want non-critical keycloak", check)
	}
	if err := check.Probe(t.Context()); err != nil {
		t.Errorf("healthy Keycloak: %v", err)
	}
	status.Store(http.StatusServiceUnavailable)
	if err := check.Probe(t.Context()); err == nil {
		t.Error("Keycloak answering 503 passed")
	}

	kc.Close()
	err := check.Probe(t.Context())
	if err == nil {
		t.Fatal("stopped Keycloak passed")
	}
	if strings.Contains(err.Error(), "?") {
		t.Errorf("probe error carries a query: %v", err)
	}
}

func TestKeycloakProbeUnconfigured(t *testing.T) {
	t.Parallel()
	if err := health.Keycloak("", "", outboundClient(t)).Probe(t.Context()); !errors.Is(err, health.ErrUnconfigured) {
		t.Errorf("err = %v, want ErrUnconfigured", err)
	}
}

type fakeFrontend struct {
	workflowservice.UnimplementedWorkflowServiceServer
}

func (fakeFrontend) GetSystemInfo(context.Context, *workflowservice.GetSystemInfoRequest) (*workflowservice.GetSystemInfoResponse, error) {
	return &workflowservice.GetSystemInfoResponse{ServerVersion: "1.28.4"}, nil
}

func TestTemporalProbeCallsGetSystemInfo(t *testing.T) {
	t.Parallel()
	lis, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := grpc.NewServer()
	workflowservice.RegisterWorkflowServiceServer(srv, fakeFrontend{})
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	check, closeFn, err := health.Temporal(lis.Addr().String(), nil, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(closeFn)
	if check.Critical || check.Name != "temporal" {
		t.Fatalf("check = %+v, want non-critical temporal", check)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	if err := check.Probe(ctx); err != nil {
		t.Errorf("healthy Temporal: %v", err)
	}

	srv.Stop()
	ctx2, cancel2 := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel2()
	if err := check.Probe(ctx2); err == nil {
		t.Error("stopped Temporal passed")
	}
}

func TestTemporalProbeUnconfiguredAndLazy(t *testing.T) {
	t.Parallel()
	check, closeFn, err := health.Temporal("", nil, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	closeFn()
	if err := check.Probe(t.Context()); !errors.Is(err, health.ErrUnconfigured) {
		t.Errorf("err = %v, want ErrUnconfigured", err)
	}
	// An address nothing listens on must not fail construction: it fails the probe.
	dead := closedAddr(t)
	check, closeFn, err = health.Temporal(dead, nil, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatalf("construction with Temporal down: %v", err)
	}
	t.Cleanup(closeFn)
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	if err := check.Probe(ctx); err == nil {
		t.Error("probe of a dead address passed")
	}
}

func TestPostgresProbeBadAndUnconfigured(t *testing.T) {
	t.Parallel()
	if _, _, err := health.Postgres(t.Context(), "postgres://u:hunter2@h:notaport/db"); err == nil || strings.Contains(err.Error(), "hunter2") {
		t.Errorf("bad DSN error = %v, want one without the password", err)
	}
	check, closeFn, err := health.Postgres(t.Context(), "")
	if err != nil {
		t.Fatal(err)
	}
	closeFn()
	if !check.Critical || !errors.Is(check.Probe(t.Context()), health.ErrUnconfigured) {
		t.Errorf("unset DSN: critical=%v, want critical and ErrUnconfigured", check.Critical)
	}
	// Unreachable database: construction succeeds, the probe fails, no password in the error.
	dsn := "postgres://app:hunter2@" + closedAddr(t) + "/nava?sslmode=disable"
	check, closeFn, err = health.Postgres(t.Context(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(closeFn)
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()
	perr := check.Probe(ctx)
	if perr == nil {
		t.Fatal("probe of a dead database passed")
	}
	if strings.Contains(perr.Error(), "hunter2") {
		t.Errorf("probe error leaks the password: %v", perr)
	}
}

func closedAddr(t *testing.T) string {
	t.Helper()
	l, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	a := l.Addr().String()
	_ = l.Close()
	return a
}

type atomicInt struct{ v atomic.Int64 }

func (a *atomicInt) Store(n int64) { a.v.Store(n) }
func (a *atomicInt) Load() int64   { return a.v.Load() }

// TICK-328: /health/ready can stay 200 while Keycloak cannot read its
// database. With a token URL the check also needs 401 from the token endpoint.
func TestKeycloakProbeTokenEndpoint(t *testing.T) {
	t.Parallel()
	var token atomicInt
	token.Store(http.StatusUnauthorized)
	var sawForm atomicInt
	kc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health/ready":
			w.WriteHeader(http.StatusOK)
		case "/realms/nava/protocol/openid-connect/token":
			if r.Method == http.MethodPost && r.Header.Get("Authorization") == "" {
				_ = r.ParseForm()
				if r.PostForm.Get("client_id") == "nava-health-probe" && r.PostForm.Get("client_secret") == "" {
					sawForm.Store(1)
				}
			}
			w.WriteHeader(int(token.Load()))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(kc.Close)
	check := health.Keycloak(kc.URL, kc.URL+"/realms/nava/protocol/openid-connect/token", outboundClient(t))
	if err := check.Probe(t.Context()); err != nil {
		t.Errorf("healthy Keycloak: %v", err)
	}
	if sawForm.Load() != 1 {
		t.Error("token request lacked the probe client_id or carried a credential")
	}
	for _, code := range []int{http.StatusBadRequest, http.StatusInternalServerError, http.StatusOK} {
		token.Store(int64(code))
		if err := check.Probe(t.Context()); err == nil {
			t.Errorf("token endpoint answering %d passed", code)
		}
	}
}
